package entities

// Project represents a logical grouping of resources (apps, databases, storage, gateways).
// Plan limits are enforced per-user across all projects, not per-project.
type Project struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Timestamps
}
