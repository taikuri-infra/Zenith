package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresBrandingRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBrandingRepository(pool *pgxpool.Pool) *PostgresBrandingRepository {
	return &PostgresBrandingRepository{pool: pool}
}

// --- DPA ---

func (r *PostgresBrandingRepository) GetDPA(ctx context.Context, userID string) (*entities.DPARecord, error) {
	var d entities.DPARecord
	var status string
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, status, signed_by, signed_at, ip_address FROM dpa_records WHERE user_id = $1`, userID,
	).Scan(&d.UserID, &status, &d.SignedBy, &d.SignedAt, &d.IPAddress)
	if err != nil {
		// Not found = unsigned
		return &entities.DPARecord{
			UserID: userID,
			Status: entities.DPAUnsigned,
		}, nil
	}
	d.Status = entities.DPAStatus(status)
	return &d, nil
}

func (r *PostgresBrandingRepository) SignDPA(ctx context.Context, userID, signedBy, ipAddress string) (*entities.DPARecord, error) {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO dpa_records (user_id, status, signed_by, signed_at, ip_address)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id) DO UPDATE SET status = $2, signed_by = $3, signed_at = $4, ip_address = $5`,
		userID, string(entities.DPASigned), signedBy, now, ipAddress,
	)
	if err != nil {
		return nil, fmt.Errorf("sign dpa: %w", err)
	}
	return &entities.DPARecord{
		UserID:    userID,
		Status:    entities.DPASigned,
		SignedBy:  signedBy,
		SignedAt:  now,
		IPAddress: ipAddress,
	}, nil
}

// --- Branding ---

func (r *PostgresBrandingRepository) GetBranding(ctx context.Context, userID string) (*entities.BrandingConfig, error) {
	var b entities.BrandingConfig
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, company_name, logo_url, primary_color, dashboard_domain, domain_verified, hide_branding, updated_at
		 FROM branding_configs WHERE user_id = $1`, userID,
	).Scan(&b.UserID, &b.CompanyName, &b.LogoURL, &b.PrimaryColor, &b.DashboardDomain,
		&b.DomainVerified, &b.HideBranding, &b.UpdatedAt)
	if err != nil {
		// Not found = empty config
		return &entities.BrandingConfig{UserID: userID}, nil
	}
	return &b, nil
}

func (r *PostgresBrandingRepository) UpdateBranding(ctx context.Context, userID string, companyName, logoURL, primaryColor *string, hideBranding *bool) (*entities.BrandingConfig, error) {
	now := time.Now()
	// Ensure row exists
	r.pool.Exec(ctx,
		`INSERT INTO branding_configs (user_id, updated_at) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`,
		userID, now,
	)

	sets := []string{"updated_at = $1"}
	args := []interface{}{now}
	argIdx := 2

	if companyName != nil {
		sets = append(sets, fmt.Sprintf("company_name = $%d", argIdx))
		args = append(args, *companyName)
		argIdx++
	}
	if logoURL != nil {
		sets = append(sets, fmt.Sprintf("logo_url = $%d", argIdx))
		args = append(args, *logoURL)
		argIdx++
	}
	if primaryColor != nil {
		sets = append(sets, fmt.Sprintf("primary_color = $%d", argIdx))
		args = append(args, *primaryColor)
		argIdx++
	}
	if hideBranding != nil {
		sets = append(sets, fmt.Sprintf("hide_branding = $%d", argIdx))
		args = append(args, *hideBranding)
		argIdx++
	}

	args = append(args, userID)
	query := fmt.Sprintf("UPDATE branding_configs SET %s WHERE user_id = $%d",
		joinStrings(sets, ", "), argIdx)

	r.pool.Exec(ctx, query, args...)
	return r.GetBranding(ctx, userID)
}

func (r *PostgresBrandingRepository) SetDashboardDomain(ctx context.Context, userID, domain string) (*entities.BrandingConfig, error) {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO branding_configs (user_id, dashboard_domain, updated_at) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE SET dashboard_domain = $2, domain_verified = false, updated_at = $3`,
		userID, domain, now,
	)
	if err != nil {
		return nil, fmt.Errorf("set dashboard domain: %w", err)
	}
	return r.GetBranding(ctx, userID)
}
