package entities

// AppAuthConfig holds per-app auth configuration.
type AppAuthConfig struct {
	AppID      string `json:"app_id"`
	Enabled    bool   `json:"enabled"`
	MaxUsers   int    `json:"max_users"`   // plan limit
	JWTSecret  string `json:"-"`           // per-app secret, never exposed in API
	Timestamps
}

// AppUser represents an end-user registered through an app's built-in auth.
type AppUser struct {
	ID       string `json:"id"`
	AppID    string `json:"app_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Verified bool   `json:"verified"`
	Timestamps
}
