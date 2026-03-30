package handlers

import (
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// validateRoutePath checks that a gateway route path is safe and well-formed.
func validateRoutePath(path string) error {
	if len(path) > 512 {
		return fiber.NewError(fiber.StatusBadRequest, "path too long (max 512 characters)")
	}
	if !strings.HasPrefix(path, "/") {
		return fiber.NewError(fiber.StatusBadRequest, "path must start with /")
	}
	for _, blocked := range []string{".{1000", "(.+)+", "((", "\\x"} {
		if strings.Contains(path, blocked) {
			return fiber.NewError(fiber.StatusBadRequest, "path contains invalid pattern")
		}
	}
	return nil
}

// requireProPlan checks that the user is on Pro plan or higher.
func requireProPlan(planRepo ports.UserPlanRepository, c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	plan, err := planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusForbidden, "could not determine plan")
	}
	if plan.Tier == entities.PlanFree {
		return fiber.NewError(fiber.StatusForbidden, "custom domains require Pro plan or higher")
	}
	return nil
}

// GatewayHandler manages API gateway HTTP endpoints.
type GatewayHandler struct {
	gwSvc       *services.GatewayService
	gwRepo      ports.GatewayRepository
	projectRepo ports.ProjectRepository
	planRepo    ports.UserPlanRepository
}

// NewGatewayHandler creates a new GatewayHandler.
func NewGatewayHandler(gwSvc *services.GatewayService, gwRepo ports.GatewayRepository, projectRepo ports.ProjectRepository, planRepo ports.UserPlanRepository) *GatewayHandler {
	return &GatewayHandler{gwSvc: gwSvc, gwRepo: gwRepo, projectRepo: projectRepo, planRepo: planRepo}
}

// CreateGateway creates a new API gateway.
// POST /api/v1/gateways
func (h *GatewayHandler) CreateGateway(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var input struct {
		Name      string `json:"name"`
		ProjectID string `json:"project_id"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	// Resolve project_id: use provided or fall back to default project
	projectID := input.ProjectID
	if projectID == "" && h.projectRepo != nil {
		if dp, err := h.projectRepo.GetDefaultProject(c.Context(), userID); err == nil {
			projectID = dp.ID
		}
	}

	gw, err := h.gwSvc.CreateGateway(c.Context(), userID, projectID, input.Name)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(gw)
}

// ListGateways lists the user's gateways.
// GET /api/v1/gateways?project_id=xxx
func (h *GatewayHandler) ListGateways(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var gws []entities.Gateway
	var err error
	projectID := c.Query("project_id")
	if projectID != "" {
		gws, err = h.gwRepo.ListGatewaysByProject(c.Context(), projectID)
	} else {
		gws, err = h.gwSvc.ListGateways(c.Context(), userID)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if gws == nil {
		gws = []entities.Gateway{}
	}

	return c.JSON(gws)
}

// GetGateway returns a single gateway with its routes and groups.
// GET /api/v1/gateways/:gwId
func (h *GatewayHandler) GetGateway(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwSvc.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	routes, err := h.gwRepo.ListRoutesByGateway(c.Context(), gwID)
	if err != nil {
		routes = []entities.GatewayRoute{}
	}
	if routes == nil {
		routes = []entities.GatewayRoute{}
	}

	groups, err := h.gwRepo.ListGroupsByGateway(c.Context(), gwID)
	if err != nil {
		groups = []entities.GatewayGroup{}
	}
	if groups == nil {
		groups = []entities.GatewayGroup{}
	}

	return c.JSON(fiber.Map{
		"gateway": gw,
		"routes":  routes,
		"groups":  groups,
	})
}

// UpdateGateway updates a gateway name.
// PUT /api/v1/gateways/:gwId
func (h *GatewayHandler) UpdateGateway(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	var input struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	updated, err := h.gwSvc.UpdateGateway(c.Context(), gwID, input.Name)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(updated)
}

// DeleteGateway deletes a gateway and all its K8s resources.
// DELETE /api/v1/gateways/:gwId
func (h *GatewayHandler) DeleteGateway(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	if err := h.gwSvc.DeleteGateway(c.Context(), gwID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "gateway deleted"})
}

// CreateRoute adds a route to a gateway.
// POST /api/v1/gateways/:gwId/routes
func (h *GatewayHandler) CreateRoute(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	var input struct {
		Name        string                       `json:"name"`
		Path        string                       `json:"path"`
		Methods     []string                     `json:"methods"`
		AppID       string                       `json:"app_id"`
		GroupID     string                       `json:"group_id"`
		StripPrefix bool                         `json:"strip_prefix"`
		Auth        entities.GatewayRouteAuth    `json:"auth"`
		AuthPoolID  string                       `json:"auth_pool_id"`
		Plugins     []entities.GatewayRoutePlugin `json:"plugins"`
		Priority    int                          `json:"priority"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" || input.Path == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name and path are required")
	}
	if err := validateRoutePath(input.Path); err != nil {
		return err
	}
	// app_id is required for standalone routes (not in a group)
	if input.AppID == "" && input.GroupID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "app_id is required (or assign to a group)")
	}
	if len(input.Methods) == 0 {
		input.Methods = []string{"GET"}
	}

	route := &entities.GatewayRoute{
		Name:        input.Name,
		Path:        input.Path,
		Methods:     input.Methods,
		AppID:       input.AppID,
		GroupID:     input.GroupID,
		StripPrefix: input.StripPrefix,
		Auth:        input.Auth,
		AuthPoolID:  input.AuthPoolID,
		Plugins:     input.Plugins,
		Priority:    input.Priority,
	}

	created, err := h.gwSvc.CreateRoute(c.Context(), gwID, route)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

// ListRoutes lists routes in a gateway.
// GET /api/v1/gateways/:gwId/routes
func (h *GatewayHandler) ListRoutes(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	routes, err := h.gwRepo.ListRoutesByGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if routes == nil {
		routes = []entities.GatewayRoute{}
	}

	return c.JSON(routes)
}

// UpdateRoute updates a route.
// PUT /api/v1/gateways/:gwId/routes/:routeId
func (h *GatewayHandler) UpdateRoute(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")
	routeID := c.Params("routeId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	var input struct {
		Name        string                       `json:"name"`
		Path        string                       `json:"path"`
		Methods     []string                     `json:"methods"`
		AppID       string                       `json:"app_id"`
		GroupID     string                       `json:"group_id"`
		StripPrefix bool                         `json:"strip_prefix"`
		Auth        entities.GatewayRouteAuth    `json:"auth"`
		AuthPoolID  string                       `json:"auth_pool_id"`
		Plugins     []entities.GatewayRoutePlugin `json:"plugins"`
		Priority    int                          `json:"priority"`
		Status      entities.GatewayRouteStatus  `json:"status"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Path != "" {
		if err := validateRoutePath(input.Path); err != nil {
			return err
		}
	}

	route := &entities.GatewayRoute{
		Name:        input.Name,
		Path:        input.Path,
		Methods:     input.Methods,
		AppID:       input.AppID,
		GroupID:     input.GroupID,
		StripPrefix: input.StripPrefix,
		Auth:        input.Auth,
		AuthPoolID:  input.AuthPoolID,
		Plugins:     input.Plugins,
		Priority:    input.Priority,
		Status:      input.Status,
	}

	updated, err := h.gwSvc.UpdateRoute(c.Context(), gwID, routeID, route)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(updated)
}

// DeleteRoute deletes a route.
// DELETE /api/v1/gateways/:gwId/routes/:routeId
func (h *GatewayHandler) DeleteRoute(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")
	routeID := c.Params("routeId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	if err := h.gwSvc.DeleteRoute(c.Context(), gwID, routeID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "route deleted"})
}

// SyncGateway forces reconciliation of K8s CRDs from DB state.
// POST /api/v1/gateways/:gwId/sync
func (h *GatewayHandler) SyncGateway(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	if err := h.gwSvc.SyncGateway(c.Context(), gwID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "gateway synced"})
}

// --- Group Handlers ---

// CreateGroup creates a new group in a gateway.
// POST /api/v1/gateways/:gwId/groups
func (h *GatewayHandler) CreateGroup(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	var input struct {
		Name    string                        `json:"name"`
		AppID   string                        `json:"app_id"`
		Plugins []entities.GatewayRoutePlugin  `json:"plugins"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" || input.AppID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name and app_id are required")
	}

	group := &entities.GatewayGroup{
		Name:    input.Name,
		AppID:   input.AppID,
		Plugins: input.Plugins,
	}

	created, err := h.gwSvc.CreateGroup(c.Context(), gwID, group)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

// ListGroups lists groups in a gateway.
// GET /api/v1/gateways/:gwId/groups
func (h *GatewayHandler) ListGroups(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	groups, err := h.gwSvc.ListGroups(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if groups == nil {
		groups = []entities.GatewayGroup{}
	}

	return c.JSON(groups)
}

// UpdateGroup updates a group.
// PUT /api/v1/gateways/:gwId/groups/:groupId
func (h *GatewayHandler) UpdateGroup(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")
	groupID := c.Params("groupId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	var input struct {
		Name    string                        `json:"name"`
		AppID   string                        `json:"app_id"`
		Plugins []entities.GatewayRoutePlugin  `json:"plugins"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	group := &entities.GatewayGroup{
		Name:    input.Name,
		AppID:   input.AppID,
		Plugins: input.Plugins,
	}

	updated, err := h.gwSvc.UpdateGroup(c.Context(), gwID, groupID, group)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(updated)
}

// DeleteGroup deletes a group.
// DELETE /api/v1/gateways/:gwId/groups/:groupId
func (h *GatewayHandler) DeleteGroup(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")
	groupID := c.Params("groupId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	if err := h.gwSvc.DeleteGroup(c.Context(), gwID, groupID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "group deleted"})
}

// --- Custom Domain Handlers ---

// AddDomain adds a custom domain to a gateway (Pro+ only).
// POST /api/v1/gateways/:gwId/domains
func (h *GatewayHandler) AddDomain(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	if err := requireProPlan(h.planRepo, c); err != nil {
		return err
	}

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	var input struct {
		Domain string `json:"domain"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Domain == "" {
		return fiber.NewError(fiber.StatusBadRequest, "domain is required")
	}

	cd, err := h.gwSvc.AddGatewayDomain(c.Context(), gwID, userID, input.Domain)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(cd)
}

// ListDomains lists custom domains for a gateway.
// GET /api/v1/gateways/:gwId/domains
func (h *GatewayHandler) ListDomains(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	domains, err := h.gwSvc.ListGatewayDomains(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(domains)
}

// DeleteDomain deletes a custom domain from a gateway.
// DELETE /api/v1/gateways/:gwId/domains/:domainId
func (h *GatewayHandler) DeleteDomain(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")
	domainID := c.Params("domainId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	if err := h.gwSvc.DeleteGatewayDomain(c.Context(), gwID, domainID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "domain deleted"})
}

// --- Analytics Handlers ---

// GetAnalytics returns gateway analytics overview.
// GET /api/v1/gateways/:gwId/analytics
func (h *GatewayHandler) GetAnalytics(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	overview, err := h.gwSvc.GetGatewayAnalytics(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(overview)
}

// GetAnalyticsTimeSeries returns gateway analytics time series.
// GET /api/v1/gateways/:gwId/analytics/timeseries?metric=requests|latency|errors&range=1h|6h|24h|7d
func (h *GatewayHandler) GetAnalyticsTimeSeries(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	gwID := c.Params("gwId")

	gw, err := h.gwRepo.GetGateway(c.Context(), gwID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "gateway not found")
	}
	if gw.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your gateway")
	}

	metric := c.Query("metric", "requests")
	timeRange := c.Query("range", "1h")

	ts, err := h.gwSvc.GetGatewayTimeSeries(c.Context(), gwID, metric, timeRange)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(ts)
}
