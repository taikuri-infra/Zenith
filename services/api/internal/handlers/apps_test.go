package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/gofiber/fiber/v2"
)

func setupAppTest() (*fiber.App, *handlers.AppHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	client := k8s.NewMemoryClient()
	handler := handlers.NewAppHandler(client)
	return app, handler
}

func TestCreateApp(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)

	body := `{"name":"web","image":"nginx:latest","port":3000}`
	req := httptest.NewRequest("POST", "/api/v1/projects/myproj/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.AppResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Name != "web" {
		t.Errorf("Expected name 'web', got '%s'", result.Name)
	}
	if result.Image != "nginx:latest" {
		t.Errorf("Expected image 'nginx:latest', got '%s'", result.Image)
	}
	if result.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", result.Port)
	}
	if result.Phase != "Pending" {
		t.Errorf("Expected phase 'Pending', got '%s'", result.Phase)
	}
}

func TestCreateAppDefaults(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)

	body := `{"name":"api","image":"myapp:v1"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.AppResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Replicas != 1 {
		t.Errorf("Expected default replicas 1, got %d", result.Replicas)
	}
	if result.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", result.Port)
	}
}

func TestCreateAppNoName(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)

	body := `{"image":"nginx:latest"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateAppNoImage(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)

	body := `{"name":"web"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestListApps(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Get("/api/v1/projects/:id/apps", handler.List)

	// Create 2 apps
	for _, name := range []string{"web", "api"} {
		body := `{"name":"` + name + `","image":"img:v1"}`
		req := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}

	req := httptest.NewRequest("GET", "/api/v1/projects/proj1/apps", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []handlers.AppResponse `json:"items"`
		Total int                    `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Items) != 2 {
		t.Errorf("Expected 2 apps, got %d", len(result.Items))
	}
}

func TestGetApp(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Get("/api/v1/projects/:id/apps/:name", handler.Get)

	body := `{"name":"web","image":"nginx:latest"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	getReq := httptest.NewRequest("GET", "/api/v1/projects/proj1/apps/"+created.ID, nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", getResp.StatusCode)
	}
}

func TestGetAppNotFound(t *testing.T) {
	app, handler := setupAppTest()
	app.Get("/api/v1/projects/:id/apps/:name", handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/projects/proj1/apps/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteApp(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Delete("/api/v1/projects/:id/apps/:name", handler.Delete)

	body := `{"name":"web","image":"nginx:latest"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	deleteReq := httptest.NewRequest("DELETE", "/api/v1/projects/proj1/apps/"+created.ID, nil)
	deleteResp, _ := app.Test(deleteReq)

	if deleteResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", deleteResp.StatusCode)
	}
}

func TestRedeployApp(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Post("/api/v1/projects/:id/apps/:name/redeploy", handler.Redeploy)

	body := `{"name":"web","image":"nginx:latest"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	redeployReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps/"+created.ID+"/redeploy", nil)
	redeployResp, _ := app.Test(redeployReq)

	if redeployResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", redeployResp.StatusCode)
	}
}
