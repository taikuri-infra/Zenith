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

#### Resources & Core Stack

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

#### Developer Experience

| | **Free** | **Pro €29/mo** | **Team €199/mo** | **Enterprise (custom)** |
|---|---|---|---|---|
| **GitHub Integration** | ✅ | ✅ | ✅ | ✅ |
| **GitLab / Bitbucket** | ❌ | ✅ | ✅ | ✅ |
| **Preview Deployments (per PR)** | ❌ | ❌ | ✅ | ✅ |
| **Rollback (one-click)** | ❌ | ✅ | ✅ | ✅ |
| **API Keys** | 1 | 5 | 20 | Unlimited |
| **Webhook Events** | ❌ | ✅ | ✅ | ✅ |
| **Data Export** | ✅ | ✅ | ✅ | ✅ |

#### Security & Compliance

| | **Free** | **Pro €29/mo** | **Team €199/mo** | **Enterprise (custom)** |
|---|---|---|---|---|
| **MFA / 2FA** | ❌ | ✅ | ✅ | ✅ |
| **SSO (SAML/OIDC)** | ❌ | ❌ | ✅ | ✅ |
| **SCIM Provisioning** | ❌ | ❌ | ❌ | ✅ |
| **Audit Log** | ❌ | ❌ | ✅ | ✅ export + retention policy |
| **Custom Roles (RBAC)** | ❌ | ❌ | ❌ | ✅ |
| **IP Whitelisting** | ❌ | ❌ | ❌ | ✅ |
| **Session Management** | ❌ | ❌ | ✅ | ✅ |
| **GDPR / EU Data Residency** | ✅ | ✅ | ✅ | ✅ |
| **DPA (Data Processing Agreement)** | ❌ | ❌ | ✅ | ✅ |
| **Compliance Dashboard** | ❌ | ❌ | ❌ | ✅ |
| **Pen Test Report (annual)** | ❌ | ❌ | ❌ | ✅ |
| **SOC 2 / ISO 27001 aligned** | ❌ | ❌ | ❌ | ✅ |

#### Infrastructure & Reliability

| | **Free** | **Pro €29/mo** | **Team €199/mo** | **Enterprise (custom)** |
|---|---|---|---|---|
| **Shared Infrastructure** | ✅ | ✅ | ✅ | ❌ dedicated |
| **Dedicated Infrastructure** | ❌ | ❌ | ❌ | ✅ |
| **Private Networking (VPC)** | ❌ | ❌ | ❌ | ✅ |
| **Auto-scaling** | ❌ | ❌ | ✅ | ✅ |
| **SLA** | ❌ | ❌ | 99.5% | 99.9% |
| **Incident Response SLA** | ❌ | ❌ | 24h | 1h |
| **Scheduled Maintenance Window** | ❌ | ❌ | ❌ | ✅ (customer chooses) |
| **Multi-region (future)** | ❌ | ❌ | ❌ | ✅ |

#### Support

| | **Free** | **Pro €29/mo** | **Team €199/mo** | **Enterprise (custom)** |
|---|---|---|---|---|
| **Community (Discord)** | ✅ | ✅ | ✅ | ✅ |
| **Email Support** | ❌ | ✅ (48h) | ✅ (24h) | ✅ (4h) |
| **Slack Channel** | ❌ | ❌ | ❌ | ✅ dedicated |
| **Phone Support** | ❌ | ❌ | ❌ | ✅ |
| **Onboarding Call** | ❌ | ❌ | ❌ | ✅ |

#### White-Label (Enterprise Add-on)

| | **Free** | **Pro** | **Team** | **Enterprise** |
|---|---|---|---|---|
| **Custom Branding (logo, colors)** | ❌ | ❌ | ❌ | ✅ |
| **Custom Dashboard Domain** | ❌ | ❌ | ❌ | ✅ |
| **Remove "Powered by Zenith"** | ❌ | ❌ | ❌ | ✅ |

#### Feature Count per Plan

| | **Free** | **Pro €29** | **Team €199** | **Enterprise** |
|---|---|---|---|---|
| Resources & Core Stack (12) | 10 | 12 | 12 | 12 |
| Developer Experience (7) | 3 | 6 | 7 | 7 |
| Security & Compliance (12) | 1 | 2 | 6 | 12 |
| Infrastructure & Reliability (8) | 1 | 1 | 4 | 7 |
| Support (5) | 1 | 2 | 2 | 5 |
| White-Label (3) | 0 | 0 | 0 | 3 |
| **Total (out of 47)** | **16 (34%)** | **23 (49%)** | **31 (66%)** | **46 (98%)** |

**What each upgrade adds:**
- **Free → Pro** (+7): custom domain, backups, GitLab/Bitbucket, rollback, webhooks, MFA, email support
- **Pro → Team** (+8): SSO, audit log, DPA, session management, preview deploys, auto-scaling, SLA 99.5%, incident SLA 24h
- **Team → Enterprise** (+15): SCIM, custom roles, IP whitelisting, compliance dashboard, pen test report, SOC 2/ISO 27001, VPC, dedicated infra, maintenance window, multi-region, Slack channel, phone support, onboarding call, white-label (×3)

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

### Flow 7: Full Stack Setup (Every Plan Gets the Full Stack)

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
    - Free: 1 app, 500MB DB, 1GB S3, 1 member, 1 day logs
    - Pro: 5 apps, 5GB DB, 10GB S3, 3 members, 7 day logs
    - Team: 20 apps, 20GB DB, 100GB S3, 10 members, 30 day logs
11. Pro adds: MFA, custom domain, rollback, webhooks, GitLab/Bitbucket, email support
12. Team adds: SSO, audit log, DPA, preview deploys, auto-scaling, SLA 99.5%, session mgmt
13. Enterprise: SCIM, custom roles, IP whitelisting, compliance dashboard, VPC,
    dedicated infra, 99.9% SLA, 1h incident response, Slack/phone, white-label
```

---

## Test Plan

### E2E Customer Journey Scenarios

These scenarios simulate a real customer using the platform end to end.
Each scenario is a script: register → do stuff → verify with curl → confirm everything works.

---

#### Scenario 1: Free User — Full Stack Setup

> A new user signs up, deploys a backend + frontend, creates a database, enables auth, uploads to S3, and verifies everything works via subdomain.

```
# --- SETUP ---
API=https://api.freezenith.com/api/v1

# 1. Register a new free user
TOKEN=$(curl -s -X POST $API/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Test User","email":"testuser@example.com","password":"Test1234!"}' \
  | jq -r '.token')
# EXPECT: token is non-empty string

# 2. Verify free plan assigned
curl -s -H "Authorization: Bearer $TOKEN" $API/me | jq '.plan'
# EXPECT: "free"

# 3. Check dashboard — empty state
curl -s -H "Authorization: Bearer $TOKEN" $API/apps | jq '.apps | length'
# EXPECT: 0

# --- DEPLOY BACKEND ---

# 4. Create backend app (Go test API)
APP_BE=$(curl -s -X POST $API/apps \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-api","repo_url":"https://github.com/taikuri-infra/zenith-test-go","branch":"main"}' \
  | jq -r '.id')
# EXPECT: app ID returned, status: "pending"

# 5. Wait for build to complete (poll status)
for i in $(seq 1 60); do
  STATUS=$(curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_BE | jq -r '.status')
  echo "Backend build: $STATUS"
  [ "$STATUS" = "running" ] && break
  sleep 5
done
# EXPECT: status goes pending → building → deploying → running (< 3 min)

# 6. Verify backend is live
curl -s https://test-api.freezenith.com/health
# EXPECT: {"status":"ok"}

curl -s https://test-api.freezenith.com/api/ping
# EXPECT: {"message":"pong"} (test app endpoint)

# --- DEPLOY FRONTEND ---

# 7. Create frontend app (Next.js test app)
APP_FE=$(curl -s -X POST $API/apps \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-web","repo_url":"https://github.com/taikuri-infra/zenith-test-nextjs","branch":"main"}' \
  | jq -r '.id')
# EXPECT: FAIL — 403 "Free plan allows 1 app"
# (Free plan = 1 app only, backend already used the slot)

# --- CREATE DATABASE ---

# 8. Create PostgreSQL for backend
DB=$(curl -s -X POST $API/apps/$APP_BE/databases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"postgresql"}' \
  | jq -r '.connection_string')
# EXPECT: postgresql://user:pass@host:5432/test_api_db

# 9. Verify DATABASE_URL injected into app
curl -s https://test-api.freezenith.com/api/db-check
# EXPECT: {"connected":true,"tables":0} (test app reads DATABASE_URL)

# 10. Insert dummy data via test app
curl -s -X POST https://test-api.freezenith.com/api/items \
  -H "Content-Type: application/json" \
  -d '{"name":"Widget","price":9.99}'
# EXPECT: {"id":1,"name":"Widget","price":9.99}

curl -s https://test-api.freezenith.com/api/items
# EXPECT: [{"id":1,"name":"Widget","price":9.99}]

# --- ENABLE AUTH ---

# 11. Enable built-in auth on the app
curl -s -X POST $API/apps/$APP_BE/auth/enable \
  -H "Authorization: Bearer $TOKEN"
# EXPECT: {"enabled":true,"endpoint":"https://test-api.freezenith.com/auth"}

# 12. Register a user in the APP's auth (not platform auth)
curl -s -X POST https://test-api.freezenith.com/auth/signup \
  -d '{"email":"appuser@test.com","password":"Pass1234!"}'
# EXPECT: {"user_id":"...","jwt":"..."}

# 13. Login with that user
APP_USER_TOKEN=$(curl -s -X POST https://test-api.freezenith.com/auth/login \
  -d '{"email":"appuser@test.com","password":"Pass1234!"}' \
  | jq -r '.jwt')
# EXPECT: valid JWT

# 14. Access protected endpoint
curl -s https://test-api.freezenith.com/api/items \
  -H "Authorization: Bearer $APP_USER_TOKEN"
# EXPECT: 200 OK with items list

# --- S3 STORAGE ---

# 15. Create S3 bucket
S3_CREDS=$(curl -s -X POST $API/apps/$APP_BE/storage \
  -H "Authorization: Bearer $TOKEN")
# EXPECT: {"bucket":"test-api-storage","access_key":"...","secret_key":"...","endpoint":"..."}

# 16. Upload a file via test app
curl -s -X POST https://test-api.freezenith.com/api/upload \
  -F "file=@/tmp/test.txt"
# EXPECT: {"url":"https://s3.freezenith.com/test-api-storage/test.txt"}

# 17. Download the file
curl -s https://s3.freezenith.com/test-api-storage/test.txt
# EXPECT: contents of test.txt

# --- MONITORING ---

# 18. Check app logs
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_BE/logs?lines=10
# EXPECT: last 10 log lines from the running app

# 19. Check app metrics
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_BE/metrics
# EXPECT: {"cpu_percent":2.1,"ram_mb":64,"requests_1h":15,"status":"running"}

# --- VERIFY LIMITS ---

# 20. Try to create second database (should fail on free)
curl -s -X POST $API/apps/$APP_BE/databases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"postgresql","name":"second-db"}'
# EXPECT: 403 "Free plan allows 1 database"

# 21. Check usage dashboard
curl -s -H "Authorization: Bearer $TOKEN" $API/usage
# EXPECT: {"apps":{"used":1,"limit":1},"databases":{"used":1,"limit":1},"storage_mb":{"used":0.01,"limit":1024},"auth_users":{"used":1,"limit":1000}}

# --- CLEANUP ---

# 22. Delete app
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" $API/apps/$APP_BE
# EXPECT: 200 — app, database, auth, storage all cleaned up

# 23. Verify subdomain is gone
curl -s https://test-api.freezenith.com
# EXPECT: 404 or Traefik default page

# 24. Verify namespace cleaned
curl -s -H "Authorization: Bearer $TOKEN" $API/apps | jq '.apps | length'
# EXPECT: 0
```

**Pass criteria:** All 24 steps return expected results. Total time < 5 minutes.

---

#### Scenario 2: Pro User — Multi-App + Database + Custom Domain

> Pro user deploys backend + frontend + second backend, connects databases, sets custom domain, pushes update via git, rolls back.

```
API=https://api.freezenith.com/api/v1

# 1. Register + upgrade to Pro (or use test account with Pro plan)
TOKEN=$(curl -s -X POST $API/auth/login \
  -d '{"email":"pro-test@example.com","password":"ProTest1234!"}' | jq -r '.token')

# 2. Verify Pro plan
curl -s -H "Authorization: Bearer $TOKEN" $API/me | jq '.plan'
# EXPECT: "pro"

# --- DEPLOY 3 APPS ---

# 3. Deploy Go backend
APP1=$(curl -s -X POST $API/apps \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"myapi","repo_url":"https://github.com/taikuri-infra/zenith-test-go","branch":"main"}' \
  | jq -r '.id')
# EXPECT: created

# 4. Deploy Next.js frontend
APP2=$(curl -s -X POST $API/apps \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"myweb","repo_url":"https://github.com/taikuri-infra/zenith-test-nextjs","branch":"main"}' \
  | jq -r '.id')
# EXPECT: created

# 5. Deploy Python worker
APP3=$(curl -s -X POST $API/apps \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"myworker","repo_url":"https://github.com/taikuri-infra/zenith-test-python","branch":"main"}' \
  | jq -r '.id')
# EXPECT: created

# 6. Wait for all 3 to be running
# (poll each app's status until "running")
# EXPECT: all 3 running within 5 minutes

# 7. Verify all subdomains
curl -s https://myapi.freezenith.com/health     # EXPECT: 200
curl -s https://myweb.freezenith.com             # EXPECT: 200 HTML
curl -s https://myworker.freezenith.com/health   # EXPECT: 200

# 8. Try to create 4th app (over limit)
curl -s -X POST $API/apps \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"toomany","repo_url":"https://github.com/user/repo","branch":"main"}'
# EXPECT: 403 "Pro plan allows 5 apps" (we have 3, should work actually)
# Create 4th and 5th → succeed. 6th → 403.

# --- DATABASES ---

# 9. Create DB for backend
curl -s -X POST $API/apps/$APP1/databases \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"type":"postgresql"}'
# EXPECT: connection string returned

# 10. Create DB for frontend (shared queries)
curl -s -X POST $API/apps/$APP2/databases \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"type":"postgresql"}'
# EXPECT: second DB created (Pro allows 3)

# 11. Create 3rd DB
curl -s -X POST $API/apps/$APP3/databases \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"type":"postgresql"}'
# EXPECT: 200 (3rd of 3 allowed)

# 12. Create 4th DB (over limit)
curl -s -X POST $API/apps/$APP1/databases \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"type":"postgresql","name":"extra"}'
# EXPECT: 403 "Pro plan allows 3 databases"

# --- CUSTOM DOMAIN ---

# 13. Add custom domain to frontend
curl -s -X POST $API/apps/$APP2/domains \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"domain":"app.testcustomer.com"}'
# EXPECT: {"status":"pending_verification","cname_target":"myweb.freezenith.com"}

# 14. After DNS CNAME is set → verify
curl -s -X POST $API/apps/$APP2/domains/verify \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"domain":"app.testcustomer.com"}'
# EXPECT: {"status":"active","ssl":"provisioning"}

# 15. Wait for SSL, then verify custom domain
curl -s https://app.testcustomer.com
# EXPECT: 200, same content as myweb.freezenith.com

# --- GIT PUSH REDEPLOY ---

# 16. Push a change to the test repo (simulate via API or actual git push)
# GitHub webhook fires → new build triggered
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP1 | jq '.deployments | length'
# EXPECT: 2 (original + new deploy)

# 17. Verify new version is live
curl -s https://myapi.freezenith.com/api/version
# EXPECT: {"version":"v2"} (updated in the push)

# --- ROLLBACK ---

# 18. Rollback backend to previous version
curl -s -X POST $API/apps/$APP1/rollback \
  -H "Authorization: Bearer $TOKEN"
# EXPECT: 200, deploying previous image

# 19. Verify rollback
curl -s https://myapi.freezenith.com/api/version
# EXPECT: {"version":"v1"} (back to original)

# --- ENV VARS ---

# 20. Set environment variable
curl -s -X PUT $API/apps/$APP1/env \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"STRIPE_KEY":"sk_test_xxx","APP_ENV":"production"}'
# EXPECT: 200, app restarting

# 21. Verify env var is injected
curl -s https://myapi.freezenith.com/api/env-check
# EXPECT: {"APP_ENV":"production"} (STRIPE_KEY should NOT be exposed)

# --- WEBHOOKS ---

# 22. Create webhook
curl -s -X POST $API/webhooks \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"url":"https://hooks.slack.com/test","events":["deploy.success","deploy.failed"]}'
# EXPECT: 201

# --- MFA ---

# 23. Enable MFA
curl -s -X POST $API/auth/mfa/enable \
  -H "Authorization: Bearer $TOKEN"
# EXPECT: {"secret":"JBSWY3DPEHPK3PXP","qr_url":"otpauth://..."}

# --- CLEANUP ---

# 24. Delete all apps
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" $API/apps/$APP1
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" $API/apps/$APP2
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" $API/apps/$APP3
# EXPECT: all deleted, subdomains gone, DBs dropped
```

**Pass criteria:** All 24 steps pass. Pro limits enforced correctly. Custom domain works. Rollback works.

---

#### Scenario 3: Team User — SSO + Audit + Preview Deploys + Team Members

> Team plan user sets up SSO, invites team members, uses preview deployments, checks audit log.

```
API=https://api.freezenith.com/api/v1

# 1. Login as Team user
TOKEN=$(curl -s -X POST $API/auth/login \
  -d '{"email":"team-test@example.com","password":"TeamTest1234!"}' | jq -r '.token')

# --- SSO SETUP ---

# 2. Configure SSO (SAML)
curl -s -X POST $API/settings/sso \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"provider":"saml","entity_id":"https://idp.testcorp.com","sso_url":"https://idp.testcorp.com/sso","certificate":"-----BEGIN CERTIFICATE-----\nMIIC..."}'
# EXPECT: {"status":"configured","login_url":"https://freezenith.com/sso/testcorp"}

# 3. Test SSO login (simulate)
curl -s -X POST $API/auth/sso/callback \
  -d '{"saml_response":"<base64...>"}'
# EXPECT: JWT returned, user auto-provisioned in team

# --- TEAM MEMBERS ---

# 4. Invite team member
curl -s -X POST $API/team/invite \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"email":"dev@testcorp.com","role":"developer"}'
# EXPECT: {"status":"invited","role":"developer"}

# 5. Invite another (viewer)
curl -s -X POST $API/team/invite \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"email":"pm@testcorp.com","role":"viewer"}'
# EXPECT: invited

# 6. List team members
curl -s -H "Authorization: Bearer $TOKEN" $API/team | jq '.members | length'
# EXPECT: 3 (owner + 2 invited)

# 7. Verify role-based access — developer can deploy
DEV_TOKEN=$(curl -s -X POST $API/auth/login \
  -d '{"email":"dev@testcorp.com","password":"Dev1234!"}' | jq -r '.token')
curl -s -X POST $API/apps \
  -H "Authorization: Bearer $DEV_TOKEN" \
  -d '{"name":"dev-app","repo_url":"https://github.com/taikuri-infra/zenith-test-go","branch":"main"}'
# EXPECT: 201 created

# 8. Verify role-based access — viewer CANNOT deploy
VIEWER_TOKEN=$(curl -s -X POST $API/auth/login \
  -d '{"email":"pm@testcorp.com","password":"Pm1234!"}' | jq -r '.token')
curl -s -X POST $API/apps \
  -H "Authorization: Bearer $VIEWER_TOKEN" \
  -d '{"name":"viewer-app","repo_url":"https://github.com/user/repo","branch":"main"}'
# EXPECT: 403 "Viewer role cannot create apps"

# 9. Viewer CAN read apps
curl -s -H "Authorization: Bearer $VIEWER_TOKEN" $API/apps | jq '.apps | length'
# EXPECT: number >= 1

# --- PREVIEW DEPLOYMENTS ---

# 10. Open a PR on the repo (simulate webhook)
# GitHub sends pull_request event → Zenith creates preview deployment
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID/previews | jq '.[0]'
# EXPECT: {"pr":42,"url":"https://dev-app-pr-42.freezenith.com","status":"running"}

# 11. Verify preview URL works
curl -s https://dev-app-pr-42.freezenith.com/health
# EXPECT: 200

# 12. Merge PR → preview auto-deleted
# (simulate webhook: pull_request closed+merged)
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID/previews | jq 'length'
# EXPECT: 0

# --- SESSION MANAGEMENT ---

# 13. List active sessions
curl -s -H "Authorization: Bearer $TOKEN" $API/auth/sessions
# EXPECT: [{"id":"...","ip":"1.2.3.4","device":"Chrome/Mac","created_at":"...","current":true}, ...]

# 14. Revoke a session (force logout another device)
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" $API/auth/sessions/SESSION_ID
# EXPECT: 200, that session's token is now invalid

# --- DPA ---

# 15. Download DPA document
curl -s -H "Authorization: Bearer $TOKEN" $API/settings/dpa --output dpa.pdf
# EXPECT: PDF file, valid DPA document

# 16. Sign DPA
curl -s -X POST $API/settings/dpa/sign \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"signer_name":"CTO Test","signer_email":"cto@testcorp.com"}'
# EXPECT: {"status":"signed","signed_at":"2026-02-21T..."}

# --- AUDIT LOG ---

# 17. Check audit log (all actions we did should be logged)
AUDIT=$(curl -s -H "Authorization: Bearer $TOKEN" "$API/audit?limit=20")
echo $AUDIT | jq '.[0]'
# EXPECT: {"action":"dpa.signed","actor":"team-test@example.com","timestamp":"...","details":{...}}

# 18. Verify our previous actions appear in audit
echo $AUDIT | jq '[.[] | .action]'
# EXPECT: includes "sso.configured", "team.member_invited", "app.created",
#         "session.revoked", "dpa.signed" — all actions logged

# 19. Export audit log (CSV)
curl -s -H "Authorization: Bearer $TOKEN" "$API/audit/export?format=csv" --output audit.csv
# EXPECT: CSV file with all audit entries

# --- AUTO-SCALING ---

# 20. Check auto-scaling config
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID/scaling
# EXPECT: {"min_replicas":1,"max_replicas":5,"cpu_threshold":80}

# 21. Update scaling
curl -s -X PUT $API/apps/$APP_ID/scaling \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"min_replicas":2,"max_replicas":10}'
# EXPECT: 200

# --- SLA VERIFICATION ---

# 22. Check SLA status
curl -s -H "Authorization: Bearer $TOKEN" $API/settings/sla
# EXPECT: {"sla":"99.5%","current_uptime":"99.98%","incidents_this_month":0}

# --- TEAM LIMIT ---

# 23. Invite members up to 10 (Team limit)
# ... invite 7 more ...
# 11th invite:
curl -s -X POST $API/team/invite \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"email":"toomany@testcorp.com","role":"developer"}'
# EXPECT: 403 "Team plan allows 10 members"
```

**Pass criteria:** SSO works, RBAC enforced, preview deploys create/destroy, audit log captures everything, DPA downloadable, session management works.

---

#### Scenario 4: Free → Pro Upgrade Journey

> Free user hits limits, sees upgrade prompts, upgrades via Stripe, limits expand immediately.

```
API=https://api.freezenith.com/api/v1

# 1. Login as free user (already has 1 app deployed from Scenario 1)
TOKEN=$(curl -s -X POST $API/auth/login \
  -d '{"email":"testuser@example.com","password":"Test1234!"}' | jq -r '.token')

# --- HIT LIMITS ---

# 2. Try second app → blocked
curl -s -X POST $API/apps \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"second","repo_url":"https://github.com/user/repo","branch":"main"}'
# EXPECT: 403 {"error":"Free plan allows 1 app","upgrade_url":"/billing/upgrade"}

# 3. Try custom domain → blocked
curl -s -X POST $API/apps/$APP_ID/domains \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"domain":"myapp.com"}'
# EXPECT: 403 {"error":"Custom domains require Pro plan","upgrade_url":"/billing/upgrade"}

# 4. Check usage → at ceiling
curl -s -H "Authorization: Bearer $TOKEN" $API/usage
# EXPECT: {"apps":{"used":1,"limit":1,"percent":100},...}

# --- STRIPE CHECKOUT ---

# 5. Start upgrade flow
CHECKOUT=$(curl -s -X POST $API/billing/checkout \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"plan":"pro"}')
echo $CHECKOUT | jq '.checkout_url'
# EXPECT: "https://checkout.stripe.com/c/pay/cs_test_..."

# 6. Complete Stripe checkout (test mode card 4242...)
# Stripe webhook fires: checkout.session.completed

# 7. Verify plan upgraded
curl -s -H "Authorization: Bearer $TOKEN" $API/me | jq '.plan'
# EXPECT: "pro"

# --- LIMITS EXPANDED ---

# 8. Create second app → NOW works
curl -s -X POST $API/apps \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"second","repo_url":"https://github.com/user/repo","branch":"main"}'
# EXPECT: 201 created

# 9. Custom domain → NOW works
curl -s -X POST $API/apps/$APP_ID/domains \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"domain":"myapp.com"}'
# EXPECT: 200 {"status":"pending_verification"}

# 10. Check new limits
curl -s -H "Authorization: Bearer $TOKEN" $API/usage
# EXPECT: {"apps":{"used":2,"limit":5},"databases":{"used":1,"limit":3},...}

# 11. Verify app is NO LONGER sleeping (always-on)
sleep 1200  # wait 20 min (longer than 15 min sleep threshold)
curl -s https://test-api.freezenith.com/health
# EXPECT: 200 instantly (no cold start, no wake-up delay)

# --- BILLING ---

# 12. Check billing page
curl -s -H "Authorization: Bearer $TOKEN" $API/billing
# EXPECT: {"plan":"pro","price":"€29.00","next_billing":"2026-03-21","payment_method":"visa-4242"}

# 13. Check invoices
curl -s -H "Authorization: Bearer $TOKEN" $API/billing/invoices
# EXPECT: [{"id":"inv_...","amount":"€29.00","status":"paid","date":"2026-02-21"}]
```

**Pass criteria:** Limits block correctly, Stripe checkout works, plan upgrades instantly, limits expand, sleep mode disabled for Pro.

---

#### Scenario 5: Sleep Mode (Free Tier KEDA)

> Verify free apps sleep after 15 min and wake on request.

```
API=https://api.freezenith.com/api/v1

# 1. Deploy app on free plan (from Scenario 1)
# App is running at test-api.freezenith.com

# 2. Verify app is running
curl -s -o /dev/null -w "%{http_code}" https://test-api.freezenith.com/health
# EXPECT: 200

# 3. Check replicas
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID | jq '.replicas'
# EXPECT: 1

# 4. Wait 16 minutes (past 15 min idle threshold)
sleep 960

# 5. Check replicas — should be 0
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID | jq '.replicas'
# EXPECT: 0

# 6. Check status in dashboard
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID | jq '.status'
# EXPECT: "sleeping"

# 7. Send request to sleeping app — measure wake time
START=$(date +%s%N)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" https://test-api.freezenith.com/health)
END=$(date +%s%N)
WAKE_MS=$(( ($END - $START) / 1000000 ))
echo "Wake time: ${WAKE_MS}ms, HTTP: $HTTP_CODE"
# EXPECT: HTTP 200, wake time < 5000ms

# 8. First request might get loading page
curl -s -D - https://test-api.freezenith.com/health | head -5
# EXPECT: either 200 (fast wake) or 503 with Retry-After + Zenith loading page

# 9. Second request — app is warm now
curl -s https://test-api.freezenith.com/health
# EXPECT: 200 {"status":"ok"} (instant, no delay)

# 10. Check replicas — back to 1
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID | jq '.replicas'
# EXPECT: 1

# 11. Check status
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID | jq '.status'
# EXPECT: "running"

# 12. Verify DB data survived sleep (data persists)
curl -s https://test-api.freezenith.com/api/items
# EXPECT: [{"id":1,"name":"Widget","price":9.99}] (same data from Scenario 1)
```

**Pass criteria:** App sleeps after 15 min, wakes in <5 sec, data persists, loading page shown during wake.

---

#### Scenario 6: Admin Platform Management

> Admin verifies the full admin panel: users, apps, usage, infrastructure.

```
API=https://api.freezenith.com/api/v1

# 1. Login as admin
TOKEN=$(curl -s -X POST $API/auth/login \
  -d '{"email":"admin@freezenith.com","password":"AdminPass!"}' | jq -r '.token')

# --- DASHBOARD ---

# 2. Platform overview
curl -s -H "Authorization: Bearer $TOKEN" $API/admin/dashboard
# EXPECT: {"total_users":150,"active_apps":87,"total_revenue":"€2,340","server_count":3,"cpu_percent":62,"ram_percent":58}

# --- USER MANAGEMENT ---

# 3. List users
curl -s -H "Authorization: Bearer $TOKEN" "$API/admin/users?page=1&limit=20" | jq '.total'
# EXPECT: number of registered users

# 4. View specific user
curl -s -H "Authorization: Bearer $TOKEN" $API/admin/users/USER_ID
# EXPECT: {"email":"...","plan":"free","apps":1,"databases":1,"storage_mb":12,"created_at":"..."}

# 5. View user's apps
curl -s -H "Authorization: Bearer $TOKEN" $API/admin/users/USER_ID/apps
# EXPECT: list of user's apps with status, URL, resource usage

# 6. Suspend a user
curl -s -X POST -H "Authorization: Bearer $TOKEN" $API/admin/users/USER_ID/suspend
# EXPECT: 200, user's apps stopped, login blocked

# 7. Verify suspended user can't login
curl -s -X POST $API/auth/login \
  -d '{"email":"testuser@example.com","password":"Test1234!"}'
# EXPECT: 403 "Account suspended"

# 8. Verify suspended user's app is down
curl -s -o /dev/null -w "%{http_code}" https://test-api.freezenith.com/health
# EXPECT: 503

# 9. Reactivate user
curl -s -X POST -H "Authorization: Bearer $TOKEN" $API/admin/users/USER_ID/activate
# EXPECT: 200, apps restart, login allowed

# --- INFRASTRUCTURE ---

# 10. Server status
curl -s -H "Authorization: Bearer $TOKEN" $API/admin/infrastructure
# EXPECT: {"servers":[{"id":"srv1","cpu_percent":65,"ram_percent":58,"pods":42,"status":"ready"},...],"autoscaler":{"enabled":true,"min":2,"max":10,"current":3}}

# 11. Platform resource usage
curl -s -H "Authorization: Bearer $TOKEN" $API/admin/dashboard/usage
# EXPECT: {"total_cpu_cores":12.5,"total_ram_gb":45.2,"total_storage_gb":120,"customers_reporting":87}

# --- BILLING OVERVIEW ---

# 12. MRR and revenue
curl -s -H "Authorization: Bearer $TOKEN" $API/admin/billing
# EXPECT: {"mrr":"€2,340","active_subscriptions":80,"free_users":120,"pro_users":65,"team_users":12,"enterprise_users":3,"churn_rate":2.1}

# 13. Failed payments
curl -s -H "Authorization: Bearer $TOKEN" $API/admin/billing/failed
# EXPECT: [{"user":"...","amount":"€29","failed_at":"...","retry_count":2}]

# --- OVERRIDE ---

# 14. Override user plan limits
curl -s -X PUT -H "Authorization: Bearer $TOKEN" $API/admin/users/USER_ID/limits \
  -d '{"max_apps":10,"max_databases":5,"note":"Special deal for early adopter"}'
# EXPECT: 200

# 15. Verify override in audit log
curl -s -H "Authorization: Bearer $TOKEN" "$API/admin/audit?limit=1" | jq '.[0]'
# EXPECT: {"action":"admin.limits_override","actor":"admin@freezenith.com","target":"USER_ID",...}
```

**Pass criteria:** Admin sees all users/apps/infra, can suspend/activate, billing overview correct, audit trail complete.

---

#### Scenario 7: Git Push → Auto-Redeploy + Build Failure

> Verify the full CI/CD pipeline: push → build → deploy, and graceful failure handling.

```
API=https://api.freezenith.com/api/v1

# 1. App is running (test-api.freezenith.com)
curl -s https://test-api.freezenith.com/api/version
# EXPECT: {"version":"v1"}

# --- SUCCESSFUL REDEPLOY ---

# 2. Push a code change to GitHub (git push with version bump)
# GitHub webhook POST arrives at API

# 3. Check build started
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID | jq '.status'
# EXPECT: "building"

# 4. Stream build logs
curl -s -N -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID/logs/build
# EXPECT: streaming output:
#   Cloning repository...
#   Detected framework: go
#   Building image...
#   Step 1/8: FROM golang:1.22-alpine
#   ...
#   Successfully built a1b2c3d4
#   Deploying...
#   Deployment complete.

# 5. App transitions: building → deploying → running
# (poll or wait for SSE event)

# 6. Verify new version live (zero-downtime rolling update)
curl -s https://test-api.freezenith.com/api/version
# EXPECT: {"version":"v2"}

# 7. Check deployment history
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID/deployments
# EXPECT: [
#   {"id":"dep2","image_tag":"abc1234","status":"active","created_at":"..."},
#   {"id":"dep1","image_tag":"def5678","status":"superseded","created_at":"..."}
# ]

# --- FAILED BUILD ---

# 8. Push broken code (syntax error in main.go)
# GitHub webhook fires

# 9. Build fails
curl -s -H "Authorization: Bearer $TOKEN" $API/apps/$APP_ID | jq '.status'
# EXPECT: "running" (NOT "failed" — old version keeps running)

# 10. Check build log shows error
curl -s -H "Authorization: Bearer $TOKEN" "$API/apps/$APP_ID/deployments?limit=1" | jq '.[0]'
# EXPECT: {"status":"failed","error":"build failed: syntax error at main.go:15","image_tag":null}

# 11. App still serves old version
curl -s https://test-api.freezenith.com/api/version
# EXPECT: {"version":"v2"} (still the last successful build)

# 12. Dashboard shows failed deploy notification
curl -s -H "Authorization: Bearer $TOKEN" $API/notifications
# EXPECT: [{"type":"deploy_failed","app":"myapi","message":"Build failed: syntax error","timestamp":"..."}]

# --- WEBHOOK NOTIFICATION ---

# 13. If webhook configured, verify it was called
# (check webhook delivery log)
curl -s -H "Authorization: Bearer $TOKEN" $API/webhooks/deliveries
# EXPECT: [{"event":"deploy.failed","url":"https://hooks.slack.com/test","status":200,"timestamp":"..."}]
```

**Pass criteria:** Successful push redeploys with zero downtime. Failed build keeps old version. Build logs available. Notifications sent.

---

#### Scenario 8: Self-Hosted Open-Source (docker-compose)

> Verify the open-source version works completely offline on a fresh machine.

```
# 1. Clone repo
git clone https://github.com/taikuri-infra/Zenith
cd Zenith

# 2. Start platform
docker compose up -d
# EXPECT: 4 containers start (api, dashboard, postgres, traefik)

# 3. Wait for healthy
docker compose ps
# EXPECT: all 4 "healthy" within 2 min

# 4. Open dashboard
curl -s -o /dev/null -w "%{http_code}" http://localhost:3100
# EXPECT: 200

# 5. Login with defaults
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -d '{"email":"admin@zenith.local","password":"changeme"}' | jq -r '.token')
# EXPECT: valid token

# 6. First-run: prompted to change password
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/me | jq '.must_change_password'
# EXPECT: true

# 7. Change password
curl -s -X PUT -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/auth/password \
  -d '{"old_password":"changeme","new_password":"MyNewPass1234!"}'
# EXPECT: 200

# 8. Deploy an app (same flow as SaaS but local)
APP=$(curl -s -X POST http://localhost:8080/api/v1/apps \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"myapp","repo_url":"https://github.com/taikuri-infra/zenith-test-go","branch":"main"}' \
  | jq -r '.id')
# EXPECT: app created

# 9. Wait for running
# (poll status)

# 10. Access app on local port
curl -s http://localhost:3200/health  # or whatever port is assigned
# EXPECT: 200

# 11. Create database
curl -s -X POST http://localhost:8080/api/v1/apps/$APP/databases \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"type":"postgresql"}'
# EXPECT: connection string with localhost:5432

# 12. Verify no billing pages
curl -s -o /dev/null -w "%{http_code}" http://localhost:3100/billing
# EXPECT: 404 (billing route doesn't exist in standalone mode)

# 13. Verify no sleep mode (always-on in standalone)
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/apps/$APP | jq '.sleep_mode'
# EXPECT: null or false

# 14. Verify no multi-user (no invite endpoint)
curl -s -X POST http://localhost:8080/api/v1/team/invite \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"email":"other@test.com"}'
# EXPECT: 404 (team endpoint not registered in standalone mode)

# 15. Shutdown cleanly
docker compose down
# EXPECT: all containers stopped, data persisted in volume

# 16. Restart — data survives
docker compose up -d
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -d '{"email":"admin@zenith.local","password":"MyNewPass1234!"}' | jq -r '.token')
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/apps | jq '.apps | length'
# EXPECT: 1 (app still exists)
```

**Pass criteria:** docker-compose up works, deploy app locally, no billing/sleep/multi-user, data persists across restart.

---

### Summary: 8 E2E Scenarios, 134 Verification Points

| # | Scenario | Steps | Covers |
|---|----------|-------|--------|
| 1 | Free User — Full Stack | 24 | signup, deploy, DB, auth, S3, monitoring, limits |
| 2 | Pro User — Multi-App | 24 | 3 apps, 3 DBs, custom domain, rollback, env vars, MFA, webhooks |
| 3 | Team User — SSO + Audit | 23 | SSO, RBAC, preview deploys, sessions, DPA, audit log, auto-scaling |
| 4 | Free → Pro Upgrade | 13 | limit blocks, Stripe checkout, plan upgrade, limits expand |
| 5 | Sleep Mode (KEDA) | 12 | sleep after 15 min, wake <5 sec, data persists |
| 6 | Admin Management | 15 | users, suspend/activate, infra, billing overview, audit |
| 7 | Git Push + Build Fail | 13 | CI/CD pipeline, zero-downtime, failed build handling |
| 8 | Self-Hosted (docker-compose) | 16 | offline setup, no billing, no sleep, data persistence |
| **Total** | | **140** | |

### Phase-level Unit & Integration Tests (kept for reference)

| Phase | Tests | Type |
|-------|-------|------|
| Phase 2: App Deploy | T2-01 to T2-17 (17 tests) | Framework detection, webhook validation, build pipeline |
| Phase 3: Services | T3-01 to T3-13 (13 tests) | DB CRUD, auth flow, S3 limits |
| Phase 4: KEDA | T4-01 to T4-11 (11 tests) | Scale-to-zero, ResourceQuota, plan enforcement |
| Phase 5: Autoscaler | T5-01 to T5-07 (7 tests) | Scale up/down, budget cap, node join/leave |
| Phase 6: Billing | T6-01 to T6-07 (7 tests) | Stripe webhooks, subscription lifecycle |
| Phase 7: Open-Source | T7-01 to T7-05 (5 tests) | docker-compose, standalone mode |
| **Total** | **60 unit/integration tests** | |

**Grand total: 140 E2E + 60 unit/integration = 200 test points**

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

---

## Deliverables — What You Get When This Plan Is Complete

### For Your Customers (SaaS — freezenith.com)

**A fully working cloud platform where developers:**

1. **Sign up** at freezenith.com → get a dashboard instantly
2. **Connect GitHub** → paste repo URL, pick branch
3. **Deploy in 3 minutes** → Zenith detects framework (Go/Next.js/Python/Rails/Django), builds, deploys
4. **Get a live URL** → `myapp.freezenith.com` with SSL, instantly shareable
5. **One-click database** → PostgreSQL provisioned, connection string injected as env var
6. **One-click auth** → built-in sign up/login for their app's users, SDK snippet ready
7. **S3 storage** → file upload/download, access keys provided
8. **Container registry** → images stored, pull commands shown
9. **Monitoring** → CPU, RAM, request count, logs — all in dashboard
10. **Team collaboration** → invite members, assign roles (owner/admin/dev/viewer)
11. **Git push = redeploy** → push to main, webhook fires, new version live in 3 min, zero downtime
12. **Rollback** → one click to go back to previous version if something breaks
13. **Custom domain** (Pro+) → CNAME their domain, SSL auto-provisioned
14. **Free tier sleeps** → apps scale to zero after 15 min, wake in <5 sec on request
15. **Upgrade via Stripe** → self-service, limits expand instantly
16. **SSO** (Team+) → SAML/OIDC, connect to Azure AD/Okta/Google Workspace
17. **Audit log** (Team+) → every action logged, exportable CSV
18. **Preview deploys** (Team+) → every PR gets a temporary URL

### For You (Business Owner)

1. **Admin dashboard** → total users, MRR, active apps, server utilization
2. **User management** → view any user's apps/DBs/usage, suspend/activate, override limits
3. **Billing overview** → MRR, active subscriptions, failed payments, churn rate, revenue per plan
4. **Infrastructure view** → server count, CPU/RAM per server, autoscaler status, cost vs budget
5. **Audit trail** → everything every user and admin does, logged and exportable
6. **Auto-scaling** → servers scale 2→10 based on demand, budget cap at €450/month
7. **Stripe integration** → subscriptions auto-renew, failed payments handled with grace period
8. **4 revenue streams** → Free (funnel) / Pro €29 (individuals) / Team €199 (startups) / Enterprise (custom)
9. **B2B ready** → white-label, dedicated infra, compliance dashboard, SLA, DPA

### For Open-Source Community

1. **`docker compose up`** → full platform running locally in <2 minutes
2. **README** with screenshots, quick start, architecture diagram
3. **AGPLv3 license** (same as GitLab)
4. **One-liner install** → `curl -fsSL https://get.freezenith.com | bash`
5. **GitHub repo** → clean, badges, issues templates, CI, contributing guide
6. **Standalone mode** → single user, no billing, no sleep, no multi-tenant — just works

### Numbers

| Metric | Value |
|--------|-------|
| Total features | 47 across 6 categories |
| Plan tiers | 4 (Free / Pro / Team / Enterprise) |
| API endpoints | ~60 (auth, apps, databases, storage, billing, admin, webhooks) |
| Dashboard pages | ~25 (deploy, apps, databases, auth, storage, monitoring, billing, settings, admin) |
| Supported frameworks | 6 (Next.js, Go, Python, Django, Rails, static + any Dockerfile) |
| E2E test scenarios | 8 scenarios, 140 verification points |
| Unit/integration tests | 60 test cases |
| Total test coverage | 200 test points |

---

## Verification Process — How I Ensure Zero Bugs

### Per-Phase Verification

Every phase follows this process before moving to the next:

```
1. WRITE CODE
   → Go handlers, tests, models, migrations
   → Next.js pages, components, API client

2. UNIT TESTS (Go)
   → go test ./... — MUST pass 100%
   → Test every handler with memory repository
   → Test edge cases: invalid input, missing auth, over limit, duplicate

3. BUILD CHECK
   → go build ./... — compiles with zero errors
   → npm run build — Next.js builds with zero errors
   → Docker build — images build successfully

4. DEPLOY TO SERVER
   → ssh ghasi "cd /opt/zenith && bash scripts/deploy.sh"
   → All pods running, no CrashLoopBackoff
   → Health endpoints respond 200

5. E2E VERIFICATION (curl scripts)
   → Run the relevant scenario(s) against live API
   → Every EXPECT line must match actual response
   → If any step fails → fix → redeploy → re-verify from step 1

6. CROSS-SCENARIO CHECK
   → After Phase 2: Scenario 1 (full stack) + Scenario 7 (git push) must pass
   → After Phase 3: Scenario 1 + 2 (multi-app + DB) must pass
   → After Phase 4: Scenario 4 (upgrade) + 5 (sleep) must pass
   → After Phase 6: Scenario 4 (Stripe) must pass
   → After Phase 7: Scenario 8 (docker-compose) must pass
   → After Phase 8: ALL 8 scenarios must pass
```

### Phase → Scenario Mapping

| Phase | After completion, these scenarios MUST pass |
|-------|---------------------------------------------|
| Phase 2: App Deploy | Scenario 1 (steps 1-6, deploy only), Scenario 7 |
| Phase 3: Services | Scenario 1 (full), Scenario 2 (steps 1-12) |
| Phase 4: KEDA + Plans | Scenario 1 (full), Scenario 4, Scenario 5 |
| Phase 5: Autoscaler | Scenario 6 (infra section) |
| Phase 6: Billing | Scenario 2 (full), Scenario 3 (full), Scenario 4 (full), Scenario 6 (full) |
| Phase 7: Open-Source | Scenario 8 |
| Phase 8: Launch | ALL 8 scenarios — full regression |

### Bug Prevention Rules

1. **No code without tests.** Every handler gets a `_test.go` file. No exceptions.
2. **No deploy without green tests.** `go test ./...` must pass before `deploy.sh` runs.
3. **No phase transition without E2E.** The curl scenarios run against live API before marking a phase complete.
4. **Regression testing.** After each phase, ALL previous scenarios re-run. Nothing breaks.
5. **Error responses are tested too.** Not just happy path — test 403s, 404s, 400s, rate limits.
6. **Build failures are safe.** Failed builds never take down running apps (tested in Scenario 7).
7. **Data persistence verified.** After every deploy, check existing data survived (no migration breaks).
8. **Demo mode stays working.** After every change, demo-ms.freezenith.com still works with mock data.

### Automated Test Script (built during Phase 2)

```bash
#!/bin/bash
# scripts/e2e-test-full.sh — runs ALL scenarios against live API

API=https://api.freezenith.com/api/v1
PASS=0
FAIL=0

run_test() {
  local name=$1
  local expected=$2
  local actual=$3
  if [ "$expected" = "$actual" ]; then
    echo "  ✅ $name"
    PASS=$((PASS + 1))
  else
    echo "  ❌ $name — expected: $expected, got: $actual"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== Scenario 1: Free User Full Stack ==="
# ... all 24 steps ...

echo "=== Scenario 2: Pro User Multi-App ==="
# ... all 24 steps ...

# ... scenarios 3-8 ...

echo ""
echo "========================="
echo "  PASS: $PASS"
echo "  FAIL: $FAIL"
echo "  TOTAL: $((PASS + FAIL))"
echo "========================="

[ $FAIL -eq 0 ] && echo "🟢 ALL TESTS PASSED" || echo "🔴 SOME TESTS FAILED"
exit $FAIL
```

This script runs after every deploy. 140 checks. If anything fails, we know exactly what broke and where.
