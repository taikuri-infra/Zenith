package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresOrgRepository is a PostgreSQL-backed OrgRepository.
type PostgresOrgRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresOrgRepository(pool *pgxpool.Pool) *PostgresOrgRepository {
	return &PostgresOrgRepository{pool: pool}
}

const orgCols = `id, project_id, name, slug, namespace, is_default, created_at, updated_at`

func scanOrg(s interface{ Scan(dest ...any) error }) (*entities.Org, error) {
	var o entities.Org
	if err := s.Scan(&o.ID, &o.ProjectID, &o.Name, &o.Slug, &o.Namespace, &o.IsDefault, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *PostgresOrgRepository) insert(ctx context.Context, projectID, name string, isDefault bool) (*entities.Org, error) {
	slug := entities.OrgSlug(name)
	if slug == "" {
		return nil, fmt.Errorf("org name must contain letters or digits")
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO orgs (id, project_id, name, slug, namespace, is_default)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING `+orgCols,
		uuid.New().String(), projectID, name, slug, entities.OrgNamespace(projectID, slug), isDefault)
	o, err := scanOrg(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, fmt.Errorf("org '%s' already exists", name)
	}
	return o, err
}

func (r *PostgresOrgRepository) CreateOrg(ctx context.Context, projectID, name string) (*entities.Org, error) {
	return r.insert(ctx, projectID, name, false)
}

func (r *PostgresOrgRepository) GetOrg(ctx context.Context, id string) (*entities.Org, error) {
	return scanOrg(r.pool.QueryRow(ctx, `SELECT `+orgCols+` FROM orgs WHERE id = $1`, id))
}

func (r *PostgresOrgRepository) ListOrgsByProject(ctx context.Context, projectID string) ([]entities.Org, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+orgCols+` FROM orgs WHERE project_id = $1 ORDER BY created_at, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entities.Org{}
	for rows.Next() {
		o, err := scanOrg(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func (r *PostgresOrgRepository) RenameOrg(ctx context.Context, id, name string) (*entities.Org, error) {
	return scanOrg(r.pool.QueryRow(ctx,
		`UPDATE orgs SET name = $2, updated_at = NOW() WHERE id = $1 RETURNING `+orgCols, id, name))
}

func (r *PostgresOrgRepository) DeleteOrg(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM orgs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete org: %w (it still holds resources?)", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("org not found")
	}
	return nil
}

func (r *PostgresOrgRepository) GetDefaultOrg(ctx context.Context, projectID string) (*entities.Org, error) {
	o, err := scanOrg(r.pool.QueryRow(ctx,
		`SELECT `+orgCols+` FROM orgs WHERE project_id = $1 AND is_default`, projectID))
	if err == nil {
		return o, nil
	}
	o, err = r.insert(ctx, projectID, "Default", true)
	if err != nil { // lost a creation race: read the winner
		return scanOrg(r.pool.QueryRow(ctx,
			`SELECT `+orgCols+` FROM orgs WHERE project_id = $1 AND is_default`, projectID))
	}
	return o, nil
}
