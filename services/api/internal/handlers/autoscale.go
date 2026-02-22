package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// AutoscaleHandler serves autoscaler admin endpoints.
type AutoscaleHandler struct {
	repo store.AutoscaleRepository
}

// NewAutoscaleHandler creates a new AutoscaleHandler.
func NewAutoscaleHandler(repo store.AutoscaleRepository) *AutoscaleHandler {
	return &AutoscaleHandler{repo: repo}
}

// GetStatus returns the current autoscaler status.
// GET /api/v1/admin/autoscaler/status
func (h *AutoscaleHandler) GetStatus(c *fiber.Ctx) error {
	status, err := h.repo.GetStatus(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get autoscaler status")
	}
	return c.JSON(status)
}

// ListNodes returns all Hetzner nodes managed by the autoscaler.
// GET /api/v1/admin/autoscaler/nodes
func (h *AutoscaleHandler) ListNodes(c *fiber.Ctx) error {
	nodes, err := h.repo.ListNodes(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list nodes")
	}
	return c.JSON(fiber.Map{"items": nodes, "total": len(nodes)})
}

// ListEvents returns recent autoscale events.
// GET /api/v1/admin/autoscaler/events
func (h *AutoscaleHandler) ListEvents(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	events, err := h.repo.ListScaleEvents(c.Context(), limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list events")
	}
	return c.JSON(fiber.Map{"items": events, "total": len(events)})
}
