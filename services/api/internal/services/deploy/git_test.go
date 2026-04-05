package deploy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCloneRepo_EmptyURL(t *testing.T) {
	err := CloneRepo(context.Background(), "", "main", "/tmp/test-clone")
	if err == nil {
		t.Error("Expected error for empty repo URL")
	}
}

func TestCleanupClone(t *testing.T) {
	// Create a temp directory to clean up
	dir, err := os.MkdirTemp("", "zenith-cleanup-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a file inside it
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Clean up
	if err := CleanupClone(dir); err != nil {
		t.Fatalf("CleanupClone failed: %v", err)
	}

	// Verify directory is gone
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("Expected directory to be removed")
	}
}

func TestCleanupClone_NonexistentDir(t *testing.T) {
	// Should not error on nonexistent directory
	err := CleanupClone("/tmp/zenith-nonexistent-dir-12345")
	if err != nil {
		t.Errorf("CleanupClone should not error on nonexistent dir: %v", err)
	}
}

func TestGetLatestCommitSHA_InvalidDir(t *testing.T) {
	_, err := GetLatestCommitSHA(context.Background(), "/tmp/zenith-not-a-repo")
	if err == nil {
		t.Error("Expected error for non-git directory")
	}
}
