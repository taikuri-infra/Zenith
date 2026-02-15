# Galaxy - Product Architecture & Strategy

> PaaS built on Hetzner Cloud. Simple like Fly.io, priced like Hetzner.

---

## 1. Product Architecture

### Logical Architecture

```
+------------------------------------------------------------------+
|                        GALAXY CONTROL PLANE                       |
|  (runs on dedicated k8s cluster - 3x CX33 nodes)                |
|                                                                   |
|  +------------+  +------------+  +-------------+  +------------+ |
|  |  Galaxy    |  |  Galaxy    |  |  Cluster    |  |  Billing   | |
|  |  API       |  |  Web UI    |  |  Manager    |  |  Engine    | |
|  |  (FastAPI) |  |  (Next.js) |  |  (CAPI +   |  |  (usage    | |
|  |            |  |            |  |   Hetzner)  |  |   metering)| |
|  +-----+------+  +-----+------+  +------+------+  +-----+------+ |
|        |               |                |                |        |
|  +-----+------+  +-----+------+  +------+------+  +-----+------+ |
|  |  Auth /    |  |  Deploy    |  |  DNS /      |  |  Metrics / | |
|  |  IAM       |  |  Pipeline  |  |  Cert Mgr   |  |  Logging   | |
|  |  (NextGate)|  |  (Tekton)  |  |  (external- |  |  (Loki +   | |
|  |            |  |            |  |   dns+LE)   |  |   Grafana) | |
|  +------------+  +------------+  +-------------+  +------------+ |
+------------------------------------------------------------------+
         |                    |                  |
         v                    v                  v
+------------------+  +------------------+  +------------------+
| SHARED CLUSTER   |  | DEDICATED CLUSTER|  | DEDICATED CLUSTER|
| (Starter Plan)   |  | (Pro - Customer A|  | (Pro - Customer B|
|                  |  |                  |  |                  |
| ns: customer-1   |  | 3+ nodes         |  | 3+ nodes         |
| ns: customer-2   |  | full isolation   |  | full isolation   |
| ns: customer-3   |  | own LB           |  | own LB           |
| ...              |  | own volumes      |  | own volumes      |
|                  |  |                  |  |                  |
| Shared LB        |  | Customer workload|  | Customer workload|
| ResourceQuotas   |  |                  |  |                  |
| NetworkPolicies  |  |                  |  |                  |
+------------------+  +------------------+  +------------------+
         \                    |                  /
          \                   |                 /
           +------------------+-----------------+
           |         HETZNER CLOUD              |
           |  VMs | Volumes | LBs | Object Stg  |
           |  Networks | Floating IPs            |
           +------------------------------------+
```

### Infrastructure Architecture

```
Control Plane Cluster (always running):
  - 3x CX33 (4 vCPU, 8GB) = 3 x 5.49 = ~17 EUR/mo
  - 1x LB11 = ~5.49 EUR/mo
  - Object Storage = 4.99 EUR/mo
  - Total control plane cost: ~28 EUR/mo

Shared Cluster (Starter customers):
  - Starts with 3x CX33, scales by adding nodes
  - 1x LB11 shared across all starter customers
  - Hetzner Volumes via CSI for PVCs

Pro Clusters (per customer):
  - Minimum 3x CX23 (2 vCPU, 4GB) = 3 x 3.49 = ~10.50 EUR/mo
  - 1x LB11 = 5.49 EUR/mo
  - Scales by adding nodes ("Add a Planet")
```

---

## 2. MVP Scope (60-Day Delivery)

### Week 1-2: Foundation
- [ ] Control plane k3s cluster on Hetzner (3 nodes, Terraform)
- [ ] Galaxy API skeleton (FastAPI, same Lich arch as FairBroker)
- [ ] Auth system (email/password + GitHub OAuth)
- [ ] Hetzner Cloud API integration (create/delete VMs, volumes, LBs)
- [ ] Database: PostgreSQL for control plane state

### Week 3-4: Cluster Lifecycle
- [ ] Shared cluster provisioning (k3s on Hetzner VMs)
- [ ] Namespace-per-customer isolation (Starter plan)
- [ ] ResourceQuota + LimitRange per namespace
- [ ] NetworkPolicy isolation between namespaces
- [ ] Hetzner CSI driver for persistent volumes
- [ ] Hetzner Cloud Controller Manager (LB integration)

### Week 5-6: App Deployment
- [ ] GitHub repo connect + webhook-based deploy
- [ ] Docker image deploy (pull from any registry)
- [ ] Build pipeline: Dockerfile -> build -> push to internal registry -> deploy
- [ ] Automatic SSL via cert-manager + Let's Encrypt
- [ ] Custom domain support (CNAME verification)
- [ ] Environment variables management
- [ ] App logs (kubectl logs proxy via API)

### Week 7-8: Data Services + Billing + Polish
- [ ] Managed Postgres (CloudNativePG operator, single instance for MVP)
- [ ] Managed Redis (single instance via Helm)
- [ ] Billing engine: track compute hours, storage GB, bandwidth
- [ ] Stripe integration for payments
- [ ] Pricing calculator page
- [ ] Landing page + docs
- [ ] Basic monitoring dashboard (CPU, RAM, requests)

### NOT in MVP (Phase 2+)
See section 10.

---

## 3. Technical Components by Layer

### 3.1 Control Plane Layer

| Component | Tech | Purpose |
|-----------|------|---------|
| Galaxy API | FastAPI (Python) | All platform operations |
| Galaxy Web | Next.js 15 | Dashboard, deploy UI, billing |
| Auth/IAM | NextGate (custom) | JWT, OAuth, RBAC |
| Database | PostgreSQL 16 | Users, projects, billing, state |
| Cache | Redis 7 | Sessions, rate limiting |
| Task Queue | Temporal | Async operations (provisioning, builds) |
| Internal Registry | Harbor or Nexus | Store customer container images |

### 3.2 Provisioning Layer

| Component | Tech | Purpose |
|-----------|------|---------|
| Cluster Provisioner | Terraform + Hetzner provider | Create/destroy k3s clusters |
| Node Scaler | Custom (Hetzner API) | Add/remove nodes to clusters |
| Image Builder | Kaniko (in-cluster) | Build Docker images from source |
| Deploy Controller | Custom K8s operator or Temporal workflows | Roll out app updates |

**Why NOT CAPI for MVP:** Cluster API is powerful but complex to set up. For a 60-day MVP, direct Terraform + k3s install scripts (like FairBroker's cluster-ember module) is faster. Migrate to CAPI in Phase 2 when managing 20+ clusters.

### 3.3 Cluster Layer

| Component | Tech | Purpose |
|-----------|------|---------|
| Kubernetes | k3s v1.29+ | Container orchestration |
| CNI | Flannel (k3s default) | Pod networking |
| Ingress | Traefik (k3s default) | HTTP routing |
| Cert Manager | cert-manager | Auto SSL |
| Storage | Hetzner CSI Driver | Persistent volumes |
| Cloud Controller | hcloud-cloud-controller | LB + node integration |

### 3.4 Service Layer (Managed Services)

| Service | Operator/Tech | MVP? |
|---------|---------------|------|
| PostgreSQL | CloudNativePG | Yes |
| Redis | Helm chart (Bitnami) | Yes |
| MySQL | MySQL Operator | No (Phase 2) |
| MongoDB | MongoDB Community Operator | No (Phase 2) |
| Object Storage | Proxy to Hetzner S3 | No (Phase 2) |

### 3.5 Data Layer

| Component | Tech | Purpose |
|-----------|------|---------|
| Backups (DB) | pg_dump + cron -> Hetzner S3 | Daily DB backups |
| Backups (Volumes) | Hetzner volume snapshots | Weekly volume snapshots |
| Metrics storage | Prometheus | Time-series metrics |
| Log storage | Loki | Log aggregation |

### 3.6 Networking Layer

| Component | Tech | Purpose |
|-----------|------|---------|
| External LB | Hetzner Load Balancer | Customer traffic entry |
| DNS | Cloudflare API (or Hetzner DNS) | Auto DNS for *.galaxy.dev |
| TLS | Let's Encrypt via cert-manager | Auto SSL |
| Network isolation | K8s NetworkPolicy | Starter plan tenant isolation |
| Private networking | Hetzner vSwitch / Networks | Inter-node communication |

---

## 4. Pricing Model

### Hetzner Cost Basis (what WE pay)

| Resource | Hetzner Price | Our Markup | Galaxy Price |
|----------|--------------|------------|--------------|
| 1 shared vCPU | ~1.75 EUR/mo | 2.5x | ~4.50 EUR/mo |
| 1 GB RAM | ~0.87 EUR/mo | 2.5x | ~2.20 EUR/mo |
| 1 GB SSD | 0.044 EUR/mo | 3x | ~0.13 EUR/mo |
| 1 GB bandwidth | included (20TB) | - | free (fair use) |
| Load Balancer | 5.49 EUR/mo | shared | included |
| Object Storage | 4.99 EUR/mo (1TB) | 2x | ~10 EUR/TB/mo |

### Starter Plan - $5/mo base

Target: Indie devs, side projects, MVPs

| Included | Spec |
|----------|------|
| Compute | 1 shared vCPU, 512MB RAM |
| Storage | 1 GB persistent volume |
| Apps | Up to 3 |
| Databases | 1x Postgres (256MB) |
| Bandwidth | 100 GB/mo |
| SSL | Automatic |
| Custom Domain | 1 |
| Builds | 100/mo |

Scale up:
- Extra CPU: $4/vCPU/mo
- Extra RAM: $2/GB/mo
- Extra storage: $0.15/GB/mo
- Extra apps: $2/app/mo

### Pro Plan - $49/mo base

Target: Startups, production workloads

| Included | Spec |
|----------|------|
| Compute | Dedicated 3-node cluster (2 vCPU, 4GB each) |
| Storage | 20 GB persistent volume |
| Apps | Unlimited |
| Databases | 2x managed (Postgres + Redis) |
| Bandwidth | 1 TB/mo |
| SSL | Automatic |
| Custom Domains | Unlimited |
| Builds | Unlimited |
| Backups | Daily (7-day retention) |
| Monitoring | Full Grafana dashboard |

Scale up:
- Add node ("Add a Planet"): $12/node/mo (CX23) to $65/node/mo (CCX23)
- Extra storage: $0.10/GB/mo
- Extra bandwidth: $1/TB

### Enterprise Plan - Custom

Target: Larger teams, compliance needs

- Dedicated cluster with custom node sizes
- SLA guarantee
- Priority support
- Custom domains + IP allowlisting
- GDPR compliance pack
- Volume: negotiated pricing

### Margin Analysis

```
Starter Plan ($5/mo):
  Our cost: ~2 EUR (fraction of shared node + overhead)
  Margin: ~55%

Pro Plan ($49/mo):
  Our cost: 3x CX23 (10.50) + LB (5.49) + storage + overhead = ~20 EUR
  Margin: ~55%

Pro Plan + 2 extra nodes ($49 + $24 = $73/mo):
  Our cost: 5x CX23 (17.45) + LB (5.49) = ~23 EUR
  Margin: ~65%
```

The margin improves as customers scale. This is the Hetzner advantage - our cost basis is 3-5x lower than Fly.io's.

---

## 5. Scaling Model

### Starter Plan (Shared Cluster)

```
Phase 1: 3 nodes serve ~50 customers
Phase 2: Auto-add nodes when cluster CPU > 70%
Phase 3: Multiple shared clusters by region (eu-central, eu-north, us-east)

Scaling trigger: Cluster avg CPU > 70% for 10 min
Action: Terraform adds a CX33 node, k3s auto-joins
```

### Pro Plan (Dedicated Cluster)

```
Customer clicks "Add a Planet" (add node)
  -> Galaxy API -> Hetzner API: create VM
  -> Run k3s agent join script
  -> Node joins cluster in ~90 seconds
  -> Billing updated immediately

Customer clicks "Remove a Planet"
  -> Cordon + drain node
  -> Delete VM
  -> Billing updated
```

### Node Size Options ("Planet Sizes")

| Planet | Hetzner Type | Specs | Price |
|--------|-------------|-------|-------|
| Nano | CX23 | 2 vCPU, 4GB, 40GB | $12/mo |
| Small | CX33 | 4 vCPU, 8GB, 80GB | $18/mo |
| Medium | CX43 | 8 vCPU, 16GB, 160GB | $32/mo |
| Large | CX53 | 16 vCPU, 32GB, 320GB | $58/mo |
| Mega | CCX23 | 4 dedicated vCPU, 16GB | $65/mo |
| Ultra | CCX33 | 8 dedicated vCPU, 32GB | $130/mo |

---

## 6. Isolation Model

### Starter Plan (Multi-Tenant)

```
Isolation Layer          Mechanism
─────────────────────────────────────────
Namespace                K8s namespace per customer
Compute limits           ResourceQuota per namespace
Memory limits            LimitRange per container
Network                  NetworkPolicy (deny all cross-namespace)
Storage                  Separate PVCs, no shared volumes
Ingress                  Per-namespace Ingress rules
Secrets                  K8s RBAC (namespace-scoped)
Registry                 Separate image repos per customer
```

**What's shared:** Nodes, control plane, Traefik, cert-manager, LB, DNS.
**Risk:** Noisy neighbor. Mitigated by ResourceQuota + LimitRange.

### Pro Plan (Single-Tenant)

```
Isolation Layer          Mechanism
─────────────────────────────────────────
Cluster                  Dedicated k3s cluster
Network                  Dedicated Hetzner private network
Load Balancer            Dedicated Hetzner LB
Storage                  Dedicated Hetzner volumes
DNS                      Dedicated wildcard cert
```

**Full isolation.** Customer workload cannot affect others.

### Comparison

| Aspect | Starter | Pro |
|--------|---------|-----|
| Blast radius | Namespace | Full cluster |
| Noisy neighbor | Possible (limited) | Impossible |
| Compliance | Basic | GDPR-ready |
| Scale ceiling | Limited by quota | Limited by node count |
| Cost | Low | Higher |

---

## 7. Biggest Risks

| # | Risk | Impact | Mitigation |
|---|------|--------|------------|
| 1 | **Cluster provisioning reliability** | Pro customers can't start | Extensively test Terraform + k3s. Have manual fallback. Keep it simple (no CAPI in v1). |
| 2 | **Noisy neighbor on shared cluster** | Starter customers affect each other | Strict ResourceQuota. Monitor per-namespace. Auto-evict over-limit pods. |
| 3 | **Build pipeline failures** | Deploys stuck | Kaniko is battle-tested. Add build timeout (10 min). Show clear error logs to user. |
| 4 | **Hetzner API rate limits** | Can't scale fast enough | Cache plan data. Batch operations. Hetzner limits are generous (3600 req/hr). |
| 5 | **Database management complexity** | Data loss, customer trust | Start with single-instance Postgres (not HA). Add replication in Phase 2. Daily backups from day 1. |
| 6 | **SSL/DNS propagation delays** | Custom domains slow to activate | Use Cloudflare API for fast DNS. cert-manager handles retries. Set expectations (up to 10 min). |
| 7 | **Billing accuracy** | Under/overcharging | Meter at Hetzner API level (actual VMs running). Reconcile hourly. |
| 8 | **Security breach in shared cluster** | All starter customers exposed | NetworkPolicy + RBAC + pod security standards. No privileged containers. Regular audits. |
| 9 | **Support burden** | 2-person team overwhelmed | Good docs, status page, self-service. Limit Starter plan support to community/email. |
| 10 | **Hetzner dependency** | Single cloud vendor lock-in | Acceptable for v1. Abstract provider layer for future multi-cloud. |

---

## 8. UX Structure

### Pages / Screens

```
Landing Page (galaxy.dev)
  ├── Hero: "Deploy your app in 30 seconds"
  ├── How it works (3 steps)
  ├── Pricing calculator
  ├── Comparison table (vs Fly.io, Railway, Render)
  └── CTA: "Start Free"

Auth
  ├── Sign up (email or GitHub)
  └── Login

Dashboard (after login)
  ├── Projects list
  │     └── Create Project (name, plan)
  │
  ├── Project View
  │     ├── Apps tab
  │     │     ├── Deploy new app (GitHub / Docker image)
  │     │     ├── App detail: logs, env vars, domains, scaling
  │     │     └── Redeploy button
  │     │
  │     ├── Databases tab
  │     │     ├── Create database (Postgres/Redis)
  │     │     ├── Connection string (copy)
  │     │     └── Backups list
  │     │
  │     ├── Storage tab
  │     │     ├── Volumes list
  │     │     └── Create volume (size)
  │     │
  │     ├── Planets tab (Pro only - scaling)
  │     │     ├── Current nodes list
  │     │     ├── "Add a Planet" button
  │     │     └── Node metrics (CPU/RAM/disk)
  │     │
  │     ├── Monitoring tab
  │     │     ├── CPU / RAM / Network charts
  │     │     ├── Request count
  │     │     └── Error rate
  │     │
  │     └── Settings tab
  │           ├── Custom domains
  │           ├── Environment variables
  │           ├── Danger zone (delete project)
  │           └── Plan upgrade
  │
  ├── Billing
  │     ├── Current month usage
  │     ├── Invoice history
  │     └── Payment method
  │
  └── Account
        ├── Profile
        ├── API keys
        └── Team members
```

### UX Principles

1. **No infrastructure language.** Say "App", not "Deployment". Say "Database", not "StatefulSet".
2. **One-click deploy.** Connect GitHub -> select branch -> deploy. Three clicks max.
3. **Live cost.** Always show "This will cost ~$X/mo" before any action.
4. **Copy-paste friendly.** Connection strings, env vars, CLI commands - one click copy.
5. **Status always visible.** Green/yellow/red dot on every resource. No guessing.

---

## 9. Competitive Advantage Positioning

### Galaxy vs Competition

| Feature | Fly.io | Railway | Render | Galaxy |
|---------|--------|---------|--------|--------|
| Cheapest VM | $2.02/mo | $5/mo | $7/mo | **$5/mo (more resources)** |
| EU data residency | Limited | No | No | **Yes (Hetzner DE/FI)** |
| Dedicated cluster | No | No | No | **Yes (Pro plan)** |
| Managed Postgres | $0 (DIY) | $5/mo | $7/mo | **Included in Pro** |
| Bandwidth | $0.02/GB | metered | metered | **Generous free tier** |
| GDPR by default | No | No | No | **Yes (EU infra)** |
| Pricing transparency | Good | OK | OK | **Calculator + live cost** |
| Kubernetes access | No | No | No | **Optional (Pro plan)** |

### Positioning Statement

> **Galaxy**: The developer cloud that doesn't charge cloud prices.
>
> Deploy apps, databases, and storage on European infrastructure.
> Start at $5/mo. Scale to production with dedicated clusters.
> No DevOps degree required.

### Key Differentiators to Emphasize

1. **3-5x cheaper than Fly.io/Railway** for equivalent resources (Hetzner cost advantage)
2. **EU-first, GDPR-compliant** by default (all data in Germany/Finland)
3. **Dedicated cluster option** - no other PaaS offers true isolation at this price point
4. **Transparent pricing** - calculator shows exact cost, no surprise bills
5. **"Add a Planet" scaling** - unique, fun metaphor. Makes scaling feel simple.

---

## 10. What NOT to Build in Phase 1

| Feature | Why Skip | When |
|---------|----------|------|
| CAPI (Cluster API) | Overkill for <20 clusters. Terraform + scripts is fine. | Phase 2 (month 4+) |
| MySQL / MongoDB | Focus on Postgres + Redis. Cover 90% of use cases. | Phase 2 |
| Multi-region | Start with eu-central (Falkenstein/Nuremberg). | Phase 2 |
| Auto-scaling (HPA) | Let users manually add/remove planets first. | Phase 2 |
| CLI tool (`galaxy deploy`) | Web UI + API is enough for launch. | Phase 2 |
| Object Storage proxy | Point users to Hetzner S3 directly with credentials. | Phase 2 |
| Team management / RBAC | Single user per account for MVP. | Phase 2 |
| Marketplace / Add-ons | No plugin ecosystem yet. | Phase 3 |
| Mobile app | Not needed. Responsive web is fine. | Never (unless demand) |
| GPU instances | Hetzner doesn't have great GPU offering. | Evaluate later |
| Custom buildpacks | Support Dockerfile only. Nixpacks/buildpacks later. | Phase 2 |
| Webhooks / API for CI/CD | GitHub webhook deploy is enough. No external API v1. | Phase 2 |
| Log retention > 7 days | Keep costs low. 7 days for Starter, 30 for Pro. | Phase 2 |
| HA databases | Single-instance Postgres in MVP. Replication later. | Phase 2 |

---

## Appendix: 60-Day Calendar

```
Week  1-2:  Control plane cluster + API skeleton + auth + DB schema
Week  3-4:  Shared cluster provisioning + namespace isolation + CSI
Week  5-6:  App deploy (GitHub + Docker) + SSL + custom domains + builds
Week  7:    Managed Postgres + Redis + billing engine + Stripe
Week  8:    Landing page + docs + pricing calculator + beta launch
Week  9-10: (buffer) Bug fixes, Pro plan cluster provisioning, polish
```

## Appendix: Tech Stack Summary

```
Control Plane:
  API:        Python 3.12, FastAPI, SQLAlchemy, Pydantic
  Web:        Next.js 15, TypeScript, Tailwind CSS
  DB:         PostgreSQL 16
  Cache:      Redis 7
  Queue:      Temporal
  Auth:       NextGate (custom JWT + OAuth)

Infrastructure:
  IaC:        Terraform + Hetzner provider
  Kubernetes: k3s v1.29
  Ingress:    Traefik
  TLS:        cert-manager + Let's Encrypt
  Storage:    Hetzner CSI driver
  LB:         Hetzner Cloud LB
  Registry:   Harbor (or Nexus, reuse from FairBroker)
  Builds:     Kaniko
  DNS:        Cloudflare API (or Hetzner DNS)

Monitoring:
  Metrics:    Prometheus + Grafana
  Logs:       Loki + Grafana
  Alerts:     Alertmanager
```
