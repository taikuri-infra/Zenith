package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestReferralCreateReward(t *testing.T) {
	repo := NewMemoryReferralRepository()
	ctx := context.Background()

	reward := &entities.ReferralReward{
		ReferrerID: "referrer-1",
		ReferredID: "referred-1",
		Status:     entities.ReferralPending,
		RewardType: "pro_month",
	}

	err := repo.CreateReward(ctx, reward)
	if err != nil {
		t.Fatalf("CreateReward: expected no error, got %v", err)
	}
	if reward.ID == "" {
		t.Error("CreateReward: expected ID to be auto-generated")
	}
}

func TestReferralCreateDuplicate(t *testing.T) {
	repo := NewMemoryReferralRepository()
	ctx := context.Background()

	repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r1", ReferredID: "ref1",
		Status: entities.ReferralPending, RewardType: "pro_month",
	})

	err := repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r1", ReferredID: "ref1",
		Status: entities.ReferralPending, RewardType: "pro_month",
	})
	if err == nil {
		t.Error("CreateReward duplicate: expected error")
	}
}

func TestReferralListByReferrer(t *testing.T) {
	repo := NewMemoryReferralRepository()
	ctx := context.Background()

	repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r1", ReferredID: "ref1",
		Status: entities.ReferralPending, RewardType: "pro_month",
	})
	repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r1", ReferredID: "ref2",
		Status: entities.ReferralCredited, RewardType: "pro_month",
	})
	repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r2", ReferredID: "ref3",
		Status: entities.ReferralPending, RewardType: "pro_month",
	})

	rewards, err := repo.ListByReferrer(ctx, "r1")
	if err != nil {
		t.Fatalf("ListByReferrer: %v", err)
	}
	if len(rewards) != 2 {
		t.Errorf("ListByReferrer: expected 2, got %d", len(rewards))
	}
}

func TestReferralCountByReferrer(t *testing.T) {
	repo := NewMemoryReferralRepository()
	ctx := context.Background()

	repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r1", ReferredID: "ref1",
		Status: entities.ReferralPending, RewardType: "pro_month",
	})
	repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r1", ReferredID: "ref2",
		Status: entities.ReferralPending, RewardType: "pro_month",
	})

	since := time.Now().Add(-1 * time.Hour)
	count, err := repo.CountByReferrer(ctx, "r1", since)
	if err != nil {
		t.Fatalf("CountByReferrer: %v", err)
	}
	if count != 2 {
		t.Errorf("CountByReferrer: expected 2, got %d", count)
	}
}

func TestReferralCreditReward(t *testing.T) {
	repo := NewMemoryReferralRepository()
	ctx := context.Background()

	reward := &entities.ReferralReward{
		ReferrerID: "r1", ReferredID: "ref1",
		Status: entities.ReferralPending, RewardType: "pro_month",
	}
	repo.CreateReward(ctx, reward)

	err := repo.CreditReward(ctx, reward.ID)
	if err != nil {
		t.Fatalf("CreditReward: %v", err)
	}

	rewards, _ := repo.ListByReferrer(ctx, "r1")
	if len(rewards) != 1 {
		t.Fatalf("expected 1 reward")
	}
	if rewards[0].Status != entities.ReferralCredited {
		t.Errorf("CreditReward: expected status 'credited', got %s", rewards[0].Status)
	}
	if rewards[0].CreditedAt == nil {
		t.Error("CreditReward: expected CreditedAt to be set")
	}
}

func TestReferralCreditNotFound(t *testing.T) {
	repo := NewMemoryReferralRepository()
	ctx := context.Background()

	err := repo.CreditReward(ctx, "nonexistent")
	if err == nil {
		t.Error("CreditReward nonexistent: expected error")
	}
}

func TestReferralGetSummary(t *testing.T) {
	repo := NewMemoryReferralRepository()
	ctx := context.Background()

	// Simulate a user with a referral code
	repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r1", ReferredID: "ref1",
		Status: entities.ReferralCredited, RewardType: "pro_month",
	})
	repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r1", ReferredID: "ref2",
		Status: entities.ReferralPending, RewardType: "pro_month",
	})

	summary, err := repo.GetSummary(ctx, "r1", "https://app.example.com")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summary.TotalReferrals != 2 {
		t.Errorf("GetSummary TotalReferrals: expected 2, got %d", summary.TotalReferrals)
	}
	if summary.Credited != 1 {
		t.Errorf("GetSummary Credited: expected 1, got %d", summary.Credited)
	}
	if summary.Pending != 1 {
		t.Errorf("GetSummary Pending: expected 1, got %d", summary.Pending)
	}
}

func TestReferralListAll(t *testing.T) {
	repo := NewMemoryReferralRepository()
	ctx := context.Background()

	repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r1", ReferredID: "ref1",
		Status: entities.ReferralPending, RewardType: "pro_month",
	})
	repo.CreateReward(ctx, &entities.ReferralReward{
		ReferrerID: "r2", ReferredID: "ref2",
		Status: entities.ReferralPending, RewardType: "pro_month",
	})

	all, err := repo.ListAll(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("ListAll: expected 2, got %d", len(all))
	}
}
