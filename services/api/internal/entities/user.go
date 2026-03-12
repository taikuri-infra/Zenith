package entities

import "time"

// Role represents a user's role in a project.
type Role string

const (
	RoleOwner     Role = "owner"
	RoleAdmin     Role = "admin"
	RoleDeveloper Role = "developer"
	RoleViewer    Role = "viewer"
	RoleCustomer  Role = "customer"
)

// CanPerform checks if a role can perform a given action.
func (r Role) CanPerform(action string) bool {
	permissions := map[Role]map[string]bool{
		RoleOwner: {
			"read": true, "write": true, "deploy": true, "delete": true,
			"manage_members": true, "manage_billing": true, "manage_settings": true,
			"view": true, "manage": true,
		},
		RoleAdmin: {
			"read": true, "write": true, "deploy": true, "delete": true,
			"manage_members": true, "manage_settings": true,
			"view": true, "manage": true,
		},
		RoleDeveloper: {
			"read": true, "write": true, "deploy": true,
			"view": true,
		},
		RoleViewer: {
			"read": true,
			"view": true,
		},
		RoleCustomer: {
			"read": true, "write": true, "deploy": true, "delete": true,
			"view": true, "manage_billing": true,
		},
	}

	if perms, ok := permissions[r]; ok {
		return perms[action]
	}
	return false
}

// IsValid checks if the role is a valid known role.
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleDeveloper, RoleViewer, RoleCustomer:
		return true
	}
	return false
}

// User represents a platform user.
type User struct {
	ID               string     `json:"id"`
	Email            string     `json:"email"`
	Name             string     `json:"name"`
	AvatarURL        string     `json:"avatar_url,omitempty"`
	Role             Role       `json:"role"`
	ProjectID        string     `json:"project_id,omitempty"`
	StripeCustomerID string     `json:"stripe_customer_id,omitempty"`
	EmailVerified    bool       `json:"email_verified"`
	EmailVerifiedAt  *time.Time `json:"email_verified_at,omitempty"`
	AuthProvider     string     `json:"auth_provider"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Signup source / UTM tracking
	SignupSource string `json:"signup_source,omitempty"`
	UTMSource    string `json:"utm_source,omitempty"`
	UTMMedium    string `json:"utm_medium,omitempty"`
	UTMCampaign  string `json:"utm_campaign,omitempty"`
	UTMContent   string `json:"utm_content,omitempty"`
	UTMTerm      string `json:"utm_term,omitempty"`
	ReferrerURL  string `json:"referrer_url,omitempty"`
	SignupIP     string `json:"signup_ip,omitempty"`

	// Onboarding
	OnboardingCompleted   bool       `json:"onboarding_completed"`
	OnboardingStep        int        `json:"onboarding_step"`
	OnboardingCompletedAt *time.Time `json:"onboarding_completed_at,omitempty"`

	// Referral
	ReferralCode string `json:"referral_code,omitempty"`
	ReferredBy   string `json:"referred_by,omitempty"`

	// Activity
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// TeamMemberStatus represents the status of a team member invite.
type TeamMemberStatus string

const (
	TeamMemberPending   TeamMemberStatus = "pending"
	TeamMemberActive    TeamMemberStatus = "active"
	TeamMemberSuspended TeamMemberStatus = "suspended"
)

// TeamMember represents an invited team member sharing the owner's resources.
type TeamMember struct {
	ID              string           `json:"id"`
	AccountID       string           `json:"account_id"`
	UserID          string           `json:"user_id,omitempty"`
	Email           string           `json:"email"`
	Role            Role             `json:"role"`
	Status          TeamMemberStatus `json:"status"`
	InviteTokenHash string           `json:"-"`
	InviteExpiresAt *time.Time       `json:"-"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// APIKeyType represents the type of API key.
type APIKeyType string

const (
	APIKeyPersonal APIKeyType = "personal"
	APIKeyService  APIKeyType = "service"
)

// APIKey represents an API key for CI/CD access.
type APIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	KeyHash    string     `json:"-"`
	Key        string     `json:"key,omitempty"`
	Scopes     []string   `json:"scopes"`
	Type       APIKeyType `json:"type"`
	UserID     string     `json:"user_id"`
	ProjectID  string     `json:"project_id"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// HasScope checks if the API key has the given scope.
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}
