package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/google/uuid"
)

// Compile-time interface check.
var _ CustomerRepository = (*MemoryCustomerRepository)(nil)

// MemoryCustomerRepository provides an in-memory store for customers and plans.
type MemoryCustomerRepository struct {
	mu        sync.RWMutex
	plans     map[string]*models.Plan
	customers map[string]*models.Customer
}

// NewMemoryCustomerRepository creates a MemoryCustomerRepository pre-seeded with demo data.
func NewMemoryCustomerRepository() *MemoryCustomerRepository {
	now := time.Now()
	r := &MemoryCustomerRepository{
		plans:     make(map[string]*models.Plan),
		customers: make(map[string]*models.Customer),
	}

	// Seed plans
	seedPlans := []models.Plan{
		{ID: "plan-starter", Name: "Starter", CPUCores: 4, RAMGB: 8, S3TB: 0, DBStorageGB: 10, VolumeGB: 50, LBCount: 1, PriceCents: 9900, Currency: "EUR", BillingCycle: "monthly", Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: "plan-pro", Name: "Pro", CPUCores: 16, RAMGB: 32, S3TB: 1, DBStorageGB: 100, VolumeGB: 500, LBCount: 3, PriceCents: 49900, Currency: "EUR", BillingCycle: "monthly", Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: "plan-enterprise", Name: "Enterprise", CPUCores: 64, RAMGB: 128, S3TB: 10, DBStorageGB: 1000, VolumeGB: 5000, LBCount: 10, PriceCents: 199900, Currency: "EUR", BillingCycle: "monthly", Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for i := range seedPlans {
		r.plans[seedPlans[i].ID] = &seedPlans[i]
	}

	// Seed customers
	seedCustomers := []models.Customer{
		{ID: "cust-001", Name: "Embermind", Domain: "embermind.app", PlanID: "plan-pro", ContactEmail: "ops@embermind.app", ContactName: "Sarah Chen", Status: "active", ClusterStatus: "running", Notes: "", CreatedAt: now.Add(-30 * 24 * time.Hour), UpdatedAt: now},
		{ID: "cust-002", Name: "Acme Corp", Domain: "acme-corp.com", PlanID: "plan-pro", ContactEmail: "infra@acme-corp.com", ContactName: "James Wilson", Status: "active", ClusterStatus: "running", Notes: "", CreatedAt: now.Add(-20 * 24 * time.Hour), UpdatedAt: now},
		{ID: "cust-003", Name: "Starship IO", Domain: "starship.io", PlanID: "plan-starter", ContactEmail: "admin@starship.io", ContactName: "Alex Rivera", Status: "active", ClusterStatus: "running", Notes: "", CreatedAt: now.Add(-10 * 24 * time.Hour), UpdatedAt: now},
	}
	for i := range seedCustomers {
		r.customers[seedCustomers[i].ID] = &seedCustomers[i]
	}

	return r
}

func (r *MemoryCustomerRepository) CreatePlan(_ context.Context, input *models.CreatePlanInput) (*models.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, p := range r.plans {
		if p.Name == input.Name {
			return nil, fmt.Errorf("plan name already exists")
		}
	}

	now := time.Now()
	plan := &models.Plan{
		ID:           uuid.New().String(),
		Name:         input.Name,
		CPUCores:     input.CPUCores,
		RAMGB:        input.RAMGB,
		S3TB:         input.S3TB,
		DBStorageGB:  input.DBStorageGB,
		VolumeGB:     input.VolumeGB,
		LBCount:      input.LBCount,
		PriceCents:   input.PriceCents,
		Currency:     input.Currency,
		BillingCycle: input.BillingCycle,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if plan.Currency == "" {
		plan.Currency = "EUR"
	}
	if plan.BillingCycle == "" {
		plan.BillingCycle = "monthly"
	}

	r.plans[plan.ID] = plan
	copied := *plan
	return &copied, nil
}

func (r *MemoryCustomerRepository) GetPlan(_ context.Context, id string) (*models.Plan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plan, ok := r.plans[id]
	if !ok {
		return nil, fmt.Errorf("plan not found")
	}
	copied := *plan
	return &copied, nil
}

func (r *MemoryCustomerRepository) ListPlans(_ context.Context) ([]models.Plan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plans := make([]models.Plan, 0, len(r.plans))
	for _, p := range r.plans {
		plans = append(plans, *p)
	}
	return plans, nil
}

func (r *MemoryCustomerRepository) UpdatePlan(_ context.Context, id string, input *models.UpdatePlanInput) (*models.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	plan, ok := r.plans[id]
	if !ok {
		return nil, fmt.Errorf("plan not found")
	}

	if input.Name != nil {
		// Check uniqueness
		for _, p := range r.plans {
			if p.ID != id && p.Name == *input.Name {
				return nil, fmt.Errorf("plan name already exists")
			}
		}
		plan.Name = *input.Name
	}
	if input.CPUCores != nil {
		plan.CPUCores = *input.CPUCores
	}
	if input.RAMGB != nil {
		plan.RAMGB = *input.RAMGB
	}
	if input.S3TB != nil {
		plan.S3TB = *input.S3TB
	}
	if input.DBStorageGB != nil {
		plan.DBStorageGB = *input.DBStorageGB
	}
	if input.VolumeGB != nil {
		plan.VolumeGB = *input.VolumeGB
	}
	if input.LBCount != nil {
		plan.LBCount = *input.LBCount
	}
	if input.PriceCents != nil {
		plan.PriceCents = *input.PriceCents
	}
	if input.Currency != nil {
		plan.Currency = *input.Currency
	}
	if input.BillingCycle != nil {
		plan.BillingCycle = *input.BillingCycle
	}
	if input.Active != nil {
		plan.Active = *input.Active
	}
	plan.UpdatedAt = time.Now()

	copied := *plan
	return &copied, nil
}

func (r *MemoryCustomerRepository) CreateCustomer(_ context.Context, input *models.CreateCustomerInput) (*models.Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Verify plan exists
	if _, ok := r.plans[input.PlanID]; !ok {
		return nil, fmt.Errorf("plan not found")
	}

	// Check domain uniqueness
	for _, c := range r.customers {
		if c.Domain == input.Domain {
			return nil, fmt.Errorf("domain already in use")
		}
	}

	now := time.Now()
	customer := &models.Customer{
		ID:            uuid.New().String(),
		Name:          input.Name,
		Domain:        input.Domain,
		PlanID:        input.PlanID,
		ContactEmail:  input.ContactEmail,
		ContactName:   input.ContactName,
		Status:        "active",
		ClusterStatus: "pending",
		Notes:         "",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	r.customers[customer.ID] = customer
	copied := *customer
	return &copied, nil
}

func (r *MemoryCustomerRepository) GetCustomer(_ context.Context, id string) (*models.Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	customer, ok := r.customers[id]
	if !ok {
		return nil, fmt.Errorf("customer not found")
	}
	copied := *customer
	if plan, ok := r.plans[customer.PlanID]; ok {
		planCopy := *plan
		copied.Plan = &planCopy
	}
	return &copied, nil
}

func (r *MemoryCustomerRepository) ListCustomers(_ context.Context) ([]models.Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	customers := make([]models.Customer, 0, len(r.customers))
	for _, c := range r.customers {
		copied := *c
		if plan, ok := r.plans[c.PlanID]; ok {
			planCopy := *plan
			copied.Plan = &planCopy
		}
		customers = append(customers, copied)
	}
	return customers, nil
}

func (r *MemoryCustomerRepository) UpdateCustomer(_ context.Context, id string, input *models.UpdateCustomerInput) (*models.Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	customer, ok := r.customers[id]
	if !ok {
		return nil, fmt.Errorf("customer not found")
	}

	if input.Name != nil {
		customer.Name = *input.Name
	}
	if input.Domain != nil {
		// Check uniqueness
		for _, c := range r.customers {
			if c.ID != id && c.Domain == *input.Domain {
				return nil, fmt.Errorf("domain already in use")
			}
		}
		customer.Domain = *input.Domain
	}
	if input.PlanID != nil {
		if _, ok := r.plans[*input.PlanID]; !ok {
			return nil, fmt.Errorf("plan not found")
		}
		customer.PlanID = *input.PlanID
	}
	if input.ContactEmail != nil {
		customer.ContactEmail = *input.ContactEmail
	}
	if input.ContactName != nil {
		customer.ContactName = *input.ContactName
	}
	if input.Notes != nil {
		customer.Notes = *input.Notes
	}
	customer.UpdatedAt = time.Now()

	copied := *customer
	if plan, ok := r.plans[customer.PlanID]; ok {
		planCopy := *plan
		copied.Plan = &planCopy
	}
	return &copied, nil
}

func (r *MemoryCustomerRepository) DeleteCustomer(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.customers[id]; !ok {
		return fmt.Errorf("customer not found")
	}
	delete(r.customers, id)
	return nil
}

func (r *MemoryCustomerRepository) SuspendCustomer(_ context.Context, id string) (*models.Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	customer, ok := r.customers[id]
	if !ok {
		return nil, fmt.Errorf("customer not found")
	}
	customer.Status = "suspended"
	customer.UpdatedAt = time.Now()

	copied := *customer
	if plan, ok := r.plans[customer.PlanID]; ok {
		planCopy := *plan
		copied.Plan = &planCopy
	}
	return &copied, nil
}

func (r *MemoryCustomerRepository) ActivateCustomer(_ context.Context, id string) (*models.Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	customer, ok := r.customers[id]
	if !ok {
		return nil, fmt.Errorf("customer not found")
	}
	customer.Status = "active"
	customer.UpdatedAt = time.Now()

	copied := *customer
	if plan, ok := r.plans[customer.PlanID]; ok {
		planCopy := *plan
		copied.Plan = &planCopy
	}
	return &copied, nil
}

func (r *MemoryCustomerRepository) GetCustomerStats(_ context.Context) (*models.CustomerStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := len(r.customers)
	active := 0
	mrrCents := 0
	newThisMonth := 0
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	for _, c := range r.customers {
		if c.Status == "active" {
			active++
			if plan, ok := r.plans[c.PlanID]; ok {
				mrrCents += plan.PriceCents
			}
		}
		if c.CreatedAt.After(monthStart) {
			newThisMonth++
		}
	}

	return &models.CustomerStats{
		TotalCustomers:  total,
		ActiveCustomers: active,
		MRR:             fmt.Sprintf("€%d", mrrCents/100),
		NewThisMonth:    newThisMonth,
	}, nil
}
