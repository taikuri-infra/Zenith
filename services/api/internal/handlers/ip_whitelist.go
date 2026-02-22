package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

type IPWhitelistHandler struct {
	ipRepo   ports.IPWhitelistRepository
	planRepo ports.UserPlanRepository
}

func NewIPWhitelistHandler(ipRepo ports.IPWhitelistRepository, planRepo ports.UserPlanRepository) *IPWhitelistHandler {
	return &IPWhitelistHandler{ipRepo: ipRepo, planRepo: planRepo}
}

func (h *IPWhitelistHandler) Add(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier != entities.PlanEnterprise {
		return fiber.NewError(fiber.StatusForbidden, "IP whitelisting requires Enterprise plan")
	}

	var body struct {
		CIDR        string `json:"cidr"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.CIDR == "" {
		return fiber.NewError(fiber.StatusBadRequest, "cidr is required")
	}

	entry, err := h.ipRepo.AddEntry(c.Context(), userID, body.CIDR, body.Description)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(entry)
}

func (h *IPWhitelistHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	entries, err := h.ipRepo.ListByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if entries == nil {
		entries = []entities.IPWhitelistEntry{}
	}
	return c.JSON(fiber.Map{"items": entries})
}

func (h *IPWhitelistHandler) Delete(c *fiber.Ctx) error {
	entryID := c.Params("entryId")
	if err := h.ipRepo.DeleteEntry(c.Context(), entryID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "entry not found")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
