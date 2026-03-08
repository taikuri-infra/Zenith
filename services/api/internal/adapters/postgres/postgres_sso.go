package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSSORepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSSORepository(pool *pgxpool.Pool) *PostgresSSORepository {
	return &PostgresSSORepository{pool: pool}
}

func (r *PostgresSSORepository) CreateConfig(ctx context.Context, userID string, provider entities.SSOProvider, config *entities.SSOConfig) (*entities.SSOConfig, error) {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sso_configs (id, user_id, provider, entity_id, sso_url, certificate,
		 client_id, client_secret, discovery_url, enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		config.ID, userID, string(provider), config.EntityID, config.SSOURL, config.Certificate,
		config.ClientID, config.ClientSecret, config.DiscoveryURL, config.Enabled, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create sso config: %w", err)
	}
	config.UserID = userID
	config.Provider = provider
	config.CreatedAt = now
	config.UpdatedAt = now
	return config, nil
}

func (r *PostgresSSORepository) GetConfig(ctx context.Context, id string) (*entities.SSOConfig, error) {
	var c entities.SSOConfig
	var provider string
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, provider, entity_id, sso_url, certificate,
		 client_id, client_secret, discovery_url, enabled, created_at, updated_at
		 FROM sso_configs WHERE id = $1`, id,
	).Scan(&c.ID, &c.UserID, &provider, &c.EntityID, &c.SSOURL, &c.Certificate,
		&c.ClientID, &c.ClientSecret, &c.DiscoveryURL, &c.Enabled, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("sso config not found: %s", id)
	}
	c.Provider = entities.SSOProvider(provider)
	return &c, nil
}

func (r *PostgresSSORepository) ListConfigsByUser(ctx context.Context, userID string) ([]entities.SSOConfig, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, provider, entity_id, sso_url, certificate,
		 client_id, client_secret, discovery_url, enabled, created_at, updated_at
		 FROM sso_configs WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list sso configs: %w", err)
	}
	defer rows.Close()

	var configs []entities.SSOConfig
	for rows.Next() {
		var c entities.SSOConfig
		var provider string
		if err := rows.Scan(&c.ID, &c.UserID, &provider, &c.EntityID, &c.SSOURL, &c.Certificate,
			&c.ClientID, &c.ClientSecret, &c.DiscoveryURL, &c.Enabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sso config: %w", err)
		}
		c.Provider = entities.SSOProvider(provider)
		configs = append(configs, c)
	}
	return configs, nil
}

func (r *PostgresSSORepository) DeleteConfig(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM sso_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sso config: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("sso config not found: %s", id)
	}
	return nil
}

func (r *PostgresSSORepository) ToggleConfig(ctx context.Context, id string, enabled bool) (*entities.SSOConfig, error) {
	ct, err := r.pool.Exec(ctx,
		`UPDATE sso_configs SET enabled = $1, updated_at = $2 WHERE id = $3`,
		enabled, time.Now(), id,
	)
	if err != nil {
		return nil, fmt.Errorf("toggle sso config: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("sso config not found: %s", id)
	}
	return r.GetConfig(ctx, id)
}
