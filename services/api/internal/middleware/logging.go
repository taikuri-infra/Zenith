package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

// StructuredLogger replaces the default Fiber logger with JSON structured logging.
// It includes request_id, method, path, status, latency, and client IP.
func StructuredLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()

		attrs := []slog.Attr{
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("ip", c.IP()),
		}

		if rid, ok := c.Locals("request_id").(string); ok && rid != "" {
			attrs = append(attrs, slog.String("request_id", rid))
		}

		if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
			attrs = append(attrs, slog.String("user_id", userID))
		}

		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		args := make([]any, len(attrs))
		for i, a := range attrs {
			args[i] = a
		}

		switch {
		case status >= 500:
			slog.Error("request", args...)
		case status >= 400:
			slog.Warn("request", args...)
		default:
			slog.Info("request", args...)
		}

		return err
	}
}
