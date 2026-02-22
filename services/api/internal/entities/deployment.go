package entities

import "time"

// DeploymentStatus represents the lifecycle status of a single deployment.
type DeploymentStatus string

const (
	DeployStatusPending    DeploymentStatus = "pending"
	DeployStatusBuilding   DeploymentStatus = "building"
	DeployStatusDeploying  DeploymentStatus = "deploying"
	DeployStatusActive     DeploymentStatus = "active"
	DeployStatusSuperseded DeploymentStatus = "superseded"
	DeployStatusFailed     DeploymentStatus = "failed"
)

// Deployment represents a single deployment attempt for an app.
type Deployment struct {
	ID        string           `json:"id"`
	AppID     string           `json:"app_id"`
	ImageTag  string           `json:"image_tag"`
	GitSHA    string           `json:"git_sha"`
	Status    DeploymentStatus `json:"status"`
	BuildLog  string           `json:"build_log,omitempty"`
	Error     string           `json:"error,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

// EnvVar represents an environment variable for an app.
type EnvVar struct {
	ID    string `json:"id"`
	AppID string `json:"app_id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Secret represents an encrypted key-value secret for an app.
type Secret struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
}

// SecretWithValue is returned when the caller requests decrypted values.
type SecretWithValue struct {
	Secret
	Value string `json:"value"`
}

// Release represents a versioned image build registered by zenith-actions or CI.
type Release struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	Image     string    `json:"image"`
	GitSHA    string    `json:"git_sha"`
	Branch    string    `json:"branch"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
