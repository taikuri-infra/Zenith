package services

import (
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func newTestBackupService() *BackupService {
	k8s := k8sclient.NewMemoryClient()
	backupRepo := memory.NewMemoryBackupRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	return NewBackupService(k8s, backupRepo, dbRepo, nil, nil, "zenith-backups", "zenith-staging")
}

// --- NewBackupService tests ---

func TestNewBackupService(t *testing.T) {
	svc := newTestBackupService()
	if svc == nil {
		t.Fatal("Expected non-nil BackupService")
	}
}

func TestNewBackupService_NilDeps(t *testing.T) {
	svc := NewBackupService(nil, nil, nil, nil, nil, "", "")
	if svc == nil {
		t.Fatal("Expected non-nil BackupService even with nil deps")
	}
}

// --- buildBackupJobSpec tests ---

func TestBuildBackupJobSpec(t *testing.T) {
	svc := newTestBackupService()

	db := &entities.UserDatabase{
		ID:     "db-123",
		Host:   "pg-rw.zenith-staging.svc",
		Port:   5432,
		DBUser: "myuser",
		DBName: "mydb",
	}

	job := svc.buildBackupJobSpec(db, "secretpass", "backups/db-123/bk-456.sql.gz", "backup-job-1")

	if job["apiVersion"] != "batch/v1" {
		t.Error("Expected apiVersion batch/v1")
	}
	if job["kind"] != "Job" {
		t.Error("Expected kind Job")
	}

	metadata := job["metadata"].(map[string]interface{})
	if metadata["name"] != "backup-job-1" {
		t.Errorf("Expected name 'backup-job-1', got '%s'", metadata["name"])
	}
	if metadata["namespace"] != "zenith-staging" {
		t.Errorf("Expected namespace 'zenith-staging', got '%s'", metadata["namespace"])
	}

	spec := job["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})

	if len(containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(containers))
	}
	if containers[0]["name"] != "pg-backup" {
		t.Errorf("Expected container name 'pg-backup', got '%s'", containers[0]["name"])
	}

	// Check env vars contain expected values
	envVars := containers[0]["env"].([]map[string]interface{})
	envMap := make(map[string]interface{})
	for _, e := range envVars {
		envMap[e["name"].(string)] = e
	}

	if envMap["PGPASSWORD"] == nil {
		t.Error("Expected PGPASSWORD env var")
	}
	if envMap["PGHOST"] == nil {
		t.Error("Expected PGHOST env var")
	}
	if envMap["S3_BUCKET"] == nil {
		t.Error("Expected S3_BUCKET env var")
	}
	if envMap["S3_KEY"] == nil {
		t.Error("Expected S3_KEY env var")
	}
}

// --- buildRestoreJobSpec tests ---

func TestBuildRestoreJobSpec(t *testing.T) {
	svc := newTestBackupService()

	db := &entities.UserDatabase{
		ID:     "db-123",
		Host:   "pg-rw.zenith-staging.svc",
		Port:   5432,
		DBUser: "myuser",
		DBName: "mydb",
	}

	job := svc.buildRestoreJobSpec(db, "secretpass", "backups/db-123/bk-456.sql.gz", "restore-job-1")

	if job["apiVersion"] != "batch/v1" {
		t.Error("Expected apiVersion batch/v1")
	}
	if job["kind"] != "Job" {
		t.Error("Expected kind Job")
	}

	metadata := job["metadata"].(map[string]interface{})
	if metadata["name"] != "restore-job-1" {
		t.Errorf("Expected name 'restore-job-1', got '%s'", metadata["name"])
	}

	spec := job["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	podSpec := tmpl["spec"].(map[string]interface{})
	containers := podSpec["containers"].([]map[string]interface{})
	if containers[0]["name"] != "pg-restore" {
		t.Errorf("Expected container name 'pg-restore', got '%s'", containers[0]["name"])
	}

	// Verify command includes restore script
	command := containers[0]["command"].([]string)
	fullCmd := strings.Join(command, " ")
	if !strings.Contains(fullCmd, "psql") {
		t.Error("Expected restore command to contain 'psql'")
	}
}

// --- BackupStorageKey additional tests ---

func TestBackupStorageKey_Format(t *testing.T) {
	key := BackupStorageKey("abc", "def")
	if !strings.HasPrefix(key, "backups/") {
		t.Error("Expected key to start with 'backups/'")
	}
	if !strings.HasSuffix(key, ".sql.gz") {
		t.Error("Expected key to end with '.sql.gz'")
	}
}
