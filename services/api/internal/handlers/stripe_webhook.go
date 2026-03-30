package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	stripeClient "github.com/dotechhq/zenith/services/api/internal/adapters/stripeclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
	gostripe "github.com/stripe/stripe-go/v82"
)

// PlanChangeFunc is a callback to trigger the PlanOrchestrator Temporal workflow.
type PlanChangeFunc func(ctx context.Context, userID, email string, oldTier, newTier entities.PlanTier, stripeSub, stripeCust string)

// StripeWebhookHandler processes incoming Stripe webhook events.
// This handler lives in the handler layer and holds a direct reference to the
// StripeAPI adapter (for ConstructWebhookEvent which returns a concrete Stripe type).
type StripeWebhookHandler struct {
	billingSvc      *services.BillingService
	stripeAPI       stripeClient.StripeAPI
	eventBus        ports.EventBus
	onPlanChange    PlanChangeFunc
}

// NewStripeWebhookHandler creates a new StripeWebhookHandler.
func NewStripeWebhookHandler(billingSvc *services.BillingService, stripeAPI stripeClient.StripeAPI) *StripeWebhookHandler {
	return &StripeWebhookHandler{billingSvc: billingSvc, stripeAPI: stripeAPI}
}

// SetEventBus configures the NATS event bus for billing event publishing.
func (h *StripeWebhookHandler) SetEventBus(bus ports.EventBus) {
	h.eventBus = bus
}

// SetOnPlanChange registers a callback to trigger the PlanOrchestrator workflow.
func (h *StripeWebhookHandler) SetOnPlanChange(fn PlanChangeFunc) {
	h.onPlanChange = fn
}

// publishEvent publishes a billing event to the event bus (best-effort).
func (h *StripeWebhookHandler) publishEvent(subject entities.EventSubject, userID string, data map[string]interface{}) {
	if h.eventBus == nil {
		return
	}
	evt := &entities.PlatformEvent{
		Subject:   subject,
		UserID:    userID,
		Timestamp: time.Now(),
		Data:      data,
	}
	if err := h.eventBus.Publish(context.Background(), evt); err != nil {
		slog.Error("failed to publish billing event", "subject", subject, "error", err)
	}
}

// HandleEvent processes a Stripe webhook event.
// POST /api/v1/webhooks/stripe
func (h *StripeWebhookHandler) HandleEvent(c *fiber.Ctx) error {
	payload := c.Body()
	signature := c.Get("Stripe-Signature")

	event, err := h.stripeAPI.ConstructWebhookEvent(payload, signature)
	if err != nil {
		slog.Warn("stripe webhook signature verification failed", "error", err)
		return fiber.NewError(fiber.StatusBadRequest, "invalid webhook signature")
	}

	slog.Info("stripe webhook received", "type", event.Type)

	switch event.Type {
	case "checkout.session.completed":
		return h.handleCheckoutCompleted(c, event)
	case "invoice.paid":
		return h.handleInvoicePaid(c, event)
	case "invoice.payment_failed":
		return h.handleInvoicePaymentFailed(c, event)
	case "customer.subscription.updated":
		return h.handleSubscriptionUpdated(c, event)
	case "customer.subscription.deleted":
		return h.handleSubscriptionDeleted(c, event)
	default:
		slog.Info("unhandled stripe event type", "type", event.Type)
	}

	return c.JSON(fiber.Map{"received": true})
}

func (h *StripeWebhookHandler) handleCheckoutCompleted(c *fiber.Ctx, event *gostripe.Event) error {
	var session gostripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		slog.Error("failed to parse checkout session", "error", err)
		return c.JSON(fiber.Map{"received": true})
	}

	userID := session.Metadata["user_id"]
	tierStr := session.Metadata["tier"]
	if userID == "" || tierStr == "" {
		slog.Warn("checkout missing metadata", "user_id", userID, "tier", tierStr)
		return c.JSON(fiber.Map{"received": true})
	}

	tier := entities.PlanTier(tierStr)
	customerID := session.Customer.ID
	subscriptionID := session.Subscription.ID

	billingRepo := h.billingSvc.BillingRepo()
	planRepo := h.billingSvc.PlanRepo()

	if err := billingRepo.SetStripeCustomerID(c.Context(), userID, customerID); err != nil {
		slog.Error("failed to set stripe customer ID", "user_id", userID, "error", err)
	}

	_, priceID := services.PriceForTier(tier, h.billingSvc.ProPriceID(), h.billingSvc.TeamPriceID(), h.billingSvc.BusinessPriceID())

	sub := &entities.Subscription{
		ID:                   "sub_" + subscriptionID,
		UserID:               userID,
		StripeSubscriptionID: subscriptionID,
		StripeCustomerID:     customerID,
		StripePriceID:        priceID,
		Tier:                 tier,
		Status:               entities.SubscriptionActive,
		CurrentPeriodStart:   time.Now(),
		CurrentPeriodEnd:     time.Now().Add(30 * 24 * time.Hour),
	}
	if err := billingRepo.CreateSubscription(c.Context(), sub); err != nil {
		slog.Error("failed to create subscription", "user_id", userID, "error", err)
	}

	// Capture old tier BEFORE saving the new plan
	var oldTier entities.PlanTier
	if h.onPlanChange != nil {
		if oldPlan, pErr := planRepo.GetUserPlan(c.Context(), userID); pErr == nil {
			oldTier = oldPlan.Tier
		} else {
			oldTier = entities.PlanFree
		}
	}

	if _, err := planRepo.SetUserPlan(c.Context(), userID, tier); err != nil {
		slog.Error("failed to set user plan", "user_id", userID, "tier", tier, "error", err)
	}

	// Provision infrastructure for the new tier (S3 bucket, etc.)
	h.billingSvc.ProvisionUpgradeResources(c.Context(), userID, tier)

	// Trigger PlanOrchestrator workflow (async via Temporal if configured)
	if h.onPlanChange != nil {
		h.onPlanChange(c.Context(), userID, "", oldTier, tier, subscriptionID, customerID)
	}

	slog.Info("user upgraded via checkout", "user_id", userID, "tier", tier, "subscription_id", subscriptionID)

	h.publishEvent(entities.EventBillingCheckoutCompleted, userID, map[string]interface{}{
		"tier":            string(tier),
		"subscription_id": subscriptionID,
		"customer_id":     customerID,
	})

	return c.JSON(fiber.Map{"received": true})
}

func (h *StripeWebhookHandler) handleInvoicePaid(c *fiber.Ctx, event *gostripe.Event) error {
	var invoice gostripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		slog.Error("failed to parse invoice", "error", err)
		return c.JSON(fiber.Map{"received": true})
	}

	billingRepo := h.billingSvc.BillingRepo()
	customerID := invoice.Customer.ID

	userID, err := billingRepo.GetUserByStripeCustomerID(c.Context(), customerID)
	if err != nil {
		slog.Warn("no user found for stripe customer", "customer_id", customerID, "error", err)
		return c.JSON(fiber.Map{"received": true})
	}

	inv := &entities.Invoice{
		ID:              "inv_" + invoice.ID,
		UserID:          userID,
		StripeInvoiceID: invoice.ID,
		AmountCents:     int(invoice.AmountPaid),
		Currency:        string(invoice.Currency),
		Status:          entities.InvoicePaid,
		InvoiceURL:      invoice.HostedInvoiceURL,
		InvoicePDF:      invoice.InvoicePDF,
		PeriodStart:     time.Unix(invoice.PeriodStart, 0),
		PeriodEnd:       time.Unix(invoice.PeriodEnd, 0),
	}

	if err := billingRepo.UpsertInvoice(c.Context(), inv); err != nil {
		slog.Error("failed to upsert invoice", "user_id", userID, "error", err)
	}

	slog.Info("invoice paid", "user_id", userID, "amount_cents", inv.AmountCents, "currency", inv.Currency)

	h.publishEvent(entities.EventBillingInvoicePaid, userID, map[string]interface{}{
		"invoice_id":  invoice.ID,
		"amount":      inv.AmountCents,
		"currency":    inv.Currency,
	})

	return c.JSON(fiber.Map{"received": true})
}

func (h *StripeWebhookHandler) handleInvoicePaymentFailed(c *fiber.Ctx, event *gostripe.Event) error {
	var invoice gostripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		slog.Error("failed to parse invoice", "error", err)
		return c.JSON(fiber.Map{"received": true})
	}

	subID := invoiceSubscriptionID(&invoice)
	if subID != "" {
		if err := h.billingSvc.BillingRepo().UpdateSubscriptionStatus(c.Context(), subID, entities.SubscriptionPastDue); err != nil {
			slog.Error("failed to mark subscription past_due", "subscription_id", subID, "error", err)
		}
		slog.Warn("subscription marked past_due", "subscription_id", subID)

		// Look up user for the event
		customerID := invoice.Customer.ID
		userID, _ := h.billingSvc.BillingRepo().GetUserByStripeCustomerID(c.Context(), customerID)
		if userID != "" {
			h.publishEvent(entities.EventBillingPaymentFailed, userID, map[string]interface{}{
				"subscription_id": subID,
				"invoice_id":      invoice.ID,
			})
		}
	}

	return c.JSON(fiber.Map{"received": true})
}

// invoiceSubscriptionID extracts the subscription ID from an invoice's parent.
func invoiceSubscriptionID(inv *gostripe.Invoice) string {
	if inv.Parent != nil &&
		inv.Parent.SubscriptionDetails != nil &&
		inv.Parent.SubscriptionDetails.Subscription != nil {
		return inv.Parent.SubscriptionDetails.Subscription.ID
	}
	return ""
}

func (h *StripeWebhookHandler) handleSubscriptionUpdated(c *fiber.Ctx, event *gostripe.Event) error {
	var sub gostripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		slog.Error("failed to parse subscription", "error", err)
		return c.JSON(fiber.Map{"received": true})
	}

	billingRepo := h.billingSvc.BillingRepo()
	planRepo := h.billingSvc.PlanRepo()

	status := mapStripeStatus(sub.Status)
	if err := billingRepo.UpdateSubscriptionStatus(c.Context(), sub.ID, status); err != nil {
		slog.Error("failed to update subscription status", "subscription_id", sub.ID, "error", err)
	}

	if len(sub.Items.Data) > 0 {
		newPriceID := sub.Items.Data[0].Price.ID
		tier := tierFromPriceID(newPriceID, h.billingSvc.ProPriceID(), h.billingSvc.TeamPriceID(), h.billingSvc.BusinessPriceID())
		if tier != "" {
			if err := billingRepo.UpdateSubscriptionTier(c.Context(), sub.ID, tier, newPriceID); err != nil {
				slog.Error("failed to update subscription tier", "subscription_id", sub.ID, "error", err)
			}

			existing, err := billingRepo.GetSubscriptionByStripeID(c.Context(), sub.ID)
			if err == nil && existing != nil {
				if _, err := planRepo.SetUserPlan(c.Context(), existing.UserID, tier); err != nil {
					slog.Error("failed to update user plan", "user_id", existing.UserID, "error", err)
				}
			}
		}
	}

	slog.Info("subscription updated", "subscription_id", sub.ID, "status", status)

	// Publish event
	existing2, err := billingRepo.GetSubscriptionByStripeID(c.Context(), sub.ID)
	if err == nil && existing2 != nil {
		h.publishEvent(entities.EventBillingSubscriptionUpdated, existing2.UserID, map[string]interface{}{
			"subscription_id": sub.ID,
			"status":          string(status),
		})
	}

	return c.JSON(fiber.Map{"received": true})
}

func (h *StripeWebhookHandler) handleSubscriptionDeleted(c *fiber.Ctx, event *gostripe.Event) error {
	var sub gostripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		slog.Error("failed to parse subscription", "error", err)
		return c.JSON(fiber.Map{"received": true})
	}

	billingRepo := h.billingSvc.BillingRepo()
	planRepo := h.billingSvc.PlanRepo()

	if err := billingRepo.UpdateSubscriptionStatus(c.Context(), sub.ID, entities.SubscriptionCanceled); err != nil {
		slog.Error("failed to mark subscription canceled", "subscription_id", sub.ID, "error", err)
	}

	existing, err := billingRepo.GetSubscriptionByStripeID(c.Context(), sub.ID)
	if err == nil && existing != nil {
		if _, err := planRepo.SetUserPlan(c.Context(), existing.UserID, entities.PlanFree); err != nil {
			slog.Error("failed to downgrade user to free", "user_id", existing.UserID, "error", err)
		}
		slog.Info("user downgraded to free", "user_id", existing.UserID, "reason", "subscription_deleted")

		h.publishEvent(entities.EventBillingSubscriptionCanceled, existing.UserID, map[string]interface{}{
			"subscription_id": sub.ID,
			"previous_tier":   string(existing.Tier),
		})
	}

	return c.JSON(fiber.Map{"received": true})
}

func tierFromPriceID(priceID, proPriceID, teamPriceID, businessPriceID string) entities.PlanTier {
	switch priceID {
	case proPriceID:
		return entities.PlanPro
	case teamPriceID:
		return entities.PlanTeam
	case businessPriceID:
		return entities.PlanBusiness
	default:
		return ""
	}
}

func mapStripeStatus(status gostripe.SubscriptionStatus) entities.SubscriptionStatus {
	switch status {
	case gostripe.SubscriptionStatusActive:
		return entities.SubscriptionActive
	case gostripe.SubscriptionStatusPastDue:
		return entities.SubscriptionPastDue
	case gostripe.SubscriptionStatusCanceled:
		return entities.SubscriptionCanceled
	case gostripe.SubscriptionStatusIncomplete:
		return entities.SubscriptionIncomplete
	case gostripe.SubscriptionStatusTrialing:
		return entities.SubscriptionTrialing
	case gostripe.SubscriptionStatusUnpaid:
		return entities.SubscriptionUnpaid
	default:
		return entities.SubscriptionActive
	}
}
