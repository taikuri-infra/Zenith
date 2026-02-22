package handlers

import (
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

type PreviewHandler struct {
	previewRepo ports.PreviewRepository
	appRepo     ports.AppRepository
	planRepo    ports.UserPlanRepository
}

func NewPreviewHandler(previewRepo ports.PreviewRepository, appRepo ports.AppRepository, planRepo ports.UserPlanRepository) *PreviewHandler {
	return &PreviewHandler{previewRepo: previewRepo, appRepo: appRepo, planRepo: planRepo}
}

func (h *PreviewHandler) Create(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier != entities.PlanTeam && plan.Tier != entities.PlanEnterprise {
		return fiber.NewError(fiber.StatusForbidden, "preview deployments require Team plan or higher")
	}

	var body struct {
		PRNumber int    `json:"pr_number"`
		Branch   string `json:"branch"`
		GitSHA   string `json:"git_sha"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.PRNumber == 0 || body.Branch == "" {
		return fiber.NewError(fiber.StatusBadRequest, "pr_number and branch are required")
	}

	// Generate preview URL
	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}
	previewURL := fmt.Sprintf("https://%s-pr-%d.freezenith.com", app.Name, body.PRNumber)

	preview, err := h.previewRepo.CreatePreview(c.Context(), appID, body.PRNumber, body.Branch, body.GitSHA, previewURL)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(preview)
}

func (h *PreviewHandler) List(c *fiber.Ctx) error {
	appID := c.Params("appId")
	previews, err := h.previewRepo.ListPreviewsByApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if previews == nil {
		previews = []entities.PreviewDeployment{}
	}
	return c.JSON(fiber.Map{"items": previews})
}

func (h *PreviewHandler) Delete(c *fiber.Ctx) error {
	previewID := c.Params("previewId")
	if err := h.previewRepo.DeletePreview(c.Context(), previewID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "preview not found")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
