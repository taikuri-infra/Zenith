package deploy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"

	pkgCrypto "github.com/dotechhq/zenith/services/api/internal/crypto"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/validate"
)

// DockerDeployer runs user apps as plain Docker containers on the host — the
// compute backend for the standalone self-host edition (no Kubernetes). Routing
// and HTTPS are handled by Traefik, which reads the traefik.* container labels
// this deployer sets — the same proxy the cloud/k8s edition uses (IngressRoute).
type DockerDeployer struct {
	cli          *client.Client
	appRepo      ports.AppRepository
	envVarRepo   ports.EnvVarRepository
	planRepo     ports.UserPlanRepository
	envCrypto    *pkgCrypto.EnvCrypto
	baseDomain   string
	domainRepo   ports.DomainRepository
	certResolver string
	network      string // docker network shared with Traefik
	// Per-app locks serialize deploy/delete for a given app. Deploys are
	// triggered from several places (app creation, project deploy, env-var
	// apply) that can fire concurrently; without this they race on the
	// remove→create→start of the same container.
	locks sync.Map // app.ID -> *sync.Mutex
}

var _ Backend = (*DockerDeployer)(nil)

// appLock returns the per-app mutex, creating it on first use.
func (d *DockerDeployer) appLock(appID string) *sync.Mutex {
	v, _ := d.locks.LoadOrStore(appID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// NewDockerDeployer creates a Docker-backed deployer. network is the docker
// network app containers join (must be the one Traefik watches).
func NewDockerDeployer(cli *client.Client, appRepo ports.AppRepository, envVarRepo ports.EnvVarRepository, planRepo ports.UserPlanRepository, baseDomain, network string) *DockerDeployer {
	return &DockerDeployer{
		cli:          cli,
		appRepo:      appRepo,
		envVarRepo:   envVarRepo,
		planRepo:     planRepo,
		baseDomain:   baseDomain,
		certResolver: "le",
		network:      network,
	}
}

// SetEnvCrypto wires the secret-decryption helper (parity with the k8s deployer).
func (d *DockerDeployer) SetEnvCrypto(c *pkgCrypto.EnvCrypto) { d.envCrypto = c }

// SetEnvVarRepo wires the per-environment env-var repo (parity with the k8s deployer).
func (d *DockerDeployer) SetEnvVarRepo(repo ports.EnvVarRepository) { d.envVarRepo = repo }

// containerName is the deterministic name for an app's container.
func (d *DockerDeployer) containerName(app *entities.App) string { return appContainerName(app) }

// appContainerName is the deterministic container name for an app (shared by
// the deployer and the log source).
func appContainerName(app *entities.App) string {
	return "zenith-app-" + sanitizeName(app.Subdomain)
}

// DeployApp pulls the image and (re)starts the app container with env + routing.
func (d *DockerDeployer) DeployApp(ctx context.Context, app *entities.App, imageTag string) error {
	lk := d.appLock(app.ID)
	lk.Lock()
	defer lk.Unlock()

	slog.Info("docker-deploy: deploying app", "app", app.Name, "image", imageTag)

	envVars, err := d.resolveEnvVars(ctx, app)
	if err != nil {
		return err
	}
	env := make([]string, 0, len(envVars)+1)
	env = append(env, fmt.Sprintf("PORT=%d", app.Port))
	for _, v := range envVars {
		env = append(env, v.Key+"="+v.Value)
	}

	// Pull the image (with optional per-app registry auth).
	pullOpts := image.PullOptions{}
	if app.RegistryUser != "" && app.RegistryPassword != "" {
		password := app.RegistryPassword
		if d.envCrypto != nil && pkgCrypto.IsEncrypted(password) {
			if dec, derr := d.envCrypto.Decrypt(app.UserID, password); derr == nil {
				password = dec
			}
		}
		authJSON, _ := json.Marshal(registrytypes.AuthConfig{
			Username: app.RegistryUser,
			Password: password,
		})
		pullOpts.RegistryAuth = base64.URLEncoding.EncodeToString(authJSON)
	}
	reader, err := d.cli.ImagePull(ctx, imageTag, pullOpts)
	if err != nil {
		return fmt.Errorf("docker-deploy: pull %s: %w", imageTag, err)
	}
	_, _ = io.Copy(io.Discard, reader)
	_ = reader.Close()

	name := d.containerName(app)
	// Remove any previous container so we deploy fresh.
	_ = d.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})

	// Traefik labels: route <subdomain>.<baseDomain> to this container's port.
	// Traefik obtains the Let's Encrypt cert (certresolver "le"). Same proxy
	// model as the cloud/k8s edition, which routes via Traefik IngressRoute.
	labels := proxyLabels(app, d.appHosts(ctx, app), d.network, d.certResolver)

	// Network aliases let other services reach this one by app name and subdomain.
	aliases := networkAliases(app)
	resp, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Image:  imageTag,
			Env:    env,
			Labels: labels,
		},
		&container.HostConfig{
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				d.network: {Aliases: aliases},
			},
		},
		nil, name)
	if err != nil {
		return fmt.Errorf("docker-deploy: create container %s: %w", name, err)
	}
	if err := d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("docker-deploy: start container %s: %w", name, err)
	}
	slog.Info("docker-deploy: app running", "app", app.Name, "container", name,
		"host", fmt.Sprintf("%s.%s", app.Subdomain, d.baseDomain))
	d.markRunning(ctx, app)
	return nil
}

// markRunning records that the app's container started.
func (d *DockerDeployer) markRunning(ctx context.Context, app *entities.App) {
	if d.appRepo == nil {
		return
	}
	running := entities.AppStatusRunning
	if _, err := d.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{Status: &running}); err != nil {
		slog.Warn("docker-deploy: mark app running failed", "app", app.Name, "error", err)
	}
}

// DeleteApp stops and removes the app's container.
func (d *DockerDeployer) DeleteApp(ctx context.Context, app *entities.App) error {
	lk := d.appLock(app.ID)
	lk.Lock()
	defer lk.Unlock()

	name := d.containerName(app)
	if err := d.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true}); err != nil {
		if client.IsErrNotFound(err) {
			return nil
		}
		return fmt.Errorf("docker-deploy: remove container %s: %w", name, err)
	}
	return nil
}

// resolveEnvVars builds the container env from three sources, later ones
// overriding earlier ones by key:
//  1. legacy app_env_vars (app create, compose import, database attach)
//  2. v2 app-wide vars (environment_id NULL; the UI Environment tab)
//  3. v2 vars scoped to the app's environment
//
// Secrets are decrypted; a secret that fails to decrypt is skipped.
func (d *DockerDeployer) resolveEnvVars(ctx context.Context, app *entities.App) ([]entities.EnvVar, error) {
	merged := map[string]string{}
	var order []string
	put := func(k, v string) {
		if _, ok := merged[k]; !ok {
			order = append(order, k)
		}
		merged[k] = v
	}
	decrypt := func(key, value string) (string, bool) {
		if d.envCrypto != nil && pkgCrypto.IsEncrypted(value) {
			dec, err := d.envCrypto.Decrypt(app.UserID, value)
			if err != nil {
				slog.Error("docker-deploy: decrypt env var failed, skipping", "key", key, "error", err)
				return "", false
			}
			return dec, true
		}
		return value, true
	}

	legacy, err := d.appRepo.GetEnvVars(ctx, app.ID)
	if err != nil {
		return nil, fmt.Errorf("docker-deploy: get env vars: %w", err)
	}
	for _, v := range legacy {
		if val, ok := decrypt(v.Key, v.Value); ok {
			put(v.Key, val)
		}
	}

	if d.envVarRepo != nil {
		scopes := []string{""}
		if app.EnvironmentID != "" {
			scopes = append(scopes, app.EnvironmentID)
		}
		for _, envID := range scopes {
			vars, err := d.envVarRepo.GetEnvVarsByEnvironment(ctx, app.ID, envID)
			if err != nil {
				return nil, fmt.Errorf("docker-deploy: get env vars: %w", err)
			}
			for _, v := range vars {
				if val, ok := decrypt(v.Key, v.Value); ok {
					put(v.Key, val)
				}
			}
		}
	}

	out := make([]entities.EnvVar, 0, len(order))
	for _, k := range order {
		out = append(out, entities.EnvVar{AppID: app.ID, Key: k, Value: merged[k]})
	}
	return out, nil
}

// networkAliases are the names other containers on the app network can use
// for this app. Never the bare compose service name: apps, databases and the
// control plane share one network, so e.g. "postgres" would shadow the API's
// own database host (F-21 needs per-project networks).
func networkAliases(app *entities.App) []string {
	return []string{sanitizeName(app.Name), sanitizeName(app.Subdomain)}
}

// SetDomainRepo lets the deployer route an app's active custom domains.
func (d *DockerDeployer) SetDomainRepo(repo ports.DomainRepository) { d.domainRepo = repo }

// SetCertResolver names the Traefik ACME resolver app routers use ("le").
// Empty means no resolver label (Traefik serves its default certificate).
func (d *DockerDeployer) SetCertResolver(name string) { d.certResolver = name }

// appHosts is the app's generated host plus its active custom domains.
func (d *DockerDeployer) appHosts(ctx context.Context, app *entities.App) []string {
	hosts := []string{fmt.Sprintf("%s.%s", app.Subdomain, d.baseDomain)}
	if d.domainRepo == nil {
		return hosts
	}
	domains, err := d.domainRepo.ListDomainsByApp(ctx, app.ID)
	if err != nil {
		slog.Warn("docker-deploy: list custom domains failed", "app", app.Name, "error", err)
		return hosts
	}
	for _, cd := range domains {
		if cd.Status != entities.DomainStatusActive && cd.Status != entities.DomainStatusVerified {
			continue
		}
		// Defense in depth: the domain handler validates on add, but a row
		// stored before that (or written elsewhere) must never reach the
		// Traefik Host rule unvalidated.
		if !validate.Hostname(cd.Domain) {
			slog.Warn("docker-deploy: skipping invalid custom domain", "app", app.Name, "domain", cd.Domain)
			continue
		}
		hosts = append(hosts, cd.Domain)
	}
	return hosts
}

// proxyLabels builds the Traefik router/service labels that expose an app
// container on the given hosts over HTTPS. Routers must name the resolver:
// in Traefik v3 a router with its own tls section ignores the entrypoint's
// default resolver, which left routes on the default (self-signed) cert.
//
// A non-public app gets no traefik.* labels: it is reachable only on the app
// network. Self-host has no JWT gateway, so "protected" (what the compose
// import sends for private services) means private here.
func proxyLabels(app *entities.App, hosts []string, network, certResolver string) map[string]string {
	if app.Exposure == entities.ExposureProtected {
		return map[string]string{"zenith.app.id": app.ID}
	}
	rn := sanitizeName(app.Subdomain)
	labels := map[string]string{
		"zenith.app.id":          app.ID,
		"traefik.enable":         "true",
		"traefik.docker.network": network,
		fmt.Sprintf("traefik.http.routers.%s.rule", rn):                      hostRule(hosts),
		fmt.Sprintf("traefik.http.routers.%s.entrypoints", rn):               "websecure",
		fmt.Sprintf("traefik.http.routers.%s.tls", rn):                       "true",
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", rn): fmt.Sprintf("%d", app.Port),
	}
	if certResolver != "" {
		labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", rn)] = certResolver
	}
	return labels
}

// hostRule turns hostnames into a Traefik rule: Host(`a`) || Host(`b`).
func hostRule(hosts []string) string {
	const bq = "`"
	parts := make([]string, len(hosts))
	for i, h := range hosts {
		parts[i] = "Host(" + bq + h + bq + ")"
	}
	return strings.Join(parts, " || ")
}

func sanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
