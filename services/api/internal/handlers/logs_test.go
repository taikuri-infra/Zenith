package handlers_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/deploy"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

func setupLogTest() (*fiber.App, *handlers.LogHandler, *deploy.LogHub, ports.AppRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	repo := memory.NewMemoryAppRepository()
	hub := deploy.NewLogHub(100)
	logHandler := handlers.NewLogHandler(repo, hub)
	return app, logHandler, hub, repo
}

func TestGetLogsHistoryEmpty(t *testing.T) {
	fiberApp, logHandler, _, repo := setupLogTest()

	ctx := context.Background()
	userApp, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "logtest", RepoURL: "https://github.com/u/r",
	})
	dep, _ := repo.CreateDeployment(ctx, userApp.ID, "abc123")

	fiberApp.Get("/api/v1/apps/:appId/deployments/:did/logs/history", logHandler.GetLogs)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+userApp.ID+"/deployments/"+dep.ID+"/logs/history", nil)
	resp, err := fiberApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []interface{} `json:"items"`
		Total int           `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 0 {
		t.Errorf("Expected 0 entries, got %d", result.Total)
	}
}

func TestGetLogsHistoryWithEntries(t *testing.T) {
	fiberApp, logHandler, hub, repo := setupLogTest()

	ctx := context.Background()
	userApp, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "logtest2", RepoURL: "https://github.com/u/r",
	})
	dep, _ := repo.CreateDeployment(ctx, userApp.ID, "def456")

	// Publish entries to the hub before the request
	hub.PublishInfo(dep.ID, "cloning repository...")
	hub.PublishBuild(dep.ID, "building docker image...")
	hub.PublishDeploy(dep.ID, "applying kubernetes manifests...")

	fiberApp.Get("/api/v1/apps/:appId/deployments/:did/logs/history", logHandler.GetLogs)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+userApp.ID+"/deployments/"+dep.ID+"/logs/history", nil)
	resp, err := fiberApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			Level   string `json:"level"`
			Message string `json:"message"`
		} `json:"items"`
		Total int `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 3 {
		t.Fatalf("Expected 3 log entries, got %d", result.Total)
	}
	if result.Items[0].Level != "info" {
		t.Errorf("Expected first entry level 'info', got '%s'", result.Items[0].Level)
	}
	if result.Items[1].Level != "build" {
		t.Errorf("Expected second entry level 'build', got '%s'", result.Items[1].Level)
	}
	if result.Items[2].Level != "deploy" {
		t.Errorf("Expected third entry level 'deploy', got '%s'", result.Items[2].Level)
	}
}

func TestGetLogsHistoryAppNotFound(t *testing.T) {
	fiberApp, logHandler, _, _ := setupLogTest()

	fiberApp.Get("/api/v1/apps/:appId/deployments/:did/logs/history", logHandler.GetLogs)

	req := httptest.NewRequest("GET", "/api/v1/apps/nonexistent/deployments/deploy-1/logs/history", nil)
	resp, err := fiberApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}
