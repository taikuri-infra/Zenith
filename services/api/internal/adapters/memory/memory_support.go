package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// MemorySupportRepository is an in-memory implementation of SupportRepository.
type MemorySupportRepository struct {
	mu       sync.RWMutex
	tickets  map[string]*entities.SupportTicket
	messages map[string][]entities.SupportMessage // ticketID -> messages
}

// NewMemorySupportRepository creates a new MemorySupportRepository.
func NewMemorySupportRepository() *MemorySupportRepository {
	return &MemorySupportRepository{
		tickets:  make(map[string]*entities.SupportTicket),
		messages: make(map[string][]entities.SupportMessage),
	}
}

func (r *MemorySupportRepository) CreateTicket(_ context.Context, ticket *entities.SupportTicket, initialMsg *entities.SupportMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tickets[ticket.ID] = ticket
	r.messages[ticket.ID] = []entities.SupportMessage{*initialMsg}
	return nil
}

func (r *MemorySupportRepository) GetTicket(_ context.Context, id string) (*entities.SupportTicket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tickets[id]
	if !ok {
		return nil, fmt.Errorf("ticket not found: %s", id)
	}
	return t, nil
}

func (r *MemorySupportRepository) ListTicketsByUser(_ context.Context, userID string) ([]entities.SupportTicket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.SupportTicket
	for _, t := range r.tickets {
		if t.UserID == userID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (r *MemorySupportRepository) ListAllTickets(_ context.Context, status string, limit, offset int) ([]entities.SupportTicket, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []entities.SupportTicket
	for _, t := range r.tickets {
		if status == "" || string(t.Status) == status {
			all = append(all, *t)
		}
	}

	total := len(all)
	if offset >= len(all) {
		return []entities.SupportTicket{}, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (r *MemorySupportRepository) UpdateTicketStatus(_ context.Context, id string, status entities.TicketStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.tickets[id]
	if !ok {
		return fmt.Errorf("ticket not found: %s", id)
	}
	t.Status = status
	now := time.Now()
	t.UpdatedAt = now
	if status == entities.TicketStatusClosed || status == entities.TicketStatusResolved {
		t.ClosedAt = &now
	}
	return nil
}

func (r *MemorySupportRepository) UpdateTicketAssignee(_ context.Context, id, adminUserID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.tickets[id]
	if !ok {
		return fmt.Errorf("ticket not found: %s", id)
	}
	t.AssignedTo = adminUserID
	t.UpdatedAt = time.Now()
	return nil
}

func (r *MemorySupportRepository) AddMessage(_ context.Context, msg *entities.SupportMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tickets[msg.TicketID]; !ok {
		return fmt.Errorf("ticket not found: %s", msg.TicketID)
	}
	r.messages[msg.TicketID] = append(r.messages[msg.TicketID], *msg)
	r.tickets[msg.TicketID].UpdatedAt = msg.CreatedAt
	return nil
}

func (r *MemorySupportRepository) ListMessages(_ context.Context, ticketID string) ([]entities.SupportMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	msgs, ok := r.messages[ticketID]
	if !ok {
		return []entities.SupportMessage{}, nil
	}
	result := make([]entities.SupportMessage, len(msgs))
	copy(result, msgs)
	return result, nil
}
