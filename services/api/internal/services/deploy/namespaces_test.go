package deploy

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestProjectNamespacesFallsBackToLegacy(t *testing.T) {
	repo := memory.NewMemoryProjectRepository()
	ns := NewProjectNamespaces(repo)
	ctx := context.Background()
	if got := ns.For(ctx, ""); got != entities.LegacyAppsNamespace {
		t.Fatalf("empty project: %q", got)
	}
	if got := ns.For(ctx, "missing"); got != entities.LegacyAppsNamespace {
		t.Fatalf("unknown project: %q", got)
	}
	p, _ := repo.CreateProject(ctx, "u1", "Site", "site", "")
	if got := ns.For(ctx, p.ID); got != p.Namespace || got == entities.LegacyAppsNamespace {
		t.Fatalf("project namespace: %q (want %q)", got, p.Namespace)
	}
}

func TestEnsureNamespaceCreatesOnce(t *testing.T) {
	c := k8sclient.NewMemoryClient()
	ctx := context.Background()
	if err := EnsureNamespace(ctx, c, "zp-aaaa1111-site"); err != nil {
		t.Fatal(err)
	}
	if c.GetNamespace(ctx, "zp-aaaa1111-site") != nil {
		t.Fatal("namespace not created")
	}
	if err := EnsureNamespace(ctx, c, "zp-aaaa1111-site"); err != nil {
		t.Fatalf("second call must be a no-op: %v", err)
	}
	if err := EnsureNamespace(ctx, c, entities.LegacyAppsNamespace); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateK8sResourcesUsesAppNamespace(t *testing.T) {
	app := &entities.App{ID: "a1", Name: "api", Subdomain: "api-x", Port: 8080, Namespace: "zp-aaaa1111-site"}
	res := GenerateK8sResources(app, "img:1", "example.com", nil, nil, entities.PlanFree, nil)
	md := res.Deployment["metadata"].(map[string]interface{})
	if md["namespace"] != "zp-aaaa1111-site" {
		t.Fatalf("namespace %v", md["namespace"])
	}
	app.Namespace = ""
	res = GenerateK8sResources(app, "img:1", "example.com", nil, nil, entities.PlanFree, nil)
	if res.Deployment["metadata"].(map[string]interface{})["namespace"] != entities.LegacyAppsNamespace {
		t.Fatal("empty namespace must fall back to the legacy one")
	}
}
