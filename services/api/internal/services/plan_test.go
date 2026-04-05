package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func newTestPlanService() *PlanService {
	planRepo := memory.NewMemoryUserPlanRepository()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	storageRepo := memory.NewMemoryStorageRepository()
	authRepo := memory.NewMemoryAppAuthRepository()
	svc := NewPlanService(planRepo, appRepo, dbRepo, storageRepo, authRepo)
	svc.SetGatewayRepo(memory.NewMemoryGatewayRepository())
	svc.SetAuthPoolRepo(memory.NewMemoryAuthPoolRepository())
	return svc
}

// --- GetUserPlan tests ---

func TestGetUserPlan_DefaultFree(t *testing.T) {
	svc := newTestPlanService()
	ctx := context.Background()

	resp, err := svc.GetUserPlan(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserPlan failed: %v", err)
	}
	if resp.Tier != entities.PlanFree {
		t.Errorf("Expected tier free, got %s", resp.Tier)
	}
	if resp.Limits.MaxApps != 1 {
		t.Errorf("Expected max apps 1 for free plan, got %d", resp.Limits.MaxApps)
	}
}

func TestGetUserPlan_AfterUpgrade(t *testing.T) {
	svc := newTestPlanService()
	ctx := context.Background()

	_, err := svc.UpgradePlan(ctx, "user-1", entities.PlanPro)
	if err != nil {
		t.Fatalf("UpgradePlan failed: %v", err)
	}

	resp, err := svc.GetUserPlan(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserPlan failed: %v", err)
	}
	if resp.Tier != entities.PlanPro {
		t.Errorf("Expected tier pro, got %s", resp.Tier)
	}
	if resp.Limits.MaxApps != 5 {
		t.Errorf("Expected max apps 5 for pro plan, got %d", resp.Limits.MaxApps)
	}
}

// --- UpgradePlan tests ---

func TestUpgradePlan_FreeToProWithStripeDisabled(t *testing.T) {
	svc := newTestPlanService()
	ctx := context.Background()

	resp, err := svc.UpgradePlan(ctx, "user-1", entities.PlanPro)
	if err != nil {
		t.Fatalf("UpgradePlan failed: %v", err)
	}
	if resp.Tier != entities.PlanPro {
		t.Errorf("Expected tier pro, got %s", resp.Tier)
	}
}

func TestUpgradePlan_PaidTierBlockedWithStripe(t *testing.T) {
	svc := newTestPlanService()
	svc.SetStripeEnabled(true)
	ctx := context.Background()

	_, err := svc.UpgradePlan(ctx, "user-1", entities.PlanPro)
	if err == nil {
		t.Error("Expected error when upgrading to paid tier with Stripe enabled")
	}
}

func TestUpgradePlan_FreeAllowedWithStripe(t *testing.T) {
	svc := newTestPlanService()
	svc.SetStripeEnabled(true)
	ctx := context.Background()

	resp, err := svc.UpgradePlan(ctx, "user-1", entities.PlanFree)
	if err != nil {
		t.Fatalf("UpgradePlan to free with Stripe should work: %v", err)
	}
	if resp.Tier != entities.PlanFree {
		t.Errorf("Expected tier free, got %s", resp.Tier)
	}
}

// --- CheckLimit tests ---

func TestCheckLimit_UnderLimit(t *testing.T) {
	svc := newTestPlanService()
	ctx := context.Background()

	err := svc.CheckLimit(ctx, "user-1", "apps", 0)
	if err != nil {
		t.Errorf("Expected no error when under limit, got: %v", err)
	}
}

func TestCheckLimit_AtLimit(t *testing.T) {
	svc := newTestPlanService()
	ctx := context.Background()

	// Free plan max apps = 1
	err := svc.CheckLimit(ctx, "user-1", "apps", 1)
	if err == nil {
		t.Error("Expected error when at limit")
	}
}

func TestCheckLimit_OverLimit(t *testing.T) {
	svc := newTestPlanService()
	ctx := context.Background()

	err := svc.CheckLimit(ctx, "user-1", "apps", 5)
	if err == nil {
		t.Error("Expected error when over limit")
	}
}

func TestCheckLimit_UnknownResource(t *testing.T) {
	svc := newTestPlanService()
	ctx := context.Background()

	err := svc.CheckLimit(ctx, "user-1", "unknown-resource", 999)
	if err != nil {
		t.Errorf("Expected nil for unknown resource, got: %v", err)
	}
}

func TestCheckLimit_AllResources_ProPlan(t *testing.T) {
	planRepo := memory.NewMemoryUserPlanRepository()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	storageRepo := memory.NewMemoryStorageRepository()
	authRepo := memory.NewMemoryAppAuthRepository()
	svc := NewPlanService(planRepo, appRepo, dbRepo, storageRepo, authRepo)
	svc.SetGatewayRepo(memory.NewMemoryGatewayRepository())
	svc.SetAuthPoolRepo(memory.NewMemoryAuthPoolRepository())
	ctx := context.Background()

	// Use Pro plan which has non-zero limits for all resources
	planRepo.SetUserPlan(ctx, "user-pro", entities.PlanPro)

	resources := []string{"apps", "databases", "buckets", "gateways", "gateway_routes", "auth_pools"}
	for _, res := range resources {
		err := svc.CheckLimit(ctx, "user-pro", res, 0)
		if err != nil {
			t.Errorf("Expected no error for resource %s with count 0 on Pro plan, got: %v", res, err)
		}
	}
}

func TestCheckLimit_FreePlanBucketsBlocked(t *testing.T) {
	svc := newTestPlanService()
	ctx := context.Background()

	// Free plan has MaxBuckets = 0, so even count 0 is at limit
	err := svc.CheckLimit(ctx, "user-1", "buckets", 0)
	if err == nil {
		t.Error("Expected error: free plan has 0 bucket limit, so count 0 should be at/over limit")
	}
}

// --- CalculateUsage tests ---

func TestCalculateUsage_Empty(t *testing.T) {
	svc := newTestPlanService()
	ctx := context.Background()

	usage := svc.CalculateUsage(ctx, "user-1")
	if usage.Apps != 0 {
		t.Errorf("Expected 0 apps, got %d", usage.Apps)
	}
	if usage.Databases != 0 {
		t.Errorf("Expected 0 databases, got %d", usage.Databases)
	}
	if usage.Buckets != 0 {
		t.Errorf("Expected 0 buckets, got %d", usage.Buckets)
	}
	if usage.Gateways != 0 {
		t.Errorf("Expected 0 gateways, got %d", usage.Gateways)
	}
}

// --- EnforceDowngrade tests ---

func TestEnforceDowngrade_SuspendsExcessApps(t *testing.T) {
	planRepo := memory.NewMemoryUserPlanRepository()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	storageRepo := memory.NewMemoryStorageRepository()
	authRepo := memory.NewMemoryAppAuthRepository()
	svc := NewPlanService(planRepo, appRepo, dbRepo, storageRepo, authRepo)
	ctx := context.Background()

	userID := "user-downgrade"
	projectID := "proj-1"

	// Create 3 apps (as if user was on Pro plan)
	for i := 0; i < 3; i++ {
		appRepo.CreateApp(ctx, &dto.CreateAppInput{
			Name:         "app-" + string(rune('a'+i)),
			UserID:       userID,
			ProjectID:    projectID,
			DeploySource: entities.DeploySourceImage,
			ImageURL:     "registry.example.com/img:latest",
		})
	}

	// Downgrade to Free (max 1 app)
	svc.EnforceDowngrade(ctx, userID, entities.PlanFree)

	apps, _ := appRepo.ListAppsByUser(ctx, userID)
	suspendedCount := 0
	for _, a := range apps {
		if a.Status == entities.AppStatusSuspended {
			suspendedCount++
		}
	}
	if suspendedCount != 2 {
		t.Errorf("Expected 2 suspended apps after downgrade, got %d", suspendedCount)
	}
}

// --- tierRank (indirectly via UpgradePlan downgrade detection) ---

func TestUpgradePlan_Downgrade(t *testing.T) {
	planRepo := memory.NewMemoryUserPlanRepository()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	storageRepo := memory.NewMemoryStorageRepository()
	authRepo := memory.NewMemoryAppAuthRepository()
	svc := NewPlanService(planRepo, appRepo, dbRepo, storageRepo, authRepo)
	ctx := context.Background()

	userID := "user-downgrade-test"

	// Start on Pro
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro)

	// Downgrade to Free
	resp, err := svc.UpgradePlan(ctx, userID, entities.PlanFree)
	if err != nil {
		t.Fatalf("Downgrade failed: %v", err)
	}
	if resp.Tier != entities.PlanFree {
		t.Errorf("Expected tier free after downgrade, got %s", resp.Tier)
	}
}
