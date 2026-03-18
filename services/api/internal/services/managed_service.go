package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	GetCRDStatus(ctx context.Context, apiVersion, kind, namespace, name string) (json.RawMessage, error)
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

	if s.k8s != nil {
		// Create Redis password Secret
		secretName := resourceName + "-auth"
		secretObj := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      secretName,
				"namespace": s.namespace,
				"labels": map[string]interface{}{
					"app.zenith.dev/managed-service": resourceName,
				},
			},
			"type": "Opaque",
			"stringData": map[string]interface{}{
				"redis-password": pass,
			},
		}
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, secretObj); err != nil {
			slog.Error("failed to create Redis secret", "name", secretName, "error", err)
		}

		// Create Redis StatefulSet
		sts := buildRedisStatefulSet(resourceName, s.namespace, svc.Version, secretName, storageGB)
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, sts); err != nil {
			slog.Error("failed to create Redis StatefulSet", "name", resourceName, "error", err)
			s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceError, err.Error(), "", "", 0)
			return svc, nil
		}

		// Create headless Service for StatefulSet DNS
		redisSvc := buildRedisService(resourceName, s.namespace, port)
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, redisSvc); err != nil {
			slog.Error("failed to create Redis service", "name", resourceName, "error", err)
		}

		// Mark ready (Redis StatefulSet is simple enough to be ready quickly)
		s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceReady, "", connURL, host, port)
		svc.Status = entities.ManagedServiceReady
	} else {
		// Dev mode: mark ready immediately
		s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceReady, "", connURL, host, port)
		svc.Status = entities.ManagedServiceReady
	}

	return svc, nil
}

// ProvisionMySQL creates a MySQL StatefulSet for a managed MySQL service.
func (s *ManagedServiceService) ProvisionMySQL(ctx context.Context, projectID, userID, name, version string, storageGB int) (*entities.ManagedService, error) {
	id := uuid.New().String()
	user, pass, dbName := generateCredentials(name)

	resourceName := fmt.Sprintf("ms-%s", sanitizeK8sName(name))
	host := fmt.Sprintf("%s.%s.svc", resourceName, s.namespace)
	port := entities.DefaultPort(entities.ServiceTypeMySQL)
	connURL := fmt.Sprintf("mysql://%s:%s@%s:%d/%s", user, pass, host, port, dbName)

	svc := &entities.ManagedService{
		ID:              id,
		ProjectID:       projectID,
		UserID:          userID,
		ServiceType:     entities.ServiceTypeMySQL,
		Name:            name,
		Version:         normalizeVersion(version, "8"),
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

	if s.k8s != nil {
		secretName := resourceName + "-auth"
		secretObj := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      secretName,
				"namespace": s.namespace,
				"labels":    map[string]interface{}{"app.zenith.dev/managed-service": resourceName},
			},
			"type": "Opaque",
			"stringData": map[string]interface{}{
				"mysql-root-password": pass,
				"mysql-user":          user,
				"mysql-password":      pass,
				"mysql-database":      dbName,
			},
		}
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, secretObj); err != nil {
			slog.Error("failed to create MySQL secret", "name", secretName, "error", err)
		}

		sts := buildMySQLStatefulSet(resourceName, s.namespace, svc.Version, secretName, storageGB)
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, sts); err != nil {
			slog.Error("failed to create MySQL StatefulSet", "name", resourceName, "error", err)
			s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceError, err.Error(), "", "", 0)
			return svc, nil
		}

		svcObj := buildGenericService(resourceName, s.namespace, port, 3306, "mysql")
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, svcObj); err != nil {
			slog.Error("failed to create MySQL service", "name", resourceName, "error", err)
		}

		s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceReady, "", connURL, host, port)
		svc.Status = entities.ManagedServiceReady
	} else {
		s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceReady, "", connURL, host, port)
		svc.Status = entities.ManagedServiceReady
	}

	return svc, nil
}

// ProvisionMongoDB creates a MongoDB StatefulSet for a managed MongoDB service.
func (s *ManagedServiceService) ProvisionMongoDB(ctx context.Context, projectID, userID, name, version string, storageGB int) (*entities.ManagedService, error) {
	id := uuid.New().String()
	user, pass, dbName := generateCredentials(name)

	resourceName := fmt.Sprintf("ms-%s", sanitizeK8sName(name))
	host := fmt.Sprintf("%s.%s.svc", resourceName, s.namespace)
	port := entities.DefaultPort(entities.ServiceTypeMongoDB)
	connURL := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s", user, pass, host, port, dbName)

	svc := &entities.ManagedService{
		ID:              id,
		ProjectID:       projectID,
		UserID:          userID,
		ServiceType:     entities.ServiceTypeMongoDB,
		Name:            name,
		Version:         normalizeVersion(version, "7"),
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

	if s.k8s != nil {
		secretName := resourceName + "-auth"
		secretObj := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      secretName,
				"namespace": s.namespace,
				"labels":    map[string]interface{}{"app.zenith.dev/managed-service": resourceName},
			},
			"type": "Opaque",
			"stringData": map[string]interface{}{
				"mongo-root-username": user,
				"mongo-root-password": pass,
			},
		}
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, secretObj); err != nil {
			slog.Error("failed to create MongoDB secret", "name", secretName, "error", err)
		}

		sts := buildMongoDBStatefulSet(resourceName, s.namespace, svc.Version, secretName, dbName, storageGB)
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, sts); err != nil {
			slog.Error("failed to create MongoDB StatefulSet", "name", resourceName, "error", err)
			s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceError, err.Error(), "", "", 0)
			return svc, nil
		}

		svcObj := buildGenericService(resourceName, s.namespace, port, 27017, "mongo")
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, svcObj); err != nil {
			slog.Error("failed to create MongoDB service", "name", resourceName, "error", err)
		}

		s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceReady, "", connURL, host, port)
		svc.Status = entities.ManagedServiceReady
	} else {
		s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceReady, "", connURL, host, port)
		svc.Status = entities.ManagedServiceReady
	}

	return svc, nil
}

// ProvisionRabbitMQ creates a RabbitMQ StatefulSet for a managed RabbitMQ service.
func (s *ManagedServiceService) ProvisionRabbitMQ(ctx context.Context, projectID, userID, name, version string, storageGB int) (*entities.ManagedService, error) {
	id := uuid.New().String()
	user := "zenith"
	pass := randomHex(16)

	resourceName := fmt.Sprintf("ms-%s", sanitizeK8sName(name))
	host := fmt.Sprintf("%s.%s.svc", resourceName, s.namespace)
	port := entities.DefaultPort(entities.ServiceTypeRabbitMQ)
	connURL := fmt.Sprintf("amqp://%s:%s@%s:%d/", user, pass, host, port)

	svc := &entities.ManagedService{
		ID:              id,
		ProjectID:       projectID,
		UserID:          userID,
		ServiceType:     entities.ServiceTypeRabbitMQ,
		Name:            name,
		Version:         normalizeVersion(version, "3"),
		ConnectionURL:   connURL,
		InternalHost:    host,
		Port:            port,
		Username:        user,
		Password:        pass,
		K8sNamespace:    s.namespace,
		K8sResourceName: resourceName,
		Status:          entities.ManagedServiceProvisioning,
		StorageGB:       storageGB,
	}

	if err := s.msRepo.CreateManagedService(ctx, svc); err != nil {
		return nil, err
	}

	if s.k8s != nil {
		secretName := resourceName + "-auth"
		secretObj := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      secretName,
				"namespace": s.namespace,
				"labels":    map[string]interface{}{"app.zenith.dev/managed-service": resourceName},
			},
			"type": "Opaque",
			"stringData": map[string]interface{}{
				"rabbitmq-user":     user,
				"rabbitmq-password": pass,
			},
		}
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, secretObj); err != nil {
			slog.Error("failed to create RabbitMQ secret", "name", secretName, "error", err)
		}

		sts := buildRabbitMQStatefulSet(resourceName, s.namespace, svc.Version, secretName, storageGB)
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, sts); err != nil {
			slog.Error("failed to create RabbitMQ StatefulSet", "name", resourceName, "error", err)
			s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceError, err.Error(), "", "", 0)
			return svc, nil
		}

		svcObj := buildGenericService(resourceName, s.namespace, port, 5672, "amqp")
		if err := s.k8s.ApplyUnstructured(ctx, s.namespace, svcObj); err != nil {
			slog.Error("failed to create RabbitMQ service", "name", resourceName, "error", err)
		}

		s.msRepo.UpdateManagedServiceStatus(ctx, id, entities.ManagedServiceReady, "", connURL, host, port)
		svc.Status = entities.ManagedServiceReady
	} else {
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
		case entities.ServiceTypeRedis, entities.ServiceTypeMySQL, entities.ServiceTypeMongoDB, entities.ServiceTypeRabbitMQ:
			// All non-CNPG managed services use StatefulSet + Service + Secret
			if err := s.k8s.DeleteResource(ctx, ms.K8sNamespace, "apps/v1", "StatefulSet", ms.K8sResourceName); err != nil {
				slog.Warn("failed to delete StatefulSet", "type", ms.ServiceType, "name", ms.K8sResourceName, "error", err)
			}
			if err := s.k8s.DeleteResource(ctx, ms.K8sNamespace, "v1", "Service", ms.K8sResourceName); err != nil {
				slog.Warn("failed to delete Service", "type", ms.ServiceType, "name", ms.K8sResourceName, "error", err)
			}
			if err := s.k8s.DeleteResource(ctx, ms.K8sNamespace, "v1", "Secret", ms.K8sResourceName+"-auth"); err != nil {
				slog.Warn("failed to delete Secret", "type", ms.ServiceType, "name", ms.K8sResourceName, "error", err)
			}
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
			ms, err := s.msRepo.GetManagedService(ctx, msID)
			if err != nil {
				return
			}
			if ms.Status == entities.ManagedServiceReady || ms.Status == entities.ManagedServiceError {
				return
			}

			// Check actual CNPG cluster status via K8s API
			if s.k8s != nil {
				statusRaw, crdErr := s.k8s.GetCRDStatus(ctx, "postgresql.cnpg.io/v1", "Cluster", s.namespace, resourceName)
				if crdErr != nil {
					slog.Debug("waiting for CNPG cluster", "name", resourceName, "error", crdErr)
					continue
				}
				if len(statusRaw) > 0 {
					var statusMap map[string]interface{}
					if jsonErr := json.Unmarshal(statusRaw, &statusMap); jsonErr == nil {
						if phase, ok := statusMap["phase"].(string); ok && phase == "Cluster in healthy state" {
							s.msRepo.UpdateManagedServiceStatus(ctx, msID, entities.ManagedServiceReady, "", ms.ConnectionURL, ms.InternalHost, ms.Port)
							slog.Info("CNPG cluster ready", "name", resourceName)
							return
						}
					}
				}
			} else {
				// Dev mode fallback: mark ready
				s.msRepo.UpdateManagedServiceStatus(ctx, msID, entities.ManagedServiceReady, "", ms.ConnectionURL, ms.InternalHost, ms.Port)
				return
			}
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

// buildRedisStatefulSet returns an unstructured Redis StatefulSet manifest.
func buildRedisStatefulSet(name, namespace, version, secretName string, storageGB int) map[string]interface{} {
	imageTag := version
	if !strings.Contains(imageTag, ".") {
		imageTag = version + "-alpine"
	}

	return map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.zenith.dev/managed-service": name,
			},
		},
		"spec": map[string]interface{}{
			"replicas":    1,
			"serviceName": name,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app.zenith.dev/managed-service": name,
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app.zenith.dev/managed-service": name,
					},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "redis",
							"image": fmt.Sprintf("redis:%s", imageTag),
							"command": []string{
								"redis-server",
								"--requirepass",
								"$(REDIS_PASSWORD)",
								"--appendonly",
								"yes",
							},
							"env": []map[string]interface{}{
								{
									"name": "REDIS_PASSWORD",
									"valueFrom": map[string]interface{}{
										"secretKeyRef": map[string]interface{}{
											"name": secretName,
											"key":  "redis-password",
										},
									},
								},
							},
							"ports": []map[string]interface{}{
								{
									"containerPort": 6379,
									"name":          "redis",
								},
							},
							"volumeMounts": []map[string]interface{}{
								{
									"name":      "data",
									"mountPath": "/data",
								},
							},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    "50m",
									"memory": "64Mi",
								},
								"limits": map[string]interface{}{
									"cpu":    "500m",
									"memory": "256Mi",
								},
							},
						},
					},
				},
			},
			"volumeClaimTemplates": []map[string]interface{}{
				{
					"metadata": map[string]interface{}{
						"name": "data",
					},
					"spec": map[string]interface{}{
						"accessModes": []string{"ReadWriteOnce"},
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"storage": fmt.Sprintf("%dGi", storageGB),
							},
						},
					},
				},
			},
		},
	}
}

// buildGenericService returns an unstructured headless Service for a StatefulSet.
func buildGenericService(name, namespace string, port, containerPort int, portName string) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.zenith.dev/managed-service": name,
			},
		},
		"spec": map[string]interface{}{
			"clusterIP": "None",
			"selector": map[string]interface{}{
				"app.zenith.dev/managed-service": name,
			},
			"ports": []map[string]interface{}{
				{
					"port":       port,
					"targetPort": containerPort,
					"name":       portName,
				},
			},
		},
	}
}

// buildRedisService is kept for backward compat — delegates to buildGenericService.
func buildRedisService(name, namespace string, port int) map[string]interface{} {
	return buildGenericService(name, namespace, port, 6379, "redis")
}

// buildMySQLStatefulSet returns an unstructured MySQL StatefulSet manifest.
func buildMySQLStatefulSet(name, namespace, version, secretName string, storageGB int) map[string]interface{} {
	imageTag := version
	if !strings.Contains(imageTag, ".") {
		imageTag = version + "-debian"
	}

	return map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels":    map[string]interface{}{"app.zenith.dev/managed-service": name},
		},
		"spec": map[string]interface{}{
			"replicas":    1,
			"serviceName": name,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{"app.zenith.dev/managed-service": name},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{"app.zenith.dev/managed-service": name},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "mysql",
							"image": fmt.Sprintf("mysql:%s", imageTag),
							"env": []map[string]interface{}{
								{"name": "MYSQL_ROOT_PASSWORD", "valueFrom": map[string]interface{}{"secretKeyRef": map[string]interface{}{"name": secretName, "key": "mysql-root-password"}}},
								{"name": "MYSQL_USER", "valueFrom": map[string]interface{}{"secretKeyRef": map[string]interface{}{"name": secretName, "key": "mysql-user"}}},
								{"name": "MYSQL_PASSWORD", "valueFrom": map[string]interface{}{"secretKeyRef": map[string]interface{}{"name": secretName, "key": "mysql-password"}}},
								{"name": "MYSQL_DATABASE", "valueFrom": map[string]interface{}{"secretKeyRef": map[string]interface{}{"name": secretName, "key": "mysql-database"}}},
							},
							"ports":        []map[string]interface{}{{"containerPort": 3306, "name": "mysql"}},
							"volumeMounts": []map[string]interface{}{{"name": "data", "mountPath": "/var/lib/mysql"}},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{"cpu": "100m", "memory": "256Mi"},
								"limits":   map[string]interface{}{"cpu": "1000m", "memory": "1Gi"},
							},
						},
					},
				},
			},
			"volumeClaimTemplates": []map[string]interface{}{
				{
					"metadata": map[string]interface{}{"name": "data"},
					"spec": map[string]interface{}{
						"accessModes": []string{"ReadWriteOnce"},
						"resources":   map[string]interface{}{"requests": map[string]interface{}{"storage": fmt.Sprintf("%dGi", storageGB)}},
					},
				},
			},
		},
	}
}

// buildMongoDBStatefulSet returns an unstructured MongoDB StatefulSet manifest.
func buildMongoDBStatefulSet(name, namespace, version, secretName, dbName string, storageGB int) map[string]interface{} {
	imageTag := version
	if !strings.Contains(imageTag, ".") {
		imageTag = version + ".0"
	}

	return map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels":    map[string]interface{}{"app.zenith.dev/managed-service": name},
		},
		"spec": map[string]interface{}{
			"replicas":    1,
			"serviceName": name,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{"app.zenith.dev/managed-service": name},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{"app.zenith.dev/managed-service": name},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "mongo",
							"image": fmt.Sprintf("mongo:%s", imageTag),
							"env": []map[string]interface{}{
								{"name": "MONGO_INITDB_ROOT_USERNAME", "valueFrom": map[string]interface{}{"secretKeyRef": map[string]interface{}{"name": secretName, "key": "mongo-root-username"}}},
								{"name": "MONGO_INITDB_ROOT_PASSWORD", "valueFrom": map[string]interface{}{"secretKeyRef": map[string]interface{}{"name": secretName, "key": "mongo-root-password"}}},
								{"name": "MONGO_INITDB_DATABASE", "value": dbName},
							},
							"ports":        []map[string]interface{}{{"containerPort": 27017, "name": "mongo"}},
							"volumeMounts": []map[string]interface{}{{"name": "data", "mountPath": "/data/db"}},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{"cpu": "100m", "memory": "256Mi"},
								"limits":   map[string]interface{}{"cpu": "1000m", "memory": "1Gi"},
							},
						},
					},
				},
			},
			"volumeClaimTemplates": []map[string]interface{}{
				{
					"metadata": map[string]interface{}{"name": "data"},
					"spec": map[string]interface{}{
						"accessModes": []string{"ReadWriteOnce"},
						"resources":   map[string]interface{}{"requests": map[string]interface{}{"storage": fmt.Sprintf("%dGi", storageGB)}},
					},
				},
			},
		},
	}
}

// buildRabbitMQStatefulSet returns an unstructured RabbitMQ StatefulSet manifest.
func buildRabbitMQStatefulSet(name, namespace, version, secretName string, storageGB int) map[string]interface{} {
	imageTag := version
	if !strings.Contains(imageTag, "-") {
		imageTag = version + "-management-alpine"
	}

	return map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels":    map[string]interface{}{"app.zenith.dev/managed-service": name},
		},
		"spec": map[string]interface{}{
			"replicas":    1,
			"serviceName": name,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{"app.zenith.dev/managed-service": name},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{"app.zenith.dev/managed-service": name},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "rabbitmq",
							"image": fmt.Sprintf("rabbitmq:%s", imageTag),
							"env": []map[string]interface{}{
								{"name": "RABBITMQ_DEFAULT_USER", "valueFrom": map[string]interface{}{"secretKeyRef": map[string]interface{}{"name": secretName, "key": "rabbitmq-user"}}},
								{"name": "RABBITMQ_DEFAULT_PASS", "valueFrom": map[string]interface{}{"secretKeyRef": map[string]interface{}{"name": secretName, "key": "rabbitmq-password"}}},
							},
							"ports": []map[string]interface{}{
								{"containerPort": 5672, "name": "amqp"},
								{"containerPort": 15672, "name": "management"},
							},
							"volumeMounts": []map[string]interface{}{{"name": "data", "mountPath": "/var/lib/rabbitmq"}},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{"cpu": "100m", "memory": "256Mi"},
								"limits":   map[string]interface{}{"cpu": "1000m", "memory": "512Mi"},
							},
						},
					},
				},
			},
			"volumeClaimTemplates": []map[string]interface{}{
				{
					"metadata": map[string]interface{}{"name": "data"},
					"spec": map[string]interface{}{
						"accessModes": []string{"ReadWriteOnce"},
						"resources":   map[string]interface{}{"requests": map[string]interface{}{"storage": fmt.Sprintf("%dGi", storageGB)}},
					},
				},
			},
		},
	}
}
