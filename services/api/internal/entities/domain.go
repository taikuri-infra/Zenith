package entities

// DomainStatus represents the lifecycle of a custom domain.
type DomainStatus string

const (
	DomainStatusPending  DomainStatus = "pending"
	DomainStatusVerified DomainStatus = "verified"
	DomainStatusActive   DomainStatus = "active"
	DomainStatusFailed   DomainStatus = "failed"
)

// CustomDomain represents a custom domain attached to an app.
type CustomDomain struct {
	ID       string       `json:"id"`
	AppID    string       `json:"app_id"`
	UserID   string       `json:"user_id"`
	Domain   string       `json:"domain"`
	Status   DomainStatus `json:"status"`
	TLSReady bool         `json:"tls_ready"`
	Timestamps
}
