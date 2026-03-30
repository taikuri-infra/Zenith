package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// WebhookDeliveryService dispatches webhook events to user-registered URLs.
type WebhookDeliveryService struct {
	webhookRepo ports.UserWebhookRepository
	httpClient  *http.Client
}

// NewWebhookDeliveryService creates a new WebhookDeliveryService.
func NewWebhookDeliveryService(webhookRepo ports.UserWebhookRepository) *WebhookDeliveryService {
	return &WebhookDeliveryService{
		webhookRepo: webhookRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DispatchEvent sends webhook payloads to all active webhooks matching the event.
// Delivery is async per-webhook — failures don't block the caller.
func (s *WebhookDeliveryService) DispatchEvent(ctx context.Context, userID string, event entities.WebhookEvent, payload map[string]interface{}) {
	webhooks, err := s.webhookRepo.ListWebhooksByUser(ctx, userID)
	if err != nil {
		slog.Error("webhook dispatch: failed to list webhooks", "user_id", userID, "error", err)
		return
	}

	payloadJSON, _ := json.Marshal(payload)

	for _, wh := range webhooks {
		if !wh.Active {
			continue
		}
		subscribed := false
		for _, e := range wh.Events {
			if e == event {
				subscribed = true
				break
			}
		}
		if !subscribed {
			continue
		}

		go s.deliver(ctx, wh, event, payloadJSON)
	}
}

func (s *WebhookDeliveryService) deliver(ctx context.Context, wh entities.UserWebhook, event entities.WebhookEvent, payload []byte) {
	req, err := http.NewRequestWithContext(ctx, "POST", wh.URL, bytes.NewReader(payload))
	if err != nil {
		s.record(ctx, wh.ID, event, string(payload), entities.WebhookDeliveryFailed, 0, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Zenith-Event", string(event))
	req.Header.Set("X-Zenith-Delivery-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

	// HMAC-SHA256 signature
	if wh.Secret != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret))
		mac.Write(payload)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Zenith-Signature", "sha256="+sig)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.record(ctx, wh.ID, event, string(payload), entities.WebhookDeliveryFailed, 0, err.Error())
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	status := entities.WebhookDeliverySuccess
	errMsg := ""
	if resp.StatusCode >= 400 {
		status = entities.WebhookDeliveryFailed
		errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	s.record(ctx, wh.ID, event, string(payload), status, resp.StatusCode, errMsg)
}

func (s *WebhookDeliveryService) record(ctx context.Context, webhookID string, event entities.WebhookEvent, payload string, status entities.WebhookDeliveryStatus, code int, errMsg string) {
	if _, err := s.webhookRepo.RecordDelivery(ctx, webhookID, event, payload, status, code, errMsg); err != nil {
		slog.Error("webhook: failed to record delivery", "webhook_id", webhookID, "error", err)
	}
}
