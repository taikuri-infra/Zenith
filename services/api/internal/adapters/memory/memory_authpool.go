package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// MemoryAuthPoolRepository is an in-memory implementation of AuthPoolRepository.
type MemoryAuthPoolRepository struct {
	mu    sync.RWMutex
	pools map[string]*entities.AuthPool
}

// NewMemoryAuthPoolRepository creates a new MemoryAuthPoolRepository.
func NewMemoryAuthPoolRepository() *MemoryAuthPoolRepository {
	return &MemoryAuthPoolRepository{
		pools: make(map[string]*entities.AuthPool),
	}
}

func (r *MemoryAuthPoolRepository) CreatePool(_ context.Context, id, userID, projectID, name, realmName, clientID, clientSecret, issuerURL string, maxUsers int) (*entities.AuthPool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check unique name per user
	for _, p := range r.pools {
		if p.UserID == userID && p.Name == name {
			return nil, fmt.Errorf("auth pool %q already exists", name)
		}
	}

	now := time.Now()
	pool := &entities.AuthPool{
		ID: id, UserID: userID, ProjectID: projectID, Name: name,
		RealmName: realmName, ClientID: clientID, ClientSecret: clientSecret,
		IssuerURL: issuerURL, Status: entities.AuthPoolStatusProvisioning,
		MaxUsers: maxUsers,
		Timestamps: entities.Timestamps{CreatedAt: now, UpdatedAt: now},
	}
	r.pools[id] = pool
	return pool, nil
}

func (r *MemoryAuthPoolRepository) GetPool(_ context.Context, id string) (*entities.AuthPool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.pools[id]
	if !ok {
		return nil, fmt.Errorf("auth pool not found: %s", id)
	}
	return p, nil
}

func (r *MemoryAuthPoolRepository) ListPoolsByUser(_ context.Context, userID string) ([]entities.AuthPool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.AuthPool
	for _, p := range r.pools {
		if p.UserID == userID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (r *MemoryAuthPoolRepository) DeletePool(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.pools[id]; !ok {
		return fmt.Errorf("auth pool not found: %s", id)
	}
	delete(r.pools, id)
	return nil
}

func (r *MemoryAuthPoolRepository) UpdatePoolStatus(_ context.Context, id string, status entities.AuthPoolStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.pools[id]
	if !ok {
		return fmt.Errorf("auth pool not found: %s", id)
	}
	p.Status = status
	p.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryAuthPoolRepository) UpdatePoolUserCount(_ context.Context, id string, delta int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.pools[id]
	if !ok {
		return fmt.Errorf("auth pool not found: %s", id)
	}
	p.UserCount += delta
	p.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryAuthPoolRepository) CountPoolsByUser(_ context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, p := range r.pools {
		if p.UserID == userID {
			count++
		}
	}
	return count, nil
}
