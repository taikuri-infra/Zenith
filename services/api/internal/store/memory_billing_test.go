package store

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestMemoryBillingRepository_Subscription(t *testing.T) {
	repo := NewMemoryBillingRepository()
	ctx := context.Background()

	sub := &entities.Subscription{
		ID:                   "sub-1",
		UserID:               "user-1",
		StripeSubscriptionID: "sub_stripe_123",
		StripeCustomerID:     "cus_stripe_456",
		StripePriceID:        "price_pro",
		Tier:                 entities.PlanPro,
		Status:               entities.SubscriptionActive,
		CurrentPeriodStart:   time.Now(),
		CurrentPeriodEnd:     time.Now().Add(30 * 24 * time.Hour),
	}

	if err := repo.CreateSubscription(ctx, sub); err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}

	// Duplicate should fail
	if err := repo.CreateSubscription(ctx, sub); err == nil {
		t.Fatal("expected error on duplicate subscription")
	}

	// Get by user
	got, err := repo.GetSubscriptionByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetSubscriptionByUser: %v", err)
	}
	if got.Tier != entities.PlanPro {
		t.Errorf("expected tier pro, got %s", got.Tier)
	}

	// Get by stripe ID
	got2, err := repo.GetSubscriptionByStripeID(ctx, "sub_stripe_123")
	if err != nil {
		t.Fatalf("GetSubscriptionByStripeID: %v", err)
	}
	if got2.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", got2.UserID)
	}

	// Update status
	if err := repo.UpdateSubscriptionStatus(ctx, "sub_stripe_123", entities.SubscriptionPastDue); err != nil {
		t.Fatalf("UpdateSubscriptionStatus: %v", err)
	}
	got3, _ := repo.GetSubscriptionByStripeID(ctx, "sub_stripe_123")
	if got3.Status != entities.SubscriptionPastDue {
		t.Errorf("expected past_due, got %s", got3.Status)
	}

	// Update tier
	if err := repo.UpdateSubscriptionTier(ctx, "sub_stripe_123", entities.PlanTeam, "price_team"); err != nil {
		t.Fatalf("UpdateSubscriptionTier: %v", err)
	}
	got4, _ := repo.GetSubscriptionByStripeID(ctx, "sub_stripe_123")
	if got4.Tier != entities.PlanTeam {
		t.Errorf("expected team, got %s", got4.Tier)
	}
}

func TestMemoryBillingRepository_CustomerMapping(t *testing.T) {
	repo := NewMemoryBillingRepository()
	ctx := context.Background()

	if err := repo.SetStripeCustomerID(ctx, "user-1", "cus_123"); err != nil {
		t.Fatalf("SetStripeCustomerID: %v", err)
	}

	cid, err := repo.GetStripeCustomerID(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetStripeCustomerID: %v", err)
	}
	if cid != "cus_123" {
		t.Errorf("expected cus_123, got %s", cid)
	}

	uid, err := repo.GetUserByStripeCustomerID(ctx, "cus_123")
	if err != nil {
		t.Fatalf("GetUserByStripeCustomerID: %v", err)
	}
	if uid != "user-1" {
		t.Errorf("expected user-1, got %s", uid)
	}

	// Unknown customer returns empty
	cid2, err := repo.GetStripeCustomerID(ctx, "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cid2 != "" {
		t.Errorf("expected empty, got %s", cid2)
	}
}

func TestMemoryBillingRepository_Invoices(t *testing.T) {
	repo := NewMemoryBillingRepository()
	ctx := context.Background()

	inv := &entities.Invoice{
		ID:              "inv-1",
		UserID:          "user-1",
		StripeInvoiceID: "in_stripe_789",
		AmountCents:     2900,
		Currency:        "eur",
		Status:          entities.InvoicePaid,
		InvoiceURL:      "https://stripe.com/invoice/1",
		PeriodStart:     time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:       time.Now(),
	}

	if err := repo.UpsertInvoice(ctx, inv); err != nil {
		t.Fatalf("UpsertInvoice: %v", err)
	}

	invoices, err := repo.ListInvoicesByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListInvoicesByUser: %v", err)
	}
	if len(invoices) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(invoices))
	}
	if invoices[0].AmountCents != 2900 {
		t.Errorf("expected 2900, got %d", invoices[0].AmountCents)
	}

	// Upsert updates existing
	inv2 := &entities.Invoice{
		StripeInvoiceID: "in_stripe_789",
		UserID:          "user-1",
		AmountCents:     3500,
		Status:          entities.InvoicePaid,
		InvoiceURL:      "https://stripe.com/invoice/1-updated",
	}
	if err := repo.UpsertInvoice(ctx, inv2); err != nil {
		t.Fatalf("UpsertInvoice (update): %v", err)
	}
	invoices2, _ := repo.ListInvoicesByUser(ctx, "user-1")
	if invoices2[0].AmountCents != 3500 {
		t.Errorf("expected 3500 after upsert, got %d", invoices2[0].AmountCents)
	}
}

func TestMemoryBillingRepository_Overview(t *testing.T) {
	repo := NewMemoryBillingRepository()
	ctx := context.Background()

	_ = repo.CreateSubscription(ctx, &entities.Subscription{
		ID: "s1", UserID: "u1", StripeSubscriptionID: "ss1",
		Tier: entities.PlanPro, Status: entities.SubscriptionActive,
	})
	_ = repo.CreateSubscription(ctx, &entities.Subscription{
		ID: "s2", UserID: "u2", StripeSubscriptionID: "ss2",
		Tier: entities.PlanTeam, Status: entities.SubscriptionActive,
	})

	overview, err := repo.GetBillingOverview(ctx)
	if err != nil {
		t.Fatalf("GetBillingOverview: %v", err)
	}
	if overview.ActiveSubscriptions != 2 {
		t.Errorf("expected 2 active, got %d", overview.ActiveSubscriptions)
	}
	expectedMRR := 2900 + 19900
	if overview.MRRCents != expectedMRR {
		t.Errorf("expected MRR %d, got %d", expectedMRR, overview.MRRCents)
	}
}

// Compile-time check
var _ BillingRepository = (*MemoryBillingRepository)(nil)
