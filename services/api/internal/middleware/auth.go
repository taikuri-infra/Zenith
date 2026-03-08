package middleware

import (
	"crypto/subtle"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	zenithJWT "github.com/dotechhq/zenith/services/api/pkg/jwt"
	"github.com/gofiber/fiber/v2"
)

// Claims is an alias for pkg/jwt.Claims. Kept here for backward compatibility
// with code that references middleware.Claims.
type Claims = zenithJWT.Claims

// JWTAuth validates JWT tokens from the Authorization header
func JWTAuth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid authorization format")
		}

		claims, err := zenithJWT.ParseToken(secret, parts[1])
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		// AccountID trick: team members use the owner's ID for resource lookups
		if claims.AccountID != "" {
			c.Locals("user_id", claims.AccountID)
			c.Locals("member_id", claims.MemberID)
		} else {
			c.Locals("user_id", claims.Subject)
		}
		c.Locals("email", claims.Email)
		c.Locals("name", claims.Name)
		c.Locals("role", claims.Role)
		c.Locals("project_id", claims.ProjectID)

		return c.Next()
	}
}

// APIKeyAuth validates API keys from the X-API-Key header
func APIKeyAuth(validateKey func(key string) (*entities.APIKey, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Next() // Let JWT middleware handle it
		}

		key, err := validateKey(apiKey)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid API key")
		}

		c.Locals("user_id", key.UserID)
		c.Locals("project_id", key.ProjectID)
		c.Locals("api_key", key)
		c.Locals("role", entities.RoleDeveloper) // API keys get developer role by default

		return c.Next()
	}
}

// RequireAuth ensures the request is authenticated (either JWT or API key)
func RequireAuth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if already authenticated (e.g., by API key middleware)
		if c.Locals("user_id") != nil {
			return c.Next()
		}

		// Try JWT auth
		return JWTAuth(secret)(c)
	}
}

// RequireRole ensures the user has at least the given role
func RequireRole(minRole entities.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(entities.Role)
		if !ok {
			return fiber.NewError(fiber.StatusForbidden, "no role assigned")
		}

		roleOrder := map[entities.Role]int{
			entities.RoleOwner:     4,
			entities.RoleAdmin:     3,
			entities.RoleDeveloper: 2,
			entities.RoleViewer:    1,
			entities.RoleCustomer:  0,
		}

		if roleOrder[role] < roleOrder[minRole] {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}

		return c.Next()
	}
}

// RequireScope checks if the API key has the required scope
func RequireScope(scope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key, ok := c.Locals("api_key").(*entities.APIKey)
		if !ok {
			// Not using API key, skip scope check
			return c.Next()
		}

		if !key.HasScope(scope) {
			return fiber.NewError(fiber.StatusForbidden, "API key missing required scope: "+scope)
		}

		return c.Next()
	}
}

// RequireInternalSecret validates the X-Internal-Secret header using constant-time comparison.
func RequireInternalSecret(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provided := c.Get("X-Internal-Secret")
		if provided == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing internal secret")
		}
		if !ConstantTimeCompare(provided, secret) {
			return fiber.NewError(fiber.StatusForbidden, "invalid internal secret")
		}
		return c.Next()
	}
}

// GenerateToken creates a JWT token for a user. Delegates to pkg/jwt.
func GenerateToken(secret string, user *entities.User, expiry time.Duration) (string, error) {
	return zenithJWT.GenerateToken(secret, user, expiry)
}

// ParseToken validates a JWT string and returns its claims. Delegates to pkg/jwt.
func ParseToken(secret, tokenString string) (*Claims, error) {
	return zenithJWT.ParseToken(secret, tokenString)
}

// ConstantTimeCompare provides timing-safe string comparison
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
