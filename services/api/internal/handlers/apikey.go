package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// APIKeyHandler manages API key operations.
type APIKeyHandler struct {
	keyRepo  ports.APIKeyRepository
	planRepo ports.UserPlanRepository
}

// NewAPIKeyHandler creates a new APIKeyHandler.
func NewAPIKeyHandler(keyRepo ports.APIKeyRepository, planRepo ports.UserPlanRepository) *APIKeyHandler {
	return &APIKeyHandler{keyRepo: keyRepo, planRepo: planRepo}
}

// Create generates a new API key.
// POST /api/v1/api-keys
func (h *APIKeyHandler) Create(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var input struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	// Check plan limit
	plan, _ := h.planRepo.GetUserPlan(c.Context(), userID)
	if plan != nil {
		count, _ := h.keyRepo.CountAPIKeysByUser(c.Context(), userID)
		var limit int
		switch plan.Tier {
		case "free":
			limit = 1
		case "pro":
			limit = 5
		case "team":
			limit = 20
		default:
			limit = 1000
		}
		if count >= limit {
			return fiber.NewError(fiber.StatusForbidden, "API key limit reached. Upgrade your plan for more.")
		}
	}

	key, err := h.keyRepo.CreateAPIKey(c.Context(), userID, input.Name, input.Scopes)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(key)
}

// List returns all API keys for the authenticated user.
// GET /api/v1/api-keys
func (h *APIKeyHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	keys, err := h.keyRepo.ListAPIKeysByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"items": keys, "total": len(keys)})
}

// Delete revokes an API key.
// DELETE /api/v1/api-keys/:keyId
func (h *APIKeyHandler) Delete(c *fiber.Ctx) error {
	keyID := c.Params("keyId")
	userID, _ := c.Locals("user_id").(string)

	key, err := h.keyRepo.GetAPIKey(c.Context(), keyID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "API key not found")
	}
	if key.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your API key")
	}

	if err := h.keyRepo.DeleteAPIKey(c.Context(), keyID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "API key revoked"})
}
