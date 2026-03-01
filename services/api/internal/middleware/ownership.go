package middleware

import (
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// RequireAppOwnership validates that the authenticated user owns the app
// identified by the :appId URL parameter. On success, stores the app in
// c.Locals("app") so downstream handlers can use it without re-fetching.
// Returns 404 (not 403) to avoid leaking resource existence to other users.
func RequireAppOwnership(appRepo ports.AppRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		appID := c.Params("appId")
		if appID == "" {
			return c.Next()
		}

		userID, _ := c.Locals("user_id").(string)
		if userID == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}

		app, err := appRepo.GetApp(c.Context(), appID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "app not found")
		}

		if app.UserID != userID {
			return fiber.NewError(fiber.StatusNotFound, "app not found")
		}

		c.Locals("app", app)
		return c.Next()
	}
}
