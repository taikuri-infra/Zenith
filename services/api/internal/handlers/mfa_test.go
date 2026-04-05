package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupMFATest() (*fiber.App, *handlers.MFAHandler, *memory.MemoryMFARepository, *memory.MemoryUserPlanRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	mfaRepo := memory.NewMemoryMFARepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	handler := handlers.NewMFAHandler(mfaRepo, planRepo)
	return app, handler, mfaRepo, planRepo
}

func TestMFAGetStatusNoEnrollment(t *testing.T) {
	app, handler, _, _ := setupMFATest()
	app.Get("/api/v1/mfa/status", injectUserID("user-1"), handler.GetStatus)

	req := httptest.NewRequest("GET", "/api/v1/mfa/status", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != string(entities.MFAStatusDisabled) {
		t.Errorf("Expected status 'disabled', got '%v'", result["status"])
	}
}

func TestMFAEnableFreePlanForbidden(t *testing.T) {
	app, handler, _, _ := setupMFATest()
	// Default plan is free
	app.Post("/api/v1/mfa/enable", injectUserID("user-1"), handler.Enable)

	req := httptest.NewRequest("POST", "/api/v1/mfa/enable", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestMFAEnableProPlan(t *testing.T) {
	app, handler, _, planRepo := setupMFATest()
	planRepo.SetUserPlan(nil, "user-1", entities.PlanPro)

	app.Post("/api/v1/mfa/enable", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		c.Locals("email", "test@example.com")
		return c.Next()
	}, handler.Enable)

	req := httptest.NewRequest("POST", "/api/v1/mfa/enable", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["secret"] == nil || result["secret"] == "" {
		t.Error("Expected non-empty secret")
	}
	if result["otpauth_uri"] == nil || result["otpauth_uri"] == "" {
		t.Error("Expected non-empty otpauth_uri")
	}
	if result["backup_codes"] == nil {
		t.Error("Expected backup_codes")
	}
}

func TestMFAVerifyNoCode(t *testing.T) {
	app, handler, _, _ := setupMFATest()
	app.Post("/api/v1/mfa/verify", injectUserID("user-1"), handler.Verify)

	body := `{"code":""}`
	req := httptest.NewRequest("POST", "/api/v1/mfa/verify", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestMFAVerifyInvalidCodeLength(t *testing.T) {
	app, handler, _, _ := setupMFATest()
	app.Post("/api/v1/mfa/verify", injectUserID("user-1"), handler.Verify)

	body := `{"code":"12345"}`
	req := httptest.NewRequest("POST", "/api/v1/mfa/verify", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestMFAVerifyNoPendingEnrollment(t *testing.T) {
	app, handler, _, _ := setupMFATest()
	app.Post("/api/v1/mfa/verify", injectUserID("user-1"), handler.Verify)

	// The memory repo returns a disabled enrollment (no error) when no enrollment exists
	// The handler should check that status is pending
	body := `{"code":"123456"}`
	req := httptest.NewRequest("POST", "/api/v1/mfa/verify", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestMFADisableNoCode(t *testing.T) {
	app, handler, _, _ := setupMFATest()
	app.Post("/api/v1/mfa/disable", injectUserID("user-1"), handler.Disable)

	body := `{"code":""}`
	req := httptest.NewRequest("POST", "/api/v1/mfa/disable", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestMFADisableNotEnabled(t *testing.T) {
	app, handler, _, _ := setupMFATest()
	app.Post("/api/v1/mfa/disable", injectUserID("user-1"), handler.Disable)

	body := `{"code":"123456"}`
	req := httptest.NewRequest("POST", "/api/v1/mfa/disable", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestMFARegenerateBackupCodesNotEnabled(t *testing.T) {
	app, handler, _, _ := setupMFATest()
	app.Post("/api/v1/mfa/regenerate", injectUserID("user-1"), handler.RegenerateBackupCodes)

	req := httptest.NewRequest("POST", "/api/v1/mfa/regenerate", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestMFAGetStatusWithEnrollment(t *testing.T) {
	app, handler, mfaRepo, _ := setupMFATest()

	// Start and confirm enrollment manually
	mfaRepo.StartEnrollment(nil, "user-1")
	mfaRepo.ConfirmEnrollment(nil, "user-1")

	app.Get("/api/v1/mfa/status", injectUserID("user-1"), handler.GetStatus)

	req := httptest.NewRequest("GET", "/api/v1/mfa/status", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != string(entities.MFAStatusEnabled) {
		t.Errorf("Expected status 'enabled', got '%v'", result["status"])
	}
}

func TestMFARegenerateBackupCodesEnabled(t *testing.T) {
	app, handler, mfaRepo, _ := setupMFATest()

	mfaRepo.StartEnrollment(nil, "user-1")
	mfaRepo.ConfirmEnrollment(nil, "user-1")

	app.Post("/api/v1/mfa/regenerate", injectUserID("user-1"), handler.RegenerateBackupCodes)

	req := httptest.NewRequest("POST", "/api/v1/mfa/regenerate", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	codes, ok := result["backup_codes"].([]interface{})
	if !ok || len(codes) == 0 {
		t.Error("Expected non-empty backup_codes array")
	}
}
