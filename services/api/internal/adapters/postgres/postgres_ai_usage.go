package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAIUsageRepository tracks AI feature usage in PostgreSQL.
type PostgresAIUsageRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAIUsageRepository creates a new PostgresAIUsageRepository.
func NewPostgresAIUsageRepository(pool *pgxpool.Pool) *PostgresAIUsageRepository {
	return &PostgresAIUsageRepository{pool: pool}
}

// RecordUsage inserts an AI usage record.
func (r *PostgresAIUsageRepository) RecordUsage(ctx context.Context, userID, usageType, model string, tokensIn, tokensOut int, costUSD float64) error {
	id := uuid.New().String()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ai_usage (id, user_id, usage_type, model, tokens_in, tokens_out, cost_usd, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		id, userID, usageType, model, tokensIn, tokensOut, costUSD,
	)
	return err
}

// GetMonthlyUsage returns the number of AI calls a user has made in the given month.
func (r *PostgresAIUsageRepository) GetMonthlyUsage(ctx context.Context, userID string, month time.Time) (int, error) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ai_usage WHERE user_id = $1 AND created_at >= $2 AND created_at < $3`,
		userID, start, end,
	).Scan(&count)
	return count, err
}
