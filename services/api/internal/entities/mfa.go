package entities

import "time"

// MFAStatus represents the state of MFA enrollment.
type MFAStatus string

const (
	MFAStatusDisabled MFAStatus = "disabled"
	MFAStatusPending  MFAStatus = "pending"  // secret generated, not yet verified
	MFAStatusEnabled  MFAStatus = "enabled"
)

// MFAEnrollment tracks a user's MFA configuration.
type MFAEnrollment struct {
	UserID      string    `json:"user_id"`
	Status      MFAStatus `json:"status"`
	Secret      string    `json:"-"` // TOTP secret (never exposed in JSON)
	BackupCodes []string  `json:"-"` // one-time recovery codes (never in JSON)
	UsedCodes   []string  `json:"-"` // already-used backup codes
	EnabledAt   time.Time `json:"enabled_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
