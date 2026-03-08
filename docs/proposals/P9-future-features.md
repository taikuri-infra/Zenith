# P9 — Future Feature Design Proposals

## P9-01 / P9-02: Serverless Functions

### API Contract

```
POST /api/v1/apps/:appId/functions     Create function
GET  /api/v1/apps/:appId/functions     List functions
PUT  /api/v1/apps/:appId/functions/:id Update function code/config
DELETE /api/v1/apps/:appId/functions/:id Delete function
POST /api/v1/apps/:appId/functions/:id/invoke  Invoke synchronously
```

### Function Model

```go
type Function struct {
    ID          string
    AppID       string
    Name        string
    Runtime     string            // "nodejs20", "python312", "go122"
    Handler     string            // "index.handler"
    MemoryMB    int               // 128, 256, 512, 1024
    TimeoutSec  int               // max 30
    Trigger     FunctionTrigger   // "http", "cron", "event"
    CronExpr    string            // if trigger=cron
    EventSource string            // if trigger=event (NATS subject)
    EnvVars     map[string]string
    Status      string
}
```

### Implementation: KEDA + Custom Runtime

**Why not Knative**: Knative requires its own networking layer (Kourier/Istio) which conflicts with our Traefik+APISIX stack. Too heavy for our single-cluster setup.

**Proposed approach**: KEDA HTTPScaledObject + lightweight function runtime container.

1. **Function runtime**: A small Go HTTP server that loads user code (Node.js via V8 isolates, Python via subprocess, Go via plugin)
2. **Deployment**: Each function is a Deployment + Service + KEDA HTTPScaledObject (scales 0→N)
3. **Cold start**: Use the existing cold-start splash page pattern from free-tier apps
4. **Code storage**: Upload to S3 bucket, mount as init container

### Tier Limits

| Plan | Functions | Invocations/day |
|------|-----------|-----------------|
| Free | 0 | - |
| Pro | 5 | 10,000 |
| Team | 20 | 100,000 |
| Enterprise | Unlimited | Unlimited |

---

## P9-03: Blue/Green Deployments

### Design

Use APISIX traffic splitting to implement blue/green:

```
1. Deploy "green" version as new Deployment (app-name-green)
2. Run health checks against green
3. Switch APISIX upstream from blue to green (100% traffic)
4. Keep blue running for 10 minutes (rollback window)
5. Delete blue if no rollback
```

### API Extension

```
POST /api/v1/apps/:appId/deploy
  Body: { "strategy": "blue-green", "image": "..." }

POST /api/v1/apps/:appId/rollback
  // Switches APISIX back to previous version
```

### APISIX Implementation

```lua
-- Two upstreams: blue (current) and green (new)
-- Route with traffic-split plugin:
{
  "plugins": {
    "traffic-split": {
      "rules": [{
        "weighted_upstreams": [
          { "upstream_id": "green-upstream", "weight": 100 },
          { "upstream_id": "blue-upstream", "weight": 0 }
        ]
      }]
    }
  }
}
```

---

## P9-04: Canary Deployments

### Design

Gradual traffic shift from current to new version:

```
1. Deploy canary as new Deployment (app-name-canary, 1 replica)
2. Route 5% traffic to canary via APISIX traffic-split
3. Monitor error rate + latency for 5 minutes
4. If healthy: increase to 25% → 50% → 100%
5. If unhealthy: automatic rollback to 0% canary
```

### API Extension

```
POST /api/v1/apps/:appId/deploy
  Body: {
    "strategy": "canary",
    "steps": [5, 25, 50, 100],
    "step_duration_minutes": 5,
    "auto_rollback": true,
    "rollback_threshold": { "error_rate": 5, "p95_latency_ms": 500 }
  }
```

### Automatic Rollback

A background goroutine monitors Prometheus during canary:
- Query: `rate(apisix_http_status{app="<name>",code=~"5.."}[2m])`
- If error rate exceeds threshold → set canary weight to 0 → delete canary deployment

---

## P9-05: Multi-Region Architecture

### Design (Hetzner Locations)

```
Primary: Falkenstein (FSN1) — current
Secondary: Helsinki (HEL1) — read replicas + disaster recovery
Future: Ashburn (ASH) — US presence
```

### Architecture

```
                    ┌─────────────────────────────┐
                    │  Cloudflare (GeoDNS/LB)     │
                    └──────┬──────────┬───────────┘
                           │          │
                   ┌───────▼───┐  ┌───▼────────┐
                   │ FSN1      │  │ HEL1       │
                   │ (Primary) │  │ (Secondary)│
                   │           │  │            │
                   │ k3s + API │  │ k3s + API  │
                   │ CNPG(RW)  │  │ CNPG(RO)   │
                   │ Harbor    │  │ Harbor     │
                   │ Full stack│  │ Read-only  │
                   └───────────┘  └────────────┘
```

### Implementation Plan

1. **Phase 1**: Set up HEL1 k3s cluster with Terraform
2. **Phase 2**: CNPG streaming replication (FSN1→HEL1)
3. **Phase 3**: Deploy read-only API to HEL1
4. **Phase 4**: Cloudflare load balancing (geo-based routing)
5. **Phase 5**: Promote HEL1 to read-write for DR failover

### Data Replication

- **CNPG**: Built-in streaming replication with barman for WAL archiving
- **Harbor**: Replication policies (push from FSN1 to HEL1)
- **S3**: Hetzner Object Storage is region-specific → cross-region copy via CronJob
- **Secrets**: Sealed Secrets in Git (available in both clusters)

---

## P9-06: Service Mesh (mTLS Between Customer Services)

### Design

Use **Cilium** (already deployed as CNI) for mTLS between pods, avoiding the overhead of a full Istio/Linkerd deployment.

### Cilium Service Mesh

```yaml
# CiliumNetworkPolicy for mTLS between customer apps
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: app-to-app-mtls
  namespace: zenith-apps
spec:
  endpointSelector:
    matchLabels:
      zenith.dev/user: "<user-id>"
  ingress:
    - fromEndpoints:
        - matchLabels:
            zenith.dev/user: "<user-id>"
      authentication:
        mode: "required"
```

### Hubble Integration

- Already have Hubble UI deployed
- Shows service-to-service traffic flows
- Expose in Zenith dashboard: **Monitoring → Service Map**

### Tier Gating

- **Team+**: Service mesh enabled by default
- **Pro**: Opt-in via `zenith apps update --mesh=true`
- **Free**: Not available

---

## P9-07: GitOps Mode

### Design

Customers push to their Git repo → Zenith automatically deploys.

### How It Works

```
1. Customer enables GitOps: zenith apps update my-app --gitops=true
2. Zenith creates an ArgoCD Application CR pointing to customer's repo
3. Customer pushes code → GitHub webhook → ArgoCD detects change → builds → deploys
4. Build status visible in Zenith dashboard
```

### Alternative: Webhook-Only (Simpler)

Instead of full ArgoCD integration, use the existing webhook flow:

```
GitHub push → Zenith webhook endpoint → Kaniko build → deploy
```

This already works. "GitOps mode" just means:
1. Auto-deploy on push to configured branch
2. Show commit SHA in deployment history
3. Allow rollback to previous commit

### API Extension

```
PUT /api/v1/apps/:appId
  Body: { "auto_deploy": true, "deploy_branch": "main" }
```

### Webhook Registration

When `auto_deploy` is enabled:
1. Call GitHub API to create/update webhook on the customer's repo
2. Webhook points to `https://api.freezenith.com/api/v1/webhooks/github`
3. On push event matching `deploy_branch` → trigger build + deploy

---

## P9-08: Implementation Priority

| Feature | Impact | Effort | Priority |
|---------|--------|--------|----------|
| GitOps mode (P9-07) | High | Low | 1 — Already mostly built |
| Blue/Green (P9-03) | Medium | Medium | 2 — APISIX traffic-split |
| Canary (P9-04) | Medium | Medium | 3 — Builds on blue/green |
| Serverless (P9-01/02) | High | High | 4 — Major new feature |
| Multi-region (P9-05) | High | Very High | 5 — Infra investment |
| Service mesh (P9-06) | Low | Medium | 6 — Niche demand |
