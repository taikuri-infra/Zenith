package deploy

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// NamespaceResolver maps a project to the Kubernetes namespace its workloads run in.
type NamespaceResolver func(ctx context.Context, projectID string) string

// ProjectNamespaces resolves namespaces from the project record, falling back to
// the legacy shared namespace when a project has none stored.
type ProjectNamespaces struct {
	repo  ports.ProjectRepository
	mu    sync.RWMutex
	cache map[string]string
}

func NewProjectNamespaces(repo ports.ProjectRepository) *ProjectNamespaces {
	return &ProjectNamespaces{repo: repo, cache: map[string]string{}}
}

// For returns the project's namespace.
func (p *ProjectNamespaces) For(ctx context.Context, projectID string) string {
	if projectID == "" {
		return entities.LegacyAppsNamespace
	}
	p.mu.RLock()
	ns, ok := p.cache[projectID]
	p.mu.RUnlock()
	if ok {
		return ns
	}
	proj, err := p.repo.GetProject(ctx, projectID)
	if err != nil || proj.Namespace == "" {
		return entities.LegacyAppsNamespace
	}
	p.mu.Lock()
	p.cache[projectID] = proj.Namespace
	p.mu.Unlock()
	return proj.Namespace
}

// Forget drops a cached entry (after a project is deleted).
func (p *ProjectNamespaces) Forget(projectID string) {
	p.mu.Lock()
	delete(p.cache, projectID)
	p.mu.Unlock()
}

// operatorNamespaces run controllers that must reach managed database pods.
var operatorNamespaces = []string{"kube-system", "cnpg-system", "mariadb-operator", "redis-operator", "zenith-system"}

// EnsureNamespace creates a project namespace with an isolation policy if it is
// missing: pods accept traffic only from their own namespace, the ingress
// controller and the database operators, never from other projects.
func EnsureNamespace(ctx context.Context, c k8sclient.Client, ns string) error {
	if ns == "" || ns == entities.LegacyAppsNamespace {
		return nil // created by the chart
	}
	if c.GetNamespace(ctx, ns) == nil {
		return nil
	}
	if err := c.CreateNamespace(ctx, ns, map[string]string{"zenith.dev/managed": "true"}); err != nil {
		return err
	}
	from := []map[string]any{{"podSelector": map[string]any{}}}
	for _, o := range operatorNamespaces {
		from = append(from, map[string]any{"namespaceSelector": map[string]any{
			"matchLabels": map[string]string{"kubernetes.io/metadata.name": o}}})
	}
	spec, _ := json.Marshal(map[string]any{
		"podSelector": map[string]any{},
		"policyTypes": []string{"Ingress"},
		"ingress":     []map[string]any{{"from": from}},
	})
	if err := c.CreateCRD(ctx, &k8sclient.CRDObject{
		APIVersion: "networking.k8s.io/v1", Kind: "NetworkPolicy",
		Metadata: k8sclient.ObjectMeta{Name: "project-isolation", Namespace: ns},
		Spec:     spec,
	}); err != nil {
		slog.Warn("project isolation policy not created", "namespace", ns, "error", err)
	}
	return nil
}
