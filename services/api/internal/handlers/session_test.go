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

func setupSessionTest() (*fiber.App, *handlers.SessionHandler, *memory.MemorySessionRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	sessionRepo := memory.NewMemorySessionRepository()
	handler := handlers.NewSessionHandler(sessionRepo)
	return app, handler, sessionRepo
}

func TestSessionList(t *testing.T) {
	app, handler, sessionRepo := setupSessionTest()

	sessionRepo.CreateSession(nil, "user-1", "127.0.0.1", "Mozilla/5.0")
	sessionRepo.CreateSession(nil, "user-1", "192.168.1.1", "curl/7.68")

	app.Get("/api/v1/auth/sessions", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/auth/sessions", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.Session `json:"items"`
		Total int                `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 sessions, got %d", result.Total)
	}
}

func TestSessionListEmpty(t *testing.T) {
	app, handler, _ := setupSessionTest()
	app.Get("/api/v1/auth/sessions", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/auth/sessions", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestSessionRevoke(t *testing.T) {
	app, handler, sessionRepo := setupSessionTest()

	session, _ := sessionRepo.CreateSession(nil, "user-1", "127.0.0.1", "Mozilla/5.0")

	app.Delete("/api/v1/auth/sessions/:sessionId", injectUserID("user-1"), handler.Revoke)

	req := httptest.NewRequest("DELETE", "/api/v1/auth/sessions/"+session.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "session revoked" {
		t.Errorf("Expected 'session revoked', got '%v'", result["message"])
	}
}

func TestSessionRevokeNotFound(t *testing.T) {
	app, handler, _ := setupSessionTest()
	app.Delete("/api/v1/auth/sessions/:sessionId", injectUserID("user-1"), handler.Revoke)

	req := httptest.NewRequest("DELETE", "/api/v1/auth/sessions/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestSessionRevokeForbidden(t *testing.T) {
	app, handler, sessionRepo := setupSessionTest()

	session, _ := sessionRepo.CreateSession(nil, "user-1", "127.0.0.1", "Mozilla/5.0")

	app.Delete("/api/v1/auth/sessions/:sessionId", injectUserID("user-2"), handler.Revoke)

	req := httptest.NewRequest("DELETE", "/api/v1/auth/sessions/"+session.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestSessionRevokeAll(t *testing.T) {
	app, handler, sessionRepo := setupSessionTest()

	sessionRepo.CreateSession(nil, "user-1", "127.0.0.1", "Mozilla/5.0")
	sessionRepo.CreateSession(nil, "user-1", "192.168.1.1", "curl/7.68")

	app.Delete("/api/v1/auth/sessions", injectUserID("user-1"), handler.RevokeAll)

	req := httptest.NewRequest("DELETE", "/api/v1/auth/sessions", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "all sessions revoked" {
		t.Errorf("Expected 'all sessions revoked', got '%v'", result["message"])
	}

	// Verify sessions are gone
	sessions, _ := sessionRepo.ListSessionsByUser(nil, "user-1")
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions after revoke all, got %d", len(sessions))
	}
}
