package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/crypto"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// normalizeImageRef resolves short Docker image references the same way
// the Docker CLI does:
//
//	"nginx"                → "docker.io/library/nginx:latest"
//	"hashicorp/http-echo"  → "docker.io/hashicorp/http-echo:latest"
//	"ghcr.io/foo/bar:v1"   → "ghcr.io/foo/bar:v1"  (already qualified)
func normalizeImageRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ref
	}

	// If it already has a tag with a digest, leave it alone
	// If no tag, append :latest
	addTag := !strings.Contains(ref, ":") || (strings.Count(ref, ":") == 1 && strings.Contains(ref, "://"))

	// Check if the first component contains a dot or colon (registry host)
	// e.g. "ghcr.io/...", "registry.example.com/...", "localhost:5000/..."
	parts := strings.SplitN(ref, "/", 2)
	hasRegistry := len(parts) > 1 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":"))

	if !hasRegistry {
		if !strings.Contains(ref, "/") {
			// Bare name like "nginx" → "docker.io/library/nginx"
			ref = "docker.io/library/" + ref
		} else {
			// User/image like "hashicorp/http-echo" → "docker.io/hashicorp/http-echo"
			ref = "docker.io/" + ref
		}
	}

	if addTag {
		ref += ":latest"
	}

	return ref
}

// AppDeleter is the subset of deploy.Deployer needed for cleanup.
type AppDeleter interface {
	DeleteApp(ctx context.Context, app *entities.App) error
}

// AppImageDeployer triggers image-based deploys.
type AppImageDeployer interface {
	TriggerImageDeploy(app *entities.App, deployment *entities.Deployment, image string) error
}

// AppGatewayService is the subset of GatewayService needed for auto-gateway creation.
type AppGatewayService interface {
	EnsureProjectGateway(ctx context.Context, userID, projectID, projectSlug string) (*entities.Gateway, error)
	AutoCreateRoute(ctx context.Context, gw *entities.Gateway, app *entities.App) error
}

// AppHandlerV2 handles app CRUD operations using the AppRepository.
// This replaces the original CRD-based AppHandler for Phase 2.
type AppHandlerV2 struct {
	appRepo      ports.AppRepository
	projectRepo  ports.ProjectRepository
	planRepo     ports.UserPlanRepository
	baseDomain   string
	deployer     AppDeleter
	pipeline     AppImageDeployer
	gwService    AppGatewayService
	onAppDeleted func(ctx context.Context, appID string) // callback for cascade (e.g. stop gateway routes)
	eventRepo    ports.UserEventRepository
	envCrypto    *crypto.EnvCrypto
}

// NewAppHandlerV2 creates a new AppHandlerV2.
func NewAppHandlerV2(appRepo ports.AppRepository, baseDomain string, deployer AppDeleter, pipeline AppImageDeployer) *AppHandlerV2 {
	return &AppHandlerV2{appRepo: appRepo, baseDomain: baseDomain, deployer: deployer, pipeline: pipeline}
}

// SetProjectRepo configures the project repository for default project resolution.
func (h *AppHandlerV2) SetProjectRepo(repo ports.ProjectRepository) {
	h.projectRepo = repo
}

// SetOnAppDeleted sets a callback invoked after an app is deleted (e.g. to stop gateway routes).
func (h *AppHandlerV2) SetOnAppDeleted(fn func(ctx context.Context, appID string)) {
	h.onAppDeleted = fn
}

// SetGatewayService configures the gateway service for auto-gateway creation.
func (h *AppHandlerV2) SetGatewayService(svc AppGatewayService) {
	h.gwService = svc
}

// SetPlanRepo enables plan limit checking on app creation.
func (h *AppHandlerV2) SetPlanRepo(repo ports.UserPlanRepository) {
	h.planRepo = repo
}

// SetEventRepo enables event tracking on app actions.
func (h *AppHandlerV2) SetEventRepo(repo ports.UserEventRepository) {
	h.eventRepo = repo
}

// SetEnvCrypto enables AES-256-GCM encryption for registry passwords at rest.
func (h *AppHandlerV2) SetEnvCrypto(c *crypto.EnvCrypto) {
	h.envCrypto = c
}

// wellKnownPorts maps popular base image names to their default listening port.
var wellKnownPorts = map[string]int{
	"nginx":           80,
	"httpd":           80,
	"node":            3000,
	"python":          8000,
	"golang":          8080,
	"redis":           6379,
	"postgres":        5432,
	"mysql":           3306,
	"mongo":           27017,
	"traefik":         80,
	"caddy":           80,
	"grafana/grafana":  3000,
	"prom/prometheus":  9090,
}

// resolveWellKnownPort returns the default port for a well-known image, or 0.
func resolveWellKnownPort(imageRef string) int {
	// Strip registry prefix and tag to get the base name
	ref := strings.TrimSpace(imageRef)
	// Remove registry host (contains . or :)
	parts := strings.SplitN(ref, "/", 3)
	var name string
	if len(parts) >= 2 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		// e.g. "docker.io/library/nginx:latest" → "library/nginx:latest"
		name = strings.Join(parts[1:], "/")
	} else {
		name = ref
	}
	// Strip "library/" prefix (Docker Hub official images)
	name = strings.TrimPrefix(name, "library/")
	// Strip tag
	if idx := strings.LastIndex(name, ":"); idx > 0 {
		name = name[:idx]
	}
	if p, ok := wellKnownPorts[name]; ok {
		return p
	}
	return 0
}

// --- Request/Response types ---

// CreateAppV2Request is the request body for creating a new app.
type CreateAppV2Request struct {
	ProjectID        string   `json:"project_id,omitempty"`
	Name             string   `json:"name"`
	DeploySource     string   `json:"deploy_source"`
	RepoURL          string   `json:"repo_url,omitempty"`
	Branch           string   `json:"branch,omitempty"`
	ImageURL         string   `json:"image_url,omitempty"`
	Port             int      `json:"port,omitempty"`
	RegistryUsername string   `json:"registry_username,omitempty"`
	RegistryPassword string   `json:"registry_password,omitempty"`
	AppType          string   `json:"app_type,omitempty"`
	Command          string   `json:"command,omitempty"`
	CronSchedule     string   `json:"cron_schedule,omitempty"`
	Exposure         string   `json:"exposure,omitempty"`         // "public" or "protected"
	HealthCheckPath  string   `json:"health_check_path,omitempty"` // custom liveness/readiness probe path (default "/")
	EnvironmentID    string   `json:"environment_id,omitempty"`    // link to a specific environment
	// DependsOn is a list of K8s service names (app subdomains) this app waits for.
	// The deployer generates init containers for each dependency.
	DependsOn []string `json:"depends_on,omitempty"`
	EnvVars   []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"env_vars,omitempty"`
}

// AppV2Response is the API response for an app.
type AppV2Response struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"project_id"`
	EnvironmentID   string     `json:"environment_id,omitempty"`
	Name            string     `json:"name"`
	DeploySource    string     `json:"deploy_source"`
	RepoURL         string     `json:"repo_url,omitempty"`
	Branch          string     `json:"branch,omitempty"`
	ImageURL        string     `json:"image_url,omitempty"`
	Framework       string     `json:"framework"`
	Status          string     `json:"status"`
	Subdomain       string     `json:"subdomain"`
	URL             string     `json:"url"`
	Port            int        `json:"port"`
	Replicas        int        `json:"replicas"`
	HealthCheckPath string     `json:"health_check_path"`
	AppType         string     `json:"app_type"`
	Command         string     `json:"command,omitempty"`
	CronSchedule    string     `json:"cron_schedule,omitempty"`
	Exposure        string     `json:"exposure"`
	AutoGatewayID   string     `json:"auto_gateway_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

func (h *AppHandlerV2) appToResponse(app *entities.App) AppV2Response {
	url := ""
	if app.Subdomain != "" && h.baseDomain != "" && app.AppType == entities.AppTypeWeb {
		url = "https://" + app.Subdomain + "." + h.baseDomain
	}
	appType := string(app.AppType)
	if appType == "" {
		appType = "web"
	}
	exposure := string(app.Exposure)
	if exposure == "" {
		exposure = "public"
	}
	replicas := app.Replicas
	if replicas == 0 {
		replicas = 1
	}
	healthCheckPath := app.HealthCheckPath
	if healthCheckPath == "" {
		healthCheckPath = "/"
	}
	return AppV2Response{
		ID:              app.ID,
		ProjectID:       app.ProjectID,
		EnvironmentID:   app.EnvironmentID,
		Name:            app.Name,
		DeploySource:    string(app.DeploySource),
		RepoURL:         app.RepoURL,
		Branch:          app.Branch,
		ImageURL:        app.ImageURL,
		Framework:       string(app.Framework),
		Status:          string(app.Status),
		Subdomain:       app.Subdomain,
		URL:             url,
		Port:            app.Port,
		Replicas:        replicas,
		HealthCheckPath: healthCheckPath,
		AppType:         appType,
		Command:         app.Command,
		CronSchedule:    app.CronSchedule,
		Exposure:        exposure,
		AutoGatewayID:   app.AutoGatewayID,
		CreatedAt:       app.CreatedAt,
		UpdatedAt:       app.UpdatedAt,
		DeletedAt:       app.DeletedAt,
	}
}

// Create handles POST /api/v1/apps
func (h *AppHandlerV2) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return NewUnauthorized("authentication required")
	}

	var req CreateAppV2Request
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}

	if req.Name == "" {
		return NewBadRequest("name is required")
	}

	deploySource := entities.DeploySource(req.DeploySource)
	if deploySource == "" {
		deploySource = entities.DeploySourceGit
	}
	if deploySource != entities.DeploySourceGit && deploySource != entities.DeploySourceImage {
		return NewBadRequest("deploy_source must be 'git' or 'image'")
	}

	if deploySource == entities.DeploySourceGit && req.RepoURL == "" {
		return NewBadRequest("repo_url is required for git deploys")
	}
	if deploySource == entities.DeploySourceImage && req.ImageURL == "" {
		return NewBadRequest("image_url is required for image deploys")
	}
	if deploySource == entities.DeploySourceImage {
		req.ImageURL = normalizeImageRef(req.ImageURL)
		// Smart port: if port not specified, try well-known image catalog before default
		if req.Port == 0 {
			if p := resolveWellKnownPort(req.ImageURL); p > 0 {
				req.Port = p
			}
		}
	}

	appType := entities.AppType(req.AppType)
	if appType == "" {
		appType = entities.AppTypeWeb
	}
	if appType != entities.AppTypeWeb && appType != entities.AppTypeWorker && appType != entities.AppTypeCron {
		return NewBadRequest("app_type must be 'web', 'worker', or 'cron'")
	}

	// Check plan app limit before creating
	if h.planRepo != nil {
		plan, planErr := h.planRepo.GetUserPlan(c.Context(), userID.(string))
		if planErr == nil {
			currentApps, _ := h.appRepo.ListAppsByUser(c.Context(), userID.(string))
			if len(currentApps) >= plan.Limits.MaxApps {
				// Track the feature-gated event
				if h.eventRepo != nil {
					go h.eventRepo.Track(context.Background(), &entities.UserEvent{
						UserID:    userID.(string),
						EventType: entities.EventFeatureGated,
						Properties: map[string]interface{}{
							"resource": "apps",
							"current":  len(currentApps),
							"limit":    plan.Limits.MaxApps,
						},
					})
				}
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":        "app limit reached",
					"code":         "PLAN_LIMIT_REACHED",
					"resource":     "apps",
					"current":      len(currentApps),
					"limit":        plan.Limits.MaxApps,
					"upgrade_tier": "pro",
				})
			}
		}
	}

	// Resolve project_id: use provided or fall back to default project
	projectID := req.ProjectID
	if projectID != "" && h.projectRepo != nil {
		proj, pErr := h.projectRepo.GetProject(c.Context(), projectID)
		if pErr != nil {
			return NewNotFound("project")
		}
		if proj.UserID != userID.(string) {
			return NewForbidden("not your project")
		}
	} else if projectID == "" && h.projectRepo != nil {
		if dp, err := h.projectRepo.GetDefaultProject(c.Context(), userID.(string)); err == nil {
			projectID = dp.ID
		}
	}

	// Parse exposure (default: public for web, none for worker/cron)
	exposure := entities.AppExposure(req.Exposure)
	if exposure == "" {
		exposure = entities.ExposurePublic
	}
	if exposure != entities.ExposurePublic && exposure != entities.ExposureProtected {
		return NewBadRequest("exposure must be 'public' or 'protected'")
	}

	// Validate health check path
	healthCheckPath := strings.TrimSpace(req.HealthCheckPath)
	if healthCheckPath != "" && !strings.HasPrefix(healthCheckPath, "/") {
		return NewBadRequest("health_check_path must start with /")
	}
	if len(healthCheckPath) > 512 {
		return NewBadRequest("health_check_path too long (max 512)")
	}

	// Encrypt the registry password before persisting it.
	registryPassword := req.RegistryPassword
	if registryPassword != "" && h.envCrypto != nil {
		encrypted, encErr := h.envCrypto.Encrypt(userID.(string), registryPassword)
		if encErr != nil {
			slog.Error("failed to encrypt registry password", "error", encErr)
			return NewInternal("failed to secure registry credentials")
		}
		registryPassword = encrypted
	}

	app, err := h.appRepo.CreateApp(c.Context(), &dto.CreateAppInput{
		UserID:           userID.(string),
		ProjectID:        projectID,
		EnvironmentID:    req.EnvironmentID,
		Name:             req.Name,
		DeploySource:     deploySource,
		RepoURL:          req.RepoURL,
		Branch:           req.Branch,
		ImageURL:         req.ImageURL,
		Port:             req.Port,
		RegistryUsername: req.RegistryUsername,
		RegistryPassword: registryPassword,
		AppType:          appType,
		Command:          req.Command,
		CronSchedule:     req.CronSchedule,
		Exposure:         exposure,
		HealthCheckPath:  healthCheckPath,
		DependsOn:        req.DependsOn,
	})
	if err != nil {
		if isAlreadyExists(err) {
			return NewConflict(err.Error())
		}
		slog.Error("failed to create app", "error", err)
		return NewInternal("failed to create app")
	}

	// Bulk-insert env vars if provided
	if len(req.EnvVars) > 0 {
		vars := make(map[string]string, len(req.EnvVars))
		for _, ev := range req.EnvVars {
			k := strings.TrimSpace(ev.Key)
			if k != "" {
				vars[k] = ev.Value
			}
		}
		if len(vars) > 0 {
			if err := h.appRepo.SetEnvVars(c.Context(), app.ID, vars); err != nil {
				slog.Warn("failed to set initial env vars", "app_id", app.ID, "error", err)
			}
		}
	}

	// Auto-gateway: for web apps, create per-project gateway and route via APISIX
	if appType == entities.AppTypeWeb && h.gwService != nil && projectID != "" {
		projectSlug := ""
		if h.projectRepo != nil {
			if p, err := h.projectRepo.GetProject(c.Context(), projectID); err == nil {
				projectSlug = p.Slug
			}
		}

		gw, err := h.gwService.EnsureProjectGateway(c.Context(), userID.(string), projectID, projectSlug)
		if err != nil {
			slog.Warn("auto-gateway: failed to ensure project gateway", "project_id", projectID, "error", err)
		} else {
			if err := h.gwService.AutoCreateRoute(c.Context(), gw, app); err != nil {
				slog.Warn("auto-gateway: failed to create route", "app_id", app.ID, "error", err)
			} else {
				// Update app with auto_gateway_id
				app.AutoGatewayID = gw.ID
				h.appRepo.SetAutoGatewayID(c.Context(), app.ID, gw.ID)
			}
		}
	}

	// For image deploys, immediately trigger deployment (no build step needed)
	if deploySource == entities.DeploySourceImage && h.pipeline != nil {
		deployment, err := h.appRepo.CreateDeployment(c.Context(), app.ID, req.ImageURL)
		if err != nil {
			slog.Error("failed to create deployment record", "error", err)
		} else {
			if err := h.pipeline.TriggerImageDeploy(app, deployment, req.ImageURL); err != nil {
				slog.Warn("deploy rejected", "error", err)
			}
		}
	}

	// Track app creation event
	if h.eventRepo != nil {
		go h.eventRepo.Track(context.Background(), &entities.UserEvent{
			UserID:    app.UserID,
			EventType: entities.EventAppCreate,
			Properties: map[string]interface{}{
				"app_id":   app.ID,
				"app_name": app.Name,
			},
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(h.appToResponse(app))
}

// List handles GET /api/v1/apps?project_id=xxx
func (h *AppHandlerV2) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return NewUnauthorized("authentication required")
	}

	var apps []entities.App
	var err error
	projectID := c.Query("project_id")
	if projectID != "" {
		// Verify project ownership before listing
		if h.projectRepo != nil {
			proj, pErr := h.projectRepo.GetProject(c.Context(), projectID)
			if pErr != nil {
				return NewNotFound("project")
			}
			if proj.UserID != userID.(string) {
				return NewForbidden("not your project")
			}
		}
		apps, err = h.appRepo.ListAppsByProject(c.Context(), projectID)
	} else {
		apps, err = h.appRepo.ListAppsByUser(c.Context(), userID.(string))
	}
	if err != nil {
		return NewInternal("failed to list apps")
	}

	items := make([]AppV2Response, 0, len(apps))
	for i := range apps {
		items = append(items, h.appToResponse(&apps[i]))
	}

	return c.JSON(fiber.Map{
		"items": items,
		"total": len(items),
	})
}

// Get handles GET /api/v1/apps/:id
func (h *AppHandlerV2) Get(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	appID := c.Params("appId")
	if appID == "" {
		return NewBadRequest("app ID is required")
	}

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found")
	}
	if app.UserID != userID {
		return NewNotFound("app not found")
	}

	return c.JSON(h.appToResponse(app))
}

// Delete handles DELETE /api/v1/apps/:id
// Uses soft delete — the app is marked as deleted but can be restored.
// Pass ?hard=true to permanently delete (also cleans up K8s resources).
func (h *AppHandlerV2) Delete(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	appID := c.Params("appId")
	if appID == "" {
		return NewBadRequest("app ID is required")
	}

	// Fetch app first so we can clean up K8s resources
	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found")
	}
	if app.UserID != userID {
		return NewNotFound("app not found")
	}

	hard := c.Query("hard") == "true"

	if hard {
		// Hard delete: clean up K8s resources and permanently remove from DB
		if h.deployer != nil && app.Status != entities.AppStatusPending {
			if err := h.deployer.DeleteApp(context.Background(), app); err != nil {
				slog.Warn("failed to delete K8s resources for app", "app_id", app.Name, "error", err)
			}
		}
		if err := h.appRepo.DeleteApp(c.Context(), appID); err != nil {
			return NewNotFound("app not found")
		}
	} else {
		// Soft delete: mark as deleted, stop K8s resources
		if h.deployer != nil && app.Status != entities.AppStatusPending {
			if err := h.deployer.DeleteApp(context.Background(), app); err != nil {
				slog.Warn("failed to delete K8s resources for app", "app_id", app.Name, "error", err)
			}
		}
		if err := h.appRepo.SoftDeleteApp(c.Context(), appID); err != nil {
			return NewInternal("failed to delete app")
		}
	}

	// Cascade: stop gateway routes pointing to this app
	if h.onAppDeleted != nil {
		go h.onAppDeleted(context.Background(), appID)
	}

	// Track app deletion event
	if h.eventRepo != nil {
		go h.eventRepo.Track(context.Background(), &entities.UserEvent{
			UserID:    userID,
			EventType: entities.EventAppDelete,
			Properties: map[string]interface{}{
				"app_id":   app.ID,
				"app_name": app.Name,
				"hard":     hard,
			},
		})
	}

	return c.JSON(fiber.Map{"message": "app deleted"})
}

// Restore handles POST /api/v1/apps/:appId/restore
// Undeletes a soft-deleted app.
func (h *AppHandlerV2) Restore(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	appID := c.Params("appId")
	if appID == "" {
		return NewBadRequest("app ID is required")
	}

	app, err := h.appRepo.RestoreApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found or not deleted")
	}

	// Ownership check
	if app.UserID != userID {
		return NewNotFound("app not found or not deleted")
	}

	return c.JSON(h.appToResponse(app))
}

// ListDeleted handles GET /api/v1/apps/trash
// Returns all soft-deleted apps for the current user.
func (h *AppHandlerV2) ListDeleted(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	apps, err := h.appRepo.ListDeletedAppsByUser(c.Context(), userID)
	if err != nil {
		return NewInternal("failed to list deleted apps")
	}

	items := make([]AppV2Response, 0, len(apps))
	for i := range apps {
		items = append(items, h.appToResponse(&apps[i]))
	}

	return c.JSON(fiber.Map{
		"items": items,
		"total": len(items),
	})
}

// CheckName handles GET /api/v1/apps/check-name?name=xxx&project_id=yyy
// Returns the auto-generated subdomain (with suffix) and URL.
// With the new suffix system, names are always available (deterministic, no collisions).
func (h *AppHandlerV2) CheckName(c *fiber.Ctx) error {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		return NewBadRequest("name query parameter is required")
	}

	projectID := c.Query("project_id")
	if projectID == "" {
		// Fall back to default project
		userID := c.Locals("user_id")
		if userID != nil && h.projectRepo != nil {
			if dp, err := h.projectRepo.GetDefaultProject(c.Context(), userID.(string)); err == nil {
				projectID = dp.ID
			}
		}
	}

	// Generate the same subdomain the CreateApp path uses (DNS-safe slug + suffix)
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, " ", "-")
	// Strip non-DNS characters (match postgres_app.go sanitizeSlug)
	for _, c := range slug {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			slug = strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
					return r
				}
				return -1
			}, slug)
			break
		}
	}
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if len(slug) > 48 {
		slug = strings.TrimRight(slug[:48], "-")
	}
	if slug == "" {
		return NewBadRequest("name produces an empty slug after sanitization")
	}
	hash := sha256.Sum256([]byte(projectID + name))
	suffix := hex.EncodeToString(hash[:2]) // 4 hex chars
	subdomain := slug + "-" + suffix

	url := ""
	if subdomain != "" && h.baseDomain != "" {
		url = "https://" + subdomain + "." + h.baseDomain
	}

	// With deterministic suffix, always available unless exact collision exists
	available := true
	if _, err := h.appRepo.GetAppBySubdomain(c.Context(), subdomain); err == nil {
		available = false
	}

	return c.JSON(fiber.Map{
		"available": available,
		"subdomain": subdomain,
		"url":       url,
	})
}

// isAlreadyExists checks if an error indicates a duplicate resource.
func isAlreadyExists(err error) bool {
	return err != nil && (contains_str(err.Error(), "already exists") || contains_str(err.Error(), "already taken"))
}

func contains_str(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
