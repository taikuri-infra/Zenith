package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// GatewayHandler manages API gateway HTTP endpoints.
type GatewayHandler struct {
	gwSvc       *services.GatewayService
	gwRepo      ports.GatewayRepository
	projectRepo ports.ProjectRepository
}

// NewGatewayHandler creates a new GatewayHandler.
func NewGatewayHandler(gwSvc *services.GatewayService, gwRepo ports.GatewayRepository, projectRepo ports.ProjectRepository) *GatewayHandler {
	return &GatewayHandler{gwSvc: gwSvc, gwRepo: gwRepo, projectRepo: projectRepo}
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

// GetGateway returns a single gateway with its routes.
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

	return c.JSON(fiber.Map{
		"gateway": gw,
		"routes":  routes,
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
		StripPrefix bool                         `json:"strip_prefix"`
		Auth        entities.GatewayRouteAuth    `json:"auth"`
		AuthPoolID  string                       `json:"auth_pool_id"`
		Plugins     []entities.GatewayRoutePlugin `json:"plugins"`
		Priority    int                          `json:"priority"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" || input.Path == "" || input.AppID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name, path, and app_id are required")
	}
	if len(input.Methods) == 0 {
		input.Methods = []string{"GET"}
	}

	route := &entities.GatewayRoute{
		Name:        input.Name,
		Path:        input.Path,
		Methods:     input.Methods,
		AppID:       input.AppID,
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

	route := &entities.GatewayRoute{
		Name:        input.Name,
		Path:        input.Path,
		Methods:     input.Methods,
		AppID:       input.AppID,
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
