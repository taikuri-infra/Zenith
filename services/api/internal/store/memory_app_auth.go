package store

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// MemoryAppAuthRepository is an in-memory implementation of AppAuthRepository.
type MemoryAppAuthRepository struct {
	mu       sync.RWMutex
	configs  map[string]*entities.AppAuthConfig // appID -> config
	users    map[string]map[string]*appUserEntry // appID -> userID -> entry
	emails   map[string]map[string]string        // appID -> email -> userID
}

type appUserEntry struct {
	user         entities.AppUser
	passwordHash string
}

// NewMemoryAppAuthRepository creates a new in-memory app auth repository.
func NewMemoryAppAuthRepository() *MemoryAppAuthRepository {
	return &MemoryAppAuthRepository{
		configs: make(map[string]*entities.AppAuthConfig),
		users:   make(map[string]map[string]*appUserEntry),
		emails:  make(map[string]map[string]string),
	}
}

func (r *MemoryAppAuthRepository) EnableAuth(_ context.Context, appID string, maxUsers int) (*entities.AppAuthConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cfg, ok := r.configs[appID]; ok {
		cfg.Enabled = true
		cfg.MaxUsers = maxUsers
		cfg.UpdatedAt = time.Now()
		return cfg, nil
	}

	secret := make([]byte, 32)
	rand.Read(secret)

	cfg := &entities.AppAuthConfig{
		AppID:    appID,
		Enabled:  true,
		MaxUsers: maxUsers,
		JWTSecret: fmt.Sprintf("%x", secret),
		Timestamps: entities.Timestamps{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	r.configs[appID] = cfg
	r.users[appID] = make(map[string]*appUserEntry)
	r.emails[appID] = make(map[string]string)
	return cfg, nil
}

func (r *MemoryAppAuthRepository) DisableAuth(_ context.Context, appID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, ok := r.configs[appID]
	if !ok {
		return fmt.Errorf("auth not configured for app %s", appID)
	}
	cfg.Enabled = false
	cfg.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryAppAuthRepository) GetAuthConfig(_ context.Context, appID string) (*entities.AppAuthConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg, ok := r.configs[appID]
	if !ok {
		return nil, fmt.Errorf("auth not configured for app %s", appID)
	}
	return cfg, nil
}

func (r *MemoryAppAuthRepository) CreateAppUser(_ context.Context, appID, email, password, name string) (*entities.AppUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, ok := r.configs[appID]
	if !ok || !cfg.Enabled {
		return nil, fmt.Errorf("auth not enabled for app %s", appID)
	}

	// Check email uniqueness within app
	if _, exists := r.emails[appID][email]; exists {
		return nil, fmt.Errorf("email already registered")
	}

	// Check user limit
	if len(r.users[appID]) >= cfg.MaxUsers {
		return nil, fmt.Errorf("user limit reached (%d)", cfg.MaxUsers)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	id := uuid.New().String()
	user := entities.AppUser{
		ID:       id,
		AppID:    appID,
		Email:    email,
		Name:     name,
		Verified: false,
		Timestamps: entities.Timestamps{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	r.users[appID][id] = &appUserEntry{
		user:         user,
		passwordHash: string(hash),
	}
	r.emails[appID][email] = id

	return &user, nil
}

func (r *MemoryAppAuthRepository) GetAppUserByEmail(_ context.Context, appID, email string) (*entities.AppUser, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, ok := r.emails[appID][email]
	if !ok {
		return nil, "", fmt.Errorf("user not found")
	}

	entry := r.users[appID][userID]
	return &entry.user, entry.passwordHash, nil
}

func (r *MemoryAppAuthRepository) CountAppUsers(_ context.Context, appID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.users[appID]), nil
}

func (r *MemoryAppAuthRepository) ListAppUsers(_ context.Context, appID string, limit, offset int) ([]entities.AppUser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	appUsers := r.users[appID]
	if appUsers == nil {
		return nil, nil
	}

	// Collect all users
	all := make([]entities.AppUser, 0, len(appUsers))
	for _, entry := range appUsers {
		all = append(all, entry.user)
	}

	// Apply offset/limit
	if offset >= len(all) {
		return nil, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (r *MemoryAppAuthRepository) DeleteAppUser(_ context.Context, appID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.users[appID][userID]
	if !ok {
		return fmt.Errorf("user not found")
	}

	delete(r.emails[appID], entry.user.Email)
	delete(r.users[appID], userID)
	return nil
}
