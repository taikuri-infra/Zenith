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

const userSelectCols = `id, email, name, role, password_hash, email_verified, email_verified_at, auth_provider, created_at, updated_at,
	COALESCE(signup_source,''), COALESCE(utm_source,''), COALESCE(utm_medium,''), COALESCE(utm_campaign,''),
	COALESCE(utm_content,''), COALESCE(utm_term,''), COALESCE(referrer_url,''),
	signup_ip, COALESCE(onboarding_completed, false), COALESCE(onboarding_step, 0), onboarding_completed_at,
	referral_code, referred_by, last_login_at`

func (s *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*ports.StoredUser, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE email = $1`, email)
	return scanStoredUser(row)
}

func (s *PostgresUserRepository) GetByID(ctx context.Context, id string) (*ports.StoredUser, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE id = $1`, id)
	return scanStoredUser(row)
}

// UpdateSignupSource sets UTM and signup source fields on user creation.
func (s *PostgresUserRepository) UpdateSignupSource(ctx context.Context, userID string, source, utmSource, utmMedium, utmCampaign, utmContent, utmTerm, referrerURL, signupIP string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET signup_source = $2, utm_source = $3, utm_medium = $4, utm_campaign = $5,
		 utm_content = $6, utm_term = $7, referrer_url = $8, signup_ip = $9::inet, updated_at = NOW()
		 WHERE id = $1`,
		userID, source, utmSource, utmMedium, utmCampaign, utmContent, utmTerm, referrerURL, nullableString(signupIP),
	)
	return err
}

// UpdateOnboarding updates the user's onboarding progress.
func (s *PostgresUserRepository) UpdateOnboarding(ctx context.Context, userID string, step int, completed bool) error {
	if completed {
		now := time.Now()
		_, err := s.pool.Exec(ctx,
			`UPDATE users SET onboarding_step = $2, onboarding_completed = true, onboarding_completed_at = $3, updated_at = NOW() WHERE id = $1`,
			userID, step, now,
		)
		return err
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET onboarding_step = $2, updated_at = NOW() WHERE id = $1`,
		userID, step,
	)
	return err
}

// SetReferralCode sets the user's referral code.
func (s *PostgresUserRepository) SetReferralCode(ctx context.Context, userID, code string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET referral_code = $2, updated_at = NOW() WHERE id = $1`,
		userID, code,
	)
	return err
}

// SetReferredBy marks who referred this user.
func (s *PostgresUserRepository) SetReferredBy(ctx context.Context, userID, referrerID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET referred_by = $2, updated_at = NOW() WHERE id = $1`,
		userID, referrerID,
	)
	return err
}

// GetByReferralCode finds a user by their referral code.
func (s *PostgresUserRepository) GetByReferralCode(ctx context.Context, code string) (*ports.StoredUser, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE referral_code = $1`, code)
	return scanStoredUser(row)
}

// UpdateLastLogin updates the user's last login timestamp.
func (s *PostgresUserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, userID,
	)
	return err
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

// nullableString returns nil if empty, for SQL NULL values.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func scanStoredUser(row pgx.Row) (*ports.StoredUser, error) {
	var u ports.StoredUser
	var role string
	var signupIP, referralCode, referredBy *string
	err := row.Scan(&u.ID, &u.Email, &u.Name, &role, &u.PasswordHash,
		&u.EmailVerified, &u.EmailVerifiedAt, &u.AuthProvider,
		&u.CreatedAt, &u.UpdatedAt,
		&u.SignupSource, &u.UTMSource, &u.UTMMedium, &u.UTMCampaign,
		&u.UTMContent, &u.UTMTerm, &u.ReferrerURL,
		&signupIP, &u.OnboardingCompleted, &u.OnboardingStep, &u.OnboardingCompletedAt,
		&referralCode, &referredBy, &u.LastLoginAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Role = entities.Role(role)
	if signupIP != nil {
		u.SignupIP = *signupIP
	}
	if referralCode != nil {
		u.ReferralCode = *referralCode
	}
	if referredBy != nil {
		u.ReferredBy = *referredBy
	}
	return &u, nil
}
