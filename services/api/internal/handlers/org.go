package handlers

import (
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// OrgNamespaces provisions and removes the Kubernetes namespace behind an Org.
type OrgNamespaces interface {
	Ensure(c *fiber.Ctx, org *entities.Org) error
	Delete(c *fiber.Ctx, org *entities.Org) error
}

// OrgHandler handles Organization endpoints.
type OrgHandler struct {
	orgRepo     ports.OrgRepository
	projectRepo ports.ProjectRepository
	namespaces  OrgNamespaces // optional
}

func NewOrgHandler(orgRepo ports.OrgRepository, projectRepo ports.ProjectRepository) *OrgHandler {
	return &OrgHandler{orgRepo: orgRepo, projectRepo: projectRepo}
}

// SetNamespaces enables namespace provisioning for new and deleted Orgs.
func (h *OrgHandler) SetNamespaces(n OrgNamespaces) { h.namespaces = n }

func (h *OrgHandler) ownedProject(c *fiber.Ctx, projectID string) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}
	p, err := h.projectRepo.GetProject(c.Context(), projectID)
	if err != nil {
		return NewNotFound("project not found")
	}
	if p.UserID != userID {
		return NewForbidden("not your project")
	}
	return nil
}

func (h *OrgHandler) ownedOrg(c *fiber.Ctx) (*entities.Org, error) {
	org, err := h.orgRepo.GetOrg(c.Context(), c.Params("orgId"))
	if err != nil {
		return nil, NewNotFound("org not found")
	}
	if err := h.ownedProject(c, org.ProjectID); err != nil {
		return nil, err
	}
	return org, nil
}

// List handles GET /projects/:projectId/orgs (the default Org is created on first use).
func (h *OrgHandler) List(c *fiber.Ctx) error {
	pid := c.Params("projectId")
	if err := h.ownedProject(c, pid); err != nil {
		return err
	}
	if _, err := h.orgRepo.GetDefaultOrg(c.Context(), pid); err != nil {
		return NewInternal("failed to prepare default org")
	}
	orgs, err := h.orgRepo.ListOrgsByProject(c.Context(), pid)
	if err != nil {
		return NewInternal("failed to list orgs")
	}
	return c.JSON(fiber.Map{"orgs": orgs})
}

// Create handles POST /projects/:projectId/orgs.
func (h *OrgHandler) Create(c *fiber.Ctx) error {
	pid := c.Params("projectId")
	if err := h.ownedProject(c, pid); err != nil {
		return err
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		return NewBadRequest("name is required")
	}
	org, err := h.orgRepo.CreateOrg(c.Context(), pid, strings.TrimSpace(req.Name))
	if err != nil {
		return NewBadRequest(err.Error())
	}
	if h.namespaces != nil {
		if err := h.namespaces.Ensure(c, org); err != nil {
			_ = h.orgRepo.DeleteOrg(c.Context(), org.ID)
			return NewInternal("failed to create org namespace")
		}
	}
	return c.Status(fiber.StatusCreated).JSON(org)
}

// Get handles GET /orgs/:orgId.
func (h *OrgHandler) Get(c *fiber.Ctx) error {
	org, err := h.ownedOrg(c)
	if err != nil {
		return err
	}
	return c.JSON(org)
}

// Rename handles PATCH /orgs/:orgId.
func (h *OrgHandler) Rename(c *fiber.Ctx) error {
	org, err := h.ownedOrg(c)
	if err != nil {
		return err
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		return NewBadRequest("name is required")
	}
	updated, err := h.orgRepo.RenameOrg(c.Context(), org.ID, strings.TrimSpace(req.Name))
	if err != nil {
		return NewInternal("failed to rename org")
	}
	return c.JSON(updated)
}

// Delete handles DELETE /orgs/:orgId?confirm=<slug>. The default Org cannot be deleted.
func (h *OrgHandler) Delete(c *fiber.Ctx) error {
	org, err := h.ownedOrg(c)
	if err != nil {
		return err
	}
	if org.IsDefault {
		return NewBadRequest("the default org cannot be deleted")
	}
	if c.Query("confirm") != org.Slug {
		return NewBadRequest("pass ?confirm=" + org.Slug + " to delete this org and everything in it")
	}
	if err := h.orgRepo.DeleteOrg(c.Context(), org.ID); err != nil {
		return NewBadRequest(err.Error())
	}
	if h.namespaces != nil {
		if err := h.namespaces.Delete(c, org); err != nil {
			return NewInternal("org removed but its namespace could not be deleted")
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}
