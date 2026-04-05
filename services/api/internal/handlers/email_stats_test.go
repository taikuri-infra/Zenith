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

func setupEmailStatsTest() (*fiber.App, *handlers.EmailStatsHandler, *memory.MemoryEmailSendRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	emailRepo := memory.NewMemoryEmailSendRepository()
	handler := handlers.NewEmailStatsHandler(emailRepo)
	return app, handler, emailRepo
}

func TestEmailStatsGetStatsEmpty(t *testing.T) {
	app, handler, _ := setupEmailStatsTest()
	app.Get("/api/v1/admin/emails/stats", handler.GetStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/emails/stats", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result entities.EmailStats
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Sent != 0 {
		t.Errorf("Expected 0 sent, got %d", result.Sent)
	}
}

func TestEmailStatsGetStatsWithData(t *testing.T) {
	app, handler, emailRepo := setupEmailStatsTest()

	emailRepo.Record(nil, &entities.EmailSend{
		UserID:      "user-1",
		TemplateKey: "welcome",
	})
	emailRepo.Record(nil, &entities.EmailSend{
		UserID:      "user-2",
		TemplateKey: "welcome",
	})
	emailRepo.Record(nil, &entities.EmailSend{
		UserID:      "user-1",
		TemplateKey: "onboarding",
	})

	app.Get("/api/v1/admin/emails/stats", handler.GetStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/emails/stats", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result entities.EmailStats
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Sent != 3 {
		t.Errorf("Expected 3 sent, got %d", result.Sent)
	}
	if result.ByTemplate["welcome"] != 2 {
		t.Errorf("Expected 2 welcome emails, got %d", result.ByTemplate["welcome"])
	}
}

func TestEmailStatsGetStatsWithOpened(t *testing.T) {
	app, handler, emailRepo := setupEmailStatsTest()

	emailRepo.Record(nil, &entities.EmailSend{
		ID:          "email-1",
		UserID:      "user-1",
		TemplateKey: "welcome",
	})
	emailRepo.MarkOpened(nil, "email-1")

	app.Get("/api/v1/admin/emails/stats", handler.GetStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/emails/stats", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result entities.EmailStats
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Opened != 1 {
		t.Errorf("Expected 1 opened, got %d", result.Opened)
	}
}
