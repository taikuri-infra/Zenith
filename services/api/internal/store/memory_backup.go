package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemoryBackupRepository is an in-memory implementation of BackupRepository.
type MemoryBackupRepository struct {
	mu      sync.RWMutex
	backups map[string]*entities.DatabaseBackup // id -> backup
}

// NewMemoryBackupRepository creates a new MemoryBackupRepository.
func NewMemoryBackupRepository() *MemoryBackupRepository {
	return &MemoryBackupRepository{
		backups: make(map[string]*entities.DatabaseBackup),
	}
}

func (r *MemoryBackupRepository) CreateBackup(_ context.Context, databaseID, userID string, backupType entities.BackupType) (*entities.DatabaseBackup, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if backupType == "" {
		backupType = entities.BackupTypeManual
	}

	now := time.Now()
	backup := &entities.DatabaseBackup{
		ID:         uuid.New().String(),
		DatabaseID: databaseID,
		UserID:     userID,
		Type:       backupType,
		Status:     entities.BackupStatusPending,
		StorageKey: fmt.Sprintf("backups/%s/%s.sql.gz", databaseID, uuid.New().String()),
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	r.backups[backup.ID] = backup

	// Simulate async backup completion
	go func() {
		time.Sleep(2 * time.Second)
		r.mu.Lock()
		defer r.mu.Unlock()
		if b, ok := r.backups[backup.ID]; ok && b.Status == entities.BackupStatusPending {
			b.Status = entities.BackupStatusCompleted
			b.SizeMB = 12 // simulated size
			b.UpdatedAt = time.Now()
		}
	}()

	return backup, nil
}

func (r *MemoryBackupRepository) GetBackup(_ context.Context, id string) (*entities.DatabaseBackup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backup, ok := r.backups[id]
	if !ok {
		return nil, fmt.Errorf("backup not found: %s", id)
	}
	return backup, nil
}

func (r *MemoryBackupRepository) ListBackupsByDatabase(_ context.Context, databaseID string) ([]entities.DatabaseBackup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.DatabaseBackup
	for _, b := range r.backups {
		if b.DatabaseID == databaseID {
			result = append(result, *b)
		}
	}
	return result, nil
}

func (r *MemoryBackupRepository) ListBackupsByUser(_ context.Context, userID string) ([]entities.DatabaseBackup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.DatabaseBackup
	for _, b := range r.backups {
		if b.UserID == userID {
			result = append(result, *b)
		}
	}
	return result, nil
}

func (r *MemoryBackupRepository) UpdateBackupStatus(_ context.Context, id string, status entities.BackupStatus, sizeMB int, errMsg string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	backup, ok := r.backups[id]
	if !ok {
		return fmt.Errorf("backup not found: %s", id)
	}

	backup.Status = status
	backup.SizeMB = sizeMB
	backup.Error = errMsg
	backup.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryBackupRepository) DeleteBackup(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.backups[id]; !ok {
		return fmt.Errorf("backup not found: %s", id)
	}
	delete(r.backups, id)
	return nil
}

func (r *MemoryBackupRepository) CountBackupsByUser(_ context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, b := range r.backups {
		if b.UserID == userID {
			count++
		}
	}
	return count, nil
}
