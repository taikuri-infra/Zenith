package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Max response body size from Keycloak (1MB).
const maxKeycloakResponseSize = 1 << 20

// Max OTP entries to prevent OOM from flooding.
const maxOTPEntries = 10000

// Max webhooks per pool.
const maxWebhooksPerPool = 20

// In-memory OTP store: key = poolId:phone → {code, exp}
type otpEntry struct {
	Code string
	Exp  time.Time
}

var (
	otpStore   = make(map[string]otpEntry)
	otpMu      sync.Mutex
	webhooksMu sync.Mutex
	webhooks   = make(map[string][]ports.WebhookConfig) // poolId → webhooks
)

// Safe name pattern: alphanumeric, hyphens, underscores, dots, max 64 chars.
var safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,64}$`)

// httpClient with timeout for Keycloak proxy calls.
var keycloakHTTPClient = &http.Client{Timeout: 10 * time.Second}

// AuthPoolHandler manages auth pool HTTP endpoints.
type AuthPoolHandler struct {
	poolSvc     *services.AuthPoolService
	poolRepo    ports.AuthPoolRepository
	keycloakURL string // internal URL for proxying token requests
	email       ports.EmailSender
	appURL      string
}

// NewAuthPoolHandler creates a new AuthPoolHandler.
func NewAuthPoolHandler(poolSvc *services.AuthPoolService, poolRepo ports.AuthPoolRepository, keycloakURL string) *AuthPoolHandler {
	return &AuthPoolHandler{poolSvc: poolSvc, poolRepo: poolRepo, keycloakURL: keycloakURL}
}

// SetEmailSender sets the email sender for magic link delivery.
func (h *AuthPoolHandler) SetEmailSender(email ports.EmailSender, appURL string) {
	h.email = email
	h.appURL = appURL
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

// ResetPassword resets a user's password using an HMAC-signed reset token.
// The token is generated by ForgotPassword (sent via Keycloak email) and validated here.
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
	if input.Email == "" || input.ResetToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and reset_token are required")
	}
	if input.NewPassword == "" || len(input.NewPassword) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "new_password must be at least 8 characters")
	}

	// Validate the reset token (HMAC-signed, same scheme as magic link, 15 min window)
	now := time.Now().Unix()
	valid := false
	for sec := int64(0); sec <= 900; sec++ {
		exp := now + 900 - sec
		expected := magicLinkHMAC(pool.ClientSecret, "reset:"+input.Email, exp)
		if hmac.Equal([]byte(input.ResetToken), []byte(expected)) {
			valid = true
			break
		}
	}
	if !valid {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired reset token")
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

// ---------------------------------------------------------------------------
// Email verification (authenticated — pool owner)
// ---------------------------------------------------------------------------

// SendVerificationEmail sends an email verification to a user.
// POST /api/v1/auth-pools/:poolId/users/:userId/verify-email
func (h *AuthPoolHandler) SendVerificationEmail(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	if err := h.poolSvc.SendVerifyEmail(c.Context(), pool, c.Params("userId")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to send verification email")
	}
	return c.JSON(fiber.Map{"message": "verification email sent"})
}

// ---------------------------------------------------------------------------
// User metadata (authenticated — pool owner)
// ---------------------------------------------------------------------------

// GetUserMetadata returns custom attributes for a user.
// GET /api/v1/auth-pools/:poolId/users/:userId/metadata
func (h *AuthPoolHandler) GetUserMetadata(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	metadata, err := h.poolSvc.GetUserMetadata(c.Context(), pool, c.Params("userId"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get user metadata")
	}
	return c.JSON(metadata)
}

// SetUserMetadata sets custom attributes on a user.
// PUT /api/v1/auth-pools/:poolId/users/:userId/metadata
func (h *AuthPoolHandler) SetUserMetadata(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Metadata map[string][]string `json:"metadata"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Metadata == nil {
		return fiber.NewError(fiber.StatusBadRequest, "metadata is required")
	}
	// Limit: max 20 keys, each value max 256 chars
	if len(input.Metadata) > 20 {
		return fiber.NewError(fiber.StatusBadRequest, "maximum 20 metadata keys allowed")
	}
	for k, vals := range input.Metadata {
		if !safeNamePattern.MatchString(k) {
			return fiber.NewError(fiber.StatusBadRequest, "metadata key must be alphanumeric")
		}
		for _, v := range vals {
			if len(v) > 256 {
				return fiber.NewError(fiber.StatusBadRequest, "metadata values must be at most 256 characters")
			}
		}
	}

	if err := h.poolSvc.SetUserMetadata(c.Context(), pool, c.Params("userId"), input.Metadata); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to set user metadata")
	}
	return c.JSON(fiber.Map{"message": "metadata updated"})
}

// ---------------------------------------------------------------------------
// MFA / Credentials (authenticated — pool owner)
// ---------------------------------------------------------------------------

// GetUserCredentials returns all credentials (password, TOTP) for a user.
// GET /api/v1/auth-pools/:poolId/users/:userId/credentials
func (h *AuthPoolHandler) GetUserCredentials(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	creds, err := h.poolSvc.GetUserCredentials(c.Context(), pool, c.Params("userId"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get credentials")
	}
	if creds == nil {
		creds = []ports.IdentityCredential{}
	}
	return c.JSON(creds)
}

// DeleteUserCredential removes a specific credential (e.g., remove TOTP).
// DELETE /api/v1/auth-pools/:poolId/users/:userId/credentials/:credentialId
func (h *AuthPoolHandler) DeleteUserCredential(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	if err := h.poolSvc.DeleteUserCredential(c.Context(), pool, c.Params("userId"), c.Params("credentialId")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete credential")
	}
	return c.JSON(fiber.Map{"message": "credential deleted"})
}

// ---------------------------------------------------------------------------
// Session management (authenticated — pool owner)
// ---------------------------------------------------------------------------

// GetUserSessions returns all active sessions for a user.
// GET /api/v1/auth-pools/:poolId/users/:userId/sessions
func (h *AuthPoolHandler) GetUserSessions(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	sessions, err := h.poolSvc.GetUserSessions(c.Context(), pool, c.Params("userId"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get sessions")
	}
	if sessions == nil {
		sessions = []ports.IdentitySession{}
	}
	return c.JSON(sessions)
}

// RevokeUserSession revokes a single session.
// DELETE /api/v1/auth-pools/:poolId/users/:userId/sessions/:sessionId
func (h *AuthPoolHandler) RevokeUserSession(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	if err := h.poolSvc.RevokeUserSession(c.Context(), pool, c.Params("sessionId")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to revoke session")
	}
	return c.JSON(fiber.Map{"message": "session revoked"})
}

// RevokeAllUserSessions revokes all sessions for a user.
// DELETE /api/v1/auth-pools/:poolId/users/:userId/sessions
func (h *AuthPoolHandler) RevokeAllUserSessions(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	if err := h.poolSvc.RevokeAllUserSessions(c.Context(), pool, c.Params("userId")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to revoke sessions")
	}
	return c.JSON(fiber.Map{"message": "all sessions revoked"})
}

// ---------------------------------------------------------------------------
// Social / Identity Provider management (authenticated — pool owner)
// ---------------------------------------------------------------------------

// CreateSocialProvider adds a social login provider (Google, GitHub, Apple, etc.).
// POST /api/v1/auth-pools/:poolId/providers
func (h *AuthPoolHandler) CreateSocialProvider(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	var input ports.IdentityProviderConfig
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.ProviderID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "provider_id is required (google, github, apple, etc.)")
	}
	if input.ClientID == "" || input.ClientSecret == "" {
		return fiber.NewError(fiber.StatusBadRequest, "client_id and client_secret are required")
	}
	// Auto-generate alias from provider_id if empty
	if input.Alias == "" {
		input.Alias = input.ProviderID
	}
	if input.DisplayName == "" {
		input.DisplayName = strings.ToUpper(input.ProviderID[:1]) + input.ProviderID[1:]
	}
	input.Enabled = true

	if err := h.poolSvc.CreateIdentityProvider(c.Context(), pool, input); err != nil {
		return fiber.NewError(fiber.StatusConflict, "failed to create provider")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "provider created", "alias": input.Alias})
}

// ListSocialProviders returns all configured social login providers.
// GET /api/v1/auth-pools/:poolId/providers
func (h *AuthPoolHandler) ListSocialProviders(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	providers, err := h.poolSvc.ListIdentityProviders(c.Context(), pool)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list providers")
	}
	if providers == nil {
		providers = []ports.IdentityProviderConfig{}
	}
	return c.JSON(providers)
}

// DeleteSocialProvider removes a social login provider.
// DELETE /api/v1/auth-pools/:poolId/providers/:alias
func (h *AuthPoolHandler) DeleteSocialProvider(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	if err := h.poolSvc.DeleteIdentityProvider(c.Context(), pool, c.Params("alias")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete provider")
	}
	return c.JSON(fiber.Map{"message": "provider deleted"})
}

// ---------------------------------------------------------------------------
// Public: user metadata self-service (requires Bearer token)
// ---------------------------------------------------------------------------

// GetCurrentUserMetadata returns the current user's custom metadata.
// GET /api/v1/auth-pools/:poolId/user/metadata
func (h *AuthPoolHandler) GetCurrentUserMetadata(c *fiber.Ctx) error {
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
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	metadata, err := h.poolSvc.GetUserMetadata(c.Context(), pool, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get metadata")
	}
	return c.JSON(metadata)
}

// SetCurrentUserMetadata updates the current user's custom metadata.
// PUT /api/v1/auth-pools/:poolId/user/metadata
func (h *AuthPoolHandler) SetCurrentUserMetadata(c *fiber.Ctx) error {
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
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	var input struct {
		Metadata map[string][]string `json:"metadata"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Metadata == nil {
		return fiber.NewError(fiber.StatusBadRequest, "metadata is required")
	}
	if len(input.Metadata) > 20 {
		return fiber.NewError(fiber.StatusBadRequest, "maximum 20 metadata keys allowed")
	}
	for k, vals := range input.Metadata {
		if !safeNamePattern.MatchString(k) {
			return fiber.NewError(fiber.StatusBadRequest, "metadata key must be alphanumeric")
		}
		for _, v := range vals {
			if len(v) > 256 {
				return fiber.NewError(fiber.StatusBadRequest, "metadata values must be at most 256 characters")
			}
		}
	}

	if err := h.poolSvc.SetUserMetadata(c.Context(), pool, userID, input.Metadata); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to set metadata")
	}
	return c.JSON(fiber.Map{"message": "metadata updated"})
}

// ---------------------------------------------------------------------------
// Invite user (authenticated — pool owner)
// ---------------------------------------------------------------------------

// InviteUser sends an invitation email to a new user.
// POST /api/v1/auth-pools/:poolId/users/invite
func (h *AuthPoolHandler) InviteUser(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}

	user, err := h.poolSvc.InviteUser(c.Context(), pool, input.Email, input.FirstName, input.LastName)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "failed to invite user")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user":    user,
		"message": "invitation email sent",
	})
}

// ---------------------------------------------------------------------------
// Public: Anonymous sign-in
// ---------------------------------------------------------------------------

// AnonymousSignIn creates a temporary anonymous user and returns tokens (public).
// POST /api/v1/auth-pools/:poolId/anonymous
func (h *AuthPoolHandler) AnonymousSignIn(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	user, password, err := h.poolSvc.CreateAnonymousUser(c.Context(), pool)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "failed to create anonymous session")
	}

	// Auto-login the anonymous user
	tokens, tokenErr := h.doKeycloakTokenRequest(pool, url.Values{
		"client_id":     {pool.ClientID},
		"client_secret": {pool.ClientSecret},
		"grant_type":    {"password"},
		"username":      {user.Email},
		"password":      {password},
		"scope":         {"openid"},
	})

	resp := fiber.Map{
		"user":      user,
		"anonymous": true,
	}
	if tokenErr == nil {
		resp["access_token"] = tokens["access_token"]
		resp["refresh_token"] = tokens["refresh_token"]
		resp["expires_in"] = tokens["expires_in"]
		resp["token_type"] = tokens["token_type"]
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// ---------------------------------------------------------------------------
// Public: Magic link (passwordless email login)
// ---------------------------------------------------------------------------

// magicLinkSecret generates an HMAC for magic link tokens using pool secret as key.
func magicLinkHMAC(poolSecret, email string, exp int64) string {
	h := hmac.New(sha256.New, []byte(poolSecret))
	h.Write([]byte(fmt.Sprintf("%s:%d", email, exp)))
	return hex.EncodeToString(h.Sum(nil))
}

// SendMagicLink generates a magic link token and returns it (public).
// In production, this would send an email with the link.
// POST /api/v1/auth-pools/:poolId/magic-link
func (h *AuthPoolHandler) SendMagicLink(c *fiber.Ctx) error {
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

	// Generate magic link token (HMAC-signed, 15 min expiry)
	exp := time.Now().Add(15 * time.Minute).Unix()
	token := magicLinkHMAC(pool.ClientSecret, input.Email, exp)

	// Send the magic link via email
	if h.email != nil && h.appURL != "" {
		magicURL := fmt.Sprintf("%s/auth/magic-link?pool=%s&token=%s&exp=%d&email=%s",
			strings.TrimRight(h.appURL, "/"), pool.ID, url.QueryEscape(token), exp, url.QueryEscape(input.Email))
		subject := "Sign in to " + pool.Name
		htmlBody := fmt.Sprintf(`
			<h2 style="color: #fafafa; font-size: 20px; margin: 0 0 16px;">Magic Link Sign In</h2>
			<p style="color: #a3a3a3; font-size: 14px; line-height: 1.6; margin: 0 0 24px;">
				Click the button below to sign in. This link expires in 15 minutes.
			</p>
			<div style="text-align: center; margin: 24px 0;">
				<a href="%s" style="display: inline-block; background-color: #10b981; color: #ffffff; text-decoration: none; font-weight: 600; font-size: 14px; padding: 12px 32px; border-radius: 8px;">
					Sign In
				</a>
			</div>
			<p style="color: #737373; font-size: 12px; line-height: 1.5; margin: 24px 0 0;">
				If you didn't request this link, you can safely ignore this email.
			</p>`, magicURL)
		if err := h.email.SendGenericEmail(c.Context(), input.Email, subject, htmlBody); err != nil {
			slog.Error("failed to send magic link email", "email", input.Email, "error", err)
		}
	} else {
		slog.Warn("magic link generated but email sender not configured", "email", input.Email)
	}
	_ = token // token is sent via email only, never in API response
	_ = exp

	// Always return success to prevent email enumeration
	return c.JSON(fiber.Map{
		"message": "if the email exists, a magic link has been sent",
	})
}

// VerifyMagicLink verifies a magic link token and returns auth tokens (public).
// POST /api/v1/auth-pools/:poolId/magic-link/verify
func (h *AuthPoolHandler) VerifyMagicLink(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" || input.Token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and token are required")
	}

	// Token format: "hex_hmac:expiry_unix" — verify HMAC and check not expired
	now := time.Now().Unix()
	valid := false
	parts := strings.SplitN(input.Token, ":", 2)
	if len(parts) == 2 {
		exp, parseErr := strconv.ParseInt(parts[1], 10, 64)
		if parseErr == nil && exp > now {
			expected := magicLinkHMAC(pool.ClientSecret, input.Email, exp)
			if hmac.Equal([]byte(parts[0]), []byte(expected)) {
				valid = true
			}
		}
	}

	if !valid {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired magic link")
	}

	// Find user and issue admin-granted token via password reset + login
	user, err := h.poolSvc.FindUserByEmail(c.Context(), pool, input.Email)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired magic link")
	}

	// Set a random temporary password (not deterministic) and login
	tmpPassword := uuid.New().String()
	if err := h.poolSvc.SetUserPassword(c.Context(), pool, user.ID, tmpPassword); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "authentication failed")
	}

	tokens, err := h.doKeycloakTokenRequest(pool, url.Values{
		"client_id":     {pool.ClientID},
		"client_secret": {pool.ClientSecret},
		"grant_type":    {"password"},
		"username":      {input.Email},
		"password":      {tmpPassword},
		"scope":         {"openid"},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "authentication failed")
	}

	return c.JSON(tokens)
}

// ---------------------------------------------------------------------------
// Public: PKCE authorization URL
// ---------------------------------------------------------------------------

// GetAuthorizationURL returns the OIDC authorization URL for PKCE/code flow (public).
// GET /api/v1/auth-pools/:poolId/authorize
func (h *AuthPoolHandler) GetAuthorizationURL(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")
	codeChallenge := c.Query("code_challenge")
	codeChallengeMethod := c.Query("code_challenge_method", "S256")
	scope := c.Query("scope", "openid")

	if redirectURI == "" {
		return fiber.NewError(fiber.StatusBadRequest, "redirect_uri is required")
	}

	// Build Keycloak authorization URL
	authURL := pool.IssuerURL + "/protocol/openid-connect/auth"
	params := url.Values{
		"client_id":     {pool.ClientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {scope},
	}
	if state != "" {
		params.Set("state", state)
	}
	if codeChallenge != "" {
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", codeChallengeMethod)
	}

	return c.JSON(fiber.Map{
		"authorization_url": authURL + "?" + params.Encode(),
	})
}

// ExchangeCode exchanges an authorization code for tokens (PKCE flow, public).
// POST /api/v1/auth-pools/:poolId/token/code
func (h *AuthPoolHandler) ExchangeCode(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Code         string `json:"code"`
		RedirectURI  string `json:"redirect_uri"`
		CodeVerifier string `json:"code_verifier"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Code == "" || input.RedirectURI == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code and redirect_uri are required")
	}

	form := url.Values{
		"client_id":     {pool.ClientID},
		"client_secret": {pool.ClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {input.Code},
		"redirect_uri":  {input.RedirectURI},
	}
	if input.CodeVerifier != "" {
		form.Set("code_verifier", input.CodeVerifier)
	}

	tokens, err := h.doKeycloakTokenRequest(pool, form)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired code")
	}

	return c.JSON(tokens)
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
// Email / SMTP settings (authenticated — pool owner)
// ---------------------------------------------------------------------------

// GetEmailSettings returns SMTP and email config for a pool.
// GET /api/v1/auth-pools/:poolId/email-settings
func (h *AuthPoolHandler) GetEmailSettings(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}
	settings, err := h.poolSvc.GetEmailSettings(c.Context(), pool)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get email settings")
	}
	// Never expose password
	settings.Password = ""
	return c.JSON(settings)
}

// UpdateEmailSettings updates SMTP and email config for a pool.
// PUT /api/v1/auth-pools/:poolId/email-settings
func (h *AuthPoolHandler) UpdateEmailSettings(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	var input ports.EmailSettings
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Host == "" || input.From == "" {
		return fiber.NewError(fiber.StatusBadRequest, "host and from are required")
	}

	if err := h.poolSvc.UpdateEmailSettings(c.Context(), pool, &input); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update email settings")
	}
	return c.JSON(fiber.Map{"message": "email settings updated"})
}

// ---------------------------------------------------------------------------
// Phone/SMS OTP (public)
// ---------------------------------------------------------------------------

// generateOTP generates a 6-digit OTP.
func generateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(999999))
	return fmt.Sprintf("%06d", n.Int64())
}

// SendOTP generates and stores a 6-digit OTP for a phone number (public).
// POST /api/v1/auth-pools/:poolId/otp/send
func (h *AuthPoolHandler) SendOTP(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Phone == "" {
		return fiber.NewError(fiber.StatusBadRequest, "phone is required")
	}

	code := generateOTP()
	key := pool.ID + ":" + input.Phone

	otpMu.Lock()
	// Evict expired entries if store is full to prevent OOM
	if len(otpStore) >= maxOTPEntries {
		now := time.Now()
		for k, v := range otpStore {
			if now.After(v.Exp) {
				delete(otpStore, k)
			}
		}
		// If still full after cleanup, reject
		if len(otpStore) >= maxOTPEntries {
			otpMu.Unlock()
			return fiber.NewError(fiber.StatusServiceUnavailable, "OTP service is busy, try again later")
		}
	}
	otpStore[key] = otpEntry{Code: code, Exp: time.Now().Add(5 * time.Minute)}
	otpMu.Unlock()

	// SMS/OTP delivery is not supported. Use email-based authentication (magic link) instead.
	// The OTP is stored but cannot be delivered — this endpoint exists for future extensibility.
	slog.Warn("OTP requested but SMS delivery is not configured — use magic link auth instead", "phone", input.Phone)
	return c.JSON(fiber.Map{
		"message":    "OTP sent",
		"expires_in": 300,
	})
}

// VerifyOTP verifies a phone OTP and returns tokens (public).
// POST /api/v1/auth-pools/:poolId/otp/verify
func (h *AuthPoolHandler) VerifyOTP(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	var input struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Phone == "" || input.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "phone and code are required")
	}

	key := pool.ID + ":" + input.Phone

	otpMu.Lock()
	entry, exists := otpStore[key]
	if exists {
		delete(otpStore, key)
	}
	otpMu.Unlock()

	if !exists || time.Now().After(entry.Exp) || subtle.ConstantTimeCompare([]byte(entry.Code), []byte(input.Code)) != 1 {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired OTP")
	}

	// Find or create user by phone-as-email
	email := input.Phone + "@phone.zenith"
	user, _ := h.poolSvc.FindUserByEmail(c.Context(), pool, email)
	if user == nil {
		// Auto-create user on first OTP verify
		tmpPwd := uuid.New().String()
		created, createErr := h.poolSvc.CreateUser(c.Context(), pool, email, tmpPwd, input.Phone, "")
		if createErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to create user")
		}
		user = created
	}

	// Set temp password and login
	tmpPwd := uuid.New().String()
	_ = h.poolSvc.SetUserPassword(c.Context(), pool, user.ID, tmpPwd)

	tokens, err := h.doKeycloakTokenRequest(pool, url.Values{
		"client_id":     {pool.ClientID},
		"client_secret": {pool.ClientSecret},
		"grant_type":    {"password"},
		"username":      {email},
		"password":      {tmpPwd},
		"scope":         {"openid"},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "authentication failed")
	}

	return c.JSON(tokens)
}

// ---------------------------------------------------------------------------
// Webhooks (authenticated — pool owner, in-memory store)
// ---------------------------------------------------------------------------

// CreateWebhook registers a webhook for auth events.
// POST /api/v1/auth-pools/:poolId/webhooks
func (h *AuthPoolHandler) CreateWebhook(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	var input struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.URL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url is required")
	}

	// SSRF protection: validate webhook URL
	if err := validateWebhookURL(input.URL); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if len(input.Events) == 0 {
		input.Events = []string{"signup", "login", "logout", "password_reset", "user_deleted"}
	}

	// Enforce max webhooks per pool
	webhooksMu.Lock()
	if len(webhooks[pool.ID]) >= maxWebhooksPerPool {
		webhooksMu.Unlock()
		return fiber.NewError(fiber.StatusConflict, fmt.Sprintf("maximum %d webhooks per pool", maxWebhooksPerPool))
	}
	webhooksMu.Unlock()

	wh := ports.WebhookConfig{
		ID:     uuid.New().String(), // full UUID, not truncated
		URL:    input.URL,
		Events: input.Events,
		Secret: uuid.New().String(),
		Active: true,
	}

	webhooksMu.Lock()
	webhooks[pool.ID] = append(webhooks[pool.ID], wh)
	webhooksMu.Unlock()

	return c.Status(fiber.StatusCreated).JSON(wh)
}

// ListWebhooks returns all webhooks for a pool.
// GET /api/v1/auth-pools/:poolId/webhooks
func (h *AuthPoolHandler) ListWebhooks(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	webhooksMu.Lock()
	whs := webhooks[pool.ID]
	webhooksMu.Unlock()

	if whs == nil {
		whs = []ports.WebhookConfig{}
	}
	// Don't expose secrets in list
	result := make([]fiber.Map, len(whs))
	for i, wh := range whs {
		result[i] = fiber.Map{
			"id":     wh.ID,
			"url":    wh.URL,
			"events": wh.Events,
			"active": wh.Active,
		}
	}
	return c.JSON(result)
}

// DeleteWebhook removes a webhook.
// DELETE /api/v1/auth-pools/:poolId/webhooks/:webhookId
func (h *AuthPoolHandler) DeleteWebhook(c *fiber.Ctx) error {
	pool, err := h.requirePool(c)
	if err != nil {
		return err
	}

	whID := c.Params("webhookId")

	webhooksMu.Lock()
	whs := webhooks[pool.ID]
	for i, wh := range whs {
		if wh.ID == whID {
			webhooks[pool.ID] = append(whs[:i], whs[i+1:]...)
			break
		}
	}
	webhooksMu.Unlock()

	return c.JSON(fiber.Map{"message": "webhook deleted"})
}

// ---------------------------------------------------------------------------
// Client SDK (public — serves a JS SDK)
// ---------------------------------------------------------------------------

// GetSDK returns a JavaScript SDK for the pool (public).
// GET /api/v1/auth-pools/:poolId/sdk.js
func (h *AuthPoolHandler) GetSDK(c *fiber.Ctx) error {
	pool, err := h.loadActivePool(c)
	if err != nil {
		return err
	}

	origin := c.Get("Origin")
	if origin == "" {
		origin = c.Get("Referer")
	}
	if origin == "" {
		origin = c.Protocol() + "://" + c.Hostname()
	}

	// JSON-encode origin to prevent XSS via crafted Origin/Referer headers
	originJSON, _ := json.Marshal(origin)
	poolIDJSON, _ := json.Marshal(pool.ID)

	sdk := fmt.Sprintf(`// ZenAuth SDK — Auto-generated
(function(global) {
  'use strict';
  var BASE = %s + '/api/v1/auth-pools/' + %s;

  function request(path, opts) {
    opts = opts || {};
    var headers = { 'Content-Type': 'application/json' };
    if (opts.token) headers['Authorization'] = 'Bearer ' + opts.token;
    return fetch(BASE + path, {
      method: opts.method || 'GET',
      headers: headers,
      body: opts.body ? JSON.stringify(opts.body) : undefined
    }).then(function(r) { return r.json(); });
  }

  global.ZenAuth = {
    signup: function(email, password, firstName, lastName) {
      return request('/signup', { method: 'POST', body: { email: email, password: password, first_name: firstName || '', last_name: lastName || '' } });
    },
    login: function(email, password) {
      return request('/login', { method: 'POST', body: { email: email, password: password } });
    },
    logout: function(refreshToken) {
      return request('/logout', { method: 'POST', body: { refresh_token: refreshToken } });
    },
    refresh: function(refreshToken) {
      return request('/refresh', { method: 'POST', body: { refresh_token: refreshToken } });
    },
    getUser: function(token) {
      return request('/user', { token: token });
    },
    updateUser: function(token, firstName, lastName) {
      return request('/user', { method: 'PUT', token: token, body: { first_name: firstName, last_name: lastName } });
    },
    changePassword: function(token, currentPassword, newPassword) {
      return request('/user/password', { method: 'POST', token: token, body: { current_password: currentPassword, new_password: newPassword } });
    },
    forgotPassword: function(email) {
      return request('/forgot-password', { method: 'POST', body: { email: email } });
    },
    resetPassword: function(email, newPassword, resetToken) {
      return request('/reset-password', { method: 'POST', body: { email: email, new_password: newPassword, reset_token: resetToken || '' } });
    },
    anonymous: function() {
      return request('/anonymous', { method: 'POST' });
    },
    sendMagicLink: function(email) {
      return request('/magic-link', { method: 'POST', body: { email: email } });
    },
    verifyMagicLink: function(email, token) {
      return request('/magic-link/verify', { method: 'POST', body: { email: email, token: token } });
    },
    sendOTP: function(phone) {
      return request('/otp/send', { method: 'POST', body: { phone: phone } });
    },
    verifyOTP: function(phone, code) {
      return request('/otp/verify', { method: 'POST', body: { phone: phone, code: code } });
    },
    getMetadata: function(token) {
      return request('/user/metadata', { token: token });
    },
    setMetadata: function(token, metadata) {
      return request('/user/metadata', { method: 'PUT', token: token, body: { metadata: metadata } });
    }
  };
})(typeof window !== 'undefined' ? window : this);
`, string(originJSON), string(poolIDJSON))

	c.Set("Content-Type", "application/javascript")
	c.Set("Cache-Control", "public, max-age=3600")
	return c.SendString(sdk)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// validateWebhookURL checks that a webhook URL is safe (no SSRF to internal services).
func validateWebhookURL(rawURL string) error {
	if !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("webhook URL must use HTTPS")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	host := parsed.Hostname()

	// Block Kubernetes internal domains
	if strings.HasSuffix(host, ".svc.cluster.local") || strings.HasSuffix(host, ".svc") ||
		strings.HasSuffix(host, ".local") || host == "kubernetes" || host == "kubernetes.default" {
		return fmt.Errorf("internal URLs are not allowed")
	}

	// Resolve and block private IPs
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve hostname")
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("webhook URL must point to a public IP address")
		}
	}
	return nil
}

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
