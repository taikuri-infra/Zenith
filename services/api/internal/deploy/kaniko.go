package deploy

import (
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// KanikoJobSpec represents the configuration for a Kaniko build job.
// In production, this is converted to a Kubernetes batch/v1 Job.
type KanikoJobSpec struct {
	// Job metadata
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	// Build configuration
	ContextDir     string `json:"context_dir"`
	DockerfilePath string `json:"dockerfile_path"`
	Destination    string `json:"destination"`

	// Resource limits
	CPULimit    string `json:"cpu_limit"`
	MemoryLimit string `json:"memory_limit"`

	// Labels for identification
	AppID        string `json:"app_id"`
	DeploymentID string `json:"deployment_id"`
}

// DefaultKanikoImage is the Kaniko executor image.
const DefaultKanikoImage = "gcr.io/kaniko-project/executor:v1.23.0"

// NewKanikoJobSpec creates a Kaniko job spec for building an app.
func NewKanikoJobSpec(app *entities.App, deploymentID, imageTag, contextDir string) *KanikoJobSpec {
	jobName := fmt.Sprintf("build-%s-%s", app.Subdomain, deploymentID[:min(8, len(deploymentID))])

	return &KanikoJobSpec{
		Name:           jobName,
		Namespace:      "zenith-builds",
		ContextDir:     contextDir,
		DockerfilePath: "Dockerfile",
		Destination:    imageTag,
		CPULimit:       "2",
		MemoryLimit:    "4Gi",
		AppID:          app.ID,
		DeploymentID:   deploymentID,
	}
}

// ToK8sJobManifest generates the Kubernetes Job YAML manifest for Kaniko.
// This returns a map that can be serialized to JSON/YAML for the K8s API.
func (s *KanikoJobSpec) ToK8sJobManifest() map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      s.Name,
			"namespace": s.Namespace,
			"labels": map[string]string{
				"zenith.dev/component":  "build",
				"zenith.dev/app-id":     s.AppID,
				"zenith.dev/deployment": s.DeploymentID,
			},
		},
		"spec": map[string]interface{}{
			"backoffLimit":            int32(0),
			"ttlSecondsAfterFinished": int32(3600),
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]string{
						"zenith.dev/component": "build",
						"zenith.dev/app-id":    s.AppID,
					},
				},
				"spec": map[string]interface{}{
					"restartPolicy": "Never",
					"containers": []map[string]interface{}{
						{
							"name":  "kaniko",
							"image": DefaultKanikoImage,
							"args": []string{
								"--context=dir://" + s.ContextDir,
								"--dockerfile=" + s.DockerfilePath,
								"--destination=" + s.Destination,
								"--cache=true",
								"--cache-ttl=72h",
								"--snapshotMode=redo",
								"--compressed-caching=false",
							},
							"resources": map[string]interface{}{
								"limits": map[string]string{
									"cpu":    s.CPULimit,
									"memory": s.MemoryLimit,
								},
								"requests": map[string]string{
									"cpu":    "500m",
									"memory": "1Gi",
								},
							},
							"volumeMounts": []map[string]interface{}{
								{
									"name":      "build-context",
									"mountPath": s.ContextDir,
								},
								{
									"name":      "docker-config",
									"mountPath": "/kaniko/.docker",
								},
							},
						},
					},
					"volumes": []map[string]interface{}{
						{
							"name": "build-context",
							"emptyDir": map[string]interface{}{
								"sizeLimit": "2Gi",
							},
						},
						{
							"name": "docker-config",
							"secret": map[string]interface{}{
								"secretName": "registry-credentials",
							},
						},
					},
				},
			},
		},
	}
}
