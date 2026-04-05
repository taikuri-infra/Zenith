package services

import (
	"context"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func newTestManagedServiceService() *ManagedServiceService {
	msRepo := memory.NewMemoryManagedServiceRepository()
	// k8s = nil → dev mode (marks services ready immediately)
	return NewManagedServiceService(msRepo, nil, "zenith-apps")
}

// --- NewManagedServiceService tests ---

func TestNewManagedServiceService(t *testing.T) {
	svc := newTestManagedServiceService()
	if svc == nil {
		t.Fatal("Expected non-nil ManagedServiceService")
	}
}

// --- ProvisionPostgreSQL tests ---

func TestProvisionPostgreSQL_DevMode(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	ms, err := svc.ProvisionPostgreSQL(ctx, "proj-1", "user-1", "my-postgres", "16", 10)
	if err != nil {
		t.Fatalf("ProvisionPostgreSQL failed: %v", err)
	}
	if ms == nil {
		t.Fatal("Expected non-nil managed service")
	}
	if ms.ServiceType != entities.ServiceTypePostgreSQL {
		t.Errorf("Expected type postgresql, got %s", ms.ServiceType)
	}
	if ms.Status != entities.ManagedServiceReady {
		t.Errorf("Expected status ready (dev mode), got %s", ms.Status)
	}
	if !strings.Contains(ms.ConnectionURL, "postgresql://") {
		t.Errorf("Expected postgresql:// connection URL, got '%s'", ms.ConnectionURL)
	}
	if ms.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", ms.Port)
	}
	if ms.Username == "" {
		t.Error("Expected non-empty username")
	}
	if ms.Password == "" {
		t.Error("Expected non-empty password")
	}
	if ms.DatabaseName == "" {
		t.Error("Expected non-empty database name")
	}
	if ms.Version != "16" {
		t.Errorf("Expected version '16', got '%s'", ms.Version)
	}
}

func TestProvisionPostgreSQL_DefaultVersion(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	ms, err := svc.ProvisionPostgreSQL(ctx, "proj-1", "user-1", "pg-default", "", 5)
	if err != nil {
		t.Fatalf("ProvisionPostgreSQL failed: %v", err)
	}
	if ms.Version != "16" {
		t.Errorf("Expected default version '16', got '%s'", ms.Version)
	}
}

func TestProvisionPostgreSQL_DuplicateName(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	_, err := svc.ProvisionPostgreSQL(ctx, "proj-1", "user-1", "dup-pg", "16", 10)
	if err != nil {
		t.Fatalf("First provision failed: %v", err)
	}

	_, err = svc.ProvisionPostgreSQL(ctx, "proj-1", "user-1", "dup-pg", "16", 10)
	if err == nil {
		t.Error("Expected error for duplicate name in same project")
	}
}

// --- ProvisionRedis tests ---

func TestProvisionRedis_DevMode(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	ms, err := svc.ProvisionRedis(ctx, "proj-1", "user-1", "my-redis", "7", 5)
	if err != nil {
		t.Fatalf("ProvisionRedis failed: %v", err)
	}
	if ms.ServiceType != entities.ServiceTypeRedis {
		t.Errorf("Expected type redis, got %s", ms.ServiceType)
	}
	if ms.Status != entities.ManagedServiceReady {
		t.Errorf("Expected status ready, got %s", ms.Status)
	}
	if !strings.Contains(ms.ConnectionURL, "redis://") {
		t.Errorf("Expected redis:// connection URL, got '%s'", ms.ConnectionURL)
	}
	if ms.Port != 6379 {
		t.Errorf("Expected port 6379, got %d", ms.Port)
	}
	if ms.Password == "" {
		t.Error("Expected non-empty password")
	}
}

func TestProvisionRedis_DefaultVersion(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	ms, _ := svc.ProvisionRedis(ctx, "proj-1", "user-1", "redis-def", "", 5)
	if ms.Version != "7" {
		t.Errorf("Expected default version '7', got '%s'", ms.Version)
	}
}

// --- ProvisionMySQL tests ---

func TestProvisionMySQL_DevMode(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	ms, err := svc.ProvisionMySQL(ctx, "proj-1", "user-1", "my-mysql", "8", 10)
	if err != nil {
		t.Fatalf("ProvisionMySQL failed: %v", err)
	}
	if ms.ServiceType != entities.ServiceTypeMySQL {
		t.Errorf("Expected type mysql, got %s", ms.ServiceType)
	}
	if ms.Status != entities.ManagedServiceReady {
		t.Errorf("Expected status ready, got %s", ms.Status)
	}
	if !strings.Contains(ms.ConnectionURL, "mysql://") {
		t.Errorf("Expected mysql:// connection URL, got '%s'", ms.ConnectionURL)
	}
	if ms.Port != 3306 {
		t.Errorf("Expected port 3306, got %d", ms.Port)
	}
	if ms.Username == "" {
		t.Error("Expected non-empty username")
	}
}

// --- ProvisionMongoDB tests ---

func TestProvisionMongoDB_DevMode(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	ms, err := svc.ProvisionMongoDB(ctx, "proj-1", "user-1", "my-mongo", "7", 10)
	if err != nil {
		t.Fatalf("ProvisionMongoDB failed: %v", err)
	}
	if ms.ServiceType != entities.ServiceTypeMongoDB {
		t.Errorf("Expected type mongodb, got %s", ms.ServiceType)
	}
	if ms.Status != entities.ManagedServiceReady {
		t.Errorf("Expected status ready, got %s", ms.Status)
	}
	if !strings.Contains(ms.ConnectionURL, "mongodb://") {
		t.Errorf("Expected mongodb:// connection URL, got '%s'", ms.ConnectionURL)
	}
	if ms.Port != 27017 {
		t.Errorf("Expected port 27017, got %d", ms.Port)
	}
}

// --- ProvisionRabbitMQ tests ---

func TestProvisionRabbitMQ_DevMode(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	ms, err := svc.ProvisionRabbitMQ(ctx, "proj-1", "user-1", "my-rabbit", "3", 5)
	if err != nil {
		t.Fatalf("ProvisionRabbitMQ failed: %v", err)
	}
	if ms.ServiceType != entities.ServiceTypeRabbitMQ {
		t.Errorf("Expected type rabbitmq, got %s", ms.ServiceType)
	}
	if ms.Status != entities.ManagedServiceReady {
		t.Errorf("Expected status ready, got %s", ms.Status)
	}
	if !strings.Contains(ms.ConnectionURL, "amqp://") {
		t.Errorf("Expected amqp:// connection URL, got '%s'", ms.ConnectionURL)
	}
	if ms.Port != 5672 {
		t.Errorf("Expected port 5672, got %d", ms.Port)
	}
	if ms.Username != "zenith" {
		t.Errorf("Expected username 'zenith', got '%s'", ms.Username)
	}
}

// --- DeleteManagedService tests ---

func TestDeleteManagedService_DevMode(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	ms, _ := svc.ProvisionPostgreSQL(ctx, "proj-1", "user-1", "to-delete", "16", 10)

	err := svc.DeleteManagedService(ctx, ms.ID)
	if err != nil {
		t.Fatalf("DeleteManagedService failed: %v", err)
	}

	// Verify it's gone
	list, _ := svc.ListByProject(ctx, "proj-1")
	for _, s := range list {
		if s.ID == ms.ID {
			t.Error("Managed service should have been deleted")
		}
	}
}

func TestDeleteManagedService_NotFound(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	err := svc.DeleteManagedService(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent managed service")
	}
}

// --- ListByProject tests ---

func TestListByProject_Empty(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	list, err := svc.ListByProject(ctx, "proj-1")
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Expected 0 services, got %d", len(list))
	}
}

func TestListByProject_MultipleServices(t *testing.T) {
	svc := newTestManagedServiceService()
	ctx := context.Background()

	svc.ProvisionPostgreSQL(ctx, "proj-1", "user-1", "pg", "16", 5)
	svc.ProvisionRedis(ctx, "proj-1", "user-1", "redis", "7", 5)
	svc.ProvisionMySQL(ctx, "proj-2", "user-1", "mysql", "8", 5)

	list, err := svc.ListByProject(ctx, "proj-1")
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("Expected 2 services for proj-1, got %d", len(list))
	}

	list2, _ := svc.ListByProject(ctx, "proj-2")
	if len(list2) != 1 {
		t.Errorf("Expected 1 service for proj-2, got %d", len(list2))
	}
}

// --- Provisioning with K8s client (exercises full code paths) ---

func newTestManagedServiceServiceWithK8s() *ManagedServiceService {
	msRepo := memory.NewMemoryManagedServiceRepository()
	k8s := NewK8sProvisionerAdapter(k8sclient.NewMemoryClient())
	return NewManagedServiceService(msRepo, k8s, "zenith-apps")
}

func TestProvisionPostgreSQL_WithK8s(t *testing.T) {
	svc := newTestManagedServiceServiceWithK8s()
	ctx := context.Background()

	ms, err := svc.ProvisionPostgreSQL(ctx, "proj-1", "user-1", "pg-k8s", "16", 10)
	if err != nil {
		t.Fatalf("ProvisionPostgreSQL with K8s failed: %v", err)
	}
	if ms == nil {
		t.Fatal("Expected non-nil managed service")
	}
	if ms.ServiceType != entities.ServiceTypePostgreSQL {
		t.Errorf("Expected type postgresql, got %s", ms.ServiceType)
	}
	// With K8s, PostgreSQL starts background polling, so status may still be provisioning
	if ms.K8sResourceName == "" {
		t.Error("Expected non-empty K8s resource name")
	}
}

func TestProvisionRedis_WithK8s(t *testing.T) {
	svc := newTestManagedServiceServiceWithK8s()
	ctx := context.Background()

	ms, err := svc.ProvisionRedis(ctx, "proj-1", "user-1", "redis-k8s", "7", 5)
	if err != nil {
		t.Fatalf("ProvisionRedis with K8s failed: %v", err)
	}
	if ms.Status != entities.ManagedServiceReady {
		t.Errorf("Expected status ready with K8s, got %s", ms.Status)
	}
}

func TestProvisionMySQL_WithK8s(t *testing.T) {
	svc := newTestManagedServiceServiceWithK8s()
	ctx := context.Background()

	ms, err := svc.ProvisionMySQL(ctx, "proj-1", "user-1", "mysql-k8s", "8", 10)
	if err != nil {
		t.Fatalf("ProvisionMySQL with K8s failed: %v", err)
	}
	if ms.Status != entities.ManagedServiceReady {
		t.Errorf("Expected status ready with K8s, got %s", ms.Status)
	}
}

func TestProvisionMongoDB_WithK8s(t *testing.T) {
	svc := newTestManagedServiceServiceWithK8s()
	ctx := context.Background()

	ms, err := svc.ProvisionMongoDB(ctx, "proj-1", "user-1", "mongo-k8s", "7", 10)
	if err != nil {
		t.Fatalf("ProvisionMongoDB with K8s failed: %v", err)
	}
	if ms.Status != entities.ManagedServiceReady {
		t.Errorf("Expected status ready with K8s, got %s", ms.Status)
	}
}

func TestProvisionRabbitMQ_WithK8s(t *testing.T) {
	svc := newTestManagedServiceServiceWithK8s()
	ctx := context.Background()

	ms, err := svc.ProvisionRabbitMQ(ctx, "proj-1", "user-1", "rabbit-k8s", "3", 5)
	if err != nil {
		t.Fatalf("ProvisionRabbitMQ with K8s failed: %v", err)
	}
	if ms.Status != entities.ManagedServiceReady {
		t.Errorf("Expected status ready with K8s, got %s", ms.Status)
	}
}

func TestDeleteManagedService_WithK8s_PostgreSQL(t *testing.T) {
	svc := newTestManagedServiceServiceWithK8s()
	ctx := context.Background()

	ms, _ := svc.ProvisionPostgreSQL(ctx, "proj-1", "user-1", "del-pg-k8s", "16", 10)
	err := svc.DeleteManagedService(ctx, ms.ID)
	if err != nil {
		t.Fatalf("DeleteManagedService (PostgreSQL) failed: %v", err)
	}
}

func TestDeleteManagedService_WithK8s_Redis(t *testing.T) {
	svc := newTestManagedServiceServiceWithK8s()
	ctx := context.Background()

	ms, _ := svc.ProvisionRedis(ctx, "proj-1", "user-1", "del-redis-k8s", "7", 5)
	err := svc.DeleteManagedService(ctx, ms.ID)
	if err != nil {
		t.Fatalf("DeleteManagedService (Redis) failed: %v", err)
	}
}

func TestDeleteManagedService_WithK8s_MySQL(t *testing.T) {
	svc := newTestManagedServiceServiceWithK8s()
	ctx := context.Background()

	ms, _ := svc.ProvisionMySQL(ctx, "proj-1", "user-1", "del-mysql-k8s", "8", 10)
	err := svc.DeleteManagedService(ctx, ms.ID)
	if err != nil {
		t.Fatalf("DeleteManagedService (MySQL) failed: %v", err)
	}
}

func TestDeleteManagedService_WithK8s_MongoDB(t *testing.T) {
	svc := newTestManagedServiceServiceWithK8s()
	ctx := context.Background()

	ms, _ := svc.ProvisionMongoDB(ctx, "proj-1", "user-1", "del-mongo-k8s", "7", 10)
	err := svc.DeleteManagedService(ctx, ms.ID)
	if err != nil {
		t.Fatalf("DeleteManagedService (MongoDB) failed: %v", err)
	}
}

func TestDeleteManagedService_WithK8s_RabbitMQ(t *testing.T) {
	svc := newTestManagedServiceServiceWithK8s()
	ctx := context.Background()

	ms, _ := svc.ProvisionRabbitMQ(ctx, "proj-1", "user-1", "del-rabbit-k8s", "3", 5)
	err := svc.DeleteManagedService(ctx, ms.ID)
	if err != nil {
		t.Fatalf("DeleteManagedService (RabbitMQ) failed: %v", err)
	}
}

// --- randomHex tests ---

func TestRandomHex_Length(t *testing.T) {
	h := randomHex(16)
	if len(h) != 32 {
		t.Errorf("Expected 32-char hex string, got %d chars", len(h))
	}
}

func TestRandomHex_Unique(t *testing.T) {
	h1 := randomHex(16)
	h2 := randomHex(16)
	if h1 == h2 {
		t.Error("Expected unique hex strings")
	}
}
