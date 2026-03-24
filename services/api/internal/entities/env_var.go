package entities

// EnvVarSource indicates how an environment variable was created.
type EnvVarSource string

const (
	EnvVarSourceManual         EnvVarSource = "manual"
	EnvVarSourceManagedService EnvVarSource = "managed_service"
	EnvVarSourceServiceLink    EnvVarSource = "service_link"
	EnvVarSourceComposeImport  EnvVarSource = "compose_import"
)

// AppEnvVar represents an environment variable with source tracking.
type AppEnvVar struct {
	ID            string       `json:"id"`
	AppID         string       `json:"app_id"`
	EnvironmentID string       `json:"environment_id,omitempty"` // empty = production/default
	Key           string       `json:"key"`
	Value         string       `json:"value,omitempty"`
	IsSecret      bool         `json:"is_secret"`
	Source        EnvVarSource `json:"source"`
	SourceID      string       `json:"source_id,omitempty"`
	Timestamps
}
