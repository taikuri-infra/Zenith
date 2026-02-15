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
)

type AuthRealmReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func NewAuthRealmReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder) *AuthRealmReconciler {
	return &AuthRealmReconciler{Client: c, Scheme: s, Recorder: r}
}

func (r *AuthRealmReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var realm zenithv1.AuthRealm
	if err := r.Get(ctx, req.NamespacedName, &realm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !realm.DeletionTimestamp.IsZero() {
		logger.Info("AuthRealm being deleted", "name", realm.Name)
		return ctrl.Result{}, nil
	}

	if realm.Status.Phase == "" {
		realm.Status.Phase = "Provisioning"
		if err := r.Status().Update(ctx, &realm); err != nil {
			return ctrl.Result{}, err
		}
	}

	// TODO: Create Zenith Auth realm (custom auth service)
	logger.Info("Reconciling AuthRealm", "name", realm.Name, "providers", len(realm.Spec.Providers))

	realm.Status.Phase = "Ready"
	realm.Status.ClientCount = len(realm.Spec.Clients)
	realm.Status.Endpoint = fmt.Sprintf("https://auth.zenith.dev/realms/%s/.well-known/openid-configuration", realm.Name)

	if err := r.Status().Update(ctx, &realm); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&realm, corev1.EventTypeNormal, "Ready", "AuthRealm is ready")
	return ctrl.Result{}, nil
}

func (r *AuthRealmReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.AuthRealm{}).
		Named("authrealm").
		Complete(r)
}
