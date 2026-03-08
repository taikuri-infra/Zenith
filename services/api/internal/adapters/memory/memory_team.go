package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

var _ ports.TeamMemberRepository = (*MemoryTeamMemberRepository)(nil)

// MemoryTeamMemberRepository is an in-memory implementation of TeamMemberRepository.
type MemoryTeamMemberRepository struct {
	mu      sync.RWMutex
	members map[string]*entities.TeamMember // id -> member
}

// NewMemoryTeamMemberRepository creates a new MemoryTeamMemberRepository.
func NewMemoryTeamMemberRepository() *MemoryTeamMemberRepository {
	return &MemoryTeamMemberRepository{
		members: make(map[string]*entities.TeamMember),
	}
}

func (r *MemoryTeamMemberRepository) CreateMember(_ context.Context, member *entities.TeamMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check unique constraint: account_id + email
	for _, m := range r.members {
		if m.AccountID == member.AccountID && m.Email == member.Email {
			return fmt.Errorf("member with email %s already exists for this account", member.Email)
		}
	}

	r.members[member.ID] = member
	return nil
}

func (r *MemoryTeamMemberRepository) GetMember(_ context.Context, id string) (*entities.TeamMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.members[id]
	if !ok {
		return nil, fmt.Errorf("team member not found: %s", id)
	}
	result := *m
	return &result, nil
}

func (r *MemoryTeamMemberRepository) GetMemberByEmail(_ context.Context, accountID, email string) (*entities.TeamMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.members {
		if m.AccountID == accountID && m.Email == email {
			result := *m
			return &result, nil
		}
	}
	return nil, fmt.Errorf("team member not found")
}

func (r *MemoryTeamMemberRepository) GetMemberByUserID(_ context.Context, userID string) (*entities.TeamMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.members {
		if m.UserID == userID && m.Status == entities.TeamMemberActive {
			result := *m
			return &result, nil
		}
	}
	return nil, fmt.Errorf("team member not found")
}

func (r *MemoryTeamMemberRepository) GetMemberByInviteHash(_ context.Context, hash string) (*entities.TeamMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.members {
		if m.InviteTokenHash == hash {
			result := *m
			return &result, nil
		}
	}
	return nil, fmt.Errorf("invite not found")
}

func (r *MemoryTeamMemberRepository) ListMembers(_ context.Context, accountID string) ([]entities.TeamMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.TeamMember
	for _, m := range r.members {
		if m.AccountID == accountID {
			result = append(result, *m)
		}
	}
	return result, nil
}

func (r *MemoryTeamMemberRepository) UpdateMember(_ context.Context, member *entities.TeamMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.members[member.ID]; !ok {
		return fmt.Errorf("team member not found: %s", member.ID)
	}
	r.members[member.ID] = member
	return nil
}

func (r *MemoryTeamMemberRepository) DeleteMember(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.members[id]; !ok {
		return fmt.Errorf("team member not found: %s", id)
	}
	delete(r.members, id)
	return nil
}

func (r *MemoryTeamMemberRepository) CountMembers(_ context.Context, accountID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, m := range r.members {
		if m.AccountID == accountID {
			count++
		}
	}
	return count, nil
}
