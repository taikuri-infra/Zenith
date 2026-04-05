package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupAdminCryptoTest() (*fiber.App, *handlers.AdminCryptoHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	envVarRepo := memory.NewMemoryEnvVarRepository()
	appRepo := memory.NewMemoryAppRepository()
	// Pass nil for envCrypto to test the "encryption not configured" path
	handler := handlers.NewAdminCryptoHandler(nil, envVarRepo, appRepo)
	return app, handler
}

func TestAdminCryptoRotateKeysNoEncryption(t *testing.T) {
	app, handler := setupAdminCryptoTest()
	app.Post("/api/v2/admin/crypto/rotate", handler.RotateKeys)

	req := httptest.NewRequest("POST", "/api/v2/admin/crypto/rotate", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Fatalf("Expected 400 (encryption not configured), got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "encryption not configured" {
		t.Errorf("Expected error message 'encryption not configured', got '%v'", result["message"])
	}
}
