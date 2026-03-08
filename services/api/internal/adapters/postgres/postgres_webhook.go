package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserWebhookRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserWebhookRepository(pool *pgxpool.Pool) *PostgresUserWebhookRepository {
	return &PostgresUserWebhookRepository{pool: pool}
}

func (r *PostgresUserWebhookRepository) CreateWebhook(ctx context.Context, userID, url string, events []entities.WebhookEvent) (*entities.UserWebhook, error) {
	id := uuid.New().String()
	now := time.Now()
	secret := generateWebhookSecretValue()

	eventStrings := make([]string, len(events))
	for i, e := range events {
		eventStrings[i] = string(e)
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_webhooks (id, user_id, url, events, secret, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, true, $6, $7)`,
		id, userID, url, eventStrings, secret, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create webhook: %w", err)
	}
	return &entities.UserWebhook{
		ID:        id,
		UserID:    userID,
		URL:       url,
		Events:    events,
		Secret:    secret,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *PostgresUserWebhookRepository) GetWebhook(ctx context.Context, id string) (*entities.UserWebhook, error) {
	var w entities.UserWebhook
	var eventStrings []string
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, url, events, secret, active, created_at, updated_at
		 FROM user_webhooks WHERE id = $1`, id,
	).Scan(&w.ID, &w.UserID, &w.URL, &eventStrings, &w.Secret, &w.Active, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("webhook not found: %s", id)
	}
	w.Events = make([]entities.WebhookEvent, len(eventStrings))
	for i, s := range eventStrings {
		w.Events[i] = entities.WebhookEvent(s)
	}
	return &w, nil
}

func (r *PostgresUserWebhookRepository) ListWebhooksByUser(ctx context.Context, userID string) ([]entities.UserWebhook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, url, events, secret, active, created_at, updated_at
		 FROM user_webhooks WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []entities.UserWebhook
	for rows.Next() {
		var w entities.UserWebhook
		var eventStrings []string
		if err := rows.Scan(&w.ID, &w.UserID, &w.URL, &eventStrings, &w.Secret, &w.Active,
			&w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		w.Events = make([]entities.WebhookEvent, len(eventStrings))
		for i, s := range eventStrings {
			w.Events[i] = entities.WebhookEvent(s)
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, nil
}

func (r *PostgresUserWebhookRepository) UpdateWebhook(ctx context.Context, id string, url *string, events []entities.WebhookEvent, active *bool) (*entities.UserWebhook, error) {
	now := time.Now()
	sets := []string{"updated_at = $1"}
	args := []interface{}{now}
	argIdx := 2

	if url != nil {
		sets = append(sets, fmt.Sprintf("url = $%d", argIdx))
		args = append(args, *url)
		argIdx++
	}
	if events != nil {
		eventStrings := make([]string, len(events))
		for i, e := range events {
			eventStrings[i] = string(e)
		}
		sets = append(sets, fmt.Sprintf("events = $%d", argIdx))
		args = append(args, eventStrings)
		argIdx++
	}
	if active != nil {
		sets = append(sets, fmt.Sprintf("active = $%d", argIdx))
		args = append(args, *active)
		argIdx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE user_webhooks SET %s WHERE id = $%d",
		joinStrings(sets, ", "), argIdx)

	ct, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update webhook: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("webhook not found: %s", id)
	}
	return r.GetWebhook(ctx, id)
}

func (r *PostgresUserWebhookRepository) DeleteWebhook(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM user_webhooks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("webhook not found: %s", id)
	}
	return nil
}

func (r *PostgresUserWebhookRepository) CountWebhooksByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_webhooks WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count webhooks: %w", err)
	}
	return count, nil
}

func (r *PostgresUserWebhookRepository) RecordDelivery(ctx context.Context, webhookID string, event entities.WebhookEvent, payload string, status entities.WebhookDeliveryStatus, statusCode int, errMsg string) (*entities.WebhookDelivery, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status, status_code, error, attempts, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 1, $8)`,
		id, webhookID, string(event), payload, string(status), statusCode, errMsg, now,
	)
	if err != nil {
		return nil, fmt.Errorf("record delivery: %w", err)
	}
	return &entities.WebhookDelivery{
		ID:         id,
		WebhookID:  webhookID,
		Event:      event,
		Payload:    payload,
		Status:     status,
		StatusCode: statusCode,
		Error:      errMsg,
		Attempts:   1,
		CreatedAt:  now,
	}, nil
}

func (r *PostgresUserWebhookRepository) ListDeliveries(ctx context.Context, webhookID string, limit int) ([]entities.WebhookDelivery, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, webhook_id, event, payload, status, status_code, error, attempts, created_at
		 FROM webhook_deliveries WHERE webhook_id = $1 ORDER BY created_at DESC LIMIT $2`,
		webhookID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []entities.WebhookDelivery
	for rows.Next() {
		var d entities.WebhookDelivery
		var event, status string
		if err := rows.Scan(&d.ID, &d.WebhookID, &event, &d.Payload, &status,
			&d.StatusCode, &d.Error, &d.Attempts, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan delivery: %w", err)
		}
		d.Event = entities.WebhookEvent(event)
		d.Status = entities.WebhookDeliveryStatus(status)
		deliveries = append(deliveries, d)
	}
	return deliveries, nil
}

func generateWebhookSecretValue() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "whsec_" + hex.EncodeToString(b)
}

func joinStrings(s []string, sep string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += sep
		}
		result += v
	}
	return result
}
