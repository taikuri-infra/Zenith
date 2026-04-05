package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
)

func newTestCustomerService() (*CustomerService, *memory.MemoryCustomerRepository, *memory.MemoryAdminRepository) {
	custRepo := memory.NewMemoryCustomerRepository()
	adminRepo := memory.NewMemoryAdminRepository()
	svc := NewCustomerService(custRepo, adminRepo, nil)
	return svc, custRepo, adminRepo
}

// --- tenantNamespace tests ---

func TestTenantNamespace(t *testing.T) {
	cases := []struct {
		domain   string
		expected string
	}{
		{"embermind.app", "zenith-embermind-app"},
		{"acme-corp.com", "zenith-acme-corp-com"},
		{"simple", "zenith-simple"},
		{"a.b.c", "zenith-a-b-c"},
		{"under_score.io", "zenith-under-score-io"},
	}
	for _, tc := range cases {
		got := tenantNamespace(tc.domain)
		if got != tc.expected {
			t.Errorf("tenantNamespace(%q) = %q, want %q", tc.domain, got, tc.expected)
		}
	}
}

// --- ListCustomers tests ---

func TestListCustomers_Seeded(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	customers, err := svc.ListCustomers(ctx)
	if err != nil {
		t.Fatalf("ListCustomers failed: %v", err)
	}
	// Memory repo is pre-seeded with 3 customers
	if len(customers) < 3 {
		t.Errorf("Expected at least 3 seeded customers, got %d", len(customers))
	}
}

// --- GetCustomerStats tests ---

func TestGetCustomerStats(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	stats, err := svc.GetCustomerStats(ctx)
	if err != nil {
		t.Fatalf("GetCustomerStats failed: %v", err)
	}
	if stats.TotalCustomers < 3 {
		t.Errorf("Expected at least 3 total customers, got %d", stats.TotalCustomers)
	}
	if stats.ActiveCustomers == 0 {
		t.Error("Expected some active customers")
	}
}

// --- GetCustomer tests ---

func TestGetCustomer_Exists(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	customer, err := svc.GetCustomer(ctx, "cust-001")
	if err != nil {
		t.Fatalf("GetCustomer failed: %v", err)
	}
	if customer.Name != "Embermind" {
		t.Errorf("Expected name 'Embermind', got '%s'", customer.Name)
	}
}

func TestGetCustomer_NotFound(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	_, err := svc.GetCustomer(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent customer")
	}
}

// --- CreateCustomer tests ---

func TestCreateCustomer_Success(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	customer, err := svc.CreateCustomer(ctx, &dto.CreateCustomerInput{
		Name:         "NewCo",
		Domain:       "newco.io",
		PlanID:       "plan-starter",
		ContactEmail: "admin@newco.io",
		ContactName:  "Admin User",
	}, "admin")

	if err != nil {
		t.Fatalf("CreateCustomer failed: %v", err)
	}
	if customer.Name != "NewCo" {
		t.Errorf("Expected name 'NewCo', got '%s'", customer.Name)
	}
	if customer.Domain != "newco.io" {
		t.Errorf("Expected domain 'newco.io', got '%s'", customer.Domain)
	}
	if customer.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", customer.Status)
	}
}

func TestCreateCustomer_DuplicateDomain(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	_, err := svc.CreateCustomer(ctx, &dto.CreateCustomerInput{
		Name:         "Dup",
		Domain:       "embermind.app", // already seeded
		PlanID:       "plan-starter",
		ContactEmail: "dup@test.com",
		ContactName:  "Dup User",
	}, "admin")

	if err == nil {
		t.Error("Expected error for duplicate domain")
	}
}

func TestCreateCustomer_InvalidPlan(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	_, err := svc.CreateCustomer(ctx, &dto.CreateCustomerInput{
		Name:         "BadPlan",
		Domain:       "badplan.io",
		PlanID:       "nonexistent-plan",
		ContactEmail: "bp@test.com",
		ContactName:  "BP User",
	}, "admin")

	if err == nil {
		t.Error("Expected error for invalid plan ID")
	}
}

// --- UpdateCustomer tests ---

func TestUpdateCustomer_Success(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	newName := "Embermind Updated"
	customer, err := svc.UpdateCustomer(ctx, "cust-001", &dto.UpdateCustomerInput{
		Name: &newName,
	}, "admin")

	if err != nil {
		t.Fatalf("UpdateCustomer failed: %v", err)
	}
	if customer.Name != "Embermind Updated" {
		t.Errorf("Expected updated name, got '%s'", customer.Name)
	}
}

func TestUpdateCustomer_NotFound(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	newName := "Ghost"
	_, err := svc.UpdateCustomer(ctx, "nonexistent", &dto.UpdateCustomerInput{
		Name: &newName,
	}, "admin")

	if err == nil {
		t.Error("Expected error for nonexistent customer update")
	}
}

// --- DeleteCustomer tests ---

func TestDeleteCustomer_Success(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	err := svc.DeleteCustomer(ctx, "cust-003", "admin")
	if err != nil {
		t.Fatalf("DeleteCustomer failed: %v", err)
	}

	// Should no longer exist
	_, err = svc.GetCustomer(ctx, "cust-003")
	if err == nil {
		t.Error("Expected error getting deleted customer")
	}
}

func TestDeleteCustomer_NotFound(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	err := svc.DeleteCustomer(ctx, "nonexistent", "admin")
	if err == nil {
		t.Error("Expected error deleting nonexistent customer")
	}
}

// --- SuspendCustomer tests ---

func TestSuspendCustomer_Success(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	customer, err := svc.SuspendCustomer(ctx, "cust-001", "admin")
	if err != nil {
		t.Fatalf("SuspendCustomer failed: %v", err)
	}
	if customer.Status != "suspended" {
		t.Errorf("Expected status 'suspended', got '%s'", customer.Status)
	}
}

func TestSuspendCustomer_NotFound(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	_, err := svc.SuspendCustomer(ctx, "nonexistent", "admin")
	if err == nil {
		t.Error("Expected error suspending nonexistent customer")
	}
}

// --- ActivateCustomer tests ---

func TestActivateCustomer_Success(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	// First suspend, then activate
	_, _ = svc.SuspendCustomer(ctx, "cust-002", "admin")
	customer, err := svc.ActivateCustomer(ctx, "cust-002", "admin")
	if err != nil {
		t.Fatalf("ActivateCustomer failed: %v", err)
	}
	if customer.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", customer.Status)
	}
}

// --- GetCustomerCluster tests ---

func TestGetCustomerCluster_NoOrchestrator(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	cluster, err := svc.GetCustomerCluster(ctx, "cust-001")
	if err != nil {
		t.Fatalf("GetCustomerCluster failed: %v", err)
	}
	clusterMap, ok := cluster.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	if clusterMap["capiClusterName"] == nil {
		t.Error("Expected capiClusterName in result")
	}
}

func TestGetCustomerCluster_NotFound(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	_, err := svc.GetCustomerCluster(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent customer cluster")
	}
}

// --- ScaleCluster tests ---

func TestScaleCluster_NoOrchestrator(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	err := svc.ScaleCluster(ctx, "cust-001", 5)
	if err == nil {
		t.Error("Expected error when orchestrator is nil")
	}
}

// --- UpgradeCluster tests ---

func TestUpgradeCluster_NoOrchestrator(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	err := svc.UpgradeCluster(ctx, "cust-001", "v1.32.0")
	if err == nil {
		t.Error("Expected error when orchestrator is nil")
	}
}

// --- Plans CRUD via CustomerService ---

func TestListPlans(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	plans, err := svc.ListPlans(ctx)
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	// Memory repo is pre-seeded with 3 plans
	if len(plans) < 3 {
		t.Errorf("Expected at least 3 seeded plans, got %d", len(plans))
	}
}

func TestCreatePlan_Success(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	plan, err := svc.CreatePlan(ctx, &dto.CreatePlanInput{
		Name:       "Custom Plan",
		CPUCores:   8,
		RAMGB:      16,
		PriceCents: 19900,
	}, "admin")

	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	if plan.Name != "Custom Plan" {
		t.Errorf("Expected plan name 'Custom Plan', got '%s'", plan.Name)
	}
}

func TestCreatePlan_DuplicateName(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	_, err := svc.CreatePlan(ctx, &dto.CreatePlanInput{
		Name:       "Starter", // already seeded
		CPUCores:   4,
		RAMGB:      8,
		PriceCents: 9900,
	}, "admin")

	if err == nil {
		t.Error("Expected error for duplicate plan name")
	}
}

func TestUpdatePlan_Success(t *testing.T) {
	svc, _, _ := newTestCustomerService()
	ctx := context.Background()

	newName := "Starter Plus"
	plan, err := svc.UpdatePlan(ctx, "plan-starter", &dto.UpdatePlanInput{
		Name: &newName,
	}, "admin")

	if err != nil {
		t.Fatalf("UpdatePlan failed: %v", err)
	}
	if plan.Name != "Starter Plus" {
		t.Errorf("Expected plan name 'Starter Plus', got '%s'", plan.Name)
	}
}

// --- Error helper tests ---

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(fmt.Errorf("customer not found")) {
		t.Error("Expected IsNotFound to return true")
	}
	if IsNotFound(nil) {
		t.Error("Expected IsNotFound(nil) to return false")
	}
	if IsNotFound(fmt.Errorf("some other error")) {
		t.Error("Expected IsNotFound to return false for unrelated error")
	}
}

func TestIsDomainConflict(t *testing.T) {
	if !IsDomainConflict(fmt.Errorf("domain already in use")) {
		t.Error("Expected IsDomainConflict to return true")
	}
	if IsDomainConflict(nil) {
		t.Error("Expected IsDomainConflict(nil) to return false")
	}
}

func TestIsPlanConflict(t *testing.T) {
	if !IsPlanConflict(fmt.Errorf("plan name already exists")) {
		t.Error("Expected IsPlanConflict to return true")
	}
	if IsPlanConflict(nil) {
		t.Error("Expected IsPlanConflict(nil) to return false")
	}
}
