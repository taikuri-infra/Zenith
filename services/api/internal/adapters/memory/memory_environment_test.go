package memory

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestEnvironmentCreate(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	env := &entities.Environment{
		ID:        "env-1",
		ProjectID: "proj-1",
		Name:      entities.EnvironmentProduction,
		Slug:      "production",
		Status:    entities.EnvironmentStatusActive,
		IsDefault: true,
	}
	err := repo.CreateEnvironment(ctx, env)
	if err != nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}
	if env.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestEnvironmentCreateDuplicate(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	env := &entities.Environment{ID: "env-1", ProjectID: "proj-1", Name: entities.EnvironmentProduction}
	repo.CreateEnvironment(ctx, env)

	env2 := &entities.Environment{ID: "env-2", ProjectID: "proj-1", Name: entities.EnvironmentProduction}
	err := repo.CreateEnvironment(ctx, env2)
	if err == nil {
		t.Error("Expected error on duplicate name in same project")
	}
}

func TestEnvironmentCreateSameNameDifferentProject(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	repo.CreateEnvironment(ctx, &entities.Environment{ID: "env-1", ProjectID: "proj-1", Name: entities.EnvironmentProduction})

	err := repo.CreateEnvironment(ctx, &entities.Environment{ID: "env-2", ProjectID: "proj-2", Name: entities.EnvironmentProduction})
	if err != nil {
		t.Errorf("Should allow same name in different project, got: %v", err)
	}
}

func TestEnvironmentGet(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	repo.CreateEnvironment(ctx, &entities.Environment{
		ID: "env-1", ProjectID: "proj-1", Name: entities.EnvironmentProduction, IsDefault: true,
	})

	env, err := repo.GetEnvironment(ctx, "env-1")
	if err != nil {
		t.Fatalf("GetEnvironment failed: %v", err)
	}
	if env.Name != entities.EnvironmentProduction {
		t.Errorf("Expected name 'production', got '%s'", env.Name)
	}
	if !env.IsDefault {
		t.Error("Expected IsDefault to be true")
	}
}

func TestEnvironmentGetNotFound(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	_, err := repo.GetEnvironment(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent environment")
	}
}

func TestEnvironmentGetByName(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	repo.CreateEnvironment(ctx, &entities.Environment{
		ID: "env-1", ProjectID: "proj-1", Name: entities.EnvironmentProduction,
	})
	repo.CreateEnvironment(ctx, &entities.Environment{
		ID: "env-2", ProjectID: "proj-1", Name: entities.EnvironmentStaging,
	})

	env, err := repo.GetEnvironmentByName(ctx, "proj-1", entities.EnvironmentStaging)
	if err != nil {
		t.Fatalf("GetEnvironmentByName failed: %v", err)
	}
	if env.ID != "env-2" {
		t.Errorf("Expected env-2, got %s", env.ID)
	}
}

func TestEnvironmentGetByNameNotFound(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	_, err := repo.GetEnvironmentByName(ctx, "proj-1", entities.EnvironmentStaging)
	if err == nil {
		t.Error("Expected error for nonexistent environment name")
	}
}

func TestEnvironmentListByProject(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	repo.CreateEnvironment(ctx, &entities.Environment{ID: "env-1", ProjectID: "proj-1", Name: entities.EnvironmentProduction, IsDefault: true})
	repo.CreateEnvironment(ctx, &entities.Environment{ID: "env-2", ProjectID: "proj-1", Name: entities.EnvironmentStaging})
	repo.CreateEnvironment(ctx, &entities.Environment{ID: "env-3", ProjectID: "proj-2", Name: entities.EnvironmentProduction})

	envs, err := repo.ListEnvironmentsByProject(ctx, "proj-1")
	if err != nil {
		t.Fatalf("ListEnvironmentsByProject failed: %v", err)
	}
	if len(envs) != 2 {
		t.Fatalf("Expected 2 envs for proj-1, got %d", len(envs))
	}
	// Default environment should be first
	if !envs[0].IsDefault {
		t.Error("Expected default environment to be first")
	}
}

func TestEnvironmentListByProjectEmpty(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	envs, err := repo.ListEnvironmentsByProject(ctx, "proj-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(envs) != 0 {
		t.Errorf("Expected 0 envs, got %d", len(envs))
	}
}

func TestEnvironmentUpdateStatus(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	repo.CreateEnvironment(ctx, &entities.Environment{
		ID: "env-1", ProjectID: "proj-1", Name: entities.EnvironmentProduction,
		Status: entities.EnvironmentStatusProvisioning,
	})

	err := repo.UpdateEnvironmentStatus(ctx, "env-1", entities.EnvironmentStatusActive)
	if err != nil {
		t.Fatalf("UpdateEnvironmentStatus failed: %v", err)
	}

	env, _ := repo.GetEnvironment(ctx, "env-1")
	if env.Status != entities.EnvironmentStatusActive {
		t.Errorf("Expected status 'active', got '%s'", env.Status)
	}
}

func TestEnvironmentUpdateStatusNotFound(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	err := repo.UpdateEnvironmentStatus(ctx, "nonexistent", entities.EnvironmentStatusActive)
	if err == nil {
		t.Error("Expected error for nonexistent environment")
	}
}

func TestEnvironmentDelete(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	repo.CreateEnvironment(ctx, &entities.Environment{ID: "env-1", ProjectID: "proj-1", Name: entities.EnvironmentProduction})

	err := repo.DeleteEnvironment(ctx, "env-1")
	if err != nil {
		t.Fatalf("DeleteEnvironment failed: %v", err)
	}

	_, err = repo.GetEnvironment(ctx, "env-1")
	if err == nil {
		t.Error("Expected error after delete")
	}
}

func TestEnvironmentDeleteNotFound(t *testing.T) {
	repo := NewMemoryEnvironmentRepository()
	ctx := context.Background()

	err := repo.DeleteEnvironment(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent environment")
	}
}

func TestEnvironmentIsStaging(t *testing.T) {
	prod := &entities.Environment{Name: entities.EnvironmentProduction}
	stage := &entities.Environment{Name: entities.EnvironmentStaging}

	if prod.IsStaging() {
		t.Error("Production should not be staging")
	}
	if !stage.IsStaging() {
		t.Error("Staging should be staging")
	}
}
