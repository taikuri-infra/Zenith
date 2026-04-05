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

func setupWAFTest() (*fiber.App, *handlers.WAFHandler, *memory.MemoryAppRepository, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewWAFHandler(appRepo, planRepo)
	return app, handler, appRepo, planRepo
}

func createWAFTestApp(appRepo *memory.MemoryAppRepository, userID string) *entities.App {
	a, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{
		Name:         "waf-app",
		UserID:       userID,
		ProjectID:    "proj-1",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "registry.example.com/test:latest",
	})
	return a
}

func TestWAFListRulesBusinessPlan(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createWAFTestApp(appRepo, "user-1")

	app.Get("/api/v1/apps/:appId/waf/rules", injectUserID("user-1"), handler.ListRules)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/waf/rules", nil)
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

func TestWAFListRulesFreePlanForbidden(t *testing.T) {
	app, handler, appRepo, _ := setupWAFTest()
	testApp := createWAFTestApp(appRepo, "user-1")
	// No plan set = free plan

	app.Get("/api/v1/apps/:appId/waf/rules", injectUserID("user-1"), handler.ListRules)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/waf/rules", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestWAFListRulesProPlanForbidden(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)
	testApp := createWAFTestApp(appRepo, "user-1")

	app.Get("/api/v1/apps/:appId/waf/rules", injectUserID("user-1"), handler.ListRules)

	req := httptest.NewRequest("GET", "/api/v1/apps/"+testApp.ID+"/waf/rules", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestWAFCreateRule(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createWAFTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/waf/rules", injectUserID("user-1"), handler.CreateRule)

	body := `{"name":"Block bad IPs","type":"ip_block","priority":10}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/waf/rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["name"] != "Block bad IPs" {
		t.Errorf("Expected name 'Block bad IPs', got '%v'", result["name"])
	}
	if result["enabled"] != true {
		t.Errorf("Expected enabled true, got %v", result["enabled"])
	}
}

func TestWAFCreateRuleNoName(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createWAFTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/waf/rules", injectUserID("user-1"), handler.CreateRule)

	body := `{"type":"ip_block"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/waf/rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestWAFCreateRuleInvalidType(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createWAFTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/waf/rules", injectUserID("user-1"), handler.CreateRule)

	body := `{"name":"Test","type":"invalid_type"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/waf/rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestWAFCreateRuleNotYourApp(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-2", entities.PlanBusiness)
	testApp := createWAFTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/waf/rules", injectUserID("user-2"), handler.CreateRule)

	body := `{"name":"Test","type":"ip_block"}`
	req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/waf/rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestWAFUpdateRule(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createWAFTestApp(appRepo, "user-1")

	// Create a rule first via the handler
	app.Post("/api/v1/apps/:appId/waf/rules", injectUserID("user-1"), handler.CreateRule)
	app.Put("/api/v1/apps/:appId/waf/rules/:ruleId", injectUserID("user-1"), handler.UpdateRule)

	createBody := `{"name":"Original","type":"rate_limit","priority":1}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/waf/rules", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	ruleID := created["id"].(string)

	// Update the rule
	updateBody := `{"name":"Updated"}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/apps/"+testApp.ID+"/waf/rules/"+ruleID, bytes.NewBufferString(updateBody))
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

func TestWAFUpdateRuleNotFound(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createWAFTestApp(appRepo, "user-1")

	app.Put("/api/v1/apps/:appId/waf/rules/:ruleId", injectUserID("user-1"), handler.UpdateRule)

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/apps/"+testApp.ID+"/waf/rules/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestWAFDeleteRule(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createWAFTestApp(appRepo, "user-1")

	// Create a rule first
	app.Post("/api/v1/apps/:appId/waf/rules", injectUserID("user-1"), handler.CreateRule)
	app.Delete("/api/v1/apps/:appId/waf/rules/:ruleId", injectUserID("user-1"), handler.DeleteRule)

	createBody := `{"name":"ToDelete","type":"ip_allow","priority":1}`
	createReq := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/waf/rules", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := app.Test(createReq)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	ruleID := created["id"].(string)

	// Delete the rule
	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+testApp.ID+"/waf/rules/"+ruleID, nil)
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

func TestWAFDeleteRuleNotFound(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanBusiness)
	testApp := createWAFTestApp(appRepo, "user-1")

	app.Delete("/api/v1/apps/:appId/waf/rules/:ruleId", injectUserID("user-1"), handler.DeleteRule)

	req := httptest.NewRequest("DELETE", "/api/v1/apps/"+testApp.ID+"/waf/rules/nonexistent", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestWAFCreateRuleAllTypes(t *testing.T) {
	app, handler, appRepo, planRepo := setupWAFTest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanEnterprise)
	testApp := createWAFTestApp(appRepo, "user-1")

	app.Post("/api/v1/apps/:appId/waf/rules", injectUserID("user-1"), handler.CreateRule)

	types := []string{"rate_limit", "ip_block", "ip_allow", "body_limit", "geo_block", "header_rule"}
	for _, ruleType := range types {
		body := `{"name":"Rule ` + ruleType + `","type":"` + ruleType + `"}`
		req := httptest.NewRequest("POST", "/api/v1/apps/"+testApp.ID+"/waf/rules", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		if resp.StatusCode != 201 {
			t.Errorf("Expected 201 for type %s, got %d", ruleType, resp.StatusCode)
		}
	}
}
