package memory

import (
	"context"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemoryAutoscaleRepository is an in-memory implementation of AutoscaleRepository.
type MemoryAutoscaleRepository struct {
	mu     sync.RWMutex
	nodes  map[int64]*entities.HetznerNode
	events []entities.AutoscaleEvent
	status *entities.AutoscalerStatus
}

// NewMemoryAutoscaleRepository creates a new in-memory autoscale repository.
func NewMemoryAutoscaleRepository() *MemoryAutoscaleRepository {
	return &MemoryAutoscaleRepository{
		nodes:  make(map[int64]*entities.HetznerNode),
		events: []entities.AutoscaleEvent{},
		status: &entities.AutoscalerStatus{},
	}
}

func (r *MemoryAutoscaleRepository) SaveNode(_ context.Context, node *entities.HetznerNode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodes[node.ServerID] = node
	return nil
}

func (r *MemoryAutoscaleRepository) DeleteNode(_ context.Context, serverID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.nodes, serverID)
	return nil
}

func (r *MemoryAutoscaleRepository) ListNodes(_ context.Context) ([]entities.HetznerNode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]entities.HetznerNode, 0, len(r.nodes))
	for _, n := range r.nodes {
		result = append(result, *n)
	}
	return result, nil
}

func (r *MemoryAutoscaleRepository) LogScaleEvent(_ context.Context, event *entities.AutoscaleEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	r.events = append([]entities.AutoscaleEvent{*event}, r.events...)
	// Keep only last 200 events
	if len(r.events) > 200 {
		r.events = r.events[:200]
	}
	return nil
}

func (r *MemoryAutoscaleRepository) ListScaleEvents(_ context.Context, limit int) ([]entities.AutoscaleEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > len(r.events) {
		limit = len(r.events)
	}
	result := make([]entities.AutoscaleEvent, limit)
	copy(result, r.events[:limit])
	return result, nil
}

func (r *MemoryAutoscaleRepository) GetStatus(_ context.Context) (*entities.AutoscalerStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s := *r.status
	return &s, nil
}

func (r *MemoryAutoscaleRepository) UpdateStatus(_ context.Context, status *entities.AutoscalerStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = status
	return nil
}
