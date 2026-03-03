# 19 — Velero Cluster Backup

> **Purpose:** Understand how Velero backs up all Kubernetes resources and how to restore the entire cluster or individual namespaces.
> **Audience:** Any developer who needs to perform backups, test restores, or understand disaster recovery.
> **Last Updated:** 2026-03-03
> **Related:** [07-backup-disaster-recovery.md](./07-backup-disaster-recovery.md) (full backup strategy: CNPG WAL + Velero + pg_dump), [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) (installation steps), [20-sealed-secrets.md](./20-sealed-secrets.md) (Velero backs up Sealed Secrets keys)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose It](#2-why-we-chose-it)
3. [Architecture Diagram](#3-architecture-diagram)
4. [How Velero Backs Up the Cluster](#4-how-velero-backs-up-the-cluster)
5. [Restore Flow](#5-restore-flow)
6. [Backup Schedule](#6-backup-schedule)
7. [Configuration Reference](#7-configuration-reference)
8. [Troubleshooting](#8-troubleshooting)
9. [Upgrade Path](#9-upgrade-path)

---

## 1. Overview

**Velero** backs up all Kubernetes resources (Deployments, Services, ConfigMaps, Secrets, CRDs, etc.) to Hetzner S3. It does NOT back up database contents — that's CNPG's job.

```
What Velero backs up:     All K8s manifests (the "infrastructure as YAML")
What Velero does NOT:     Database rows, S3 bucket contents, etcd data

Think of it this way:
  CNPG WAL  →  backs up your DATA (PostgreSQL rows)
  Velero    →  backs up your INFRASTRUCTURE (K8s resources)
  Together  →  full disaster recovery
```

---

## 2. Why We Chose It

| Feature | Velero | Kasten K10 | Stash | Manual kubectl export |
|---------|--------|-----------|-------|----------------------|
| S3 backend | Yes (AWS plugin) | Yes | Yes | No |
| Scheduled backups | Built-in | Built-in | Built-in | CronJob hack |
| Namespace filtering | Yes | Yes | Yes | Manual |
| Restore to different cluster | Yes | Yes | No | Manual |
| Volume snapshots | Yes (optional) | Yes | Yes | No |
| Cost | Free | $$$ | $$ | Free |
| Community | Large (VMware/CNCF) | Veeam | Small | N/A |

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         VELERO IN THE ZENITH CLUSTER                        │
│                         Namespace: velero                                   │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    VELERO SERVER (Deployment)                          │  │
│  │                    Resources: 50m CPU, 128Mi RAM                      │  │
│  │                                                                       │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ Velero Controller                                                │ │  │
│  │  │                                                                  │ │  │
│  │  │ 1. Watches for Backup/Restore CRDs                              │ │  │
│  │  │ 2. Executes backup on schedule (daily at 03:00 UTC)             │ │  │
│  │  │ 3. Queries K8s API for all resources                            │ │  │
│  │  │ 4. Serializes to JSON                                           │ │  │
│  │  │ 5. Uploads to S3 via AWS plugin                                 │ │  │
│  │  │ 6. Manages TTL (30-day retention)                               │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  │                                                                       │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ AWS Plugin (init container: velero-plugin-for-aws:v1.11.0)      │ │  │
│  │  │                                                                  │ │  │
│  │  │ Provides S3-compatible storage backend                          │ │  │
│  │  │ Config: s3ForcePathStyle=true (Hetzner uses path-style URLs)    │ │  │
│  │  │ Region: fsn1                                                     │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  CONNECTIONS:                                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                                                                       │  │
│  │  Velero ──HTTPS──▶ K8s API (:6443)                                   │  │
│  │    (list all resources in all namespaces)                              │  │
│  │                                                                       │  │
│  │  Velero ──HTTPS──▶ Hetzner S3                                        │  │
│  │    Bucket: zenith-backups                                              │  │
│  │    Prefix: velero/                                                     │  │
│  │    Credentials: AWS-format Secret (access_key + secret_key)           │  │
│  │                                                                       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. How Velero Backs Up the Cluster

```
Daily at 03:00 UTC (scheduled backup):
    │
    ▼
┌──────────────────────────────────────────────────────────────────────┐
│  VELERO BACKUP PROCESS                                                │
│                                                                       │
│  Step 1: Velero creates a Backup object                              │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ Name: daily-backup-20260303030000                              │   │
│  │ TTL: 720h (30 days)                                            │   │
│  │ Excluded namespaces: [velero]                                  │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  Step 2: Query K8s API for ALL resources                             │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ GET /api/v1/namespaces                                         │   │
│  │ GET /api/v1/pods (all namespaces)                              │   │
│  │ GET /api/v1/services (all namespaces)                          │   │
│  │ GET /api/v1/secrets (all namespaces)                           │   │
│  │ GET /api/v1/configmaps (all namespaces)                        │   │
│  │ GET /apis/apps/v1/deployments (all namespaces)                 │   │
│  │ GET /apis/traefik.io/v1alpha1/ingressroutes (all namespaces)   │   │
│  │ GET /apis/postgresql.cnpg.io/v1/clusters (all namespaces)      │   │
│  │ ... (every resource type in the cluster)                       │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  Step 3: Serialize to JSON and compress                               │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ Create tarball:                                                │   │
│  │   velero-backup.tar.gz containing:                             │   │
│  │     /resources/deployments.apps/namespaces/zenith-staging/     │   │
│  │       zenith-api.json                                          │   │
│  │       zenith-landing.json                                      │   │
│  │     /resources/services/namespaces/apisix/                     │   │
│  │       apisix-gateway.json                                      │   │
│  │     /resources/clusters.postgresql.cnpg.io/namespaces/...      │   │
│  │     ... (all resources)                                        │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  Step 4: Upload to S3                                                │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ PUT s3://zenith-backups/velero/backups/daily-backup-20260303/  │   │
│  │   velero-backup.json   (metadata)                              │   │
│  │   velero-backup.tar.gz (resource data)                         │   │
│  │   backup-log.gz        (operation log)                         │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  Step 5: Update Backup status = Completed                            │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ kubectl get backup -n velero                                   │   │
│  │ NAME                          STATUS      CREATED              │   │
│  │ daily-backup-20260303030000   Completed   2026-03-03T03:00:00  │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  After 30 days (TTL expired):                                        │
│    Velero deletes the backup from S3 and the Backup object           │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 5. Restore Flow

```
SCENARIO: Cluster is destroyed, need full restore
    │
    │  New cluster is set up (Terraform Phase 1 + Ansible Phase 2)
    │  Velero is installed (Terraform Phase 3)
    ▼
┌──────────────────────────────────────────────────────────────────────┐
│  VELERO RESTORE PROCESS                                               │
│                                                                       │
│  Step 1: List available backups                                      │
│  $ velero backup get                                                  │
│  NAME                          STATUS      CREATED                    │
│  daily-backup-20260303030000   Completed   2026-03-03T03:00:00        │
│  daily-backup-20260302030000   Completed   2026-03-02T03:00:00        │
│                                                                       │
│  Step 2: Restore from latest backup                                  │
│  $ velero restore create --from-backup daily-backup-20260303030000    │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ Velero downloads tarball from S3                               │   │
│  │ Extracts JSON resources                                        │   │
│  │ Applies to cluster via K8s API:                                │   │
│  │                                                                │   │
│  │  kubectl apply: namespace/zenith-staging                       │   │
│  │  kubectl apply: deployment/zenith-api                          │   │
│  │  kubectl apply: service/zenith-api                             │   │
│  │  kubectl apply: ingressroute/zenith-api                        │   │
│  │  kubectl apply: cluster/free-pg (CNPG)                         │   │
│  │  kubectl apply: secret/free-pg-app (credentials)               │   │
│  │  ... (all resources)                                           │   │
│  │                                                                │   │
│  │  Skips: existing resources (won't overwrite)                   │   │
│  │  Skips: PVs (data volumes need CNPG restore separately)        │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  Step 3: Verify                                                      │
│  $ velero restore describe <restore-name>                             │
│  Phase: Completed                                                     │
│  Warnings: 0                                                          │
│  Errors: 0                                                            │
│                                                                       │
│  Step 4: Restore databases (separate step — CNPG)                    │
│  CNPG clusters will start empty. Need to restore from WAL archives:  │
│  $ kubectl apply -f cnpg-cluster-restore.yaml                         │
│  (See 07-backup-disaster-recovery.md for CNPG PITR restore)          │
└──────────────────────────────────────────────────────────────────────┘

PARTIAL RESTORE (single namespace):
  $ velero restore create --from-backup daily-backup-20260303 \
      --include-namespaces zenith-staging
  (Only restores resources from zenith-staging namespace)
```

---

## 6. Backup Schedule

```
┌─────────────────────────────────────────────────────────────────────────┐
│              BACKUP SCHEDULE TIMELINE                                     │
│                                                                          │
│  Time (UTC)   What happens                                               │
│  ─────────    ────────────────────────────────────────────────────────  │
│  Continuous   CNPG WAL archiving (every few seconds)                    │
│               → PostgreSQL data is backed up in near-real-time          │
│                                                                          │
│  02:00        CNPG scheduled base backup (prefer-standby)               │
│               → Full PostgreSQL dump to S3                              │
│               → Runs on replica (no performance impact on primary)      │
│                                                                          │
│  03:00        Velero cluster backup                                      │
│               → All K8s resources to S3                                 │
│               → Excludes: velero namespace itself                       │
│               → Retention: 30 days (720h TTL)                           │
│                                                                          │
│  RPO (Recovery Point Objective):                                         │
│    Database: seconds (continuous WAL archiving)                          │
│    K8s resources: 24 hours (daily Velero backup)                         │
│                                                                          │
│  RTO (Recovery Time Objective):                                          │
│    Full cluster restore: ~30-60 minutes                                  │
│    (15 min Velero restore + 15-30 min CNPG PITR + DNS propagation)      │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Configuration Reference

### Terraform File

**File:** `infra/terraform/modules/k8s-platform/security.tf`

| Setting | Value | Purpose |
|---------|-------|---------|
| Namespace | velero | Deployment namespace |
| Plugin | velero-plugin-for-aws:v1.11.0 | S3-compatible storage |
| Bucket | zenith-backups | S3 bucket name |
| Prefix | velero | S3 path prefix |
| Region | fsn1 | Hetzner region |
| s3ForcePathStyle | true | Required for Hetzner S3 |
| s3Url | (from var.s3_endpoint) | Hetzner S3 endpoint |
| Schedule | `0 3 * * *` | Daily at 03:00 UTC |
| TTL | 720h | 30-day retention |
| Excluded namespaces | velero | Don't backup itself |
| upgradeCRDs | false | Skip CRD job (bitnami image issues) |
| VolumeSnapshotLocation | default (aws) | Volume snapshot provider |
| Resources | 50m CPU, 128Mi RAM | Server resources |

---

## 8. Troubleshooting

### Backup failed

```bash
# 1. Check backup status
velero backup get
velero backup describe <backup-name> --details

# 2. Check Velero logs
kubectl logs -n velero deploy/velero --tail=100

# 3. Common issues:
#    - S3 credentials expired → check Secret in velero namespace
#    - S3 bucket doesn't exist → create via Hetzner Console
#    - Timeout → increase timeout in Helm values
```

### Restore failed

```bash
# 1. Check restore status
velero restore describe <restore-name> --details

# 2. Common issues:
#    - CRDs missing (restore CRDs before resources)
#    - Resource conflicts (resource already exists → use --existing-resource-policy=update)
#    - PVC stuck in Pending (StorageClass not available)
```

### S3 connectivity issues

```bash
# Test from Velero pod
kubectl exec -n velero deploy/velero -- \
  aws s3 ls s3://zenith-backups/velero/ \
    --endpoint-url=https://hel1.your-objectstorage.com

# Check credentials
kubectl get secret -n velero velero-credentials -o yaml
```

---

## 9. Upgrade Path

### Upgrading Velero

```bash
# Update version in variables.tf
terraform plan -target=helm_release.velero
terraform apply -target=helm_release.velero

# NOTE: Set upgradeCRDs=false (we manage CRDs manually to avoid bitnami image issues)
# After upgrade, verify:
velero version
velero backup get
```

### Changing backup schedule

Update in `security.tf`:
```hcl
set {
  name  = "schedules.daily-backup.schedule"
  value = "0 3 * * *"  # Change cron expression
}
```

### Adding namespace exclusion

```hcl
set {
  name  = "schedules.daily-backup.template.excludedNamespaces[1]"
  value = "my-excluded-namespace"
}
```
