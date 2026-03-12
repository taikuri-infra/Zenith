package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ ports.UserEventRepository = (*PostgresUserEventRepository)(nil)

type PostgresUserEventRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserEventRepository(pool *pgxpool.Pool) *PostgresUserEventRepository {
	return &PostgresUserEventRepository{pool: pool}
}

func (r *PostgresUserEventRepository) Track(ctx context.Context, event *entities.UserEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	props, _ := json.Marshal(event.Properties)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_events (id, user_id, event_type, properties, ip_address, user_agent, created_at)
		 VALUES ($1, $2, $3, $4, $5::inet, $6, $7)`,
		event.ID, event.UserID, event.EventType, props,
		nilIfEmptyInet(event.IPAddress), event.UserAgent, event.CreatedAt,
	)
	return err
}

func (r *PostgresUserEventRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]entities.UserEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, event_type, properties, ip_address, user_agent, created_at
		 FROM user_events WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserEvents(rows)
}

func (r *PostgresUserEventRepository) ListByType(ctx context.Context, eventType string, limit, offset int) ([]entities.UserEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, event_type, properties, ip_address, user_agent, created_at
		 FROM user_events WHERE event_type = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		eventType, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserEvents(rows)
}

func (r *PostgresUserEventRepository) CountByType(ctx context.Context, eventType string, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_events WHERE event_type = $1 AND created_at >= $2`,
		eventType, since,
	).Scan(&count)
	return count, err
}

func (r *PostgresUserEventRepository) GetUserActivity(ctx context.Context, userID string, since time.Time) ([]entities.UserEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, event_type, properties, ip_address, user_agent, created_at
		 FROM user_events WHERE user_id = $1 AND created_at >= $2 ORDER BY created_at DESC`,
		userID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserEvents(rows)
}

func (r *PostgresUserEventRepository) GetFunnelData(ctx context.Context, steps []string, since time.Time) (map[string]int, error) {
	result := make(map[string]int)
	for _, step := range steps {
		var count int
		err := r.pool.QueryRow(ctx,
			`SELECT COUNT(DISTINCT user_id) FROM user_events WHERE event_type = $1 AND created_at >= $2`,
			step, since,
		).Scan(&count)
		if err != nil {
			return nil, err
		}
		result[step] = count
	}
	return result, nil
}

func (r *PostgresUserEventRepository) PurgeOlderThan(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM user_events WHERE created_at < $1`, before,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func nilIfEmptyInet(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func scanUserEvents(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]entities.UserEvent, error) {
	var events []entities.UserEvent
	for rows.Next() {
		var e entities.UserEvent
		var props []byte
		var ip *string
		err := rows.Scan(&e.ID, &e.UserID, &e.EventType, &props, &ip, &e.UserAgent, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		if ip != nil {
			e.IPAddress = *ip
		}
		if len(props) > 0 {
			_ = json.Unmarshal(props, &e.Properties)
		}
		if e.Properties == nil {
			e.Properties = make(map[string]interface{})
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
