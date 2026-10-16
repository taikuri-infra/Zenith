package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemoryDomainRepository is an in-memory implementation of DomainRepository.
type MemoryDomainRepository struct {
	mu      sync.RWMutex
	domains map[string]*entities.CustomDomain // id -> domain
}

// NewMemoryDomainRepository creates a new MemoryDomainRepository.
func NewMemoryDomainRepository() *MemoryDomainRepository {
	return &MemoryDomainRepository{
		domains: make(map[string]*entities.CustomDomain),
	}
}

func (r *MemoryDomainRepository) AddDomain(_ context.Context, appID, userID, domain string) (*entities.CustomDomain, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate domain
	for _, d := range r.domains {
		if d.Domain == domain {
			return nil, fmt.Errorf("domain %q already in use", domain)
		}
	}

	now := time.Now()
	d := &entities.CustomDomain{
		ID:     uuid.New().String(),
		AppID:  appID,
		UserID: userID,
		Domain: domain,
		Status: entities.DomainStatusActive,
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	r.domains[d.ID] = d
	return d, nil
}

func (r *MemoryDomainRepository) GetDomain(_ context.Context, id string) (*entities.CustomDomain, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, ok := r.domains[id]
	if !ok {
		return nil, fmt.Errorf("domain not found: %s", id)
	}
	return d, nil
}

func (r *MemoryDomainRepository) ListDomainsByApp(_ context.Context, appID string) ([]entities.CustomDomain, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.CustomDomain
	for _, d := range r.domains {
		if d.AppID == appID {
			result = append(result, *d)
		}
	}
	return result, nil
}

func (r *MemoryDomainRepository) ListDomainsByUser(_ context.Context, userID string) ([]entities.CustomDomain, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.CustomDomain
	for _, d := range r.domains {
		if d.UserID == userID {
			result = append(result, *d)
		}
	}
	return result, nil
}

func (r *MemoryDomainRepository) UpdateDomainStatus(_ context.Context, id string, status entities.DomainStatus, tlsReady bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.domains[id]
	if !ok {
		return fmt.Errorf("domain not found: %s", id)
	}
	d.Status = status
	d.TLSReady = tlsReady
	d.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryDomainRepository) DeleteDomain(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.domains[id]; !ok {
		return fmt.Errorf("domain not found: %s", id)
	}
	delete(r.domains, id)
	return nil
}

func (r *MemoryDomainRepository) UpdateDomainCheck(_ context.Context, id string, status entities.DomainStatus, tlsReady bool, message string, checkedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.domains[id]
	if !ok {
		return fmt.Errorf("domain not found: %s", id)
	}
	d.Status, d.TLSReady, d.StatusMessage, d.LastCheckedAt = status, tlsReady, message, nil
	if !checkedAt.IsZero() {
		at := checkedAt
		d.LastCheckedAt = &at
	}
	d.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryDomainRepository) ListAllDomains(_ context.Context) ([]entities.CustomDomain, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]entities.CustomDomain, 0, len(r.domains))
	for _, d := range r.domains {
		out = append(out, *d)
	}
	return out, nil
}
