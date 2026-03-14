# 07 — Day-2 Operations

> **Read time:** 30 minutes
> **Prerequisite:** [06 — CI/CD Guide](./06-cicd-guide.md)
> **Next:** [08 — 10/10 Implementation Plan](./08-implementation-plan.md)

---

## Daily Development Workflow

### Starting a Feature

```bash
# 1. Make sure you're on staging branch
git checkout staging
git pull origin staging

# 2. Make your changes (backend, frontend, or infra)

# 3. Verify
cd services/api && GO111MODULE=on go vet ./internal/...   # Backend
cd apps/web && npx next lint --quiet                       # Frontend

# 4. Commit with conventional commit message
git add -A
git commit -m "feat(gateway): add rate-limit per gateway"
# Types: feat, fix, chore, docs, refactor, test, perf

# 5. Push to staging
git push origin staging

# 6. Deploy (optional — ArgoCD can auto-sync, or use make)
make deploy-api  # Build + push + ArgoCD sync
```

### Debugging on Staging

```bash
# Connect to staging
ssh zen-stage

# Check pods
kubectl get pods -n zenith-staging
kubectl get pods -n zenith-apps
kubectl get pods -n zenith-builds

# Check logs
kubectl logs -n zenith-staging deploy/zenith-api --tail=100 -f
kubectl logs -n zenith-staging deploy/zenith-web --tail=50

# Check events
kubectl get events -n zenith-staging --sort-by='.lastTimestamp'

# Describe a failing pod
kubectl describe pod -n zenith-staging zenith-api-xxx

# Exec into a pod
kubectl exec -n zenith-staging -it deploy/zenith-api -- /bin/sh

# Check CNPG primary (ALWAYS check which pod is primary!)
kubectl get pods -n zenith-staging -l cnpg.io/cluster=zenith-postgres -o json | \
  jq '.items[] | {name: .metadata.name, role: .metadata.labels.role}'

# Run SQL on primary
kubectl exec -n zenith-staging zenith-postgres-1 -c postgres -- \
  psql -U zenith -d zenith -c "SELECT COUNT(*) FROM users;"

# Check ArgoCD sync
kubectl get applications -n argocd

# Check APISIX routes
kubectl get apisixroutes -n zenith-staging
```

### Fixing Migration Issues

```bash
# If migration is stuck (dirty state):
kubectl exec -n zenith-staging zenith-postgres-1 -c postgres -- \
  psql -U zenith -d zenith -c "UPDATE schema_migrations SET version = 37, dirty = false;"

# Then restart the API pod to re-run migrations:
kubectl rollout restart deploy/zenith-api -n zenith-staging
```

### Checking APISIX Rate Limits

If you're getting 429 errors:
```bash
# Current rate limits:
# Public routes (/api/v1/auth/*): 30 req/60s per IP
# Protected routes (/api/*): 500 req/60s per IP
# Smoke tests include 429 retry helper with 5s backoff
```

---

## Common Tasks

### Add a New API Endpoint

1. Add entity/struct if needed → `entities/`
2. Add port method if needed → `ports/repositories.go`
3. Implement in postgres + memory adapters
4. Add service method → `services/`
5. Add handler → `handlers/`
6. Register route → `cmd/server/main.go`
7. Verify: `GO111MODULE=on go vet ./internal/...`

### Add a New Frontend Page

1. Create `apps/web/src/app/my-page/page.tsx`
2. Add API methods to `apps/web/src/lib/api.ts`
3. Add demo stubs to `apps/web/src/lib/demo-api.ts`
4. Verify: `cd apps/web && npx next lint --quiet`

### Add a New Helm Value

1. Edit `infra/helm/zenith-api/values.yaml` (default)
2. Edit `infra/helm/zenith-api/values-staging.yaml` (staging override)
3. Use in template: `{{ .Values.myValue }}`
4. Verify: `helm lint infra/helm/zenith-api/`

### Add a New Terraform Resource

1. Edit appropriate `.tf` file in `modules/k8s-platform/`
2. Add variables if needed → `variables.tf`
3. Plan: `cd infra/terraform/staging-k8s && terraform plan`
4. Apply: `terraform apply`

### Add a New Migration

```bash
# Create migration files
lich migration create "add_my_table"
# This creates:
#   037_add_my_table.up.sql
#   037_add_my_table.down.sql

# Write SQL in the .up.sql file
# Write rollback SQL in the .down.sql file
# Migration runs automatically on API startup
```

---

## Monitoring & Alerting

### Key Dashboards

| Dashboard | URL | What It Shows |
|-----------|-----|---------------|
| Grafana | grafana.stage.freezenith.com | Metrics, logs, traces |
| ArgoCD | argocd.stage.freezenith.com | GitOps sync status |
| Hubble | hubble.stage.freezenith.com | Network flow visualization |

### Key Metrics

| Metric | Source | Alert Threshold |
|--------|--------|----------------|
| API error rate | Prometheus | > 5% = critical |
| API P99 latency | Prometheus | > 2000ms = critical |
| Node CPU | Prometheus | > 90% = critical |
| Node memory | Prometheus | > 90% = critical |
| CNPG replication lag | Prometheus | > 30s = critical |
| Certificate expiry | Prometheus | < 3 days = critical |
| Pod CrashLoopBackoff | K8s events | Any = investigate |

---

## Runbooks (Quick Reference)

### Pod CrashLoopBackoff
```bash
kubectl describe pod <pod-name> -n <namespace>
kubectl logs <pod-name> -n <namespace> --previous
# Common causes: bad config, missing secret, DB connection, OOM
```

### Database Issues
```bash
# Check CNPG cluster health
kubectl get clusters.postgresql.cnpg.io -A
# Check backup status
kubectl get scheduledbackups.postgresql.cnpg.io -A
# Connection issues → check secret exists
kubectl get secret -n zenith-staging | grep postgres
```

### Certificate Expiry
```bash
kubectl get certificates -A
kubectl describe certificate <name> -n <namespace>
# If stuck: delete the certificate, cert-manager will re-create
kubectl delete certificate <name> -n <namespace>
```

### Build Failures
```bash
# Check Kaniko job
kubectl get jobs -n zenith-builds
kubectl logs job/<job-name> -n zenith-builds
# Common: Harbor auth failure → check kaniko-registry-auth secret
```

---

**Next → [08 — 10/10 Implementation Plan](./08-implementation-plan.md)**
