package entities

import "time"

// DPAStatus represents the state of a DPA.
type DPAStatus string

const (
	DPAUnsigned DPAStatus = "unsigned"
	DPASigned   DPAStatus = "signed"
)

// DPARecord tracks whether a user has signed the Data Processing Agreement.
type DPARecord struct {
	UserID    string    `json:"user_id"`
	Status    DPAStatus `json:"status"`
	SignedBy  string    `json:"signed_by,omitempty"`
	SignedAt  time.Time `json:"signed_at,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
}

// BrandingConfig stores white-label customization for a user/org.
type BrandingConfig struct {
	UserID          string    `json:"user_id"`
	CompanyName     string    `json:"company_name"`
	LogoURL         string    `json:"logo_url"`
	PrimaryColor    string    `json:"primary_color"` // hex color
	DashboardDomain string    `json:"dashboard_domain,omitempty"` // e.g., "cloud.theirdomain.com"
	DomainVerified  bool      `json:"domain_verified"`
	HideBranding    bool      `json:"hide_branding"`
	UpdatedAt       time.Time `json:"updated_at"`
}
