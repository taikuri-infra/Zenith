package handlers

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/capi"
	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// AdminHandler serves all /api/v1/admin/* endpoints for Mission Control.
type AdminHandler struct {
	capiClient *capi.Client
	k8sClient  k8s.Client
	store      store.AdminRepository
}

// NewAdminHandler creates a new AdminHandler with the given dependencies.
func NewAdminHandler(k8sClient k8s.Client, capiClient *capi.Client, adminStore store.AdminRepository) *AdminHandler {
	return &AdminHandler{
		capiClient: capiClient,
		k8sClient:  k8sClient,
		store:      adminStore,
	}
}

// ---------- Dashboard ----------

// GetDashboardStats returns aggregate statistics for the Mission Control dashboard.
// GET /api/v1/admin/dashboard/stats
func (h *AdminHandler) GetDashboardStats(c *fiber.Ctx) error {
	clusters, err := h.capiClient.ListClusters(c.Context())
	if err != nil {
		clusters = []models.Cluster{}
	}

	allHealthy := true
	for _, cl := range clusters {
		if cl.Status != "healthy" {
			allHealthy = false
			break
		}
	}

	// Count tenants from Project CRDs
	projects, _ := h.k8sClient.ListCRDs(c.Context(), "Project", "")
	tenantCount := len(projects)

	// Count active tenants (simplification: all non-suspended projects are active)
	activeToday := 0
	for _, p := range projects {
		suspended := p.Metadata.Annotations["zenith.dev/suspended"]
		if suspended != "true" {
			activeToday++
		}
	}

	modules, _ := h.store.ListModules(c.Context())
	updatesAvailable := 0
	for _, m := range modules {
		if m.Status == "update_available" {
			updatesAvailable++
		}
	}

	return c.JSON(models.DashboardStats{
		ClusterCount:     len(clusters),
		AllHealthy:       allHealthy,
		TenantCount:      tenantCount,
		ActiveToday:      activeToday,
		MonthlyCost:      "EUR 47.60",
		CostProvider:     "Hetzner Cloud",
		UpdatesAvailable: updatesAvailable,
	})
}

// ---------- Clusters ----------

// ListClusters returns all CAPI-managed clusters.
// GET /api/v1/admin/clusters
func (h *AdminHandler) ListClusters(c *fiber.Ctx) error {
	clusters, err := h.capiClient.ListClusters(c.Context())
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

	cluster, err := h.capiClient.GetCluster(c.Context(), name)
	if err != nil {
		return NewNotFound("cluster")
	}
	return c.JSON(cluster)
}

// CreateCluster provisions a new CAPI cluster.
// POST /api/v1/admin/clusters
func (h *AdminHandler) CreateCluster(c *fiber.Ctx) error {
	var input models.CreateClusterInput
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

	cluster, err := h.capiClient.CreateCluster(c.Context(), input)
	if err != nil {
		return NewConflict("cluster already exists")
	}

	_ = h.store.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:    time.Now().Format("15:04"),
		Actor:   actorFromContext(c),
		Action:  "Created cluster " + input.Name,
		Cluster: input.Name,
	})

	return c.Status(fiber.StatusCreated).JSON(cluster)
}

// DeleteCluster removes a CAPI cluster.
// DELETE /api/v1/admin/clusters/:name
func (h *AdminHandler) DeleteCluster(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("cluster name is required")
	}

	if err := h.capiClient.DeleteCluster(c.Context(), name); err != nil {
		return NewNotFound("cluster")
	}

	_ = h.store.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:    time.Now().Format("15:04"),
		Actor:   actorFromContext(c),
		Action:  "Deleted cluster " + name,
		Cluster: name,
	})

	return c.JSON(fiber.Map{"message": "cluster scheduled for deletion"})
}

// UpgradeCluster initiates a Kubernetes version upgrade.
// POST /api/v1/admin/clusters/:name/upgrade
func (h *AdminHandler) UpgradeCluster(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("cluster name is required")
	}

	var input models.UpgradeClusterInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}
	if input.Version == "" {
		return NewBadRequest("version is required")
	}

	if err := h.capiClient.UpgradeCluster(c.Context(), name, input.Version); err != nil {
		return NewNotFound("cluster")
	}

	_ = h.store.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:    time.Now().Format("15:04"),
		Actor:   actorFromContext(c),
		Action:  "Initiated upgrade to " + input.Version,
		Cluster: name,
	})

	return c.JSON(fiber.Map{"message": "cluster upgrade initiated"})
}

// ---------- Tenants ----------

// ListTenants returns all tenants derived from Project CRDs.
// GET /api/v1/admin/tenants
func (h *AdminHandler) ListTenants(c *fiber.Ctx) error {
	projects, err := h.k8sClient.ListCRDs(c.Context(), "Project", "")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list tenants")
	}

	tenants := make([]models.Tenant, 0, len(projects))
	for _, p := range projects {
		tenants = append(tenants, projectToTenant(p))
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

	proj, err := h.k8sClient.GetCRD(c.Context(), "Project", "", id)
	if err != nil {
		return NewNotFound("tenant")
	}

	tenant := projectToTenant(proj)
	return c.JSON(tenant)
}

// SuspendTenant suspends a tenant by adding an annotation.
// POST /api/v1/admin/tenants/:id/suspend
func (h *AdminHandler) SuspendTenant(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("tenant id is required")
	}

	proj, err := h.k8sClient.GetCRD(c.Context(), "Project", "", id)
	if err != nil {
		return NewNotFound("tenant")
	}

	if proj.Metadata.Annotations == nil {
		proj.Metadata.Annotations = make(map[string]string)
	}
	proj.Metadata.Annotations["zenith.dev/suspended"] = "true"
	proj.Metadata.Annotations["zenith.dev/suspended-at"] = time.Now().Format(time.RFC3339)

	if err := h.k8sClient.UpdateCRD(c.Context(), proj); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to suspend tenant")
	}

	_ = h.store.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Suspended tenant " + id,
	})

	return c.JSON(fiber.Map{"message": "tenant suspended"})
}

// ---------- Modules ----------

// ListModules returns all platform modules.
// GET /api/v1/admin/modules
func (h *AdminHandler) ListModules(c *fiber.Ctx) error {
	modules, err := h.store.ListModules(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list modules")
	}
	return c.JSON(modules)
}

// InstallModule triggers installation of a module (placeholder).
// POST /api/v1/admin/modules/:name/install
func (h *AdminHandler) InstallModule(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("module name is required")
	}

	_ = h.store.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Installed module " + name,
	})

	return c.JSON(fiber.Map{"message": "module installation initiated"})
}

// UninstallModule triggers uninstallation of a module (placeholder).
// POST /api/v1/admin/modules/:name/uninstall
func (h *AdminHandler) UninstallModule(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("module name is required")
	}

	_ = h.store.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Uninstalled module " + name,
	})

	return c.JSON(fiber.Map{"message": "module uninstallation initiated"})
}

// UpdateModule updates a module to its latest version.
// POST /api/v1/admin/modules/:name/update
func (h *AdminHandler) UpdateModule(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("module name is required")
	}

	mod, err := h.store.UpdateModule(c.Context(), name)
	if err != nil {
		return NewNotFound("module")
	}

	_ = h.store.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Updated module " + name + " to " + mod.Installed,
	})

	return c.JSON(mod)
}

// UpdateAllModules updates all modules that have updates available.
// POST /api/v1/admin/modules/update-all
func (h *AdminHandler) UpdateAllModules(c *fiber.Ctx) error {
	modules, err := h.store.ListModules(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list modules")
	}

	updated := 0
	for _, m := range modules {
		if m.Status == "update_available" {
			_, _ = h.store.UpdateModule(c.Context(), m.Name)
			updated++
		}
	}

	_ = h.store.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Updated all modules (" + strconv.Itoa(updated) + " updated)",
	})

	return c.JSON(fiber.Map{"message": "all modules updated", "count": updated})
}

// ---------- Audit Log ----------

// ListAuditLog returns audit-log entries with optional filtering.
// GET /api/v1/admin/audit
func (h *AdminHandler) ListAuditLog(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	entries, err := h.store.ListAuditLog(c.Context(), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list audit log")
	}
	return c.JSON(entries)
}

// ---------- Platform Updates ----------

// CheckUpdates returns available platform updates.
// GET /api/v1/admin/updates/check
func (h *AdminHandler) CheckUpdates(c *fiber.Ctx) error {
	update, err := h.store.GetPlatformUpdate(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to check updates")
	}
	return c.JSON(update)
}

// ApplyUpdate applies a platform update (placeholder).
// POST /api/v1/admin/updates/apply
func (h *AdminHandler) ApplyUpdate(c *fiber.Ctx) error {
	var input models.ApplyUpdateInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}
	if input.Version == "" {
		return NewBadRequest("version is required")
	}

	_ = h.store.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Applied platform update " + input.Version,
	})

	return c.JSON(fiber.Map{"message": "platform update initiated", "version": input.Version})
}

// ListUpdateHistory returns the history of past platform updates.
// GET /api/v1/admin/updates/history
func (h *AdminHandler) ListUpdateHistory(c *fiber.Ctx) error {
	history, err := h.store.ListUpdateHistory(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list update history")
	}
	return c.JSON(history)
}

// ---------- Infrastructure ----------

// GetInfraOverview returns a summary of Hetzner infrastructure.
// GET /api/v1/admin/infrastructure
func (h *AdminHandler) GetInfraOverview(c *fiber.Ctx) error {
	clusters, _ := h.capiClient.ListClusters(c.Context())

	totalServers := 0
	resources := make([]models.InfraNode, 0)
	for _, cl := range clusters {
		totalServers += cl.Nodes
		resources = append(resources, models.InfraNode{
			Name:        cl.Name + "-pool",
			Type:        "CX22",
			Count:       cl.Nodes,
			Cluster:     cl.Name,
			MonthlyCost: "EUR " + strconv.Itoa(cl.Nodes*5) + ".00",
		})
	}

	// Management plane is always 1 server
	managementServers := 1
	totalServers += managementServers
	resources = append(resources, models.InfraNode{
		Name:        "management-plane",
		Type:        "CX22",
		Count:       managementServers,
		Cluster:     "management",
		MonthlyCost: "EUR 5.00",
	})

	return c.JSON(models.InfraOverview{
		Servers:       totalServers,
		Volumes:       len(clusters) * 2,
		VolumeSize:    strconv.Itoa(len(clusters)*20) + " GB",
		LoadBalancers: len(clusters),
		LBPublic:      len(clusters),
		LBInternal:    0,
		MonthlyCost:   "EUR " + strconv.Itoa(totalServers*5+len(clusters)*10) + ".00",
		Resources:     resources,
	})
}

// ---------- Platform State ----------

// GetPlatformState returns the current platform state summary.
// GET /api/v1/admin/state
func (h *AdminHandler) GetPlatformState(c *fiber.Ctx) error {
	update, _ := h.store.GetPlatformUpdate(c.Context())
	settings, err := h.store.GetSettings(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get settings")
	}

	updateAvailable := ""
	if update != nil && update.Version != update.Current {
		updateAvailable = update.Version
	}

	return c.JSON(models.PlatformState{
		PlatformVersion:       "v1.2.1",
		UpdateAvailable:       updateAvailable,
		InstalledDate:         "2026-01-15",
		InstalledDaysAgo:      31,
		ManagementK8sVersion:  "v1.30.2",
		ManagementK8sUpToDate: true,
		Domain:                settings.BaseDomain,
		WildcardTLS:           true,
	})
}

// ExportState exports the full platform state as JSON.
// GET /api/v1/admin/state/export
func (h *AdminHandler) ExportState(c *fiber.Ctx) error {
	clusters, _ := h.capiClient.ListClusters(c.Context())
	projects, _ := h.k8sClient.ListCRDs(c.Context(), "Project", "")
	settings, _ := h.store.GetSettings(c.Context())
	modules, _ := h.store.ListModules(c.Context())

	tenants := make([]models.Tenant, 0, len(projects))
	for _, p := range projects {
		tenants = append(tenants, projectToTenant(p))
	}

	export := map[string]interface{}{
		"exportedAt": time.Now().Format(time.RFC3339),
		"platform": map[string]interface{}{
			"version":  "v1.2.1",
			"settings": settings,
		},
		"clusters": clusters,
		"tenants":  tenants,
		"modules":  modules,
	}

	data, _ := json.MarshalIndent(export, "", "  ")
	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename=zenith-state.json")
	return c.Send(data)
}

// ---------- Settings ----------

// GetSettings returns the current platform settings.
// GET /api/v1/admin/settings
func (h *AdminHandler) GetSettings(c *fiber.Ctx) error {
	settings, err := h.store.GetSettings(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get settings")
	}
	return c.JSON(settings)
}

// UpdateSettings updates platform settings.
// PATCH /api/v1/admin/settings (also supports PUT)
func (h *AdminHandler) UpdateSettings(c *fiber.Ctx) error {
	var input models.PlatformSettings
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	updated, err := h.store.UpdateSettings(c.Context(), &input)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update settings")
	}

	_ = h.store.AddAuditEntry(c.Context(), models.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actorFromContext(c),
		Action: "Updated platform settings",
	})

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

// projectToTenant converts a Project CRD to a Tenant model.
func projectToTenant(p *k8s.CRDObject) models.Tenant {
	var spec map[string]interface{}
	_ = json.Unmarshal(p.Spec, &spec)

	displayName, _ := spec["displayName"].(string)
	if displayName == "" {
		displayName = p.Metadata.Name
	}
	plan, _ := spec["plan"].(string)
	if plan == "" {
		plan = "starter"
	}

	status := "active"
	if p.Metadata.Annotations != nil && p.Metadata.Annotations["zenith.dev/suspended"] == "true" {
		status = "suspended"
	}

	return models.Tenant{
		Name:      displayName,
		Plan:      plan,
		Apps:      0,
		Databases: 0,
		CPUUsed:   "0",
		CPULimit:  "4",
		RAMUsed:   "0",
		RAMLimit:  "4",
		Status:    status,
	}
}
