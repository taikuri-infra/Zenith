package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupDomainTest() (*fiber.App, *handlers.DomainHandler, *memory.MemoryDomainRepository, string) {
	fiberApp := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	domainRepo := memory.NewMemoryDomainRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	// Pro plan for custom domains
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)
	handler := handlers.NewDomainHandler(domainRepo, appRepo, planRepo)

	app, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "test-app",
		RepoURL: "https://github.com/user/repo",
	})

	return fiberApp, handler, domainRepo, app.ID
}

func TestDomainAdd(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	body := `{"domain":"example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", result.Domain)
	}
	if result.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestDomainAddNoDomain(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestDomainAddAppNotFound(t *testing.T) {
	fiberApp, handler, _, _ := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	body := `{"domain":"example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/nonexistent/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDomainAddForbidden(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-2"), handler.Add)

	body := `{"domain":"example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestDomainAddDuplicate(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	body := `{"domain":"example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	fiberApp.Test(req)

	// Duplicate
	req2 := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	resp, _ := fiberApp.Test(req2)
	if resp.StatusCode != 409 {
		t.Errorf("Expected 409, got %d", resp.StatusCode)
	}
}

func TestDomainList(t *testing.T) {
	fiberApp, handler, domainRepo, appID := setupDomainTest()

	domainRepo.AddDomain(nil, appID, "user-1", "a.example.com")
	domainRepo.AddDomain(nil, appID, "user-1", "b.example.com")

	fiberApp.Get("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/domains", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(result))
	}
}

func TestDomainListEmpty(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Get("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/domains", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestDomainDelete(t *testing.T) {
	fiberApp, handler, domainRepo, appID := setupDomainTest()

	domain, _ := domainRepo.AddDomain(nil, appID, "user-1", "example.com")

	fiberApp.Delete("/api/v1/apps/:appId/domains/:domainId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/domains/"+domain.ID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestDomainDeleteNotFound(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Delete("/api/v1/apps/:appId/domains/:domainId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/domains/nonexistent", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDomainDeleteForbidden(t *testing.T) {
	fiberApp, handler, domainRepo, appID := setupDomainTest()

	domain, _ := domainRepo.AddDomain(nil, appID, "user-1", "example.com")

	fiberApp.Delete("/api/v1/apps/:appId/domains/:domainId", injectUserID("user-2"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/domains/"+domain.ID, nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestDomainListByUser(t *testing.T) {
	fiberApp, handler, domainRepo, appID := setupDomainTest()

	domainRepo.AddDomain(nil, appID, "user-1", "a.example.com")
	domainRepo.AddDomain(nil, appID, "user-1", "b.example.com")

	fiberApp.Get("/api/v1/domains", injectUserID("user-1"), handler.ListByUser)

	req := httptest.NewRequest("GET", "/api/v1/domains", nil)
	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(result))
	}
}
