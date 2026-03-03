package ports

import (
	"context"
	"encoding/json"
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
