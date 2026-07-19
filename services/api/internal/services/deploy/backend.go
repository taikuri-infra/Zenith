package deploy

import (
	"context"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// Backend is the compute backend that actually runs a user's app. It has two
// implementations selected by ZENITH_MODE:
//   - Deployer (Kubernetes) for saas/cloud/enterprise
//   - DockerDeployer (plain Docker containers on the host) for standalone self-host
//
// The pipeline and handlers depend on this interface, not a concrete type, so the
// backend can be swapped without touching the deploy flow.
type Backend interface {
	// DeployApp creates or updates the running app for the given image tag.
	DeployApp(ctx context.Context, app *entities.App, imageTag string) error
	// DeleteApp tears down the running app.
	DeleteApp(ctx context.Context, app *entities.App) error
}

// Compile-time assertion that the Kubernetes deployer satisfies Backend.
var _ Backend = (*Deployer)(nil)
