package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitSyncSpec defines the desired state of a GitSync resource
type GitSyncSpec struct {
	// RepoURL is the Git repository URL
	RepoURL string `json:"repoURL"`

	// Branch to sync from
	// +kubebuilder:default="main"
	Branch string `json:"branch,omitempty"`

	// Path within the repo to sync
	// +kubebuilder:default="/"
	Path string `json:"path,omitempty"`

	// Interval is the sync interval (e.g., "5m", "1h")
	// +kubebuilder:default="5m"
	Interval string `json:"interval,omitempty"`

	// SecretRef is the reference to git credentials secret
	SecretRef *SecretKeyRef `json:"secretRef,omitempty"`

	// AutoSync enables automatic sync on the configured interval
	// +kubebuilder:default=true
	AutoSync bool `json:"autoSync,omitempty"`

	// PruneResources removes resources from the cluster that are not present in git
	PruneResources bool `json:"pruneResources,omitempty"`
}

// GitSyncStatus defines the observed state of GitSync
type GitSyncStatus struct {
	// Phase represents the current sync lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Syncing;Synced;Failed
	Phase string `json:"phase,omitempty"`

	// LastSyncTime is the timestamp of the last successful sync
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// LastCommitHash is the git commit hash of the last successful sync
	LastCommitHash string `json:"lastCommitHash,omitempty"`

	// Message is a human-readable status message
	Message string `json:"message,omitempty"`

	// SyncedResources is the number of resources applied from git
	SyncedResources int `json:"syncedResources,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=gs
// +kubebuilder:printcolumn:name="Repo",type=string,JSONPath=`.spec.repoURL`
// +kubebuilder:printcolumn:name="Branch",type=string,JSONPath=`.spec.branch`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Last Sync",type=date,JSONPath=`.status.lastSyncTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// GitSync is the Schema for the gitsyncs API. Represents a GitOps sync configuration.
type GitSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitSyncSpec   `json:"spec,omitempty"`
	Status GitSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitSyncList contains a list of GitSync
type GitSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitSync{}, &GitSyncList{})
}
