# Zenith V2 — Platform Architecture Overview

> **Status:** Design Complete, Implementation Pending
> **Last Updated:** 2026-02-25
> **Author:** Babak + Claude (Platform Architecture Session)

---

## Table of Contents

1. [What is Zenith V2](#what-is-zenith-v2)
2. [4-Tier Customer Model](#4-tier-customer-model)
3. [System Architecture Diagram](#system-architecture-diagram)
4. [Component Stack (6 Layers)](#component-stack-6-layers)
5. [Deployment Pipeline (4 Phases)](#deployment-pipeline-4-phases)
6. [Per-Customer Namespace Design](#per-customer-namespace-design)
7. [Routing Architecture](#routing-architecture)
8. [Data Architecture](#data-architecture)
9. [Provisioning Flow](#provisioning-flow)
10. [What Changed from V1](#what-changed-from-v1)
11. [Document Index](#document-index)

---

## What is Zenith V2

Zenith is a **Kubernetes-native Platform-as-a-Service (PaaS)** built on Hetzner Cloud. It provides multi-tenant infrastructure where customers deploy their applications with full isolation, automatic DNS, TLS, databases, and object storage.

**V1** (current) is a monolithic Helm chart with basic namespace isolation, Kong for API gateway, and manual provisioning.

**V2** (this design) is a complete redesign with:
- **APISIX** replacing Kong as the API gateway (etcd-backed, richer plugin ecosystem)
- **Keycloak** for identity management (realm per customer)
- **Temporal** for automated provisioning workflows
- **ArgoCD** for GitOps application deployment
- **Cilium** with WireGuard encryption and L7 network policies
- **Full observability** (Prometheus + Grafana + Loki + Tempo + Hubble)
- **Defense-in-depth security** (6 layers from Cloudflare to Falco)
- **Tested backup/restore** with CNPG WAL archiving + Velero

---

## 4-Tier Customer Model

```
+---------------------------------------------------------------------+
|                    SHARED SERVER FARM (Hetzner)                       |
|                    Same kernel, namespace isolation                    |
|                                                                       |
|  Shared services:                                                     |
|    Traefik, APISIX, Keycloak, CNPG operator, cert-manager,           |
|    external-dns, Cilium, Harbor, ArgoCD, Temporal, Monitoring         |
|                                                                       |
|  +-------------+ +-------------+ +-------------+                     |
|  | FREE user A | | FREE user B | | PRO user C  |  ...                |
|  | namespace   | | namespace   | | namespace   |                     |
|  | own DB(s)   | | own DB(s)   | | own DB(s)   |                     |
|  | own S3      | | own S3      | | own S3      |                     |
|  | own DNS     | | own DNS     | | own DNS +   |                     |
|  | (auto)      | | (auto)      | |  custom dom |                     |
|  +-------------+ +-------------+ +-------------+                     |
|                                                                       |
|  Elastic: scale nodes horizontally as customer count grows            |
+---------------------------------------------------------------------+

+---------------------+  +---------------------+
|  TEAM customer      |  |  ENTERPRISE customer|
|  Dedicated VMs      |  |  Dedicated VMs      |
|  via CAPI + CAPH    |  |  via CAPI + CAPH    |
|  Own k8s cluster    |  |  Own k8s cluster    |
|  Own everything     |  |  Own everything     |
|  Kernel isolation   |  |  Kernel isolation   |
+---------------------+  +---------------------+
```

| Tier | Infra | Isolation | Database | S3 Storage | Container Registry | API Gateway | Identity |
|------|-------|-----------|----------|------------|-------------------|-------------|----------|
| **Free** | Shared cluster | Namespace + Cilium | Shared CNPG Cluster, own DB | Shared Hetzner S3, own bucket | None | Shared APISIX | Shared Keycloak, own realm |
| **Pro** | Shared cluster | Namespace + Cilium | Sharded CNPG (~20 users/cluster), up to 5 DBs | Shared Hetzner S3, own bucket | Own project in Customer Harbor (`hub.stage.freezenith.com`) with storage quota | Shared APISIX | Shared Keycloak, own realm |
| **Team** | Dedicated VMs (CAPI) | Kernel-level | Own CNPG Cluster | Own S3 | Own Harbor project | Own APISIX | Own Keycloak |
| **Enterprise** | Dedicated VMs (CAPI) | Kernel-level | Own CNPG Cluster | Own S3 | Own Harbor project | Own APISIX | Own Keycloak |

**Key insight:** Free and Pro share the same infrastructure at the kernel level. Team and Enterprise get completely separate machines provisioned via CAPI+CAPH. This means a security breach in a Free customer's container cannot affect a Team customer because they are on different physical/virtual machines.

---

## System Architecture Diagram

```
                              Internet
                                 |
                    +------------+------------+
                    |     Cloudflare Proxy     |   Layer 0: DDoS, WAF, CDN
                    |  (DNS-01 for certs)      |
                    +------------+------------+
                                 |
                    +------------+------------+
                    |       Traefik (k3s)      |   TLS termination
                    |    IngressRoute CRDs     |   L7 routing
                    +-----+-------------+-----+
                          |             |
              +-----------+             +------------+
              |                                      |
    ingressClass: traefik                  ingressClass: apisix
    (frontends, landing,                   (all backend APIs)
     static pages)                                   |
              |                                      v
              v                         +------------------------+
    +------------------+                | APISIX (single gateway)|
    | Direct to pods   |                | Route-level plugins:   |
    |                  |                |                        |
    | - landing page   |                | /v1/api/* → jwt-auth + |
    | - customer       |                |   cors + rate-limit    |
    |   frontends      |                |                        |
    | - demo UIs       |                | /v1/webhooks/* → cors  |
    +------------------+                |   + rate-limit (no JWT)|
                                        +----------+-------------+
                                                   |
                                                   v
                                             Backend pods
                                        (customer + public APIs)
                                        |
                    +-------------------+-------------------+
                    |                   |                   |
            +-------+------+   +-------+------+   +-------+------+
            | Keycloak     |   | Shared CNPG  |   | Hetzner S3   |
            | (realm/cust) |   | (DB/customer)|   | (bucket/cust)|
            +--------------+   +--------------+   +--------------+
```

---

## Component Stack (6 Layers)

### Layer 1: Networking
| Component | Purpose | Notes |
|-----------|---------|-------|
| **Traefik** | TLS termination, frontend routing | Built into k3s, IngressRoute CRDs |
| **APISIX + etcd** | API gateway, JWT verify, CORS, rate-limit | Replaces Kong; etcd-backed (not PG) |
| **Cilium + Hubble** | CNI, NetworkPolicy, WireGuard encryption, L7 filtering | Replaces Flannel; full network observability |
| **external-dns** | Auto DNS via Cloudflare | Watches Ingress resources, creates A/CNAME records |

### Layer 2: Identity & Security
| Component | Purpose | Notes |
|-----------|---------|-------|
| **Keycloak** | Identity provider | One realm per customer, dedicated CNPG Cluster |
| **cert-manager** | TLS automation | Let's Encrypt, DNS-01 challenge (Cloudflare API) |
| **Kyverno** | Policy engine | Block unsigned images, enforce labels, resource limits |
| **Falco** | Runtime security | Detect anomalous container behavior |
| **Sealed Secrets** | Encrypted secrets in Git | Safe for ArgoCD GitOps |
| **etcd encryption** | Secrets at rest | k3s `--secrets-encryption` flag |
| **Pod Security Standards** | Container hardening | `restricted` on customer namespaces |

### Layer 3: Data
| Component | Purpose | Notes |
|-----------|---------|-------|
| **CNPG Operator** | Manages all PostgreSQL clusters | One operator, multiple clusters |
| **Keycloak PG** | Dedicated CNPG Cluster | Own Hetzner Volume, isolated from app DBs |
| **Free users PG** | Shared CNPG Cluster | All free customer DBs in one cluster |
| **Pro users PG (x N)** | Sharded CNPG Clusters | ~20 customers per cluster |
| **Hetzner S3** | Object storage | One account, bucket per customer |

### Layer 4: Platform (Zenith's own services)
| Component | Purpose | Notes |
|-----------|---------|-------|
| **zenith-api** | Go backend | Provisioning, admin API, customer API |
| **zenith-admin** | Admin panel UI | Tier management, infrastructure overview |
| **Temporal** | Provisioning workflows | Customer creation, CAPI cluster creation |
| **Internal Harbor** | Platform registry (`registry.stage.freezenith.com`) | Stores platform images + Helm charts. Managed outside the cluster (separate server). CI pushes here. |
| **Customer Harbor** | Pro-tier customer registry (`hub.stage.freezenith.com`) | Single in-cluster Harbor instance. One project per pro customer with storage quotas. Free users do NOT get a registry. Managed by Terraform (`registry.tf`). |
| **ArgoCD** | GitOps app deployment | App-of-Apps pattern |

### Layer 5: Observability
| Component | Purpose | Notes |
|-----------|---------|-------|
| **Prometheus** | Metrics collection | + Hubble metrics from Cilium |
| **Grafana** | Dashboards | Single pane: metrics, logs, traces, network |
| **Loki** | Log aggregation | + K8s audit logs |
| **Tempo** | Distributed traces | APISIX OTel plugin + Go SDK |
| **OpenTelemetry Collector** | Trace/metric pipeline | DaemonSet on all nodes |
| **Hubble** | Network flow observability | Service map, DNS, dropped packets |
| **Alertmanager** | Alerts | Slack/PagerDuty/Telegram |

### Layer 6: Resilience & Backup
| Component | Purpose | Notes |
|-----------|---------|-------|
| **Velero** | Cluster-level backup | All K8s resources + PV snapshots → S3 |
| **CNPG WAL archiving** | Continuous PG backup | Point-in-time recovery (PITR) → S3 |
| **pg_dump CronJobs** | Per-customer backup | Granular restore capability |
| **Keycloak realm export** | Identity backup | CronJob → S3 |
| **etcd snapshots** | APISIX config backup | CronJob → S3 |
| **Hetzner Volume snapshots** | Block-level backup | Weekly via API |
| **PriorityClasses** | Pod eviction order | system > infra > platform > customer |
| **PDBs** | HA protection | Prevent drain from killing all replicas |
| **ResourceQuota** | Per-namespace limits | Configurable per tier from admin panel |
| **LimitRange** | Per-pod limits | Prevent single pod resource hog |

---

## Deployment Pipeline (4 Phases)

```
Phase 1                Phase 2              Phase 3                Phase 4
Terraform              Ansible              Terraform              ArgoCD
(Hetzner+CF)           (OS+k3s)             (Cluster Bootstrap)    (Apps — automatic)

+----------+     +----------+     +-------------------+     +-----------------+
| Hetzner  |     | k3s      |     | cert-manager      |     | zenith-api      |
| VM       | --> | Cilium   | --> | CNPG operator     | --> | zenith-landing  |
| Firewall |     | hcloud   |     | APISIX + etcd     |     | zenith-tenant   |
| SSH keys |     | CSI      |     | Keycloak          |     | zenith-demo     |
|          |     |          |     | external-dns      |     |                 |
| Cloudflare     |          |     | Temporal          |     | (from here,     |
| DNS      |     |          |     | Harbor            |     |  every git push |
| records  |     |          |     | Monitoring        |     |  auto-deploys)  |
+----------+     +----------+     | ArgoCD            |     +-----------------+
                                  | Kyverno           |
                                  | Falco             |
                                  | Sealed Secrets    |
                                  | Velero            |
                                  +-------------------+
```

**How to run each phase:**

```bash
# Phase 1: Create Hetzner server + Cloudflare DNS
cd infra/terraform/staging
terraform init && terraform apply

# Phase 2: Install k3s + OS config
cd infra/ansible
ansible-playbook playbooks/site.yml -i inventory/staging.yml

# Phase 3: Bootstrap cluster with all infrastructure
cd infra/terraform/staging-k8s
terraform init && terraform apply

# Phase 4: Automatic! ArgoCD is running and watching the Git repo.
# Just push code to main → ArgoCD syncs → apps deployed.
```

**For production or reinstalling staging:** Same 4 phases with different values files.

See detailed docs:
- [Phase 1: Hetzner + Cloudflare](./01-phase1-hetzner-cloudflare.md)
- [Phase 2: Ansible + k3s](./02-phase2-ansible-k3s.md)
- [Phase 3: Cluster Bootstrap](./03-phase3-cluster-bootstrap.md)
- [Phase 4: ArgoCD Applications](./04-phase4-argocd-apps.md)

---

## Per-Customer Namespace Design

When a customer signs up (Free or Pro), Temporal provisions:

```
namespace: zenith-<customer>
  |
  |-- Deployments:
  |     |-- customer-frontend (1 replica)
  |     |     Ingress: ingressClass: traefik (direct, no APISIX)
  |     |     URL: <customer>.freezenith.com
  |     |
  |     |-- customer-backend-1 (1 replica)
  |     |     Ingress: ingressClass: apisix (JWT-protected)
  |     |     URL: api.<customer>.freezenith.com/v1/*
  |     |
  |     |-- customer-backend-2 (1 replica)
  |           Ingress: ingressClass: apisix
  |           Routes: /v1/* (JWT-protected), /v1/webhooks/* (public, no JWT)
  |
  |-- Database:
  |     Credentials in Secret (created by zenith-api via SQL)
  |     Points to shared CNPG Cluster (Free) or sharded cluster (Pro)
  |
  |-- S3:
  |     Credentials in Secret (created by zenith-api via Hetzner API)
  |     Bucket: zenith-<customer>-data
  |
  |-- Keycloak:
  |     Client credentials in Secret
  |     Realm: <customer> (created via Keycloak Admin API)
  |
  |-- APISIX CRDs:
  |     |-- ApisixRoute (protected backend routes)
  |     |-- ApisixRoute (public backend routes)
  |     |-- ApisixPluginConfig (CORS: origins = customer domain)
  |     |-- ApisixPluginConfig (rate-limit: per tier)
  |
  |-- Security:
  |     |-- CiliumNetworkPolicy (default deny + explicit allows)
  |     |-- ResourceQuota (CPU/memory/pods limits per tier)
  |     |-- LimitRange (per-pod limits)
  |     |-- PodSecurityStandard: restricted
  |
  |-- DNS:
        external-dns annotation on Ingress
        -> creates <customer>.freezenith.com in Cloudflare
        Pro+: can add custom domain (CNAME to proxy.freezenith.com)
```

---

## Routing Architecture

### Frontend Routes (Traefik — Direct)

```
Traefik receives request
  |
  |-- Host: freezenith.com          -> zenith-landing:3000
  |-- Host: admin.freezenith.com    -> zenith-admin:3000
  |-- Host: <customer>.freezenith.com -> customer-frontend:3000
  |-- Host: custom-domain.com       -> customer-frontend:3000 (Pro+)
```

No APISIX involved. Traefik terminates TLS, routes directly to the frontend pod. Frontend is a Next.js app serving HTML/JS — no need for JWT verification.

### Backend Routes (APISIX — Protected + Public)

```
Traefik receives request for api.<customer>.freezenith.com
  |
  v
APISIX (ingressClass: apisix)
  |
  |-- Route: /v1/*  (protected)
  |     Plugins: jwt-auth (Keycloak JWKS), cors, rate-limit
  |     -> customer-backend:8080
  |
  |-- Route: /v1/webhooks/* (public — no JWT)
  |     Plugins: cors, rate-limit
  |     -> customer-backend:8080
  |
  |-- Route: /v1/unsubscribe/* (public — no JWT)
        Plugins: cors, rate-limit
        -> customer-backend:8080
```

### How APISIX JWT Verification Works

```
1. Customer frontend sends request with Authorization: Bearer <token>
2. Traefik routes to APISIX
3. APISIX jwt-auth plugin:
   a. Extracts token from Authorization header
   b. Fetches Keycloak JWKS from: https://auth.freezenith.com/realms/<customer>/protocol/openid-connect/certs
   c. Verifies token signature, expiration, audience
   d. If valid: forwards request to backend with X-Consumer-* headers
   e. If invalid: returns 401 Unauthorized
4. Backend receives verified request (trusts APISIX headers)
```

---

## Data Architecture

### PostgreSQL Sharding Strategy

```
CNPG Operator (cnpg-system namespace)
  |
  |-- Watches ALL namespaces for Cluster CRs
  |
  |-- Keycloak PG Cluster (namespace: keycloak)
  |     |-- keycloak-pg-1 (primary)   [Hetzner Volume: 10Gi]
  |     |-- keycloak-pg-2 (replica)   [Hetzner Volume: 10Gi]
  |     |-- Database: keycloak
  |     WAL archiving -> Hetzner S3: zenith-backups/keycloak-wal/
  |
  |-- Free Users PG Cluster (namespace: zenith-shared)
  |     |-- free-pg-1 (primary)       [Hetzner Volume: 50Gi]
  |     |-- free-pg-2 (replica)       [Hetzner Volume: 50Gi]
  |     |-- Databases:
  |     |     zenith_platform (Zenith's own DB)
  |     |     customer_abc
  |     |     customer_def
  |     |     customer_ghi
  |     |     ... (all free users)
  |     WAL archiving -> Hetzner S3: zenith-backups/free-pg-wal/
  |     pg_dump CronJob -> per-customer dumps to S3
  |
  |-- Pro Shard 1 PG Cluster (namespace: zenith-shared)
  |     |-- pro-shard1-pg-1 (primary)  [Hetzner Volume: 100Gi]
  |     |-- pro-shard1-pg-2 (replica)  [Hetzner Volume: 100Gi]
  |     |-- Databases:
  |     |     customer_pro_001 (up to 5 DBs)
  |     |     customer_pro_002
  |     |     ... (max ~20 pro customers per shard)
  |     WAL archiving -> Hetzner S3: zenith-backups/pro-shard1-wal/
  |
  |-- Pro Shard 2 PG Cluster (created when shard 1 is full)
        ...
```

**Database creation flow:**
1. Customer signs up (Free)
2. Temporal workflow calls zenith-api
3. zenith-api connects to free-pg primary
4. Runs: `CREATE DATABASE customer_xxx; CREATE USER customer_xxx ...`
5. Stores credentials in K8s Secret in customer namespace
6. Customer's backend reads Secret, connects to their DB

**Pro user shard assignment:**
1. zenith-api checks which pro shards have capacity (< 20 customers)
2. Assigns customer to first available shard
3. If no shard has capacity, alerts admin (or auto-creates new shard)

### S3 Strategy

```
Hetzner Object Storage Account (one per environment)
  |
  |-- zenith-backups/           (platform backups)
  |     |-- keycloak-wal/
  |     |-- free-pg-wal/
  |     |-- pro-shard1-wal/
  |     |-- velero/
  |     |-- keycloak-realm-exports/
  |
  |-- zenith-harbor/            (Customer Harbor S3 backend — hub.stage.freezenith.com)
  |
  |-- customer-abc-data/        (Free customer bucket)
  |-- customer-def-data/        (Free customer bucket)
  |-- customer-pro-001-data/    (Pro customer bucket)
  |-- customer-pro-001-assets/  (Pro customer, 2nd bucket)
```

---

## Provisioning Flow

### Free/Pro Customer Signup

```
User clicks "Sign Up" on freezenith.com
         |
         v
zenith-api receives POST /api/v1/auth/register
         |
         v
zenith-api starts Temporal workflow: "provision-customer"
         |
         +----> Activity 1: CreateKeycloakRealm()
         |        Keycloak Admin API: create realm, client, roles
         |        Store client_id + client_secret
         |
         +----> Activity 2: CreateDatabase()
         |        Connect to assigned CNPG Cluster (free or pro shard)
         |        SQL: CREATE DATABASE, CREATE USER, GRANT ALL
         |        Store credentials
         |
         +----> Activity 3: CreateS3Bucket()
         |        Hetzner API: create bucket + access key
         |        Store access_key + secret_key
         |
         +----> Activity 4: CreateK8sNamespace()
         |        K8s API: create namespace with labels
         |        Apply: ResourceQuota, LimitRange, PodSecurityStandard
         |        Apply: CiliumNetworkPolicy (default deny + allows)
         |
         +----> Activity 5: CreateK8sSecrets()
         |        K8s API: create Secrets (DB creds, S3 creds, KC creds)
         |
         +----> Activity 6: CreateK8sDeployments()
         |        K8s API: create Deployments, Services
         |        (frontend + backends from customer's selected template)
         |
         +----> Activity 7: CreateIngressRoutes()
         |        K8s API: create Traefik IngressRoute (frontend)
         |        K8s API: create APISIX Route CRDs (backends)
         |        K8s API: create ApisixPluginConfig (CORS, rate-limit)
         |
         +----> Activity 8: WaitForDNS()
         |        external-dns creates Cloudflare record
         |        Poll until DNS resolves
         |
         +----> Activity 9: WaitForTLSCert()
         |        cert-manager issues certificate
         |        Poll until Secret exists
         |
         +----> Activity 10: NotifyCustomerReady()
                  Update customer status in DB
                  Send welcome email
                  Customer sees "Your environment is ready!" in dashboard
```

### Team/Enterprise Customer Provisioning

```
Admin creates customer in admin panel
         |
         v
zenith-api starts Temporal workflow: "provision-dedicated-cluster"
         |
         +----> Activity 1: CreateHetznerVMs()
         |        CAPI + CAPH: create Machine + MachineDeployment
         |        Wait for VMs to be provisioned (5-10 min)
         |
         +----> Activity 2: WaitForClusterReady()
         |        CAPI: wait for Cluster status = Provisioned
         |        Wait for kubeconfig to be available
         |
         +----> Activity 3: InstallBaseComponents()
         |        Helm: cert-manager, CNPG operator, Cilium
         |
         +----> Activity 4: InstallPlatformStack()
         |        Helm: APISIX, Keycloak, monitoring
         |
         +----> Activity 5: CreateCustomerResources()
         |        Same as Free/Pro Activities 1-9 but on dedicated cluster
         |
         +----> Activity 6: ConfigureDNS()
         |        Cloudflare: create records for dedicated cluster IP
         |
         +----> Activity 7: NotifyCustomerReady()
                  Total time: 10-20 minutes
```

---

## What Changed from V1

| Aspect | V1 (Current) | V2 (New Design) |
|--------|-------------|-----------------|
| **API Gateway** | Kong OSS (DB-less) | APISIX (etcd-backed) |
| **Identity** | JWT in zenith-api | Keycloak (realm per customer) |
| **Helm Charts** | One monolithic chart | 5 modular charts + ArgoCD |
| **GitOps** | None (Terraform only) | ArgoCD (App-of-Apps) |
| **Provisioning** | Manual | Temporal workflows |
| **Database** | One shared PG, manual setup | Sharded CNPG, auto-provisioned |
| **S3** | None | Hetzner S3, bucket per customer |
| **DNS** | Manual Cloudflare | external-dns (automatic) |
| **Network Security** | None (Flannel) | Cilium + WireGuard + L7 policy |
| **Backup** | None | CNPG WAL + pg_dump + Velero |
| **Image Security** | Harbor Trivy | + cosign signing + Kyverno admission |
| **Runtime Security** | None | Falco anomaly detection |
| **Tracing** | None | OpenTelemetry + Tempo |
| **Secrets** | Plain K8s Secrets | etcd encryption + Sealed Secrets |
| **Container Registry** | Harbor (manual push) | Two Harbors: Internal (platform images, separate server) + Customer (in-cluster, Pro-tier only) + ArgoCD Image Updater |
| **cert-manager** | HTTP-01 (proxy OFF) | DNS-01 (proxy ON, Cloudflare) |

---

## Document Index

### Deployment Phase Docs

| Document | Description |
|----------|-------------|
| [00-overview.md](./00-overview.md) | This document — full architecture overview |
| [01-phase1-hetzner-cloudflare.md](./01-phase1-hetzner-cloudflare.md) | Phase 1: Hetzner VM + Cloudflare DNS |
| [02-phase2-ansible-k3s.md](./02-phase2-ansible-k3s.md) | Phase 2: Ansible + k3s + Cilium |
| [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) | Phase 3: All infra components via Terraform |
| [04-phase4-argocd-apps.md](./04-phase4-argocd-apps.md) | Phase 4: ArgoCD application deployment |

### Cross-Cutting Concern Docs

| Document | Description |
|----------|-------------|
| [05-user-flows.md](./05-user-flows.md) | Customer, admin, and developer user flows |
| [06-security-model.md](./06-security-model.md) | Defense-in-depth security architecture |
| [07-backup-disaster-recovery.md](./07-backup-disaster-recovery.md) | Backup strategy and restore procedures |
| [08-observability.md](./08-observability.md) | Monitoring, logging, tracing, alerting |
| [09-migration-v1-to-v2.md](./09-migration-v1-to-v2.md) | Migration plan from current to v2 |
| [10-backend-architecture.md](./10-backend-architecture.md) | Go backend Lich Architecture, Keycloak design, request flow diagrams |

### Per-Component Architecture Docs (NEW)

| Document | Component(s) | Description |
|----------|-------------|-------------|
| [SYSTEM-MAP.md](./SYSTEM-MAP.md) | ALL | Master system diagram, communication table, 5 request flows, namespace map, DNS map |
| [11-infrastructure-provisioning.md](./11-infrastructure-provisioning.md) | Terraform + Ansible | 3-layer provisioning model, full environment setup flow |
| [12-traefik-ingress.md](./12-traefik-ingress.md) | Traefik | TLS termination, IngressRoute CRDs, cross-namespace routing |
| [13-apisix-gateway.md](./13-apisix-gateway.md) | APISIX + etcd | JWT auth, CORS, rate-limiting, plugin pipeline, route configuration |
| [14-cilium-networking.md](./14-cilium-networking.md) | Cilium + Hubble | eBPF networking, WireGuard encryption, network policies, flow observability |
| [15-argocd-gitops.md](./15-argocd-gitops.md) | ArgoCD | App-of-Apps pattern, image updater, sync waves, deployment flow |
| [16-data-storage.md](./16-data-storage.md) | CNPG + Hetzner S3 + KEDA | PostgreSQL sharding, S3 buckets, scale-to-zero, customer data lifecycle |
| [17-temporal-workflows.md](./17-temporal-workflows.md) | Temporal | Provisioning workflows, activity execution, retry/compensation |
| [18-kyverno-policies.md](./18-kyverno-policies.md) | Kyverno + Falco | Admission policies, runtime security, detection rules |
| [19-velero-backup.md](./19-velero-backup.md) | Velero | Cluster backup/restore, schedule, S3 storage |
| [20-sealed-secrets.md](./20-sealed-secrets.md) | Sealed Secrets | Encrypted secrets for GitOps, key management |

### Developer Experience Docs (NEW)

| Document | Description |
|----------|-------------|
| [21-local-development-setup.md](./21-local-development-setup.md) | Prerequisites, Docker Compose, local processes, staging access, IDE setup |
| [22-day-to-day-operations.md](./22-day-to-day-operations.md) | Add API endpoints, add pages, migrations, deploy, logs, debug, Git workflow |
| [23-frontend-architecture.md](./23-frontend-architecture.md) | 3 Next.js apps: landing, web dashboard, mission control — routing, API client, auth, styling |
| [24-ci-cd-pipeline.md](./24-ci-cd-pipeline.md) | GitHub Actions workflows, image tagging, security scanning, ArgoCD handoff |
| [25-monitoring-runbook.md](./25-monitoring-runbook.md) | Alert response procedures, Grafana dashboards, Loki queries, Tempo traces |
| [26-keycloak-administration.md](./26-keycloak-administration.md) | Realm-per-tenant model, admin console guide, JWT flow, API integration |

### Implementation Plans

| Document | Description |
|----------|-------------|
| [BACKEND-REFACTOR.md](./BACKEND-REFACTOR.md) | Backend refactoring implementation plan (step-by-step with checkboxes) |
| [HANDOVER.md](./HANDOVER.md) | AI handover document for multi-account workflow |
