package handlers

import (
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/middleware"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	svc       *services.AuthService
	appURL    string // frontend URL for OAuth redirect
	blacklist *middleware.TokenBlacklist
	eventRepo ports.UserEventRepository
}

func NewAuthHandler(svc *services.AuthService, blacklist *middleware.TokenBlacklist) *AuthHandler {
	return &AuthHandler{svc: svc, blacklist: blacklist}
}

// SetEventRepo enables event tracking on auth actions.
func (h *AuthHandler) SetEventRepo(repo ports.UserEventRepository) {
	h.eventRepo = repo
}

// SetAppURL configures the frontend URL for OAuth callback redirects.
func (h *AuthHandler) SetAppURL(appURL string) {
	h.appURL = appURL
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	UTMSource    string `json:"utm_source"`
	UTMMedium    string `json:"utm_medium"`
	UTMCampaign  string `json:"utm_campaign"`
	UTMContent   string `json:"utm_content"`
	UTMTerm      string `json:"utm_term"`
	ReferrerURL  string `json:"referrer_url"`
	ReferralCode string `json:"referral_code"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type resendVerificationRequest struct {
	Email string `json:"email"`
}

type exchangeCodeRequest struct {
	Code string `json:"code"`
}

type mfaLoginRequest struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type messageResponse struct {
	Message string `json:"message"`
}

// Login authenticates a user and returns JWT tokens (or MFA challenge).
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}
	if len(req.Email) > 254 || len(req.Password) > 72 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid input length")
	}

	result, err := h.svc.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		// Return 403 for email verification errors, 401 for auth errors
		if err.Error() == "please verify your email before logging in" {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	// MFA required — return challenge token
	if result.MFARequired {
		return c.JSON(fiber.Map{
			"mfa_required": true,
			"mfa_token":    result.MFAToken,
		})
	}

	// Track login event
	h.trackEvent(c, entities.EventLogin, nil)

	return c.JSON(tokenResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		TokenType:    "bearer",
		ExpiresIn:    result.Tokens.ExpiresIn,
	})
}

// MFALogin completes a login that requires MFA verification.
func (h *AuthHandler) MFALogin(c *fiber.Ctx) error {
	var req mfaLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.MFAToken == "" || req.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "mfa_token and code are required")
	}

	tokens, err := h.svc.MFALogin(c.Context(), req.MFAToken, req.Code)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	return c.JSON(tokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "bearer",
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// Register creates a new user and sends a verification email.
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
	if len(req.Password) > 72 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be 72 characters or fewer")
	}
	if len(req.Email) > 254 || len(req.Name) > 255 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid input length")
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid email format")
	}

	result, err := h.svc.Register(c.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		// Generic message to prevent user enumeration
		return fiber.NewError(fiber.StatusConflict, "registration failed")
	}

	// Track signup event with UTM properties
	props := map[string]interface{}{}
	if req.UTMSource != "" {
		props["utm_source"] = req.UTMSource
	}
	if req.UTMMedium != "" {
		props["utm_medium"] = req.UTMMedium
	}
	if req.UTMCampaign != "" {
		props["utm_campaign"] = req.UTMCampaign
	}
	if req.ReferralCode != "" {
		props["referral_code"] = req.ReferralCode
	}

	// Store UTM data on user record via service
	if result.UserID != "" {
		h.svc.UpdateSignupSource(c.Context(), result.UserID, req.UTMSource, req.UTMMedium, req.UTMCampaign, req.UTMContent, req.UTMTerm, req.ReferrerURL, c.IP())
		h.svc.ProcessReferralCode(c.Context(), result.UserID, req.ReferralCode)
		h.trackEventForUser(c, result.UserID, entities.EventSignup, props)
	}

	// Email/password registration returns a message (verify email first)
	if result.Tokens == nil {
		return c.Status(fiber.StatusCreated).JSON(messageResponse{
			Message: result.Message,
		})
	}

	// OAuth registration returns tokens directly
	return c.Status(fiber.StatusCreated).JSON(tokenResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		TokenType:    "bearer",
		ExpiresIn:    result.Tokens.ExpiresIn,
	})
}

// VerifyEmail validates a verification token and returns JWT tokens.
func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	var req verifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "token is required")
	}

	tokens, err := h.svc.VerifyEmail(c.Context(), req.Token)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(tokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "bearer",
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// ResendVerification sends a new verification email.
func (h *AuthHandler) ResendVerification(c *fiber.Ctx) error {
	var req resendVerificationRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}

	// Always return 200 to not reveal whether the email exists
	_ = h.svc.ResendVerification(c.Context(), req.Email)

	return c.JSON(messageResponse{
		Message: "If an account exists with that email, a verification link has been sent",
	})
}

// OAuthRedirect initiates an OAuth flow by redirecting the user to the provider.
func (h *AuthHandler) OAuthRedirect(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider != "google" && provider != "github" {
		return fiber.NewError(fiber.StatusBadRequest, "unsupported OAuth provider")
	}

	// Build the callback URL from the current request.
	// Always use https — the API runs behind TLS-terminating proxy (Traefik/APISIX)
	// and OAuth providers require https redirect URIs.
	callbackURL := fmt.Sprintf("https://%s%s/callback", c.Hostname(), c.Path())

	redirectURL, state, err := h.svc.GetOAuthRedirectURLWithCallback(provider, callbackURL)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Set CSRF state cookie
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   600, // 10 minutes
		Path:     "/",
	})

	return c.Redirect(redirectURL, fiber.StatusFound)
}

// OAuthCallback handles the provider callback, validates state, and redirects to the frontend.
func (h *AuthHandler) OAuthCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		errorMsg := url.QueryEscape(c.Query("error", "missing code or state"))
		return c.Redirect(h.appURL+"/login?error="+errorMsg, fiber.StatusFound)
	}

	// Validate state against cookie
	cookieState := c.Cookies("oauth_state")
	if cookieState == "" || cookieState != state {
		return c.Redirect(h.appURL+"/login?error=invalid_state", fiber.StatusFound)
	}

	// Clear the state cookie
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   -1,
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
	})

	// Build the callback URL that matches what was sent in the initial redirect.
	// Always https — same as the initial redirect.
	callbackURL := fmt.Sprintf("https://%s%s", c.Hostname(), c.Path())

	oneTimeCode, err := h.svc.HandleOAuthCallbackWithURL(c.Context(), provider, code, callbackURL)
	if err != nil {
		return c.Redirect(h.appURL+"/login?error=oauth_failed", fiber.StatusFound)
	}

	return c.Redirect(h.appURL+"/auth/callback?code="+oneTimeCode, fiber.StatusFound)
}

// ExchangeOAuthCode exchanges a one-time code for JWT tokens.
func (h *AuthHandler) ExchangeOAuthCode(c *fiber.Ctx) error {
	var req exchangeCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code is required")
	}

	tokens, err := h.svc.ExchangeOAuthCode(c.Context(), req.Code)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	return c.JSON(tokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "bearer",
		ExpiresIn:    tokens.ExpiresIn,
	})
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

	tokens, err := h.svc.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	// Blacklist the old refresh token AFTER successful validation to prevent
	// DoS via premature revocation on validation failure.
	if h.blacklist != nil && req.RefreshToken != "" {
		h.blacklist.Revoke(req.RefreshToken, time.Now().Add(7*24*time.Hour))
	}

	return c.JSON(tokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "bearer",
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// Logout revokes the current access token so it can no longer be used.
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	if h.blacklist == nil {
		return c.JSON(fiber.Map{"message": "logged out"})
	}

	authHeader := c.Get("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 {
		// Revoke access token — expires in 1 hour max
		h.blacklist.Revoke(parts[1], time.Now().Add(1*time.Hour))
	}

	// Also revoke refresh token if provided in body
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&body); err == nil && body.RefreshToken != "" {
		h.blacklist.Revoke(body.RefreshToken, time.Now().Add(7*24*time.Hour))
	}

	return c.JSON(fiber.Map{"message": "logged out"})
}

// UpdateOnboarding updates the current user's onboarding progress.
// PUT /api/v1/auth/onboarding
func (h *AuthHandler) UpdateOnboarding(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var req struct {
		Step       int                    `json:"step"`
		Completed  bool                   `json:"completed"`
		SurveyData map[string]interface{} `json:"survey_data,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	h.svc.UpdateOnboarding(c.Context(), userID, req.Step, req.Completed)

	// Track onboarding events
	if req.Completed {
		h.trackEvent(c, entities.EventOnboardingDone, req.SurveyData)
	} else {
		h.trackEvent(c, entities.EventOnboardingStep, map[string]interface{}{"step": req.Step})
	}

	return c.JSON(fiber.Map{"message": "onboarding updated"})
}

// GetMe returns the authenticated user's profile with onboarding status.
// GET /api/v1/auth/me
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	user, err := h.svc.GetUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	return c.JSON(user)
}

// trackEvent fires a user event in the background for the currently authenticated user.
func (h *AuthHandler) trackEvent(c *fiber.Ctx, eventType string, props map[string]interface{}) {
	if h.eventRepo == nil {
		return
	}
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return
	}
	h.trackEventForUser(c, userID, eventType, props)
}

// trackEventForUser fires a user event for a specific user.
func (h *AuthHandler) trackEventForUser(c *fiber.Ctx, userID, eventType string, props map[string]interface{}) {
	if h.eventRepo == nil {
		return
	}
	if props == nil {
		props = make(map[string]interface{})
	}
	go h.eventRepo.Track(context.Background(), &entities.UserEvent{
		UserID:     userID,
		EventType:  eventType,
		Properties: props,
		IPAddress:  c.IP(),
		UserAgent:  c.Get("User-Agent"),
	})
}
