package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupAddOnTest() (*fiber.App, *handlers.AddOnHandler, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewAddOnHandler(planRepo)
	return app, handler, planRepo
}

func TestAddOnListCatalogFreeUser(t *testing.T) {
	app, handler, _ := setupAddOnTest()
	app.Get("/api/v1/addons", injectUserID("user-1"), handler.ListCatalog)

	req := httptest.NewRequest("GET", "/api/v1/addons", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) == 0 {
		t.Fatal("Expected non-empty add-on catalog")
	}

	// Free users should have some add-ons marked unavailable
	hasUnavailable := false
	for _, addon := range result {
		if addon["available"] == false {
			hasUnavailable = true
			break
		}
	}
	if !hasUnavailable {
		t.Error("Expected some add-ons to be unavailable for free users")
	}
}

func TestAddOnListCatalogProUser(t *testing.T) {
	app, handler, planRepo := setupAddOnTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)
	app.Get("/api/v1/addons", injectUserID("user-1"), handler.ListCatalog)

	req := httptest.NewRequest("GET", "/api/v1/addons", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) == 0 {
		t.Fatal("Expected non-empty add-on catalog")
	}

	// Pro users should have more available add-ons than free users
	availableCount := 0
	for _, addon := range result {
		if addon["available"] == true {
			availableCount++
		}
	}
	if availableCount == 0 {
		t.Error("Expected at least some available add-ons for pro users")
	}
}

func TestAddOnListCatalogEnterpriseUser(t *testing.T) {
	app, handler, planRepo := setupAddOnTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanEnterprise)
	app.Get("/api/v1/addons", injectUserID("user-1"), handler.ListCatalog)

	req := httptest.NewRequest("GET", "/api/v1/addons", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Enterprise users should have all add-ons available
	for _, addon := range result {
		if addon["available"] != true {
			t.Errorf("Expected all add-ons available for enterprise, but '%v' is not", addon["name"])
		}
	}
}

func TestAddOnGetByID(t *testing.T) {
	app, handler, _ := setupAddOnTest()
	app.Get("/api/v1/addons/:addonId", handler.GetAddOn)

	req := httptest.NewRequest("GET", "/api/v1/addons/gold-support", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["id"] != "gold-support" {
		t.Errorf("Expected id 'gold-support', got '%v'", result["id"])
	}
}

func TestAddOnGetNotFound(t *testing.T) {
	app, handler, _ := setupAddOnTest()
	app.Get("/api/v1/addons/:addonId", handler.GetAddOn)

	req := httptest.NewRequest("GET", "/api/v1/addons/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}
