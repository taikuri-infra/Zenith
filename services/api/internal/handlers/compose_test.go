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

func setupComposeTest() (*fiber.App, *handlers.ComposeHandler, *memory.MemoryProjectRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	projectRepo := memory.NewMemoryProjectRepository()
	handler := handlers.NewComposeHandler(projectRepo)
	handler.SetBaseDomain("apps.stage.freezenith.com")
	return app, handler, projectRepo
}

func TestComposeImportNoAuth(t *testing.T) {
	app, handler, _ := setupComposeTest()
	app.Post("/projects/:projectId/import-compose", handler.ImportCompose)

	body := `{"compose_content":"version: '3'"}`
	req := httptest.NewRequest("POST", "/projects/some-id/import-compose", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestComposeImportProjectNotFound(t *testing.T) {
	app, handler, _ := setupComposeTest()
	app.Post("/projects/:projectId/import-compose", injectUserID("user-1"), handler.ImportCompose)

	body := `{"compose_content":"version: '3'"}`
	req := httptest.NewRequest("POST", "/projects/nonexistent/import-compose", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestComposeImportNotYourProject(t *testing.T) {
	app, handler, projectRepo := setupComposeTest()
	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	app.Post("/projects/:projectId/import-compose", injectUserID("user-2"), handler.ImportCompose)

	body := `{"compose_content":"version: '3'"}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/import-compose", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestComposeImportEmptyContent(t *testing.T) {
	app, handler, projectRepo := setupComposeTest()
	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	app.Post("/projects/:projectId/import-compose", injectUserID("user-1"), handler.ImportCompose)

	body := `{"compose_content":""}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/import-compose", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestComposeImportValidCompose(t *testing.T) {
	app, handler, projectRepo := setupComposeTest()
	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	app.Post("/projects/:projectId/import-compose", injectUserID("user-1"), handler.ImportCompose)

	composeContent := `services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
  db:
    image: postgres:16
    environment:
      POSTGRES_DB: mydb
      POSTGRES_PASSWORD: secret`

	body, _ := json.Marshal(map[string]string{"compose_content": composeContent})
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/import-compose", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	// Should have parsed services
	if result["services"] == nil {
		t.Error("Expected services in response")
	}
}

func TestComposeFormatNoAuth(t *testing.T) {
	app, handler, _ := setupComposeTest()
	app.Post("/projects/:projectId/format-compose", handler.FormatCompose)

	body := `{"compose_content":"broken yaml"}`
	req := httptest.NewRequest("POST", "/projects/some-id/format-compose", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestComposeFormatNoAIValidator(t *testing.T) {
	app, handler, projectRepo := setupComposeTest()
	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	// AI validator is not set
	app.Post("/projects/:projectId/format-compose", injectUserID("user-1"), handler.FormatCompose)

	body := `{"compose_content":"broken yaml"}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/format-compose", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 500 {
		t.Errorf("Expected 500 (AI validator not configured), got %d", resp.StatusCode)
	}
}

func TestComposeFormatEmptyContent(t *testing.T) {
	app, handler, projectRepo := setupComposeTest()
	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	app.Post("/projects/:projectId/format-compose", injectUserID("user-1"), handler.FormatCompose)

	body := `{"compose_content":""}`
	req := httptest.NewRequest("POST", "/projects/"+project.ID+"/format-compose", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
