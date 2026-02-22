package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

type MemoryPreviewRepository struct {
	mu       sync.RWMutex
	previews map[string]*entities.PreviewDeployment
}

func NewMemoryPreviewRepository() *MemoryPreviewRepository {
	return &MemoryPreviewRepository{previews: make(map[string]*entities.PreviewDeployment)}
}

func (r *MemoryPreviewRepository) CreatePreview(ctx context.Context, appID string, prNumber int, branch, gitSHA, url string) (*entities.PreviewDeployment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := &entities.PreviewDeployment{
		ID:        uuid.New().String(),
		AppID:     appID,
		PRNumber:  prNumber,
		Branch:    branch,
		URL:       url,
		Status:    "building",
		GitSHA:    gitSHA,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.previews[p.ID] = p

	// Simulate async build completion
	go func() {
		time.Sleep(3 * time.Second)
		r.mu.Lock()
		defer r.mu.Unlock()
		if p, ok := r.previews[p.ID]; ok {
			p.Status = "running"
			p.UpdatedAt = time.Now()
		}
	}()

	return p, nil
}

func (r *MemoryPreviewRepository) GetPreview(ctx context.Context, id string) (*entities.PreviewDeployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.previews[id]
	if !ok {
		return nil, fmt.Errorf("preview not found")
	}
	return p, nil
}

func (r *MemoryPreviewRepository) ListPreviewsByApp(ctx context.Context, appID string) ([]entities.PreviewDeployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.PreviewDeployment
	for _, p := range r.previews {
		if p.AppID == appID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (r *MemoryPreviewRepository) DeletePreview(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.previews[id]; !ok {
		return fmt.Errorf("preview not found")
	}
	delete(r.previews, id)
	return nil
}
