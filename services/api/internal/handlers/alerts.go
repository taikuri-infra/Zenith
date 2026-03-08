package handlers

import (
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AlertsHandler provides endpoints for managing per-app custom alert rules and metrics (Business+ only).
type AlertsHandler struct {
	appRepo    ports.AppRepository
	planRepo   ports.UserPlanRepository
	mu         sync.RWMutex
	alertRules map[string][]entities.AlertRule    // appID -> rules
	metrics    map[string][]entities.CustomMetric // appID -> metrics
}

// NewAlertsHandler creates a new AlertsHandler.
func NewAlertsHandler(appRepo ports.AppRepository, planRepo ports.UserPlanRepository) *AlertsHandler {
	return &AlertsHandler{
		appRepo:    appRepo,
		planRepo:   planRepo,
		alertRules: make(map[string][]entities.AlertRule),
		metrics:    make(map[string][]entities.CustomMetric),
	}
}

func (h *AlertsHandler) requireBusiness(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusForbidden, "could not determine plan")
	}
	if plan.Tier != entities.PlanBusiness && plan.Tier != entities.PlanEnterprise {
		return fiber.NewError(fiber.StatusForbidden, "custom alerts require Business plan or higher")
	}
	return nil
}

// === Alert Rules ===

// ListAlertRules returns all alert rules for an app.
// GET /api/v1/apps/:appId/alerts
func (h *AlertsHandler) ListAlertRules(c *fiber.Ctx) error {
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
	rules := h.alertRules[appID]
	h.mu.RUnlock()

	if rules == nil {
		rules = []entities.AlertRule{}
	}
	return c.JSON(fiber.Map{"rules": rules, "total": len(rules)})
}

// CreateAlertRule creates a new alert rule for an app.
// POST /api/v1/apps/:appId/alerts
func (h *AlertsHandler) CreateAlertRule(c *fiber.Ctx) error {
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
		Name        string                `json:"name"`
		Metric      string                `json:"metric"`
		Condition   string                `json:"condition"`
		Duration    string                `json:"duration"`
		Severity    entities.AlertSeverity `json:"severity"`
		Description string                `json:"description"`
		Enabled     *bool                 `json:"enabled"`
		NotifyEmail bool                  `json:"notify_email"`
		NotifySlack bool                  `json:"notify_slack"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" || input.Metric == "" || input.Condition == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name, metric, and condition are required")
	}

	switch input.Severity {
	case entities.AlertSeverityCritical, entities.AlertSeverityWarning, entities.AlertSeverityInfo:
		// valid
	default:
		input.Severity = entities.AlertSeverityWarning
	}

	if input.Duration == "" {
		input.Duration = "5m"
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	now := time.Now()
	rule := entities.AlertRule{
		ID:          uuid.New().String(),
		UserID:      userID,
		AppID:       appID,
		Name:        input.Name,
		Enabled:     enabled,
		Metric:      input.Metric,
		Condition:   input.Condition,
		Duration:    input.Duration,
		Severity:    input.Severity,
		Description: input.Description,
		NotifyEmail: input.NotifyEmail,
		NotifySlack: input.NotifySlack,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	h.mu.Lock()
	h.alertRules[appID] = append(h.alertRules[appID], rule)
	h.mu.Unlock()

	return c.Status(fiber.StatusCreated).JSON(rule)
}

// UpdateAlertRule updates an existing alert rule.
// PUT /api/v1/apps/:appId/alerts/:ruleId
func (h *AlertsHandler) UpdateAlertRule(c *fiber.Ctx) error {
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
		Name        *string                `json:"name"`
		Enabled     *bool                  `json:"enabled"`
		Metric      *string                `json:"metric"`
		Condition   *string                `json:"condition"`
		Duration    *string                `json:"duration"`
		Severity    *entities.AlertSeverity `json:"severity"`
		Description *string                `json:"description"`
		NotifyEmail *bool                  `json:"notify_email"`
		NotifySlack *bool                  `json:"notify_slack"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	rules := h.alertRules[appID]
	for i, r := range rules {
		if r.ID == ruleID {
			if input.Name != nil {
				rules[i].Name = *input.Name
			}
			if input.Enabled != nil {
				rules[i].Enabled = *input.Enabled
			}
			if input.Metric != nil {
				rules[i].Metric = *input.Metric
			}
			if input.Condition != nil {
				rules[i].Condition = *input.Condition
			}
			if input.Duration != nil {
				rules[i].Duration = *input.Duration
			}
			if input.Severity != nil {
				rules[i].Severity = *input.Severity
			}
			if input.Description != nil {
				rules[i].Description = *input.Description
			}
			if input.NotifyEmail != nil {
				rules[i].NotifyEmail = *input.NotifyEmail
			}
			if input.NotifySlack != nil {
				rules[i].NotifySlack = *input.NotifySlack
			}
			rules[i].UpdatedAt = time.Now()
			return c.JSON(rules[i])
		}
	}

	return fiber.NewError(fiber.StatusNotFound, "alert rule not found")
}

// DeleteAlertRule deletes an alert rule.
// DELETE /api/v1/apps/:appId/alerts/:ruleId
func (h *AlertsHandler) DeleteAlertRule(c *fiber.Ctx) error {
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

	rules := h.alertRules[appID]
	for i, r := range rules {
		if r.ID == ruleID {
			h.alertRules[appID] = append(rules[:i], rules[i+1:]...)
			return c.JSON(fiber.Map{"message": "alert rule deleted"})
		}
	}

	return fiber.NewError(fiber.StatusNotFound, "alert rule not found")
}

// === Custom Metrics ===

// ListMetrics returns all custom metrics for an app.
// GET /api/v1/apps/:appId/custom-metrics
func (h *AlertsHandler) ListMetrics(c *fiber.Ctx) error {
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
	metrics := h.metrics[appID]
	h.mu.RUnlock()

	if metrics == nil {
		metrics = []entities.CustomMetric{}
	}
	return c.JSON(fiber.Map{"metrics": metrics, "total": len(metrics)})
}

// CreateMetric creates a new custom metric recording rule.
// POST /api/v1/apps/:appId/custom-metrics
func (h *AlertsHandler) CreateMetric(c *fiber.Ctx) error {
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
		Name       string            `json:"name"`
		Expression string            `json:"expression"`
		Labels     map[string]string `json:"labels"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" || input.Expression == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name and expression are required")
	}

	now := time.Now()
	metric := entities.CustomMetric{
		ID:         uuid.New().String(),
		UserID:     userID,
		AppID:      appID,
		Name:       input.Name,
		Expression: input.Expression,
		Labels:     input.Labels,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	h.mu.Lock()
	h.metrics[appID] = append(h.metrics[appID], metric)
	h.mu.Unlock()

	return c.Status(fiber.StatusCreated).JSON(metric)
}

// DeleteMetric deletes a custom metric.
// DELETE /api/v1/apps/:appId/custom-metrics/:metricId
func (h *AlertsHandler) DeleteMetric(c *fiber.Ctx) error {
	if err := h.requireBusiness(c); err != nil {
		return err
	}
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")
	metricID := c.Params("metricId")

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil || app.UserID != userID {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	metrics := h.metrics[appID]
	for i, m := range metrics {
		if m.ID == metricID {
			h.metrics[appID] = append(metrics[:i], metrics[i+1:]...)
			return c.JSON(fiber.Map{"message": "metric deleted"})
		}
	}

	return fiber.NewError(fiber.StatusNotFound, "metric not found")
}
