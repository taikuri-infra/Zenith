package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
)

type ProjectReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func NewProjectReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder) *ProjectReconciler {
	return &ProjectReconciler{Client: c, Scheme: s, Recorder: r}
}

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var project zenithv1.Project
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !project.DeletionTimestamp.IsZero() {
		logger.Info("Project being deleted", "name", project.Name)
		return ctrl.Result{}, nil
	}

	nsName := fmt.Sprintf("zenith-%s", project.Name)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"zenith.dev/project": project.Name,
				"zenith.dev/owner":   project.Spec.Owner,
			},
		},
	}

	if err := r.Get(ctx, client.ObjectKeyFromObject(ns), ns); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Creating namespace for project", "namespace", nsName)
			if err := r.Create(ctx, ns); err != nil {
				r.Recorder.Eventf(&project, corev1.EventTypeWarning, "NamespaceCreationFailed", "Failed to create namespace %s: %v", nsName, err)
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(&project, corev1.EventTypeNormal, "NamespaceCreated", "Created namespace %s", nsName)
		} else {
			return ctrl.Result{}, err
		}
	}

	project.Status.Phase = "Active"
	project.Status.Namespace = nsName
	if err := r.Status().Update(ctx, &project); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.Project{}).
		Named("project").
		Complete(r)
}
