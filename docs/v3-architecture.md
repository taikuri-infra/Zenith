# Zenith V3 — Complete Platform Architecture & Implementation Plan

> **Status:** Active — Single Source of Truth
> **Version:** 3.0.0
> **Last Updated:** 2026-03-08
> **Audience:** Junior developers, senior architects, new AI agents, business owners
> **Rule:** If you are a new developer or AI agent, read this document FIRST. Everything you need is here.

---

## How to Read This Document

This document is divided into **4 major sections**:

| Section | Who Should Read | What You'll Learn |
|---------|----------------|-------------------|
| **Part A: Business & Product** | Everyone | What Zenith is, who pays, what they get |
| **Part B: Architecture & Design** | Engineers + Architects | How everything connects, why each choice was made |
| **Part C: Implementation Phases** | Engineers (hands-on) | What to build, in what order, with checkboxes |
| **Part D: Day-2 Operations** | DevOps + SRE | How to maintain, scale, backup, recover |

**Convention:** Technology names (e.g., APISIX, Loki, Prometheus) are used in owner/engineering sections only. Customer-facing features use generic names (API Gateway, Logs, Metrics). This is intentional — customers should never see infrastructure details.

---

# PART A: BUSINESS & PRODUCT

---

## A1. What Is Zenith?

Zenith is a **Kubernetes-native Platform-as-a-Service (PaaS)** built exclusively on Hetzner Cloud. Think of it as a European, affordable, open-source alternative to Railway, Render, or Heroku — but with the depth of AWS.

**What we provide:**
- Deploy applications (Docker containers, git repos)
- Managed databases (PostgreSQL, Redis, MongoDB)
- Object storage (S3-compatible buckets)
- API gateway with auth (JWT, OIDC, rate limiting)
- Managed authentication (SSO, OAuth, user pools)
- Message queues (RabbitMQ, Kafka)
- Monitoring (metrics, logs, pod health)
- Custom domains with automatic TLS
- Container registry with security scanning
- Team management with RBAC

**What we DON'T do (yet):**
- We don't build customer images — they build their own (GitHub Actions, GitLab CI, etc.)
- We don't provide VMs — everything is containers
- We don't do full APM with code instrumentation (v2 feature)
- We don't do multi-region (v2 feature)

**What makes us different:**
- European infrastructure (Hetzner Germany) — GDPR compliant by default
- 3-5x cheaper than AWS/GCP for equivalent resources
- Open-source stack — no vendor lock-in
- Premium support with on-call engineers (Gold/Platinum)
- Self-host option available

---

## A2. Pricing — The Audi Strategy

> **Philosophy:** We are Audi. Premium quality, accessible pricing. Not BMW/Mercedes expensive, not Toyota cheap. Customers should feel they're getting exceptional value without questioning our quality.

### McDonald's Decoy Effect

The pricing is deliberately structured so that **Business** is the obvious best choice:

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│   FREE   │    │   PRO    │    │   TEAM   │    │ BUSINESS │    │ENTERPRISE│
│   €0/mo  │    │  €29/mo  │    │ €99/seat │    │€149/seat │    │ Custom   │
│          │    │          │    │  min 3   │    │  min 3   │    │ €2000+   │
│  Small   │    │  Small   │    │  Medium  │    │  Large   │    │          │
│  Coke    │    │  Coke    │    │  Coke    │    │  Coke    │    │  Custom  │
│          │    │ (worth   │    │ (pricey) │    │ (BEST    │    │  (sales  │
│  (test)  │    │  it)     │    │ DECOY    │    │  DEAL)   │    │  call)   │
└──────────┘    └──────────┘    └──────────┘    └──────────┘    └──────────┘
```

**Why this works:**
- **Pro** (€29) = genuinely good value for solo developers
- **Team** (€99/seat) = seems expensive for what you get (shared infra, no SSO)
- **Business** (€149/seat) = only €50 more than Team but gets dedicated infrastructure, SSO, audit logs, compliance → 90% choose Business
- **Enterprise** = custom for large companies, handled by sales

### Tier Comparison

```
╔══════════════════╦═══════╦═══════╦══════════╦══════════════╦════════════╗
║ Feature          ║ FREE  ║  PRO  ║   TEAM   ║   BUSINESS   ║ ENTERPRISE ║
╠══════════════════╬═══════╬═══════╬══════════╬══════════════╬════════════╣
║ Price            ║  €0   ║ €29/m ║ €99/seat ║  €149/seat   ║  Custom    ║
║ Min Seats        ║   1   ║   1   ║    3     ║      3       ║   Custom   ║
║ Infrastructure   ║Shared ║Shared ║ Shared   ║  Dedicated   ║  Dedicated ║
║ Apps             ║   1   ║   5   ║   20     ║  No Limit    ║  No Limit  ║
║ Databases (PG)   ║   1   ║   3   ║   10     ║  No Limit    ║  No Limit  ║
║ Redis            ║   -   ║   ✓   ║    ✓     ║      ✓       ║     ✓      ║
║ MongoDB          ║   -   ║   -   ║    ✓     ║      ✓       ║     ✓      ║
║ RabbitMQ         ║   -   ║   ✓   ║    ✓     ║      ✓       ║     ✓      ║
║ Kafka            ║   -   ║   -   ║    -     ║      ✓       ║     ✓      ║
║ Storage          ║  1GB  ║ 10GB  ║  100GB   ║   No Limit   ║  No Limit  ║
║ Custom Domain    ║   -   ║   ✓   ║    ✓     ║      ✓       ║     ✓      ║
║ Container Reg    ║   -   ║   ✓   ║    ✓     ║      ✓       ║     ✓      ║
║ Always-On        ║   -   ║   ✓   ║    ✓     ║      ✓       ║     ✓      ║
║ Sleep After Idle ║ 15min ║ Never ║  Never   ║    Never     ║   Never    ║
║ RBAC             ║   -   ║   -   ║    ✓     ║      ✓       ║     ✓      ║
║ SSO (SAML/OIDC)  ║   -   ║   -   ║    -     ║      ✓       ║     ✓      ║
║ Audit Logs       ║   -   ║   -   ║    -     ║      ✓       ║     ✓      ║
║ Compliance       ║   -   ║   -   ║    -     ║      ✓       ║     ✓      ║
║ Blue/Green Deploy║   -   ║   -   ║    -     ║      ✓       ║     ✓      ║
║ Custom Metrics   ║   -   ║   -   ║    -     ║      ✓       ║     ✓      ║
║ WAF Config       ║   -   ║   -   ║    -     ║      ✓       ║     ✓      ║
║ Firewall Config  ║   -   ║   -   ║    -     ║      ✓       ║     ✓      ║
║ SSH to Pods      ║   -   ║   -   ║    -     ║  ✓ (recorded)║ ✓ (recorded)║
║ MFA              ║   -   ║  ✓*   ║   ✓*    ║     ✓*       ║    ✓*      ║
║ Monitoring       ║ Basic ║ Full  ║  Full    ║ Full+Custom  ║ Full+Custom║
║ Support          ║ Comm. ║ Std   ║  Std     ║  Priority    ║  Dedicated ║
║ Terraform/CLI    ║   -   ║   ✓   ║    ✓     ║      ✓       ║     ✓      ║
║ SLA              ║   -   ║   -   ║    -     ║    99.5%     ║   99.9%    ║
╚══════════════════╩═══════╩═══════╩══════════╩══════════════╩════════════╝

* MFA is MANDATORY for Pro and above (not optional)
```

**Business tier = NO LIMITS.** We calculate our cost and add margin. Whatever they need, we provide.

### Support Model

```
╔═════════════╦═══════════╦════════════╦═══════════╦══════════════════════════╗
║ Level       ║ Available ║  Response  ║   Cost    ║ What They Get            ║
╠═════════════╬═══════════╬════════════╬═══════════╬══════════════════════════╣
║ Community   ║ Free      ║ Best-effor ║   €0      ║ Docs + GitHub Issues     ║
║ Standard    ║ Pro+      ║ 48h email  ║ Included  ║ Email support            ║
║ Priority    ║ Business  ║ 12h ticket ║ Included  ║ Priority queue           ║
║ Gold        ║ Pro+      ║ 10 min     ║ €699/mo   ║ On-call + 1 arch session ║
║ Platinum    ║ Business+ ║ 5 min      ║ €1499/mo  ║ Dedicated eng + proactive║
╚═════════════╩═══════════╩════════════╩═══════════╩══════════════════════════╝
```

**Gold Support Value Prop:**
> "You don't need a DevOps team. We monitor your infrastructure, handle alerts, and within 10 minutes someone calls you. You handle implementations, we handle everything else. Includes 1 free architecture consultation session per month."

**Platinum Support Value Prop:**
> "Your dedicated engineer proactively monitors your infrastructure, suggests optimizations, and is available within 5 minutes. Unlimited architecture sessions."

### Add-on Marketplace

| Add-on | Price | Available From |
|--------|-------|----------------|
| Gold Support | €699/mo | Pro+ |
| Platinum Support | €1499/mo | Business+ |
| Extra Compute (+2GB RAM) | €15/mo | Pro+ |
| Extra Storage (+10GB) | €8/mo | Pro+ |
| Dedicated Build Runner | €49/mo | Team+ |
| Extended Backup (90-day retention) | €25/mo | Pro+ |
| RabbitMQ Instance | €15/mo | Pro+ |
| Kafka Cluster | €49/mo | Business+ |
| MongoDB Instance | €20/mo | Team+ |
| Redis Instance | €10/mo | Pro+ |

---

## A3. What the Customer Sees

> **Rule:** The customer NEVER sees infrastructure names. They see features, not technology.

### Customer-Facing Feature Names

| Internal Name | Customer Sees | Description |
|--------------|---------------|-------------|
| APISIX | API Gateway | Route, protect, rate-limit APIs |
| Loki | Logs | Application log viewer |
| Prometheus + Grafana | Metrics | CPU, memory, request charts |
| CNPG PostgreSQL | Managed Database (PostgreSQL) | One-click PostgreSQL |
| Redis Operator | Managed Cache (Redis) | In-memory cache/store |
| Percona MongoDB | Managed Database (MongoDB) | Document database |
| RabbitMQ Operator | Message Queue | Async messaging |
| Strimzi Kafka | Event Streaming | High-throughput events |
| Keycloak | Authentication | SSO, OAuth, user pools |
| Harbor | Container Registry | Push/pull Docker images |
| cert-manager | Custom Domains | Automatic TLS certificates |
| Kyverno | Security Scanning | Image vulnerability checks |
| Cilium NetworkPolicy | Network Isolation | App-to-app firewall |
| Cloudflare WAF | Web Application Firewall | DDoS + attack protection |
| Velero + CNPG WAL | Backups | Automated backup/restore |
| KEDA | Auto-Scaling | Scale up/down on demand |

### Customer Journey

```
┌─────────────────────────────────────────────────────────────────┐
│                    CUSTOMER JOURNEY                              │
├──────────┬──────────────────────────────────────────────────────┤
│          │                                                       │
│  STEP 1  │  Visit freezenith.com → See pricing → Sign up        │
│  Sign Up │  Choose plan (Free to try, Pro/Team/Business)         │
│          │  Pay via Stripe (if paid plan)                        │
│          │                                                       │
├──────────┼──────────────────────────────────────────────────────┤
│          │                                                       │
│  STEP 2  │  Land on Dashboard → See empty project                │
│  Orient  │  Guided tour: "Create your first app"                 │
│          │  Quick-start: connect GitHub repo or push image        │
│          │                                                       │
├──────────┼──────────────────────────────────────────────────────┤
│          │                                                       │
│  STEP 3  │  Build happens automatically (or push pre-built       │
│  Deploy  │  image to our Container Registry)                     │
│          │  Security scan runs → if clean → deploy               │
│          │  App gets: URL, TLS cert, monitoring, logs             │
│          │                                                       │
├──────────┼──────────────────────────────────────────────────────┤
│          │                                                       │
│  STEP 4  │  Add database (PostgreSQL, Redis, MongoDB)            │
│  Extend  │  Add storage bucket (S3-compatible)                   │
│          │  Add custom domain                                    │
│          │  Set up API Gateway routes                             │
│          │  Configure auth (SSO, user pools)                     │
│          │  Add team members (Team+)                             │
│          │                                                       │
├──────────┼──────────────────────────────────────────────────────┤
│          │                                                       │
│  STEP 5  │  View metrics dashboard (CPU, memory, requests)       │
│  Monitor │  Check logs (search, filter, stream)                  │
│          │  Set up alerts (Business+)                            │
│          │  View pod health                                      │
│          │                                                       │
├──────────┼──────────────────────────────────────────────────────┤
│          │                                                       │
│  STEP 6  │  Scale app replicas                                   │
│  Scale   │  Upgrade plan for more resources                      │
│          │  Add compute/storage via add-ons                      │
│          │  Auto-scaling kicks in (Business+)                    │
│          │                                                       │
├──────────┼──────────────────────────────────────────────────────┤
│          │                                                       │
│  ALT     │  Use Zenith CLI: `zenith deploy`, `zenith logs`       │
│  PATH    │  Use Terraform: `resource "zenith_app" { ... }`       │
│          │  Use GitHub Actions: zenith-deploy action              │
│          │                                                       │
└──────────┴──────────────────────────────────────────────────────┘
```

### Customer Dashboard Pages

```
┌─────────────────────────────────────────────────────────────────┐
│  ZENITH DASHBOARD                                    [user@co]  │
├──────────────┬──────────────────────────────────────────────────┤
│              │                                                   │
│  Overview    │  Stats: Apps, Databases, Storage, Health          │
│  ──────────  │  Plan usage bars                                  │
│  Apps        │  Quick actions                                    │
│  Databases   │                                                   │
│  Storage     │                                                   │
│  Gateway     │                                                   │
│  Auth        │                                                   │
│  ──────────  │                                                   │
│  Monitoring  │                                                   │
│  Logs        │                                                   │
│  ──────────  │                                                   │
│  Registry    │                                                   │
│  Queues      │                                                   │
│  ──────────  │                                                   │
│  Team (RBAC) │                                                   │
│  Settings    │                                                   │
│  Billing     │                                                   │
│  Support     │                                                   │
│  Docs        │                                                   │
│              │                                                   │
└──────────────┴──────────────────────────────────────────────────┘
```

### Customer Tools

#### Zenith CLI (`zenith`)

```bash
# Authentication
zenith login                    # Login with browser OAuth
zenith logout                   # Clear credentials
zenith whoami                   # Show current user + plan

# Apps
zenith apps list                # List all apps
zenith apps create my-app       # Create new app
zenith apps deploy my-app       # Deploy latest image
zenith apps logs my-app         # Stream live logs
zenith apps env set KEY=VALUE   # Set environment variable
zenith apps scale my-app --replicas 3

# Databases
zenith db list                  # List databases
zenith db create --engine postgres --name my-db
zenith db connect my-db         # Open psql shell
zenith db backup my-db          # Create backup
zenith db restore my-db --backup-id bk_123

# Storage
zenith storage list             # List buckets
zenith storage create my-bucket
zenith storage upload my-bucket ./file.txt
zenith storage download my-bucket file.txt

# Domains
zenith domains add my-app example.com
zenith domains list

# Monitoring
zenith metrics my-app           # Show current metrics
zenith logs my-app --since 1h   # View logs
zenith pods my-app              # List pods

# Project
zenith projects list
zenith projects switch my-project
```

#### Terraform Provider

```hcl
terraform {
  required_providers {
    zenith = {
      source  = "freezenith/zenith"
      version = "~> 1.0"
    }
  }
}

provider "zenith" {
  api_key = var.zenith_api_key
  # or: api_url = "https://api.freezenith.com"
}

resource "zenith_app" "backend" {
  name      = "my-backend"
  image     = "my-org/my-backend:v1.2.3"
  port      = 8080
  replicas  = 2

  env = {
    DATABASE_URL = zenith_database.main.connection_string
    REDIS_URL    = zenith_redis.cache.connection_string
  }

  domain {
    name = "api.example.com"
  }
}

resource "zenith_database" "main" {
  name   = "my-db"
  engine = "postgresql"
  size   = "5GB"
}

resource "zenith_redis" "cache" {
  name = "my-cache"
  size = "1GB"
}

resource "zenith_storage" "uploads" {
  name   = "user-uploads"
  access = "private"
}

resource "zenith_gateway" "api" {
  name = "api-gateway"

  route {
    path    = "/api/v1/*"
    target  = zenith_app.backend.id
    auth    = "jwt"
    plugins = ["rate-limit", "cors"]
  }
}
```

---

## A4. What the Owner Sees

As Zenith owners/operators, we need different dashboards and tools.

### Business Metrics Dashboard

```
┌──────────────────────────────────────────────────────────────────┐
│  ZENITH MISSION CONTROL                         [admin@zenith]   │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│  │   MRR   │ │ Active  │ │  Churn  │ │ Deploys │ │  Build  │   │
│  │ €12,450 │ │  Users  │ │  Rate   │ │  /day   │ │ Success │   │
│  │  ↑12%   │ │   347   │ │  2.1%   │ │   89    │ │  97.2%  │   │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘   │
│                                                                   │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│  │  Cost/  │ │ Support │ │Customer │ │  Nodes  │ │ Storage │   │
│  │Customer │ │  MTTR   │ │ Growth  │ │  Count  │ │  Usage  │   │
│  │  €4.20  │ │  23min  │ │ +18/mo  │ │    6    │ │  340GB  │   │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘   │
│                                                                   │
│  Revenue Breakdown:                                               │
│  ├── Pro:      43 users × €29    = €1,247                        │
│  ├── Team:     12 seats × €99    = €1,188                        │
│  ├── Business: 52 seats × €149   = €7,748                        │
│  ├── Gold:      2 users × €699   = €1,398                        │
│  ├── Platinum:  1 user  × €1,499 = €1,499                        │
│  └── Add-ons:                    = €370                           │
│                                                                   │
│  Hetzner Cost: €890/mo    Margin: 93%                            │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

### Alerting Stack

```
Event Severity → Destination:

  CRITICAL  (system down)     → PagerDuty (phone call to on-call)
  WARNING   (degraded)        → Slack (#platform-alerts)
  BUSINESS  (daily summary)   → Telegram (Babak)
  SECURITY  (anomaly)         → Slack (#security) + PagerDuty
```

| Metric | Warning Threshold | Critical Threshold |
|--------|------------------|--------------------|
| API Error Rate | > 1% | > 5% |
| API P99 Latency | > 500ms | > 2000ms |
| Build Queue Depth | > 10 | > 50 |
| Node CPU | > 70% | > 90% |
| Node Memory | > 75% | > 90% |
| CNPG Replication Lag | > 5s | > 30s |
| Certificate Expiry | < 14 days | < 3 days |
| S3 Storage | > 70% | > 90% |
| Pod CrashLoopBackoff | any | sustained > 5min |

---

# PART B: ARCHITECTURE & DESIGN

---

## B1. High-Level System Architecture

```
                              ┌─────────────────────┐
                              │      INTERNET        │
                              └──────────┬──────────┘
                                         │
                              ┌──────────▼──────────┐
                              │   Cloudflare (CDN)   │  ← DDoS protection
                              │   WAF + DNS proxy    │  ← Web Application Firewall
                              │   DNS-01 challenges  │  ← TLS cert validation
                              └──────────┬──────────┘
                                         │
                    ┌────────────────────▼────────────────────┐
                    │         INGRESS CONTROLLER              │
                    │    TLS termination, L7 routing          │
                    │    IngressRoute CRDs                    │
                    ├────────────┬───────────────────────────┤
                    │            │                            │
              ┌─────▼─────┐  ┌──▼──────────────┐  ┌────────▼────────┐
              │ Frontend   │  │  API GATEWAY     │  │  Platform UIs    │
              │ Routes     │  │  JWT verify      │  │  (Admin, Docs)   │
              │ (Direct)   │  │  Rate limiting   │  │                  │
              │            │  │  CORS            │  │                  │
              └─────┬──────┘  │  OIDC            │  └─────────────────┘
                    │         └──────┬───────────┘
                    │                │
              ┌─────▼──────┐  ┌─────▼──────────────────────────────┐
              │ Customer   │  │           ZENITH API                │
              │ Frontend   │  │  Go backend (Fiber)                 │
              │ Apps       │  │  ├── Auth (JWT + OAuth)             │
              │            │  │  ├── Apps (CRUD + Deploy)           │
              └────────────┘  │  ├── Databases (Provision)          │
                              │  ├── Storage (S3 proxy)             │
                              │  ├── Gateway (Route mgmt)           │
                              │  ├── Billing (Stripe)               │
                              │  ├── Team (IAM + RBAC)              │
                              │  ├── Monitoring (Metrics proxy)     │
                              │  └── Plan Orchestrator (Temporal)   │
                              └──────┬──────────────────────────────┘
                                     │
               ┌─────────────────────┼──────────────────────┐
               │                     │                      │
        ┌──────▼──────┐    ┌────────▼────────┐    ┌───────▼───────┐
        │  DATA LAYER │    │  IDENTITY LAYER  │    │  INFRA LAYER  │
        │             │    │                  │    │               │
        │ PostgreSQL  │    │ Identity Provider│    │ K8s API       │
        │ (Operator)  │    │ (Operator)       │    │ Container Reg │
        │ Redis       │    │ Realms/Clients   │    │ Build System  │
        │ (Operator)  │    │ SSO/OIDC/SAML    │    │ DNS Manager   │
        │ MongoDB     │    │                  │    │ TLS Manager   │
        │ (Operator)  │    └──────────────────┘    │ Policy Engine │
        │ RabbitMQ    │                            │ Secret Manager│
        │ (Operator)  │                            └───────────────┘
        │ Kafka       │
        │ (Operator)  │         ┌──────────────────────────┐
        └─────────────┘         │    OBSERVABILITY LAYER    │
                                │                           │
                                │  Metrics Collector        │
                                │  (Operator)               │
                                │  Log Aggregator           │
                                │  (Operator)               │
                                │  Trace Collector          │
                                │  (Operator)               │
                                │  Dashboards               │
                                │  Alert Manager            │
                                └──────────────────────────┘
```

### Key Design Principles

1. **Operator-First:** Every stateful service uses a Kubernetes Operator (not plain Helm). Operators provide self-healing, automated upgrades, backup management, and CRD-based configuration.

2. **Multi-Tenant Security:** Customer workloads are isolated via namespaces (Free/Pro/Team) or dedicated infrastructure (Business/Enterprise). Network policies block cross-tenant traffic.

3. **API-as-Proxy:** The Zenith API acts as a secure multi-tenant proxy. Customers never directly access Prometheus, Loki, or K8s API. All queries are scoped to the customer's own resources.

4. **Event-Driven:** Internal operations (deploy, scale, notify) flow through a message queue. This ensures concurrent purchases and deployments don't conflict.

5. **Rebuildable from Git:** The entire platform can be rebuilt from scratch using Terraform + ArgoCD + Sealed Secrets. No manual configuration lives outside Git.

6. **Day-2 First:** Every architectural decision prioritizes maintainability, upgradability, and automated operations over initial setup simplicity.

---

## B2. Component Stack (8 Layers)

Every component uses an **Operator** where one exists. This is non-negotiable.

```
╔══════════════════════════════════════════════════════════════════╗
║  LAYER 0: EDGE                                                   ║
║  ┌────────────┐                                                  ║
║  │ Cloudflare │ CDN, WAF, DDoS, DNS proxy, DNS-01 challenges    ║
║  └────────────┘                                                  ║
╠══════════════════════════════════════════════════════════════════╣
║  LAYER 1: NETWORKING                                             ║
║  ┌────────────┐ ┌──────────────┐ ┌─────────────┐ ┌────────────┐║
║  │ Ingress    │ │ API Gateway  │ │ CNI + Network│ │ DNS Auto   │║
║  │ Controller │ │ (Operator)   │ │ Policy       │ │ Manager    │║
║  │ (Traefik)  │ │ JWT, CORS,   │ │ WireGuard    │ │            │║
║  │            │ │ Rate-limit   │ │ L7 Filtering │ │            │║
║  └────────────┘ └──────────────┘ └─────────────┘ └────────────┘║
╠══════════════════════════════════════════════════════════════════╣
║  LAYER 2: IDENTITY & SECURITY                                   ║
║  ┌────────────┐ ┌──────────────┐ ┌─────────────┐ ┌────────────┐║
║  │ Identity   │ │ TLS Automate │ │ Policy       │ │ Runtime    │║
║  │ Provider   │ │ (Operator)   │ │ Engine       │ │ Security   │║
║  │ (Operator) │ │ Let's Encrypt│ │ (Admission)  │ │ (Anomaly   │║
║  │ SSO/OIDC   │ │              │ │ Image Verify │ │  Detection)│║
║  └────────────┘ └──────────────┘ └─────────────┘ └────────────┘║
║  ┌────────────┐ ┌──────────────┐                                ║
║  │ Encrypted  │ │ Secret       │                                ║
║  │ Secrets    │ │ Manager      │                                ║
║  │ (GitOps)   │ │              │                                ║
║  └────────────┘ └──────────────┘                                ║
╠══════════════════════════════════════════════════════════════════╣
║  LAYER 3: DATA                                                   ║
║  ┌────────────┐ ┌──────────────┐ ┌─────────────┐ ┌────────────┐║
║  │ PostgreSQL │ │ Redis        │ │ MongoDB      │ │ RabbitMQ   │║
║  │ (Operator) │ │ (Operator)   │ │ (Operator)   │ │ (Operator) │║
║  │ CNPG       │ │ Spotahome    │ │ Percona      │ │ RabbitMQ   │║
║  │ WAL→S3     │ │ Sentinel     │ │ Replica Set  │ │ Cluster Op │║
║  └────────────┘ └──────────────┘ └─────────────┘ └────────────┘║
║  ┌────────────┐ ┌──────────────┐                                ║
║  │ Kafka      │ │ Object       │                                ║
║  │ (Operator) │ │ Storage      │                                ║
║  │ Strimzi    │ │ (Hetzner S3) │                                ║
║  └────────────┘ └──────────────┘                                ║
╠══════════════════════════════════════════════════════════════════╣
║  LAYER 4: PLATFORM (Zenith's Own Services)                       ║
║  ┌────────────┐ ┌──────────────┐ ┌─────────────┐ ┌────────────┐║
║  │ Zenith API │ │ Zenith Web   │ │ Mission Ctrl│ │ Landing    │║
║  │ (Go/Fiber) │ │ (Next.js)    │ │ (Next.js)   │ │ (Next.js)  │║
║  └────────────┘ └──────────────┘ └─────────────┘ └────────────┘║
║  ┌────────────┐ ┌──────────────┐ ┌─────────────┐               ║
║  │ Zenith     │ │ Workflow     │ │ Container   │               ║
║  │ Operator   │ │ Engine       │ │ Registry    │               ║
║  │ (CRDs)     │ │ (Temporal)   │ │ (Harbor)    │               ║
║  └────────────┘ └──────────────┘ └─────────────┘               ║
║  ┌────────────┐ ┌──────────────┐                                ║
║  │ GitOps     │ │ Internal     │                                ║
║  │ Engine     │ │ Message Queue│                                ║
║  │ (ArgoCD)   │ │ (NATS)       │                                ║
║  └────────────┘ └──────────────┘                                ║
╠══════════════════════════════════════════════════════════════════╣
║  LAYER 5: OBSERVABILITY                                          ║
║  ┌────────────┐ ┌──────────────┐ ┌─────────────┐ ┌────────────┐║
║  │ Metrics    │ │ Log Aggregatr│ │ Trace Store  │ │ Telemetry  │║
║  │ Collector  │ │ (Operator)   │ │ (Operator)   │ │ Collector  │║
║  │ (Operator) │ │ S3 backend   │ │ S3 backend   │ │ (Operator) │║
║  │ Prometheus │ │              │ │              │ │ OTel       │║
║  └────────────┘ └──────────────┘ └─────────────┘ └────────────┘║
║  ┌────────────┐ ┌──────────────┐ ┌─────────────┐               ║
║  │ Dashboards │ │ Alert Manager│ │ Network Flow│               ║
║  │ (Grafana)  │ │              │ │ (Hubble)    │               ║
║  └────────────┘ └──────────────┘ └─────────────┘               ║
╠══════════════════════════════════════════════════════════════════╣
║  LAYER 6: RESILIENCE & BACKUP                                    ║
║  ┌────────────┐ ┌──────────────┐ ┌─────────────┐               ║
║  │ Cluster    │ │ DB WAL       │ │ Per-Customer │               ║
║  │ Backup     │ │ Archiving    │ │ pg_dump      │               ║
║  │ (Velero)   │ │ (CNPG→S3)   │ │ CronJobs     │               ║
║  └────────────┘ └──────────────┘ └─────────────┘               ║
║  ┌────────────┐ ┌──────────────┐                                ║
║  │ DR: Rebuild│ │ DR: Live     │                                ║
║  │ 30min      │ │ (Prod only)  │                                ║
║  │ from Git   │ │ Finland      │                                ║
║  └────────────┘ └──────────────┘                                ║
╠══════════════════════════════════════════════════════════════════╣
║  LAYER 7: AUTO-SCALING                                           ║
║  ┌────────────┐ ┌──────────────┐ ┌─────────────┐               ║
║  │ Pod Scaling│ │ Node Scaling │ │ Scale-to-    │               ║
║  │ (KEDA)     │ │ (Hetzner     │ │ Zero (Free)  │               ║
║  │ HPA+Custom │ │  Autoscaler) │ │ KEDA HTTP    │               ║
║  └────────────┘ └──────────────┘ └─────────────┘               ║
╚══════════════════════════════════════════════════════════════════╝
```

---

## B3. Operator Migration Map

Current state (Helm Release) → Target state (Operator):

| Component | Current | Target Operator | CRD | Priority |
|-----------|---------|----------------|-----|----------|
| PostgreSQL | CNPG Operator | CNPG (already operator) | Cluster, Backup, ScheduledBackup | ✅ Done |
| cert-manager | cert-manager (already operator) | cert-manager | Certificate, ClusterIssuer | ✅ Done |
| KEDA | KEDA (already operator) | KEDA | ScaledObject, HTTPScaledObject | ✅ Done |
| Kyverno | Kyverno (already operator) | Kyverno | ClusterPolicy, Policy | ✅ Done |
| ArgoCD | ArgoCD (already operator) | ArgoCD | Application, AppProject | ✅ Done |
| Sealed Secrets | Sealed Secrets (already operator) | Sealed Secrets | SealedSecret | ✅ Done |
| APISIX | Helm Release + Ingress Controller | APISIX Ingress Controller (CRDs) | ApisixRoute, ApisixUpstream | ✅ Done |
| Loki | Helm Release (SingleBinary) | **Loki Operator** | LokiStack | 🔴 P1 |
| OTel Collector | Helm Release (DaemonSet) | **OTel Operator** | OpenTelemetryCollector, Instrumentation | 🔴 P1 |
| Tempo | Helm Release (Monolithic) | **Tempo Operator** | TempoStack | 🟡 P2 |
| Keycloak | Helm Release | **Keycloak Operator** | Keycloak, KeycloakRealmImport | 🟡 P2 |
| Redis | Not deployed yet | **Redis Operator** (Spotahome) | RedisFailover | 🟡 P2 |
| MongoDB | Not deployed yet | **Percona MongoDB Operator** | PerconaServerMongoDB | 🟢 P3 |
| RabbitMQ | Not deployed yet | **RabbitMQ Cluster Operator** | RabbitmqCluster | 🟢 P3 |
| Kafka | Not deployed yet | **Strimzi Kafka Operator** | Kafka, KafkaTopic, KafkaUser | 🟢 P3 |
| Prometheus | kube-prometheus-stack (has operator) | Prometheus Operator (already bundled) | Prometheus, ServiceMonitor | ✅ Done |

---

## B4. Namespace Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│                    K8s CLUSTER (k3s / k8s)                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  SYSTEM NAMESPACES (managed by k3s/k8s):                         │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│  │ kube-system  │ │ kube-public  │ │ kube-node-   │             │
│  │ (Traefik,    │ │              │ │ lease        │             │
│  │  CoreDNS)    │ │              │ │              │             │
│  └──────────────┘ └──────────────┘ └──────────────┘             │
│                                                                  │
│  PLATFORM NAMESPACES (managed by Terraform):                     │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│  │ zenith-      │ │ zenith-      │ │ monitoring   │             │
│  │ platform     │ │ staging      │ │ (Prom, Loki, │             │
│  │ (API, Web,   │ │ (Staging-    │ │  Tempo, OTel,│             │
│  │  Landing, MC)│ │  specific)   │ │  Grafana)    │             │
│  └──────────────┘ └──────────────┘ └──────────────┘             │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│  │ argocd       │ │ cert-manager │ │ sealed-      │             │
│  │              │ │              │ │ secrets      │             │
│  └──────────────┘ └──────────────┘ └──────────────┘             │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│  │ keycloak     │ │ temporal     │ │ harbor       │             │
│  └──────────────┘ └──────────────┘ └──────────────┘             │
│  ┌──────────────┐ ┌──────────────┐                               │
│  │ kyverno      │ │ falco        │                               │
│  └──────────────┘ └──────────────┘                               │
│                                                                  │
│  SHARED CUSTOMER NAMESPACES (Free/Pro/Team):                     │
│  ┌──────────────┐ ┌──────────────┐                               │
│  │ zenith-apps  │ │ zenith-builds│                               │
│  │ (Deployments,│ │ (Kaniko jobs)│                               │
│  │  Services)   │ │              │                               │
│  └──────────────┘ └──────────────┘                               │
│  ┌──────────────┐                                                │
│  │ zenith-shared│ (Shared CNPG clusters, cold-start page)        │
│  └──────────────┘                                                │
│                                                                  │
│  DEDICATED CUSTOMER NAMESPACES (Business):                       │
│  ┌──────────────────────────────────────┐                        │
│  │ zenith-customer-<id>                 │                        │
│  │ ├── Deployments (customer apps)      │                        │
│  │ ├── Services                         │                        │
│  │ ├── CNPG Cluster (dedicated DB)      │                        │
│  │ ├── Redis (if ordered)               │                        │
│  │ ├── RabbitMQ (if ordered)            │                        │
│  │ ├── ResourceQuota                    │                        │
│  │ ├── LimitRange                       │                        │
│  │ ├── NetworkPolicy (Cilium)           │                        │
│  │ └── Secrets                          │                        │
│  └──────────────────────────────────────┘                        │
│                                                                  │
│  NETWORK NAMESPACE:                                              │
│  ┌──────────────┐                                                │
│  │ cilium       │ (CNI + Hubble + WireGuard)                     │
│  └──────────────┘                                                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## B5. Request Flow Paths

### Path 1: Customer API Request (Protected)

```
Customer Browser/CLI
       │
       ▼
  Cloudflare (WAF + CDN)
       │
       ▼
  Traefik (:443, TLS termination)
       │  Matches IngressRoute for api.freezenith.com
       ▼
  API Gateway (APISIX)
       │  Plugins execute in order:
       │  1. CORS check
       │  2. Rate limiting (per IP/user)
       │  3. JWT verification (Keycloak JWKS)
       │  4. Request ID injection
       ▼
  Zenith API (Go/Fiber, port 8080)
       │  Middleware: extract user_id from JWT
       │  Middleware: check app ownership
       │  Middleware: check plan limits
       │  Handler: process request
       ▼
  Data Layer (PostgreSQL / S3 / K8s API)
       │
       ▼
  Response → reverse path → Customer
```

### Path 2: Customer App Request (User's deployed app)

```
End User (customer's user)
       │
       ▼
  Cloudflare (WAF + CDN)
       │
       ▼
  Traefik (:443, TLS termination)
       │  Matches IngressRoute for {app}.apps.freezenith.com
       │  OR customer custom domain
       ▼
  Customer's App Pod (port 8080/3000/etc)
       │
       ▼
  Response → reverse path → End User
```

### Path 3: Plan Upgrade (Subscription Lifecycle)

```
Customer clicks "Upgrade to Business"
       │
       ▼
  Zenith API → creates Stripe Checkout Session
       │
       ▼
  Stripe Hosted Checkout Page
       │  Customer pays
       ▼
  Stripe Webhook → Zenith API
       │
       ▼
  Plan Orchestrator (Temporal Workflow):
       │
       ├── Step 1: Verify payment (Stripe API)
       ├── Step 2: Update plan in database
       ├── Step 3: Create dedicated namespace (Business)
       ├── Step 4: Provision dedicated CNPG cluster
       ├── Step 5: Migrate customer apps to new namespace
       ├── Step 6: Update API Gateway rate limits
       ├── Step 7: Enable feature flags (SSO, audit, etc.)
       ├── Step 8: Update Policy Engine rules
       └── Step 9: Send confirmation email + in-app notification

  ℹ️  Temporal ensures idempotency — if Step 5 fails,
      it retries from Step 5 (not from Step 1)
```

### Path 4: Image Push + Security Scan

```
Customer builds image (GitHub Actions / local Docker)
       │
       ▼
  docker push hub.freezenith.com/customer-project/my-app:v1.2
       │
       ▼
  Harbor receives image
       │  Webhook triggers automatic scan
       ▼
  Trivy Scanner (Harbor built-in)
       │
       ├── PASS (no critical/high CVEs)
       │   → Label: "verified"
       │   → Customer notified: "Image ready to deploy"
       │
       └── FAIL (critical CVEs found)
           → Label: "rejected"
           → Customer notified: "Security issues found"
           → Details: CVE list, severity, fix suggestions

  Deploy Request:
       │
       ▼
  Policy Engine (Kyverno) checks:
       ├── Is image from our Harbor? ✓
       ├── Does image have "verified" label? ✓
       ├── Is SecurityContext set correctly? ✓
       └── → Allow deploy

       If ANY check fails → Reject with explanation
```

---

## B6. Security Architecture (9 Layers)

```
┌──────────────────────────────────────────────────────────────┐
│  LAYER 1: EDGE                                                │
│  Cloudflare WAF + DDoS protection + Bot management            │
│  ├── Shared tiers: We configure WAF rules                    │
│  └── Business tier: Customer configures their own WAF rules  │
├──────────────────────────────────────────────────────────────┤
│  LAYER 2: NETWORK                                             │
│  Cilium CNI + WireGuard encryption (pod-to-pod)              │
│  ├── Default DENY in customer namespaces                     │
│  ├── Explicit ALLOW: customer → DB, Gateway, DNS             │
│  ├── BLOCK: namespace A ↛ namespace B                        │
│  └── L7 HTTP-aware filtering on sensitive routes             │
│  ├── Shared tiers: We configure firewall rules               │
│  └── Business tier: Customer configures their own firewall   │
├──────────────────────────────────────────────────────────────┤
│  LAYER 3: API GATEWAY                                         │
│  APISIX: JWT verification, rate limiting, CORS               │
│  ├── Per-route plugin configuration                          │
│  └── Keycloak JWKS validation                                │
├──────────────────────────────────────────────────────────────┤
│  LAYER 4: APPLICATION AUTH                                    │
│  MFA mandatory for Pro+ (Google Authenticator / TOTP)         │
│  JWT tokens with short expiry + refresh token rotation        │
│  Session management (view, revoke)                            │
│  API key scoping                                              │
├──────────────────────────────────────────────────────────────┤
│  LAYER 5: CONTAINER SECURITY                                  │
│  SecurityContext on ALL pods:                                 │
│  ├── runAsNonRoot: true                                      │
│  ├── readOnlyRootFilesystem: true                            │
│  ├── allowPrivilegeEscalation: false                         │
│  └── capabilities: { drop: [ALL] }                           │
│  Pod Security Standards: "restricted" on customer namespaces │
├──────────────────────────────────────────────────────────────┤
│  LAYER 6: IMAGE SECURITY                                      │
│  Harbor Trivy scan on push                                    │
│  Kyverno: deny unscanned images, deny non-Harbor images       │
│  Image signing with cosign (future)                           │
├──────────────────────────────────────────────────────────────┤
│  LAYER 7: RUNTIME SECURITY                                    │
│  Falco: detect anomalous container behavior                   │
│  ├── Shell in container alert                                │
│  ├── Unexpected network connection alert                     │
│  └── File modification in read-only filesystem alert         │
├──────────────────────────────────────────────────────────────┤
│  LAYER 8: DATA ENCRYPTION                                     │
│  etcd encryption at rest (k3s --secrets-encryption)           │
│  PostgreSQL: TLS in transit, encrypted S3 backups             │
│  All inter-pod traffic encrypted (WireGuard)                  │
│  Secrets: AES-256-GCM before storage                          │
├──────────────────────────────────────────────────────────────┤
│  LAYER 9: AUDIT & COMPLIANCE                                  │
│  K8s API audit logs → Log Aggregator                          │
│  Application audit logs (who did what, when)                  │
│  Network flow logs (Hubble) → Metrics Collector               │
│  Compliance dashboard (SOC2, GDPR) for Business+              │
└──────────────────────────────────────────────────────────────┘
```

### SSH to Pods (Business+ Only)

Business and Enterprise customers can SSH into their running pods for debugging:

```
Customer → Zenith Dashboard → "Connect to Pod" button
       │
       ▼
  Zenith API creates secure WebSocket tunnel
       │
       ▼
  kubectl exec (scoped to customer's namespace only)
       │
       ▼
  Terminal session (web-based terminal)
       │
       ▼
  ALL sessions recorded:
  ├── Session ID, user, pod, start/end time
  ├── Full terminal output (asciinema format)
  ├── Stored in audit logs
  └── Accessible from Compliance → SSH Sessions
```

### MFA Enforcement

```
  Pro+  Users:
  ├── On first login after upgrade: "Set up MFA" (mandatory)
  ├── TOTP (Google Authenticator / Authy)
  ├── 10 backup codes generated
  ├── Cannot access dashboard without MFA
  └── MFA bypass for API keys (keys have their own auth)
```

---

## B7. Data Architecture

### PostgreSQL Strategy (CNPG Operator)

```
┌──────────────────────────────────────────────────────┐
│              CNPG OPERATOR (cluster-wide)              │
│              Manages all PostgreSQL clusters           │
├──────────────────────────────────────────────────────┤
│                                                       │
│  PLATFORM CLUSTER (zenith-platform namespace)         │
│  ├── Database: zenith_api (API data, all repos)      │
│  ├── Database: temporal (Temporal state)              │
│  ├── Database: temporal_visibility                   │
│  ├── Storage: 10Gi Hetzner Volume                    │
│  ├── Backups: WAL → Hetzner S3, pg_dump daily        │
│  └── Instances: 2 (primary + standby)                │
│                                                       │
│  KEYCLOAK CLUSTER (keycloak namespace)                │
│  ├── Database: keycloak                              │
│  ├── Storage: 10Gi Hetzner Volume                    │
│  ├── Backups: WAL → Hetzner S3                       │
│  └── Instances: 2 (primary + standby)                │
│                                                       │
│  FREE-TIER CLUSTER (zenith-shared namespace)          │
│  ├── All free-tier customer databases                │
│  ├── Storage: 20Gi Hetzner Volume                    │
│  ├── Max connections: 200                            │
│  └── Instances: 1 (single)                           │
│                                                       │
│  PRO-TIER CLUSTERS (zenith-shared namespace)          │
│  ├── Sharded: ~20 Pro users per cluster              │
│  ├── Storage: 50Gi Hetzner Volume each               │
│  ├── Max connections: 400                            │
│  ├── Backups: WAL → S3, pg_dump per customer         │
│  └── Instances: 2 (primary + standby)                │
│                                                       │
│  BUSINESS/ENTERPRISE (zenith-customer-<id> namespace) │
│  ├── Dedicated CNPG Cluster per customer             │
│  ├── Storage: based on plan                          │
│  ├── Backups: WAL → S3, pg_dump, configurable        │
│  └── Instances: 2-3 (HA)                             │
│                                                       │
└──────────────────────────────────────────────────────┘
```

### Full Data Services Matrix

| Service | Operator | Provision Model | Available From |
|---------|----------|----------------|----------------|
| **PostgreSQL** | CNPG | Per-customer DB or dedicated cluster | Free+ |
| **Redis** | Spotahome Redis Operator | RedisFailover CR per customer | Pro+ |
| **MongoDB** | Percona Server MongoDB Operator | PerconaServerMongoDB CR | Team+ |
| **RabbitMQ** | RabbitMQ Cluster Operator | RabbitmqCluster CR per customer | Pro+ |
| **Kafka** | Strimzi Kafka Operator | Kafka CR + KafkaTopic CRs | Business+ |
| **S3 Storage** | Hetzner S3 (API) | Bucket per customer | Pro+ |

---

## B8. Disaster Recovery

### Strategy: Two-Layer DR

```
┌──────────────────────────────────────────────────────┐
│  LAYER 1: REBUILD (30 minutes)                        │
│                                                       │
│  Everything stored in Git + S3:                       │
│  ├── Terraform state → S3                            │
│  ├── Sealed Secrets → Git (encrypted)                │
│  ├── ArgoCD apps → Git                               │
│  ├── Helm values → Git                               │
│  ├── DB backups → S3 (WAL + pg_dump)                 │
│  └── Customer images → Harbor (S3 backend)           │
│                                                       │
│  Rebuild procedure:                                   │
│  1. terraform apply (new Hetzner VM in Finland)  5min│
│  2. ansible-playbook (k3s + Cilium)              5min│
│  3. terraform apply (Helm releases)             10min│
│  4. ArgoCD syncs all apps                        5min│
│  5. CNPG restores from S3 WAL                    5min│
│  Total: ~30 minutes                                  │
│                                                       │
│  ✅ Used for: Staging + Prod                          │
│  ✅ Tested: Monthly automated drill                   │
└──────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────┐
│  LAYER 2: LIVE DR (Production Only)                   │
│                                                       │
│  Secondary cluster in Hetzner Finland:                │
│  ├── CNPG: streaming replication (primary→standby)   │
│  ├── S3: cross-region replication                    │
│  ├── ArgoCD: synced from same Git branch             │
│  ├── DNS: Cloudflare health check → auto failover    │
│  │                                                   │
│  Mode: DORMANT (saves cost)                          │
│  ├── Cluster exists but scaled to minimum            │
│  ├── DB replication is active (real-time)            │
│  ├── Apps are NOT running (0 replicas)               │
│  │                                                   │
│  Testing: Every 10 days                              │
│  ├── Automated pipeline scales up DR cluster          │
│  ├── Runs comprehensive smoke test suite             │
│  ├── Validates: API health, DB queries, S3 access    │
│  ├── Generates report → Slack/Telegram               │
│  ├── If PASS → scale back to dormant                 │
│  ├── If FAIL → alert + investigation                 │
│  └── Pipeline: GitHub Actions + custom smoke runner  │
│                                                       │
│  Failover procedure (automated):                     │
│  1. Cloudflare health check detects primary down     │
│  2. DNS failover to Finland IP                 ~2min │
│  3. DR cluster scales up (replicas: 0 → target) 3min │
│  4. CNPG promotes standby to primary           ~1min │
│  5. Traffic flows to DR cluster                      │
│  Total: ~6 minutes                                   │
│                                                       │
│  ⚠️ Production only (staging has no live DR)          │
└──────────────────────────────────────────────────────┘
```

### DR Smoke Test Pipeline (Automated)

```yaml
# Runs every 10 days via GitHub Actions
name: DR Smoke Test
schedule: "0 3 */10 * *"  # 3 AM every 10 days

steps:
  1. Scale DR cluster → active replicas
  2. Wait for all pods ready (timeout: 10min)
  3. Run smoke test suite:
     ├── API /health endpoint responds 200
     ├── Can query PostgreSQL (read from replica)
     ├── Can list S3 buckets
     ├── Can resolve DNS records
     ├── Can issue TLS certificate
     ├── ArgoCD app sync status = Healthy
     └── All ServiceMonitors scraping
  4. Generate report (JSON + markdown)
  5. Post to Slack (#dr-reports) + Telegram
  6. Scale DR cluster → dormant (0 replicas)
  7. If any check failed:
     ├── Alert: PagerDuty
     ├── Don't scale down (keep for investigation)
     └── Create GitHub Issue automatically
```

---

## B9. k3s → k8s Migration Path

Current: **k3s** (lightweight K8s)
Future: **k8s** (standard Kubernetes) when scale demands it

### Migration must be smooth:

```
Phase 1: k3s (current)
  ├── Single-node staging
  ├── Multi-node production
  ├── All operators work on k3s
  └── Traefik as ingress (built-in)

Phase 2: k8s (when needed)
  ├── Provision k8s cluster via CAPI+CAPH
  ├── Install same Terraform modules (identical Helm charts)
  ├── Migrate data:
  │   ├── CNPG: promote new cluster from S3 WAL backup
  │   ├── Sealed Secrets: same sealed secrets from Git
  │   ├── ArgoCD: point to same Git repo
  │   └── Harbor: same S3 backend (images available instantly)
  ├── DNS cutover (Cloudflare → new IP)
  └── Validate: smoke test suite

Why this works:
  ├── We use standard K8s APIs (not k3s-specific features)
  ├── Traefik CRDs work on both k3s and k8s
  ├── All operators are standard K8s operators
  ├── Terraform modules are k8s-agnostic
  └── Only difference: k3s bundles Traefik, k8s needs explicit install
```

---

# PART C: IMPLEMENTATION PHASES

> Each phase has: Description, Design Changes, Tasks (checkboxes), Validation

---

## Phase 0: Foundation (Critical Production Blockers)

**Goal:** Fix the 18 memory-only repositories and harden K8s deployments before anything else.

**Why first:** Without persistent storage, any pod restart loses all data. Without SecurityContext, containers run as root. These are non-negotiable for production.

### Design Change: Repository Migration

```
BEFORE (memory):                      AFTER (PostgreSQL):
┌──────────────────┐                  ┌──────────────────┐
│ In-Memory Map    │                  │ PostgreSQL Table  │
│ (lost on restart)│      →→→         │ (persistent)      │
│ var store = {}   │                  │ WITH migrations   │
└──────────────────┘                  └──────────────────┘

All 18 repositories → PostgreSQL adapters with SQL migrations
```

### Tasks

- [x] **P0-01** Create PostgreSQL adapter for `SessionRepository`
- [x] **P0-02** Create PostgreSQL adapter for `MFARepository`
- [x] **P0-03** Create PostgreSQL adapter for `APIKeyRepository`
- [x] **P0-04** Create PostgreSQL adapter for `BillingRepository` (subscriptions, invoices)
- [x] **P0-05** Create PostgreSQL adapter for `UserWebhookRepository`
- [x] **P0-06** Create PostgreSQL adapter for `RoleRepository` (custom roles, assignments)
- [x] **P0-07** Create PostgreSQL adapter for `IPWhitelistRepository`
- [x] **P0-08** Create PostgreSQL adapter for `SSORepository`
- [x] **P0-09** Create PostgreSQL adapter for `PreviewRepository`
- [x] **P0-10** Create PostgreSQL adapter for `BrandingRepository` (DPA, branding config)
- [x] **P0-11** Create PostgreSQL adapter for `BackupRepository`
- [x] **P0-12** Create PostgreSQL adapter for `AutoscaleRepository`
- [x] **P0-13** Create PostgreSQL adapter for `AppAuthRepository`
- [x] **P0-14** Create PostgreSQL adapter for `MeteringRepository` *(already existed)*
- [x] **P0-15** Create SQL migrations for all new tables (single migration file)
- [x] **P0-16** Update `main.go` wiring: use PostgreSQL adapters when `DATABASE_URL` is set
- [x] **P0-17** Add SecurityContext to ALL Helm chart deployments (API, Web, Landing, MC, Operator)
- [x] **P0-18** Add podAntiAffinity to all multi-replica deployments
- [x] **P0-19** Add PodDisruptionBudget to all deployments (even single-replica)
- [x] **P0-20** Run `go vet ./... && go build ./...` — must pass clean
- [x] **P0-21** Run `npm run build` for web — must pass clean

### Validation
```bash
# Backend builds
cd services/api && go vet ./... && go build ./...
# Frontend builds
cd apps/web && npm run build
# All Helm charts lint
helm lint infra/helm/zenith-api
helm lint infra/helm/zenith-platform
```

---

## Phase 1: Operator Migration (Observability Stack)

**Goal:** Migrate Loki, OTel Collector, and Tempo from Helm releases to Operator-managed CRDs.

### Design Change: Loki Operator

```
BEFORE:                              AFTER:
┌──────────────────┐                 ┌──────────────────┐
│ Loki Helm Release│                 │ Loki Operator     │
│ SingleBinary     │      →→→        │ LokiStack CRD    │
│ Filesystem store │                 │ S3 backend        │
│ No HA            │                 │ Micro-services    │
└──────────────────┘                 └──────────────────┘
```

### Tasks

- [x] **P1-01** Install Loki Operator via Terraform (Helm release for operator itself)
- [x] **P1-02** Create `LokiStack` CRD resource with S3 backend (Hetzner Object Storage)
- [x] **P1-03** Create S3 bucket for Loki storage (`zenith-loki-{env}`)
- [x] **P1-04** Migrate existing log data or accept clean start (clean start — operator manages new storage)
- [x] **P1-05** Install OTel Operator via Terraform
- [x] **P1-06** Create `OpenTelemetryCollector` CRD (DaemonSet mode for logs/metrics, Deployment mode for traces)
- [x] **P1-07** Create `Instrumentation` CRD for auto-instrumentation (Go, Node.js)
- [x] **P1-08** Install Tempo Operator via Terraform
- [x] **P1-09** Create `TempoStack` CRD with S3 backend
- [x] **P1-10** Create S3 bucket for Tempo storage (`zenith-tempo-{env}`)
- [x] **P1-11** Update Grafana datasources to point to new Loki/Tempo endpoints
- [x] **P1-12** Remove old Helm releases (loki, otel-collector, tempo) from Terraform (disable via enable_monitoring=false after enabling enable_v3_operators=true)
- [ ] **P1-13** Verify: logs visible in Grafana, traces flowing, metrics scraping
- [x] **P1-14** Update monitoring API endpoints in Zenith API (if URLs changed) (already configurable via env vars)

### Validation
```bash
# Loki receiving logs
kubectl -n monitoring get lokistack
kubectl -n monitoring logs -l app=loki-distributor
# OTel collecting
kubectl -n monitoring get opentelemetrycollectors
# Tempo receiving traces
kubectl -n monitoring get tempostacks
# Grafana shows data
curl -s http://grafana.stage.freezenith.com/api/health
```

---

## Phase 2: Operator Migration (Identity & Data)

**Goal:** Migrate Keycloak to Operator. Deploy Redis and MongoDB operators.

### Tasks

- [x] **P2-01** Install Keycloak Operator via Terraform
- [x] **P2-02** Create `Keycloak` CRD (pointing to existing CNPG database)
- [x] **P2-03** Export existing Keycloak realms
- [x] **P2-04** Import realms via `KeycloakRealmImport` CRDs
- [x] **P2-05** Update API config to point to new Keycloak endpoint (if changed) (configurable via env)
- [ ] **P2-06** Validate OAuth login flow works
- [x] **P2-07** Remove old Keycloak Helm release from Terraform (disable via enable_keycloak=false after enabling enable_v3_operators=true)
- [x] **P2-08** Install Redis Operator (Spotahome/OpsTree) via Terraform
- [x] **P2-09** Create API handler for customer Redis provisioning
- [x] **P2-10** Create `RedisFailover` CR template for customer Redis instances
- [x] **P2-11** Add Redis to Plan limits (Pro+: 2 instances, Team+: 5, Business: unlimited)
- [x] **P2-12** Add Redis page to web dashboard (`/redis` or integrate into `/databases`)
- [x] **P2-13** Install Percona MongoDB Operator via Terraform
- [x] **P2-14** Create API handler for customer MongoDB provisioning
- [x] **P2-15** Create `PerconaServerMongoDB` CR template for customer instances
- [x] **P2-16** Add MongoDB to Plan limits (Team+: 2 instances, Business: unlimited)
- [x] **P2-17** Add MongoDB to web dashboard

### Validation
```bash
# Keycloak Operator managing instance
kubectl -n keycloak get keycloaks
# Login flow works
curl -X POST https://api.stage.freezenith.com/api/v1/auth/login
# Redis Operator ready
kubectl get redisfailovers --all-namespaces
# MongoDB Operator ready
kubectl get perconaservermongodbs --all-namespaces
```

---

## Phase 3: Message Queues & Internal Event Bus

**Goal:** Deploy NATS for internal events. Deploy RabbitMQ and Kafka operators for customers.

### Design Change: Internal Event Architecture

```
BEFORE:                              AFTER:
┌──────────────────┐                 ┌──────────────────┐
│ Direct function  │                 │ NATS JetStream    │
│ calls between    │      →→→        │ Event bus for:    │
│ services         │                 │ - deploy events   │
│ (coupled)        │                 │ - build events    │
│                  │                 │ - billing events  │
│                  │                 │ - notification    │
└──────────────────┘                 └──────────────────┘
```

### Tasks

- [x] **P3-01** Deploy NATS JetStream via Terraform (Helm release for platform use)
- [x] **P3-02** Create NATS client adapter in API (`adapters/natsclient/`)
- [x] **P3-03** Publish events: deploy.started, deploy.completed, deploy.failed
- [x] **P3-04** Publish events: billing.checkout, billing.upgrade, billing.cancel
- [x] **P3-05** Subscribe: notification service consumes events → sends emails
- [x] **P3-06** Install RabbitMQ Cluster Operator via Terraform
- [x] **P3-07** Create API handler for customer RabbitMQ provisioning
- [x] **P3-08** Create `RabbitmqCluster` CR template for customer instances
- [x] **P3-09** Add RabbitMQ to Plan limits (Pro+: 1 instance, Business: unlimited)
- [x] **P3-10** Add RabbitMQ page to web dashboard (`/queues`)
- [x] **P3-11** Install Strimzi Kafka Operator via Terraform
- [x] **P3-12** Create API handler for customer Kafka provisioning
- [x] **P3-13** Create `Kafka` + `KafkaTopic` + `KafkaUser` CR templates
- [x] **P3-14** Add Kafka to Plan limits (Business+ only)
- [x] **P3-15** Add Kafka to web dashboard (under `/queues`)

### Validation
```bash
# NATS running
kubectl -n zenith-platform get pods -l app=nats
# Publish test event
nats pub deploy.test "hello"
# RabbitMQ Operator
kubectl get rabbitmqclusters --all-namespaces
# Strimzi ready
kubectl get kafkas --all-namespaces
```

---

## Phase 4: Security Hardening

**Goal:** Image security pipeline, WAF configuration, MFA enforcement, SSH recording.

### Tasks

- [x] **P4-01** Enable Harbor Trivy vulnerability scanning (webhook on push)
- [x] **P4-02** Create Kyverno `ClusterPolicy`: deny pods without "verified" scan label
- [x] **P4-03** Create Kyverno `ClusterPolicy`: deny images not from Harbor
- [x] **P4-04** Add scan status to web dashboard (Registry page: show scan results)
- [x] **P4-05** MFA enforcement: login returns mfa_required challenge when MFA enabled, POST /auth/login/mfa completes flow
- [x] **P4-06** Real TOTP validation via pquerna/otp library (replaces accept-any-code stub)
- [x] **P4-07** MFA setup banner for Pro+ users without MFA (dismissible, links to settings)
- [x] **P4-08** Implement SSH-to-pod for Business+ (WebSocket exec tunnel)
- [x] **P4-09** Record all SSH sessions (asciinema format → S3 storage)
- [x] **P4-10** Add SSH Sessions page to Compliance section in dashboard
- [x] **P4-11** Configure Cloudflare WAF rules for shared tiers
- [x] **P4-12** Create API endpoint for Business+ customers to manage their WAF rules
- [x] **P4-13** Add WAF configuration page to Business+ dashboard
- [x] **P4-14** Add firewall (Cilium NetworkPolicy) configuration for Business+
- [x] **P4-15** Expand health check endpoint — removed info disclosure (git commit), added runtime go version
- [x] **P4-16** Implement JWT token blacklist (in-memory, SHA256-hashed) for logout enforcement
- [x] **P4-17** Add rate limiting to all API endpoints (not just auth) — body limit reduced to 50MB, read/write timeouts
- [x] **P4-18** Add OWASP security headers to API responses (CSP, X-Frame-Options, HSTS, Referrer-Policy, Permissions-Policy)
- [x] **P4-19** Add graceful shutdown with 30-second timeout

### Validation
```bash
# Push unscanned image → should be rejected on deploy
docker push hub.stage.freezenith.com/test/bad-image:latest
# → Kyverno should block deployment

# MFA enforcement
curl -X POST /api/v1/auth/login # → should require MFA code for Pro+

# Health check expanded
curl https://api.stage.freezenith.com/api/v1/health
# → should show status of all dependencies
```

---

## Phase 5: CI/CD Improvements

**Goal:** Add Trivy to CI, make security gates blocking, add Terraform approval.

### Tasks

- [x] **P5-01** Add Trivy container scan to `build-images.yml` workflow (API + Web images)
- [x] **P5-02** Make Semgrep security gate blocking (`continue-on-error: false`)
- [x] **P5-03** Terraform plan-only on PR (blocking, posts plan to PR comment)
- [x] **P5-04** Terraform apply gated by `staging-terraform` environment (manual approval)
- [x] **P5-05** Add Helm chart validation step (helm lint --strict in CI)
- [x] **P5-06** Create automated DR smoke test pipeline (smoke-test.yml: customer + owner + infra)
- [x] **P5-07** Schedule DR test every 6 hours (cron in smoke-test.yml)
- [x] **P5-08** DR test reports to Slack + Telegram
- [x] **P5-09** Fix frontend Docker images: runtime env injection (entrypoint.sh + runtime-env.ts)
- [x] **P5-10** Staging→production promotion workflow (smoke test, image retag, Helm version bump, release tag)

### Validation
```bash
# Push vulnerable code → CI should block
# Push vulnerable image → Trivy should fail build
# Terraform PR → plan only, no apply
# DR test → report generated
```

---

## Phase 6: Plan Orchestrator & Business Features

**Goal:** Implement the subscription lifecycle manager and business-tier features.

### Design Change: Plan Orchestrator

```
┌────────────────────────────────────────────────────────────┐
│  PLAN ORCHESTRATOR (Temporal Workflow)                       │
│                                                             │
│  Trigger: Stripe webhook (payment confirmed)                │
│                                                             │
│  Activities (idempotent, retryable):                        │
│  ├── VerifyPayment()     — confirm with Stripe API         │
│  ├── UpdatePlanDB()      — update user plan tier + limits  │
│  ├── ProvisionInfra()    — create namespace (Business)      │
│  ├── ProvisionDB()       — create dedicated CNPG (Business) │
│  ├── MigrateApps()       — move apps to new namespace       │
│  ├── UpdateGateway()     — adjust rate limits               │
│  ├── EnableFeatures()    — turn on SSO, audit, etc.         │
│  ├── UpdatePolicies()    — Kyverno rules for new tier       │
│  └── NotifyUser()        — email + in-app notification      │
│                                                             │
│  On downgrade:                                               │
│  ├── CheckResourceUsage() — verify under new limits         │
│  ├── DisableFeatures()    — turn off SSO, audit, etc.       │
│  ├── MigrateToShared()    — move from dedicated → shared    │
│  └── NotifyUser()         — "Your plan changed" email       │
└────────────────────────────────────────────────────────────┘
```

### Tasks

- [x] **P6-01** Create `PlanOrchestrator` Temporal workflow
- [x] **P6-02** Implement all activities (verify, provision, migrate, notify)
- [x] **P6-03** Wire Stripe webhook to trigger PlanOrchestrator
- [x] **P6-04** Update billing page to show tier comparison with Audi pricing
- [x] **P6-05** Implement Business tier: no limits, cost-based billing
- [x] **P6-06** Implement Team tier (€99/seat, min 3)
- [x] **P6-07** Update landing page pricing to match new tiers (5 tiers, Business featured)
- [x] **P6-08** Implement add-on marketplace API (Gold Support, Extra Compute, etc.)
- [x] **P6-09** Add add-on marketplace page to dashboard
- [x] **P6-10** Create SSO configuration page (Team+) — /sso with SAML/OIDC forms
- [x] **P6-11** Create Audit Log page (Business+) — /audit with search, export CSV/JSON, pagination
- [x] **P6-12** Create Compliance Dashboard — /compliance with score ring, SOC2/ISO27001 readiness
- [x] **P6-13** Update API to support new tier: "business" (between team and enterprise)

### Validation
```bash
# Upgrade to Business → Temporal workflow runs → namespace created
# Downgrade from Business → apps migrated back to shared
# Landing page shows correct pricing
```

---

## Phase 7: Customer Tools

**Goal:** Build Zenith CLI and Terraform Provider.

### Tasks

- [x] **P7-01** Design CLI command structure (see A3 for reference)
- [x] **P7-02** Implement `zenith login` (browser OAuth flow)
- [x] **P7-03** Implement `zenith apps` commands (list, create, deploy, logs, env, scale)
- [x] **P7-04** Implement `zenith db` commands (list, create, connect, backup, restore)
- [x] **P7-05** Implement `zenith storage` commands (list, create, upload, download)
- [x] **P7-06** Implement `zenith domains` commands (add, list)
- [x] **P7-07** Implement `zenith metrics` and `zenith logs` commands
- [x] **P7-08** Build CLI binary for Linux, macOS, Windows (GitHub Releases)
- [x] **P7-09** Add CLI install script: `curl -fsSL https://get.freezenith.com | sh`
- [x] **P7-10** Design Terraform Provider schema (see A3 for reference)
- [x] **P7-11** Implement `zenith_app` resource
- [x] **P7-12** Implement `zenith_database` resource
- [x] **P7-13** Implement `zenith_redis` resource (covered by database resource with engine="redis")
- [x] **P7-14** Implement `zenith_storage` resource
- [x] **P7-15** Implement `zenith_gateway` resource with route blocks
- [x] **P7-16** Implement `zenith_domain` resource
- [x] **P7-17** Publish Terraform Provider to Terraform Registry (goreleaser + signing configured)
- [x] **P7-18** Write documentation for CLI and Terraform Provider (examples/main.tf)

### Validation
```bash
# CLI works
zenith login && zenith apps list
zenith apps create test-app --image nginx:latest
zenith apps logs test-app --since 5m

# Terraform provider works
terraform init && terraform plan && terraform apply
```

---

## Phase 8: Monitoring & Business Metrics

**Goal:** Build the business owner dashboard and customer-facing monitoring improvements.

### Tasks

- [x] **P8-01** Create Grafana dashboards for business metrics (MRR, churn, growth)
- [x] **P8-02** Integrate Stripe API data into Grafana (via Prometheus exporter or direct)
- [x] **P8-03** Create platform health dashboard (API error rate, latency, build queue)
- [x] **P8-04** Set up Alertmanager routes: Critical→PagerDuty, Warning→Slack, Business→Telegram
- [x] **P8-05** Create daily business summary Telegram bot
- [x] **P8-06** Add custom metrics support for Business+ customers (custom Prometheus rules)
- [x] **P8-07** Add custom alert rules for Business+ customers
- [x] **P8-08** Implement structured logging in API (log/slog JSON handler + StructuredLogger middleware)
- [x] **P8-09** Add request tracing (correlation ID via requestid middleware → request_id in all structured logs)
- [x] **P8-10** Create operational runbooks for common incidents

### Validation
```bash
# Grafana dashboards show real data
# Telegram bot sends daily summary
# Structured logs visible in Loki
# Alerts fire correctly (test with synthetic failure)
```

---

## Phase 9: Advanced Features (v2)

**Goal:** Serverless, Blue/Green deploy, multi-region (design now, implement when ready).

### Tasks

- [x] **P9-01** Design serverless API contract (`POST /api/v1/apps/:appId/functions`)
- [x] **P9-02** Evaluate Knative vs KEDA + custom runtime for serverless
- [x] **P9-03** Design Blue/Green deployment strategy (APISIX traffic splitting)
- [x] **P9-04** Design Canary deployment with automatic rollback
- [x] **P9-05** Design multi-region architecture (Hetzner Falkenstein + Helsinki + Ashburn)
- [x] **P9-06** Design service mesh (mTLS between customer services)
- [x] **P9-07** Design GitOps mode (customer repo push → automatic deploy)
- [x] **P9-08** Create OpenSpec proposals for each feature (docs/proposals/P9-future-features.md)

### Validation
```
These are DESIGN tasks — validation is completed proposals, not running code.
```

---

## Phase 10: Production Launch Checklist

**Goal:** Everything needed before first real customer.

### Tasks

- [x] **P10-01** Run penetration test tools: kube-bench, kube-hunter, trivy, kubeaudit (scripts/security-scan.sh)
- [ ] **P10-02** Fix all critical/high findings from pen test
- [ ] **P10-03** Professional penetration test (HackerOne or Bugcrowd)
- [ ] **P10-04** Set up production Hetzner server(s)
- [ ] **P10-05** Deploy production cluster (identical to staging, more resources)
- [ ] **P10-06** Set up production DR (Finland, dormant, 10-day test)
- [ ] **P10-07** Verify all backups (CNPG WAL, Velero, S3)
- [ ] **P10-08** Run full restore drill (restore entire cluster from backup)
- [ ] **P10-09** Set up production monitoring + alerting
- [ ] **P10-10** Load test: simulate 100 concurrent users deploying
- [ ] **P10-11** Update landing page with final content
- [ ] **P10-12** Set up support email + ticketing system
- [x] **P10-13** Create onboarding documentation for customers (docs/onboarding.md)
- [x] **P10-14** Create internal operations handbook (docs/operations-handbook.md)
- [ ] **P10-15** DNS cutover: freezenith.com → production cluster

### Validation
```bash
# Full smoke test passes on production
# DR test passes
# Load test passes (no errors under 100 concurrent users)
# All security scans clean
# Documentation complete
```

---

# PART D: DAY-2 OPERATIONS

---

## D1. Routine Operations

### Daily

| Task | How | Who |
|------|-----|-----|
| Check platform health dashboard | Grafana | Automated (alerts on failure) |
| Review support tickets | Mission Control | On-call engineer |
| Check build queue | Grafana metric | Automated alert if > 10 |
| Review security alerts | Slack #security | On-call engineer |

### Weekly

| Task | How | Who |
|------|-----|-----|
| Review error rate trends | Grafana | Platform engineer |
| Review resource utilization | Grafana | Platform engineer |
| Update Helm chart dependencies | Renovate bot / manual | Platform engineer |
| Review Kyverno policy violations | Kyverno UI | Security engineer |

### Monthly

| Task | How | Who |
|------|-----|-----|
| DR restore drill | Automated pipeline | Platform engineer (verify) |
| Review & rotate secrets | Sealed Secrets | Security engineer |
| Review customer resource usage | Mission Control | Business owner |
| MRR report | Grafana + Stripe | Business owner |
| Certificate expiry check | cert-manager | Automated (alert < 14 days) |

### Every 10 Days

| Task | How | Who |
|------|-----|-----|
| DR smoke test (production) | Automated pipeline | Automated (report to Slack) |

---

## D2. Upgrade Procedures

### Operator Upgrades

Because we use operators, upgrades are simplified:

```
1. Update Terraform Helm release version
2. terraform plan → review changes
3. terraform apply
4. Operator handles rolling upgrade of managed resources
5. Verify: kubectl get <crd> → all resources Healthy

Example (CNPG upgrade):
  - Update cnpg chart version in variables.tf
  - terraform apply
  - CNPG Operator automatically upgrades PostgreSQL pods
  - Zero-downtime (rolling update with failover)
```

### K8s Cluster Upgrade (k3s)

```
1. Review k3s release notes
2. Test on staging first
3. Update Ansible k3s_version variable
4. Run ansible-playbook on staging
5. Verify all operators + workloads healthy
6. Run same on production
7. Keep old version documented for rollback
```

---

## D3. Scaling Procedures

### Horizontal Pod Autoscaling

```
Managed by KEDA:
  Free tier:  HTTPScaledObject (scale to zero, min 0, max 1)
  Pro tier:   ScaledObject (min 1, max 5, based on CPU/memory)
  Team tier:  ScaledObject (min 1, max 10, based on CPU/memory)
  Business:   ScaledObject (min 2, max unlimited, custom metrics)
```

### Node Scaling

```
Managed by Hetzner Autoscaler service:
  Monitor: total cluster CPU > 70% → add node
  Monitor: total cluster CPU < 30% → remove node
  Budget cap: configurable max cost/month
  Cool-down: 5 minutes between scale events
```

### Database Scaling

```
Managed by CNPG Operator:
  Vertical: increase resource limits in Cluster CR → operator applies
  Horizontal: add replicas in Cluster CR → operator creates new pods
  Storage: resize PVC (Hetzner CSI supports volume expansion)
```

---

## D4. Incident Response

### Severity Levels

| Level | Definition | Response Time | Example |
|-------|-----------|--------------|---------|
| SEV1 | Platform down, all customers affected | 10 min | API returning 500, DB cluster down |
| SEV2 | Degraded, some customers affected | 30 min | Build queue stuck, high error rate |
| SEV3 | Single customer issue | 4 hours | Customer app crash loop |
| SEV4 | Cosmetic or minor | Next business day | Dashboard UI glitch |

### Response Flow

```
Alert fires (PagerDuty / Slack)
       │
       ▼
  On-call engineer acknowledges
       │
       ▼
  Triage: determine severity
       │
       ├── SEV1: Start incident call, all hands
       │   ├── Check: kubectl get pods -A (any crashlooping?)
       │   ├── Check: CNPG cluster status
       │   ├── Check: APISIX logs (routing issues?)
       │   ├── Fix or failover to DR
       │   └── Post-mortem within 24 hours
       │
       ├── SEV2: Single engineer investigates
       │   ├── Check relevant component logs
       │   ├── Check Grafana dashboards
       │   └── Fix or escalate to SEV1
       │
       └── SEV3/4: Queue for next available engineer
```

---

## D5. Backup & Restore Procedures

### Backup Schedule

| Data | Method | Frequency | Retention | Storage |
|------|--------|-----------|-----------|---------|
| Platform PostgreSQL | CNPG WAL archiving | Continuous | 30 days | Hetzner S3 |
| Platform PostgreSQL | pg_dump | Daily 2 AM | 14 days | Hetzner S3 |
| Keycloak PostgreSQL | CNPG WAL archiving | Continuous | 30 days | Hetzner S3 |
| Customer PostgreSQL | CNPG WAL + pg_dump | Continuous + Daily | Per plan (7-90 days) | Hetzner S3 |
| K8s etcd | Velero snapshot | Every 6 hours | 14 days | Hetzner S3 |
| Keycloak realms | CRD export CronJob | Daily | 30 days | Hetzner S3 |
| Harbor images | S3 backend (already in S3) | Real-time | Indefinite | Hetzner S3 |
| Hetzner Volumes | Volume snapshots (API) | Weekly | 4 snapshots | Hetzner |

### Restore Procedures

**Single customer database:**
```bash
# From pg_dump
pg_restore -d customer_db backup_file.dump

# From CNPG WAL (point-in-time)
kubectl apply -f - <<EOF
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
spec:
  bootstrap:
    recovery:
      source: original-cluster
      recoveryTarget:
        targetTime: "2026-03-08T15:00:00Z"
EOF
```

**Full cluster:**
```bash
# 1. Rebuild from scratch
terraform apply  # new VM + k3s
# 2. Restore K8s state
velero restore create --from-backup latest
# 3. Verify
kubectl get pods -A
```

---

## D6. Monitoring Reference

### What We Monitor (Platform Owner View)

| Category | Metrics | Dashboard |
|----------|---------|-----------|
| **API Health** | Error rate, latency p50/p95/p99, request rate | Grafana: API Dashboard |
| **K8s Cluster** | Node CPU/memory, pod count, evictions | Grafana: K8s Overview |
| **Databases** | Connection count, replication lag, WAL size | Grafana: CNPG Dashboard |
| **Build System** | Queue depth, build duration, success rate | Grafana: Build Pipeline |
| **Security** | Falco alerts, Kyverno violations, failed logins | Grafana: Security Dashboard |
| **Business** | MRR, users, churn, cost per customer | Grafana: Business Dashboard |
| **Networking** | Request rate per app, bandwidth, error codes | Grafana: APISIX Dashboard |

### What Customer Monitors (Customer View)

| Category | Metrics | Where |
|----------|---------|-------|
| **App Health** | CPU%, memory%, request rate, error rate, latency | Dashboard → Monitoring |
| **Logs** | Application logs with search/filter/stream | Dashboard → Logs |
| **Pods** | Pod status, restarts, resource usage | Dashboard → Monitoring → Pods |
| **Database** | Connection count, storage used | Dashboard → Databases |
| **Storage** | Bucket size, object count | Dashboard → Storage |

---

# APPENDICES

---

## Appendix A: Technology Stack Reference

> **For internal use only.** This maps generic feature names to specific technologies.

| Feature (Customer Sees) | Technology (We Use) | Why This Choice |
|--------------------------|--------------------|-|
| API Gateway | Apache APISIX + etcd | Rich plugins, route-level config, no enterprise licensing |
| Identity Provider | Keycloak | Open source, realm per customer, OIDC/SAML |
| Managed Database (PG) | CloudNativePG (CNPG) | K8s operator, WAL archiving, declarative |
| Managed Cache | Spotahome Redis Operator | RedisFailover CRD, Sentinel HA |
| Document Database | Percona MongoDB Operator | HA replica set, operator-managed |
| Message Queue | RabbitMQ Cluster Operator | CRD-based, operator manages upgrades |
| Event Streaming | Strimzi Kafka Operator | Full Kafka via CRDs, topic/user management |
| Container Registry | Harbor | Open source, Trivy scanning, OCI support |
| Log Aggregation | Grafana Loki (Operator) | LogQL, S3 backend, label-based |
| Metrics Collection | Prometheus (Operator) | PromQL, ServiceMonitor CRDs |
| Trace Collection | Grafana Tempo (Operator) | S3 backend, OTLP native |
| Telemetry Pipeline | OpenTelemetry (Operator) | Auto-instrumentation, vendor neutral |
| Dashboards | Grafana | Universal, supports all datasources |
| TLS Automation | cert-manager | DNS-01 via Cloudflare, automatic renewal |
| Policy Engine | Kyverno | YAML-based policies, no Rego |
| Runtime Security | Falco | eBPF-based syscall monitoring |
| Cluster Backup | Velero | K8s-native backup to S3 |
| Secret Management | Sealed Secrets | GitOps-compatible encrypted secrets |
| GitOps Engine | ArgoCD | App-of-Apps, UI, auto-sync |
| Workflow Engine | Temporal | Durable workflows, retry, visibility |
| CNI | Cilium | WireGuard encryption, L7 policies, Hubble |
| DNS Automation | external-dns | Cloudflare provider, auto A/CNAME |
| Ingress Controller | Traefik (k3s built-in) | IngressRoute CRDs, cross-namespace |
| Internal Queue | NATS JetStream | Lightweight, cloud-native, persistent |
| Network Observability | Hubble (Cilium) | eBPF flow visualization |
| CDN + WAF | Cloudflare | DDoS, bot management, edge caching |
| Cloud Provider | Hetzner Cloud | European, affordable, S3 + Volumes + VMs |
| Payments | Stripe | Checkout, portal, webhooks |
| Email | Resend | Transactional email API |

---

## Appendix B: Environment Parity

> **Rule:** Staging and Production MUST have identical infrastructure. Only resource amounts differ.

| Component | Staging | Production |
|-----------|---------|------------|
| K8s | k3s (1 node) | k3s (3+ nodes) |
| CNPG instances | 1 per cluster | 2-3 per cluster (HA) |
| APISIX replicas | 1 | 2 |
| Prometheus retention | 15 days | 90 days |
| Prometheus storage | 20Gi | 50Gi |
| Loki storage | 10Gi (S3) | 100Gi (S3) |
| Node size | CPX31 (4vCPU/8GB) | CPX51+ (8vCPU/16GB) |
| DR | Rebuild only (30 min) | Rebuild + Live DR (Finland) |
| DR testing | Monthly drill | Every 10 days |

**Everything else is IDENTICAL:**
- Same Terraform modules
- Same Helm charts (different values.yaml)
- Same operators and CRD versions
- Same security policies
- Same monitoring stack
- Same CI/CD pipelines

---

## Appendix C: Glossary for Junior Developers

| Term | Meaning |
|------|---------|
| **Operator** | A K8s controller that watches CRDs and manages the lifecycle of a service (install, upgrade, backup, scale, heal) |
| **CRD** | Custom Resource Definition — extends K8s API with new resource types (e.g., `Cluster` for PostgreSQL) |
| **Helm** | K8s package manager — bundles YAML templates into installable charts |
| **Terraform** | Infrastructure as Code — declares desired state of cloud resources |
| **ArgoCD** | GitOps tool — watches Git repo, auto-syncs K8s resources to match |
| **Temporal** | Workflow engine — runs multi-step processes with automatic retry and failure recovery |
| **CNPG** | CloudNativePG — PostgreSQL operator for K8s (backup, HA, scaling) |
| **APISIX** | API Gateway — routes requests, verifies JWT, applies rate limiting |
| **Keycloak** | Identity provider — manages users, realms, SSO, OAuth/OIDC |
| **Cilium** | CNI (network plugin) — provides network policies, encryption, observability |
| **Kyverno** | Policy engine — validates/mutates K8s resources before admission |
| **Falco** | Runtime security — detects suspicious syscalls in containers |
| **Velero** | Backup tool — snapshots K8s resources + PVCs to S3 |
| **KEDA** | Event-driven autoscaler — scales pods based on custom metrics |
| **NATS** | Lightweight message queue — used for internal platform events |
| **Hubble** | Network observability — visualizes traffic flows between pods |
| **Day-2** | Operations after initial deployment (upgrades, scaling, backup, monitoring, incident response) |
| **HA** | High Availability — system continues working when components fail |
| **DR** | Disaster Recovery — ability to restore service after a major failure |
| **WAF** | Web Application Firewall — protects against common web attacks (XSS, SQL injection) |
| **PITR** | Point-In-Time Recovery — restore database to any specific moment |
| **PDB** | PodDisruptionBudget — prevents K8s from evicting too many pods at once |
| **SLA** | Service Level Agreement — guaranteed uptime percentage |
| **MRR** | Monthly Recurring Revenue — total subscription income per month |

---

## Appendix D: File Structure Reference

```
Zenith/
├── apps/
│   ├── web/                    # Customer dashboard (Next.js)
│   ├── mission-control/        # Admin panel (Next.js)
│   └── landing/                # Marketing site (Next.js)
├── services/
│   ├── api/                    # Go backend (Fiber, hexagonal architecture)
│   │   ├── cmd/server/main.go  # Entry point + wiring
│   │   └── internal/
│   │       ├── entities/       # Domain models (28 types)
│   │       ├── services/       # Business logic
│   │       ├── handlers/       # HTTP handlers (55+ files)
│   │       ├── ports/          # Interfaces (25 repos + 8 infra)
│   │       ├── adapters/       # Implementations
│   │       │   ├── postgres/   # PostgreSQL repositories
│   │       │   ├── memory/     # In-memory fallback (dev/test)
│   │       │   ├── k8sclient/  # Kubernetes client
│   │       │   ├── promclient/ # Prometheus query client
│   │       │   ├── lokiclient/ # Loki log query client
│   │       │   └── ...         # Other adapters
│   │       ├── dto/            # Request/response schemas
│   │       ├── middleware/     # Auth, ownership, rate limiting
│   │       └── config/         # Environment-based config
│   └── operator/               # K8s operator (controller-runtime)
│       ├── api/v1alpha1/       # CRD type definitions (8 types)
│       └── internal/controllers/ # Reconciliation logic
├── cli/                        # Zenith CLI (Go + Cobra)
├── packages/
│   └── ui/                     # Shared UI components (Tailwind)
├── infra/
│   ├── terraform/
│   │   ├── staging/            # Phase 1: Hetzner VM + DNS
│   │   ├── staging-k8s/        # Phase 3: Helm releases
│   │   ├── production/         # Phase 1: Production VM + DNS
│   │   ├── production-k8s/     # Phase 3: Production Helm
│   │   └── modules/
│   │       └── k8s-platform/   # 14 focused .tf files
│   │           ├── certmanager.tf
│   │           ├── sealed_secrets.tf
│   │           ├── storage.tf      # CNPG + Hetzner CSI
│   │           ├── identity.tf     # Keycloak
│   │           ├── gateway.tf      # APISIX + external-dns
│   │           ├── gitops.tf       # ArgoCD
│   │           ├── registry.tf     # Harbor
│   │           ├── temporal.tf
│   │           ├── security.tf     # Kyverno + Falco + Velero
│   │           ├── observability.tf # Prometheus + Loki + Tempo + OTel
│   │           ├── autoscaling.tf  # KEDA
│   │           ├── traefik.tf
│   │           ├── apps.tf         # Platform apps
│   │           └── variables.tf    # 40+ input variables
│   ├── helm/
│   │   ├── zenith-platform/    # Shared infrastructure chart
│   │   ├── zenith-api/         # API server chart
│   │   ├── zenith-landing/     # Landing page chart
│   │   ├── zenith-tenant/      # Per-customer chart
│   │   └── monitoring/         # Observability chart
│   ├── ansible/                # Server bootstrap (k3s + Cilium)
│   ├── argocd/                 # ArgoCD Application manifests
│   └── smoke/                  # E2E smoke test container
├── docs/
│   ├── v3-architecture.md      # THIS DOCUMENT (single source of truth)
│   ├── v2-architecture/        # Legacy detailed design (35 files)
│   └── runbooks/               # Operational procedures
├── .github/
│   └── workflows/
│       ├── build-images.yml    # Test + build + push Docker images
│       ├── build-chart.yml     # Lint + push Helm charts
│       ├── terraform.yml       # Plan + apply infrastructure
│       └── dr-smoke-test.yml   # Disaster recovery testing
├── AGENTS.md                   # AI agent rules (Lich Framework)
├── CLAUDE.md                   # Claude CLI auto-detect (→ AGENTS.md)
└── agentlog.md                 # Change history
```

---

## Appendix E: Reading Order for New Team Members

```
Day 1: Understanding the Product
  1. Read Part A of this document (Business & Product)
  2. Sign up on staging (app.stage.freezenith.com)
  3. Deploy a test app, create a database, view logs

Day 2: Understanding the Architecture
  1. Read Part B of this document (Architecture & Design)
  2. Read AGENTS.md (development rules)
  3. SSH to staging server, explore with kubectl

Day 3: Understanding the Code
  1. Read services/api/internal/entities/ (domain models)
  2. Read services/api/internal/ports/ (interfaces)
  3. Read services/api/internal/handlers/ (pick 3 handlers)
  4. Read apps/web/src/lib/api.ts (frontend API client)

Day 4: Understanding the Infrastructure
  1. Read infra/terraform/modules/k8s-platform/*.tf
  2. Read infra/helm/zenith-api/values.yaml
  3. Run: kubectl get pods -A (on staging)
  4. Open Grafana, explore dashboards

Day 5: First Task
  1. Pick a task from Part C (Phase 0 is a good start)
  2. Follow the Lich Framework rules
  3. Create PR, get review
  4. Update agentlog.md
```

---

*This document is the single source of truth for the Zenith platform. If something contradicts this document, this document wins. Update this document when decisions change.*

*Last updated: 2026-03-08 by Babak + Claude (Architecture Session)*
