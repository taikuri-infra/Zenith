package handlers

import (
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gofiber/fiber/v2"
)

var startTime = time.Now()

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

type ReadinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

type VersionResponse struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

func HealthCheck(version string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(HealthResponse{
			Status:  "healthy",
			Version: version,
			Uptime:  time.Since(startTime).String(),
		})
	}
}

func ReadinessCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		checks := map[string]string{
			"server": "ready",
		}

		status := "ready"
		if pool != nil {
			if err := pool.Ping(c.Context()); err != nil {
				checks["database"] = "unhealthy"
				status = "not_ready"
			} else {
				checks["database"] = "ready"
			}
		}

		code := fiber.StatusOK
		if status != "ready" {
			code = fiber.StatusServiceUnavailable
		}

		return c.Status(code).JSON(ReadinessResponse{
			Status: status,
			Checks: checks,
		})
	}
}

func VersionInfo(version, buildTime string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(VersionResponse{
			Version:   version,
			BuildTime: buildTime,
			GoVersion: runtime.Version(),
		})
	}
}
