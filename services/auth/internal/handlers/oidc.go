package handlers

import (
	"time"

	"github.com/dotechhq/zenith/services/auth/internal/crypto"
	"github.com/dotechhq/zenith/services/auth/internal/models"
	"github.com/dotechhq/zenith/services/auth/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type OIDCHandler struct {
	store     storage.Store
	jwtSecret string
	issuer    string
}

func NewOIDCHandler(store storage.Store, jwtSecret, issuer string) *OIDCHandler {
	return &OIDCHandler{store: store, jwtSecret: jwtSecret, issuer: issuer}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	ClientID string `json:"client_id"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// Discovery returns the OIDC discovery document
func (h *OIDCHandler) Discovery(c *fiber.Ctx) error {
	realmID := c.Params("realm")
	baseURL := h.issuer + "/realms/" + realmID

	return c.JSON(fiber.Map{
		"issuer":                 baseURL,
		"authorization_endpoint": baseURL + "/protocol/openid-connect/auth",
		"token_endpoint":         baseURL + "/protocol/openid-connect/token",
		"userinfo_endpoint":      baseURL + "/protocol/openid-connect/userinfo",
		"jwks_uri":               baseURL + "/protocol/openid-connect/certs",
		"end_session_endpoint":   baseURL + "/protocol/openid-connect/logout",
		"grant_types_supported": []string{
			"authorization_code",
			"refresh_token",
			"client_credentials",
		},
		"response_types_supported": []string{"code", "id_token", "token"},
		"subject_types_supported":  []string{"public"},
		"id_token_signing_alg_values_supported": []string{"HS256"},
		"scopes_supported": []string{"openid", "profile", "email"},
		"token_endpoint_auth_methods_supported": []string{
			"client_secret_basic", "client_secret_post",
		},
	})
}

// Token handles the token endpoint (login)
func (h *OIDCHandler) Token(c *fiber.Ctx) error {
	realmID := c.Params("realm")

	_, err := h.store.GetRealm(realmID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "realm not found")
	}

	grantType := c.FormValue("grant_type")

	switch grantType {
	case "password":
		return h.handlePasswordGrant(c, realmID)
	case "refresh_token":
		return h.handleRefreshGrant(c, realmID)
	case "client_credentials":
		return h.handleClientCredentials(c, realmID)
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unsupported grant_type")
	}
}

func (h *OIDCHandler) handlePasswordGrant(c *fiber.Ctx, realmID string) error {
	email := c.FormValue("username")
	password := c.FormValue("password")

	if email == "" || password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "username and password required")
	}

	user, err := h.store.GetUserByEmail(realmID, email)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	if user.Blocked {
		return fiber.NewError(fiber.StatusForbidden, "account is blocked")
	}

	if !crypto.CheckPassword(password, user.PasswordHash) {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	return h.issueTokens(c, user, realmID)
}

func (h *OIDCHandler) handleRefreshGrant(c *fiber.Ctx, realmID string) error {
	refreshToken := c.FormValue("refresh_token")
	if refreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh_token required")
	}

	// Validate refresh token (simple JWT validation)
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid refresh token")
	}

	user, err := h.store.GetUser(realmID, claims.Subject)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}

	return h.issueTokens(c, user, realmID)
}

func (h *OIDCHandler) handleClientCredentials(c *fiber.Ctx, realmID string) error {
	clientID := c.FormValue("client_id")
	clientSecret := c.FormValue("client_secret")

	if clientID == "" || clientSecret == "" {
		return fiber.NewError(fiber.StatusBadRequest, "client_id and client_secret required")
	}

	client, err := h.store.GetClient(realmID, clientID)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid client")
	}

	if client.Type != "confidential" {
		return fiber.NewError(fiber.StatusBadRequest, "client_credentials only for confidential clients")
	}

	if client.Secret != clientSecret {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid client secret")
	}

	accessToken, err := h.generateAccessToken(clientID, realmID, client.Scopes, 1*time.Hour)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(models.TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		Scope:       "openid",
	})
}

func (h *OIDCHandler) issueTokens(c *fiber.Ctx, user *models.User, realmID string) error {
	accessToken, err := h.generateAccessToken(user.ID, realmID, user.Roles, 15*time.Minute)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate access token")
	}

	refreshToken, err := h.generateRefreshToken(user.ID, realmID, 7*24*time.Hour)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate refresh token")
	}

	idToken, err := h.generateIDToken(user, realmID, 15*time.Minute)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate ID token")
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	h.store.UpdateUser(user)

	return c.JSON(models.TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		Scope:        "openid profile email",
	})
}

// UserInfo returns user claims
func (h *OIDCHandler) UserInfo(c *fiber.Ctx) error {
	realmID := c.Params("realm")
	userID := c.Locals("user_id").(string)

	user, err := h.store.GetUser(realmID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	return c.JSON(models.UserInfo{
		Sub:           user.ID,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Name:          user.Name,
	})
}

// Register creates a new user
func (h *OIDCHandler) Register(c *fiber.Ctx) error {
	realmID := c.Params("realm")

	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password required")
	}

	if len(req.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}

	user := &models.User{
		ID:           crypto.GenerateID(),
		RealmID:      realmID,
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: hash,
		Roles:        []string{"user"},
	}

	if err := h.store.CreateUser(user); err != nil {
		return fiber.NewError(fiber.StatusConflict, "user already exists")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
}

func (h *OIDCHandler) generateAccessToken(sub, realmID string, roles []string, expiry time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":      sub,
		"iss":      h.issuer + "/realms/" + realmID,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(expiry).Unix(),
		"realm_id": realmID,
		"roles":    roles,
		"typ":      "Bearer",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func (h *OIDCHandler) generateRefreshToken(sub, realmID string, expiry time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   sub,
		Issuer:    h.issuer + "/realms/" + realmID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func (h *OIDCHandler) generateIDToken(user *models.User, realmID string, expiry time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":            user.ID,
		"iss":            h.issuer + "/realms/" + realmID,
		"iat":            time.Now().Unix(),
		"exp":            time.Now().Add(expiry).Unix(),
		"email":          user.Email,
		"email_verified": user.EmailVerified,
		"name":           user.Name,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}
