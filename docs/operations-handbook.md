# Zenith — Internal Operations Handbook

## Architecture Overview

```
User → Cloudflare (DNS + WAF) → Traefik (Ingress) → APISIX (Gateway) → Zenith API
                                                                        ↓
                                        ┌───────────────────────────────────────────┐
                                        │ Platform Services                          │
                                        │ ┌─────────┐ ┌──────────┐ ┌─────────────┐ │
                                        │ │ CNPG PG  │ │ Keycloak │ │ Harbor      │ │
                                        │ │ (shared) │ │ (auth)   │ │ (registry)  │ │
                                        │ └─────────┘ └──────────┘ └─────────────┘ │
                                        │ ┌─────────┐ ┌──────────┐ ┌─────────────┐ │
                                        │ │ Temporal │ │ NATS     │ │ KEDA        │ │
                                        │ │ (wflow)  │ │ (events) │ │ (autoscale) │ │
                                        │ └─────────┘ └──────────┘ └─────────────┘ │
                                        └───────────────────────────────────────────┘
```

## Key Namespaces

| Namespace | Purpose | Critical? |
|-----------|---------|-----------|
| `zenith-staging` / `zenith-production` | Platform services (API, web, admin) | Yes |
| `zenith-apps` | Customer applications | Yes |
| `zenith-builds` | Kaniko build jobs | No |
| `monitoring` | Prometheus, Grafana, Loki, Tempo, OTel | Yes |
| `keycloak` | Keycloak identity provider | Yes |
| `apisix` | APISIX API Gateway | Yes |
| `argocd` | ArgoCD GitOps | Yes |
| `cert-manager` | TLS certificate automation | Yes |
| `cnpg-system` | CloudNativePG operator | Yes |
| `keda` | KEDA autoscaler | No |
| `kyverno` | Admission controller | No |
| `nats` | NATS JetStream event bus | No |

## Daily Operations

### Morning Checklist

1. Check Grafana dashboards:
   - Platform health: `grafana.<domain>/d/zenith-platform`
   - Business metrics: `grafana.<domain>/d/zenith-business`
2. Check Telegram for overnight alerts
3. Review ArgoCD sync status: `argocd.<domain>`
4. Check cert-manager certificate expiry: `kubectl get certificates -A`

### Common Tasks

#### Deploy API Update

```bash
# 1. Build and push image
cd services/api
docker build -t registry.stage.freezenith.com/zenith-stage/zenith-api:v0.X.Y -f Dockerfile ../..
docker push registry.stage.freezenith.com/zenith-stage/zenith-api:v0.X.Y

# 2. Update Helm values
# Edit helm/zenith-api/values-staging.yaml → image.tag

# 3. Merge to staging branch (ArgoCD auto-syncs)
git checkout staging && git merge main && git push origin staging
```

#### Check Customer App Status

```bash
kubectl -n zenith-apps get pods -l zenith.dev/app=<app-name>
kubectl -n zenith-apps logs -l zenith.dev/app=<app-name> --tail=50
kubectl -n zenith-apps describe pod <pod-name>
```

#### Force Rebuild Customer App

```bash
# Via API
curl -X POST https://api.<domain>/api/v1/apps/<app-id>/deploy \
  -H "Authorization: Bearer <admin-token>"
```

#### Database Operations

```bash
# Check CNPG clusters
kubectl -n zenith-staging get clusters
kubectl -n zenith-staging get pods -l cnpg.io/cluster=<cluster-name>

# Backup status
kubectl -n zenith-staging get backups

# Connect to database
kubectl -n zenith-staging port-forward svc/<cluster>-rw 5432:5432
psql -h localhost -U <user> -d <dbname>
```

#### Certificate Issues

```bash
# Check certificates
kubectl get certificates -A
kubectl get certificaterequests -A
kubectl get challenges -A

# Force renewal
kubectl delete certificate <name> -n <namespace>
# cert-manager will auto-recreate
```

## Incident Response

### Severity Levels

| Level | Description | Response Time | Example |
|-------|-------------|---------------|---------|
| P0 (Critical) | Platform down, all users affected | 15 min | API unreachable, DB cluster failure |
| P1 (High) | Major feature broken, many users affected | 1 hour | Builds failing, auth broken |
| P2 (Medium) | Minor feature broken, some users affected | 4 hours | Single app crash, slow queries |
| P3 (Low) | Cosmetic, no impact | Next business day | UI bug, docs error |

### Response Playbook

1. **Acknowledge**: Respond in Telegram/PagerDuty within SLA
2. **Assess**: Check Grafana, Loki logs, `kubectl` for affected components
3. **Communicate**: Update status page if P0/P1
4. **Fix**: Apply the relevant runbook from `docs/runbooks/`
5. **Verify**: Confirm fix with metrics/logs
6. **Post-mortem**: Write RCA for P0/P1 incidents

### Quick Diagnostic Commands

```bash
# Overall cluster health
kubectl get nodes
kubectl top nodes
kubectl get pods -A | grep -v Running | grep -v Completed

# API health
kubectl -n zenith-staging logs deploy/zenith-api --tail=100
kubectl -n zenith-staging exec deploy/zenith-api -- wget -qO- localhost:8080/health

# APISIX status
kubectl -n apisix logs deploy/apisix --tail=50
kubectl -n apisix exec deploy/apisix -- curl -s localhost:9180/apisix/admin/routes -H "X-API-KEY: $APISIX_ADMIN_KEY"

# Database health
kubectl -n zenith-staging get clusters
kubectl cnpg -n zenith-staging status <cluster-name>
```

## Backup & Recovery

### Automated Backups

| Component | Method | Frequency | Retention | Storage |
|-----------|--------|-----------|-----------|---------|
| CNPG databases | WAL archiving + barman | Continuous | 30 days | Hetzner S3 |
| Cluster state | Velero | Daily | 30 days | Hetzner S3 |
| Harbor registry | Built-in | Daily | 14 days | Hetzner S3 |
| Secrets | Sealed Secrets in Git | Every commit | Git history | GitHub |

### Recovery Procedures

See `docs/runbooks/disaster-recovery.md` for full procedures:
- Single database restore (PITR)
- Full cluster recovery from Velero
- Registry data recovery
- Secrets recovery from Git

## Scaling Guidelines

### When to Scale

| Metric | Threshold | Action |
|--------|-----------|--------|
| API CPU > 80% | 5 min sustained | Add replica or increase limits |
| API memory > 80% | Sustained | Increase memory limit |
| DB connections > 80% | Sustained | Increase pool or add read replica |
| Disk usage > 75% | Any | Expand PVC or clean old data |
| Node CPU > 85% | 10 min sustained | Add node to cluster |

### Horizontal Scaling

```bash
# Scale API
kubectl -n zenith-staging scale deploy/zenith-api --replicas=3

# Scale APISIX
kubectl -n apisix scale deploy/apisix --replicas=2
```

### Vertical Scaling (requires maintenance window)

```bash
# 1. Cordon node
kubectl cordon <node-name>

# 2. Resize in Hetzner Cloud console

# 3. Uncordon
kubectl uncordon <node-name>
```

## Access Control

### Admin Access

| System | URL | Auth |
|--------|-----|------|
| Grafana | `grafana.<domain>` | admin / (from Terraform state) |
| ArgoCD | `argocd.<domain>` | admin / (initial password in secret) |
| Keycloak | `auth.<domain>` | admin / (from keycloak-admin secret) |
| Harbor | `registry.<domain>` | admin / (from Terraform state) |
| APISIX Dashboard | (port-forward only) | Admin API key |
| Kubernetes | SSH + kubectl | kubeconfig |

### SSH Access

```bash
ssh zen-stage   # Staging server (77.42.88.149)
ssh ghasi       # Legacy server (161.35.82.211)
```

## Maintenance Windows

- **Preferred**: Tuesday/Thursday 06:00-08:00 UTC (low traffic)
- **Notify**: Post in Telegram 24h before
- **Duration**: Max 2 hours for planned maintenance
- **Rollback**: Always have a rollback plan documented before starting

## Runbook Index

See `docs/runbooks/README.md` for the complete list:
- Pod crash loops
- High resource usage
- Database issues
- Certificate expiry
- Build failures
- API degradation
- Storage issues
- Gateway issues
- Disaster recovery
