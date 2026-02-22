package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/store"
	stripeClient "github.com/dotechhq/zenith/services/api/internal/stripe"
	"github.com/gofiber/fiber/v2"
)

// PriceForTier returns (priceCents, priceID) for a given tier.
func PriceForTier(tier entities.PlanTier, proPriceID, teamPriceID string) (int, string) {
	switch tier {
	case entities.PlanPro:
		return 2900, proPriceID
	case entities.PlanTeam:
		return 19900, teamPriceID
	default:
		return 0, ""
	}
}

// BillingHandler manages billing endpoints.
type BillingHandler struct {
	stripe      stripeClient.StripeAPI
	billingRepo store.BillingRepository
	planRepo    store.UserPlanRepository
	appRepo     store.AppRepository
	dbRepo      store.DatabaseRepository
	storageRepo store.StorageRepository
	authRepo    store.AppAuthRepository
	proPriceID  string
	teamPriceID string
	baseDomain  string
}

// NewBillingHandler creates a new BillingHandler.
func NewBillingHandler(
	stripe stripeClient.StripeAPI,
	billingRepo store.BillingRepository,
	planRepo store.UserPlanRepository,
	appRepo store.AppRepository,
	dbRepo store.DatabaseRepository,
	storageRepo store.StorageRepository,
	authRepo store.AppAuthRepository,
	proPriceID, teamPriceID, baseDomain string,
) *BillingHandler {
	return &BillingHandler{
		stripe:      stripe,
		billingRepo: billingRepo,
		planRepo:    planRepo,
		appRepo:     appRepo,
		dbRepo:      dbRepo,
		storageRepo: storageRepo,
		authRepo:    authRepo,
		proPriceID:  proPriceID,
		teamPriceID: teamPriceID,
		baseDomain:  baseDomain,
	}
}

// GetBillingStatus returns the current user's plan + subscription + usage.
// GET /api/v1/billing
func (h *BillingHandler) GetBillingStatus(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	usage := h.calculateUsage(c, userID)
	priceCents, _ := PriceForTier(plan.Tier, h.proPriceID, h.teamPriceID)

	resp := dto.BillingStatusResponse{
		Tier:          string(plan.Tier),
		BillingStatus: "none",
		PriceCents:    priceCents,
		Currency:      "eur",
		Limits:        plan.Limits,
		Usage:         usage,
		StripeEnabled: h.stripe != nil,
	}

	// Enrich with subscription data if available
	sub, err := h.billingRepo.GetSubscriptionByUser(c.Context(), userID)
	if err == nil && sub != nil {
		resp.BillingStatus = string(sub.Status)
		resp.CancelAtPeriodEnd = sub.CancelAtPeriodEnd
		periodEnd := sub.CurrentPeriodEnd.Format("2006-01-02T15:04:05Z")
		resp.PeriodEnd = &periodEnd
	}

	return c.JSON(resp)
}

// CreateCheckoutSession creates a Stripe Checkout session and returns the URL.
// POST /api/v1/billing/checkout
func (h *BillingHandler) CreateCheckoutSession(c *fiber.Ctx) error {
	if h.stripe == nil {
		return fiber.NewError(fiber.StatusBadRequest, "Stripe billing is not enabled")
	}

	userID, _ := c.Locals("user_id").(string)
	userEmail, _ := c.Locals("user_email").(string)

	var input dto.CreateCheckoutInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	tier := entities.PlanTier(input.Tier)
	_, priceID := PriceForTier(tier, h.proPriceID, h.teamPriceID)
	if priceID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tier or tier not available for checkout")
	}

	// Check if already on this tier or higher
	plan, _ := h.planRepo.GetUserPlan(c.Context(), userID)
	if plan != nil && plan.Tier == tier {
		return fiber.NewError(fiber.StatusBadRequest, "you are already on the "+input.Tier+" plan")
	}

	customerID, _ := h.billingRepo.GetStripeCustomerID(c.Context(), userID)

	successURL := "https://" + h.baseDomain + "/billing?success=true"
	cancelURL := "https://" + h.baseDomain + "/billing?canceled=true"

	result, err := h.stripe.CreateCheckoutSession(c.Context(), stripeClient.CheckoutParams{
		CustomerID: customerID,
		PriceID:    priceID,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
		UserEmail:  userEmail,
		Metadata: map[string]string{
			"user_id": userID,
			"tier":    input.Tier,
		},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create checkout session")
	}

	return c.JSON(dto.CheckoutResponse{
		CheckoutURL: result.URL,
		SessionID:   result.SessionID,
	})
}

// CreatePortalSession creates a Stripe Customer Portal session.
// POST /api/v1/billing/portal
func (h *BillingHandler) CreatePortalSession(c *fiber.Ctx) error {
	if h.stripe == nil {
		return fiber.NewError(fiber.StatusBadRequest, "Stripe billing is not enabled")
	}

	userID, _ := c.Locals("user_id").(string)

	customerID, _ := h.billingRepo.GetStripeCustomerID(c.Context(), userID)
	if customerID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "no Stripe customer found; subscribe first")
	}

	returnURL := "https://" + h.baseDomain + "/billing"

	result, err := h.stripe.CreatePortalSession(c.Context(), customerID, returnURL)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create portal session")
	}

	return c.JSON(dto.PortalResponse{PortalURL: result.URL})
}

// CancelSubscription cancels the user's subscription (at period end by default).
// POST /api/v1/billing/cancel
func (h *BillingHandler) CancelSubscription(c *fiber.Ctx) error {
	if h.stripe == nil {
		return fiber.NewError(fiber.StatusBadRequest, "Stripe billing is not enabled")
	}

	userID, _ := c.Locals("user_id").(string)

	var input dto.CancelSubscriptionInput
	if err := c.BodyParser(&input); err != nil {
		// Default to cancel at period end
		input.Immediate = false
	}

	sub, err := h.billingRepo.GetSubscriptionByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "no active subscription found")
	}

	if err := h.stripe.CancelSubscription(c.Context(), sub.StripeSubscriptionID, !input.Immediate); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to cancel subscription")
	}

	return c.JSON(fiber.Map{"status": "canceling", "cancel_at_period_end": !input.Immediate})
}

// ListInvoices returns the user's invoice history.
// GET /api/v1/billing/invoices
func (h *BillingHandler) ListInvoices(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	invoices, err := h.billingRepo.ListInvoicesByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	items := make([]dto.InvoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		items = append(items, dto.InvoiceResponse{
			ID:          inv.ID,
			AmountCents: inv.AmountCents,
			Currency:    inv.Currency,
			Status:      string(inv.Status),
			InvoiceURL:  inv.InvoiceURL,
			InvoicePDF:  inv.InvoicePDF,
			PeriodStart: inv.PeriodStart.Format("2006-01-02T15:04:05Z"),
			PeriodEnd:   inv.PeriodEnd.Format("2006-01-02T15:04:05Z"),
			CreatedAt:   inv.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

// GetAdminBillingOverview returns admin billing metrics.
// GET /api/v1/admin/billing/overview
func (h *BillingHandler) GetAdminBillingOverview(c *fiber.Ctx) error {
	overview, err := h.billingRepo.GetBillingOverview(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(dto.AdminBillingOverviewResponse{
		MRRCents:            overview.MRRCents,
		ActiveSubscriptions: overview.ActiveSubscriptions,
		PastDueCount:        overview.PastDueCount,
		CanceledThisMonth:   overview.CanceledThisMonth,
		ChurnRatePercent:    overview.ChurnRatePercent,
	})
}

func (h *BillingHandler) calculateUsage(c *fiber.Ctx, userID string) dto.PlanUsage {
	appCount, _ := h.appRepo.CountAppsByUser(c.Context(), userID)
	dbCount, _ := h.dbRepo.CountDatabasesByUser(c.Context(), userID)
	bucketCount, _ := h.storageRepo.CountBucketsByUser(c.Context(), userID)

	return dto.PlanUsage{
		Apps:      appCount,
		Databases: dbCount,
		Buckets:   bucketCount,
	}
}
