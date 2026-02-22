package memory

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"time"
)

// MemoryMFARepository is an in-memory MFARepository.
type MemoryMFARepository struct {
	mu          sync.RWMutex
	enrollments map[string]*entities.MFAEnrollment // keyed by userID
}

func NewMemoryMFARepository() *MemoryMFARepository {
	return &MemoryMFARepository{enrollments: make(map[string]*entities.MFAEnrollment)}
}

func (r *MemoryMFARepository) GetEnrollment(ctx context.Context, userID string) (*entities.MFAEnrollment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.enrollments[userID]
	if !ok {
		return &entities.MFAEnrollment{
			UserID: userID,
			Status: entities.MFAStatusDisabled,
		}, nil
	}
	return e, nil
}

func (r *MemoryMFARepository) StartEnrollment(ctx context.Context, userID string) (*entities.MFAEnrollment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	secret := generateTOTPSecret()
	codes := generateBackupCodes(10)

	e := &entities.MFAEnrollment{
		UserID:      userID,
		Status:      entities.MFAStatusPending,
		Secret:      secret,
		BackupCodes: codes,
		UsedCodes:   []string{},
		CreatedAt:   time.Now(),
	}
	r.enrollments[userID] = e
	return e, nil
}

func (r *MemoryMFARepository) ConfirmEnrollment(ctx context.Context, userID string) (*entities.MFAEnrollment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.enrollments[userID]
	if !ok {
		return nil, fmt.Errorf("no pending enrollment for user %s", userID)
	}
	if e.Status != entities.MFAStatusPending {
		return nil, fmt.Errorf("enrollment is not in pending state")
	}
	e.Status = entities.MFAStatusEnabled
	e.EnabledAt = time.Now()
	return e, nil
}

func (r *MemoryMFARepository) DisableEnrollment(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.enrollments, userID)
	return nil
}

func (r *MemoryMFARepository) UseBackupCode(ctx context.Context, userID, code string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.enrollments[userID]
	if !ok || e.Status != entities.MFAStatusEnabled {
		return false, fmt.Errorf("MFA not enabled for user %s", userID)
	}
	for i, c := range e.BackupCodes {
		if c == code {
			// Mark as used
			e.BackupCodes = append(e.BackupCodes[:i], e.BackupCodes[i+1:]...)
			e.UsedCodes = append(e.UsedCodes, code)
			return true, nil
		}
	}
	return false, nil
}

func (r *MemoryMFARepository) RegenerateBackupCodes(ctx context.Context, userID string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.enrollments[userID]
	if !ok || e.Status != entities.MFAStatusEnabled {
		return nil, fmt.Errorf("MFA not enabled for user %s", userID)
	}
	codes := generateBackupCodes(10)
	e.BackupCodes = codes
	e.UsedCodes = []string{}
	return codes, nil
}

// generateTOTPSecret creates a random 20-byte base32-encoded secret.
func generateTOTPSecret() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

// generateBackupCodes creates n 8-char alphanumeric recovery codes.
func generateBackupCodes(n int) []string {
	codes := make([]string, n)
	for i := 0; i < n; i++ {
		b := make([]byte, 4)
		_, _ = rand.Read(b)
		code := fmt.Sprintf("%x", b)
		codes[i] = strings.ToUpper(code)
	}
	return codes
}
