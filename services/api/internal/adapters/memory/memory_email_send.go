package memory

import (
	"context"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
)

var _ ports.EmailSendRepository = (*MemoryEmailSendRepository)(nil)

type MemoryEmailSendRepository struct {
	mu    sync.RWMutex
	sends map[string]*entities.EmailSend
}

func NewMemoryEmailSendRepository() *MemoryEmailSendRepository {
	return &MemoryEmailSendRepository{sends: make(map[string]*entities.EmailSend)}
}

func (r *MemoryEmailSendRepository) Record(_ context.Context, send *entities.EmailSend) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Check unique constraint
	for _, s := range r.sends {
		if s.UserID == send.UserID && s.TemplateKey == send.TemplateKey {
			return nil // already sent
		}
	}
	if send.ID == "" {
		send.ID = uuid.New().String()
	}
	if send.SentAt.IsZero() {
		send.SentAt = time.Now()
	}
	cp := *send
	r.sends[send.ID] = &cp
	return nil
}

func (r *MemoryEmailSendRepository) HasSent(_ context.Context, userID, templateKey string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.sends {
		if s.UserID == userID && s.TemplateKey == templateKey {
			return true, nil
		}
	}
	return false, nil
}

func (r *MemoryEmailSendRepository) MarkOpened(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.sends[id]; ok && s.OpenedAt == nil {
		now := time.Now()
		s.OpenedAt = &now
	}
	return nil
}

func (r *MemoryEmailSendRepository) MarkClicked(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.sends[id]; ok && s.ClickedAt == nil {
		now := time.Now()
		s.ClickedAt = &now
	}
	return nil
}

func (r *MemoryEmailSendRepository) GetStats(_ context.Context) (*entities.EmailStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stats := &entities.EmailStats{ByTemplate: make(map[string]int)}
	for _, s := range r.sends {
		stats.Sent++
		if s.OpenedAt != nil {
			stats.Opened++
		}
		if s.ClickedAt != nil {
			stats.Clicked++
		}
		stats.ByTemplate[s.TemplateKey]++
	}
	return stats, nil
}

func (r *MemoryEmailSendRepository) ListByUser(_ context.Context, userID string) ([]entities.EmailSend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.EmailSend
	for _, s := range r.sends {
		if s.UserID == userID {
			result = append(result, *s)
		}
	}
	return result, nil
}
