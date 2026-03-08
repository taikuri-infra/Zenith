package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// tenantNamespace derives the Kubernetes namespace for a customer domain.
func tenantNamespace(domain string) string {
	return "zenith-" + strings.ReplaceAll(strings.ReplaceAll(domain, ".", "-"), "_", "-")
}

// CustomerService handles customer management business logic.
type CustomerService struct {
	store        ports.CustomerRepository
	admin        ports.AdminRepository
	orchestrator ports.ClusterOrchestrator     // CAPI provisioner (fallback)
	workflows    ports.ProvisioningWorkflow    // Temporal workflows (preferred)
}

// NewCustomerService creates a new CustomerService.
func NewCustomerService(store ports.CustomerRepository, admin ports.AdminRepository, orchestrator ports.ClusterOrchestrator) *CustomerService {
	return &CustomerService{store: store, admin: admin, orchestrator: orchestrator}
}

// SetWorkflows enables Temporal-based provisioning workflows.
func (s *CustomerService) SetWorkflows(wf ports.ProvisioningWorkflow) {
	s.workflows = wf
}

// ListCustomers returns all customers.
func (s *CustomerService) ListCustomers(ctx context.Context) ([]entities.Customer, error) {
	return s.store.ListCustomers(ctx)
}

// GetCustomerStats returns aggregate customer statistics.
func (s *CustomerService) GetCustomerStats(ctx context.Context) (*entities.CustomerStats, error) {
	return s.store.GetCustomerStats(ctx)
}

// GetCustomer returns a single customer by ID.
func (s *CustomerService) GetCustomer(ctx context.Context, id string) (*entities.Customer, error) {
	return s.store.GetCustomer(ctx, id)
}

// CreateCustomer creates a new customer and triggers cluster provisioning.
func (s *CustomerService) CreateCustomer(ctx context.Context, input *dto.CreateCustomerInput, actor string) (*entities.Customer, error) {
	customer, err := s.store.CreateCustomer(ctx, input)
	if err != nil {
		return nil, err
	}

	_ = s.admin.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Created customer " + input.Name + " (" + input.Domain + ")",
	})

	// Trigger provisioning: prefer Temporal workflow, fallback to CAPI goroutine
	if s.workflows != nil {
		planTier := "free"
		if customer.Plan != nil && customer.Plan.PriceCents > 10000 {
			planTier = "pro"
		}
		err := s.workflows.StartProvision(ctx, ports.ProvisionInput{
			CustomerID:   customer.ID,
			CustomerName: customer.Name,
			Domain:       customer.Domain,
			PlanTier:     planTier,
			ContactEmail: customer.ContactEmail,
		})
		if err != nil {
			slog.Error("failed to start provision workflow", "customer_name", customer.Name, "error", err)
		}
	} else if s.orchestrator != nil {
		go func() {
			// Use background context — the request context is invalidated once the handler returns.
			_ = s.orchestrator.ProvisionCluster(context.Background(), customer)
		}()
	}

	return customer, nil
}

// UpdateCustomer updates an existing customer.
func (s *CustomerService) UpdateCustomer(ctx context.Context, id string, input *dto.UpdateCustomerInput, actor string) (*entities.Customer, error) {
	customer, err := s.store.UpdateCustomer(ctx, id, input)
	if err != nil {
		return nil, err
	}

	_ = s.admin.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Updated customer " + customer.Name,
	})

	return customer, nil
}

// DeleteCustomer deletes a customer and tears down its cluster.
func (s *CustomerService) DeleteCustomer(ctx context.Context, id, actor string) error {
	customer, _ := s.store.GetCustomer(ctx, id)
	customerName := id
	if customer != nil {
		customerName = customer.Name

		// Trigger deprovisioning: prefer Temporal, fallback to CAPI
		if s.workflows != nil {
			ns := tenantNamespace(customer.Domain)
			err := s.workflows.StartDeprovision(ctx, ports.DeprovisionInput{
				CustomerID:   customer.ID,
				CustomerName: customer.Name,
				Domain:       customer.Domain,
				Namespace:    ns,
			})
			if err != nil {
				slog.Error("failed to start deprovision workflow", "customer_name", customer.Name, "error", err)
			}
		} else if s.orchestrator != nil {
			_ = s.orchestrator.TeardownCluster(ctx, customer)
		}
	}

	if err := s.store.DeleteCustomer(ctx, id); err != nil {
		return err
	}

	_ = s.admin.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Deleted customer " + customerName,
	})

	return nil
}

// SuspendCustomer suspends a customer.
func (s *CustomerService) SuspendCustomer(ctx context.Context, id, actor string) (*entities.Customer, error) {
	customer, err := s.store.SuspendCustomer(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = s.admin.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Suspended customer " + customer.Name,
	})

	return customer, nil
}

// ActivateCustomer activates a suspended customer.
func (s *CustomerService) ActivateCustomer(ctx context.Context, id, actor string) (*entities.Customer, error) {
	customer, err := s.store.ActivateCustomer(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = s.admin.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Activated customer " + customer.Name,
	})

	return customer, nil
}

// GetCustomerCluster returns cluster info for a customer.
func (s *CustomerService) GetCustomerCluster(ctx context.Context, id string) (interface{}, error) {
	customer, err := s.store.GetCustomer(ctx, id)
	if err != nil {
		return nil, err
	}

	if customer.CAPIClusterName == "" {
		return map[string]interface{}{
			"clusterStatus": customer.ClusterStatus,
			"message":       "no cluster provisioned",
		}, nil
	}

	if s.orchestrator == nil {
		return map[string]interface{}{
			"clusterStatus":   customer.ClusterStatus,
			"capiClusterName": customer.CAPIClusterName,
			"clusterRegion":   customer.ClusterRegion,
			"clusterNodes":    customer.ClusterNodes,
			"k8sVersion":      customer.ClusterK8sVersion,
		}, nil
	}

	cl, err := s.orchestrator.GetCluster(ctx, customer.CAPIClusterName)
	if err != nil {
		return map[string]interface{}{
			"clusterStatus":   customer.ClusterStatus,
			"capiClusterName": customer.CAPIClusterName,
			"error":           err.Error(),
		}, nil
	}

	return cl, nil
}

// ScaleCluster scales the customer's cluster.
func (s *CustomerService) ScaleCluster(ctx context.Context, id string, nodes int) error {
	customer, err := s.store.GetCustomer(ctx, id)
	if err != nil {
		return err
	}

	if s.orchestrator == nil {
		return fmt.Errorf("cluster provisioner not available")
	}

	return s.orchestrator.ScaleCluster(ctx, customer, nodes)
}

// UpgradeCluster upgrades the customer's cluster K8s version.
func (s *CustomerService) UpgradeCluster(ctx context.Context, id, version string) error {
	customer, err := s.store.GetCustomer(ctx, id)
	if err != nil {
		return err
	}

	if s.orchestrator == nil {
		return fmt.Errorf("cluster provisioner not available")
	}

	return s.orchestrator.UpgradeCluster(ctx, customer, version)
}

// Plans

// ListPlans returns all plans.
func (s *CustomerService) ListPlans(ctx context.Context) ([]entities.Plan, error) {
	return s.store.ListPlans(ctx)
}

// CreatePlan creates a new plan.
func (s *CustomerService) CreatePlan(ctx context.Context, input *dto.CreatePlanInput, actor string) (*entities.Plan, error) {
	plan, err := s.store.CreatePlan(ctx, input)
	if err != nil {
		return nil, err
	}

	_ = s.admin.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Created plan " + input.Name,
	})

	return plan, nil
}

// UpdatePlan updates an existing plan.
func (s *CustomerService) UpdatePlan(ctx context.Context, id string, input *dto.UpdatePlanInput, actor string) (*entities.Plan, error) {
	plan, err := s.store.UpdatePlan(ctx, id, input)
	if err != nil {
		return nil, err
	}

	_ = s.admin.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Updated plan " + plan.Name,
	})

	return plan, nil
}

// IsNotFound checks if an error is a not-found error.
func IsNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

// IsDomainConflict checks if an error is a domain conflict.
func IsDomainConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "domain already in use")
}

// IsPlanConflict checks if an error is a plan name conflict.
func IsPlanConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
}
