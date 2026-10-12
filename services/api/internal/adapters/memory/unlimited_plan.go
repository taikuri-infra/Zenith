package memory

import (
	"context"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// UnlimitedPlanRepository is used in standalone self-host mode. The customer
// owns the hardware, so SaaS plan tiers do not apply: every user reads as
// enterprise with enterprise limits. Writes and listings pass through so the
// (unused) SaaS admin code keeps working.
type UnlimitedPlanRepository struct {
	inner ports.UserPlanRepository
}

// NewUnlimitedPlanRepository wraps inner.
func NewUnlimitedPlanRepository(inner ports.UserPlanRepository) *UnlimitedPlanRepository {
	return &UnlimitedPlanRepository{inner: inner}
}

func (r *UnlimitedPlanRepository) GetUserPlan(_ context.Context, userID string) (*entities.UserPlan, error) {
	return &entities.UserPlan{
		UserID: userID,
		Tier:   entities.PlanEnterprise,
		Limits: entities.DefaultPlanLimits(entities.PlanEnterprise),
	}, nil
}

func (r *UnlimitedPlanRepository) SetUserPlan(ctx context.Context, userID string, tier entities.PlanTier) (*entities.UserPlan, error) {
	return r.inner.SetUserPlan(ctx, userID, tier)
}

func (r *UnlimitedPlanRepository) ListUsersByPlan(ctx context.Context, tier entities.PlanTier) ([]entities.UserPlan, error) {
	return r.inner.ListUsersByPlan(ctx, tier)
}

func (r *UnlimitedPlanRepository) ListAllPlans(ctx context.Context) ([]entities.UserPlan, error) {
	return r.inner.ListAllPlans(ctx)
}

var _ ports.UserPlanRepository = (*UnlimitedPlanRepository)(nil)
