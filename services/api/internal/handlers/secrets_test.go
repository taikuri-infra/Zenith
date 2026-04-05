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

func setupSecretTest() (*fiber.App, *handlers.SecretHandler, *memory.MemoryAppRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	appRepo := memory.NewMemoryAppRepository()
	// Use a valid 32-byte hex key (64 hex chars)
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	handler, _ := handlers.NewSecretHandler(appRepo, hexKey)
	return app, handler, appRepo
}

func createSecretTestApp(appRepo *memory.MemoryAppRepository, userID string) *entities.App {
	a, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		Name:         "secret-app",
		UserID:       userID,
		ProjectID:    "proj-1",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "registry.example.com/test:latest",
	})
	return a
}

func TestSecretHandlerNilOnEmptyKey(t *testing.T) {
	handler, err := handlers.NewSecretHandler(nil, "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if handler != nil {
		t.Error("Expected nil handler for empty key (dev mode)")
	}
}

func TestSecretSetAndList(t *testing.T) {
	app, handler, appRepo := setupSecretTest()
	testApp := createSecretTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/secrets", injectUserID("user-1"), handler.SetSecret)
	app.Get("/api/v1/apps/:appId/secrets", injectUserID("user-1"), handler.ListSecrets)

	// Set a secret
	body := `{"key":"DB_PASSWORD","value":"mysecretvalue"}`
	setReq := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/secrets", bytes.NewBufferString(body))
	setReq.Header.Set("Content-Type", "application/json")
	setResp, _ := app.Test(setReq)
	if setResp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", setResp.StatusCode)
	}

	// List secrets
	listReq := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/secrets", nil)
	listResp, _ := app.Test(listReq)
	if listResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", listResp.StatusCode)
	}
}

func TestSecretSetMissingKey(t *testing.T) {
	app, handler, appRepo := setupSecretTest()
	testApp := createSecretTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/secrets", injectUserID("user-1"), handler.SetSecret)

	body := `{"value":"mysecretvalue"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/secrets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestSecretSetNotOwner(t *testing.T) {
	app, handler, appRepo := setupSecretTest()
	testApp := createSecretTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/secrets", injectUserID("user-2"), handler.SetSecret)

	body := `{"key":"DB_PASSWORD","value":"mysecretvalue"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/secrets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		// Handler returns 404 for ownership mismatch (intentional)
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestSecretSetAppNotFound(t *testing.T) {
	app, handler, _ := setupSecretTest()

	app.Post("/api/v1/apps/:appId/secrets", injectUserID("user-1"), handler.SetSecret)

	body := `{"key":"DB_PASSWORD","value":"mysecretvalue"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/nonexistent/secrets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestSecretGetValueAndDelete(t *testing.T) {
	app, handler, appRepo := setupSecretTest()
	testApp := createSecretTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/secrets", injectUserID("user-1"), handler.SetSecret)
	app.Get("/api/v1/apps/:appId/secrets/:key/value", injectUserID("user-1"), handler.GetSecretValue)
	app.Delete("/api/v1/apps/:appId/secrets/:key", injectUserID("user-1"), handler.DeleteSecret)

	// Set a secret
	body := `{"key":"API_KEY","value":"secret-api-key-123"}`
	setReq := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/secrets", bytes.NewBufferString(body))
	setReq.Header.Set("Content-Type", "application/json")
	app.Test(setReq)

	// Get secret value
	getReq := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/secrets/API_KEY/value", nil)
	getResp, _ := app.Test(getReq)
	if getResp.StatusCode != 200 {
		t.Fatalf("Expected 200 for get secret value, got %d", getResp.StatusCode)
	}

	var getValue map[string]interface{}
	json.NewDecoder(getResp.Body).Decode(&getValue)
	if getValue["value"] != "secret-api-key-123" {
		t.Errorf("Expected decrypted value 'secret-api-key-123', got '%v'", getValue["value"])
	}

	// Delete secret
	delReq := httptest.NewRequest("DELETE", "/api/v1/apps/"+testApp.ID+"/secrets/API_KEY", nil)
	delResp, _ := app.Test(delReq)
	if delResp.StatusCode != 200 {
		t.Fatalf("Expected 200 for delete, got %d", delResp.StatusCode)
	}

	var delResult map[string]interface{}
	json.NewDecoder(delResp.Body).Decode(&delResult)
	if delResult["status"] != "deleted" {
		t.Errorf("Expected status 'deleted', got '%v'", delResult["status"])
	}
}

func TestSecretGetValueNotFound(t *testing.T) {
	app, handler, appRepo := setupSecretTest()
	testApp := createSecretTestApp(appRepo, "user-1")

	app.Get("/api/v1/apps/:appId/secrets/:key/value", injectUserID("user-1"), handler.GetSecretValue)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/secrets/NONEXISTENT/value", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestSecretDeleteNotFound(t *testing.T) {
	app, handler, appRepo := setupSecretTest()
	testApp := createSecretTestApp(appRepo, "user-1")

	app.Delete("/api/v1/apps/:appId/secrets/:key", injectUserID("user-1"), handler.DeleteSecret)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+testApp.ID+"/secrets/NONEXISTENT", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestSecretListNoAuth(t *testing.T) {
	app, handler, appRepo := setupSecretTest()
	testApp := createSecretTestApp(appRepo, "user-1")

	// No injectUserID middleware
	app.Get("/api/v1/apps/:appId/secrets", handler.ListSecrets)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/secrets", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}
