package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/middleware"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

const (
	AccessTokenExpiry  = 1 * time.Hour
	RefreshTokenExpiry = 7 * 24 * time.Hour
)

// TokenPair holds issued JWT tokens.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// AuthService handles authentication business logic.
type AuthService struct {
	users          ports.UserRepository
	jwtSecret      string
	googleClientID string
}

// NewAuthService creates a new AuthService.
func NewAuthService(users ports.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
}

// SetGoogleClientID configures the Google OAuth client ID for token verification.
func (s *AuthService) SetGoogleClientID(clientID string) {
	s.googleClientID = clientID
}

// GoogleClientID returns the configured Google client ID.
func (s *AuthService) GoogleClientID() string {
	return s.googleClientID
}

// Login validates credentials and returns a token pair.
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil || !s.users.CheckPassword(user, password) {
		return nil, fmt.Errorf("invalid email or password")
	}
	return s.issueTokens(&user.User)
}

// Register creates a new user and returns a token pair.
// The first user gets owner role; subsequent users get developer.
func (s *AuthService) Register(ctx context.Context, email, password, name string) (*TokenPair, error) {
	role := entities.RoleDeveloper
	count, err := s.users.Count(ctx)
	if err == nil && count == 0 {
		role = entities.RoleOwner
	}

	user, err := s.users.Create(ctx, email, password, name, role)
	if err != nil {
		return nil, err
	}

	return s.issueTokens(user)
}

// Refresh validates a refresh token and returns a new token pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := middleware.ParseToken(s.jwtSecret, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	user, err := s.users.GetByID(ctx, claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.issueTokens(&user.User)
}

// GoogleLogin verifies a Google ID token and creates/logs in the user.
func (s *AuthService) GoogleLogin(ctx context.Context, idToken string) (*TokenPair, error) {
	if s.googleClientID == "" {
		return nil, fmt.Errorf("Google OAuth is not configured")
	}

	// Verify token with Google's tokeninfo endpoint
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify Google token")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid Google ID token")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Google token response")
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Aud           string `json:"aud"`
	}
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse Google token claims")
	}

	// Verify audience matches our client ID
	if claims.Aud != s.googleClientID {
		return nil, fmt.Errorf("token audience mismatch")
	}

	if claims.Email == "" || claims.EmailVerified != "true" {
		return nil, fmt.Errorf("email not verified")
	}

	// Check if user exists
	existing, err := s.users.GetByEmail(ctx, claims.Email)
	if err == nil {
		return s.issueTokens(&existing.User)
	}

	// Create new user with random password (OAuth users don't use passwords)
	randBytes := make([]byte, 32)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random password")
	}
	randomPassword := hex.EncodeToString(randBytes)

	name := claims.Name
	if name == "" {
		name = claims.Email
	}

	role := entities.RoleDeveloper
	count, err := s.users.Count(ctx)
	if err == nil && count == 0 {
		role = entities.RoleOwner
	}

	user, err := s.users.Create(ctx, claims.Email, randomPassword, name, role)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return s.issueTokens(user)
}

func (s *AuthService) issueTokens(user *entities.User) (*TokenPair, error) {
	accessToken, err := middleware.GenerateToken(s.jwtSecret, user, AccessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token")
	}

	refreshToken, err := middleware.GenerateToken(s.jwtSecret, user, RefreshTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token")
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(AccessTokenExpiry.Seconds()),
	}, nil
}
