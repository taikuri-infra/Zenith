# 10 — Backend Architecture (Go / Lich Clean Architecture)

> **Status:** Design Complete, Implementation Pending
> **Last Updated:** 2026-03-03
> **Author:** Babak + Claude (Backend Architecture Session)

---

## Table of Contents

1. [Overview](#overview)
2. [Standard Lich Go Template](#standard-lich-go-template)
3. [Dependency Rules](#dependency-rules)
4. [Layer Specifications](#layer-specifications)
5. [Request Flow Diagrams](#request-flow-diagrams)
6. [DI Composition Root — `cmd/server/main.go`](#di-composition-root--cmdservermain-go)
7. [Current API Service — Structure Audit](#current-api-service--structure-audit)
8. [Proposed API Service Structure](#proposed-api-service-structure)
9. [Keycloak Per-Tenant Integration](#keycloak-per-tenant-integration)
10. [Dead Code Removal — `services/auth/`](#dead-code-removal--servicesauth)
11. [Current Violations](#current-violations)
12. [Implementation Plan](#implementation-plan)

---

## Overview

Zenith's Go backend follows the **Lich Architecture** — a pragmatic clean/hexagonal architecture designed for Go microservices. The core principle is **dependency inversion**: business logic never imports infrastructure. All external systems (Kubernetes, Stripe, Keycloak, S3, PostgreSQL) are hidden behind interfaces defined in the `ports` package.

### Design Goals

- **Testable:** Every service can be unit-tested with in-memory adapters (no Docker, no network)
- **Swappable:** Replace Stripe with Paddle, PostgreSQL with CockroachDB, etc. — services don't change
- **Auditable:** Clear dependency graph; `go vet` + import analysis catches violations
- **Deployable:** Single binary per service, 12-factor config, graceful shutdown

### Go Services in the Monorepo

| Service | Path | Purpose |
|---------|------|---------|
| **zenith-api** | `services/api/` | Main API — auth, apps, billing, admin, deploy, provisioning |
| **zenith-operator** | `services/operator/` | Kubernetes operator for Zenith CRDs |
| ~~zenith-auth~~ | ~~`services/auth/`~~ | **DEAD CODE** — prototype OIDC provider, replaced by Keycloak |

---

## Standard Lich Go Template

Every Go service in Zenith follows this canonical layout:

```
services/<name>/
├── cmd/
│   └── server/
│       └── main.go              # Entrypoint: config, DI wiring, signal handling
├── internal/
│   ├── config/
│   │   └── config.go            # Env-based configuration (12-factor)
│   ├── entities/                # Pure domain models — ZERO external imports
│   │   ├── user.go
│   │   ├── app.go
│   │   └── plan.go
│   ├── ports/                   # Interfaces only — imports entities, nothing else
│   │   ├── repositories.go      # Data persistence interfaces
│   │   └── infrastructure.go    # External system interfaces (K8s, Stripe, S3, etc.)
│   ├── services/                # Business logic — imports entities, ports, dto
│   │   ├── auth.go              # User authentication + JWT
│   │   ├── billing.go           # Stripe checkout, subscriptions
│   │   ├── plan.go              # Plan limits, usage tracking
│   │   ├── admin.go             # Admin dashboard, cluster ops
│   │   ├── customer.go          # SaaS tenant management
│   │   ├── deploy/              # Build pipeline orchestration (git → Kaniko → K8s)
│   │   ├── provisioning/        # Cluster provisioner + Temporal workflows
│   │   └── autoscale/           # Hetzner node autoscaler
│   ├── dto/                     # Request/response shapes — imports entities only
│   │   ├── inputs.go
│   │   └── responses.go
│   ├── adapters/                # Interface implementations — imports entities, ports
│   │   ├── postgres/            # PostgreSQL repository implementations
│   │   ├── memory/              # In-memory implementations (dev/test)
│   │   ├── k8sclient/           # Kubernetes client (real + memory)
│   │   ├── stripeclient/        # Stripe payment adapter
│   │   ├── keycloakclient/      # Keycloak identity adapter
│   │   ├── s3client/            # Hetzner S3 adapter
│   │   ├── capiclient/          # CAPI cluster provisioning adapter
│   │   └── hetznerclient/       # Hetzner Cloud API adapter
│   ├── handlers/                # HTTP layer — imports services, dto
│   │   ├── health.go
│   │   ├── auth.go
│   │   ├── apps.go
│   │   └── admin.go
│   ├── middleware/              # HTTP middleware (auth, CORS, rate-limit)
│   └── telemetry/              # OpenTelemetry setup
├── pkg/
│   └── jwt/                     # Shared JWT utilities (extracted from middleware)
├── docs/
│   ├── embed.go                 # Swagger embed
│   └── handler.go               # Swagger route registration
└── go.mod
```

### Key Conventions

- **`cmd/`** — Only wiring. Create adapters, inject into services, inject services into handlers. No business logic.
- **`internal/`** — All code is internal to the service (Go enforces this — no external imports possible).
- **One file per entity** in `entities/`, one file per domain area in `services/`.
- **Adapters define their own interface** when they own it (e.g., `k8sclient.Client`, `stripeclient.StripeAPI`). These MUST be moved to `ports/infrastructure.go` during refactoring.
- **In-memory adapters** in `adapters/memory/` implement every repository port — used for local dev and unit tests.

### API Layer Convention — Why `handlers/` + `middleware/`

Go projects use several conventions for the HTTP layer:
- `internal/handler/` — singular, flat (Go standard project layout, most OSS)
- `internal/api/http/` — nested under `api/` (Lich Python template)
- `internal/transport/http/` — transport-based (go-kit, hexagonal purists)

**We keep `handlers/` + `middleware/`.** This is the most common Go convention and matches our existing codebase. The `api/http/` pattern adds nested folders for no benefit — our API is HTTP-only (no gRPC or CLI transports). The Lich Python template uses `api/http/` because Python projects often serve multiple transports.

### Port File Naming — `repositories.go` + `infrastructure.go`

The `ports/` package is split into two files, mapping to the two kinds of external dependencies:

| File | Purpose | Examples |
|------|---------|---------|
| `repositories.go` | Data persistence | `UserRepository`, `AppRepository`, `BillingRepository` |
| `infrastructure.go` | External systems | `KubernetesClient`, `PaymentGateway`, `ObjectStorage`, `IdentityProvider` |

This is clean and self-documenting. A developer immediately knows whether to look in `repositories.go` (DB/cache) or `infrastructure.go` (K8s, Stripe, S3, Keycloak, Hetzner).

---

## Dependency Rules

```
┌─────────────────────────────────────────────────────────────┐
│                         cmd/server/main.go                  │
│  (wiring only — creates adapters, injects into services)    │
│  CAN import: everything                                     │
└─────────┬───────────────────────────────────────────────────┘
          │ injects
          ▼
┌─────────────────────┐    ┌──────────────┐    ┌──────────┐
│  handlers/           │───▶│  services/   │───▶│  ports/  │
│  (HTTP layer)        │    │  (business)  │    │  (ifaces)│
│                      │    │              │    │          │
│  CAN import:         │    │  CAN import: │    │ CAN only │
│  services, dto       │    │  entities    │    │ import:  │
│                      │    │  ports       │    │ entities │
│  CANNOT import:      │    │  dto         │    │          │
│  adapters, entities  │    │              │    │ CANNOT:  │
│  directly            │    │  CANNOT:     │    │ anything │
│                      │    │  adapters    │    │ else     │
└──────────────────────┘    │  handlers    │    └────▲─────┘
                            │  config      │         │
                            └──────────────┘         │ implements
                                                     │
                            ┌────────────────────────┘
                            │
                    ┌───────┴──────┐
                    │  adapters/   │
                    │              │
                    │  CAN import: │
                    │  entities    │
                    │  ports       │
                    │  pkg (shared)│
                    │              │
                    │  CANNOT:     │
                    │  services    │
                    │  handlers    │
                    └──────────────┘

┌──────────────┐
│  entities/   │  ──▶  NOTHING. Zero imports from project packages.
│  (domain)    │       stdlib only (time, fmt, errors, etc.)
└──────────────┘

┌──────────────┐
│  dto/        │  ──▶  entities ONLY
│  (shapes)    │       (references entity types in responses)
└──────────────┘
```

### The Golden Rule

> **Services depend on ports (interfaces), never on adapters (implementations).**
> Adapters implement ports. `cmd/` wires adapters into services.

### Import Enforcement

These rules can be enforced with Go import analysis. A future `lich lint` command will check:

```
# FORBIDDEN patterns:
services/ importing adapters/*      → violation
entities/ importing anything        → violation
ports/ importing adapters/*         → violation
handlers/ importing adapters/*      → violation

# ALLOWED patterns:
services/ importing ports/          → OK
services/ importing entities/       → OK
services/ importing dto/            → OK
adapters/ importing ports/          → OK
adapters/ importing entities/       → OK
cmd/ importing everything           → OK (wiring layer)
```

---

## Layer Specifications

### Entities (`internal/entities/`)

Pure domain models. No framework imports, no database tags, no HTTP types.

```go
// GOOD — pure domain model
package entities

import "time"

type PlanTier string

const (
    PlanFree       PlanTier = "free"
    PlanPro        PlanTier = "pro"
    PlanTeam       PlanTier = "team"
    PlanEnterprise PlanTier = "enterprise"
)

type App struct {
    ID          string
    UserID      string
    Name        string
    Subdomain   string
    GitRepo     string
    Status      AppStatus
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Domain logic lives on entities
func (a *App) CanDeploy() bool {
    return a.Status == AppStatusRunning || a.Status == AppStatusStopped
}
```

```go
// BAD — leaking infrastructure
package entities

import "gorm.io/gorm"  // ❌ ORM in entity

type App struct {
    gorm.Model            // ❌ framework type
    ID string `gorm:"primaryKey"` // ❌ DB tags
}
```

**Rules:**
- One file per aggregate root
- Domain validation and invariants live here
- Enums (status types, plan tiers) defined here
- No constructors that need external deps

### Ports (`internal/ports/`)

Interfaces only. Split into two files:

**`repositories.go`** — Data persistence (already exists, well-structured):
```go
type UserRepository interface {
    Create(ctx context.Context, email, password, name string, role entities.Role) (*entities.User, error)
    GetByEmail(ctx context.Context, email string) (*StoredUser, error)
    GetByID(ctx context.Context, id string) (*StoredUser, error)
    // ...
}
```

**`infrastructure.go`** — External systems (NEEDS CREATION):
```go
package ports

import "context"

// KubernetesClient abstracts Kubernetes API operations.
type KubernetesClient interface {
    CreateNamespace(ctx context.Context, name string, labels map[string]string) error
    GetNamespace(ctx context.Context, name string) error
    DeleteNamespace(ctx context.Context, name string) error
    CreateSecret(ctx context.Context, namespace, name string, data map[string][]byte, labels map[string]string) error
    // ... (full interface extracted from k8sclient.Client)
}

// IdentityProvider abstracts identity management (Keycloak).
type IdentityProvider interface {
    CreateRealm(ctx context.Context, realmName, displayName string) error
    DeleteRealm(ctx context.Context, realmName string) error
    CreateClient(ctx context.Context, realmName, clientID, redirectURI string) (secret string, err error)
}

// PaymentGateway abstracts payment processing (Stripe).
type PaymentGateway interface {
    CreateCheckoutSession(ctx context.Context, params CheckoutParams) (*CheckoutResult, error)
    CreatePortalSession(ctx context.Context, customerID, returnURL string) (*PortalResult, error)
    CancelSubscription(ctx context.Context, subID string, atPeriodEnd bool) error
    GetSubscription(ctx context.Context, subID string) (*SubscriptionResult, error)
    VerifyWebhookSignature(payload []byte, signature string) error
}

// ObjectStorage abstracts S3-compatible storage.
type ObjectStorage interface {
    CreateBucket(ctx context.Context, bucketName string) error
    DeleteBucket(ctx context.Context, bucketName string) error
}

// ClusterProvisioner abstracts CAPI cluster operations.
type ClusterProvisioner interface {
    ListClusters(ctx context.Context) ([]entities.Cluster, error)
    GetCluster(ctx context.Context, name string) (*entities.Cluster, error)
    CreateCluster(ctx context.Context, input dto.CreateClusterInput) (*entities.Cluster, error)
    DeleteCluster(ctx context.Context, name string) error
    UpgradeCluster(ctx context.Context, name, version string) error
}

// CloudProvider abstracts cloud infrastructure (Hetzner).
type CloudProvider interface {
    CreateServer(ctx context.Context, name, serverType, location, userData string) (int64, string, error)
    DeleteServer(ctx context.Context, serverID int64) error
    ListServers(ctx context.Context, labelSelector string) ([]CloudServer, error)
}
```

**Rules:**
- Interfaces ONLY — no structs, no implementations
- Named after capabilities, not implementations (e.g., `IdentityProvider` not `KeycloakClient`)
- Imports `entities` only (and `context` from stdlib)
- Parameter/return types use entities or simple types — never adapter-specific types

### Services (`internal/services/`)

Business logic. One service per domain area. Depends only on ports (interfaces).

```go
// GOOD — depends on port interfaces
package services

type BillingService struct {
    payments    ports.PaymentGateway      // ✅ port interface
    billing     ports.BillingRepository   // ✅ port interface
    plans       ports.UserPlanRepository  // ✅ port interface
    storage     ports.ObjectStorage       // ✅ port interface
}

// BAD — current violation (depends on adapter directly)
type BillingService struct {
    stripe      stripeClient.StripeAPI    // ❌ adapter type
    s3          s3client.S3API            // ❌ adapter type
}
```

**Rules:**
- Constructor takes port interfaces, stored as struct fields
- Returns entities or dto types
- Raises domain errors (not HTTP errors, not adapter errors)
- No `*fiber.Ctx`, no `http.Request`, no framework types
- One file per service, named after the domain area

### Adapters (`internal/adapters/`)

Implement port interfaces. Each adapter is in its own sub-package.

```go
// adapters/stripeclient/client.go
package stripeclient

// Client implements ports.PaymentGateway using Stripe SDK.
type Client struct { ... }

func (c *Client) CreateCheckoutSession(ctx context.Context, params ports.CheckoutParams) (*ports.CheckoutResult, error) {
    // Stripe SDK calls here
}
```

**Rules:**
- Each adapter in its own package (`adapters/postgres/`, `adapters/k8sclient/`, etc.)
- Implements one or more port interfaces
- Contains adapter-specific types for internal use only
- Memory implementations in `adapters/memory/` for every port
- No business logic — pure translation between domain types and external APIs

### Handlers (`internal/handlers/`)

HTTP layer. Parses requests, calls services, returns responses.

```go
package handlers

type BillingHandler struct {
    svc *services.BillingService
}

func (h *BillingHandler) CreateCheckoutSession(c *fiber.Ctx) error {
    // 1. Parse & validate request
    // 2. Call service
    // 3. Map to response DTO
    // 4. Return HTTP response
}
```

**Rules:**
- No business logic — delegates everything to services
- Validates input (required fields, format)
- Maps between HTTP and domain types
- Sets HTTP status codes and error responses
- CAN import services and dto; CANNOT import adapters or entities directly

### Domain-Specific Service Modules

These packages encapsulate complex, multi-step orchestration. They live under `services/` because they contain business logic — but unlike flat service files, they're large enough to warrant their own sub-package:

| Current Location | Target Location | Purpose |
|-----------------|-----------------|---------|
| `internal/deploy/` | `internal/services/deploy/` | Git push → Kaniko build → K8s deploy pipeline |
| `internal/cluster/` + `internal/temporal/` | `internal/services/provisioning/` | CAPI cluster provisioner + Temporal workflows (merged) |
| `internal/autoscale/` | `internal/services/autoscale/` | Hetzner node autoscaler (metrics → scale decision → API call) |

**Why merge `cluster/` and `temporal/`?** Both handle customer provisioning — `cluster/` manages CAPI clusters for Team/Enterprise, and `temporal/` orchestrates the full provisioning workflow (Keycloak realm, DB, S3, K8s namespace, etc.). They belong together as `services/provisioning/`.

**Dependency rule:** These modules currently import adapters directly. After refactoring, they should depend on port interfaces just like flat service files. The `cmd/server/main.go` wiring layer injects concrete adapters.

---

## Request Flow Diagrams

Three representative flows showing how a request travels through every layer.

### Flow 1: `POST /api/v1/auth/register` — User Registration

```
Client
  │
  │  POST /api/v1/auth/register
  │  Body: {"email":"...", "password":"...", "name":"..."}
  │
  ▼
┌──────────────────────────────────────────────────────────┐
│  Fiber Router  (cmd/server/main.go line ~350)            │
│  Route: api.Post("/auth/register", authHandler.Register) │
│  No JWT middleware — public endpoint                     │
└──────────────┬───────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────┐
│  handlers/auth.go → AuthHandler.Register()               │
│                                                          │
│  1. Parse JSON body into registerRequest struct           │
│  2. Validate: email, password, name required              │
│  3. Validate: password >= 8 chars                         │
│  4. Call: h.svc.Register(ctx, email, password, name)      │
│  5. Map TokenPair → tokenResponse JSON                    │
│  6. Return 200 or 409 (conflict)                          │
└──────────────┬───────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────┐
│  services/auth.go → AuthService.Register()               │
│                                                          │
│  1. Determine role: if user count == 0 → RoleOwner       │
│     else → RoleDeveloper                                  │
│  2. Call: s.users.Create(ctx, email, password, name, role)│
│  3. Call: s.issueTokens(user) → generates JWT pair        │
│     └─ middleware.GenerateToken(secret, user, 1h)  ← ⚠️  │
│     └─ middleware.GenerateToken(secret, user, 7d)  ← ⚠️  │
│  4. Return TokenPair{access, refresh, expiresIn}          │
│                                                          │
│  ⚠️ Violation: imports middleware for JWT generation       │
└──────────────┬───────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────┐
│  ports/repositories.go → UserRepository.Create()         │
│  (interface — actual impl is postgres or memory)          │
│                                                          │
│  adapters/postgres/postgres_user.go:                      │
│    1. Hash password with bcrypt                           │
│    2. INSERT INTO users (id, email, password_hash, ...)   │
│    3. Return *entities.User                               │
│                                                          │
│  adapters/memory/memory_user.go:                          │
│    1. Hash password, store in map                          │
│    2. Return *entities.User                               │
└──────────────────────────────────────────────────────────┘
```

**Key takeaway:** Clean flow except for the `middleware.GenerateToken` violation in the service layer.

---

### Flow 2: `POST /api/v1/apps` — App Creation with Plan Limit Check

```
Client
  │
  │  POST /api/v1/apps
  │  Headers: Authorization: Bearer <JWT>
  │  Body: {"name":"my-app", "repo_url":"https://github.com/..."}
  │
  ▼
┌──────────────────────────────────────────────────────────┐
│  Fiber Router  (cmd/server/main.go line ~390)            │
│                                                          │
│  Route chain:                                             │
│    1. middleware.RequireAuth(jwtSecret)  ← JWT validation  │
│    2. handlers.CheckLimit(planRepo, "apps", countFn)      │
│    3. appHandlerV2.Create                                  │
└──────────────┬───────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────┐
│  middleware/auth.go → RequireAuth()                       │
│                                                          │
│  1. Extract "Authorization: Bearer <token>" header        │
│  2. ParseToken(secret, token) → Claims{sub, email, role} │
│  3. Store in c.Locals: user_id, email, name, role         │
│  4. Call c.Next() → proceed to next middleware             │
└──────────────┬───────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────┐
│  handlers/plan.go → CheckLimit() middleware factory       │
│                                                          │
│  1. Get userID from c.Locals("user_id")                   │
│  2. planRepo.GetUserPlan(ctx, userID) → UserPlan{Limits}  │
│  3. countFn(c, userID) → appRepo.CountAppsByUser()        │
│  4. If count >= plan.Limits.MaxApps → 403 "plan limit     │
│     reached: apps. Upgrade your plan for more."           │
│  5. Else → c.Next()                                       │
└──────────────┬───────────────────────────────────────────┘
               │  (limit not exceeded)
               ▼
┌──────────────────────────────────────────────────────────┐
│  handlers/apps_v2.go → AppHandlerV2.Create()             │
│                                                          │
│  1. Get userID from c.Locals("user_id")                   │
│  2. Parse JSON body → CreateAppV2Request{name, repo_url}  │
│  3. Validate: name required, repo_url required            │
│  4. Call: appRepo.CreateApp(ctx, &dto.CreateAppInput{...})│
│  5. Map *entities.App → AppV2Response JSON                │
│  6. Return 201 Created                                    │
└──────────────┬───────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────┐
│  ports/repositories.go → AppRepository.CreateApp()       │
│  (interface — postgres or memory impl)                    │
│                                                          │
│  1. Generate UUID, sanitize subdomain from app name       │
│  2. Check uniqueness (name + user, subdomain)             │
│  3. INSERT INTO apps (id, user_id, name, repo_url, ...)   │
│  4. Return *entities.App with status=pending              │
└──────────────────────────────────────────────────────────┘
```

**Key takeaway:** Plan limit enforcement is a middleware (`CheckLimit`), not embedded in the service. This keeps the handler and service layers clean — the middleware reads the plan limits from `UserPlanRepository` and short-circuits with 403 before the handler runs.

---

### Flow 3: `POST /api/v1/billing/checkout` — Stripe Checkout Session

```
Client
  │
  │  POST /api/v1/billing/checkout
  │  Headers: Authorization: Bearer <JWT>
  │  Body: {"tier":"pro"}
  │
  ▼
┌──────────────────────────────────────────────────────────┐
│  Fiber Router  (cmd/server/main.go line ~483)            │
│                                                          │
│  Route: protected.Post("/billing/checkout",               │
│           billingHandler.CreateCheckoutSession)            │
│  Middleware: RequireAuth (JWT required)                    │
└──────────────┬───────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────┐
│  handlers/billing.go → BillingHandler.CreateCheckoutSession() │
│                                                          │
│  1. Get userID, email from c.Locals (set by JWT middleware)│
│  2. Parse JSON body → dto.CreateCheckoutInput{tier}       │
│  3. Call: h.svc.CreateCheckoutSession(ctx, userID, email,  │
│           input.Tier)                                      │
│  4. Return JSON: {session_id, checkout_url}               │
└──────────────┬───────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────┐
│  services/billing.go → BillingService.CreateCheckoutSession() │
│                                                          │
│  1. Check: s.stripe != nil (Stripe enabled?)              │
│  2. PriceForTier(tier) → priceID                          │
│     Pro = 2900 cents (€29), Team = 19900 cents (€199)     │
│  3. planRepo.GetUserPlan(ctx, userID)                      │
│     → Reject if already on requested tier                  │
│  4. billingRepo.GetStripeCustomerID(ctx, userID)           │
│     → Existing Stripe customer or empty                    │
│  5. s.stripe.CreateCheckoutSession(ctx, CheckoutParams{    │  ← ⚠️ adapter type
│       CustomerID, PriceID, SuccessURL, CancelURL,          │
│       UserEmail, Metadata{user_id, tier}                   │
│     })                                                     │
│  6. Return dto.CheckoutResponse{CheckoutURL, SessionID}    │
│                                                          │
│  ⚠️ Violation: s.stripe is stripeClient.StripeAPI          │
│     (adapter interface, not port interface)                 │
└──────┬────────────────────────┬──────────────────────────┘
       │                        │
       ▼                        ▼
┌────────────────────┐  ┌──────────────────────────────────┐
│  ports/repos →     │  │  adapters/stripeclient/ →        │
│  BillingRepository │  │  StripeAPI.CreateCheckoutSession()│
│  .GetStripeCustomer│  │                                   │
│  ID()              │  │  1. Build stripe.CheckoutSession   │
│                    │  │     Params{mode=subscription}      │
│  UserPlanRepository│  │  2. stripe SDK: session.New(p)     │
│  .GetUserPlan()    │  │  3. Return {SessionID, URL}        │
└────────────────────┘  └──────────────────────────────────┘
                                     │
                                     ▼
                              Stripe API (external)
                                     │
                                     ▼
                        Client redirects to checkout_url
                        → User pays → Stripe sends webhook
                        → POST /api/v1/webhooks/stripe
                        → BillingService.HandleWebhook()
                        → planRepo.SetUserPlan(ctx, userID, "pro")
```

**Key takeaway:** The billing flow crosses two boundaries — repository ports (clean) and Stripe adapter (violation). After refactoring, `s.stripe` becomes `ports.PaymentGateway` and the `CheckoutParams` type moves to `ports/`.

---

### Layer-by-Layer Dependency Map

Visual overview of what each layer can and cannot import:

```
                    ┌─────────────────────────┐
                    │    cmd/server/main.go    │
                    │    (composition root)    │
                    │    CAN IMPORT: *         │
                    └──┬──┬──┬──┬──┬──┬───────┘
                       │  │  │  │  │  │
        ┌──────────────┘  │  │  │  │  └──────────────────┐
        │     ┌───────────┘  │  │  └──────────┐          │
        ▼     ▼              ▼  ▼             ▼          ▼
   ┌────────┐ ┌──────────┐ ┌───────┐  ┌──────────┐ ┌─────────┐
   │handlers│ │middleware │ │config │  │ adapters/ │ │telemetry│
   │        │ │          │ │       │  │ postgres/ │ │         │
   │imports:│ │imports:  │ │imports│  │ k8sclient/│ │imports: │
   │services│ │entities  │ │stdlib │  │ stripe../ │ │stdlib   │
   │dto     │ │pkg/jwt   │ │only   │  │ s3../etc  │ │         │
   │        │ │          │ │       │  │           │ │         │
   │CANNOT: │ │CANNOT:   │ └───────┘  │imports:   │ └─────────┘
   │adapters│ │services  │            │entities   │
   │entities│ │adapters  │            │ports      │
   │(direct)│ │handlers  │            │           │
   └───┬────┘ └──────────┘            │CANNOT:    │
       │                              │services   │
       ▼                              │handlers   │
   ┌────────────────┐                 └─────┬─────┘
   │   services/    │                       │
   │                │                       │ implements
   │ imports:       │                       │
   │ entities       │                       ▼
   │ ports          │◄────────────── ┌──────────┐
   │ dto            │  depends on    │  ports/  │
   │                │  interfaces    │          │
   │ CANNOT:        │                │ imports: │
   │ adapters       │                │ entities │
   │ handlers       │                │ ONLY     │
   │ config         │                └────┬─────┘
   └────────────────┘                     │
                                          │ references
                                          ▼
                                   ┌──────────┐
                                   │entities/ │
                                   │          │
                                   │ imports: │
                                   │ NOTHING  │
                                   │ (stdlib  │
                                   │  only)   │
                                   └──────────┘
```

---

## DI Composition Root — `cmd/server/main.go`

The `main.go` file (~746 lines) is the **single place** where concrete implementations are chosen and wired together. No business logic here — only configuration, adapter creation, and dependency injection.

### Wiring Sequence

```
main()
  │
  ├── 1. config.Load()                      → Load env vars (12-factor)
  │
  ├── 2. Choose adapter implementations:
  │      ├── if DATABASE_URL set:
  │      │     postgres.New(dsn)            → Real PostgreSQL repos
  │      │     postgres.Migrate(pool)       → Run SQL migrations
  │      │   else:
  │      │     memory.New*()                → In-memory repos (dev mode)
  │      │
  │      ├── if K8S_MODE == "real":
  │      │     k8sclient.NewRealClient()    → Real Kubernetes client
  │      │   else:
  │      │     k8sclient.NewMemoryClient()  → In-memory K8s mock
  │      │
  │      ├── if STRIPE_ENABLED:
  │      │     stripeclient.NewClient()     → Real Stripe API
  │      │
  │      ├── if KEYCLOAK_URL set:
  │      │     keycloakclient.NewClient()   → Real Keycloak admin
  │      │
  │      ├── if S3_ENDPOINT set:
  │      │     s3client.NewClient()         → Real Hetzner S3
  │      │
  │      └── if HETZNER_TOKEN set:
  │            hetznerclient.NewClient()    → Real Hetzner Cloud API
  │
  ├── 3. Create services (inject port interfaces):
  │      ├── AuthService(userRepo, jwtSecret)
  │      ├── PlanService(planRepo, appRepo, dbRepo, storageRepo, authRepo)
  │      ├── BillingService(stripeAPI, billingRepo, planRepo, ...)
  │      ├── AdminService(k8sClient, capiClient, adminRepo)
  │      └── CustomerService(customerRepo, adminRepo, provisioner)
  │
  ├── 4. Create handlers (inject services):
  │      ├── AuthHandler(authSvc)
  │      ├── BillingHandler(billingSvc)
  │      ├── PlanHandler(planSvc)
  │      ├── AppHandlerV2(appRepo, baseDomain, deployer)
  │      └── ... (~15 more handlers)
  │
  ├── 5. Register routes:
  │      ├── Public:  /auth/*, /webhooks/github, /health
  │      ├── Protected: /apps/*, /billing/*, /plan, /domains/*
  │      └── Admin:   /admin/* (RequireRole(owner))
  │
  ├── 6. Start background goroutines:
  │      ├── cluster.Provisioner.StartSync(60s)
  │      ├── autoscale.Autoscaler.Start(60s)
  │      └── temporal.Worker.Start()
  │
  └── 7. app.Listen(":port") + signal.Notify(SIGTERM)
```

### Key Pattern: Feature Flags via Config

Adapters are conditionally created based on environment variables. When a feature is disabled, the corresponding adapter is `nil` and services handle the nil gracefully:

```go
// In main.go:
var stripeAPI stripeClient.StripeAPI  // nil by default
if cfg.StripeBillingEnabled && cfg.StripeSecretKey != "" {
    stripeAPI = stripeClient.NewClient(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
    planSvc.SetStripeEnabled(true)
}

// In BillingService:
func (s *BillingService) CreateCheckoutSession(...) {
    if s.stripe == nil {
        return nil, fmt.Errorf("Stripe billing is not enabled")
    }
    // ... proceed with Stripe
}
```

This allows the same binary to run in different modes without code changes — just different env vars.

---

## Current API Service — Structure Audit

### Current Directory Layout

```
services/api/
├── cmd/server/main.go              ✅ Wiring layer
├── docs/                            ✅ Swagger
├── internal/
│   ├── config/config.go             ✅ 12-factor config
│   ├── entities/                    ✅ 25 entity files, pure domain
│   │   ├── admin.go, app.go, billing.go, customer.go, ...
│   │   └── plan.go, user.go, webhook.go
│   ├── ports/
│   │   └── repositories.go          ✅ 20 repository interfaces (well-structured)
│   │   └── ❌ infrastructure.go      MISSING — infra ports not extracted
│   ├── services/
│   │   ├── admin.go                  ❌ Imports capiclient, k8sclient directly
│   │   ├── auth.go                   ⚠️ Imports middleware (for JWT generation)
│   │   ├── billing.go                ❌ Imports stripeclient, s3client directly
│   │   ├── customer.go               ⚠️ Imports cluster package directly
│   │   └── plan.go                   ✅ Clean — only imports ports
│   ├── dto/
│   │   ├── inputs.go                 ✅ Clean
│   │   └── responses.go              ✅ Clean
│   ├── adapters/
│   │   ├── capiclient/               ✅ Interface defined (but in adapter pkg)
│   │   ├── hetznerclient/            ✅ Interface defined (but in adapter pkg)
│   │   ├── k8sclient/                ✅ Interface defined (but in adapter pkg)
│   │   ├── keycloakclient/           ✅ Interface defined (but in adapter pkg)
│   │   ├── memory/                   ✅ Full in-memory implementations
│   │   ├── postgres/                 ✅ Real DB implementations
│   │   ├── s3client/                 ✅ Interface defined (but in adapter pkg)
│   │   └── stripeclient/             ✅ Interface defined (but in adapter pkg)
│   ├── handlers/                     ✅ ~30 handler files
│   ├── middleware/                   ✅ Auth middleware, context helpers
│   ├── deploy/                      ⚠️ Should move to services/deploy/
│   ├── cluster/                     ⚠️ Should merge into services/provisioning/
│   ├── autoscale/                   ⚠️ Should move to services/autoscale/
│   ├── temporal/                    ⚠️ Should merge into services/provisioning/
│   └── telemetry/                   ✅ OTel setup
```

### What's Good

1. **Entities are clean** — 25 files, no external imports, pure domain models
2. **Repository ports are comprehensive** — 20 interfaces covering all persistence
3. **In-memory adapters exist for everything** — enables testing without Docker
4. **Adapters each define their own interface** — just in the wrong place
5. **DTO layer is clean** — proper request/response separation
6. **Config is 12-factor** — env-based, no hardcoded values
7. **Handlers delegate to services** — no business logic in HTTP layer

### What Needs Fixing

See [Current Violations](#current-violations) section below.

---

## Proposed API Service Structure

After refactoring, three categories of changes:

### 1. New Port Definitions

```diff
 services/api/internal/ports/
     repositories.go       # (exists — 23 repository interfaces, keep as-is)
+    infrastructure.go     # NEW — interfaces for K8s, Stripe, S3, Keycloak, CAPI, Hetzner
```

### 2. Service Layer Cleanup

```diff
 services/api/internal/services/
-    admin.go              # imports capiclient, k8sclient → REFACTOR
-    billing.go            # imports stripeclient, s3client → REFACTOR
-    customer.go           # imports cluster, temporal → REFACTOR
-    auth.go               # imports middleware → MINOR FIX
+    admin.go              # imports ports.KubernetesClient, ports.ClusterProvisioner
+    billing.go            # imports ports.PaymentGateway, ports.ObjectStorage
+    customer.go           # imports ports.ClusterProvisioner, ports.ProvisioningWorkflow
+    auth.go               # JWT via pkg/jwt (extracted from middleware)
     plan.go               # (already clean — only imports ports)
```

### 3. Domain Module Consolidation

```diff
 services/api/internal/
-    deploy/               # standalone package → MOVE
-    cluster/              # standalone package → MERGE
-    autoscale/            # standalone package → MOVE
-    temporal/             # standalone package → MERGE
+    services/deploy/      # build pipeline (git → Kaniko → K8s)
+    services/provisioning/# cluster provisioner + Temporal workflows (merged)
+    services/autoscale/   # Hetzner node autoscaler
```

### 4. JWT Extraction

```diff
+services/api/pkg/jwt/
+    jwt.go                # GenerateToken(), ParseToken() — extracted from middleware
```

The adapters themselves stay unchanged — they just need to formally satisfy the port interfaces defined in `ports/infrastructure.go`.

---

## Keycloak Per-Tenant Integration

### Architecture

Zenith uses a **single shared Keycloak instance** with **one realm per customer**. This provides identity isolation without the operational cost of per-customer Keycloak deployments.

```
Keycloak Instance (zenith-staging namespace)
├── master realm              — Keycloak admin operations
├── zenith-platform realm     — Platform admin/developer users
├── customer-abc realm        — Customer ABC's end-users
├── customer-def realm        — Customer DEF's end-users
└── customer-xyz realm        — Customer XYZ's end-users
```

### Per-Realm Configuration

When a customer is provisioned (via Temporal workflow), the API creates:

| Resource | Details |
|----------|---------|
| **Realm** | Name: `<customer-slug>`, display name from customer profile |
| **OIDC Client** | `zenith-app` client in the realm, confidential, redirect to `<customer>.freezenith.com/*` |
| **Client Secret** | Stored in K8s Secret `keycloak-client-<customer>` in customer namespace |
| **Roles** | `user`, `admin` realm roles (customer's app roles, not Zenith platform roles) |
| **User Limits** | Enforced by plan tier (see below) |

### User Limits by Plan Tier

| Tier | Max Users Per Realm | Max Realms | Notes |
|------|--------------------:|:----------:|-------|
| **Free** | 100 | 1 | Single app, built-in auth only |
| **Pro** | 5,000 | 1 | Keycloak realm + custom branding |
| **Team** | 50,000 | 1 | Dedicated Keycloak (own CNPG cluster) |
| **Enterprise** | Unlimited | Multiple | Dedicated Keycloak + custom federation |

### Enforcement Strategy

User limits are enforced at the **API level**, not in Keycloak itself (Keycloak doesn't have built-in user count limits per realm):

```
POST /api/v1/apps/:appId/auth/signup
    │
    ▼
AuthHandler.Signup()
    │
    ▼
Check plan tier → get max_users limit
    │
    ▼
Count existing users in Keycloak realm (via Admin API)
    │
    ▼
If count >= limit → 403 "User limit reached for your plan"
If count < limit  → Create user in Keycloak realm → 201
```

### Keycloak Port Interface

```go
// ports/infrastructure.go

// IdentityProvider abstracts identity management (Keycloak).
type IdentityProvider interface {
    // Realm management
    CreateRealm(ctx context.Context, realmName, displayName string) error
    DeleteRealm(ctx context.Context, realmName string) error

    // Client management
    CreateClient(ctx context.Context, realmName, clientID, redirectURI string) (secret string, err error)
    DeleteClient(ctx context.Context, realmName, clientID string) error

    // User management (per-realm)
    CreateUser(ctx context.Context, realmName, email, password, firstName, lastName string) (userID string, err error)
    DeleteUser(ctx context.Context, realmName, userID string) error
    CountUsers(ctx context.Context, realmName string) (int, error)
    ListUsers(ctx context.Context, realmName string, offset, limit int) ([]RealmUser, error)

    // Realm configuration
    SetRealmUserLimit(ctx context.Context, realmName string, maxUsers int) error
    GetRealmStats(ctx context.Context, realmName string) (*RealmStats, error)
}

type RealmUser struct {
    ID        string
    Email     string
    FirstName string
    LastName  string
    Enabled   bool
    CreatedAt time.Time
}

type RealmStats struct {
    UserCount    int
    ClientCount  int
    SessionCount int
}
```

### JWT Flow with Keycloak

```
1. User logs in via customer frontend
2. Frontend redirects to Keycloak login page
   URL: https://auth.freezenith.com/realms/<customer>/protocol/openid-connect/auth
3. User authenticates with Keycloak
4. Keycloak issues JWT with realm-scoped claims
5. Frontend receives token, sends to backend API
6. APISIX validates JWT using Keycloak JWKS endpoint:
   https://auth.freezenith.com/realms/<customer>/protocol/openid-connect/certs
7. Backend receives verified request (trusts APISIX headers)
```

### Keycloak CNPG Strategy

| Tier | Keycloak Database |
|------|------------------|
| Free + Pro | Shared CNPG cluster (`keycloak-pg` in `keycloak` namespace) |
| Team | Dedicated CNPG cluster in customer's VM cluster |
| Enterprise | Dedicated CNPG cluster in customer's VM cluster |

---

## Dead Code Removal — `services/auth/`

The `services/auth/` directory is a **dead prototype** — an attempt at building a custom OIDC provider before the decision to use Keycloak. It should be deleted entirely.

### What was in `services/auth/` (now deleted)

- Go module with custom OIDC endpoints
- `internal/kong/integration.go` — Kong gateway integration (Kong was replaced by APISIX — see [13-apisix-gateway.md](./13-apisix-gateway.md))
- No references from any live code, Helm charts, or CI pipelines
- Was not deployed anywhere

**Status:** This directory has been deleted. The `GatewayRoute` CRD and its controller in `services/operator/` have also been removed as part of the Kong → APISIX migration.

---

## Current Violations

### Violation 1: `services/admin.go` imports adapter packages

**File:** `services/api/internal/services/admin.go`
**Imports:**
```go
import (
    "github.com/dotechhq/zenith/services/api/internal/adapters/capiclient"  // ❌
    "github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"   // ❌
)
```

**Problem:** `AdminService` directly depends on `capiclient.Client` and `k8sclient.Client` (adapter types), violating the rule that services only depend on ports.

**Fix:** Extract `ports.ClusterProvisioner` and `ports.KubernetesClient` interfaces. `AdminService` constructor takes these interfaces:

```go
// Before (violation)
type AdminService struct {
    capiClient *capiclient.Client
    k8sClient  k8sclient.Client
    store      ports.AdminRepository
}

// After (clean)
type AdminService struct {
    clusters ports.ClusterProvisioner
    k8s      ports.KubernetesClient
    store    ports.AdminRepository
}
```

### Violation 2: `services/billing.go` imports adapter packages

**File:** `services/api/internal/services/billing.go`
**Imports:**
```go
import (
    "github.com/dotechhq/zenith/services/api/internal/adapters/s3client"           // ❌
    stripeClient "github.com/dotechhq/zenith/services/api/internal/adapters/stripeclient"  // ❌
)
```

**Problem:** `BillingService` depends on `stripeclient.StripeAPI` and `s3client.S3API` — types defined in adapter packages.

**Fix:** Move the interfaces to `ports/infrastructure.go` as `ports.PaymentGateway` and `ports.ObjectStorage`. The adapter packages keep their implementations but the interface definitions move to ports.

```go
// Before (violation)
type BillingService struct {
    stripe stripeClient.StripeAPI   // ❌ adapter type
    s3     s3client.S3API           // ❌ adapter type
}

// After (clean)
type BillingService struct {
    payments ports.PaymentGateway   // ✅ port interface
    storage  ports.ObjectStorage    // ✅ port interface
}
```

### Violation 3: `services/customer.go` imports domain module directly

**File:** `services/api/internal/services/customer.go`
**Imports:**
```go
import (
    "github.com/dotechhq/zenith/services/api/internal/cluster"    // ⚠️ domain module
    zenithTemporal "github.com/dotechhq/zenith/services/api/internal/temporal"  // ⚠️
)
```

**Problem:** `CustomerService` depends on `cluster.Provisioner` (concrete type) and Temporal workflow types.

**Fix:** Extract a `ports.ClusterProvisioner` interface. For Temporal, define a `ports.ProvisioningWorkflow` interface:

```go
// ports/infrastructure.go
type ProvisioningWorkflow interface {
    StartProvision(ctx context.Context, input ProvisionInput) error
    StartDeprovision(ctx context.Context, input DeprovisionInput) error
}
```

### Violation 4: `services/auth.go` imports middleware

**File:** `services/api/internal/services/auth.go`
**Imports:**
```go
import (
    "github.com/dotechhq/zenith/services/api/internal/middleware"  // ⚠️
)
```

**Problem:** `AuthService` calls `middleware.GenerateToken` and `middleware.ParseToken` — coupling service to HTTP middleware.

**Fix:** Extract JWT operations to a `pkg/jwt/` package or define a `ports.TokenGenerator` interface:

```go
// Option A: Move JWT to pkg/jwt
import "github.com/dotechhq/zenith/services/api/pkg/jwt"

// Option B: Port interface
type TokenGenerator interface {
    GenerateToken(user *entities.User, expiry time.Duration) (string, error)
    ParseToken(token string) (*TokenClaims, error)
}
```

### Violation 5: Adapter interfaces defined in adapter packages

**Files:** `k8sclient/client.go`, `stripeclient/client.go`, `s3client/client.go`, `keycloakclient/client.go`

**Problem:** Each adapter defines its own interface in its own package. This is backwards — consumers (services) should define the interfaces they need (in `ports/`), and adapters implement them.

**Fix:** Move interfaces to `ports/infrastructure.go`. Adapter packages keep implementations only. This is the single most impactful change — it breaks all the coupling.

### Violation Summary

| Service File | Violation | Severity | Fix Effort |
|-------------|-----------|----------|------------|
| `admin.go` | Imports `capiclient`, `k8sclient` | HIGH | Create `ports.ClusterProvisioner`, `ports.KubernetesClient` |
| `billing.go` | Imports `stripeclient`, `s3client` | HIGH | Move interfaces to `ports.PaymentGateway`, `ports.ObjectStorage` |
| `customer.go` | Imports `cluster`, `temporal` | MEDIUM | Create `ports.ClusterProvisioner`, `ports.ProvisioningWorkflow` |
| `auth.go` | Imports `middleware` | LOW | Extract JWT to `pkg/jwt` |
| All adapters | Interface in wrong package | HIGH | Move all interfaces to `ports/infrastructure.go` |

---

## Implementation Plan

The detailed, step-by-step implementation plan with checkboxes, file paths, code snippets, and validation commands is in a separate document:

**[BACKEND-REFACTOR.md](./BACKEND-REFACTOR.md)** — Backend Refactoring Implementation Plan

### Phase Summary

| Phase | Goal | Risk | Key Files |
|-------|------|------|-----------|
| **Pre-flight** | Verify current state compiles and tests pass | None | `go build`, `go test` |
| **Phase 1** | Create `ports/infrastructure.go` (additive, non-breaking) | None | `ports/infrastructure.go` |
| **Phase 2** | Delete `services/auth/` dead code | None | `rm -rf services/auth/` |
| **Phase 3** | Refactor services to use port interfaces | Medium | `admin.go`, `billing.go`, `customer.go`, `auth.go` |
| **Phase 4** | Move domain modules under `services/` | Medium | `deploy/`, `cluster/`, `autoscale/`, `temporal/` |
| **Phase 5** | Update `cmd/server/main.go` wiring | Low | `main.go` |
| **Phase 6** | Build, push, deploy via ArgoCD | Low | Docker, Helm, ArgoCD |
| **Phase 7** | E2E tests — run, fix, repeat until green | Medium | Full stack |

---

## Appendix: Quick Reference

### Creating a New Feature (Checklist)

1. Define entity in `entities/<name>.go`
2. Define port in `ports/repositories.go` (or `infrastructure.go`)
3. Implement adapter in `adapters/<provider>/<name>.go`
4. Implement memory adapter in `adapters/memory/memory_<name>.go`
5. Write service in `services/<domain>.go` (depends on ports only)
6. Write handler in `handlers/<domain>.go` (depends on service)
7. Wire in `cmd/server/main.go`
8. Write tests using memory adapters

### Testing Strategy

| Layer | Test Type | Dependencies |
|-------|-----------|-------------|
| `entities/` | Unit test | None (pure logic) |
| `services/` | Unit test | Memory adapters (injected via ports) |
| `handlers/` | HTTP test | Real services + memory adapters |
| `adapters/postgres/` | Integration test | Real PostgreSQL (Docker) |
| `adapters/k8sclient/` | Integration test | Real K8s (kind/k3d) |
| Full stack | E2E test | Real everything |
