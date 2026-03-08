package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

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
	gwRepo        ports.GatewayRepository
	authPoolRepo  ports.AuthPoolRepository
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

// SetGatewayRepo injects the gateway repository for usage tracking.
func (s *PlanService) SetGatewayRepo(repo ports.GatewayRepository) {
	s.gwRepo = repo
}

// SetAuthPoolRepo injects the auth pool repository for usage tracking.
func (s *PlanService) SetAuthPoolRepo(repo ports.AuthPoolRepository) {
	s.authPoolRepo = repo
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

	// Check if this is a downgrade
	oldPlan, _ := s.planRepo.GetUserPlan(ctx, userID)
	isDowngrade := oldPlan != nil && tierRank(tier) < tierRank(oldPlan.Tier)

	plan, err := s.planRepo.SetUserPlan(ctx, userID, tier)
	if err != nil {
		return nil, err
	}

	// Enforce resource limits on downgrade
	if isDowngrade {
		s.EnforceDowngrade(ctx, userID, tier)
	}

	usage := s.CalculateUsage(ctx, userID)

	return &dto.UserPlanResponse{
		Tier:   plan.Tier,
		Limits: plan.Limits,
		Usage:  usage,
	}, nil
}

// EnforceDowngrade suspends excess apps when a user downgrades to a lower tier.
// Storage buckets become inaccessible via CheckLimit (fail-closed), and data
// is preserved in S3 in case the user re-upgrades.
func (s *PlanService) EnforceDowngrade(ctx context.Context, userID string, newTier entities.PlanTier) {
	limits := entities.DefaultPlanLimits(newTier)

	apps, err := s.appRepo.ListAppsByUser(ctx, userID)
	if err != nil {
		slog.Error("EnforceDowngrade: failed to list apps", "user_id", userID, "error", err)
		return
	}

	if len(apps) > limits.MaxApps {
		// Sort by creation time descending (newest first) — keep the newest up to MaxApps
		sort.Slice(apps, func(i, j int) bool {
			return apps[i].CreatedAt.After(apps[j].CreatedAt)
		})

		for i := limits.MaxApps; i < len(apps); i++ {
			status := entities.AppStatusSuspended
			if _, err := s.appRepo.UpdateApp(ctx, apps[i].ID, &dto.UpdateAppInput{Status: &status}); err != nil {
				slog.Error("EnforceDowngrade: failed to suspend app", "app_id", apps[i].ID, "error", err)
			} else {
				slog.Info("EnforceDowngrade: suspended app", "app_id", apps[i].ID, "app_name", apps[i].Name, "user_id", userID)
			}
		}
	}
}

// tierRank returns numeric rank for plan comparison (higher = better plan).
func tierRank(t entities.PlanTier) int {
	switch t {
	case entities.PlanEnterprise:
		return 5
	case entities.PlanBusiness:
		return 4
	case entities.PlanTeam:
		return 3
	case entities.PlanPro:
		return 2
	default:
		return 1
	}
}

// CalculateUsage returns current resource usage for a user.
func (s *PlanService) CalculateUsage(ctx context.Context, userID string) dto.PlanUsage {
	appCount, _ := s.appRepo.CountAppsByUser(ctx, userID)
	dbCount, _ := s.dbRepo.CountDatabasesByUser(ctx, userID)
	bucketCount, _ := s.storageRepo.CountBucketsByUser(ctx, userID)

	usage := dto.PlanUsage{
		Apps:      appCount,
		Databases: dbCount,
		Buckets:   bucketCount,
	}

	if s.gwRepo != nil {
		usage.Gateways, _ = s.gwRepo.CountGatewaysByUser(ctx, userID)
		usage.GatewayRoutes, _ = s.gwRepo.CountRoutesByUser(ctx, userID)
	}

	if s.authPoolRepo != nil {
		usage.AuthPools, _ = s.authPoolRepo.CountPoolsByUser(ctx, userID)
	}

	return usage
}

// CheckLimit verifies that the user hasn't exceeded their plan limit for a resource.
// Returns nil if under limit, error if at/over limit.
func (s *PlanService) CheckLimit(ctx context.Context, userID, resource string, currentCount int) error {
	plan, err := s.planRepo.GetUserPlan(ctx, userID)
	if err != nil {
		return fmt.Errorf("unable to verify plan limits")
	}

	var limit int
	switch resource {
	case "apps":
		limit = plan.Limits.MaxApps
	case "databases":
		limit = plan.Limits.MaxDatabases
	case "buckets":
		limit = plan.Limits.MaxBuckets
	case "gateways":
		limit = plan.Limits.MaxGateways
	case "gateway_routes":
		limit = plan.Limits.MaxGatewayRoutes
	case "auth_pools":
		limit = plan.Limits.MaxAuthPools
	default:
		return nil
	}

	if currentCount >= limit {
		return fmt.Errorf("plan limit reached: %s. Upgrade your plan for more.", resource)
	}

	return nil
}
