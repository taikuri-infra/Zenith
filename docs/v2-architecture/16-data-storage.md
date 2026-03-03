# 16 — Data & Storage (CNPG + Hetzner S3 + KEDA)

> **Purpose:** Understand how PostgreSQL databases, object storage, and scale-to-zero work in Zenith.
> **Audience:** Any developer who needs to manage databases, debug storage issues, or understand the data layer.
> **Last Updated:** 2026-03-03
> **Related:** [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) (installation steps), [07-backup-disaster-recovery.md](./07-backup-disaster-recovery.md) (backup/restore for CNPG + S3), [17-temporal-workflows.md](./17-temporal-workflows.md) (database provisioning via Temporal)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose These Tools](#2-why-we-chose-these-tools)
3. [Architecture Diagram](#3-architecture-diagram)
4. [CNPG — PostgreSQL Operator](#4-cnpg--postgresql-operator)
5. [Database Sharding Strategy](#5-database-sharding-strategy)
6. [Hetzner S3 — Object Storage](#6-hetzner-s3--object-storage)
7. [KEDA — Scale to Zero](#7-keda--scale-to-zero)
8. [Data Flow: Customer Lifecycle](#8-data-flow-customer-lifecycle)
9. [Configuration Reference](#9-configuration-reference)
10. [Troubleshooting](#10-troubleshooting)
11. [Upgrade Path](#11-upgrade-path)

---

## 1. Overview

The data layer has three components:

- **CNPG (CloudNativePG)** — Kubernetes operator that manages PostgreSQL clusters with automatic failover, WAL archiving, and backups
- **Hetzner S3** — S3-compatible object storage for backups, customer files, and Harbor image layers
- **KEDA** — Event-driven autoscaler that scales free-tier customer apps to zero when idle

```
Data storage in Zenith:

  Structured data  →  CNPG PostgreSQL (Hetzner Block Volumes)
  Unstructured data →  Hetzner S3 (Object Storage)
  Backup data       →  Hetzner S3 (WAL archives + Velero backups)
```

---

## 2. Why We Chose These Tools

| Tool | Alternative | Why We Chose It |
|------|-------------|----------------|
| **CNPG** | Zalando PG Operator, CrunchyData | Best K8s-native PG operator, active community, Barman integration for S3 backups |
| **Hetzner S3** | AWS S3, MinIO, Cloudflare R2 | Cheapest (included in Hetzner), EU data residency, S3-compatible API |
| **KEDA** | VPA, custom HPA | Only option for true scale-to-zero, HTTP-based triggers, minimal overhead |

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DATA LAYER ARCHITECTURE                             │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ CNPG OPERATOR (cnpg-system namespace)                                 │  │
│  │ Watches ALL namespaces for Cluster CRDs                               │  │
│  │ Resources: 25m-200m CPU, 64Mi-256Mi RAM                              │  │
│  └──────────┬─────────────────────────────┬──────────────────────────────┘  │
│             │                             │                                 │
│             ▼                             ▼                                 │
│  ┌─────────────────────────┐  ┌──────────────────────────────────────────┐ │
│  │ keycloak-pg              │  │ free-pg                                  │ │
│  │ (keycloak namespace)     │  │ (zenith-shared namespace)                │ │
│  │                          │  │                                          │ │
│  │ ┌────────────────────┐  │  │ ┌────────────────────────────────────┐   │ │
│  │ │ keycloak-pg-1      │  │  │ │ free-pg-1 (PRIMARY)                │   │ │
│  │ │ PRIMARY (read/write)│  │  │ │ read/write                        │   │ │
│  │ │ Service: keycloak- │  │  │ │ Service: free-pg-rw               │   │ │
│  │ │   pg-rw            │  │  │ │                                    │   │ │
│  │ │ PVC: 10Gi hcloud   │  │  │ │ Databases:                        │   │ │
│  │ │ DB: keycloak       │  │  │ │   zenith_platform (API's own DB)  │   │ │
│  │ └────────────────────┘  │  │ │   temporal (workflow engine DB)    │   │ │
│  │                          │  │ │   temporal_visibility              │   │ │
│  │ ┌────────────────────┐  │  │ │   customer_abc (free user)        │   │ │
│  │ │ keycloak-pg-2      │  │  │ │   customer_def (free user)        │   │ │
│  │ │ REPLICA (read only)│  │  │ │   ...                              │   │ │
│  │ │ Service: keycloak- │  │  │ │ PVC: 50Gi hcloud-volumes          │   │ │
│  │ │   pg-ro            │  │  │ └────────────────────────────────────┘   │ │
│  │ │ Streaming replication│  │ │                                          │ │
│  │ │ from primary        │  │  │ ┌────────────────────────────────────┐   │ │
│  │ └────────────────────┘  │  │ │ free-pg-2 (REPLICA)                │   │ │
│  │                          │  │ │ read only + backup target          │   │ │
│  │ WAL archiving:           │  │ │ Service: free-pg-ro               │   │ │
│  │ s3://zenith-backups/     │  │ │ Streaming replication from primary│   │ │
│  │   keycloak-wal/          │  │ └────────────────────────────────────┘   │ │
│  │ Retention: 14 days       │  │                                          │ │
│  │ Daily backup: 02:00 UTC  │  │ WAL archiving:                          │ │
│  └─────────────────────────┘  │ s3://zenith-backups/free-pg-wal/          │ │
│                                │ Retention: 14 days                        │ │
│                                │ Daily backup: 02:00 UTC                   │ │
│                                └──────────────────────────────────────────┘ │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ HETZNER S3 (external, S3-compatible)                                  │  │
│  │                                                                       │  │
│  │ ┌────────────────┐ ┌────────────────┐ ┌──────────────────────────┐   │  │
│  │ │ zenith-backups │ │ zenith-harbor  │ │ customer-xxx-data       │   │  │
│  │ │                │ │                │ │ (one per customer)       │   │  │
│  │ │ /keycloak-wal/ │ │ Harbor image   │ │ Customer files,          │   │  │
│  │ │ /free-pg-wal/  │ │ layers for     │ │ uploads, assets          │   │  │
│  │ │ /velero/       │ │ customer       │ │                          │   │  │
│  │ │                │ │ registry       │ │ Created by zenith-api    │   │  │
│  │ │ WAL archives + │ │                │ │ during provisioning      │   │  │
│  │ │ base backups + │ │                │ │                          │   │  │
│  │ │ cluster backups│ │                │ │                          │   │  │
│  │ └────────────────┘ └────────────────┘ └──────────────────────────┘   │  │
│  │ Region: fsn1       Endpoint: hel1.your-objectstorage.com            │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ KEDA (keda namespace)                                                 │  │
│  │                                                                       │  │
│  │ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────────┐   │  │
│  │ │ KEDA Operator    │ │ KEDA Metrics     │ │ KEDA HTTP Add-on     │   │  │
│  │ │ 1 replica        │ │ Server           │ │                      │   │  │
│  │ │ 100m CPU, 128Mi  │ │ 50m CPU, 64Mi    │ │ Intercepts HTTP      │   │  │
│  │ │                  │ │                  │ │ requests for          │   │  │
│  │ │ Watches for      │ │ Provides custom  │ │ scaled-to-zero apps  │   │  │
│  │ │ ScaledObject and │ │ metrics to HPA   │ │                      │   │  │
│  │ │ HTTPScaledObject │ │                  │ │ Triggers scale-up    │   │  │
│  │ │ CRDs             │ │                  │ │ on first request     │   │  │
│  │ └──────────────────┘ └──────────────────┘ └──────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ HETZNER CSI DRIVER (kube-system namespace)                            │  │
│  │                                                                       │  │
│  │ StorageClass: hcloud-volumes                                          │  │
│  │ ReclaimPolicy: Retain (data preserved when PVC deleted)               │  │
│  │ Provisioner: csi.hetzner.cloud                                        │  │
│  │                                                                       │  │
│  │ Provides PersistentVolumes for:                                       │  │
│  │   PostgreSQL (CNPG), etcd (APISIX), Prometheus, Loki, Tempo          │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. CNPG — PostgreSQL Operator

### How CNPG Manages PostgreSQL

```
┌─────────────────────────────────────────────────────────────────────────┐
│              CNPG CLUSTER LIFECYCLE                                       │
│                                                                          │
│  1. You create a Cluster CRD:                                            │
│     apiVersion: postgresql.cnpg.io/v1                                    │
│     kind: Cluster                                                        │
│     spec: instances: 2, storage: 50Gi                                    │
│                                                                          │
│  2. CNPG Operator sees the CRD and creates:                             │
│     ┌────────────────────────────────────────────────────────────────┐  │
│     │  Pod: free-pg-1 (primary)                                      │  │
│     │    Container: PostgreSQL 16.6                                   │  │
│     │    PVC: 50Gi hcloud-volumes                                    │  │
│     │    Service: free-pg-rw (read/write → primary only)             │  │
│     │                                                                │  │
│     │  Pod: free-pg-2 (replica)                                      │  │
│     │    Container: PostgreSQL 16.6                                   │  │
│     │    PVC: 50Gi hcloud-volumes (streaming replication from primary)│  │
│     │    Service: free-pg-ro (read-only → replicas only)             │  │
│     │                                                                │  │
│     │  Service: free-pg-r (read → any instance, for connection pool) │  │
│     │                                                                │  │
│     │  Secret: free-pg-app (auto-generated credentials)              │  │
│     │    username, password, dbname, host, port, uri, jdbc-uri       │  │
│     │                                                                │  │
│     │  PodMonitor: free-pg (Prometheus scraping)                     │  │
│     └────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  3. Continuous operations:                                               │
│     - WAL archiving to S3 (every few seconds)                           │
│     - Health checks (liveness + readiness probes)                        │
│     - Automatic failover (primary dies → promote replica)                │
│     - Scheduled base backups (daily at 02:00 UTC)                       │
│     - Metrics exported to Prometheus                                     │
│                                                                          │
│  4. Automatic failover:                                                  │
│     ┌──────────┐  primary dies    ┌──────────────────────────────────┐  │
│     │ free-pg-1│ ──────────────▶  │ CNPG detects failure              │  │
│     │ PRIMARY  │                  │ 1. Promote free-pg-2 to PRIMARY   │  │
│     │  (dead)  │                  │ 2. Update free-pg-rw service      │  │
│     └──────────┘                  │ 3. Create new replica pod          │  │
│                                   │ 4. Total downtime: ~10-30 seconds │  │
│     ┌──────────┐                  └──────────────────────────────────┘  │
│     │ free-pg-2│ ◀── promoted to PRIMARY                                │
│     │ REPLICA  │     Service free-pg-rw now points here                 │
│     └──────────┘                                                        │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Database Sharding Strategy

```
┌─────────────────────────────────────────────────────────────────────────┐
│              DATABASE SHARDING STRATEGY                                   │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ CNPG OPERATOR (watches all namespaces)                             │ │
│  └──────┬───────────────────────────────────┬────────────────────────┘ │
│         │                                   │                          │
│         ▼                                   ▼                          │
│  ┌──────────────────┐            ┌──────────────────────────────────┐ │
│  │ keycloak-pg       │            │ free-pg                          │ │
│  │ DEDICATED          │            │ SHARED                           │ │
│  │                   │            │                                  │ │
│  │ 1 database:       │            │ Platform databases:              │ │
│  │   keycloak        │            │   zenith_platform (API's own)    │ │
│  │                   │            │   temporal                       │ │
│  │ Why dedicated?    │            │   temporal_visibility             │ │
│  │ Identity data is  │            │                                  │ │
│  │ critical — must   │            │ Customer databases:              │ │
│  │ not share with    │            │   customer_abc  (free user)      │ │
│  │ customer workloads│            │   customer_def  (free user)      │ │
│  └──────────────────┘            │   customer_ghi  (free user)      │ │
│                                   │   ...up to ~100 free users       │ │
│                                   │                                  │ │
│                                   │ Parameters:                      │ │
│                                   │   max_connections: 200           │ │
│                                   │   shared_buffers: 256MB          │ │
│                                   │   statement_timeout: 30s         │ │
│                                   └──────────────────────────────────┘ │
│                                                                        │
│  FUTURE: Pro user sharding (when needed)                               │
│  ┌────────────────────────────────────────────────────────────────────┐│
│  │                                                                    ││
│  │  pro-shard-1 (Cluster)      pro-shard-2 (Cluster)                 ││
│  │  ┌────────────────────┐     ┌────────────────────┐                ││
│  │  │ ~20 pro customers  │     │ ~20 pro customers  │                ││
│  │  │ Each: up to 5 DBs  │     │ Each: up to 5 DBs  │                ││
│  │  │ Each: up to 5GB    │     │ Each: up to 5GB    │                ││
│  │  │ 100Gi PVC          │     │ 100Gi PVC          │                ││
│  │  └────────────────────┘     └────────────────────┘                ││
│  │                                                                    ││
│  │  Assignment algorithm:                                             ││
│  │    1. API checks which shards have < 20 customers                 ││
│  │    2. Assigns to first available shard                             ││
│  │    3. If all shards full → alert admin (or auto-create new shard) ││
│  └────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Hetzner S3 — Object Storage

```
┌─────────────────────────────────────────────────────────────────────────┐
│              S3 BUCKET LAYOUT                                            │
│                                                                          │
│  Hetzner Object Storage (region: fsn1)                                   │
│  S3-compatible API (works with AWS SDK, s3cmd, mc)                      │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ zenith-backups/                          (platform bucket)         │ │
│  │ ├── keycloak-wal/                        CNPG WAL + base backups  │ │
│  │ │   ├── base/20260303T020000/            Daily base backup        │ │
│  │ │   └── wals/000000010000000000000042    WAL segments (gzip)      │ │
│  │ ├── free-pg-wal/                         CNPG WAL + base backups  │ │
│  │ │   ├── base/20260303T020000/            Daily base backup        │ │
│  │ │   └── wals/000000010000000000000099    WAL segments (gzip)      │ │
│  │ └── velero/                              Velero cluster backups   │ │
│  │     ├── backups/daily-backup-20260303/   K8s manifests            │ │
│  │     └── restores/                        Restore metadata         │ │
│  ├────────────────────────────────────────────────────────────────────┤ │
│  │ zenith-harbor/                           (Harbor image storage)    │ │
│  │ └── docker/registry/v2/blobs/            Image layers             │ │
│  ├────────────────────────────────────────────────────────────────────┤ │
│  │ customer-abc-data/                       (per free customer)      │ │
│  │ customer-def-data/                       (per free customer)      │ │
│  │ customer-pro-001-data/                   (per pro customer)       │ │
│  │ customer-pro-001-assets/                 (pro: 2nd bucket)        │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  Access pattern:                                                         │
│    CNPG → uses barman-cloud-wal-archive (built into CNPG containers)     │
│    Velero → uses velero-plugin-for-aws (S3 compatible)                   │
│    Harbor → uses S3 storage driver (built into Harbor)                   │
│    zenith-api → uses Go AWS SDK to create customer buckets               │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. KEDA — Scale to Zero

```
┌─────────────────────────────────────────────────────────────────────────┐
│              KEDA SCALE-TO-ZERO FLOW (Free tier apps)                    │
│                                                                          │
│  IDLE STATE (no traffic for 15 minutes):                                 │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ Customer app: 0 replicas (scaled to zero by KEDA)                  │ │
│  │ KEDA HTTP interceptor: watching for incoming requests              │ │
│  │ cold-start-page: nginx serving splash page                        │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  COLD START (first request arrives):                                     │
│                                                                          │
│  User Browser                                                            │
│      │  GET https://myapp.apps.stage.freezenith.com                     │
│      ▼                                                                   │
│  ┌──────────┐                                                           │
│  │ Traefik  │ Routes to customer service                                │
│  └────┬─────┘                                                           │
│       │                                                                  │
│       ▼                                                                  │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │ KEDA HTTP Interceptor                                             │   │
│  │                                                                   │   │
│  │ 1. Sees: destination has 0 replicas                               │   │
│  │ 2. Triggers scale-up: 0 → 1 replica                              │   │
│  │ 3. While pod is starting (30-60 seconds):                         │   │
│  │    → Redirects to cold-start-page middleware                      │   │
│  │    → cold-start-page shows splash screen:                         │   │
│  │                                                                   │   │
│  │    ┌──────────────────────────────────────────────────────────┐  │   │
│  │    │                                                          │  │   │
│  │    │           ⏳ Your app is waking up...                     │  │   │
│  │    │                                                          │  │   │
│  │    │      This app was sleeping to save resources.             │  │   │
│  │    │      It will be ready in a few seconds.                   │  │   │
│  │    │                                                          │  │   │
│  │    │      (auto-refreshes every 5 seconds)                    │  │   │
│  │    │                                                          │  │   │
│  │    └──────────────────────────────────────────────────────────┘  │   │
│  │                                                                   │   │
│  │ 4. Once pod is Ready:                                             │   │
│  │    → KEDA interceptor routes traffic to the actual pod            │   │
│  │    → User sees their app (after auto-refresh)                     │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  WARM STATE (has traffic):                                               │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ Customer app: 1 replica (running)                                  │ │
│  │ KEDA: monitoring request count                                     │ │
│  │ If no requests for 15 minutes → scale back to 0                   │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  WHY SCALE TO ZERO?                                                      │
│  Free tier gets 1 app. If 100 free users each have an app,              │
│  that's 100 pods always running. Scale-to-zero means only               │
│  active apps consume resources. Typical: 5-10 active out of 100.        │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Data Flow: Customer Lifecycle

```
SIGN UP: Customer creates account
    │
    │  Temporal workflow: ProvisionCustomer
    ▼
┌──────────┐
│zenith-api│
│          │  1. CREATE DATABASE customer_abc
│          │     on free-pg-rw:5432
│          │  2. CREATE USER customer_abc WITH PASSWORD '...'
│          │  3. Store credentials in K8s Secret
│          │  4. Create S3 bucket: customer-abc-data (via Hetzner API)
│          │  5. Store S3 credentials in K8s Secret
└──────────┘

RUNNING: Customer app uses their database
    │
    │  Customer app reads credentials from K8s Secret
    ▼
┌──────────────┐  TCP :5432  ┌──────────────┐
│ customer-app │────────────▶│ free-pg-rw   │
│ (pod)        │             │ (primary)    │
│              │             │              │
│ DB_HOST=free-│             │ DB: customer_│
│  pg-rw.zenith│             │   abc        │
│  -shared.svc │             │              │
└──────────────┘             └──────────────┘

UPGRADE: Customer upgrades from Free to Pro
    │
    │  Temporal workflow: UpgradeCustomer
    ▼
┌──────────┐
│zenith-api│
│          │  1. pg_dump customer_abc from free-pg
│          │  2. Assign pro shard (find one with < 20 customers)
│          │  3. pg_restore customer_abc to pro-shard-1
│          │  4. Update K8s Secret with new DB host
│          │  5. Create Harbor project for customer (hub.stage.freezenith.com)
│          │  6. Update limits: 5 apps, 3 DBs, 5GB each
└──────────┘

DELETE: Customer deletes account
    │
    │  Temporal workflow: DeprovisionCustomer
    ▼
┌──────────┐
│zenith-api│
│          │  1. pg_dump customer_abc → S3 (final backup, 90-day retention)
│          │  2. DROP DATABASE customer_abc
│          │  3. DROP USER customer_abc
│          │  4. Delete S3 bucket (after retention period)
│          │  5. Delete K8s namespace + all resources
│          │  6. Delete Keycloak realm
│          │  7. Delete DNS records
└──────────┘
```

---

## 9. Configuration Reference

### CNPG Operator

**File:** `infra/terraform/modules/k8s-platform/storage.tf`

| Setting | Staging | Production |
|---------|---------|------------|
| Namespace | cnpg-system | cnpg-system |
| Resources | 25m-200m CPU, 64Mi-256Mi RAM | Same |
| Inherited annotations | `cert-manager.io/*` | Same |
| Inherited labels | `app.kubernetes.io/*` | Same |

### Keycloak PG Cluster

| Setting | Staging | Production |
|---------|---------|------------|
| Instances | 2 | 3 |
| Storage | 10Gi | 10Gi |
| max_connections | 100 | 100 |
| shared_buffers | 128MB | 128MB |
| WAL destination | `s3://zenith-backups/keycloak-wal/` | Same |
| WAL compression | gzip, 4 parallel | Same |
| WAL retention | 14 days | 14 days |
| Daily backup | 02:00 UTC (prefer-standby) | Same |
| PriorityClass | infra-critical | infra-critical |

### Free PG Cluster

| Setting | Staging | Production |
|---------|---------|------------|
| Instances | 2 | 3 |
| Storage | 50Gi | 50Gi |
| max_connections | 200 | 200 |
| shared_buffers | 256MB | 256MB |
| statement_timeout | 30s | 30s |
| superuserAccess | true | true |
| WAL destination | `s3://zenith-backups/free-pg-wal/` | Same |
| PriorityClass | infra-critical | infra-critical |

### Hetzner CSI

| Setting | Value |
|---------|-------|
| StorageClass | hcloud-volumes |
| ReclaimPolicy | Retain |
| defaultStorageClass | false |

### KEDA

**File:** `infra/terraform/modules/k8s-platform/autoscaling.tf`

| Setting | Value |
|---------|-------|
| Operator replicas | 1 |
| Operator CPU | 100m |
| Operator memory | 128Mi |
| Metrics server CPU | 50m |
| HTTP add-on | Enabled (keda-add-ons-http chart) |

---

## 10. Troubleshooting

### CNPG cluster not healthy

```bash
# Check cluster status
kubectl get cluster -A
kubectl describe cluster free-pg -n zenith-shared

# Check pod status
kubectl get pods -n zenith-shared -l cnpg.io/cluster=free-pg

# Check CNPG operator logs
kubectl logs -n cnpg-system deploy/cnpg-cloudnative-pg --tail=50

# Check PostgreSQL logs
kubectl logs -n zenith-shared free-pg-1 --tail=50
```

### WAL archiving failing

```bash
# Check backup status
kubectl get backup -n zenith-shared
kubectl describe scheduledbackup free-pg-daily -n zenith-shared

# Check S3 credentials
kubectl get secret cnpg-s3-credentials -n zenith-shared -o yaml

# Test S3 connectivity from inside the cluster
kubectl exec -n zenith-shared free-pg-1 -- \
  barman-cloud-wal-archive --test s3://zenith-backups/free-pg-wal/
```

### Customer can't connect to database

```bash
# Check if Secret exists with correct credentials
kubectl get secret -n zenith-apps | grep customer-abc

# Check if free-pg-rw service has endpoints
kubectl get endpoints free-pg-rw -n zenith-shared

# Test connection from customer namespace
kubectl exec -n zenith-apps <customer-pod> -- \
  pg_isready -h free-pg-rw.zenith-shared.svc -p 5432

# Check CiliumNetworkPolicy allows traffic
hubble observe --from-namespace zenith-apps --to-namespace zenith-shared
```

### KEDA not scaling to zero

```bash
# Check HTTPScaledObject
kubectl get httpscaledobject -n zenith-apps

# Check KEDA operator logs
kubectl logs -n keda deploy/keda-operator --tail=50

# Check if HTTP add-on interceptor is running
kubectl get pods -n keda -l app=keda-add-ons-http-interceptor
```

---

## 11. Upgrade Path

### Upgrading CNPG Operator

```bash
# Update version in variables.tf, then:
terraform plan -target=helm_release.cnpg
terraform apply -target=helm_release.cnpg
# Operator upgrade is non-disruptive — PG clusters keep running
```

### Upgrading PostgreSQL Version

```yaml
# In the Cluster CRD, update imageName:
spec:
  imageName: ghcr.io/cloudnative-pg/postgresql:17.0
# CNPG performs a rolling update (replica first, then switchover primary)
```

### Scaling a CNPG cluster

```bash
# Change instances count in storage.tf (e.g., 2 → 3):
terraform apply -target=kubernetes_manifest.cnpg_free
# CNPG creates a new replica automatically
```
