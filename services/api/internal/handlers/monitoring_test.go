package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

func setupMonitoringTest(t *testing.T) (*fiber.App, string) {
	t.Helper()

	appRepo := memory.NewMemoryAppRepository()
	k8sClient := k8sclient.NewMemoryClient()

	// Create a test app via the DTO
	created, err := appRepo.CreateApp(context.Background(), &dto.CreateAppInput{
		UserID:       "user-1",
		Name:         "test-app",
		DeploySource: "image",
		ImageURL:     "nginx:latest",
		Port:         8080,
	})
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	// MonitoringService with nil prom/loki (graceful fallback)
	svc := services.NewMonitoringService(nil, nil, k8sClient, appRepo)
	handler := handlers.NewMonitoringHandler(svc)

	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})

	// Simulate auth middleware by setting user_id in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return c.Next()
	})

	app.Get("/api/v1/apps/:appId/metrics/overview", handler.GetOverview)
	app.Get("/api/v1/apps/:appId/metrics/timeseries", handler.GetTimeSeries)
	app.Get("/api/v1/apps/:appId/logs", handler.GetLogs)
	app.Get("/api/v1/apps/:appId/pods", handler.GetPods)

	return app, created.ID
}

func TestGetMetricsOverview(t *testing.T) {
	app, appID := setupMonitoringTest(t)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/metrics/overview", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test metrics overview: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var overview dto.MetricsOverview
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &overview); err != nil {
		t.Fatalf("Failed to unmarshal overview: %v", err)
	}

	if overview.PodCount < 0 {
		t.Errorf("Expected non-negative pod count, got %d", overview.PodCount)
	}
}

func TestGetMetricsOverviewNotFound(t *testing.T) {
	app, _ := setupMonitoringTest(t)

	req := httptest.NewRequest("GET", "/api/v1/apps/nonexistent-id/metrics/overview", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("Expected 404 for nonexistent app, got %d", resp.StatusCode)
	}
}

func TestGetTimeSeries(t *testing.T) {
	app, appID := setupMonitoringTest(t)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/metrics/timeseries?metric=cpu&range=1h", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test timeseries: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var ts dto.TimeSeriesResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &ts); err != nil {
		t.Fatalf("Failed to unmarshal timeseries: %v", err)
	}

	if ts.Metric != "cpu" {
		t.Errorf("Expected metric 'cpu', got '%s'", ts.Metric)
	}
}

func TestGetLogs(t *testing.T) {
	app, appID := setupMonitoringTest(t)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/logs?limit=10&since=1h", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test logs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestGetPods(t *testing.T) {
	app, appID := setupMonitoringTest(t)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/pods", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test pods: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestGetPodsNotFound(t *testing.T) {
	app, _ := setupMonitoringTest(t)

	req := httptest.NewRequest("GET", "/api/v1/apps/nonexistent-id/pods", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("Expected 404 for nonexistent app, got %d", resp.StatusCode)
	}
}
