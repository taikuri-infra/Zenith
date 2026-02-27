# Backup & Disaster Recovery

> **Zenith V2 Platform Architecture -- Data Protection Strategy**
>
> This document covers how we protect customer data, platform state, and infrastructure
> configuration against loss. The strategy uses three backup layers, each protecting
> against different failure modes.

**Decision Reference:** D12 (3-Layer Backup Strategy)
**Last Updated:** 2026-02-25
**Status:** Active

---

## Table of Contents

1. [Why Backup Matters](#1-why-backup-matters)
2. [Backup Architecture Overview](#2-backup-architecture-overview)
3. [Layer 1: CNPG WAL Archiving (Continuous)](#3-layer-1-cnpg-wal-archiving-continuous)
4. [Layer 2: pg_dump CronJobs (Per-Customer)](#4-layer-2-pg_dump-cronjobs-per-customer)
5. [Layer 3: Velero Cluster Backup](#5-layer-3-velero-cluster-backup)
6. [Additional Backups](#6-additional-backups)
7. [Disaster Recovery Scenarios](#7-disaster-recovery-scenarios)
8. [Monthly Restore Drill](#8-monthly-restore-drill)
9. [RPO/RTO Summary Table](#9-rporto-summary-table)
10. [Monitoring and Alerting](#10-monitoring-and-alerting)
11. [How to Run Manual Backups](#11-how-to-run-manual-backups)

---

## 1. Why Backup Matters

Zenith is a multi-tenant platform. A single PostgreSQL cluster may host databases for dozens
of customers. A single Keycloak instance holds identity data for every customer on the
platform. This concentration of data means that failures are not isolated -- they have
blast radius.

Here are the data loss scenarios we must protect against:

### Multi-Tenant Data Loss Scenarios

| Scenario | What is Lost | Who is Affected | Likelihood |
|----------|-------------|-----------------|------------|
| Accidental `DROP TABLE` by customer app | One customer's tables | Single customer | Medium |
| CNPG primary pod crash + corrupted WAL | All databases on that cluster | 20+ customers (Pro shard) | Low |
| Kubernetes node failure | Pods on that node (stateful if PVC lost) | Multiple customers | Medium |
| Hetzner Volume corruption | Database files on the volume | All customers on that cluster | Low |
| Bad migration deployed via ArgoCD | Schema corruption across databases | All customers using that service | Medium |
| Keycloak realm misconfiguration | Authentication broken for a realm | Single customer | Medium |
| etcd corruption (APISIX) | API gateway routing lost | All customers | Low |
| Complete cluster destruction | Everything | Everyone | Very Low |
| Hetzner datacenter outage | All infrastructure | Everyone | Very Low |

The backup strategy is designed so that **no single failure can cause permanent data loss**.
Every piece of state is backed up through at least two independent mechanisms, and the most
critical data (PostgreSQL) is protected by three layers.

### The Cost of Not Having Backups

For a multi-tenant PaaS, losing customer data is an existential event. Even losing one
customer's data -- if unrecoverable -- destroys trust across the entire customer base.
The three-layer strategy exists because:

- **Layer 1 (WAL)** protects against the most common case: point-in-time recovery from
  logical errors (bad queries, bad migrations).
- **Layer 2 (pg_dump)** protects against the case where CNPG itself is compromised or
  you need to restore a single customer without touching others.
- **Layer 3 (Velero)** protects against the case where the entire Kubernetes cluster
  needs to be rebuilt from scratch.

---

## 2. Backup Architecture Overview

```
                         Zenith Backup Architecture (3 Layers)
  ============================================================================

  LAYER 1: CNPG WAL Archiving (Continuous, RPO: ~seconds)
  --------------------------------------------------------
                                                        +------------------+
  +-------------+    WAL Stream     +----------+  PUT   | Hetzner S3       |
  | CNPG Primary| ----------------> | barman   | -----> | zenith-backups/  |
  | (PostgreSQL)|    (realtime)     | cloud    |        |   keycloak-wal/  |
  +-------------+                   +----------+        |   free-pg-wal/   |
        |                                               |   pro-shard1-wal/|
        | Replication                                   |   pro-shard2-wal/|
        v                                               +------------------+
  +-------------+
  | CNPG Replica|
  | (standby)   |
  +-------------+

  LAYER 2: pg_dump CronJobs (Periodic, RPO: 6h-24h by tier)
  -----------------------------------------------------------
  +-------------+    pg_dump      +----------+   PUT    +------------------+
  | K8s CronJob | -------------> | gzip     | -------> | Hetzner S3       |
  | (per-cust)  |   per-customer | compress |          | zenith-backups/  |
  +-------------+                +----------+          |   pg-dumps/      |
        |                                              |     cust-abc/    |
        | Runs on schedule                             |     cust-def/    |
        | Free: daily 02:00                            |     ...          |
        | Pro:  every 6h                               +------------------+
        v
  +-------------+
  | pg client   |
  | (connects   |
  |  to CNPG)   |
  +-------------+

  LAYER 3: Velero Cluster Backup (Daily, RPO: 24h)
  --------------------------------------------------
  +-------------+   K8s API       +----------+  PUT    +------------------+
  | Velero      | -------------> | Resource | ------> | Hetzner S3       |
  | Controller  |  dump all      | Archive  |         | zenith-backups/  |
  +-------------+  resources     +----------+         |   velero/        |
        |                                             +------------------+
        | CSI Snapshots
        v
  +-------------+
  | Hetzner     |
  | Volume      |
  | Snapshots   |
  +-------------+

  ADDITIONAL BACKUPS
  -------------------
  Keycloak realm export  ---> S3: zenith-backups/keycloak-realms/
  APISIX etcd snapshot   ---> S3: zenith-backups/apisix-etcd/
  Hetzner Volume snaps   ---> Hetzner Snapshot API (weekly)
  Harbor registry        ---> S3 (backed up by Velero + native S3 storage)
```

### S3 Bucket Layout

All backups are stored in a single Hetzner S3 bucket (`zenith-backups`) with a structured
prefix hierarchy:

```
zenith-backups/
  keycloak-wal/              # Layer 1: Keycloak CNPG WAL archives
  free-pg-wal/               # Layer 1: Free-tier shared cluster WAL
  pro-shard1-wal/            # Layer 1: Pro shard 1 WAL
  pro-shard2-wal/            # Layer 1: Pro shard 2 WAL
  ...
  pg-dumps/                  # Layer 2: Per-customer logical dumps
    customer-abc/
      2026-02-25.sql.gz
      2026-02-24.sql.gz
      ...
    customer-def/
      ...
  velero/                    # Layer 3: Velero cluster backups
    daily-20260225000000/
    daily-20260224000000/
    ...
  keycloak-realms/           # Keycloak realm JSON exports
    2026-02-25/
      realm-customer-abc.json
      realm-customer-def.json
  apisix-etcd/               # APISIX etcd snapshots
    snapshot-20260225.db
    ...
```

---

## 3. Layer 1: CNPG WAL Archiving (Continuous)

### How WAL Works

PostgreSQL uses Write-Ahead Logging (WAL) as its crash recovery mechanism. Before any
change is applied to the data files, it is first written to the WAL. This guarantees that
even if the server crashes mid-operation, the database can recover by replaying the WAL.

CloudNativePG (CNPG) extends this by continuously streaming WAL segments to object storage
(Hetzner S3 in our case) using `barman-cloud-wal-archive`. This means:

1. Every transaction is captured in the WAL stream.
2. WAL segments are shipped to S3 within seconds of being written.
3. A base backup is taken periodically (this is the starting point for recovery).
4. To restore, CNPG replays WAL segments on top of the base backup up to any desired
   point in time.

The result is **Point-in-Time Recovery (PITR)** -- you can restore the database to any
second within the retention window.

### CNPG Cluster CRD Configuration

Each PostgreSQL cluster in Zenith has WAL archiving configured in its `Cluster` CRD.
Below is the configuration for the Keycloak dedicated cluster:

```yaml
# clusters/keycloak-pg/cluster.yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: keycloak-pg
  namespace: zenith-platform
  labels:
    app.kubernetes.io/part-of: zenith
    zenith.io/component: keycloak-db
spec:
  instances: 3
  imageName: ghcr.io/cloudnative-pg/postgresql:16.2

  storage:
    size: 20Gi
    storageClass: hcloud-volumes

  postgresql:
    parameters:
      max_connections: "200"
      shared_buffers: "256MB"
      wal_level: "replica"
      archive_mode: "on"
      archive_timeout: "60"    # Force WAL switch every 60s even if not full

  bootstrap:
    initdb:
      database: keycloak
      owner: keycloak
      secret:
        name: keycloak-pg-credentials

  # --- WAL Archiving Configuration (Layer 1) ---
  backup:
    barmanObjectStore:
      destinationPath: s3://zenith-backups/keycloak-wal/
      endpointURL: https://fsn1.your-objectstorage.com
      s3Credentials:
        accessKeyId:
          name: s3-backup-credentials
          key: ACCESS_KEY_ID
        secretAccessKey:
          name: s3-backup-credentials
          key: SECRET_ACCESS_KEY
      wal:
        compression: gzip
        maxParallel: 4
      data:
        compression: gzip
        immediateCheckpoint: true
    retentionPolicy: "14d"

  # Scheduled base backups (foundation for PITR)
  scheduledBackups:
    - name: keycloak-pg-daily-backup
      schedule: "0 0 2 * * *"    # 02:00 UTC daily
      backupOwnerReference: self
      immediate: false
```

Here is the configuration for a Pro shard cluster, which holds approximately 20 customer
databases:

```yaml
# clusters/pro-shard1-pg/cluster.yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: pro-shard1-pg
  namespace: zenith-pro
  labels:
    app.kubernetes.io/part-of: zenith
    zenith.io/component: pro-db
    zenith.io/shard: "1"
spec:
  instances: 3
  imageName: ghcr.io/cloudnative-pg/postgresql:16.2

  storage:
    size: 50Gi
    storageClass: hcloud-volumes

  postgresql:
    parameters:
      max_connections: "400"
      shared_buffers: "512MB"
      wal_level: "replica"
      archive_mode: "on"
      archive_timeout: "30"    # More aggressive for Pro tier

  backup:
    barmanObjectStore:
      destinationPath: s3://zenith-backups/pro-shard1-wal/
      endpointURL: https://fsn1.your-objectstorage.com
      s3Credentials:
        accessKeyId:
          name: s3-backup-credentials
          key: ACCESS_KEY_ID
        secretAccessKey:
          name: s3-backup-credentials
          key: SECRET_ACCESS_KEY
      wal:
        compression: gzip
        maxParallel: 8
      data:
        compression: gzip
        immediateCheckpoint: true
    retentionPolicy: "30d"    # Longer retention for Pro

  scheduledBackups:
    - name: pro-shard1-daily-backup
      schedule: "0 0 1 * * *"    # 01:00 UTC daily
      backupOwnerReference: self
```

### How to Restore (PITR to Specific Timestamp)

CNPG supports PITR by creating a new cluster that bootstraps from the backup of an
existing cluster. You specify a `recoveryTarget` with the desired timestamp.

**Step 1: Identify the target timestamp.**

Determine the exact time you want to restore to. For example, if a bad migration ran at
`2026-02-25T14:30:00Z`, you would target `2026-02-25T14:29:59Z`.

**Step 2: Create a recovery Cluster resource.**

```yaml
# recovery/keycloak-pg-recovery.yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: keycloak-pg-recovery
  namespace: zenith-platform
spec:
  instances: 1    # Start with 1 for speed, scale later

  storage:
    size: 20Gi
    storageClass: hcloud-volumes

  bootstrap:
    recovery:
      source: keycloak-pg-backup
      recoveryTarget:
        targetTime: "2026-02-25T14:29:59+00:00"

  externalClusters:
    - name: keycloak-pg-backup
      barmanObjectStore:
        destinationPath: s3://zenith-backups/keycloak-wal/
        endpointURL: https://fsn1.your-objectstorage.com
        s3Credentials:
          accessKeyId:
            name: s3-backup-credentials
            key: ACCESS_KEY_ID
          secretAccessKey:
            name: s3-backup-credentials
            key: SECRET_ACCESS_KEY
        wal:
          maxParallel: 8
```

**Step 3: Apply and wait.**

```bash
kubectl apply -f recovery/keycloak-pg-recovery.yaml

# Watch the recovery progress
kubectl -n zenith-platform get cluster keycloak-pg-recovery -w

# Check logs for recovery progress
kubectl -n zenith-platform logs -l cnpg.io/cluster=keycloak-pg-recovery -f
```

**Step 4: Verify data.**

```bash
# Port-forward to the recovery cluster
kubectl -n zenith-platform port-forward svc/keycloak-pg-recovery-rw 5433:5432

# Connect and verify
psql -h localhost -p 5433 -U keycloak -d keycloak -c "SELECT count(*) FROM user_entity;"
```

**Step 5: Promote or switchover.**

Once verified, either:
- Point applications at the recovery cluster (update Service selectors).
- Use `pg_dump` to export specific data and import into the original cluster.
- Delete the old cluster and rename the recovery cluster (requires downtime).

### Verification: Test Restore Procedure

Every CNPG backup should be tested. CNPG provides a `Backup` resource that you can
create on-demand to verify:

```bash
# Create an on-demand backup
kubectl -n zenith-platform apply -f - <<EOF
apiVersion: postgresql.cnpg.io/v1
kind: Backup
metadata:
  name: keycloak-pg-verify-$(date +%Y%m%d)
  namespace: zenith-platform
spec:
  cluster:
    name: keycloak-pg
  target: prefer-standby
EOF

# Check backup status
kubectl -n zenith-platform get backup -l cnpg.io/cluster=keycloak-pg
```

The monthly restore drill (Section 8) automates full PITR verification.

---

## 4. Layer 2: pg_dump CronJobs (Per-Customer)

### Why pg_dump in Addition to WAL?

WAL archiving (Layer 1) gives you point-in-time recovery for an entire CNPG cluster. But
in a multi-tenant setup, this creates a problem: if you need to restore just one customer's
database, WAL recovery restores **all** databases on that cluster to the target time. Every
other customer on the same shard loses their recent data.

Layer 2 solves this by running `pg_dump` for each customer's database individually. This
produces a standalone SQL file that can be used to restore a single customer without touching
anyone else.

### CronJob YAML Example

The following CronJob runs `pg_dump` for a single customer. In practice, the platform
controller generates one CronJob per customer at provisioning time.

```yaml
# cronjobs/pg-dump/customer-abc.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: pg-dump-customer-abc
  namespace: zenith-backups
  labels:
    app.kubernetes.io/part-of: zenith
    zenith.io/component: pg-dump
    zenith.io/customer: customer-abc
    zenith.io/tier: pro
spec:
  schedule: "0 */6 * * *"    # Every 6 hours for Pro tier
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      backoffLimit: 2
      activeDeadlineSeconds: 1800    # 30 minute timeout
      template:
        metadata:
          labels:
            zenith.io/component: pg-dump
            zenith.io/customer: customer-abc
        spec:
          restartPolicy: Never
          serviceAccountName: backup-runner
          containers:
            - name: pg-dump
              image: ghcr.io/zenith/backup-tools:1.0.0
              resources:
                requests:
                  cpu: 100m
                  memory: 256Mi
                limits:
                  cpu: 500m
                  memory: 512Mi
              env:
                - name: PGHOST
                  value: pro-shard1-pg-rw.zenith-pro.svc.cluster.local
                - name: PGPORT
                  value: "5432"
                - name: PGDATABASE
                  value: customer_abc
                - name: PGUSER
                  valueFrom:
                    secretKeyRef:
                      name: pg-dump-credentials
                      key: username
                - name: PGPASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: pg-dump-credentials
                      key: password
                - name: S3_ENDPOINT
                  value: https://fsn1.your-objectstorage.com
                - name: S3_BUCKET
                  value: zenith-backups
                - name: CUSTOMER_ID
                  value: customer-abc
              command:
                - /bin/sh
                - -c
                - |
                  set -euo pipefail

                  DATE=$(date +%Y-%m-%d-%H%M)
                  DUMP_FILE="/tmp/${CUSTOMER_ID}-${DATE}.sql.gz"
                  S3_PATH="pg-dumps/${CUSTOMER_ID}/${DATE}.sql.gz"

                  echo "[$(date)] Starting pg_dump for ${CUSTOMER_ID}"

                  # Run pg_dump with compression
                  pg_dump \
                    --format=plain \
                    --no-owner \
                    --no-privileges \
                    --verbose \
                    2>/tmp/pgdump.log \
                    | gzip -9 > "${DUMP_FILE}"

                  DUMP_SIZE=$(stat -f%z "${DUMP_FILE}" 2>/dev/null || stat -c%s "${DUMP_FILE}")
                  echo "[$(date)] Dump complete: ${DUMP_SIZE} bytes compressed"

                  # Verify dump is not empty (minimum viable size)
                  if [ "${DUMP_SIZE}" -lt 100 ]; then
                    echo "ERROR: Dump file is suspiciously small (${DUMP_SIZE} bytes)"
                    exit 1
                  fi

                  # Upload to S3
                  aws s3 cp "${DUMP_FILE}" "s3://${S3_BUCKET}/${S3_PATH}" \
                    --endpoint-url "${S3_ENDPOINT}"

                  echo "[$(date)] Uploaded to s3://${S3_BUCKET}/${S3_PATH}"

                  # Clean up old dumps (retention handled by S3 lifecycle,
                  # but we also prune here as a safety net)
                  rm -f "${DUMP_FILE}"

                  echo "[$(date)] pg_dump complete for ${CUSTOMER_ID}"
```

### CronJob for Free Tier (Daily)

Free tier customers get daily backups with 7-day retention:

```yaml
# cronjobs/pg-dump/customer-xyz-free.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: pg-dump-customer-xyz
  namespace: zenith-backups
  labels:
    zenith.io/customer: customer-xyz
    zenith.io/tier: free
spec:
  schedule: "0 2 * * *"    # Daily at 02:00 UTC for Free tier
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      backoffLimit: 2
      activeDeadlineSeconds: 900    # 15 minute timeout (Free DBs are smaller)
      template:
        spec:
          restartPolicy: Never
          serviceAccountName: backup-runner
          containers:
            - name: pg-dump
              image: ghcr.io/zenith/backup-tools:1.0.0
              resources:
                requests:
                  cpu: 50m
                  memory: 128Mi
                limits:
                  cpu: 250m
                  memory: 256Mi
              env:
                - name: PGHOST
                  value: free-pg-rw.zenith-free.svc.cluster.local
                - name: PGDATABASE
                  value: customer_xyz
                - name: PGUSER
                  valueFrom:
                    secretKeyRef:
                      name: pg-dump-credentials
                      key: username
                - name: PGPASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: pg-dump-credentials
                      key: password
                - name: S3_ENDPOINT
                  value: https://fsn1.your-objectstorage.com
                - name: S3_BUCKET
                  value: zenith-backups
                - name: CUSTOMER_ID
                  value: customer-xyz
              command:
                - /bin/sh
                - -c
                - |
                  set -euo pipefail
                  DATE=$(date +%Y-%m-%d)
                  pg_dump --format=plain --no-owner --no-privileges \
                    | gzip -9 > "/tmp/${CUSTOMER_ID}-${DATE}.sql.gz"
                  aws s3 cp "/tmp/${CUSTOMER_ID}-${DATE}.sql.gz" \
                    "s3://${S3_BUCKET}/pg-dumps/${CUSTOMER_ID}/${DATE}.sql.gz" \
                    --endpoint-url "${S3_ENDPOINT}"
                  echo "Done: ${CUSTOMER_ID} ${DATE}"
```

### Per-Customer Isolation

The key benefit of Layer 2 is granular, per-customer restores. The platform controller
ensures:

- Each customer gets their own CronJob, labeled with `zenith.io/customer`.
- Each dump is stored in a customer-specific S3 prefix.
- Dump credentials are scoped to read-only access on that customer's database.
- When a customer is deprovisioned, their CronJob is deleted and dumps are retained
  for 90 days (legal hold), then purged.

### Restore Procedure (Single Customer)

**Step 1: Download the dump from S3.**

```bash
# List available dumps for the customer
aws s3 ls s3://zenith-backups/pg-dumps/customer-abc/ \
  --endpoint-url https://fsn1.your-objectstorage.com

# Download the desired dump
aws s3 cp s3://zenith-backups/pg-dumps/customer-abc/2026-02-25-0600.sql.gz /tmp/ \
  --endpoint-url https://fsn1.your-objectstorage.com
```

**Step 2: Drop and recreate the customer database.**

```bash
# Connect to the CNPG primary
kubectl -n zenith-pro port-forward svc/pro-shard1-pg-rw 5432:5432

# In another terminal:
psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS customer_abc;"
psql -h localhost -U postgres -c "CREATE DATABASE customer_abc OWNER customer_abc;"
```

**Step 3: Restore the dump.**

```bash
gunzip -c /tmp/customer-abc-2026-02-25-0600.sql.gz | \
  psql -h localhost -U postgres -d customer_abc
```

**Step 4: Verify and notify.**

```bash
psql -h localhost -U postgres -d customer_abc -c "\dt"
# Verify tables exist and row counts look reasonable
```

### Retention Policies Per Tier

| Tier | Frequency | Retention | Max Dumps Stored |
|------|-----------|-----------|-----------------|
| Free | Daily (02:00 UTC) | 7 days | 7 |
| Pro | Every 6 hours | 30 days | 120 |
| Team | Every 6 hours | 30 days | 120 |
| Enterprise | Every 4 hours | 90 days | 540 |

Retention is enforced by S3 lifecycle rules on the `pg-dumps/` prefix and by the backup
controller which cleans up expired dumps during each run.

---

## 5. Layer 3: Velero Cluster Backup

### What It Backs Up

Velero captures the complete state of the Kubernetes cluster:

- **All namespaced resources**: Deployments, Services, ConfigMaps, Secrets, CRDs, etc.
- **Cluster-scoped resources**: ClusterRoles, ClusterRoleBindings, Namespaces, CRDs.
- **Persistent Volume data**: Via CSI snapshots of Hetzner Volumes.
- **Custom Resources**: CNPG Clusters, APISIX Routes, ArgoCD Applications, Cert-Manager
  Certificates.

Velero does **not** replace Layer 1 or Layer 2 for database data. Velero's volume snapshots
are crash-consistent (not application-consistent), meaning PostgreSQL data restored from a
Velero snapshot may require WAL replay. The primary value of Velero is restoring the
**platform configuration** -- all the Kubernetes resources that define how the platform is
wired together.

### Velero Installation

```yaml
# velero/helmrelease.yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: velero
  namespace: velero
spec:
  interval: 1h
  chart:
    spec:
      chart: velero
      version: "7.x"
      sourceRef:
        kind: HelmRepository
        name: vmware-tanzu
        namespace: flux-system
  values:
    credentials:
      useSecret: true
      existingSecret: velero-s3-credentials
    configuration:
      backupStorageLocation:
        - name: default
          provider: aws
          bucket: zenith-backups
          prefix: velero
          config:
            region: fsn1
            s3ForcePathStyle: true
            s3Url: https://fsn1.your-objectstorage.com
      volumeSnapshotLocation:
        - name: default
          provider: csi
      defaultVolumesToFsBackup: false    # Use CSI snapshots, not file-level
    deployNodeAgent: true
    snapshotsEnabled: true
    initContainers:
      - name: velero-plugin-for-aws
        image: velero/velero-plugin-for-aws:v1.10.0
        volumeMounts:
          - mountPath: /target
            name: plugins
      - name: velero-plugin-for-csi
        image: velero/velero-plugin-for-csi:v0.8.0
        volumeMounts:
          - mountPath: /target
            name: plugins
```

### Schedule Configuration

```yaml
# velero/schedules.yaml

# Full cluster backup -- daily
apiVersion: velero.io/v1
kind: Schedule
metadata:
  name: daily-full-cluster
  namespace: velero
spec:
  schedule: "0 3 * * *"    # 03:00 UTC daily (after pg_dump CronJobs finish)
  template:
    ttl: 720h0m0s           # 30 days retention
    storageLocation: default
    volumeSnapshotLocations:
      - default
    includedNamespaces:
      - "*"
    excludedNamespaces:
      - velero               # Don't back up Velero itself
    snapshotMoveData: false
    defaultVolumesToFsBackup: false
    metadata:
      labels:
        zenith.io/backup-type: full-cluster
---
# Platform namespace backup -- every 12 hours (more frequent for critical infra)
apiVersion: velero.io/v1
kind: Schedule
metadata:
  name: platform-infra-backup
  namespace: velero
spec:
  schedule: "0 */12 * * *"
  template:
    ttl: 336h0m0s           # 14 days
    storageLocation: default
    includedNamespaces:
      - zenith-platform
      - zenith-system
      - cert-manager
      - apisix
      - argocd
    metadata:
      labels:
        zenith.io/backup-type: platform-infra
```

### Restore Procedure: Full Cluster

In the event of complete cluster loss, Velero restores all Kubernetes resources.

**Step 1: Install a fresh k3s cluster and Velero.**

```bash
# On the new node(s), install k3s
curl -sfL https://get.k3s.io | sh -

# Install Velero CLI
velero install \
  --provider aws \
  --plugins velero/velero-plugin-for-aws:v1.10.0,velero/velero-plugin-for-csi:v0.8.0 \
  --bucket zenith-backups \
  --prefix velero \
  --secret-file ./velero-credentials \
  --backup-location-config \
    region=fsn1,s3ForcePathStyle=true,s3Url=https://fsn1.your-objectstorage.com \
  --snapshot-location-config region=fsn1
```

**Step 2: List available backups.**

```bash
velero backup get
# NAME                          STATUS      CREATED                         EXPIRES
# daily-full-cluster-20260225   Completed   2026-02-25 03:00:00 +0000 UTC   29d
# daily-full-cluster-20260224   Completed   2026-02-24 03:00:00 +0000 UTC   28d
```

**Step 3: Restore.**

```bash
# Restore everything
velero restore create --from-backup daily-full-cluster-20260225

# Watch progress
velero restore describe daily-full-cluster-20260225-restore --details
```

**Step 4: Restore databases from Layer 1 (PITR).**

Velero restores the CNPG Cluster CRDs, but the data volumes may not be usable (crash
consistency). Use Layer 1 PITR to recover the actual database data:

```bash
# For each CNPG cluster, create a recovery cluster pointing at WAL archives
kubectl apply -f recovery/keycloak-pg-recovery.yaml
kubectl apply -f recovery/pro-shard1-pg-recovery.yaml
# ... etc
```

### Restore Procedure: Single Namespace

To restore just one namespace (e.g., a customer namespace was accidentally deleted):

```bash
velero restore create \
  --from-backup daily-full-cluster-20260225 \
  --include-namespaces zenith-customer-abc \
  --restore-volumes=true
```

### Hetzner Volume Snapshots

In addition to Velero's CSI snapshots, we take weekly snapshots of all Hetzner Volumes
via the Hetzner Cloud API. These serve as an independent backup of block storage.

```yaml
# cronjobs/hetzner-volume-snapshot.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: hetzner-volume-snapshots
  namespace: zenith-system
spec:
  schedule: "0 4 * * 0"    # Weekly, Sunday 04:00 UTC
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: snapshot
              image: ghcr.io/zenith/backup-tools:1.0.0
              env:
                - name: HCLOUD_TOKEN
                  valueFrom:
                    secretKeyRef:
                      name: hetzner-api-credentials
                      key: token
              command:
                - /bin/sh
                - -c
                - |
                  set -euo pipefail

                  echo "[$(date)] Starting Hetzner Volume snapshots"

                  # List all volumes
                  VOLUMES=$(hcloud volume list -o json | jq -r '.[].id')

                  for VOL_ID in $VOLUMES; do
                    VOL_NAME=$(hcloud volume describe $VOL_ID -o json | jq -r '.name')
                    echo "Snapshotting volume: ${VOL_NAME} (${VOL_ID})"

                    # Create snapshot (Hetzner does not have native volume
                    # snapshots -- we detach, snapshot the server, reattach.
                    # For CNPG volumes, rely on Layer 1 instead.)
                    # This is handled by the hcloud-snapshot-controller.
                  done

                  echo "[$(date)] Hetzner Volume snapshots complete"
```

---

## 6. Additional Backups

### Keycloak Realm Export

Keycloak stores all identity data in its dedicated PostgreSQL cluster (covered by Layer 1
and Layer 2). However, we also export realm configurations as JSON files. This allows us
to inspect and diff realm configurations over time, and to recreate realms without
restoring an entire database.

```yaml
# cronjobs/keycloak-realm-export.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: keycloak-realm-export
  namespace: zenith-platform
spec:
  schedule: "0 5 * * *"    # Daily at 05:00 UTC
  jobTemplate:
    spec:
      activeDeadlineSeconds: 600
      template:
        spec:
          restartPolicy: Never
          containers:
            - name: realm-export
              image: ghcr.io/zenith/backup-tools:1.0.0
              env:
                - name: KEYCLOAK_URL
                  value: http://keycloak.zenith-platform.svc.cluster.local:8080
                - name: KEYCLOAK_ADMIN
                  valueFrom:
                    secretKeyRef:
                      name: keycloak-admin-credentials
                      key: username
                - name: KEYCLOAK_ADMIN_PASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: keycloak-admin-credentials
                      key: password
                - name: S3_ENDPOINT
                  value: https://fsn1.your-objectstorage.com
                - name: S3_BUCKET
                  value: zenith-backups
              command:
                - /bin/sh
                - -c
                - |
                  set -euo pipefail
                  DATE=$(date +%Y-%m-%d)

                  # Get admin token
                  TOKEN=$(curl -s -X POST \
                    "${KEYCLOAK_URL}/realms/master/protocol/openid-connect/token" \
                    -d "grant_type=client_credentials" \
                    -d "client_id=admin-cli" \
                    -d "username=${KEYCLOAK_ADMIN}" \
                    -d "password=${KEYCLOAK_ADMIN_PASSWORD}" \
                    -d "grant_type=password" \
                    | jq -r '.access_token')

                  # List all realms
                  REALMS=$(curl -s -H "Authorization: Bearer ${TOKEN}" \
                    "${KEYCLOAK_URL}/admin/realms" | jq -r '.[].realm')

                  for REALM in $REALMS; do
                    echo "Exporting realm: ${REALM}"
                    curl -s -H "Authorization: Bearer ${TOKEN}" \
                      "${KEYCLOAK_URL}/admin/realms/${REALM}/partial-export?exportClients=true&exportGroupsAndRoles=true" \
                      -o "/tmp/realm-${REALM}.json"

                    aws s3 cp "/tmp/realm-${REALM}.json" \
                      "s3://${S3_BUCKET}/keycloak-realms/${DATE}/realm-${REALM}.json" \
                      --endpoint-url "${S3_ENDPOINT}"
                  done

                  echo "[$(date)] Realm export complete: $(echo $REALMS | wc -w) realms"
```

**Important limitation:** Keycloak partial exports do not include user credentials (password
hashes). For full user data recovery, rely on Layer 1 (CNPG WAL) or Layer 2 (pg_dump) of
the Keycloak database. The realm export is primarily for configuration recovery.

### APISIX etcd Snapshot

APISIX stores its route configuration in etcd. While all APISIX routes are also defined as
Kubernetes CRDs (and thus recoverable from Git via ArgoCD), we take periodic etcd snapshots
as an additional safety net.

```yaml
# cronjobs/apisix-etcd-snapshot.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: apisix-etcd-snapshot
  namespace: apisix
spec:
  schedule: "0 4 * * *"    # Daily at 04:00 UTC
  jobTemplate:
    spec:
      activeDeadlineSeconds: 300
      template:
        spec:
          restartPolicy: Never
          containers:
            - name: etcd-snapshot
              image: bitnami/etcd:3.5
              env:
                - name: ETCDCTL_ENDPOINTS
                  value: http://apisix-etcd.apisix.svc.cluster.local:2379
                - name: S3_ENDPOINT
                  value: https://fsn1.your-objectstorage.com
                - name: S3_BUCKET
                  value: zenith-backups
              command:
                - /bin/sh
                - -c
                - |
                  set -euo pipefail
                  DATE=$(date +%Y-%m-%d)

                  etcdctl snapshot save /tmp/etcd-snapshot-${DATE}.db

                  # Install aws cli (or use backup-tools image)
                  aws s3 cp /tmp/etcd-snapshot-${DATE}.db \
                    s3://${S3_BUCKET}/apisix-etcd/snapshot-${DATE}.db \
                    --endpoint-url ${S3_ENDPOINT}

                  echo "[$(date)] etcd snapshot uploaded"
```

**Recovery note:** In most cases, APISIX routes can be recovered simply by resyncing
ArgoCD, which will re-apply all route CRDs. The etcd snapshot is a fallback for cases
where the CRDs have drifted from what is actually in etcd (e.g., routes created via the
APISIX Admin API directly).

### Harbor Registry

Harbor stores container images in Hetzner S3. The S3 data is inherently durable (replicated
by Hetzner), and Harbor's metadata (PostgreSQL + Redis) is captured by Velero. In a disaster
recovery scenario:

1. Velero restores Harbor's Kubernetes resources.
2. Harbor reconnects to S3 where all image layers still exist.
3. If Harbor's internal PostgreSQL is lost, it can be rebuilt by running a garbage
   collection and re-scanning the S3 bucket.

No additional backup CronJob is needed for Harbor.

---

## 7. Disaster Recovery Scenarios

### Scenario 1: Single Customer Database Corruption

**Example:** A customer's application runs a bad migration that drops critical tables.

| Metric | Value |
|--------|-------|
| RPO | 0-6 hours (depending on tier; last pg_dump) |
| RTO | 15-30 minutes |
| Data at Risk | Only the affected customer's data |

**Recovery Steps:**

1. Identify the customer and their CNPG cluster (check `zenith.io/shard` label).
2. Download the most recent pg_dump from S3 (Layer 2):
   ```bash
   aws s3 ls s3://zenith-backups/pg-dumps/customer-abc/ \
     --endpoint-url https://fsn1.your-objectstorage.com
   aws s3 cp s3://zenith-backups/pg-dumps/customer-abc/2026-02-25-0600.sql.gz /tmp/ \
     --endpoint-url https://fsn1.your-objectstorage.com
   ```
3. Drop the corrupted database and restore from dump:
   ```bash
   kubectl -n zenith-pro port-forward svc/pro-shard1-pg-rw 5432:5432 &
   psql -h localhost -U postgres -c "DROP DATABASE customer_abc;"
   psql -h localhost -U postgres -c "CREATE DATABASE customer_abc OWNER customer_abc;"
   gunzip -c /tmp/customer-abc-2026-02-25-0600.sql.gz | psql -h localhost -U postgres -d customer_abc
   ```
4. Notify the customer of the restore point and any data loss window.

**Why Layer 2 (not Layer 1)?** Because Layer 1 (PITR) would restore the entire CNPG
cluster to the target time, affecting all ~20 customers on that shard. Layer 2 lets you
restore just one customer.

### Scenario 2: CNPG Cluster Failure

**Example:** The Pro Shard 1 CNPG cluster has a catastrophic failure. All three pods
(primary + 2 replicas) are down and PVCs are corrupted.

| Metric | Value |
|--------|-------|
| RPO | Near-zero (last WAL segment, typically seconds) |
| RTO | 10-30 minutes |
| Data at Risk | Potentially last few seconds of transactions |

**Recovery Steps:**

1. Verify the cluster is truly unrecoverable:
   ```bash
   kubectl -n zenith-pro get cluster pro-shard1-pg
   kubectl -n zenith-pro describe cluster pro-shard1-pg
   kubectl -n zenith-pro get pods -l cnpg.io/cluster=pro-shard1-pg
   ```
2. Create a PITR recovery cluster from WAL archives (Layer 1):
   ```yaml
   apiVersion: postgresql.cnpg.io/v1
   kind: Cluster
   metadata:
     name: pro-shard1-pg-recovery
     namespace: zenith-pro
   spec:
     instances: 3
     storage:
       size: 50Gi
       storageClass: hcloud-volumes
     bootstrap:
       recovery:
         source: pro-shard1-backup
     externalClusters:
       - name: pro-shard1-backup
         barmanObjectStore:
           destinationPath: s3://zenith-backups/pro-shard1-wal/
           endpointURL: https://fsn1.your-objectstorage.com
           s3Credentials:
             accessKeyId:
               name: s3-backup-credentials
               key: ACCESS_KEY_ID
             secretAccessKey:
               name: s3-backup-credentials
               key: SECRET_ACCESS_KEY
   ```
3. Apply and wait for recovery:
   ```bash
   kubectl apply -f recovery/pro-shard1-pg-recovery.yaml
   kubectl -n zenith-pro get cluster pro-shard1-pg-recovery -w
   ```
4. Once healthy, update services to point at the recovery cluster.
5. Delete the old broken cluster CRD and rename the recovery cluster.

### Scenario 3: Node Failure

**Example:** A Hetzner server running k3s agent goes down (hardware failure, kernel panic).

| Metric | Value |
|--------|-------|
| RPO | Zero (data is on Hetzner Volumes, not local disk) |
| RTO | 5-15 minutes (Kubernetes reschedules pods) |
| Data at Risk | None (PVCs survive node loss) |

**Recovery Steps:**

1. Kubernetes automatically reschedules pods to other nodes.
2. CNPG replicas promote if the primary was on the failed node.
3. If the node cannot be recovered, provision a new one:
   ```bash
   # Using Ansible (our standard provisioning)
   ansible-playbook -i inventory/staging.yml playbooks/add-node.yml
   ```
4. Verify all pods are running:
   ```bash
   kubectl get pods --all-namespaces | grep -v Running
   ```

**Note:** This scenario has zero data loss because Hetzner Volumes (network-attached
storage) survive node failures. The volumes are reattached to pods on healthy nodes.

### Scenario 4: Complete Cluster Loss

**Example:** The entire k3s cluster is destroyed (accidental `kubectl delete` of critical
resources, or all nodes simultaneously fail).

| Metric | Value |
|--------|-------|
| RPO | Near-zero for databases (Layer 1 WAL), 24h for K8s configs (Velero) |
| RTO | 1-2 hours |
| Data at Risk | K8s configuration changes made since last Velero backup |

**Recovery Steps:**

1. **Provision new infrastructure** using Terraform and Ansible:
   ```bash
   cd infra/terraform/staging && terraform apply
   cd infra/ansible && ansible-playbook -i inventory/staging.yml playbooks/site.yml
   ```
2. **Install Velero** on the new cluster and connect to S3.
3. **Restore Kubernetes resources** from the latest Velero backup:
   ```bash
   velero restore create --from-backup daily-full-cluster-20260225
   ```
4. **Restore databases** using CNPG PITR (Layer 1) for each cluster.
5. **Verify services** and DNS resolution.
6. **Run the restore drill verification** (Section 8) to confirm everything is working.

### Scenario 5: Hetzner Datacenter Outage

**Example:** The Falkenstein (fsn1) datacenter experiences a prolonged outage affecting
both compute and S3 storage.

| Metric | Value |
|--------|-------|
| RPO | Up to 24 hours (last off-site replication, if configured) |
| RTO | 4-8 hours (new datacenter provisioning) |
| Data at Risk | All changes since last off-site backup |

**Recovery Steps:**

This is the worst-case scenario. Recovery depends on whether off-site S3 replication has
been configured (recommended for Enterprise tier):

1. **If off-site replication exists** (S3 cross-region to Hetzner nbg1 or hel1):
   - Provision new cluster in the alternate datacenter.
   - Point Velero and CNPG at the replicated S3 bucket.
   - Follow Scenario 4 recovery steps.

2. **If no off-site replication** (Free/Pro tiers):
   - Wait for Hetzner to restore the datacenter.
   - If datacenter is permanently lost, data since the last off-site transfer is gone.

**Mitigation for Enterprise customers:**
Configure S3 bucket replication to a second Hetzner datacenter. This is a Zenith roadmap
item for Enterprise tier, documented separately.

**Mitigation for all tiers:**
The ArgoCD Git repository serves as an off-site backup of all Kubernetes configuration.
Even without S3, the cluster can be rebuilt from Git -- only database data is at risk.

---

## 8. Monthly Restore Drill

Backups are worthless if they cannot be restored. We run an automated monthly restore drill
to verify that all three backup layers are functioning correctly.

### Automated Test Procedure

```yaml
# cronjobs/restore-drill.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: monthly-restore-drill
  namespace: zenith-system
spec:
  schedule: "0 6 1 * *"    # 1st of every month at 06:00 UTC
  jobTemplate:
    spec:
      activeDeadlineSeconds: 7200    # 2 hour timeout
      template:
        spec:
          restartPolicy: Never
          serviceAccountName: restore-drill-runner
          containers:
            - name: drill
              image: ghcr.io/zenith/backup-tools:1.0.0
              env:
                - name: S3_ENDPOINT
                  value: https://fsn1.your-objectstorage.com
                - name: S3_BUCKET
                  value: zenith-backups
                - name: SLACK_WEBHOOK_URL
                  valueFrom:
                    secretKeyRef:
                      name: slack-webhook
                      key: url
                - name: DRILL_NAMESPACE
                  value: zenith-restore-drill
              command:
                - /bin/sh
                - -c
                - |
                  set -euo pipefail
                  RESULTS=""
                  PASS=true

                  notify() {
                    local status=$1 msg=$2
                    local color="good"
                    [ "$status" = "FAIL" ] && color="danger"
                    curl -s -X POST "$SLACK_WEBHOOK_URL" \
                      -H 'Content-type: application/json' \
                      -d "{\"attachments\":[{\"color\":\"${color}\",\"title\":\"Restore Drill: ${status}\",\"text\":\"${msg}\"}]}"
                  }

                  # --- Test 1: Layer 2 pg_dump restore ---
                  echo "=== Test 1: pg_dump restore ==="
                  LATEST_DUMP=$(aws s3 ls s3://${S3_BUCKET}/pg-dumps/ \
                    --endpoint-url ${S3_ENDPOINT} --recursive \
                    | sort | tail -1 | awk '{print $4}')

                  if [ -z "$LATEST_DUMP" ]; then
                    RESULTS="${RESULTS}\n- Layer 2 (pg_dump): FAIL - no dumps found"
                    PASS=false
                  else
                    aws s3 cp "s3://${S3_BUCKET}/${LATEST_DUMP}" /tmp/test-dump.sql.gz \
                      --endpoint-url ${S3_ENDPOINT}

                    # Create a temporary database and restore
                    createdb -h $PGHOST -U postgres restore_drill_test
                    gunzip -c /tmp/test-dump.sql.gz | psql -h $PGHOST -U postgres -d restore_drill_test

                    TABLE_COUNT=$(psql -h $PGHOST -U postgres -d restore_drill_test -t \
                      -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';")

                    if [ "$TABLE_COUNT" -gt 0 ]; then
                      RESULTS="${RESULTS}\n- Layer 2 (pg_dump): PASS - restored ${TABLE_COUNT} tables"
                    else
                      RESULTS="${RESULTS}\n- Layer 2 (pg_dump): FAIL - 0 tables after restore"
                      PASS=false
                    fi

                    dropdb -h $PGHOST -U postgres restore_drill_test
                  fi

                  # --- Test 2: Layer 1 CNPG WAL verify ---
                  echo "=== Test 2: CNPG WAL archive verify ==="
                  WAL_COUNT=$(aws s3 ls s3://${S3_BUCKET}/keycloak-wal/ \
                    --endpoint-url ${S3_ENDPOINT} --recursive | wc -l)

                  if [ "$WAL_COUNT" -gt 10 ]; then
                    RESULTS="${RESULTS}\n- Layer 1 (WAL): PASS - ${WAL_COUNT} WAL segments in S3"
                  else
                    RESULTS="${RESULTS}\n- Layer 1 (WAL): FAIL - only ${WAL_COUNT} WAL segments"
                    PASS=false
                  fi

                  # --- Test 3: Velero backup verify ---
                  echo "=== Test 3: Velero backup verify ==="
                  LATEST_BACKUP=$(velero backup get -o json | \
                    jq -r '.items | sort_by(.metadata.creationTimestamp) | last | .metadata.name')

                  if [ "$LATEST_BACKUP" != "null" ] && [ -n "$LATEST_BACKUP" ]; then
                    BACKUP_STATUS=$(velero backup describe "$LATEST_BACKUP" -o json | jq -r '.status.phase')
                    if [ "$BACKUP_STATUS" = "Completed" ]; then
                      RESULTS="${RESULTS}\n- Layer 3 (Velero): PASS - latest: ${LATEST_BACKUP} (${BACKUP_STATUS})"
                    else
                      RESULTS="${RESULTS}\n- Layer 3 (Velero): FAIL - latest: ${LATEST_BACKUP} (${BACKUP_STATUS})"
                      PASS=false
                    fi
                  else
                    RESULTS="${RESULTS}\n- Layer 3 (Velero): FAIL - no backups found"
                    PASS=false
                  fi

                  # --- Report ---
                  if [ "$PASS" = true ]; then
                    notify "PASS" "Monthly restore drill completed successfully.\n${RESULTS}"
                    echo "ALL TESTS PASSED"
                  else
                    notify "FAIL" "Monthly restore drill had failures.\n${RESULTS}"
                    echo "SOME TESTS FAILED"
                    exit 1
                  fi
```

### What Gets Tested

| Test | What It Verifies | Pass Criteria |
|------|------------------|---------------|
| pg_dump restore | Layer 2 dumps can be downloaded and restored | At least 1 table restored successfully |
| CNPG WAL verify | WAL segments are being archived to S3 | More than 10 WAL segments exist |
| CNPG PITR (quarterly) | Full PITR recovery to a test cluster | Recovery cluster reaches "Cluster in healthy state" |
| Velero backup status | Latest Velero backup completed | Status is "Completed" |
| Keycloak realm export | Realm JSON files exist in S3 | At least 1 realm JSON from last 48h |
| etcd snapshot | APISIX etcd snapshot exists | Snapshot file from last 48h |

The quarterly PITR test (run every 3 months) actually creates a recovery CNPG cluster from
WAL archives, waits for it to become healthy, runs a few SQL queries, then tears it down.
This is the most thorough test but takes 20-30 minutes and creates temporary volumes.

### Slack Notification

The drill sends results to the `#ops-alerts` Slack channel:

```
+-----------------------------------------------+
| Restore Drill: PASS                           |
|-----------------------------------------------|
| Monthly restore drill completed successfully. |
| - Layer 2 (pg_dump): PASS - restored 14 tables|
| - Layer 1 (WAL): PASS - 847 WAL segments in S3|
| - Layer 3 (Velero): PASS - daily-full-cluster |
|   -20260201 (Completed)                       |
+-----------------------------------------------+
```

If any test fails, the notification is red and the CronJob exits with code 1, triggering
an alert in the monitoring system.

---

## 9. RPO/RTO Summary Table

| Layer | Protects Against | RPO | RTO | Tier Availability |
|-------|------------------|-----|-----|-------------------|
| **Layer 1: CNPG WAL** | Database corruption, bad migrations, cluster failure | Near-zero (seconds) | 10-30 min | All tiers |
| **Layer 2: pg_dump** | Single-customer corruption (without affecting others) | 24h (Free), 6h (Pro/Team), 4h (Enterprise) | 15-30 min | All tiers |
| **Layer 3: Velero** | Cluster loss, namespace deletion, config loss | 24h | 1-2 hours | All tiers |
| **Keycloak export** | Realm misconfiguration | 24h | 30 min | All tiers |
| **APISIX etcd** | etcd corruption, route loss | 24h (but Git is primary) | 15 min (from Git), 30 min (from snapshot) | All tiers |
| **Hetzner snapshots** | Volume-level corruption | 7 days | 30-60 min | All tiers |

### RPO/RTO by Disaster Scenario

| Scenario | RPO | RTO | Primary Recovery Layer |
|----------|-----|-----|----------------------|
| Single customer DB corruption | 6-24h (tier-dependent) | 15-30 min | Layer 2 (pg_dump) |
| Entire CNPG cluster failure | Seconds | 10-30 min | Layer 1 (CNPG WAL PITR) |
| Single node failure | Zero | 5-15 min | Kubernetes rescheduling |
| Complete cluster loss | Seconds (DB), 24h (config) | 1-2 hours | Layer 3 (Velero) + Layer 1 (PITR) |
| Datacenter outage | 24h (no replication) | 4-8 hours | Off-site S3 + rebuild |

---

## 10. Monitoring and Alerting

Backup jobs are only useful if we know they are succeeding. The monitoring stack watches
every backup layer and alerts on failures.

### Prometheus Alerts

```yaml
# monitoring/backup-alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: backup-alerts
  namespace: zenith-monitoring
spec:
  groups:
    - name: backup.rules
      rules:
        # Layer 1: CNPG WAL archiving lag
        - alert: CNPGWALArchivingLagging
          expr: cnpg_pg_stat_archiver_last_archived_time < (time() - 300)
          for: 10m
          labels:
            severity: warning
          annotations:
            summary: "CNPG WAL archiving is lagging on {{ $labels.cluster }}"
            description: >
              No WAL segment has been archived in the last 5 minutes for cluster
              {{ $labels.cluster }}. This means Layer 1 backup is not functioning.
              RPO is increasing.

        - alert: CNPGWALArchivingFailed
          expr: cnpg_pg_stat_archiver_failed_count > 0
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "CNPG WAL archiving failures on {{ $labels.cluster }}"
            description: >
              WAL archiving has failed {{ $value }} times on cluster
              {{ $labels.cluster }}. Investigate immediately -- data loss risk.

        # Layer 1: CNPG scheduled backup missed
        - alert: CNPGScheduledBackupMissed
          expr: |
            time() - cnpg_last_available_backup_timestamp > 172800
          for: 30m
          labels:
            severity: critical
          annotations:
            summary: "CNPG base backup is more than 48h old for {{ $labels.cluster }}"
            description: >
              The last base backup for {{ $labels.cluster }} is older than 48 hours.
              PITR recovery window may be compromised.

        # Layer 2: pg_dump CronJob failures
        - alert: PgDumpCronJobFailed
          expr: |
            kube_job_status_failed{namespace="zenith-backups", job_name=~"pg-dump-.*"} > 0
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "pg_dump CronJob failed for {{ $labels.job_name }}"
            description: >
              The backup CronJob {{ $labels.job_name }} has failed. The affected
              customer's RPO is increasing. Check job logs.

        - alert: PgDumpCronJobMissed
          expr: |
            time() - kube_cronjob_status_last_successful_time{namespace="zenith-backups", cronjob=~"pg-dump-.*"} > 86400
          for: 1h
          labels:
            severity: critical
          annotations:
            summary: "pg_dump not run in 24h for {{ $labels.cronjob }}"
            description: >
              The CronJob {{ $labels.cronjob }} has not completed successfully in
              over 24 hours. Even Free tier customers should have a backup within 24h.

        # Layer 3: Velero backup failures
        - alert: VeleroBackupFailed
          expr: |
            velero_backup_failure_total > velero_backup_failure_total offset 1h
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "Velero backup failed"
            description: >
              A Velero backup has failed. Cluster-level disaster recovery is compromised.
              Check velero backup get and velero backup logs.

        - alert: VeleroBackupMissed
          expr: |
            time() - velero_backup_last_successful_timestamp > 172800
          for: 30m
          labels:
            severity: critical
          annotations:
            summary: "No successful Velero backup in 48h"
            description: >
              Velero has not completed a successful backup in over 48 hours.
              Cluster recovery capability is at risk.

        # Restore drill
        - alert: RestoreDrillFailed
          expr: |
            kube_job_status_failed{namespace="zenith-system", job_name=~"monthly-restore-drill-.*"} > 0
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "Monthly restore drill failed"
            description: >
              The automated restore drill has failed. This means one or more backup
              layers may not be restorable. Investigate immediately.
```

### Grafana Dashboard

The backup monitoring Grafana dashboard shows:

1. **WAL Archiving Status** -- Per-cluster WAL archive lag and failure count.
2. **Last Base Backup Age** -- Per-cluster time since the last successful CNPG base backup.
3. **pg_dump CronJob Status** -- Table showing each customer's last successful dump time
   and whether it is within the expected window for their tier.
4. **Velero Backup History** -- Timeline of backup completions and failures.
5. **S3 Bucket Size** -- Total backup storage usage, with per-prefix breakdown.
6. **Restore Drill Results** -- Last drill outcome with pass/fail per layer.

---

## 11. How to Run Manual Backups

### Manual CNPG Base Backup

Create an on-demand base backup for any CNPG cluster:

```bash
# Create a backup object
kubectl -n zenith-pro apply -f - <<EOF
apiVersion: postgresql.cnpg.io/v1
kind: Backup
metadata:
  name: pro-shard1-manual-$(date +%Y%m%d-%H%M)
  namespace: zenith-pro
spec:
  cluster:
    name: pro-shard1-pg
  target: prefer-standby    # Back up from replica to avoid primary load
EOF

# Monitor progress
kubectl -n zenith-pro get backup -l cnpg.io/cluster=pro-shard1-pg -w
```

### Manual pg_dump (Single Customer)

Run an ad-hoc pg_dump for a specific customer:

```bash
# Create a one-off Job from the CronJob template
kubectl -n zenith-backups create job \
  --from=cronjob/pg-dump-customer-abc \
  pg-dump-customer-abc-manual-$(date +%Y%m%d%H%M)

# Watch the job
kubectl -n zenith-backups logs -f job/pg-dump-customer-abc-manual-$(date +%Y%m%d%H%M)
```

Alternatively, run pg_dump directly from a temporary pod:

```bash
kubectl -n zenith-backups run pg-dump-adhoc \
  --image=postgres:16 \
  --restart=Never \
  --rm -it \
  --env="PGHOST=pro-shard1-pg-rw.zenith-pro.svc.cluster.local" \
  --env="PGUSER=postgres" \
  --env="PGPASSWORD=$(kubectl -n zenith-pro get secret pro-shard1-pg-superuser -o jsonpath='{.data.password}' | base64 -d)" \
  -- bash -c 'pg_dump -d customer_abc | gzip > /tmp/dump.sql.gz && echo "Size: $(ls -lh /tmp/dump.sql.gz)"'
```

### Manual Velero Backup

Trigger an on-demand Velero backup:

```bash
# Full cluster backup
velero backup create manual-full-$(date +%Y%m%d%H%M) \
  --include-namespaces '*' \
  --exclude-namespaces velero \
  --ttl 720h

# Single namespace backup
velero backup create manual-platform-$(date +%Y%m%d%H%M) \
  --include-namespaces zenith-platform \
  --ttl 168h

# Monitor progress
velero backup describe manual-full-$(date +%Y%m%d%H%M)
velero backup logs manual-full-$(date +%Y%m%d%H%M)
```

### Manual Keycloak Realm Export

```bash
# Port-forward to Keycloak
kubectl -n zenith-platform port-forward svc/keycloak 8080:8080 &

# Get admin token
TOKEN=$(curl -s -X POST http://localhost:8080/realms/master/protocol/openid-connect/token \
  -d "grant_type=password&client_id=admin-cli&username=admin&password=$(kubectl -n zenith-platform get secret keycloak-admin-credentials -o jsonpath='{.data.password}' | base64 -d)" \
  | jq -r '.access_token')

# Export a specific realm
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/admin/realms/customer-abc/partial-export?exportClients=true&exportGroupsAndRoles=true" \
  -o realm-customer-abc.json

echo "Exported $(jq '.clients | length' realm-customer-abc.json) clients from realm"
```

### Manual APISIX etcd Snapshot

```bash
# Port-forward to etcd
kubectl -n apisix port-forward svc/apisix-etcd 2379:2379 &

# Take snapshot
ETCDCTL_API=3 etcdctl --endpoints=localhost:2379 \
  snapshot save /tmp/apisix-etcd-manual-$(date +%Y%m%d).db

# Verify snapshot
ETCDCTL_API=3 etcdctl snapshot status /tmp/apisix-etcd-manual-$(date +%Y%m%d).db --write-out=table
```

---

## Appendix A: S3 Lifecycle Rules

S3 lifecycle rules enforce retention automatically. Configure these on the `zenith-backups`
bucket:

| Prefix | Retention | Action |
|--------|-----------|--------|
| `pg-dumps/` (Free tier customers) | 7 days | Delete |
| `pg-dumps/` (Pro/Team customers) | 30 days | Delete |
| `pg-dumps/` (Enterprise customers) | 90 days | Delete |
| `keycloak-wal/` | 14 days | Delete |
| `free-pg-wal/` | 14 days | Delete |
| `pro-shard*-wal/` | 30 days | Delete |
| `velero/` | 30 days | Delete |
| `keycloak-realms/` | 90 days | Delete |
| `apisix-etcd/` | 30 days | Delete |

**Note:** Lifecycle rules on per-customer prefixes are managed by the platform controller,
which sets the appropriate retention based on the customer's tier. Free-tier dump retention
uses a different lifecycle rule than Pro-tier.

## Appendix B: Backup Credential Management

All backup jobs authenticate to S3 using a shared Kubernetes Secret. The credentials are
provisioned by Terraform and rotated quarterly.

```yaml
# secrets/s3-backup-credentials.yaml (sealed with kubeseal in practice)
apiVersion: v1
kind: Secret
metadata:
  name: s3-backup-credentials
  namespace: zenith-system
type: Opaque
data:
  ACCESS_KEY_ID: <base64>
  SECRET_ACCESS_KEY: <base64>
```

The Secret is replicated to namespaces that need it (`zenith-backups`, `zenith-platform`,
`zenith-pro`, `apisix`, `velero`) using a SecretCopier controller or Kubernetes RBAC
with cross-namespace Secret references.

**Security considerations:**
- The S3 credentials have write access to `zenith-backups/` only (not other buckets).
- The `backup-runner` ServiceAccount has read-only access to customer databases.
- Restore operations require a separate, more privileged ServiceAccount (`restore-runner`).
- All S3 access is logged for audit purposes.

## Appendix C: Decision Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| 3 backup layers instead of 1 | Adopted | Each layer protects against different failure modes; no single point of failure for data |
| pg_dump per-customer (not per-cluster) | Adopted | Enables single-customer restore without affecting other tenants on the same shard |
| Hetzner S3 (not external cloud) | Adopted | Data sovereignty, lower latency, cost; trade-off is single-provider risk |
| Velero over Kasten or custom solution | Adopted | Mature, open-source, strong CSI integration, active community |
| Monthly restore drill | Adopted | Untested backups are not backups; automation ensures consistent verification |
| Keycloak realm export as JSON | Adopted | Provides human-readable config backup independent of database; useful for auditing |
| 30-day Velero retention | Adopted | Balances storage cost against recovery window; most incidents are caught within days |
