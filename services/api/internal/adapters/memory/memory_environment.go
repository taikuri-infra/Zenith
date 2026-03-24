package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// MemoryEnvironmentRepository is an in-memory EnvironmentRepository for testing.
type MemoryEnvironmentRepository struct {
	mu   sync.RWMutex
	envs map[string]*entities.Environment
}

// NewMemoryEnvironmentRepository creates a new in-memory EnvironmentRepository.
func NewMemoryEnvironmentRepository() *MemoryEnvironmentRepository {
	return &MemoryEnvironmentRepository{
		envs: make(map[string]*entities.Environment),
	}
}

func (r *MemoryEnvironmentRepository) CreateEnvironment(_ context.Context, env *entities.Environment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range r.envs {
		if e.ProjectID == env.ProjectID && e.Name == env.Name {
			return fmt.Errorf("environment '%s' already exists in this project", env.Name)
		}
	}

	now := time.Now()
	env.CreatedAt = now
	env.UpdatedAt = now
	cp := *env
	r.envs[env.ID] = &cp
	return nil
}

func (r *MemoryEnvironmentRepository) GetEnvironment(_ context.Context, id string) (*entities.Environment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.envs[id]
	if !ok {
		return nil, fmt.Errorf("environment not found: %s", id)
	}
	return e, nil
}

func (r *MemoryEnvironmentRepository) GetEnvironmentByName(_ context.Context, projectID string, name entities.EnvironmentName) (*entities.Environment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, e := range r.envs {
		if e.ProjectID == projectID && e.Name == name {
			return e, nil
		}
	}
	return nil, fmt.Errorf("environment '%s' not found in project %s", name, projectID)
}

func (r *MemoryEnvironmentRepository) ListEnvironmentsByProject(_ context.Context, projectID string) ([]entities.Environment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.Environment
	for _, e := range r.envs {
		if e.ProjectID == projectID {
			result = append(result, *e)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDefault != result[j].IsDefault {
			return result[i].IsDefault
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryEnvironmentRepository) UpdateEnvironmentStatus(_ context.Context, id string, status entities.EnvironmentStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.envs[id]
	if !ok {
		return fmt.Errorf("environment not found: %s", id)
	}
	e.Status = status
	e.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryEnvironmentRepository) DeleteEnvironment(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.envs[id]; !ok {
		return fmt.Errorf("environment not found: %s", id)
	}
	delete(r.envs, id)
	return nil
}
