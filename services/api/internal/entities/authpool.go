package entities

// AuthPoolStatus represents the lifecycle status of an auth pool.
type AuthPoolStatus string

const (
	AuthPoolStatusProvisioning AuthPoolStatus = "provisioning"
	AuthPoolStatusActive       AuthPoolStatus = "active"
	AuthPoolStatusError        AuthPoolStatus = "error"
	AuthPoolStatusDeleting     AuthPoolStatus = "deleting"
)

// AuthPool represents a managed authentication pool backed by a Keycloak realm.
type AuthPool struct {
	ID           string         `json:"id"`
	UserID       string         `json:"user_id"`
	ProjectID    string         `json:"project_id"`
	Name         string         `json:"name"`
	RealmName    string         `json:"realm_name"`
	ClientID     string         `json:"client_id"`
	ClientSecret string         `json:"client_secret"`
	IssuerURL    string         `json:"issuer_url"`
	Status       AuthPoolStatus `json:"status"`
	UserCount    int            `json:"user_count"`
	MaxUsers     int            `json:"max_users"`
	Timestamps
}
