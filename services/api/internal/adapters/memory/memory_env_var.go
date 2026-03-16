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

// MemoryEnvVarRepository is an in-memory EnvVarRepository for testing.
type MemoryEnvVarRepository struct {
	mu   sync.RWMutex
	vars map[string]*entities.AppEnvVar // keyed by ID
}

// NewMemoryEnvVarRepository creates a new in-memory EnvVarRepository.
func NewMemoryEnvVarRepository() *MemoryEnvVarRepository {
	return &MemoryEnvVarRepository{
		vars: make(map[string]*entities.AppEnvVar),
	}
}

func (r *MemoryEnvVarRepository) SetEnvVar(_ context.Context, envVar *entities.AppEnvVar) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if envVar.ID == "" {
		envVar.ID = uuid.New().String()
	}
	now := time.Now()

	// Check for existing var with same app_id + key (upsert)
	for id, v := range r.vars {
		if v.AppID == envVar.AppID && v.Key == envVar.Key {
			v.Value = envVar.Value
			v.IsSecret = envVar.IsSecret
			v.Source = envVar.Source
			v.SourceID = envVar.SourceID
			v.UpdatedAt = now
			envVar.ID = id
			return nil
		}
	}

	envVar.CreatedAt = now
	envVar.UpdatedAt = now
	cp := *envVar
	r.vars[envVar.ID] = &cp
	return nil
}

func (r *MemoryEnvVarRepository) GetEnvVars(_ context.Context, appID string) ([]entities.AppEnvVar, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.AppEnvVar
	for _, v := range r.vars {
		if v.AppID == appID {
			result = append(result, *v)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})
	return result, nil
}

func (r *MemoryEnvVarRepository) DeleteEnvVar(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.vars[id]; !ok {
		return fmt.Errorf("env var not found: %s", id)
	}
	delete(r.vars, id)
	return nil
}

func (r *MemoryEnvVarRepository) BulkSetEnvVars(ctx context.Context, appID string, vars []entities.AppEnvVar) error {
	for i := range vars {
		vars[i].AppID = appID
		if err := r.SetEnvVar(ctx, &vars[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *MemoryEnvVarRepository) DeleteEnvVarsBySource(_ context.Context, appID string, source entities.EnvVarSource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, v := range r.vars {
		if v.AppID == appID && v.Source == source {
			delete(r.vars, id)
		}
	}
	return nil
}
