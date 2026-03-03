package entities

import "time"

// Role represents a user's role in a project.
type Role string

const (
	RoleOwner     Role = "owner"
	RoleAdmin     Role = "admin"
	RoleDeveloper Role = "developer"
	RoleViewer    Role = "viewer"
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
	}

	if perms, ok := permissions[r]; ok {
		return perms[action]
	}
	return false
}

// IsValid checks if the role is a valid known role.
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleDeveloper, RoleViewer:
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
}

// APIKey represents an API key for CI/CD access.
type APIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	KeyHash    string     `json:"-"`
	Key        string     `json:"key,omitempty"`
	Scopes     []string   `json:"scopes"`
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
