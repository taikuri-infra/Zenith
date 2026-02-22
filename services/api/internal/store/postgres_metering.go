package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Compile-time interface check.
var _ MeteringRepository = (*PostgresMeteringRepository)(nil)

// PostgresMeteringRepository persists resource usage in PostgreSQL.
type PostgresMeteringRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresMeteringRepository creates a new PostgresMeteringRepository.
func NewPostgresMeteringRepository(pool *pgxpool.Pool) *PostgresMeteringRepository {
	return &PostgresMeteringRepository{pool: pool}
}

func (r *PostgresMeteringRepository) RecordUsage(ctx context.Context, input *dto.MeteringInput) (*entities.ResourceUsage, error) {
	id := uuid.New().String()
	var entry entities.ResourceUsage

	err := r.pool.QueryRow(ctx,
		`INSERT INTO resource_usage (id, customer_id, cpu_cores, ram_gb, s3_tb, db_storage_gb, volume_gb, lb_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, customer_id, cpu_cores, ram_gb, s3_tb, db_storage_gb, volume_gb, lb_count, recorded_at`,
		id, input.CustomerID, input.CPUCores, input.RAMGB, input.S3TB, input.DBStorageGB, input.VolumeGB, input.LBCount,
	).Scan(&entry.ID, &entry.CustomerID, &entry.CPUCores, &entry.RAMGB, &entry.S3TB,
		&entry.DBStorageGB, &entry.VolumeGB, &entry.LBCount, &entry.RecordedAt)
	if err != nil {
		return nil, fmt.Errorf("insert resource usage: %w", err)
	}

	return &entry, nil
}

func (r *PostgresMeteringRepository) GetLatestUsage(ctx context.Context, customerID string) (*entities.ResourceUsage, error) {
	var entry entities.ResourceUsage
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, cpu_cores, ram_gb, s3_tb, db_storage_gb, volume_gb, lb_count, recorded_at
		 FROM resource_usage WHERE customer_id = $1 ORDER BY recorded_at DESC LIMIT 1`, customerID,
	).Scan(&entry.ID, &entry.CustomerID, &entry.CPUCores, &entry.RAMGB, &entry.S3TB,
		&entry.DBStorageGB, &entry.VolumeGB, &entry.LBCount, &entry.RecordedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no usage data found")
		}
		return nil, fmt.Errorf("get latest usage: %w", err)
	}
	return &entry, nil
}

func (r *PostgresMeteringRepository) GetUsageHistory(ctx context.Context, customerID string, days int) ([]dto.UsageHistoryEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT date_trunc('day', recorded_at)::date AS date,
		        AVG(cpu_cores), MAX(cpu_cores),
		        AVG(ram_gb), MAX(ram_gb),
		        MAX(db_storage_gb), MAX(volume_gb), MAX(lb_count)
		 FROM resource_usage
		 WHERE customer_id = $1 AND recorded_at >= now() - ($2 || ' days')::interval
		 GROUP BY date_trunc('day', recorded_at)::date
		 ORDER BY date`, customerID, fmt.Sprintf("%d", days))
	if err != nil {
		return nil, fmt.Errorf("get usage history: %w", err)
	}
	defer rows.Close()

	var result []dto.UsageHistoryEntry
	for rows.Next() {
		var e dto.UsageHistoryEntry
		var date string
		if err := rows.Scan(&date, &e.CPUAvg, &e.CPUMax, &e.RAMAvg, &e.RAMMax,
			&e.DBStorageGB, &e.VolumeGB, &e.LBCount); err != nil {
			return nil, fmt.Errorf("scan usage history: %w", err)
		}
		e.Date = date
		result = append(result, e)
	}
	if result == nil {
		result = []dto.UsageHistoryEntry{}
	}
	return result, rows.Err()
}

func (r *PostgresMeteringRepository) GetPlatformUsageSummary(ctx context.Context) (*dto.PlatformUsageSummary, error) {
	var summary dto.PlatformUsageSummary
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(cpu_cores), 0),
		        COALESCE(SUM(ram_gb), 0),
		        COALESCE(SUM(db_storage_gb + volume_gb), 0),
		        COUNT(*)
		 FROM (
		     SELECT DISTINCT ON (customer_id) cpu_cores, ram_gb, db_storage_gb, volume_gb
		     FROM resource_usage
		     ORDER BY customer_id, recorded_at DESC
		 ) latest`,
	).Scan(&summary.TotalCPU, &summary.TotalRAM, &summary.TotalStorage, &summary.CustomersReporting)
	if err != nil {
		return nil, fmt.Errorf("get platform usage summary: %w", err)
	}
	return &summary, nil
}
