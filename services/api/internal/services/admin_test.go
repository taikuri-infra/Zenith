package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func newTestAdminService() (*AdminService, *k8sclient.MemoryClient, *memory.MemoryAdminRepository) {
	k8s := k8sclient.NewMemoryClient()
	store := memory.NewMemoryAdminRepository()
	svc := NewAdminService(k8s, nil, store)
	return svc, k8s, store
}

// --- ListClusters tests ---

func TestAdminListClusters_NilProvisioner(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	clusters, err := svc.ListClusters(ctx)
	if err != nil {
		t.Fatalf("ListClusters failed: %v", err)
	}
	if len(clusters) != 0 {
		t.Errorf("Expected 0 clusters with nil provisioner, got %d", len(clusters))
	}
}

// --- GetCluster tests ---

func TestAdminGetCluster_NilProvisioner(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	_, err := svc.GetCluster(ctx, "test-cluster")
	if err == nil {
		t.Error("Expected error when cluster provisioner is nil")
	}
}

// --- CreateCluster tests ---

func TestAdminCreateCluster_NilProvisioner(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	_, err := svc.CreateCluster(ctx, dto.CreateClusterInput{Name: "test", Region: "fsn1", Nodes: 3}, "admin")
	if err == nil {
		t.Error("Expected error when cluster provisioner is nil")
	}
}

// --- DeleteCluster tests ---

func TestAdminDeleteCluster_NilProvisioner(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	err := svc.DeleteCluster(ctx, "test-cluster", "admin")
	if err == nil {
		t.Error("Expected error when cluster provisioner is nil")
	}
}

// --- UpgradeCluster tests ---

func TestAdminUpgradeCluster_NilProvisioner(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	err := svc.UpgradeCluster(ctx, "test-cluster", "v1.32.0", "admin")
	if err == nil {
		t.Error("Expected error when cluster provisioner is nil")
	}
}

// --- ListModules tests ---

func TestAdminListModules(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	modules, err := svc.ListModules(ctx)
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	if len(modules) == 0 {
		t.Error("Expected seeded modules")
	}
}

// --- InstallModule tests ---

func TestAdminInstallModule(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	err := svc.InstallModule(ctx, "test-module", "admin")
	if err != nil {
		t.Fatalf("InstallModule failed: %v", err)
	}
}

// --- UninstallModule tests ---

func TestAdminUninstallModule(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	err := svc.UninstallModule(ctx, "test-module", "admin")
	if err != nil {
		t.Fatalf("UninstallModule failed: %v", err)
	}
}

// --- UpdateModule tests ---

func TestAdminUpdateModule_Exists(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	// "Zenith Operator" is seeded with update_available status
	mod, err := svc.UpdateModule(ctx, "Zenith Operator", "admin")
	if err != nil {
		t.Fatalf("UpdateModule failed: %v", err)
	}
	if mod.Status != "up_to_date" {
		t.Errorf("Expected status 'up_to_date' after update, got '%s'", mod.Status)
	}
}

func TestAdminUpdateModule_NotFound(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	_, err := svc.UpdateModule(ctx, "nonexistent-module", "admin")
	if err == nil {
		t.Error("Expected error for nonexistent module")
	}
}

// --- UpdateAllModules tests ---

func TestAdminUpdateAllModules(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	count, err := svc.UpdateAllModules(ctx, "admin")
	if err != nil {
		t.Fatalf("UpdateAllModules failed: %v", err)
	}
	// Memory repo has modules with "update_available" status
	if count == 0 {
		t.Error("Expected at least some modules to be updated")
	}
}

// --- ListAuditLog tests ---

func TestAdminListAuditLog(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	entries, err := svc.ListAuditLog(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListAuditLog failed: %v", err)
	}
	// Memory repo is pre-seeded with audit entries
	if len(entries) == 0 {
		t.Error("Expected seeded audit entries")
	}
}

// --- CheckUpdates tests ---

func TestAdminCheckUpdates(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	update, err := svc.CheckUpdates(ctx)
	if err != nil {
		t.Fatalf("CheckUpdates failed: %v", err)
	}
	if update.Version == "" {
		t.Error("Expected non-empty update version")
	}
}

// --- ApplyUpdate tests ---

func TestAdminApplyUpdate(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	err := svc.ApplyUpdate(ctx, "v1.3.0", "admin")
	if err != nil {
		t.Fatalf("ApplyUpdate failed: %v", err)
	}
}

// --- ListUpdateHistory tests ---

func TestAdminListUpdateHistory(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	history, err := svc.ListUpdateHistory(ctx)
	if err != nil {
		t.Fatalf("ListUpdateHistory failed: %v", err)
	}
	if len(history) == 0 {
		t.Error("Expected seeded update history")
	}
}

// --- GetInfraOverview tests ---

func TestAdminGetInfraOverview_NilClusters(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	overview, err := svc.GetInfraOverview(ctx)
	if err != nil {
		t.Fatalf("GetInfraOverview failed: %v", err)
	}
	// With nil cluster provisioner, only management plane counted
	if overview.Servers != 1 {
		t.Errorf("Expected 1 server (management), got %d", overview.Servers)
	}
}

// --- GetPlatformState tests ---

func TestAdminGetPlatformState(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	state, err := svc.GetPlatformState(ctx)
	if err != nil {
		t.Fatalf("GetPlatformState failed: %v", err)
	}
	if state.PlatformVersion == "" {
		t.Error("Expected non-empty platform version")
	}
	if state.Domain == "" {
		t.Error("Expected non-empty domain")
	}
}

// --- ExportState tests ---

func TestAdminExportState(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	data, err := svc.ExportState(ctx)
	if err != nil {
		t.Fatalf("ExportState failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty export data")
	}
	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Expected valid JSON, got: %v", err)
	}
	if result["exportedAt"] == nil {
		t.Error("Expected 'exportedAt' in export")
	}
}

// --- GetSettings tests ---

func TestAdminGetSettings(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	settings, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings.PlatformName != "Zenith" {
		t.Errorf("Expected platform name 'Zenith', got '%s'", settings.PlatformName)
	}
}

// --- UpdateSettings tests ---

func TestAdminUpdateSettings(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	updated, err := svc.UpdateSettings(ctx, &entities.PlatformSettings{
		PlatformName: "Zenith Dev",
	}, "admin")
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}
	if updated.PlatformName != "Zenith Dev" {
		t.Errorf("Expected 'Zenith Dev', got '%s'", updated.PlatformName)
	}
}

// --- GetDashboardStats tests ---

func TestAdminGetDashboardStats(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	stats, err := svc.GetDashboardStats(ctx)
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}
	// With nil cluster provisioner, cluster count should be 0
	if stats.ClusterCount != 0 {
		t.Errorf("Expected 0 clusters, got %d", stats.ClusterCount)
	}
	// AllHealthy should be true when no clusters
	if !stats.AllHealthy {
		t.Error("Expected AllHealthy=true when no clusters")
	}
}

// --- ListTenants tests ---

func TestAdminListTenants_Empty(t *testing.T) {
	svc, _, _ := newTestAdminService()
	ctx := context.Background()

	tenants, err := svc.ListTenants(ctx)
	if err != nil {
		t.Fatalf("ListTenants failed: %v", err)
	}
	if len(tenants) != 0 {
		t.Errorf("Expected 0 tenants, got %d", len(tenants))
	}
}

// --- projectToTenant tests ---

func TestProjectToTenant_Basic(t *testing.T) {
	spec, _ := json.Marshal(map[string]interface{}{
		"displayName": "My Project",
		"plan":        "pro",
	})

	p := &k8sclient.CRDObject{
		Metadata: k8sclient.ObjectMeta{
			Name:      "my-project",
			Namespace: "zenith-apps",
		},
		Spec: spec,
	}

	tenant := projectToTenant(p)
	if tenant.Name != "My Project" {
		t.Errorf("Expected name 'My Project', got '%s'", tenant.Name)
	}
	if tenant.Plan != "pro" {
		t.Errorf("Expected plan 'pro', got '%s'", tenant.Plan)
	}
	if tenant.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", tenant.Status)
	}
}

func TestProjectToTenant_Suspended(t *testing.T) {
	spec, _ := json.Marshal(map[string]interface{}{})

	p := &k8sclient.CRDObject{
		Metadata: k8sclient.ObjectMeta{
			Name: "suspended-project",
			Annotations: map[string]string{
				"zenith.dev/suspended": "true",
			},
		},
		Spec: spec,
	}

	tenant := projectToTenant(p)
	if tenant.Status != "suspended" {
		t.Errorf("Expected status 'suspended', got '%s'", tenant.Status)
	}
}

func TestProjectToTenant_DefaultValues(t *testing.T) {
	spec, _ := json.Marshal(map[string]interface{}{})

	p := &k8sclient.CRDObject{
		Metadata: k8sclient.ObjectMeta{
			Name: "fallback-project",
		},
		Spec: spec,
	}

	tenant := projectToTenant(p)
	// When displayName is empty, falls back to metadata name
	if tenant.Name != "fallback-project" {
		t.Errorf("Expected name 'fallback-project', got '%s'", tenant.Name)
	}
	// When plan is empty, falls back to "starter"
	if tenant.Plan != "starter" {
		t.Errorf("Expected plan 'starter', got '%s'", tenant.Plan)
	}
}
