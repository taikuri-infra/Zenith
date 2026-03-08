package memory

import (
	"context"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// MemoryNotificationRepository is an in-memory notification and activity store.
type MemoryNotificationRepository struct {
	mu             sync.RWMutex
	notifications  []entities.Notification
	activities     []entities.ActivityEntry
}

// NewMemoryNotificationRepository creates a new in-memory notification repo.
func NewMemoryNotificationRepository() *MemoryNotificationRepository {
	return &MemoryNotificationRepository{}
}

func (r *MemoryNotificationRepository) CreateNotification(_ context.Context, notif *entities.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifications = append(r.notifications, *notif)
	return nil
}

func (r *MemoryNotificationRepository) ListByUser(_ context.Context, userID string, limit int) ([]entities.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.Notification
	// Iterate in reverse for newest-first
	for i := len(r.notifications) - 1; i >= 0; i-- {
		if r.notifications[i].UserID == userID {
			result = append(result, r.notifications[i])
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (r *MemoryNotificationRepository) MarkRead(_ context.Context, userID string, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	for i := range r.notifications {
		if r.notifications[i].UserID == userID && idSet[r.notifications[i].ID] {
			r.notifications[i].Read = true
		}
	}
	return nil
}

func (r *MemoryNotificationRepository) MarkAllRead(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.notifications {
		if r.notifications[i].UserID == userID {
			r.notifications[i].Read = true
		}
	}
	return nil
}

func (r *MemoryNotificationRepository) CountUnread(_ context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, n := range r.notifications {
		if n.UserID == userID && !n.Read {
			count++
		}
	}
	return count, nil
}

func (r *MemoryNotificationRepository) AddActivity(_ context.Context, entry *entities.ActivityEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.activities = append(r.activities, *entry)
	return nil
}

func (r *MemoryNotificationRepository) ListActivity(_ context.Context, userID string, limit int) ([]entities.ActivityEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.ActivityEntry
	for i := len(r.activities) - 1; i >= 0; i-- {
		if r.activities[i].UserID == userID {
			result = append(result, r.activities[i])
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}
