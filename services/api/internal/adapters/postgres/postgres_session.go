package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

func (r *PostgresSessionRepository) CreateSession(ctx context.Context, userID, ipAddress, userAgent string) (*entities.Session, error) {
	id := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour) // 7 days
	device := detectSessionDevice(userAgent)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, ip_address, user_agent, device, created_at, expires_at, last_seen_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, userID, ipAddress, userAgent, device, now, expiresAt, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &entities.Session{
		ID:         id,
		UserID:     userID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Device:     device,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		LastSeenAt: now,
	}, nil
}

func (r *PostgresSessionRepository) GetSession(ctx context.Context, id string) (*entities.Session, error) {
	var s entities.Session
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, ip_address, user_agent, device, created_at, expires_at, last_seen_at
		 FROM sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.UserID, &s.IPAddress, &s.UserAgent, &s.Device, &s.CreatedAt, &s.ExpiresAt, &s.LastSeenAt)
	if err != nil {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return &s, nil
}

func (r *PostgresSessionRepository) ListSessionsByUser(ctx context.Context, userID string) ([]entities.Session, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, ip_address, user_agent, device, created_at, expires_at, last_seen_at
		 FROM sessions WHERE user_id = $1 ORDER BY last_seen_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []entities.Session
	for rows.Next() {
		var s entities.Session
		if err := rows.Scan(&s.ID, &s.UserID, &s.IPAddress, &s.UserAgent, &s.Device,
			&s.CreatedAt, &s.ExpiresAt, &s.LastSeenAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *PostgresSessionRepository) DeleteSession(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

func (r *PostgresSessionRepository) DeleteAllUserSessions(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete all sessions: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) UpdateLastSeen(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE sessions SET last_seen_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

func detectSessionDevice(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone"):
		return "Mobile"
	case strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad"):
		return "Tablet"
	case strings.Contains(ua, "curl") || strings.Contains(ua, "httpie") || strings.Contains(ua, "postman"):
		return "CLI/API"
	default:
		return "Desktop"
	}
}
