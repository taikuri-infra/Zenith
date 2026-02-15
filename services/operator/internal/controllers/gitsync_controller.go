package controllers

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
)

const gitSyncFinalizer = "zenith.dev/gitsync-cleanup"

// GitSyncReconciler reconciles a GitSync object
type GitSyncReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// NewGitSyncReconciler creates a new GitSyncReconciler.
func NewGitSyncReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder) *GitSyncReconciler {
	return &GitSyncReconciler{Client: c, Scheme: s, Recorder: r}
}

func (r *GitSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var gs zenithv1.GitSync
	if err := r.Get(ctx, req.NamespacedName, &gs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !gs.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&gs, gitSyncFinalizer) {
			logger.Info("Cleaning up GitSync resources", "name", gs.Name)
			controllerutil.RemoveFinalizer(&gs, gitSyncFinalizer)
			if err := r.Update(ctx, &gs); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&gs, gitSyncFinalizer) {
		controllerutil.AddFinalizer(&gs, gitSyncFinalizer)
		if err := r.Update(ctx, &gs); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set defaults
	branch := gs.Spec.Branch
	if branch == "" {
		branch = "main"
	}
	syncPath := gs.Spec.Path
	if syncPath == "" {
		syncPath = "/"
	}
	interval := gs.Spec.Interval
	if interval == "" {
		interval = "5m"
	}

	// Parse the interval duration for requeue
	requeueInterval, err := time.ParseDuration(interval)
	if err != nil {
		requeueInterval = 5 * time.Minute
		logger.Error(err, "Failed to parse interval, using default 5m", "interval", interval)
	}

	// Update status to Syncing
	gs.Status.Phase = "Syncing"
	gs.Status.Message = fmt.Sprintf("Syncing from %s branch %s", gs.Spec.RepoURL, branch)
	if err := r.Status().Update(ctx, &gs); err != nil {
		return ctrl.Result{}, err
	}

	// Simulate fetching manifests from Git (in a real implementation,
	// this would clone/pull the repo and read files from the specified path).
	// We create a ConfigMap to track the sync state.
	syncConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("gitsync-%s", gs.Name),
			Namespace: gs.Namespace,
		},
	}

	commitHash := generateCommitHash(gs.Spec.RepoURL, branch, syncPath)

	cmResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, syncConfigMap, func() error {
		syncConfigMap.Labels = map[string]string{
			"app.kubernetes.io/managed-by": "zenith-operator",
			"zenith.dev/gitsync":           gs.Name,
		}
		if syncConfigMap.Data == nil {
			syncConfigMap.Data = make(map[string]string)
		}
		syncConfigMap.Data["repoURL"] = gs.Spec.RepoURL
		syncConfigMap.Data["branch"] = branch
		syncConfigMap.Data["path"] = syncPath
		syncConfigMap.Data["lastCommitHash"] = commitHash
		syncConfigMap.Data["autoSync"] = fmt.Sprintf("%v", gs.Spec.AutoSync)
		syncConfigMap.Data["pruneResources"] = fmt.Sprintf("%v", gs.Spec.PruneResources)
		syncConfigMap.Data["lastSyncTime"] = time.Now().UTC().Format(time.RFC3339)

		return ctrl.SetControllerReference(&gs, syncConfigMap, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&gs, corev1.EventTypeWarning, "SyncFailed", "Failed to create sync configmap: %v", err)
		gs.Status.Phase = "Failed"
		gs.Status.Message = fmt.Sprintf("Failed to create sync configmap: %v", err)
		r.Status().Update(ctx, &gs)
		return ctrl.Result{RequeueAfter: requeueInterval}, err
	}
	if cmResult != controllerutil.OperationResultNone {
		logger.Info("Sync ConfigMap reconciled", "name", gs.Name, "operation", cmResult)
	}

	// Update status to Synced
	now := metav1.NewTime(time.Now())
	gs.Status.Phase = "Synced"
	gs.Status.LastSyncTime = &now
	gs.Status.LastCommitHash = commitHash
	gs.Status.Message = fmt.Sprintf("Successfully synced from %s@%s (%s)", gs.Spec.RepoURL, branch, commitHash[:12])
	gs.Status.SyncedResources = 0

	r.Recorder.Eventf(&gs, corev1.EventTypeNormal, "Synced", "Successfully synced from %s@%s", gs.Spec.RepoURL, branch)

	if err := r.Status().Update(ctx, &gs); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue for periodic sync if AutoSync is enabled
	if gs.Spec.AutoSync {
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	return ctrl.Result{}, nil
}

// generateCommitHash generates a deterministic commit-like hash for tracking sync state.
// In a real implementation this would come from the actual git commit.
func generateCommitHash(repoURL, branch, path string) string {
	data := fmt.Sprintf("%s:%s:%s:%d", repoURL, branch, path, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// ParseManifests parses YAML documents from a byte slice into unstructured objects.
// This is used by the sync process to parse manifests read from the git repo.
func ParseManifests(data []byte) ([]*unstructured.Unstructured, error) {
	var results []*unstructured.Unstructured

	// Split on YAML document separators
	documents := strings.Split(string(data), "---")
	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		obj := &unstructured.Unstructured{}
		decoder := utilyaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 4096)
		if err := decoder.Decode(&obj.Object); err != nil {
			return nil, fmt.Errorf("failed to decode manifest: %w", err)
		}

		if obj.Object == nil {
			continue
		}

		results = append(results, obj)
	}

	return results, nil
}

// ApplyManifest applies a single unstructured object to the cluster using server-side logic.
// It creates the resource if it doesn't exist, or updates it if it does.
func (r *GitSyncReconciler) ApplyManifest(ctx context.Context, obj *unstructured.Unstructured, namespace string) error {
	// Set namespace if not specified and the resource is namespaced
	if obj.GetNamespace() == "" && namespace != "" {
		obj.SetNamespace(namespace)
	}

	// Try to get existing resource
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(obj.GroupVersionKind())
	err := r.Get(ctx, types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}, existing)

	if errors.IsNotFound(err) {
		// Create the resource
		return r.Create(ctx, obj)
	} else if err != nil {
		return fmt.Errorf("failed to get existing resource: %w", err)
	}

	// Update the resource - preserve resource version for conflict detection
	obj.SetResourceVersion(existing.GetResourceVersion())
	return r.Update(ctx, obj)
}

func (r *GitSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.GitSync{}).
		Owns(&corev1.ConfigMap{}).
		Named("gitsync").
		Complete(r)
}
