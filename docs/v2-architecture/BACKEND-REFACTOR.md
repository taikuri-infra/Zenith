# Backend Refactoring — Implementation Plan

> **Status:** Ready to Execute
> **Last Updated:** 2026-03-03
> **Author:** Babak + Claude (Backend Architecture Session)
> **Parent Doc:** [10-backend-architecture.md](./10-backend-architecture.md)

---

## Table of Contents

1. [Pre-Flight Checklist](#pre-flight-checklist)
2. [Phase 1: Create `ports/infrastructure.go`](#phase-1-create-portsinfrastructurego)
3. [Phase 2: Delete `services/auth/` Dead Code](#phase-2-delete-servicesauth-dead-code)
4. [Phase 3: Refactor Services to Use Port Interfaces](#phase-3-refactor-services-to-use-port-interfaces)
5. [Phase 4: Move Domain Modules Under `services/`](#phase-4-move-domain-modules-under-services)
6. [Phase 5: Update `cmd/server/main.go` Wiring](#phase-5-update-cmdservermain-go-wiring)
7. [Phase 6: Build, Push, Deploy](#phase-6-build-push-deploy)
8. [Phase 7: E2E Tests](#phase-7-e2e-tests)

---

## Pre-Flight Checklist

Before making any changes, verify the codebase compiles and all existing tests pass.

- [ ] **Verify Go build succeeds**
  ```bash
  cd services/api && go build ./...
  ```
  **Expected:** No errors.

- [ ] **Verify all tests pass**
  ```bash
  cd services/api && go test ./...
  ```
  **Expected:** All tests PASS (no failures).

- [ ] **Verify operator builds separately** (not affected by API refactor)
  ```bash
  cd services/operator && go build ./...
  ```
  **Expected:** No errors.

- [ ] **Git status is clean**
  ```bash
  git status
  ```
  **Expected:** No uncommitted changes (or stash them first).

- [ ] **Create feature branch**
  ```bash
  git checkout -b refactor/backend-clean-architecture
  ```

---

## Phase 1: Create `ports/infrastructure.go`

**Goal:** Define all infrastructure port interfaces in one file. This is purely additive — no existing code changes.

**Risk:** None. No imports change. No tests break.

### Task 1.1: Create `ports/infrastructure.go`

- [ ] Create file: `services/api/internal/ports/infrastructure.go`

```go
package ports

import (
	"context"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// ---------------------------------------------------------------------------
// Kubernetes
// ---------------------------------------------------------------------------

// KubernetesClient abstracts Kubernetes API operations.
// Implemented by: adapters/k8sclient/real_client.go, adapters/k8sclient/memory_client.go
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

// K8sCRDObject, K8sJobObject, K8sLimitRangeSpec are port-level types.
// These mirror the adapter types but live in ports/ so services don't import adapters.
// NOTE: During refactoring, these can be type aliases to the adapter types initially,
// then gradually decoupled. See Phase 3 notes.

// ---------------------------------------------------------------------------
// Payment Gateway (Stripe)
// ---------------------------------------------------------------------------

// PaymentGateway abstracts payment processing.
// Implemented by: adapters/stripeclient/client.go
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
	// Realm management
	CreateRealm(ctx context.Context, realmName, displayName string) error
	DeleteRealm(ctx context.Context, realmName string) error

	// Client management
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
	Subject   string // user ID
	Email     string
	Name      string
	Role      string
	ExpiresAt time.Time
}
```

### Task 1.2: Verify build still passes

- [ ] Run build to confirm new file compiles:
  ```bash
  cd services/api && go build ./...
  ```
  **Expected:** No errors. The new file only defines interfaces — nothing imports it yet.

### Task 1.3: Commit

- [ ] Commit:
  ```bash
  git add services/api/internal/ports/infrastructure.go
  git commit -m "feat(api): add ports/infrastructure.go with all infra interfaces"
  ```

**Decision Note — K8s Port Types:**

The `KubernetesClient` interface references `K8sCRDObject`, `K8sJobObject`, and `K8sLimitRangeSpec`. These types need to be defined in `ports/`. Two approaches:

1. **Type aliases** (quick): `type K8sCRDObject = k8sclient.CRDObject` — works but creates a dependency on the adapter package from ports, which defeats the purpose.
2. **Duplicate types** (clean): Define `K8sCRDObject` etc. directly in `ports/infrastructure.go` with the same fields. Services use port types. Adapters convert between port types and their internal types.

**Recommendation:** Start with option 2 (duplicate types). The K8s types are simple structs with JSON fields — duplicating them is 20 lines and keeps ports clean. Add conversion functions in the adapter package.

---

## Phase 2: Delete `services/auth/` Dead Code

**Goal:** Remove the dead OIDC prototype service that was replaced by Keycloak.

**Risk:** None. No live code references this module.

### Task 2.1: Verify nothing imports `services/auth/`

- [ ] Search for imports:
  ```bash
  grep -r "services/auth" services/api/ infra/ apps/
  ```
  **Expected:** Zero results (except maybe this doc or CLAUDE.md references).

- [ ] Check Dockerfiles:
  ```bash
  grep -r "services/auth" services/api/Dockerfile infra/helm/
  ```
  **Expected:** Zero results.

### Task 2.2: Delete the directory

- [ ] Remove dead code:
  ```bash
  rm -rf services/auth/
  ```

### Task 2.3: Verify build still passes

- [ ] Run build:
  ```bash
  cd services/api && go build ./...
  ```
  **Expected:** No errors.

### Task 2.4: Commit

- [ ] Commit:
  ```bash
  git add -A services/auth/
  git commit -m "chore(api): remove dead services/auth/ OIDC prototype (replaced by Keycloak)"
  ```

---

## Phase 3: Refactor Services to Use Port Interfaces

**Goal:** Fix all service-layer violations. Services depend on `ports/` interfaces, not adapter packages.

**Risk:** Medium — changes function signatures and constructors. Requires updating tests and `main.go`.

### Task 3.1: Extract JWT to `pkg/jwt/`

**Current violation:** `services/auth.go` imports `middleware.GenerateToken()` and `middleware.ParseToken()`.

- [ ] Create file: `services/api/pkg/jwt/jwt.go`

```go
package jwt

import (
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT payload.
type Claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// GenerateToken creates a signed JWT for the given user.
func GenerateToken(secret string, user *entities.User, expiry time.Duration) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Email: user.Email,
		Name:  user.Name,
		Role:  string(user.Role),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken validates and parses a JWT token.
func ParseToken(secret, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
```

- [ ] Update `services/auth.go`: replace `middleware.GenerateToken` → `jwt.GenerateToken`, `middleware.ParseToken` → `jwt.ParseToken`
  ```diff
  import (
  -    "github.com/dotechhq/zenith/services/api/internal/middleware"
  +    zenithJWT "github.com/dotechhq/zenith/services/api/pkg/jwt"
  )

  func (s *AuthService) issueTokens(user *entities.User) (*TokenPair, error) {
  -    accessToken, err := middleware.GenerateToken(s.jwtSecret, user, AccessTokenExpiry)
  +    accessToken, err := zenithJWT.GenerateToken(s.jwtSecret, user, AccessTokenExpiry)
       ...
  -    refreshToken, err := middleware.GenerateToken(s.jwtSecret, user, RefreshTokenExpiry)
  +    refreshToken, err := zenithJWT.GenerateToken(s.jwtSecret, user, RefreshTokenExpiry)
       ...
  }

  func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
  -    claims, err := middleware.ParseToken(s.jwtSecret, refreshToken)
  +    claims, err := zenithJWT.ParseToken(s.jwtSecret, refreshToken)
       ...
  }
  ```

- [ ] Update `middleware/auth.go`: import from `pkg/jwt` instead of duplicating logic
  ```diff
  import (
  +    zenithJWT "github.com/dotechhq/zenith/services/api/pkg/jwt"
  )

  // GenerateToken wraps pkg/jwt for backward compatibility.
  func GenerateToken(secret string, user *entities.User, expiry time.Duration) (string, error) {
  -    // ... inline implementation
  +    return zenithJWT.GenerateToken(secret, user, expiry)
  }

  func ParseToken(secret, tokenString string) (*Claims, error) {
  -    // ... inline implementation
  +    parsed, err := zenithJWT.ParseToken(secret, tokenString)
  +    if err != nil {
  +        return nil, err
  +    }
  +    return &Claims{...convert...}, nil
  }
  ```

- [ ] Verify:
  ```bash
  cd services/api && go build ./... && go test ./...
  ```

### Task 3.2: Refactor `BillingService` — remove adapter imports

**Current violation:** Imports `stripeclient.StripeAPI` and `s3client.S3API`.

- [ ] Update `services/billing.go`:
  ```diff
  import (
  -    "github.com/dotechhq/zenith/services/api/internal/adapters/s3client"
  -    stripeClient "github.com/dotechhq/zenith/services/api/internal/adapters/stripeclient"
  +    "github.com/dotechhq/zenith/services/api/internal/ports"
  )

  type BillingService struct {
  -    stripe      stripeClient.StripeAPI
  +    payments    ports.PaymentGateway
       billingRepo ports.BillingRepository
       planRepo    ports.UserPlanRepository
       appRepo     ports.AppRepository
       dbRepo      ports.DatabaseRepository
       storageRepo ports.StorageRepository
       authRepo    ports.AppAuthRepository
  -    s3          s3client.S3API
  +    storage     ports.ObjectStorage
       proPriceID  string
       teamPriceID string
       baseDomain  string
  }
  ```

- [ ] Update `NewBillingService()` constructor signature
- [ ] Update all method bodies: `s.stripe` → `s.payments`, `s.s3` → `s.storage`
- [ ] Update `CheckoutParams` usage: `stripeClient.CheckoutParams` → `ports.CheckoutParams`
- [ ] Update `SetS3()` → `SetStorage()`
- [ ] Update `Stripe()` accessor → `Payments()` (used by webhook handler)
- [ ] Verify:
  ```bash
  cd services/api && go build ./... && go test ./...
  ```

### Task 3.3: Refactor `AdminService` — remove adapter imports

**Current violation:** Imports `capiclient.Client` and `k8sclient.Client`.

- [ ] Update `services/admin.go`:
  ```diff
  import (
  -    "github.com/dotechhq/zenith/services/api/internal/adapters/capiclient"
  -    "github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
  +    "github.com/dotechhq/zenith/services/api/internal/ports"
  )

  type AdminService struct {
  -    capiClient *capiclient.Client
  -    k8sClient  k8sclient.Client
  +    clusters   ports.ClusterProvisioner
  +    k8s        ports.KubernetesClient
       store      ports.AdminRepository
  }
  ```

- [ ] Update `NewAdminService()` constructor
- [ ] Update all method bodies: `s.capiClient.ListClusters()` → `s.clusters.ListClusters()`, etc.
- [ ] Verify:
  ```bash
  cd services/api && go build ./... && go test ./...
  ```

### Task 3.4: Refactor `CustomerService` — remove domain module imports

**Current violation:** Imports `cluster.Provisioner` and `temporal.Client`.

- [ ] Update `services/customer.go`:
  ```diff
  import (
  -    "github.com/dotechhq/zenith/services/api/internal/cluster"
  -    zenithTemporal "github.com/dotechhq/zenith/services/api/internal/temporal"
  -    "go.temporal.io/sdk/client"
  +    "github.com/dotechhq/zenith/services/api/internal/ports"
  )

  type CustomerService struct {
       store          ports.CustomerRepository
       admin          ports.AdminRepository
  -    provisioner    *cluster.Provisioner
  -    temporalClient client.Client
  +    clusters       ports.ClusterProvisioner
  +    provisioning   ports.ProvisioningWorkflow
  }
  ```

  **Note:** `ports.ProvisioningWorkflow` wraps the Temporal client:
  ```go
  // In ports/infrastructure.go:
  type ProvisioningWorkflow interface {
      StartProvision(ctx context.Context, input ProvisionInput) error
      StartDeprovision(ctx context.Context, input DeprovisionInput) error
  }
  ```

- [ ] Update `NewCustomerService()` constructor
- [ ] Wrap the existing Temporal client call in a `ProvisioningWorkflow` adapter
- [ ] Verify:
  ```bash
  cd services/api && go build ./... && go test ./...
  ```

### Task 3.5: Add compile-time interface checks to adapters

- [ ] Add to each adapter file:
  ```go
  // adapters/stripeclient/client.go
  var _ ports.PaymentGateway = (*Client)(nil)

  // adapters/s3client/client.go
  var _ ports.ObjectStorage = (*Client)(nil)
  var _ ports.ObjectStorage = (*MemoryS3Client)(nil)

  // adapters/keycloakclient/client.go
  var _ ports.IdentityProvider = (*Client)(nil)
  var _ ports.IdentityProvider = (*MemoryKeycloakClient)(nil)

  // adapters/capiclient/client.go
  var _ ports.ClusterProvisioner = (*Client)(nil)

  // adapters/hetznerclient/client.go
  var _ ports.CloudProvider = (*Client)(nil)
  ```

- [ ] Fix any method signature mismatches (the compiler will tell you)
- [ ] Verify:
  ```bash
  cd services/api && go build ./... && go test ./...
  ```

### Task 3.6: Commit Phase 3

- [ ] Commit:
  ```bash
  git add -A
  git commit -m "refactor(api): services depend on port interfaces, not adapter packages

  - Extract JWT to pkg/jwt/ (auth.go no longer imports middleware)
  - BillingService uses ports.PaymentGateway + ports.ObjectStorage
  - AdminService uses ports.ClusterProvisioner + ports.KubernetesClient
  - CustomerService uses ports.ProvisioningWorkflow
  - Add compile-time interface checks to all adapters"
  ```

---

## Phase 4: Move Domain Modules Under `services/`

**Goal:** Reorganize `deploy/`, `cluster/`, `autoscale/`, `temporal/` under `services/` to reflect that they contain business logic.

**Risk:** Medium — Go package paths change, all imports must be updated.

### Task 4.1: Move `deploy/` → `services/deploy/`

- [ ] Move directory:
  ```bash
  mv services/api/internal/deploy services/api/internal/services/deploy
  ```

- [ ] Update package declaration in all files:
  ```diff
  - package deploy
  + package deploy  // (stays the same — package name doesn't change)
  ```
  **Note:** Package name stays `deploy` — only the import path changes.

- [ ] Find and update all imports:
  ```bash
  grep -r '"github.com/dotechhq/zenith/services/api/internal/deploy"' services/api/
  ```
  Replace with:
  ```
  "github.com/dotechhq/zenith/services/api/internal/services/deploy"
  ```

- [ ] Verify: `cd services/api && go build ./...`

### Task 4.2: Merge `cluster/` + `temporal/` → `services/provisioning/`

- [ ] Create directory:
  ```bash
  mkdir -p services/api/internal/services/provisioning
  ```

- [ ] Move cluster files:
  ```bash
  mv services/api/internal/cluster/*.go services/api/internal/services/provisioning/
  ```

- [ ] Move temporal files:
  ```bash
  mv services/api/internal/temporal/*.go services/api/internal/services/provisioning/
  ```

- [ ] Update package declarations in all moved files:
  ```diff
  - package cluster
  + package provisioning
  ```
  ```diff
  - package temporal
  + package provisioning
  ```

- [ ] Rename any conflicting types/functions (unlikely but check)

- [ ] Find and update all imports:
  ```bash
  grep -r '"github.com/dotechhq/zenith/services/api/internal/cluster"' services/api/
  grep -r '"github.com/dotechhq/zenith/services/api/internal/temporal"' services/api/
  ```
  Replace both with:
  ```
  "github.com/dotechhq/zenith/services/api/internal/services/provisioning"
  ```

- [ ] Remove old directories:
  ```bash
  rm -rf services/api/internal/cluster services/api/internal/temporal
  ```

- [ ] Verify: `cd services/api && go build ./...`

### Task 4.3: Move `autoscale/` → `services/autoscale/`

- [ ] Move directory:
  ```bash
  mv services/api/internal/autoscale services/api/internal/services/autoscale
  ```

- [ ] Update all imports:
  ```bash
  grep -r '"github.com/dotechhq/zenith/services/api/internal/autoscale"' services/api/
  ```
  Replace with:
  ```
  "github.com/dotechhq/zenith/services/api/internal/services/autoscale"
  ```

- [ ] Verify: `cd services/api && go build ./...`

### Task 4.4: Commit Phase 4

- [ ] Commit:
  ```bash
  git add -A
  git commit -m "refactor(api): move domain modules under services/

  - deploy/ → services/deploy/
  - cluster/ + temporal/ → services/provisioning/ (merged)
  - autoscale/ → services/autoscale/"
  ```

---

## Phase 5: Update `cmd/server/main.go` Wiring

**Goal:** Update the composition root to use the refactored types and import paths.

**Risk:** Low — mechanical import path updates.

### Task 5.1: Update import paths in `main.go`

- [ ] Update all import paths:
  ```diff
  import (
  -    "github.com/dotechhq/zenith/services/api/internal/deploy"
  -    "github.com/dotechhq/zenith/services/api/internal/cluster"
  -    "github.com/dotechhq/zenith/services/api/internal/autoscale"
  -    zenithTemporal "github.com/dotechhq/zenith/services/api/internal/temporal"
  +    "github.com/dotechhq/zenith/services/api/internal/services/deploy"
  +    "github.com/dotechhq/zenith/services/api/internal/services/provisioning"
  +    "github.com/dotechhq/zenith/services/api/internal/services/autoscale"
  )
  ```

### Task 5.2: Update service constructors to use port interfaces

- [ ] Update `NewBillingService()` call to pass `stripeAPI` as `ports.PaymentGateway`
- [ ] Update `NewAdminService()` call to pass `capiClient` as `ports.ClusterProvisioner`
- [ ] Update `NewCustomerService()` call to pass provisioning workflow interface

### Task 5.3: Verify full build + tests

- [ ] Run build and all tests:
  ```bash
  cd services/api && go build ./... && go test ./...
  ```
  **Expected:** All pass.

### Task 5.4: Commit Phase 5

- [ ] Commit:
  ```bash
  git add -A
  git commit -m "refactor(api): update main.go wiring for new package structure"
  ```

---

## Phase 6: Build, Push, Deploy

**Goal:** Build the refactored API, push to Harbor, deploy via ArgoCD on staging.

**Risk:** Low — same binary, same behavior, just cleaner code.

### Task 6.1: Build Docker image

- [ ] Build on staging server:
  ```bash
  ssh zen-stage
  cd /path/to/zenith
  git pull origin refactor/backend-clean-architecture

  docker build -t registry.stage.freezenith.com/zenith/zenith-api:0.5.0 \
    -f services/api/Dockerfile .
  ```
  **Expected:** Build succeeds.

### Task 6.2: Push to Harbor

- [ ] Push image:
  ```bash
  docker push registry.stage.freezenith.com/zenith/zenith-api:0.5.0
  ```

### Task 6.3: Update Helm values for ArgoCD

- [ ] Update the API image tag in Helm values:
  ```yaml
  # infra/helm/zenith-api/values-staging.yaml (or similar)
  image:
    tag: "0.5.0"
  ```

### Task 6.4: Merge to staging and let ArgoCD sync

- [ ] Merge to staging branch:
  ```bash
  git checkout staging
  git merge refactor/backend-clean-architecture
  git push origin staging
  ```

- [ ] Verify ArgoCD syncs:
  ```bash
  ssh zen-stage
  kubectl get pods -n zenith-staging -l app=zenith-api
  ```
  **Expected:** New pod running with image tag `0.5.0`.

### Task 6.5: Smoke test

- [ ] Health check:
  ```bash
  curl -s https://api.stage.freezenith.com/api/v1/health | jq .
  ```
  **Expected:** `{"status":"ok"}`

- [ ] Register:
  ```bash
  curl -s -X POST https://api.stage.freezenith.com/api/v1/auth/register \
    -H 'Content-Type: application/json' \
    -d '{"email":"test@test.com","password":"testpass123","name":"Test"}' | jq .
  ```
  **Expected:** Token response.

---

## Phase 7: E2E Tests

**Goal:** Comprehensive testing of all API endpoints post-refactor.

**Risk:** Medium — may surface hidden coupling or regressions.

### Task 7.1: Core auth flow

- [ ] Register new user
- [ ] Login with credentials
- [ ] Refresh token
- [ ] Access protected endpoint with JWT

### Task 7.2: App lifecycle

- [ ] Create app (POST /api/v1/apps)
- [ ] List apps (GET /api/v1/apps)
- [ ] Get app (GET /api/v1/apps/:id)
- [ ] Delete app (DELETE /api/v1/apps/:id)
- [ ] Verify plan limit enforcement (create apps up to limit, verify 403)

### Task 7.3: Billing flow

- [ ] Get billing status (GET /api/v1/billing)
- [ ] Create checkout session (POST /api/v1/billing/checkout)
- [ ] Verify Stripe webhook handling (if Stripe enabled)
- [ ] Create portal session (POST /api/v1/billing/portal)

### Task 7.4: Admin endpoints

- [ ] Get dashboard stats (GET /api/v1/admin/dashboard/stats)
- [ ] List clusters (GET /api/v1/admin/clusters)
- [ ] Customer CRUD (POST/GET/PUT/DELETE /api/v1/admin/customers)

### Task 7.5: Database & storage

- [ ] Create database (POST /api/v1/apps/:id/databases)
- [ ] Create storage bucket (POST /api/v1/apps/:id/storage)
- [ ] Verify plan limit enforcement for databases and buckets

### Task 7.6: Fix any failures

- [ ] For each failing test, identify root cause
- [ ] Fix the issue
- [ ] Re-run the failing test
- [ ] Repeat until all tests pass

### Task 7.7: Final commit and merge

- [ ] Squash or clean up commit history if needed
- [ ] Merge `refactor/backend-clean-architecture` → `main`
- [ ] Merge `main` → `staging`
- [ ] Push both branches
- [ ] Verify ArgoCD syncs successfully

---

## Appendix: Files Changed Summary

| Phase | Files Created | Files Modified | Files Deleted |
|-------|---------------|----------------|---------------|
| Phase 1 | `ports/infrastructure.go` | — | — |
| Phase 2 | — | — | `services/auth/*` |
| Phase 3 | `pkg/jwt/jwt.go` | `services/auth.go`, `services/billing.go`, `services/admin.go`, `services/customer.go`, `middleware/auth.go`, all adapters (compile checks) | — |
| Phase 4 | `services/deploy/*`, `services/provisioning/*`, `services/autoscale/*` | All files that imported old paths | `deploy/*`, `cluster/*`, `autoscale/*`, `temporal/*` |
| Phase 5 | — | `cmd/server/main.go` | — |

**Total estimated changes:** ~15-20 files modified, ~3 files created, ~20+ files moved/deleted.

---

## Rollback Plan

If anything goes wrong after deployment:

1. **Revert Helm values** to previous image tag (e.g., `0.4.3`)
2. **ArgoCD auto-syncs** the rollback
3. **Git revert** the merge commit on `staging` if needed

The refactoring is purely internal — no API contract changes, no database migrations, no behavioral changes. A rollback to the previous image is safe.
