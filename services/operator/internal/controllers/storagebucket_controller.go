package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
	"github.com/dotechhq/zenith/services/operator/internal/provider/hetzner"
)

type StorageBucketReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Hetzner  *hetzner.Client
}

func NewStorageBucketReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder, h *hetzner.Client) *StorageBucketReconciler {
	return &StorageBucketReconciler{Client: c, Scheme: s, Recorder: r, Hetzner: h}
}

func (r *StorageBucketReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var sb zenithv1.StorageBucket
	if err := r.Get(ctx, req.NamespacedName, &sb); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !sb.DeletionTimestamp.IsZero() {
		logger.Info("StorageBucket being deleted", "name", sb.Name)
		return ctrl.Result{}, nil
	}

	if sb.Status.Phase == "" {
		sb.Status.Phase = "Creating"
		if err := r.Status().Update(ctx, &sb); err != nil {
			return ctrl.Result{}, err
		}
	}

	if r.Hetzner.IsConfigured() {
		region := sb.Spec.Region
		if region == "" {
			region = "fsn1"
		}
		bucket, err := r.Hetzner.CreateBucket(ctx, sb.Name, region)
		if err != nil {
			r.Recorder.Eventf(&sb, corev1.EventTypeWarning, "BucketCreationFailed", "Failed to create bucket: %v", err)
			sb.Status.Phase = "Failed"
			_ = r.Status().Update(ctx, &sb)
			return ctrl.Result{}, err
		}

		sb.Status.BucketName = bucket.Name
		sb.Status.Endpoint = bucket.Endpoint
	}

	sb.Status.Phase = "Ready"
	if err := r.Status().Update(ctx, &sb); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&sb, corev1.EventTypeNormal, "Ready", "Storage bucket is ready")
	return ctrl.Result{}, nil
}

func (r *StorageBucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.StorageBucket{}).
		Named("storagebucket").
		Complete(r)
}
