package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services/firewall"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// NetworkPolicyHandler manages per-app firewall rules. Rules are stored in the
// repository and enforced as a NetworkPolicy (or CiliumNetworkPolicy) in the
// app's project namespace.
type NetworkPolicyHandler struct {
	appRepo    ports.AppRepository
	planRepo   ports.UserPlanRepository
	repo       ports.FirewallRuleRepository
	applier    *firewall.Applier
	selfHosted bool
}

// NewNetworkPolicyHandler creates a new NetworkPolicyHandler.
func NewNetworkPolicyHandler(appRepo ports.AppRepository, planRepo ports.UserPlanRepository, repo ports.FirewallRuleRepository) *NetworkPolicyHandler {
	return &NetworkPolicyHandler{appRepo: appRepo, planRepo: planRepo, repo: repo}
}

// SetApplier enforces rules in the cluster after every change.
func (h *NetworkPolicyHandler) SetApplier(a *firewall.Applier) { h.applier = a }

// SetSelfHosted turns off plan gating: self-hosted installs get every feature.
func (h *NetworkPolicyHandler) SetSelfHosted(v bool) { h.selfHosted = v }

// requireBusiness checks that the user is on Business plan or higher (SaaS only).
func (h *NetworkPolicyHandler) requireBusiness(c *fiber.Ctx) error {
	if h.selfHosted {
		return nil
	}
	userID, _ := c.Locals("user_id").(string)
	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusForbidden, "could not determine plan")
	}
	if plan.Tier != entities.PlanBusiness && plan.Tier != entities.PlanEnterprise {
		return fiber.NewError(fiber.StatusForbidden, "network policy configuration requires Business plan or higher")
	}
	return nil
}

// ownedApp returns the app in the URL if it belongs to the caller.
func (h *NetworkPolicyHandler) ownedApp(c *fiber.Ctx) (*entities.App, error) {
	if err := h.requireBusiness(c); err != nil {
		return nil, err
	}
	userID, _ := c.Locals("user_id").(string)
	app, err := h.appRepo.GetApp(c.Context(), c.Params("appId"))
	if err != nil || app == nil || app.UserID != userID {
		return nil, fiber.NewError(fiber.StatusNotFound, "app not found")
	}
	return app, nil
}

// ownedRule returns the rule in the URL if it belongs to the app in the URL.
func (h *NetworkPolicyHandler) ownedRule(c *fiber.Ctx, appID string) (*entities.NetworkPolicyRule, error) {
	rule, err := h.repo.Get(c.Context(), c.Params("ruleId"))
	if err != nil || rule.AppID != appID {
		return nil, fiber.NewError(fiber.StatusNotFound, "rule not found")
	}
	return rule, nil
}

func (h *NetworkPolicyHandler) cilium(ctx context.Context) bool {
	return h.applier != nil && h.applier.Cilium(ctx)
}

// enforce applies the app's policy; the rule is already stored, so a failure
// is reported but not rolled back (startup reconcile retries it).
func (h *NetworkPolicyHandler) enforce(c *fiber.Ctx, appID string) error {
	if h.applier == nil {
		return nil
	}
	if err := h.applier.Apply(c.Context(), appID); err != nil {
		slog.Error("firewall policy not applied", "app_id", appID, "error", err)
		return fiber.NewError(fiber.StatusBadGateway, "firewall rule saved but could not be applied; it will be retried")
	}
	return nil
}

// ListRules returns all network policy rules for an app.
// GET /api/v1/apps/:appId/network-policies
func (h *NetworkPolicyHandler) ListRules(c *fiber.Ctx) error {
	app, err := h.ownedApp(c)
	if err != nil {
		return err
	}
	rules, err := h.repo.ListByApp(c.Context(), app.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not list rules")
	}
	return c.JSON(fiber.Map{"rules": rules, "total": len(rules), "cilium": h.cilium(c.Context())})
}

// CreateRule creates a new network policy rule for an app.
// POST /api/v1/apps/:appId/network-policies
func (h *NetworkPolicyHandler) CreateRule(c *fiber.Ctx) error {
	app, err := h.ownedApp(c)
	if err != nil {
		return err
	}

	var input struct {
		Name      string                          `json:"name"`
		Direction entities.NetworkPolicyDirection `json:"direction"`
		Action    entities.NetworkPolicyAction    `json:"action"`
		Enabled   *bool                           `json:"enabled"`
		Priority  int                             `json:"priority"`
		Config    entities.NetworkPolicyConfig    `json:"config"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if input.Direction != entities.NetworkPolicyIngress && input.Direction != entities.NetworkPolicyEgress {
		return fiber.NewError(fiber.StatusBadRequest, "direction must be 'ingress' or 'egress'")
	}
	if input.Action != entities.NetworkPolicyAllow && input.Action != entities.NetworkPolicyDeny {
		return fiber.NewError(fiber.StatusBadRequest, "action must be 'allow' or 'deny'")
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	now := time.Now()
	rule := entities.NetworkPolicyRule{
		ID:        uuid.New().String(),
		UserID:    app.UserID,
		AppID:     app.ID,
		Name:      input.Name,
		Direction: input.Direction,
		Action:    input.Action,
		Enabled:   enabled,
		Priority:  input.Priority,
		Config:    input.Config,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := firewall.Validate(rule, h.cilium(c.Context())); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := h.repo.Create(c.Context(), &rule); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not save rule")
	}
	if err := h.enforce(c, app.ID); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(rule)
}

// UpdateRule updates an existing network policy rule.
// PUT /api/v1/apps/:appId/network-policies/:ruleId
func (h *NetworkPolicyHandler) UpdateRule(c *fiber.Ctx) error {
	app, err := h.ownedApp(c)
	if err != nil {
		return err
	}
	rule, err := h.ownedRule(c, app.ID)
	if err != nil {
		return err
	}

	var input struct {
		Name     *string                       `json:"name"`
		Enabled  *bool                         `json:"enabled"`
		Priority *int                          `json:"priority"`
		Config   *entities.NetworkPolicyConfig `json:"config"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name != nil {
		rule.Name = *input.Name
	}
	if input.Enabled != nil {
		rule.Enabled = *input.Enabled
	}
	if input.Priority != nil {
		rule.Priority = *input.Priority
	}
	if input.Config != nil {
		rule.Config = *input.Config
	}
	rule.UpdatedAt = time.Now()
	if err := firewall.Validate(*rule, h.cilium(c.Context())); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := h.repo.Update(c.Context(), rule); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not save rule")
	}
	if err := h.enforce(c, app.ID); err != nil {
		return err
	}
	return c.JSON(rule)
}

// DeleteRule deletes a network policy rule.
// DELETE /api/v1/apps/:appId/network-policies/:ruleId
func (h *NetworkPolicyHandler) DeleteRule(c *fiber.Ctx) error {
	app, err := h.ownedApp(c)
	if err != nil {
		return err
	}
	rule, err := h.ownedRule(c, app.ID)
	if err != nil {
		return err
	}
	if err := h.repo.Delete(c.Context(), rule.ID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not delete rule")
	}
	if err := h.enforce(c, app.ID); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"message": "rule deleted"})
}
