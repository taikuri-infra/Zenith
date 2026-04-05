package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// mockPaymentGateway implements ports.PaymentGateway for testing.
type mockPaymentGateway struct {
	checkoutURL   string
	portalURL     string
	cancelErr     error
	cancelCalled  bool
}

func (m *mockPaymentGateway) CreateCheckoutSession(_ context.Context, params ports.CheckoutParams) (*ports.CheckoutResult, error) {
	return &ports.CheckoutResult{
		SessionID: "sess_test_123",
		URL:       m.checkoutURL,
	}, nil
}

func (m *mockPaymentGateway) CreatePortalSession(_ context.Context, customerID, returnURL string) (*ports.PortalResult, error) {
	return &ports.PortalResult{URL: m.portalURL}, nil
}

func (m *mockPaymentGateway) CancelSubscription(_ context.Context, subID string, atPeriodEnd bool) error {
	m.cancelCalled = true
	return m.cancelErr
}

func (m *mockPaymentGateway) GetSubscription(_ context.Context, subID string) (*ports.SubscriptionResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func newTestBillingService(payments ports.PaymentGateway) *BillingService {
	billingRepo := memory.NewMemoryBillingRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	storageRepo := memory.NewMemoryStorageRepository()
	authRepo := memory.NewMemoryAppAuthRepository()
	return NewBillingService(
		payments, billingRepo, planRepo, appRepo, dbRepo, storageRepo, authRepo,
		"price_pro", "price_team", "price_biz", "app.zenith.dev",
	)
}

// --- PriceForTier tests ---

func TestPriceForTier_Pro(t *testing.T) {
	cents, priceID := PriceForTier(entities.PlanPro, "price_pro", "price_team", "price_biz")
	if cents != 2900 {
		t.Errorf("Expected 2900 cents for Pro, got %d", cents)
	}
	if priceID != "price_pro" {
		t.Errorf("Expected price_pro, got '%s'", priceID)
	}
}

func TestPriceForTier_Team(t *testing.T) {
	cents, priceID := PriceForTier(entities.PlanTeam, "price_pro", "price_team", "price_biz")
	if cents != 9900 {
		t.Errorf("Expected 9900 cents for Team, got %d", cents)
	}
	if priceID != "price_team" {
		t.Errorf("Expected price_team, got '%s'", priceID)
	}
}

func TestPriceForTier_Business(t *testing.T) {
	cents, priceID := PriceForTier(entities.PlanBusiness, "price_pro", "price_team", "price_biz")
	if cents != 14900 {
		t.Errorf("Expected 14900 cents for Business, got %d", cents)
	}
	if priceID != "price_biz" {
		t.Errorf("Expected price_biz, got '%s'", priceID)
	}
}

func TestPriceForTier_Free(t *testing.T) {
	cents, priceID := PriceForTier(entities.PlanFree, "price_pro", "price_team", "price_biz")
	if cents != 0 {
		t.Errorf("Expected 0 cents for Free, got %d", cents)
	}
	if priceID != "" {
		t.Errorf("Expected empty priceID for Free, got '%s'", priceID)
	}
}

func TestPriceForTier_Enterprise(t *testing.T) {
	cents, priceID := PriceForTier(entities.PlanEnterprise, "price_pro", "price_team", "price_biz")
	if cents != 0 {
		t.Errorf("Expected 0 cents for Enterprise (custom pricing), got %d", cents)
	}
	if priceID != "" {
		t.Errorf("Expected empty priceID for Enterprise, got '%s'", priceID)
	}
}

// --- BillingService accessor tests ---

func TestBillingService_Accessors(t *testing.T) {
	svc := newTestBillingService(nil)

	if svc.ProPriceID() != "price_pro" {
		t.Errorf("Expected ProPriceID 'price_pro', got '%s'", svc.ProPriceID())
	}
	if svc.TeamPriceID() != "price_team" {
		t.Errorf("Expected TeamPriceID 'price_team', got '%s'", svc.TeamPriceID())
	}
	if svc.BusinessPriceID() != "price_biz" {
		t.Errorf("Expected BusinessPriceID 'price_biz', got '%s'", svc.BusinessPriceID())
	}
	if svc.BillingRepo() == nil {
		t.Error("Expected non-nil BillingRepo")
	}
	if svc.PlanRepo() == nil {
		t.Error("Expected non-nil PlanRepo")
	}
	if svc.Payments() != nil {
		t.Error("Expected nil Payments when created without payment gateway")
	}
}

// --- GetBillingStatus tests ---

func TestGetBillingStatus_FreeUser(t *testing.T) {
	svc := newTestBillingService(nil)
	ctx := context.Background()

	resp, err := svc.GetBillingStatus(ctx, "user-free")
	if err != nil {
		t.Fatalf("GetBillingStatus failed: %v", err)
	}
	if resp.Tier != string(entities.PlanFree) {
		t.Errorf("Expected tier 'free', got '%s'", resp.Tier)
	}
	if resp.BillingStatus != "none" {
		t.Errorf("Expected billing status 'none', got '%s'", resp.BillingStatus)
	}
	if resp.PriceCents != 0 {
		t.Errorf("Expected 0 price cents for free, got %d", resp.PriceCents)
	}
	if resp.Currency != "eur" {
		t.Errorf("Expected currency 'eur', got '%s'", resp.Currency)
	}
	if resp.StripeEnabled {
		t.Error("Expected StripeEnabled=false when no payment gateway")
	}
}

func TestGetBillingStatus_WithStripe(t *testing.T) {
	payments := &mockPaymentGateway{checkoutURL: "https://checkout.stripe.com/test"}
	svc := newTestBillingService(payments)
	ctx := context.Background()

	resp, err := svc.GetBillingStatus(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetBillingStatus failed: %v", err)
	}
	if !resp.StripeEnabled {
		t.Error("Expected StripeEnabled=true when payment gateway is set")
	}
}

// --- CreateCheckoutSession tests ---

func TestCreateCheckoutSession_NoStripe(t *testing.T) {
	svc := newTestBillingService(nil)
	ctx := context.Background()

	_, err := svc.CreateCheckoutSession(ctx, "user-1", "user@test.com", "pro")
	if err == nil {
		t.Error("Expected error when Stripe is not enabled")
	}
}

func TestCreateCheckoutSession_InvalidTier(t *testing.T) {
	payments := &mockPaymentGateway{checkoutURL: "https://checkout.stripe.com/test"}
	svc := newTestBillingService(payments)
	ctx := context.Background()

	_, err := svc.CreateCheckoutSession(ctx, "user-1", "user@test.com", "free")
	if err == nil {
		t.Error("Expected error for free tier checkout (no priceID)")
	}
}

func TestCreateCheckoutSession_Success(t *testing.T) {
	payments := &mockPaymentGateway{checkoutURL: "https://checkout.stripe.com/test"}
	svc := newTestBillingService(payments)
	ctx := context.Background()

	resp, err := svc.CreateCheckoutSession(ctx, "user-1", "user@test.com", "pro")
	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}
	if resp.CheckoutURL != "https://checkout.stripe.com/test" {
		t.Errorf("Expected checkout URL, got '%s'", resp.CheckoutURL)
	}
	if resp.SessionID != "sess_test_123" {
		t.Errorf("Expected session ID 'sess_test_123', got '%s'", resp.SessionID)
	}
}

func TestCreateCheckoutSession_AlreadyOnPlan(t *testing.T) {
	payments := &mockPaymentGateway{checkoutURL: "https://checkout.stripe.com/test"}
	svc := newTestBillingService(payments)
	ctx := context.Background()

	// Upgrade user to pro first
	svc.PlanRepo().SetUserPlan(ctx, "user-already-pro", entities.PlanPro)

	_, err := svc.CreateCheckoutSession(ctx, "user-already-pro", "user@test.com", "pro")
	if err == nil {
		t.Error("Expected error when already on the same plan")
	}
}

// --- CreatePortalSession tests ---

func TestCreatePortalSession_NoStripe(t *testing.T) {
	svc := newTestBillingService(nil)
	ctx := context.Background()

	_, err := svc.CreatePortalSession(ctx, "user-1")
	if err == nil {
		t.Error("Expected error when Stripe is not enabled")
	}
}

func TestCreatePortalSession_NoCustomer(t *testing.T) {
	payments := &mockPaymentGateway{portalURL: "https://billing.stripe.com/test"}
	svc := newTestBillingService(payments)
	ctx := context.Background()

	_, err := svc.CreatePortalSession(ctx, "user-no-stripe")
	if err == nil {
		t.Error("Expected error when no Stripe customer exists")
	}
}

func TestCreatePortalSession_Success(t *testing.T) {
	payments := &mockPaymentGateway{portalURL: "https://billing.stripe.com/test"}
	svc := newTestBillingService(payments)
	ctx := context.Background()

	// Set stripe customer ID
	svc.BillingRepo().SetStripeCustomerID(ctx, "user-portal", "cus_test_123")

	resp, err := svc.CreatePortalSession(ctx, "user-portal")
	if err != nil {
		t.Fatalf("CreatePortalSession failed: %v", err)
	}
	if resp.PortalURL != "https://billing.stripe.com/test" {
		t.Errorf("Expected portal URL, got '%s'", resp.PortalURL)
	}
}

// --- CancelSubscription tests ---

func TestCancelSubscription_NoStripe(t *testing.T) {
	svc := newTestBillingService(nil)
	ctx := context.Background()

	err := svc.CancelSubscription(ctx, "user-1", false)
	if err == nil {
		t.Error("Expected error when Stripe is not enabled")
	}
}

func TestCancelSubscription_NoSubscription(t *testing.T) {
	payments := &mockPaymentGateway{}
	svc := newTestBillingService(payments)
	ctx := context.Background()

	err := svc.CancelSubscription(ctx, "user-no-sub", false)
	if err == nil {
		t.Error("Expected error when no subscription exists")
	}
}

func TestCancelSubscription_Success(t *testing.T) {
	payments := &mockPaymentGateway{}
	svc := newTestBillingService(payments)
	ctx := context.Background()

	// Create a subscription first
	svc.BillingRepo().CreateSubscription(ctx, &entities.Subscription{
		ID:                   "sub-1",
		UserID:               "user-cancel",
		StripeSubscriptionID: "sub_stripe_123",
		Tier:                 entities.PlanPro,
		Status:               entities.SubscriptionActive,
	})

	err := svc.CancelSubscription(ctx, "user-cancel", false)
	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}
	if !payments.cancelCalled {
		t.Error("Expected cancel to be called on payment gateway")
	}
}

// --- ListInvoices tests ---

func TestListInvoices_Empty(t *testing.T) {
	svc := newTestBillingService(nil)
	ctx := context.Background()

	invoices, err := svc.ListInvoices(ctx, "user-no-invoices")
	if err != nil {
		t.Fatalf("ListInvoices failed: %v", err)
	}
	if len(invoices) != 0 {
		t.Errorf("Expected 0 invoices, got %d", len(invoices))
	}
}

// --- GetAdminBillingOverview tests ---

func TestGetAdminBillingOverview_Empty(t *testing.T) {
	svc := newTestBillingService(nil)
	ctx := context.Background()

	overview, err := svc.GetAdminBillingOverview(ctx)
	if err != nil {
		t.Fatalf("GetAdminBillingOverview failed: %v", err)
	}
	if overview.ActiveSubscriptions != 0 {
		t.Errorf("Expected 0 active subscriptions, got %d", overview.ActiveSubscriptions)
	}
	if overview.MRRCents != 0 {
		t.Errorf("Expected 0 MRR, got %d", overview.MRRCents)
	}
}

// --- ProvisionUpgradeResources tests ---

func TestProvisionUpgradeResources_Free(t *testing.T) {
	svc := newTestBillingService(nil)
	ctx := context.Background()

	// Free tier should be a no-op (no panic)
	svc.ProvisionUpgradeResources(ctx, "user-1", entities.PlanFree)
}

func TestProvisionUpgradeResources_Pro_NoHarbor(t *testing.T) {
	svc := newTestBillingService(nil)
	ctx := context.Background()

	// Pro without Harbor should log but not panic
	svc.ProvisionUpgradeResources(ctx, "user-1", entities.PlanPro)
}

func TestProvisionUpgradeResources_Team_NoHarbor(t *testing.T) {
	svc := newTestBillingService(nil)
	ctx := context.Background()

	svc.ProvisionUpgradeResources(ctx, "user-1", entities.PlanTeam)
}

func TestProvisionUpgradeResources_Business_NoHarbor(t *testing.T) {
	svc := newTestBillingService(nil)
	ctx := context.Background()

	svc.ProvisionUpgradeResources(ctx, "user-1", entities.PlanBusiness)
}
