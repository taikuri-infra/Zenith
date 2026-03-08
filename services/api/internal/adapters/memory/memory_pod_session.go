package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// MemoryPodExecSessionRepository is an in-memory implementation for dev/test.
type MemoryPodExecSessionRepository struct {
	mu       sync.RWMutex
	sessions []entities.PodExecSession
}

func NewMemoryPodExecSessionRepository() *MemoryPodExecSessionRepository {
	return &MemoryPodExecSessionRepository{}
}

func (r *MemoryPodExecSessionRepository) CreateSession(_ context.Context, session *entities.PodExecSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions = append(r.sessions, *session)
	return nil
}

func (r *MemoryPodExecSessionRepository) EndSession(_ context.Context, id string, recordingKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, s := range r.sessions {
		if s.ID == id {
			now := time.Now()
			r.sessions[i].Status = entities.PodSessionCompleted
			r.sessions[i].EndedAt = &now
			r.sessions[i].DurationSecs = int(now.Sub(s.StartedAt).Seconds())
			r.sessions[i].RecordingKey = recordingKey
			return nil
		}
	}
	return fmt.Errorf("session not found: %s", id)
}

func (r *MemoryPodExecSessionRepository) GetSession(_ context.Context, id string) (*entities.PodExecSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.sessions {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("session not found: %s", id)
}

func (r *MemoryPodExecSessionRepository) ListByUser(_ context.Context, userID string, limit, offset int) ([]entities.PodExecSession, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []entities.PodExecSession
	for i := len(r.sessions) - 1; i >= 0; i-- {
		if r.sessions[i].UserID == userID {
			filtered = append(filtered, r.sessions[i])
		}
	}

	total := len(filtered)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (r *MemoryPodExecSessionRepository) ListAll(_ context.Context, limit, offset int) ([]entities.PodExecSession, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := len(r.sessions)
	// Return newest first
	reversed := make([]entities.PodExecSession, total)
	for i, s := range r.sessions {
		reversed[total-1-i] = s
	}

	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return reversed[offset:end], total, nil
}
