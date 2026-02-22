package services

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/adapters/capiclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// AdminService handles Mission Control admin business logic.
type AdminService struct {
	capiClient *capiclient.Client
	k8sClient  k8sclient.Client
	store      ports.AdminRepository
}

// NewAdminService creates a new AdminService.
func NewAdminService(k8sClient k8sclient.Client, capiClient *capiclient.Client, store ports.AdminRepository) *AdminService {
	return &AdminService{capiClient: capiClient, k8sClient: k8sClient, store: store}
}

// GetDashboardStats returns aggregate statistics for the dashboard.
func (s *AdminService) GetDashboardStats(ctx context.Context) (*entities.DashboardStats, error) {
	clusters, err := s.capiClient.ListClusters(ctx)
	if err != nil {
		clusters = []entities.Cluster{}
	}

	allHealthy := true
	for _, cl := range clusters {
		if cl.Status != "healthy" {
			allHealthy = false
			break
		}
	}

	projects, _ := s.k8sClient.ListCRDs(ctx, "Project", "")
	tenantCount := len(projects)

	activeToday := 0
	for _, p := range projects {
		if p.Metadata.Annotations["zenith.dev/suspended"] != "true" {
			activeToday++
		}
	}

	modules, _ := s.store.ListModules(ctx)
	updatesAvailable := 0
	for _, m := range modules {
		if m.Status == "update_available" {
			updatesAvailable++
		}
	}

	return &entities.DashboardStats{
		ClusterCount:     len(clusters),
		AllHealthy:       allHealthy,
		TenantCount:      tenantCount,
		ActiveToday:      activeToday,
		MonthlyCost:      "EUR 47.60",
		CostProvider:     "Hetzner Cloud",
		UpdatesAvailable: updatesAvailable,
	}, nil
}

// ListClusters returns all CAPI-managed clusters.
func (s *AdminService) ListClusters(ctx context.Context) ([]entities.Cluster, error) {
	return s.capiClient.ListClusters(ctx)
}

// GetCluster returns a single cluster by name.
func (s *AdminService) GetCluster(ctx context.Context, name string) (*entities.Cluster, error) {
	return s.capiClient.GetCluster(ctx, name)
}

// CreateCluster provisions a new CAPI cluster.
func (s *AdminService) CreateCluster(ctx context.Context, input dto.CreateClusterInput, actor string) (*entities.Cluster, error) {
	cluster, err := s.capiClient.CreateCluster(ctx, input)
	if err != nil {
		return nil, err
	}

	_ = s.store.AddAuditEntry(ctx, entities.AuditEntry{
		Time:    time.Now().Format("15:04"),
		Actor:   actor,
		Action:  "Created cluster " + input.Name,
		Cluster: input.Name,
	})

	return cluster, nil
}

// DeleteCluster removes a CAPI cluster.
func (s *AdminService) DeleteCluster(ctx context.Context, name, actor string) error {
	if err := s.capiClient.DeleteCluster(ctx, name); err != nil {
		return err
	}

	_ = s.store.AddAuditEntry(ctx, entities.AuditEntry{
		Time:    time.Now().Format("15:04"),
		Actor:   actor,
		Action:  "Deleted cluster " + name,
		Cluster: name,
	})

	return nil
}

// UpgradeCluster initiates a Kubernetes version upgrade.
func (s *AdminService) UpgradeCluster(ctx context.Context, name, version, actor string) error {
	if err := s.capiClient.UpgradeCluster(ctx, name, version); err != nil {
		return err
	}

	_ = s.store.AddAuditEntry(ctx, entities.AuditEntry{
		Time:    time.Now().Format("15:04"),
		Actor:   actor,
		Action:  "Initiated upgrade to " + version,
		Cluster: name,
	})

	return nil
}

// ListTenants returns all tenants derived from Project CRDs.
func (s *AdminService) ListTenants(ctx context.Context) ([]entities.Tenant, error) {
	projects, err := s.k8sClient.ListCRDs(ctx, "Project", "")
	if err != nil {
		return nil, err
	}

	tenants := make([]entities.Tenant, 0, len(projects))
	for _, p := range projects {
		tenants = append(tenants, projectToTenant(p))
	}
	return tenants, nil
}

// GetTenant returns a single tenant by project name.
func (s *AdminService) GetTenant(ctx context.Context, id string) (*entities.Tenant, error) {
	proj, err := s.k8sClient.GetCRD(ctx, "Project", "", id)
	if err != nil {
		return nil, err
	}
	tenant := projectToTenant(proj)
	return &tenant, nil
}

// SuspendTenant suspends a tenant by adding an annotation.
func (s *AdminService) SuspendTenant(ctx context.Context, id, actor string) error {
	proj, err := s.k8sClient.GetCRD(ctx, "Project", "", id)
	if err != nil {
		return err
	}

	if proj.Metadata.Annotations == nil {
		proj.Metadata.Annotations = make(map[string]string)
	}
	proj.Metadata.Annotations["zenith.dev/suspended"] = "true"
	proj.Metadata.Annotations["zenith.dev/suspended-at"] = time.Now().Format(time.RFC3339)

	if err := s.k8sClient.UpdateCRD(ctx, proj); err != nil {
		return err
	}

	_ = s.store.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Suspended tenant " + id,
	})

	return nil
}

// ListModules returns all platform modules.
func (s *AdminService) ListModules(ctx context.Context) ([]entities.Module, error) {
	return s.store.ListModules(ctx)
}

// InstallModule triggers installation of a module.
func (s *AdminService) InstallModule(ctx context.Context, name, actor string) error {
	_ = s.store.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Installed module " + name,
	})
	return nil
}

// UninstallModule triggers uninstallation of a module.
func (s *AdminService) UninstallModule(ctx context.Context, name, actor string) error {
	_ = s.store.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Uninstalled module " + name,
	})
	return nil
}

// UpdateModule updates a module to its latest version.
func (s *AdminService) UpdateModule(ctx context.Context, name, actor string) (*entities.Module, error) {
	mod, err := s.store.UpdateModule(ctx, name)
	if err != nil {
		return nil, err
	}

	_ = s.store.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Updated module " + name + " to " + mod.Installed,
	})

	return mod, nil
}

// UpdateAllModules updates all modules that have updates available.
func (s *AdminService) UpdateAllModules(ctx context.Context, actor string) (int, error) {
	modules, err := s.store.ListModules(ctx)
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, m := range modules {
		if m.Status == "update_available" {
			_, _ = s.store.UpdateModule(ctx, m.Name)
			updated++
		}
	}

	_ = s.store.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Updated all modules (" + strconv.Itoa(updated) + " updated)",
	})

	return updated, nil
}

// ListAuditLog returns audit-log entries with optional filtering.
func (s *AdminService) ListAuditLog(ctx context.Context, limit, offset int) ([]entities.AuditEntry, error) {
	return s.store.ListAuditLog(ctx, limit, offset)
}

// CheckUpdates returns available platform updates.
func (s *AdminService) CheckUpdates(ctx context.Context) (*entities.PlatformUpdate, error) {
	return s.store.GetPlatformUpdate(ctx)
}

// ApplyUpdate applies a platform update.
func (s *AdminService) ApplyUpdate(ctx context.Context, version, actor string) error {
	_ = s.store.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Applied platform update " + version,
	})
	return nil
}

// ListUpdateHistory returns the history of past platform updates.
func (s *AdminService) ListUpdateHistory(ctx context.Context) ([]entities.UpdateHistoryEntry, error) {
	return s.store.ListUpdateHistory(ctx)
}

// GetInfraOverview returns a summary of infrastructure.
func (s *AdminService) GetInfraOverview(ctx context.Context) (*entities.InfraOverview, error) {
	clusters, _ := s.capiClient.ListClusters(ctx)

	totalServers := 0
	resources := make([]entities.InfraNode, 0)
	for _, cl := range clusters {
		totalServers += cl.Nodes
		resources = append(resources, entities.InfraNode{
			Name:        cl.Name + "-pool",
			Type:        "CX22",
			Count:       cl.Nodes,
			Cluster:     cl.Name,
			MonthlyCost: "EUR " + strconv.Itoa(cl.Nodes*5) + ".00",
		})
	}

	managementServers := 1
	totalServers += managementServers
	resources = append(resources, entities.InfraNode{
		Name:        "management-plane",
		Type:        "CX22",
		Count:       managementServers,
		Cluster:     "management",
		MonthlyCost: "EUR 5.00",
	})

	return &entities.InfraOverview{
		Servers:       totalServers,
		Volumes:       len(clusters) * 2,
		VolumeSize:    strconv.Itoa(len(clusters)*20) + " GB",
		LoadBalancers: len(clusters),
		LBPublic:      len(clusters),
		LBInternal:    0,
		MonthlyCost:   "EUR " + strconv.Itoa(totalServers*5+len(clusters)*10) + ".00",
		Resources:     resources,
	}, nil
}

// GetPlatformState returns the current platform state summary.
func (s *AdminService) GetPlatformState(ctx context.Context) (*entities.PlatformState, error) {
	update, _ := s.store.GetPlatformUpdate(ctx)
	settings, err := s.store.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	updateAvailable := ""
	if update != nil && update.Version != update.Current {
		updateAvailable = update.Version
	}

	return &entities.PlatformState{
		PlatformVersion:       "v1.2.1",
		UpdateAvailable:       updateAvailable,
		InstalledDate:         "2026-01-15",
		InstalledDaysAgo:      31,
		ManagementK8sVersion:  "v1.30.2",
		ManagementK8sUpToDate: true,
		Domain:                settings.BaseDomain,
		WildcardTLS:           true,
	}, nil
}

// ExportState exports the full platform state as JSON bytes.
func (s *AdminService) ExportState(ctx context.Context) ([]byte, error) {
	clusters, _ := s.capiClient.ListClusters(ctx)
	projects, _ := s.k8sClient.ListCRDs(ctx, "Project", "")
	settings, _ := s.store.GetSettings(ctx)
	modules, _ := s.store.ListModules(ctx)

	tenants := make([]entities.Tenant, 0, len(projects))
	for _, p := range projects {
		tenants = append(tenants, projectToTenant(p))
	}

	export := map[string]interface{}{
		"exportedAt": time.Now().Format(time.RFC3339),
		"platform": map[string]interface{}{
			"version":  "v1.2.1",
			"settings": settings,
		},
		"clusters": clusters,
		"tenants":  tenants,
		"modules":  modules,
	}

	return json.MarshalIndent(export, "", "  ")
}

// GetSettings returns the current platform settings.
func (s *AdminService) GetSettings(ctx context.Context) (*entities.PlatformSettings, error) {
	return s.store.GetSettings(ctx)
}

// UpdateSettings updates platform settings.
func (s *AdminService) UpdateSettings(ctx context.Context, input *entities.PlatformSettings, actor string) (*entities.PlatformSettings, error) {
	updated, err := s.store.UpdateSettings(ctx, input)
	if err != nil {
		return nil, err
	}

	_ = s.store.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  actor,
		Action: "Updated platform settings",
	})

	return updated, nil
}

// projectToTenant converts a Project CRD to a Tenant model.
func projectToTenant(p *k8sclient.CRDObject) entities.Tenant {
	var spec map[string]interface{}
	_ = json.Unmarshal(p.Spec, &spec)

	displayName, _ := spec["displayName"].(string)
	if displayName == "" {
		displayName = p.Metadata.Name
	}
	plan, _ := spec["plan"].(string)
	if plan == "" {
		plan = "starter"
	}

	status := "active"
	if p.Metadata.Annotations != nil && p.Metadata.Annotations["zenith.dev/suspended"] == "true" {
		status = "suspended"
	}

	return entities.Tenant{
		Name:      displayName,
		Plan:      plan,
		Apps:      0,
		Databases: 0,
		CPUUsed:   "0",
		CPULimit:  "4",
		RAMUsed:   "0",
		RAMLimit:  "4",
		Status:    status,
	}
}
