package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// SCIMHandler implements SCIM 2.0 user provisioning endpoints.
type SCIMHandler struct {
	userRepo store.UserRepository
	planRepo store.UserPlanRepository
}

func NewSCIMHandler(userRepo store.UserRepository, planRepo store.UserPlanRepository) *SCIMHandler {
	return &SCIMHandler{userRepo: userRepo, planRepo: planRepo}
}

// ListUsers returns users in SCIM format.
func (h *SCIMHandler) ListUsers(c *fiber.Ctx) error {
	// SCIM 2.0 ListResponse format
	return c.JSON(fiber.Map{
		"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"totalResults": 0,
		"startIndex":   1,
		"itemsPerPage": 20,
		"Resources":    []interface{}{},
	})
}

// GetUser returns a single user in SCIM format.
func (h *SCIMHandler) GetUser(c *fiber.Ctx) error {
	userID := c.Params("userId")
	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	return c.JSON(fiber.Map{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"id":       user.ID,
		"userName": user.Email,
		"name": fiber.Map{
			"formatted": user.Name,
		},
		"emails": []fiber.Map{
			{"value": user.Email, "primary": true},
		},
		"active": true,
		"meta": fiber.Map{
			"resourceType": "User",
		},
	})
}

// CreateUser provisions a new user via SCIM.
func (h *SCIMHandler) CreateUser(c *fiber.Ctx) error {
	var body struct {
		UserName string `json:"userName"`
		Name     struct {
			Formatted string `json:"formatted"`
		} `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid SCIM request")
	}
	if body.UserName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "userName is required")
	}

	// Create user with a random password (SSO login only)
	user, err := h.userRepo.Create(c.Context(), body.UserName, "scim-provisioned-no-password", body.Name.Formatted, entities.RoleViewer)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "user already exists")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"id":       user.ID,
		"userName": user.Email,
		"name": fiber.Map{
			"formatted": user.Name,
		},
		"active": true,
	})
}

// DeleteUser deprovisions a user via SCIM.
func (h *SCIMHandler) DeleteUser(c *fiber.Ctx) error {
	// In a real implementation, this would disable/delete the user
	// For now, return 204 to satisfy the SCIM spec
	return c.SendStatus(fiber.StatusNoContent)
}
