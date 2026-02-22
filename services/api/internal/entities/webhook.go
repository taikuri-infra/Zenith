package entities

import "time"

// WebhookEvent represents the type of event that triggers a webhook.
type WebhookEvent string

const (
	WebhookEventDeploySuccess WebhookEvent = "deploy.success"
	WebhookEventDeployFailed  WebhookEvent = "deploy.failed"
	WebhookEventAppSleeping   WebhookEvent = "app.sleeping"
	WebhookEventAppWaking     WebhookEvent = "app.waking"
	WebhookEventDBCreated     WebhookEvent = "db.created"
	WebhookEventLimitReached  WebhookEvent = "limit.reached"
)

// WebhookDeliveryStatus represents the result of a webhook delivery attempt.
type WebhookDeliveryStatus string

const (
	WebhookDeliveryPending WebhookDeliveryStatus = "pending"
	WebhookDeliverySuccess WebhookDeliveryStatus = "success"
	WebhookDeliveryFailed  WebhookDeliveryStatus = "failed"
)

// UserWebhook is a user-registered webhook endpoint.
type UserWebhook struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	URL       string         `json:"url"`
	Events    []WebhookEvent `json:"events"`
	Secret    string         `json:"secret,omitempty"` // HMAC signing secret
	Active    bool           `json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// WebhookDelivery records a single delivery attempt.
type WebhookDelivery struct {
	ID         string                `json:"id"`
	WebhookID  string                `json:"webhook_id"`
	Event      WebhookEvent          `json:"event"`
	Payload    string                `json:"payload"`
	Status     WebhookDeliveryStatus `json:"status"`
	StatusCode int                   `json:"status_code,omitempty"`
	Error      string                `json:"error,omitempty"`
	Attempts   int                   `json:"attempts"`
	CreatedAt  time.Time             `json:"created_at"`
}
