package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupAppAuthTest() (*fiber.App, *handlers.AppAuthHandler, *memory.MemoryAppAuthRepository, *memory.MemoryAppRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	authRepo := memory.NewMemoryAppAuthRepository()
	appRepo := memory.NewMemoryAppRepository()
	handler := handlers.NewAppAuthHandler(authRepo, appRepo)
	return app, handler, authRepo, appRepo
}

func createAppAuthTestApp(appRepo *memory.MemoryAppRepository, userID, appName string) *entities.App {
	app, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		Name:         appName,
		UserID:       userID,
		ProjectID:    "proj-1",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "registry.example.com/test:latest",
	})
	return app
}

func TestAppAuthEnable(t *testing.T) {
	app, handler, _, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	app.Post("/api/v1/apps/:appId/auth/enable", injectUserID("user-1"), handler.Enable)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/enable", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["enabled"] != true {
		t.Errorf("Expected enabled true, got %v", result["enabled"])
	}
}

func TestAppAuthEnableNotYourApp(t *testing.T) {
	app, handler, _, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	app.Post("/api/v1/apps/:appId/auth/enable", injectUserID("user-2"), handler.Enable)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/enable", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestAppAuthEnableAppNotFound(t *testing.T) {
	app, handler, _, _ := setupAppAuthTest()
	app.Post("/api/v1/apps/:appId/auth/enable", injectUserID("user-1"), handler.Enable)

	req := httptest.NewRequest("POST", "/api/v1/apps/nonexistent/auth/enable", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestAppAuthDisable(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)

	app.Post("/api/v1/apps/:appId/auth/disable", injectUserID("user-1"), handler.Disable)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/disable", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "auth disabled" {
		t.Errorf("Expected 'auth disabled', got '%v'", result["message"])
	}
}

func TestAppAuthDisableNotYourApp(t *testing.T) {
	app, handler, _, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	app.Post("/api/v1/apps/:appId/auth/disable", injectUserID("user-2"), handler.Disable)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/disable", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestAppAuthStatus(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)

	app.Get("/api/v1/apps/:appId/auth", handler.Status)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/auth", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["enabled"] != true {
		t.Errorf("Expected enabled true, got %v", result["enabled"])
	}
}

func TestAppAuthStatusNotConfigured(t *testing.T) {
	app, handler, _, _ := setupAppAuthTest()
	app.Get("/api/v1/apps/:appId/auth", handler.Status)

	req := httptest.NewRequest("GET", "/api/v1/apps/not-configured/auth", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["enabled"] != false {
		t.Errorf("Expected enabled false, got %v", result["enabled"])
	}
}

func TestAppAuthSignup(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)

	app.Post("/api/v1/apps/:appId/auth/signup", handler.Signup)

	body := `{"email":"enduser@example.com","password":"securepass123","name":"End User"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["access_token"] == nil || result["access_token"] == "" {
		t.Error("Expected non-empty access_token")
	}
	if result["token_type"] != "bearer" {
		t.Errorf("Expected token_type 'bearer', got '%v'", result["token_type"])
	}
}

func TestAppAuthSignupAuthNotEnabled(t *testing.T) {
	app, handler, _, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	app.Post("/api/v1/apps/:appId/auth/signup", handler.Signup)

	body := `{"email":"enduser@example.com","password":"securepass123","name":"End User"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestAppAuthSignupMissingFields(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)

	app.Post("/api/v1/apps/:appId/auth/signup", handler.Signup)

	body := `{"email":"enduser@example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestAppAuthSignupShortPassword(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)

	app.Post("/api/v1/apps/:appId/auth/signup", handler.Signup)

	body := `{"email":"enduser@example.com","password":"short","name":"User"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestAppAuthLogin(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)
	authRepo.CreateAppUser(nil, testApp.ID, "enduser@example.com", "securepass123", "End User")

	app.Post("/api/v1/apps/:appId/auth/login", handler.Login)

	body := `{"email":"enduser@example.com","password":"securepass123"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["access_token"] == nil || result["access_token"] == "" {
		t.Error("Expected non-empty access_token")
	}
}

func TestAppAuthLoginWrongPassword(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)
	authRepo.CreateAppUser(nil, testApp.ID, "enduser@example.com", "securepass123", "End User")

	app.Post("/api/v1/apps/:appId/auth/login", handler.Login)

	body := `{"email":"enduser@example.com","password":"wrongpass"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestAppAuthLoginUserNotFound(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)

	app.Post("/api/v1/apps/:appId/auth/login", handler.Login)

	body := `{"email":"nobody@example.com","password":"securepass123"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestAppAuthListUsers(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)
	authRepo.CreateAppUser(nil, testApp.ID, "user1@example.com", "password123", "User 1")
	authRepo.CreateAppUser(nil, testApp.ID, "user2@example.com", "password123", "User 2")

	app.Get("/api/v1/apps/:appId/auth/users", injectUserID("user-1"), handler.ListUsers)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/auth/users", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 users, got %d", result.Total)
	}
}

func TestAppAuthListUsersNotOwner(t *testing.T) {
	app, handler, _, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	app.Get("/api/v1/apps/:appId/auth/users", injectUserID("user-2"), handler.ListUsers)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/auth/users", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestAppAuthDeleteUser(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)
	appUser, _ := authRepo.CreateAppUser(nil, testApp.ID, "todelete@example.com", "password123", "Delete Me")

	app.Delete("/api/v1/apps/:appId/auth/users/:userId", injectUserID("user-1"), handler.DeleteUser)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+testApp.ID+"/auth/users/"+appUser.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "user deleted" {
		t.Errorf("Expected 'user deleted', got '%v'", result["message"])
	}
}

func TestAppAuthDeleteUserNotOwner(t *testing.T) {
	app, handler, authRepo, appRepo := setupAppAuthTest()

	testApp := createAppAuthTestApp(appRepo, "user-1", "test-app")
	authRepo.EnableAuth(nil, testApp.ID, 1000)
	appUser, _ := authRepo.CreateAppUser(nil, testApp.ID, "todelete@example.com", "password123", "Delete Me")

	app.Delete("/api/v1/apps/:appId/auth/users/:userId", injectUserID("user-2"), handler.DeleteUser)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+testApp.ID+"/auth/users/"+appUser.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}
