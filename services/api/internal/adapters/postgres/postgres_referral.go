package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ ports.ReferralRepository = (*PostgresReferralRepository)(nil)

type PostgresReferralRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresReferralRepository(pool *pgxpool.Pool) *PostgresReferralRepository {
	return &PostgresReferralRepository{pool: pool}
}

func (r *PostgresReferralRepository) CreateReward(ctx context.Context, reward *entities.ReferralReward) error {
	if reward.ID == "" {
		reward.ID = uuid.New().String()
	}
	if reward.CreatedAt.IsZero() {
		reward.CreatedAt = time.Now()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO referral_rewards (id, referrer_id, referred_id, status, reward_type, reward_amount, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (referrer_id, referred_id) DO NOTHING`,
		reward.ID, reward.ReferrerID, reward.ReferredID, reward.Status,
		reward.RewardType, reward.RewardAmount, reward.CreatedAt,
	)
	return err
}

func (r *PostgresReferralRepository) ListByReferrer(ctx context.Context, referrerID string) ([]entities.ReferralReward, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, referrer_id, referred_id, status, reward_type, reward_amount, credited_at, created_at
		 FROM referral_rewards WHERE referrer_id = $1 ORDER BY created_at DESC`, referrerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReferralRewards(rows)
}

func (r *PostgresReferralRepository) CountByReferrer(ctx context.Context, referrerID string, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM referral_rewards WHERE referrer_id = $1 AND created_at >= $2`,
		referrerID, since,
	).Scan(&count)
	return count, err
}

func (r *PostgresReferralRepository) CreditReward(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE referral_rewards SET status = $1, credited_at = NOW() WHERE id = $2`,
		entities.ReferralCredited, id,
	)
	return err
}

func (r *PostgresReferralRepository) GetSummary(ctx context.Context, userID, baseURL string) (*entities.ReferralSummary, error) {
	var code *string
	_ = r.pool.QueryRow(ctx, `SELECT referral_code FROM users WHERE id = $1`, userID).Scan(&code)

	summary := &entities.ReferralSummary{}
	if code != nil && *code != "" {
		summary.Code = *code
		summary.Link = fmt.Sprintf("%s/r/%s", baseURL, *code)
	}

	_ = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM referral_rewards WHERE referrer_id = $1`, userID,
	).Scan(&summary.TotalReferrals)
	_ = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM referral_rewards WHERE referrer_id = $1 AND status = $2`,
		userID, entities.ReferralCredited,
	).Scan(&summary.Credited)
	_ = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM referral_rewards WHERE referrer_id = $1 AND status = $2`,
		userID, entities.ReferralPending,
	).Scan(&summary.Pending)

	return summary, nil
}

func (r *PostgresReferralRepository) ListAll(ctx context.Context, limit, offset int) ([]entities.ReferralReward, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, referrer_id, referred_id, status, reward_type, reward_amount, credited_at, created_at
		 FROM referral_rewards ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReferralRewards(rows)
}

func scanReferralRewards(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]entities.ReferralReward, error) {
	var rewards []entities.ReferralReward
	for rows.Next() {
		var rr entities.ReferralReward
		if err := rows.Scan(&rr.ID, &rr.ReferrerID, &rr.ReferredID, &rr.Status,
			&rr.RewardType, &rr.RewardAmount, &rr.CreditedAt, &rr.CreatedAt); err != nil {
			return nil, err
		}
		rewards = append(rewards, rr)
	}
	return rewards, rows.Err()
}
