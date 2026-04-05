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

func setupReleaseTest() (*fiber.App, *handlers.ReleaseHandler, *memory.MemoryAppRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	appRepo := memory.NewMemoryAppRepository()
	// Pass nil for deploy.Pipeline — CreateRelease and ListReleases don't use it
	handler := handlers.NewReleaseHandler(appRepo, nil)
	return app, handler, appRepo
}

func createReleaseTestApp(appRepo *memory.MemoryAppRepository) *entities.App {
	a, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		Name:         "release-app",
		UserID:       "user-1",
		ProjectID:    "proj-1",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "registry.example.com/test:latest",
	})
	return a
}

func TestReleaseCreate(t *testing.T) {
	app, handler, appRepo := setupReleaseTest()
	testApp := createReleaseTestApp(appRepo)

	app.Post("/api/v1/apps/:appId/releases", handler.CreateRelease)

	body := `{"image":"registry.example.com/test:v1.0.0","git_sha":"abc123","branch":"main","message":"first release"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/releases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["image"] != "registry.example.com/test:v1.0.0" {
		t.Errorf("Expected image in response, got '%v'", result["image"])
	}
}

func TestReleaseCreateNoImage(t *testing.T) {
	app, handler, appRepo := setupReleaseTest()
	testApp := createReleaseTestApp(appRepo)

	app.Post("/api/v1/apps/:appId/releases", handler.CreateRelease)

	body := `{"git_sha":"abc123"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/releases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestReleaseCreateDefaultBranch(t *testing.T) {
	app, handler, appRepo := setupReleaseTest()
	testApp := createReleaseTestApp(appRepo)

	app.Post("/api/v1/apps/:appId/releases", handler.CreateRelease)

	// No branch specified — should default to "main"
	body := `{"image":"registry.example.com/test:v1.0.0"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/releases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}
}

func TestReleaseList(t *testing.T) {
	app, handler, appRepo := setupReleaseTest()
	testApp := createReleaseTestApp(appRepo)

	// Create releases directly
	appRepo.CreateRelease(nil, testApp.ID, &dto.CreateReleaseInput{Image: "test:v1", Branch: "main"})
	appRepo.CreateRelease(nil, testApp.ID, &dto.CreateReleaseInput{Image: "test:v2", Branch: "main"})

	app.Get("/api/v1/apps/:appId/releases", handler.ListReleases)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/releases", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Releases []entities.Release `json:"releases"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Releases) != 2 {
		t.Errorf("Expected 2 releases, got %d", len(result.Releases))
	}
}

func TestReleaseListEmpty(t *testing.T) {
	app, handler, appRepo := setupReleaseTest()
	testApp := createReleaseTestApp(appRepo)

	app.Get("/api/v1/apps/:appId/releases", handler.ListReleases)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/releases", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}
