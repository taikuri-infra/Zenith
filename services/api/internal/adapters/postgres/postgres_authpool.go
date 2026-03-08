package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAuthPoolRepository is a PostgreSQL-backed AuthPoolRepository.
type PostgresAuthPoolRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAuthPoolRepository creates a new PostgreSQL AuthPoolRepository.
func NewPostgresAuthPoolRepository(pool *pgxpool.Pool) *PostgresAuthPoolRepository {
	return &PostgresAuthPoolRepository{pool: pool}
}

const authPoolSelectCols = `id, user_id, project_id, name, realm_name, client_id, client_secret, issuer_url, status, user_count, max_users, created_at, updated_at`

func scanAuthPool(scanner interface{ Scan(dest ...any) error }) (*entities.AuthPool, error) {
	var p entities.AuthPool
	err := scanner.Scan(
		&p.ID, &p.UserID, &p.ProjectID, &p.Name, &p.RealmName,
		&p.ClientID, &p.ClientSecret, &p.IssuerURL, &p.Status,
		&p.UserCount, &p.MaxUsers, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PostgresAuthPoolRepository) CreatePool(ctx context.Context, id, userID, projectID, name, realmName, clientID, clientSecret, issuerURL string, maxUsers int) (*entities.AuthPool, error) {
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO auth_pools (id, user_id, project_id, name, realm_name, client_id, client_secret, issuer_url, status, max_users, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		id, userID, projectID, name, realmName, clientID, clientSecret, issuerURL,
		string(entities.AuthPoolStatusProvisioning), maxUsers, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_auth_pools_user_name") {
			return nil, fmt.Errorf("auth pool %q already exists", name)
		}
		return nil, fmt.Errorf("create auth pool: %w", err)
	}

	return &entities.AuthPool{
		ID: id, UserID: userID, ProjectID: projectID, Name: name,
		RealmName: realmName, ClientID: clientID, ClientSecret: clientSecret,
		IssuerURL: issuerURL, Status: entities.AuthPoolStatusProvisioning,
		MaxUsers: maxUsers,
		Timestamps: entities.Timestamps{CreatedAt: now, UpdatedAt: now},
	}, nil
}

func (r *PostgresAuthPoolRepository) GetPool(ctx context.Context, id string) (*entities.AuthPool, error) {
	p, err := scanAuthPool(r.pool.QueryRow(ctx,
		`SELECT `+authPoolSelectCols+` FROM auth_pools WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("auth pool not found: %s", id)
	}
	return p, nil
}

func (r *PostgresAuthPoolRepository) ListPoolsByUser(ctx context.Context, userID string) ([]entities.AuthPool, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+authPoolSelectCols+` FROM auth_pools WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list auth pools: %w", err)
	}
	defer rows.Close()

	var pools []entities.AuthPool
	for rows.Next() {
		p, err := scanAuthPool(rows)
		if err != nil {
			return nil, fmt.Errorf("scan auth pool: %w", err)
		}
		pools = append(pools, *p)
	}
	return pools, nil
}

func (r *PostgresAuthPoolRepository) DeletePool(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM auth_pools WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete auth pool: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("auth pool not found: %s", id)
	}
	return nil
}

func (r *PostgresAuthPoolRepository) UpdatePoolStatus(ctx context.Context, id string, status entities.AuthPoolStatus) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE auth_pools SET status = $1, updated_at = $2 WHERE id = $3`,
		string(status), time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("update auth pool status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("auth pool not found: %s", id)
	}
	return nil
}

func (r *PostgresAuthPoolRepository) UpdatePoolUserCount(ctx context.Context, id string, delta int) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE auth_pools SET user_count = user_count + $1, updated_at = $2 WHERE id = $3`,
		delta, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("update auth pool user count: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("auth pool not found: %s", id)
	}
	return nil
}

func (r *PostgresAuthPoolRepository) CountPoolsByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM auth_pools WHERE user_id = $1`, userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count auth pools: %w", err)
	}
	return count, nil
}
