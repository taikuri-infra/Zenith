package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAutoscaleRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAutoscaleRepository(pool *pgxpool.Pool) *PostgresAutoscaleRepository {
	return &PostgresAutoscaleRepository{pool: pool}
}

func (r *PostgresAutoscaleRepository) SaveNode(ctx context.Context, node *entities.HetznerNode) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO autoscaler_nodes (server_id, name, ip, status, server_type, cpu_cores, ram_mb, monthly_cost, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (server_id) DO UPDATE SET
		 name = $2, ip = $3, status = $4, server_type = $5, cpu_cores = $6, ram_mb = $7, monthly_cost = $8`,
		node.ServerID, node.Name, node.IP, node.Status, node.ServerType,
		node.CPUCores, node.RAMMB, node.MonthlyCost, node.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save node: %w", err)
	}
	return nil
}

func (r *PostgresAutoscaleRepository) DeleteNode(ctx context.Context, serverID int64) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM autoscaler_nodes WHERE server_id = $1`, serverID)
	if err != nil {
		return fmt.Errorf("delete node: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("node not found: %d", serverID)
	}
	return nil
}

func (r *PostgresAutoscaleRepository) ListNodes(ctx context.Context) ([]entities.HetznerNode, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT server_id, name, ip, status, server_type, cpu_cores, ram_mb, monthly_cost, created_at
		 FROM autoscaler_nodes ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []entities.HetznerNode
	for rows.Next() {
		var n entities.HetznerNode
		if err := rows.Scan(&n.ServerID, &n.Name, &n.IP, &n.Status, &n.ServerType,
			&n.CPUCores, &n.RAMMB, &n.MonthlyCost, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (r *PostgresAutoscaleRepository) LogScaleEvent(ctx context.Context, event *entities.AutoscaleEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO autoscale_events (id, timestamp, action, old_count, new_count, reason, server_name)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		event.ID, event.Timestamp, string(event.Action), event.OldCount, event.NewCount,
		event.Reason, event.ServerName,
	)
	if err != nil {
		return fmt.Errorf("log scale event: %w", err)
	}
	return nil
}

func (r *PostgresAutoscaleRepository) ListScaleEvents(ctx context.Context, limit int) ([]entities.AutoscaleEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, timestamp, action, old_count, new_count, reason, server_name
		 FROM autoscale_events ORDER BY timestamp DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list scale events: %w", err)
	}
	defer rows.Close()

	var events []entities.AutoscaleEvent
	for rows.Next() {
		var e entities.AutoscaleEvent
		var action string
		if err := rows.Scan(&e.ID, &e.Timestamp, &action, &e.OldCount, &e.NewCount,
			&e.Reason, &e.ServerName); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.Action = entities.AutoscaleAction(action)
		events = append(events, e)
	}
	return events, nil
}

func (r *PostgresAutoscaleRepository) GetStatus(ctx context.Context) (*entities.AutoscalerStatus, error) {
	var s entities.AutoscalerStatus
	err := r.pool.QueryRow(ctx,
		`SELECT enabled, node_count, min_nodes, max_nodes, cpu_percent, ram_percent,
		 budget_cap_eur, budget_used_eur, last_scale_up, last_scale_down, last_check_at
		 FROM autoscaler_status WHERE id = 1`,
	).Scan(&s.Enabled, &s.NodeCount, &s.MinNodes, &s.MaxNodes, &s.CPUPercent, &s.RAMPercent,
		&s.BudgetCapEUR, &s.BudgetUsedEUR, &s.LastScaleUp, &s.LastScaleDown, &s.LastCheckAt)
	if err != nil {
		// Not found = return default
		return &entities.AutoscalerStatus{}, nil
	}
	return &s, nil
}

func (r *PostgresAutoscaleRepository) UpdateStatus(ctx context.Context, status *entities.AutoscalerStatus) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO autoscaler_status (id, enabled, node_count, min_nodes, max_nodes, cpu_percent, ram_percent,
		 budget_cap_eur, budget_used_eur, last_scale_up, last_scale_down, last_check_at)
		 VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (id) DO UPDATE SET
		 enabled = $1, node_count = $2, min_nodes = $3, max_nodes = $4, cpu_percent = $5, ram_percent = $6,
		 budget_cap_eur = $7, budget_used_eur = $8, last_scale_up = $9, last_scale_down = $10, last_check_at = $11`,
		status.Enabled, status.NodeCount, status.MinNodes, status.MaxNodes,
		status.CPUPercent, status.RAMPercent, status.BudgetCapEUR, status.BudgetUsedEUR,
		status.LastScaleUp, status.LastScaleDown, status.LastCheckAt,
	)
	if err != nil {
		return fmt.Errorf("update autoscaler status: %w", err)
	}
	return nil
}
