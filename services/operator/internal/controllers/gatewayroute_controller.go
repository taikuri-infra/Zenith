package controllers

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
)

const gatewayRouteFinalizer = "zenith.dev/gatewayroute-cleanup"

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

	// Handle deletion
	if !route.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&route, gatewayRouteFinalizer) {
			logger.Info("Cleaning up gateway route resources", "name", route.Name)
			// K8s cascade handles children via owner references
			controllerutil.RemoveFinalizer(&route, gatewayRouteFinalizer)
			if err := r.Update(ctx, &route); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&route, gatewayRouteFinalizer) {
		controllerutil.AddFinalizer(&route, gatewayRouteFinalizer)
		if err := r.Update(ctx, &route); err != nil {
			return ctrl.Result{}, err
		}
	}

	if route.Status.Phase == "" {
		route.Status.Phase = "Configuring"
		if err := r.Status().Update(ctx, &route); err != nil {
			return ctrl.Result{}, err
		}
	}

	labels := map[string]string{
		"app.kubernetes.io/managed-by": "zenith-operator",
		"zenith.dev/gatewayroute":      route.Name,
	}

	// Collect plugin names for annotation
	var pluginNames []string

	// Step 1: If rate limit specified, CreateOrUpdate KongPlugin CR for rate limiting
	if route.Spec.RateLimit != nil {
		rateLimitPluginName := fmt.Sprintf("%s-rate-limiting", route.Name)
		pluginNames = append(pluginNames, rateLimitPluginName)

		kongPlugin := &unstructured.Unstructured{}
		kongPlugin.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "configuration.konghq.com",
			Version: "v1",
			Kind:    "KongPlugin",
		})
		kongPlugin.SetName(rateLimitPluginName)
		kongPlugin.SetNamespace(route.Namespace)

		rlResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, kongPlugin, func() error {
			kongPlugin.SetLabels(labels)
			kongPlugin.Object["plugin"] = "rate-limiting"

			config := map[string]interface{}{}
			if route.Spec.RateLimit.RequestsPerSecond > 0 {
				config["second"] = int64(route.Spec.RateLimit.RequestsPerSecond)
			}
			if route.Spec.RateLimit.RequestsPerMinute > 0 {
				config["minute"] = int64(route.Spec.RateLimit.RequestsPerMinute)
			}
			config["policy"] = "local"
			kongPlugin.Object["config"] = config

			return ctrl.SetControllerReference(&route, kongPlugin, r.Scheme)
		})
		if err != nil {
			// KongPlugin CRD may not be installed
			logger.Error(err, "Failed to ensure KongPlugin for rate limiting - Kong may not be installed")
			r.Recorder.Eventf(&route, corev1.EventTypeWarning, "KongPluginFailed", "Failed to ensure rate-limiting plugin: %v", err)
		} else if rlResult != controllerutil.OperationResultNone {
			logger.Info("Rate limiting KongPlugin reconciled", "operation", rlResult)
		}
	}

	// Step 2: If CORS specified, CreateOrUpdate KongPlugin CR for CORS
	if route.Spec.CORS != nil {
		corsPluginName := fmt.Sprintf("%s-cors", route.Name)
		pluginNames = append(pluginNames, corsPluginName)

		kongPlugin := &unstructured.Unstructured{}
		kongPlugin.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "configuration.konghq.com",
			Version: "v1",
			Kind:    "KongPlugin",
		})
		kongPlugin.SetName(corsPluginName)
		kongPlugin.SetNamespace(route.Namespace)

		corsResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, kongPlugin, func() error {
			kongPlugin.SetLabels(labels)
			kongPlugin.Object["plugin"] = "cors"

			config := map[string]interface{}{}
			if len(route.Spec.CORS.AllowedOrigins) > 0 {
				origins := make([]interface{}, len(route.Spec.CORS.AllowedOrigins))
				for i, o := range route.Spec.CORS.AllowedOrigins {
					origins[i] = o
				}
				config["origins"] = origins
			}
			if len(route.Spec.CORS.AllowedMethods) > 0 {
				methods := make([]interface{}, len(route.Spec.CORS.AllowedMethods))
				for i, m := range route.Spec.CORS.AllowedMethods {
					methods[i] = m
				}
				config["methods"] = methods
			}
			if len(route.Spec.CORS.AllowedHeaders) > 0 {
				headers := make([]interface{}, len(route.Spec.CORS.AllowedHeaders))
				for i, h := range route.Spec.CORS.AllowedHeaders {
					headers[i] = h
				}
				config["headers"] = headers
			}
			config["credentials"] = true
			kongPlugin.Object["config"] = config

			return ctrl.SetControllerReference(&route, kongPlugin, r.Scheme)
		})
		if err != nil {
			logger.Error(err, "Failed to ensure KongPlugin for CORS - Kong may not be installed")
			r.Recorder.Eventf(&route, corev1.EventTypeWarning, "KongPluginFailed", "Failed to ensure CORS plugin: %v", err)
		} else if corsResult != controllerutil.OperationResultNone {
			logger.Info("CORS KongPlugin reconciled", "operation", corsResult)
		}
	}

	// Step 3: If auth enabled, add JWT plugin
	if route.Spec.Auth != nil && route.Spec.Auth.Enabled {
		jwtPluginName := fmt.Sprintf("%s-jwt", route.Name)
		pluginNames = append(pluginNames, jwtPluginName)

		kongPlugin := &unstructured.Unstructured{}
		kongPlugin.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "configuration.konghq.com",
			Version: "v1",
			Kind:    "KongPlugin",
		})
		kongPlugin.SetName(jwtPluginName)
		kongPlugin.SetNamespace(route.Namespace)

		jwtResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, kongPlugin, func() error {
			kongPlugin.SetLabels(labels)
			kongPlugin.Object["plugin"] = "jwt"
			config := map[string]interface{}{
				"key_claim_name": "kid",
			}
			kongPlugin.Object["config"] = config
			return ctrl.SetControllerReference(&route, kongPlugin, r.Scheme)
		})
		if err != nil {
			logger.Error(err, "Failed to ensure KongPlugin for JWT - Kong may not be installed")
		} else if jwtResult != controllerutil.OperationResultNone {
			logger.Info("JWT KongPlugin reconciled", "operation", jwtResult)
		}
	}

	// Add any additional plugins from spec
	for _, plugin := range route.Spec.Plugins {
		pluginNames = append(pluginNames, fmt.Sprintf("%s-%s", route.Name, plugin.Name))
	}

	// Step 4: CreateOrUpdate Ingress with Kong annotations
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("gwr-%s", route.Name),
			Namespace: route.Namespace,
		},
	}
	ingressResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		ingress.Labels = labels
		ingress.Annotations = map[string]string{
			"kubernetes.io/ingress.class": "kong",
			"konghq.com/strip-path":       "true",
		}

		// Add plugin annotations if any plugins configured
		if len(pluginNames) > 0 {
			ingress.Annotations["konghq.com/plugins"] = strings.Join(pluginNames, ",")
		}

		// Add method filtering annotation if methods specified
		if len(route.Spec.Methods) > 0 {
			ingress.Annotations["konghq.com/methods"] = strings.Join(route.Spec.Methods, ",")
		}

		pathType := networkingv1.PathTypeImplementationSpecific
		ingress.Spec = networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     route.Spec.Path,
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: route.Spec.Service.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: route.Spec.Service.Port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		return ctrl.SetControllerReference(&route, ingress, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&route, corev1.EventTypeWarning, "IngressFailed", "Failed to ensure Ingress: %v", err)
		return ctrl.Result{}, err
	}
	if ingressResult != controllerutil.OperationResultNone {
		logger.Info("Gateway Ingress reconciled", "operation", ingressResult)
	}

	// Update status
	route.Status.Phase = "Active"
	route.Status.KongRouteID = fmt.Sprintf("%s/%s", route.Namespace, ingress.Name)

	if err := r.Status().Update(ctx, &route); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&route, corev1.EventTypeNormal, "Active", "Gateway route is active")
	return ctrl.Result{}, nil
}

func (r *GatewayRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.GatewayRoute{}).
		Owns(&networkingv1.Ingress{}).
		Named("gatewayroute").
		Complete(r)
}
