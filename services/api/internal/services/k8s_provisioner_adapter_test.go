package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
)

func newTestProvisionerAdapter() (*K8sProvisionerAdapter, *k8sclient.MemoryClient) {
	k8s := k8sclient.NewMemoryClient()
	adapter := NewK8sProvisionerAdapter(k8s)
	return adapter, k8s
}

// --- ApplyUnstructured tests ---

func TestApplyUnstructured_Secret(t *testing.T) {
	adapter, k8s := newTestProvisionerAdapter()
	ctx := context.Background()

	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name": "my-secret",
		},
		"stringData": map[string]interface{}{
			"username": "admin",
			"password": "s3cret",
		},
	}

	err := adapter.ApplyUnstructured(ctx, "test-ns", obj)
	if err != nil {
		t.Fatalf("ApplyUnstructured Secret failed: %v", err)
	}

	// Verify secret was created
	data, err := k8s.GetSecret(ctx, "test-ns", "my-secret")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(data["username"]) != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", string(data["username"]))
	}
}

func TestApplyUnstructured_SecretWithData(t *testing.T) {
	adapter, k8s := newTestProvisionerAdapter()
	ctx := context.Background()

	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name": "data-secret",
		},
		"data": map[string]interface{}{
			"token": "base64-encoded-data",
		},
	}

	err := adapter.ApplyUnstructured(ctx, "test-ns", obj)
	if err != nil {
		t.Fatalf("ApplyUnstructured Secret with data failed: %v", err)
	}

	data, err := k8s.GetSecret(ctx, "test-ns", "data-secret")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if string(data["token"]) != "base64-encoded-data" {
		t.Errorf("Expected token, got '%s'", string(data["token"]))
	}
}

func TestApplyUnstructured_ConfigMap(t *testing.T) {
	adapter, _ := newTestProvisionerAdapter()
	ctx := context.Background()

	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": "my-config",
		},
		"data": map[string]interface{}{
			"key1": "value1",
		},
	}

	err := adapter.ApplyUnstructured(ctx, "test-ns", obj)
	if err != nil {
		t.Fatalf("ApplyUnstructured ConfigMap failed: %v", err)
	}
}

func TestApplyUnstructured_CRD(t *testing.T) {
	adapter, _ := newTestProvisionerAdapter()
	ctx := context.Background()

	obj := map[string]interface{}{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Cluster",
		"metadata": map[string]interface{}{
			"name": "my-pg-cluster",
			"labels": map[string]interface{}{
				"app": "postgres",
			},
		},
		"spec": map[string]interface{}{
			"instances": 1,
		},
	}

	err := adapter.ApplyUnstructured(ctx, "test-ns", obj)
	if err != nil {
		t.Fatalf("ApplyUnstructured CRD failed: %v", err)
	}
}

func TestApplyUnstructured_ZenithCRD(t *testing.T) {
	adapter, _ := newTestProvisionerAdapter()
	ctx := context.Background()

	obj := map[string]interface{}{
		"apiVersion": "zenith.dev/v1alpha1",
		"kind":       "Database",
		"metadata": map[string]interface{}{
			"name": "my-db",
		},
		"spec": map[string]interface{}{
			"size": "10Gi",
		},
	}

	err := adapter.ApplyUnstructured(ctx, "test-ns", obj)
	if err != nil {
		t.Fatalf("ApplyUnstructured Zenith CRD failed: %v", err)
	}
}

// --- DeleteResource tests ---

func TestDeleteResource_Secret(t *testing.T) {
	adapter, k8s := newTestProvisionerAdapter()
	ctx := context.Background()

	// Create a secret first
	k8s.CreateSecret(ctx, "test-ns", "to-delete", map[string][]byte{"k": []byte("v")}, nil)

	err := adapter.DeleteResource(ctx, "test-ns", "v1", "Secret", "to-delete")
	if err != nil {
		t.Fatalf("DeleteResource Secret failed: %v", err)
	}

	// Should be gone
	_, err = k8s.GetSecret(ctx, "test-ns", "to-delete")
	if err == nil {
		t.Error("Expected error getting deleted secret")
	}
}

func TestDeleteResource_ConfigMap(t *testing.T) {
	adapter, k8s := newTestProvisionerAdapter()
	ctx := context.Background()

	k8s.CreateConfigMap(ctx, "test-ns", "to-delete-cm", map[string]string{"k": "v"})

	err := adapter.DeleteResource(ctx, "test-ns", "v1", "ConfigMap", "to-delete-cm")
	if err != nil {
		t.Fatalf("DeleteResource ConfigMap failed: %v", err)
	}
}

func TestDeleteResource_CRD(t *testing.T) {
	adapter, _ := newTestProvisionerAdapter()
	ctx := context.Background()

	// For CRD, the memory client will return an error for non-existent resources
	// but the delete should not panic
	err := adapter.DeleteResource(ctx, "test-ns", "postgresql.cnpg.io/v1", "Cluster", "nonexistent")
	// Error is acceptable for non-existent resource
	_ = err
}

// --- GetCRDStatus tests ---

func TestGetCRDStatus_NotFound(t *testing.T) {
	adapter, _ := newTestProvisionerAdapter()
	ctx := context.Background()

	_, err := adapter.GetCRDStatus(ctx, "postgresql.cnpg.io/v1", "Cluster", "test-ns", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent CRD")
	}
}

// --- extractLabels tests ---

func TestExtractLabels(t *testing.T) {
	metadata := map[string]interface{}{
		"labels": map[string]interface{}{
			"app":     "postgres",
			"version": "16",
		},
	}
	labels := extractLabels(metadata)
	if labels["app"] != "postgres" {
		t.Errorf("Expected label app=postgres, got '%s'", labels["app"])
	}
	if labels["version"] != "16" {
		t.Errorf("Expected label version=16, got '%s'", labels["version"])
	}
}

func TestExtractLabels_NoLabels(t *testing.T) {
	metadata := map[string]interface{}{}
	labels := extractLabels(metadata)
	if len(labels) != 0 {
		t.Errorf("Expected empty labels, got %d", len(labels))
	}
}

func TestExtractLabels_Nil(t *testing.T) {
	labels := extractLabels(nil)
	if len(labels) != 0 {
		t.Errorf("Expected empty labels for nil metadata, got %d", len(labels))
	}
}

// --- SecretWithLabels tests ---

func TestApplyUnstructured_SecretWithLabels(t *testing.T) {
	adapter, _ := newTestProvisionerAdapter()
	ctx := context.Background()

	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name": "labeled-secret",
			"labels": map[string]interface{}{
				"managed-by": "zenith",
			},
		},
		"stringData": map[string]interface{}{
			"key": "value",
		},
	}

	err := adapter.ApplyUnstructured(ctx, "test-ns", obj)
	if err != nil {
		t.Fatalf("ApplyUnstructured Secret with labels failed: %v", err)
	}
}
