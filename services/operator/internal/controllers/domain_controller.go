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

type DomainReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Hetzner  *hetzner.Client
}

func NewDomainReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder, h *hetzner.Client) *DomainReconciler {
	return &DomainReconciler{Client: c, Scheme: s, Recorder: r, Hetzner: h}
}

func (r *DomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var domain zenithv1.Domain
	if err := r.Get(ctx, req.NamespacedName, &domain); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !domain.DeletionTimestamp.IsZero() {
		logger.Info("Domain being deleted", "name", domain.Name)
		return ctrl.Result{}, nil
	}

	if domain.Status.Phase == "" {
		domain.Status.Phase = "Configuring"
		if err := r.Status().Update(ctx, &domain); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Step 1: Configure DNS records via Hetzner
	if domain.Spec.DNS != nil && domain.Spec.DNS.AutoConfigure && r.Hetzner.IsConfigured() {
		logger.Info("Configuring DNS for domain", "domain", domain.Spec.Domain)
		// TODO: Create DNS records via Hetzner DNS API
		domain.Status.DNSConfigured = true
	}

	// Step 2: Configure SSL via cert-manager
	if domain.Spec.SSL == nil || domain.Spec.SSL.Enabled {
		logger.Info("Configuring SSL for domain", "domain", domain.Spec.Domain)
		// TODO: Create cert-manager Certificate resource
		domain.Status.SSLReady = true
	}

	// Step 3: Configure Ingress
	// TODO: Create/update Ingress resource pointing to the app

	domain.Status.Phase = "Active"
	if err := r.Status().Update(ctx, &domain); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&domain, corev1.EventTypeNormal, "Active", "Domain is active")
	return ctrl.Result{}, nil
}

func (r *DomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.Domain{}).
		Named("domain").
		Complete(r)
}
