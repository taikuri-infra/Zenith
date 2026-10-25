package deploy

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type fakeAppRT struct {
	deployed map[string]bool
	redeploy []string
}

func (f *fakeAppRT) AppDeployed(_ context.Context, a *entities.App) (bool, error) {
	return f.deployed[a.ID], nil
}
func (f *fakeAppRT) DeployApp(_ context.Context, a *entities.App, image string) error {
	f.redeploy = append(f.redeploy, a.ID+"="+image)
	return nil
}

type fakeDBs struct {
	exists map[string]bool
	redone []string
}

func (f *fakeDBs) Exists(_ context.Context, db *entities.UserDatabase) (bool, error) {
	return f.exists[db.ID], nil
}
func (f *fakeDBs) Reprovision(_ context.Context, db *entities.UserDatabase) error {
	f.redone = append(f.redone, db.ID)
	return nil
}

func readyDB(t *testing.T, repo *memory.MemoryDatabaseRepository, name string) *entities.UserDatabase {
	t.Helper()
	db, err := repo.CreateDatabase(context.Background(), "app-"+name, "u1", &dto.CreateDatabaseInput{Name: name, Engine: entities.DatabaseEnginePostgres})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateDatabaseStatus(context.Background(), db.ID, entities.DatabaseStatusReady); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestReconciler_ReprovisionsOnlyMissingDatabases(t *testing.T) {
	apps := memory.NewMemoryAppRepository()
	dbs := memory.NewMemoryDatabaseRepository()
	present := readyDB(t, dbs, "present")
	missing := readyDB(t, dbs, "missing")
	d := &fakeDBs{exists: map[string]bool{present.ID: true}}

	NewReconciler(apps, dbs, &fakeAppRT{}, d, d).Run(context.Background())

	if len(d.redone) != 1 || d.redone[0] != missing.ID {
		t.Fatalf("reprovisioned %v, want only %s", d.redone, missing.ID)
	}
}

func TestReconciler_NothingRecordedDoesNothing(t *testing.T) {
	d := &fakeDBs{exists: map[string]bool{}}
	rt := &fakeAppRT{}
	NewReconciler(memory.NewMemoryAppRepository(), memory.NewMemoryDatabaseRepository(), rt, d, d).Run(context.Background())
	if len(d.redone) != 0 || len(rt.redeploy) != 0 {
		t.Fatalf("unexpected work: %v %v", d.redone, rt.redeploy)
	}
}
