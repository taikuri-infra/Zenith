package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestStructuredLoggerSuccess(t *testing.T) {
	app := fiber.New()
	app.Use(StructuredLogger())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestStructuredLoggerWithRequestID(t *testing.T) {
	app := fiber.New()
	// Set request_id before the logger
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("request_id", "req-12345")
		return c.Next()
	})
	app.Use(StructuredLogger())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestStructuredLoggerWithUserID(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-42")
		return c.Next()
	})
	app.Use(StructuredLogger())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestStructuredLogger4xxStatus(t *testing.T) {
	app := fiber.New()
	app.Use(StructuredLogger())
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestStructuredLogger5xxStatus(t *testing.T) {
	app := fiber.New()
	app.Use(StructuredLogger())
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusInternalServerError, "server error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("Expected 500, got %d", resp.StatusCode)
	}
}

func TestStructuredLoggerWithError(t *testing.T) {
	app := fiber.New()
	app.Use(StructuredLogger())
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusServiceUnavailable, "service unavailable")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 503 {
		t.Errorf("Expected 503, got %d", resp.StatusCode)
	}
}
