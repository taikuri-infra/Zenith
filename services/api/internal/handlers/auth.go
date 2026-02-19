package handlers

import (
	"time"

	"github.com/dotechhq/zenith/services/api/internal/middleware"
	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

const (
	accessTokenExpiry  = 1 * time.Hour
	refreshTokenExpiry = 7 * 24 * time.Hour
)

type AuthHandler struct {
	store     *store.UserStore
	jwtSecret string
}

func NewAuthHandler(userStore *store.UserStore, jwtSecret string) *AuthHandler {
	return &AuthHandler{store: userStore, jwtSecret: jwtSecret}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Login authenticates a user and returns JWT tokens.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}

	user, err := h.store.GetByEmail(req.Email)
	if err != nil || !h.store.CheckPassword(user, req.Password) {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}

	return h.issueTokens(c, &user.User)
}

// Register creates a new user and returns JWT tokens.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email, password, and name are required")
	}
	if len(req.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	// First user gets owner role, subsequent users get developer
	role := models.RoleDeveloper
	if h.store.Count() == 0 {
		role = models.RoleOwner
	}

	user, err := h.store.Create(req.Email, req.Password, req.Name, role)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	return h.issueTokens(c, user)
}

// Refresh exchanges a refresh token for new access + refresh tokens.
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req refreshRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
	}

	// Validate the refresh token (it's a JWT too)
	claims, err := middleware.ParseToken(h.jwtSecret, req.RefreshToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired refresh token")
	}

	user, err := h.store.GetByID(claims.Subject)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}

	return h.issueTokens(c, &user.User)
}

func (h *AuthHandler) issueTokens(c *fiber.Ctx, user *models.User) error {
	accessToken, err := middleware.GenerateToken(h.jwtSecret, user, accessTokenExpiry)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate access token")
	}

	refreshToken, err := middleware.GenerateToken(h.jwtSecret, user, refreshTokenExpiry)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate refresh token")
	}

	return c.JSON(tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "bearer",
		ExpiresIn:    int(accessTokenExpiry.Seconds()),
	})
}
