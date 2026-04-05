package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

func setupGatewayTest() (*fiber.App, *handlers.GatewayHandler, *memory.MemoryGatewayRepository, *memory.MemoryUserPlanRepository, *memory.MemoryAppRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	gwRepo := memory.NewMemoryGatewayRepository()
	appRepo := memory.NewMemoryAppRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	k8s := k8sclient.NewMemoryClient()
	gwSvc := services.NewGatewayService(gwRepo, appRepo, planRepo, k8s, "gw.test.com", "zenith-apps")
	handler := handlers.NewGatewayHandler(gwSvc, gwRepo, nil, planRepo)
	return app, handler, gwRepo, planRepo, appRepo
}

// --- Gateway CRUD ---

func TestGatewayCreate(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()
	fiberApp.Post("/api/v1/gateways", injectUserID("user-1"), handler.CreateGateway)

	body := `{"name":"my-api-gateway"}`
	req := httptest.NewRequest("POST", "/api/v1/gateways", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["name"] != "my-api-gateway" {
		t.Errorf("Expected name 'my-api-gateway', got '%v'", result["name"])
	}
}

func TestGatewayCreateNoName(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()
	fiberApp.Post("/api/v1/gateways", injectUserID("user-1"), handler.CreateGateway)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/gateways", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestGatewayCreateInvalidBody(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()
	fiberApp.Post("/api/v1/gateways", injectUserID("user-1"), handler.CreateGateway)

	req := httptest.NewRequest("POST", "/api/v1/gateways", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestGatewayList(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()
	fiberApp.Post("/api/v1/gateways", injectUserID("user-1"), handler.CreateGateway)
	fiberApp.Get("/api/v1/gateways", injectUserID("user-1"), handler.ListGateways)

	// Create 2 gateways
	for _, name := range []string{"gw-1", "gw-2"} {
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest("POST", "/api/v1/gateways", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fiberApp.Test(req)
	}

	listReq := httptest.NewRequest("GET", "/api/v1/gateways", nil)
	listResp, _ := fiberApp.Test(listReq)

	if listResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", listResp.StatusCode)
	}

	var result []map[string]interface{}
	json.NewDecoder(listResp.Body).Decode(&result)

	if len(result) != 2 {
		t.Errorf("Expected 2 gateways, got %d", len(result))
	}
}

func TestGatewayListEmpty(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()
	fiberApp.Get("/api/v1/gateways", injectUserID("user-1"), handler.ListGateways)

	req := httptest.NewRequest("GET", "/api/v1/gateways", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result) != 0 {
		t.Errorf("Expected 0 gateways, got %d", len(result))
	}
}

func TestGatewayGet(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()
	fiberApp.Post("/api/v1/gateways", injectUserID("user-1"), handler.CreateGateway)
	fiberApp.Get("/api/v1/gateways/:gwId", injectUserID("user-1"), handler.GetGateway)

	// Create a gateway
	body := `{"name":"my-gw"}`
	createReq := httptest.NewRequest("POST", "/api/v1/gateways", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	gwID := created["id"].(string)

	// Get the gateway
	getReq := httptest.NewRequest("GET", "/api/v1/gateways/"+gwID, nil)
	getResp, _ := fiberApp.Test(getReq)

	if getResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", getResp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(getResp.Body).Decode(&result)

	// Result should contain gateway, routes, groups
	if result["gateway"] == nil {
		t.Error("Expected 'gateway' in response")
	}
	if result["routes"] == nil {
		t.Error("Expected 'routes' in response")
	}
	if result["groups"] == nil {
		t.Error("Expected 'groups' in response")
	}
}

func TestGatewayGetNotFound(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()
	fiberApp.Get("/api/v1/gateways/:gwId", injectUserID("user-1"), handler.GetGateway)

	req := httptest.NewRequest("GET", "/api/v1/gateways/nonexistent", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestGatewayGetForbidden(t *testing.T) {
	fiberApp, handler, gwRepo, _, _ := setupGatewayTest()

	// Create gateway for user-1
	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "test-gw", "test-gw")

	fiberApp.Get("/api/v1/gateways/:gwId", injectUserID("user-2"), handler.GetGateway)

	req := httptest.NewRequest("GET", "/api/v1/gateways/"+gw.ID, nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestGatewayUpdate(t *testing.T) {
	fiberApp, handler, gwRepo, _, _ := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "old-name", "old-name")

	fiberApp.Put("/api/v1/gateways/:gwId", injectUserID("user-1"), handler.UpdateGateway)

	body := `{"name":"new-name"}`
	req := httptest.NewRequest("PUT", "/api/v1/gateways/"+gw.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["name"] != "new-name" {
		t.Errorf("Expected name 'new-name', got '%v'", result["name"])
	}
}

func TestGatewayUpdateNoName(t *testing.T) {
	fiberApp, handler, gwRepo, _, _ := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw", "my-gw")

	fiberApp.Put("/api/v1/gateways/:gwId", injectUserID("user-1"), handler.UpdateGateway)

	body := `{}`
	req := httptest.NewRequest("PUT", "/api/v1/gateways/"+gw.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestGatewayUpdateNotFound(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()
	fiberApp.Put("/api/v1/gateways/:gwId", injectUserID("user-1"), handler.UpdateGateway)

	body := `{"name":"new-name"}`
	req := httptest.NewRequest("PUT", "/api/v1/gateways/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestGatewayUpdateForbidden(t *testing.T) {
	fiberApp, handler, gwRepo, _, _ := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw", "my-gw")

	fiberApp.Put("/api/v1/gateways/:gwId", injectUserID("user-2"), handler.UpdateGateway)

	body := `{"name":"new-name"}`
	req := httptest.NewRequest("PUT", "/api/v1/gateways/"+gw.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestGatewayDelete(t *testing.T) {
	fiberApp, handler, gwRepo, _, _ := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw", "my-gw")

	fiberApp.Delete("/api/v1/gateways/:gwId", injectUserID("user-1"), handler.DeleteGateway)

	req := httptest.NewRequest("DELETE", "/api/v1/gateways/"+gw.ID, nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["message"] != "gateway deleted" {
		t.Errorf("Expected message 'gateway deleted', got '%v'", result["message"])
	}
}

func TestGatewayDeleteNotFound(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()
	fiberApp.Delete("/api/v1/gateways/:gwId", injectUserID("user-1"), handler.DeleteGateway)

	req := httptest.NewRequest("DELETE", "/api/v1/gateways/nonexistent", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestGatewayDeleteForbidden(t *testing.T) {
	fiberApp, handler, gwRepo, _, _ := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw", "my-gw")

	fiberApp.Delete("/api/v1/gateways/:gwId", injectUserID("user-2"), handler.DeleteGateway)

	req := httptest.NewRequest("DELETE", "/api/v1/gateways/"+gw.ID, nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

// --- Route Tests ---

func TestGatewayCreateRoute(t *testing.T) {
	fiberApp, handler, gwRepo, _, appRepo := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw", "my-gw")
	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{UserID: "user-1", Name: "test-app", RepoURL: "https://github.com/user/repo"})

	fiberApp.Post("/api/v1/gateways/:gwId/routes", injectUserID("user-1"), handler.CreateRoute)

	body := `{"name":"users","path":"/api/users","methods":["GET","POST"],"app_id":"` + testApp.ID + `"}`
	req := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("Expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["name"] != "users" {
		t.Errorf("Expected name 'users', got '%v'", result["name"])
	}
	if result["path"] != "/api/users" {
		t.Errorf("Expected path '/api/users', got '%v'", result["path"])
	}
}

func TestGatewayCreateRouteNoNameOrPath(t *testing.T) {
	fiberApp, handler, gwRepo, _, appRepo := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw", "my-gw")
	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{UserID: "user-1", Name: "test-app", RepoURL: "https://github.com/user/repo"})

	fiberApp.Post("/api/v1/gateways/:gwId/routes", injectUserID("user-1"), handler.CreateRoute)

	body := `{"app_id":"` + testApp.ID + `"}`
	req := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestGatewayCreateRouteNoAppID(t *testing.T) {
	fiberApp, handler, gwRepo, _, _ := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw-noapp", "my-gw-noapp")

	fiberApp.Post("/api/v1/gateways/:gwId/routes", injectUserID("user-1"), handler.CreateRoute)

	body := `{"name":"test","path":"/test"}`
	req := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for missing app_id, got %d", resp.StatusCode)
	}
}

func TestGatewayCreateRouteInvalidPath(t *testing.T) {
	fiberApp, handler, gwRepo, _, appRepo := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw-ip", "my-gw-ip")
	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{UserID: "user-1", Name: "test-app-ip", RepoURL: "https://github.com/user/repo"})

	fiberApp.Post("/api/v1/gateways/:gwId/routes", injectUserID("user-1"), handler.CreateRoute)

	body := `{"name":"test","path":"no-leading-slash","app_id":"` + testApp.ID + `"}`
	req := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for invalid path, got %d", resp.StatusCode)
	}
}

func TestGatewayCreateRouteGatewayNotFound(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()

	fiberApp.Post("/api/v1/gateways/:gwId/routes", injectUserID("user-1"), handler.CreateRoute)

	body := `{"name":"test","path":"/test","app_id":"some-app-id"}`
	req := httptest.NewRequest("POST", "/api/v1/gateways/nonexistent/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestGatewayListRoutes(t *testing.T) {
	fiberApp, handler, gwRepo, _, appRepo := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw-lr", "my-gw-lr")
	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{UserID: "user-1", Name: "test-app-lr", RepoURL: "https://github.com/user/repo"})

	fiberApp.Post("/api/v1/gateways/:gwId/routes", injectUserID("user-1"), handler.CreateRoute)
	fiberApp.Get("/api/v1/gateways/:gwId/routes", injectUserID("user-1"), handler.ListRoutes)

	// Create a route
	body := `{"name":"users","path":"/api/users","app_id":"` + testApp.ID + `"}`
	createReq := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/routes", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	fiberApp.Test(createReq)

	// List routes
	listReq := httptest.NewRequest("GET", "/api/v1/gateways/"+gw.ID+"/routes", nil)
	listResp, _ := fiberApp.Test(listReq)

	if listResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", listResp.StatusCode)
	}

	var result []map[string]interface{}
	json.NewDecoder(listResp.Body).Decode(&result)

	if len(result) != 1 {
		t.Errorf("Expected 1 route, got %d", len(result))
	}
}

func TestGatewayDeleteRoute(t *testing.T) {
	fiberApp, handler, gwRepo, _, appRepo := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw-dr", "my-gw-dr")
	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{UserID: "user-1", Name: "test-app-dr", RepoURL: "https://github.com/user/repo"})

	fiberApp.Post("/api/v1/gateways/:gwId/routes", injectUserID("user-1"), handler.CreateRoute)
	fiberApp.Delete("/api/v1/gateways/:gwId/routes/:routeId", injectUserID("user-1"), handler.DeleteRoute)

	// Create a route
	body := `{"name":"test","path":"/test","app_id":"` + testApp.ID + `"}`
	createReq := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/routes", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq)

	if createResp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(createResp.Body).Decode(&errBody)
		t.Fatalf("Expected 201 for route creation, got %d: %v", createResp.StatusCode, errBody)
	}

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	routeID := created["id"].(string)

	// Delete route
	delReq := httptest.NewRequest("DELETE", "/api/v1/gateways/"+gw.ID+"/routes/"+routeID, nil)
	delResp, _ := fiberApp.Test(delReq)

	if delResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", delResp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(delResp.Body).Decode(&result)

	if result["message"] != "route deleted" {
		t.Errorf("Expected message 'route deleted', got '%v'", result["message"])
	}
}

// --- Group Tests ---

func TestGatewayCreateGroup(t *testing.T) {
	fiberApp, handler, gwRepo, _, appRepo := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw-cg", "my-gw-cg")
	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{UserID: "user-1", Name: "test-app-cg", RepoURL: "https://github.com/user/repo"})

	fiberApp.Post("/api/v1/gateways/:gwId/groups", injectUserID("user-1"), handler.CreateGroup)

	body := `{"name":"user-group","app_id":"` + testApp.ID + `"}`
	req := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/groups", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("Expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["name"] != "user-group" {
		t.Errorf("Expected name 'user-group', got '%v'", result["name"])
	}
}

func TestGatewayCreateGroupNoNameOrAppID(t *testing.T) {
	fiberApp, handler, gwRepo, _, _ := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw", "my-gw")

	fiberApp.Post("/api/v1/gateways/:gwId/groups", injectUserID("user-1"), handler.CreateGroup)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/groups", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestGatewayListGroups(t *testing.T) {
	fiberApp, handler, gwRepo, _, appRepo := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw-lg", "my-gw-lg")
	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{UserID: "user-1", Name: "test-app-lg", RepoURL: "https://github.com/user/repo"})

	fiberApp.Post("/api/v1/gateways/:gwId/groups", injectUserID("user-1"), handler.CreateGroup)
	fiberApp.Get("/api/v1/gateways/:gwId/groups", injectUserID("user-1"), handler.ListGroups)

	// Create group
	body := `{"name":"grp1","app_id":"` + testApp.ID + `"}`
	createReq := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/groups", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	fiberApp.Test(createReq)

	// List groups
	listReq := httptest.NewRequest("GET", "/api/v1/gateways/"+gw.ID+"/groups", nil)
	listResp, _ := fiberApp.Test(listReq)

	if listResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", listResp.StatusCode)
	}

	var result []map[string]interface{}
	json.NewDecoder(listResp.Body).Decode(&result)

	if len(result) != 1 {
		t.Errorf("Expected 1 group, got %d", len(result))
	}
}

func TestGatewayDeleteGroup(t *testing.T) {
	fiberApp, handler, gwRepo, _, appRepo := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw-dg", "my-gw-dg")
	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{UserID: "user-1", Name: "test-app-dg", RepoURL: "https://github.com/user/repo"})

	fiberApp.Post("/api/v1/gateways/:gwId/groups", injectUserID("user-1"), handler.CreateGroup)
	fiberApp.Delete("/api/v1/gateways/:gwId/groups/:groupId", injectUserID("user-1"), handler.DeleteGroup)

	// Create group
	body := `{"name":"grp1","app_id":"` + testApp.ID + `"}`
	createReq := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/groups", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, _ := fiberApp.Test(createReq)

	if createResp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(createResp.Body).Decode(&errBody)
		t.Fatalf("Expected 201 for group creation, got %d: %v", createResp.StatusCode, errBody)
	}

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	groupID := created["id"].(string)

	// Delete
	delReq := httptest.NewRequest("DELETE", "/api/v1/gateways/"+gw.ID+"/groups/"+groupID, nil)
	delResp, _ := fiberApp.Test(delReq)

	if delResp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", delResp.StatusCode)
	}
}

func TestGatewaySyncNotFound(t *testing.T) {
	fiberApp, handler, _, _, _ := setupGatewayTest()
	fiberApp.Post("/api/v1/gateways/:gwId/sync", injectUserID("user-1"), handler.SyncGateway)

	req := httptest.NewRequest("POST", "/api/v1/gateways/nonexistent/sync", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestGatewayRouteDefaultMethods(t *testing.T) {
	fiberApp, handler, gwRepo, _, appRepo := setupGatewayTest()

	gw, _ := gwRepo.CreateGateway(nil, "user-1", "", "my-gw-dm", "my-gw-dm")
	testApp, _ := appRepo.CreateApp(nil, &dto.CreateAppInput{UserID: "user-1", Name: "test-app-dm", RepoURL: "https://github.com/user/repo"})

	fiberApp.Post("/api/v1/gateways/:gwId/routes", injectUserID("user-1"), handler.CreateRoute)

	// No methods specified — should default to GET
	body := `{"name":"test","path":"/test","app_id":"` + testApp.ID + `"}`
	req := httptest.NewRequest("POST", "/api/v1/gateways/"+gw.ID+"/routes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 201 {
		var errBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("Expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	methods, ok := result["methods"].([]interface{})
	if !ok || len(methods) != 1 || methods[0] != "GET" {
		t.Errorf("Expected default methods [GET], got %v", result["methods"])
	}
}
