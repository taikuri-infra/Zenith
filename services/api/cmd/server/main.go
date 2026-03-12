package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dotechhq/zenith/services/api/docs"
	"github.com/dotechhq/zenith/services/api/internal/adapters/capiclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/keycloakclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/lokiclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/harborclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/natsclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/promclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/redisclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/adapters/postgres"
	"github.com/dotechhq/zenith/services/api/internal/adapters/postgres/migrations"
	"github.com/dotechhq/zenith/services/api/internal/adapters/resendclient"
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
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	temporalWorker "go.temporal.io/sdk/worker"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	cfg := config.Load()

	// Initialize structured JSON logging
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(logHandler))

	ctx := context.Background()

	// Database (optional — falls back to in-memory stores when DATABASE_URL is empty)
	var pool *pgxpool.Pool
	var userRepo ports.UserRepository
	var adminRepo ports.AdminRepository
	var customerRepo ports.CustomerRepository
	var meteringRepo ports.MeteringRepository
	var appRepo ports.AppRepository

	if cfg.DatabaseURL != "" {
		slog.Info("connecting to PostgreSQL")
		if err := postgres.RunMigrations(cfg.DatabaseURL, migrations.FS); err != nil {
			slog.Error("database migrations failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations applied")

		var err error
		pool, err = postgres.NewPostgresPool(ctx, cfg.DatabaseURL)
		if err != nil {
			slog.Error("database connection failed", "error", err)
			os.Exit(1)
		}
		slog.Info("connected to PostgreSQL")

		userRepo = postgres.NewPostgresUserRepository(pool)
		adminRepo = postgres.NewPostgresAdminRepository(pool)
		customerRepo = postgres.NewPostgresCustomerRepository(pool)
		meteringRepo = postgres.NewPostgresMeteringRepository(pool)
		appRepo = postgres.NewPostgresAppRepository(pool)
	} else {
		slog.Info("no DATABASE_URL set, using in-memory stores")
		userRepo = memory.NewMemoryUserRepository()
		adminRepo = memory.NewMemoryAdminRepository()
		customerRepo = memory.NewMemoryCustomerRepository()
		meteringRepo = memory.NewMemoryMeteringRepository()
		appRepo = memory.NewMemoryAppRepository()
	}

	// Seed admin user
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		if _, err := userRepo.Create(ctx, cfg.AdminEmail, cfg.AdminPassword, "Admin", entities.RoleOwner); err != nil {
			slog.Info("admin seed skipped", "error", err)
		} else {
			slog.Info("admin user seeded", "email", cfg.AdminEmail)
		}
	}

	slog.Info("zenith mode", "mode", cfg.Mode)
	if cfg.Mode == "standalone" {
		slog.Info("running in standalone mode, multi-tenant and billing features disabled")
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
			slog.Warn("OpenTelemetry init failed, continuing without tracing", "error", err)
		} else {
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := otelShutdown(shutdownCtx); err != nil {
					slog.Warn("OpenTelemetry shutdown error", "error", err)
				}
			}()
			slog.Info("OpenTelemetry enabled", "endpoint", cfg.OTELEndpoint)
		}
	}

	app := fiber.New(fiber.Config{
		AppName:      "Zenith API",
		ServerHeader: "",
		ErrorHandler: handlers.ErrorHandler,
		BodyLimit:    50 * 1024 * 1024, // 50 MB (covers file uploads, auth requests are small)
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(middleware.SecurityHeaders())
	app.Use(middleware.StructuredLogger())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Request-ID,X-API-Key",
		AllowCredentials: true,
		MaxAge:           3600,
	}))
	app.Use(middleware.RequestContext())

	// OpenTelemetry middleware (only effective when OTel SDK is initialized)
	if cfg.OTELEndpoint != "" {
		app.Use(telemetry.Middleware(telemetry.MiddlewareConfig{
			SkipPaths: []string{"/health", "/ready"},
		}))
		slog.Info("OpenTelemetry tracing middleware active")
	}

	docs.RegisterRoutes(app)
	provisioner, autoscaler, temporalW, tokenBlacklist, eventBus, redisClient := setupRoutes(app, cfg, userRepo, adminRepo, customerRepo, meteringRepo, appRepo, pool)

	// Start cluster sync if provisioner is available
	if provisioner != nil {
		provisioner.StartSync(30 * time.Second)
		slog.Info("cluster provisioner sync started", "interval", "30s")
	}

	// Start Temporal worker if configured
	if temporalW != nil {
		if err := temporalW.Start(); err != nil {
			slog.Error("failed to start Temporal worker", "error", err)
			os.Exit(1)
		}
		slog.Info("temporal worker started", "queue", zenithTemporal.TaskQueue)
	}

	// Start autoscaler if enabled
	if autoscaler != nil {
		autoscaler.Start(time.Duration(cfg.AutoscalerInterval) * time.Second)
		slog.Info("hetzner autoscaler started", "interval_s", cfg.AutoscalerInterval, "min_nodes", cfg.AutoscalerMinNodes, "max_nodes", cfg.AutoscalerMaxNodes)
	}

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		slog.Info("zenith API starting", "version", Version, "addr", addr)
		if err := app.Listen(addr); err != nil {
			slog.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server", "grace_period", "30s")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop background workers first
	if temporalW != nil {
		temporalW.Stop()
		slog.Info("temporal worker stopped")
	}
	if autoscaler != nil {
		autoscaler.Stop()
		slog.Info("hetzner autoscaler stopped")
	}
	if provisioner != nil {
		provisioner.Stop()
		slog.Info("cluster provisioner stopped")
	}
	if tokenBlacklist != nil {
		tokenBlacklist.Stop()
		slog.Info("token blacklist stopped")
	}

	// Shutdown HTTP server with timeout
	done := make(chan struct{})
	go func() {
		if err := app.Shutdown(); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
		close(done)
	}()

	select {
	case <-done:
		slog.Info("HTTP server stopped gracefully")
	case <-shutdownCtx.Done():
		slog.Warn("shutdown timeout reached, forcing exit")
	}

	if eventBus != nil {
		eventBus.Close()
		slog.Info("event bus closed")
	}
	if redisClient != nil {
		redisClient.Close()
		slog.Info("redis closed")
	}
	if pool != nil {
		pool.Close()
		slog.Info("database pool closed")
	}
	slog.Info("server stopped")
}

func setupRoutes(app *fiber.App, cfg *config.Config, userRepo ports.UserRepository, adminRepo ports.AdminRepository, customerRepo ports.CustomerRepository, meteringRepo ports.MeteringRepository, appRepo ports.AppRepository, pool *pgxpool.Pool) (*cluster.Provisioner, *autoscale.Autoscaler, temporalWorker.Worker, *middleware.TokenBlacklist, ports.EventBus, *redisclient.Client) {
	app.Get("/health", handlers.HealthCheck(Version))
	app.Get("/ready", handlers.ReadinessCheck(pool))

	// K8s client (in-memory for dev, real client-go for production)
	var k8sClient k8sclient.Client
	if cfg.K8sMode == "real" {
		realClient, err := k8sclient.NewRealClient()
		if err != nil {
			slog.Error("failed to create real K8s client", "error", err)
			os.Exit(1)
		}
		k8sClient = realClient
		slog.Info("k8s using real client-go connection")
	} else {
		k8sClient = k8sclient.NewMemoryClient()
		slog.Info("k8s using in-memory client (dev mode)")
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

	// Plan management (Phase 4 — user plan + limits)
	var planRepo ports.UserPlanRepository
	if pool != nil {
		planRepo = postgres.NewPostgresUserPlanRepository(pool)
	} else {
		planRepo = memory.NewMemoryUserPlanRepository()
	}

	// Team members (IAM)
	var teamRepo ports.TeamMemberRepository
	if pool != nil {
		teamRepo = postgres.NewPostgresTeamMemberRepository(pool)
	} else {
		teamRepo = memory.NewMemoryTeamMemberRepository()
	}

	// Support Tickets (Pro+ only)
	var supportRepo ports.SupportRepository
	if pool != nil {
		supportRepo = postgres.NewPostgresSupportRepository(pool)
	} else {
		supportRepo = memory.NewMemorySupportRepository()
	}
	supportSvc := services.NewSupportService(supportRepo, planRepo, userRepo)

	authSvc := services.NewAuthService(userRepo, cfg.JWTSecret, planRepo)
	teamSvc := services.NewTeamMemberService(teamRepo, userRepo, planRepo, cfg.JWTSecret)

	// Email verification (opt-in: only when RESEND_API_KEY is set)
	if cfg.ResendAPIKey != "" {
		emailFrom := cfg.EmailFrom
		if emailFrom == "" {
			emailFrom = "Zenith <noreply@freezenith.com>"
		}
		emailSender := resendclient.NewClient(cfg.ResendAPIKey, emailFrom)
		authSvc.SetEmailSender(emailSender, cfg.AppURL)
		teamSvc.SetEmailSender(emailSender, cfg.AppURL)
		supportSvc.SetEmailSender(emailSender, cfg.AppURL, cfg.AdminEmail)
		slog.Info("email verification enabled (Resend)")
	} else {
		slog.Info("email verification disabled (no RESEND_API_KEY)")
	}

	// NATS JetStream event bus (opt-in: only when NATS_ENABLED=true)
	var eventBus ports.EventBus
	if cfg.NATSEnabled {
		natsClient, err := natsclient.New(cfg.NATSServers, cfg.NATSStreamName)
		if err != nil {
			slog.Warn("NATS failed to connect, falling back to memory", "error", err)
			eventBus = memory.NewMemoryEventBus()
		} else {
			eventBus = natsClient
		}
	} else {
		eventBus = memory.NewMemoryEventBus()
		slog.Info("NATS disabled, using in-memory event bus")
	}

	// Notification subscriber — consumes platform events and sends emails
	notifSvc := services.NewNotificationService(eventBus, nil, userRepo, cfg.AppURL)
	if err := notifSvc.Start(); err != nil {
		slog.Warn("notification service failed to start", "error", err)
	}

	var customerSvc *services.CustomerService
	var tw temporalWorker.Worker
	var temporalWfClient *zenithTemporal.WorkflowClient
	if cfg.Mode == "saas" {
		customerSvc = services.NewCustomerService(customerRepo, adminRepo, provisioner)

		// Temporal provisioning (opt-in: only when TEMPORAL_ENABLED=true)
		if cfg.TemporalEnabled {
			tc, err := zenithTemporal.NewClient(cfg.TemporalHost, cfg.TemporalNamespace)
			if err != nil {
				slog.Warn("temporal failed to connect, provisioning falls back to goroutine", "error", err)
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

				// PlanActivities repos are set later after repo creation (fields are pointer-safe)
				planActivities := &zenithTemporal.PlanActivities{
					PlanRepo: planRepo,
					Admin:    adminRepo,
				}

				tw = zenithTemporal.NewWorker(tc, activities, planActivities)
				temporalWfClient = zenithTemporal.NewWorkflowClient(tc)
				customerSvc.SetWorkflows(temporalWfClient)
				slog.Info("temporal connected", "host", cfg.TemporalHost, "namespace", cfg.TemporalNamespace)
			}
		} else {
			slog.Info("temporal disabled")
		}
	}

	// Project repository (DB-backed project CRUD)
	var projectRepo ports.ProjectRepository
	if pool != nil {
		projectRepo = postgres.NewPostgresProjectRepository(pool)
	} else {
		projectRepo = memory.NewMemoryProjectRepository()
	}

	// Wire project repo into auth service (auto-create default project on registration)
	authSvc.SetProjectRepo(projectRepo)

	// Wire team repo into auth service (team member login enrichment)
	authSvc.SetTeamRepo(teamRepo)

	// Handlers
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
	tokenBlacklist := middleware.NewTokenBlacklist()

	// Redis (rate limiting + token blacklist) — optional, falls back to in-memory
	var redisClient *redisclient.Client
	var rateLimiterStorage fiber.Storage
	if cfg.RedisURL != "" {
		rc, err := redisclient.New(cfg.RedisURL)
		if err != nil {
			slog.Warn("redis unavailable, falling back to in-memory", "error", err)
		} else {
			redisClient = rc
			rateLimiterStorage = rc.NewRateLimiterStorage("zenith:ratelimit:")
			tokenBlacklist.SetRedisBackend(rc.NewTokenBlacklist())
			slog.Info("redis-backed rate limiter and token blacklist enabled")
		}
	}

	// User Event Tracking (business growth — all features depend on this)
	var eventRepo ports.UserEventRepository
	if pool != nil {
		eventRepo = postgres.NewPostgresUserEventRepository(pool)
	} else {
		eventRepo = memory.NewMemoryUserEventRepository()
	}

	// Email Sends (drip campaign tracking)
	var emailSendRepo ports.EmailSendRepository
	if pool != nil {
		emailSendRepo = postgres.NewPostgresEmailSendRepository(pool)
	} else {
		emailSendRepo = memory.NewMemoryEmailSendRepository()
	}

	// Exit Surveys
	var exitSurveyRepo ports.ExitSurveyRepository
	if pool != nil {
		exitSurveyRepo = postgres.NewPostgresExitSurveyRepository(pool)
	} else {
		exitSurveyRepo = memory.NewMemoryExitSurveyRepository()
	}

	// Referral Rewards
	var referralRepo ports.ReferralRepository
	if pool != nil {
		referralRepo = postgres.NewPostgresReferralRepository(pool)
	} else {
		referralRepo = memory.NewMemoryReferralRepository()
	}

	authHandler := handlers.NewAuthHandler(authSvc, tokenBlacklist)
	authHandler.SetEventRepo(eventRepo)

	// App Auth (created early so public routes can be registered before protected group)
	var appAuthRepo ports.AppAuthRepository
	if pool != nil {
		appAuthRepo = postgres.NewPostgresAppAuthRepository(pool)
	} else {
		appAuthRepo = memory.NewMemoryAppAuthRepository()
	}
	appAuthHandler := handlers.NewAppAuthHandler(appAuthRepo, appRepo)

	// Phase 2 handlers
	logHub := deploy.NewLogHub(500)
	eventHub := deploy.NewEventHub(50)
	deployer := deploy.NewDeployer(k8sClient, appRepo, planRepo, cfg.BaseDomain)
	pipeline := deploy.NewPipeline(deployer, appRepo, logHub, eventHub, cfg.MaxConcurrentDeploys)
	pipeline.SetEventBus(eventBus)

	appHandlerV2 := handlers.NewAppHandlerV2(appRepo, cfg.BaseDomain, deployer, pipeline)
	appHandlerV2.SetProjectRepo(projectRepo)
	appHandlerV2.SetPlanRepo(planRepo)
	appHandlerV2.SetEventRepo(eventRepo)
	projectHandlerV2 := handlers.NewProjectHandlerV2(projectRepo, appRepo, deployer)
	deployHandler := handlers.NewDeployHandler(appRepo, pipeline)
	logHandler := handlers.NewLogHandler(appRepo, logHub)

	secretHandler, err := handlers.NewSecretHandler(appRepo, cfg.SecretsKey)
	if err != nil {
		slog.Error("invalid SECRETS_ENCRYPTION_KEY", "error", err)
		os.Exit(1)
	}

	eventHandler := handlers.NewEventHandler(eventHub)
	backstageHandler := handlers.NewBackstageHandler(k8sClient)

	api := app.Group("/api/v1")
	api.Get("/version", handlers.VersionInfo(Version, BuildTime))

	// Public auth routes (no token required) — rate limited
	authLimiterConfig := limiter.Config{
		Max:        10,
		Expiration: 60 * time.Second,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests. Please try again later.",
			})
		},
	}
	if rateLimiterStorage != nil {
		authLimiterConfig.Storage = rateLimiterStorage
	}
	authLimiter := limiter.New(authLimiterConfig)
	// Auth body limiter — auth payloads are small JSON (< 10 KB), reject oversized requests
	authBodyLimit := func(c *fiber.Ctx) error {
		if len(c.Body()) > 10*1024 {
			return fiber.NewError(fiber.StatusRequestEntityTooLarge, "request body too large")
		}
		return c.Next()
	}
	authRoutes := api.Group("/auth", authLimiter, authBodyLimit)
	authRoutes.Get("/me", middleware.RequireAuth(cfg.JWTSecret, tokenBlacklist), authHandler.GetMe)
	authRoutes.Put("/onboarding", middleware.RequireAuth(cfg.JWTSecret, tokenBlacklist), authHandler.UpdateOnboarding)
	authRoutes.Post("/login", authHandler.Login)
	authRoutes.Post("/login/mfa", authHandler.MFALogin)
	authRoutes.Post("/register", authHandler.Register)
	authRoutes.Post("/refresh", authHandler.Refresh)
	authRoutes.Post("/logout", middleware.RequireAuth(cfg.JWTSecret, tokenBlacklist), authHandler.Logout)
	authRoutes.Post("/verify-email", authHandler.VerifyEmail)
	authRoutes.Post("/resend-verification", authHandler.ResendVerification)
	if cfg.GoogleClientID != "" || cfg.GitHubClientID != "" {
		authSvc.SetOAuthConfig(services.OAuthConfig{
			GoogleClientID:     cfg.GoogleClientID,
			GoogleClientSecret: cfg.GoogleClientSecret,
			GitHubClientID:     cfg.GitHubClientID,
			GitHubClientSecret: cfg.GitHubClientSecret,
			AppURL:             cfg.AppURL,
		})
		authHandler.SetAppURL(cfg.AppURL)
		authRoutes.Get("/oauth/:provider", authHandler.OAuthRedirect)
		authRoutes.Get("/oauth/:provider/callback", authHandler.OAuthCallback)
		authRoutes.Post("/exchange", authHandler.ExchangeOAuthCode)
		if cfg.GoogleClientID != "" {
			slog.Info("Google OAuth enabled", "client_id_prefix", cfg.GoogleClientID[:8])
		}
		if cfg.GitHubClientID != "" {
			slog.Info("GitHub OAuth enabled", "client_id_prefix", cfg.GitHubClientID[:8])
		}
	}

	// Public app auth routes (signup/login — no platform JWT required)
	// Registered directly on api (not via Group) to avoid Fiber middleware leakage
	// from the protected group's empty-prefix matcher.
	api.Post("/apps/:appId/auth/signup", appAuthHandler.Signup)
	api.Post("/apps/:appId/auth/login", appAuthHandler.Login)

	// Public team invite accept (no auth required)
	teamHandler := handlers.NewTeamMemberHandler(teamSvc)
	api.Post("/team/accept-invite", teamHandler.AcceptInvite)

	// Internal routes (metering agent — SaaS only)
	if cfg.Mode == "saas" && cfg.InternalSecret != "" {
		internal := api.Group("/internal", middleware.RequireInternalSecret(cfg.InternalSecret))
		internal.Post("/metering", meteringHandler.RecordUsage)
	}

	// Protected routes
	protected := api.Group("", middleware.RequireAuth(cfg.JWTSecret, tokenBlacklist))

	// Team members (IAM)
	protected.Post("/team/invite", teamHandler.InviteMember)
	protected.Get("/team/members", teamHandler.ListMembers)
	protected.Put("/team/members/:id/role", teamHandler.UpdateRole)
	protected.Delete("/team/members/:id", teamHandler.RemoveMember)

	// Projects (DB-backed)
	projects := protected.Group("/projects")
	projects.Post("/", projectHandlerV2.Create)
	projects.Get("/", projectHandlerV2.List)
	projects.Get("/:projectId", projectHandlerV2.Get)
	projects.Put("/:projectId", projectHandlerV2.Update)
	projects.Delete("/:projectId", projectHandlerV2.Delete)

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
	apps.Get("/check-name", appHandlerV2.CheckName)
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
	var dbRepo ports.DatabaseRepository
	if pool != nil {
		dbRepo = postgres.NewPostgresDatabaseRepository(pool)
	} else {
		dbRepo = memory.NewMemoryDatabaseRepository()
	}

	// Real CNPG provisioning (opt-in: only when CNPG_ADMIN_DSN is set)
	var dbSvc *services.DatabaseService
	if cfg.CNPGAdminDSN != "" {
		dbSvc = services.NewDatabaseService(dbRepo, appRepo, planRepo, k8sClient, cfg.CNPGAdminDSN, "zenith-apps")
		slog.Info("CNPG database provisioning enabled")
	} else {
		slog.Info("CNPG not configured, metadata-only mode (dev)")
	}

	dbHandlerV2 := handlers.NewDatabaseHandlerV2(dbSvc, dbRepo, appRepo)
	appByID.Post("/databases", handlers.CheckLimit(planRepo, "databases", func(c *fiber.Ctx, userID string) (int, error) {
		return dbRepo.CountDatabasesByUser(c.Context(), userID)
	}), dbHandlerV2.Create)
	appByID.Get("/databases", dbHandlerV2.List)
	appByID.Get("/databases/:dbId", dbHandlerV2.Get)
	appByID.Post("/databases/:dbId/reset-password", dbHandlerV2.ResetPassword)
	appByID.Delete("/databases/:dbId", dbHandlerV2.Delete)

	// Standalone databases (not tied to an app)
	protected.Post("/databases", handlers.CheckLimit(planRepo, "databases", func(c *fiber.Ctx, userID string) (int, error) {
		return dbRepo.CountDatabasesByUser(c.Context(), userID)
	}), dbHandlerV2.CreateStandalone)
	protected.Get("/databases", dbHandlerV2.ListByUser)
	protected.Get("/databases/:dbId", dbHandlerV2.GetStandalone)
	protected.Post("/databases/:dbId/reset-password", dbHandlerV2.ResetPassword)
	protected.Delete("/databases/:dbId", dbHandlerV2.DeleteStandalone)

	// Database Explorer (pgweb — on-demand database browser)
	pgwebSvc := services.NewPgwebService(dbRepo, k8sClient, "zenith-apps", cfg.BaseDomain)
	go pgwebSvc.CleanupExpired(context.Background())
	pgwebHandler := handlers.NewPgwebHandler(pgwebSvc, dbRepo)
	protected.Post("/databases/:dbId/explorer", pgwebHandler.Start)
	protected.Get("/databases/:dbId/explorer", pgwebHandler.Status)
	protected.Delete("/databases/:dbId/explorer", pgwebHandler.Stop)

	// API Gateways (APISIX-powered customer gateways)
	var gwRepo ports.GatewayRepository
	if pool != nil {
		gwRepo = postgres.NewPostgresGatewayRepository(pool)
	} else {
		gwRepo = memory.NewMemoryGatewayRepository()
	}
	gwSvc := services.NewGatewayService(gwRepo, appRepo, planRepo, k8sClient, cfg.GatewayDomain, "zenith-apps")
	gwSvc.SetAppsDomain(cfg.BaseDomain)
	gwHandler := handlers.NewGatewayHandler(gwSvc, gwRepo, projectRepo)
	onAppDeleted := func(ctx context.Context, appID string) {
		gwSvc.HandleAppDeleted(ctx, appID)
	}
	appHandlerV2.SetOnAppDeleted(onAppDeleted)
	appHandlerV2.SetGatewayService(gwSvc)
	projectHandlerV2.SetOnAppDeleted(onAppDeleted)
	go gwSvc.ReconcileAll(context.Background())

	gateways := protected.Group("/gateways")
	gateways.Post("/", handlers.CheckLimit(planRepo, "gateways", func(c *fiber.Ctx, userID string) (int, error) {
		return gwRepo.CountGatewaysByUser(c.Context(), userID)
	}), gwHandler.CreateGateway)
	gateways.Get("/", gwHandler.ListGateways)

	gwByID := gateways.Group("/:gwId")
	gwByID.Get("/", gwHandler.GetGateway)
	gwByID.Put("/", gwHandler.UpdateGateway)
	gwByID.Delete("/", gwHandler.DeleteGateway)
	gwByID.Post("/sync", gwHandler.SyncGateway)

	gwByID.Post("/routes", handlers.CheckLimit(planRepo, "gateway_routes", func(c *fiber.Ctx, userID string) (int, error) {
		return gwRepo.CountRoutesByUser(c.Context(), userID)
	}), gwHandler.CreateRoute)
	gwByID.Get("/routes", gwHandler.ListRoutes)
	gwByID.Put("/routes/:routeId", gwHandler.UpdateRoute)
	gwByID.Delete("/routes/:routeId", gwHandler.DeleteRoute)

	gwByID.Post("/groups", gwHandler.CreateGroup)
	gwByID.Get("/groups", gwHandler.ListGroups)
	gwByID.Put("/groups/:groupId", gwHandler.UpdateGroup)
	gwByID.Delete("/groups/:groupId", gwHandler.DeleteGroup)

	// Database Backups (Phase 3 — per-database backup/restore, Pro+ only)
	var backupRepo ports.BackupRepository
	if pool != nil {
		backupRepo = postgres.NewPostgresBackupRepository(pool)
	} else {
		backupRepo = memory.NewMemoryBackupRepository()
	}
	backupHandler := handlers.NewBackupHandlerV2(backupRepo, dbRepo, planRepo)
	appByID.Post("/databases/:dbId/backups", backupHandler.Create)
	appByID.Get("/databases/:dbId/backups", backupHandler.List)
	appByID.Get("/databases/:dbId/backups/:backupId", backupHandler.Get)
	appByID.Delete("/databases/:dbId/backups/:backupId", backupHandler.Delete)
	appByID.Post("/databases/:dbId/backups/:backupId/restore", backupHandler.Restore)
	protected.Get("/backups", backupHandler.ListByUser)

	// Storage (Phase 3 — per-app S3-compatible storage)
	var storageRepo ports.StorageRepository
	if pool != nil {
		storageRepo = postgres.NewPostgresStorageRepository(pool)
	} else {
		storageRepo = memory.NewMemoryStorageRepository()
	}
	// S3 client + BucketService (real per-customer S3 buckets)
	var objStorage ports.ObjectStorage
	if cfg.S3Endpoint != "" {
		objStorage = s3client.NewClient(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Region)
	} else {
		objStorage = s3client.NewMemoryClient()
	}
	bucketSvc := services.NewBucketService(objStorage)

	storageHandlerV2 := handlers.NewStorageHandlerV2(storageRepo, appRepo, bucketSvc)
	appByID.Post("/storage", handlers.CheckLimit(planRepo, "buckets", func(c *fiber.Ctx, userID string) (int, error) {
		return storageRepo.CountBucketsByUser(c.Context(), userID)
	}), storageHandlerV2.Create)
	appByID.Get("/storage", storageHandlerV2.List)
	appByID.Get("/storage/:bucketId", storageHandlerV2.Get)
	appByID.Delete("/storage/:bucketId", storageHandlerV2.Delete)

	tenantStorage := services.NewTenantStorage(objStorage, cfg.S3PlatformBucket)
	storageHandlerV2.SetTenantStorage(tenantStorage)

	// Standalone storage buckets (not app-scoped)
	storageBuckets := protected.Group("/storage-buckets")
	storageBuckets.Post("/", handlers.CheckLimit(planRepo, "buckets", func(c *fiber.Ctx, userID string) (int, error) {
		return storageRepo.CountBucketsByUser(c.Context(), userID)
	}), storageHandlerV2.CreateStandalone)
	storageBuckets.Get("/", storageHandlerV2.ListByUser)

	storageBucketByID := storageBuckets.Group("/:bucketId")
	storageBucketByID.Get("/", storageHandlerV2.GetStandalone)
	storageBucketByID.Put("/", storageHandlerV2.UpdateBucket)
	storageBucketByID.Delete("/", storageHandlerV2.DeleteStandalone)

	// Object operations within buckets
	quotaSvc := services.NewStorageQuotaService(objStorage, planRepo, storageRepo, cfg.S3PlatformBucket)
	storageObjHandler := handlers.NewStorageObjectHandler(storageRepo, tenantStorage, quotaSvc)
	storageBucketByID.Get("/objects", storageObjHandler.ListObjects)
	storageBucketByID.Post("/objects/upload", storageObjHandler.GetUploadURL)
	storageBucketByID.Get("/objects/download", storageObjHandler.GetDownloadURL)
	storageBucketByID.Put("/objects/content", storageObjHandler.UploadObject)
	storageBucketByID.Get("/objects/content", storageObjHandler.DownloadObject)
	storageBucketByID.Delete("/objects", storageObjHandler.DeleteObject)
	storageBucketByID.Post("/objects/folder", storageObjHandler.CreateFolder)

	// App Auth (Phase 3 — built-in auth per app)
	appByID.Get("/auth", appAuthHandler.Status)
	appByID.Post("/auth/enable", appAuthHandler.Enable)
	appByID.Post("/auth/disable", appAuthHandler.Disable)
	appByID.Get("/auth/users", appAuthHandler.ListUsers)
	appByID.Delete("/auth/users/:userId", appAuthHandler.DeleteUser)

	// Auth Pools (managed authentication backed by Keycloak)
	var authPoolRepo ports.AuthPoolRepository
	if pool != nil {
		authPoolRepo = postgres.NewPostgresAuthPoolRepository(pool)
	} else {
		authPoolRepo = memory.NewMemoryAuthPoolRepository()
	}

	var keycloakIDP ports.IdentityProvider
	if cfg.KeycloakURL != "" {
		keycloakIDP = keycloakclient.NewClient(cfg.KeycloakURL, cfg.KeycloakAdminUser, cfg.KeycloakAdminPassword)
	} else {
		keycloakIDP = keycloakclient.NewMemoryClient()
	}

	authPoolSvc := services.NewAuthPoolService(authPoolRepo, planRepo, keycloakIDP, cfg.KeycloakURL, cfg.KeycloakExternalURL)
	gwSvc.SetAuthPoolRepo(authPoolRepo)
	authPoolSvc.SetGatewayDependencies(gwRepo, gwSvc)
	authPoolHandler := handlers.NewAuthPoolHandler(authPoolSvc, authPoolRepo, cfg.KeycloakURL)

	authPools := protected.Group("/auth-pools")
	authPools.Post("/", handlers.CheckLimit(planRepo, "auth_pools", func(c *fiber.Ctx, userID string) (int, error) {
		return authPoolRepo.CountPoolsByUser(c.Context(), userID)
	}), authPoolHandler.CreatePool)
	authPools.Get("/", authPoolHandler.ListPools)

	poolByID := authPools.Group("/:poolId")
	poolByID.Get("/", authPoolHandler.GetPool)
	poolByID.Delete("/", authPoolHandler.DeletePool)
	poolByID.Post("/users", authPoolHandler.CreateUser)
	poolByID.Get("/users", authPoolHandler.ListUsers)
	poolByID.Get("/users/:userId", authPoolHandler.GetUser)
	poolByID.Delete("/users/:userId", authPoolHandler.DeleteUser)
	poolByID.Post("/users/:userId/disable", authPoolHandler.DisableUser)
	poolByID.Post("/users/:userId/enable", authPoolHandler.EnableUser)
	poolByID.Get("/users/:userId/roles", authPoolHandler.GetUserRoles)
	poolByID.Post("/users/:userId/roles", authPoolHandler.AssignRoleToUser)
	poolByID.Delete("/users/:userId/roles/:roleName", authPoolHandler.RemoveRoleFromUser)
	poolByID.Post("/roles", authPoolHandler.CreateRole)
	poolByID.Get("/roles", authPoolHandler.ListRoles)
	poolByID.Delete("/roles/:roleName", authPoolHandler.DeleteRole)

	// Public token exchange endpoint (no JWT required — used by pool end-users)
	api.Post("/auth-pools/:poolId/token", authPoolHandler.TokenExchange)

	// Monitoring (Prometheus + Loki + k8s pod metrics)
	var promClient *promclient.Client
	var lokiClient *lokiclient.Client
	if cfg.PrometheusURL != "" {
		promClient = promclient.New(cfg.PrometheusURL)
		slog.Info("Prometheus client configured", "url", cfg.PrometheusURL)
	}
	if cfg.LokiURL != "" {
		lokiClient = lokiclient.New(cfg.LokiURL)
		slog.Info("Loki client configured", "url", cfg.LokiURL)
	}
	monitoringSvc := services.NewMonitoringService(promClient, lokiClient, k8sClient, appRepo)
	monitoringHandler := handlers.NewMonitoringHandler(monitoringSvc)
	appByID.Get("/metrics/overview", monitoringHandler.GetOverview)
	appByID.Get("/metrics/timeseries", monitoringHandler.GetTimeSeries)
	appByID.Get("/logs", monitoringHandler.GetLogs)
	appByID.Get("/logs/stream", monitoringHandler.StreamLogs)
	appByID.Get("/pods", monitoringHandler.GetPods)

	// Business metrics (Prometheus exporter for Grafana dashboards)
	bizMetrics := handlers.NewBusinessMetrics(userRepo, appRepo, dbRepo, planRepo)
	bizMetrics.StartCollector(context.Background())
	app.Get("/metrics", bizMetrics.Handler())

	supportHandler := handlers.NewSupportHandler(supportSvc)

	supportRoutes := protected.Group("/support/tickets")
	supportRoutes.Post("/", supportHandler.CreateTicket)
	supportRoutes.Get("/", supportHandler.ListTickets)
	supportRoutes.Get("/:ticketId", supportHandler.GetTicket)
	supportRoutes.Post("/:ticketId/messages", supportHandler.AddMessage)

	planSvc := services.NewPlanService(planRepo, appRepo, dbRepo, storageRepo, appAuthRepo)
	planSvc.SetGatewayRepo(gwRepo)
	planSvc.SetAuthPoolRepo(authPoolRepo)
	planHandler := handlers.NewPlanHandler(planSvc)
	protected.Get("/plan", planHandler.GetMyPlan)
	protected.Post("/plan/upgrade", planHandler.UpgradePlan)

	// Referral System
	referralHandler := handlers.NewReferralHandler(referralRepo, eventRepo, cfg.AppURL)
	protected.Get("/referral", referralHandler.GetSummary)
	protected.Get("/referral/rewards", referralHandler.ListRewards)
	protected.Post("/referral/share", referralHandler.TrackShare)

	// Stripe Billing (Phase 6)
	var billingRepo ports.BillingRepository
	if pool != nil {
		billingRepo = postgres.NewPostgresBillingRepository(pool)
	} else {
		billingRepo = memory.NewMemoryBillingRepository()
	}
	var stripeAPI stripeClient.StripeAPI
	if cfg.StripeBillingEnabled && cfg.StripeSecretKey != "" {
		stripeAPI = stripeClient.NewClient(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
		planSvc.SetStripeEnabled(true)
		slog.Info("Stripe billing enabled")
	} else {
		slog.Info("Stripe billing disabled, direct plan changes allowed")
	}

	billingSvc := services.NewBillingService(
		stripeAPI, billingRepo, planRepo, appRepo, dbRepo, storageRepo, appAuthRepo,
		cfg.StripeProPriceID, cfg.StripeTeamPriceID, cfg.StripeBusinessPriceID, cfg.BaseDomain,
	)
	billingHandler := handlers.NewBillingHandler(billingSvc)
	protected.Get("/billing", billingHandler.GetBillingStatus)
	protected.Post("/billing/checkout", billingHandler.CreateCheckoutSession)
	protected.Post("/billing/portal", billingHandler.CreatePortalSession)
	protected.Post("/billing/cancel", billingHandler.CancelSubscription)
	protected.Get("/billing/invoices", billingHandler.ListInvoices)

	// Exit Survey (captures feedback before cancellation)
	exitSurveyHandler := handlers.NewExitSurveyHandler(exitSurveyRepo, eventRepo, planRepo)
	protected.Post("/billing/exit-survey", exitSurveyHandler.SubmitAndCancel)

	// Stripe webhook (unauthenticated — uses Stripe signature verification)
	if stripeAPI != nil {
		webhookHandlerStripe := handlers.NewStripeWebhookHandler(billingSvc, stripeAPI)
		webhookHandlerStripe.SetEventBus(eventBus)
		if temporalWfClient != nil {
			webhookHandlerStripe.SetOnPlanChange(func(ctx context.Context, userID, email string, oldTier, newTier entities.PlanTier, stripeSub, stripeCust string) {
				if err := temporalWfClient.StartPlanChange(ctx, zenithTemporal.PlanChangeInput{
					UserID:             userID,
					UserEmail:          email,
					OldTier:            oldTier,
					NewTier:            newTier,
					StripeSubscription: stripeSub,
					StripeCustomer:     stripeCust,
				}); err != nil {
					slog.Error("failed to start PlanOrchestrator", "error", err)
				}
			})
		}
		api.Post("/webhooks/stripe", webhookHandlerStripe.HandleEvent)
	}

	// Custom Domains (Phase 4 — Pro+ only)
	var domainRepo ports.DomainRepository
	if pool != nil {
		domainRepo = postgres.NewPostgresDomainRepository(pool)
	} else {
		domainRepo = memory.NewMemoryDomainRepository()
	}
	deployer.SetDomainRepo(domainRepo)
	domainHandler := handlers.NewDomainHandler(domainRepo, appRepo, planRepo)
	domainHandler.SetDeployer(deployer)
	appByID.Post("/domains", domainHandler.Add)
	appByID.Get("/domains", domainHandler.List)
	appByID.Delete("/domains/:domainId", domainHandler.Delete)
	protected.Get("/domains", domainHandler.ListByUser)

	// API Keys (Phase 6.5)
	var apiKeyRepo ports.APIKeyRepository
	if pool != nil {
		apiKeyRepo = postgres.NewPostgresAPIKeyRepository(pool)
	} else {
		apiKeyRepo = memory.NewMemoryAPIKeyRepository()
	}
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeyRepo, planRepo)
	protected.Post("/api-keys", apiKeyHandler.Create)
	protected.Get("/api-keys", apiKeyHandler.List)
	protected.Delete("/api-keys/:keyId", apiKeyHandler.Delete)

	// Sessions (Phase 6.5)
	var sessionRepo ports.SessionRepository
	if pool != nil {
		sessionRepo = postgres.NewPostgresSessionRepository(pool)
	} else {
		sessionRepo = memory.NewMemorySessionRepository()
	}
	sessionHandler := handlers.NewSessionHandler(sessionRepo)
	protected.Get("/auth/sessions", sessionHandler.List)
	protected.Delete("/auth/sessions/:sessionId", sessionHandler.Revoke)
	protected.Delete("/auth/sessions", sessionHandler.RevokeAll)

	// MFA (Phase 6.5 — Pro+ only)
	var mfaRepo ports.MFARepository
	if pool != nil {
		mfaRepo = postgres.NewPostgresMFARepository(pool)
	} else {
		mfaRepo = memory.NewMemoryMFARepository()
	}
	authSvc.SetMFARepo(mfaRepo)
	mfaHandler := handlers.NewMFAHandler(mfaRepo, planRepo)
	protected.Get("/auth/mfa", mfaHandler.GetStatus)
	protected.Post("/auth/mfa/enable", mfaHandler.Enable)
	protected.Post("/auth/mfa/verify", mfaHandler.Verify)
	protected.Post("/auth/mfa/disable", mfaHandler.Disable)
	protected.Post("/auth/mfa/backup-codes", mfaHandler.RegenerateBackupCodes)

	// User Webhooks (Phase 6.5 — Pro+ only)
	var userWebhookRepo ports.UserWebhookRepository
	if pool != nil {
		userWebhookRepo = postgres.NewPostgresUserWebhookRepository(pool)
	} else {
		userWebhookRepo = memory.NewMemoryUserWebhookRepository()
	}
	userWebhookHandler := handlers.NewUserWebhookHandler(userWebhookRepo, planRepo)
	protected.Post("/webhooks", userWebhookHandler.Create)
	protected.Get("/webhooks", userWebhookHandler.List)
	protected.Put("/webhooks/:webhookId", userWebhookHandler.Update)
	protected.Delete("/webhooks/:webhookId", userWebhookHandler.Delete)
	protected.Get("/webhooks/:webhookId/deliveries", userWebhookHandler.ListDeliveries)

	// Custom Roles / RBAC (Phase 6.5 — Team+ only)
	var roleRepo ports.RoleRepository
	if pool != nil {
		roleRepo = postgres.NewPostgresRoleRepository(pool)
	} else {
		roleRepo = memory.NewMemoryRoleRepository()
	}
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
	var ipRepo ports.IPWhitelistRepository
	if pool != nil {
		ipRepo = postgres.NewPostgresIPWhitelistRepository(pool)
	} else {
		ipRepo = memory.NewMemoryIPWhitelistRepository()
	}
	ipHandler := handlers.NewIPWhitelistHandler(ipRepo, planRepo)
	protected.Post("/settings/ip-whitelist", ipHandler.Add)
	protected.Get("/settings/ip-whitelist", ipHandler.List)
	protected.Delete("/settings/ip-whitelist/:entryId", ipHandler.Delete)

	// Compliance Dashboard (Phase 6.5)
	complianceHandler := handlers.NewComplianceHandler(mfaRepo, ipRepo, planRepo, adminRepo)
	protected.Get("/compliance", complianceHandler.GetStatus)

	// User Audit Log (Business+)
	userAuditHandler := handlers.NewUserAuditHandler(adminRepo, planRepo)
	protected.Get("/audit", userAuditHandler.List)
	protected.Get("/audit/export/csv", userAuditHandler.ExportCSV)
	protected.Get("/audit/export/json", userAuditHandler.ExportJSON)

	// Add-on Marketplace
	addonHandler := handlers.NewAddOnHandler(planRepo)
	protected.Get("/addons", addonHandler.ListCatalog)
	protected.Get("/addons/:addonId", addonHandler.GetAddOn)

	// Registry (Harbor) — scan status + browse
	var harborClient *harborclient.Client
	if cfg.HarborURL != "" && cfg.HarborUser != "" {
		harborClient = harborclient.New(cfg.HarborURL, cfg.HarborUser, cfg.HarborPassword)
		billingSvc.SetHarborClient(harborClient)
	}
	registryHandler := handlers.NewRegistryHandler(harborClient, "zenith-stage")
	protected.Get("/registry/repos", registryHandler.ListRepositories)
	protected.Get("/registry/repos/:name", registryHandler.GetRepository)

	// Pod Exec — SSH-to-pod terminal access (Business+ only)
	podSessionRepo := memory.NewMemoryPodExecSessionRepository()
	podExecHandler := handlers.NewPodExecHandler(k8sClient, appRepo, planRepo, userRepo, podSessionRepo, objStorage, cfg.S3PlatformBucket, "zenith-apps")
	appByID.Use("/pods/:podName/exec", podExecHandler.UpgradeCheck)
	appByID.Get("/pods/:podName/exec", podExecHandler.HandleExec())
	protected.Get("/pod-sessions", podExecHandler.ListSessions)
	protected.Get("/pod-sessions/:sessionId/recording", podExecHandler.GetRecordingURL)

	// WAF Configuration (Business+ only)
	wafHandler := handlers.NewWAFHandler(appRepo, planRepo)
	appByID.Get("/waf/rules", wafHandler.ListRules)
	appByID.Post("/waf/rules", wafHandler.CreateRule)
	appByID.Put("/waf/rules/:ruleId", wafHandler.UpdateRule)
	appByID.Delete("/waf/rules/:ruleId", wafHandler.DeleteRule)

	// Cilium Network Policies (Business+ only)
	netpolHandler := handlers.NewNetworkPolicyHandler(appRepo, planRepo)
	appByID.Get("/network-policies", netpolHandler.ListRules)
	appByID.Post("/network-policies", netpolHandler.CreateRule)
	appByID.Put("/network-policies/:ruleId", netpolHandler.UpdateRule)
	appByID.Delete("/network-policies/:ruleId", netpolHandler.DeleteRule)

	// Custom Alert Rules + Metrics (Business+ only)
	alertsHandler := handlers.NewAlertsHandler(appRepo, planRepo)
	appByID.Get("/alerts", alertsHandler.ListAlertRules)
	appByID.Post("/alerts", alertsHandler.CreateAlertRule)
	appByID.Put("/alerts/:ruleId", alertsHandler.UpdateAlertRule)
	appByID.Delete("/alerts/:ruleId", alertsHandler.DeleteAlertRule)
	appByID.Get("/custom-metrics", alertsHandler.ListMetrics)
	appByID.Post("/custom-metrics", alertsHandler.CreateMetric)
	appByID.Delete("/custom-metrics/:metricId", alertsHandler.DeleteMetric)

	// DPA + White-label Branding (Phase 6.5)
	var brandingRepo ports.BrandingRepository
	if pool != nil {
		brandingRepo = postgres.NewPostgresBrandingRepository(pool)
	} else {
		brandingRepo = memory.NewMemoryBrandingRepository()
	}
	brandingHandler := handlers.NewBrandingHandler(brandingRepo, planRepo)
	protected.Get("/settings/dpa", brandingHandler.GetDPA)
	protected.Post("/settings/dpa/sign", brandingHandler.SignDPA)
	protected.Get("/settings/branding", brandingHandler.GetBranding)
	protected.Put("/settings/branding", brandingHandler.UpdateBranding)
	protected.Post("/settings/domain", brandingHandler.SetDashboardDomain)

	// SSO (Phase 6.5 — Team+ only)
	var ssoRepo ports.SSORepository
	if pool != nil {
		ssoRepo = postgres.NewPostgresSSORepository(pool)
	} else {
		ssoRepo = memory.NewMemorySSORepository()
	}
	ssoHandler := handlers.NewSSOHandler(ssoRepo, planRepo)
	protected.Post("/settings/sso/saml", ssoHandler.ConfigureSAML)
	protected.Post("/settings/sso/oidc", ssoHandler.ConfigureOIDC)
	protected.Get("/settings/sso", ssoHandler.ListConfigs)
	protected.Delete("/settings/sso/:configId", ssoHandler.DeleteConfig)

	// Preview Deployments (Phase 6.5 — Team+ only)
	var previewRepo ports.PreviewRepository
	if pool != nil {
		previewRepo = postgres.NewPostgresPreviewRepository(pool)
	} else {
		previewRepo = memory.NewMemoryPreviewRepository()
	}
	previewHandler := handlers.NewPreviewHandler(previewRepo, appRepo, planRepo)
	appByID.Post("/previews", previewHandler.Create)
	appByID.Get("/previews", previewHandler.List)
	appByID.Delete("/previews/:previewId", previewHandler.Delete)

	// SCIM 2.0 Provisioning (Phase 6.5 — Enterprise only, SaaS only)
	// Protected with admin auth — in production, replace with SCIM bearer token validation
	if cfg.Mode == "saas" {
		scimHandler := handlers.NewSCIMHandler(userRepo, planRepo)
		scim := api.Group("/scim/v2", middleware.RequireAuth(cfg.JWTSecret, tokenBlacklist), middleware.RequireRole(entities.RoleAdmin))
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

	// Notifications + Activity Log
	var notifRepo ports.NotificationRepository
	if pool != nil {
		notifRepo = postgres.NewPostgresNotificationRepository(pool)
	} else {
		notifRepo = memory.NewMemoryNotificationRepository()
	}
	notifHandler := handlers.NewNotificationHandler(notifRepo)
	notifSvc.SetNotificationRepo(notifRepo)
	protected.Get("/notifications", notifHandler.List)
	protected.Post("/notifications/read", notifHandler.MarkRead)
	protected.Get("/activity", notifHandler.ListActivity)

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

	// Admin Support Tickets
	admin.Get("/support/tickets", supportHandler.AdminListTickets)
	admin.Get("/support/tickets/:ticketId", supportHandler.AdminGetTicket)
	admin.Post("/support/tickets/:ticketId/reply", supportHandler.AdminReply)
	admin.Put("/support/tickets/:ticketId/status", supportHandler.AdminUpdateStatus)
	admin.Put("/support/tickets/:ticketId/assign", supportHandler.AdminAssignTicket)

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

	// ── Business Growth Engine ──

	// User Event Tracking (admin analytics)
	userEventHandler := handlers.NewUserEventHandler(eventRepo)
	admin.Get("/events", userEventHandler.ListEvents)
	admin.Get("/events/funnel", userEventHandler.GetFunnel)
	admin.Get("/events/user/:id", userEventHandler.GetUserActivity)
	admin.Get("/surveys", userEventHandler.SurveyInsights)

	// Exit Survey (admin analytics)
	admin.Get("/exit-surveys", exitSurveyHandler.AdminList)
	admin.Get("/exit-surveys/stats", exitSurveyHandler.AdminStats)

	// Email Campaign Stats
	emailStatsHandler := handlers.NewEmailStatsHandler(emailSendRepo)
	admin.Get("/emails/stats", emailStatsHandler.GetStats)

	// Referral Admin
	admin.Get("/referrals", referralHandler.AdminList)

	// ── Mission Control v2 handlers ──

	// War Room + Analytics
	analyticsHandler := handlers.NewAdminAnalyticsHandler(pool, userRepo)
	admin.Get("/war-room", analyticsHandler.GetWarRoom)
	admin.Get("/analytics/revenue", analyticsHandler.GetRevenue)
	admin.Get("/analytics/growth", analyticsHandler.GetGrowth)
	admin.Get("/analytics/usage", analyticsHandler.GetUsageAnalytics)
	admin.Get("/analytics/cohorts", analyticsHandler.GetCohorts)

	// CRM Pipeline
	crmHandler := handlers.NewAdminCRMHandler(pool)
	admin.Get("/crm/pipeline", crmHandler.GetPipeline)
	admin.Get("/crm/health-scores", crmHandler.GetHealthScores)
	admin.Post("/crm/customers/:id/notes", crmHandler.SaveNote)
	admin.Get("/crm/customers/:id/notes", crmHandler.GetNotes)
	admin.Put("/crm/customers/:id/tags", crmHandler.UpdateTags)

	// Services Health
	servicesHandler := handlers.NewAdminServicesHandler(k8sClient)
	admin.Get("/services", servicesHandler.ListServices)
	admin.Get("/services/:name", servicesHandler.GetService)
	admin.Post("/services/:name/restart", servicesHandler.RestartService)

	// Observability (Grafana, Loki, Prometheus, Tempo proxying)
	observabilityHandler := handlers.NewAdminObservabilityHandler(
		cfg.GrafanaURL, cfg.LokiURL, cfg.PrometheusURL, cfg.TempoURL,
	)
	admin.Get("/observability/dashboards", observabilityHandler.ListDashboards)
	admin.Post("/observability/logs/query", observabilityHandler.QueryLogs)
	admin.Get("/observability/logs/labels", observabilityHandler.GetLogLabels)
	admin.Get("/observability/alerts", observabilityHandler.ListAlerts)
	admin.Get("/observability/alerts/stats", observabilityHandler.GetAlertStats)
	admin.Get("/observability/alerts/rules", observabilityHandler.ListAlertRules)
	admin.Post("/observability/alerts/silence", observabilityHandler.CreateSilence)
	admin.Get("/observability/traces", observabilityHandler.SearchTraces)
	admin.Get("/observability/traces/:id", observabilityHandler.GetTrace)

	// Security Ops
	securityHandler := handlers.NewAdminSecurityHandler(pool, k8sClient, harborClient)
	admin.Get("/security/posture", securityHandler.GetPosture)
	admin.Get("/security/policies", securityHandler.ListPolicies)
	admin.Get("/security/policies/stats", securityHandler.GetPolicyStats)
	admin.Get("/security/falco/alerts", securityHandler.ListFalcoAlerts)
	admin.Get("/security/rate-limits", securityHandler.GetRateLimits)
	admin.Get("/security/images", securityHandler.ListImages)
	admin.Get("/security/images/stats", securityHandler.GetImageStats)
	admin.Post("/security/images/:name/scan", securityHandler.TriggerImageScan)
	admin.Get("/security/sessions", securityHandler.ListSessions)
	admin.Delete("/security/sessions/:id", securityHandler.TerminateSession)

	// Platform Ops (Backups, GitOps, Registry, Databases, Storage, Networking, Quality)
	platformOpsHandler := handlers.NewAdminPlatformOpsHandler(pool, k8sClient, harborClient, objStorage)
	admin.Get("/backups", platformOpsHandler.GetBackups)
	admin.Get("/backups/stats", platformOpsHandler.GetBackupStats)
	admin.Get("/backups/velero", platformOpsHandler.ListVeleroSchedules)
	admin.Get("/backups/cnpg", platformOpsHandler.ListCNPGBackups)
	admin.Post("/backups/trigger", platformOpsHandler.TriggerBackup)
	admin.Get("/gitops/apps", platformOpsHandler.ListArgoApps)
	admin.Get("/gitops/stats", platformOpsHandler.GetGitOpsStats)
	admin.Post("/gitops/apps/:name/sync", platformOpsHandler.SyncArgoApp)
	admin.Get("/gitops/apps/:name/history", platformOpsHandler.GetArgoAppHistory)
	admin.Get("/registry/projects", platformOpsHandler.ListRegistryProjects)
	admin.Get("/registry/stats", platformOpsHandler.GetRegistryStats)
	admin.Get("/registry/projects/:name/repos", platformOpsHandler.ListRegistryRepos)
	admin.Get("/databases", platformOpsHandler.ListDatabaseClusters)
	admin.Get("/databases/stats", platformOpsHandler.GetDatabaseStats)
	admin.Get("/databases/:name", platformOpsHandler.GetDatabaseCluster)
	admin.Get("/storage/s3", platformOpsHandler.ListS3Buckets)
	admin.Get("/storage/volumes", platformOpsHandler.ListVolumes)
	admin.Get("/storage/stats", platformOpsHandler.GetStorageStats)
	admin.Get("/networking/dns", platformOpsHandler.ListDNSRecords)
	admin.Get("/networking/routes", platformOpsHandler.ListRoutes)
	admin.Get("/networking/certificates", platformOpsHandler.ListCertificates)
	admin.Get("/quality/metrics", platformOpsHandler.GetQualityMetrics)
	admin.Get("/quality/tickets", platformOpsHandler.GetQualityTickets)

	// Admin Proxy (authenticated reverse proxy to internal services)
	proxyHandler := handlers.NewAdminProxyHandler(map[string]string{
		"grafana":  cfg.GrafanaURL,
		"argocd":   "https://argocd-server.argocd.svc.cluster.local:443",
		"harbor":   cfg.HarborURL,
		"keycloak": cfg.KeycloakURL,
	})
	admin.All("/proxy/:service/*", proxyHandler.Proxy)

	// Email Campaign Worker (drip campaign processor)
	campaignSvc := services.NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, cfg.AppURL)
	if cfg.ResendAPIKey != "" {
		emailFrom := cfg.EmailFrom
		if emailFrom == "" {
			emailFrom = "Zenith <noreply@freezenith.com>"
		}
		campaignSvc.SetEmailSender(resendclient.NewClient(cfg.ResendAPIKey, emailFrom))
	}
	campaignSvc.Start()

	// Dormant Account Cleanup Worker
	dormantSvc := services.NewDormantCleanupService(userRepo, planRepo, appRepo, eventRepo)
	dormantSvc.Start()

	// Admin RBAC (admin user roles and permissions)
	rbacHandler := handlers.NewAdminRBACHandler(pool, userRepo)
	admin.Get("/admin-users", rbacHandler.ListAdminUsers)
	admin.Post("/admin-users", rbacHandler.InviteAdminUser)
	admin.Put("/admin-users/:id/role", rbacHandler.UpdateAdminRole)
	admin.Delete("/admin-users/:id", rbacHandler.RemoveAdminUser)

	// Hetzner Autoscaler (Phase 5 — SaaS only)
	var as *autoscale.Autoscaler
	if cfg.Mode == "saas" {
		var autoscaleRepo ports.AutoscaleRepository
		if pool != nil {
			autoscaleRepo = postgres.NewPostgresAutoscaleRepository(pool)
		} else {
			autoscaleRepo = memory.NewMemoryAutoscaleRepository()
		}
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
			slog.Info("hetzner autoscaler configured")
		} else {
			slog.Info("autoscaler disabled")
		}
	}

	return provisioner, as, tw, tokenBlacklist, eventBus, redisClient
}
