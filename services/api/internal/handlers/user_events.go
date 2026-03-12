package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// UserEventHandler exposes admin endpoints for querying tracked user events.
type UserEventHandler struct {
	eventRepo ports.UserEventRepository
}

func NewUserEventHandler(repo ports.UserEventRepository) *UserEventHandler {
	return &UserEventHandler{eventRepo: repo}
}

// ListEvents returns events filtered by type and/or since date.
// GET /api/v1/admin/events?type=signup&since=2026-03-01&limit=100&offset=0
func (h *UserEventHandler) ListEvents(c *fiber.Ctx) error {
	eventType := c.Query("type")
	sinceStr := c.Query("since")
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	if eventType != "" {
		events, err := h.eventRepo.ListByType(c.Context(), eventType, limit, offset)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to list events")
		}
		if events == nil {
			events = []entities.UserEvent{}
		}
		return c.JSON(fiber.Map{"items": events, "total": len(events)})
	}

	// If no type filter, return counts by type for overview
	since := time.Now().AddDate(0, -1, 0)
	if sinceStr != "" {
		if t, err := time.Parse("2006-01-02", sinceStr); err == nil {
			since = t
		}
	}

	types := []string{
		"signup", "login", "app.create", "app.deploy", "app.delete",
		"db.create", "domain.add", "bucket.create",
		"upgrade.start", "upgrade.complete", "plan.cancel",
		"feature.gated", "trial.start", "trial.end",
		"onboarding.step", "onboarding.done",
		"referral.share", "referral.signup",
	}
	counts := make(map[string]int)
	for _, t := range types {
		count, _ := h.eventRepo.CountByType(c.Context(), t, since)
		if count > 0 {
			counts[t] = count
		}
	}
	return c.JSON(fiber.Map{"counts": counts, "since": since.Format("2006-01-02")})
}

// GetFunnel returns funnel conversion data.
// GET /api/v1/admin/events/funnel?steps=signup,app.create,app.deploy&since=2026-03-01
func (h *UserEventHandler) GetFunnel(c *fiber.Ctx) error {
	stepsStr := c.Query("steps", "signup,app.create,app.deploy")
	sinceStr := c.Query("since")

	steps := strings.Split(stepsStr, ",")
	since := time.Now().AddDate(0, -1, 0)
	if sinceStr != "" {
		if t, err := time.Parse("2006-01-02", sinceStr); err == nil {
			since = t
		}
	}

	data, err := h.eventRepo.GetFunnelData(c.Context(), steps, since)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get funnel data")
	}

	return c.JSON(fiber.Map{"funnel": data, "steps": steps, "since": since.Format("2006-01-02")})
}

// GetUserActivity returns the event timeline for a specific user.
// GET /api/v1/admin/events/user/:id
func (h *UserEventHandler) GetUserActivity(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "user id is required")
	}

	since := time.Now().AddDate(0, -3, 0)
	events, err := h.eventRepo.GetUserActivity(c.Context(), userID, since)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get user activity")
	}
	if events == nil {
		events = []entities.UserEvent{}
	}
	return c.JSON(fiber.Map{"items": events, "user_id": userID})
}
