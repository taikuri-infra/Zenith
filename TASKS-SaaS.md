# Zenith — Open-Core PaaS on Hetzner

> **SaaS-first, open-source second.**
> Build the full SaaS product on freezenith.com first (sign up → deploy app → database → billing).
> Then extract the open-source self-hosted version by stripping multi-tenant, billing, KEDA.
> Revenue comes from SaaS. Open-source comes from SaaS extraction → drives community → more SaaS customers.

---

## What Zenith Is (For Sales / B2B Pitch)

**Zenith: Your own cloud. Secure. Automated. No maintenance needed. Full support. 70% cheaper than AWS.**

### What's Built and Working TODAY

| Feature | Status | Live Demo |
|---------|--------|-----------|
| **Admin Dashboard** — manage customers, plans, clusters, usage | ✅ Live | [demo-ms.freezenith.com](https://demo-ms.freezenith.com) |
| **Customer Management** — create, edit, suspend, delete customers | ✅ Live | |
| **Plan Management** — Starter/Pro/Enterprise with resource ceilings | ✅ Live | |
| **Resource Metering** — CPU, RAM, Storage, DB usage tracking per customer | ✅ Live | |
| **Usage Dashboards** — visual gauges, history, platform-wide aggregates | ✅ Live | |
| **Cluster Lifecycle** — provision, scale, upgrade, delete K8s clusters | ✅ Live | |
| **JWT Auth System** — login, registration, role-based access control | ✅ Live | |
| **Go API Server** — RESTful, PostgreSQL-backed, auto-migration | ✅ Live | [api.freezenith.com/health](https://api.freezenith.com/health) |
| **PostgreSQL** — persistent state, all data survives restarts | ✅ Live | |
| **Auto-Deploy Pipeline** — git push → build 6 images → deploy to k3s | ✅ Live | |
| **TLS/SSL** — automatic Let's Encrypt certificates for all domains | ✅ Live | |
| **Landing Page** | ✅ Live | [freezenith.com](https://freezenith.com) |

### What's Coming Next

| Feature | ETA | Impact |
|---------|-----|--------|
| **Git Push Deploy** — connect GitHub, push code, app goes live | Building | Core product |
| **Built-in Database** — one-click PostgreSQL per app | Phase 3 | Supabase-like |
| **Built-in Auth** — SDK for sign up/login in user's app | Phase 3 | Supabase-like |
| **Free Tier** — apps sleep when idle, wake on request (KEDA) | Phase 4 | Growth engine |
| **Billing (Stripe)** — self-service upgrade/downgrade | Phase 6 | Revenue |
| **Open-Source Version** — docker-compose, self-host for free | Phase 7 | Community / marketing |

### Pricing (SaaS)

**Every plan gets the FULL stack.** Plans differ by resources and limits, NOT by features.

| | **Free** | **Pro €29/mo** | **Team €199/mo** | **Enterprise (custom)** |
|---|---|---|---|---|
| **Apps (frontend + backend)** | 1 | 5 | 20 | Unlimited |
| **PostgreSQL databases** | 1 × 500MB | 3 × 5GB | 10 × 20GB | Unlimited |
| **Built-in Auth** | ✅ 1K users | ✅ 10K users | ✅ 100K users | Unlimited |
| **S3 Storage** | 1 GB | 10 GB | 100 GB | Custom |
| **Container Registry** | ✅ 500MB | ✅ 5GB | ✅ 50GB | Unlimited |
| **Monitoring & Logs** | ✅ 1 day | ✅ 7 days | ✅ 30 days | 90 days |
| **Team Members** | 1 | 3 | 10 | Unlimited |
| **Custom Domain** | ❌ subdomain only | ✅ | ✅ | ✅ |
| **Backups** | ❌ | Daily | Hourly | Continuous |
| **CPU / RAM per app** | 0.5 / 512MB | 2 / 2GB | 4 / 4GB | Custom |
| **Sleep mode** | After 15 min | ❌ always-on | ❌ always-on | ❌ always-on |
| **SSL/TLS** | ✅ | ✅ | ✅ | ✅ |
| **SSO (SAML/OIDC)** | ❌ | ❌ | ✅ | ✅ |
| **Audit Log** | ❌ | ❌ | ✅ | ✅ |
| **SLA** | ❌ | ❌ | ❌ | ✅ 99.9% |
| **Dedicated Infrastructure** | ❌ | ❌ | ❌ | ✅ |
| **Priority Support** | ❌ | Email | Email | Slack + Phone |

> **Key insight:** Free user gets backend + frontend + DB + auth + S3 + registry + monitoring.
> The only things gated to Enterprise: SLA, dedicated infra, priority support.
> SSO and audit log gated to Team+.

### For B2B Sales — What You Can Sell Today

**Option A: Managed DevOps / Infrastructure Setup (one-time)**
> "We set up your cloud infrastructure in Europe, configure CI/CD, monitoring, and SSL. 70% cheaper than AWS."
> **Price: €2,500–5,000 per project**

**Option B: Managed Hosting (monthly)**
> "We host and manage your applications. Auto-deploy from GitHub, PostgreSQL, monitoring, backups. You just push code."
> **Price: €500–2,000/month depending on resources**

**Option C: White-label Cloud Platform (enterprise)**
> "Your own cloud platform under your brand. Dashboard at cloud.yourdomain.com. Manage your clients' infrastructure."
> **Price: €2,000–5,000/month**

### Key Selling Points
- **70% cheaper than AWS** — European cloud infrastructure, same reliability
- **European data residency** — GDPR-compliant, data stays in EU data centers
- **Governance & compliance** — audit logs, role-based access, resource metering built in
- **Dedicated resources** — not shared hosting, each customer gets isolated infrastructure
- **Zero maintenance** — no servers to manage, no Kubernetes to learn, just push code
- **Beautiful dashboard** — professional admin panel, not a terminal
- **Open-source core** — no vendor lock-in, transparent

---

## Progress Summary

| Phase | Description | Tasks | Done | Status |
|-------|-------------|-------|------|--------|
| Pre | Foundation (Auth, API scaffold, Deploy, IaC) | 24 | 24 | **COMPLETE** |
| 0 | PostgreSQL + Persistent State | 18 | 15 | **COMPLETE** (remaining deferred) |
| 1 | Customer/Plan Management (Admin) | 16 | 16 | **COMPLETE** |
| 1.5 | Resource Metering (Admin) | 11 | 7 | **COMPLETE** (remaining deferred) |
| 2 | App Deploy Engine (git push → live) | 14 | 0 | **NEXT** |
| 3 | Built-in Services (DB, Auth, S3) | 10 | 0 | NOT STARTED |
| 4 | KEDA Scale-to-Zero + SaaS Free Tier | 11 | 0 | NOT STARTED |
| 5 | Hetzner Autoscaler (up to 10 servers) | 8 | 0 | NOT STARTED |
| 6 | Billing (Stripe) | 9 | 0 | NOT STARTED |
| 7 | Open-Source Extraction | 12 | 0 | NOT STARTED |
| 8 | Launch & Marketing | 8 | 0 | NOT STARTED |
| **Total** | | **141** | **62** | **44%** |

---

## Business Model

**Open-Core PaaS.** Developer tries free → gets hooked → company pays.

### Open-Source (Self-Hosted)

```bash
git clone https://github.com/taikuri-infra/Zenith
cd Zenith
docker compose up
# → localhost:3100 = Dashboard
# → localhost:8080 = API
# → localhost:5432 = PostgreSQL
```

Developer gets: app deploy, database, auth, dashboard. Runs on their own server.
No scaling, no backups, no metrics, no audit log. Simple and clean.

### SaaS (freezenith.com)

| Plan | Price | What You Get |
|------|-------|-------------|
| **Free** | €0 | 1 app, 1 DB (500MB), 512MB RAM, 0.5 CPU, subdomain, **sleep after 15min** |
| **Pro** | €29/mah | 3 apps, 2 DB (5GB), 2GB RAM, 2 CPU, custom domain, always-on, daily backup |
| **Team** | €199/mah | 10 apps, 5 DB (20GB), 4GB RAM, 4 CPU, SSO, monitoring, team mgmt, hourly backup |
| **Managed** | €990+/mah | Unlimited, dedicated resources, SLA 99.9%, Grafana, priority support |

### Unit Economics

```
Free user:    sleep mode → ~€0.70/mah cost (KEDA scale-to-zero)
Pro user:     always-on → ~€12/mah cost → €17/mah margin (59%)
Team user:    always-on → ~€30/mah cost → €169/mah margin (85%)
Managed:      dedicated → ~€200/mah cost → €790+/mah margin (80%)

Break even: 16 Pro users = €464 revenue vs €450 infra cost
Target:     200 free + 30 paid = €870 revenue, €450 cost, €420 profit
```

### Infrastructure Budget

```
10 x CPX52 (12 vCPU, 24GB RAM, 480GB SSD)  = €285/mah
S3 storage (10TB) + Volumes (600GB)          = €52/mah
Load Balancer + management                   = €30/mah
DNS, domain, tools                           = €83/mah
═══════════════════════════════════════════════════════
Total: ~€450/mah

Capacity: 120 vCPU, 240GB RAM
  → ~1,000 free users (with sleep mode)
  → ~50 paid users (always-on)
  → Hetzner autoscaler adds servers as needed (cap: 10)
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Zenith SaaS Platform                     │
│                   (Hetzner K8s Cluster)                       │
│                                                               │
│  ┌─────────────┐  ┌────────────┐  ┌─────────────────────┐   │
│  │  Dashboard   │  │  Zenith    │  │  PostgreSQL          │   │
│  │  (Next.js)   │  │  API (Go)  │  │  (platform + user)   │   │
│  │              │  │            │  │                       │   │
│  │  - Apps      │  │  - Auth    │  │  - Users              │   │
│  │  - Databases │  │  - Deploy  │  │  - Apps               │   │
│  │  - Usage     │  │  - Git     │  │  - Databases          │   │
│  │  - Settings  │  │  - Build   │  │  - Metering           │   │
│  │  - Billing   │  │  - Scale   │  │  - Billing            │   │
│  └─────────────┘  └────────────┘  └─────────────────────┘   │
│                                                               │
│  ┌─────────────┐  ┌────────────┐  ┌─────────────────────┐   │
│  │  Traefik     │  │  KEDA +    │  │  Hetzner Autoscaler  │   │
│  │  (ingress)   │  │  HTTP      │  │  (scale 1→10 servers) │   │
│  │              │  │  Add-on    │  │                       │   │
│  │  *.freeze    │  │            │  │  CPX52 pool           │   │
│  │  zenith.com  │  │  sleep/    │  │  auto scale-up >80%   │   │
│  │              │  │  wake      │  │  auto scale-down <40% │   │
│  └─────────────┘  └────────────┘  └─────────────────────┘   │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐   │
│  │  User Apps (namespaced per user)                       │   │
│  │                                                         │   │
│  │  [App A: running]  [App B: sleeping]  [App C: running] │   │
│  │  [App D: sleeping] [App E: sleeping]  [App F: running] │   │
│  └───────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘

Open-Source (self-hosted) = same stack, single docker-compose
  No KEDA, no autoscaler, no billing, no multi-user
  Just: API + Dashboard + PostgreSQL + Traefik
```

### User Experience

```
Developer signs up → freezenith.com

1. Connect GitHub repo
2. Zenith detects: Next.js / Django / Go / Rails / ...
3. Auto-build (Dockerfile or buildpack)
4. Auto-deploy to K8s
5. Live at: my-app.freezenith.com
6. Database tab → "Create PostgreSQL" → connection string
7. Auth tab → "Enable auth" → SDK snippet
8. git push origin main → auto-redeploy

Free: sleeps after 15 min, wakes in 2-5 sec
Pro:  always-on, custom domain
```

### Open-Source vs SaaS Features

| Feature | Open-Source (docker-compose) | SaaS |
|---------|---------------------------|------|
| App deploy (git push) | ✅ | ✅ |
| PostgreSQL per app | ✅ | ✅ |
| Built-in auth | ✅ | ✅ |
| S3 storage | ✅ | ✅ |
| Dashboard | ✅ | ✅ |
| Auto-TLS (SSL) | ❌ (manual) | ✅ |
| Custom domain | ✅ (your server) | Pro+ |
| Auto-scaling | ❌ | ✅ (KEDA) |
| Backups | ❌ | Pro+ |
| Monitoring / Metrics | ❌ | Team+ |
| Audit log | ❌ | Team+ |
| Team management | ❌ | Team+ |
| SSO | ❌ | Team+ |
| SLA | ❌ | Managed |
| Multi-user admin | ❌ | ✅ |
| Billing | ❌ | ✅ |
| Sleep mode (scale-to-zero) | ❌ | Free tier |

---

## Completed Work

### Pre-SaaS Foundation (24/24) — COMPLETE

#### API Server (`services/api/`)
- [x] **PRE-01** Go API server with Fiber framework, structured routes, error handling
- [x] **PRE-02** JWT authentication: login, register, refresh endpoints (`handlers/auth.go`)
- [x] **PRE-03** User store with bcrypt password hashing (`store/user_store.go`, in-memory)
- [x] **PRE-04** Auth middleware: JWT validation, API key header, role-based access (Owner/Admin/Developer/Viewer)
- [x] **PRE-05** CRD-based resource architecture: Projects, Apps, Databases, Storage (in-memory `k8s.MemoryClient`)
- [x] **PRE-06** CAPI client wrapper for cluster CRUD operations (in-memory `capi.MemoryStore`)
- [x] **PRE-07** Admin handlers: dashboard stats, clusters, tenants, modules, audit, settings, infra, state
- [x] **PRE-08** Config from env vars: PORT, JWT_SECRET, ADMIN_EMAIL/PASSWORD, CORS_ORIGINS, etc.
- [x] **PRE-09** Dockerfile with multi-stage build, non-root user, port 8080

#### Mission Control (`apps/mission-control/`)
- [x] **PRE-10** Login page (`/login`) with email/password form, error handling, loading states
- [x] **PRE-11** `useAuth()` hook: JWT token parsing, localStorage persistence, demo mode bypass
- [x] **PRE-12** API client with auth methods: login, logout, refresh, token management (`api.ts`)
- [x] **PRE-13** Protected shell: auth gating, redirect to `/login` if not authenticated
- [x] **PRE-14** Demo mode: `NEXT_PUBLIC_DEMO_MODE=true` build-time flag, `demoApi` with mock data
- [x] **PRE-15** Full page set: Dashboard, Clusters, Tenants, Modules, Updates, Infrastructure, State, Audit, Settings

#### Infrastructure & IaC
- [x] **PRE-16** K8s manifests (`k8s/*.yaml`): namespaces, deployments, services, certificates, IngressRoutes
- [x] **PRE-17** `scripts/deploy.sh`: Full pipeline — git pull, build 6 images, import to k3s, apply manifests, rollout
- [x] **PRE-18** Terraform DNS (`terraform/`): Cloudflare provider, 7 A records (freezenith.com + embermind.app)
- [x] **PRE-19** `scripts/cloudflare-dns.sh`: Quick DNS CRUD via Cloudflare API (create/delete/status)
- [x] **PRE-20** `scripts/e2e-test.sh`: Post-deploy validation (DNS, HTTPS, redirects, SSL, content, API health)
- [x] **PRE-21** Helm chart `helm/zenith/`: API + Operator + Auth + Kong + OTEL + RBAC + service mesh templates
- [x] **PRE-22** Helm chart `helm/monitoring/`: kube-prometheus-stack + Loki + Promtail + alerting rules
- [x] **PRE-23** cert-manager with letsencrypt-prod ClusterIssuer, HTTP-01 solver
- [x] **PRE-24** Traefik 3.5.1 IngressRoutes with HTTP→HTTPS redirect middleware

### Phase 0: PostgreSQL + Persistent State (15/18) — COMPLETE (remaining deferred)

- [x] **S0-01** K8s StatefulSet for PostgreSQL with PVC
- [x] **S0-02** Deploy script updated for PostgreSQL
- [x] **S0-03** DB env vars wired into API deployment
- [x] **S0-04** golang-migrate with embedded SQL files
- [x] **S0-05** Migration 001: users, platform_settings, modules, audit_log, update_history
- [x] **S0-06** Migration 003: customers table
- [x] **S0-07** Migration 003: plans table
- [x] **S0-09** Migration 005: resource_usage table
- [x] **S0-11** Migration 001 includes audit_log
- [x] **S0-12** Migration 002: Seed default modules + settings
- [x] **S0-13** pgx/v5 driver + pgxpool, conditional startup
- [x] **S0-14** PostgresUserRepository
- [x] **S0-16** PostgresAdminRepository
- [x] **S0-17** API auto-migrates on startup
- [x] **S0-18** SQL files embedded via go:embed
- [ ] ~~S0-08~~ clusters migration — deferred (new app model replaces dedicated clusters)
- [ ] ~~S0-10~~ invoices migration — deferred to Phase 7
- [ ] ~~S0-15~~ real K8s client — deferred to Phase 3

### Phase 1: Customer/Plan Management (16/16) — COMPLETE

- [x] **S1-01** through **S1-10**: Full customer + plan CRUD API
- [x] **S1-11** through **S1-16**: MC frontend pages (customers, plans, detail, dashboard)

### Phase 1.5: Resource Metering (7/11) — COMPLETE (remaining deferred)

- [x] **M-01** Internal metering endpoint with X-Internal-Secret auth
- [x] **M-02** resource_usage table, MeteringRepository (Memory + Postgres)
- [x] **M-03** Customer usage endpoint (current vs ceiling with percentages)
- [x] **M-04** Usage history endpoint (daily aggregation)
- [x] **M-05** MC customer detail: resource usage gauges
- [x] **M-06** MC customer detail: usage history table
- [x] **M-07** MC dashboard: aggregate platform usage
- [ ] ~~S3-01~~ Metering agent — deferred (will be built into app runtime)
- [ ] ~~S3-06~~ Ceiling alerts — deferred to Phase 5
- [ ] ~~S3-07~~ Admission webhook — deferred to Phase 5
- [ ] ~~S3-11~~ Alert system — deferred to Phase 5

---

## Phase 2: App Deploy Engine

> **Goal:** Developer pushes code → app is built and deployed automatically on freezenith.com.
> **Status:** NOT STARTED (0/14)
> **Priority:** NEXT — this is the CORE product feature. Without this, there's no product.

### Tasks

- [ ] **S2-01** App model + migration
  - Table: `apps` (id, user_id, name, repo_url, branch, framework, status, subdomain, created_at)
  - Status: pending → building → deploying → running → sleeping → failed
  - Framework detection: nextjs, django, go, rails, flask, express, static

- [ ] **S2-02** GitHub webhook receiver
  - `POST /api/v1/webhooks/github` — receives push events
  - Validates webhook signature (HMAC-SHA256)
  - Triggers build pipeline for matching app

- [ ] **S2-03** Git clone + framework detection
  - Clone repo (shallow, specific branch)
  - Detect framework from files: `package.json`, `go.mod`, `requirements.txt`, `Gemfile`, `Dockerfile`
  - If `Dockerfile` exists → use it directly
  - Otherwise → generate Dockerfile from detected framework

- [ ] **S2-04** Dockerfile templates per framework
  - Next.js: multi-stage, standalone output
  - Go: multi-stage, static binary
  - Python/Django: pip install, gunicorn
  - Rails: bundle install, puma
  - Static: nginx
  - Generic: Dockerfile required

- [ ] **S2-05** Build pipeline (in-cluster)
  - Option A: Kaniko (build Docker images without Docker daemon)
  - Option B: BuildKit with rootless
  - Build image, tag with git SHA, store in local registry
  - Build logs streamed to API (websocket or SSE)

- [ ] **S2-06** Deploy to Kubernetes
  - Create per-app: Namespace (per user), Deployment, Service, IngressRoute
  - Subdomain: `{app-name}.freezenith.com` via wildcard DNS
  - Environment variables from user config
  - Resource limits based on plan (free: 0.5 CPU/512MB, pro: 2 CPU/2GB)

- [ ] **S2-07** Real K8s client (client-go)
  - Replace `k8s.MemoryClient` with real in-cluster client
  - CRUD for Deployments, Services, IngressRoutes, Namespaces
  - Watch for pod status changes (building → running → failed)

- [ ] **S2-08** Auto-TLS for app subdomains
  - Wildcard certificate for `*.freezenith.com` via cert-manager + DNS-01
  - Or per-app cert with HTTP-01 (simpler but slower)

- [ ] **S2-09** Environment variables management
  - API: CRUD for app env vars (stored encrypted in DB)
  - Dashboard: env var editor per app
  - Injected into Deployment as K8s Secret

- [ ] **S2-10** Build + deploy logs
  - Stream build output to user (websocket or polling)
  - Store last N build logs per app
  - Dashboard: build log viewer with ANSI color support

- [ ] **S2-11** Rollback support
  - Keep last 3 deployments (image tags)
  - One-click rollback to previous version
  - API: `POST /api/v1/apps/:id/rollback`

- [ ] **S2-12** Dashboard: App management pages
  - `/apps` — list of user's apps with status badges
  - `/apps/new` — connect repo, select branch, deploy
  - `/apps/:id` — overview, logs, env vars, settings, usage
  - `/apps/:id/deployments` — deployment history, rollback

- [ ] **S2-13** Dashboard: Deploy flow
  - Step 1: Enter GitHub repo URL (or connect GitHub account)
  - Step 2: Zenith detects framework, shows config
  - Step 3: Click deploy → live build log → app is live
  - Step 4: Shows URL: `my-app.freezenith.com`

- [ ] **S2-14** CLI deploy (optional, stretch goal)
  - `zen deploy` from repo root
  - Reads `.zenith.yml` or auto-detects
  - Pushes to API, streams build logs

### Definition of Done
- `git push` → app auto-rebuilds and redeploys
- Dashboard shows build logs in real-time
- Subdomain `{app}.freezenith.com` works with SSL
- Supports: Next.js, Go, Python, Rails, static, Dockerfile

---

## Phase 3: Built-in Services (Database, Auth, S3)

> **Goal:** Users get database, auth, and storage with one click. Like Supabase but part of the platform.
> **Status:** NOT STARTED (0/10)

### Tasks

- [ ] **S3-01** Per-app PostgreSQL provisioning
  - API: `POST /api/v1/apps/:id/databases` → create a PostgreSQL database
  - Uses shared PostgreSQL instance (separate DB per user, not per app)
  - Connection string returned to user + injected as env var `DATABASE_URL`
  - Size limits by plan (free: 500MB, pro: 5GB, team: 20GB)

- [ ] **S3-02** Database dashboard
  - Show databases per user with size, connection count
  - Connection string (copyable, masked by default)
  - Table browser (stretch goal: simple SQL explorer)

- [ ] **S3-03** Built-in auth service
  - User auth for apps: sign up, login, JWT, password reset
  - API: `POST /api/v1/apps/:id/auth/enable` → enable auth for app
  - Per-app user table in shared PostgreSQL
  - Limit by plan (free: 1K users, pro: 10K, team: 100K)

- [ ] **S3-04** Auth SDK snippet
  - JavaScript/TypeScript: `import { zenith } from '@zenith/sdk'`
  - `zenith.auth.signUp({ email, password })`
  - `zenith.auth.signIn({ email, password })`
  - `zenith.auth.getUser()`
  - REST API fallback for other languages

- [ ] **S3-05** S3 object storage per user
  - API: create bucket per user (Hetzner S3-compatible)
  - Access via `zenith.storage.upload(file)` or S3 API directly
  - Limits by plan (free: 1GB, pro: 10GB, team: 100GB)

- [ ] **S3-06** Storage dashboard
  - File browser, upload/download, delete
  - Usage bar (used vs limit)

- [ ] **S3-07** Redis provisioning (stretch goal)
  - Shared Redis instance, per-app database number
  - Connection string as env var `REDIS_URL`

- [ ] **S3-08** Database backups (Pro+ only)
  - Daily pg_dump per user database
  - Store in S3
  - Restore from backup (one-click)

- [ ] **S3-09** Service health dashboard
  - Show all user services: app, database, auth, storage
  - Status indicators, connection health
  - Quick actions: restart, view logs

- [ ] **S3-10** Documentation: "Deploy a full-stack app in 5 minutes"
  - Tutorial: Next.js + PostgreSQL + Auth
  - Shows the full flow from signup to live app

### Definition of Done
- User creates database with one click, gets connection string
- Auth works with SDK for JavaScript apps
- S3 storage accessible per user
- Dashboard shows all services with status

---

## Phase 4: KEDA Scale-to-Zero + SaaS Free Tier

> **Goal:** Free tier apps sleep when idle (€0.70/user/mah cost). Paid apps always-on. Plan enforcement.
> **Status:** NOT STARTED (0/11)

### Tasks

- [ ] **S4-01** Install KEDA + HTTP Add-on on K8s cluster
  - Helm install keda + keda-add-ons-http
  - Configure: interceptor timeout, scale-down period
  - Test with sample app: deploy → idle → scale to 0 → request → wake

- [ ] **S4-02** HTTPScaledObject per app (free tier)
  - When app is deployed on free plan:
    - Create `HTTPScaledObject` with `min: 0, max: 1, scaledownPeriod: 900` (15 min)
  - When app is on paid plan:
    - Create `HTTPScaledObject` with `min: 1, max: N` (always-on)
  - API generates correct YAML based on user's plan

- [ ] **S4-03** Cold start UX
  - When sleeping app receives request: show "Starting up..." splash page (2-5 sec)
  - Branded loading page with Zenith logo
  - Auto-redirect when app is ready
  - Paid plan → no cold start, no splash

- [ ] **S4-04** User registration + self-service signup
  - `POST /api/v1/auth/register` — create account (email, password, name)
  - Email verification (stretch: magic link)
  - Auto-assign Free plan
  - Create K8s namespace for user

- [ ] **S4-05** Plan assignment + resource quotas
  - Each user gets a plan (free/pro/team/managed)
  - K8s ResourceQuota per user namespace:
    - Free: 0.5 CPU, 512MB RAM, 1 app, 1 DB
    - Pro: 6 CPU, 6GB RAM, 3 apps, 2 DBs
    - Team: 40 CPU, 40GB RAM, 10 apps, 5 DBs
  - API enforces: can't create more apps than plan allows

- [ ] **S4-06** Plan upgrade trigger points
  - Show upgrade prompt when user hits limits:
    - "You've used 100% of your free database storage. Upgrade to Pro for 5GB."
    - "Custom domains require Pro plan. Upgrade now."
    - "Your app sleeps after 15 min. Upgrade to Pro for always-on."
  - Dashboard: persistent upgrade banner for free users

- [ ] **S4-07** Custom domain support (Pro+)
  - API: `POST /api/v1/apps/:id/domains` — add custom domain
  - User adds CNAME: `myapp.com → {app}.freezenith.com`
  - Verify DNS, provision TLS cert via cert-manager
  - Only available on paid plans

- [ ] **S4-08** Resource metering per user (reuse existing metering infra)
  - Collect: CPU, RAM, storage, DB size per namespace
  - Show in dashboard: usage vs plan ceiling
  - Reuse ProgressBar gauges from Phase 1.5

- [ ] **S4-09** Ceiling enforcement
  - When user hits plan limit → reject new deployments
  - Clear error: "Plan limit reached. Upgrade to deploy more apps."
  - Admin dashboard: see users approaching limits

- [ ] **S4-10** Sleep mode indicator in dashboard
  - Show which apps are sleeping vs active
  - "💤 Sleeping — will wake on next request (~3s)"
  - Last active timestamp

- [ ] **S4-11** Admin panel: user management (SaaS mode)
  - List all users with plan, usage, status
  - Suspend/activate users
  - Override plan limits
  - View user's apps and databases

### Definition of Done
- Free tier apps scale to zero after 15 min idle
- Cold start in 2-5 seconds with loading page
- Plan limits enforced (apps, DB, CPU, RAM)
- Users see upgrade prompts when hitting limits
- Admin can manage all users

---

## Phase 5: Hetzner Autoscaler

> **Goal:** Automatically scale Hetzner server pool based on demand. Cap at 10 servers (~€450/mah).
> **Status:** NOT STARTED (0/8)

### Tasks

- [ ] **S5-01** Hetzner Cloud API client (Go)
  - Server CRUD: create, delete, list, resize
  - Use `hcloud-go` SDK
  - Auth via HCLOUD_TOKEN env var

- [ ] **S5-02** Node pool manager
  - Track server pool: current count, total CPU/RAM, utilization
  - Minimum: 2 servers (availability)
  - Maximum: 10 servers (budget cap)
  - Server type: CPX52 (12 vCPU, 24GB RAM, €28.49/mah)

- [ ] **S5-03** Scale-up trigger
  - Monitor cluster resource utilization (Kubernetes metrics-server)
  - When total CPU > 80% or RAM > 80% → add 1 server
  - Cooldown: 5 min between scale-ups
  - New server: create Hetzner server → join K8s cluster → ready for pods

- [ ] **S5-04** Scale-down trigger
  - When total CPU < 40% AND RAM < 40% → remove 1 server
  - Cooldown: 15 min between scale-downs
  - Drain pods first (kubectl drain), then delete Hetzner server
  - Never scale below minimum (2 servers)

- [ ] **S5-05** K8s node join/leave automation
  - New Hetzner server → install k3s agent → join cluster
  - Use k3s token for cluster join
  - Remove: cordon → drain → delete node → delete Hetzner server

- [ ] **S5-06** Cost tracking
  - Track actual Hetzner spend vs budget cap (€450/mah)
  - Alert if approaching budget
  - Admin dashboard: current server count, cost, utilization

- [ ] **S5-07** Warm buffer (stretch goal)
  - Keep 1 standby server pre-provisioned
  - When scale-up needed → promote standby (instant) + provision new standby
  - Reduces cold-start time from ~2min to ~10sec

- [ ] **S5-08** Admin dashboard: infrastructure view
  - Server list with CPU/RAM/disk per server
  - Total cluster capacity vs used
  - Scale history (when servers were added/removed)
  - Budget: current spend vs cap

### Definition of Done
- Cluster auto-scales from 2 to 10 servers based on demand
- Budget cap prevents overspending
- New servers join cluster within 2 minutes
- Admin can see infrastructure status and cost

---

## Phase 6: Billing (Stripe)

> **Goal:** Users upgrade from Free → Pro/Team, pay via Stripe. Revenue flows.
> **Status:** NOT STARTED (0/9)

### Tasks

- [ ] **S6-01** Stripe Go SDK integration
  - Products: Free, Pro (€29), Team (€199), Managed (custom)
  - Stripe Products + Prices created via API or dashboard

- [ ] **S6-02** Checkout flow
  - User clicks "Upgrade to Pro" → Stripe Checkout Session
  - Success → update user plan in DB → apply new resource quotas
  - Cancel → stay on current plan

- [ ] **S6-03** Stripe webhook handler
  - `checkout.session.completed` → upgrade plan
  - `invoice.paid` → record payment
  - `invoice.payment_failed` → notify, grace period
  - `customer.subscription.deleted` → downgrade to free

- [ ] **S6-04** Subscription management
  - User can: view current plan, upgrade, downgrade, cancel
  - Stripe Customer Portal for payment method management
  - Pro-rated billing for mid-cycle changes

- [ ] **S6-05** Dashboard: billing page (user-facing)
  - Current plan, next billing date, payment method
  - Usage vs limits
  - Invoice history
  - Upgrade/downgrade buttons

- [ ] **S6-06** Admin dashboard: billing overview
  - MRR, total revenue, active subscriptions
  - Failed payments, churn rate
  - Revenue per plan breakdown

- [ ] **S6-07** Invoices migration + storage
  - Table: invoices (id, user_id, stripe_invoice_id, amount, currency, status, period)
  - API: list invoices per user

- [ ] **S6-08** Downgrade handling
  - When user downgrades: enforce new limits
  - If over limit: don't delete — show warning, prevent new resources
  - Grace period: 7 days to reduce usage before suspension

- [ ] **S6-09** Free → Pro conversion optimization
  - Track: which limit triggered upgrade (DB size, custom domain, always-on)
  - A/B test upgrade prompts
  - Usage-based nudges in dashboard

### Definition of Done
- Users can upgrade/downgrade via Stripe
- Subscriptions auto-renew monthly
- Failed payments handled with grace period
- Admin sees MRR and billing overview

---

## Phase 7: Open-Source Extraction

> **Goal:** Extract self-hosted version from SaaS. docker-compose, README, LICENSE, install script.
> **Status:** NOT STARTED (0/12)
> **Timing:** AFTER SaaS is working. Strip down the full product to open-source version.

### Tasks

- [ ] **S7-01** `docker-compose.yml` at repo root
  - Services: zenith-api, zenith-dashboard, postgres, traefik (optional)
  - Ports: API 8080, Dashboard 3100, Postgres 5432
  - Postgres with seed data (demo user, sample app)
  - Single command: `docker compose up` → working platform
  - Environment: `ZENITH_MODE=standalone`

- [ ] **S7-02** `ZENITH_MODE` env var in API + Dashboard
  - `standalone`: single user, no billing, no KEDA, no multi-tenant, no admin panel
  - `saas`: multi-user, all features, used on freezenith.com
  - API: conditionally register routes based on mode
  - Dashboard: conditionally show sidebar items based on mode

- [ ] **S7-03** Simplified Dashboard for standalone mode
  - Remove: Customers, Plans, Billing, Metering, Admin pages
  - Keep: Apps, Databases, Settings
  - Clean, developer-focused UX

- [ ] **S7-04** `README.md` rewrite for open-source
  - Hero: "Open-source PaaS. Deploy apps, databases, auth. Runs anywhere."
  - Quick Start (docker compose up)
  - Screenshots / GIF of dashboard
  - Feature comparison table (Zenith vs Coolify vs Railway)
  - Architecture diagram, contributing guide, badges

- [ ] **S7-05** `LICENSE` file — AGPLv3 (same as GitLab open-core model)

- [ ] **S7-06** `.github/` setup
  - Issue templates, PR template, `CONTRIBUTING.md`
  - GitHub Actions: CI (Go test, TypeScript check, Docker build)

- [ ] **S7-07** Seed data for standalone mode
  - Default admin user (admin@zenith.local / changeme)
  - Sample app deployment, sample PostgreSQL database
  - First-run setup wizard (change password, set domain)

- [ ] **S7-08** One-liner install script
  ```bash
  curl -fsSL https://get.freezenith.com | bash
  ```
  - Detects OS, installs Docker if missing, runs docker compose up -d

- [ ] **S7-09** Docs site (`docs/` or separate)
  - Getting started, architecture, deploy your first app, configuration reference

- [ ] **S7-10** Landing page update (`apps/landing/`)
  - Two CTAs: "Self-Host Free" → GitHub | "Try Cloud" → freezenith.com/signup
  - Feature grid, pricing table, screenshots

- [ ] **S7-11** Security hardening for public release
  - Remove hardcoded secrets, CORS config, rate limiting, Dependabot

- [ ] **S7-12** Rename `apps/mission-control/` → `apps/dashboard/` for open-source branding

### Definition of Done
- `docker compose up` → working Zenith in under 2 minutes
- README with screenshots, LICENSE, contributing guide
- GitHub repo presentable for HN/Reddit launch

---

## Phase 8: Launch & Marketing

> **Goal:** Public launch. Get 1,000+ GitHub stars and first paying customers.
> **Status:** NOT STARTED (0/8)

### Tasks

- [ ] **S8-01** GitHub repo public launch
  - Clean commit history (squash if needed)
  - Badges: stars, license, Discord, CI status
  - Releases page with changelog

- [ ] **S8-02** Hacker News "Show HN" post
  - Title: "Show HN: Zenith – Open-source PaaS, deploy with git push, 70% cheaper than AWS"
  - Timing: Tuesday/Wednesday, 8-9am EST
  - Prepare for feedback: be responsive in comments

- [ ] **S8-03** Reddit launch (parallel)
  - r/selfhosted (1.2M), r/kubernetes (300K), r/devops (500K), r/opensource (100K)
  - Style: "I built X, here's how, what do you think?"

- [ ] **S8-04** LinkedIn content campaign
  - 32K followers — leverage existing audience
  - 3 posts/week: technical insights, building in public, cost comparisons
  - Boost best post: €100/mah

- [ ] **S8-05** Product Hunt launch
  - Screenshots, 60-second demo video
  - Maker comment with story
  - Coordinate with community for upvotes

- [ ] **S8-06** Discord community setup
  - Channels: general, help, feature-requests, showcase
  - Bot: GitHub notifications, deploy status
  - Community support → reduce support load

- [ ] **S8-07** Demo video (60-90 seconds)
  - Show: sign up → connect repo → deploy → live app
  - Screen recording with voiceover
  - Post on: YouTube, Twitter/X, LinkedIn, landing page

- [ ] **S8-08** First 5 beta customers
  - Offer 3 months Pro free for early adopters
  - Collect testimonials and case studies
  - Anonymized metrics for marketing ("deployed 50 apps, 99.9% uptime")

### Definition of Done
- 1,000+ GitHub stars within first month
- 50+ Discord members
- 5+ beta customers on Pro plan
- HN front page (target, not guarantee)

---

## Priority Order

```
Phase 2: App Deploy Engine          ← CORE FEATURE (git push → live on freezenith.com)
Phase 3: Built-in Services          ← ADD VALUE (DB, auth, S3)
Phase 4: KEDA + Free Tier           ← SCALE FREE USERS (sleep mode, plans)
Phase 6: Billing (Stripe)           ← GET REVENUE (Pro/Team plans)
Phase 5: Hetzner Autoscaler         ← SCALE INFRA (auto scale servers)
Phase 7: Open-Source Extraction     ← EXTRACT docker-compose from SaaS
Phase 8: Launch & Marketing         ← GET USERS (HN, Reddit, LinkedIn, Product Hunt)
```

Phase 2-3 make the **SaaS product work** (deploy apps, databases, auth).
Phase 4 makes it **cost-efficient** (1000+ free users with sleep mode).
Phase 6 makes it **profitable** (Stripe subscriptions).
Phase 5 makes it **reliable** (auto-scaling infrastructure).
Phase 7 makes it **famous** (open-source community).
Phase 8 makes it **known** (marketing push).

---

## User Flows

### Flow 1: Developer Sign Up → First Deploy (SaaS)

```
1. Developer visits freezenith.com
2. Clicks "Try Cloud Free" → /signup
3. Enters: name, email, password
4. Email verification (or instant for beta)
5. Dashboard loads → empty state: "Deploy your first app"
6. Clicks "New App" → enters GitHub repo URL
7. Zenith detects framework (e.g. Next.js)
8. Shows config: branch, build command, env vars
9. Clicks "Deploy" → build log streams in real-time
10. Build complete → app live at my-app.freezenith.com
11. Dashboard shows: app running, URL, logs, usage
```

### Flow 2: Developer Adds Database

```
1. Goes to app detail → "Databases" tab
2. Clicks "Create Database" → PostgreSQL
3. Database created → connection string shown
4. Clicks "Add to app" → DATABASE_URL injected as env var
5. App auto-redeploys with database connected
```

### Flow 3: Free → Pro Upgrade

```
1. Developer's DB hits 500MB limit → warning banner
2. Or: wants custom domain → "Requires Pro plan"
3. Or: tired of 15 min sleep → wants always-on
4. Clicks "Upgrade to Pro" → Stripe Checkout
5. Pays €29 → plan updated → limits increased
6. App instantly gets: more RAM, always-on, custom domain option
```

### Flow 4: Git Push → Auto-Redeploy

```
1. Developer pushes to main branch on GitHub
2. GitHub webhook fires → Zenith API receives push event
3. API triggers build: clone → detect → build image → tag with SHA
4. Old deployment replaced with new image (rolling update)
5. Dashboard shows: new deployment, build log, previous version
6. If build fails → old version stays running, user sees error in logs
```

### Flow 5: Admin Manages Platform (SaaS Mode)

```
1. Admin logs into admin.freezenith.com
2. Dashboard: total users, MRR, active apps, server utilization
3. User list: filter by plan, usage, status
4. Click user → see their apps, databases, usage vs limits
5. Actions: suspend, upgrade, override limits
6. Infrastructure tab: server count, auto-scaler status, cost
```

### Flow 6: Self-Hosted Setup (Open-Source)

```
1. Developer clones repo from GitHub
2. Runs: docker compose up
3. Opens localhost:3100 → dashboard loads
4. Logs in with default credentials → prompted to change password
5. Same deploy flow as SaaS but on their own machine
6. No billing, no sleep mode, no multi-user — just works
```

### Flow 7: Full Stack Setup (Every Plan Gets Everything)

```
1. User signs up (Free, Pro, Team — doesn't matter)
2. Dashboard → "New App" → deploys backend (Go/Node/Python)
3. Dashboard → "New App" → deploys frontend (Next.js/React)
4. Databases tab → "Create PostgreSQL" → connection string auto-injected
5. Auth tab → "Enable Auth" → SDK snippet, sign up/login ready
6. Team tab → "Invite Member" → add teammate by email
7. Storage tab → "Create Bucket" → S3 access keys + upload UI
8. Monitoring tab → logs, CPU/RAM gauges, request metrics
9. Registry tab → container images listed, pull commands shown
10. Everything works on EVERY plan — only RESOURCE LIMITS differ:
    - Free: 1 app, 500MB DB, 1GB S3, 1 team member, 1 day logs
    - Pro: 5 apps, 5GB DB, 10GB S3, 3 members, 7 day logs
    - Team: 20 apps, 20GB DB, 100GB S3, 10 members, 30 day logs
11. Enterprise-only extras: SLA 99.9%, dedicated infra, priority support
12. SSO + audit log: Team+ only
```

---

## Test Plan

### Phase 2: App Deploy Engine — Tests

| # | Test Case | Type | Expected Result |
|---|-----------|------|-----------------|
| T2-01 | Create app with valid GitHub repo URL | API | App created, status: pending |
| T2-02 | Create app with invalid repo URL | API | 400 error, clear message |
| T2-03 | GitHub webhook triggers build on push | Integration | Build starts, status: building |
| T2-04 | GitHub webhook with invalid signature | Security | 401 rejected |
| T2-05 | Framework detection: Next.js (package.json + next) | Unit | Detected as "nextjs" |
| T2-06 | Framework detection: Go (go.mod) | Unit | Detected as "go" |
| T2-07 | Framework detection: Python (requirements.txt) | Unit | Detected as "python" |
| T2-08 | Framework detection: Dockerfile present | Unit | Uses Dockerfile directly |
| T2-09 | Framework detection: unknown project | Unit | Error: "Dockerfile required" |
| T2-10 | Build succeeds → deployment created | Integration | Pod running, IngressRoute created |
| T2-11 | Build fails → old version stays | Integration | Previous deployment unchanged |
| T2-12 | App accessible via subdomain | E2E | HTTPS 200 at {app}.freezenith.com |
| T2-13 | Env vars injected into deployment | Integration | App reads DATABASE_URL correctly |
| T2-14 | Rollback to previous version | API | Previous image tag deployed |
| T2-15 | Delete app → all resources cleaned | Integration | Deployment, Service, IngressRoute removed |
| T2-16 | Build log streamed to client | Integration | Real-time log output via SSE/WS |
| T2-17 | Concurrent builds (2 apps same time) | Load | Both build successfully, no conflicts |

### Phase 3: Built-in Services — Tests

| # | Test Case | Type | Expected Result |
|---|-----------|------|-----------------|
| T3-01 | Create PostgreSQL database | API | DB created, connection string returned |
| T3-02 | Create DB over plan limit | API | 403: "Plan limit reached" |
| T3-03 | Database size tracking | Integration | Size reported accurately in dashboard |
| T3-04 | Delete database | API | DB dropped, resources freed |
| T3-05 | Enable auth on app | API | Auth tables created, endpoints active |
| T3-06 | Auth: sign up new user | API | User created, JWT returned |
| T3-07 | Auth: sign in | API | JWT returned, valid claims |
| T3-08 | Auth: invalid credentials | API | 401 unauthorized |
| T3-09 | Auth user limit enforcement | API | 403 at plan limit (1K free, 10K pro) |
| T3-10 | S3 bucket creation | API | Bucket created, credentials returned |
| T3-11 | S3 storage limit enforcement | API | Upload rejected at plan limit |
| T3-12 | Database backup (Pro) | Integration | pg_dump created, stored in S3 |
| T3-13 | Database restore from backup | Integration | Data restored correctly |

### Phase 4: KEDA + Free Tier — Tests

| # | Test Case | Type | Expected Result |
|---|-----------|------|-----------------|
| T4-01 | Free app scales to zero after 15 min idle | Integration | Replicas: 0 after idle period |
| T4-02 | Request to sleeping app wakes it | E2E | App responds within 5 sec, loading page shown |
| T4-03 | Pro app never sleeps | Integration | Replicas stays at 1 after 30 min idle |
| T4-04 | User signup creates namespace | Integration | K8s namespace created with ResourceQuota |
| T4-05 | Free user can't create 2nd app | API | 403: "Free plan allows 1 app" |
| T4-06 | Pro user can create 3 apps | API | All 3 apps created successfully |
| T4-07 | Resource quota enforced | Integration | Pod rejected if over namespace quota |
| T4-08 | Custom domain only on paid plan | API | 403 for free, 200 for pro |
| T4-09 | Custom domain DNS verification | Integration | CNAME verified, cert issued |
| T4-10 | Upgrade trigger shown at limit | E2E | Banner appears when DB at 90% |
| T4-11 | Sleep indicator in dashboard | E2E | Shows sleeping/active status per app |

### Phase 5: Hetzner Autoscaler — Tests

| # | Test Case | Type | Expected Result |
|---|-----------|------|-----------------|
| T5-01 | Scale up when CPU > 80% | Integration | New server created and joined cluster |
| T5-02 | Scale down when CPU < 40% | Integration | Server drained and deleted |
| T5-03 | Never scale below 2 servers | Integration | Scale-down rejected at minimum |
| T5-04 | Never scale above 10 servers | Integration | Scale-up rejected at maximum |
| T5-05 | Budget cap prevents overspend | Integration | Alert at 90% budget, block at 100% |
| T5-06 | New node joins cluster | Integration | Node ready, pods schedulable |
| T5-07 | Node drain before removal | Integration | Pods rescheduled, zero downtime |

### Phase 6: Billing — Tests

| # | Test Case | Type | Expected Result |
|---|-----------|------|-----------------|
| T6-01 | Stripe checkout creates subscription | Integration | User plan upgraded in DB |
| T6-02 | Webhook: invoice.paid | Integration | Payment recorded |
| T6-03 | Webhook: payment_failed | Integration | User notified, grace period started |
| T6-04 | Webhook: subscription.deleted | Integration | User downgraded to free |
| T6-05 | Downgrade: over limit handling | Integration | Warning shown, no data deleted |
| T6-06 | Pro-rated billing on upgrade | Integration | Correct amount charged by Stripe |
| T6-07 | Invoice history displayed | E2E | All invoices shown with status |

### Phase 7: Open-Source — Tests

| # | Test Case | Type | Expected Result |
|---|-----------|------|-----------------|
| T7-01 | `docker compose up` works | E2E | All services healthy in < 2 min |
| T7-02 | Default login works | E2E | admin@zenith.local / changeme → dashboard |
| T7-03 | Deploy app in standalone mode | E2E | App builds and runs on localhost |
| T7-04 | No billing pages in standalone | E2E | Sidebar has no billing/plans links |
| T7-05 | Install script on fresh Ubuntu | E2E | Docker installed, Zenith running |

---

## Non-Functional Requirements

### Performance
| Metric | Target |
|--------|--------|
| App deploy (build + deploy) | < 3 minutes |
| Cold start (sleeping app wake) | < 5 seconds |
| API response time (p95) | < 200ms |
| Dashboard page load | < 2 seconds |
| Concurrent builds | 5 simultaneous |

### Security
| Requirement | Implementation |
|-------------|---------------|
| Auth | JWT with refresh tokens, bcrypt passwords |
| Secrets | Env vars encrypted at rest in DB, K8s Secrets |
| Network | Namespace isolation per user, NetworkPolicies |
| TLS | All traffic encrypted, auto-cert via Let's Encrypt |
| Webhooks | HMAC-SHA256 signature validation |
| Rate limiting | Auth endpoints: 10 req/min, API: 100 req/min |
| RBAC | Owner > Admin > Developer > Viewer roles |

### Availability
| Metric | Target |
|--------|--------|
| Platform uptime (SaaS) | 99.9% (Pro+) |
| Data durability | Daily backups (Pro), hourly (Team) |
| Failover | Auto-restart on crash, pod rescheduling |
| Autoscaler response | < 2 min to add new server |

### Compliance
| Requirement | Status |
|-------------|--------|
| GDPR — data in EU | ✅ (European data centers) |
| Audit log | ✅ Built (Phase 1.5) |
| Data export | Planned (Phase 7) |
| SOC 2 Type 1 | Future (when enterprise customers require it) |

---

## Risk & Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Server OOM during Docker builds | High | Medium | In-cluster builds (Kaniko), not Docker daemon |
| Free tier abuse (crypto mining, spam) | High | High | CPU limits, network policies, abuse detection |
| Hetzner outage | High | Low | Multi-region (future), status page |
| GitHub API rate limits | Medium | Medium | Cache webhook payloads, retry with backoff |
| Stripe webhook failures | Medium | Low | Idempotent handlers, retry queue |
| Build pipeline security (malicious Dockerfile) | High | Medium | Kaniko sandboxing, no privileged containers |
| Cost overrun (too many free users) | Medium | Medium | Budget cap on autoscaler, waitlist for free tier |
| Competition (Coolify, Railway) | Medium | High | Differentiate: K8s-native, European, compliance |

---

## What Exists Today

### API (`services/api/`) — Go + Fiber + PostgreSQL
- JWT auth, user management, customer/plan CRUD, metering
- Repository pattern with Memory + Postgres implementations
- Admin endpoints: dashboard, customers, plans, usage, metering
- Health check, CORS, structured error handling

### Dashboard (`apps/mission-control/`) — Next.js 15
- Login, Dashboard, Customers, Plans, Clusters, Modules, Settings
- Demo mode with mock data
- Auth integration, protected routes
- Resource usage gauges, metering history

### Infrastructure — Live (European Data Center)
- k3s v1.34.3, Traefik 3.5.1
- PostgreSQL 16 with persistent storage
- cert-manager, Let's Encrypt
- 6 Docker images, automated deploy script

### What Needs to Change for New Direction
- Dashboard: add app deploy pages, simplify for standalone mode
- API: add app/build/deploy handlers, GitHub webhook, K8s client
- Infrastructure: KEDA, Hetzner autoscaler, in-cluster build pipeline
- New: docker-compose.yml, README, install script, docs
