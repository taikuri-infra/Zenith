package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppSpec defines the desired state of an App deployment
type AppSpec struct {
	// Image is the container image to deploy
	Image string `json:"image"`

	// Replicas is the desired number of replicas
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// Port is the container port the app listens on
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=8080
	Port int32 `json:"port,omitempty"`

	// Env is a list of environment variables
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Domain is an optional custom domain for the app
	Domain string `json:"domain,omitempty"`

	// Resources defines CPU/memory resource limits
	Resources *AppResources `json:"resources,omitempty"`

	// HealthCheck defines the health check configuration
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`

	// AutoScale defines autoscaling settings
	AutoScale *AutoScale `json:"autoScale,omitempty"`

	// BuildSource defines source-based deployment (Git URL)
	BuildSource *BuildSource `json:"buildSource,omitempty"`
}

type AppResources struct {
	// CPU limit (e.g., "500m", "1")
	CPU resource.Quantity `json:"cpu,omitempty"`
	// Memory limit (e.g., "256Mi", "1Gi")
	Memory resource.Quantity `json:"memory,omitempty"`
}

type HealthCheck struct {
	// Path is the HTTP path for health checks
	// +kubebuilder:default="/health"
	Path string `json:"path,omitempty"`
	// Port is the port for health checks (defaults to app port)
	Port int32 `json:"port,omitempty"`
	// IntervalSeconds is the interval between health checks
	// +kubebuilder:default=30
	IntervalSeconds int32 `json:"intervalSeconds,omitempty"`
}

type AutoScale struct {
	// MinReplicas is the minimum number of replicas
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	MinReplicas int32 `json:"minReplicas,omitempty"`
	// MaxReplicas is the maximum number of replicas
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=10
	MaxReplicas int32 `json:"maxReplicas,omitempty"`
	// TargetCPUPercent is the target CPU utilization percentage
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=80
	TargetCPUPercent int32 `json:"targetCPUPercent,omitempty"`
}

type BuildSource struct {
	// GitURL is the git repository URL
	GitURL string `json:"gitURL"`
	// Branch is the git branch to build from
	// +kubebuilder:default="main"
	Branch string `json:"branch,omitempty"`
	// Dockerfile path relative to repo root
	// +kubebuilder:default="Dockerfile"
	Dockerfile string `json:"dockerfile,omitempty"`
}

// AppStatus defines the observed state of App
type AppStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Pending;Building;Deploying;Running;Failed;Stopped
	Phase string `json:"phase,omitempty"`

	// ReadyReplicas is the number of ready replicas
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// URL is the external URL of the app
	URL string `json:"url,omitempty"`

	// InternalURL is the cluster-internal URL
	InternalURL string `json:"internalURL,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastDeployedAt is the timestamp of the last deployment
	LastDeployedAt *metav1.Time `json:"lastDeployedAt,omitempty"`

	// CurrentImage is the currently running image
	CurrentImage string `json:"currentImage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=app
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// App is the Schema for the apps API. Represents an application deployment.
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}
