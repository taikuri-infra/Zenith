# Zenith — Developer Onboarding Guide

> **Audience:** New developers joining the Zenith team. No prior context assumed.
> **Goal:** After reading this, you should understand what Zenith is, how it works, and how to contribute code.
> **Last updated:** 2026-03-09 (based on V3 architecture)

---

## Table of Contents

1. [What Is Zenith?](#1-what-is-zenith)
2. [Business Model & Pricing](#2-business-model--pricing)
3. [Prerequisites & Setup](#3-prerequisites--setup)
4. [Project Structure](#4-project-structure)
5. [Architecture Overview](#5-architecture-overview)
6. [The Go API (Backend)](#6-the-go-api-backend)
7. [The Web Dashboard (Frontend)](#7-the-web-dashboard-frontend)
8. [Infrastructure & DevOps](#8-infrastructure--devops)
9. [How Deployments Work](#9-how-deployments-work)
10. [Authentication & Authorization](#10-authentication--authorization)
11. [Database & Data Layer](#11-database--data-layer)
12. [Monitoring & Observability](#12-monitoring--observability)
13. [Security Model](#13-security-model)
14. [Development Workflow](#14-development-workflow)
15. [CI/CD & Release Process](#15-cicd--release-process)
16. [Key Patterns & Conventions](#16-key-patterns--conventions)
17. [Glossary](#17-glossary)
18. [Your First Week](#18-your-first-week)

---

## 1. What Is Zenith?

Zenith is a **Kubernetes-native Platform-as-a-Service (PaaS)**. Think of it as a self-hosted Heroku or Railway, but built on Hetzner Cloud with enterprise features.

**What customers do with Zenith:**
- Deploy web apps from Git repos or Docker images
- Get managed PostgreSQL, Redis, MongoDB databases
- Use S3-compatible object storage
- Configure API gateways with JWT auth and rate limiting
- Set up custom domains with automatic TLS certificates
- Monitor apps with metrics, logs, and pod health
- Manage teams with RBAC (role-based access control)

**What makes Zenith different:**
- European infrastructure (Hetzner, Germany) — GDPR compliant
- 3-5x cheaper than AWS/GCP
- Open-source stack — no vendor lock-in
- Everything runs on Kubernetes with operators for Day-2 operations

**Three user-facing applications:**
1. **Web Dashboard** (`apps/web`) — Where customers deploy and manage their apps
2. **Mission Control** (`apps/mission-control`) — Admin panel for platform operators
3. **Landing Page** (`apps/landing`) — Marketing site at freezenith.com

**One core backend:**
- **Zenith API** (`services/api`) — Go/Fiber REST API that powers everything

---

## 2. Business Model & Pricing

### The "Audi Strategy"

Premium quality at accessible pricing. Not the cheapest, not the most expensive. We use a **McDonald's Decoy Effect** — the Team tier is priced to make Business look like the obvious choice.

### Plan Tiers

| Tier | Price | Apps | Databases | Key Feature |
|------|-------|------|-----------|-------------|
| **Free** | €0/mo | 1 | 1 (100MB) | Scale-to-zero after 15 min |
| **Pro** | €29/mo | 5 | 3 (5GB) | Always-on, custom domains |
| **Team** | €99/seat (min 3) | 20 | 10 (20GB) | RBAC, team management |
| **Business** | €149/seat (min 3) | Unlimited | Unlimited | Dedicated infra, SSO, audit |
| **Enterprise** | Custom | Custom | Custom | SLA, dedicated support |

**Revenue comes from:**
- Subscriptions (Stripe)
- Support add-ons (Gold €699/mo, Platinum €1499/mo)
- Resource add-ons (extra compute, storage, databases)

### Plan Enforcement in Code

Plan limits are defined in `services/api/internal/entities/plan.go` → `DefaultPlanLimits()`. Every resource-creating API endpoint checks limits via the `CheckLimit` middleware:

```go
// In main.go route setup
apps.Post("/", handlers.CheckLimit(planRepo, "apps", countFunc), appHandler.Create)
```

If a user exceeds their plan's limit, the API returns 403 with a message like "App limit reached. Upgrade your plan."

---

## 3. Prerequisites & Setup

Before you can work on Zenith, install these tools on your machine.

### Required Tools

| Tool | Version | Install | Why |
|------|---------|---------|-----|
| **Git** | 2.40+ | [git-scm.com](https://git-scm.com/) | Version control |
| **Go** | 1.23+ | [go.dev/dl](https://go.dev/dl/) | Backend (API, CLI, Operator, Terraform Provider) |
| **Node.js** | 20 LTS | [nodejs.org](https://nodejs.org/) or `nvm install 20` | Frontend build tooling |
| **pnpm** | 10.x | `npm install -g pnpm@10` | Monorepo package manager |
| **Docker** | 24+ | [docker.com](https://www.docker.com/) | Container builds, local PostgreSQL |

### Recommended Tools

| Tool | Install | Why |
|------|---------|-----|
| **Helm** | `brew install helm` | Validate Helm charts locally |
| **kubectl** | `brew install kubectl` | Interact with staging K8s cluster |
| **act** | `brew install act` | Run GitHub Actions workflows locally |
| **gh** | `brew install gh` | GitHub CLI (PRs, issues, auth tokens) |
| **release-please** | `npm install -g release-please` | Test release dry-runs locally |

### Clone & Install

```bash
# 1. Clone the repo
git clone git@github.com:DoTechHQ/Zenith.git
cd Zenith

# 2. Install Node.js dependencies (frontend + commitlint)
pnpm install

# 3. Verify Go builds
cd services/api && go build ./cmd/server && cd ../..
cd services/cli && go build ./cmd/zenith/ && cd ../..

# 4. (Optional) Start PostgreSQL for persistent dev data
docker compose up -d

# 5. Verify everything works
pnpm turbo lint                    # Frontend lint
cd services/api && go test ./...   # Backend tests
npx commitlint --from HEAD~1      # Commit message lint
```

### Environment Setup

Copy the example env file and fill in values:

```bash
cp .env.example .env
```

Key variables for local dev:

| Variable | Value for Local Dev |
|----------|-------------------|
| `DATABASE_URL` | `postgres://zenith:zenith@localhost:5432/zenith?sslmode=disable` (or leave empty for in-memory) |
| `JWT_SECRET` | Any random string (e.g., `dev-secret-123`) |
| `K8S_MODE` | `memory` (no real K8s needed locally) |
| `ZENITH_MODE` | `standalone` |
| `CORS_ORIGINS` | `http://localhost:3000` |

---

## 4. Project Structure

```
Zenith/
├── apps/                      # Frontend applications
│   ├── web/                   # Customer dashboard (Next.js 15)
│   ├── mission-control/       # Admin panel (Next.js 15)
│   └── landing/               # Marketing site (Next.js 15)
│
├── services/                  # Backend services
│   ├── api/                   # Core REST API (Go/Fiber)
│   ├── operator/              # Kubernetes operator (Go)
│   ├── cli/                   # CLI tool (Go/Cobra)
│   └── terraform-provider-zenith/  # Terraform provider
│
├── packages/
│   └── ui/                    # Shared UI utilities
│
├── infra/                     # Infrastructure as Code
│   ├── terraform/             # Hetzner VMs, DNS, Helm releases
│   ├── ansible/               # Server bootstrap (k3s, Cilium)
│   ├── helm/                  # Kubernetes Helm charts
│   ├── argocd/                # GitOps Application manifests
│   └── scripts/               # Deployment & smoke test scripts
│
├── docs/                      # Documentation
│   ├── v3-architecture.md     # THE source of truth (2000+ lines)
│   ├── v2-architecture/       # Legacy docs (still useful for detail)
│   └── runbooks/              # Operational procedures
│
├── .lich/                     # Lich Framework (project scaffolding)
│   ├── rules/                 # Architecture rules for AI agents
│   └── workflows/             # Step-by-step guides
│
├── openspec/                  # Change proposals & specs
├── .github/workflows/         # CI/CD pipelines
├── AGENTS.md                  # Master rules file
└── agentlog.md                # Change history
```

### Key Configuration Files

| File | Purpose |
|------|---------|
| `services/api/go.mod` | Go dependencies |
| `apps/web/package.json` | Frontend dependencies |
| `pnpm-workspace.yaml` | Monorepo workspace config |
| `docker-compose.yml` | Local dev environment |
| `.env.example` | Environment variable template |

---

## 5. Architecture Overview

### High-Level Flow

```
User → Cloudflare → Traefik (TLS) → APISIX (JWT, Rate Limit) → Zenith API → Data Layer
```

1. **Cloudflare** — CDN, WAF, DDoS protection, DNS
2. **Traefik** — Ingress controller, TLS termination (uses IngressRoute CRD, NOT standard Ingress)
3. **APISIX** — API Gateway for JWT verification, CORS, rate limiting
4. **Zenith API** — Go/Fiber REST API, the brain of the platform
5. **Data Layer** — PostgreSQL (CNPG), Redis, S3, Keycloak, Kubernetes API

### Component Stack (8 Layers)

| Layer | Components |
|-------|-----------|
| **Edge** | Cloudflare (CDN, WAF, DDoS) |
| **Networking** | Traefik, APISIX, Cilium (CNI + WireGuard), external-dns |
| **Identity** | Keycloak (SSO/OIDC), cert-manager (TLS), Kyverno (policies) |
| **Data** | CNPG (PostgreSQL), Redis, MongoDB, RabbitMQ, Kafka, Hetzner S3 |
| **Platform** | Zenith API, Web, Admin, Operator, Temporal, Harbor, ArgoCD, NATS |
| **Observability** | Prometheus, Loki, Tempo, OpenTelemetry, Grafana, Alertmanager |
| **Resilience** | Velero (backup), CNPG WAL archiving, DR cluster (Finland) |
| **Scaling** | KEDA (pod autoscaling), Hetzner node autoscaler |

### Design Principles

1. **Operator-First** — Every stateful service uses a Kubernetes Operator
2. **Multi-Tenant** — Isolated via namespaces (Free/Pro) or dedicated infra (Business)
3. **API-as-Proxy** — The API is a secure proxy to infrastructure. Users never touch K8s directly
4. **Event-Driven** — Operations flow through NATS JetStream event bus
5. **Rebuildable from Git** — Terraform + ArgoCD + Sealed Secrets = full rebuild in 30 min
6. **Day-2 First** — Everything is designed for maintainability, not just initial deployment

### Kubernetes Namespace Strategy

```
PLATFORM:    zenith-staging, monitoring, argocd, cert-manager, keycloak, temporal, harbor
SHARED:      zenith-apps (customer pods), zenith-builds (Kaniko jobs), zenith-shared (shared CNPG)
DEDICATED:   zenith-customer-<id> (Business tier gets their own namespace)
```

---

## 6. The Go API (Backend)

The API is the most important part of the codebase. It lives in `services/api/`.

### Architecture: Hexagonal (Ports & Adapters)

```
HTTP Request → Handler → Service → Port (interface) → Adapter (implementation)
```

- **Entities** (`internal/entities/`) — Pure domain models. No dependencies. 34 files.
- **Ports** (`internal/ports/`) — Interfaces for repositories and infrastructure. 50+ interfaces.
- **Adapters** (`internal/adapters/`) — Implementations:
  - `postgres/` — PostgreSQL repositories (31 files)
  - `memory/` — In-memory fallback stores (37 files, used in dev/standalone mode)
  - `k8sclient/` — Kubernetes API client
  - `s3client/` — S3 object storage
  - `harborclient/` — Harbor container registry
  - `keycloakclient/` — Keycloak identity provider
  - `stripeclient/` — Stripe payments
  - `natsclient/` — NATS event bus
  - `redisclient/` — Redis cache + rate limiting
  - `resendclient/` — Email (Resend API)
  - `promclient/` — Prometheus metrics
  - `lokiclient/` — Loki log aggregation
  - `hetznerclient/` — Hetzner Cloud API
  - `capiclient/` — Cluster API provisioner
- **Services** (`internal/services/`) — Business logic. 20+ service modules.
- **Handlers** (`internal/handlers/`) — HTTP route handlers. 56 files.
- **Middleware** (`internal/middleware/`) — Auth, ownership checks, rate limiting, security headers.
- **Config** (`internal/config/`) — Environment variable loading with defaults.

### Entry Point: `cmd/server/main.go`

The main function does these things in order:

1. **Load config** from environment variables
2. **Connect to PostgreSQL** (if `DATABASE_URL` or `DB_HOST` is set) and run migrations
3. **Initialize repositories** — PostgreSQL if DB is available, else in-memory
4. **Seed admin user** — Creates admin from `ADMIN_EMAIL` / `ADMIN_PASSWORD`
5. **Set up HTTP server** (Fiber) with middleware stack
6. **Register all routes** — 100+ endpoints
7. **Start background workers** — Temporal, autoscaler, cluster provisioner
8. **Listen on :8080** with graceful shutdown (30s timeout)

### Key Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `DATABASE_URL` or `DB_HOST` | PostgreSQL connection | (none = in-memory) |
| `JWT_SECRET` | Token signing key | (required) |
| `K8S_MODE` | `memory` or `real` | `memory` |
| `ZENITH_MODE` | `standalone` or `saas` | `standalone` |
| `CORS_ORIGINS` | Allowed origins | `http://localhost:3000` |
| `RESEND_API_KEY` | Email service | (optional) |
| `STRIPE_SECRET_KEY` | Payment processing | (optional) |
| `PROMETHEUS_URL` | Metrics source | `http://kube-prometheus-stack-prometheus...` |
| `LOKI_URL` | Logs source | `http://loki...` |
| `REDIS_URL` | Cache + rate limiting | (optional, falls back to in-memory) |

Full list: `services/api/internal/config/config.go`

### API Routes Overview

All routes are under `/api/v1/`. Here are the main groups:

**Public (no auth):**
- `POST /auth/register` — Create account
- `POST /auth/login` — Login (returns JWT)
- `POST /auth/verify-email` — Email verification
- `GET /auth/oauth/:provider` — OAuth redirect (Google, GitHub)

**Protected (JWT required):**
- `GET/POST /apps` — App CRUD
- `GET/POST /databases` — Database management
- `GET/POST /gateways` — API Gateway management
- `GET /plan` — Current plan & usage
- `POST /plan/upgrade` — Change plan
- `GET /billing` — Billing status
- `GET /api-keys` — API key management
- `GET /team/members` — Team management
- `GET /notifications` — Notifications
- `GET /activity` — Activity log
- `GET /support/tickets` — Support tickets (Pro+)
- `GET /auth/sessions` — Session management
- `GET /auth/mfa` — MFA status

**Per-app (JWT + ownership check):**
- `GET /apps/:appId/deployments` — Deployment history
- `GET /apps/:appId/env` — Environment variables
- `GET /apps/:appId/secrets` — Encrypted secrets
- `GET /apps/:appId/metrics/overview` — App metrics
- `GET /apps/:appId/pods` — Pod status
- `GET /apps/:appId/logs` — App logs (from Loki)
- `POST /apps/:appId/domains` — Custom domains

### How to Add a New Endpoint

1. Define the entity in `internal/entities/`
2. Define the repository interface in `internal/ports/repositories.go`
3. Implement PostgreSQL adapter in `internal/adapters/postgres/`
4. Implement memory adapter in `internal/adapters/memory/`
5. Write the handler in `internal/handlers/`
6. Add a SQL migration in `internal/adapters/postgres/migrations/`
7. Wire the route in `cmd/server/main.go` → `setupRoutes()`
8. Run `go vet ./...` and `go build ./...`

---

## 7. The Web Dashboard (Frontend)

The customer dashboard lives in `apps/web/`. It's a Next.js 15 app with App Router.

### Tech Stack

- **Next.js 15** with App Router (not Pages Router)
- **React 19** with TypeScript
- **Tailwind CSS** for styling (dark theme by default)
- **Lucide React** for icons
- **Docker output: standalone** for production deployment

### Key Files

| File | Purpose |
|------|---------|
| `src/app/layout.tsx` | Root layout (dark mode, ToastProvider) |
| `src/app/login/page.tsx` | Login/register/OAuth page |
| `src/components/shell.tsx` | Authenticated layout wrapper |
| `src/components/sidebar.tsx` | Navigation sidebar |
| `src/components/deploy-wizard.tsx` | App deployment form (51KB) |
| `src/lib/api.ts` | API client (54KB, ALL endpoints) |
| `src/lib/demo-api.ts` | Demo mode mock data (70KB) |
| `src/lib/get-api.ts` | Returns real or demo API based on mode |
| `src/lib/runtime-env.ts` | Runtime environment config |
| `src/hooks/useAuth.ts` | Auth state management |
| `src/hooks/useApi.ts` | Data fetching hook |
| `src/middleware.ts` | Route protection |

### Pages (35 total)

Core pages: `/` (dashboard), `/apps`, `/apps/[name]`, `/databases`, `/storage`, `/gateway`, `/monitoring`, `/logs`, `/billing`, `/settings`, `/support`, `/iam`, `/registry`, `/queues`, `/alerts`

### API Client Pattern

```typescript
// src/lib/api.ts
const api = {
  apps: {
    list: () => apiFetch<App[]>('/apps'),
    get: (id: string) => apiFetch<App>(`/apps/${id}`),
    create: (data: CreateAppInput) => apiFetch<App>('/apps', { method: 'POST', body: data }),
    delete: (id: string) => apiFetch<void>(`/apps/${id}`, { method: 'DELETE' }),
  },
  // ... 30+ namespaces
};
```

### Demo Mode

When `NEXT_PUBLIC_DEMO_MODE=true`, the dashboard uses mock data from `demo-api.ts` instead of real API calls. This is for marketing demos. The API client has two implementations:

```typescript
// src/lib/get-api.ts
export function getApi() {
  return isDemoMode() ? demoApi : realApi;
}
```

### How to Add a New Page

1. Create `src/app/your-page/page.tsx`
2. Import `useApi` hook for data fetching
3. Add navigation link in `src/components/sidebar.tsx`
4. Add API endpoints to `src/lib/api.ts` if needed
5. Add demo data to `src/lib/demo-api.ts` for demo mode

---

## 8. Infrastructure & DevOps

### Staging Environment

- **Server:** Hetzner CPX31 (4 vCPU, 8GB RAM), IP: 77.42.88.149
- **K8s:** k3s v1.34.3 with Traefik 3.5.1
- **SSH alias:** `zen-stage`
- **Domain:** `*.stage.freezenith.com`
- **ArgoCD:** Watches `staging` branch, auto-syncs Helm releases

### How Staging Deployment Works

```
1. Developer pushes to `staging` branch
2. ArgoCD detects change in Helm values
3. ArgoCD syncs → updates Kubernetes Deployment
4. New pod pulls image from Harbor registry
5. Old pod terminates (rolling update)
```

For the API specifically:
```
1. Build Docker image on staging server:
   ssh zen-stage "cd /root/Zenith && docker build -f services/api/Dockerfile -t zenith-api:X.Y.Z ."

2. Push to Harbor:
   docker tag zenith-api:X.Y.Z registry.stage.freezenith.com/zenith-stage/zenith-api:X.Y.Z
   docker push ...

3. Update Helm values:
   infra/helm/zenith-api/values-staging.yaml → image: zenith-api:X.Y.Z

4. Commit and push to `staging` branch → ArgoCD picks it up
```

### Terraform Structure

```
infra/terraform/
├── staging/           # Hetzner VM + Cloudflare DNS for staging
├── staging-k8s/       # K8s resources (Helm releases, CNPG clusters)
├── production/        # Same, for production
└── modules/           # Reusable modules
    └── k8s-platform/  # All Helm charts and K8s resources
```

### Helm Charts

Each service has its own Helm chart in `infra/helm/`:

| Chart | Purpose |
|-------|---------|
| `zenith-api/` | API deployment + service + ingress + RBAC |
| `zenith-web/` | Web dashboard deployment |
| `zenith-platform/` | Shared secrets, CNPG cluster, RBAC |
| `zenith-tenant/` | Per-customer resources (Business tier) |
| `monitoring/` | Prometheus, Grafana, Loki, Tempo |

Each chart has `values-staging.yaml` and `values-production.yaml`.

### Two Harbor Registries (Important!)

1. **Internal Harbor** (`registry.stage.freezenith.com`) — Platform images (zenith-api, zenith-web). Managed OUTSIDE the cluster.
2. **Customer Harbor** (`hub.stage.freezenith.com`) — Pro-tier customer images. Deployed INSIDE the cluster via Terraform.

Don't confuse them.

---

## 9. How Deployments Work

When a customer deploys an app, here's what happens:

### Image Deploy Flow

```
1. Customer creates app via dashboard:
   POST /api/v1/apps { name: "my-app", deploy_source: "image", image_url: "nginx:alpine" }

2. API creates App entity in PostgreSQL (status: "pending")

3. API triggers deploy pipeline (async goroutine):
   services/api/internal/services/deploy/pipeline.go → TriggerImageDeploy()

4. Deployer creates Kubernetes resources:
   - Deployment (pod spec with customer's image)
   - Service (ClusterIP, routes to pod)
   - IngressRoute (Traefik CRD, {app}.apps.stage.freezenith.com → Service)
   - Certificate (cert-manager, auto TLS)
   - NetworkPolicy (Cilium, tenant isolation)
   - HTTPScaledObject (KEDA, scale-to-zero for Free tier)

5. Status updates: pending → deploying → running

6. App is accessible at https://{app-name}.apps.stage.freezenith.com
```

### Git Deploy Flow

```
1. Customer creates app with Git source:
   POST /api/v1/apps { name: "my-app", deploy_source: "git", repo_url: "https://github.com/..." }

2. GitHub webhook triggers on push

3. API creates Kaniko build Job in zenith-builds namespace:
   - Clones repo
   - Builds Dockerfile
   - Pushes image to Harbor

4. On build success: same deploy flow as Image Deploy

5. Build logs streamed via WebSocket (LogHub)
```

### Free Tier Scale-to-Zero

Free tier apps use KEDA HTTPScaledObject:
- After 15 min of no traffic → scale to 0 pods
- When traffic arrives → show cold-start splash page (nginx) for ~5 seconds
- KEDA scales pod back up → traffic flows to real app
- Splash page auto-refreshes every 5 seconds

---

## 10. Authentication & Authorization

### JWT Token Flow

```
Login → API generates JWT (24h expiry) → Stored in localStorage → Sent as Bearer token
```

JWT Claims:
```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "role": "customer",
  "project_id": "project-uuid",
  "account_id": "",
  "exp": 1709913600,
  "iss": "zenith"
}
```

### Team Member Access (AccountID Pattern)

When a team member logs in, their JWT includes `account_id` (the team owner's user ID). The API middleware swaps `user_id` to `account_id` so team members see the owner's resources:

```go
// middleware/auth.go
if claims.AccountID != "" {
  c.Locals("user_id", claims.AccountID)  // Use owner's ID
  c.Locals("member_id", claims.MemberID) // Keep member ID for audit
}
```

### OAuth2 (Google & GitHub)

```
Dashboard → "Login with Google" → /auth/oauth/google → Google consent → /auth/oauth/google/callback → JWT issued
```

### MFA (Pro+ users)

Pro and higher plans require MFA (TOTP, Google Authenticator). Login returns `mfa_required: true` + `mfa_token`. User enters 6-digit code → verified via TOTP → real JWT issued.

### IDOR Prevention

Every resource access checks ownership:
```go
// Middleware
appByID := apps.Group("/:appId", middleware.RequireAppOwnership(appRepo))
```

User A cannot access User B's apps, databases, or any other resource.

---

## 11. Database & Data Layer

### Platform Database

The Zenith API stores its own data in PostgreSQL via CloudNativePG (CNPG) operator.

**Staging setup:**
- CNPG cluster: `zenith-postgres` in `zenith-staging` namespace
- Database name: `zenith`
- 48 tables including: users, apps, customers, plans, deployments, api_keys, sessions, notifications, etc.

**Migrations:**
- Located in `services/api/internal/adapters/postgres/migrations/`
- Numbered: `001_initial.up.sql` through `028_fix_api_keys_project_id.up.sql`
- Embedded in binary via Go `//go:embed`
- Run automatically on startup via `golang-migrate`

### Customer Databases

Customers get their own PostgreSQL databases provisioned via CNPG:

- **Free/Pro:** Database on shared CNPG cluster (`free-pg` in `zenith-shared`)
- **Business:** Dedicated CNPG cluster in customer's namespace

Provisioning uses the `CNPG_ADMIN_DSN` to run `CREATE DATABASE` and `CREATE USER` commands.

### Dual Storage Mode

The API supports two storage modes:
- **PostgreSQL** — Production. Used when `DATABASE_URL` or `DB_HOST` env var is set.
- **In-Memory** — Development. Used when no database is configured. Data lost on restart.

Both implement the same `ports.Repository` interfaces, so the business logic doesn't change.

---

## 12. Monitoring & Observability

### Stack

| Tool | Purpose | Where |
|------|---------|-------|
| Prometheus | Metrics collection (CPU, memory, request rates) | `monitoring` namespace |
| Loki | Log aggregation (all pod logs) | `monitoring` namespace |
| Tempo | Distributed tracing | `monitoring` namespace |
| Grafana | Dashboards & visualization | `monitoring` namespace |
| Promtail | Ships pod logs to Loki | DaemonSet |
| OpenTelemetry | Trace collection & forwarding | `monitoring` namespace |
| Alertmanager | Alert routing (Slack, PagerDuty) | `monitoring` namespace |

### Customer-Facing Monitoring

The API proxies Prometheus and Loki queries scoped to each customer's pods:

```
GET /api/v1/apps/:appId/metrics/overview → Queries Prometheus for CPU, memory, request stats
GET /api/v1/apps/:appId/logs → Queries Loki for app logs
GET /api/v1/apps/:appId/pods → Queries K8s API for pod status
```

Customers never get direct access to Prometheus or Loki.

### API Logging

The API uses Go's `slog` for structured JSON logging:
```json
{"time":"2026-03-09T00:00:19Z","level":"INFO","msg":"request","method":"GET","path":"/health","status":200,"latency":18927}
```

---

## 13. Security Model

### 9 Security Layers

1. **Edge** — Cloudflare WAF, DDoS protection
2. **Network** — Cilium CNI with WireGuard encryption between all pods
3. **API Gateway** — APISIX: JWT verification, rate limiting, CORS
4. **Application Auth** — JWT with short expiry, refresh token rotation, MFA
5. **Container Security** — SecurityContext: runAsNonRoot, readOnlyRootFilesystem, drop ALL capabilities
6. **Image Security** — Harbor Trivy scan, Kyverno denies unscanned images
7. **Runtime Security** — Falco detects anomalous container behavior
8. **Data Encryption** — TLS everywhere, AES-256-GCM for secrets, WireGuard for pod traffic
9. **Audit** — K8s API audit logs, application audit trail

### Key Security Features in Code

**Secrets encryption** (`services/api/pkg/crypto/`):
```go
// App secrets are encrypted with AES-256-GCM before storage
Encrypt(plaintext, key) → ciphertext
Decrypt(ciphertext, key) → plaintext
```

**Rate limiting:**
- Auth endpoints: 10 requests per 60 seconds
- General API: configurable per-endpoint limits
- Backed by Redis (or in-memory fallback)

**Token blacklist:**
- On logout, JWT is blacklisted until expiry
- Stored in Redis with TTL = token remaining lifetime

---

## 14. Development Workflow

### Running Locally

See [Section 3: Prerequisites & Setup](#3-prerequisites--setup) for installation. Quick start:

```bash
# Terminal 1: Start API (standalone mode, in-memory DB)
cd services/api && go run ./cmd/server
# API runs at localhost:8080

# Terminal 2: Start Web Dashboard
cd apps/web && pnpm dev
# Dashboard runs at localhost:3000
```

In standalone mode (no DB configured), the API uses in-memory storage. Fine for frontend development but data is lost on restart.

### With PostgreSQL (recommended)

```bash
docker compose up -d
export DATABASE_URL="postgres://zenith:zenith@localhost:5432/zenith?sslmode=disable"
cd services/api && go run ./cmd/server
```

### Git Workflow

- **`main`** — Production-ready code. Release Please watches this branch.
- **`staging`** — Deployed to staging (ArgoCD watches this branch). Auto-updated by release pipeline.
- **Feature branches** — `feat/description`, `fix/description`
- **PRs to `main`** — CI validates tests, security, and commit messages before merge

### Commit Convention (Enforced by CI)

We use [Conventional Commits](https://www.conventionalcommits.org/). This is **enforced by CI** — PRs with bad commit messages will fail the `commitlint` check.

**Format:** `type(scope): description`

```
feat(api): add support ticket system       # → triggers minor version bump
fix(web): fix toast notification positioning # → triggers patch version bump
feat!: redesign authentication flow         # → triggers major version bump (breaking)
docs: update onboarding guide               # → no version bump (hidden in changelog)
chore(deps): bump Go to 1.25               # → no version bump (hidden in changelog)
perf(api): optimize database queries        # → triggers patch version bump
ci: add commitlint to PR checks            # → no version bump (hidden in changelog)
```

**Allowed scopes:** `api`, `web`, `admin`, `landing`, `mc`, `cli`, `operator`, `helm`, `terraform`, `ci`, `infra`, `deps`

**Rules:**
- Subject must be **lowercase** (enforced)
- Scopes are optional but recommended (warning if unknown scope)
- Use `!` after the type for breaking changes: `feat!:` or `fix!:`

**Validate locally:**
```bash
npx commitlint --from HEAD~5
```

### Building & Testing

```bash
# Backend
cd services/api
go vet ./...              # Lint
go test ./... -count=1    # Test (89+ tests)
go build ./cmd/server     # Build

# Frontend
cd apps/web
pnpm run build            # Build
pnpm run lint             # Lint

# All frontend (from repo root)
pnpm turbo lint           # Lint all apps
pnpm turbo build          # Build all apps
```

### Deploying to Staging (Automated)

Staging deployments are **fully automated** via Release Please. You never manually bump versions, build images, or edit Helm values.

```
1. Write code on a feature branch
2. Open PR to main → CI runs tests, security scan, commitlint
3. Merge PR to main
4. Release Please creates a "Release PR" (e.g., "chore(main): release 0.9.0")
   - Updates CHANGELOG.md with your feat/fix commits
   - Bumps version.txt, Chart.yaml version + appVersion
5. Review & merge the Release PR
6. Automatic pipeline:
   - Git tag v0.9.0 created
   - Docker images built: zenith-api:0.9.0, zenith-web:0.9.0, etc.
   - Pushed to Harbor registry
   - main merged into staging branch
   - All values-staging.yaml image tags updated
   - Staging branch pushed → ArgoCD auto-syncs
   - Smoke tests run automatically
```

**Emergency manual deploy** (if automation is broken):
```bash
# Use the manual workflow_dispatch trigger on build-images.yml
gh workflow run build-images.yml -f version=0.9.0-hotfix
```

For full pipeline documentation, see `docs/ci-cd-pipelines.md`.

---

## 15. CI/CD & Release Process

### Workflows Overview

| Workflow | File | When | What |
|----------|------|------|------|
| **CI** | `ci.yml` | PR to `main`/`staging` | Tests, security scan, commitlint |
| **Release** | `release.yml` | Push to `main` | Release Please → build images → update staging |
| **Build Images** | `build-images.yml` | Called by Release | Docker build + push + Trivy scan |
| **Smoke Tests** | `smoke-test.yml` | After Release + every 6h | E2E tests against staging API |
| **Terraform** | `terraform.yml` | PR/push to `main` | Plan/apply infrastructure changes |
| **Promote** | `promote-to-prod.yml` | Manual only | Promote staging → production |

### Release Please Flow

```
feat: commit → main → Release Please creates Release PR
                            ↓ (merge it)
                       tag v0.9.0 → build images → update staging → ArgoCD sync
```

**Key files:**
- `release-please-config.json` — Configuration (release type, changelog sections, extra files)
- `.release-please-manifest.json` — Current version (updated by Release Please)
- `version.txt` — Version file (updated by Release Please)
- `.commitlintrc.json` — Commit message validation rules

### Running CI Locally

```bash
# Commit messages
npx commitlint --from HEAD~5

# Go tests
cd services/api && go test ./... -race -count=1

# Frontend lint
pnpm turbo lint

# Helm lint
helm lint infra/helm/zenith-api --strict

# Full CI with act (requires Docker)
act pull_request -W .github/workflows/ci.yml -j test --container-architecture linux/amd64

# Release Please dry-run
release-please release-pr --repo-url=https://github.com/DoTechHQ/Zenith --token=$(gh auth token) --dry-run
```

---

## 16. Key Patterns & Conventions

### Backend Patterns

**Hexagonal Architecture** — Business logic (services) depends on interfaces (ports), not implementations (adapters). This lets us swap PostgreSQL for in-memory without changing any business logic.

**Middleware Pipeline:**
```
Request → Recover → RequestID → Security Headers → Logging → CORS → Context → [Auth] → [Rate Limit] → Handler
```

**Ownership Checks:**
Every resource access verifies the requesting user owns the resource. This prevents IDOR attacks.

**Plan Limit Checks:**
Resource creation endpoints check plan limits before proceeding.

### Frontend Patterns

**`useApi` hook** — Standard data fetching:
```tsx
const { data, loading, error, refetch } = useApi(() => api.apps.list(), []);
```

**`useAuth` hook** — Auth state:
```tsx
const { user, login, logout, isAuthenticated } = useAuth();
```

**Toast notifications** — `useToast` hook:
```tsx
const { toast } = useToast();
toast("success", "App deployed successfully");
toast("error", "Failed to create database");
```

### Conventions

- **Traefik uses IngressRoute CRD** — NOT standard Ingress resources
- **Dockerfiles build from repo root** — `docker build -f services/api/Dockerfile .`
- **Next.js uses `output: standalone`** — For Docker builds
- **Dark theme by default** — `<html className="dark">`
- **No `any` types** in TypeScript
- **Conventional commits** — `feat:`, `fix:`, `docs:`, `chore:`

### Important Rules

1. Never hardcode secrets — use environment variables
2. Every new table needs a SQL migration file
3. Every new repository needs both PostgreSQL AND memory implementations
4. Plan limits are in `entities/plan.go`, NOT hardcoded in handlers
5. ArgoCD watches the `staging` branch — always merge main → staging

---

## 17. Glossary

| Term | Meaning |
|------|---------|
| **CNPG** | CloudNativePG — PostgreSQL operator for Kubernetes |
| **APISIX** | Apache APISIX — API Gateway (routes, JWT, rate limiting) |
| **Traefik** | Ingress controller bundled with k3s (TLS, routing) |
| **IngressRoute** | Traefik's custom CRD for HTTP routing (NOT standard Ingress) |
| **Keycloak** | Identity provider for SSO, OAuth, OIDC, user management |
| **KEDA** | Kubernetes Event-Driven Autoscaler (scale-to-zero for Free tier) |
| **Cilium** | CNI plugin — network policies, WireGuard encryption, observability |
| **Kyverno** | Policy engine — validates K8s resources at admission time |
| **Falco** | Runtime security — detects anomalous container behavior |
| **Temporal** | Workflow engine — runs multi-step processes with automatic retry |
| **Velero** | Kubernetes backup tool — snapshots resources + volumes to S3 |
| **Harbor** | Container registry with Trivy vulnerability scanning |
| **ArgoCD** | GitOps tool — watches Git, syncs K8s state to match |
| **NATS** | Lightweight message queue for internal platform events |
| **Sealed Secrets** | Encrypts K8s Secrets for safe storage in Git |
| **Helm** | K8s package manager — bundles YAML into installable charts |
| **Terraform** | Infrastructure as Code — provisions cloud resources |
| **CRD** | Custom Resource Definition — extends K8s API with new types |
| **Operator** | K8s controller that manages a service's full lifecycle |
| **Day-2** | Operations after initial deployment (upgrades, backup, monitoring) |
| **IDOR** | Insecure Direct Object Reference — accessing other users' data |
| **PDB** | PodDisruptionBudget — prevents K8s from evicting too many pods |
| **WAL** | Write-Ahead Log — PostgreSQL's transaction log (used for backup) |
| **PITR** | Point-In-Time Recovery — restore database to any specific moment |
| **MRR** | Monthly Recurring Revenue — total subscription income per month |
| **Release Please** | Google's tool for automated semantic versioning via conventional commits |
| **Conventional Commits** | Commit message format (`feat:`, `fix:`) that drives automatic versioning |
| **Commitlint** | Linter that validates commit messages follow conventional commit format |
| **Lich Framework** | Project scaffolding framework in `.lich/` directory |
| **Mission Control** | Admin panel for platform operators |
| **AccountID** | Team owner's user ID, used by team members to access shared resources |

---

## 18. Your First Week

### Day 1: Understand the Product

1. Read this document completely
2. Open the staging dashboard: https://app.stage.freezenith.com
3. Log in (ask for credentials or register a test account)
4. Deploy a test app (use `nginx:alpine` as image)
5. Create a database, view logs, check monitoring

### Day 2: Understand the Architecture

1. Read `docs/v3-architecture.md` — the source of truth (2000+ lines, take your time)
2. Read `AGENTS.md` — development rules
3. SSH to staging: `ssh zen-stage`
4. Run `kubectl get pods -A` — see everything running
5. Run `kubectl get pods -n zenith-staging` — see our platform pods

### Day 3: Understand the Backend

1. Read `services/api/internal/entities/` — all domain models
2. Read `services/api/internal/ports/` — all interfaces
3. Read `services/api/cmd/server/main.go` — how it all wires together
4. Pick 3 handlers and read them:
   - `handlers/auth.go` — authentication flow
   - `handlers/apps_v2.go` — app CRUD
   - `handlers/plan.go` — plan management

### Day 4: Understand the Frontend

1. Read `apps/web/src/lib/api.ts` — the API client
2. Read `apps/web/src/hooks/useAuth.ts` — auth flow
3. Read `apps/web/src/app/apps/[name]/page.tsx` — a complex page
4. Read `apps/web/src/components/deploy-wizard.tsx` — deployment UI

### Day 5: Understand the Infrastructure

1. Read `infra/helm/zenith-api/templates/deployment.yaml` — how the API is deployed
2. Read `infra/helm/zenith-api/values-staging.yaml` — staging config
3. Open Grafana on staging and explore dashboards
4. Read a runbook: `docs/runbooks/pod-crash-loop.md`

### Day 6-7: Your First Task

Pick one of these starter tasks:
- Add a new field to an existing entity
- Fix a bug in a handler
- Add a new page to the dashboard
- Improve error messages

Follow the patterns you've seen. Run `go vet ./...` and `go build ./...` before committing.

---

## Need Help?

- **Architecture questions:** Read `docs/v3-architecture.md`
- **Code patterns:** Read `AGENTS.md` and `.lich/rules/backend.md`
- **Operational procedures:** Read `docs/runbooks/`
- **Report bugs:** https://github.com/anthropics/claude-code/issues

Welcome to the team!
