package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// CustomerHandler serves all /api/v1/admin/customers/* and /api/v1/admin/plans/* endpoints.
type CustomerHandler struct {
	svc *services.CustomerService
}

// NewCustomerHandler creates a new CustomerHandler.
func NewCustomerHandler(svc *services.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

// ---------- Customers ----------

// ListCustomers returns all customers.
// GET /api/v1/admin/customers
func (h *CustomerHandler) ListCustomers(c *fiber.Ctx) error {
	customers, err := h.svc.ListCustomers(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list customers")
	}
	return c.JSON(customers)
}

// GetCustomerStats returns aggregate customer statistics.
// GET /api/v1/admin/customers/stats
func (h *CustomerHandler) GetCustomerStats(c *fiber.Ctx) error {
	stats, err := h.svc.GetCustomerStats(c.Context())
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

	customer, err := h.svc.GetCustomer(c.Context(), id)
	if err != nil {
		if services.IsNotFound(err) {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get customer")
	}
	return c.JSON(customer)
}

// CreateCustomer creates a new customer.
// POST /api/v1/admin/customers
func (h *CustomerHandler) CreateCustomer(c *fiber.Ctx) error {
	var input dto.CreateCustomerInput
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

	customer, err := h.svc.CreateCustomer(c.Context(), &input, actorFromContext(c))
	if err != nil {
		if services.IsDomainConflict(err) {
			return NewConflict("domain already in use")
		}
		if services.IsNotFound(err) {
			return NewBadRequest("plan not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create customer")
	}

	return c.Status(fiber.StatusCreated).JSON(customer)
}

// UpdateCustomer updates an existing customer.
// PUT /api/v1/admin/customers/:id
func (h *CustomerHandler) UpdateCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	var input dto.UpdateCustomerInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	customer, err := h.svc.UpdateCustomer(c.Context(), id, &input, actorFromContext(c))
	if err != nil {
		if services.IsNotFound(err) {
			return NewNotFound("customer")
		}
		if services.IsDomainConflict(err) {
			return NewConflict("domain already in use")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update customer")
	}

	return c.JSON(customer)
}

// DeleteCustomer deletes a customer.
// DELETE /api/v1/admin/customers/:id
func (h *CustomerHandler) DeleteCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	if err := h.svc.DeleteCustomer(c.Context(), id, actorFromContext(c)); err != nil {
		if services.IsNotFound(err) {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete customer")
	}

	return c.JSON(fiber.Map{"message": "customer deleted"})
}

// SuspendCustomer suspends a customer.
// POST /api/v1/admin/customers/:id/suspend
func (h *CustomerHandler) SuspendCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	customer, err := h.svc.SuspendCustomer(c.Context(), id, actorFromContext(c))
	if err != nil {
		if services.IsNotFound(err) {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to suspend customer")
	}

	return c.JSON(customer)
}

// ActivateCustomer activates a suspended customer.
// POST /api/v1/admin/customers/:id/activate
func (h *CustomerHandler) ActivateCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	customer, err := h.svc.ActivateCustomer(c.Context(), id, actorFromContext(c))
	if err != nil {
		if services.IsNotFound(err) {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to activate customer")
	}

	return c.JSON(customer)
}

// ---------- Cluster ----------

// GetCustomerCluster returns the CAPI cluster info for a customer.
// GET /api/v1/admin/customers/:id/cluster
func (h *CustomerHandler) GetCustomerCluster(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	result, err := h.svc.GetCustomerCluster(c.Context(), id)
	if err != nil {
		if services.IsNotFound(err) {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get customer")
	}

	return c.JSON(result)
}

// ScaleCluster scales the customer's cluster.
// POST /api/v1/admin/customers/:id/cluster/scale
func (h *CustomerHandler) ScaleCluster(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	var input dto.ScaleClusterInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	if input.Nodes < 1 {
		return NewBadRequest("nodes must be at least 1")
	}

	if err := h.svc.ScaleCluster(c.Context(), id, input.Nodes); err != nil {
		if services.IsNotFound(err) {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to scale cluster: "+err.Error())
	}

	return c.JSON(fiber.Map{"message": "cluster scaled", "nodes": input.Nodes})
}

// UpgradeCluster upgrades the customer's cluster K8s version.
// POST /api/v1/admin/customers/:id/cluster/upgrade
func (h *CustomerHandler) UpgradeCluster(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	var input dto.UpgradeClusterInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	if input.Version == "" {
		return NewBadRequest("version is required")
	}

	if err := h.svc.UpgradeCluster(c.Context(), id, input.Version); err != nil {
		if services.IsNotFound(err) {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to upgrade cluster: "+err.Error())
	}

	return c.JSON(fiber.Map{"message": "cluster upgrade started", "version": input.Version})
}

// ---------- Plans ----------

// ListPlans returns all plans.
// GET /api/v1/admin/plans
func (h *CustomerHandler) ListPlans(c *fiber.Ctx) error {
	plans, err := h.svc.ListPlans(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list plans")
	}
	return c.JSON(plans)
}

// CreatePlan creates a new plan.
// POST /api/v1/admin/plans
func (h *CustomerHandler) CreatePlan(c *fiber.Ctx) error {
	var input dto.CreatePlanInput
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

	plan, err := h.svc.CreatePlan(c.Context(), &input, actorFromContext(c))
	if err != nil {
		if services.IsPlanConflict(err) {
			return NewConflict("plan name already exists")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create plan")
	}

	return c.Status(fiber.StatusCreated).JSON(plan)
}

// UpdatePlan updates an existing plan.
// PUT /api/v1/admin/plans/:id
func (h *CustomerHandler) UpdatePlan(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("plan id is required")
	}

	var input dto.UpdatePlanInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	plan, err := h.svc.UpdatePlan(c.Context(), id, &input, actorFromContext(c))
	if err != nil {
		if services.IsNotFound(err) {
			return NewNotFound("plan")
		}
		if services.IsPlanConflict(err) {
			return NewConflict("plan name already exists")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update plan")
	}

	return c.JSON(plan)
}
