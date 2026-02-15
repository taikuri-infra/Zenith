package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

var startTime = time.Now()

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
	Uptime    string `json:"uptime"`
}

type ReadinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

type VersionResponse struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
	GoVersion string `json:"go_version"`
}

func HealthCheck(version, buildTime, gitCommit string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(HealthResponse{
			Status:    "healthy",
			Version:   version,
			BuildTime: buildTime,
			GitCommit: gitCommit,
			Uptime:    time.Since(startTime).String(),
		})
	}
}

func ReadinessCheck() fiber.Handler {
	return func(c *fiber.Ctx) error {
		checks := map[string]string{
			"server": "ready",
		}

		return c.JSON(ReadinessResponse{
			Status: "ready",
			Checks: checks,
		})
	}
}

func VersionInfo(version, buildTime, gitCommit string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(VersionResponse{
			Version:   version,
			BuildTime: buildTime,
			GitCommit: gitCommit,
			GoVersion: "go1.26",
		})
	}
}
