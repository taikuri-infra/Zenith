package handlers_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: handlers.ErrorHandler,
	})
	return app
}

func TestHealthCheck(t *testing.T) {
	app := setupApp()
	app.Get("/health", handlers.HealthCheck("1.0.0", "2026-01-01", "abc123"))

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var health handlers.HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}
	if health.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", health.Version)
	}
	if health.Uptime == "" {
		t.Error("Expected non-empty uptime")
	}
}

func TestReadinessCheck(t *testing.T) {
	app := setupApp()
	app.Get("/ready", handlers.ReadinessCheck())

	req := httptest.NewRequest("GET", "/ready", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test readiness endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var ready handlers.ReadinessResponse
	if err := json.Unmarshal(body, &ready); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if ready.Status != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", ready.Status)
	}
	if ready.Checks["server"] != "ready" {
		t.Errorf("Expected server check 'ready', got '%s'", ready.Checks["server"])
	}
}

func TestVersionInfo(t *testing.T) {
	app := setupApp()
	app.Get("/version", handlers.VersionInfo("1.0.0", "2026-01-01", "abc123"))

	req := httptest.NewRequest("GET", "/version", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test version endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var version handlers.VersionResponse
	if err := json.Unmarshal(body, &version); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if version.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", version.Version)
	}
}

func TestErrorHandler(t *testing.T) {
	app := setupApp()
	app.Get("/error", func(c *fiber.Ctx) error {
		return handlers.NewNotFound("project")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test error handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var apiErr handlers.APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if apiErr.Code != 404 {
		t.Errorf("Expected error code 404, got %d", apiErr.Code)
	}
}

func TestErrorHandlerBadRequest(t *testing.T) {
	app := setupApp()
	app.Get("/bad", func(c *fiber.Ctx) error {
		return handlers.NewBadRequest("invalid input")
	})

	req := httptest.NewRequest("GET", "/bad", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test bad request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Fatalf("Expected status 400, got %d", resp.StatusCode)
	}
}
