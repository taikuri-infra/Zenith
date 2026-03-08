package postgres

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"math/big"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresMFARepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMFARepository(pool *pgxpool.Pool) *PostgresMFARepository {
	return &PostgresMFARepository{pool: pool}
}

func (r *PostgresMFARepository) GetEnrollment(ctx context.Context, userID string) (*entities.MFAEnrollment, error) {
	var e entities.MFAEnrollment
	var status string
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, status, secret, backup_codes, used_codes, enabled_at, created_at
		 FROM mfa_enrollments WHERE user_id = $1`, userID,
	).Scan(&e.UserID, &status, &e.Secret, &e.BackupCodes, &e.UsedCodes, &e.EnabledAt, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("MFA not enrolled for user %s", userID)
	}
	e.Status = entities.MFAStatus(status)
	return &e, nil
}

func (r *PostgresMFARepository) StartEnrollment(ctx context.Context, userID string) (*entities.MFAEnrollment, error) {
	now := time.Now()
	secret := generateMFATOTPSecret()
	codes := generateMFABackupCodes(8)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO mfa_enrollments (user_id, status, secret, backup_codes, used_codes, created_at)
		 VALUES ($1, $2, $3, $4, '{}', $5)
		 ON CONFLICT (user_id) DO UPDATE SET status = $2, secret = $3, backup_codes = $4, used_codes = '{}', created_at = $5`,
		userID, string(entities.MFAStatusPending), secret, codes, now,
	)
	if err != nil {
		return nil, fmt.Errorf("start enrollment: %w", err)
	}
	return &entities.MFAEnrollment{
		UserID:      userID,
		Status:      entities.MFAStatusPending,
		Secret:      secret,
		BackupCodes: codes,
		CreatedAt:   now,
	}, nil
}

func (r *PostgresMFARepository) ConfirmEnrollment(ctx context.Context, userID string) (*entities.MFAEnrollment, error) {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE mfa_enrollments SET status = $1, enabled_at = $2 WHERE user_id = $3 AND status = $4`,
		string(entities.MFAStatusEnabled), now, userID, string(entities.MFAStatusPending),
	)
	if err != nil {
		return nil, fmt.Errorf("confirm enrollment: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("no pending enrollment for user %s", userID)
	}
	return r.GetEnrollment(ctx, userID)
}

func (r *PostgresMFARepository) DisableEnrollment(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM mfa_enrollments WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("disable enrollment: %w", err)
	}
	return nil
}

func (r *PostgresMFARepository) UseBackupCode(ctx context.Context, userID, code string) (bool, error) {
	e, err := r.GetEnrollment(ctx, userID)
	if err != nil {
		return false, err
	}
	// Check if code is valid and not used
	found := false
	for _, c := range e.BackupCodes {
		if c == code {
			found = true
			break
		}
	}
	if !found {
		return false, nil
	}
	for _, c := range e.UsedCodes {
		if c == code {
			return false, nil // already used
		}
	}
	// Mark code as used
	_, err = r.pool.Exec(ctx,
		`UPDATE mfa_enrollments SET used_codes = array_append(used_codes, $1) WHERE user_id = $2`,
		code, userID,
	)
	if err != nil {
		return false, fmt.Errorf("use backup code: %w", err)
	}
	return true, nil
}

func (r *PostgresMFARepository) RegenerateBackupCodes(ctx context.Context, userID string) ([]string, error) {
	codes := generateMFABackupCodes(8)
	ct, err := r.pool.Exec(ctx,
		`UPDATE mfa_enrollments SET backup_codes = $1, used_codes = '{}' WHERE user_id = $2`,
		codes, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("regenerate backup codes: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("MFA not enrolled for user %s", userID)
	}
	return codes, nil
}

func generateMFATOTPSecret() string {
	b := make([]byte, 20)
	rand.Read(b)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

func generateMFABackupCodes(n int) []string {
	codes := make([]string, n)
	for i := range codes {
		num, _ := rand.Int(rand.Reader, big.NewInt(100000000))
		codes[i] = fmt.Sprintf("%08d", num.Int64())
	}
	return codes
}
