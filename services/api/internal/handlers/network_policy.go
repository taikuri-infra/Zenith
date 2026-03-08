package handlers

import (
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// NetworkPolicyHandler provides endpoints for managing per-app Cilium network policies (Business+ only).
type NetworkPolicyHandler struct {
	appRepo  ports.AppRepository
	planRepo ports.UserPlanRepository
	mu       sync.RWMutex
	policies map[string][]entities.NetworkPolicyRule // appID -> rules (in-memory store)
}

// NewNetworkPolicyHandler creates a new NetworkPolicyHandler.
func NewNetworkPolicyHandler(appRepo ports.AppRepository, planRepo ports.UserPlanRepository) *NetworkPolicyHandler {
	return &NetworkPolicyHandler{
		appRepo:  appRepo,
		planRepo: planRepo,
		policies: make(map[string][]entities.NetworkPolicyRule),
	}
}

// requireBusiness checks that the user is on Business plan or higher.
func (h *NetworkPolicyHandler) requireBusiness(c *fiber.Ctx) error {
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

// ListRules returns all network policy rules for an app.
// GET /api/v1/apps/:appId/network-policies
func (h *NetworkPolicyHandler) ListRules(c *fiber.Ctx) error {
	if err := h.requireBusiness(c); err != nil {
		return err
	}
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil || app.UserID != userID {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}

	h.mu.RLock()
	rules := h.policies[appID]
	h.mu.RUnlock()

	if rules == nil {
		rules = []entities.NetworkPolicyRule{}
	}
	return c.JSON(fiber.Map{"rules": rules, "total": len(rules)})
}

// CreateRule creates a new network policy rule for an app.
// POST /api/v1/apps/:appId/network-policies
func (h *NetworkPolicyHandler) CreateRule(c *fiber.Ctx) error {
	if err := h.requireBusiness(c); err != nil {
		return err
	}
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil || app.UserID != userID {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}

	var input struct {
		Name      string                         `json:"name"`
		Direction entities.NetworkPolicyDirection `json:"direction"`
		Action    entities.NetworkPolicyAction    `json:"action"`
		Enabled   *bool                          `json:"enabled"`
		Priority  int                            `json:"priority"`
		Config    entities.NetworkPolicyConfig    `json:"config"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	// Validate direction
	if input.Direction != entities.NetworkPolicyIngress && input.Direction != entities.NetworkPolicyEgress {
		return fiber.NewError(fiber.StatusBadRequest, "direction must be 'ingress' or 'egress'")
	}
	// Validate action
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
		UserID:    userID,
		AppID:     appID,
		Name:      input.Name,
		Direction: input.Direction,
		Action:    input.Action,
		Enabled:   enabled,
		Priority:  input.Priority,
		Config:    input.Config,
		CreatedAt: now,
		UpdatedAt: now,
	}

	h.mu.Lock()
	h.policies[appID] = append(h.policies[appID], rule)
	h.mu.Unlock()

	return c.Status(fiber.StatusCreated).JSON(rule)
}

// UpdateRule updates an existing network policy rule.
// PUT /api/v1/apps/:appId/network-policies/:ruleId
func (h *NetworkPolicyHandler) UpdateRule(c *fiber.Ctx) error {
	if err := h.requireBusiness(c); err != nil {
		return err
	}
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")
	ruleID := c.Params("ruleId")

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil || app.UserID != userID {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}

	var input struct {
		Name     *string                      `json:"name"`
		Enabled  *bool                        `json:"enabled"`
		Priority *int                         `json:"priority"`
		Config   *entities.NetworkPolicyConfig `json:"config"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	rules := h.policies[appID]
	for i, r := range rules {
		if r.ID == ruleID {
			if input.Name != nil {
				rules[i].Name = *input.Name
			}
			if input.Enabled != nil {
				rules[i].Enabled = *input.Enabled
			}
			if input.Priority != nil {
				rules[i].Priority = *input.Priority
			}
			if input.Config != nil {
				rules[i].Config = *input.Config
			}
			rules[i].UpdatedAt = time.Now()
			return c.JSON(rules[i])
		}
	}

	return fiber.NewError(fiber.StatusNotFound, "rule not found")
}

// DeleteRule deletes a network policy rule.
// DELETE /api/v1/apps/:appId/network-policies/:ruleId
func (h *NetworkPolicyHandler) DeleteRule(c *fiber.Ctx) error {
	if err := h.requireBusiness(c); err != nil {
		return err
	}
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")
	ruleID := c.Params("ruleId")

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil || app.UserID != userID {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	rules := h.policies[appID]
	for i, r := range rules {
		if r.ID == ruleID {
			h.policies[appID] = append(rules[:i], rules[i+1:]...)
			return c.JSON(fiber.Map{"message": "rule deleted"})
		}
	}

	return fiber.NewError(fiber.StatusNotFound, "rule not found")
}
