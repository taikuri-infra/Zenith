package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemoryUserWebhookRepository is an in-memory UserWebhookRepository.
type MemoryUserWebhookRepository struct {
	mu         sync.RWMutex
	webhooks   map[string]*entities.UserWebhook    // id -> webhook
	deliveries map[string]*entities.WebhookDelivery // id -> delivery
}

func NewMemoryUserWebhookRepository() *MemoryUserWebhookRepository {
	return &MemoryUserWebhookRepository{
		webhooks:   make(map[string]*entities.UserWebhook),
		deliveries: make(map[string]*entities.WebhookDelivery),
	}
}

func (r *MemoryUserWebhookRepository) CreateWebhook(ctx context.Context, userID, url string, events []entities.WebhookEvent) (*entities.UserWebhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	secret := generateWebhookSecret()
	w := &entities.UserWebhook{
		ID:        uuid.New().String(),
		UserID:    userID,
		URL:       url,
		Events:    events,
		Secret:    secret,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.webhooks[w.ID] = w
	return w, nil
}

func (r *MemoryUserWebhookRepository) GetWebhook(ctx context.Context, id string) (*entities.UserWebhook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.webhooks[id]
	if !ok {
		return nil, fmt.Errorf("webhook not found")
	}
	return w, nil
}

func (r *MemoryUserWebhookRepository) ListWebhooksByUser(ctx context.Context, userID string) ([]entities.UserWebhook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.UserWebhook
	for _, w := range r.webhooks {
		if w.UserID == userID {
			result = append(result, *w)
		}
	}
	return result, nil
}

func (r *MemoryUserWebhookRepository) UpdateWebhook(ctx context.Context, id string, url *string, events []entities.WebhookEvent, active *bool) (*entities.UserWebhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.webhooks[id]
	if !ok {
		return nil, fmt.Errorf("webhook not found")
	}
	if url != nil {
		w.URL = *url
	}
	if events != nil {
		w.Events = events
	}
	if active != nil {
		w.Active = *active
	}
	w.UpdatedAt = time.Now()
	return w, nil
}

func (r *MemoryUserWebhookRepository) DeleteWebhook(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.webhooks[id]; !ok {
		return fmt.Errorf("webhook not found")
	}
	delete(r.webhooks, id)
	return nil
}

func (r *MemoryUserWebhookRepository) CountWebhooksByUser(ctx context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, w := range r.webhooks {
		if w.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *MemoryUserWebhookRepository) RecordDelivery(ctx context.Context, webhookID string, event entities.WebhookEvent, payload string, status entities.WebhookDeliveryStatus, statusCode int, errMsg string) (*entities.WebhookDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := &entities.WebhookDelivery{
		ID:         uuid.New().String(),
		WebhookID:  webhookID,
		Event:      event,
		Payload:    payload,
		Status:     status,
		StatusCode: statusCode,
		Error:      errMsg,
		Attempts:   1,
		CreatedAt:  time.Now(),
	}
	r.deliveries[d.ID] = d
	return d, nil
}

func (r *MemoryUserWebhookRepository) ListDeliveries(ctx context.Context, webhookID string, limit int) ([]entities.WebhookDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.WebhookDelivery
	for _, d := range r.deliveries {
		if d.WebhookID == webhookID {
			result = append(result, *d)
		}
	}
	// Simple limit
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result, nil
}

func generateWebhookSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "whsec_" + hex.EncodeToString(b)
}
