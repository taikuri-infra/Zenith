package handlers

import (
	"log/slog"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ManagedServiceHandler handles managed service CRUD endpoints.
type ManagedServiceHandler struct {
	projectRepo ports.ProjectRepository
	msRepo      ports.ManagedServiceRepository
	msSvc       *services.ManagedServiceService
}

// NewManagedServiceHandler creates a new ManagedServiceHandler.
func NewManagedServiceHandler(projectRepo ports.ProjectRepository, msRepo ports.ManagedServiceRepository) *ManagedServiceHandler {
	return &ManagedServiceHandler{projectRepo: projectRepo, msRepo: msRepo}
}

// SetService sets the managed service service for K8s provisioning.
func (h *ManagedServiceHandler) SetService(svc *services.ManagedServiceService) {
	h.msSvc = svc
}

type provisionManagedServiceRequest struct {
	ServiceType string `json:"service_type"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	StorageGB   int    `json:"storage_gb"`
}

type managedServiceResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	ServiceType string `json:"service_type"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	StatusMsg   string `json:"status_message,omitempty"`
	InternalHost string `json:"internal_host,omitempty"`
	Port        int    `json:"port,omitempty"`
	StorageGB   int    `json:"storage_gb"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toManagedServiceResponse(ms *entities.ManagedService) managedServiceResponse {
	return managedServiceResponse{
		ID:           ms.ID,
		ProjectID:    ms.ProjectID,
		ServiceType:  string(ms.ServiceType),
		Name:         ms.Name,
		Version:      ms.Version,
		Status:       string(ms.Status),
		StatusMsg:    ms.StatusMessage,
		InternalHost: ms.InternalHost,
		Port:         ms.Port,
		StorageGB:    ms.StorageGB,
		CreatedAt:    ms.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    ms.UpdatedAt.Format(time.RFC3339),
	}
}

// Provision handles POST /projects/:projectId/managed-services
func (h *ManagedServiceHandler) Provision(c *fiber.Ctx) error {
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

	var req provisionManagedServiceRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if req.Name == "" {
		return NewBadRequest("name is required")
	}
	if req.ServiceType == "" {
		return NewBadRequest("service_type is required")
	}

	// Validate service type
	st := entities.ServiceType(req.ServiceType)
	if st != entities.ServiceTypePostgreSQL && st != entities.ServiceTypeRedis {
		return NewBadRequest("service_type must be 'postgresql' or 'redis'")
	}

	if req.Version == "" {
		req.Version = "latest"
	}
	if req.StorageGB <= 0 {
		req.StorageGB = 5
	}

	// Use service layer for K8s provisioning if available
	if h.msSvc != nil {
		var svc *entities.ManagedService
		var provErr error
		switch st {
		case entities.ServiceTypePostgreSQL:
			svc, provErr = h.msSvc.ProvisionPostgreSQL(c.Context(), projectID, userID, req.Name, req.Version, req.StorageGB)
		case entities.ServiceTypeRedis:
			svc, provErr = h.msSvc.ProvisionRedis(c.Context(), projectID, userID, req.Name, req.Version, req.StorageGB)
		}
		if provErr != nil {
			if isAlreadyExists(provErr) {
				return NewConflict(provErr.Error())
			}
			slog.Error("failed to provision managed service", "error", provErr)
			return NewInternal("failed to provision managed service")
		}
		return c.Status(fiber.StatusCreated).JSON(toManagedServiceResponse(svc))
	}

	// Fallback: create DB record only (no K8s provisioning)
	svc := &entities.ManagedService{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		UserID:      userID,
		ServiceType: st,
		Name:        req.Name,
		Version:     req.Version,
		Port:        entities.DefaultPort(st),
		Status:      entities.ManagedServiceProvisioning,
		StorageGB:   req.StorageGB,
	}

	if err := h.msRepo.CreateManagedService(c.Context(), svc); err != nil {
		if isAlreadyExists(err) {
			return NewConflict(err.Error())
		}
		slog.Error("failed to create managed service", "error", err)
		return NewInternal("failed to create managed service")
	}

	return c.Status(fiber.StatusCreated).JSON(toManagedServiceResponse(svc))
}

// List handles GET /projects/:projectId/managed-services
func (h *ManagedServiceHandler) List(c *fiber.Ctx) error {
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

	services, err := h.msRepo.ListManagedServicesByProject(c.Context(), projectID)
	if err != nil {
		slog.Error("failed to list managed services", "error", err)
		return NewInternal("failed to list managed services")
	}

	items := make([]managedServiceResponse, 0, len(services))
	for i := range services {
		items = append(items, toManagedServiceResponse(&services[i]))
	}

	return c.JSON(fiber.Map{
		"items": items,
		"total": len(items),
	})
}

// Get handles GET /projects/:projectId/managed-services/:msId
func (h *ManagedServiceHandler) Get(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	msID := c.Params("msId")
	ms, err := h.msRepo.GetManagedService(c.Context(), msID)
	if err != nil {
		return NewNotFound("managed service not found")
	}

	// Verify project ownership
	project, err := h.projectRepo.GetProject(c.Context(), ms.ProjectID)
	if err != nil || project.UserID != userID {
		return NewForbidden("not your managed service")
	}

	return c.JSON(toManagedServiceResponse(ms))
}

// Delete handles DELETE /projects/:projectId/managed-services/:msId
func (h *ManagedServiceHandler) Delete(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	msID := c.Params("msId")
	ms, err := h.msRepo.GetManagedService(c.Context(), msID)
	if err != nil {
		return NewNotFound("managed service not found")
	}

	// Verify project ownership
	project, err := h.projectRepo.GetProject(c.Context(), ms.ProjectID)
	if err != nil || project.UserID != userID {
		return NewForbidden("not your managed service")
	}

	// Use service layer for K8s cleanup if available
	if h.msSvc != nil {
		if err := h.msSvc.DeleteManagedService(c.Context(), msID); err != nil {
			slog.Error("failed to delete managed service", "error", err)
			return NewInternal("failed to delete managed service")
		}
		return c.JSON(fiber.Map{"message": "managed service deleted"})
	}

	// Fallback: DB-only delete
	if err := h.msRepo.DeleteManagedService(c.Context(), msID); err != nil {
		slog.Error("failed to delete managed service", "error", err)
		return NewInternal("failed to delete managed service")
	}

	return c.JSON(fiber.Map{"message": "managed service deleted"})
}
