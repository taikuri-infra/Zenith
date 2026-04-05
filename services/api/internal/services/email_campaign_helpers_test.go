package services

import (
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
)

// --- processAll with nil email sender ---

func TestProcessAll_NilSender(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
	// Don't set email sender

	// Should not panic
	svc.processAll()
}

// --- Stop tests ---

func TestEmailCampaignService_Stop(t *testing.T) {
	svc := newTestEmailCampaignSvc()
	// Should not panic
	svc.Stop()
}
