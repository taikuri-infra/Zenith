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

func setupRoleTest() (*fiber.App, *handlers.RoleHandler, *memory.MemoryRoleRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	roleRepo := memory.NewMemoryRoleRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	// Team plan required for custom roles
	planRepo.SetUserPlan(nil, "user-1", entities.PlanTeam)
	handler := handlers.NewRoleHandler(roleRepo, planRepo)
	return app, handler, roleRepo
}

func TestRoleCreate(t *testing.T) {
	app, handler, _ := setupRoleTest()
	app.Post("/api/v1/roles", injectUserID("user-1"), handler.Create)

	body := `{"name":"Developer","description":"Dev role","permissions":["deploy","view_logs"]}`
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result entities.CustomRole
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Name != "Developer" {
		t.Errorf("Expected name 'Developer', got '%s'", result.Name)
	}
	if len(result.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(result.Permissions))
	}
}

func TestRoleCreateNoName(t *testing.T) {
	app, handler, _ := setupRoleTest()
	app.Post("/api/v1/roles", injectUserID("user-1"), handler.Create)

	body := `{"permissions":["deploy"]}`
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestRoleCreateNoPermissions(t *testing.T) {
	app, handler, _ := setupRoleTest()
	app.Post("/api/v1/roles", injectUserID("user-1"), handler.Create)

	body := `{"name":"Empty Role"}`
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestRoleCreateFreePlanForbidden(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	roleRepo := memory.NewMemoryRoleRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewRoleHandler(roleRepo, planRepo)

	app.Post("/api/v1/roles", injectUserID("user-1"), handler.Create)

	body := `{"name":"Dev","permissions":["deploy"]}`
	req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestRoleList(t *testing.T) {
	app, handler, roleRepo := setupRoleTest()

	roleRepo.CreateRole(nil, "user-1", "Dev", "Developer", []entities.Permission{"deploy"})
	roleRepo.CreateRole(nil, "user-1", "Viewer", "Read only", []entities.Permission{"view_logs"})

	app.Get("/api/v1/roles", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/roles", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.CustomRole `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(result.Items))
	}
}

func TestRoleListEmpty(t *testing.T) {
	app, handler, _ := setupRoleTest()
	app.Get("/api/v1/roles", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/roles", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestRoleUpdate(t *testing.T) {
	app, handler, roleRepo := setupRoleTest()

	role, _ := roleRepo.CreateRole(nil, "user-1", "Dev", "Developer", []entities.Permission{"deploy"})

	app.Put("/api/v1/roles/:roleId", injectUserID("user-1"), handler.Update)

	body := `{"name":"Senior Dev","description":"Updated desc"}`
	req := httptest.NewRequest("PUT", "/api/v1/roles/"+role.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result entities.CustomRole
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Name != "Senior Dev" {
		t.Errorf("Expected 'Senior Dev', got '%s'", result.Name)
	}
}

func TestRoleUpdateNotFound(t *testing.T) {
	app, handler, _ := setupRoleTest()
	app.Put("/api/v1/roles/:roleId", injectUserID("user-1"), handler.Update)

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/roles/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestRoleUpdateForbidden(t *testing.T) {
	app, handler, roleRepo := setupRoleTest()

	role, _ := roleRepo.CreateRole(nil, "user-1", "Dev", "Developer", []entities.Permission{"deploy"})

	app.Put("/api/v1/roles/:roleId", injectUserID("user-2"), handler.Update)

	body := `{"name":"Hacked"}`
	req := httptest.NewRequest("PUT", "/api/v1/roles/"+role.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestRoleDelete(t *testing.T) {
	app, handler, roleRepo := setupRoleTest()

	role, _ := roleRepo.CreateRole(nil, "user-1", "ToDelete", "temp", []entities.Permission{"deploy"})

	app.Delete("/api/v1/roles/:roleId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/roles/"+role.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 204 {
		t.Fatalf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestRoleDeleteNotFound(t *testing.T) {
	app, handler, _ := setupRoleTest()
	app.Delete("/api/v1/roles/:roleId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/roles/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestRoleAssign(t *testing.T) {
	app, handler, roleRepo := setupRoleTest()

	role, _ := roleRepo.CreateRole(nil, "user-1", "Dev", "Developer", []entities.Permission{"deploy"})

	app.Post("/api/v1/roles/:roleId/assign", injectUserID("user-1"), handler.AssignRole)

	body := `{"member_id":"member-1"}`
	req := httptest.NewRequest("POST", "/api/v1/roles/"+role.ID+"/assign", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result entities.RoleAssignment
	json.NewDecoder(resp.Body).Decode(&result)
	if result.MemberID != "member-1" {
		t.Errorf("Expected member_id 'member-1', got '%s'", result.MemberID)
	}
}

func TestRoleAssignNoMemberID(t *testing.T) {
	app, handler, roleRepo := setupRoleTest()

	role, _ := roleRepo.CreateRole(nil, "user-1", "Dev", "Developer", []entities.Permission{"deploy"})

	app.Post("/api/v1/roles/:roleId/assign", injectUserID("user-1"), handler.AssignRole)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/roles/"+role.ID+"/assign", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestRoleListAssignments(t *testing.T) {
	app, handler, roleRepo := setupRoleTest()

	role, _ := roleRepo.CreateRole(nil, "user-1", "Dev", "Developer", []entities.Permission{"deploy"})
	roleRepo.AssignRole(nil, role.ID, "member-1", "user-1")
	roleRepo.AssignRole(nil, role.ID, "member-2", "user-1")

	app.Get("/api/v1/roles/:roleId/assignments", injectUserID("user-1"), handler.ListAssignments)

	req := httptest.NewRequest("GET", "/api/v1/roles/"+role.ID+"/assignments", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.RoleAssignment `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 assignments, got %d", len(result.Items))
	}
}

func TestRoleRemoveAssignment(t *testing.T) {
	app, handler, roleRepo := setupRoleTest()

	role, _ := roleRepo.CreateRole(nil, "user-1", "Dev", "Developer", []entities.Permission{"deploy"})
	assignment, _ := roleRepo.AssignRole(nil, role.ID, "member-1", "user-1")

	app.Delete("/api/v1/roles/:roleId/assignments/:assignmentId", injectUserID("user-1"), handler.RemoveAssignment)

	req := httptest.NewRequest("DELETE", "/api/v1/roles/"+role.ID+"/assignments/"+assignment.ID, nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 204 {
		t.Fatalf("Expected 204, got %d", resp.StatusCode)
	}
}

func TestRoleListPermissions(t *testing.T) {
	app, handler, _ := setupRoleTest()
	app.Get("/api/v1/permissions", handler.ListPermissions)

	req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Permissions []entities.Permission `json:"permissions"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Permissions) == 0 {
		t.Error("Expected at least one permission")
	}
}
