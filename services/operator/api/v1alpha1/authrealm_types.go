package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AuthRealmSpec defines the desired state of an AuthRealm
type AuthRealmSpec struct {
	// DisplayName is the human-readable realm name
	DisplayName string `json:"displayName"`

	// Providers is the list of identity providers
	Providers []IdentityProvider `json:"providers,omitempty"`

	// Clients is the list of OAuth2/OIDC clients
	Clients []AuthClient `json:"clients,omitempty"`

	// Settings configures realm-level settings
	Settings *RealmSettings `json:"settings,omitempty"`
}

type IdentityProvider struct {
	// Name is the provider identifier
	Name string `json:"name"`
	// Type is the provider type
	// +kubebuilder:validation:Enum=google;github;azure-ad;saml;oidc;ldap
	Type string `json:"type"`
	// ClientID is the OAuth2 client ID
	ClientID string `json:"clientID,omitempty"`
	// ClientSecretRef is a reference to a Secret containing the client secret
	ClientSecretRef *SecretKeyRef `json:"clientSecretRef,omitempty"`
	// Enabled controls whether this provider is active
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
	// Config holds provider-specific configuration
	Config map[string]string `json:"config,omitempty"`
}

type SecretKeyRef struct {
	// Name is the Secret name
	Name string `json:"name"`
	// Key is the key within the Secret
	Key string `json:"key"`
}

type AuthClient struct {
	// Name is the client identifier
	Name string `json:"name"`
	// Type is the client type
	// +kubebuilder:validation:Enum=public;confidential
	// +kubebuilder:default=public
	Type string `json:"type,omitempty"`
	// RedirectURIs is the list of allowed redirect URIs
	RedirectURIs []string `json:"redirectURIs,omitempty"`
	// Scopes is the list of allowed scopes
	Scopes []string `json:"scopes,omitempty"`
}

type RealmSettings struct {
	// MFARequired requires multi-factor authentication
	// +kubebuilder:default=false
	MFARequired bool `json:"mfaRequired,omitempty"`
	// SessionTimeout is the session timeout duration
	// +kubebuilder:default="24h"
	SessionTimeout string `json:"sessionTimeout,omitempty"`
	// PasswordPolicy configures password requirements
	PasswordPolicy *PasswordPolicy `json:"passwordPolicy,omitempty"`
}

type PasswordPolicy struct {
	// MinLength is the minimum password length
	// +kubebuilder:default=8
	MinLength int `json:"minLength,omitempty"`
	// RequireUppercase requires uppercase characters
	RequireUppercase bool `json:"requireUppercase,omitempty"`
	// RequireNumbers requires numeric characters
	RequireNumbers bool `json:"requireNumbers,omitempty"`
	// RequireSpecial requires special characters
	RequireSpecial bool `json:"requireSpecial,omitempty"`
}

// AuthRealmStatus defines the observed state of AuthRealm
type AuthRealmStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Provisioning;Ready;Failed
	Phase string `json:"phase,omitempty"`

	// Endpoint is the OIDC discovery endpoint
	Endpoint string `json:"endpoint,omitempty"`

	// UserCount is the number of registered users
	UserCount int `json:"userCount,omitempty"`

	// ClientCount is the number of configured clients
	ClientCount int `json:"clientCount,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=realm
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Users",type=integer,JSONPath=`.status.userCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AuthRealm is the Schema for the authrealms API. Represents an authentication realm.
type AuthRealm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthRealmSpec   `json:"spec,omitempty"`
	Status AuthRealmStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AuthRealmList contains a list of AuthRealm
type AuthRealmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthRealm `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuthRealm{}, &AuthRealmList{})
}
