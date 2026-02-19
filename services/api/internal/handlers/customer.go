package handlers

import (
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// CustomerHandler serves all /api/v1/admin/customers/* and /api/v1/admin/plans/* endpoints.
type CustomerHandler struct {
	store store.CustomerRepository
	admin store.AdminRepository
}

// NewCustomerHandler creates a new CustomerHandler.
func NewCustomerHandler(customerStore store.CustomerRepository, adminStore store.AdminRepository) *CustomerHandler {
	return &CustomerHandler{
		store: customerStore,
		admin: adminStore,
	}
}

// ---------- Customers ----------

// ListCustomers returns all customers.
// GET /api/v1/admin/customers
func (h *CustomerHandler) ListCustomers(c *fiber.Ctx) error {
	customers, err := h.store.ListCustomers(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list customers")
	}
	return c.JSON(customers)
}

// GetCustomerStats returns aggregate customer statistics.
// GET /api/v1/admin/customers/stats
func (h *CustomerHandler) GetCustomerStats(c *fiber.Ctx) error {
	stats, err := h.store.GetCustomerStats(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get customer stats")
	}
	return c.JSON(stats)
}

// GetCustomer returns a single customer by ID.
// GET /api/v1/admin/customers/:id
func (h *CustomerHandler) GetCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	customer, err := h.store.GetCustomer(c.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get customer")
	}
	return c.JSON(customer)
}

// CreateCustomer creates a new customer.
// POST /api/v1/admin/customers
func (h *CustomerHandler) CreateCustomer(c *fiber.Ctx) error {
	var input models.CreateCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	if input.Name == "" {
		return NewBadRequest("name is required")
	}
	if input.Domain == "" {
		return NewBadRequest("domain is required")
	}
	if input.PlanID == "" {
		return NewBadRequest("planId is required")
	}
	if input.ContactEmail == "" {
		return NewBadRequest("contactEmail is required")
	}

	customer, err := h.store.CreateCustomer(c.Context(), &input)
	if err != nil {
		if strings.Contains(err.Error(), "domain already in use") {
			return NewConflict("domain already in use")
		}
		if strings.Contains(err.Error(), "plan not found") {
			return NewBadRequest("plan not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create customer")
	}

	_ = h.admin.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Created customer " + input.Name + " (" + input.Domain + ")",
	})

	return c.Status(fiber.StatusCreated).JSON(customer)
}

// UpdateCustomer updates an existing customer.
// PUT /api/v1/admin/customers/:id
func (h *CustomerHandler) UpdateCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	var input models.UpdateCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	customer, err := h.store.UpdateCustomer(c.Context(), id, &input)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		if strings.Contains(err.Error(), "domain already in use") {
			return NewConflict("domain already in use")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update customer")
	}

	_ = h.admin.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Updated customer " + customer.Name,
	})

	return c.JSON(customer)
}

// DeleteCustomer deletes a customer.
// DELETE /api/v1/admin/customers/:id
func (h *CustomerHandler) DeleteCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	// Get customer name for audit log before deleting
	customer, _ := h.store.GetCustomer(c.Context(), id)
	customerName := id
	if customer != nil {
		customerName = customer.Name
	}

	if err := h.store.DeleteCustomer(c.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete customer")
	}

	_ = h.admin.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Deleted customer " + customerName,
	})

	return c.JSON(fiber.Map{"message": "customer deleted"})
}

// SuspendCustomer suspends a customer.
// POST /api/v1/admin/customers/:id/suspend
func (h *CustomerHandler) SuspendCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	customer, err := h.store.SuspendCustomer(c.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to suspend customer")
	}

	_ = h.admin.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Suspended customer " + customer.Name,
	})

	return c.JSON(customer)
}

// ActivateCustomer activates a suspended customer.
// POST /api/v1/admin/customers/:id/activate
func (h *CustomerHandler) ActivateCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	customer, err := h.store.ActivateCustomer(c.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to activate customer")
	}

	_ = h.admin.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Activated customer " + customer.Name,
	})

	return c.JSON(customer)
}

// ---------- Plans ----------

// ListPlans returns all plans.
// GET /api/v1/admin/plans
func (h *CustomerHandler) ListPlans(c *fiber.Ctx) error {
	plans, err := h.store.ListPlans(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list plans")
	}
	return c.JSON(plans)
}

// CreatePlan creates a new plan.
// POST /api/v1/admin/plans
func (h *CustomerHandler) CreatePlan(c *fiber.Ctx) error {
	var input models.CreatePlanInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	if input.Name == "" {
		return NewBadRequest("name is required")
	}
	if input.CPUCores <= 0 {
		return NewBadRequest("cpuCores must be positive")
	}
	if input.RAMGB <= 0 {
		return NewBadRequest("ramGb must be positive")
	}
	if input.PriceCents <= 0 {
		return NewBadRequest("priceCents must be positive")
	}

	plan, err := h.store.CreatePlan(c.Context(), &input)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return NewConflict("plan name already exists")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create plan")
	}

	_ = h.admin.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Created plan " + input.Name,
	})

	return c.Status(fiber.StatusCreated).JSON(plan)
}

// UpdatePlan updates an existing plan.
// PUT /api/v1/admin/plans/:id
func (h *CustomerHandler) UpdatePlan(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("plan id is required")
	}

	var input models.UpdatePlanInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	plan, err := h.store.UpdatePlan(c.Context(), id, &input)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("plan")
		}
		if strings.Contains(err.Error(), "already exists") {
			return NewConflict("plan name already exists")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update plan")
	}

	_ = h.admin.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Updated plan " + plan.Name,
	})

	return c.JSON(plan)
}
