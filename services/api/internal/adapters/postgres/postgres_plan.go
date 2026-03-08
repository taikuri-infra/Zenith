package postgres

import (
	"context"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUserPlanRepository reads user plan data from the subscriptions table.
type PostgresUserPlanRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserPlanRepository creates a new PostgreSQL UserPlanRepository.
func NewPostgresUserPlanRepository(pool *pgxpool.Pool) *PostgresUserPlanRepository {
	return &PostgresUserPlanRepository{pool: pool}
}

func (r *PostgresUserPlanRepository) GetUserPlan(ctx context.Context, userID string) (*entities.UserPlan, error) {
	var tier string
	var createdAt, updatedAt time.Time
	var stripeSubID *string
	var periodEnd *time.Time
	var cancelAtEnd bool

	err := r.pool.QueryRow(ctx,
		`SELECT tier, stripe_subscription_id, current_period_end, cancel_at_period_end, created_at, updated_at
		 FROM subscriptions
		 WHERE user_id = $1 AND status = 'active'
		 ORDER BY created_at DESC
		 LIMIT 1`, userID).Scan(&tier, &stripeSubID, &periodEnd, &cancelAtEnd, &createdAt, &updatedAt)

	if err == pgx.ErrNoRows {
		// No active subscription — default to free
		now := time.Now()
		defaults := entities.DefaultPlanLimits(entities.PlanFree)
		return &entities.UserPlan{
			UserID: userID,
			Tier:   entities.PlanFree,
			Limits: defaults,
			Timestamps: entities.Timestamps{
				CreatedAt: now,
				UpdatedAt: now,
			},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	planTier := entities.PlanTier(tier)
	limits := entities.DefaultPlanLimits(planTier)

	plan := &entities.UserPlan{
		UserID: userID,
		Tier:   planTier,
		Limits: limits,
		Timestamps: entities.Timestamps{
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		CancelAtPeriodEnd: cancelAtEnd,
	}
	if stripeSubID != nil {
		plan.StripeSubscriptionID = *stripeSubID
	}
	if periodEnd != nil {
		plan.CurrentPeriodEnd = periodEnd
	}

	return plan, nil
}

func (r *PostgresUserPlanRepository) SetUserPlan(ctx context.Context, userID string, tier entities.PlanTier) (*entities.UserPlan, error) {
	now := time.Now()

	// Upsert: update existing active subscription or insert new one
	_, err := r.pool.Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, stripe_subscription_id, stripe_customer_id, stripe_price_id, tier, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $7)
		 ON CONFLICT (stripe_subscription_id) DO UPDATE SET tier = $6, updated_at = $7`,
		uuid.New().String(), userID, "manual_"+uuid.New().String(), "manual", "manual", string(tier), now)
	if err != nil {
		return nil, err
	}

	return r.GetUserPlan(ctx, userID)
}

func (r *PostgresUserPlanRepository) ListUsersByPlan(ctx context.Context, tier entities.PlanTier) ([]entities.UserPlan, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, tier, created_at, updated_at FROM subscriptions WHERE tier = $1 AND status = 'active'`,
		string(tier))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entities.UserPlan
	for rows.Next() {
		var userID, t string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&userID, &t, &createdAt, &updatedAt); err != nil {
			continue
		}
		planTier := entities.PlanTier(t)
		result = append(result, entities.UserPlan{
			UserID: userID,
			Tier:   planTier,
			Limits: entities.DefaultPlanLimits(planTier),
			Timestamps: entities.Timestamps{
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
		})
	}
	return result, nil
}

func (r *PostgresUserPlanRepository) ListAllPlans(ctx context.Context) ([]entities.UserPlan, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, tier, created_at, updated_at FROM subscriptions WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entities.UserPlan
	for rows.Next() {
		var userID, t string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&userID, &t, &createdAt, &updatedAt); err != nil {
			continue
		}
		planTier := entities.PlanTier(t)
		result = append(result, entities.UserPlan{
			UserID: userID,
			Tier:   planTier,
			Limits: entities.DefaultPlanLimits(planTier),
			Timestamps: entities.Timestamps{
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
		})
	}
	return result, nil
}
