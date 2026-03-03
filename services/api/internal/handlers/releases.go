package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/services/deploy"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// ReleaseHandler manages app releases (versioned images from zenith-actions).
type ReleaseHandler struct {
	appRepo  ports.AppRepository
	pipeline *deploy.Pipeline
}

func NewReleaseHandler(appRepo ports.AppRepository, pipeline *deploy.Pipeline) *ReleaseHandler {
	return &ReleaseHandler{appRepo: appRepo, pipeline: pipeline}
}

// CreateRelease POST /api/v1/apps/:appId/releases
// Called by zenith-actions after pushing a new image.
func (h *ReleaseHandler) CreateRelease(c *fiber.Ctx) error {
	appID := c.Params("appId")

	var input dto.CreateReleaseInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Image == "" {
		return fiber.NewError(fiber.StatusBadRequest, "image is required")
	}
	if input.Branch == "" {
		input.Branch = "main"
	}

	release, err := h.appRepo.CreateRelease(c.Context(), appID, &input)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to register release")
	}

	return c.Status(fiber.StatusCreated).JSON(release)
}

// ListReleases GET /api/v1/apps/:appId/releases
func (h *ReleaseHandler) ListReleases(c *fiber.Ctx) error {
	appID := c.Params("appId")

	releases, err := h.appRepo.ListReleases(c.Context(), appID, 20)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list releases")
	}

	return c.JSON(fiber.Map{"releases": releases})
}

// DeployRelease POST /api/v1/apps/:appId/releases/:releaseId/deploy
// Triggers deployment of a specific release version.
func (h *ReleaseHandler) DeployRelease(c *fiber.Ctx) error {
	appID := c.Params("appId")
	releaseID := c.Params("releaseId")

	release, err := h.appRepo.GetRelease(c.Context(), releaseID)
	if err != nil || release.AppID != appID {
		return fiber.NewError(fiber.StatusNotFound, "release not found")
	}

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}

	// Create a deployment record for this release
	deployment, err := h.appRepo.CreateDeployment(c.Context(), appID, release.GitSHA)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create deployment")
	}

	// Trigger async pipeline with the pre-built image
	go h.pipeline.TriggerImageDeploy(app, deployment, release.Image)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"deployment_id": deployment.ID,
		"release_id":    releaseID,
		"image":         release.Image,
		"status":        "deploying",
	})
}
