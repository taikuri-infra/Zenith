package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dotechhq/zenith/services/api/docs"
	"github.com/dotechhq/zenith/services/api/internal/adapters/capiclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/keycloakclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/adapters/postgres"
	"github.com/dotechhq/zenith/services/api/internal/adapters/postgres/migrations"
	"github.com/dotechhq/zenith/services/api/internal/adapters/s3client"
	stripeClient "github.com/dotechhq/zenith/services/api/internal/adapters/stripeclient"
	"github.com/dotechhq/zenith/services/api/internal/services/autoscale"
	"github.com/dotechhq/zenith/services/api/internal/services/cluster"
	"github.com/dotechhq/zenith/services/api/internal/config"
	"github.com/dotechhq/zenith/services/api/internal/services/deploy"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/adapters/hetznerclient"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/middleware"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/dotechhq/zenith/services/api/internal/telemetry"
	zenithTemporal "github.com/dotechhq/zenith/services/api/internal/services/temporal"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	temporalWorker "go.temporal.io/sdk/worker"
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
	var userRepo ports.UserRepository
	var adminRepo ports.AdminRepository
	var customerRepo ports.CustomerRepository
	var meteringRepo ports.MeteringRepository
	var appRepo ports.AppRepository

	if cfg.DatabaseURL != "" {
		log.Println("Connecting to PostgreSQL...")
		if err := postgres.RunMigrations(cfg.DatabaseURL, migrations.FS); err != nil {
			log.Fatalf("Database migrations failed: %v", err)
		}
		log.Println("Migrations applied")

		var err error
		pool, err = postgres.NewPostgresPool(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Database connection failed: %v", err)
		}
		log.Println("Connected to PostgreSQL")

		userRepo = postgres.NewPostgresUserRepository(pool)
		adminRepo = postgres.NewPostgresAdminRepository(pool)
		customerRepo = postgres.NewPostgresCustomerRepository(pool)
		meteringRepo = postgres.NewPostgresMeteringRepository(pool)
		appRepo = postgres.NewPostgresAppRepository(pool)
	} else {
		log.Println("No DATABASE_URL set — using in-memory stores")
		userRepo = memory.NewMemoryUserRepository()
		adminRepo = memory.NewMemoryAdminRepository()
		customerRepo = memory.NewMemoryCustomerRepository()
		meteringRepo = memory.NewMemoryMeteringRepository()
		appRepo = memory.NewMemoryAppRepository()
	}

	// Seed admin user
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		if _, err := userRepo.Create(ctx, cfg.AdminEmail, cfg.AdminPassword, "Admin", entities.RoleOwner); err != nil {
			log.Printf("Admin seed skipped: %v", err)
		} else {
			log.Printf("Admin user seeded: %s", cfg.AdminEmail)
		}
	}

	log.Printf("Zenith mode: %s", cfg.Mode)
	if cfg.Mode == "standalone" {
		log.Println("Running in standalone mode — multi-tenant and billing features disabled")
	}

	// OpenTelemetry (opt-in: only when OTEL_EXPORTER_OTLP_ENDPOINT is set)
	if cfg.OTELEndpoint != "" {
		otelShutdown, err := telemetry.Init(ctx, telemetry.Config{
			ServiceName:    "zenith-api",
			ServiceVersion: Version,
			OTLPEndpoint:   cfg.OTELEndpoint,
			Environment:    cfg.Environment,
			Insecure:       cfg.OTELInsecure,
			SampleRate:     cfg.OTELSampleRate,
		})
		if err != nil {
			log.Printf("[WARN] OpenTelemetry init failed: %v (continuing without tracing)", err)
		} else {
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := otelShutdown(shutdownCtx); err != nil {
					log.Printf("[WARN] OpenTelemetry shutdown error: %v", err)
				}
			}()
			log.Printf("OpenTelemetry enabled → %s", cfg.OTELEndpoint)
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

	// OpenTelemetry middleware (only effective when OTel SDK is initialized)
	if cfg.OTELEndpoint != "" {
		app.Use(telemetry.Middleware(telemetry.MiddlewareConfig{
			SkipPaths: []string{"/health", "/ready"},
		}))
		log.Println("OpenTelemetry tracing middleware active")
	}

	docs.RegisterRoutes(app)
	provisioner, autoscaler, temporalW := setupRoutes(app, cfg, userRepo, adminRepo, customerRepo, meteringRepo, appRepo, pool)

	// Start cluster sync if provisioner is available
	if provisioner != nil {
		provisioner.StartSync(30 * time.Second)
		log.Println("Cluster provisioner sync started (30s interval)")
	}

	// Start Temporal worker if configured
	if temporalW != nil {
		if err := temporalW.Start(); err != nil {
			log.Fatalf("Failed to start Temporal worker: %v", err)
		}
		log.Println("[temporal] Worker started on queue: " + zenithTemporal.TaskQueue)
	}

	// Start autoscaler if enabled
	if autoscaler != nil {
		autoscaler.Start(time.Duration(cfg.AutoscalerInterval) * time.Second)
		log.Printf("Hetzner autoscaler started (%ds interval, min=%d, max=%d)",
			cfg.AutoscalerInterval, cfg.AutoscalerMinNodes, cfg.AutoscalerMaxNodes)
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
	if temporalW != nil {
		temporalW.Stop()
		log.Println("Temporal worker stopped")
	}
	if autoscaler != nil {
		autoscaler.Stop()
		log.Println("Hetzner autoscaler stopped")
	}
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

func setupRoutes(app *fiber.App, cfg *config.Config, userRepo ports.UserRepository, adminRepo ports.AdminRepository, customerRepo ports.CustomerRepository, meteringRepo ports.MeteringRepository, appRepo ports.AppRepository, pool *pgxpool.Pool) (*cluster.Provisioner, *autoscale.Autoscaler, temporalWorker.Worker) {
	app.Get("/health", handlers.HealthCheck(Version, BuildTime, GitCommit))
	app.Get("/ready", handlers.ReadinessCheck(pool))

	// K8s client (in-memory for dev, real client-go for production)
	var k8sClient k8sclient.Client
	if cfg.K8sMode == "real" {
		realClient, err := k8sclient.NewRealClient()
		if err != nil {
			log.Fatalf("failed to create real K8s client: %v", err)
		}
		k8sClient = realClient
		log.Println("[k8s] using real client-go connection")
	} else {
		k8sClient = k8sclient.NewMemoryClient()
		log.Println("[k8s] using in-memory client (dev mode)")
	}

	// CAPI client + provisioner (SaaS mode only)
	var capiClient *capiclient.Client
	var provisioner *cluster.Provisioner
	if cfg.Mode == "saas" {
		capiClient = capiclient.NewClient(k8sClient)
		provisioner = cluster.NewProvisioner(capiClient, customerRepo, adminRepo)
	}

	// Services
	adminSvc := services.NewAdminService(k8sClient, capiClient, adminRepo)
	authSvc := services.NewAuthService(userRepo, cfg.JWTSecret)

	var customerSvc *services.CustomerService
	var tw temporalWorker.Worker
	if cfg.Mode == "saas" {
		customerSvc = services.NewCustomerService(customerRepo, adminRepo, provisioner)

		// Temporal provisioning (opt-in: only when TEMPORAL_ENABLED=true)
		if cfg.TemporalEnabled {
			tc, err := zenithTemporal.NewClient(cfg.TemporalHost, cfg.TemporalNamespace)
			if err != nil {
				log.Printf("[temporal] WARN: failed to connect: %v (provisioning falls back to goroutine)", err)
			} else {
				// Build adapters for activities
				var keycloakAPI keycloakclient.KeycloakAPI
				if cfg.KeycloakURL != "" {
					keycloakAPI = keycloakclient.NewClient(cfg.KeycloakURL, cfg.KeycloakAdminUser, cfg.KeycloakAdminPassword)
				} else {
					keycloakAPI = keycloakclient.NewMemoryClient()
				}

				var s3API s3client.S3API
				if cfg.S3Endpoint != "" {
					s3API = s3client.NewClient(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Region)
				} else {
					s3API = s3client.NewMemoryClient()
				}

				activities := &zenithTemporal.Activities{
					K8s:        k8sClient,
					Keycloak:   keycloakAPI,
					S3:         s3API,
					AdminDSN:   cfg.CNPGAdminDSN,
					Customers:  customerRepo,
					Admin:      adminRepo,
					BaseDomain: cfg.BaseDomain,
				}

				tw = zenithTemporal.NewWorker(tc, activities)
				customerSvc.SetWorkflows(zenithTemporal.NewWorkflowClient(tc))
				log.Printf("[temporal] Connected to %s (namespace: %s)", cfg.TemporalHost, cfg.TemporalNamespace)
			}
		} else {
			log.Println("[temporal] Disabled (TEMPORAL_ENABLED=false)")
		}
	}

	// Handlers
	projectHandler := handlers.NewProjectHandler(k8sClient)
	appHandler := handlers.NewAppHandler(k8sClient)
	dbHandler := handlers.NewDatabaseHandler(k8sClient)
	storageHandler := handlers.NewStorageHandler(k8sClient)
	adminHandler := handlers.NewAdminHandler(adminSvc)
	var customerHandler *handlers.CustomerHandler
	var meteringHandler *handlers.MeteringHandler
	if cfg.Mode == "saas" {
		customerHandler = handlers.NewCustomerHandler(customerSvc)
		meteringHandler = handlers.NewMeteringHandler(meteringRepo, customerRepo)
	}
	authHandler := handlers.NewAuthHandler(authSvc)

	// Plan management (Phase 4 — user plan + limits)
	planRepo := memory.NewMemoryUserPlanRepository()

	// App Auth (created early so public routes can be registered before protected group)
	appAuthRepo := memory.NewMemoryAppAuthRepository()
	appAuthHandler := handlers.NewAppAuthHandler(appAuthRepo, appRepo)

	// Phase 2 handlers
	logHub := deploy.NewLogHub(500)
	eventHub := deploy.NewEventHub(50)
	builder := deploy.NewBuilder(appRepo, cfg.BuildWorkDir, cfg.Registry, k8sClient, logHub)
	deployer := deploy.NewDeployer(k8sClient, appRepo, planRepo, cfg.BaseDomain)
	pipeline := deploy.NewPipeline(builder, deployer, appRepo, logHub, eventHub)

	appHandlerV2 := handlers.NewAppHandlerV2(appRepo, cfg.BaseDomain, deployer)
	webhookHandler := handlers.NewWebhookHandler(appRepo, pipeline, cfg.GitHubWebhookSecret)
	deployHandler := handlers.NewDeployHandler(appRepo, pipeline)
	logHandler := handlers.NewLogHandler(appRepo, logHub)

	secretHandler, err := handlers.NewSecretHandler(appRepo, cfg.SecretsKey)
	if err != nil {
		log.Fatalf("invalid SECRETS_ENCRYPTION_KEY: %v", err)
	}

	eventHandler := handlers.NewEventHandler(eventHub)
	backstageHandler := handlers.NewBackstageHandler(k8sClient)

	api := app.Group("/api/v1")
	api.Get("/version", handlers.VersionInfo(Version, BuildTime, GitCommit))

	// Public auth routes (no token required) — rate limited
	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 60 * time.Second,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests. Please try again later.",
			})
		},
	})
	authRoutes := api.Group("/auth", authLimiter)
	authRoutes.Post("/login", authHandler.Login)
	authRoutes.Post("/register", authHandler.Register)
	authRoutes.Post("/refresh", authHandler.Refresh)
	if cfg.GoogleClientID != "" {
		authSvc.SetGoogleClientID(cfg.GoogleClientID)
		authRoutes.Post("/google", authHandler.GoogleLogin)
		log.Printf("[auth] Google OAuth enabled (client_id: %s...)", cfg.GoogleClientID[:8])
	}

	// Webhook routes (no auth — uses HMAC signature)
	api.Post("/webhooks/github", webhookHandler.HandlePush)
	api.Post("/webhooks/gitlab", webhookHandler.HandleGitLabPush)
	api.Post("/webhooks/bitbucket", webhookHandler.HandleBitbucketPush)

	// Public app auth routes (signup/login — no platform JWT required)
	// Registered directly on api (not via Group) to avoid Fiber middleware leakage
	// from the protected group's empty-prefix matcher.
	api.Post("/apps/:appId/auth/signup", appAuthHandler.Signup)
	api.Post("/apps/:appId/auth/login", appAuthHandler.Login)

	// Internal routes (metering agent — SaaS only)
	if cfg.Mode == "saas" && cfg.InternalSecret != "" {
		internal := api.Group("/internal", middleware.RequireInternalSecret(cfg.InternalSecret))
		internal.Post("/metering", meteringHandler.RecordUsage)
	}

	// Protected routes
	protected := api.Group("", middleware.RequireAuth(cfg.JWTSecret))

	// Projects (legacy CRD-based)
	projects := protected.Group("/projects")
	projects.Post("/", projectHandler.Create)
	projects.Get("/", projectHandler.List)
	projects.Get("/:id", projectHandler.Get)
	projects.Put("/:id", projectHandler.Update)
	projects.Delete("/:id", middleware.RequireRole(entities.RoleOwner), projectHandler.Delete)

	// Apps — legacy CRD-based (under /projects/:id/apps)
	legacyApps := protected.Group("/projects/:id/apps")
	legacyApps.Post("/", appHandler.Create)
	legacyApps.Get("/", appHandler.List)
	legacyApps.Get("/:name", appHandler.Get)
	legacyApps.Put("/:name", appHandler.Update)
	legacyApps.Delete("/:name", appHandler.Delete)
	legacyApps.Post("/:name/redeploy", appHandler.Redeploy)

	// Apps — Phase 2 (under /apps)
	apps := protected.Group("/apps")
	apps.Post("/", handlers.CheckLimit(planRepo, "apps", func(c *fiber.Ctx, userID string) (int, error) {
		return appRepo.CountAppsByUser(c.Context(), userID)
	}), appHandlerV2.Create)
	apps.Get("/", appHandlerV2.List)

	// All /apps/:appId routes require ownership check (IDOR prevention)
	appByID := apps.Group("/:appId", middleware.RequireAppOwnership(appRepo))
	appByID.Get("/", appHandlerV2.Get)
	appByID.Delete("/", appHandlerV2.Delete)

	// Deployments (nested under /apps/:appId)
	appByID.Get("/deployments", deployHandler.ListDeployments)
	appByID.Get("/deployments/:deployId", deployHandler.GetDeployment)
	appByID.Post("/rollback", deployHandler.Rollback)

	// Env vars (nested under /apps/:appId)
	appByID.Put("/env", deployHandler.SetEnvVars)
	appByID.Get("/env", deployHandler.GetEnvVars)
	appByID.Delete("/env/:key", deployHandler.DeleteEnvVar)

	// Secrets (nested under /apps/:appId) — only if SECRETS_ENCRYPTION_KEY is set
	if secretHandler != nil {
		appByID.Get("/secrets", secretHandler.ListSecrets)
		appByID.Post("/secrets", secretHandler.SetSecret)
		appByID.Get("/secrets/:key/value", secretHandler.GetSecretValue)
		appByID.Delete("/secrets/:key", secretHandler.DeleteSecret)
	}

	// Databases (Phase 3 — per-app provisioning under /apps/:appId)
	dbRepo := memory.NewMemoryDatabaseRepository()
	dbHandlerV2 := handlers.NewDatabaseHandlerV2(dbRepo, appRepo)
	appByID.Post("/databases", handlers.CheckLimit(planRepo, "databases", func(c *fiber.Ctx, userID string) (int, error) {
		return dbRepo.CountDatabasesByUser(c.Context(), userID)
	}), dbHandlerV2.Create)
	appByID.Get("/databases", dbHandlerV2.List)
	appByID.Get("/databases/:dbId", dbHandlerV2.Get)
	appByID.Delete("/databases/:dbId", dbHandlerV2.Delete)
	protected.Get("/databases", dbHandlerV2.ListByUser)

	// Database Backups (Phase 3 — per-database backup/restore, Pro+ only)
	backupRepo := memory.NewMemoryBackupRepository()
	backupHandler := handlers.NewBackupHandlerV2(backupRepo, dbRepo, planRepo)
	appByID.Post("/databases/:dbId/backups", backupHandler.Create)
	appByID.Get("/databases/:dbId/backups", backupHandler.List)
	appByID.Get("/databases/:dbId/backups/:backupId", backupHandler.Get)
	appByID.Delete("/databases/:dbId/backups/:backupId", backupHandler.Delete)
	appByID.Post("/databases/:dbId/backups/:backupId/restore", backupHandler.Restore)
	protected.Get("/backups", backupHandler.ListByUser)

	// Storage (Phase 3 — per-app S3-compatible storage)
	storageRepo := memory.NewMemoryStorageRepository()
	storageHandlerV2 := handlers.NewStorageHandlerV2(storageRepo, appRepo)
	appByID.Post("/storage", handlers.CheckLimit(planRepo, "buckets", func(c *fiber.Ctx, userID string) (int, error) {
		return storageRepo.CountBucketsByUser(c.Context(), userID)
	}), storageHandlerV2.Create)
	appByID.Get("/storage", storageHandlerV2.List)
	appByID.Get("/storage/:bucketId", storageHandlerV2.Get)
	appByID.Delete("/storage/:bucketId", storageHandlerV2.Delete)
	protected.Get("/storage-buckets", storageHandlerV2.ListByUser)

	// App Auth (Phase 3 — built-in auth per app)
	appByID.Get("/auth", appAuthHandler.Status)
	appByID.Post("/auth/enable", appAuthHandler.Enable)
	appByID.Post("/auth/disable", appAuthHandler.Disable)
	appByID.Get("/auth/users", appAuthHandler.ListUsers)
	appByID.Delete("/auth/users/:userId", appAuthHandler.DeleteUser)

	planSvc := services.NewPlanService(planRepo, appRepo, dbRepo, storageRepo, appAuthRepo)
	planHandler := handlers.NewPlanHandler(planSvc)
	protected.Get("/plan", planHandler.GetMyPlan)
	protected.Post("/plan/upgrade", planHandler.UpgradePlan)

	// Stripe Billing (Phase 6)
	billingRepo := memory.NewMemoryBillingRepository()
	var stripeAPI stripeClient.StripeAPI
	if cfg.StripeBillingEnabled && cfg.StripeSecretKey != "" {
		stripeAPI = stripeClient.NewClient(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
		planSvc.SetStripeEnabled(true)
		log.Println("[billing] Stripe billing enabled")
	} else {
		log.Println("[billing] Stripe billing disabled (direct plan changes allowed)")
	}

	billingSvc := services.NewBillingService(
		stripeAPI, billingRepo, planRepo, appRepo, dbRepo, storageRepo, appAuthRepo,
		cfg.StripeProPriceID, cfg.StripeTeamPriceID, cfg.BaseDomain,
	)
	// Wire S3 for upgrade provisioning (reuse S3 config if available)
	if cfg.S3Endpoint != "" {
		billingSvc.SetStorage(s3client.NewClient(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Region))
	}
	billingHandler := handlers.NewBillingHandler(billingSvc)
	protected.Get("/billing", billingHandler.GetBillingStatus)
	protected.Post("/billing/checkout", billingHandler.CreateCheckoutSession)
	protected.Post("/billing/portal", billingHandler.CreatePortalSession)
	protected.Post("/billing/cancel", billingHandler.CancelSubscription)
	protected.Get("/billing/invoices", billingHandler.ListInvoices)

	// Stripe webhook (unauthenticated — uses Stripe signature verification)
	if stripeAPI != nil {
		webhookHandlerStripe := handlers.NewStripeWebhookHandler(billingSvc, stripeAPI)
		api.Post("/webhooks/stripe", webhookHandlerStripe.HandleEvent)
	}

	// Custom Domains (Phase 4 — Pro+ only)
	domainRepo := memory.NewMemoryDomainRepository()
	domainHandler := handlers.NewDomainHandler(domainRepo, appRepo, planRepo)
	appByID.Post("/domains", domainHandler.Add)
	appByID.Get("/domains", domainHandler.List)
	appByID.Delete("/domains/:domainId", domainHandler.Delete)
	protected.Get("/domains", domainHandler.ListByUser)

	// API Keys (Phase 6.5)
	apiKeyRepo := memory.NewMemoryAPIKeyRepository()
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeyRepo, planRepo)
	protected.Post("/api-keys", apiKeyHandler.Create)
	protected.Get("/api-keys", apiKeyHandler.List)
	protected.Delete("/api-keys/:keyId", apiKeyHandler.Delete)

	// Sessions (Phase 6.5)
	sessionRepo := memory.NewMemorySessionRepository()
	sessionHandler := handlers.NewSessionHandler(sessionRepo)
	protected.Get("/auth/sessions", sessionHandler.List)
	protected.Delete("/auth/sessions/:sessionId", sessionHandler.Revoke)
	protected.Delete("/auth/sessions", sessionHandler.RevokeAll)

	// MFA (Phase 6.5 — Pro+ only)
	mfaRepo := memory.NewMemoryMFARepository()
	mfaHandler := handlers.NewMFAHandler(mfaRepo, planRepo)
	protected.Get("/auth/mfa", mfaHandler.GetStatus)
	protected.Post("/auth/mfa/enable", mfaHandler.Enable)
	protected.Post("/auth/mfa/verify", mfaHandler.Verify)
	protected.Post("/auth/mfa/disable", mfaHandler.Disable)
	protected.Post("/auth/mfa/backup-codes", mfaHandler.RegenerateBackupCodes)

	// User Webhooks (Phase 6.5 — Pro+ only)
	userWebhookRepo := memory.NewMemoryUserWebhookRepository()
	userWebhookHandler := handlers.NewUserWebhookHandler(userWebhookRepo, planRepo)
	protected.Post("/webhooks", userWebhookHandler.Create)
	protected.Get("/webhooks", userWebhookHandler.List)
	protected.Put("/webhooks/:webhookId", userWebhookHandler.Update)
	protected.Delete("/webhooks/:webhookId", userWebhookHandler.Delete)
	protected.Get("/webhooks/:webhookId/deliveries", userWebhookHandler.ListDeliveries)

	// Custom Roles / RBAC (Phase 6.5 — Team+ only)
	roleRepo := memory.NewMemoryRoleRepository()
	roleHandler := handlers.NewRoleHandler(roleRepo, planRepo)
	protected.Post("/roles", roleHandler.Create)
	protected.Get("/roles", roleHandler.List)
	protected.Get("/roles/permissions", roleHandler.ListPermissions)
	protected.Put("/roles/:roleId", roleHandler.Update)
	protected.Delete("/roles/:roleId", roleHandler.Delete)
	protected.Post("/roles/:roleId/assign", roleHandler.AssignRole)
	protected.Get("/roles/:roleId/assignments", roleHandler.ListAssignments)
	protected.Delete("/roles/:roleId/assignments/:assignmentId", roleHandler.RemoveAssignment)

	// IP Whitelisting (Phase 6.5 — Enterprise only)
	ipRepo := memory.NewMemoryIPWhitelistRepository()
	ipHandler := handlers.NewIPWhitelistHandler(ipRepo, planRepo)
	protected.Post("/settings/ip-whitelist", ipHandler.Add)
	protected.Get("/settings/ip-whitelist", ipHandler.List)
	protected.Delete("/settings/ip-whitelist/:entryId", ipHandler.Delete)

	// Compliance Dashboard (Phase 6.5)
	complianceHandler := handlers.NewComplianceHandler(mfaRepo, ipRepo, planRepo, adminRepo)
	protected.Get("/compliance", complianceHandler.GetStatus)

	// DPA + White-label Branding (Phase 6.5)
	brandingRepo := memory.NewMemoryBrandingRepository()
	brandingHandler := handlers.NewBrandingHandler(brandingRepo, planRepo)
	protected.Get("/settings/dpa", brandingHandler.GetDPA)
	protected.Post("/settings/dpa/sign", brandingHandler.SignDPA)
	protected.Get("/settings/branding", brandingHandler.GetBranding)
	protected.Put("/settings/branding", brandingHandler.UpdateBranding)
	protected.Post("/settings/domain", brandingHandler.SetDashboardDomain)

	// SSO (Phase 6.5 — Team+ only)
	ssoRepo := memory.NewMemorySSORepository()
	ssoHandler := handlers.NewSSOHandler(ssoRepo, planRepo)
	protected.Post("/settings/sso/saml", ssoHandler.ConfigureSAML)
	protected.Post("/settings/sso/oidc", ssoHandler.ConfigureOIDC)
	protected.Get("/settings/sso", ssoHandler.ListConfigs)
	protected.Delete("/settings/sso/:configId", ssoHandler.DeleteConfig)

	// Preview Deployments (Phase 6.5 — Team+ only)
	previewRepo := memory.NewMemoryPreviewRepository()
	previewHandler := handlers.NewPreviewHandler(previewRepo, appRepo, planRepo)
	appByID.Post("/previews", previewHandler.Create)
	appByID.Get("/previews", previewHandler.List)
	appByID.Delete("/previews/:previewId", previewHandler.Delete)

	// SCIM 2.0 Provisioning (Phase 6.5 — Enterprise only, SaaS only)
	// Protected with admin auth — in production, replace with SCIM bearer token validation
	if cfg.Mode == "saas" {
		scimHandler := handlers.NewSCIMHandler(userRepo, planRepo)
		scim := api.Group("/scim/v2", middleware.RequireAuth(cfg.JWTSecret), middleware.RequireRole(entities.RoleAdmin))
		scim.Get("/Users", scimHandler.ListUsers)
		scim.Get("/Users/:userId", scimHandler.GetUser)
		scim.Post("/Users", scimHandler.CreateUser)
		scim.Delete("/Users/:userId", scimHandler.DeleteUser)
	}

	// Releases (nested under /apps/:appId) — versioned image deployment
	releaseHandler := handlers.NewReleaseHandler(appRepo, pipeline)
	appByID.Post("/releases", releaseHandler.CreateRelease)
	appByID.Get("/releases", releaseHandler.ListReleases)
	appByID.Post("/releases/:releaseId/deploy", releaseHandler.DeployRelease)

	// Build/deploy log streaming (nested under /apps/:appId/deployments/:did)
	appByID.Get("/deployments/:did/logs", logHandler.StreamLogs)
	appByID.Get("/deployments/:did/logs/history", logHandler.GetLogs)

	// Real-time deployment events (SSE)
	protected.Get("/events", eventHandler.StreamEvents)
	protected.Get("/events/history", eventHandler.GetRecentEvents)

	// Backstage catalog integration
	protected.Get("/backstage/catalog", backstageHandler.GetCatalog)
	protected.Get("/backstage/catalog/:kind", backstageHandler.GetCatalogByKind)

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
	admin := protected.Group("/admin", middleware.RequireRole(entities.RoleAdmin))

	// Dashboard
	admin.Get("/dashboard/stats", adminHandler.GetDashboardStats)

	// SaaS-only admin routes: customer mgmt, metering, cluster, tenants, plan admin
	if cfg.Mode == "saas" {
		admin.Get("/dashboard/usage", meteringHandler.GetPlatformUsageSummary)

		// Customer management
		admin.Post("/customers", customerHandler.CreateCustomer)
		admin.Get("/customers", customerHandler.ListCustomers)
		admin.Get("/customers/stats", customerHandler.GetCustomerStats) // before :id
		admin.Get("/customers/:id", customerHandler.GetCustomer)
		admin.Get("/customers/:id/usage", meteringHandler.GetCustomerUsage)
		admin.Get("/customers/:id/usage/history", meteringHandler.GetCustomerUsageHistory)
		admin.Put("/customers/:id", customerHandler.UpdateCustomer)
		admin.Post("/customers/:id/suspend", customerHandler.SuspendCustomer)
		admin.Post("/customers/:id/activate", customerHandler.ActivateCustomer)
		admin.Delete("/customers/:id", customerHandler.DeleteCustomer)
		admin.Get("/customers/:id/cluster", customerHandler.GetCustomerCluster)
		admin.Post("/customers/:id/cluster/scale", customerHandler.ScaleCluster)
		admin.Post("/customers/:id/cluster/upgrade", customerHandler.UpgradeCluster)

		// Plan management (admin)
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
	}

	// Module management
	admin.Get("/modules", adminHandler.ListModules)
	admin.Post("/modules/update-all", adminHandler.UpdateAllModules)
	admin.Post("/modules/:name/install", adminHandler.InstallModule)
	admin.Post("/modules/:name/uninstall", adminHandler.UninstallModule)
	admin.Post("/modules/:name/update", adminHandler.UpdateModule)

	// User management (SaaS — Phase 4)
	adminUserHandler := handlers.NewAdminUserHandler(userRepo, planRepo, appRepo, dbRepo, storageRepo)
	admin.Get("/users/:userId", adminUserHandler.GetUser)
	admin.Post("/users/:userId/plan", adminUserHandler.SetUserPlan)
	admin.Get("/users/:userId/apps", adminUserHandler.ListUserApps)
	admin.Get("/users/:userId/databases", adminUserHandler.ListUserDatabases)

	// Admin Billing Overview (Phase 6 — SaaS only)
	if cfg.Mode == "saas" {
		admin.Get("/billing/overview", billingHandler.GetAdminBillingOverview)
	}

	// Audit log
	admin.Get("/audit", adminHandler.ListAuditLog)
	auditExportHandler := handlers.NewAuditExportHandler(adminRepo)
	admin.Get("/audit/export/csv", auditExportHandler.ExportCSV)
	admin.Get("/audit/export/json", auditExportHandler.ExportJSON)

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

	// Hetzner Autoscaler (Phase 5 — SaaS only)
	var as *autoscale.Autoscaler
	if cfg.Mode == "saas" {
		autoscaleRepo := memory.NewMemoryAutoscaleRepository()
		autoscaleHandler := handlers.NewAutoscaleHandler(autoscaleRepo)
		admin.Get("/autoscaler/status", autoscaleHandler.GetStatus)
		admin.Get("/autoscaler/nodes", autoscaleHandler.ListNodes)
		admin.Get("/autoscaler/events", autoscaleHandler.ListEvents)

		// Create autoscaler if enabled and token present
		if cfg.AutoscalerEnabled && cfg.HetznerToken != "" {
			hetznerClient := hetznerclient.NewClient(cfg.HetznerToken)
			metricsProvider := autoscale.NewK8sMetricsProvider(k8sClient)
			asCfg := entities.AutoscalerConfig{
				MinNodes:     cfg.AutoscalerMinNodes,
				MaxNodes:     cfg.AutoscalerMaxNodes,
				ScaleUpCPU:   80,
				ScaleUpRAM:   80,
				ScaleDownCPU: 40,
				ScaleDownRAM: 40,
				CooldownUp:   5 * time.Minute,
				CooldownDown: 15 * time.Minute,
				BudgetCapEUR: 450,
				ServerType:   cfg.HetznerServerType,
				Location:     cfg.HetznerLocation,
			}
			as = autoscale.NewAutoscaler(hetznerClient, metricsProvider, autoscaleRepo, adminRepo, asCfg, cfg.K3sToken, "https://"+cfg.BaseDomain+":6443")
			log.Println("[autoscaler] Hetzner autoscaler configured")
		} else {
			log.Println("[autoscaler] disabled (AUTOSCALER_ENABLED=false or HCLOUD_TOKEN not set)")
		}
	}

	return provisioner, as, tw
}
