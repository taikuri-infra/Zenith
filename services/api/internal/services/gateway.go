package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// GatewayService handles gateway CRUD and K8s CRD synchronization.
type GatewayService struct {
	gwRepo    ports.GatewayRepository
	appRepo   ports.AppRepository
	planRepo  ports.UserPlanRepository
	k8sClient k8sclient.Client
	gwDomain  string // e.g. "gw.stage.freezenith.com"
	namespace string // e.g. "zenith-apps"
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
func (s *GatewayService) CreateGateway(ctx context.Context, userID, name string) (*entities.Gateway, error) {
	slug := slugify(name)

	// Check slug uniqueness (also check app subdomains to prevent DNS hijacking)
	if _, err := s.gwRepo.GetGatewayBySlug(ctx, slug); err == nil {
		return nil, fmt.Errorf("gateway slug '%s' already exists", slug)
	}
	if _, err := s.appRepo.GetAppBySubdomain(ctx, slug); err == nil {
		return nil, fmt.Errorf("slug '%s' conflicts with an existing app subdomain", slug)
	}

	// Create DB record
	gw, err := s.gwRepo.CreateGateway(ctx, userID, name, slug)
	if err != nil {
		return nil, err
	}

	gw.Endpoint = fmt.Sprintf("https://%s.%s", slug, s.gwDomain)

	// Create K8s resources
	if err := s.ensureExternalNameBridge(ctx); err != nil {
		log.Printf("[gateway] Warning: failed to ensure ExternalName bridge: %v", err)
	}

	if err := s.createIngressRoute(ctx, gw); err != nil {
		log.Printf("[gateway] Warning: failed to create IngressRoute for %s: %v", slug, err)
		s.gwRepo.UpdateGatewayStatus(ctx, gw.ID, entities.GatewayStatusError)
		gw.Status = entities.GatewayStatusError
		return gw, nil
	}

	// Create empty ApisixRoute
	if err := s.syncApisixRoute(ctx, gw, nil); err != nil {
		log.Printf("[gateway] Warning: failed to create ApisixRoute for %s: %v", slug, err)
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
		log.Printf("[gateway] Warning: failed to delete ApisixRoute %s: %v", apisixName, err)
	}

	// Delete IngressRoute
	ingressName := "gw-" + gw.Slug
	if err := s.k8sClient.DeleteCRD(ctx, "IngressRoute", s.namespace, ingressName); err != nil {
		log.Printf("[gateway] Warning: failed to delete IngressRoute %s: %v", ingressName, err)
	}

	return s.gwRepo.DeleteGateway(ctx, id)
}

// CreateRoute adds a route to the gateway.
func (s *GatewayService) CreateRoute(ctx context.Context, gwID string, route *entities.GatewayRoute) (*entities.GatewayRoute, error) {
	gw, err := s.gwRepo.GetGateway(ctx, gwID)
	if err != nil {
		return nil, err
	}

	// Validate target app exists and belongs to user
	app, err := s.appRepo.GetApp(ctx, route.AppID)
	if err != nil {
		return nil, fmt.Errorf("target app not found: %s", route.AppID)
	}
	if app.UserID != gw.UserID {
		return nil, fmt.Errorf("target app does not belong to you")
	}

	route.GatewayID = gwID
	route.AppSubdomain = app.Subdomain

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

	// If app changed, validate it
	if route.AppID != "" && route.AppID != existing.AppID {
		app, err := s.appRepo.GetApp(ctx, route.AppID)
		if err != nil {
			return nil, fmt.Errorf("target app not found: %s", route.AppID)
		}
		if app.UserID != gw.UserID {
			return nil, fmt.Errorf("target app does not belong to you")
		}
		route.AppSubdomain = app.Subdomain
	} else {
		route.AppID = existing.AppID
		route.AppSubdomain = existing.AppSubdomain
	}

	route.ID = routeID
	route.GatewayID = gwID

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

// SyncGateway forces a reconciliation of K8s CRDs from the DB state.
func (s *GatewayService) SyncGateway(ctx context.Context, id string) error {
	gw, err := s.gwRepo.GetGateway(ctx, id)
	if err != nil {
		return err
	}

	// Recreate IngressRoute
	if err := s.createIngressRoute(ctx, gw); err != nil {
		log.Printf("[gateway] Sync: failed to recreate IngressRoute for %s: %v", gw.Slug, err)
	}

	s.rebuildApisixRoute(ctx, gw)
	return nil
}

// ReconcileAll syncs all active gateways (run once on startup).
func (s *GatewayService) ReconcileAll(ctx context.Context) {
	// We need to list all gateways across all users - just log for now
	// In production, add a ListAllActive method to the repo
	log.Println("[gateway] Reconcile: gateway reconciliation available via /sync endpoint")
}

// HandleAppDeleted stops all routes pointing to the deleted app and rebuilds CRDs.
func (s *GatewayService) HandleAppDeleted(ctx context.Context, appID string) {
	gwIDs, err := s.gwRepo.StopRoutesByApp(ctx, appID)
	if err != nil {
		log.Printf("[gateway] HandleAppDeleted: failed to stop routes for app %s: %v", appID, err)
		return
	}

	for _, gwID := range gwIDs {
		gw, err := s.gwRepo.GetGateway(ctx, gwID)
		if err != nil {
			continue
		}
		s.rebuildApisixRoute(ctx, gw)
	}

	if len(gwIDs) > 0 {
		log.Printf("[gateway] HandleAppDeleted: stopped routes in %d gateways for deleted app %s", len(gwIDs), appID)
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
		log.Printf("[gateway] rebuildApisixRoute: failed to list routes for %s: %v", gw.Slug, err)
		return
	}

	if err := s.syncApisixRoute(ctx, gw, routes); err != nil {
		log.Printf("[gateway] rebuildApisixRoute: failed to sync ApisixRoute for %s: %v", gw.Slug, err)
	}
}

// syncApisixRoute creates or updates the ApisixRoute CRD for a gateway.
func (s *GatewayService) syncApisixRoute(ctx context.Context, gw *entities.Gateway, routes []entities.GatewayRoute) error {
	name := "gw-" + gw.Slug
	host := gw.Slug + "." + s.gwDomain

	httpRoutes := make([]map[string]interface{}, 0, len(routes))
	for _, rt := range routes {
		routeName := "r-" + rt.ID[:8]

		// Build plugins list
		plugins := make([]map[string]interface{}, 0, len(rt.Plugins))
		for _, p := range rt.Plugins {
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
					"serviceName": rt.AppSubdomain,
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

// validatePlugins checks that all plugins are in the allowlist.
func validatePlugins(plugins []entities.GatewayRoutePlugin) error {
	for _, p := range plugins {
		if !entities.AllowedPlugins[p.Name] {
			return fmt.Errorf("plugin '%s' is not allowed. Allowed: cors, limit-count, jwt-auth, key-auth, ip-restriction, proxy-rewrite, request-id", p.Name)
		}
	}
	return nil
}
