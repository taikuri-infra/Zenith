package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupPreviewTest() (*fiber.App, *handlers.PreviewHandler, *memory.MemoryPreviewRepository, *memory.MemoryAppRepository, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	previewRepo := memory.NewMemoryPreviewRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewPreviewHandler(previewRepo, appRepo, planRepo)
	return app, handler, previewRepo, appRepo, planRepo
}

func TestPreviewCreateTeamPlan(t *testing.T) {
	app, handler, _, appRepo, planRepo := setupPreviewTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanTeam)

	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		Name:         "myapp",
		UserID:       "user-1",
		ProjectID:    "proj-1",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "registry.example.com/test:latest",
	})

	app.Post("/api/v1/apps/:appId/previews", injectUserID("user-1"), handler.Create)

	body := `{"pr_number":42,"branch":"feature/test","git_sha":"abc123"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/previews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["pr_number"] == nil {
		t.Error("Expected pr_number in response")
	}
	if result["status"] != "building" {
		t.Errorf("Expected status 'building', got '%v'", result["status"])
	}
}

func TestPreviewCreateFreePlanForbidden(t *testing.T) {
	app, handler, _, appRepo, _ := setupPreviewTest()
	// Default is free plan — no SetUserPlan call

	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		Name:         "myapp",
		UserID:       "user-1",
		ProjectID:    "proj-1",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "registry.example.com/test:latest",
	})

	app.Post("/api/v1/apps/:appId/previews", injectUserID("user-1"), handler.Create)

	body := `{"pr_number":42,"branch":"feature/test"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/previews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestPreviewCreateMissingFields(t *testing.T) {
	app, handler, _, appRepo, planRepo := setupPreviewTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanTeam)

	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		Name:         "myapp",
		UserID:       "user-1",
		ProjectID:    "proj-1",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "registry.example.com/test:latest",
	})

	app.Post("/api/v1/apps/:appId/previews", injectUserID("user-1"), handler.Create)

	body := `{"branch":"feature/test"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/previews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestPreviewList(t *testing.T) {
	app, handler, previewRepo, _, _ := setupPreviewTest()

	previewRepo.CreatePreview(nil, "app-1", 1, "feat-a", "sha1", "https://app-pr-1.freezenith.com")
	previewRepo.CreatePreview(nil, "app-1", 2, "feat-b", "sha2", "https://app-pr-2.freezenith.com")

	app.Get("/api/v1/apps/:appId/previews", handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/app-1/previews", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.PreviewDeployment `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 previews, got %d", len(result.Items))
	}
}

func TestPreviewListEmpty(t *testing.T) {
	app, handler, _, _, _ := setupPreviewTest()
	app.Get("/api/v1/apps/:appId/previews", handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/app-1/previews", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.PreviewDeployment `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 0 {
		t.Errorf("Expected 0 previews, got %d", len(result.Items))
	}
}

func TestPreviewDelete(t *testing.T) {
	app, handler, previewRepo, _, _ := setupPreviewTest()

	preview, _ := previewRepo.CreatePreview(nil, "app-1", 1, "feat-a", "sha1", "https://app-pr-1.freezenith.com")

	app.Delete("/api/v1/apps/:appId/previews/:previewId", handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/app-1/previews/"+preview.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 204 {
		t.Fatalf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestPreviewDeleteNotFound(t *testing.T) {
	app, handler, _, _, _ := setupPreviewTest()
	app.Delete("/api/v1/apps/:appId/previews/:previewId", handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/app-1/previews/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}
