# Zenith V5 — Implementation Tasks

> **Source:** `docs/v5-developer-experience.md` (V5.2.0)
> **Last Updated:** 2026-03-16
> **Purpose:** Complete, ordered, code-ready task list. Follow top to bottom.
> **Convention:** Every task references exact file paths under `services/api/internal/` for backend and `apps/web/src/` for frontend.
> **Architecture:** Entity → Port → Adapter → Service → Handler → Route → Frontend
> **Module path:** `github.com/dotechhq/zenith/services/api/internal/...`

---

## How to Use This File

1. Work **phase by phase**, **task by task**, in order
2. Each task has a **checklist** — check items off as you complete them
3. Tasks marked `[DEPENDS: X.Y]` cannot start until task X.Y is done
4. After each phase, run the **verification commands** at the bottom
5. Use `lich make entity/service/api` when available — do NOT write files manually if lich generates them
6. If lich doesn't support a generator for something, write it manually following the patterns documented below

---

## Codebase Patterns Reference (Quick)

```
Entity:     String enums (not iota), embedded Timestamps, json tags, pointer receiver methods
Port:       context.Context first param, (*Entity, error) return, interface in ports/repositories.go
Postgres:   pgxpool.Pool, NewPostgresXXXRepository(pool), scanXXX() helper, dbSelectCols const
Memory:     sync.RWMutex, map[string]*Entity storage, NewMemoryXXXRepository()
Service:    Multiple repo injection, NewXXXService(...) constructor, context.Context first param
Handler:    fiber.Ctx, c.Params("id"), c.BodyParser(&req), c.Status(fiber.StatusCreated).JSON(resp)
DTO:        CreateXXXRequest/Response inline in handler, validate:"required" tags
Migration:  TEXT PRIMARY KEY, REFERENCES x(id) ON DELETE CASCADE, idx_table_field naming
Frontend:   api.ts functional methods, localStorage with SSR checks
```

---

# PHASE 1 — Foundation (Week 1-2)

> **Goal:** Project entity, Environment Variables, K8s Secret/ConfigMap sync
> **New tables:** projects, managed_services, app_env_vars
> **New endpoints:** 8

---

## 1.1 — Database Migration: Projects Table

**File:** `services/api/internal/adapters/postgres/migrations/038_projects.up.sql`
**File:** `services/api/internal/adapters/postgres/migrations/038_projects.down.sql`

- [ ] Create `038_projects.up.sql`:

```sql
CREATE TABLE projects (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    description     TEXT DEFAULT '',
    harbor_project_name TEXT,
    harbor_robot_user   TEXT,
    harbor_robot_pass   TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_user ON projects(user_id);
CREATE INDEX idx_projects_slug ON projects(slug);

ALTER TABLE apps ADD COLUMN project_id TEXT REFERENCES projects(id) ON DELETE SET NULL;
ALTER TABLE apps ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT true;
CREATE INDEX idx_apps_project ON apps(project_id);
```

- [ ] Create `038_projects.down.sql`:

```sql
DROP INDEX IF EXISTS idx_apps_project;
ALTER TABLE apps DROP COLUMN IF EXISTS is_public;
ALTER TABLE apps DROP COLUMN IF EXISTS project_id;
DROP TABLE IF EXISTS projects;
```

- [ ] Verify migration runs: `lich migration up` or manual test

---

## 1.2 — Database Migration: Managed Services Table

**File:** `services/api/internal/adapters/postgres/migrations/039_managed_services.up.sql`
**File:** `services/api/internal/adapters/postgres/migrations/039_managed_services.down.sql`

**[DEPENDS: 1.1]**

- [ ] Create `039_managed_services.up.sql`:

```sql
CREATE TABLE managed_services (
    id                TEXT PRIMARY KEY,
    project_id        TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id           TEXT NOT NULL REFERENCES users(id),
    service_type      TEXT NOT NULL,
    name              TEXT NOT NULL,
    version           TEXT NOT NULL,
    connection_url    TEXT,
    internal_host     TEXT,
    port              INTEGER,
    username          TEXT,
    password          TEXT,
    database_name     TEXT,
    k8s_namespace     TEXT,
    k8s_resource_name TEXT,
    status            TEXT NOT NULL DEFAULT 'provisioning',
    status_message    TEXT,
    storage_gb        INTEGER DEFAULT 5,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_managed_services_project ON managed_services(project_id);
CREATE INDEX idx_managed_services_user ON managed_services(user_id);
```

- [ ] Create `039_managed_services.down.sql`:

```sql
DROP TABLE IF EXISTS managed_services;
```

---

## 1.3 — Database Migration: App Environment Variables Table

**File:** `services/api/internal/adapters/postgres/migrations/040_app_env_vars.up.sql`
**File:** `services/api/internal/adapters/postgres/migrations/040_app_env_vars.down.sql`

- [ ] Create `040_app_env_vars.up.sql`:

```sql
CREATE TABLE app_env_vars (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    is_secret   BOOLEAN NOT NULL DEFAULT false,
    source      TEXT NOT NULL DEFAULT 'manual',
    source_id   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(app_id, key)
);

CREATE INDEX idx_env_vars_app ON app_env_vars(app_id);
```

- [ ] Create `040_app_env_vars.down.sql`:

```sql
DROP TABLE IF EXISTS app_env_vars;
```

---

## 1.4 — Entity: Project

**File:** `services/api/internal/entities/project.go`

- [ ] Create Project entity with these types and fields:

```go
package entities

import "time"

type ProjectStatus string

const (
    ProjectStatusActive   ProjectStatus = "active"
    ProjectStatusArchived ProjectStatus = "archived"
)

type Project struct {
    ID          string        `json:"id"`
    UserID      string        `json:"user_id"`
    Name        string        `json:"name"`
    Slug        string        `json:"slug"`
    Description string        `json:"description"`

    HarborProjectName string `json:"-"`
    HarborRobotUser   string `json:"harbor_robot_user,omitempty"`
    HarborRobotPass   string `json:"-"`

    Status    ProjectStatus `json:"status"`
    CreatedAt time.Time     `json:"created_at"`
    UpdatedAt time.Time     `json:"updated_at"`

    // Populated on read (not stored in projects table)
    Services        []App            `json:"services,omitempty"`
    ManagedServices []ManagedService `json:"managed_services,omitempty"`
}
```

- [ ] Add slug generation helper: `func GenerateSlug(name string) string` — lowercase, replace spaces with `-`, strip non-alphanumeric

---

## 1.5 — Entity: ManagedService

**File:** `services/api/internal/entities/managed_service.go`

- [ ] Create ManagedService entity:

```go
package entities

import "time"

type ServiceType string

const (
    ServiceTypePostgreSQL ServiceType = "postgresql"
    ServiceTypeRedis      ServiceType = "redis"
)

type ManagedServiceStatus string

const (
    ManagedServiceProvisioning ManagedServiceStatus = "provisioning"
    ManagedServiceReady        ManagedServiceStatus = "ready"
    ManagedServiceError        ManagedServiceStatus = "error"
    ManagedServiceDeleting     ManagedServiceStatus = "deleting"
)

type ManagedService struct {
    ID              string               `json:"id"`
    ProjectID       string               `json:"project_id"`
    UserID          string               `json:"user_id"`
    ServiceType     ServiceType          `json:"service_type"`
    Name            string               `json:"name"`
    Version         string               `json:"version"`
    ConnectionURL   string               `json:"connection_url,omitempty"`
    InternalHost    string               `json:"internal_host,omitempty"`
    Port            int                  `json:"port"`
    Username        string               `json:"-"`
    Password        string               `json:"-"`
    DatabaseName    string               `json:"database_name,omitempty"`
    K8sNamespace    string               `json:"-"`
    K8sResourceName string               `json:"-"`
    Status          ManagedServiceStatus `json:"status"`
    StatusMessage   string               `json:"status_message,omitempty"`
    StorageGB       int                  `json:"storage_gb"`
    CreatedAt       time.Time            `json:"created_at"`
    UpdatedAt       time.Time            `json:"updated_at"`
}

// DefaultPort returns the default port for the given service type
func DefaultPort(st ServiceType) int {
    switch st {
    case ServiceTypePostgreSQL:
        return 5432
    case ServiceTypeRedis:
        return 6379
    default:
        return 0
    }
}
```

---

## 1.6 — Entity: AppEnvVar

**File:** `services/api/internal/entities/env_var.go`

- [ ] Create AppEnvVar entity:

```go
package entities

import "time"

type EnvVarSource string

const (
    EnvVarSourceManual         EnvVarSource = "manual"
    EnvVarSourceManagedService EnvVarSource = "managed_service"
    EnvVarSourceServiceLink    EnvVarSource = "service_link"
    EnvVarSourceComposeImport  EnvVarSource = "compose_import"
)

type AppEnvVar struct {
    ID        string       `json:"id"`
    AppID     string       `json:"app_id"`
    Key       string       `json:"key"`
    Value     string       `json:"value,omitempty"`
    IsSecret  bool         `json:"is_secret"`
    Source    EnvVarSource  `json:"source"`
    SourceID  string       `json:"source_id,omitempty"`
    CreatedAt time.Time    `json:"created_at"`
    UpdatedAt time.Time    `json:"updated_at"`
}
```

---

## 1.7 — Port: Repository Interfaces

**File:** `services/api/internal/ports/repositories.go` (MODIFY — add to existing file)

- [ ] Add `ProjectRepository` interface:

```go
type ProjectRepository interface {
    CreateProject(ctx context.Context, project *entities.Project) error
    GetProject(ctx context.Context, id string) (*entities.Project, error)
    GetProjectBySlug(ctx context.Context, slug string) (*entities.Project, error)
    ListProjects(ctx context.Context, userID string) ([]entities.Project, error)
    UpdateProject(ctx context.Context, project *entities.Project) error
    DeleteProject(ctx context.Context, id string) error
}
```

- [ ] Add `ManagedServiceRepository` interface:

```go
type ManagedServiceRepository interface {
    CreateManagedService(ctx context.Context, svc *entities.ManagedService) error
    GetManagedService(ctx context.Context, id string) (*entities.ManagedService, error)
    ListManagedServicesByProject(ctx context.Context, projectID string) ([]entities.ManagedService, error)
    UpdateManagedServiceStatus(ctx context.Context, id string, status entities.ManagedServiceStatus, connURL, host string, port int) error
    DeleteManagedService(ctx context.Context, id string) error
}
```

- [ ] Add `EnvVarRepository` interface:

```go
type EnvVarRepository interface {
    SetEnvVar(ctx context.Context, envVar *entities.AppEnvVar) error
    GetEnvVars(ctx context.Context, appID string) ([]entities.AppEnvVar, error)
    DeleteEnvVar(ctx context.Context, id string) error
    BulkSetEnvVars(ctx context.Context, appID string, vars []entities.AppEnvVar) error
    DeleteEnvVarsBySource(ctx context.Context, appID string, source entities.EnvVarSource) error
}
```

---

## 1.8 — Postgres Adapter: Project Repository

**File:** `services/api/internal/adapters/postgres/postgres_project.go` (NEW)

- [ ] Create `PostgresProjectRepository` struct with `pool *pgxpool.Pool`
- [ ] Create constructor `NewPostgresProjectRepository(pool *pgxpool.Pool) *PostgresProjectRepository`
- [ ] Define `projectSelectCols` const with all column names
- [ ] Create `scanProject()` helper function
- [ ] Implement `CreateProject` — INSERT with uuid.New().String(), time.Now()
- [ ] Implement `GetProject` — SELECT by id
- [ ] Implement `GetProjectBySlug` — SELECT by slug
- [ ] Implement `ListProjects` — SELECT by user_id, ORDER BY created_at DESC
- [ ] Implement `UpdateProject` — UPDATE name, description, status, updated_at WHERE id
- [ ] Implement `DeleteProject` — DELETE by id (CASCADE handles related records)
- [ ] Handle unique constraint on slug — return meaningful error

---

## 1.9 — Postgres Adapter: ManagedService Repository

**File:** `services/api/internal/adapters/postgres/postgres_managed_service.go` (NEW)

- [ ] Create `PostgresManagedServiceRepository` with pool
- [ ] Constructor
- [ ] `managedServiceSelectCols` const
- [ ] `scanManagedService()` helper
- [ ] Implement all 5 methods from port interface
- [ ] `UpdateManagedServiceStatus` — UPDATE status, connection_url, internal_host, port, updated_at

---

## 1.10 — Postgres Adapter: EnvVar Repository

**File:** `services/api/internal/adapters/postgres/postgres_env_var.go` (NEW)

- [ ] Create `PostgresEnvVarRepository` with pool
- [ ] Constructor
- [ ] `envVarSelectCols` const
- [ ] `scanEnvVar()` helper
- [ ] Implement `SetEnvVar` — UPSERT (INSERT ON CONFLICT(app_id, key) DO UPDATE)
- [ ] Implement `GetEnvVars` — SELECT by app_id, ORDER BY key
- [ ] Implement `DeleteEnvVar` — DELETE by id
- [ ] Implement `BulkSetEnvVars` — transaction with multiple UPSERTs
- [ ] Implement `DeleteEnvVarsBySource` — DELETE by app_id AND source
- [ ] For `GetEnvVars`: if `is_secret=true`, mask value as `"••••••••"` (or leave it to service layer)

---

## 1.11 — Memory Adapter: Project Repository

**File:** `services/api/internal/adapters/memory/memory_project.go` (NEW)

- [ ] Create `MemoryProjectRepository` with `sync.RWMutex` and `map[string]*entities.Project`
- [ ] Constructor
- [ ] Implement all 6 methods
- [ ] Slug uniqueness check in CreateProject

---

## 1.12 — Memory Adapter: ManagedService Repository

**File:** `services/api/internal/adapters/memory/memory_managed_service.go` (NEW)

- [ ] Create `MemoryManagedServiceRepository`
- [ ] Constructor
- [ ] Implement all 5 methods

---

## 1.13 — Memory Adapter: EnvVar Repository

**File:** `services/api/internal/adapters/memory/memory_env_var.go` (NEW)

- [ ] Create `MemoryEnvVarRepository`
- [ ] Constructor
- [ ] Implement all 5 methods
- [ ] BulkSetEnvVars — iterate and call SetEnvVar internally

---

## 1.14 — Service: Project Service

**File:** `services/api/internal/services/project.go` (NEW)

- [ ] Create `ProjectService` struct with dependencies:
  - `projectRepo ports.ProjectRepository`
  - `appRepo ports.AppRepository`
  - `harborClient harborclient.Client` (or port interface)
  - `namespace string`
- [ ] Constructor `NewProjectService(...)`
- [ ] Method `CreateProject(ctx, userID, name, description string) (*entities.Project, error)`:
  - Generate slug from name
  - Create Harbor project (call harborClient)
  - Create Harbor robot account
  - Save to DB
  - Return project with Harbor credentials
- [ ] Method `GetProject(ctx, id string) (*entities.Project, error)`:
  - Get project from DB
  - Get services (apps) by project_id
  - Get managed services by project_id
  - Populate `project.Services` and `project.ManagedServices`
- [ ] Method `ListProjects(ctx, userID string) ([]entities.Project, error)`
- [ ] Method `UpdateProject(ctx, id, name, description string) (*entities.Project, error)`
- [ ] Method `DeleteProject(ctx, id string) error`:
  - Delete Harbor project
  - Delete from DB (CASCADE handles apps, managed services, env vars)

---

## 1.15 — Service: EnvVar Service

**File:** `services/api/internal/services/env_var.go` (NEW)

- [ ] Create `EnvVarService` struct with:
  - `envVarRepo ports.EnvVarRepository`
  - `k8sClient k8sclient.Client`
  - `namespace string`
- [ ] Constructor
- [ ] Method `SetEnvVar(ctx, appID, key, value string, isSecret bool) (*entities.AppEnvVar, error)`:
  - Save to DB
  - Sync to K8s (call syncToK8s)
- [ ] Method `BulkSetEnvVars(ctx, appID string, vars []entities.AppEnvVar) error`:
  - Save all to DB
  - Sync to K8s once
- [ ] Method `GetEnvVars(ctx, appID string) ([]entities.AppEnvVar, error)`:
  - Get from DB
  - Mask secret values on read (replace with `"••••••••"`)
- [ ] Method `DeleteEnvVar(ctx, id, appID string) error`:
  - Delete from DB
  - Re-sync to K8s
- [ ] Private method `syncToK8s(ctx, appID string) error`:
  - Get all env vars for app
  - Split into secrets (is_secret=true) and config (is_secret=false)
  - Create/update K8s Secret `{app-slug}-env` with secret vars
  - Create/update K8s ConfigMap `{app-slug}-config` with config vars
  - Both in customer namespace

---

## 1.16 — Handler: Project Handler

**File:** `services/api/internal/handlers/project.go` (NEW)

- [ ] Define request/response DTOs:

```go
type CreateProjectRequest struct {
    Name        string `json:"name" validate:"required"`
    Description string `json:"description"`
}

type UpdateProjectRequest struct {
    Name        *string `json:"name,omitempty"`
    Description *string `json:"description,omitempty"`
}

type ProjectResponse struct {
    ID          string                   `json:"id"`
    Name        string                   `json:"name"`
    Slug        string                   `json:"slug"`
    Description string                   `json:"description"`
    Status      string                   `json:"status"`
    HarborRobotUser string              `json:"harbor_robot_user,omitempty"`
    Services    []AppResponse            `json:"services,omitempty"`
    ManagedServices []ManagedServiceResp `json:"managed_services,omitempty"`
    CreatedAt   time.Time                `json:"created_at"`
    UpdatedAt   time.Time                `json:"updated_at"`
}
```

- [ ] Create `ProjectHandler` struct with `projectService *services.ProjectService`
- [ ] Constructor
- [ ] Handler `Create(c *fiber.Ctx) error` — POST
  - Extract userID from context (middleware sets it)
  - BodyParser
  - Validate name not empty
  - Call service.CreateProject
  - Return 201 with ProjectResponse
- [ ] Handler `List(c *fiber.Ctx) error` — GET
  - Extract userID
  - Call service.ListProjects
  - Return 200
- [ ] Handler `Get(c *fiber.Ctx) error` — GET :id
  - c.Params("id")
  - Call service.GetProject
  - Verify ownership (project.UserID == userID)
  - Return 200
- [ ] Handler `Update(c *fiber.Ctx) error` — PUT :id
  - Verify ownership
  - Call service.UpdateProject
  - Return 200
- [ ] Handler `Delete(c *fiber.Ctx) error` — DELETE :id
  - Verify ownership
  - Call service.DeleteProject
  - Return 204

---

## 1.17 — Handler: EnvVar Handler

**File:** `services/api/internal/handlers/env_var.go` (NEW)

- [ ] Define request/response DTOs:

```go
type SetEnvVarsRequest struct {
    Vars []EnvVarInput `json:"vars" validate:"required"`
}

type EnvVarInput struct {
    Key      string `json:"key" validate:"required"`
    Value    string `json:"value" validate:"required"`
    IsSecret bool   `json:"is_secret"`
}

type EnvVarResponse struct {
    ID       string `json:"id"`
    AppID    string `json:"app_id"`
    Key      string `json:"key"`
    Value    string `json:"value"`
    IsSecret bool   `json:"is_secret"`
    Source   string `json:"source"`
    CreatedAt time.Time `json:"created_at"`
}
```

- [ ] Create `EnvVarHandler` struct with `envVarService *services.EnvVarService`
- [ ] Constructor
- [ ] Handler `Set(c *fiber.Ctx) error` — POST /apps/:appId/env
  - BodyParser → SetEnvVarsRequest
  - Validate: key not empty, no duplicates in request
  - Call service.BulkSetEnvVars
  - Return 200
- [ ] Handler `List(c *fiber.Ctx) error` — GET /apps/:appId/env
  - Call service.GetEnvVars (values masked for secrets)
  - Return 200
- [ ] Handler `Delete(c *fiber.Ctx) error` — DELETE /apps/:appId/env/:varId
  - Call service.DeleteEnvVar
  - Return 204

---

## 1.18 — Route Registration

**File:** `services/api/cmd/server/main.go` (MODIFY)

- [ ] Create adapter instances (conditional like existing pattern):
  - `projectRepo = postgres.NewPostgresProjectRepository(pool)` or `memory.NewMemoryProjectRepository()`
  - Same for ManagedServiceRepository, EnvVarRepository
- [ ] Create service instances:
  - `projectService = services.NewProjectService(projectRepo, appRepo, harborClient, namespace)`
  - `envVarService = services.NewEnvVarService(envVarRepo, k8sClient, namespace)`
- [ ] Create handler instances:
  - `projectHandler = handlers.NewProjectHandler(projectService)`
  - `envVarHandler = handlers.NewEnvVarHandler(envVarService)`
- [ ] Register routes (under authenticated group):

```go
// Projects
projects := api.Group("/projects")
projects.Post("/", projectHandler.Create)
projects.Get("/", projectHandler.List)

projectByID := projects.Group("/:projectId")
projectByID.Get("/", projectHandler.Get)
projectByID.Put("/", projectHandler.Update)
projectByID.Delete("/", projectHandler.Delete)

// Env vars (under existing apps group)
appByID.Post("/env", envVarHandler.Set)
appByID.Get("/env", envVarHandler.List)
appByID.Delete("/env/:varId", envVarHandler.Delete)
```

---

## 1.19 — Update Existing App Entity

**File:** `services/api/internal/entities/app.go` (MODIFY)

- [ ] Add fields to `App` struct:

```go
ProjectID string `json:"project_id,omitempty"`
IsPublic  bool   `json:"is_public"`
```

- [ ] Update any `App` struct initialization to include `IsPublic: true` as default

---

## 1.20 — Frontend: API Client Methods

**File:** `apps/web/src/lib/api.ts` (MODIFY)

- [ ] Add TypeScript types:

```typescript
export interface Project {
  id: string;
  name: string;
  slug: string;
  description: string;
  status: string;
  harbor_robot_user?: string;
  services?: App[];
  managed_services?: ManagedService[];
  created_at: string;
  updated_at: string;
}

export interface ManagedService {
  id: string;
  project_id: string;
  service_type: string;
  name: string;
  version: string;
  status: string;
  storage_gb: number;
  created_at: string;
}

export interface AppEnvVar {
  id: string;
  app_id: string;
  key: string;
  value: string;
  is_secret: boolean;
  source: string;
  created_at: string;
}
```

- [ ] Add API methods:

```typescript
// Projects
export async function createProject(data: { name: string; description?: string }) { ... }
export async function listProjects() { ... }
export async function getProject(id: string) { ... }
export async function updateProject(id: string, data: { name?: string; description?: string }) { ... }
export async function deleteProject(id: string) { ... }

// Env vars
export async function setEnvVars(appId: string, vars: { key: string; value: string; is_secret: boolean }[]) { ... }
export async function getEnvVars(appId: string) { ... }
export async function deleteEnvVar(appId: string, varId: string) { ... }
```

---

## 1.21 — Frontend: Project List Page

**File:** `apps/web/src/app/projects/page.tsx` (NEW)

- [ ] List all user projects (cards with name, slug, service count, status)
- [ ] "New Project" button → links to `/projects/new`
- [ ] Each card links to `/projects/[id]`
- [ ] Show managed service icons (PG, Redis) on each card
- [ ] Empty state: "No projects yet. Deploy your first app →"

---

## 1.22 — Frontend: Project Dashboard Page

**File:** `apps/web/src/app/projects/[id]/page.tsx` (NEW)

- [ ] Show project name, slug, URL
- [ ] Services table (name, status, type, URL)
- [ ] Managed services section (PG, Redis with status indicators)
- [ ] Quick actions: View Logs, Env Vars, CI Setup, Domains
- [ ] Stats cards (services count, uptime, etc.)

---

## 1.23 — Frontend: Demo API Stubs

**File:** `apps/web/src/lib/demo-api.ts` (MODIFY)

- [ ] Add in-memory project storage and CRUD methods matching API interface
- [ ] Add in-memory managed service storage
- [ ] Add in-memory env var storage
- [ ] Generate demo data (1-2 sample projects)

---

### Phase 1 Verification

```bash
# Backend compiles
cd services/api && go vet ./internal/...

# Tests pass
cd services/api && go test ./internal/... -v

# Frontend compiles
cd apps/web && npx next lint --quiet
cd apps/web && npx next build

# Migration applies cleanly
# (test on local PostgreSQL or staging)

# Smoke test (manual or curl)
# POST /api/v1/projects → 201
# GET /api/v1/projects → 200, returns list
# POST /api/v1/apps/{id}/env → 200
# GET /api/v1/apps/{id}/env → 200, secrets masked
```

---

# PHASE 2 — Compose Import + Deploy (Week 3-4)

> **Goal:** Docker Compose parser, managed service provisioning, image verification, one-click deploy
> **New tables:** compose_imports
> **New endpoints:** 7

---

## 2.1 — Database Migration: Compose Imports Audit Table

**File:** `services/api/internal/adapters/postgres/migrations/041_compose_imports.up.sql`
**File:** `services/api/internal/adapters/postgres/migrations/041_compose_imports.down.sql`

- [ ] Create `041_compose_imports.up.sql`:

```sql
CREATE TABLE compose_imports (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL,
    compose_content TEXT NOT NULL,
    parsed_result   JSONB NOT NULL,
    ai_result       JSONB,
    status          TEXT NOT NULL DEFAULT 'success',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_compose_imports_project ON compose_imports(project_id);
```

- [ ] Create down migration

---

## 2.2 — Service: Docker Compose Parser

**File:** `services/api/internal/services/compose_parser.go` (NEW)

This is the CORE feature. Parse docker-compose.yml and detect everything.

- [ ] Add `gopkg.in/yaml.v3` dependency to `go.mod` (if not present)
- [ ] Define compose YAML structs (minimal — only what we need):

```go
type ComposeFile struct {
    Version  string                    `yaml:"version"`
    Services map[string]ComposeService `yaml:"services"`
}

type ComposeService struct {
    Image       string            `yaml:"image"`
    Build       interface{}       `yaml:"build"`       // string or map
    Ports       []string          `yaml:"ports"`
    Environment interface{}       `yaml:"environment"` // map or list
    DependsOn   interface{}       `yaml:"depends_on"`  // list or map
    Volumes     []string          `yaml:"volumes"`
    Command     interface{}       `yaml:"command"`
    Restart     string            `yaml:"restart"`
}
```

- [ ] Define result types:

```go
type ParsedCompose struct {
    Valid           bool                `json:"valid"`
    Services        []ParsedService     `json:"services"`
    ManagedServices []ParsedManaged     `json:"managed_services"`
    Warnings        []string            `json:"warnings"`
    Errors          []string            `json:"errors"`
}

type ParsedService struct {
    Name        string           `json:"name"`
    BuildContext string          `json:"build_context,omitempty"`
    Port        int              `json:"port"`
    IsPublic    bool             `json:"is_public"`
    URL         string           `json:"url,omitempty"`
    EnvVars     []ParsedEnvVar   `json:"env_vars"`
    DependsOn   []string         `json:"depends_on"`
}

type ParsedEnvVar struct {
    Key      string `json:"key"`
    Original string `json:"original"`
    Zenith   string `json:"zenith"`
}

type ParsedManaged struct {
    Name         string `json:"name"`
    Type         string `json:"type"`
    Version      string `json:"version"`
    DetectedFrom string `json:"detected_from"`
}
```

- [ ] Define managed image detection map:

```go
var managedImages = map[string]entities.ServiceType{
    "postgres":   entities.ServiceTypePostgreSQL,
    "postgresql": entities.ServiceTypePostgreSQL,
    "redis":      entities.ServiceTypeRedis,
    "valkey":     entities.ServiceTypeRedis,
}
```

- [ ] Implement `ParseCompose(content string, projectSlug, namespace string) (*ParsedCompose, error)`:
  1. YAML parse (Layer 1)
  2. Iterate services
  3. For each service: check if `image` matches managedImages → add to ManagedServices
  4. For each service with `build`: add to Services
  5. Detect ports → if has ports, `is_public = true`
  6. Parse environment vars → translate service URLs to K8s DNS
  7. Generate warnings for hardcoded passwords, missing health checks
  8. Generate URLs: `{service}-{project}.apps.freezenith.com`

- [ ] Implement `extractPort(ports []string) int` — parse "3000:3000" → 3000
- [ ] Implement `parseEnvironment(env interface{}) map[string]string` — handle both map and list format
- [ ] Implement `translateServiceURL(value, projectSlug, namespace string, serviceNames map[string]bool) string` — replace `http://api:8080` with K8s DNS
- [ ] Implement `detectVersion(image string) string` — extract version from "postgres:16" → "16"

---

## 2.3 — Service: Compose Validator (Layer 2)

**File:** `services/api/internal/services/compose_validator.go` (NEW)

- [ ] Implement `ValidateCompose(parsed *ParsedCompose) []string`:
  - Check: at least 1 app service (not all managed)
  - Check: no duplicate service names
  - Check: ports are valid (1-65535)
  - Check: env var references exist (if `${SERVICE}_URL` references a service that exists)
  - Warning: hardcoded passwords in env vars (regex: `password=`, `://user:pass@`)
  - Warning: no health check defined
  - Warning: privileged mode (we don't support it)
  - Return list of warning/error strings

---

## 2.4 — Service: Managed Service Provisioner

**File:** `services/api/internal/services/managed_service.go` (NEW)

- [ ] Create `ManagedServiceService` struct with:
  - `msRepo ports.ManagedServiceRepository`
  - `k8sClient k8sclient.Client`
  - `namespace string`
- [ ] Constructor
- [ ] Method `ProvisionPostgreSQL(ctx, projectID, userID, name, version string) (*entities.ManagedService, error)`:
  - Generate credentials (username, password, database_name)
  - Create CNPG Cluster CRD via k8sClient
  - Save to DB with status "provisioning"
  - Start background goroutine to poll CNPG status → update to "ready" when primary is up
  - Return managed service
- [ ] Method `ProvisionRedis(ctx, projectID, userID, name, version string) (*entities.ManagedService, error)`:
  - Generate password
  - Create StatefulSet + Service + PVC via k8sClient
  - Save to DB
  - Poll until ready
  - Return managed service
- [ ] Method `DeleteManagedService(ctx, id string) error`:
  - Get managed service from DB
  - Delete K8s resources
  - Delete from DB
- [ ] Method `ListManagedServices(ctx, projectID string) ([]entities.ManagedService, error)`
- [ ] Private helper `generateCredentials() (user, pass, dbName string)`

---

## 2.5 — Handler: Compose Import Handler

**File:** `services/api/internal/handlers/compose.go` (NEW)

- [ ] Define DTOs:

```go
type ImportComposeRequest struct {
    ComposeContent string `json:"compose_content" validate:"required"`
}

type ImportComposeResponse struct {
    Valid           bool                     `json:"valid"`
    Services        []ParsedServiceResponse  `json:"services"`
    ManagedServices []ParsedManagedResponse  `json:"managed_services"`
    Warnings        []string                 `json:"warnings"`
    AIsuggestions   []string                 `json:"ai_suggestions,omitempty"`
}
```

- [ ] Create `ComposeHandler` struct with compose parser service, project service
- [ ] Constructor
- [ ] Handler `ImportCompose(c *fiber.Ctx) error` — POST /projects/:projectId/import-compose
  - Get projectID from params
  - Verify project ownership
  - BodyParser → ImportComposeRequest
  - Call ParseCompose
  - Call ValidateCompose
  - Save to compose_imports audit table
  - If AI enabled, fire async AI validation (non-blocking — use goroutine, append results later or return separately)
  - Return 200 with ImportComposeResponse

---

## 2.6 — Handler: Managed Service Handler

**File:** `services/api/internal/handlers/managed_service.go` (NEW)

- [ ] DTOs:

```go
type ProvisionManagedServiceRequest struct {
    ServiceType string `json:"service_type" validate:"required"` // postgresql, redis
    Name        string `json:"name" validate:"required"`
    Version     string `json:"version" validate:"required"`
}

type ManagedServiceResponse struct {
    ID          string `json:"id"`
    ProjectID   string `json:"project_id"`
    ServiceType string `json:"service_type"`
    Name        string `json:"name"`
    Version     string `json:"version"`
    Status      string `json:"status"`
    StorageGB   int    `json:"storage_gb"`
    CreatedAt   string `json:"created_at"`
}
```

- [ ] Create `ManagedServiceHandler` struct
- [ ] Handler `Provision(c *fiber.Ctx) error` — POST /projects/:projectId/managed-services
- [ ] Handler `List(c *fiber.Ctx) error` — GET /projects/:projectId/managed-services
- [ ] Handler `Delete(c *fiber.Ctx) error` — DELETE /projects/:projectId/managed-services/:msId

---

## 2.7 — Handler: Image Status Handler

**File:** `services/api/internal/handlers/image_status.go` (NEW)

- [ ] Create `ImageStatusHandler` struct with `harborClient`
- [ ] Handler `GetImageStatus(c *fiber.Ctx) error` — GET /projects/:projectId/images/status
  - Get project from DB
  - Get all app services in project
  - For each service, check Harbor API: does image `{harbor_project}/{service_name}` exist?
  - Return list with pushed/not-pushed status per service
  - Return `all_ready: true` only when ALL services have pushed images

---

## 2.8 — Handler: Deploy Handler

**File:** `services/api/internal/handlers/deploy.go` (MODIFY — add project deploy)

- [ ] Add `DeployProject(c *fiber.Ctx) error` — POST /projects/:projectId/deploy
  - Get project with all services
  - Verify all images pushed (call image status check)
  - Verify all managed services ready
  - For each app service:
    - Sync env vars to K8s Secret/ConfigMap
    - Create/update K8s Deployment with image from Harbor
    - Create K8s Service
    - If is_public: create IngressRoute + Certificate
    - If free tier: create KEDA HTTPScaledObject
  - Return deployment status per service
- [ ] Add `DeploySingle(c *fiber.Ctx) error` — POST /apps/:appId/deploy
  - Deploy/redeploy a single app (for CI trigger use case)
  - Accept `{"image_tag": "sha-abc123"}` in body
  - Update deployment with new image tag

---

## 2.9 — Route Registration (Phase 2)

**File:** `services/api/cmd/server/main.go` (MODIFY)

- [ ] Wire compose handler, managed service handler, image status handler
- [ ] Register routes under projectByID group:

```go
projectByID.Post("/import-compose", composeHandler.ImportCompose)
projectByID.Post("/managed-services", msHandler.Provision)
projectByID.Get("/managed-services", msHandler.List)
projectByID.Delete("/managed-services/:msId", msHandler.Delete)
projectByID.Get("/images/status", imageStatusHandler.GetImageStatus)
projectByID.Post("/deploy", deployHandler.DeployProject)

// Single app deploy (under apps group)
appByID.Post("/deploy", deployHandler.DeploySingle)
```

---

## 2.10 — Frontend: 3-Step Onboarding Wizard

**File:** `apps/web/src/app/projects/new/page.tsx` (NEW)

- [ ] Step 1 component: Project name input + docker-compose textarea
- [ ] Step 2 component: Services review, image status, env vars editor, push instructions
- [ ] Step 3 component: Deploy progress, status indicators, success screen
- [ ] State management: useState for step, project data, compose results
- [ ] Poll image status every 5 seconds in Step 2
- [ ] Poll deploy status in Step 3

---

## 2.11 — Frontend: Compose Editor Component

**File:** `apps/web/src/components/compose/ComposeEditor.tsx` (NEW)

- [ ] Textarea with monospace font for YAML
- [ ] File upload button (read .yml file, put contents in textarea)
- [ ] Syntax error display (from Layer 1)
- [ ] "Or add services manually" link

---

## 2.12 — Frontend: Image Status Component

**File:** `apps/web/src/components/compose/ImageStatus.tsx` (NEW)

- [ ] Per-service row: name, status icon (⏳/✅), tag, push time
- [ ] Docker push commands with copy button (pre-filled with project credentials)
- [ ] Poll every 5s, stop when all_ready
- [ ] Link to CI templates page

---

### Phase 2 Verification

```bash
# Backend compiles
cd services/api && go vet ./internal/...
cd services/api && go test ./internal/... -v

# Frontend
cd apps/web && npx next build

# Integration test (on staging):
# 1. POST /projects → create project
# 2. POST /projects/{id}/import-compose with sample docker-compose
# 3. Verify response has detected services and managed
# 4. POST /projects/{id}/managed-services → provision PG
# 5. Wait for status "ready"
# 6. docker push a test image to Harbor
# 7. GET /projects/{id}/images/status → all_ready: true
# 8. POST /projects/{id}/deploy → all services running
# 9. curl app URL → 200 with SSL
```

---

# PHASE 3 — AI + Logs + CI Templates (Week 5-6)

> **Goal:** LiteLLM integration, AI compose validation, AI error analysis, logs dashboard, CI templates
> **New tables:** ai_usage
> **New endpoints:** 5

---

## 3.1 — Database Migration: AI Usage Tracking

**File:** `services/api/internal/adapters/postgres/migrations/042_ai_usage.up.sql`
**File:** `services/api/internal/adapters/postgres/migrations/042_ai_usage.down.sql`

- [ ] Create `042_ai_usage.up.sql`:

```sql
CREATE TABLE ai_usage (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    usage_type  TEXT NOT NULL,
    model       TEXT NOT NULL,
    tokens_in   INTEGER NOT NULL,
    tokens_out  INTEGER NOT NULL,
    cost_usd    DECIMAL(10,6),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_usage_user ON ai_usage(user_id);
CREATE INDEX idx_ai_usage_month ON ai_usage(user_id, created_at);
```

- [ ] Create down migration

---

## 3.2 — Service: LiteLLM Client

**File:** `services/api/internal/services/ai_client.go` (NEW)

- [ ] Create `AIClient` struct with:
  - `httpClient *http.Client`
  - `baseURL string`
  - `apiKey string`
  - `model string`
  - `enabled bool`
- [ ] Constructor `NewAIClient(url, apiKey, model string, enabled bool) *AIClient`
- [ ] Method `Complete(ctx, systemPrompt, userPrompt string) (*AIResponse, error)`:
  - If `!enabled`, return nil, nil (skip gracefully)
  - Build OpenAI-compatible request to LiteLLM
  - POST to `{baseURL}/chat/completions`
  - Parse response
  - Return AIResponse with content, tokens_in, tokens_out, model
  - Timeout: 10 seconds
  - On error: return nil, nil (graceful degradation, NEVER block)
- [ ] Type `AIResponse`:

```go
type AIResponse struct {
    Content   string `json:"content"`
    TokensIn  int    `json:"tokens_in"`
    TokensOut int    `json:"tokens_out"`
    Model     string `json:"model"`
}
```

---

## 3.3 — Service: PII Scrubber

**File:** `services/api/internal/services/pii_scrubber.go` (NEW)

- [ ] Define PII patterns (compile once at package init):

```go
var piiPatterns = []struct {
    Pattern     *regexp.Regexp
    Replacement string
}{
    {regexp.MustCompile(`\S+@\S+\.\S+`), "[EMAIL]"},
    {regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), "[IP]"},
    {regexp.MustCompile(`Bearer\s+eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`), "Bearer [TOKEN]"},
    {regexp.MustCompile(`(sk_live_|sk_test_|pk_live_|pk_test_|api_key=|apikey=)[A-Za-z0-9_-]+`), "[API_KEY]"},
    {regexp.MustCompile(`postgresql://\w+:([^@]+)@`), "postgresql://[USER]:[REDACTED]@"},
    {regexp.MustCompile(`redis://:[^@]+@`), "redis://:[REDACTED]@"},
    {regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), "[UUID]"},
}
```

- [ ] Implement `ScrubPII(log string) string` — apply all patterns
- [ ] Write tests for each pattern — `services/pii_scrubber_test.go`

---

## 3.4 — Service: AI Compose Validation (Layer 3)

**File:** `services/api/internal/services/ai_compose.go` (NEW)

- [ ] Create `AIComposeValidator` struct with `aiClient *AIClient`
- [ ] Constructor
- [ ] Method `ValidateCompose(ctx context.Context, composeContent string) ([]string, error)`:
  - Build prompt (see v5 doc for exact prompt text)
  - Call aiClient.Complete
  - Parse JSON response → extract suggestions
  - Return list of suggestion strings
  - On any error → return empty list (never block)

---

## 3.5 — Service: AI Error Analysis

**File:** `services/api/internal/services/ai_error.go` (NEW)

- [ ] Create `AIErrorAnalyzer` struct with:
  - `aiClient *AIClient`
  - `lokiClient lokiclient.Client`
  - `k8sClient k8sclient.Client`
- [ ] Constructor
- [ ] Method `AnalyzeError(ctx, appID, namespace string, logLines int) (*ErrorAnalysis, error)`:
  1. Fetch last N log lines from Loki (or kubectl fallback)
  2. Call `ScrubPII(rawLogs)`
  3. Build prompt: "You are a DevOps expert. Analyze this app error log. Return JSON: {problem, cause, fix, confidence}"
  4. Call aiClient.Complete
  5. Parse response
  6. Return ErrorAnalysis
- [ ] Type:

```go
type ErrorAnalysis struct {
    Problem       string `json:"problem"`
    Cause         string `json:"cause"`
    Fix           string `json:"fix"`
    Confidence    string `json:"confidence"`
    PIIDisclaimer string `json:"pii_disclaimer"`
}
```

---

## 3.6 — AI Usage Tracking

**File:** `services/api/internal/adapters/postgres/postgres_ai_usage.go` (NEW)

- [ ] `PostgresAIUsageRepository` — simple INSERT and COUNT per user per month
- [ ] Port interface in `ports/repositories.go`:

```go
type AIUsageRepository interface {
    RecordUsage(ctx context.Context, userID, usageType, model string, tokensIn, tokensOut int, costUSD float64) error
    GetMonthlyUsage(ctx context.Context, userID string, month time.Time) (int, error)
}
```

- [ ] Memory adapter stub

---

## 3.7 — Handler: AI Handler

**File:** `services/api/internal/handlers/ai.go` (NEW)

- [ ] Handler `AnalyzeError(c *fiber.Ctx) error` — POST /apps/:appId/ai/analyze-error
  - Check AI usage limit for user's plan
  - Call AIErrorAnalyzer.AnalyzeError
  - Record usage
  - Return ErrorAnalysis
- [ ] Handler `GetUsage(c *fiber.Ctx) error` — GET /ai/usage
  - Return monthly usage count vs plan limit

---

## 3.8 — Service: Logs Proxy

**File:** `services/api/internal/services/logs.go` (NEW)

- [ ] Create `LogsService` struct with `lokiClient lokiclient.Client`
- [ ] Method `GetLogs(ctx, appID, namespace, since string, limit int) ([]LogEntry, error)`:
  - Build Loki query scoped to app labels: `{namespace="{ns}", app="{app-slug}"}`
  - Execute query
  - Return log entries
- [ ] Type:

```go
type LogEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Line      string    `json:"line"`
    Level     string    `json:"level,omitempty"` // parsed from log line
}
```

---

## 3.9 — Handler: Logs Handler

**File:** `services/api/internal/handlers/logs.go` (MODIFY if exists, or NEW)

- [ ] Handler `GetLogs(c *fiber.Ctx) error` — GET /apps/:appId/logs
  - Query params: `since` (1h, 6h, 24h), `limit` (default 500)
  - Verify app ownership
  - Call LogsService.GetLogs
  - Return log entries
- [ ] Handler `StreamLogs(c *fiber.Ctx) error` — WebSocket /apps/:appId/logs/stream
  - Upgrade to WebSocket
  - Tail Loki query
  - Stream new log lines to client
  - Close on disconnect

---

## 3.10 — CI Templates (Static Content)

**File:** `services/api/internal/handlers/ci_templates.go` (NEW)

- [ ] Embed CI template files using `embed.FS` or define as string constants
- [ ] Templates for: go, nextjs, python, nodejs, rust
- [ ] Each template is a GitHub Actions YAML with placeholders:
  - `<your-project>` — replaced with project slug from query
  - `<your-service>` — replaced with service name
- [ ] Handler `GetTemplate(c *fiber.Ctx) error` — GET /ci-templates/:framework
  - Query params: `project` (optional, for personalization)
  - Return YAML content with Content-Type text/yaml
  - If project provided, replace placeholders with actual values

---

## 3.11 — Route Registration (Phase 3)

**File:** `services/api/cmd/server/main.go` (MODIFY)

- [ ] Wire AI client, AI services, logs service
- [ ] Register routes:

```go
appByID.Post("/ai/analyze-error", aiHandler.AnalyzeError)
api.Get("/ai/usage", aiHandler.GetUsage)
api.Get("/ci-templates/:framework", ciTemplateHandler.GetTemplate)
appByID.Get("/logs", logsHandler.GetLogs)
// WebSocket route for log streaming
app.Get("/api/v1/apps/:appId/logs/stream", logsHandler.StreamLogs)
```

---

## 3.12 — Config: AI Configuration

**File:** `services/api/internal/config/config.go` (MODIFY)

- [ ] Add AI config fields:

```go
AIEnabled    bool   `env:"AI_ENABLED" envDefault:"false"`
AILiteLLMURL string `env:"AI_LITELLM_URL" envDefault:"https://api.openai.com/v1"`
AIAPIKey     string `env:"AI_API_KEY"`
AIModel      string `env:"AI_MODEL" envDefault:"gpt-4o-mini"`
```

---

## 3.13 — Frontend: Logs Page

**File:** `apps/web/src/app/projects/[id]/logs/page.tsx` (NEW)

- [ ] Service selector dropdown
- [ ] Time range selector (1h, 6h, 24h)
- [ ] Search input
- [ ] Log entries with timestamp, level coloring (INFO=gray, ERROR=red, WARN=yellow)
- [ ] "Live" toggle — switches between REST polling and WebSocket streaming
- [ ] "Why did this crash?" button → calls AI analyze-error → shows result panel
- [ ] PII disclaimer at bottom of AI result

---

## 3.14 — Frontend: CI Templates Page

**File:** `apps/web/src/app/projects/[id]/ci/page.tsx` (NEW)

- [ ] Framework selector (Go, Next.js, Python, Node.js, Rust) with icons
- [ ] CI provider selector (GitHub Actions, GitLab CI — start with GitHub only)
- [ ] Code block with YAML template (pre-filled with project values)
- [ ] Copy button
- [ ] "Secrets to add to your CI" section with values + copy buttons
  - ZENITH_REGISTRY_USER
  - ZENITH_REGISTRY_PASS (masked, reveal button)
  - ZENITH_API_KEY (masked, reveal button)
  - ZENITH_APP_ID

---

## 3.15 — Frontend: AI Error Analysis Component

**File:** `apps/web/src/components/ai/ErrorAnalysis.tsx` (NEW)

- [ ] Button: "Why did this crash?" / "AI Analyze"
- [ ] Loading state with spinner
- [ ] Result card: Problem, Cause, Fix sections
- [ ] Confidence indicator
- [ ] PII disclaimer
- [ ] Usage counter ("3 of 5 AI analyses used this month")

---

### Phase 3 Verification

```bash
# Backend
cd services/api && go vet ./internal/...
cd services/api && go test ./internal/... -v

# PII scrubber tests specifically
cd services/api && go test ./internal/services/ -run TestScrubPII -v

# Frontend
cd apps/web && npx next build

# Integration (staging):
# 1. Set AI_ENABLED=true, AI_API_KEY=sk-...
# 2. POST /apps/{id}/ai/analyze-error → get analysis
# 3. GET /ai/usage → see usage count
# 4. GET /ci-templates/go → get YAML
# 5. GET /apps/{id}/logs?since=1h → get log entries
# 6. Connect WebSocket /apps/{id}/logs/stream → get real-time logs
```

---

# POST-LAUNCH TASKS

> These are not Phase 4. These are done AFTER all 3 phases ship and customers give feedback.

---

## P1. Content Marketing

- [ ] Record 90-second demo video (script in v5 doc, section A8)
- [ ] Upload to YouTube, get embed URL
- [ ] Replace `PLACEHOLDER_VIDEO_ID` in `docs/v5-developer-experience.md` section A8
- [ ] Embed video on landing page (`apps/landing/`)
- [ ] Write blog post: "From docker-compose to production in 2 minutes"
- [ ] Create LinkedIn post with demo GIF
- [ ] Add competitor comparison table to landing page pricing section

---

## P2. Monitoring & Analytics

- [ ] Add analytics events for funnel tracking:
  - `project_created`
  - `compose_imported`
  - `image_pushed`
  - `deploy_triggered`
  - `deploy_succeeded`
  - `ai_error_analyzed`
- [ ] Dashboard in Mission Control for funnel visualization
- [ ] Alert if conversion drops below threshold

---

## P3. Deferred Features (Build When Customers Ask)

| Feature | Trigger to Build | Estimated Effort |
|---------|-----------------|------------------|
| Auth modes (BYOP, Managed Keycloak) | 3+ customers ask | 2 weeks |
| External registry (Docker Hub, GHCR) | 3+ customers ask | 1 week |
| MongoDB managed | 2+ customers ask | 1 week |
| RabbitMQ managed | 2+ customers ask | 1 week |
| GitLab CI templates | 5+ customers ask | 2 days |
| Preview environments (per-PR) | 5+ customers ask | 2 weeks |
| `docker-compose up` equivalent CLI | 10+ customers ask | 3 weeks |

---

# TASK SUMMARY

```
Phase 1: 23 tasks (Foundation)
  Migrations:    3 (projects, managed_services, env_vars)
  Entities:      3 (project, managed_service, env_var)
  Ports:         1 (3 interfaces)
  Adapters:      6 (3 postgres + 3 memory)
  Services:      2 (project, env_var)
  Handlers:      2 (project, env_var)
  Wiring:        1 (main.go)
  App entity:    1 (add project_id, is_public)
  Frontend:      4 (api.ts, projects page, dashboard, demo stubs)

Phase 2: 12 tasks (Compose + Deploy)
  Migrations:    1 (compose_imports)
  Services:      3 (compose_parser, compose_validator, managed_service)
  Handlers:      4 (compose, managed_service, image_status, deploy)
  Wiring:        1 (main.go)
  Frontend:      3 (wizard page, compose editor, image status)

Phase 3: 15 tasks (AI + Logs + CI)
  Migrations:    1 (ai_usage)
  Services:      5 (ai_client, pii_scrubber, ai_compose, ai_error, logs)
  Adapters:      1 (ai_usage postgres + memory)
  Handlers:      3 (ai, logs, ci_templates)
  Config:        1 (AI config)
  Wiring:        1 (main.go)
  Frontend:      3 (logs page, CI page, AI component)

Total: 50 tasks across 3 phases (~6 weeks)

New files:    ~40
Modified:     ~8
New endpoints: 20
New DB tables: 4
```
