package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresBillingRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBillingRepository(pool *pgxpool.Pool) *PostgresBillingRepository {
	return &PostgresBillingRepository{pool: pool}
}

func (r *PostgresBillingRepository) CreateSubscription(ctx context.Context, sub *entities.Subscription) error {
	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, stripe_subscription_id, stripe_customer_id, stripe_price_id,
		 tier, status, current_period_start, current_period_end, cancel_at_period_end, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		sub.ID, sub.UserID, sub.StripeSubscriptionID, sub.StripeCustomerID, sub.StripePriceID,
		string(sub.Tier), string(sub.Status), sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd, sub.CreatedAt, sub.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}
	return nil
}

func (r *PostgresBillingRepository) GetSubscriptionByUser(ctx context.Context, userID string) (*entities.Subscription, error) {
	var s entities.Subscription
	var tier, status string
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, stripe_subscription_id, stripe_customer_id, stripe_price_id,
		 tier, status, current_period_start, current_period_end, cancel_at_period_end, created_at, updated_at
		 FROM subscriptions WHERE user_id = $1`, userID,
	).Scan(&s.ID, &s.UserID, &s.StripeSubscriptionID, &s.StripeCustomerID, &s.StripePriceID,
		&tier, &status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
		&s.CancelAtPeriodEnd, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("subscription not found for user %s", userID)
	}
	s.Tier = entities.PlanTier(tier)
	s.Status = entities.SubscriptionStatus(status)
	return &s, nil
}

func (r *PostgresBillingRepository) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*entities.Subscription, error) {
	var s entities.Subscription
	var tier, status string
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, stripe_subscription_id, stripe_customer_id, stripe_price_id,
		 tier, status, current_period_start, current_period_end, cancel_at_period_end, created_at, updated_at
		 FROM subscriptions WHERE stripe_subscription_id = $1`, stripeSubID,
	).Scan(&s.ID, &s.UserID, &s.StripeSubscriptionID, &s.StripeCustomerID, &s.StripePriceID,
		&tier, &status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
		&s.CancelAtPeriodEnd, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("subscription not found for stripe ID %s", stripeSubID)
	}
	s.Tier = entities.PlanTier(tier)
	s.Status = entities.SubscriptionStatus(status)
	return &s, nil
}

func (r *PostgresBillingRepository) UpdateSubscriptionStatus(ctx context.Context, stripeSubID string, status entities.SubscriptionStatus) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE subscriptions SET status = $1, updated_at = $2 WHERE stripe_subscription_id = $3`,
		string(status), time.Now(), stripeSubID,
	)
	if err != nil {
		return fmt.Errorf("update subscription status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found: %s", stripeSubID)
	}
	return nil
}

func (r *PostgresBillingRepository) UpdateSubscriptionTier(ctx context.Context, stripeSubID string, tier entities.PlanTier, priceID string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE subscriptions SET tier = $1, stripe_price_id = $2, updated_at = $3 WHERE stripe_subscription_id = $4`,
		string(tier), priceID, time.Now(), stripeSubID,
	)
	if err != nil {
		return fmt.Errorf("update subscription tier: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found: %s", stripeSubID)
	}
	return nil
}

func (r *PostgresBillingRepository) SetStripeCustomerID(ctx context.Context, userID, customerID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO stripe_customers (user_id, customer_id) VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET customer_id = $2`,
		userID, customerID,
	)
	if err != nil {
		return fmt.Errorf("set stripe customer: %w", err)
	}
	return nil
}

func (r *PostgresBillingRepository) GetStripeCustomerID(ctx context.Context, userID string) (string, error) {
	var customerID string
	err := r.pool.QueryRow(ctx,
		`SELECT customer_id FROM stripe_customers WHERE user_id = $1`, userID,
	).Scan(&customerID)
	if err != nil {
		return "", fmt.Errorf("stripe customer not found for user %s", userID)
	}
	return customerID, nil
}

func (r *PostgresBillingRepository) GetUserByStripeCustomerID(ctx context.Context, customerID string) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx,
		`SELECT user_id FROM stripe_customers WHERE customer_id = $1`, customerID,
	).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("user not found for stripe customer %s", customerID)
	}
	return userID, nil
}

func (r *PostgresBillingRepository) UpsertInvoice(ctx context.Context, inv *entities.Invoice) error {
	if inv.ID == "" {
		inv.ID = uuid.New().String()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO invoices (id, user_id, stripe_invoice_id, amount_cents, currency, status,
		 invoice_url, invoice_pdf, period_start, period_end, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (stripe_invoice_id) WHERE stripe_invoice_id != ''
		 DO UPDATE SET status = $6, invoice_url = $7, invoice_pdf = $8, amount_cents = $4`,
		inv.ID, inv.UserID, inv.StripeInvoiceID, inv.AmountCents, inv.Currency,
		string(inv.Status), inv.InvoiceURL, inv.InvoicePDF,
		inv.PeriodStart, inv.PeriodEnd, inv.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert invoice: %w", err)
	}
	return nil
}

func (r *PostgresBillingRepository) ListInvoicesByUser(ctx context.Context, userID string) ([]entities.Invoice, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, stripe_invoice_id, amount_cents, currency, status,
		 invoice_url, invoice_pdf, period_start, period_end, created_at
		 FROM invoices WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	var invoices []entities.Invoice
	for rows.Next() {
		var inv entities.Invoice
		var status string
		if err := rows.Scan(&inv.ID, &inv.UserID, &inv.StripeInvoiceID, &inv.AmountCents,
			&inv.Currency, &status, &inv.InvoiceURL, &inv.InvoicePDF,
			&inv.PeriodStart, &inv.PeriodEnd, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		inv.Status = entities.InvoiceStatus(status)
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

func (r *PostgresBillingRepository) GetBillingOverview(ctx context.Context) (*entities.BillingOverview, error) {
	overview := &entities.BillingOverview{}

	// Active subscriptions count
	r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscriptions WHERE status = 'active'`,
	).Scan(&overview.ActiveSubscriptions)

	// Past due count
	r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscriptions WHERE status = 'past_due'`,
	).Scan(&overview.PastDueCount)

	// MRR = sum of active subscription amounts
	r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(i.amount_cents), 0) FROM invoices i
		 JOIN subscriptions s ON s.user_id = i.user_id
		 WHERE s.status = 'active' AND i.status = 'paid'
		 AND i.created_at >= date_trunc('month', NOW())`,
	).Scan(&overview.MRRCents)

	// Canceled this month
	r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscriptions
		 WHERE status = 'canceled' AND updated_at >= date_trunc('month', NOW())`,
	).Scan(&overview.CanceledThisMonth)

	return overview, nil
}
