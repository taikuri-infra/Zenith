package controllers

import (
	"context"
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
	"github.com/dotechhq/zenith/services/operator/internal/provider/hetzner"
)

const storageBucketFinalizer = "zenith.dev/storagebucket-cleanup"

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

	// Handle deletion
	if !sb.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&sb, storageBucketFinalizer) {
			logger.Info("Cleaning up storage bucket resources", "name", sb.Name)

			// Delete bucket via Hetzner client
			if r.Hetzner.IsConfigured() && sb.Status.BucketName != "" {
				if err := r.Hetzner.DeleteBucket(ctx, sb.Status.BucketName); err != nil {
					logger.Error(err, "Failed to delete Hetzner bucket", "bucketName", sb.Status.BucketName)
					return ctrl.Result{}, err
				}
				r.Recorder.Event(&sb, corev1.EventTypeNormal, "BucketDeleted", "Deleted Hetzner storage bucket")
			}

			controllerutil.RemoveFinalizer(&sb, storageBucketFinalizer)
			if err := r.Update(ctx, &sb); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&sb, storageBucketFinalizer) {
		controllerutil.AddFinalizer(&sb, storageBucketFinalizer)
		if err := r.Update(ctx, &sb); err != nil {
			return ctrl.Result{}, err
		}
	}

	if sb.Status.Phase == "" {
		sb.Status.Phase = "Creating"
		if err := r.Status().Update(ctx, &sb); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Step 1: Create bucket via Hetzner if not exists
	if sb.Status.BucketName == "" && r.Hetzner.IsConfigured() {
		region := sb.Spec.Region
		if region == "" {
			region = "fsn1"
		}

		bucketName := fmt.Sprintf("zenith-%s-%s", sb.Namespace, sb.Name)
		bucket, err := r.Hetzner.CreateBucket(ctx, bucketName, region)
		if err != nil {
			r.Recorder.Eventf(&sb, corev1.EventTypeWarning, "BucketCreationFailed", "Failed to create bucket: %v", err)
			sb.Status.Phase = "Failed"
			_ = r.Status().Update(ctx, &sb)
			return ctrl.Result{}, err
		}

		sb.Status.BucketName = bucket.Name
		sb.Status.Endpoint = bucket.Endpoint
		r.Recorder.Event(&sb, corev1.EventTypeNormal, "BucketCreated", "Created Hetzner storage bucket")

		if err := r.Status().Update(ctx, &sb); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Step 2: CreateOrUpdate Secret with S3 credentials
	secretName := fmt.Sprintf("%s-s3-credentials", sb.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: sb.Namespace,
		},
	}
	secretResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Labels = map[string]string{
			"app.kubernetes.io/managed-by": "zenith-operator",
			"zenith.dev/storagebucket":     sb.Name,
		}

		// Preserve existing credentials if already generated
		existingAccessKey := ""
		existingSecretKey := ""
		if secret.Data != nil {
			if ak, ok := secret.Data["access-key"]; ok {
				existingAccessKey = string(ak)
			}
			if sk, ok := secret.Data["secret-key"]; ok {
				existingSecretKey = string(sk)
			}
		}

		accessKey := existingAccessKey
		secretKey := existingSecretKey
		if accessKey == "" {
			accessKey = generatePassword(20)
		}
		if secretKey == "" {
			secretKey = generatePassword(40)
		}

		endpoint := sb.Status.Endpoint
		if endpoint == "" {
			region := sb.Spec.Region
			if region == "" {
				region = "fsn1"
			}
			endpoint = fmt.Sprintf("https://%s.your-objectstorage.com", region)
		}

		bucketName := sb.Status.BucketName
		if bucketName == "" {
			bucketName = fmt.Sprintf("zenith-%s-%s", sb.Namespace, sb.Name)
		}

		region := sb.Spec.Region
		if region == "" {
			region = "fsn1"
		}

		secret.Type = corev1.SecretTypeOpaque
		secret.Data = map[string][]byte{
			"endpoint":   []byte(endpoint),
			"bucket":     []byte(bucketName),
			"access-key": []byte(accessKey),
			"secret-key": []byte(secretKey),
			"region":     []byte(region),
		}

		return ctrl.SetControllerReference(&sb, secret, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&sb, corev1.EventTypeWarning, "SecretFailed", "Failed to ensure S3 credentials secret: %v", err)
		return ctrl.Result{}, err
	}
	if secretResult != controllerutil.OperationResultNone {
		logger.Info("S3 credentials secret reconciled", "operation", secretResult)
	}

	// Update status
	sb.Status.Phase = "Ready"
	sb.Status.SecretName = secretName
	if err := r.Status().Update(ctx, &sb); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&sb, corev1.EventTypeNormal, "Ready", "Storage bucket is ready")
	return ctrl.Result{}, nil
}

func (r *StorageBucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.StorageBucket{}).
		Owns(&corev1.Secret{}).
		Named("storagebucket").
		Complete(r)
}
