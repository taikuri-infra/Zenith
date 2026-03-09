package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RequireAdminPermission checks if the authenticated admin user has the required
// permission group. Owner role always has all permissions.
func RequireAdminPermission(pool *pgxpool.Pool, permissionGroup string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		// Owner always has all permissions
		if role == "owner" {
			return c.Next()
		}

		userID, ok := c.Locals("user_id").(string)
		if !ok || userID == "" {
			return fiber.NewError(fiber.StatusForbidden, "no user context")
		}

		if pool == nil {
			// No DB — fallback to role-based check
			return c.Next()
		}

		var permissions []string
		err := pool.QueryRow(c.Context(),
			"SELECT permissions FROM admin_roles WHERE user_id = $1", userID,
		).Scan(&permissions)
		if err != nil {
			// No admin_roles entry — check base role
			if role == "admin" {
				return c.Next()
			}
			return fiber.NewError(fiber.StatusForbidden, "insufficient admin permissions")
		}

		for _, p := range permissions {
			if p == permissionGroup {
				return c.Next()
			}
		}

		return fiber.NewError(fiber.StatusForbidden, "missing permission: "+permissionGroup)
	}
}
