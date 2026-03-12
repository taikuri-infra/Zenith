package ports

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// ---------------------------------------------------------------------------
// Kubernetes
// ---------------------------------------------------------------------------

// KubernetesClient abstracts Kubernetes API operations.
// Implemented by: adapters/k8sclient/client.go (MemoryClient), adapters/k8sclient/real_client.go (RealClient)
type KubernetesClient interface {
	// CRD operations
	CreateCRD(ctx context.Context, obj *K8sCRDObject) error
	GetCRD(ctx context.Context, kind, namespace, name string) (*K8sCRDObject, error)
	UpdateCRD(ctx context.Context, obj *K8sCRDObject) error
	PatchCRD(ctx context.Context, obj *K8sCRDObject) error
	DeleteCRD(ctx context.Context, kind, namespace, name string) error
	ListCRDs(ctx context.Context, kind, namespace string) ([]*K8sCRDObject, error)

	// Job operations (Kaniko builds)
	CreateJob(ctx context.Context, job *K8sJobObject) error
	GetJob(ctx context.Context, namespace, name string) (*K8sJobObject, error)
	DeleteJob(ctx context.Context, namespace, name string) error
	GetPodLogs(ctx context.Context, namespace, podSelector string, logCh chan<- string) error

	// ConfigMap operations
	CreateConfigMap(ctx context.Context, namespace, name string, data map[string]string) error
	DeleteConfigMap(ctx context.Context, namespace, name string) error

	// Namespace operations
	CreateNamespace(ctx context.Context, name string, labels map[string]string) error
	GetNamespace(ctx context.Context, name string) error
	DeleteNamespace(ctx context.Context, name string) error

	// Secret operations
	CreateSecret(ctx context.Context, namespace, name string, data map[string][]byte, labels map[string]string) error
	GetSecret(ctx context.Context, namespace, name string) (map[string][]byte, error)
	DeleteSecret(ctx context.Context, namespace, name string) error

	// ResourceQuota & LimitRange
	CreateResourceQuota(ctx context.Context, namespace, name string, hard map[string]string) error
	CreateLimitRange(ctx context.Context, namespace, name string, limits K8sLimitRangeSpec) error

	// Generic versioned CRD operations
	GetCRDWithVersion(ctx context.Context, apiVersion, kind, namespace, name string) (*K8sCRDObject, error)
	DeleteCRDWithVersion(ctx context.Context, apiVersion, kind, namespace, name string) error

	// Pod monitoring operations
	ListPods(ctx context.Context, namespace, labelSelector string) ([]K8sPodInfo, error)
	GetPodMetrics(ctx context.Context, namespace, labelSelector string) ([]K8sPodMetrics, error)

	// Node monitoring (for autoscaler cluster metrics)
	GetNodeMetrics(ctx context.Context) ([]K8sNodeMetrics, error)
}

// K8sPodInfo holds basic pod status information.
type K8sPodInfo struct {
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	Restarts         int32     `json:"restarts"`
	StartedAt        time.Time `json:"started_at"`
	Ready            bool      `json:"ready"`
	MemoryLimitBytes int64     `json:"memory_limit_bytes,omitempty"`
	StatusReason     string    `json:"status_reason,omitempty"`
	StatusMessage    string    `json:"status_message,omitempty"`
	LastExitCode     int32     `json:"last_exit_code,omitempty"`
}

// K8sPodMetrics holds pod-level resource usage from metrics-server.
type K8sPodMetrics struct {
	Name          string `json:"name"`
	CPUMillicores int64  `json:"cpu_millicores"`
	MemoryBytes   int64  `json:"memory_bytes"`
}

// K8sNodeMetrics holds node-level resource usage and capacity.
type K8sNodeMetrics struct {
	Name              string `json:"name"`
	CPUCapacityMillis int64  `json:"cpu_capacity_millis"`
	CPUUsageMillis    int64  `json:"cpu_usage_millis"`
	MemCapacityBytes  int64  `json:"mem_capacity_bytes"`
	MemUsageBytes     int64  `json:"mem_usage_bytes"`
}

// K8sCRDObject represents a Kubernetes custom resource at the port layer.
// Mirrors k8sclient.CRDObject; adapters convert between them.
type K8sCRDObject struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Metadata   K8sObjectMeta   `json:"metadata"`
	Spec       json.RawMessage `json:"spec"`
	Status     json.RawMessage `json:"status,omitempty"`
}

// K8sObjectMeta holds standard Kubernetes object metadata.
type K8sObjectMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// K8sJobObject represents a Kubernetes Job at the port layer.
type K8sJobObject struct {
	Name      string
	Namespace string
	Labels    map[string]string
	Spec      map[string]interface{}
	Active    int32
	Succeeded int32
	Failed    int32
}

// K8sLimitRangeSpec defines container resource defaults and limits.
type K8sLimitRangeSpec struct {
	DefaultCPU       string
	DefaultMemory    string
	DefaultReqCPU    string
	DefaultReqMemory string
	MaxCPU           string
	MaxMemory        string
	MinCPU           string
	MinMemory        string
}

// ---------------------------------------------------------------------------
// Payment Gateway (Stripe)
// ---------------------------------------------------------------------------

// PaymentGateway abstracts payment processing.
// Implemented by: adapters/stripeclient/client.go
//
// Note: ConstructWebhookEvent is intentionally excluded. Webhook signature
// verification stays in the handler layer with a direct Stripe dependency,
// since it needs the raw HTTP payload + signature header.
type PaymentGateway interface {
	CreateCheckoutSession(ctx context.Context, params CheckoutParams) (*CheckoutResult, error)
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (*PortalResult, error)
	CancelSubscription(ctx context.Context, subID string, atPeriodEnd bool) error
	GetSubscription(ctx context.Context, subID string) (*SubscriptionResult, error)
}

// CheckoutParams holds parameters for creating a checkout session.
type CheckoutParams struct {
	CustomerID string
	PriceID    string
	SuccessURL string
	CancelURL  string
	UserEmail  string
	Metadata   map[string]string
}

// CheckoutResult is the result of creating a checkout session.
type CheckoutResult struct {
	SessionID string
	URL       string
}

// PortalResult is the result of creating a billing portal session.
type PortalResult struct {
	URL string
}

// SubscriptionResult wraps relevant subscription fields.
type SubscriptionResult struct {
	ID                string
	CustomerID        string
	PriceID           string
	Status            string
	CurrentPeriodEnd  int64
	CancelAtPeriodEnd bool
}

// ---------------------------------------------------------------------------
// Object Storage (S3)
// ---------------------------------------------------------------------------

// ObjectStorage abstracts S3-compatible storage operations.
// Implemented by: adapters/s3client/client.go, adapters/s3client/memory.go
type ObjectStorage interface {
	CreateBucket(ctx context.Context, bucketName string) error
	DeleteBucket(ctx context.Context, bucketName string) error
	ListObjects(ctx context.Context, bucket, prefix, delimiter string, maxKeys int) (*ObjectListResult, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	GeneratePresignedUploadURL(ctx context.Context, bucket, key, contentType string, expiry time.Duration) (string, error)
	GeneratePresignedDownloadURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	PutObject(ctx context.Context, bucket, key, contentType string, body io.Reader, size int64) error
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, string, int64, error) // returns body, contentType, size
	CreateFolder(ctx context.Context, bucket, prefix string) error
}

// ObjectListResult holds the response from listing objects in a bucket.
type ObjectListResult struct {
	Objects        []ObjectInfo `json:"objects"`
	CommonPrefixes []string    `json:"common_prefixes"`
	Prefix         string      `json:"prefix"`
	IsTruncated    bool        `json:"is_truncated"`
}

// ObjectInfo holds metadata for a single S3 object.
type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	ETag         string    `json:"etag"`
}

// ---------------------------------------------------------------------------
// Identity Provider (Keycloak)
// ---------------------------------------------------------------------------

// IdentityProvider abstracts identity management.
// Implemented by: adapters/keycloakclient/client.go, adapters/keycloakclient/memory.go
type IdentityProvider interface {
	CreateRealm(ctx context.Context, realmName, displayName string) error
	DeleteRealm(ctx context.Context, realmName string) error
	CreateClient(ctx context.Context, realmName, clientID, redirectURI string) (secret string, err error)

	// User management (auth pools)
	CreateUser(ctx context.Context, realmName, email, password, firstName, lastName string) (userID string, err error)
	GetUser(ctx context.Context, realmName, userID string) (*IdentityUser, error)
	ListUsers(ctx context.Context, realmName string, first, max int) ([]IdentityUser, int, error)
	DeleteUser(ctx context.Context, realmName, userID string) error
	DisableUser(ctx context.Context, realmName, userID string) error
	EnableUser(ctx context.Context, realmName, userID string) error
	CountUsers(ctx context.Context, realmName string) (int, error)
}

// IdentityUser represents a user in an identity provider realm.
type IdentityUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Enabled       bool   `json:"enabled"`
	EmailVerified bool   `json:"email_verified"`
	CreatedAt     int64  `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Cluster Provisioner (CAPI)
// ---------------------------------------------------------------------------

// ClusterProvisioner abstracts CAPI cluster lifecycle operations.
// Implemented by: adapters/capiclient/client.go
type ClusterProvisioner interface {
	ListClusters(ctx context.Context) ([]entities.Cluster, error)
	GetCluster(ctx context.Context, name string) (*entities.Cluster, error)
	CreateCluster(ctx context.Context, input dto.CreateClusterInput) (*entities.Cluster, error)
	DeleteCluster(ctx context.Context, name string) error
	ScaleCluster(ctx context.Context, name string, nodes int) error
	UpgradeCluster(ctx context.Context, name, version string) error
}

// ---------------------------------------------------------------------------
// Cloud Provider (Hetzner)
// ---------------------------------------------------------------------------

// CloudProvider abstracts cloud infrastructure operations.
// Implemented by: adapters/hetznerclient/client.go
type CloudProvider interface {
	CreateServer(ctx context.Context, name, serverType, location, userData string) (*CloudServerResult, error)
	DeleteServer(ctx context.Context, serverID int64) error
	ListServers(ctx context.Context) ([]CloudServerResult, error)
	GetServer(ctx context.Context, serverID int64) (*CloudServerResult, error)
}

// CloudServerResult holds information about a cloud server.
type CloudServerResult struct {
	ID          int64
	Name        string
	PublicIPv4  string
	Status      string
	ServerType  string
	CPUCores    int
	RAMMB       int
	MonthlyCost float64
}

// ---------------------------------------------------------------------------
// Cluster Orchestrator (Provisioning / Deprovisioning)
// ---------------------------------------------------------------------------

// ClusterOrchestrator abstracts cluster lifecycle operations for customer management.
// Implemented by: cluster.Provisioner (CAPI-based goroutine fallback)
type ClusterOrchestrator interface {
	ProvisionCluster(ctx context.Context, customer *entities.Customer) error
	TeardownCluster(ctx context.Context, customer *entities.Customer) error
	GetCluster(ctx context.Context, clusterName string) (*entities.Cluster, error)
	ScaleCluster(ctx context.Context, customer *entities.Customer, nodes int) error
	UpgradeCluster(ctx context.Context, customer *entities.Customer, version string) error
}

// ProvisioningWorkflow abstracts Temporal-based provisioning.
// Implemented by: temporal.WorkflowClient (wraps go.temporal.io/sdk/client)
type ProvisioningWorkflow interface {
	StartProvision(ctx context.Context, input ProvisionInput) error
	StartDeprovision(ctx context.Context, input DeprovisionInput) error
}

// ProvisionInput holds parameters for tenant provisioning workflows.
type ProvisionInput struct {
	CustomerID   string
	CustomerName string
	Domain       string
	PlanTier     string
	ContactEmail string
}

// DeprovisionInput holds parameters for tenant teardown workflows.
type DeprovisionInput struct {
	CustomerID   string
	CustomerName string
	Domain       string
	Namespace    string
}

// ---------------------------------------------------------------------------
// Email Sender
// ---------------------------------------------------------------------------

// EmailSender abstracts email delivery.
// Implemented by: adapters/resendclient/client.go
type EmailSender interface {
	SendVerificationEmail(ctx context.Context, to, name, verificationURL string) error
	SendTeamInviteEmail(ctx context.Context, to, inviterName, teamName, inviteURL string) error
	SendSupportTicketNotification(ctx context.Context, to, ticketSubject, ticketURL string) error
	SendSupportReplyNotification(ctx context.Context, to, userName, ticketSubject, ticketURL string) error
	SendGenericEmail(ctx context.Context, to, subject, htmlBody string) error
}

// ---------------------------------------------------------------------------
// Event Bus (NATS JetStream)
// ---------------------------------------------------------------------------

// EventBus abstracts an event bus for publishing and subscribing to platform events.
// Implemented by: adapters/natsclient (real) and adapters/memory (in-memory).
type EventBus interface {
	// Publish sends an event to the bus.
	Publish(ctx context.Context, event *entities.PlatformEvent) error
	// Subscribe registers a handler for a subject pattern (e.g. "zenith.deploy.>").
	Subscribe(subject string, handler func(event *entities.PlatformEvent)) error
	// Close shuts down the connection.
	Close() error
}

// ---------------------------------------------------------------------------
// Token Generator (JWT)
// ---------------------------------------------------------------------------

// TokenGenerator abstracts JWT token operations.
// Implemented by: pkg/jwt/jwt.go (extracted from middleware)
type TokenGenerator interface {
	GenerateToken(user *entities.User, expiry time.Duration) (string, error)
	ParseToken(tokenString string) (*TokenClaims, error)
}

// TokenClaims holds parsed JWT claims.
type TokenClaims struct {
	Subject   string
	Email     string
	Name      string
	Role      entities.Role
	ProjectID string
	ExpiresAt time.Time
}
