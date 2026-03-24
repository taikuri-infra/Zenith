package handlers

import (
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// DeployTokenHandler handles deploy token CRUD endpoints.
type DeployTokenHandler struct {
	tokenRepo   ports.DeployTokenRepository
	projectRepo ports.ProjectRepository
}

// NewDeployTokenHandler creates a new DeployTokenHandler.
func NewDeployTokenHandler(tokenRepo ports.DeployTokenRepository, projectRepo ports.ProjectRepository) *DeployTokenHandler {
	return &DeployTokenHandler{tokenRepo: tokenRepo, projectRepo: projectRepo}
}

type createDeployTokenRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresIn string   `json:"expires_in"` // "30d", "90d", "180d", "365d"
}

// Create handles POST /projects/:projectId/deploy-tokens
func (h *DeployTokenHandler) Create(c *fiber.Ctx) error {
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

	var req createDeployTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if req.Name == "" {
		return NewBadRequest("name is required")
	}
	if len(req.Scopes) == 0 {
		return NewBadRequest("at least one scope is required")
	}

	// Validate scopes
	for _, s := range req.Scopes {
		if !entities.ValidDeployTokenScope(s) {
			return NewBadRequest("invalid scope: " + s)
		}
	}

	// Parse expiry duration
	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		duration, err := parseDuration(req.ExpiresIn)
		if err != nil {
			return NewBadRequest("invalid expires_in: use 30d, 90d, 180d, or 365d")
		}
		t := time.Now().Add(duration)
		expiresAt = &t
	} else {
		// Default: 90 days
		t := time.Now().Add(90 * 24 * time.Hour)
		expiresAt = &t
	}

	// Max expiry: 1 year
	maxExpiry := time.Now().Add(366 * 24 * time.Hour)
	if expiresAt.After(maxExpiry) {
		return NewBadRequest("expires_in cannot exceed 365 days")
	}

	token, err := h.tokenRepo.CreateDeployToken(c.Context(), userID, projectID, req.Name, req.Scopes, expiresAt)
	if err != nil {
		return NewInternal("failed to create deploy token")
	}

	return c.Status(fiber.StatusCreated).JSON(token)
}

// List handles GET /projects/:projectId/deploy-tokens
func (h *DeployTokenHandler) List(c *fiber.Ctx) error {
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

	tokens, err := h.tokenRepo.ListDeployTokensByProject(c.Context(), projectID)
	if err != nil {
		return NewInternal("failed to list deploy tokens")
	}

	return c.JSON(fiber.Map{"tokens": tokens})
}

// Revoke handles DELETE /projects/:projectId/deploy-tokens/:tokenId
func (h *DeployTokenHandler) Revoke(c *fiber.Ctx) error {
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

	tokenID := c.Params("tokenId")
	token, err := h.tokenRepo.GetDeployToken(c.Context(), tokenID)
	if err != nil {
		return NewNotFound("deploy token not found")
	}
	if token.ProjectID != projectID {
		return NewForbidden("token does not belong to this project")
	}

	if err := h.tokenRepo.RevokeDeployToken(c.Context(), tokenID); err != nil {
		return NewInternal("failed to revoke deploy token")
	}

	return c.JSON(fiber.Map{"message": "token revoked"})
}

// Rotate handles POST /projects/:projectId/deploy-tokens/:tokenId/rotate
func (h *DeployTokenHandler) Rotate(c *fiber.Ctx) error {
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

	tokenID := c.Params("tokenId")
	token, err := h.tokenRepo.GetDeployToken(c.Context(), tokenID)
	if err != nil {
		return NewNotFound("deploy token not found")
	}
	if token.ProjectID != projectID {
		return NewForbidden("token does not belong to this project")
	}

	rotated, err := h.tokenRepo.RotateDeployToken(c.Context(), tokenID)
	if err != nil {
		return NewInternal("failed to rotate deploy token")
	}

	return c.JSON(rotated)
}

// parseDuration parses human-readable durations like "30d", "90d".
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration")
	}
	unit := s[len(s)-1]
	valStr := s[:len(s)-1]
	var val int
	for _, c := range valStr {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid duration")
		}
		val = val*10 + int(c-'0')
	}
	switch unit {
	case 'd':
		return time.Duration(val) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(val) * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported unit: %c", unit)
	}
}

