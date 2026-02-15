package controllers

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
)

const crossplaneFinalizer = "zenith.dev/crossplane-cleanup"

// CrossplaneReconciler reconciles CrossplaneResource objects by creating
// corresponding Crossplane managed resources using the unstructured client.
type CrossplaneReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// NewCrossplaneReconciler creates a new CrossplaneReconciler.
func NewCrossplaneReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder) *CrossplaneReconciler {
	return &CrossplaneReconciler{Client: c, Scheme: s, Recorder: r}
}

// Reconcile handles CrossplaneResource create/update/delete events.
func (r *CrossplaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cr zenithv1.CrossplaneResource
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cr, crossplaneFinalizer) {
			logger.Info("Cleaning up Crossplane resource", "name", cr.Name)

			if err := r.deleteCrossplaneResource(ctx, &cr); err != nil {
				logger.Error(err, "Failed to delete Crossplane managed resource")
				// Continue to remove finalizer even if delete fails
			}

			controllerutil.RemoveFinalizer(&cr, crossplaneFinalizer)
			if err := r.Update(ctx, &cr); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&cr, crossplaneFinalizer) {
		controllerutil.AddFinalizer(&cr, crossplaneFinalizer)
		if err := r.Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Build the Crossplane managed resource
	managedResource := r.buildCrossplaneResource(&cr)

	// Create or update the Crossplane resource using unstructured client
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(managedResource.GroupVersionKind())
	err := r.Get(ctx, types.NamespacedName{Name: managedResource.GetName()}, existing)

	if err != nil {
		// Resource does not exist, create it
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check existing resource: %w", err)
		}

		logger.Info("Creating Crossplane managed resource",
			"name", managedResource.GetName(),
			"kind", cr.Spec.ResourceKind,
			"provider", cr.Spec.Provider,
		)

		if err := r.Create(ctx, managedResource); err != nil {
			r.Recorder.Eventf(&cr, corev1.EventTypeWarning, "CreateFailed",
				"Failed to create Crossplane resource: %v", err)

			cr.Status.Phase = "Failed"
			cr.Status.Message = fmt.Sprintf("Failed to create: %v", err)
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{}, err
		}

		r.Recorder.Event(&cr, corev1.EventTypeNormal, "Created", "Crossplane managed resource created")
	} else {
		// Resource exists, update spec
		managedResource.SetResourceVersion(existing.GetResourceVersion())
		if err := r.Update(ctx, managedResource); err != nil {
			r.Recorder.Eventf(&cr, corev1.EventTypeWarning, "UpdateFailed",
				"Failed to update Crossplane resource: %v", err)
			return ctrl.Result{}, err
		}
	}

	// Read back the Crossplane resource status
	readBack := &unstructured.Unstructured{}
	readBack.SetGroupVersionKind(managedResource.GroupVersionKind())
	if err := r.Get(ctx, types.NamespacedName{Name: managedResource.GetName()}, readBack); err == nil {
		r.updateStatusFromCrossplane(ctx, &cr, readBack)
	} else {
		cr.Status.Phase = "Provisioning"
		cr.Status.CrossplaneResourceName = managedResource.GetName()
		cr.Status.Message = "Crossplane resource created, waiting for ready"
	}

	if err := r.Status().Update(ctx, &cr); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// buildCrossplaneResource creates an unstructured Crossplane managed resource from the CR spec.
func (r *CrossplaneReconciler) buildCrossplaneResource(cr *zenithv1.CrossplaneResource) *unstructured.Unstructured {
	resource := &unstructured.Unstructured{}

	// Determine the GVK for the Crossplane resource
	apiVersion := cr.Spec.ResourceAPIVersion
	if apiVersion == "" {
		apiVersion = defaultAPIVersion(cr.Spec.Provider, cr.Spec.ResourceKind)
	}

	resource.SetAPIVersion(apiVersion)
	resource.SetKind(cr.Spec.ResourceKind)
	resource.SetName(fmt.Sprintf("zenith-%s-%s", cr.Namespace, cr.Name))
	resource.SetLabels(map[string]string{
		"app.kubernetes.io/managed-by": "zenith-operator",
		"zenith.dev/crossplane-resource": cr.Name,
		"zenith.dev/namespace":           cr.Namespace,
	})

	// Build the spec
	spec := map[string]interface{}{
		"deletionPolicy": cr.Spec.DeletionPolicy,
		"providerConfigRef": map[string]interface{}{
			"name": cr.Spec.ProviderConfigRef,
		},
	}

	// Add forProvider configuration from the config map
	forProvider := make(map[string]interface{})
	for k, v := range cr.Spec.Config {
		forProvider[k] = v
	}
	spec["forProvider"] = forProvider

	// Add connection secret reference if specified
	if cr.Spec.WriteConnectionSecretToRef != nil {
		spec["writeConnectionSecretToRef"] = map[string]interface{}{
			"name":      cr.Spec.WriteConnectionSecretToRef.Name,
			"namespace": cr.Namespace,
		}
	}

	resource.Object["spec"] = spec

	return resource
}

// defaultAPIVersion returns a default Crossplane API version based on provider and resource kind.
func defaultAPIVersion(provider, resourceKind string) string {
	kindLower := strings.ToLower(resourceKind)

	switch provider {
	case "aws":
		switch kindLower {
		case "bucket":
			return "s3.aws.upbound.io/v1beta1"
		case "instance":
			return "ec2.aws.upbound.io/v1beta1"
		case "database", "rdsinstance":
			return "rds.aws.upbound.io/v1beta1"
		default:
			return "aws.upbound.io/v1beta1"
		}
	case "gcp":
		switch kindLower {
		case "bucket":
			return "storage.gcp.upbound.io/v1beta1"
		case "instance":
			return "compute.gcp.upbound.io/v1beta1"
		default:
			return "gcp.upbound.io/v1beta1"
		}
	case "azure":
		switch kindLower {
		case "storageaccount":
			return "storage.azure.upbound.io/v1beta1"
		case "virtualmachine":
			return "compute.azure.upbound.io/v1beta1"
		default:
			return "azure.upbound.io/v1beta1"
		}
	case "hetzner":
		return "hcloud.crossplane.io/v1alpha1"
	default:
		return provider + ".crossplane.io/v1alpha1"
	}
}

// deleteCrossplaneResource deletes the Crossplane managed resource.
func (r *CrossplaneReconciler) deleteCrossplaneResource(ctx context.Context, cr *zenithv1.CrossplaneResource) error {
	if cr.Status.CrossplaneResourceName == "" {
		return nil
	}

	apiVersion := cr.Spec.ResourceAPIVersion
	if apiVersion == "" {
		apiVersion = defaultAPIVersion(cr.Spec.Provider, cr.Spec.ResourceKind)
	}

	resource := &unstructured.Unstructured{}
	resource.SetAPIVersion(apiVersion)
	resource.SetKind(cr.Spec.ResourceKind)
	resource.SetName(cr.Status.CrossplaneResourceName)

	if err := r.Delete(ctx, resource); err != nil {
		return client.IgnoreNotFound(err)
	}

	r.Recorder.Event(cr, corev1.EventTypeNormal, "Deleted", "Crossplane managed resource deleted")
	return nil
}

// updateStatusFromCrossplane reads the Crossplane resource conditions and
// propagates them back to the Zenith CrossplaneResource status.
func (r *CrossplaneReconciler) updateStatusFromCrossplane(ctx context.Context, cr *zenithv1.CrossplaneResource, managed *unstructured.Unstructured) {
	cr.Status.CrossplaneResourceName = managed.GetName()

	// Check for the Crossplane "Ready" condition
	conditions, found, _ := unstructured.NestedSlice(managed.Object, "status", "conditions")
	if !found {
		cr.Status.Phase = "Provisioning"
		cr.Status.Message = "Waiting for Crossplane resource conditions"
		return
	}

	ready := false
	for _, c := range conditions {
		condMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _ := condMap["type"].(string)
		condStatus, _ := condMap["status"].(string)

		if condType == "Ready" && condStatus == "True" {
			ready = true
			break
		}
	}

	cr.Status.CrossplaneReady = ready

	if ready {
		cr.Status.Phase = "Ready"
		cr.Status.Message = "Crossplane resource is ready"
	} else {
		cr.Status.Phase = "Provisioning"
		cr.Status.Message = "Crossplane resource is not yet ready"
	}

	// Extract external name if available
	externalName, found, _ := unstructured.NestedString(managed.Object, "metadata", "annotations", "crossplane.io/external-name")
	if found {
		cr.Status.ExternalName = externalName
	}

	// Extract connection secret name if available
	secretName, found, _ := unstructured.NestedString(managed.Object, "spec", "writeConnectionSecretToRef", "name")
	if found {
		cr.Status.ConnectionSecretName = secretName
	}

	// Propagate conditions
	cr.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionStatus(boolToString(ready)),
			LastTransitionTime: metav1.Now(),
			Reason:             phaseReason(ready),
			Message:            cr.Status.Message,
		},
	}
}

func boolToString(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

func phaseReason(ready bool) string {
	if ready {
		return "ResourceReady"
	}
	return "ResourceNotReady"
}

// SetupWithManager sets up the controller with the Manager.
func (r *CrossplaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.CrossplaneResource{}).
		Named("crossplane").
		Complete(r)
}

// crossplaneGVR returns the GroupVersionResource for a Crossplane managed resource.
// This is exported for use in testing.
func crossplaneGVR(apiVersion, kind string) schema.GroupVersionResource {
	parts := strings.SplitN(apiVersion, "/", 2)
	group := ""
	version := apiVersion
	if len(parts) == 2 {
		group = parts[0]
		version = parts[1]
	}
	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: strings.ToLower(kind) + "s",
	}
}
