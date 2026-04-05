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

func setupSCIMTest() (*fiber.App, *handlers.SCIMHandler, *memory.MemoryUserRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewSCIMHandler(userRepo, planRepo)
	return app, handler, userRepo
}

func TestSCIMListUsers(t *testing.T) {
	app, handler, _ := setupSCIMTest()
	app.Get("/scim/v2/Users", handler.ListUsers)

	req := httptest.NewRequest("GET", "/scim/v2/Users", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	total, _ := result["totalResults"].(float64)
	if total != 0 {
		t.Errorf("Expected totalResults 0, got %v", total)
	}

	schemas, _ := result["schemas"].([]interface{})
	if len(schemas) == 0 || schemas[0] != "urn:ietf:params:scim:api:messages:2.0:ListResponse" {
		t.Error("Expected SCIM ListResponse schema")
	}
}

func TestSCIMGetUser(t *testing.T) {
	app, handler, userRepo := setupSCIMTest()

	user, _ := userRepo.Create(nil, "scimuser@example.com", "password123", "SCIM User", entities.RoleViewer)

	app.Get("/scim/v2/Users/:userId", handler.GetUser)

	req := httptest.NewRequest("GET", "/scim/v2/Users/"+user.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["userName"] != "scimuser@example.com" {
		t.Errorf("Expected userName 'scimuser@example.com', got '%v'", result["userName"])
	}
}

func TestSCIMGetUserNotFound(t *testing.T) {
	app, handler, _ := setupSCIMTest()
	app.Get("/scim/v2/Users/:userId", handler.GetUser)

	req := httptest.NewRequest("GET", "/scim/v2/Users/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestSCIMCreateUser(t *testing.T) {
	app, handler, _ := setupSCIMTest()
	app.Post("/scim/v2/Users", handler.CreateUser)

	body := `{"userName":"newuser@example.com","name":{"formatted":"New User"}}`
	req := httptest.NewRequest("POST", "/scim/v2/Users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["userName"] != "newuser@example.com" {
		t.Errorf("Expected userName 'newuser@example.com', got '%v'", result["userName"])
	}
}

func TestSCIMCreateUserNoUserName(t *testing.T) {
	app, handler, _ := setupSCIMTest()
	app.Post("/scim/v2/Users", handler.CreateUser)

	body := `{"name":{"formatted":"No User Name"}}`
	req := httptest.NewRequest("POST", "/scim/v2/Users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestSCIMCreateUserDuplicate(t *testing.T) {
	app, handler, userRepo := setupSCIMTest()
	userRepo.Create(nil, "existing@example.com", "password123", "Existing", entities.RoleViewer)

	app.Post("/scim/v2/Users", handler.CreateUser)

	body := `{"userName":"existing@example.com","name":{"formatted":"Duplicate"}}`
	req := httptest.NewRequest("POST", "/scim/v2/Users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 409 {
		t.Errorf("Expected 409, got %d", resp.StatusCode)
	}
}

func TestSCIMDeleteUser(t *testing.T) {
	app, handler, _ := setupSCIMTest()
	app.Delete("/scim/v2/Users/:userId", handler.DeleteUser)

	req := httptest.NewRequest("DELETE", "/scim/v2/Users/some-user-id", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 204 {
		t.Errorf("Expected 204, got %d", resp.StatusCode)
	}
}
