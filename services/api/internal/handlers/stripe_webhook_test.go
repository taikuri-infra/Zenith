package handlers_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	gostripe "github.com/stripe/stripe-go/v82"
)

// mockStripeAPI is a test double for the StripeAPI interface.
type mockStripeAPI struct {
	shouldFail bool
	event      *gostripe.Event
}

func (m *mockStripeAPI) CreateCheckoutSession(_ context.Context, _ ports.CheckoutParams) (*ports.CheckoutResult, error) {
	return &ports.CheckoutResult{SessionID: "cs_test_123", URL: "https://checkout.stripe.com/test"}, nil
}

func (m *mockStripeAPI) CreatePortalSession(_ context.Context, _, _ string) (*ports.PortalResult, error) {
	return &ports.PortalResult{URL: "https://billing.stripe.com/session/test"}, nil
}

func (m *mockStripeAPI) CancelSubscription(_ context.Context, _ string, _ bool) error {
	return nil
}

func (m *mockStripeAPI) GetSubscription(_ context.Context, _ string) (*ports.SubscriptionResult, error) {
	return &ports.SubscriptionResult{ID: "sub_test", Status: "active"}, nil
}

func (m *mockStripeAPI) ConstructWebhookEvent(payload []byte, signature string) (*gostripe.Event, error) {
	if m.shouldFail {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid signature")
	}
	if m.event != nil {
		return m.event, nil
	}
	return &gostripe.Event{
		Type: "unknown.event.type",
	}, nil
}

func TestStripeWebhookInvalidSignature(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	mock := &mockStripeAPI{shouldFail: true}
	handler := handlers.NewStripeWebhookHandler(nil, mock)

	app.Post("/api/v1/webhooks/stripe", handler.HandleEvent)

	req := httptest.NewRequest("POST", "/api/v1/webhooks/stripe", nil)
	req.Header.Set("Stripe-Signature", "bad-sig")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for invalid signature, got %d", resp.StatusCode)
	}
}

func TestStripeWebhookUnknownEventType(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	mock := &mockStripeAPI{
		event: &gostripe.Event{
			Type: "some.unknown.event",
		},
	}
	handler := handlers.NewStripeWebhookHandler(nil, mock)

	app.Post("/api/v1/webhooks/stripe", handler.HandleEvent)

	req := httptest.NewRequest("POST", "/api/v1/webhooks/stripe", nil)
	req.Header.Set("Stripe-Signature", "valid-sig")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 for unhandled event type, got %d", resp.StatusCode)
	}
}
