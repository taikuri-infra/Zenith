package services

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
)

// --- shouldSend* trigger tests ---

func newTestEmailCampaignSvc() *EmailCampaignService {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	return NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
}

func TestShouldSendWelcome_RecentSignup(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	// Signed up 30 minutes ago
	signupTime := time.Now().Add(-30 * time.Minute)
	result := svc.shouldSendWelcome(context.Background(), "user-1", signupTime)
	if !result {
		t.Error("Expected shouldSendWelcome=true for recent signup")
	}
}

func TestShouldSendWelcome_OldSignup(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	// Signed up 3 hours ago
	signupTime := time.Now().Add(-3 * time.Hour)
	result := svc.shouldSendWelcome(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendWelcome=false for old signup")
	}
}

func TestShouldSendDay1_TooEarly(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	// Signed up 10 hours ago (too early for day 1)
	signupTime := time.Now().Add(-10 * time.Hour)
	result := svc.shouldSendDay1(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendDay1=false for too-early signup")
	}
}

func TestShouldSendDay1_InWindow(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	// Signed up 24 hours ago (in the 20h-48h window)
	signupTime := time.Now().Add(-24 * time.Hour)
	result := svc.shouldSendDay1(context.Background(), "user-1", signupTime)
	if !result {
		t.Error("Expected shouldSendDay1=true for signup in window with no app")
	}
}

func TestShouldSendDay1_TooLate(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	// Signed up 3 days ago (too late for day 1)
	signupTime := time.Now().Add(-72 * time.Hour)
	result := svc.shouldSendDay1(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendDay1=false for too-late signup")
	}
}

func TestShouldSendDay3Engage_TooEarly(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	signupTime := time.Now().Add(-48 * time.Hour)
	result := svc.shouldSendDay3Engage(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendDay3Engage=false when too early")
	}
}

func TestShouldSendDay3Engage_TooLate(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	signupTime := time.Now().Add(-120 * time.Hour)
	result := svc.shouldSendDay3Engage(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendDay3Engage=false when too late")
	}
}

func TestShouldSendDay3Nudge_TooEarly(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	signupTime := time.Now().Add(-48 * time.Hour)
	result := svc.shouldSendDay3Nudge(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendDay3Nudge=false when too early")
	}
}

func TestShouldSendDay3Nudge_InWindow_NoApp(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	signupTime := time.Now().Add(-72 * time.Hour)
	result := svc.shouldSendDay3Nudge(context.Background(), "user-1", signupTime)
	if !result {
		t.Error("Expected shouldSendDay3Nudge=true in window with no app")
	}
}

func TestShouldSendDay3Nudge_TooLate(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	signupTime := time.Now().Add(-120 * time.Hour)
	result := svc.shouldSendDay3Nudge(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendDay3Nudge=false when too late")
	}
}

func TestShouldSendDay7Trial_TooEarly(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	signupTime := time.Now().Add(-3 * 24 * time.Hour)
	result := svc.shouldSendDay7Trial(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendDay7Trial=false when too early")
	}
}

func TestShouldSendDay7Trial_TooLate(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	signupTime := time.Now().Add(-10 * 24 * time.Hour)
	result := svc.shouldSendDay7Trial(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendDay7Trial=false when too late")
	}
}

func TestShouldSendDay14_TooEarly(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	signupTime := time.Now().Add(-10 * 24 * time.Hour)
	result := svc.shouldSendDay14(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendDay14=false when too early")
	}
}

func TestShouldSendDay14_InWindow(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	signupTime := time.Now().Add(-14 * 24 * time.Hour)
	result := svc.shouldSendDay14(context.Background(), "user-1", signupTime)
	if !result {
		t.Error("Expected shouldSendDay14=true in window for free user")
	}
}

func TestShouldSendDay14_TooLate(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	signupTime := time.Now().Add(-20 * 24 * time.Hour)
	result := svc.shouldSendDay14(context.Background(), "user-1", signupTime)
	if result {
		t.Error("Expected shouldSendDay14=false when too late")
	}
}
