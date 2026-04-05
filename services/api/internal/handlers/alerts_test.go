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

func setupAlertsTest() (*fiber.App, *handlers.AlertsHandler, string) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	// Business plan required for alerts
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	handler := handlers.NewAlertsHandler(appRepo, planRepo)

	// Create a test app owned by user-1
	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "test-app",
		RepoURL: "https://github.com/user/repo",
	})

	return app, handler, testApp.ID
}

func TestAlertsListRulesEmpty(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Get("/api/v1/apps/:appId/alerts", injectUserID("user-1"), handler.ListAlertRules)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/alerts", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Rules []entities.AlertRule `json:"rules"`
		Total int                  `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0 rules, got %d", result.Total)
	}
}

func TestAlertsCreateRule(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Post("/api/v1/apps/:appId/alerts", injectUserID("user-1"), handler.CreateAlertRule)

	body := `{"name":"High CPU","metric":"cpu_usage","condition":">80","severity":"critical","duration":"5m"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result entities.AlertRule
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Name != "High CPU" {
		t.Errorf("Expected name 'High CPU', got '%s'", result.Name)
	}
	if result.Metric != "cpu_usage" {
		t.Errorf("Expected metric 'cpu_usage', got '%s'", result.Metric)
	}
	if result.Severity != entities.AlertSeverityCritical {
		t.Errorf("Expected severity 'critical', got '%s'", result.Severity)
	}
	if !result.Enabled {
		t.Error("Expected enabled=true by default")
	}
}

func TestAlertsCreateRuleMissingFields(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Post("/api/v1/apps/:appId/alerts", injectUserID("user-1"), handler.CreateAlertRule)

	body := `{"name":"No metric"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestAlertsCreateRuleFreePlanForbidden(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	// user-1 defaults to free plan
	handler := handlers.NewAlertsHandler(appRepo, planRepo)

	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "test-app",
		RepoURL: "https://github.com/user/repo",
	})

	app.Post("/api/v1/apps/:appId/alerts", injectUserID("user-1"), handler.CreateAlertRule)

	body := `{"name":"High CPU","metric":"cpu_usage","condition":">80"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/alerts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestAlertsUpdateRule(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Post("/api/v1/apps/:appId/alerts", injectUserID("user-1"), handler.CreateAlertRule)
	app.Put("/api/v1/apps/:appId/alerts/:ruleId", injectUserID("user-1"), handler.UpdateAlertRule)

	// Create a rule first
	body := `{"name":"High CPU","metric":"cpu_usage","condition":">80"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/alerts", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created entities.AlertRule
	json.NewDecoder(createResp.Body).Decode(&created)

	// Update it
	updateBody := `{"name":"Updated Alert","enabled":false}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/apps/"+appID+"/alerts/"+created.ID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, _ := app.Test(updateReq)

	if updateResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", updateResp.StatusCode)
	}

	var result entities.AlertRule
	json.NewDecoder(updateResp.Body).Decode(&result)
	if result.Name != "Updated Alert" {
		t.Errorf("Expected 'Updated Alert', got '%s'", result.Name)
	}
	if result.Enabled {
		t.Error("Expected enabled=false")
	}
}

func TestAlertsUpdateRuleNotFound(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Put("/api/v1/apps/:appId/alerts/:ruleId", injectUserID("user-1"), handler.UpdateAlertRule)

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/apps/"+appID+"/alerts/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestAlertsDeleteRule(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Post("/api/v1/apps/:appId/alerts", injectUserID("user-1"), handler.CreateAlertRule)
	app.Delete("/api/v1/apps/:appId/alerts/:ruleId", injectUserID("user-1"), handler.DeleteAlertRule)

	// Create a rule
	body := `{"name":"To Delete","metric":"cpu_usage","condition":">80"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/alerts", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created entities.AlertRule
	json.NewDecoder(createResp.Body).Decode(&created)

	// Delete it
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/alerts/"+created.ID, nil)
	deleteResp, _ := app.Test(deleteReq)
	if deleteResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", deleteResp.StatusCode)
	}
}

func TestAlertsDeleteRuleNotFound(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Delete("/api/v1/apps/:appId/alerts/:ruleId", injectUserID("user-1"), handler.DeleteAlertRule)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/alerts/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

// --- Custom Metrics ---

func TestAlertsListMetricsEmpty(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Get("/api/v1/apps/:appId/custom-metrics", injectUserID("user-1"), handler.ListMetrics)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+appID+"/custom-metrics", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestAlertsCreateMetric(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Post("/api/v1/apps/:appId/custom-metrics", injectUserID("user-1"), handler.CreateMetric)

	body := `{"name":"request_count","expression":"sum(rate(http_requests_total[5m]))"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/custom-metrics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result entities.CustomMetric
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Name != "request_count" {
		t.Errorf("Expected name 'request_count', got '%s'", result.Name)
	}
}

func TestAlertsCreateMetricMissingFields(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Post("/api/v1/apps/:appId/custom-metrics", injectUserID("user-1"), handler.CreateMetric)

	body := `{"name":"incomplete"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/custom-metrics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestAlertsDeleteMetric(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Post("/api/v1/apps/:appId/custom-metrics", injectUserID("user-1"), handler.CreateMetric)
	app.Delete("/api/v1/apps/:appId/custom-metrics/:metricId", injectUserID("user-1"), handler.DeleteMetric)

	body := `{"name":"to_delete","expression":"sum(rate(http_requests_total[5m]))"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/custom-metrics", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created entities.CustomMetric
	json.NewDecoder(createResp.Body).Decode(&created)

	deleteReq := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/custom-metrics/"+created.ID, nil)
	deleteResp, _ := app.Test(deleteReq)
	if deleteResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", deleteResp.StatusCode)
	}
}

func TestAlertsDeleteMetricNotFound(t *testing.T) {
	app, handler, appID := setupAlertsTest()
	app.Delete("/api/v1/apps/:appId/custom-metrics/:metricId", injectUserID("user-1"), handler.DeleteMetric)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+appID+"/custom-metrics/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}
