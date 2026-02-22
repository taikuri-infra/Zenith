package deploy

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/dotechhq/zenith/services/api/internal/store"
)

func TestDeployAppFreeTier(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	appRepo := store.NewMemoryAppRepository()
	planRepo := store.NewMemoryUserPlanRepository()

	ctx := context.Background()

	// Set up a free-tier user
	planRepo.SetUserPlan(ctx, "user-1", entities.PlanFree)

	// Create an app
	app := createTestApp(t, ctx, appRepo, "user-1", "sleepy-app")

	deployer := NewDeployer(k8sClient, appRepo, planRepo, "freezenith.com")
	if err := deployer.DeployApp(ctx, app, "sleepy-app:v1"); err != nil {
		t.Fatalf("DeployApp failed: %v", err)
	}

	// Verify HTTPScaledObject was created
	_, err := k8sClient.GetCRD(ctx, "HTTPScaledObject", "zenith-apps", "sleepy-app")
	if err != nil {
		t.Errorf("HTTPScaledObject not found: %v", err)
	}

	// Verify app status is sleeping
	updated, err := appRepo.GetApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetApp failed: %v", err)
	}
	if updated.Status != entities.AppStatusSleeping {
		t.Errorf("app status = %v, want sleeping", updated.Status)
	}
}

func TestDeployAppPaidTier(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	appRepo := store.NewMemoryAppRepository()
	planRepo := store.NewMemoryUserPlanRepository()

	ctx := context.Background()

	// Set up a pro-tier user
	planRepo.SetUserPlan(ctx, "user-2", entities.PlanPro)

	app := createTestApp(t, ctx, appRepo, "user-2", "pro-app")

	deployer := NewDeployer(k8sClient, appRepo, planRepo, "freezenith.com")
	if err := deployer.DeployApp(ctx, app, "pro-app:v1"); err != nil {
		t.Fatalf("DeployApp failed: %v", err)
	}

	// Verify no HTTPScaledObject was created
	_, err := k8sClient.GetCRD(ctx, "HTTPScaledObject", "zenith-apps", "pro-app")
	if err == nil {
		t.Error("HTTPScaledObject should not exist for paid tier")
	}

	// Verify app status is running
	updated, err := appRepo.GetApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetApp failed: %v", err)
	}
	if updated.Status != entities.AppStatusRunning {
		t.Errorf("app status = %v, want running", updated.Status)
	}
}

func TestDeleteAppCleansUpHTTPScaledObject(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	appRepo := store.NewMemoryAppRepository()
	planRepo := store.NewMemoryUserPlanRepository()

	ctx := context.Background()

	planRepo.SetUserPlan(ctx, "user-3", entities.PlanFree)

	app := createTestApp(t, ctx, appRepo, "user-3", "delete-me")

	deployer := NewDeployer(k8sClient, appRepo, planRepo, "freezenith.com")
	if err := deployer.DeployApp(ctx, app, "delete-me:v1"); err != nil {
		t.Fatalf("DeployApp failed: %v", err)
	}

	// Confirm HTTPScaledObject exists
	_, err := k8sClient.GetCRD(ctx, "HTTPScaledObject", "zenith-apps", "delete-me")
	if err != nil {
		t.Fatalf("HTTPScaledObject should exist before delete: %v", err)
	}

	// Delete app
	if err := deployer.DeleteApp(ctx, app); err != nil {
		t.Fatalf("DeleteApp failed: %v", err)
	}

	// Verify HTTPScaledObject was cleaned up
	_, err = k8sClient.GetCRD(ctx, "HTTPScaledObject", "zenith-apps", "delete-me")
	if err == nil {
		t.Error("HTTPScaledObject should not exist after DeleteApp")
	}
}

func TestDeployAppNilPlanRepo(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	appRepo := store.NewMemoryAppRepository()

	ctx := context.Background()

	app := createTestApp(t, ctx, appRepo, "user-4", "no-plan-repo")

	// nil planRepo should still work (backwards compatible, always-on)
	deployer := NewDeployer(k8sClient, appRepo, nil, "freezenith.com")
	if err := deployer.DeployApp(ctx, app, "no-plan-repo:v1"); err != nil {
		t.Fatalf("DeployApp failed: %v", err)
	}

	// Should be running (not sleeping)
	updated, err := appRepo.GetApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetApp failed: %v", err)
	}
	if updated.Status != entities.AppStatusRunning {
		t.Errorf("app status = %v, want running", updated.Status)
	}
}

// createTestApp is a helper that creates a test app via the app repo.
func createTestApp(t *testing.T, ctx context.Context, appRepo store.AppRepository, userID, name string) *entities.App {
	t.Helper()
	app, err := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		UserID:  userID,
		Name:    name,
		RepoURL: "https://github.com/test/" + name,
		Branch:  "main",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	return app
}
