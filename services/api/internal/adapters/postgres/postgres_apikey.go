package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAPIKeyRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAPIKeyRepository(pool *pgxpool.Pool) *PostgresAPIKeyRepository {
	return &PostgresAPIKeyRepository{pool: pool}
}

func (r *PostgresAPIKeyRepository) CreateAPIKey(ctx context.Context, userID, name string, scopes []string) (*entities.APIKey, error) {
	id := uuid.New().String()
	key, prefix, keyHash := generateAPIKeyValues()
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO api_keys (id, user_id, name, key_prefix, key_hash, scopes, type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, userID, name, prefix, keyHash, scopes, string(entities.APIKeyPersonal), now,
	)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return &entities.APIKey{
		ID:        id,
		UserID:    userID,
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   keyHash,
		Key:       key,
		Scopes:    scopes,
		Type:      entities.APIKeyPersonal,
		CreatedAt: now,
	}, nil
}

func (r *PostgresAPIKeyRepository) GetAPIKey(ctx context.Context, id string) (*entities.APIKey, error) {
	var k entities.APIKey
	var keyType string
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, project_id, name, key_prefix, key_hash, scopes, type, last_used_at, expires_at, created_at
		 FROM api_keys WHERE id = $1`, id,
	).Scan(&k.ID, &k.UserID, &k.ProjectID, &k.Name, &k.KeyPrefix, &k.KeyHash,
		&k.Scopes, &keyType, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("api key not found: %s", id)
	}
	k.Type = entities.APIKeyType(keyType)
	return &k, nil
}

func (r *PostgresAPIKeyRepository) GetAPIKeyByHash(ctx context.Context, keyHash string) (*entities.APIKey, error) {
	var k entities.APIKey
	var keyType string
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, project_id, name, key_prefix, key_hash, scopes, type, last_used_at, expires_at, created_at
		 FROM api_keys WHERE key_hash = $1`, keyHash,
	).Scan(&k.ID, &k.UserID, &k.ProjectID, &k.Name, &k.KeyPrefix, &k.KeyHash,
		&k.Scopes, &keyType, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("api key not found")
	}
	k.Type = entities.APIKeyType(keyType)
	return &k, nil
}

func (r *PostgresAPIKeyRepository) ListAPIKeysByUser(ctx context.Context, userID string) ([]entities.APIKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, project_id, name, key_prefix, key_hash, scopes, type, last_used_at, expires_at, created_at
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []entities.APIKey
	for rows.Next() {
		var k entities.APIKey
		var keyType string
		if err := rows.Scan(&k.ID, &k.UserID, &k.ProjectID, &k.Name, &k.KeyPrefix, &k.KeyHash,
			&k.Scopes, &keyType, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		k.Type = entities.APIKeyType(keyType)
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *PostgresAPIKeyRepository) DeleteAPIKey(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("api key not found: %s", id)
	}
	return nil
}

func (r *PostgresAPIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

func (r *PostgresAPIKeyRepository) CountAPIKeysByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM api_keys WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count api keys: %w", err)
	}
	return count, nil
}

func generateAPIKeyValues() (key, prefix, hash string) {
	b := make([]byte, 32)
	rand.Read(b)
	key = "zk_" + hex.EncodeToString(b)
	prefix = key[:10]
	h := sha256.Sum256([]byte(key))
	hash = hex.EncodeToString(h[:])
	return
}
