package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPreviewRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPreviewRepository(pool *pgxpool.Pool) *PostgresPreviewRepository {
	return &PostgresPreviewRepository{pool: pool}
}

func (r *PostgresPreviewRepository) CreatePreview(ctx context.Context, appID string, prNumber int, branch, gitSHA, url string) (*entities.PreviewDeployment, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO preview_deployments (id, app_id, pr_number, branch, url, status, git_sha, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, appID, prNumber, branch, url, "building", gitSHA, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create preview: %w", err)
	}
	return &entities.PreviewDeployment{
		ID:        id,
		AppID:     appID,
		PRNumber:  prNumber,
		Branch:    branch,
		URL:       url,
		Status:    "building",
		GitSHA:    gitSHA,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *PostgresPreviewRepository) GetPreview(ctx context.Context, id string) (*entities.PreviewDeployment, error) {
	var p entities.PreviewDeployment
	err := r.pool.QueryRow(ctx,
		`SELECT id, app_id, pr_number, branch, url, status, git_sha, created_at, updated_at
		 FROM preview_deployments WHERE id = $1`, id,
	).Scan(&p.ID, &p.AppID, &p.PRNumber, &p.Branch, &p.URL, &p.Status, &p.GitSHA, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("preview not found: %s", id)
	}
	return &p, nil
}

func (r *PostgresPreviewRepository) ListPreviewsByApp(ctx context.Context, appID string) ([]entities.PreviewDeployment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, app_id, pr_number, branch, url, status, git_sha, created_at, updated_at
		 FROM preview_deployments WHERE app_id = $1 ORDER BY created_at DESC`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("list previews: %w", err)
	}
	defer rows.Close()

	var previews []entities.PreviewDeployment
	for rows.Next() {
		var p entities.PreviewDeployment
		if err := rows.Scan(&p.ID, &p.AppID, &p.PRNumber, &p.Branch, &p.URL, &p.Status,
			&p.GitSHA, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan preview: %w", err)
		}
		previews = append(previews, p)
	}
	return previews, nil
}

func (r *PostgresPreviewRepository) DeletePreview(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM preview_deployments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete preview: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("preview not found: %s", id)
	}
	return nil
}
