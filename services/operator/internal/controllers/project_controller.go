package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
)

const projectFinalizer = "zenith.dev/project-cleanup"

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

	// Handle deletion
	if !project.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&project, projectFinalizer) {
			logger.Info("Cleaning up project resources", "name", project.Name)
			// Namespace cleanup is handled by K8s cascade when namespace is deleted.
			// The namespace has an owner reference so it will be garbage collected.
			// For cluster-scoped Project owning a namespace, we delete it explicitly.
			nsName := fmt.Sprintf("zenith-%s", project.Name)
			ns := &corev1.Namespace{}
			if err := r.Get(ctx, client.ObjectKey{Name: nsName}, ns); err == nil {
				if err := r.Delete(ctx, ns); client.IgnoreNotFound(err) != nil {
					return ctrl.Result{}, err
				}
			}

			controllerutil.RemoveFinalizer(&project, projectFinalizer)
			if err := r.Update(ctx, &project); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&project, projectFinalizer) {
		controllerutil.AddFinalizer(&project, projectFinalizer)
		if err := r.Update(ctx, &project); err != nil {
			return ctrl.Result{}, err
		}
	}

	nsName := fmt.Sprintf("zenith-%s", project.Name)

	// Step 1: CreateOrUpdate Namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
	nsResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["zenith.dev/project"] = project.Name
		ns.Labels["zenith.dev/owner"] = project.Spec.Owner
		ns.Labels["zenith.dev/plan"] = project.Spec.Plan
		ns.Labels["app.kubernetes.io/managed-by"] = "zenith-operator"
		return nil
	})
	if err != nil {
		r.Recorder.Eventf(&project, corev1.EventTypeWarning, "NamespaceFailed", "Failed to ensure namespace %s: %v", nsName, err)
		return ctrl.Result{}, err
	}
	if nsResult != controllerutil.OperationResultNone {
		logger.Info("Namespace reconciled", "namespace", nsName, "operation", nsResult)
		r.Recorder.Eventf(&project, corev1.EventTypeNormal, "NamespaceReady", "Namespace %s %s", nsName, nsResult)
	}

	// Step 2: Determine quota limits based on plan
	maxApps, maxDatabases, maxStorageGB, cpuLimit, memoryLimit := planQuotas(project.Spec.Plan)

	// Allow overrides from spec
	if project.Spec.ResourceQuota != nil {
		if project.Spec.ResourceQuota.MaxApps > 0 {
			maxApps = project.Spec.ResourceQuota.MaxApps
		}
		if project.Spec.ResourceQuota.MaxDatabases > 0 {
			maxDatabases = project.Spec.ResourceQuota.MaxDatabases
		}
		if project.Spec.ResourceQuota.MaxStorageGB > 0 {
			maxStorageGB = project.Spec.ResourceQuota.MaxStorageGB
		}
	}

	// Step 3: CreateOrUpdate ResourceQuota
	rq := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-quota", project.Name),
			Namespace: nsName,
		},
	}
	rqResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, rq, func() error {
		rq.Spec = corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceLimitsCPU:       resource.MustParse(cpuLimit),
				corev1.ResourceLimitsMemory:    resource.MustParse(memoryLimit),
				corev1.ResourceRequestsCPU:     resource.MustParse(cpuLimit),
				corev1.ResourceRequestsMemory:  resource.MustParse(memoryLimit),
				corev1.ResourcePods:            *resource.NewQuantity(int64(maxApps*3), resource.DecimalSI),
				corev1.ResourceServices:        *resource.NewQuantity(int64(maxApps+maxDatabases), resource.DecimalSI),
				corev1.ResourceRequestsStorage: resource.MustParse(fmt.Sprintf("%dGi", maxStorageGB)),
			},
		}
		return nil
	})
	if err != nil {
		r.Recorder.Eventf(&project, corev1.EventTypeWarning, "ResourceQuotaFailed", "Failed to ensure ResourceQuota: %v", err)
		return ctrl.Result{}, err
	}
	if rqResult != controllerutil.OperationResultNone {
		logger.Info("ResourceQuota reconciled", "operation", rqResult)
	}

	// Step 4: CreateOrUpdate LimitRange
	lr := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-limits", project.Name),
			Namespace: nsName,
		},
	}
	lrResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, lr, func() error {
		lr.Spec = corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					Default: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("250m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					DefaultRequest: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
					Max: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(cpuLimit),
						corev1.ResourceMemory: resource.MustParse(memoryLimit),
					},
					Min: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("50m"),
						corev1.ResourceMemory: resource.MustParse("32Mi"),
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		r.Recorder.Eventf(&project, corev1.EventTypeWarning, "LimitRangeFailed", "Failed to ensure LimitRange: %v", err)
		return ctrl.Result{}, err
	}
	if lrResult != controllerutil.OperationResultNone {
		logger.Info("LimitRange reconciled", "operation", lrResult)
	}

	// Step 5: Count current apps and databases in namespace
	var appList zenithv1.AppList
	if err := r.List(ctx, &appList, client.InNamespace(nsName)); err != nil {
		logger.Error(err, "Failed to list apps in namespace")
	}

	var dbList zenithv1.DatabaseList
	if err := r.List(ctx, &dbList, client.InNamespace(nsName)); err != nil {
		logger.Error(err, "Failed to list databases in namespace")
	}

	// Update status
	project.Status.Phase = "Active"
	project.Status.Namespace = nsName
	project.Status.AppCount = len(appList.Items)
	project.Status.DatabaseCount = len(dbList.Items)

	if err := r.Status().Update(ctx, &project); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// planQuotas returns resource limits for a given plan.
func planQuotas(plan string) (maxApps, maxDatabases, maxStorageGB int, cpuLimit, memoryLimit string) {
	switch plan {
	case "pro":
		return 25, 10, 100, "8", "16Gi"
	case "enterprise":
		return 100, 50, 1000, "32", "64Gi"
	default: // free
		return 5, 2, 10, "2", "4Gi"
	}
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.Project{}).
		Owns(&corev1.Namespace{}).
		Named("project").
		Complete(r)
}
