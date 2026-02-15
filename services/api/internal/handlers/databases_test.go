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

func TestGetDatabaseNotFound(t *testing.T) {
	app, handler := setupDBTest()
	app.Get("/api/v1/projects/:id/databases/:name", handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/projects/proj1/databases/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteDatabaseNotFound(t *testing.T) {
	app, handler := setupDBTest()
	app.Delete("/api/v1/projects/:id/databases/:name", handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/projects/proj1/databases/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteDatabaseResponseMessage(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)
	app.Delete("/api/v1/projects/:id/databases/:name", handler.Delete)

	body := `{"name":"todelete","engine":"redis","version":"7.2","storage":"5Gi"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.DatabaseResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	deleteReq := httptest.NewRequest("DELETE", "/api/v1/projects/proj1/databases/"+created.ID, nil)
	deleteResp, _ := app.Test(deleteReq)

	var result map[string]interface{}
	json.NewDecoder(deleteResp.Body).Decode(&result)

	if result["message"] != "database scheduled for deletion" {
		t.Errorf("Expected message 'database scheduled for deletion', got '%v'", result["message"])
	}
}

func TestDeleteDatabaseThenGetReturns404(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)
	app.Delete("/api/v1/projects/:id/databases/:name", handler.Delete)
	app.Get("/api/v1/projects/:id/databases/:name", handler.Get)

	body := `{"name":"maindb","engine":"postgresql","version":"16","storage":"20Gi"}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.DatabaseResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Delete
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/projects/proj1/databases/"+created.ID, nil)
	app.Test(deleteReq)

	// Verify 404
	getReq := httptest.NewRequest("GET", "/api/v1/projects/proj1/databases/"+created.ID, nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 404 {
		t.Errorf("Expected 404 after deletion, got %d", getResp.StatusCode)
	}
}

func TestListDatabasesEmpty(t *testing.T) {
	app, handler := setupDBTest()
	app.Get("/api/v1/projects/:id/databases", handler.List)

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

	if len(result.Items) != 0 {
		t.Errorf("Expected 0 databases, got %d", len(result.Items))
	}
	if result.Total != 0 {
		t.Errorf("Expected total 0, got %d", result.Total)
	}
}

func TestListBackupsEmpty(t *testing.T) {
	app, handler := setupDBTest()
	app.Get("/api/v1/projects/:id/databases/:name/backups", handler.ListBackups)

	req := httptest.NewRequest("GET", "/api/v1/projects/proj1/databases/some-db/backups", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []handlers.BackupResponse `json:"items"`
		Total int                       `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Items) != 0 {
		t.Errorf("Expected 0 backups, got %d", len(result.Items))
	}
	if result.Total != 0 {
		t.Errorf("Expected total 0, got %d", result.Total)
	}
}

func TestCreateBackupNotFound(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases/:name/backups", handler.CreateBackup)

	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases/nonexistent/backups", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestCreateDatabaseInvalidBody(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)

	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateDatabaseMySQLPort(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)

	body := `{"name":"mydb","engine":"mysql","version":"8.0","storage":"10Gi"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.DatabaseResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Port != 3306 {
		t.Errorf("Expected MySQL port 3306, got %d", result.Port)
	}
	if result.Engine != "mysql" {
		t.Errorf("Expected engine 'mysql', got '%s'", result.Engine)
	}
}

func TestCreateDatabaseMongoDBPort(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)

	body := `{"name":"docs","engine":"mongodb","version":"7.0","storage":"15Gi"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.DatabaseResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Port != 27017 {
		t.Errorf("Expected MongoDB port 27017, got %d", result.Port)
	}
}

func TestCreateDatabaseWithReplicas(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)

	body := `{"name":"ha-db","engine":"postgresql","version":"16","storage":"50Gi","replicas":3}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.DatabaseResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Replicas != 3 {
		t.Errorf("Expected replicas 3, got %d", result.Replicas)
	}
}

func TestGetDatabaseResponseFields(t *testing.T) {
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

	var result handlers.DatabaseResponse
	json.NewDecoder(getResp.Body).Decode(&result)

	if result.Name != "maindb" {
		t.Errorf("Expected name 'maindb', got '%s'", result.Name)
	}
	if result.Engine != "postgresql" {
		t.Errorf("Expected engine 'postgresql', got '%s'", result.Engine)
	}
	if result.Version != "16" {
		t.Errorf("Expected version '16', got '%s'", result.Version)
	}
	if result.Storage != "20Gi" {
		t.Errorf("Expected storage '20Gi', got '%s'", result.Storage)
	}
	if result.Host == "" {
		t.Error("Expected non-empty host")
	}
	if result.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", result.Port)
	}
	// After Get, phase should be Ready (from CRD conversion)
	if result.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", result.Phase)
	}
}

func TestCreateBackupResponseFields(t *testing.T) {
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

	var backup handlers.BackupResponse
	json.NewDecoder(backupResp.Body).Decode(&backup)

	if backup.ID == "" {
		t.Error("Expected non-empty backup ID")
	}
	if backup.Database != created.ID {
		t.Errorf("Expected database '%s', got '%s'", created.ID, backup.Database)
	}
	if backup.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", backup.Status)
	}
}

func TestListDatabasesIsolatedByProject(t *testing.T) {
	app, handler := setupDBTest()
	app.Post("/api/v1/projects/:id/databases", handler.Create)
	app.Get("/api/v1/projects/:id/databases", handler.List)

	// Create db in proj1
	body := `{"name":"db1","engine":"postgresql","version":"16","storage":"20Gi"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/proj1/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Test(req)

	// Create db in proj2
	body2 := `{"name":"db2","engine":"redis","version":"7.2","storage":"5Gi"}`
	req2 := httptest.NewRequest("POST", "/api/v1/projects/proj2/databases", bytes.NewBufferString(body2))
	req2.Header.Set("Content-Type", "application/json")
	app.Test(req2)

	// List proj1 only
	listReq := httptest.NewRequest("GET", "/api/v1/projects/proj1/databases", nil)
	listResp, _ := app.Test(listReq)

	var result struct {
		Items []handlers.DatabaseResponse `json:"items"`
		Total int                         `json:"total"`
	}
	json.NewDecoder(listResp.Body).Decode(&result)

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 database for proj1, got %d", len(result.Items))
	}
}
