package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/capi"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/dotechhq/zenith/services/api/internal/store"
)

func newTestProvisioner() (*Provisioner, *store.MemoryCustomerRepository) {
	k8sClient := k8s.NewMemoryClient()
	capiClient := capi.NewClient(k8sClient)
	customerRepo := store.NewMemoryCustomerRepository()
	adminRepo := store.NewMemoryAdminRepository()
	p := NewProvisioner(capiClient, customerRepo, adminRepo)
	return p, customerRepo
}

func TestProvisionCluster(t *testing.T) {
	p, customerRepo := newTestProvisioner()
	ctx := context.Background()

	// Create a new customer
	cust, err := customerRepo.CreateCustomer(ctx, &dto.CreateCustomerInput{
		Name:         "TestCo",
		Domain:       "testco.dev",
		PlanID:       "plan-starter",
		ContactEmail: "admin@testco.dev",
		ContactName:  "Test Admin",
	})
	if err != nil {
		t.Fatalf("CreateCustomer failed: %v", err)
	}

	if cust.CAPIClusterName != "testco-dev" {
		t.Errorf("Expected cluster name 'testco-dev', got '%s'", cust.CAPIClusterName)
	}

	// Provision
	if err := p.ProvisionCluster(ctx, cust); err != nil {
		t.Fatalf("ProvisionCluster failed: %v", err)
	}

	// Verify status changed to provisioning
	updated, err := customerRepo.GetCustomer(ctx, cust.ID)
	if err != nil {
		t.Fatalf("GetCustomer failed: %v", err)
	}
	if updated.ClusterStatus != entities.ClusterStatusProvisioning {
		t.Errorf("Expected status 'provisioning', got '%s'", updated.ClusterStatus)
	}

	// Verify CAPI cluster was created
	cluster, err := p.GetCluster(ctx, "testco-dev")
	if err != nil {
		t.Fatalf("GetCluster failed: %v", err)
	}
	if cluster.Name != "testco-dev" {
		t.Errorf("Expected CAPI cluster name 'testco-dev', got '%s'", cluster.Name)
	}
	if cluster.Nodes != 3 {
		t.Errorf("Expected 3 nodes, got %d", cluster.Nodes)
	}
}

func TestTeardownCluster(t *testing.T) {
	p, customerRepo := newTestProvisioner()
	ctx := context.Background()

	// Use seeded customer (Embermind, has cluster)
	cust, err := customerRepo.GetCustomer(ctx, "cust-001")
	if err != nil {
		t.Fatalf("GetCustomer failed: %v", err)
	}

	// First provision so CAPI cluster exists
	if err := p.ProvisionCluster(ctx, cust); err != nil {
		t.Fatalf("ProvisionCluster failed: %v", err)
	}

	// Teardown
	if err := p.TeardownCluster(ctx, cust); err != nil {
		t.Fatalf("TeardownCluster failed: %v", err)
	}

	// Verify status changed to deleting
	updated, err := customerRepo.GetCustomer(ctx, cust.ID)
	if err != nil {
		t.Fatalf("GetCustomer failed: %v", err)
	}
	if updated.ClusterStatus != entities.ClusterStatusDeleting {
		t.Errorf("Expected status 'deleting', got '%s'", updated.ClusterStatus)
	}
}

func TestTeardownClusterNoClusterName(t *testing.T) {
	p, _ := newTestProvisioner()
	ctx := context.Background()

	// Customer with empty cluster name
	cust := &entities.Customer{
		ID:              "test-empty",
		CAPIClusterName: "",
	}

	// Should return nil (no-op)
	if err := p.TeardownCluster(ctx, cust); err != nil {
		t.Fatalf("Expected no error for empty cluster name, got: %v", err)
	}
}

func TestScaleCluster(t *testing.T) {
	p, customerRepo := newTestProvisioner()
	ctx := context.Background()

	// Provision a customer cluster first
	cust, _ := customerRepo.CreateCustomer(ctx, &dto.CreateCustomerInput{
		Name:         "ScaleCo",
		Domain:       "scaleco.io",
		PlanID:       "plan-pro",
		ContactEmail: "admin@scaleco.io",
		ContactName:  "Scale Admin",
	})
	if err := p.ProvisionCluster(ctx, cust); err != nil {
		t.Fatalf("ProvisionCluster failed: %v", err)
	}

	// Refresh customer after provisioning
	cust, _ = customerRepo.GetCustomer(ctx, cust.ID)

	// Scale to 5 nodes
	if err := p.ScaleCluster(ctx, cust, 5); err != nil {
		t.Fatalf("ScaleCluster failed: %v", err)
	}

	// Verify DB updated
	updated, _ := customerRepo.GetCustomer(ctx, cust.ID)
	if updated.ClusterNodes != 5 {
		t.Errorf("Expected 5 nodes in DB, got %d", updated.ClusterNodes)
	}

	// Verify CAPI cluster updated
	cluster, _ := p.GetCluster(ctx, cust.CAPIClusterName)
	if cluster.Nodes != 5 {
		t.Errorf("Expected 5 nodes in CAPI, got %d", cluster.Nodes)
	}
}

func TestUpgradeCluster(t *testing.T) {
	p, customerRepo := newTestProvisioner()
	ctx := context.Background()

	cust, _ := customerRepo.CreateCustomer(ctx, &dto.CreateCustomerInput{
		Name:         "UpgradeCo",
		Domain:       "upgradeco.io",
		PlanID:       "plan-pro",
		ContactEmail: "admin@upgradeco.io",
		ContactName:  "Upgrade Admin",
	})
	if err := p.ProvisionCluster(ctx, cust); err != nil {
		t.Fatalf("ProvisionCluster failed: %v", err)
	}

	cust, _ = customerRepo.GetCustomer(ctx, cust.ID)

	// Upgrade to v1.32.0
	if err := p.UpgradeCluster(ctx, cust, "v1.32.0"); err != nil {
		t.Fatalf("UpgradeCluster failed: %v", err)
	}

	// Verify DB updated
	updated, _ := customerRepo.GetCustomer(ctx, cust.ID)
	if updated.ClusterK8sVersion != "v1.32.0" {
		t.Errorf("Expected k8s version 'v1.32.0' in DB, got '%s'", updated.ClusterK8sVersion)
	}

	// Verify CAPI cluster updated
	cluster, _ := p.GetCluster(ctx, cust.CAPIClusterName)
	if cluster.K8sVersion != "v1.32.0" {
		t.Errorf("Expected k8s version 'v1.32.0' in CAPI, got '%s'", cluster.K8sVersion)
	}
}

func TestDomainToClusterName(t *testing.T) {
	tests := []struct {
		domain   string
		expected string
	}{
		{"newco.app", "newco-app"},
		{"multi-dash.com", "multi-dash-com"},
		{"deep.nested.domain.com", "deep-nested-domain-com"},
	}

	for _, tc := range tests {
		// Test indirectly via CreateCustomer which sets CAPIClusterName
		repo := store.NewMemoryCustomerRepository()
		cust, err := repo.CreateCustomer(context.Background(), &dto.CreateCustomerInput{
			Name:         "Test",
			Domain:       tc.domain,
			PlanID:       "plan-starter",
			ContactEmail: "a@b.com",
		})
		if err != nil {
			t.Fatalf("CreateCustomer(%s) failed: %v", tc.domain, err)
		}
		if cust.CAPIClusterName != tc.expected {
			t.Errorf("domainToClusterName(%s) = '%s', want '%s'", tc.domain, cust.CAPIClusterName, tc.expected)
		}
	}
}

func TestSyncUpdatesClusterStatus(t *testing.T) {
	p, customerRepo := newTestProvisioner()
	ctx := context.Background()

	// Create customer and provision
	cust, _ := customerRepo.CreateCustomer(ctx, &dto.CreateCustomerInput{
		Name:         "SyncCo",
		Domain:       "syncco.dev",
		PlanID:       "plan-starter",
		ContactEmail: "admin@syncco.dev",
	})
	if err := p.ProvisionCluster(ctx, cust); err != nil {
		t.Fatalf("ProvisionCluster failed: %v", err)
	}

	// Customer should be in provisioning state
	updated, _ := customerRepo.GetCustomer(ctx, cust.ID)
	if updated.ClusterStatus != entities.ClusterStatusProvisioning {
		t.Fatalf("Expected provisioning, got %s", updated.ClusterStatus)
	}

	// Run sync — the memory k8s client sets status "healthy" by default
	p.syncOnce()

	// Verify status updated to running
	synced, _ := customerRepo.GetCustomer(ctx, cust.ID)
	if synced.ClusterStatus != entities.ClusterStatusRunning {
		t.Errorf("Expected status 'running' after sync, got '%s'", synced.ClusterStatus)
	}
}

func TestStartAndStop(t *testing.T) {
	p, _ := newTestProvisioner()

	p.StartSync(100 * time.Millisecond)
	time.Sleep(250 * time.Millisecond)
	p.Stop()
	// Should not panic or hang
}
