package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

type SSOHandler struct {
	ssoRepo  store.SSORepository
	planRepo store.UserPlanRepository
}

func NewSSOHandler(ssoRepo store.SSORepository, planRepo store.UserPlanRepository) *SSOHandler {
	return &SSOHandler{ssoRepo: ssoRepo, planRepo: planRepo}
}

func (h *SSOHandler) ConfigureSAML(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier != entities.PlanTeam && plan.Tier != entities.PlanEnterprise {
		return fiber.NewError(fiber.StatusForbidden, "SSO requires Team plan or higher")
	}

	var body struct {
		EntityID    string `json:"entity_id"`
		SSOURL      string `json:"sso_url"`
		Certificate string `json:"certificate"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.EntityID == "" || body.SSOURL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "entity_id and sso_url are required")
	}

	config := &entities.SSOConfig{
		EntityID:    body.EntityID,
		SSOURL:      body.SSOURL,
		Certificate: body.Certificate,
	}
	result, err := h.ssoRepo.CreateConfig(c.Context(), userID, entities.SSOProviderSAML, config)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *SSOHandler) ConfigureOIDC(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier != entities.PlanTeam && plan.Tier != entities.PlanEnterprise {
		return fiber.NewError(fiber.StatusForbidden, "SSO requires Team plan or higher")
	}

	var body struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		DiscoveryURL string `json:"discovery_url"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.ClientID == "" || body.DiscoveryURL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "client_id and discovery_url are required")
	}

	config := &entities.SSOConfig{
		ClientID:     body.ClientID,
		ClientSecret: body.ClientSecret,
		DiscoveryURL: body.DiscoveryURL,
	}
	result, err := h.ssoRepo.CreateConfig(c.Context(), userID, entities.SSOProviderOIDC, config)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *SSOHandler) ListConfigs(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	configs, err := h.ssoRepo.ListConfigsByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if configs == nil {
		configs = []entities.SSOConfig{}
	}
	return c.JSON(fiber.Map{"items": configs})
}

func (h *SSOHandler) DeleteConfig(c *fiber.Ctx) error {
	configID := c.Params("configId")
	if err := h.ssoRepo.DeleteConfig(c.Context(), configID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "SSO config not found")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
