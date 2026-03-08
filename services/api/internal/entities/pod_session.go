package entities

import "time"

// PodExecSessionStatus represents the lifecycle of a terminal session.
type PodExecSessionStatus string

const (
	PodSessionActive    PodExecSessionStatus = "active"
	PodSessionCompleted PodExecSessionStatus = "completed"
)

// PodExecSession tracks a user's interactive terminal session to a pod.
type PodExecSession struct {
	ID           string               `json:"id"`
	UserID       string               `json:"user_id"`
	UserEmail    string               `json:"user_email"`
	AppID        string               `json:"app_id"`
	AppName      string               `json:"app_name"`
	PodName      string               `json:"pod_name"`
	Container    string               `json:"container"`
	Command      string               `json:"command"`
	Status       PodExecSessionStatus `json:"status"`
	IPAddress    string               `json:"ip_address"`
	RecordingKey string               `json:"recording_key,omitempty"` // S3 key for asciinema recording
	StartedAt    time.Time            `json:"started_at"`
	EndedAt      *time.Time           `json:"ended_at,omitempty"`
	DurationSecs int                  `json:"duration_secs"`
}
