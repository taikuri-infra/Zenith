package deploy

import (
	"context"
	"errors"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestCurrentImage(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMemoryAppRepository()
	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{UserID: "u1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})

	if _, err := CurrentImage(ctx, repo, app.ID); !errors.Is(err, ErrNoImage) {
		t.Fatalf("want ErrNoImage, got %v", err)
	}

	d1, _ := repo.CreateDeployment(ctx, app.ID, "nginx:1")
	_ = repo.UpdateDeploymentStatus(ctx, d1.ID, entities.DeployStatusActive, "", "")
	d2, _ := repo.CreateDeployment(ctx, app.ID, "nginx:2")
	_ = repo.SetDeploymentImage(ctx, d2.ID, "nginx:2")
	_ = repo.UpdateDeploymentStatus(ctx, d2.ID, entities.DeployStatusFailed, "", "x")

	img, err := CurrentImage(ctx, repo, app.ID)
	if err != nil || img != "nginx:1" {
		t.Fatalf("got %q, %v; want the active deployment's image nginx:1", img, err)
	}
}

func TestCurrentImageSkipsLegacyNonImageGitSHA(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMemoryAppRepository()
	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{UserID: "u1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})

	d, _ := repo.CreateDeployment(ctx, app.ID, "project-deploy")
	_ = repo.UpdateDeploymentStatus(ctx, d.ID, entities.DeployStatusActive, "", "")
	if _, err := repo.CreateRelease(ctx, app.ID, &dto.CreateReleaseInput{Image: "ghcr.io/acme/web:v3"}); err != nil {
		t.Fatal(err)
	}

	img, err := CurrentImage(ctx, repo, app.ID)
	if err != nil || img != "ghcr.io/acme/web:v3" {
		t.Fatalf("got %q, %v; want the release image", img, err)
	}
}
