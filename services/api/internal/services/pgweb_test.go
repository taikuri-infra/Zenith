package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
)

func newTestPgwebService() (*PgwebService, *memory.MemoryDatabaseRepository) {
	dbRepo := memory.NewMemoryDatabaseRepository()
	k8s := k8sclient.NewMemoryClient()
	return NewPgwebService(dbRepo, k8s, "zenith-staging", "apps.stage.freezenith.com"), dbRepo
}

// --- NewPgwebService tests ---

func TestNewPgwebService(t *testing.T) {
	svc, _ := newTestPgwebService()
	if svc == nil {
		t.Fatal("Expected non-nil PgwebService")
	}
}

// --- GetSession tests ---

func TestGetSession_NoSession(t *testing.T) {
	svc, _ := newTestPgwebService()
	ctx := context.Background()

	_, err := svc.GetSession(ctx, "db-123")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

// --- StartSession tests ---

func TestStartSession_DatabaseNotFound(t *testing.T) {
	svc, _ := newTestPgwebService()
	ctx := context.Background()

	_, err := svc.StartSession(ctx, "nonexistent-db", "user-1", true)
	if err == nil {
		t.Error("Expected error for non-existent database")
	}
}

func TestStartSession_NonPostgres(t *testing.T) {
	svc, dbRepo := newTestPgwebService()
	ctx := context.Background()

	// Create a Redis database
	db, _ := dbRepo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Name:   "my-redis",
		Engine: "redis",
	})

	_, err := svc.StartSession(ctx, db.ID, "user-1", false)
	if err == nil {
		t.Error("Expected error for non-PostgreSQL database")
	}
}

func TestStartSession_Success(t *testing.T) {
	svc, dbRepo := newTestPgwebService()
	ctx := context.Background()

	// Create a PostgreSQL database
	db, _ := dbRepo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Name:   "my-postgres",
		Engine: "postgresql",
	})

	session, err := svc.StartSession(ctx, db.ID, "user-1", true)
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}
	if session == nil {
		t.Fatal("Expected non-nil session")
	}
	if session.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", session.Status)
	}
	if session.URL == "" {
		t.Error("Expected non-empty URL")
	}
	if session.Token == "" {
		t.Error("Expected non-empty token")
	}
	if !session.ReadOnly {
		t.Error("Expected readonly=true")
	}
}

func TestStartSession_ExistingSession(t *testing.T) {
	svc, dbRepo := newTestPgwebService()
	ctx := context.Background()

	db, _ := dbRepo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Name:   "my-postgres",
		Engine: "postgresql",
	})

	session1, _ := svc.StartSession(ctx, db.ID, "user-1", true)
	session2, err := svc.StartSession(ctx, db.ID, "user-1", true)
	if err != nil {
		t.Fatalf("StartSession (existing) failed: %v", err)
	}
	// Should return the same session
	if session2.Token != session1.Token {
		t.Error("Expected same session to be returned for existing database")
	}
}

// --- GetSession with active session ---

func TestGetSession_Active(t *testing.T) {
	svc, dbRepo := newTestPgwebService()
	ctx := context.Background()

	db, _ := dbRepo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Name:   "my-postgres",
		Engine: "postgresql",
	})

	svc.StartSession(ctx, db.ID, "user-1", false)

	session, err := svc.GetSession(ctx, db.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session.DatabaseID != db.ID {
		t.Errorf("Expected database_id '%s', got '%s'", db.ID, session.DatabaseID)
	}
}

// --- StopSession tests ---

func TestStopSession_NoSession(t *testing.T) {
	svc, _ := newTestPgwebService()
	ctx := context.Background()

	err := svc.StopSession(ctx, "db-123")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestStopSession_Success(t *testing.T) {
	svc, dbRepo := newTestPgwebService()
	ctx := context.Background()

	db, _ := dbRepo.CreateDatabase(ctx, "app-1", "user-1", &dto.CreateDatabaseInput{
		Name:   "my-postgres",
		Engine: "postgresql",
	})

	svc.StartSession(ctx, db.ID, "user-1", true)
	err := svc.StopSession(ctx, db.ID)
	if err != nil {
		t.Fatalf("StopSession failed: %v", err)
	}

	// Session should be gone
	_, err = svc.GetSession(ctx, db.ID)
	if err == nil {
		t.Error("Expected error after stopping session")
	}
}
