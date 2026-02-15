package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GatewayRouteSpec defines the desired state of a GatewayRoute
type GatewayRouteSpec struct {
	// Path is the URL path pattern (e.g., "/api/v1/users")
	Path string `json:"path"`

	// Methods is the list of allowed HTTP methods
	// +kubebuilder:default={"GET","POST","PUT","DELETE"}
	Methods []string `json:"methods,omitempty"`

	// Service is the backend service reference
	Service ServiceRef `json:"service"`

	// Plugins is the list of Kong plugins to apply
	Plugins []GatewayPlugin `json:"plugins,omitempty"`

	// Auth configures authentication for this route
	Auth *RouteAuth `json:"auth,omitempty"`

	// RateLimit configures rate limiting
	RateLimit *RateLimit `json:"rateLimit,omitempty"`

	// CORS configures CORS for this route
	CORS *RouteCORS `json:"cors,omitempty"`
}

type ServiceRef struct {
	// Name is the service name
	Name string `json:"name"`
	// Port is the service port
	Port int32 `json:"port"`
}

type GatewayPlugin struct {
	// Name is the plugin name (e.g., "jwt-auth", "rate-limiting")
	Name string `json:"name"`
	// Config is the plugin configuration
	Config map[string]string `json:"config,omitempty"`
}

type RouteAuth struct {
	// Enabled enables JWT authentication
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
	// Scopes required for this route
	Scopes []string `json:"scopes,omitempty"`
}

type RateLimit struct {
	// RequestsPerSecond is the rate limit
	RequestsPerSecond int `json:"requestsPerSecond,omitempty"`
	// RequestsPerMinute is the rate limit per minute
	RequestsPerMinute int `json:"requestsPerMinute,omitempty"`
}

type RouteCORS struct {
	// AllowedOrigins is the list of allowed origins
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`
	// AllowedMethods is the list of allowed methods
	AllowedMethods []string `json:"allowedMethods,omitempty"`
	// AllowedHeaders is the list of allowed headers
	AllowedHeaders []string `json:"allowedHeaders,omitempty"`
}

// GatewayRouteStatus defines the observed state of GatewayRoute
type GatewayRouteStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Configuring;Active;Failed
	Phase string `json:"phase,omitempty"`

	// KongRouteID is the ID of the route in Kong
	KongRouteID string `json:"kongRouteId,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=gwr
// +kubebuilder:printcolumn:name="Path",type=string,JSONPath=`.spec.path`
// +kubebuilder:printcolumn:name="Service",type=string,JSONPath=`.spec.service.name`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// GatewayRoute is the Schema for the gatewayroutes API. Represents a Kong API gateway route.
type GatewayRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatewayRouteSpec   `json:"spec,omitempty"`
	Status GatewayRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GatewayRouteList contains a list of GatewayRoute
type GatewayRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GatewayRoute `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GatewayRoute{}, &GatewayRouteList{})
}
