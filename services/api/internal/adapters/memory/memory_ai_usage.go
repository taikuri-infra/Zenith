package memory

import (
	"context"
	"sync"
	"time"
)

type aiUsageEntry struct {
	UserID    string
	UsageType string
	Model     string
	TokensIn  int
	TokensOut int
	CostUSD   float64
	CreatedAt time.Time
}

// MemoryAIUsageRepository is an in-memory AI usage repository for dev/testing.
type MemoryAIUsageRepository struct {
	mu      sync.RWMutex
	entries []aiUsageEntry
}

// NewMemoryAIUsageRepository creates a new MemoryAIUsageRepository.
func NewMemoryAIUsageRepository() *MemoryAIUsageRepository {
	return &MemoryAIUsageRepository{}
}

// RecordUsage records an AI usage event.
func (r *MemoryAIUsageRepository) RecordUsage(_ context.Context, userID, usageType, model string, tokensIn, tokensOut int, costUSD float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, aiUsageEntry{
		UserID:    userID,
		UsageType: usageType,
		Model:     model,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		CostUSD:   costUSD,
		CreatedAt: time.Now(),
	})
	return nil
}

// GetMonthlyUsage returns the count of AI calls in the given month.
func (r *MemoryAIUsageRepository) GetMonthlyUsage(_ context.Context, userID string, month time.Time) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	count := 0
	for _, e := range r.entries {
		if e.UserID == userID && !e.CreatedAt.Before(start) && e.CreatedAt.Before(end) {
			count++
		}
	}
	return count, nil
}
