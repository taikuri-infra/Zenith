package handlers

import (
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/adapters/harborclient"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// ImageStatusHandler checks whether images have been pushed to Harbor.
type ImageStatusHandler struct {
	projectRepo ports.ProjectRepository
	appRepo     ports.AppRepository
	harbor      *harborclient.Client
}

// NewImageStatusHandler creates a new ImageStatusHandler.
func NewImageStatusHandler(projectRepo ports.ProjectRepository, appRepo ports.AppRepository, harbor *harborclient.Client) *ImageStatusHandler {
	return &ImageStatusHandler{projectRepo: projectRepo, appRepo: appRepo, harbor: harbor}
}

type imageServiceStatus struct {
	Name    string `json:"name"`
	AppID   string `json:"app_id"`
	Pushed  bool   `json:"pushed"`
	Tag     string `json:"tag,omitempty"`
}

type imageStatusResponse struct {
	AllReady bool                 `json:"all_ready"`
	Services []imageServiceStatus `json:"services"`
}

// GetImageStatus handles GET /projects/:projectId/images/status
func (h *ImageStatusHandler) GetImageStatus(c *fiber.Ctx) error {
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

	// Get all apps in this project
	apps, err := h.appRepo.ListAppsByProject(c.Context(), projectID)
	if err != nil {
		slog.Error("failed to list apps", "error", err)
		return NewInternal("failed to list project apps")
	}

	resp := imageStatusResponse{
		AllReady: true,
		Services: make([]imageServiceStatus, 0, len(apps)),
	}

	// Check Harbor for each app
	harborProject := project.HarborProjectName
	if harborProject == "" {
		harborProject = project.Slug
	}

	for _, app := range apps {
		status := imageServiceStatus{
			Name:  app.Name,
			AppID: app.ID,
		}

		if h.harbor != nil {
			repos, err := h.harbor.ListRepositories(c.Context(), harborProject)
			if err == nil {
				for _, repo := range repos {
					repoName := harborProject + "/" + app.Name
					if repo.Name == repoName && repo.ArtifactCount > 0 {
						status.Pushed = true
						// Get latest tag
						artifacts, err := h.harbor.ListArtifacts(c.Context(), harborProject, app.Name, false)
						if err == nil && len(artifacts) > 0 {
							for _, a := range artifacts {
								if len(a.Tags) > 0 {
									status.Tag = a.Tags[0].Name
									break
								}
							}
						}
						break
					}
				}
			}
		}

		if !status.Pushed {
			resp.AllReady = false
		}

		resp.Services = append(resp.Services, status)
	}

	return c.JSON(resp)
}
