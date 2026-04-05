package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupImageVerifyTest() (*fiber.App, *handlers.ImageVerifyHandler, *memory.MemoryProjectRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	projectRepo := memory.NewMemoryProjectRepository()
	handler := handlers.NewImageVerifyHandler(projectRepo, "registry.stage.freezenith.com", "zenith-stage", "robot$zenith", "secret123")
	return app, handler, projectRepo
}

func TestImageVerifyGetRegistryCredentials(t *testing.T) {
	app, handler, projectRepo := setupImageVerifyTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "A test project")

	app.Get("/projects/:projectId/registry-credentials", injectUserID("user-1"), handler.GetRegistryCredentials)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/registry-credentials", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["available"] != true {
		t.Errorf("Expected available true, got %v", result["available"])
	}
	if result["registry"] != "registry.stage.freezenith.com" {
		t.Errorf("Expected registry host, got '%v'", result["registry"])
	}
	if result["username"] != "robot$zenith" {
		t.Errorf("Expected username 'robot$zenith', got '%v'", result["username"])
	}
}

func TestImageVerifyGetRegistryCredentialsNoAuth(t *testing.T) {
	fiberApp := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	projectRepo := memory.NewMemoryProjectRepository()
	handler := handlers.NewImageVerifyHandler(projectRepo, "registry.stage.freezenith.com", "zenith-stage", "", "")

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "A test project")

	fiberApp.Get("/projects/:projectId/registry-credentials", injectUserID("user-1"), handler.GetRegistryCredentials)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/registry-credentials", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["available"] != false {
		t.Errorf("Expected available false when no credentials, got %v", result["available"])
	}
}

func TestImageVerifyGetRegistryCredentialsNotYourProject(t *testing.T) {
	app, handler, projectRepo := setupImageVerifyTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "A test project")

	app.Get("/projects/:projectId/registry-credentials", injectUserID("user-2"), handler.GetRegistryCredentials)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/registry-credentials", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestImageVerifyGetRegistryCredentialsProjectNotFound(t *testing.T) {
	app, handler, _ := setupImageVerifyTest()

	app.Get("/projects/:projectId/registry-credentials", injectUserID("user-1"), handler.GetRegistryCredentials)

	req := httptest.NewRequest("GET", "/projects/nonexistent/registry-credentials", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestImageVerifyGetRegistryCredentialsNoUserID(t *testing.T) {
	app, handler, _ := setupImageVerifyTest()

	// No injectUserID middleware
	app.Get("/projects/:projectId/registry-credentials", handler.GetRegistryCredentials)

	req := httptest.NewRequest("GET", "/projects/some-id/registry-credentials", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestImageVerifyVerifyImagesNoAuth(t *testing.T) {
	app, handler, _ := setupImageVerifyTest()

	app.Post("/projects/:projectId/verify-images", handler.VerifyImages)

	body := `{"images":[]}`
	req := httptest.NewRequest("POST", "/projects/some-id/verify-images", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestImageVerifyVerifyImagesEmptyList(t *testing.T) {
	app, handler, projectRepo := setupImageVerifyTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "A test project")

	app.Post("/projects/:projectId/verify-images", injectUserID("user-1"), handler.VerifyImages)

	body := `{"images":[]}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/verify-images", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result handlers.VerifyImagesResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if !result.AllReady {
		t.Error("Expected AllReady true for empty images list")
	}
}

func TestImageVerifyVerifyImagesProjectNotFound(t *testing.T) {
	app, handler, _ := setupImageVerifyTest()

	app.Post("/projects/:projectId/verify-images", injectUserID("user-1"), handler.VerifyImages)

	body := `{"images":[{"name":"app","image":"nginx:latest"}]}`
	req := httptest.NewRequest("POST", "/projects/nonexistent/verify-images", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestImageVerifyVerifyImagesNotYourProject(t *testing.T) {
	app, handler, projectRepo := setupImageVerifyTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "A test project")

	app.Post("/projects/:projectId/verify-images", injectUserID("user-2"), handler.VerifyImages)

	body := `{"images":[{"name":"app","image":"nginx:latest"}]}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/verify-images", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}
