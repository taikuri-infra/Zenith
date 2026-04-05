package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupBackupV2Test() (*fiber.App, *handlers.BackupHandlerV2, *memory.MemoryBackupRepository, *memory.MemoryDatabaseRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	backupRepo := memory.NewMemoryBackupRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	// Set user to Pro plan so backups are allowed
	planRepo.SetUserPlan(nil, "user-1", "pro")
	handler := handlers.NewBackupHandlerV2(backupRepo, dbRepo, planRepo)
	return app, handler, backupRepo, dbRepo
}

func createTestDatabase(t *testing.T, dbRepo *memory.MemoryDatabaseRepository, appID string) string {
	t.Helper()
	db, err := dbRepo.CreateDatabase(nil, appID, "user-1", &dto.CreateDatabaseInput{Engine: "postgresql"})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db.ID
}

func TestBackupV2Create(t *testing.T) {
	app, handler, _, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1")

	app.Post("/api/v1/apps/:appId/databases/:dbId/backups", injectUserID("user-1"), handler.Create)

	body := `{"type":"manual"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/app-1/databases/"+dbID+"/backups", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result dto.BackupInfo
	json.NewDecoder(resp.Body).Decode(&result)

	if result.DatabaseID != dbID {
		t.Errorf("Expected database_id '%s', got '%s'", dbID, result.DatabaseID)
	}
	if result.ID == "" {
		t.Error("Expected non-empty backup ID")
	}
}

func TestBackupV2CreateDatabaseNotFound(t *testing.T) {
	app, handler, _, _ := setupBackupV2Test()
	app.Post("/api/v1/apps/:appId/databases/:dbId/backups", injectUserID("user-1"), handler.Create)

	req := httptest.NewRequest("POST", "/api/v1/apps/app-1/databases/nonexistent/backups", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestBackupV2CreateForbidden(t *testing.T) {
	app, handler, _, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1") // owned by user-1

	app.Post("/api/v1/apps/:appId/databases/:dbId/backups", injectUserID("user-2"), handler.Create)

	req := httptest.NewRequest("POST", "/api/v1/apps/app-1/databases/"+dbID+"/backups", nil)
	resp, _ := app.Test(req)
	// user-2 on free plan gets 403 for plan, or 403 for ownership
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestBackupV2List(t *testing.T) {
	app, handler, backupRepo, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1")

	// Create 2 backups directly
	backupRepo.CreateBackup(nil, dbID, "user-1", "manual")
	backupRepo.CreateBackup(nil, dbID, "user-1", "manual")

	app.Get("/api/v1/apps/:appId/databases/:dbId/backups", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/app-1/databases/"+dbID+"/backups", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.BackupInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("Expected 2 backups, got %d", len(result))
	}
}

func TestBackupV2ListDatabaseNotFound(t *testing.T) {
	app, handler, _, _ := setupBackupV2Test()
	app.Get("/api/v1/apps/:appId/databases/:dbId/backups", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/app-1/databases/nonexistent/backups", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestBackupV2Get(t *testing.T) {
	app, handler, backupRepo, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1")

	backup, _ := backupRepo.CreateBackup(nil, dbID, "user-1", "manual")

	app.Get("/api/v1/apps/:appId/databases/:dbId/backups/:backupId", injectUserID("user-1"), handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/apps/app-1/databases/"+dbID+"/backups/"+backup.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result dto.BackupInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if result.ID != backup.ID {
		t.Errorf("Expected backup ID '%s', got '%s'", backup.ID, result.ID)
	}
}

func TestBackupV2GetNotFound(t *testing.T) {
	app, handler, _, _ := setupBackupV2Test()
	app.Get("/api/v1/apps/:appId/databases/:dbId/backups/:backupId", injectUserID("user-1"), handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/apps/app-1/databases/db-1/backups/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestBackupV2Delete(t *testing.T) {
	app, handler, backupRepo, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1")

	backup, _ := backupRepo.CreateBackup(nil, dbID, "user-1", "manual")

	app.Delete("/api/v1/apps/:appId/databases/:dbId/backups/:backupId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/app-1/databases/"+dbID+"/backups/"+backup.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestBackupV2DeleteNotFound(t *testing.T) {
	app, handler, _, _ := setupBackupV2Test()
	app.Delete("/api/v1/apps/:appId/databases/:dbId/backups/:backupId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/app-1/databases/db-1/backups/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestBackupV2DeleteForbidden(t *testing.T) {
	app, handler, backupRepo, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1")

	backup, _ := backupRepo.CreateBackup(nil, dbID, "user-1", "manual")

	app.Delete("/api/v1/apps/:appId/databases/:dbId/backups/:backupId", injectUserID("user-2"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/app-1/databases/"+dbID+"/backups/"+backup.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestBackupV2Restore(t *testing.T) {
	app, handler, backupRepo, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1")

	backup, _ := backupRepo.CreateBackup(nil, dbID, "user-1", "manual")
	// Mark backup as completed so restore is possible
	backupRepo.UpdateBackupStatus(nil, backup.ID, "completed", 12, "")

	app.Post("/api/v1/apps/:appId/databases/:dbId/backups/:backupId/restore", injectUserID("user-1"), handler.Restore)

	req := httptest.NewRequest("POST", "/api/v1/apps/app-1/databases/"+dbID+"/backups/"+backup.ID+"/restore", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "restore initiated" {
		t.Errorf("Expected 'restore initiated', got '%v'", result["message"])
	}
}

func TestBackupV2RestoreNotCompleted(t *testing.T) {
	app, handler, backupRepo, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1")

	backup, _ := backupRepo.CreateBackup(nil, dbID, "user-1", "manual")
	// Backup is still "pending" — restore should fail

	app.Post("/api/v1/apps/:appId/databases/:dbId/backups/:backupId/restore", injectUserID("user-1"), handler.Restore)

	req := httptest.NewRequest("POST", "/api/v1/apps/app-1/databases/"+dbID+"/backups/"+backup.ID+"/restore", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestBackupV2ListByUser(t *testing.T) {
	app, handler, backupRepo, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1")

	backupRepo.CreateBackup(nil, dbID, "user-1", "manual")

	app.Get("/api/v1/backups", injectUserID("user-1"), handler.ListByUser)

	req := httptest.NewRequest("GET", "/api/v1/backups", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.BackupInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 1 {
		t.Errorf("Expected 1 backup, got %d", len(result))
	}
}

func TestBackupV2Download(t *testing.T) {
	app, handler, backupRepo, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1")

	backup, _ := backupRepo.CreateBackup(nil, dbID, "user-1", "manual")
	backupRepo.UpdateBackupStatus(nil, backup.ID, "completed", 12, "")

	app.Get("/api/v1/apps/:appId/databases/:dbId/backups/:backupId/download", injectUserID("user-1"), handler.Download)

	req := httptest.NewRequest("GET", "/api/v1/apps/app-1/databases/"+dbID+"/backups/"+backup.ID+"/download", nil)
	resp, _ := app.Test(req)
	// No backup service configured in test -> 503
	if resp.StatusCode != 503 {
		t.Errorf("Expected 503 (dev mode), got %d", resp.StatusCode)
	}
}

func TestBackupV2DownloadNotCompleted(t *testing.T) {
	app, handler, backupRepo, dbRepo := setupBackupV2Test()
	dbID := createTestDatabase(t, dbRepo, "app-1")

	backup, _ := backupRepo.CreateBackup(nil, dbID, "user-1", "manual")

	app.Get("/api/v1/apps/:appId/databases/:dbId/backups/:backupId/download", injectUserID("user-1"), handler.Download)

	req := httptest.NewRequest("GET", "/api/v1/apps/app-1/databases/"+dbID+"/backups/"+backup.ID+"/download", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
