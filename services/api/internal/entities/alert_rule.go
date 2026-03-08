package entities

import "time"

// AlertSeverity defines the severity level of an alert.
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityInfo     AlertSeverity = "info"
)

// AlertRule represents a user-configurable Prometheus alerting rule for an app.
type AlertRule struct {
	ID          string        `json:"id"`
	UserID      string        `json:"user_id"`
	AppID       string        `json:"app_id"`
	Name        string        `json:"name"`
	Enabled     bool          `json:"enabled"`
	Metric      string        `json:"metric"`      // Prometheus metric name or PromQL expression
	Condition   string        `json:"condition"`    // e.g. "> 80", "< 1", "== 0"
	Duration    string        `json:"duration"`     // e.g. "5m", "10m", "1h" (Prometheus for: duration)
	Severity    AlertSeverity `json:"severity"`
	Description string        `json:"description"`  // Human-readable description
	NotifyEmail bool          `json:"notify_email"` // Send email on alert
	NotifySlack bool          `json:"notify_slack"` // Send to Slack webhook
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// CustomMetric represents a user-defined custom Prometheus recording rule.
type CustomMetric struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	AppID      string    `json:"app_id"`
	Name       string    `json:"name"`        // Recording rule name (e.g. "app:request_rate:5m")
	Expression string    `json:"expression"`  // PromQL expression
	Labels     map[string]string `json:"labels,omitempty"` // Additional labels
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
