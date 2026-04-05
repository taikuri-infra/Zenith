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

func setupAdminUserTest() (*fiber.App, *handlers.AdminUserHandler, *memory.MemoryUserRepository, *memory.MemoryUserPlanRepository, *memory.MemoryAppRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	storageRepo := memory.NewMemoryStorageRepository()
	handler := handlers.NewAdminUserHandler(userRepo, planRepo, appRepo, dbRepo, storageRepo)
	return app, handler, userRepo, planRepo, appRepo
}

func TestAdminUserGetUser(t *testing.T) {
	app, handler, userRepo, _, _ := setupAdminUserTest()

	user, _ := userRepo.Create(nil, "alice@example.com", "password123", "Alice", entities.RoleViewer)

	app.Get("/api/v1/admin/users/:userId", handler.GetUser)

	req := httptest.NewRequest("GET", "/api/v1/admin/users/"+user.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["email"] != "alice@example.com" {
		t.Errorf("Expected email 'alice@example.com', got '%v'", result["email"])
	}
	if result["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got '%v'", result["name"])
	}
}

func TestAdminUserGetUserNotFound(t *testing.T) {
	app, handler, _, _, _ := setupAdminUserTest()
	app.Get("/api/v1/admin/users/:userId", handler.GetUser)

	req := httptest.NewRequest("GET", "/api/v1/admin/users/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestAdminUserGetUserWithPlan(t *testing.T) {
	app, handler, userRepo, planRepo, _ := setupAdminUserTest()

	user, _ := userRepo.Create(nil, "bob@example.com", "password123", "Bob", entities.RoleViewer)
	planRepo.SetUserPlan(nil, user.ID, entities.PlanPro)

	app.Get("/api/v1/admin/users/:userId", handler.GetUser)

	req := httptest.NewRequest("GET", "/api/v1/admin/users/"+user.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["tier"] != string(entities.PlanPro) {
		t.Errorf("Expected tier 'pro', got '%v'", result["tier"])
	}
}

func TestAdminUserSetUserPlan(t *testing.T) {
	app, handler, userRepo, _, _ := setupAdminUserTest()

	user, _ := userRepo.Create(nil, "carol@example.com", "password123", "Carol", entities.RoleViewer)

	app.Post("/api/v1/admin/users/:userId/plan", handler.SetUserPlan)

	body := `{"tier":"pro"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/users/"+user.ID+"/plan", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["tier"] != "pro" {
		t.Errorf("Expected tier 'pro', got '%v'", result["tier"])
	}
}

func TestAdminUserSetUserPlanNotFound(t *testing.T) {
	app, handler, _, _, _ := setupAdminUserTest()
	app.Post("/api/v1/admin/users/:userId/plan", handler.SetUserPlan)

	body := `{"tier":"pro"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/users/nonexistent/plan", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestAdminUserSetUserPlanInvalidBody(t *testing.T) {
	app, handler, userRepo, _, _ := setupAdminUserTest()

	user, _ := userRepo.Create(nil, "dave@example.com", "password123", "Dave", entities.RoleViewer)

	app.Post("/api/v1/admin/users/:userId/plan", handler.SetUserPlan)

	req := httptest.NewRequest("POST", "/api/v1/admin/users/"+user.ID+"/plan", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestAdminUserListUserApps(t *testing.T) {
	app, handler, userRepo, _, _ := setupAdminUserTest()

	user, _ := userRepo.Create(nil, "eve@example.com", "password123", "Eve", entities.RoleViewer)

	app.Get("/api/v1/admin/users/:userId/apps", handler.ListUserApps)

	req := httptest.NewRequest("GET", "/api/v1/admin/users/"+user.ID+"/apps", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0 apps, got %d", result.Total)
	}
}

func TestAdminUserListUserDatabases(t *testing.T) {
	app, handler, userRepo, _, _ := setupAdminUserTest()

	user, _ := userRepo.Create(nil, "frank@example.com", "password123", "Frank", entities.RoleViewer)

	app.Get("/api/v1/admin/users/:userId/databases", handler.ListUserDatabases)

	req := httptest.NewRequest("GET", "/api/v1/admin/users/"+user.ID+"/databases", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0 databases, got %d", result.Total)
	}
}
