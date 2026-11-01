package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupNetworkPolicyTest() (*fiber.App, *handlers.NetworkPolicyHandler, *memory.MemoryAppRepository, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewNetworkPolicyHandler(appRepo, planRepo, memory.NewFirewallRuleRepository())
	return app, handler, appRepo, planRepo
}

func createNetPolTestApp(appRepo *memory.MemoryAppRepository, userID string) *entities.App {
	a, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		Name:         "netpol-app",
		UserID:       userID,
		ProjectID:    "proj-1",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "registry.example.com/test:latest",
	})
	return a
}

func TestNetworkPolicyListRulesBusinessPlan(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Get("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.ListRules)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/network-policies", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Rules []interface{} `json:"rules"`
		Total int           `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0 rules, got %d", result.Total)
	}
}

func TestNetworkPolicyListRulesFreePlanForbidden(t *testing.T) {
	app, handler, appRepo, _ := setupNetworkPolicyTest()
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Get("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.ListRules)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/network-policies", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestNetworkPolicyCreateRule(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)

	body := `{"name":"Allow DB","direction":"egress","action":"allow","priority":10,"config":{"namespaces":["other"]}}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/network-policies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["name"] != "Allow DB" {
		t.Errorf("Expected name 'Allow DB', got '%v'", result["name"])
	}
	if result["direction"] != "egress" {
		t.Errorf("Expected direction 'egress', got '%v'", result["direction"])
	}
}

func TestNetworkPolicyCreateRuleNoName(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)

	body := `{"direction":"ingress","action":"allow"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/network-policies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestNetworkPolicyCreateRuleInvalidDirection(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)

	body := `{"name":"Test","direction":"invalid","action":"allow"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/network-policies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestNetworkPolicyCreateRuleInvalidAction(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)

	body := `{"name":"Test","direction":"ingress","action":"maybe"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/network-policies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestNetworkPolicyUpdateRule(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)
	app.Put("/api/v1/apps/:appId/network-policies/:ruleId", injectUserID("user-1"), handler.UpdateRule)

	createBody := `{"name":"Original","direction":"ingress","action":"allow","priority":1,"config":{"namespaces":["other"]}}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/network-policies", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq, -1)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	ruleID := created["id"].(string)

	updateBody := `{"name":"Updated"}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/apps/"+testApp.ID+"/network-policies/"+ruleID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(updateReq, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["name"] != "Updated" {
		t.Errorf("Expected name 'Updated', got '%v'", result["name"])
	}
}

func TestNetworkPolicyUpdateRuleNotFound(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Put("/api/v1/apps/:appId/network-policies/:ruleId", injectUserID("user-1"), handler.UpdateRule)

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/apps/"+testApp.ID+"/network-policies/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestNetworkPolicyDeleteRule(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)
	app.Delete("/api/v1/apps/:appId/network-policies/:ruleId", injectUserID("user-1"), handler.DeleteRule)

	createBody := `{"name":"ToDelete","direction":"ingress","action":"allow","config":{"namespaces":["other"]}}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/network-policies", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq, -1)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	ruleID := created["id"].(string)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+testApp.ID+"/network-policies/"+ruleID, nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "rule deleted" {
		t.Errorf("Expected 'rule deleted', got '%v'", result["message"])
	}
}

func TestNetworkPolicyDeleteRuleNotFound(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Delete("/api/v1/apps/:appId/network-policies/:ruleId", injectUserID("user-1"), handler.DeleteRule)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+testApp.ID+"/network-policies/nonexistent", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func postRule(t *testing.T, app *fiber.App, appID, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/network-policies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	return resp
}

func errorMessage(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func TestFirewallCreateInvalidCIDR400(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	a := createNetPolTestApp(appRepo, "user-1")
	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)
	app.Get("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.ListRules)

	resp := postRule(t, app, a.ID, `{"name":"x","direction":"ingress","action":"allow","config":{"cidrs":["10.0.0.0/33"]}}`)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	list, _ := app.Test(httptest.NewRequest("GET", "/api/v1/apps/"+a.ID+"/network-policies", nil), -1)
	var result struct{ Total int }
	json.NewDecoder(list.Body).Decode(&result)
	if result.Total != 0 {
		t.Fatalf("invalid rule was stored")
	}
}

func TestFirewallCreateDenyWithoutCilium400(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	a := createNetPolTestApp(appRepo, "user-1")
	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)

	resp := postRule(t, app, a.ID, `{"name":"x","direction":"ingress","action":"deny","config":{"cidrs":["1.1.1.1/32"]}}`)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if msg := errorMessage(resp); !strings.Contains(msg, "deny rules and FQDN egress need Cilium") {
		t.Fatalf("message = %s", msg)
	}
}

func TestFirewallOtherUsersApp404(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-2", entities.PlanBusiness)
	a := createNetPolTestApp(appRepo, "user-1")
	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-2"), handler.CreateRule)

	resp := postRule(t, app, a.ID, `{"name":"x","direction":"ingress","action":"allow","config":{"namespaces":["other"]}}`)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestFirewallRuleFromOtherApp404(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	a := createNetPolTestApp(appRepo, "user-1")
	b, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{Name: "netpol-app-2", UserID: "user-1", ProjectID: "proj-1",
		DeploySource: entities.DeploySourceImage, ImageURL: "registry.example.com/test:latest"})
	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)
	app.Put("/api/v1/apps/:appId/network-policies/:ruleId", injectUserID("user-1"), handler.UpdateRule)
	app.Delete("/api/v1/apps/:appId/network-policies/:ruleId", injectUserID("user-1"), handler.DeleteRule)

	var created map[string]any
	json.NewDecoder(postRule(t, app, a.ID, `{"name":"x","direction":"ingress","action":"allow","config":{"namespaces":["other"]}}`).Body).Decode(&created)
	id := created["id"].(string)

	put := httptest.NewRequest("PUT", "/api/v1/apps/"+b.ID+"/network-policies/"+id, bytes.NewBufferString(`{"name":"y"}`))
	put.Header.Set("Content-Type", "application/json")
	if resp, _ := app.Test(put, -1); resp.StatusCode != 404 {
		t.Fatalf("update via other app: expected 404, got %d", resp.StatusCode)
	}
	del := httptest.NewRequest("DELETE", "/api/v1/apps/"+b.ID+"/network-policies/"+id, nil)
	if resp, _ := app.Test(del, -1); resp.StatusCode != 404 {
		t.Fatalf("delete via other app: expected 404, got %d", resp.StatusCode)
	}
}

func TestFirewallCRUDRoundTrip(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	a := createNetPolTestApp(appRepo, "user-1")
	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)
	app.Get("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.ListRules)
	app.Delete("/api/v1/apps/:appId/network-policies/:ruleId", injectUserID("user-1"), handler.DeleteRule)

	var created map[string]any
	resp := postRule(t, app, a.ID, `{"name":"web","direction":"ingress","action":"allow","config":{"cidrs":["10.0.0.0/8"],"ports":[{"protocol":"TCP","port":443}]}}`)
	if resp.StatusCode != 201 {
		t.Fatalf("create: %d", resp.StatusCode)
	}
	json.NewDecoder(resp.Body).Decode(&created)
	list, _ := app.Test(httptest.NewRequest("GET", "/api/v1/apps/"+a.ID+"/network-policies", nil), -1)
	var result struct{ Total int }
	json.NewDecoder(list.Body).Decode(&result)
	if result.Total != 1 {
		t.Fatalf("total = %d", result.Total)
	}
	del, _ := app.Test(httptest.NewRequest("DELETE", "/api/v1/apps/"+a.ID+"/network-policies/"+created["id"].(string), nil), -1)
	if del.StatusCode != 200 {
		t.Fatalf("delete: %d", del.StatusCode)
	}
}

func TestFirewallNoPlanGateWhenSelfHosted(t *testing.T) {
	app, handler, appRepo, _ := setupNetworkPolicyTest()
	handler.SetSelfHosted(true)
	a := createNetPolTestApp(appRepo, "user-1")
	app.Get("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.ListRules)

	resp, _ := app.Test(httptest.NewRequest("GET", "/api/v1/apps/"+a.ID+"/network-policies", nil), -1)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 without a plan in self-host, got %d", resp.StatusCode)
	}
}

func TestFirewallRuleWithoutPeer400(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	a := createNetPolTestApp(appRepo, "user-1")
	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)
	if resp := postRule(t, app, a.ID, `{"name":"open","direction":"ingress","action":"allow"}`); resp.StatusCode != 400 {
		t.Fatalf("a rule with no peer would open the app to everything; expected 400, got %d", resp.StatusCode)
	}
}
