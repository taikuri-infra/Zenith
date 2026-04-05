package services

import (
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
)

func TestDormantCleanupService_Cleanup(t *testing.T) {
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	appRepo := memory.NewMemoryAppRepository()
	eventRepo := memory.NewMemoryUserEventRepository()

	svc := NewDormantCleanupService(userRepo, planRepo, appRepo, eventRepo)
	// Direct call to cleanup — should not panic
	svc.cleanup()
}
