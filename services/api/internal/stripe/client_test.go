package stripe

import (
	"context"
	"testing"

	gostripe "github.com/stripe/stripe-go/v82"
)

// MockStripeAPI is a test double for StripeAPI.
type MockStripeAPI struct {
	CreateCheckoutSessionFn func(ctx context.Context, params CheckoutParams) (*CheckoutResult, error)
	CreatePortalSessionFn   func(ctx context.Context, customerID, returnURL string) (*PortalResult, error)
	CancelSubscriptionFn    func(ctx context.Context, subID string, atPeriodEnd bool) error
	GetSubscriptionFn       func(ctx context.Context, subID string) (*SubscriptionResult, error)
	ConstructWebhookEventFn func(payload []byte, signature string) (*gostripe.Event, error)
}

func (m *MockStripeAPI) CreateCheckoutSession(ctx context.Context, params CheckoutParams) (*CheckoutResult, error) {
	if m.CreateCheckoutSessionFn != nil {
		return m.CreateCheckoutSessionFn(ctx, params)
	}
	return &CheckoutResult{SessionID: "cs_test", URL: "https://checkout.stripe.com/test"}, nil
}

func (m *MockStripeAPI) CreatePortalSession(ctx context.Context, customerID, returnURL string) (*PortalResult, error) {
	if m.CreatePortalSessionFn != nil {
		return m.CreatePortalSessionFn(ctx, customerID, returnURL)
	}
	return &PortalResult{URL: "https://billing.stripe.com/test"}, nil
}

func (m *MockStripeAPI) CancelSubscription(ctx context.Context, subID string, atPeriodEnd bool) error {
	if m.CancelSubscriptionFn != nil {
		return m.CancelSubscriptionFn(ctx, subID, atPeriodEnd)
	}
	return nil
}

func (m *MockStripeAPI) GetSubscription(ctx context.Context, subID string) (*SubscriptionResult, error) {
	if m.GetSubscriptionFn != nil {
		return m.GetSubscriptionFn(ctx, subID)
	}
	return &SubscriptionResult{
		ID:         subID,
		CustomerID: "cus_test",
		PriceID:    "price_test",
		Status:     "active",
	}, nil
}

func (m *MockStripeAPI) ConstructWebhookEvent(payload []byte, signature string) (*gostripe.Event, error) {
	if m.ConstructWebhookEventFn != nil {
		return m.ConstructWebhookEventFn(payload, signature)
	}
	return &gostripe.Event{Type: "checkout.session.completed"}, nil
}

// Compile-time interface check
var _ StripeAPI = (*MockStripeAPI)(nil)
var _ StripeAPI = (*Client)(nil)

func TestMockStripeAPI_Defaults(t *testing.T) {
	mock := &MockStripeAPI{}
	ctx := context.Background()

	result, err := mock.CreateCheckoutSession(ctx, CheckoutParams{PriceID: "price_pro"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SessionID != "cs_test" {
		t.Errorf("expected cs_test, got %s", result.SessionID)
	}

	portal, err := mock.CreatePortalSession(ctx, "cus_123", "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if portal.URL == "" {
		t.Error("expected non-empty portal URL")
	}

	if err := mock.CancelSubscription(ctx, "sub_123", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, err := mock.GetSubscription(ctx, "sub_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.Status != "active" {
		t.Errorf("expected active, got %s", sub.Status)
	}

	event, err := mock.ConstructWebhookEvent([]byte("{}"), "sig")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Type != "checkout.session.completed" {
		t.Errorf("expected checkout.session.completed, got %s", event.Type)
	}
}
