package store

import (
	"context"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// MemoryUserPlanRepository is an in-memory implementation of UserPlanRepository.
type MemoryUserPlanRepository struct {
	mu    sync.RWMutex
	plans map[string]*entities.UserPlan // userID -> plan
}

// NewMemoryUserPlanRepository creates a new MemoryUserPlanRepository.
func NewMemoryUserPlanRepository() *MemoryUserPlanRepository {
	return &MemoryUserPlanRepository{
		plans: make(map[string]*entities.UserPlan),
	}
}

func (r *MemoryUserPlanRepository) GetUserPlan(_ context.Context, userID string) (*entities.UserPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plan, ok := r.plans[userID]
	if !ok {
		// Default to free plan
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
	return plan, nil
}

func (r *MemoryUserPlanRepository) SetUserPlan(_ context.Context, userID string, tier entities.PlanTier) (*entities.UserPlan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	limits := entities.DefaultPlanLimits(tier)
	plan := &entities.UserPlan{
		UserID: userID,
		Tier:   tier,
		Limits: limits,
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	if existing, ok := r.plans[userID]; ok {
		plan.CreatedAt = existing.CreatedAt
	}

	r.plans[userID] = plan
	return plan, nil
}

func (r *MemoryUserPlanRepository) ListUsersByPlan(_ context.Context, tier entities.PlanTier) ([]entities.UserPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.UserPlan
	for _, p := range r.plans {
		if p.Tier == tier {
			result = append(result, *p)
		}
	}
	return result, nil
}
