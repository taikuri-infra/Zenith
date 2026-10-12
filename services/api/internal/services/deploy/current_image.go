package deploy

import (
	"context"
	"errors"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// ErrNoImage means the app has never been deployed with a known image.
var ErrNoImage = errors.New("no deployed image for app")

// CurrentImage returns the image the app is running: the active deployment's,
// else the newest deployment's, else the newest release's.
func CurrentImage(ctx context.Context, repo ports.AppRepository, appID string) (string, error) {
	if d, err := repo.GetActiveDeployment(ctx, appID); err == nil && d != nil && d.DeployedImage() != "" {
		return d.DeployedImage(), nil
	}
	if ds, err := repo.ListDeployments(ctx, appID, 20); err == nil {
		for _, d := range ds {
			if d.Status != entities.DeployStatusFailed && d.DeployedImage() != "" {
				return d.DeployedImage(), nil
			}
		}
	}
	if rs, err := repo.ListReleases(ctx, appID, 1); err == nil && len(rs) > 0 && rs[0].Image != "" {
		return rs[0].Image, nil
	}
	return "", ErrNoImage
}
