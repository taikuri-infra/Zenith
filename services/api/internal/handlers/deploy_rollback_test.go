package handlers_test

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

type recordingImagePipeline struct{ images []string }

func (p *recordingImagePipeline) TriggerImageDeploy(_ *entities.App, _ *entities.Deployment, image string) error {
	p.images = append(p.images, image)
	return nil
}

func rollbackTo(t *testing.T, gitSHA string) (*recordingImagePipeline, int) {
	t.Helper()
	ctx := context.Background()
	repo := memory.NewMemoryAppRepository()
	a, _ := repo.CreateApp(ctx, &dto.CreateAppInput{UserID: "user-1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})
	// Pre-2a row: image_tag empty, image (or not) in git_sha.
	old, _ := repo.CreateDeployment(ctx, a.ID, gitSHA)
	_ = repo.UpdateDeploymentStatus(ctx, old.ID, entities.DeployStatusSuperseded, "", "")

	pipe := &recordingImagePipeline{}
	h := handlers.NewDeployHandler(repo, pipe)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Post("/apps/:appId/rollback", h.Rollback)

	req := httptest.NewRequest("POST", "/apps/"+a.ID+"/rollback", bytes.NewBufferString(`{"deployment_id":"`+old.ID+`"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	return pipe, resp.StatusCode
}

func TestRollbackToLegacyRowUsesImageFromGitSHA(t *testing.T) {
	pipe, code := rollbackTo(t, "docker.io/traefik/whoami:latest")
	if code != 200 {
		t.Fatalf("status %d", code)
	}
	if len(pipe.images) != 1 || pipe.images[0] != "docker.io/traefik/whoami:latest" {
		t.Fatalf("pipeline got %v", pipe.images)
	}
}

func TestRollbackToLegacyRowWithoutImageIsRejected(t *testing.T) {
	pipe, code := rollbackTo(t, "project-deploy")
	if code != 400 {
		t.Fatalf("status %d, want 400", code)
	}
	if len(pipe.images) != 0 {
		t.Fatalf("pipeline must not be called, got %v", pipe.images)
	}
}
