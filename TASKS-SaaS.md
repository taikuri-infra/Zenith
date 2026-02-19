# Zenith SaaS — Enterprise Cloud Transformation

> DoTech operates Zenith as a managed multi-tenant cloud platform on Hetzner.
> Customers buy "Zenith Enterprise" plans with guaranteed resource ceilings.
> Resources are provisioned on-demand (not pre-built). CAPI creates a dedicated cluster per customer.
> In the future, an open-core version is extracted so customers can self-host Zenith with their own Mission Control.

---

## Progress Summary

| Phase | Description | Tasks | Done | Status |
|-------|-------------|-------|------|--------|
| Pre | Foundation (Auth, API scaffold, Deploy, IaC) | 24 | 24 | **COMPLETE** |
| 0 | PostgreSQL + Persistent State | 18 | 0 | NOT STARTED |
| 1 | Customer Management in Admin | 16 | 0 | NOT STARTED |
| 2 | CAPI Cluster Provisioning | 20 | 0 | NOT STARTED |
| 3 | Resource Metering & Limits | 11 | 0 | NOT STARTED |
| 4 | Billing (Stripe + Fairbroker) | 11 | 0 | NOT STARTED |
| 5 | Customer Onboarding Automation | 5 | 0 | NOT STARTED |
| 6 | Open-Core Extraction (Future) | 7 | 0 | NOT STARTED |
| **Total** | | **112** | **24** | **21%** |

---

## Business Model

**We (DoTech) ARE the management plane.** Customers come to us, pick a plan, and get:
- A dedicated Kubernetes cluster (CAPI-provisioned on Hetzner)
- A Web Platform dashboard at `cloud.{customer-domain}`
- Guaranteed resource ceilings (RAM, CPU, S3, DB storage, volumes)
- Resources created on-demand — we only provision what they actually use
- Hetzner cluster autoscaler underneath scales our infrastructure as needed

**Example Enterprise Plan (~€2,000–4,000/month):**

| Resource | Ceiling | Our Hetzner Cost |
|----------|---------|-----------------|
| CPU | 160 cores | 10x CPX servers = ~€400/mo |
| RAM | 320 GB | (included in servers) |
| S3 Storage | 20 TB | ~€100/mo |
| DB + Volumes | 600 GiB | ~€33/mo |
| Load Balancers | 4 | ~€20/mo |
| **Total infra** | | **~€553/mo** |
| **Sell for** | | **€2,000–4,000/mo** |
| **Margin** | | **~70–85%** |

Customers don't hit 100% of ceilings, so actual cost is lower. Multi-customer pooling makes margins even better.

**Billing:** Stripe (international) + Toman/IRR (via Fairbroker — details TBD)

**Future — Open-Core:**
Extract a free self-hosted Zenith where customers install their own Mission Control and manage their own clusters. This becomes the marketing engine. The SaaS is the premium offering.

---

## Architecture (SaaS Mode)

```
┌─────────────────────────────────────────────────────────────────┐
│                    DoTech Management Plane                       │
│                  (Hetzner server, our control)                   │
│                                                                  │
│  ┌──────────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  Zenith Admin     │  │  Zenith API  │  │  PostgreSQL      │  │
│  │  (current MC,     │  │  (multi-     │  │  (persistent     │  │
│  │   rebranded)      │  │   tenant)    │  │   state)         │  │
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
│  │  (provisions     │  │  (scales server pool up/down          │ │
│  │   clusters)      │  │   based on total demand)              │ │
│  └──────────────────┘  └──────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
          │                          │                    │
          ▼                          ▼                    ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ Customer A       │  │ Customer B       │  │ Customer C       │
│ Cluster          │  │ Cluster          │  │ Cluster          │
│ (CAPI-managed)   │  │ (CAPI-managed)   │  │ (CAPI-managed)   │
│                  │  │                  │  │                  │
│ - Zenith Op.     │  │ - Zenith Op.     │  │ - Zenith Op.     │
│ - Web Platform   │  │ - Web Platform   │  │ - Web Platform   │
│ - Apps, DBs      │  │ - Apps, DBs      │  │ - Apps, DBs      │
│ - S3, Auth, etc. │  │ - S3, Auth, etc. │  │ - S3, Auth, etc. │
│                  │  │                  │  │                  │
│ cloud.cust-a.com │  │ cloud.cust-b.io  │  │ cloud.cust-c.dev │
└──────────────────┘  └──────────────────┘  └──────────────────┘

Future (open-core):
  Customer installs their own MC → ms.{domain}
  Customer manages their own Zenith clusters
  Our SaaS Admin = premium managed version of the same thing
```

### Key Distinction: Admin vs Mission Control

| | Zenith Admin (SaaS) | Mission Control (Open-Core, future) |
|---|---|---|
| **Who runs it** | DoTech | Customer |
| **Purpose** | Manage all customers, billing, infra | Manage own clusters |
| **URL** | admin.freezenith.com | ms.{customer-domain} |
| **Code** | `apps/mission-control/` (enhanced) | Same codebase, different mode |
| **Auth** | DoTech staff only | Customer's own auth |
| **Sees** | All customers, all clusters | Only own clusters |
| **When** | Now (priority) | Future (open-core release) |

The same `apps/mission-control/` codebase serves both roles. A config flag (`ZENITH_MODE=saas` vs `ZENITH_MODE=standalone`) determines which features are available. For now we build SaaS mode only, but keep the architecture clean so standalone extraction is straightforward later.

---

## IaC Approach

> **Everything is code. No manual SSH. No ClickOps.**

| Layer | Tool | What It Manages |
|-------|------|----------------|
| **Cloud Infra** | Terraform / CDKTF | Hetzner servers, volumes, LBs, networks, firewalls, object storage |
| **DNS** | Terraform (Cloudflare provider) | DNS records for all domains (platform + customer) |
| **Server Config** | Ansible | k3s install, system packages, certs, users, firewall rules |
| **K8s Platform** | Helm + Kustomize | Zenith platform services (API, MC, Web, monitoring, operators) |
| **K8s Customers** | CAPI + CAPH (via API) | Customer cluster lifecycle (create, scale, upgrade, delete) |
| **Secrets** | Ansible Vault / SOPS | API tokens, DB passwords, Stripe keys, Cloudflare tokens |
| **CI/CD** | GitHub Actions | Build images, run tests, deploy to server |

```
infra/
├── terraform/              # Cloud resources (Hetzner + Cloudflare DNS)
│   ├── modules/
│   │   ├── management/     # Management plane server(s)
│   │   ├── dns/            # DNS records (platform + per-customer)
│   │   ├── network/        # VPC, firewalls, SSH keys
│   │   └── storage/        # Volumes, S3 buckets
│   ├── environments/
│   │   ├── production/     # Production tfvars + state
│   │   └── staging/        # Staging tfvars + state
│   └── main.tf
├── ansible/                # Server configuration
│   ├── playbooks/
│   │   ├── setup-server.yml       # Base OS, packages, users
│   │   ├── install-k3s.yml        # k3s + CAPI + CAPH
│   │   ├── deploy-platform.yml    # Build + deploy Zenith
│   │   └── setup-postgres.yml     # PostgreSQL on k8s
│   ├── inventory/
│   │   ├── production.yml
│   │   └── staging.yml
│   └── roles/
├── helm/                   # (existing) Helm charts
└── k8s/                    # (existing) Raw manifests for simple resources
```

---

## Pre-SaaS Foundation (COMPLETE)

> These are foundational pieces already built that SaaS phases depend on.

### API Server (`services/api/`)
- [x] **PRE-01** Go API server with Fiber framework, structured routes, error handling
- [x] **PRE-02** JWT authentication: login, register, refresh endpoints (`handlers/auth.go`)
- [x] **PRE-03** User store with bcrypt password hashing (`store/user_store.go`, in-memory)
- [x] **PRE-04** Auth middleware: JWT validation, API key header, role-based access (Owner/Admin/Developer/Viewer)
- [x] **PRE-05** CRD-based resource architecture: Projects, Apps, Databases, Storage (in-memory `k8s.MemoryClient`)
- [x] **PRE-06** CAPI client wrapper for cluster CRUD operations (in-memory `capi.MemoryStore`)
- [x] **PRE-07** Admin handlers: dashboard stats, clusters, tenants, modules, audit, settings, infra, state
- [x] **PRE-08** Config from env vars: PORT, JWT_SECRET, ADMIN_EMAIL/PASSWORD, CORS_ORIGINS, etc.
- [x] **PRE-09** Dockerfile with multi-stage build, non-root user, port 8080

### Mission Control (`apps/mission-control/`)
- [x] **PRE-10** Login page (`/login`) with email/password form, error handling, loading states
- [x] **PRE-11** `useAuth()` hook: JWT token parsing, localStorage persistence, demo mode bypass
- [x] **PRE-12** API client with auth methods: login, logout, refresh, token management (`api.ts`)
- [x] **PRE-13** Protected shell: auth gating, redirect to `/login` if not authenticated
- [x] **PRE-14** Demo mode: `NEXT_PUBLIC_DEMO_MODE=true` build-time flag, `demoApi` with mock data
- [x] **PRE-15** Full page set: Dashboard, Clusters, Tenants, Modules, Updates, Infrastructure, State, Audit, Settings

### Infrastructure & IaC
- [x] **PRE-16** K8s manifests (`k8s/*.yaml`): namespaces, deployments, services, certificates, IngressRoutes
- [x] **PRE-17** `scripts/deploy.sh`: Full pipeline — git pull, build 6 images, import to k3s, apply manifests, rollout
- [x] **PRE-18** Terraform DNS (`terraform/`): Cloudflare provider, 7 A records (freezenith.com + embermind.app)
- [x] **PRE-19** `scripts/cloudflare-dns.sh`: Quick DNS CRUD via Cloudflare API (create/delete/status)
- [x] **PRE-20** `scripts/e2e-test.sh`: Post-deploy validation (DNS, HTTPS, redirects, SSL, content, API health)
- [x] **PRE-21** Helm chart `helm/zenith/`: API + Operator + Auth + Kong + OTEL + RBAC + service mesh templates
- [x] **PRE-22** Helm chart `helm/monitoring/`: kube-prometheus-stack + Loki + Promtail + alerting rules
- [x] **PRE-23** cert-manager with letsencrypt-prod ClusterIssuer, HTTP-01 solver
- [x] **PRE-24** Traefik 3.5.1 IngressRoutes with HTTP→HTTPS redirect middleware

---

## Phase 0: PostgreSQL + Persistent State

> **Goal:** Replace all in-memory stores with PostgreSQL. Nothing else works without this.
> **Status:** NOT STARTED (0/18)

### Tasks — IaC (Provision PostgreSQL)

- [ ] **S0-01** Terraform: Add Hetzner volume for PostgreSQL persistent data (or use CNPG on existing server)
- [ ] **S0-02** Ansible playbook `setup-postgres.yml`: Deploy PostgreSQL on k3s via CNPG operator or StatefulSet with PVC
  - Persistent volume backed by Hetzner volume
  - Automated backups to Hetzner S3
  - Secrets managed via Ansible Vault / SOPS
- [ ] **S0-03** Ansible: Wire `DATABASE_URL` into k8s secrets, update `k8s/api.yaml` env

### Tasks — Database Schema (golang-migrate)

- [ ] **S0-04** Add golang-migrate for schema migrations (embedded in API binary)
- [ ] **S0-05** Migration 001: `users`, `sessions`, `api_keys` tables
- [ ] **S0-06** Migration 002: `customers` (id, name, domain, plan_id, status, created_at, ...)
- [ ] **S0-07** Migration 003: `plans` (id, name, cpu_limit, ram_limit, s3_limit, db_storage_limit, volume_limit, lb_limit, price_cents, currency, billing_cycle)
- [ ] **S0-08** Migration 004: `clusters` (id, customer_id, name, region, k8s_version, status, capi_cluster_name, node_count, created_at)
- [ ] **S0-09** Migration 005: `resource_usage` (id, customer_id, cluster_id, resource_type, amount, unit, recorded_at) — time-series metering
- [ ] **S0-10** Migration 006: `invoices` (id, customer_id, plan_id, amount, currency, status, stripe_invoice_id, period_start, period_end)
- [ ] **S0-11** Migration 007: `audit_log` (id, actor, action, resource_type, resource_id, customer_id, metadata jsonb, created_at)
- [ ] **S0-12** Migration 008: `modules`, `settings`, `platform_state` tables

### Tasks — Go API (Replace In-Memory Stores)

- [ ] **S0-13** Add pgx driver + connection pool in API startup
- [ ] **S0-14** Replace `store.UserStore` with PostgreSQL-backed implementation
- [ ] **S0-15** Replace `k8s.MemoryClient` with real K8s client (client-go, in-cluster config)
- [ ] **S0-16** Replace `capi.MemoryStore` with PostgreSQL for settings/modules/audit
- [ ] **S0-17** API auto-migrates on startup (golang-migrate `Up()`)
- [ ] **S0-18** Update Dockerfile: ensure migrate files embedded in binary

### Definition of Done
- PostgreSQL provisioned via Ansible (not manual SSH)
- API server restarts without losing data
- Users, sessions, settings persist across deployments
- Real K8s client connects to k3s cluster API
- `ansible-playbook setup-postgres.yml` is the only command needed

---

## Phase 1: Customer Management in Admin Panel

> **Goal:** Admin (DoTech staff) can create, view, and manage customer accounts from the MC dashboard.
> **Status:** NOT STARTED (0/16)

### Tasks — API Endpoints

- [ ] **S1-01** `POST /api/v1/admin/customers` — create customer
  - Body: `{ name, domain, plan_id, contact_email, contact_name }`
- [ ] **S1-02** `GET /api/v1/admin/customers` — list all customers
  - Response: customer list with plan, cluster status, resource usage summary
- [ ] **S1-03** `GET /api/v1/admin/customers/:id` — get customer detail
  - Response: full customer profile, cluster info, usage, invoices
- [ ] **S1-04** `PUT /api/v1/admin/customers/:id` — update customer (name, plan, status)
- [ ] **S1-05** `POST /api/v1/admin/customers/:id/suspend` — suspend customer
- [ ] **S1-06** `POST /api/v1/admin/customers/:id/activate` — reactivate customer
- [ ] **S1-07** `DELETE /api/v1/admin/customers/:id` — delete customer (+ cluster teardown)
- [ ] **S1-08** `GET /api/v1/admin/plans` — list available plans
- [ ] **S1-09** `POST /api/v1/admin/plans` — create plan
  - Body: `{ name, cpu_cores, ram_gb, s3_tb, db_storage_gb, volume_gb, lb_count, price_cents, currency, billing_cycle }`
- [ ] **S1-10** `PUT /api/v1/admin/plans/:id` — update plan

### Tasks — MC Frontend

- [ ] **S1-11** MC page: `/customers` — customer list table
  - Columns: Name, Domain, Plan, Cluster Status, CPU/RAM usage bars, Monthly Cost, Status
  - Actions: + New Customer, search/filter
- [ ] **S1-12** MC page: `/customers/[id]` — customer detail
  - Sections: Profile, Cluster info, Resource usage gauges (CPU/RAM/S3/DB vs ceiling), Recent activity, Invoice history, Actions (suspend/activate/upgrade plan/delete)
- [ ] **S1-13** MC page: `/customers/new` — create customer wizard
  - Steps: Company info → Select plan → Configure domain → Review → Create
- [ ] **S1-14** MC page: `/plans` — plan management
  - Table: Plan name, Resources (CPU/RAM/S3/DB), Price, Active customers count
  - Actions: + New Plan, Edit, Archive
- [ ] **S1-15** MC sidebar: Add "Customers" and "Plans" nav items (above Clusters)
- [ ] **S1-16** MC dashboard (`/`): Replace "Tenants" with "Customers" card + stats
  - Show: total customers, MRR (monthly recurring revenue), new this month

### Definition of Done
- DoTech admin can create a customer and see them listed
- Customer detail page shows plan, domain, status
- Plan CRUD works

---

## Phase 2: CAPI Cluster Provisioning Per Customer

> **Goal:** Creating a customer automatically provisions a dedicated Kubernetes cluster on Hetzner via CAPI.
> **Status:** NOT STARTED (0/20)

### Tasks — IaC (Management Plane Setup)

- [ ] **S2-01** Ansible playbook `install-capi.yml`: Install CAPI + CAPH on management plane k3s
  - clusterctl init, provider versions pinned
  - Hetzner token injected via Ansible Vault / SOPS
- [ ] **S2-02** Terraform module `modules/network/`: Hetzner VPC + firewall rules for customer clusters
  - Private network per customer, firewall templates, SSH key management
- [ ] **S2-03** Terraform module `modules/dns/`: Dynamic DNS for customer domains
  - Cloudflare provider, per-customer records (`cloud.{domain}`, `ms.{domain}`)
  - Called from Go API via Terraform Cloud API or `go-cloudflare` SDK
- [ ] **S2-04** CAPH ClusterTemplate (versioned in `infra/capi-templates/`)
  - Config: k3s, Hetzner CPX servers, private network, firewall
  - Parameterized: customer name, node count, server type, region
- [ ] **S2-05** Ansible role for customer cluster bootstrap
  - Install Zenith Operator + Web Platform + cert-manager into new cluster
  - Idempotent, can re-run on existing clusters

### Tasks — Go API (Cluster Lifecycle)

- [ ] **S2-06** API: When customer is created (S1-01), trigger CAPI cluster creation
  - Generate CAPI Cluster manifest from template
  - Apply to management cluster via client-go
  - Store `capi_cluster_name` in customers table
- [ ] **S2-07** API: Watch/poll CAPI Cluster status, update `customers.cluster_status`
- [ ] **S2-08** API: When cluster is ready, install Zenith Operator into customer cluster
  - Use Helm client-go or raw manifests
- [ ] **S2-09** API: When cluster is ready, install Web Platform into customer cluster
  - Deploy generic image with runtime env vars (customer domain, API URL)
- [ ] **S2-10** API: Configure DNS for customer domain (Cloudflare Go SDK)
  - `cloud.{customer-domain}` → customer cluster ingress IP
  - (future) `ms.{customer-domain}` → customer MC (open-core)
- [ ] **S2-11** API: Issue TLS certificates for customer domain (cert-manager in customer cluster)
- [ ] **S2-12** API: `POST /api/v1/admin/customers/:id/cluster/scale`
  - Body: `{ nodes: N }` — scale worker nodes up/down via CAPI MachineDeployment
- [ ] **S2-13** API: `POST /api/v1/admin/customers/:id/cluster/upgrade`
  - Body: `{ k8s_version }` — upgrade customer cluster K8s via CAPI

### Tasks — MC Frontend

- [ ] **S2-14** MC `/customers/[id]`: Show cluster provisioning progress
  - States: Pending → Provisioning → Installing Zenith → Configuring DNS → Ready
- [ ] **S2-15** MC `/customers/[id]`: Show cluster detail (nodes, K8s version, health)

### Tasks — Cluster Teardown & Scaling

- [ ] **S2-16** Implement cluster teardown on customer deletion
  - Delete CAPI Cluster → Hetzner servers auto-cleaned
  - Remove DNS records via Cloudflare API
  - Archive customer data in PostgreSQL
- [ ] **S2-17** Hetzner autoscaler integration
  - Monitor total resource demand across all customer clusters
  - When approaching capacity, scale up the Hetzner server pool
  - When demand drops, scale down (with drain + cordon)
- [ ] **S2-18** Node pool warm buffer: keep 1-2 standby servers for instant scaling
- [ ] **S2-19** Terraform state management: Remote backend (S3 or Terraform Cloud) for infra state
- [ ] **S2-20** Ansible playbook `deploy-platform.yml`: Full platform deploy (replaces `scripts/deploy.sh`)
  - Build images, push/import, apply Helm charts, verify rollout

### Definition of Done
- Creating a customer in Admin → CAPI provisions a Hetzner cluster (no SSH needed)
- Customer gets `cloud.{domain}` with TLS within 5 minutes
- Cluster can be scaled and upgraded from Admin panel
- Deleting a customer tears down the cluster
- All infra changes tracked in Terraform state
- All server config reproducible via Ansible

---

## Phase 3: Resource Metering & Limits

> **Goal:** Track what each customer uses. Enforce plan ceilings. Show usage in Admin.
> **Status:** NOT STARTED (0/11)

### Tasks

- [ ] **S3-01** Metering agent: Deploy into each customer cluster
  - Collects every 60s: CPU cores used, RAM used, pod count, PVC total size, S3 bucket total size, DB storage used, LB count
- [ ] **S3-02** Metering agent pushes to management API:
  - `POST /api/v1/internal/metering` — Body: `{ customer_id, metrics: [...] }`
  - Internal endpoint, service-to-service auth via shared secret
- [ ] **S3-03** API: Store metering data in `resource_usage` table
- [ ] **S3-04** API: `GET /api/v1/admin/customers/:id/usage`
  - Response: current usage vs plan ceiling for each resource type
  - Include: current, limit, percent, trend (last 7 days)
- [ ] **S3-05** API: `GET /api/v1/admin/customers/:id/usage/history`
  - Response: time-series usage data for charts (hourly/daily/monthly)
- [ ] **S3-06** Ceiling enforcement: When customer approaches limit (>80%), send alert to Admin dashboard
- [ ] **S3-07** Ceiling enforcement: When customer hits 100%, reject new resource creation in Zenith Operator (admission webhook)
  - Return clear error: "Plan limit reached. Contact support to upgrade."
- [ ] **S3-08** MC `/customers/[id]`: Resource usage dashboard
  - Visual gauges: `CPU [███████░░░] 72/160 cores (45%)`
  - Per resource: CPU, RAM, S3, DB Storage, Volumes, LBs
- [ ] **S3-09** MC `/customers/[id]`: Usage history charts (line chart, last 30 days)
- [ ] **S3-10** MC dashboard: Aggregate usage across all customers
  - Total platform: 450/800 cores used (56%), 900GB/1600GB RAM, etc.
- [ ] **S3-11** Alert system: Notify DoTech admin when any customer approaches ceiling
  - In-app notification + optional email/Slack webhook

### Definition of Done
- Real-time resource tracking per customer
- Usage gauges visible in Admin panel
- Plan ceilings enforced at the Operator level
- Alerts when customers approach limits

---

## Phase 4: Billing (Stripe + Toman/Fairbroker)

> **Goal:** Automated billing. Customers are charged monthly based on their plan.
> **Status:** NOT STARTED (0/11)

### Tasks

- [ ] **S4-01** Integrate Stripe SDK in Go API
- [ ] **S4-02** API: `POST /api/v1/admin/customers/:id/billing/setup`
  - Create Stripe Customer, attach to our customer record
- [ ] **S4-03** API: Stripe webhook handler for payment events
  - `invoice.paid` → mark invoice as paid
  - `invoice.payment_failed` → mark as failed, notify admin
  - `customer.subscription.deleted` → handle cancellation
- [ ] **S4-04** API: Create monthly invoice automatically
  - Cron job: on 1st of month, create invoice for each active customer
  - Amount = plan.price (fixed for now, usage-based later)
- [ ] **S4-05** API: `GET /api/v1/admin/customers/:id/invoices` — list invoices
- [ ] **S4-06** API: `GET /api/v1/admin/billing/overview`
  - Response: MRR, total revenue, outstanding invoices, failed payments
- [ ] **S4-07** MC page: `/billing` — billing overview dashboard
  - Cards: MRR, Customers, Revenue this month, Outstanding
  - Table: Recent invoices across all customers
- [ ] **S4-08** MC `/customers/[id]`: Billing tab
  - Payment method, Invoice history, Plan changes, Upcoming invoice
- [ ] **S4-09** Fairbroker integration for Toman/IRR payments (details TBD)
  - Placeholder: API endpoint + webhook handler
  - Will be wired when Fairbroker specs are provided
- [ ] **S4-10** Web Platform `/billing` page: Show customer their own plan usage + invoices
  - Customer-facing, not admin — they see their own billing only
- [ ] **S4-11** Plan upgrade/downgrade flow
  - Customer requests upgrade → Admin approves → Stripe subscription updated
  - Pro-rated billing for mid-cycle changes

### Definition of Done
- Stripe integration creates invoices and processes payments
- Admin sees MRR and billing overview
- Fairbroker integration placeholder ready
- Customers see their own billing in Web Platform

---

## Phase 5: Customer Onboarding Automation

> **Goal:** "Create customer" is a single click that does everything end-to-end.
> **Status:** NOT STARTED (0/5)

### Tasks

- [ ] **S5-01** Automated onboarding pipeline (triggered by S1-01):
  1. Create customer record in DB
  2. Create Stripe customer + subscription
  3. Provision CAPI cluster (S2-03)
  4. Wait for cluster ready
  5. Install Zenith Operator + Web Platform
  6. Configure DNS (Cloudflare API)
  7. Issue TLS certificates
  8. Create initial admin user for customer
  9. Send welcome email with credentials
- [ ] **S5-02** MC `/customers/new`: Real-time progress display
  - Step-by-step: [✓] Account created → [✓] Payment setup → [◌] Cluster provisioning → [ ] Installing platform → [ ] DNS configured → [ ] Ready!
- [ ] **S5-03** Customer self-service sign-up page (landing page integration)
  - Form: company name, email, plan selection, payment method
  - Triggers the same pipeline (S5-01) without admin intervention
- [ ] **S5-04** Welcome email template: credentials, `cloud.{domain}` URL, getting started guide
- [ ] **S5-05** Automated health check: Verify customer platform is accessible after onboarding
  - Retry DNS/TLS if needed

### Definition of Done
- Single-click customer creation in Admin
- Customer gets working `cloud.{domain}` within 5 minutes
- No manual steps required

---

## Phase 6: Open-Core Extraction (Future)

> **Goal:** Extract a free self-hostable version of Zenith. Not a priority now — design for it.
> **Status:** NOT STARTED (0/7)

### Design Principles (apply now, build later)

- [ ] **S6-01** `ZENITH_MODE` env var: `saas` vs `standalone`
  - SaaS mode: multi-customer, billing, centralized admin
  - Standalone mode: single-tenant, no billing, self-managed
- [ ] **S6-02** MC codebase: Feature flags based on `ZENITH_MODE`
  - SaaS: Customers, Plans, Billing pages visible
  - Standalone: Clusters, Modules, Infrastructure pages only
- [ ] **S6-03** API codebase: Same pattern
  - SaaS: `/admin/customers/*`, `/admin/billing/*` endpoints active
  - Standalone: `/admin/clusters/*`, `/admin/modules/*` only
- [ ] **S6-04** CLI: `zen install` provisions a standalone Zenith
  - Installs k3s + CAPI + MC + API on a single server
  - Customer runs their own MC at `ms.{their-domain}`
- [ ] **S6-05** Helm chart: Supports both modes via `values.yaml`
  - `zenith.mode: saas | standalone`
- [ ] **S6-06** Documentation: Open-core quickstart guide
- [ ] **S6-07** Landing page: "Self-host for free" CTA alongside "Enterprise" CTA

### Architecture Compatibility

The SaaS and open-core share:
- Same Web Platform (customer-facing dashboard)
- Same Zenith Operator (manages resources inside a cluster)
- Same API (resource CRUD, auth)
- Same CRD definitions

The SaaS adds:
- Multi-customer management layer
- Billing integration
- Centralized metering
- Cross-cluster admin dashboard

The open-core replaces:
- SaaS Admin → standalone Mission Control (manages own clusters only)
- Centralized billing → removed (self-hosted = free)
- Cross-cluster → single-cluster focus

---

## Priority Order

```
Phase 0: PostgreSQL + Persistence          ← MUST DO FIRST (everything depends on this)
Phase 1: Customer Management in Admin      ← Core SaaS feature
Phase 2: CAPI Cluster Provisioning         ← Actually delivers infrastructure
Phase 3: Resource Metering & Limits        ← Enforcement + visibility
Phase 4: Billing (Stripe + Fairbroker)     ← Revenue collection
Phase 5: Customer Onboarding Automation    ← Scale without manual work
Phase 6: Open-Core Extraction              ← Future, design for it now
```

Phases 0–2 make the product **viable** (you can sell it with manual billing).
Phases 3–4 make it **scalable** (automated billing, enforced limits).
Phase 5 makes it **self-service**.
Phase 6 makes it **famous**.

---

## What Exists Today (Technical Detail)

### API (`services/api/`) — In-Memory, No Persistence
- **Framework:** Go + Fiber v2, all routes defined and working
- **Auth:** JWT login/register/refresh, bcrypt passwords, role hierarchy (Owner > Admin > Developer > Viewer)
- **Stores:** All in-memory — `UserStore`, `MemoryClient` (K8s CRDs), `MemoryStore` (CAPI/admin data)
- **Handlers:** Full set — projects, apps, databases, storage, clusters, tenants, modules, audit, settings, infra
- **Config:** PORT, JWT_SECRET, ADMIN_EMAIL/PASSWORD, CORS_ORIGINS, IN_CLUSTER
- **Missing:** No PostgreSQL driver (pgx/lib/pq), no migrations, no `database/sql`, no real K8s client (client-go)
- **Missing:** No customer CRUD, no plan CRUD, no billing endpoints

### Mission Control (`apps/mission-control/`) — UI Ready, No SaaS Pages
- **Pages:** Dashboard, Clusters, Tenants, Modules, Updates, Infrastructure, State, Audit, Settings, Login
- **Auth:** Login page, `useAuth()` hook, token persistence in localStorage, protected shell
- **API Client:** Full integration with all current endpoints + auth methods
- **Demo Mode:** Build-time flag `NEXT_PUBLIC_DEMO_MODE=true`, mock data for all endpoints
- **Missing:** No `/customers` page, no `/plans` page, no `/billing` page, no `ZENITH_MODE` flag

### IaC (`terraform/`, `helm/`, `scripts/`) — DNS Only, No Server Provisioning
- **Terraform:** Cloudflare DNS only (7 A records, 2 zones). No Hetzner provider, no server/volume/network resources
- **Helm charts:** `zenith/` (API + Operator + Auth + Kong + OTEL + RBAC) + `monitoring/` (Prometheus + Loki + Promtail) — defined but NOT deployed via deploy.sh
- **Scripts:** `deploy.sh` (full pipeline), `cloudflare-dns.sh` (DNS CRUD, has hardcoded token — needs cleanup), `e2e-test.sh` (post-deploy validation)
- **Ansible:** Does not exist yet — server setup is manual SSH
- **Missing:** No Hetzner Terraform (servers, volumes, networks, firewalls), no Ansible playbooks, no remote Terraform state, no CAPI templates, no secrets management (Vault/SOPS)

### Infrastructure — Live, No Database
- **Server:** ghasi (161.35.82.211), k3s v1.34.3, Traefik 3.5.1
- **Namespaces:** `zenith-platform` (landing, api, demo-mc, demo-web), `zenith-embermind` (customer mc, web)
- **Deploy:** `scripts/deploy.sh` builds 6 Docker images, imports to k3s, applies manifests
- **TLS:** cert-manager with letsencrypt-prod ClusterIssuer
- **Missing:** No PostgreSQL deployment, no `DATABASE_URL` in api.yaml, no CAPI/CAPH installed

---

## Files That Need Changes

| File / Directory | What Changes |
|-----------------|-------------|
| `services/api/` | PostgreSQL (pgx), customer/plan/billing handlers, metering endpoints, real K8s client (client-go) |
| `apps/mission-control/` | Customer pages, billing pages, plan management, usage dashboards, ZENITH_MODE flag |
| `apps/web/` | Customer-facing billing page (sees own usage/invoices) |
| `k8s/api.yaml` | Add DATABASE_URL, remove in-memory-only config |
| `k8s/` | Add PostgreSQL deployment or CNPG operator |
| `services/operator/` | Admission webhook for ceiling enforcement |
| `helm/zenith/` | ZENITH_MODE values, customer cluster template |
| **NEW** `infra/terraform/` | Restructure: Hetzner provider (servers, volumes, networks, firewalls) + Cloudflare DNS modules |
| **NEW** `infra/ansible/` | Playbooks: `setup-server.yml`, `install-k3s.yml`, `setup-postgres.yml`, `deploy-platform.yml` |
| **NEW** `infra/capi-templates/` | CAPH ClusterTemplate for customer clusters (parameterized) |
| `terraform/` | Migrate to `infra/terraform/` with module structure |
| `scripts/deploy.sh` | Eventually replaced by `ansible-playbook deploy-platform.yml` |
| `scripts/cloudflare-dns.sh` | Remove hardcoded token, migrate to Terraform DNS module |
