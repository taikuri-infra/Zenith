package handlers

import (
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/cluster"
	"github.com/dotechhq/zenith/services/api/internal/dto"
"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// CustomerHandler serves all /api/v1/admin/customers/* and /api/v1/admin/plans/* endpoints.
type CustomerHandler struct {
	store       store.CustomerRepository
	admin       store.AdminRepository
	provisioner *cluster.Provisioner
}

// NewCustomerHandler creates a new CustomerHandler.
func NewCustomerHandler(customerStore store.CustomerRepository, adminStore store.AdminRepository, provisioner *cluster.Provisioner) *CustomerHandler {
	return &CustomerHandler{
		store:       customerStore,
		admin:       adminStore,
		provisioner: provisioner,
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

	_ = h.admin.AddAuditEntry(c.Context(), entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Created customer " + input.Name + " (" + input.Domain + ")",
	})

	// Trigger cluster provisioning in background
	if h.provisioner != nil {
		go func() {
			if err := h.provisioner.ProvisionCluster(c.Context(), customer); err != nil {
				// Logged internally; status set to error in DB
				_ = err
			}
		}()
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

	_ = h.admin.AddAuditEntry(c.Context(), entities.AuditEntry{
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

	// Get customer for audit log and cluster teardown before deleting
	customer, _ := h.store.GetCustomer(c.Context(), id)
	customerName := id
	if customer != nil {
		customerName = customer.Name
		// Teardown cluster before deleting customer
		if h.provisioner != nil {
			_ = h.provisioner.TeardownCluster(c.Context(), customer)
		}
	}

	if err := h.store.DeleteCustomer(c.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete customer")
	}

	_ = h.admin.AddAuditEntry(c.Context(), entities.AuditEntry{
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

	_ = h.admin.AddAuditEntry(c.Context(), entities.AuditEntry{
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

	_ = h.admin.AddAuditEntry(c.Context(), entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Activated customer " + customer.Name,
	})

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

	customer, err := h.store.GetCustomer(c.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get customer")
	}

	if customer.CAPIClusterName == "" {
		return c.JSON(fiber.Map{
			"clusterStatus": customer.ClusterStatus,
			"message":       "no cluster provisioned",
		})
	}

	if h.provisioner == nil {
		return c.JSON(fiber.Map{
			"clusterStatus":   customer.ClusterStatus,
			"capiClusterName": customer.CAPIClusterName,
			"clusterRegion":   customer.ClusterRegion,
			"clusterNodes":    customer.ClusterNodes,
			"k8sVersion":      customer.ClusterK8sVersion,
		})
	}

	cluster, err := h.provisioner.GetCluster(c.Context(), customer.CAPIClusterName)
	if err != nil {
		return c.JSON(fiber.Map{
			"clusterStatus":   customer.ClusterStatus,
			"capiClusterName": customer.CAPIClusterName,
			"error":           err.Error(),
		})
	}

	return c.JSON(cluster)
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

	customer, err := h.store.GetCustomer(c.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get customer")
	}

	if h.provisioner == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "cluster provisioner not available")
	}

	if err := h.provisioner.ScaleCluster(c.Context(), customer, input.Nodes); err != nil {
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

	customer, err := h.store.GetCustomer(c.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get customer")
	}

	if h.provisioner == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "cluster provisioner not available")
	}

	if err := h.provisioner.UpgradeCluster(c.Context(), customer, input.Version); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to upgrade cluster: "+err.Error())
	}

	return c.JSON(fiber.Map{"message": "cluster upgrade started", "version": input.Version})
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

	plan, err := h.store.CreatePlan(c.Context(), &input)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return NewConflict("plan name already exists")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create plan")
	}

	_ = h.admin.AddAuditEntry(c.Context(), entities.AuditEntry{
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

	var input dto.UpdatePlanInput
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

	_ = h.admin.AddAuditEntry(c.Context(), entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Updated plan " + plan.Name,
	})

	return c.JSON(plan)
}
