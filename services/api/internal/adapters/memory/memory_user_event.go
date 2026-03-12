package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
)

var _ ports.UserEventRepository = (*MemoryUserEventRepository)(nil)

type MemoryUserEventRepository struct {
	mu     sync.RWMutex
	events []entities.UserEvent
}

func NewMemoryUserEventRepository() *MemoryUserEventRepository {
	return &MemoryUserEventRepository{}
}

func (r *MemoryUserEventRepository) Track(_ context.Context, event *entities.UserEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	r.events = append(r.events, *event)
	return nil
}

func (r *MemoryUserEventRepository) ListByUser(_ context.Context, userID string, limit, offset int) ([]entities.UserEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.UserEvent
	for i := len(r.events) - 1; i >= 0; i-- {
		if r.events[i].UserID == userID {
			result = append(result, r.events[i])
		}
	}
	if offset >= len(result) {
		return nil, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (r *MemoryUserEventRepository) ListByType(_ context.Context, eventType string, limit, offset int) ([]entities.UserEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.UserEvent
	for i := len(r.events) - 1; i >= 0; i-- {
		if r.events[i].EventType == eventType {
			result = append(result, r.events[i])
		}
	}
	if offset >= len(result) {
		return nil, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (r *MemoryUserEventRepository) CountByType(_ context.Context, eventType string, since time.Time) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, e := range r.events {
		if e.EventType == eventType && !e.CreatedAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (r *MemoryUserEventRepository) GetUserActivity(_ context.Context, userID string, since time.Time) ([]entities.UserEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.UserEvent
	for _, e := range r.events {
		if e.UserID == userID && !e.CreatedAt.Before(since) {
			result = append(result, e)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryUserEventRepository) GetFunnelData(_ context.Context, steps []string, since time.Time) (map[string]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]int)
	for _, step := range steps {
		users := make(map[string]bool)
		for _, e := range r.events {
			if e.EventType == step && !e.CreatedAt.Before(since) {
				users[e.UserID] = true
			}
		}
		result[step] = len(users)
	}
	return result, nil
}

func (r *MemoryUserEventRepository) PurgeOlderThan(_ context.Context, before time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var kept []entities.UserEvent
	purged := int64(0)
	for _, e := range r.events {
		if e.CreatedAt.Before(before) {
			purged++
		} else {
			kept = append(kept, e)
		}
	}
	r.events = kept
	return purged, nil
}
