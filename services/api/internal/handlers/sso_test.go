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

func setupSSOTest() (*fiber.App, *handlers.SSOHandler, *memory.MemorySSORepository, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	ssoRepo := memory.NewMemorySSORepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewSSOHandler(ssoRepo, planRepo)
	return app, handler, ssoRepo, planRepo
}

func TestSSOConfigureSAMLTeamPlan(t *testing.T) {
	app, handler, _, planRepo := setupSSOTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanTeam)

	app.Post("/api/v1/sso/saml", injectUserID("user-1"), handler.ConfigureSAML)

	body := `{"entity_id":"https://idp.example.com","sso_url":"https://idp.example.com/sso","certificate":"MIID..."}`
	req := httptest.NewRequest("POST", "/api/v1/sso/saml", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["provider"] != string(entities.SSOProviderSAML) {
		t.Errorf("Expected provider 'saml', got '%v'", result["provider"])
	}
}

func TestSSOConfigureSAMLFreePlanForbidden(t *testing.T) {
	app, handler, _, _ := setupSSOTest()
	// Default is free plan
	app.Post("/api/v1/sso/saml", injectUserID("user-1"), handler.ConfigureSAML)

	body := `{"entity_id":"https://idp.example.com","sso_url":"https://idp.example.com/sso"}`
	req := httptest.NewRequest("POST", "/api/v1/sso/saml", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestSSOConfigureSAMLProPlanForbidden(t *testing.T) {
	app, handler, _, planRepo := setupSSOTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)

	app.Post("/api/v1/sso/saml", injectUserID("user-1"), handler.ConfigureSAML)

	body := `{"entity_id":"https://idp.example.com","sso_url":"https://idp.example.com/sso"}`
	req := httptest.NewRequest("POST", "/api/v1/sso/saml", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestSSOConfigureSAMLNoEntityID(t *testing.T) {
	app, handler, _, planRepo := setupSSOTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanTeam)

	app.Post("/api/v1/sso/saml", injectUserID("user-1"), handler.ConfigureSAML)

	body := `{"sso_url":"https://idp.example.com/sso"}`
	req := httptest.NewRequest("POST", "/api/v1/sso/saml", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestSSOConfigureOIDCTeamPlan(t *testing.T) {
	app, handler, _, planRepo := setupSSOTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanTeam)

	app.Post("/api/v1/sso/oidc", injectUserID("user-1"), handler.ConfigureOIDC)

	body := `{"client_id":"my-client","client_secret":"s3cr3t","discovery_url":"https://auth.example.com/.well-known/openid-configuration"}`
	req := httptest.NewRequest("POST", "/api/v1/sso/oidc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["provider"] != string(entities.SSOProviderOIDC) {
		t.Errorf("Expected provider 'oidc', got '%v'", result["provider"])
	}
}

func TestSSOConfigureOIDCFreePlanForbidden(t *testing.T) {
	app, handler, _, _ := setupSSOTest()
	app.Post("/api/v1/sso/oidc", injectUserID("user-1"), handler.ConfigureOIDC)

	body := `{"client_id":"my-client","discovery_url":"https://auth.example.com/"}`
	req := httptest.NewRequest("POST", "/api/v1/sso/oidc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestSSOConfigureOIDCNoClientID(t *testing.T) {
	app, handler, _, planRepo := setupSSOTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanTeam)

	app.Post("/api/v1/sso/oidc", injectUserID("user-1"), handler.ConfigureOIDC)

	body := `{"discovery_url":"https://auth.example.com/"}`
	req := httptest.NewRequest("POST", "/api/v1/sso/oidc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestSSOListConfigs(t *testing.T) {
	app, handler, ssoRepo, _ := setupSSOTest()

	ssoRepo.CreateConfig(nil, "user-1", entities.SSOProviderSAML, &entities.SSOConfig{
		EntityID: "https://idp.example.com",
		SSOURL:   "https://idp.example.com/sso",
	})

	app.Get("/api/v1/sso", injectUserID("user-1"), handler.ListConfigs)

	req := httptest.NewRequest("GET", "/api/v1/sso", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.SSOConfig `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 1 {
		t.Errorf("Expected 1 config, got %d", len(result.Items))
	}
}

func TestSSOListConfigsEmpty(t *testing.T) {
	app, handler, _, _ := setupSSOTest()
	app.Get("/api/v1/sso", injectUserID("user-1"), handler.ListConfigs)

	req := httptest.NewRequest("GET", "/api/v1/sso", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.SSOConfig `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 0 {
		t.Errorf("Expected 0 configs, got %d", len(result.Items))
	}
}

func TestSSODeleteConfig(t *testing.T) {
	app, handler, ssoRepo, _ := setupSSOTest()

	config, _ := ssoRepo.CreateConfig(nil, "user-1", entities.SSOProviderSAML, &entities.SSOConfig{
		EntityID: "https://idp.example.com",
		SSOURL:   "https://idp.example.com/sso",
	})

	app.Delete("/api/v1/sso/:configId", injectUserID("user-1"), handler.DeleteConfig)

	req := httptest.NewRequest("DELETE", "/api/v1/sso/"+config.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 204 {
		t.Fatalf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestSSODeleteConfigNotFound(t *testing.T) {
	app, handler, _, _ := setupSSOTest()
	app.Delete("/api/v1/sso/:configId", injectUserID("user-1"), handler.DeleteConfig)

	req := httptest.NewRequest("DELETE", "/api/v1/sso/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestSSODeleteConfigForbidden(t *testing.T) {
	app, handler, ssoRepo, _ := setupSSOTest()

	config, _ := ssoRepo.CreateConfig(nil, "user-1", entities.SSOProviderSAML, &entities.SSOConfig{
		EntityID: "https://idp.example.com",
		SSOURL:   "https://idp.example.com/sso",
	})

	app.Delete("/api/v1/sso/:configId", injectUserID("user-2"), handler.DeleteConfig)

	req := httptest.NewRequest("DELETE", "/api/v1/sso/"+config.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		// Handler intentionally returns 404 for ownership mismatch
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}
