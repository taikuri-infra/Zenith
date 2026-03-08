package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// AddOnHandler manages the add-on marketplace.
type AddOnHandler struct {
	planRepo ports.UserPlanRepository
}

// NewAddOnHandler creates a new AddOnHandler.
func NewAddOnHandler(planRepo ports.UserPlanRepository) *AddOnHandler {
	return &AddOnHandler{planRepo: planRepo}
}

// ListCatalog returns all available add-ons, filtered by the user's plan tier.
// GET /api/v1/addons
func (h *AddOnHandler) ListCatalog(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		plan = &entities.UserPlan{Tier: entities.PlanFree}
	}

	allAddons := entities.AvailableAddOns()
	tierRank := tierToRank(plan.Tier)

	type addOnResponse struct {
		entities.AddOn
		Available bool `json:"available"`
	}

	result := make([]addOnResponse, 0, len(allAddons))
	for _, addon := range allAddons {
		result = append(result, addOnResponse{
			AddOn:     addon,
			Available: tierRank >= tierToRank(addon.MinTier),
		})
	}

	return c.JSON(result)
}

// GetAddOn returns a single add-on by ID.
// GET /api/v1/addons/:addonId
func (h *AddOnHandler) GetAddOn(c *fiber.Ctx) error {
	addonID := c.Params("addonId")

	for _, addon := range entities.AvailableAddOns() {
		if addon.ID == addonID {
			return c.JSON(addon)
		}
	}

	return fiber.NewError(fiber.StatusNotFound, "add-on not found")
}

func tierToRank(tier entities.PlanTier) int {
	switch tier {
	case entities.PlanFree:
		return 0
	case entities.PlanPro:
		return 1
	case entities.PlanTeam:
		return 2
	case entities.PlanBusiness:
		return 3
	case entities.PlanEnterprise:
		return 4
	default:
		return 0
	}
}
