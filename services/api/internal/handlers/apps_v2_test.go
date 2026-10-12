package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupAppsV2Extended() (*fiber.App, *handlers.AppHandlerV2, *memory.MemoryAppRepository, *memory.MemoryProjectRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	appRepo := memory.NewMemoryAppRepository()
	projectRepo := memory.NewMemoryProjectRepository()
	handler := handlers.NewAppHandlerV2(appRepo, "freezenith.com", nil, nil)
	handler.SetProjectRepo(projectRepo)
	return app, handler, appRepo, projectRepo
}

// --- Restore / SoftDelete / ListDeleted tests ---

func TestV2SoftDeleteAndRestore(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)
	fiberApp.Delete("/api/v1/apps/:appId", injectUserID("user-1"), handler.Delete)
	fiberApp.Post("/api/v1/apps/:appId/restore", injectUserID("user-1"), handler.Restore)
	fiberApp.Get("/api/v1/apps/:appId", injectUserID("user-1"), handler.Get)

	// Create app
	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq, -1)

	var created handlers.AppV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	// Soft delete
	delReq := httptest.NewRequest("DELETE", "/api/v1/apps/"+created.ID, nil)
	delResp, _ := fiberApp.Test(delReq, -1)
	if delResp.StatusCode != 200 {
		t.Fatalf("Expected 200 for soft delete, got %d", delResp.StatusCode)
	}

	// After soft delete, Get should return 404
	getReq := httptest.NewRequest("GET", "/api/v1/apps/"+created.ID, nil)
	getResp, _ := fiberApp.Test(getReq, -1)
	if getResp.StatusCode != 404 {
		t.Errorf("Expected 404 after soft delete, got %d", getResp.StatusCode)
	}

	// Restore
	restoreReq := httptest.NewRequest("POST", "/api/v1/apps/"+created.ID+"/restore", nil)
	restoreResp, _ := fiberApp.Test(restoreReq, -1)
	if restoreResp.StatusCode != 200 {
		t.Fatalf("Expected 200 for restore, got %d", restoreResp.StatusCode)
	}

	// After restore, Get should return 200
	getReq2 := httptest.NewRequest("GET", "/api/v1/apps/"+created.ID, nil)
	getResp2, _ := fiberApp.Test(getReq2, -1)
	if getResp2.StatusCode != 200 {
		t.Errorf("Expected 200 after restore, got %d", getResp2.StatusCode)
	}
}

func TestV2ListDeleted(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)
	fiberApp.Delete("/api/v1/apps/:appId", injectUserID("user-1"), handler.Delete)
	fiberApp.Get("/api/v1/apps/trash", injectUserID("user-1"), handler.ListDeleted)

	// Create and soft-delete an app
	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq, -1)

	var created handlers.AppV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	delReq := httptest.NewRequest("DELETE", "/api/v1/apps/"+created.ID, nil)
	fiberApp.Test(delReq, -1)

	// List deleted
	listReq := httptest.NewRequest("GET", "/api/v1/apps/trash", nil)
	listResp, _ := fiberApp.Test(listReq, -1)

	if listResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", listResp.StatusCode)
	}

	var result struct {
		Items []handlers.AppV2Response `json:"items"`
		Total int                      `json:"total"`
	}
	json.NewDecoder(listResp.Body).Decode(&result)

	if result.Total != 1 {
		t.Errorf("Expected 1 deleted app, got %d", result.Total)
	}
}

func TestV2ListDeletedEmpty(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Get("/api/v1/apps/trash", injectUserID("user-1"), handler.ListDeleted)

	req := httptest.NewRequest("GET", "/api/v1/apps/trash", nil)
	resp, _ := fiberApp.Test(req, -1)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 0 {
		t.Errorf("Expected 0 deleted apps, got %d", result.Total)
	}
}

func TestV2ListDeletedNoAuth(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Get("/api/v1/apps/trash", handler.ListDeleted)

	req := httptest.NewRequest("GET", "/api/v1/apps/trash", nil)
	resp, _ := fiberApp.Test(req, -1)

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestV2RestoreNoAuth(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps/:appId/restore", handler.Restore)

	req := httptest.NewRequest("POST", "/api/v1/apps/some-id/restore", nil)
	resp, _ := fiberApp.Test(req, -1)

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestV2RestoreNotFound(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps/:appId/restore", injectUserID("user-1"), handler.Restore)

	req := httptest.NewRequest("POST", "/api/v1/apps/nonexistent/restore", nil)
	resp, _ := fiberApp.Test(req, -1)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

// --- CheckName tests ---

func TestV2CheckName(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Get("/api/v1/apps/check-name", injectUserID("user-1"), handler.CheckName)

	req := httptest.NewRequest("GET", "/api/v1/apps/check-name?name=my-cool-app", nil)
	resp, _ := fiberApp.Test(req, -1)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["subdomain"] == nil || result["subdomain"] == "" {
		t.Error("Expected subdomain to be generated")
	}
	if result["url"] == nil || result["url"] == "" {
		t.Error("Expected url to be generated")
	}
	if result["available"] != true {
		t.Error("Expected name to be available")
	}
}

func TestV2CheckNameEmpty(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Get("/api/v1/apps/check-name", injectUserID("user-1"), handler.CheckName)

	req := httptest.NewRequest("GET", "/api/v1/apps/check-name", nil)
	resp, _ := fiberApp.Test(req, -1)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for empty name, got %d", resp.StatusCode)
	}
}

// --- HardDelete tests ---

func TestV2HardDelete(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)
	fiberApp.Delete("/api/v1/apps/:appId", injectUserID("user-1"), handler.Delete)
	fiberApp.Post("/api/v1/apps/:appId/restore", injectUserID("user-1"), handler.Restore)

	// Create app
	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq, -1)

	var created handlers.AppV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	// Hard delete
	delReq := httptest.NewRequest("DELETE", "/api/v1/apps/"+created.ID+"?hard=true", nil)
	delResp, _ := fiberApp.Test(delReq, -1)
	if delResp.StatusCode != 200 {
		t.Fatalf("Expected 200 for hard delete, got %d", delResp.StatusCode)
	}

	// After hard delete, Restore should return 404
	restoreReq := httptest.NewRequest("POST", "/api/v1/apps/"+created.ID+"/restore", nil)
	restoreResp, _ := fiberApp.Test(restoreReq, -1)
	if restoreResp.StatusCode != 404 {
		t.Errorf("Expected 404 after hard delete, got %d", restoreResp.StatusCode)
	}
}

// --- List by project ---

func TestV2ListByProject(t *testing.T) {
	fiberApp, handler, appRepo, projectRepo := setupAppsV2Extended()

	// Create project
	project, _ := projectRepo.CreateProject(nil, "user-1", "my-proj", "my-proj", "")

	// Create app via repo with project_id
	appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:    "user-1",
		ProjectID: project.ID,
		Name:      "app1",
		RepoURL:   "https://github.com/user/repo1",
	})
	appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:    "user-1",
		ProjectID: project.ID,
		Name:      "app2",
		RepoURL:   "https://github.com/user/repo2",
	})

	fiberApp.Get("/api/v1/apps", injectUserID("user-1"), handler.List)

	req := httptest.NewRequest("GET", "/api/v1/apps?project_id="+project.ID, nil)
	resp, _ := fiberApp.Test(req, -1)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []handlers.AppV2Response `json:"items"`
		Total int                      `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 2 {
		t.Errorf("Expected 2 apps for project, got %d", result.Total)
	}
}

// --- Worker / Cron app type tests ---

func TestV2CreateWorkerApp(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"worker","deploy_source":"image","image_url":"myworker:latest","app_type":"worker","command":"python worker.py"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("Expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var result handlers.AppV2Response
	json.NewDecoder(resp.Body).Decode(&result)

	if result.AppType != "worker" {
		t.Errorf("Expected app_type 'worker', got '%s'", result.AppType)
	}
}

func TestV2CreateCronApp(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"cron-job","deploy_source":"image","image_url":"mycron:latest","app_type":"cron","cron_schedule":"*/5 * * * *"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("Expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var result handlers.AppV2Response
	json.NewDecoder(resp.Body).Decode(&result)

	if result.AppType != "cron" {
		t.Errorf("Expected app_type 'cron', got '%s'", result.AppType)
	}
}

func TestV2CreateInvalidAppType(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"bad","deploy_source":"image","image_url":"img:latest","app_type":"invalid"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for invalid app_type, got %d", resp.StatusCode)
	}
}

// --- Exposure tests ---

func TestV2CreateProtectedApp(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"protected-app","repo_url":"https://github.com/user/repo","exposure":"protected"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.AppV2Response
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Exposure != "protected" {
		t.Errorf("Expected exposure 'protected', got '%s'", result.Exposure)
	}
}

func TestV2CreateInvalidExposure(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"bad","repo_url":"https://github.com/user/repo","exposure":"invalid"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for invalid exposure, got %d", resp.StatusCode)
	}
}

// --- HealthCheckPath validation ---

func TestV2CreateInvalidHealthCheckPath(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"app","repo_url":"https://github.com/user/repo","health_check_path":"no-slash"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for health_check_path without leading /, got %d", resp.StatusCode)
	}
}

func TestV2CreateCustomHealthCheckPath(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"app","repo_url":"https://github.com/user/repo","health_check_path":"/healthz"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result handlers.AppV2Response
	json.NewDecoder(resp.Body).Decode(&result)

	if result.HealthCheckPath != "/healthz" {
		t.Errorf("Expected health_check_path '/healthz', got '%s'", result.HealthCheckPath)
	}
}

// --- Image normalization tests ---

func TestV2CreateImageNormalization(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	// Bare name like "nginx" should be normalized
	body := `{"name":"web","deploy_source":"image","image_url":"nginx"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("Expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var result handlers.AppV2Response
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ImageURL != "docker.io/library/nginx:latest" {
		t.Errorf("Expected normalized image_url 'docker.io/library/nginx:latest', got '%s'", result.ImageURL)
	}
	// nginx well-known port is 80
	if result.Port != 80 {
		t.Errorf("Expected port 80 for nginx, got %d", result.Port)
	}
}

func TestV2CreateWithEnvVars(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"app","repo_url":"https://github.com/user/repo","env_vars":[{"key":"NODE_ENV","value":"production"},{"key":"PORT","value":"3000"}]}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("Expected 201, got %d: %v", resp.StatusCode, errBody)
	}
}

// --- Delete forbidden (another user) ---

func TestV2DeleteForbidden(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)
	fiberApp.Delete("/api/v1/apps/:appId", injectUserID("user-2"), handler.Delete)

	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq, -1)

	var created handlers.AppV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	delReq := httptest.NewRequest("DELETE", "/api/v1/apps/"+created.ID, nil)
	delResp, _ := fiberApp.Test(delReq, -1)

	if delResp.StatusCode != 404 {
		t.Errorf("Expected 404 (not your app), got %d", delResp.StatusCode)
	}
}

// --- Get forbidden ---

func TestV2GetForbidden(t *testing.T) {
	fiberApp, handler, _, _ := setupAppsV2Extended()
	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)
	fiberApp.Get("/api/v1/apps/:appId", injectUserID("user-2"), handler.Get)

	body := `{"name":"web","repo_url":"https://github.com/user/repo"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq, -1)

	var created handlers.AppV2Response
	json.NewDecoder(createResp.Body).Decode(&created)

	getReq := httptest.NewRequest("GET", "/api/v1/apps/"+created.ID, nil)
	getResp, _ := fiberApp.Test(getReq, -1)

	// Should return 404 (not your app — no information leakage)
	if getResp.StatusCode != 404 {
		t.Errorf("Expected 404 (not your app), got %d", getResp.StatusCode)
	}
}

// --- Plan limit on app creation ---

func TestV2CreateAppPlanLimit(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupAppsV2Extended()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler.SetPlanRepo(planRepo)

	// Free plan typically allows 1 app — create one first
	appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "existing-app",
		RepoURL: "https://github.com/user/repo",
	})

	fiberApp.Post("/api/v1/apps", injectUserID("user-1"), handler.Create)

	body := `{"name":"second-app","repo_url":"https://github.com/user/repo2"}`
	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for plan limit, got %d", resp.StatusCode)
	}
}

func TestAppCreate_CreatesDefaultProjectWhenNone(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	projectRepo := memory.NewMemoryProjectRepository()
	h := handlers.NewAppHandlerV2(appRepo, "example.com", nil, nil)
	h.SetProjectRepo(projectRepo)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Post("/api/v1/apps", injectUserID("user-1"), h.Create)

	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(`{"name":"whoami","deploy_source":"image","image_url":"traefik/whoami:latest","port":80}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d: %s", resp.StatusCode, b)
	}
	ps, _ := projectRepo.ListProjectsByUser(context.Background(), "user-1")
	if len(ps) != 1 || ps[0].Name != "Default" {
		t.Fatalf("projects = %+v, want one named Default", ps)
	}
}

// raceConditionProjectRepo wraps a ProjectRepository and simulates a concurrent winner
// by returning a duplicate error on CreateProject after the inner repo creates it.
type raceConditionProjectRepo struct {
	inner *memory.MemoryProjectRepository
}

func newRaceConditionProjectRepo() *raceConditionProjectRepo {
	return &raceConditionProjectRepo{
		inner: memory.NewMemoryProjectRepository(),
	}
}

func (r *raceConditionProjectRepo) GetProject(ctx context.Context, id string) (*entities.Project, error) {
	return r.inner.GetProject(ctx, id)
}

func (r *raceConditionProjectRepo) GetDefaultProject(ctx context.Context, userID string) (*entities.Project, error) {
	return r.inner.GetDefaultProject(ctx, userID)
}

func (r *raceConditionProjectRepo) CreateProject(ctx context.Context, userID, name, slug, description string) (*entities.Project, error) {
	// First, actually create the project (simulating the concurrent winner)
	_, _ = r.inner.CreateProject(ctx, userID, name, slug, description)
	// Then return a duplicate error (simulating the loser hitting the unique constraint)
	return nil, fmt.Errorf("duplicate project slug")
}

func (r *raceConditionProjectRepo) ListProjectsByUser(ctx context.Context, userID string) ([]entities.Project, error) {
	return r.inner.ListProjectsByUser(ctx, userID)
}

func (r *raceConditionProjectRepo) UpdateProject(ctx context.Context, id string, name, description *string) (*entities.Project, error) {
	return r.inner.UpdateProject(ctx, id, name, description)
}

func (r *raceConditionProjectRepo) DeleteProject(ctx context.Context, id string) error {
	return r.inner.DeleteProject(ctx, id)
}

func (r *raceConditionProjectRepo) CountProjectsByUser(ctx context.Context, userID string) (int, error) {
	return r.inner.CountProjectsByUser(ctx, userID)
}

func (r *raceConditionProjectRepo) SetHarborCredentials(ctx context.Context, id, harborProjectName, robotUser, robotPass string) error {
	return r.inner.SetHarborCredentials(ctx, id, harborProjectName, robotUser, robotPass)
}

func (r *raceConditionProjectRepo) UpdateProjectStatus(ctx context.Context, id string, status entities.ProjectStatus) error {
	return r.inner.UpdateProjectStatus(ctx, id, status)
}

func TestAppCreate_ConcurrentDefaultProjectCreation(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	projectRepo := newRaceConditionProjectRepo()
	h := handlers.NewAppHandlerV2(appRepo, "example.com", nil, nil)
	h.SetProjectRepo(projectRepo)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Post("/api/v1/apps", injectUserID("user-1"), h.Create)

	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(`{"name":"whoami","deploy_source":"image","image_url":"traefik/whoami:latest","port":80}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d (expected 201): %s", resp.StatusCode, b)
	}
	ps, _ := projectRepo.ListProjectsByUser(context.Background(), "user-1")
	if len(ps) != 1 || ps[0].Name != "Default" {
		t.Fatalf("projects = %+v, want one named Default", ps)
	}
}
