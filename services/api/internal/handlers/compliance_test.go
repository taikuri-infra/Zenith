package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupComplianceTest() (*fiber.App, *handlers.ComplianceHandler) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	mfaRepo := memory.NewMemoryMFARepository()
	ipRepo := memory.NewMemoryIPWhitelistRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	adminRepo := memory.NewMemoryAdminRepository()
	handler := handlers.NewComplianceHandler(mfaRepo, ipRepo, planRepo, adminRepo)
	return app, handler
}

func TestComplianceGetStatus(t *testing.T) {
	app, handler := setupComplianceTest()
	app.Get("/api/v1/compliance", injectUserID("user-1"), handler.GetStatus)

	req := httptest.NewRequest("GET", "/api/v1/compliance", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Checks  []handlers.ComplianceCheck `json:"checks"`
		Summary map[string]interface{}     `json:"summary"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Checks) == 0 {
		t.Fatal("Expected non-empty compliance checks")
	}

	// Should have summary with counts
	total, _ := result.Summary["total"].(float64)
	if total == 0 {
		t.Error("Expected non-zero total in summary")
	}
}

func TestComplianceGetStatusMFADisabled(t *testing.T) {
	app, handler := setupComplianceTest()
	app.Get("/api/v1/compliance", injectUserID("user-1"), handler.GetStatus)

	req := httptest.NewRequest("GET", "/api/v1/compliance", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Checks []handlers.ComplianceCheck `json:"checks"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	// Find MFA check - should be "fail" since no enrollment
	for _, check := range result.Checks {
		if check.Item == "Multi-Factor Authentication (MFA)" {
			if check.Status != "fail" {
				t.Errorf("Expected MFA status 'fail', got '%s'", check.Status)
			}
			return
		}
	}
	t.Error("Expected to find MFA compliance check")
}

func TestComplianceGetStatusWithMFAEnabled(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	mfaRepo := memory.NewMemoryMFARepository()
	ipRepo := memory.NewMemoryIPWhitelistRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	adminRepo := memory.NewMemoryAdminRepository()

	// Enable MFA for user
	mfaRepo.StartEnrollment(nil, "user-1")
	mfaRepo.ConfirmEnrollment(nil, "user-1")

	handler := handlers.NewComplianceHandler(mfaRepo, ipRepo, planRepo, adminRepo)
	app.Get("/api/v1/compliance", injectUserID("user-1"), handler.GetStatus)

	req := httptest.NewRequest("GET", "/api/v1/compliance", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Checks []handlers.ComplianceCheck `json:"checks"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	for _, check := range result.Checks {
		if check.Item == "Multi-Factor Authentication (MFA)" {
			if check.Status != "pass" {
				t.Errorf("Expected MFA status 'pass', got '%s'", check.Status)
			}
			return
		}
	}
	t.Error("Expected to find MFA compliance check")
}

func TestComplianceEncryptionAlwaysPass(t *testing.T) {
	app, handler := setupComplianceTest()
	app.Get("/api/v1/compliance", injectUserID("user-1"), handler.GetStatus)

	req := httptest.NewRequest("GET", "/api/v1/compliance", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Checks []handlers.ComplianceCheck `json:"checks"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	// Encryption at Rest should always be pass
	for _, check := range result.Checks {
		if check.Item == "Encryption at Rest" {
			if check.Status != "pass" {
				t.Errorf("Expected encryption at rest 'pass', got '%s'", check.Status)
			}
			return
		}
	}
	t.Error("Expected to find Encryption at Rest check")
}
