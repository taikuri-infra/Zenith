package memory

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemoryDeployTokenRepository is an in-memory DeployTokenRepository for testing.
type MemoryDeployTokenRepository struct {
	mu     sync.RWMutex
	tokens map[string]*entities.DeployToken
}

// NewMemoryDeployTokenRepository creates a new in-memory DeployTokenRepository.
func NewMemoryDeployTokenRepository() *MemoryDeployTokenRepository {
	return &MemoryDeployTokenRepository{
		tokens: make(map[string]*entities.DeployToken),
	}
}

func (r *MemoryDeployTokenRepository) CreateDeployToken(_ context.Context, userID, projectID, name string, scopes []string, expiresAt *time.Time) (*entities.DeployToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	tokenIDBytes := make([]byte, 8)
	rand.Read(tokenIDBytes)
	tokenID := "znt_id_" + hex.EncodeToString(tokenIDBytes)

	secretBytes := make([]byte, 32)
	rand.Read(secretBytes)
	secret := "znt_sk_" + hex.EncodeToString(secretBytes)

	hash := sha256.Sum256([]byte(secret))

	dt := &entities.DeployToken{
		ID:          uuid.New().String(),
		UserID:      userID,
		ProjectID:   projectID,
		Name:        name,
		TokenID:     tokenID,
		TokenPrefix: secret[:15],
		TokenHash:   hex.EncodeToString(hash[:]),
		Secret:      secret,
		Scopes:      scopes,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}

	cp := *dt
	r.tokens[dt.ID] = &cp
	return dt, nil
}

func (r *MemoryDeployTokenRepository) GetDeployToken(_ context.Context, id string) (*entities.DeployToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dt, ok := r.tokens[id]
	if !ok {
		return nil, fmt.Errorf("deploy token not found: %s", id)
	}
	return dt, nil
}

func (r *MemoryDeployTokenRepository) GetDeployTokenByTokenID(_ context.Context, tokenID string) (*entities.DeployToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, dt := range r.tokens {
		if dt.TokenID == tokenID {
			return dt, nil
		}
	}
	return nil, fmt.Errorf("deploy token not found: %s", tokenID)
}

func (r *MemoryDeployTokenRepository) ListDeployTokensByProject(_ context.Context, projectID string) ([]entities.DeployToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.DeployToken
	for _, dt := range r.tokens {
		if dt.ProjectID == projectID && dt.RevokedAt == nil {
			result = append(result, *dt)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryDeployTokenRepository) RevokeDeployToken(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dt, ok := r.tokens[id]
	if !ok || dt.RevokedAt != nil {
		return fmt.Errorf("deploy token not found or already revoked: %s", id)
	}
	now := time.Now()
	dt.RevokedAt = &now
	return nil
}

func (r *MemoryDeployTokenRepository) RotateDeployToken(_ context.Context, id string) (*entities.DeployToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	dt, ok := r.tokens[id]
	if !ok {
		return nil, fmt.Errorf("deploy token not found: %s", id)
	}

	secretBytes := make([]byte, 32)
	rand.Read(secretBytes)
	newSecret := "znt_sk_" + hex.EncodeToString(secretBytes)
	hash := sha256.Sum256([]byte(newSecret))

	graceExpiry := time.Now().Add(24 * time.Hour)
	now := time.Now()

	dt.PreviousHash = dt.TokenHash
	dt.PreviousExpiresAt = &graceExpiry
	dt.TokenHash = hex.EncodeToString(hash[:])
	dt.TokenPrefix = newSecret[:15]
	dt.RotatedAt = &now
	dt.Secret = newSecret

	return dt, nil
}

func (r *MemoryDeployTokenRepository) UpdateLastUsed(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dt, ok := r.tokens[id]
	if !ok {
		return nil
	}
	now := time.Now()
	dt.LastUsedAt = &now
	return nil
}

func (r *MemoryDeployTokenRepository) VerifySecret(dt *entities.DeployToken, secret string) bool {
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:]) == dt.TokenHash
}
