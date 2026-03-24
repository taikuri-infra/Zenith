package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresEnvironmentRepository is a PostgreSQL-backed EnvironmentRepository.
type PostgresEnvironmentRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresEnvironmentRepository creates a new PostgreSQL EnvironmentRepository.
func NewPostgresEnvironmentRepository(pool *pgxpool.Pool) *PostgresEnvironmentRepository {
	return &PostgresEnvironmentRepository{pool: pool}
}

const environmentSelectCols = `id, project_id, name, slug, status, is_default, created_at, updated_at`

func scanEnvironment(scanner interface{ Scan(dest ...any) error }) (*entities.Environment, error) {
	var env entities.Environment
	var name, status string
	err := scanner.Scan(
		&env.ID, &env.ProjectID, &name, &env.Slug, &status, &env.IsDefault,
		&env.CreatedAt, &env.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	env.Name = entities.EnvironmentName(name)
	env.Status = entities.EnvironmentStatus(status)
	return &env, nil
}

func (r *PostgresEnvironmentRepository) CreateEnvironment(ctx context.Context, env *entities.Environment) error {
	now := time.Now()
	env.CreatedAt = now
	env.UpdatedAt = now

	_, err := r.pool.Exec(ctx,
		`INSERT INTO environments (id, project_id, name, slug, status, is_default, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		env.ID, env.ProjectID, string(env.Name), env.Slug, string(env.Status), env.IsDefault,
		now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_environments_project_name") {
			return fmt.Errorf("environment '%s' already exists in this project", env.Name)
		}
		return fmt.Errorf("create environment: %w", err)
	}
	return nil
}

func (r *PostgresEnvironmentRepository) GetEnvironment(ctx context.Context, id string) (*entities.Environment, error) {
	env, err := scanEnvironment(r.pool.QueryRow(ctx,
		`SELECT `+environmentSelectCols+` FROM environments WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", id)
	}
	return env, nil
}

func (r *PostgresEnvironmentRepository) GetEnvironmentByName(ctx context.Context, projectID string, name entities.EnvironmentName) (*entities.Environment, error) {
	env, err := scanEnvironment(r.pool.QueryRow(ctx,
		`SELECT `+environmentSelectCols+` FROM environments WHERE project_id = $1 AND name = $2`,
		projectID, string(name),
	))
	if err != nil {
		return nil, fmt.Errorf("environment '%s' not found in project %s", name, projectID)
	}
	return env, nil
}

func (r *PostgresEnvironmentRepository) ListEnvironmentsByProject(ctx context.Context, projectID string) ([]entities.Environment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+environmentSelectCols+` FROM environments WHERE project_id = $1 ORDER BY is_default DESC, created_at ASC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	defer rows.Close()

	var envs []entities.Environment
	for rows.Next() {
		env, err := scanEnvironment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan environment: %w", err)
		}
		envs = append(envs, *env)
	}
	return envs, nil
}

func (r *PostgresEnvironmentRepository) UpdateEnvironmentStatus(ctx context.Context, id string, status entities.EnvironmentStatus) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE environments SET status = $1, updated_at = now() WHERE id = $2`,
		string(status), id,
	)
	if err != nil {
		return fmt.Errorf("update environment status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("environment not found: %s", id)
	}
	return nil
}

func (r *PostgresEnvironmentRepository) DeleteEnvironment(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM environments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("environment not found: %s", id)
	}
	return nil
}
