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
	"github.com/google/uuid"
)

func setupNotificationTest() (*fiber.App, *handlers.NotificationHandler, *memory.MemoryNotificationRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	notifRepo := memory.NewMemoryNotificationRepository()
	handler := handlers.NewNotificationHandler(notifRepo)
	return app, handler, notifRepo
}

func TestNotificationList(t *testing.T) {
	app, handler, notifRepo := setupNotificationTest()

	notifRepo.CreateNotification(nil, &entities.Notification{
		ID:      uuid.New().String(),
		UserID:  "user-1",
		Title:   "Welcome",
		Message: "Welcome to Zenith",
		Read:    false,
	})
	notifRepo.CreateNotification(nil, &entities.Notification{
		ID:      uuid.New().String(),
		UserID:  "user-1",
		Title:   "Deploy",
		Message: "Deploy succeeded",
		Read:    false,
	})

	app.Get("/api/v1/notifications", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/notifications", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items  []entities.Notification `json:"items"`
		Unread int                     `json:"unread"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 notifications, got %d", len(result.Items))
	}
	if result.Unread != 2 {
		t.Errorf("Expected 2 unread, got %d", result.Unread)
	}
}

func TestNotificationListEmpty(t *testing.T) {
	app, handler, _ := setupNotificationTest()
	app.Get("/api/v1/notifications", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/notifications", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items  []entities.Notification `json:"items"`
		Unread int                     `json:"unread"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 0 {
		t.Errorf("Expected 0 notifications, got %d", len(result.Items))
	}
	if result.Unread != 0 {
		t.Errorf("Expected 0 unread, got %d", result.Unread)
	}
}

func TestNotificationMarkRead(t *testing.T) {
	app, handler, notifRepo := setupNotificationTest()

	id := uuid.New().String()
	notifRepo.CreateNotification(nil, &entities.Notification{
		ID:      id,
		UserID:  "user-1",
		Title:   "Welcome",
		Message: "Welcome to Zenith",
		Read:    false,
	})

	app.Post("/api/v1/notifications/read", injectUserID("user-1"), handler.MarkRead)

	body := `{"ids":["` + id + `"]}`
	req := httptest.NewRequest("POST", "/api/v1/notifications/read", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	// Verify unread count is now 0
	unread, _ := notifRepo.CountUnread(nil, "user-1")
	if unread != 0 {
		t.Errorf("Expected 0 unread, got %d", unread)
	}
}

func TestNotificationMarkAllRead(t *testing.T) {
	app, handler, notifRepo := setupNotificationTest()

	notifRepo.CreateNotification(nil, &entities.Notification{
		ID:      uuid.New().String(),
		UserID:  "user-1",
		Title:   "N1",
		Message: "msg1",
		Read:    false,
	})
	notifRepo.CreateNotification(nil, &entities.Notification{
		ID:      uuid.New().String(),
		UserID:  "user-1",
		Title:   "N2",
		Message: "msg2",
		Read:    false,
	})

	app.Post("/api/v1/notifications/read", injectUserID("user-1"), handler.MarkRead)

	// Send empty body to mark all as read
	req := httptest.NewRequest("POST", "/api/v1/notifications/read", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	unread, _ := notifRepo.CountUnread(nil, "user-1")
	if unread != 0 {
		t.Errorf("Expected 0 unread after mark all, got %d", unread)
	}
}

func TestNotificationListWithLimit(t *testing.T) {
	app, handler, _ := setupNotificationTest()
	app.Get("/api/v1/notifications", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/notifications?limit=10", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestNotificationListActivity(t *testing.T) {
	app, handler, notifRepo := setupNotificationTest()

	notifRepo.AddActivity(nil, &entities.ActivityEntry{
		UserID:  "user-1",
		Action:  "deploy",
		Details: "Deployed app",
	})

	app.Get("/api/v1/activity", injectUserID("user-1"), handler.ListActivity)

	req := httptest.NewRequest("GET", "/api/v1/activity", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []entities.ActivityEntry
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 1 {
		t.Errorf("Expected 1 activity, got %d", len(result))
	}
}

func TestNotificationListActivityEmpty(t *testing.T) {
	app, handler, _ := setupNotificationTest()
	app.Get("/api/v1/activity", injectUserID("user-1"), handler.ListActivity)

	req := httptest.NewRequest("GET", "/api/v1/activity", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}
