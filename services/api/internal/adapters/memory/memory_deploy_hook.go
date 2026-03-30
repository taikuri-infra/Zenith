package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemoryDeployHookRepository is an in-memory DeployHookRepository.
type MemoryDeployHookRepository struct {
	mu    sync.RWMutex
	hooks map[string]*entities.DeployHook
}

func NewMemoryDeployHookRepository() *MemoryDeployHookRepository {
	return &MemoryDeployHookRepository{hooks: make(map[string]*entities.DeployHook)}
}

func (r *MemoryDeployHookRepository) CreateHook(_ context.Context, hook *entities.DeployHook) (*entities.DeployHook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	hook.ID = uuid.New().String()
	now := time.Now()
	hook.CreatedAt = now
	hook.UpdatedAt = now
	r.hooks[hook.ID] = hook
	return hook, nil
}

func (r *MemoryDeployHookRepository) GetHook(_ context.Context, id string) (*entities.DeployHook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.hooks[id]
	if !ok {
		return nil, fmt.Errorf("deploy hook not found")
	}
	return h, nil
}

func (r *MemoryDeployHookRepository) ListHooksByApp(_ context.Context, appID string) ([]entities.DeployHook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.DeployHook
	for _, h := range r.hooks {
		if h.AppID == appID {
			result = append(result, *h)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Order != result[j].Order {
			return result[i].Order < result[j].Order
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryDeployHookRepository) UpdateHook(_ context.Context, id string, name *string, url *string, command *string, order *int, active *bool) (*entities.DeployHook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.hooks[id]
	if !ok {
		return nil, fmt.Errorf("deploy hook not found")
	}
	if name != nil {
		h.Name = *name
	}
	if url != nil {
		h.URL = *url
	}
	if command != nil {
		h.Command = *command
	}
	if order != nil {
		h.Order = *order
	}
	if active != nil {
		h.Active = *active
	}
	h.UpdatedAt = time.Now()
	return h, nil
}

func (r *MemoryDeployHookRepository) DeleteHook(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.hooks[id]; !ok {
		return fmt.Errorf("deploy hook not found")
	}
	delete(r.hooks, id)
	return nil
}

func (r *MemoryDeployHookRepository) CountHooksByApp(_ context.Context, appID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, h := range r.hooks {
		if h.AppID == appID {
			count++
		}
	}
	return count, nil
}
