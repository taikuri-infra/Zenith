package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dotechhq/zenith/services/api/internal/config"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/dotechhq/zenith/services/api/internal/middleware"
	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	cfg := config.Load()

	app := fiber.New(fiber.Config{
		AppName:      "Zenith API",
		ServerHeader: "Zenith",
		ErrorHandler: handlers.ErrorHandler,
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${method} | ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Request-ID",
	}))
	app.Use(middleware.RequestContext())

	setupRoutes(app, cfg)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		log.Printf("Zenith API %s starting on %s", Version, addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server stopped")
}

func setupRoutes(app *fiber.App, cfg *config.Config) {
	app.Get("/health", handlers.HealthCheck(Version, BuildTime, GitCommit))
	app.Get("/ready", handlers.ReadinessCheck())

	// K8s client (in-memory for dev, real client for production)
	k8sClient := k8s.NewMemoryClient()

	// Handlers
	projectHandler := handlers.NewProjectHandler(k8sClient)

	api := app.Group("/api/v1")
	api.Get("/version", handlers.VersionInfo(Version, BuildTime, GitCommit))

	// Protected routes
	protected := api.Group("", middleware.RequireAuth(cfg.JWTSecret))

	// Projects
	projects := protected.Group("/projects")
	projects.Post("/", projectHandler.Create)
	projects.Get("/", projectHandler.List)
	projects.Get("/:id", projectHandler.Get)
	projects.Put("/:id", projectHandler.Update)
	projects.Delete("/:id", middleware.RequireRole(models.RoleOwner), projectHandler.Delete)
}
