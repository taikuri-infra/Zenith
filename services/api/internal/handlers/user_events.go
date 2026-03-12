package handlers

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// UserEventHandler exposes admin endpoints for querying tracked user events.
type UserEventHandler struct {
	eventRepo ports.UserEventRepository
}

func NewUserEventHandler(repo ports.UserEventRepository) *UserEventHandler {
	return &UserEventHandler{eventRepo: repo}
}

// ListEvents returns events filtered by type and/or since date.
// GET /api/v1/admin/events?type=signup&since=2026-03-01&limit=100&offset=0
func (h *UserEventHandler) ListEvents(c *fiber.Ctx) error {
	eventType := c.Query("type")
	sinceStr := c.Query("since")
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	if eventType != "" {
		events, err := h.eventRepo.ListByType(c.Context(), eventType, limit, offset)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to list events")
		}
		if events == nil {
			events = []entities.UserEvent{}
		}
		return c.JSON(fiber.Map{"items": events, "total": len(events)})
	}

	// If no type filter, return counts by type for overview
	since := time.Now().AddDate(0, -1, 0)
	if sinceStr != "" {
		if t, err := time.Parse("2006-01-02", sinceStr); err == nil {
			since = t
		}
	}

	types := []string{
		"signup", "login", "app.create", "app.deploy", "app.delete",
		"db.create", "domain.add", "bucket.create",
		"upgrade.start", "upgrade.complete", "plan.cancel",
		"feature.gated", "trial.start", "trial.end",
		"onboarding.step", "onboarding.done",
		"referral.share", "referral.signup",
	}
	counts := make(map[string]int)
	for _, t := range types {
		count, _ := h.eventRepo.CountByType(c.Context(), t, since)
		if count > 0 {
			counts[t] = count
		}
	}
	return c.JSON(fiber.Map{"counts": counts, "since": since.Format("2006-01-02")})
}

// GetFunnel returns funnel conversion data.
// GET /api/v1/admin/events/funnel?steps=signup,app.create,app.deploy&since=2026-03-01
func (h *UserEventHandler) GetFunnel(c *fiber.Ctx) error {
	stepsStr := c.Query("steps", "signup,app.create,app.deploy")
	sinceStr := c.Query("since")

	steps := strings.Split(stepsStr, ",")
	since := time.Now().AddDate(0, -1, 0)
	if sinceStr != "" {
		if t, err := time.Parse("2006-01-02", sinceStr); err == nil {
			since = t
		}
	}

	data, err := h.eventRepo.GetFunnelData(c.Context(), steps, since)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get funnel data")
	}

	return c.JSON(fiber.Map{"funnel": data, "steps": steps, "since": since.Format("2006-01-02")})
}

// GetUserActivity returns the event timeline for a specific user.
// GET /api/v1/admin/events/user/:id
func (h *UserEventHandler) GetUserActivity(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "user id is required")
	}

	since := time.Now().AddDate(0, -3, 0)
	events, err := h.eventRepo.GetUserActivity(c.Context(), userID, since)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get user activity")
	}
	if events == nil {
		events = []entities.UserEvent{}
	}
	return c.JSON(fiber.Map{"items": events, "user_id": userID})
}

// SurveyInsights aggregates onboarding survey responses.
// GET /api/v1/admin/surveys
func (h *UserEventHandler) SurveyInsights(c *fiber.Ctx) error {
	// Fetch all onboarding.done events (large limit to get everything).
	events, err := h.eventRepo.ListByType(c.Context(), "onboarding.done", 10000, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list survey events")
	}
	if events == nil {
		events = []entities.UserEvent{}
	}

	// Sort by most recent first.
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})

	// Fields to aggregate (single-value).
	singleFields := []string{
		"use_case", "role", "team_size", "current_provider",
		"monthly_spend", "biggest_pain", "expected_traffic",
		"timeline", "most_important", "discovery",
	}

	breakdowns := make(map[string]map[string]int)
	for _, f := range singleFields {
		breakdowns[f] = make(map[string]int)
	}
	breakdowns["stack"] = make(map[string]int)

	type surveyResponse struct {
		UserID          string      `json:"user_id"`
		CreatedAt       time.Time   `json:"created_at"`
		UseCase         string      `json:"use_case,omitempty"`
		Role            string      `json:"role,omitempty"`
		TeamSize        string      `json:"team_size,omitempty"`
		CompanyName     string      `json:"company_name,omitempty"`
		CurrentProvider string      `json:"current_provider,omitempty"`
		MonthlySpend    string      `json:"monthly_spend,omitempty"`
		BiggestPain     string      `json:"biggest_pain,omitempty"`
		ExpectedTraffic string      `json:"expected_traffic,omitempty"`
		Timeline        string      `json:"timeline,omitempty"`
		MostImportant   string      `json:"most_important,omitempty"`
		Stack           []string    `json:"stack,omitempty"`
		Discovery       string      `json:"discovery,omitempty"`
	}

	responses := make([]surveyResponse, 0, len(events))

	for _, ev := range events {
		props := ev.Properties
		resp := surveyResponse{
			UserID:    ev.UserID,
			CreatedAt: ev.CreatedAt,
		}

		// Extract single-value fields.
		for _, f := range singleFields {
			if v, ok := props[f]; ok {
				if s, ok := v.(string); ok && s != "" {
					breakdowns[f][s]++
					switch f {
					case "use_case":
						resp.UseCase = s
					case "role":
						resp.Role = s
					case "team_size":
						resp.TeamSize = s
					case "current_provider":
						resp.CurrentProvider = s
					case "monthly_spend":
						resp.MonthlySpend = s
					case "biggest_pain":
						resp.BiggestPain = s
					case "expected_traffic":
						resp.ExpectedTraffic = s
					case "timeline":
						resp.Timeline = s
					case "most_important":
						resp.MostImportant = s
					case "discovery":
						resp.Discovery = s
					}
				}
			}
		}

		// Extract company_name (not aggregated, just included in response).
		if v, ok := props["company_name"]; ok {
			if s, ok := v.(string); ok {
				resp.CompanyName = s
			}
		}

		// Extract stack (array field).
		if v, ok := props["stack"]; ok {
			switch arr := v.(type) {
			case []interface{}:
				for _, item := range arr {
					if s, ok := item.(string); ok && s != "" {
						resp.Stack = append(resp.Stack, s)
						breakdowns["stack"][s]++
					}
				}
			case []string:
				for _, s := range arr {
					if s != "" {
						resp.Stack = append(resp.Stack, s)
						breakdowns["stack"][s]++
					}
				}
			}
		}

		responses = append(responses, resp)
	}

	return c.JSON(fiber.Map{
		"total_responses": len(responses),
		"responses":       responses,
		"breakdowns":      breakdowns,
	})
}
