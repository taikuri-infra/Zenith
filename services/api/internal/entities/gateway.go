package entities

import "encoding/json"

// GatewayStatus represents the lifecycle state of a gateway.
type GatewayStatus string

const (
	GatewayStatusProvisioning GatewayStatus = "provisioning"
	GatewayStatusActive       GatewayStatus = "active"
	GatewayStatusError        GatewayStatus = "error"
	GatewayStatusDeleting     GatewayStatus = "deleting"
)

// GatewayRouteStatus represents the state of a gateway route.
type GatewayRouteStatus string

const (
	GatewayRouteStatusActive  GatewayRouteStatus = "active"
	GatewayRouteStatusStopped GatewayRouteStatus = "stopped"
)

// GatewayRouteAuth represents the authentication type for a route.
type GatewayRouteAuth string

const (
	GatewayRouteAuthNone    GatewayRouteAuth = "none"
	GatewayRouteAuthJWT     GatewayRouteAuth = "jwt"
	GatewayRouteAuthKeyAuth GatewayRouteAuth = "key-auth"
	GatewayRouteAuthOIDC    GatewayRouteAuth = "oidc"
)

// GatewayRoutePlugin represents an inline APISIX plugin on a route.
type GatewayRoutePlugin struct {
	Name   string          `json:"name"`
	Enable bool            `json:"enable"`
	Config json.RawMessage `json:"config"`
}

// AllowedPlugins is the allowlist for Phase 1 (validated server-side).
var AllowedPlugins = map[string]bool{
	"cors":           true,
	"limit-count":    true,
	"jwt-auth":       true,
	"key-auth":       true,
	"ip-restriction": true,
	"proxy-rewrite":  true,
	"request-id":     true,
	"openid-connect": true,
}

// Gateway represents a customer API gateway backed by APISIX.
type Gateway struct {
	ID         string        `json:"id"`
	UserID     string        `json:"user_id"`
	ProjectID  string        `json:"project_id"`
	Name       string        `json:"name"`
	Slug       string        `json:"slug"`
	Status     GatewayStatus `json:"status"`
	Endpoint   string        `json:"endpoint"`
	RouteCount int           `json:"route_count"`
	Timestamps
}

// GatewayRoute represents a single route within a gateway.
type GatewayRoute struct {
	ID           string               `json:"id"`
	GatewayID    string               `json:"gateway_id"`
	GroupID      string               `json:"group_id,omitempty"`
	Name         string               `json:"name"`
	Path         string               `json:"path"`
	Methods      []string             `json:"methods"`
	AppID        string               `json:"app_id"`
	AppSubdomain string               `json:"app_subdomain"`
	StripPrefix  bool                 `json:"strip_prefix"`
	Auth         GatewayRouteAuth     `json:"auth"`
	AuthPoolID   string               `json:"auth_pool_id,omitempty"`
	Plugins      []GatewayRoutePlugin `json:"plugins"`
	Priority     int                  `json:"priority"`
	Status       GatewayRouteStatus   `json:"status"`
	Timestamps
}

// GatewayCustomDomainStatus represents the lifecycle state of a custom domain.
type GatewayCustomDomainStatus string

const (
	GatewayCustomDomainStatusPending GatewayCustomDomainStatus = "pending"
	GatewayCustomDomainStatusActive  GatewayCustomDomainStatus = "active"
	GatewayCustomDomainStatusFailed  GatewayCustomDomainStatus = "failed"
)

// GatewayCustomDomain represents a custom domain attached to a gateway.
type GatewayCustomDomain struct {
	ID        string                    `json:"id"`
	GatewayID string                    `json:"gateway_id"`
	UserID    string                    `json:"user_id"`
	Domain    string                    `json:"domain"`
	Status    GatewayCustomDomainStatus `json:"status"`
	TLSReady  bool                      `json:"tls_ready"`
	Timestamps
}

// GatewayGroup represents a service abstraction that bundles routes pointing to the same app.
type GatewayGroup struct {
	ID           string               `json:"id"`
	GatewayID    string               `json:"gateway_id"`
	Name         string               `json:"name"`
	AppID        string               `json:"app_id"`
	AppSubdomain string               `json:"app_subdomain"`
	Plugins      []GatewayRoutePlugin `json:"plugins"`
	Timestamps
}
