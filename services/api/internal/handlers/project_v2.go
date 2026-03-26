package handlers

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// ManagedServiceDeleter cleans up K8s resources for managed services.
type ManagedServiceDeleter interface {
	DeleteManagedService(ctx context.Context, id string) error
}

// ProjectHandlerV2 handles DB-backed project CRUD (replaces legacy CRD-based ProjectHandler).
type ProjectHandlerV2 struct {
	projectRepo ports.ProjectRepository
	appRepo     ports.AppRepository
	msRepo      ports.ManagedServiceRepository
	deployer    AppDeleter
	msDeleter   ManagedServiceDeleter
	onAppDeleted func(ctx context.Context, appID string)
	envHandler  *EnvironmentHandler
	planRepo    ports.UserPlanRepository
}

// NewProjectHandlerV2 creates a new ProjectHandlerV2.
func NewProjectHandlerV2(projectRepo ports.ProjectRepository, appRepo ports.AppRepository, deployer AppDeleter) *ProjectHandlerV2 {
	return &ProjectHandlerV2{projectRepo: projectRepo, appRepo: appRepo, deployer: deployer}
}

// SetManagedServiceDeleter sets the managed service deleter for K8s cleanup on project delete.
func (h *ProjectHandlerV2) SetManagedServiceDeleter(msRepo ports.ManagedServiceRepository, msDeleter ManagedServiceDeleter) {
	h.msRepo = msRepo
	h.msDeleter = msDeleter
}

// SetOnAppDeleted sets a callback invoked after an app within a project is deleted.
func (h *ProjectHandlerV2) SetOnAppDeleted(fn func(ctx context.Context, appID string)) {
	h.onAppDeleted = fn
}

// SetEnvironmentHandler sets the environment handler for auto-creating environments on project creation.
func (h *ProjectHandlerV2) SetEnvironmentHandler(eh *EnvironmentHandler) {
	h.envHandler = eh
}

// SetPlanRepo sets the plan repo for checking user tier on project creation.
func (h *ProjectHandlerV2) SetPlanRepo(planRepo ports.UserPlanRepository) {
	h.planRepo = planRepo
}

type ProjectV2Response struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create handles POST /api/v1/projects
func (h *ProjectHandlerV2) Create(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if req.Name == "" {
		return NewBadRequest("name is required")
	}

	slug := projectSlug(req.Name)
	if slug == "" {
		slug = "project"
	}

	project, err := h.projectRepo.CreateProject(c.Context(), userID, req.Name, slug, req.Description)
	if err != nil {
		if isAlreadyExists(err) || strings.Contains(err.Error(), "already exists") {
			return NewConflict(err.Error())
		}
		slog.Error("failed to create project", "error", err)
		return NewInternal("failed to create project")
	}

	// Auto-create environments (production always, staging for Pro+)
	if h.envHandler != nil {
		includeStaging := false
		if h.planRepo != nil {
			plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
			if err == nil && plan.Tier != entities.PlanFree {
				includeStaging = true
			}
		}
		if _, err := h.envHandler.CreateEnvironmentsForProject(c, project.ID, includeStaging); err != nil {
			slog.Error("failed to create environments", "project", project.ID, "error", err)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(ProjectV2Response{
		ID:          project.ID,
		Name:        project.Name,
		Slug:        project.Slug,
		Description: project.Description,
		Status:      string(project.Status),
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	})
}

// List handles GET /api/v1/projects
func (h *ProjectHandlerV2) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	projects, err := h.projectRepo.ListProjectsByUser(c.Context(), userID)
	if err != nil {
		return NewInternal("failed to list projects")
	}

	items := make([]ProjectV2Response, 0, len(projects))
	for _, p := range projects {
		items = append(items, ProjectV2Response{
			ID:          p.ID,
			Name:        p.Name,
			Slug:        p.Slug,
			Description: p.Description,
			Status:      string(p.Status),
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"items": items,
		"total": len(items),
	})
}

// Get handles GET /api/v1/projects/:projectId
func (h *ProjectHandlerV2) Get(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	projectID := c.Params("projectId")

	project, err := h.projectRepo.GetProject(c.Context(), projectID)
	if err != nil {
		return NewNotFound("project not found")
	}
	if project.UserID != userID {
		return NewForbidden("not your project")
	}

	return c.JSON(ProjectV2Response{
		ID:          project.ID,
		Name:        project.Name,
		Slug:        project.Slug,
		Description: project.Description,
		Status:      string(project.Status),
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	})
}

// Update handles PUT /api/v1/projects/:projectId
func (h *ProjectHandlerV2) Update(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	projectID := c.Params("projectId")

	project, err := h.projectRepo.GetProject(c.Context(), projectID)
	if err != nil {
		return NewNotFound("project not found")
	}
	if project.UserID != userID {
		return NewForbidden("not your project")
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}

	updated, err := h.projectRepo.UpdateProject(c.Context(), projectID, req.Name, req.Description)
	if err != nil {
		return NewInternal("failed to update project")
	}

	// Allow activating a draft project (e.g. after first successful deploy)
	if req.Status != nil && *req.Status == string(entities.ProjectStatusActive) {
		if err := h.projectRepo.UpdateProjectStatus(c.Context(), projectID, entities.ProjectStatusActive); err != nil {
			slog.Error("failed to activate project", "project", projectID, "error", err)
		} else {
			updated.Status = entities.ProjectStatusActive
		}
	}

	return c.JSON(ProjectV2Response{
		ID:          updated.ID,
		Name:        updated.Name,
		Slug:        updated.Slug,
		Description: updated.Description,
		Status:      string(updated.Status),
		CreatedAt:   updated.CreatedAt,
		UpdatedAt:   updated.UpdatedAt,
	})
}

// Delete handles DELETE /api/v1/projects/:projectId
func (h *ProjectHandlerV2) Delete(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	projectID := c.Params("projectId")

	project, err := h.projectRepo.GetProject(c.Context(), projectID)
	if err != nil {
		return NewNotFound("project not found")
	}
	if project.UserID != userID {
		return NewForbidden("not your project")
	}

	// Prevent deletion of last active project (drafts can always be deleted)
	if project.Status != entities.ProjectStatusDraft {
		count, err := h.projectRepo.CountProjectsByUser(c.Context(), userID)
		if err != nil {
			return NewInternal("failed to check project count")
		}
		if count <= 1 {
			return NewBadRequest("cannot delete your only project")
		}
	}

	// Cleanup K8s resources for all managed services in this project
	if h.msRepo != nil && h.msDeleter != nil {
		msList, err := h.msRepo.ListManagedServicesByProject(c.Context(), projectID)
		if err == nil {
			for i := range msList {
				if err := h.msDeleter.DeleteManagedService(context.Background(), msList[i].ID); err != nil {
					slog.Warn("failed to delete K8s resources for managed service", "name", msList[i].Name, "type", msList[i].ServiceType, "error", err)
				}
			}
		}
	}

	// Cleanup K8s resources for all apps in this project
	if h.appRepo != nil && h.deployer != nil {
		apps, err := h.appRepo.ListAppsByProject(c.Context(), projectID)
		if err == nil {
			for i := range apps {
				if err := h.deployer.DeleteApp(context.Background(), &apps[i]); err != nil {
					slog.Warn("failed to delete K8s resources for app", "app_id", apps[i].Name, "error", err)
				}
				if h.onAppDeleted != nil {
					go h.onAppDeleted(context.Background(), apps[i].ID)
				}
			}
		}
	}

	// CASCADE delete (project + all resources via FK)
	if err := h.projectRepo.DeleteProject(c.Context(), projectID); err != nil {
		return NewInternal("failed to delete project")
	}

	return c.JSON(fiber.Map{"message": "project deleted"})
}

// projectSlug generates a URL-safe slug from a project name.
func projectSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	var result []byte
	for _, c := range []byte(slug) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		}
	}
	s := string(result)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	return s
}
