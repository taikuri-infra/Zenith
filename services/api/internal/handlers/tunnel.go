package handlers

import (
	"fmt"
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// TunnelHandler handles tunnel session endpoints for `zen dev`.
type TunnelHandler struct {
	projectRepo ports.ProjectRepository
	envRepo     ports.EnvironmentRepository
	msRepo      ports.ManagedServiceRepository
}

// NewTunnelHandler creates a new TunnelHandler.
func NewTunnelHandler(projectRepo ports.ProjectRepository, envRepo ports.EnvironmentRepository, msRepo ports.ManagedServiceRepository) *TunnelHandler {
	return &TunnelHandler{projectRepo: projectRepo, envRepo: envRepo, msRepo: msRepo}
}

type tunnelServiceInfo struct {
	Name          string `json:"name"`
	ServiceType   string `json:"service_type"`
	ConnectionURL string `json:"connection_url"`
	InternalHost  string `json:"internal_host"`
	Port          int    `json:"port"`
	Status        string `json:"status"`
}

// GetDevInfo handles GET /projects/:projectId/dev-info
// Returns staging environment services connection details for `zen dev`.
func (h *TunnelHandler) GetDevInfo(c *fiber.Ctx) error {
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

	// Get managed services for this project
	services, err := h.msRepo.ListManagedServicesByProject(c.Context(), projectID)
	if err != nil {
		slog.Error("failed to list managed services", "error", err)
		return NewInternal("failed to list services")
	}

	// Build response with connection info
	var svcInfos []tunnelServiceInfo
	for _, svc := range services {
		svcInfos = append(svcInfos, tunnelServiceInfo{
			Name:          svc.Name,
			ServiceType:   string(svc.ServiceType),
			ConnectionURL: svc.ConnectionURL,
			InternalHost:  svc.InternalHost,
			Port:          svc.Port,
			Status:        string(svc.Status),
		})
	}

	// Get environments
	var envInfo []map[string]interface{}
	if h.envRepo != nil {
		envs, err := h.envRepo.ListEnvironmentsByProject(c.Context(), projectID)
		if err == nil {
			for _, env := range envs {
				envInfo = append(envInfo, map[string]interface{}{
					"id":     env.ID,
					"name":   env.Name,
					"slug":   env.Slug,
					"status": env.Status,
				})
			}
		}
	}

	return c.JSON(fiber.Map{
		"project_id":       projectID,
		"project_name":     project.Name,
		"managed_services": svcInfos,
		"environments":     envInfo,
	})
}

// CreateTunnel handles POST /projects/:projectId/environments/:envId/tunnels
// Creates a tunnel session to staging services.
// TODO: Implement WebSocket-based tunneling for direct port access.
func (h *TunnelHandler) CreateTunnel(c *fiber.Ctx) error {
	_ = c.Locals("user_id")

	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"message": "Tunnel sessions not yet implemented. Use connection URLs from /dev-info endpoint.",
		"hint":    fmt.Sprintf("GET /api/v1/projects/%s/dev-info returns connection details for managed services", c.Params("projectId")),
	})
}
