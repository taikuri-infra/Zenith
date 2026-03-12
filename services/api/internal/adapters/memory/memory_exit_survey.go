package memory

import (
	"context"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
)

var _ ports.ExitSurveyRepository = (*MemoryExitSurveyRepository)(nil)

type MemoryExitSurveyRepository struct {
	mu      sync.RWMutex
	surveys []entities.ExitSurvey
}

func NewMemoryExitSurveyRepository() *MemoryExitSurveyRepository {
	return &MemoryExitSurveyRepository{}
}

func (r *MemoryExitSurveyRepository) Create(_ context.Context, survey *entities.ExitSurvey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if survey.ID == "" {
		survey.ID = uuid.New().String()
	}
	if survey.CreatedAt.IsZero() {
		survey.CreatedAt = time.Now()
	}
	r.surveys = append(r.surveys, *survey)
	return nil
}

func (r *MemoryExitSurveyRepository) List(_ context.Context, limit, offset int) ([]entities.ExitSurvey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if offset >= len(r.surveys) {
		return nil, nil
	}
	end := offset + limit
	if end > len(r.surveys) {
		end = len(r.surveys)
	}
	result := make([]entities.ExitSurvey, end-offset)
	copy(result, r.surveys[offset:end])
	return result, nil
}

func (r *MemoryExitSurveyRepository) GetStats(_ context.Context) (*entities.ExitSurveyStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stats := &entities.ExitSurveyStats{ByReason: make(map[string]int), Total: len(r.surveys)}
	for _, s := range r.surveys {
		stats.ByReason[s.Reason]++
	}
	return stats, nil
}
