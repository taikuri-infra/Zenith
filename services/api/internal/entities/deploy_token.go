package entities

import "time"

// DeployTokenScope represents a permission scope for a deploy token.
type DeployTokenScope string

const (
	ScopeDeployStaging    DeployTokenScope = "deploy:staging"
	ScopeDeployProduction DeployTokenScope = "deploy:production"
	ScopeAppRead          DeployTokenScope = "app:read"
	ScopeAppWrite         DeployTokenScope = "app:write"
	ScopeDBRead           DeployTokenScope = "db:read"
	ScopeLogsRead         DeployTokenScope = "logs:read"
	ScopeInfraAll         DeployTokenScope = "infra:*"
)

// AllDeployTokenScopes returns all valid deploy token scopes.
func AllDeployTokenScopes() []DeployTokenScope {
	return []DeployTokenScope{
		ScopeDeployStaging, ScopeDeployProduction,
		ScopeAppRead, ScopeAppWrite,
		ScopeDBRead, ScopeLogsRead,
		ScopeInfraAll,
	}
}

// ValidDeployTokenScope checks if a scope string is valid.
func ValidDeployTokenScope(s string) bool {
	for _, scope := range AllDeployTokenScopes() {
		if string(scope) == s {
			return true
		}
	}
	return false
}

// DeployToken represents a CI/CD deploy token with scoped permissions.
// Token format: ID = "znt_id_..." (public), Secret = "znt_sk_..." (hashed with Argon2id).
type DeployToken struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	ProjectID         string     `json:"project_id"`
	Name              string     `json:"name"`
	TokenID           string     `json:"token_id"`
	TokenPrefix       string     `json:"token_prefix"`
	TokenHash         string     `json:"-"`
	Secret            string     `json:"secret,omitempty"` // Only returned on creation
	Scopes            []string   `json:"scopes"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	PreviousHash      string     `json:"-"`
	PreviousExpiresAt *time.Time `json:"-"`
	RotatedAt         *time.Time `json:"rotated_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty"`
}

// IsExpired checks if the token has expired.
func (t *DeployToken) IsExpired() bool {
	return t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now())
}

// IsRevoked checks if the token has been revoked.
func (t *DeployToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// HasScope checks if the token has a specific scope.
func (t *DeployToken) HasScope(scope string) bool {
	for _, s := range t.Scopes {
		if s == scope || s == string(ScopeInfraAll) {
			return true
		}
	}
	return false
}

// InGracePeriod returns true if the token was rotated and the old hash is still valid.
func (t *DeployToken) InGracePeriod() bool {
	return t.PreviousHash != "" && t.PreviousExpiresAt != nil && t.PreviousExpiresAt.After(time.Now())
}
