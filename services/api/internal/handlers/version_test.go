package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func TestVersionInfoReportsMode(t *testing.T) {
	app := fiber.New()
	app.Get("/version", handlers.VersionInfo("0.1.0-alpha.4", "2026-10-16", "standalone"))
	resp, _ := app.Test(httptest.NewRequest("GET", "/version", nil), -1)
	var body handlers.VersionResponse
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Version != "0.1.0-alpha.4" || body.Mode != "standalone" {
		t.Fatalf("got %+v", body)
	}
}
