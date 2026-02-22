package capiclient

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/k8s"
)

func TestCreateAndGetCluster(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	input := dto.CreateClusterInput{
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

	input := dto.CreateClusterInput{
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

	input := dto.CreateClusterInput{
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
		input := dto.CreateClusterInput{
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

	input := dto.CreateClusterInput{
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

func TestScaleCluster(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	input := dto.CreateClusterInput{
		Name:       "scale-me",
		Region:     "fsn1",
		Type:       "shared",
		Nodes:      2,
		K8sVersion: "v1.30.2",
	}
	if _, err := client.CreateCluster(ctx, input); err != nil {
		t.Fatalf("CreateCluster failed: %v", err)
	}

	if err := client.ScaleCluster(ctx, "scale-me", 5); err != nil {
		t.Fatalf("ScaleCluster failed: %v", err)
	}

	// Verify nodes was updated
	cluster, err := client.GetCluster(ctx, "scale-me")
	if err != nil {
		t.Fatalf("GetCluster after scale failed: %v", err)
	}

	if cluster.Nodes != 5 {
		t.Errorf("Expected 5 nodes after scale, got %d", cluster.Nodes)
	}
}

func TestScaleClusterNotFound(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	err := client.ScaleCluster(ctx, "nonexistent", 3)
	if err == nil {
		t.Error("Expected error for nonexistent scale, got nil")
	}
}

func TestUpgradeCluster(t *testing.T) {
	k8sClient := k8s.NewMemoryClient()
	client := NewClient(k8sClient)
	ctx := context.Background()

	input := dto.CreateClusterInput{
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

