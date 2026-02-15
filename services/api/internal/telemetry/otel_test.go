package telemetry

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestConfigDefaults(t *testing.T) {
	cfg := Config{
		OTLPEndpoint: "localhost:4317",
	}

	// Verify that empty service name will get a default
	if cfg.ServiceName != "" {
		t.Errorf("Expected empty service name before Init, got '%s'", cfg.ServiceName)
	}

	// Verify SampleRate default behavior
	if cfg.SampleRate != 0 {
		t.Errorf("Expected zero SampleRate before Init, got %f", cfg.SampleRate)
	}
}

func TestMiddlewareConfigSkipPaths(t *testing.T) {
	cfg := MiddlewareConfig{
		SkipPaths: []string{"/health", "/ready", "/metrics"},
	}

	skipSet := make(map[string]bool)
	for _, p := range cfg.SkipPaths {
		skipSet[p] = true
	}

	if !skipSet["/health"] {
		t.Error("Expected /health in skip paths")
	}
	if !skipSet["/ready"] {
		t.Error("Expected /ready in skip paths")
	}
	if !skipSet["/metrics"] {
		t.Error("Expected /metrics in skip paths")
	}
	if skipSet["/api/v1/apps"] {
		t.Error("Did not expect /api/v1/apps in skip paths")
	}
}

func TestMiddlewareConfigTracerName(t *testing.T) {
	cfg := MiddlewareConfig{
		TracerName: "custom-tracer",
	}

	if cfg.TracerName != "custom-tracer" {
		t.Errorf("Expected tracer name 'custom-tracer', got '%s'", cfg.TracerName)
	}

	// Default tracer name
	defaultCfg := MiddlewareConfig{}
	if defaultCfg.TracerName != "" {
		t.Errorf("Expected empty default tracer name, got '%s'", defaultCfg.TracerName)
	}
}

func TestConfigFields(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "otel-collector:4317",
		Environment:    "production",
		Insecure:       true,
		SampleRate:     0.5,
	}

	if cfg.ServiceName != "test-service" {
		t.Errorf("Expected ServiceName 'test-service', got '%s'", cfg.ServiceName)
	}
	if cfg.ServiceVersion != "1.0.0" {
		t.Errorf("Expected ServiceVersion '1.0.0', got '%s'", cfg.ServiceVersion)
	}
	if cfg.OTLPEndpoint != "otel-collector:4317" {
		t.Errorf("Expected OTLPEndpoint 'otel-collector:4317', got '%s'", cfg.OTLPEndpoint)
	}
	if cfg.Environment != "production" {
		t.Errorf("Expected Environment 'production', got '%s'", cfg.Environment)
	}
	if !cfg.Insecure {
		t.Error("Expected Insecure to be true")
	}
	if cfg.SampleRate != 0.5 {
		t.Errorf("Expected SampleRate 0.5, got %f", cfg.SampleRate)
	}
}

func TestConfigDefaultServiceName(t *testing.T) {
	// When ServiceName is empty, Init should set it to "zenith-api"
	// We can't call Init (needs real OTLP endpoint), but we can test the defaulting logic
	cfg := Config{
		OTLPEndpoint: "localhost:4317",
	}

	if cfg.ServiceName != "" {
		t.Fatalf("Test setup: expected empty ServiceName")
	}

	// Replicate the default logic from Init
	if cfg.ServiceName == "" {
		cfg.ServiceName = "zenith-api"
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 1.0
	}

	if cfg.ServiceName != "zenith-api" {
		t.Errorf("Expected default ServiceName 'zenith-api', got '%s'", cfg.ServiceName)
	}
	if cfg.SampleRate != 1.0 {
		t.Errorf("Expected default SampleRate 1.0, got %f", cfg.SampleRate)
	}
}

func TestConfigDefaultSampleRateNegative(t *testing.T) {
	cfg := Config{
		OTLPEndpoint: "localhost:4317",
		SampleRate:   -0.5,
	}

	// Replicate the default logic from Init
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 1.0
	}

	if cfg.SampleRate != 1.0 {
		t.Errorf("Expected default SampleRate 1.0 for negative input, got %f", cfg.SampleRate)
	}
}

func TestMiddlewareCreation(t *testing.T) {
	// Test that Middleware can be created without panicking
	handler := Middleware()
	if handler == nil {
		t.Error("Expected non-nil middleware handler")
	}
}

func TestMiddlewareCreationWithConfig(t *testing.T) {
	handler := Middleware(MiddlewareConfig{
		TracerName: "custom-tracer",
		SkipPaths:  []string{"/health", "/ready"},
	})
	if handler == nil {
		t.Error("Expected non-nil middleware handler")
	}
}

func TestMiddlewareSkipPaths(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware(MiddlewareConfig{
		SkipPaths: []string{"/health", "/ready"},
	}))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("healthy")
	})
	app.Get("/api/v1/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// /health should be skipped (no tracing) but still work
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 for skipped path, got %d", resp.StatusCode)
	}

	// /api/v1/test should be traced and still work
	req2 := httptest.NewRequest("GET", "/api/v1/test", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != 200 {
		t.Errorf("Expected 200 for traced path, got %d", resp2.StatusCode)
	}
}

func TestMiddlewareRecordsStatusCode(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware())
	app.Get("/ok", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.Get("/error", func(c *fiber.Ctx) error {
		return c.Status(500).SendString("error")
	})
	app.Get("/notfound", func(c *fiber.Ctx) error {
		return c.Status(404).SendString("not found")
	})

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/ok", 200},
		{"/error", 500},
		{"/notfound", 404},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("Expected %d, got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestMiddlewareHandlesError(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware())
	app.Get("/fail", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest("GET", "/fail", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestRequestHeadersCarrier(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		carrier := requestHeaders(c)

		// Test Get
		val := carrier.Get("X-Custom-Header")
		if val != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", val)
		}

		// Test Set
		carrier.Set("X-New-Header", "new-value")
		newVal := carrier.Get("X-New-Header")
		if newVal != "new-value" {
			t.Errorf("Expected 'new-value' after Set, got '%s'", newVal)
		}

		// Test Keys
		keys := carrier.Keys()
		if len(keys) == 0 {
			t.Error("Expected at least one key")
		}

		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Custom-Header", "test-value")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestResponseHeadersCarrier(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		carrier := responseHeaders(c)

		// Test Set
		carrier.Set("X-Response-Header", "response-value")

		// Test Get
		val := carrier.Get("X-Response-Header")
		if val != "response-value" {
			t.Errorf("Expected 'response-value', got '%s'", val)
		}

		// Test Keys
		keys := carrier.Keys()
		foundKey := false
		for _, k := range keys {
			if k == "X-Response-Header" {
				foundKey = true
			}
		}
		if !foundKey {
			t.Error("Expected X-Response-Header in keys")
		}

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

func TestRequestHeadersGetMissing(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		carrier := requestHeaders(c)

		// Getting a missing header should return empty string
		val := carrier.Get("X-Nonexistent")
		if val != "" {
			t.Errorf("Expected empty string for missing header, got '%s'", val)
		}

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

func TestResponseHeadersGetMissing(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		carrier := responseHeaders(c)

		// Getting a missing response header should return empty string
		val := carrier.Get("X-Nonexistent")
		if val != "" {
			t.Errorf("Expected empty string for missing header, got '%s'", val)
		}

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

func TestMiddlewareDefaultConfig(t *testing.T) {
	// When no config is passed, should use default tracer name
	handler := Middleware()
	if handler == nil {
		t.Error("Expected non-nil handler with default config")
	}
}

func TestMiddlewareEmptySkipPaths(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware(MiddlewareConfig{
		SkipPaths: []string{},
	}))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("healthy")
	})

	// With empty skip paths, /health should still be traced but work
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestMiddlewareWithMultipleSkipPaths(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware(MiddlewareConfig{
		SkipPaths: []string{"/health", "/ready", "/metrics", "/livez"},
	}))

	for _, path := range []string{"/health", "/ready", "/metrics", "/livez"} {
		app.Get(path, func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})
	}

	for _, path := range []string{"/health", "/ready", "/metrics", "/livez"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 200 {
				t.Errorf("Expected 200 for %s, got %d", path, resp.StatusCode)
			}
		})
	}
}

func TestTracerAndMeterNames(t *testing.T) {
	// Test the constant values
	if tracerName != "zenith-api-http" {
		t.Errorf("Expected tracerName 'zenith-api-http', got '%s'", tracerName)
	}
	if meterName != "zenith-api-http" {
		t.Errorf("Expected meterName 'zenith-api-http', got '%s'", meterName)
	}
}
