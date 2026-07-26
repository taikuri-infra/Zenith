# Zenith - Complete Project Explanation

> This document is a comprehensive walkthrough of the entire Zenith codebase as of February 2026 — every app, service, component, API endpoint, deployment script, and infrastructure config.

---

## What is Zenith?

Zenith is a **Kubernetes-native Platform-as-a-Service (PaaS)** built on Hetzner Cloud, operated by DoTech as a **managed multi-tenant enterprise cloud service**.

**Business Model:** DoTech runs the management plane. Customers choose from 4 tiers:

| Tier | Price | Infra | Isolation |
|---|---|---|---|
| **Free** | €0 | 1 pod, 1 DB, 1 S3 (limited resources) | Namespace + Cilium policies |
| **Pro** | €29/mo | 5 apps, 10 DBs, S3, custom domains | Namespace + Cilium L7 + bandwidth |
| **Team** | Custom | Dedicated space, special support | CAPI cluster on shared machines |
| **Enterprise** | Contact us | Fully dedicated infrastructure | CAPI cluster on dedicated machines |

Free/Pro share a k8s cluster (namespace isolation with Cilium). Team gets a separate CAPI-provisioned cluster on shared machines (own control plane + etcd). Enterprise gets fully dedicated Hetzner servers (kernel-level isolation).

Resources are provisioned on-demand — we guarantee the ceiling but only create what the customer actually uses.

**Future — Open-Core:** The plan is to extract a free, self-hostable open-core version of Zenith where customers install their own Mission Control and manage their own clusters. This becomes the marketing engine. The SaaS is the premium managed offering. The codebase is designed to support both modes via a `ZENITH_MODE` flag (`saas` vs `standalone`).

**Domain:** freezenith.com

---

## SaaS Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    DoTech Management Plane                       │
│                  (Hetzner server, our control)                   │
│                                                                  │
│  ┌──────────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  Zenith Admin     │  │  Zenith API  │  │  PostgreSQL      │  │
│  │  (MC rebranded    │  │  (multi-     │  │  (persistent     │  │
│  │   for SaaS)       │  │   tenant)    │  │   state)         │  │
│  │                   │  │              │  │                   │  │
│  │  - Customers      │  │  - Auth      │  │  - Users          │  │
│  │  - Plans/Billing  │  │  - Projects  │  │  - Customers      │  │
│  │  - Clusters       │  │  - Apps/DBs  │  │  - Billing        │  │
│  │  - Metering       │  │  - Admin     │  │  - Audit log      │  │
│  │  - Infra          │  │  - Metering  │  │  - Metering data  │  │
│  └──────────────────┘  └──────────────┘  └──────────────────┘  │
│                                                                  │
│  ┌──────────────────┐  ┌──────────────────────────────────────┐ │
│  │  CAPI + CAPH     │  │  Hetzner Autoscaler                  │ │
│  │  (provisions     │  │  (scales server pool based on         │ │
│  │   clusters)      │  │   total demand across customers)      │ │
│  └──────────────────┘  └──────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
          │                          │                    │
          ▼                          ▼                    ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ Customer A       │  │ Customer B       │  │ Customer C       │
│ (dedicated K8s)  │  │ (dedicated K8s)  │  │ (dedicated K8s)  │
│                  │  │                  │  │                  │
│ - Zenith Op.     │  │ - Zenith Op.     │  │ - Zenith Op.     │
│ - Web Platform   │  │ - Web Platform   │  │ - Web Platform   │
│ - Apps, DBs, S3  │  │ - Apps, DBs, S3  │  │ - Apps, DBs, S3  │
│                  │  │                  │  │                  │
│ cloud.cust-a.com │  │ cloud.cust-b.io  │  │ cloud.cust-c.dev │
└──────────────────┘  └──────────────────┘  └──────────────────┘
```

### Admin Panel vs Mission Control

| | Zenith Admin (SaaS, now) | Mission Control (Open-Core, future) |
|---|---|---|
| **Who runs it** | DoTech staff | Customer |
| **Purpose** | Manage ALL customers, billing, infra | Manage OWN clusters |
| **URL** | admin.freezenith.com | ms.{customer-domain} |
| **Codebase** | `apps/mission-control/` | Same code, `ZENITH_MODE=standalone` |
| **Sees** | All customers, all clusters | Only own clusters |

### Economics (per infrastructure pool)

| Resource | Ceiling | Hetzner Cost |
|----------|---------|-------------|
| 160 CPU cores, 320 GB RAM | 10x CPX servers | ~€400/mo |
| 20 TB S3 | Hetzner Object Storage | ~€100/mo |
| 600 GiB volumes | Hetzner Volumes | ~€33/mo |
| 4 Load Balancers | Hetzner ALB | ~€20/mo |
| **Total infra** | | **~€553/mo** |
| **Sell as Enterprise plan** | | **€2,000–4,000/mo** |
| **Margin** | | **~70–85%** |

Billing: Stripe (international) + Toman/IRR (via Fairbroker).

See [TASKS-SaaS.md](TASKS-SaaS.md) for the full implementation roadmap.

---

## Monorepo Structure

```
Zenith/
├── apps/
│   ├── landing/              # Public marketing site (freezenith.com)
│   ├── mission-control/      # Operator management plane (ms.{domain})
│   └── web/                  # User-facing dashboard (cloud.{domain})
├── packages/
│   └── ui/                   # Shared design system (@zenith/ui)
├── services/
│   ├── api/                  # Go REST API server (port 8080)
│   ├── auth/                 # Go OIDC/SAML auth service (port 8090)
│   └── operator/             # Go Kubernetes operator (controller-runtime)
├── cli/                      # zen CLI (Go + Cobra + Charm TUI)
├── infra/
│   ├── terraform/            # Cloudflare DNS as code
│   ├── ansible/              # Ansible deployment (playbooks, roles, inventory)
│   ├── helm/
│   │   ├── zenith/           # Main Helm chart
│   │   └── monitoring/       # Prometheus + Grafana chart
│   ├── k8s/                  # Raw Kubernetes manifests (currently used in prod)
│   └── scripts/              # deploy.sh, cloudflare-dns.sh, e2e-test.sh
└── docs/                     # Architecture, frontend design, phases
```

**Workspace management:** pnpm workspaces + Turborepo. Root `package.json` defines filtered scripts like `dev:mc`, `build:web`, etc.

---

## Architecture at a Glance

```
User sees: Simple UI (like Supabase/Fly.io)
Backend does: Creates a CRD in Kubernetes
Zenith Operator does: Creates Hetzner resources + service CRDs
Service Operators do: Create the actual services (PostgreSQL, Redis, etc.)

User NEVER sees: K8s, operators, PVCs, ingress, nodes
```

**Core principle:** Everything is a CRD. The flow is always: User action -> Go API creates a CRD -> Zenith Operator watches CRDs -> Operator provisions Hetzner/K8s resources.

**Two planes:**
- **Management Plane** — A single k3s server (Hetzner CX22, ~5/month) runs Mission Control + CAPI + CAPH. Mission Control IS the management plane, not just a UI.
- **Workload Clusters** — CAPI-managed clusters where tenant apps, databases, and services actually run.

**Domain convention:**
- `ms.{domain}` — Mission Control (operator management)
- `cloud.{domain}` — Web Platform (user-facing)
- Root domain — reserved for the customer

---

## 1. Landing Page (`apps/landing/`)

**URL:** https://freezenith.com
**Tech:** Next.js 15, TypeScript, Tailwind CSS, Framer Motion
**Port:** 3200 (dev), 3000 (production)

A polished marketing site with animated sections:

- **Hero:** Animated headline "Your Own Cloud Platform. 10x Cheaper.", animated terminal typing `zen install` commands, CTA buttons
- **Features:** 6 cards — Apps, Databases, Auth, Storage, API Gateway, Monitoring
- **How it Works:** 3-step guide (install CLI, deploy platform, ship apps)
- **Architecture Diagram:** Interactive component showing the system
- **Pricing Comparison:** Side-by-side vs AWS EKS ($553/mo), Fly.io ($240/mo), Railway ($180/mo), Zenith (38/mo)
- **Cost Calculator:** Interactive Hetzner infrastructure cost estimator
- **Open Source:** MIT license, tech stack badges (Go, Kubernetes, Next.js, TypeScript, Helm, Kong, Grafana, PostgreSQL)
- **CTA:** Copyable `zen install --provider hetzner --token hc_xxx` with clipboard button
- **Pricing page** (`/pricing`): Three tiers (Free, Pro Support $49/mo, Enterprise), 22-feature comparison table, FAQ accordion
- **Docs page** (`/docs`): Quick links and topic sections (placeholder — docs not yet written)

**Components:** `Header` (sticky with scroll blur), `Footer` (5-column grid), `AnimatedTerminal`, `ArchitectureDiagram`, `PricingComparison`, `CostCalculator`, `FeatureCard`, `PricingCard`, `TrustBar`, `Section`

---

## 2. Mission Control (`apps/mission-control/`)

**URL:** https://ms.embermind.app (customer), https://demo-ms.freezenith.com (demo)
**Tech:** Next.js 15, TypeScript, Tailwind CSS
**Port:** 3100
**Color accent:** Blue (#2563eb) — distinct from Web Platform's emerald

The admin dashboard for platform operators. Manages clusters, tenants, modules, infrastructure, and platform settings across the entire Zenith installation.

### Pages

| Route | What it Shows |
|-------|---------------|
| `/` | Dashboard: 4 stat cards (clusters, tenants, cost, updates), cluster table with K8s version/nodes/CPU/RAM bars, available updates list, recent audit activity |
| `/clusters` | Full cluster table: name/region, K8s version (amber warning if upgrade available), nodes, CPU/RAM progress bars, status |
| `/clusters/[name]` | Cluster detail: K8s version, nodes, pods/PVCs counts, CPU/RAM usage bars, "Upgrade Kubernetes" and "Scale Nodes" actions |
| `/modules` | Installed K8s modules/operators: name, description, installed/latest version, status; "Update All" button |
| `/tenants` | Tenant list: name, plan (pro/starter), app/database count, CPU/RAM used vs limit, status |
| `/infrastructure` | Hetzner resource overview: servers, volumes (total TB), load balancers, monthly cost; per-resource breakdown table |
| `/updates` | Available platform update card (version, features, breaking changes, "Upgrade" button); update history table |
| `/state` | Platform version, installed date, management K8s version, domain/TLS status; clusters list; module version matrix per cluster |
| `/audit` | Filterable audit log: actor, cluster, time period filters; timestamped action entries |
| `/settings` | Editable platform name/domain; read-only cloud provider/region; backup config (enabled, retention days) |
| `/login` | Email + password login; in demo mode auto-redirects to `/` |

### Auth Flow

- **Demo mode:** `useAuth()` returns a fake admin user (`admin@zenith.dev`), skips all auth checks
- **Production:** JWT tokens in `localStorage` (`mc_token`, `mc_refresh_token`). The `Shell` component redirects to `/login` if not authenticated. Login calls `api.auth.login()`, decodes the JWT payload to extract `email`, `name`, `role`.

### API Layer

Three files implement a clean switching pattern:

- **`api.ts`** — Real HTTP client. `ApiClient` class with namespaced methods: `auth`, `dashboard`, `clusters`, `tenants`, `modules`, `audit`, `updates`, `infrastructure`, `state`, `settings`. All admin routes hit `/api/v1/admin/...`. Attaches `Bearer` token from localStorage.
- **`demo-api.ts`** — Drop-in mock. Same method signatures. Returns data from `demo-data.ts` after 300ms delay. Mutations throw `"Not available in demo mode"`.
- **`get-api.ts`** — The switch. `getApi()` returns `demoApi` if `NEXT_PUBLIC_DEMO_MODE === "true"` (build-time constant), otherwise the real `api`.

### Demo Data

Hardcoded in `demo-data.ts`: 3 clusters, 5 tenants, 8 modules, 15 audit entries, infrastructure stats (19 servers, 48 volumes, 4 LBs, EUR 287.40/month), platform state v1.3.2, update available v1.4.0.

### Key Components

- **`Shell`** — Layout wrapper. Fixed sidebar + header + demo banner + main content. Handles auth redirect.
- **`Sidebar`** — Fixed left nav with 8 sections. Active state via `usePathname()`. Blue "Z" badge logo.
- **`DemoBanner`** — Emerald top banner "Demo Mode — Viewing with sample data" (only in demo).
- **`DemoButton`** — Intercepts clicks in demo mode, shows tooltip "Available in your own installation" instead of executing.
- **`StatCard`**, **`StatusBadge`**, **`ProgressBar`** — Reusable metric/status display components.
- **`LoadingSkeleton`** — Comprehensive skeleton variants for every page section.
- **`ErrorState`**, **`EmptyState`** — Error and empty state displays.

### Data Fetching Hooks

- **`useApi<T>(fetcher, deps)`** — Runs on mount + dep changes. Returns `{ data, loading, error, refetch }`.
- **`useMutation<TInput, TOutput>(mutator)`** — Not auto-run. Call `.execute(input)` manually. Returns `{ execute, loading, error }`.

---

## 3. Web Platform (`apps/web/`)

**URL:** https://cloud.embermind.app (customer), https://demo-cloud.freezenith.com (demo)
**Tech:** Next.js 15, TypeScript, Tailwind CSS
**Port:** 3000
**Color accent:** Emerald (#10b981) — the Zenith brand color

The user-facing dashboard where developers manage their apps, databases, storage, and all platform services — scoped to a project.

### Pages

| Route | Feature | Data Source |
|-------|---------|-------------|
| `/` | Project overview: 5-col stat cards (Legacy Apps, Deploy Engine running/building, Databases, Region, Status), deploy engine card grid with status dots + framework + branch, legacy apps table, databases list | API (useApi + appsDeploy) |
| `/login` | Login/register form with OAuth (Google, GitHub) | API |
| `/apps` | App list with search/filter, columns: name, status, replicas, CPU, memory, image, port, domain | API |
| `/apps/[name]` | App detail (legacy CRD): deployment info, env vars, stats | API |
| `/apps/[id]` | App detail (Phase 2): 6-tab UI — Overview (details + quick links), Deployments (table + rollback), Releases (image versions + one-click Deploy), Logs (SSE build log viewer), Secrets (add/reveal/delete encrypted secrets), Environment (add/delete/show-hide env vars) | API (appsDeploy) |
| `/deploy` | Deploy Engine dashboard: card grid with status dots (running=green, building=amber pulse, failed=red), framework labels, branch display, URLs, "Deploy from Git" modal, delete with confirmation | API (appsDeploy) |
| `/databases` | Database list with engine icons (P=Postgres blue, M=MySQL orange, M=Mongo green, R=Redis red) | API |
| `/databases/[name]` | Database detail with connection string copy, engine/storage/port/created stats | API |
| `/storage` | Object storage buckets: name, objects, size, access (private/public), region | API |
| `/monitoring` | Grafana dashboards, Prometheus targets, Loki logs (color-coded by level) | Hardcoded mock |
| `/gateway` | Kong API gateway: routes (with method colors), plugins, consumers, stats | Hardcoded mock |
| `/auth` | Auth service: realms, users (with MFA status), clients, identity providers | Hardcoded mock |
| `/iam` | Platform access: API keys (with scoped pills), team members, role cards | Hardcoded mock |
| `/registry` | Container registry: repos, tags, vulnerability scan results, pull commands | Hardcoded mock |
| `/networking` | Domains, firewalls, load balancers (placeholder) | Placeholder |
| `/planets` | Compute nodes (placeholder — "will appear once infrastructure API connected") | Placeholder |
| `/billing` | Current plan, resource usage, payment method (placeholder) | API (partial) |
| `/settings` | Project name, plan badge, region, danger zone (delete project) | API |
| `/docs` | Auto-generated from live infrastructure: apps table, database cards, env metadata | API |

### Sidebar Navigation

```
OVERVIEW       -> Overview (/)
DEPLOY         -> Deploy Engine (/deploy)      [NEW — Rocket icon]
COMPUTE        -> Apps, Databases, Storage
NETWORKING     -> Gateway, Domains
SECURITY       -> Auth, IAM
OBSERVABILITY  -> Monitoring, Registry
INFRASTRUCTURE -> Planets
Bottom         -> Docs, Billing, Settings
```

### API Layer

Same switching pattern as Mission Control:

- **`api.ts`** — Real HTTP client. Token keys: `zenith_access_token` / `zenith_refresh_token`. Includes `tryRefreshToken()` on 401 responses. Endpoints: `auth`, `projects`, `apps` (legacy CRD), `databases`, `storage`, **`appsDeploy`** (Phase 2 deploy engine: CRUD apps from Git repos, deployments list/rollback, env vars CRUD). Types: `DeployApp`, `Deployment`, `EnvVar`. Also has `connectWebSocket()` for real-time events.
- **`demo-api.ts`** — Mock client returning data from `mock-data.ts` after 300ms delay. Includes `demoAppsDeploy` with 3 mock deploy apps (my-next-app, go-api, flask-ml) and all method stubs matching real API shape.
- **`get-api.ts`** — Build-time switch based on `NEXT_PUBLIC_DEMO_MODE`. Exports `appsDeploy` alongside legacy APIs.

### Mock Data (`mock-data.ts`)

The most comprehensive mock dataset: 6 apps, 3 databases, 2 storage buckets, 3 domains, 5 planets, auth realms/users/clients, Kong gateway routes/plugins, IAM API keys/team members, registry repos with vulnerability scan data, Grafana dashboards. Used by both `demo-api.ts` and static server-rendered pages.

### Hooks

- **`useApi`** / **`useMutation`** — Same pattern as MC.
- **`useAuth`** — JWT-based auth with login/register/logout. Demo mode returns fake user.
- **`useProject`** — Returns current project ID. Demo mode returns `"demo-project"`, otherwise reads from `ProjectContext` or localStorage.
- **`useWebSocket`** — Real-time event handling (deployment_progress, log, status_change). Infrastructure ready but not yet used by pages.

---

## 4. Go API Server (`services/api/`)

**Framework:** Fiber v2
**Port:** 8080
**Module:** `github.com/dotechhq/zenith/services/api`
**URL:** https://api.freezenith.com

The central REST API backing both Mission Control and Web Platform.

### Configuration (from environment)

| Env Var | Default | Purpose |
|---------|---------|---------|
| `PORT` | 8080 | Listen port |
| `CORS_ORIGINS` | `*` | Allowed origins |
| `JWT_SECRET` | `""` | HMAC secret for JWT signing |
| `JWT_ISSUER` | `zenith` | JWT issuer claim |
| `ADMIN_EMAIL` | `""` | Seed admin email |
| `ADMIN_PASSWORD` | `""` | Seed admin password |
| `DATABASE_URL` | `""` | PostgreSQL connection string (if empty, uses in-memory) |
| `GITHUB_WEBHOOK_SECRET` | `""` | HMAC secret for GitHub webhook signature verification |
| `BASE_DOMAIN` | `freezenith.com` | Base domain for app subdomain URLs |

### Middleware Stack

1. **Panic recovery** — `recover.New()`
2. **Request ID** — `requestid.New()` injects `X-Request-Id`
3. **Access logging** — `time | status | latency | method | path`
4. **CORS** — Configurable origins, methods: GET/POST/PUT/DELETE/PATCH/OPTIONS
5. **JWT Auth** — Validates `Authorization: Bearer <token>`, HS256. Sets user_id, email, name, role in context.
6. **API Key Auth** — Reads `X-API-Key` header, assigns `RoleDeveloper` by default.
7. **Role hierarchy** — Owner(4) > Admin(3) > Developer(2) > Viewer(1)

### All Endpoints

**Public:**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (status, version, uptime) |
| GET | `/ready` | Readiness probe |
| GET | `/api/v1/version` | Version info |
| POST | `/api/v1/auth/login` | Email+password -> JWT pair (1h access, 7d refresh) |
| POST | `/api/v1/auth/register` | Create account (first user = owner, subsequent = developer) |
| POST | `/api/v1/auth/refresh` | Refresh token rotation |

**Webhook (HMAC signature, no JWT):**

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/webhooks/github` | GitHub push webhook (HMAC-SHA256 verification, triggers deployment) |

**Protected (JWT required):**

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/projects` | Create project (plans: free/pro/enterprise) |
| GET | `/api/v1/projects` | List own projects (filtered by email) |
| GET | `/api/v1/projects/:id` | Get project |
| PUT | `/api/v1/projects/:id` | Update project |
| DELETE | `/api/v1/projects/:id` | Delete project (owner role required) |
| POST | `/api/v1/projects/:id/apps` | Create app (legacy CRD-based: name, image, replicas, port, env, domain) |
| GET | `/api/v1/projects/:id/apps` | List apps (legacy) |
| GET | `/api/v1/projects/:id/apps/:name` | Get app (legacy) |
| PUT | `/api/v1/projects/:id/apps/:name` | Update app (legacy) |
| DELETE | `/api/v1/projects/:id/apps/:name` | Delete app (legacy) |
| POST | `/api/v1/projects/:id/apps/:name/redeploy` | Trigger redeploy (legacy) |
| POST | `/api/v1/apps` | Create app from Git repo (Phase 2 deploy engine) |
| GET | `/api/v1/apps` | List user's apps (Phase 2) |
| GET | `/api/v1/apps/:id` | Get app detail (Phase 2) |
| DELETE | `/api/v1/apps/:id` | Delete app (Phase 2) |
| GET | `/api/v1/apps/:id/deployments` | List deployment history |
| GET | `/api/v1/apps/:id/deployments/:did` | Get single deployment |
| POST | `/api/v1/apps/:id/rollback` | Rollback to a previous deployment |
| PUT | `/api/v1/apps/:id/env` | Set environment variables |
| GET | `/api/v1/apps/:id/env` | Get environment variables |
| DELETE | `/api/v1/apps/:id/env/:key` | Delete an environment variable |
| GET | `/api/v1/apps/:id/deployments/:did/logs` | Stream build/deploy logs via SSE (keepalive 30s, `event: done` on finish) |
| GET | `/api/v1/apps/:id/deployments/:did/logs/history` | Get stored log history as JSON snapshot |
| POST | `/api/v1/projects/:id/databases` | Create database (postgresql/mysql/mongodb/redis) |
| GET | `/api/v1/projects/:id/databases` | List databases |
| GET | `/api/v1/projects/:id/databases/:name` | Get database |
| DELETE | `/api/v1/projects/:id/databases/:name` | Delete database |
| GET | `/api/v1/projects/:id/databases/:name/backups` | List backups (stub) |
| POST | `/api/v1/projects/:id/databases/:name/backups` | Create backup (stub) |
| POST | `/api/v1/projects/:id/storage` | Create bucket (private/public-read) |
| GET | `/api/v1/projects/:id/storage` | List buckets |
| GET | `/api/v1/projects/:id/storage/:name` | Get bucket |
| DELETE | `/api/v1/projects/:id/storage/:name` | Delete bucket |

**Admin (requires admin/owner role):**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/admin/dashboard/stats` | Cluster/tenant/cost/updates stats |
| GET | `/api/v1/admin/clusters` | List CAPI clusters |
| POST | `/api/v1/admin/clusters` | Create cluster |
| GET | `/api/v1/admin/clusters/:name` | Get cluster |
| DELETE | `/api/v1/admin/clusters/:name` | Delete cluster |
| POST | `/api/v1/admin/clusters/:name/upgrade` | Upgrade K8s version |
| GET | `/api/v1/admin/tenants` | List tenants (derived from Project CRDs) |
| GET | `/api/v1/admin/tenants/:id` | Get tenant |
| POST | `/api/v1/admin/tenants/:id/suspend` | Suspend tenant |
| GET | `/api/v1/admin/modules` | List installed modules |
| POST | `/api/v1/admin/modules/update-all` | Update all modules |
| POST | `/api/v1/admin/modules/:name/install` | Install module |
| POST | `/api/v1/admin/modules/:name/uninstall` | Uninstall module |
| POST | `/api/v1/admin/modules/:name/update` | Update single module |
| GET | `/api/v1/admin/audit` | Paginated audit log (limit, offset) |
| GET | `/api/v1/admin/updates/check` | Available platform update |
| POST | `/api/v1/admin/updates/apply` | Apply update |
| GET | `/api/v1/admin/updates/history` | Update history |
| GET | `/api/v1/admin/infrastructure` | Infrastructure overview |
| GET | `/api/v1/admin/state` | Platform state |
| GET | `/api/v1/admin/state/export` | Export full state as JSON download |
| GET | `/api/v1/admin/settings` | Get platform settings |
| PUT | `/api/v1/admin/settings` | Update settings |
| PATCH | `/api/v1/admin/settings` | Partial update settings |

### Data Layer

**Legacy (CRD-based, in-memory):**
- **`k8s.MemoryClient`** — In-memory map simulating the Kubernetes API. Objects keyed by `{Kind}/{namespace}/{name}`. Thread-safe with `sync.RWMutex`. All CRD operations (CRUD) go through this interface. The real K8s client is not yet wired — data is lost on restart.
- **`capi.Client`** — Wraps K8s client. Stores CAPI Cluster resources in `zenith-system` namespace as `cluster.x-k8s.io/v1beta1` CRDs.
- **`capi.MemoryStore`** — Holds admin data that doesn't map to CRDs: settings, modules (11 pre-seeded), audit log, update history.
- **`store.UserStore`** — In-memory user store with bcrypt password hashing.

**Phase 2 Deploy Engine (Repository pattern, supports PostgreSQL):**
- **`store.AppRepository`** — Interface for Apps, Deployments, and EnvVars CRUD. Methods: `CreateApp`, `GetApp`, `ListAppsByUser`, `UpdateApp`, `DeleteApp`, `CreateDeployment`, `ListDeployments`, `GetActiveDeployment`, `UpdateDeploymentStatus`, `SetEnvVars`, `GetEnvVars`, `DeleteEnvVar`.
- **`store.MemoryAppRepository`** — In-memory implementation with full test coverage (30 tests).
- **`store.PostgresAppRepository`** — PostgreSQL implementation using `pgx/v5`. Activated when `DATABASE_URL` is set.
- **SQL Migrations** — `store/migrations/` contains `001_create_apps.sql` (apps, deployments, app_env_vars tables).
- **Models** — `models.App` (ID, UserID, Name, RepoURL, Branch, Framework, Status, Subdomain, Port), `models.Deployment` (ID, AppID, GitSHA, Status, BuildLog, DeployLog), `models.EnvVar` (Key, Value).

**Deploy Pipeline (`internal/deploy/`):**
- **`detect.go`** — Framework detection from file markers (9 frameworks: Next.js, Go, Python, Django, Flask, Rails, Express, Static, Dockerfile).
- **`dockerfile.go`** — Multi-stage Dockerfile generation with non-root users for all frameworks.
- **`git.go`** — Shallow clone, commit SHA extraction.
- **`builder.go`** — Build orchestrator: clone → detect → generate Dockerfile → prepare image tag.
- **`kaniko.go`** — Kaniko K8s Job spec generator with caching and resource limits.
- **`pipeline.go`** — Async pipeline runner with goroutine management, cancellation, and **Deployer integration** — after build completes, calls `deployer.DeployApp()` to create K8s Deployment+Service+IngressRoute. Nil-safe for tests.
- **`k8s_resources.go`** — Generates K8s Deployment (probes, resource limits), Service, and Traefik IngressRoute (TLS) manifests.
- **`deployer.go`** — K8s deployer that applies manifests via CRD client.
- **`log_hub.go`** — In-memory log broadcaster. Ring-buffer history (max 500 entries per deployment). Fan-out pub/sub to multiple SSE subscribers. Methods: `Publish`, `PublishInfo`, `PublishBuild`, `PublishDeploy`, `PublishError`, `Subscribe`, `History`, `Cleanup`.
- **`log_hub.go`•`LogSubscriber`** — Per-subscriber buffered channel (`Ch chan LogEntry`) with graceful `Close()`. Replays history on subscribe.

**End-to-End Wiring (Phase 3):**
- `main.go` constructs the full chain: Builder → Deployer → Pipeline → WebhookHandler
- Webhook handler includes `findAppsByRepo()` which scans all apps by repo URL + branch to match incoming pushes
- Full flow: `git push → GitHub webhook → findAppsByRepo() → pipeline.TriggerBuild() → Builder.BuildApp() → Deployer.DeployApp() → K8s resources created`

### Not Yet Wired

- **Backstage Catalog handler** (`handlers/backstage.go`) — Converts Zenith CRDs to Backstage catalog entities. Fully implemented but routes are not registered in `main.go`.
- **OpenTelemetry middleware** (`internal/telemetry/`) — Full OTel SDK setup with traces and metrics. Implemented but not activated in `main.go`.

---

## 5. Go Auth Service (`services/auth/`)

**Framework:** Fiber v2
**Port:** 8090
**Module:** `github.com/dotechhq/zenith/services/auth`

An embedded Keycloak-like OIDC/SAML auth service designed to give each tenant their own identity realm.

### Endpoints

- **Admin API:** Create/list/get/delete realms, create clients, manage users
- **OIDC:** `/.well-known/openid-configuration`, token endpoint, user registration — all per-realm
- **Kong integration:** Configures Kong JWT plugin per realm automatically

**Storage:** In-memory (same pattern as API service).

---

## 6. Kubernetes Operator (`services/operator/`)

**Framework:** controller-runtime (kubebuilder-style)
**Module:** `github.com/dotechhq/zenith/services/operator`

### CRD Types Defined (`api/v1alpha1/`)

| CRD | Short Name | Key Fields |
|-----|-----------|-----------|
| `App` | `app` | image, replicas (0-100), port, env[], domain, resources, healthCheck, autoScale, buildSource |
| `Database` | `db` | engine (postgresql/mysql/mongodb/redis), version, storage, replicas (1-5), backup config |
| `Project` | — | displayName, owner, plan, region |
| `StorageBucket` | — | name, region, access, versioning |
| `Domain` | — | domain name, TLS config |
| `GatewayRoute` | — | Kong route config |
| `AuthRealm` | — | Auth realm config |
| `CrossplaneResource` | — | Crossplane provider resource |

### Controllers

The operator main registers 8 reconcilers:

1. **ProjectReconciler** — Creates namespace per project, sets up RBAC
2. **AppReconciler** — Full lifecycle: creates Deployment (rolling update 25/25%), Service (ClusterIP), Ingress (nginx + Let's Encrypt), HPA if autoScale set. Updates status with phase/readyReplicas/URLs. Finalizer: `zenith.dev/app-cleanup`.
3. **DatabaseReconciler** — Provisions databases via Hetzner volumes + CNPG/Redis/MySQL operators
4. **StorageBucketReconciler** — Manages Hetzner object storage
5. **DomainReconciler** — Manages DNS records via Hetzner DNS API
6. **AuthRealmReconciler** — Provisions auth realms
7. **GatewayRouteReconciler** — Configures Kong routes/plugins
8. **GitSyncReconciler** — GitOps synchronization

**Hetzner Provider** (`internal/provider/hetzner/client.go`): Direct Hetzner Cloud API client for volumes, DNS records, object storage.

---

## 7. CLI (`cli/`)

**Language:** Go
**Framework:** Cobra (commands) + Charm TUI (Bubbletea, Lipgloss, Huh forms)

### Commands

| Command | Description |
|---------|-------------|
| `zen install` | Install Zenith Mission Control (interactive wizard or flags) |
| `zen deploy` | Deploy an app |
| `zen status` | Check platform health |
| `zen logs` | Stream app logs |
| `zen db` | Database management |
| `zen top` | Resource usage top |
| `zen apply` | GitOps apply |
| `zen diff` | GitOps diff |
| `zen export` | Export platform state |
| `zen version` | Show version |

### `zen install` (most complete)

**Interactive wizard** uses Charm's `huh` forms library for a beautiful TUI experience.

**Flags:** `--domain`, `--provider` (hetzner/existing), `--hetzner-token`, `--region` (nuremberg/falkenstein/helsinki/ashburn), `--server-type` (cx22/cx32/cx42), `--dns-provider` (cloudflare/manual), `--dns-token`, `--with-cluster`, `--ssh-host`, `--ssh-user`

**Result:** Displays a boxed success with Mission Control URL, Cloud URL, admin credentials, server IP. If manual DNS, shows required DNS records.

---

## 8. Shared UI Package (`packages/ui/`)

**Package:** `@zenith/ui`

Currently minimal — exports only the `cn()` utility (clsx + tailwind-merge). Each app has its own component copies. The plan is to extract shared components here over time.

Dependencies: `clsx`, `tailwind-merge`, `class-variance-authority`, `lucide-react`. Peer deps: React 19.

---

## 9. Kubernetes Manifests (`infra/k8s/`)

### Namespaces

- **`zenith-platform`** — Landing page, demo MC, demo Web, API server
- **`zenith-embermind`** — Customer MC + Web for embermind.app tenant

### Deployments

| Deployment | Namespace | Image | Port | Notes |
|-----------|-----------|-------|------|-------|
| `zenith-landing` | zenith-platform | zenith-landing:latest | 3000 | Marketing site |
| `zenith-api` | zenith-platform | zenith-api:latest | 8080 | Go API, reads `zenith-secrets` |
| `zenith-mc-demo` | zenith-platform | zenith-mc-demo:latest | 3100 | Demo mode baked in |
| `zenith-web-demo` | zenith-platform | zenith-web-demo:latest | 3000 | Demo mode baked in |
| `zenith-mc` | zenith-embermind | zenith-mc:latest | 3100 | Real API mode |
| `zenith-web` | zenith-embermind | zenith-web:latest | 3000 | Real API mode |

All use `imagePullPolicy: Never` (local builds imported into k3s containerd).

### Ingress (Traefik IngressRoute CRD)

| Domain | Service | Port | TLS Secret |
|--------|---------|------|------------|
| freezenith.com + www | zenith-landing | 3000 | freezenith-tls |
| api.freezenith.com | zenith-api | 8080 | freezenith-tls |
| demo-ms.freezenith.com | zenith-mc-demo | 3100 | freezenith-tls |
| demo-cloud.freezenith.com | zenith-web-demo | 3000 | freezenith-tls |
| ms.embermind.app | zenith-mc | 3100 | embermind-tls |
| cloud.embermind.app | zenith-web | 3000 | embermind-tls |

Each domain has HTTPS + HTTP-to-HTTPS redirect middleware.

### Certificates

cert-manager `Certificate` resources using `letsencrypt-prod` ClusterIssuer with HTTP-01 solver.

---

## 10. Infrastructure Pipeline (3-Phase Architecture)

Zenith uses a **3-phase infrastructure pipeline**. Everything is Infrastructure as Code — reproducible, versioned, and state-tracked.

```
Phase 1: Terraform     → Create server + DNS
Phase 2: Ansible       → Configure server (packages, Docker, k3s, Helm)
Phase 3: Terraform     → Deploy Helm charts into k8s (state-tracked)
```

### Servers

| Server | IP | Provider | Purpose | Provisioned By |
|---|---|---|---|---|
| zenith-staging | 77.42.88.149 | Hetzner (cx23, hel1) | Staging k3s cluster | Terraform Phase 1 |
| ghasi | 161.35.82.211 | DigitalOcean | Production (legacy manual deploy) | Manual |
| harbor-staging | 65.108.210.253 | Hetzner (cx23, hel1) | Harbor container/chart registry | Terraform (separate repo) |

### Phase 1: Terraform — Server + DNS (`infra/terraform/staging/`)

Creates Hetzner server with firewall + Cloudflare DNS records.

```bash
cd infra/terraform/staging
terraform apply
# Creates: server, SSH key, firewall, DNS (stage.freezenith.com, api.stage, ms.stage, cloud.stage)
```

**State:** Local file committed to git (technical debt — future: Hetzner S3 backend).

### Phase 2: Ansible — Server Configuration (`infra/ansible/`)

Installs base packages, Docker, k3s, and Helm on a fresh server.

```bash
cd infra/ansible
ansible-playbook playbooks/server-setup.yml -i inventory/staging.yml
# Installs: common packages, Docker, k3s, Helm
# Fetches kubeconfig to ~/.kube/zenith-staging.yaml
```

**Roles:** `common` (packages, Docker, Helm, sysctl) → `k3s` (k3s install + config).

### Phase 3: Terraform — K8s Resources (`infra/terraform/staging-k8s/`)

Deploys all Helm charts into k3s using `helm_release` resources. **State-tracked** — Terraform knows exactly what's installed and at what version.

```bash
cd infra/terraform/staging-k8s
terraform apply
# Installs: cert-manager, ClusterIssuer, Zenith Helm chart (all components)
```

**Module:** `infra/terraform/modules/k8s-platform/` — reusable for staging and production.

### Harbor Container & Helm Registry

**Separate repo:** `taikuri-infra/harbor-registry` (Terraform + Ansible, per-environment).

**URL:** https://registry.stage.freezenith.com

| Project | Purpose |
|---|---|
| `zenith-stage` | All CI builds go here — push, test, iterate |
| `zenith` | Verified versions only — promoted from zenith-stage |
| `fairbroker-stage` | Fairbroker staging builds |
| `fairbroker` | Fairbroker verified versions |
| `library` | Public/shared images |

**Usage:**
```bash
# Docker images
docker push registry.stage.freezenith.com/zenith-stage/zenith-api:v0.2.0

# Helm charts (OCI — no ChartMuseum needed)
helm push zenith-0.2.0.tgz oci://registry.stage.freezenith.com/zenith-stage

# Terraform pulls from Harbor OCI
resource "helm_release" "zenith" {
  repository = "oci://registry.stage.freezenith.com/zenith-stage"
  chart      = "zenith"
  version    = "0.2.0"
}
```

### DNS (Cloudflare)

Managed via Terraform. All A records with Cloudflare proxy OFF (required for cert-manager HTTP-01).

### Deployment Key Constraints

- `NEXT_PUBLIC_DEMO_MODE` is a **build-time** variable (baked into JS bundle). Demo images require separate Docker builds with `--build-arg NEXT_PUBLIC_DEMO_MODE=true`.
- HTTP redirect IngressRoutes must be applied AFTER cert-manager issues certs.
- Cloudflare proxy must be OFF for cert-manager HTTP-01 challenges.

### Legacy Deployment (`infra/scripts/deploy.sh`)

The old manual pipeline (still used on ghasi/production):
1. SSH in, `git pull`, build Docker images locally
2. Import into k3s via `docker save | k3s ctr images import -`
3. Apply raw K8s manifests from `infra/k8s/`

**Being replaced by:** Harbor + Terraform Phase 3 pipeline (see `openspec/changes/infra-pipeline-v1/`).

---

## 11. Helm Charts (`infra/helm/`)

### Main Chart (`infra/helm/zenith/`)

**Version:** 0.2.0 (in `Chart.yaml`)

Templates: namespace, API deployment, landing deployment, demo apps, PostgreSQL, secrets, certificates, ingress (Traefik IngressRoute), tenant deployments.

**Values files:**
- `values.yaml` — defaults (production-like)
- `values-staging.yaml` — staging overrides (fewer resources, staging domains)

**Image naming convention:**
```
registry.stage.freezenith.com/zenith-stage/<app>:<version>
```

| Image | Source | Port |
|---|---|---|
| `zenith-api` | `services/api/Dockerfile` | 8080 |
| `zenith-landing` | `apps/landing/Dockerfile` | 3000 |
| `zenith-mc` | `apps/mission-control/Dockerfile` | 3100 |
| `zenith-web` | `apps/web/Dockerfile` | 3000 |
| `zenith-operator` | `services/operator/Dockerfile` | — |
| `zenith-mc-demo` | MC Dockerfile + `NEXT_PUBLIC_DEMO_MODE=true` | 3100 |
| `zenith-web-demo` | Web Dockerfile + `NEXT_PUBLIC_DEMO_MODE=true` | 3000 |

### Monitoring Chart (`infra/helm/monitoring/`)

Deploys Prometheus + Grafana with pre-built dashboards, alerting rules, and ServiceMonitor CRDs. Optional — enabled via Terraform variable.

---

## 12. Demo Mode Architecture

Demo mode is a central design feature that lets anyone experience the platform without a backend.

**How it works:**
1. `NEXT_PUBLIC_DEMO_MODE=true` is set as a Docker **build arg** (compile-time, not runtime)
2. `getApi()` checks this env var and returns either the real API client or the demo API client
3. Demo API returns hardcoded mock data after a 300ms simulated delay (so skeleton states flash briefly)
4. `DemoButton` components intercept clicks and show "Available in your own installation" tooltip
5. `DemoBanner` shows at the top: "Demo Mode — Viewing with sample data"
6. Login page auto-redirects to `/` in demo mode
7. `Shell` component skips auth checks in demo mode

**Two separate image builds:**
- `zenith-mc:latest` / `zenith-web:latest` — Production images (connect to real API)
- `zenith-mc-demo:latest` / `zenith-web-demo:latest` — Demo images (self-contained mock data)

---

## 13. Complete CRD Catalog

The Zenith platform defines 25+ CRDs under `zenith.dev/v1alpha1`:

```
Core:        Project, Application, Build, Planet
Data:        Database, ObjectStore, KeyValueStore, BackupPolicy
Networking:  Domain, Firewall, Network, FloatingIP, LoadBalancer, VPNPeer,
             DNSZone, DNSRecord, Gateway, CloudConnector
Platform:    Registry, AuthRealm, Monitoring, LogPipeline, AlertRule
Compute:     CronTask
Billing:     UsageRecord, Invoice (internal)
```

Each CRD follows the same lifecycle: User action -> API creates CRD -> Operator watches CRD -> Operator provisions infrastructure.

---

## 14. Live Endpoints

### Production (ghasi — legacy manual deploy)

| URL | Service | Description |
|-----|---------|-------------|
| https://freezenith.com | zenith-landing | Public marketing site |
| https://demo-ms.freezenith.com | zenith-mc-demo | Demo Mission Control (mock data) |
| https://demo-cloud.freezenith.com | zenith-web-demo | Demo Web Platform (mock data) |
| https://api.freezenith.com | zenith-api | Go REST API |
| https://ms.embermind.app | zenith-mc | Customer Mission Control |
| https://cloud.embermind.app | zenith-web | Customer Web Platform |

### Staging (77.42.88.149 — Terraform + Ansible pipeline)

| URL | Service | Status |
|-----|---------|--------|
| https://stage.freezenith.com | zenith-landing | DNS ready, app not yet deployed |
| https://api.stage.freezenith.com | zenith-api | DNS ready, app not yet deployed |
| https://ms.stage.freezenith.com | zenith-mc | DNS ready, app not yet deployed |
| https://cloud.stage.freezenith.com | zenith-web | DNS ready, app not yet deployed |

### Infrastructure

| URL | Service | Status |
|-----|---------|--------|
| https://registry.stage.freezenith.com | Harbor registry | Running |

---

## 15. Current Development Status (Feb 2026)

### Complete and Live
- Landing page with full marketing content
- Mission Control UI (10 pages, demo + customer deployments)
- Web Platform UI (18 pages, demo + customer deployments)
- Go API server (all CRUD endpoints + admin endpoints + deploy engine endpoints, JWT auth)
- Go Auth service (OIDC endpoints, realm management)
- Go Kubernetes Operator (CRD types defined, 8 controllers implemented)
- CLI `zen install` (interactive wizard + flags)
- K8s manifests, TLS certificates, Traefik ingress, full deployment pipeline
- Terraform DNS config
- Helm chart templates
- ✅ **App Deploy Engine (Phase 2 + Phase 3 wiring)** — Complete end-to-end:
  - Database schema + SQL migrations for apps, deployments, env vars
  - AppRepository (in-memory + PostgreSQL implementations)
  - Framework detection (9 frameworks) + Dockerfile auto-generation
  - GitHub webhook with HMAC-SHA256 signature verification + `findAppsByRepo()` matching
  - Build pipeline (clone → detect → Dockerfile → Kaniko build → K8s deploy)
  - **Pipeline→Deployer integration** — after build, calls `DeployApp()` for K8s resources
  - **Full chain wired in `main.go`** — Builder → Deployer → Pipeline → WebhookHandler
  - K8s resource generation (Deployment + Service + IngressRoute with TLS)
  - API endpoints: 11 new routes for apps, deployments, rollback, env vars
  - Frontend: `/deploy` page (card grid + deploy-from-git modal), app detail page (3 tabs), dashboard overview with Deploy Engine stats
  - Deploy Engine added to sidebar navigation (Rocket icon)
  - Demo API includes `demoAppsDeploy` with 3 mock apps
  - 89 unit tests across all deploy engine components

- ✅ **Phase 3: Build Log Streaming** (2026-02-21)
  - SSE routes registered: `GET /apps/:id/deployments/:did/logs` + `/logs/history`
  - Fixed fasthttp `bufio.Writer` API bug in `logs.go`
  - Frontend: `useDeployLogs` hook + `BuildLogViewer` terminal component + Logs tab in app detail page

- ✅ **Phase 4: Kaniko Build Execution** (2026-02-21)
  - `k8s.Client` interface extended with `JobObject` + 4 Job methods (`CreateJob`, `GetJob`, `DeleteJob`, `GetPodLogs`)
  - `KanikoRunner` submits K8s Job, polls completion, streams pod logs → LogHub, cleans up
  - `Builder` wired to `KanikoRunner` — nil-safe dev mode fallthrough
  - `REGISTRY` + `BUILD_WORKDIR` env vars added to config

### In-Memory / Development Mode
- Legacy CRD API data layer uses `MemoryClient` — all state lost on restart
- Phase 2 deploy engine supports both in-memory and PostgreSQL (via `DATABASE_URL`)
- `MemoryClient` for K8s now supports Jobs (immediately marks as Succeeded, emits fake build logs)
- User store is in-memory (no persistent database)

### Not Yet Wired
- Several Web Platform pages use hardcoded mock data (monitoring, gateway, auth, IAM, registry)
- `zenith-actions` GitHub Action (not yet published)

### Remaining Major Work

#### Infrastructure Pipeline (in progress — see `openspec/changes/infra-pipeline-v1/`)
| Step | Status | Description |
|------|--------|-------------|
| Docker images → Harbor | TODO | GitHub Action to build + push 7 images to Harbor |
| imagePullSecret in k8s | TODO | Terraform creates secret so k3s can pull from Harbor |
| Helm chart → Harbor | TODO | GitHub Action to package + push chart as OCI artifact |
| Terraform Phase 3 from Harbor | TODO | `helm_release` pulls from Harbor OCI instead of local |
| Terraform CI | TODO | Plan on PR, apply on merge |
| Version tagging strategy | TODO | Semver for chart + images, git tags |
| Local CI with `act` | TODO | Makefile + `.secrets` for running Actions locally |

#### Infrastructure Tooling (decided, partially built)
| Component | Tool | Status |
|-----------|------|--------|
| Container & Helm registry | **Harbor** | ✅ Running at `registry.stage.freezenith.com` |
| CNI + Security | **Cilium** | Decided, not installed |
| API Gateway | **Kong** (DB-less KIC) | Decided, not installed |
| PostgreSQL operator | **CloudNativePG** | Decided, not installed |
| IAM / OIDC | **Keycloak** | Decided, not installed |
| Monitoring | **kube-prometheus-stack** | Decided, Helm chart exists |
| Secrets | **Postgres + AES-256-GCM** | Decided, not built |
| Cluster provisioning | **CAPI + CAPH** | Decided, not installed (Team/Enterprise tier) |

#### Customer App Deploy Flow (designed, not built)
- `zenith-actions` GitHub Action — builds image, tags with git SHA, pushes to customer's own registry (GHCR/ECR/DockerHub), POSTs release to Zenith API
- `releases` table — stores image URL, SHA, branch, commit message per app
- Panel Deployments tab — version list, one-click deploy, live SSE logs during rollout
- Customer never writes Helm/K8s YAML — Zenith generates `HelmRelease` on first app creation
- Privacy model: customer images never touch Zenith infrastructure

#### Other Remaining Work
- Auth service integration with Web/MC login flows
- Custom domain management with automatic TLS
- Full CLI command implementation (`deploy`, `logs`, `status`, `db`)
- Production Helm chart deployment (currently raw manifests)
- Operator reconciliation with real Hetzner resources

---

## Tech Stack Summary

| Component | Technology |
|-----------|-----------|
| Frontend Apps | Next.js 15, TypeScript, Tailwind CSS, React 19 |
| Landing Animations | Framer Motion |
| Backend API | Go 1.25, Fiber v2, JWT (golang-jwt/v5) |
| Auth Service | Go, Fiber v2, OIDC, bcrypt |
| Kubernetes Operator | Go, controller-runtime, kubebuilder CRDs |
| CLI | Go, Cobra, Charm TUI (Bubbletea, Lipgloss, Huh) |
| Infrastructure | Hetzner Cloud (k3s, CX23 server) |
| Infra as Code | Terraform (state-tracked) + Ansible (server config) |
| Cluster Provisioning | CAPI + CAPH (Team/Enterprise tiers) |
| CI/CD | GitHub Actions (runnable locally via `act`) |
| CNI / Network Security | **Cilium** (eBPF, ClusterMesh, mTLS via WireGuard) |
| Ingress | Traefik 3.5.1 (IngressRoute CRD) |
| API Gateway | **Kong** DB-less KIC |
| TLS | cert-manager, Let's Encrypt (HTTP-01) |
| DNS | Cloudflare (Terraform + bash script) |
| Container & Helm Registry | **Harbor** (OCI — Zenith infra only) |
| Customer App Registry | Customer-owned (GHCR / DockerHub / ECR) |
| Image Build | Kaniko (in-cluster, on customer cluster) |
| CI Integration | `zenith-actions` GitHub Action |
| Database Operator | **CloudNativePG** (per-tenant Postgres) |
| IAM / OIDC | **Keycloak** (per-tenant realm) |
| Monitoring | kube-prometheus-stack (Prometheus + Grafana) |
| Secrets | Postgres `app_secrets` table + AES-256-GCM (no Sealed Secrets — fewer moving parts) |
| Build System | pnpm workspaces + Turborepo |

---

## Backend Architecture Refactoring Plan

### Current State (Violations against Lich/Clean Architecture)

The backend (`services/api/`) was built pragmatically using a flat handler → store pattern. While functional, it violates the Clean/Hexagonal Architecture defined in `.lich/rules/backend.md`:

#### Current structure

```
services/api/internal/
├── config/      ← ✅ OK (pkg/config equivalent)
├── models/      ← ❌ mixes domain entities WITH DTOs (json tags everywhere)
├── handlers/    ← ❌ contains business logic (no services layer)
├── store/       ← ❌ mixes ports (interfaces.go) with adapters (postgres_*.go, memory_*.go)
├── middleware/  ← ✅ OK
├── deploy/      ← ❌ mixes domain logic with K8s infrastructure
├── telemetry/   ← ✅ OK
├── k8s/         ← ✅ OK (infra adapter, internal Go package)
├── capi/        ← ✅ OK (infra adapter)
├── cluster/     ← ✅ OK (infra adapter)
```

#### 8 Violations Identified

| # | Violation | Current | Lich Requirement |
|---|-----------|---------|------------------|
| 1 | **No entities layer** | `models/` has structs with `json:` tags | Pure domain structs, no serialization tags |
| 2 | **No services layer** | `handlers/` contains business logic + calls store directly | `services/` with injected ports |
| 3 | **Ports mixed with adapters** | `store/interfaces.go` + `store/postgres_*.go` in same package | `ports/` (interfaces only) vs `adapters/` (implementations) |
| 4 | **No DTO package** | Request/response structs inline in `handlers/` (`loginRequest`, `tokenResponse`) | Separate `dto/` package |
| 5 | **No validators** | Validation inline in handler methods | Separate `validators/` package |
| 6 | **Handlers → Store coupling** | `AuthHandler` directly imports `store.UserRepository` | Handlers should import services, not stores |
| 7 | **Models = entities + DTOs** | `CreatePlanInput`, `UpdateCustomerInput` live alongside `Customer`, `Plan` | Input DTOs should be in `dto/`, entities should be pure |
| 8 | **Deploy mixes concerns** | `deploy/pipeline.go` has both domain logic and K8s infra calls | Separate domain events from infra adapters |

#### Dependency flow (current vs target)

```
CURRENT:   handlers → store (direct) → models
TARGET:    handlers → services → ports ← adapters
                        ↓
                     entities
```

### Target Structure

```
services/api/internal/
├── entities/            # Pure domain models (NO json tags, NO external deps)
│   ├── user.go          # User, Role, APIKey
│   ├── customer.go      # Customer, Plan
│   ├── app.go           # App, Deployment, EnvVar, Secret, Release
│   ├── metering.go      # ResourceUsage
│   └── admin.go         # PlatformSettings, Module, AuditEntry
├── services/            # Use cases (business logic, inject ports)
│   ├── auth_service.go  # Login, Register, Refresh (currently in handlers/auth.go)
│   ├── customer_service.go
│   ├── app_service.go
│   ├── deploy_service.go
│   ├── metering_service.go
│   └── admin_service.go
├── ports/               # Interfaces ONLY (extracted from store/interfaces.go)
│   ├── user_port.go
│   ├── customer_port.go
│   ├── app_port.go
│   ├── metering_port.go
│   └── admin_port.go
├── adapters/            # Implementations (moved from store/)
│   ├── postgres/
│   │   ├── user_repo.go
│   │   ├── customer_repo.go
│   │   ├── app_repo.go
│   │   ├── metering_repo.go
│   │   └── admin_repo.go
│   └── memory/
│       ├── user_repo.go
│       ├── customer_repo.go
│       ├── app_repo.go
│       └── metering_repo.go
├── dto/                 # Request/Response shapes (extracted from handlers + models)
│   ├── auth.go          # loginRequest, registerRequest, tokenResponse
│   ├── customer.go      # CreateCustomerInput, UpdateCustomerInput
│   ├── app.go           # CreateAppInput, UpdateAppInput
│   └── admin.go         # PlatformSettings input/output shapes
├── validators/          # Input validation (extracted from inline handler checks)
│   └── validators.go    # ValidateEmail, ValidatePassword, etc.
├── handlers/            # HTTP layer (thin — parse, validate, call service, respond)
├── deploy/              # Keep as domain module, extract K8s to adapters/
├── config/
├── middleware/
├── telemetry/
├── k8s/
├── capi/
└── cluster/
```

### Migration Strategy (Phased, Non-breaking)

#### Phase A — Foundation: Create `entities/` and `dto/` (LOW RISK)

1. Create `internal/entities/` with pure domain structs (copy from `models/`, strip `json:` tags)
2. Create `internal/dto/` with all input/output shapes (move `Create*Input`, `Update*Input` from `models/`)
3. Keep `models/` as a thin adapter that re-exports for backward compat
4. Zero behavior change — just structural separation

#### Phase B — Extract `ports/` and `adapters/` (LOW RISK)

1. Create `internal/ports/` — move interfaces from `store/interfaces.go`
2. Create `internal/adapters/postgres/` — move `store/postgres_*.go`
3. Create `internal/adapters/memory/` — move `store/memory_*.go`
4. Keep `store/` package as a re-export wrapper during transition
5. Update `main.go` imports gradually

#### Phase C — Introduce `services/` (MEDIUM RISK)

1. Create `internal/services/auth_service.go` — extract business logic from `handlers/auth.go`
2. Create `internal/services/customer_service.go` — extract from `handlers/customer.go`
3. Repeat for each domain area
4. Handlers become thin: parse → validate → call service → respond
5. **This is the biggest change** — requires careful method-by-method extraction

#### Phase D — Validators + Cleanup (LOW RISK)

1. Create `internal/validators/` — extract inline validation from handlers
2. Remove backward-compat wrappers (`models/`, `store/`)
3. Final dependency audit: ensure no layer violations remain

### Key Decisions

- **Go-native approach**: The Lich framework was designed for Python (dataclasses, Pydantic). In Go, we adapt the spirit:
  - `entities/` = plain Go structs (no tags) with domain methods
  - `dto/` = structs with `json:` tags for API serialization
  - `ports/` = Go interfaces (not Python Protocol)
  - `adapters/` = interface implementations
- **Keep `deploy/` as a semi-autonomous module**: It's a bounded context with its own domain (Pipeline, Builder, Deployer). Extract only K8s-specific code to `adapters/`.
- **Phased migration**: Each phase is independently deployable. No big-bang rewrite.

### Priority

This refactoring is **important but not urgent**. The current code works and is well-tested. The refactoring should happen when:
- Adding new major features (refactor the touched domain area first)
- Onboarding new developers (clear architecture makes onboarding faster)
- Before scaling the team (prevents architectural drift)

Estimated effort: ~2–3 days for Phases A–D (incremental, one domain area at a time).

