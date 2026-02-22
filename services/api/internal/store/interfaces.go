package store

import (
	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/adapters/postgres"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// Type aliases for backward compatibility — all interfaces are defined in ports/.
// These aliases allow existing code to keep using store.XxxRepository until Phase 7.

type StoredUser = ports.StoredUser

type UserRepository = ports.UserRepository
type CustomerRepository = ports.CustomerRepository
type MeteringRepository = ports.MeteringRepository
type AppRepository = ports.AppRepository
type DatabaseRepository = ports.DatabaseRepository
type StorageRepository = ports.StorageRepository
type AppAuthRepository = ports.AppAuthRepository
type BackupRepository = ports.BackupRepository
type UserPlanRepository = ports.UserPlanRepository
type DomainRepository = ports.DomainRepository
type APIKeyRepository = ports.APIKeyRepository
type SessionRepository = ports.SessionRepository
type MFARepository = ports.MFARepository
type UserWebhookRepository = ports.UserWebhookRepository
type RoleRepository = ports.RoleRepository
type IPWhitelistRepository = ports.IPWhitelistRepository
type SSORepository = ports.SSORepository
type PreviewRepository = ports.PreviewRepository
type BrandingRepository = ports.BrandingRepository
type AutoscaleRepository = ports.AutoscaleRepository
type BillingRepository = ports.BillingRepository
type AdminRepository = ports.AdminRepository

// Constructor aliases — forward to adapters/memory and adapters/postgres.
// These keep store.NewMemoryXxx() and store.NewPostgresXxx() working until Phase 7.

var (
	NewMemoryAdminRepository       = memory.NewMemoryAdminRepository
	NewMemoryAPIKeyRepository      = memory.NewMemoryAPIKeyRepository
	NewMemoryAppRepository         = memory.NewMemoryAppRepository
	NewMemoryAppAuthRepository     = memory.NewMemoryAppAuthRepository
	NewMemoryAutoscaleRepository   = memory.NewMemoryAutoscaleRepository
	NewMemoryBackupRepository      = memory.NewMemoryBackupRepository
	NewMemoryBillingRepository     = memory.NewMemoryBillingRepository
	NewMemoryBrandingRepository    = memory.NewMemoryBrandingRepository
	NewMemoryCustomerRepository    = memory.NewMemoryCustomerRepository
	NewMemoryDatabaseRepository    = memory.NewMemoryDatabaseRepository
	NewMemoryDomainRepository      = memory.NewMemoryDomainRepository
	NewMemoryIPWhitelistRepository = memory.NewMemoryIPWhitelistRepository
	NewMemoryMeteringRepository    = memory.NewMemoryMeteringRepository
	NewMemoryMFARepository         = memory.NewMemoryMFARepository
	NewMemoryUserPlanRepository    = memory.NewMemoryUserPlanRepository
	NewMemoryPreviewRepository     = memory.NewMemoryPreviewRepository
	NewMemoryRoleRepository        = memory.NewMemoryRoleRepository
	NewMemorySessionRepository     = memory.NewMemorySessionRepository
	NewMemoryStorageRepository     = memory.NewMemoryStorageRepository
	NewMemorySSORepository         = memory.NewMemorySSORepository
	NewMemoryUserWebhookRepository = memory.NewMemoryUserWebhookRepository

	NewPostgresPool                = postgres.NewPostgresPool
	NewPostgresAdminRepository     = postgres.NewPostgresAdminRepository
	NewPostgresAppRepository       = postgres.NewPostgresAppRepository
	NewPostgresCustomerRepository  = postgres.NewPostgresCustomerRepository
	NewPostgresMeteringRepository  = postgres.NewPostgresMeteringRepository
	NewPostgresUserRepository      = postgres.NewPostgresUserRepository
)

// Concrete type aliases for code that type-asserts on implementations.
type MemoryDatabaseRepository = memory.MemoryDatabaseRepository
type MemoryAppRepository = memory.MemoryAppRepository
type MemoryCustomerRepository = memory.MemoryCustomerRepository

// Migrate alias for backward compatibility.
var RunMigrations = postgres.RunMigrations
