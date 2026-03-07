package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresGatewayRepository is a PostgreSQL-backed GatewayRepository.
type PostgresGatewayRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresGatewayRepository creates a new PostgreSQL GatewayRepository.
func NewPostgresGatewayRepository(pool *pgxpool.Pool) *PostgresGatewayRepository {
	return &PostgresGatewayRepository{pool: pool}
}

const gwSelectCols = `id, user_id, project_id, name, slug, status, route_count, created_at, updated_at`

func scanGateway(scanner interface{ Scan(dest ...any) error }) (*entities.Gateway, error) {
	var g entities.Gateway
	err := scanner.Scan(
		&g.ID, &g.UserID, &g.ProjectID, &g.Name, &g.Slug, &g.Status,
		&g.RouteCount, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

const gwRouteSelectCols = `id, gateway_id, name, path, methods, app_id, app_subdomain, strip_prefix, auth, plugins, priority, status, created_at, updated_at`

func scanGatewayRoute(scanner interface{ Scan(dest ...any) error }) (*entities.GatewayRoute, error) {
	var r entities.GatewayRoute
	var methodsStr string
	var pluginsJSON []byte
	err := scanner.Scan(
		&r.ID, &r.GatewayID, &r.Name, &r.Path, &methodsStr,
		&r.AppID, &r.AppSubdomain, &r.StripPrefix, &r.Auth,
		&pluginsJSON, &r.Priority, &r.Status,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Methods = strings.Split(methodsStr, ",")
	if len(pluginsJSON) > 0 {
		json.Unmarshal(pluginsJSON, &r.Plugins)
	}
	if r.Plugins == nil {
		r.Plugins = []entities.GatewayRoutePlugin{}
	}
	return &r, nil
}

// --- Gateway CRUD ---

func (r *PostgresGatewayRepository) CreateGateway(ctx context.Context, userID, name, slug string) (*entities.Gateway, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO gateways (id, user_id, name, slug, status, route_count, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, userID, name, slug, string(entities.GatewayStatusProvisioning), 0, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_gateways_slug") {
			return nil, fmt.Errorf("gateway slug '%s' already exists", slug)
		}
		return nil, fmt.Errorf("create gateway: %w", err)
	}

	return &entities.Gateway{
		ID:         id,
		UserID:     userID,
		Name:       name,
		Slug:       slug,
		Status:     entities.GatewayStatusProvisioning,
		RouteCount: 0,
		Timestamps: entities.Timestamps{CreatedAt: now, UpdatedAt: now},
	}, nil
}

func (r *PostgresGatewayRepository) GetGateway(ctx context.Context, id string) (*entities.Gateway, error) {
	g, err := scanGateway(r.pool.QueryRow(ctx,
		`SELECT `+gwSelectCols+` FROM gateways WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("gateway not found: %s", id)
	}
	return g, nil
}

func (r *PostgresGatewayRepository) GetGatewayBySlug(ctx context.Context, slug string) (*entities.Gateway, error) {
	g, err := scanGateway(r.pool.QueryRow(ctx,
		`SELECT `+gwSelectCols+` FROM gateways WHERE slug = $1`, slug,
	))
	if err != nil {
		return nil, fmt.Errorf("gateway not found: %s", slug)
	}
	return g, nil
}

func (r *PostgresGatewayRepository) ListGatewaysByUser(ctx context.Context, userID string) ([]entities.Gateway, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+gwSelectCols+` FROM gateways WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list gateways: %w", err)
	}
	defer rows.Close()

	var gws []entities.Gateway
	for rows.Next() {
		g, err := scanGateway(rows)
		if err != nil {
			return nil, fmt.Errorf("scan gateway: %w", err)
		}
		gws = append(gws, *g)
	}
	return gws, nil
}

func (r *PostgresGatewayRepository) ListGatewaysByProject(ctx context.Context, projectID string) ([]entities.Gateway, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+gwSelectCols+` FROM gateways WHERE project_id = $1 ORDER BY created_at DESC`, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list gateways by project: %w", err)
	}
	defer rows.Close()

	var gws []entities.Gateway
	for rows.Next() {
		g, err := scanGateway(rows)
		if err != nil {
			return nil, fmt.Errorf("scan gateway: %w", err)
		}
		gws = append(gws, *g)
	}
	return gws, nil
}

func (r *PostgresGatewayRepository) UpdateGateway(ctx context.Context, id, name string) (*entities.Gateway, error) {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE gateways SET name = $1, updated_at = $2 WHERE id = $3`,
		name, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update gateway: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("gateway not found: %s", id)
	}
	return r.GetGateway(ctx, id)
}

func (r *PostgresGatewayRepository) DeleteGateway(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM gateways WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete gateway: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("gateway not found: %s", id)
	}
	return nil
}

func (r *PostgresGatewayRepository) CountGatewaysByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM gateways WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count gateways: %w", err)
	}
	return count, nil
}

func (r *PostgresGatewayRepository) UpdateGatewayStatus(ctx context.Context, id string, status entities.GatewayStatus) error {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE gateways SET status = $1, updated_at = $2 WHERE id = $3`,
		string(status), now, id,
	)
	if err != nil {
		return fmt.Errorf("update gateway status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("gateway not found: %s", id)
	}
	return nil
}

// --- Route CRUD ---

func (r *PostgresGatewayRepository) CreateRoute(ctx context.Context, route *entities.GatewayRoute) (*entities.GatewayRoute, error) {
	route.ID = uuid.New().String()
	now := time.Now()
	route.CreatedAt = now
	route.UpdatedAt = now

	if route.Status == "" {
		route.Status = entities.GatewayRouteStatusActive
	}
	if route.Auth == "" {
		route.Auth = entities.GatewayRouteAuthNone
	}
	if route.Plugins == nil {
		route.Plugins = []entities.GatewayRoutePlugin{}
	}

	pluginsJSON, _ := json.Marshal(route.Plugins)
	methodsStr := strings.Join(route.Methods, ",")

	_, err := r.pool.Exec(ctx,
		`INSERT INTO gateway_routes (id, gateway_id, name, path, methods, app_id, app_subdomain, strip_prefix, auth, plugins, priority, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		route.ID, route.GatewayID, route.Name, route.Path, methodsStr,
		route.AppID, route.AppSubdomain, route.StripPrefix, string(route.Auth),
		pluginsJSON, route.Priority, string(route.Status), now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_gw_routes_gw_name") {
			return nil, fmt.Errorf("route name '%s' already exists in this gateway", route.Name)
		}
		return nil, fmt.Errorf("create route: %w", err)
	}

	// Update route_count
	r.pool.Exec(ctx,
		`UPDATE gateways SET route_count = (SELECT COUNT(*) FROM gateway_routes WHERE gateway_id = $1), updated_at = $2 WHERE id = $1`,
		route.GatewayID, now,
	)

	return route, nil
}

func (r *PostgresGatewayRepository) GetRoute(ctx context.Context, id string) (*entities.GatewayRoute, error) {
	rt, err := scanGatewayRoute(r.pool.QueryRow(ctx,
		`SELECT `+gwRouteSelectCols+` FROM gateway_routes WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("route not found: %s", id)
	}
	return rt, nil
}

func (r *PostgresGatewayRepository) ListRoutesByGateway(ctx context.Context, gatewayID string) ([]entities.GatewayRoute, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+gwRouteSelectCols+` FROM gateway_routes WHERE gateway_id = $1 ORDER BY priority DESC, created_at ASC`, gatewayID,
	)
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	defer rows.Close()

	var routes []entities.GatewayRoute
	for rows.Next() {
		rt, err := scanGatewayRoute(rows)
		if err != nil {
			return nil, fmt.Errorf("scan route: %w", err)
		}
		routes = append(routes, *rt)
	}
	return routes, nil
}

func (r *PostgresGatewayRepository) ListActiveRoutesByGateway(ctx context.Context, gatewayID string) ([]entities.GatewayRoute, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+gwRouteSelectCols+` FROM gateway_routes WHERE gateway_id = $1 AND status = 'active' ORDER BY priority DESC, created_at ASC`, gatewayID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active routes: %w", err)
	}
	defer rows.Close()

	var routes []entities.GatewayRoute
	for rows.Next() {
		rt, err := scanGatewayRoute(rows)
		if err != nil {
			return nil, fmt.Errorf("scan route: %w", err)
		}
		routes = append(routes, *rt)
	}
	return routes, nil
}

func (r *PostgresGatewayRepository) UpdateRoute(ctx context.Context, route *entities.GatewayRoute) (*entities.GatewayRoute, error) {
	now := time.Now()
	pluginsJSON, _ := json.Marshal(route.Plugins)
	methodsStr := strings.Join(route.Methods, ",")

	ct, err := r.pool.Exec(ctx,
		`UPDATE gateway_routes SET name = $1, path = $2, methods = $3, app_id = $4, app_subdomain = $5,
		 strip_prefix = $6, auth = $7, plugins = $8, priority = $9, status = $10, updated_at = $11
		 WHERE id = $12`,
		route.Name, route.Path, methodsStr, route.AppID, route.AppSubdomain,
		route.StripPrefix, string(route.Auth), pluginsJSON, route.Priority, string(route.Status), now,
		route.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update route: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("route not found: %s", route.ID)
	}

	route.UpdatedAt = now
	return route, nil
}

func (r *PostgresGatewayRepository) DeleteRoute(ctx context.Context, id string) error {
	// Get gateway_id before deleting for route_count update
	var gatewayID string
	err := r.pool.QueryRow(ctx, `SELECT gateway_id FROM gateway_routes WHERE id = $1`, id).Scan(&gatewayID)
	if err != nil {
		return fmt.Errorf("route not found: %s", id)
	}

	ct, err := r.pool.Exec(ctx, `DELETE FROM gateway_routes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete route: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("route not found: %s", id)
	}

	// Update route_count
	now := time.Now()
	r.pool.Exec(ctx,
		`UPDATE gateways SET route_count = (SELECT COUNT(*) FROM gateway_routes WHERE gateway_id = $1), updated_at = $2 WHERE id = $1`,
		gatewayID, now,
	)

	return nil
}

func (r *PostgresGatewayRepository) CountRoutesByGateway(ctx context.Context, gatewayID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM gateway_routes WHERE gateway_id = $1`, gatewayID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count routes by gateway: %w", err)
	}
	return count, nil
}

func (r *PostgresGatewayRepository) CountRoutesByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM gateway_routes gr JOIN gateways g ON gr.gateway_id = g.id WHERE g.user_id = $1`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count routes by user: %w", err)
	}
	return count, nil
}

func (r *PostgresGatewayRepository) StopRoutesByApp(ctx context.Context, appID string) ([]string, error) {
	now := time.Now()
	rows, err := r.pool.Query(ctx,
		`UPDATE gateway_routes SET status = 'stopped', updated_at = $1 WHERE app_id = $2 AND status = 'active' RETURNING gateway_id`,
		now, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("stop routes by app: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var gwIDs []string
	for rows.Next() {
		var gwID string
		if err := rows.Scan(&gwID); err != nil {
			continue
		}
		if !seen[gwID] {
			seen[gwID] = true
			gwIDs = append(gwIDs, gwID)
		}
	}
	return gwIDs, nil
}
