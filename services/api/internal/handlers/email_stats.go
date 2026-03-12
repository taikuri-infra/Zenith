package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// EmailStatsHandler exposes admin endpoints for email campaign stats.
type EmailStatsHandler struct {
	emailSendRepo ports.EmailSendRepository
}

func NewEmailStatsHandler(repo ports.EmailSendRepository) *EmailStatsHandler {
	return &EmailStatsHandler{emailSendRepo: repo}
}

// GetStats returns aggregated email campaign metrics.
// GET /api/v1/admin/emails/stats
func (h *EmailStatsHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.emailSendRepo.GetStats(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get email stats")
	}
	return c.JSON(stats)
}
