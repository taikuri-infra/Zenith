package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DomainSpec defines the desired state of a Domain
type DomainSpec struct {
	// Domain is the fully qualified domain name
	Domain string `json:"domain"`

	// AppRef is the name of the App this domain points to
	AppRef string `json:"appRef"`

	// SSL configures SSL/TLS settings
	SSL *SSLConfig `json:"ssl,omitempty"`

	// DNS configures DNS record settings
	DNS *DNSConfig `json:"dns,omitempty"`
}

type SSLConfig struct {
	// Enabled enables automatic SSL via cert-manager
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
	// Issuer is the cert-manager issuer to use
	// +kubebuilder:default="letsencrypt-prod"
	Issuer string `json:"issuer,omitempty"`
}

type DNSConfig struct {
	// AutoConfigure enables automatic DNS record creation via Hetzner DNS
	// +kubebuilder:default=true
	AutoConfigure bool `json:"autoConfigure,omitempty"`
	// Type is the DNS record type
	// +kubebuilder:validation:Enum=A;CNAME
	// +kubebuilder:default=A
	Type string `json:"type,omitempty"`
}

// DomainStatus defines the observed state of Domain
type DomainStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Configuring;Active;Failed
	Phase string `json:"phase,omitempty"`

	// SSLReady indicates if the SSL certificate is ready
	SSLReady bool `json:"sslReady,omitempty"`

	// DNSConfigured indicates if DNS records are configured
	DNSConfigured bool `json:"dnsConfigured,omitempty"`

	// CertificateExpiry is the SSL certificate expiry date
	CertificateExpiry *metav1.Time `json:"certificateExpiry,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=dom
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=`.spec.domain`
// +kubebuilder:printcolumn:name="App",type=string,JSONPath=`.spec.appRef`
// +kubebuilder:printcolumn:name="SSL",type=boolean,JSONPath=`.status.sslReady`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Domain is the Schema for the domains API. Represents a custom domain with auto SSL.
type Domain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DomainSpec   `json:"spec,omitempty"`
	Status DomainStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DomainList contains a list of Domain
type DomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Domain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Domain{}, &DomainList{})
}
