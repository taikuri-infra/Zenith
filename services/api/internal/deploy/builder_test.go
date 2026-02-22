package deploy

import (
	"encoding/json"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// --- Builder tests ---

func TestNewBuilder(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	b := NewBuilder(repo, "", "", nil, nil)
	if b.workDir != "/tmp/zenith-builds" {
		t.Errorf("Expected default workDir, got '%s'", b.workDir)
	}
	if b.registry != "registry.freezenith.com" {
		t.Errorf("Expected default registry, got '%s'", b.registry)
	}
}

func TestNewBuilderCustom(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	b := NewBuilder(repo, "/builds", "ghcr.io/myorg", nil, nil)
	if b.workDir != "/builds" {
		t.Errorf("Expected '/builds', got '%s'", b.workDir)
	}
	if b.registry != "ghcr.io/myorg" {
		t.Errorf("Expected 'ghcr.io/myorg', got '%s'", b.registry)
	}
}

// --- Kaniko tests ---

func TestNewKanikoJobSpec(t *testing.T) {
	app := &entities.App{
		ID:        "app-123",
		Name:      "web",
		Subdomain: "web",
	}
	deployID := "deploy-abc12345"
	spec := NewKanikoJobSpec(app, deployID, "registry/web:abc12345", "/workspace")

	if spec.Name != "build-web-deploy-a" {
		t.Errorf("Expected job name 'build-web-deploy-a', got '%s'", spec.Name)
	}
	if spec.Namespace != "zenith-builds" {
		t.Errorf("Expected namespace 'zenith-builds', got '%s'", spec.Namespace)
	}
	if spec.Destination != "registry/web:abc12345" {
		t.Errorf("Expected correct destination, got '%s'", spec.Destination)
	}
	if spec.AppID != "app-123" {
		t.Errorf("Expected app ID 'app-123', got '%s'", spec.AppID)
	}
}

func TestKanikoJobManifest(t *testing.T) {
	app := &entities.App{
		ID:        "app-123",
		Name:      "web",
		Subdomain: "web",
	}
	spec := NewKanikoJobSpec(app, "deploy-abc", "reg/web:latest", "/workspace")
	manifest := spec.ToK8sJobManifest()

	// Verify it can be serialized to JSON
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to serialize manifest: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	if parsed["apiVersion"] != "batch/v1" {
		t.Errorf("Expected apiVersion 'batch/v1', got '%v'", parsed["apiVersion"])
	}
	if parsed["kind"] != "Job" {
		t.Errorf("Expected kind 'Job', got '%v'", parsed["kind"])
	}
}

func TestKanikoJobManifestLabels(t *testing.T) {
	app := &entities.App{
		ID:        "app-123",
		Name:      "web",
		Subdomain: "web",
	}
	spec := NewKanikoJobSpec(app, "deploy-xyz", "reg/web:xyz", "/ws")
	manifest := spec.ToK8sJobManifest()

	metadata := manifest["metadata"].(map[string]interface{})
	labels := metadata["labels"].(map[string]string)

	if labels["zenith.dev/component"] != "build" {
		t.Errorf("Expected component label 'build'")
	}
	if labels["zenith.dev/app-id"] != "app-123" {
		t.Errorf("Expected app-id label")
	}
}

// --- Pipeline tests ---

func TestPipelineRunningCount(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	builder := NewBuilder(repo, "/tmp/test-builds", "test-registry", nil, nil)
	pipeline := NewPipeline(builder, nil, repo, nil, nil)

	if pipeline.RunningCount() != 0 {
		t.Errorf("Expected 0 running builds, got %d", pipeline.RunningCount())
	}
}

func TestPipelineIsRunning(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	builder := NewBuilder(repo, "/tmp/test-builds", "test-registry", nil, nil)
	pipeline := NewPipeline(builder, nil, repo, nil, nil)

	if pipeline.IsRunning("nonexistent") {
		t.Error("Expected false for nonexistent deployment")
	}
}

func TestPipelineCancelNonExistent(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	builder := NewBuilder(repo, "/tmp/test-builds", "test-registry", nil, nil)
	pipeline := NewPipeline(builder, nil, repo, nil, nil)

	err := pipeline.CancelBuild("nonexistent")
	if err == nil {
		t.Error("Expected error when cancelling nonexistent build")
	}
}

func TestMinHelper(t *testing.T) {
	if min(3, 5) != 3 {
		t.Error("min(3,5) should be 3")
	}
	if min(5, 3) != 3 {
		t.Error("min(5,3) should be 3")
	}
	if min(3, 3) != 3 {
		t.Error("min(3,3) should be 3")
	}
}
