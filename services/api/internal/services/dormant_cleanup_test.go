package services

import (
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
)

func TestNewDormantCleanupService(t *testing.T) {
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	appRepo := memory.NewMemoryAppRepository()
	eventRepo := memory.NewMemoryUserEventRepository()

	svc := NewDormantCleanupService(userRepo, planRepo, appRepo, eventRepo)
	if svc == nil {
		t.Fatal("Expected non-nil DormantCleanupService")
	}
}

func TestDormantCleanupService_StartStop(t *testing.T) {
	userRepo := memory.NewMemoryUserRepository()
	planRepo := memory.NewMemoryUserPlanRepository()
	appRepo := memory.NewMemoryAppRepository()
	eventRepo := memory.NewMemoryUserEventRepository()

	svc := NewDormantCleanupService(userRepo, planRepo, appRepo, eventRepo)
	svc.Start()
	svc.Stop()
	// No panic or hang means success
}
