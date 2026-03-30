package services

import (
	"testing"
)

func TestParseCompose_BasicApp(t *testing.T) {
	content := `
version: "3.8"
services:
  api:
    build: ./api
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgresql://db:5432/mydb
      REDIS_URL: redis://cache:6379
    depends_on:
      - db
      - cache
  db:
    image: postgres:16
    environment:
      POSTGRES_DB: mydb
      POSTGRES_PASSWORD: secret123
  cache:
    image: redis:7-alpine
`
	result, err := ParseCompose(content, "my-saas", "zenith-apps", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}

	// Should detect 1 app service
	if len(result.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(result.Services))
	}
	if result.Services[0].Name != "api" {
		t.Errorf("expected service name 'api', got '%s'", result.Services[0].Name)
	}
	if result.Services[0].Port != 8080 {
		t.Errorf("expected port 8080, got %d", result.Services[0].Port)
	}
	if result.Services[0].IsPublic {
		t.Error("expected 'api' service to be internal (not public)")
	}

	// Should detect 2 managed services
	if len(result.ManagedServices) != 2 {
		t.Fatalf("expected 2 managed services, got %d", len(result.ManagedServices))
	}

	managedByName := make(map[string]ParsedManaged)
	for _, ms := range result.ManagedServices {
		managedByName[ms.Name] = ms
	}

	pg, ok := managedByName["db"]
	if !ok {
		t.Fatal("expected managed service 'db'")
	}
	if pg.Type != "postgresql" {
		t.Errorf("expected type 'postgresql', got '%s'", pg.Type)
	}
	if pg.Version != "16" {
		t.Errorf("expected version '16', got '%s'", pg.Version)
	}

	redis, ok := managedByName["cache"]
	if !ok {
		t.Fatal("expected managed service 'cache'")
	}
	if redis.Type != "redis" {
		t.Errorf("expected type 'redis', got '%s'", redis.Type)
	}
}

func TestParseCompose_URLTranslation(t *testing.T) {
	content := `
services:
  web:
    build: .
    ports:
      - "3000:3000"
    environment:
      API_URL: http://api:8080
  api:
    build: ./api
    ports:
      - "8080:8080"
    environment:
      DB_URL: postgresql://db:5432/mydb
  db:
    image: postgres:16
`
	result, err := ParseCompose(content, "myproject", "zenith-apps", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the web service's API_URL env var
	for _, svc := range result.Services {
		if svc.Name == "web" {
			for _, ev := range svc.EnvVars {
				if ev.Key == "API_URL" {
					expected := "http://api-myproject.zenith-apps.svc:8080"
					if ev.Zenith != expected {
						t.Errorf("expected translated URL '%s', got '%s'", expected, ev.Zenith)
					}
					return
				}
			}
			t.Error("API_URL env var not found in web service")
			return
		}
	}
	t.Error("web service not found")
}

func TestParseCompose_InvalidYAML(t *testing.T) {
	content := `this is not valid yaml: [[[`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for bad YAML")
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors for bad YAML")
	}
}

func TestParseCompose_NoServices(t *testing.T) {
	content := `version: "3.8"`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for empty services")
	}
}

func TestParseCompose_OnlyManagedServices(t *testing.T) {
	content := `
services:
  db:
    image: postgres:16
  cache:
    image: redis:7
`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid result")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning about no app services")
	}
}

func TestExtractPort(t *testing.T) {
	tests := []struct {
		ports    []string
		expected int
	}{
		{[]string{"3000:3000"}, 3000},
		{[]string{"8080:8080/tcp"}, 8080},
		{[]string{"80:3000"}, 3000},
		{[]string{}, 0},
		{nil, 0},
	}

	for _, tt := range tests {
		got := extractPort(tt.ports)
		if got != tt.expected {
			t.Errorf("extractPort(%v) = %d, want %d", tt.ports, got, tt.expected)
		}
	}
}

func TestExtractImageBase(t *testing.T) {
	tests := []struct {
		image    string
		expected string
	}{
		{"postgres:16", "postgres"},
		{"docker.io/library/redis:7-alpine", "redis"},
		{"my-registry.io/my-app:latest", "my-app"},
		{"postgres", "postgres"},
	}

	for _, tt := range tests {
		got := extractImageBase(tt.image)
		if got != tt.expected {
			t.Errorf("extractImageBase(%q) = %q, want %q", tt.image, got, tt.expected)
		}
	}
}

func TestDetectVersion(t *testing.T) {
	tests := []struct {
		image    string
		expected string
	}{
		{"postgres:16", "16"},
		{"redis:7-alpine", "7-alpine"},
		{"postgres:latest", "latest"},
		{"postgres", "latest"},
	}

	for _, tt := range tests {
		got := detectVersion(tt.image)
		if got != tt.expected {
			t.Errorf("detectVersion(%q) = %q, want %q", tt.image, got, tt.expected)
		}
	}
}

func TestValidateCompose(t *testing.T) {
	// Valid compose
	parsed := &ParsedCompose{
		Valid: true,
		Services: []ParsedService{
			{Name: "api", Port: 8080},
		},
		ManagedServices: []ParsedManaged{
			{Name: "db", Type: "postgresql"},
		},
	}

	issues := ValidateCompose(parsed)
	if len(issues) != 0 {
		t.Errorf("expected no issues, got: %v", issues)
	}

	// No app services
	parsed2 := &ParsedCompose{
		Valid:    true,
		Services: []ParsedService{},
		ManagedServices: []ParsedManaged{
			{Name: "db", Type: "postgresql"},
		},
	}

	issues2 := ValidateCompose(parsed2)
	if len(issues2) == 0 {
		t.Error("expected issues for no app services")
	}
}
