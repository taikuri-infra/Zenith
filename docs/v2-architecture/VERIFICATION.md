# Zenith V2 — Verification & User Flow Scenarios

> **Purpose:** A concrete, testable verification plan for every major user flow in the Zenith platform.
> Each scenario traces the exact API calls, expected responses, and infrastructure side-effects.
> Use this document to walk through the platform and verify correctness.

> **Last Updated:** 2026-03-01
> **Status:** Active verification document

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [How Sections Work Together](#2-how-sections-work-together)
3. [User Flow Scenarios](#3-user-flow-scenarios)
   - [Scenario 1: Developer Signup & First App Deploy](#scenario-1-developer-signup--first-app-deploy)
   - [Scenario 2: Git Push → Auto-Deploy (Webhook)](#scenario-2-git-push--auto-deploy-webhook)
   - [Scenario 3: Tenant Purchase → Full Environment (Temporal)](#scenario-3-tenant-purchase--full-environment-temporal)
   - [Scenario 4: Tenant Deprovision → Clean Teardown](#scenario-4-tenant-deprovision--clean-teardown)
   - [Scenario 5: Admin Dashboard & Customer Management](#scenario-5-admin-dashboard--customer-management)
   - [Scenario 6: App Rollback to Previous Version](#scenario-6-app-rollback-to-previous-version)
   - [Scenario 7: Environment Variables & Secrets](#scenario-7-environment-variables--secrets)
   - [Scenario 8: APISIX JWT-Protected Route (Tenant API)](#scenario-8-apisix-jwt-protected-route-tenant-api)
4. [Component Interaction Map](#4-component-interaction-map)
5. [Known Bug Registry](#5-known-bug-registry)
6. [Staging Test Commands](#6-staging-test-commands)

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Internet                                  │
│    *.stage.freezenith.com → Cloudflare Proxy → Hetzner VM       │
└──────────────────────────────┬──────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│  k3s Cluster (Hetzner cx43)                                      │
│                                                                   │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────────┐    │
│  │  Traefik     │  │  APISIX +    │  │  cert-manager        │    │
│  │  (Ingress)   │  │  etcd        │  │  (DNS-01/Cloudflare) │    │
│  └──────┬───────┘  └──────┬───────┘  └──────────────────────┘    │
│         │                  │                                       │
│  ┌──────▼───────┐  ┌──────▼───────┐  ┌──────────────────────┐    │
│  │ zenith-api   │  │ Keycloak     │  │  ArgoCD              │    │
│  │ (Go/Fiber)   │  │ (OIDC/Auth)  │  │  (GitOps sync)       │    │
│  └──────┬───────┘  └──────────────┘  └──────────────────────┘    │
│         │                                                          │
│  ┌──────▼───────────────────────────────────────────────────┐    │
│  │ Deploy Engine (Pipeline → Builder → Kaniko → Deployer)   │    │
│  └──────┬───────────────────────────────────────────────────┘    │
│         │                                                          │
│  ┌──────▼───────┐  ┌──────────────┐  ┌──────────────────────┐    │
│  │ zenith-apps  │  │  Temporal    │  │  CNPG (PostgreSQL)   │    │
│  │ (user pods)  │  │ (workflows)  │  │  free-pg / kc-pg     │    │
│  └──────────────┘  └──────────────┘  └──────────────────────┘    │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐    │
│  │  Harbor      │  │  Velero      │  │  Prometheus/Grafana  │    │
│  │ (registry)   │  │ (backups)    │  │  Loki/Tempo/OTel     │    │
│  └──────────────┘  └──────────────┘  └──────────────────────┘    │
└───────────────────────────────────────────────────────────────────┘
```

---

## 2. How Sections Work Together

### Request Flow (User → App)

```
User Browser
  → Cloudflare (WAF/DDoS, DNS → Hetzner IP)
  → Traefik (TLS termination, IngressRoute matching)
  → zenith-api (Go/Fiber, JWT auth, business logic)
  → PostgreSQL (CNPG, data persistence)
```

### Deploy Flow (Git → Running App)

```
Developer pushes code
  → POST /api/v1/apps/:id/deploy (or webhook)
  → Pipeline.TriggerBuild() (goroutine)
    → Builder.CloneRepo() (shallow git clone locally)
    → Builder.DetectFramework() (file markers)
    → Builder.GenerateDockerfile() (template per framework)
    → KanikoRunner.Build()
      → ConfigMap with Dockerfile (if generated)
      → K8s Job: init-container (git clone) + Kaniko (docker build+push)
      → Polls Job status every 5s (30min timeout)
      → Streams logs to LogHub
    → Deployer.DeployApp()
      → Creates/patches: Deployment, Service, IngressRoute
      → Optionally: HTTPScaledObject (free-tier scale-to-zero)
  → App accessible at https://{subdomain}.apps.stage.freezenith.com
```

### Provisioning Flow (Purchase → Full Tenant Environment)

```
Admin creates customer via API
  → CustomerService.ProvisionCustomer()
  → Temporal: ProvisionCustomerWorkflow (11 steps)
    1. UpdateStatusProvisioning
    2. CreateKeycloakRealm (realm + OIDC client)
    3. CreateDatabase (CNPG: CREATE USER + CREATE DATABASE)
    4. CreateS3Bucket (Hetzner Object Storage)
    5. CreateNamespace (zenith-{domain})
    6. CreateNetworkPolicies (4 policies: deny-all, allow-dns, allow-intra, allow-traefik)
    7. CreateSecrets (db-credentials, keycloak-credentials, s3-credentials)
    8. CreateResourceQuota + LimitRange
    9. CreateRouting (Traefik IngressRoute)
    10. CreateTLS (cert-manager Certificate)
    11. CreateArgoCD (ArgoCD Application → tenant Helm chart)
    12. NotifyReady (status=active, audit log)
  → Tenant accessible at https://{domain}.stage.freezenith.com
```

### Auth Flow

```
User → POST /auth/login (email + password)
  → bcrypt password verification
  → JWT access_token (1h) + refresh_token (7d)
  → All protected routes: Authorization: Bearer <token>
  → RequireAuth middleware validates JWT, sets user_id/email/role in Fiber locals
  → RequireRole middleware checks role hierarchy (owner > admin > developer > viewer)
```

---

## 3. User Flow Scenarios

### Scenario 1: Developer Signup & First App Deploy

**Actors:** New developer
**Goal:** Register, create an app from a Git repo, deploy it, access it via URL

#### Step 1: Register

```bash
curl -X POST https://api.stage.freezenith.com/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"dev@example.com","password":"SecurePass123!","name":"Dev User"}'
```

**Expected Response:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": { "id": "uuid", "email": "dev@example.com", "name": "Dev User", "role": "owner" }
}
```

**Side-effects:**
- First user gets `owner` role; subsequent users get `developer`
- Password stored as bcrypt hash (cost=10)
- User stored in PostgreSQL `users` table

#### Step 2: Create App

```bash
TOKEN="eyJ..."
curl -X POST https://api.stage.freezenith.com/api/v1/apps \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"hello-world","repo_url":"https://github.com/crccheck/docker-hello-world","branch":"master"}'
```

**Expected Response:**
```json
{
  "id": "uuid",
  "name": "hello-world",
  "repo_url": "https://github.com/crccheck/docker-hello-world",
  "branch": "master",
  "framework": "",
  "status": "pending",
  "subdomain": "hello-world",
  "url": "https://hello-world.apps.stage.freezenith.com",
  "port": 8080,
  "created_at": "...",
  "updated_at": "..."
}
```

**Side-effects:**
- App record created in PostgreSQL
- Subdomain auto-generated from name (sanitized, deduplicated)
- Status = `pending` (not yet deployed)

#### Step 3: Deploy App

```bash
APP_ID="uuid-from-step-2"
curl -X POST "https://api.stage.freezenith.com/api/v1/apps/$APP_ID/deploy" \
  -H "Authorization: Bearer $TOKEN"
```

**Expected Response:**
```json
{
  "deployment_id": "uuid",
  "status": "building",
  "message": "build started"
}
```

**Side-effects (async pipeline):**
1. Deployment record created (status=`building`)
2. Pipeline goroutine starts:
   - Clones repo locally → detects framework → generates Dockerfile
   - Creates ConfigMap `build-dockerfile-{name}` in `zenith-builds` (if Dockerfile was generated)
   - Creates K8s Job in `zenith-builds`: init-container clones repo, Kaniko builds image
   - Image pushed to `registry.stage.freezenith.com/zenith-stage/hello-world:{deployId[:8]}`
   - Deployer creates: Deployment + Service + IngressRoute in `zenith-apps`
3. App status transitions: `pending` → `building` → `deploying` → `running`

#### Step 4: Stream Build Logs (SSE)

```bash
DEPLOY_ID="uuid-from-step-3"
curl -N "https://api.stage.freezenith.com/api/v1/apps/$APP_ID/deployments/$DEPLOY_ID/logs" \
  -H "Authorization: Bearer $TOKEN"
```

**Expected:** SSE stream with build logs, ending with `event: done`

#### Step 5: Verify App is Accessible

```bash
curl https://hello-world.apps.stage.freezenith.com
```

**Expected:** App responds with HTTP 200

**K8s Verification:**
```bash
KUBECONFIG=~/.kube/zenith-staging.yaml
kubectl get deploy,svc,ingressroute -n zenith-apps -l app=hello-world
```

---

### Scenario 2: Git Push → Auto-Deploy (Webhook)

**Actors:** Developer with connected GitHub repo
**Goal:** Push code to GitHub, app auto-deploys

#### Step 1: Configure GitHub Webhook

In GitHub repo settings → Webhooks:
- URL: `https://api.stage.freezenith.com/api/v1/webhooks/github`
- Content type: `application/json`
- Secret: (value of `GITHUB_WEBHOOK_SECRET` env var)
- Events: Push

#### Step 2: Push Code

```bash
git push origin main
```

#### Step 3: GitHub sends POST

```
POST /api/v1/webhooks/github
Headers: X-GitHub-Event: push, X-Hub-Signature-256: sha256=...
Body: { "ref": "refs/heads/main", "after": "abc123...", "repository": { "clone_url": "..." } }
```

**Expected Flow:**
1. Webhook handler verifies HMAC signature
2. Extracts branch from `refs/heads/main` → `main`
3. Scans all apps matching `clone_url` + `branch`
4. For each match: creates Deployment, triggers Pipeline.TriggerBuild()
5. Build + deploy same as Scenario 1 Steps 3-5

**Expected Response:**
```json
{
  "message": "deployments triggered",
  "triggered": ["hello-world"],
  "commit": "abc123...",
  "branch": "main"
}
```

---

### Scenario 3: Tenant Purchase → Full Environment (Temporal)

**Actors:** Platform admin
**Goal:** Provision a new free-tier customer with full isolated environment

#### Step 1: Create Customer

```bash
ADMIN_TOKEN="eyJ..."
curl -X POST https://api.stage.freezenith.com/api/v1/admin/customers \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Acme Corp",
    "domain": "acme.test",
    "plan_tier": "free",
    "contact_email": "admin@acme.test"
  }'
```

**Expected Response:**
```json
{
  "id": "uuid",
  "name": "Acme Corp",
  "domain": "acme.test",
  "status": "provisioning"
}
```

#### Step 2: Temporal Workflow Executes (Async)

**Monitor in Temporal UI:** `https://temporal.stage.freezenith.com`

Each step creates real infrastructure:

| Step | Resource | Verification Command |
|------|----------|---------------------|
| 1 | Keycloak realm `acme-test` | `kubectl exec -n keycloak sts/keycloak -- curl -s localhost:8080/realms/acme-test` |
| 2 | Database `z_acme_test` | `kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "\l" \| grep z_acme` |
| 3 | S3 bucket `zenith-acme-test` | Check via S3 API or Hetzner console |
| 4 | Namespace `zenith-acme-test` | `kubectl get ns zenith-acme-test` |
| 5 | 4 NetworkPolicies | `kubectl get networkpolicy -n zenith-acme-test` |
| 6 | 3 Secrets | `kubectl get secrets -n zenith-acme-test` |
| 7 | ResourceQuota + LimitRange | `kubectl get quota,limitrange -n zenith-acme-test` |
| 8 | IngressRoute | `kubectl get ingressroute -n zenith-acme-test` |
| 9 | Certificate | `kubectl get certificate -n zenith-acme-test` |
| 10 | ArgoCD Application | `kubectl get app -n argocd tenant-acme-test` |
| 11 | Status = `active` | `curl .../api/v1/admin/customers/{id}` → status=running |

#### Step 3: Verify Tenant Accessible

```bash
curl https://acme-test.stage.freezenith.com
# Should return the tenant app (once ArgoCD deploys the tenant Helm chart)
```

#### Step 4: Verify Network Isolation

```bash
# From acme-test namespace, should NOT be able to reach other tenant namespaces
kubectl exec -n zenith-acme-test <pod> -- curl -s --connect-timeout 2 <other-tenant-svc>
# Expected: Connection timeout (Cilium NetworkPolicy blocks cross-namespace traffic)
```

---

### Scenario 4: Tenant Deprovision → Clean Teardown

**Actors:** Platform admin
**Goal:** Delete a customer and all their resources

#### Step 1: Delete Customer

```bash
curl -X DELETE "https://api.stage.freezenith.com/api/v1/admin/customers/$CUSTOMER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

#### Step 2: Temporal DeprovisionCustomerWorkflow Executes

Deletes in order: ArgoCD app → Namespace → S3 bucket → Database → Keycloak realm

**Verification:**
```bash
kubectl get ns zenith-acme-test         # Should be gone
kubectl get app -n argocd tenant-acme-test  # Should be gone
# S3 bucket, DB, and Keycloak realm should all be deleted
```

---

### Scenario 5: Admin Dashboard & Customer Management

**Actors:** Platform admin
**Goal:** View platform stats, manage customers

```bash
# Dashboard stats
curl https://api.stage.freezenith.com/api/v1/admin/dashboard/stats \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# List all customers
curl https://api.stage.freezenith.com/api/v1/admin/customers \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Suspend a customer
curl -X POST "https://api.stage.freezenith.com/api/v1/admin/customers/$ID/suspend" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Audit log
curl https://api.stage.freezenith.com/api/v1/admin/audit \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

### Scenario 6: App Rollback to Previous Version

**Actors:** Developer
**Goal:** Roll back an app to a previously working deployment

```bash
# List deployments
curl "https://api.stage.freezenith.com/api/v1/apps/$APP_ID/deployments" \
  -H "Authorization: Bearer $TOKEN"

# Rollback to a specific deployment
curl -X POST "https://api.stage.freezenith.com/api/v1/apps/$APP_ID/rollback" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"deployment_id": "previous-deploy-uuid"}'
```

**Expected:** Old deployment's image is re-deployed to K8s (Deployment updated with old image tag)

---

### Scenario 7: Environment Variables & Secrets

**Actors:** Developer
**Goal:** Set env vars and encrypted secrets for an app

```bash
# Set env vars
curl -X PUT "https://api.stage.freezenith.com/api/v1/apps/$APP_ID/env" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"vars": {"DATABASE_URL": "postgres://...", "NODE_ENV": "production"}}'

# Get env vars
curl "https://api.stage.freezenith.com/api/v1/apps/$APP_ID/env" \
  -H "Authorization: Bearer $TOKEN"

# Set encrypted secret
curl -X POST "https://api.stage.freezenith.com/api/v1/apps/$APP_ID/secrets" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"key": "STRIPE_KEY", "value": "sk_live_..."}'

# Secrets are AES-256-GCM encrypted at rest
# Next deploy will inject these as container env vars
```

---

### Scenario 8: APISIX JWT-Protected Route (Tenant API)

**Actors:** Tenant app making API calls through APISIX
**Goal:** APISIX validates JWT from Keycloak before forwarding to backend

```bash
# Get JWT from Keycloak
KC_TOKEN=$(curl -s -X POST "https://auth.stage.freezenith.com/realms/{realm}/protocol/openid-connect/token" \
  -d "grant_type=password" \
  -d "client_id=zenith-app" \
  -d "client_secret={secret}" \
  -d "username=testuser" \
  -d "password=testpass" | jq -r '.access_token')

# Call APISIX-protected route (should pass JWT validation)
curl -H "Authorization: Bearer $KC_TOKEN" \
  https://api.stage.freezenith.com/protected-endpoint

# Without JWT → 401 Unauthorized (APISIX rejects)
curl https://api.stage.freezenith.com/protected-endpoint
```

---

## 4. Component Interaction Map

```
Component          │ Depends On                    │ Provides
───────────────────┼───────────────────────────────┼──────────────────────
Traefik            │ cert-manager (TLS)            │ Ingress routing, TLS termination
zenith-api         │ PostgreSQL, Keycloak (opt)    │ REST API, JWT auth, deploy engine
Pipeline/Builder   │ zenith-api, K8s API           │ Git clone, framework detection
Kaniko             │ Harbor (push), Git (clone)    │ Container image builds
Deployer           │ K8s API                       │ Deployments, Services, IngressRoutes
Keycloak           │ CNPG (keycloak-pg)            │ OIDC, realms, user management
Temporal           │ CNPG (free-pg), K8s API       │ Workflow orchestration
ArgoCD             │ Git repo, K8s API             │ GitOps sync, app deployment
Harbor             │ Hetzner S3 (backend)          │ Container + Helm registry
CNPG               │ hcloud-csi (volumes), S3 (WAL)│ PostgreSQL clusters
cert-manager       │ Cloudflare API                │ TLS certificates
external-dns       │ Cloudflare API                │ DNS record management
Cilium             │ Kernel (eBPF)                 │ NetworkPolicy, WireGuard encryption
APISIX             │ etcd                          │ API gateway, JWT validation
```

---

## 5. Known Bug Registry

Discovered via deep code analysis on 2026-03-01. Categorized by severity.

### Critical (Crashes / Completely Broken Features)

| # | Bug | File | Line | Impact |
|---|-----|------|------|--------|
| 1 | Labels type assertion fails — JSON maps are `map[string]interface{}`, not `map[string]string` | `deploy/deployer.go` | 129 | All deployed K8s resources missing labels |
| 2 | LogHub.Cleanup double-close panic — `sub.Ch` can be closed by both `Publish()` and `Cleanup()` | `deploy/log_hub.go` | 154 | Server crash on deployment cleanup |
| 3 | CreateDatabase password — panics if CustomerID < 8 chars or Domain < 4 chars; password is deterministic/guessable | `temporal/activities.go` | 104 | Crash + security: anyone can guess DB password |
| 4 | Rollback is DB-only — doesn't call `deployer.DeployApp()`, pods keep running old image | `handlers/deploy.go` | 83-97 | Rollback appears to succeed but doesn't work |
| 5 | Delete app is DB-only — doesn't call `deployer.DeleteApp()`, K8s resources leak | `handlers/apps_v2.go` | 142-153 | Deleted apps keep running, consuming resources |

### High (Broken Functionality / Security)

| # | Bug | File | Line | Impact |
|---|-----|------|------|--------|
| 6 | LogHandler reads `c.Params("id")` but route param is `:appId` | `handlers/logs.go` | 38,93 | Log streaming always returns 404 |
| 7 | BillingHandler reads `user_email` but middleware sets `email` | `handlers/billing.go` | 36 | Stripe checkout created without customer email |
| 8 | GitHub webhook `gitSHA[:8]` panics on empty SHA | `handlers/webhook.go` | 79,108 | Server crash on malformed webhook |
| 9 | `findAppsByRepo("")` — Postgres returns nothing (empty userID filter) | `handlers/webhook.go` | 133 | Webhooks silently fail with Postgres adapter |
| 10 | Deprovision workflow calls `UpdateStatusProvisioning` + `NotifyReady` | `temporal/workflow.go` | 128,142 | Deleted customer marked as "active" |
| 11 | SCIM endpoints have zero authentication | `cmd/server/main.go` | 561-568 | Anyone can create/list users via SCIM |
| 12 | ~15 handlers missing ownership checks | Various | — | Users can read/modify other users' secrets, env vars, deployments |

### Medium (Incorrect Behavior)

| # | Bug | File | Line | Impact |
|---|-----|------|------|--------|
| 13 | planRepo always uses memory implementation | `cmd/server/main.go` | — | Plan data lost on restart |
| 14 | Resource limits hardcoded (500m/512Mi), ignores plan | `deploy/k8s_resources.go` | — | Free/Pro users get same resources |
| 15 | Kaniko logs streamed after job completion | `deploy/kaniko_runner.go` | — | Build logs may be lost |
| 16 | `hasNextDep` substring match false positive | `deploy/detect.go` | — | `next-auth` detected as Next.js |
| 17 | APIKeyAuth defined but never wired | `middleware/auth.go` | — | API keys can be created but never used |

---

## 6. Staging Test Commands

```bash
# Environment setup
export KUBECONFIG=~/.kube/zenith-staging.yaml
export API="https://api.stage.freezenith.com/api/v1"

# Login
TOKEN=$(curl -s -X POST "$API/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@freezenith.com","password":"ZenithStaging2026!"}' | jq -r '.access_token')

# Health check
curl -s "$API/../health" | jq .

# List apps
curl -s "$API/apps" -H "Authorization: Bearer $TOKEN" | jq .

# Check K8s state
kubectl get all -n zenith-apps
kubectl get all -n zenith-staging
kubectl top nodes

# Temporal workflows
kubectl exec -n temporal deploy/temporal-frontend -- tctl workflow list

# ArgoCD app status
kubectl get applications -n argocd

# Check all certificates
kubectl get certificates --all-namespaces

# Check network policies
kubectl get networkpolicy --all-namespaces
```
