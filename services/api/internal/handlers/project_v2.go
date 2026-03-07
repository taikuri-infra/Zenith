package handlers

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// ProjectHandlerV2 handles DB-backed project CRUD (replaces legacy CRD-based ProjectHandler).
type ProjectHandlerV2 struct {
	projectRepo ports.ProjectRepository
	appRepo     ports.AppRepository
	deployer    AppDeleter
	onAppDeleted func(ctx context.Context, appID string)
}

// NewProjectHandlerV2 creates a new ProjectHandlerV2.
func NewProjectHandlerV2(projectRepo ports.ProjectRepository, appRepo ports.AppRepository, deployer AppDeleter) *ProjectHandlerV2 {
	return &ProjectHandlerV2{projectRepo: projectRepo, appRepo: appRepo, deployer: deployer}
}

// SetOnAppDeleted sets a callback invoked after an app within a project is deleted.
func (h *ProjectHandlerV2) SetOnAppDeleted(fn func(ctx context.Context, appID string)) {
	h.onAppDeleted = fn
}

type ProjectV2Response struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
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
		log.Printf("[project_v2] failed to create project: %v", err)
		return NewInternal("failed to create project")
	}

	return c.Status(fiber.StatusCreated).JSON(ProjectV2Response{
		ID:          project.ID,
		Name:        project.Name,
		Slug:        project.Slug,
		Description: project.Description,
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
	}
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}

	updated, err := h.projectRepo.UpdateProject(c.Context(), projectID, req.Name, req.Description)
	if err != nil {
		return NewInternal("failed to update project")
	}

	return c.JSON(ProjectV2Response{
		ID:          updated.ID,
		Name:        updated.Name,
		Slug:        updated.Slug,
		Description: updated.Description,
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

	// Prevent deletion of last project
	count, err := h.projectRepo.CountProjectsByUser(c.Context(), userID)
	if err != nil {
		return NewInternal("failed to check project count")
	}
	if count <= 1 {
		return NewBadRequest("cannot delete your only project")
	}

	// Cleanup K8s resources for all apps in this project
	if h.appRepo != nil && h.deployer != nil {
		apps, err := h.appRepo.ListAppsByProject(c.Context(), projectID)
		if err == nil {
			for i := range apps {
				if err := h.deployer.DeleteApp(context.Background(), &apps[i]); err != nil {
					log.Printf("[project_v2] Warning: failed to delete K8s resources for app %s: %v", apps[i].Name, err)
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
