package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func TestRegisterDisabledInStandalone(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	h := handlers.NewAuthHandler(nil, nil)
	h.SetRegistrationDisabled(true)
	app.Post("/api/v1/auth/register", h.Register)

	body := `{"email":"a@b.c","password":"longenough","name":"A"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 403 {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	// handlers.ErrorHandler serialises fiber.Error as {"code":..,"message":..}.
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	want := "registration is disabled on self-hosted installs; the administrator account is created by the installer"
	if out["message"] != want {
		t.Fatalf("message = %v, want %q", out["message"], want)
	}
}

func TestRegisterStillValidatesWhenEnabled(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	h := handlers.NewAuthHandler(nil, nil) // svc is never reached: the body is invalid
	app.Post("/api/v1/auth/register", h.Register)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400 (gate must not apply when enabled)", resp.StatusCode)
	}
}
