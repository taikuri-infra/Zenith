package handlers

import (
	"context"
	"log"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

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
		log.Printf("[apps_v2] failed to create app: %v", err)
		return NewInternal("failed to create app")
	}

	// For image deploys, immediately trigger deployment (no build step needed)
	if deploySource == entities.DeploySourceImage && h.pipeline != nil {
		deployment, err := h.appRepo.CreateDeployment(c.Context(), app.ID, req.ImageURL)
		if err != nil {
			log.Printf("[apps_v2] failed to create deployment record: %v", err)
		} else {
			if err := h.pipeline.TriggerImageDeploy(app, deployment, req.ImageURL); err != nil {
				log.Printf("[apps_v2] deploy rejected: %v", err)
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
			log.Printf("[apps_v2] Warning: failed to delete K8s resources for app %s: %v", app.Name, err)
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
