package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

type MemorySSORepository struct {
	mu      sync.RWMutex
	configs map[string]*entities.SSOConfig // keyed by id
}

func NewMemorySSORepository() *MemorySSORepository {
	return &MemorySSORepository{configs: make(map[string]*entities.SSOConfig)}
}

func (r *MemorySSORepository) CreateConfig(ctx context.Context, userID string, provider entities.SSOProvider, config *entities.SSOConfig) (*entities.SSOConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	config.ID = uuid.New().String()
	config.UserID = userID
	config.Provider = provider
	config.Enabled = true
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()
	r.configs[config.ID] = config
	return config, nil
}

func (r *MemorySSORepository) GetConfig(ctx context.Context, id string) (*entities.SSOConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.configs[id]
	if !ok {
		return nil, fmt.Errorf("SSO config not found")
	}
	return c, nil
}

func (r *MemorySSORepository) ListConfigsByUser(ctx context.Context, userID string) ([]entities.SSOConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.SSOConfig
	for _, c := range r.configs {
		if c.UserID == userID {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (r *MemorySSORepository) DeleteConfig(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.configs[id]; !ok {
		return fmt.Errorf("SSO config not found")
	}
	delete(r.configs, id)
	return nil
}

func (r *MemorySSORepository) ToggleConfig(ctx context.Context, id string, enabled bool) (*entities.SSOConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.configs[id]
	if !ok {
		return nil, fmt.Errorf("SSO config not found")
	}
	c.Enabled = enabled
	c.UpdatedAt = time.Now()
	return c, nil
}
