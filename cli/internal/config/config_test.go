package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Expected no error for missing config, got: %v", err)
	}

	if cfg.APIEndpoint != "http://localhost:8080" {
		t.Errorf("Expected default API endpoint, got '%s'", cfg.APIEndpoint)
	}
	if cfg.Region != "fsn1" {
		t.Errorf("Expected default region 'fsn1', got '%s'", cfg.Region)
	}
	if cfg.OutputFormat != "table" {
		t.Errorf("Expected default output format 'table', got '%s'", cfg.OutputFormat)
	}
}

func TestLoadDefaults_OptionalFieldsEmpty(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Expected no error for missing config, got: %v", err)
	}

	if cfg.Token != "" {
		t.Errorf("Expected empty Token by default, got '%s'", cfg.Token)
	}
	if cfg.Project != "" {
		t.Errorf("Expected empty Project by default, got '%s'", cfg.Project)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		APIEndpoint: "https://api.zenith.dev",
		Token:       "test-token-123",
		Project:     "my-project",
		Region:      "nbg1",
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.APIEndpoint != "https://api.zenith.dev" {
		t.Errorf("Expected API endpoint 'https://api.zenith.dev', got '%s'", loaded.APIEndpoint)
	}
	if loaded.Token != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", loaded.Token)
	}
	if loaded.Project != "my-project" {
		t.Errorf("Expected project 'my-project', got '%s'", loaded.Project)
	}
	if loaded.Region != "nbg1" {
		t.Errorf("Expected region 'nbg1', got '%s'", loaded.Region)
	}
}

func TestSaveAndLoad_AllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		APIEndpoint:  "https://api.zenith.dev",
		Token:        "test-token-456",
		Project:      "production",
		Region:       "hel1",
		OutputFormat: "json",
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.APIEndpoint != cfg.APIEndpoint {
		t.Errorf("APIEndpoint mismatch: got '%s', want '%s'", loaded.APIEndpoint, cfg.APIEndpoint)
	}
	if loaded.Token != cfg.Token {
		t.Errorf("Token mismatch: got '%s', want '%s'", loaded.Token, cfg.Token)
	}
	if loaded.Project != cfg.Project {
		t.Errorf("Project mismatch: got '%s', want '%s'", loaded.Project, cfg.Project)
	}
	if loaded.Region != cfg.Region {
		t.Errorf("Region mismatch: got '%s', want '%s'", loaded.Region, cfg.Region)
	}
	if loaded.OutputFormat != cfg.OutputFormat {
		t.Errorf("OutputFormat mismatch: got '%s', want '%s'", loaded.OutputFormat, cfg.OutputFormat)
	}
}

func TestSaveAndLoad_MinimalConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		APIEndpoint: "http://localhost:9090",
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.APIEndpoint != "http://localhost:9090" {
		t.Errorf("Expected API endpoint 'http://localhost:9090', got '%s'", loaded.APIEndpoint)
	}
	// Optional fields should remain empty
	if loaded.Token != "" {
		t.Errorf("Expected empty Token, got '%s'", loaded.Token)
	}
	if loaded.Project != "" {
		t.Errorf("Expected empty Project, got '%s'", loaded.Project)
	}
}

func TestSaveCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.yaml")

	cfg := &Config{APIEndpoint: "https://api.zenith.dev"}
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Expected config file to exist after save")
	}
}

func TestSaveCreatesDeepDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "config.yaml")

	cfg := &Config{APIEndpoint: "https://api.zenith.dev"}
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Expected config file to exist after save with deep path")
	}
}

func TestSave_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{APIEndpoint: "https://api.zenith.dev", Token: "secret"}
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// File should be created with 0600 permissions (owner read/write only)
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("Expected file permissions 0600, got %04o", perm)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Error("Expected non-empty config path")
	}
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("Expected config.yaml, got '%s'", filepath.Base(path))
	}
	// Should be inside .zen directory
	dir := filepath.Base(filepath.Dir(path))
	if dir != ".zen" {
		t.Errorf("Expected parent directory '.zen', got '%s'", dir)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestLoadFrom_MissingFile(t *testing.T) {
	cfg, err := LoadFrom("/this/path/definitely/does/not/exist/config.yaml")
	if err != nil {
		t.Fatalf("Expected no error for missing file (returns defaults), got: %v", err)
	}

	// Should return defaults
	if cfg.APIEndpoint != "http://localhost:8080" {
		t.Errorf("Expected default API endpoint, got '%s'", cfg.APIEndpoint)
	}
	if cfg.Region != "fsn1" {
		t.Errorf("Expected default region, got '%s'", cfg.Region)
	}
	if cfg.OutputFormat != "table" {
		t.Errorf("Expected default output format, got '%s'", cfg.OutputFormat)
	}
}

func TestLoadFrom_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Write empty file
	os.WriteFile(path, []byte(""), 0600)

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("Expected no error for empty file, got: %v", err)
	}

	// Defaults should still apply since config struct is initialized with defaults
	if cfg.APIEndpoint != "http://localhost:8080" {
		t.Errorf("Expected default API endpoint for empty file, got '%s'", cfg.APIEndpoint)
	}
}

func TestLoadFrom_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Write partial config - only override some fields
	content := `api_endpoint: "https://custom.api.dev"
token: "my-token"
`
	os.WriteFile(path, []byte(content), 0600)

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("Failed to load partial config: %v", err)
	}

	if cfg.APIEndpoint != "https://custom.api.dev" {
		t.Errorf("Expected custom API endpoint, got '%s'", cfg.APIEndpoint)
	}
	if cfg.Token != "my-token" {
		t.Errorf("Expected token 'my-token', got '%s'", cfg.Token)
	}
	// Fields not in file should retain defaults
	// Note: YAML unmarshal into the pre-initialized struct preserves non-zero values for absent fields
}

func TestSave_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Save first config
	cfg1 := &Config{
		APIEndpoint: "http://first.api",
		Token:       "token-1",
	}
	if err := cfg1.SaveTo(path); err != nil {
		t.Fatalf("First save failed: %v", err)
	}

	// Save second config to same path
	cfg2 := &Config{
		APIEndpoint: "http://second.api",
		Token:       "token-2",
	}
	if err := cfg2.SaveTo(path); err != nil {
		t.Fatalf("Second save failed: %v", err)
	}

	// Load and verify it's the second config
	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("Failed to load overwritten config: %v", err)
	}
	if loaded.APIEndpoint != "http://second.api" {
		t.Errorf("Expected second API endpoint, got '%s'", loaded.APIEndpoint)
	}
	if loaded.Token != "token-2" {
		t.Errorf("Expected second token, got '%s'", loaded.Token)
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	// Verify what the defaults are when loading from a non-existent path
	cfg, _ := LoadFrom("/nonexistent")

	defaults := map[string]string{
		"APIEndpoint":  "http://localhost:8080",
		"Region":       "fsn1",
		"OutputFormat": "table",
	}

	if cfg.APIEndpoint != defaults["APIEndpoint"] {
		t.Errorf("Default APIEndpoint: got '%s', want '%s'", cfg.APIEndpoint, defaults["APIEndpoint"])
	}
	if cfg.Region != defaults["Region"] {
		t.Errorf("Default Region: got '%s', want '%s'", cfg.Region, defaults["Region"])
	}
	if cfg.OutputFormat != defaults["OutputFormat"] {
		t.Errorf("Default OutputFormat: got '%s', want '%s'", cfg.OutputFormat, defaults["OutputFormat"])
	}
}
