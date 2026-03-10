package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	// When no config file exists, Load returns defaults
	orig := os.Getenv("HOME")
	t.Setenv("HOME", t.TempDir())
	defer os.Setenv("HOME", orig)

	cfg := Load()
	if cfg.APIBaseURL != DefaultBaseURL {
		t.Errorf("Expected default base URL %s, got %s", DefaultBaseURL, cfg.APIBaseURL)
	}
	if cfg.AccessToken != "" {
		t.Error("Expected empty access token for fresh config")
	}
	if cfg.IsLoggedIn() {
		t.Error("Expected IsLoggedIn() == false for fresh config")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	orig := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer os.Setenv("HOME", orig)

	cfg := &Config{
		APIBaseURL:   "https://api.test.example.com",
		AccessToken:  "test-token-abc",
		RefreshToken: "refresh-token-xyz",
		ProjectID:    "proj-123",
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(ConfigPath())
	if err != nil {
		t.Fatalf("Config file not found: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", perm)
	}

	// Verify directory permissions
	dirInfo, err := os.Stat(filepath.Dir(ConfigPath()))
	if err != nil {
		t.Fatalf("Config dir not found: %v", err)
	}
	dirPerm := dirInfo.Mode().Perm()
	if dirPerm != 0700 {
		t.Errorf("Expected dir permissions 0700, got %o", dirPerm)
	}

	// Load and verify
	loaded := Load()
	if loaded.APIBaseURL != cfg.APIBaseURL {
		t.Errorf("APIBaseURL mismatch: %s vs %s", loaded.APIBaseURL, cfg.APIBaseURL)
	}
	if loaded.AccessToken != cfg.AccessToken {
		t.Errorf("AccessToken mismatch: %s vs %s", loaded.AccessToken, cfg.AccessToken)
	}
	if loaded.RefreshToken != cfg.RefreshToken {
		t.Errorf("RefreshToken mismatch: %s vs %s", loaded.RefreshToken, cfg.RefreshToken)
	}
	if loaded.ProjectID != cfg.ProjectID {
		t.Errorf("ProjectID mismatch: %s vs %s", loaded.ProjectID, cfg.ProjectID)
	}
}

func TestIsLoggedIn(t *testing.T) {
	cfg := &Config{APIBaseURL: DefaultBaseURL}
	if cfg.IsLoggedIn() {
		t.Error("Expected not logged in with empty token")
	}

	cfg.AccessToken = "some-token"
	if !cfg.IsLoggedIn() {
		t.Error("Expected logged in with token set")
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	tmp := t.TempDir()
	orig := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer os.Setenv("HOME", orig)

	// Write garbage to config file
	dir := filepath.Join(tmp, ".zenith")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "config.json"), []byte("{invalid json"), 0600)

	cfg := Load()
	// Should still return a valid config with defaults
	if cfg.APIBaseURL != DefaultBaseURL {
		t.Errorf("Expected default URL on corrupt file, got %s", cfg.APIBaseURL)
	}
}

func TestLoadEmptyBaseURL(t *testing.T) {
	tmp := t.TempDir()
	orig := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer os.Setenv("HOME", orig)

	// Write config with empty base URL
	dir := filepath.Join(tmp, ".zenith")
	os.MkdirAll(dir, 0700)
	data, _ := json.Marshal(Config{APIBaseURL: "", AccessToken: "tok"})
	os.WriteFile(filepath.Join(dir, "config.json"), data, 0600)

	cfg := Load()
	if cfg.APIBaseURL != DefaultBaseURL {
		t.Errorf("Expected default URL when empty, got %s", cfg.APIBaseURL)
	}
	if cfg.AccessToken != "tok" {
		t.Errorf("Expected preserved token, got %s", cfg.AccessToken)
	}
}

func TestConfigJSON(t *testing.T) {
	cfg := &Config{
		APIBaseURL:   "https://api.example.com",
		AccessToken:  "access",
		RefreshToken: "refresh",
		ProjectID:    "proj-1",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Config
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.APIBaseURL != cfg.APIBaseURL {
		t.Errorf("APIBaseURL mismatch after roundtrip")
	}
	if parsed.ProjectID != cfg.ProjectID {
		t.Errorf("ProjectID mismatch after roundtrip")
	}
}

func TestProjectIDOmitted(t *testing.T) {
	cfg := &Config{
		APIBaseURL:  DefaultBaseURL,
		AccessToken: "tok",
	}

	data, _ := json.Marshal(cfg)
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if _, ok := raw["project_id"]; ok {
		t.Error("Expected project_id to be omitted when empty")
	}
}
