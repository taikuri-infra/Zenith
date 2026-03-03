package handlers

import (
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	svc    *services.AuthService
	appURL string // frontend URL for OAuth redirect
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
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
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
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

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type messageResponse struct {
	Message string `json:"message"`
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

	tokens, err := h.svc.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		// Return 403 for email verification errors, 401 for auth errors
		if err.Error() == "please verify your email before logging in" {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
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

	result, err := h.svc.Register(c.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
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
		errorMsg := c.Query("error", "missing code or state")
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

	return c.JSON(tokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "bearer",
		ExpiresIn:    tokens.ExpiresIn,
	})
}
