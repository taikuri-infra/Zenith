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

func setupNetworkPolicyTest() (*fiber.App, *handlers.NetworkPolicyHandler, *memory.MemoryAppRepository, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewNetworkPolicyHandler(appRepo, planRepo)
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
	resp, _ := app.Test(req)
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
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestNetworkPolicyCreateRule(t *testing.T) {
	app, handler, appRepo, planRepo := setupNetworkPolicyTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createNetPolTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/network-policies", injectUserID("user-1"), handler.CreateRule)

	body := `{"name":"Allow DB","direction":"egress","action":"allow","priority":10}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/network-policies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
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

	resp, _ := app.Test(req)
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

	resp, _ := app.Test(req)
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

	resp, _ := app.Test(req)
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

	createBody := `{"name":"Original","direction":"ingress","action":"deny","priority":1}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/network-policies", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	ruleID := created["id"].(string)

	updateBody := `{"name":"Updated"}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/apps/"+testApp.ID+"/network-policies/"+ruleID, bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(updateReq)
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

	resp, _ := app.Test(req)
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

	createBody := `{"name":"ToDelete","direction":"ingress","action":"allow"}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/network-policies", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	ruleID := created["id"].(string)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+testApp.ID+"/network-policies/"+ruleID, nil)
	resp, _ := app.Test(req)
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
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}
