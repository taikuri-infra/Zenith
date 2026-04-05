# Zenith Remaining Features — Detailed Implementation Plan

> **Context**: These tasks continue the audit fix work. 4 items from the original list were already implemented:
> - Deployment rollback ✅ (`POST /apps/:appId/rollback` — fully working)
> - Deployment webhooks ✅ (CRUD + delivery entity — but dispatcher NOT wired, see Task 4)
> - API documentation ✅ (OpenAPI 3.0.3 + Scalar UI at `/docs`)
> - Team audit trail ✅ (admin + user-scoped audit + CSV/JSON export)
>
> **Test command**: `GO111MODULE=on go test ./...` — must pass 34/34 packages after each task.
> **Build command**: `GO111MODULE=on go build ./...` — must compile.
> **DO NOT** run go build/test without `GO111MODULE=on`.

---

## Task 1: Custom Health Check Path Per App

**Goal**: Let users set a custom health check endpoint (e.g., `/healthz`, `/api/health`) instead of hardcoded `/`.

### 1a. Migration 047 — add `health_check_path` column

**Create** `internal/adapters/postgres/migrations/047_app_health_check_path.up.sql`:
```sql
ALTER TABLE apps ADD COLUMN IF NOT EXISTS health_check_path TEXT NOT NULL DEFAULT '/';
```

**Create** `internal/adapters/postgres/migrations/047_app_health_check_path.down.sql`:
```sql
ALTER TABLE apps DROP COLUMN IF EXISTS health_check_path;
```

### 1b. Entity — add field

**File**: `internal/entities/app.go`

Add to the `App` struct, after the `Replicas` field:
```go
// HealthCheckPath is the HTTP path used for K8s liveness/readiness probes. Defaults to "/".
HealthCheckPath string `json:"health_check_path"`
```

### 1c. Postgres adapter — wire the column

**File**: `internal/adapters/postgres/postgres_app.go`

1. Update `appColumns` constant — add `health_check_path` between `replicas` and `created_at`:
   ```
   ..., replicas, health_check_path, created_at, updated_at
   ```

2. Update `scanApp` — add `&app.HealthCheckPath` between `&app.Replicas` and `&app.CreatedAt`:
   ```go
   &app.Replicas, &app.HealthCheckPath,
   &app.CreatedAt, &app.UpdatedAt)
   ```
   After scan, add default:
   ```go
   if app.HealthCheckPath == "" {
       app.HealthCheckPath = "/"
   }
   ```

3. Update `INSERT INTO apps` — add `health_check_path` to column list and values (`$21`), pass `"/"` as the value.

4. In `UpdateApp`, add:
   ```go
   if input.HealthCheckPath != nil {
       sets = append(sets, fmt.Sprintf("health_check_path = $%d", argIdx))
       args = append(args, *input.HealthCheckPath)
       argIdx++
   }
   ```

### 1d. DTO — add to UpdateAppInput

**File**: `internal/dto/inputs.go`

Add to `UpdateAppInput`:
```go
HealthCheckPath *string `json:"health_check_path,omitempty"`
```

### 1e. Memory adapter — update UpdateApp

**File**: `internal/adapters/memory/memory_app.go`

In `UpdateApp`, add:
```go
if input.HealthCheckPath != nil {
    app.HealthCheckPath = *input.HealthCheckPath
}
```

### 1f. K8s resources — use app.HealthCheckPath

**File**: `internal/services/deploy/k8s_resources.go`

In `generateDeployment`, in the `default` case for web apps (around line 248-266):

Replace the two hardcoded `"path": "/"` with:
```go
healthPath := app.HealthCheckPath
if healthPath == "" {
    healthPath = "/"
}
```
Then use `healthPath` in both `readinessProbe` and `livenessProbe`:
```go
"httpGet": map[string]interface{}{
    "path": healthPath,
    "port": port,
},
```

### 1g. API handler — accept health_check_path on create/update

**File**: `internal/handlers/apps_v2.go`

1. Add `HealthCheckPath string` to `CreateAppV2Request` struct:
   ```go
   HealthCheckPath string `json:"health_check_path,omitempty"`
   ```

2. In `Create`, after setting the app fields, add:
   ```go
   // Default health check path
   healthPath := req.HealthCheckPath
   if healthPath == "" {
       healthPath = "/"
   }
   ```
   And pass it to `CreateApp` (add `HealthCheckPath` to `CreateAppInput` DTO too).

3. Add `HealthCheckPath` to `AppV2Response`:
   ```go
   HealthCheckPath string `json:"health_check_path"`
   ```
   And populate it in `appToResponse`:
   ```go
   HealthCheckPath: app.HealthCheckPath,
   ```

### 1h. Validate health check path

In `Create` and `Update` handlers, validate:
```go
if req.HealthCheckPath != "" {
    if !strings.HasPrefix(req.HealthCheckPath, "/") {
        return NewBadRequest("health_check_path must start with /")
    }
    if len(req.HealthCheckPath) > 256 {
        return NewBadRequest("health_check_path too long")
    }
}
```

**Verify**: `GO111MODULE=on go build ./... && GO111MODULE=on go test ./...`

---

## Task 2: Environment-Aware Deploys (CI Deploy → Staging/Production)

**Goal**: When `ciDeployRequest.Environment = "staging"`, deploy to the staging environment. Apps get linked to environments.

### 2a. Update CI deploy handler to resolve environment

**File**: `internal/handlers/ci_deploy.go`

The handler already has `EnvironmentRepository` available via the project repo. Add `envRepo ports.EnvironmentRepository` to `CIDeployHandler`:

```go
type CIDeployHandler struct {
    appRepo     ports.AppRepository
    projectRepo ports.ProjectRepository
    envRepo     ports.EnvironmentRepository  // ADD THIS
    pipeline    AppImageDeployer
    baseDomain  string
}
```

Update constructor:
```go
func NewCIDeployHandler(appRepo ports.AppRepository, projectRepo ports.ProjectRepository, envRepo ports.EnvironmentRepository, pipeline AppImageDeployer, baseDomain string) *CIDeployHandler {
```

### 2b. Resolve environment in Deploy handler

In the `Deploy` method, after finding the app but before creating the deployment:

```go
// Resolve target environment
if req.Environment != "" && projectID != "" {
    envName := entities.EnvironmentProduction
    if req.Environment == "staging" {
        envName = entities.EnvironmentStaging
    }
    env, envErr := h.envRepo.GetEnvironmentByName(c.Context(), projectID, envName)
    if envErr != nil && req.Environment == "staging" {
        return NewBadRequest("staging environment not available for this project (requires Pro+ plan)")
    }
    if envErr == nil && app.EnvironmentID != env.ID {
        // Update app's environment ID
        app.EnvironmentID = env.ID
    }
}
```

### 2c. Wire envRepo in main.go

**File**: `cmd/server/main.go`

Find the `NewCIDeployHandler` call and add `envRepo`:
```go
ciDeployHandler := handlers.NewCIDeployHandler(appRepo, projectRepo, envRepo, pipeline, cfg.BaseDomain)
```

### 2d. Update app creation to link environment

**File**: `internal/handlers/project_v2.go` or `apps_v2.go`

In `CreateAppV2Request`, add:
```go
EnvironmentID string `json:"environment_id,omitempty"`
```

In the `Create` handler, if `EnvironmentID` is provided, validate it belongs to the project:
```go
if req.EnvironmentID != "" {
    env, envErr := h.envRepo.GetEnvironment(c.Context(), req.EnvironmentID)
    if envErr != nil {
        return NewNotFound("environment not found")
    }
    if env.ProjectID != projectID {
        return NewBadRequest("environment does not belong to this project")
    }
}
```

Then pass `req.EnvironmentID` to `CreateAppInput`.

### 2e. Update CreateAppInput + postgres adapter

**File**: `internal/dto/inputs.go`

Add to `CreateAppInput`:
```go
EnvironmentID string `json:"environment_id,omitempty"`
```

**File**: `internal/adapters/postgres/postgres_app.go`

Update the INSERT statement to include `environment_id`:
```sql
INSERT INTO apps (id, ..., environment_id, ...) VALUES ($1, ..., $N, ...)
```
Pass `input.EnvironmentID` (can be empty string, which is fine for the nullable TEXT column — use `nil` if empty).

### 2f. Update return struct to show EnvironmentID

**File**: `internal/handlers/apps_v2.go`

Add `EnvironmentID` to `AppV2Response`:
```go
EnvironmentID string `json:"environment_id,omitempty"`
```

Populate in `appToResponse`:
```go
EnvironmentID: app.EnvironmentID,
```

**Verify**: `GO111MODULE=on go build ./... && GO111MODULE=on go test ./...`

---

## Task 3: Post-Deploy Hooks

**Goal**: Allow users to define commands that run after a successful deployment (e.g., database migrations, cache clearing).

### 3a. Entity

**Create** `internal/entities/deploy_hook.go`:
```go
package entities

type DeployHookType string

const (
    DeployHookHTTP    DeployHookType = "http"    // POST to URL
    DeployHookCommand DeployHookType = "command" // Run in app container via kubectl exec
)

type DeployHook struct {
    ID        string         `json:"id"`
    AppID     string         `json:"app_id"`
    Name      string         `json:"name"`
    Type      DeployHookType `json:"type"`
    // For HTTP hooks: the URL to POST to
    URL string `json:"url,omitempty"`
    // For command hooks: the command to exec in the running container
    Command string `json:"command,omitempty"`
    // Order determines execution sequence (lower = first)
    Order  int  `json:"order"`
    Active bool `json:"active"`
    Timestamps
}
```

### 3b. Migration 048

**Create** `internal/adapters/postgres/migrations/048_deploy_hooks.up.sql`:
```sql
CREATE TABLE IF NOT EXISTS deploy_hooks (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'http',
    url TEXT,
    command TEXT,
    "order" INT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_deploy_hooks_app ON deploy_hooks(app_id);
```

**Create** `internal/adapters/postgres/migrations/048_deploy_hooks.down.sql`:
```sql
DROP TABLE IF EXISTS deploy_hooks;
```

### 3c. Repository interface

**File**: `internal/ports/repositories.go`

Add:
```go
// DeployHookRepository defines deploy hook persistence operations.
type DeployHookRepository interface {
    CreateHook(ctx context.Context, hook *entities.DeployHook) error
    GetHook(ctx context.Context, id string) (*entities.DeployHook, error)
    ListHooksByApp(ctx context.Context, appID string) ([]entities.DeployHook, error)
    UpdateHook(ctx context.Context, id string, name *string, url *string, command *string, order *int, active *bool) (*entities.DeployHook, error)
    DeleteHook(ctx context.Context, id string) error
}
```

### 3d. Postgres implementation

**Create** `internal/adapters/postgres/postgres_deploy_hook.go`:

Standard CRUD implementation:
- `CreateHook`: INSERT with uuid.New()
- `GetHook`: SELECT by id
- `ListHooksByApp`: SELECT WHERE app_id ORDER BY "order"
- `UpdateHook`: Dynamic SET clause
- `DeleteHook`: DELETE by id

### 3e. Memory implementation

**Create** `internal/adapters/memory/memory_deploy_hook.go`:

In-memory map implementation matching the interface.

### 3f. Handler

**Create** `internal/handlers/deploy_hook.go`:

Endpoints:
- `POST /api/v1/apps/:appId/hooks` — Create hook
- `GET /api/v1/apps/:appId/hooks` — List hooks
- `PUT /api/v1/apps/:appId/hooks/:hookId` — Update hook
- `DELETE /api/v1/apps/:appId/hooks/:hookId` — Delete hook

Validation:
- `name` required, max 100 chars
- `type` must be "http" or "command"
- If type=http, `url` required, must be valid URL
- If type=command, `command` required, max 512 chars
- Max 10 hooks per app

### 3g. Execute hooks after deploy

**File**: `internal/services/deploy/pipeline.go`

After the successful deploy (line 111-113, after `UpdateDeploymentStatus → Active`):

```go
// Execute post-deploy hooks
if p.hookRepo != nil {
    hooks, hErr := p.hookRepo.ListHooksByApp(ctx, app.ID)
    if hErr == nil {
        for _, hook := range hooks {
            if !hook.Active {
                continue
            }
            p.emitLog(deployment.ID, "hook", fmt.Sprintf("Running hook: %s", hook.Name))
            switch hook.Type {
            case entities.DeployHookHTTP:
                p.executeHTTPHook(hook, app, deployment, image)
            case entities.DeployHookCommand:
                p.executeCommandHook(ctx, hook, app)
            }
        }
    }
}
```

Add `hookRepo ports.DeployHookRepository` to the `Pipeline` struct and constructor.

### 3h. HTTP hook execution

```go
func (p *Pipeline) executeHTTPHook(hook entities.DeployHook, app *entities.App, deployment *entities.Deployment, image string) {
    payload, _ := json.Marshal(map[string]string{
        "app":           app.Name,
        "deployment_id": deployment.ID,
        "image":         image,
        "status":        "success",
    })
    req, _ := http.NewRequest("POST", hook.URL, bytes.NewReader(payload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Zenith-Hook", hook.Name)

    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        slog.Warn("hook failed", "hook", hook.Name, "error", err)
        return
    }
    resp.Body.Close()
}
```

### 3i. Command hook execution (kubectl exec)

```go
func (p *Pipeline) executeCommandHook(ctx context.Context, hook entities.DeployHook, app *entities.App) {
    // kubectl exec into the first ready pod of the app's deployment
    // The k8sClient needs an ExecInPod method — or use a simple exec fallback.
    slog.Info("command hook executed", "hook", hook.Name, "app", app.Name, "command", hook.Command)
    // TODO: implement via k8sClient.ExecInPod(ctx, "zenith-apps", app.Subdomain, hook.Command)
}
```

Note: Full `kubectl exec` implementation requires adding `ExecInPod` to the k8sclient. For now, log a placeholder and implement the HTTP hooks fully.

### 3j. Wire in main.go

```go
hookRepo := postgres.NewPostgresDeployHookRepository(pool) // or memory.New...
pipeline.SetHookRepo(hookRepo)

// Routes under appByID:
hookHandler := handlers.NewDeployHookHandler(hookRepo)
appByID.Post("/hooks", hookHandler.Create)
appByID.Get("/hooks", hookHandler.List)
appByID.Put("/hooks/:hookId", hookHandler.Update)
appByID.Delete("/hooks/:hookId", hookHandler.Delete)
```

**Verify**: `GO111MODULE=on go build ./... && GO111MODULE=on go test ./...`

---

## Task 4: Webhook Dispatcher (Fire HTTP Requests on Events)

**Goal**: The webhook system (CRUD) exists but never actually sends HTTP requests. Wire it up.

### 4a. Create webhook delivery service

**Create** `internal/services/webhook_delivery.go`:

```go
package services

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "log/slog"
    "net/http"
    "time"

    "github.com/dotechhq/zenith/services/api/internal/entities"
    "github.com/dotechhq/zenith/services/api/internal/ports"
)

type WebhookDeliveryService struct {
    webhookRepo ports.UserWebhookRepository
    httpClient  *http.Client
}

func NewWebhookDeliveryService(webhookRepo ports.UserWebhookRepository) *WebhookDeliveryService {
    return &WebhookDeliveryService{
        webhookRepo: webhookRepo,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

// DispatchEvent sends webhook payloads to all active webhooks matching the event.
func (s *WebhookDeliveryService) DispatchEvent(ctx context.Context, userID string, event entities.WebhookEvent, payload map[string]interface{}) {
    webhooks, err := s.webhookRepo.ListWebhooksByUser(ctx, userID)
    if err != nil {
        slog.Error("webhook dispatch: failed to list webhooks", "user_id", userID, "error", err)
        return
    }

    payloadJSON, _ := json.Marshal(payload)

    for _, wh := range webhooks {
        if !wh.Active {
            continue
        }
        // Check if webhook subscribes to this event
        subscribed := false
        for _, e := range wh.Events {
            if e == event {
                subscribed = true
                break
            }
        }
        if !subscribed {
            continue
        }

        go s.deliver(ctx, wh, event, payloadJSON)
    }
}

func (s *WebhookDeliveryService) deliver(ctx context.Context, wh entities.UserWebhook, event entities.WebhookEvent, payload []byte) {
    req, err := http.NewRequestWithContext(ctx, "POST", wh.URL, bytes.NewReader(payload))
    if err != nil {
        s.record(ctx, wh.ID, event, string(payload), entities.WebhookDeliveryFailed, 0, err.Error())
        return
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Zenith-Event", string(event))

    // HMAC-SHA256 signature
    if wh.Secret != "" {
        mac := hmac.New(sha256.New, []byte(wh.Secret))
        mac.Write(payload)
        sig := hex.EncodeToString(mac.Sum(nil))
        req.Header.Set("X-Zenith-Signature", "sha256="+sig)
    }

    resp, err := s.httpClient.Do(req)
    if err != nil {
        s.record(ctx, wh.ID, event, string(payload), entities.WebhookDeliveryFailed, 0, err.Error())
        return
    }
    defer resp.Body.Close()
    io.Copy(io.Discard, resp.Body)

    status := entities.WebhookDeliverySuccess
    errMsg := ""
    if resp.StatusCode >= 400 {
        status = entities.WebhookDeliveryFailed
        errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
    }
    s.record(ctx, wh.ID, event, string(payload), status, resp.StatusCode, errMsg)
}

func (s *WebhookDeliveryService) record(ctx context.Context, webhookID string, event entities.WebhookEvent, payload string, status entities.WebhookDeliveryStatus, code int, errMsg string) {
    if _, err := s.webhookRepo.RecordDelivery(ctx, webhookID, event, payload, status, code, errMsg); err != nil {
        slog.Error("webhook: failed to record delivery", "webhook_id", webhookID, "error", err)
    }
}
```

### 4b. Wire into deploy pipeline

**File**: `internal/services/deploy/pipeline.go`

Add `webhookSvc *services.WebhookDeliveryService` to the Pipeline struct and a setter `SetWebhookService`.

After `emitEvent(EventDeployComplete, ...)` (line 113), add:
```go
if p.webhookSvc != nil {
    p.webhookSvc.DispatchEvent(ctx, app.UserID, entities.WebhookEventDeploySuccess, map[string]interface{}{
        "app":           app.Name,
        "app_id":        app.ID,
        "deployment_id": deployment.ID,
        "image":         image,
        "status":        "success",
    })
}
```

After `emitEvent(EventDeployFailed, ...)` (line 99), add similar with `WebhookEventDeployFailed` and `"status": "failed"`.

### 4c. Wire in main.go

```go
webhookDeliverySvc := services.NewWebhookDeliveryService(webhookRepo)
pipeline.SetWebhookService(webhookDeliverySvc)
```

**Verify**: `GO111MODULE=on go build ./... && GO111MODULE=on go test ./...`

---

## Task 5: Soft Delete / Undelete for Apps

**Goal**: Instead of hard-deleting apps, soft-delete them and allow restore within 30 days.

### 5a. Migration 049

**Create** `internal/adapters/postgres/migrations/049_app_soft_delete.up.sql`:
```sql
ALTER TABLE apps ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_apps_deleted_at ON apps(deleted_at) WHERE deleted_at IS NOT NULL;
```

**Create** `internal/adapters/postgres/migrations/049_app_soft_delete.down.sql`:
```sql
DROP INDEX IF EXISTS idx_apps_deleted_at;
ALTER TABLE apps DROP COLUMN IF EXISTS deleted_at;
```

### 5b. Update appColumns and scanApp

**File**: `internal/adapters/postgres/postgres_app.go`

1. Add `deleted_at` to `appColumns`:
   ```
   ..., health_check_path, deleted_at, created_at, updated_at
   ```

2. In `scanApp`, add `&app.DeletedAt` between `&app.HealthCheckPath` and `&app.CreatedAt`.

3. In all SELECT queries (`ListAppsByUser`, `ListAppsByProject`, `GetApp`, `GetAppBySubdomain`), add `WHERE deleted_at IS NULL` to exclude soft-deleted apps. For `GetApp`, use:
   ```sql
   SELECT ... FROM apps WHERE id = $1 AND deleted_at IS NULL
   ```

### 5c. Soft delete method

Add to `PostgresAppRepository`:
```go
func (r *PostgresAppRepository) SoftDeleteApp(ctx context.Context, id string) error {
    tag, err := r.pool.Exec(ctx, "UPDATE apps SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL", id)
    if err != nil {
        return fmt.Errorf("failed to soft delete app: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return fmt.Errorf("app not found")
    }
    return nil
}
```

### 5d. Restore method

```go
func (r *PostgresAppRepository) RestoreApp(ctx context.Context, id string) (*entities.App, error) {
    tag, err := r.pool.Exec(ctx, "UPDATE apps SET deleted_at = NULL, updated_at = now() WHERE id = $1 AND deleted_at IS NOT NULL", id)
    if err != nil {
        return nil, fmt.Errorf("failed to restore app: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return nil, fmt.Errorf("app not found or not deleted")
    }
    // Read back using a special query that includes deleted
    row := r.pool.QueryRow(ctx, `SELECT `+appColumns+` FROM apps WHERE id = $1`, id)
    return scanApp(row.Scan)
}
```

### 5e. List deleted apps

```go
func (r *PostgresAppRepository) ListDeletedAppsByUser(ctx context.Context, userID string) ([]entities.App, error) {
    rows, err := r.pool.Query(ctx,
        `SELECT `+appColumns+` FROM apps WHERE user_id = $1 AND deleted_at IS NOT NULL ORDER BY deleted_at DESC`, userID)
    // ... standard scan loop
}
```

### 5f. Update port interface

**File**: `internal/ports/repositories.go`

Add to `AppRepository`:
```go
SoftDeleteApp(ctx context.Context, id string) error
RestoreApp(ctx context.Context, id string) (*entities.App, error)
ListDeletedAppsByUser(ctx context.Context, userID string) ([]entities.App, error)
```

### 5g. Memory adapter

**File**: `internal/adapters/memory/memory_app.go`

Implement `SoftDeleteApp` (set `DeletedAt`), `RestoreApp` (clear `DeletedAt`), `ListDeletedAppsByUser` (filter by `DeletedAt != nil`). Update existing `ListAppsByUser` and `ListAppsByProject` to skip apps where `DeletedAt != nil`.

### 5h. Handler — change Delete to soft delete + add restore

**File**: `internal/handlers/apps_v2.go`

In `Delete`:
- Replace `h.appRepo.DeleteApp(ctx, appID)` with `h.appRepo.SoftDeleteApp(ctx, appID)`
- Keep the K8s resource cleanup (Deployment, Service, IngressRoute, etc.)
- Return `{"message": "app deleted", "restore_until": "..."}` with 30-day window

Add `Restore` handler:
```go
func (h *AppHandlerV2) Restore(c *fiber.Ctx) error {
    appID := c.Params("appId")
    app, err := h.appRepo.RestoreApp(c.Context(), appID)
    if err != nil {
        return NewNotFound("app not found or not deleted")
    }
    return c.JSON(h.appToResponse(app))
}
```

Add `ListDeleted` handler:
```go
func (h *AppHandlerV2) ListDeleted(c *fiber.Ctx) error {
    userID := c.Locals("user_id")
    apps, err := h.appRepo.ListDeletedAppsByUser(c.Context(), userID.(string))
    // ... return as list
}
```

### 5i. Wire routes

**File**: `cmd/server/main.go`

```go
apps.Get("/trash", appHandlerV2.ListDeleted)
appByID.Post("/restore", appHandlerV2.Restore)
```

**Verify**: `GO111MODULE=on go build ./... && GO111MODULE=on go test ./...`

---

## Task 6: Crypto Key Rotation for Env Var Master Key

**Goal**: Allow rotating the AES-256 master key used for env var encryption without downtime.

### 6a. Support multiple key versions in EnvCrypto

**File**: `internal/crypto/env_crypto.go`

Current format: `enc:v1:<base64>` — already version-tagged!

Update `EnvCrypto` to hold multiple keys:
```go
type EnvCrypto struct {
    currentKey     []byte
    currentVersion int
    oldKeys        map[int][]byte // version → key (for decryption of old data)
}
```

Constructor:
```go
func NewEnvCrypto(currentKey []byte) *EnvCrypto {
    return &EnvCrypto{
        currentKey:     currentKey,
        currentVersion: 1,
        oldKeys:        make(map[int][]byte),
    }
}

// AddOldKey registers a previous master key for decryption.
func (c *EnvCrypto) AddOldKey(version int, key []byte) {
    c.oldKeys[version] = key
}
```

### 6b. Update Encrypt to use version tag

```go
func (c *EnvCrypto) Encrypt(userID, plaintext string) (string, error) {
    // ... same HKDF + AES-256-GCM logic ...
    return fmt.Sprintf("enc:v%d:%s", c.currentVersion, base64.StdEncoding.EncodeToString(ciphertext)), nil
}
```

### 6c. Update Decrypt to handle multiple versions

```go
func (c *EnvCrypto) Decrypt(userID, value string) (string, error) {
    if !IsEncrypted(value) {
        return value, nil
    }

    // Parse version: "enc:v1:..." → version=1, data=...
    parts := strings.SplitN(value, ":", 3)
    if len(parts) != 3 || !strings.HasPrefix(parts[1], "v") {
        return "", fmt.Errorf("invalid encrypted format")
    }
    version, err := strconv.Atoi(parts[1][1:])
    if err != nil {
        return "", fmt.Errorf("invalid version: %w", err)
    }

    // Select key
    var key []byte
    if version == c.currentVersion {
        key = c.currentKey
    } else if k, ok := c.oldKeys[version]; ok {
        key = k
    } else {
        return "", fmt.Errorf("unknown key version %d", version)
    }

    // Derive per-user key from selected master key
    userKey := deriveUserKeyFromMaster(key, userID)

    // Decrypt AES-256-GCM
    data, err := base64.StdEncoding.DecodeString(parts[2])
    // ... standard GCM decrypt using userKey ...
}
```

### 6d. Config — support old keys

**File**: `internal/config/config.go`

Add:
```go
// OldSecretsKeys is a comma-separated list of "version:hex_key" for previous master keys.
// Example: "1:aabbcc..." — version 1 used key aabbcc...
OldSecretsKeys string
```

Load from env: `ZENITH_OLD_SECRETS_KEYS`.

### 6e. Wire in main.go

After creating `envCrypto`, parse `cfg.OldSecretsKeys` and call `envCrypto.AddOldKey(version, key)` for each.

### 6f. Admin re-encryption endpoint

**Create** `internal/handlers/admin_crypto.go`:

Endpoint: `POST /api/v2/admin/crypto/rotate`

This endpoint:
1. Lists ALL env vars in the DB (across all apps)
2. For each encrypted value: decrypt with old key, re-encrypt with current key
3. Update in DB
4. Return count of re-encrypted values

```go
func (h *AdminHandler) RotateKeys(c *fiber.Ctx) error {
    // ... iterate all env vars, decrypt with old key, re-encrypt with new key ...
    return c.JSON(fiber.Map{"re_encrypted": count})
}
```

**Verify**: `GO111MODULE=on go build ./... && GO111MODULE=on go test ./...`

---

## Dependency Order

```
Task 1 (health check path) — independent, start immediately
Task 2 (environment deploys) — independent, start immediately
Task 3 (post-deploy hooks) — independent, start immediately
Task 4 (webhook dispatcher) — independent, start immediately
Task 5 (soft delete) — depends on Task 1 (shares migration numbering + appColumns changes)
Task 6 (crypto rotation) — independent, start immediately
```

**Recommended execution order**: 1 → 5 (share appColumns) → 2 → 4 → 3 → 6

After EACH task: run `GO111MODULE=on go build ./... && GO111MODULE=on go test ./...`

---

## Files Summary

| Task | New Files | Modified Files |
|------|-----------|----------------|
| 1 | 2 migrations | app.go, postgres_app.go, memory_app.go, inputs.go, k8s_resources.go, apps_v2.go |
| 2 | — | ci_deploy.go, main.go, apps_v2.go, inputs.go, postgres_app.go |
| 3 | 5 files (entity, migrations, repos, handler) | pipeline.go, main.go, ports/repositories.go |
| 4 | 1 file (webhook_delivery.go) | pipeline.go, main.go |
| 5 | 2 migrations | postgres_app.go, memory_app.go, apps_v2.go, main.go, ports/repositories.go |
| 6 | 1 handler (admin_crypto.go) | env_crypto.go, config.go, main.go |
