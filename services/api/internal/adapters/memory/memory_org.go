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

// MemoryOrgRepository is an in-memory OrgRepository for tests and development.
type MemoryOrgRepository struct {
	mu   sync.RWMutex
	orgs map[string]*entities.Org
}

func NewMemoryOrgRepository() *MemoryOrgRepository {
	return &MemoryOrgRepository{orgs: map[string]*entities.Org{}}
}

func (r *MemoryOrgRepository) create(projectID, name string, isDefault bool) (*entities.Org, error) {
	slug := entities.OrgSlug(name)
	if slug == "" {
		return nil, fmt.Errorf("org name must contain letters or digits")
	}
	for _, o := range r.orgs {
		if o.ProjectID == projectID && o.Slug == slug {
			return nil, fmt.Errorf("org '%s' already exists", name)
		}
	}
	now := time.Now()
	o := &entities.Org{
		ID: uuid.New().String(), ProjectID: projectID, Name: name, Slug: slug,
		Namespace: entities.OrgNamespace(projectID, slug), IsDefault: isDefault,
		Timestamps: entities.Timestamps{CreatedAt: now, UpdatedAt: now},
	}
	r.orgs[o.ID] = o
	return o, nil
}

func (r *MemoryOrgRepository) CreateOrg(_ context.Context, projectID, name string) (*entities.Org, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.create(projectID, name, false)
}

func (r *MemoryOrgRepository) GetOrg(_ context.Context, id string) (*entities.Org, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orgs[id]
	if !ok {
		return nil, fmt.Errorf("org not found")
	}
	c := *o
	return &c, nil
}

func (r *MemoryOrgRepository) ListOrgsByProject(_ context.Context, projectID string) ([]entities.Org, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []entities.Org{}
	for _, o := range r.orgs {
		if o.ProjectID == projectID {
			out = append(out, *o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *MemoryOrgRepository) RenameOrg(_ context.Context, id, name string) (*entities.Org, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.orgs[id]
	if !ok {
		return nil, fmt.Errorf("org not found")
	}
	o.Name, o.UpdatedAt = name, time.Now()
	c := *o
	return &c, nil
}

func (r *MemoryOrgRepository) DeleteOrg(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.orgs[id]; !ok {
		return fmt.Errorf("org not found")
	}
	delete(r.orgs, id)
	return nil
}

func (r *MemoryOrgRepository) GetDefaultOrg(_ context.Context, projectID string) (*entities.Org, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.orgs {
		if o.ProjectID == projectID && o.IsDefault {
			c := *o
			return &c, nil
		}
	}
	o, err := r.create(projectID, "Default", true)
	if err != nil {
		return nil, err
	}
	c := *o
	return &c, nil
}
