package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
)

const (
	backstageCatalogFinalizer = "zenith.dev/backstage-catalog"
	backstageConfigMapName    = "backstage-catalog"
	backstageNamespace        = "zenith-system"
	backstageAPIVersion       = "backstage.io/v1alpha1"
)

// BackstageCatalogReconciler watches all Zenith CRDs and generates
// Backstage catalog-info.yaml entries in a ConfigMap.
type BackstageCatalogReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// NewBackstageCatalogReconciler creates a new BackstageCatalogReconciler.
func NewBackstageCatalogReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder) *BackstageCatalogReconciler {
	return &BackstageCatalogReconciler{Client: c, Scheme: s, Recorder: r}
}

// backstageEntity represents a single Backstage catalog entity.
type backstageEntity struct {
	APIVersion string                 `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                 `json:"kind" yaml:"kind"`
	Metadata   backstageMetadata      `json:"metadata" yaml:"metadata"`
	Spec       map[string]interface{} `json:"spec" yaml:"spec"`
}

type backstageMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Tags        []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Reconcile is triggered on Project changes and regenerates the entire Backstage catalog.
func (r *BackstageCatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Collect all entities from all CRD types
	entities, err := r.collectAllEntities(ctx)
	if err != nil {
		logger.Error(err, "Failed to collect Backstage entities")
		return ctrl.Result{}, err
	}

	// Serialize entities to JSON
	catalogData, err := json.MarshalIndent(entities, "", "  ")
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to marshal catalog: %w", err)
	}

	// Write to ConfigMap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backstageConfigMapName,
			Namespace: backstageNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Labels = map[string]string{
			"app.kubernetes.io/managed-by": "zenith-operator",
			"app.kubernetes.io/component":  "backstage-catalog",
		}
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data["catalog-info.json"] = string(catalogData)
		return nil
	})
	if err != nil {
		logger.Error(err, "Failed to update Backstage catalog ConfigMap")
		return ctrl.Result{}, err
	}

	logger.Info("Backstage catalog updated", "entities", len(entities))
	return ctrl.Result{}, nil
}

// collectAllEntities gathers all Zenith CRDs and converts them to Backstage entities.
func (r *BackstageCatalogReconciler) collectAllEntities(ctx context.Context) ([]backstageEntity, error) {
	var entities []backstageEntity

	// Collect Projects as Systems
	var projects zenithv1.ProjectList
	if err := r.List(ctx, &projects); err == nil {
		for _, p := range projects.Items {
			entities = append(entities, projectToSystem(p))
		}
	}

	// Collect Apps as Components
	var apps zenithv1.AppList
	if err := r.List(ctx, &apps); err == nil {
		for _, a := range apps.Items {
			entities = append(entities, appToComponent(a))
		}
	}

	// Collect Databases as Resources
	var databases zenithv1.DatabaseList
	if err := r.List(ctx, &databases); err == nil {
		for _, d := range databases.Items {
			entities = append(entities, databaseToResource(d))
		}
	}

	// Collect StorageBuckets as Resources
	var buckets zenithv1.StorageBucketList
	if err := r.List(ctx, &buckets); err == nil {
		for _, b := range buckets.Items {
			entities = append(entities, storageBucketToResource(b))
		}
	}

	// Collect Domains as APIs
	var domains zenithv1.DomainList
	if err := r.List(ctx, &domains); err == nil {
		for _, d := range domains.Items {
			entities = append(entities, domainToAPI(d))
		}
	}

	return entities, nil
}

func projectToSystem(p zenithv1.Project) backstageEntity {
	return backstageEntity{
		APIVersion: backstageAPIVersion,
		Kind:       "System",
		Metadata: backstageMetadata{
			Name:        p.Name,
			Description: p.Spec.DisplayName,
			Annotations: map[string]string{
				"zenith.dev/project": p.Name,
				"zenith.dev/type":    "project",
				"zenith.dev/plan":    p.Spec.Plan,
			},
			Tags: []string{"zenith", "project", p.Spec.Plan},
		},
		Spec: map[string]interface{}{
			"owner": p.Spec.Owner,
			"type":  "project",
		},
	}
}

func appToComponent(a zenithv1.App) backstageEntity {
	project := a.Labels["zenith.dev/project"]
	owner := a.Labels["zenith.dev/owner"]
	if owner == "" {
		owner = "zenith"
	}

	componentSpec := map[string]interface{}{
		"type":      "service",
		"lifecycle": "production",
		"owner":     owner,
		"system":    project,
	}

	if a.Spec.Domain != "" {
		componentSpec["providesApis"] = []string{a.Name + "-api"}
	}

	return backstageEntity{
		APIVersion: backstageAPIVersion,
		Kind:       "Component",
		Metadata: backstageMetadata{
			Name:        a.Name,
			Namespace:   a.Namespace,
			Description: fmt.Sprintf("App %s (%s)", a.Name, a.Spec.Image),
			Annotations: map[string]string{
				"zenith.dev/project": project,
				"zenith.dev/type":    "app",
				"zenith.dev/image":   a.Spec.Image,
			},
			Tags: []string{"zenith", "app"},
		},
		Spec: componentSpec,
	}
}

func databaseToResource(d zenithv1.Database) backstageEntity {
	project := d.Labels["zenith.dev/project"]
	owner := d.Labels["zenith.dev/owner"]
	if owner == "" {
		owner = "zenith"
	}

	return backstageEntity{
		APIVersion: backstageAPIVersion,
		Kind:       "Resource",
		Metadata: backstageMetadata{
			Name:        d.Name,
			Namespace:   d.Namespace,
			Description: fmt.Sprintf("%s %s database", d.Spec.Engine, d.Spec.Version),
			Annotations: map[string]string{
				"zenith.dev/project": project,
				"zenith.dev/type":    "database",
				"zenith.dev/engine":  d.Spec.Engine,
			},
			Tags: []string{"zenith", "database", d.Spec.Engine},
		},
		Spec: map[string]interface{}{
			"type":      "database",
			"owner":     owner,
			"system":    project,
			"lifecycle": "production",
		},
	}
}

func storageBucketToResource(b zenithv1.StorageBucket) backstageEntity {
	project := b.Labels["zenith.dev/project"]
	owner := b.Labels["zenith.dev/owner"]
	if owner == "" {
		owner = "zenith"
	}

	return backstageEntity{
		APIVersion: backstageAPIVersion,
		Kind:       "Resource",
		Metadata: backstageMetadata{
			Name:        b.Name,
			Namespace:   b.Namespace,
			Description: "S3-compatible storage bucket",
			Annotations: map[string]string{
				"zenith.dev/project": project,
				"zenith.dev/type":    "storage",
			},
			Tags: []string{"zenith", "storage", "s3"},
		},
		Spec: map[string]interface{}{
			"type":      "storage",
			"owner":     owner,
			"system":    project,
			"lifecycle": "production",
		},
	}
}

func domainToAPI(d zenithv1.Domain) backstageEntity {
	project := d.Labels["zenith.dev/project"]
	owner := d.Labels["zenith.dev/owner"]
	if owner == "" {
		owner = "zenith"
	}

	return backstageEntity{
		APIVersion: backstageAPIVersion,
		Kind:       "API",
		Metadata: backstageMetadata{
			Name:        d.Name,
			Namespace:   d.Namespace,
			Description: fmt.Sprintf("Domain %s -> %s", d.Spec.Domain, d.Spec.AppRef),
			Annotations: map[string]string{
				"zenith.dev/project": project,
				"zenith.dev/type":    "domain",
				"zenith.dev/domain":  d.Spec.Domain,
				"zenith.dev/app-ref": d.Spec.AppRef,
			},
			Tags: []string{"zenith", "domain"},
		},
		Spec: map[string]interface{}{
			"type":       "openapi",
			"lifecycle":  "production",
			"owner":      owner,
			"system":     project,
			"definition": "# API served at " + d.Spec.Domain,
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
// It watches all Zenith CRD types that should appear in the Backstage catalog.
func (r *BackstageCatalogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.Project{}).
		Named("backstage-catalog").
		Complete(r)
}
