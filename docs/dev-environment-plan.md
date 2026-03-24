# Dev/Staging Environments, `zen dev` CLI, and Deploy Token System

> Implementation plan — created 2026-03-23

---

## 1. Overview

Three interconnected features:

1. **Dev/Staging Environments** — Pro+ users get auto-created staging environment alongside production
2. **`zen dev` CLI** — Local development with tunnels to remote staging services
3. **Deploy Token System** — Secure tokens for GitHub Actions CI/CD
4. **Terraform Provider** — Future work (not in this plan)

---

## 2. URL Pattern

```
Production:  {appname}.apps.freezenith.com
Staging:     {appname}.dev.apps.freezenith.com
```

Platform (our infra):
```
stage.freezenith.com              ← Zenith platform staging
app.stage.freezenith.com          ← Dashboard
*.apps.stage.freezenith.com       ← Customer production apps
*.dev.apps.stage.freezenith.com   ← Customer staging apps
```

---

## 3. Feature 1: Environments

### 3.1 Database Migration (`043_environments.up.sql`)

```sql
CREATE TABLE IF NOT EXISTS environments (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,             -- "production" or "staging"
    slug            TEXT NOT NULL,             -- "prod" or "staging"
    status          TEXT NOT NULL DEFAULT 'provisioning',
    is_default      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_environments_project_name ON environments(project_id, name);

ALTER TABLE apps ADD COLUMN IF NOT EXISTS environment_id TEXT REFERENCES environments(id);
ALTER TABLE managed_services ADD COLUMN IF NOT EXISTS environment_id TEXT REFERENCES environments(id);
ALTER TABLE user_buckets ADD COLUMN IF NOT EXISTS environment_id TEXT REFERENCES environments(id);
```

### 3.2 Entity

New file: `services/api/internal/entities/environment.go`

```go
type Environment struct {
    ID        string
    ProjectID string
    Name      string  // "production" | "staging"
    Slug      string  // "prod" | "staging"
    Status    string  // "provisioning" | "active" | "error"
    IsDefault bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

Add `EnvironmentID string` to `App`, `ManagedService`, `UserBucket` entities.

### 3.3 Repository

New interface `EnvironmentRepository` in `ports/repositories.go`:
- `CreateEnvironment(ctx, env) error`
- `GetEnvironment(ctx, id) (*Environment, error)`
- `ListEnvironmentsByProject(ctx, projectID) ([]Environment, error)`
- `GetEnvironmentByName(ctx, projectID, name) (*Environment, error)`
- `UpdateEnvironmentStatus(ctx, id, status) error`
- `DeleteEnvironment(ctx, id) error`

New adapter: `adapters/postgres/postgres_environment.go`

### 3.4 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/projects/:id/environments` | List environments |
| GET | `/projects/:id/environments/:envId` | Get environment |
| GET | `/projects/:id/environments/:envId/services` | List apps + managed services |

Modify existing:
- `POST /projects/:id/apps` — accept optional `environment_id` (default: production)
- `POST /projects/:id/managed-services` — accept optional `environment_id`

### 3.5 Auto-Create on Project Creation

When Pro+ user creates project via compose import:
1. Create `production` environment (is_default=true)
2. Create `staging` environment
3. Provision managed services in BOTH environments
4. Staging gets minimal resources

### 3.6 Staging Resource Limits

| Service | Production | Staging |
|---------|-----------|---------|
| App (CPU) | Per plan | 0.25 CPU |
| App (Memory) | Per plan | 256MB |
| PostgreSQL | Per plan | 256MB RAM, 1GB disk |
| Redis | Per plan | 64MB |

### 3.7 IngressRoute Changes

In `generateIngressRoute()`:
- Production: `` Host(`{subdomain}.apps.{baseDomain}`) `` (unchanged)
- Staging: `` Host(`{subdomain}.dev.apps.{baseDomain}`) ``

### 3.8 Infrastructure

- DNS: Add `*.dev.apps.stage.freezenith.com` wildcard A record → same Traefik LB IP
- TLS: Wildcard cert `dev-apps-wildcard-tls` via cert-manager

---

## 4. Feature 2: `zen dev` CLI

### 4.1 Command

```bash
zen dev [--project <name>] [--env staging]
```

### 4.2 Flow

```
1. Read docker-compose.yml from current directory
2. Authenticate (token from ~/.zen/config.yaml)
3. Find matching project on Zenith Cloud (by name/slug or .zenith.yaml)
4. Get staging environment + its managed services
5. Open WebSocket tunnels to staging services
6. Generate .env.zenith with local connection strings
7. Add .env.zenith to .gitignore
```

### 4.3 Output

```
$ zen dev

✓ Detected docker-compose.yml
✓ Found project "my-saas" on Zenith Cloud
✓ Staging environment found

Connecting to staging services...
  ✓ PostgreSQL  → localhost:5432 (tunnel)
  ✓ Redis       → localhost:6379 (tunnel)
  ✓ S3          → localhost:9000 (tunnel)

Environment variables written to .env.zenith

Ready! Run your app:
  npm run dev
```

### 4.4 Tunnel Architecture (API-proxied, no kubeconfig needed)

New API endpoints:
- `POST /projects/:id/environments/:envId/tunnels` — Create tunnel session
- `GET /tunnels/:tunnelId/ws` — WebSocket data channel
- `DELETE /tunnels/:tunnelId` — Close tunnel

Server-side: uses `k8s.io/client-go/tools/portforward` to connect to managed service pods.
Client-side: WebSocket → local TCP port.

### 4.5 Files to Create

| File | Description |
|------|-------------|
| `cli/cmd/dev/dev.go` | Main command |
| `cli/cmd/dev/tunnel.go` | WebSocket tunnel client |
| `cli/cmd/dev/envfile.go` | .env.zenith generation |
| `services/api/internal/handlers/tunnel.go` | Tunnel API handler |
| `services/api/internal/services/tunnel.go` | Tunnel service (K8s port-forward) |

### 4.6 .zenith.yaml (optional project config)

```yaml
project_id: proj_abc123
default_env: staging
```

If present in project root, `zen dev` skips project matching.

---

## 5. Feature 3: Deploy Token System

### 5.1 Token Format

```
Token ID:     znt_id_a1b2c3d4e5f6g7h8     (prefix + 16 hex chars)
Token Secret: znt_sk_K8xP2mQ9vL...         (prefix + 64 hex chars)
```

- ID is for lookup (indexed, public)
- Secret is hashed with Argon2id (never stored in plain text)
- Secret shown ONCE on creation

### 5.2 Database Migration (`044_deploy_tokens.up.sql`)

```sql
CREATE TABLE IF NOT EXISTS deploy_tokens (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id          TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    token_id            TEXT NOT NULL UNIQUE,      -- znt_id_...
    token_prefix        TEXT NOT NULL,              -- first 8 chars of secret
    token_hash          TEXT NOT NULL,              -- Argon2id hash
    scopes              TEXT[] NOT NULL DEFAULT '{}',
    last_used_at        TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ,
    -- Rotation
    previous_hash       TEXT,
    previous_expires_at TIMESTAMPTZ,
    rotated_at          TIMESTAMPTZ,
    -- Lifecycle
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at          TIMESTAMPTZ
);
CREATE INDEX idx_deploy_tokens_token_id ON deploy_tokens(token_id);
CREATE INDEX idx_deploy_tokens_user ON deploy_tokens(user_id);
CREATE INDEX idx_deploy_tokens_project ON deploy_tokens(project_id);
```

### 5.3 Scopes

```go
const (
    ScopeDeployStaging    = "deploy:staging"
    ScopeDeployProduction = "deploy:production"
    ScopeAppRead          = "app:read"
    ScopeAppWrite         = "app:write"
    ScopeDBRead           = "db:read"
    ScopeLogsRead         = "logs:read"
    ScopeInfraAll         = "infra:*"  // for Terraform provider
)
```

### 5.4 Auth Flow

```
Header: Authorization: DeployToken znt_id_xxx:znt_sk_yyy

1. Parse → extract token_id
2. DB lookup by token_id
3. Check: revoked_at IS NULL
4. Check: expires_at > NOW()
5. Check: scope includes required permission
6. Argon2id verify(secret, token_hash)
   → if fails, try previous_hash (if within grace period)
7. Update last_used_at
8. Allow request
```

### 5.5 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/projects/:id/deploy-tokens` | Create token (returns ID + secret once) |
| GET | `/projects/:id/deploy-tokens` | List tokens (ID + prefix only) |
| DELETE | `/projects/:id/deploy-tokens/:tokenId` | Revoke token |
| POST | `/projects/:id/deploy-tokens/:tokenId/rotate` | Rotate (24h grace period) |

### 5.6 Argon2id Parameters

```go
// time=1, memory=64MB, threads=4, keyLen=32
argon2.IDKey([]byte(secret), salt, 1, 64*1024, 4, 32)
```

### 5.7 GitHub Actions

Two separate actions in `dotechhq/zenith-actions` repo:

**`zenith-stage@v1`:**
```yaml
inputs:
  token-id:
    required: true
  token-secret:
    required: true
  app:
    required: true
  image:
    required: true
```

**`zenith-prod@v1`:** Same inputs, deploys to production.

Both call `POST /projects/:id/deploy` with environment parameter.

---

## 6. Implementation Phases

### Phase 1: Database & Entity Foundation
| # | Task | Files |
|---|------|-------|
| 1 | Migration: environments table | `migrations/043_environments.up.sql` |
| 2 | Environment entity | `entities/environment.go` |
| 3 | Add EnvironmentID to App/ManagedService/UserBucket | `entities/app.go`, `entities/managed_service.go` |
| 4 | EnvironmentRepository interface | `ports/repositories.go` |
| 5 | Postgres environment adapter | `adapters/postgres/postgres_environment.go` |

### Phase 2: Environment API
| # | Task | Files |
|---|------|-------|
| 6 | Environment handler (CRUD) | `handlers/environment.go` |
| 7 | Auto-create envs on project creation (Pro+) | `handlers/compose.go` (modify) |
| 8 | Environment-aware managed service provisioning | `services/managed_service.go` (modify) |
| 9 | Staging resource limits | `services/deploy/k8s_resources.go` (modify) |
| 10 | Staging IngressRoute subdomain | `services/deploy/k8s_resources.go` (modify) |

### Phase 3: Infrastructure
| # | Task | Files |
|---|------|-------|
| 11 | DNS wildcard `*.dev.apps.` | Hetzner DNS / Cloudflare |
| 12 | Cert-manager staging wildcard | `infra/helm/` values |

### Phase 4: Deploy Token System
| # | Task | Files |
|---|------|-------|
| 13 | Migration: deploy_tokens table | `migrations/044_deploy_tokens.up.sql` |
| 14 | DeployToken entity + scopes | `entities/deploy_token.go` |
| 15 | DeployTokenRepository + Argon2id | `adapters/postgres/postgres_deploy_token.go` |
| 16 | Deploy token handler (CRUD + rotate) | `handlers/deploy_token.go` |
| 17 | DeployTokenAuth middleware | `middleware/auth.go` (modify) |

### Phase 5: `zen dev` CLI
| # | Task | Files |
|---|------|-------|
| 18 | `zen dev` command structure | `cli/cmd/dev/dev.go` |
| 19 | Compose file reader | `cli/cmd/dev/compose.go` |
| 20 | Tunnel API handler + service | `handlers/tunnel.go`, `services/tunnel.go` |
| 21 | WebSocket tunnel client | `cli/cmd/dev/tunnel.go` |
| 22 | `.env.zenith` generation | `cli/cmd/dev/envfile.go` |

### Phase 6: GitHub Actions
| # | Task | Files |
|---|------|-------|
| 23 | Create `dotechhq/zenith-actions` repo | Separate repo |
| 24 | `zenith-stage` action | `action.yml`, `entrypoint.sh` |
| 25 | `zenith-prod` action | `action.yml`, `entrypoint.sh` |

### Phase 7: Testing & Docs
| # | Task | Files |
|---|------|-------|
| 26 | Unit tests (environment, tokens) | `*_test.go` files |
| 27 | Integration tests (`zen dev`) | CLI test files |
| 28 | Update OpenAPI spec | `docs/openapi.yaml` |

---

## 7. Summary Table

| # | Task | Phase | Status |
|---|------|-------|--------|
| 1 | Environments migration + entity | P1 | ✅ Done |
| 2 | EnvironmentRepository + adapter | P1 | ✅ Done |
| 3 | Environment API handlers | P2 | ✅ Done |
| 4 | Auto-create staging on project creation | P2 | ✅ Done |
| 5 | Staging resource limits | P2 | ✅ Done |
| 6 | Staging IngressRoute (`*.dev.apps.`) | P2 | ✅ Done |
| 7 | DNS wildcard + cert-manager | P3 | ⬜ Needs infra work |
| 8 | Deploy tokens migration + entity | P4 | ✅ Done |
| 9 | Deploy token Argon2id + repository | P4 | ✅ Done |
| 10 | Deploy token handlers (CRUD + rotate) | P4 | ✅ Done |
| 11 | DeployTokenAuth middleware | P4 | ✅ Done |
| 12 | `zen dev` command | P5 | ✅ Done |
| 13 | Tunnel API (WebSocket port-forward) | P5 | ⬜ Stub only |
| 14 | `.env.zenith` generation | P5 | ✅ Done |
| 15 | GitHub Actions (stage + prod) | P6 | ✅ Done |
| 16 | Frontend (tokens + environments pages) | P6 | ✅ Done |
| 17 | Tests + docs | P7 | ⬜ Not started |
| 18 | Deploy to staging | — | ⬜ Needs deploy |
| 19 | Publish GitHub Actions repo | — | ⬜ Needs separate repo |

---

## 8. Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| WebSocket tunnel complexity | Start with simple TCP-over-WebSocket. Use `gorilla/websocket` |
| Argon2id performance (~100ms) | Lookup by token_id first, only verify on match. Optional Redis cache |
| Staging resource overhead | Minimal resources (0.25 CPU, 256MB). Monitor cluster capacity |
| Backward compat with `api_keys` | Keep existing `api_keys` table untouched. Deploy tokens are separate |
| DNS propagation | Same Hetzner DNS. Wildcard propagates in minutes |
