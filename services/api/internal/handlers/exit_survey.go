package handlers

import (
	"strconv"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// ExitSurveyHandler manages exit survey endpoints.
type ExitSurveyHandler struct {
	surveyRepo ports.ExitSurveyRepository
	eventRepo  ports.UserEventRepository
	planRepo   ports.UserPlanRepository
	billingSvc ExitSurveyBillingService
}

// ExitSurveyBillingService is the subset of billing service needed by exit survey.
type ExitSurveyBillingService interface {
	CancelSubscription(userID string) error
}

func NewExitSurveyHandler(surveyRepo ports.ExitSurveyRepository, eventRepo ports.UserEventRepository, planRepo ports.UserPlanRepository) *ExitSurveyHandler {
	return &ExitSurveyHandler{surveyRepo: surveyRepo, eventRepo: eventRepo, planRepo: planRepo}
}

func (h *ExitSurveyHandler) SetBillingService(svc ExitSurveyBillingService) {
	h.billingSvc = svc
}

type exitSurveyRequest struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

// SubmitAndCancel saves the exit survey then cancels the subscription.
// POST /api/v1/billing/exit-survey
func (h *ExitSurveyHandler) SubmitAndCancel(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req exitSurveyRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Reason == "" {
		return fiber.NewError(fiber.StatusBadRequest, "reason is required")
	}

	// Get current plan tier
	planTier := "free"
	if plan, err := h.planRepo.GetUserPlan(c.Context(), userID); err == nil && plan != nil {
		planTier = string(plan.Tier)
	}

	// Save survey
	survey := &entities.ExitSurvey{
		UserID:   userID,
		Reason:   req.Reason,
		Details:  req.Details,
		PlanTier: planTier,
	}
	if err := h.surveyRepo.Create(c.Context(), survey); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save survey")
	}

	// Track event
	if h.eventRepo != nil {
		go h.eventRepo.Track(c.Context(), &entities.UserEvent{
			UserID:    userID,
			EventType: entities.EventPlanCancel,
			Properties: map[string]interface{}{
				"reason":    req.Reason,
				"plan_tier": planTier,
			},
		})
	}

	// Cancel subscription via billing service
	if h.billingSvc != nil {
		if err := h.billingSvc.CancelSubscription(userID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to cancel subscription")
		}
	}

	return c.JSON(fiber.Map{"message": "subscription canceled", "survey_id": survey.ID})
}

// AdminList returns all exit surveys.
// GET /api/v1/admin/exit-surveys
func (h *ExitSurveyHandler) AdminList(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	surveys, err := h.surveyRepo.List(c.Context(), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list surveys")
	}
	if surveys == nil {
		surveys = []entities.ExitSurvey{}
	}
	return c.JSON(fiber.Map{"items": surveys, "total": len(surveys)})
}

// AdminStats returns aggregated exit survey statistics.
// GET /api/v1/admin/exit-surveys/stats
func (h *ExitSurveyHandler) AdminStats(c *fiber.Ctx) error {
	stats, err := h.surveyRepo.GetStats(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get survey stats")
	}
	return c.JSON(stats)
}
