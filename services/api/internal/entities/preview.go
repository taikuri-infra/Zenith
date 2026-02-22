package entities

import "time"

// PreviewDeployment represents a per-PR preview environment.
type PreviewDeployment struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	PRNumber  int       `json:"pr_number"`
	Branch    string    `json:"branch"`
	URL       string    `json:"url"`
	Status    string    `json:"status"` // building, running, stopped, deleted
	GitSHA    string    `json:"git_sha"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
