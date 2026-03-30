package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDeployHookRepository is a PostgreSQL-backed DeployHookRepository.
type PostgresDeployHookRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresDeployHookRepository creates a new PostgresDeployHookRepository.
func NewPostgresDeployHookRepository(pool *pgxpool.Pool) *PostgresDeployHookRepository {
	return &PostgresDeployHookRepository{pool: pool}
}

func (r *PostgresDeployHookRepository) CreateHook(ctx context.Context, hook *entities.DeployHook) (*entities.DeployHook, error) {
	hook.ID = uuid.New().String()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO deploy_hooks (id, app_id, name, type, url, command, "order", active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())`,
		hook.ID, hook.AppID, hook.Name, string(hook.Type), hook.URL, hook.Command, hook.Order, hook.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create deploy hook: %w", err)
	}
	return hook, nil
}

func (r *PostgresDeployHookRepository) GetHook(ctx context.Context, id string) (*entities.DeployHook, error) {
	var h entities.DeployHook
	var hookType string
	err := r.pool.QueryRow(ctx,
		`SELECT id, app_id, name, type, url, command, "order", active, created_at, updated_at
		 FROM deploy_hooks WHERE id = $1`, id,
	).Scan(&h.ID, &h.AppID, &h.Name, &hookType, &h.URL, &h.Command, &h.Order, &h.Active, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("deploy hook not found")
	}
	h.Type = entities.DeployHookType(hookType)
	return &h, nil
}

func (r *PostgresDeployHookRepository) ListHooksByApp(ctx context.Context, appID string) ([]entities.DeployHook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, app_id, name, type, url, command, "order", active, created_at, updated_at
		 FROM deploy_hooks WHERE app_id = $1 ORDER BY "order", created_at`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list deploy hooks: %w", err)
	}
	defer rows.Close()

	var hooks []entities.DeployHook
	for rows.Next() {
		var h entities.DeployHook
		var hookType string
		if err := rows.Scan(&h.ID, &h.AppID, &h.Name, &hookType, &h.URL, &h.Command, &h.Order, &h.Active, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan deploy hook: %w", err)
		}
		h.Type = entities.DeployHookType(hookType)
		hooks = append(hooks, h)
	}
	return hooks, nil
}

func (r *PostgresDeployHookRepository) UpdateHook(ctx context.Context, id string, name *string, url *string, command *string, order *int, active *bool) (*entities.DeployHook, error) {
	sets := []string{"updated_at = now()"}
	args := []interface{}{}
	argIdx := 1

	if name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *name)
		argIdx++
	}
	if url != nil {
		sets = append(sets, fmt.Sprintf("url = $%d", argIdx))
		args = append(args, *url)
		argIdx++
	}
	if command != nil {
		sets = append(sets, fmt.Sprintf("command = $%d", argIdx))
		args = append(args, *command)
		argIdx++
	}
	if order != nil {
		sets = append(sets, fmt.Sprintf(`"order" = $%d`, argIdx))
		args = append(args, *order)
		argIdx++
	}
	if active != nil {
		sets = append(sets, fmt.Sprintf("active = $%d", argIdx))
		args = append(args, *active)
		argIdx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE deploy_hooks SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update deploy hook: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("deploy hook not found")
	}
	return r.GetHook(ctx, id)
}

func (r *PostgresDeployHookRepository) DeleteHook(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM deploy_hooks WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete deploy hook: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("deploy hook not found")
	}
	return nil
}

func (r *PostgresDeployHookRepository) CountHooksByApp(ctx context.Context, appID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM deploy_hooks WHERE app_id = $1", appID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count hooks: %w", err)
	}
	return count, nil
}
