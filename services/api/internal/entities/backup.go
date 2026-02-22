package entities

// BackupStatus represents the lifecycle status of a backup.
type BackupStatus string

const (
	BackupStatusPending    BackupStatus = "pending"
	BackupStatusRunning    BackupStatus = "running"
	BackupStatusCompleted  BackupStatus = "completed"
	BackupStatusFailed     BackupStatus = "failed"
)

// BackupType represents how the backup was triggered.
type BackupType string

const (
	BackupTypeManual    BackupType = "manual"
	BackupTypeScheduled BackupType = "scheduled"
)

// DatabaseBackup represents a point-in-time backup of a user database.
type DatabaseBackup struct {
	ID         string       `json:"id"`
	DatabaseID string       `json:"database_id"`
	UserID     string       `json:"user_id"`
	Type       BackupType   `json:"type"`
	Status     BackupStatus `json:"status"`
	SizeMB     int          `json:"size_mb"`
	StorageKey string       `json:"storage_key"` // S3 key for the backup file
	Error      string       `json:"error,omitempty"`
	Timestamps
}
