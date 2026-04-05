package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func newTestGatewayService() (*GatewayService, *memory.MemoryGatewayRepository, *memory.MemoryAppRepository) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	return svc, gwRepo, appRepo
}

// --- ReconcileAll tests ---

func TestReconcileAll(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()
	// Should not panic
	svc.ReconcileAll(ctx)
}

// --- SyncGateway tests ---

func TestSyncGateway_Success(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Sync GW")
	err := svc.SyncGateway(ctx, gw.ID)
	if err != nil {
		t.Fatalf("SyncGateway failed: %v", err)
	}
}

func TestSyncGateway_NotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	err := svc.SyncGateway(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

// --- EnsureProjectGateway tests ---

func TestEnsureProjectGateway_CreatesNew(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, err := svc.EnsureProjectGateway(ctx, "user-1", "proj-12345678", "my-project")
	if err != nil {
		t.Fatalf("EnsureProjectGateway failed: %v", err)
	}
	if gw == nil {
		t.Fatal("Expected non-nil gateway")
	}
	if gw.Endpoint == "" {
		t.Error("Expected non-empty endpoint")
	}
}

func TestEnsureProjectGateway_ReturnsExisting(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw1, _ := svc.EnsureProjectGateway(ctx, "user-1", "proj-12345678", "my-project")
	gw2, _ := svc.EnsureProjectGateway(ctx, "user-1", "proj-12345678", "my-project")
	if gw1.ID != gw2.ID {
		t.Error("Expected same gateway ID on second call")
	}
}

func TestEnsureProjectGateway_EmptySlug(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, err := svc.EnsureProjectGateway(ctx, "user-1", "proj-abcdef12", "")
	if err != nil {
		t.Fatalf("EnsureProjectGateway with empty slug failed: %v", err)
	}
	if gw.Slug == "" {
		t.Error("Expected non-empty slug even with empty project slug")
	}
}

// --- CreateRoute tests ---

func TestCreateRoute_Success(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	// Create app owned by user-1
	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "my-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Route GW")

	route := &entities.GatewayRoute{
		Path:    "/api/v1",
		AppID:   app.ID,
		Methods: []string{"GET"},
	}
	created, err := svc.CreateRoute(ctx, gw.ID, route)
	if err != nil {
		t.Fatalf("CreateRoute failed: %v", err)
	}
	if created == nil {
		t.Fatal("Expected non-nil route")
	}
	if created.Path != "/api/v1" {
		t.Errorf("Expected path '/api/v1', got '%s'", created.Path)
	}
}

func TestCreateRoute_GatewayNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	route := &entities.GatewayRoute{Path: "/test", AppID: "app-1"}
	_, err := svc.CreateRoute(ctx, "nonexistent", route)
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

func TestCreateRoute_NoAppID(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "No App GW")
	route := &entities.GatewayRoute{Path: "/test"}
	_, err := svc.CreateRoute(ctx, gw.ID, route)
	if err == nil {
		t.Error("Expected error when app_id is missing")
	}
}

// --- DeleteRoute tests ---

func TestDeleteRoute_Success(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "my-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Del Route GW")
	route, _ := svc.CreateRoute(ctx, gw.ID, &entities.GatewayRoute{Path: "/delete-me", AppID: app.ID})

	err := svc.DeleteRoute(ctx, gw.ID, route.ID)
	if err != nil {
		t.Fatalf("DeleteRoute failed: %v", err)
	}
}

func TestDeleteRoute_NotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "GW")
	err := svc.DeleteRoute(ctx, gw.ID, "nonexistent-route")
	if err == nil {
		t.Error("Expected error for nonexistent route")
	}
}

// --- ListGatewayDomains tests ---

func TestListGatewayDomains_Empty(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Domain GW")
	domains, err := svc.ListGatewayDomains(ctx, gw.ID)
	if err != nil {
		t.Fatalf("ListGatewayDomains failed: %v", err)
	}
	if len(domains) != 0 {
		t.Errorf("Expected 0 domains, got %d", len(domains))
	}
}

// --- HandleAuthPoolDeleted tests ---

func TestHandleAuthPoolDeleted_EmptyList(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	// Should not panic with empty list
	svc.HandleAuthPoolDeleted(ctx, []string{})
}

func TestHandleAuthPoolDeleted_NonexistentGateways(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	// Should not panic with non-existent gateway IDs
	svc.HandleAuthPoolDeleted(ctx, []string{"gw-1", "gw-2"})
}

func TestHandleAuthPoolDeleted_WithRealGateway(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "AuthPool GW")
	svc.HandleAuthPoolDeleted(ctx, []string{gw.ID})
	// No panic means success
}

// --- AddGatewayDomain tests ---

func TestAddGatewayDomain_Success(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Domain GW2")
	cd, err := svc.AddGatewayDomain(ctx, gw.ID, "user-1", "api.example.com")
	if err != nil {
		t.Fatalf("AddGatewayDomain failed: %v", err)
	}
	if cd == nil {
		t.Fatal("Expected non-nil custom domain")
	}
}

func TestAddGatewayDomain_WrongUser(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Wrong User GW")
	_, err := svc.AddGatewayDomain(ctx, gw.ID, "user-2", "api.example.com")
	if err == nil {
		t.Error("Expected error for wrong user")
	}
}

func TestAddGatewayDomain_NotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	_, err := svc.AddGatewayDomain(ctx, "nonexistent", "user-1", "api.example.com")
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

// --- DeleteGatewayDomain tests ---

func TestDeleteGatewayDomain_GatewayNotFound(t *testing.T) {
	svc, _, _ := newTestGatewayService()
	ctx := context.Background()

	err := svc.DeleteGatewayDomain(ctx, "nonexistent", "dom-1")
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

// --- HandleAppDeleted tests ---

func TestHandleAppDeleted(t *testing.T) {
	svc, _, appRepo := newTestGatewayService()
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "deleted-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	gw, _ := svc.CreateGateway(ctx, "user-1", "project-1", "AppDel GW")
	svc.CreateRoute(ctx, gw.ID, &entities.GatewayRoute{Path: "/app-route", AppID: app.ID})

	// Should not panic even if routes reference the deleted app
	svc.HandleAppDeleted(ctx, app.ID)
}
