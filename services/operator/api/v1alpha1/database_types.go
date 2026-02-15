package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseSpec defines the desired state of a Database
type DatabaseSpec struct {
	// Engine is the database engine type
	// +kubebuilder:validation:Enum=postgresql;mysql;mongodb;redis
	Engine string `json:"engine"`

	// Version is the database engine version
	Version string `json:"version"`

	// Storage is the storage size (e.g., "10Gi", "100Gi")
	Storage resource.Quantity `json:"storage"`

	// Replicas is the number of database instances (for HA)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=5
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	// Backup configures automated backups
	Backup *BackupConfig `json:"backup,omitempty"`

	// Resources defines CPU/memory limits
	Resources *DatabaseResources `json:"resources,omitempty"`

	// Parameters are engine-specific configuration parameters
	Parameters map[string]string `json:"parameters,omitempty"`
}

type BackupConfig struct {
	// Enabled enables automated backups
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
	// Schedule is the cron schedule for backups
	// +kubebuilder:default="0 2 * * *"
	Schedule string `json:"schedule,omitempty"`
	// RetentionDays is the number of days to retain backups
	// +kubebuilder:default=7
	RetentionDays int `json:"retentionDays,omitempty"`
}

type DatabaseResources struct {
	// CPU limit
	CPU resource.Quantity `json:"cpu,omitempty"`
	// Memory limit
	Memory resource.Quantity `json:"memory,omitempty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Provisioning;Ready;Failed;Deleting
	Phase string `json:"phase,omitempty"`

	// ConnectionString is the connection string (stored in Secret)
	ConnectionString string `json:"connectionString,omitempty"`

	// Host is the database host
	Host string `json:"host,omitempty"`

	// Port is the database port
	Port int32 `json:"port,omitempty"`

	// HetznerVolumeID is the Hetzner volume ID backing this database
	HetznerVolumeID string `json:"hetznerVolumeId,omitempty"`

	// SecretName is the name of the K8s Secret containing credentials
	SecretName string `json:"secretName,omitempty"`

	// StorageUsed is the current storage usage
	StorageUsed string `json:"storageUsed,omitempty"`

	// LastBackupTime is the timestamp of the last backup
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=db
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.spec.engine`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Storage",type=string,JSONPath=`.spec.storage`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Database is the Schema for the databases API. Represents a managed database.
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}
