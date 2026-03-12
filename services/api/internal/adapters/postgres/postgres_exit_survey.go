package postgres

import (
	"context"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ ports.ExitSurveyRepository = (*PostgresExitSurveyRepository)(nil)

type PostgresExitSurveyRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresExitSurveyRepository(pool *pgxpool.Pool) *PostgresExitSurveyRepository {
	return &PostgresExitSurveyRepository{pool: pool}
}

func (r *PostgresExitSurveyRepository) Create(ctx context.Context, survey *entities.ExitSurvey) error {
	if survey.ID == "" {
		survey.ID = uuid.New().String()
	}
	if survey.CreatedAt.IsZero() {
		survey.CreatedAt = time.Now()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO exit_surveys (id, user_id, reason, details, plan_tier, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		survey.ID, survey.UserID, survey.Reason, survey.Details, survey.PlanTier, survey.CreatedAt,
	)
	return err
}

func (r *PostgresExitSurveyRepository) List(ctx context.Context, limit, offset int) ([]entities.ExitSurvey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, reason, details, plan_tier, created_at
		 FROM exit_surveys ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var surveys []entities.ExitSurvey
	for rows.Next() {
		var s entities.ExitSurvey
		if err := rows.Scan(&s.ID, &s.UserID, &s.Reason, &s.Details, &s.PlanTier, &s.CreatedAt); err != nil {
			return nil, err
		}
		surveys = append(surveys, s)
	}
	return surveys, rows.Err()
}

func (r *PostgresExitSurveyRepository) GetStats(ctx context.Context) (*entities.ExitSurveyStats, error) {
	stats := &entities.ExitSurveyStats{ByReason: make(map[string]int)}
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM exit_surveys`).Scan(&stats.Total)

	rows, err := r.pool.Query(ctx,
		`SELECT reason, COUNT(*) FROM exit_surveys GROUP BY reason`,
	)
	if err != nil {
		return stats, nil
	}
	defer rows.Close()
	for rows.Next() {
		var reason string
		var count int
		if err := rows.Scan(&reason, &count); err == nil {
			stats.ByReason[reason] = count
		}
	}
	return stats, nil
}
