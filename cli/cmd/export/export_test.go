package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMarshalManifest_YAML(t *testing.T) {
	manifest := ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: ManifestMetadata{
			Name: "web-app",
			Labels: map[string]string{
				"zenith.dev/project": "test",
			},
		},
		Spec: map[string]interface{}{
			"image":    "nginx:latest",
			"replicas": 2,
			"port":     8080,
		},
	}

	data, err := MarshalManifest(manifest, "yaml")
	if err != nil {
		t.Fatalf("MarshalManifest failed: %v", err)
	}

	// Parse it back
	var parsed ZenithManifest
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML output: %v", err)
	}

	if parsed.APIVersion != "zenith.dev/v1alpha1" {
		t.Errorf("Expected apiVersion 'zenith.dev/v1alpha1', got '%s'", parsed.APIVersion)
	}
	if parsed.Kind != "App" {
		t.Errorf("Expected kind 'App', got '%s'", parsed.Kind)
	}
	if parsed.Metadata.Name != "web-app" {
		t.Errorf("Expected name 'web-app', got '%s'", parsed.Metadata.Name)
	}
}

func TestMarshalManifest_JSON(t *testing.T) {
	manifest := ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Database",
		Metadata: ManifestMetadata{
			Name: "my-db",
		},
		Spec: map[string]interface{}{
			"engine":  "postgresql",
			"version": "16",
		},
	}

	data, err := MarshalManifest(manifest, "json")
	if err != nil {
		t.Fatalf("MarshalManifest failed: %v", err)
	}

	var parsed ZenithManifest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if parsed.Kind != "Database" {
		t.Errorf("Expected kind 'Database', got '%s'", parsed.Kind)
	}
	if parsed.Spec["engine"] != "postgresql" {
		t.Errorf("Expected engine 'postgresql', got '%v'", parsed.Spec["engine"])
	}
}

func TestParseManifestFile_YAML(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "app.yaml")

	content := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: test-app
spec:
  image: nginx:latest
  replicas: 3
  port: 8080
`

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	manifest, err := ParseManifestFile(filePath)
	if err != nil {
		t.Fatalf("ParseManifestFile failed: %v", err)
	}

	if manifest.APIVersion != "zenith.dev/v1alpha1" {
		t.Errorf("Expected apiVersion 'zenith.dev/v1alpha1', got '%s'", manifest.APIVersion)
	}
	if manifest.Kind != "App" {
		t.Errorf("Expected kind 'App', got '%s'", manifest.Kind)
	}
	if manifest.Metadata.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got '%s'", manifest.Metadata.Name)
	}
	if manifest.Spec["image"] != "nginx:latest" {
		t.Errorf("Expected image 'nginx:latest', got '%v'", manifest.Spec["image"])
	}
}

func TestParseManifestFile_JSON(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "db.json")

	content := `{
  "apiVersion": "zenith.dev/v1alpha1",
  "kind": "Database",
  "metadata": {
    "name": "my-db"
  },
  "spec": {
    "engine": "postgresql",
    "version": "16"
  }
}`

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	manifest, err := ParseManifestFile(filePath)
	if err != nil {
		t.Fatalf("ParseManifestFile failed: %v", err)
	}

	if manifest.Kind != "Database" {
		t.Errorf("Expected kind 'Database', got '%s'", manifest.Kind)
	}
	if manifest.Spec["engine"] != "postgresql" {
		t.Errorf("Expected engine 'postgresql', got '%v'", manifest.Spec["engine"])
	}
}

func TestParseManifestFile_NotFound(t *testing.T) {
	_, err := ParseManifestFile("/nonexistent/path/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestCollectManifests(t *testing.T) {
	dir := t.TempDir()

	// Create subdirectories
	appsDir := filepath.Join(dir, "apps")
	dbsDir := filepath.Join(dir, "databases")
	os.MkdirAll(appsDir, 0755)
	os.MkdirAll(dbsDir, 0755)

	// Write app manifest
	appContent := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: web-app
spec:
  image: nginx:latest
`
	os.WriteFile(filepath.Join(appsDir, "web-app.yaml"), []byte(appContent), 0644)

	// Write database manifest
	dbContent := `apiVersion: zenith.dev/v1alpha1
kind: Database
metadata:
  name: my-db
spec:
  engine: postgresql
`
	os.WriteFile(filepath.Join(dbsDir, "my-db.yaml"), []byte(dbContent), 0644)

	// Write a non-manifest file (should be skipped)
	os.WriteFile(filepath.Join(dir, "README.txt"), []byte("not a manifest"), 0644)

	manifests, err := CollectManifests(dir)
	if err != nil {
		t.Fatalf("CollectManifests failed: %v", err)
	}

	if len(manifests) != 2 {
		t.Fatalf("Expected 2 manifests, got %d", len(manifests))
	}

	// Verify we got both kinds
	kinds := make(map[string]bool)
	for _, m := range manifests {
		kinds[m.Kind] = true
	}

	if !kinds["App"] {
		t.Error("Expected App manifest to be collected")
	}
	if !kinds["Database"] {
		t.Error("Expected Database manifest to be collected")
	}
}

func TestCollectManifests_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	manifests, err := CollectManifests(dir)
	if err != nil {
		t.Fatalf("CollectManifests failed: %v", err)
	}

	if len(manifests) != 0 {
		t.Errorf("Expected 0 manifests from empty dir, got %d", len(manifests))
	}
}

func TestCollectManifests_MixedFormats(t *testing.T) {
	dir := t.TempDir()

	yamlContent := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: yaml-app
spec:
  image: nginx:latest
`
	os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(yamlContent), 0644)

	jsonContent := `{
  "apiVersion": "zenith.dev/v1alpha1",
  "kind": "App",
  "metadata": {"name": "json-app"},
  "spec": {"image": "redis:latest"}
}`
	os.WriteFile(filepath.Join(dir, "app.json"), []byte(jsonContent), 0644)

	manifests, err := CollectManifests(dir)
	if err != nil {
		t.Fatalf("CollectManifests failed: %v", err)
	}

	if len(manifests) != 2 {
		t.Fatalf("Expected 2 manifests, got %d", len(manifests))
	}
}

func TestParseManifestData_YAML(t *testing.T) {
	data := []byte(`apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: data-app
spec:
  image: test:v1
`)

	manifest, err := ParseManifestData(data, "test.yaml")
	if err != nil {
		t.Fatalf("ParseManifestData failed: %v", err)
	}

	if manifest.Kind != "App" {
		t.Errorf("Expected kind 'App', got '%s'", manifest.Kind)
	}
	if manifest.Metadata.Name != "data-app" {
		t.Errorf("Expected name 'data-app', got '%s'", manifest.Metadata.Name)
	}
}

func TestParseManifestData_JSON(t *testing.T) {
	data := []byte(`{"apiVersion":"zenith.dev/v1alpha1","kind":"Database","metadata":{"name":"data-db"},"spec":{"engine":"redis"}}`)

	manifest, err := ParseManifestData(data, "test.json")
	if err != nil {
		t.Fatalf("ParseManifestData failed: %v", err)
	}

	if manifest.Kind != "Database" {
		t.Errorf("Expected kind 'Database', got '%s'", manifest.Kind)
	}
	if manifest.Spec["engine"] != "redis" {
		t.Errorf("Expected engine 'redis', got '%v'", manifest.Spec["engine"])
	}
}
