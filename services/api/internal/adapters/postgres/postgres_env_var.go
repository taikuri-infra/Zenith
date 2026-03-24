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

const envVarSelectCols = `id, app_id, COALESCE(environment_id, ''), key, value, is_secret, source, COALESCE(source_id, ''), created_at, updated_at`

func scanEnvVar(scanner interface{ Scan(dest ...any) error }) (*entities.AppEnvVar, error) {
	var ev entities.AppEnvVar
	var source string
	err := scanner.Scan(
		&ev.ID, &ev.AppID, &ev.EnvironmentID, &ev.Key, &ev.Value, &ev.IsSecret,
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

	var envID interface{}
	if envVar.EnvironmentID != "" {
		envID = envVar.EnvironmentID
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO app_env_vars_v2 (id, app_id, environment_id, key, value, is_secret, source, source_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (app_id, key, environment_id) DO UPDATE
		   SET value = $5, is_secret = $6, source = $7, source_id = $8, updated_at = $10`,
		envVar.ID, envVar.AppID, envID, envVar.Key, envVar.Value, envVar.IsSecret,
		string(envVar.Source), envVar.SourceID, now, now,
	)
	if err != nil {
		return fmt.Errorf("set env var: %w", err)
	}
	return nil
}

// GetEnvVars returns env vars for an app scoped to a specific environment.
// Pass environmentID="" to get production/default vars (environment_id IS NULL).
func (r *PostgresEnvVarRepository) GetEnvVars(ctx context.Context, appID string) ([]entities.AppEnvVar, error) {
	return r.GetEnvVarsByEnvironment(ctx, appID, "")
}

// GetEnvVarsByEnvironment returns env vars for a specific environment.
// environmentID="" returns production/default vars (environment_id IS NULL).
func (r *PostgresEnvVarRepository) GetEnvVarsByEnvironment(ctx context.Context, appID, environmentID string) ([]entities.AppEnvVar, error) {
	var query string
	var args []interface{}

	if environmentID == "" {
		query = `SELECT ` + envVarSelectCols + ` FROM app_env_vars_v2
			 WHERE app_id = $1 AND environment_id IS NULL ORDER BY key ASC`
		args = []interface{}{appID}
	} else {
		query = `SELECT ` + envVarSelectCols + ` FROM app_env_vars_v2
			 WHERE app_id = $1 AND environment_id = $2 ORDER BY key ASC`
		args = []interface{}{appID, environmentID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
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

		var envID interface{}
		if vars[i].EnvironmentID != "" {
			envID = vars[i].EnvironmentID
		}

		_, err := tx.Exec(ctx,
			`INSERT INTO app_env_vars_v2 (id, app_id, environment_id, key, value, is_secret, source, source_id, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			 ON CONFLICT (app_id, key, environment_id) DO UPDATE
			   SET value = $5, is_secret = $6, source = $7, source_id = $8, updated_at = $10`,
			vars[i].ID, appID, envID, vars[i].Key, vars[i].Value, vars[i].IsSecret,
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
