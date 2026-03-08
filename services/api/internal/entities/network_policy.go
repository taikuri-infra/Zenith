package entities

import "time"

// NetworkPolicyAction defines the allowed action for a network policy rule.
type NetworkPolicyAction string

const (
	NetworkPolicyAllow NetworkPolicyAction = "allow"
	NetworkPolicyDeny  NetworkPolicyAction = "deny"
)

// NetworkPolicyDirection indicates traffic direction.
type NetworkPolicyDirection string

const (
	NetworkPolicyIngress NetworkPolicyDirection = "ingress"
	NetworkPolicyEgress  NetworkPolicyDirection = "egress"
)

// NetworkPolicyProtocol defines the transport protocol.
type NetworkPolicyProtocol string

const (
	NetworkPolicyTCP NetworkPolicyProtocol = "TCP"
	NetworkPolicyUDP NetworkPolicyProtocol = "UDP"
)

// NetworkPolicyRule represents a user-configurable Cilium network policy rule for an app.
type NetworkPolicyRule struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	AppID     string                 `json:"app_id"`
	Name      string                 `json:"name"`
	Direction NetworkPolicyDirection `json:"direction"`
	Action    NetworkPolicyAction    `json:"action"`
	Enabled   bool                   `json:"enabled"`
	Priority  int                    `json:"priority"`
	Config    NetworkPolicyConfig    `json:"config"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// NetworkPolicyConfig holds rule-specific configuration.
type NetworkPolicyConfig struct {
	// CIDR-based rules
	CIDRs []string `json:"cidrs,omitempty"` // e.g. ["10.0.0.0/8", "192.168.0.0/16"]

	// Port rules
	Ports []NetworkPolicyPort `json:"ports,omitempty"`

	// Namespace-based rules (allow traffic from/to specific namespaces)
	Namespaces []string `json:"namespaces,omitempty"`

	// Label-based pod selectors (key=value)
	PodLabels map[string]string `json:"pod_labels,omitempty"`

	// DNS-based egress rules (Cilium FQDN)
	FQDNs []string `json:"fqdns,omitempty"` // e.g. ["*.googleapis.com", "api.stripe.com"]
}

// NetworkPolicyPort defines a port rule.
type NetworkPolicyPort struct {
	Protocol NetworkPolicyProtocol `json:"protocol"`
	Port     int                   `json:"port"`
}
