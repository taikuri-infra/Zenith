package config

import (
	"os"
	"strings"
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

func TestValidate_EmptyJWTSecret(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty JWT_SECRET")
	}
}

func TestValidate_ShortJWTSecret(t *testing.T) {
	cfg := &Config{JWTSecret: "tooshort"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for short JWT_SECRET")
	}
}

func TestValidate_BadSecretsKey(t *testing.T) {
	cfg := &Config{JWTSecret: strings.Repeat("a", 32), SecretsKey: "not-hex"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid SECRETS_ENCRYPTION_KEY")
	}
}

func TestValidate_WrongLengthSecretsKey(t *testing.T) {
	cfg := &Config{JWTSecret: strings.Repeat("a", 32), SecretsKey: "aabbcc"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for wrong-length SECRETS_ENCRYPTION_KEY")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{JWTSecret: strings.Repeat("a", 32)}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_ValidConfigWithSecretsKey(t *testing.T) {
	cfg := &Config{
		JWTSecret:  strings.Repeat("a", 32),
		SecretsKey: strings.Repeat("ab", 32), // 64 hex chars
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildDatabaseURL_SpecialCharsInPassword(t *testing.T) {
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PASSWORD", "p@ss#word")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PASSWORD")
	}()
	u := buildDatabaseURL()
	if !strings.Contains(u, "p%40ss%23word") {
		t.Errorf("expected URL-escaped password, got: %s", u)
	}
}
