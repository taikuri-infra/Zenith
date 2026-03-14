# 03 — Backend Complete Guide (Go / Hexagonal Architecture)

> **Read time:** 2-3 hours
> **Prerequisite:** [02 — Architecture Deep Dive](./02-architecture.md)
> **Next:** [04 — Frontend Complete Guide](./04-frontend-guide.md)

---

## Overview

The Zenith backend is a single Go binary (`zenith-api`) using the **Fiber** HTTP framework and **hexagonal architecture** (also called ports & adapters). The core principle: **business logic never imports infrastructure**.

```
services/api/
├── cmd/server/main.go          ← Entry point: DI wiring, 1500 lines
└── internal/
    ├── config/config.go        ← 65+ env vars
    ├── entities/               ← 37 files — pure domain models (ZERO imports)
    ├── ports/                  ← 2 files — 46 interfaces
    ├── services/               ← 24 files — business logic
    ├── handlers/               ← 83 files — 376 HTTP endpoints
    ├── adapters/               ← 16 packages — external system implementations
    │   ├── postgres/           ← 34 files + 37 migrations
    │   ├── memory/             ← 45 in-memory stubs (dev/test)
    │   ├── k8sclient/          ← Kubernetes (real + memory)
    │   ├── stripeclient/       ← Stripe payments
    │   ├── keycloakclient/     ← Keycloak identity
    │   ├── s3client/           ← Hetzner S3
    │   ├── promclient/         ← Prometheus queries
    │   ├── lokiclient/         ← Loki log queries
    │   ├── natsclient/         ← NATS event bus
    │   ├── resendclient/       ← Email delivery
    │   ├── harborclient/       ← Harbor registry
    │   ├── hetznerclient/      ← Hetzner Cloud API
    │   ├── capiclient/         ← CAPI cluster provisioning
    │   └── redisclient/        ← Redis (rate-limit, token blacklist)
    ├── dto/                    ← Request/response shapes
    ├── middleware/              ← 10 files (auth, CORS, security headers)
    └── telemetry/              ← OpenTelemetry setup
```

---

## The Golden Rule: Dependency Inversion

```
                        ┌──────────────┐
                        │   entities/   │  ← Knows NOTHING about the outside world
                        │  Pure Go      │     No imports from other packages
                        │  structs      │     Can be tested with zero setup
                        └──────┬───────┘
                               │ imports
                        ┌──────▼───────┐
                        │    ports/     │  ← Defines INTERFACES only
                        │  Contracts    │     "I need a UserRepository that can..."
                        │  No impl     │     Imports only: entities
                        └──────┬───────┘
                               │ implements
              ┌────────────────┼────────────────┐
              │                │                 │
       ┌──────▼──────┐  ┌─────▼──────┐  ┌──────▼──────┐
       │  postgres/   │  │  memory/   │  │ k8sclient/  │
       │  Real DB     │  │  In-memory │  │ Real K8s    │
       │  SQL queries │  │  Maps/slices│ │  client-go  │
       └─────────────┘  └────────────┘  └─────────────┘
                               │ used by
                        ┌──────▼───────┐
                        │  services/   │  ← Business logic
                        │  Imports:    │     Imports: entities, ports, dto
                        │  ports only  │     NEVER imports adapters directly
                        └──────┬───────┘
                               │ used by
                        ┌──────▼───────┐
                        │  handlers/   │  ← HTTP layer
                        │  Imports:    │     Imports: services, dto
                        │  services    │     Parses requests, calls services, returns JSON
                        └──────┬───────┘
                               │ wired by
                        ┌──────▼───────┐
                        │   main.go    │  ← Composition root
                        │  DI wiring   │     Creates all instances, connects everything
                        └──────────────┘
```

**Why this matters:**
- To test a service, you inject memory adapters — no Docker, no database needed
- To swap PostgreSQL for CockroachDB, you write a new adapter — services don't change
- To add a new feature, you follow the same pattern every time

---

## Layer 1: Entities (37 Files)

Entities are **pure Go structs** with zero external imports. They define the domain model.

### Key Entities

| File | Main Struct | Purpose |
|------|------------|---------|
| `user.go` | `User`, `TeamMember`, `APIKey` | Platform users, team, API keys |
| `app.go` | `App` | Container application (name, image, replicas, port, status) |
| `deployment.go` | `Deployment`, `Release`, `EnvVar`, `Secret` | Deploy history, env vars, encrypted secrets |
| `database.go` | `UserDatabase` | CNPG provisioned database (engine, host, creds) |
| `storage.go` | `UserBucket` | S3 bucket (name, access, size, endpoint) |
| `gateway.go` | `Gateway`, `GatewayRoute`, `GatewayGroup`, `GatewayCustomDomain` | APISIX gateway config |
| `authpool.go` | `AuthPool` | Keycloak-backed managed auth realm |
| `plan.go` | `PlanTier`, `PlanLimits`, `UserPlan` | Plan tiers + resource limits |
| `billing.go` | `Subscription`, `Invoice` | Stripe billing records |
| `customer.go` | `Customer`, `CustomerStats` | SaaS customer management |
| `admin.go` | `DashboardStats`, `Cluster`, `Tenant`, `Module` | Admin dashboard entities |
| `admin_v2.go` | `RevenueStats`, `GrowthStats`, `CRMPipeline` | Advanced analytics |
| `notification.go` | `Notification`, `ActivityEntry` | Event notifications |
| `support.go` | `SupportTicket`, `SupportMessage` | Support system |
| `webhook.go` | `Webhook`, `WebhookEvent` | User webhooks |
| `mfa.go` | `MFAEnrollment` | Multi-factor authentication |
| `session.go` | `Session` | User session management |
| `sso.go` | `SSOConfig` | Single sign-on config |
| `role.go` | `CustomRole`, `RoleAssignment` | RBAC custom roles |
| `waf.go` | `WAFRule`, `WAFEvent`, `WAFConfig` | Web application firewall |
| `alert_rule.go` | `AlertRule`, `CustomMetric` | Monitoring alert rules |
| `backup.go` | `DatabaseBackup` | Database backup records |
| `domain.go` | `CustomDomain` | Custom domain management |
| `project.go` | `Project` | Project entity (org unit) |
| `pod_session.go` | `PodExecSession` | SSH-to-pod sessions |
| `autoscale.go` | `AutoscalerConfig`, `HetznerNode` | Hetzner autoscaling |
| `network_policy.go` | `NetworkPolicyRule`, `NetworkPolicyConfig` | K8s network policies |
| `ip_whitelist.go` | `IPWhitelistEntry` | IP allowlist |
| `preview.go` | `PreviewDeployment` | Preview environments |
| `branding.go` | `DPARecord`, `BrandingConfig` | DPA + branding |
| `email.go` | `EmailSend` | Email campaign tracking |
| `exit_survey.go` | `ExitSurvey` | Churn feedback |
| `referral.go` | `ReferralReward` | Referral tracking |
| `user_event.go` | `UserEvent` | Analytics events |
| `events.go` | `PlatformEvent` | Event bus messages |
| `addon.go` | `AddOn`, `AddOnSubscription` | Marketplace add-ons |
| `common.go` | `Timestamps` | Shared `created_at`/`updated_at` |

### Example Entity

```go
// entities/app.go
type AppStatus string
const (
    AppStatusRunning  AppStatus = "running"
    AppStatusStopped  AppStatus = "stopped"
    AppStatusBuilding AppStatus = "building"
    AppStatusFailed   AppStatus = "failed"
)

type App struct {
    ID          string    `json:"id"`
    ProjectID   string    `json:"project_id"`
    UserID      string    `json:"user_id"`
    Name        string    `json:"name"`
    Image       string    `json:"image"`
    Replicas    int       `json:"replicas"`
    Port        int       `json:"port"`
    Status      AppStatus `json:"status"`
    Subdomain   string    `json:"subdomain"`
    URL         string    `json:"url"`
    Framework   Framework `json:"framework,omitempty"`
    AppType     AppType   `json:"app_type,omitempty"`
    Timestamps
}
```

**Rules:**
- No imports from other packages (only standard library)
- JSON tags for API serialization
- Status constants as typed strings
- Embed `Timestamps` for created_at/updated_at

---

## Layer 2: Ports (46 Interfaces)

Two files define ALL contracts:

### `ports/repositories.go` — 34 Data Interfaces

| Interface | Methods | Purpose |
|-----------|---------|---------|
| `UserRepository` | Create, Get, GetByEmail, Update, Delete, List | User CRUD |
| `ProjectRepository` | Create, Get, List, Update, Delete | Project CRUD |
| `AppRepository` | Create, Get, List, Update, Delete | App CRUD |
| `DatabaseRepository` | Create, Get, List, Update, Delete | Database CRUD |
| `StorageRepository` | Create, Get, List, Update, Delete | Bucket CRUD |
| `GatewayRepository` | Create, Get, List, Update, Delete + Routes, Groups, Domains (20+ methods) | Gateway CRUD |
| `AuthPoolRepository` | Create, Get, List, Update, Delete | Auth pool CRUD |
| `UserPlanRepository` | Get, Update, GetLimits | Plan limits |
| `BillingRepository` | Create, Get, List subscriptions + invoices | Billing records |
| `TeamMemberRepository` | Create, Get, List, Update, Delete | Team CRUD |
| `SupportRepository` | Create, Get, List tickets + messages | Support system |
| `AdminRepository` | Stats, Clusters, Tenants, Modules, Audit, Updates | Admin operations |
| `NotificationRepository` | Create, List, MarkRead | Notifications |
| `SessionRepository` | Create, Get, List, Delete | User sessions |
| `MFARepository` | Create, Get, Update, Delete | MFA enrollment |
| `APIKeyRepository` | Create, Get, List, Delete | API keys |
| `UserWebhookRepository` | Create, Get, List, Update, Delete | Webhooks |
| `RoleRepository` | Create, Get, List, Update, Delete | Custom roles |
| `IPWhitelistRepository` | Create, List, Delete | IP allowlist |
| `SSORepository` | Create, Get, List, Delete | SSO config |
| `BackupRepository` | Create, Get, List, Restore | Database backups |
| `PreviewRepository` | Create, Get, List | Preview envs |
| `BrandingRepository` | Get, Update | Branding config |
| `AutoscaleRepository` | Get, Update, ListNodes, ListActions | Autoscaler |
| `CustomerRepository` | Create, Get, List, Update, Delete | SaaS customers |
| `PodExecSessionRepository` | Create, Get, List, Update | Pod exec sessions |
| `UserEventRepository` | Create, List | Analytics events |
| `EmailSendRepository` | Create, List, Stats | Email tracking |
| `ExitSurveyRepository` | Create, List, Stats | Exit surveys |
| `ReferralRepository` | Create, Get, List, Summary | Referral tracking |
| `MeteringRepository` | Record, Query | Usage metering |
| `DomainRepository` | Create, Get, List, Delete | Custom domains |

### `ports/infrastructure.go` — 12 External System Interfaces

| Interface | Methods | Purpose |
|-----------|---------|---------|
| `KubernetesClient` | CRD operations (Create/Get/List/Update/Delete for App, Database, Gateway) | K8s API |
| `PaymentGateway` | CreateCheckout, CreatePortal, Cancel, HandleWebhook | Stripe |
| `ObjectStorage` | CreateBucket, DeleteBucket, PutObject, GetObject, DeleteObject, Presign | Hetzner S3 |
| `IdentityProvider` | CreateRealm, CreateUser, CreateRole, etc. | Keycloak |
| `ClusterProvisioner` | ProvisionCluster, DecommissionCluster, ScaleCluster | CAPI |
| `CloudProvider` | CreateServer, DeleteServer, ListServers | Hetzner Cloud |
| `ClusterOrchestrator` | StartWorkflow, SignalWorkflow | Temporal |
| `ProvisioningWorkflow` | Execute (the actual workflow) | Temporal workflows |
| `EmailSender` | SendEmail, SendBatch | Resend |
| `EventBus` | Publish, Subscribe, Close | NATS JetStream |
| `TokenGenerator` | Generate, Validate | JWT |

---

## Layer 3: Services (24 Files)

Services contain business logic. They import `entities`, `ports`, and `dto` — NEVER adapters.

### Key Services

| File | Key Methods | What It Does |
|------|------------|-------------|
| `auth.go` | Login, Register, VerifyEmail, Refresh, OAuth | Platform authentication |
| `authpool.go` | CreatePool, CreateUser, ListUsers, CreateRole, SendVerification | Keycloak realm management |
| `billing.go` | GetBillingStatus, CreateCheckoutSession, CancelSubscription | Stripe billing |
| `database.go` | ProvisionDatabase, DeleteDatabase, GetPassword, ResetPassword | CNPG database lifecycle |
| `gateway.go` | CreateGateway, CreateRoute, SyncGateway, ReconcileAll, AddDomain, GetAnalytics | APISIX gateway management |
| `monitoring.go` | GetOverview, GetTimeSeries, GetLogs, StreamLogs, GetPods | Prometheus/Loki proxy |
| `deploy/deployer.go` | StartBuild, WatchBuild, Deploy | Kaniko → Harbor → K8s |
| `plan.go` | GetUserPlan, UpgradePlan, CheckLimit, CalculateUsage | Plan tier enforcement |
| `support.go` | CreateTicket, ListMyTickets, AddMessage, AdminReply | Support ticket system |
| `team.go` | InviteMember, AcceptInvite, ListMembers, UpdateRole | Team management |
| `tenant_storage.go` | ListObjects, PutObject, GetObject, DeleteObject, Presign | S3 object operations |
| `storage_quota.go` | CheckUploadAllowed, InvalidateCache | Storage quota enforcement |
| `customer.go` | CreateCustomer, UpdateCustomer, DeleteCustomer | SaaS customer management |
| `admin.go` | GetDashboardStats, ListClusters, SuspendTenant, ApplyUpdate | Admin operations |
| `pgweb.go` | StartSession, GetSession, StopSession | Database explorer |
| `notifications.go` | Start (event subscriber) | Event-driven notifications |
| `email_campaign.go` | Start (drip campaigns) | Automated email flows |
| `dormant_cleanup.go` | Start (background cleanup) | Inactive resource cleanup |

### Example Service Method

```go
// services/gateway.go
func (s *GatewayService) CreateGateway(ctx context.Context, userID, projectID string, input dto.CreateGatewayInput) (*entities.Gateway, error) {
    // 1. Check plan limits
    plan, err := s.planRepo.GetUserPlan(ctx, userID)
    if err != nil {
        return nil, err
    }
    // ... limit check ...

    // 2. Create entity
    gw := &entities.Gateway{
        ID:        generateID(),
        UserID:    userID,
        ProjectID: projectID,
        Name:      input.Name,
        Slug:      slug.Make(input.Name),
        Status:    entities.GatewayStatusActive,
    }

    // 3. Persist via repository (interface — could be postgres or memory)
    gw, err = s.gatewayRepo.CreateGateway(ctx, gw)
    if err != nil {
        return nil, err
    }

    // 4. Create K8s resources (interface — could be real K8s or memory)
    if err := s.createIngressRoute(ctx, gw); err != nil {
        // cleanup...
    }

    return gw, nil
}
```

**Notice:** The service doesn't know if it's talking to PostgreSQL or an in-memory map. It doesn't know if it's creating a real K8s IngressRoute or a fake one. That's the power of hexagonal architecture.

---

## Layer 4: Handlers (83 Files, 376 Endpoints)

Handlers are the HTTP layer. They parse requests, call services, and return JSON.

### Route Groups

| Group | Path | Auth | Endpoints |
|-------|------|------|-----------|
| Health | `/health`, `/ready` | None | 2 |
| Metrics | `/metrics` | None | 1 |
| Auth | `/api/v1/auth/*` | Rate-limited | 8 (login, register, oauth, verify, refresh) |
| Auth Pools (public) | `/api/v1/auth-pools/:poolId/*` | Rate-limited | ~15 (signup, login, token, otp, magic-link) |
| Auth Pools (protected) | `/api/v1/auth-pools/*` | JWT | ~45 (pool/user/role CRUD) |
| Projects | `/api/v1/projects` | JWT | 5 |
| Apps | `/api/v1/apps` | JWT | 12 (CRUD + deploy + env + secrets) |
| Databases | `/api/v1/databases` | JWT | 10 (CRUD + backup + explorer) |
| Storage | `/api/v1/storage-buckets` | JWT | 12 (CRUD + objects + presign) |
| Gateways | `/api/v1/gateways` | JWT | 18 (CRUD + routes + groups + domains + analytics) |
| Team | `/api/v1/team` | JWT | 4 (invite, list, update, remove) |
| Billing | `/api/v1/billing` | JWT | 3 (checkout, portal, cancel) |
| Support | `/api/v1/support/tickets` | JWT | 4 (CRUD + messages) |
| Plan | `/api/v1/plan` | JWT | 2 (get, upgrade) |
| Notifications | `/api/v1/notifications` | JWT | 2 (list, mark-read) |
| MFA | `/api/v1/auth/mfa/*` | JWT | 5 (enable, verify, disable, backup codes) |
| Sessions | `/api/v1/sessions` | JWT | 3 (list, revoke) |
| API Keys | `/api/v1/api-keys` | JWT | 3 |
| Webhooks | `/api/v1/webhooks` | JWT | 4 |
| Domains | `/api/v1/domains` | JWT | 3 |
| Roles | `/api/v1/roles` | JWT | 5 |
| Admin | `/api/v1/admin/*` | Admin JWT | 100+ (dashboard, clusters, customers, security, CRM) |
| Monitoring | `/api/v1/monitoring` | JWT | 4 |

### Example Handler

```go
// handlers/gateway.go
func (h *GatewayHandler) CreateGateway(c *fiber.Ctx) error {
    userID := c.Locals("userID").(string)
    projectID := c.Params("projectId")

    var input dto.CreateGatewayInput
    if err := c.BodyParser(&input); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
    }

    gw, err := h.service.CreateGateway(c.Context(), userID, projectID, input)
    if err != nil {
        return handleServiceError(c, err)
    }

    return c.Status(201).JSON(gw)
}
```

---

## Layer 5: Adapters (16 Packages)

### PostgreSQL Adapter (`adapters/postgres/`)

**34 implementation files** + **37 migration files**

Each repository interface gets a PostgreSQL implementation. Example:

```go
// adapters/postgres/postgres_gateway.go
func (r *GatewayRepository) CreateGateway(ctx context.Context, gw *entities.Gateway) (*entities.Gateway, error) {
    _, err := r.db.ExecContext(ctx,
        `INSERT INTO gateways (id, user_id, project_id, name, slug, status, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
        gw.ID, gw.UserID, gw.ProjectID, gw.Name, gw.Slug, gw.Status,
    )
    return gw, err
}
```

### Memory Adapter (`adapters/memory/`)

**45 in-memory implementations** for dev/test. Uses Go maps and slices.

```go
// adapters/memory/memory_gateway.go
func (r *GatewayRepository) CreateGateway(ctx context.Context, gw *entities.Gateway) (*entities.Gateway, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.gateways[gw.ID] = gw
    return gw, nil
}
```

### External Adapters

| Adapter | What It Wraps | Key Methods |
|---------|-------------|-------------|
| `k8sclient/` | Kubernetes API (client-go) | CRD CRUD operations |
| `stripeclient/` | Stripe API | Checkout, Portal, Webhooks |
| `keycloakclient/` | Keycloak Admin API | Realm, User, Role management |
| `s3client/` | Hetzner S3 (AWS SDK) | PutObject, GetObject, Presign |
| `promclient/` | Prometheus HTTP API | Query, QueryRange |
| `lokiclient/` | Loki HTTP API | QueryRange, Labels |
| `natsclient/` | NATS JetStream | Publish, Subscribe |
| `resendclient/` | Resend Email API | Send transactional emails |
| `harborclient/` | Harbor API | Project, Repository management |
| `hetznerclient/` | Hetzner Cloud API | Server provisioning |
| `capiclient/` | CAPI API | Cluster provisioning |
| `redisclient/` | Redis | Rate limiter, token blacklist |

---

## Middleware (10 Files)

| File | Function | What It Does |
|------|----------|-------------|
| `auth.go` | `RequireAuth()` | Extracts JWT from header, validates, sets `userID` in context |
| `ownership.go` | `RequireAppOwnership()` | Checks if the authenticated user owns the requested resource (IDOR prevention) |
| `logging.go` | `StructuredLogger()` | JSON structured logging with request ID, duration, status |
| `security.go` | `SecurityHeaders()` | HSTS, CSP, X-Frame-Options, Referrer-Policy |
| `context.go` | `RequestContext()` | Extract request context (user, project) for downstream use |
| `admin_permission.go` | `RequireAdminRole()` | Admin-only endpoint protection |
| `tokenblacklist.go` | `NewTokenBlacklist()` | JWT revocation on logout (in-memory or Redis-backed) |

---

## Configuration (65+ Environment Variables)

Key configuration categories:

| Category | Variables | Example |
|----------|----------|---------|
| Core | `PORT`, `ENVIRONMENT`, `MODE`, `CORS_ORIGINS` | `8080`, `staging`, `saas` |
| Database | `DATABASE_URL` | `postgres://user:pass@host:5432/zenith` |
| JWT | `JWT_SECRET`, `JWT_ISSUER` | Random 64-char string |
| Admin | `ADMIN_EMAIL`, `ADMIN_PASSWORD` | `admin@freezenith.com` |
| OAuth | `GOOGLE_CLIENT_ID`, `GITHUB_CLIENT_ID` | Google/GitHub OAuth app IDs |
| Stripe | `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET` | Stripe API keys |
| K8s | `K8S_MODE`, `KUBECONFIG`, `IN_CLUSTER` | `memory` or `real` |
| Deploy | `BASE_DOMAIN`, `GATEWAY_DOMAIN`, `HARBOR_URL` | `apps.stage.freezenith.com` |
| S3 | `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` | Hetzner S3 creds |
| Keycloak | `KEYCLOAK_URL`, `KEYCLOAK_ADMIN_USER` | Keycloak admin API |
| Monitoring | `PROMETHEUS_URL`, `LOKI_URL`, `GRAFANA_URL` | Internal service URLs |
| Email | `RESEND_API_KEY`, `EMAIL_FROM` | Resend API key |
| NATS | `NATS_ENABLED`, `NATS_SERVERS` | NATS connection |
| Temporal | `TEMPORAL_ENABLED`, `TEMPORAL_HOST` | Temporal connection |
| Secrets | `SECRETS_KEY` | AES-256 encryption key (64 hex chars) |

**Feature Flags:** `K8S_MODE=memory` → uses in-memory adapters (no real K8s needed for dev)

---

## Database Migrations (37 Versioned)

Located at `services/api/internal/adapters/postgres/migrations/`

Each migration has an `.up.sql` (apply) and `.down.sql` (rollback).

| # | Name | What It Creates |
|---|------|-----------------|
| 001 | initial | users, apps, auth tables |
| 002 | seed_defaults | Default roles, plan tiers |
| 003 | customers_plans | SaaS customer + plan tables |
| 004-005 | cluster/metering | CAPI metadata, usage tracking |
| 006-008 | apps extended | App secrets, releases |
| 009 | billing | Stripe subscription + invoice tables |
| 010-011 | email/customer | Email verification, customer role |
| 012-013 | app deploy | Image-based deploy, app type enum |
| 014-018 | storage/database | S3 buckets, database provisioning |
| 019 | gateways | APISIX gateway + route tables |
| 020-023 | projects/domains/auth | Projects, custom domains, auth pools |
| 024-025 | team/support | Team members, support tickets |
| 026 | phase0_all_tables | Comprehensive table rebuild |
| 027-029 | notifications/admin | Notifications, API key fix, admin v2 |
| 030-037 | features | Gateway groups, app exposure, analytics, referrals, custom domains |

**Running migrations:**
```bash
# Applied automatically on API startup (golang-migrate/migrate)
# Or manually:
lich migration up
lich migration create "description"
```

---

## main.go — The Composition Root (1500 Lines)

This is where everything gets wired together. The flow:

```
1. Load Config (env vars)
2. Setup Logging (slog JSON)
3. Connect Database (PostgreSQL or memory fallback)
4. Create ALL Repositories (34 interfaces × 2 implementations)
5. Create ALL Services (24 services, inject repos)
6. Create External Adapters (K8s, Stripe, Keycloak, S3, etc.)
7. Inject Adapters into Services
8. Create Middleware (auth, logging, security)
9. Create ALL Handlers (inject services)
10. Register ALL Routes (376 endpoints)
11. Start Background Workers (provisioner, autoscaler, event bus, cleanup)
12. Listen on :8080
13. Graceful Shutdown (30s timeout)
```

**Deployment Modes:**
- `MODE=standalone` — Single-tenant, no billing, no Temporal
- `MODE=saas` — Multi-tenant, Stripe billing, cluster provisioning, Temporal workflows

---

## How to Add a New Feature (Step-by-Step)

### Example: Adding "App Rollback" Feature

**Step 1: Entity** (if new data model needed)
```go
// entities/deployment.go — already exists, add:
type RollbackRequest struct {
    DeploymentID string `json:"deployment_id"`
    Reason       string `json:"reason,omitempty"`
}
```

**Step 2: Port** (if new repository method needed)
```go
// ports/repositories.go — add to AppRepository:
GetDeploymentByID(ctx context.Context, id string) (*entities.Deployment, error)
```

**Step 3: Adapter** (implement the new method)
```go
// adapters/postgres/postgres_app.go
func (r *AppRepository) GetDeploymentByID(ctx context.Context, id string) (*entities.Deployment, error) {
    // SQL query...
}

// adapters/memory/memory_app.go
func (r *AppRepository) GetDeploymentByID(ctx context.Context, id string) (*entities.Deployment, error) {
    // Map lookup...
}
```

**Step 4: Service** (business logic)
```go
// services/deploy/deployer.go
func (s *DeployService) RollbackApp(ctx context.Context, userID, appID, deploymentID string) error {
    // 1. Get deployment
    // 2. Verify ownership
    // 3. Update K8s Deployment to old image
    // 4. Create new deployment record
    // 5. Publish event
}
```

**Step 5: Handler** (HTTP endpoint)
```go
// handlers/deployments.go
func (h *DeployHandler) RollbackDeployment(c *fiber.Ctx) error {
    // Parse request, call service, return response
}
```

**Step 6: Route** (register in main.go)
```go
// cmd/server/main.go
appByID.Post("/rollback", deployHandler.RollbackDeployment)
```

**Step 7: Verify**
```bash
cd services/api && GO111MODULE=on go vet ./internal/...
```

---

## Testing

```bash
# Run all tests
cd services/api && go test ./internal/...

# Run specific package
go test ./internal/handlers/ -v

# Run with coverage
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Tests use memory adapters — no Docker, no database, no network needed.

---

**Next → [04 — Frontend Complete Guide](./04-frontend-guide.md)**
