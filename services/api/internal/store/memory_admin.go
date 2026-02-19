package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/models"
)

// Compile-time interface check.
var _ AdminRepository = (*MemoryAdminRepository)(nil)

// MemoryAdminRepository provides an in-memory store for admin-specific data
// (settings, modules, audit log, update history).
type MemoryAdminRepository struct {
	mu       sync.RWMutex
	settings *models.PlatformSettings
	modules  []models.Module
	audit    []models.AuditEntry
	updates  []models.UpdateHistoryEntry
}

// NewMemoryAdminRepository creates a MemoryAdminRepository pre-seeded with default data.
func NewMemoryAdminRepository() *MemoryAdminRepository {
	return &MemoryAdminRepository{
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

func (s *MemoryAdminRepository) GetSettings(_ context.Context) (*models.PlatformSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copied := *s.settings
	return &copied, nil
}

func (s *MemoryAdminRepository) UpdateSettings(_ context.Context, update *models.PlatformSettings) (*models.PlatformSettings, error) {
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
	s.settings.AutoBackups = update.AutoBackups
	if update.RetentionDays > 0 {
		s.settings.RetentionDays = update.RetentionDays
	}

	copied := *s.settings
	return &copied, nil
}

func (s *MemoryAdminRepository) ListModules(_ context.Context) ([]models.Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.Module, len(s.modules))
	copy(result, s.modules)
	return result, nil
}

func (s *MemoryAdminRepository) GetModule(_ context.Context, name string) (*models.Module, error) {
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

func (s *MemoryAdminRepository) UpdateModule(_ context.Context, name string) (*models.Module, error) {
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

func (s *MemoryAdminRepository) ListAuditLog(_ context.Context, limit, offset int) ([]models.AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if offset >= len(s.audit) {
		return []models.AuditEntry{}, nil
	}

	end := offset + limit
	if end > len(s.audit) || limit <= 0 {
		end = len(s.audit)
	}

	result := make([]models.AuditEntry, end-offset)
	copy(result, s.audit[offset:end])
	return result, nil
}

func (s *MemoryAdminRepository) AddAuditEntry(_ context.Context, entry models.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audit = append([]models.AuditEntry{entry}, s.audit...)
	return nil
}

func (s *MemoryAdminRepository) GetPlatformUpdate(_ context.Context) (*models.PlatformUpdate, error) {
	return &models.PlatformUpdate{
		Version:         "v1.3.0",
		Current:         "v1.2.1",
		ReleasedAt:      "February 10, 2026",
		Features:        []string{"MongoDB support", "Cloud Connections (AWS/GCP/Azure VPN)", "GitOps mode (zen export/apply)", "Auto-generated documentation"},
		BreakingChanges: false,
	}, nil
}

func (s *MemoryAdminRepository) ListUpdateHistory(_ context.Context) ([]models.UpdateHistoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.UpdateHistoryEntry, len(s.updates))
	copy(result, s.updates)
	return result, nil
}
