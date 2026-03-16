package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/google/uuid"
)

// K8sProvisioner is the interface for creating K8s resources.
// This avoids importing the k8sclient package directly.
type K8sProvisioner interface {
	ApplyUnstructured(ctx context.Context, namespace string, obj map[string]interface{}) error
	DeleteResource(ctx context.Context, namespace, apiVersion, kind, name string) error
}

// ManagedServiceService handles provisioning and lifecycle of managed services.
type ManagedServiceService struct {
	msRepo    ports.ManagedServiceRepository
	k8s       K8sProvisioner
	namespace string
}

// NewManagedServiceService creates a new ManagedServiceService.
func NewManagedServiceService(msRepo ports.ManagedServiceRepository, k8s K8sProvisioner, namespace string) *ManagedServiceService {
	return &ManagedServiceService{
		msRepo:    msRepo,
		k8s:       k8s,
		namespace: namespace,
	}
}

// ProvisionPostgreSQL creates a CNPG Cluster for a managed PostgreSQL service.
func (s *ManagedServiceService) ProvisionPostgreSQL(ctx context.Context, projectID, userID, name, version string, storageGB int) (*entities.ManagedService, error) {
	id := uuid.New().String()
	user, pass, dbName := generateCredentials(name)

	resourceName := fmt.Sprintf("ms-%s", sanitizeK8sName(name))
	host := fmt.Sprintf("%s-rw.%s.svc", resourceName, s.namespace)
	port := entities.DefaultPort(entities.ServiceTypePostgreSQL)
	connURL := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable", user, pass, host, port, dbName)

	svc := &entities.ManagedService{
		ID:              id,
		ProjectID:       projectID,
		UserID:          userID,
		ServiceType:     entities.ServiceTypePostgreSQL,
		Name:            name,
		Version:         normalizeVersion(version, "16"),
		ConnectionURL:   connURL,
		InternalHost:    host,
		Port:            port,
		Username:        user,
		Password:        pass,
		DatabaseName:    dbName,
		K8sNamespace:    s.namespace,
		K8sResourceName: resourceName,
		Status:          entities.ManagedServiceProvisioning,
		StorageGB:       storageGB,
	}

	if err := s.msRepo.CreateManagedService(ctx, svc); err != nil {
		return nil, err
	}

	// Create CNPG Cluster CRD
	if s.k8s != nil {
		cluster := buildCNPGCluster(resourceName, s.namespace, svc.Version, user, pass, dbName, storageGB)
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, cluster); err != nil {
			slog.Error("failed to create CNPG cluster", "name", resourceName, "error", err)
			s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceError, err.Error(), "", "", 0)
			return svc, nil // return svc with status=provisioning, don't fail hard
		}

		// Start background status polling
		go s.pollCNPGReady(id, resourceName)
	} else {
		// Dev mode: mark ready immediately
		s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceReady, "", connURL, host, port)
		svc.Status = entities.ManagedServiceReady
	}

	return svc, nil
}

// ProvisionRedis creates a Redis StatefulSet for a managed Redis service.
func (s *ManagedServiceService) ProvisionRedis(ctx context.Context, projectID, userID, name, version string, storageGB int) (*entities.ManagedService, error) {
	id := uuid.New().String()
	pass := randomHex(16)

	resourceName := fmt.Sprintf("ms-%s", sanitizeK8sName(name))
	host := fmt.Sprintf("%s.%s.svc", resourceName, s.namespace)
	port := entities.DefaultPort(entities.ServiceTypeRedis)
	connURL := fmt.Sprintf("redis://:%s@%s:%d/0", pass, host, port)

	svc := &entities.ManagedService{
		ID:              id,
		ProjectID:       projectID,
		UserID:          userID,
		ServiceType:     entities.ServiceTypeRedis,
		Name:            name,
		Version:         normalizeVersion(version, "7"),
		ConnectionURL:   connURL,
		InternalHost:    host,
		Port:            port,
		Password:        pass,
		K8sNamespace:    s.namespace,
		K8sResourceName: resourceName,
		Status:          entities.ManagedServiceProvisioning,
		StorageGB:       storageGB,
	}

	if err := s.msRepo.CreateManagedService(ctx, svc); err != nil {
		return nil, err
	}

	// TODO: Create Redis StatefulSet + Service + PVC + Secret via k8s
	// For now, mark ready in dev mode
	if s.k8s == nil {
		s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceReady, "", connURL, host, port)
		svc.Status = entities.ManagedServiceReady
	}

	return svc, nil
}

// DeleteManagedService removes a managed service and its K8s resources.
func (s *ManagedServiceService) DeleteManagedService(ctx context.Context, id string) error {
	ms, err := s.msRepo.GetManagedService(ctx, id)
	if err != nil {
		return err
	}

	// Update status to deleting
	s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceDeleting, "", "", "", 0)

	// Delete K8s resources
	if s.k8s != nil && ms.K8sResourceName != "" {
		switch ms.ServiceType {
		case entities.ServiceTypePostgreSQL:
			if err := s.k8s.DeleteResource(ctx, ms.K8sNamespace, "postgresql.cnpg.io/v1", "Cluster", ms.K8sResourceName); err != nil {
				slog.Warn("failed to delete CNPG cluster", "name", ms.K8sResourceName, "error", err)
			}
		case entities.ServiceTypeRedis:
			// TODO: delete StatefulSet + Service + PVC
			slog.Info("redis cleanup not yet implemented", "name", ms.K8sResourceName)
		}
	}

	return s.msRepo.DeleteManagedService(ctx, id)
}

// ListByProject returns all managed services for a project.
func (s *ManagedServiceService) ListByProject(ctx context.Context, projectID string) ([]entities.ManagedService, error) {
	return s.msRepo.ListManagedServicesByProject(ctx, projectID)
}

// pollCNPGReady polls CNPG cluster status until ready or timeout.
func (s *ManagedServiceService) pollCNPGReady(msID, resourceName string) {
	ctx := context.Background()
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			slog.Warn("CNPG cluster provision timeout", "name", resourceName)
			s.msRepo.UpdateManagedServiceStatus(ctx, msID, entities.ManagedServiceError, "provision timeout after 5m", "", "", 0)
			return
		case <-ticker.C:
			// TODO: check actual CNPG cluster status via k8s API
			// For now, mark ready after first tick (placeholder)
			ms, err := s.msRepo.GetManagedService(ctx, msID)
			if err != nil {
				return
			}
			if ms.Status == entities.ManagedServiceReady || ms.Status == entities.ManagedServiceError {
				return
			}
			// Placeholder: mark ready
			s.msRepo.UpdateManagedServiceStatus(ctx, msID, entities.ManagedServiceReady, "", ms.ConnectionURL, ms.InternalHost, ms.Port)
			return
		}
	}
}

// buildCNPGCluster returns an unstructured CNPG Cluster manifest.
func buildCNPGCluster(name, namespace, version, user, password, dbName string, storageGB int) map[string]interface{} {
	imageTag := version
	if !strings.Contains(imageTag, ".") {
		imageTag = version + ".6" // default to latest patch
	}

	return map[string]interface{}{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Cluster",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"instances":  1,
			"imageName":  fmt.Sprintf("ghcr.io/cloudnative-pg/postgresql:%s", imageTag),
			"primaryUpdateStrategy": "unsupervised",
			"bootstrap": map[string]interface{}{
				"initdb": map[string]interface{}{
					"database": dbName,
					"owner":    user,
					"secret": map[string]interface{}{
						"name": name + "-superuser",
					},
				},
			},
			"storage": map[string]interface{}{
				"size": fmt.Sprintf("%dGi", storageGB),
			},
		},
	}
}

// Helpers

func generateCredentials(serviceName string) (user, pass, dbName string) {
	clean := sanitizeK8sName(serviceName)
	user = clean + "_user"
	pass = randomHex(16)
	dbName = clean + "_db"
	return
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func sanitizeK8sName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	var result []byte
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		}
	}
	r := string(result)
	for strings.Contains(r, "--") {
		r = strings.ReplaceAll(r, "--", "-")
	}
	return strings.Trim(r, "-")
}

func normalizeVersion(v, fallback string) string {
	if v == "" || v == "latest" {
		return fallback
	}
	return v
}
