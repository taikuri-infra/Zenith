package store

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemorySessionRepository is an in-memory implementation of SessionRepository.
type MemorySessionRepository struct {
	mu       sync.RWMutex
	sessions map[string]*entities.Session
}

// NewMemorySessionRepository creates a new MemorySessionRepository.
func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{
		sessions: make(map[string]*entities.Session),
	}
}

func detectDevice(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone"):
		return "Mobile"
	case strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad"):
		return "Tablet"
	default:
		return "Desktop"
	}
}

func (r *MemorySessionRepository) CreateSession(_ context.Context, userID, ipAddress, userAgent string) (*entities.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	session := &entities.Session{
		ID:         uuid.New().String(),
		UserID:     userID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Device:     detectDevice(userAgent),
		CreatedAt:  now,
		ExpiresAt:  now.Add(24 * time.Hour),
		LastSeenAt: now,
	}

	r.sessions[session.ID] = session
	return session, nil
}

func (r *MemorySessionRepository) GetSession(_ context.Context, id string) (*entities.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, ok := r.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return session, nil
}

func (r *MemorySessionRepository) ListSessionsByUser(_ context.Context, userID string) ([]entities.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.Session
	for _, s := range r.sessions {
		if s.UserID == userID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (r *MemorySessionRepository) DeleteSession(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.sessions[id]; !ok {
		return fmt.Errorf("session not found: %s", id)
	}
	delete(r.sessions, id)
	return nil
}

func (r *MemorySessionRepository) DeleteAllUserSessions(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, s := range r.sessions {
		if s.UserID == userID {
			delete(r.sessions, id)
		}
	}
	return nil
}

func (r *MemorySessionRepository) UpdateLastSeen(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[id]
	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}
	session.LastSeenAt = time.Now()
	return nil
}
