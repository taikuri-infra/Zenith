package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// EnvironmentHandler handles environment endpoints.
type EnvironmentHandler struct {
	envRepo     ports.EnvironmentRepository
	projectRepo ports.ProjectRepository
}

// NewEnvironmentHandler creates a new EnvironmentHandler.
func NewEnvironmentHandler(envRepo ports.EnvironmentRepository, projectRepo ports.ProjectRepository) *EnvironmentHandler {
	return &EnvironmentHandler{envRepo: envRepo, projectRepo: projectRepo}
}

// List handles GET /projects/:projectId/environments
func (h *EnvironmentHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	projectID := c.Params("projectId")
	project, err := h.projectRepo.GetProject(c.Context(), projectID)
	if err != nil {
		return NewNotFound("project not found")
	}
	if project.UserID != userID {
		return NewForbidden("not your project")
	}

	envs, err := h.envRepo.ListEnvironmentsByProject(c.Context(), projectID)
	if err != nil {
		return NewInternal("failed to list environments")
	}

	return c.JSON(fiber.Map{"environments": envs})
}

// Get handles GET /projects/:projectId/environments/:envId
func (h *EnvironmentHandler) Get(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	projectID := c.Params("projectId")
	project, err := h.projectRepo.GetProject(c.Context(), projectID)
	if err != nil {
		return NewNotFound("project not found")
	}
	if project.UserID != userID {
		return NewForbidden("not your project")
	}

	envID := c.Params("envId")
	env, err := h.envRepo.GetEnvironment(c.Context(), envID)
	if err != nil {
		return NewNotFound("environment not found")
	}
	if env.ProjectID != projectID {
		return NewForbidden("environment does not belong to this project")
	}

	return c.JSON(env)
}

// CreateEnvironmentsForProject creates production (and optionally staging) environments for a project.
// Called during project creation. Returns the created environments.
func (h *EnvironmentHandler) CreateEnvironmentsForProject(c *fiber.Ctx, projectID string, includeStaging bool) ([]entities.Environment, error) {
	prodEnv := &entities.Environment{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Name:      entities.EnvironmentProduction,
		Slug:      "prod",
		Status:    entities.EnvironmentStatusActive,
		IsDefault: true,
	}
	if err := h.envRepo.CreateEnvironment(c.Context(), prodEnv); err != nil {
		return nil, err
	}

	envs := []entities.Environment{*prodEnv}

	if includeStaging {
		stagingEnv := &entities.Environment{
			ID:        uuid.New().String(),
			ProjectID: projectID,
			Name:      entities.EnvironmentStaging,
			Slug:      "staging",
			Status:    entities.EnvironmentStatusActive,
			IsDefault: false,
		}
		if err := h.envRepo.CreateEnvironment(c.Context(), stagingEnv); err != nil {
			return envs, err
		}
		envs = append(envs, *stagingEnv)
	}

	return envs, nil
}
