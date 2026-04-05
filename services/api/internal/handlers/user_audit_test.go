package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupUserAuditTest() (*fiber.App, *handlers.UserAuditHandler, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	adminRepo := memory.NewMemoryAdminRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewUserAuditHandler(adminRepo, planRepo)
	return app, handler, planRepo
}

func TestUserAuditListBusinessPlan(t *testing.T) {
	app, handler, planRepo := setupUserAuditTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)

	app.Get("/api/v1/audit", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/audit", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []interface{} `json:"items"`
		Total int           `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	// Should return only entries matching user-1 as actor (likely 0 from pre-seeded data)
	// The important thing is it doesn't error
}

func TestUserAuditListFreePlanForbidden(t *testing.T) {
	app, handler, _ := setupUserAuditTest()
	// Default is free plan

	app.Get("/api/v1/audit", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/audit", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestUserAuditListProPlanForbidden(t *testing.T) {
	app, handler, planRepo := setupUserAuditTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)

	app.Get("/api/v1/audit", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/audit", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestUserAuditListEnterprisePlan(t *testing.T) {
	app, handler, planRepo := setupUserAuditTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanEnterprise)

	app.Get("/api/v1/audit", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/audit?limit=10&offset=0", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestUserAuditListWithActionFilter(t *testing.T) {
	app, handler, planRepo := setupUserAuditTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)

	app.Get("/api/v1/audit", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/audit?action=deploy", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestUserAuditExportCSV(t *testing.T) {
	app, handler, planRepo := setupUserAuditTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)

	app.Get("/api/v1/audit/export/csv", injectUserID("user-1"), handler.ExportCSV)

	req := httptest.NewRequest("GET", "/api/v1/audit/export/csv", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Errorf("Expected Content-Type text/csv, got '%s'", contentType)
	}
}

func TestUserAuditExportCSVForbidden(t *testing.T) {
	app, handler, _ := setupUserAuditTest()

	app.Get("/api/v1/audit/export/csv", injectUserID("user-1"), handler.ExportCSV)

	req := httptest.NewRequest("GET", "/api/v1/audit/export/csv", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestUserAuditExportJSON(t *testing.T) {
	app, handler, planRepo := setupUserAuditTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)

	app.Get("/api/v1/audit/export/json", injectUserID("user-1"), handler.ExportJSON)

	req := httptest.NewRequest("GET", "/api/v1/audit/export/json", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []interface{} `json:"items"`
		Total int           `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
}

func TestUserAuditExportJSONForbidden(t *testing.T) {
	app, handler, _ := setupUserAuditTest()

	app.Get("/api/v1/audit/export/json", injectUserID("user-1"), handler.ExportJSON)

	req := httptest.NewRequest("GET", "/api/v1/audit/export/json", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}
