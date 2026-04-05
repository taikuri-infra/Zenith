package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func setupReferralTest() (*fiber.App, *handlers.ReferralHandler, *memory.MemoryReferralRepository) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	referralRepo := memory.NewMemoryReferralRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	handler := handlers.NewReferralHandler(referralRepo, eventRepo, "https://zenith.dev")
	return app, handler, referralRepo
}

func TestReferralGetSummary(t *testing.T) {
	app, handler, _ := setupReferralTest()
	app.Get("/api/v1/referral", injectUserID("user-1"), handler.GetSummary)

	req := httptest.NewRequest("GET", "/api/v1/referral", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result entities.ReferralSummary
	json.NewDecoder(resp.Body).Decode(&result)
	if result.TotalReferrals != 0 {
		t.Errorf("Expected 0 total referrals, got %d", result.TotalReferrals)
	}
}

func TestReferralGetSummaryWithRewards(t *testing.T) {
	app, handler, referralRepo := setupReferralTest()

	referralRepo.CreateReward(nil, &entities.ReferralReward{
		ReferrerID: "user-1",
		ReferredID: "user-2",
		Status:     entities.ReferralPending,
	})
	referralRepo.CreateReward(nil, &entities.ReferralReward{
		ReferrerID: "user-1",
		ReferredID: "user-3",
		Status:     entities.ReferralCredited,
	})

	app.Get("/api/v1/referral", injectUserID("user-1"), handler.GetSummary)

	req := httptest.NewRequest("GET", "/api/v1/referral", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result entities.ReferralSummary
	json.NewDecoder(resp.Body).Decode(&result)
	if result.TotalReferrals != 2 {
		t.Errorf("Expected 2 total referrals, got %d", result.TotalReferrals)
	}
}

func TestReferralListRewards(t *testing.T) {
	app, handler, referralRepo := setupReferralTest()

	referralRepo.CreateReward(nil, &entities.ReferralReward{
		ReferrerID: "user-1",
		ReferredID: "user-2",
		Status:     entities.ReferralPending,
	})

	app.Get("/api/v1/referral/rewards", injectUserID("user-1"), handler.ListRewards)

	req := httptest.NewRequest("GET", "/api/v1/referral/rewards", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.ReferralReward `json:"items"`
		Total int                       `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 1 {
		t.Errorf("Expected 1 reward, got %d", result.Total)
	}
}

func TestReferralListRewardsEmpty(t *testing.T) {
	app, handler, _ := setupReferralTest()
	app.Get("/api/v1/referral/rewards", injectUserID("user-1"), handler.ListRewards)

	req := httptest.NewRequest("GET", "/api/v1/referral/rewards", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.ReferralReward `json:"items"`
		Total int                       `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 0 {
		t.Errorf("Expected 0, got %d", result.Total)
	}
}

func TestReferralTrackShare(t *testing.T) {
	app, handler, _ := setupReferralTest()
	app.Post("/api/v1/referral/share", injectUserID("user-1"), handler.TrackShare)

	req := httptest.NewRequest("POST", "/api/v1/referral/share", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "share tracked" {
		t.Errorf("Expected 'share tracked', got '%v'", result["message"])
	}
}

func TestReferralAdminList(t *testing.T) {
	app, handler, referralRepo := setupReferralTest()

	referralRepo.CreateReward(nil, &entities.ReferralReward{
		ReferrerID: "user-1",
		ReferredID: "user-2",
		Status:     entities.ReferralPending,
	})

	app.Get("/api/v1/admin/referrals", handler.AdminList)

	req := httptest.NewRequest("GET", "/api/v1/admin/referrals", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Items []entities.ReferralReward `json:"items"`
		Total int                       `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Total != 1 {
		t.Errorf("Expected 1 referral, got %d", result.Total)
	}
}

func TestReferralAdminListEmpty(t *testing.T) {
	app, handler, _ := setupReferralTest()
	app.Get("/api/v1/admin/referrals", handler.AdminList)

	req := httptest.NewRequest("GET", "/api/v1/admin/referrals", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}
