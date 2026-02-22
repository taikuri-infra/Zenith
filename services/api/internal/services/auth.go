package services

import (
	"context"
	"fmt"
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
	users     ports.UserRepository
	jwtSecret string
}

// NewAuthService creates a new AuthService.
func NewAuthService(users ports.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
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
