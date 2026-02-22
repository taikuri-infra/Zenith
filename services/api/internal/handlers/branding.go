package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

type BrandingHandler struct {
	brandingRepo ports.BrandingRepository
	planRepo     ports.UserPlanRepository
}

func NewBrandingHandler(brandingRepo ports.BrandingRepository, planRepo ports.UserPlanRepository) *BrandingHandler {
	return &BrandingHandler{brandingRepo: brandingRepo, planRepo: planRepo}
}

// GetDPA returns the DPA status for the current user.
func (h *BrandingHandler) GetDPA(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	dpa, err := h.brandingRepo.GetDPA(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(dpa)
}

// SignDPA records the user's digital signature on the DPA.
func (h *BrandingHandler) SignDPA(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier == entities.PlanFree || plan.Tier == entities.PlanPro {
		return fiber.NewError(fiber.StatusForbidden, "DPA requires Team plan or higher")
	}

	var body struct {
		SignedBy string `json:"signed_by"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.SignedBy == "" {
		return fiber.NewError(fiber.StatusBadRequest, "signed_by is required")
	}

	dpa, err := h.brandingRepo.SignDPA(c.Context(), userID, body.SignedBy, c.IP())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(dpa)
}

// GetBranding returns the branding config for the current user.
func (h *BrandingHandler) GetBranding(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	branding, err := h.brandingRepo.GetBranding(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(branding)
}

// UpdateBranding updates the branding config.
func (h *BrandingHandler) UpdateBranding(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier != entities.PlanEnterprise {
		return fiber.NewError(fiber.StatusForbidden, "white-label branding requires Enterprise plan")
	}

	var body struct {
		CompanyName  *string `json:"company_name,omitempty"`
		LogoURL      *string `json:"logo_url,omitempty"`
		PrimaryColor *string `json:"primary_color,omitempty"`
		HideBranding *bool   `json:"hide_branding,omitempty"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	branding, err := h.brandingRepo.UpdateBranding(c.Context(), userID, body.CompanyName, body.LogoURL, body.PrimaryColor, body.HideBranding)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(branding)
}

// SetDashboardDomain sets a custom dashboard domain.
func (h *BrandingHandler) SetDashboardDomain(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier != entities.PlanEnterprise {
		return fiber.NewError(fiber.StatusForbidden, "custom dashboard domain requires Enterprise plan")
	}

	var body struct {
		Domain string `json:"domain"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.Domain == "" {
		return fiber.NewError(fiber.StatusBadRequest, "domain is required")
	}

	branding, err := h.brandingRepo.SetDashboardDomain(c.Context(), userID, body.Domain)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(branding)
}
