package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Port)
	}
	if cfg.Environment != "development" {
		t.Errorf("Expected default environment 'development', got '%s'", cfg.Environment)
	}
	// standalone mode (default) uses localhost CORS
	if cfg.CORSOrigins != "http://localhost:3000" {
		t.Errorf("Expected default CORS origins 'http://localhost:3000', got '%s'", cfg.CORSOrigins)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got '%s'", cfg.LogLevel)
	}
	if cfg.JWTIssuer != "zenith" {
		t.Errorf("Expected default JWT issuer 'zenith', got '%s'", cfg.JWTIssuer)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("CORS_ORIGINS", "https://zenith.dev")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("CORS_ORIGINS")
	}()

	cfg := Load()

	if cfg.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Port)
	}
	if cfg.Environment != "production" {
		t.Errorf("Expected environment 'production', got '%s'", cfg.Environment)
	}
	if cfg.CORSOrigins != "https://zenith.dev" {
		t.Errorf("Expected CORS origins 'https://zenith.dev', got '%s'", cfg.CORSOrigins)
	}
}

func TestGetEnvInt_InvalidValue(t *testing.T) {
	os.Setenv("PORT", "not-a-number")
	defer os.Unsetenv("PORT")

	val := getEnvInt("PORT", 8080)
	if val != 8080 {
		t.Errorf("Expected fallback 8080 for invalid int, got %d", val)
	}
}

func TestGetEnvBool(t *testing.T) {
	os.Setenv("IN_CLUSTER", "true")
	defer os.Unsetenv("IN_CLUSTER")

	val := getEnvBool("IN_CLUSTER", false)
	if !val {
		t.Error("Expected true for IN_CLUSTER=true")
	}
}
