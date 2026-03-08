package entities

import "time"

// WAFRuleType categorizes the type of WAF rule.
type WAFRuleType string

const (
	WAFRuleRateLimit  WAFRuleType = "rate_limit"
	WAFRuleIPBlock    WAFRuleType = "ip_block"
	WAFRuleIPAllow    WAFRuleType = "ip_allow"
	WAFRuleBodyLimit  WAFRuleType = "body_limit"
	WAFRuleGeoBlock   WAFRuleType = "geo_block"
	WAFRuleHeaderRule WAFRuleType = "header_rule"
)

// WAFRule represents a user-configurable WAF rule for an app.
type WAFRule struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	AppID     string      `json:"app_id"`
	Name      string      `json:"name"`
	Type      WAFRuleType `json:"type"`
	Enabled   bool        `json:"enabled"`
	Priority  int         `json:"priority"`
	Config    WAFConfig   `json:"config"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// WAFConfig holds rule-specific configuration.
type WAFConfig struct {
	// Rate limiting
	RatePerSecond int `json:"rate_per_second,omitempty"`
	BurstSize     int `json:"burst_size,omitempty"`

	// IP blocking/allowing
	IPAddresses []string `json:"ip_addresses,omitempty"`

	// Body limit
	MaxBodySizeKB int `json:"max_body_size_kb,omitempty"`

	// Geo blocking
	Countries []string `json:"countries,omitempty"` // ISO 3166-1 alpha-2

	// Header rules
	HeaderName  string `json:"header_name,omitempty"`
	HeaderMatch string `json:"header_match,omitempty"` // regex pattern
	Action      string `json:"action,omitempty"`       // "block" or "allow"
}
