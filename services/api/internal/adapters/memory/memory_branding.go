package memory

import (
	"context"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type MemoryBrandingRepository struct {
	mu       sync.RWMutex
	dpas     map[string]*entities.DPARecord     // keyed by userID
	branding map[string]*entities.BrandingConfig // keyed by userID
}

func NewMemoryBrandingRepository() *MemoryBrandingRepository {
	return &MemoryBrandingRepository{
		dpas:     make(map[string]*entities.DPARecord),
		branding: make(map[string]*entities.BrandingConfig),
	}
}

// DPA methods

func (r *MemoryBrandingRepository) GetDPA(ctx context.Context, userID string) (*entities.DPARecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.dpas[userID]
	if !ok {
		return &entities.DPARecord{UserID: userID, Status: entities.DPAUnsigned}, nil
	}
	return d, nil
}

func (r *MemoryBrandingRepository) SignDPA(ctx context.Context, userID, signedBy, ipAddress string) (*entities.DPARecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := &entities.DPARecord{
		UserID:    userID,
		Status:    entities.DPASigned,
		SignedBy:  signedBy,
		SignedAt:  time.Now(),
		IPAddress: ipAddress,
	}
	r.dpas[userID] = d
	return d, nil
}

// Branding methods

func (r *MemoryBrandingRepository) GetBranding(ctx context.Context, userID string) (*entities.BrandingConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.branding[userID]
	if !ok {
		return &entities.BrandingConfig{UserID: userID}, nil
	}
	return b, nil
}

func (r *MemoryBrandingRepository) UpdateBranding(ctx context.Context, userID string, companyName, logoURL, primaryColor *string, hideBranding *bool) (*entities.BrandingConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.branding[userID]
	if !ok {
		b = &entities.BrandingConfig{UserID: userID}
		r.branding[userID] = b
	}
	if companyName != nil {
		b.CompanyName = *companyName
	}
	if logoURL != nil {
		b.LogoURL = *logoURL
	}
	if primaryColor != nil {
		b.PrimaryColor = *primaryColor
	}
	if hideBranding != nil {
		b.HideBranding = *hideBranding
	}
	b.UpdatedAt = time.Now()
	return b, nil
}

func (r *MemoryBrandingRepository) SetDashboardDomain(ctx context.Context, userID, domain string) (*entities.BrandingConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.branding[userID]
	if !ok {
		b = &entities.BrandingConfig{UserID: userID}
		r.branding[userID] = b
	}
	b.DashboardDomain = domain
	b.DomainVerified = false
	b.UpdatedAt = time.Now()

	// Simulate async DNS verification
	go func() {
		time.Sleep(3 * time.Second)
		r.mu.Lock()
		defer r.mu.Unlock()
		if b, ok := r.branding[userID]; ok && b.DashboardDomain == domain {
			b.DomainVerified = true
		}
	}()

	return b, nil
}
