package handlers

import (
	"encoding/json"
	"log"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/store"
	stripeClient "github.com/dotechhq/zenith/services/api/internal/stripe"
	"github.com/gofiber/fiber/v2"
	gostripe "github.com/stripe/stripe-go/v82"
)

// StripeWebhookHandler processes incoming Stripe webhook events.
type StripeWebhookHandler struct {
	stripe      stripeClient.StripeAPI
	billingRepo store.BillingRepository
	planRepo    store.UserPlanRepository
	proPriceID  string
	teamPriceID string
}

// NewStripeWebhookHandler creates a new StripeWebhookHandler.
func NewStripeWebhookHandler(
	stripe stripeClient.StripeAPI,
	billingRepo store.BillingRepository,
	planRepo store.UserPlanRepository,
	proPriceID, teamPriceID string,
) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		stripe:      stripe,
		billingRepo: billingRepo,
		planRepo:    planRepo,
		proPriceID:  proPriceID,
		teamPriceID: teamPriceID,
	}
}

// HandleEvent processes a Stripe webhook event.
// POST /api/v1/webhooks/stripe
func (h *StripeWebhookHandler) HandleEvent(c *fiber.Ctx) error {
	payload := c.Body()
	signature := c.Get("Stripe-Signature")

	event, err := h.stripe.ConstructWebhookEvent(payload, signature)
	if err != nil {
		log.Printf("[stripe-webhook] signature verification failed: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "invalid webhook signature")
	}

	log.Printf("[stripe-webhook] received event: %s", event.Type)

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
		log.Printf("[stripe-webhook] unhandled event type: %s", event.Type)
	}

	return c.JSON(fiber.Map{"received": true})
}

func (h *StripeWebhookHandler) handleCheckoutCompleted(c *fiber.Ctx, event *gostripe.Event) error {
	var session gostripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		log.Printf("[stripe-webhook] failed to parse checkout session: %v", err)
		return c.JSON(fiber.Map{"received": true})
	}

	userID := session.Metadata["user_id"]
	tierStr := session.Metadata["tier"]
	if userID == "" || tierStr == "" {
		log.Printf("[stripe-webhook] checkout missing metadata: user_id=%s tier=%s", userID, tierStr)
		return c.JSON(fiber.Map{"received": true})
	}

	tier := entities.PlanTier(tierStr)
	customerID := session.Customer.ID
	subscriptionID := session.Subscription.ID

	// Store customer mapping
	if err := h.billingRepo.SetStripeCustomerID(c.Context(), userID, customerID); err != nil {
		log.Printf("[stripe-webhook] failed to set customer ID: %v", err)
	}

	// Determine price ID from tier
	_, priceID := PriceForTier(tier, h.proPriceID, h.teamPriceID)

	// Create subscription record
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
	if err := h.billingRepo.CreateSubscription(c.Context(), sub); err != nil {
		log.Printf("[stripe-webhook] failed to create subscription: %v", err)
	}

	// Upgrade user plan
	if _, err := h.planRepo.SetUserPlan(c.Context(), userID, tier); err != nil {
		log.Printf("[stripe-webhook] failed to set user plan: %v", err)
	}

	log.Printf("[stripe-webhook] user %s upgraded to %s (sub=%s)", userID, tier, subscriptionID)
	return c.JSON(fiber.Map{"received": true})
}

func (h *StripeWebhookHandler) handleInvoicePaid(c *fiber.Ctx, event *gostripe.Event) error {
	var invoice gostripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		log.Printf("[stripe-webhook] failed to parse invoice: %v", err)
		return c.JSON(fiber.Map{"received": true})
	}

	customerID := invoice.Customer.ID

	userID, err := h.billingRepo.GetUserByStripeCustomerID(c.Context(), customerID)
	if err != nil {
		log.Printf("[stripe-webhook] no user for customer %s: %v", customerID, err)
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

	if err := h.billingRepo.UpsertInvoice(c.Context(), inv); err != nil {
		log.Printf("[stripe-webhook] failed to upsert invoice: %v", err)
	}

	log.Printf("[stripe-webhook] invoice paid for user %s: %d %s", userID, inv.AmountCents, inv.Currency)
	return c.JSON(fiber.Map{"received": true})
}

func (h *StripeWebhookHandler) handleInvoicePaymentFailed(c *fiber.Ctx, event *gostripe.Event) error {
	var invoice gostripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		log.Printf("[stripe-webhook] failed to parse invoice: %v", err)
		return c.JSON(fiber.Map{"received": true})
	}

	subID := invoiceSubscriptionID(&invoice)
	if subID != "" {
		if err := h.billingRepo.UpdateSubscriptionStatus(c.Context(), subID, entities.SubscriptionPastDue); err != nil {
			log.Printf("[stripe-webhook] failed to mark subscription past_due: %v", err)
		}
		log.Printf("[stripe-webhook] subscription %s marked past_due (payment failed)", subID)
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
		log.Printf("[stripe-webhook] failed to parse subscription: %v", err)
		return c.JSON(fiber.Map{"received": true})
	}

	status := mapStripeStatus(sub.Status)
	if err := h.billingRepo.UpdateSubscriptionStatus(c.Context(), sub.ID, status); err != nil {
		log.Printf("[stripe-webhook] failed to update subscription status: %v", err)
	}

	// Check if price changed (plan change via Stripe portal)
	if len(sub.Items.Data) > 0 {
		newPriceID := sub.Items.Data[0].Price.ID
		tier := h.tierFromPriceID(newPriceID)
		if tier != "" {
			if err := h.billingRepo.UpdateSubscriptionTier(c.Context(), sub.ID, tier, newPriceID); err != nil {
				log.Printf("[stripe-webhook] failed to update subscription tier: %v", err)
			}

			// Also update user plan
			existing, err := h.billingRepo.GetSubscriptionByStripeID(c.Context(), sub.ID)
			if err == nil && existing != nil {
				if _, err := h.planRepo.SetUserPlan(c.Context(), existing.UserID, tier); err != nil {
					log.Printf("[stripe-webhook] failed to update user plan: %v", err)
				}
			}
		}
	}

	log.Printf("[stripe-webhook] subscription %s updated: status=%s", sub.ID, status)
	return c.JSON(fiber.Map{"received": true})
}

func (h *StripeWebhookHandler) handleSubscriptionDeleted(c *fiber.Ctx, event *gostripe.Event) error {
	var sub gostripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		log.Printf("[stripe-webhook] failed to parse subscription: %v", err)
		return c.JSON(fiber.Map{"received": true})
	}

	if err := h.billingRepo.UpdateSubscriptionStatus(c.Context(), sub.ID, entities.SubscriptionCanceled); err != nil {
		log.Printf("[stripe-webhook] failed to mark subscription canceled: %v", err)
	}

	// Downgrade user to Free
	existing, err := h.billingRepo.GetSubscriptionByStripeID(c.Context(), sub.ID)
	if err == nil && existing != nil {
		if _, err := h.planRepo.SetUserPlan(c.Context(), existing.UserID, entities.PlanFree); err != nil {
			log.Printf("[stripe-webhook] failed to downgrade user to free: %v", err)
		}
		log.Printf("[stripe-webhook] user %s downgraded to free (subscription deleted)", existing.UserID)
	}

	return c.JSON(fiber.Map{"received": true})
}

func (h *StripeWebhookHandler) tierFromPriceID(priceID string) entities.PlanTier {
	switch priceID {
	case h.proPriceID:
		return entities.PlanPro
	case h.teamPriceID:
		return entities.PlanTeam
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
