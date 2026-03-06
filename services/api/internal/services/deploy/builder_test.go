package deploy

import (
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
)

func TestPipelineRunningCount(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	pipeline := NewPipeline(nil, repo, nil, nil, 5)

	if pipeline.RunningCount() != 0 {
		t.Errorf("Expected 0 running builds, got %d", pipeline.RunningCount())
	}
}

func TestPipelineIsRunning(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	pipeline := NewPipeline(nil, repo, nil, nil, 5)

	if pipeline.IsRunning("nonexistent") {
		t.Error("Expected false for nonexistent deployment")
	}
}

func TestPipelineCancelNonExistent(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	pipeline := NewPipeline(nil, repo, nil, nil, 5)

	err := pipeline.CancelBuild("nonexistent")
	if err == nil {
		t.Error("Expected error when cancelling nonexistent build")
	}
}
