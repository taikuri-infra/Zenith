package apply

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dotechhq/zenith/cli/cmd/export"
)

func TestApplyManifestData_YAML(t *testing.T) {
	data := []byte(`apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: test-app
spec:
  image: nginx:latest
  replicas: 2
  port: 8080
`)

	manifest, err := ApplyManifestData(data, "yaml")
	if err != nil {
		t.Fatalf("ApplyManifestData failed: %v", err)
	}

	if manifest.Kind != "App" {
		t.Errorf("Expected kind 'App', got '%s'", manifest.Kind)
	}
	if manifest.Metadata.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got '%s'", manifest.Metadata.Name)
	}
}

func TestApplyManifestData_JSON(t *testing.T) {
	data := []byte(`{
  "apiVersion": "zenith.dev/v1alpha1",
  "kind": "Database",
  "metadata": {"name": "my-db"},
  "spec": {"engine": "postgresql", "version": "16"}
}`)

	manifest, err := ApplyManifestData(data, "json")
	if err != nil {
		t.Fatalf("ApplyManifestData failed: %v", err)
	}

	if manifest.Kind != "Database" {
		t.Errorf("Expected kind 'Database', got '%s'", manifest.Kind)
	}
	if manifest.Spec["engine"] != "postgresql" {
		t.Errorf("Expected engine 'postgresql', got '%v'", manifest.Spec["engine"])
	}
}

func TestApplyManifestData_InvalidJSON(t *testing.T) {
	data := []byte(`{invalid json`)

	_, err := ApplyManifestData(data, "json")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestApplyCollectsManifestsFromDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create app manifest
	appContent := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: web-app
spec:
  image: nginx:latest
  replicas: 2
  port: 8080
`
	appsDir := filepath.Join(dir, "apps")
	os.MkdirAll(appsDir, 0755)
	os.WriteFile(filepath.Join(appsDir, "web-app.yaml"), []byte(appContent), 0644)

	// Create database manifest
	dbContent := `apiVersion: zenith.dev/v1alpha1
kind: Database
metadata:
  name: my-db
spec:
  engine: postgresql
  version: "16"
  storage: 20Gi
`
	dbsDir := filepath.Join(dir, "databases")
	os.MkdirAll(dbsDir, 0755)
	os.WriteFile(filepath.Join(dbsDir, "my-db.yaml"), []byte(dbContent), 0644)

	// Use the export package's CollectManifests to verify
	manifests, err := export.CollectManifests(dir)
	if err != nil {
		t.Fatalf("CollectManifests failed: %v", err)
	}

	if len(manifests) != 2 {
		t.Fatalf("Expected 2 manifests, got %d", len(manifests))
	}

	// Verify manifest kinds
	kinds := make(map[string]bool)
	for _, m := range manifests {
		kinds[m.Kind] = true
	}

	if !kinds["App"] {
		t.Error("Expected App manifest")
	}
	if !kinds["Database"] {
		t.Error("Expected Database manifest")
	}
}

func TestApplyCollectsManifestsFromFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "app.yaml")

	content := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: single-app
spec:
  image: nginx:latest
`

	os.WriteFile(filePath, []byte(content), 0644)

	manifest, err := export.ParseManifestFile(filePath)
	if err != nil {
		t.Fatalf("ParseManifestFile failed: %v", err)
	}

	if manifest.Kind != "App" {
		t.Errorf("Expected kind 'App', got '%s'", manifest.Kind)
	}
	if manifest.Metadata.Name != "single-app" {
		t.Errorf("Expected name 'single-app', got '%s'", manifest.Metadata.Name)
	}
}

func TestApplyResult_Types(t *testing.T) {
	// Test that ApplyResult properly represents different outcomes
	results := []ApplyResult{
		{Name: "app1", Kind: "App", Action: "created"},
		{Name: "app2", Kind: "App", Action: "updated"},
		{Name: "app3", Kind: "App", Action: "unchanged"},
		{Name: "app4", Kind: "App", Action: "failed", Error: nil},
	}

	actions := make(map[string]int)
	for _, r := range results {
		actions[r.Action]++
	}

	if actions["created"] != 1 {
		t.Errorf("Expected 1 created, got %d", actions["created"])
	}
	if actions["updated"] != 1 {
		t.Errorf("Expected 1 updated, got %d", actions["updated"])
	}
	if actions["unchanged"] != 1 {
		t.Errorf("Expected 1 unchanged, got %d", actions["unchanged"])
	}
	if actions["failed"] != 1 {
		t.Errorf("Expected 1 failed, got %d", actions["failed"])
	}
}

func TestApplySkipsNonManifestFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a non-manifest file
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0644)
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[config]"), 0644)

	// Create one actual manifest
	content := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: real-app
spec:
  image: nginx:latest
`
	os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(content), 0644)

	manifests, err := export.CollectManifests(dir)
	if err != nil {
		t.Fatalf("CollectManifests failed: %v", err)
	}

	if len(manifests) != 1 {
		t.Fatalf("Expected 1 manifest (non-yaml/json skipped), got %d", len(manifests))
	}
}
