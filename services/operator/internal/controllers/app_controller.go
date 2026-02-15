package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
)

const appFinalizer = "zenith.dev/app-cleanup"

type AppReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func NewAppReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder) *AppReconciler {
	return &AppReconciler{Client: c, Scheme: s, Recorder: r}
}

func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var app zenithv1.App
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !app.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&app, appFinalizer) {
			logger.Info("Cleaning up app resources", "name", app.Name)
			// K8s cascade handles children via owner references
			controllerutil.RemoveFinalizer(&app, appFinalizer)
			if err := r.Update(ctx, &app); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&app, appFinalizer) {
		controllerutil.AddFinalizer(&app, appFinalizer)
		if err := r.Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
	}

	replicas := int32(1)
	if app.Spec.Replicas != nil {
		replicas = *app.Spec.Replicas
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       app.Name,
		"app.kubernetes.io/managed-by": "zenith-operator",
		"zenith.dev/app":               app.Name,
	}

	// Step 1: CreateOrUpdate Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}
	depResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Labels = labels

		// Build container
		container := corev1.Container{
			Name:  app.Name,
			Image: app.Spec.Image,
			Ports: []corev1.ContainerPort{
				{ContainerPort: app.Spec.Port, Protocol: corev1.ProtocolTCP},
			},
			Env: app.Spec.Env,
		}

		// Set resource limits/requests
		if app.Spec.Resources != nil {
			container.Resources = corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    app.Spec.Resources.CPU,
					corev1.ResourceMemory: app.Spec.Resources.Memory,
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(app.Spec.Resources.CPU.MilliValue()/2, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(app.Spec.Resources.Memory.Value()/2, resource.BinarySI),
				},
			}
		}

		// Set health check probes
		if app.Spec.HealthCheck != nil {
			healthPort := app.Spec.Port
			if app.Spec.HealthCheck.Port > 0 {
				healthPort = app.Spec.HealthCheck.Port
			}
			interval := int32(30)
			if app.Spec.HealthCheck.IntervalSeconds > 0 {
				interval = app.Spec.HealthCheck.IntervalSeconds
			}

			probe := &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: app.Spec.HealthCheck.Path,
						Port: intstr.FromInt32(healthPort),
					},
				},
				PeriodSeconds:    interval,
				TimeoutSeconds:   5,
				FailureThreshold: 3,
			}
			container.ReadinessProbe = probe
			container.LivenessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: app.Spec.HealthCheck.Path,
						Port: intstr.FromInt32(healthPort),
					},
				},
				PeriodSeconds:       interval,
				TimeoutSeconds:      5,
				FailureThreshold:    3,
				InitialDelaySeconds: 15,
			}
		}

		maxUnavailable := intstr.FromString("25%")
		maxSurge := intstr.FromString("25%")

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
		}

		return ctrl.SetControllerReference(&app, deployment, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&app, corev1.EventTypeWarning, "DeploymentFailed", "Failed to ensure deployment: %v", err)
		return ctrl.Result{}, err
	}
	if depResult != controllerutil.OperationResultNone {
		logger.Info("Deployment reconciled", "name", app.Name, "operation", depResult)
		r.Recorder.Eventf(&app, corev1.EventTypeNormal, "DeploymentReady", "Deployment %s", depResult)
	}

	// Step 2: CreateOrUpdate Service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}
	svcResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Labels = labels
		svc.Spec = corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       app.Spec.Port,
					TargetPort: intstr.FromInt32(app.Spec.Port),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		}
		return ctrl.SetControllerReference(&app, svc, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&app, corev1.EventTypeWarning, "ServiceFailed", "Failed to ensure service: %v", err)
		return ctrl.Result{}, err
	}
	if svcResult != controllerutil.OperationResultNone {
		logger.Info("Service reconciled", "name", app.Name, "operation", svcResult)
		r.Recorder.Eventf(&app, corev1.EventTypeNormal, "ServiceReady", "Service %s", svcResult)
	}

	// Step 3: If domain specified, CreateOrUpdate Ingress
	if app.Spec.Domain != "" {
		ingress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name,
				Namespace: app.Namespace,
			},
		}
		ingressResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
			ingress.Labels = labels
			ingress.Annotations = map[string]string{
				"kubernetes.io/ingress.class":            "nginx",
				"cert-manager.io/cluster-issuer":         "letsencrypt-prod",
				"nginx.ingress.kubernetes.io/ssl-redirect": "true",
			}

			pathType := networkingv1.PathTypePrefix
			ingress.Spec = networkingv1.IngressSpec{
				TLS: []networkingv1.IngressTLS{
					{
						Hosts:      []string{app.Spec.Domain},
						SecretName: fmt.Sprintf("%s-tls", app.Name),
					},
				},
				Rules: []networkingv1.IngressRule{
					{
						Host: app.Spec.Domain,
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: &pathType,
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: app.Name,
												Port: networkingv1.ServiceBackendPort{
													Number: app.Spec.Port,
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
			return ctrl.SetControllerReference(&app, ingress, r.Scheme)
		})
		if err != nil {
			r.Recorder.Eventf(&app, corev1.EventTypeWarning, "IngressFailed", "Failed to ensure ingress: %v", err)
			return ctrl.Result{}, err
		}
		if ingressResult != controllerutil.OperationResultNone {
			logger.Info("Ingress reconciled", "name", app.Name, "operation", ingressResult)
		}
	}

	// Step 4: If autoscale configured, CreateOrUpdate HPA
	if app.Spec.AutoScale != nil {
		hpa := &autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name,
				Namespace: app.Namespace,
			},
		}
		hpaResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, hpa, func() error {
			hpa.Labels = labels
			hpa.Spec = autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       app.Name,
				},
				MinReplicas: &app.Spec.AutoScale.MinReplicas,
				MaxReplicas: app.Spec.AutoScale.MaxReplicas,
				Metrics: []autoscalingv2.MetricSpec{
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: corev1.ResourceCPU,
							Target: autoscalingv2.MetricTarget{
								Type:               autoscalingv2.UtilizationMetricType,
								AverageUtilization: &app.Spec.AutoScale.TargetCPUPercent,
							},
						},
					},
				},
			}
			return ctrl.SetControllerReference(&app, hpa, r.Scheme)
		})
		if err != nil {
			r.Recorder.Eventf(&app, corev1.EventTypeWarning, "HPAFailed", "Failed to ensure HPA: %v", err)
			return ctrl.Result{}, err
		}
		if hpaResult != controllerutil.OperationResultNone {
			logger.Info("HPA reconciled", "name", app.Name, "operation", hpaResult)
		}
	}

	// Step 5: Update status
	// Re-fetch deployment to get current status
	if err := r.Get(ctx, client.ObjectKeyFromObject(deployment), deployment); err == nil {
		app.Status.ReadyReplicas = deployment.Status.ReadyReplicas
	}

	app.Status.Phase = "Running"
	app.Status.CurrentImage = app.Spec.Image
	app.Status.InternalURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", app.Name, app.Namespace, app.Spec.Port)
	now := metav1.NewTime(time.Now())
	app.Status.LastDeployedAt = &now

	if app.Spec.Domain != "" {
		app.Status.URL = fmt.Sprintf("https://%s", app.Spec.Domain)
	}

	if err := r.Status().Update(ctx, &app); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.App{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Named("app").
		Complete(r)
}
