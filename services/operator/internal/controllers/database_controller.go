package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
	"github.com/dotechhq/zenith/services/operator/internal/provider/hetzner"
)

type DatabaseReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Hetzner  *hetzner.Client
}

func NewDatabaseReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder, h *hetzner.Client) *DatabaseReconciler {
	return &DatabaseReconciler{Client: c, Scheme: s, Recorder: r, Hetzner: h}
}

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var db zenithv1.Database
	if err := r.Get(ctx, req.NamespacedName, &db); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !db.DeletionTimestamp.IsZero() {
		logger.Info("Database being deleted", "name", db.Name)
		return ctrl.Result{}, nil
	}

	// Set initial status
	if db.Status.Phase == "" {
		db.Status.Phase = "Provisioning"
		if err := r.Status().Update(ctx, &db); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(&db, corev1.EventTypeNormal, "Provisioning", "Starting database provisioning")
	}

	// Step 1: Ensure Hetzner Volume
	if db.Status.HetznerVolumeID == "" && r.Hetzner.IsConfigured() {
		sizeGB := int(db.Spec.Storage.Value() / (1024 * 1024 * 1024))
		if sizeGB < 10 {
			sizeGB = 10
		}

		vol, err := r.Hetzner.CreateVolume(ctx, fmt.Sprintf("zenith-db-%s", db.Name), sizeGB, "fsn1")
		if err != nil {
			r.Recorder.Eventf(&db, corev1.EventTypeWarning, "VolumeCreationFailed", "Failed to create volume: %v", err)
			db.Status.Phase = "Failed"
			_ = r.Status().Update(ctx, &db)
			return ctrl.Result{}, err
		}

		db.Status.HetznerVolumeID = fmt.Sprintf("%d", vol.ID)
		r.Recorder.Eventf(&db, corev1.EventTypeNormal, "VolumeCreated", "Created Hetzner volume %d", vol.ID)
	}

	// Step 2: Create service-specific CRs based on engine
	// TODO: Create CNPG Cluster for postgresql, Redis CR for redis, etc.
	logger.Info("Reconciling database", "engine", db.Spec.Engine, "version", db.Spec.Version)

	// Step 3: Generate connection secret
	secretName := fmt.Sprintf("%s-conn", db.Name)
	db.Status.SecretName = secretName

	// Step 4: Set default port per engine
	switch db.Spec.Engine {
	case "postgresql":
		db.Status.Port = 5432
		db.Status.Host = fmt.Sprintf("%s.%s.svc.cluster.local", db.Name, db.Namespace)
	case "mysql":
		db.Status.Port = 3306
		db.Status.Host = fmt.Sprintf("%s.%s.svc.cluster.local", db.Name, db.Namespace)
	case "mongodb":
		db.Status.Port = 27017
		db.Status.Host = fmt.Sprintf("%s.%s.svc.cluster.local", db.Name, db.Namespace)
	case "redis":
		db.Status.Port = 6379
		db.Status.Host = fmt.Sprintf("%s.%s.svc.cluster.local", db.Name, db.Namespace)
	}

	db.Status.Phase = "Ready"
	db.Status.ConnectionString = fmt.Sprintf("%s://%s:%d", db.Spec.Engine, db.Status.Host, db.Status.Port)

	if err := r.Status().Update(ctx, &db); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&db, corev1.EventTypeNormal, "Ready", "Database is ready")
	return ctrl.Result{}, nil
}

func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.Database{}).
		Named("database").
		Complete(r)
}
