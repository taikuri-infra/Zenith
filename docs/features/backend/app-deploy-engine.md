# App Deploy Engine — Backend Module Doc

## 1. Purpose

The App Deploy Engine enables users to deploy applications from Git repositories to Kubernetes with zero configuration. Full lifecycle: clone → detect framework → generate Dockerfile → build container (Kaniko) → deploy to K8s → serve with HTTPS.

## 2. Entities

| Entity | Key Fields |
|--------|------------|
| `App` | ID, UserID, Name, RepoURL, Branch, Framework, Status, Subdomain, Port, URL |
| `Deployment` | ID, AppID, GitSHA, Status, BuildLog, DeployLog |
| `EnvVar` | Key, Value |

### Status Flow

```
App:        pending → building → running → stopped | failed
Deployment: pending → building → deploying → active | failed | rolled_back
```

## 3. Services (Use Cases)

| Service | File | Responsibility |
|---------|------|----------------|
| `Builder` | `deploy/builder.go` | Clone → detect → Dockerfile → image tag |
| `Pipeline` | `deploy/pipeline.go` | Async build runner, goroutine management, cancellation |
| `Deployer` | `deploy/deployer.go` | Creates K8s Deployment + Service + IngressRoute |

### Framework Detection (`deploy/detect.go`)

9 frameworks: Next.js, Go, Python, Django, Flask, Rails, Express, Static, Dockerfile.

## 4. Ports

```go
type AppRepository interface {
    CreateApp / GetApp / ListAppsByUser / UpdateApp / DeleteApp
    CreateDeployment / ListDeployments / GetActiveDeployment / UpdateDeploymentStatus
    SetEnvVars / GetEnvVars / DeleteEnvVar
}
```

## 5. Adapters

| Adapter | File | Backend |
|---------|------|---------|
| `MemoryAppRepository` | `store/memory_app.go` | In-memory (dev/test) |
| `PostgresAppRepository` | `store/postgres_app.go` | PostgreSQL via `pgx/v5` |

Schema: `store/migrations/001_create_apps.sql` — `apps`, `deployments`, `app_env_vars` tables.

## 6. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/apps` | Create app from Git repo |
| GET | `/api/v1/apps` | List user's apps |
| GET | `/api/v1/apps/:id` | Get app detail |
| DELETE | `/api/v1/apps/:id` | Delete app |
| GET | `/api/v1/apps/:id/deployments` | Deployment history |
| POST | `/api/v1/apps/:id/rollback` | Rollback to deployment |
| GET/PUT | `/api/v1/apps/:id/env` | Get/Set env vars |
| DELETE | `/api/v1/apps/:id/env/:key` | Delete env var |
| POST | `/api/v1/webhooks/github` | GitHub push webhook (HMAC-SHA256) |

## 7. Validation Rules

- App name: required, sanitized to lowercase + hyphens for subdomain
- Repo URL: required, valid URL
- Branch: defaults to `main`
- Webhook: HMAC-SHA256 signature required (`X-Hub-Signature-256`)

## 8. Security Model

- All endpoints require JWT (except webhook with HMAC verification)
- Non-root users in all generated Dockerfiles
- K8s resource limits enforced (CPU: 500m, Memory: 512Mi)
- Cascade delete: app → deployments + env vars

## 9. Testing Strategy

| Test File | Tests |
|-----------|-------|
| `store/app_repo_test.go` | 30 — Repository CRUD |
| `deploy/detect_test.go` | 28 — Detection + Dockerfile |
| `deploy/builder_test.go` | 9 — Builder, Kaniko, Pipeline |
| `deploy/k8s_test.go` | 7 — K8s resources, Deployer |
| `handlers/handlers_v2_test.go` | 15 — API handlers |
| **Total** | **89** |

Run: `cd services/api && go test ./...`

## 10. Future Improvements

- Build log streaming via WebSocket
- Kaniko job submission to K8s
- Custom domains with automatic TLS
- Preview environments for PRs
- Blue-green / canary deployments
