package entities

import "time"

// NotificationType categorizes notifications.
type NotificationType string

const (
	NotifDeploy  NotificationType = "deploy"
	NotifBilling NotificationType = "billing"
	NotifSystem  NotificationType = "system"
	NotifAlert   NotificationType = "alert"
)

// Notification represents a user-facing notification.
type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	Read      bool             `json:"read"`
	CreatedAt time.Time        `json:"created_at"`
}

// ActivityEntry represents a user activity log entry.
type ActivityEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Details   string    `json:"details,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
