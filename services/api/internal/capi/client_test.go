package capi

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/dotechhq/zenith/services/api/internal/models"
)

func TestCreateAndGetCluster(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	input := models.CreateClusterInput{
		Name:       "test-cluster",
		Region:     "fsn1",
		Type:       "shared",
		Nodes:      3,
		K8sVersion: "v1.30.2",
	}

	cluster, err := client.CreateCluster(ctx, input)
	if err != nil {
		t.Fatalf("CreateCluster failed: %v", err)
	}

	if cluster.Name != "test-cluster" {
		t.Errorf("Expected name 'test-cluster', got '%s'", cluster.Name)
	}
	if cluster.K8sVersion != "v1.30.2" {
		t.Errorf("Expected k8s version 'v1.30.2', got '%s'", cluster.K8sVersion)
	}
	if cluster.Nodes != 3 {
		t.Errorf("Expected 3 nodes, got %d", cluster.Nodes)
	}
	if cluster.Region != "fsn1" {
		t.Errorf("Expected region 'fsn1', got '%s'", cluster.Region)
	}
	if cluster.Type != "shared" {
		t.Errorf("Expected type 'shared', got '%s'", cluster.Type)
	}

	// Get the same cluster
	got, err := client.GetCluster(ctx, "test-cluster")
	if err != nil {
		t.Fatalf("GetCluster failed: %v", err)
	}
	if got.Name != cluster.Name {
		t.Errorf("Expected name '%s', got '%s'", cluster.Name, got.Name)
	}
}

func TestCreateClusterWithTenant(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	input := models.CreateClusterInput{
		Name:       "dedicated-cluster",
		Region:     "nbg1",
		Type:       "dedicated",
		Tenant:     "acme-corp",
		Nodes:      5,
		K8sVersion: "v1.30.2",
	}

	cluster, err := client.CreateCluster(ctx, input)
	if err != nil {
		t.Fatalf("CreateCluster failed: %v", err)
	}

	if cluster.Tenant != "acme-corp" {
		t.Errorf("Expected tenant 'acme-corp', got '%s'", cluster.Tenant)
	}
	if cluster.Type != "dedicated" {
		t.Errorf("Expected type 'dedicated', got '%s'", cluster.Type)
	}
}

func TestCreateClusterDuplicate(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	input := models.CreateClusterInput{
		Name:       "dup-cluster",
		Region:     "fsn1",
		Type:       "shared",
		Nodes:      1,
		K8sVersion: "v1.30.2",
	}

	_, err := client.CreateCluster(ctx, input)
	if err != nil {
		t.Fatalf("First CreateCluster failed: %v", err)
	}

	_, err = client.CreateCluster(ctx, input)
	if err == nil {
		t.Error("Expected error on duplicate create, got nil")
	}
}

func TestGetClusterNotFound(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	_, err := client.GetCluster(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent cluster, got nil")
	}
}

func TestListClusters(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	// Create 3 clusters
	for _, name := range []string{"alpha", "beta", "gamma"} {
		input := models.CreateClusterInput{
			Name:       name,
			Region:     "fsn1",
			Type:       "shared",
			Nodes:      1,
			K8sVersion: "v1.30.2",
		}
		if _, err := client.CreateCluster(ctx, input); err != nil {
			t.Fatalf("CreateCluster %s failed: %v", name, err)
		}
	}

	clusters, err := client.ListClusters(ctx)
	if err != nil {
		t.Fatalf("ListClusters failed: %v", err)
	}

	if len(clusters) != 3 {
		t.Errorf("Expected 3 clusters, got %d", len(clusters))
	}
}

func TestListClustersEmpty(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	clusters, err := client.ListClusters(ctx)
	if err != nil {
		t.Fatalf("ListClusters failed: %v", err)
	}

	if len(clusters) != 0 {
		t.Errorf("Expected 0 clusters, got %d", len(clusters))
	}
}

func TestDeleteCluster(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	input := models.CreateClusterInput{
		Name:       "to-delete",
		Region:     "fsn1",
		Type:       "shared",
		Nodes:      1,
		K8sVersion: "v1.30.2",
	}
	if _, err := client.CreateCluster(ctx, input); err != nil {
		t.Fatalf("CreateCluster failed: %v", err)
	}

	if err := client.DeleteCluster(ctx, "to-delete"); err != nil {
		t.Fatalf("DeleteCluster failed: %v", err)
	}

	// Verify it's gone
	_, err := client.GetCluster(ctx, "to-delete")
	if err == nil {
		t.Error("Expected error after deletion, got nil")
	}
}

func TestDeleteClusterNotFound(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	err := client.DeleteCluster(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent delete, got nil")
	}
}

func TestUpgradeCluster(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	input := models.CreateClusterInput{
		Name:       "upgrade-me",
		Region:     "fsn1",
		Type:       "shared",
		Nodes:      2,
		K8sVersion: "v1.28.0",
	}
	if _, err := client.CreateCluster(ctx, input); err != nil {
		t.Fatalf("CreateCluster failed: %v", err)
	}

	if err := client.UpgradeCluster(ctx, "upgrade-me", "v1.30.2"); err != nil {
		t.Fatalf("UpgradeCluster failed: %v", err)
	}

	// Verify version was updated
	cluster, err := client.GetCluster(ctx, "upgrade-me")
	if err != nil {
		t.Fatalf("GetCluster after upgrade failed: %v", err)
	}

	if cluster.K8sVersion != "v1.30.2" {
		t.Errorf("Expected k8s version 'v1.30.2' after upgrade, got '%s'", cluster.K8sVersion)
	}
}

func TestUpgradeClusterNotFound(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	err := client.UpgradeCluster(ctx, "nonexistent", "v1.30.2")
	if err == nil {
		t.Error("Expected error for nonexistent upgrade, got nil")
	}
}

// ---------- MemoryStore ----------

func TestMemoryStoreSettings(t *testing.T) {
	store := NewMemoryStore()

	settings := store.GetSettings()
	if settings.PlatformName != "Zenith" {
		t.Errorf("Expected default platform name 'Zenith', got '%s'", settings.PlatformName)
	}

	updated := store.UpdateSettings(&models.PlatformSettings{
		PlatformName: "My Platform",
		BaseDomain:   "example.com",
	})
	if updated.PlatformName != "My Platform" {
		t.Errorf("Expected updated name 'My Platform', got '%s'", updated.PlatformName)
	}
	if updated.BaseDomain != "example.com" {
		t.Errorf("Expected updated domain 'example.com', got '%s'", updated.BaseDomain)
	}
	// Provider should remain default
	if updated.Provider != "Hetzner Cloud" {
		t.Errorf("Expected provider to remain 'Hetzner Cloud', got '%s'", updated.Provider)
	}
}

func TestMemoryStoreModules(t *testing.T) {
	store := NewMemoryStore()

	modules := store.ListModules()
	if len(modules) == 0 {
		t.Fatal("Expected non-empty default modules")
	}

	// Find a module that has an update
	mod, err := store.GetModule("Zenith Operator")
	if err != nil {
		t.Fatalf("GetModule failed: %v", err)
	}
	if mod.Status != "update_available" {
		t.Errorf("Expected status 'update_available', got '%s'", mod.Status)
	}

	// Update it
	updated, err := store.UpdateModule("Zenith Operator")
	if err != nil {
		t.Fatalf("UpdateModule failed: %v", err)
	}
	if updated.Status != "up_to_date" {
		t.Errorf("Expected status 'up_to_date' after update, got '%s'", updated.Status)
	}
	if updated.Installed != updated.Latest {
		t.Error("Expected installed == latest after update")
	}
}

func TestMemoryStoreGetModuleNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.GetModule("Nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent module")
	}
}

func TestMemoryStoreUpdateModuleNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.UpdateModule("Nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent module update")
	}
}

func TestMemoryStoreAuditLog(t *testing.T) {
	store := NewMemoryStore()

	entries := store.ListAuditLog(50, 0)
	initialCount := len(entries)

	store.AddAuditEntry(models.AuditEntry{
		Time:   "10:00",
		Actor:  "test",
		Action: "test action",
	})

	entries = store.ListAuditLog(50, 0)
	if len(entries) != initialCount+1 {
		t.Errorf("Expected %d entries after add, got %d", initialCount+1, len(entries))
	}

	// New entry should be first (prepended)
	if entries[0].Action != "test action" {
		t.Errorf("Expected newest entry first, got '%s'", entries[0].Action)
	}
}

func TestMemoryStoreAuditLogLimitOffset(t *testing.T) {
	store := NewMemoryStore()

	entries := store.ListAuditLog(2, 0)
	if len(entries) > 2 {
		t.Errorf("Expected at most 2 entries with limit=2, got %d", len(entries))
	}

	allEntries := store.ListAuditLog(50, 0)
	if len(allEntries) > 2 {
		entries = store.ListAuditLog(50, 2)
		expectedLen := len(allEntries) - 2
		if len(entries) != expectedLen {
			t.Errorf("Expected %d entries with offset=2, got %d", expectedLen, len(entries))
		}
	}

	// Offset beyond length
	entries = store.ListAuditLog(50, 9999)
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with large offset, got %d", len(entries))
	}
}

func TestMemoryStorePlatformUpdate(t *testing.T) {
	store := NewMemoryStore()
	update := store.GetPlatformUpdate()

	if update.Version == "" {
		t.Error("Expected non-empty version")
	}
	if update.Current == "" {
		t.Error("Expected non-empty current")
	}
	if len(update.Features) == 0 {
		t.Error("Expected non-empty features")
	}
}

func TestMemoryStoreUpdateHistory(t *testing.T) {
	store := NewMemoryStore()
	history := store.ListUpdateHistory()

	if len(history) == 0 {
		t.Error("Expected non-empty update history")
	}
}
