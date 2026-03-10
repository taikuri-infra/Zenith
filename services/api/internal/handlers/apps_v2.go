package handlers

import (
	"context"
	"log/slog"
	"strings"
	"time"

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

// AppHandlerV2 handles app CRUD operations using the AppRepository.
// This replaces the original CRD-based AppHandler for Phase 2.
type AppHandlerV2 struct {
	appRepo      ports.AppRepository
	projectRepo  ports.ProjectRepository
	baseDomain   string
	deployer     AppDeleter
	pipeline     AppImageDeployer
	onAppDeleted func(ctx context.Context, appID string) // callback for cascade (e.g. stop gateway routes)
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
	ProjectID        string `json:"project_id,omitempty"`
	Name             string `json:"name"`
	DeploySource     string `json:"deploy_source"`
	RepoURL          string `json:"repo_url,omitempty"`
	Branch           string `json:"branch,omitempty"`
	ImageURL         string `json:"image_url,omitempty"`
	Port             int    `json:"port,omitempty"`
	RegistryUsername string `json:"registry_username,omitempty"`
	RegistryPassword string `json:"registry_password,omitempty"`
	AppType          string `json:"app_type,omitempty"`
	Command          string `json:"command,omitempty"`
	CronSchedule     string `json:"cron_schedule,omitempty"`
	EnvVars          []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"env_vars,omitempty"`
}

// AppV2Response is the API response for an app.
type AppV2Response struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Name         string    `json:"name"`
	DeploySource string    `json:"deploy_source"`
	RepoURL      string    `json:"repo_url,omitempty"`
	Branch       string    `json:"branch,omitempty"`
	ImageURL     string    `json:"image_url,omitempty"`
	Framework    string    `json:"framework"`
	Status       string    `json:"status"`
	Subdomain    string    `json:"subdomain"`
	URL          string    `json:"url"`
	Port         int       `json:"port"`
	AppType      string    `json:"app_type"`
	Command      string    `json:"command,omitempty"`
	CronSchedule string    `json:"cron_schedule,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (h *AppHandlerV2) appToResponse(app *entities.App) AppV2Response {
	url := ""
	if app.Subdomain != "" && h.baseDomain != "" {
		url = "https://" + app.Subdomain + "." + h.baseDomain
	}
	appType := string(app.AppType)
	if appType == "" {
		appType = "web"
	}
	return AppV2Response{
		ID:           app.ID,
		ProjectID:    app.ProjectID,
		Name:         app.Name,
		DeploySource: string(app.DeploySource),
		RepoURL:      app.RepoURL,
		Branch:       app.Branch,
		ImageURL:     app.ImageURL,
		Framework:    string(app.Framework),
		Status:       string(app.Status),
		Subdomain:    app.Subdomain,
		URL:          url,
		Port:         app.Port,
		AppType:      appType,
		Command:      app.Command,
		CronSchedule: app.CronSchedule,
		CreatedAt:    app.CreatedAt,
		UpdatedAt:    app.UpdatedAt,
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

	// Resolve project_id: use provided or fall back to default project
	projectID := req.ProjectID
	if projectID == "" && h.projectRepo != nil {
		if dp, err := h.projectRepo.GetDefaultProject(c.Context(), userID.(string)); err == nil {
			projectID = dp.ID
		}
	}

	app, err := h.appRepo.CreateApp(c.Context(), &dto.CreateAppInput{
		UserID:           userID.(string),
		ProjectID:        projectID,
		Name:             req.Name,
		DeploySource:     deploySource,
		RepoURL:          req.RepoURL,
		Branch:           req.Branch,
		ImageURL:         req.ImageURL,
		Port:             req.Port,
		RegistryUsername:  req.RegistryUsername,
		RegistryPassword: req.RegistryPassword,
		AppType:          appType,
		Command:          req.Command,
		CronSchedule:     req.CronSchedule,
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
	appID := c.Params("appId")
	if appID == "" {
		return NewBadRequest("app ID is required")
	}

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found")
	}

	return c.JSON(h.appToResponse(app))
}

// Delete handles DELETE /api/v1/apps/:id
func (h *AppHandlerV2) Delete(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if appID == "" {
		return NewBadRequest("app ID is required")
	}

	// Fetch app first so we can clean up K8s resources
	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found")
	}

	// Clean up K8s resources (Deployment, Service, IngressRoute, HTTPScaledObject)
	if h.deployer != nil && app.Status != entities.AppStatusPending {
		if err := h.deployer.DeleteApp(context.Background(), app); err != nil {
			slog.Warn("failed to delete K8s resources for app", "app_id", app.Name, "error", err)
		}
	}

	if err := h.appRepo.DeleteApp(c.Context(), appID); err != nil {
		return NewNotFound("app not found")
	}

	// Cascade: stop gateway routes pointing to this app
	if h.onAppDeleted != nil {
		go h.onAppDeleted(context.Background(), appID)
	}

	return c.JSON(fiber.Map{"message": "app deleted"})
}

// CheckName handles GET /api/v1/apps/check-name?name=xxx
// Returns availability and the resulting subdomain/URL.
func (h *AppHandlerV2) CheckName(c *fiber.Ctx) error {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		return NewBadRequest("name query parameter is required")
	}

	subdomain := strings.ToLower(strings.ReplaceAll(name, "_", "-"))
	subdomain = strings.ReplaceAll(subdomain, " ", "-")

	url := ""
	if subdomain != "" && h.baseDomain != "" {
		url = "https://" + subdomain + "." + h.baseDomain
	}

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
