package deploy

import (
	"context"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// Backend is the compute backend that actually runs a user's app. Zenith runs
// on Kubernetes only, so Deployer is the one implementation; the pipeline and
// handlers depend on this interface so tests can substitute a fake.
type Backend interface {
	// DeployApp creates or updates the running app for the given image tag.
	DeployApp(ctx context.Context, app *entities.App, imageTag string) error
	// DeleteApp tears down the running app.
	DeleteApp(ctx context.Context, app *entities.App) error
}

// Compile-time assertion that the Kubernetes deployer satisfies Backend.
var _ Backend = (*Deployer)(nil)
