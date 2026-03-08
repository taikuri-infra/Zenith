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

// PostgresDomainRepository is a PostgreSQL-backed DomainRepository.
type PostgresDomainRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresDomainRepository creates a new PostgreSQL DomainRepository.
func NewPostgresDomainRepository(pool *pgxpool.Pool) *PostgresDomainRepository {
	return &PostgresDomainRepository{pool: pool}
}

const domainSelectCols = `id, app_id, user_id, domain, status, tls_ready, created_at, updated_at`

func scanDomain(scanner interface{ Scan(dest ...any) error }) (*entities.CustomDomain, error) {
	var d entities.CustomDomain
	err := scanner.Scan(
		&d.ID, &d.AppID, &d.UserID, &d.Domain, &d.Status,
		&d.TLSReady, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *PostgresDomainRepository) AddDomain(ctx context.Context, appID, userID, domain string) (*entities.CustomDomain, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO custom_domains (id, app_id, user_id, domain, status, tls_ready, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, appID, userID, domain, string(entities.DomainStatusActive), false, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "custom_domains_domain_key") {
			return nil, fmt.Errorf("domain %q already in use", domain)
		}
		return nil, fmt.Errorf("add domain: %w", err)
	}

	return &entities.CustomDomain{
		ID:     id,
		AppID:  appID,
		UserID: userID,
		Domain: domain,
		Status: entities.DomainStatusActive,
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

func (r *PostgresDomainRepository) GetDomain(ctx context.Context, id string) (*entities.CustomDomain, error) {
	d, err := scanDomain(r.pool.QueryRow(ctx,
		`SELECT `+domainSelectCols+` FROM custom_domains WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("domain not found: %s", id)
	}
	return d, nil
}

func (r *PostgresDomainRepository) ListDomainsByApp(ctx context.Context, appID string) ([]entities.CustomDomain, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+domainSelectCols+` FROM custom_domains WHERE app_id = $1 ORDER BY created_at DESC`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	defer rows.Close()

	var domains []entities.CustomDomain
	for rows.Next() {
		d, err := scanDomain(rows)
		if err != nil {
			return nil, fmt.Errorf("scan domain: %w", err)
		}
		domains = append(domains, *d)
	}
	return domains, nil
}

func (r *PostgresDomainRepository) ListDomainsByUser(ctx context.Context, userID string) ([]entities.CustomDomain, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+domainSelectCols+` FROM custom_domains WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	defer rows.Close()

	var domains []entities.CustomDomain
	for rows.Next() {
		d, err := scanDomain(rows)
		if err != nil {
			return nil, fmt.Errorf("scan domain: %w", err)
		}
		domains = append(domains, *d)
	}
	return domains, nil
}

func (r *PostgresDomainRepository) UpdateDomainStatus(ctx context.Context, id string, status entities.DomainStatus, tlsReady bool) error {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE custom_domains SET status = $1, tls_ready = $2, updated_at = $3 WHERE id = $4`,
		string(status), tlsReady, now, id,
	)
	if err != nil {
		return fmt.Errorf("update domain status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("domain not found: %s", id)
	}
	return nil
}

func (r *PostgresDomainRepository) DeleteDomain(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM custom_domains WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete domain: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("domain not found: %s", id)
	}
	return nil
}
