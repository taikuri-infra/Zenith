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

func setupProjectV2Test() (*fiber.App, *handlers.ProjectHandlerV2, *memory.MemoryProjectRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	projectRepo := memory.NewMemoryProjectRepository()
	appRepo := memory.NewMemoryAppRepository()
	handler := handlers.NewProjectHandlerV2(projectRepo, appRepo, nil)
	return app, handler, projectRepo
}

func TestProjectV2Create(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)

	body := `{"name":"My Project","description":"A test project"}`
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.ProjectV2Response
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Name != "My Project" {
		t.Errorf("Expected name 'My Project', got '%s'", result.Name)
	}
	if result.Slug != "my-project" {
		t.Errorf("Expected slug 'my-project', got '%s'", result.Slug)
	}
	if result.Description != "A test project" {
		t.Errorf("Expected description 'A test project', got '%s'", result.Description)
	}
	if result.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestProjectV2CreateNoName(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)

	body := `{"description":"No name"}`
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestProjectV2CreateNoAuth(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", handler.Create)

	body := `{"name":"My Project"}`
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestProjectV2CreateDuplicate(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)

	body := `{"name":"My Project"}`
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Test(req)

	// Create duplicate
	req2 := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req2)
	if resp.StatusCode != 409 {
		t.Errorf("Expected 409 for duplicate, got %d", resp.StatusCode)
	}
}

func TestProjectV2List(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)
	app.Get("/api/v1/projects", injectUserID("user-1"), handler.List)

	// Create 2 projects
	for _, name := range []string{"Project A", "Project B"} {
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}

	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []handlers.ProjectV2Response `json:"items"`
		Total int                          `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 projects, got %d", result.Total)
	}
}

func TestProjectV2ListEmpty(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Get("/api/v1/projects", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []handlers.ProjectV2Response `json:"items"`
		Total int                          `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0 projects, got %d", result.Total)
	}
}

func TestProjectV2ListNoAuth(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Get("/api/v1/projects", handler.List)

	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestProjectV2Get(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)
	app.Get("/api/v1/projects/:projectId", injectUserID("user-1"), handler.Get)

	body := `{"name":"My Project"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.ProjectV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	getReq := httptest.NewRequest("GET", "/api/v1/projects/"+created.ID, nil)
	getResp, _ := app.Test(getReq)
	if getResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", getResp.StatusCode)
	}

	var result handlers.ProjectV2Response
	json.NewDecoder(getResp.Body).Decode(&result)
	if result.Name != "My Project" {
		t.Errorf("Expected 'My Project', got '%s'", result.Name)
	}
}

func TestProjectV2GetNotFound(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Get("/api/v1/projects/:projectId", injectUserID("user-1"), handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/projects/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestProjectV2GetForbidden(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)
	app.Get("/api/v1/projects/:projectId", injectUserID("user-2"), handler.Get)

	body := `{"name":"My Project"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.ProjectV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	getReq := httptest.NewRequest("GET", "/api/v1/projects/"+created.ID, nil)
	getResp, _ := app.Test(getReq)
	if getResp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", getResp.StatusCode)
	}
}

func TestProjectV2Update(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)
	app.Put("/api/v1/projects/:projectId", injectUserID("user-1"), handler.Update)

	body := `{"name":"My Project"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.ProjectV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	updateBody := `{"name":"Updated Project","description":"New desc"}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/projects/"+created.ID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)
	if updateResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", updateResp.StatusCode)
	}

	var result handlers.ProjectV2Response
	json.NewDecoder(updateResp.Body).Decode(&result)
	if result.Name != "Updated Project" {
		t.Errorf("Expected 'Updated Project', got '%s'", result.Name)
	}
	if result.Description != "New desc" {
		t.Errorf("Expected 'New desc', got '%s'", result.Description)
	}
}

func TestProjectV2UpdateForbidden(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)
	app.Put("/api/v1/projects/:projectId", injectUserID("user-2"), handler.Update)

	body := `{"name":"My Project"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.ProjectV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	updateBody := `{"name":"Hacked"}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/projects/"+created.ID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)
	if updateResp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", updateResp.StatusCode)
	}
}

func TestProjectV2UpdateNotFound(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Put("/api/v1/projects/:projectId", injectUserID("user-1"), handler.Update)

	updateBody := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/projects/nonexistent", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestProjectV2Delete(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)
	app.Delete("/api/v1/projects/:projectId", injectUserID("user-1"), handler.Delete)

	// Create two projects so we can delete one (can't delete the only project)
	for _, name := range []string{"Project A", "Project B"} {
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}

	// List to get IDs (we need to pick one to delete)
	app.Get("/api/v1/projects", injectUserID("user-1"), handler.List)
	listReq := httptest.NewRequest("GET", "/api/v1/projects", nil)
	listResp, _ := app.Test(listReq)

	var listResult struct {
		Items []handlers.ProjectV2Response `json:"items"`
	}
	json.NewDecoder(listResp.Body).Decode(&listResult)

	if len(listResult.Items) < 2 {
		t.Fatal("Expected at least 2 projects for delete test")
	}

	deleteReq := httptest.NewRequest("DELETE", "/api/v1/projects/"+listResult.Items[0].ID, nil)
	deleteResp, _ := app.Test(deleteReq)
	if deleteResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", deleteResp.StatusCode)
	}
}

func TestProjectV2DeleteNotFound(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Delete("/api/v1/projects/:projectId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/projects/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestProjectV2DeleteForbidden(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)
	app.Delete("/api/v1/projects/:projectId", injectUserID("user-2"), handler.Delete)

	body := `{"name":"My Project"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.ProjectV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	deleteReq := httptest.NewRequest("DELETE", "/api/v1/projects/"+created.ID, nil)
	deleteResp, _ := app.Test(deleteReq)
	if deleteResp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", deleteResp.StatusCode)
	}
}

func TestProjectV2CreateInvalidBody(t *testing.T) {
	app, handler, _ := setupProjectV2Test()
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)

	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestProjectV2ListIsolatedByUser(t *testing.T) {
	app, handler, _ := setupProjectV2Test()

	// Create project for user-1
	app.Post("/api/v1/projects", injectUserID("user-1"), handler.Create)
	body1 := `{"name":"User1 Project"}`
	req1 := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body1))
	req1.Header.Set("Content-Type", "application/json")
	app.Test(req1)

	// Create project for user-2 (need separate route registration)
	app2 := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app2.Post("/api/v1/projects", injectUserID("user-2"), handler.Create)
	body2 := `{"name":"User2 Project"}`
	req2 := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewBufferString(body2))
	req2.Header.Set("Content-Type", "application/json")
	app2.Test(req2)

	// List for user-1
	app.Get("/api/v1/projects", injectUserID("user-1"), handler.List)
	listReq := httptest.NewRequest("GET", "/api/v1/projects", nil)
	listResp, _ := app.Test(listReq)

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(listResp.Body).Decode(&result)
	if result.Total != 1 {
		t.Errorf("Expected 1 project for user-1, got %d", result.Total)
	}
}
