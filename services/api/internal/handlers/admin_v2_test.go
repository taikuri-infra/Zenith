package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

// injectTestAdmin is a middleware that injects admin context into the request.
func injectTestAdmin(c *fiber.Ctx) error {
	c.Locals("user_id", "admin-001")
	c.Locals("email", "admin@zenith.dev")
	c.Locals("name", "Admin")
	c.Locals("role", entities.RoleAdmin)
	return c.Next()
}

// newTestApp creates a Fiber app configured with the standard error handler.
func newTestApp() *fiber.App {
	return fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
}

// requireJSON unmarshals the response body into the given target.
func requireJSON(t *testing.T, body io.Reader, target interface{}) {
	t.Helper()
	b, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if err := json.Unmarshal(b, target); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nbody: %s", err, string(b))
	}
}

// jsonBody encodes the given value as JSON and returns a reader.
func jsonBody(t *testing.T, v interface{}) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON body: %v", err)
	}
	return bytes.NewReader(b)
}

// ============================================================================
// 1. AdminAnalyticsHandler
// ============================================================================

func TestAnalytics_GetWarRoom(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminAnalyticsHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/war-room", h.GetWarRoom)

	req := httptest.NewRequest("GET", "/api/v1/admin/war-room", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.WarRoomData
	requireJSON(t, resp.Body, &data)
}

func TestAnalytics_GetRevenue(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminAnalyticsHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/analytics/revenue", h.GetRevenue)

	req := httptest.NewRequest("GET", "/api/v1/admin/analytics/revenue", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.RevenueStats
	requireJSON(t, resp.Body, &data)
}

func TestAnalytics_GetGrowth(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminAnalyticsHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/analytics/growth", h.GetGrowth)

	req := httptest.NewRequest("GET", "/api/v1/admin/analytics/growth", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.GrowthStats
	requireJSON(t, resp.Body, &data)
}

func TestAnalytics_GetUsageAnalytics(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminAnalyticsHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/analytics/usage", h.GetUsageAnalytics)

	req := httptest.NewRequest("GET", "/api/v1/admin/analytics/usage", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.UsageStats
	requireJSON(t, resp.Body, &data)
}

func TestAnalytics_GetCohorts(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminAnalyticsHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/analytics/cohorts", h.GetCohorts)

	req := httptest.NewRequest("GET", "/api/v1/admin/analytics/cohorts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.CohortData
	requireJSON(t, resp.Body, &data)
}

// ============================================================================
// 2. AdminCRMHandler
// ============================================================================

func TestCRM_GetPipeline(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminCRMHandler(nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/crm/pipeline", h.GetPipeline)

	req := httptest.NewRequest("GET", "/api/v1/admin/crm/pipeline", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.CRMPipeline
	requireJSON(t, resp.Body, &data)
}

func TestCRM_GetHealthScores(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminCRMHandler(nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/crm/health-scores", h.GetHealthScores)

	req := httptest.NewRequest("GET", "/api/v1/admin/crm/health-scores", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.HealthScore
	requireJSON(t, resp.Body, &data)
}

func TestCRM_SaveNote(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminCRMHandler(nil)
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/crm/customers/:id/notes", h.SaveNote)

	body := jsonBody(t, map[string]interface{}{
		"note": "Customer is happy with the product",
		"tags": []string{"happy", "retention"},
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/crm/customers/user-123/notes", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if data["message"] != "note saved" {
		t.Fatalf("expected 'note saved' message, got: %v", data["message"])
	}
}

func TestCRM_SaveNote_InvalidBody(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminCRMHandler(nil)
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/crm/customers/:id/notes", h.SaveNote)

	req := httptest.NewRequest("POST", "/api/v1/admin/crm/customers/user-123/notes",
		bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestCRM_GetNotes(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminCRMHandler(nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/crm/customers/:id/notes", h.GetNotes)

	req := httptest.NewRequest("GET", "/api/v1/admin/crm/customers/user-123/notes", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.CustomerNote
	requireJSON(t, resp.Body, &data)
}

func TestCRM_UpdateTags(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminCRMHandler(nil)
	app.Use(injectTestAdmin)
	app.Put("/api/v1/admin/crm/customers/:id/tags", h.UpdateTags)

	body := jsonBody(t, map[string]interface{}{
		"tags": []string{"vip", "enterprise"},
	})
	req := httptest.NewRequest("PUT", "/api/v1/admin/crm/customers/user-123/tags", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if data["message"] != "tags updated" {
		t.Fatalf("expected 'tags updated' message, got: %v", data["message"])
	}
}

func TestCRM_UpdateTags_InvalidBody(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminCRMHandler(nil)
	app.Use(injectTestAdmin)
	app.Put("/api/v1/admin/crm/customers/:id/tags", h.UpdateTags)

	req := httptest.NewRequest("PUT", "/api/v1/admin/crm/customers/user-123/tags",
		bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(b))
	}
}

// ============================================================================
// 3. AdminServicesHandler
// ============================================================================

func TestServices_ListServices(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminServicesHandler(k8s)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/services", h.ListServices)

	req := httptest.NewRequest("GET", "/api/v1/admin/services", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.ServiceStatus
	requireJSON(t, resp.Body, &data)
	if len(data) == 0 {
		t.Fatal("expected non-empty service list")
	}
}

func TestServices_GetService_KnownService(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminServicesHandler(k8s)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/services/:name", h.GetService)

	req := httptest.NewRequest("GET", "/api/v1/admin/services/traefik", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.ServiceDetail
	requireJSON(t, resp.Body, &data)
	if data.Name != "traefik" {
		t.Fatalf("expected service name 'traefik', got %q", data.Name)
	}
}

func TestServices_GetService_UnknownService(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminServicesHandler(k8s)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/services/:name", h.GetService)

	req := httptest.NewRequest("GET", "/api/v1/admin/services/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestServices_RestartService_KnownService(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminServicesHandler(k8s)
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/services/:name/restart", h.RestartService)

	req := httptest.NewRequest("POST", "/api/v1/admin/services/traefik/restart", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if data["message"] != "restart initiated" {
		t.Fatalf("expected 'restart initiated', got: %v", data["message"])
	}
}

func TestServices_RestartService_UnknownService(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminServicesHandler(k8s)
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/services/:name/restart", h.RestartService)

	req := httptest.NewRequest("POST", "/api/v1/admin/services/nonexistent/restart", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(b))
	}
}

// ============================================================================
// 4. AdminObservabilityHandler
// ============================================================================

func TestObservability_ListDashboards(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminObservabilityHandler("", "", "", "")
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/observability/dashboards", h.ListDashboards)

	req := httptest.NewRequest("GET", "/api/v1/admin/observability/dashboards", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.GrafanaDashboard
	requireJSON(t, resp.Body, &data)
}

func TestObservability_QueryLogs_ValidBody(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminObservabilityHandler("", "", "", "")
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/observability/logs/query", h.QueryLogs)

	body := jsonBody(t, map[string]interface{}{
		"query": "{namespace=\"zenith-platform\"}",
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/observability/logs/query", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// With empty lokiURL, returns empty LogQueryResult (200)
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.LogQueryResult
	requireJSON(t, resp.Body, &data)
}

func TestObservability_QueryLogs_InvalidBody(t *testing.T) {
	app := newTestApp()
	// Use a non-empty lokiURL so the handler proceeds past the nil check and parses the body
	h := handlers.NewAdminObservabilityHandler("", "http://loki:3100", "", "")
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/observability/logs/query", h.QueryLogs)

	req := httptest.NewRequest("POST", "/api/v1/admin/observability/logs/query",
		bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestObservability_QueryLogs_EmptyQuery(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminObservabilityHandler("", "http://loki:3100", "", "")
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/observability/logs/query", h.QueryLogs)

	body := jsonBody(t, map[string]interface{}{
		"query": "",
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/observability/logs/query", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 for empty query, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestObservability_GetLogLabels(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminObservabilityHandler("", "", "", "")
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/observability/logs/labels", h.GetLogLabels)

	req := httptest.NewRequest("GET", "/api/v1/admin/observability/logs/labels", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []string
	requireJSON(t, resp.Body, &data)
}

func TestObservability_ListAlerts(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminObservabilityHandler("", "", "", "")
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/observability/alerts", h.ListAlerts)

	req := httptest.NewRequest("GET", "/api/v1/admin/observability/alerts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.AlertInfo
	requireJSON(t, resp.Body, &data)
}

func TestObservability_GetAlertStats(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminObservabilityHandler("", "", "", "")
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/observability/alerts/stats", h.GetAlertStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/observability/alerts/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.AlertStats
	requireJSON(t, resp.Body, &data)
}

func TestObservability_ListAlertRules(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminObservabilityHandler("", "", "", "")
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/observability/alerts/rules", h.ListAlertRules)

	req := httptest.NewRequest("GET", "/api/v1/admin/observability/alerts/rules", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.AdminAlertRule
	requireJSON(t, resp.Body, &data)
}

func TestObservability_SearchTraces(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminObservabilityHandler("", "", "", "")
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/observability/traces", h.SearchTraces)

	req := httptest.NewRequest("GET", "/api/v1/admin/observability/traces", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.TraceInfo
	requireJSON(t, resp.Body, &data)
}

func TestObservability_GetTrace_NoTempo(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminObservabilityHandler("", "", "", "")
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/observability/traces/:id", h.GetTrace)

	req := httptest.NewRequest("GET", "/api/v1/admin/observability/traces/abc123", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// With empty tempoURL, GetTrace returns 503 Service Unavailable
	if resp.StatusCode != 503 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 503, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestObservability_CreateSilence(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminObservabilityHandler("", "", "", "")
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/observability/alerts/silence", h.CreateSilence)

	req := httptest.NewRequest("POST", "/api/v1/admin/observability/alerts/silence", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if data["message"] != "silence created" {
		t.Fatalf("expected 'silence created', got: %v", data["message"])
	}
}

// ============================================================================
// 5. AdminSecurityHandler
// ============================================================================

func TestSecurity_GetPosture(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminSecurityHandler(nil, k8s, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/security/posture", h.GetPosture)

	req := httptest.NewRequest("GET", "/api/v1/admin/security/posture", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.SecurityPosture
	requireJSON(t, resp.Body, &data)
	if data.OverallScore <= 0 {
		t.Fatalf("expected positive overall score, got %d", data.OverallScore)
	}
}

func TestSecurity_ListPolicies(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminSecurityHandler(nil, k8s, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/security/policies", h.ListPolicies)

	req := httptest.NewRequest("GET", "/api/v1/admin/security/policies", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.PolicyInfo
	requireJSON(t, resp.Body, &data)
}

func TestSecurity_GetPolicyStats(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminSecurityHandler(nil, k8s, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/security/policies/stats", h.GetPolicyStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/security/policies/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.WafStats
	requireJSON(t, resp.Body, &data)
}

func TestSecurity_ListFalcoAlerts(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminSecurityHandler(nil, k8s, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/security/falco/alerts", h.ListFalcoAlerts)

	req := httptest.NewRequest("GET", "/api/v1/admin/security/falco/alerts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.FalcoAlert
	requireJSON(t, resp.Body, &data)
}

func TestSecurity_GetRateLimits(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminSecurityHandler(nil, k8s, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/security/rate-limits", h.GetRateLimits)

	req := httptest.NewRequest("GET", "/api/v1/admin/security/rate-limits", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if _, ok := data["global"]; !ok {
		t.Fatal("expected 'global' key in rate limits response")
	}
}

func TestSecurity_ListImages(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminSecurityHandler(nil, k8s, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/security/images", h.ListImages)

	req := httptest.NewRequest("GET", "/api/v1/admin/security/images", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.ImageScanResult
	requireJSON(t, resp.Body, &data)
	if len(data) == 0 {
		t.Fatal("expected non-empty image list")
	}
}

func TestSecurity_GetImageStats(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminSecurityHandler(nil, k8s, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/security/images/stats", h.GetImageStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/security/images/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.ImageScanStats
	requireJSON(t, resp.Body, &data)
	if data.TotalImages == 0 {
		t.Fatal("expected non-zero TotalImages")
	}
}

func TestSecurity_TriggerImageScan(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminSecurityHandler(nil, k8s, nil)
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/security/images/:name/scan", h.TriggerImageScan)

	req := httptest.NewRequest("POST", "/api/v1/admin/security/images/zenith-api/scan", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if data["message"] != "scan triggered" {
		t.Fatalf("expected 'scan triggered', got: %v", data["message"])
	}
	if data["image"] != "zenith-api" {
		t.Fatalf("expected image 'zenith-api', got: %v", data["image"])
	}
}

func TestSecurity_ListSessions(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminSecurityHandler(nil, k8s, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/security/sessions", h.ListSessions)

	req := httptest.NewRequest("GET", "/api/v1/admin/security/sessions", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.AdminSession
	requireJSON(t, resp.Body, &data)
}

func TestSecurity_TerminateSession(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminSecurityHandler(nil, k8s, nil)
	app.Use(injectTestAdmin)
	app.Delete("/api/v1/admin/security/sessions/:id", h.TerminateSession)

	req := httptest.NewRequest("DELETE", "/api/v1/admin/security/sessions/session-abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if data["message"] != "session terminated" {
		t.Fatalf("expected 'session terminated', got: %v", data["message"])
	}
}

// ============================================================================
// 6. AdminPlatformOpsHandler
// ============================================================================

// --- Backups ---

func TestPlatformOps_GetBackups(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/backups", h.GetBackups)

	req := httptest.NewRequest("GET", "/api/v1/admin/backups", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.AdminBackupOverview
	requireJSON(t, resp.Body, &data)
}

func TestPlatformOps_GetBackupStats(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/backups/stats", h.GetBackupStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/backups/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.BackupStats
	requireJSON(t, resp.Body, &data)
	if data.CNPGClusters != 2 {
		t.Fatalf("expected 2 CNPG clusters, got %d", data.CNPGClusters)
	}
}

func TestPlatformOps_ListVeleroSchedules(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/backups/velero", h.ListVeleroSchedules)

	req := httptest.NewRequest("GET", "/api/v1/admin/backups/velero", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.VeleroSchedule
	requireJSON(t, resp.Body, &data)
}

func TestPlatformOps_ListCNPGBackups(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/backups/cnpg", h.ListCNPGBackups)

	req := httptest.NewRequest("GET", "/api/v1/admin/backups/cnpg", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.CNPGBackupStatus
	requireJSON(t, resp.Body, &data)
	if len(data) != 2 {
		t.Fatalf("expected 2 CNPG backup entries, got %d", len(data))
	}
}

func TestPlatformOps_TriggerBackup(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/backups/trigger", h.TriggerBackup)

	req := httptest.NewRequest("POST", "/api/v1/admin/backups/trigger", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if data["message"] != "backup triggered" {
		t.Fatalf("expected 'backup triggered', got: %v", data["message"])
	}
}

// --- GitOps ---

func TestPlatformOps_ListArgoApps(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/gitops/apps", h.ListArgoApps)

	req := httptest.NewRequest("GET", "/api/v1/admin/gitops/apps", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.ArgoApp
	requireJSON(t, resp.Body, &data)
}

func TestPlatformOps_GetGitOpsStats(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/gitops/stats", h.GetGitOpsStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/gitops/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.GitOpsStats
	requireJSON(t, resp.Body, &data)
}

func TestPlatformOps_SyncArgoApp(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/gitops/apps/:name/sync", h.SyncArgoApp)

	req := httptest.NewRequest("POST", "/api/v1/admin/gitops/apps/zenith-platform/sync", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if data["message"] != "sync triggered" {
		t.Fatalf("expected 'sync triggered', got: %v", data["message"])
	}
	if data["app"] != "zenith-platform" {
		t.Fatalf("expected app 'zenith-platform', got: %v", data["app"])
	}
}

func TestPlatformOps_GetArgoAppHistory(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/gitops/apps/:name/history", h.GetArgoAppHistory)

	req := httptest.NewRequest("GET", "/api/v1/admin/gitops/apps/zenith-platform/history", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.ArgoDeployment
	requireJSON(t, resp.Body, &data)
}

// --- Registry ---

func TestPlatformOps_ListRegistryProjects(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/registry/projects", h.ListRegistryProjects)

	req := httptest.NewRequest("GET", "/api/v1/admin/registry/projects", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.RegistryProject
	requireJSON(t, resp.Body, &data)
	if len(data) != 2 {
		t.Fatalf("expected 2 registry projects, got %d", len(data))
	}
}

func TestPlatformOps_GetRegistryStats(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/registry/stats", h.GetRegistryStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/registry/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.RegistryStats
	requireJSON(t, resp.Body, &data)
	if data.TotalProjects != 2 {
		t.Fatalf("expected 2 total projects, got %d", data.TotalProjects)
	}
}

func TestPlatformOps_ListRegistryRepos(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/registry/projects/:name/repos", h.ListRegistryRepos)

	req := httptest.NewRequest("GET", "/api/v1/admin/registry/projects/zenith/repos", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.RegistryRepo
	requireJSON(t, resp.Body, &data)
}

// --- Databases ---

func TestPlatformOps_ListDatabaseClusters(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/databases", h.ListDatabaseClusters)

	req := httptest.NewRequest("GET", "/api/v1/admin/databases", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.AdminDatabaseCluster
	requireJSON(t, resp.Body, &data)
}

func TestPlatformOps_GetDatabaseStats(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/databases/stats", h.GetDatabaseStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/databases/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.DatabaseStats
	requireJSON(t, resp.Body, &data)
}

func TestPlatformOps_GetDatabaseCluster_NotFound(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/databases/:name", h.GetDatabaseCluster)

	req := httptest.NewRequest("GET", "/api/v1/admin/databases/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, string(b))
	}
}

// --- Storage ---

func TestPlatformOps_ListS3Buckets(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/storage/s3", h.ListS3Buckets)

	req := httptest.NewRequest("GET", "/api/v1/admin/storage/s3", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.AdminS3Bucket
	requireJSON(t, resp.Body, &data)
	if len(data) != 2 {
		t.Fatalf("expected 2 S3 buckets, got %d", len(data))
	}
}

func TestPlatformOps_ListVolumes(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/storage/volumes", h.ListVolumes)

	req := httptest.NewRequest("GET", "/api/v1/admin/storage/volumes", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.AdminVolume
	requireJSON(t, resp.Body, &data)
}

func TestPlatformOps_GetStorageStats(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/storage/stats", h.GetStorageStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/storage/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.StorageStats
	requireJSON(t, resp.Body, &data)
	if data.TotalBuckets != 2 {
		t.Fatalf("expected 2 total buckets, got %d", data.TotalBuckets)
	}
}

// --- Networking ---

func TestPlatformOps_ListDNSRecords(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/networking/dns", h.ListDNSRecords)

	req := httptest.NewRequest("GET", "/api/v1/admin/networking/dns", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.AdminDNSRecord
	requireJSON(t, resp.Body, &data)
}

func TestPlatformOps_ListRoutes(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/networking/routes", h.ListRoutes)

	req := httptest.NewRequest("GET", "/api/v1/admin/networking/routes", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.AdminRoute
	requireJSON(t, resp.Body, &data)
}

func TestPlatformOps_ListCertificates(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/networking/certificates", h.ListCertificates)

	req := httptest.NewRequest("GET", "/api/v1/admin/networking/certificates", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.AdminCertificate
	requireJSON(t, resp.Body, &data)
}

// --- Quality ---

func TestPlatformOps_GetQualityMetrics(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/quality/metrics", h.GetQualityMetrics)

	req := httptest.NewRequest("GET", "/api/v1/admin/quality/metrics", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data entities.QualityMetrics
	requireJSON(t, resp.Body, &data)
}

func TestPlatformOps_GetQualityTickets(t *testing.T) {
	app := newTestApp()
	k8s := k8sclient.NewMemoryClient()
	h := handlers.NewAdminPlatformOpsHandler(nil, k8s, nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/quality/tickets", h.GetQualityTickets)

	req := httptest.NewRequest("GET", "/api/v1/admin/quality/tickets", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.QualityTicket
	requireJSON(t, resp.Body, &data)
}

// ============================================================================
// 7. AdminProxyHandler
// ============================================================================

func TestProxy_UnknownService(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminProxyHandler(map[string]string{})
	app.Use(injectTestAdmin)
	app.All("/api/v1/admin/proxy/:service/*", h.Proxy)

	req := httptest.NewRequest("GET", "/api/v1/admin/proxy/grafana/api/dashboards", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404 for unknown proxy target, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestProxy_UnknownService_POST(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminProxyHandler(map[string]string{})
	app.Use(injectTestAdmin)
	app.All("/api/v1/admin/proxy/:service/*", h.Proxy)

	req := httptest.NewRequest("POST", "/api/v1/admin/proxy/loki/api/v1/query", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404 for unknown proxy target, got %d: %s", resp.StatusCode, string(b))
	}
}

// ============================================================================
// 8. AdminRBACHandler
// ============================================================================

func TestRBAC_ListAdminUsers(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminRBACHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Get("/api/v1/admin/admin-users", h.ListAdminUsers)

	req := httptest.NewRequest("GET", "/api/v1/admin/admin-users", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data []entities.AdminRole
	requireJSON(t, resp.Body, &data)
}

func TestRBAC_InviteAdminUser_InvalidBody(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminRBACHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/admin-users", h.InviteAdminUser)

	req := httptest.NewRequest("POST", "/api/v1/admin/admin-users",
		bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestRBAC_InviteAdminUser_MissingEmail(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminRBACHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/admin-users", h.InviteAdminUser)

	body := jsonBody(t, map[string]interface{}{
		"adminRole": "viewer",
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/admin-users", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 for missing email, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestRBAC_InviteAdminUser_InvalidRole(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminRBACHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Post("/api/v1/admin/admin-users", h.InviteAdminUser)

	body := jsonBody(t, map[string]interface{}{
		"email":     "test@zenith.dev",
		"adminRole": "superadmin",
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/admin-users", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 for invalid role, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestRBAC_UpdateAdminRole_InvalidBody(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminRBACHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Put("/api/v1/admin/admin-users/:id/role", h.UpdateAdminRole)

	req := httptest.NewRequest("PUT", "/api/v1/admin/admin-users/role-123/role",
		bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestRBAC_UpdateAdminRole_InvalidRole(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminRBACHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Put("/api/v1/admin/admin-users/:id/role", h.UpdateAdminRole)

	body := jsonBody(t, map[string]interface{}{
		"adminRole": "megaadmin",
	})
	req := httptest.NewRequest("PUT", "/api/v1/admin/admin-users/role-123/role", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 for invalid role, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestRBAC_UpdateAdminRole_ValidRole(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminRBACHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Put("/api/v1/admin/admin-users/:id/role", h.UpdateAdminRole)

	body := jsonBody(t, map[string]interface{}{
		"adminRole": "admin",
	})
	req := httptest.NewRequest("PUT", "/api/v1/admin/admin-users/role-123/role", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if data["message"] != "role updated" {
		t.Fatalf("expected 'role updated', got: %v", data["message"])
	}
	if data["adminRole"] != "admin" {
		t.Fatalf("expected adminRole 'admin', got: %v", data["adminRole"])
	}
}

func TestRBAC_RemoveAdminUser(t *testing.T) {
	app := newTestApp()
	h := handlers.NewAdminRBACHandler(nil, nil)
	app.Use(injectTestAdmin)
	app.Delete("/api/v1/admin/admin-users/:id", h.RemoveAdminUser)

	req := httptest.NewRequest("DELETE", "/api/v1/admin/admin-users/role-123", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	var data map[string]interface{}
	requireJSON(t, resp.Body, &data)
	if data["message"] != "admin role removed" {
		t.Fatalf("expected 'admin role removed', got: %v", data["message"])
	}
}
