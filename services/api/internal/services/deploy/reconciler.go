package deploy

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
)

// AppRuntime is what the reconciler needs from an app backend.
type AppRuntime interface {
	AppContainerRunning(ctx context.Context, app *entities.App) (bool, error)
	DeployApp(ctx context.Context, app *entities.App, image string) error
}

// DBProber reports whether a database instance is up.
type DBProber interface {
	ContainerRunning(ctx context.Context, db *entities.UserDatabase) (bool, error)
}

// DBReprovisioner recreates a database instance from its record.
type DBReprovisioner interface {
	Reprovision(ctx context.Context, db *entities.UserDatabase) error
}

// Reconciler brings apps and databases back to their recorded state after
// the host restarts, a container is removed, or a backup is restored onto a
// new server. Databases first, so apps start against live databases.
type Reconciler struct {
	apps    ports.AppRepository
	dbs     ports.DatabaseRepository
	appRT   AppRuntime
	dbProbe DBProber
	dbRT    DBReprovisioner
}

func NewReconciler(apps ports.AppRepository, dbs ports.DatabaseRepository, appRT AppRuntime, dbProbe DBProber, dbRT DBReprovisioner) *Reconciler {
	return &Reconciler{apps: apps, dbs: dbs, appRT: appRT, dbProbe: dbProbe, dbRT: dbRT}
}

// Run reconciles every item once. Each failure is logged; none stops the rest.
func (r *Reconciler) Run(ctx context.Context) {
	dbs, err := r.dbs.ListDatabasesByStatus(ctx, entities.DatabaseStatusReady)
	if err != nil {
		slog.Error("reconcile: list databases", "error", err)
	}
	hp, canHostPort := r.dbProbe.(services.HostPorter)
	for i := range dbs {
		db := &dbs[i]
		up, err := r.dbProbe.ContainerRunning(ctx, db)
		if err != nil {
			slog.Warn("reconcile: database probe failed", "db", db.Name, "error", err)
			continue
		}
		if up {
			// Docker may pick a new loopback port when a container restarts.
			if canHostPort && r.refreshHostPort(ctx, hp, db) {
				slog.Info("reconcile: database has no loopback port; re-provisioning", "db", db.Name)
				if err := r.dbRT.Reprovision(ctx, db); err != nil {
					slog.Error("reconcile: database failed", "db", db.Name, "error", err)
				}
			}
			continue
		}
		slog.Info("reconcile: re-provisioning database", "db", db.Name)
		if err := r.dbRT.Reprovision(ctx, db); err != nil {
			slog.Error("reconcile: database failed", "db", db.Name, "error", err)
		}
	}

	apps, err := r.apps.ListAllApps(ctx)
	if err != nil {
		slog.Error("reconcile: list apps", "error", err)
		return
	}
	for i := range apps {
		app := &apps[i]
		if app.Status != entities.AppStatusRunning {
			continue
		}
		up, err := r.appRT.AppContainerRunning(ctx, app)
		if err != nil {
			slog.Warn("reconcile: app probe failed", "app", app.Name, "error", err)
			continue
		}
		if up {
			continue
		}
		image, err := CurrentImage(ctx, r.apps, app.ID)
		if err != nil || image == "" {
			slog.Warn("reconcile: app has no image to redeploy", "app", app.Name)
			continue
		}
		slog.Info("reconcile: redeploying app", "app", app.Name, "image", image)
		if err := r.appRT.DeployApp(ctx, app, image); err != nil {
			slog.Error("reconcile: app failed", "app", app.Name, "error", err)
		}
	}
}

// refreshHostPort stores the current loopback port. It reports true when the
// running container has no loopback binding at all and must be re-provisioned.
func (r *Reconciler) refreshHostPort(ctx context.Context, hp services.HostPorter, db *entities.UserDatabase) (needsReprovision bool) {
	port, err := hp.HostPort(ctx, db)
	if err != nil {
		if errors.Is(err, ErrNoLoopbackPort) {
			return true
		}
		slog.Warn("reconcile: database host port unknown", "db", db.Name, "error", err)
		return false
	}
	if port == db.HostPort {
		return false
	}
	if err := r.dbs.SetDatabaseHostPort(ctx, db.ID, port); err != nil {
		slog.Warn("store database host port failed", "db", db.Name, "error", err)
	}
	return false
}
