package deploy

import (
	"context"
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// AppRuntime is what the reconciler needs from an app backend.
type AppRuntime interface {
	AppDeployed(ctx context.Context, app *entities.App) (bool, error)
	DeployApp(ctx context.Context, app *entities.App, image string) error
}

// DBProber reports whether a database's operator resource exists.
type DBProber interface {
	Exists(ctx context.Context, db *entities.UserDatabase) (bool, error)
}

// DBReprovisioner recreates a database instance from its record.
type DBReprovisioner interface {
	Reprovision(ctx context.Context, db *entities.UserDatabase) error
}

// Reconciler brings apps and databases back to their recorded state after a
// backup is restored onto a new cluster or a resource is deleted by hand.
// Databases first, so apps start against live databases.
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
	for i := range dbs {
		db := &dbs[i]
		up, err := r.dbProbe.Exists(ctx, db)
		if err != nil {
			slog.Warn("reconcile: database probe failed", "db", db.Name, "error", err)
			continue
		}
		if up {
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
		up, err := r.appRT.AppDeployed(ctx, app)
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

// AppDeployed reports whether the app's Deployment exists.
func (d *Deployer) AppDeployed(ctx context.Context, app *entities.App) (bool, error) {
	_, err := d.k8sClient.GetCRDWithVersion(ctx, "apps/v1", "Deployment", d.namespaceFor(ctx, app), app.Subdomain)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Exists reports whether the database's operator resource exists.
func (b *K8sDBBackend) Exists(ctx context.Context, db *entities.UserDatabase) (bool, error) {
	apiVersion, kind := dbKind(db.Engine)
	_, err := b.client.GetCRDWithVersion(ctx, apiVersion, kind, b.nsFor(ctx, db), k8sDBName(db))
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isNotFound(err error) bool { return ignoreNotFound(err) == nil }
