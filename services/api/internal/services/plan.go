package services

import (
	"context"
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// PlanService handles user plan business logic.
type PlanService struct {
	planRepo      ports.UserPlanRepository
	appRepo       ports.AppRepository
	dbRepo        ports.DatabaseRepository
	storageRepo   ports.StorageRepository
	authRepo      ports.AppAuthRepository
	stripeEnabled bool
}

// NewPlanService creates a new PlanService.
func NewPlanService(
	planRepo ports.UserPlanRepository,
	appRepo ports.AppRepository,
	dbRepo ports.DatabaseRepository,
	storageRepo ports.StorageRepository,
	authRepo ports.AppAuthRepository,
) *PlanService {
	return &PlanService{
		planRepo:    planRepo,
		appRepo:     appRepo,
		dbRepo:      dbRepo,
		storageRepo: storageRepo,
		authRepo:    authRepo,
	}
}

// SetStripeEnabled marks whether Stripe billing is active.
func (s *PlanService) SetStripeEnabled(enabled bool) {
	s.stripeEnabled = enabled
}

// GetUserPlan returns the current user's plan and usage.
func (s *PlanService) GetUserPlan(ctx context.Context, userID string) (*dto.UserPlanResponse, error) {
	plan, err := s.planRepo.GetUserPlan(ctx, userID)
	if err != nil {
		return nil, err
	}

	usage := s.CalculateUsage(ctx, userID)

	return &dto.UserPlanResponse{
		Tier:   plan.Tier,
		Limits: plan.Limits,
		Usage:  usage,
	}, nil
}

// UpgradePlan changes the user's plan tier.
// When Stripe is enabled, paid tiers must go through billing checkout.
func (s *PlanService) UpgradePlan(ctx context.Context, userID string, tier entities.PlanTier) (*dto.UserPlanResponse, error) {
	if s.stripeEnabled && tier != entities.PlanFree {
		return nil, fmt.Errorf("paid plan upgrades require payment; use POST /api/v1/billing/checkout instead")
	}

	plan, err := s.planRepo.SetUserPlan(ctx, userID, tier)
	if err != nil {
		return nil, err
	}

	usage := s.CalculateUsage(ctx, userID)

	return &dto.UserPlanResponse{
		Tier:   plan.Tier,
		Limits: plan.Limits,
		Usage:  usage,
	}, nil
}

// CalculateUsage returns current resource usage for a user.
func (s *PlanService) CalculateUsage(ctx context.Context, userID string) dto.PlanUsage {
	appCount, _ := s.appRepo.CountAppsByUser(ctx, userID)
	dbCount, _ := s.dbRepo.CountDatabasesByUser(ctx, userID)
	bucketCount, _ := s.storageRepo.CountBucketsByUser(ctx, userID)

	return dto.PlanUsage{
		Apps:      appCount,
		Databases: dbCount,
		Buckets:   bucketCount,
	}
}

// CheckLimit verifies that the user hasn't exceeded their plan limit for a resource.
// Returns nil if under limit, error if at/over limit.
func (s *PlanService) CheckLimit(ctx context.Context, userID, resource string, currentCount int) error {
	plan, err := s.planRepo.GetUserPlan(ctx, userID)
	if err != nil {
		return nil // don't block on plan lookup failure
	}

	var limit int
	switch resource {
	case "apps":
		limit = plan.Limits.MaxApps
	case "databases":
		limit = plan.Limits.MaxDatabases
	case "buckets":
		limit = plan.Limits.MaxBuckets
	default:
		return nil
	}

	if currentCount >= limit {
		return fmt.Errorf("plan limit reached: %s. Upgrade your plan for more.", resource)
	}

	return nil
}
