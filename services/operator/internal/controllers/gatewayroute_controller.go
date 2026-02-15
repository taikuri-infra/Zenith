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
)

type GatewayRouteReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func NewGatewayRouteReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder) *GatewayRouteReconciler {
	return &GatewayRouteReconciler{Client: c, Scheme: s, Recorder: r}
}

func (r *GatewayRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var route zenithv1.GatewayRoute
	if err := r.Get(ctx, req.NamespacedName, &route); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !route.DeletionTimestamp.IsZero() {
		logger.Info("GatewayRoute being deleted", "name", route.Name)
		return ctrl.Result{}, nil
	}

	if route.Status.Phase == "" {
		route.Status.Phase = "Configuring"
		if err := r.Status().Update(ctx, &route); err != nil {
			return ctrl.Result{}, err
		}
	}

	// TODO: Create Kong Ingress/Service/Route resources
	logger.Info("Reconciling GatewayRoute", "path", route.Spec.Path, "service", route.Spec.Service.Name)

	route.Status.Phase = "Active"
	if err := r.Status().Update(ctx, &route); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&route, corev1.EventTypeNormal, "Active", "Gateway route is active")
	return ctrl.Result{}, nil
}

func (r *GatewayRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.GatewayRoute{}).
		Named("gatewayroute").
		Complete(r)
}
