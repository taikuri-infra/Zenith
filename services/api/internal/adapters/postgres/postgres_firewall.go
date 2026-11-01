package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FirewallRuleRepository is a PostgreSQL-backed ports.FirewallRuleRepository.
type FirewallRuleRepository struct {
	pool *pgxpool.Pool
}

// NewFirewallRuleRepository creates a PostgreSQL firewall rule store.
func NewFirewallRuleRepository(pool *pgxpool.Pool) *FirewallRuleRepository {
	return &FirewallRuleRepository{pool: pool}
}

const firewallCols = `id, user_id, app_id, name, direction, action, enabled, priority, config, created_at, updated_at`

func scanFirewallRule(s interface{ Scan(dest ...any) error }) (*entities.NetworkPolicyRule, error) {
	var r entities.NetworkPolicyRule
	var cfg []byte
	if err := s.Scan(&r.ID, &r.UserID, &r.AppID, &r.Name, &r.Direction, &r.Action, &r.Enabled, &r.Priority, &cfg, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(cfg, &r.Config); err != nil {
		return nil, fmt.Errorf("decode firewall config: %w", err)
	}
	return &r, nil
}

func (r *FirewallRuleRepository) list(ctx context.Context, where string, args ...any) ([]entities.NetworkPolicyRule, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+firewallCols+` FROM firewall_rules `+where+` ORDER BY priority, created_at`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entities.NetworkPolicyRule{}
	for rows.Next() {
		rule, err := scanFirewallRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rule)
	}
	return out, rows.Err()
}

func (r *FirewallRuleRepository) ListByApp(ctx context.Context, appID string) ([]entities.NetworkPolicyRule, error) {
	return r.list(ctx, `WHERE app_id = $1`, appID)
}

func (r *FirewallRuleRepository) ListAll(ctx context.Context) ([]entities.NetworkPolicyRule, error) {
	return r.list(ctx, ``)
}

func (r *FirewallRuleRepository) Get(ctx context.Context, id string) (*entities.NetworkPolicyRule, error) {
	return scanFirewallRule(r.pool.QueryRow(ctx, `SELECT `+firewallCols+` FROM firewall_rules WHERE id = $1`, id))
}

func (r *FirewallRuleRepository) Create(ctx context.Context, rule *entities.NetworkPolicyRule) error {
	cfg, err := json.Marshal(rule.Config)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO firewall_rules (`+firewallCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		rule.ID, rule.UserID, rule.AppID, rule.Name, rule.Direction, rule.Action, rule.Enabled, rule.Priority, cfg, rule.CreatedAt, rule.UpdatedAt)
	return err
}

func (r *FirewallRuleRepository) Update(ctx context.Context, rule *entities.NetworkPolicyRule) error {
	cfg, err := json.Marshal(rule.Config)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE firewall_rules SET name=$2, direction=$3, action=$4, enabled=$5, priority=$6, config=$7, updated_at=$8 WHERE id=$1`,
		rule.ID, rule.Name, rule.Direction, rule.Action, rule.Enabled, rule.Priority, cfg, rule.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("firewall rule not found")
	}
	return nil
}

func (r *FirewallRuleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM firewall_rules WHERE id = $1`, id)
	return err
}
