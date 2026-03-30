package handlers

import (
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// DeployHookHandler handles CRUD for post-deploy hooks.
type DeployHookHandler struct {
	hookRepo ports.DeployHookRepository
}

// NewDeployHookHandler creates a new DeployHookHandler.
func NewDeployHookHandler(hookRepo ports.DeployHookRepository) *DeployHookHandler {
	return &DeployHookHandler{hookRepo: hookRepo}
}

type createHookRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`    // "http" or "command"
	URL     string `json:"url"`     // required for http
	Command string `json:"command"` // required for command
	Order   int    `json:"order"`
}

// Create handles POST /apps/:appId/hooks
func (h *DeployHookHandler) Create(c *fiber.Ctx) error {
	appID := c.Params("appId")

	var req createHookRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if req.Name == "" {
		return NewBadRequest("name is required")
	}
	if len(req.Name) > 100 {
		return NewBadRequest("name too long (max 100)")
	}

	hookType := entities.DeployHookType(req.Type)
	if hookType != entities.DeployHookHTTP && hookType != entities.DeployHookCommand {
		return NewBadRequest("type must be 'http' or 'command'")
	}
	if hookType == entities.DeployHookHTTP && req.URL == "" {
		return NewBadRequest("url is required for http hooks")
	}
	if hookType == entities.DeployHookHTTP && !strings.HasPrefix(req.URL, "http") {
		return NewBadRequest("url must start with http:// or https://")
	}
	if hookType == entities.DeployHookCommand && req.Command == "" {
		return NewBadRequest("command is required for command hooks")
	}
	if len(req.Command) > 512 {
		return NewBadRequest("command too long (max 512)")
	}

	// Max 10 hooks per app
	count, err := h.hookRepo.CountHooksByApp(c.Context(), appID)
	if err != nil {
		return NewInternal("failed to count hooks")
	}
	if count >= 10 {
		return NewBadRequest("maximum 10 hooks per app")
	}

	hook, err := h.hookRepo.CreateHook(c.Context(), &entities.DeployHook{
		AppID:   appID,
		Name:    req.Name,
		Type:    hookType,
		URL:     req.URL,
		Command: req.Command,
		Order:   req.Order,
		Active:  true,
	})
	if err != nil {
		return NewInternal("failed to create hook")
	}

	return c.Status(fiber.StatusCreated).JSON(hook)
}

// List handles GET /apps/:appId/hooks
func (h *DeployHookHandler) List(c *fiber.Ctx) error {
	appID := c.Params("appId")

	hooks, err := h.hookRepo.ListHooksByApp(c.Context(), appID)
	if err != nil {
		return NewInternal("failed to list hooks")
	}
	if hooks == nil {
		hooks = []entities.DeployHook{}
	}

	return c.JSON(fiber.Map{"items": hooks, "total": len(hooks)})
}

type updateHookRequest struct {
	Name    *string `json:"name"`
	URL     *string `json:"url"`
	Command *string `json:"command"`
	Order   *int    `json:"order"`
	Active  *bool   `json:"active"`
}

// Update handles PUT /apps/:appId/hooks/:hookId
func (h *DeployHookHandler) Update(c *fiber.Ctx) error {
	hookID := c.Params("hookId")
	appID := c.Params("appId")

	hook, err := h.hookRepo.GetHook(c.Context(), hookID)
	if err != nil {
		return NewNotFound("hook not found")
	}
	if hook.AppID != appID {
		return NewForbidden("hook does not belong to this app")
	}

	var req updateHookRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}

	updated, err := h.hookRepo.UpdateHook(c.Context(), hookID, req.Name, req.URL, req.Command, req.Order, req.Active)
	if err != nil {
		return NewInternal("failed to update hook")
	}

	return c.JSON(updated)
}

// Delete handles DELETE /apps/:appId/hooks/:hookId
func (h *DeployHookHandler) Delete(c *fiber.Ctx) error {
	hookID := c.Params("hookId")
	appID := c.Params("appId")

	hook, err := h.hookRepo.GetHook(c.Context(), hookID)
	if err != nil {
		return NewNotFound("hook not found")
	}
	if hook.AppID != appID {
		return NewForbidden("hook does not belong to this app")
	}

	if err := h.hookRepo.DeleteHook(c.Context(), hookID); err != nil {
		return NewInternal("failed to delete hook")
	}

	return c.JSON(fiber.Map{"message": "hook deleted"})
}
