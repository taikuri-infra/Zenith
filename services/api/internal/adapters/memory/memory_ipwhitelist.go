package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

type MemoryIPWhitelistRepository struct {
	mu      sync.RWMutex
	entries map[string]*entities.IPWhitelistEntry
}

func NewMemoryIPWhitelistRepository() *MemoryIPWhitelistRepository {
	return &MemoryIPWhitelistRepository{entries: make(map[string]*entities.IPWhitelistEntry)}
}

func (r *MemoryIPWhitelistRepository) AddEntry(ctx context.Context, userID, cidr, description string) (*entities.IPWhitelistEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := &entities.IPWhitelistEntry{
		ID:          uuid.New().String(),
		UserID:      userID,
		CIDR:        cidr,
		Description: description,
		CreatedAt:   time.Now(),
	}
	r.entries[entry.ID] = entry
	return entry, nil
}

func (r *MemoryIPWhitelistRepository) GetEntry(ctx context.Context, id string) (*entities.IPWhitelistEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[id]
	if !ok {
		return nil, fmt.Errorf("entry not found")
	}
	return entry, nil
}

func (r *MemoryIPWhitelistRepository) ListByUser(ctx context.Context, userID string) ([]entities.IPWhitelistEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.IPWhitelistEntry
	for _, e := range r.entries {
		if e.UserID == userID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (r *MemoryIPWhitelistRepository) DeleteEntry(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[id]; !ok {
		return fmt.Errorf("entry not found")
	}
	delete(r.entries, id)
	return nil
}

func (r *MemoryIPWhitelistRepository) IsIPAllowed(ctx context.Context, userID, ip string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// If no entries exist for user, allow all (whitelist not configured)
	hasEntries := false
	for _, e := range r.entries {
		if e.UserID == userID {
			hasEntries = true
			break
		}
	}
	if !hasEntries {
		return true, nil
	}
	// Check if IP matches any CIDR range (simplified — just exact match for memory store)
	for _, e := range r.entries {
		if e.UserID == userID && e.CIDR == ip+"/32" {
			return true, nil
		}
		if e.UserID == userID && e.CIDR == ip {
			return true, nil
		}
	}
	return false, nil
}
