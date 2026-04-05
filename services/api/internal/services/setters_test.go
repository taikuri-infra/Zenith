package services

import (
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
)

// --- BillingService.SetHarborClient ---

func TestBillingService_SetHarborClient(t *testing.T) {
	svc := newTestBillingService(nil)
	svc.SetHarborClient(nil) // nil = no Harbor
	// No panic means success
}

// --- CustomerService.SetWorkflows ---

func TestCustomerService_SetWorkflows(t *testing.T) {
	customerRepo := memory.NewMemoryCustomerRepository()
	adminRepo := memory.NewMemoryAdminRepository()

	svc := NewCustomerService(customerRepo, adminRepo, nil)
	svc.SetWorkflows(nil)
	// No panic means success
}

// --- TeamMemberService.SetEmailSender ---

func TestTeamMemberService_SetEmailSender(t *testing.T) {
	teamRepo := memory.NewMemoryTeamMemberRepository()
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()

	svc := NewTeamMemberService(teamRepo, userRepo, planRepo, "test-jwt-secret")
	svc.SetEmailSender(nil, "https://app.zenith.dev")
	// No panic means success
}

// --- SupportService.SetEmailSender ---

func TestSupportService_SetEmailSender(t *testing.T) {
	supportRepo := memory.NewMemorySupportRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	userRepo := memory.NewMemoryUserRepository()

	svc := NewSupportService(supportRepo, planRepo, userRepo)
	svc.SetEmailSender(nil, "https://app.zenith.dev", "admin@zenith.dev")
	// No panic means success
}
