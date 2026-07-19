package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// safeString is a goroutine-safe string holder used to capture values written
// by an httptest handler goroutine and read from the test goroutine, without a
// data race (the -race detector flags an unsynchronized shared string here).
type safeString struct {
	mu sync.Mutex
	v  string
}

func (s *safeString) set(v string) {
	s.mu.Lock()
	s.v = v
	s.mu.Unlock()
}

func (s *safeString) get() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.v
}

func newTestWebhookService() (*WebhookDeliveryService, *memory.MemoryUserWebhookRepository) {
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	svc := NewWebhookDeliveryService(webhookRepo)
	return svc, webhookRepo
}

func TestWebhookDispatch_NoWebhooks(t *testing.T) {
	svc, _ := newTestWebhookService()
	ctx := context.Background()

	// Should not panic even with no webhooks
	svc.DispatchEvent(ctx, "user-1", entities.WebhookEventDeploySuccess, map[string]interface{}{
		"app": "my-app",
	})
}

func TestWebhookDispatch_InactiveWebhookSkipped(t *testing.T) {
	svc, webhookRepo := newTestWebhookService()
	ctx := context.Background()

	userID := "user-inactive"
	wh, _ := webhookRepo.CreateWebhook(ctx, userID, "https://example.com/hook", []entities.WebhookEvent{entities.WebhookEventDeploySuccess})
	// Deactivate
	active := false
	webhookRepo.UpdateWebhook(ctx, wh.ID, nil, nil, &active)

	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeploySuccess, map[string]interface{}{"app": "test"})
	time.Sleep(50 * time.Millisecond)

	deliveries, _ := webhookRepo.ListDeliveries(ctx, wh.ID, 10)
	if len(deliveries) != 0 {
		t.Errorf("Expected 0 deliveries for inactive webhook, got %d", len(deliveries))
	}
}

func TestWebhookDispatch_UnmatchedEventSkipped(t *testing.T) {
	svc, webhookRepo := newTestWebhookService()
	ctx := context.Background()

	userID := "user-unmatched"
	wh, _ := webhookRepo.CreateWebhook(ctx, userID, "https://example.com/hook", []entities.WebhookEvent{entities.WebhookEventDBCreated})

	// Dispatch deploy event — webhook is subscribed to db.created only
	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeploySuccess, map[string]interface{}{"app": "test"})
	time.Sleep(50 * time.Millisecond)

	deliveries, _ := webhookRepo.ListDeliveries(ctx, wh.ID, 10)
	if len(deliveries) != 0 {
		t.Errorf("Expected 0 deliveries for unmatched event, got %d", len(deliveries))
	}
}

func TestWebhookDispatch_SuccessfulDelivery(t *testing.T) {
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	ctx := context.Background()

	// Set up a test HTTP server
	var receivedEvent safeString
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedEvent.set(r.Header.Get("X-Zenith-Event"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	userID := "user-success"
	wh, _ := webhookRepo.CreateWebhook(ctx, userID, server.URL, []entities.WebhookEvent{entities.WebhookEventDeploySuccess})

	svc := NewWebhookDeliveryService(webhookRepo)
	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeploySuccess, map[string]interface{}{"app": "my-app"})
	time.Sleep(200 * time.Millisecond)

	if got := receivedEvent.get(); got != string(entities.WebhookEventDeploySuccess) {
		t.Errorf("Expected event header %s, got %s", entities.WebhookEventDeploySuccess, got)
	}

	deliveries, _ := webhookRepo.ListDeliveries(ctx, wh.ID, 10)
	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery, got %d", len(deliveries))
	}
	if deliveries[0].Status != entities.WebhookDeliverySuccess {
		t.Errorf("Expected delivery status success, got %s", deliveries[0].Status)
	}
}

func TestWebhookDispatch_FailedDelivery_4xx(t *testing.T) {
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	userID := "user-fail"
	wh, _ := webhookRepo.CreateWebhook(ctx, userID, server.URL, []entities.WebhookEvent{entities.WebhookEventDeployFailed})

	svc := NewWebhookDeliveryService(webhookRepo)
	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeployFailed, map[string]interface{}{"error": "OOM"})
	time.Sleep(200 * time.Millisecond)

	deliveries, _ := webhookRepo.ListDeliveries(ctx, wh.ID, 10)
	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery, got %d", len(deliveries))
	}
	if deliveries[0].Status != entities.WebhookDeliveryFailed {
		t.Errorf("Expected delivery status failed, got %s", deliveries[0].Status)
	}
	if deliveries[0].StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", deliveries[0].StatusCode)
	}
}

func TestWebhookDispatch_FailedDelivery_ConnectionRefused(t *testing.T) {
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	ctx := context.Background()

	userID := "user-connrefused"
	wh, _ := webhookRepo.CreateWebhook(ctx, userID, "http://127.0.0.1:1", []entities.WebhookEvent{entities.WebhookEventDeploySuccess})

	svc := NewWebhookDeliveryService(webhookRepo)
	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeploySuccess, map[string]interface{}{"app": "test"})
	time.Sleep(500 * time.Millisecond)

	deliveries, _ := webhookRepo.ListDeliveries(ctx, wh.ID, 10)
	if len(deliveries) != 1 {
		t.Fatalf("Expected 1 delivery, got %d", len(deliveries))
	}
	if deliveries[0].Status != entities.WebhookDeliveryFailed {
		t.Errorf("Expected delivery status failed, got %s", deliveries[0].Status)
	}
}

func TestWebhookDispatch_HMAC_Signature(t *testing.T) {
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	ctx := context.Background()

	var signatureHeader safeString
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signatureHeader.set(r.Header.Get("X-Zenith-Signature"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	userID := "user-hmac"
	wh, _ := webhookRepo.CreateWebhook(ctx, userID, server.URL, []entities.WebhookEvent{entities.WebhookEventDeploySuccess})
	// Webhook secret is auto-generated by memory adapter (whsec_...)
	if wh.Secret == "" {
		t.Fatal("Expected webhook secret to be generated")
	}

	svc := NewWebhookDeliveryService(webhookRepo)
	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeploySuccess, map[string]interface{}{"app": "test"})
	time.Sleep(200 * time.Millisecond)

	sig := signatureHeader.get()
	if sig == "" {
		t.Error("Expected X-Zenith-Signature header to be set when webhook has a secret")
	}
	if len(sig) < 10 || sig[:7] != "sha256=" {
		t.Errorf("Expected signature to start with 'sha256=', got: %s", sig)
	}
}

func TestWebhookDispatch_TimestampHeader(t *testing.T) {
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	ctx := context.Background()

	var timestampHeader safeString
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestampHeader.set(r.Header.Get("X-Zenith-Delivery-Timestamp"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	userID := "user-ts"
	webhookRepo.CreateWebhook(ctx, userID, server.URL, []entities.WebhookEvent{entities.WebhookEventDeploySuccess})

	svc := NewWebhookDeliveryService(webhookRepo)
	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeploySuccess, map[string]interface{}{"app": "test"})
	time.Sleep(200 * time.Millisecond)

	if timestampHeader.get() == "" {
		t.Error("Expected X-Zenith-Delivery-Timestamp header")
	}
}

func TestWebhookDispatch_MultipleWebhooks(t *testing.T) {
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	ctx := context.Background()

	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	userID := "user-multi"
	webhookRepo.CreateWebhook(ctx, userID, server.URL+"/a", []entities.WebhookEvent{entities.WebhookEventDeploySuccess})
	webhookRepo.CreateWebhook(ctx, userID, server.URL+"/b", []entities.WebhookEvent{entities.WebhookEventDeploySuccess})

	svc := NewWebhookDeliveryService(webhookRepo)
	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeploySuccess, map[string]interface{}{"app": "test"})
	time.Sleep(300 * time.Millisecond)

	if callCount.Load() != 2 {
		t.Errorf("Expected 2 webhook calls, got %d", callCount.Load())
	}
}
