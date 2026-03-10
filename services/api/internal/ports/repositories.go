package ports

import (
	"context"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// StoredUser wraps a User with the password hash (shared by all implementations).
type StoredUser struct {
	entities.User
	PasswordHash string
}

// UserRepository defines user persistence operations.
type UserRepository interface {
	Create(ctx context.Context, email, password, name string, role entities.Role) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*StoredUser, error)
	GetByID(ctx context.Context, id string) (*StoredUser, error)
	CheckPassword(user *StoredUser, password string) bool
	Count(ctx context.Context) (int, error)

	// Email verification
	SetEmailVerified(ctx context.Context, userID string) error
	SetAuthProvider(ctx context.Context, userID, provider string) error
	CreateVerificationToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error
	GetVerificationToken(ctx context.Context, tokenHash string) (userID string, err error)
	DeleteVerificationTokens(ctx context.Context, userID string) error
}

// CustomerRepository defines customer and plan persistence operations.
type CustomerRepository interface {
	// Plans
	CreatePlan(ctx context.Context, input *dto.CreatePlanInput) (*entities.Plan, error)
	GetPlan(ctx context.Context, id string) (*entities.Plan, error)
	ListPlans(ctx context.Context) ([]entities.Plan, error)
	UpdatePlan(ctx context.Context, id string, input *dto.UpdatePlanInput) (*entities.Plan, error)

	// Customers
	CreateCustomer(ctx context.Context, input *dto.CreateCustomerInput) (*entities.Customer, error)
	GetCustomer(ctx context.Context, id string) (*entities.Customer, error)
	ListCustomers(ctx context.Context) ([]entities.Customer, error)
	UpdateCustomer(ctx context.Context, id string, input *dto.UpdateCustomerInput) (*entities.Customer, error)
	DeleteCustomer(ctx context.Context, id string) error
	SuspendCustomer(ctx context.Context, id string) (*entities.Customer, error)
	ActivateCustomer(ctx context.Context, id string) (*entities.Customer, error)

	// Stats
	GetCustomerStats(ctx context.Context) (*entities.CustomerStats, error)

	// Cluster provisioning
	UpdateClusterStatus(ctx context.Context, id, status string) error
	SetCAPIClusterName(ctx context.Context, id, clusterName string) error
	UpdateClusterInfo(ctx context.Context, id string, nodes int, k8sVersion string) error
	GetCustomerByClusterName(ctx context.Context, clusterName string) (*entities.Customer, error)
	ListProvisioningCustomers(ctx context.Context) ([]entities.Customer, error)
}

// MeteringRepository defines resource usage metering operations.
type MeteringRepository interface {
	RecordUsage(ctx context.Context, input *dto.MeteringInput) (*entities.ResourceUsage, error)
	GetLatestUsage(ctx context.Context, customerID string) (*entities.ResourceUsage, error)
	GetUsageHistory(ctx context.Context, customerID string, days int) ([]dto.UsageHistoryEntry, error)
	GetPlatformUsageSummary(ctx context.Context) (*dto.PlatformUsageSummary, error)
}

// ProjectRepository defines project persistence operations.
type ProjectRepository interface {
	CreateProject(ctx context.Context, userID, name, slug, description string) (*entities.Project, error)
	GetProject(ctx context.Context, id string) (*entities.Project, error)
	ListProjectsByUser(ctx context.Context, userID string) ([]entities.Project, error)
	UpdateProject(ctx context.Context, id string, name, description *string) (*entities.Project, error)
	DeleteProject(ctx context.Context, id string) error
	CountProjectsByUser(ctx context.Context, userID string) (int, error)
	GetDefaultProject(ctx context.Context, userID string) (*entities.Project, error)
}

// AppRepository defines app, deployment, and env var persistence operations.
type AppRepository interface {
	// Apps
	CreateApp(ctx context.Context, input *dto.CreateAppInput) (*entities.App, error)
	GetApp(ctx context.Context, id string) (*entities.App, error)
	GetAppBySubdomain(ctx context.Context, subdomain string) (*entities.App, error)
	ListAppsByUser(ctx context.Context, userID string) ([]entities.App, error)
	ListAppsByProject(ctx context.Context, projectID string) ([]entities.App, error)
	UpdateApp(ctx context.Context, id string, input *dto.UpdateAppInput) (*entities.App, error)
	DeleteApp(ctx context.Context, id string) error
	SetAutoGatewayID(ctx context.Context, appID, gatewayID string) error
	CountAppsByUser(ctx context.Context, userID string) (int, error)
	CountApps(ctx context.Context) (int, error)

	// Deployments
	CreateDeployment(ctx context.Context, appID, gitSHA string) (*entities.Deployment, error)
	GetDeployment(ctx context.Context, id string) (*entities.Deployment, error)
	ListDeployments(ctx context.Context, appID string, limit int) ([]entities.Deployment, error)
	UpdateDeploymentStatus(ctx context.Context, id string, status entities.DeploymentStatus, buildLog, errMsg string) error
	GetActiveDeployment(ctx context.Context, appID string) (*entities.Deployment, error)

	// Env Vars
	SetEnvVars(ctx context.Context, appID string, vars map[string]string) error
	GetEnvVars(ctx context.Context, appID string) ([]entities.EnvVar, error)
	DeleteEnvVar(ctx context.Context, appID, key string) error

	// Secrets (values are AES-256-GCM encrypted before storage)
	SetSecret(ctx context.Context, appID, key string, encryptedValue []byte) error
	GetSecrets(ctx context.Context, appID string) ([]entities.Secret, error)
	GetSecretValue(ctx context.Context, appID, key string) ([]byte, error)
	DeleteSecret(ctx context.Context, appID, key string) error

	// Releases (image versions registered by zenith-actions / CI)
	CreateRelease(ctx context.Context, appID string, input *dto.CreateReleaseInput) (*entities.Release, error)
	ListReleases(ctx context.Context, appID string, limit int) ([]entities.Release, error)
	GetRelease(ctx context.Context, id string) (*entities.Release, error)
}

// DatabaseRepository defines per-app database provisioning operations.
type DatabaseRepository interface {
	CreateDatabase(ctx context.Context, appID, userID string, input *dto.CreateDatabaseInput) (*entities.UserDatabase, error)
	GetDatabase(ctx context.Context, id string) (*entities.UserDatabase, error)
	ListDatabasesByApp(ctx context.Context, appID string) ([]entities.UserDatabase, error)
	ListDatabasesByUser(ctx context.Context, userID string) ([]entities.UserDatabase, error)
	ListDatabasesByProject(ctx context.Context, projectID string) ([]entities.UserDatabase, error)
	DeleteDatabase(ctx context.Context, id string) error
	UpdateDatabaseStatus(ctx context.Context, id string, status entities.DatabaseStatus) error
	CountDatabasesByUser(ctx context.Context, userID string) (int, error)
	CountDatabases(ctx context.Context) (int, error)
}

// StorageRepository defines per-app storage bucket operations.
type StorageRepository interface {
	CreateBucket(ctx context.Context, appID, userID string, input *dto.CreateBucketInput) (*entities.UserBucket, error)
	GetBucket(ctx context.Context, id string) (*entities.UserBucket, error)
	GetBucketByName(ctx context.Context, userID, name string) (*entities.UserBucket, error)
	ListBucketsByApp(ctx context.Context, appID string) ([]entities.UserBucket, error)
	ListBucketsByUser(ctx context.Context, userID string) ([]entities.UserBucket, error)
	ListBucketsByProject(ctx context.Context, projectID string) ([]entities.UserBucket, error)
	UpdateBucket(ctx context.Context, id string, access entities.BucketAccess) (*entities.UserBucket, error)
	DeleteBucket(ctx context.Context, id string) error
	CountBucketsByUser(ctx context.Context, userID string) (int, error)
}

// AppAuthRepository defines per-app authentication operations.
type AppAuthRepository interface {
	// Config
	EnableAuth(ctx context.Context, appID string, maxUsers int) (*entities.AppAuthConfig, error)
	DisableAuth(ctx context.Context, appID string) error
	GetAuthConfig(ctx context.Context, appID string) (*entities.AppAuthConfig, error)

	// Users
	CreateAppUser(ctx context.Context, appID, email, password, name string) (*entities.AppUser, error)
	GetAppUserByEmail(ctx context.Context, appID, email string) (*entities.AppUser, string, error) // returns user + passwordHash
	CountAppUsers(ctx context.Context, appID string) (int, error)
	ListAppUsers(ctx context.Context, appID string, limit, offset int) ([]entities.AppUser, error)
	DeleteAppUser(ctx context.Context, appID, userID string) error
}

// BackupRepository defines database backup operations.
type BackupRepository interface {
	CreateBackup(ctx context.Context, databaseID, userID string, backupType entities.BackupType) (*entities.DatabaseBackup, error)
	GetBackup(ctx context.Context, id string) (*entities.DatabaseBackup, error)
	ListBackupsByDatabase(ctx context.Context, databaseID string) ([]entities.DatabaseBackup, error)
	ListBackupsByUser(ctx context.Context, userID string) ([]entities.DatabaseBackup, error)
	UpdateBackupStatus(ctx context.Context, id string, status entities.BackupStatus, sizeMB int, errMsg string) error
	DeleteBackup(ctx context.Context, id string) error
	CountBackupsByUser(ctx context.Context, userID string) (int, error)
}

// UserPlanRepository defines user plan tracking operations.
type UserPlanRepository interface {
	GetUserPlan(ctx context.Context, userID string) (*entities.UserPlan, error)
	SetUserPlan(ctx context.Context, userID string, tier entities.PlanTier) (*entities.UserPlan, error)
	ListUsersByPlan(ctx context.Context, tier entities.PlanTier) ([]entities.UserPlan, error)
	ListAllPlans(ctx context.Context) ([]entities.UserPlan, error)
}

// DomainRepository defines custom domain operations.
type DomainRepository interface {
	AddDomain(ctx context.Context, appID, userID, domain string) (*entities.CustomDomain, error)
	GetDomain(ctx context.Context, id string) (*entities.CustomDomain, error)
	ListDomainsByApp(ctx context.Context, appID string) ([]entities.CustomDomain, error)
	ListDomainsByUser(ctx context.Context, userID string) ([]entities.CustomDomain, error)
	UpdateDomainStatus(ctx context.Context, id string, status entities.DomainStatus, tlsReady bool) error
	DeleteDomain(ctx context.Context, id string) error
}

// TeamMemberRepository defines team member persistence operations.
type TeamMemberRepository interface {
	CreateMember(ctx context.Context, member *entities.TeamMember) error
	GetMember(ctx context.Context, id string) (*entities.TeamMember, error)
	GetMemberByEmail(ctx context.Context, accountID, email string) (*entities.TeamMember, error)
	GetMemberByUserID(ctx context.Context, userID string) (*entities.TeamMember, error)
	GetMemberByInviteHash(ctx context.Context, hash string) (*entities.TeamMember, error)
	ListMembers(ctx context.Context, accountID string) ([]entities.TeamMember, error)
	UpdateMember(ctx context.Context, member *entities.TeamMember) error
	DeleteMember(ctx context.Context, id string) error
	CountMembers(ctx context.Context, accountID string) (int, error)
}

// APIKeyRepository defines API key management operations.
type APIKeyRepository interface {
	CreateAPIKey(ctx context.Context, userID, name string, scopes []string) (*entities.APIKey, error)
	GetAPIKey(ctx context.Context, id string) (*entities.APIKey, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*entities.APIKey, error)
	ListAPIKeysByUser(ctx context.Context, userID string) ([]entities.APIKey, error)
	DeleteAPIKey(ctx context.Context, id string) error
	UpdateLastUsed(ctx context.Context, id string) error
	CountAPIKeysByUser(ctx context.Context, userID string) (int, error)
}

// SessionRepository defines user session tracking operations.
type SessionRepository interface {
	CreateSession(ctx context.Context, userID, ipAddress, userAgent string) (*entities.Session, error)
	GetSession(ctx context.Context, id string) (*entities.Session, error)
	ListSessionsByUser(ctx context.Context, userID string) ([]entities.Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteAllUserSessions(ctx context.Context, userID string) error
	UpdateLastSeen(ctx context.Context, id string) error
}

// MFARepository defines MFA enrollment operations.
type MFARepository interface {
	GetEnrollment(ctx context.Context, userID string) (*entities.MFAEnrollment, error)
	StartEnrollment(ctx context.Context, userID string) (*entities.MFAEnrollment, error)
	ConfirmEnrollment(ctx context.Context, userID string) (*entities.MFAEnrollment, error)
	DisableEnrollment(ctx context.Context, userID string) error
	UseBackupCode(ctx context.Context, userID, code string) (bool, error)
	RegenerateBackupCodes(ctx context.Context, userID string) ([]string, error)
}

// UserWebhookRepository defines user-defined webhook operations.
type UserWebhookRepository interface {
	CreateWebhook(ctx context.Context, userID, url string, events []entities.WebhookEvent) (*entities.UserWebhook, error)
	GetWebhook(ctx context.Context, id string) (*entities.UserWebhook, error)
	ListWebhooksByUser(ctx context.Context, userID string) ([]entities.UserWebhook, error)
	UpdateWebhook(ctx context.Context, id string, url *string, events []entities.WebhookEvent, active *bool) (*entities.UserWebhook, error)
	DeleteWebhook(ctx context.Context, id string) error
	CountWebhooksByUser(ctx context.Context, userID string) (int, error)
	RecordDelivery(ctx context.Context, webhookID string, event entities.WebhookEvent, payload string, status entities.WebhookDeliveryStatus, statusCode int, errMsg string) (*entities.WebhookDelivery, error)
	ListDeliveries(ctx context.Context, webhookID string, limit int) ([]entities.WebhookDelivery, error)
}

// RoleRepository defines custom role (RBAC) operations.
type RoleRepository interface {
	CreateRole(ctx context.Context, userID, name, description string, permissions []entities.Permission) (*entities.CustomRole, error)
	GetRole(ctx context.Context, id string) (*entities.CustomRole, error)
	ListRolesByUser(ctx context.Context, userID string) ([]entities.CustomRole, error)
	UpdateRole(ctx context.Context, id string, name, description *string, permissions []entities.Permission) (*entities.CustomRole, error)
	DeleteRole(ctx context.Context, id string) error
	AssignRole(ctx context.Context, roleID, memberID, assignedBy string) (*entities.RoleAssignment, error)
	ListAssignmentsByRole(ctx context.Context, roleID string) ([]entities.RoleAssignment, error)
	RemoveAssignment(ctx context.Context, assignmentID string) error
}

// IPWhitelistRepository defines IP whitelist operations.
type IPWhitelistRepository interface {
	AddEntry(ctx context.Context, userID, cidr, description string) (*entities.IPWhitelistEntry, error)
	GetEntry(ctx context.Context, id string) (*entities.IPWhitelistEntry, error)
	ListByUser(ctx context.Context, userID string) ([]entities.IPWhitelistEntry, error)
	DeleteEntry(ctx context.Context, id string) error
	IsIPAllowed(ctx context.Context, userID, ip string) (bool, error)
}

// SSORepository defines SSO configuration operations.
type SSORepository interface {
	CreateConfig(ctx context.Context, userID string, provider entities.SSOProvider, config *entities.SSOConfig) (*entities.SSOConfig, error)
	GetConfig(ctx context.Context, id string) (*entities.SSOConfig, error)
	ListConfigsByUser(ctx context.Context, userID string) ([]entities.SSOConfig, error)
	DeleteConfig(ctx context.Context, id string) error
	ToggleConfig(ctx context.Context, id string, enabled bool) (*entities.SSOConfig, error)
}

// PreviewRepository defines preview deployment operations.
type PreviewRepository interface {
	CreatePreview(ctx context.Context, appID string, prNumber int, branch, gitSHA, url string) (*entities.PreviewDeployment, error)
	GetPreview(ctx context.Context, id string) (*entities.PreviewDeployment, error)
	ListPreviewsByApp(ctx context.Context, appID string) ([]entities.PreviewDeployment, error)
	DeletePreview(ctx context.Context, id string) error
}

// BrandingRepository defines DPA and white-label branding operations.
type BrandingRepository interface {
	GetDPA(ctx context.Context, userID string) (*entities.DPARecord, error)
	SignDPA(ctx context.Context, userID, signedBy, ipAddress string) (*entities.DPARecord, error)
	GetBranding(ctx context.Context, userID string) (*entities.BrandingConfig, error)
	UpdateBranding(ctx context.Context, userID string, companyName, logoURL, primaryColor *string, hideBranding *bool) (*entities.BrandingConfig, error)
	SetDashboardDomain(ctx context.Context, userID, domain string) (*entities.BrandingConfig, error)
}

// GatewayRepository defines gateway and gateway route persistence operations.
type GatewayRepository interface {
	// Gateways
	CreateGateway(ctx context.Context, userID, projectID, name, slug string) (*entities.Gateway, error)
	GetGateway(ctx context.Context, id string) (*entities.Gateway, error)
	GetGatewayBySlug(ctx context.Context, slug string) (*entities.Gateway, error)
	GetGatewayByProject(ctx context.Context, projectID string) (*entities.Gateway, error)
	ListGatewaysByUser(ctx context.Context, userID string) ([]entities.Gateway, error)
	ListGatewaysByProject(ctx context.Context, projectID string) ([]entities.Gateway, error)
	UpdateGateway(ctx context.Context, id, name string) (*entities.Gateway, error)
	DeleteGateway(ctx context.Context, id string) error
	CountGatewaysByUser(ctx context.Context, userID string) (int, error)
	UpdateGatewayStatus(ctx context.Context, id string, status entities.GatewayStatus) error

	// Routes
	CreateRoute(ctx context.Context, route *entities.GatewayRoute) (*entities.GatewayRoute, error)
	GetRoute(ctx context.Context, id string) (*entities.GatewayRoute, error)
	ListRoutesByGateway(ctx context.Context, gatewayID string) ([]entities.GatewayRoute, error)
	ListActiveRoutesByGateway(ctx context.Context, gatewayID string) ([]entities.GatewayRoute, error)
	UpdateRoute(ctx context.Context, route *entities.GatewayRoute) (*entities.GatewayRoute, error)
	DeleteRoute(ctx context.Context, id string) error
	CountRoutesByGateway(ctx context.Context, gatewayID string) (int, error)
	CountRoutesByUser(ctx context.Context, userID string) (int, error)
	StopRoutesByApp(ctx context.Context, appID string) ([]string, error) // returns affected gateway IDs
	ClearAuthPoolFromRoutes(ctx context.Context, authPoolID string) ([]string, error) // returns affected gateway IDs
	ListRoutesByAuthPool(ctx context.Context, authPoolID string) ([]entities.GatewayRoute, error)

	// Groups
	CreateGroup(ctx context.Context, group *entities.GatewayGroup) (*entities.GatewayGroup, error)
	GetGroup(ctx context.Context, id string) (*entities.GatewayGroup, error)
	ListGroupsByGateway(ctx context.Context, gatewayID string) ([]entities.GatewayGroup, error)
	UpdateGroup(ctx context.Context, group *entities.GatewayGroup) (*entities.GatewayGroup, error)
	DeleteGroup(ctx context.Context, id string) error
	StopGroupsByApp(ctx context.Context, appID string) ([]string, error) // returns affected gateway IDs
}

// AutoscaleRepository defines autoscaler node and event persistence operations.
type AutoscaleRepository interface {
	SaveNode(ctx context.Context, node *entities.HetznerNode) error
	DeleteNode(ctx context.Context, serverID int64) error
	ListNodes(ctx context.Context) ([]entities.HetznerNode, error)
	LogScaleEvent(ctx context.Context, event *entities.AutoscaleEvent) error
	ListScaleEvents(ctx context.Context, limit int) ([]entities.AutoscaleEvent, error)
	GetStatus(ctx context.Context) (*entities.AutoscalerStatus, error)
	UpdateStatus(ctx context.Context, status *entities.AutoscalerStatus) error
}

// BillingRepository defines subscription and invoice persistence operations.
type BillingRepository interface {
	// Subscriptions
	CreateSubscription(ctx context.Context, sub *entities.Subscription) error
	GetSubscriptionByUser(ctx context.Context, userID string) (*entities.Subscription, error)
	GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*entities.Subscription, error)
	UpdateSubscriptionStatus(ctx context.Context, stripeSubID string, status entities.SubscriptionStatus) error
	UpdateSubscriptionTier(ctx context.Context, stripeSubID string, tier entities.PlanTier, priceID string) error

	// Customer mapping
	SetStripeCustomerID(ctx context.Context, userID, customerID string) error
	GetStripeCustomerID(ctx context.Context, userID string) (string, error)
	GetUserByStripeCustomerID(ctx context.Context, customerID string) (string, error)

	// Invoices
	UpsertInvoice(ctx context.Context, inv *entities.Invoice) error
	ListInvoicesByUser(ctx context.Context, userID string) ([]entities.Invoice, error)

	// Admin
	GetBillingOverview(ctx context.Context) (*entities.BillingOverview, error)
}

// AuthPoolRepository defines auth pool persistence operations.
type AuthPoolRepository interface {
	CreatePool(ctx context.Context, id, userID, projectID, name, realmName, clientID, clientSecret, issuerURL string, maxUsers int) (*entities.AuthPool, error)
	GetPool(ctx context.Context, id string) (*entities.AuthPool, error)
	ListPoolsByUser(ctx context.Context, userID string) ([]entities.AuthPool, error)
	DeletePool(ctx context.Context, id string) error
	UpdatePoolStatus(ctx context.Context, id string, status entities.AuthPoolStatus) error
	UpdatePoolUserCount(ctx context.Context, id string, delta int) error
	CountPoolsByUser(ctx context.Context, userID string) (int, error)
}

// SupportRepository defines support ticket persistence operations.
type SupportRepository interface {
	CreateTicket(ctx context.Context, ticket *entities.SupportTicket, initialMsg *entities.SupportMessage) error
	GetTicket(ctx context.Context, id string) (*entities.SupportTicket, error)
	ListTicketsByUser(ctx context.Context, userID string) ([]entities.SupportTicket, error)
	ListAllTickets(ctx context.Context, status string, limit, offset int) ([]entities.SupportTicket, int, error)
	UpdateTicketStatus(ctx context.Context, id string, status entities.TicketStatus) error
	UpdateTicketAssignee(ctx context.Context, id, adminUserID string) error
	AddMessage(ctx context.Context, msg *entities.SupportMessage) error
	ListMessages(ctx context.Context, ticketID string) ([]entities.SupportMessage, error)
}

// AdminRepository defines admin/platform persistence operations.
type AdminRepository interface {
	GetSettings(ctx context.Context) (*entities.PlatformSettings, error)
	UpdateSettings(ctx context.Context, update *entities.PlatformSettings) (*entities.PlatformSettings, error)
	ListModules(ctx context.Context) ([]entities.Module, error)
	GetModule(ctx context.Context, name string) (*entities.Module, error)
	UpdateModule(ctx context.Context, name string) (*entities.Module, error)
	ListAuditLog(ctx context.Context, limit, offset int) ([]entities.AuditEntry, error)
	AddAuditEntry(ctx context.Context, entry entities.AuditEntry) error
	GetPlatformUpdate(ctx context.Context) (*entities.PlatformUpdate, error)
	ListUpdateHistory(ctx context.Context) ([]entities.UpdateHistoryEntry, error)
}

// NotificationRepository stores user notifications and activity log entries.
type NotificationRepository interface {
	CreateNotification(ctx context.Context, notif *entities.Notification) error
	ListByUser(ctx context.Context, userID string, limit int) ([]entities.Notification, error)
	MarkRead(ctx context.Context, userID string, ids []string) error
	MarkAllRead(ctx context.Context, userID string) error
	CountUnread(ctx context.Context, userID string) (int, error)

	AddActivity(ctx context.Context, entry *entities.ActivityEntry) error
	ListActivity(ctx context.Context, userID string, limit int) ([]entities.ActivityEntry, error)
}

// PodExecSessionRepository stores terminal session audit records.
type PodExecSessionRepository interface {
	CreateSession(ctx context.Context, session *entities.PodExecSession) error
	EndSession(ctx context.Context, id string, recordingKey string) error
	GetSession(ctx context.Context, id string) (*entities.PodExecSession, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]entities.PodExecSession, int, error)
	ListAll(ctx context.Context, limit, offset int) ([]entities.PodExecSession, int, error)
}
