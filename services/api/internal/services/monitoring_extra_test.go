package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
)

// --- NewMonitoringService tests ---

func TestNewMonitoringService(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	if svc == nil {
		t.Fatal("Expected non-nil MonitoringService")
	}
}

// --- resolveApp tests ---

func TestResolveApp_NotFound(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	_, err := svc.resolveApp(ctx, "user-1", "nonexistent-app")
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

func TestResolveApp_WrongUser(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	// Create an app owned by user-1
	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "my-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	// Try to resolve as user-2
	_, err := svc.resolveApp(ctx, "user-2", app.ID)
	if err == nil {
		t.Error("Expected error for wrong user")
	}
}

func TestResolveApp_Success(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "my-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	name, err := svc.resolveApp(ctx, "user-1", app.ID)
	if err != nil {
		t.Fatalf("resolveApp failed: %v", err)
	}
	if name != "my-app" {
		t.Errorf("Expected name 'my-app', got '%s'", name)
	}
}

// --- GetOverview tests ---

func TestGetOverview_AppNotFound(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	_, err := svc.GetOverview(ctx, "user-1", "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

func TestGetOverview_NilProm(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "my-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	overview, err := svc.GetOverview(ctx, "user-1", app.ID)
	if err != nil {
		t.Fatalf("GetOverview failed: %v", err)
	}
	if overview == nil {
		t.Fatal("Expected non-nil overview")
	}
	// With MemoryClient, ListPods returns 1 fake pod
	if overview.PodCount != 1 {
		t.Errorf("Expected 1 pod, got %d", overview.PodCount)
	}
	if overview.CPUPercent <= 0 {
		t.Error("Expected positive CPU percent from fake metrics")
	}
}

// --- GetTimeSeries tests ---

func TestGetTimeSeries_NilProm(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "ts-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	resp, err := svc.GetTimeSeries(ctx, "user-1", app.ID, "cpu", "1h")
	if err != nil {
		t.Fatalf("GetTimeSeries failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.Metric != "cpu" {
		t.Errorf("Expected metric 'cpu', got '%s'", resp.Metric)
	}
}

// --- GetPods tests ---

func TestGetPods_Success(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "pod-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	pods, err := svc.GetPods(ctx, "user-1", app.ID)
	if err != nil {
		t.Fatalf("GetPods failed: %v", err)
	}
	if len(pods.Pods) != 1 {
		t.Errorf("Expected 1 pod from MemoryClient, got %d", len(pods.Pods))
	}
}

func TestGetPods_AppNotFound(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	_, err := svc.GetPods(ctx, "user-1", "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}
