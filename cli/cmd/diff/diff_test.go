package diff

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dotechhq/zenith/cli/cmd/export"
	"github.com/dotechhq/zenith/cli/internal/api"
)

func TestCompareSpecs_Identical(t *testing.T) {
	local := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
		"port":     8080,
	}
	remote := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
		"port":     8080,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 0 {
		t.Errorf("Expected no changes for identical specs, got %d", len(changes))
	}
}

func TestCompareSpecs_Modified(t *testing.T) {
	local := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 3,
		"port":     8080,
	}
	remote := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
		"port":     8080,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}

	if changes[0].Field != "replicas" {
		t.Errorf("Expected changed field 'replicas', got '%s'", changes[0].Field)
	}
	if changes[0].OldValue != "2" {
		t.Errorf("Expected old value '2', got '%s'", changes[0].OldValue)
	}
	if changes[0].NewValue != "3" {
		t.Errorf("Expected new value '3', got '%s'", changes[0].NewValue)
	}
}

func TestCompareSpecs_FieldAdded(t *testing.T) {
	local := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
		"domain":   "app.example.com",
	}
	remote := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change (new field), got %d", len(changes))
	}

	if changes[0].Field != "domain" {
		t.Errorf("Expected field 'domain', got '%s'", changes[0].Field)
	}
	if changes[0].OldValue != "" {
		t.Errorf("Expected empty old value for new field, got '%s'", changes[0].OldValue)
	}
	if changes[0].NewValue != "app.example.com" {
		t.Errorf("Expected new value 'app.example.com', got '%s'", changes[0].NewValue)
	}
}

func TestCompareSpecs_FieldRemoved(t *testing.T) {
	local := map[string]interface{}{
		"image": "nginx:latest",
	}
	remote := map[string]interface{}{
		"image":    "nginx:latest",
		"replicas": 2,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change (removed field), got %d", len(changes))
	}

	if changes[0].Field != "replicas" {
		t.Errorf("Expected field 'replicas', got '%s'", changes[0].Field)
	}
	if changes[0].OldValue != "2" {
		t.Errorf("Expected old value '2', got '%s'", changes[0].OldValue)
	}
	if changes[0].NewValue != "" {
		t.Errorf("Expected empty new value for removed field, got '%s'", changes[0].NewValue)
	}
}

func TestCompareSpecs_MultipleChanges(t *testing.T) {
	local := map[string]interface{}{
		"image":    "nginx:1.25",
		"replicas": 5,
		"port":     9090,
		"domain":   "new.example.com",
	}
	remote := map[string]interface{}{
		"image":    "nginx:1.24",
		"replicas": 2,
		"port":     8080,
		"env":      "production",
	}

	changes := CompareSpecs(local, remote)

	// 3 modified (image, replicas, port), 1 added (domain), 1 removed (env) = 5
	if len(changes) != 5 {
		t.Fatalf("Expected 5 changes, got %d", len(changes))
	}

	changeMap := make(map[string]FieldChange)
	for _, c := range changes {
		changeMap[c.Field] = c
	}

	// Verify image change
	if c, ok := changeMap["image"]; ok {
		if c.OldValue != "nginx:1.24" || c.NewValue != "nginx:1.25" {
			t.Errorf("Image change: expected '1.24' -> '1.25', got '%s' -> '%s'", c.OldValue, c.NewValue)
		}
	} else {
		t.Error("Expected image change")
	}

	// Verify domain addition
	if c, ok := changeMap["domain"]; ok {
		if c.OldValue != "" || c.NewValue != "new.example.com" {
			t.Errorf("Domain addition: expected '' -> 'new.example.com', got '%s' -> '%s'", c.OldValue, c.NewValue)
		}
	} else {
		t.Error("Expected domain change")
	}

	// Verify env removal
	if c, ok := changeMap["env"]; ok {
		if c.OldValue != "production" || c.NewValue != "" {
			t.Errorf("Env removal: expected 'production' -> '', got '%s' -> '%s'", c.OldValue, c.NewValue)
		}
	} else {
		t.Error("Expected env change")
	}
}

func TestCompareSpecs_EmptySpecs(t *testing.T) {
	local := map[string]interface{}{}
	remote := map[string]interface{}{}

	changes := CompareSpecs(local, remote)
	if len(changes) != 0 {
		t.Errorf("Expected no changes for empty specs, got %d", len(changes))
	}
}

func TestCompareSpecs_AllNew(t *testing.T) {
	local := map[string]interface{}{
		"image": "nginx:latest",
		"port":  8080,
	}
	remote := map[string]interface{}{}

	changes := CompareSpecs(local, remote)
	if len(changes) != 2 {
		t.Fatalf("Expected 2 changes (all new), got %d", len(changes))
	}

	for _, c := range changes {
		if c.OldValue != "" {
			t.Errorf("Expected empty old value for new field %s, got '%s'", c.Field, c.OldValue)
		}
	}
}

func TestCompareSpecs_AllRemoved(t *testing.T) {
	local := map[string]interface{}{}
	remote := map[string]interface{}{
		"image": "nginx:latest",
		"port":  8080,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 2 {
		t.Fatalf("Expected 2 changes (all removed), got %d", len(changes))
	}

	for _, c := range changes {
		if c.NewValue != "" {
			t.Errorf("Expected empty new value for removed field %s, got '%s'", c.Field, c.NewValue)
		}
	}
}

func TestCompareSpecs_NestedObjects(t *testing.T) {
	local := map[string]interface{}{
		"resources": map[string]interface{}{
			"cpu":    "500m",
			"memory": "256Mi",
		},
	}
	remote := map[string]interface{}{
		"resources": map[string]interface{}{
			"cpu":    "250m",
			"memory": "128Mi",
		},
	}

	changes := CompareSpecs(local, remote)
	// Nested objects are compared by their string representation via fmt.Sprintf("%v", ...)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change (resources differs), got %d", len(changes))
	}

	if changes[0].Field != "resources" {
		t.Errorf("Expected field 'resources', got '%s'", changes[0].Field)
	}
	// Both old and new values should be non-empty since the maps differ
	if changes[0].OldValue == "" {
		t.Error("Expected non-empty old value for nested object change")
	}
	if changes[0].NewValue == "" {
		t.Error("Expected non-empty new value for nested object change")
	}
}

func TestCompareSpecs_NestedObjectsIdentical(t *testing.T) {
	nested := map[string]interface{}{
		"cpu":    "500m",
		"memory": "256Mi",
	}
	local := map[string]interface{}{
		"resources": nested,
	}
	remote := map[string]interface{}{
		"resources": nested,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 0 {
		t.Errorf("Expected no changes for identical nested specs, got %d", len(changes))
	}
}

func TestCompareSpecs_ArrayValues(t *testing.T) {
	local := map[string]interface{}{
		"ports": []interface{}{8080, 8443},
	}
	remote := map[string]interface{}{
		"ports": []interface{}{8080, 9090},
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change (array differs), got %d", len(changes))
	}
	if changes[0].Field != "ports" {
		t.Errorf("Expected field 'ports', got '%s'", changes[0].Field)
	}
}

func TestCompareSpecs_ArrayIdentical(t *testing.T) {
	local := map[string]interface{}{
		"ports": []interface{}{8080, 8443},
	}
	remote := map[string]interface{}{
		"ports": []interface{}{8080, 8443},
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 0 {
		t.Errorf("Expected no changes for identical arrays, got %d", len(changes))
	}
}

func TestCompareSpecs_NumericTypeDifferences(t *testing.T) {
	// When parsed from JSON, numbers are float64; from Go code, they could be int
	// The comparison uses fmt.Sprintf("%v", val), so int(2) == "2" and float64(2) == "2"
	local := map[string]interface{}{
		"replicas": 2,     // int
		"port":     8080,  // int
	}
	remote := map[string]interface{}{
		"replicas": float64(2),    // float64 (from JSON)
		"port":     float64(8080), // float64 (from JSON)
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 0 {
		t.Errorf("Expected no changes for int vs float64 with same value, got %d", len(changes))
		for _, c := range changes {
			t.Logf("  Field: %s, Old: %s, New: %s", c.Field, c.OldValue, c.NewValue)
		}
	}
}

func TestCompareSpecs_NumericTypeDifferencesWithDecimal(t *testing.T) {
	local := map[string]interface{}{
		"ratio": 0.5,
	}
	remote := map[string]interface{}{
		"ratio": 0.75,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change for different float values, got %d", len(changes))
	}
	if changes[0].Field != "ratio" {
		t.Errorf("Expected field 'ratio', got '%s'", changes[0].Field)
	}
}

func TestCompareSpecs_BooleanValues(t *testing.T) {
	local := map[string]interface{}{
		"tls":    true,
		"debug":  false,
	}
	remote := map[string]interface{}{
		"tls":    false,
		"debug":  false,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change (tls changed), got %d", len(changes))
	}
	if changes[0].Field != "tls" {
		t.Errorf("Expected field 'tls', got '%s'", changes[0].Field)
	}
	if changes[0].OldValue != "false" {
		t.Errorf("Expected old value 'false', got '%s'", changes[0].OldValue)
	}
	if changes[0].NewValue != "true" {
		t.Errorf("Expected new value 'true', got '%s'", changes[0].NewValue)
	}
}

func TestCompareSpecs_NilValues(t *testing.T) {
	local := map[string]interface{}{
		"image":  "nginx:latest",
		"config": nil,
	}
	remote := map[string]interface{}{
		"image":  "nginx:latest",
		"config": nil,
	}

	changes := CompareSpecs(local, remote)
	if len(changes) != 0 {
		t.Errorf("Expected no changes for identical nil values, got %d", len(changes))
	}
}

func TestDiffEntry_Types(t *testing.T) {
	entries := []DiffEntry{
		{Kind: "App", Name: "app1", Action: "added"},
		{Kind: "App", Name: "app2", Action: "modified"},
		{Kind: "Database", Name: "db1", Action: "deleted"},
		{Kind: "App", Name: "app3", Action: "unchanged"},
	}

	actions := make(map[string]int)
	for _, e := range entries {
		actions[e.Action]++
	}

	if actions["added"] != 1 {
		t.Errorf("Expected 1 added, got %d", actions["added"])
	}
	if actions["modified"] != 1 {
		t.Errorf("Expected 1 modified, got %d", actions["modified"])
	}
	if actions["deleted"] != 1 {
		t.Errorf("Expected 1 deleted, got %d", actions["deleted"])
	}
	if actions["unchanged"] != 1 {
		t.Errorf("Expected 1 unchanged, got %d", actions["unchanged"])
	}
}

func TestDiffEntry_WithChanges(t *testing.T) {
	entry := DiffEntry{
		Kind:   "App",
		Name:   "web-app",
		Action: "modified",
		LocalSpec: map[string]interface{}{
			"image":    "nginx:1.25",
			"replicas": 3,
		},
		RemoteSpec: map[string]interface{}{
			"image":    "nginx:1.24",
			"replicas": 2,
		},
		Changes: []FieldChange{
			{Field: "image", OldValue: "nginx:1.24", NewValue: "nginx:1.25"},
			{Field: "replicas", OldValue: "2", NewValue: "3"},
		},
	}

	if len(entry.Changes) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(entry.Changes))
	}
	if entry.LocalSpec["image"] != "nginx:1.25" {
		t.Errorf("Expected local image 'nginx:1.25', got '%v'", entry.LocalSpec["image"])
	}
	if entry.RemoteSpec["image"] != "nginx:1.24" {
		t.Errorf("Expected remote image 'nginx:1.24', got '%v'", entry.RemoteSpec["image"])
	}
}

func TestDiffEntry_Added(t *testing.T) {
	entry := DiffEntry{
		Kind:   "App",
		Name:   "new-app",
		Action: "added",
		LocalSpec: map[string]interface{}{
			"image": "nginx:latest",
			"port":  8080,
		},
		RemoteSpec: nil,
	}

	if entry.Action != "added" {
		t.Errorf("Expected action 'added', got '%s'", entry.Action)
	}
	if entry.RemoteSpec != nil {
		t.Error("Expected nil RemoteSpec for added entry")
	}
	if entry.LocalSpec == nil {
		t.Error("Expected non-nil LocalSpec for added entry")
	}
}

func TestDiffEntry_Deleted(t *testing.T) {
	entry := DiffEntry{
		Kind:      "Database",
		Name:      "old-db",
		Action:    "deleted",
		LocalSpec: nil,
		RemoteSpec: map[string]interface{}{
			"engine": "postgresql",
		},
	}

	if entry.Action != "deleted" {
		t.Errorf("Expected action 'deleted', got '%s'", entry.Action)
	}
	if entry.LocalSpec != nil {
		t.Error("Expected nil LocalSpec for deleted entry")
	}
	if entry.RemoteSpec == nil {
		t.Error("Expected non-nil RemoteSpec for deleted entry")
	}
}

func TestFieldChange_Representation(t *testing.T) {
	tests := []struct {
		name   string
		change FieldChange
		isAdd  bool
		isMod  bool
		isDel  bool
	}{
		{
			name:   "addition",
			change: FieldChange{Field: "domain", OldValue: "", NewValue: "app.example.com"},
			isAdd:  true,
		},
		{
			name:   "modification",
			change: FieldChange{Field: "replicas", OldValue: "2", NewValue: "3"},
			isMod:  true,
		},
		{
			name:   "deletion",
			change: FieldChange{Field: "env", OldValue: "production", NewValue: ""},
			isDel:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAdd := tt.change.OldValue == "" && tt.change.NewValue != ""
			isMod := tt.change.OldValue != "" && tt.change.NewValue != ""
			isDel := tt.change.OldValue != "" && tt.change.NewValue == ""

			if isAdd != tt.isAdd {
				t.Errorf("Expected isAdd=%v, got %v", tt.isAdd, isAdd)
			}
			if isMod != tt.isMod {
				t.Errorf("Expected isMod=%v, got %v", tt.isMod, isMod)
			}
			if isDel != tt.isDel {
				t.Errorf("Expected isDel=%v, got %v", tt.isDel, isDel)
			}
		})
	}
}

func TestFieldChange_StringFormatting(t *testing.T) {
	tests := []struct {
		name        string
		change      FieldChange
		expectedStr string
	}{
		{
			name:        "addition format",
			change:      FieldChange{Field: "domain", OldValue: "", NewValue: "app.example.com"},
			expectedStr: "domain: app.example.com",
		},
		{
			name:        "modification format",
			change:      FieldChange{Field: "replicas", OldValue: "2", NewValue: "3"},
			expectedStr: "replicas: 2 -> 3",
		},
		{
			name:        "deletion format",
			change:      FieldChange{Field: "env", OldValue: "production", NewValue: ""},
			expectedStr: "env: production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var str string
			if tt.change.OldValue == "" {
				str = fmt.Sprintf("%s: %s", tt.change.Field, tt.change.NewValue)
			} else if tt.change.NewValue == "" {
				str = fmt.Sprintf("%s: %s", tt.change.Field, tt.change.OldValue)
			} else {
				str = fmt.Sprintf("%s: %s -> %s", tt.change.Field, tt.change.OldValue, tt.change.NewValue)
			}

			if str != tt.expectedStr {
				t.Errorf("Expected '%s', got '%s'", tt.expectedStr, str)
			}
		})
	}
}

func TestBuildDiffEntry_AppAdded(t *testing.T) {
	manifest := &export.ZenithManifest{
		Kind:     "App",
		Metadata: export.ManifestMetadata{Name: "new-app"},
		Spec: map[string]interface{}{
			"image": "nginx:latest",
			"port":  8080,
		},
	}

	remoteApps := map[string]map[string]interface{}{}
	remoteDbs := map[string]map[string]interface{}{}

	entry := buildDiffEntry(manifest, remoteApps, remoteDbs)
	if entry.Action != "added" {
		t.Errorf("Expected action 'added', got '%s'", entry.Action)
	}
	if entry.Kind != "App" {
		t.Errorf("Expected kind 'App', got '%s'", entry.Kind)
	}
	if entry.Name != "new-app" {
		t.Errorf("Expected name 'new-app', got '%s'", entry.Name)
	}
}

func TestBuildDiffEntry_AppUnchanged(t *testing.T) {
	manifest := &export.ZenithManifest{
		Kind:     "App",
		Metadata: export.ManifestMetadata{Name: "existing-app"},
		Spec: map[string]interface{}{
			"image":    "nginx:latest",
			"replicas": 2,
			"port":     8080,
		},
	}

	remoteApps := map[string]map[string]interface{}{
		"existing-app": {
			"image":    "nginx:latest",
			"replicas": 2,
			"port":     8080,
		},
	}
	remoteDbs := map[string]map[string]interface{}{}

	entry := buildDiffEntry(manifest, remoteApps, remoteDbs)
	if entry.Action != "unchanged" {
		t.Errorf("Expected action 'unchanged', got '%s'", entry.Action)
	}
}

func TestBuildDiffEntry_AppModified(t *testing.T) {
	manifest := &export.ZenithManifest{
		Kind:     "App",
		Metadata: export.ManifestMetadata{Name: "changing-app"},
		Spec: map[string]interface{}{
			"image":    "nginx:1.25",
			"replicas": 3,
		},
	}

	remoteApps := map[string]map[string]interface{}{
		"changing-app": {
			"image":    "nginx:1.24",
			"replicas": 2,
		},
	}
	remoteDbs := map[string]map[string]interface{}{}

	entry := buildDiffEntry(manifest, remoteApps, remoteDbs)
	if entry.Action != "modified" {
		t.Errorf("Expected action 'modified', got '%s'", entry.Action)
	}
	if len(entry.Changes) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(entry.Changes))
	}
}

func TestBuildDiffEntry_DatabaseAdded(t *testing.T) {
	manifest := &export.ZenithManifest{
		Kind:     "Database",
		Metadata: export.ManifestMetadata{Name: "new-db"},
		Spec: map[string]interface{}{
			"engine":  "postgresql",
			"version": "16",
		},
	}

	remoteApps := map[string]map[string]interface{}{}
	remoteDbs := map[string]map[string]interface{}{}

	entry := buildDiffEntry(manifest, remoteApps, remoteDbs)
	if entry.Action != "added" {
		t.Errorf("Expected action 'added', got '%s'", entry.Action)
	}
}

func TestBuildDiffEntry_DatabaseModified(t *testing.T) {
	manifest := &export.ZenithManifest{
		Kind:     "Database",
		Metadata: export.ManifestMetadata{Name: "my-db"},
		Spec: map[string]interface{}{
			"engine":  "postgresql",
			"version": "16",
			"storage": "40Gi",
		},
	}

	remoteApps := map[string]map[string]interface{}{}
	remoteDbs := map[string]map[string]interface{}{
		"my-db": {
			"engine":  "postgresql",
			"version": "16",
			"storage": "20Gi",
		},
	}

	entry := buildDiffEntry(manifest, remoteApps, remoteDbs)
	if entry.Action != "modified" {
		t.Errorf("Expected action 'modified', got '%s'", entry.Action)
	}
	if len(entry.Changes) != 1 {
		t.Fatalf("Expected 1 change (storage), got %d", len(entry.Changes))
	}
	if entry.Changes[0].Field != "storage" {
		t.Errorf("Expected changed field 'storage', got '%s'", entry.Changes[0].Field)
	}
}

func TestBuildDiffEntry_UnsupportedKind(t *testing.T) {
	manifest := &export.ZenithManifest{
		Kind:     "StorageBucket",
		Metadata: export.ManifestMetadata{Name: "my-bucket"},
		Spec:     map[string]interface{}{"size": "50Gi"},
	}

	remoteApps := map[string]map[string]interface{}{}
	remoteDbs := map[string]map[string]interface{}{}

	entry := buildDiffEntry(manifest, remoteApps, remoteDbs)
	// Unsupported kinds default to "added"
	if entry.Action != "added" {
		t.Errorf("Expected action 'added' for unsupported kind, got '%s'", entry.Action)
	}
}

func TestFetchRemoteApps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []api.App{
				{Name: "app1", Image: "nginx:latest", Replicas: 2, Port: 8080},
				{Name: "app2", Image: "redis:latest", Replicas: 1, Port: 6379},
			},
		})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	result := fetchRemoteApps(client, "default")

	if len(result) != 2 {
		t.Fatalf("Expected 2 remote apps, got %d", len(result))
	}
	if result["app1"]["image"] != "nginx:latest" {
		t.Errorf("Expected app1 image 'nginx:latest', got '%v'", result["app1"]["image"])
	}
	if result["app2"]["image"] != "redis:latest" {
		t.Errorf("Expected app2 image 'redis:latest', got '%v'", result["app2"]["image"])
	}
}

func TestFetchRemoteApps_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	result := fetchRemoteApps(client, "default")

	// On error, returns empty map
	if len(result) != 0 {
		t.Errorf("Expected 0 remote apps on error, got %d", len(result))
	}
}

func TestFetchRemoteDatabases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []api.Database{
				{Name: "db1", Engine: "postgresql", Version: "16", Storage: "20Gi"},
			},
		})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	result := fetchRemoteDatabases(client, "default")

	if len(result) != 1 {
		t.Fatalf("Expected 1 remote database, got %d", len(result))
	}
	if result["db1"]["engine"] != "postgresql" {
		t.Errorf("Expected db1 engine 'postgresql', got '%v'", result["db1"]["engine"])
	}
}

func TestFetchRemoteDatabases_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	result := fetchRemoteDatabases(client, "default")

	if len(result) != 0 {
		t.Errorf("Expected 0 remote databases on error, got %d", len(result))
	}
}

func TestDiffCollectManifestsFromDirectory(t *testing.T) {
	dir := t.TempDir()

	appContent := `apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: app1
spec:
  image: nginx:latest
`
	dbContent := `apiVersion: zenith.dev/v1alpha1
kind: Database
metadata:
  name: db1
spec:
  engine: postgresql
`

	os.MkdirAll(filepath.Join(dir, "apps"), 0755)
	os.MkdirAll(filepath.Join(dir, "databases"), 0755)
	os.WriteFile(filepath.Join(dir, "apps", "app1.yaml"), []byte(appContent), 0644)
	os.WriteFile(filepath.Join(dir, "databases", "db1.yaml"), []byte(dbContent), 0644)
	// Non-manifest files should be ignored
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)

	manifests, err := export.CollectManifests(dir)
	if err != nil {
		t.Fatalf("CollectManifests failed: %v", err)
	}

	if len(manifests) != 2 {
		t.Fatalf("Expected 2 manifests, got %d", len(manifests))
	}
}

func TestRenderAddedDiff_DoesNotPanic(t *testing.T) {
	entry := DiffEntry{
		Kind:   "App",
		Name:   "test-app",
		Action: "added",
		LocalSpec: map[string]interface{}{
			"image": "nginx:latest",
		},
	}
	// Calling render functions should not panic
	renderAddedDiff(entry)
}

func TestRenderModifiedDiff_DoesNotPanic(t *testing.T) {
	entry := DiffEntry{
		Kind:   "App",
		Name:   "test-app",
		Action: "modified",
		Changes: []FieldChange{
			{Field: "image", OldValue: "nginx:1.24", NewValue: "nginx:1.25"},
			{Field: "domain", OldValue: "", NewValue: "app.example.com"},
			{Field: "env", OldValue: "staging", NewValue: ""},
		},
	}
	renderModifiedDiff(entry)
}

func TestRenderDeletedDiff_DoesNotPanic(t *testing.T) {
	entry := DiffEntry{
		Kind:   "Database",
		Name:   "old-db",
		Action: "deleted",
		RemoteSpec: map[string]interface{}{
			"engine": "postgresql",
		},
	}
	renderDeletedDiff(entry)
}

func TestRenderUnchangedDiff_DoesNotPanic(t *testing.T) {
	entry := DiffEntry{
		Kind:   "App",
		Name:   "stable-app",
		Action: "unchanged",
	}
	renderUnchangedDiff(entry)
}

func TestRenderAddedDiff_NilSpec_DoesNotPanic(t *testing.T) {
	entry := DiffEntry{
		Kind:      "App",
		Name:      "empty-app",
		Action:    "added",
		LocalSpec: nil,
	}
	renderAddedDiff(entry)
}

func TestRenderDeletedDiff_NilSpec_DoesNotPanic(t *testing.T) {
	entry := DiffEntry{
		Kind:       "App",
		Name:       "empty-app",
		Action:     "deleted",
		RemoteSpec: nil,
	}
	renderDeletedDiff(entry)
}
