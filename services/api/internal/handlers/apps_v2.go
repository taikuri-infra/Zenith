package handlers

import (
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// AppHandlerV2 handles app CRUD operations using the AppRepository.
// This replaces the original CRD-based AppHandler for Phase 2.
type AppHandlerV2 struct {
	appRepo    ports.AppRepository
	baseDomain string
}

// NewAppHandlerV2 creates a new AppHandlerV2.
func NewAppHandlerV2(appRepo ports.AppRepository, baseDomain string) *AppHandlerV2 {
	return &AppHandlerV2{appRepo: appRepo, baseDomain: baseDomain}
}

// --- Request/Response types ---

// CreateAppV2Request is the request body for creating a new app.
type CreateAppV2Request struct {
	Name    string `json:"name"`
	RepoURL string `json:"repo_url"`
	Branch  string `json:"branch,omitempty"`
}

// AppV2Response is the API response for an app.
type AppV2Response struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	RepoURL   string          `json:"repo_url"`
	Branch    string          `json:"branch"`
	Framework string          `json:"framework"`
	Status    string          `json:"status"`
	Subdomain string          `json:"subdomain"`
	URL       string          `json:"url"`
	Port      int             `json:"port"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (h *AppHandlerV2) appToResponse(app *entities.App) AppV2Response {
	url := ""
	if app.Subdomain != "" && h.baseDomain != "" {
		url = "https://" + app.Subdomain + "." + h.baseDomain
	}
	return AppV2Response{
		ID:        app.ID,
		Name:      app.Name,
		RepoURL:   app.RepoURL,
		Branch:    app.Branch,
		Framework: string(app.Framework),
		Status:    string(app.Status),
		Subdomain: app.Subdomain,
		URL:       url,
		Port:      app.Port,
		CreatedAt: app.CreatedAt,
		UpdatedAt: app.UpdatedAt,
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
	if req.RepoURL == "" {
		return NewBadRequest("repo_url is required")
	}

	app, err := h.appRepo.CreateApp(c.Context(), &dto.CreateAppInput{
		UserID:  userID.(string),
		Name:    req.Name,
		RepoURL: req.RepoURL,
		Branch:  req.Branch,
	})
	if err != nil {
		if isAlreadyExists(err) {
			return NewConflict(err.Error())
		}
		return NewInternal("failed to create app")
	}

	return c.Status(fiber.StatusCreated).JSON(h.appToResponse(app))
}

// List handles GET /api/v1/apps
func (h *AppHandlerV2) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return NewUnauthorized("authentication required")
	}

	apps, err := h.appRepo.ListAppsByUser(c.Context(), userID.(string))
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

	if err := h.appRepo.DeleteApp(c.Context(), appID); err != nil {
		return NewNotFound("app not found")
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
