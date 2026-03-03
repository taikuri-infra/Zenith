package stripeclient

import (
	"context"
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/ports"
	gostripe "github.com/stripe/stripe-go/v82"
	portalsession "github.com/stripe/stripe-go/v82/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"
)

// StripeAPI extends ports.PaymentGateway with Stripe-specific webhook verification.
// The handler layer uses this interface; the service layer uses ports.PaymentGateway.
type StripeAPI interface {
	ports.PaymentGateway
	ConstructWebhookEvent(payload []byte, signature string) (*gostripe.Event, error)
}

// Compile-time check: Client implements both StripeAPI and ports.PaymentGateway.
var _ StripeAPI = (*Client)(nil)

// Client wraps the stripe-go SDK.
type Client struct {
	webhookSecret string
}

// NewClient creates a Stripe client. The secretKey configures the global
// stripe backend; webhookSecret is stored for signature verification.
func NewClient(secretKey, webhookSecret string) *Client {
	gostripe.Key = secretKey
	return &Client{
		webhookSecret: webhookSecret,
	}
}

func (c *Client) CreateCheckoutSession(_ context.Context, params ports.CheckoutParams) (*ports.CheckoutResult, error) {
	p := &gostripe.CheckoutSessionParams{
		Mode: gostripe.String(string(gostripe.CheckoutSessionModeSubscription)),
		LineItems: []*gostripe.CheckoutSessionLineItemParams{
			{
				Price:    gostripe.String(params.PriceID),
				Quantity: gostripe.Int64(1),
			},
		},
		SuccessURL: gostripe.String(params.SuccessURL),
		CancelURL:  gostripe.String(params.CancelURL),
	}

	if params.CustomerID != "" {
		p.Customer = gostripe.String(params.CustomerID)
	} else if params.UserEmail != "" {
		p.CustomerEmail = gostripe.String(params.UserEmail)
	}

	if len(params.Metadata) > 0 {
		p.Metadata = params.Metadata
	}

	sess, err := checkoutsession.New(p)
	if err != nil {
		return nil, fmt.Errorf("stripe: create checkout session: %w", err)
	}

	return &ports.CheckoutResult{
		SessionID: sess.ID,
		URL:       sess.URL,
	}, nil
}

func (c *Client) CreatePortalSession(_ context.Context, customerID, returnURL string) (*ports.PortalResult, error) {
	p := &gostripe.BillingPortalSessionParams{
		Customer:  gostripe.String(customerID),
		ReturnURL: gostripe.String(returnURL),
	}

	sess, err := portalsession.New(p)
	if err != nil {
		return nil, fmt.Errorf("stripe: create portal session: %w", err)
	}

	return &ports.PortalResult{URL: sess.URL}, nil
}

func (c *Client) CancelSubscription(_ context.Context, subID string, atPeriodEnd bool) error {
	if atPeriodEnd {
		_, err := subscription.Update(subID, &gostripe.SubscriptionParams{
			CancelAtPeriodEnd: gostripe.Bool(true),
		})
		if err != nil {
			return fmt.Errorf("stripe: cancel subscription at period end: %w", err)
		}
		return nil
	}

	_, err := subscription.Cancel(subID, nil)
	if err != nil {
		return fmt.Errorf("stripe: cancel subscription immediately: %w", err)
	}
	return nil
}

func (c *Client) GetSubscription(_ context.Context, subID string) (*ports.SubscriptionResult, error) {
	sub, err := subscription.Get(subID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe: get subscription: %w", err)
	}

	priceID := ""
	var periodEnd int64
	if len(sub.Items.Data) > 0 {
		priceID = sub.Items.Data[0].Price.ID
		periodEnd = sub.Items.Data[0].CurrentPeriodEnd
	}

	return &ports.SubscriptionResult{
		ID:                sub.ID,
		CustomerID:        sub.Customer.ID,
		PriceID:           priceID,
		Status:            string(sub.Status),
		CurrentPeriodEnd:  periodEnd,
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	}, nil
}

func (c *Client) ConstructWebhookEvent(payload []byte, signature string) (*gostripe.Event, error) {
	event, err := webhook.ConstructEvent(payload, signature, c.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("stripe: verify webhook signature: %w", err)
	}
	return &event, nil
}
