package jwt

import (
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/golang-jwt/jwt/v5"
)

// TokenType distinguishes access tokens from refresh tokens.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// Claims holds the JWT payload.
type Claims struct {
	jwt.RegisteredClaims
	Email         string        `json:"email"`
	Name          string        `json:"name"`
	Role          entities.Role `json:"role"`
	ProjectID     string        `json:"project_id,omitempty"`
	EmailVerified bool          `json:"email_verified"`
	AccountID     string        `json:"account_id,omitempty"` // owner's user_id (set for team members)
	MemberID      string        `json:"member_id,omitempty"`  // team_members.id (set for team members)
	Type          TokenType     `json:"type,omitempty"`        // "access" or "refresh"
}

// GenerateToken creates a signed JWT for the given user with a specific token type.
func GenerateToken(secret string, user *entities.User, expiry time.Duration, tokenType ...TokenType) (string, error) {
	tt := TokenTypeAccess
	if len(tokenType) > 0 {
		tt = tokenType[0]
	}

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    "zenith",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		},
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		ProjectID:     user.ProjectID,
		EmailVerified: user.EmailVerified,
		Type:          tt,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// TeamMemberOverrides holds optional overrides for team member token generation.
type TeamMemberOverrides struct {
	AccountID string        // owner's user_id
	MemberID  string        // team_members.id
	Role      entities.Role // team member's role
}

// GenerateTeamMemberToken creates a signed JWT for a team member.
// The Subject is the member's own user_id, but AccountID points to the owner.
func GenerateTeamMemberToken(secret string, user *entities.User, expiry time.Duration, overrides TeamMemberOverrides, tokenType ...TokenType) (string, error) {
	tt := TokenTypeAccess
	if len(tokenType) > 0 {
		tt = tokenType[0]
	}

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    "zenith",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		},
		Email:         user.Email,
		Name:          user.Name,
		Role:          overrides.Role,
		ProjectID:     user.ProjectID,
		EmailVerified: user.EmailVerified,
		AccountID:     overrides.AccountID,
		MemberID:      overrides.MemberID,
		Type:          tt,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken validates and parses a JWT token with issuer validation.
func ParseToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	parser := jwt.NewParser(
		jwt.WithIssuedAt(),
		jwt.WithIssuer("zenith"),
		jwt.WithValidMethods([]string{"HS256"}),
	)
	token, err := parser.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid or expired token")
	}
	return claims, nil
}

// ParseTokenWithType validates a JWT and ensures it matches the expected token type.
// This prevents access tokens from being used as refresh tokens and vice versa.
func ParseTokenWithType(secret, tokenString string, expectedType TokenType) (*Claims, error) {
	claims, err := ParseToken(secret, tokenString)
	if err != nil {
		return nil, err
	}
	// Enforce type claim if present (backward compat: tokens without type are accepted)
	if claims.Type != "" && claims.Type != expectedType {
		return nil, fmt.Errorf("invalid token type: expected %s", expectedType)
	}
	return claims, nil
}
