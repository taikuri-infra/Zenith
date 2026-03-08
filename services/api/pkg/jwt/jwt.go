package jwt

import (
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/golang-jwt/jwt/v5"
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
}

// GenerateToken creates a signed JWT for the given user.
func GenerateToken(secret string, user *entities.User, expiry time.Duration) (string, error) {
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
func GenerateTeamMemberToken(secret string, user *entities.User, expiry time.Duration, overrides TeamMemberOverrides) (string, error) {
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
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken validates and parses a JWT token.
func ParseToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
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
