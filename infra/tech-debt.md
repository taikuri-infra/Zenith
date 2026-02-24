# Zenith Infrastructure — Tech Debt

> Last updated: 2026-02-24

---

## Critical

### 1. PostgreSQL: Raw StatefulSet instead of CloudNativePG Operator

**File:** `infra/helm/zenith/templates/postgres.yaml`

**Current state:** Plain `StatefulSet` running `postgres:16-alpine` container. No backups, no HA, no failover, no point-in-time recovery.

**What's missing:**
- Automated backups (WAL archiving, scheduled base backups)
- High availability / automatic failover
- Point-in-time recovery (PITR)
- Connection pooling (PgBouncer)
- Automated minor version upgrades
- Monitoring integration (built-in Prometheus metrics)

**Fix:** Install [CloudNativePG](https://cloudnative-pg.io/) operator via Terraform `helm_release`, replace `postgres.yaml` with a `postgresql.cnpg.io/v1 Cluster` CR.

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: zenith-postgres
spec:
  instances: 1        # staging (3 for prod)
  storage:
    size: 2Gi
  bootstrap:
    initdb:
      database: zenith
      owner: zenith
  backup:
    barmanObjectStore:
      destinationPath: s3://zenith-backups/
```

**Priority:** HIGH — data loss risk in current setup

---

### 2. Secrets: Plain K8s Secrets from Helm Values

**Files:** `infra/helm/zenith/templates/secrets.yaml`, `infra/helm/zenith/templates/registry-secret.yaml`

**Current state:** Secrets (JWT, DB password, admin credentials, registry auth) are injected via `helm set_sensitive` from Terraform `tfvars`. Anyone with cluster access can read them via `helm get values zenith` or `kubectl get secret -o yaml`.

**What's missing:**
- Secret rotation mechanism
- Audit trail (who created/accessed secrets)
- Encryption at rest (beyond etcd default)
- Separation of secret management from deployment pipeline

**Fix:** Install [External Secrets Operator](https://external-secrets.io/) and integrate with a secret backend:
- **Staging:** HashiCorp Vault (self-hosted) or simple SecretStore
- **Production:** HashiCorp Vault or AWS Secrets Manager

**Priority:** HIGH — security risk, but acceptable for initial staging

---

## Medium

### 3. Kong Manager UI Not Enabled

**Current state:** Kong Gateway is deployed in DB-less mode but the built-in admin UI (Kong Manager OSS, free since Kong 3.4+) is not enabled. No visibility into routes, plugins, or traffic.

**Fix:** Enable in Terraform `helm_release.kong`:
```hcl
set {
  name  = "gateway.admin.http.enabled"
  value = "true"
}
set {
  name  = "gateway.manager.enabled"
  value = "true"
}
```
Then expose via Traefik IngressRoute (internal only, behind auth).

**Reference:** https://github.com/Kong/kong-manager

**Priority:** MEDIUM — operational visibility

---

### 4. Docker Images Built on Mac (ARM) — No CI Cross-Compilation Guard

**Current state:** Images built locally via `docker buildx --platform linux/amd64` work, but there's no guardrail preventing someone from building without `--platform` flag and pushing arm64 images to Harbor. This caused CrashLoopBackOff on the amd64 staging server.

**Fix:**
- CI workflow updated with `platforms: linux/amd64` (done)
- Add a Makefile target that always includes `--platform linux/amd64`
- Consider adding a Harbor webhook or admission controller that rejects non-amd64 images

**Priority:** MEDIUM — causes deployment failures

---

### 5. Traefik Middleware Duplication

**Files:** `infra/helm/zenith/templates/ingress.yaml`, `infra/helm/zenith/templates/tenants.yaml`

**Current state:** `redirect-to-https` Middleware is defined separately in every namespace (platform + each tenant). Identical config duplicated.

**Fix:** Either use a single Middleware in a shared namespace with cross-namespace reference, or extract to a dedicated `middleware.yaml` template applied once.

**Priority:** LOW — works fine, just messy

---

### 6. No NetworkPolicy / Pod Security

**Current state:** No `NetworkPolicy` resources. Any pod in any namespace can talk to any other pod. Tenant namespaces are not isolated from each other or from platform namespace.

**Fix:** Add NetworkPolicy per namespace:
- Tenant pods can only reach their own services + platform API
- Platform namespace restricts ingress to Traefik only
- Monitoring namespace can scrape all namespaces (egress)

**Priority:** MEDIUM — multi-tenant security requirement

---

### 7. No PodDisruptionBudget / HorizontalPodAutoscaler

**Current state:** Stateless services (api, landing, mc, web) have no PDB or HPA. Single replica with no disruption budget.

**Fix:**
- Add `PodDisruptionBudget` for production (minAvailable: 1)
- Add `HorizontalPodAutoscaler` or integrate with KEDA `ScaledObject` for auto-scaling
- KEDA is already installed but not wired to any workloads

**Priority:** LOW for staging, HIGH for production

---

### 8. zenith-web Next.js Code Bug

**Error:** `You cannot use different slug names for the same dynamic path ('name' !== 'id')`

**Where:** `apps/web/` — two dynamic route files use conflicting parameter names for the same path segment.

**Impact:** zenith-web pod returns HTTP 500, readiness probe fails, pod stays 0/1 Ready.

**Priority:** HIGH — tenant web app completely broken

---

## Low / Future

### 9. No GitOps (FluxCD / ArgoCD)

**Current state:** Deployments are triggered manually via `terraform apply`. No automated sync from git to cluster.

**Fix:** Consider FluxCD or ArgoCD for automated deployments when CI/CD pipeline matures.

---

### 10. Monitoring Stack Not Exposed

**Current state:** Grafana, Prometheus, and Alertmanager are running but have no IngressRoute. Only accessible via `kubectl port-forward`.

**Fix:** Add Traefik IngressRoutes for:
- `grafana.stage.freezenith.com` (with auth)
- `prometheus.stage.freezenith.com` (internal only)

---

### 11. No RBAC for Application Service Accounts

**Current state:** All pods run with the `default` ServiceAccount. No least-privilege RBAC.

**Fix:** Create dedicated ServiceAccounts with minimal RBAC for each component (especially zenith-api which may need K8s API access).

---

## Resolved

| Item | Date | Resolution |
|------|------|------------|
| Namespace double-creation conflict | 2026-02-24 | Emptied `namespace.yaml`, Terraform `create_namespace = true` handles it |
| Loki CrashLoopBackOff (missing compactor config) | 2026-02-24 | Added `delete_request_store: filesystem` |
| Loki cache pods Pending (insufficient resources) | 2026-02-24 | Disabled `chunksCache` and `resultsCache` for SingleBinary mode |
| ARM64 images on AMD64 server | 2026-02-24 | Rebuilt all images with `--platform linux/amd64`, cleared node image cache |
| KEDA HTTP add-on repo 404 | 2026-02-24 | Fixed repo URL to `https://kedacore.github.io/charts` |
| cert-manager Helm name conflict | 2026-02-24 | Imported existing release into Terraform state |
