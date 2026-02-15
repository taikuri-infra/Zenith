package k8s

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
)

// ---------- Interface Compliance ----------

func TestMemoryClientImplementsInterface(t *testing.T) {
	var _ Client = (*MemoryClient)(nil)
}

// ---------- NewMemoryClient ----------

func TestNewMemoryClient(t *testing.T) {
	c := NewMemoryClient()
	if c == nil {
		t.Fatal("Expected non-nil MemoryClient")
	}
	if c.objects == nil {
		t.Fatal("Expected non-nil objects map")
	}
}

// ---------- CreateCRD ----------

func TestCreateCRD(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	obj := &CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: ObjectMeta{
			Name:      "web-app",
			Namespace: "zenith-proj1",
		},
		Spec: json.RawMessage(`{"image":"nginx:latest"}`),
	}

	err := c.CreateCRD(ctx, obj)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestCreateCRDDuplicate(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	obj := &CRDObject{
		Kind: "App",
		Metadata: ObjectMeta{
			Name:      "web-app",
			Namespace: "zenith-proj1",
		},
	}

	err := c.CreateCRD(ctx, obj)
	if err != nil {
		t.Fatalf("First create should succeed, got: %v", err)
	}

	err = c.CreateCRD(ctx, obj)
	if err == nil {
		t.Error("Expected error for duplicate create")
	}
}

// ---------- GetCRD ----------

func TestGetCRD(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	spec := json.RawMessage(`{"image":"nginx:latest","replicas":2}`)
	obj := &CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: ObjectMeta{
			Name:      "web-app",
			Namespace: "zenith-proj1",
			Labels:    map[string]string{"app": "web"},
		},
		Spec: spec,
	}

	c.CreateCRD(ctx, obj)

	result, err := c.GetCRD(ctx, "App", "zenith-proj1", "web-app")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Metadata.Name != "web-app" {
		t.Errorf("Expected name 'web-app', got '%s'", result.Metadata.Name)
	}
	if result.Metadata.Namespace != "zenith-proj1" {
		t.Errorf("Expected namespace 'zenith-proj1', got '%s'", result.Metadata.Namespace)
	}
	if result.Metadata.Labels["app"] != "web" {
		t.Errorf("Expected label app=web, got '%s'", result.Metadata.Labels["app"])
	}
}

func TestGetCRDNotFound(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	_, err := c.GetCRD(ctx, "App", "zenith-proj1", "nonexistent")
	if err == nil {
		t.Error("Expected error for not found")
	}
}

// ---------- UpdateCRD ----------

func TestUpdateCRD(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	obj := &CRDObject{
		Kind: "App",
		Metadata: ObjectMeta{
			Name:      "web-app",
			Namespace: "zenith-proj1",
		},
		Spec: json.RawMessage(`{"image":"nginx:1.0"}`),
	}

	c.CreateCRD(ctx, obj)

	// Update spec
	obj.Spec = json.RawMessage(`{"image":"nginx:2.0"}`)
	err := c.UpdateCRD(ctx, obj)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify update
	result, _ := c.GetCRD(ctx, "App", "zenith-proj1", "web-app")
	var spec map[string]interface{}
	json.Unmarshal(result.Spec, &spec)

	if spec["image"] != "nginx:2.0" {
		t.Errorf("Expected updated image 'nginx:2.0', got '%v'", spec["image"])
	}
}

func TestUpdateCRDNotFound(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	obj := &CRDObject{
		Kind: "App",
		Metadata: ObjectMeta{
			Name:      "nonexistent",
			Namespace: "zenith-proj1",
		},
	}

	err := c.UpdateCRD(ctx, obj)
	if err == nil {
		t.Error("Expected error for updating non-existent object")
	}
}

// ---------- DeleteCRD ----------

func TestDeleteCRD(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	obj := &CRDObject{
		Kind: "App",
		Metadata: ObjectMeta{
			Name:      "to-delete",
			Namespace: "zenith-proj1",
		},
	}

	c.CreateCRD(ctx, obj)

	err := c.DeleteCRD(ctx, "App", "zenith-proj1", "to-delete")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify deleted
	_, err = c.GetCRD(ctx, "App", "zenith-proj1", "to-delete")
	if err == nil {
		t.Error("Expected error after deletion")
	}
}

func TestDeleteCRDNotFound(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	err := c.DeleteCRD(ctx, "App", "zenith-proj1", "nonexistent")
	if err == nil {
		t.Error("Expected error for deleting non-existent object")
	}
}

// ---------- ListCRDs ----------

func TestListCRDs(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	// Create 3 apps in the same namespace
	for _, name := range []string{"app1", "app2", "app3"} {
		c.CreateCRD(ctx, &CRDObject{
			Kind: "App",
			Metadata: ObjectMeta{
				Name:      name,
				Namespace: "zenith-proj1",
			},
		})
	}

	result, err := c.ListCRDs(ctx, "App", "zenith-proj1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}
}

func TestListCRDsEmpty(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	result, err := c.ListCRDs(ctx, "App", "zenith-proj1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 items, got %d", len(result))
	}
}

func TestListCRDsFiltersByKind(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	ns := "zenith-proj1"

	c.CreateCRD(ctx, &CRDObject{
		Kind:     "App",
		Metadata: ObjectMeta{Name: "app1", Namespace: ns},
	})
	c.CreateCRD(ctx, &CRDObject{
		Kind:     "Database",
		Metadata: ObjectMeta{Name: "db1", Namespace: ns},
	})

	apps, _ := c.ListCRDs(ctx, "App", ns)
	if len(apps) != 1 {
		t.Errorf("Expected 1 App, got %d", len(apps))
	}

	dbs, _ := c.ListCRDs(ctx, "Database", ns)
	if len(dbs) != 1 {
		t.Errorf("Expected 1 Database, got %d", len(dbs))
	}
}

func TestListCRDsFiltersByNamespace(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	c.CreateCRD(ctx, &CRDObject{
		Kind:     "App",
		Metadata: ObjectMeta{Name: "app1", Namespace: "zenith-proj1"},
	})
	c.CreateCRD(ctx, &CRDObject{
		Kind:     "App",
		Metadata: ObjectMeta{Name: "app2", Namespace: "zenith-proj2"},
	})

	proj1Apps, _ := c.ListCRDs(ctx, "App", "zenith-proj1")
	if len(proj1Apps) != 1 {
		t.Errorf("Expected 1 app in proj1, got %d", len(proj1Apps))
	}

	proj2Apps, _ := c.ListCRDs(ctx, "App", "zenith-proj2")
	if len(proj2Apps) != 1 {
		t.Errorf("Expected 1 app in proj2, got %d", len(proj2Apps))
	}
}

func TestListCRDsEmptyNamespace(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	// Create a cluster-scoped resource (no namespace)
	c.CreateCRD(ctx, &CRDObject{
		Kind:     "Project",
		Metadata: ObjectMeta{Name: "proj1"},
	})
	c.CreateCRD(ctx, &CRDObject{
		Kind:     "Project",
		Metadata: ObjectMeta{Name: "proj2"},
	})

	result, _ := c.ListCRDs(ctx, "Project", "")
	if len(result) != 2 {
		t.Errorf("Expected 2 cluster-scoped items, got %d", len(result))
	}
}

// ---------- ObjectKey ----------

func TestObjectKey(t *testing.T) {
	tests := []struct {
		kind, namespace, name string
		expected              string
	}{
		{"App", "zenith-proj1", "web-app", "App/zenith-proj1/web-app"},
		{"Database", "zenith-proj1", "db1", "Database/zenith-proj1/db1"},
		{"Project", "", "proj1", "Project//proj1"},
	}

	for _, tt := range tests {
		result := objectKey(tt.kind, tt.namespace, tt.name)
		if result != tt.expected {
			t.Errorf("objectKey(%q, %q, %q) = %q, want %q", tt.kind, tt.namespace, tt.name, result, tt.expected)
		}
	}
}

// ---------- Concurrency ----------

func TestConcurrentAccess(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	var wg sync.WaitGroup
	count := 100

	// Concurrent creates
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := "app-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
			c.CreateCRD(ctx, &CRDObject{
				Kind:     "App",
				Metadata: ObjectMeta{Name: name, Namespace: "zenith-proj1"},
			})
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.ListCRDs(ctx, "App", "zenith-proj1")
		}()
	}
	wg.Wait()
}

// ---------- Annotations and Labels ----------

func TestCRDObjectAnnotations(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	obj := &CRDObject{
		Kind: "App",
		Metadata: ObjectMeta{
			Name:      "annotated-app",
			Namespace: "zenith-proj1",
			Annotations: map[string]string{
				"zenith.dev/redeploy-at": "2026-01-15T10:00:00Z",
			},
			Labels: map[string]string{
				"zenith.dev/project": "proj1",
			},
		},
	}

	c.CreateCRD(ctx, obj)

	result, _ := c.GetCRD(ctx, "App", "zenith-proj1", "annotated-app")
	if result.Metadata.Annotations["zenith.dev/redeploy-at"] != "2026-01-15T10:00:00Z" {
		t.Errorf("Expected annotation preserved, got '%s'", result.Metadata.Annotations["zenith.dev/redeploy-at"])
	}
	if result.Metadata.Labels["zenith.dev/project"] != "proj1" {
		t.Errorf("Expected label preserved, got '%s'", result.Metadata.Labels["zenith.dev/project"])
	}
}

// ---------- Update preserves all fields ----------

func TestUpdateCRDPreservesFields(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	obj := &CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Database",
		Metadata: ObjectMeta{
			Name:      "main-db",
			Namespace: "zenith-proj1",
			Labels: map[string]string{
				"engine": "postgresql",
			},
		},
		Spec: json.RawMessage(`{"engine":"postgresql","version":"16"}`),
	}

	c.CreateCRD(ctx, obj)

	// Update spec but keep everything else
	obj.Spec = json.RawMessage(`{"engine":"postgresql","version":"16","storage":"50Gi"}`)
	c.UpdateCRD(ctx, obj)

	result, _ := c.GetCRD(ctx, "Database", "zenith-proj1", "main-db")

	if result.APIVersion != "zenith.dev/v1alpha1" {
		t.Errorf("Expected APIVersion preserved, got '%s'", result.APIVersion)
	}
	if result.Metadata.Labels["engine"] != "postgresql" {
		t.Errorf("Expected label preserved, got '%s'", result.Metadata.Labels["engine"])
	}

	var spec map[string]interface{}
	json.Unmarshal(result.Spec, &spec)
	if spec["storage"] != "50Gi" {
		t.Errorf("Expected updated storage field, got '%v'", spec["storage"])
	}
}

// ---------- Nil context ----------

func TestNilContext(t *testing.T) {
	c := NewMemoryClient()

	obj := &CRDObject{
		Kind:     "App",
		Metadata: ObjectMeta{Name: "nil-ctx", Namespace: "ns"},
	}

	// nil context should still work (memory client ignores context)
	err := c.CreateCRD(nil, obj)
	if err != nil {
		t.Fatalf("Expected no error with nil context, got: %v", err)
	}

	result, err := c.GetCRD(nil, "App", "ns", "nil-ctx")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Metadata.Name != "nil-ctx" {
		t.Errorf("Expected name 'nil-ctx', got '%s'", result.Metadata.Name)
	}

	result.Spec = json.RawMessage(`{"updated":true}`)
	err = c.UpdateCRD(nil, result)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	err = c.DeleteCRD(nil, "App", "ns", "nil-ctx")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	items, err := c.ListCRDs(nil, "App", "ns")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items after delete, got %d", len(items))
	}
}

// ---------- CRDObject Status field ----------

func TestCRDObjectStatusField(t *testing.T) {
	c := NewMemoryClient()
	ctx := context.Background()

	obj := &CRDObject{
		Kind: "App",
		Metadata: ObjectMeta{
			Name:      "status-app",
			Namespace: "ns",
		},
		Spec:   json.RawMessage(`{"image":"nginx"}`),
		Status: json.RawMessage(`{"phase":"Running","readyReplicas":3}`),
	}

	c.CreateCRD(ctx, obj)

	result, _ := c.GetCRD(ctx, "App", "ns", "status-app")
	if result.Status == nil {
		t.Fatal("Expected non-nil status")
	}

	var status map[string]interface{}
	json.Unmarshal(result.Status, &status)

	if status["phase"] != "Running" {
		t.Errorf("Expected phase 'Running', got '%v'", status["phase"])
	}
}
