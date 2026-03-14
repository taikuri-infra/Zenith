# 02 — Architecture Deep Dive

> **Read time:** 60 minutes
> **Prerequisite:** [01 — What is Zenith](./01-what-is-zenith.md)
> **Next:** [03 — Backend Complete Guide](./03-backend-guide.md)

---

## System Architecture (30,000 Feet)

```
                            ┌─────────────────────┐
                            │      INTERNET        │
                            └──────────┬──────────┘
                                       │
                            ┌──────────▼──────────┐
                            │   Cloudflare (CDN)   │ ← Layer 0: DDoS, WAF, DNS proxy
                            │   DNS-01 challenges  │ ← TLS cert validation
                            └──────────┬──────────┘
                                       │
                  ┌────────────────────▼────────────────────┐
                  │         TRAEFIK (built into k3s)         │ ← Layer 1: TLS termination
                  │    IngressRoute CRDs, L7 routing         │    NOT standard Ingress!
                  ├────────────┬───────────────────────────┤
                  │            │                            │
            ┌─────▼─────┐  ┌──▼──────────────┐  ┌────────▼────────┐
            │ Frontend   │  │  APISIX Gateway  │  │  Platform UIs    │
            │ Routes     │  │  JWT verify      │  │  (Admin, Keycloak│
            │ (Direct)   │  │  Rate limiting   │  │   Harbor)        │
            └─────┬──────┘  │  CORS            │  └─────────────────┘
                  │         │  WAF (uri-blocker)│
            ┌─────▼──────┐  └──────┬───────────┘
            │ Customer   │         │
            │ Apps       │  ┌──────▼──────────────────────────────┐
            │ (deployed) │  │           ZENITH API (Go/Fiber)      │
            └────────────┘  │  ├── 376 endpoints                   │
                            │  ├── 37 entity types                 │
                            │  ├── 46 interface contracts           │
                            │  ├── 24 service files                │
                            │  └── 16 adapter packages             │
                            └──────┬──────────────────────────────┘
                                   │
             ┌─────────────────────┼──────────────────────┐
             │                     │                      │
      ┌──────▼──────┐    ┌───────▼────────┐    ┌───────▼───────┐
      │  DATA LAYER │    │ IDENTITY LAYER  │    │  INFRA LAYER  │
      │ PostgreSQL  │    │ Keycloak        │    │ K8s API       │
      │ (CNPG)      │    │ (Operator)      │    │ Harbor        │
      │ Redis (Op)  │    │ Realms/OIDC     │    │ Kaniko (build)│
      │ MongoDB (Op)│    │ OAuth2          │    │ cert-manager  │
      │ Hetzner S3  │    │                 │    │ external-dns  │
      └─────────────┘    └─────────────────┘    └───────────────┘
```

---

## The 8-Layer Stack

Every component uses a **Kubernetes Operator** where one exists. This is non-negotiable.

### Layer 0: Edge (Cloudflare)
- DDoS protection, CDN caching
- DNS proxy (orange cloud)
- Web Application Firewall (WAF)
- DNS-01 challenges for TLS certificates

### Layer 1: Networking
| Component | What It Does | Why We Chose It |
|-----------|-------------|-----------------|
| **Traefik** (k3s built-in) | TLS termination, L7 routing | Built into k3s, IngressRoute CRDs |
| **APISIX** + etcd | API gateway: JWT, CORS, rate-limit | Richer plugin ecosystem than Kong, etcd-backed |
| **Cilium** + Hubble | CNI, NetworkPolicy, WireGuard | eBPF-based, L7 filtering, network observability |
| **external-dns** | Auto DNS via Cloudflare | Watches IngressRoute CRDs → creates DNS records |

**Key Concept — Traefik vs APISIX:**
```
Traefik handles:                    APISIX handles:
• TLS termination                   • JWT verification
• Frontend routing (direct)         • Rate limiting
• Platform UI routing               • CORS
                                    • WAF (uri-blocker)
                                    • All /api/* routes
```

Traefik routes `api.stage.freezenith.com` to APISIX via ExternalName Service. APISIX then routes to zenith-api pods.

### Layer 2: Identity & Security
| Component | What It Does |
|-----------|-------------|
| **Keycloak** (Operator) | Identity provider, realm per customer, OAuth/OIDC/SAML |
| **cert-manager** (Operator) | TLS automation, Let's Encrypt, DNS-01 |
| **Kyverno** (Operator) | Admission policies: deny unscanned images, enforce labels |
| **Falco** | Runtime security: detect shell exec, anomalous network |
| **Sealed Secrets** | Encrypted secrets in Git (safe for ArgoCD) |

### Layer 3: Data
| Component | What It Does |
|-----------|-------------|
| **CNPG** (Operator) | PostgreSQL: auto-failover, WAL→S3, PITR |
| **Redis** (Operator) | In-memory cache, customer-provisioned |
| **MongoDB** (Percona Operator) | Document database, customer-provisioned |
| **RabbitMQ** (Operator) | Message queue, customer-provisioned |
| **Kafka** (Strimzi Operator) | Event streaming, Business+ only |
| **Hetzner S3** | Object storage, bucket per customer |

### Layer 4: Platform (Our Services)
| Component | What It Does |
|-----------|-------------|
| **zenith-api** | Go backend (1 binary, 376 endpoints, Fiber HTTP) |
| **zenith-web** | Customer dashboard (Next.js 15, 40 pages) |
| **zenith-mc** | Admin panel (Next.js 15, 39 pages) |
| **zenith-landing** | Marketing site (Next.js 15, 3 pages) |
| **zenith-operator** | K8s operator for Zenith CRDs |
| **Temporal** | Workflow engine for provisioning |
| **Harbor** (2 instances!) | Internal registry + Customer registry |
| **ArgoCD** | GitOps engine (watches `staging` branch) |
| **NATS** | Internal event bus (deploy events, billing, notifications) |

### Layer 5: Observability
| Component | What It Does |
|-----------|-------------|
| **Prometheus** (Operator) | Metrics collection |
| **Loki** (Operator) | Log aggregation |
| **Tempo** (Operator) | Distributed traces |
| **OTel Collector** (Operator) | Telemetry pipeline |
| **Grafana** | Dashboards (metrics + logs + traces) |
| **Hubble** | Network flow observability |
| **Alertmanager** | Alert routing (Slack, PagerDuty, Telegram) |

### Layer 6: Resilience & Backup
| Component | What It Does |
|-----------|-------------|
| **Velero** | Cluster-level backup → S3 |
| **CNPG WAL** | Continuous PostgreSQL backup → S3 (PITR) |
| **pg_dump CronJobs** | Per-customer database dumps |

### Layer 7: Auto-Scaling
| Component | What It Does |
|-----------|-------------|
| **KEDA** | Pod scaling (HPA + custom metrics) |
| **KEDA HTTP** | Scale-to-zero for free tier (HTTPScaledObject) |
| **Hetzner Autoscaler** | Node scaling (add/remove VMs) |

---

## Namespace Strategy

```
K8s CLUSTER (k3s)
│
├── SYSTEM NAMESPACES (k3s managed)
│   ├── kube-system (Traefik, CoreDNS)
│   ├── kube-public
│   └── kube-node-lease
│
├── PLATFORM NAMESPACES (Terraform managed)
│   ├── zenith-staging (API, Web, MC, Landing)
│   ├── zenith-shared (Shared CNPG clusters, cold-start page)
│   ├── zenith-apps (Customer app deployments)
│   ├── zenith-builds (Kaniko build jobs)
│   ├── monitoring (Prometheus, Loki, Tempo, OTel, Grafana)
│   ├── apisix (APISIX + etcd + ingress controller)
│   ├── argocd (ArgoCD + Image Updater)
│   ├── keycloak (Keycloak + dedicated CNPG)
│   ├── temporal (Temporal workflow engine)
│   ├── harbor (Customer Harbor registry)
│   ├── cert-manager
│   ├── sealed-secrets
│   ├── kyverno
│   ├── falco
│   ├── cilium (CNI + Hubble)
│   └── external-dns
│
└── CUSTOMER NAMESPACES (Business tier — dedicated)
    └── zenith-customer-<id> (own apps, DB, storage, network policy)
```

---

## Request Flow Paths

### Path 1: Customer API Request (e.g., `GET /api/v1/apps`)

```
Browser → Cloudflare → Traefik (:443 TLS) → APISIX
                                               │
                                               ├── 1. CORS check
                                               ├── 2. uri-blocker (WAF)
                                               ├── 3. ua-restriction
                                               ├── 4. limit-count (rate limit)
                                               ├── 5. JWT verify (if protected route)
                                               │
                                               └──→ zenith-api (Go/Fiber)
                                                      │
                                                      ├── middleware.RequireAuth()
                                                      ├── middleware.RequireAppOwnership()
                                                      ├── handler.ListApps()
                                                      ├── service.ListApps() → repo.ListApps()
                                                      └── PostgreSQL query → JSON response
```

### Path 2: Customer App Request (e.g., `my-app.apps.stage.freezenith.com`)

```
Browser → Cloudflare → Traefik (:443 TLS)
                          │
                          ├── Matches IngressRoute for *.apps.stage.freezenith.com
                          ├── Wildcard TLS cert (apps-wildcard-tls)
                          └──→ Customer App Pod (port 3000/8080)
                                 └── Direct response (no APISIX involved)
```

### Path 3: App Deployment (git push → live app)

```
GitHub Webhook → zenith-api
  │
  ├── 1. Create Kaniko Job (zenith-builds namespace)
  │      ├── Clone git repo
  │      ├── Build Docker image
  │      └── Push to Harbor (registry.stage.freezenith.com)
  │
  ├── 2. Create/Update K8s Deployment (zenith-apps namespace)
  │      ├── Set image to new tag
  │      ├── Apply resource limits
  │      └── Wait for rollout
  │
  ├── 3. Create/Update IngressRoute + Service
  │      ├── Route: my-app.apps.stage.freezenith.com
  │      ├── TLS: wildcard cert
  │      └── external-dns → Cloudflare DNS record
  │
  └── 4. Notify user (WebSocket event + in-app notification)
```

---

## Security Architecture (9 Layers)

| Layer | What | How |
|-------|------|-----|
| 1. Edge | DDoS + CDN | Cloudflare proxy |
| 2. Network | Encryption + isolation | Cilium + WireGuard, NetworkPolicy |
| 3. Gateway | JWT + rate-limit + WAF | APISIX plugins (uri-blocker, ua-restriction) |
| 4. Application | MFA + sessions + API keys | TOTP, JWT with short expiry + refresh rotation |
| 5. Container | Non-root, read-only FS | SecurityContext, Pod Security Standards |
| 6. Image | Scan + verify | Harbor Trivy + Kyverno admission |
| 7. Runtime | Anomaly detection | Falco alerts (shell exec, network anomaly) |
| 8. Data | Encryption | etcd encryption, TLS in transit, AES-256-GCM secrets |
| 9. Audit | Who did what, when | K8s audit logs, app audit logs, Hubble flows |

---

## Two Harbors (CRITICAL — Do Not Confuse!)

```
1. INTERNAL HARBOR (registry.stage.freezenith.com)
   ├── Purpose: Platform images + Helm charts
   ├── Location: OUTSIDE the cluster (separate server)
   ├── Repo: /Users/babak/codes/DoTech/harbor-registry
   ├── CI pushes here (deploy-staging.yml, build-images.yml)
   └── NOT managed by Terraform in Zenith project

2. CUSTOMER HARBOR (hub.stage.freezenith.com)
   ├── Purpose: Pro-tier customer container registry
   ├── Location: INSIDE the cluster
   ├── Managed by: Terraform (registry.tf)
   ├── One project per pro customer with storage quotas
   └── Free users do NOT get a registry
```

---

## Two CNPG Clusters (CRITICAL — Do Not Confuse!)

```
1. zenith-postgres (namespace: zenith-staging)
   ├── Purpose: Zenith API's own database
   ├── Database: zenith
   ├── Tables: 48 (users, apps, gateways, billing, etc.)
   ├── Backup: S3 s3://zenith-backups/zenith-postgres-wal/
   └── PRIMARY is pod-1 (NOT pod-0 — always check metadata.labels.role)

2. free-pg (namespace: zenith-shared)
   ├── Purpose: Customer-provisioned databases
   ├── Database: zenith_platform (+ temporal, temporal_visibility)
   ├── Used by: API's database provisioning service
   ├── Backup: S3 s3://zenith-backups/free-pg-wal/
   └── Customers get: CREATE DATABASE customer_xxx; CREATE USER customer_xxx;
```

---

## Deployment Pipeline (4 Phases)

```
Phase 1 (Terraform)          Phase 2 (Ansible)         Phase 3 (Terraform)        Phase 4 (ArgoCD)
Hetzner + Cloudflare         OS + k3s                  Cluster Bootstrap          Apps (auto)

┌──────────────┐      ┌──────────────┐      ┌──────────────────┐      ┌───────────────┐
│ Hetzner VM   │      │ k3s install  │      │ cert-manager     │      │ zenith-api    │
│ Firewall     │ ──→  │ Cilium CNI   │ ──→  │ CNPG operator    │ ──→  │ zenith-web    │
│ SSH keys     │      │ hcloud CSI   │      │ APISIX + etcd    │      │ zenith-mc     │
│ Cloudflare   │      │              │      │ Keycloak         │      │ zenith-landing│
│ DNS records  │      │              │      │ Temporal         │      │               │
└──────────────┘      └──────────────┘      │ Harbor           │      │ (ArgoCD syncs │
                                            │ ArgoCD           │      │  from staging │
                                            │ Monitoring       │      │  branch)      │
                                            │ Kyverno, Falco   │      └───────────────┘
                                            │ Sealed Secrets   │
                                            │ Velero           │
                                            └──────────────────┘
```

```bash
# Phase 1: Create Hetzner server + Cloudflare DNS
cd infra/terraform/staging && terraform init && terraform apply

# Phase 2: Install k3s + OS config
cd infra/ansible && ansible-playbook playbooks/site.yml -i inventory/staging.yml

# Phase 3: Bootstrap cluster with all infrastructure
cd infra/terraform/staging-k8s && terraform init && terraform apply

# Phase 4: Automatic! ArgoCD watches staging branch
git push origin staging  # → ArgoCD syncs → apps deployed
```

---

## Key Design Principles

1. **Operator-First:** Every stateful service uses a K8s Operator. Operators provide self-healing, automated upgrades, backup management.

2. **Hexagonal Architecture:** Business logic never imports infrastructure. All external systems hidden behind interfaces in `ports/`.

3. **API-as-Proxy:** Customers never directly access Prometheus, Loki, or K8s API. All queries scoped to the customer's own resources.

4. **Event-Driven:** Internal operations flow through NATS JetStream. Ensures concurrent operations don't conflict.

5. **Rebuildable from Git:** Entire platform can be rebuilt from scratch using Terraform + ArgoCD + Sealed Secrets.

6. **Day-2 First:** Architectural decisions prioritize maintainability and automated operations over initial setup simplicity.

---

**Next → [03 — Backend Complete Guide](./03-backend-guide.md)**
