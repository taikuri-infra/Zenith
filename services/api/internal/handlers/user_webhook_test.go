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

func setupWebhookTest() (*fiber.App, *handlers.UserWebhookHandler, *memory.MemoryUserWebhookRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	// Set user-1 to Pro so webhooks are allowed
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)
	handler := handlers.NewUserWebhookHandler(webhookRepo, planRepo)
	return app, handler, webhookRepo
}

func TestWebhookCreate(t *testing.T) {
	app, handler, _ := setupWebhookTest()
	app.Post("/api/v1/webhooks", injectUserID("user-1"), handler.Create)

	body := `{"url":"https://example.com/hook","events":["deploy.success"]}`
	req := httptest.NewRequest("POST", "/api/v1/webhooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result entities.UserWebhook
	json.NewDecoder(resp.Body).Decode(&result)

	if result.URL != "https://example.com/hook" {
		t.Errorf("Expected URL 'https://example.com/hook', got '%s'", result.URL)
	}
	if result.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if result.Secret == "" {
		t.Error("Expected non-empty secret")
	}
}

func TestWebhookCreateNoURL(t *testing.T) {
	app, handler, _ := setupWebhookTest()
	app.Post("/api/v1/webhooks", injectUserID("user-1"), handler.Create)

	body := `{"events":["deploy.success"]}`
	req := httptest.NewRequest("POST", "/api/v1/webhooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestWebhookCreateNoEvents(t *testing.T) {
	app, handler, _ := setupWebhookTest()
	app.Post("/api/v1/webhooks", injectUserID("user-1"), handler.Create)

	body := `{"url":"https://example.com/hook","events":[]}`
	req := httptest.NewRequest("POST", "/api/v1/webhooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestWebhookCreateFreePlanForbidden(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	// user-1 is on free plan by default
	handler := handlers.NewUserWebhookHandler(webhookRepo, planRepo)
	app.Post("/api/v1/webhooks", injectUserID("user-1"), handler.Create)

	body := `{"url":"https://example.com/hook","events":["deploy.success"]}`
	req := httptest.NewRequest("POST", "/api/v1/webhooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for free plan, got %d", resp.StatusCode)
	}
}

func TestWebhookList(t *testing.T) {
	app, handler, webhookRepo := setupWebhookTest()

	webhookRepo.CreateWebhook(nil, "user-1", "https://example.com/hook1", []entities.WebhookEvent{"deploy.success"})
	webhookRepo.CreateWebhook(nil, "user-1", "https://example.com/hook2", []entities.WebhookEvent{"deploy.failed"})

	app.Get("/api/v1/webhooks", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/webhooks", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.UserWebhook `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 webhooks, got %d", len(result.Items))
	}
}

func TestWebhookListEmpty(t *testing.T) {
	app, handler, _ := setupWebhookTest()
	app.Get("/api/v1/webhooks", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/webhooks", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.UserWebhook `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 0 {
		t.Errorf("Expected 0 webhooks, got %d", len(result.Items))
	}
}

func TestWebhookUpdate(t *testing.T) {
	app, handler, webhookRepo := setupWebhookTest()

	webhook, _ := webhookRepo.CreateWebhook(nil, "user-1", "https://example.com/hook", []entities.WebhookEvent{"deploy.success"})

	app.Put("/api/v1/webhooks/:webhookId", injectUserID("user-1"), handler.Update)

	body := `{"url":"https://example.com/updated","active":false}`
	req := httptest.NewRequest("PUT", "/api/v1/webhooks/"+webhook.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result entities.UserWebhook
	json.NewDecoder(resp.Body).Decode(&result)
	if result.URL != "https://example.com/updated" {
		t.Errorf("Expected updated URL, got '%s'", result.URL)
	}
	if result.Active != false {
		t.Error("Expected active=false")
	}
}

func TestWebhookUpdateNotFound(t *testing.T) {
	app, handler, _ := setupWebhookTest()
	app.Put("/api/v1/webhooks/:webhookId", injectUserID("user-1"), handler.Update)

	body := `{"url":"https://example.com/updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/webhooks/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestWebhookUpdateForbidden(t *testing.T) {
	app, handler, webhookRepo := setupWebhookTest()

	webhook, _ := webhookRepo.CreateWebhook(nil, "user-1", "https://example.com/hook", []entities.WebhookEvent{"deploy.success"})

	app.Put("/api/v1/webhooks/:webhookId", injectUserID("user-2"), handler.Update)

	body := `{"url":"https://example.com/hacked"}`
	req := httptest.NewRequest("PUT", "/api/v1/webhooks/"+webhook.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestWebhookDelete(t *testing.T) {
	app, handler, webhookRepo := setupWebhookTest()

	webhook, _ := webhookRepo.CreateWebhook(nil, "user-1", "https://example.com/hook", []entities.WebhookEvent{"deploy.success"})

	app.Delete("/api/v1/webhooks/:webhookId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/webhooks/"+webhook.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 204 {
		t.Fatalf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestWebhookDeleteNotFound(t *testing.T) {
	app, handler, _ := setupWebhookTest()
	app.Delete("/api/v1/webhooks/:webhookId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/webhooks/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestWebhookDeleteForbidden(t *testing.T) {
	app, handler, webhookRepo := setupWebhookTest()

	webhook, _ := webhookRepo.CreateWebhook(nil, "user-1", "https://example.com/hook", []entities.WebhookEvent{"deploy.success"})

	app.Delete("/api/v1/webhooks/:webhookId", injectUserID("user-2"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/webhooks/"+webhook.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestWebhookListDeliveries(t *testing.T) {
	app, handler, webhookRepo := setupWebhookTest()

	webhook, _ := webhookRepo.CreateWebhook(nil, "user-1", "https://example.com/hook", []entities.WebhookEvent{"deploy.success"})
	webhookRepo.RecordDelivery(nil, webhook.ID, "deploy.success", `{"event":"deploy.success"}`, "delivered", 200, "")

	app.Get("/api/v1/webhooks/:webhookId/deliveries", injectUserID("user-1"), handler.ListDeliveries)

	req := httptest.NewRequest("GET", "/api/v1/webhooks/"+webhook.ID+"/deliveries", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.WebhookDelivery `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 1 {
		t.Errorf("Expected 1 delivery, got %d", len(result.Items))
	}
}

func TestWebhookListDeliveriesForbidden(t *testing.T) {
	app, handler, webhookRepo := setupWebhookTest()

	webhook, _ := webhookRepo.CreateWebhook(nil, "user-1", "https://example.com/hook", []entities.WebhookEvent{"deploy.success"})

	app.Get("/api/v1/webhooks/:webhookId/deliveries", injectUserID("user-2"), handler.ListDeliveries)

	req := httptest.NewRequest("GET", "/api/v1/webhooks/"+webhook.ID+"/deliveries", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}
