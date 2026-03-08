package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// RequestContext reads the request ID from the requestid middleware's Locals
// and stores it under "request_id" for use in structured logging and correlation.
func RequestContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if rid := c.Locals("requestid"); rid != nil {
			c.Locals("request_id", rid)
		}
		return c.Next()
	}
}
