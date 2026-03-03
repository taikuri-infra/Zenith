package deploy

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
)

func makeTestApp() *entities.App {
	return &entities.App{
		ID:        "app-test-1",
		Name:      "testapp",
		Subdomain: "testapp",
		UserID:    "user-1",
		RepoURL:   "https://github.com/test/repo",
		Branch:    "main",
		Port:      8080,
	}
}

func makeTestJobSpec(app *entities.App) *KanikoJobSpec {
	return NewKanikoJobSpec(app, "deploy-abc12345", "registry.example.com/testapp:abc12345")
}

// TestKanikoRunnerNilClient ensures a nil KanikoRunner is a no-op (dev mode).
func TestKanikoRunnerNilClient(t *testing.T) {
	var runner *KanikoRunner // nil
	app := makeTestApp()
	spec := makeTestJobSpec(app)

	err := runner.Build(context.Background(), spec, "deploy-abc12345")
	if err != nil {
		t.Errorf("expected nil error from nil runner, got: %v", err)
	}
}

// TestNewKanikoRunnerNilClient ensures NewKanikoRunner returns nil when k8sClient is nil.
func TestNewKanikoRunnerNilClient(t *testing.T) {
	runner := NewKanikoRunner(nil, nil)
	if runner != nil {
		t.Error("expected nil runner when k8sClient is nil")
	}
}

// TestKanikoRunnerBuildSuccess uses MemoryClient to simulate a successful build.
func TestKanikoRunnerBuildSuccess(t *testing.T) {
	hub := NewLogHub(100)
	k8sClient := k8sclient.NewMemoryClient()
	runner := NewKanikoRunner(k8sClient, hub)

	if runner == nil {
		t.Fatal("expected non-nil runner")
	}

	app := makeTestApp()
	spec := makeTestJobSpec(app)
	deploymentID := "deploy-abc12345"

	ctx := context.Background()
	err := runner.Build(ctx, spec, deploymentID)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Job should be cleaned up (deleted after success)
	_, getErr := k8sClient.GetJob(ctx, "zenith-builds", spec.Name)
	if getErr == nil {
		t.Error("expected job to be deleted after success, but it still exists")
	}
}

// TestKanikoRunnerLogsEmitted confirms that MemoryClient fake logs are emitted to LogHub.
func TestKanikoRunnerLogsEmitted(t *testing.T) {
	hub := NewLogHub(100)
	k8sClient := k8sclient.NewMemoryClient()
	runner := NewKanikoRunner(k8sClient, hub)

	app := makeTestApp()
	spec := makeTestJobSpec(app)
	deploymentID := "deploy-log-test"

	if err := runner.Build(context.Background(), spec, deploymentID); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	history := hub.History(deploymentID)
	if len(history) == 0 {
		t.Error("expected log entries to be emitted to LogHub, got none")
	}
}

// TestKanikoRunnerBuilderIntegration tests the full Builder → KanikoRunner flow
// in dev (repository-only) mode using MemoryClient (no real clone).
func TestKanikoRunnerBuilderIntegration(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	hub := NewLogHub(100)
	k8sClient := k8sclient.NewMemoryClient()
	// Build with k8sClient — but we won't actually call BuildApp (no git)
	// Just verify NewBuilder constructs correctly with the new signature.
	builder := NewBuilder(repo, "/tmp/test", "registry.example.com", k8sClient, hub)
	if builder == nil {
		t.Fatal("expected non-nil builder")
	}
	if builder.kanikoRunner == nil {
		t.Error("expected non-nil kanikoRunner when k8sClient is provided")
	}
}
