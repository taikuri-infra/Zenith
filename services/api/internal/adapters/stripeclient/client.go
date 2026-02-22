package stripeclient

import (
	"context"
	"fmt"

	gostripe "github.com/stripe/stripe-go/v82"
	portalsession "github.com/stripe/stripe-go/v82/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"
)

// CheckoutParams holds the parameters for creating a checkout session.
type CheckoutParams struct {
	CustomerID string
	PriceID    string
	SuccessURL string
	CancelURL  string
	UserEmail  string
	Metadata   map[string]string
}

// CheckoutResult is the result of creating a checkout session.
type CheckoutResult struct {
	SessionID  string
	URL        string
}

// PortalResult is the result of creating a portal session.
type PortalResult struct {
	URL string
}

// SubscriptionResult wraps relevant fields from a Stripe subscription.
type SubscriptionResult struct {
	ID                string
	CustomerID        string
	PriceID           string
	Status            string
	CurrentPeriodEnd  int64
	CancelAtPeriodEnd bool
}

// StripeAPI defines the operations the billing system needs from Stripe.
type StripeAPI interface {
	CreateCheckoutSession(ctx context.Context, params CheckoutParams) (*CheckoutResult, error)
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (*PortalResult, error)
	CancelSubscription(ctx context.Context, subID string, atPeriodEnd bool) error
	GetSubscription(ctx context.Context, subID string) (*SubscriptionResult, error)
	ConstructWebhookEvent(payload []byte, signature string) (*gostripe.Event, error)
}

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

func (c *Client) CreateCheckoutSession(_ context.Context, params CheckoutParams) (*CheckoutResult, error) {
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

	return &CheckoutResult{
		SessionID: sess.ID,
		URL:       sess.URL,
	}, nil
}

func (c *Client) CreatePortalSession(_ context.Context, customerID, returnURL string) (*PortalResult, error) {
	p := &gostripe.BillingPortalSessionParams{
		Customer:  gostripe.String(customerID),
		ReturnURL: gostripe.String(returnURL),
	}

	sess, err := portalsession.New(p)
	if err != nil {
		return nil, fmt.Errorf("stripe: create portal session: %w", err)
	}

	return &PortalResult{URL: sess.URL}, nil
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

func (c *Client) GetSubscription(_ context.Context, subID string) (*SubscriptionResult, error) {
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

	return &SubscriptionResult{
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
