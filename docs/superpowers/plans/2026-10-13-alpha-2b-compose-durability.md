# Alpha 2b — Compose Durability and Operations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Compose install survives reboot, upgrade and disk loss. Compose services reach each other by service name, and managed databases are reachable over SSH.

**Architecture:**
- **API (Go, `services/api`):**
  - the control plane moves to its own Docker networks;
  - each project gets a network carrying compose service-name aliases;
  - databases publish a loopback-only port;
  - a startup reconciler restores apps and databases from their recorded state.
- **CLI (Go, `cli/`, the released `zen`):** a new `composeops` package drives `docker compose` on the server for backup, restore and upgrade. The existing Helm/CNPG paths stay untouched.
- **Releases:** carry a version from a `VERSION` file.

**Tech Stack:** Go 1.25, Docker Engine API (`github.com/docker/docker/client`), Fiber, pgx, cobra, bash (installer), GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-10-12-alpha-2b-compose-durability-design.md`

## Global Constraints

- Only ports 80 and 443 may be public. Any new host port binds to `127.0.0.1`.
- SaaS code is not changed. The Helm `zen upgrade` and CNPG `zen backup create` paths keep working unchanged when no Compose install is detected.
- All Go tests must pass with `-race`. CI runs `go test ./... -race -count=1`.
- **Commits:**
  - conventional-commit subjects, lowercase after the type (commitlint `subject-case`);
  - author `Babak Doraniarab <babak.dorani@gmail.com>` via `git -c user.name=... -c user.email=...`;
  - message body ends with exactly these two trailer lines, with a blank line before them:
    `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
    `Claude-Session: https://claude.ai/code/session_01T4AidohHj3qvUYek6RLV3F`
- Network names: apps network `zenith_default` (compose key `default`), control plane `zenith_internal`, edge `zenith_edge`, project networks `zenith-proj-<first 12 chars of sanitized project id>`.
- Database container names stay `zenith-db-<id8>`; app containers stay `zenith-app-<subdomain>`.
- Secret keys copied by restore, exactly: `JWT_SECRET SECRETS_ENCRYPTION_KEY OLD_SECRETS_KEYS DB_PASSWORD S3_ACCESS_KEY S3_SECRET_KEY ADMIN_EMAIL ADMIN_PASSWORD`.
- Upgrade health check: `http://127.0.0.1:8080/health` returns 200, and `http://127.0.0.1:8080/api/v1/version` reports the target version, within 180 s.

## Review Focus

1. **A compose stack whose own service is named `postgres` or `api`.** Its sibling `web` must reach the stack's service, never Zenith's control plane. Pinned by Task 1's compose network test and Task 11's live check.
2. **An app deployed before 2b** (no `service_name`, name `<slug>-<svc>`) must still get a project alias on redeploy. Pinned by `TestProjectAlias_PrefixFallback`, Task 3.
3. **Restore onto a server whose `DB_PASSWORD` differs** from the archive's: the API must connect afterwards. Pinned by `TestRestore_ResetsControlPlanePassword`, Task 9.
4. **Upgrade where the new API never becomes healthy:** the install must end on the old version with the old ref checked out. Pinned by `TestUpgrade_RollsBackOnHealthFailure`, Task 10.
5. **A database whose container vanished while its row says `ready`:** the reconciler re-provisions it with its existing password and volume, and app reconciliation still runs if one database fails. Pinned by `TestReconciler_DBFailureDoesNotStopApps`, Task 5.

---

### Task 1: Control-plane network split

**Files:**
- Modify: `docker-compose.yml` (services `postgres`, `storage`, `api`, `web`, `traefik`; the `networks:` block at the end)
- Create: `infra/scripts/compose_networks_test.sh`

**Interfaces:**
- Produces: networks `zenith_default` (apps), `zenith_internal`, `zenith_edge`. Later tasks keep using `ZENITH_APP_NETWORK=zenith_default`.

- [ ] **Step 1: Write the failing test** — `infra/scripts/compose_networks_test.sh`:

```bash
#!/usr/bin/env bash
# Checks the compose file keeps the control plane off the app network.
set -euo pipefail
repo="$(cd "$(dirname "$0")/../.." && pwd)"
fail() { echo "FAIL: $*" >&2; exit 1; }
cfg="$(cd "$repo" && JWT_SECRET=x ZENITH_DOMAIN=example.com docker compose -f docker-compose.yml --profile tls config --format json)"
nets() { echo "$cfg" | python3 -c "import sys,json;s=json.load(sys.stdin)['services']['$1'];print(' '.join(sorted((s.get('networks') or {'default':None}).keys())))"; }
[ "$(nets postgres)" = "internal" ] || fail "postgres networks: $(nets postgres)"
[ "$(nets storage)" = "internal" ] || fail "storage networks: $(nets storage)"
[ "$(nets api)" = "edge internal" ] || fail "api networks: $(nets api)"
[ "$(nets web)" = "edge" ] || fail "web networks: $(nets web)"
[ "$(nets traefik)" = "default edge" ] || fail "traefik networks: $(nets traefik)"
name() { echo "$cfg" | python3 -c "import sys,json;print(json.load(sys.stdin)['networks']['$1']['name'])"; }
[ "$(name default)" = "zenith_default" ] || fail "default network name"
[ "$(name internal)" = "zenith_internal" ] || fail "internal network name"
[ "$(name edge)" = "zenith_edge" ] || fail "edge network name"
echo "$cfg" | grep -q 'traefik.docker.network=zenith_edge' || echo "$cfg" | python3 -c "import sys,json;l=json.load(sys.stdin)['services']['api']['labels'];assert l.get('traefik.docker.network')=='zenith_edge'" || fail "api router network label"
echo "compose_networks_test: ok"
```

- [ ] **Step 2: Run it to verify it fails**

Run: `bash infra/scripts/compose_networks_test.sh`
Expected: `FAIL: postgres networks: default`

- [ ] **Step 3: Implement.** Edit `docker-compose.yml`:
  - **`postgres` and `storage`:** add `networks: [internal]`.
  - **`api`:** add `networks: [internal, edge]` and the label `"traefik.docker.network=zenith_edge"`.
  - **`web`:** add `networks: [edge]` and the label `"traefik.docker.network=zenith_edge"`.
  - **`traefik`:** add `networks: [default, edge]`.
  - **`networks:` block at the end:** replace it with:

```yaml
networks:
  # User apps, managed databases and Traefik. Key and name are unchanged from
  # earlier versions so existing installs keep their app network.
  default:
    name: zenith_default
  # Control plane only: apps can't resolve or reach postgres/storage/api.
  internal:
    name: zenith_internal
  # Traefik <-> dashboard/API routing.
  edge:
    name: zenith_edge
```

  Add one comment line above the `api` networks explaining that the API needs no access to the app network: it drives apps through the Docker socket.

- [ ] **Step 4: Run test to verify it passes**

Run: `bash infra/scripts/compose_networks_test.sh`
Expected: `compose_networks_test: ok`

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml infra/scripts/compose_networks_test.sh
git commit -F msg.txt   # subject: "fix(compose): move the control plane off the app network"
```

---

### Task 2: Schema — `apps.service_name` and `user_databases.host_port`

**Files:**
- Create: `services/api/internal/adapters/postgres/migrations/051_service_name_host_port.up.sql`, `.down.sql`
- Modify:
  - `services/api/internal/entities/app.go` (App struct)
  - `services/api/internal/entities/database.go` (UserDatabase struct)
  - `services/api/internal/dto/` (the file with `CreateAppInput` and `DatabaseInfo`)
  - `services/api/internal/adapters/postgres/postgres_app.go`: INSERT at :155, `scanApp` :202, `appColumns` :244
  - `services/api/internal/adapters/postgres/postgres_database.go`
  - `services/api/internal/adapters/memory/` app and database repos
  - `services/api/internal/ports/repositories.go` (DatabaseRepository)
  - `services/api/internal/handlers/databases_v2.go` (`toDatabaseInfoV2`)
  - `apps/web/src/lib/api.ts`: the app-create input type
  - `apps/web/src/app/projects/new/page.tsx:530-546`
- Test: `services/api/internal/adapters/memory/` (new `service_name_test.go`), `services/api/internal/adapters/postgres/integration_test.go` (extend)

**Interfaces:**
- Produces:
  - `entities.App.ServiceName string` (json `service_name,omitempty`);
  - `dto.CreateAppInput.ServiceName string` (json `service_name`);
  - `entities.UserDatabase.HostPort int` (json `host_port`) and `dto.DatabaseInfo.HostPort int` (json `host_port,omitempty`);
  - repo methods `SetDatabaseHostPort(ctx context.Context, id string, port int) error` and `ListDatabasesByStatus(ctx context.Context, status entities.DatabaseStatus) ([]entities.UserDatabase, error)` on `ports.DatabaseRepository`, implemented in memory and postgres.

- [ ] **Step 1: Write the failing tests** — `services/api/internal/adapters/memory/service_name_test.go`:

```go
package memory_test

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestCreateApp_KeepsServiceName(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	a, err := repo.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "u1", Name: "shop-db", ServiceName: "db", ImageURL: "postgres:16", DeploySource: entities.DeploySourceImage})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetApp(context.Background(), a.ID)
	if got.ServiceName != "db" {
		t.Fatalf("ServiceName = %q, want db", got.ServiceName)
	}
}

func TestDatabaseHostPortAndStatusList(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMemoryDatabaseRepository()
	db, err := repo.CreateDatabase(ctx, "", "u1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "pg"})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetDatabaseHostPort(ctx, db.ID, 49153); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetDatabase(ctx, db.ID)
	if got.HostPort != 49153 {
		t.Fatalf("HostPort = %d", got.HostPort)
	}
	ready, _ := repo.ListDatabasesByStatus(ctx, entities.DatabaseStatusReady)
	if len(ready) != 1 || ready[0].ID != db.ID {
		t.Fatalf("ready = %+v", ready)
	}
}
```

Extend `adapters/postgres/integration_test.go` in its existing style (it uses testcontainers):
- an app created with `ServiceName: "db"` reads back `"db"`;
- `SetDatabaseHostPort`, then `GetDatabase` returns the port;
- `ListDatabasesByStatus(ready)` returns only ready rows.

- [ ] **Step 2: Run to verify failure**

Run: `cd services/api && go test ./internal/adapters/memory -run 'ServiceName|HostPort' -race`
Expected: compile error — `unknown field ServiceName`.

- [ ] **Step 3: Implement**

Migration `051_service_name_host_port.up.sql`:

```sql
ALTER TABLE apps ADD COLUMN IF NOT EXISTS service_name TEXT NOT NULL DEFAULT '';
ALTER TABLE user_databases ADD COLUMN IF NOT EXISTS host_port INTEGER NOT NULL DEFAULT 0;
```

`.down.sql`:

```sql
ALTER TABLE apps DROP COLUMN IF EXISTS service_name;
ALTER TABLE user_databases DROP COLUMN IF EXISTS host_port;
```

**Entity, DTO and memory repos:**
- Add the fields named in Interfaces, and the comment `// ServiceName is the compose service this app was created from ("" if none).`
- Memory app repo `CreateApp` copies `input.ServiceName`.
- Memory database repo:
  - `SetDatabaseHostPort` mirrors `SetDatabasePort`: it locks, returns not-found if missing, and sets `HostPort`;
  - `ListDatabasesByStatus` copies rows whose `Status` matches.

**Postgres app repo:**
- add `service_name` to the INSERT column list and values;
- append `service_name` to `appColumns` immediately before `deleted_at`;
- scan it into `&app.ServiceName` at the matching position in `scanApp`.

**Postgres database repo:**
- add `host_port` to its select column list and scan;
- `SetDatabaseHostPort` mirrors `SetDatabasePort` (`UPDATE user_databases SET host_port = $2, updated_at = NOW() WHERE id = $1`);
- `ListDatabasesByStatus` runs `SELECT <cols> FROM user_databases WHERE status = $1 ORDER BY created_at`.

Fix every other implementation of `ports.DatabaseRepository`: mocks in handler tests must compile. Run `go build ./... && go vet ./...` to find them.

**`toDatabaseInfoV2`:** sets `HostPort: db.HostPort`.

**Web:**
- add `service_name?: string` to the create-app input type in `apps/web/src/lib/api.ts`;
- in `projects/new/page.tsx`, the `api.appsDeploy.create({...})` call adds `service_name: svc.name,`.

- [ ] **Step 4: Run tests**

Run:
- `cd services/api && go build ./... && go test ./internal/adapters/... ./internal/handlers/... -race -count=1`
- `cd apps/web && npx vitest run`

Expected: PASS. The postgres integration test needs Docker, which this host has.

- [ ] **Step 5: Commit** — subject `feat(api): store compose service name on apps and database host port`

---

### Task 3: Per-project networks in `DockerDeployer`

**Files:**
- Create: `services/api/internal/services/deploy/project_network.go`
- Modify:
  - `services/api/internal/services/deploy/docker_deployer.go`: struct, `DeployApp` :84-160, `DeleteApp`
  - `services/api/cmd/server/main.go` (~line 541, after `NewDockerDeployer`)
- Test: `services/api/internal/services/deploy/project_network_test.go`

**Interfaces:**
- Consumes: `entities.App.ServiceName` (Task 2); `ports.ProjectRepository.GetProject(ctx, id) (*entities.Project, error)`, whose `Project.Slug` is used.
- Produces:
  - `func projectNetworkName(projectID string) string`;
  - `func projectAlias(app *entities.App, projectSlug string) string`;
  - `func (d *DockerDeployer) SetProjectRepo(r ports.ProjectRepository)`;
  - container label `zenith.project.id` on app containers.

- [ ] **Step 1: Write the failing tests** — `project_network_test.go`:

```go
package deploy

import (
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestProjectNetworkName(t *testing.T) {
	if got := projectNetworkName("7f3e2a10-9b1c-4d2e-8f00-123456789abc"); got != "zenith-proj-7f3e2a10-9b1" {
		t.Fatalf("got %q", got)
	}
	if got := projectNetworkName(""); got != "" {
		t.Fatalf("empty project id gave %q", got)
	}
}

func TestProjectAlias_ServiceNameWins(t *testing.T) {
	app := &entities.App{Name: "shop-db", ServiceName: "DB"}
	if got := projectAlias(app, "shop"); got != "db" {
		t.Fatalf("got %q", got)
	}
}

func TestProjectAlias_PrefixFallback(t *testing.T) {
	app := &entities.App{Name: "shop-db"}
	if got := projectAlias(app, "shop"); got != "db" {
		t.Fatalf("got %q", got)
	}
}

func TestProjectAlias_None(t *testing.T) {
	for _, tc := range []struct {
		app  *entities.App
		slug string
	}{
		{&entities.App{Name: "standalone"}, "shop"},
		{&entities.App{Name: "shop-"}, "shop"},
		{&entities.App{Name: "shop-db"}, ""},
	} {
		if got := projectAlias(tc.app, tc.slug); got != "" {
			t.Errorf("app %q slug %q: got %q, want none", tc.app.Name, tc.slug, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd services/api && go test ./internal/services/deploy -run 'ProjectNetwork|ProjectAlias' -race`
Expected: compile error, `undefined: projectNetworkName`.

- [ ] **Step 3: Implement** — `project_network.go`:

```go
package deploy

import (
	"context"
	"log/slog"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

const projectLabel = "zenith.project.id"

// projectNetworkName is the per-project bridge network. Only that project's
// app containers join it, so compose service names (db, api, postgres...)
// resolve to the project's own services.
func projectNetworkName(projectID string) string {
	id := sanitizeName(projectID)
	if id == "" {
		return ""
	}
	if len(id) > 12 {
		id = id[:12]
	}
	return "zenith-proj-" + id
}

// projectAlias is the name siblings in the project use for this app: its
// compose service name, else (apps created before service_name existed) the
// app name without the "<project slug>-" prefix the compose flow adds.
func projectAlias(app *entities.App, projectSlug string) string {
	if s := sanitizeName(app.ServiceName); s != "" {
		return s
	}
	if projectSlug == "" {
		return ""
	}
	rest, ok := strings.CutPrefix(sanitizeName(app.Name), sanitizeName(projectSlug)+"-")
	if !ok {
		return ""
	}
	return rest
}

// SetProjectRepo lets the deployer read project slugs for the alias fallback.
func (d *DockerDeployer) SetProjectRepo(r ports.ProjectRepository) { d.projectRepo = r }

func (d *DockerDeployer) projectSlug(ctx context.Context, app *entities.App) string {
	if d.projectRepo == nil || app.ProjectID == "" {
		return ""
	}
	p, err := d.projectRepo.GetProject(ctx, app.ProjectID)
	if err != nil || p == nil {
		return ""
	}
	return p.Slug
}

// ensureProjectNetwork creates the project network if it doesn't exist.
func (d *DockerDeployer) ensureProjectNetwork(ctx context.Context, app *entities.App) (string, error) {
	name := projectNetworkName(app.ProjectID)
	if name == "" {
		return "", nil
	}
	if _, err := d.cli.NetworkInspect(ctx, name, network.InspectOptions{}); err == nil {
		return name, nil
	} else if !client.IsErrNotFound(err) {
		return "", err
	}
	_, err := d.cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
		Labels: map[string]string{projectLabel: app.ProjectID},
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return "", err
	}
	return name, nil
}

// removeProjectNetworkIfUnused drops the project network once no app
// container of the project is left. Failures are logged, never returned.
func (d *DockerDeployer) removeProjectNetworkIfUnused(ctx context.Context, projectID string) {
	name := projectNetworkName(projectID)
	if name == "" {
		return
	}
	left, err := d.cli.ContainerList(ctx, container.ListOptions{All: true,
		Filters: filters.NewArgs(filters.Arg("label", projectLabel+"="+projectID))})
	if err != nil || len(left) > 0 {
		return
	}
	if err := d.cli.NetworkRemove(ctx, name); err != nil && !client.IsErrNotFound(err) {
		slog.Warn("docker-deploy: remove project network failed", "network", name, "error", err)
	}
}
```

If the vendored Docker client predates `network.InspectOptions` / `network.CreateOptions`, use the type names that version's `client.NetworkAPIClient` declares (`types.NetworkInspectOptions` / `types.NetworkCreate`). Check `go doc github.com/docker/docker/client.NetworkAPIClient`.

In `docker_deployer.go`:
- **Struct:** add `projectRepo ports.ProjectRepository`.
- **`DeployApp`:**
  - after computing `labels`, set `labels[projectLabel] = app.ProjectID` when it isn't empty;
  - before `ContainerCreate`, call `projNet, err := d.ensureProjectNetwork(ctx, app)` and return wrapped errors;
  - after `ContainerCreate` and **before** `ContainerStart`, when `projNet != ""`, connect the container to the project network:

```go
	if projNet != "" {
		var aliases []string
		if a := projectAlias(app, d.projectSlug(ctx, app)); a != "" {
			aliases = []string{a}
		}
		if err := d.cli.NetworkConnect(ctx, projNet, resp.ID, &network.EndpointSettings{Aliases: aliases}); err != nil {
			return fmt.Errorf("docker-deploy: join project network %s: %w", projNet, err)
		}
	}
```

- **`DeleteApp`:** after a successful remove, or a not-found, call `d.removeProjectNetworkIfUnused(ctx, app.ProjectID)`.
- **`main.go`:** after `dockerDeployer = deploy.NewDockerDeployer(...)`, call `dockerDeployer.SetProjectRepo(projectRepo)`. `projectRepo` is declared at :442, before this point.

- [ ] **Step 4: Run tests**

Run: `cd services/api && go vet ./internal/services/deploy && go test ./internal/services/deploy -race -count=1`
Expected: PASS

- [ ] **Step 5: Commit** — subject `feat(api): per-project docker network with compose service-name aliases (F-21)`

---

### Task 4: Loopback host port for managed databases

**Files:**
- Modify:
  - `services/api/internal/services/deploy/docker_db.go`: `Provision` :73-111; new `HostPort` method
  - `services/api/internal/services/db_runtime.go`: `DBBackend` comment, `provision` :94-108
- Test:
  - `services/api/internal/services/deploy/docker_db_test.go` (create if absent; package `deploy`)
  - `services/api/internal/services/db_runtime_test.go` (existing tests live in `services`)

**Interfaces:**
- Consumes: `ports.DatabaseRepository.SetDatabaseHostPort` (Task 2).
- Produces:
  - `func loopbackBinding(port int) (nat.PortSet, nat.PortMap)` in `deploy`;
  - optional interface in `services`: `type HostPorter interface { HostPort(ctx context.Context, db *entities.UserDatabase) (int, error) }`;
  - `(*DockerDBBackend).HostPort` implements it.

- [ ] **Step 1: Write the failing tests**

`docker_db_test.go`:

```go
package deploy

import "testing"

func TestLoopbackBinding(t *testing.T) {
	exposed, bindings := loopbackBinding(5432)
	if _, ok := exposed["5432/tcp"]; !ok {
		t.Fatalf("exposed = %v", exposed)
	}
	b := bindings["5432/tcp"]
	if len(b) != 1 || b[0].HostIP != "127.0.0.1" || b[0].HostPort != "" {
		t.Fatalf("binding = %+v, want 127.0.0.1 with a docker-picked port", b)
	}
}
```

In `db_runtime_test.go`, add a backend fake that also implements `HostPorter`, returning 49200. Then assert the stored row:

```go
type hostPortBackend struct{ okBackend } // use the existing fake backend type name in this file

func (hostPortBackend) HostPort(context.Context, *entities.UserDatabase) (int, error) { return 49200, nil }

func TestProvision_StoresHostPort(t *testing.T) {
	repo := memory.NewMemoryDatabaseRepository()
	rt := services.NewDBRuntimeService(repo, hostPortBackend{}, nil)
	db, err := rt.Create(context.Background(), "", "u1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "pg"})
	if err != nil {
		t.Fatal(err)
	}
	rt.Wait(db.ID)
	got, _ := repo.GetDatabase(context.Background(), db.ID)
	if got.HostPort != 49200 {
		t.Fatalf("HostPort = %d", got.HostPort)
	}
}
```

The existing ok-backend fake in `db_runtime_test.go` may be named differently. Embed whichever fake there returns success, and keep its name.

- [ ] **Step 2: Run to verify failure**

Run: `cd services/api && go test ./internal/services/... -run 'Loopback|StoresHostPort' -race`
Expected: compile errors (`undefined: loopbackBinding`).

- [ ] **Step 3: Implement**

In `docker_db.go` (import `github.com/docker/go-connections/nat`, already a dependency of the Docker client):

```go
// loopbackBinding publishes the engine port on 127.0.0.1 only, with a port
// Docker picks. Loopback isn't reachable from the internet; it is reachable
// through an SSH tunnel (zen db connect --tunnel).
func loopbackBinding(port int) (nat.PortSet, nat.PortMap) {
	p := nat.Port(fmt.Sprintf("%d/tcp", port))
	return nat.PortSet{p: struct{}{}}, nat.PortMap{p: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: ""}}}
}

// HostPort is the loopback port Docker assigned to the engine port.
func (b *DockerDBBackend) HostPort(ctx context.Context, db *entities.UserDatabase) (int, error) {
	spec, err := specFor(db, "")
	if err != nil {
		return 0, err
	}
	insp, err := b.cli.ContainerInspect(ctx, dbContainerName(db))
	if err != nil {
		return 0, err
	}
	if insp.NetworkSettings == nil {
		return 0, fmt.Errorf("no network settings")
	}
	for _, pb := range insp.NetworkSettings.Ports[nat.Port(fmt.Sprintf("%d/tcp", spec.port))] {
		if pb.HostIP == "127.0.0.1" {
			return strconv.Atoi(pb.HostPort)
		}
	}
	return 0, fmt.Errorf("%s has no loopback port", dbContainerName(db))
}
```

In `Provision`:
- call `exposed, bindings := loopbackBinding(spec.port)`;
- set `ExposedPorts: exposed` on `container.Config` and `PortBindings: bindings` on `container.HostConfig`;
- update the type comment: "Data lives in a named volume; the engine port is published on 127.0.0.1 only."

In `db_runtime.go`:
- define `HostPorter` (doc comment: optional `DBBackend` capability: the host-side port for SSH access);
- at the end of `provision`, after `SetDatabaseProvisioned` succeeds, run:

```go
	if hp, ok := s.backend.(HostPorter); ok {
		if port, err := hp.HostPort(ctx, db); err != nil {
			slog.Warn("database host port unknown", "db", db.Name, "error", err)
		} else if err := s.repo.SetDatabaseHostPort(ctx, db.ID, port); err != nil {
			slog.Warn("store database host port failed", "db", db.Name, "error", err)
		}
	}
	return nil
```

Restructure the final `return s.repo.SetDatabaseProvisioned(...)` into an `if err := ...; err != nil { return err }`.

- [ ] **Step 4: Run tests**

Run: `cd services/api && go test ./internal/services/... ./internal/handlers/... -race -count=1`
Expected: PASS

- [ ] **Step 5: Commit** — subject `feat(api): publish managed databases on a loopback port for ssh access`

---

### Task 5: Startup reconciler

**Files:**
- Create: `services/api/internal/services/deploy/reconciler.go`
- Modify:
  - `services/api/internal/services/db_runtime.go` (new `Reprovision`)
  - `services/api/internal/services/deploy/docker_deployer.go` (new `AppContainerRunning`)
  - `services/api/internal/services/deploy/docker_db.go` (new `ContainerRunning`)
  - `services/api/cmd/server/main.go`: build one `DBRuntimeService` at :898 and reuse it; start the reconciler
- Test: `services/api/internal/services/deploy/reconciler_test.go`, `services/api/internal/services/db_runtime_test.go`

**Interfaces:**
- Consumes:
  - `ports.DatabaseRepository.ListDatabasesByStatus` (Task 2);
  - `deploy.CurrentImage(ctx, repo, appID) (string, error)`;
  - `ports.AppRepository.ListAllApps(ctx) ([]entities.App, error)`;
  - `services.HostPorter` (Task 4).
- Produces:
  - `func (s *DBRuntimeService) Reprovision(ctx context.Context, db *entities.UserDatabase) error`;
  - `type AppRuntime interface { AppContainerRunning(ctx context.Context, app *entities.App) (bool, error); DeployApp(ctx context.Context, app *entities.App, image string) error }`;
  - `type DBProber interface { ContainerRunning(ctx context.Context, db *entities.UserDatabase) (bool, error) }`;
  - `type DBReprovisioner interface { Reprovision(ctx context.Context, db *entities.UserDatabase) error }`;
  - `func NewReconciler(apps ports.AppRepository, dbs ports.DatabaseRepository, appRT AppRuntime, dbProbe DBProber, dbRT DBReprovisioner) *Reconciler`;
  - `func (r *Reconciler) Run(ctx context.Context)`.

- [ ] **Step 1: Write the failing tests**

`reconciler_test.go`:

```go
package deploy

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type fakeAppRT struct {
	mu       sync.Mutex
	running  map[string]bool
	deployed []string
}

func (f *fakeAppRT) AppContainerRunning(_ context.Context, a *entities.App) (bool, error) {
	return f.running[a.ID], nil
}
func (f *fakeAppRT) DeployApp(_ context.Context, a *entities.App, image string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deployed = append(f.deployed, a.ID+"@"+image)
	return nil
}

type fakeDBProbe struct{ running map[string]bool }

func (f fakeDBProbe) ContainerRunning(_ context.Context, db *entities.UserDatabase) (bool, error) {
	return f.running[db.ID], nil
}

type fakeDBRT struct {
	fail  map[string]bool
	calls []string
}

func (f *fakeDBRT) Reprovision(_ context.Context, db *entities.UserDatabase) error {
	f.calls = append(f.calls, db.ID)
	if f.fail[db.ID] {
		return errors.New("boom")
	}
	return nil
}

func seedApp(t *testing.T, repo *memory.MemoryAppRepository, name string, status entities.AppStatus, image string) *entities.App {
	t.Helper()
	ctx := context.Background()
	a, err := repo.CreateApp(ctx, &dto.CreateAppInput{UserID: "u1", Name: name, ImageURL: image, DeploySource: entities.DeploySourceImage})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateApp(ctx, a.ID, &dto.UpdateAppInput{Status: &status}); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestReconciler_RedeploysOnlyStoppedRunningApps(t *testing.T) {
	apps := memory.NewMemoryAppRepository()
	up := seedApp(t, apps, "up", entities.AppStatusRunning, "nginx:1")
	down := seedApp(t, apps, "down", entities.AppStatusRunning, "nginx:2")
	_ = seedApp(t, apps, "stopped", entities.AppStatusStopped, "nginx:3")
	rt := &fakeAppRT{running: map[string]bool{up.ID: true}}
	NewReconciler(apps, memory.NewMemoryDatabaseRepository(), rt, fakeDBProbe{}, &fakeDBRT{}).Run(context.Background())
	if len(rt.deployed) != 1 || rt.deployed[0] != down.ID+"@nginx:2" {
		t.Fatalf("deployed = %v, want only %s@nginx:2", rt.deployed, down.ID)
	}
}

func TestReconciler_DBFailureDoesNotStopApps(t *testing.T) {
	ctx := context.Background()
	apps := memory.NewMemoryAppRepository()
	down := seedApp(t, apps, "down", entities.AppStatusRunning, "nginx:2")
	dbs := memory.NewMemoryDatabaseRepository()
	a, _ := dbs.CreateDatabase(ctx, "", "u1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "a"})
	b, _ := dbs.CreateDatabase(ctx, "", "u1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEngineRedis, Name: "b"})
	dbRT := &fakeDBRT{fail: map[string]bool{a.ID: true}}
	appRT := &fakeAppRT{running: map[string]bool{}}
	NewReconciler(apps, dbs, appRT, fakeDBProbe{running: map[string]bool{}}, dbRT).Run(ctx)
	if len(dbRT.calls) != 2 {
		t.Fatalf("reprovision calls = %v, want both %s and %s", dbRT.calls, a.ID, b.ID)
	}
	if len(appRT.deployed) != 1 || appRT.deployed[0] != down.ID+"@nginx:2" {
		t.Fatalf("apps not reconciled after a db failure: %v", appRT.deployed)
	}
}
```

Notes:
- Memory `CreateApp` may refuse `Status` writes for unknown statuses. Use the `entities.AppStatus*` constants that exist; check `entities/app.go` for the stopped constant's real name.
- If `CurrentImage` needs a deployment row to return the image, the fallback in `current_image.go` reads `app.ImageURL`. Check it, and seed a deployment with `CreateDeployment` + `SetDeploymentImage` if the fallback doesn't cover it.

In `db_runtime_test.go`, add `TestReprovision_UsesStoredPassword`:
- create through `rt.Create` with a backend that records the password passed to `Provision`;
- `rt.Wait`;
- call `rt.Reprovision(ctx, db)`;
- assert the second `Provision` got the same password as the first.

- [ ] **Step 2: Run to verify failure**

Run: `cd services/api && go test ./internal/services/... -run 'Reconciler|Reprovision' -race`
Expected: compile errors (`undefined: NewReconciler`).

- [ ] **Step 3: Implement**

**`db_runtime.go`:**

```go
// Reprovision recreates a provisioned database's instance with its stored
// password. The data volume is kept, so this is safe for a container that
// vanished or a restore onto a new host.
func (s *DBRuntimeService) Reprovision(ctx context.Context, db *entities.UserDatabase) error {
	pw, err := s.Password(ctx, db)
	if err != nil {
		return err
	}
	return s.provision(ctx, db, pw)
}
```

**`docker_db.go`:**

```go
// ContainerRunning reports whether the database container exists and runs.
func (b *DockerDBBackend) ContainerRunning(ctx context.Context, db *entities.UserDatabase) (bool, error) {
	insp, err := b.cli.ContainerInspect(ctx, dbContainerName(db))
	if client.IsErrNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return insp.State != nil && (insp.State.Running || insp.State.Restarting), nil
}
```

**`docker_deployer.go`:** add `AppContainerRunning`, with the same body using `d.containerName(app)`.

**`reconciler.go`:**

```go
package deploy

import (
	"context"
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// AppRuntime is what the reconciler needs from an app backend.
type AppRuntime interface {
	AppContainerRunning(ctx context.Context, app *entities.App) (bool, error)
	DeployApp(ctx context.Context, app *entities.App, image string) error
}

// DBProber reports whether a database instance is up.
type DBProber interface {
	ContainerRunning(ctx context.Context, db *entities.UserDatabase) (bool, error)
}

// DBReprovisioner recreates a database instance from its record.
type DBReprovisioner interface {
	Reprovision(ctx context.Context, db *entities.UserDatabase) error
}

// Reconciler brings apps and databases back to their recorded state after
// the host restarts, a container is removed, or a backup is restored onto a
// new server. Databases first, so apps start against live databases.
type Reconciler struct {
	apps    ports.AppRepository
	dbs     ports.DatabaseRepository
	appRT   AppRuntime
	dbProbe DBProber
	dbRT    DBReprovisioner
}

func NewReconciler(apps ports.AppRepository, dbs ports.DatabaseRepository, appRT AppRuntime, dbProbe DBProber, dbRT DBReprovisioner) *Reconciler {
	return &Reconciler{apps: apps, dbs: dbs, appRT: appRT, dbProbe: dbProbe, dbRT: dbRT}
}

// Run reconciles every item once. Each failure is logged; none stops the rest.
func (r *Reconciler) Run(ctx context.Context) {
	dbs, err := r.dbs.ListDatabasesByStatus(ctx, entities.DatabaseStatusReady)
	if err != nil {
		slog.Error("reconcile: list databases", "error", err)
	}
	for i := range dbs {
		db := &dbs[i]
		if up, err := r.dbProbe.ContainerRunning(ctx, db); err != nil || up {
			continue
		}
		slog.Info("reconcile: re-provisioning database", "db", db.Name)
		if err := r.dbRT.Reprovision(ctx, db); err != nil {
			slog.Error("reconcile: database failed", "db", db.Name, "error", err)
		}
	}

	apps, err := r.apps.ListAllApps(ctx)
	if err != nil {
		slog.Error("reconcile: list apps", "error", err)
		return
	}
	for i := range apps {
		app := &apps[i]
		if app.Status != entities.AppStatusRunning {
			continue
		}
		if up, err := r.appRT.AppContainerRunning(ctx, app); err != nil || up {
			continue
		}
		image, err := CurrentImage(ctx, r.apps, app.ID)
		if err != nil || image == "" {
			slog.Warn("reconcile: app has no image to redeploy", "app", app.Name)
			continue
		}
		slog.Info("reconcile: redeploying app", "app", app.Name, "image", image)
		if err := r.appRT.DeployApp(ctx, app, image); err != nil {
			slog.Error("reconcile: app failed", "app", app.Name, "error", err)
		}
	}
}
```

**`main.go`:**
- at :897-899, build `dbRuntime := services.NewDBRuntimeService(dbRepo, dockerDBBackend, envCrypto)` with `dockerDBBackend := deploy.NewDockerDBBackend(dockerDeployerCli, cfg.AppNetwork)`, and pass `dbRuntime` to `SetRuntime`;
- in the same block, start the reconciler:

```go
		reconciler := deploy.NewReconciler(appRepo, dbRepo, dockerDeployer, dockerDBBackend, dbRuntime)
		go func() {
			time.Sleep(10 * time.Second)
			reconciler.Run(context.Background())
		}()
```

`dockerDeployer` is the `*deploy.DockerDeployer` from :541. Guard with `dockerDeployer != nil`.

- [ ] **Step 4: Run tests**

Run: `cd services/api && go vet ./... && go test ./... -race -count=1 2>&1 | grep -v '^ok\|no test files'`
Expected: no output (all packages ok).

- [ ] **Step 5: Commit** — subject `feat(api): startup reconciler restores apps and databases from their records`

---

### Task 6: `zen db connect` over SSH

**Files:**
- Modify:
  - `cli/internal/api/client.go`: `Database` struct :223
  - `cli/cmd/db/db.go`: flags in `init` :79, `runConnect` :121, `connectDatabase` :203-240
- Test: `cli/cmd/db/db_test.go`

**Interfaces:**
- Consumes: API field `host_port` (Task 2/4); `Database.ConnectionString`, which is in-network, e.g. `postgres://u:p@zenith-db-1a2b3c4d:5432/db`.
- Produces: `func tunnelPlan(db *cliapi.Database, apiEndpoint, sshTarget string, localPort int) (sshArgs []string, localConn string, err error)`.

- [ ] **Step 1: Write the failing test** — append to `cli/cmd/db/db_test.go`:

```go
func TestTunnelPlan(t *testing.T) {
	db := &cliapi.Database{Name: "pg1", Engine: "postgresql", HostPort: 49153,
		ConnectionString: "postgres://u_1:s3cr3t@zenith-db-1a2b3c4d:5432/z_db?sslmode=disable"}
	args, local, err := tunnelPlan(db, "https://zenith.example.com", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-N", "-L", "5432:127.0.0.1:49153", "root@zenith.example.com"}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Fatalf("ssh args = %v, want %v", args, want)
	}
	if local != "postgres://u_1:s3cr3t@127.0.0.1:5432/z_db?sslmode=disable" {
		t.Fatalf("local = %q", local)
	}
	args, local, _ = tunnelPlan(db, "https://zenith.example.com", "admin@10.0.0.5", 15432)
	if args[2] != "15432:127.0.0.1:49153" || args[3] != "admin@10.0.0.5" || !strings.Contains(local, "@127.0.0.1:15432/") {
		t.Fatalf("overrides ignored: %v %q", args, local)
	}
}

func TestTunnelPlan_NoHostPort(t *testing.T) {
	_, _, err := tunnelPlan(&cliapi.Database{Name: "pg1", Engine: "postgresql"}, "https://x.example.com", "", 0)
	if err == nil || !strings.Contains(err.Error(), "no loopback port") {
		t.Fatalf("err = %v", err)
	}
}
```

Default local ports: postgresql 5432, mysql 3306, redis 6379. Add `strings` to the test imports if missing.

- [ ] **Step 2: Run to verify failure**

Run: `cd cli && go test ./cmd/db -run TunnelPlan -race`
Expected: compile error (`unknown field HostPort` / `undefined: tunnelPlan`).

- [ ] **Step 3: Implement**
- **`client.go` `Database`:** add `` HostPort int `json:"host_port"` ``.
- **`db.go`:**
  - flags on `connectCmd`: `--tunnel` (bool), `--ssh` (string, "SSH target, default root@<API host>"), `--local-port` (int);
  - `tunnelPlan`: parse `apiEndpoint` with `net/url` and use `Hostname()` as the SSH host;
  - choose the default local port by engine;
  - rewrite the connection string host:port to `127.0.0.1:<local>` with `net/url` (`u.Host = net.JoinHostPort("127.0.0.1", strconv.Itoa(local))`);
  - error `database %q has no loopback port yet — redeploy it or upgrade Zenith` when `HostPort == 0`.
- **`connectDatabase(w, c, project, name, opts connectOpts)`:**
  - after resolving `conn` and the full record (fetch `GetDatabaseByID` when needed so `HostPort` is filled), print:
    - the in-network string (existing behaviour; reword the muted note to "Inside the app network:");
    - a blank line;
    - "From your machine:", the ssh command `ssh -N -L ...`, and the local connection string;
  - with `--tunnel`: `exec.Command("ssh", args...)` with Stdin/Stdout/Stderr attached, print the local string first, then `Run()`;
  - `opts.apiEndpoint` comes from the loaded config (`cfg.APIEndpoint`); extend `clientAndProject` to return it.

- [ ] **Step 4: Run tests**

Run: `cd cli && go test ./... -race -count=1`
Expected: PASS

- [ ] **Step 5: Commit** — subject `feat(cli): zen db connect prints an ssh tunnel command; --tunnel opens it`

---

### Task 7: Versioned releases (F-08)

**Files:**
- Create: `VERSION` (content `0.1.0-alpha.2`, one line)
- Modify:
  - `.github/workflows/build-public-images.yml`: "Resolve tag" step; build step `build-args`/`tags`; matrix
  - `.github/workflows/ci.yml`: test job, after the `services/cli` steps
  - `infra/scripts/install.sh`: the `.env` blocks :130-186
  - `infra/scripts/install_test.sh`: `run()` and new assertions
  - `cli/cmd/version/version.go`

**Interfaces:**
- Produces:
  - `ZENITH_VERSION` in `.env`, equal to the `VERSION` file;
  - images report `version` = the tag without `v`;
  - `zen version` prints `Server: <version>` when a config with `api_endpoint` exists.

- [ ] **Step 1: Write the failing test.** In `install_test.sh`:
  - `run()` also copies the `VERSION` file: `cp "$repo/VERSION" "$dir/VERSION"`;
  - add after the first `run "$t/a"` block:

```bash
grep -qx "ZENITH_VERSION=$(cat "$repo/VERSION")" "$env_a" || fail "ZENITH_VERSION not pinned from VERSION"
```

  - in the old-style `.env` upgrade case, assert the same line is present after the re-run.

- [ ] **Step 2: Run to verify failure**

Run: `bash infra/scripts/install_test.sh`
Expected: `FAIL: VERSION` copy error, or `FAIL: ZENITH_VERSION not pinned from VERSION`.

- [ ] **Step 3: Implement**

**`VERSION` file:** `0.1.0-alpha.2`.

**`install.sh`:** after `set_var ZENITH_DOMAIN "$ZENITH_DOMAIN"` in the new-`.env` block, **and** inside the existing-`.env` upgrade block, add:

```bash
  # Pin the images to this checkout's release, never a moving :latest.
  if [ -f VERSION ]; then set_var ZENITH_VERSION "$(tr -d '[:space:]' < VERSION)"; fi
```

The upgrade-block copy is unconditional. Re-running the installer after `git pull` is an upgrade to the checked-out version.

**`build-public-images.yml`:**
- in "Resolve tag", also output `latest=true` when the event is `workflow_dispatch` and the tag is `latest`, or when the release is not a prerelease (`${{ github.event.release.prerelease }}` is `false`); otherwise output `latest=false`;
- build step:
  - `build-args`: `VERSION=${{ steps.tag.outputs.tag }}` and `BUILD_TIME=${{ github.event.repository.updated_at }}`, plus the matrix `build_args`. Join with newlines as a block scalar so the web's `NEXT_PUBLIC_ZENITH_MODE=standalone` stays;
  - `labels`: `org.opencontainers.image.version=${{ steps.tag.outputs.tag }}` and `org.opencontainers.image.revision=${{ github.sha }}`;
  - `tags`: always `:${{ steps.tag.outputs.tag }}`; the `:latest` line only when `steps.tag.outputs.latest == 'true'`, using `${{ steps.tag.outputs.latest == 'true' && format('{0}/{1}/{2}:latest', env.REGISTRY, steps.owner.outputs.owner, matrix.image) || '' }}`;
- update the workflow's `workflow_dispatch` input description to say `:latest` is pushed only for the tag `latest` or a final release.

**`apps/web/Dockerfile`:** add `ARG VERSION=dev` in the builder stage, with no other use. It keeps the build-arg list uniform without a build warning.

**`ci.yml` test job:** add two steps after the `services/cli` build:

```yaml
      - name: Test zen CLI (cli/)
        run: cd cli && go test ./... -race -count=1
      - name: Build zen CLI (cli/)
        run: cd cli && go build -o /dev/null .
```

Check that `cli/` builds from its module root with `go build .`. If `main.go` lives elsewhere, use that path (the Makefile `build-cli-release` target shows it).

**`cli/cmd/version/version.go`:** after printing the CLI version, load the config (`config.Load()`). When `APIEndpoint != ""`:
- GET `<endpoint>/api/v1/version` with a 3 s timeout;
- print `Server:  <version>`, or `Server:  unreachable`;
- keep `GetVersionInfo` unchanged so its existing test still passes.

- [ ] **Step 4: Run tests**

Run:
- `bash infra/scripts/install_test.sh && (cd cli && go test ./cmd/version -race)`
- `python3 -c "import yaml,sys;[yaml.safe_load(open(f)) for f in sys.argv[1:]]" .github/workflows/build-public-images.yml .github/workflows/ci.yml`

Expected: `install_test: ok`, `ok`, and no YAML error.

- [ ] **Step 5: Commit** — subject `feat(release): version file pins installs; images report their version (F-08)`

---

### Task 8: `composeops` core and `zen backup` for Compose

**Files:**
- Create:
  - `cli/internal/composeops/install.go` (detection, `.env` helpers, command runner)
  - `cli/internal/composeops/backup.go`
  - `cli/internal/composeops/fake_test.go`
  - `cli/internal/composeops/backup_test.go`
  - `cli/internal/composeops/install_test.go`
- Modify: `cli/cmd/backup/backup.go`: `Cmd` gets a `RunE` and `--dir`/`--out` flags

**Interfaces:**
- Produces:

```go
type Runner interface {
	// Run executes name+args in dir. stdin may be nil; stdout is returned.
	Run(ctx context.Context, dir string, stdin io.Reader, name string, args ...string) ([]byte, error)
	// RunTo executes and streams stdout into w (large dumps).
	RunTo(ctx context.Context, dir string, w io.Writer, name string, args ...string) error
}
type ExecRunner struct{}            // os/exec implementation
type Install struct{ Dir string }   // a Compose install directory
func Detect(flagDir string) (Install, bool)
func (i Install) EnvGet(key string) string
func (i Install) EnvSet(key, value string) error
type DBRef struct{ ID, Engine, Name string } // Engine: postgresql|mysql|redis
func (i Install) Databases(ctx context.Context, r Runner) ([]DBRef, error)
func DBContainer(id string) string // "zenith-db-" + first 8 chars
type Manifest struct {
	Format    int     `json:"format"`     // 1
	Version   string  `json:"version"`    // ZENITH_VERSION of the install
	CreatedAt string  `json:"created_at"` // RFC3339 UTC
	Databases []DBRef `json:"databases"`
}
func Backup(ctx context.Context, r Runner, inst Install, outDir string, now time.Time) (string, error)
```

- [ ] **Step 1: Write the failing tests**

`fake_test.go`, a scripted runner that records calls and answers by command prefix:

```go
package composeops

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
)

type call struct{ dir, cmd string }

type fakeRunner struct {
	mu      sync.Mutex
	calls   []call
	answers map[string]string // command prefix -> stdout
	fail    map[string]bool   // command prefix -> error
}

func (f *fakeRunner) match(cmd string, m map[string]string) (string, bool) {
	best := ""
	for p := range m {
		if strings.HasPrefix(cmd, p) && len(p) > len(best) {
			best = p
		}
	}
	if best == "" {
		return "", false
	}
	return m[best], true
}

func (f *fakeRunner) record(dir, name string, args []string) (string, error) {
	cmd := strings.TrimSpace(name + " " + strings.Join(args, " "))
	f.mu.Lock()
	f.calls = append(f.calls, call{dir, cmd})
	f.mu.Unlock()
	for p := range f.fail {
		if strings.HasPrefix(cmd, p) {
			return "", fmt.Errorf("fake failure: %s", p)
		}
	}
	out, _ := f.match(cmd, f.answers)
	return out, nil
}

func (f *fakeRunner) Run(_ context.Context, dir string, _ io.Reader, name string, args ...string) ([]byte, error) {
	out, err := f.record(dir, name, args)
	return []byte(out), err
}

func (f *fakeRunner) RunTo(_ context.Context, dir string, w io.Writer, name string, args ...string) error {
	out, err := f.record(dir, name, args)
	if err == nil {
		_, _ = io.Copy(w, bytes.NewBufferString(out))
	}
	return err
}

func (f *fakeRunner) cmds() []string {
	var s []string
	for _, c := range f.calls {
		s = append(s, c.cmd)
	}
	return s
}
```

`install_test.go`:

```go
package composeops

import (
	"os"
	"path/filepath"
	"testing"
)

func writeInstall(t *testing.T) Install {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services:\n  api:\n    image: ghcr.io/taikuri-infra/zenith-api:${ZENITH_VERSION:-latest}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("ZENITH_VERSION=0.1.0-alpha.2\nDB_PASSWORD=old\n"), 0o600)
	return Install{Dir: dir}
}

func TestDetect(t *testing.T) {
	inst := writeInstall(t)
	if got, ok := Detect(inst.Dir); !ok || got.Dir != inst.Dir {
		t.Fatalf("Detect(%q) = %v %v", inst.Dir, got, ok)
	}
	if _, ok := Detect(t.TempDir()); ok {
		t.Fatal("empty dir detected as install")
	}
}

func TestEnvGetSet(t *testing.T) {
	inst := writeInstall(t)
	if inst.EnvGet("ZENITH_VERSION") != "0.1.0-alpha.2" {
		t.Fatal("EnvGet")
	}
	if err := inst.EnvSet("DB_PASSWORD", "n|e&w"); err != nil {
		t.Fatal(err)
	}
	if err := inst.EnvSet("NEW_KEY", "v"); err != nil {
		t.Fatal(err)
	}
	if inst.EnvGet("DB_PASSWORD") != "n|e&w" || inst.EnvGet("NEW_KEY") != "v" || inst.EnvGet("ZENITH_VERSION") != "0.1.0-alpha.2" {
		t.Fatalf("env after set: %q %q", inst.EnvGet("DB_PASSWORD"), inst.EnvGet("NEW_KEY"))
	}
	if st, _ := os.Stat(filepath.Join(inst.Dir, ".env")); st.Mode().Perm() != 0o600 {
		t.Fatalf(".env mode %v", st.Mode().Perm())
	}
}
```

`backup_test.go`:

```go
package composeops

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func archiveEntries(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	out := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(tr)
		out[h.Name] = string(b)
	}
}

func TestBackup_WritesEveryPart(t *testing.T) {
	inst := writeInstall(t)
	r := &fakeRunner{answers: map[string]string{
		"docker compose exec -T postgres psql":    "11111111-aaaa|postgresql|pg1\n22222222-bbbb|mysql|my1\n33333333-cccc|redis|rd1\n",
		"docker compose exec -T postgres pg_dump": "-- platform dump",
		"docker compose ps -q storage":            "cid-storage\n",
		"docker inspect":                          "zenith_storagedata\n",
		"docker exec zenith-db-11111111 pg_dump":  "-- pg1 dump",
		"docker exec zenith-db-22222222 sh -c":    "-- my1 dump",
	}}
	out := t.TempDir()
	path, err := Backup(context.Background(), r, inst, out, time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "zenith-backup-0.1.0-alpha.2-20261013T080000Z.tar.gz") {
		t.Fatalf("path = %s", path)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Fatalf("archive mode %v", st.Mode().Perm())
	}
	e := archiveEntries(t, path)
	for _, name := range []string{"manifest.json", "platform.sql", "storage.tar", "env", "databases/11111111-aaaa.sql", "databases/22222222-bbbb.sql", "databases/33333333-cccc.rdb"} {
		if _, ok := e[name]; !ok {
			t.Errorf("archive lacks %s (has %v)", name, keys(e))
		}
	}
	if e["platform.sql"] != "-- platform dump" || !strings.Contains(e["env"], "DB_PASSWORD=old") {
		t.Errorf("content wrong: platform=%q env=%q", e["platform.sql"], e["env"])
	}
	var m Manifest
	if err := json.Unmarshal([]byte(e["manifest.json"]), &m); err != nil || m.Version != "0.1.0-alpha.2" || len(m.Databases) != 3 || m.Format != 1 {
		t.Fatalf("manifest = %+v err %v", m, err)
	}
}

func TestBackup_DumpFailureLeavesNoArchive(t *testing.T) {
	inst := writeInstall(t)
	r := &fakeRunner{
		answers: map[string]string{"docker compose exec -T postgres psql": "11111111-aaaa|postgresql|pg1\n"},
		fail:    map[string]bool{"docker exec zenith-db-11111111 pg_dump": true},
	}
	out := t.TempDir()
	if _, err := Backup(context.Background(), r, inst, out, time.Now()); err == nil {
		t.Fatal("want error")
	}
	if left, _ := os.ReadDir(out); len(left) != 0 {
		t.Fatalf("partial files left: %v", left)
	}
}

func keys(m map[string]string) []string {
	var k []string
	for x := range m {
		k = append(k, x)
	}
	return k
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd cli && go test ./internal/composeops -race`
Expected: compile errors (`undefined: Install`).

- [ ] **Step 3: Implement**

**`install.go`:**
- **`Detect(flagDir)`:** candidates are `flagDir` (if set, it is the only candidate), the cwd, `$HOME/zenith`, `/root/zenith`. The first dir whose `docker-compose.yml` contains `zenith-api` wins.
- **`EnvGet`:** parse `.env` lines `KEY=VALUE`; the last occurrence wins; `#` comments are ignored.
- **`EnvSet`:** rewrite the file, replacing the line or appending. Write a temp file in the same dir with mode 0600, then rename. Values are written verbatim, with no shell escaping; compose `.env` takes the raw value.
- **`ExecRunner.Run`/`RunTo`:** use `exec.CommandContext` with `cmd.Dir = dir`. On error, include the command and the last 2 KB of stderr in the message.
- **`Databases`:** run `docker compose exec -T postgres psql -U zenith -d zenith -At -F '|' -c "SELECT id, engine, name FROM user_databases WHERE status = 'ready' ORDER BY created_at"` and parse the `|`-separated lines.
- **`DBContainer(id)`:** `"zenith-db-" + id[:min(8,len(id))]`.

**`backup.go`, `Backup` steps, all inside a temp dir under `outDir`:**
1. Run `Databases`, and set `version := inst.EnvGet("ZENITH_VERSION")`.
2. Write `platform.sql` via `RunTo(..., "docker", "compose", "exec", "-T", "postgres", "pg_dump", "-U", "zenith", "-d", "zenith", "--clean", "--if-exists")`.
3. Get the storage container with `docker compose ps -q storage`.
4. Get its volume with `docker inspect -f '{{range .Mounts}}{{if eq .Destination "/data"}}{{.Name}}{{end}}{{end}}' <cid>`.
5. Write `storage.tar` via `RunTo(..., "docker", "run", "--rm", "-v", vol+":/data:ro", "alpine:3.20", "tar", "-C", "/data", "-cf", "-", ".")`.
6. Dump each database by engine:
   - **postgresql:** read the user and database with `docker exec <c> printenv POSTGRES_USER` and `docker exec <c> printenv POSTGRES_DB`. Then `RunTo` `docker exec <c> pg_dump -U <user> -d <db> --clean --if-exists`. The test fakes the prefix `docker exec zenith-db-11111111 pg_dump`.
   - **mysql:** `RunTo` `docker exec <c> sh -c 'exec mysqldump -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" --single-transaction --no-tablespaces "$MYSQL_DATABASE"'`. The test fakes the prefix `docker exec zenith-db-22222222 sh -c`.
   - **redis:** `docker exec <c> sh -c 'redis-cli -a "$REDIS_PASSWORD" --no-auth-warning SAVE'`, then `docker cp <c>:/data/dump.rdb <tmp>/databases/<id>.rdb`.
7. Copy `<install>/.env` to `env`.
8. Write `manifest.json` (`Format: 1`).
9. Tar+gzip the temp dir contents into `<outDir>/zenith-backup-<version>-<now UTC 20060102T150405Z>.tar.gz` with mode 0600. Entries are relative paths, as listed in the test.
10. Remove the temp dir.

Any error removes the temp dir **and** any partial archive, then returns the error. In the test's fake, `printenv` answers `""`; pass `-U "" -d ""` through without validating so the fake works.

**`cmd/backup/backup.go`:**
- `Cmd.RunE = runComposeBackup`: `composeops.Detect(flagDir)`. If there's no install, return `cmd.Help()`, which keeps `zen backup create` (CNPG) as the documented path.
- Otherwise call `composeops.Backup(ctx, composeops.ExecRunner{}, inst, out, time.Now())`, where `out` defaults to `<inst.Dir>/backups` (created 0700).
- Print the path and two lines: "This archive contains your secrets (SECRETS_ENCRYPTION_KEY)." and "Copy it off this server: it is on the same disk as the data."
- Flags on `Cmd`: `--dir`, `--out`. Update `Short` to `Back up Zenith (Compose: whole platform; Helm: zen backup create)`.

- [ ] **Step 4: Run tests**

Run: `cd cli && go vet ./... && go test ./... -race -count=1`
Expected: PASS

- [ ] **Step 5: Commit** — subject `feat(cli): zen backup archives a compose install (platform, storage, databases, env)`

---

### Task 9: `zen backup restore` for Compose

**Files:**
- Create: `cli/internal/composeops/restore.go`, `cli/internal/composeops/restore_test.go`
- Modify: `cli/cmd/backup/backup.go`: new `restoreCmd` with `restore <archive>`, `--yes`, `--dir`

**Interfaces:**
- Consumes: `Runner`, `Install`, `Manifest`, `DBRef`, `DBContainer` (Task 8).
- Produces:
  - `type RestoreOptions struct { Wait time.Duration; Poll time.Duration }` (defaults 5m / 3s);
  - `func Restore(ctx context.Context, r Runner, inst Install, archive string, opt RestoreOptions) (*Manifest, error)`;
  - `func CompareVersions(a, b string) int` (semver with prerelease, -1/0/1).

- [ ] **Step 1: Write the failing tests** — `restore_test.go`:

```go
package composeops

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func makeArchive(t *testing.T, m Manifest, env string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "b.tar.gz")
	f, _ := os.Create(path)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	mj, _ := json.Marshal(m)
	files := map[string]string{"manifest.json": string(mj), "platform.sql": "-- p", "storage.tar": "", "env": env}
	for _, d := range m.Databases {
		ext := ".sql"
		if d.Engine == "redis" {
			ext = ".rdb"
		}
		files["databases/"+d.ID+ext] = "-- " + d.Name
	}
	for n, c := range files {
		tw.WriteHeader(&tar.Header{Name: n, Mode: 0o600, Size: int64(len(c))})
		tw.Write([]byte(c))
	}
	tw.Close()
	gz.Close()
	f.Close()
	return path
}

func TestRestore_ResetsControlPlanePassword(t *testing.T) {
	inst := writeInstall(t) // .env has DB_PASSWORD=old, ZENITH_VERSION=0.1.0-alpha.2
	arch := makeArchive(t, Manifest{Format: 1, Version: "0.1.0-alpha.2"},
		"DB_PASSWORD=fromarchive\nSECRETS_ENCRYPTION_KEY=abc\nZENITH_DOMAIN=old.example.com\nZENITH_VERSION=0.1.0-alpha.1\n")
	r := &fakeRunner{answers: map[string]string{"docker compose ps -q storage": "cid\n", "docker inspect": "vol\n"}}
	if _, err := Restore(context.Background(), r, inst, arch, RestoreOptions{Wait: time.Second, Poll: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if inst.EnvGet("DB_PASSWORD") != "fromarchive" || inst.EnvGet("SECRETS_ENCRYPTION_KEY") != "abc" {
		t.Fatal("secret keys not restored")
	}
	if inst.EnvGet("ZENITH_VERSION") != "0.1.0-alpha.2" || inst.EnvGet("ZENITH_DOMAIN") != "" {
		t.Fatal("non-secret keys must stay the target's")
	}
	joined := strings.Join(r.cmds(), "\n")
	if !strings.Contains(joined, "ALTER USER zenith PASSWORD 'fromarchive'") {
		t.Fatalf("control-plane password not reset:\n%s", joined)
	}
	stop := strings.Index(joined, "docker compose stop api web")
	load := strings.Index(joined, "psql -U zenith -d zenith -v ON_ERROR_STOP=1")
	up := strings.Index(joined, "docker compose up -d\n")
	if stop < 0 || load < 0 || up < 0 || !(stop < load && load < up) {
		t.Fatalf("order wrong (stop %d, load %d, up %d):\n%s", stop, load, up, joined)
	}
}

func TestRestore_LoadsEachDatabaseAfterHealthy(t *testing.T) {
	inst := writeInstall(t)
	m := Manifest{Format: 1, Version: "0.1.0-alpha.2", Databases: []DBRef{
		{ID: "11111111-aaaa", Engine: "postgresql", Name: "pg1"},
		{ID: "33333333-cccc", Engine: "redis", Name: "rd1"},
	}}
	r := &fakeRunner{answers: map[string]string{
		"docker compose ps -q storage": "cid\n", "docker inspect -f {{range": "vol\n",
		"docker inspect -f {{.State.Health.Status}}": "healthy\n",
	}}
	if _, err := Restore(context.Background(), r, inst, makeArchive(t, m, "DB_PASSWORD=x\n"), RestoreOptions{Wait: time.Second, Poll: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(r.cmds(), "\n")
	for _, want := range []string{"docker exec -i zenith-db-11111111", "docker stop zenith-db-33333333", "docker start zenith-db-33333333"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
}

func TestRestore_RefusesNewerArchive(t *testing.T) {
	inst := writeInstall(t)
	arch := makeArchive(t, Manifest{Format: 1, Version: "0.2.0"}, "")
	if _, err := Restore(context.Background(), &fakeRunner{}, inst, arch, RestoreOptions{}); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("err = %v", err)
	}
}

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"0.1.0-alpha.2", "0.1.0-alpha.10", -1},
		{"0.1.0-alpha.2", "0.1.0", -1},
		{"0.2.0", "0.1.9", 1},
		{"v0.1.0", "0.1.0", 0},
	} {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
```

**Health polling:** the database health probe is `docker inspect -f {{.State.Health.Status}} <container>`. The storage volume probe is `docker inspect -f {{range .Mounts}}...`. Use exactly these argument shapes, with no quotes around the template, so the fake's prefixes match.

- [ ] **Step 2: Run to verify failure**

Run: `cd cli && go test ./internal/composeops -run 'Restore|CompareVersions' -race`
Expected: compile errors (`undefined: Restore`).

- [ ] **Step 3: Implement `restore.go`**

**Setup:**
1. Extract the archive into a temp dir, rejecting entries with `..` or absolute paths. Read `manifest.json`; require `Format == 1`.
2. If `CompareVersions(m.Version, inst.EnvGet("ZENITH_VERSION")) > 0`, return `archive is from Zenith <v>, newer than this install (<v>) — upgrade first`.

**Restore sequence:**
1. `docker compose stop api web`.
2. Copy the secret keys (the Global Constraints list) that are present in the archive `env` into `inst` with `EnvSet`.
3. `docker compose up -d postgres storage`, then wait until `docker compose exec -T postgres pg_isready -U zenith` succeeds, polling `opt.Poll` up to `opt.Wait`.
4. `docker compose exec -T postgres psql -U zenith -d postgres -c "ALTER USER zenith PASSWORD '<DB_PASSWORD>'"`. Double any `'` in the value.
5. `docker compose exec -T postgres psql -U zenith -d postgres -c "DROP DATABASE IF EXISTS zenith WITH (FORCE)" -c "CREATE DATABASE zenith OWNER zenith"`.
6. `docker compose exec -T postgres psql -U zenith -d zenith -v ON_ERROR_STOP=1`, with stdin = `platform.sql`.
7. Storage volume (same lookup as backup): `docker run --rm -v <vol>:/data -v <tmp>:/in:ro alpine:3.20 sh -c "find /data -mindepth 1 -delete && tar -C /data -xf /in/storage.tar"`.
8. `docker compose up -d`. The reconciler re-creates the database containers within about 10 s of API start.

**Per database:**
1. Poll `docker inspect -f {{.State.Health.Status}} <c>` until `healthy`. Inspect errors count as not-yet, while the reconciler creates the container.
2. Load by engine:
   - **postgresql:** `docker exec -i <c> sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1'`, with stdin = the dump;
   - **mysql:** `docker exec -i <c> sh -c 'exec mysql -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE"'`, with stdin;
   - **redis:**
     1. read the volume with `docker inspect -f {{range .Mounts}}{{if eq .Destination "/data"}}{{.Name}}{{end}}{{end}} <c>`;
     2. `docker stop <c>`;
     3. `docker run --rm -v <vol>:/data -v <tmp>/databases:/in:ro alpine:3.20 sh -c "rm -rf /data/appendonlydir && cp /in/<id>.rdb /data/dump.rdb"`;
     4. `docker start <c>`.

**Error handling:** collect per-database failures and continue. Return the manifest plus a joined error naming each failed database.

**`CompareVersions`:**
- strip a leading `v`;
- split the core from the prerelease at the first `-`;
- compare the core numerically;
- a release ranks above any prerelease of the same core;
- prerelease identifiers are compared dot-wise: numeric numerically, otherwise lexically.

**`cmd/backup/backup.go`:** `restoreCmd` (`Use: "restore <archive>"`, `Args: cobra.ExactArgs(1)`):
- Detect the install, or fail with `no Compose install found (use --dir)`.
- Unless `--yes`, prompt `This replaces all data of the install in <dir>. Type 'restore' to continue:` and require exactly `restore`.
- Run `Restore` and print a summary: version, databases restored, failures.
- Register it with `Cmd.AddCommand(restoreCmd)`.

- [ ] **Step 4: Run tests**

Run: `cd cli && go vet ./... && go test ./... -race -count=1`
Expected: PASS

- [ ] **Step 5: Commit** — subject `feat(cli): zen backup restore rebuilds a compose install from an archive`

---

### Task 10: `zen upgrade` for Compose

**Files:**
- Create: `cli/internal/composeops/upgrade.go`, `cli/internal/composeops/upgrade_test.go`
- Modify: `cli/cmd/upgrade/upgrade.go`:
  - `versionRe` :20 relaxed to allow prerelease;
  - `runUpgrade` :71 begins with Compose detection;
  - new `--dir`, `--force` flags

**Interfaces:**
- Consumes: `Runner`, `Install`, `Backup`, `CompareVersions` (Tasks 8–9).
- Produces:

```go
type UpgradeOptions struct {
	Target     string                                         // required, no leading v
	Force      bool                                           // allow same/older target
	SkipBackup bool
	DryRun     bool
	BackupDir  string                                         // default <dir>/backups
	Health     func(ctx context.Context, want string) error  // nil = HTTP check on 127.0.0.1:8080
	Timeout    time.Duration                                  // default 180s
	Now        func() time.Time
}
type UpgradeResult struct{ From, To, Backup string; RolledBack bool }
func Upgrade(ctx context.Context, r Runner, inst Install, opt UpgradeOptions, log io.Writer) (*UpgradeResult, error)
func LatestRelease(ctx context.Context, repo string) (string, error) // GitHub API, includes prereleases
```

- [ ] **Step 1: Write the failing tests** — `upgrade_test.go`:

```go
package composeops

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func upgradeRunner() *fakeRunner {
	return &fakeRunner{answers: map[string]string{
		"git rev-parse HEAD":                      "oldsha\n",
		"docker compose exec -T postgres psql":    "",
		"docker compose ps -q storage":            "cid\n",
		"docker inspect":                          "vol\n",
	}}
}

func TestUpgrade_HappyPath(t *testing.T) {
	inst := writeInstall(t)
	r := upgradeRunner()
	res, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", Health: func(context.Context, string) error { return nil },
		Timeout: time.Second, Now: time.Now,
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if res.From != "0.1.0-alpha.2" || res.To != "0.1.0-alpha.3" || res.Backup == "" || res.RolledBack {
		t.Fatalf("res = %+v", res)
	}
	joined := strings.Join(r.cmds(), "\n")
	order := []string{
		"git fetch --depth 1 origin tag v0.1.0-alpha.3",
		"git checkout v0.1.0-alpha.3",
		"bash infra/scripts/install.sh",
		"docker compose pull",
		"docker compose up -d --remove-orphans",
	}
	last := -1
	for _, step := range order {
		i := strings.Index(joined, step)
		if i < 0 || i < last {
			t.Fatalf("step %q missing or out of order:\n%s", step, joined)
		}
		last = i
	}
}

func TestUpgrade_RollsBackOnHealthFailure(t *testing.T) {
	inst := writeInstall(t)
	r := upgradeRunner()
	calls := 0
	res, err := Upgrade(context.Background(), r, inst, UpgradeOptions{
		Target: "0.1.0-alpha.3", SkipBackup: true, Timeout: time.Second, Now: time.Now,
		Health: func(_ context.Context, want string) error {
			calls++
			if want == "0.1.0-alpha.3" {
				return errors.New("never healthy")
			}
			return nil
		},
	}, &bytes.Buffer{})
	if err == nil || res == nil || !res.RolledBack {
		t.Fatalf("want rollback, got res=%+v err=%v", res, err)
	}
	joined := strings.Join(r.cmds(), "\n")
	if !strings.Contains(joined, "git checkout oldsha") {
		t.Fatalf("old ref not restored:\n%s", joined)
	}
	if inst.EnvGet("ZENITH_VERSION") != "0.1.0-alpha.2" {
		t.Fatalf("ZENITH_VERSION = %q after rollback", inst.EnvGet("ZENITH_VERSION"))
	}
	if calls != 2 {
		t.Fatalf("health checks = %d, want new then old", calls)
	}
}

func TestUpgrade_RefusesDowngradeUnlessForced(t *testing.T) {
	inst := writeInstall(t)
	_, err := Upgrade(context.Background(), upgradeRunner(), inst, UpgradeOptions{Target: "0.1.0-alpha.1", SkipBackup: true, Now: time.Now, Health: func(context.Context, string) error { return nil }}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("err = %v", err)
	}
}

func TestUpgrade_DryRunChangesNothing(t *testing.T) {
	inst := writeInstall(t)
	r := upgradeRunner()
	var log bytes.Buffer
	if _, err := Upgrade(context.Background(), r, inst, UpgradeOptions{Target: "0.1.0-alpha.3", DryRun: true, Now: time.Now}, &log); err != nil {
		t.Fatal(err)
	}
	for _, c := range r.cmds() {
		if !strings.HasPrefix(c, "git rev-parse") {
			t.Fatalf("dry run executed %q", c)
		}
	}
	if !strings.Contains(log.String(), "0.1.0-alpha.3") {
		t.Fatalf("plan not printed: %s", log.String())
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd cli && go test ./internal/composeops -run Upgrade -race`
Expected: compile errors (`undefined: Upgrade`).

- [ ] **Step 3: Implement `upgrade.go`**

**Setup:**
1. `from := inst.EnvGet("ZENITH_VERSION")`.
2. Unless `opt.Force`, refuse when `CompareVersions(target, from) <= 0`: `target <t> is not newer than <from> (use --force)`.
3. Get the rollback ref with `git rev-parse HEAD` in `inst.Dir`.
4. If `DryRun`, print the numbered plan (backup path, fetch/checkout tag, env update, pull, up, health) to `log` and return.
5. Unless `SkipBackup`, run `Backup(ctx, r, inst, backupDir, opt.Now())`; the result goes into `UpgradeResult.Backup`.

**Upgrade steps, each logged to `log`:**
1. `git fetch --depth 1 origin tag v<target>`
2. `git checkout v<target>`
3. `bash infra/scripts/install.sh`, with env `ZENITH_DRY_RUN=1`. Run it as `env ZENITH_DRY_RUN=1 bash infra/scripts/install.sh` through `Runner.Run(ctx, inst.Dir, nil, "env", "ZENITH_DRY_RUN=1", "bash", "infra/scripts/install.sh")`; the fake matches on the substring `bash infra/scripts/install.sh`. This adds new `.env` keys and sets `ZENITH_VERSION` from the checked-out `VERSION`.
4. If `inst.EnvGet("ZENITH_VERSION") != target`, call `inst.EnvSet("ZENITH_VERSION", target)`. This covers tags without a `VERSION` file, and the test's fake.
5. `docker compose pull`
6. `docker compose up -d --remove-orphans`
7. Health check: `opt.Health(ctx, target)` within `opt.Timeout`.

**On a failure in any upgrade step:**
1. `git checkout <oldref>`
2. `inst.EnvSet("ZENITH_VERSION", from)`
3. `docker compose up -d --remove-orphans`
4. `opt.Health(ctx, from)`
5. Set `RolledBack = true`.
6. Return an error that names the failed step. If the old version is also unhealthy, add `the database may already be migrated — restore with: zen backup restore <backup>` (or `take a backup next time` when skipped).

**Default `Health`:** poll every 3 s until timeout, requiring both:
- `http://127.0.0.1:8080/health` returns 200;
- `http://127.0.0.1:8080/api/v1/version` returns JSON `version == want`.

**`LatestRelease`:** GET `https://api.github.com/repos/<repo>/releases?per_page=1`, take the first non-draft `tag_name`, and strip `v`.

**`cmd/upgrade/upgrade.go`:**
- Relax `versionRe` to `^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$` and update its error text example to `1.5.0 or 0.1.0-alpha.3`.
- At the top of `runUpgrade`, `if inst, ok := composeops.Detect(flagDir); ok { return runComposeUpgrade(cmd, inst) }`.
- `runComposeUpgrade`:
  1. Resolve the target: `flagVersion`, else `composeops.LatestRelease(ctx, "taikuri-infra/Zenith")`.
  2. Call `composeops.Upgrade` with `ExecRunner{}`, the flags, and `log = os.Stdout`.
  3. Print the result: from → to, the backup path, and `rolled back` if so.
- Flags: `--dir`, `--force`. The `Long` text adds one paragraph: on a Compose install, `zen upgrade` runs on the server in the install directory.

- [ ] **Step 4: Run tests**

Run: `cd cli && go vet ./... && go test ./... -race -count=1`
Expected: PASS (the existing `upgrade_test.go` still passes with the relaxed regex; update any of its cases that asserted a prerelease is invalid).

- [ ] **Step 5: Commit** — subject `feat(cli): zen upgrade for compose installs with backup and automatic rollback`

---

### Task 11: Acceptance on the live install (controller)

Run by the controller, not an implementer. It needs live judgment, and the reboot needs the owner's yes.

**Files:**
- Modify: `docs/alpha/core-audit.md`: new section "§7 After 2b", with each check's result and evidence.

- [ ] **Step 1: Build and run.**
  1. Build `zenith-api` and `zenith-web` locally from HEAD as `:0.1.0-alpha.2` (`docker build --build-arg VERSION=0.1.0-alpha.2 ...`).
  2. Point `/root/zenith-2a` at them (`ZENITH_VERSION=0.1.0-alpha.2`), copy the new `docker-compose.yml` into it, and `docker compose up -d --remove-orphans`.
  3. Check that `/api/v1/version` reports `0.1.0-alpha.2` and that `myapp.freezenith.com` returns 200.
- [ ] **Step 2: F-21 / network split.**
  1. Import a compose stack with services `web` (image `traefik/whoami`), `postgres` (`postgres:16-alpine`) and `api` (`traefik/whoami`).
  2. In its `web` container, `getent hosts postgres` resolves to the stack's own `postgres` container IP, not Zenith's control-plane postgres (compare `docker inspect` IPs).
  3. `getent hosts storage` fails.
  4. An app in another project can't resolve the first stack's `postgres` short name.
- [ ] **Step 3: SSH database access.**
  1. `ss -ltnp` shows each `zenith-db-*` port bound only to `127.0.0.1`.
  2. From the host, run `zen db connect pg1`, then run the printed ssh command against `root@127.0.0.1` with `-p 22`, then `psql` the printed local string, then `SELECT 1`.
  3. The public port scan (`ss -ltn` on 0.0.0.0/::) is unchanged: 22, 80, 443 only.
- [ ] **Step 4: Reconciler.**
  1. `docker rm -f zenith-db-<pg1 id8>` and `docker rm -f zenith-app-<myapp subdomain>`.
  2. `docker compose restart api`.
  3. Within 60 s both are back, `pg1` rows are intact, and `myapp` returns 200.
- [ ] **Step 5: Backup and restore.**
  1. Insert a marker row in `pg1` and a marker key in `rd1`.
  2. `zen backup`.
  3. Fresh second install via `install.sh` with `ZENITH_DIR=/root/zenith-restore`, its own compose project name, ports remapped to loopback-only alternates, no Traefik on 80/443 (`--profile` off), and `ZENITH_REPO_URL=/opt/Zenith`.
  4. `zen backup restore <archive> --yes --dir /root/zenith-restore`.
  5. Admin login works with the restored credentials, the marker row and key are present, and apps are listed.
  6. Tear down the restore install afterwards.
- [ ] **Step 6: Upgrade with rollback.**
  1. In a local clone used as `ZENITH_REPO_URL`, tag `v0.1.0-alpha.2` (HEAD) and `v0.1.0-alpha.3-test` (`VERSION` bumped), and build images for both locally.
  2. `zen upgrade --version 0.1.0-alpha.3-test --dir /root/zenith-2a` succeeds, and the version endpoint reports it.
  3. Then force a failure (tag `v0.1.0-alpha.4-bad` with an API image whose entrypoint exits 1). `zen upgrade` to it rolls back to `0.1.0-alpha.3-test`, and the data survives.
  4. Restore the install to `0.1.0-alpha.2` afterwards.
- [ ] **Step 7: Reboot.** Ask the owner for an explicit yes. Then `reboot`. After boot, every app and database is back with data, and `myapp.freezenith.com` and the dashboard return 200 with no manual step.
- [ ] **Step 8: Record and commit.** Write §7 in `core-audit.md`, then commit with subject `docs(alpha): core audit — status after 2b`.
