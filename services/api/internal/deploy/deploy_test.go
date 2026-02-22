package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "zenith-detect-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func touchFile(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("Failed to create file %s: %v", name, err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write file %s: %v", name, err)
	}
}

// --- DetectFramework tests ---

func TestDetectDockerfile(t *testing.T) {
	dir := createTempDir(t)
	touchFile(t, dir, "Dockerfile")
	touchFile(t, dir, "go.mod") // Dockerfile should win over go.mod

	fw := DetectFramework(dir)
	if fw != entities.FrameworkDockerfile {
		t.Errorf("Expected 'dockerfile', got '%s'", fw)
	}
}

func TestDetectNextJS(t *testing.T) {
	dir := createTempDir(t)
	touchFile(t, dir, "next.config.js")
	touchFile(t, dir, "package.json")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkNextJS {
		t.Errorf("Expected 'nextjs', got '%s'", fw)
	}
}

func TestDetectNextJSFromTS(t *testing.T) {
	dir := createTempDir(t)
	touchFile(t, dir, "next.config.ts")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkNextJS {
		t.Errorf("Expected 'nextjs', got '%s'", fw)
	}
}

func TestDetectNextJSFromPackageJSON(t *testing.T) {
	dir := createTempDir(t)
	writeFile(t, dir, "package.json", `{
		"dependencies": {
			"next": "14.0.0",
			"react": "18.0.0"
		}
	}`)

	fw := DetectFramework(dir)
	if fw != entities.FrameworkNextJS {
		t.Errorf("Expected 'nextjs' (from package.json dep), got '%s'", fw)
	}
}

func TestDetectGo(t *testing.T) {
	dir := createTempDir(t)
	touchFile(t, dir, "go.mod")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkGo {
		t.Errorf("Expected 'go', got '%s'", fw)
	}
}

func TestDetectPython(t *testing.T) {
	dir := createTempDir(t)
	writeFile(t, dir, "requirements.txt", "fastapi\nuvicorn\n")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkPython {
		t.Errorf("Expected 'python', got '%s'", fw)
	}
}

func TestDetectFlask(t *testing.T) {
	dir := createTempDir(t)
	writeFile(t, dir, "requirements.txt", "flask\ngunicorn\n")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkFlask {
		t.Errorf("Expected 'flask', got '%s'", fw)
	}
}

func TestDetectDjango(t *testing.T) {
	dir := createTempDir(t)
	touchFile(t, dir, "manage.py")
	writeFile(t, dir, "requirements.txt", "django\n")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkDjango {
		t.Errorf("Expected 'django', got '%s'", fw)
	}
}

func TestDetectRails(t *testing.T) {
	dir := createTempDir(t)
	touchFile(t, dir, "Gemfile")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkRails {
		t.Errorf("Expected 'rails', got '%s'", fw)
	}
}

func TestDetectExpress(t *testing.T) {
	dir := createTempDir(t)
	writeFile(t, dir, "package.json", `{
		"dependencies": {
			"express": "4.18.0"
		}
	}`)

	fw := DetectFramework(dir)
	if fw != entities.FrameworkExpress {
		t.Errorf("Expected 'express', got '%s'", fw)
	}
}

func TestDetectStatic(t *testing.T) {
	dir := createTempDir(t)
	touchFile(t, dir, "index.html")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkStatic {
		t.Errorf("Expected 'static', got '%s'", fw)
	}
}

func TestDetectUnknown(t *testing.T) {
	dir := createTempDir(t)
	touchFile(t, dir, "README.md")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkUnknown {
		t.Errorf("Expected 'unknown', got '%s'", fw)
	}
}

func TestDetectEmptyDir(t *testing.T) {
	dir := createTempDir(t)
	fw := DetectFramework(dir)
	if fw != entities.FrameworkUnknown {
		t.Errorf("Expected 'unknown' for empty dir, got '%s'", fw)
	}
}

func TestDetectPriority_DockerfileWins(t *testing.T) {
	dir := createTempDir(t)
	touchFile(t, dir, "Dockerfile")
	touchFile(t, dir, "next.config.js")
	touchFile(t, dir, "go.mod")
	touchFile(t, dir, "Gemfile")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkDockerfile {
		t.Errorf("Dockerfile should always win, got '%s'", fw)
	}
}

func TestDetectPriority_NextJSOverExpress(t *testing.T) {
	dir := createTempDir(t)
	touchFile(t, dir, "next.config.mjs")
	touchFile(t, dir, "package.json")

	fw := DetectFramework(dir)
	if fw != entities.FrameworkNextJS {
		t.Errorf("next.config should prioritize over package.json, got '%s'", fw)
	}
}

// --- DetectFrameworkFromFiles tests ---

func TestDetectFromFiles_Go(t *testing.T) {
	fw := DetectFrameworkFromFiles([]string{"go.mod", "go.sum", "main.go", "README.md"})
	if fw != entities.FrameworkGo {
		t.Errorf("Expected 'go', got '%s'", fw)
	}
}

func TestDetectFromFiles_NextJS(t *testing.T) {
	fw := DetectFrameworkFromFiles([]string{"package.json", "next.config.js", "tsconfig.json"})
	if fw != entities.FrameworkNextJS {
		t.Errorf("Expected 'nextjs', got '%s'", fw)
	}
}

func TestDetectFromFiles_Unknown(t *testing.T) {
	fw := DetectFrameworkFromFiles([]string{"README.md", "LICENSE"})
	if fw != entities.FrameworkUnknown {
		t.Errorf("Expected 'unknown', got '%s'", fw)
	}
}

// --- GenerateDockerfile tests ---

func TestGenerateDockerfileNextJS(t *testing.T) {
	df, err := GenerateDockerfile(entities.FrameworkNextJS, "myapp", 3000)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(df, "node:20-alpine") {
		t.Error("Expected node:20-alpine base image")
	}
	if !strings.Contains(df, "npm run build") {
		t.Error("Expected npm run build")
	}
	if !strings.Contains(df, "3000") {
		t.Error("Expected port 3000")
	}
}

func TestGenerateDockerfileGo(t *testing.T) {
	df, err := GenerateDockerfile(entities.FrameworkGo, "myapi", 8080)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(df, "golang:1.22-alpine") {
		t.Error("Expected golang base image")
	}
	if !strings.Contains(df, "CGO_ENABLED=0") {
		t.Error("Expected CGO disabled for static build")
	}
}

func TestGenerateDockerfilePython(t *testing.T) {
	df, err := GenerateDockerfile(entities.FrameworkPython, "myapp", 8000)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(df, "python:3.12-slim") {
		t.Error("Expected python slim base image")
	}
}

func TestGenerateDockerfileDjango(t *testing.T) {
	df, err := GenerateDockerfile(entities.FrameworkDjango, "myapp", 8000)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(df, "gunicorn") {
		t.Error("Expected gunicorn in Django Dockerfile")
	}
	if !strings.Contains(df, "collectstatic") {
		t.Error("Expected collectstatic step")
	}
}

func TestGenerateDockerfileRails(t *testing.T) {
	df, err := GenerateDockerfile(entities.FrameworkRails, "myapp", 3000)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(df, "ruby:3.3-slim") {
		t.Error("Expected ruby slim base image")
	}
	if !strings.Contains(df, "puma") {
		t.Error("Expected puma in Rails Dockerfile")
	}
}

func TestGenerateDockerfileExpress(t *testing.T) {
	df, err := GenerateDockerfile(entities.FrameworkExpress, "myapp", 3000)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(df, "node:20-alpine") {
		t.Error("Expected node base image")
	}
}

func TestGenerateDockerfileStatic(t *testing.T) {
	df, err := GenerateDockerfile(entities.FrameworkStatic, "myapp", 80)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(df, "nginx") {
		t.Error("Expected nginx in static Dockerfile")
	}
}

func TestGenerateDockerfileFlask(t *testing.T) {
	df, err := GenerateDockerfile(entities.FrameworkFlask, "myapp", 5000)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(df, "gunicorn") {
		t.Error("Expected gunicorn in Flask Dockerfile")
	}
}

func TestGenerateDockerfileForDockerfileFramework(t *testing.T) {
	_, err := GenerateDockerfile(entities.FrameworkDockerfile, "myapp", 8080)
	if err == nil {
		t.Error("Expected error when framework is 'dockerfile'")
	}
}

func TestGenerateDockerfileUnknown(t *testing.T) {
	_, err := GenerateDockerfile(entities.FrameworkUnknown, "myapp", 8080)
	if err == nil {
		t.Error("Expected error for unknown framework")
	}
}

func TestGenerateDockerfileAllHaveUserDirective(t *testing.T) {
	frameworks := []entities.Framework{
		entities.FrameworkNextJS,
		entities.FrameworkGo,
		entities.FrameworkPython,
		entities.FrameworkDjango,
		entities.FrameworkFlask,
		entities.FrameworkRails,
		entities.FrameworkExpress,
	}
	for _, fw := range frameworks {
		df, err := GenerateDockerfile(fw, "test", 8080)
		if err != nil {
			t.Fatalf("Error for framework %s: %v", fw, err)
		}
		if !strings.Contains(df, "USER 1001") {
			t.Errorf("Framework %s Dockerfile missing USER directive (non-root)", fw)
		}
	}
}
