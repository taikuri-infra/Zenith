package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProject_AllLanguages(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		wantLang   string
		wantFrame  string
		wantPort   int
		wantDetect string
	}{
		{
			name:       "Dockerfile",
			files:      map[string]string{"Dockerfile": "FROM alpine"},
			wantLang:   "Docker",
			wantPort:   8080,
			wantDetect: "Dockerfile",
		},
		{
			name:       "Go",
			files:      map[string]string{"go.mod": "module test"},
			wantLang:   "Go",
			wantPort:   8080,
			wantDetect: "go.mod",
		},
		{
			name:       "Node.js",
			files:      map[string]string{"package.json": "{}"},
			wantLang:   "Node.js",
			wantPort:   3000,
			wantDetect: "package.json",
		},
		{
			name:       "Python requirements.txt",
			files:      map[string]string{"requirements.txt": "flask"},
			wantLang:   "Python",
			wantPort:   5000,
			wantDetect: "requirements.txt",
		},
		{
			name:       "Python Pipfile",
			files:      map[string]string{"Pipfile": "[packages]"},
			wantLang:   "Python",
			wantFrame:  "Pipenv",
			wantPort:   5000,
			wantDetect: "Pipfile",
		},
		{
			name:       "Python pyproject.toml",
			files:      map[string]string{"pyproject.toml": "[tool.poetry]"},
			wantLang:   "Python",
			wantPort:   5000,
			wantDetect: "pyproject.toml",
		},
		{
			name:       "Ruby",
			files:      map[string]string{"Gemfile": "source 'https://rubygems.org'"},
			wantLang:   "Ruby",
			wantFrame:  "Rails",
			wantPort:   3000,
			wantDetect: "Gemfile",
		},
		{
			name:       "Java Maven",
			files:      map[string]string{"pom.xml": "<project></project>"},
			wantLang:   "Java",
			wantFrame:  "Maven",
			wantPort:   8080,
			wantDetect: "pom.xml",
		},
		{
			name:       "Java Gradle",
			files:      map[string]string{"build.gradle": "apply plugin: 'java'"},
			wantLang:   "Java",
			wantFrame:  "Gradle",
			wantPort:   8080,
			wantDetect: "build.gradle",
		},
		{
			name:       "Rust",
			files:      map[string]string{"Cargo.toml": "[package]"},
			wantLang:   "Rust",
			wantPort:   8080,
			wantDetect: "Cargo.toml",
		},
		{
			name:       "Elixir",
			files:      map[string]string{"mix.exs": "defmodule MyApp do"},
			wantLang:   "Elixir",
			wantFrame:  "Mix",
			wantPort:   4000,
			wantDetect: "mix.exs",
		},
		{
			name:       "PHP",
			files:      map[string]string{"composer.json": "{}"},
			wantLang:   "PHP",
			wantPort:   8080,
			wantDetect: "composer.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
			}

			pt := detectProject(dir)
			if pt == nil {
				t.Fatalf("Expected project detection for %s", tt.name)
			}
			if pt.Language != tt.wantLang {
				t.Errorf("Expected language '%s', got '%s'", tt.wantLang, pt.Language)
			}
			if tt.wantFrame != "" && pt.Framework != tt.wantFrame {
				t.Errorf("Expected framework '%s', got '%s'", tt.wantFrame, pt.Framework)
			}
			if pt.Port != tt.wantPort {
				t.Errorf("Expected port %d, got %d", tt.wantPort, pt.Port)
			}
			if pt.DetectedBy != tt.wantDetect {
				t.Errorf("Expected DetectedBy '%s', got '%s'", tt.wantDetect, pt.DetectedBy)
			}
		})
	}
}

func TestDetectProject_NodeJSFrameworks(t *testing.T) {
	tests := []struct {
		name      string
		extraFile string
		wantFrame string
		wantPort  int
	}{
		{
			name:      "Next.js via next.config.js",
			extraFile: "next.config.js",
			wantFrame: "Next.js",
			wantPort:  3000,
		},
		{
			name:      "Next.js via next.config.mjs",
			extraFile: "next.config.mjs",
			wantFrame: "Next.js",
			wantPort:  3000,
		},
		{
			name:      "Nuxt",
			extraFile: "nuxt.config.ts",
			wantFrame: "Nuxt",
			wantPort:  3000,
		},
		{
			name:      "Vite",
			extraFile: "vite.config.ts",
			wantFrame: "Vite",
			wantPort:  5173,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
			os.WriteFile(filepath.Join(dir, tt.extraFile), []byte("// config"), 0644)

			pt := detectProject(dir)
			if pt == nil {
				t.Fatal("Expected project detection")
			}
			if pt.Framework != tt.wantFrame {
				t.Errorf("Expected framework '%s', got '%s'", tt.wantFrame, pt.Framework)
			}
			if pt.Port != tt.wantPort {
				t.Errorf("Expected port %d, got %d", tt.wantPort, pt.Port)
			}
		})
	}
}

func TestDetectProject_Empty(t *testing.T) {
	dir := t.TempDir()

	pt := detectProject(dir)
	if pt != nil {
		t.Error("Expected nil for empty directory")
	}
}

func TestDetectProject_NoRecognizableFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# My project"), 0644)
	os.WriteFile(filepath.Join(dir, "data.csv"), []byte("a,b,c"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("notes"), 0644)

	pt := detectProject(dir)
	if pt != nil {
		t.Errorf("Expected nil for unrecognizable project, got language=%s", pt.Language)
	}
}

func TestDetectProject_DockerfilePriority(t *testing.T) {
	// When both Dockerfile and go.mod exist, Dockerfile should be detected first
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM golang"), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	pt := detectProject(dir)
	if pt == nil {
		t.Fatal("Expected project detection")
	}
	if pt.Language != "Docker" {
		t.Errorf("Expected Dockerfile to take priority, got language '%s'", pt.Language)
	}
	if pt.DetectedBy != "Dockerfile" {
		t.Errorf("Expected DetectedBy 'Dockerfile', got '%s'", pt.DetectedBy)
	}
}

func TestDetectProject_MultipleIndicators(t *testing.T) {
	// go.mod + Cargo.toml - go.mod comes first in detection order
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0644)

	pt := detectProject(dir)
	if pt == nil {
		t.Fatal("Expected project detection")
	}
	if pt.Language != "Go" {
		t.Errorf("Expected Go (first in detection order), got '%s'", pt.Language)
	}
}

func TestProjectType_StructFields(t *testing.T) {
	pt := ProjectType{
		Language:   "Go",
		Framework:  "Fiber",
		DetectedBy: "go.mod",
		Port:       8080,
	}

	if pt.Language != "Go" {
		t.Errorf("Expected Language 'Go', got '%s'", pt.Language)
	}
	if pt.Framework != "Fiber" {
		t.Errorf("Expected Framework 'Fiber', got '%s'", pt.Framework)
	}
	if pt.DetectedBy != "go.mod" {
		t.Errorf("Expected DetectedBy 'go.mod', got '%s'", pt.DetectedBy)
	}
	if pt.Port != 8080 {
		t.Errorf("Expected Port 8080, got %d", pt.Port)
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
		{"UPPERCASE", "uppercase"},
		{"with spaces and_underscores", "with-spaces-and-underscores"},
		{"app---name", "app---name"},
		{"123-numeric", "123-numeric"},
		{"special$%^chars", "specialchars"},
		{"", ""},
		{"a", "a"},
		{"-leading-dash", "-leading-dash"},
		{"trailing-dash-", "trailing-dash-"},
		{"MiXeD_CaSe App!!", "mixed-case-app"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectProject_PlainNodeJS(t *testing.T) {
	// Verify that without framework config files, Node.js has no framework set
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)

	pt := detectProject(dir)
	if pt == nil {
		t.Fatal("Expected Node.js project detection")
	}
	if pt.Language != "Node.js" {
		t.Errorf("Expected language 'Node.js', got '%s'", pt.Language)
	}
	if pt.Framework != "" {
		t.Errorf("Expected empty framework for plain Node.js, got '%s'", pt.Framework)
	}
	if pt.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", pt.Port)
	}
}
