package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CrossplaneResourceSpec defines the desired state of a CrossplaneResource.
// It represents a Crossplane managed resource that can be provisioned by
// any supported Crossplane provider (aws, gcp, azure, hetzner).
type CrossplaneResourceSpec struct {
	// Provider is the Crossplane provider (aws, gcp, azure, hetzner)
	// +kubebuilder:validation:Enum=aws;gcp;azure;hetzner
	Provider string `json:"provider"`

	// ResourceKind is the Crossplane resource kind (e.g., Bucket, Instance, Database)
	ResourceKind string `json:"resourceKind"`

	// ResourceAPIVersion is the Crossplane resource API version (e.g., "s3.aws.upbound.io/v1beta1")
	ResourceAPIVersion string `json:"resourceAPIVersion,omitempty"`

	// Config holds provider-specific configuration key-value pairs
	// that are passed directly to the Crossplane managed resource spec
	Config map[string]string `json:"config,omitempty"`

	// WriteConnectionSecretToRef specifies the Secret where connection
	// details from the Crossplane resource should be written
	WriteConnectionSecretToRef *SecretKeyRef `json:"writeConnectionSecretToRef,omitempty"`

	// ProviderConfigRef is the name of the Crossplane ProviderConfig to use
	// +kubebuilder:default="default"
	ProviderConfigRef string `json:"providerConfigRef,omitempty"`

	// DeletionPolicy specifies what happens when the CrossplaneResource is deleted
	// +kubebuilder:validation:Enum=Delete;Orphan
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// CrossplaneResourceStatus defines the observed state of CrossplaneResource
type CrossplaneResourceStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Provisioning;Ready;Failed;Deleting
	Phase string `json:"phase,omitempty"`

	// CrossplaneResourceName is the name of the underlying Crossplane managed resource
	CrossplaneResourceName string `json:"crossplaneResourceName,omitempty"`

	// CrossplaneReady indicates whether the Crossplane resource reports Ready=True
	CrossplaneReady bool `json:"crossplaneReady,omitempty"`

	// ConnectionSecretName is the name of the Secret containing connection details
	ConnectionSecretName string `json:"connectionSecretName,omitempty"`

	// ExternalName is the external identifier of the provisioned resource
	ExternalName string `json:"externalName,omitempty"`

	// Message provides a human-readable status message
	Message string `json:"message,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cpr
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Kind",type=string,JSONPath=`.spec.resourceKind`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.crossplaneReady`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// CrossplaneResource is the Schema for the crossplaneresources API.
// Represents a Crossplane managed resource provisioned through Zenith.
type CrossplaneResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CrossplaneResourceSpec   `json:"spec,omitempty"`
	Status CrossplaneResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CrossplaneResourceList contains a list of CrossplaneResource
type CrossplaneResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CrossplaneResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CrossplaneResource{}, &CrossplaneResourceList{})
}
