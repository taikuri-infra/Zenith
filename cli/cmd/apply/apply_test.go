package apply

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/cli/cmd/export"
	"github.com/dotechhq/zenith/cli/internal/api"
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

func TestApplyManifestData_InvalidYAML(t *testing.T) {
	data := []byte(`
kind: App
metadata:
  name: test
spec:
  - this is invalid: [yaml: {broken
    @@@ not valid
`)

	_, err := ApplyManifestData(data, "yaml")
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parse YAML") {
		t.Errorf("Expected error to mention 'parse YAML', got: %v", err)
	}
}

func TestApplyManifestData_EmptyData(t *testing.T) {
	data := []byte("")

	manifest, err := ApplyManifestData(data, "yaml")
	if err != nil {
		t.Fatalf("ApplyManifestData with empty data should not error, got: %v", err)
	}
	// Empty YAML unmarshals to zero-value struct
	if manifest.Kind != "" {
		t.Errorf("Expected empty Kind for empty data, got '%s'", manifest.Kind)
	}
	if manifest.Metadata.Name != "" {
		t.Errorf("Expected empty Name for empty data, got '%s'", manifest.Metadata.Name)
	}
}

func TestApplyManifestData_EmptyJSON(t *testing.T) {
	data := []byte("")

	_, err := ApplyManifestData(data, "json")
	if err == nil {
		t.Error("Expected error for empty JSON data")
	}
}

func TestApplyManifestData_DatabaseKind(t *testing.T) {
	data := []byte(`apiVersion: zenith.dev/v1alpha1
kind: Database
metadata:
  name: my-postgres
spec:
  engine: postgresql
  version: "16"
  storage: 20Gi
`)

	manifest, err := ApplyManifestData(data, "yaml")
	if err != nil {
		t.Fatalf("ApplyManifestData failed: %v", err)
	}

	if manifest.Kind != "Database" {
		t.Errorf("Expected kind 'Database', got '%s'", manifest.Kind)
	}
	if manifest.Metadata.Name != "my-postgres" {
		t.Errorf("Expected name 'my-postgres', got '%s'", manifest.Metadata.Name)
	}
	if manifest.Spec["engine"] != "postgresql" {
		t.Errorf("Expected engine 'postgresql', got '%v'", manifest.Spec["engine"])
	}
	if manifest.Spec["version"] != "16" {
		t.Errorf("Expected version '16', got '%v'", manifest.Spec["version"])
	}
	if manifest.Spec["storage"] != "20Gi" {
		t.Errorf("Expected storage '20Gi', got '%v'", manifest.Spec["storage"])
	}
}

func TestApplyManifestData_StorageBucketKind(t *testing.T) {
	data := []byte(`apiVersion: zenith.dev/v1alpha1
kind: StorageBucket
metadata:
  name: my-bucket
spec:
  size: 50Gi
  accessPolicy: private
`)

	manifest, err := ApplyManifestData(data, "yaml")
	if err != nil {
		t.Fatalf("ApplyManifestData failed: %v", err)
	}

	if manifest.Kind != "StorageBucket" {
		t.Errorf("Expected kind 'StorageBucket', got '%s'", manifest.Kind)
	}
	if manifest.Spec["size"] != "50Gi" {
		t.Errorf("Expected size '50Gi', got '%v'", manifest.Spec["size"])
	}
	if manifest.Spec["accessPolicy"] != "private" {
		t.Errorf("Expected accessPolicy 'private', got '%v'", manifest.Spec["accessPolicy"])
	}
}

func TestApplyManifestData_FormatCaseInsensitive(t *testing.T) {
	tests := []struct {
		name   string
		format string
		data   []byte
	}{
		{
			name:   "uppercase JSON",
			format: "JSON",
			data:   []byte(`{"apiVersion":"zenith.dev/v1alpha1","kind":"App","metadata":{"name":"test"},"spec":{}}`),
		},
		{
			name:   "mixed case Json",
			format: "Json",
			data:   []byte(`{"apiVersion":"zenith.dev/v1alpha1","kind":"App","metadata":{"name":"test"},"spec":{}}`),
		},
		{
			name:   "lowercase yaml",
			format: "yaml",
			data:   []byte("apiVersion: zenith.dev/v1alpha1\nkind: App\nmetadata:\n  name: test\nspec: {}\n"),
		},
		{
			name:   "uppercase YAML treated as yaml (default)",
			format: "YAML",
			data:   []byte("apiVersion: zenith.dev/v1alpha1\nkind: App\nmetadata:\n  name: test\nspec: {}\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := ApplyManifestData(tt.data, tt.format)
			if err != nil {
				t.Fatalf("ApplyManifestData with format %q failed: %v", tt.format, err)
			}
			if manifest.Kind != "App" {
				t.Errorf("Expected kind 'App', got '%s'", manifest.Kind)
			}
		})
	}
}

func TestApplyManifestData_SpecFieldExtraction(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		specKey  string
		expected interface{}
	}{
		{
			name: "image field",
			yamlData: `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: test
spec:
  image: nginx:1.25
`,
			specKey:  "image",
			expected: "nginx:1.25",
		},
		{
			name: "replicas field",
			yamlData: `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: test
spec:
  replicas: 3
`,
			specKey:  "replicas",
			expected: 3,
		},
		{
			name: "port field",
			yamlData: `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: test
spec:
  port: 9090
`,
			specKey:  "port",
			expected: 9090,
		},
		{
			name: "engine field",
			yamlData: `apiVersion: zenith.dev/v1alpha1
kind: Database
metadata:
  name: test-db
spec:
  engine: mysql
`,
			specKey:  "engine",
			expected: "mysql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := ApplyManifestData([]byte(tt.yamlData), "yaml")
			if err != nil {
				t.Fatalf("ApplyManifestData failed: %v", err)
			}
			val, ok := manifest.Spec[tt.specKey]
			if !ok {
				t.Fatalf("Expected spec key '%s' to exist", tt.specKey)
			}
			if fmt.Sprintf("%v", val) != fmt.Sprintf("%v", tt.expected) {
				t.Errorf("Expected spec[%s]=%v, got %v", tt.specKey, tt.expected, val)
			}
		})
	}
}

func TestApplyManifest_AppKind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/apps") {
			t.Errorf("Expected path to contain '/apps', got '%s'", r.URL.Path)
		}

		var body api.App
		json.NewDecoder(r.Body).Decode(&body)
		if body.Name != "web-app" {
			t.Errorf("Expected app name 'web-app', got '%s'", body.Name)
		}
		if body.Image != "nginx:latest" {
			t.Errorf("Expected image 'nginx:latest', got '%s'", body.Image)
		}
		if body.Replicas != 2 {
			t.Errorf("Expected replicas 2, got %d", body.Replicas)
		}
		if body.Port != 8080 {
			t.Errorf("Expected port 8080, got %d", body.Port)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.App{Name: "web-app", Status: "Running"})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	manifest := &export.ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata:   export.ManifestMetadata{Name: "web-app"},
		Spec: map[string]interface{}{
			"image":    "nginx:latest",
			"replicas": 2,
			"port":     8080,
		},
	}

	result := applyManifest(client, "default", manifest, "web-app.yaml")
	if result.Action != "created" {
		t.Errorf("Expected action 'created', got '%s'", result.Action)
	}
	if result.Name != "web-app" {
		t.Errorf("Expected name 'web-app', got '%s'", result.Name)
	}
	if result.Kind != "App" {
		t.Errorf("Expected kind 'App', got '%s'", result.Kind)
	}
	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}
}

func TestApplyManifest_DatabaseKind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/databases") {
			t.Errorf("Expected path to contain '/databases', got '%s'", r.URL.Path)
		}

		var body api.Database
		json.NewDecoder(r.Body).Decode(&body)
		if body.Name != "my-db" {
			t.Errorf("Expected db name 'my-db', got '%s'", body.Name)
		}
		if body.Engine != "postgresql" {
			t.Errorf("Expected engine 'postgresql', got '%s'", body.Engine)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.Database{Name: "my-db", Status: "Ready"})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	manifest := &export.ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Database",
		Metadata:   export.ManifestMetadata{Name: "my-db"},
		Spec: map[string]interface{}{
			"engine":  "postgresql",
			"version": "16",
			"storage": "20Gi",
		},
	}

	result := applyManifest(client, "default", manifest, "my-db.yaml")
	if result.Action != "created" {
		t.Errorf("Expected action 'created', got '%s'", result.Action)
	}
	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}
}

func TestApplyManifest_UnsupportedKind(t *testing.T) {
	client := api.NewClient("http://localhost:9999", "test-token")
	manifest := &export.ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "StorageBucket",
		Metadata:   export.ManifestMetadata{Name: "my-bucket"},
		Spec:       map[string]interface{}{"size": "50Gi"},
	}

	result := applyManifest(client, "default", manifest, "my-bucket.yaml")
	if result.Action != "failed" {
		t.Errorf("Expected action 'failed' for unsupported kind, got '%s'", result.Action)
	}
	if result.Error == nil {
		t.Error("Expected error for unsupported kind")
	}
	if !strings.Contains(result.Error.Error(), "unsupported resource kind") {
		t.Errorf("Expected error to mention 'unsupported resource kind', got: %v", result.Error)
	}
	if !strings.Contains(result.Error.Error(), "StorageBucket") {
		t.Errorf("Expected error to mention 'StorageBucket', got: %v", result.Error)
	}
}

func TestApplyManifest_AppAPIFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	manifest := &export.ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata:   export.ManifestMetadata{Name: "failing-app"},
		Spec: map[string]interface{}{
			"image": "nginx:latest",
		},
	}

	result := applyManifest(client, "default", manifest, "failing-app.yaml")
	if result.Action != "failed" {
		t.Errorf("Expected action 'failed', got '%s'", result.Action)
	}
	if result.Error == nil {
		t.Error("Expected error for API failure")
	}
}

func TestApplyManifest_DatabaseAPIFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	manifest := &export.ZenithManifest{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Database",
		Metadata:   export.ManifestMetadata{Name: "failing-db"},
		Spec: map[string]interface{}{
			"engine": "postgresql",
		},
	}

	result := applyManifest(client, "default", manifest, "failing-db.yaml")
	if result.Action != "failed" {
		t.Errorf("Expected action 'failed', got '%s'", result.Action)
	}
	if result.Error == nil {
		t.Error("Expected error for API failure")
	}
}

func TestApplyManifest_AppWithFloat64Replicas(t *testing.T) {
	// When parsing from JSON, numbers come as float64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body api.App
		json.NewDecoder(r.Body).Decode(&body)
		if body.Replicas != 3 {
			t.Errorf("Expected replicas 3, got %d", body.Replicas)
		}
		if body.Port != 9090 {
			t.Errorf("Expected port 9090, got %d", body.Port)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.App{Name: "float-app"})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	manifest := &export.ZenithManifest{
		Kind:     "App",
		Metadata: export.ManifestMetadata{Name: "float-app"},
		Spec: map[string]interface{}{
			"image":    "nginx:latest",
			"replicas": float64(3),
			"port":     float64(9090),
		},
	}

	result := applyManifest(client, "default", manifest, "float-app.yaml")
	if result.Action != "created" {
		t.Errorf("Expected action 'created', got '%s'", result.Action)
	}
}

func TestApplyResult_StructFields(t *testing.T) {
	result := ApplyResult{
		Name:     "my-app",
		Kind:     "App",
		Action:   "created",
		Error:    nil,
		FilePath: "/path/to/app.yaml",
	}

	if result.Name != "my-app" {
		t.Errorf("Expected Name 'my-app', got '%s'", result.Name)
	}
	if result.Kind != "App" {
		t.Errorf("Expected Kind 'App', got '%s'", result.Kind)
	}
	if result.Action != "created" {
		t.Errorf("Expected Action 'created', got '%s'", result.Action)
	}
	if result.Error != nil {
		t.Errorf("Expected nil Error, got %v", result.Error)
	}
	if result.FilePath != "/path/to/app.yaml" {
		t.Errorf("Expected FilePath '/path/to/app.yaml', got '%s'", result.FilePath)
	}
}

func TestApplyResult_WithError(t *testing.T) {
	testErr := fmt.Errorf("connection refused")
	result := ApplyResult{
		Name:     "broken-app",
		Kind:     "App",
		Action:   "failed",
		Error:    testErr,
		FilePath: "broken-app.yaml",
	}

	if result.Error == nil {
		t.Error("Expected non-nil Error")
	}
	if result.Error.Error() != "connection refused" {
		t.Errorf("Expected error 'connection refused', got '%s'", result.Error.Error())
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

func TestApplyDirectoryWalkWithMixedFileTypes(t *testing.T) {
	dir := t.TempDir()

	// Create various file types
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("notes"), 0644)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log"), 0644)
	os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all: build"), 0644)

	// Create valid YAML manifest
	yamlContent := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: yaml-app
spec:
  image: nginx:latest
`
	os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(yamlContent), 0644)

	// Create valid YML manifest
	ymlContent := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: yml-app
spec:
  image: redis:latest
`
	os.WriteFile(filepath.Join(dir, "app2.yml"), []byte(ymlContent), 0644)

	// Create valid JSON manifest
	jsonContent := `{"apiVersion":"zenith.dev/v1alpha1","kind":"Database","metadata":{"name":"json-db"},"spec":{"engine":"redis"}}`
	os.WriteFile(filepath.Join(dir, "db.json"), []byte(jsonContent), 0644)

	manifests, err := export.CollectManifests(dir)
	if err != nil {
		t.Fatalf("CollectManifests failed: %v", err)
	}

	if len(manifests) != 3 {
		t.Fatalf("Expected 3 manifests (.yaml, .yml, .json), got %d", len(manifests))
	}
}

func TestApplyManifest_AppWithMissingOptionalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body api.App
		json.NewDecoder(r.Body).Decode(&body)
		// Only name and image set, replicas and port default to zero
		if body.Name != "minimal-app" {
			t.Errorf("Expected name 'minimal-app', got '%s'", body.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.App{Name: "minimal-app"})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	manifest := &export.ZenithManifest{
		Kind:     "App",
		Metadata: export.ManifestMetadata{Name: "minimal-app"},
		Spec: map[string]interface{}{
			"image": "alpine:latest",
		},
	}

	result := applyManifest(client, "default", manifest, "minimal-app.yaml")
	if result.Action != "created" {
		t.Errorf("Expected action 'created', got '%s'", result.Action)
	}
}

func TestDecodeYAML(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "valid YAML",
			data:    "key: value\n",
			wantErr: false,
		},
		{
			name:    "empty YAML",
			data:    "",
			wantErr: false,
		},
		{
			name:    "invalid YAML",
			data:    "{{not: valid: yaml: [}",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			err := decodeYAML([]byte(tt.data), &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
