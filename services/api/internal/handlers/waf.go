package handlers

import (
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// WAFHandler provides endpoints for managing per-app WAF rules (Business+ only).
type WAFHandler struct {
	appRepo  ports.AppRepository
	planRepo ports.UserPlanRepository
	mu       sync.RWMutex
	rules    map[string][]entities.WAFRule // appID -> rules (in-memory store)
}

// NewWAFHandler creates a new WAFHandler.
func NewWAFHandler(appRepo ports.AppRepository, planRepo ports.UserPlanRepository) *WAFHandler {
	return &WAFHandler{
		appRepo:  appRepo,
		planRepo: planRepo,
		rules:    make(map[string][]entities.WAFRule),
	}
}

// requireBusiness checks that the user is on Business plan or higher.
func (h *WAFHandler) requireBusiness(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusForbidden, "could not determine plan")
	}
	if plan.Tier != entities.PlanBusiness && plan.Tier != entities.PlanEnterprise {
		return fiber.NewError(fiber.StatusForbidden, "WAF configuration requires Business plan or higher")
	}
	return nil
}

// ListRules returns all WAF rules for an app.
// GET /api/v1/apps/:appId/waf/rules
func (h *WAFHandler) ListRules(c *fiber.Ctx) error {
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
	rules := h.rules[appID]
	h.mu.RUnlock()

	if rules == nil {
		rules = []entities.WAFRule{}
	}
	return c.JSON(fiber.Map{"rules": rules, "total": len(rules)})
}

// CreateRule creates a new WAF rule for an app.
// POST /api/v1/apps/:appId/waf/rules
func (h *WAFHandler) CreateRule(c *fiber.Ctx) error {
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
		Name     string              `json:"name"`
		Type     entities.WAFRuleType `json:"type"`
		Enabled  *bool               `json:"enabled"`
		Priority int                 `json:"priority"`
		Config   entities.WAFConfig  `json:"config"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	// Validate rule type
	switch input.Type {
	case entities.WAFRuleRateLimit, entities.WAFRuleIPBlock, entities.WAFRuleIPAllow,
		entities.WAFRuleBodyLimit, entities.WAFRuleGeoBlock, entities.WAFRuleHeaderRule:
		// valid
	default:
		return fiber.NewError(fiber.StatusBadRequest, "invalid rule type")
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	now := time.Now()
	rule := entities.WAFRule{
		ID:        uuid.New().String(),
		UserID:    userID,
		AppID:     appID,
		Name:      input.Name,
		Type:      input.Type,
		Enabled:   enabled,
		Priority:  input.Priority,
		Config:    input.Config,
		CreatedAt: now,
		UpdatedAt: now,
	}

	h.mu.Lock()
	h.rules[appID] = append(h.rules[appID], rule)
	h.mu.Unlock()

	return c.Status(fiber.StatusCreated).JSON(rule)
}

// UpdateRule updates an existing WAF rule.
// PUT /api/v1/apps/:appId/waf/rules/:ruleId
func (h *WAFHandler) UpdateRule(c *fiber.Ctx) error {
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
		Name     *string             `json:"name"`
		Enabled  *bool               `json:"enabled"`
		Priority *int                `json:"priority"`
		Config   *entities.WAFConfig `json:"config"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	rules := h.rules[appID]
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

// DeleteRule deletes a WAF rule.
// DELETE /api/v1/apps/:appId/waf/rules/:ruleId
func (h *WAFHandler) DeleteRule(c *fiber.Ctx) error {
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

	rules := h.rules[appID]
	for i, r := range rules {
		if r.ID == ruleID {
			h.rules[appID] = append(rules[:i], rules[i+1:]...)
			return c.JSON(fiber.Map{"message": "rule deleted"})
		}
	}

	return fiber.NewError(fiber.StatusNotFound, "rule not found")
}
