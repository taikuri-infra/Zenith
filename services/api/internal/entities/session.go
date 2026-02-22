package entities

import "time"

// Session represents an active user session.
type Session struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Device     string    `json:"device"`
	Current    bool      `json:"current"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}
