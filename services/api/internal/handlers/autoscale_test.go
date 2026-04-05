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

func setupAutoscaleTest() (*fiber.App, *handlers.AutoscaleHandler, *memory.MemoryAutoscaleRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	repo := memory.NewMemoryAutoscaleRepository()
	handler := handlers.NewAutoscaleHandler(repo)
	return app, handler, repo
}

func TestAutoscaleGetStatus(t *testing.T) {
	app, handler, _ := setupAutoscaleTest()
	app.Get("/api/v1/admin/autoscaler/status", handler.GetStatus)

	req := httptest.NewRequest("GET", "/api/v1/admin/autoscaler/status", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestAutoscaleListNodes(t *testing.T) {
	app, handler, repo := setupAutoscaleTest()

	repo.SaveNode(nil, &entities.HetznerNode{
		ServerID: 1001,
		Name:     "worker-1",
		Status:   "running",
	})
	repo.SaveNode(nil, &entities.HetznerNode{
		ServerID: 1002,
		Name:     "worker-2",
		Status:   "running",
	})

	app.Get("/api/v1/admin/autoscaler/nodes", handler.ListNodes)

	req := httptest.NewRequest("GET", "/api/v1/admin/autoscaler/nodes", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.HetznerNode `json:"items"`
		Total int                    `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 nodes, got %d", result.Total)
	}
}

func TestAutoscaleListNodesEmpty(t *testing.T) {
	app, handler, _ := setupAutoscaleTest()
	app.Get("/api/v1/admin/autoscaler/nodes", handler.ListNodes)

	req := httptest.NewRequest("GET", "/api/v1/admin/autoscaler/nodes", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0 nodes, got %d", result.Total)
	}
}

func TestAutoscaleListEvents(t *testing.T) {
	app, handler, repo := setupAutoscaleTest()

	repo.LogScaleEvent(nil, &entities.AutoscaleEvent{
		Action: "scale_up",
		Reason: "High CPU usage",
	})

	app.Get("/api/v1/admin/autoscaler/events", handler.ListEvents)

	req := httptest.NewRequest("GET", "/api/v1/admin/autoscaler/events", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.AutoscaleEvent `json:"items"`
		Total int                       `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 1 {
		t.Errorf("Expected 1 event, got %d", result.Total)
	}
}

func TestAutoscaleListEventsEmpty(t *testing.T) {
	app, handler, _ := setupAutoscaleTest()
	app.Get("/api/v1/admin/autoscaler/events", handler.ListEvents)

	req := httptest.NewRequest("GET", "/api/v1/admin/autoscaler/events", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestAutoscaleListEventsWithLimit(t *testing.T) {
	app, handler, _ := setupAutoscaleTest()
	app.Get("/api/v1/admin/autoscaler/events", handler.ListEvents)

	req := httptest.NewRequest("GET", "/api/v1/admin/autoscaler/events?limit=10", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}
