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

	// Git source (cloned by init container inside the Kaniko pod)
	RepoURL string `json:"repo_url"`
	Branch  string `json:"branch"`

	// If non-empty, a ConfigMap with this content is mounted as /workspace/Dockerfile.
	// Used when framework detection generates a Dockerfile that doesn't exist in the repo.
	GeneratedDockerfile string `json:"generated_dockerfile,omitempty"`

	// Build configuration
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

// GitCloneImage is the init container image used to clone the repo into the build context.
const GitCloneImage = "alpine/git:latest"

// NewKanikoJobSpec creates a Kaniko job spec for building an app.
func NewKanikoJobSpec(app *entities.App, deploymentID, imageTag string) *KanikoJobSpec {
	jobName := fmt.Sprintf("build-%s-%s", app.Subdomain, deploymentID[:min(8, len(deploymentID))])

	return &KanikoJobSpec{
		Name:           jobName,
		Namespace:      "zenith-builds",
		RepoURL:        app.RepoURL,
		Branch:         app.Branch,
		DockerfilePath: "Dockerfile",
		Destination:    imageTag,
		CPULimit:       "2",
		MemoryLimit:    "4Gi",
		AppID:          app.ID,
		DeploymentID:   deploymentID,
	}
}

// DockerfileConfigMapName returns the ConfigMap name for a generated Dockerfile.
func (s *KanikoJobSpec) DockerfileConfigMapName() string {
	return fmt.Sprintf("build-dockerfile-%s", s.Name)
}

// ToK8sJobManifest generates the Kubernetes Job manifest for Kaniko.
// The init container clones the git repo into /workspace (shared emptyDir).
// If a Dockerfile was auto-generated, it is injected via a ConfigMap volume.
func (s *KanikoJobSpec) ToK8sJobManifest() map[string]interface{} {
	// Init container: clone repo into /workspace
	initContainers := []map[string]interface{}{
		{
			"name":  "git-clone",
			"image": GitCloneImage,
			"args": []string{
				"clone", "--depth", "1", "-b", s.Branch, s.RepoURL, "/workspace",
			},
			"volumeMounts": []map[string]interface{}{
				{
					"name":      "build-context",
					"mountPath": "/workspace",
				},
			},
		},
	}

	// Kaniko container
	kanikoVolumeMounts := []map[string]interface{}{
		{
			"name":      "build-context",
			"mountPath": "/workspace",
		},
		{
			"name":      "docker-config",
			"mountPath": "/kaniko/.docker",
		},
	}

	// Volumes
	volumes := []map[string]interface{}{
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
	}

	// If a Dockerfile was generated, mount it from a ConfigMap
	if s.GeneratedDockerfile != "" {
		kanikoVolumeMounts = append(kanikoVolumeMounts, map[string]interface{}{
			"name":      "dockerfile",
			"mountPath": "/workspace/Dockerfile",
			"subPath":   "Dockerfile",
		})
		volumes = append(volumes, map[string]interface{}{
			"name": "dockerfile",
			"configMap": map[string]interface{}{
				"name": s.DockerfileConfigMapName(),
			},
		})
	}

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
					"restartPolicy":  "Never",
					"initContainers": initContainers,
					"containers": []map[string]interface{}{
						{
							"name":  "kaniko",
							"image": DefaultKanikoImage,
							"args": []string{
								"--context=dir:///workspace",
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
							"volumeMounts": kanikoVolumeMounts,
						},
					},
					"volumes": volumes,
				},
			},
		},
	}
}
