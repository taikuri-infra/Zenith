package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupEnvVarTest() (*fiber.App, *handlers.EnvVarHandler, *memory.MemoryAppRepository, *memory.MemoryEnvVarRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	appRepo := memory.NewMemoryAppRepository()
	envVarRepo := memory.NewMemoryEnvVarRepository()
	handler := handlers.NewEnvVarHandler(appRepo, envVarRepo)
	return app, handler, appRepo, envVarRepo
}

func createAppForEnvVarTest(t *testing.T, appRepo *memory.MemoryAppRepository) string {
	t.Helper()
	a, err := appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "test-app",
		RepoURL: "https://github.com/user/repo",
	})
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	return a.ID
}

func TestEnvVarSet(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.Set)

	body := `{"vars":[{"key":"DATABASE_URL","value":"postgres://localhost/db"},{"key":"API_KEY","value":"secret123","is_secret":true}]}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	total := int(result["total"].(float64))
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
}

func TestEnvVarSetEmptyVars(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.Set)

	body := `{"vars":[]}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for empty vars, got %d", resp.StatusCode)
	}
}

func TestEnvVarSetEmptyKey(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.Set)

	body := `{"vars":[{"key":"","value":"something"}]}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for empty key, got %d", resp.StatusCode)
	}
}

func TestEnvVarSetDuplicateKey(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.Set)

	body := `{"vars":[{"key":"FOO","value":"bar"},{"key":"FOO","value":"baz"}]}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for duplicate key, got %d", resp.StatusCode)
	}
}

func TestEnvVarSetNoAuth(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2", handler.Set)

	body := `{"vars":[{"key":"FOO","value":"bar"}]}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestEnvVarSetForbidden(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo) // owned by user-1

	fiberApp.Post("/api/v1/apps/:appId/env-v2", injectUserID("user-2"), handler.Set)

	body := `{"vars":[{"key":"FOO","value":"bar"}]}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestEnvVarSetAppNotFound(t *testing.T) {
	fiberApp, handler, _, _ := setupEnvVarTest()

	fiberApp.Post("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.Set)

	body := `{"vars":[{"key":"FOO","value":"bar"}]}`
	req := httptest.NewRequest("POST", "/api/v1/apps/nonexistent/env-v2", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestEnvVarSetInvalidBody(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.Set)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestEnvVarList(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.Set)
	fiberApp.Get("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.List)

	// Set some vars first
	body := `{"vars":[{"key":"FOO","value":"bar"},{"key":"BAZ","value":"qux"}]}`
	setReq := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2", bytes.NewBufferString(body))
	setReq.Header.Set("Content-Type", "application/json")
	fiberApp.Test(setReq)

	// List vars
	listReq := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/env-v2", nil)
	listResp, _ := fiberApp.Test(listReq)

	if listResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", listResp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(listResp.Body).Decode(&result)

	total := int(result["total"].(float64))
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
}

func TestEnvVarListEmpty(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Get("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/env-v2", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	total := int(result["total"].(float64))
	if total != 0 {
		t.Errorf("Expected total 0, got %d", total)
	}
}

func TestEnvVarDelete(t *testing.T) {
	fiberApp, handler, appRepo, envVarRepo := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.Set)
	fiberApp.Delete("/api/v1/apps/:appId/env-v2/:varId", injectUserID("user-1"), handler.Delete)

	// Set a var first
	body := `{"vars":[{"key":"FOO","value":"bar"}]}`
	setReq := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2", bytes.NewBufferString(body))
	setReq.Header.Set("Content-Type", "application/json")
	fiberApp.Test(setReq)

	// Get the var ID
	vars, _ := envVarRepo.GetEnvVars(nil, appID)
	if len(vars) == 0 {
		t.Fatal("Expected at least 1 env var")
	}
	varID := vars[0].ID

	// Delete the var
	delReq := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/env-v2/"+varID, nil)
	delResp, _ := fiberApp.Test(delReq)

	if delResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", delResp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(delResp.Body).Decode(&result)

	if result["message"] != "environment variable deleted" {
		t.Errorf("Expected message 'environment variable deleted', got '%v'", result["message"])
	}
}

func TestEnvVarDeleteNotFound(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Delete("/api/v1/apps/:appId/env-v2/:varId", injectUserID("user-1"), handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/env-v2/nonexistent", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestEnvVarImportDotEnv(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2/import", injectUserID("user-1"), handler.ImportDotEnv)

	body := `{"content":"DATABASE_URL=postgres://localhost/db\nAPI_KEY=secret123\n# comment\nPORT=3000"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	imported := int(result["imported"].(float64))
	if imported != 3 {
		t.Errorf("Expected 3 imported, got %d", imported)
	}
}

func TestEnvVarImportDotEnvEmptyContent(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2/import", injectUserID("user-1"), handler.ImportDotEnv)

	body := `{"content":""}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for empty content, got %d", resp.StatusCode)
	}
}

func TestEnvVarImportDotEnvOnlyComments(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2/import", injectUserID("user-1"), handler.ImportDotEnv)

	body := `{"content":"# this is a comment\n# another comment"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for no valid pairs, got %d", resp.StatusCode)
	}
}

func TestEnvVarSecretMasked(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.Set)
	fiberApp.Get("/api/v1/apps/:appId/env-v2", injectUserID("user-1"), handler.List)

	// Set a secret var
	body := `{"vars":[{"key":"DB_PASSWORD","value":"supersecret","is_secret":true}]}`
	setReq := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2", bytes.NewBufferString(body))
	setReq.Header.Set("Content-Type", "application/json")
	fiberApp.Test(setReq)

	// List and check secret is masked
	listReq := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/env-v2", nil)
	listResp, _ := fiberApp.Test(listReq)

	var result struct {
		Items []struct {
			Key      string `json:"key"`
			Value    string `json:"value"`
			IsSecret bool   `json:"is_secret"`
		} `json:"items"`
	}
	json.NewDecoder(listResp.Body).Decode(&result)

	if len(result.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(result.Items))
	}
	if !result.Items[0].IsSecret {
		t.Error("Expected is_secret to be true")
	}
	if result.Items[0].Value == "supersecret" {
		t.Error("Secret value should be masked, not returned as plaintext")
	}
}

func TestEnvVarApplyNoRestarter(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/env/apply", injectUserID("user-1"), handler.Apply)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env/apply", nil)
	resp, _ := fiberApp.Test(req)

	// Should fail because no restarter is configured
	if resp.StatusCode != 500 {
		t.Errorf("Expected 500 (no restarter), got %d", resp.StatusCode)
	}
}
