package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// injectRole middleware for tests to simulate role
func injectRole(role entities.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("role", role)
		return c.Next()
	}
}

func setupTeamTest() (*fiber.App, *handlers.TeamMemberHandler, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	teamRepo := memory.NewMemoryTeamMemberRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	teamSvc := services.NewTeamMemberService(teamRepo, userRepo, planRepo, "test-jwt-secret")
	handler := handlers.NewTeamMemberHandler(teamSvc)
	return app, handler, planRepo
}

func TestTeamInviteMember(t *testing.T) {
	fiberApp, handler, planRepo := setupTeamTest()
	// Upgrade to Pro so invite is allowed (free plan MaxTeamMembers=1, blocks all invites)
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)
	fiberApp.Post("/api/v1/team/invite", injectUserID("user-1"), injectRole(entities.RoleOwner), handler.InviteMember)

	body := `{"email":"dev@example.com","role":"developer"}`
	req := httptest.NewRequest("POST", "/api/v1/team/invite", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("Expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["email"] != "dev@example.com" {
		t.Errorf("Expected email 'dev@example.com', got '%v'", result["email"])
	}
}

func TestTeamInviteMemberNotOwner(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	// Not owner role
	fiberApp.Post("/api/v1/team/invite", injectUserID("user-1"), injectRole(entities.RoleDeveloper), handler.InviteMember)

	body := `{"email":"dev@example.com","role":"developer"}`
	req := httptest.NewRequest("POST", "/api/v1/team/invite", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for non-owner, got %d", resp.StatusCode)
	}
}

func TestTeamInviteMemberNoRole(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	// No role injected — zero value for entities.Role is ""
	fiberApp.Post("/api/v1/team/invite", injectUserID("user-1"), handler.InviteMember)

	body := `{"email":"dev@example.com","role":"developer"}`
	req := httptest.NewRequest("POST", "/api/v1/team/invite", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 when no role, got %d", resp.StatusCode)
	}
}

func TestTeamInviteMemberNoEmail(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	fiberApp.Post("/api/v1/team/invite", injectUserID("user-1"), injectRole(entities.RoleOwner), handler.InviteMember)

	body := `{"role":"developer"}`
	req := httptest.NewRequest("POST", "/api/v1/team/invite", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for missing email, got %d", resp.StatusCode)
	}
}

func TestTeamInviteMemberInvalidEmail(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	fiberApp.Post("/api/v1/team/invite", injectUserID("user-1"), injectRole(entities.RoleOwner), handler.InviteMember)

	body := `{"email":"not-an-email","role":"developer"}`
	req := httptest.NewRequest("POST", "/api/v1/team/invite", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for invalid email, got %d", resp.StatusCode)
	}
}

func TestTeamInviteMemberDefaultRole(t *testing.T) {
	fiberApp, handler, planRepo := setupTeamTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)
	fiberApp.Post("/api/v1/team/invite", injectUserID("user-1"), injectRole(entities.RoleOwner), handler.InviteMember)

	// No role specified — should default to viewer
	body := `{"email":"viewer@example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/team/invite", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("Expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["role"] != "viewer" {
		t.Errorf("Expected default role 'viewer', got '%v'", result["role"])
	}
}

func TestTeamInviteMemberInvalidBody(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	fiberApp.Post("/api/v1/team/invite", injectUserID("user-1"), injectRole(entities.RoleOwner), handler.InviteMember)

	req := httptest.NewRequest("POST", "/api/v1/team/invite", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestTeamListMembers(t *testing.T) {
	fiberApp, handler, planRepo := setupTeamTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)
	fiberApp.Post("/api/v1/team/invite", injectUserID("user-1"), injectRole(entities.RoleOwner), handler.InviteMember)
	fiberApp.Get("/api/v1/team/members", injectUserID("user-1"), handler.ListMembers)

	// Invite a member first
	body := `{"email":"dev@example.com","role":"developer"}`
	inviteReq := httptest.NewRequest("POST", "/api/v1/team/invite", bytes.NewBufferString(body))
	inviteReq.Header.Set("Content-Type", "application/json")
	fiberApp.Test(inviteReq)

	// List members
	listReq := httptest.NewRequest("GET", "/api/v1/team/members", nil)
	listResp, _ := fiberApp.Test(listReq)

	if listResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", listResp.StatusCode)
	}

	var result struct {
		Items []map[string]interface{} `json:"items"`
		Total int                      `json:"total"`
	}
	json.NewDecoder(listResp.Body).Decode(&result)

	if result.Total != 1 {
		t.Errorf("Expected 1 member, got %d", result.Total)
	}
}

func TestTeamListMembersEmpty(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	fiberApp.Get("/api/v1/team/members", injectUserID("user-1"), handler.ListMembers)

	req := httptest.NewRequest("GET", "/api/v1/team/members", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []map[string]interface{} `json:"items"`
		Total int                      `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 0 {
		t.Errorf("Expected 0 members, got %d", result.Total)
	}
}

func TestTeamUpdateRoleNotOwner(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	fiberApp.Put("/api/v1/team/members/:id/role", injectUserID("user-1"), injectRole(entities.RoleDeveloper), handler.UpdateRole)

	body := `{"role":"admin"}`
	req := httptest.NewRequest("PUT", "/api/v1/team/members/some-id/role", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for non-owner, got %d", resp.StatusCode)
	}
}

func TestTeamUpdateRoleMissingRole(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	fiberApp.Put("/api/v1/team/members/:id/role", injectUserID("user-1"), injectRole(entities.RoleOwner), handler.UpdateRole)

	body := `{}`
	req := httptest.NewRequest("PUT", "/api/v1/team/members/some-id/role", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for missing role, got %d", resp.StatusCode)
	}
}

func TestTeamRemoveMemberNotOwner(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	fiberApp.Delete("/api/v1/team/members/:id", injectUserID("user-1"), injectRole(entities.RoleViewer), handler.RemoveMember)

	req := httptest.NewRequest("DELETE", "/api/v1/team/members/some-id", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for non-owner, got %d", resp.StatusCode)
	}
}

func TestTeamAcceptInviteMissingFields(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	fiberApp.Post("/api/v1/team/accept-invite", handler.AcceptInvite)

	tests := []struct {
		name string
		body string
	}{
		{"missing token", `{"email":"a@b.com","password":"pass123"}`},
		{"missing email", `{"token":"tok123","password":"pass123"}`},
		{"missing password", `{"token":"tok123","email":"a@b.com"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/team/accept-invite", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			resp, _ := fiberApp.Test(req)

			if resp.StatusCode != 400 {
				t.Errorf("Expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestTeamAcceptInviteInvalidBody(t *testing.T) {
	fiberApp, handler, _ := setupTeamTest()
	fiberApp.Post("/api/v1/team/accept-invite", handler.AcceptInvite)

	req := httptest.NewRequest("POST", "/api/v1/team/accept-invite", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
