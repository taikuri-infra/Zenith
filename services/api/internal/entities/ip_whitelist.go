package entities

import "time"

// IPWhitelistEntry represents an allowed IP range.
type IPWhitelistEntry struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	CIDR        string    `json:"cidr"` // e.g., "192.168.1.0/24" or "10.0.0.1/32"
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
