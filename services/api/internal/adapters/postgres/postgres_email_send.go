package postgres

import (
	"context"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ ports.EmailSendRepository = (*PostgresEmailSendRepository)(nil)

type PostgresEmailSendRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresEmailSendRepository(pool *pgxpool.Pool) *PostgresEmailSendRepository {
	return &PostgresEmailSendRepository{pool: pool}
}

func (r *PostgresEmailSendRepository) Record(ctx context.Context, send *entities.EmailSend) error {
	if send.ID == "" {
		send.ID = uuid.New().String()
	}
	if send.SentAt.IsZero() {
		send.SentAt = time.Now()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_sends (id, user_id, template_key, sent_at)
		 VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, template_key) DO NOTHING`,
		send.ID, send.UserID, send.TemplateKey, send.SentAt,
	)
	return err
}

func (r *PostgresEmailSendRepository) HasSent(ctx context.Context, userID, templateKey string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_sends WHERE user_id = $1 AND template_key = $2`,
		userID, templateKey,
	).Scan(&count)
	return count > 0, err
}

func (r *PostgresEmailSendRepository) MarkOpened(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_sends SET opened_at = NOW() WHERE id = $1 AND opened_at IS NULL`, id,
	)
	return err
}

func (r *PostgresEmailSendRepository) MarkClicked(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_sends SET clicked_at = NOW() WHERE id = $1 AND clicked_at IS NULL`, id,
	)
	return err
}

func (r *PostgresEmailSendRepository) GetStats(ctx context.Context) (*entities.EmailStats, error) {
	stats := &entities.EmailStats{ByTemplate: make(map[string]int)}
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM email_sends`).Scan(&stats.Sent)
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM email_sends WHERE opened_at IS NOT NULL`).Scan(&stats.Opened)
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM email_sends WHERE clicked_at IS NOT NULL`).Scan(&stats.Clicked)

	rows, err := r.pool.Query(ctx,
		`SELECT template_key, COUNT(*) FROM email_sends GROUP BY template_key`,
	)
	if err != nil {
		return stats, nil
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err == nil {
			stats.ByTemplate[key] = count
		}
	}
	return stats, nil
}

func (r *PostgresEmailSendRepository) ListByUser(ctx context.Context, userID string) ([]entities.EmailSend, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, template_key, sent_at, opened_at, clicked_at
		 FROM email_sends WHERE user_id = $1 ORDER BY sent_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sends []entities.EmailSend
	for rows.Next() {
		var s entities.EmailSend
		if err := rows.Scan(&s.ID, &s.UserID, &s.TemplateKey, &s.SentAt, &s.OpenedAt, &s.ClickedAt); err != nil {
			return nil, err
		}
		sends = append(sends, s)
	}
	return sends, rows.Err()
}
