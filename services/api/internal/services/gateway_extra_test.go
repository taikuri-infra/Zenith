package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
)

// --- slugify tests ---

func TestSlugify_Simple(t *testing.T) {
	result := slugify("My Gateway")
	if result != "my-gateway" {
		t.Errorf("Expected 'my-gateway', got '%s'", result)
	}
}

func TestSlugify_SpecialChars(t *testing.T) {
	result := slugify("My API Gateway! #v2")
	if result != "my-api-gateway-v2" {
		t.Errorf("Expected 'my-api-gateway-v2', got '%s'", result)
	}
}

func TestSlugify_MultipleHyphens(t *testing.T) {
	result := slugify("my---gateway")
	if result != "my-gateway" {
		t.Errorf("Expected 'my-gateway', got '%s'", result)
	}
}

func TestSlugify_LeadingTrailingHyphens(t *testing.T) {
	result := slugify("--my-gateway--")
	if result != "my-gateway" {
		t.Errorf("Expected 'my-gateway', got '%s'", result)
	}
}

func TestSlugify_Empty(t *testing.T) {
	result := slugify("")
	if result != "gateway" {
		t.Errorf("Expected 'gateway' for empty input, got '%s'", result)
	}
}

func TestSlugify_OnlySpecialChars(t *testing.T) {
	result := slugify("!@#$%^&*()")
	if result != "gateway" {
		t.Errorf("Expected 'gateway' for all-special input, got '%s'", result)
	}
}

func TestSlugify_AlreadyClean(t *testing.T) {
	result := slugify("clean-slug-123")
	if result != "clean-slug-123" {
		t.Errorf("Expected 'clean-slug-123', got '%s'", result)
	}
}

func TestSlugify_Whitespace(t *testing.T) {
	result := slugify("  spaced  out  ")
	if result != "spaced-out" {
		t.Errorf("Expected 'spaced-out', got '%s'", result)
	}
}

// --- NewGatewayService tests ---

func TestNewGatewayService(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	if svc == nil {
		t.Fatal("Expected non-nil GatewayService")
	}
}

// --- SetAuthPoolRepo tests ---

func TestSetAuthPoolRepo(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	poolRepo := memory.NewMemoryAuthPoolRepository()
	svc.SetAuthPoolRepo(poolRepo)
	// No panic means success
}

// --- SetAppsDomain tests ---

func TestSetAppsDomain(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	svc.SetAppsDomain("apps.example.com")
	// No panic means success
}

// --- SetPromClient tests ---

func TestSetPromClient(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	svc.SetPromClient(nil)
	// No panic means success
}

// --- CreateGateway tests ---

func TestCreateGateway_Success(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	ctx := context.Background()

	gw, err := svc.CreateGateway(ctx, "user-1", "project-1", "My API Gateway")
	if err != nil {
		t.Fatalf("CreateGateway failed: %v", err)
	}
	if gw == nil {
		t.Fatal("Expected non-nil gateway")
	}
	if gw.Name != "My API Gateway" {
		t.Errorf("Expected name 'My API Gateway', got '%s'", gw.Name)
	}
	if gw.Slug != "my-api-gateway" {
		t.Errorf("Expected slug 'my-api-gateway', got '%s'", gw.Slug)
	}
	if gw.Endpoint != "https://my-api-gateway.gw.example.com" {
		t.Errorf("Expected endpoint 'https://my-api-gateway.gw.example.com', got '%s'", gw.Endpoint)
	}
}

func TestCreateGateway_DuplicateSlug(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	ctx := context.Background()

	svc.CreateGateway(ctx, "user-1", "project-1", "My Gateway")
	_, err := svc.CreateGateway(ctx, "user-1", "project-1", "My Gateway")
	if err == nil {
		t.Error("Expected error for duplicate gateway slug")
	}
}

// --- GetGateway tests ---

func TestGetGateway_Success(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	ctx := context.Background()

	created, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Test GW")
	gw, err := svc.GetGateway(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetGateway failed: %v", err)
	}
	if gw.Endpoint == "" {
		t.Error("Expected non-empty endpoint")
	}
}

func TestGetGateway_NotFound(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	ctx := context.Background()

	_, err := svc.GetGateway(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent gateway")
	}
}

// --- ListGateways tests ---

func TestListGateways_Empty(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	ctx := context.Background()

	gws, err := svc.ListGateways(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListGateways failed: %v", err)
	}
	if len(gws) != 0 {
		t.Errorf("Expected 0 gateways, got %d", len(gws))
	}
}

func TestListGateways_WithGateways(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	ctx := context.Background()

	svc.CreateGateway(ctx, "user-1", "project-1", "Gateway A")
	svc.CreateGateway(ctx, "user-1", "project-1", "Gateway B")

	gws, err := svc.ListGateways(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListGateways failed: %v", err)
	}
	if len(gws) != 2 {
		t.Errorf("Expected 2 gateways, got %d", len(gws))
	}
	for _, gw := range gws {
		if gw.Endpoint == "" {
			t.Error("Expected non-empty endpoint on listed gateway")
		}
	}
}

// --- UpdateGateway tests ---

func TestUpdateGateway_Success(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	ctx := context.Background()

	created, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Original Name")
	updated, err := svc.UpdateGateway(ctx, created.ID, "New Name")
	if err != nil {
		t.Fatalf("UpdateGateway failed: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got '%s'", updated.Name)
	}
	if updated.Endpoint == "" {
		t.Error("Expected non-empty endpoint after update")
	}
}

// --- DeleteGateway tests ---

func TestDeleteGateway_Success(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	ctx := context.Background()

	created, _ := svc.CreateGateway(ctx, "user-1", "project-1", "Delete Me")
	err := svc.DeleteGateway(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteGateway failed: %v", err)
	}

	// Verify it's gone
	_, err = svc.GetGateway(ctx, created.ID)
	if err == nil {
		t.Error("Expected error getting deleted gateway")
	}
}

func TestDeleteGateway_NotFound(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	ctx := context.Background()

	err := svc.DeleteGateway(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error deleting nonexistent gateway")
	}
}

// --- ListGroups tests ---

func TestListGroups_NoGateway(t *testing.T) {
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()

	svc := NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.example.com", "zenith-apps")
	ctx := context.Background()

	groups, err := svc.ListGroups(ctx, "nonexistent-gw")
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("Expected 0 groups, got %d", len(groups))
	}
}
