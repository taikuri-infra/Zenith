package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// PlanHandler manages user plan operations.
type PlanHandler struct {
	svc *services.PlanService
}

// NewPlanHandler creates a new PlanHandler.
func NewPlanHandler(svc *services.PlanService) *PlanHandler {
	return &PlanHandler{svc: svc}
}

// GetMyPlan returns the current user's plan and usage.
// GET /api/v1/plan
func (h *PlanHandler) GetMyPlan(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	resp, err := h.svc.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(resp)
}

// UpgradePlan changes the user's plan tier.
// POST /api/v1/plan/upgrade
func (h *PlanHandler) UpgradePlan(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var input struct {
		Tier entities.PlanTier `json:"tier"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	resp, err := h.svc.UpgradePlan(c.Context(), userID, input.Tier)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(resp)
}

// CheckLimit is a middleware factory that checks plan limits before resource creation.
// UnlimitedMode disables plan-limit enforcement. It is set true in standalone
// self-host mode, where the customer owns the hardware and there is no billing
// or "upgrade" — SaaS plan tiers don't apply.
var UnlimitedMode bool

func CheckLimit(planRepo ports.UserPlanRepository, resource string, countFn func(c *fiber.Ctx, userID string) (int, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if UnlimitedMode {
			return c.Next()
		}
		userID, _ := c.Locals("user_id").(string)
		if userID == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}

		plan, err := planRepo.GetUserPlan(c.Context(), userID)
		if err != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "unable to verify plan limits")
		}

		count, err := countFn(c, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "unable to verify resource usage")
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
			return c.Next()
		}

		if count >= limit {
			return fiber.NewError(fiber.StatusForbidden,
				"plan limit reached: "+resource+". Upgrade your plan for more.")
		}

		return c.Next()
	}
}
