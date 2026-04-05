package services

import (
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
)

func TestNewEmailCampaignService(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
	if svc == nil {
		t.Fatal("Expected non-nil EmailCampaignService")
	}
}

func TestEmailCampaignService_SetEmailSender(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
	svc.SetEmailSender(nil) // nil = dev mode
	// No panic means success
}

func TestEmailCampaignService_ProcessAll_NilSender(t *testing.T) {
	emailSendRepo := memory.NewMemoryEmailSendRepository()
	eventRepo := memory.NewMemoryUserEventRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewEmailCampaignService(emailSendRepo, eventRepo, userRepo, planRepo, "https://app.zenith.dev")
	// processAll should return early when emailSender is nil
	svc.processAll()
}
