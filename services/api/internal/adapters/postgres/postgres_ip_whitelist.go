package postgres

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresIPWhitelistRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresIPWhitelistRepository(pool *pgxpool.Pool) *PostgresIPWhitelistRepository {
	return &PostgresIPWhitelistRepository{pool: pool}
}

func (r *PostgresIPWhitelistRepository) AddEntry(ctx context.Context, userID, cidr, description string) (*entities.IPWhitelistEntry, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ip_whitelist (id, user_id, cidr, description, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, userID, cidr, description, now,
	)
	if err != nil {
		return nil, fmt.Errorf("add ip whitelist entry: %w", err)
	}
	return &entities.IPWhitelistEntry{
		ID:          id,
		UserID:      userID,
		CIDR:        cidr,
		Description: description,
		CreatedAt:   now,
	}, nil
}

func (r *PostgresIPWhitelistRepository) GetEntry(ctx context.Context, id string) (*entities.IPWhitelistEntry, error) {
	var e entities.IPWhitelistEntry
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, cidr, description, created_at FROM ip_whitelist WHERE id = $1`, id,
	).Scan(&e.ID, &e.UserID, &e.CIDR, &e.Description, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("ip entry not found: %w", err)
	}
	return &e, nil
}

func (r *PostgresIPWhitelistRepository) ListByUser(ctx context.Context, userID string) ([]entities.IPWhitelistEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, cidr, description, created_at
		 FROM ip_whitelist WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list ip whitelist: %w", err)
	}
	defer rows.Close()

	var entries []entities.IPWhitelistEntry
	for rows.Next() {
		var e entities.IPWhitelistEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.CIDR, &e.Description, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan ip entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *PostgresIPWhitelistRepository) DeleteEntry(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM ip_whitelist WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete ip entry: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("ip entry not found: %s", id)
	}
	return nil
}

func (r *PostgresIPWhitelistRepository) IsIPAllowed(ctx context.Context, userID, ip string) (bool, error) {
	entries, err := r.ListByUser(ctx, userID)
	if err != nil {
		return false, err
	}
	// No whitelist = all allowed
	if len(entries) == 0 {
		return true, nil
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false, fmt.Errorf("invalid IP: %s", ip)
	}
	for _, entry := range entries {
		_, network, err := net.ParseCIDR(entry.CIDR)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true, nil
		}
	}
	return false, nil
}
