package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// BillingHandler manages billing endpoints.
type BillingHandler struct {
	svc *services.BillingService
}

// NewBillingHandler creates a new BillingHandler.
func NewBillingHandler(svc *services.BillingService) *BillingHandler {
	return &BillingHandler{svc: svc}
}

// GetBillingStatus returns the current user's plan + subscription + usage.
// GET /api/v1/billing
func (h *BillingHandler) GetBillingStatus(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	resp, err := h.svc.GetBillingStatus(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(resp)
}

// CreateCheckoutSession creates a Stripe Checkout session and returns the URL.
// POST /api/v1/billing/checkout
func (h *BillingHandler) CreateCheckoutSession(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	userEmail, _ := c.Locals("email").(string)

	var input dto.CreateCheckoutInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	resp, err := h.svc.CreateCheckoutSession(c.Context(), userID, userEmail, input.Tier)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(resp)
}

// CreatePortalSession creates a Stripe Customer Portal session.
// POST /api/v1/billing/portal
func (h *BillingHandler) CreatePortalSession(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	resp, err := h.svc.CreatePortalSession(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(resp)
}

// CancelSubscription cancels the user's subscription.
// POST /api/v1/billing/cancel
func (h *BillingHandler) CancelSubscription(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var input dto.CancelSubscriptionInput
	if err := c.BodyParser(&input); err != nil {
		input.Immediate = false
	}

	if err := h.svc.CancelSubscription(c.Context(), userID, input.Immediate); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"status": "canceling", "cancel_at_period_end": !input.Immediate})
}

// ListInvoices returns the user's invoice history.
// GET /api/v1/billing/invoices
func (h *BillingHandler) ListInvoices(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	items, err := h.svc.ListInvoices(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

// GetAdminBillingOverview returns admin billing metrics.
// GET /api/v1/admin/billing/overview
func (h *BillingHandler) GetAdminBillingOverview(c *fiber.Ctx) error {
	resp, err := h.svc.GetAdminBillingOverview(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(resp)
}
