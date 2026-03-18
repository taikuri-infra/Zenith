package entities

// ServiceType represents the type of managed service.
type ServiceType string

const (
	ServiceTypePostgreSQL ServiceType = "postgresql"
	ServiceTypeRedis      ServiceType = "redis"
	ServiceTypeMySQL      ServiceType = "mysql"
	ServiceTypeMongoDB    ServiceType = "mongodb"
	ServiceTypeRabbitMQ   ServiceType = "rabbitmq"
)

// ValidServiceType returns true if the given service type is supported.
func ValidServiceType(st ServiceType) bool {
	switch st {
	case ServiceTypePostgreSQL, ServiceTypeRedis, ServiceTypeMySQL, ServiceTypeMongoDB, ServiceTypeRabbitMQ:
		return true
	}
	return false
}

// ManagedServiceStatus represents the lifecycle status of a managed service.
type ManagedServiceStatus string

const (
	ManagedServiceProvisioning ManagedServiceStatus = "provisioning"
	ManagedServiceReady        ManagedServiceStatus = "ready"
	ManagedServiceError        ManagedServiceStatus = "error"
	ManagedServiceDeleting     ManagedServiceStatus = "deleting"
)

// ManagedService represents a platform-provisioned database or cache for a project.
type ManagedService struct {
	ID              string               `json:"id"`
	ProjectID       string               `json:"project_id"`
	UserID          string               `json:"user_id"`
	ServiceType     ServiceType          `json:"service_type"`
	Name            string               `json:"name"`
	Version         string               `json:"version"`
	ConnectionURL   string               `json:"connection_url,omitempty"`
	InternalHost    string               `json:"internal_host,omitempty"`
	Port            int                  `json:"port"`
	Username        string               `json:"-"`
	Password        string               `json:"-"`
	DatabaseName    string               `json:"database_name,omitempty"`
	K8sNamespace    string               `json:"-"`
	K8sResourceName string               `json:"-"`
	Status          ManagedServiceStatus `json:"status"`
	StatusMessage   string               `json:"status_message,omitempty"`
	StorageGB       int                  `json:"storage_gb"`
	Timestamps
}

// DefaultPort returns the default port for the given service type.
func DefaultPort(st ServiceType) int {
	switch st {
	case ServiceTypePostgreSQL:
		return 5432
	case ServiceTypeRedis:
		return 6379
	case ServiceTypeMySQL:
		return 3306
	case ServiceTypeMongoDB:
		return 27017
	case ServiceTypeRabbitMQ:
		return 5672
	default:
		return 0
	}
}

// DefaultVersion returns the default version for each managed service type.
func DefaultVersion(st ServiceType) string {
	switch st {
	case ServiceTypePostgreSQL:
		return "16"
	case ServiceTypeRedis:
		return "7"
	case ServiceTypeMySQL:
		return "8"
	case ServiceTypeMongoDB:
		return "7"
	case ServiceTypeRabbitMQ:
		return "3"
	default:
		return "latest"
	}
}
