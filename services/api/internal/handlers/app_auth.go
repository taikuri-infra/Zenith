package handlers

import (
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	appAuthAccessExpiry  = 1 * time.Hour
	appAuthRefreshExpiry = 7 * 24 * time.Hour
)

// AppAuthHandler manages per-app authentication (Phase 3 built-in auth).
type AppAuthHandler struct {
	authRepo ports.AppAuthRepository
	appRepo  ports.AppRepository
}

// NewAppAuthHandler creates a new AppAuthHandler.
func NewAppAuthHandler(authRepo ports.AppAuthRepository, appRepo ports.AppRepository) *AppAuthHandler {
	return &AppAuthHandler{authRepo: authRepo, appRepo: appRepo}
}

// Enable turns on built-in auth for an app.
// POST /api/v1/apps/:appId/auth/enable
func (h *AppAuthHandler) Enable(c *fiber.Ctx) error {
	appID := c.Params("appId")
	userID, _ := c.Locals("user_id").(string)

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}
	if app.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your app")
	}

	// Default free tier: 1000 users
	maxUsers := 1000
	cfg, err := h.authRepo.EnableAuth(c.Context(), appID, maxUsers)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	count, _ := h.authRepo.CountAppUsers(c.Context(), appID)
	return c.JSON(dto.AppAuthConfigResponse{
		Enabled:   cfg.Enabled,
		UserCount: count,
		MaxUsers:  cfg.MaxUsers,
	})
}

// Disable turns off built-in auth for an app.
// POST /api/v1/apps/:appId/auth/disable
func (h *AppAuthHandler) Disable(c *fiber.Ctx) error {
	appID := c.Params("appId")
	userID, _ := c.Locals("user_id").(string)

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}
	if app.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your app")
	}

	if err := h.authRepo.DisableAuth(c.Context(), appID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "auth disabled"})
}

// Status returns the auth configuration for an app.
// GET /api/v1/apps/:appId/auth
func (h *AppAuthHandler) Status(c *fiber.Ctx) error {
	appID := c.Params("appId")

	cfg, err := h.authRepo.GetAuthConfig(c.Context(), appID)
	if err != nil {
		// Auth not configured → return disabled
		return c.JSON(dto.AppAuthConfigResponse{
			Enabled:   false,
			UserCount: 0,
			MaxUsers:  0,
		})
	}

	count, _ := h.authRepo.CountAppUsers(c.Context(), appID)
	return c.JSON(dto.AppAuthConfigResponse{
		Enabled:   cfg.Enabled,
		UserCount: count,
		MaxUsers:  cfg.MaxUsers,
	})
}

// Signup registers a new end-user in an app's auth system.
// POST /api/v1/apps/:appId/auth/signup (public, no JWT required)
func (h *AppAuthHandler) Signup(c *fiber.Ctx) error {
	appID := c.Params("appId")

	cfg, err := h.authRepo.GetAuthConfig(c.Context(), appID)
	if err != nil || !cfg.Enabled {
		return fiber.NewError(fiber.StatusBadRequest, "auth not enabled for this app")
	}

	var input dto.AppAuthSignupInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" || input.Password == "" || input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email, password, and name are required")
	}
	if len(input.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	user, err := h.authRepo.CreateAppUser(c.Context(), appID, input.Email, input.Password, input.Name)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	return h.issueAppTokens(c, cfg.JWTSecret, appID, user)
}

// Login authenticates an end-user in an app's auth system.
// POST /api/v1/apps/:appId/auth/login (public, no JWT required)
func (h *AppAuthHandler) Login(c *fiber.Ctx) error {
	appID := c.Params("appId")

	cfg, err := h.authRepo.GetAuthConfig(c.Context(), appID)
	if err != nil || !cfg.Enabled {
		return fiber.NewError(fiber.StatusBadRequest, "auth not enabled for this app")
	}

	var input dto.AppAuthLoginInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" || input.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}

	user, hash, err := h.authRepo.GetAppUserByEmail(c.Context(), appID, input.Email)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}

	return h.issueAppTokens(c, cfg.JWTSecret, appID, user)
}

// ListUsers returns all users registered via the app's auth.
// GET /api/v1/apps/:appId/auth/users
func (h *AppAuthHandler) ListUsers(c *fiber.Ctx) error {
	appID := c.Params("appId")
	userID, _ := c.Locals("user_id").(string)

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}
	if app.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your app")
	}

	users, err := h.authRepo.ListAppUsers(c.Context(), appID, 100, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	count, _ := h.authRepo.CountAppUsers(c.Context(), appID)
	result := make([]dto.AppAuthUserResponse, len(users))
	for i, u := range users {
		result[i] = dto.AppAuthUserResponse{
			ID:        u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Verified:  u.Verified,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return c.JSON(fiber.Map{
		"users": result,
		"total": count,
	})
}

// DeleteUser removes an end-user from the app's auth system.
// DELETE /api/v1/apps/:appId/auth/users/:userId
func (h *AppAuthHandler) DeleteUser(c *fiber.Ctx) error {
	appID := c.Params("appId")
	targetUserID := c.Params("userId")
	ownerID, _ := c.Locals("user_id").(string)

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}
	if app.UserID != ownerID {
		return fiber.NewError(fiber.StatusForbidden, "not your app")
	}

	if err := h.authRepo.DeleteAppUser(c.Context(), appID, targetUserID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(fiber.Map{"message": "user deleted"})
}

// appAuthClaims are JWT claims for app end-users.
type appAuthClaims struct {
	jwt.RegisteredClaims
	AppID string `json:"app_id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *AppAuthHandler) issueAppTokens(c *fiber.Ctx, secret, appID string, user *entities.AppUser) error {
	accessClaims := appAuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    "zenith-app-auth",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(appAuthAccessExpiry)),
		},
		AppID: appID,
		Email: user.Email,
		Name:  user.Name,
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(secret))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate token")
	}

	refreshClaims := appAuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    "zenith-app-auth",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(appAuthRefreshExpiry)),
		},
		AppID: appID,
		Email: user.Email,
		Name:  user.Name,
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(secret))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(dto.AppAuthTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "bearer",
		ExpiresIn:    int(appAuthAccessExpiry.Seconds()),
	})
}
