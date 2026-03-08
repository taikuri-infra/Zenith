package handlers

import (
	"strconv"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// NotificationHandler handles notification and activity endpoints.
type NotificationHandler struct {
	repo ports.NotificationRepository
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(repo ports.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

// List returns the user's notifications (newest first).
// GET /api/v1/notifications?limit=50
func (h *NotificationHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	notifs, err := h.repo.ListByUser(c.Context(), userID, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list notifications")
	}
	if notifs == nil {
		notifs = make([]entities.Notification, 0)
	}

	unread, _ := h.repo.CountUnread(c.Context(), userID)

	return c.JSON(fiber.Map{
		"items":  notifs,
		"unread": unread,
	})
}

// MarkRead marks specific notifications as read.
// POST /api/v1/notifications/read { "ids": ["..."] }
func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var body struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&body); err != nil || len(body.IDs) == 0 {
		// No IDs = mark all as read
		if err := h.repo.MarkAllRead(c.Context(), userID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to mark notifications read")
		}
		return c.JSON(fiber.Map{"message": "all notifications marked as read"})
	}

	if err := h.repo.MarkRead(c.Context(), userID, body.IDs); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to mark notifications read")
	}
	return c.JSON(fiber.Map{"message": "ok"})
}

// ListActivity returns the user's activity log.
// GET /api/v1/activity?limit=50
func (h *NotificationHandler) ListActivity(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	entries, err := h.repo.ListActivity(c.Context(), userID, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list activity")
	}
	if entries == nil {
		entries = make([]entities.ActivityEntry, 0)
	}

	return c.JSON(entries)
}
