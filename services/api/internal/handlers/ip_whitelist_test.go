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

func setupIPWhitelistTest() (*fiber.App, *handlers.IPWhitelistHandler, *memory.MemoryIPWhitelistRepository, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	ipRepo := memory.NewMemoryIPWhitelistRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewIPWhitelistHandler(ipRepo, planRepo)
	return app, handler, ipRepo, planRepo
}

func TestIPWhitelistAddBusinessPlan(t *testing.T) {
	app, handler, _, planRepo := setupIPWhitelistTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)

	app.Post("/api/v1/ip-whitelist", injectUserID("user-1"), handler.Add)

	body := `{"cidr":"192.168.1.0/24","description":"Office network"}`
	req := httptest.NewRequest("POST", "/api/v1/ip-whitelist", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["cidr"] != "192.168.1.0/24" {
		t.Errorf("Expected cidr '192.168.1.0/24', got '%v'", result["cidr"])
	}
}

func TestIPWhitelistAddFreePlanForbidden(t *testing.T) {
	app, handler, _, _ := setupIPWhitelistTest()
	// Default is free plan
	app.Post("/api/v1/ip-whitelist", injectUserID("user-1"), handler.Add)

	body := `{"cidr":"192.168.1.0/24"}`
	req := httptest.NewRequest("POST", "/api/v1/ip-whitelist", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestIPWhitelistAddProPlanForbidden(t *testing.T) {
	app, handler, _, planRepo := setupIPWhitelistTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)

	app.Post("/api/v1/ip-whitelist", injectUserID("user-1"), handler.Add)

	body := `{"cidr":"192.168.1.0/24"}`
	req := httptest.NewRequest("POST", "/api/v1/ip-whitelist", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestIPWhitelistAddNoCIDR(t *testing.T) {
	app, handler, _, planRepo := setupIPWhitelistTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)

	app.Post("/api/v1/ip-whitelist", injectUserID("user-1"), handler.Add)

	body := `{"description":"Office"}`
	req := httptest.NewRequest("POST", "/api/v1/ip-whitelist", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestIPWhitelistList(t *testing.T) {
	app, handler, ipRepo, _ := setupIPWhitelistTest()

	ipRepo.AddEntry(nil, "user-1", "10.0.0.0/8", "VPN")
	ipRepo.AddEntry(nil, "user-1", "192.168.1.0/24", "Office")

	app.Get("/api/v1/ip-whitelist", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/ip-whitelist", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.IPWhitelistEntry `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(result.Items))
	}
}

func TestIPWhitelistListEmpty(t *testing.T) {
	app, handler, _, _ := setupIPWhitelistTest()
	app.Get("/api/v1/ip-whitelist", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/ip-whitelist", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.IPWhitelistEntry `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(result.Items))
	}
}

func TestIPWhitelistDelete(t *testing.T) {
	app, handler, ipRepo, _ := setupIPWhitelistTest()

	entry, _ := ipRepo.AddEntry(nil, "user-1", "10.0.0.0/8", "VPN")

	app.Delete("/api/v1/ip-whitelist/:entryId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/ip-whitelist/"+entry.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 204 {
		t.Fatalf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestIPWhitelistDeleteNotFound(t *testing.T) {
	app, handler, _, _ := setupIPWhitelistTest()
	app.Delete("/api/v1/ip-whitelist/:entryId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/ip-whitelist/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestIPWhitelistDeleteForbidden(t *testing.T) {
	app, handler, ipRepo, _ := setupIPWhitelistTest()

	entry, _ := ipRepo.AddEntry(nil, "user-1", "10.0.0.0/8", "VPN")

	app.Delete("/api/v1/ip-whitelist/:entryId", injectUserID("user-2"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/ip-whitelist/"+entry.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		// The handler returns 404 for ownership mismatch (intentionally hides existence)
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}
