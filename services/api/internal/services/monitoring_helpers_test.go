package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
)

// --- podSelector additional tests (TestPodSelector already in monitoring_test.go) ---

func TestPodSelector_Empty(t *testing.T) {
	sel := podSelector("")
	if sel != "zenith.dev/app=" {
		t.Errorf("Expected 'zenith.dev/app=', got '%s'", sel)
	}
}

func TestPodSelector_SpecialChars(t *testing.T) {
	sel := podSelector("app-with-dashes")
	if sel != "zenith.dev/app=app-with-dashes" {
		t.Errorf("Expected 'zenith.dev/app=app-with-dashes', got '%s'", sel)
	}
}

// --- podRegex additional tests (TestPodRegex already in monitoring_test.go) ---

func TestPodRegex_Empty(t *testing.T) {
	r := podRegex("")
	if r != "-.*" {
		t.Errorf("Expected '-.*', got '%s'", r)
	}
}

// --- GetTimeSeries with all metrics (nil prom) ---

func TestGetTimeSeries_AllMetrics_NilProm(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "multi-metric-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	metrics := []string{"cpu", "memory", "requests", "latency"}
	for _, m := range metrics {
		resp, err := svc.GetTimeSeries(ctx, "user-1", app.ID, m, "1h")
		if err != nil {
			t.Errorf("GetTimeSeries(%s) failed: %v", m, err)
		}
		if resp.Metric != m {
			t.Errorf("Expected metric '%s', got '%s'", m, resp.Metric)
		}
	}
}

// --- GetTimeSeries with various time ranges ---

func TestGetTimeSeries_AllTimeRanges_NilProm(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "time-range-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	ranges := []string{"1h", "6h", "24h", "7d"}
	for _, r := range ranges {
		resp, err := svc.GetTimeSeries(ctx, "user-1", app.ID, "cpu", r)
		if err != nil {
			t.Errorf("GetTimeSeries(range=%s) failed: %v", r, err)
		}
		if resp.Range != r {
			t.Errorf("Expected range '%s', got '%s'", r, resp.Range)
		}
	}
}

// --- GetTimeSeries with app not found ---

func TestGetTimeSeries_AppNotFound(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	_, err := svc.GetTimeSeries(ctx, "user-1", "nonexistent", "cpu", "1h")
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

// --- GetLogs nil loki ---

func TestGetLogs_NilLoki(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "log-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	resp, err := svc.GetLogs(ctx, "user-1", app.ID, "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if len(resp.Entries) != 0 {
		t.Errorf("Expected 0 entries with nil loki, got %d", len(resp.Entries))
	}
	if resp.Total != 0 {
		t.Errorf("Expected total=0, got %d", resp.Total)
	}
}

func TestGetLogs_AppNotFound(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	_, err := svc.GetLogs(ctx, "user-1", "nonexistent", "", "", 0, 0)
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

// --- StreamLogs nil loki ---

func TestStreamLogs_NilLoki(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "stream-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	out := make(chan dto.MonitoringLogEntry, 10)
	err := svc.StreamLogs(ctx, "user-1", app.ID, out)
	if err == nil {
		t.Error("Expected error for nil loki client")
	}
}

func TestStreamLogs_AppNotFound(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	out := make(chan dto.MonitoringLogEntry, 10)
	err := svc.StreamLogs(ctx, "user-1", "nonexistent", out)
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

// --- GetAggregatedLogs nil loki ---

func TestGetAggregatedLogs_NilLoki(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	resp, err := svc.GetAggregatedLogs(ctx, "user-1", []string{"app-1", "app-2"}, "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetAggregatedLogs failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if len(resp.Entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(resp.Entries))
	}
}

// --- GetOverview with wrong user ---

func TestGetOverview_WrongUser(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "owner-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	_, err := svc.GetOverview(ctx, "user-2", app.ID)
	if err == nil {
		t.Error("Expected error for wrong user")
	}
}

// --- GetPods with wrong user ---

func TestGetPods_WrongUser(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	k8s := k8sclient.NewMemoryClient()
	svc := NewMonitoringService(nil, nil, k8s, appRepo)
	ctx := context.Background()

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		Name:         "pod-owner-app",
		UserID:       "user-1",
		ProjectID:    "project-1",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
	})

	_, err := svc.GetPods(ctx, "user-2", app.ID)
	if err == nil {
		t.Error("Expected error for wrong user")
	}
}
