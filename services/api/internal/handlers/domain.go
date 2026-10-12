package handlers

import (
	"context"
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services/deploy"
	"github.com/dotechhq/zenith/services/api/internal/validate"
	"github.com/gofiber/fiber/v2"
)

// AppRedeployer is the subset of deploy.Deployer needed for IngressRoute refresh.
type AppRedeployer interface {
	DeployApp(ctx context.Context, app *entities.App, imageTag string) error
}

// DomainHandler manages custom domain operations.
type DomainHandler struct {
	domainRepo ports.DomainRepository
	appRepo    ports.AppRepository
	planRepo   ports.UserPlanRepository
	deployer   AppRedeployer
	reserved   string // dashboard domain; it and its subdomains can't be custom domains
}

// NewDomainHandler creates a new DomainHandler.
func NewDomainHandler(domainRepo ports.DomainRepository, appRepo ports.AppRepository, planRepo ports.UserPlanRepository) *DomainHandler {
	return &DomainHandler{domainRepo: domainRepo, appRepo: appRepo, planRepo: planRepo}
}

// SetDeployer sets the deployer for IngressRoute refresh on domain changes.
func (h *DomainHandler) SetDeployer(deployer AppRedeployer) {
	h.deployer = deployer
}

// SetReservedDomain sets the dashboard domain (BASE_DOMAIN). It and every
// subdomain of it are refused as custom domains.
func (h *DomainHandler) SetReservedDomain(domain string) { h.reserved = domain }

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
	name, err := validate.CustomDomain(input.Domain, h.reserved)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	domain, err := h.domainRepo.AddDomain(c.Context(), appID, userID, name)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	// Refresh IngressRoute + Certificate to include the new custom domain
	h.refreshIngress(c.Context(), app)

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

	// Refresh IngressRoute + Certificate to remove the deleted domain
	app, appErr := h.appRepo.GetApp(c.Context(), domain.AppID)
	if appErr == nil {
		h.refreshIngress(c.Context(), app)
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

// refreshIngress re-deploys the app so its routes (Traefik labels / IngressRoute)
// include the current domains.
func (h *DomainHandler) refreshIngress(ctx context.Context, app *entities.App) {
	if h.deployer == nil || app == nil {
		return
	}

	imageTag, err := deploy.CurrentImage(ctx, h.appRepo, app.ID)
	if err != nil {
		slog.Warn("no deployed image, skipping route refresh", "app_id", app.ID)
		return
	}
	if err := h.deployer.DeployApp(ctx, app, imageTag); err != nil {
		slog.Warn("failed to refresh app routes", "app_id", app.ID, "error", err)
	}
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
