package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresEnvVarRepository is a PostgreSQL-backed EnvVarRepository.
type PostgresEnvVarRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresEnvVarRepository creates a new PostgreSQL EnvVarRepository.
func NewPostgresEnvVarRepository(pool *pgxpool.Pool) *PostgresEnvVarRepository {
	return &PostgresEnvVarRepository{pool: pool}
}

const envVarSelectCols = `id, app_id, key, value, is_secret, source, COALESCE(source_id, ''), created_at, updated_at`

func scanEnvVar(scanner interface{ Scan(dest ...any) error }) (*entities.AppEnvVar, error) {
	var ev entities.AppEnvVar
	var source string
	err := scanner.Scan(
		&ev.ID, &ev.AppID, &ev.Key, &ev.Value, &ev.IsSecret,
		&source, &ev.SourceID, &ev.CreatedAt, &ev.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	ev.Source = entities.EnvVarSource(source)
	return &ev, nil
}

func (r *PostgresEnvVarRepository) SetEnvVar(ctx context.Context, envVar *entities.AppEnvVar) error {
	if envVar.ID == "" {
		envVar.ID = uuid.New().String()
	}
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO app_env_vars_v2 (id, app_id, key, value, is_secret, source, source_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (app_id, key) DO UPDATE SET value = $4, is_secret = $5, source = $6, source_id = $7, updated_at = $9`,
		envVar.ID, envVar.AppID, envVar.Key, envVar.Value, envVar.IsSecret,
		string(envVar.Source), envVar.SourceID, now, now,
	)
	if err != nil {
		return fmt.Errorf("set env var: %w", err)
	}
	return nil
}

func (r *PostgresEnvVarRepository) GetEnvVars(ctx context.Context, appID string) ([]entities.AppEnvVar, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+envVarSelectCols+` FROM app_env_vars_v2 WHERE app_id = $1 ORDER BY key ASC`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("get env vars: %w", err)
	}
	defer rows.Close()

	var vars []entities.AppEnvVar
	for rows.Next() {
		ev, err := scanEnvVar(rows)
		if err != nil {
			return nil, fmt.Errorf("scan env var: %w", err)
		}
		vars = append(vars, *ev)
	}
	return vars, nil
}

func (r *PostgresEnvVarRepository) DeleteEnvVar(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM app_env_vars_v2 WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete env var: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("env var not found: %s", id)
	}
	return nil
}

func (r *PostgresEnvVarRepository) BulkSetEnvVars(ctx context.Context, appID string, vars []entities.AppEnvVar) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	for i := range vars {
		if vars[i].ID == "" {
			vars[i].ID = uuid.New().String()
		}
		vars[i].AppID = appID

		_, err := tx.Exec(ctx,
			`INSERT INTO app_env_vars_v2 (id, app_id, key, value, is_secret, source, source_id, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (app_id, key) DO UPDATE SET value = $4, is_secret = $5, source = $6, source_id = $7, updated_at = $9`,
			vars[i].ID, appID, vars[i].Key, vars[i].Value, vars[i].IsSecret,
			string(vars[i].Source), vars[i].SourceID, now, now,
		)
		if err != nil {
			return fmt.Errorf("bulk set env var '%s': %w", vars[i].Key, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresEnvVarRepository) DeleteEnvVarsBySource(ctx context.Context, appID string, source entities.EnvVarSource) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM app_env_vars_v2 WHERE app_id = $1 AND source = $2`,
		appID, string(source),
	)
	if err != nil {
		return fmt.Errorf("delete env vars by source: %w", err)
	}
	return nil
}
