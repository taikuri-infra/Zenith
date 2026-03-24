package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/crypto"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// Deployer handles deploying built images to Kubernetes.
type Deployer struct {
	k8sClient  k8sclient.Client
	appRepo    ports.AppRepository
	envVarRepo ports.EnvVarRepository
	planRepo   ports.UserPlanRepository
	domainRepo ports.DomainRepository
	envCrypto  *crypto.EnvCrypto
	baseDomain string
}

// NewDeployer creates a new Deployer.
func NewDeployer(k8sClient k8sclient.Client, appRepo ports.AppRepository, planRepo ports.UserPlanRepository, baseDomain string) *Deployer {
	return &Deployer{
		k8sClient:  k8sClient,
		appRepo:    appRepo,
		planRepo:   planRepo,
		baseDomain: baseDomain,
	}
}

// SetDomainRepo sets the domain repository for custom domain support.
func (d *Deployer) SetDomainRepo(repo ports.DomainRepository) {
	d.domainRepo = repo
}

// SetEnvVarRepo injects the V2 env var repository for per-environment env vars.
func (d *Deployer) SetEnvVarRepo(repo ports.EnvVarRepository) {
	d.envVarRepo = repo
}

// SetEnvCrypto injects the encryption helper to decrypt secret env vars before K8s injection.
func (d *Deployer) SetEnvCrypto(c *crypto.EnvCrypto) {
	d.envCrypto = c
}

// DeployApp deploys an app's built image to Kubernetes.
// It creates or updates the Deployment, Service, IngressRoute, and
// optionally the KEDA HTTPScaledObject for free-tier scale-to-zero.
func (d *Deployer) DeployApp(ctx context.Context, app *entities.App, imageTag string) error {
	slog.Info("deploying app", "app", app.Name, "image", imageTag)

	// Get env vars for the app.
	// V2: use per-environment vars when envVarRepo is set (app.EnvironmentID="" = production/default).
	// Fallback to legacy appRepo.GetEnvVars for backward compatibility.
	var envVars []entities.EnvVar
	if d.envVarRepo != nil {
		v2vars, err := d.envVarRepo.GetEnvVarsByEnvironment(ctx, app.ID, app.EnvironmentID)
		if err != nil {
			return fmt.Errorf("failed to get env vars: %w", err)
		}
		for _, v := range v2vars {
			value := v.Value
			// Decrypt secret values before injecting into the pod.
			// Plaintext values pass through unchanged.
			if v.IsSecret && d.envCrypto != nil && crypto.IsEncrypted(value) {
				decrypted, err := d.envCrypto.Decrypt(app.UserID, value)
				if err != nil {
					slog.Error("failed to decrypt env var, skipping", "key", v.Key, "error", err)
					continue
				}
				value = decrypted
			}
			envVars = append(envVars, entities.EnvVar{
				ID:    v.ID,
				AppID: v.AppID,
				Key:   v.Key,
				Value: value,
			})
		}
	} else {
		var err error
		envVars, err = d.appRepo.GetEnvVars(ctx, app.ID)
		if err != nil {
			return fmt.Errorf("failed to get env vars: %w", err)
		}
	}

	// Look up user plan to decide scale-to-zero and resource limits
	var planLimits *entities.PlanLimits
	tier := entities.PlanFree
	if d.planRepo != nil {
		plan, err := d.planRepo.GetUserPlan(ctx, app.UserID)
		if err == nil {
			planLimits = &plan.Limits
			tier = plan.Tier
		} else {
			slog.Error("failed to get user plan, defaulting to free tier", "user_id", app.UserID, "error", err)
		}
	}

	// Fetch active custom domains for this app
	var customDomains []string
	if d.domainRepo != nil {
		domains, err := d.domainRepo.ListDomainsByApp(ctx, app.ID)
		if err != nil {
			slog.Error("failed to fetch custom domains", "app_id", app.ID, "error", err)
		} else {
			for _, dom := range domains {
				if dom.Status == entities.DomainStatusActive {
					customDomains = append(customDomains, dom.Domain)
				}
			}
		}
	}

	// Generate K8s resources
	resources := GenerateK8sResources(app, imageTag, d.baseDomain, envVars, planLimits, tier, customDomains)

	// Apply Deployment
	if err := d.applyCRD(ctx, "Deployment", "zenith-apps", app.Subdomain, resources.Deployment); err != nil {
		return fmt.Errorf("failed to apply Deployment: %w", err)
	}

	// Apply Service
	if err := d.applyCRD(ctx, "Service", "zenith-apps", app.Subdomain, resources.Service); err != nil {
		return fmt.Errorf("failed to apply Service: %w", err)
	}

	// Apply IngressRoute
	if err := d.applyCRD(ctx, "IngressRoute", "zenith-apps", app.Subdomain, resources.IngressRoute); err != nil {
		return fmt.Errorf("failed to apply IngressRoute: %w", err)
	}

	// Apply Certificate CRD for custom domains (before NetworkPolicy)
	if resources.Certificate != nil {
		if err := d.applyCRD(ctx, "Certificate", "zenith-apps", app.Subdomain+"-custom-tls", resources.Certificate); err != nil {
			return fmt.Errorf("failed to apply Certificate: %w", err)
		}
	}

	// Apply NetworkPolicy for tenant isolation
	if resources.NetworkPolicy != nil {
		if err := d.applyCRD(ctx, "NetworkPolicy", "zenith-apps", app.Subdomain+"-netpol", resources.NetworkPolicy); err != nil {
			return fmt.Errorf("failed to apply NetworkPolicy: %w", err)
		}
	}

	// Apply KEDA HTTPScaledObject if scale-to-zero is enabled
	if resources.HTTPScaledObject != nil {
		if err := d.applyCRD(ctx, "HTTPScaledObject", "zenith-apps", app.Subdomain, resources.HTTPScaledObject); err != nil {
			return fmt.Errorf("failed to apply HTTPScaledObject: %w", err)
		}
		// KEDA-managed: starts sleeping, will wake on first request
		status := entities.AppStatusSleeping
		d.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{
			Status: &status,
		})
		slog.Info("app deployed (sleeping)", "app", app.Name, "subdomain", app.Subdomain, "base_domain", d.baseDomain)
	} else {
		// Always-on: set status to running
		status := entities.AppStatusRunning
		d.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{
			Status: &status,
		})
		slog.Info("app deployed", "app", app.Name, "subdomain", app.Subdomain, "base_domain", d.baseDomain)
	}

	return nil
}

// DeleteApp removes all K8s resources for an app, including KEDA CRDs.
func (d *Deployer) DeleteApp(ctx context.Context, app *entities.App) error {
	slog.Info("deleting K8s resources for app", "app", app.Name, "subdomain", app.Subdomain)

	namespace := "zenith-apps"

	// Each resource type has its own apiVersion — must use DeleteCRDWithVersion
	// to correctly resolve the GroupVersionResource.
	resources := []struct {
		apiVersion string
		kind       string
		name       string
	}{
		{"apps/v1", "Deployment", app.Subdomain},
		{"v1", "Service", app.Subdomain},
		{"traefik.io/v1alpha1", "IngressRoute", app.Subdomain},
		{"keda.sh/v1alpha1", "HTTPScaledObject", app.Subdomain},
		{"networking.k8s.io/v1", "NetworkPolicy", app.Subdomain + "-netpol"},
		{"cert-manager.io/v1", "Certificate", app.Subdomain + "-custom-tls"},
	}

	for _, r := range resources {
		if err := d.k8sClient.DeleteCRDWithVersion(ctx, r.apiVersion, r.kind, namespace, r.name); err != nil {
			if !k8serrors.IsNotFound(err) {
				slog.Error("failed to delete K8s resource", "kind", r.kind, "name", r.name, "error", err)
			}
		} else {
			slog.Info("deleted K8s resource", "kind", r.kind, "name", r.name)
		}
	}

	return nil
}

// applyCRD creates or updates a K8s resource via the CRD client.
func (d *Deployer) applyCRD(ctx context.Context, kind, namespace, name string, resource map[string]interface{}) error {
	// Extract just the "spec" field from the full manifest — the CRDObject
	// already carries apiVersion/kind/metadata separately.
	specData := resource["spec"]
	if specData == nil {
		specData = resource
	}

	spec, err := json.Marshal(specData)
	if err != nil {
		return fmt.Errorf("failed to marshal resource spec: %w", err)
	}

	// Carry over labels from the manifest metadata if present
	var labels map[string]string
	if meta, ok := resource["metadata"].(map[string]interface{}); ok {
		if rawLabels, ok := meta["labels"].(map[string]interface{}); ok {
			labels = make(map[string]string, len(rawLabels))
			for k, v := range rawLabels {
				if s, ok := v.(string); ok {
					labels[k] = s
				}
			}
		}
	}

	crd := &k8sclient.CRDObject{
		APIVersion: getAPIVersion(resource),
		Kind:       kind,
		Metadata: k8sclient.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: spec,
	}

	// Try create first; if already exists, merge-patch to update
	if err := d.k8sClient.CreateCRD(ctx, crd); err != nil {
		if k8serrors.IsAlreadyExists(err) || strings.Contains(err.Error(), "already exists") {
			return d.k8sClient.PatchCRD(ctx, crd)
		}
		return err
	}

	return nil
}

func getAPIVersion(resource map[string]interface{}) string {
	if v, ok := resource["apiVersion"].(string); ok {
		return v
	}
	return "v1"
}
