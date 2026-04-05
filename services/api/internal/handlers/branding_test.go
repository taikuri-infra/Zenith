package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupBrandingTest() (*fiber.App, *handlers.BrandingHandler, *memory.MemoryBrandingRepository, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	brandingRepo := memory.NewMemoryBrandingRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewBrandingHandler(brandingRepo, planRepo)
	return app, handler, brandingRepo, planRepo
}

func TestBrandingGetDPA(t *testing.T) {
	app, handler, _, _ := setupBrandingTest()
	app.Get("/api/v1/compliance/dpa", injectUserID("user-1"), handler.GetDPA)

	req := httptest.NewRequest("GET", "/api/v1/compliance/dpa", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != string(entities.DPAUnsigned) {
		t.Errorf("Expected status 'unsigned', got '%v'", result["status"])
	}
}

func TestBrandingSignDPATeamPlan(t *testing.T) {
	app, handler, _, planRepo := setupBrandingTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanTeam)

	app.Post("/api/v1/compliance/dpa/sign", injectUserID("user-1"), handler.SignDPA)

	body := `{"signed_by":"John Doe"}`
	req := httptest.NewRequest("POST", "/api/v1/compliance/dpa/sign", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != string(entities.DPASigned) {
		t.Errorf("Expected status 'signed', got '%v'", result["status"])
	}
}

func TestBrandingSignDPAFreePlanForbidden(t *testing.T) {
	app, handler, _, _ := setupBrandingTest()
	// Default is free plan
	app.Post("/api/v1/compliance/dpa/sign", injectUserID("user-1"), handler.SignDPA)

	body := `{"signed_by":"John Doe"}`
	req := httptest.NewRequest("POST", "/api/v1/compliance/dpa/sign", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestBrandingSignDPAProPlanForbidden(t *testing.T) {
	app, handler, _, planRepo := setupBrandingTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)

	app.Post("/api/v1/compliance/dpa/sign", injectUserID("user-1"), handler.SignDPA)

	body := `{"signed_by":"John Doe"}`
	req := httptest.NewRequest("POST", "/api/v1/compliance/dpa/sign", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestBrandingSignDPANoSignedBy(t *testing.T) {
	app, handler, _, planRepo := setupBrandingTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanTeam)

	app.Post("/api/v1/compliance/dpa/sign", injectUserID("user-1"), handler.SignDPA)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/compliance/dpa/sign", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestBrandingGetBranding(t *testing.T) {
	app, handler, _, _ := setupBrandingTest()
	app.Get("/api/v1/branding", injectUserID("user-1"), handler.GetBranding)

	req := httptest.NewRequest("GET", "/api/v1/branding", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestBrandingUpdateBusinessPlan(t *testing.T) {
	app, handler, _, planRepo := setupBrandingTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)

	app.Put("/api/v1/branding", injectUserID("user-1"), handler.UpdateBranding)

	body := `{"company_name":"Acme Corp","primary_color":"#FF0000"}`
	req := httptest.NewRequest("PUT", "/api/v1/branding", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["company_name"] != "Acme Corp" {
		t.Errorf("Expected 'Acme Corp', got '%v'", result["company_name"])
	}
}

func TestBrandingUpdateFreePlanForbidden(t *testing.T) {
	app, handler, _, _ := setupBrandingTest()
	app.Put("/api/v1/branding", injectUserID("user-1"), handler.UpdateBranding)

	body := `{"company_name":"Acme Corp"}`
	req := httptest.NewRequest("PUT", "/api/v1/branding", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestBrandingSetDashboardDomainEnterprise(t *testing.T) {
	app, handler, _, planRepo := setupBrandingTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanEnterprise)

	app.Post("/api/v1/branding/domain", injectUserID("user-1"), handler.SetDashboardDomain)

	body := `{"domain":"dashboard.acme.com"}`
	req := httptest.NewRequest("POST", "/api/v1/branding/domain", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["dashboard_domain"] != "dashboard.acme.com" {
		t.Errorf("Expected 'dashboard.acme.com', got '%v'", result["dashboard_domain"])
	}
}

func TestBrandingSetDashboardDomainBusinessForbidden(t *testing.T) {
	app, handler, _, planRepo := setupBrandingTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)

	app.Post("/api/v1/branding/domain", injectUserID("user-1"), handler.SetDashboardDomain)

	body := `{"domain":"dashboard.acme.com"}`
	req := httptest.NewRequest("POST", "/api/v1/branding/domain", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestBrandingSetDashboardDomainNoDomain(t *testing.T) {
	app, handler, _, planRepo := setupBrandingTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanEnterprise)

	app.Post("/api/v1/branding/domain", injectUserID("user-1"), handler.SetDashboardDomain)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/branding/domain", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
