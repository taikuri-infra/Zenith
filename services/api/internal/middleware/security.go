package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders adds OWASP-recommended security headers to every response.
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Prevent MIME-type sniffing
		c.Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Set("X-Frame-Options", "DENY")

		// Enable XSS filter (legacy browsers)
		c.Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Prevent caching of API responses (sensitive data)
		c.Set("Cache-Control", "no-store")
		c.Set("Pragma", "no-cache")

		// Remove Server header (set by Fiber)
		c.Set("Server", "")

		// Content Security Policy (API only serves JSON)
		c.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Permissions Policy (disable browser features)
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		// HSTS — enforce HTTPS for 1 year including subdomains
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		return c.Next()
	}
}
