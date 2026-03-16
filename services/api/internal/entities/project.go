package entities

// ProjectStatus represents the lifecycle status of a project.
type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusArchived ProjectStatus = "archived"
)

// Project represents a logical grouping of resources (apps, databases, storage, gateways).
// Plan limits are enforced per-user across all projects, not per-project.
type Project struct {
	ID          string        `json:"id"`
	UserID      string        `json:"user_id"`
	Name        string        `json:"name"`
	Slug        string        `json:"slug"`
	Description string        `json:"description"`
	Status      ProjectStatus `json:"status"`

	// Harbor integration (populated when project has a registry)
	HarborProjectName string `json:"-"`
	HarborRobotUser   string `json:"harbor_robot_user,omitempty"`
	HarborRobotPass   string `json:"-"`

	Timestamps

	// Populated on read (not stored in projects table)
	Services        []App            `json:"services,omitempty"`
	ManagedServices []ManagedService `json:"managed_services,omitempty"`
}
