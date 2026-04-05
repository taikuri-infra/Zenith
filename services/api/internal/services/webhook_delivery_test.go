package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

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
	var receivedEvent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedEvent = r.Header.Get("X-Zenith-Event")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	userID := "user-success"
	wh, _ := webhookRepo.CreateWebhook(ctx, userID, server.URL, []entities.WebhookEvent{entities.WebhookEventDeploySuccess})

	svc := NewWebhookDeliveryService(webhookRepo)
	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeploySuccess, map[string]interface{}{"app": "my-app"})
	time.Sleep(200 * time.Millisecond)

	if receivedEvent != string(entities.WebhookEventDeploySuccess) {
		t.Errorf("Expected event header %s, got %s", entities.WebhookEventDeploySuccess, receivedEvent)
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

	var signatureHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signatureHeader = r.Header.Get("X-Zenith-Signature")
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

	if signatureHeader == "" {
		t.Error("Expected X-Zenith-Signature header to be set when webhook has a secret")
	}
	if len(signatureHeader) < 10 || signatureHeader[:7] != "sha256=" {
		t.Errorf("Expected signature to start with 'sha256=', got: %s", signatureHeader)
	}
}

func TestWebhookDispatch_TimestampHeader(t *testing.T) {
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	ctx := context.Background()

	var timestampHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestampHeader = r.Header.Get("X-Zenith-Delivery-Timestamp")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	userID := "user-ts"
	webhookRepo.CreateWebhook(ctx, userID, server.URL, []entities.WebhookEvent{entities.WebhookEventDeploySuccess})

	svc := NewWebhookDeliveryService(webhookRepo)
	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeploySuccess, map[string]interface{}{"app": "test"})
	time.Sleep(200 * time.Millisecond)

	if timestampHeader == "" {
		t.Error("Expected X-Zenith-Delivery-Timestamp header")
	}
}

func TestWebhookDispatch_MultipleWebhooks(t *testing.T) {
	webhookRepo := memory.NewMemoryUserWebhookRepository()
	ctx := context.Background()

	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	userID := "user-multi"
	webhookRepo.CreateWebhook(ctx, userID, server.URL+"/a", []entities.WebhookEvent{entities.WebhookEventDeploySuccess})
	webhookRepo.CreateWebhook(ctx, userID, server.URL+"/b", []entities.WebhookEvent{entities.WebhookEventDeploySuccess})

	svc := NewWebhookDeliveryService(webhookRepo)
	svc.DispatchEvent(ctx, userID, entities.WebhookEventDeploySuccess, map[string]interface{}{"app": "test"})
	time.Sleep(300 * time.Millisecond)

	if callCount != 2 {
		t.Errorf("Expected 2 webhook calls, got %d", callCount)
	}
}
