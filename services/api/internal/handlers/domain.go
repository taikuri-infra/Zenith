package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// DomainHandler manages custom domain operations.
type DomainHandler struct {
	domainRepo store.DomainRepository
	appRepo    store.AppRepository
	planRepo   store.UserPlanRepository
}

// NewDomainHandler creates a new DomainHandler.
func NewDomainHandler(domainRepo store.DomainRepository, appRepo store.AppRepository, planRepo store.UserPlanRepository) *DomainHandler {
	return &DomainHandler{domainRepo: domainRepo, appRepo: appRepo, planRepo: planRepo}
}

// Add attaches a custom domain to an app.
// POST /api/v1/apps/:appId/domains
func (h *DomainHandler) Add(c *fiber.Ctx) error {
	appID := c.Params("appId")
	userID, _ := c.Locals("user_id").(string)

	// Verify app ownership
	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}
	if app.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your app")
	}

	// Check plan allows custom domains
	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err == nil && !plan.Limits.CustomDomain {
		return fiber.NewError(fiber.StatusForbidden, "custom domains require Pro plan or higher. Upgrade to add custom domains.")
	}

	var input dto.AddDomainInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Domain == "" {
		return fiber.NewError(fiber.StatusBadRequest, "domain is required")
	}

	domain, err := h.domainRepo.AddDomain(c.Context(), appID, userID, input.Domain)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(toDomainInfo(domain))
}

// List returns all domains for an app.
// GET /api/v1/apps/:appId/domains
func (h *DomainHandler) List(c *fiber.Ctx) error {
	appID := c.Params("appId")

	domains, err := h.domainRepo.ListDomainsByApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([]dto.DomainInfo, len(domains))
	for i, d := range domains {
		result[i] = toDomainInfo(&d)
	}
	return c.JSON(result)
}

// Delete removes a custom domain.
// DELETE /api/v1/apps/:appId/domains/:domainId
func (h *DomainHandler) Delete(c *fiber.Ctx) error {
	domainID := c.Params("domainId")
	userID, _ := c.Locals("user_id").(string)

	domain, err := h.domainRepo.GetDomain(c.Context(), domainID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "domain not found")
	}
	if domain.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your domain")
	}

	if err := h.domainRepo.DeleteDomain(c.Context(), domainID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "domain removed"})
}

// ListByUser returns all custom domains for the authenticated user.
// GET /api/v1/domains
func (h *DomainHandler) ListByUser(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	domains, err := h.domainRepo.ListDomainsByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([]dto.DomainInfo, len(domains))
	for i, d := range domains {
		result[i] = toDomainInfo(&d)
	}
	return c.JSON(result)
}

func toDomainInfo(d *entities.CustomDomain) dto.DomainInfo {
	return dto.DomainInfo{
		ID:        d.ID,
		AppID:     d.AppID,
		Domain:    d.Domain,
		Status:    d.Status,
		TLSReady:  d.TLSReady,
		CreatedAt: d.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
