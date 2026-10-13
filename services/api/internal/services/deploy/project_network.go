package deploy

import (
	"context"
	"log/slog"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

const projectLabel = "zenith.project.id"

// projectNetworkName is the per-project bridge network. Only that project's
// app containers join it, so compose service names (db, api, postgres...)
// resolve to the project's own services.
func projectNetworkName(projectID string) string {
	id := sanitizeName(projectID)
	if id == "" {
		return ""
	}
	if len(id) > 12 {
		id = id[:12]
	}
	return "zenith-proj-" + id
}

// projectAlias is the name siblings in the project use for this app: its
// compose service name, else (apps created before service_name existed) the
// app name without the "<project slug>-" prefix the compose flow adds.
func projectAlias(app *entities.App, projectSlug string) string {
	if s := sanitizeName(app.ServiceName); s != "" {
		return s
	}
	if projectSlug == "" {
		return ""
	}
	rest, ok := strings.CutPrefix(sanitizeName(app.Name), sanitizeName(projectSlug)+"-")
	if !ok {
		return ""
	}
	return rest
}

// SetProjectRepo lets the deployer read project slugs for the alias fallback.
func (d *DockerDeployer) SetProjectRepo(r ports.ProjectRepository) { d.projectRepo = r }

func (d *DockerDeployer) projectSlug(ctx context.Context, app *entities.App) string {
	if d.projectRepo == nil || app.ProjectID == "" {
		return ""
	}
	p, err := d.projectRepo.GetProject(ctx, app.ProjectID)
	if err != nil || p == nil {
		return ""
	}
	return p.Slug
}

// ensureProjectNetwork creates the project network if it doesn't exist.
func (d *DockerDeployer) ensureProjectNetwork(ctx context.Context, app *entities.App) (string, error) {
	name := projectNetworkName(app.ProjectID)
	if name == "" {
		return "", nil
	}
	if _, err := d.cli.NetworkInspect(ctx, name, network.InspectOptions{}); err == nil {
		return name, nil
	} else if !client.IsErrNotFound(err) {
		return "", err
	}
	_, err := d.cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
		Labels: map[string]string{projectLabel: app.ProjectID},
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return "", err
	}
	return name, nil
}

// removeProjectNetworkIfUnused drops the project network once no app
// container of the project is left. Failures are logged, never returned.
func (d *DockerDeployer) removeProjectNetworkIfUnused(ctx context.Context, projectID string) {
	name := projectNetworkName(projectID)
	if name == "" {
		return
	}
	left, err := d.cli.ContainerList(ctx, container.ListOptions{All: true,
		Filters: filters.NewArgs(filters.Arg("label", projectLabel+"="+projectID))})
	if err != nil || len(left) > 0 {
		return
	}
	if err := d.cli.NetworkRemove(ctx, name); err != nil && !client.IsErrNotFound(err) {
		slog.Warn("docker-deploy: remove project network failed", "network", name, "error", err)
	}
}
