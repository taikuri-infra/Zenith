# Phase 3: Cluster Bootstrap (Terraform)

> **Zenith V2 Platform Architecture -- Phase 3 of 5**
>
> This is the largest phase. It takes a bare k3s cluster (from Phase 2) and installs
> every infrastructure component needed to run the Zenith platform.

---

**Status:** Design Complete, Implementation Pending
**Last Updated:** 2026-02-25
**Author:** Babak + Claude (Platform Architecture Session)
**Prerequisite:** Phase 1 (Hetzner + Cloudflare) and Phase 2 (Ansible + k3s) completed
**Estimated Time:** 15-25 minutes (first apply), 3-5 minutes (incremental changes)

---

## Table of Contents

1. [Why This Phase Exists](#why-this-phase-exists)
2. [What Gets Installed](#what-gets-installed)
3. [Architecture Overview](#architecture-overview)
4. [Namespace Layout](#namespace-layout)
5. [Installation Order (Dependency Graph)](#installation-order-dependency-graph)
6. [Component Deep Dives: Networking](#component-deep-dives-networking)
   - [APISIX + etcd](#apisix--etcd)
   - [external-dns](#external-dns)
7. [Component Deep Dives: Identity](#component-deep-dives-identity)
   - [Keycloak + Dedicated CNPG](#keycloak--dedicated-cnpg)
8. [Component Deep Dives: Data](#component-deep-dives-data)
   - [CNPG Operator](#cnpg-operator)
   - [CNPG Clusters (Keycloak, Free, Pro Shards)](#cnpg-clusters-keycloak-free-pro-shards)
9. [Component Deep Dives: Platform](#component-deep-dives-platform)
   - [ArgoCD (App-of-Apps)](#argocd-app-of-apps)
   - [Temporal](#temporal)
   - [Harbor](#harbor)
10. [Component Deep Dives: Security](#component-deep-dives-security)
    - [cert-manager](#cert-manager)
    - [Kyverno](#kyverno)
    - [Falco](#falco)
    - [Sealed Secrets](#sealed-secrets)
    - [Pod Security Standards](#pod-security-standards)
11. [Component Deep Dives: Observability](#component-deep-dives-observability)
    - [Prometheus + Grafana + Alertmanager](#prometheus--grafana--alertmanager)
    - [Loki (Logs)](#loki-logs)
    - [Tempo (Traces)](#tempo-traces)
    - [OpenTelemetry Collector](#opentelemetry-collector)
    - [Hubble (Network Flows)](#hubble-network-flows)
12. [Component Deep Dives: Resilience](#component-deep-dives-resilience)
    - [Velero](#velero)
    - [PriorityClasses](#priorityclasses)
    - [PodDisruptionBudgets](#poddisruptionbudgets)
    - [ResourceQuotas and LimitRanges](#resourcequotas-and-limitranges)
13. [Terraform Structure](#terraform-structure)
14. [Variables Reference](#variables-reference)
15. [How to Run](#how-to-run)
16. [Verification Checklist](#verification-checklist)
17. [Troubleshooting](#troubleshooting)
18. [Architecture Decision References](#architecture-decision-references)
19. [What Happens Next (Phase 4)](#what-happens-next-phase-4)

---

## Why This Phase Exists

After Phase 2, you have a bare k3s cluster. It has Cilium for CNI, the Hetzner CSI
driver for persistent volumes, Traefik for ingress, and CoreDNS for cluster DNS. It can
schedule pods and mount storage. But it has **zero platform capabilities**:

- No TLS certificate automation
- No database operator or database clusters
- No identity provider (no Keycloak, no JWT verification)
- No API gateway (no CORS, no rate limiting, no authentication at the edge)
- No automatic DNS record creation
- No GitOps engine (no ArgoCD)
- No provisioning workflow engine (no Temporal)
- No container registry (no Harbor)
- No policy enforcement (no Kyverno)
- No runtime security detection (no Falco)
- No encrypted secrets for Git (no Sealed Secrets)
- No cluster backup (no Velero)
- No observability (no Prometheus, no Grafana, no Loki, no tracing)

Phase 3 bridges the gap between "empty cluster" and "application-ready platform." It
installs **19 infrastructure concerns** across **14 Terraform resources** in strict
dependency order. Once Phase 3 completes, the cluster is fully operational and ArgoCD is
watching the Git repository, ready to deploy application workloads automatically.

```
Phase 2 output:                Phase 3 installs:                Phase 4 ready:

+------------------+     +-----------------------------+     +------------------+
| k3s cluster      |     | cert-manager + ClusterIssuer|     | ArgoCD watches   |
| Cilium CNI       |     | CNPG Operator + PG Clusters |     | Git repo and     |
| Hetzner CSI      | --> | Keycloak (identity)         | --> | deploys apps     |
| Traefik (ingress)|     | APISIX + etcd (API gateway) |     | automatically    |
| CoreDNS          |     | external-dns (auto DNS)     |     |                  |
| (bare, no apps)  |     | Temporal (provisioning)     |     | zenith-api       |
+------------------+     | Harbor (registry)           |     | zenith-landing   |
                          | Kyverno (policy)            |     | zenith-tenant    |
                          | Falco (runtime security)    |     | zenith-demo      |
                          | Sealed Secrets              |     +------------------+
                          | Velero (cluster backup)     |
                          | Prometheus + Grafana + Loki |
                          | Tempo + OTel Collector      |
                          | Hubble + Alertmanager       |
                          | ArgoCD (GitOps engine)      |
                          | PriorityClasses + PDBs      |
                          +-----------------------------+
```

**Design Principle: Infra = Terraform, Apps = ArgoCD (Decision D10).**

Everything in Phase 3 is declarative Terraform. If you lose the cluster, run Phases 1-3
again and you get an identical platform. No manual kubectl commands, no imperative steps.
The sole exception is Sealed Secrets -- the controller's private key must be backed up
separately (see the Sealed Secrets section).

Application workloads (zenith-api, zenith-landing, tenant deployments) are NOT managed by
this Terraform. They are managed by ArgoCD in Phase 4. This separation is deliberate:
infrastructure changes are slow, reviewed, and applied by operators. Application changes
are fast, continuous, and deployed automatically via Git push.

---

## What Gets Installed

All components, listed in the order Terraform applies them:

| # | Component | Namespace | Helm Chart | Version | Purpose |
|---|-----------|-----------|------------|---------|---------|
| 1 | cert-manager | `cert-manager` | `jetstack/cert-manager` | v1.17.2 | TLS automation (DNS-01 via Cloudflare) |
| 2 | ClusterIssuer | `cert-manager` | `kubernetes_manifest` | -- | Let's Encrypt ACME solver (DNS-01) |
| 3 | CNPG Operator | `cnpg-system` | `cnpg/cloudnative-pg` | 0.23.0 | PostgreSQL lifecycle management |
| 4 | Keycloak PG Cluster | `keycloak` | `kubernetes_manifest` (CNPG CR) | -- | Dedicated PG for identity provider |
| 5 | Keycloak | `keycloak` | `bitnami/keycloak` | 24.4.0 | Identity provider (realm per customer) |
| 6 | APISIX + etcd | `apisix` | `apisix/apisix` | 2.10.0 | API gateway (JWT, CORS, rate-limit) |
| 7 | APISIX Ingress Controller | `apisix` | `apisix/apisix-ingress-controller` | 0.14.0 | Translates CRDs to APISIX config |
| 8 | external-dns | `external-dns` | `bitnami/external-dns` | 8.7.0 | Auto DNS via Cloudflare API |
| 9 | ArgoCD | `argocd` | `argoproj/argo-cd` | 7.8.0 | GitOps deployment (App-of-Apps) |
| 10 | Temporal | `temporal` | `temporalio/temporal` | 0.46.0 | Provisioning workflow engine |
| 11 | Harbor | `harbor` | `harbor/harbor` | 1.16.0 | Container and Helm chart registry |
| 12 | Kyverno | `kyverno` | `kyverno/kyverno` | 3.3.4 | Admission policy engine |
| 13 | Falco | `falco` | `falcosecurity/falco` | 4.15.0 | Runtime threat detection (eBPF) |
| 14 | Sealed Secrets | `sealed-secrets` | `bitnami-labs/sealed-secrets` | 2.17.0 | Encrypted secrets for GitOps |
| 15 | Velero | `velero` | `vmware-tanzu/velero` | 8.2.0 | Cluster backup to Hetzner S3 |
| 16 | Prometheus + Grafana + Alertmanager | `monitoring` | `prometheus-community/kube-prometheus-stack` | 68.4.0 | Metrics, dashboards, alerts |
| 17 | Loki | `monitoring` | `grafana/loki` | 6.24.0 | Log aggregation |
| 18 | Tempo | `monitoring` | `grafana/tempo` | 1.15.0 | Distributed trace storage |
| 19 | OpenTelemetry Collector | `monitoring` | `open-telemetry/opentelemetry-collector` | 0.108.0 | Trace and metric collection pipeline |
| 20 | Free Users PG Cluster | `zenith-shared` | `kubernetes_manifest` (CNPG CR) | -- | Shared PG for free-tier customers |
| 21 | Pro Shard 1 PG Cluster | `zenith-shared` | `kubernetes_manifest` (CNPG CR) | -- | First Pro-tier PG shard |
| 22 | PriorityClasses | -- | `kubernetes_manifest` | -- | Eviction ordering (4 tiers) |
| 23 | PodDisruptionBudgets | various | `kubernetes_manifest` | -- | HA protection for infra pods |

**Total resource footprint (staging):** approximately 4 GB RAM, 2 vCPU requests.
Production doubles replica counts and allocates more generous resource limits.

---

## Architecture Overview

This ASCII diagram shows how all Phase 3 components fit into the cluster after
installation. Components are grouped by their namespace.

```
+=========================================================================+
|                              INTERNET                                    |
+=========================================================================+
         |
         v
+------------------+
| Cloudflare       |    DNS zones: freezenith.com, embermind.app
| DDoS / WAF / CDN |    cert-manager creates TXT records via API (DNS-01)
+--------+---------+    external-dns creates A/CNAME records via API
         |
         v
+=========================================================================+
|  k3s CLUSTER (Phase 2 provided)                                         |
|                                                                          |
|  kube-system (k3s built-in):                                            |
|  +-- Traefik (:80, :443)    TLS termination, IngressRoute routing       |
|  +-- Cilium (DaemonSet)     CNI, NetworkPolicy, WireGuard, kube-proxy   |
|  +-- Hubble (relay + UI)    Network flow visibility (part of Cilium)    |
|  +-- CoreDNS                Cluster DNS                                  |
|  +-- Metrics Server         kubectl top, HPA                            |
|                                                                          |
|  cert-manager:                                                          |
|  +-- cert-manager           Controller + Webhook + CA Injector          |
|  +-- ClusterIssuer          letsencrypt-prod (DNS-01, Cloudflare)       |
|                                                                          |
|  cnpg-system:                                                           |
|  +-- cnpg-operator          Watches ALL namespaces for Cluster CRs      |
|                                                                          |
|  keycloak:                                                              |
|  +-- keycloak-pg            CNPG Cluster (2 instances, Hetzner Volume)  |
|  +-- keycloak               Identity provider (realm per customer)      |
|  +-- keycloak-tls           Certificate (cert-manager, DNS-01)          |
|                                                                          |
|  apisix:                                                                |
|  +-- apisix-etcd            StatefulSet (1 staging, 3 production)       |
|  +-- apisix-gateway         Single gateway, route-level plugin config   |
|  +-- apisix-ingress-ctrl    Watches ApisixRoute CRDs cluster-wide       |
|                                                                          |
|  external-dns:                                                          |
|  +-- external-dns           Watches Ingress, creates Cloudflare records |
|                                                                          |
|  argocd:                                                                |
|  +-- argocd-server          Web UI + API                                |
|  +-- argocd-repo-server     Git clone + manifest rendering              |
|  +-- argocd-app-controller  Syncs Applications to cluster               |
|  +-- argocd-redis           In-memory cache                             |
|  +-- zenith-apps (root App) App-of-Apps: syncs all child Applications   |
|                                                                          |
|  temporal:                                                              |
|  +-- temporal-server        Workflow engine (frontend, history, worker)  |
|  +-- temporal-ui            Workflow visibility dashboard                |
|  +-- temporal-pg            CNPG Cluster (or shared PG)                 |
|                                                                          |
|  harbor:                                                                |
|  +-- harbor-core            Registry API + auth                         |
|  +-- harbor-registry        Image storage (S3 backend)                  |
|  +-- harbor-jobservice      Async operations (replication, GC)          |
|  +-- harbor-trivy           Vulnerability scanning                      |
|  +-- harbor-portal          Web UI                                      |
|                                                                          |
|  kyverno:                                                               |
|  +-- admission-controller   Validates/mutates on CREATE/UPDATE          |
|  +-- background-controller  Scans existing resources                    |
|  +-- reports-controller     Policy report generation                    |
|  +-- ClusterPolicies        11 enforcement policies (see Security)      |
|                                                                          |
|  falco:                                                                 |
|  +-- falco (DaemonSet)      eBPF syscall monitoring on every node       |
|  +-- falcosidekick           Alert routing (Slack, PagerDuty, Loki)     |
|                                                                          |
|  sealed-secrets:                                                        |
|  +-- sealed-secrets-ctrl    Decrypts SealedSecrets -> Secrets           |
|                                                                          |
|  velero:                                                                |
|  +-- velero                 Backup controller                           |
|  +-- node-agent (DaemonSet) File-system backup for PVs                  |
|  +-- BackupStorageLocation  s3://zenith-backups/velero/                 |
|  +-- Schedule: daily-backup 02:00 UTC, 30-day retention                 |
|                                                                          |
|  monitoring:                                                            |
|  +-- prometheus             Metrics scraping + storage                  |
|  +-- grafana                Dashboards (metrics + logs + traces)        |
|  +-- alertmanager           Alert routing (Slack, PagerDuty, Telegram)  |
|  +-- loki                   Log aggregation + query engine              |
|  +-- tempo                  Distributed trace storage                   |
|  +-- otel-collector (DS)    OTLP receiver, forwards to Tempo+Prometheus |
|  +-- node-exporter (DS)     Host-level metrics (CPU, memory, disk, net) |
|  +-- kube-state-metrics     Kubernetes object metrics                   |
|                                                                          |
|  zenith-shared:                                                         |
|  +-- free-users-pg          CNPG Cluster for free-tier customer DBs     |
|  +-- pro-shard-1-pg         CNPG Cluster for first Pro shard (~20 cust) |
|                                                                          |
|  (cluster-scoped):                                                      |
|  +-- PriorityClass: system-critical  (1000000)                          |
|  +-- PriorityClass: infra-critical   (500000)                           |
|  +-- PriorityClass: platform         (100000)                           |
|  +-- PriorityClass: customer         (10000, globalDefault)             |
+=========================================================================+
```

---

## Namespace Layout

Phase 3 creates 12 namespaces. Each namespace is purpose-scoped and contains only
related resources. This isolation is important for RBAC, NetworkPolicy, and resource
quotas.

| Namespace | Purpose | Created By | Key Resources |
|-----------|---------|------------|---------------|
| `cert-manager` | TLS certificate automation | Terraform (`helm_release`) | cert-manager pods, ClusterIssuer, Cloudflare API token Secret |
| `cnpg-system` | PostgreSQL operator | Terraform (`helm_release`) | CNPG operator pod, CRDs |
| `keycloak` | Identity provider | Terraform (`helm_release` + `kubernetes_manifest`) | Keycloak pods, keycloak-pg CNPG Cluster, TLS cert |
| `apisix` | API gateway | Terraform (`helm_release`) | etcd StatefulSet, gateway pods, ingress controller |
| `external-dns` | Automatic DNS | Terraform (`helm_release`) | external-dns pod, Cloudflare API token |
| `argocd` | GitOps engine | Terraform (`helm_release`) | ArgoCD server, repo-server, app-controller, root Application |
| `temporal` | Provisioning workflows | Terraform (`helm_release`) | temporal-server, temporal-worker, temporal-ui |
| `harbor` | Container registry | Terraform (`helm_release`) | core, registry, jobservice, trivy, portal |
| `kyverno` | Policy engine | Terraform (`helm_release`) | admission-controller, background-controller, ClusterPolicies |
| `falco` | Runtime security | Terraform (`helm_release`) | Falco DaemonSet (one pod per node) |
| `sealed-secrets` | Encrypted secrets | Terraform (`helm_release`) | controller pod, signing key Secret |
| `velero` | Cluster backup | Terraform (`helm_release`) | velero pod, node-agent DaemonSet, BackupStorageLocation |
| `monitoring` | Observability | Terraform (`helm_release`) | Prometheus, Grafana, Alertmanager, Loki, Tempo, OTel Collector |
| `zenith-shared` | Shared databases | Terraform (`kubernetes_manifest`) | free-users-pg CNPG Cluster, pro-shard-1-pg CNPG Cluster |

Namespaces that Phase 3 does **not** create (they come later):

| Namespace | Created By | When |
|-----------|------------|------|
| `zenith-platform` | ArgoCD (Phase 4) | Git push triggers sync |
| `zenith-customer-*` | Temporal (customer signup) | Provisioning workflow |

---

## Installation Order (Dependency Graph)

The order matters. cert-manager must exist before any Certificate resource can be
issued. The CNPG operator must exist before any PostgreSQL Cluster CR can be created.
Keycloak must be running before APISIX can configure JWT verification against it.
ArgoCD is installed last because it needs all infrastructure services to be healthy
before it starts syncing applications.

```
                          +-------------------+
                          |  cert-manager     |  <-- FIRST: everything needs TLS
                          |  + ClusterIssuer  |
                          +--------+----------+
                                   |
                          +--------v----------+
                          |  CNPG Operator    |  <-- Watches all namespaces
                          +--------+----------+
                                   |
              +--------------------+--------------------+
              |                                         |
     +--------v----------+                    +---------v---------+
     | Keycloak PG       |                    | PriorityClasses   |
     | (CNPG Cluster CR) |                    | (cluster-scoped)  |
     +--------+----------+                    +-------------------+
              |
     +--------v----------+
     | Keycloak Deploy   |
     | (needs PG ready)  |
     +--------+----------+
              |
     +--------v----------+
     | APISIX stack      |  <-- JWT plugin needs Keycloak JWKS endpoint
     | (etcd + gateway   |
     |  + ingress ctrl)  |
     +--------+----------+
              |
     +--------+---+---+---+---+---+---+---+--------+
     |         |   |   |   |   |   |   |            |
  +--v---+ +--v-+ | +-v-+ | +-v-+ | +-v-------+ +--v---+
  |ext-  | |Seal| | |Ky | | |Fal| | |Temporal | |Harbor|
  |dns   | |Sec | | |ver| | |co | | |(needs   | |(needs|
  +------+ +----+ | |no | | +---+ | | PG+KC)  | | TLS) |
                   | +---+ |       | +---------+ +------+
                   |       |       |
              +----v-------v-------v-----+
              |        Velero            |  <-- Needs S3 configured
              +-----------+--------------+
                          |
              +-----------v--------------+
              |    Monitoring Stack      |  <-- Scrapes ALL above
              | Prometheus + Grafana     |
              | + Loki + Tempo + OTel    |
              | + Alertmanager           |
              +-----------+--------------+
                          |
              +-----------v--------------+
              |       ArgoCD             |  <-- LAST: needs everything
              |  + root Application      |     healthy before syncing
              +-----------+--------------+
                          |
              +-----------v--------------+
              | Shared CNPG Clusters     |  <-- Can run parallel with ArgoCD
              | (free-users-pg,          |
              |  pro-shard-1-pg)         |
              +--------------------------+
```

**In Terraform terms**, the `depends_on` chain:

```hcl
cert_manager
  -> cluster_issuer
    -> cnpg_operator
      -> keycloak_pg_cluster
        -> keycloak
          -> apisix + apisix_ingress_controller
            -> external_dns, sealed_secrets, kyverno, falco (parallel)
              -> temporal, harbor (parallel)
                -> velero
                  -> monitoring (prometheus, loki, tempo, otel_collector)
                    -> argocd + argocd_root_app
                      -> shared_cnpg_clusters
```

Components at the same level in the graph (external-dns, Sealed Secrets, Kyverno, Falco)
have no interdependencies. Terraform applies them in parallel, which significantly
reduces total apply time.

---

## Component Deep Dives: Networking

### APISIX + etcd

**What it does:**
Apache APISIX is a high-performance API gateway built on NGINX and OpenResty. In
Zenith V2 it replaces Kong (Decision D1) and handles all backend API traffic:

- **JWT verification** against Keycloak (per-customer realm JWKS endpoint)
- **CORS enforcement** with per-customer allowed origins
- **Rate limiting** tiered by customer plan (Free: 100 req/min, Pro: 1000 req/min)
- **Request/response transformation** (add X-Consumer headers, strip paths)
- **OpenTelemetry integration** (inject trace IDs into every API call)

etcd is APISIX's configuration store. Unlike Kong (which uses PostgreSQL or DB-less YAML
files), APISIX reads its route configuration from etcd. Configuration changes are applied
in real time without gateway restarts or reloads.

**Why APISIX instead of Kong (Decision D1):**

| Criterion | APISIX | Kong OSS |
|-----------|--------|----------|
| Config store | etcd (real-time sync) | DB-less YAML or PostgreSQL |
| Plugin ecosystem | 80+ built-in (OTel, Keycloak, multi-auth) | 40+ (fewer built-in, enterprise behind paywall) |
| Performance | 2-3x RPS at same latency (OpenResty optimized) | Slower at high concurrency |
| JWT verification | Native JWKS discovery plugin | Requires custom plugin or Enterprise |
| Hot reload | Instant (etcd watch) | Requires restart or declarative re-apply |

**Namespace:** `apisix`

**Helm charts + versions:**
- `apisix/apisix` 2.10.0 (gateway + etcd)
- `apisix/apisix-ingress-controller` 0.14.0 (CRD watcher)

**Architecture: Three Components**

```
namespace: apisix
  |
  |-- StatefulSet: apisix-etcd (1 staging / 3 production)
  |     |-- apisix-etcd-0   -> PVC 5Gi (hcloud-volumes)
  |     |-- apisix-etcd-1   -> PVC 5Gi (production only)
  |     |-- apisix-etcd-2   -> PVC 5Gi (production only)
  |     |-- Service: apisix-etcd:2379 (client), :2380 (peer)
  |
  |-- Deployment: apisix-gateway (1 staging / 2 production)
  |     |-- Plugins applied per-route (not globally)
  |     |-- Service: apisix-gateway:9080
  |     |-- IngressClass: apisix
  |
  |-- Deployment: apisix-ingress-controller (1 replica)
  |     |-- Watches ApisixRoute CRDs in ALL namespaces
  |     |-- Translates CRDs into APISIX Admin API calls via etcd
```

**Single gateway, route-level plugins (Decision D9):**

Not all API routes require authentication. Webhooks from Stripe or GitHub, unsubscribe
links in emails, and health check endpoints must be publicly accessible without a JWT.

APISIX applies plugins **per-route**, not globally. This means a single gateway deployment
handles both protected and public routes — each route simply declares its own plugin chain:

```
APISIX (single gateway deployment)
  |
  |-- Route: /v1/api/*           plugins: [jwt-auth, cors, limit-req, opentelemetry]
  |-- Route: /v1/webhooks/*      plugins: [cors, limit-req, opentelemetry]
  |-- Route: /v1/unsubscribe/*   plugins: [cors, limit-req, opentelemetry]
  |-- Route: /health             plugins: []
```

This is simpler than running two separate APISIX deployments and is how APISIX is designed
to work. Unlike Kong OSS (which lacks workspaces), APISIX natively supports per-route
plugin configuration with no additional complexity.

**Why etcd needs 3 pods in production:**

etcd uses the Raft consensus protocol. A 3-pod cluster tolerates 1 pod failure and
continues serving reads and writes. With only 1 pod, losing it means APISIX loses all
route configuration and stops routing traffic until etcd recovers. In staging, 1 etcd
pod is acceptable because downtime is not critical. Production must run 3.

**Key Terraform configuration:**

```hcl
resource "helm_release" "apisix" {
  count = var.enable_apisix ? 1 : 0

  name             = "apisix"
  repository       = "https://charts.apiseven.com"
  chart            = "apisix"
  version          = var.apisix_version
  namespace        = "apisix"
  create_namespace = true
  wait             = true
  timeout          = 600

  values = [templatefile("${path.module}/values/apisix.yaml", {
    etcd_replicas       = var.environment == "production" ? 3 : 1
    gateway_replicas    = var.environment == "production" ? 2 : 1
    keycloak_domain     = "auth.${var.domain}"
    otel_collector_host = "otel-collector.monitoring.svc.cluster.local"
  })]

  depends_on = [helm_release.keycloak]
}
```

**How APISIX connects to Keycloak:**

APISIX does not maintain a persistent connection to Keycloak. Instead, the `jwt-auth`
plugin is configured with the JWKS discovery URL for each customer realm. When a request
arrives:

1. APISIX extracts the JWT from the `Authorization: Bearer <token>` header
2. Reads the `iss` (issuer) claim to determine which realm issued the token
3. Fetches (and caches for 5 minutes) the JWKS from Keycloak:
   `https://auth.freezenith.com/realms/<realm>/protocol/openid-connect/certs`
4. Verifies the token signature against the JWKS public key
5. Checks `exp` (not expired), `aud` (correct audience), `iss` (correct issuer)
6. If valid: forwards the request with `X-Consumer-ID`, `X-Consumer-Realm`, `X-Token-Sub`
7. If invalid: returns `401 Unauthorized`

```
Client                APISIX                      Keycloak
  |                     |                             |
  | Authorization:      |                             |
  | Bearer <JWT>        |                             |
  |-------------------->|                             |
  |                     | GET /realms/X/.../certs     |
  |                     | (only on cache miss)        |
  |                     |---------------------------->|
  |                     |                             |
  |                     | { keys: [{ kid, n, e }] }   |
  |                     |<----------------------------|
  |                     |                             |
  |                     | Verify signature (RS256)    |
  |                     | Check exp, aud, iss         |
  |                     |                             |
  |   200 OK + headers  |                             |
  |<--------------------|                             |
```

**How to verify:**

```bash
# etcd is healthy
kubectl -n apisix get pods -l app.kubernetes.io/name=etcd
# apisix-etcd-0    1/1     Running   0

# Gateway pods are running
kubectl -n apisix get pods -l app.kubernetes.io/name=apisix

# Ingress controller is watching
kubectl -n apisix get pods -l app.kubernetes.io/name=apisix-ingress-controller

# Test a public route
curl -v https://api.stage.freezenith.com/health
# Should return 200

# Test a protected route without JWT
curl -v https://api.stage.freezenith.com/v1/me
# Should return 401 Unauthorized
```

---

### external-dns

**What it does:**
external-dns watches Kubernetes Ingress and Service resources for specific annotations,
then creates, updates, or deletes DNS records in Cloudflare automatically.

**Why we need it:**
When a new customer is provisioned by Temporal, their Ingress resource is created with
a host like `customer-abc.freezenith.com`. Without external-dns, an operator would need
to manually create a Cloudflare A record for every single customer. With 100+ customers,
manual DNS management is operationally impossible. external-dns closes the loop: create
an Ingress, get a DNS record within 30 seconds.

**Namespace:** `external-dns`

**Helm chart + version:** `bitnami/external-dns` 8.7.0

**Key configuration choices:**

| Setting | Value | Why |
|---------|-------|-----|
| `provider` | `cloudflare` | Our DNS is managed in Cloudflare |
| `domainFilters` | `["freezenith.com", "embermind.app"]` | Only manage records for our zones (prevent accidental changes elsewhere) |
| `annotationFilter` | `external-dns.alpha.kubernetes.io/enabled=true` | Only process Ingresses that explicitly opt in (prevents creating records for internal services) |
| `txtOwnerId` | `zenith-staging` / `zenith-production` | TXT ownership records prevent external-dns from deleting records it did not create |

**How it connects to other components:**
- Uses the **same Cloudflare API token** as cert-manager (but for A/CNAME creation, not TXT)
- Watches **Ingress resources** created by ArgoCD (Phase 4) and Temporal workflows
- Creates A records pointing to the **Traefik external IP** (the k3s node IP)

**How to verify:**

```bash
kubectl -n external-dns get pods
kubectl -n external-dns logs -l app.kubernetes.io/name=external-dns --tail=10
# Should show: "All records are already up to date" or "Changing record..."
```

---

## Component Deep Dives: Identity

### Keycloak + Dedicated CNPG

**What it does:**
Keycloak is a full-featured identity and access management (IAM) server. In Zenith V2
it replaces the JWT signing logic that lived in the zenith-api Go backend. Keycloak
provides:

- **One realm per customer** -- complete identity isolation between tenants
- **OIDC/OAuth2 provider** -- standard-compliant JWT tokens
- **JWKS endpoint** -- APISIX fetches public keys to verify tokens
- **Admin console** -- manage users, roles, clients via web UI
- **Admin REST API** -- Temporal workflows create realms and clients programmatically

**Why we need it:**
V1 handled JWT signing directly in zenith-api. This works for a single-tenant demo but
fails with multi-tenancy: every customer needs separate token namespaces, password
policies, SSO configurations, and admin capabilities. Building this from scratch would
replicate what Keycloak already provides.

**Why a dedicated CNPG cluster (Decision D3):**
Keycloak is a critical-path dependency for every authenticated API call. If the
free-users PG cluster experiences high load from a customer running expensive queries,
Keycloak must not be affected. Separate CNPG clusters mean separate primary pods,
separate WAL writers, separate Hetzner Volumes, and separate failure domains.

**Namespace:** `keycloak`

**Helm chart + version:** `bitnami/keycloak` 24.4.0

**Architecture:**

```
namespace: keycloak
  |
  |-- CNPG Cluster: keycloak-pg
  |     |-- keycloak-pg-1  (primary)   -> Hetzner Volume 10Gi
  |     |-- keycloak-pg-2  (replica)   -> Hetzner Volume 10Gi
  |     |-- keycloak-pg-3  (replica)   -> production only
  |     |-- Service: keycloak-pg-rw    -> always points to primary
  |     |-- Service: keycloak-pg-ro    -> load-balanced across replicas
  |     |-- WAL archiving -> s3://zenith-backups/keycloak-wal/
  |
  |-- Deployment: keycloak (1-2 replicas)
  |     |-- Connects to keycloak-pg-rw:5432
  |     |-- Service: keycloak:8080
  |     |-- Ingress: auth.freezenith.com (via Traefik, TLS)
  |
  |-- Certificate: keycloak-tls (cert-manager, DNS-01)
```

**JWKS endpoint URL pattern:**

```
https://auth.freezenith.com/realms/<customer-name>/protocol/openid-connect/certs
```

Each customer realm has its own RSA signing key pair. A token issued for customer A
cannot pass verification for customer B because the key IDs and issuer claims differ.

**Key Terraform configuration:**

```hcl
# CNPG Cluster for Keycloak
resource "kubernetes_manifest" "keycloak_pg" {
  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"
    metadata = {
      name      = "keycloak-pg"
      namespace = "keycloak"
    }
    spec = {
      instances = var.environment == "production" ? 3 : 2
      storage = {
        size         = "10Gi"
        storageClass = "hcloud-volumes"
      }
      postgresql = {
        parameters = {
          max_connections      = "100"
          shared_buffers       = "128MB"
          effective_cache_size = "256MB"
        }
      }
      backup = {
        barmanObjectStore = {
          destinationPath = "s3://zenith-backups/keycloak-wal/"
          endpointURL     = var.s3_endpoint
          s3Credentials   = { ... }
          wal             = { compression = "gzip" }
        }
        retentionPolicy = "30d"
      }
    }
  }
  depends_on = [helm_release.cnpg]
}

# Keycloak Helm release
resource "helm_release" "keycloak" {
  count            = var.enable_keycloak ? 1 : 0
  name             = "keycloak"
  repository       = "https://charts.bitnami.com/bitnami"
  chart            = "keycloak"
  version          = var.keycloak_version
  namespace        = "keycloak"
  create_namespace = true
  wait             = true
  timeout          = 600

  set { name = "postgresql.enabled"; value = "false" }  # Use external PG
  set { name = "externalDatabase.host"
        value = "keycloak-pg-rw.keycloak.svc.cluster.local" }
  set { name = "proxy"; value = "edge" }  # Behind Traefik

  set_sensitive { name = "auth.adminPassword"; value = var.keycloak_admin_password }

  depends_on = [kubernetes_manifest.keycloak_pg]
}
```

**How to verify:**

```bash
# CNPG Cluster is healthy
kubectl -n keycloak get cluster keycloak-pg
# NAME          INSTANCES   READY   STATUS    AGE
# keycloak-pg   2           2       Cluster   5m

# Keycloak pod is running
kubectl -n keycloak get pods -l app.kubernetes.io/name=keycloak

# Admin console is accessible
curl -sI https://auth.stage.freezenith.com/admin/
# HTTP/2 200
```

---

## Component Deep Dives: Data

### CNPG Operator

**What it does:**
CloudNativePG (CNPG) is a Kubernetes operator that manages the full lifecycle of
PostgreSQL clusters: provisioning, automatic failover (< 30 seconds), WAL archiving
to S3, point-in-time recovery, rolling updates, and integrated connection pooling via
PgBouncer.

**Why we need it:**
Zenith runs multiple PostgreSQL clusters: one for Keycloak, one for free users, one per
Pro shard, one for Temporal's persistence, and potentially more as the platform grows.
Managing these with raw StatefulSets, manual replication configuration, and hand-written
failover scripts would be a maintenance disaster. CNPG provides declarative cluster
management via `Cluster` CRDs with production-grade automation.

**Namespace:** `cnpg-system`

**Helm chart + version:** `cnpg/cloudnative-pg` 0.23.0

**Key configuration choice -- cluster-wide watching:**
The operator is configured with `config.clusterWide = true` so it can watch Cluster CRs
in ALL namespaces: `keycloak`, `zenith-shared`, `temporal`, and any future namespaces. A
namespace-scoped operator would only see CRs in `cnpg-system`, missing all the actual
database clusters.

**How to verify:**

```bash
kubectl -n cnpg-system get pods
# cnpg-cloudnative-pg-<hash>    1/1     Running

kubectl get crd clusters.postgresql.cnpg.io
# clusters.postgresql.cnpg.io    2026-02-25T...
```

---

### CNPG Clusters (Keycloak, Free, Pro Shards)

**What gets created (Decision D2 + D3):**

Phase 3 creates three CNPG Cluster CRs. The operator provisions actual PostgreSQL pods
and Hetzner Volumes for each:

```
CNPG Operator (cnpg-system)
  |
  |-- Watches ALL namespaces for Cluster CRs
  |
  |-- Keycloak PG Cluster (namespace: keycloak) [Decision D3: isolated]
  |     |-- keycloak-pg-1 (primary)   -> Hetzner Volume: 10Gi
  |     |-- keycloak-pg-2 (replica)   -> Hetzner Volume: 10Gi
  |     |-- Database: keycloak
  |     WAL archiving -> s3://zenith-backups/keycloak-wal/
  |
  |-- Free Users PG Cluster (namespace: zenith-shared)
  |     |-- free-users-pg-1 (primary)    -> Hetzner Volume: 50Gi
  |     |-- free-users-pg-2 (replica)    -> Hetzner Volume: 50Gi
  |     |-- Databases:
  |     |     zenith_platform (Zenith's own operational DB)
  |     |     customer_abc, customer_def, ... (all free users)
  |     WAL archiving -> s3://zenith-backups/free-pg-wal/
  |     pg_dump CronJob -> per-customer dumps to S3
  |
  |-- Pro Shard 1 PG Cluster (namespace: zenith-shared)
        |-- pro-shard-1-pg-1 (primary)   -> Hetzner Volume: 100Gi
        |-- pro-shard-1-pg-2 (replica)   -> Hetzner Volume: 100Gi
        |-- Databases:
        |     customer_pro_001 (up to 5 DBs per customer)
        |     customer_pro_002 ...
        |     (max ~20 pro customers per shard)
        WAL archiving -> s3://zenith-backups/pro-shard1-wal/
```

**Sharding strategy (Decision D2):**

Free-tier customers share one PG cluster. All their databases live on the same primary
pod and the same Hetzner Volume. This is cost-effective but means a failing query in one
customer's DB can impact others sharing the cluster.

Pro-tier customers get sharded PG clusters. Each shard holds approximately 20 customers.
When a shard is full, the platform admin creates a new shard (or Temporal auto-provisions
one). This limits blast radius: a runaway query in one shard cannot affect customers on
other shards.

**Database creation flow (how Temporal uses these clusters):**

1. Customer signs up (Free tier)
2. Temporal workflow calls zenith-api
3. zenith-api connects to `free-users-pg-rw.zenith-shared.svc.cluster.local:5432`
4. Runs: `CREATE DATABASE customer_xxx; CREATE USER customer_xxx WITH PASSWORD ...;`
5. Stores credentials in a Kubernetes Secret in the customer's namespace
6. Customer's backend reads the Secret and connects to their database

**Pro user shard assignment:**

1. zenith-api checks which pro shards have capacity (< 20 customers)
2. Assigns customer to first available shard
3. If no shard has capacity, alerts admin (or auto-creates a new shard)

**How to verify:**

```bash
kubectl -n zenith-shared get cluster
# NAME              INSTANCES   READY   STATUS    AGE
# free-users-pg     2           2       Cluster   5m
# pro-shard-1-pg    2           2       Cluster   5m

kubectl -n zenith-shared exec -it free-users-pg-1 -- \
  psql -U zenith_admin -d zenith_platform -c "SELECT 1;"
# 1
```

---

## Component Deep Dives: Platform

### ArgoCD (App-of-Apps)

**What it does:**
ArgoCD is a GitOps continuous delivery tool (Decision D4). It watches a Git repository
for Kubernetes manifest changes and automatically applies them to the cluster. In Zenith
V2, ArgoCD manages ALL application deployments (Phase 4).

**Why ArgoCD instead of FluxCD (Decision D4):**

| Criterion | ArgoCD | FluxCD |
|-----------|--------|--------|
| Web UI | Built-in, rich visualization | Weave GitOps (separate install) |
| App-of-Apps | Native pattern | Kustomization nesting (less intuitive) |
| Sync status | Clear visual: Synced/OutOfSync/Degraded | Requires CLI or Weave |
| Multi-cluster | Built-in cluster management | Requires additional config |
| Rollback | One-click in UI or CLI | Manual revert + commit |
| Community | CNCF Graduated, massive adoption | CNCF Graduated, smaller community |

ArgoCD's web UI is a significant advantage for a small team. Seeing the entire cluster
state, sync status, and health of every Application in one dashboard eliminates the need
for custom monitoring of deployment pipelines.

**Namespace:** `argocd`

**Helm chart + version:** `argoproj/argo-cd` 7.8.0

**App-of-Apps pattern:**

Terraform creates exactly ONE ArgoCD Application: the "root app" called `zenith-apps`.
This root app points to a directory in the Git repository (`infra/argocd/staging/` or
`infra/argocd/production/`) that contains child Application manifests. Each child
Application manages one concern:

```
zenith-apps (root Application, created by Terraform)
  |
  |-- watches: infra/argocd/staging/
  |
  |-- child Applications (discovered automatically):
        |-- zenith-api.yaml        -> deploys zenith-api Helm chart
        |-- zenith-landing.yaml    -> deploys zenith-landing Helm chart
        |-- zenith-tenant.yaml     -> deploys tenant Helm chart
        |-- zenith-demo.yaml       -> deploys demo Helm chart (optional)
```

**Why ArgoCD is installed last:**
ArgoCD immediately starts syncing Applications from Git. If cert-manager, CNPG, Keycloak,
APISIX, or Harbor are not yet running, the Applications will fail to deploy and enter a
degraded state. By installing ArgoCD last, all dependencies are guaranteed to be healthy
before it begins syncing.

**Key Terraform configuration:**

```hcl
resource "helm_release" "argocd" {
  count = var.enable_argocd ? 1 : 0

  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = var.argocd_version
  namespace        = "argocd"
  create_namespace = true
  wait             = true
  timeout          = 600

  set { name = "server.extraArgs[0]"; value = "--insecure" }  # TLS by Traefik

  set_sensitive {
    name  = "configs.secret.argocdServerAdminPassword"
    value = var.argocd_admin_password   # bcrypt hash
  }

  depends_on = [helm_release.kube_prometheus]
}

# Root Application (App-of-Apps) -- the ONLY Application Terraform creates
resource "kubernetes_manifest" "argocd_root_app" {
  count = var.enable_argocd ? 1 : 0

  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata   = { name = "zenith-apps"; namespace = "argocd" }
    spec = {
      project = "default"
      source = {
        repoURL        = "https://github.com/DoTech/Zenith.git"
        targetRevision = var.environment == "production" ? "main" : "staging"
        path           = "infra/argocd/${var.environment}"
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "argocd"
      }
      syncPolicy = {
        automated = { prune = true; selfHeal = true }
      }
    }
  }
  depends_on = [helm_release.argocd]
}
```

**How to verify:**

```bash
kubectl -n argocd get pods
# argocd-server-<hash>                    1/1     Running
# argocd-repo-server-<hash>               1/1     Running
# argocd-application-controller-0         1/1     Running

kubectl -n argocd get application zenith-apps -o jsonpath='{.status.sync.status}'
# Synced
```

---

### Temporal

**What it does:**
Temporal is a workflow orchestration engine (Decision D5). It manages long-running,
durable workflows with automatic retries, timeouts, and full execution history. In
Zenith V2, Temporal powers the customer provisioning pipeline.

**Why we need it (Decision D5):**
Customer provisioning involves 10 sequential steps: create Keycloak realm, create DB,
create S3 bucket, create namespace, create Secrets, create Deployments, create Ingress,
wait for DNS, wait for TLS, notify customer. Each step can fail independently. Without
Temporal, a failure at step 7 would require either restarting from step 1 (wasteful and
potentially destructive) or building custom retry/checkpoint logic (complex and fragile).

Temporal provides exactly this: durable workflow execution with activity-level retries
and the ability to resume from the exact point of failure.

**Namespace:** `temporal`

**Helm chart + version:** `temporalio/temporal` 0.46.0

**Architecture:**

```
namespace: temporal
  |
  |-- Deployment: temporal-server (1 replica)
  |     |-- frontend, matching, history, worker services (all-in-one for staging)
  |     |-- Connects to temporal-pg for persistence
  |
  |-- Deployment: temporal-worker (1 replica)
  |     |-- Runs Zenith provisioning activities
  |     |-- Service accounts for: K8s API, Keycloak Admin API, Hetzner API, Cloudflare API
  |
  |-- Deployment: temporal-ui (1 replica)
  |     |-- Web UI for workflow visibility
  |     |-- Ingress: temporal.stage.freezenith.com
  |
  |-- Database:
        Option A: Own CNPG Cluster (temporal-pg, 2 instances)
        Option B: Database in the shared free-users PG cluster
```

**How to verify:**

```bash
kubectl -n temporal get pods
# temporal-server-<hash>     1/1     Running
# temporal-worker-<hash>     1/1     Running
# temporal-ui-<hash>         1/1     Running
```

---

### Harbor

**What it does:**
Harbor is an enterprise container registry with built-in vulnerability scanning (Trivy),
image signing (cosign), replication policies, and role-based access control. In Zenith V2,
Harbor stores:

- **Container images** built by CI, pulled by ArgoCD or kubelet
- **Helm charts** in OCI format for application deployments
- **Per-customer quotas** (Free: 1 GB, Pro: 10 GB, Team: 100 GB)

**Why we need it:**
Pulling images from Docker Hub or GitHub Container Registry in production is unreliable
(rate limits, outages) and insecure (no control over what gets deployed). Harbor gives us
a private registry with Trivy scanning (block images with critical CVEs), combined with
Kyverno admission policies (block unsigned images). Together they form a complete image
supply chain security solution (see Decision D11).

**Namespace:** `harbor`

**Helm chart + version:** `harbor/harbor` 1.16.0

**Storage backend:** Hetzner S3 Object Storage (not local PVCs). This means image layers
are stored in a durable, cost-effective S3 bucket (`zenith-harbor`), and Harbor pods are
stateless and can be restarted freely.

**How to verify:**

```bash
kubectl -n harbor get pods
# harbor-core-<hash>       1/1     Running
# harbor-registry-<hash>   1/1     Running
# harbor-trivy-<hash>      1/1     Running

docker login registry.stage.freezenith.com
# Username: admin
# Password: <harbor_admin_password>
```

---

## Component Deep Dives: Security

### cert-manager

**What it does:**
cert-manager automates TLS certificate issuance and renewal using Let's Encrypt. It
watches Certificate resources and creates corresponding Secrets containing the TLS key
pair. Certificates are automatically renewed 30 days before expiry.

**Why we need it:**
Every HTTPS endpoint in the cluster needs a TLS certificate. Without cert-manager, you
would manually obtain and rotate certificates for every service, every domain, and every
customer. With 100+ customers, this is operationally impossible.

**V2 change -- DNS-01 instead of HTTP-01:**

V1 used HTTP-01 challenges, which required Cloudflare proxy to be OFF during certificate
issuance (the ACME server needs to reach your server directly on port 80). V2 switches
to DNS-01 challenges via the Cloudflare API:

```
V1 (HTTP-01):                        V2 (DNS-01):

Let's Encrypt                        Let's Encrypt
     |                                    |
     | GET /.well-known/acme/...          | Check TXT _acme-challenge.<domain>
     v                                    v
  Your Server:80  <-- must be direct   Cloudflare DNS API  <-- no server contact
  (proxy OFF!)                         (proxy stays ON, DDoS protection intact)
```

Benefits of DNS-01:
- Cloudflare proxy can stay ON permanently (no DDoS protection gaps)
- Wildcard certificates are possible (`*.freezenith.com`)
- No port 80 listener required during challenge
- Works even if the server is temporarily unreachable

**Namespace:** `cert-manager`

**Helm chart + version:** `jetstack/cert-manager` v1.17.2

**Key Terraform configuration:**

```hcl
resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  version          = var.cert_manager_version
  namespace        = "cert-manager"
  create_namespace = true
  wait             = true
  timeout          = 300

  set { name = "crds.enabled"; value = "true" }
  set { name = "replicaCount"
        value = var.environment == "production" ? "2" : "1" }
}

# Cloudflare API token Secret for DNS-01 solver
resource "kubernetes_secret" "cloudflare_api_token" {
  metadata {
    name      = "cloudflare-api-token"
    namespace = "cert-manager"
  }
  data = { api-token = var.cloudflare_api_token }
  depends_on = [helm_release.cert_manager]
}

# ClusterIssuer with DNS-01 solver
resource "kubernetes_manifest" "cluster_issuer" {
  manifest = {
    apiVersion = "cert-manager.io/v1"
    kind       = "ClusterIssuer"
    metadata   = { name = "letsencrypt-prod" }
    spec = {
      acme = {
        server = "https://acme-v02.api.letsencrypt.org/directory"
        email  = var.cert_issuer_email
        privateKeySecretRef = { name = "letsencrypt-prod-key" }
        solvers = [{
          dns01 = {
            cloudflare = {
              apiTokenSecretRef = {
                name = "cloudflare-api-token"
                key  = "api-token"
              }
            }
          }
          selector = { dnsZones = [var.dns_zone] }
        }]
      }
    }
  }
  depends_on = [kubernetes_secret.cloudflare_api_token]
}
```

**Cloudflare API token permissions required:**
- `Zone:DNS:Edit` scoped to `freezenith.com` zone
- `Zone:DNS:Edit` scoped to `embermind.app` zone (for customer domains)

**How to verify:**

```bash
kubectl -n cert-manager get pods
# cert-manager-<hash>             1/1     Running
# cert-manager-cainjector-<hash>  1/1     Running
# cert-manager-webhook-<hash>     1/1     Running

kubectl get clusterissuer letsencrypt-prod
# NAME               READY   AGE
# letsencrypt-prod   True    5m

# Test certificate issuance:
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test-cert
  namespace: default
spec:
  secretName: test-cert-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - test.stage.freezenith.com
EOF
kubectl describe certificate test-cert
# Clean up: kubectl delete certificate test-cert && kubectl delete secret test-cert-tls
```

---

### Kyverno

**What it does:**
Kyverno is a Kubernetes-native policy engine. It intercepts API server requests via
admission webhooks and validates, mutates, or generates resources based on declarative
policies. Unlike OPA/Gatekeeper, Kyverno policies are written in YAML (not Rego), making
them accessible to Kubernetes practitioners without learning a new language.

**Why we need it (part of Decision D11: defense-in-depth):**
In a multi-tenant platform, we need guardrails:

- Prevent deploying unsigned container images (supply chain security)
- Enforce required labels on all resources (observability, cost tracking)
- Block hostPath volume mounts (prevent container escape)
- Enforce resource limits on all pods (prevent noisy neighbors)
- Auto-generate NetworkPolicies for new namespaces
- Block NodePort services in customer namespaces

**Namespace:** `kyverno`

**Helm chart + version:** `kyverno/kyverno` 3.3.4

**Policies installed by Phase 3:**

| Policy | Type | Action | Target | Description |
|--------|------|--------|--------|-------------|
| `disallow-host-path` | Validate | Block | All namespaces | Prevent pods from mounting host filesystem |
| `disallow-host-namespaces` | Validate | Block | All namespaces | Block hostNetwork, hostPID, hostIPC |
| `disallow-privileged-containers` | Validate | Block | All namespaces | Block `privileged: true` |
| `disallow-capabilities` | Validate | Block | `zenith-*` | Drop all capabilities except NET_BIND_SERVICE |
| `require-run-as-non-root` | Validate | Block | `zenith-*` | All containers must run as non-root |
| `require-labels` | Validate | Audit/Block | `zenith-customer-*` | `zenith.io/customer`, `zenith.io/tier`, `zenith.io/component` |
| `require-resource-limits` | Validate | Block | `zenith-*` | CPU + memory limits mandatory |
| `restrict-image-registries` | Validate | Block | `zenith-*` | Only images from `registry.*.freezenith.com` |
| `verify-image-signature` | Validate | Block | `zenith-*` | cosign verification for all images |
| `generate-default-deny-np` | Generate | Auto | New namespaces | Create default-deny NetworkPolicy |
| `disallow-latest-tag` | Validate | Block | All namespaces | Block `:latest` image tags |

**How to verify:**

```bash
kubectl -n kyverno get pods
# kyverno-admission-controller-<hash>    1/1     Running
# kyverno-background-controller-<hash>   1/1     Running
# kyverno-reports-controller-<hash>      1/1     Running

kubectl get clusterpolicy
# NAME                              ADMISSION   BACKGROUND   VALIDATE ACTION   READY
# disallow-host-path                true        true         Enforce           True
# require-labels                    true        true         Audit             True
# verify-image-signature            true        false        Enforce           True
```

---

### Falco

**What it does:**
Falco is a runtime security tool that detects anomalous behavior inside containers. It
uses eBPF to monitor system calls at the Linux kernel level and compares them against
security rules. When a rule triggers, Falco generates an alert.

**Why we need it (part of Decision D11):**
Kyverno prevents bad configurations at admission time (deploy time). Falco detects bad
behavior at runtime. Together they form two complementary layers of defense:

```
Time axis --->

Deploy time:    Kyverno blocks bad pod specs
                |
Runtime:        |    Falco detects anomalous syscalls
                |    |
                v    v
   Pod spec     Pod runs     Attacker exploits     Falco alert
   validated    normally     application vuln      -> Slack/PagerDuty
```

**Namespace:** `falco`

**Helm chart + version:** `falcosecurity/falco` 4.15.0

**Deployment model:** DaemonSet (one pod per node). Each Falco pod monitors all
containers on its node via eBPF hooks into the kernel's syscall table.

**Key detection rules:**

| Rule | Severity | What it Detects |
|------|----------|-----------------|
| Terminal shell in container | WARNING | Interactive shell opened (`kubectl exec -it`) |
| Write below /etc | ERROR | Modification of system files inside container |
| Read sensitive file | WARNING | Reading `/etc/shadow`, `/etc/passwd` |
| Launch privileged container | CRITICAL | Container running with `privileged: true` |
| Outbound connection to mining pool | CRITICAL | Container connecting to known mining IPs |
| Unexpected process spawned | NOTICE | Binary not in expected process list for the image |

**Key configuration -- eBPF driver (not kernel module):**

```hcl
set { name = "driver.kind"; value = "ebpf" }
```

The eBPF driver does not require loading a kernel module, making it safer and more
portable. It works on any kernel >= 5.4 (Ubuntu 24.04 ships with 6.x).

**Alert output pipeline:**
Falco outputs JSON to stdout. This is collected by Promtail/Alloy (part of the Loki
stack in monitoring), making all Falco alerts searchable in Grafana. For critical alerts,
Falcosidekick forwards to Slack and PagerDuty.

**How to verify:**

```bash
kubectl -n falco get daemonset
# NAME    DESIRED   CURRENT   READY
# falco   1         1         1

# Trigger a test detection:
kubectl exec -it <any-pod> -- /bin/sh
# Check Falco logs:
kubectl -n falco logs -l app.kubernetes.io/name=falco --tail=5
# {"output":"Warning Shell spawned in container ...","priority":"Warning",...}
```

---

### Sealed Secrets

**What it does:**
Sealed Secrets is a Kubernetes controller and companion CLI (`kubeseal`) that enables
encrypting Secret manifests so they can be safely stored in Git. The controller holds a
private key; `kubeseal` encrypts with the corresponding public key. Only the in-cluster
controller can decrypt.

**Why we need it:**
ArgoCD (Phase 4) deploys applications from Git. Applications need Secrets (database
credentials, API keys, S3 access keys). Storing plain Secrets in Git is a security
disaster. Sealed Secrets solves this: developers encrypt Secrets locally with `kubeseal`,
commit the `SealedSecret` resource to Git, and the controller decrypts it in-cluster.

**Namespace:** `sealed-secrets`

**Helm chart + version:** `bitnami-labs/sealed-secrets` 2.17.0

**Workflow:**

```
Developer workstation                 Git repository              Cluster
       |                                    |                        |
  kubeseal --cert <pub.pem>                 |                        |
  < secret.yaml > sealed.yaml              |                        |
       |                                    |                        |
       +---- git push sealed.yaml --------->|                        |
                                            |                        |
                                  ArgoCD syncs sealed.yaml --------->|
                                            |                        |
                                            |    SealedSecret controller
                                            |    decrypts -> creates Secret
                                            |                        |
                                            |    Pod reads Secret as env var
```

**CRITICAL: Back up the controller's private key.**

If the controller is destroyed and recreated, it generates a new key pair. All existing
SealedSecrets become permanently undecryptable. After Phase 3, immediately back up:

```bash
kubectl -n sealed-secrets get secret \
  -l sealedsecrets.bitnami.com/sealed-secrets-key \
  -o yaml > sealed-secrets-key-backup.yaml

# Store this in a SECURE location (password manager, encrypted vault)
# NOT in Git!
```

**How to verify:**

```bash
kubectl -n sealed-secrets get pods
# sealed-secrets-controller-<hash>    1/1     Running

# Fetch the public key
kubeseal --controller-namespace sealed-secrets --fetch-cert > pub-cert.pem

# Create a test sealed secret
kubectl create secret generic test-secret --dry-run=client -o yaml \
  --from-literal=password=mysecret | \
  kubeseal --cert pub-cert.pem -o yaml > sealed-test.yaml

kubectl apply -f sealed-test.yaml
kubectl get secret test-secret
# NAME          TYPE     DATA   AGE
# test-secret   Opaque   1      5s

# Clean up
kubectl delete -f sealed-test.yaml
```

---

### Pod Security Standards

**What it does:**
Pod Security Standards (PSS) are Kubernetes-native pod hardening profiles enforced via
namespace labels. They are not a separate component -- they are built into the Kubernetes
API server. Phase 3 configures PSS labels on all customer namespaces.

**Three profiles:**

| Profile | What it allows | Where used |
|---------|---------------|------------|
| **Privileged** | Everything (root, hostNetwork, hostPID) | Only `kube-system` (for Cilium, node-exporter) |
| **Baseline** | Some capabilities, non-root preferred | `monitoring`, `apisix`, `harbor` |
| **Restricted** | No root, no capabilities, read-only rootfs, seccomp | ALL customer namespaces (`zenith-customer-*`) |

Under `restricted`, customer pods CANNOT:
- Run as root (UID 0)
- Use `hostNetwork`, `hostPID`, `hostIPC`
- Mount `hostPath` volumes
- Use privileged containers
- Add Linux capabilities (no `NET_RAW`, no `SYS_ADMIN`)
- Run without a seccomp profile

---

## Component Deep Dives: Observability

### Prometheus + Grafana + Alertmanager

**What it does:**
The kube-prometheus-stack Helm chart installs three interrelated components:

- **Prometheus** scrapes `/metrics` endpoints from all pods, node-exporters, and
  kube-state-metrics. It stores time-series data with configurable retention (15 days
  staging, 90 days production).
- **Grafana** provides dashboards for visualizing metrics, logs, and traces in a single
  pane. It connects to Prometheus (metrics), Loki (logs), and Tempo (traces) as data
  sources.
- **Alertmanager** receives alerts from Prometheus alert rules and routes them to
  notification channels: Slack, PagerDuty, Telegram, or email.

**Why we need it (Decision D13):**
A platform without observability is flying blind. When a customer reports "my API is
slow," you need metrics (is CPU pegged?), logs (any errors?), and traces (which service
call is slow?). The monitoring stack answers all three questions from a single Grafana
dashboard.

**Namespace:** `monitoring`

**Helm chart + version:** `prometheus-community/kube-prometheus-stack` 68.4.0

**Key configuration:**

```hcl
resource "helm_release" "kube_prometheus" {
  count            = var.enable_monitoring ? 1 : 0
  name             = "kube-prometheus-stack"
  repository       = "https://prometheus-community.github.io/helm-charts"
  chart            = "kube-prometheus-stack"
  version          = var.kube_prometheus_version
  namespace        = "monitoring"
  create_namespace = true
  wait             = true
  timeout          = 600

  values = [templatefile("${path.module}/values/monitoring.yaml", {
    domain    = var.domain
    loki_url  = "http://loki:3100"
    tempo_url = "http://tempo:3100"
  })]

  set_sensitive {
    name  = "grafana.adminPassword"
    value = var.grafana_admin_password
  }
}
```

**How to verify:**

```bash
kubectl -n monitoring get pods
# kube-prometheus-stack-grafana-<hash>       3/3     Running
# kube-prometheus-stack-prometheus-<hash>    2/2     Running
# alertmanager-<hash>                        2/2     Running

# Access Grafana
kubectl -n monitoring port-forward svc/kube-prometheus-stack-grafana 3000:80
# Open http://localhost:3000 (admin / <grafana_admin_password>)
```

---

### Loki (Logs)

**What it does:**
Loki is a log aggregation system designed for Kubernetes. Unlike Elasticsearch, Loki does
not index log content -- it only indexes metadata labels (namespace, pod name, container
name). This makes it orders of magnitude cheaper to operate while still providing fast
log queries via LogQL.

**How it fits:**
Promtail (or Grafana Alloy) runs as a DaemonSet on every node, tailing container logs
from `/var/log/pods/`. It adds Kubernetes labels and ships logs to Loki. Grafana queries
Loki using LogQL. This is where Falco alerts, application logs, and Kubernetes audit logs
all converge.

**Namespace:** `monitoring`

**Helm chart + version:** `grafana/loki` 6.24.0

---

### Tempo (Traces)

**What it does:**
Tempo is a distributed tracing backend. It receives trace spans from the OpenTelemetry
Collector and stores them for querying in Grafana. Tempo enables you to follow a single
request as it travels through APISIX, the backend pod, the database, and back -- seeing
latency at each hop.

**How it fits:**
The APISIX `opentelemetry` plugin injects trace headers (`traceparent`) into every API
request. The Go backend SDK propagates these through internal calls. The OpenTelemetry
Collector receives spans via OTLP and forwards them to Tempo. Grafana queries Tempo by
trace ID, showing the full waterfall.

**Namespace:** `monitoring`

**Helm chart + version:** `grafana/tempo` 1.15.0

---

### OpenTelemetry Collector

**What it does:**
The OpenTelemetry Collector is a vendor-neutral telemetry pipeline. It receives traces
(OTLP/gRPC) and metrics (OTLP, Prometheus) from applications and infrastructure
components, processes them (batching, attribute enrichment), and exports them to
backend-specific systems (Tempo for traces, Prometheus for metrics).

**Deployment model:** DaemonSet (one per node). Applications send telemetry to
`localhost:4317` (OTLP/gRPC), which is the Collector running on the same node.

**How it connects:**

```
APISIX (opentelemetry plugin)  -->  OTel Collector  -->  Tempo (traces)
Go backend (OTel SDK)          -->  OTel Collector  -->  Prometheus (metrics)
```

**Namespace:** `monitoring`

**Helm chart + version:** `open-telemetry/opentelemetry-collector` 0.108.0

---

### Hubble (Network Flows)

**What it does:**
Hubble is Cilium's built-in network observability layer. It provides real-time visibility
into all network flows in the cluster: which pod talked to which pod, what DNS queries
were made, what was blocked by NetworkPolicy, and per-flow latency metrics.

**How it fits:**
Hubble is not a separate Helm release -- it is enabled as part of the Cilium installation
in Phase 2. However, Phase 3 configures Prometheus to scrape Hubble metrics and adds
Hubble dashboards to Grafana. This means network observability is unified with the rest
of the monitoring stack.

**Components:**
- **hubble-relay** (Deployment in `kube-system`): Aggregates flow data from all nodes
- **hubble-ui** (Deployment in `kube-system`): Web UI for network topology visualization
- **hubble-metrics**: Prometheus endpoints scraped by kube-prometheus-stack

**What you see in Grafana:**
- Network flow rate per namespace
- DNS query latency
- NetworkPolicy drop rate (how many packets were blocked)
- Service-to-service dependency map

---

### Alertmanager

**What it does:**
Alertmanager receives firing alerts from Prometheus alert rules and routes them to the
appropriate notification channels based on severity and labels.

**Alert routing configuration:**

| Severity | Channel | Example Alerts |
|----------|---------|---------------|
| `critical` | PagerDuty + Slack #incidents | Node down, CNPG primary failover, Keycloak unreachable |
| `warning` | Slack #alerts | High CPU (>80%), disk usage (>85%), certificate expiring in 7 days |
| `info` | Slack #monitoring | New customer provisioned, backup completed, image scanned |

Alertmanager is installed as part of the kube-prometheus-stack chart, not as a separate
Helm release.

---

## Component Deep Dives: Resilience

### Velero

**What it does:**
Velero performs cluster-level backups: all Kubernetes resources (Deployments, Services,
ConfigMaps, Secrets, CRDs, etc.) plus persistent volume snapshots. Backups are stored
in Hetzner S3 object storage (Decision D12: 3-layer backup strategy).

**Why we need it:**
CNPG handles database backups (WAL archiving + pg_dump). But the cluster has hundreds of
other resources: Ingress routes, Kyverno policies, APISIX etcd data, RBAC rules,
namespaces, CiliumNetworkPolicies, and more. Losing these would require hours of manual
recreation. Velero captures everything and enables full cluster restoration.

**Namespace:** `velero`

**Helm chart + version:** `vmware-tanzu/velero` 8.2.0

**Backup schedule:**
- Daily at 02:00 UTC
- 30-day retention
- Stored at `s3://zenith-backups/velero/`

**Key Terraform configuration:**

```hcl
resource "helm_release" "velero" {
  count = var.enable_velero ? 1 : 0

  name             = "velero"
  repository       = "https://vmware-tanzu.github.io/helm-charts"
  chart            = "velero"
  version          = var.velero_version
  namespace        = "velero"
  create_namespace = true
  wait             = true
  timeout          = 300

  set { name = "configuration.backupStorageLocation[0].provider"; value = "aws" }
  set { name = "configuration.backupStorageLocation[0].bucket"; value = "zenith-backups" }
  set { name = "configuration.backupStorageLocation[0].prefix"; value = "velero" }
  set { name = "configuration.backupStorageLocation[0].config.s3Url"
        value = var.s3_endpoint }

  set { name = "schedules.daily-backup.schedule"; value = "0 2 * * *" }
  set { name = "schedules.daily-backup.template.ttl"; value = "720h" }  # 30 days
}
```

**3-layer backup strategy (Decision D12):**

```
Layer 1: CNPG WAL archiving
  |-- Continuous, point-in-time recovery
  |-- Covers: PostgreSQL data
  |-- RPO: seconds (continuous WAL shipping)

Layer 2: pg_dump CronJobs
  |-- Hourly logical dumps per customer database
  |-- Covers: Individual customer data (granular restore)
  |-- RPO: 1 hour

Layer 3: Velero cluster backup
  |-- Daily full cluster snapshot
  |-- Covers: ALL Kubernetes resources + PV snapshots
  |-- RPO: 24 hours
```

**How to verify:**

```bash
kubectl -n velero get pods
# velero-<hash>    1/1     Running

velero backup-location get
# NAME      PROVIDER   BUCKET/PREFIX            PHASE       LAST VALIDATED
# default   aws        zenith-backups/velero     Available   2026-02-25

velero backup create test-backup --include-namespaces=default --wait
velero backup describe test-backup
# Phase: Completed
velero backup delete test-backup --confirm
```

---

### PriorityClasses

**What it does:**
PriorityClasses determine the eviction order when a node runs out of resources. Higher
priority pods are evicted last.

**The four tiers:**

```yaml
# Tier 1: System-critical (NEVER evict)
# Components: Cilium, CoreDNS, Traefik, kube-proxy
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: system-critical
value: 1000000
description: "System components that must never be evicted"

---
# Tier 2: Infrastructure-critical (evict last)
# Components: CNPG, Keycloak, APISIX, cert-manager
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: infra-critical
value: 500000
description: "Infrastructure services (database, identity, gateway)"

---
# Tier 3: Platform (evict second)
# Components: zenith-api, Temporal, Harbor, monitoring
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: platform
value: 100000
description: "Zenith platform services"

---
# Tier 4: Customer workloads (evict first)
# Components: all customer pods (frontend, backend)
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: customer-workload
value: 10000
globalDefault: true
description: "Customer workloads -- evicted first under resource pressure"
```

**Eviction order under node pressure:**

```
1. FIRST evicted:  customer-workload (10000)
   -> Customer pods restart. Service degradation but not catastrophic.

2. THEN evicted:   platform (100000)
   -> zenith-api, monitoring down. Admin notified via PagerDuty.

3. THEN evicted:   infra-critical (500000)
   -> Databases, identity, gateway down. CRITICAL incident.

4. NEVER evicted:  system-critical (1000000)
   -> Cilium, CoreDNS, Traefik stay running.
   -> Node can still route traffic and enforce network policies.
```

---

### PodDisruptionBudgets

**What it does:**
PodDisruptionBudgets (PDBs) prevent voluntary disruptions (node drains, rolling updates,
cluster autoscaler) from killing too many pods of a service simultaneously. They
guarantee minimum availability during maintenance.

**PDBs created by Phase 3:**

| Component | Namespace | minAvailable | Why |
|-----------|-----------|-------------|-----|
| CNPG keycloak-pg | `keycloak` | 1 | At least one PG instance must be running during drain |
| Keycloak | `keycloak` | 1 | Identity provider must remain available |
| APISIX etcd | `apisix` | 2 (production) | Raft quorum requires majority (2 of 3) |
| APISIX gateway | `apisix` | 1 | At least one gateway pod for routing |
| Prometheus | `monitoring` | 1 | Metrics collection must not stop |
| ArgoCD server | `argocd` | 1 | GitOps sync must continue |

**Example:**

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: keycloak-pdb
  namespace: keycloak
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: keycloak
```

Without PDBs, a `kubectl drain` during maintenance could kill all Keycloak replicas at
once, causing a cluster-wide authentication outage.

---

### ResourceQuotas and LimitRanges

**What it does:**
ResourceQuotas limit the total resources a namespace can consume. LimitRanges set
per-pod defaults and maximums. Together they prevent any single customer from starving
the cluster.

**ResourceQuota per tier:**

| Resource | Free | Pro | Team | Enterprise |
|----------|------|-----|------|------------|
| CPU requests | 2 | 8 | 32 | Custom |
| Memory requests | 2Gi | 16Gi | 64Gi | Custom |
| Pods | 10 | 50 | 200 | Custom |
| PVCs | 2 | 10 | 50 | Custom |
| Storage | 10Gi | 100Gi | 500Gi | Custom |

**LimitRange defaults:**

| Setting | Value | Why |
|---------|-------|-----|
| Default CPU limit | 250m | Prevent runaway CPU usage |
| Default memory limit | 256Mi | Prevent OOM from killing other pods |
| Max CPU per container | 2 | Single container cannot hog the node |
| Max memory per container | 2Gi | Single container cannot hog the node |
| Min CPU per container | 50m | Prevent zero-resource requests (unfair scheduling) |

ResourceQuotas and LimitRanges are not created by Phase 3 directly. They are applied by
the Temporal provisioning workflow when a customer namespace is created.

---

## Terraform Structure

Phase 3 uses a module architecture. The environment-specific root module
(`staging-k8s/` or `production-k8s/`) calls a shared platform module
(`modules/k8s-platform-v2/`) with environment-appropriate variables.

```
infra/terraform/
  |
  |-- modules/
  |     |-- k8s-platform-v2/               <-- The V2 platform module (this phase)
  |           |-- main.tf                   <-- All helm_release + kubernetes_manifest
  |           |-- variables.tf              <-- Input vars (versions, secrets, flags)
  |           |-- outputs.tf                <-- Status outputs per component
  |           |-- values/                   <-- Helm values templates
  |           |     |-- apisix.yaml
  |           |     |-- monitoring.yaml
  |           |     |-- keycloak.yaml
  |           |     |-- harbor.yaml
  |           |     |-- ...
  |           |-- rules/                    <-- Kyverno + Falco rule files
  |                 |-- kyverno-policies.yaml
  |                 |-- falco-zenith.yaml
  |
  |-- staging-k8s/
  |     |-- main.tf                         <-- Calls module "k8s-platform-v2"
  |     |-- variables.tf                    <-- Staging defaults
  |     |-- outputs.tf                      <-- Surfaces module outputs
  |     |-- terraform.tfvars                <-- Staging values (gitignored)
  |     |-- terraform.tfvars.example        <-- Template
  |
  |-- production-k8s/
        |-- main.tf                         <-- Same module, production values
        |-- variables.tf
        |-- outputs.tf
        |-- terraform.tfvars
```

**Terraform resource pattern -- every component follows this structure:**

```hcl
# =============================================================================
# <Component Name> -- <one-line purpose>
# =============================================================================

resource "helm_release" "<component>" {
  count = var.enable_<component> ? 1 : 0

  name             = "<release-name>"
  repository       = "<chart-repo-url>"
  chart            = "<chart-name>"
  version          = var.<component>_chart_version
  namespace        = "<namespace>"
  create_namespace = true
  wait             = true              # Block until all pods are Ready
  timeout          = 600               # 10 min max for large charts

  values = [templatefile("${path.module}/values/<component>.yaml", {
    environment = var.environment
    domain      = var.domain
  })]

  set_sensitive {
    name  = "some.secret"
    value = var.<component>_secret
  }

  depends_on = [helm_release.<dependency>]
}
```

**Why a single module (not one module per component)?**

These components have tight dependency chains. Splitting them into separate modules
would require passing many outputs between modules and would prevent Terraform from
optimizing the dependency graph for parallel execution. A single module with explicit
`depends_on` chains gives Terraform full visibility to apply independent components
(external-dns, Kyverno, Falco, Sealed Secrets) in parallel while respecting the
ordering constraints.

---

## Variables Reference

### Required Variables (Secrets)

| Variable | Type | Description |
|----------|------|-------------|
| `kubeconfig_path` | `string` | Path to the k3s kubeconfig file |
| `domain` | `string` | Base domain (e.g., `stage.freezenith.com`) |
| `cloudflare_api_token` | `string` (sensitive) | Cloudflare API token (DNS:Edit for cert-manager + external-dns) |
| `cert_issuer_email` | `string` | Email for Let's Encrypt registration |
| `keycloak_admin_password` | `string` (sensitive) | Keycloak admin console password |
| `harbor_admin_password` | `string` (sensitive) | Harbor admin password |
| `harbor_s3_access_key` | `string` (sensitive) | Hetzner S3 access key for Harbor storage |
| `harbor_s3_secret_key` | `string` (sensitive) | Hetzner S3 secret key for Harbor storage |
| `grafana_admin_password` | `string` (sensitive) | Grafana admin password |
| `argocd_admin_password` | `string` (sensitive) | ArgoCD admin password (bcrypt hash) |
| `velero_s3_access_key` | `string` (sensitive) | Hetzner S3 access key for Velero backups |
| `velero_s3_secret_key` | `string` (sensitive) | Hetzner S3 secret key for Velero backups |
| `github_token` | `string` (sensitive) | GitHub PAT for ArgoCD to access private repo |

### Feature Flags

Every component (except cert-manager and CNPG, which are always required) has a boolean
toggle. This allows incremental rollout and cost control in staging.

| Variable | Default | Description |
|----------|---------|-------------|
| `enable_keycloak` | `true` | Install Keycloak identity provider |
| `enable_apisix` | `true` | Install APISIX API gateway stack |
| `enable_external_dns` | `true` | Install external-dns |
| `enable_temporal` | `true` | Install Temporal workflow engine |
| `enable_harbor` | `true` | Install Harbor container registry |
| `enable_kyverno` | `true` | Install Kyverno policy engine |
| `enable_falco` | `true` | Install Falco runtime security |
| `enable_sealed_secrets` | `true` | Install Sealed Secrets controller |
| `enable_velero` | `true` | Install Velero cluster backup |
| `enable_monitoring` | `true` | Install full monitoring stack |
| `enable_argocd` | `true` | Install ArgoCD GitOps controller |
| `enable_shared_pg` | `true` | Create shared CNPG clusters |

### Chart Versions

Each component has a pinned chart version. Never use `latest` -- always pin to a tested
version and upgrade explicitly.

| Variable | Default | Chart |
|----------|---------|-------|
| `cert_manager_version` | `v1.17.2` | `jetstack/cert-manager` |
| `cnpg_version` | `0.23.0` | `cnpg/cloudnative-pg` |
| `keycloak_version` | `24.4.0` | `bitnami/keycloak` |
| `apisix_version` | `2.10.0` | `apisix/apisix` |
| `apisix_ingress_version` | `0.14.0` | `apisix/apisix-ingress-controller` |
| `external_dns_version` | `8.7.0` | `bitnami/external-dns` |
| `temporal_version` | `0.46.0` | `temporalio/temporal` |
| `harbor_version` | `1.16.0` | `harbor/harbor` |
| `kyverno_version` | `3.3.4` | `kyverno/kyverno` |
| `falco_version` | `4.15.0` | `falcosecurity/falco` |
| `sealed_secrets_version` | `2.17.0` | `bitnami-labs/sealed-secrets` |
| `velero_version` | `8.2.0` | `vmware-tanzu/velero` |
| `kube_prometheus_version` | `68.4.0` | `prometheus-community/kube-prometheus-stack` |
| `loki_version` | `6.24.0` | `grafana/loki` |
| `tempo_version` | `1.15.0` | `grafana/tempo` |
| `otel_collector_version` | `0.108.0` | `open-telemetry/opentelemetry-collector` |
| `argocd_version` | `7.8.0` | `argoproj/argo-cd` |

---

## How to Run

### Prerequisites

1. **Phase 1 + 2 completed**: A Hetzner server exists with k3s running
2. **Terraform >= 1.5** installed locally
3. **kubeconfig** available locally:
   ```bash
   scp ghasi:/etc/rancher/k3s/k3s.yaml ~/.kube/zenith-staging.yaml
   sed -i '' 's|127.0.0.1|<server-ip>|g' ~/.kube/zenith-staging.yaml
   ```
4. **Hetzner S3 credentials** for Harbor storage and Velero backups
5. **Cloudflare API token** with `Zone:DNS:Edit` for `freezenith.com` and `embermind.app`
6. **GitHub PAT** for ArgoCD to access the private Zenith repository

### Step-by-step

```bash
# 1. Navigate to the staging-k8s directory
cd infra/terraform/staging-k8s

# 2. Create your tfvars file
cp terraform.tfvars.example terraform.tfvars

# 3. Edit terraform.tfvars with your actual values
#    - Set all sensitive variables (passwords, API tokens, S3 keys)
#    - Review feature flags (disable components to save resources)

# 4. Initialize Terraform (downloads providers + chart repos)
terraform init

# 5. Review the plan (shows what will be created)
terraform plan

# 6. Apply (installs all components in dependency order)
terraform apply

# 7. Monitor progress in another terminal
export KUBECONFIG=~/.kube/zenith-staging.yaml
watch kubectl get pods -A
```

### Expected Apply Time

| Scenario | Time |
|----------|------|
| First apply (all components) | 15-25 minutes |
| Incremental apply (changed values) | 1-5 minutes |
| Destroy + recreate from scratch | 20-30 minutes |
| Single component toggle (e.g., disable Falco) | 30-60 seconds |

### terraform.tfvars.example

```hcl
# ============================================================
# Phase 3: Cluster Bootstrap Variables
# ============================================================
# Copy to terraform.tfvars and fill in values.
# This file is gitignored -- NEVER commit it.
# ============================================================

kubeconfig_path = "~/.kube/zenith-staging.yaml"
domain          = "stage.freezenith.com"
dns_zone        = "freezenith.com"
environment     = "staging"

# Cloudflare (shared by cert-manager + external-dns)
cloudflare_api_token = ""
cert_issuer_email    = "admin@freezenith.com"

# Keycloak
keycloak_admin_password = ""

# Harbor
harbor_admin_password = ""
harbor_s3_access_key  = ""
harbor_s3_secret_key  = ""
s3_endpoint           = "https://hel1.your-objectstorage.com"

# Grafana
grafana_admin_password = ""

# ArgoCD (password must be bcrypt hash)
argocd_admin_password = "$2a$12$..."
github_token          = "ghp_..."

# Velero
velero_s3_access_key = ""
velero_s3_secret_key = ""

# Feature flags (disable to save resources in staging)
enable_keycloak       = true
enable_apisix         = true
enable_external_dns   = true
enable_temporal       = true
enable_harbor         = true
enable_kyverno        = true
enable_falco          = false     # Disable in staging to save ~300MB RAM
enable_sealed_secrets = true
enable_velero         = false     # Disable if no S3 configured
enable_monitoring     = true
enable_argocd         = true
enable_shared_pg      = true
```

---

## Verification Checklist

After `terraform apply` completes, run through this checklist to verify every component:

```bash
# ============================================================
# Phase 3 Verification Playbook
# ============================================================

export KUBECONFIG=~/.kube/zenith-staging.yaml

echo "=== 1. Namespaces ==="
kubectl get namespaces | grep -E \
  "cert-manager|cnpg|keycloak|apisix|external-dns|temporal|harbor|kyverno|falco|sealed-secrets|velero|monitoring|argocd|zenith-shared"

echo "=== 2. All pods healthy (no CrashLoopBackOff) ==="
kubectl get pods -A | grep -v Running | grep -v Completed | grep -v NAME

echo "=== 3. cert-manager: ClusterIssuer Ready ==="
kubectl get clusterissuer letsencrypt-prod -o jsonpath='{.status.conditions[0].status}'
# Expected: True

echo "=== 4. CNPG: Operator running ==="
kubectl -n cnpg-system get pods -l app.kubernetes.io/name=cloudnative-pg

echo "=== 5. Keycloak: PG cluster + app healthy ==="
kubectl -n keycloak get cluster keycloak-pg
kubectl -n keycloak get pods -l app.kubernetes.io/name=keycloak

echo "=== 6. APISIX: etcd + gateway + ingress controller ==="
kubectl -n apisix get pods

echo "=== 7. external-dns: No errors in logs ==="
kubectl -n external-dns logs -l app.kubernetes.io/name=external-dns --tail=5

echo "=== 8. Temporal: Server + UI running ==="
kubectl -n temporal get pods

echo "=== 9. Harbor: All components ==="
kubectl -n harbor get pods

echo "=== 10. Kyverno: Policies installed ==="
kubectl get clusterpolicy

echo "=== 11. Falco: DaemonSet running ==="
kubectl -n falco get daemonset 2>/dev/null || echo "Falco disabled"

echo "=== 12. Sealed Secrets: Controller running ==="
kubectl -n sealed-secrets get pods

echo "=== 13. Velero: Backup location available ==="
velero backup-location get 2>/dev/null || echo "Velero disabled or CLI not installed"

echo "=== 14. Monitoring: Prometheus + Grafana + Loki + Tempo ==="
kubectl -n monitoring get pods

echo "=== 15. ArgoCD: Server running, root app synced ==="
kubectl -n argocd get pods
kubectl -n argocd get application zenith-apps -o jsonpath='{.status.sync.status}' 2>/dev/null
# Expected: Synced

echo "=== 16. Shared PG: Clusters healthy ==="
kubectl -n zenith-shared get cluster 2>/dev/null || echo "Shared PG not yet created"

echo ""
echo "Phase 3 verification complete."
```

---

## Troubleshooting

### cert-manager: Certificate stuck in "Issuing"

```bash
# Check the CertificateRequest
kubectl describe certificaterequest -A

# Check the Challenge (DNS-01)
kubectl describe challenge -A

# Common causes:
# 1. Cloudflare API token missing or wrong permissions
kubectl -n cert-manager get secret cloudflare-api-token
kubectl -n cert-manager logs -l app.kubernetes.io/name=cert-manager --tail=50

# 2. Wrong zone ID or domain filter in ClusterIssuer
kubectl get clusterissuer letsencrypt-prod -o yaml
```

### CNPG: Cluster not becoming Ready

```bash
# Check operator logs
kubectl -n cnpg-system logs -l app.kubernetes.io/name=cloudnative-pg --tail=50

# Check the Cluster status
kubectl describe cluster <name> -n <namespace>

# Common causes:
# 1. StorageClass "hcloud-volumes" not available
kubectl get storageclass

# 2. Insufficient disk space for Hetzner Volume
kubectl get pvc -n <namespace>
```

### APISIX: 502 Bad Gateway

```bash
# Check etcd health
kubectl -n apisix exec apisix-etcd-0 -- etcdctl endpoint health

# Check APISIX error logs
kubectl -n apisix logs -l app.kubernetes.io/name=apisix --tail=50

# Common cause: etcd not ready when APISIX started
kubectl -n apisix rollout restart deployment apisix-gateway
```

### Helm release failed: timed out waiting for condition

```bash
# Find which pod is not ready
kubectl -n <namespace> get pods
kubectl -n <namespace> describe pod <pod-name>
kubectl -n <namespace> logs <pod-name>

# Common cause: insufficient cluster resources
kubectl describe node | grep -A 5 "Allocated resources"

# Fix: increase node size or reduce replica counts in variables
```

### Keycloak: CrashLoopBackOff

```bash
# Usually means the PG cluster is not ready
kubectl -n keycloak get cluster keycloak-pg
kubectl -n keycloak logs keycloak-0 --tail=50

# If PG is ready but Keycloak keeps crashing, check the DB connection:
kubectl -n keycloak exec -it keycloak-pg-1 -- psql -U postgres -l
```

### ArgoCD: Application stuck in "Unknown" or "Degraded"

```bash
# Check if ArgoCD can reach the Git repository
kubectl -n argocd logs -l app.kubernetes.io/name=argocd-repo-server --tail=20

# Common causes:
# 1. GitHub token expired or invalid
# 2. Git branch does not exist (check targetRevision)
# 3. Path in Git repo does not contain valid manifests
```

### Terraform state drift

```bash
# If someone manually changed a resource, Terraform detects drift
terraform plan
# Shows: "~ update in-place" or "- destroy"

# Import a manually-created resource
terraform import 'module.platform.helm_release.argocd[0]' argocd/argocd

# Force refresh state
terraform refresh
```

### Not enough resources on the node

The full Phase 3 stack requests approximately 4 GB RAM and 2 vCPU. If your staging
server is a cx22 (2 vCPU / 4 GB), you will run out of resources. Solutions:

1. **Upgrade to cx32** (4 vCPU / 8 GB) -- recommended for staging with full stack
2. **Disable non-essential components** -- set `enable_falco = false`,
   `enable_velero = false`, `enable_harbor = false` to save approximately 1.5 GB
3. **Reduce replica counts** -- staging values should use `replicaCount: 1` everywhere

---

## Architecture Decision References

This phase implements or depends on the following architecture decisions documented in
the overview:

| Decision | Title | How Phase 3 Implements It |
|----------|-------|--------------------------|
| **D1** | APISIX not Kong | APISIX + etcd Helm release replaces Kong |
| **D2** | CNPG sharding strategy | Free-users PG + Pro shard PG clusters created |
| **D3** | Separate PG for Keycloak | Dedicated keycloak-pg CNPG Cluster in `keycloak` namespace |
| **D4** | ArgoCD not FluxCD | ArgoCD Helm release + root Application (App-of-Apps) |
| **D5** | Temporal for provisioning | Temporal Helm release for workflow engine |
| **D9** | Single APISIX, route-level plugins | One gateway, per-route plugin config (protected vs public routes) |
| **D10** | Terraform for infra, ArgoCD for apps | ALL infra via `helm_release`, apps via ArgoCD sync |
| **D11** | Defense-in-depth security | Kyverno + Falco + Sealed Secrets + PSS + Cilium |
| **D12** | 3-layer backup strategy | Velero (cluster) + CNPG WAL (database) + pg_dump (customer) |
| **D13** | Full-stack observability | Prometheus + Grafana + Loki + Tempo + OTel + Hubble + Alertmanager |

---

## What Happens Next (Phase 4)

After Phase 3 completes, you have a fully operational platform with:

- **TLS everywhere** -- cert-manager with DNS-01 (Cloudflare proxy stays ON)
- **Managed PostgreSQL** -- CNPG Operator with 3+ clusters (Keycloak, Free, Pro shard)
- **Identity management** -- Keycloak with per-customer realm isolation
- **API gateway** -- APISIX with JWT verification, per-customer CORS, rate limiting
- **Automatic DNS** -- external-dns creates Cloudflare records from Ingress annotations
- **Provisioning engine** -- Temporal ready to run customer signup workflows
- **Container registry** -- Harbor with Trivy scanning and S3 backend
- **Policy enforcement** -- Kyverno admission policies (11 ClusterPolicies)
- **Runtime security** -- Falco eBPF-based anomaly detection
- **GitOps secrets** -- Sealed Secrets for encrypted secrets in Git
- **Cluster backup** -- Velero daily backups to Hetzner S3
- **Full observability** -- Prometheus + Grafana + Loki + Tempo + OTel + Hubble + Alertmanager
- **GitOps deployment** -- ArgoCD with App-of-Apps pattern, watching the Git repo

**Phase 4 is automatic.** ArgoCD is already running and watching the Git repository. The
moment you push application manifests to the configured branch (`staging` or `main`),
ArgoCD detects the change and deploys the applications. No manual steps required.

Total Terraform resources: approximately 30 `helm_release` + 10 `kubernetes_manifest`.
Total cluster resource usage: approximately 4 GB RAM, 2 vCPU (staging).

**Previous:** [Phase 2: Ansible + k3s](./02-phase2-ansible-k3s.md)
**Next:** [Phase 4: ArgoCD Applications](./04-phase4-argocd-apps.md)
