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

func setupStorageTest() (*fiber.App, *handlers.StorageHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	client := k8sclient.NewMemoryClient()
	handler := handlers.NewStorageHandler(client)
	return app, handler
}

func TestCreateStorageBucket(t *testing.T) {
	app, handler := setupStorageTest()
	app.Post("/api/v1/projects/:id/storage", handler.Create)

	body := `{"name":"assets","access":"private","versioning":true}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.StorageResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Name != "assets" {
		t.Errorf("Expected name 'assets', got '%s'", result.Name)
	}
	if result.Access != "private" {
		t.Errorf("Expected access 'private', got '%s'", result.Access)
	}
	if !result.Versioning {
		t.Error("Expected versioning to be true")
	}
	if result.Phase != "Creating" {
		t.Errorf("Expected phase 'Creating', got '%s'", result.Phase)
	}
	if result.ProjectID != "proj1" {
		t.Errorf("Expected project_id 'proj1', got '%s'", result.ProjectID)
	}
}

func TestCreateStorageBucketDefaults(t *testing.T) {
	app, handler := setupStorageTest()
	app.Post("/api/v1/projects/:id/storage", handler.Create)

	body := `{"name":"media"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.StorageResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Access != "private" {
		t.Errorf("Expected default access 'private', got '%s'", result.Access)
	}
	if result.Region != "fsn1" {
		t.Errorf("Expected default region 'fsn1', got '%s'", result.Region)
	}
}

func TestCreateStorageBucketPublicRead(t *testing.T) {
	app, handler := setupStorageTest()
	app.Post("/api/v1/projects/:id/storage", handler.Create)

	body := `{"name":"cdn","access":"public-read"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.StorageResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Access != "public-read" {
		t.Errorf("Expected access 'public-read', got '%s'", result.Access)
	}
}

func TestCreateStorageBucketInvalidAccess(t *testing.T) {
	app, handler := setupStorageTest()
	app.Post("/api/v1/projects/:id/storage", handler.Create)

	body := `{"name":"bad","access":"public-write"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateStorageBucketNoName(t *testing.T) {
	app, handler := setupStorageTest()
	app.Post("/api/v1/projects/:id/storage", handler.Create)

	body := `{"access":"private"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateStorageBucketInvalidBody(t *testing.T) {
	app, handler := setupStorageTest()
	app.Post("/api/v1/projects/:id/storage", handler.Create)

	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/storage", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestListStorageBuckets(t *testing.T) {
	app, handler := setupStorageTest()
	app.Post("/api/v1/projects/:id/storage", handler.Create)
	app.Get("/api/v1/projects/:id/storage", handler.List)

	// Create 2 buckets
	for _, name := range []string{"assets", "backups"} {
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest("POST", "/api/v1/projects/proj1/storage", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}

	req := httptest.NewRequest("GET", "/api/v1/projects/proj1/storage", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []handlers.StorageResponse `json:"items"`
		Total int                        `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Items) != 2 {
		t.Errorf("Expected 2 buckets, got %d", len(result.Items))
	}
	if result.Total != 2 {
		t.Errorf("Expected total 2, got %d", result.Total)
	}
}

func TestListStorageBucketsEmpty(t *testing.T) {
	app, handler := setupStorageTest()
	app.Get("/api/v1/projects/:id/storage", handler.List)

	req := httptest.NewRequest("GET", "/api/v1/projects/proj1/storage", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []handlers.StorageResponse `json:"items"`
		Total int                        `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Items) != 0 {
		t.Errorf("Expected 0 buckets, got %d", len(result.Items))
	}
	if result.Total != 0 {
		t.Errorf("Expected total 0, got %d", result.Total)
	}
}

func TestGetStorageBucket(t *testing.T) {
	app, handler := setupStorageTest()
	app.Post("/api/v1/projects/:id/storage", handler.Create)
	app.Get("/api/v1/projects/:id/storage/:name", handler.Get)

	body := `{"name":"assets","access":"public-read","region":"nbg1"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/storage", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.StorageResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	getReq := httptest.NewRequest("GET", "/api/v1/projects/proj1/storage/"+created.ID, nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", getResp.StatusCode)
	}

	var result handlers.StorageResponse
	json.NewDecoder(getResp.Body).Decode(&result)

	if result.Name != "assets" {
		t.Errorf("Expected name 'assets', got '%s'", result.Name)
	}
	if result.Access != "public-read" {
		t.Errorf("Expected access 'public-read', got '%s'", result.Access)
	}
	if result.Region != "nbg1" {
		t.Errorf("Expected region 'nbg1', got '%s'", result.Region)
	}
	// After Get, phase should be Ready (from CRD conversion)
	if result.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", result.Phase)
	}
}

func TestGetStorageBucketNotFound(t *testing.T) {
	app, handler := setupStorageTest()
	app.Get("/api/v1/projects/:id/storage/:name", handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/projects/proj1/storage/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteStorageBucket(t *testing.T) {
	app, handler := setupStorageTest()
	app.Post("/api/v1/projects/:id/storage", handler.Create)
	app.Delete("/api/v1/projects/:id/storage/:name", handler.Delete)
	app.Get("/api/v1/projects/:id/storage/:name", handler.Get)

	body := `{"name":"to-delete"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/storage", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.StorageResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Delete
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/projects/proj1/storage/"+created.ID, nil)
	deleteResp, _ := app.Test(deleteReq)

	if deleteResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", deleteResp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(deleteResp.Body).Decode(&result)

	if result["message"] != "storage bucket scheduled for deletion" {
		t.Errorf("Expected deletion message, got '%v'", result["message"])
	}

	// Verify deleted
	getReq := httptest.NewRequest("GET", "/api/v1/projects/proj1/storage/"+created.ID, nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 404 {
		t.Errorf("Expected 404 after deletion, got %d", getResp.StatusCode)
	}
}

func TestDeleteStorageBucketNotFound(t *testing.T) {
	app, handler := setupStorageTest()
	app.Delete("/api/v1/projects/:id/storage/:name", handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/projects/proj1/storage/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestListStorageBucketsIsolatedByProject(t *testing.T) {
	app, handler := setupStorageTest()
	app.Post("/api/v1/projects/:id/storage", handler.Create)
	app.Get("/api/v1/projects/:id/storage", handler.List)

	// Create bucket in proj1
	body := `{"name":"assets"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/storage", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Test(req)

	// Create bucket in proj2
	body2 := `{"name":"other"}`
	req2 := httptest.NewRequest("POST", "/api/v1/projects/proj2/storage", bytes.NewBufferString(body2))
	req2.Header.Set("Content-Type", "application/json")
	app.Test(req2)

	// List proj1 only
	listReq := httptest.NewRequest("GET", "/api/v1/projects/proj1/storage", nil)
	listResp, _ := app.Test(listReq)

	var result struct {
		Items []handlers.StorageResponse `json:"items"`
		Total int                        `json:"total"`
	}
	json.NewDecoder(listResp.Body).Decode(&result)

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 bucket for proj1, got %d", len(result.Items))
	}
}
