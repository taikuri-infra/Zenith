package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// Max response body size from Keycloak (1MB).
const maxKeycloakResponseSize = 1 << 20

// Safe name pattern: alphanumeric, hyphens, underscores, dots, max 64 chars.
var safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,64}$`)

// httpClient with timeout for Keycloak proxy calls.
var keycloakHTTPClient = &http.Client{Timeout: 10 * time.Second}

// AuthPoolHandler manages auth pool HTTP endpoints.
type AuthPoolHandler struct {
	poolSvc     *services.AuthPoolService
	poolRepo    ports.AuthPoolRepository
	keycloakURL string // internal URL for proxying token requests
}

// NewAuthPoolHandler creates a new AuthPoolHandler.
func NewAuthPoolHandler(poolSvc *services.AuthPoolService, poolRepo ports.AuthPoolRepository, keycloakURL string) *AuthPoolHandler {
	return &AuthPoolHandler{poolSvc: poolSvc, poolRepo: poolRepo, keycloakURL: keycloakURL}
}

// ---------------------------------------------------------------------------
// Pool CRUD (authenticated — pool owner)
// ---------------------------------------------------------------------------

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
	if !safeNamePattern.MatchString(input.Name) {
		return fiber.NewError(fiber.StatusBadRequest, "name must be 1-64 characters: letters, numbers, hyphens, underscores, dots")
	}

	pool, err := h.poolSvc.CreatePool(c.Context(), userID, input.ProjectID, input.Name)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "failed to create pool")
	}

	return c.Status(fiber.StatusCreated).JSON(toAuthPoolInfo(pool, true))
}

// ListPools returns all pools for the current user.
// GET /api/v1/auth-pools
func (h *AuthPoolHandler) ListPools(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	pools, err := h.poolSvc.ListPools(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list pools")
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
	// Secret visible to owner but only via explicit reveal
	return c.JSON(toAuthPoolInfo(pool, false))
}

// RevealSecret returns the pool's client secret (explicit action).
// POST /api/v1/auth-pools/:poolId/reveal-secret
func (h *AuthPoolHandler) RevealSecret(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"client_secret": pool.ClientSecret})
}

// DeletePool deletes a pool and its realm.
// DELETE /api/v1/auth-pools/:poolId
func (h *AuthPoolHandler) DeletePool(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	if err := h.poolSvc.DeletePool(c.Context(), pool); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete pool")
	}

	return c.JSON(fiber.Map{"message": "auth pool deleted"})
}

// ---------------------------------------------------------------------------
// User management (authenticated — pool owner)
// ---------------------------------------------------------------------------

// CreateUser creates a user in a pool (admin action).
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
	if len(input.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	user, err := h.poolSvc.CreateUser(c.Context(), pool, input.Email, input.Password, input.FirstName, input.LastName)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "failed to create user")
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
	if first < 0 {
		first = 0
	}
	max, _ := strconv.Atoi(c.Query("limit", "20"))
	if max <= 0 || max > 100 {
		max = 20
	}

	users, total, err := h.poolSvc.ListUsers(c.Context(), pool, first, max)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list users")
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
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete user")
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
		return fiber.NewError(fiber.StatusInternalServerError, "failed to disable user")
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
		return fiber.NewError(fiber.StatusInternalServerError, "failed to enable user")
	}

	return c.JSON(fiber.Map{"message": "user enabled"})
}

// ---------------------------------------------------------------------------
// Role management (authenticated — pool owner)
// ---------------------------------------------------------------------------

// CreateRole creates a role in a pool.
// POST /api/v1/auth-pools/:poolId/roles
func (h *AuthPoolHandler) CreateRole(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if !safeNamePattern.MatchString(input.Name) {
		return fiber.NewError(fiber.StatusBadRequest, "role name must be 1-64 characters: letters, numbers, hyphens, underscores, dots")
	}
	// Block built-in Keycloak role names
	lower := strings.ToLower(input.Name)
	if lower == "uma_authorization" || lower == "offline_access" || strings.HasPrefix(lower, "default-roles-") {
		return fiber.NewError(fiber.StatusBadRequest, "this role name is reserved")
	}

	if err := h.poolSvc.CreateRole(c.Context(), pool, input.Name, input.Description); err != nil {
		return fiber.NewError(fiber.StatusConflict, "failed to create role")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "role created", "name": input.Name})
}

// ListRoles returns all custom roles in a pool.
// GET /api/v1/auth-pools/:poolId/roles
func (h *AuthPoolHandler) ListRoles(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	roles, err := h.poolSvc.ListRoles(c.Context(), pool)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list roles")
	}
	if roles == nil {
		roles = []ports.IdentityRole{}
	}

	return c.JSON(roles)
}

// DeleteRole removes a role from a pool.
// DELETE /api/v1/auth-pools/:poolId/roles/:roleName
func (h *AuthPoolHandler) DeleteRole(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	if err := h.poolSvc.DeleteRole(c.Context(), pool, c.Params("roleName")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete role")
	}

	return c.JSON(fiber.Map{"message": "role deleted"})
}

// GetUserRoles returns roles assigned to a user.
// GET /api/v1/auth-pools/:poolId/users/:userId/roles
func (h *AuthPoolHandler) GetUserRoles(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	roles, err := h.poolSvc.GetUserRoles(c.Context(), pool, c.Params("userId"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get user roles")
	}
	if roles == nil {
		roles = []ports.IdentityRole{}
	}

	return c.JSON(roles)
}

// AssignRoleToUser assigns a role to a user.
// POST /api/v1/auth-pools/:poolId/users/:userId/roles
func (h *AuthPoolHandler) AssignRoleToUser(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	var input struct {
		RoleName string `json:"role_name"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.RoleName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "role_name is required")
	}

	if err := h.poolSvc.AssignRoleToUser(c.Context(), pool, c.Params("userId"), input.RoleName); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to assign role")
	}

	return c.JSON(fiber.Map{"message": "role assigned"})
}

// RemoveRoleFromUser removes a role from a user.
// DELETE /api/v1/auth-pools/:poolId/users/:userId/roles/:roleName
func (h *AuthPoolHandler) RemoveRoleFromUser(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	if err := h.poolSvc.RemoveRoleFromUser(c.Context(), pool, c.Params("userId"), c.Params("roleName")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to remove role")
	}

	return c.JSON(fiber.Map{"message": "role removed"})
}

// ---------------------------------------------------------------------------
// Public auth endpoints (no JWT required — used by pool end-users)
// Supabase-style: signup, login, logout, refresh, forgot-password, reset-password, user
// ---------------------------------------------------------------------------

// Signup registers a new user in a pool (public).
// POST /api/v1/auth-pools/:poolId/signup
func (h *AuthPoolHandler) Signup(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" || input.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}
	if len(input.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	user, err := h.poolSvc.CreateUser(c.Context(), pool, input.Email, input.Password, input.FirstName, input.LastName)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "failed to create account")
	}

	// Auto-login: issue tokens immediately after signup
	tokens, tokenErr := h.doKeycloakTokenRequest(pool, url.Values{
		"client_id":     {pool.ClientID},
		"client_secret": {pool.ClientSecret},
		"grant_type":    {"password"},
		"username":      {input.Email},
		"password":      {input.Password},
		"scope":         {"openid"},
	})

	resp := fiber.Map{
		"user": user,
	}
	if tokenErr == nil {
		resp["access_token"] = tokens["access_token"]
		resp["refresh_token"] = tokens["refresh_token"]
		resp["expires_in"] = tokens["expires_in"]
		resp["token_type"] = tokens["token_type"]
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// Login authenticates a pool user and returns tokens (public).
// POST /api/v1/auth-pools/:poolId/login
func (h *AuthPoolHandler) Login(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" || input.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}

	tokens, err := h.doKeycloakTokenRequest(pool, url.Values{
		"client_id":     {pool.ClientID},
		"client_secret": {pool.ClientSecret},
		"grant_type":    {"password"},
		"username":      {input.Email},
		"password":      {input.Password},
		"scope":         {"openid"},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}

	return c.JSON(tokens)
}

// Refresh exchanges a refresh token for new tokens (public).
// POST /api/v1/auth-pools/:poolId/refresh
func (h *AuthPoolHandler) Refresh(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
	}

	tokens, err := h.doKeycloakTokenRequest(pool, url.Values{
		"client_id":     {pool.ClientID},
		"client_secret": {pool.ClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {input.RefreshToken},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired refresh token")
	}

	return c.JSON(tokens)
}

// Logout revokes the refresh token (public).
// POST /api/v1/auth-pools/:poolId/logout
func (h *AuthPoolHandler) Logout(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
	}

	logoutURL := h.keycloakURL + "/realms/" + pool.RealmName + "/protocol/openid-connect/logout"
	form := url.Values{
		"client_id":     {pool.ClientID},
		"client_secret": {pool.ClientSecret},
		"refresh_token": {input.RefreshToken},
	}

	resp, err := keycloakHTTPClient.PostForm(logoutURL, form)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to reach identity provider")
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, maxKeycloakResponseSize))

	return c.JSON(fiber.Map{"message": "logged out"})
}

// ForgotPassword triggers a password reset email (public).
// POST /api/v1/auth-pools/:poolId/forgot-password
func (h *AuthPoolHandler) ForgotPassword(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}

	// Trigger Keycloak's reset password action via admin API
	_ = h.poolSvc.SendPasswordReset(c.Context(), pool, input.Email)

	// Always return success to prevent email enumeration
	return c.JSON(fiber.Map{"message": "if the email exists, a reset link has been sent"})
}

// ResetPassword resets a user's password with admin override (public).
// This is a simplified flow — in production you'd use Keycloak's reset token flow.
// POST /api/v1/auth-pools/:poolId/reset-password
func (h *AuthPoolHandler) ResetPassword(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Email       string `json:"email"`
		NewPassword string `json:"new_password"`
		ResetToken  string `json:"reset_token"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.NewPassword == "" || len(input.NewPassword) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "new_password must be at least 8 characters")
	}

	if err := h.poolSvc.ResetPassword(c.Context(), pool, input.Email, input.NewPassword, input.ResetToken); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "failed to reset password")
	}

	return c.JSON(fiber.Map{"message": "password has been reset"})
}

// GetCurrentUser returns the current user info from their JWT (public, requires Bearer token).
// GET /api/v1/auth-pools/:poolId/user
func (h *AuthPoolHandler) GetCurrentUser(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	// Get userinfo from Keycloak using the bearer token
	bearerToken := c.Get("Authorization")
	if bearerToken == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "authorization header required")
	}
	bearerToken = strings.TrimPrefix(bearerToken, "Bearer ")
	bearerToken = strings.TrimPrefix(bearerToken, "bearer ")

	userinfoURL := h.keycloakURL + "/realms/" + pool.RealmName + "/protocol/openid-connect/userinfo"
	req, _ := http.NewRequestWithContext(c.Context(), "GET", userinfoURL, nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := keycloakHTTPClient.Do(req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to reach identity provider")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxKeycloakResponseSize))
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to read response")
	}

	if resp.StatusCode != 200 {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
	}

	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

// UpdateCurrentUser updates the current user's profile (public, requires Bearer token).
// PUT /api/v1/auth-pools/:poolId/user
func (h *AuthPoolHandler) UpdateCurrentUser(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	bearerToken := c.Get("Authorization")
	if bearerToken == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "authorization header required")
	}
	bearerToken = strings.TrimPrefix(bearerToken, "Bearer ")
	bearerToken = strings.TrimPrefix(bearerToken, "bearer ")

	// Get user ID from userinfo
	userInfo, err := h.getUserInfoFromToken(c, pool, bearerToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
	}
	userID, _ := userInfo["sub"].(string)
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	var input struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.poolSvc.UpdateUser(c.Context(), pool, userID, input.FirstName, input.LastName); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update profile")
	}

	return c.JSON(fiber.Map{"message": "profile updated"})
}

// ChangePassword changes the current user's password (public, requires Bearer token).
// POST /api/v1/auth-pools/:poolId/user/password
func (h *AuthPoolHandler) ChangePassword(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	bearerToken := c.Get("Authorization")
	if bearerToken == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "authorization header required")
	}
	bearerToken = strings.TrimPrefix(bearerToken, "Bearer ")
	bearerToken = strings.TrimPrefix(bearerToken, "bearer ")

	userInfo, err := h.getUserInfoFromToken(c, pool, bearerToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
	}
	userID, _ := userInfo["sub"].(string)
	email, _ := userInfo["email"].(string)
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.CurrentPassword == "" || input.NewPassword == "" {
		return fiber.NewError(fiber.StatusBadRequest, "current_password and new_password are required")
	}
	if len(input.NewPassword) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "new_password must be at least 8 characters")
	}

	// Verify current password by attempting login
	_, verifyErr := h.doKeycloakTokenRequest(pool, url.Values{
		"client_id":     {pool.ClientID},
		"client_secret": {pool.ClientSecret},
		"grant_type":    {"password"},
		"username":      {email},
		"password":      {input.CurrentPassword},
		"scope":         {"openid"},
	})
	if verifyErr != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "current password is incorrect")
	}

	if err := h.poolSvc.SetUserPassword(c.Context(), pool, userID, input.NewPassword); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to change password")
	}

	return c.JSON(fiber.Map{"message": "password changed"})
}

// TokenExchange is kept for backward compatibility.
// POST /api/v1/auth-pools/:poolId/token
func (h *AuthPoolHandler) TokenExchange(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input dto.TokenExchangeInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if input.GrantType == "" {
		input.GrantType = "password"
	}

	form := url.Values{
		"client_id":     {pool.ClientID},
		"client_secret": {pool.ClientSecret},
		"grant_type":    {input.GrantType},
	}

	switch input.GrantType {
	case "password":
		if input.Username == "" || input.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "username and password are required")
		}
		form.Set("username", input.Username)
		form.Set("password", input.Password)
		scope := input.Scope
		if scope == "" {
			scope = "openid"
		}
		form.Set("scope", scope)
	case "refresh_token":
		if input.RefreshToken == "" {
			return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
		}
		form.Set("refresh_token", input.RefreshToken)
	default:
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unsupported grant_type: %s", input.GrantType))
	}

	tokens, err := h.doKeycloakTokenRequest(pool, form)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "authentication failed")
	}

	return c.JSON(tokens)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// loadActivePool loads a pool by ID and checks it's active (no ownership check — for public endpoints).
func (h *AuthPoolHandler) loadActivePool(c *fiber.Ctx) (*entities.AuthPool, error) {
	poolID := c.Params("poolId")
	pool, err := h.poolRepo.GetPool(c.Context(), poolID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "auth pool not found")
	}
	if pool.Status != entities.AuthPoolStatusActive {
		return nil, fiber.NewError(fiber.StatusServiceUnavailable, "auth pool is not active")
	}
	return pool, nil
}

// requirePool loads the pool and verifies ownership (for admin endpoints).
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

// doKeycloakTokenRequest makes a token request to Keycloak and returns parsed response.
func (h *AuthPoolHandler) doKeycloakTokenRequest(pool *entities.AuthPool, form url.Values) (map[string]interface{}, error) {
	tokenURL := h.keycloakURL + "/realms/" + pool.RealmName + "/protocol/openid-connect/token"

	resp, err := keycloakHTTPClient.PostForm(tokenURL, form)
	if err != nil {
		return nil, fmt.Errorf("failed to reach identity provider")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxKeycloakResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("authentication failed (status %d)", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("invalid response format")
	}

	return result, nil
}

// getUserInfoFromToken retrieves user info from Keycloak using a bearer token.
func (h *AuthPoolHandler) getUserInfoFromToken(c *fiber.Ctx, pool *entities.AuthPool, token string) (map[string]interface{}, error) {
	userinfoURL := h.keycloakURL + "/realms/" + pool.RealmName + "/protocol/openid-connect/userinfo"
	req, _ := http.NewRequestWithContext(c.Context(), "GET", userinfoURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := keycloakHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxKeycloakResponseSize))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid token")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
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
