package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type PostgresAppAuthRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAppAuthRepository(pool *pgxpool.Pool) *PostgresAppAuthRepository {
	return &PostgresAppAuthRepository{pool: pool}
}

func (r *PostgresAppAuthRepository) EnableAuth(ctx context.Context, appID string, maxUsers int) (*entities.AppAuthConfig, error) {
	now := time.Now()
	secret := generateJWTSecret()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO app_auth_configs (app_id, enabled, max_users, jwt_secret, created_at, updated_at)
		 VALUES ($1, true, $2, $3, $4, $5)
		 ON CONFLICT (app_id) DO UPDATE SET enabled = true, max_users = $2, updated_at = $5`,
		appID, maxUsers, secret, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("enable auth: %w", err)
	}
	return r.GetAuthConfig(ctx, appID)
}

func (r *PostgresAppAuthRepository) DisableAuth(ctx context.Context, appID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE app_auth_configs SET enabled = false, updated_at = $1 WHERE app_id = $2`,
		time.Now(), appID,
	)
	if err != nil {
		return fmt.Errorf("disable auth: %w", err)
	}
	return nil
}

func (r *PostgresAppAuthRepository) GetAuthConfig(ctx context.Context, appID string) (*entities.AppAuthConfig, error) {
	var cfg entities.AppAuthConfig
	err := r.pool.QueryRow(ctx,
		`SELECT app_id, enabled, max_users, jwt_secret, created_at, updated_at FROM app_auth_configs WHERE app_id = $1`,
		appID,
	).Scan(&cfg.AppID, &cfg.Enabled, &cfg.MaxUsers, &cfg.JWTSecret, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("auth config not found for app %s", appID)
	}
	return &cfg, nil
}

func (r *PostgresAppAuthRepository) CreateAppUser(ctx context.Context, appID, email, password, name string) (*entities.AppUser, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	id := uuid.New().String()
	now := time.Now()
	_, err = r.pool.Exec(ctx,
		`INSERT INTO app_auth_users (id, app_id, email, name, password_hash, verified, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, false, $6, $7)`,
		id, appID, email, name, string(hash), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create app user: %w", err)
	}
	return &entities.AppUser{
		ID:    id,
		AppID: appID,
		Email: email,
		Name:  name,
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

func (r *PostgresAppAuthRepository) GetAppUserByEmail(ctx context.Context, appID, email string) (*entities.AppUser, string, error) {
	var u entities.AppUser
	var passwordHash string
	err := r.pool.QueryRow(ctx,
		`SELECT id, app_id, email, name, password_hash, verified, created_at, updated_at
		 FROM app_auth_users WHERE app_id = $1 AND email = $2`,
		appID, email,
	).Scan(&u.ID, &u.AppID, &u.Email, &u.Name, &passwordHash, &u.Verified, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("user not found")
	}
	return &u, passwordHash, nil
}

func (r *PostgresAppAuthRepository) CountAppUsers(ctx context.Context, appID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM app_auth_users WHERE app_id = $1`, appID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count app users: %w", err)
	}
	return count, nil
}

func (r *PostgresAppAuthRepository) ListAppUsers(ctx context.Context, appID string, limit, offset int) ([]entities.AppUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, app_id, email, name, verified, created_at, updated_at
		 FROM app_auth_users WHERE app_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		appID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list app users: %w", err)
	}
	defer rows.Close()

	var users []entities.AppUser
	for rows.Next() {
		var u entities.AppUser
		if err := rows.Scan(&u.ID, &u.AppID, &u.Email, &u.Name, &u.Verified, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan app user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *PostgresAppAuthRepository) DeleteAppUser(ctx context.Context, appID, userID string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM app_auth_users WHERE app_id = $1 AND id = $2`, appID, userID)
	if err != nil {
		return fmt.Errorf("delete app user: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func generateJWTSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
