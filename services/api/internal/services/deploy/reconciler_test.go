package deploy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type fakeAppRT struct {
	mu       sync.Mutex
	running  map[string]bool
	deployed []string
}

func (f *fakeAppRT) AppContainerRunning(_ context.Context, a *entities.App) (bool, error) {
	return f.running[a.ID], nil
}
func (f *fakeAppRT) DeployApp(_ context.Context, a *entities.App, image string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deployed = append(f.deployed, a.ID+"@"+image)
	return nil
}

type fakeDBProbe struct{ running map[string]bool }

func (f fakeDBProbe) ContainerRunning(_ context.Context, db *entities.UserDatabase) (bool, error) {
	return f.running[db.ID], nil
}

// fakeHostPortProbe is a DBProber that also reports host ports.
type fakeHostPortProbe struct {
	fakeDBProbe
	ports map[string]int
	err   error
}

func (f fakeHostPortProbe) HostPort(_ context.Context, db *entities.UserDatabase) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.ports[db.ID], nil
}

type fakeDBRT struct {
	fail  map[string]bool
	calls []string
}

func (f *fakeDBRT) Reprovision(_ context.Context, db *entities.UserDatabase) error {
	f.calls = append(f.calls, db.ID)
	if f.fail[db.ID] {
		return errors.New("boom")
	}
	return nil
}

func seedApp(t *testing.T, repo *memory.MemoryAppRepository, name string, status entities.AppStatus, image string) *entities.App {
	t.Helper()
	ctx := context.Background()
	a, err := repo.CreateApp(ctx, &dto.CreateAppInput{UserID: "u1", Name: name, ImageURL: image, DeploySource: entities.DeploySourceImage})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateApp(ctx, a.ID, &dto.UpdateAppInput{Status: &status}); err != nil {
		t.Fatal(err)
	}
	// CurrentImage reads deployments, not app.ImageURL.
	d, err := repo.CreateDeployment(ctx, a.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetDeploymentImage(ctx, d.ID, image); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestReconciler_RedeploysOnlyStoppedRunningApps(t *testing.T) {
	apps := memory.NewMemoryAppRepository()
	up := seedApp(t, apps, "up", entities.AppStatusRunning, "nginx:1")
	down := seedApp(t, apps, "down", entities.AppStatusRunning, "nginx:2")
	_ = seedApp(t, apps, "stopped", entities.AppStatusStopped, "nginx:3")
	rt := &fakeAppRT{running: map[string]bool{up.ID: true}}
	NewReconciler(apps, memory.NewMemoryDatabaseRepository(), rt, fakeDBProbe{}, &fakeDBRT{}).Run(context.Background())
	if len(rt.deployed) != 1 || rt.deployed[0] != down.ID+"@nginx:2" {
		t.Fatalf("deployed = %v, want only %s@nginx:2", rt.deployed, down.ID)
	}
}

func TestReconciler_DBFailureDoesNotStopApps(t *testing.T) {
	ctx := context.Background()
	apps := memory.NewMemoryAppRepository()
	down := seedApp(t, apps, "down", entities.AppStatusRunning, "nginx:2")
	dbs := memory.NewMemoryDatabaseRepository()
	a, _ := dbs.CreateDatabase(ctx, "", "u1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "a"})
	b, _ := dbs.CreateDatabase(ctx, "", "u1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEngineRedis, Name: "b"})
	dbRT := &fakeDBRT{fail: map[string]bool{a.ID: true}}
	appRT := &fakeAppRT{running: map[string]bool{}}
	NewReconciler(apps, dbs, appRT, fakeDBProbe{running: map[string]bool{}}, dbRT).Run(ctx)
	if len(dbRT.calls) != 2 {
		t.Fatalf("reprovision calls = %v, want both %s and %s", dbRT.calls, a.ID, b.ID)
	}
	if len(appRT.deployed) != 1 || appRT.deployed[0] != down.ID+"@nginx:2" {
		t.Fatalf("apps not reconciled after a db failure: %v", appRT.deployed)
	}
}

func TestReconciler_RefreshesHostPortOfRunningDB(t *testing.T) {
	ctx := context.Background()
	dbs := memory.NewMemoryDatabaseRepository()
	db, _ := dbs.CreateDatabase(ctx, "", "u1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "a"})
	if err := dbs.SetDatabaseHostPort(ctx, db.ID, 49153); err != nil {
		t.Fatal(err)
	}
	probe := fakeHostPortProbe{
		fakeDBProbe: fakeDBProbe{running: map[string]bool{db.ID: true}},
		ports:       map[string]int{db.ID: 49300},
	}
	dbRT := &fakeDBRT{}
	NewReconciler(memory.NewMemoryAppRepository(), dbs, &fakeAppRT{}, probe, dbRT).Run(ctx)
	got, _ := dbs.GetDatabase(ctx, db.ID)
	if got.HostPort != 49300 {
		t.Fatalf("HostPort = %d, want 49300", got.HostPort)
	}
	if len(dbRT.calls) != 0 {
		t.Fatalf("running db was re-provisioned: %v", dbRT.calls)
	}
}

func TestReconciler_ReprovisionsRunningDBWithoutLoopbackPort(t *testing.T) {
	ctx := context.Background()
	dbs := memory.NewMemoryDatabaseRepository()
	db, _ := dbs.CreateDatabase(ctx, "", "u1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "a"})
	probe := fakeHostPortProbe{
		fakeDBProbe: fakeDBProbe{running: map[string]bool{db.ID: true}},
		err:         fmt.Errorf("zenith-db-a: %w", ErrNoLoopbackPort),
	}
	dbRT := &fakeDBRT{}
	NewReconciler(memory.NewMemoryAppRepository(), dbs, &fakeAppRT{}, probe, dbRT).Run(ctx)
	if len(dbRT.calls) != 1 || dbRT.calls[0] != db.ID {
		t.Fatalf("calls = %v, want one reprovision of %s", dbRT.calls, db.ID)
	}
}

func TestReconciler_OtherHostPortErrorDoesNotReprovision(t *testing.T) {
	ctx := context.Background()
	dbs := memory.NewMemoryDatabaseRepository()
	db, _ := dbs.CreateDatabase(ctx, "", "u1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "a"})
	probe := fakeHostPortProbe{
		fakeDBProbe: fakeDBProbe{running: map[string]bool{db.ID: true}},
		err:         errors.New("docker hiccup"),
	}
	dbRT := &fakeDBRT{}
	NewReconciler(memory.NewMemoryAppRepository(), dbs, &fakeAppRT{}, probe, dbRT).Run(ctx)
	if len(dbRT.calls) != 0 {
		t.Fatalf("unexpected reprovision: %v", dbRT.calls)
	}
}
