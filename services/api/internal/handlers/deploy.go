package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/dto"
"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// DeployHandler handles deployment and env var operations.
type DeployHandler struct {
	appRepo  ports.AppRepository
	pipeline interface {
		TriggerImageDeploy(app *entities.App, deployment *entities.Deployment, image string) error
	}
}

// NewDeployHandler creates a new DeployHandler.
func NewDeployHandler(appRepo ports.AppRepository, pipeline interface {
	TriggerImageDeploy(app *entities.App, deployment *entities.Deployment, image string) error
}) *DeployHandler {
	return &DeployHandler{appRepo: appRepo, pipeline: pipeline}
}

// --- Deployment endpoints ---

// ListDeployments handles GET /api/v1/apps/:appId/deployments
func (h *DeployHandler) ListDeployments(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if appID == "" {
		return NewBadRequest("app ID is required")
	}

	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}

	deployments, err := h.appRepo.ListDeployments(c.Context(), appID, limit)
	if err != nil {
		return NewInternal("failed to list deployments")
	}

	return c.JSON(fiber.Map{
		"items": deployments,
		"total": len(deployments),
	})
}

// GetDeployment handles GET /api/v1/apps/:appId/deployments/:deployId
func (h *DeployHandler) GetDeployment(c *fiber.Ctx) error {
	deployID := c.Params("deployId")
	if deployID == "" {
		return NewBadRequest("deployment ID is required")
	}

	deployment, err := h.appRepo.GetDeployment(c.Context(), deployID)
	if err != nil {
		return NewNotFound("deployment not found")
	}

	return c.JSON(deployment)
}

// Rollback handles POST /api/v1/apps/:appId/rollback
func (h *DeployHandler) Rollback(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if appID == "" {
		return NewBadRequest("app ID is required")
	}

	var req struct {
		DeploymentID string `json:"deployment_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.DeploymentID == "" {
		return NewBadRequest("deployment_id is required")
	}

	// Verify the deployment exists and belongs to this app
	target, err := h.appRepo.GetDeployment(c.Context(), req.DeploymentID)
	if err != nil {
		return NewNotFound("deployment not found")
	}
	if target.AppID != appID {
		return NewBadRequest("deployment does not belong to this app")
	}

	// Mark target deployment as active, supersede the current active one
	currentActive, _ := h.appRepo.GetActiveDeployment(c.Context(), appID)
	if currentActive != nil {
		h.appRepo.UpdateDeploymentStatus(c.Context(), currentActive.ID, entities.DeployStatusSuperseded, "", "")
	}

	if err := h.appRepo.UpdateDeploymentStatus(c.Context(), target.ID, entities.DeployStatusActive, "", ""); err != nil {
		return NewInternal("failed to rollback")
	}

	// Update app status
	status := entities.AppStatusDeploying
	h.appRepo.UpdateApp(c.Context(), appID, &dto.UpdateAppInput{
		Status: &status,
	})

	// Trigger actual K8s redeployment with the old image
	if h.pipeline != nil {
		app, _ := h.appRepo.GetApp(c.Context(), appID)
		if app != nil {
			if err := h.pipeline.TriggerImageDeploy(app, target, target.ImageTag); err != nil {
				return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
			}
		}
	}

	return c.JSON(fiber.Map{
		"message":       "rollback initiated",
		"deployment_id": target.ID,
		"git_sha":       target.GitSHA,
	})
}

// --- Env Var endpoints ---

// SetEnvVarsRequest is the request body for setting env vars.
type SetEnvVarsRequest struct {
	Vars map[string]string `json:"vars"`
}

// SetEnvVars handles PUT /api/v1/apps/:appId/env
func (h *DeployHandler) SetEnvVars(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if appID == "" {
		return NewBadRequest("app ID is required")
	}

	// Verify app exists
	if _, err := h.appRepo.GetApp(c.Context(), appID); err != nil {
		return NewNotFound("app not found")
	}

	var req SetEnvVarsRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if len(req.Vars) == 0 {
		return NewBadRequest("vars cannot be empty")
	}

	if err := h.appRepo.SetEnvVars(c.Context(), appID, req.Vars); err != nil {
		return NewInternal("failed to set env vars")
	}

	return c.JSON(fiber.Map{"message": "env vars updated", "count": len(req.Vars)})
}

// GetEnvVars handles GET /api/v1/apps/:appId/env
func (h *DeployHandler) GetEnvVars(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if appID == "" {
		return NewBadRequest("app ID is required")
	}

	vars, err := h.appRepo.GetEnvVars(c.Context(), appID)
	if err != nil {
		return NewInternal("failed to get env vars")
	}

	return c.JSON(fiber.Map{
		"items": vars,
		"total": len(vars),
	})
}

// DeleteEnvVar handles DELETE /api/v1/apps/:appId/env/:key
func (h *DeployHandler) DeleteEnvVar(c *fiber.Ctx) error {
	appID := c.Params("appId")
	key := c.Params("key")
	if appID == "" || key == "" {
		return NewBadRequest("app ID and key are required")
	}

	if err := h.appRepo.DeleteEnvVar(c.Context(), appID, key); err != nil {
		return NewNotFound("env var not found")
	}

	return c.JSON(fiber.Map{"message": "env var deleted"})
}
