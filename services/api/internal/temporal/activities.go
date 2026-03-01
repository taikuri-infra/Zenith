package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/keycloakclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/s3client"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/jackc/pgx/v5"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// ProvisionInput is the workflow input for customer provisioning.
type ProvisionInput struct {
	CustomerID   string `json:"customerId"`
	CustomerName string `json:"customerName"`
	Domain       string `json:"domain"`
	PlanTier     string `json:"planTier"` // "free" or "pro"
	ContactEmail string `json:"contactEmail"`
}

// DeprovisionInput is the workflow input for customer teardown.
type DeprovisionInput struct {
	CustomerID   string `json:"customerId"`
	CustomerName string `json:"customerName"`
	Domain       string `json:"domain"`
	Namespace    string `json:"namespace"`
}

// Activities holds all dependencies for Temporal activity implementations.
type Activities struct {
	K8s        k8sclient.Client
	Keycloak   keycloakclient.KeycloakAPI
	S3         s3client.S3API
	AdminDSN   string // CNPG admin DSN for CREATE DATABASE
	Customers  ports.CustomerRepository
	Admin      ports.AdminRepository
	BaseDomain string
}

// --- helpers ---

func tenantNamespace(domain string) string {
	return "zenith-" + strings.ReplaceAll(strings.ReplaceAll(domain, ".", "-"), "_", "-")
}

func tenantDBName(domain string) string {
	return "z_" + strings.ReplaceAll(strings.ReplaceAll(domain, ".", "_"), "-", "_")
}

func tenantBucket(domain string) string {
	return "zenith-" + strings.ReplaceAll(domain, ".", "-")
}

func tenantRealm(domain string) string {
	return strings.ReplaceAll(domain, ".", "-")
}

// --- Activity: CreateKeycloakRealm ---

type CreateKeycloakRealmResult struct {
	RealmName    string `json:"realmName"`
	ClientSecret string `json:"clientSecret"`
}

func (a *Activities) CreateKeycloakRealm(ctx context.Context, input ProvisionInput) (*CreateKeycloakRealmResult, error) {
	realm := tenantRealm(input.Domain)

	if err := a.Keycloak.CreateRealm(ctx, realm, input.CustomerName); err != nil {
		return nil, fmt.Errorf("create realm: %w", err)
	}

	redirectURI := fmt.Sprintf("https://%s.%s/*", strings.ReplaceAll(input.Domain, ".", "-"), a.BaseDomain)
	secret, err := a.Keycloak.CreateClient(ctx, realm, "zenith-app", redirectURI)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	return &CreateKeycloakRealmResult{
		RealmName:    realm,
		ClientSecret: secret,
	}, nil
}

// --- Activity: CreateDatabase ---

type CreateDatabaseResult struct {
	DBName   string `json:"dbName"`
	DBUser   string `json:"dbUser"`
	DBPass   string `json:"dbPass"`
}

func (a *Activities) CreateDatabase(ctx context.Context, input ProvisionInput) (*CreateDatabaseResult, error) {
	if a.AdminDSN == "" {
		return &CreateDatabaseResult{DBName: "skipped"}, nil
	}

	dbName := tenantDBName(input.Domain)
	dbUser := dbName
	dbPass := fmt.Sprintf("zp_%s_%s", input.CustomerID[:8], input.Domain[:4])

	conn, err := pgx.Connect(ctx, a.AdminDSN)
	if err != nil {
		return nil, fmt.Errorf("connect to admin DB: %w", err)
	}
	defer conn.Close(ctx)

	// Create user and database (idempotent — ignore "already exists")
	createUser := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", pgx.Identifier{dbUser}.Sanitize(), dbPass)
	if _, err := conn.Exec(ctx, createUser); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("create user: %w", err)
		}
	}
	createDB := fmt.Sprintf("CREATE DATABASE %s OWNER %s", pgx.Identifier{dbName}.Sanitize(), pgx.Identifier{dbUser}.Sanitize())
	if _, err := conn.Exec(ctx, createDB); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("create database: %w", err)
		}
	}

	return &CreateDatabaseResult{
		DBName: dbName,
		DBUser: dbUser,
		DBPass: dbPass,
	}, nil
}

// --- Activity: CreateS3Bucket ---

type CreateS3BucketResult struct {
	BucketName string `json:"bucketName"`
}

func (a *Activities) CreateS3Bucket(ctx context.Context, input ProvisionInput) (*CreateS3BucketResult, error) {
	bucket := tenantBucket(input.Domain)
	if err := a.S3.CreateBucket(ctx, bucket); err != nil {
		return nil, err
	}
	return &CreateS3BucketResult{BucketName: bucket}, nil
}

// --- Activity: CreateNamespace ---

type CreateNamespaceResult struct {
	Namespace string `json:"namespace"`
}

func (a *Activities) CreateNamespace(ctx context.Context, input ProvisionInput) (*CreateNamespaceResult, error) {
	ns := tenantNamespace(input.Domain)
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "zenith",
		"zenith.dev/tenant":            input.Domain,
		"zenith.dev/customer-id":       input.CustomerID,
		"zenith.dev/plan":              input.PlanTier,
	}

	if err := a.K8s.CreateNamespace(ctx, ns, labels); err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("create namespace %s: %w", ns, err)
	}
	return &CreateNamespaceResult{Namespace: ns}, nil
}

// --- Activity: CreateSecrets ---

type CreateSecretsInput struct {
	ProvisionInput
	Namespace    string `json:"namespace"`
	DBName       string `json:"dbName"`
	DBUser       string `json:"dbUser"`
	DBPass       string `json:"dbPass"`
	ClientSecret string `json:"clientSecret"`
	RealmName    string `json:"realmName"`
	BucketName   string `json:"bucketName"`
}

func (a *Activities) CreateSecrets(ctx context.Context, input CreateSecretsInput) error {
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "zenith",
		"zenith.dev/tenant":            input.Domain,
	}

	// Database credentials secret
	if input.DBName != "" && input.DBName != "skipped" {
		if err := a.K8s.CreateSecret(ctx, input.Namespace, "db-credentials", map[string][]byte{
			"DB_NAME":     []byte(input.DBName),
			"DB_USER":     []byte(input.DBUser),
			"DB_PASSWORD": []byte(input.DBPass),
		}, labels); err != nil && !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("create db-credentials secret: %w", err)
		}
	}

	// Keycloak credentials secret
	if input.RealmName != "" {
		if err := a.K8s.CreateSecret(ctx, input.Namespace, "keycloak-credentials", map[string][]byte{
			"KEYCLOAK_REALM":         []byte(input.RealmName),
			"KEYCLOAK_CLIENT_ID":     []byte("zenith-app"),
			"KEYCLOAK_CLIENT_SECRET": []byte(input.ClientSecret),
		}, labels); err != nil && !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("create keycloak-credentials secret: %w", err)
		}
	}

	// S3 credentials secret
	if input.BucketName != "" {
		if err := a.K8s.CreateSecret(ctx, input.Namespace, "s3-credentials", map[string][]byte{
			"S3_BUCKET": []byte(input.BucketName),
		}, labels); err != nil && !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("create s3-credentials secret: %w", err)
		}
	}

	return nil
}

// --- Activity: CreateResourceQuota ---

func (a *Activities) CreateResourceQuota(ctx context.Context, input ProvisionInput, namespace string) error {
	var hard map[string]string
	if input.PlanTier == "pro" {
		hard = map[string]string{
			"requests.cpu":    "4",
			"requests.memory": "8Gi",
			"limits.cpu":      "8",
			"limits.memory":   "16Gi",
			"pods":            "50",
			"services":        "20",
		}
	} else {
		hard = map[string]string{
			"requests.cpu":    "1",
			"requests.memory": "2Gi",
			"limits.cpu":      "2",
			"limits.memory":   "4Gi",
			"pods":            "10",
			"services":        "5",
		}
	}

	if err := a.K8s.CreateResourceQuota(ctx, namespace, "tenant-quota", hard); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("create resource quota: %w", err)
	}

	limits := k8sclient.LimitRangeSpec{
		DefaultCPU:       "250m",
		DefaultMemory:    "256Mi",
		DefaultReqCPU:    "100m",
		DefaultReqMemory: "128Mi",
		MaxCPU:           "2",
		MaxMemory:        "4Gi",
		MinCPU:           "50m",
		MinMemory:        "64Mi",
	}
	if err := a.K8s.CreateLimitRange(ctx, namespace, "tenant-limits", limits); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("create limit range: %w", err)
	}

	return nil
}

// --- Activity: CreateRouting (Traefik IngressRoute CRD) ---

func (a *Activities) CreateRouting(ctx context.Context, input ProvisionInput, namespace string) error {
	host := fmt.Sprintf("%s.%s", strings.ReplaceAll(input.Domain, ".", "-"), a.BaseDomain)

	routeSpec := map[string]interface{}{
		"entryPoints": []string{"websecure"},
		"routes": []map[string]interface{}{
			{
				"match": fmt.Sprintf("Host(`%s`)", host),
				"kind":  "Rule",
				"services": []map[string]interface{}{
					{
						"name": "tenant-gateway",
						"port": 80,
					},
				},
			},
		},
		"tls": map[string]interface{}{
			"secretName": "tenant-tls",
		},
	}

	specBytes, _ := json.Marshal(routeSpec)
	route := &k8sclient.CRDObject{
		APIVersion: "traefik.io/v1alpha1",
		Kind:       "IngressRoute",
		Metadata: k8sclient.ObjectMeta{
			Name:      "tenant-route",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "zenith",
				"zenith.dev/tenant":            input.Domain,
			},
		},
		Spec: specBytes,
	}

	if err := a.K8s.CreateCRD(ctx, route); err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// --- Activity: CreateTLS (cert-manager Certificate CRD) ---

func (a *Activities) CreateTLS(ctx context.Context, input ProvisionInput, namespace string) error {
	host := fmt.Sprintf("%s.%s", strings.ReplaceAll(input.Domain, ".", "-"), a.BaseDomain)

	certSpec := map[string]interface{}{
		"secretName": "tenant-tls",
		"issuerRef": map[string]interface{}{
			"name": "letsencrypt-prod",
			"kind": "ClusterIssuer",
		},
		"dnsNames": []string{host},
	}

	specBytes, _ := json.Marshal(certSpec)
	cert := &k8sclient.CRDObject{
		APIVersion: "cert-manager.io/v1",
		Kind:       "Certificate",
		Metadata: k8sclient.ObjectMeta{
			Name:      "tenant-tls",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "zenith",
				"zenith.dev/tenant":            input.Domain,
			},
		},
		Spec: specBytes,
	}

	if err := a.K8s.CreateCRD(ctx, cert); err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// --- Activity: CreateArgoCD (ArgoCD Application CRD) ---

func (a *Activities) CreateArgoCD(ctx context.Context, input ProvisionInput, namespace string) error {
	appSpec := map[string]interface{}{
		"project": "default",
		"source": map[string]interface{}{
			"repoURL":        "https://github.com/dotechhq/zenith-tenant-chart",
			"targetRevision": "HEAD",
			"helm": map[string]interface{}{
				"values": fmt.Sprintf("tenant:\n  name: %s\n  domain: %s\n  namespace: %s\n  plan: %s\n",
					input.CustomerName, input.Domain, namespace, input.PlanTier),
			},
		},
		"destination": map[string]interface{}{
			"server":    "https://kubernetes.default.svc",
			"namespace": namespace,
		},
		"syncPolicy": map[string]interface{}{
			"automated": map[string]interface{}{
				"prune":    true,
				"selfHeal": true,
			},
		},
	}

	specBytes, _ := json.Marshal(appSpec)
	argoApp := &k8sclient.CRDObject{
		APIVersion: "argoproj.io/v1alpha1",
		Kind:       "Application",
		Metadata: k8sclient.ObjectMeta{
			Name:      "tenant-" + strings.ReplaceAll(input.Domain, ".", "-"),
			Namespace: "argocd",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "zenith",
				"zenith.dev/tenant":            input.Domain,
				"zenith.dev/customer-id":       input.CustomerID,
			},
		},
		Spec: specBytes,
	}

	if err := a.K8s.CreateCRD(ctx, argoApp); err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// --- Activity: NotifyReady ---

func (a *Activities) NotifyReady(ctx context.Context, input ProvisionInput, namespace string) error {
	if err := a.Customers.UpdateClusterStatus(ctx, input.CustomerID, "running"); err != nil {
		return fmt.Errorf("update customer status: %w", err)
	}

	_ = a.Admin.AddAuditEntry(ctx, auditEntry(
		"system",
		fmt.Sprintf("Tenant %s (%s) provisioned in namespace %s", input.CustomerName, input.Domain, namespace),
	))

	return nil
}

// --- Activity: UpdateStatusProvisioning ---

func (a *Activities) UpdateStatusProvisioning(ctx context.Context, customerID string) error {
	return a.Customers.UpdateClusterStatus(ctx, customerID, "provisioning")
}

// --- Activity: UpdateStatusError ---

func (a *Activities) UpdateStatusError(ctx context.Context, customerID string) error {
	return a.Customers.UpdateClusterStatus(ctx, customerID, "error")
}

// --- Deprovision Activities ---

func (a *Activities) DeleteKeycloakRealm(ctx context.Context, domain string) error {
	return a.Keycloak.DeleteRealm(ctx, tenantRealm(domain))
}

func (a *Activities) DeleteDatabase(ctx context.Context, domain string) error {
	if a.AdminDSN == "" {
		return nil
	}

	dbName := tenantDBName(domain)
	dbUser := dbName

	conn, err := pgx.Connect(ctx, a.AdminDSN)
	if err != nil {
		return fmt.Errorf("connect to admin DB: %w", err)
	}
	defer conn.Close(ctx)

	commands := []string{
		fmt.Sprintf("DROP DATABASE IF EXISTS %s", pgx.Identifier{dbName}.Sanitize()),
		fmt.Sprintf("DROP USER IF EXISTS %s", pgx.Identifier{dbUser}.Sanitize()),
	}
	for _, cmd := range commands {
		if _, err := conn.Exec(ctx, cmd); err != nil {
			return fmt.Errorf("exec %q: %w", cmd, err)
		}
	}
	return nil
}

func (a *Activities) DeleteS3Bucket(ctx context.Context, domain string) error {
	return a.S3.DeleteBucket(ctx, tenantBucket(domain))
}

func (a *Activities) DeleteNamespace(ctx context.Context, domain string) error {
	return a.K8s.DeleteNamespace(ctx, tenantNamespace(domain))
}

func (a *Activities) DeleteArgoCD(ctx context.Context, domain string) error {
	appName := "tenant-" + strings.ReplaceAll(domain, ".", "-")
	return a.K8s.DeleteCRDWithVersion(ctx, "argoproj.io/v1alpha1", "Application", "argocd", appName)
}
