package services

import (
	"testing"
)

func TestBackupStorageKey(t *testing.T) {
	key := BackupStorageKey("db-123", "bk-456")
	expected := "backups/db-123/bk-456.sql.gz"
	if key != expected {
		t.Errorf("BackupStorageKey = '%s', want '%s'", key, expected)
	}
}

func TestBackupStorageKey_SpecialChars(t *testing.T) {
	key := BackupStorageKey("my-database", "backup-2026-01-01")
	expected := "backups/my-database/backup-2026-01-01.sql.gz"
	if key != expected {
		t.Errorf("BackupStorageKey = '%s', want '%s'", key, expected)
	}
}
