package entities

// EnvironmentName represents the name of an environment.
type EnvironmentName string

const (
	EnvironmentProduction EnvironmentName = "production"
	EnvironmentStaging    EnvironmentName = "staging"
)

// EnvironmentStatus represents the lifecycle status of an environment.
type EnvironmentStatus string

const (
	EnvironmentStatusProvisioning EnvironmentStatus = "provisioning"
	EnvironmentStatusActive       EnvironmentStatus = "active"
	EnvironmentStatusError        EnvironmentStatus = "error"
)

// Environment represents a deployment environment within a project.
// Each project has at least a "production" environment.
// Pro+ users also get a "staging" environment with minimal resources.
type Environment struct {
	ID        string            `json:"id"`
	ProjectID string            `json:"project_id"`
	Name      EnvironmentName   `json:"name"`
	Slug      string            `json:"slug"`
	Status    EnvironmentStatus `json:"status"`
	IsDefault bool              `json:"is_default"`
	Timestamps
}

// IsStaging returns true if this is a staging environment.
func (e *Environment) IsStaging() bool {
	return e.Name == EnvironmentStaging
}
