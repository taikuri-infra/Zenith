package middleware

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSecurityHeaders(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-Xss-Protection":          "1; mode=block",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Cache-Control":             "no-store",
		"Pragma":                    "no-cache",
		"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
		"Permissions-Policy":        "camera=(), microphone=(), geolocation=(), payment=()",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}

	for header, expected := range expectedHeaders {
		actual := resp.Header.Get(header)
		if actual != expected {
			t.Errorf("Header %s: expected %q, got %q", header, expected, actual)
		}
	}

	// Server header should be empty
	if server := resp.Header.Get("Server"); server != "" {
		t.Errorf("Expected empty Server header, got %q", server)
	}
}

func TestSecurityHeadersDoNotAffectBody(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status":"ok"}` {
		t.Errorf("Unexpected body: %s", body)
	}
}

func TestSecurityHeadersAppliedToAllStatusCodes(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusInternalServerError, "server error")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	// Security headers should be present even on error responses
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("Expected X-Content-Type-Options on error responses")
	}
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Error("Expected X-Frame-Options on error responses")
	}
}
