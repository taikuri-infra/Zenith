package entities

import "time"

// EventSubject identifies a platform event type.
type EventSubject string

// Deploy events
const (
	EventDeployStarted   EventSubject = "zenith.deploy.started"
	EventDeployCompleted EventSubject = "zenith.deploy.completed"
	EventDeployFailed    EventSubject = "zenith.deploy.failed"
)

// Billing events
const (
	EventBillingCheckoutCompleted   EventSubject = "zenith.billing.checkout_completed"
	EventBillingInvoicePaid         EventSubject = "zenith.billing.invoice_paid"
	EventBillingPaymentFailed       EventSubject = "zenith.billing.payment_failed"
	EventBillingSubscriptionUpdated EventSubject = "zenith.billing.subscription_updated"
	EventBillingSubscriptionCanceled EventSubject = "zenith.billing.subscription_canceled"
)

// PlatformEvent is the envelope for all events published to the event bus.
type PlatformEvent struct {
	Subject   EventSubject           `json:"subject"`
	UserID    string                 `json:"user_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}
