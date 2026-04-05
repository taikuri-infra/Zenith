package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func setupEnvironmentTest() (*fiber.App, *handlers.EnvironmentHandler, *memory.MemoryProjectRepository, *memory.MemoryEnvironmentRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	envRepo := memory.NewMemoryEnvironmentRepository()
	projectRepo := memory.NewMemoryProjectRepository()
	handler := handlers.NewEnvironmentHandler(envRepo, projectRepo)
	return app, handler, projectRepo, envRepo
}

func TestEnvironmentList(t *testing.T) {
	fiberApp, handler, projectRepo, envRepo := setupEnvironmentTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	envRepo.CreateEnvironment(nil, &entities.Environment{
		ID:        uuid.New().String(),
		ProjectID: project.ID,
		Name:      entities.EnvironmentProduction,
		Slug:      "prod",
		Status:    entities.EnvironmentStatusActive,
		IsDefault: true,
	})
	envRepo.CreateEnvironment(nil, &entities.Environment{
		ID:        uuid.New().String(),
		ProjectID: project.ID,
		Name:      entities.EnvironmentStaging,
		Slug:      "staging",
		Status:    entities.EnvironmentStatusActive,
		IsDefault: false,
	})

	fiberApp.Get("/projects/:projectId/environments", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/environments", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Environments []entities.Environment `json:"environments"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Environments) != 2 {
		t.Errorf("Expected 2 environments, got %d", len(result.Environments))
	}
}

func TestEnvironmentListNoAuth(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupEnvironmentTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Get("/projects/:projectId/environments", handler.List)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/environments", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestEnvironmentListProjectNotFound(t *testing.T) {
	fiberApp, handler, _, _ := setupEnvironmentTest()
	fiberApp.Get("/projects/:projectId/environments", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/projects/nonexistent/environments", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestEnvironmentListForbidden(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupEnvironmentTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Get("/projects/:projectId/environments", injectUserID("user-2"), handler.List)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/environments", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestEnvironmentGet(t *testing.T) {
	fiberApp, handler, projectRepo, envRepo := setupEnvironmentTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")
	envID := uuid.New().String()
	envRepo.CreateEnvironment(nil, &entities.Environment{
		ID:        envID,
		ProjectID: project.ID,
		Name:      entities.EnvironmentProduction,
		Slug:      "prod",
		Status:    entities.EnvironmentStatusActive,
		IsDefault: true,
	})

	fiberApp.Get("/projects/:projectId/environments/:envId", injectUserID("user-1"), handler.Get)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/environments/"+envID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result entities.Environment
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Name != entities.EnvironmentProduction {
		t.Errorf("Expected production environment, got '%s'", result.Name)
	}
}

func TestEnvironmentGetNotFound(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupEnvironmentTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Get("/projects/:projectId/environments/:envId", injectUserID("user-1"), handler.Get)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/environments/nonexistent", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestEnvironmentGetForbidden(t *testing.T) {
	fiberApp, handler, projectRepo, envRepo := setupEnvironmentTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")
	envID := uuid.New().String()
	envRepo.CreateEnvironment(nil, &entities.Environment{
		ID:        envID,
		ProjectID: project.ID,
		Name:      entities.EnvironmentProduction,
		Slug:      "prod",
		Status:    entities.EnvironmentStatusActive,
		IsDefault: true,
	})

	fiberApp.Get("/projects/:projectId/environments/:envId", injectUserID("user-2"), handler.Get)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/environments/"+envID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestEnvironmentListEmpty(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupEnvironmentTest()

	project, _ := projectRepo.CreateProject(nil, "user-1", "My Project", "my-project", "desc")

	fiberApp.Get("/projects/:projectId/environments", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/projects/"+project.ID+"/environments", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}
