package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageBucketSpec defines the desired state of a StorageBucket
type StorageBucketSpec struct {
	// Access controls the bucket access level
	// +kubebuilder:validation:Enum=private;public-read
	// +kubebuilder:default=private
	Access string `json:"access,omitempty"`

	// Versioning enables object versioning
	// +kubebuilder:default=false
	Versioning bool `json:"versioning,omitempty"`

	// Region is the storage region
	// +kubebuilder:validation:Enum=fsn1;nbg1
	// +kubebuilder:default=fsn1
	Region string `json:"region,omitempty"`

	// LifecycleRules define object lifecycle policies
	LifecycleRules []LifecycleRule `json:"lifecycleRules,omitempty"`

	// CORSRules define CORS configuration
	CORSRules []CORSRule `json:"corsRules,omitempty"`
}

type LifecycleRule struct {
	// Prefix filters objects by prefix
	Prefix string `json:"prefix,omitempty"`
	// ExpirationDays is the number of days before objects expire
	ExpirationDays int `json:"expirationDays,omitempty"`
	// TransitionDays is the number of days before transitioning storage class
	TransitionDays int `json:"transitionDays,omitempty"`
}

type CORSRule struct {
	// AllowedOrigins is the list of allowed origins
	AllowedOrigins []string `json:"allowedOrigins"`
	// AllowedMethods is the list of allowed HTTP methods
	AllowedMethods []string `json:"allowedMethods"`
	// AllowedHeaders is the list of allowed headers
	AllowedHeaders []string `json:"allowedHeaders,omitempty"`
	// MaxAgeSeconds is the max age for preflight requests
	MaxAgeSeconds int `json:"maxAgeSeconds,omitempty"`
}

// StorageBucketStatus defines the observed state of StorageBucket
type StorageBucketStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Creating;Ready;Failed;Deleting
	Phase string `json:"phase,omitempty"`

	// Endpoint is the S3-compatible endpoint URL
	Endpoint string `json:"endpoint,omitempty"`

	// BucketName is the actual bucket name in Hetzner
	BucketName string `json:"bucketName,omitempty"`

	// SecretName is the name of the K8s Secret containing S3 credentials
	SecretName string `json:"secretName,omitempty"`

	// SizeBytes is the current bucket size in bytes
	SizeBytes int64 `json:"sizeBytes,omitempty"`

	// ObjectCount is the number of objects in the bucket
	ObjectCount int64 `json:"objectCount,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=sb
// +kubebuilder:printcolumn:name="Access",type=string,JSONPath=`.spec.access`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// StorageBucket is the Schema for the storagebuckets API. Represents S3-compatible object storage.
type StorageBucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageBucketSpec   `json:"spec,omitempty"`
	Status StorageBucketStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StorageBucketList contains a list of StorageBucket
type StorageBucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageBucket `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageBucket{}, &StorageBucketList{})
}
