package postgres

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
)

// Argon2id parameters (OWASP recommended)
const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// PostgresDeployTokenRepository is a PostgreSQL-backed DeployTokenRepository.
type PostgresDeployTokenRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresDeployTokenRepository creates a new PostgreSQL DeployTokenRepository.
func NewPostgresDeployTokenRepository(pool *pgxpool.Pool) *PostgresDeployTokenRepository {
	return &PostgresDeployTokenRepository{pool: pool}
}

const deployTokenSelectCols = `id, user_id, project_id, name, token_id, token_prefix, token_hash,
	scopes, last_used_at, expires_at, COALESCE(previous_hash, ''), previous_expires_at,
	rotated_at, created_at, revoked_at`

func scanDeployToken(scanner interface{ Scan(dest ...any) error }) (*entities.DeployToken, error) {
	var dt entities.DeployToken
	err := scanner.Scan(
		&dt.ID, &dt.UserID, &dt.ProjectID, &dt.Name, &dt.TokenID, &dt.TokenPrefix, &dt.TokenHash,
		&dt.Scopes, &dt.LastUsedAt, &dt.ExpiresAt, &dt.PreviousHash, &dt.PreviousExpiresAt,
		&dt.RotatedAt, &dt.CreatedAt, &dt.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	return &dt, nil
}

// CreateDeployToken generates a new deploy token with Argon2id-hashed secret.
// Returns the token with the plain-text secret set (only time it's available).
func (r *PostgresDeployTokenRepository) CreateDeployToken(ctx context.Context, userID, projectID, name string, scopes []string, expiresAt *time.Time) (*entities.DeployToken, error) {
	id := uuid.New().String()

	// Generate token ID: znt_id_ + 16 hex chars
	tokenIDBytes := make([]byte, 8)
	if _, err := rand.Read(tokenIDBytes); err != nil {
		return nil, fmt.Errorf("generate token id: %w", err)
	}
	tokenID := "znt_id_" + hex.EncodeToString(tokenIDBytes)

	// Generate token secret: znt_sk_ + 64 hex chars
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate token secret: %w", err)
	}
	secret := "znt_sk_" + hex.EncodeToString(secretBytes)
	prefix := secret[:15] // "znt_sk_" + first 8 hex chars

	// Hash secret with Argon2id
	hash, err := hashArgon2id(secret)
	if err != nil {
		return nil, fmt.Errorf("hash token secret: %w", err)
	}

	now := time.Now()
	dt := &entities.DeployToken{
		ID:          id,
		UserID:      userID,
		ProjectID:   projectID,
		Name:        name,
		TokenID:     tokenID,
		TokenPrefix: prefix,
		TokenHash:   hash,
		Secret:      secret, // returned only on creation
		Scopes:      scopes,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO deploy_tokens (id, user_id, project_id, name, token_id, token_prefix, token_hash, scopes, expires_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		dt.ID, dt.UserID, dt.ProjectID, dt.Name, dt.TokenID, dt.TokenPrefix, dt.TokenHash,
		dt.Scopes, dt.ExpiresAt, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create deploy token: %w", err)
	}

	return dt, nil
}

// GetDeployTokenByTokenID looks up a deploy token by its public token_id.
func (r *PostgresDeployTokenRepository) GetDeployTokenByTokenID(ctx context.Context, tokenID string) (*entities.DeployToken, error) {
	dt, err := scanDeployToken(r.pool.QueryRow(ctx,
		`SELECT `+deployTokenSelectCols+` FROM deploy_tokens WHERE token_id = $1`, tokenID,
	))
	if err != nil {
		return nil, fmt.Errorf("deploy token not found: %s", tokenID)
	}
	return dt, nil
}

// GetDeployToken retrieves a deploy token by its internal ID.
func (r *PostgresDeployTokenRepository) GetDeployToken(ctx context.Context, id string) (*entities.DeployToken, error) {
	dt, err := scanDeployToken(r.pool.QueryRow(ctx,
		`SELECT `+deployTokenSelectCols+` FROM deploy_tokens WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("deploy token not found: %s", id)
	}
	return dt, nil
}

// ListDeployTokensByProject returns all tokens for a project (without secrets).
func (r *PostgresDeployTokenRepository) ListDeployTokensByProject(ctx context.Context, projectID string) ([]entities.DeployToken, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+deployTokenSelectCols+` FROM deploy_tokens WHERE project_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list deploy tokens: %w", err)
	}
	defer rows.Close()

	var tokens []entities.DeployToken
	for rows.Next() {
		dt, err := scanDeployToken(rows)
		if err != nil {
			return nil, fmt.Errorf("scan deploy token: %w", err)
		}
		tokens = append(tokens, *dt)
	}
	return tokens, nil
}

// RevokeDeployToken soft-deletes a deploy token.
func (r *PostgresDeployTokenRepository) RevokeDeployToken(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE deploy_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("revoke deploy token: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("deploy token not found or already revoked: %s", id)
	}
	return nil
}

// RotateDeployToken generates a new secret, keeps the old hash valid for 24h grace period.
func (r *PostgresDeployTokenRepository) RotateDeployToken(ctx context.Context, id string) (*entities.DeployToken, error) {
	dt, err := r.GetDeployToken(ctx, id)
	if err != nil {
		return nil, err
	}
	if dt.IsRevoked() {
		return nil, fmt.Errorf("cannot rotate a revoked token")
	}

	// Generate new secret
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate new secret: %w", err)
	}
	newSecret := "znt_sk_" + hex.EncodeToString(secretBytes)
	newPrefix := newSecret[:15]

	newHash, err := hashArgon2id(newSecret)
	if err != nil {
		return nil, fmt.Errorf("hash new secret: %w", err)
	}

	// Old hash remains valid for 24h grace period
	graceExpiry := time.Now().Add(24 * time.Hour)

	_, err = r.pool.Exec(ctx,
		`UPDATE deploy_tokens SET
			previous_hash = token_hash,
			previous_expires_at = $1,
			token_hash = $2,
			token_prefix = $3,
			rotated_at = now()
		 WHERE id = $4`,
		graceExpiry, newHash, newPrefix, id,
	)
	if err != nil {
		return nil, fmt.Errorf("rotate deploy token: %w", err)
	}

	dt.Secret = newSecret
	dt.TokenPrefix = newPrefix
	dt.TokenHash = newHash
	return dt, nil
}

// UpdateLastUsed updates the last_used_at timestamp.
func (r *PostgresDeployTokenRepository) UpdateLastUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE deploy_tokens SET last_used_at = now() WHERE id = $1`, id,
	)
	return err
}

// VerifySecret checks if a plain-text secret matches the stored hash (or the grace-period previous hash).
func (r *PostgresDeployTokenRepository) VerifySecret(dt *entities.DeployToken, secret string) bool {
	if verifyArgon2id(secret, dt.TokenHash) {
		return true
	}
	// Check grace period hash
	if dt.InGracePeriod() && verifyArgon2id(secret, dt.PreviousHash) {
		return true
	}
	return false
}

// ---- Argon2id helpers ----

// hashArgon2id produces a hash string in the format:
// $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
func hashArgon2id(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// verifyArgon2id verifies a password against an argon2id hash string.
func verifyArgon2id(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	var memory uint32
	var time uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expectedHash)))

	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}
