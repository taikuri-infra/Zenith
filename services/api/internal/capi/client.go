package capi

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/dotechhq/zenith/services/api/internal/models"
)

const (
	// CAPINamespace is the management namespace where CAPI resources live.
	CAPINamespace = "zenith-system"

	// CRD kinds used for CAPI cluster resources.
	KindCluster = "Cluster"
)

// Client wraps the k8s.Client to provide CAPI-specific operations
// using unstructured resources (no imported CAPI Go types).
type Client struct {
	k8s k8s.Client
}

// NewClient creates a new CAPI client wrapping the given Kubernetes client.
func NewClient(k8sClient k8s.Client) *Client {
	return &Client{k8s: k8sClient}
}

// CreateCluster creates a CAPI Cluster resource in the management namespace.
func (c *Client) CreateCluster(ctx context.Context, input models.CreateClusterInput) (*models.Cluster, error) {
	spec, _ := json.Marshal(map[string]interface{}{
		"k8sVersion": input.K8sVersion,
		"nodes":      input.Nodes,
		"region":     input.Region,
		"type":       input.Type,
		"tenant":     input.Tenant,
	})

	crd := &k8s.CRDObject{
		APIVersion: "cluster.x-k8s.io/v1beta1",
		Kind:       KindCluster,
		Metadata: k8s.ObjectMeta{
			Name:      input.Name,
			Namespace: CAPINamespace,
			Labels: map[string]string{
				"zenith.dev/cluster-type": input.Type,
				"zenith.dev/region":       input.Region,
			},
		},
		Spec: spec,
	}

	if input.Tenant != "" {
		crd.Metadata.Labels["zenith.dev/tenant"] = input.Tenant
	}

	if err := c.k8s.CreateCRD(ctx, crd); err != nil {
		return nil, fmt.Errorf("create cluster: %w", err)
	}

	return crdToCluster(crd), nil
}

// GetCluster retrieves a single CAPI Cluster resource by name.
func (c *Client) GetCluster(ctx context.Context, name string) (*models.Cluster, error) {
	crd, err := c.k8s.GetCRD(ctx, KindCluster, CAPINamespace, name)
	if err != nil {
		return nil, fmt.Errorf("get cluster %s: %w", name, err)
	}
	return crdToCluster(crd), nil
}

// ListClusters returns all CAPI Cluster resources in the management namespace.
func (c *Client) ListClusters(ctx context.Context) ([]models.Cluster, error) {
	crds, err := c.k8s.ListCRDs(ctx, KindCluster, CAPINamespace)
	if err != nil {
		return nil, fmt.Errorf("list clusters: %w", err)
	}

	clusters := make([]models.Cluster, 0, len(crds))
	for _, crd := range crds {
		clusters = append(clusters, *crdToCluster(crd))
	}
	return clusters, nil
}

// DeleteCluster removes a CAPI Cluster resource by name.
func (c *Client) DeleteCluster(ctx context.Context, name string) error {
	if err := c.k8s.DeleteCRD(ctx, KindCluster, CAPINamespace, name); err != nil {
		return fmt.Errorf("delete cluster %s: %w", name, err)
	}
	return nil
}

// UpgradeCluster updates the Kubernetes version of a cluster.
func (c *Client) UpgradeCluster(ctx context.Context, name, version string) error {
	crd, err := c.k8s.GetCRD(ctx, KindCluster, CAPINamespace, name)
	if err != nil {
		return fmt.Errorf("get cluster %s for upgrade: %w", name, err)
	}

	var spec map[string]interface{}
	_ = json.Unmarshal(crd.Spec, &spec)
	spec["k8sVersion"] = version
	crd.Spec, _ = json.Marshal(spec)

	// Add annotation to track upgrade
	if crd.Metadata.Annotations == nil {
		crd.Metadata.Annotations = make(map[string]string)
	}
	crd.Metadata.Annotations["zenith.dev/upgrade-target"] = version

	if err := c.k8s.UpdateCRD(ctx, crd); err != nil {
		return fmt.Errorf("upgrade cluster %s: %w", name, err)
	}
	return nil
}

// crdToCluster converts a CRDObject into a models.Cluster.
func crdToCluster(crd *k8s.CRDObject) *models.Cluster {
	var spec map[string]interface{}
	_ = json.Unmarshal(crd.Spec, &spec)

	k8sVersion, _ := spec["k8sVersion"].(string)
	region, _ := spec["region"].(string)
	clusterType, _ := spec["type"].(string)
	tenant, _ := spec["tenant"].(string)

	nodes := 1
	if n, ok := spec["nodes"].(float64); ok {
		nodes = int(n)
	}

	// Status fields default to placeholder values.
	// In production these would be read from the status subresource.
	cpuPercent := 0
	ramPercent := 0
	status := "healthy"

	var statusMap map[string]interface{}
	if crd.Status != nil {
		_ = json.Unmarshal(crd.Status, &statusMap)
	}
	if statusMap != nil {
		if v, ok := statusMap["cpuPercent"].(float64); ok {
			cpuPercent = int(v)
		}
		if v, ok := statusMap["ramPercent"].(float64); ok {
			ramPercent = int(v)
		}
		if v, ok := statusMap["status"].(string); ok {
			status = v
		}
	}

	upgradeAvailable, _ := crd.Metadata.Annotations["zenith.dev/upgrade-available"]

	return &models.Cluster{
		Name:             crd.Metadata.Name,
		K8sVersion:       k8sVersion,
		Nodes:            nodes,
		Region:           region,
		Type:             clusterType,
		Tenant:           tenant,
		CPUPercent:       cpuPercent,
		RAMPercent:       ramPercent,
		Pods:             models.ResourcePair{Used: 0, Total: 0},
		PVCs:             models.ResourcePair{Used: 0, Total: 0},
		Status:           status,
		UpgradeAvailable: upgradeAvailable,
	}
}

// MemoryStore provides an in-memory store for admin-specific data
// that doesn't map directly to CAPI CRDs (settings, modules, etc.).
type MemoryStore struct {
	mu       sync.RWMutex
	settings *models.PlatformSettings
	modules  []models.Module
	audit    []models.AuditEntry
	updates  []models.UpdateHistoryEntry
}

// NewMemoryStore creates a MemoryStore pre-seeded with default data.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		settings: &models.PlatformSettings{
			PlatformName:  "Zenith",
			BaseDomain:    "freezenith.com",
			Provider:      "Hetzner Cloud",
			DefaultRegion: "fsn1",
			RegionLabel:   "Falkenstein",
			AutoBackups:   true,
			RetentionDays: 30,
		},
		modules: []models.Module{
			{Name: "Zenith Operator", Installed: "v1.2.1", Latest: "v1.3.0", Status: "update_available", Description: "Core platform operator"},
			{Name: "CloudNativePG", Installed: "v1.22.1", Latest: "v1.23.0", Status: "update_available", Description: "PostgreSQL operator"},
			{Name: "Redis Operator", Installed: "v7.2.0", Latest: "v7.2.0", Status: "up_to_date", Description: "Redis operator"},
			{Name: "cert-manager", Installed: "v1.14.2", Latest: "v1.14.2", Status: "up_to_date", Description: "SSL certificate management"},
			{Name: "Traefik", Installed: "v2.11.0", Latest: "v2.11.0", Status: "up_to_date", Description: "Ingress controller"},
			{Name: "Harbor", Installed: "v2.10.0", Latest: "v2.10.1", Status: "update_available", Description: "Container registry"},
			{Name: "Keycloak Operator", Installed: "v24.0.0", Latest: "v24.0.0", Status: "up_to_date", Description: "Identity & access management"},
			{Name: "Prometheus Stack", Installed: "v56.2.0", Latest: "v56.2.0", Status: "up_to_date", Description: "Monitoring & alerting"},
			{Name: "Loki", Installed: "v3.0.1", Latest: "v3.0.1", Status: "up_to_date", Description: "Log aggregation"},
			{Name: "NATS", Installed: "v2.10.0", Latest: "v2.10.0", Status: "up_to_date", Description: "Message queue & KV store"},
			{Name: "Linkerd", Installed: "v2.14.0", Latest: "v2.14.1", Status: "update_available", Description: "Service mesh"},
		},
		audit: []models.AuditEntry{
			{Time: "14:23", Actor: "admin", Action: "Upgraded CloudNativePG v1.21 -> v1.22", Cluster: "zenith-shared"},
			{Time: "12:01", Actor: "CAPI", Action: "Scaled nodes 7 -> 8", Cluster: "zenith-shared"},
			{Time: "09:45", Actor: "system", Action: "Tenant created: startup-x", Cluster: "zenith-shared"},
			{Time: "08:12", Actor: "system", Action: "Backup completed: all databases (47 tenants)"},
		},
		updates: []models.UpdateHistoryEntry{
			{Version: "v1.2.1", Date: "2026-01-15", Status: "installed"},
			{Version: "v1.2.0", Date: "2025-12-20", Status: "superseded"},
			{Version: "v1.1.0", Date: "2025-11-01", Status: "superseded"},
		},
	}
}

// GetSettings returns the current platform settings.
func (s *MemoryStore) GetSettings() *models.PlatformSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copied := *s.settings
	return &copied
}

// UpdateSettings merges the provided fields into the current settings.
func (s *MemoryStore) UpdateSettings(update *models.PlatformSettings) *models.PlatformSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if update.PlatformName != "" {
		s.settings.PlatformName = update.PlatformName
	}
	if update.BaseDomain != "" {
		s.settings.BaseDomain = update.BaseDomain
	}
	if update.Provider != "" {
		s.settings.Provider = update.Provider
	}
	if update.DefaultRegion != "" {
		s.settings.DefaultRegion = update.DefaultRegion
	}
	if update.RegionLabel != "" {
		s.settings.RegionLabel = update.RegionLabel
	}
	// Booleans and ints are always applied
	s.settings.AutoBackups = update.AutoBackups
	if update.RetentionDays > 0 {
		s.settings.RetentionDays = update.RetentionDays
	}

	copied := *s.settings
	return &copied
}

// ListModules returns all known modules.
func (s *MemoryStore) ListModules() []models.Module {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.Module, len(s.modules))
	copy(result, s.modules)
	return result
}

// GetModule returns a single module by name.
func (s *MemoryStore) GetModule(name string) (*models.Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.modules {
		if m.Name == name {
			copied := m
			return &copied, nil
		}
	}
	return nil, fmt.Errorf("module %s not found", name)
}

// UpdateModule marks a module as updated to its latest version.
func (s *MemoryStore) UpdateModule(name string) (*models.Module, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, m := range s.modules {
		if m.Name == name {
			s.modules[i].Installed = m.Latest
			s.modules[i].Status = "up_to_date"
			copied := s.modules[i]
			return &copied, nil
		}
	}
	return nil, fmt.Errorf("module %s not found", name)
}

// ListAuditLog returns audit entries, with optional limit and offset.
func (s *MemoryStore) ListAuditLog(limit, offset int) []models.AuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if offset >= len(s.audit) {
		return []models.AuditEntry{}
	}

	end := offset + limit
	if end > len(s.audit) || limit <= 0 {
		end = len(s.audit)
	}

	result := make([]models.AuditEntry, end-offset)
	copy(result, s.audit[offset:end])
	return result
}

// AddAuditEntry adds a new entry to the audit log (prepends).
func (s *MemoryStore) AddAuditEntry(entry models.AuditEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audit = append([]models.AuditEntry{entry}, s.audit...)
}

// GetPlatformUpdate returns the currently available platform update info.
func (s *MemoryStore) GetPlatformUpdate() *models.PlatformUpdate {
	return &models.PlatformUpdate{
		Version:         "v1.3.0",
		Current:         "v1.2.1",
		ReleasedAt:      "February 10, 2026",
		Features:        []string{"MongoDB support", "Cloud Connections (AWS/GCP/Azure VPN)", "GitOps mode (zen export/apply)", "Auto-generated documentation"},
		BreakingChanges: false,
	}
}

// ListUpdateHistory returns past update entries.
func (s *MemoryStore) ListUpdateHistory() []models.UpdateHistoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.UpdateHistoryEntry, len(s.updates))
	copy(result, s.updates)
	return result
}
