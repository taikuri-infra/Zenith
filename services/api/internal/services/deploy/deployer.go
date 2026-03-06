package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// Deployer handles deploying built images to Kubernetes.
type Deployer struct {
	k8sClient  k8sclient.Client
	appRepo    ports.AppRepository
	planRepo   ports.UserPlanRepository
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

// DeployApp deploys an app's built image to Kubernetes.
// It creates or updates the Deployment, Service, IngressRoute, and
// optionally the KEDA HTTPScaledObject for free-tier scale-to-zero.
func (d *Deployer) DeployApp(ctx context.Context, app *entities.App, imageTag string) error {
	log.Printf("[deployer] Deploying app=%s image=%s", app.Name, imageTag)

	// Get env vars for the app
	envVars, err := d.appRepo.GetEnvVars(ctx, app.ID)
	if err != nil {
		return fmt.Errorf("failed to get env vars: %w", err)
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
			log.Printf("[deployer] Warning: failed to get user plan for %s: %v (defaulting to free tier)", app.UserID, err)
		}
	}

	// Generate K8s resources
	resources := GenerateK8sResources(app, imageTag, d.baseDomain, envVars, planLimits, tier)

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
		log.Printf("[deployer] App deployed (sleeping): %s → https://%s.%s", app.Name, app.Subdomain, d.baseDomain)
	} else {
		// Always-on: set status to running
		status := entities.AppStatusRunning
		d.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{
			Status: &status,
		})
		log.Printf("[deployer] App deployed: %s → https://%s.%s", app.Name, app.Subdomain, d.baseDomain)
	}

	return nil
}

// DeleteApp removes all K8s resources for an app, including KEDA CRDs.
func (d *Deployer) DeleteApp(ctx context.Context, app *entities.App) error {
	log.Printf("[deployer] Deleting K8s resources for app=%s", app.Name)

	namespace := "zenith-apps"
	for _, kind := range []string{"HTTPScaledObject", "Deployment", "Service", "IngressRoute"} {
		if err := d.k8sClient.DeleteCRD(ctx, kind, namespace, app.Subdomain); err != nil {
			log.Printf("[deployer] Warning: failed to delete %s/%s: %v", kind, app.Subdomain, err)
		}
	}
	// Clean up NetworkPolicy (uses -netpol suffix)
	if err := d.k8sClient.DeleteCRD(ctx, "NetworkPolicy", namespace, app.Subdomain+"-netpol"); err != nil {
		log.Printf("[deployer] Warning: failed to delete NetworkPolicy/%s-netpol: %v", app.Subdomain, err)
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
