package telemetry

import (
	"testing"
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
