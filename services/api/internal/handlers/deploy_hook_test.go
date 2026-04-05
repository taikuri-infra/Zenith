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

func setupDeployHookTest() (*fiber.App, *handlers.DeployHookHandler, *memory.MemoryDeployHookRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	hookRepo := memory.NewMemoryDeployHookRepository()
	handler := handlers.NewDeployHookHandler(hookRepo)
	return app, handler, hookRepo
}

func TestDeployHookCreateHTTP(t *testing.T) {
	app, handler, _ := setupDeployHookTest()
	app.Post("/apps/:appId/hooks", handler.Create)

	body := `{"name":"Notify Slack","type":"http","url":"https://hooks.slack.com/test"}`
	req := httptest.NewRequest("POST", "/apps/app-1/hooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["name"] != "Notify Slack" {
		t.Errorf("Expected name 'Notify Slack', got '%v'", result["name"])
	}
}

func TestDeployHookCreateCommand(t *testing.T) {
	app, handler, _ := setupDeployHookTest()
	app.Post("/apps/:appId/hooks", handler.Create)

	body := `{"name":"Run Migrations","type":"command","command":"npm run migrate"}`
	req := httptest.NewRequest("POST", "/apps/app-1/hooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}
}

func TestDeployHookCreateNoName(t *testing.T) {
	app, handler, _ := setupDeployHookTest()
	app.Post("/apps/:appId/hooks", handler.Create)

	body := `{"type":"http","url":"https://hooks.slack.com/test"}`
	req := httptest.NewRequest("POST", "/apps/app-1/hooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestDeployHookCreateInvalidType(t *testing.T) {
	app, handler, _ := setupDeployHookTest()
	app.Post("/apps/:appId/hooks", handler.Create)

	body := `{"name":"Test","type":"invalid","url":"https://test.com"}`
	req := httptest.NewRequest("POST", "/apps/app-1/hooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestDeployHookCreateHTTPNoURL(t *testing.T) {
	app, handler, _ := setupDeployHookTest()
	app.Post("/apps/:appId/hooks", handler.Create)

	body := `{"name":"Test","type":"http"}`
	req := httptest.NewRequest("POST", "/apps/app-1/hooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestDeployHookCreateHTTPInvalidURL(t *testing.T) {
	app, handler, _ := setupDeployHookTest()
	app.Post("/apps/:appId/hooks", handler.Create)

	body := `{"name":"Test","type":"http","url":"ftp://bad.com"}`
	req := httptest.NewRequest("POST", "/apps/app-1/hooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestDeployHookCreateCommandNoCommand(t *testing.T) {
	app, handler, _ := setupDeployHookTest()
	app.Post("/apps/:appId/hooks", handler.Create)

	body := `{"name":"Test","type":"command"}`
	req := httptest.NewRequest("POST", "/apps/app-1/hooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestDeployHookList(t *testing.T) {
	app, handler, hookRepo := setupDeployHookTest()

	hookRepo.CreateHook(nil, &entities.DeployHook{AppID: "app-1", Name: "Hook 1", Type: entities.DeployHookHTTP, URL: "https://test.com", Active: true})
	hookRepo.CreateHook(nil, &entities.DeployHook{AppID: "app-1", Name: "Hook 2", Type: entities.DeployHookCommand, Command: "echo ok", Active: true})

	app.Get("/apps/:appId/hooks", handler.List)

	req := httptest.NewRequest("GET", "/apps/app-1/hooks", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.DeployHook `json:"items"`
		Total int                   `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 hooks, got %d", result.Total)
	}
}

func TestDeployHookListEmpty(t *testing.T) {
	app, handler, _ := setupDeployHookTest()
	app.Get("/apps/:appId/hooks", handler.List)

	req := httptest.NewRequest("GET", "/apps/app-1/hooks", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0, got %d", result.Total)
	}
}

func TestDeployHookUpdate(t *testing.T) {
	app, handler, hookRepo := setupDeployHookTest()

	hook, _ := hookRepo.CreateHook(nil, &entities.DeployHook{AppID: "app-1", Name: "Hook 1", Type: entities.DeployHookHTTP, URL: "https://test.com", Active: true})

	app.Put("/apps/:appId/hooks/:hookId", handler.Update)

	body := `{"name":"Updated Hook"}`
	req := httptest.NewRequest("PUT", "/apps/app-1/hooks/"+hook.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["name"] != "Updated Hook" {
		t.Errorf("Expected name 'Updated Hook', got '%v'", result["name"])
	}
}

func TestDeployHookUpdateNotFound(t *testing.T) {
	app, handler, _ := setupDeployHookTest()
	app.Put("/apps/:appId/hooks/:hookId", handler.Update)

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/apps/app-1/hooks/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeployHookUpdateWrongApp(t *testing.T) {
	app, handler, hookRepo := setupDeployHookTest()

	hook, _ := hookRepo.CreateHook(nil, &entities.DeployHook{AppID: "app-1", Name: "Hook", Type: entities.DeployHookHTTP, URL: "https://test.com", Active: true})

	app.Put("/apps/:appId/hooks/:hookId", handler.Update)

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/apps/app-2/hooks/"+hook.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestDeployHookDelete(t *testing.T) {
	app, handler, hookRepo := setupDeployHookTest()

	hook, _ := hookRepo.CreateHook(nil, &entities.DeployHook{AppID: "app-1", Name: "Hook", Type: entities.DeployHookHTTP, URL: "https://test.com", Active: true})

	app.Delete("/apps/:appId/hooks/:hookId", handler.Delete)

	req := httptest.NewRequest("DELETE", "/apps/app-1/hooks/"+hook.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "hook deleted" {
		t.Errorf("Expected 'hook deleted', got '%v'", result["message"])
	}
}

func TestDeployHookDeleteNotFound(t *testing.T) {
	app, handler, _ := setupDeployHookTest()
	app.Delete("/apps/:appId/hooks/:hookId", handler.Delete)

	req := httptest.NewRequest("DELETE", "/apps/app-1/hooks/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeployHookDeleteWrongApp(t *testing.T) {
	app, handler, hookRepo := setupDeployHookTest()

	hook, _ := hookRepo.CreateHook(nil, &entities.DeployHook{AppID: "app-1", Name: "Hook", Type: entities.DeployHookHTTP, URL: "https://test.com", Active: true})

	app.Delete("/apps/:appId/hooks/:hookId", handler.Delete)

	req := httptest.NewRequest("DELETE", "/apps/app-2/hooks/"+hook.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}
