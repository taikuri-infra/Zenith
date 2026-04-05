package services

import (
	"strings"
	"testing"
)

// --- parseEnvironment tests ---

func TestParseEnvironment_MapFormat(t *testing.T) {
	env := map[string]interface{}{
		"DB_HOST": "localhost",
		"DB_PORT": 5432,
	}
	result := parseEnvironment(env)
	if result["DB_HOST"] != "localhost" {
		t.Errorf("Expected 'localhost', got '%s'", result["DB_HOST"])
	}
	if result["DB_PORT"] != "5432" {
		t.Errorf("Expected '5432', got '%s'", result["DB_PORT"])
	}
}

func TestParseEnvironment_ListFormat(t *testing.T) {
	env := []interface{}{
		"DB_HOST=localhost",
		"DB_PORT=5432",
		"SECRET_KEY",
	}
	result := parseEnvironment(env)
	if result["DB_HOST"] != "localhost" {
		t.Errorf("Expected 'localhost', got '%s'", result["DB_HOST"])
	}
	if result["DB_PORT"] != "5432" {
		t.Errorf("Expected '5432', got '%s'", result["DB_PORT"])
	}
	if result["SECRET_KEY"] != "" {
		t.Errorf("Expected empty value for key-only env var, got '%s'", result["SECRET_KEY"])
	}
}

func TestParseEnvironment_Nil(t *testing.T) {
	result := parseEnvironment(nil)
	if len(result) != 0 {
		t.Errorf("Expected empty map for nil env, got %d entries", len(result))
	}
}

func TestParseEnvironment_ListWithNonString(t *testing.T) {
	env := []interface{}{
		"VALID_KEY=value",
		12345, // non-string, should be skipped
	}
	result := parseEnvironment(env)
	if len(result) != 1 {
		t.Errorf("Expected 1 entry (skipping non-string), got %d", len(result))
	}
}

// --- parseDependsOn tests ---

func TestParseDependsOn_ListFormat(t *testing.T) {
	dep := []interface{}{"db", "cache"}
	result := parseDependsOn(dep)
	if len(result) != 2 {
		t.Fatalf("Expected 2 depends, got %d", len(result))
	}
	if result[0] != "db" || result[1] != "cache" {
		t.Errorf("Expected [db, cache], got %v", result)
	}
}

func TestParseDependsOn_MapFormat(t *testing.T) {
	dep := map[string]interface{}{
		"db":    map[string]interface{}{"condition": "service_healthy"},
		"cache": map[string]interface{}{"condition": "service_started"},
	}
	result := parseDependsOn(dep)
	if len(result) != 2 {
		t.Fatalf("Expected 2 depends, got %d", len(result))
	}
	// Map order is not guaranteed, check both are present
	found := make(map[string]bool)
	for _, r := range result {
		found[r] = true
	}
	if !found["db"] || !found["cache"] {
		t.Errorf("Expected db and cache in depends_on, got %v", result)
	}
}

func TestParseDependsOn_Nil(t *testing.T) {
	result := parseDependsOn(nil)
	if result != nil {
		t.Errorf("Expected nil for nil input, got %v", result)
	}
}

// --- extractCommand tests ---

func TestExtractCommand_String(t *testing.T) {
	cmd := extractCommand("npm start")
	if cmd != "npm start" {
		t.Errorf("Expected 'npm start', got '%s'", cmd)
	}
}

func TestExtractCommand_List(t *testing.T) {
	cmd := extractCommand([]interface{}{"npm", "run", "start"})
	if cmd != "npm run start" {
		t.Errorf("Expected 'npm run start', got '%s'", cmd)
	}
}

func TestExtractCommand_Nil(t *testing.T) {
	cmd := extractCommand(nil)
	if cmd != "" {
		t.Errorf("Expected empty string for nil, got '%s'", cmd)
	}
}

func TestExtractCommand_Other(t *testing.T) {
	cmd := extractCommand(12345)
	if cmd != "" {
		t.Errorf("Expected empty string for non-string/list, got '%s'", cmd)
	}
}

// --- containsHardcodedPassword tests ---

func TestContainsHardcodedPassword_True(t *testing.T) {
	cases := []string{
		"password=secret123",
		"POSTGRES_PASSWORD=mypass",
		"SECRET=abc",
	}
	for _, c := range cases {
		if !containsHardcodedPassword(c) {
			t.Errorf("Expected true for '%s'", c)
		}
	}
}

func TestContainsHardcodedPassword_False(t *testing.T) {
	cases := []string{
		"DATABASE_URL=postgresql://host:5432/db",
		"PORT=3000",
		"NODE_ENV=production",
	}
	for _, c := range cases {
		if containsHardcodedPassword(c) {
			t.Errorf("Expected false for '%s'", c)
		}
	}
}

// --- isFrontendService tests ---

func TestIsFrontendService_BackendNames(t *testing.T) {
	names := []string{"api", "backend", "server", "worker", "grpc-service", "queue", "cron", "scheduler", "internal-svc"}
	for _, name := range names {
		if isFrontendService(name, "", 3000) {
			t.Errorf("Expected '%s' to be identified as backend (not frontend)", name)
		}
	}
}

func TestIsFrontendService_FrontendImages(t *testing.T) {
	images := []string{"nginx:latest", "httpd:2.4", "caddy:2", "traefik:v2.10"}
	for _, img := range images {
		if !isFrontendService("myapp", img, 8080) {
			t.Errorf("Expected image '%s' to be identified as frontend", img)
		}
	}
}

func TestIsFrontendService_FrontendPorts(t *testing.T) {
	ports := []int{80, 443, 3000}
	for _, port := range ports {
		if !isFrontendService("myapp", "myimage", port) {
			t.Errorf("Expected port %d to be identified as frontend", port)
		}
	}
}

func TestIsFrontendService_DefaultInternal(t *testing.T) {
	// Port 8080 with non-frontend name and image should be internal
	if isFrontendService("myapp", "myimage", 8080) {
		t.Error("Expected port 8080 with generic name to be internal by default")
	}
}

// --- translateServiceURL tests ---

func TestTranslateServiceURL_MatchingHost(t *testing.T) {
	serviceNames := map[string]bool{"api": true, "db": true}
	result := translateServiceURL("http://api:8080/path", "my-proj", "zenith-apps", serviceNames)
	expected := "http://api-my-proj.zenith-apps.svc:8080/path"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestTranslateServiceURL_NonMatchingHost(t *testing.T) {
	serviceNames := map[string]bool{"api": true}
	result := translateServiceURL("http://external:8080/path", "my-proj", "zenith-apps", serviceNames)
	expected := "http://external:8080/path"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestTranslateServiceURL_PostgresURL(t *testing.T) {
	serviceNames := map[string]bool{"db": true}
	result := translateServiceURL("postgresql://db:5432/mydb", "proj", "ns", serviceNames)
	expected := "postgresql://db-proj.ns.svc:5432/mydb"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestTranslateServiceURL_NoMatch(t *testing.T) {
	serviceNames := map[string]bool{"api": true}
	result := translateServiceURL("just-a-string", "proj", "ns", serviceNames)
	if result != "just-a-string" {
		t.Errorf("Expected unchanged string, got '%s'", result)
	}
}

// --- ParseCompose edge cases ---

func TestParseCompose_PublicURLGeneration(t *testing.T) {
	content := `
services:
  frontend:
    image: node:18
    ports:
      - "3000:3000"
`
	result, err := ParseCompose(content, "myproj", "zenith-apps", "apps.stage.freezenith.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(result.Services))
	}
	svc := result.Services[0]
	if !svc.IsPublic {
		t.Error("Expected frontend on port 3000 to be public")
	}
	if svc.URL == "" {
		t.Error("Expected URL to be generated with baseDomain")
	}
	if !strings.Contains(svc.URL, "apps.stage.freezenith.com") {
		t.Errorf("Expected URL to contain base domain, got '%s'", svc.URL)
	}
}

func TestParseCompose_HardcodedPasswordWarning(t *testing.T) {
	// The containsHardcodedPassword pattern matches "password=VALUE" in the env var VALUE.
	// This typically catches connection strings like "postgresql://user:password=secret@host"
	content := `
services:
  app:
    image: myapp:latest
    ports:
      - "3000:3000"
    environment:
      DB_URL: "postgresql://user:password=s3cret@db:5432/mydb"
`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "hardcoded password") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected hardcoded password warning, got warnings: %v", result.Warnings)
	}
}

func TestParseCompose_VolumeWarning(t *testing.T) {
	content := `
services:
  app:
    image: myapp:latest
    ports:
      - "3000:3000"
    volumes:
      - ./data:/app/data
`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "volumes") && strings.Contains(w, "EPHEMERAL") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected volume ephemeral warning, got warnings: %v", result.Warnings)
	}
}

func TestParseCompose_BuildContextMap(t *testing.T) {
	content := `
services:
  app:
    build:
      context: ./api
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(result.Services))
	}
	if result.Services[0].BuildContext != "./api" {
		t.Errorf("Expected build context './api', got '%s'", result.Services[0].BuildContext)
	}
}

func TestParseCompose_BuildContextString(t *testing.T) {
	content := `
services:
  app:
    build: ./api
    ports:
      - "8080:8080"
`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Services[0].BuildContext != "./api" {
		t.Errorf("Expected build context './api', got '%s'", result.Services[0].BuildContext)
	}
}

func TestParseCompose_CommandString(t *testing.T) {
	content := `
services:
  app:
    image: myapp:latest
    ports:
      - "8080:8080"
    command: npm start
`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Services[0].Command != "npm start" {
		t.Errorf("Expected command 'npm start', got '%s'", result.Services[0].Command)
	}
}

func TestParseCompose_CommandList(t *testing.T) {
	content := `
services:
  app:
    image: myapp:latest
    ports:
      - "8080:8080"
    command: ["npm", "run", "start"]
`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Services[0].Command != "npm run start" {
		t.Errorf("Expected command 'npm run start', got '%s'", result.Services[0].Command)
	}
}

func TestParseCompose_PlainHostTranslation(t *testing.T) {
	content := `
services:
  app:
    image: myapp:latest
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
  postgres:
    image: postgres:16
`
	result, err := ParseCompose(content, "myproj", "zenith-apps", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, svc := range result.Services {
		if svc.Name == "app" {
			for _, ev := range svc.EnvVars {
				if ev.Key == "DB_HOST" {
					expected := "postgres-myproj.zenith-apps.svc"
					if ev.Zenith != expected {
						t.Errorf("Expected plain host translated to '%s', got '%s'", expected, ev.Zenith)
					}
					return
				}
			}
			t.Error("DB_HOST env var not found")
			return
		}
	}
	t.Error("app service not found")
}

func TestParseCompose_NoBuildNoPort_Warning(t *testing.T) {
	content := `
services:
  app:
    build: .
    environment:
      PORT: "3000"
`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "no port detected") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'no port detected' warning for service with build but no port, got: %v", result.Warnings)
	}
}

func TestParseCompose_AllManagedImages(t *testing.T) {
	content := `
services:
  pg:
    image: postgres:16
  mysql:
    image: mysql:8
  redis:
    image: redis:7
  mongo:
    image: mongo:7
  rabbit:
    image: rabbitmq:3
  es:
    image: elasticsearch:8.12
  nats:
    image: nats:latest
  memcached:
    image: memcached:1.6
  minio:
    image: minio/minio:latest
  clickhouse:
    image: clickhouse:latest
  kafka:
    image: kafka:3
  valkey:
    image: valkey:latest
  mariadb:
    image: mariadb:10
  zookeeper:
    image: zookeeper:3.9
`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ManagedServices) < 14 {
		t.Errorf("Expected at least 14 managed services, got %d", len(result.ManagedServices))
	}
	if len(result.Services) != 0 {
		t.Errorf("Expected 0 app services (all managed), got %d", len(result.Services))
	}
}

func TestParseCompose_RegistryPrefixImage(t *testing.T) {
	content := `
services:
  pg:
    image: docker.io/library/postgres:16
`
	result, err := ParseCompose(content, "test", "ns", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ManagedServices) != 1 {
		t.Fatalf("Expected 1 managed service, got %d", len(result.ManagedServices))
	}
	if result.ManagedServices[0].Type != "postgresql" {
		t.Errorf("Expected type 'postgresql', got '%s'", result.ManagedServices[0].Type)
	}
}

func TestExtractPort_SinglePort(t *testing.T) {
	// Single port without host mapping
	got := extractPort([]string{"3000"})
	if got != 3000 {
		t.Errorf("extractPort(['3000']) = %d, want 3000", got)
	}
}

func TestExtractPort_ProtocolSuffix(t *testing.T) {
	got := extractPort([]string{"3000/udp"})
	if got != 3000 {
		t.Errorf("extractPort(['3000/udp']) = %d, want 3000", got)
	}
}

func TestExtractPort_InvalidFormat(t *testing.T) {
	got := extractPort([]string{"abc"})
	if got != 0 {
		t.Errorf("extractPort(['abc']) = %d, want 0", got)
	}
}

func TestDetectVersion_Latest(t *testing.T) {
	got := detectVersion("postgres:latest")
	if got != "latest" {
		t.Errorf("detectVersion('postgres:latest') = '%s', want 'latest'", got)
	}
}

func TestDetectVersion_NoTag(t *testing.T) {
	got := detectVersion("postgres")
	if got != "latest" {
		t.Errorf("detectVersion('postgres') = '%s', want 'latest'", got)
	}
}
