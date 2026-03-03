# Zenith V2 — System Map

> **Purpose:** One document to understand the entire Zenith platform — every component, every connection, every flow.
> **Audience:** Any developer (junior to senior) who needs to understand how the system works as a whole.
> **Last Updated:** 2026-03-03
> **Start here:** If you're new to the project, read this document first, then follow the reading order below.

---

## Table of Contents

0. [How to Read This Documentation](#0-how-to-read-this-documentation)
1. [Master System Diagram](#1-master-system-diagram)
2. [Communication Table](#2-communication-table)
3. [Request Flow Paths](#3-request-flow-paths)
4. [Namespace Map](#4-namespace-map)
5. [DNS Map](#5-dns-map)
6. [PriorityClass & Eviction Order](#6-priorityclass--eviction-order)
7. [Storage Map](#7-storage-map)
8. [Key Architectural Decisions](#key-architectural-decisions)

---

## 0. How to Read This Documentation

### Prerequisites

Before working on Zenith, you should be comfortable with:

| Concept | Level Needed | Where to Learn |
|---------|-------------|----------------|
| Kubernetes basics (Pods, Deployments, Services, Namespaces) | Intermediate | [kubernetes.io/docs](https://kubernetes.io/docs/tutorials/) |
| Helm charts | Basic | [helm.sh/docs](https://helm.sh/docs/intro/quickstart/) |
| Terraform | Basic | [developer.hashicorp.com/terraform](https://developer.hashicorp.com/terraform/tutorials) |
| Docker & container images | Intermediate | [docs.docker.com](https://docs.docker.com/get-started/) |
| Git branching & PRs | Comfortable | Your team's workflow |
| YAML syntax | Comfortable | Used everywhere in K8s |

You do NOT need prior experience with: APISIX, Cilium, Temporal, ArgoCD, CNPG, Keycloak, Kyverno, Falco, or Sealed Secrets. These docs teach you each tool from scratch.

### Recommended Reading Order

```
For a NEW developer joining the team:

  DAY 1 — UNDERSTAND THE SYSTEM:
  1. SYSTEM-MAP.md (this file)       ← Big picture, all components, all connections
  2. 00-overview.md                   ← Architecture decisions, tier model, deployment phases
  3. 21-local-development-setup      ← Set up your laptop, run everything locally

  DAY 2 — LEARN THE STACK:
  4. 23-frontend-architecture         ← How the 3 Next.js apps work
  5. 10-backend-architecture          ← How the Go backend code is structured
  6. 22-day-to-day-operations         ← How to add features, deploy, check logs
  7. 24-ci-cd-pipeline                ← How code goes from git push to cluster

  WEEK 1 — LEARN THE INFRASTRUCTURE:
  8. 12-traefik-ingress               ← How traffic enters the cluster
  9. 13-apisix-gateway                ← How API requests are authenticated and routed
  10. 15-argocd-gitops                ← How GitOps deployments work
  11. 16-data-storage                 ← How databases and S3 work
  12. 26-keycloak-administration      ← How identity / auth works

  WEEK 2 — DEEPEN YOUR KNOWLEDGE:
  13. 14-cilium-networking            ← How pods communicate securely
  14. 17-temporal-workflows           ← How customer provisioning works
  15. 11-infrastructure-provisioning  ← How the platform is built from scratch
  16. 18-kyverno-policies             ← How security policies are enforced
  17. 19-velero-backup                ← How the cluster is backed up
  18. 20-sealed-secrets               ← How secrets are stored in Git safely
  19. 25-monitoring-runbook           ← What to do when alerts fire

For DEBUGGING a specific issue:
  → Find the component in the Master System Diagram below
  → Jump to that component's dedicated doc (12-26)
  → Check the Troubleshooting section at the bottom of each doc

For DEPLOYMENT steps:
  → 03-phase3-cluster-bootstrap.md has Helm values and install order
  → Each component doc (12-20) has architecture + troubleshooting
  → They complement each other — Phase 3 = "how to install", component docs = "how it works"
```

### Document Relationships

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    DOCUMENTATION MAP                                     │
│                                                                          │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐               │
│  │ SYSTEM-MAP   │───▶│ 00-overview  │───▶│ Phase docs   │               │
│  │ (big picture)│    │ (decisions)  │    │ (01-04)      │               │
│  └──────┬───────┘    └──────────────┘    │ (how to      │               │
│         │                                 │  install)    │               │
│         │                                 └──────┬───────┘               │
│         ▼                                        │                       │
│  ┌──────────────────────────────────────────────────────────────┐       │
│  │            PER-COMPONENT DOCS (11-20)                        │       │
│  │            (how each component works, troubleshooting)       │       │
│  │                                                              │       │
│  │  11-infra  12-traefik  13-apisix  14-cilium  15-argocd     │       │
│  │  16-data   17-temporal 18-kyverno 19-velero  20-sealed     │       │
│  └──────────────────────────────────────────────────────────────┘       │
│         │                                                               │
│         ▼                                                               │
│  ┌──────────────────────────────────────────────────────────────┐       │
│  │            CROSS-CUTTING DOCS                                │       │
│  │                                                              │       │
│  │  06-security   07-backup   08-observability  10-backend     │       │
│  │  (threat model) (DR plan)   (dashboards)      (Go code)     │       │
│  └──────────────────────────────────────────────────────────────┘       │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 1. Master System Diagram

This diagram shows every component in the Zenith platform, organized by layer. Arrows show the direction of communication.

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                                    INTERNET                                         │
│                           (Users, GitHub, DNS queries)                               │
└──────────────────────────────────────┬──────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                          CLOUDFLARE  (Layer 0)                                      │
│                                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌────────────┐  ┌─────────────┐  ┌──────────────────┐ │
│  │ DDoS     │  │ WAF      │  │ CDN Cache  │  │ DNS Zones   │  │ SSL/TLS Proxy    │ │
│  │ Shield   │  │ Rules    │  │ (static)   │  │ freezenith  │  │ (Full Strict)    │ │
│  └──────────┘  └──────────┘  └────────────┘  │ .com        │  └──────────────────┘ │
│                                               └─────────────┘                       │
└──────────────────────────────────────┬──────────────────────────────────────────────┘
                                       │ HTTPS (443)
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                     HETZNER CLOUD VM  (zen-stage / zen-prod)                        │
│                     Ubuntu 25.04, k3s v1.34.3                                       │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                    NETWORKING LAYER  (Layer 1)                                 │  │
│  │                                                                               │  │
│  │  ┌─────────────────────────────────────────────────────────────────────────┐  │  │
│  │  │                    TRAEFIK  (kube-system)                               │  │  │
│  │  │              TLS termination · L7 routing · IngressRoute CRDs          │  │  │
│  │  │         Entrypoints: :80 (web) → redirect → :443 (websecure)           │  │  │
│  │  └──────────────┬──────────────────────────────────┬───────────────────────┘  │  │
│  │                 │ Frontend routes                  │ API routes               │  │
│  │                 │ (direct to pods)                 │ (via ExternalName svc)   │  │
│  │                 ▼                                  ▼                          │  │
│  │  ┌──────────────────────┐   ┌─────────────────────────────────────────────┐  │  │
│  │  │ Landing (3000)       │   │          APISIX  (apisix ns)                │  │  │
│  │  │ Web Dashboard (3000) │   │  Gateway: ClusterIP :9080 (HTTP)            │  │  │
│  │  │ ArgoCD UI (80)       │   │  Admin API: :9180                           │  │  │
│  │  │ Grafana (3000)       │   │  Plugins: jwt-auth, cors, limit-count,     │  │  │
│  │  │ Temporal UI (8080)   │   │           openid-connect, opentelemetry,    │  │  │
│  │  │ Hubble UI (80)       │   │           prometheus                        │  │  │
│  │  │ Harbor UI (80)       │   │  Config store: etcd (5Gi PVC)               │  │  │
│  │  │ Customer apps (3000) │   └───────────────────┬─────────────────────────┘  │  │
│  │  └──────────────────────┘                       │                            │  │
│  │                                                 ▼                            │  │
│  │  ┌─────────────────────────────────────────────────────────────────────────┐  │  │
│  │  │                    CILIUM  (kube-system)                                │  │  │
│  │  │  CNI · kube-proxy replacement · WireGuard pod-to-pod encryption        │  │  │
│  │  │  CiliumNetworkPolicy (L3-L7) · Hubble flow observability              │  │  │
│  │  └─────────────────────────────────────────────────────────────────────────┘  │  │
│  │                                                                               │  │
│  │  ┌────────────────────────┐  ┌────────────────────────────────────────────┐   │  │
│  │  │ external-dns           │  │ cert-manager (cert-manager ns)             │   │  │
│  │  │ (external-dns ns)      │  │ ClusterIssuer: letsencrypt-prod            │   │  │
│  │  │ Cloudflare sync        │  │ DNS-01 solver via Cloudflare API           │   │  │
│  │  │ Sources: svc, ingress, │  │ Auto-provisions TLS certs for all          │   │  │
│  │  │   traefik-proxy        │  │ IngressRoutes + Certificates               │   │  │
│  │  └────────────────────────┘  └────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                    IDENTITY & SECURITY LAYER  (Layer 2)                        │  │
│  │                                                                               │  │
│  │  ┌────────────────────┐  ┌─────────────────┐  ┌──────────────────────────┐   │  │
│  │  │ Keycloak           │  │ Kyverno          │  │ Falco                    │   │  │
│  │  │ (keycloak ns)      │  │ (kyverno ns)     │  │ (falco ns)               │   │  │
│  │  │ Realm per customer │  │ Admission policy │  │ Runtime security         │   │  │
│  │  │ OIDC/JWT issuer    │  │ Block unsigned   │  │ eBPF syscall monitoring  │   │  │
│  │  │ DB: keycloak-pg    │  │ images, enforce  │  │ Falcosidekick for        │   │  │
│  │  │ Port: 8080         │  │ labels, limits   │  │ alert forwarding         │   │  │
│  │  └────────────────────┘  └─────────────────┘  └──────────────────────────┘   │  │
│  │                                                                               │  │
│  │  ┌────────────────────┐  ┌───────────────────────────────────────────────┐    │  │
│  │  │ Sealed Secrets     │  │ Pod Security Standards                        │    │  │
│  │  │ (sealed-secrets ns)│  │ restricted: customer namespaces               │    │  │
│  │  │ Decrypts           │  │ baseline: platform namespaces                 │    │  │
│  │  │ SealedSecrets →    │  │ k3s --secrets-encryption (etcd at rest)       │    │  │
│  │  │ Secrets in-cluster │  └───────────────────────────────────────────────┘    │  │
│  │  └────────────────────┘                                                       │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                    DATA LAYER  (Layer 3)                                       │  │
│  │                                                                               │  │
│  │  ┌──────────────────────────────────────────────────────────────────────────┐ │  │
│  │  │                 CNPG Operator  (cnpg-system ns)                          │ │  │
│  │  │          Watches all namespaces for Cluster CRDs                         │ │  │
│  │  └──────────┬───────────────────────────────────┬──────────────────────────┘ │  │
│  │             │                                   │                            │  │
│  │             ▼                                   ▼                            │  │
│  │  ┌─────────────────────┐          ┌──────────────────────────────────────┐   │  │
│  │  │ keycloak-pg          │          │ free-pg (zenith-shared ns)           │   │  │
│  │  │ (keycloak ns)        │          │ DB: zenith_platform (API's own DB)  │   │  │
│  │  │ 2 instances (stg)    │          │ DB: temporal, temporal_visibility   │   │  │
│  │  │ 3 instances (prod)   │          │ DB: customer_xxx (one per free user)│   │  │
│  │  │ DB: keycloak         │          │ 2 instances (stg) / 3 (prod)        │   │  │
│  │  │ WAL → S3             │          │ WAL → S3                            │   │  │
│  │  └─────────────────────┘          └──────────────────────────────────────┘   │  │
│  │                                                                               │  │
│  │  ┌──────────────────────────────┐  ┌──────────────────────────────────────┐  │  │
│  │  │ Hetzner CSI (kube-system)    │  │ Hetzner S3 (external)                │  │  │
│  │  │ StorageClass: hcloud-volumes │  │ Bucket: zenith-backups               │  │  │
│  │  │ ReclaimPolicy: Retain        │  │ Bucket: zenith-harbor                │  │  │
│  │  │ Provides PVs for PG, Loki,  │  │ Bucket: customer-xxx-data (per user) │  │  │
│  │  │ Tempo, Prometheus, etcd      │  │ Region: fsn1                         │  │  │
│  │  └──────────────────────────────┘  └──────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                    PLATFORM LAYER  (Layer 4)                                   │  │
│  │                                                                               │  │
│  │  ┌────────────────────┐  ┌────────────────────┐  ┌────────────────────────┐  │  │
│  │  │ zenith-api         │  │ zenith-web         │  │ zenith-landing         │  │  │
│  │  │ (zenith-staging)   │  │ (zenith-staging)   │  │ (zenith-staging)       │  │  │
│  │  │ Go backend :8080   │  │ Next.js :3000      │  │ Next.js :3000          │  │  │
│  │  │ Metrics :9090      │  │ Web dashboard      │  │ Marketing site         │  │  │
│  │  └────────────────────┘  └────────────────────┘  └────────────────────────┘  │  │
│  │                                                                               │  │
│  │  ┌────────────────────┐  ┌────────────────────┐  ┌────────────────────────┐  │  │
│  │  │ Temporal           │  │ ArgoCD             │  │ Harbor (customer)      │  │  │
│  │  │ (temporal ns)      │  │ (argocd ns)        │  │ (harbor ns)            │  │  │
│  │  │ Frontend :7233     │  │ Server :80 (UI)    │  │ Core :80 (API/UI)      │  │  │
│  │  │ Web UI :8080       │  │ Repo Server        │  │ Registry :5000         │  │  │
│  │  │ History, Matching  │  │ App Controller     │  │ Trivy scanner          │  │  │
│  │  │ Worker             │  │ Image Updater      │  │ S3 backend (zenith-    │  │  │
│  │  │ DB: free-pg        │  │ Repo: GitHub       │  │ harbor bucket)         │  │  │
│  │  └────────────────────┘  └────────────────────┘  └────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                    OBSERVABILITY LAYER  (Layer 5)                              │  │
│  │                    All in "monitoring" namespace                               │  │
│  │                                                                               │  │
│  │  ┌───────────┐ ┌─────────┐ ┌──────┐ ┌───────┐ ┌──────────────┐ ┌──────────┐ │  │
│  │  │Prometheus │ │Grafana  │ │Loki  │ │Tempo  │ │OTel Collector│ │Alertmgr  │ │  │
│  │  │Metrics    │ │Dashbrd  │ │Logs  │ │Traces │ │(DaemonSet)   │ │Alerts    │ │  │
│  │  │20Gi PVC   │ │UI :3000 │ │10Gi  │ │10Gi   │ │OTLP :4317    │ │:9093     │ │  │
│  │  │Ret: 15d   │ │         │ │TSDB  │ │PVC    │ │HTTP :4318    │ │          │ │  │
│  │  │stg/90d pr │ │         │ │      │ │       │ │→ Tempo       │ │          │ │  │
│  │  └───────────┘ └─────────┘ └──────┘ └───────┘ └──────────────┘ └──────────┘ │  │
│  │                                                                               │  │
│  │  ┌──────────────────────────────────────────────────────────────────────────┐ │  │
│  │  │ Hubble (kube-system) — Cilium's built-in flow observability             │ │  │
│  │  │ UI: hubble.stage.freezenith.com · Metrics exported to Prometheus        │ │  │
│  │  └──────────────────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                    RESILIENCE LAYER  (Layer 6)                                 │  │
│  │                                                                               │  │
│  │  ┌────────────────────┐  ┌─────────────────────────────────────────────────┐  │  │
│  │  │ Velero             │  │ CNPG WAL Archiving                              │  │  │
│  │  │ (velero ns)        │  │ keycloak-pg → s3://zenith-backups/keycloak-wal/ │  │  │
│  │  │ Daily 03:00 UTC    │  │ free-pg → s3://zenith-backups/free-pg-wal/      │  │  │
│  │  │ 30-day retention   │  │ Continuous + daily base backup at 02:00 UTC     │  │  │
│  │  │ S3: zenith-backups │  │ 14-day WAL retention                            │  │  │
│  │  │ /velero prefix     │  │ Prefer-standby target for minimal prod impact   │  │  │
│  │  └────────────────────┘  └─────────────────────────────────────────────────┘  │  │
│  │                                                                               │  │
│  │  ┌───────────────────────────┐  ┌─────────────────────────────────────────┐   │  │
│  │  │ PriorityClasses           │  │ PodDisruptionBudgets                    │   │  │
│  │  │ core-critical: 1000000    │  │ minAvailable: 1 for:                    │   │  │
│  │  │ infra-critical: 500000    │  │ APISIX, ArgoCD, CNPG, Prometheus,      │   │  │
│  │  │ platform:       100000    │  │ Grafana, Alertmanager, Loki,            │   │  │
│  │  │ customer:        10000    │  │ Temporal, Harbor, cert-manager          │   │  │
│  │  │ (default)                 │  │                                         │   │  │
│  │  └───────────────────────────┘  └─────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                    CUSTOMER WORKLOADS                                          │  │
│  │                                                                               │  │
│  │  ┌─────────────────────────────────────────────────────────────────────────┐  │  │
│  │  │ zenith-apps namespace (shared for Free/Pro)                             │  │  │
│  │  │                                                                         │  │  │
│  │  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌──────────────────┐  │  │  │
│  │  │  │ customer-a │  │ customer-b │  │ customer-c │  │ cold-start-page  │  │  │  │
│  │  │  │ Deployment │  │ Deployment │  │ Deployment │  │ (nginx splash)   │  │  │  │
│  │  │  │ + Service  │  │ + Service  │  │ + Service  │  │ KEDA redirects   │  │  │  │
│  │  │  │ + Ingress  │  │ + Ingress  │  │ + Ingress  │  │ here when scaled │  │  │  │
│  │  │  └────────────┘  └────────────┘  └────────────┘  │ to zero          │  │  │  │
│  │  │                                                   └──────────────────┘  │  │  │
│  │  └─────────────────────────────────────────────────────────────────────────┘  │  │
│  │                                                                               │  │
│  │  ┌─────────────────────────────────────────────────────────────────────────┐  │  │
│  │  │ zenith-builds namespace (CI/CD builds)                                  │  │  │
│  │  │                                                                         │  │  │
│  │  │  ┌─────────────────────────────────────────────────────────────────┐    │  │  │
│  │  │  │ Kaniko Jobs (ephemeral)                                         │    │  │  │
│  │  │  │ GitHub webhook → zenith-api → Kaniko build → push to Harbor     │    │  │  │
│  │  │  └─────────────────────────────────────────────────────────────────┘    │  │  │
│  │  └─────────────────────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────────┘

EXTERNAL SERVICES (not in cluster):
┌────────────────────────────┐  ┌──────────────────────────────┐
│ Internal Harbor             │  │ GitHub                        │
│ registry.stage.freezenith   │  │ taikuri-infra/Zenith repo     │
│ .com                        │  │ Webhooks → zenith-api         │
│ Platform images + Helm      │  │ ArgoCD watches staging branch │
│ Managed on separate server  │  │                               │
└────────────────────────────┘  └──────────────────────────────┘
```

---

## 2. Communication Table

Every connection in the system. Use this to debug connectivity issues or configure firewalls.

### External → Cluster

| Source | Destination | Protocol | Port | Auth | Purpose |
|--------|-------------|----------|------|------|---------|
| Cloudflare | Traefik | HTTPS | 443 | TLS cert | All external traffic entry |
| Cloudflare | Traefik | HTTP | 80 | none | Redirect to HTTPS |
| GitHub | zenith-api | HTTPS | 443 (via Traefik) | Webhook secret | Git push notifications |
| GitHub | ArgoCD | HTTPS | 443 | PAT token | Git repo polling |
| cert-manager | Let's Encrypt | HTTPS | 443 | ACME account | Certificate issuance |
| external-dns | Cloudflare API | HTTPS | 443 | API token | DNS record management |
| Velero | Hetzner S3 | HTTPS | 443 | Access key | Backup storage |
| CNPG | Hetzner S3 | HTTPS | 443 | Access key | WAL archiving |
| Harbor | Hetzner S3 | HTTPS | 443 | Access key | Image layer storage |
| ArgoCD Image Updater | Internal Harbor | HTTPS | 443 | Robot token | Check for new image tags |

### Cluster Internal — Networking Layer

| Source | Destination | Protocol | Port | Auth | Purpose |
|--------|-------------|----------|------|------|---------|
| Traefik | APISIX gateway | HTTP | 9080 | none (internal) | Forward API requests |
| Traefik | zenith-landing | HTTP | 3000 | none | Landing page |
| Traefik | zenith-web | HTTP | 3000 | none | Web dashboard |
| Traefik | ArgoCD server | HTTP | 80 | none | ArgoCD UI |
| Traefik | Grafana | HTTP | 3000 | none | Monitoring UI |
| Traefik | Temporal Web | HTTP | 8080 | none | Temporal UI |
| Traefik | Hubble UI | HTTP | 80 | none | Network flows UI |
| Traefik | Harbor core | HTTP | 80 | none | Registry UI/API |
| Traefik | Customer apps | HTTP | 3000 | none | Customer frontends |
| APISIX | zenith-api | HTTP | 8080 | JWT header | API backend calls |
| APISIX | etcd | gRPC | 2379 | none (internal) | Route/plugin config store |
| APISIX ingress ctrl | APISIX admin | HTTP | 9180 | Admin API key | Sync CRD → routes |

### Cluster Internal — Identity Layer

| Source | Destination | Protocol | Port | Auth | Purpose |
|--------|-------------|----------|------|------|---------|
| APISIX | Keycloak | HTTP | 8080 | none | JWKS endpoint for JWT verification |
| zenith-api | Keycloak | HTTP | 8080 | Admin credentials | Realm/client management |
| Keycloak | keycloak-pg-rw | TCP | 5432 | Password | Keycloak database |
| Kyverno | K8s API | HTTPS | 6443 | ServiceAccount | Admission webhook |
| Sealed Secrets | K8s API | HTTPS | 6443 | ServiceAccount | Decrypt SealedSecrets |

### Cluster Internal — Data Layer

| Source | Destination | Protocol | Port | Auth | Purpose |
|--------|-------------|----------|------|------|---------|
| zenith-api | free-pg-rw | TCP | 5432 | Password | Platform DB + customer DBs |
| Temporal | free-pg-rw | TCP | 5432 | Password | temporal + temporal_visibility DBs |
| CNPG operator | keycloak-pg | TCP | 5432 | Operator SA | Health checks, failover |
| CNPG operator | free-pg | TCP | 5432 | Operator SA | Health checks, failover |
| Hetzner CSI | Hetzner API | HTTPS | 443 | API token | Volume provisioning |

### Cluster Internal — Platform Layer

| Source | Destination | Protocol | Port | Auth | Purpose |
|--------|-------------|----------|------|------|---------|
| zenith-api | Temporal frontend | gRPC | 7233 | none (internal) | Start/query workflows |
| zenith-api | K8s API | HTTPS | 6443 | ServiceAccount | Create namespaces, secrets, deployments |
| zenith-api | Hetzner S3 | HTTPS | 443 | Access key | Create customer buckets |
| ArgoCD | K8s API | HTTPS | 6443 | ServiceAccount | Deploy/sync applications |
| ArgoCD | GitHub | HTTPS | 443 | PAT | Fetch manifests |
| Harbor | K8s API | HTTPS | 6443 | ServiceAccount | Registry operations |

### Cluster Internal — Observability Layer

| Source | Destination | Protocol | Port | Auth | Purpose |
|--------|-------------|----------|------|------|---------|
| Prometheus | All pods | HTTP | various | none | Scrape /metrics endpoints |
| Prometheus | zenith-api | HTTP | 9090 | none | API metrics |
| Prometheus | APISIX | HTTP | 9091 | none | Gateway metrics |
| OTel Collector | Tempo | gRPC | 4317 | none | Forward traces |
| zenith-api | OTel Collector | gRPC | 4317 | none | Send traces |
| APISIX | OTel Collector | gRPC | 4317 | none | Send traces |
| Grafana | Prometheus | HTTP | 9090 | none | Query metrics |
| Grafana | Loki | HTTP | 3100 | none | Query logs |
| Grafana | Tempo | HTTP | 3200 | none | Query traces |
| Falco | K8s API | HTTPS | 6443 | ServiceAccount | Audit events |

---

## 3. Request Flow Paths

### Flow 1: User Visits Landing Page

```
User Browser
    │
    │  GET https://stage.freezenith.com
    ▼
┌──────────┐
│Cloudflare│  DNS resolves stage.freezenith.com → 77.42.88.149
│          │  DDoS check, WAF rules, CDN cache check
└────┬─────┘
     │  HTTPS :443
     ▼
┌──────────┐
│ Traefik  │  TLS termination (cert from cert-manager)
│          │  Match: Host(`stage.freezenith.com`)
│          │  IngressRoute → zenith-landing:3000
└────┬─────┘
     │  HTTP :3000
     ▼
┌──────────────┐
│zenith-landing│  Next.js renders landing page
│  (pod)       │  Returns HTML + static assets
└──────────────┘

Total hops: 3 (Cloudflare → Traefik → Landing pod)
Latency: ~50-100ms (Cloudflare cache miss)
```

### Flow 2: Authenticated API Call

```
Web Dashboard (browser)
    │
    │  POST https://api.stage.freezenith.com/v1/apps
    │  Header: Authorization: Bearer <JWT from Keycloak>
    ▼
┌──────────┐
│Cloudflare│  DNS: api.stage.freezenith.com → 77.42.88.149
└────┬─────┘
     │  HTTPS :443
     ▼
┌──────────┐
│ Traefik  │  TLS termination
│          │  Match: Host(`api.stage.freezenith.com`)
│          │  IngressRoute → ExternalName svc → apisix-gateway-proxy
└────┬─────┘
     │  HTTP :9080
     ▼
┌──────────┐
│  APISIX  │  1. Route match: /v1/apps
│          │  2. jwt-auth plugin:
│          │     → Fetch JWKS from Keycloak
│          │     → Verify token signature + expiry
│          │     → Extract claims (user_id, roles)
│          │  3. cors plugin: check Origin header
│          │  4. limit-count plugin: check rate limit
│          │  5. Forward request with X-Consumer-* headers
└────┬─────┘
     │  HTTP :8080
     ▼
┌──────────┐
│zenith-api│  1. Read X-Consumer-Username header (trusted — APISIX verified)
│  (pod)   │  2. Business logic: create app
│          │  3. Query free-pg database
│          │  4. Send OTel trace span
│          │  5. Return JSON response
└────┬─────┘
     │  TCP :5432
     ▼
┌──────────┐
│ free-pg  │  PostgreSQL primary (free-pg-rw service)
│ (CNPG)   │  Database: zenith_platform
│          │  Returns query results
└──────────┘

Total hops: 5 (Cloudflare → Traefik → APISIX → API → PostgreSQL)
Latency: ~100-200ms
```

### Flow 3: Git Push → Build → Deploy

```
Developer pushes code to GitHub
    │
    │  git push origin main
    ▼
┌──────────┐
│  GitHub  │  Triggers webhook
│          │  POST https://api.stage.freezenith.com/v1/webhooks/github
└────┬─────┘
     │  HTTPS (via Cloudflare → Traefik → APISIX)
     ▼
┌──────────┐
│zenith-api│  1. Verify webhook signature (HMAC-SHA256)
│          │  2. Parse push event (repo, branch, commit)
│          │  3. Look up customer + app config
│          │  4. Create Kaniko Job in zenith-builds namespace
└────┬─────┘
     │  K8s API :6443
     ▼
┌──────────────────────────────────────────────────────────────┐
│  zenith-builds namespace                                      │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  Kaniko Job (ephemeral pod)                              │ │
│  │                                                          │ │
│  │  1. Clone git repo (from GitHub)                         │ │
│  │  2. Build Docker image (no Docker daemon needed)         │ │
│  │  3. Push image to Internal Harbor                        │ │
│  │     registry.stage.freezenith.com/zenith-stage/app:tag   │ │
│  │  4. Job completes → pod terminates                       │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────┬────────────────────────────────────────────────────┘
           │  Image pushed to Harbor
           ▼
┌────────────────────┐
│  Internal Harbor   │  Stores image layers in local storage
│  (separate server) │  Image available for pull
└──────────┬─────────┘
           │  ArgoCD Image Updater detects new tag
           ▼
┌──────────┐
│  ArgoCD  │  1. Image Updater sees new tag in Harbor
│          │  2. Updates Application spec (image tag)
│          │  3. Syncs: creates/updates Deployment in zenith-apps
└────┬─────┘
     │  K8s API
     ▼
┌──────────────────────────────────────────────────────────────┐
│  zenith-apps namespace                                        │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐   │
│  │  Customer App Deployment (updated)                     │   │
│  │  New pod starts with updated image                     │   │
│  │  Old pod terminates (rolling update)                   │   │
│  │  Traffic served via Traefik IngressRoute                │   │
│  └───────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘

Total time: ~3-8 minutes (build + push + sync)
```

### Flow 4: Customer Provisioning (Temporal)

```
User clicks "Sign Up" on freezenith.com
    │
    │  POST /v1/auth/register { email, password, plan: "free" }
    ▼
┌──────────┐
│zenith-api│  1. Validate input
│          │  2. Create user record in zenith_platform DB
│          │  3. Start Temporal workflow: "provision-customer"
└────┬─────┘
     │  gRPC :7233
     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  TEMPORAL  (temporal namespace)                                          │
│                                                                          │
│  Workflow: ProvisionCustomer(customerID, plan="free")                    │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │ Activity 1: CreateKeycloakRealm                                    │  │
│  │   → POST keycloak:8080/admin/realms  (create realm: customer-xxx)  │  │
│  │   → Create OIDC client + roles                                     │  │
│  │   ← Returns: client_id, client_secret                              │  │
│  ├────────────────────────────────────────────────────────────────────┤  │
│  │ Activity 2: CreateDatabase                                         │  │
│  │   → SQL to free-pg-rw:5432                                         │  │
│  │     CREATE DATABASE customer_xxx;                                   │  │
│  │     CREATE USER customer_xxx WITH PASSWORD '...';                   │  │
│  │   ← Returns: db_host, db_name, db_user, db_password                │  │
│  ├────────────────────────────────────────────────────────────────────┤  │
│  │ Activity 3: CreateS3Bucket                                         │  │
│  │   → Hetzner S3 API: create bucket customer-xxx-data                │  │
│  │   ← Returns: access_key, secret_key, bucket_name                   │  │
│  ├────────────────────────────────────────────────────────────────────┤  │
│  │ Activity 4: CreateK8sNamespace                                     │  │
│  │   → K8s API: create namespace zenith-customer-xxx                  │  │
│  │   → Apply: ResourceQuota (per free tier limits)                    │  │
│  │   → Apply: LimitRange, CiliumNetworkPolicy                        │  │
│  ├────────────────────────────────────────────────────────────────────┤  │
│  │ Activity 5: CreateK8sSecrets                                       │  │
│  │   → K8s API: create Secrets (DB, S3, Keycloak creds)              │  │
│  ├────────────────────────────────────────────────────────────────────┤  │
│  │ Activity 6: CreateDeployments                                      │  │
│  │   → K8s API: create Deployment + Service for customer app          │  │
│  ├────────────────────────────────────────────────────────────────────┤  │
│  │ Activity 7: CreateIngress                                          │  │
│  │   → K8s API: create IngressRoute (Traefik) + APISIX routes        │  │
│  │   → external-dns auto-creates DNS record in Cloudflare             │  │
│  ├────────────────────────────────────────────────────────────────────┤  │
│  │ Activity 8: WaitForReady                                           │  │
│  │   → Poll DNS resolution + TLS cert issuance                        │  │
│  │   → Update customer status = "ready"                               │  │
│  │   → Send welcome email                                             │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  If any activity fails → Temporal retries with backoff                   │
│  Full workflow history preserved for debugging                           │
└──────────────────────────────────────────────────────────────────────────┘
```

### Flow 5: Backup Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        BACKUP TIMELINE (daily)                          │
│                                                                         │
│  00:00 ─────── 02:00 ──────── 03:00 ──────── 04:00 ──────── 23:59     │
│    │              │               │               │              │      │
│    │         CNPG base        Velero           (idle)            │      │
│    │         backup           cluster                            │      │
│    │         (prefer-         backup                             │      │
│    │          standby)        (all K8s                           │      │
│    │                           resources)                        │      │
│    │                                                             │      │
│    └─────────── Continuous WAL archiving (every few seconds) ────┘      │
└─────────────────────────────────────────────────────────────────────────┘

Backup Flow Detail:

CNPG WAL Archiving (continuous, real-time):
┌────────────┐     ┌──────────────┐     ┌─────────────────────────────┐
│ free-pg    │     │ barman-cloud │     │ Hetzner S3                  │
│ PostgreSQL │────▶│ WAL shipper  │────▶│ s3://zenith-backups/        │
│ Writes WAL │     │ gzip compress│     │   free-pg-wal/              │
│ segments   │     │ 4 parallel   │     │ 14-day retention            │
└────────────┘     └──────────────┘     └─────────────────────────────┘

┌────────────┐     ┌──────────────┐     ┌─────────────────────────────┐
│keycloak-pg │     │ barman-cloud │     │ Hetzner S3                  │
│ PostgreSQL │────▶│ WAL shipper  │────▶│ s3://zenith-backups/        │
│ Writes WAL │     │ gzip compress│     │   keycloak-wal/             │
│ segments   │     │ 4 parallel   │     │ 14-day retention            │
└────────────┘     └──────────────┘     └─────────────────────────────┘

CNPG Scheduled Backup (daily at 02:00 UTC):
┌────────────┐     ┌──────────────┐     ┌─────────────────────────────┐
│ free-pg    │     │ barman-cloud │     │ Hetzner S3                  │
│ STANDBY    │────▶│ pg_basebackup│────▶│ s3://zenith-backups/        │
│ (minimal   │     │ full backup  │     │   free-pg-wal/              │
│  impact)   │     │              │     │   (base/ subdirectory)      │
└────────────┘     └──────────────┘     └─────────────────────────────┘

Velero Cluster Backup (daily at 03:00 UTC):
┌────────────┐     ┌──────────────┐     ┌─────────────────────────────┐
│ K8s API    │     │ Velero       │     │ Hetzner S3                  │
│ All        │────▶│ Server       │────▶│ s3://zenith-backups/        │
│ resources  │     │ + AWS plugin │     │   velero/                   │
│ (excl.     │     │ 30-day TTL   │     │ 30-day retention            │
│  velero ns)│     │              │     │                             │
└────────────┘     └──────────────┘     └─────────────────────────────┘
```

---

## 4. Namespace Map

Every Kubernetes namespace and what lives inside it.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        KUBERNETES NAMESPACES                            │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  SYSTEM NAMESPACES                                                      │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │ kube-system                                                      │   │
│  │   Traefik (ingress controller, DaemonSet)                        │   │
│  │   Cilium (agent DaemonSet + operator)                            │   │
│  │   Hubble (relay + UI)                                            │   │
│  │   CoreDNS                                                        │   │
│  │   Hetzner CSI (controller + node DaemonSet)                      │   │
│  │   k3s system components                                          │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  INFRASTRUCTURE NAMESPACES                                              │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐  │
│  │ cert-manager     │  │ sealed-secrets   │  │ cnpg-system          │  │
│  │  cert-manager    │  │  sealed-secrets  │  │  CNPG operator       │  │
│  │  controller      │  │  controller      │  │  (watches all ns)    │  │
│  │  webhook         │  │                  │  │                      │  │
│  │  cainjector      │  │                  │  │                      │  │
│  └──────────────────┘  └──────────────────┘  └──────────────────────┘  │
│                                                                         │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐  │
│  │ apisix           │  │ external-dns     │  │ keycloak             │  │
│  │  APISIX gateway  │  │  external-dns    │  │  Keycloak (1 pod)    │  │
│  │  APISIX ingress  │  │  controller      │  │  keycloak-pg         │  │
│  │  controller      │  │  (Cloudflare)    │  │  (CNPG cluster:      │  │
│  │  etcd (1 pod stg │  │                  │  │   2 stg / 3 prod)    │  │
│  │        3 prod)   │  │                  │  │                      │  │
│  └──────────────────┘  └──────────────────┘  └──────────────────────┘  │
│                                                                         │
│  PLATFORM NAMESPACES                                                    │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐  │
│  │ argocd           │  │ temporal         │  │ harbor               │  │
│  │  ArgoCD server   │  │  Temporal server │  │  Harbor core         │  │
│  │  Repo server     │  │  (frontend,      │  │  Harbor registry     │  │
│  │  App controller  │  │   history,       │  │  Harbor jobservice   │  │
│  │  Image updater   │  │   matching,      │  │  Harbor portal       │  │
│  │  harbor-image-   │  │   worker)        │  │  Trivy               │  │
│  │  updater-creds   │  │  Temporal Web UI │  │  Redis               │  │
│  └──────────────────┘  └──────────────────┘  └──────────────────────┘  │
│                                                                         │
│  ┌──────────────────┐  ┌──────────────────────────────────────────────┐ │
│  │ zenith-staging   │  │ zenith-shared                                │ │
│  │  zenith-api      │  │  free-pg (CNPG cluster: 2 stg / 3 prod)    │ │
│  │  zenith-web      │  │  DBs: zenith_platform, temporal,            │ │
│  │  zenith-landing  │  │        temporal_visibility,                  │ │
│  │  zenith-operator │  │        customer_xxx...                       │ │
│  │  zenith-demo     │  │                                              │ │
│  └──────────────────┘  └──────────────────────────────────────────────┘ │
│                                                                         │
│  SECURITY NAMESPACES                                                    │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐  │
│  │ kyverno          │  │ falco            │  │ velero               │  │
│  │  Kyverno         │  │  Falco DaemonSet │  │  Velero server       │  │
│  │  admission       │  │  Falcosidekick   │  │  AWS plugin          │  │
│  │  controller      │  │  (eBPF driver)   │  │  Daily schedule      │  │
│  └──────────────────┘  └──────────────────┘  └──────────────────────┘  │
│                                                                         │
│  OBSERVABILITY NAMESPACE                                                │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │ monitoring                                                       │   │
│  │  Prometheus (+ storage 20Gi stg / 50Gi prod)                     │   │
│  │  Grafana                                                         │   │
│  │  Alertmanager                                                    │   │
│  │  Loki (SingleBinary, 10Gi PVC)                                   │   │
│  │  Tempo (10Gi PVC)                                                │   │
│  │  OTel Collector (DaemonSet)                                      │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  AUTOSCALING NAMESPACE                                                  │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │ keda                                                             │   │
│  │  KEDA operator                                                   │   │
│  │  KEDA metrics server                                             │   │
│  │  KEDA HTTP add-on (HTTPScaledObject for scale-to-zero)           │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  CUSTOMER NAMESPACES                                                    │
│  ┌──────────────────┐  ┌──────────────────────────────────────────────┐ │
│  │ zenith-builds    │  │ zenith-apps                                  │ │
│  │  Kaniko build    │  │  Customer app Deployments + Services         │ │
│  │  jobs            │  │  cold-start-page (nginx for KEDA splash)    │ │
│  │  kaniko-registry │  │  app-registry-auth (pull secret)            │ │
│  │  -auth (push     │  │  HTTPScaledObjects (scale-to-zero)          │ │
│  │   secret)        │  │                                              │ │
│  └──────────────────┘  └──────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. DNS Map

Every subdomain in the Zenith platform and where it routes.

### Staging (`*.stage.freezenith.com`)

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         DNS MAP — STAGING                                        │
│                         Cloudflare Zone: freezenith.com                          │
│                         Server IP: 77.42.88.149                                  │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  PLATFORM SERVICES                                                               │
│  ─────────────────                                                               │
│  stage.freezenith.com           → Traefik → zenith-landing:3000 (landing page)  │
│  api.stage.freezenith.com       → Traefik → APISIX → zenith-api:8080 (API)     │
│  app.stage.freezenith.com       → Traefik → zenith-web:3000 (web dashboard)    │
│  ms.stage.freezenith.com        → Traefik → zenith-demo-mc:3100 (demo MC)      │
│  cloud.stage.freezenith.com     → Traefik → zenith-demo-web:3000 (demo Web)    │
│                                                                                  │
│  INFRASTRUCTURE UIs                                                              │
│  ──────────────────                                                              │
│  argocd.stage.freezenith.com    → Traefik → argocd-server:80 (GitOps UI)       │
│  auth.stage.freezenith.com      → Traefik → keycloak:8080 (Identity UI)        │
│  grafana.stage.freezenith.com   → Traefik → grafana:3000 (Dashboards)          │
│  hubble.stage.freezenith.com    → Traefik → hubble-ui:80 (Network flows)       │
│  temporal.stage.freezenith.com  → Traefik → temporal-web:8080 (Workflow UI)    │
│  hub.stage.freezenith.com       → Traefik → harbor-core:80 (Customer Harbor)   │
│  alerts.stage.freezenith.com    → Traefik → alertmanager:9093 (Alerts UI)      │
│  prometheus.stage.freezenith.com→ Traefik → prometheus:9090 (Metrics UI)       │
│  tempo.stage.freezenith.com     → Traefik → tempo:3200 (Traces UI)            │
│                                                                                  │
│  CUSTOMER APPS                                                                   │
│  ─────────────                                                                   │
│  *.apps.stage.freezenith.com    → Traefik → customer apps in zenith-apps        │
│  (wildcard A record)               e.g. myapp.apps.stage.freezenith.com         │
│                                                                                  │
│  LEGACY CUSTOMERS (V1)                                                           │
│  ─────────────────────                                                           │
│  embermind-ms.stage.freezenith.com → Traefik → embermind-mc:3100                │
│  embermind.stage.freezenith.com    → Traefik → embermind-web:3000               │
│                                                                                  │
│  REGISTRIES (separate servers)                                                   │
│  ─────────────────────────────                                                   │
│  registry.stage.freezenith.com  → Internal Harbor (separate server, NOT k3s)    │
│                                    Platform images + Helm charts                 │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Production (`*.freezenith.com`) — planned

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         DNS MAP — PRODUCTION (planned)                           │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  freezenith.com                 → zenith-landing (marketing site)                │
│  www.freezenith.com             → zenith-landing (redirect)                      │
│  app.freezenith.com             → zenith-web (web dashboard)                     │
│  api.freezenith.com             → APISIX → zenith-api (backend)                 │
│  auth.freezenith.com            → Keycloak (identity)                            │
│  argocd.freezenith.com          → ArgoCD UI                                      │
│  grafana.freezenith.com         → Grafana dashboards                             │
│  *.apps.freezenith.com          → Customer applications                          │
│  hub.freezenith.com             → Customer Harbor registry                       │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. PriorityClass & Eviction Order

When the node runs low on resources, Kubernetes evicts pods in this order (lowest priority first):

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    POD EVICTION ORDER                                    │
│                    (under resource pressure, lowest evicted FIRST)       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  EVICTED FIRST (lowest priority)                                        │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ Priority: 10000 — "customer" (global default)                      │ │
│  │ Who: All customer workloads in zenith-apps                         │ │
│  │ Impact: Customer apps go down first, but can reschedule            │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│        ↑ Evict first                                                    │
│        │                                                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ Priority: 100000 — "platform"                                      │ │
│  │ Who: zenith-api, Temporal, Harbor, Prometheus, monitoring stack     │ │
│  │ Impact: Platform degraded but infra stays up                       │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│        │                                                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ Priority: 500000 — "infra-critical"                                │ │
│  │ Who: CNPG clusters, Keycloak, APISIX, cert-manager                 │ │
│  │ Impact: Databases and identity down — major outage                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│        │                                                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ Priority: 1000000 — "core-critical"                                │ │
│  │ Who: Cilium, CoreDNS, Traefik                                      │ │
│  │ Impact: NEVER evicted — node is dead without these                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│        ↓ Evicted last                                                   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Storage Map

All persistent storage in the platform.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        STORAGE MAP                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  HETZNER BLOCK VOLUMES (StorageClass: hcloud-volumes, Retain)           │
│  ─────────────────────────────────────────────────────────────          │
│  keycloak-pg PVC          │ keycloak ns      │ 10Gi (staging)          │
│  free-pg PVC              │ zenith-shared    │ 50Gi (staging)          │
│  etcd PVC (APISIX)        │ apisix           │ 5Gi                     │
│  prometheus-server PVC    │ monitoring       │ 20Gi stg / 50Gi prod   │
│  loki PVC                 │ monitoring       │ 10Gi                    │
│  tempo PVC                │ monitoring       │ 10Gi                    │
│                                                                         │
│  HETZNER S3 BUCKETS (region: fsn1)                                      │
│  ──────────────────────────────────                                     │
│  zenith-backups/                                                        │
│  ├── keycloak-wal/        │ CNPG WAL archive (keycloak-pg)             │
│  ├── free-pg-wal/         │ CNPG WAL archive (free-pg)                 │
│  └── velero/              │ Velero cluster backups                      │
│                                                                         │
│  zenith-harbor/           │ Harbor image layer storage                  │
│                                                                         │
│  customer-xxx-data/       │ Per-customer object storage (created by API)│
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Quick Reference: Which Doc to Read

| I want to understand... | Read this doc |
|------------------------|---------------|
| How the whole system fits together | **SYSTEM-MAP.md** (this doc) |
| How to provision a new environment from scratch | [11-infrastructure-provisioning.md](./11-infrastructure-provisioning.md) |
| How external traffic enters the cluster | [12-traefik-ingress.md](./12-traefik-ingress.md) |
| How API routing, JWT, CORS, rate-limiting work | [13-apisix-gateway.md](./13-apisix-gateway.md) |
| How pod networking, encryption, network policies work | [14-cilium-networking.md](./14-cilium-networking.md) |
| How deployments are managed via GitOps | [15-argocd-gitops.md](./15-argocd-gitops.md) |
| How PostgreSQL, S3, and scale-to-zero work | [16-data-storage.md](./16-data-storage.md) |
| How automated provisioning workflows work | [17-temporal-workflows.md](./17-temporal-workflows.md) |
| How admission policies and runtime security work | [18-kyverno-policies.md](./18-kyverno-policies.md) |
| How cluster backups and restores work | [19-velero-backup.md](./19-velero-backup.md) |
| How encrypted secrets work in GitOps | [20-sealed-secrets.md](./20-sealed-secrets.md) |
| **How to set up my local dev environment** | [21-local-development-setup.md](./21-local-development-setup.md) |
| **How to add an endpoint, deploy, check logs** | [22-day-to-day-operations.md](./22-day-to-day-operations.md) |
| **How the Next.js apps are structured** | [23-frontend-architecture.md](./23-frontend-architecture.md) |
| **How CI/CD pipeline works (GitHub Actions → Harbor → ArgoCD)** | [24-ci-cd-pipeline.md](./24-ci-cd-pipeline.md) |
| **An alert fired — what do I do?** | [25-monitoring-runbook.md](./25-monitoring-runbook.md) |
| **How Keycloak manages customer identity** | [26-keycloak-administration.md](./26-keycloak-administration.md) |
| Go backend code architecture | [10-backend-architecture.md](./10-backend-architecture.md) |
| Security threat model | [06-security-model.md](./06-security-model.md) |
| Full observability stack | [08-observability.md](./08-observability.md) |

---

## Key Architectural Decisions

These are the critical "why" decisions behind the platform. Understanding these helps you avoid re-debating settled choices.

| # | Decision | What We Chose | Why (short) | Alternatives Considered |
|---|----------|---------------|-------------|------------------------|
| D1 | API Gateway | APISIX + etcd | Free plugin ecosystem (JWT, OTel, rate-limit), no Enterprise licensing cost, etcd is simpler than PostgreSQL | Kong OSS (limited free plugins), Kong Enterprise ($$$), Envoy (complex config) |
| D2 | Identity Provider | Keycloak | Mature, realm-per-tenant model, standard OIDC/SAML, dedicated CNPG database | Custom OIDC (too much work), Auth0 ($$$), Zitadel (less mature) |
| D3 | CNI | Cilium | eBPF (no iptables), WireGuard encryption, L7 network policies, Hubble observability | Flannel (no policies), Calico (no L7), default k3s CNI |
| D4 | GitOps | ArgoCD | App-of-Apps pattern, Image Updater, UI for debugging, CKA/ArgoCD exam prep | FluxCD (no UI), Terraform-only (no auto-sync) |
| D5 | PostgreSQL Operator | CNPG | Native K8s CRDs, WAL archiving to S3, PITR, automated failover, no extra operator DB | Zalando PG Operator, CrunchyData PGO |
| D6 | Workflow Engine | Temporal | Reliable retries, saga compensation, visibility UI, language-native SDK (Go) | Argo Workflows (YAML-based), custom queue + cron |
| D7 | Object Storage | Hetzner S3 | Same provider as compute (no egress costs), S3-compatible, cheap | AWS S3 (expensive egress), MinIO (self-managed) |
| D8 | Container Registry | Two Harbors (internal + customer) | Internal = platform images (separate server), Customer = pro-tier (in-cluster, Terraform-managed) | Single Harbor (blast radius), Docker Hub ($$$) |
| D9 | Secrets in Git | Sealed Secrets | Simple, no external dependencies, CRD-based, works with ArgoCD | SOPS (manual key mgmt), External Secrets (needs Vault), HashiCorp Vault ($$$) |
| D10 | Cluster Backup | Velero + CNPG WAL + pg_dump | 3-layer strategy: infrastructure (Velero) + continuous DB (WAL) + per-customer (pg_dump) | Velero alone (no DB granularity), Kasten K10 ($$$) |
| D11 | Policy Engine | Kyverno + Falco | Kyverno = admission-time (block bad configs), Falco = runtime (detect anomalies). Different phases, complementary. | OPA/Gatekeeper (steeper learning curve), Kyverno alone (no runtime detection) |
| D12 | Hosting Provider | Hetzner Cloud | 10x cheaper than AWS for same specs, EU data residency, good API, S3-compatible storage | AWS (expensive), DigitalOcean (less features), bare metal (too much work) |
| D13 | Ingress Controller | Traefik (built-in k3s) | Already bundled with k3s, IngressRoute CRD, cross-namespace routing, simple TLS | Nginx Ingress (less features), Contour (overkill) |
| D14 | Monitoring Stack | kube-prometheus-stack + Loki + Tempo | Industry standard, Grafana unifies metrics/logs/traces, works with Cilium Hubble | Datadog ($$$), ELK (heavy), custom Prometheus |

> **Rule:** When someone asks "why don't we use X instead?", check this table first. If the decision is listed, the alternative was already evaluated.
