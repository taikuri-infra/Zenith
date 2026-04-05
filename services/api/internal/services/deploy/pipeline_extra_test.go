package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestNewPipeline_DefaultMaxConcurrent(t *testing.T) {
	pipeline := NewPipeline(nil, nil, nil, nil, 0)
	if pipeline.maxConcurrent != 5 {
		t.Errorf("Expected default maxConcurrent 5, got %d", pipeline.maxConcurrent)
	}
}

func TestNewPipeline_NegativeMaxConcurrent(t *testing.T) {
	pipeline := NewPipeline(nil, nil, nil, nil, -3)
	if pipeline.maxConcurrent != 5 {
		t.Errorf("Expected default maxConcurrent 5, got %d", pipeline.maxConcurrent)
	}
}

func TestNewPipeline_CustomMaxConcurrent(t *testing.T) {
	pipeline := NewPipeline(nil, nil, nil, nil, 10)
	if pipeline.maxConcurrent != 10 {
		t.Errorf("Expected maxConcurrent 10, got %d", pipeline.maxConcurrent)
	}
}

func TestPipeline_LogHub_Accessor(t *testing.T) {
	logHub := NewLogHub(100)
	pipeline := NewPipeline(nil, nil, logHub, nil, 5)
	if pipeline.LogHub() != logHub {
		t.Error("LogHub() should return the configured log hub")
	}
}

func TestPipeline_LogHub_Nil(t *testing.T) {
	pipeline := NewPipeline(nil, nil, nil, nil, 5)
	if pipeline.LogHub() != nil {
		t.Error("LogHub() should return nil when not configured")
	}
}

func TestPipeline_EmitLog_WithLogHub(t *testing.T) {
	logHub := NewLogHub(100)
	pipeline := NewPipeline(nil, nil, logHub, nil, 5)

	pipeline.emitLog("deploy-1", "info", "starting deploy")

	history := logHub.History("deploy-1")
	if len(history) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(history))
	}
	if history[0].Level != "info" {
		t.Errorf("Expected level 'info', got '%s'", history[0].Level)
	}
	if history[0].Message != "starting deploy" {
		t.Errorf("Expected message 'starting deploy', got '%s'", history[0].Message)
	}
}

func TestPipeline_EmitLog_NilLogHub(t *testing.T) {
	pipeline := NewPipeline(nil, nil, nil, nil, 5)
	// Should not panic
	pipeline.emitLog("deploy-1", "info", "should not panic")
}

func TestPipeline_SetEventBus(t *testing.T) {
	pipeline := NewPipeline(nil, nil, nil, nil, 5)
	if pipeline.eventBus != nil {
		t.Error("Expected nil eventBus initially")
	}
	// SetEventBus with nil should not panic
	pipeline.SetEventBus(nil)
}

func TestPipeline_SetWebhookService(t *testing.T) {
	pipeline := NewPipeline(nil, nil, nil, nil, 5)
	if pipeline.webhookSvc != nil {
		t.Error("Expected nil webhookSvc initially")
	}
	// SetWebhookService with nil should not panic
	pipeline.SetWebhookService(nil)
}

func TestPipeline_SetHookRepo(t *testing.T) {
	pipeline := NewPipeline(nil, nil, nil, nil, 5)
	if pipeline.hookRepo != nil {
		t.Error("Expected nil hookRepo initially")
	}
	pipeline.SetHookRepo(nil)
}

func TestPipeline_TriggerImageDeploy_QueueFull(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	pipeline := NewPipeline(nil, repo, nil, nil, 1)

	// Manually fill the running map
	pipeline.mu.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	_ = ctx
	pipeline.running["existing-deploy"] = &runningBuild{Cancel: cancel, UserID: "user-1"}
	pipeline.mu.Unlock()

	app := &entities.App{ID: "app-1", UserID: "user-2"}
	deployment := &entities.Deployment{ID: "deploy-new"}

	err := pipeline.TriggerImageDeploy(app, deployment, "image:v1")
	if err == nil {
		t.Error("Expected error when deploy queue is full")
	}

	// Clean up
	pipeline.mu.Lock()
	delete(pipeline.running, "existing-deploy")
	pipeline.mu.Unlock()
}

func TestPipeline_TriggerImageDeploy_UserLimitReached(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	pipeline := NewPipeline(nil, repo, nil, nil, 10)

	// Fill up with 2 deploys for user-1 (maxPerUser = 2)
	pipeline.mu.Lock()
	for i := 0; i < 2; i++ {
		_, cancel := context.WithCancel(context.Background())
		pipeline.running["deploy-"+string(rune('A'+i))] = &runningBuild{Cancel: cancel, UserID: "user-1"}
	}
	pipeline.mu.Unlock()

	app := &entities.App{ID: "app-1", UserID: "user-1"}
	deployment := &entities.Deployment{ID: "deploy-new"}

	err := pipeline.TriggerImageDeploy(app, deployment, "image:v1")
	if err == nil {
		t.Error("Expected error when user has too many concurrent deploys")
	}

	// Clean up
	pipeline.mu.Lock()
	for k, v := range pipeline.running {
		v.Cancel()
		delete(pipeline.running, k)
	}
	pipeline.mu.Unlock()
}

func TestPipeline_EmitEvent_WithEventHub(t *testing.T) {
	eventHub := NewEventHub(50)
	pipeline := NewPipeline(nil, nil, nil, eventHub, 5)

	app := &entities.App{ID: "app-1", Name: "web", UserID: "user-1"}
	deployment := &entities.Deployment{ID: "deploy-1", Status: entities.DeployStatusDeploying}

	pipeline.emitEvent(EventDeploymentStarted, app, deployment, "web:v1", "deploy started")

	history := eventHub.History()
	if len(history) != 1 {
		t.Fatalf("Expected 1 event in hub, got %d", len(history))
	}
	if history[0].Type != EventDeploymentStarted {
		t.Errorf("Expected event type %s, got %s", EventDeploymentStarted, history[0].Type)
	}
	if history[0].AppID != "app-1" {
		t.Errorf("Expected AppID 'app-1', got '%s'", history[0].AppID)
	}
	if history[0].DeploymentID != "deploy-1" {
		t.Errorf("Expected DeploymentID 'deploy-1', got '%s'", history[0].DeploymentID)
	}
}

func TestPipeline_EmitEvent_NilEventHub(t *testing.T) {
	pipeline := NewPipeline(nil, nil, nil, nil, 5)
	app := &entities.App{ID: "app-1", Name: "web"}
	deployment := &entities.Deployment{ID: "deploy-1"}

	// Should not panic with nil eventHub
	pipeline.emitEvent(EventDeployComplete, app, deployment, "web:v1", "done")
}

func TestPipeline_CancelBuild_Existing(t *testing.T) {
	pipeline := NewPipeline(nil, nil, nil, nil, 5)

	_, cancel := context.WithCancel(context.Background())
	pipeline.mu.Lock()
	pipeline.running["deploy-1"] = &runningBuild{Cancel: cancel, UserID: "user-1"}
	pipeline.mu.Unlock()

	if !pipeline.IsRunning("deploy-1") {
		t.Error("Expected deploy-1 to be running")
	}

	err := pipeline.CancelBuild("deploy-1")
	if err != nil {
		t.Fatalf("CancelBuild failed: %v", err)
	}

	if pipeline.IsRunning("deploy-1") {
		t.Error("Expected deploy-1 to not be running after cancel")
	}
	if pipeline.RunningCount() != 0 {
		t.Errorf("Expected 0 running after cancel, got %d", pipeline.RunningCount())
	}
}
