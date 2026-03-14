# 09 — Quick Reference (Cheatsheet)

> **Purpose:** Fast lookup for commands, paths, URLs, credentials, and common tasks

---

## Key Paths

| What | Path |
|------|------|
| Backend API | `services/api/` |
| Entities | `services/api/internal/entities/` |
| Ports | `services/api/internal/ports/` |
| Services | `services/api/internal/services/` |
| Handlers | `services/api/internal/handlers/` |
| PostgreSQL adapter | `services/api/internal/adapters/postgres/` |
| Memory adapter | `services/api/internal/adapters/memory/` |
| Migrations | `services/api/internal/adapters/postgres/migrations/` |
| Config | `services/api/internal/config/config.go` |
| Main.go | `services/api/cmd/server/main.go` |
| Web app | `apps/web/` |
| Web pages | `apps/web/src/app/` |
| Web API client | `apps/web/src/lib/api.ts` |
| Web hooks | `apps/web/src/hooks/` |
| Mission Control | `apps/mission-control/` |
| Landing page | `apps/landing/` |
| Terraform modules | `infra/terraform/modules/k8s-platform/` |
| Helm charts | `infra/helm/` |
| CI workflows | `.github/workflows/` |
| Smoke tests | `infra/scripts/` |

---

## Commands

### Backend
```bash
# Verify code compiles
cd services/api && GO111MODULE=on go vet ./internal/...

# Run tests
cd services/api && go test ./internal/... -v

# Run specific test
cd services/api && go test ./internal/handlers/ -run TestMyHandler -v

# Create migration
lich migration create "my_change"
```

### Frontend
```bash
# Install deps
pnpm install

# Run all apps
pnpm dev

# Run single app
cd apps/web && pnpm dev              # :3000
cd apps/mission-control && pnpm dev  # :3100
cd apps/landing && pnpm dev          # :3200

# Lint
cd apps/web && npx next lint --quiet

# Test
cd apps/web && pnpm test
```

### Deploy
```bash
# Deploy to staging (most common)
make deploy-api     # API
make deploy-web     # Web
make deploy-mc      # Mission Control
make deploy-all     # Everything

# Terraform
cd infra/terraform/staging-k8s && terraform plan
cd infra/terraform/staging-k8s && terraform apply

# Helm lint
helm lint infra/helm/zenith-api/
```

### Kubernetes (on staging)
```bash
ssh zen-stage

# Pods
kubectl get pods -n zenith-staging
kubectl get pods -n zenith-apps
kubectl get pods -n zenith-builds
kubectl get pods -A | grep -v Running  # Find problems

# Logs
kubectl logs -n zenith-staging deploy/zenith-api -f --tail=100
kubectl logs -n zenith-apps deploy/<app-name> -f

# Database
kubectl exec -n zenith-staging zenith-postgres-1 -c postgres -- \
  psql -U zenith -d zenith -c "SELECT * FROM users LIMIT 5;"

# Events
kubectl get events -n zenith-staging --sort-by='.lastTimestamp' | tail -20

# Restart
kubectl rollout restart deploy/zenith-api -n zenith-staging

# ArgoCD
kubectl get applications -n argocd
```

### Git
```bash
# Daily flow
git checkout staging
# ... make changes ...
git add -A
git commit -m "feat(section): description"
git push origin staging

# Conventional commits
feat:     # New feature
fix:      # Bug fix
chore:    # Maintenance
docs:     # Documentation
refactor: # Code refactor
test:     # Tests
perf:     # Performance
```

---

## URLs

| Service | Staging | Purpose |
|---------|---------|---------|
| Landing | stage.freezenith.com | Marketing |
| Web | app.stage.freezenith.com | Customer dashboard |
| MC | mc.stage.freezenith.com | Admin panel |
| API | api.stage.freezenith.com | Backend API |
| Keycloak | auth.stage.freezenith.com | Identity |
| Harbor (internal) | registry.stage.freezenith.com | Platform images |
| Harbor (customer) | hub.stage.freezenith.com | Customer images |
| ArgoCD | argocd.stage.freezenith.com | GitOps |
| Customer apps | *.apps.stage.freezenith.com | Deployed apps |
| Gateways | *.gw.stage.freezenith.com | API gateways |

---

## Credentials (Staging)

| Service | Username | Password |
|---------|----------|----------|
| Admin login | admin@freezenith.com | 8i3wIotgaZEgxVnXMEpA |
| Smoke test | smoke-ci@zenith.dev | SmokeTest1234 |

---

## SSH

| Alias | IP | Purpose |
|-------|------|---------|
| `zen-stage` | 77.42.88.149 | Staging cluster |
| `ghasi` | 161.35.82.211 | Old production (legacy) |

**NEVER** confuse these two servers!

---

## Architecture Quick Reference

```
Entity → Port (interface) → Adapter (implementation) → Service → Handler → Route
  │         │                    │                        │         │
  │         │                    ├── postgres/             │         │
  │         │                    ├── memory/               │         │
  │         │                    └── k8sclient/            │         │
  │         │                                              │         │
  │         ├── repositories.go (34 interfaces)            │         │
  │         └── infrastructure.go (12 interfaces)          │         │
  │                                                        │         │
  └── 37 files, zero imports                    24 files   83 files

main.go wires everything together (1500 lines)
```

---

## APISIX Rate Limits

| Route Type | Limit | Applies To |
|-----------|-------|-----------|
| Public (`/api/v1/auth/*`) | 30 req/60s per IP | Login, register, OAuth |
| Protected (`/api/*`) | 500 req/60s per IP | All authenticated APIs |

If you get 429: wait 60 seconds, or test from different IP.

---

## Plan Limits Quick Reference

| Resource | Free | Pro | Team | Business |
|----------|------|-----|------|----------|
| Apps | 1 | 5 | 20 | Unlimited |
| Databases | 1 | 3 | 10 | Unlimited |
| Storage | 1GB | 10GB | 100GB | Unlimited |
| Team members | 1 | 1 | ∞ | ∞ |
| Sleep | 15min | Never | Never | Never |
| Custom domain | No | Yes | Yes | Yes |
| SSO | No | No | No | Yes |

---

## Adding a Feature (Checklist)

```
□ Entity (if new data model)        → entities/
□ Port (if new repo method)         → ports/repositories.go
□ Postgres adapter                  → adapters/postgres/
□ Memory adapter                    → adapters/memory/
□ Service (business logic)          → services/
□ Handler (HTTP endpoint)           → handlers/
□ Route registration                → cmd/server/main.go
□ Backend verify                    → go vet ./internal/...
□ Frontend API method               → apps/web/src/lib/api.ts
□ Frontend demo stub                → apps/web/src/lib/demo-api.ts
□ Frontend page/component           → apps/web/src/app/
□ Frontend verify                   → npx next lint --quiet
□ Commit (conventional)             → git commit -m "feat(scope): description"
□ Push to staging                   → git push origin staging
□ Deploy                            → make deploy-api / deploy-web
```
