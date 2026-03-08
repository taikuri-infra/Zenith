package postgres

import (
	"context"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresNotificationRepository stores notifications and activity in PostgreSQL.
type PostgresNotificationRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresNotificationRepository creates a new Postgres-backed notification repo.
func NewPostgresNotificationRepository(pool *pgxpool.Pool) *PostgresNotificationRepository {
	return &PostgresNotificationRepository{pool: pool}
}

func (r *PostgresNotificationRepository) CreateNotification(ctx context.Context, notif *entities.Notification) error {
	if notif.ID == "" {
		notif.ID = uuid.New().String()
	}
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notifications (id, user_id, type, title, message, read, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		notif.ID, notif.UserID, notif.Type, notif.Title, notif.Message, notif.Read, notif.CreatedAt,
	)
	return err
}

func (r *PostgresNotificationRepository) ListByUser(ctx context.Context, userID string, limit int) ([]entities.Notification, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, type, title, message, read, created_at
		 FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entities.Notification
	for rows.Next() {
		var n entities.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, nil
}

func (r *PostgresNotificationRepository) MarkRead(ctx context.Context, userID string, ids []string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read = true WHERE user_id = $1 AND id = ANY($2)`,
		userID, ids,
	)
	return err
}

func (r *PostgresNotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`,
		userID,
	)
	return err
}

func (r *PostgresNotificationRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = false`,
		userID,
	).Scan(&count)
	return count, err
}

func (r *PostgresNotificationRepository) AddActivity(ctx context.Context, entry *entities.ActivityEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO activity_log (id, user_id, action, resource, details, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		entry.ID, entry.UserID, entry.Action, entry.Resource, entry.Details, entry.CreatedAt,
	)
	return err
}

func (r *PostgresNotificationRepository) ListActivity(ctx context.Context, userID string, limit int) ([]entities.ActivityEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, action, resource, details, created_at
		 FROM activity_log WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entities.ActivityEntry
	for rows.Next() {
		var e entities.ActivityEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Action, &e.Resource, &e.Details, &e.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}
