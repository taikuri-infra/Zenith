// services/api/internal/adapters/memory/unlimited_plan_test.go
package memory_test

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestUnlimitedPlan_AlwaysEnterprise(t *testing.T) {
	inner := memory.NewMemoryUserPlanRepository()
	repo := memory.NewUnlimitedPlanRepository(inner)

	p, err := repo.GetUserPlan(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if p.Tier != entities.PlanEnterprise {
		t.Errorf("tier = %q, want enterprise", p.Tier)
	}
	if !p.Limits.CustomDomain || p.Limits.MaxRedisInstances == 0 || p.Limits.MaxApps < 1000 {
		t.Errorf("limits not unlimited: %+v", p.Limits)
	}
	if p.UserID != "user-1" {
		t.Errorf("user id = %q", p.UserID)
	}
}

func TestUnlimitedPlan_IgnoresStoredFreePlan(t *testing.T) {
	inner := memory.NewMemoryUserPlanRepository()
	_, _ = inner.SetUserPlan(context.Background(), "user-1", entities.PlanFree)
	repo := memory.NewUnlimitedPlanRepository(inner)
	p, _ := repo.GetUserPlan(context.Background(), "user-1")
	if p.Tier != entities.PlanEnterprise {
		t.Errorf("tier = %q, want enterprise", p.Tier)
	}
}
