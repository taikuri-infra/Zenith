package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func TestErrorHandlerFiberError(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "bad input")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
	}

	var result handlers.APIError
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Code != 400 {
		t.Errorf("Expected code 400, got %d", result.Code)
	}
	if result.Message != "bad input" {
		t.Errorf("Expected message 'bad input', got '%s'", result.Message)
	}
}

func TestErrorHandlerGenericError(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.ErrInternalServerError
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 500 {
		t.Fatalf("Expected 500, got %d", resp.StatusCode)
	}

	var result handlers.APIError
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Code != 500 {
		t.Errorf("Expected code 500, got %d", result.Code)
	}
}

func TestNewBadRequest(t *testing.T) {
	err := handlers.NewBadRequest("invalid field")
	if err.Code != 400 {
		t.Errorf("Expected 400, got %d", err.Code)
	}
	if err.Message != "invalid field" {
		t.Errorf("Expected 'invalid field', got '%s'", err.Message)
	}
}

func TestNewNotFound(t *testing.T) {
	err := handlers.NewNotFound("project")
	if err.Code != 404 {
		t.Errorf("Expected 404, got %d", err.Code)
	}
	if err.Message != "project not found" {
		t.Errorf("Expected 'project not found', got '%s'", err.Message)
	}
}

func TestNewUnauthorized(t *testing.T) {
	err := handlers.NewUnauthorized("not logged in")
	if err.Code != 401 {
		t.Errorf("Expected 401, got %d", err.Code)
	}
}

func TestNewForbidden(t *testing.T) {
	err := handlers.NewForbidden("access denied")
	if err.Code != 403 {
		t.Errorf("Expected 403, got %d", err.Code)
	}
}

func TestNewConflict(t *testing.T) {
	err := handlers.NewConflict("already exists")
	if err.Code != 409 {
		t.Errorf("Expected 409, got %d", err.Code)
	}
}

func TestNewInternal(t *testing.T) {
	err := handlers.NewInternal("something broke")
	if err.Code != 500 {
		t.Errorf("Expected 500, got %d", err.Code)
	}
}
