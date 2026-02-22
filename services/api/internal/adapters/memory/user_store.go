package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Compile-time interface check.
var _ ports.UserRepository = (*MemoryUserRepository)(nil)

// MemoryUserRepository is a thread-safe, in-memory user store.
type MemoryUserRepository struct {
	users   map[string]*ports.StoredUser // keyed by ID
	byEmail map[string]string           // email -> ID
	mu      sync.RWMutex
}

// NewMemoryUserRepository returns an empty in-memory user store.
func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users:   make(map[string]*ports.StoredUser),
		byEmail: make(map[string]string),
	}
}

// Create adds a new user. Returns an error if the email is already taken.
func (s *MemoryUserRepository) Create(_ context.Context, email, password, name string, role entities.Role) (*entities.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[email]; exists {
		return nil, fmt.Errorf("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &ports.StoredUser{
		User: entities.User{
			ID:        uuid.New().String(),
			Email:     email,
			Name:      name,
			Role:      role,
			CreatedAt: now,
			UpdatedAt: now,
		},
		PasswordHash: string(hash),
	}

	s.users[user.ID] = user
	s.byEmail[email] = user.ID
	return &user.User, nil
}

// GetByEmail returns the stored user for the given email.
func (s *MemoryUserRepository) GetByEmail(_ context.Context, email string) (*ports.StoredUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return s.users[id], nil
}

// GetByID returns the stored user for the given ID.
func (s *MemoryUserRepository) GetByID(_ context.Context, id string) (*ports.StoredUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// CheckPassword returns true if the password matches the stored hash.
func (s *MemoryUserRepository) CheckPassword(user *ports.StoredUser, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}

// Count returns the total number of users.
func (s *MemoryUserRepository) Count(_ context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users), nil
}
