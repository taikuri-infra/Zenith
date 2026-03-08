package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSupportRepository is a PostgreSQL-backed SupportRepository.
type PostgresSupportRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSupportRepository creates a new PostgresSupportRepository.
func NewPostgresSupportRepository(pool *pgxpool.Pool) *PostgresSupportRepository {
	return &PostgresSupportRepository{pool: pool}
}

const ticketSelectCols = `id, user_id, subject, category, priority, status, assigned_to, created_at, updated_at, closed_at`

func scanTicket(scanner interface{ Scan(dest ...any) error }) (*entities.SupportTicket, error) {
	var t entities.SupportTicket
	var assignedTo *string
	err := scanner.Scan(
		&t.ID, &t.UserID, &t.Subject, &t.Category, &t.Priority, &t.Status,
		&assignedTo, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt,
	)
	if err != nil {
		return nil, err
	}
	if assignedTo != nil {
		t.AssignedTo = *assignedTo
	}
	return &t, nil
}

func (r *PostgresSupportRepository) CreateTicket(ctx context.Context, ticket *entities.SupportTicket, initialMsg *entities.SupportMessage) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO support_tickets (id, user_id, subject, category, priority, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		ticket.ID, ticket.UserID, ticket.Subject, string(ticket.Category), string(ticket.Priority),
		string(ticket.Status), ticket.CreatedAt, ticket.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ticket: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO support_messages (id, ticket_id, sender_id, sender_role, body, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		initialMsg.ID, initialMsg.TicketID, initialMsg.SenderID, string(initialMsg.SenderRole),
		initialMsg.Body, initialMsg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert initial message: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresSupportRepository) GetTicket(ctx context.Context, id string) (*entities.SupportTicket, error) {
	t, err := scanTicket(r.pool.QueryRow(ctx,
		`SELECT `+ticketSelectCols+` FROM support_tickets WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %s", id)
	}
	return t, nil
}

func (r *PostgresSupportRepository) ListTicketsByUser(ctx context.Context, userID string) ([]entities.SupportTicket, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+ticketSelectCols+` FROM support_tickets WHERE user_id = $1 ORDER BY updated_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tickets by user: %w", err)
	}
	defer rows.Close()

	var tickets []entities.SupportTicket
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ticket: %w", err)
		}
		tickets = append(tickets, *t)
	}
	return tickets, nil
}

func (r *PostgresSupportRepository) ListAllTickets(ctx context.Context, status string, limit, offset int) ([]entities.SupportTicket, int, error) {
	// Count
	var total int
	countQuery := `SELECT COUNT(*) FROM support_tickets`
	args := []interface{}{}
	if status != "" {
		countQuery += ` WHERE status = $1`
		args = append(args, status)
	}
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tickets: %w", err)
	}

	// List
	listQuery := `SELECT ` + ticketSelectCols + ` FROM support_tickets`
	listArgs := []interface{}{}
	paramIdx := 1
	if status != "" {
		listQuery += fmt.Sprintf(` WHERE status = $%d`, paramIdx)
		listArgs = append(listArgs, status)
		paramIdx++
	}
	listQuery += ` ORDER BY updated_at DESC`
	listQuery += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, paramIdx, paramIdx+1)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list all tickets: %w", err)
	}
	defer rows.Close()

	var tickets []entities.SupportTicket
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan ticket: %w", err)
		}
		tickets = append(tickets, *t)
	}
	return tickets, total, nil
}

func (r *PostgresSupportRepository) UpdateTicketStatus(ctx context.Context, id string, status entities.TicketStatus) error {
	now := time.Now()
	var closedAt *time.Time
	if status == entities.TicketStatusClosed || status == entities.TicketStatusResolved {
		closedAt = &now
	}
	ct, err := r.pool.Exec(ctx,
		`UPDATE support_tickets SET status = $1, updated_at = $2, closed_at = $3 WHERE id = $4`,
		string(status), now, closedAt, id,
	)
	if err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("ticket not found: %s", id)
	}
	return nil
}

func (r *PostgresSupportRepository) UpdateTicketAssignee(ctx context.Context, id, adminUserID string) error {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE support_tickets SET assigned_to = $1, updated_at = $2 WHERE id = $3`,
		adminUserID, now, id,
	)
	if err != nil {
		return fmt.Errorf("update ticket assignee: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("ticket not found: %s", id)
	}
	return nil
}

func (r *PostgresSupportRepository) AddMessage(ctx context.Context, msg *entities.SupportMessage) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO support_messages (id, ticket_id, sender_id, sender_role, body, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		msg.ID, msg.TicketID, msg.SenderID, string(msg.SenderRole), msg.Body, msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	// Touch ticket updated_at
	r.pool.Exec(ctx, `UPDATE support_tickets SET updated_at = $1 WHERE id = $2`, msg.CreatedAt, msg.TicketID)
	return nil
}

func (r *PostgresSupportRepository) ListMessages(ctx context.Context, ticketID string) ([]entities.SupportMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, ticket_id, sender_id, sender_role, body, created_at
		 FROM support_messages WHERE ticket_id = $1 ORDER BY created_at ASC`, ticketID,
	)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var messages []entities.SupportMessage
	for rows.Next() {
		var m entities.SupportMessage
		if err := rows.Scan(&m.ID, &m.TicketID, &m.SenderID, &m.SenderRole, &m.Body, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, nil
}
