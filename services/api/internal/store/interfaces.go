package store

import (
	"context"

	"github.com/dotechhq/zenith/services/api/internal/models"
)

// StoredUser wraps a User with the password hash (shared by all implementations).
type StoredUser struct {
	models.User
	PasswordHash string
}

// UserRepository defines user persistence operations.
type UserRepository interface {
	Create(ctx context.Context, email, password, name string, role models.Role) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*StoredUser, error)
	GetByID(ctx context.Context, id string) (*StoredUser, error)
	CheckPassword(user *StoredUser, password string) bool
	Count(ctx context.Context) (int, error)
}

// CustomerRepository defines customer and plan persistence operations.
type CustomerRepository interface {
	// Plans
	CreatePlan(ctx context.Context, input *models.CreatePlanInput) (*models.Plan, error)
	GetPlan(ctx context.Context, id string) (*models.Plan, error)
	ListPlans(ctx context.Context) ([]models.Plan, error)
	UpdatePlan(ctx context.Context, id string, input *models.UpdatePlanInput) (*models.Plan, error)

	// Customers
	CreateCustomer(ctx context.Context, input *models.CreateCustomerInput) (*models.Customer, error)
	GetCustomer(ctx context.Context, id string) (*models.Customer, error)
	ListCustomers(ctx context.Context) ([]models.Customer, error)
	UpdateCustomer(ctx context.Context, id string, input *models.UpdateCustomerInput) (*models.Customer, error)
	DeleteCustomer(ctx context.Context, id string) error
	SuspendCustomer(ctx context.Context, id string) (*models.Customer, error)
	ActivateCustomer(ctx context.Context, id string) (*models.Customer, error)

	// Stats
	GetCustomerStats(ctx context.Context) (*models.CustomerStats, error)

	// Cluster provisioning
	UpdateClusterStatus(ctx context.Context, id, status string) error
	SetCAPIClusterName(ctx context.Context, id, clusterName string) error
	UpdateClusterInfo(ctx context.Context, id string, nodes int, k8sVersion string) error
	GetCustomerByClusterName(ctx context.Context, clusterName string) (*models.Customer, error)
	ListProvisioningCustomers(ctx context.Context) ([]models.Customer, error)
}

// AdminRepository defines admin/platform persistence operations.
type AdminRepository interface {
	GetSettings(ctx context.Context) (*models.PlatformSettings, error)
	UpdateSettings(ctx context.Context, update *models.PlatformSettings) (*models.PlatformSettings, error)
	ListModules(ctx context.Context) ([]models.Module, error)
	GetModule(ctx context.Context, name string) (*models.Module, error)
	UpdateModule(ctx context.Context, name string) (*models.Module, error)
	ListAuditLog(ctx context.Context, limit, offset int) ([]models.AuditEntry, error)
	AddAuditEntry(ctx context.Context, entry models.AuditEntry) error
	GetPlatformUpdate(ctx context.Context) (*models.PlatformUpdate, error)
	ListUpdateHistory(ctx context.Context) ([]models.UpdateHistoryEntry, error)
}
