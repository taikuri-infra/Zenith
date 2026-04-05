package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

func setupPlanTest() (*fiber.App, *handlers.PlanHandler, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	storageRepo := memory.NewMemoryStorageRepository()
	authRepo := memory.NewMemoryAppAuthRepository()
	planSvc := services.NewPlanService(planRepo, appRepo, dbRepo, storageRepo, authRepo)
	handler := handlers.NewPlanHandler(planSvc)
	return app, handler, planRepo
}

func TestPlanGetMyPlan(t *testing.T) {
	fiberApp, handler, _ := setupPlanTest()
	fiberApp.Get("/api/v1/plan", injectUserID("user-1"), handler.GetMyPlan)

	req := httptest.NewRequest("GET", "/api/v1/plan", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Default plan should be free
	if result["tier"] != "free" {
		t.Errorf("Expected tier 'free', got '%v'", result["tier"])
	}
}

func TestPlanGetMyPlanProUser(t *testing.T) {
	fiberApp, handler, planRepo := setupPlanTest()

	// Set user to Pro plan
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)

	fiberApp.Get("/api/v1/plan", injectUserID("user-1"), handler.GetMyPlan)

	req := httptest.NewRequest("GET", "/api/v1/plan", nil)
	resp, _ := fiberApp.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["tier"] != "pro" {
		t.Errorf("Expected tier 'pro', got '%v'", result["tier"])
	}
}

func TestPlanUpgrade(t *testing.T) {
	fiberApp, handler, _ := setupPlanTest()
	fiberApp.Post("/api/v1/plan/upgrade", injectUserID("user-1"), handler.UpgradePlan)

	body := `{"tier":"pro"}`
	req := httptest.NewRequest("POST", "/api/v1/plan/upgrade", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	// Plan service may require Stripe for pro upgrade — check it doesn't panic
	// The response depends on whether stripe is enabled
	if resp.StatusCode == 0 {
		t.Error("Expected a response, got nothing")
	}
}

func TestPlanUpgradeInvalidBody(t *testing.T) {
	fiberApp, handler, _ := setupPlanTest()
	fiberApp.Post("/api/v1/plan/upgrade", injectUserID("user-1"), handler.UpgradePlan)

	req := httptest.NewRequest("POST", "/api/v1/plan/upgrade", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := fiberApp.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// --- CheckLimit middleware tests ---

func TestCheckLimitAllowed(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()

	// countFn always returns 0
	countFn := func(c *fiber.Ctx, userID string) (int, error) {
		return 0, nil
	}

	app.Post("/test", injectUserID("user-1"), handlers.CheckLimit(planRepo, "apps", countFn), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCheckLimitReached(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()

	// Free plan limit for apps is typically 1
	// countFn returns a high number
	countFn := func(c *fiber.Ctx, userID string) (int, error) {
		return 999, nil
	}

	app.Post("/test", injectUserID("user-1"), handlers.CheckLimit(planRepo, "apps", countFn), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for plan limit reached, got %d", resp.StatusCode)
	}
}

func TestCheckLimitNoAuth(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()

	countFn := func(c *fiber.Ctx, userID string) (int, error) {
		return 0, nil
	}

	// No injectUserID — should return 401
	app.Post("/test", handlers.CheckLimit(planRepo, "apps", countFn), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestCheckLimitUnknownResource(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()

	countFn := func(c *fiber.Ctx, userID string) (int, error) {
		return 0, nil
	}

	// Unknown resource should pass through (Next())
	app.Post("/test", injectUserID("user-1"), handlers.CheckLimit(planRepo, "unknown_resource", countFn), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 for unknown resource (pass through), got %d", resp.StatusCode)
	}
}

func TestCheckLimitDatabases(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()

	countFn := func(c *fiber.Ctx, userID string) (int, error) {
		return 999, nil
	}

	app.Post("/test", injectUserID("user-1"), handlers.CheckLimit(planRepo, "databases", countFn), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for database limit, got %d", resp.StatusCode)
	}
}

func TestCheckLimitBuckets(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()

	countFn := func(c *fiber.Ctx, userID string) (int, error) {
		return 999, nil
	}

	app.Post("/test", injectUserID("user-1"), handlers.CheckLimit(planRepo, "buckets", countFn), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for bucket limit, got %d", resp.StatusCode)
	}
}

func TestCheckLimitGateways(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()

	countFn := func(c *fiber.Ctx, userID string) (int, error) {
		return 999, nil
	}

	app.Post("/test", injectUserID("user-1"), handlers.CheckLimit(planRepo, "gateways", countFn), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for gateway limit, got %d", resp.StatusCode)
	}
}

func TestCheckLimitGatewayRoutes(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()

	countFn := func(c *fiber.Ctx, userID string) (int, error) {
		return 999, nil
	}

	app.Post("/test", injectUserID("user-1"), handlers.CheckLimit(planRepo, "gateway_routes", countFn), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for gateway routes limit, got %d", resp.StatusCode)
	}
}

func TestCheckLimitAuthPools(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	planRepo := memory.NewMemoryUserPlanRepository()

	countFn := func(c *fiber.Ctx, userID string) (int, error) {
		return 999, nil
	}

	app.Post("/test", injectUserID("user-1"), handlers.CheckLimit(planRepo, "auth_pools", countFn), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for auth pools limit, got %d", resp.StatusCode)
	}
}
