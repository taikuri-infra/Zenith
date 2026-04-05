package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupExitSurveyTest() (*fiber.App, *handlers.ExitSurveyHandler, *memory.MemoryExitSurveyRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	surveyRepo := memory.NewMemoryExitSurveyRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewExitSurveyHandler(surveyRepo, eventRepo, planRepo)
	return app, handler, surveyRepo
}

func TestExitSurveySubmitAndCancel(t *testing.T) {
	app, handler, _ := setupExitSurveyTest()
	app.Post("/api/v1/billing/exit-survey", injectUserID("user-1"), handler.SubmitAndCancel)

	body := `{"reason":"too_expensive","details":"Found a cheaper alternative"}`
	req := httptest.NewRequest("POST", "/api/v1/billing/exit-survey", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "subscription canceled" {
		t.Errorf("Expected 'subscription canceled', got '%v'", result["message"])
	}
	if result["survey_id"] == "" {
		t.Error("Expected non-empty survey_id")
	}
}

func TestExitSurveySubmitNoReason(t *testing.T) {
	app, handler, _ := setupExitSurveyTest()
	app.Post("/api/v1/billing/exit-survey", injectUserID("user-1"), handler.SubmitAndCancel)

	body := `{"details":"some details"}`
	req := httptest.NewRequest("POST", "/api/v1/billing/exit-survey", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestExitSurveySubmitInvalidBody(t *testing.T) {
	app, handler, _ := setupExitSurveyTest()
	app.Post("/api/v1/billing/exit-survey", injectUserID("user-1"), handler.SubmitAndCancel)

	req := httptest.NewRequest("POST", "/api/v1/billing/exit-survey", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestExitSurveyAdminList(t *testing.T) {
	app, handler, surveyRepo := setupExitSurveyTest()

	surveyRepo.Create(nil, &entities.ExitSurvey{
		UserID:   "user-1",
		Reason:   "too_expensive",
		Details:  "Cost too high",
		PlanTier: "pro",
	})
	surveyRepo.Create(nil, &entities.ExitSurvey{
		UserID:   "user-2",
		Reason:   "missing_features",
		Details:  "No SSO support",
		PlanTier: "free",
	})

	app.Get("/api/v1/admin/exit-surveys", handler.AdminList)

	req := httptest.NewRequest("GET", "/api/v1/admin/exit-surveys", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.ExitSurvey `json:"items"`
		Total int                   `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 surveys, got %d", result.Total)
	}
}

func TestExitSurveyAdminListEmpty(t *testing.T) {
	app, handler, _ := setupExitSurveyTest()
	app.Get("/api/v1/admin/exit-surveys", handler.AdminList)

	req := httptest.NewRequest("GET", "/api/v1/admin/exit-surveys", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.ExitSurvey `json:"items"`
		Total int                   `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0 surveys, got %d", result.Total)
	}
}

func TestExitSurveyAdminListWithPagination(t *testing.T) {
	app, handler, _ := setupExitSurveyTest()
	app.Get("/api/v1/admin/exit-surveys", handler.AdminList)

	req := httptest.NewRequest("GET", "/api/v1/admin/exit-surveys?limit=10&offset=0", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestExitSurveyAdminStats(t *testing.T) {
	app, handler, surveyRepo := setupExitSurveyTest()

	surveyRepo.Create(nil, &entities.ExitSurvey{UserID: "u1", Reason: "too_expensive", PlanTier: "pro"})
	surveyRepo.Create(nil, &entities.ExitSurvey{UserID: "u2", Reason: "too_expensive", PlanTier: "free"})
	surveyRepo.Create(nil, &entities.ExitSurvey{UserID: "u3", Reason: "missing_features", PlanTier: "pro"})

	app.Get("/api/v1/admin/exit-surveys/stats", handler.AdminStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/exit-surveys/stats", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result entities.ExitSurveyStats
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 3 {
		t.Errorf("Expected 3 total, got %d", result.Total)
	}
	if result.ByReason["too_expensive"] != 2 {
		t.Errorf("Expected 2 for 'too_expensive', got %d", result.ByReason["too_expensive"])
	}
}
