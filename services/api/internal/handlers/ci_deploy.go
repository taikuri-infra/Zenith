package handlers

import (
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// CIDeployHandler handles CI/CD-initiated deploys via deploy tokens.
// Called by the GitHub Action (services/github-action/action.yml) and the zen CLI.
type CIDeployHandler struct {
	appRepo     ports.AppRepository
	projectRepo ports.ProjectRepository
	envRepo     ports.EnvironmentRepository
	pipeline    AppImageDeployer
	baseDomain  string
}

// NewCIDeployHandler creates a new CIDeployHandler.
func NewCIDeployHandler(appRepo ports.AppRepository, projectRepo ports.ProjectRepository, envRepo ports.EnvironmentRepository, pipeline AppImageDeployer, baseDomain string) *CIDeployHandler {
	return &CIDeployHandler{
		appRepo:     appRepo,
		projectRepo: projectRepo,
		envRepo:     envRepo,
		pipeline:    pipeline,
		baseDomain:  baseDomain,
	}
}

type ciDeployRequest struct {
	App         string `json:"app"`         // app name on the platform
	Image       string `json:"image"`       // full image URL with tag
	Environment string `json:"environment"` // "staging" or "production"
	Replicas    int    `json:"replicas"`    // desired replica count (0 = no change)
}

type ciDeployResponse struct {
	DeploymentID string `json:"deployment_id"`
	AppID        string `json:"app_id"`
	AppName      string `json:"app_name"`
	Status       string `json:"status"`
	URL          string `json:"url,omitempty"`
}

// Deploy handles POST /api/v1/deploy
// Auth: DeployToken header ("DeployToken <id>:<secret>")
// Looks up the app by name within the token's project, then triggers an image deploy.
func (h *CIDeployHandler) Deploy(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}
	projectID, _ := c.Locals("project_id").(string)

	var req ciDeployRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if req.App == "" {
		return NewBadRequest("app is required")
	}
	if req.Image == "" {
		return NewBadRequest("image is required")
	}

	// Normalise image ref
	req.Image = normalizeImageRef(req.Image)

	// Find the app by name within the project (or user-wide if no project token)
	var app *entities.App
	if projectID != "" {
		apps, err := h.appRepo.ListAppsByProject(c.Context(), projectID)
		if err != nil {
			return NewInternal("failed to list project apps")
		}
		for i := range apps {
			if apps[i].Name == req.App {
				app = &apps[i]
				break
			}
		}
	} else {
		// No project bound to token — search across all user apps
		apps, err := h.appRepo.ListAppsByUser(c.Context(), userID)
		if err != nil {
			return NewInternal("failed to list apps")
		}
		for i := range apps {
			if apps[i].Name == req.App {
				app = &apps[i]
				break
			}
		}
	}

	if app == nil {
		return NewNotFound("app '" + req.App + "' not found")
	}

	// Resolve target environment if specified
	if req.Environment != "" && h.envRepo != nil && app.ProjectID != "" {
		envName := entities.EnvironmentProduction
		if req.Environment == "staging" {
			envName = entities.EnvironmentStaging
		}
		envs, lErr := h.envRepo.ListEnvironmentsByProject(c.Context(), app.ProjectID)
		if lErr == nil {
			for _, env := range envs {
				if env.Name == envName && app.EnvironmentID != env.ID {
					// Link app to the target environment
					app.EnvironmentID = env.ID
					h.appRepo.UpdateApp(c.Context(), app.ID, &dto.UpdateAppInput{
						EnvironmentID: &env.ID,
					})
					break
				}
			}
		}
	}

	// Update replica count if specified
	if req.Replicas > 0 && req.Replicas != app.Replicas {
		updated, uErr := h.appRepo.UpdateApp(c.Context(), app.ID, &dto.UpdateAppInput{
			Replicas: &req.Replicas,
		})
		if uErr == nil {
			app = updated
		} else {
			slog.Warn("ci-deploy: failed to update replicas", "app", req.App, "error", uErr)
		}
	}

	// Create deployment record
	deployment, err := h.appRepo.CreateDeployment(c.Context(), app.ID, req.Image)
	if err != nil {
		slog.Error("ci-deploy: failed to create deployment", "app", req.App, "error", err)
		return NewInternal("failed to create deployment")
	}

	// Trigger async deploy
	if h.pipeline != nil {
		if err := h.pipeline.TriggerImageDeploy(app, deployment, req.Image); err != nil {
			slog.Warn("ci-deploy: deploy rejected", "app", req.App, "error", err)
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": err.Error(),
				"code":  "DEPLOY_QUEUE_FULL",
			})
		}
	}

	url := ""
	if app.Subdomain != "" && h.baseDomain != "" {
		url = "https://" + app.Subdomain + "." + h.baseDomain
	}

	return c.Status(fiber.StatusAccepted).JSON(ciDeployResponse{
		DeploymentID: deployment.ID,
		AppID:        app.ID,
		AppName:      app.Name,
		Status:       string(deployment.Status),
		URL:          url,
	})
}

// GetDeployment handles GET /api/v1/deployments/:deploymentId
// Returns the current status of a deployment (used by GitHub Action to poll).
func (h *CIDeployHandler) GetDeployment(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	deploymentID := c.Params("deploymentId")
	if deploymentID == "" {
		return NewBadRequest("deployment ID is required")
	}

	deployment, err := h.appRepo.GetDeployment(c.Context(), deploymentID)
	if err != nil {
		return NewNotFound("deployment not found")
	}

	// Ownership check: caller must own the app this deployment belongs to.
	app, err := h.appRepo.GetApp(c.Context(), deployment.AppID)
	if err != nil || app.UserID != userID {
		return NewNotFound("deployment not found") // 404 not 403 — avoids leaking existence
	}

	url := ""
	if app.Subdomain != "" && h.baseDomain != "" {
		url = "https://" + app.Subdomain + "." + h.baseDomain
	}

	return c.JSON(fiber.Map{
		"id":         deployment.ID,
		"app_id":     deployment.AppID,
		"status":     string(deployment.Status),
		"image":      deployment.ImageTag,
		"created_at": deployment.CreatedAt,
		"url":        url,
	})
}
