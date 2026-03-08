package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresBackupRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBackupRepository(pool *pgxpool.Pool) *PostgresBackupRepository {
	return &PostgresBackupRepository{pool: pool}
}

const backupSelectCols = `id, database_id, user_id, type, status, size_mb, storage_key, error, created_at, updated_at`

func scanBackup(scanner interface{ Scan(dest ...any) error }) (*entities.DatabaseBackup, error) {
	var b entities.DatabaseBackup
	var bType, bStatus string
	err := scanner.Scan(
		&b.ID, &b.DatabaseID, &b.UserID, &bType, &bStatus,
		&b.SizeMB, &b.StorageKey, &b.Error, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	b.Type = entities.BackupType(bType)
	b.Status = entities.BackupStatus(bStatus)
	return &b, nil
}

func (r *PostgresBackupRepository) CreateBackup(ctx context.Context, databaseID, userID string, backupType entities.BackupType) (*entities.DatabaseBackup, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO database_backups (id, database_id, user_id, type, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, databaseID, userID, string(backupType), string(entities.BackupStatusPending), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create backup: %w", err)
	}
	return &entities.DatabaseBackup{
		ID:         id,
		DatabaseID: databaseID,
		UserID:     userID,
		Type:       backupType,
		Status:     entities.BackupStatusPending,
		Timestamps: entities.Timestamps{CreatedAt: now, UpdatedAt: now},
	}, nil
}

func (r *PostgresBackupRepository) GetBackup(ctx context.Context, id string) (*entities.DatabaseBackup, error) {
	b, err := scanBackup(r.pool.QueryRow(ctx,
		`SELECT `+backupSelectCols+` FROM database_backups WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("backup not found: %s", id)
	}
	return b, nil
}

func (r *PostgresBackupRepository) ListBackupsByDatabase(ctx context.Context, databaseID string) ([]entities.DatabaseBackup, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+backupSelectCols+` FROM database_backups WHERE database_id = $1 ORDER BY created_at DESC`, databaseID,
	)
	if err != nil {
		return nil, fmt.Errorf("list backups by database: %w", err)
	}
	defer rows.Close()
	var backups []entities.DatabaseBackup
	for rows.Next() {
		b, err := scanBackup(rows)
		if err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}
		backups = append(backups, *b)
	}
	return backups, nil
}

func (r *PostgresBackupRepository) ListBackupsByUser(ctx context.Context, userID string) ([]entities.DatabaseBackup, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+backupSelectCols+` FROM database_backups WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list backups by user: %w", err)
	}
	defer rows.Close()
	var backups []entities.DatabaseBackup
	for rows.Next() {
		b, err := scanBackup(rows)
		if err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}
		backups = append(backups, *b)
	}
	return backups, nil
}

func (r *PostgresBackupRepository) UpdateBackupStatus(ctx context.Context, id string, status entities.BackupStatus, sizeMB int, errMsg string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE database_backups SET status = $1, size_mb = $2, error = $3, updated_at = $4 WHERE id = $5`,
		string(status), sizeMB, errMsg, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("update backup status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("backup not found: %s", id)
	}
	return nil
}

func (r *PostgresBackupRepository) DeleteBackup(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM database_backups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete backup: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("backup not found: %s", id)
	}
	return nil
}

func (r *PostgresBackupRepository) CountBackupsByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM database_backups WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count backups: %w", err)
	}
	return count, nil
}
