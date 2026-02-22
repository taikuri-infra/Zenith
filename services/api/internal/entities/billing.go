package entities

import "time"

// SubscriptionStatus represents the state of a Stripe subscription.
type SubscriptionStatus string

const (
	SubscriptionActive     SubscriptionStatus = "active"
	SubscriptionPastDue    SubscriptionStatus = "past_due"
	SubscriptionCanceled   SubscriptionStatus = "canceled"
	SubscriptionIncomplete SubscriptionStatus = "incomplete"
	SubscriptionTrialing   SubscriptionStatus = "trialing"
	SubscriptionUnpaid     SubscriptionStatus = "unpaid"
)

// Subscription tracks a user's Stripe subscription.
type Subscription struct {
	ID                   string             `json:"id"`
	UserID               string             `json:"user_id"`
	StripeSubscriptionID string             `json:"stripe_subscription_id"`
	StripeCustomerID     string             `json:"stripe_customer_id"`
	StripePriceID        string             `json:"stripe_price_id"`
	Tier                 PlanTier           `json:"tier"`
	Status               SubscriptionStatus `json:"status"`
	CurrentPeriodStart   time.Time          `json:"current_period_start"`
	CurrentPeriodEnd     time.Time          `json:"current_period_end"`
	CancelAtPeriodEnd    bool               `json:"cancel_at_period_end"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// InvoiceStatus represents the state of an invoice.
type InvoiceStatus string

const (
	InvoicePaid          InvoiceStatus = "paid"
	InvoiceOpen          InvoiceStatus = "open"
	InvoiceVoid          InvoiceStatus = "void"
	InvoiceUncollectible InvoiceStatus = "uncollectible"
	InvoiceDraft         InvoiceStatus = "draft"
)

// Invoice tracks a Stripe invoice for a user.
type Invoice struct {
	ID              string        `json:"id"`
	UserID          string        `json:"user_id"`
	StripeInvoiceID string        `json:"stripe_invoice_id"`
	AmountCents     int           `json:"amount_cents"`
	Currency        string        `json:"currency"`
	Status          InvoiceStatus `json:"status"`
	InvoiceURL      string        `json:"invoice_url"`
	InvoicePDF      string        `json:"invoice_pdf"`
	PeriodStart     time.Time     `json:"period_start"`
	PeriodEnd       time.Time     `json:"period_end"`
	CreatedAt       time.Time     `json:"created_at"`
}

// BillingOverview provides admin-level billing metrics.
type BillingOverview struct {
	MRRCents            int     `json:"mrr_cents"`
	ActiveSubscriptions int     `json:"active_subscriptions"`
	PastDueCount        int     `json:"past_due_count"`
	CanceledThisMonth   int     `json:"canceled_this_month"`
	ChurnRatePercent    float64 `json:"churn_rate_percent"`
}
