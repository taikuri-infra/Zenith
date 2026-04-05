package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupDeployTokenTest() (*fiber.App, *handlers.DeployTokenHandler, *memory.MemoryProjectRepository, *memory.MemoryDeployTokenRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	projectRepo := memory.NewMemoryProjectRepository()
	tokenRepo := memory.NewMemoryDeployTokenRepository()
	handler := handlers.NewDeployTokenHandler(tokenRepo, projectRepo)
	return app, handler, projectRepo, tokenRepo
}

func createProjectForTokenTest(t *testing.T, projectRepo *memory.MemoryProjectRepository) string {
	t.Helper()
	p, err := projectRepo.CreateProject(nil, "user-1", "test-project", "test-project", "Test Project")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}
	return p.ID
}

func TestDeployTokenCreate(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)

	body := `{"name":"ci-token","scopes":["deploy:staging","app:read"],"expires_in":"90d"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["name"] != "ci-token" {
		t.Errorf("Expected name 'ci-token', got '%v'", result["name"])
	}
	if result["secret"] == nil || result["secret"] == "" {
		t.Error("Expected secret to be returned on creation")
	}
}

func TestDeployTokenCreateNoAuth(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", handler.Create)

	body := `{"name":"ci-token","scopes":["deploy:staging"]}`
	req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestDeployTokenCreateForbidden(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo) // owned by user-1

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-2"), handler.Create)

	body := `{"name":"ci-token","scopes":["deploy:staging"]}`
	req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestDeployTokenCreateProjectNotFound(t *testing.T) {
	fiberApp, handler, _, _ := setupDeployTokenTest()

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)

	body := `{"name":"ci-token","scopes":["deploy:staging"]}`
	req := httptest.NewRequest("POST", "/api/v1/projects/nonexistent/deploy-tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeployTokenCreateNoName(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)

	body := `{"scopes":["deploy:staging"]}`
	req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for missing name, got %d", resp.StatusCode)
	}
}

func TestDeployTokenCreateNoScopes(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)

	body := `{"name":"ci-token"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for missing scopes, got %d", resp.StatusCode)
	}
}

func TestDeployTokenCreateInvalidScope(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)

	body := `{"name":"ci-token","scopes":["invalid:scope"]}`
	req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for invalid scope, got %d", resp.StatusCode)
	}
}

func TestDeployTokenCreateInvalidExpiry(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)

	body := `{"name":"ci-token","scopes":["deploy:staging"],"expires_in":"invalid"}`
	req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for invalid expires_in, got %d", resp.StatusCode)
	}
}

func TestDeployTokenList(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)
	fiberApp.Get("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.List)

	// Create 2 tokens
	for _, name := range []string{"token-1", "token-2"} {
		body := `{"name":"` + name + `","scopes":["deploy:staging"]}`
		req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fiberApp.Test(req)
	}

	// List tokens
	listReq := httptest.NewRequest("GET", "/api/v1/projects/"+projectID+"/deploy-tokens", nil)
	listResp, _ := fiberApp.Test(listReq)

	if listResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", listResp.StatusCode)
	}

	var result struct {
		Tokens []map[string]interface{} `json:"tokens"`
	}
	json.NewDecoder(listResp.Body).Decode(&result)

	if len(result.Tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(result.Tokens))
	}
}

func TestDeployTokenRevoke(t *testing.T) {
	fiberApp, handler, projectRepo, tokenRepo := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)
	fiberApp.Delete("/api/v1/projects/:projectId/deploy-tokens/:tokenId", injectUserID("user-1"), handler.Revoke)

	// Create a token
	body := `{"name":"ci-token","scopes":["deploy:staging"]}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	tokenID := created["id"].(string)

	// Revoke
	revokeReq := httptest.NewRequest("DELETE", "/api/v1/projects/"+projectID+"/deploy-tokens/"+tokenID, nil)
	revokeResp, _ := fiberApp.Test(revokeReq)

	if revokeResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", revokeResp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(revokeResp.Body).Decode(&result)
	if result["message"] != "token revoked" {
		t.Errorf("Expected message 'token revoked', got '%v'", result["message"])
	}

	// Verify the token is revoked
	dt, _ := tokenRepo.GetDeployToken(nil, tokenID)
	if !dt.IsRevoked() {
		t.Error("Expected token to be revoked")
	}
}

func TestDeployTokenRevokeNotFound(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Delete("/api/v1/projects/:projectId/deploy-tokens/:tokenId", injectUserID("user-1"), handler.Revoke)

	req := httptest.NewRequest("DELETE", "/api/v1/projects/"+projectID+"/deploy-tokens/nonexistent", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeployTokenRotate(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)
	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens/:tokenId/rotate", injectUserID("user-1"), handler.Rotate)

	// Create a token
	body := `{"name":"ci-token","scopes":["deploy:staging"]}`
	createReq := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	tokenID := created["id"].(string)
	originalSecret := created["secret"].(string)

	// Rotate
	rotateReq := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens/"+tokenID+"/rotate", nil)
	rotateResp, _ := fiberApp.Test(rotateReq)

	if rotateResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", rotateResp.StatusCode)
	}

	var rotated map[string]interface{}
	json.NewDecoder(rotateResp.Body).Decode(&rotated)

	newSecret := rotated["secret"].(string)
	if newSecret == originalSecret {
		t.Error("Expected rotated secret to differ from original")
	}
}

func TestDeployTokenRotateNotFound(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens/:tokenId/rotate", injectUserID("user-1"), handler.Rotate)

	req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens/nonexistent/rotate", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeployTokenCreateDefaultExpiry(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)

	// No expires_in — should default to 90d
	body := `{"name":"ci-token","scopes":["deploy:staging"]}`
	req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["expires_at"] == nil {
		t.Error("Expected expires_at to be set with default 90d")
	}
}

func TestDeployTokenInvalidBody(t *testing.T) {
	fiberApp, handler, projectRepo, _ := setupDeployTokenTest()
	projectID := createProjectForTokenTest(t, projectRepo)

	fiberApp.Post("/api/v1/projects/:projectId/deploy-tokens", injectUserID("user-1"), handler.Create)

	req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/deploy-tokens", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for invalid body, got %d", resp.StatusCode)
	}
}
