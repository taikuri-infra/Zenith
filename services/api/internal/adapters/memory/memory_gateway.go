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

// MemoryGatewayRepository is an in-memory GatewayRepository for dev/testing.
type MemoryGatewayRepository struct {
	mu       sync.RWMutex
	gateways map[string]*entities.Gateway
	routes   map[string]*entities.GatewayRoute
}

// NewMemoryGatewayRepository creates a new in-memory GatewayRepository.
func NewMemoryGatewayRepository() *MemoryGatewayRepository {
	return &MemoryGatewayRepository{
		gateways: make(map[string]*entities.Gateway),
		routes:   make(map[string]*entities.GatewayRoute),
	}
}

// --- Gateway CRUD ---

func (r *MemoryGatewayRepository) CreateGateway(_ context.Context, userID, name, slug string) (*entities.Gateway, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, g := range r.gateways {
		if g.Slug == slug {
			return nil, fmt.Errorf("gateway slug '%s' already exists", slug)
		}
	}

	now := time.Now()
	gw := &entities.Gateway{
		ID:         uuid.New().String(),
		UserID:     userID,
		Name:       name,
		Slug:       slug,
		Status:     entities.GatewayStatusProvisioning,
		RouteCount: 0,
		Timestamps: entities.Timestamps{CreatedAt: now, UpdatedAt: now},
	}
	r.gateways[gw.ID] = gw
	return gw, nil
}

func (r *MemoryGatewayRepository) GetGateway(_ context.Context, id string) (*entities.Gateway, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gw, ok := r.gateways[id]
	if !ok {
		return nil, fmt.Errorf("gateway not found: %s", id)
	}
	return gw, nil
}

func (r *MemoryGatewayRepository) GetGatewayBySlug(_ context.Context, slug string) (*entities.Gateway, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, g := range r.gateways {
		if g.Slug == slug {
			return g, nil
		}
	}
	return nil, fmt.Errorf("gateway not found: %s", slug)
}

func (r *MemoryGatewayRepository) ListGatewaysByUser(_ context.Context, userID string) ([]entities.Gateway, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var gws []entities.Gateway
	for _, g := range r.gateways {
		if g.UserID == userID {
			gws = append(gws, *g)
		}
	}
	sort.Slice(gws, func(i, j int) bool {
		return gws[i].CreatedAt.After(gws[j].CreatedAt)
	})
	return gws, nil
}

func (r *MemoryGatewayRepository) UpdateGateway(_ context.Context, id, name string) (*entities.Gateway, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	gw, ok := r.gateways[id]
	if !ok {
		return nil, fmt.Errorf("gateway not found: %s", id)
	}
	gw.Name = name
	gw.UpdatedAt = time.Now()
	return gw, nil
}

func (r *MemoryGatewayRepository) DeleteGateway(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.gateways[id]; !ok {
		return fmt.Errorf("gateway not found: %s", id)
	}

	// Delete associated routes
	for rid, rt := range r.routes {
		if rt.GatewayID == id {
			delete(r.routes, rid)
		}
	}

	delete(r.gateways, id)
	return nil
}

func (r *MemoryGatewayRepository) CountGatewaysByUser(_ context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, g := range r.gateways {
		if g.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *MemoryGatewayRepository) UpdateGatewayStatus(_ context.Context, id string, status entities.GatewayStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	gw, ok := r.gateways[id]
	if !ok {
		return fmt.Errorf("gateway not found: %s", id)
	}
	gw.Status = status
	gw.UpdatedAt = time.Now()
	return nil
}

// --- Route CRUD ---

func (r *MemoryGatewayRepository) CreateRoute(_ context.Context, route *entities.GatewayRoute) (*entities.GatewayRoute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check name uniqueness within gateway
	for _, rt := range r.routes {
		if rt.GatewayID == route.GatewayID && rt.Name == route.Name {
			return nil, fmt.Errorf("route name '%s' already exists in this gateway", route.Name)
		}
	}

	now := time.Now()
	route.ID = uuid.New().String()
	route.CreatedAt = now
	route.UpdatedAt = now
	if route.Status == "" {
		route.Status = entities.GatewayRouteStatusActive
	}
	if route.Auth == "" {
		route.Auth = entities.GatewayRouteAuthNone
	}
	if route.Plugins == nil {
		route.Plugins = []entities.GatewayRoutePlugin{}
	}

	r.routes[route.ID] = route

	// Update route_count
	if gw, ok := r.gateways[route.GatewayID]; ok {
		count := 0
		for _, rt := range r.routes {
			if rt.GatewayID == route.GatewayID {
				count++
			}
		}
		gw.RouteCount = count
	}

	return route, nil
}

func (r *MemoryGatewayRepository) GetRoute(_ context.Context, id string) (*entities.GatewayRoute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rt, ok := r.routes[id]
	if !ok {
		return nil, fmt.Errorf("route not found: %s", id)
	}
	return rt, nil
}

func (r *MemoryGatewayRepository) ListRoutesByGateway(_ context.Context, gatewayID string) ([]entities.GatewayRoute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var routes []entities.GatewayRoute
	for _, rt := range r.routes {
		if rt.GatewayID == gatewayID {
			routes = append(routes, *rt)
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Priority != routes[j].Priority {
			return routes[i].Priority > routes[j].Priority
		}
		return routes[i].CreatedAt.Before(routes[j].CreatedAt)
	})
	return routes, nil
}

func (r *MemoryGatewayRepository) ListActiveRoutesByGateway(_ context.Context, gatewayID string) ([]entities.GatewayRoute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var routes []entities.GatewayRoute
	for _, rt := range r.routes {
		if rt.GatewayID == gatewayID && rt.Status == entities.GatewayRouteStatusActive {
			routes = append(routes, *rt)
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Priority != routes[j].Priority {
			return routes[i].Priority > routes[j].Priority
		}
		return routes[i].CreatedAt.Before(routes[j].CreatedAt)
	})
	return routes, nil
}

func (r *MemoryGatewayRepository) UpdateRoute(_ context.Context, route *entities.GatewayRoute) (*entities.GatewayRoute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.routes[route.ID]
	if !ok {
		return nil, fmt.Errorf("route not found: %s", route.ID)
	}

	route.CreatedAt = existing.CreatedAt
	route.UpdatedAt = time.Now()
	r.routes[route.ID] = route
	return route, nil
}

func (r *MemoryGatewayRepository) DeleteRoute(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rt, ok := r.routes[id]
	if !ok {
		return fmt.Errorf("route not found: %s", id)
	}
	gwID := rt.GatewayID
	delete(r.routes, id)

	// Update route_count
	if gw, ok := r.gateways[gwID]; ok {
		count := 0
		for _, rt := range r.routes {
			if rt.GatewayID == gwID {
				count++
			}
		}
		gw.RouteCount = count
	}

	return nil
}

func (r *MemoryGatewayRepository) CountRoutesByGateway(_ context.Context, gatewayID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, rt := range r.routes {
		if rt.GatewayID == gatewayID {
			count++
		}
	}
	return count, nil
}

func (r *MemoryGatewayRepository) CountRoutesByUser(_ context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gwIDs := make(map[string]bool)
	for _, g := range r.gateways {
		if g.UserID == userID {
			gwIDs[g.ID] = true
		}
	}

	count := 0
	for _, rt := range r.routes {
		if gwIDs[rt.GatewayID] {
			count++
		}
	}
	return count, nil
}

func (r *MemoryGatewayRepository) StopRoutesByApp(_ context.Context, appID string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	seen := make(map[string]bool)
	var gwIDs []string
	for _, rt := range r.routes {
		if rt.AppID == appID && rt.Status == entities.GatewayRouteStatusActive {
			rt.Status = entities.GatewayRouteStatusStopped
			rt.UpdatedAt = time.Now()
			if !seen[rt.GatewayID] {
				seen[rt.GatewayID] = true
				gwIDs = append(gwIDs, rt.GatewayID)
			}
		}
	}
	return gwIDs, nil
}
