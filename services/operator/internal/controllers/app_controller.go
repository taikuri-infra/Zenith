package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
)

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

	if !app.DeletionTimestamp.IsZero() {
		logger.Info("App being deleted", "name", app.Name)
		return ctrl.Result{}, nil
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

	// Ensure Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(deployment), deployment); err != nil {
		if errors.IsNotFound(err) {
			deployment.Spec = appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{MatchLabels: labels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: labels},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  app.Name,
								Image: app.Spec.Image,
								Ports: []corev1.ContainerPort{
									{ContainerPort: app.Spec.Port, Protocol: corev1.ProtocolTCP},
								},
								Env: app.Spec.Env,
							},
						},
					},
				},
			}

			if app.Spec.HealthCheck != nil {
				deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: app.Spec.HealthCheck.Path,
							Port: intstr.FromInt32(app.Spec.Port),
						},
					},
					PeriodSeconds: app.Spec.HealthCheck.IntervalSeconds,
				}
			}

			logger.Info("Creating deployment", "name", app.Name)
			if err := r.Create(ctx, deployment); err != nil {
				r.Recorder.Eventf(&app, corev1.EventTypeWarning, "DeploymentFailed", "Failed to create deployment: %v", err)
				return ctrl.Result{}, err
			}
			r.Recorder.Event(&app, corev1.EventTypeNormal, "DeploymentCreated", "Created deployment")
		} else {
			return ctrl.Result{}, err
		}
	}

	// Ensure Service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(svc), svc); err != nil {
		if errors.IsNotFound(err) {
			svc.Spec = corev1.ServiceSpec{
				Selector: labels,
				Ports: []corev1.ServicePort{
					{
						Port:       app.Spec.Port,
						TargetPort: intstr.FromInt32(app.Spec.Port),
						Protocol:   corev1.ProtocolTCP,
					},
				},
			}
			logger.Info("Creating service", "name", app.Name)
			if err := r.Create(ctx, svc); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Event(&app, corev1.EventTypeNormal, "ServiceCreated", "Created service")
		} else {
			return ctrl.Result{}, err
		}
	}

	// Update status
	app.Status.Phase = "Running"
	app.Status.ReadyReplicas = deployment.Status.ReadyReplicas
	app.Status.CurrentImage = app.Spec.Image
	app.Status.InternalURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", app.Name, app.Namespace, app.Spec.Port)

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
		Named("app").
		Complete(r)
}
