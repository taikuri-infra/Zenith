package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dotechhq/zenith/services/api/internal/capi"
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

	// CAPI client and admin store
	capiClient := capi.NewClient(k8sClient)
	adminStore := capi.NewMemoryStore()

	// Handlers
	projectHandler := handlers.NewProjectHandler(k8sClient)
	appHandler := handlers.NewAppHandler(k8sClient)
	dbHandler := handlers.NewDatabaseHandler(k8sClient)
	storageHandler := handlers.NewStorageHandler(k8sClient)
	adminHandler := handlers.NewAdminHandler(k8sClient, capiClient, adminStore)

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

	// Apps (nested under projects)
	apps := protected.Group("/projects/:id/apps")
	apps.Post("/", appHandler.Create)
	apps.Get("/", appHandler.List)
	apps.Get("/:name", appHandler.Get)
	apps.Put("/:name", appHandler.Update)
	apps.Delete("/:name", appHandler.Delete)
	apps.Post("/:name/redeploy", appHandler.Redeploy)

	// Databases (nested under projects)
	databases := protected.Group("/projects/:id/databases")
	databases.Post("/", dbHandler.Create)
	databases.Get("/", dbHandler.List)
	databases.Get("/:name", dbHandler.Get)
	databases.Delete("/:name", dbHandler.Delete)
	databases.Get("/:name/backups", dbHandler.ListBackups)
	databases.Post("/:name/backups", dbHandler.CreateBackup)

	// Storage (nested under projects)
	storage := protected.Group("/projects/:id/storage")
	storage.Post("/", storageHandler.Create)
	storage.Get("/", storageHandler.List)
	storage.Get("/:name", storageHandler.Get)
	storage.Delete("/:name", storageHandler.Delete)

	// Admin routes (Mission Control) - require admin role
	admin := protected.Group("/admin", middleware.RequireRole(models.RoleAdmin))

	// Dashboard
	admin.Get("/dashboard/stats", adminHandler.GetDashboardStats)

	// Cluster management (CAPI)
	admin.Get("/clusters", adminHandler.ListClusters)
	admin.Post("/clusters", adminHandler.CreateCluster)
	admin.Get("/clusters/:name", adminHandler.GetCluster)
	admin.Delete("/clusters/:name", adminHandler.DeleteCluster)
	admin.Post("/clusters/:name/upgrade", adminHandler.UpgradeCluster)

	// Tenant management
	admin.Get("/tenants", adminHandler.ListTenants)
	admin.Get("/tenants/:id", adminHandler.GetTenant)
	admin.Post("/tenants/:id/suspend", adminHandler.SuspendTenant)

	// Module management
	admin.Get("/modules", adminHandler.ListModules)
	admin.Post("/modules/update-all", adminHandler.UpdateAllModules)
	admin.Post("/modules/:name/install", adminHandler.InstallModule)
	admin.Post("/modules/:name/uninstall", adminHandler.UninstallModule)
	admin.Post("/modules/:name/update", adminHandler.UpdateModule)

	// Audit log
	admin.Get("/audit", adminHandler.ListAuditLog)

	// Platform updates
	admin.Get("/updates/check", adminHandler.CheckUpdates)
	admin.Post("/updates/apply", adminHandler.ApplyUpdate)
	admin.Get("/updates/history", adminHandler.ListUpdateHistory)

	// Infrastructure
	admin.Get("/infrastructure", adminHandler.GetInfraOverview)

	// Platform state
	admin.Get("/state", adminHandler.GetPlatformState)
	admin.Get("/state/export", adminHandler.ExportState)

	// Settings
	admin.Get("/settings", adminHandler.GetSettings)
	admin.Put("/settings", adminHandler.UpdateSettings)
	admin.Patch("/settings", adminHandler.UpdateSettings)
}
