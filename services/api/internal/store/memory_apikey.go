package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemoryAPIKeyRepository is an in-memory implementation of APIKeyRepository.
type MemoryAPIKeyRepository struct {
	mu     sync.RWMutex
	keys   map[string]*entities.APIKey // id -> key
	byHash map[string]string           // keyHash -> id
}

// NewMemoryAPIKeyRepository creates a new MemoryAPIKeyRepository.
func NewMemoryAPIKeyRepository() *MemoryAPIKeyRepository {
	return &MemoryAPIKeyRepository{
		keys:   make(map[string]*entities.APIKey),
		byHash: make(map[string]string),
	}
}

func generateAPIKey() (key string, prefix string, hash string) {
	raw := make([]byte, 32)
	rand.Read(raw)
	key = "zk_" + hex.EncodeToString(raw)
	prefix = key[:10]
	h := sha256.Sum256([]byte(key))
	hash = hex.EncodeToString(h[:])
	return
}

func (r *MemoryAPIKeyRepository) CreateAPIKey(_ context.Context, userID, name string, scopes []string) (*entities.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key, prefix, hash := generateAPIKey()

	if scopes == nil {
		scopes = []string{"read"}
	}

	apiKey := &entities.APIKey{
		ID:        uuid.New().String(),
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   hash,
		Key:       key, // Only returned on creation
		Scopes:    scopes,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	r.keys[apiKey.ID] = apiKey
	r.byHash[hash] = apiKey.ID
	return apiKey, nil
}

func (r *MemoryAPIKeyRepository) GetAPIKey(_ context.Context, id string) (*entities.APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key, ok := r.keys[id]
	if !ok {
		return nil, fmt.Errorf("API key not found: %s", id)
	}
	// Don't return the raw key on get
	result := *key
	result.Key = ""
	return &result, nil
}

func (r *MemoryAPIKeyRepository) GetAPIKeyByHash(_ context.Context, keyHash string) (*entities.APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byHash[keyHash]
	if !ok {
		return nil, fmt.Errorf("API key not found")
	}
	key := r.keys[id]
	result := *key
	result.Key = ""
	return &result, nil
}

func (r *MemoryAPIKeyRepository) ListAPIKeysByUser(_ context.Context, userID string) ([]entities.APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.APIKey
	for _, k := range r.keys {
		if k.UserID == userID {
			copy := *k
			copy.Key = "" // Never return raw key in list
			result = append(result, copy)
		}
	}
	return result, nil
}

func (r *MemoryAPIKeyRepository) DeleteAPIKey(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key, ok := r.keys[id]
	if !ok {
		return fmt.Errorf("API key not found: %s", id)
	}
	delete(r.byHash, key.KeyHash)
	delete(r.keys, id)
	return nil
}

func (r *MemoryAPIKeyRepository) UpdateLastUsed(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key, ok := r.keys[id]
	if !ok {
		return fmt.Errorf("API key not found: %s", id)
	}
	now := time.Now()
	key.LastUsedAt = &now
	return nil
}

func (r *MemoryAPIKeyRepository) CountAPIKeysByUser(_ context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, k := range r.keys {
		if k.UserID == userID {
			count++
		}
	}
	return count, nil
}
