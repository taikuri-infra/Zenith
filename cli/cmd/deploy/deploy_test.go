package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProject_Go(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	pt := detectProject(dir)
	if pt == nil {
		t.Fatal("Expected Go project detection")
	}
	if pt.Language != "Go" {
		t.Errorf("Expected language 'Go', got '%s'", pt.Language)
	}
	if pt.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", pt.Port)
	}
}

func TestDetectProject_NodeJS(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)

	pt := detectProject(dir)
	if pt == nil {
		t.Fatal("Expected Node.js project detection")
	}
	if pt.Language != "Node.js" {
		t.Errorf("Expected language 'Node.js', got '%s'", pt.Language)
	}
	if pt.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", pt.Port)
	}
}

func TestDetectProject_NextJS(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "next.config.js"), []byte("module.exports={}"), 0644)

	pt := detectProject(dir)
	if pt == nil {
		t.Fatal("Expected Next.js project detection")
	}
	if pt.Framework != "Next.js" {
		t.Errorf("Expected framework 'Next.js', got '%s'", pt.Framework)
	}
}

func TestDetectProject_Python(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask"), 0644)

	pt := detectProject(dir)
	if pt == nil {
		t.Fatal("Expected Python project detection")
	}
	if pt.Language != "Python" {
		t.Errorf("Expected language 'Python', got '%s'", pt.Language)
	}
	if pt.Port != 5000 {
		t.Errorf("Expected port 5000, got %d", pt.Port)
	}
}

func TestDetectProject_Rust(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0644)

	pt := detectProject(dir)
	if pt == nil {
		t.Fatal("Expected Rust project detection")
	}
	if pt.Language != "Rust" {
		t.Errorf("Expected language 'Rust', got '%s'", pt.Language)
	}
}

func TestDetectProject_Dockerfile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0644)

	pt := detectProject(dir)
	if pt == nil {
		t.Fatal("Expected Dockerfile detection")
	}
	if pt.Language != "Docker" {
		t.Errorf("Expected language 'Docker', got '%s'", pt.Language)
	}
}

func TestDetectProject_Empty(t *testing.T) {
	dir := t.TempDir()

	pt := detectProject(dir)
	if pt != nil {
		t.Error("Expected nil for empty directory")
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My App", "my-app"},
		{"my_app", "my-app"},
		{"MY-APP-123", "my-app-123"},
		{"app@v2!", "appv2"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		result := sanitizeName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
