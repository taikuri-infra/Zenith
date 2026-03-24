package memory

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestEnvVarSetAndGet(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	ev := &entities.AppEnvVar{
		AppID: "app-1",
		Key:   "DATABASE_URL",
		Value: "postgres://localhost/db",
	}
	err := repo.SetEnvVar(ctx, ev)
	if err != nil {
		t.Fatalf("SetEnvVar failed: %v", err)
	}
	if ev.ID == "" {
		t.Error("Expected ID to be assigned")
	}

	vars, err := repo.GetEnvVars(ctx, "app-1")
	if err != nil {
		t.Fatalf("GetEnvVars failed: %v", err)
	}
	if len(vars) != 1 {
		t.Fatalf("Expected 1 var, got %d", len(vars))
	}
	if vars[0].Value != "postgres://localhost/db" {
		t.Errorf("Expected value 'postgres://localhost/db', got '%s'", vars[0].Value)
	}
}

func TestEnvVarUpsert(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "KEY", Value: "old"})
	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "KEY", Value: "new"})

	vars, _ := repo.GetEnvVars(ctx, "app-1")
	if len(vars) != 1 {
		t.Fatalf("Expected 1 var after upsert, got %d", len(vars))
	}
	if vars[0].Value != "new" {
		t.Errorf("Expected value 'new', got '%s'", vars[0].Value)
	}
}

func TestEnvVarGetSorted(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "Z_VAR", Value: "z"})
	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "A_VAR", Value: "a"})
	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "M_VAR", Value: "m"})

	vars, _ := repo.GetEnvVars(ctx, "app-1")
	if len(vars) != 3 {
		t.Fatalf("Expected 3 vars, got %d", len(vars))
	}
	if vars[0].Key != "A_VAR" || vars[2].Key != "Z_VAR" {
		t.Errorf("Expected sorted order A, M, Z; got %s, %s, %s", vars[0].Key, vars[1].Key, vars[2].Key)
	}
}

func TestEnvVarGetEmpty(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	vars, err := repo.GetEnvVars(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("Expected 0 vars, got %d", len(vars))
	}
}

func TestEnvVarDelete(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	ev := &entities.AppEnvVar{AppID: "app-1", Key: "KEY", Value: "val"}
	repo.SetEnvVar(ctx, ev)

	err := repo.DeleteEnvVar(ctx, ev.ID)
	if err != nil {
		t.Fatalf("DeleteEnvVar failed: %v", err)
	}

	vars, _ := repo.GetEnvVars(ctx, "app-1")
	if len(vars) != 0 {
		t.Errorf("Expected 0 vars after delete, got %d", len(vars))
	}
}

func TestEnvVarDeleteNotFound(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	err := repo.DeleteEnvVar(ctx, "nonexistent-id")
	if err == nil {
		t.Error("Expected error for nonexistent ID")
	}
}

func TestEnvVarGetByEnvironment_Production(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	// Production (empty environment_id)
	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "KEY", Value: "prod-val", EnvironmentID: ""})
	// Staging environment
	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "KEY", Value: "stage-val", EnvironmentID: "env-staging"})

	prod, _ := repo.GetEnvVarsByEnvironment(ctx, "app-1", "")
	if len(prod) != 1 || prod[0].Value != "prod-val" {
		t.Errorf("Expected 1 prod var with value 'prod-val', got %v", prod)
	}

	stage, _ := repo.GetEnvVarsByEnvironment(ctx, "app-1", "env-staging")
	if len(stage) != 1 || stage[0].Value != "stage-val" {
		t.Errorf("Expected 1 stage var with value 'stage-val', got %v", stage)
	}
}

func TestEnvVarGetByEnvironment_Isolation(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "A", Value: "1", EnvironmentID: ""})
	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "B", Value: "2", EnvironmentID: "env-stage"})
	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-2", Key: "A", Value: "3", EnvironmentID: ""})

	prod, _ := repo.GetEnvVarsByEnvironment(ctx, "app-1", "")
	if len(prod) != 1 {
		t.Errorf("Expected only app-1 prod vars, got %d", len(prod))
	}
}

func TestEnvVarGetDelegatesToEnvironment(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "KEY", Value: "prod", EnvironmentID: ""})
	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "KEY", Value: "stage", EnvironmentID: "env-1"})

	// GetEnvVars should return only production (empty env ID)
	vars, _ := repo.GetEnvVars(ctx, "app-1")
	if len(vars) != 1 || vars[0].Value != "prod" {
		t.Errorf("GetEnvVars should return production vars only, got %v", vars)
	}
}

func TestEnvVarBulkSet(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	vars := []entities.AppEnvVar{
		{Key: "A", Value: "1"},
		{Key: "B", Value: "2"},
		{Key: "C", Value: "3"},
	}
	err := repo.BulkSetEnvVars(ctx, "app-1", vars)
	if err != nil {
		t.Fatalf("BulkSetEnvVars failed: %v", err)
	}

	result, _ := repo.GetEnvVars(ctx, "app-1")
	if len(result) != 3 {
		t.Errorf("Expected 3 vars, got %d", len(result))
	}
}

func TestEnvVarDeleteBySource(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "MANUAL_KEY", Value: "v1", Source: entities.EnvVarSourceManual})
	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "SERVICE_KEY", Value: "v2", Source: entities.EnvVarSourceManagedService})

	err := repo.DeleteEnvVarsBySource(ctx, "app-1", entities.EnvVarSourceManagedService)
	if err != nil {
		t.Fatalf("DeleteEnvVarsBySource failed: %v", err)
	}

	vars, _ := repo.GetEnvVars(ctx, "app-1")
	if len(vars) != 1 || vars[0].Key != "MANUAL_KEY" {
		t.Errorf("Expected only manual var to remain, got %v", vars)
	}
}

func TestEnvVarIsSecret(t *testing.T) {
	repo := NewMemoryEnvVarRepository()
	ctx := context.Background()

	repo.SetEnvVar(ctx, &entities.AppEnvVar{AppID: "app-1", Key: "SECRET_KEY", Value: "s3cret", IsSecret: true})

	vars, _ := repo.GetEnvVars(ctx, "app-1")
	if !vars[0].IsSecret {
		t.Error("Expected IsSecret to be true")
	}
}
