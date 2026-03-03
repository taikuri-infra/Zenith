# 13 — APISIX API Gateway

> **Purpose:** Understand how API traffic is routed, authenticated, rate-limited, and traced through APISIX.
> **Audience:** Any developer who needs to add API routes, debug auth issues, or understand the gateway layer.
> **Last Updated:** 2026-03-03
> **Related:** [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) (installation steps), [12-traefik-ingress.md](./12-traefik-ingress.md) (Traefik routes traffic to APISIX), [10-backend-architecture.md](./10-backend-architecture.md) (Go backend behind APISIX)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose It](#2-why-we-chose-it)
3. [Architecture Diagram](#3-architecture-diagram)
4. [How a Request Flows Through APISIX](#4-how-a-request-flows-through-apisix)
5. [Plugin Pipeline](#5-plugin-pipeline)
6. [Route Configuration](#6-route-configuration)
7. [etcd — The Configuration Store](#7-etcd--the-configuration-store)
8. [Configuration Reference](#8-configuration-reference)
9. [Troubleshooting](#9-troubleshooting)
10. [Upgrade Path](#10-upgrade-path)

---

## 1. Overview

**APISIX** is the API gateway for all backend API traffic in Zenith. It sits between Traefik and the backend pods, handling:

- **JWT authentication** — Verifies tokens issued by Keycloak
- **CORS** — Cross-Origin Resource Sharing headers
- **Rate limiting** — Per-route request throttling
- **OpenTelemetry** — Distributed tracing spans
- **Prometheus metrics** — Gateway performance metrics

APISIX does NOT handle TLS termination (Traefik does that) and does NOT handle frontend traffic (only API routes go through APISIX).

```
What goes through APISIX:     api.stage.freezenith.com/v1/*
What does NOT go through:      stage.freezenith.com (landing)
                               app.stage.freezenith.com (web dashboard)
                               argocd.stage.freezenith.com (ArgoCD UI)
                               Customer frontend apps
```

---

## 2. Why We Chose It

| Feature | APISIX | Kong OSS | Envoy | Traefik |
|---------|--------|----------|-------|---------|
| etcd-backed (no DB) | Yes | No (needs PG) | N/A | N/A |
| Plugin ecosystem | 80+ built-in | 40+ (Enterprise) | Filters (complex) | Middleware |
| JWT verification | Built-in plugin | Built-in | Custom filter | Not built-in |
| CORS plugin | Built-in | Built-in | Custom | Built-in |
| Rate limiting | Built-in (in-memory) | Built-in (needs Redis) | External service | Built-in |
| OpenTelemetry | Built-in plugin | Enterprise only | Built-in | Limited |
| K8s CRD support | Yes (ingress controller) | Yes | Yes (Gateway API) | Yes |
| Cost | Free (Apache 2.0) | Free + Enterprise | Free | Free |

**Decision:** APISIX offers the richest free plugin ecosystem with native etcd storage (no PostgreSQL dependency). Kong Enterprise features like OTel would cost $$$. Babak also wanted to learn a new tool rather than staying with Kong.

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         APISIX IN THE ZENITH CLUSTER                        │
│                         Namespace: apisix                                   │
│                                                                             │
│  From Traefik (via ExternalName service)                                    │
│      │                                                                      │
│      │ HTTP :9080                                                           │
│      ▼                                                                      │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    APISIX GATEWAY POD                                  │  │
│  │                    Replicas: 1 (staging) / 2 (production)             │  │
│  │                    PriorityClass: infra-critical (500000)             │  │
│  │                    Resources: 100m-500m CPU, 256Mi-512Mi RAM          │  │
│  │                                                                       │  │
│  │  ┌────────────────────────────────────────────────────────────────┐   │  │
│  │  │                    PLUGIN PIPELINE                              │   │  │
│  │  │                                                                │   │  │
│  │  │  Request enters                                                │   │  │
│  │  │      │                                                         │   │  │
│  │  │      ▼                                                         │   │  │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │   │  │
│  │  │  │ jwt-auth │  │ cors     │  │ limit-   │  │ opentelemetry│  │   │  │
│  │  │  │          │  │          │  │ count    │  │              │  │   │  │
│  │  │  │ Verify   │  │ Add      │  │ Check    │  │ Create trace │  │   │  │
│  │  │  │ JWT from │  │ CORS     │  │ rate     │  │ span, add    │  │   │  │
│  │  │  │ Keycloak │  │ headers  │  │ limit    │  │ to OTel      │  │   │  │
│  │  │  │ JWKS     │  │          │  │ counter  │  │ collector    │  │   │  │
│  │  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────┬───────┘  │   │  │
│  │  │       │ Pass/Fail    │ Always      │ Pass/429      │ Always   │   │  │
│  │  │       ▼              ▼             ▼               ▼          │   │  │
│  │  │  ┌──────────────────────────────────────────────────────────┐ │   │  │
│  │  │  │                    PROMETHEUS PLUGIN                     │ │   │  │
│  │  │  │  Exposes metrics at :9091 for Prometheus scraping        │ │   │  │
│  │  │  └──────────────────────────────────────────────────────────┘ │   │  │
│  │  │       │                                                        │   │  │
│  │  │       ▼ Forward to upstream                                    │   │  │
│  │  └────────────────────────────────────────────────────────────────┘   │  │
│  │                                                                       │  │
│  │  PORTS:                                                               │  │
│  │    :9080  — Gateway (HTTP, receives traffic from Traefik)             │  │
│  │    :9180  — Admin API (internal only, for ingress controller)         │  │
│  │    :9091  — Prometheus metrics                                        │  │
│  │    :9443  — Gateway (HTTPS, not used — Traefik handles TLS)          │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                   │                                       ▲                  │
│                   │ HTTP :8080                            │                  │
│                   ▼                                       │                  │
│  ┌────────────────────────┐         ┌────────────────────┴────────────┐    │
│  │ zenith-api              │         │ etcd (apisix namespace)         │    │
│  │ (zenith-staging ns)     │         │ Replicas: 1 (stg) / 3 (prod)  │    │
│  │ Go backend :8080        │         │ PVC: 5Gi on hcloud-volumes     │    │
│  │ Receives verified       │         │ Stores: routes, plugins,       │    │
│  │ requests with           │         │   upstreams, consumers         │    │
│  │ X-Consumer-* headers    │         │ APISIX reads config from etcd  │    │
│  └────────────────────────┘         └─────────────────────────────────┘    │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ APISIX INGRESS CONTROLLER (apisix namespace)                          │  │
│  │                                                                       │  │
│  │  Watches K8s for ApisixRoute, ApisixPluginConfig CRDs                 │  │
│  │  Translates CRDs → APISIX Admin API calls → stored in etcd           │  │
│  │  Admin API version: v3                                                │  │
│  │  Resources: 50m CPU, 128Mi RAM                                        │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. How a Request Flows Through APISIX

### Protected API Request (JWT Required)

```
POST https://api.stage.freezenith.com/v1/apps
Header: Authorization: Bearer eyJhbGciOiJSUzI1NiIs...
Header: Origin: https://app.stage.freezenith.com
    │
    │ (Already through Cloudflare → Traefik → TLS terminated)
    │ Arrives at APISIX :9080 as plain HTTP
    ▼
┌─────────────────────────────────────────────────────────────────┐
│  APISIX GATEWAY                                                  │
│                                                                  │
│  Step 1: ROUTE MATCHING                                          │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Match: URI prefix = /v1/apps                               │ │
│  │ Host: api.stage.freezenith.com                             │ │
│  │ Matched route: "zenith-api-protected"                      │ │
│  │ Plugins attached: jwt-auth + cors + limit-count + otel     │ │
│  └────────────────────────────────────────────────────────────┘ │
│                         │                                        │
│                         ▼                                        │
│  Step 2: JWT-AUTH PLUGIN                                         │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ 1. Extract token from Authorization: Bearer <token>        │ │
│  │ 2. Decode token header → get kid (key ID)                  │ │
│  │ 3. Fetch JWKS from Keycloak:                               │ │
│  │    GET http://keycloak.keycloak.svc:8080/realms/zenith/    │ │
│  │        protocol/openid-connect/certs                       │ │
│  │ 4. Find matching public key by kid                         │ │
│  │ 5. Verify:                                                 │ │
│  │    ✓ Signature (RSA256 with Keycloak's public key)         │ │
│  │    ✓ Expiration (exp claim not passed)                     │ │
│  │    ✓ Audience (aud matches expected client)                │ │
│  │    ✓ Issuer (iss matches Keycloak realm URL)               │ │
│  │ 6. If VALID:                                               │ │
│  │    → Add X-Consumer-Username: user@email.com               │ │
│  │    → Add X-Consumer-Custom-ID: uuid-of-user                │ │
│  │    → Continue to next plugin                               │ │
│  │ 7. If INVALID:                                             │ │
│  │    → Return 401 Unauthorized immediately                   │ │
│  │    → Do NOT forward to backend                             │ │
│  └────────────────────────────────────────────────────────────┘ │
│                         │ (if valid)                             │
│                         ▼                                        │
│  Step 3: CORS PLUGIN                                             │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ 1. Check Origin header against allowed origins:            │ │
│  │    - https://app.stage.freezenith.com                      │ │
│  │    - https://stage.freezenith.com                          │ │
│  │ 2. Add response headers:                                   │ │
│  │    Access-Control-Allow-Origin: https://app.stage...       │ │
│  │    Access-Control-Allow-Methods: GET,POST,PUT,DELETE        │ │
│  │    Access-Control-Allow-Headers: Authorization,Content-Type│ │
│  │ 3. For preflight OPTIONS → return 200 immediately          │ │
│  └────────────────────────────────────────────────────────────┘ │
│                         │                                        │
│                         ▼                                        │
│  Step 4: LIMIT-COUNT PLUGIN                                      │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ 1. Check in-memory counter for this route + client IP      │ │
│  │ 2. If under limit → increment counter, continue            │ │
│  │ 3. If over limit → return 429 Too Many Requests            │ │
│  │ 4. Counter resets per configured time window                │ │
│  └────────────────────────────────────────────────────────────┘ │
│                         │ (if under limit)                       │
│                         ▼                                        │
│  Step 5: OPENTELEMETRY PLUGIN                                    │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ 1. Create trace span: "APISIX → zenith-api"               │ │
│  │ 2. Add attributes: route, method, status_code              │ │
│  │ 3. Export span to OTel Collector:                          │ │
│  │    otel-collector.monitoring.svc:4317 (gRPC)               │ │
│  │ 4. Inject trace headers into forwarded request             │ │
│  └────────────────────────────────────────────────────────────┘ │
│                         │                                        │
│                         ▼                                        │
│  Step 6: FORWARD TO UPSTREAM                                     │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Forward: HTTP POST http://zenith-api.zenith-staging:8080   │ │
│  │          /v1/apps                                          │ │
│  │ Headers: original + X-Consumer-Username + trace headers    │ │
│  │ Body: original request body (unmodified)                   │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Public Webhook Request (No JWT)

```
POST https://api.stage.freezenith.com/v1/webhooks/github
Header: X-Hub-Signature-256: sha256=abc123...
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│  APISIX GATEWAY                                                  │
│                                                                  │
│  Route: "zenith-api-webhooks"                                    │
│  URI prefix: /v1/webhooks/*                                      │
│  Plugins: cors + limit-count (NO jwt-auth!)                      │
│                                                                  │
│  cors: Add CORS headers                                          │
│  limit-count: Stricter limit (webhooks are less frequent)        │
│                                                                  │
│  Forward: HTTP POST zenith-api:8080/v1/webhooks/github           │
│  (zenith-api verifies webhook signature itself using HMAC)       │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Plugin Pipeline

The enabled plugins and their execution order:

```
┌─────────────────────────────────────────────────────────────────┐
│                    APISIX PLUGIN EXECUTION ORDER                 │
│                    (configured in gateway.tf)                    │
│                                                                  │
│  Plugins list:                                                   │
│    ["jwt-auth", "cors", "limit-count", "openid-connect",        │
│     "opentelemetry", "prometheus"]                               │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Phase: access (before forwarding to upstream)             │   │
│  │                                                           │   │
│  │  1. jwt-auth        — Verify JWT token (if route uses it) │   │
│  │  2. openid-connect  — OIDC integration (if route uses it)│   │
│  │  3. cors            — Add CORS headers                    │   │
│  │  4. limit-count     — Check rate limit                    │   │
│  │  5. opentelemetry   — Start trace span                    │   │
│  └──────────────────────────────────────────────────────────┘   │
│                         │                                        │
│                         ▼                                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Phase: header_filter (after upstream responds)            │   │
│  │                                                           │   │
│  │  - cors: Add Access-Control-* response headers            │   │
│  └──────────────────────────────────────────────────────────┘   │
│                         │                                        │
│                         ▼                                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Phase: log (after response sent)                          │   │
│  │                                                           │   │
│  │  - prometheus: Record request metrics                     │   │
│  │  - opentelemetry: End trace span, export                  │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 6. Route Configuration

### How Routes Are Configured

```
┌─────────────────────────────────────────────────────────────────┐
│              ROUTE CONFIGURATION FLOW                             │
│                                                                  │
│  Developer creates ApisixRoute CRD in K8s:                       │
│                                                                  │
│  ┌──────────────┐     ┌──────────────────┐     ┌─────────────┐ │
│  │ ApisixRoute  │────▶│ APISIX Ingress   │────▶│ APISIX      │ │
│  │ CRD (YAML)   │     │ Controller       │     │ Admin API   │ │
│  │              │     │ (watches CRDs,   │     │ (:9180)     │ │
│  │              │     │  translates to    │     │             │ │
│  │              │     │  Admin API calls) │     │ Stores in   │ │
│  │              │     │                  │     │ etcd         │ │
│  └──────────────┘     └──────────────────┘     └──────┬──────┘ │
│                                                        │        │
│                                                        ▼        │
│                                                  ┌───────────┐  │
│                                                  │ etcd      │  │
│                                                  │ Routes,   │  │
│                                                  │ plugins,  │  │
│                                                  │ upstreams │  │
│                                                  └───────────┘  │
│                                                                  │
│  APISIX gateway reads route config from etcd in real-time        │
│  No restart needed — routes are live immediately                 │
└─────────────────────────────────────────────────────────────────┘
```

### Current Routes

| Route | URI Match | Plugins | Upstream | Purpose |
|-------|-----------|---------|----------|---------|
| zenith-api-protected | `/v1/*` | jwt-auth + cors + limit-count + otel | zenith-api:8080 | All authenticated API calls |
| zenith-api-public | `/v1/auth/*` | cors + limit-count | zenith-api:8080 | Login, register (no JWT) |
| zenith-api-webhooks | `/v1/webhooks/*` | cors + limit-count | zenith-api:8080 | GitHub webhooks (no JWT) |

---

## 7. etcd — The Configuration Store

```
┌─────────────────────────────────────────────────────────────────┐
│                    etcd IN APISIX                                 │
│                                                                  │
│  What is etcd?                                                   │
│    A distributed key-value store (same one K8s uses internally)  │
│    APISIX uses its own etcd (separate from k3s etcd)             │
│                                                                  │
│  Why etcd instead of PostgreSQL?                                 │
│    - Faster reads (in-memory + disk)                             │
│    - Built-in watch (APISIX gets instant config updates)         │
│    - No external DB dependency for the gateway                   │
│    - Smaller footprint than a full PostgreSQL                    │
│                                                                  │
│  Deployment:                                                     │
│    Staging:    1 replica (no HA, acceptable for staging)          │
│    Production: 3 replicas (Raft consensus, survives 1 failure)   │
│    Storage: 5Gi PVC on hcloud-volumes (Retain policy)            │
│                                                                  │
│  What's stored in etcd:                                          │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ /apisix/routes/       → All route definitions              │ │
│  │ /apisix/upstreams/    → Backend service targets            │ │
│  │ /apisix/plugins/      → Plugin configurations              │ │
│  │ /apisix/consumers/    → API consumers (JWT keys)           │ │
│  │ /apisix/ssl/          → TLS certificates (not used — Traef)│ │
│  │ /apisix/services/     → Service abstractions               │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  APISIX watches etcd for changes:                                │
│    etcd sends notification → APISIX updates routing table        │
│    Zero-downtime route changes (no reload, no restart)           │
└─────────────────────────────────────────────────────────────────┘
```

---

## 8. Configuration Reference

### Terraform File

**File:** `infra/terraform/modules/k8s-platform/gateway.tf`

### APISIX Settings

| Setting | Staging | Production | Purpose |
|---------|---------|------------|---------|
| `replicaCount` | 1 | 2 | Gateway pod replicas |
| `etcd.replicaCount` | 1 | 3 | etcd cluster size |
| `etcd.persistence.size` | 5Gi | 5Gi | etcd storage |
| `etcd.persistence.storageClass` | hcloud-volumes | hcloud-volumes | Block storage |
| `gateway.type` | ClusterIP | ClusterIP | Not exposed directly (Traefik fronts it) |
| `admin.allow.list` | 127.0.0.1/24, 10.42.0.0/16 | same | Admin API access control |
| `resources.requests.cpu` | 100m | 100m | CPU request |
| `resources.requests.memory` | 256Mi | 256Mi | Memory request |
| `resources.limits.cpu` | 500m | 500m | CPU limit |
| `resources.limits.memory` | 512Mi | 512Mi | Memory limit |
| `priorityClassName` | infra-critical | infra-critical | Eviction protection |

### Enabled Plugins

```
jwt-auth         — JWT token verification (Keycloak JWKS)
cors             — Cross-Origin Resource Sharing
limit-count      — Request rate limiting (in-memory counter)
openid-connect   — OIDC integration (for Keycloak SSO)
opentelemetry    — Distributed tracing (export to OTel Collector)
prometheus       — Metrics exposure (:9091)
```

### APISIX Ingress Controller Settings

| Setting | Value | Purpose |
|---------|-------|---------|
| `config.apisix.adminAPIVersion` | v3 | Admin API version |
| `config.apisix.serviceNamespace` | apisix | Namespace where APISIX runs |
| `resources.requests.cpu` | 50m | Controller CPU |
| `resources.requests.memory` | 128Mi | Controller memory |

---

## 9. Troubleshooting

### "401 Unauthorized" on API calls

```bash
# 1. Check if JWT token is valid
# Decode it (jwt.io or command line):
echo "eyJhbGciOiJSUzI1NiIs..." | cut -d'.' -f2 | base64 -d | jq .
# Check: exp (not expired?), iss (correct Keycloak URL?), aud (correct client?)

# 2. Check if Keycloak JWKS endpoint is reachable from APISIX
kubectl exec -n apisix deploy/apisix -- curl -s \
  http://keycloak.keycloak.svc:8080/realms/zenith/protocol/openid-connect/certs | jq .

# 3. Check APISIX logs for auth errors
kubectl logs -n apisix deploy/apisix --tail=50

# 4. Check if the route has jwt-auth plugin enabled
kubectl get apisixroute -A -o yaml | grep jwt-auth
```

### "429 Too Many Requests"

```bash
# Rate limit exceeded
# Check limit-count configuration in APISIX:
kubectl exec -n apisix deploy/apisix -- \
  curl -s http://127.0.0.1:9180/apisix/admin/routes -H 'X-API-KEY: your-key' | jq .

# Reset: rate limits use in-memory counters — restarting APISIX resets them
# (only for debugging — don't restart in production to fix rate limits)
```

### "502 Bad Gateway" from APISIX

```bash
# APISIX can reach the route but the upstream (backend) is down

# 1. Check if zenith-api is running
kubectl get pods -n zenith-staging -l app=zenith-api

# 2. Check if the service has endpoints
kubectl get endpoints zenith-api -n zenith-staging
# If <none>, the pod is not ready

# 3. Test direct connectivity from APISIX to backend
kubectl exec -n apisix deploy/apisix -- \
  curl -s http://zenith-api.zenith-staging.svc:8080/health
```

### "APISIX not receiving traffic"

```bash
# 1. Check Traefik IngressRoute for API
kubectl get ingressroute -n zenith-staging -o yaml | grep api

# 2. Check ExternalName service exists
kubectl get svc apisix-gateway-proxy -n zenith-staging -o yaml
# Should show: type: ExternalName
# externalName: apisix-gateway.apisix.svc.cluster.local

# 3. Check APISIX gateway service
kubectl get svc -n apisix apisix-gateway
# Should be ClusterIP with port 9080

# 4. Test APISIX directly
kubectl port-forward svc/apisix-gateway -n apisix 9080:9080
curl -H "Host: api.stage.freezenith.com" http://localhost:9080/v1/health
```

### etcd issues

```bash
# Check etcd health
kubectl exec -n apisix deploy/apisix-etcd -- etcdctl endpoint health

# Check etcd member list
kubectl exec -n apisix deploy/apisix-etcd -- etcdctl member list

# Check etcd storage usage
kubectl exec -n apisix deploy/apisix-etcd -- etcdctl endpoint status --write-out=table

# Backup etcd (route config):
kubectl exec -n apisix deploy/apisix-etcd -- \
  etcdctl snapshot save /tmp/etcd-backup.db
kubectl cp apisix/apisix-etcd-0:/tmp/etcd-backup.db ./etcd-backup.db
```

### Ingress controller not syncing CRDs

```bash
# Check ingress controller logs
kubectl logs -n apisix deploy/apisix-ingress-controller --tail=50

# Check if CRDs are recognized
kubectl get apisixroute -A
kubectl get apisixpluginconfig -A

# Restart ingress controller
kubectl rollout restart deploy/apisix-ingress-controller -n apisix
```

---

## 10. Upgrade Path

### Upgrading APISIX

```bash
# 1. Update version in variables.tf
# variable "apisix_version" { default = "2.10.0" }

# 2. Plan and apply
cd infra/terraform/staging-k8s
terraform plan -target=helm_release.apisix
terraform apply -target=helm_release.apisix

# 3. Verify
kubectl get pods -n apisix
kubectl logs -n apisix deploy/apisix --tail=20
```

### Adding a new plugin

1. Add plugin name to the `plugins` list in `gateway.tf`:
   ```hcl
   plugins = ["jwt-auth", "cors", "limit-count", "openid-connect",
              "opentelemetry", "prometheus", "NEW-PLUGIN"]
   ```
2. `terraform apply`
3. Configure the plugin on routes via ApisixPluginConfig CRD

### Adding a new route

Create an ApisixRoute CRD:
```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: my-new-route
  namespace: zenith-staging
spec:
  http:
    - name: my-route
      match:
        hosts: ["api.stage.freezenith.com"]
        paths: ["/v1/my-endpoint/*"]
      backends:
        - serviceName: my-service
          servicePort: 8080
      plugins:
        - name: cors
          enable: true
```
