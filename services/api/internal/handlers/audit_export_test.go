package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupAuditExportTest() (*fiber.App, *handlers.AuditExportHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	adminRepo := memory.NewMemoryAdminRepository()
	handler := handlers.NewAuditExportHandler(adminRepo)
	return app, handler
}

func TestAuditExportCSV(t *testing.T) {
	app, handler := setupAuditExportTest()
	app.Get("/api/v1/admin/audit/export/csv", handler.ExportCSV)

	req := httptest.NewRequest("GET", "/api/v1/admin/audit/export/csv", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Errorf("Expected Content-Type text/csv, got '%s'", contentType)
	}

	disposition := resp.Header.Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment") {
		t.Errorf("Expected attachment Content-Disposition, got '%s'", disposition)
	}
}

func TestAuditExportCSVWithActionFilter(t *testing.T) {
	app, handler := setupAuditExportTest()
	app.Get("/api/v1/admin/audit/export/csv", handler.ExportCSV)

	req := httptest.NewRequest("GET", "/api/v1/admin/audit/export/csv?action=deploy", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestAuditExportJSON(t *testing.T) {
	app, handler := setupAuditExportTest()
	app.Get("/api/v1/admin/audit/export/json", handler.ExportJSON)

	req := httptest.NewRequest("GET", "/api/v1/admin/audit/export/json", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []interface{} `json:"items"`
		Total int           `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	// MemoryAdminRepository is pre-seeded with audit entries
	if result.Total == 0 {
		t.Error("Expected non-zero total from pre-seeded admin repo")
	}
}

func TestAuditExportJSONWithActionFilter(t *testing.T) {
	app, handler := setupAuditExportTest()
	app.Get("/api/v1/admin/audit/export/json", handler.ExportJSON)

	// Use a filter that won't match any pre-seeded entries
	req := httptest.NewRequest("GET", "/api/v1/admin/audit/export/json?action=nonexistent_action", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0 results for nonexistent action, got %d", result.Total)
	}
}

func TestAuditExportCSVWithLimit(t *testing.T) {
	app, handler := setupAuditExportTest()
	app.Get("/api/v1/admin/audit/export/csv", handler.ExportCSV)

	req := httptest.NewRequest("GET", "/api/v1/admin/audit/export/csv?limit=5", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}
