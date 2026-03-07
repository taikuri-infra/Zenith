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

// MemoryProjectRepository is an in-memory ProjectRepository for testing and development.
type MemoryProjectRepository struct {
	mu       sync.RWMutex
	projects map[string]*entities.Project
}

// NewMemoryProjectRepository creates a new in-memory ProjectRepository.
func NewMemoryProjectRepository() *MemoryProjectRepository {
	return &MemoryProjectRepository{
		projects: make(map[string]*entities.Project),
	}
}

func (r *MemoryProjectRepository) CreateProject(_ context.Context, userID, name, slug, description string) (*entities.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate slug under same user
	for _, p := range r.projects {
		if p.UserID == userID && p.Slug == slug {
			return nil, fmt.Errorf("project slug '%s' already exists", slug)
		}
	}

	now := time.Now()
	p := &entities.Project{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Slug:        slug,
		Description: description,
		Timestamps:  entities.Timestamps{CreatedAt: now, UpdatedAt: now},
	}
	r.projects[p.ID] = p
	return p, nil
}

func (r *MemoryProjectRepository) GetProject(_ context.Context, id string) (*entities.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.projects[id]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	return p, nil
}

func (r *MemoryProjectRepository) ListProjectsByUser(_ context.Context, userID string) ([]entities.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.Project
	for _, p := range r.projects {
		if p.UserID == userID {
			result = append(result, *p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryProjectRepository) UpdateProject(_ context.Context, id string, name, description *string) (*entities.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.projects[id]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", id)
	}

	if name != nil {
		p.Name = *name
	}
	if description != nil {
		p.Description = *description
	}
	p.UpdatedAt = time.Now()
	return p, nil
}

func (r *MemoryProjectRepository) DeleteProject(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.projects[id]; !ok {
		return fmt.Errorf("project not found: %s", id)
	}
	delete(r.projects, id)
	return nil
}

func (r *MemoryProjectRepository) CountProjectsByUser(_ context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, p := range r.projects {
		if p.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *MemoryProjectRepository) GetDefaultProject(_ context.Context, userID string) (*entities.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Look for slug=default first
	for _, p := range r.projects {
		if p.UserID == userID && p.Slug == "default" {
			return p, nil
		}
	}
	// Fallback: return the first project
	for _, p := range r.projects {
		if p.UserID == userID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no projects found for user: %s", userID)
}
