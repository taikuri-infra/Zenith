package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Compile-time interface check.
var _ ports.UserRepository = (*PostgresUserRepository)(nil)

// PostgresUserRepository persists users in PostgreSQL.
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (s *PostgresUserRepository) Create(ctx context.Context, email, password, name string, role entities.Role) (*entities.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	id := uuid.New().String()

	_, err = s.pool.Exec(ctx,
		`INSERT INTO users (id, email, name, role, password_hash, email_verified, auth_provider, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, email, name, string(role), string(hash), false, "email", now, now,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("email already registered")
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &entities.User{
		ID:           id,
		Email:        email,
		Name:         name,
		Role:         role,
		AuthProvider: "email",
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (s *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*ports.StoredUser, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, email, name, role, password_hash, email_verified, email_verified_at, auth_provider, created_at, updated_at FROM users WHERE email = $1`, email)
	return scanStoredUser(row)
}

func (s *PostgresUserRepository) GetByID(ctx context.Context, id string) (*ports.StoredUser, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, email, name, role, password_hash, email_verified, email_verified_at, auth_provider, created_at, updated_at FROM users WHERE id = $1`, id)
	return scanStoredUser(row)
}

func (s *PostgresUserRepository) CheckPassword(user *ports.StoredUser, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}

func (s *PostgresUserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (s *PostgresUserRepository) SetEmailVerified(ctx context.Context, userID string) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET email_verified = true, email_verified_at = $1, updated_at = $1 WHERE id = $2`,
		now, userID,
	)
	return err
}

func (s *PostgresUserRepository) SetAuthProvider(ctx context.Context, userID, provider string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET auth_provider = $1, updated_at = now() WHERE id = $2`,
		provider, userID,
	)
	return err
}

func (s *PostgresUserRepository) CreateVerificationToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO email_verification_tokens (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		id, userID, tokenHash, expiresAt,
	)
	return err
}

func (s *PostgresUserRepository) GetVerificationToken(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx,
		`SELECT user_id FROM email_verification_tokens WHERE token_hash = $1 AND expires_at > now()`,
		tokenHash,
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("invalid or expired verification token")
		}
		return "", err
	}
	return userID, nil
}

func (s *PostgresUserRepository) DeleteVerificationTokens(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM email_verification_tokens WHERE user_id = $1`, userID)
	return err
}

func scanStoredUser(row pgx.Row) (*ports.StoredUser, error) {
	var u ports.StoredUser
	var role string
	err := row.Scan(&u.ID, &u.Email, &u.Name, &role, &u.PasswordHash,
		&u.EmailVerified, &u.EmailVerifiedAt, &u.AuthProvider,
		&u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Role = entities.Role(role)
	return &u, nil
}
