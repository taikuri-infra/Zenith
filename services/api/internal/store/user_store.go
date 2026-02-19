package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// StoredUser wraps a User with the password hash.
type StoredUser struct {
	models.User
	PasswordHash string
}

// UserStore is a thread-safe, in-memory user store.
type UserStore struct {
	users   map[string]*StoredUser // keyed by ID
	byEmail map[string]string     // email -> ID
	mu      sync.RWMutex
}

// NewUserStore returns an empty UserStore.
func NewUserStore() *UserStore {
	return &UserStore{
		users:   make(map[string]*StoredUser),
		byEmail: make(map[string]string),
	}
}

// Create adds a new user. Returns an error if the email is already taken.
func (s *UserStore) Create(email, password, name string, role models.Role) (*models.User, error) {
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
	user := &StoredUser{
		User: models.User{
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
func (s *UserStore) GetByEmail(email string) (*StoredUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return s.users[id], nil
}

// GetByID returns the stored user for the given ID.
func (s *UserStore) GetByID(id string) (*StoredUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// CheckPassword returns true if the password matches the stored hash.
func (s *UserStore) CheckPassword(user *StoredUser, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}

// Count returns the total number of users.
func (s *UserStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}
