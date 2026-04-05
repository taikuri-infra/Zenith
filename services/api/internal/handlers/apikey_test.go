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

func setupAPIKeyTest() (*fiber.App, *handlers.APIKeyHandler, *memory.MemoryAPIKeyRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	keyRepo := memory.NewMemoryAPIKeyRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewAPIKeyHandler(keyRepo, planRepo)
	return app, handler, keyRepo
}

func TestAPIKeyCreate(t *testing.T) {
	app, handler, _ := setupAPIKeyTest()
	app.Post("/api/v1/api-keys", injectUserID("user-1"), handler.Create)

	body := `{"name":"My Key","scopes":["read","write"]}`
	req := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result entities.APIKey
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Name != "My Key" {
		t.Errorf("Expected name 'My Key', got '%s'", result.Name)
	}
	if result.Key == "" {
		t.Error("Expected non-empty key on creation")
	}
	if result.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestAPIKeyCreateNoName(t *testing.T) {
	app, handler, _ := setupAPIKeyTest()
	app.Post("/api/v1/api-keys", injectUserID("user-1"), handler.Create)

	body := `{"scopes":["read"]}`
	req := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestAPIKeyCreateDefaultType(t *testing.T) {
	app, handler, _ := setupAPIKeyTest()
	app.Post("/api/v1/api-keys", injectUserID("user-1"), handler.Create)

	body := `{"name":"Default Type"}`
	req := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}
}

func TestAPIKeyList(t *testing.T) {
	app, handler, keyRepo := setupAPIKeyTest()

	keyRepo.CreateAPIKey(nil, "user-1", "Key 1", []string{"read"})
	keyRepo.CreateAPIKey(nil, "user-1", "Key 2", []string{"read", "write"})

	app.Get("/api/v1/api-keys", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/api-keys", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.APIKey `json:"items"`
		Total int               `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 keys, got %d", result.Total)
	}
}

func TestAPIKeyListEmpty(t *testing.T) {
	app, handler, _ := setupAPIKeyTest()
	app.Get("/api/v1/api-keys", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/api-keys", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIKeyDelete(t *testing.T) {
	app, handler, keyRepo := setupAPIKeyTest()

	key, _ := keyRepo.CreateAPIKey(nil, "user-1", "ToDelete", []string{"read"})

	app.Delete("/api/v1/api-keys/:keyId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/api-keys/"+key.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "API key revoked" {
		t.Errorf("Expected 'API key revoked', got '%v'", result["message"])
	}
}

func TestAPIKeyDeleteNotFound(t *testing.T) {
	app, handler, _ := setupAPIKeyTest()
	app.Delete("/api/v1/api-keys/:keyId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/api-keys/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestAPIKeyDeleteForbidden(t *testing.T) {
	app, handler, keyRepo := setupAPIKeyTest()

	key, _ := keyRepo.CreateAPIKey(nil, "user-1", "OtherKey", []string{"read"})

	app.Delete("/api/v1/api-keys/:keyId", injectUserID("user-2"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/api-keys/"+key.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestAPIKeyCreateLimitReached(t *testing.T) {
	app, handler, _ := setupAPIKeyTest()
	app.Post("/api/v1/api-keys", injectUserID("user-1"), handler.Create)

	// Free plan allows 1 key. Create one first.
	body := `{"name":"Key 1"}`
	req := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	app.Test(req)

	// Second key should be forbidden on free plan
	body2 := `{"name":"Key 2"}`
	req2 := httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewBufferString(body2))
	req2.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req2)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for plan limit, got %d", resp.StatusCode)
	}
}
