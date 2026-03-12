package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
)

var _ ports.ReferralRepository = (*MemoryReferralRepository)(nil)

type MemoryReferralRepository struct {
	mu      sync.RWMutex
	rewards map[string]*entities.ReferralReward
}

func NewMemoryReferralRepository() *MemoryReferralRepository {
	return &MemoryReferralRepository{rewards: make(map[string]*entities.ReferralReward)}
}

func (r *MemoryReferralRepository) CreateReward(_ context.Context, reward *entities.ReferralReward) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rr := range r.rewards {
		if rr.ReferrerID == reward.ReferrerID && rr.ReferredID == reward.ReferredID {
			return fmt.Errorf("duplicate referral: referrer %s already referred %s", reward.ReferrerID, reward.ReferredID)
		}
	}
	if reward.ID == "" {
		reward.ID = uuid.New().String()
	}
	if reward.CreatedAt.IsZero() {
		reward.CreatedAt = time.Now()
	}
	cp := *reward
	r.rewards[reward.ID] = &cp
	return nil
}

func (r *MemoryReferralRepository) ListByReferrer(_ context.Context, referrerID string) ([]entities.ReferralReward, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.ReferralReward
	for _, rr := range r.rewards {
		if rr.ReferrerID == referrerID {
			result = append(result, *rr)
		}
	}
	return result, nil
}

func (r *MemoryReferralRepository) CountByReferrer(_ context.Context, referrerID string, since time.Time) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, rr := range r.rewards {
		if rr.ReferrerID == referrerID && !rr.CreatedAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (r *MemoryReferralRepository) CreditReward(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rr, ok := r.rewards[id]
	if !ok {
		return fmt.Errorf("referral reward %s not found", id)
	}
	rr.Status = entities.ReferralCredited
	now := time.Now()
	rr.CreditedAt = &now
	return nil
}

func (r *MemoryReferralRepository) GetSummary(_ context.Context, userID, _ string) (*entities.ReferralSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	summary := &entities.ReferralSummary{}
	for _, rr := range r.rewards {
		if rr.ReferrerID == userID {
			summary.TotalReferrals++
			switch rr.Status {
			case entities.ReferralCredited:
				summary.Credited++
			case entities.ReferralPending:
				summary.Pending++
			}
		}
	}
	return summary, nil
}

func (r *MemoryReferralRepository) ListAll(_ context.Context, limit, offset int) ([]entities.ReferralReward, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []entities.ReferralReward
	for _, rr := range r.rewards {
		all = append(all, *rr)
	}
	if offset >= len(all) {
		return nil, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}
