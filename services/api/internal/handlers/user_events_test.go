package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupUserEventTest() (*fiber.App, *handlers.UserEventHandler, *memory.MemoryUserEventRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	eventRepo := memory.NewMemoryUserEventRepository()
	handler := handlers.NewUserEventHandler(eventRepo)
	return app, handler, eventRepo
}

func TestUserEventListByType(t *testing.T) {
	app, handler, eventRepo := setupUserEventTest()

	eventRepo.Track(nil, &entities.UserEvent{
		UserID:    "user-1",
		EventType: "signup",
	})
	eventRepo.Track(nil, &entities.UserEvent{
		UserID:    "user-2",
		EventType: "signup",
	})

	app.Get("/api/v1/admin/events", handler.ListEvents)

	req := httptest.NewRequest("GET", "/api/v1/admin/events?type=signup", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.UserEvent `json:"items"`
		Total int                  `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 events, got %d", result.Total)
	}
}

func TestUserEventListOverview(t *testing.T) {
	app, handler, eventRepo := setupUserEventTest()

	eventRepo.Track(nil, &entities.UserEvent{
		UserID:    "user-1",
		EventType: "signup",
	})
	eventRepo.Track(nil, &entities.UserEvent{
		UserID:    "user-1",
		EventType: "app.create",
	})

	app.Get("/api/v1/admin/events", handler.ListEvents)

	// No type filter returns counts overview
	req := httptest.NewRequest("GET", "/api/v1/admin/events", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Counts map[string]int `json:"counts"`
		Since  string         `json:"since"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Counts["signup"] != 1 {
		t.Errorf("Expected 1 signup count, got %d", result.Counts["signup"])
	}
}

func TestUserEventListByTypeEmpty(t *testing.T) {
	app, handler, _ := setupUserEventTest()
	app.Get("/api/v1/admin/events", handler.ListEvents)

	req := httptest.NewRequest("GET", "/api/v1/admin/events?type=signup", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.UserEvent `json:"items"`
		Total int                  `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0, got %d", result.Total)
	}
}

func TestUserEventGetFunnel(t *testing.T) {
	app, handler, eventRepo := setupUserEventTest()

	eventRepo.Track(nil, &entities.UserEvent{UserID: "user-1", EventType: "signup"})
	eventRepo.Track(nil, &entities.UserEvent{UserID: "user-1", EventType: "app.create"})
	eventRepo.Track(nil, &entities.UserEvent{UserID: "user-2", EventType: "signup"})

	app.Get("/api/v1/admin/events/funnel", handler.GetFunnel)

	req := httptest.NewRequest("GET", "/api/v1/admin/events/funnel?steps=signup,app.create", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Funnel map[string]int `json:"funnel"`
		Steps  []string       `json:"steps"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Funnel["signup"] != 2 {
		t.Errorf("Expected 2 for signup step, got %d", result.Funnel["signup"])
	}
	if result.Funnel["app.create"] != 1 {
		t.Errorf("Expected 1 for app.create step, got %d", result.Funnel["app.create"])
	}
}

func TestUserEventGetUserActivity(t *testing.T) {
	app, handler, eventRepo := setupUserEventTest()

	eventRepo.Track(nil, &entities.UserEvent{UserID: "user-1", EventType: "signup"})
	eventRepo.Track(nil, &entities.UserEvent{UserID: "user-1", EventType: "app.create"})
	eventRepo.Track(nil, &entities.UserEvent{UserID: "user-2", EventType: "signup"})

	app.Get("/api/v1/admin/events/user/:id", handler.GetUserActivity)

	req := httptest.NewRequest("GET", "/api/v1/admin/events/user/user-1", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items  []entities.UserEvent `json:"items"`
		UserID string               `json:"user_id"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 events for user-1, got %d", len(result.Items))
	}
	if result.UserID != "user-1" {
		t.Errorf("Expected user_id 'user-1', got '%s'", result.UserID)
	}
}

func TestUserEventGetUserActivityEmpty(t *testing.T) {
	app, handler, _ := setupUserEventTest()
	app.Get("/api/v1/admin/events/user/:id", handler.GetUserActivity)

	req := httptest.NewRequest("GET", "/api/v1/admin/events/user/user-1", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.UserEvent `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 0 {
		t.Errorf("Expected 0 events, got %d", len(result.Items))
	}
}

func TestUserEventSurveyInsights(t *testing.T) {
	app, handler, eventRepo := setupUserEventTest()

	eventRepo.Track(nil, &entities.UserEvent{
		UserID:    "user-1",
		EventType: "onboarding.done",
		Properties: map[string]interface{}{
			"use_case": "side_project",
			"role":     "developer",
		},
	})

	app.Get("/api/v1/admin/surveys", handler.SurveyInsights)

	req := httptest.NewRequest("GET", "/api/v1/admin/surveys", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		TotalResponses int `json:"total_responses"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.TotalResponses != 1 {
		t.Errorf("Expected 1 response, got %d", result.TotalResponses)
	}
}

func TestUserEventSurveyInsightsEmpty(t *testing.T) {
	app, handler, _ := setupUserEventTest()
	app.Get("/api/v1/admin/surveys", handler.SurveyInsights)

	req := httptest.NewRequest("GET", "/api/v1/admin/surveys", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		TotalResponses int `json:"total_responses"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.TotalResponses != 0 {
		t.Errorf("Expected 0 responses, got %d", result.TotalResponses)
	}
}
