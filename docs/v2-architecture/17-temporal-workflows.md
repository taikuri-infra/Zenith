# 17 — Temporal Workflow Engine

> **Purpose:** Understand how Temporal orchestrates customer provisioning, upgrades, and deprovisioning as reliable, retryable workflows.
> **Audience:** Any developer who needs to add workflow activities, debug failed provisioning, or understand the automation layer.
> **Last Updated:** 2026-03-03
> **Related:** [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) (installation steps), [10-backend-architecture.md](./10-backend-architecture.md) (Go worker code in `services/provisioning/`), [16-data-storage.md](./16-data-storage.md) (Temporal uses CNPG for persistence)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose It](#2-why-we-chose-it)
3. [Architecture Diagram](#3-architecture-diagram)
4. [How Temporal Works (Concepts)](#4-how-temporal-works-concepts)
5. [Zenith Workflows](#5-zenith-workflows)
6. [Activity Execution Flow](#6-activity-execution-flow)
7. [Error Handling & Retries](#7-error-handling--retries)
8. [Configuration Reference](#8-configuration-reference)
9. [Troubleshooting](#9-troubleshooting)
10. [Upgrade Path](#10-upgrade-path)

---

## 1. Overview

**Temporal** is the workflow engine that automates all long-running operations in Zenith:

- **Customer provisioning** — Create Keycloak realm, database, S3 bucket, K8s namespace
- **Customer upgrades** — Migrate data between tiers, adjust limits
- **Customer deprovisioning** — Clean up all resources
- **Enterprise provisioning** — Create dedicated VM clusters via CAPI

```
Why not just use a simple background job?

  Simple job:     Step 1 → Step 2 → Step 3 (if Step 2 fails → everything lost)
  Temporal:       Step 1 → Step 2 → Step 3 (if Step 2 fails → retry Step 2 only)
                  Full history preserved. Can inspect/replay/fix. Never lose progress.
```

---

## 2. Why We Chose It

| Feature | Temporal | Argo Workflows | Celery | Custom goroutines |
|---------|----------|---------------|--------|------------------|
| Durable execution | Yes | Partial | No | No |
| Automatic retries | Built-in (per-activity) | Built-in | Manual | Manual |
| Workflow history | Full (searchable) | Full | None | None |
| Long-running (days) | Yes | Yes | Timeout issues | Complex |
| Language SDK | Go, Java, Python, TS | YAML only | Python only | Go only |
| Web UI | Yes (built-in) | Yes | Flower | None |
| Compensation (undo) | Saga pattern built-in | Manual | Manual | Manual |

**Decision:** Temporal gives us durable execution — if the server crashes mid-provisioning, it resumes exactly where it left off. This is critical for operations that create real infrastructure (databases, S3 buckets, DNS records).

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         TEMPORAL IN THE ZENITH CLUSTER                      │
│                         Namespace: temporal                                 │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    TEMPORAL SERVER                                     │  │
│  │                    PriorityClass: platform (100000)                    │  │
│  │                    Resources: 100m CPU, 256Mi RAM                     │  │
│  │                                                                       │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ Frontend Service (:7233)                                         │ │  │
│  │  │ - gRPC API for clients (zenith-api calls this)                   │ │  │
│  │  │ - Start workflows, query status, signal workflows                │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ History Service                                                  │ │  │
│  │  │ - Stores workflow execution history                              │ │  │
│  │  │ - Manages timers and activity scheduling                        │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ Matching Service                                                 │ │  │
│  │  │ - Matches activity tasks to available workers                    │ │  │
│  │  │ - Manages task queues                                            │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ Worker Service                                                   │ │  │
│  │  │ - Internal Temporal workers (system workflows)                   │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ Temporal Web UI (:8080)                                               │  │
│  │ Image: temporalio/ui:2.47.2                                           │  │
│  │ URL: temporal.stage.freezenith.com (via Traefik)                      │  │
│  │                                                                       │  │
│  │ Shows: workflow executions, activity details, history, search          │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  DATABASE (external — uses free-pg in zenith-shared):                      │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ Host: free-pg-rw.zenith-shared.svc.cluster.local:5432                 │  │
│  │ Databases: temporal (workflow data), temporal_visibility (search)      │  │
│  │ User: temporal (created by CNPG bootstrap postInitSQL)                │  │
│  │ Driver: postgres12                                                    │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  CONNECTIONS:                                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                                                                       │  │
│  │  zenith-api ──gRPC :7233──▶ Temporal Frontend                        │  │
│  │    (start workflow, query status)                                      │  │
│  │                                                                       │  │
│  │  zenith-api ◀──gRPC──────── Temporal (activity dispatch)             │  │
│  │    (zenith-api IS the worker — runs activities)                        │  │
│  │                                                                       │  │
│  │  Temporal ──TCP :5432──▶ free-pg-rw (PostgreSQL)                     │  │
│  │    (workflow history, visibility store)                                │  │
│  │                                                                       │  │
│  │  Traefik ──HTTP :8080──▶ Temporal Web UI                             │  │
│  │    (developer access to workflow dashboard)                            │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. How Temporal Works (Concepts)

```
┌─────────────────────────────────────────────────────────────────────────┐
│              TEMPORAL CONCEPTS (for someone who's never used it)          │
│                                                                          │
│  WORKFLOW = A function that orchestrates a series of steps               │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ func ProvisionCustomer(ctx, customerID, plan) {                    │ │
│  │   realm := activity.CreateKeycloakRealm(ctx, customerID)          │ │
│  │   db := activity.CreateDatabase(ctx, customerID, plan)            │ │
│  │   s3 := activity.CreateS3Bucket(ctx, customerID)                  │ │
│  │   activity.CreateK8sNamespace(ctx, customerID, plan)              │ │
│  │   activity.CreateK8sSecrets(ctx, realm, db, s3)                   │ │
│  │   activity.NotifyReady(ctx, customerID)                           │ │
│  │ }                                                                  │ │
│  │                                                                    │ │
│  │ - Looks like normal Go code                                       │ │
│  │ - But if server crashes after CreateDatabase:                     │ │
│  │   → Temporal replays from CreateDatabase (skips completed steps)  │ │
│  │   → No duplicate databases created                                │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ACTIVITY = A single side-effecting operation                           │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ func CreateDatabase(ctx, customerID, plan) (*DBInfo, error) {     │ │
│  │   db, _ := sql.Open("postgres", freePgDSN)                       │ │
│  │   db.Exec("CREATE DATABASE customer_" + customerID)               │ │
│  │   db.Exec("CREATE USER customer_" + customerID + " ...")          │ │
│  │   return &DBInfo{Host: "free-pg-rw", Name: "customer_" + id}, nil│ │
│  │ }                                                                  │ │
│  │                                                                    │ │
│  │ - Does real work (SQL, API calls, K8s operations)                 │ │
│  │ - Has retry policy (retry up to 5 times on failure)               │ │
│  │ - Has timeout (e.g., 60 seconds)                                   │ │
│  │ - Result is stored in Temporal's history DB                       │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  WORKER = Process that executes activities (zenith-api IS the worker)   │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ zenith-api on startup:                                             │ │
│  │   1. Connect to Temporal frontend (gRPC :7233)                    │ │
│  │   2. Register workflows: ProvisionCustomer, UpgradeCustomer, etc. │ │
│  │   3. Register activities: CreateDatabase, CreateS3Bucket, etc.    │ │
│  │   4. Poll for tasks on "zenith-provisioning" task queue           │ │
│  │   5. Execute activities when Temporal dispatches them              │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  TASK QUEUE = Named channel connecting workflows to workers             │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ "zenith-provisioning" task queue:                                  │ │
│  │                                                                    │ │
│  │  Temporal Frontend    →→→ [task1, task2, task3] →→→    zenith-api │ │
│  │  (dispatches tasks)       (queue in DB)              (polls & runs)│ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Zenith Workflows

### ProvisionCustomer (Free/Pro)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  WORKFLOW: ProvisionCustomer                                             │
│  Trigger: POST /v1/auth/register (user signs up)                        │
│  Duration: ~30-60 seconds                                                │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Input: { customerID, email, plan: "free"|"pro" }                  │ │
│  │                                                                    │ │
│  │  Step 1: CreateKeycloakRealm                                       │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │ POST keycloak:8080/admin/realms                              │  │ │
│  │  │ Create realm: customer-{customerID}                          │  │ │
│  │  │ Create OIDC client + default roles                           │  │ │
│  │  │ Output: { client_id, client_secret, realm_name }             │  │ │
│  │  │ Retry: 3 attempts, 5s backoff                                │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  │       │                                                            │ │
│  │       ▼                                                            │ │
│  │  Step 2: CreateDatabase                                            │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │ SQL to free-pg-rw:5432:                                      │  │ │
│  │  │   CREATE DATABASE customer_{id}                               │  │ │
│  │  │   CREATE USER customer_{id} WITH PASSWORD '...'               │  │ │
│  │  │   GRANT ALL ON DATABASE customer_{id} TO customer_{id}       │  │ │
│  │  │ Output: { db_host, db_name, db_user, db_password }           │  │ │
│  │  │ Retry: 3 attempts, 5s backoff                                │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  │       │                                                            │ │
│  │       ▼                                                            │ │
│  │  Step 3: CreateS3Bucket                                            │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │ Hetzner S3 API: create bucket customer-{id}-data             │  │ │
│  │  │ Create access key for customer                                │  │ │
│  │  │ Output: { bucket, access_key, secret_key, endpoint }         │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  │       │                                                            │ │
│  │       ▼                                                            │ │
│  │  Step 4: CreateK8sResources                                        │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │ K8s API:                                                      │  │ │
│  │  │   Create namespace zenith-customer-{id}                       │  │ │
│  │  │   Apply ResourceQuota (tier limits)                           │  │ │
│  │  │   Apply LimitRange                                            │  │ │
│  │  │   Apply CiliumNetworkPolicy (isolation)                       │  │ │
│  │  │   Create Secrets (DB, S3, Keycloak creds)                     │  │ │
│  │  │   Create Deployment + Service                                 │  │ │
│  │  │   Create IngressRoute + APISIX routes                        │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  │       │                                                            │ │
│  │       ▼                                                            │ │
│  │  Step 5: WaitForReady                                              │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │ Poll: DNS resolution (external-dns creates Cloudflare record)│  │ │
│  │  │ Poll: TLS cert (cert-manager issues Let's Encrypt cert)      │  │ │
│  │  │ Poll: Pod readiness (Deployment has Ready pods)              │  │ │
│  │  │ Timeout: 5 minutes                                           │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  │       │                                                            │ │
│  │       ▼                                                            │ │
│  │  Step 6: NotifyCustomerReady                                       │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │ Update customer status in DB: "provisioning" → "ready"       │  │ │
│  │  │ Send welcome email                                            │  │ │
│  │  │ Dashboard shows: "Your environment is ready!"                │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Activity Execution Flow

```
zenith-api (worker)           Temporal Server              free-pg (database)
    │                              │                            │
    │  Register worker +           │                            │
    │  poll task queue             │                            │
    ├─────────────────────────────▶│                            │
    │                              │                            │
    │   (API receives signup)      │                            │
    │  Start workflow               │                            │
    ├─────────────────────────────▶│                            │
    │                              │ Store workflow in DB       │
    │                              │                            │
    │  Poll: got activity task     │                            │
    │◀─────────────────────────────┤                            │
    │                              │                            │
    │  Execute: CreateDatabase     │                            │
    ├──────────────────────────────┼───────────────────────────▶│
    │                              │         CREATE DATABASE     │
    │                              │                            │
    │  Activity result: success    │                            │
    ├─────────────────────────────▶│                            │
    │                              │ Store result in history    │
    │                              │                            │
    │  Poll: got next activity     │                            │
    │◀─────────────────────────────┤                            │
    │                              │                            │
    │  Execute: CreateS3Bucket     │                            │
    │  ... (continues)             │                            │
    ▼                              ▼                            ▼
```

---

## 7. Error Handling & Retries

```
┌─────────────────────────────────────────────────────────────────────────┐
│              ERROR HANDLING IN TEMPORAL                                   │
│                                                                          │
│  SCENARIO: CreateDatabase fails (PostgreSQL is temporarily down)         │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ Attempt 1: CreateDatabase()                                        │ │
│  │   → Error: "connection refused" to free-pg-rw:5432                │ │
│  │   → Temporal: retry after 5 seconds                               │ │
│  │                                                                    │ │
│  │ Attempt 2: CreateDatabase() (5s later)                             │ │
│  │   → Error: "connection refused" (still down)                      │ │
│  │   → Temporal: retry after 10 seconds (exponential backoff)        │ │
│  │                                                                    │ │
│  │ Attempt 3: CreateDatabase() (10s later)                            │ │
│  │   → Success! Database created.                                    │ │
│  │   → Temporal: store result, move to next activity                 │ │
│  │                                                                    │ │
│  │ KEY INSIGHT: Activities 1 (CreateKeycloakRealm) is NOT re-run.   │ │
│  │ Temporal knows it already succeeded — it's in the history.         │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  SCENARIO: All retries exhausted                                         │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ After max retries:                                                 │ │
│  │   → Workflow status: "Failed"                                      │ │
│  │   → Customer status in DB: "provisioning_failed"                  │ │
│  │   → Visible in Temporal Web UI (temporal.stage.freezenith.com)    │ │
│  │   → Admin can:                                                     │ │
│  │     1. Fix the issue (e.g., restart PostgreSQL)                   │ │
│  │     2. Retry the workflow from the Temporal UI                    │ │
│  │     3. Or: cancel and clean up (saga compensation)                │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  SAGA COMPENSATION (cleanup on permanent failure):                       │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ If CreateK8sResources fails permanently:                           │ │
│  │                                                                    │ │
│  │   Undo Step 3: Delete S3 bucket (compensation activity)           │ │
│  │   Undo Step 2: Drop database (compensation activity)              │ │
│  │   Undo Step 1: Delete Keycloak realm (compensation activity)      │ │
│  │                                                                    │ │
│  │ Temporal ensures compensations run even if server restarts         │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Configuration Reference

### Terraform File

**File:** `infra/terraform/modules/k8s-platform/temporal.tf`

| Setting | Value | Purpose |
|---------|-------|---------|
| `namespace` | temporal | Deployment namespace |
| `persistence.default.sql.driver` | postgres12 | PostgreSQL driver |
| `persistence.default.sql.host` | free-pg-rw.zenith-shared.svc.cluster.local | Primary PG |
| `persistence.default.sql.port` | 5432 | PG port |
| `persistence.default.sql.database` | temporal | Workflow data |
| `persistence.visibility.sql.database` | temporal_visibility | Search/query data |
| `cassandra.enabled` | false | Not using Cassandra |
| `mysql.enabled` | false | Not using MySQL |
| `postgresql.enabled` | false | Using external CNPG |
| `elasticsearch.enabled` | false | Using PG visibility |
| `web.enabled` | true | Web UI enabled |
| `web.image.tag` | 2.47.2 | Pinned (2.30.5 deleted from Docker Hub) |
| `schema.createDatabase.enabled` | true | Auto-create DB schemas |
| `server.resources.requests.cpu` | 100m | Server CPU request |
| `server.resources.requests.memory` | 256Mi | Server memory request |
| `server.priorityClassName` | platform | Eviction priority |

### Database Setup

Temporal's databases are created by CNPG's `postInitSQL` in `storage.tf`:
```sql
CREATE ROLE temporal WITH LOGIN PASSWORD '...' CREATEDB;
CREATE DATABASE temporal OWNER temporal;
CREATE DATABASE temporal_visibility OWNER temporal;
```

---

## 9. Troubleshooting

### Workflow stuck in "Running"

```bash
# 1. Check Temporal Web UI
# Open: temporal.stage.freezenith.com
# Find the workflow → look at pending activities

# 2. Check if zenith-api worker is connected
kubectl logs -n zenith-staging deploy/zenith-api --tail=50 | grep temporal

# 3. Check Temporal server logs
kubectl logs -n temporal deploy/temporal-frontend --tail=50
kubectl logs -n temporal deploy/temporal-history --tail=50

# 4. Check if activity is timing out
# In Web UI: click the pending activity → see retry count and last error
```

### "Workflow execution already started"

```bash
# Duplicate workflow ID — previous execution still running
# In Temporal UI: search for the workflow ID
# Options:
#   1. Wait for it to complete
#   2. Terminate it: tctl workflow terminate -w <workflow-id>
#   3. Cancel it: tctl workflow cancel -w <workflow-id>
```

### Temporal pods crashing

```bash
# Usually database connectivity
# 1. Check if free-pg is healthy
kubectl get cluster free-pg -n zenith-shared

# 2. Check Temporal can reach the database
kubectl exec -n temporal deploy/temporal-frontend -- \
  pg_isready -h free-pg-rw.zenith-shared.svc -p 5432

# 3. Check schema migration status
kubectl logs -n temporal job/temporal-schema-default -setup --tail=50
```

### Worker not picking up tasks

```bash
# zenith-api must be running and connected to Temporal
# 1. Check zenith-api is running
kubectl get pods -n zenith-staging -l app=zenith-api

# 2. Check worker registration in logs
kubectl logs -n zenith-staging deploy/zenith-api | grep "temporal\|worker\|registered"

# 3. Verify Temporal frontend is reachable from zenith-api
kubectl exec -n zenith-staging deploy/zenith-api -- \
  curl -v telnet://temporal-frontend.temporal.svc:7233
```

---

## 10. Upgrade Path

### Upgrading Temporal

```bash
# Update version in variables.tf
# IMPORTANT: Check Temporal release notes for schema migrations
terraform plan -target=helm_release.temporal
terraform apply -target=helm_release.temporal

# Schema migrations run automatically via Jobs
# Monitor: kubectl get jobs -n temporal
```

### Adding a new workflow

1. Define workflow function in `services/api/internal/temporal/workflows/`
2. Define activity functions in `services/api/internal/temporal/activities/`
3. Register both in the worker startup code
4. Start workflow from API handler: `temporalClient.ExecuteWorkflow(...)`
5. Test via Temporal UI: manually start workflow with test input
