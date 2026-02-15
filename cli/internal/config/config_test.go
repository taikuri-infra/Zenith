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

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Error("Expected non-empty config path")
	}
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("Expected config.yaml, got '%s'", filepath.Base(path))
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
