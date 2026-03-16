package handlers

import (
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// ProjectDeployHandler handles deploying all services in a project at once.
type ProjectDeployHandler struct {
	projectRepo ports.ProjectRepository
	appRepo     ports.AppRepository
	msRepo      ports.ManagedServiceRepository
	deployer    AppImageDeployer // uses TriggerImageDeploy
}

// NewProjectDeployHandler creates a new ProjectDeployHandler.
func NewProjectDeployHandler(
	projectRepo ports.ProjectRepository,
	appRepo ports.AppRepository,
	msRepo ports.ManagedServiceRepository,
	deployer AppImageDeployer,
) *ProjectDeployHandler {
	return &ProjectDeployHandler{
		projectRepo: projectRepo,
		appRepo:     appRepo,
		msRepo:      msRepo,
		deployer:    deployer,
	}
}

type serviceDeployStatus struct {
	Name   string `json:"name"`
	AppID  string `json:"app_id"`
	Status string `json:"status"` // deployed, failed, skipped
	Error  string `json:"error,omitempty"`
	URL    string `json:"url,omitempty"`
}

type projectDeployResponse struct {
	ProjectID string                `json:"project_id"`
	Services  []serviceDeployStatus `json:"services"`
	AllOK     bool                  `json:"all_ok"`
}

// DeployProject handles POST /projects/:projectId/deploy
func (h *ProjectDeployHandler) DeployProject(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	projectID := c.Params("projectId")
	project, err := h.projectRepo.GetProject(c.Context(), projectID)
	if err != nil {
		return NewNotFound("project not found")
	}
	if project.UserID != userID {
		return NewForbidden("not your project")
	}

	// Check all managed services are ready
	managedServices, err := h.msRepo.ListManagedServicesByProject(c.Context(), projectID)
	if err != nil {
		slog.Error("failed to list managed services", "error", err)
		return NewInternal("failed to check managed services")
	}
	for _, ms := range managedServices {
		if ms.Status != entities.ManagedServiceReady {
			return NewBadRequest("managed service '" + ms.Name + "' is not ready (status: " + string(ms.Status) + ")")
		}
	}

	// Get all apps in the project
	apps, err := h.appRepo.ListAppsByProject(c.Context(), projectID)
	if err != nil {
		slog.Error("failed to list apps", "error", err)
		return NewInternal("failed to list project apps")
	}

	if len(apps) == 0 {
		return NewBadRequest("no app services found in this project")
	}

	resp := projectDeployResponse{
		ProjectID: projectID,
		Services:  make([]serviceDeployStatus, 0, len(apps)),
		AllOK:     true,
	}

	// Deploy each app
	for i := range apps {
		app := &apps[i]
		status := serviceDeployStatus{
			Name:  app.Name,
			AppID: app.ID,
		}

		// Create a deployment record
		deployment, err := h.appRepo.CreateDeployment(c.Context(), app.ID, "project-deploy")
		if err != nil {
			status.Status = "failed"
			status.Error = "failed to create deployment record"
			resp.AllOK = false
			resp.Services = append(resp.Services, status)
			continue
		}

		// Determine image URL
		imageURL := app.ImageURL
		if imageURL == "" && project.HarborProjectName != "" {
			imageURL = "registry.stage.freezenith.com/" + project.HarborProjectName + "/" + app.Name + ":latest"
		}

		if imageURL == "" {
			status.Status = "skipped"
			status.Error = "no image URL configured"
			resp.Services = append(resp.Services, status)
			continue
		}

		// Trigger deploy
		if h.deployer != nil {
			if err := h.deployer.TriggerImageDeploy(app, deployment, imageURL); err != nil {
				status.Status = "failed"
				status.Error = err.Error()
				resp.AllOK = false
				slog.Error("deploy failed", "app", app.Name, "error", err)
			} else {
				status.Status = "deployed"
				if app.Subdomain != "" {
					status.URL = "https://" + app.Subdomain
				}
			}
		} else {
			// Dev mode: mark as deployed
			status.Status = "deployed"
			h.appRepo.UpdateDeploymentStatus(c.Context(), deployment.ID, entities.DeployStatusActive, "", "")
		}

		resp.Services = append(resp.Services, status)
	}

	return c.JSON(resp)
}
