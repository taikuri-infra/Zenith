package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestMarshalManifest_ComplexSpec(t *testing.T) {
	manifest := ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: ManifestMetadata{
			Name:      "complex-app",
			Namespace: "production",
			Labels: map[string]string{
				"zenith.dev/project": "test",
				"env":                "production",
				"team":               "platform",
			},
		},
		Spec: map[string]interface{}{
			"image":    "myapp:v2.1.0",
			"replicas": 3,
			"port":     8080,
			"env": map[string]interface{}{
				"DATABASE_URL": "postgres://localhost/mydb",
				"REDIS_URL":    "redis://localhost:6379",
			},
			"resources": map[string]interface{}{
				"cpu":    "500m",
				"memory": "256Mi",
			},
			"ports": []interface{}{8080, 8443, 9090},
		},
	}

	// Test YAML round-trip
	yamlData, err := MarshalManifest(manifest, "yaml")
	if err != nil {
		t.Fatalf("MarshalManifest YAML failed: %v", err)
	}

	var parsedYAML ZenithManifest
	if err := yaml.Unmarshal(yamlData, &parsedYAML); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	if parsedYAML.Metadata.Namespace != "production" {
		t.Errorf("Expected namespace 'production', got '%s'", parsedYAML.Metadata.Namespace)
	}
	if len(parsedYAML.Metadata.Labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(parsedYAML.Metadata.Labels))
	}
	envMap, ok := parsedYAML.Spec["env"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'env' to be a map")
	}
	if envMap["DATABASE_URL"] != "postgres://localhost/mydb" {
		t.Errorf("Expected DATABASE_URL, got '%v'", envMap["DATABASE_URL"])
	}

	// Test JSON round-trip
	jsonData, err := MarshalManifest(manifest, "json")
	if err != nil {
		t.Fatalf("MarshalManifest JSON failed: %v", err)
	}

	var parsedJSON ZenithManifest
	if err := json.Unmarshal(jsonData, &parsedJSON); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if parsedJSON.Metadata.Namespace != "production" {
		t.Errorf("Expected namespace 'production', got '%s'", parsedJSON.Metadata.Namespace)
	}
}

func TestMarshalManifest_YAMLJSONConsistency(t *testing.T) {
	manifest := ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: ManifestMetadata{
			Name: "consistency-app",
			Labels: map[string]string{
				"zenith.dev/project": "test",
			},
		},
		Spec: map[string]interface{}{
			"image":    "nginx:latest",
			"replicas": 2,
		},
	}

	yamlData, err := MarshalManifest(manifest, "yaml")
	if err != nil {
		t.Fatalf("YAML marshal failed: %v", err)
	}

	jsonData, err := MarshalManifest(manifest, "json")
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var fromYAML ZenithManifest
	if err := yaml.Unmarshal(yamlData, &fromYAML); err != nil {
		t.Fatalf("YAML unmarshal failed: %v", err)
	}

	var fromJSON ZenithManifest
	if err := json.Unmarshal(jsonData, &fromJSON); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Both should produce the same logical content
	if fromYAML.APIVersion != fromJSON.APIVersion {
		t.Errorf("APIVersion mismatch: yaml=%s, json=%s", fromYAML.APIVersion, fromJSON.APIVersion)
	}
	if fromYAML.Kind != fromJSON.Kind {
		t.Errorf("Kind mismatch: yaml=%s, json=%s", fromYAML.Kind, fromJSON.Kind)
	}
	if fromYAML.Metadata.Name != fromJSON.Metadata.Name {
		t.Errorf("Name mismatch: yaml=%s, json=%s", fromYAML.Metadata.Name, fromJSON.Metadata.Name)
	}
	if fromYAML.Spec["image"] != fromJSON.Spec["image"] {
		t.Errorf("Image mismatch: yaml=%v, json=%v", fromYAML.Spec["image"], fromJSON.Spec["image"])
	}
}

func TestMarshalManifest_FormatDefault(t *testing.T) {
	manifest := ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata:   ManifestMetadata{Name: "test"},
		Spec:       map[string]interface{}{"image": "nginx"},
	}

	// Unknown format should default to YAML
	data, err := MarshalManifest(manifest, "unknown-format")
	if err != nil {
		t.Fatalf("MarshalManifest with unknown format failed: %v", err)
	}

	// Verify it's valid YAML
	var parsed ZenithManifest
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Expected YAML output for unknown format, got invalid YAML: %v", err)
	}
	if parsed.Kind != "App" {
		t.Errorf("Expected kind 'App', got '%s'", parsed.Kind)
	}
}

func TestMarshalManifest_MetadataPreservation(t *testing.T) {
	manifest := ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: ManifestMetadata{
			Name:      "labeled-app",
			Namespace: "my-namespace",
			Labels: map[string]string{
				"zenith.dev/project": "demo",
				"app.kubernetes.io/name": "labeled-app",
				"app.kubernetes.io/version": "v1.0.0",
			},
		},
		Spec: map[string]interface{}{
			"image": "myapp:v1.0.0",
		},
	}

	data, err := MarshalManifest(manifest, "yaml")
	if err != nil {
		t.Fatalf("MarshalManifest failed: %v", err)
	}

	var parsed ZenithManifest
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Metadata.Name != "labeled-app" {
		t.Errorf("Expected name 'labeled-app', got '%s'", parsed.Metadata.Name)
	}
	if parsed.Metadata.Namespace != "my-namespace" {
		t.Errorf("Expected namespace 'my-namespace', got '%s'", parsed.Metadata.Namespace)
	}
	if len(parsed.Metadata.Labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(parsed.Metadata.Labels))
	}
	if parsed.Metadata.Labels["zenith.dev/project"] != "demo" {
		t.Errorf("Expected label 'zenith.dev/project'='demo', got '%s'", parsed.Metadata.Labels["zenith.dev/project"])
	}
	if parsed.Metadata.Labels["app.kubernetes.io/name"] != "labeled-app" {
		t.Errorf("Expected label 'app.kubernetes.io/name'='labeled-app', got '%s'", parsed.Metadata.Labels["app.kubernetes.io/name"])
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
	if !strings.Contains(err.Error(), "read file") {
		t.Errorf("Expected error to mention 'read file', got: %v", err)
	}
}

func TestParseManifestFile_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "bad.yaml")

	content := `
kind: App
metadata: {{{
  invalid: [yaml: broken
`
	os.WriteFile(filePath, []byte(content), 0644)

	_, err := ParseManifestFile(filePath)
	if err == nil {
		t.Error("Expected error for malformed YAML file")
	}
	if !strings.Contains(err.Error(), "parse YAML") {
		t.Errorf("Expected error to mention 'parse YAML', got: %v", err)
	}
}

func TestParseManifestFile_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "bad.json")

	os.WriteFile(filePath, []byte("{invalid json content"), 0644)

	_, err := ParseManifestFile(filePath)
	if err == nil {
		t.Error("Expected error for malformed JSON file")
	}
	if !strings.Contains(err.Error(), "parse JSON") {
		t.Errorf("Expected error to mention 'parse JSON', got: %v", err)
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

func TestCollectManifests_DeeplyNested(t *testing.T) {
	dir := t.TempDir()

	// Create deeply nested directories
	deepPath := filepath.Join(dir, "level1", "level2", "level3", "level4")
	os.MkdirAll(deepPath, 0755)

	content := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: deep-app
spec:
  image: alpine:latest
`
	os.WriteFile(filepath.Join(deepPath, "deep.yaml"), []byte(content), 0644)

	// Also one at root
	rootContent := `apiVersion: zenith.dev/v1alpha1
kind: Database
metadata:
  name: root-db
spec:
  engine: redis
`
	os.WriteFile(filepath.Join(dir, "root-db.yaml"), []byte(rootContent), 0644)

	manifests, err := CollectManifests(dir)
	if err != nil {
		t.Fatalf("CollectManifests failed: %v", err)
	}

	if len(manifests) != 2 {
		t.Fatalf("Expected 2 manifests from deeply nested dir, got %d", len(manifests))
	}

	names := make(map[string]bool)
	for _, m := range manifests {
		names[m.Metadata.Name] = true
	}
	if !names["deep-app"] {
		t.Error("Expected to find deep-app manifest")
	}
	if !names["root-db"] {
		t.Error("Expected to find root-db manifest")
	}
}

func TestCollectManifests_YMLExtension(t *testing.T) {
	dir := t.TempDir()

	content := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: yml-app
spec:
  image: nginx:latest
`
	os.WriteFile(filepath.Join(dir, "app.yml"), []byte(content), 0644)

	manifests, err := CollectManifests(dir)
	if err != nil {
		t.Fatalf("CollectManifests failed: %v", err)
	}

	if len(manifests) != 1 {
		t.Fatalf("Expected 1 manifest with .yml extension, got %d", len(manifests))
	}
	if manifests[0].Metadata.Name != "yml-app" {
		t.Errorf("Expected name 'yml-app', got '%s'", manifests[0].Metadata.Name)
	}
}

func TestCollectManifests_NonexistentDir(t *testing.T) {
	_, err := CollectManifests("/nonexistent/directory/path")
	if err == nil {
		t.Error("Expected error for non-existent directory")
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

func TestParseManifestData_MalformedYAML(t *testing.T) {
	data := []byte(`{{{invalid yaml content`)

	_, err := ParseManifestData(data, "bad.yaml")
	if err == nil {
		t.Error("Expected error for malformed YAML data")
	}
}

func TestParseManifestData_MalformedJSON(t *testing.T) {
	data := []byte(`{invalid json`)

	_, err := ParseManifestData(data, "bad.json")
	if err == nil {
		t.Error("Expected error for malformed JSON data")
	}
}

func TestZenithManifest_Fields(t *testing.T) {
	manifest := ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: ManifestMetadata{
			Name:      "test-app",
			Namespace: "production",
			Labels: map[string]string{
				"env": "prod",
			},
		},
		Spec: map[string]interface{}{
			"image":    "nginx:latest",
			"replicas": 3,
		},
	}

	if manifest.APIVersion != "zenith.dev/v1alpha1" {
		t.Errorf("Expected APIVersion 'zenith.dev/v1alpha1', got '%s'", manifest.APIVersion)
	}
	if manifest.Kind != "App" {
		t.Errorf("Expected Kind 'App', got '%s'", manifest.Kind)
	}
	if manifest.Metadata.Name != "test-app" {
		t.Errorf("Expected Name 'test-app', got '%s'", manifest.Metadata.Name)
	}
	if manifest.Metadata.Namespace != "production" {
		t.Errorf("Expected Namespace 'production', got '%s'", manifest.Metadata.Namespace)
	}
	if manifest.Metadata.Labels["env"] != "prod" {
		t.Errorf("Expected label 'env'='prod', got '%s'", manifest.Metadata.Labels["env"])
	}
}

func TestManifestMetadata_OmitEmpty(t *testing.T) {
	// Namespace and Labels should be omitted when empty
	manifest := ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: ManifestMetadata{
			Name: "minimal-app",
		},
		Spec: map[string]interface{}{},
	}

	yamlData, err := MarshalManifest(manifest, "yaml")
	if err != nil {
		t.Fatalf("MarshalManifest failed: %v", err)
	}

	yamlStr := string(yamlData)
	if strings.Contains(yamlStr, "namespace") {
		t.Error("Expected namespace to be omitted when empty")
	}
	if strings.Contains(yamlStr, "labels") {
		t.Error("Expected labels to be omitted when empty/nil")
	}

	jsonData, err := MarshalManifest(manifest, "json")
	if err != nil {
		t.Fatalf("MarshalManifest JSON failed: %v", err)
	}

	jsonStr := string(jsonData)
	if strings.Contains(jsonStr, "namespace") {
		t.Error("Expected namespace to be omitted from JSON when empty")
	}
}

func TestMarshalManifest_NestedArraysInSpec(t *testing.T) {
	manifest := ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata:   ManifestMetadata{Name: "array-app"},
		Spec: map[string]interface{}{
			"domains": []interface{}{
				"app.example.com",
				"www.example.com",
			},
			"envVars": []interface{}{
				map[string]interface{}{"name": "DB_HOST", "value": "localhost"},
				map[string]interface{}{"name": "DB_PORT", "value": "5432"},
			},
		},
	}

	data, err := MarshalManifest(manifest, "yaml")
	if err != nil {
		t.Fatalf("MarshalManifest failed: %v", err)
	}

	var parsed ZenithManifest
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	domains, ok := parsed.Spec["domains"].([]interface{})
	if !ok {
		t.Fatal("Expected 'domains' to be an array")
	}
	if len(domains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(domains))
	}

	envVars, ok := parsed.Spec["envVars"].([]interface{})
	if !ok {
		t.Fatal("Expected 'envVars' to be an array")
	}
	if len(envVars) != 2 {
		t.Errorf("Expected 2 envVars, got %d", len(envVars))
	}
}
