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

const gwRouteSelectCols = `id, gateway_id, group_id, name, path, methods, app_id, app_subdomain, strip_prefix, auth, auth_pool_id, plugins, priority, status, created_at, updated_at`

func scanGatewayRoute(scanner interface{ Scan(dest ...any) error }) (*entities.GatewayRoute, error) {
	var r entities.GatewayRoute
	var methodsStr string
	var pluginsJSON []byte
	var authPoolID *string
	var groupID *string
	var appID *string
	var appSubdomain *string
	err := scanner.Scan(
		&r.ID, &r.GatewayID, &groupID, &r.Name, &r.Path, &methodsStr,
		&appID, &appSubdomain, &r.StripPrefix, &r.Auth,
		&authPoolID, &pluginsJSON, &r.Priority, &r.Status,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if groupID != nil {
		r.GroupID = *groupID
	}
	if appID != nil {
		r.AppID = *appID
	}
	if appSubdomain != nil {
		r.AppSubdomain = *appSubdomain
	}
	if authPoolID != nil {
		r.AuthPoolID = *authPoolID
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

const gwGroupSelectCols = `id, gateway_id, name, app_id, app_subdomain, plugins, created_at, updated_at`

func scanGatewayGroup(scanner interface{ Scan(dest ...any) error }) (*entities.GatewayGroup, error) {
	var g entities.GatewayGroup
	var pluginsJSON []byte
	err := scanner.Scan(
		&g.ID, &g.GatewayID, &g.Name, &g.AppID, &g.AppSubdomain,
		&pluginsJSON, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(pluginsJSON) > 0 {
		json.Unmarshal(pluginsJSON, &g.Plugins)
	}
	if g.Plugins == nil {
		g.Plugins = []entities.GatewayRoutePlugin{}
	}
	return &g, nil
}

// --- Gateway CRUD ---

func (r *PostgresGatewayRepository) CreateGateway(ctx context.Context, userID, projectID, name, slug string) (*entities.Gateway, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO gateways (id, user_id, project_id, name, slug, status, route_count, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, userID, projectID, name, slug, string(entities.GatewayStatusProvisioning), 0, now, now,
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
		ProjectID:  projectID,
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

func (r *PostgresGatewayRepository) GetGatewayByProject(ctx context.Context, projectID string) (*entities.Gateway, error) {
	g, err := scanGateway(r.pool.QueryRow(ctx,
		`SELECT `+gwSelectCols+` FROM gateways WHERE project_id = $1 ORDER BY created_at ASC LIMIT 1`, projectID,
	))
	if err != nil {
		return nil, fmt.Errorf("no gateway found for project: %s", projectID)
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

	var authPoolParam interface{}
	if route.AuthPoolID != "" {
		authPoolParam = route.AuthPoolID
	}
	var groupIDParam interface{}
	if route.GroupID != "" {
		groupIDParam = route.GroupID
	}
	var appIDParam interface{}
	if route.AppID != "" {
		appIDParam = route.AppID
	}
	var appSubdomainParam interface{}
	if route.AppSubdomain != "" {
		appSubdomainParam = route.AppSubdomain
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO gateway_routes (id, gateway_id, group_id, name, path, methods, app_id, app_subdomain, strip_prefix, auth, auth_pool_id, plugins, priority, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		route.ID, route.GatewayID, groupIDParam, route.Name, route.Path, methodsStr,
		appIDParam, appSubdomainParam, route.StripPrefix, string(route.Auth),
		authPoolParam, pluginsJSON, route.Priority, string(route.Status), now, now,
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

	var authPoolParam interface{}
	if route.AuthPoolID != "" {
		authPoolParam = route.AuthPoolID
	}
	var groupIDParam interface{}
	if route.GroupID != "" {
		groupIDParam = route.GroupID
	}
	var appIDParam interface{}
	if route.AppID != "" {
		appIDParam = route.AppID
	}
	var appSubdomainParam interface{}
	if route.AppSubdomain != "" {
		appSubdomainParam = route.AppSubdomain
	}

	ct, err := r.pool.Exec(ctx,
		`UPDATE gateway_routes SET name = $1, path = $2, methods = $3, app_id = $4, app_subdomain = $5,
		 strip_prefix = $6, auth = $7, auth_pool_id = $8, plugins = $9, priority = $10, status = $11, updated_at = $12,
		 group_id = $14
		 WHERE id = $13`,
		route.Name, route.Path, methodsStr, appIDParam, appSubdomainParam,
		route.StripPrefix, string(route.Auth), authPoolParam, pluginsJSON, route.Priority, string(route.Status), now,
		route.ID, groupIDParam,
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

func (r *PostgresGatewayRepository) ClearAuthPoolFromRoutes(ctx context.Context, authPoolID string) ([]string, error) {
	now := time.Now()
	rows, err := r.pool.Query(ctx,
		`UPDATE gateway_routes SET auth_pool_id = NULL, auth = 'none', updated_at = $1 WHERE auth_pool_id = $2 RETURNING gateway_id`,
		now, authPoolID,
	)
	if err != nil {
		return nil, fmt.Errorf("clear auth pool from routes: %w", err)
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

func (r *PostgresGatewayRepository) ListRoutesByAuthPool(ctx context.Context, authPoolID string) ([]entities.GatewayRoute, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+gwRouteSelectCols+` FROM gateway_routes WHERE auth_pool_id = $1 ORDER BY created_at ASC`, authPoolID,
	)
	if err != nil {
		return nil, fmt.Errorf("list routes by auth pool: %w", err)
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

// --- Custom Domain CRUD ---

func (r *PostgresGatewayRepository) AddGatewayDomain(ctx context.Context, gatewayID, userID, domain string) (*entities.GatewayCustomDomain, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO gateway_custom_domains (id, gateway_id, user_id, domain, status, tls_ready, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, gatewayID, userID, domain, string(entities.GatewayCustomDomainStatusPending), false, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "gateway_custom_domains_domain_key") {
			return nil, fmt.Errorf("domain '%s' is already in use", domain)
		}
		return nil, fmt.Errorf("add gateway domain: %w", err)
	}

	return &entities.GatewayCustomDomain{
		ID:         id,
		GatewayID:  gatewayID,
		UserID:     userID,
		Domain:     domain,
		Status:     entities.GatewayCustomDomainStatusPending,
		TLSReady:   false,
		Timestamps: entities.Timestamps{CreatedAt: now, UpdatedAt: now},
	}, nil
}

func (r *PostgresGatewayRepository) ListGatewayDomains(ctx context.Context, gatewayID string) ([]entities.GatewayCustomDomain, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, gateway_id, user_id, domain, status, tls_ready, created_at, updated_at
		 FROM gateway_custom_domains WHERE gateway_id = $1 ORDER BY created_at ASC`, gatewayID,
	)
	if err != nil {
		return nil, fmt.Errorf("list gateway domains: %w", err)
	}
	defer rows.Close()

	var domains []entities.GatewayCustomDomain
	for rows.Next() {
		var d entities.GatewayCustomDomain
		if err := rows.Scan(&d.ID, &d.GatewayID, &d.UserID, &d.Domain, &d.Status, &d.TLSReady, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan gateway domain: %w", err)
		}
		domains = append(domains, d)
	}
	return domains, nil
}

func (r *PostgresGatewayRepository) DeleteGatewayDomain(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM gateway_custom_domains WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete gateway domain: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("gateway domain not found: %s", id)
	}
	return nil
}

func (r *PostgresGatewayRepository) UpdateGatewayDomainStatus(ctx context.Context, id string, status entities.GatewayCustomDomainStatus, tlsReady bool) error {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE gateway_custom_domains SET status = $1, tls_ready = $2, updated_at = $3 WHERE id = $4`,
		string(status), tlsReady, now, id,
	)
	if err != nil {
		return fmt.Errorf("update gateway domain status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("gateway domain not found: %s", id)
	}
	return nil
}

// --- Group CRUD ---

func (r *PostgresGatewayRepository) CreateGroup(ctx context.Context, group *entities.GatewayGroup) (*entities.GatewayGroup, error) {
	group.ID = uuid.New().String()
	now := time.Now()
	group.CreatedAt = now
	group.UpdatedAt = now

	if group.Plugins == nil {
		group.Plugins = []entities.GatewayRoutePlugin{}
	}
	pluginsJSON, _ := json.Marshal(group.Plugins)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO gateway_groups (id, gateway_id, name, app_id, app_subdomain, plugins, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		group.ID, group.GatewayID, group.Name, group.AppID, group.AppSubdomain,
		pluginsJSON, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_gw_groups_gw_name") {
			return nil, fmt.Errorf("group name '%s' already exists in this gateway", group.Name)
		}
		return nil, fmt.Errorf("create group: %w", err)
	}

	return group, nil
}

func (r *PostgresGatewayRepository) GetGroup(ctx context.Context, id string) (*entities.GatewayGroup, error) {
	g, err := scanGatewayGroup(r.pool.QueryRow(ctx,
		`SELECT `+gwGroupSelectCols+` FROM gateway_groups WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("group not found: %s", id)
	}
	return g, nil
}

func (r *PostgresGatewayRepository) ListGroupsByGateway(ctx context.Context, gatewayID string) ([]entities.GatewayGroup, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+gwGroupSelectCols+` FROM gateway_groups WHERE gateway_id = $1 ORDER BY created_at ASC`, gatewayID,
	)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var groups []entities.GatewayGroup
	for rows.Next() {
		g, err := scanGatewayGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, *g)
	}
	return groups, nil
}

func (r *PostgresGatewayRepository) UpdateGroup(ctx context.Context, group *entities.GatewayGroup) (*entities.GatewayGroup, error) {
	now := time.Now()
	if group.Plugins == nil {
		group.Plugins = []entities.GatewayRoutePlugin{}
	}
	pluginsJSON, _ := json.Marshal(group.Plugins)

	ct, err := r.pool.Exec(ctx,
		`UPDATE gateway_groups SET name = $1, app_id = $2, app_subdomain = $3, plugins = $4, updated_at = $5 WHERE id = $6`,
		group.Name, group.AppID, group.AppSubdomain, pluginsJSON, now, group.ID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_gw_groups_gw_name") {
			return nil, fmt.Errorf("group name '%s' already exists in this gateway", group.Name)
		}
		return nil, fmt.Errorf("update group: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("group not found: %s", group.ID)
	}

	group.UpdatedAt = now
	return group, nil
}

func (r *PostgresGatewayRepository) DeleteGroup(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM gateway_groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("group not found: %s", id)
	}
	return nil
}

func (r *PostgresGatewayRepository) StopGroupsByApp(ctx context.Context, appID string) ([]string, error) {
	// Find gateways affected by groups targeting this app, then delete the groups
	rows, err := r.pool.Query(ctx,
		`DELETE FROM gateway_groups WHERE app_id = $1 RETURNING gateway_id`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("stop groups by app: %w", err)
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
