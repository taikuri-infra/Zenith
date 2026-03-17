package handlers

import (
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// ComposeHandler handles docker-compose import endpoints.
type ComposeHandler struct {
	projectRepo ports.ProjectRepository
	aiValidator *services.AIComposeValidator
}

// NewComposeHandler creates a new ComposeHandler.
func NewComposeHandler(projectRepo ports.ProjectRepository) *ComposeHandler {
	return &ComposeHandler{projectRepo: projectRepo}
}

// SetAIValidator sets the AI compose validator for smart suggestions.
func (h *ComposeHandler) SetAIValidator(v *services.AIComposeValidator) {
	h.aiValidator = v
}

type importComposeRequest struct {
	ComposeContent string `json:"compose_content"`
}

type importComposeResponse struct {
	Valid           bool                        `json:"valid"`
	Services        []parsedServiceResponse     `json:"services"`
	ManagedServices []parsedManagedResponse     `json:"managed_services"`
	Warnings        []string                    `json:"warnings"`
	Errors          []string                    `json:"errors"`
	AISuggestions   []string                    `json:"ai_suggestions,omitempty"`
}

type parsedServiceResponse struct {
	Name         string                `json:"name"`
	BuildContext string                `json:"build_context,omitempty"`
	Image        string                `json:"image,omitempty"`
	Port         int                   `json:"port"`
	IsPublic     bool                  `json:"is_public"`
	URL          string                `json:"url,omitempty"`
	EnvVars      []parsedEnvVarResponse `json:"env_vars"`
	DependsOn    []string              `json:"depends_on"`
}

type parsedEnvVarResponse struct {
	Key      string `json:"key"`
	Original string `json:"original"`
	Zenith   string `json:"zenith"`
}

type parsedManagedResponse struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Version      string `json:"version"`
	DetectedFrom string `json:"detected_from"`
}

// ImportCompose handles POST /projects/:projectId/import-compose
func (h *ComposeHandler) ImportCompose(c *fiber.Ctx) error {
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

	var req importComposeRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if req.ComposeContent == "" {
		return NewBadRequest("compose_content is required")
	}

	// Parse compose (Layer 1)
	parsed, err := services.ParseCompose(req.ComposeContent, project.Slug, "zenith-apps")
	if err != nil {
		slog.Error("failed to parse compose", "error", err)
		return NewInternal("failed to parse compose file")
	}

	// Validate (Layer 2)
	validationIssues := services.ValidateCompose(parsed)
	parsed.Warnings = append(parsed.Warnings, validationIssues...)

	// Build response
	resp := importComposeResponse{
		Valid:    parsed.Valid,
		Warnings: parsed.Warnings,
		Errors:   parsed.Errors,
	}

	for _, svc := range parsed.Services {
		envVars := make([]parsedEnvVarResponse, 0, len(svc.EnvVars))
		for _, ev := range svc.EnvVars {
			envVars = append(envVars, parsedEnvVarResponse{
				Key:      ev.Key,
				Original: ev.Original,
				Zenith:   ev.Zenith,
			})
		}
		resp.Services = append(resp.Services, parsedServiceResponse{
			Name:         svc.Name,
			BuildContext: svc.BuildContext,
			Image:        svc.Image,
			Port:         svc.Port,
			IsPublic:     svc.IsPublic,
			URL:          svc.URL,
			EnvVars:      envVars,
			DependsOn:    svc.DependsOn,
		})
	}

	for _, ms := range parsed.ManagedServices {
		resp.ManagedServices = append(resp.ManagedServices, parsedManagedResponse{
			Name:         ms.Name,
			Type:         ms.Type,
			Version:      ms.Version,
			DetectedFrom: ms.DetectedFrom,
		})
	}

	// Layer 3: AI suggestions (non-blocking — errors silently return empty)
	if h.aiValidator != nil {
		suggestions := h.aiValidator.ValidateCompose(c.Context(), req.ComposeContent)
		if len(suggestions) > 0 {
			resp.AISuggestions = suggestions
		}
	}

	return c.JSON(resp)
}

