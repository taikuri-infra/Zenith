package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// UserWebhookHandler handles user-defined webhook endpoints.
type UserWebhookHandler struct {
	webhookRepo store.UserWebhookRepository
	planRepo    store.UserPlanRepository
}

func NewUserWebhookHandler(webhookRepo store.UserWebhookRepository, planRepo store.UserPlanRepository) *UserWebhookHandler {
	return &UserWebhookHandler{webhookRepo: webhookRepo, planRepo: planRepo}
}

// Create registers a new webhook endpoint.
func (h *UserWebhookHandler) Create(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var body struct {
		URL    string                  `json:"url"`
		Events []entities.WebhookEvent `json:"events"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.URL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url is required")
	}
	if len(body.Events) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "at least one event is required")
	}

	// Check plan limit: free=0, pro=5, team=10, enterprise=50
	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier == entities.PlanFree {
		return fiber.NewError(fiber.StatusForbidden, "webhooks require Pro plan or higher")
	}

	maxWebhooks := 5
	switch plan.Tier {
	case entities.PlanTeam:
		maxWebhooks = 10
	case entities.PlanEnterprise:
		maxWebhooks = 50
	}

	count, _ := h.webhookRepo.CountWebhooksByUser(c.Context(), userID)
	if count >= maxWebhooks {
		return fiber.NewError(fiber.StatusForbidden, "webhook limit reached for your plan")
	}

	webhook, err := h.webhookRepo.CreateWebhook(c.Context(), userID, body.URL, body.Events)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(webhook)
}

// List returns all webhooks for the authenticated user.
func (h *UserWebhookHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	webhooks, err := h.webhookRepo.ListWebhooksByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if webhooks == nil {
		webhooks = []entities.UserWebhook{}
	}

	return c.JSON(fiber.Map{"items": webhooks})
}

// Update modifies a webhook.
func (h *UserWebhookHandler) Update(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	webhookID := c.Params("webhookId")

	// Verify ownership
	webhook, err := h.webhookRepo.GetWebhook(c.Context(), webhookID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "webhook not found")
	}
	if webhook.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your webhook")
	}

	var body struct {
		URL    *string                 `json:"url,omitempty"`
		Events []entities.WebhookEvent `json:"events,omitempty"`
		Active *bool                   `json:"active,omitempty"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	updated, err := h.webhookRepo.UpdateWebhook(c.Context(), webhookID, body.URL, body.Events, body.Active)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(updated)
}

// Delete removes a webhook.
func (h *UserWebhookHandler) Delete(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	webhookID := c.Params("webhookId")

	// Verify ownership
	webhook, err := h.webhookRepo.GetWebhook(c.Context(), webhookID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "webhook not found")
	}
	if webhook.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your webhook")
	}

	if err := h.webhookRepo.DeleteWebhook(c.Context(), webhookID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListDeliveries returns delivery log for a specific webhook.
func (h *UserWebhookHandler) ListDeliveries(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	webhookID := c.Params("webhookId")

	// Verify ownership
	webhook, err := h.webhookRepo.GetWebhook(c.Context(), webhookID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "webhook not found")
	}
	if webhook.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your webhook")
	}

	deliveries, err := h.webhookRepo.ListDeliveries(c.Context(), webhookID, 50)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if deliveries == nil {
		deliveries = []entities.WebhookDelivery{}
	}

	return c.JSON(fiber.Map{"items": deliveries})
}
