package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/capiclient"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/gofiber/fiber/v2"
)

func setupAdminApp() (*fiber.App, *handlers.AdminHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	k8sClient := k8sclient.NewMemoryClient()
	capiClient := capiclient.NewClient(k8sClient)
	store := memory.NewMemoryAdminRepository()
	handler := handlers.NewAdminHandler(services.NewAdminService(k8sClient, capiClient, store))
	return app, handler
}

func injectAdmin(c *fiber.Ctx) error {
	c.Locals("user_id", "admin-001")
	c.Locals("email", "admin@zenith.dev")
	c.Locals("name", "Admin")
	c.Locals("role", entities.RoleAdmin)
	return c.Next()
}

// ---------- Dashboard ----------

func TestGetDashboardStats(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/dashboard/stats", handler.GetDashboardStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/dashboard/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var stats entities.DashboardStats
	json.NewDecoder(resp.Body).Decode(&stats)

	if stats.CostProvider != "Hetzner Cloud" {
		t.Errorf("Expected cost provider 'Hetzner Cloud', got '%s'", stats.CostProvider)
	}
	// Modules should have updates available from the seeded data
	if stats.UpdatesAvailable == 0 {
		t.Error("Expected some updates available from seeded modules")
	}
}

// ---------- Clusters ----------

func TestCreateCluster(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)

	body := `{"name":"test-cluster","region":"fsn1","type":"shared","nodes":3,"k8sVersion":"v1.30.2"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(b))
	}

	var cluster entities.Cluster
	json.NewDecoder(resp.Body).Decode(&cluster)

	if cluster.Name != "test-cluster" {
		t.Errorf("Expected name 'test-cluster', got '%s'", cluster.Name)
	}
	if cluster.K8sVersion != "v1.30.2" {
		t.Errorf("Expected k8s version 'v1.30.2', got '%s'", cluster.K8sVersion)
	}
	if cluster.Nodes != 3 {
		t.Errorf("Expected 3 nodes, got %d", cluster.Nodes)
	}
	if cluster.Region != "fsn1" {
		t.Errorf("Expected region 'fsn1', got '%s'", cluster.Region)
	}
	if cluster.Type != "shared" {
		t.Errorf("Expected type 'shared', got '%s'", cluster.Type)
	}
}

func TestCreateClusterMissingName(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)

	body := `{"region":"fsn1","k8sVersion":"v1.30.2"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateClusterMissingRegion(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)

	body := `{"name":"test","k8sVersion":"v1.30.2"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateClusterInvalidType(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)

	body := `{"name":"test","region":"fsn1","k8sVersion":"v1.30.2","type":"invalid"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateClusterDuplicate(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)

	body := `{"name":"dup-cluster","region":"fsn1","type":"shared","nodes":1,"k8sVersion":"v1.30.2"}`

	// Create first
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Test(req)

	// Create duplicate
	req2 := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req2)

	if resp.StatusCode != 409 {
		t.Errorf("Expected 409 for duplicate, got %d", resp.StatusCode)
	}
}

func TestListClusters(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)
	app.Get("/api/v1/admin/clusters", handler.ListClusters)

	// Create 2 clusters
	for _, name := range []string{"cluster-a", "cluster-b"} {
		body := `{"name":"` + name + `","region":"fsn1","type":"shared","nodes":2,"k8sVersion":"v1.30.2"}`
		req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}

	req := httptest.NewRequest("GET", "/api/v1/admin/clusters", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var clusters []entities.Cluster
	json.NewDecoder(resp.Body).Decode(&clusters)

	if len(clusters) != 2 {
		t.Errorf("Expected 2 clusters, got %d", len(clusters))
	}
}

func TestGetCluster(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)
	app.Get("/api/v1/admin/clusters/:name", handler.GetCluster)

	body := `{"name":"get-me","region":"fsn1","type":"shared","nodes":4,"k8sVersion":"v1.30.2"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Test(req)

	getReq := httptest.NewRequest("GET", "/api/v1/admin/clusters/get-me", nil)
	resp, _ := app.Test(getReq)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var cluster entities.Cluster
	json.NewDecoder(resp.Body).Decode(&cluster)

	if cluster.Name != "get-me" {
		t.Errorf("Expected name 'get-me', got '%s'", cluster.Name)
	}
	if cluster.Nodes != 4 {
		t.Errorf("Expected 4 nodes, got %d", cluster.Nodes)
	}
}

func TestGetClusterNotFound(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/clusters/:name", handler.GetCluster)

	req := httptest.NewRequest("GET", "/api/v1/admin/clusters/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteCluster(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)
	app.Delete("/api/v1/admin/clusters/:name", handler.DeleteCluster)
	app.Get("/api/v1/admin/clusters/:name", handler.GetCluster)

	body := `{"name":"to-delete","region":"fsn1","type":"shared","nodes":1,"k8sVersion":"v1.30.2"}`
	createReq := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	app.Test(createReq)

	deleteReq := httptest.NewRequest("DELETE", "/api/v1/admin/clusters/to-delete", nil)
	deleteResp, _ := app.Test(deleteReq)

	if deleteResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", deleteResp.StatusCode)
	}

	// Verify deleted
	getReq := httptest.NewRequest("GET", "/api/v1/admin/clusters/to-delete", nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 404 {
		t.Errorf("Expected 404 after deletion, got %d", getResp.StatusCode)
	}
}

func TestDeleteClusterNotFound(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Delete("/api/v1/admin/clusters/:name", handler.DeleteCluster)

	req := httptest.NewRequest("DELETE", "/api/v1/admin/clusters/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpgradeCluster(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)
	app.Post("/api/v1/admin/clusters/:name/upgrade", handler.UpgradeCluster)
	app.Get("/api/v1/admin/clusters/:name", handler.GetCluster)

	// Create cluster
	createBody := `{"name":"upgrade-me","region":"fsn1","type":"shared","nodes":2,"k8sVersion":"v1.28.0"}`
	createReq := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	app.Test(createReq)

	// Upgrade
	upgradeBody := `{"version":"v1.30.2"}`
	upgradeReq := httptest.NewRequest("POST", "/api/v1/admin/clusters/upgrade-me/upgrade", bytes.NewBufferString(upgradeBody))
	upgradeReq.Header.Set("Content-Type", "application/json")
	upgradeResp, _ := app.Test(upgradeReq)

	if upgradeResp.StatusCode != 200 {
		b, _ := io.ReadAll(upgradeResp.Body)
		t.Fatalf("Expected 200, got %d: %s", upgradeResp.StatusCode, string(b))
	}

	// Verify version updated
	getReq := httptest.NewRequest("GET", "/api/v1/admin/clusters/upgrade-me", nil)
	getResp, _ := app.Test(getReq)
	defer getResp.Body.Close()

	var cluster entities.Cluster
	json.NewDecoder(getResp.Body).Decode(&cluster)

	if cluster.K8sVersion != "v1.30.2" {
		t.Errorf("Expected k8s version 'v1.30.2' after upgrade, got '%s'", cluster.K8sVersion)
	}
}

func TestUpgradeClusterNotFound(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters/:name/upgrade", handler.UpgradeCluster)

	body := `{"version":"v1.30.2"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters/nonexistent/upgrade", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpgradeClusterMissingVersion(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters/:name/upgrade", handler.UpgradeCluster)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters/some-cluster/upgrade", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// ---------- Tenants ----------

func TestListTenantsEmpty(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/tenants", handler.ListTenants)

	req := httptest.NewRequest("GET", "/api/v1/admin/tenants", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var tenants []entities.Tenant
	json.NewDecoder(resp.Body).Decode(&tenants)

	if len(tenants) != 0 {
		t.Errorf("Expected 0 tenants, got %d", len(tenants))
	}
}

func TestSuspendTenantNotFound(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/tenants/:id/suspend", handler.SuspendTenant)

	req := httptest.NewRequest("POST", "/api/v1/admin/tenants/nonexistent/suspend", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

// ---------- Modules ----------

func TestListModules(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/modules", handler.ListModules)

	req := httptest.NewRequest("GET", "/api/v1/admin/modules", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var modules []entities.Module
	json.NewDecoder(resp.Body).Decode(&modules)

	if len(modules) == 0 {
		t.Error("Expected non-empty module list from seeded data")
	}

	// Verify a known module is present
	found := false
	for _, m := range modules {
		if m.Name == "Zenith Operator" {
			found = true
			if m.Status != "update_available" {
				t.Errorf("Expected Zenith Operator status 'update_available', got '%s'", m.Status)
			}
			break
		}
	}
	if !found {
		t.Error("Expected to find 'Zenith Operator' in modules list")
	}
}

func TestUpdateModule(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/modules/:name/update", handler.UpdateModule)

	req := httptest.NewRequest("POST", "/api/v1/admin/modules/CloudNativePG/update", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var mod entities.Module
	json.NewDecoder(resp.Body).Decode(&mod)

	if mod.Status != "up_to_date" {
		t.Errorf("Expected status 'up_to_date' after update, got '%s'", mod.Status)
	}
	if mod.Installed != mod.Latest {
		t.Errorf("Expected installed == latest after update, got installed=%s latest=%s", mod.Installed, mod.Latest)
	}
}

func TestUpdateModuleNotFound(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/modules/:name/update", handler.UpdateModule)

	req := httptest.NewRequest("POST", "/api/v1/admin/modules/NonexistentModule/update", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateAllModules(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/modules/update-all", handler.UpdateAllModules)
	app.Get("/api/v1/admin/modules", handler.ListModules)

	// Update all
	updateReq := httptest.NewRequest("POST", "/api/v1/admin/modules/update-all", nil)
	updateResp, _ := app.Test(updateReq)

	if updateResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", updateResp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(updateResp.Body).Decode(&result)

	count, _ := result["count"].(float64)
	if count == 0 {
		t.Error("Expected at least one module to be updated")
	}

	// Verify all are up to date now
	listReq := httptest.NewRequest("GET", "/api/v1/admin/modules", nil)
	listResp, _ := app.Test(listReq)
	defer listResp.Body.Close()

	var modules []entities.Module
	json.NewDecoder(listResp.Body).Decode(&modules)

	for _, m := range modules {
		if m.Status != "up_to_date" {
			t.Errorf("Expected module '%s' to be up_to_date after update-all, got '%s'", m.Name, m.Status)
		}
	}
}

func TestInstallModule(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/modules/:name/install", handler.InstallModule)

	req := httptest.NewRequest("POST", "/api/v1/admin/modules/NewModule/install", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestUninstallModule(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/modules/:name/uninstall", handler.UninstallModule)

	req := httptest.NewRequest("POST", "/api/v1/admin/modules/SomeModule/uninstall", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

// ---------- Audit Log ----------

func TestListAuditLog(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/audit", handler.ListAuditLog)

	req := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var entries []entities.AuditEntry
	json.NewDecoder(resp.Body).Decode(&entries)

	if len(entries) == 0 {
		t.Error("Expected non-empty audit log from seeded data")
	}
}

func TestListAuditLogWithLimit(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/audit", handler.ListAuditLog)

	req := httptest.NewRequest("GET", "/api/v1/admin/audit?limit=2", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var entries []entities.AuditEntry
	json.NewDecoder(resp.Body).Decode(&entries)

	if len(entries) > 2 {
		t.Errorf("Expected at most 2 entries with limit=2, got %d", len(entries))
	}
}

// ---------- Platform Updates ----------

func TestCheckUpdates(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/updates/check", handler.CheckUpdates)

	req := httptest.NewRequest("GET", "/api/v1/admin/updates/check", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var update entities.PlatformUpdate
	json.NewDecoder(resp.Body).Decode(&update)

	if update.Version == "" {
		t.Error("Expected non-empty version")
	}
	if update.Current == "" {
		t.Error("Expected non-empty current version")
	}
	if len(update.Features) == 0 {
		t.Error("Expected non-empty features list")
	}
}

func TestApplyUpdate(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/updates/apply", handler.ApplyUpdate)

	body := `{"version":"v1.3.0"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/updates/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestApplyUpdateMissingVersion(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/updates/apply", handler.ApplyUpdate)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/admin/updates/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestListUpdateHistory(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/updates/history", handler.ListUpdateHistory)

	req := httptest.NewRequest("GET", "/api/v1/admin/updates/history", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var history []entities.UpdateHistoryEntry
	json.NewDecoder(resp.Body).Decode(&history)

	if len(history) == 0 {
		t.Error("Expected non-empty update history")
	}
}

// ---------- Infrastructure ----------

func TestGetInfraOverview(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)
	app.Get("/api/v1/admin/infrastructure", handler.GetInfraOverview)

	// Create a cluster first
	body := `{"name":"infra-test","region":"fsn1","type":"shared","nodes":3,"k8sVersion":"v1.30.2"}`
	createReq := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	app.Test(createReq)

	req := httptest.NewRequest("GET", "/api/v1/admin/infrastructure", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var infra entities.InfraOverview
	json.NewDecoder(resp.Body).Decode(&infra)

	// 3 nodes + 1 management = 4
	if infra.Servers != 4 {
		t.Errorf("Expected 4 servers (3 + management), got %d", infra.Servers)
	}
	if len(infra.Resources) == 0 {
		t.Error("Expected non-empty resources list")
	}
}

func TestGetInfraOverviewEmpty(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/infrastructure", handler.GetInfraOverview)

	req := httptest.NewRequest("GET", "/api/v1/admin/infrastructure", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var infra entities.InfraOverview
	json.NewDecoder(resp.Body).Decode(&infra)

	// Only management plane server
	if infra.Servers != 1 {
		t.Errorf("Expected 1 server (management only), got %d", infra.Servers)
	}
}

// ---------- Platform State ----------

func TestGetPlatformState(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/state", handler.GetPlatformState)

	req := httptest.NewRequest("GET", "/api/v1/admin/state", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var state entities.PlatformState
	json.NewDecoder(resp.Body).Decode(&state)

	if state.PlatformVersion == "" {
		t.Error("Expected non-empty platform version")
	}
	if state.Domain == "" {
		t.Error("Expected non-empty domain")
	}
	if state.ManagementK8sVersion == "" {
		t.Error("Expected non-empty management k8s version")
	}
}

func TestExportState(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/state/export", handler.ExportState)

	req := httptest.NewRequest("GET", "/api/v1/admin/state/export", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	// Should be valid JSON
	var export map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &export); err != nil {
		t.Fatalf("Expected valid JSON export, got error: %v", err)
	}

	// Check required keys
	for _, key := range []string{"exportedAt", "platform", "clusters", "tenants", "modules"} {
		if _, ok := export[key]; !ok {
			t.Errorf("Expected key '%s' in export", key)
		}
	}

	// Check Content-Disposition header
	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		t.Error("Expected Content-Disposition header for file download")
	}
}

// ---------- Settings ----------

func TestGetSettings(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/settings", handler.GetSettings)

	req := httptest.NewRequest("GET", "/api/v1/admin/settings", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var settings entities.PlatformSettings
	json.NewDecoder(resp.Body).Decode(&settings)

	if settings.PlatformName != "Zenith" {
		t.Errorf("Expected platform name 'Zenith', got '%s'", settings.PlatformName)
	}
	if settings.Provider != "Hetzner Cloud" {
		t.Errorf("Expected provider 'Hetzner Cloud', got '%s'", settings.Provider)
	}
	if settings.BaseDomain != "freezenith.com" {
		t.Errorf("Expected base domain 'freezenith.com', got '%s'", settings.BaseDomain)
	}
	if !settings.AutoBackups {
		t.Error("Expected auto backups to be enabled by default")
	}
}

func TestUpdateSettingsPATCH(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Patch("/api/v1/admin/settings", handler.UpdateSettings)

	body := `{"platformName":"My Zenith","baseDomain":"example.com"}`
	req := httptest.NewRequest("PATCH", "/api/v1/admin/settings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var updated entities.PlatformSettings
	json.NewDecoder(resp.Body).Decode(&updated)

	if updated.PlatformName != "My Zenith" {
		t.Errorf("Expected platform name 'My Zenith', got '%s'", updated.PlatformName)
	}
	if updated.BaseDomain != "example.com" {
		t.Errorf("Expected base domain 'example.com', got '%s'", updated.BaseDomain)
	}
	// Provider should remain unchanged
	if updated.Provider != "Hetzner Cloud" {
		t.Errorf("Expected provider to remain 'Hetzner Cloud', got '%s'", updated.Provider)
	}
}

func TestUpdateSettingsPUT(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Put("/api/v1/admin/settings", handler.UpdateSettings)

	body := `{"platformName":"Updated Zenith"}`
	req := httptest.NewRequest("PUT", "/api/v1/admin/settings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var updated entities.PlatformSettings
	json.NewDecoder(resp.Body).Decode(&updated)

	if updated.PlatformName != "Updated Zenith" {
		t.Errorf("Expected platform name 'Updated Zenith', got '%s'", updated.PlatformName)
	}
}

// ---------- Integration-like: Audit log fills on actions ----------

func TestAuditLogPopulatedByActions(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)
	app.Delete("/api/v1/admin/clusters/:name", handler.DeleteCluster)
	app.Get("/api/v1/admin/audit", handler.ListAuditLog)

	// Get initial audit count
	initReq := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	initResp, _ := app.Test(initReq)
	var initEntries []entities.AuditEntry
	json.NewDecoder(initResp.Body).Decode(&initEntries)
	initialCount := len(initEntries)

	// Create a cluster (should add audit entry)
	body := `{"name":"audit-test","region":"fsn1","type":"shared","nodes":1,"k8sVersion":"v1.30.2"}`
	createReq := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	app.Test(createReq)

	// Delete the cluster (should add another audit entry)
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/admin/clusters/audit-test", nil)
	app.Test(deleteReq)

	// Check audit log grew
	auditReq := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	auditResp, _ := app.Test(auditReq)
	defer auditResp.Body.Close()

	var entries []entities.AuditEntry
	json.NewDecoder(auditResp.Body).Decode(&entries)

	expectedCount := initialCount + 2
	if len(entries) != expectedCount {
		t.Errorf("Expected %d audit entries after create+delete, got %d", expectedCount, len(entries))
	}
}

// ---------- Additional Admin Edge Case Tests ----------

func TestListAuditLogEmptyStore(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	k8sClient := k8sclient.NewMemoryClient()
	capiClient := capiclient.NewClient(k8sClient)
	// Create a store but clear the audit log manually
	store := memory.NewMemoryAdminRepository()
	handler := handlers.NewAdminHandler(services.NewAdminService(k8sClient, capiClient, store))
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/audit", handler.ListAuditLog)

	// Request with large offset to get empty results
	req := httptest.NewRequest("GET", "/api/v1/admin/audit?offset=1000", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var entries []entities.AuditEntry
	json.NewDecoder(resp.Body).Decode(&entries)

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with large offset, got %d", len(entries))
	}
}

func TestListAuditLogWithOffset(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/audit", handler.ListAuditLog)

	// offset=2 should skip first 2 seeded entries
	req := httptest.NewRequest("GET", "/api/v1/admin/audit?offset=2&limit=1", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var entries []entities.AuditEntry
	json.NewDecoder(resp.Body).Decode(&entries)

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry with offset=2 limit=1, got %d", len(entries))
	}
}

func TestUpdateSettingsPartialUpdate(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Patch("/api/v1/admin/settings", handler.UpdateSettings)
	app.Get("/api/v1/admin/settings", handler.GetSettings)

	// Only update baseDomain, everything else should remain
	body := `{"baseDomain":"newdomain.dev"}`
	req := httptest.NewRequest("PATCH", "/api/v1/admin/settings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var updated entities.PlatformSettings
	json.NewDecoder(resp.Body).Decode(&updated)

	if updated.BaseDomain != "newdomain.dev" {
		t.Errorf("Expected baseDomain 'newdomain.dev', got '%s'", updated.BaseDomain)
	}
	// PlatformName should remain "Zenith" (unchanged)
	if updated.PlatformName != "Zenith" {
		t.Errorf("Expected platformName 'Zenith' (unchanged), got '%s'", updated.PlatformName)
	}
	// Provider should remain "Hetzner Cloud" (unchanged)
	if updated.Provider != "Hetzner Cloud" {
		t.Errorf("Expected provider 'Hetzner Cloud' (unchanged), got '%s'", updated.Provider)
	}
}

func TestUpdateSettingsInvalidBody(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Patch("/api/v1/admin/settings", handler.UpdateSettings)

	req := httptest.NewRequest("PATCH", "/api/v1/admin/settings", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateClusterMissingK8sVersion(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)

	body := `{"name":"test","region":"fsn1"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for missing k8sVersion, got %d", resp.StatusCode)
	}
}

func TestCreateClusterInvalidBody(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)

	req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateClusterDefaultNodes(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)
	app.Get("/api/v1/admin/clusters/:name", handler.GetCluster)

	// Omit nodes - should default to 1
	body := `{"name":"default-nodes","region":"fsn1","k8sVersion":"v1.30.2"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var cluster entities.Cluster
	json.NewDecoder(resp.Body).Decode(&cluster)

	if cluster.Nodes != 1 {
		t.Errorf("Expected default nodes 1, got %d", cluster.Nodes)
	}
}

func TestCreateClusterDefaultType(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters", handler.CreateCluster)

	// Omit type - should default to "shared"
	body := `{"name":"default-type","region":"fsn1","k8sVersion":"v1.30.2","nodes":2}`
	req := httptest.NewRequest("POST", "/api/v1/admin/clusters", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var cluster entities.Cluster
	json.NewDecoder(resp.Body).Decode(&cluster)

	if cluster.Type != "shared" {
		t.Errorf("Expected default type 'shared', got '%s'", cluster.Type)
	}
}

func TestUpgradeClusterInvalidBody(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/clusters/:name/upgrade", handler.UpgradeCluster)

	req := httptest.NewRequest("POST", "/api/v1/admin/clusters/some-cluster/upgrade", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestGetTenantNotFound(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/tenants/:id", handler.GetTenant)

	req := httptest.NewRequest("GET", "/api/v1/admin/tenants/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestSuspendTenantSuccess(t *testing.T) {
	// We need to use the k8sClient directly to create a project
	k8sClient := k8sclient.NewMemoryClient()
	capiClient := capiclient.NewClient(k8sClient)
	store := memory.NewMemoryAdminRepository()
	h := handlers.NewAdminHandler(services.NewAdminService(k8sClient, capiClient, store))

	fiberApp := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	fiberApp.Use(injectAdmin)
	fiberApp.Post("/api/v1/admin/tenants/:id/suspend", h.SuspendTenant)
	fiberApp.Get("/api/v1/admin/tenants/:id", h.GetTenant)

	// Create a project CRD
	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "Test Tenant",
		"plan":        "pro",
	})
	k8sClient.CreateCRD(nil, &k8sclient.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata:   k8sclient.ObjectMeta{Name: "test-tenant"},
		Spec:       projSpec,
	})

	// Suspend
	suspendReq := httptest.NewRequest("POST", "/api/v1/admin/tenants/test-tenant/suspend", nil)
	suspendResp, _ := fiberApp.Test(suspendReq)

	if suspendResp.StatusCode != 200 {
		b, _ := io.ReadAll(suspendResp.Body)
		t.Fatalf("Expected 200, got %d: %s", suspendResp.StatusCode, string(b))
	}

	var result map[string]interface{}
	json.NewDecoder(suspendResp.Body).Decode(&result)

	if result["message"] != "tenant suspended" {
		t.Errorf("Expected message 'tenant suspended', got '%v'", result["message"])
	}

	// Verify tenant is now suspended
	getReq := httptest.NewRequest("GET", "/api/v1/admin/tenants/test-tenant", nil)
	getResp, _ := fiberApp.Test(getReq)

	var tenant entities.Tenant
	json.NewDecoder(getResp.Body).Decode(&tenant)

	if tenant.Status != "suspended" {
		t.Errorf("Expected tenant status 'suspended', got '%s'", tenant.Status)
	}
}

func TestListTenantsWithProjects(t *testing.T) {
	k8sClient := k8sclient.NewMemoryClient()
	capiClient := capiclient.NewClient(k8sClient)
	store := memory.NewMemoryAdminRepository()
	handler := handlers.NewAdminHandler(services.NewAdminService(k8sClient, capiClient, store))

	fiberApp := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	fiberApp.Use(injectAdmin)
	fiberApp.Get("/api/v1/admin/tenants", handler.ListTenants)

	// Create project CRDs
	for _, name := range []string{"tenant-a", "tenant-b"} {
		projSpec, _ := json.Marshal(map[string]interface{}{
			"displayName": name,
			"plan":        "free",
		})
		k8sClient.CreateCRD(nil, &k8sclient.CRDObject{
			APIVersion: "zenith.dev/v1alpha1",
			Kind:       "Project",
			Metadata:   k8sclient.ObjectMeta{Name: name},
			Spec:       projSpec,
		})
	}

	req := httptest.NewRequest("GET", "/api/v1/admin/tenants", nil)
	resp, _ := fiberApp.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var tenants []entities.Tenant
	json.NewDecoder(resp.Body).Decode(&tenants)

	if len(tenants) != 2 {
		t.Errorf("Expected 2 tenants, got %d", len(tenants))
	}
}

func TestApplyUpdateInvalidBody(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/updates/apply", handler.ApplyUpdate)

	req := httptest.NewRequest("POST", "/api/v1/admin/updates/apply", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestDashboardStatsWithClusters(t *testing.T) {
	k8sClient := k8sclient.NewMemoryClient()
	capiClient := capiclient.NewClient(k8sClient)
	store := memory.NewMemoryAdminRepository()
	handler := handlers.NewAdminHandler(services.NewAdminService(k8sClient, capiClient, store))

	fiberApp := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	fiberApp.Use(injectAdmin)
	fiberApp.Get("/api/v1/admin/dashboard/stats", handler.GetDashboardStats)

	// Create a cluster and project to get real stats
	capiClient.CreateCluster(nil, dto.CreateClusterInput{
		Name:       "test",
		Region:     "fsn1",
		K8sVersion: "v1.30.2",
		Nodes:      3,
		Type:       "shared",
	})

	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "Test",
	})
	k8sClient.CreateCRD(nil, &k8sclient.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata:   k8sclient.ObjectMeta{Name: "proj1"},
		Spec:       projSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/dashboard/stats", nil)
	resp, _ := fiberApp.Test(req)
	defer resp.Body.Close()

	var stats entities.DashboardStats
	json.NewDecoder(resp.Body).Decode(&stats)

	if stats.ClusterCount != 1 {
		t.Errorf("Expected 1 cluster, got %d", stats.ClusterCount)
	}
	if stats.TenantCount != 1 {
		t.Errorf("Expected 1 tenant, got %d", stats.TenantCount)
	}
	if stats.ActiveToday != 1 {
		t.Errorf("Expected 1 active today, got %d", stats.ActiveToday)
	}
	if !stats.AllHealthy {
		t.Error("Expected all healthy to be true")
	}
}

func TestDashboardStatsWithSuspendedTenant(t *testing.T) {
	k8sClient := k8sclient.NewMemoryClient()
	capiClient := capiclient.NewClient(k8sClient)
	store := memory.NewMemoryAdminRepository()
	handler := handlers.NewAdminHandler(services.NewAdminService(k8sClient, capiClient, store))

	fiberApp := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	fiberApp.Use(injectAdmin)
	fiberApp.Get("/api/v1/admin/dashboard/stats", handler.GetDashboardStats)

	// Create a suspended project
	projSpec, _ := json.Marshal(map[string]interface{}{
		"displayName": "Suspended",
	})
	k8sClient.CreateCRD(nil, &k8sclient.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata: k8sclient.ObjectMeta{
			Name:        "suspended-proj",
			Annotations: map[string]string{"zenith.dev/suspended": "true"},
		},
		Spec: projSpec,
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/dashboard/stats", nil)
	resp, _ := fiberApp.Test(req)

	var stats entities.DashboardStats
	json.NewDecoder(resp.Body).Decode(&stats)

	if stats.TenantCount != 1 {
		t.Errorf("Expected 1 tenant, got %d", stats.TenantCount)
	}
	if stats.ActiveToday != 0 {
		t.Errorf("Expected 0 active today (suspended), got %d", stats.ActiveToday)
	}
}

func TestGetPlatformStateUpdateAvailable(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/state", handler.GetPlatformState)

	req := httptest.NewRequest("GET", "/api/v1/admin/state", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	var state entities.PlatformState
	json.NewDecoder(resp.Body).Decode(&state)

	// Store has current=v1.2.1 and version=v1.3.0, so update is available
	if state.UpdateAvailable != "v1.3.0" {
		t.Errorf("Expected update available 'v1.3.0', got '%s'", state.UpdateAvailable)
	}
	if state.PlatformVersion != "v1.2.1" {
		t.Errorf("Expected platform version 'v1.2.1', got '%s'", state.PlatformVersion)
	}
	if !state.WildcardTLS {
		t.Error("Expected wildcard TLS to be true")
	}
}

func TestExportStateContentDisposition(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Get("/api/v1/admin/state/export", handler.ExportState)

	req := httptest.NewRequest("GET", "/api/v1/admin/state/export", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	cd := resp.Header.Get("Content-Disposition")
	if cd != "attachment; filename=zenith-state.json" {
		t.Errorf("Expected specific Content-Disposition, got '%s'", cd)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", ct)
	}
}

func TestActorFromContextFallback(t *testing.T) {
	app, handler := setupAdminApp()
	// Use middleware with no email/name to test fallback to "admin"
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", entities.RoleAdmin)
		return c.Next()
	})
	app.Post("/api/v1/admin/modules/:name/install", handler.InstallModule)
	app.Get("/api/v1/admin/audit", handler.ListAuditLog)

	installReq := httptest.NewRequest("POST", "/api/v1/admin/modules/TestModule/install", nil)
	app.Test(installReq)

	auditReq := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	auditResp, _ := app.Test(auditReq)
	defer auditResp.Body.Close()

	var entries []entities.AuditEntry
	json.NewDecoder(auditResp.Body).Decode(&entries)

	// The newest entry should have actor "admin" (fallback)
	if len(entries) > 0 && entries[0].Actor != "admin" {
		t.Errorf("Expected actor 'admin' (fallback), got '%s'", entries[0].Actor)
	}
}

func TestUninstallModuleAuditEntry(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/modules/:name/uninstall", handler.UninstallModule)
	app.Get("/api/v1/admin/audit", handler.ListAuditLog)

	// Get initial count
	initReq := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	initResp, _ := app.Test(initReq)
	var initEntries []entities.AuditEntry
	json.NewDecoder(initResp.Body).Decode(&initEntries)
	initialCount := len(initEntries)

	// Uninstall a module
	uninstallReq := httptest.NewRequest("POST", "/api/v1/admin/modules/SomeModule/uninstall", nil)
	uninstallResp, _ := app.Test(uninstallReq)

	if uninstallResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", uninstallResp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(uninstallResp.Body).Decode(&result)

	if result["message"] != "module uninstallation initiated" {
		t.Errorf("Expected uninstall message, got '%v'", result["message"])
	}

	// Check audit log grew by 1
	auditReq := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	auditResp, _ := app.Test(auditReq)
	var entries []entities.AuditEntry
	json.NewDecoder(auditResp.Body).Decode(&entries)

	if len(entries) != initialCount+1 {
		t.Errorf("Expected %d audit entries, got %d", initialCount+1, len(entries))
	}
}

func TestApplyUpdateAuditEntry(t *testing.T) {
	app, handler := setupAdminApp()
	app.Use(injectAdmin)
	app.Post("/api/v1/admin/updates/apply", handler.ApplyUpdate)
	app.Get("/api/v1/admin/audit", handler.ListAuditLog)

	body := `{"version":"v2.0.0"}`
	applyReq := httptest.NewRequest("POST", "/api/v1/admin/updates/apply", bytes.NewBufferString(body))
	applyReq.Header.Set("Content-Type", "application/json")
	applyResp, _ := app.Test(applyReq)

	if applyResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", applyResp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(applyResp.Body).Decode(&result)

	if result["version"] != "v2.0.0" {
		t.Errorf("Expected version 'v2.0.0' in response, got '%v'", result["version"])
	}
}
