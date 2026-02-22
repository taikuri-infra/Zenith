package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

type MemoryRoleRepository struct {
	mu          sync.RWMutex
	roles       map[string]*entities.CustomRole
	assignments map[string]*entities.RoleAssignment
}

func NewMemoryRoleRepository() *MemoryRoleRepository {
	return &MemoryRoleRepository{
		roles:       make(map[string]*entities.CustomRole),
		assignments: make(map[string]*entities.RoleAssignment),
	}
}

func (r *MemoryRoleRepository) CreateRole(ctx context.Context, userID, name, description string, permissions []entities.Permission) (*entities.CustomRole, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	role := &entities.CustomRole{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Description: description,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	r.roles[role.ID] = role
	return role, nil
}

func (r *MemoryRoleRepository) GetRole(ctx context.Context, id string) (*entities.CustomRole, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	role, ok := r.roles[id]
	if !ok {
		return nil, fmt.Errorf("role not found")
	}
	return role, nil
}

func (r *MemoryRoleRepository) ListRolesByUser(ctx context.Context, userID string) ([]entities.CustomRole, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.CustomRole
	for _, role := range r.roles {
		if role.UserID == userID {
			result = append(result, *role)
		}
	}
	return result, nil
}

func (r *MemoryRoleRepository) UpdateRole(ctx context.Context, id string, name, description *string, permissions []entities.Permission) (*entities.CustomRole, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	role, ok := r.roles[id]
	if !ok {
		return nil, fmt.Errorf("role not found")
	}
	if name != nil {
		role.Name = *name
	}
	if description != nil {
		role.Description = *description
	}
	if permissions != nil {
		role.Permissions = permissions
	}
	role.UpdatedAt = time.Now()
	return role, nil
}

func (r *MemoryRoleRepository) DeleteRole(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.roles[id]; !ok {
		return fmt.Errorf("role not found")
	}
	delete(r.roles, id)
	// Also remove assignments for this role
	for aID, a := range r.assignments {
		if a.RoleID == id {
			delete(r.assignments, aID)
		}
	}
	return nil
}

func (r *MemoryRoleRepository) AssignRole(ctx context.Context, roleID, memberID, assignedBy string) (*entities.RoleAssignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.roles[roleID]; !ok {
		return nil, fmt.Errorf("role not found")
	}
	a := &entities.RoleAssignment{
		ID:         uuid.New().String(),
		RoleID:     roleID,
		MemberID:   memberID,
		AssignedBy: assignedBy,
		CreatedAt:  time.Now(),
	}
	r.assignments[a.ID] = a
	return a, nil
}

func (r *MemoryRoleRepository) ListAssignmentsByRole(ctx context.Context, roleID string) ([]entities.RoleAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.RoleAssignment
	for _, a := range r.assignments {
		if a.RoleID == roleID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (r *MemoryRoleRepository) RemoveAssignment(ctx context.Context, assignmentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.assignments[assignmentID]; !ok {
		return fmt.Errorf("assignment not found")
	}
	delete(r.assignments, assignmentID)
	return nil
}
