# 12 — Traefik Ingress Controller

> **Purpose:** Understand how all external traffic enters the Zenith cluster and gets routed to the correct service.
> **Audience:** Any developer who needs to debug routing issues, add new subdomains, or understand TLS.
> **Last Updated:** 2026-03-03
> **Related:** [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) (installation steps), [13-apisix-gateway.md](./13-apisix-gateway.md) (API gateway behind Traefik), [SYSTEM-MAP.md](./SYSTEM-MAP.md) (full system overview)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose It](#2-why-we-chose-it)
3. [Architecture Diagram](#3-architecture-diagram)
4. [How Traffic Flows Through Traefik](#4-how-traffic-flows-through-traefik)
5. [IngressRoute CRDs — The Routing Rules](#5-ingressroute-crds--the-routing-rules)
6. [TLS & Certificate Management](#6-tls--certificate-management)
7. [Cross-Namespace Routing](#7-cross-namespace-routing)
8. [Configuration Reference](#8-configuration-reference)
9. [Troubleshooting](#9-troubleshooting)
10. [Upgrade Path](#10-upgrade-path)

---

## 1. Overview

**Traefik** is the entry point for ALL traffic into the Zenith cluster. It:

- Terminates TLS (HTTPS) using certificates from cert-manager
- Routes requests to the correct backend service based on hostname
- Runs as a **DaemonSet** in `kube-system` (built into k3s)
- Uses **IngressRoute CRDs** (NOT standard Kubernetes Ingress)

**Key concept:** Traefik does NOT handle JWT auth, CORS, or rate-limiting. For API routes, Traefik forwards traffic to APISIX, which handles those concerns. For frontend routes, Traefik forwards directly to the pod.

```
Traefik's Two Routing Modes:

  Frontend traffic:   Traefik ──────────────────▶ Pod (direct)
  API traffic:        Traefik ──▶ APISIX ──▶ Pod (via gateway)
```

---

## 2. Why We Chose It

| Feature | Traefik | NGINX Ingress | HAProxy |
|---------|---------|---------------|---------|
| Built into k3s | Yes (zero setup) | No (manual install) | No |
| IngressRoute CRD | Yes (rich routing) | No (annotations only) | No |
| Cross-namespace routing | Yes (built-in) | Hacky (ExternalName) | No |
| Dashboard | Yes (built-in) | No | Yes |
| cert-manager integration | Native | Native | Manual |
| ExternalName services | Yes (first-class) | Limited | No |

**Decision:** k3s ships with Traefik. Rather than replacing it (which requires disabling it and installing another controller), we use it as the TLS termination layer and delegate API-specific concerns to APISIX.

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         TRAEFIK IN THE ZENITH CLUSTER                       │
│                                                                             │
│  INTERNET                                                                   │
│      │                                                                      │
│      │ HTTPS :443 / HTTP :80                                               │
│      ▼                                                                      │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    TRAEFIK (kube-system namespace)                     │  │
│  │                    DaemonSet: 1 pod per node                          │  │
│  │                    PriorityClass: core-critical (1000000)             │  │
│  │                                                                       │  │
│  │  ENTRYPOINTS:                                                         │  │
│  │  ┌──────────────────────┐  ┌──────────────────────────────────────┐  │  │
│  │  │ web (:80)            │  │ websecure (:443)                     │  │  │
│  │  │ Redirects all HTTP   │  │ TLS termination                      │  │  │
│  │  │ to HTTPS (301)       │  │ Routes based on Host() match         │  │  │
│  │  └──────────┬───────────┘  └──────────────────┬───────────────────┘  │  │
│  │             │ 301 redirect                     │                      │  │
│  │             └──────────────────────────────────┤                      │  │
│  │                                                │                      │  │
│  │  PROVIDERS:                                                           │  │
│  │  ┌───────────────────────────────────────────────────────────────┐   │  │
│  │  │ KubernetesCRD Provider                                        │   │  │
│  │  │   allowCrossNamespace: true                                   │   │  │
│  │  │   allowExternalNameServices: true                             │   │  │
│  │  │                                                               │   │  │
│  │  │ Watches for IngressRoute CRDs in ALL namespaces               │   │  │
│  │  │ Each IngressRoute defines:                                    │   │  │
│  │  │   - Host match rule (e.g., Host(`api.stage.freezenith.com`))  │   │  │
│  │  │   - Backend service (name + port)                             │   │  │
│  │  │   - TLS secret (from cert-manager)                            │   │  │
│  │  └───────────────────────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                         │                                                   │
│                         │ Route to backend services:                        │
│                         ▼                                                   │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                        ROUTING TABLE                                 │   │
│  │                                                                      │   │
│  │  Host Match                          │ Backend Service     │ Port    │   │
│  │  ────────────────────────────────────┼─────────────────────┼─────── │   │
│  │  stage.freezenith.com               │ zenith-landing      │ 3000   │   │
│  │  app.stage.freezenith.com           │ zenith-web          │ 3000   │   │
│  │  api.stage.freezenith.com           │ apisix-gateway-proxy│ 9080   │   │
│  │                                      │ (ExternalName svc)  │        │   │
│  │  argocd.stage.freezenith.com        │ argocd-server       │ 80     │   │
│  │  auth.stage.freezenith.com          │ keycloak            │ 8080   │   │
│  │  grafana.stage.freezenith.com       │ grafana             │ 3000   │   │
│  │  hubble.stage.freezenith.com        │ hubble-ui           │ 80     │   │
│  │  hub.stage.freezenith.com           │ harbor-core         │ 80     │   │
│  │  *.apps.stage.freezenith.com        │ customer apps       │ 3000   │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. How Traffic Flows Through Traefik

### Frontend Request (Direct)

```
Browser: GET https://stage.freezenith.com
    │
    ▼
┌──────────┐
│Cloudflare│ 1. Resolves DNS: stage.freezenith.com → 77.42.88.149
│          │ 2. TLS handshake with origin (Traefik)
│          │ 3. Forwards HTTPS request
└────┬─────┘
     │ :443
     ▼
┌──────────┐
│ Traefik  │ 1. Entrypoint: websecure (:443)
│          │ 2. TLS termination: uses cert from Secret "landing-tls"
│          │ 3. Match: Host(`stage.freezenith.com`) → IngressRoute
│          │ 4. Backend: zenith-landing service, port 3000
│          │ 5. Load balance: round-robin to pod(s)
└────┬─────┘
     │ :3000 (HTTP, internal)
     ▼
┌──────────────┐
│zenith-landing│ Next.js SSR → returns HTML
│  Pod         │
└──────────────┘
```

### API Request (Via APISIX)

```
Browser: POST https://api.stage.freezenith.com/v1/apps
    │
    ▼
┌──────────┐
│Cloudflare│ Resolves DNS → 77.42.88.149
└────┬─────┘
     │ :443
     ▼
┌──────────┐
│ Traefik  │ 1. TLS termination
│          │ 2. Match: Host(`api.stage.freezenith.com`)
│          │ 3. IngressRoute points to ExternalName service:
│          │    "apisix-gateway-proxy" (type: ExternalName)
│          │    → resolves to: apisix-gateway.apisix.svc.cluster.local
│          │ 4. Forwards HTTP to APISIX gateway
└────┬─────┘
     │ :9080 (HTTP, cross-namespace via ExternalName)
     ▼
┌──────────┐
│  APISIX  │ Handles JWT auth, CORS, rate-limiting
│          │ Then forwards to zenith-api:8080
└──────────┘

Why ExternalName?
  Traefik is in kube-system namespace
  APISIX is in apisix namespace
  ExternalName service bridges the gap:
    Service "apisix-gateway-proxy" in zenith-staging
    → externalName: apisix-gateway.apisix.svc.cluster.local
  This requires: allowExternalNameServices: true (set in traefik.tf)
```

### Customer App Request

```
Browser: GET https://myapp.apps.stage.freezenith.com
    │
    ▼
┌──────────┐
│ Traefik  │ 1. TLS termination (wildcard cert for *.apps.stage)
│          │ 2. Match: Host(`myapp.apps.stage.freezenith.com`)
│          │ 3. IngressRoute in zenith-apps namespace
│          │    (cross-namespace routing enabled)
└────┬─────┘
     │ :3000
     ▼
┌──────────────────────────────────────────────────┐
│ zenith-apps namespace                             │
│                                                   │
│  IF app is running (replicas > 0):                │
│    → Route to customer app pod :3000              │
│                                                   │
│  IF app is scaled to zero (KEDA):                 │
│    → KEDA interceptor catches request             │
│    → cold-start-page middleware shows splash page  │
│    → KEDA scales app from 0 → 1                   │
│    → Splash page auto-refreshes every 5s          │
│    → Once pod is ready, request succeeds           │
└──────────────────────────────────────────────────┘
```

---

## 5. IngressRoute CRDs — The Routing Rules

Traefik uses **IngressRoute** CRDs instead of standard Kubernetes Ingress objects. They're more powerful and don't require annotation hacking.

### Example: ArgoCD IngressRoute (from gitops.tf)

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: argocd
  namespace: argocd
spec:
  entryPoints:
    - websecure          # Only HTTPS (port 443)
  routes:
    - match: Host(`argocd.stage.freezenith.com`)
      kind: Rule
      services:
        - name: argocd-server     # Service in same namespace
          port: 80
  tls:
    secretName: argocd-tls        # Certificate from cert-manager
```

### How IngressRoute → Traefik Routing Works

```
┌─────────────────────────────────────────────────────────────────────┐
│  IngressRoute CRD                                                    │
│  (stored in K8s API server as a custom resource)                     │
│                                                                      │
│  1. Developer creates IngressRoute YAML                              │
│     (usually via Terraform kubernetes_manifest or Helm template)     │
│                                                                      │
│  2. Traefik's KubernetesCRD provider watches for IngressRoute CRDs   │
│     across ALL namespaces                                            │
│                                                                      │
│  3. Traefik reads the IngressRoute and creates a dynamic route:      │
│     Host(`argocd.stage.freezenith.com`) → argocd-server:80           │
│                                                                      │
│  4. When a request arrives matching the Host header:                 │
│     - Traefik loads the TLS cert from the referenced Secret          │
│     - Terminates TLS                                                 │
│     - Forwards the request to the backend service                    │
│                                                                      │
│  5. No Traefik restart needed — routes are dynamic                   │
└─────────────────────────────────────────────────────────────────────┘
```

### All IngressRoutes in the Platform

| IngressRoute | Namespace | Host Match | Backend | TLS Secret |
|-------------|-----------|------------|---------|------------|
| argocd | argocd | `argocd.stage.freezenith.com` | argocd-server:80 | argocd-tls |
| harbor | harbor | `hub.stage.freezenith.com` | harbor-core:80 | harbor-tls |
| hubble-ui | kube-system | `hubble.stage.freezenith.com` | hubble-ui:80 | hubble-tls |
| zenith-api | zenith-staging | `api.stage.freezenith.com` | apisix-gateway-proxy:9080 | api-tls |
| zenith-landing | zenith-staging | `stage.freezenith.com` | zenith-landing:3000 | landing-tls |
| zenith-web | zenith-staging | `app.stage.freezenith.com` | zenith-web:3000 | web-tls |
| customer apps | zenith-apps | `*.apps.stage.freezenith.com` | customer-svc:3000 | apps-wildcard-tls |

---

## 6. TLS & Certificate Management

```
┌─────────────────────────────────────────────────────────────────────┐
│                    TLS CERTIFICATE LIFECYCLE                         │
│                                                                      │
│  Step 1: Create Certificate CRD                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  apiVersion: cert-manager.io/v1                               │   │
│  │  kind: Certificate                                            │   │
│  │  spec:                                                        │   │
│  │    secretName: argocd-tls          ← where cert is stored     │   │
│  │    issuerRef:                                                 │   │
│  │      name: letsencrypt-prod        ← ClusterIssuer            │   │
│  │      kind: ClusterIssuer                                      │   │
│  │    dnsNames:                                                  │   │
│  │      - argocd.stage.freezenith.com ← domain(s) to cover      │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                         │                                            │
│                         ▼                                            │
│  Step 2: cert-manager processes Certificate                          │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  cert-manager controller:                                     │   │
│  │    1. Creates CertificateRequest                              │   │
│  │    2. Creates Order (ACME)                                    │   │
│  │    3. Uses DNS-01 solver:                                     │   │
│  │       → Creates TXT record in Cloudflare via API              │   │
│  │       → Let's Encrypt verifies TXT record                    │   │
│  │       → Let's Encrypt issues certificate                     │   │
│  │    4. Stores cert + key in Secret "argocd-tls"               │   │
│  │    5. Auto-renews 30 days before expiry                      │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                         │                                            │
│                         ▼                                            │
│  Step 3: Traefik uses the certificate                                │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  IngressRoute references:                                     │   │
│  │    tls:                                                       │   │
│  │      secretName: argocd-tls                                   │   │
│  │                                                               │   │
│  │  Traefik:                                                     │   │
│  │    1. Reads cert from Secret "argocd-tls"                     │   │
│  │    2. Serves it for matching Host()                           │   │
│  │    3. Watches for Secret changes (auto-reload on renewal)     │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 7. Cross-Namespace Routing

Traefik is in `kube-system` but needs to route to services in other namespaces. This requires special configuration.

```
┌─────────────────────────────────────────────────────────────────────┐
│              CROSS-NAMESPACE ROUTING                                 │
│                                                                      │
│  Problem:                                                            │
│    Traefik (kube-system) needs to reach:                             │
│      - zenith-api in zenith-staging                                  │
│      - argocd-server in argocd                                       │
│      - harbor-core in harbor                                         │
│      - APISIX gateway in apisix                                      │
│                                                                      │
│  Solution:                                                           │
│    HelmChartConfig in traefik.tf enables:                            │
│                                                                      │
│    providers:                                                        │
│      kubernetesCRD:                                                  │
│        allowCrossNamespace: true       ← Route to any namespace      │
│        allowExternalNameServices: true ← Route via ExternalName      │
│                                                                      │
│  How it works:                                                       │
│                                                                      │
│  ┌─────────────┐  IngressRoute     ┌──────────────┐                │
│  │ kube-system  │  (in argocd ns)  │ argocd        │                │
│  │   Traefik    │ ───────────────▶ │ argocd-server │                │
│  │              │  Cross-namespace  │ :80           │                │
│  └─────────────┘  allowed!          └──────────────┘                │
│                                                                      │
│  For APISIX (different pattern — ExternalName):                      │
│                                                                      │
│  ┌─────────────┐  IngressRoute     ┌──────────────────┐             │
│  │ kube-system  │  (zenith-staging)│ zenith-staging     │             │
│  │   Traefik    │ ───────────────▶ │ apisix-gateway-   │             │
│  │              │                  │ proxy (ExternalNam)│             │
│  └─────────────┘                   └────────┬──────────┘             │
│                                              │ Resolves to:          │
│                                              │ apisix-gateway.       │
│                                              │ apisix.svc.cluster    │
│                                              │ .local                │
│                                              ▼                       │
│                                    ┌──────────────────┐             │
│                                    │ apisix            │             │
│                                    │ apisix-gateway    │             │
│                                    │ :9080             │             │
│                                    └──────────────────┘             │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 8. Configuration Reference

### Terraform File

**File:** `infra/terraform/modules/k8s-platform/traefik.tf`

```hcl
# HelmChartConfig — k3s built-in Traefik customization
resource "kubernetes_manifest" "traefik_config" {
  manifest = {
    apiVersion = "helm.cattle.io/v1"
    kind       = "HelmChartConfig"
    metadata = {
      name      = "traefik"
      namespace = "kube-system"
    }
    spec = {
      valuesContent = yamlencode({
        providers = {
          kubernetesCRD = {
            allowCrossNamespace       = true
            allowExternalNameServices = true
          }
        }
      })
    }
  }
}
```

### Key Settings

| Setting | Value | Why |
|---------|-------|-----|
| `allowCrossNamespace` | `true` | IngressRoutes can reference services in other namespaces |
| `allowExternalNameServices` | `true` | IngressRoutes can route to ExternalName services (APISIX bridge) |
| Entrypoints | `:80` (web), `:443` (websecure) | Standard HTTP/HTTPS ports |
| DaemonSet | 1 pod per node | k3s default, ensures traffic reaches every node |
| PriorityClass | core-critical (1000000) | Never evicted — without Traefik, no traffic enters |

---

## 9. Troubleshooting

### "Connection refused" or "502 Bad Gateway"

```bash
# 1. Check if Traefik is running
kubectl get pods -n kube-system -l app.kubernetes.io/name=traefik

# 2. Check Traefik logs for errors
kubectl logs -n kube-system -l app.kubernetes.io/name=traefik --tail=50

# 3. Check if the IngressRoute exists and is correct
kubectl get ingressroute -A

# 4. Check if the backend service exists and has endpoints
kubectl get svc -n zenith-staging
kubectl get endpoints -n zenith-staging zenith-api
# If endpoints show <none>, the backend pods are not running

# 5. Verify Traefik can see the route
kubectl port-forward -n kube-system svc/traefik 9000:9000
# Open http://localhost:9000/dashboard/ to see all routes
```

### "404 Not Found" (Traefik returns its default 404)

```bash
# The Host header doesn't match any IngressRoute
# Check what routes Traefik knows about:
kubectl get ingressroute -A -o wide

# Verify the domain resolves to the right IP:
dig stage.freezenith.com +short
# Should return the VM IP (e.g., 77.42.88.149)

# Check if Cloudflare is proxying (orange cloud):
# If proxied, Traefik sees Cloudflare's IP, not the client's
# This can affect Host header matching
```

### "Certificate not ready" (TLS handshake error)

```bash
# 1. Check Certificate status
kubectl get certificate -A
kubectl describe certificate <name> -n <namespace>

# 2. Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager --tail=50

# 3. Check Order and Challenge status
kubectl get order -A
kubectl get challenge -A

# 4. Common issue: DNS-01 challenge fails
# Verify Cloudflare API token has DNS edit permissions
# Verify the domain zone is accessible
```

### "ExternalName resolution failed"

```bash
# APISIX routing via ExternalName not working
# 1. Check the ExternalName service exists
kubectl get svc apisix-gateway-proxy -n zenith-staging -o yaml
# Should show: type: ExternalName, externalName: apisix-gateway.apisix.svc.cluster.local

# 2. Check APISIX is running
kubectl get pods -n apisix

# 3. Check Traefik config allows ExternalName
kubectl get helmchartconfig traefik -n kube-system -o yaml
# Should show: allowExternalNameServices: true
```

---

## 10. Upgrade Path

### Upgrading Traefik (via k3s)

Traefik is bundled with k3s. To upgrade:

1. **Upgrade k3s** — Traefik version comes with the k3s version
2. **Or override via HelmChartConfig** — Pin a specific Traefik version:

```hcl
spec = {
  valuesContent = yamlencode({
    image = {
      tag = "v3.2.0"  # Pin specific version
    }
  })
}
```

### Adding a new subdomain

1. Add DNS record in `infra/terraform/staging/main.tf`
2. Create Certificate CRD in the target namespace
3. Create IngressRoute CRD pointing to the service
4. `terraform apply` → DNS + cert + route all created

### Switching from Traefik to another ingress controller

**Not recommended** — would require:
- Disabling Traefik in k3s (`--disable=traefik`)
- Migrating all IngressRoute CRDs to standard Ingress
- Reconfiguring cert-manager integration
- Reconfiguring external-dns sources
