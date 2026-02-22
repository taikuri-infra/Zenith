package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// PlanHandler manages user plan operations.
type PlanHandler struct {
	planRepo       store.UserPlanRepository
	appRepo        store.AppRepository
	dbRepo         store.DatabaseRepository
	storageRepo    store.StorageRepository
	authRepo       store.AppAuthRepository
	stripeEnabled  bool
}

// NewPlanHandler creates a new PlanHandler.
func NewPlanHandler(
	planRepo store.UserPlanRepository,
	appRepo store.AppRepository,
	dbRepo store.DatabaseRepository,
	storageRepo store.StorageRepository,
	authRepo store.AppAuthRepository,
) *PlanHandler {
	return &PlanHandler{
		planRepo:    planRepo,
		appRepo:     appRepo,
		dbRepo:      dbRepo,
		storageRepo: storageRepo,
		authRepo:    authRepo,
	}
}

// SetStripeEnabled marks whether Stripe billing is active.
// When enabled, paid tier upgrades are rejected here and must go through /billing/checkout.
func (h *PlanHandler) SetStripeEnabled(enabled bool) {
	h.stripeEnabled = enabled
}

// GetMyPlan returns the current user's plan and usage.
// GET /api/v1/plan
func (h *PlanHandler) GetMyPlan(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	usage, err := h.calculateUsage(c, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(dto.UserPlanResponse{
		Tier:   plan.Tier,
		Limits: plan.Limits,
		Usage:  *usage,
	})
}

// UpgradePlan changes the user's plan tier.
// POST /api/v1/plan/upgrade
//
// When Stripe billing is enabled, paid tiers (pro, team, enterprise) are rejected
// with a message to use /billing/checkout instead. Free downgrades still work.
func (h *PlanHandler) UpgradePlan(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var input struct {
		Tier entities.PlanTier `json:"tier"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// When Stripe is enabled, paid tiers must go through /billing/checkout
	if h.stripeEnabled && input.Tier != entities.PlanFree {
		return fiber.NewError(fiber.StatusBadRequest,
			"paid plan upgrades require payment; use POST /api/v1/billing/checkout instead")
	}

	plan, err := h.planRepo.SetUserPlan(c.Context(), userID, input.Tier)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	usage, err := h.calculateUsage(c, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(dto.UserPlanResponse{
		Tier:   plan.Tier,
		Limits: plan.Limits,
		Usage:  *usage,
	})
}

func (h *PlanHandler) calculateUsage(c *fiber.Ctx, userID string) (*dto.PlanUsage, error) {
	appCount, _ := h.appRepo.CountAppsByUser(c.Context(), userID)
	dbCount, _ := h.dbRepo.CountDatabasesByUser(c.Context(), userID)
	bucketCount, _ := h.storageRepo.CountBucketsByUser(c.Context(), userID)

	return &dto.PlanUsage{
		Apps:      appCount,
		Databases: dbCount,
		Buckets:   bucketCount,
	}, nil
}

// CheckLimit is a middleware factory that checks plan limits before resource creation.
func CheckLimit(planRepo store.UserPlanRepository, resource string, countFn func(c *fiber.Ctx, userID string) (int, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(string)
		if userID == "" {
			return c.Next()
		}

		plan, err := planRepo.GetUserPlan(c.Context(), userID)
		if err != nil {
			return c.Next() // don't block on plan lookup failure
		}

		count, err := countFn(c, userID)
		if err != nil {
			return c.Next()
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
			return c.Next()
		}

		if count >= limit {
			return fiber.NewError(fiber.StatusForbidden,
				"plan limit reached: "+resource+". Upgrade your plan for more.")
		}

		return c.Next()
	}
}
