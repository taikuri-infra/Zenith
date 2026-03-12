package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// AuthPoolHandler manages auth pool HTTP endpoints.
type AuthPoolHandler struct {
	poolSvc  *services.AuthPoolService
	poolRepo ports.AuthPoolRepository
}

// NewAuthPoolHandler creates a new AuthPoolHandler.
func NewAuthPoolHandler(poolSvc *services.AuthPoolService, poolRepo ports.AuthPoolRepository) *AuthPoolHandler {
	return &AuthPoolHandler{poolSvc: poolSvc, poolRepo: poolRepo}
}

// CreatePool creates a new auth pool.
// POST /api/v1/auth-pools
func (h *AuthPoolHandler) CreatePool(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var input dto.CreateAuthPoolInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	pool, err := h.poolSvc.CreatePool(c.Context(), userID, input.ProjectID, input.Name)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(toAuthPoolInfo(pool, true))
}

// ListPools returns all pools for the current user.
// GET /api/v1/auth-pools
func (h *AuthPoolHandler) ListPools(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	pools, err := h.poolSvc.ListPools(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([]dto.AuthPoolInfo, len(pools))
	for i, p := range pools {
		result[i] = toAuthPoolInfo(&p, false)
	}
	return c.JSON(result)
}

// GetPool returns a single pool.
// GET /api/v1/auth-pools/:poolId
func (h *AuthPoolHandler) GetPool(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	return c.JSON(toAuthPoolInfo(pool, false))
}

// DeletePool deletes a pool and its Keycloak realm.
// DELETE /api/v1/auth-pools/:poolId
func (h *AuthPoolHandler) DeletePool(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	if err := h.poolSvc.DeletePool(c.Context(), pool); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "auth pool deleted"})
}

// CreateUser creates a user in a pool.
// POST /api/v1/auth-pools/:poolId/users
func (h *AuthPoolHandler) CreateUser(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	var input dto.CreateAuthPoolUserInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" || input.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}

	user, err := h.poolSvc.CreateUser(c.Context(), pool, input.Email, input.Password, input.FirstName, input.LastName)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

// ListUsers returns users in a pool (paginated).
// GET /api/v1/auth-pools/:poolId/users
func (h *AuthPoolHandler) ListUsers(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	first, _ := strconv.Atoi(c.Query("offset", "0"))
	max, _ := strconv.Atoi(c.Query("limit", "20"))
	if max <= 0 || max > 100 {
		max = 20
	}

	users, total, err := h.poolSvc.ListUsers(c.Context(), pool, first, max)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"users": users,
		"total": total,
	})
}

// GetUser returns a single user from a pool.
// GET /api/v1/auth-pools/:poolId/users/:userId
func (h *AuthPoolHandler) GetUser(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	user, err := h.poolSvc.GetUser(c.Context(), pool, c.Params("userId"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	return c.JSON(user)
}

// DeleteUser removes a user from a pool.
// DELETE /api/v1/auth-pools/:poolId/users/:userId
func (h *AuthPoolHandler) DeleteUser(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	if err := h.poolSvc.DeleteUser(c.Context(), pool, c.Params("userId")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "user deleted"})
}

// DisableUser disables a user in a pool.
// POST /api/v1/auth-pools/:poolId/users/:userId/disable
func (h *AuthPoolHandler) DisableUser(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	if err := h.poolSvc.DisableUser(c.Context(), pool, c.Params("userId")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "user disabled"})
}

// EnableUser enables a user in a pool.
// POST /api/v1/auth-pools/:poolId/users/:userId/enable
func (h *AuthPoolHandler) EnableUser(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	if err := h.poolSvc.EnableUser(c.Context(), pool, c.Params("userId")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "user enabled"})
}

// TokenExchange proxies token requests to Keycloak for a pool's realm.
// POST /api/v1/auth-pools/:poolId/token (PUBLIC — no JWT required)
func (h *AuthPoolHandler) TokenExchange(c *fiber.Ctx) error {
	poolID := c.Params("poolId")

	pool, err := h.poolRepo.GetPool(c.Context(), poolID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "auth pool not found")
	}
	if pool.Status != entities.AuthPoolStatusActive {
		return fiber.NewError(fiber.StatusServiceUnavailable, "auth pool is not active")
	}

	var input dto.TokenExchangeInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Default grant type
	if input.GrantType == "" {
		input.GrantType = "password"
	}

	tokenURL := pool.IssuerURL + "/protocol/openid-connect/token"

	form := url.Values{}
	form.Set("client_id", pool.ClientID)
	form.Set("client_secret", pool.ClientSecret)
	form.Set("grant_type", input.GrantType)

	switch input.GrantType {
	case "password":
		if input.Username == "" || input.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "username and password are required")
		}
		form.Set("username", input.Username)
		form.Set("password", input.Password)
		if input.Scope != "" {
			form.Set("scope", input.Scope)
		} else {
			form.Set("scope", "openid")
		}
	case "refresh_token":
		if input.RefreshToken == "" {
			return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
		}
		form.Set("refresh_token", input.RefreshToken)
	default:
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unsupported grant_type: %s", input.GrantType))
	}

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to reach identity provider")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to read identity provider response")
	}

	// Forward Keycloak response as-is (includes error details on failure)
	c.Set("Content-Type", "application/json")
	return c.Status(resp.StatusCode).Send(body)
}

// requirePool loads the pool and verifies ownership.
func (h *AuthPoolHandler) requirePool(c *fiber.Ctx) (*entities.AuthPool, error) {
	userID, _ := c.Locals("user_id").(string)
	poolID := c.Params("poolId")

	pool, err := h.poolRepo.GetPool(c.Context(), poolID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "auth pool not found")
	}
	if pool.UserID != userID {
		return nil, fiber.NewError(fiber.StatusForbidden, "not your auth pool")
	}

	return pool, nil
}

func toAuthPoolInfo(p *entities.AuthPool, includeSecret bool) dto.AuthPoolInfo {
	info := dto.AuthPoolInfo{
		ID:        p.ID,
		Name:      p.Name,
		ProjectID: p.ProjectID,
		Status:    p.Status,
		IssuerURL: p.IssuerURL,
		ClientID:  p.ClientID,
		UserCount: p.UserCount,
		MaxUsers:  p.MaxUsers,
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if includeSecret {
		info.ClientSecret = p.ClientSecret
	}
	return info
}
