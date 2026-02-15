package middleware

import (
	"github.com/gofiber/fiber/v2"
)

func RequestContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("request_id", c.GetRespHeader("X-Request-Id"))
		return c.Next()
	}
}
