package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// MemoryManagedServiceRepository is an in-memory ManagedServiceRepository for testing.
type MemoryManagedServiceRepository struct {
	mu       sync.RWMutex
	services map[string]*entities.ManagedService
}

// NewMemoryManagedServiceRepository creates a new in-memory ManagedServiceRepository.
func NewMemoryManagedServiceRepository() *MemoryManagedServiceRepository {
	return &MemoryManagedServiceRepository{
		services: make(map[string]*entities.ManagedService),
	}
}

func (r *MemoryManagedServiceRepository) CreateManagedService(_ context.Context, svc *entities.ManagedService) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range r.services {
		if s.ProjectID == svc.ProjectID && s.Name == svc.Name {
			return fmt.Errorf("managed service '%s' already exists in this project", svc.Name)
		}
	}

	now := time.Now()
	svc.CreatedAt = now
	svc.UpdatedAt = now
	cp := *svc
	r.services[svc.ID] = &cp
	return nil
}

func (r *MemoryManagedServiceRepository) GetManagedService(_ context.Context, id string) (*entities.ManagedService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.services[id]
	if !ok {
		return nil, fmt.Errorf("managed service not found: %s", id)
	}
	return s, nil
}

func (r *MemoryManagedServiceRepository) ListManagedServicesByProject(_ context.Context, projectID string) ([]entities.ManagedService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.ManagedService
	for _, s := range r.services {
		if s.ProjectID == projectID {
			result = append(result, *s)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryManagedServiceRepository) ListManagedServicesByUser(_ context.Context, userID string) ([]entities.ManagedService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.ManagedService
	for _, s := range r.services {
		if s.UserID == userID {
			result = append(result, *s)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryManagedServiceRepository) UpdateManagedServiceStatus(_ context.Context, id string, status entities.ManagedServiceStatus, statusMsg, connURL, host string, port int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.services[id]
	if !ok {
		return fmt.Errorf("managed service not found: %s", id)
	}
	s.Status = status
	s.StatusMessage = statusMsg
	s.ConnectionURL = connURL
	s.InternalHost = host
	s.Port = port
	s.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryManagedServiceRepository) DeleteManagedService(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.services[id]; !ok {
		return fmt.Errorf("managed service not found: %s", id)
	}
	delete(r.services, id)
	return nil
}

func (r *MemoryManagedServiceRepository) CountManagedServicesByUser(_ context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, s := range r.services {
		if s.UserID == userID {
			count++
		}
	}
	return count, nil
}
