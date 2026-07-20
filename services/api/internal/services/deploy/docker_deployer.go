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

	"github.com/dotechhq/zenith/services/api/internal/entities"
	pkgCrypto "github.com/dotechhq/zenith/services/api/internal/crypto"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// DockerDeployer runs user apps as plain Docker containers on the host — the
// compute backend for the standalone self-host edition (no Kubernetes). Routing
// and HTTPS are handled by Traefik, which reads the traefik.* container labels
// this deployer sets — the same proxy the cloud/k8s edition uses (IngressRoute).
type DockerDeployer struct {
	cli        *client.Client
	appRepo    ports.AppRepository
	envVarRepo ports.EnvVarRepository
	planRepo   ports.UserPlanRepository
	envCrypto  *pkgCrypto.EnvCrypto
	baseDomain string
	network    string // docker network shared with Traefik
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
		cli:        cli,
		appRepo:    appRepo,
		envVarRepo: envVarRepo,
		planRepo:   planRepo,
		baseDomain: baseDomain,
		network:    network,
	}
}

// SetEnvCrypto wires the secret-decryption helper (parity with the k8s deployer).
func (d *DockerDeployer) SetEnvCrypto(c *pkgCrypto.EnvCrypto) { d.envCrypto = c }

// SetEnvVarRepo wires the per-environment env-var repo (parity with the k8s deployer).
func (d *DockerDeployer) SetEnvVarRepo(repo ports.EnvVarRepository) { d.envVarRepo = repo }

// containerName is the deterministic name for an app's container.
func (d *DockerDeployer) containerName(app *entities.App) string {
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
	labels := proxyLabels(app, d.baseDomain, d.network)

	// Network aliases let other services reach this one by its original compose
	// service name (e.g. a backend connecting to "db:5432") and by subdomain.
	aliases := []string{sanitizeName(app.Name), sanitizeName(app.Subdomain)}
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
	return nil
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

// resolveEnvVars fetches and decrypts the app's env vars, mirroring the k8s deployer.
func (d *DockerDeployer) resolveEnvVars(ctx context.Context, app *entities.App) ([]entities.EnvVar, error) {
	var out []entities.EnvVar
	if d.envVarRepo != nil {
		vars, err := d.envVarRepo.GetEnvVarsByEnvironment(ctx, app.ID, app.EnvironmentID)
		if err != nil {
			return nil, fmt.Errorf("docker-deploy: get env vars: %w", err)
		}
		for _, v := range vars {
			value := v.Value
			if v.IsSecret && d.envCrypto != nil && pkgCrypto.IsEncrypted(value) {
				if dec, derr := d.envCrypto.Decrypt(app.UserID, value); derr == nil {
					value = dec
				} else {
					slog.Error("docker-deploy: decrypt env var failed, skipping", "key", v.Key, "error", derr)
					continue
				}
			}
			out = append(out, entities.EnvVar{ID: v.ID, AppID: v.AppID, Key: v.Key, Value: value})
		}
		return out, nil
	}
	legacy, err := d.appRepo.GetEnvVars(ctx, app.ID)
	if err != nil {
		return nil, fmt.Errorf("docker-deploy: get env vars: %w", err)
	}
	for _, v := range legacy {
		value := v.Value
		if d.envCrypto != nil && pkgCrypto.IsEncrypted(value) {
			if dec, derr := d.envCrypto.Decrypt(app.UserID, value); derr == nil {
				value = dec
			} else {
				continue
			}
		}
		out = append(out, entities.EnvVar{ID: v.ID, AppID: v.AppID, Key: v.Key, Value: value})
	}
	return out, nil
}

// sanitizeName makes a subdomain safe for a docker container name.
// proxyLabels builds the Traefik router/service labels that expose an app
// container at <subdomain>.<baseDomain> over HTTPS. Router and service names are
// derived from the subdomain so they're unique and human-readable. This mirrors
// the Traefik IngressRoute the cloud/k8s edition creates for the same app.
func proxyLabels(app *entities.App, baseDomain, network string) map[string]string {
	hosts := []string{fmt.Sprintf("%s.%s", app.Subdomain, baseDomain)}
	rn := sanitizeName(app.Subdomain)
	return map[string]string{
		"zenith.app.id":          app.ID,
		"traefik.enable":         "true",
		"traefik.docker.network": network,
		fmt.Sprintf("traefik.http.routers.%s.rule", rn):                      hostRule(hosts),
		fmt.Sprintf("traefik.http.routers.%s.entrypoints", rn):               "websecure",
		fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", rn):          "le",
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", rn): fmt.Sprintf("%d", app.Port),
	}
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
