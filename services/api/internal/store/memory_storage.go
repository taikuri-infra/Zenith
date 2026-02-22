package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemoryStorageRepository is an in-memory implementation of StorageRepository.
type MemoryStorageRepository struct {
	mu      sync.RWMutex
	buckets map[string]*entities.UserBucket
}

// NewMemoryStorageRepository creates a new in-memory storage repository.
func NewMemoryStorageRepository() *MemoryStorageRepository {
	return &MemoryStorageRepository{
		buckets: make(map[string]*entities.UserBucket),
	}
}

func (r *MemoryStorageRepository) CreateBucket(_ context.Context, appID, userID string, input *dto.CreateBucketInput) (*entities.UserBucket, error) {
	if appID == "" || userID == "" {
		return nil, fmt.Errorf("app_id and user_id are required")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Check for duplicate name within app
	r.mu.RLock()
	for _, b := range r.buckets {
		if b.AppID == appID && b.Name == input.Name {
			r.mu.RUnlock()
			return nil, fmt.Errorf("bucket %q already exists for this app", input.Name)
		}
	}
	r.mu.RUnlock()

	access := input.Access
	if access == "" {
		access = entities.BucketAccessPrivate
	}

	id := uuid.New().String()
	bucket := &entities.UserBucket{
		ID:        id,
		AppID:     appID,
		UserID:    userID,
		Name:      input.Name,
		Access:    access,
		Region:    "fsn1",
		SizeMB:    0,
		MaxSizeMB: 1024, // free tier: 1GB
		Objects:   0,
		Status:    entities.BucketStatusActive,
		Endpoint:  fmt.Sprintf("https://%s.s3.zenith.local", input.Name),
		Timestamps: entities.Timestamps{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	r.mu.Lock()
	r.buckets[id] = bucket
	r.mu.Unlock()

	return bucket, nil
}

func (r *MemoryStorageRepository) GetBucket(_ context.Context, id string) (*entities.UserBucket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.buckets[id]
	if !ok {
		return nil, fmt.Errorf("bucket not found: %s", id)
	}
	return b, nil
}

func (r *MemoryStorageRepository) ListBucketsByApp(_ context.Context, appID string) ([]entities.UserBucket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.UserBucket
	for _, b := range r.buckets {
		if b.AppID == appID {
			result = append(result, *b)
		}
	}
	return result, nil
}

func (r *MemoryStorageRepository) ListBucketsByUser(_ context.Context, userID string) ([]entities.UserBucket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.UserBucket
	for _, b := range r.buckets {
		if b.UserID == userID {
			result = append(result, *b)
		}
	}
	return result, nil
}

func (r *MemoryStorageRepository) DeleteBucket(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.buckets[id]; !ok {
		return fmt.Errorf("bucket not found: %s", id)
	}
	delete(r.buckets, id)
	return nil
}

func (r *MemoryStorageRepository) CountBucketsByUser(_ context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, b := range r.buckets {
		if b.UserID == userID {
			count++
		}
	}
	return count, nil
}
