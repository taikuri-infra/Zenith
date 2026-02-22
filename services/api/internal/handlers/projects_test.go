package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/gofiber/fiber/v2"
)

func setupProjectApp() (*fiber.App, *handlers.ProjectHandler) {
	app := fiber.New(fiber.Config{
		ErrorHandler: handlers.ErrorHandler,
	})
	client := k8sclient.NewMemoryClient()
	handler := handlers.NewProjectHandler(client)
	return app, handler
}

func injectUser(c *fiber.Ctx) error {
	c.Locals("user_id", "user-123")
	c.Locals("email", "test@example.com")
	c.Locals("role", entities.RoleOwner)
	return c.Next()
}

func TestCreateProject(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)

	body := `{"name": "My Project", "plan": "pro", "region": "fsn1"}`
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(b))
	}

	var result handlers.ProjectResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Name != "My Project" {
		t.Errorf("Expected name 'My Project', got '%s'", result.Name)
	}
	if result.Plan != "pro" {
		t.Errorf("Expected plan 'pro', got '%s'", result.Plan)
	}
	if result.Owner != "test@example.com" {
		t.Errorf("Expected owner 'test@example.com', got '%s'", result.Owner)
	}
	if result.Phase != "Pending" {
		t.Errorf("Expected phase 'Pending', got '%s'", result.Phase)
	}
}

func TestCreateProjectDefaults(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)

	body := `{"name": "Default Project"}`
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.ProjectResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Plan != "free" {
		t.Errorf("Expected default plan 'free', got '%s'", result.Plan)
	}
	if result.Region != "fsn1" {
		t.Errorf("Expected default region 'fsn1', got '%s'", result.Region)
	}
}

func TestCreateProjectNoName(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)

	body := `{"plan": "free"}`
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateProjectInvalidPlan(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)

	body := `{"name": "Test", "plan": "invalid"}`
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestListProjects(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)
	app.Get("/api/v1/projects", handler.List)

	// Create 2 projects
	for _, name := range []string{"Project One", "Project Two"} {
		body := `{"name": "` + name + `"}`
		req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}

	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result dto.ListResponse[handlers.ProjectResponse]
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Items) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(result.Items))
	}
}

func TestGetProject(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)
	app.Get("/api/v1/projects/:id", handler.Get)

	// Create project
	body := `{"name": "Get Me"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)
	defer createResp.Body.Close()

	var created handlers.ProjectResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Get project
	req := httptest.NewRequest("GET", "/api/v1/projects/"+created.ID, nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result handlers.ProjectResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ID != created.ID {
		t.Errorf("Expected ID '%s', got '%s'", created.ID, result.ID)
	}
}

func TestGetProjectNotFound(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Get("/api/v1/projects/:id", handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/projects/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateProject(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)
	app.Put("/api/v1/projects/:id", handler.Update)

	// Create
	createBody := `{"name": "Original"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)
	defer createResp.Body.Close()

	var created handlers.ProjectResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Update
	updateBody := `{"name": "Updated", "plan": "enterprise"}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/projects/"+created.ID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)
	defer updateResp.Body.Close()

	if updateResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", updateResp.StatusCode)
	}

	var result handlers.ProjectResponse
	json.NewDecoder(updateResp.Body).Decode(&result)

	if result.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got '%s'", result.Name)
	}
	if result.Plan != "enterprise" {
		t.Errorf("Expected plan 'enterprise', got '%s'", result.Plan)
	}
}

func TestDeleteProject(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)
	app.Delete("/api/v1/projects/:id", handler.Delete)
	app.Get("/api/v1/projects/:id", handler.Get)

	// Create
	body := `{"name": "To Delete"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)
	defer createResp.Body.Close()

	var created handlers.ProjectResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Delete
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/projects/"+created.ID, nil)
	deleteResp, _ := app.Test(deleteReq)

	if deleteResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", deleteResp.StatusCode)
	}

	// Verify deleted
	getReq := httptest.NewRequest("GET", "/api/v1/projects/"+created.ID, nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 404 {
		t.Errorf("Expected 404 after deletion, got %d", getResp.StatusCode)
	}
}

func TestDeleteProjectNotFound(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Delete("/api/v1/projects/:id", handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/projects/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateProjectNotFound(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Put("/api/v1/projects/:id", handler.Update)

	updateBody := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/projects/nonexistent", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateProjectInvalidBody(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)
	app.Put("/api/v1/projects/:id", handler.Update)

	// Create
	createBody := `{"name":"Original"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)
	defer createResp.Body.Close()

	var created handlers.ProjectResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Update with invalid body
	updateReq := httptest.NewRequest("PUT", "/api/v1/projects/"+created.ID, bytes.NewBufferString("{invalid"))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)

	if updateResp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", updateResp.StatusCode)
	}
}

func TestUpdateProjectInvalidPlan(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)
	app.Put("/api/v1/projects/:id", handler.Update)

	// Create
	createBody := `{"name":"Original"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)
	defer createResp.Body.Close()

	var created handlers.ProjectResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Update with invalid plan
	updateBody := `{"plan":"ultimate"}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/projects/"+created.ID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)

	if updateResp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", updateResp.StatusCode)
	}
}

func TestUpdateProjectNameOnly(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)
	app.Put("/api/v1/projects/:id", handler.Update)

	// Create with plan pro
	createBody := `{"name":"Original","plan":"pro"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)
	defer createResp.Body.Close()

	var created handlers.ProjectResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Update name only - plan should remain pro
	updateBody := `{"name":"Renamed"}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/projects/"+created.ID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)
	defer updateResp.Body.Close()

	if updateResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", updateResp.StatusCode)
	}

	var result handlers.ProjectResponse
	json.NewDecoder(updateResp.Body).Decode(&result)

	if result.Name != "Renamed" {
		t.Errorf("Expected name 'Renamed', got '%s'", result.Name)
	}
	if result.Plan != "pro" {
		t.Errorf("Expected plan 'pro' (unchanged), got '%s'", result.Plan)
	}
}

func TestDeleteProjectResponseMessage(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)
	app.Delete("/api/v1/projects/:id", handler.Delete)

	createBody := `{"name":"ToDelete"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)
	defer createResp.Body.Close()

	var created handlers.ProjectResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	deleteReq := httptest.NewRequest("DELETE", "/api/v1/projects/"+created.ID, nil)
	deleteResp, _ := app.Test(deleteReq)
	defer deleteResp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(deleteResp.Body).Decode(&result)

	msg, ok := result["message"].(string)
	if !ok || msg == "" {
		t.Error("Expected non-empty message in delete response")
	}
}

func TestCreateProjectInvalidBody(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)

	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateProjectSlugGeneration(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)

	body := `{"name":"My Cool Project"}`
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	var result handlers.ProjectResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Slug != "my-cool-project" {
		t.Errorf("Expected slug 'my-cool-project', got '%s'", result.Slug)
	}
}

func TestCreateProjectNamespacePrefix(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Post("/api/v1/projects", handler.Create)

	body := `{"name":"Test"}`
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	var result handlers.ProjectResponse
	json.NewDecoder(resp.Body).Decode(&result)

	expectedPrefix := "zenith-"
	if len(result.Namespace) < len(expectedPrefix) || result.Namespace[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected namespace to start with 'zenith-', got '%s'", result.Namespace)
	}
}

func TestListProjectsEmpty(t *testing.T) {
	app, handler := setupProjectApp()
	app.Use(injectUser)
	app.Get("/api/v1/projects", handler.List)

	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result dto.ListResponse[handlers.ProjectResponse]
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Items) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(result.Items))
	}
}
