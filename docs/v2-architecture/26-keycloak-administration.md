# 26 — Keycloak Identity Administration

> **Purpose:** Understand how Keycloak manages identity for the Zenith platform — realms, users, clients, and how the API automates tenant provisioning.
> **Audience:** Any developer who needs to debug authentication issues, manage customer realms, or understand the identity layer.
> **Last Updated:** 2026-03-03
> **Related:** [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) (installation steps), [10-backend-architecture.md](./10-backend-architecture.md) (Keycloak integration in Go), [13-apisix-gateway.md](./13-apisix-gateway.md) (JWT verification), [17-temporal-workflows.md](./17-temporal-workflows.md) (realm provisioning via Temporal)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture Diagram](#2-architecture-diagram)
3. [Realm-Per-Tenant Model](#3-realm-per-tenant-model)
4. [How Realms Are Created](#4-how-realms-are-created)
5. [Admin Console Guide](#5-admin-console-guide)
6. [JWT Token Flow](#6-jwt-token-flow)
7. [API Integration (Go Code)](#7-api-integration-go-code)
8. [Configuration Reference](#8-configuration-reference)
9. [Troubleshooting](#9-troubleshooting)
10. [Upgrade Path](#10-upgrade-path)

---

## 1. Overview

**Keycloak** is Zenith's identity provider. It handles authentication (login/register), authorization (roles), and token issuance (JWT) for all platform customers.

**Key concept:** Each customer gets their own **realm** — a completely isolated identity space with its own users, clients, roles, and login page.

```
Think of it like this:

  master realm          → Zenith platform operators (admin@freezenith.com)
  customer-acme realm   → Acme Corp's users (alice@acme.com, bob@acme.com)
  customer-xyz realm    → XYZ Inc's users (admin@xyz.io)
  ...

  Each realm is completely isolated:
  - Acme's users can't see XYZ's users
  - Each realm has its own login page
  - Each realm issues its own JWT tokens
  - APISIX verifies tokens using the correct realm's JWKS endpoint
```

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│             KEYCLOAK ARCHITECTURE IN ZENITH                              │
│                                                                          │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  KEYCLOAK SERVER                                                   │  │
│  │  Namespace: keycloak                                               │  │
│  │  URL: https://auth.stage.freezenith.com                           │  │
│  │  Resources: 200m-1000m CPU, 512Mi-1Gi RAM                        │  │
│  │                                                                    │  │
│  │  ┌──────────────────────────────────────────────────────────────┐ │  │
│  │  │  master realm                                                │ │  │
│  │  │  ├── Admin user: admin (Zenith operators)                    │ │  │
│  │  │  └── Used for: Admin Console, Admin API access               │ │  │
│  │  ├──────────────────────────────────────────────────────────────┤ │  │
│  │  │  customer-acme realm                                         │ │  │
│  │  │  ├── Client: zenith-app (OIDC confidential)                  │ │  │
│  │  │  ├── Users: alice@acme.com, bob@acme.com                    │ │  │
│  │  │  ├── Roles: admin, developer, viewer                        │ │  │
│  │  │  └── JWKS: /realms/customer-acme/protocol/openid-connect/    │ │  │
│  │  │           certs                                               │ │  │
│  │  ├──────────────────────────────────────────────────────────────┤ │  │
│  │  │  customer-xyz realm                                          │ │  │
│  │  │  ├── Client: zenith-app                                      │ │  │
│  │  │  ├── Users: admin@xyz.io                                    │ │  │
│  │  │  └── ...                                                     │ │  │
│  │  └──────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────┬───────────────────────────────────┘  │
│                                  │ SQL (:5432)                           │
│                                  ▼                                       │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  KEYCLOAK DATABASE (CNPG Cluster)                                  │  │
│  │  Namespace: keycloak                                               │  │
│  │  Name: keycloak-pg                                                 │  │
│  │  Instances: 2 (staging) / 3 (production)                          │  │
│  │  Storage: 10Gi hcloud-volumes                                     │  │
│  │  Backup: WAL → s3://zenith-backups/keycloak-wal/                  │  │
│  │  Monitoring: PodMonitor → Prometheus (:9187/metrics)              │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  CONNECTIONS TO KEYCLOAK:                                                │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                                                                    │  │
│  │  zenith-api ───── Admin API ──────▶ Keycloak (:8080)              │  │
│  │  (creates realms, clients)          /admin/realms/...              │  │
│  │                                                                    │  │
│  │  APISIX ────────── JWKS fetch ────▶ Keycloak (:8080)              │  │
│  │  (verify JWT tokens)                /realms/<name>/protocol/       │  │
│  │                                     openid-connect/certs           │  │
│  │                                                                    │  │
│  │  Customer app ──── Login redirect ▶ Keycloak (:8080)              │  │
│  │  (user authentication)              /realms/<name>/protocol/       │  │
│  │                                     openid-connect/auth            │  │
│  │                                                                    │  │
│  │  Grafana ───────── OAuth login ───▶ Keycloak (:8080)              │  │
│  │  (optional, admin SSO)              /realms/master/...            │  │
│  │                                                                    │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Realm-Per-Tenant Model

```
┌─────────────────────────────────────────────────────────────────────────┐
│             REALM-PER-TENANT MODEL                                       │
│                                                                          │
│  WHY one realm per customer?                                            │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  • Complete isolation: Customer A can't see Customer B's users    │ │
│  │  • Independent login pages: Each realm has its own branding       │ │
│  │  • Independent JWKS: Each realm signs tokens with its own keys   │ │
│  │  • Independent roles: Each customer defines their own roles      │ │
│  │  • Easy deletion: Delete realm = all customer identity is gone   │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  REALM NAMING CONVENTION:                                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Customer domain: customer.acme.com                               │ │
│  │  Realm name:      customer-acme (dots replaced with dashes)       │ │
│  │                                                                    │ │
│  │  Code: strings.ReplaceAll(domain, ".", "-")                       │ │
│  │  File: services/api/internal/temporal/activities.go               │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  WHAT EACH REALM CONTAINS:                                               │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Realm: customer-acme                                              │ │
│  │  │                                                                 │ │
│  │  ├── Client: zenith-app                                           │ │
│  │  │   ├── Type: OIDC Confidential                                  │ │
│  │  │   ├── Protocol: openid-connect                                 │ │
│  │  │   ├── Direct Access Grants: enabled                            │ │
│  │  │   ├── Redirect URI: https://customer-acme.freezenith.com/*    │ │
│  │  │   └── Client Secret: (stored in K8s Secret)                   │ │
│  │  │                                                                 │ │
│  │  ├── Users                                                         │ │
│  │  │   ├── alice@acme.com (admin role)                              │ │
│  │  │   └── bob@acme.com (developer role)                            │ │
│  │  │                                                                 │ │
│  │  ├── Roles                                                         │ │
│  │  │   ├── admin                                                     │ │
│  │  │   ├── developer                                                │ │
│  │  │   └── viewer                                                    │ │
│  │  │                                                                 │ │
│  │  └── JWKS Endpoint                                                 │ │
│  │      https://auth.freezenith.com/realms/customer-acme/             │ │
│  │      protocol/openid-connect/certs                                 │ │
│  │      (APISIX fetches public keys from here)                        │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 4. How Realms Are Created

```
┌─────────────────────────────────────────────────────────────────────────┐
│             REALM CREATION FLOW (automated via Temporal)                  │
│                                                                          │
│  User clicks "Sign Up" on freezenith.com                                │
│         │                                                                │
│         ▼                                                                │
│  zenith-api: POST /api/v1/auth/register                                 │
│         │                                                                │
│         ▼                                                                │
│  Temporal workflow: ProvisionCustomer                                    │
│         │                                                                │
│         ▼                                                                │
│  Activity: CreateKeycloakRealm                                           │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                                                                    │ │
│  │  1. Convert domain to realm name                                  │ │
│  │     "customer.acme.com" → "customer-acme"                        │ │
│  │                                                                    │ │
│  │  2. Login to Keycloak Admin API                                   │ │
│  │     POST /realms/master/protocol/openid-connect/token             │ │
│  │     { grant_type: "password", client_id: "admin-cli",             │ │
│  │       username: "admin", password: <admin_password> }             │ │
│  │     → returns admin access_token                                  │ │
│  │                                                                    │ │
│  │  3. Create realm                                                  │ │
│  │     POST /admin/realms                                            │ │
│  │     { realm: "customer-acme", enabled: true,                      │ │
│  │       displayName: "Acme Corp" }                                  │ │
│  │                                                                    │ │
│  │  4. Create OIDC client in the realm                               │ │
│  │     POST /admin/realms/customer-acme/clients                      │ │
│  │     { clientId: "zenith-app", protocol: "openid-connect",         │ │
│  │       directAccessGrantsEnabled: true,                            │ │
│  │       redirectUris: ["https://customer-acme.freezenith.com/*"] }  │ │
│  │     → returns client UUID                                         │ │
│  │                                                                    │ │
│  │  5. Get client secret                                             │ │
│  │     GET /admin/realms/customer-acme/clients/{uuid}/client-secret  │ │
│  │     → returns client_secret                                       │ │
│  │                                                                    │ │
│  │  6. Store client_secret in K8s Secret                             │ │
│  │     namespace: zenith-<customer>                                  │ │
│  │     secret: keycloak-credentials                                  │ │
│  │     data: { realm, client_id, client_secret }                     │ │
│  │                                                                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  Retry policy: 5 attempts, 10s initial backoff, 2x multiplier          │
│  If all retries fail: workflow sets customer status to "error"          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Admin Console Guide

### Accessing Keycloak Admin Console

```bash
# Via public URL
open https://auth.stage.freezenith.com/admin/master/console/

# Via port-forward (if DNS not available)
kubectl port-forward -n keycloak svc/keycloak 9080:80
open http://localhost:9080/admin/master/console/

# Login credentials:
# Username: admin
# Password: <keycloak_admin_password from Terraform variables>
# (stored in keycloak-admin Secret in zenith-staging namespace)
```

### Common Admin Tasks

```
┌─────────────────────────────────────────────────────────────────────────┐
│             KEYCLOAK ADMIN TASKS                                         │
│                                                                          │
│  VIEW ALL REALMS:                                                       │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  1. Login to admin console                                         │ │
│  │  2. Click realm dropdown (top-left, shows "master")               │ │
│  │  3. See all realms — each is a customer                           │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  VIEW USERS IN A REALM:                                                  │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  1. Switch to the customer's realm (dropdown)                      │ │
│  │  2. Navigate: Users → View all users                              │ │
│  │  3. See all registered users in that realm                        │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  RESET A USER'S PASSWORD:                                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  1. Switch to the customer's realm                                 │ │
│  │  2. Users → find the user → click                                 │ │
│  │  3. Credentials tab → Set Password                                │ │
│  │  4. Enter new password, toggle "Temporary" off                    │ │
│  │  5. Save                                                           │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  VIEW CLIENT CONFIGURATION:                                              │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  1. Switch to the customer's realm                                 │ │
│  │  2. Clients → click "zenith-app"                                  │ │
│  │  3. See: redirect URIs, client secret, grant types                │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  DELETE A REALM (removes all customer identity):                        │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  1. Switch to the realm                                            │ │
│  │  2. Realm Settings → Action → Delete                              │ │
│  │  3. Confirm deletion                                               │ │
│  │                                                                    │ │
│  │  WARNING: This is irreversible! All users, clients, and           │ │
│  │  sessions in that realm are permanently deleted.                   │ │
│  │  The Temporal deprovision workflow handles this automatically.    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 6. JWT Token Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│             JWT TOKEN LIFECYCLE                                          │
│                                                                          │
│  Step 1: User logs in via customer frontend                             │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  POST https://auth.freezenith.com/realms/customer-acme/           │ │
│  │       protocol/openid-connect/token                               │ │
│  │  Body: {                                                           │ │
│  │    grant_type: "password",                                        │ │
│  │    client_id: "zenith-app",                                       │ │
│  │    client_secret: "<from K8s Secret>",                            │ │
│  │    username: "alice@acme.com",                                    │ │
│  │    password: "..."                                                │ │
│  │  }                                                                 │ │
│  │  → Response: { access_token, refresh_token, expires_in }         │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Step 2: Frontend stores tokens in localStorage                         │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  localStorage.setItem("access_token", access_token)               │ │
│  │  localStorage.setItem("refresh_token", refresh_token)             │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Step 3: Frontend makes API call with Bearer token                      │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  GET https://api.customer-acme.freezenith.com/api/v1/apps         │ │
│  │  Authorization: Bearer <access_token>                             │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Step 4: APISIX verifies JWT                                            │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  1. Extract Bearer token from Authorization header                │ │
│  │  2. Fetch JWKS from Keycloak:                                     │ │
│  │     GET https://auth.freezenith.com/realms/customer-acme/         │ │
│  │         protocol/openid-connect/certs                             │ │
│  │  3. Verify: signature, expiration, audience                       │ │
│  │  4. If valid → forward to backend with X-Consumer-* headers      │ │
│  │  5. If invalid → return 401 Unauthorized                         │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Step 5: Backend receives verified request                              │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  zenith-api trusts APISIX headers (no re-verification needed)     │ │
│  │  Extracts customer ID from token claims                           │ │
│  │  Processes the request                                             │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. API Integration (Go Code)

### Files

| File | Purpose |
|------|---------|
| `services/api/internal/adapters/keycloakclient/client.go` | Real Keycloak client (gocloak v13) |
| `services/api/internal/adapters/keycloakclient/memory.go` | In-memory mock (for dev/test) |
| `services/api/internal/temporal/activities.go` | Temporal activity: CreateKeycloakRealm |
| `services/api/internal/temporal/workflow.go` | Workflow calling the activity |

### KeycloakAPI Interface

```go
// Defined in ports/ — the contract between service and adapter
type KeycloakAPI interface {
    CreateRealm(ctx context.Context, realmName, displayName string) error
    DeleteRealm(ctx context.Context, realmName string) error
    CreateClient(ctx context.Context, realmName, clientID, redirectURI string) (string, error)
}
```

### Mode Selection

| ZENITH_MODE | Keycloak Client | Behavior |
|-------------|----------------|----------|
| `standalone` | `MemoryKeycloakClient` | No-op (no realm creation) |
| `saas` | `KeycloakClient` (real) | Creates realms via Admin API |

---

## 8. Configuration Reference

### Terraform File

**File:** `infra/terraform/modules/k8s-platform/identity.tf`

| Setting | Value |
|---------|-------|
| Chart | bitnami keycloak (OCI) |
| Version | 25.2.0 |
| Namespace | keycloak |
| Image | docker.io/bitnamilegacy/keycloak |
| Admin user | admin |
| Admin password | var.keycloak_admin_password |
| External DB host | keycloak-pg-rw.keycloak.svc.cluster.local |
| External DB port | 5432 |
| External DB name | keycloak |
| CPU requests/limits | 200m / 1000m |
| Memory requests/limits | 512Mi / 1Gi |
| Priority class | infra-critical |
| Proxy headers | xforwarded |
| Depends on | CNPG operator, keycloak-pg cluster |

### Keycloak Database (CNPG)

**File:** `infra/terraform/modules/k8s-platform/storage.tf`

| Setting | Value |
|---------|-------|
| Cluster name | keycloak-pg |
| Instances | 2 (staging) / 3 (production) |
| Storage | 10Gi hcloud-volumes |
| Database | keycloak |
| Owner | keycloak |
| max_connections | 100 |
| shared_buffers | 128MB |
| WAL backup | s3://zenith-backups/keycloak-wal/ |
| Backup retention | 14 days |

---

## 9. Troubleshooting

### User can't login

```bash
# Step 1: Check if Keycloak is running
kubectl -n keycloak get pods
kubectl -n keycloak logs deploy/keycloak --tail=50

# Step 2: Check if the realm exists
# Admin Console → realm dropdown → look for customer's realm

# Step 3: Check if client is configured correctly
# Admin Console → realm → Clients → zenith-app
# Verify: redirect URI matches the customer's domain

# Step 4: Test token endpoint directly
curl -X POST https://auth.stage.freezenith.com/realms/<realm>/protocol/openid-connect/token \
  -d "grant_type=password" \
  -d "client_id=zenith-app" \
  -d "client_secret=<secret>" \
  -d "username=<user>" \
  -d "password=<pass>"
```

### APISIX returns 401 but token is valid

```bash
# Check if APISIX can reach Keycloak JWKS endpoint
kubectl -n apisix exec deploy/apisix -- \
  curl -s http://keycloak.keycloak.svc.cluster.local/realms/<realm>/protocol/openid-connect/certs

# Check if JWKS response is valid JSON with keys
# If empty or error: Keycloak might be overloaded or the realm doesn't exist

# Check APISIX jwt-auth plugin configuration
kubectl -n apisix get apisixpluginconfig -o yaml
```

### Keycloak is slow / unresponsive

```bash
# Check database connectivity
kubectl -n keycloak exec deploy/keycloak -- \
  env | grep DB  # Check DB connection vars

# Check database performance
kubectl -n keycloak get cluster keycloak-pg
kubectl -n keycloak exec keycloak-pg-1 -- psql -U postgres -c \
  "SELECT count(*) FROM pg_stat_activity WHERE state = 'active';"

# Check Keycloak resource usage
kubectl -n keycloak top pods

# Restart if needed (rolling restart, no downtime with 2+ replicas)
kubectl -n keycloak rollout restart deploy/keycloak
```

### Realm creation failed in Temporal

```bash
# Check Temporal workflow status
# Open https://temporal.stage.freezenith.com
# Find the ProvisionCustomer workflow
# Check the CreateKeycloakRealm activity for errors

# Common errors:
# • "Realm already exists" → customer signed up twice
# • "401 Unauthorized" → admin credentials wrong (check keycloak-admin Secret)
# • "Connection refused" → Keycloak not running

# Check the admin credentials Secret
kubectl -n zenith-staging get secret keycloak-admin -o jsonpath='{.data}' | \
  python3 -c "import sys,json,base64; d=json.load(sys.stdin); [print(k,base64.b64decode(v).decode()) for k,v in d.items()]"
```

---

## 10. Upgrade Path

### Upgrading Keycloak

```bash
# 1. Update version in variables.tf
variable "keycloak_version" {
  default = "26.0.0"  # new version
}

# 2. Plan and apply
terraform plan -target=helm_release.keycloak
terraform apply -target=helm_release.keycloak

# 3. Verify
kubectl -n keycloak get pods
kubectl -n keycloak logs deploy/keycloak --tail=20

# 4. Test login to admin console
open https://auth.stage.freezenith.com/admin/master/console/

# NOTE: Keycloak runs database migrations automatically on startup.
# The CNPG database is backed up continuously via WAL archiving.
# If upgrade fails, you can restore from WAL backup.
```

### Adding a Custom Theme

```bash
# 1. Create a theme directory in the Keycloak image
# 2. Mount via ConfigMap or custom Docker image
# 3. Set theme in realm settings:
#    Admin Console → Realm Settings → Themes → Login Theme
```
