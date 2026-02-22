package handlers_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

func setupV2Test() (*fiber.App, *handlers.AppHandlerV2, *handlers.DeployHandler, *handlers.WebhookHandler, ports.AppRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	repo := memory.NewMemoryAppRepository()
	appHandler := handlers.NewAppHandlerV2(repo, "freezenith.com")
	deployHandler := handlers.NewDeployHandler(repo)
	webhookHandler := handlers.NewWebhookHandler(repo, nil, "test-secret")
	return app, appHandler, deployHandler, webhookHandler, repo
}

// injectUserID is middleware for tests to simulate auth
func injectUserID(userID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		return c.Next()
	}
}

// --- AppHandlerV2 tests ---

func TestV2CreateApp(t *testing.T) {
	app, handler, _, _, _ := setupV2Test()
	app.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.AppV2Response
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Name != "web" {
		t.Errorf("Expected name 'web', got '%s'", result.Name)
	}
	if result.RepoURL != "https://github.com/user/repo" {
		t.Errorf("Expected repo_url, got '%s'", result.RepoURL)
	}
	if result.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", result.Status)
	}
	if result.URL != "https://web.freezenith.com" {
		t.Errorf("Expected URL 'https://web.freezenith.com', got '%s'", result.URL)
	}
}

func TestV2CreateAppNoName(t *testing.T) {
	app, handler, _, _, _ := setupV2Test()
	app.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"repo_url":"https://github.com/user/repo"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestV2CreateAppNoRepoURL(t *testing.T) {
	app, handler, _, _, _ := setupV2Test()
	app.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"web"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestV2CreateAppNoAuth(t *testing.T) {
	app, handler, _, _, _ := setupV2Test()
	// No injectUserID middleware
	app.Post("/api/v1/apps", handler.Create)

	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestV2ListApps(t *testing.T) {
	app, handler, _, _, _ := setupV2Test()
	app.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)
	app.Get("/api/v1/apps", injectUserID("user-1"), handler.List)

	// Create 2 apps
	for _, name := range []string{"web", "api"} {
		body := `{"name":"` + name + `","repo_url":"https://github.com/user/repo"}`
		req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}

	req := httptest.NewRequest("GET", "/api/v1/apps", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []handlers.AppV2Response `json:"items"`
		Total int                      `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 2 {
		t.Errorf("Expected 2 apps, got %d", result.Total)
	}
}

func TestV2GetApp(t *testing.T) {
	app, handler, _, _, _ := setupV2Test()
	app.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)
	app.Get("/api/v1/apps/:appId", injectUserID("user-1"), handler.Get)

	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	getReq := httptest.NewRequest("GET", "/api/v1/apps/"+created.ID, nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", getResp.StatusCode)
	}
}

func TestV2GetAppNotFound(t *testing.T) {
	app, handler, _, _, _ := setupV2Test()
	app.Get("/api/v1/apps/:appId", injectUserID("user-1"), handler.Get)

	req := httptest.NewRequest("GET", "/api/v1/apps/nonexistent", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestV2DeleteApp(t *testing.T) {
	app, handler, _, _, _ := setupV2Test()
	app.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)
	app.Delete("/api/v1/apps/:appId", injectUserID("user-1"), handler.Delete)

	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created handlers.AppV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	deleteReq := httptest.NewRequest("DELETE", "/api/v1/apps/"+created.ID, nil)
	deleteResp, _ := app.Test(deleteReq)

	if deleteResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", deleteResp.StatusCode)
	}
}

// --- Webhook tests ---

func TestWebhookPingEvent(t *testing.T) {
	app, _, _, webhookHandler, _ := setupV2Test()
	app.Post("/webhooks/github", webhookHandler.HandlePush)

	body := `{"zen":"Keep it logically awesome."}`
	req := httptest.NewRequest("POST", "/webhooks/github", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "ping")

	// Compute HMAC signature
	mac := hmac.New(sha256.New, []byte("test-secret"))
	mac.Write([]byte(body))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	req.Header.Set("X-Hub-Signature-256", sig)

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "event ignored" {
		t.Errorf("Expected 'event ignored', got '%v'", result["message"])
	}
}

func TestWebhookInvalidSignature(t *testing.T) {
	app, _, _, webhookHandler, _ := setupV2Test()
	app.Post("/webhooks/github", webhookHandler.HandlePush)

	body := `{"ref":"refs/heads/main"}`
	req := httptest.NewRequest("POST", "/webhooks/github", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", "sha256=invalidsignature")

	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestWebhookMissingSignature(t *testing.T) {
	app, _, _, webhookHandler, _ := setupV2Test()
	app.Post("/webhooks/github", webhookHandler.HandlePush)

	body := `{"ref":"refs/heads/main"}`
	req := httptest.NewRequest("POST", "/webhooks/github", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")
	// No signature header

	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestWebhookPushNoMatchingApps(t *testing.T) {
	app, _, _, webhookHandler, _ := setupV2Test()
	app.Post("/webhooks/github", webhookHandler.HandlePush)

	body := `{"ref":"refs/heads/main","after":"abc123def456","repository":{"full_name":"user/repo","clone_url":"https://github.com/user/repo.git"},"head_commit":{"id":"abc123","message":"test"}}`
	req := httptest.NewRequest("POST", "/webhooks/github", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")

	mac := hmac.New(sha256.New, []byte("test-secret"))
	mac.Write([]byte(body))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	req.Header.Set("X-Hub-Signature-256", sig)

	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "no matching apps" {
		t.Errorf("Expected 'no matching apps', got '%v'", result["message"])
	}
}

// --- Deploy handler tests ---

func TestListDeployments(t *testing.T) {
	fiberApp, appHandler, deployHandler, _, appRepo := setupV2Test()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), appHandler.Create)
	fiberApp.Get("/api/v1/apps/:appId/deployments", injectUserID("user-1"), deployHandler.ListDeployments)

	// Create app
	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq)

	var created handlers.AppV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	// Create deployments directly via repo
	appRepo.CreateDeployment(nil, created.ID, "sha1")
	appRepo.CreateDeployment(nil, created.ID, "sha2")

	// List deployments
	listReq := httptest.NewRequest("GET", "/api/v1/apps/"+created.ID+"/deployments", nil)
	listResp, _ := fiberApp.Test(listReq)

	if listResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", listResp.StatusCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(listResp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 deployments, got %d", result.Total)
	}
}

func TestSetAndGetEnvVars(t *testing.T) {
	fiberApp, appHandler, deployHandler, _, _ := setupV2Test()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), appHandler.Create)
	fiberApp.Put("/api/v1/apps/:appId/env", injectUserID("user-1"), deployHandler.SetEnvVars)
	fiberApp.Get("/api/v1/apps/:appId/env", injectUserID("user-1"), deployHandler.GetEnvVars)

	// Create app
	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq)

	var created handlers.AppV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	// Set env vars
	envBody := `{"vars":{"DATABASE_URL":"postgres://...","API_KEY":"secret"}}`
	setReq := httptest.NewRequest("PUT", "/api/v1/apps/"+created.ID+"/env", bytes.NewBufferString(envBody))
	setReq.Header.Set("Content-Type", "application/json")
	setResp, _ := fiberApp.Test(setReq)

	if setResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", setResp.StatusCode)
	}

	// Get env vars
	getReq := httptest.NewRequest("GET", "/api/v1/apps/"+created.ID+"/env", nil)
	getResp, _ := fiberApp.Test(getReq)

	if getResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", getResp.StatusCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(getResp.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("Expected 2 env vars, got %d", result.Total)
	}
}

func TestDeleteEnvVar(t *testing.T) {
	fiberApp, appHandler, deployHandler, _, _ := setupV2Test()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), appHandler.Create)
	fiberApp.Put("/api/v1/apps/:appId/env", injectUserID("user-1"), deployHandler.SetEnvVars)
	fiberApp.Delete("/api/v1/apps/:appId/env/:key", injectUserID("user-1"), deployHandler.DeleteEnvVar)

	// Create app
	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq)

	var created handlers.AppV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	// Set env var
	envBody := `{"vars":{"MY_KEY":"value"}}`
	setReq := httptest.NewRequest("PUT", "/api/v1/apps/"+created.ID+"/env", bytes.NewBufferString(envBody))
	setReq.Header.Set("Content-Type", "application/json")
	fiberApp.Test(setReq)

	// Delete env var
	delReq := httptest.NewRequest("DELETE", "/api/v1/apps/"+created.ID+"/env/MY_KEY", nil)
	delResp, _ := fiberApp.Test(delReq)

	if delResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", delResp.StatusCode)
	}
}

// --- DatabaseHandlerV2 tests ---

func setupDatabaseTest() (*fiber.App, *handlers.DatabaseHandlerV2, ports.AppRepository, *memory.MemoryDatabaseRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	dbHandler := handlers.NewDatabaseHandlerV2(dbRepo, appRepo)
	return app, dbHandler, appRepo, dbRepo
}

func createTestApp(t *testing.T, fiberApp *fiber.App, appRepo ports.AppRepository) string {
	t.Helper()
	app, err := appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "test-app",
		RepoURL: "https://github.com/user/repo",
	})
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	return app.ID
}

func TestV2CreateDatabase(t *testing.T) {
	fiberApp, dbHandler, appRepo, _ := setupDatabaseTest()
	appID := createTestApp(t, fiberApp, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/databases", injectUserID("user-1"), dbHandler.Create)

	body := `{"engine":"postgresql"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result dto.DatabaseInfo
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Engine != "postgresql" {
		t.Errorf("Expected engine postgresql, got %s", result.Engine)
	}
	if result.Status != "ready" {
		t.Errorf("Expected status ready, got %s", result.Status)
	}
}

func TestV2CreateDatabaseDefaultEngine(t *testing.T) {
	fiberApp, dbHandler, appRepo, _ := setupDatabaseTest()
	appID := createTestApp(t, fiberApp, appRepo)

	fiberApp.Post("/api/v1/apps/:appId/databases", injectUserID("user-1"), dbHandler.Create)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result dto.DatabaseInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Engine != "postgresql" {
		t.Errorf("Expected default engine postgresql, got %s", result.Engine)
	}
}

func TestV2CreateDatabaseAppNotFound(t *testing.T) {
	fiberApp, dbHandler, _, _ := setupDatabaseTest()
	fiberApp.Post("/api/v1/apps/:appId/databases", injectUserID("user-1"), dbHandler.Create)

	body := `{"engine":"postgresql"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/nonexistent/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestV2CreateDatabaseForbidden(t *testing.T) {
	fiberApp, dbHandler, appRepo, _ := setupDatabaseTest()
	appID := createTestApp(t, fiberApp, appRepo) // owned by user-1

	fiberApp.Post("/api/v1/apps/:appId/databases", injectUserID("user-2"), dbHandler.Create)

	body := `{"engine":"postgresql"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/databases", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestV2ListDatabases(t *testing.T) {
	fiberApp, dbHandler, appRepo, dbRepo := setupDatabaseTest()
	appID := createTestApp(t, fiberApp, appRepo)

	dbRepo.CreateDatabase(nil, appID, "user-1", &dto.CreateDatabaseInput{Engine: "postgresql"})
	dbRepo.CreateDatabase(nil, appID, "user-1", &dto.CreateDatabaseInput{Engine: "redis"})

	fiberApp.Get("/api/v1/apps/:appId/databases", injectUserID("user-1"), dbHandler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/databases", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.DatabaseInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("Expected 2 databases, got %d", len(result))
	}
}

func TestV2GetDatabase(t *testing.T) {
	fiberApp, dbHandler, appRepo, dbRepo := setupDatabaseTest()
	appID := createTestApp(t, fiberApp, appRepo)

	db, _ := dbRepo.CreateDatabase(nil, appID, "user-1", &dto.CreateDatabaseInput{Engine: "postgresql"})

	fiberApp.Get("/api/v1/apps/:appId/databases/:dbId", injectUserID("user-1"), dbHandler.Get)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/databases/"+db.ID, nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result dto.DatabaseInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if result.ConnectionString == "" {
		t.Error("Expected connection string to be returned")
	}
}

func TestV2GetDatabaseNotFound(t *testing.T) {
	fiberApp, dbHandler, _, _ := setupDatabaseTest()
	fiberApp.Get("/api/v1/apps/:appId/databases/:dbId", injectUserID("user-1"), dbHandler.Get)

	req := httptest.NewRequest("GET", "/api/v1/apps/app-1/databases/nonexistent", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestV2DeleteDatabase(t *testing.T) {
	fiberApp, dbHandler, appRepo, dbRepo := setupDatabaseTest()
	appID := createTestApp(t, fiberApp, appRepo)

	db, _ := dbRepo.CreateDatabase(nil, appID, "user-1", &dto.CreateDatabaseInput{Engine: "postgresql"})

	fiberApp.Delete("/api/v1/apps/:appId/databases/:dbId", injectUserID("user-1"), dbHandler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/databases/"+db.ID, nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestV2DeleteDatabaseForbidden(t *testing.T) {
	fiberApp, dbHandler, appRepo, dbRepo := setupDatabaseTest()
	appID := createTestApp(t, fiberApp, appRepo)

	db, _ := dbRepo.CreateDatabase(nil, appID, "user-1", &dto.CreateDatabaseInput{Engine: "postgresql"})

	fiberApp.Delete("/api/v1/apps/:appId/databases/:dbId", injectUserID("user-2"), dbHandler.Delete)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/databases/"+db.ID, nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestV2ListDatabasesByUser(t *testing.T) {
	fiberApp, dbHandler, appRepo, dbRepo := setupDatabaseTest()
	appID := createTestApp(t, fiberApp, appRepo)

	dbRepo.CreateDatabase(nil, appID, "user-1", &dto.CreateDatabaseInput{Engine: "postgresql"})

	fiberApp.Get("/api/v1/databases", injectUserID("user-1"), dbHandler.ListByUser)

	req := httptest.NewRequest("GET", "/api/v1/databases", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []dto.DatabaseInfo
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result) != 1 {
		t.Errorf("Expected 1 database, got %d", len(result))
	}
}
