package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// GatewayService handles gateway CRUD and K8s CRD synchronization.
type GatewayService struct {
	gwRepo       ports.GatewayRepository
	appRepo      ports.AppRepository
	planRepo     ports.UserPlanRepository
	authPoolRepo ports.AuthPoolRepository
	k8sClient    k8sclient.Client
	gwDomain     string // e.g. "gw.stage.freezenith.com"
	namespace    string // e.g. "zenith-apps"
}

// SetAuthPoolRepo wires the auth pool repo (breaks import cycle).
func (s *GatewayService) SetAuthPoolRepo(repo ports.AuthPoolRepository) {
	s.authPoolRepo = repo
}

// NewGatewayService creates a new GatewayService.
func NewGatewayService(
	gwRepo ports.GatewayRepository,
	appRepo ports.AppRepository,
	planRepo ports.UserPlanRepository,
	k8sClient k8sclient.Client,
	gwDomain string,
	namespace string,
) *GatewayService {
	return &GatewayService{
		gwRepo:    gwRepo,
		appRepo:   appRepo,
		planRepo:  planRepo,
		k8sClient: k8sClient,
		gwDomain:  gwDomain,
		namespace: namespace,
	}
}

var slugRegexp = regexp.MustCompile(`[^a-z0-9-]`)

// slugify generates a URL-safe slug from a name.
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugRegexp.ReplaceAllString(s, "-")
	// Collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if s == "" {
		s = "gateway"
	}
	return s
}

// CreateGateway provisions a new API gateway.
func (s *GatewayService) CreateGateway(ctx context.Context, userID, projectID, name string) (*entities.Gateway, error) {
	slug := slugify(name)

	// Check slug uniqueness (also check app subdomains to prevent DNS hijacking)
	if _, err := s.gwRepo.GetGatewayBySlug(ctx, slug); err == nil {
		return nil, fmt.Errorf("gateway slug '%s' already exists", slug)
	}
	if _, err := s.appRepo.GetAppBySubdomain(ctx, slug); err == nil {
		return nil, fmt.Errorf("slug '%s' conflicts with an existing app subdomain", slug)
	}

	// Create DB record
	gw, err := s.gwRepo.CreateGateway(ctx, userID, projectID, name, slug)
	if err != nil {
		return nil, err
	}

	gw.Endpoint = fmt.Sprintf("https://%s.%s", slug, s.gwDomain)

	// Create K8s resources
	if err := s.ensureExternalNameBridge(ctx); err != nil {
		slog.Warn("failed to ensure ExternalName bridge", "error", err)
	}

	if err := s.createIngressRoute(ctx, gw); err != nil {
		slog.Warn("failed to create IngressRoute", "slug", slug, "error", err)
		s.gwRepo.UpdateGatewayStatus(ctx, gw.ID, entities.GatewayStatusError)
		gw.Status = entities.GatewayStatusError
		return gw, nil
	}

	// Create empty ApisixRoute
	if err := s.syncApisixRoute(ctx, gw, nil, nil); err != nil {
		slog.Warn("failed to create ApisixRoute", "slug", slug, "error", err)
	}

	s.gwRepo.UpdateGatewayStatus(ctx, gw.ID, entities.GatewayStatusActive)
	gw.Status = entities.GatewayStatusActive
	return gw, nil
}

// GetGateway returns a gateway with its endpoint populated.
func (s *GatewayService) GetGateway(ctx context.Context, id string) (*entities.Gateway, error) {
	gw, err := s.gwRepo.GetGateway(ctx, id)
	if err != nil {
		return nil, err
	}
	gw.Endpoint = fmt.Sprintf("https://%s.%s", gw.Slug, s.gwDomain)
	return gw, nil
}

// ListGateways returns all gateways for a user with endpoints populated.
func (s *GatewayService) ListGateways(ctx context.Context, userID string) ([]entities.Gateway, error) {
	gws, err := s.gwRepo.ListGatewaysByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range gws {
		gws[i].Endpoint = fmt.Sprintf("https://%s.%s", gws[i].Slug, s.gwDomain)
	}
	return gws, nil
}

// UpdateGateway updates the gateway name.
func (s *GatewayService) UpdateGateway(ctx context.Context, id, name string) (*entities.Gateway, error) {
	gw, err := s.gwRepo.UpdateGateway(ctx, id, name)
	if err != nil {
		return nil, err
	}
	gw.Endpoint = fmt.Sprintf("https://%s.%s", gw.Slug, s.gwDomain)
	return gw, nil
}

// DeleteGateway removes the gateway and all K8s resources.
func (s *GatewayService) DeleteGateway(ctx context.Context, id string) error {
	gw, err := s.gwRepo.GetGateway(ctx, id)
	if err != nil {
		return err
	}

	s.gwRepo.UpdateGatewayStatus(ctx, id, entities.GatewayStatusDeleting)

	// Delete ApisixRoute
	apisixName := "gw-" + gw.Slug
	if err := s.k8sClient.DeleteCRDWithVersion(ctx, "apisix.apache.org/v2", "ApisixRoute", s.namespace, apisixName); err != nil {
		slog.Warn("failed to delete ApisixRoute", "name", apisixName, "error", err)
	}

	// Delete IngressRoute
	ingressName := "gw-" + gw.Slug
	if err := s.k8sClient.DeleteCRD(ctx, "IngressRoute", s.namespace, ingressName); err != nil {
		slog.Warn("failed to delete IngressRoute", "name", ingressName, "error", err)
	}

	return s.gwRepo.DeleteGateway(ctx, id)
}

// CreateRoute adds a route to the gateway.
func (s *GatewayService) CreateRoute(ctx context.Context, gwID string, route *entities.GatewayRoute) (*entities.GatewayRoute, error) {
	gw, err := s.gwRepo.GetGateway(ctx, gwID)
	if err != nil {
		return nil, err
	}

	route.GatewayID = gwID

	// If route belongs to a group, inherit app from group
	if route.GroupID != "" {
		group, err := s.gwRepo.GetGroup(ctx, route.GroupID)
		if err != nil {
			return nil, fmt.Errorf("group not found: %s", route.GroupID)
		}
		if group.GatewayID != gwID {
			return nil, fmt.Errorf("group does not belong to this gateway")
		}
		// Clear route-level app fields (inherited from group)
		route.AppID = ""
		route.AppSubdomain = ""
	} else {
		// Standalone route — validate target app
		if route.AppID == "" {
			return nil, fmt.Errorf("app_id is required for standalone routes (not in a group)")
		}
		app, err := s.appRepo.GetApp(ctx, route.AppID)
		if err != nil {
			return nil, fmt.Errorf("target app not found: %s", route.AppID)
		}
		if app.UserID != gw.UserID {
			return nil, fmt.Errorf("target app does not belong to you")
		}
		route.AppSubdomain = app.Subdomain
	}

	// Validate auth pool ownership + status
	if route.AuthPoolID != "" {
		if s.authPoolRepo == nil {
			return nil, fmt.Errorf("auth pool integration not configured")
		}
		pool, err := s.authPoolRepo.GetPool(ctx, route.AuthPoolID)
		if err != nil {
			return nil, fmt.Errorf("auth pool not found: %s", route.AuthPoolID)
		}
		if pool.UserID != gw.UserID {
			return nil, fmt.Errorf("auth pool does not belong to you")
		}
		if pool.Status != entities.AuthPoolStatusActive {
			return nil, fmt.Errorf("auth pool is not active (status: %s)", pool.Status)
		}
		route.Auth = entities.GatewayRouteAuthOIDC
	}

	// Validate plugins
	if err := validatePlugins(route.Plugins); err != nil {
		return nil, err
	}

	created, err := s.gwRepo.CreateRoute(ctx, route)
	if err != nil {
		return nil, err
	}

	// Rebuild ApisixRoute
	s.rebuildApisixRoute(ctx, gw)

	return created, nil
}

// UpdateRoute updates a route and rebuilds the CRD.
func (s *GatewayService) UpdateRoute(ctx context.Context, gwID, routeID string, route *entities.GatewayRoute) (*entities.GatewayRoute, error) {
	gw, err := s.gwRepo.GetGateway(ctx, gwID)
	if err != nil {
		return nil, err
	}

	existing, err := s.gwRepo.GetRoute(ctx, routeID)
	if err != nil {
		return nil, err
	}
	if existing.GatewayID != gwID {
		return nil, fmt.Errorf("route does not belong to this gateway")
	}

	// Handle group assignment
	if route.GroupID != "" {
		group, err := s.gwRepo.GetGroup(ctx, route.GroupID)
		if err != nil {
			return nil, fmt.Errorf("group not found: %s", route.GroupID)
		}
		if group.GatewayID != gwID {
			return nil, fmt.Errorf("group does not belong to this gateway")
		}
		// Clear route-level app fields (inherited from group)
		route.AppID = ""
		route.AppSubdomain = ""
	} else {
		// Standalone route — validate app
		if route.AppID != "" && route.AppID != existing.AppID {
			app, err := s.appRepo.GetApp(ctx, route.AppID)
			if err != nil {
				return nil, fmt.Errorf("target app not found: %s", route.AppID)
			}
			if app.UserID != gw.UserID {
				return nil, fmt.Errorf("target app does not belong to you")
			}
			route.AppSubdomain = app.Subdomain
		} else if route.AppID == "" && existing.GroupID == "" {
			route.AppID = existing.AppID
			route.AppSubdomain = existing.AppSubdomain
		}
	}

	route.ID = routeID
	route.GatewayID = gwID

	// Validate auth pool ownership + status
	if route.AuthPoolID != "" {
		if s.authPoolRepo == nil {
			return nil, fmt.Errorf("auth pool integration not configured")
		}
		pool, err := s.authPoolRepo.GetPool(ctx, route.AuthPoolID)
		if err != nil {
			return nil, fmt.Errorf("auth pool not found: %s", route.AuthPoolID)
		}
		if pool.UserID != gw.UserID {
			return nil, fmt.Errorf("auth pool does not belong to you")
		}
		if pool.Status != entities.AuthPoolStatusActive {
			return nil, fmt.Errorf("auth pool is not active (status: %s)", pool.Status)
		}
		route.Auth = entities.GatewayRouteAuthOIDC
	}

	// Validate plugins
	if err := validatePlugins(route.Plugins); err != nil {
		return nil, err
	}

	updated, err := s.gwRepo.UpdateRoute(ctx, route)
	if err != nil {
		return nil, err
	}

	s.rebuildApisixRoute(ctx, gw)
	return updated, nil
}

// DeleteRoute removes a route and rebuilds the CRD.
func (s *GatewayService) DeleteRoute(ctx context.Context, gwID, routeID string) error {
	existing, err := s.gwRepo.GetRoute(ctx, routeID)
	if err != nil {
		return err
	}
	if existing.GatewayID != gwID {
		return fmt.Errorf("route does not belong to this gateway")
	}

	if err := s.gwRepo.DeleteRoute(ctx, routeID); err != nil {
		return err
	}

	gw, err := s.gwRepo.GetGateway(ctx, gwID)
	if err != nil {
		return nil // gateway was deleted
	}

	s.rebuildApisixRoute(ctx, gw)
	return nil
}

// --- Group CRUD ---

// CreateGroup adds a group to the gateway.
func (s *GatewayService) CreateGroup(ctx context.Context, gwID string, group *entities.GatewayGroup) (*entities.GatewayGroup, error) {
	gw, err := s.gwRepo.GetGateway(ctx, gwID)
	if err != nil {
		return nil, err
	}

	// Validate target app
	app, err := s.appRepo.GetApp(ctx, group.AppID)
	if err != nil {
		return nil, fmt.Errorf("target app not found: %s", group.AppID)
	}
	if app.UserID != gw.UserID {
		return nil, fmt.Errorf("target app does not belong to you")
	}

	group.GatewayID = gwID
	group.AppSubdomain = app.Subdomain

	if err := validatePlugins(group.Plugins); err != nil {
		return nil, err
	}

	created, err := s.gwRepo.CreateGroup(ctx, group)
	if err != nil {
		return nil, err
	}

	// Rebuild CRD (group plugins now apply to any routes assigned to this group)
	s.rebuildApisixRoute(ctx, gw)
	return created, nil
}

// ListGroups lists groups in a gateway.
func (s *GatewayService) ListGroups(ctx context.Context, gwID string) ([]entities.GatewayGroup, error) {
	return s.gwRepo.ListGroupsByGateway(ctx, gwID)
}

// UpdateGroup updates a group and rebuilds the CRD.
func (s *GatewayService) UpdateGroup(ctx context.Context, gwID, groupID string, group *entities.GatewayGroup) (*entities.GatewayGroup, error) {
	gw, err := s.gwRepo.GetGateway(ctx, gwID)
	if err != nil {
		return nil, err
	}

	existing, err := s.gwRepo.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if existing.GatewayID != gwID {
		return nil, fmt.Errorf("group does not belong to this gateway")
	}

	// If app changed, validate it
	if group.AppID != "" && group.AppID != existing.AppID {
		app, err := s.appRepo.GetApp(ctx, group.AppID)
		if err != nil {
			return nil, fmt.Errorf("target app not found: %s", group.AppID)
		}
		if app.UserID != gw.UserID {
			return nil, fmt.Errorf("target app does not belong to you")
		}
		group.AppSubdomain = app.Subdomain
	} else {
		group.AppID = existing.AppID
		group.AppSubdomain = existing.AppSubdomain
	}

	if group.Name == "" {
		group.Name = existing.Name
	}

	group.ID = groupID
	group.GatewayID = gwID

	if err := validatePlugins(group.Plugins); err != nil {
		return nil, err
	}

	updated, err := s.gwRepo.UpdateGroup(ctx, group)
	if err != nil {
		return nil, err
	}

	s.rebuildApisixRoute(ctx, gw)
	return updated, nil
}

// DeleteGroup removes a group (routes get group_id = NULL via ON DELETE SET NULL).
func (s *GatewayService) DeleteGroup(ctx context.Context, gwID, groupID string) error {
	existing, err := s.gwRepo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if existing.GatewayID != gwID {
		return fmt.Errorf("group does not belong to this gateway")
	}

	if err := s.gwRepo.DeleteGroup(ctx, groupID); err != nil {
		return err
	}

	gw, err := s.gwRepo.GetGateway(ctx, gwID)
	if err != nil {
		return nil // gateway was deleted
	}

	s.rebuildApisixRoute(ctx, gw)
	return nil
}

// SyncGateway forces a reconciliation of K8s CRDs from the DB state.
func (s *GatewayService) SyncGateway(ctx context.Context, id string) error {
	gw, err := s.gwRepo.GetGateway(ctx, id)
	if err != nil {
		return err
	}

	// Recreate IngressRoute
	if err := s.createIngressRoute(ctx, gw); err != nil {
		slog.Warn("gateway sync: failed to recreate IngressRoute", "slug", gw.Slug, "error", err)
	}

	s.rebuildApisixRoute(ctx, gw)
	return nil
}

// ReconcileAll syncs all active gateways (run once on startup).
func (s *GatewayService) ReconcileAll(ctx context.Context) {
	// We need to list all gateways across all users - just log for now
	// In production, add a ListAllActive method to the repo
	slog.Info("gateway reconciliation available via /sync endpoint")
}

// HandleAppDeleted stops all routes pointing to the deleted app, removes groups, and rebuilds CRDs.
func (s *GatewayService) HandleAppDeleted(ctx context.Context, appID string) {
	// Stop standalone routes pointing to the app
	gwIDs, err := s.gwRepo.StopRoutesByApp(ctx, appID)
	if err != nil {
		slog.Error("failed to stop routes for deleted app", "app_id", appID, "error", err)
	}

	// Remove groups targeting the app (routes get group_id = NULL via ON DELETE SET NULL)
	groupGwIDs, err := s.gwRepo.StopGroupsByApp(ctx, appID)
	if err != nil {
		slog.Error("failed to stop groups for deleted app", "app_id", appID, "error", err)
	}

	// Merge affected gateway IDs
	seen := make(map[string]bool)
	var allGwIDs []string
	for _, id := range gwIDs {
		if !seen[id] {
			seen[id] = true
			allGwIDs = append(allGwIDs, id)
		}
	}
	for _, id := range groupGwIDs {
		if !seen[id] {
			seen[id] = true
			allGwIDs = append(allGwIDs, id)
		}
	}

	for _, gwID := range allGwIDs {
		gw, err := s.gwRepo.GetGateway(ctx, gwID)
		if err != nil {
			continue
		}
		s.rebuildApisixRoute(ctx, gw)
	}

	if len(allGwIDs) > 0 {
		slog.Info("stopped routes/groups for deleted app", "gateway_count", len(allGwIDs), "app_id", appID)
	}
}

// --- K8s Resource Generation ---

// ensureExternalNameBridge creates the shared ExternalName service that bridges
// zenith-apps → apisix-gateway.apisix.svc.cluster.local (idempotent via PatchCRD).
func (s *GatewayService) ensureExternalNameBridge(ctx context.Context) error {
	spec, _ := json.Marshal(map[string]interface{}{
		"type":         "ExternalName",
		"externalName": "apisix-gateway.apisix.svc.cluster.local",
		"ports": []map[string]interface{}{
			{"port": 80, "targetPort": 80, "protocol": "TCP"},
		},
	})

	svc := &k8sclient.CRDObject{
		APIVersion: "v1",
		Kind:       "Service",
		Metadata: k8sclient.ObjectMeta{
			Name:      "apisix-gateway-bridge",
			Namespace: s.namespace,
			Labels: map[string]string{
				"zenith.io/managed-by": "gateway",
			},
		},
		Spec: spec,
	}

	// Use PatchCRD for idempotent create-or-update
	return s.k8sClient.PatchCRD(ctx, svc)
}

// createIngressRoute creates a Traefik IngressRoute for the gateway hostname.
func (s *GatewayService) createIngressRoute(ctx context.Context, gw *entities.Gateway) error {
	host := gw.Slug + "." + s.gwDomain
	name := "gw-" + gw.Slug

	spec, _ := json.Marshal(map[string]interface{}{
		"entryPoints": []string{"websecure"},
		"routes": []map[string]interface{}{
			{
				"match": fmt.Sprintf("Host(`%s`)", host),
				"kind":  "Rule",
				"services": []map[string]interface{}{
					{
						"name": "apisix-gateway-bridge",
						"port": 80,
					},
				},
			},
		},
		"tls": map[string]interface{}{},
	})

	crd := &k8sclient.CRDObject{
		APIVersion: "traefik.io/v1alpha1",
		Kind:       "IngressRoute",
		Metadata: k8sclient.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
			Labels: map[string]string{
				"zenith.io/gateway": gw.ID,
				"zenith.io/user":    gw.UserID,
			},
		},
		Spec: spec,
	}

	// Try create, fall back to patch for idempotency
	if err := s.k8sClient.CreateCRD(ctx, crd); err != nil {
		return s.k8sClient.PatchCRD(ctx, crd)
	}
	return nil
}

// rebuildApisixRoute rebuilds the ApisixRoute CRD from DB state.
func (s *GatewayService) rebuildApisixRoute(ctx context.Context, gw *entities.Gateway) {
	routes, err := s.gwRepo.ListActiveRoutesByGateway(ctx, gw.ID)
	if err != nil {
		slog.Error("failed to list routes for ApisixRoute rebuild", "slug", gw.Slug, "error", err)
		return
	}

	groups, err := s.gwRepo.ListGroupsByGateway(ctx, gw.ID)
	if err != nil {
		slog.Error("failed to list groups for ApisixRoute rebuild", "slug", gw.Slug, "error", err)
		groups = nil
	}

	if err := s.syncApisixRoute(ctx, gw, routes, groups); err != nil {
		slog.Error("failed to sync ApisixRoute", "slug", gw.Slug, "error", err)
	}
}

// mergePlugins merges group-level and route-level plugins. Route plugins take precedence.
func mergePlugins(groupPlugins, routePlugins []entities.GatewayRoutePlugin) []entities.GatewayRoutePlugin {
	merged := make([]entities.GatewayRoutePlugin, 0, len(groupPlugins)+len(routePlugins))
	seen := map[string]bool{}
	// Route plugins take precedence
	for _, p := range routePlugins {
		merged = append(merged, p)
		seen[p.Name] = true
	}
	// Add group plugins not overridden by route
	for _, p := range groupPlugins {
		if !seen[p.Name] {
			merged = append(merged, p)
		}
	}
	return merged
}

// syncApisixRoute creates or updates the ApisixRoute CRD for a gateway.
func (s *GatewayService) syncApisixRoute(ctx context.Context, gw *entities.Gateway, routes []entities.GatewayRoute, groups []entities.GatewayGroup) error {
	name := "gw-" + gw.Slug
	host := gw.Slug + "." + s.gwDomain

	// Build group lookup
	groupMap := make(map[string]*entities.GatewayGroup, len(groups))
	for i := range groups {
		groupMap[groups[i].ID] = &groups[i]
	}

	httpRoutes := make([]map[string]interface{}, 0, len(routes))
	for _, rt := range routes {
		routeName := "r-" + rt.ID[:8]

		// Resolve app from group if route belongs to one
		appSubdomain := rt.AppSubdomain
		var effectivePlugins []entities.GatewayRoutePlugin
		if rt.GroupID != "" {
			if grp, ok := groupMap[rt.GroupID]; ok {
				appSubdomain = grp.AppSubdomain
				effectivePlugins = mergePlugins(grp.Plugins, rt.Plugins)
			} else {
				effectivePlugins = rt.Plugins
			}
		} else {
			effectivePlugins = rt.Plugins
		}

		// Skip routes with no resolved backend
		if appSubdomain == "" {
			slog.Warn("skipping route with no app subdomain", "route_id", rt.ID, "route_name", rt.Name)
			continue
		}

		// Build plugins list
		plugins := make([]map[string]interface{}, 0, len(effectivePlugins))
		for _, p := range effectivePlugins {
			plugin := map[string]interface{}{
				"name":   p.Name,
				"enable": p.Enable,
			}
			if len(p.Config) > 0 {
				var cfg interface{}
				json.Unmarshal(p.Config, &cfg)
				plugin["config"] = cfg
			}
			plugins = append(plugins, plugin)
		}

		// Inject openid-connect plugin for auth pool-protected routes
		if rt.AuthPoolID != "" && s.authPoolRepo != nil {
			pool, _ := s.authPoolRepo.GetPool(ctx, rt.AuthPoolID)
			if pool != nil && pool.Status == entities.AuthPoolStatusActive {
				plugins = append(plugins, map[string]interface{}{
					"name":   "openid-connect",
					"enable": true,
					"config": map[string]interface{}{
						"client_id":     pool.ClientID,
						"client_secret": pool.ClientSecret,
						"discovery":     pool.IssuerURL + "/.well-known/openid-configuration",
						"bearer_only":   true,
						"realm":         pool.RealmName,
					},
				})
			}
		}

		// Add proxy-rewrite if strip_prefix is enabled
		if rt.StripPrefix {
			plugins = append(plugins, map[string]interface{}{
				"name":   "proxy-rewrite",
				"enable": true,
				"config": map[string]interface{}{
					"regex_uri": []string{fmt.Sprintf("^%s(.*)", strings.TrimSuffix(rt.Path, "/*")), "/$1"},
				},
			})
		}

		route := map[string]interface{}{
			"name": routeName,
			"match": map[string]interface{}{
				"hosts": []string{host},
				"paths": []string{rt.Path},
			},
			"backends": []map[string]interface{}{
				{
					"serviceName": appSubdomain,
					"servicePort": 80,
				},
			},
		}

		if len(rt.Methods) > 0 {
			route["methods"] = rt.Methods
		}
		if len(plugins) > 0 {
			route["plugins"] = plugins
		}

		httpRoutes = append(httpRoutes, route)
	}

	spec, _ := json.Marshal(map[string]interface{}{
		"http": httpRoutes,
	})

	crd := &k8sclient.CRDObject{
		APIVersion: "apisix.apache.org/v2",
		Kind:       "ApisixRoute",
		Metadata: k8sclient.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
			Labels: map[string]string{
				"zenith.io/gateway": gw.ID,
				"zenith.io/user":    gw.UserID,
			},
		},
		Spec: spec,
	}

	// Delete then create for clean replacement
	s.k8sClient.DeleteCRDWithVersion(ctx, "apisix.apache.org/v2", "ApisixRoute", s.namespace, name)
	return s.k8sClient.CreateCRD(ctx, crd)
}

// HandleAuthPoolDeleted rebuilds CRDs for gateways affected by an auth pool deletion.
func (s *GatewayService) HandleAuthPoolDeleted(ctx context.Context, gwIDs []string) {
	for _, gwID := range gwIDs {
		gw, err := s.gwRepo.GetGateway(ctx, gwID)
		if err != nil {
			continue
		}
		s.rebuildApisixRoute(ctx, gw)
	}
	if len(gwIDs) > 0 {
		slog.Info("rebuilt CRDs after auth pool deletion", "gateway_count", len(gwIDs))
	}
}

// validatePlugins checks that all plugins are in the allowlist.
func validatePlugins(plugins []entities.GatewayRoutePlugin) error {
	for _, p := range plugins {
		if !entities.AllowedPlugins[p.Name] {
			return fmt.Errorf("plugin '%s' is not allowed. Allowed: cors, limit-count, jwt-auth, key-auth, ip-restriction, proxy-rewrite, request-id", p.Name)
		}
	}
	return nil
}
