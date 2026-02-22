package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// MemoryBillingRepository implements BillingRepository with in-memory maps.
type MemoryBillingRepository struct {
	mu              sync.RWMutex
	subscriptions   map[string]*entities.Subscription // keyed by subscription ID
	subsByUser      map[string]string                 // userID → subscription ID
	subsByStripeID  map[string]string                 // stripeSubID → subscription ID
	customerMapping map[string]string                 // userID → stripeCustomerID
	reverseCustomer map[string]string                 // stripeCustomerID → userID
	invoices        map[string]*entities.Invoice       // keyed by invoice ID
	invoicesByUser  map[string][]string               // userID → []invoice IDs
}

// NewMemoryBillingRepository creates an empty in-memory billing store.
func NewMemoryBillingRepository() *MemoryBillingRepository {
	return &MemoryBillingRepository{
		subscriptions:   make(map[string]*entities.Subscription),
		subsByUser:      make(map[string]string),
		subsByStripeID:  make(map[string]string),
		customerMapping: make(map[string]string),
		reverseCustomer: make(map[string]string),
		invoices:        make(map[string]*entities.Invoice),
		invoicesByUser:  make(map[string][]string),
	}
}

func (r *MemoryBillingRepository) CreateSubscription(_ context.Context, sub *entities.Subscription) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.subscriptions[sub.ID]; exists {
		return fmt.Errorf("subscription %s already exists", sub.ID)
	}

	now := time.Now()
	sub.CreatedAt = now
	sub.UpdatedAt = now

	r.subscriptions[sub.ID] = sub
	r.subsByUser[sub.UserID] = sub.ID
	r.subsByStripeID[sub.StripeSubscriptionID] = sub.ID
	return nil
}

func (r *MemoryBillingRepository) GetSubscriptionByUser(_ context.Context, userID string) (*entities.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	subID, ok := r.subsByUser[userID]
	if !ok {
		return nil, fmt.Errorf("no subscription for user %s", userID)
	}
	sub := r.subscriptions[subID]
	return sub, nil
}

func (r *MemoryBillingRepository) GetSubscriptionByStripeID(_ context.Context, stripeSubID string) (*entities.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	subID, ok := r.subsByStripeID[stripeSubID]
	if !ok {
		return nil, fmt.Errorf("no subscription with stripe ID %s", stripeSubID)
	}
	sub := r.subscriptions[subID]
	return sub, nil
}

func (r *MemoryBillingRepository) UpdateSubscriptionStatus(_ context.Context, stripeSubID string, status entities.SubscriptionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	subID, ok := r.subsByStripeID[stripeSubID]
	if !ok {
		return fmt.Errorf("no subscription with stripe ID %s", stripeSubID)
	}
	sub := r.subscriptions[subID]
	sub.Status = status
	sub.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryBillingRepository) UpdateSubscriptionTier(_ context.Context, stripeSubID string, tier entities.PlanTier, priceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	subID, ok := r.subsByStripeID[stripeSubID]
	if !ok {
		return fmt.Errorf("no subscription with stripe ID %s", stripeSubID)
	}
	sub := r.subscriptions[subID]
	sub.Tier = tier
	sub.StripePriceID = priceID
	sub.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryBillingRepository) SetStripeCustomerID(_ context.Context, userID, customerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.customerMapping[userID] = customerID
	r.reverseCustomer[customerID] = userID
	return nil
}

func (r *MemoryBillingRepository) GetStripeCustomerID(_ context.Context, userID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cid, ok := r.customerMapping[userID]
	if !ok {
		return "", nil
	}
	return cid, nil
}

func (r *MemoryBillingRepository) GetUserByStripeCustomerID(_ context.Context, customerID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	uid, ok := r.reverseCustomer[customerID]
	if !ok {
		return "", fmt.Errorf("no user for stripe customer %s", customerID)
	}
	return uid, nil
}

func (r *MemoryBillingRepository) UpsertInvoice(_ context.Context, inv *entities.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.invoices[inv.StripeInvoiceID]
	if exists {
		// Update existing invoice in-place.
		existing.AmountCents = inv.AmountCents
		existing.Status = inv.Status
		existing.InvoiceURL = inv.InvoiceURL
		existing.InvoicePDF = inv.InvoicePDF
		return nil
	}

	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = time.Now()
	}
	r.invoices[inv.StripeInvoiceID] = inv
	r.invoicesByUser[inv.UserID] = append(r.invoicesByUser[inv.UserID], inv.StripeInvoiceID)
	return nil
}

func (r *MemoryBillingRepository) ListInvoicesByUser(_ context.Context, userID string) ([]entities.Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.invoicesByUser[userID]
	result := make([]entities.Invoice, 0, len(ids))
	for _, id := range ids {
		if inv, ok := r.invoices[id]; ok {
			result = append(result, *inv)
		}
	}
	return result, nil
}

func (r *MemoryBillingRepository) GetBillingOverview(_ context.Context) (*entities.BillingOverview, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	overview := &entities.BillingOverview{}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	for _, sub := range r.subscriptions {
		switch sub.Status {
		case entities.SubscriptionActive:
			overview.ActiveSubscriptions++
			switch sub.Tier {
			case entities.PlanPro:
				overview.MRRCents += 2900
			case entities.PlanTeam:
				overview.MRRCents += 19900
			}
		case entities.SubscriptionPastDue:
			overview.PastDueCount++
		case entities.SubscriptionCanceled:
			if sub.UpdatedAt.After(monthStart) {
				overview.CanceledThisMonth++
			}
		}
	}

	total := overview.ActiveSubscriptions + overview.CanceledThisMonth
	if total > 0 {
		overview.ChurnRatePercent = float64(overview.CanceledThisMonth) / float64(total) * 100
	}

	return overview, nil
}
