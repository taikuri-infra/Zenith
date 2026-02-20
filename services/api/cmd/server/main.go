package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/capi"
	"github.com/dotechhq/zenith/services/api/internal/cluster"
	"github.com/dotechhq/zenith/services/api/internal/config"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/dotechhq/zenith/services/api/internal/middleware"
	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/dotechhq/zenith/services/api/internal/store/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
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
	ctx := context.Background()

	// Database (optional — falls back to in-memory stores when DATABASE_URL is empty)
	var pool *pgxpool.Pool
	var userRepo store.UserRepository
	var adminRepo store.AdminRepository
	var customerRepo store.CustomerRepository

	if cfg.DatabaseURL != "" {
		log.Println("Connecting to PostgreSQL...")
		if err := store.RunMigrations(cfg.DatabaseURL, migrations.FS); err != nil {
			log.Fatalf("Database migrations failed: %v", err)
		}
		log.Println("Migrations applied")

		var err error
		pool, err = store.NewPostgresPool(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Database connection failed: %v", err)
		}
		log.Println("Connected to PostgreSQL")

		userRepo = store.NewPostgresUserRepository(pool)
		adminRepo = store.NewPostgresAdminRepository(pool)
		customerRepo = store.NewPostgresCustomerRepository(pool)
	} else {
		log.Println("No DATABASE_URL set — using in-memory stores")
		userRepo = store.NewMemoryUserRepository()
		adminRepo = store.NewMemoryAdminRepository()
		customerRepo = store.NewMemoryCustomerRepository()
	}

	// Seed admin user
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		if _, err := userRepo.Create(ctx, cfg.AdminEmail, cfg.AdminPassword, "Admin", models.RoleOwner); err != nil {
			log.Printf("Admin seed skipped: %v", err)
		} else {
			log.Printf("Admin user seeded: %s", cfg.AdminEmail)
		}
	}

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

	provisioner := setupRoutes(app, cfg, userRepo, adminRepo, customerRepo, pool)

	// Start cluster sync if provisioner is available
	if provisioner != nil {
		provisioner.StartSync(30 * time.Second)
		log.Println("Cluster provisioner sync started (30s interval)")
	}

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
	if provisioner != nil {
		provisioner.Stop()
		log.Println("Cluster provisioner stopped")
	}
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	if pool != nil {
		pool.Close()
		log.Println("Database pool closed")
	}
	log.Println("Server stopped")
}

func setupRoutes(app *fiber.App, cfg *config.Config, userRepo store.UserRepository, adminRepo store.AdminRepository, customerRepo store.CustomerRepository, pool *pgxpool.Pool) *cluster.Provisioner {
	app.Get("/health", handlers.HealthCheck(Version, BuildTime, GitCommit))
	app.Get("/ready", handlers.ReadinessCheck(pool))

	// K8s client (in-memory for dev, real client for production)
	k8sClient := k8s.NewMemoryClient()

	// CAPI client
	capiClient := capi.NewClient(k8sClient)

	// Cluster provisioner
	provisioner := cluster.NewProvisioner(capiClient, customerRepo, adminRepo)

	// Handlers
	projectHandler := handlers.NewProjectHandler(k8sClient)
	appHandler := handlers.NewAppHandler(k8sClient)
	dbHandler := handlers.NewDatabaseHandler(k8sClient)
	storageHandler := handlers.NewStorageHandler(k8sClient)
	adminHandler := handlers.NewAdminHandler(k8sClient, capiClient, adminRepo)
	customerHandler := handlers.NewCustomerHandler(customerRepo, adminRepo, provisioner)
	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret)

	api := app.Group("/api/v1")
	api.Get("/version", handlers.VersionInfo(Version, BuildTime, GitCommit))

	// Public auth routes (no token required)
	authRoutes := api.Group("/auth")
	authRoutes.Post("/login", authHandler.Login)
	authRoutes.Post("/register", authHandler.Register)
	authRoutes.Post("/refresh", authHandler.Refresh)

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

	// Customer management
	admin.Post("/customers", customerHandler.CreateCustomer)
	admin.Get("/customers", customerHandler.ListCustomers)
	admin.Get("/customers/stats", customerHandler.GetCustomerStats) // before :id
	admin.Get("/customers/:id", customerHandler.GetCustomer)
	admin.Put("/customers/:id", customerHandler.UpdateCustomer)
	admin.Post("/customers/:id/suspend", customerHandler.SuspendCustomer)
	admin.Post("/customers/:id/activate", customerHandler.ActivateCustomer)
	admin.Delete("/customers/:id", customerHandler.DeleteCustomer)
	admin.Get("/customers/:id/cluster", customerHandler.GetCustomerCluster)
	admin.Post("/customers/:id/cluster/scale", customerHandler.ScaleCluster)
	admin.Post("/customers/:id/cluster/upgrade", customerHandler.UpgradeCluster)

	// Plan management
	admin.Get("/plans", customerHandler.ListPlans)
	admin.Post("/plans", customerHandler.CreatePlan)
	admin.Put("/plans/:id", customerHandler.UpdatePlan)

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

	return provisioner
}
