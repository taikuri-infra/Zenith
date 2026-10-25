package deploy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type stubBackend struct {
	err  error
	repo *memory.MemoryAppRepository
	set  entities.AppStatus
}

func (s *stubBackend) DeployApp(ctx context.Context, app *entities.App, _ string) error {
	if s.set != "" {
		st := s.set
		s.repo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{Status: &st})
	}
	return s.err
}
func (s *stubBackend) DeleteApp(context.Context, *entities.App) error { return nil }

func runDeploy(t *testing.T, backendErr error) (*memory.MemoryAppRepository, *entities.App, *entities.Deployment) {
	return runDeployWith(t, backendErr, "")
}

func runDeployWith(t *testing.T, backendErr error, set entities.AppStatus) (*memory.MemoryAppRepository, *entities.App, *entities.Deployment) {
	t.Helper()
	repo := memory.NewMemoryAppRepository()
	app, err := repo.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "u1", Name: "web", DeploySource: entities.DeploySourceImage, ImageURL: "nginx:1"})
	if err != nil {
		t.Fatal(err)
	}
	dep, _ := repo.CreateDeployment(context.Background(), app.ID, "nginx:1")
	p := NewPipeline(&stubBackend{err: backendErr, repo: repo, set: set}, repo, NewLogHub(50), nil, 5)
	if err := p.TriggerImageDeploy(app, dep, "nginx:1"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for p.RunningCount() > 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	return repo, app, dep
}

func TestPipeline_RecordsImageAndLeavesStatusToBackend(t *testing.T) {
	repo, app, dep := runDeploy(t, nil)
	got, _ := repo.GetDeployment(context.Background(), dep.ID)
	if got.ImageTag != "nginx:1" {
		t.Errorf("image_tag = %q, want nginx:1", got.ImageTag)
	}
	a, _ := repo.GetApp(context.Background(), app.ID)
	if a.Status == entities.AppStatusRunning {
		t.Errorf("pipeline must not overwrite backend-owned status, got %q", a.Status)
	}
}

func TestPipeline_KeepsBackendSleepingStatus(t *testing.T) {
	repo, app, _ := runDeployWith(t, nil, entities.AppStatusSleeping)
	a, _ := repo.GetApp(context.Background(), app.ID)
	if a.Status != entities.AppStatusSleeping {
		t.Errorf("app status = %q, want sleeping", a.Status)
	}
}

func TestPipeline_FailureMarksAppFailed(t *testing.T) {
	repo, app, dep := runDeploy(t, errors.New("pull denied"))
	a, _ := repo.GetApp(context.Background(), app.ID)
	if a.Status != entities.AppStatusFailed {
		t.Errorf("app status = %q, want failed", a.Status)
	}
	got, _ := repo.GetDeployment(context.Background(), dep.ID)
	if got.Error == "" {
		t.Error("deployment error not recorded")
	}
}

func TestPipeline_FailedEventCarriesError(t *testing.T) {
	data := deployEventData(&entities.App{ID: "a", Name: "web"}, &entities.Deployment{ID: "d"}, "img", "Deploy failed: boom", EventDeployFailed)
	if data["error"] != "Deploy failed: boom" {
		t.Errorf("error key = %v", data["error"])
	}
}
