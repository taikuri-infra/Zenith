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

func setupDBTest() (*fiber.App, *handlers.DatabaseHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	client := k8s.NewMemoryClient()
	handler := handlers.NewDatabaseHandler(client)
	return app, handler
}

func TestCreateDatabase(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)

	body := `{"name":"maindb","engine":"postgresql","version":"16","storage":"20Gi"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.DatabaseResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Engine != "postgresql" {
		t.Errorf("Expected engine 'postgresql', got '%s'", result.Engine)
	}
	if result.Version != "16" {
		t.Errorf("Expected version '16', got '%s'", result.Version)
	}
	if result.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", result.Port)
	}
	if result.ConnectionString == "" {
		t.Error("Expected non-empty connection string")
	}
	if result.Phase != "Provisioning" {
		t.Errorf("Expected phase 'Provisioning', got '%s'", result.Phase)
	}
}

func TestCreateDatabaseRedis(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)

	body := `{"name":"cache","engine":"redis","version":"7.2","storage":"5Gi"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.DatabaseResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Port != 6379 {
		t.Errorf("Expected port 6379, got %d", result.Port)
	}
}

func TestCreateDatabaseInvalidEngine(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)

	body := `{"name":"db","engine":"oracle","version":"19","storage":"20Gi"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateDatabaseInvalidVersion(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)

	body := `{"name":"db","engine":"postgresql","version":"99","storage":"20Gi"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateDatabaseMissingFields(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)

	tests := []struct {
		name string
		body string
	}{
		{"no name", `{"engine":"postgresql","version":"16","storage":"20Gi"}`},
		{"no engine", `{"name":"db","version":"16","storage":"20Gi"}`},
		{"no version", `{"name":"db","engine":"postgresql","storage":"20Gi"}`},
		{"no storage", `{"name":"db","engine":"postgresql","version":"16"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp, _ := app.Test(req)
			if resp.StatusCode != 400 {
				t.Errorf("Expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestListDatabases(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)
	app.Get("/api/v1/projects/:id/databases", handler.List)

	// Create 2 databases
	for _, body := range []string{
		`{"name":"db1","engine":"postgresql","version":"16","storage":"20Gi"}`,
		`{"name":"db2","engine":"redis","version":"7.2","storage":"5Gi"}`,
	} {
		req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}

	req := httptest.NewRequest("GET", "/api/v1/projects/proj1/databases", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []handlers.DatabaseResponse `json:"items"`
		Total int                         `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Items) != 2 {
		t.Errorf("Expected 2 databases, got %d", len(result.Items))
	}
}

func TestGetDatabase(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)
	app.Get("/api/v1/projects/:id/databases/:name", handler.Get)

	body := `{"name":"maindb","engine":"postgresql","version":"16","storage":"20Gi"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.DatabaseResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	getReq := httptest.NewRequest("GET", "/api/v1/projects/proj1/databases/"+created.ID, nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", getResp.StatusCode)
	}
}

func TestDeleteDatabase(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)
	app.Delete("/api/v1/projects/:id/databases/:name", handler.Delete)

	body := `{"name":"maindb","engine":"postgresql","version":"16","storage":"20Gi"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.DatabaseResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	deleteReq := httptest.NewRequest("DELETE", "/api/v1/projects/proj1/databases/"+created.ID, nil)
	deleteResp, _ := app.Test(deleteReq)

	if deleteResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", deleteResp.StatusCode)
	}
}

func TestCreateBackup(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)
	app.Post("/api/v1/projects/:id/databases/:name/backups", handler.CreateBackup)

	body := `{"name":"maindb","engine":"postgresql","version":"16","storage":"20Gi"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.DatabaseResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	backupReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases/"+created.ID+"/backups", nil)
	backupResp, _ := app.Test(backupReq)

	if backupResp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", backupResp.StatusCode)
	}

	var backup handlers.BackupResponse
	json.NewDecoder(backupResp.Body).Decode(&backup)

	if backup.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", backup.Status)
	}
}
