# 22 — Day-to-Day Operations Guide

> **Purpose:** Step-by-step instructions for the tasks a developer does 90% of the time — adding features, deploying, debugging, and maintaining the platform.
> **Audience:** Any developer who has completed the local setup and needs to start contributing.
> **Last Updated:** 2026-03-03
> **Related:** [21-local-development-setup.md](./21-local-development-setup.md) (initial setup), [15-argocd-gitops.md](./15-argocd-gitops.md) (deployment flow), [10-backend-architecture.md](./10-backend-architecture.md) (Go code structure)

---

## Table of Contents

1. [Adding a New API Endpoint](#1-adding-a-new-api-endpoint)
2. [Adding a New Frontend Page](#2-adding-a-new-frontend-page)
3. [Running Database Migrations](#3-running-database-migrations)
4. [Deploying to Staging](#4-deploying-to-staging)
5. [Deploying to Production](#5-deploying-to-production)
6. [Checking Logs](#6-checking-logs)
7. [Debugging a Customer Issue](#7-debugging-a-customer-issue)
8. [Adding a New Helm Value](#8-adding-a-new-helm-value)
9. [Adding a New Environment Variable](#9-adding-a-new-environment-variable)
10. [Working with Sealed Secrets](#10-working-with-sealed-secrets)
11. [Creating a New Grafana Dashboard](#11-creating-a-new-grafana-dashboard)
12. [Git Workflow & Branch Strategy](#12-git-workflow--branch-strategy)

---

## 1. Adding a New API Endpoint

Follow the Lich Architecture layers. Never skip a layer.

```
┌─────────────────────────────────────────────────────────────────────────┐
│             ADDING A NEW API ENDPOINT (step by step)                     │
│                                                                          │
│  Example: Add GET /api/v1/usage — returns platform usage stats          │
│                                                                          │
│  Step 1: ENTITY (if needed)          services/api/internal/entities/     │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Only if you need a new domain model.                              │ │
│  │  Entities have ZERO external imports — only standard library.      │ │
│  │                                                                    │ │
│  │  File: entities/usage.go                                           │ │
│  │  type UsageStats struct {                                          │ │
│  │      TotalApps     int                                             │ │
│  │      TotalDBs      int                                             │ │
│  │      StorageUsedMB int64                                           │ │
│  │  }                                                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                              │                                           │
│                              ▼                                           │
│  Step 2: PORT (interface)            services/api/internal/ports/        │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Define what the service needs — NOT how it's implemented.         │ │
│  │  Ports import entities only.                                       │ │
│  │                                                                    │ │
│  │  File: ports/repositories.go (or ports/infrastructure.go)          │ │
│  │  type UsageRepository interface {                                  │ │
│  │      GetUsageStats(ctx context.Context, customerID string)         │ │
│  │          (*entities.UsageStats, error)                             │ │
│  │  }                                                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                              │                                           │
│                              ▼                                           │
│  Step 3: DTO (request/response)      services/api/internal/dto/         │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Shape of the JSON response the API returns.                       │ │
│  │  DTOs import entities only.                                        │ │
│  │                                                                    │ │
│  │  File: dto/responses.go                                            │ │
│  │  type UsageResponse struct {                                       │ │
│  │      TotalApps     int   `json:"totalApps"`                       │ │
│  │      TotalDBs      int   `json:"totalDatabases"`                  │ │
│  │      StorageUsedMB int64 `json:"storageUsedMB"`                   │ │
│  │  }                                                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                              │                                           │
│                              ▼                                           │
│  Step 4: SERVICE (business logic)    services/api/internal/services/    │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Business logic. Imports entities, ports, dto. NEVER adapters.     │ │
│  │                                                                    │ │
│  │  File: services/usage.go  (or add method to existing service)      │ │
│  │  func (s *UsageService) GetUsageStats(ctx, customerID)             │ │
│  │      (*dto.UsageResponse, error) {                                │ │
│  │      stats, err := s.usageRepo.GetUsageStats(ctx, customerID)     │ │
│  │      return &dto.UsageResponse{...}, nil                          │ │
│  │  }                                                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                              │                                           │
│                              ▼                                           │
│  Step 5: ADAPTER (implementation)    services/api/internal/adapters/    │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Implement the port interface with real PostgreSQL queries.         │ │
│  │  Also create an in-memory version for tests.                       │ │
│  │                                                                    │ │
│  │  File: adapters/postgres/usage_repo.go                             │ │
│  │  func (r *UsageRepo) GetUsageStats(ctx, customerID)               │ │
│  │      (*entities.UsageStats, error) {                              │ │
│  │      row := r.pool.QueryRow(ctx, `SELECT count(*)...`)           │ │
│  │      ...                                                           │ │
│  │  }                                                                 │ │
│  │                                                                    │ │
│  │  File: adapters/memory/usage_repo.go                               │ │
│  │  (in-memory implementation for unit tests)                         │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                              │                                           │
│                              ▼                                           │
│  Step 6: HANDLER (HTTP layer)        services/api/internal/handlers/    │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Wire the HTTP request to the service method.                      │ │
│  │                                                                    │ │
│  │  File: handlers/usage.go                                           │ │
│  │  func (h *UsageHandler) GetUsageStats(c *fiber.Ctx) error {       │ │
│  │      customerID := c.Locals("customerID").(string)                │ │
│  │      stats, err := h.usageService.GetUsageStats(ctx, customerID)  │ │
│  │      return c.JSON(stats)                                          │ │
│  │  }                                                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                              │                                           │
│                              ▼                                           │
│  Step 7: ROUTE REGISTRATION          services/api/cmd/server/main.go   │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Register the route in main.go (the DI composition root).          │ │
│  │                                                                    │ │
│  │  // In the protected routes group:                                 │ │
│  │  api.Get("/usage", usageHandler.GetUsageStats)                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                              │                                           │
│                              ▼                                           │
│  Step 8: TEST                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  cd services/api && go test ./... -race -count=1                  │ │
│  │  curl http://localhost:8080/api/v1/usage                          │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

**Key rule:** Services NEVER import adapters. If you find yourself importing `adapters/postgres` from a service file, you're violating Lich Architecture. Define an interface in `ports/` instead.

---

## 2. Adding a New Frontend Page

```
┌─────────────────────────────────────────────────────────────────────────┐
│             ADDING A NEW PAGE (Next.js App Router)                       │
│                                                                          │
│  Example: Add /analytics page to the web dashboard                      │
│                                                                          │
│  Step 1: Create the page file                                           │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  File: apps/web/src/app/analytics/page.tsx                         │ │
│  │                                                                    │ │
│  │  "use client";                                                     │ │
│  │  import { useApi } from "@/hooks/use-api";                        │ │
│  │  import { Shell } from "@/components/shell";                      │ │
│  │  import { StatCard } from "@/components/stat-card";               │ │
│  │                                                                    │ │
│  │  export default function AnalyticsPage() {                        │ │
│  │    const { data, loading, error } = useApi(                       │ │
│  │      () => api.getAnalytics(), []                                 │ │
│  │    );                                                              │ │
│  │    if (loading) return <Shell><Skeleton /></Shell>;                │ │
│  │    return <Shell>...</Shell>;                                      │ │
│  │  }                                                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  Step 2: Add API function (if new endpoint)                             │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  File: apps/web/src/lib/api.ts                                     │ │
│  │                                                                    │ │
│  │  getAnalytics: async () => {                                      │ │
│  │    const res = await fetchWithAuth("/api/v1/analytics");          │ │
│  │    return res.json();                                              │ │
│  │  },                                                                │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  Step 3: Add mock data (for demo mode)                                  │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  File: apps/web/src/lib/mock-data.ts                               │ │
│  │  File: apps/web/src/lib/demo-api.ts                                │ │
│  │                                                                    │ │
│  │  Add matching mock function that returns test data.                │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  Step 4: Add to sidebar navigation                                      │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  File: apps/web/src/components/sidebar.tsx                         │ │
│  │                                                                    │ │
│  │  Add { name: "Analytics", href: "/analytics", icon: BarChart }    │ │
│  │  Optionally add mode: "saas" to hide in standalone mode.          │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Running Database Migrations

```
┌─────────────────────────────────────────────────────────────────────────┐
│             DATABASE MIGRATIONS                                          │
│                                                                          │
│  Migrations run automatically on API startup.                           │
│  The API reads SQL files from:                                          │
│  services/api/internal/adapters/postgres/migrations/                    │
│                                                                          │
│  To add a new migration:                                                │
│                                                                          │
│  1. Create a new SQL file with a sequential number:                     │
│     migrations/005_add_analytics_table.up.sql                           │
│     migrations/005_add_analytics_table.down.sql                         │
│                                                                          │
│  2. Write the SQL:                                                      │
│     -- 005_add_analytics_table.up.sql                                   │
│     CREATE TABLE analytics (                                             │
│         id UUID PRIMARY KEY DEFAULT gen_random_uuid(),                   │
│         customer_id UUID NOT NULL REFERENCES customers(id),             │
│         event_type TEXT NOT NULL,                                        │
│         created_at TIMESTAMP DEFAULT NOW()                              │
│     );                                                                   │
│                                                                          │
│     -- 005_add_analytics_table.down.sql                                 │
│     DROP TABLE IF EXISTS analytics;                                     │
│                                                                          │
│  3. Restart the API — migration runs automatically.                     │
│                                                                          │
│  4. On staging/production:                                               │
│     The API pod restarts on deploy → migration runs.                    │
│     ArgoCD + Image Updater handles this automatically.                  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Deploying to Staging

```
┌─────────────────────────────────────────────────────────────────────────┐
│             DEPLOYMENT TO STAGING                                        │
│                                                                          │
│  There are two paths — automated (normal) and manual (emergency).       │
│                                                                          │
│  ═══════════════════════════════════════════════════════════════════     │
│  PATH 1: AUTOMATED (normal development)                                  │
│  ═══════════════════════════════════════════════════════════════════     │
│                                                                          │
│  Step 1: Push to main                                                   │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  git add -A && git commit -m "feat: add analytics endpoint"     │   │
│  │  git push origin main                                            │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                          │                                               │
│                          ▼                                               │
│  Step 2: GitHub Actions runs (automatic)                                │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  tests → security scan → build images → push to Harbor           │   │
│  │  (takes ~5-8 minutes)                                             │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                          │                                               │
│                          ▼                                               │
│  Step 3: Merge main → staging                                           │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  git checkout staging                                             │   │
│  │  git merge main                                                   │   │
│  │  git push origin staging                                          │   │
│  │  git checkout main                                                │   │
│  │                                                                   │   │
│  │  IMPORTANT: ArgoCD watches the `staging` branch, NOT `main`.     │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                          │                                               │
│                          ▼                                               │
│  Step 4: ArgoCD syncs (automatic, ~2 minutes)                           │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  ArgoCD Image Updater detects new image tag in Harbor            │   │
│  │  → Updates Helm values → ArgoCD syncs → Pods restart            │   │
│  │                                                                   │   │
│  │  Watch it: https://argocd.stage.freezenith.com                   │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                          │                                               │
│                          ▼                                               │
│  Step 5: Verify                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  curl https://api.stage.freezenith.com/api/v1/health             │   │
│  │  open https://app.stage.freezenith.com                           │   │
│  │  kubectl -n zenith-staging get pods                              │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  ═══════════════════════════════════════════════════════════════════     │
│  PATH 2: MANUAL (emergency / quick fix)                                  │
│  ═══════════════════════════════════════════════════════════════════     │
│                                                                          │
│  # Build and push manually via Makefile                                 │
│  make build-api push-api                                                 │
│                                                                          │
│  # Or restart pods to pick up new image (if tag is `latest`)            │
│  kubectl -n zenith-staging rollout restart deployment zenith-api         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Deploying to Production

```
Same as staging but with extra care:

1. Verify everything works on staging first
2. Create a Git tag for the release
3. Update infra/helm/zenith/Chart.yaml appVersion
4. Merge main → production branch (when it exists)
5. ArgoCD production instance syncs

Production deployment is NOT yet configured — staging is the active
environment. When production is set up, it will follow the same
ArgoCD pattern with a separate `production` branch and values files.
```

---

## 6. Checking Logs

```
┌─────────────────────────────────────────────────────────────────────────┐
│             WHERE TO FIND LOGS                                           │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  METHOD 1: kubectl (quick, real-time)                           │    │
│  │                                                                  │    │
│  │  # API logs (last 100 lines, follow)                            │    │
│  │  kubectl -n zenith-staging logs deploy/zenith-api --tail=100 -f │    │
│  │                                                                  │    │
│  │  # Specific pod                                                  │    │
│  │  kubectl -n zenith-staging logs zenith-api-abc123 -f            │    │
│  │                                                                  │    │
│  │  # Previous crashed container                                    │    │
│  │  kubectl -n zenith-staging logs zenith-api-abc123 --previous    │    │
│  │                                                                  │    │
│  │  # All pods with a label                                         │    │
│  │  kubectl -n zenith-staging logs -l app=zenith-api --tail=50     │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  METHOD 2: Grafana / Loki (historical, searchable)              │    │
│  │                                                                  │    │
│  │  1. Open https://grafana.stage.freezenith.com                   │    │
│  │  2. Go to Explore → Select "Loki" datasource                   │    │
│  │  3. Query examples:                                              │    │
│  │                                                                  │    │
│  │  # All API logs                                                  │    │
│  │  {namespace="zenith-staging", app="zenith-api"}                 │    │
│  │                                                                  │    │
│  │  # Errors only                                                   │    │
│  │  {namespace="zenith-staging"} |= "error"                       │    │
│  │                                                                  │    │
│  │  # Specific customer                                             │    │
│  │  {namespace="zenith-staging"} |= "customer-abc"                 │    │
│  │                                                                  │    │
│  │  # HTTP 500 errors                                               │    │
│  │  {namespace="zenith-staging"} | json | status >= 500            │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  METHOD 3: ArgoCD (deployment logs)                              │    │
│  │                                                                  │    │
│  │  1. Open https://argocd.stage.freezenith.com                    │    │
│  │  2. Click the application (e.g., zenith-api)                    │    │
│  │  3. Click on a pod → "Logs" tab                                 │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Debugging a Customer Issue

```
┌─────────────────────────────────────────────────────────────────────────┐
│             DEBUGGING WORKFLOW                                           │
│                                                                          │
│  Customer reports: "My app isn't working"                               │
│                                                                          │
│  Step 1: Identify the customer's namespace                              │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  kubectl get ns | grep zenith-                                     │ │
│  │  # zenith-customer-abc    Active   5d                              │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Step 2: Check pod status                                               │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  kubectl -n zenith-customer-abc get pods                           │ │
│  │  # Is the pod Running? CrashLoopBackOff? OOMKilled?               │ │
│  │                                                                    │ │
│  │  kubectl -n zenith-customer-abc describe pod <pod-name>           │ │
│  │  # Check Events section for errors                                │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Step 3: Check application logs                                         │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  kubectl -n zenith-customer-abc logs deploy/<app> --tail=100      │ │
│  │  # Look for error messages, stack traces, connection refused      │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Step 4: Check database connectivity                                    │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  # Find which CNPG shard the customer is on                       │ │
│  │  kubectl get secret -n zenith-customer-abc db-credentials -o yaml │ │
│  │                                                                    │ │
│  │  # Test database connection                                       │ │
│  │  kubectl -n zenith-shared exec -it free-pg-1 -- psql -U postgres │ │
│  │  \l  -- list databases, find customer's DB                        │ │
│  │  \c customer_abc  -- connect to it                                │ │
│  │  \dt  -- list tables                                              │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Step 5: Check networking (Hubble)                                      │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  # Is traffic reaching the pod?                                   │ │
│  │  hubble observe --namespace zenith-customer-abc --last 100        │ │
│  │                                                                    │ │
│  │  # Check CiliumNetworkPolicy isn't blocking                      │ │
│  │  kubectl -n zenith-customer-abc get cnp                           │ │
│  │                                                                    │ │
│  │  # Or use Hubble UI                                               │ │
│  │  open https://hubble.stage.freezenith.com                         │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Step 6: Check APISIX routing                                           │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  # Test the route directly                                        │ │
│  │  curl -v https://api.customer-abc.freezenith.com/api/v1/health   │ │
│  │                                                                    │ │
│  │  # Check APISIX route exists                                     │ │
│  │  kubectl -n apisix get apisixroute                                │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Adding a New Helm Value

```bash
# 1. Add the value to the chart's values.yaml
#    File: infra/helm/zenith-api/values.yaml
#    env:
#      MY_NEW_VAR: "default-value"

# 2. Reference it in the deployment template
#    File: infra/helm/zenith-api/templates/deployment.yaml
#    - name: MY_NEW_VAR
#      value: {{ .Values.env.MY_NEW_VAR | quote }}

# 3. Override for staging
#    File: infra/helm/zenith-api/values-staging.yaml
#    env:
#      MY_NEW_VAR: "staging-value"

# 4. Override for production
#    File: infra/helm/zenith-api/values-production.yaml
#    env:
#      MY_NEW_VAR: "production-value"

# 5. Test locally
helm template infra/helm/zenith-api/ -f infra/helm/zenith-api/values-staging.yaml

# 6. Commit, push, merge to staging → ArgoCD syncs
```

---

## 9. Adding a New Environment Variable

```
┌─────────────────────────────────────────────────────────────────────────┐
│             ADDING AN ENV VAR (end to end)                               │
│                                                                          │
│  Files to update:                                                       │
│                                                                          │
│  1. services/api/internal/config/config.go                              │
│     → Add field to Config struct + Load() function                      │
│                                                                          │
│  2. infra/helm/zenith-api/values.yaml                                   │
│     → Add default value under env:                                      │
│                                                                          │
│  3. infra/helm/zenith-api/values-staging.yaml                           │
│     → Add staging-specific value                                        │
│                                                                          │
│  4. infra/helm/zenith-api/templates/deployment.yaml                     │
│     → Add env var to container spec                                     │
│                                                                          │
│  5. docker-compose.yml (if needed locally)                               │
│     → Add to api service environment                                    │
│                                                                          │
│  6. .env.example                                                         │
│     → Document the variable                                             │
│                                                                          │
│  IF IT'S A SECRET:                                                       │
│  7. Create SealedSecret (see section 10)                                │
│  8. Reference via secretKeyRef instead of value in deployment.yaml      │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Working with Sealed Secrets

```bash
# Create a secret (DON'T commit the plain YAML!)
kubectl create secret generic my-secret \
  --from-literal=API_KEY=sk_live_abc123 \
  --namespace=zenith-staging \
  --dry-run=client -o yaml > /tmp/secret.yaml

# Seal it with the cluster's public key
kubeseal \
  --controller-name=sealed-secrets \
  --controller-namespace=sealed-secrets \
  --format=yaml \
  < /tmp/secret.yaml > infra/helm/zenith-platform/templates/my-sealed-secret.yaml

# Delete the plain secret immediately
rm /tmp/secret.yaml

# Commit the sealed version (safe — it's encrypted)
git add infra/helm/zenith-platform/templates/my-sealed-secret.yaml
git commit -m "Add sealed secret for external API key"
```

See [20-sealed-secrets.md](./20-sealed-secrets.md) for full details.

---

## 11. Creating a New Grafana Dashboard

```bash
# 1. Create the dashboard in Grafana UI
#    Open https://grafana.stage.freezenith.com
#    Create dashboard → Add panels → Save

# 2. Export as JSON
#    Dashboard settings → JSON Model → Copy

# 3. Save to the monitoring chart
#    File: infra/helm/monitoring/dashboards/my-dashboard.json
#    Paste the JSON

# 4. The ConfigMap template auto-discovers dashboards in that directory
#    (labeled with grafana_dashboard: "1")

# 5. Commit and push → ArgoCD syncs → Grafana auto-loads the dashboard
```

---

## 12. Git Workflow & Branch Strategy

```
┌─────────────────────────────────────────────────────────────────────────┐
│             GIT WORKFLOW                                                 │
│                                                                          │
│  main ──────────────────────────────────────────────────────────────    │
│    │         │         │         │                                       │
│    │    feature/xyz    │    fix/abc                                      │
│    │         │         │         │                                       │
│    │    ┌────┘    ┌────┘    ┌────┘                                      │
│    │    ▼         ▼         ▼                                            │
│    ├── merge ──── merge ─── merge ──────────────────────────────────    │
│    │                                                                     │
│    │    CI runs: test → security scan → build images → push to Harbor   │
│    │                                                                     │
│    └── merge to staging branch ─────────────────────────────────────    │
│                                    │                                     │
│                                    ▼                                     │
│  staging ──────────────── ArgoCD watches this branch ───────────────    │
│                                    │                                     │
│                               ArgoCD syncs                               │
│                                    │                                     │
│                                    ▼                                     │
│                           zen-stage cluster                              │
│                                                                          │
│  RULES:                                                                  │
│  1. Develop on feature branches: feature/add-analytics                  │
│  2. PR to main, get review, merge                                       │
│  3. CI builds and pushes images automatically                           │
│  4. To deploy: merge main → staging, push both                         │
│  5. ArgoCD auto-syncs staging cluster                                   │
│  6. NEVER push directly to staging branch                               │
│  7. ALWAYS merge main → staging (not the other way around)              │
└─────────────────────────────────────────────────────────────────────────┘
```
