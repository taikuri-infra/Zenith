package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of a Project (tenant boundary)
type ProjectSpec struct {
	// DisplayName is the human-readable project name
	DisplayName string `json:"displayName"`

	// Owner is the email of the project owner
	Owner string `json:"owner"`

	// Plan is the subscription plan (free, pro, enterprise)
	// +kubebuilder:validation:Enum=free;pro;enterprise
	// +kubebuilder:default=free
	Plan string `json:"plan,omitempty"`

	// Region is the Hetzner datacenter region
	// +kubebuilder:validation:Enum=fsn1;nbg1;hel1;ash;hil
	// +kubebuilder:default=fsn1
	Region string `json:"region,omitempty"`

	// ResourceQuota defines resource limits for the project
	ResourceQuota *ResourceQuota `json:"resourceQuota,omitempty"`
}

type ResourceQuota struct {
	// MaxApps is the maximum number of apps
	MaxApps int `json:"maxApps,omitempty"`
	// MaxDatabases is the maximum number of databases
	MaxDatabases int `json:"maxDatabases,omitempty"`
	// MaxStorageGB is the maximum storage in GB
	MaxStorageGB int `json:"maxStorageGB,omitempty"`
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Active;Suspended;Deleting
	Phase string `json:"phase,omitempty"`

	// Namespace is the Kubernetes namespace for this project
	Namespace string `json:"namespace,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// AppCount is the current number of deployed apps
	AppCount int `json:"appCount,omitempty"`

	// DatabaseCount is the current number of databases
	DatabaseCount int `json:"databaseCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=proj
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.spec.owner`
// +kubebuilder:printcolumn:name="Plan",type=string,JSONPath=`.spec.plan`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Project is the Schema for the projects API. Represents a tenant boundary.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
