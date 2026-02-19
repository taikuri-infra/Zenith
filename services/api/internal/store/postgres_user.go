package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Compile-time interface check.
var _ UserRepository = (*PostgresUserRepository)(nil)

// PostgresUserRepository persists users in PostgreSQL.
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (s *PostgresUserRepository) Create(ctx context.Context, email, password, name string, role models.Role) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	id := uuid.New().String()

	_, err = s.pool.Exec(ctx,
		`INSERT INTO users (id, email, name, role, password_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, email, name, string(role), string(hash), now, now,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("email already registered")
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &models.User{
		ID:        id,
		Email:     email,
		Name:      name,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*StoredUser, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, email, name, role, password_hash, created_at, updated_at FROM users WHERE email = $1`, email)
	return scanStoredUser(row)
}

func (s *PostgresUserRepository) GetByID(ctx context.Context, id string) (*StoredUser, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, email, name, role, password_hash, created_at, updated_at FROM users WHERE id = $1`, id)
	return scanStoredUser(row)
}

func (s *PostgresUserRepository) CheckPassword(user *StoredUser, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}

func (s *PostgresUserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func scanStoredUser(row pgx.Row) (*StoredUser, error) {
	var u StoredUser
	var role string
	err := row.Scan(&u.ID, &u.Email, &u.Name, &role, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Role = models.Role(role)
	return &u, nil
}
