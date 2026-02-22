package handlers

import (
	"strconv"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// AdminHandler serves all /api/v1/admin/* endpoints for Mission Control.
type AdminHandler struct {
	svc *services.AdminService
}

// NewAdminHandler creates a new AdminHandler with the given dependencies.
func NewAdminHandler(svc *services.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ---------- Dashboard ----------

// GetDashboardStats returns aggregate statistics for the Mission Control dashboard.
// GET /api/v1/admin/dashboard/stats
func (h *AdminHandler) GetDashboardStats(c *fiber.Ctx) error {
	stats, err := h.svc.GetDashboardStats(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get dashboard stats")
	}
	return c.JSON(stats)
}

// ---------- Clusters ----------

// ListClusters returns all CAPI-managed clusters.
// GET /api/v1/admin/clusters
func (h *AdminHandler) ListClusters(c *fiber.Ctx) error {
	clusters, err := h.svc.ListClusters(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list clusters")
	}
	return c.JSON(clusters)
}

// GetCluster returns a single cluster by name.
// GET /api/v1/admin/clusters/:name
func (h *AdminHandler) GetCluster(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("cluster name is required")
	}

	cluster, err := h.svc.GetCluster(c.Context(), name)
	if err != nil {
		return NewNotFound("cluster")
	}
	return c.JSON(cluster)
}

// CreateCluster provisions a new CAPI cluster.
// POST /api/v1/admin/clusters
func (h *AdminHandler) CreateCluster(c *fiber.Ctx) error {
	var input dto.CreateClusterInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	if input.Name == "" {
		return NewBadRequest("name is required")
	}
	if input.Region == "" {
		return NewBadRequest("region is required")
	}
	if input.K8sVersion == "" {
		return NewBadRequest("k8sVersion is required")
	}
	if input.Nodes <= 0 {
		input.Nodes = 1
	}
	if input.Type == "" {
		input.Type = "shared"
	}

	validTypes := map[string]bool{"shared": true, "dedicated": true}
	if !validTypes[input.Type] {
		return NewBadRequest("type must be 'shared' or 'dedicated'")
	}

	cluster, err := h.svc.CreateCluster(c.Context(), input, actorFromContext(c))
	if err != nil {
		return NewConflict("cluster already exists")
	}

	return c.Status(fiber.StatusCreated).JSON(cluster)
}

// DeleteCluster removes a CAPI cluster.
// DELETE /api/v1/admin/clusters/:name
func (h *AdminHandler) DeleteCluster(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("cluster name is required")
	}

	if err := h.svc.DeleteCluster(c.Context(), name, actorFromContext(c)); err != nil {
		return NewNotFound("cluster")
	}

	return c.JSON(fiber.Map{"message": "cluster scheduled for deletion"})
}

// UpgradeCluster initiates a Kubernetes version upgrade.
// POST /api/v1/admin/clusters/:name/upgrade
func (h *AdminHandler) UpgradeCluster(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("cluster name is required")
	}

	var input dto.UpgradeClusterInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}
	if input.Version == "" {
		return NewBadRequest("version is required")
	}

	if err := h.svc.UpgradeCluster(c.Context(), name, input.Version, actorFromContext(c)); err != nil {
		return NewNotFound("cluster")
	}

	return c.JSON(fiber.Map{"message": "cluster upgrade initiated"})
}

// ---------- Tenants ----------

// ListTenants returns all tenants derived from Project CRDs.
// GET /api/v1/admin/tenants
func (h *AdminHandler) ListTenants(c *fiber.Ctx) error {
	tenants, err := h.svc.ListTenants(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list tenants")
	}
	return c.JSON(tenants)
}

// GetTenant returns a single tenant by project name.
// GET /api/v1/admin/tenants/:id
func (h *AdminHandler) GetTenant(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("tenant id is required")
	}

	tenant, err := h.svc.GetTenant(c.Context(), id)
	if err != nil {
		return NewNotFound("tenant")
	}

	return c.JSON(tenant)
}

// SuspendTenant suspends a tenant by adding an annotation.
// POST /api/v1/admin/tenants/:id/suspend
func (h *AdminHandler) SuspendTenant(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("tenant id is required")
	}

	if err := h.svc.SuspendTenant(c.Context(), id, actorFromContext(c)); err != nil {
		if services.IsNotFound(err) {
			return NewNotFound("tenant")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to suspend tenant")
	}

	return c.JSON(fiber.Map{"message": "tenant suspended"})
}

// ---------- Modules ----------

// ListModules returns all platform modules.
// GET /api/v1/admin/modules
func (h *AdminHandler) ListModules(c *fiber.Ctx) error {
	modules, err := h.svc.ListModules(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list modules")
	}
	return c.JSON(modules)
}

// InstallModule triggers installation of a module.
// POST /api/v1/admin/modules/:name/install
func (h *AdminHandler) InstallModule(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("module name is required")
	}

	if err := h.svc.InstallModule(c.Context(), name, actorFromContext(c)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "module installation initiated"})
}

// UninstallModule triggers uninstallation of a module.
// POST /api/v1/admin/modules/:name/uninstall
func (h *AdminHandler) UninstallModule(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("module name is required")
	}

	if err := h.svc.UninstallModule(c.Context(), name, actorFromContext(c)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "module uninstallation initiated"})
}

// UpdateModule updates a module to its latest version.
// POST /api/v1/admin/modules/:name/update
func (h *AdminHandler) UpdateModule(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("module name is required")
	}

	mod, err := h.svc.UpdateModule(c.Context(), name, actorFromContext(c))
	if err != nil {
		return NewNotFound("module")
	}

	return c.JSON(mod)
}

// UpdateAllModules updates all modules that have updates available.
// POST /api/v1/admin/modules/update-all
func (h *AdminHandler) UpdateAllModules(c *fiber.Ctx) error {
	updated, err := h.svc.UpdateAllModules(c.Context(), actorFromContext(c))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list modules")
	}

	return c.JSON(fiber.Map{"message": "all modules updated", "count": updated})
}

// ---------- Audit Log ----------

// ListAuditLog returns audit-log entries with optional filtering.
// GET /api/v1/admin/audit
func (h *AdminHandler) ListAuditLog(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	entries, err := h.svc.ListAuditLog(c.Context(), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list audit log")
	}
	return c.JSON(entries)
}

// ---------- Platform Updates ----------

// CheckUpdates returns available platform updates.
// GET /api/v1/admin/updates/check
func (h *AdminHandler) CheckUpdates(c *fiber.Ctx) error {
	update, err := h.svc.CheckUpdates(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to check updates")
	}
	return c.JSON(update)
}

// ApplyUpdate applies a platform update.
// POST /api/v1/admin/updates/apply
func (h *AdminHandler) ApplyUpdate(c *fiber.Ctx) error {
	var input dto.ApplyUpdateInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}
	if input.Version == "" {
		return NewBadRequest("version is required")
	}

	if err := h.svc.ApplyUpdate(c.Context(), input.Version, actorFromContext(c)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "platform update initiated", "version": input.Version})
}

// ListUpdateHistory returns the history of past platform updates.
// GET /api/v1/admin/updates/history
func (h *AdminHandler) ListUpdateHistory(c *fiber.Ctx) error {
	history, err := h.svc.ListUpdateHistory(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list update history")
	}
	return c.JSON(history)
}

// ---------- Infrastructure ----------

// GetInfraOverview returns a summary of Hetzner infrastructure.
// GET /api/v1/admin/infrastructure
func (h *AdminHandler) GetInfraOverview(c *fiber.Ctx) error {
	overview, err := h.svc.GetInfraOverview(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get infrastructure overview")
	}
	return c.JSON(overview)
}

// ---------- Platform State ----------

// GetPlatformState returns the current platform state summary.
// GET /api/v1/admin/state
func (h *AdminHandler) GetPlatformState(c *fiber.Ctx) error {
	state, err := h.svc.GetPlatformState(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get settings")
	}
	return c.JSON(state)
}

// ExportState exports the full platform state as JSON.
// GET /api/v1/admin/state/export
func (h *AdminHandler) ExportState(c *fiber.Ctx) error {
	data, err := h.svc.ExportState(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to export state")
	}

	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename=zenith-state.json")
	return c.Send(data)
}

// ---------- Settings ----------

// GetSettings returns the current platform settings.
// GET /api/v1/admin/settings
func (h *AdminHandler) GetSettings(c *fiber.Ctx) error {
	settings, err := h.svc.GetSettings(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get settings")
	}
	return c.JSON(settings)
}

// UpdateSettings updates platform settings.
// PATCH /api/v1/admin/settings (also supports PUT)
func (h *AdminHandler) UpdateSettings(c *fiber.Ctx) error {
	var input entities.PlatformSettings
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	updated, err := h.svc.UpdateSettings(c.Context(), &input, actorFromContext(c))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update settings")
	}

	return c.JSON(updated)
}

// ---------- Helpers ----------

// actorFromContext extracts the actor name from the Fiber context locals.
func actorFromContext(c *fiber.Ctx) string {
	if email, ok := c.Locals("email").(string); ok && email != "" {
		return email
	}
	if name, ok := c.Locals("name").(string); ok && name != "" {
		return name
	}
	return "admin"
}
