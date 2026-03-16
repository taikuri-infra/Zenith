package handlers

import (
	"time"

	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// AIHandler handles AI-powered features.
type AIHandler struct {
	analyzer    *services.AIErrorAnalyzer
	aiUsageRepo ports.AIUsageRepository
	appRepo     ports.AppRepository
	planRepo    ports.UserPlanRepository
	aiClient    *services.AIClient
}

// NewAIHandler creates a new AIHandler.
func NewAIHandler(
	analyzer *services.AIErrorAnalyzer,
	aiUsageRepo ports.AIUsageRepository,
	appRepo ports.AppRepository,
	planRepo ports.UserPlanRepository,
	aiClient *services.AIClient,
) *AIHandler {
	return &AIHandler{
		analyzer:    analyzer,
		aiUsageRepo: aiUsageRepo,
		appRepo:     appRepo,
		planRepo:    planRepo,
		aiClient:    aiClient,
	}
}

type analyzeErrorRequest struct {
	LogLines int `json:"log_lines"`
}

type analyzeErrorResponse struct {
	Problem       string `json:"problem"`
	Cause         string `json:"cause"`
	Fix           string `json:"fix"`
	Confidence    string `json:"confidence"`
	PIIDisclaimer string `json:"pii_disclaimer"`
}

// AnalyzeError handles POST /apps/:appId/ai/analyze-error
func (h *AIHandler) AnalyzeError(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	if !h.aiClient.IsEnabled() {
		return NewBadRequest("AI features are not enabled")
	}

	appID := c.Params("appId")
	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found")
	}

	// Check monthly AI usage limit (based on plan)
	plan, _ := h.planRepo.GetUserPlan(c.Context(), userID)
	monthlyLimit := 5 // free tier default
	if plan != nil {
		switch plan.Tier {
		case "pro":
			monthlyLimit = 50
		case "team":
			monthlyLimit = 200
		case "business", "enterprise":
			monthlyLimit = 1000
		}
	}

	currentUsage, _ := h.aiUsageRepo.GetMonthlyUsage(c.Context(), userID, time.Now())
	if currentUsage >= monthlyLimit {
		return NewBadRequest("AI usage limit reached for this month")
	}

	var req analyzeErrorRequest
	if err := c.BodyParser(&req); err != nil {
		req.LogLines = 100
	}

	analysis, aiResp, err := h.analyzer.AnalyzeError(c.Context(), app.Subdomain, "zenith-apps", req.LogLines)
	if err != nil {
		return NewInternal("AI analysis failed")
	}
	if analysis == nil {
		return NewBadRequest("no logs available for analysis")
	}

	// Record usage
	if aiResp != nil {
		_ = h.aiUsageRepo.RecordUsage(c.Context(), userID, "error_analysis", aiResp.Model, aiResp.TokensIn, aiResp.TokensOut, 0)
	}

	return c.JSON(analyzeErrorResponse{
		Problem:       analysis.Problem,
		Cause:         analysis.Cause,
		Fix:           analysis.Fix,
		Confidence:    analysis.Confidence,
		PIIDisclaimer: analysis.PIIDisclaimer,
	})
}

type aiUsageResponse struct {
	MonthlyUsed  int  `json:"monthly_used"`
	MonthlyLimit int  `json:"monthly_limit"`
	AIEnabled    bool `json:"ai_enabled"`
}

// GetUsage handles GET /ai/usage
func (h *AIHandler) GetUsage(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	monthlyLimit := 5
	plan, _ := h.planRepo.GetUserPlan(c.Context(), userID)
	if plan != nil {
		switch plan.Tier {
		case "pro":
			monthlyLimit = 50
		case "team":
			monthlyLimit = 200
		case "business", "enterprise":
			monthlyLimit = 1000
		}
	}

	currentUsage, _ := h.aiUsageRepo.GetMonthlyUsage(c.Context(), userID, time.Now())

	return c.JSON(aiUsageResponse{
		MonthlyUsed:  currentUsage,
		MonthlyLimit: monthlyLimit,
		AIEnabled:    h.aiClient.IsEnabled(),
	})
}
