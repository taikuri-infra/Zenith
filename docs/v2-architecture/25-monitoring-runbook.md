# 25 — Monitoring & Alerting Runbook

> **Purpose:** When an alert fires or something looks wrong in Grafana, this runbook tells you exactly what to do — step by step.
> **Audience:** Any developer on-call or investigating a production/staging issue.
> **Last Updated:** 2026-03-03
> **Related:** [08-observability.md](./08-observability.md) (full observability architecture), [07-backup-disaster-recovery.md](./07-backup-disaster-recovery.md) (disaster recovery), [SYSTEM-MAP.md](./SYSTEM-MAP.md) (system overview)

---

## Table of Contents

1. [Alert Overview](#1-alert-overview)
2. [Where to Look First](#2-where-to-look-first)
3. [Platform Health Alerts](#3-platform-health-alerts)
4. [Application Alerts](#4-application-alerts)
5. [Database Alerts](#5-database-alerts)
6. [Node Alerts](#6-node-alerts)
7. [Grafana Dashboards Guide](#7-grafana-dashboards-guide)
8. [Log Investigation (Loki)](#8-log-investigation-loki)
9. [Trace Investigation (Tempo)](#9-trace-investigation-tempo)
10. [Network Investigation (Hubble)](#10-network-investigation-hubble)

---

## 1. Alert Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│             ALERT ARCHITECTURE                                           │
│                                                                          │
│  Prometheus                                                              │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Scrapes metrics every 15s from:                                   │ │
│  │  • zenith-api (:8080/metrics)                                     │ │
│  │  • Traefik (:8080/metrics)                                        │ │
│  │  • APISIX (:9091/apisix/prometheus/metrics)                       │ │
│  │  • CNPG PostgreSQL (:9187/metrics)                                │ │
│  │  • Node Exporter (system metrics)                                 │ │
│  │  • kube-state-metrics (K8s object state)                          │ │
│  │  • kubelet cAdvisor (container metrics)                           │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │ evaluate PrometheusRules                      │
│                          ▼                                               │
│  Alertmanager                                                            │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Receives fired alerts from Prometheus                             │ │
│  │  Routes to configured receivers (Slack, PagerDuty, email)         │ │
│  │                                                                    │ │
│  │  NOTE: Receivers are NOT yet configured for staging.              │ │
│  │  Currently alerts are visible only in Grafana Alerting UI.        │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Grafana (for viewing alerts)                                            │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  URL: https://grafana.stage.freezenith.com                        │ │
│  │  Go to: Alerting → Alert rules                                   │ │
│  │  See all firing, pending, and OK alerts                           │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

### All Configured Alerts

| Alert | Severity | Threshold | For | Category |
|-------|----------|-----------|-----|----------|
| `ZenithAPIDown` | Critical | API unreachable | 2 min | Platform |
| `ZenithOperatorDown` | Critical | Operator unreachable | 2 min | Platform |
| `ZenithAuthDown` | Critical | Auth service unreachable | 2 min | Platform |
| `AppHighCPU` | Warning | Pod CPU > 80% | 5 min | Application |
| `AppHighMemory` | Warning | Pod memory > 85% | 5 min | Application |
| `AppCrashLooping` | Critical | > 5 restarts in 1h | 5 min | Application |
| `AppNotReady` | Warning | Pod not ready | 5 min | Application |
| `DeploymentReplicasMismatch` | Warning | Desired ≠ available | 10 min | Application |
| `DatabaseHighConnectionCount` | Warning | > 80 connections | 5 min | Database |
| `DatabaseStorageFull` | Critical | PVC > 85% full | 5 min | Database |
| `DatabaseBackupFailed` | Critical | No backup in 24h | 1 hour | Database |
| `NodeHighCPU` | Warning | CPU > 85% | 10 min | Node |
| `NodeHighMemory` | Critical | Memory > 90% | 5 min | Node |
| `NodeDiskPressure` | Warning | Disk > 85% | 5 min | Node |

---

## 2. Where to Look First

```
┌─────────────────────────────────────────────────────────────────────────┐
│             TRIAGE FLOWCHART                                             │
│                                                                          │
│  Something is wrong!                                                     │
│        │                                                                 │
│        ▼                                                                 │
│  Is it a specific customer app?                                         │
│  ┌─── YES ──▶ Check their namespace:                                   │
│  │            kubectl -n zenith-<customer> get pods                     │
│  │            kubectl -n zenith-<customer> logs deploy/<app>            │
│  │            → Go to Section 4 (Application Alerts)                   │
│  │                                                                      │
│  └─── NO ───▶ Is it platform-wide?                                     │
│               ┌─── YES ──▶ Check core services:                        │
│               │            kubectl -n zenith-staging get pods           │
│               │            kubectl -n apisix get pods                   │
│               │            kubectl -n argocd get pods                   │
│               │            → Go to Section 3 (Platform Alerts)         │
│               │                                                         │
│               └─── MAYBE ─▶ Check the node:                            │
│                             kubectl get nodes                           │
│                             kubectl top nodes                           │
│                             → Go to Section 6 (Node Alerts)            │
│                                                                          │
│  ALWAYS CHECK THESE:                                                     │
│  1. Grafana dashboards: https://grafana.stage.freezenith.com           │
│  2. ArgoCD status:      https://argocd.stage.freezenith.com            │
│  3. kubectl overview:   kubectl get pods -A | grep -v Running          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Platform Health Alerts

### ZenithAPIDown (Critical)

**Meaning:** The main API is unreachable for 2+ minutes.
**Impact:** No user can login, register, deploy, or manage apps.

```bash
# Step 1: Check pod status
kubectl -n zenith-staging get pods -l app=zenith-api
# Is it Running? CrashLoopBackOff? OOMKilled?

# Step 2: Check events
kubectl -n zenith-staging describe deploy zenith-api
# Look at Events section — scheduling failures? image pull errors?

# Step 3: Check logs
kubectl -n zenith-staging logs deploy/zenith-api --tail=100
# Look for panic, fatal, connection refused

# Step 4: Check if DB is reachable
kubectl -n zenith-staging exec deploy/zenith-api -- \
  wget -qO- http://localhost:8080/api/v1/health
# If health check passes internally, the issue is networking (APISIX/Traefik)

# Step 5: Check APISIX → API routing
kubectl -n apisix get pods
kubectl -n apisix logs deploy/apisix --tail=50

# Step 6: Restart if needed
kubectl -n zenith-staging rollout restart deploy/zenith-api
```

### ZenithOperatorDown (Critical)

**Meaning:** The Kubernetes operator is unreachable. Tenant CRD reconciliation stops.

```bash
# Step 1: Check operator pod
kubectl -n zenith-staging get pods -l app=zenith-operator

# Step 2: Check logs for reconciliation errors
kubectl -n zenith-staging logs deploy/zenith-operator --tail=100

# Step 3: Restart
kubectl -n zenith-staging rollout restart deploy/zenith-operator
```

---

## 4. Application Alerts

### AppCrashLooping (Critical)

**Meaning:** A pod has restarted more than 5 times in the last hour.

```bash
# Step 1: Find the crashing pod
kubectl get pods -A --field-selector=status.phase!=Running | grep -v Completed

# Step 2: Check why it's crashing
kubectl -n <namespace> describe pod <pod-name>
# Look at: Last State → Reason (OOMKilled, Error, etc.)
#          Events → pull errors, scheduling failures

# Step 3: Check logs from the PREVIOUS crash
kubectl -n <namespace> logs <pod-name> --previous

# Common causes:
# • OOMKilled → increase memory limits in Helm values
# • Error (exit code 1) → application bug, check logs
# • ImagePullBackOff → wrong image tag or registry auth issue
# • Pending → not enough resources on node, check node capacity
```

### AppHighCPU / AppHighMemory (Warning)

**Meaning:** A pod is using too many resources.

```bash
# Step 1: See current usage
kubectl top pods -n <namespace> --sort-by=cpu
kubectl top pods -n <namespace> --sort-by=memory

# Step 2: Compare with limits
kubectl -n <namespace> get pod <pod-name> -o jsonpath='{.spec.containers[0].resources}'

# Step 3: Options:
# a) Increase limits in Helm values (if underprovisioned)
# b) Investigate the workload (memory leak? CPU-bound loop?)
# c) Scale horizontally (add replicas)
```

### DeploymentReplicasMismatch (Warning)

**Meaning:** The deployment wants N replicas but fewer are available.

```bash
# Step 1: Check deployment
kubectl -n <namespace> get deploy
# READY column shows available/desired

# Step 2: Check why pods aren't starting
kubectl -n <namespace> get events --sort-by=.lastTimestamp
# Look for: FailedScheduling, FailedMount, ImagePull errors

# Step 3: Common fixes:
# • FailedScheduling → node is full, need to drain or add node
# • PVC not bound → StorageClass issue
# • ImagePullBackOff → check registry secret
```

---

## 5. Database Alerts

### DatabaseStorageFull (Critical)

**Meaning:** PostgreSQL PVC is over 85% full. If it hits 100%, writes fail.

```bash
# Step 1: Check PVC usage
kubectl -n <namespace> exec <pg-primary-pod> -- df -h /var/lib/postgresql/data
# Shows actual disk usage

# Step 2: Check which databases are largest
kubectl -n <namespace> exec <pg-primary-pod> -- psql -U postgres -c \
  "SELECT datname, pg_size_pretty(pg_database_size(datname)) FROM pg_database ORDER BY pg_database_size(datname) DESC;"

# Step 3: Options:
# a) VACUUM FULL on large tables (reclaims space, but locks table)
# b) Increase PVC size (CNPG supports online resize):
#    Edit Cluster CRD → spec.storage.size: "100Gi" (was 50Gi)
# c) Delete old data (customer cleanup)

# Step 4: Monitor after fix
# Grafana → Zenith - Service Health → Disk usage panel
```

### DatabaseBackupFailed (Critical)

**Meaning:** No successful CNPG backup in the last 24 hours.

```bash
# Step 1: Check backup status
kubectl -n <namespace> get scheduledbackup
kubectl -n <namespace> get backup --sort-by=.status.startedAt

# Step 2: Check CNPG operator logs
kubectl -n cnpg-system logs deploy/cnpg-controller-manager --tail=100

# Step 3: Check S3 connectivity
kubectl -n <namespace> exec <pg-primary-pod> -- \
  env | grep S3  # Check if S3 credentials are set

# Step 4: Manual backup trigger
kubectl -n <namespace> create -f - <<EOF
apiVersion: postgresql.cnpg.io/v1
kind: Backup
metadata:
  name: manual-backup-$(date +%Y%m%d%H%M)
spec:
  cluster:
    name: <cluster-name>
EOF
```

### DatabaseHighConnectionCount (Warning)

**Meaning:** More than 80 active connections. Default max is 100-200.

```bash
# Step 1: See who's connected
kubectl -n <namespace> exec <pg-primary-pod> -- psql -U postgres -c \
  "SELECT datname, usename, client_addr, state, count(*)
   FROM pg_stat_activity
   GROUP BY datname, usename, client_addr, state
   ORDER BY count DESC;"

# Step 2: Kill idle connections (if safe)
kubectl -n <namespace> exec <pg-primary-pod> -- psql -U postgres -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle' AND query_start < now() - interval '30 minutes';"

# Step 3: Long-term fix:
# • Add connection pooling (PgBouncer)
# • Increase max_connections in CNPG Cluster spec
# • Fix application connection leaks
```

---

## 6. Node Alerts

### NodeHighMemory (Critical)

**Meaning:** Node memory is above 90%. Risk of OOM kills.

```bash
# Step 1: Check node resources
kubectl top nodes
# Shows CPU and memory usage per node

# Step 2: Find the heaviest pods
kubectl top pods -A --sort-by=memory | head -20

# Step 3: Check PriorityClasses (who gets evicted first)
# customer (10000) → platform (100000) → infra (500000) → core (1000000)
# Kubernetes evicts lowest priority first

# Step 4: Options:
# a) Scale down customer workloads (KEDA scale-to-zero)
# b) Evict low-priority pods: kubectl drain <node> --ignore-daemonsets
# c) Add a new node (Hetzner Console or Terraform)
# d) Increase node size (Terraform: server_type)
```

### NodeDiskPressure (Warning)

**Meaning:** Node disk is above 85%.

```bash
# Step 1: Check what's using disk
ssh zen-stage  # SSH to the node
df -h          # Show disk usage
du -sh /var/log/*          # Log files
du -sh /var/lib/containerd # Container images + layers
du -sh /var/lib/rancher    # k3s data

# Step 2: Cleanup
# Remove old container images
crictl rmi --prune

# Rotate logs
journalctl --vacuum-size=500M

# Step 3: Long-term fix:
# Increase disk size in Hetzner Console, then:
# growpart /dev/sda 1 && resize2fs /dev/sda1
```

---

## 7. Grafana Dashboards Guide

```
┌─────────────────────────────────────────────────────────────────────────┐
│             GRAFANA DASHBOARDS                                           │
│                                                                          │
│  URL: https://grafana.stage.freezenith.com                              │
│  Login: admin / <admin_password from Terraform>                         │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  DASHBOARD: Zenith - Platform Overview                             │ │
│  │  ──────────────────────────────────                                │ │
│  │  • Cluster CPU / Memory / Disk usage (gauge)                      │ │
│  │  • Total running pods (stat)                                      │ │
│  │  • Per-app CPU and memory usage (time series)                     │ │
│  │  • HTTP request rate (requests/sec)                               │ │
│  │  • HTTP latency P99 (milliseconds)                                │ │
│  │                                                                    │ │
│  │  USE WHEN: Daily health check, capacity planning                  │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  DASHBOARD: Zenith - Service Health                                │ │
│  │  ──────────────────────────────────                                │ │
│  │  • Service up/down status (API, Operator, Auth, APISIX)           │ │
│  │  • Error rate (5xx responses per service)                         │ │
│  │  • Response time P95 (per service)                                │ │
│  │  • Operator reconciliation rate and errors                        │ │
│  │                                                                    │ │
│  │  USE WHEN: Investigating service degradation or errors            │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  DASHBOARD: Zenith - Node Metrics                                  │ │
│  │  ──────────────────────────────────                                │ │
│  │  • Total nodes and ready count                                    │ │
│  │  • CPU cores and memory per node                                  │ │
│  │  • CPU / Memory / Disk usage per node (time series)               │ │
│  │  • Network traffic per node                                       │ │
│  │                                                                    │ │
│  │  USE WHEN: Node capacity issues, scaling decisions                │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  EXPLORE (Ad-hoc queries):                                               │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Datasource: Prometheus → PromQL queries for metrics              │ │
│  │  Datasource: Loki → LogQL queries for logs                       │ │
│  │  Datasource: Tempo → Trace ID lookup                             │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Log Investigation (Loki)

```bash
# In Grafana → Explore → Datasource: Loki

# All logs from a namespace
{namespace="zenith-staging"}

# Filter by app
{namespace="zenith-staging", app="zenith-api"}

# Search for errors
{namespace="zenith-staging"} |= "error"

# Search for specific customer
{namespace="zenith-staging"} |= "customer-abc"

# JSON parsing + filtering
{namespace="zenith-staging"} | json | level="error"
{namespace="zenith-staging"} | json | status >= 500
{namespace="zenith-staging"} | json | latency_ms > 1000

# Count errors per minute
count_over_time({namespace="zenith-staging"} |= "error" [1m])

# Top error messages
{namespace="zenith-staging"} |= "error" | pattern `<msg>`
  | line_format "{{.msg}}"
```

---

## 9. Trace Investigation (Tempo)

```
┌─────────────────────────────────────────────────────────────────────────┐
│             DISTRIBUTED TRACING                                          │
│                                                                          │
│  Trace flow:                                                             │
│  APISIX (OTel plugin) → OTel Collector (DaemonSet) → Tempo → Grafana  │
│  zenith-api (Go SDK)  ↗                                                │
│                                                                          │
│  To find a trace:                                                        │
│  1. Open Grafana → Explore → Datasource: Tempo                         │
│  2. Search by:                                                           │
│     • Service name: zenith-api                                          │
│     • HTTP method: GET, POST                                            │
│     • Duration: > 1s (slow requests)                                    │
│     • Status: error                                                      │
│  3. Click a trace to see the waterfall view                             │
│                                                                          │
│  OR: If you have a trace ID from logs:                                   │
│  1. Grafana → Explore → Tempo                                           │
│  2. Paste the trace ID → see full request path                         │
│                                                                          │
│  Trace shows:                                                            │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  APISIX gateway ─── 2ms ──┐                                       │ │
│  │                            ▼                                       │ │
│  │  zenith-api handler ────── 15ms ──┐                                │ │
│  │                                    ▼                                │ │
│  │  zenith-api service ────── 8ms ───┐                                │ │
│  │                                    ▼                                │ │
│  │  PostgreSQL query ──────── 3ms    (slowest span = bottleneck)      │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Network Investigation (Hubble)

```bash
# Hubble CLI (if installed)
hubble observe --namespace zenith-staging --last 100
hubble observe --namespace zenith-staging --verdict DROPPED
hubble observe --to-namespace zenith-staging --protocol TCP --port 5432

# Hubble UI (visual service map)
open https://hubble.stage.freezenith.com
# Shows: service-to-service connections, dropped packets, DNS queries

# Common network issues:
# 1. CiliumNetworkPolicy blocking traffic
kubectl -n <namespace> get cnp
kubectl -n <namespace> describe cnp <policy-name>

# 2. DNS resolution failing
hubble observe --namespace <namespace> --protocol DNS
kubectl -n <namespace> exec <pod> -- nslookup free-pg-rw.zenith-shared.svc.cluster.local

# 3. Cross-namespace traffic blocked
hubble observe --from-namespace <ns1> --to-namespace <ns2> --verdict DROPPED
```
