package memory

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestCreateDatabase(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	db, err := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEnginePostgres,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if db.ID == "" {
		t.Error("Expected ID to be set")
	}
	if db.Engine != entities.DatabaseEnginePostgres {
		t.Errorf("Expected engine postgres, got %s", db.Engine)
	}
	if db.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", db.Port)
	}
	if db.Status != entities.DatabaseStatusReady {
		t.Errorf("Expected status ready, got %s", db.Status)
	}
}

func TestCreateDatabaseMySQL(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	db, err := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEngineMySQL,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if db.Port != 3306 {
		t.Errorf("Expected port 3306, got %d", db.Port)
	}
}

func TestCreateDatabaseRedis(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	db, err := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEngineRedis,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if db.Port != 6379 {
		t.Errorf("Expected port 6379, got %d", db.Port)
	}
}

func TestCreateDatabaseDefaultEngine(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	db, err := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if db.Engine != entities.DatabaseEnginePostgres {
		t.Errorf("Expected default engine postgres, got %s", db.Engine)
	}
}

func TestCreateDatabaseDuplicate(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	_, err := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEnginePostgres,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_, err = repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEnginePostgres,
	})
	if err == nil {
		t.Error("Expected duplicate error, got nil")
	}
}

func TestCreateDatabaseDifferentEngines(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	_, err := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEnginePostgres,
	})
	if err != nil {
		t.Fatalf("Expected no error for postgres, got %v", err)
	}

	_, err = repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEngineRedis,
	})
	if err != nil {
		t.Fatalf("Expected no error for redis on same app, got %v", err)
	}
}

func TestCreateDatabaseUnsupportedEngine(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	_, err := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: "cockroachdb",
	})
	if err == nil {
		t.Error("Expected unsupported engine error, got nil")
	}
}

func TestGetDatabase(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	created, _ := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEnginePostgres,
	})

	got, err := repo.GetDatabase(ctx, created.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, got.ID)
	}
}

func TestGetDatabaseNotFound(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	_, err := repo.GetDatabase(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected not found error, got nil")
	}
}

func TestListDatabasesByApp(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres})
	repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEngineRedis})
	repo.CreateDatabase(ctx, "app-2", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres})

	dbs, err := repo.ListDatabasesByApp(ctx, "app-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(dbs) != 2 {
		t.Errorf("Expected 2 databases for app-1, got %d", len(dbs))
	}
}

func TestListDatabasesByUser(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres})
	repo.CreateDatabase(ctx, "app-2", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres})
	repo.CreateDatabase(ctx, "app-3", "user-2", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres})

	dbs, err := repo.ListDatabasesByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(dbs) != 2 {
		t.Errorf("Expected 2 databases for user-1, got %d", len(dbs))
	}
}

func TestDeleteDatabase(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	db, _ := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEnginePostgres,
	})

	err := repo.DeleteDatabase(ctx, db.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_, err = repo.GetDatabase(ctx, db.ID)
	if err == nil {
		t.Error("Expected not found after delete, got nil")
	}

	// Password should also be deleted
	_, ok := repo.GetPassword(db.ID)
	if ok {
		t.Error("Expected password to be deleted")
	}
}

func TestDeleteDatabaseNotFound(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	err := repo.DeleteDatabase(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected not found error, got nil")
	}
}

func TestUpdateDatabaseStatus(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	db, _ := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEnginePostgres,
	})

	err := repo.UpdateDatabaseStatus(ctx, db.ID, entities.DatabaseStatusError)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, _ := repo.GetDatabase(ctx, db.ID)
	if got.Status != entities.DatabaseStatusError {
		t.Errorf("Expected status error, got %s", got.Status)
	}
}

func TestCountDatabasesByUser(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres})
	repo.CreateDatabase(ctx, "app-2", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres})
	repo.CreateDatabase(ctx, "app-3", "user-2", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres})

	count, err := repo.CountDatabasesByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestGetPassword(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	db, _ := repo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEnginePostgres,
	})

	pw, ok := repo.GetPassword(db.ID)
	if !ok {
		t.Fatal("Expected password to exist")
	}
	if len(pw) != 24 {
		t.Errorf("Expected 24-char password, got %d", len(pw))
	}
}

func TestConnectionString(t *testing.T) {
	repo := NewMemoryDatabaseRepository()
	ctx := context.Background()

	db, _ := repo.CreateDatabase(ctx, "app-12345678", "user-1", &dto.CreateDatabaseInput{
		Engine: entities.DatabaseEnginePostgres,
	})

	pw, _ := repo.GetPassword(db.ID)
	connStr := db.ConnectionString(pw)
	if connStr == "" {
		t.Error("Expected non-empty connection string")
	}
	if len(connStr) < 20 {
		t.Errorf("Connection string too short: %s", connStr)
	}
}
