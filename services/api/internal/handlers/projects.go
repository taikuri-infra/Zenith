package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ProjectHandler struct {
	k8sClient k8s.Client
}

func NewProjectHandler(client k8s.Client) *ProjectHandler {
	return &ProjectHandler{k8sClient: client}
}

type CreateProjectRequest struct {
	Name   string `json:"name" validate:"required"`
	Plan   string `json:"plan,omitempty"`
	Region string `json:"region,omitempty"`
}

type UpdateProjectRequest struct {
	Name   string `json:"name,omitempty"`
	Plan   string `json:"plan,omitempty"`
}

type ProjectResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Owner       string    `json:"owner"`
	Plan        string    `json:"plan"`
	Region      string    `json:"region"`
	Phase       string    `json:"phase"`
	Namespace   string    `json:"namespace"`
	AppCount    int       `json:"app_count"`
	DBCount     int       `json:"db_count"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *ProjectHandler) Create(c *fiber.Ctx) error {
	var req CreateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}

	if req.Name == "" {
		return NewBadRequest("name is required")
	}

	email, _ := c.Locals("email").(string)
	if email == "" {
		email = "unknown@zenith.dev"
	}

	if req.Plan == "" {
		req.Plan = "free"
	}
	if req.Region == "" {
		req.Region = "fsn1"
	}

	validPlans := map[string]bool{"free": true, "pro": true, "enterprise": true}
	if !validPlans[req.Plan] {
		return NewBadRequest("invalid plan: must be free, pro, or enterprise")
	}

	id := uuid.New().String()[:8]
	slug := toSlug(req.Name)

	spec, _ := json.Marshal(map[string]interface{}{
		"displayName": req.Name,
		"owner":       email,
		"plan":        req.Plan,
		"region":      req.Region,
	})

	crd := &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Project",
		Metadata: k8s.ObjectMeta{
			Name: slug + "-" + id,
			Labels: map[string]string{
				"zenith.dev/project":      slug,
				"zenith.dev/owner":        email,
				"zenith.dev/display-name": req.Name,
			},
		},
		Spec: spec,
	}

	if err := h.k8sClient.CreateCRD(c.Context(), crd); err != nil {
		return NewConflict("project already exists")
	}

	return c.Status(fiber.StatusCreated).JSON(ProjectResponse{
		ID:        slug + "-" + id,
		Name:      req.Name,
		Slug:      slug,
		Owner:     email,
		Plan:      req.Plan,
		Region:    req.Region,
		Phase:     "Pending",
		Namespace: "zenith-" + slug + "-" + id,
		CreatedAt: time.Now(),
	})
}

func (h *ProjectHandler) List(c *fiber.Ctx) error {
	email, _ := c.Locals("email").(string)

	projects, err := h.k8sClient.ListCRDs(c.Context(), "Project", "")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list projects")
	}

	var result []ProjectResponse
	for _, p := range projects {
		ownerLabel := p.Metadata.Labels["zenith.dev/owner"]
		if email != "" && ownerLabel != email {
			continue
		}

		var spec map[string]interface{}
		_ = json.Unmarshal(p.Spec, &spec)

		displayName, _ := spec["displayName"].(string)
		plan, _ := spec["plan"].(string)
		region, _ := spec["region"].(string)

		result = append(result, ProjectResponse{
			ID:        p.Metadata.Name,
			Name:      displayName,
			Slug:      p.Metadata.Labels["zenith.dev/project"],
			Owner:     ownerLabel,
			Plan:      plan,
			Region:    region,
			Phase:     "Active",
			Namespace: "zenith-" + p.Metadata.Name,
		})
	}

	if result == nil {
		result = []ProjectResponse{}
	}

	return c.JSON(dto.ListResponse[ProjectResponse]{
		Items: result,
		Pagination: dto.Pagination{
			Page:     1,
			PageSize: len(result),
			Total:    len(result),
		},
	})
}

func (h *ProjectHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("project id is required")
	}

	proj, err := h.k8sClient.GetCRD(c.Context(), "Project", "", id)
	if err != nil {
		return NewNotFound("project")
	}

	var spec map[string]interface{}
	_ = json.Unmarshal(proj.Spec, &spec)

	displayName, _ := spec["displayName"].(string)
	plan, _ := spec["plan"].(string)
	region, _ := spec["region"].(string)

	return c.JSON(ProjectResponse{
		ID:        proj.Metadata.Name,
		Name:      displayName,
		Slug:      proj.Metadata.Labels["zenith.dev/project"],
		Owner:     proj.Metadata.Labels["zenith.dev/owner"],
		Plan:      plan,
		Region:    region,
		Phase:     "Active",
		Namespace: "zenith-" + proj.Metadata.Name,
	})
}

func (h *ProjectHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("project id is required")
	}

	var req UpdateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}

	proj, err := h.k8sClient.GetCRD(c.Context(), "Project", "", id)
	if err != nil {
		return NewNotFound("project")
	}

	var spec map[string]interface{}
	_ = json.Unmarshal(proj.Spec, &spec)

	if req.Name != "" {
		spec["displayName"] = req.Name
	}
	if req.Plan != "" {
		validPlans := map[string]bool{"free": true, "pro": true, "enterprise": true}
		if !validPlans[req.Plan] {
			return NewBadRequest("invalid plan")
		}
		spec["plan"] = req.Plan
	}

	proj.Spec, _ = json.Marshal(spec)

	if err := h.k8sClient.UpdateCRD(c.Context(), proj); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update project")
	}

	displayName, _ := spec["displayName"].(string)
	plan, _ := spec["plan"].(string)
	region, _ := spec["region"].(string)

	return c.JSON(ProjectResponse{
		ID:     proj.Metadata.Name,
		Name:   displayName,
		Slug:   proj.Metadata.Labels["zenith.dev/project"],
		Owner:  proj.Metadata.Labels["zenith.dev/owner"],
		Plan:   plan,
		Region: region,
		Phase:  "Active",
	})
}

func (h *ProjectHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("project id is required")
	}

	// Verify project exists
	if _, err := h.k8sClient.GetCRD(c.Context(), "Project", "", id); err != nil {
		return NewNotFound("project")
	}

	if err := h.k8sClient.DeleteCRD(c.Context(), "Project", "", id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete project")
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("project %s scheduled for deletion", id),
	})
}

func toSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove non-alphanumeric characters except hyphens
	var result []byte
	for _, c := range []byte(slug) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		}
	}
	return string(result)
}
