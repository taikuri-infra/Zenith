package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/gofiber/fiber/v2"
)

func setupAppTest() (*fiber.App, *handlers.AppHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	client := k8sclient.NewMemoryClient()
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

func TestUpdateApp(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Put("/api/v1/projects/:id/apps/:name", handler.Update)

	// Create an app first
	body := `{"name":"web","image":"nginx:latest","port":3000}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Update the app
	updateBody := `{"image":"nginx:1.25","domain":"web.example.com"}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/projects/proj1/apps/"+created.ID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)

	if updateResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", updateResp.StatusCode)
	}

	var result handlers.AppResponse
	json.NewDecoder(updateResp.Body).Decode(&result)

	if result.Image != "nginx:1.25" {
		t.Errorf("Expected image 'nginx:1.25', got '%s'", result.Image)
	}
	if result.Domain != "web.example.com" {
		t.Errorf("Expected domain 'web.example.com', got '%s'", result.Domain)
	}
}

func TestUpdateAppReplicas(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Put("/api/v1/projects/:id/apps/:name", handler.Update)

	body := `{"name":"web","image":"nginx:latest"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Update replicas
	updateBody := `{"replicas":5}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/projects/proj1/apps/"+created.ID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)

	if updateResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", updateResp.StatusCode)
	}

	var result handlers.AppResponse
	json.NewDecoder(updateResp.Body).Decode(&result)

	if result.Replicas != 5 {
		t.Errorf("Expected replicas 5, got %d", result.Replicas)
	}
}

func TestUpdateAppEnv(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Put("/api/v1/projects/:id/apps/:name", handler.Update)

	body := `{"name":"web","image":"nginx:latest","env":{"PORT":"3000"}}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	updateBody := `{"env":{"PORT":"4000","NODE_ENV":"production"}}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/projects/proj1/apps/"+created.ID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)

	if updateResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", updateResp.StatusCode)
	}
}

func TestUpdateAppNotFound(t *testing.T) {
	app, handler := setupAppTest()
	app.Put("/api/v1/projects/:id/apps/:name", handler.Update)

	updateBody := `{"image":"nginx:1.25"}`
	req := httptest.NewRequest("PUT", "/api/v1/projects/proj1/apps/nonexistent", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateAppInvalidBody(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Put("/api/v1/projects/:id/apps/:name", handler.Update)

	body := `{"name":"web","image":"nginx:latest"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Send invalid JSON
	updateReq := httptest.NewRequest("PUT", "/api/v1/projects/proj1/apps/"+created.ID, bytes.NewBufferString("{invalid"))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)

	if updateResp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", updateResp.StatusCode)
	}
}

func TestDeleteAppNotFound(t *testing.T) {
	app, handler := setupAppTest()
	app.Delete("/api/v1/projects/:id/apps/:name", handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/projects/proj1/apps/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteAppThenGetReturns404(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Delete("/api/v1/projects/:id/apps/:name", handler.Delete)
	app.Get("/api/v1/projects/:id/apps/:name", handler.Get)

	body := `{"name":"web","image":"nginx:latest"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Delete
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/projects/proj1/apps/"+created.ID, nil)
	app.Test(deleteReq)

	// Get should return 404
	getReq := httptest.NewRequest("GET", "/api/v1/projects/proj1/apps/"+created.ID, nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 404 {
		t.Errorf("Expected 404 after deletion, got %d", getResp.StatusCode)
	}
}

func TestRedeployAppNotFound(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps/:name/redeploy", handler.Redeploy)

	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps/nonexistent/redeploy", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestListAppsEmpty(t *testing.T) {
	app, handler := setupAppTest()
	app.Get("/api/v1/projects/:id/apps", handler.List)

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

	if len(result.Items) != 0 {
		t.Errorf("Expected 0 apps, got %d", len(result.Items))
	}
	if result.Total != 0 {
		t.Errorf("Expected total 0, got %d", result.Total)
	}
}

func TestListAppsIsolatedByProject(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Get("/api/v1/projects/:id/apps", handler.List)

	// Create app in proj1
	body := `{"name":"web","image":"nginx:latest"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Test(req)

	// Create app in proj2
	body2 := `{"name":"api","image":"node:latest"}`
	req2 := httptest.NewRequest("POST", "/api/v1/projects/proj2/apps", bytes.NewBufferString(body2))
	req2.Header.Set("Content-Type", "application/json")
	app.Test(req2)

	// List proj1 apps
	listReq := httptest.NewRequest("GET", "/api/v1/projects/proj1/apps", nil)
	listResp, _ := app.Test(listReq)

	var result struct {
		Items []handlers.AppResponse `json:"items"`
		Total int                    `json:"total"`
	}
	json.NewDecoder(listResp.Body).Decode(&result)

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 app for proj1, got %d", len(result.Items))
	}
}

func TestGetAppResponseFields(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)
	app.Get("/api/v1/projects/:id/apps/:name", handler.Get)

	body := `{"name":"web","image":"nginx:latest","port":3000,"domain":"web.test.com"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	getReq := httptest.NewRequest("GET", "/api/v1/projects/proj1/apps/"+created.ID, nil)
	getResp, _ := app.Test(getReq)

	var result handlers.AppResponse
	json.NewDecoder(getResp.Body).Decode(&result)

	if result.ProjectID != "proj1" {
		t.Errorf("Expected project_id 'proj1', got '%s'", result.ProjectID)
	}
	if result.Image != "nginx:latest" {
		t.Errorf("Expected image 'nginx:latest', got '%s'", result.Image)
	}
	if result.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", result.Port)
	}
	if result.Domain != "web.test.com" {
		t.Errorf("Expected domain 'web.test.com', got '%s'", result.Domain)
	}
	// After Get, phase should be Running (from CRD conversion)
	if result.Phase != "Running" {
		t.Errorf("Expected phase 'Running', got '%s'", result.Phase)
	}
}

func TestCreateAppWithCustomReplicas(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)

	body := `{"name":"web","image":"nginx:latest","replicas":3}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.AppResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Replicas != 3 {
		t.Errorf("Expected replicas 3, got %d", result.Replicas)
	}
}

func TestCreateAppInvalidBody(t *testing.T) {
	app, handler := setupAppTest()
	app.Post("/api/v1/projects/:id/apps", handler.Create)

	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/apps", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestRedeployAppSetsAnnotation(t *testing.T) {
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

	var result map[string]interface{}
	json.NewDecoder(redeployResp.Body).Decode(&result)

	if result["message"] != "redeploy triggered" {
		t.Errorf("Expected message 'redeploy triggered', got '%v'", result["message"])
	}
}

func TestDeleteAppResponseMessage(t *testing.T) {
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

	var result map[string]interface{}
	json.NewDecoder(deleteResp.Body).Decode(&result)

	if result["message"] != "app scheduled for deletion" {
		t.Errorf("Expected message 'app scheduled for deletion', got '%v'", result["message"])
	}
}
