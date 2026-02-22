package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Compile-time interface check.
var _ AdminRepository = (*PostgresAdminRepository)(nil)

// PostgresAdminRepository persists admin data in PostgreSQL.
type PostgresAdminRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAdminRepository creates a new PostgresAdminRepository.
func NewPostgresAdminRepository(pool *pgxpool.Pool) *PostgresAdminRepository {
	return &PostgresAdminRepository{pool: pool}
}

func (s *PostgresAdminRepository) GetSettings(ctx context.Context) (*entities.PlatformSettings, error) {
	var ps entities.PlatformSettings
	err := s.pool.QueryRow(ctx,
		`SELECT platform_name, base_domain, provider, default_region, region_label, auto_backups, retention_days
		 FROM platform_settings WHERE id = 1`,
	).Scan(&ps.PlatformName, &ps.BaseDomain, &ps.Provider, &ps.DefaultRegion, &ps.RegionLabel, &ps.AutoBackups, &ps.RetentionDays)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &entities.PlatformSettings{PlatformName: "Zenith"}, nil
		}
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return &ps, nil
}

func (s *PostgresAdminRepository) UpdateSettings(ctx context.Context, update *entities.PlatformSettings) (*entities.PlatformSettings, error) {
	// Upsert: read current, merge non-empty fields, write back
	current, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	if update.PlatformName != "" {
		current.PlatformName = update.PlatformName
	}
	if update.BaseDomain != "" {
		current.BaseDomain = update.BaseDomain
	}
	if update.Provider != "" {
		current.Provider = update.Provider
	}
	if update.DefaultRegion != "" {
		current.DefaultRegion = update.DefaultRegion
	}
	if update.RegionLabel != "" {
		current.RegionLabel = update.RegionLabel
	}
	current.AutoBackups = update.AutoBackups
	if update.RetentionDays > 0 {
		current.RetentionDays = update.RetentionDays
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO platform_settings (id, platform_name, base_domain, provider, default_region, region_label, auto_backups, retention_days)
		 VALUES (1, $1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET
		   platform_name = EXCLUDED.platform_name,
		   base_domain = EXCLUDED.base_domain,
		   provider = EXCLUDED.provider,
		   default_region = EXCLUDED.default_region,
		   region_label = EXCLUDED.region_label,
		   auto_backups = EXCLUDED.auto_backups,
		   retention_days = EXCLUDED.retention_days`,
		current.PlatformName, current.BaseDomain, current.Provider, current.DefaultRegion,
		current.RegionLabel, current.AutoBackups, current.RetentionDays,
	)
	if err != nil {
		return nil, fmt.Errorf("update settings: %w", err)
	}

	return current, nil
}

func (s *PostgresAdminRepository) ListModules(ctx context.Context) ([]entities.Module, error) {
	rows, err := s.pool.Query(ctx, `SELECT name, installed, latest, status, description FROM modules ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list modules: %w", err)
	}
	defer rows.Close()

	var modules []entities.Module
	for rows.Next() {
		var m entities.Module
		if err := rows.Scan(&m.Name, &m.Installed, &m.Latest, &m.Status, &m.Description); err != nil {
			return nil, fmt.Errorf("scan module: %w", err)
		}
		modules = append(modules, m)
	}
	return modules, rows.Err()
}

func (s *PostgresAdminRepository) GetModule(ctx context.Context, name string) (*entities.Module, error) {
	var m entities.Module
	err := s.pool.QueryRow(ctx,
		`SELECT name, installed, latest, status, description FROM modules WHERE name = $1`, name,
	).Scan(&m.Name, &m.Installed, &m.Latest, &m.Status, &m.Description)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("module %s not found", name)
		}
		return nil, fmt.Errorf("get module: %w", err)
	}
	return &m, nil
}

func (s *PostgresAdminRepository) UpdateModule(ctx context.Context, name string) (*entities.Module, error) {
	var m entities.Module
	err := s.pool.QueryRow(ctx,
		`UPDATE modules SET installed = latest, status = 'up_to_date' WHERE name = $1
		 RETURNING name, installed, latest, status, description`, name,
	).Scan(&m.Name, &m.Installed, &m.Latest, &m.Status, &m.Description)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("module %s not found", name)
		}
		return nil, fmt.Errorf("update module: %w", err)
	}
	return &m, nil
}

func (s *PostgresAdminRepository) ListAuditLog(ctx context.Context, limit, offset int) ([]entities.AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx,
		`SELECT time, actor, action, cluster FROM audit_log ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()

	var entries []entities.AuditEntry
	for rows.Next() {
		var e entities.AuditEntry
		if err := rows.Scan(&e.Time, &e.Actor, &e.Action, &e.Cluster); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []entities.AuditEntry{}
	}
	return entries, rows.Err()
}

func (s *PostgresAdminRepository) AddAuditEntry(ctx context.Context, entry entities.AuditEntry) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_log (time, actor, action, cluster) VALUES ($1, $2, $3, $4)`,
		entry.Time, entry.Actor, entry.Action, entry.Cluster,
	)
	if err != nil {
		return fmt.Errorf("add audit entry: %w", err)
	}
	return nil
}

func (s *PostgresAdminRepository) GetPlatformUpdate(_ context.Context) (*entities.PlatformUpdate, error) {
	// Platform update info is not DB-driven yet — return static values.
	return &entities.PlatformUpdate{
		Version:         "v1.3.0",
		Current:         "v1.2.1",
		ReleasedAt:      "February 10, 2026",
		Features:        []string{"MongoDB support", "Cloud Connections (AWS/GCP/Azure VPN)", "GitOps mode (zen export/apply)", "Auto-generated documentation"},
		BreakingChanges: false,
	}, nil
}

func (s *PostgresAdminRepository) ListUpdateHistory(ctx context.Context) ([]entities.UpdateHistoryEntry, error) {
	rows, err := s.pool.Query(ctx, `SELECT version, date, status FROM update_history ORDER BY date DESC`)
	if err != nil {
		return nil, fmt.Errorf("list update history: %w", err)
	}
	defer rows.Close()

	var entries []entities.UpdateHistoryEntry
	for rows.Next() {
		var e entities.UpdateHistoryEntry
		if err := rows.Scan(&e.Version, &e.Date, &e.Status); err != nil {
			return nil, fmt.Errorf("scan update history: %w", err)
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []entities.UpdateHistoryEntry{}
	}
	return entries, rows.Err()
}
