package dto

import "github.com/dotechhq/zenith/services/api/internal/entities"

// --- App DTOs ---

// CreateAppInput is the request body for creating a new app.
type CreateAppInput struct {
	UserID           string                `json:"user_id" validate:"required"`
	Name             string                `json:"name" validate:"required"`
	DeploySource     entities.DeploySource `json:"deploy_source"`
	RepoURL          string                `json:"repo_url"`
	Branch           string                `json:"branch"`
	ImageURL         string                `json:"image_url"`
	Port             int                   `json:"port"`
	RegistryUsername string                `json:"registry_username"`
	RegistryPassword string                `json:"registry_password"`
	AppType          entities.AppType      `json:"app_type"`
	Command          string                `json:"command"`
	CronSchedule     string                `json:"cron_schedule"`
}

// UpdateAppInput is the request body for updating an existing app.
type UpdateAppInput struct {
	Status    *entities.AppStatus `json:"status,omitempty"`
	Framework *entities.Framework `json:"framework,omitempty"`
	Port      *int                `json:"port,omitempty"`
	Branch    *string             `json:"branch,omitempty"`
}

// --- Deployment DTOs ---

// CreateReleaseInput is the payload sent by zenith-actions when a new image is ready.
type CreateReleaseInput struct {
	Image   string `json:"image"   validate:"required"`
	GitSHA  string `json:"git_sha"`
	Branch  string `json:"branch"`
	Message string `json:"message"`
}

// --- Secret DTOs ---

// CreateSecretInput is the API input for creating/updating a secret.
type CreateSecretInput struct {
	Key   string `json:"key"   validate:"required,max=256"`
	Value string `json:"value" validate:"required"`
}

// --- Metering DTOs ---

// MeteringInput is the POST body for recording a usage snapshot.
type MeteringInput struct {
	CustomerID  string  `json:"customerId"`
	CPUCores    float64 `json:"cpuCores"`
	RAMGB       float64 `json:"ramGb"`
	S3TB        float64 `json:"s3Tb"`
	DBStorageGB float64 `json:"dbStorageGb"`
	VolumeGB    float64 `json:"volumeGb"`
	LBCount     int     `json:"lbCount"`
}

// --- Customer DTOs ---

// CreatePlanInput is the request body for creating a plan.
type CreatePlanInput struct {
	Name         string `json:"name"`
	CPUCores     int    `json:"cpuCores"`
	RAMGB        int    `json:"ramGb"`
	S3TB         int    `json:"s3Tb"`
	DBStorageGB  int    `json:"dbStorageGb"`
	VolumeGB     int    `json:"volumeGb"`
	LBCount      int    `json:"lbCount"`
	StorageGB    int    `json:"storageGb"`
	S3StorageTB  int    `json:"s3StorageTb"`
	LoadBalancer int    `json:"loadBalancer"`
	PriceCents   int    `json:"priceCents"`
	Currency     string `json:"currency"`
	BillingCycle string `json:"billingCycle"`
}

// UpdatePlanInput is the request body for updating a plan.
type UpdatePlanInput struct {
	Name         *string `json:"name,omitempty"`
	CPUCores     *int    `json:"cpuCores,omitempty"`
	RAMGB        *int    `json:"ramGb,omitempty"`
	S3TB         *int    `json:"s3Tb,omitempty"`
	DBStorageGB  *int    `json:"dbStorageGb,omitempty"`
	VolumeGB     *int    `json:"volumeGb,omitempty"`
	LBCount      *int    `json:"lbCount,omitempty"`
	StorageGB    *int    `json:"storageGb,omitempty"`
	S3StorageTB  *int    `json:"s3StorageTb,omitempty"`
	LoadBalancer *int    `json:"loadBalancer,omitempty"`
	PriceCents   *int    `json:"priceCents,omitempty"`
	Currency     *string `json:"currency,omitempty"`
	BillingCycle *string `json:"billingCycle,omitempty"`
	Active       *bool   `json:"active,omitempty"`
}

// CreateCustomerInput is the request body for creating a customer.
type CreateCustomerInput struct {
	Name         string `json:"name"`
	Domain       string `json:"domain"`
	PlanID       string `json:"planId"`
	ContactEmail string `json:"contactEmail"`
	ContactName  string `json:"contactName"`
}

// UpdateCustomerInput is the request body for updating a customer.
type UpdateCustomerInput struct {
	Name         *string `json:"name,omitempty"`
	Domain       *string `json:"domain,omitempty"`
	PlanID       *string `json:"planId,omitempty"`
	ContactEmail *string `json:"contactEmail,omitempty"`
	ContactName  *string `json:"contactName,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

// ScaleClusterInput is the request body for scaling a customer cluster.
type ScaleClusterInput struct {
	Nodes int `json:"nodes"`
}

// --- Admin DTOs ---

// CreateClusterInput is the request body for creating a new cluster.
type CreateClusterInput struct {
	Name       string `json:"name"`
	Region     string `json:"region"`
	Type       string `json:"type"`
	Tenant     string `json:"tenant,omitempty"`
	Nodes      int    `json:"nodes"`
	K8sVersion string `json:"k8sVersion"`
}

// UpgradeClusterInput is the request body for upgrading a cluster.
type UpgradeClusterInput struct {
	Version string `json:"version"`
}

// ApplyUpdateInput is the request body for applying a platform update.
type ApplyUpdateInput struct {
	Version string `json:"version"`
}

// --- Database DTOs ---

// CreateDatabaseInput is the request body for provisioning a database for an app.
type CreateDatabaseInput struct {
	Engine entities.DatabaseEngine `json:"engine" validate:"required"`
	Name   string                 `json:"name"`
}

// DatabaseInfo is the response for a provisioned database (includes connection info).
type DatabaseInfo struct {
	ID               string                 `json:"id"`
	AppID            string                 `json:"app_id"`
	Name             string                 `json:"name"`
	Engine           entities.DatabaseEngine `json:"engine"`
	Host             string                 `json:"host"`
	Port             int                    `json:"port"`
	DBName           string                 `json:"db_name"`
	DBUser           string                 `json:"db_user"`
	ConnectionString string                 `json:"connection_string,omitempty"`
	SizeMB           int                    `json:"size_mb"`
	MaxSizeMB        int                    `json:"max_size_mb"`
	Status           entities.DatabaseStatus `json:"status"`
	CreatedAt        string                 `json:"created_at"`
}

// --- Storage DTOs ---

// CreateBucketInput is the request body for creating a storage bucket.
type CreateBucketInput struct {
	Name   string                 `json:"name"   validate:"required"`
	Access entities.BucketAccess  `json:"access"`
}

// BucketInfo is the response for a provisioned bucket.
type BucketInfo struct {
	ID        string                 `json:"id"`
	AppID     string                 `json:"app_id"`
	Name      string                 `json:"name"`
	Access    entities.BucketAccess  `json:"access"`
	Region    string                 `json:"region"`
	SizeMB    int                    `json:"size_mb"`
	MaxSizeMB int                    `json:"max_size_mb"`
	Objects   int                    `json:"objects"`
	Status    entities.BucketStatus  `json:"status"`
	Endpoint  string                 `json:"endpoint"`
	CreatedAt string                 `json:"created_at"`
}

// UpdateBucketInput is the request body for updating a bucket's access.
type UpdateBucketInput struct {
	Access entities.BucketAccess `json:"access" validate:"required"`
}

// ObjectEntry represents a single file or folder in a bucket listing.
type ObjectEntry struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified,omitempty"`
	ETag         string `json:"etag,omitempty"`
	IsFolder     bool   `json:"is_folder"`
}

// ListObjectsResponse is the response for listing objects in a bucket.
type ListObjectsResponse struct {
	Objects        []ObjectEntry `json:"objects"`
	CommonPrefixes []string      `json:"common_prefixes"`
	Prefix         string        `json:"prefix"`
	IsTruncated    bool          `json:"is_truncated"`
}

// PresignedURLResponse is the response containing a presigned URL.
type PresignedURLResponse struct {
	URL       string `json:"url"`
	Method    string `json:"method"`
	ExpiresIn int    `json:"expires_in"`
}

// UploadURLInput is the request body for generating a presigned upload URL.
type UploadURLInput struct {
	Key         string `json:"key"          validate:"required"`
	ContentType string `json:"content_type"`
}

// CreateFolderInput is the request body for creating a folder in a bucket.
type CreateFolderInput struct {
	Prefix string `json:"prefix" validate:"required"`
}

// --- App Auth DTOs ---

// AppAuthSignupInput is the request body for registering a user in an app's auth.
type AppAuthSignupInput struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name"     validate:"required"`
}

// AppAuthLoginInput is the request body for authenticating a user in an app's auth.
type AppAuthLoginInput struct {
	Email    string `json:"email"    validate:"required"`
	Password string `json:"password" validate:"required"`
}

// AppAuthTokenResponse is the response after a successful login/signup.
type AppAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// AppAuthConfigResponse is the response for an app's auth configuration.
type AppAuthConfigResponse struct {
	Enabled    bool `json:"enabled"`
	UserCount  int  `json:"user_count"`
	MaxUsers   int  `json:"max_users"`
}

// AppAuthUserResponse is the response for an app user.
type AppAuthUserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Verified  bool   `json:"verified"`
	CreatedAt string `json:"created_at"`
}

// --- Backup DTOs ---

// CreateBackupInput is the request body for creating a database backup.
type CreateBackupInput struct {
	Type entities.BackupType `json:"type"`
}

// BackupInfo is the response for a database backup.
type BackupInfo struct {
	ID         string               `json:"id"`
	DatabaseID string               `json:"database_id"`
	Type       entities.BackupType  `json:"type"`
	Status     entities.BackupStatus `json:"status"`
	SizeMB     int                  `json:"size_mb"`
	Error      string               `json:"error,omitempty"`
	CreatedAt  string               `json:"created_at"`
}

// RestoreBackupInput is the request body for restoring from a backup.
type RestoreBackupInput struct {
	BackupID string `json:"backup_id" validate:"required"`
}

// --- Plan DTOs ---

// UserPlanResponse is the response for a user's plan info.
type UserPlanResponse struct {
	Tier   entities.PlanTier   `json:"tier"`
	Limits entities.PlanLimits `json:"limits"`
	Usage  PlanUsage           `json:"usage"`
}

// PlanUsage shows current resource usage against plan limits.
type PlanUsage struct {
	Apps       int `json:"apps"`
	Databases  int `json:"databases"`
	StorageMB  int `json:"storage_mb"`
	AuthUsers  int `json:"auth_users"`
	Buckets    int `json:"buckets"`
}

// --- Domain DTOs ---

// AddDomainInput is the request body for adding a custom domain.
type AddDomainInput struct {
	Domain string `json:"domain" validate:"required"`
}

// DomainInfo is the response for a custom domain.
type DomainInfo struct {
	ID       string               `json:"id"`
	AppID    string               `json:"app_id"`
	Domain   string               `json:"domain"`
	Status   entities.DomainStatus `json:"status"`
	TLSReady bool                 `json:"tls_ready"`
	CreatedAt string              `json:"created_at"`
}

// --- Billing DTOs (Phase 6) ---

// CreateCheckoutInput is the request body for creating a Stripe checkout session.
type CreateCheckoutInput struct {
	Tier string `json:"tier" validate:"required"`
}

// CancelSubscriptionInput is the request body for canceling a subscription.
type CancelSubscriptionInput struct {
	Immediate bool `json:"immediate"`
}

// BillingStatusResponse is the response for the billing status endpoint.
type BillingStatusResponse struct {
	Tier              string      `json:"tier"`
	BillingStatus     string      `json:"billing_status"`
	PriceCents        int         `json:"price_cents"`
	Currency          string      `json:"currency"`
	PeriodEnd         *string     `json:"period_end,omitempty"`
	CancelAtPeriodEnd bool        `json:"cancel_at_period_end"`
	Limits            interface{} `json:"limits"`
	Usage             PlanUsage   `json:"usage"`
	StripeEnabled     bool        `json:"stripe_enabled"`
}

// CheckoutResponse is the response containing a Stripe checkout URL.
type CheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
	SessionID   string `json:"session_id"`
}

// PortalResponse is the response containing a Stripe portal URL.
type PortalResponse struct {
	PortalURL string `json:"portal_url"`
}

// InvoiceResponse is the response for a single invoice.
type InvoiceResponse struct {
	ID          string `json:"id"`
	AmountCents int    `json:"amount_cents"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
	InvoiceURL  string `json:"invoice_url,omitempty"`
	InvoicePDF  string `json:"invoice_pdf,omitempty"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	CreatedAt   string `json:"created_at"`
}

// AdminBillingOverviewResponse is the admin billing overview response.
type AdminBillingOverviewResponse struct {
	MRRCents            int     `json:"mrr_cents"`
	ActiveSubscriptions int     `json:"active_subscriptions"`
	PastDueCount        int     `json:"past_due_count"`
	CanceledThisMonth   int     `json:"canceled_this_month"`
	ChurnRatePercent    float64 `json:"churn_rate_percent"`
}

// --- Common DTOs ---

// Pagination represents pagination parameters and totals.
type Pagination struct {
	Page     int `json:"page" query:"page"`
	PageSize int `json:"page_size" query:"page_size"`
	Total    int `json:"total"`
}

// ListResponse wraps a paginated list of items.
type ListResponse[T any] struct {
	Items      []T        `json:"items"`
	Pagination Pagination `json:"pagination"`
}

// DefaultPagination returns sensible defaults.
func DefaultPagination() Pagination {
	return Pagination{
		Page:     1,
		PageSize: 20,
	}
}
