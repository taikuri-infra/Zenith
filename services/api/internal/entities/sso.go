package entities

import "time"

// SSOProvider represents the type of SSO integration.
type SSOProvider string

const (
	SSOProviderSAML SSOProvider = "saml"
	SSOProviderOIDC SSOProvider = "oidc"
)

// SSOConfig stores SSO configuration for a user/org.
type SSOConfig struct {
	ID            string      `json:"id"`
	UserID        string      `json:"user_id"`
	Provider      SSOProvider `json:"provider"`
	EntityID      string      `json:"entity_id,omitempty"`       // SAML
	SSOURL        string      `json:"sso_url,omitempty"`         // SAML
	Certificate   string      `json:"certificate,omitempty"`     // SAML
	ClientID      string      `json:"client_id,omitempty"`       // OIDC
	ClientSecret  string      `json:"-"`                         // OIDC (never in JSON)
	DiscoveryURL  string      `json:"discovery_url,omitempty"`   // OIDC
	Enabled       bool        `json:"enabled"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}
