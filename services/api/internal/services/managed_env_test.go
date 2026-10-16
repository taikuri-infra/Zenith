// services/api/internal/services/managed_env_test.go
package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type prefixEncrypter struct{}

func (prefixEncrypter) Encrypt(userID, value string) (string, error) {
	return "enc:" + userID + ":" + value, nil
}

func managedEnvFixture(t *testing.T) (*ManagedEnv, *memory.MemoryEnvVarRepository, *memory.MemoryAppRepository, string) {
	t.Helper()
	vars := memory.NewMemoryEnvVarRepository()
	apps := memory.NewMemoryAppRepository()
	app, err := apps.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "user-1", Name: "web", RepoURL: "https://github.com/u/r"})
	if err != nil {
		t.Fatal(err)
	}
	return NewManagedEnv(vars, apps), vars, apps, app.ID
}

func readyDB(t *testing.T, dbs *memory.MemoryDatabaseRepository, appID string, engine entities.DatabaseEngine, name string) *entities.UserDatabase {
	t.Helper()
	db, err := dbs.CreateDatabase(context.Background(), appID, "user-1", &dto.CreateDatabaseInput{Name: name, Engine: engine})
	if err != nil {
		t.Fatal(err)
	}
	if err := dbs.UpdateDatabaseStatus(context.Background(), db.ID, entities.DatabaseStatusReady); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestManagedEnv_SetWritesEncryptedSecretWithSource(t *testing.T) {
	env, vars, _, appID := managedEnvFixture(t)
	env.SetEncrypter(prefixEncrypter{})
	if err := env.Set(context.Background(), appID, "DATABASE_URL", "postgres://x", "db-1"); err != nil {
		t.Fatal(err)
	}
	got, _ := vars.GetEnvVarsByEnvironment(context.Background(), appID, "")
	if len(got) != 1 || got[0].Key != "DATABASE_URL" || got[0].Value != "enc:user-1:postgres://x" || !got[0].IsSecret ||
		got[0].Source != entities.EnvVarSourceManagedService || got[0].SourceID != "db-1" || got[0].EnvironmentID != "" {
		t.Fatalf("got %+v", got)
	}
	has, _ := env.Has(context.Background(), appID, "db-1")
	if !has {
		t.Fatal("Has = false")
	}
}

func TestManagedEnv_ClearRemovesOnlyThatDatabase(t *testing.T) {
	env, vars, _, appID := managedEnvFixture(t)
	_ = env.Set(context.Background(), appID, "DATABASE_URL", "pg", "db-1")
	_ = env.Set(context.Background(), appID, "REDIS_URL", "redis", "db-2")
	_ = vars.SetEnvVar(context.Background(), &entities.AppEnvVar{AppID: appID, Key: "MANUAL", Value: "x", Source: entities.EnvVarSourceManual})
	if err := env.Clear(context.Background(), appID, "db-1"); err != nil {
		t.Fatal(err)
	}
	got, _ := vars.GetEnvVarsByEnvironment(context.Background(), appID, "")
	keys := map[string]bool{}
	for _, v := range got {
		keys[v.Key] = true
	}
	if keys["DATABASE_URL"] || !keys["REDIS_URL"] || !keys["MANUAL"] {
		t.Fatalf("keys after Clear = %v", keys)
	}
}

func TestMigrateManagedEnv_CopiesLegacyAndDeletesIt(t *testing.T) {
	env, vars, apps, appID := managedEnvFixture(t)
	dbs := memory.NewMemoryDatabaseRepository()
	db := readyDB(t, dbs, appID, entities.DatabaseEnginePostgres, "pg1")
	_ = apps.SetEnvVars(context.Background(), appID, map[string]string{"DATABASE_URL": "enc:v1:abc", "OTHER": "keep"})

	n, err := MigrateManagedEnv(context.Background(), dbs, apps, env)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	v2, _ := vars.GetEnvVarsByEnvironment(context.Background(), appID, "")
	if len(v2) != 1 || v2[0].Value != "enc:v1:abc" || v2[0].SourceID != db.ID || v2[0].Source != entities.EnvVarSourceManagedService {
		t.Fatalf("v2 = %+v", v2)
	}
	legacy, _ := apps.GetEnvVars(context.Background(), appID)
	if len(legacy) != 1 || legacy[0].Key != "OTHER" {
		t.Fatalf("legacy = %+v (DATABASE_URL must be gone, OTHER kept)", legacy)
	}
}

func TestMigrateManagedEnv_Idempotent(t *testing.T) {
	env, vars, apps, appID := managedEnvFixture(t)
	dbs := memory.NewMemoryDatabaseRepository()
	readyDB(t, dbs, appID, entities.DatabaseEngineRedis, "r1")
	_ = apps.SetEnvVars(context.Background(), appID, map[string]string{"REDIS_URL": "redis://x"})
	if n, _ := MigrateManagedEnv(context.Background(), dbs, apps, env); n != 1 {
		t.Fatalf("first run n=%d", n)
	}
	if n, _ := MigrateManagedEnv(context.Background(), dbs, apps, env); n != 0 {
		t.Fatalf("second run n=%d, want 0", n)
	}
	v2, _ := vars.GetEnvVarsByEnvironment(context.Background(), appID, "")
	if len(v2) != 1 {
		t.Fatalf("v2 = %+v", v2)
	}
}

func TestMigrateManagedEnv_SkipsUnattachedAndNotReady(t *testing.T) {
	env, _, apps, appID := managedEnvFixture(t)
	dbs := memory.NewMemoryDatabaseRepository()
	readyDB(t, dbs, "", entities.DatabaseEnginePostgres, "x") // ready but unattached
	y, err := dbs.CreateDatabase(context.Background(), appID, "user-1", &dto.CreateDatabaseInput{Name: "y", Engine: entities.DatabaseEnginePostgres})
	if err != nil {
		t.Fatal(err)
	}
	// the memory repo creates databases ready; force the provisioning state
	if err := dbs.UpdateDatabaseStatus(context.Background(), y.ID, entities.DatabaseStatusProvisioning); err != nil {
		t.Fatal(err)
	}
	_ = apps.SetEnvVars(context.Background(), appID, map[string]string{"DATABASE_URL": "pg"})
	if n, err := MigrateManagedEnv(context.Background(), dbs, apps, env); n != 0 || err != nil {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
