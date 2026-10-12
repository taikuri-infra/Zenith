# Alpha 2a — Core Fixes (Compose) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the audit's 10-step core scenario pass on a fresh Compose install, through UI, API and CLI.

**Architecture:** Fix the shared root causes, not each symptom.
- A standalone-only plan repository returns an unlimited plan.
- The deployment row records its image.
- `DockerDeployer` merges env tables, routes custom domains and names the `le` resolver.
- New Docker-backed database and log sources sit behind small interfaces.
- The web calls the API on its own origin.
- Traefik is always on with Let's Encrypt, and `ZENITH_DOMAIN` defaults to `<ip>.sslip.io`.

**Tech Stack:** Go 1.25 (Fiber v2, Docker SDK `github.com/docker/docker/client`, pgx), Next.js 15 / React 19 / Vitest, Docker Compose, Traefik v3.7, bash.

**Spec:** `docs/superpowers/specs/2026-10-12-alpha-2a-core-fixes-design.md` (parent: `docs/superpowers/specs/2026-10-12-alpha-release-design.md`)

## Global Constraints

- Self-hosted only; SaaS code is **not deleted**. In standalone mode it is routed around (as `handlers.UnlimitedMode` does).
- **Only ports 80 and 443 are public.** No other host port is opened by the stack, deployed apps, or database containers. 3000/8080 bind to `127.0.0.1`.
- One TLS mode: HTTPS via Let's Encrypt, resolver name `le`. Every TLS router names `tls.certresolver=le` explicitly.
- `ZENITH_DOMAIN` defaults to `<public-ip>.sslip.io`. Apps live at `<sub>.<ZENITH_DOMAIN>`.
- Database images: `postgres:16-alpine` (5432), `mysql:8.4` (3306), `redis:7-alpine` (6379). Containers are named `zenith-db-<first 8 chars of id>`, attached to network `zenith_default`, with restart `unless-stopped`, volume `zenith-db-<first 8 chars of id>`, and label `zenith.db.id=<id>`.
- Commit identity: `git -c user.name="Babak Doraniarab" -c user.email="babak.dorani@gmail.com" commit`, and every message ends with:
  ```
  Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_01T4AidohHj3qvUYek6RLV3F
  ```
- Branch: `feat/alpha-2a-core-fixes`. Go tests: `cd services/api && go test ./...`. Web tests: `cd apps/web && pnpm vitest run`. The 3 existing `tsc` errors in `use-auth.test.ts` are known (F-03) and out of scope.
- Docker-backed Go tests carry `//go:build docker` and run with `go test -tags docker ./internal/services/deploy/...` on a host with Docker.
- Never print the admin password or tokens.

## Review Focus

1. **App created before any custom domain, then a domain added.** The running container must be recreated with the new Host rule. Test: Task 6, `TestDomainAdd_RedeploysWithCurrentImage`.
2. **Env var with the same key in the legacy table and in v2.** The v2 value (UI) wins; an environment-scoped v2 value beats the app-wide one. Test: Task 5, `TestResolveEnvVars_Precedence`.
3. **Database container that never becomes healthy** (bad image, OOM). Status must end `error`, not stay `provisioning` forever. Test: Task 9, `TestDBService_ProvisionFailureMarksError`.
4. **API restarted between deploy and opening the deploy log.** `/logs/history` must still return the lines. Test: Task 11, `TestLogHistory_FallsBackToBuildLog`.
5. **Runtime `NEXT_PUBLIC_API_URL` empty and build-time value empty.** The browser must call its own origin, never a `freezenith.com` host. Test: Task 2, `resolveApiBaseUrl` cases.

---

## File Structure

| File | Responsibility |
|---|---|
| `services/api/internal/adapters/memory/unlimited_plan.go` (new) | `UnlimitedPlanRepository` wrapper: unlimited plan in standalone |
| `services/api/cmd/server/main.go` (modify) | wiring only |
| `services/api/internal/ports/repositories.go` (modify) | `SetDeploymentImage`, DB credential methods |
| `services/api/internal/entities/deployment.go` (modify) | `Deployment.DeployedImage()` |
| `services/api/internal/services/deploy/pipeline.go` (modify) | persist image, error key, build_log |
| `services/api/internal/services/deploy/docker_deployer.go` (modify) | status, env merge, domains, resolver, log lines |
| `services/api/internal/services/deploy/current_image.go` (new) | `CurrentImage` helper shared by env apply + domain refresh |
| `services/api/internal/services/deploy/docker_logs.go` (new) | `DockerLogSource` (container logs) |
| `services/api/internal/services/deploy/docker_db.go` (new) | `DockerDBBackend` (database containers) |
| `services/api/internal/services/db_runtime.go` (new) | `DBRuntimeService`: create → provision → credentials |
| `services/api/internal/adapters/postgres/migrations/050_db_password.{up,down}.sql` (new) | `password_enc` column |
| `services/api/internal/handlers/{env_var,domain,apps_v2,databases_v2,logs}.go` (modify) | use the above |
| `services/api/internal/services/monitoring.go` (modify) | optional `AppLogSource` |
| `services/api/internal/config/config.go` (modify) | k8s URLs empty in standalone; `CertResolver` |
| `services/api/cmd/server/storage_bootstrap.go` (new) | ensure platform bucket |
| `apps/web/src/lib/runtime-env.ts`, `apps/web/Dockerfile` (modify) | same-origin API |
| `apps/web/src/lib/edition.ts` (new) + pages (modify) | hide SaaS upsell in standalone |
| `cli/internal/api/client.go` (modify) | DB endpoints |
| `docker-compose.yml`, `infra/scripts/install.sh`, `.env.example` (modify) | install + Traefik |
| `infra/scripts/install_test.sh` (new) | installer render test |
| `docs/alpha/core-audit.md` (modify) | §6 re-run |

---

### Task 1: Unlimited plan in standalone (F-06)

**Files:**
- Create: `services/api/internal/adapters/memory/unlimited_plan.go`
- Create: `services/api/internal/adapters/memory/unlimited_plan_test.go`
- Modify: `services/api/cmd/server/main.go` (the `planRepo` block near line 319)

**Interfaces:**
- Produces: `memory.NewUnlimitedPlanRepository(inner ports.UserPlanRepository) *UnlimitedPlanRepository`. `GetUserPlan` returns tier `enterprise` with `entities.DefaultPlanLimits(entities.PlanEnterprise)`.

- [ ] **Step 1: Write the failing test**

```go
// services/api/internal/adapters/memory/unlimited_plan_test.go
package memory_test

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestUnlimitedPlan_AlwaysEnterprise(t *testing.T) {
	inner := memory.NewMemoryUserPlanRepository()
	repo := memory.NewUnlimitedPlanRepository(inner)

	p, err := repo.GetUserPlan(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if p.Tier != entities.PlanEnterprise {
		t.Errorf("tier = %q, want enterprise", p.Tier)
	}
	if !p.Limits.CustomDomain || p.Limits.MaxRedisInstances == 0 || p.Limits.MaxApps < 1000 {
		t.Errorf("limits not unlimited: %+v", p.Limits)
	}
	if p.UserID != "user-1" {
		t.Errorf("user id = %q", p.UserID)
	}
}

func TestUnlimitedPlan_IgnoresStoredFreePlan(t *testing.T) {
	inner := memory.NewMemoryUserPlanRepository()
	_, _ = inner.SetUserPlan(context.Background(), "user-1", entities.PlanFree)
	repo := memory.NewUnlimitedPlanRepository(inner)
	p, _ := repo.GetUserPlan(context.Background(), "user-1")
	if p.Tier != entities.PlanEnterprise {
		t.Errorf("tier = %q, want enterprise", p.Tier)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/api && go test ./internal/adapters/memory/ -run TestUnlimitedPlan`
Expected: FAIL to compile (`undefined: memory.NewUnlimitedPlanRepository`).

- [ ] **Step 3: Implement**

```go
// services/api/internal/adapters/memory/unlimited_plan.go
package memory

import (
	"context"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// UnlimitedPlanRepository is used in standalone self-host mode. The customer
// owns the hardware, so SaaS plan tiers do not apply: every user reads as
// enterprise with enterprise limits. Writes and listings pass through so the
// (unused) SaaS admin code keeps working.
type UnlimitedPlanRepository struct {
	inner ports.UserPlanRepository
}

// NewUnlimitedPlanRepository wraps inner.
func NewUnlimitedPlanRepository(inner ports.UserPlanRepository) *UnlimitedPlanRepository {
	return &UnlimitedPlanRepository{inner: inner}
}

func (r *UnlimitedPlanRepository) GetUserPlan(_ context.Context, userID string) (*entities.UserPlan, error) {
	return &entities.UserPlan{
		UserID: userID,
		Tier:   entities.PlanEnterprise,
		Limits: entities.DefaultPlanLimits(entities.PlanEnterprise),
	}, nil
}

func (r *UnlimitedPlanRepository) SetUserPlan(ctx context.Context, userID string, tier entities.PlanTier) (*entities.UserPlan, error) {
	return r.inner.SetUserPlan(ctx, userID, tier)
}

func (r *UnlimitedPlanRepository) ListUsersByPlan(ctx context.Context, tier entities.PlanTier) ([]entities.UserPlan, error) {
	return r.inner.ListUsersByPlan(ctx, tier)
}

func (r *UnlimitedPlanRepository) ListAllPlans(ctx context.Context) ([]entities.UserPlan, error) {
	return r.inner.ListAllPlans(ctx)
}

var _ ports.UserPlanRepository = (*UnlimitedPlanRepository)(nil)
```

- [ ] **Step 4: Wire in main.go**

Directly after the existing block:
```go
	var planRepo ports.UserPlanRepository
	if pool != nil {
		planRepo = postgres.NewPostgresUserPlanRepository(pool)
	} else {
		planRepo = memory.NewMemoryUserPlanRepository()
	}
```
add:
```go
	// Self-host: no plan tiers. Every caller of GetUserPlan (domains, MFA,
	// webhooks, Redis limits, …) sees enterprise limits.
	if cfg.Mode != "saas" {
		planRepo = memory.NewUnlimitedPlanRepository(planRepo)
	}
```

- [ ] **Step 5: Run tests**

Run: `cd services/api && go test ./internal/adapters/memory/ -run TestUnlimitedPlan && go build ./... && go test ./internal/handlers/ ./cmd/server/`
Expected: PASS, build ok.

- [ ] **Step 6: Commit**

```bash
git add services/api/internal/adapters/memory/unlimited_plan*.go services/api/cmd/server/main.go
git commit -m "fix(api): standalone mode reports an unlimited plan to every caller (F-06)"
```
(Use the commit identity and trailer from Global Constraints. This applies to every commit step below.)

---

### Task 2: Web calls the API on its own origin (F-05)

**Files:**
- Modify: `apps/web/src/lib/runtime-env.ts`
- Create: `apps/web/src/lib/__tests__/runtime-env.test.ts`
- Modify: `apps/web/Dockerfile:26`

**Interfaces:**
- Produces: `resolveApiBaseUrl(runtime: string | undefined, buildTime: string | undefined, origin: string | undefined): string`, exported from `runtime-env.ts`. `API_BASE_URL` keeps its name and type (`string`).

- [ ] **Step 1: Write the failing test**

```ts
// apps/web/src/lib/__tests__/runtime-env.test.ts
import { describe, it, expect } from "vitest";
import { resolveApiBaseUrl } from "../runtime-env";

describe("resolveApiBaseUrl", () => {
  it("runtime value wins", () => {
    expect(resolveApiBaseUrl("https://api.example.com", "https://build", "https://origin")).toBe("https://api.example.com");
  });
  it("build-time value when runtime is empty", () => {
    expect(resolveApiBaseUrl("", "https://build", "https://origin")).toBe("https://build");
  });
  it("browser origin when runtime and build-time are empty", () => {
    expect(resolveApiBaseUrl("", "", "https://zenith.example.com")).toBe("https://zenith.example.com");
    expect(resolveApiBaseUrl(undefined, undefined, "http://1.2.3.4")).toBe("http://1.2.3.4");
  });
  it("server fallback when nothing is known", () => {
    expect(resolveApiBaseUrl(undefined, undefined, undefined)).toBe("http://localhost:8080");
  });
  it("strips a trailing slash", () => {
    expect(resolveApiBaseUrl("https://a.example.com/", "", "")).toBe("https://a.example.com");
  });
  it("never returns a freezenith host unless it was configured", () => {
    expect(resolveApiBaseUrl("", "", "https://my.box")).not.toContain("freezenith");
  });
});
```

- [ ] **Step 2: Run it**

Run: `cd apps/web && pnpm vitest run src/lib/__tests__/runtime-env.test.ts`
Expected: FAIL (`resolveApiBaseUrl is not a function` / not exported).

- [ ] **Step 3: Implement**

In `apps/web/src/lib/runtime-env.ts`, replace:
```ts
export const API_BASE_URL = getEnv(
  "NEXT_PUBLIC_API_URL",
  process.env.NEXT_PUBLIC_API_URL,
  "http://localhost:8080"
);
```
with:
```ts
/**
 * API base URL. Order: runtime env.js value, build-time value, the page's own
 * origin (self-host: Traefik routes /api on the dashboard host), then a
 * server-side fallback. Empty strings count as unset.
 */
export function resolveApiBaseUrl(
  runtime: string | undefined,
  buildTime: string | undefined,
  origin: string | undefined
): string {
  const pick = [runtime, buildTime, origin].find((v) => v && v.trim() !== "");
  return (pick ?? "http://localhost:8080").replace(/\/+$/, "");
}

export const API_BASE_URL = resolveApiBaseUrl(
  typeof window !== "undefined" ? window.__ENV?.NEXT_PUBLIC_API_URL : undefined,
  process.env.NEXT_PUBLIC_API_URL,
  typeof window !== "undefined" ? window.location.origin : undefined
);
```

In `apps/web/Dockerfile`, change line 26:
```dockerfile
ARG NEXT_PUBLIC_API_URL=
```
(`build-images.yml` and `deploy-staging.yml` already pass the stage URL explicitly. Do not touch them.)

- [ ] **Step 4: Run tests**

Run: `cd apps/web && pnpm vitest run && pnpm exec tsc --noEmit 2>&1 | grep -c 'error TS'`
Expected: all vitest files pass; tsc count `3` (the known F-03 errors only).

- [ ] **Step 5: Commit**

```bash
git add apps/web/src/lib/runtime-env.ts apps/web/src/lib/__tests__/runtime-env.test.ts apps/web/Dockerfile
git commit -m "fix(web): call the API on the page's own origin; no baked stage URL (F-05)"
```

---

### Task 3: Hide SaaS upsell in standalone; real hosts in Networking (F-11)

**Files:**
- Create: `apps/web/src/lib/edition.ts`
- Create: `apps/web/src/lib/__tests__/edition.test.ts`
- Modify: `apps/web/src/app/page.tsx` (plan banner ~288-330, referral banner ~458-469)
- Modify: `apps/web/src/components/upgrade-nudge.tsx:13`
- Modify: `apps/web/src/app/settings/page.tsx:52` (referral tab), `:397`, `:638`
- Modify: `apps/web/src/components/onboarding-wizard.tsx:150-160`
- Modify: `apps/web/src/app/networking/page.tsx:35`
- Modify: `apps/web/src/app/apps/[name]/page.tsx:1790` and the "Requires Pro plan or higher." line in the Domains tab

**Interfaces:**
- Produces, from `@/lib/edition`:
  - `SHOW_PLANS: boolean`, which is `!IS_STANDALONE`;
  - `appHost(app: { url?: string; subdomain?: string }): string`;
  - `customDomainInstruction(): string`.

- [ ] **Step 1: Write the failing test**

```ts
// apps/web/src/lib/__tests__/edition.test.ts
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true }));
import { SHOW_PLANS, appHost, customDomainInstruction } from "../edition";

describe("edition (standalone)", () => {
  it("hides plans", () => {
    expect(SHOW_PLANS).toBe(false);
  });
  it("uses the host from the app URL", () => {
    expect(appHost({ url: "https://whoami-1a2b.zenith.example.com", subdomain: "whoami-1a2b" })).toBe("whoami-1a2b.zenith.example.com");
  });
  it("falls back to the subdomain when the URL is missing or bad", () => {
    expect(appHost({ subdomain: "web-9" })).toBe("web-9");
    expect(appHost({ url: "not a url", subdomain: "web-9" })).toBe("web-9");
  });
  it("tells self-hosters to use an A record, never freezenith", () => {
    const s = customDomainInstruction();
    expect(s).toMatch(/A record/);
    expect(s).not.toMatch(/freezenith|CNAME/);
  });
});
```

- [ ] **Step 2: Run it**

Run: `cd apps/web && pnpm vitest run src/lib/__tests__/edition.test.ts`
Expected: FAIL (cannot resolve `../edition`).

- [ ] **Step 3: Implement `edition.ts`**

```ts
// apps/web/src/lib/edition.ts
import { IS_STANDALONE } from "@/lib/runtime-env";

/** Plans, upgrades and referrals exist only in the SaaS edition. */
export const SHOW_PLANS = !IS_STANDALONE;

/** Public host of a deployed app, taken from the URL the API reports. */
export function appHost(app: { url?: string; subdomain?: string }): string {
  if (app.url) {
    try {
      return new URL(app.url).host;
    } catch {
      /* fall through */
    }
  }
  return app.subdomain ?? "";
}

/** DNS instruction shown when adding a custom domain. */
export function customDomainInstruction(): string {
  return "First, add an A record pointing your domain to this server's public IP (the same IP as this dashboard).";
}
```

- [ ] **Step 4: Apply it in the pages**

- `app/page.tsx`:
  - Change `{plan && (` (the "Plan usage banner") to `{SHOW_PLANS && plan && (`.
  - Wrap the whole `{/* Referral banner */}` `<div …>…</div>` in `{SHOW_PLANS && ( … )}`.
  - Add `import { SHOW_PLANS } from "@/lib/edition";`.
- `components/upgrade-nudge.tsx`:
  - The first line of `UpgradeNudge` becomes `if (!SHOW_PLANS || current < limit) return null;`.
  - Add the import.
- `app/settings/page.tsx`:
  - Line 52: build the tab list as `[...].filter((t) => SHOW_PLANS || t.key !== "referral")` on the array that contains `{ key: "referral", … }`.
  - Lines 397 and 638: wrap the sentence `Requires Pro plan or higher.` in `{SHOW_PLANS && " Requires Pro plan or higher."}`.
- `components/onboarding-wizard.tsx`:
  - At :150 replace the paragraph text with `{SHOW_PLANS ? "Your app is available at a free Zenith subdomain. Upgrade to Pro for custom domains." : "Your app already has its own subdomain. You can add your own domain from the app's Domains tab."}`.
  - Wrap the `<Link href="/billing" …>Upgrade to Pro for custom domains</Link>` in `{SHOW_PLANS && ( … )}`.
- `app/networking/page.tsx:35`: replace `` domain: `${a.subdomain}.apps.stage.freezenith.com`, `` with `domain: appHost(a),`, and import `appHost`.
- `app/apps/[name]/page.tsx`:
  - At :1790 replace the CNAME line contents with `{customDomainInstruction()}`.
  - Replace the Domains tab's `Requires Pro plan or higher.` text with `{SHOW_PLANS && "Requires Pro plan or higher."}`.
  - Import both.

- [ ] **Step 5: Run tests**

Run: `cd apps/web && pnpm vitest run && pnpm exec tsc --noEmit 2>&1 | grep -c 'error TS' && grep -rn 'apps.stage.freezenith\|freezenith.com</code>\|{appId}.freezenith' src/app src/components | grep -v demo-api`
Expected: vitest all pass; tsc `3`; the grep prints nothing.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src
git commit -m "fix(web): hide plans/referrals in standalone; real app hosts and A-record domain hint (F-11)"
```

---

### Task 4: Deployments record their image; app status and failure error are correct (F-14, F-20)

**Files:**
- Modify: `services/api/internal/entities/deployment.go`
- Modify: `services/api/internal/ports/repositories.go:137` (add method)
- Modify: `services/api/internal/adapters/memory/memory_app.go`, `services/api/internal/adapters/postgres/postgres_app.go`, `services/api/internal/middleware/ownership_test.go` (mock)
- Modify: `services/api/internal/services/deploy/pipeline.go` (`TriggerImageDeploy`, `emitEvent`)
- Modify: `services/api/internal/services/deploy/docker_deployer.go` (`DeployApp` end)
- Modify: `services/api/internal/handlers/deploy.go:105` (rollback uses `DeployedImage()`)
- Test: `services/api/internal/services/deploy/pipeline_image_test.go` (new), `services/api/internal/entities/deployment_test.go` (new)

**Interfaces:**
- Produces:
  - `AppRepository.SetDeploymentImage(ctx context.Context, id, image string) error`;
  - `(*entities.Deployment).DeployedImage() string`, which returns `ImageTag`, or `GitSHA` when `ImageTag` is empty (old rows wrote the image there).

- [ ] **Step 1: Write the failing tests**

```go
// services/api/internal/entities/deployment_test.go
package entities

import "testing"

func TestDeployedImage(t *testing.T) {
	if got := (&Deployment{ImageTag: "a:1", GitSHA: "b:2"}).DeployedImage(); got != "a:1" {
		t.Errorf("got %q", got)
	}
	if got := (&Deployment{GitSHA: "docker.io/traefik/whoami:latest"}).DeployedImage(); got != "docker.io/traefik/whoami:latest" {
		t.Errorf("got %q", got)
	}
}
```

```go
// services/api/internal/services/deploy/pipeline_image_test.go
package deploy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type stubBackend struct{ err error }

func (s *stubBackend) DeployApp(context.Context, *entities.App, string) error { return s.err }
func (s *stubBackend) DeleteApp(context.Context, *entities.App) error      { return nil }

func runDeploy(t *testing.T, backendErr error) (*memory.MemoryAppRepository, *entities.App, *entities.Deployment) {
	t.Helper()
	repo := memory.NewMemoryAppRepository()
	app, err := repo.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "u1", Name: "web", ImageURL: "nginx:1"})
	if err != nil {
		t.Fatal(err)
	}
	dep, _ := repo.CreateDeployment(context.Background(), app.ID, "nginx:1")
	p := NewPipeline(&stubBackend{err: backendErr}, repo, NewLogHub(50), nil, 5)
	if err := p.TriggerImageDeploy(app, dep, "nginx:1"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for p.RunningCount() > 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	return repo, app, dep
}

func TestPipeline_RecordsImageAndRunningStatus(t *testing.T) {
	repo, app, dep := runDeploy(t, nil)
	got, _ := repo.GetDeployment(context.Background(), dep.ID)
	if got.ImageTag != "nginx:1" {
		t.Errorf("image_tag = %q, want nginx:1", got.ImageTag)
	}
	a, _ := repo.GetApp(context.Background(), app.ID)
	if a.Status != entities.AppStatusRunning {
		t.Errorf("app status = %q, want running", a.Status)
	}
}

func TestPipeline_FailureMarksAppFailed(t *testing.T) {
	repo, app, dep := runDeploy(t, errors.New("pull denied"))
	a, _ := repo.GetApp(context.Background(), app.ID)
	if a.Status != entities.AppStatusFailed {
		t.Errorf("app status = %q, want failed", a.Status)
	}
	got, _ := repo.GetDeployment(context.Background(), dep.ID)
	if got.Error == "" {
		t.Error("deployment error not recorded")
	}
}

func TestPipeline_FailedEventCarriesError(t *testing.T) {
	data := deployEventData(&entities.App{ID: "a", Name: "web"}, &entities.Deployment{ID: "d"}, "img", "Deploy failed: boom", EventDeployFailed)
	if data["error"] != "Deploy failed: boom" {
		t.Errorf("error key = %v", data["error"])
	}
}
```

- [ ] **Step 2: Run them**

Run: `cd services/api && go test ./internal/entities/ -run TestDeployedImage; go test ./internal/services/deploy/ -run 'TestPipeline_(Records|Failure|FailedEvent)'`
Expected: compile errors: `DeployedImage` undefined; `deployEventData` undefined.

- [ ] **Step 3: Implement**

`entities/deployment.go`, after the `Deployment` struct:
```go
// DeployedImage is the image this deployment runs. Rows written before
// image_tag was recorded carry the image in git_sha.
func (d *Deployment) DeployedImage() string {
	if d.ImageTag != "" {
		return d.ImageTag
	}
	return d.GitSHA
}
```

`ports/repositories.go`, below `UpdateDeploymentStatus(...)`:
```go
	SetDeploymentImage(ctx context.Context, id, image string) error
```

`memory/memory_app.go`:
```go
func (r *MemoryAppRepository) SetDeploymentImage(_ context.Context, id, image string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deployments[id]
	if !ok {
		return fmt.Errorf("deployment not found")
	}
	d.ImageTag = image
	return nil
}
```

`postgres/postgres_app.go`:
```go
func (r *PostgresAppRepository) SetDeploymentImage(ctx context.Context, id, image string) error {
	if _, err := r.pool.Exec(ctx, `UPDATE deployments SET image_tag = $1 WHERE id = $2`, image, id); err != nil {
		return fmt.Errorf("set deployment image: %w", err)
	}
	return nil
}
```

`middleware/ownership_test.go` mock:
```go
func (m *mockAppRepository) SetDeploymentImage(_ context.Context, _, _ string) error { return nil }
```

`pipeline.go`, in the goroutine of `TriggerImageDeploy`, right after `p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID, entities.DeployStatusDeploying, "", "")`:
```go
		_ = p.appRepo.SetDeploymentImage(ctx, deployment.ID, image)
```
In the same goroutine, replace the success path right after the `if p.deployer != nil { … } else { … }` block. That is, before `p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID, entities.DeployStatusActive, "", "")`, add:
```go
		runningStatus := entities.AppStatusRunning
		p.appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{Status: &runningStatus})
```
Then delete the now-duplicate `runningStatus` lines inside the `else` branch (keep the `else` comment, or remove the empty `else`).

Refactor `emitEvent`'s data map into a function and use it:
```go
// deployEventData is the payload of a deploy platform event. Failure events
// carry "error" so notification consumers can log the cause.
func deployEventData(app *entities.App, deployment *entities.Deployment, image, message string, eventType EventType) map[string]interface{} {
	data := map[string]interface{}{
		"app_id":        app.ID,
		"app_name":      app.Name,
		"deployment_id": deployment.ID,
		"image":         image,
		"message":       message,
	}
	if eventType == EventDeployFailed {
		data["error"] = message
	}
	return data
}
```
In `emitEvent`, replace the inline `Data: map[string]interface{}{…}` with `Data: deployEventData(app, deployment, image, message, eventType),`.

`handlers/deploy.go:105`: change `target.ImageTag` to `target.DeployedImage()`.

- [ ] **Step 4: Run tests**

Run: `cd services/api && go build ./... && go test ./internal/entities/ ./internal/services/deploy/ ./internal/handlers/ ./internal/middleware/ ./internal/adapters/memory/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/api
git commit -m "fix(api): deployments record their image; app goes running/failed; failure events carry the error (F-14, F-20)"
```

---

### Task 5: Env vars reach the container; env apply works in Docker mode (F-15, F-17)

**Files:**
- Modify: `services/api/internal/services/deploy/docker_deployer.go` (`resolveEnvVars`)
- Create: `services/api/internal/services/deploy/current_image.go`
- Modify: `services/api/internal/handlers/env_var.go` (`Apply`)
- Modify: `services/api/cmd/server/main.go:1284` (`SetRestarter(backend)`)
- Test: `services/api/internal/services/deploy/docker_env_test.go` (new), `services/api/internal/services/deploy/current_image_test.go` (new), `services/api/internal/handlers/env_var_test.go` (add test)

**Interfaces:**
- Consumes: `Deployment.DeployedImage()` (Task 4).
- Produces: `deploy.CurrentImage(ctx context.Context, repo ports.AppRepository, appID string) (string, error)`. It returns the image of the active deployment, else the newest deployment, else the newest release. It returns `deploy.ErrNoImage` when none exists.

- [ ] **Step 1: Write the failing tests**

```go
// services/api/internal/services/deploy/docker_env_test.go
package deploy

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestResolveEnvVars_Precedence(t *testing.T) {
	ctx := context.Background()
	appRepo := memory.NewMemoryAppRepository()
	envRepo := memory.NewMemoryEnvVarRepository()
	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{UserID: "u1", Name: "web", ImageURL: "nginx:1"})
	app.EnvironmentID = "env-prod"

	// legacy table: compose import / db attach
	_ = appRepo.SetEnvVars(ctx, app.ID, map[string]string{"POSTGRES_PASSWORD": "audit", "SHARED": "legacy", "ONLY_LEGACY": "1"})
	// v2 app-wide (UI Environment tab without env scope)
	_ = envRepo.SetEnvVar(ctx, &entities.AppEnvVar{ID: "v1", AppID: app.ID, Key: "SHARED", Value: "app-wide"})
	_ = envRepo.SetEnvVar(ctx, &entities.AppEnvVar{ID: "v2", AppID: app.ID, Key: "AUDIT", Value: "1"})
	// v2 env-scoped
	_ = envRepo.SetEnvVar(ctx, &entities.AppEnvVar{ID: "v3", AppID: app.ID, EnvironmentID: "env-prod", Key: "AUDIT", Value: "2"})

	d := &DockerDeployer{appRepo: appRepo, envVarRepo: envRepo}
	vars, err := d.resolveEnvVars(ctx, app)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, v := range vars {
		got[v.Key] = v.Value
	}
	want := map[string]string{"POSTGRES_PASSWORD": "audit", "ONLY_LEGACY": "1", "SHARED": "app-wide", "AUDIT": "2"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d vars, want %d: %v", len(got), len(want), got)
	}
}
```

```go
// services/api/internal/services/deploy/current_image_test.go
package deploy

import (
	"context"
	"errors"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestCurrentImage(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMemoryAppRepository()
	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{UserID: "u1", Name: "web", ImageURL: "nginx:1"})

	if _, err := CurrentImage(ctx, repo, app.ID); !errors.Is(err, ErrNoImage) {
		t.Fatalf("want ErrNoImage, got %v", err)
	}

	d1, _ := repo.CreateDeployment(ctx, app.ID, "nginx:1")
	_ = repo.UpdateDeploymentStatus(ctx, d1.ID, entities.DeployStatusActive, "", "")
	d2, _ := repo.CreateDeployment(ctx, app.ID, "nginx:2")
	_ = repo.SetDeploymentImage(ctx, d2.ID, "nginx:2")
	_ = repo.UpdateDeploymentStatus(ctx, d2.ID, entities.DeployStatusFailed, "", "x")

	img, err := CurrentImage(ctx, repo, app.ID)
	if err != nil || img != "nginx:1" {
		t.Fatalf("got %q, %v; want the active deployment's image nginx:1", img, err)
	}
}
```

Add to `services/api/internal/handlers/env_var_test.go`:
```go
type recordingRestarter struct{ image string }

func (r *recordingRestarter) DeployApp(_ context.Context, _ *entities.App, image string) error {
	r.image = image
	return nil
}

func TestEnvVarApply_UsesActiveDeployment(t *testing.T) {
	fiberApp, handler, appRepo, _ := setupEnvVarTest()
	appID := createAppForEnvVarTest(t, appRepo)
	d, _ := appRepo.CreateDeployment(context.Background(), appID, "docker.io/traefik/whoami:latest")
	_ = appRepo.UpdateDeploymentStatus(context.Background(), d.ID, entities.DeployStatusActive, "", "")
	rec := &recordingRestarter{}
	handler.SetRestarter(rec)
	fiberApp.Post("/api/v1/apps/:appId/env-v2/apply", injectUserID("user-1"), handler.Apply)

	resp, _ := fiberApp.Test(httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/env-v2/apply", nil), -1)
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if rec.image != "docker.io/traefik/whoami:latest" {
		t.Errorf("redeployed image = %q", rec.image)
	}
}
```
(Add the imports `context` and `github.com/dotechhq/zenith/services/api/internal/entities` to that file.)

- [ ] **Step 2: Run them**

Run: `cd services/api && go test ./internal/services/deploy/ -run 'TestResolveEnvVars_Precedence|TestCurrentImage'; go test ./internal/handlers/ -run TestEnvVarApply_UsesActiveDeployment`
Expected:
- precedence test FAILs (the legacy vars are missing, `got 3 vars, want 4` or similar);
- `CurrentImage` fails to compile;
- the apply test FAILs with status 409.

- [ ] **Step 3: Implement `current_image.go`**

```go
// services/api/internal/services/deploy/current_image.go
package deploy

import (
	"context"
	"errors"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// ErrNoImage means the app has never been deployed with a known image.
var ErrNoImage = errors.New("no deployed image for app")

// CurrentImage returns the image the app is running: the active deployment's,
// else the newest deployment's, else the newest release's.
func CurrentImage(ctx context.Context, repo ports.AppRepository, appID string) (string, error) {
	if d, err := repo.GetActiveDeployment(ctx, appID); err == nil && d != nil && d.DeployedImage() != "" {
		return d.DeployedImage(), nil
	}
	if ds, err := repo.ListDeployments(ctx, appID, 20); err == nil {
		for _, d := range ds {
			if d.Status != entities.DeployStatusFailed && d.DeployedImage() != "" {
				return d.DeployedImage(), nil
			}
		}
	}
	if rs, err := repo.ListReleases(ctx, appID, 1); err == nil && len(rs) > 0 && rs[0].Image != "" {
		return rs[0].Image, nil
	}
	return "", ErrNoImage
}
```

- [ ] **Step 4: Replace `resolveEnvVars` in `docker_deployer.go`**

```go
// resolveEnvVars builds the container env from three sources, later ones
// overriding earlier ones by key:
//  1. legacy app_env_vars (app create, compose import, database attach)
//  2. v2 app-wide vars (environment_id NULL; the UI Environment tab)
//  3. v2 vars scoped to the app's environment
// Secrets are decrypted; a secret that fails to decrypt is skipped.
func (d *DockerDeployer) resolveEnvVars(ctx context.Context, app *entities.App) ([]entities.EnvVar, error) {
	merged := map[string]string{}
	var order []string
	put := func(k, v string) {
		if _, ok := merged[k]; !ok {
			order = append(order, k)
		}
		merged[k] = v
	}
	decrypt := func(key, value string) (string, bool) {
		if d.envCrypto != nil && pkgCrypto.IsEncrypted(value) {
			dec, err := d.envCrypto.Decrypt(app.UserID, value)
			if err != nil {
				slog.Error("docker-deploy: decrypt env var failed, skipping", "key", key, "error", err)
				return "", false
			}
			return dec, true
		}
		return value, true
	}

	legacy, err := d.appRepo.GetEnvVars(ctx, app.ID)
	if err != nil {
		return nil, fmt.Errorf("docker-deploy: get env vars: %w", err)
	}
	for _, v := range legacy {
		if val, ok := decrypt(v.Key, v.Value); ok {
			put(v.Key, val)
		}
	}

	if d.envVarRepo != nil {
		scopes := []string{""}
		if app.EnvironmentID != "" {
			scopes = append(scopes, app.EnvironmentID)
		}
		for _, envID := range scopes {
			vars, err := d.envVarRepo.GetEnvVarsByEnvironment(ctx, app.ID, envID)
			if err != nil {
				return nil, fmt.Errorf("docker-deploy: get env vars: %w", err)
			}
			for _, v := range vars {
				if val, ok := decrypt(v.Key, v.Value); ok {
					put(v.Key, val)
				}
			}
		}
	}

	out := make([]entities.EnvVar, 0, len(order))
	for _, k := range order {
		out = append(out, entities.EnvVar{AppID: app.ID, Key: k, Value: merged[k]})
	}
	return out, nil
}
```

- [ ] **Step 5: Fix `Apply` in `handlers/env_var.go`**

Replace the block from `// Get current image from latest release` through `imageTag := releases[0].Image` with:
```go
	imageTag, err := deploy.CurrentImage(c.Context(), h.appRepo, appID)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "no deployment found — deploy the app first before applying env changes",
		})
	}
```
Add the import `"github.com/dotechhq/zenith/services/api/internal/services/deploy"`. If that creates an import cycle, check with `go build ./...` and move `CurrentImage` and `ErrNoImage` into `internal/services/currentimage.go` (package `services`) instead, keeping the same names. Record the move as a ruling.

In `cmd/server/main.go`, replace `envVarHandler.SetRestarter(deployer)` with `envVarHandler.SetRestarter(backend)`.

- [ ] **Step 6: Run tests**

Run: `cd services/api && go build ./... && go test ./internal/services/deploy/ ./internal/handlers/`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/api
git commit -m "fix(api): container env merges legacy+v2 vars; env apply redeploys the active image via the runtime backend (F-15, F-17)"
```

---

### Task 6: Custom domains routed with Let's Encrypt; app routers name the resolver (F-12, F-18)

**Files:**
- Modify: `services/api/internal/services/deploy/docker_deployer.go` (`proxyLabels`, `DeployApp`, new setters)
- Modify: `services/api/internal/services/deploy/docker_deployer_labels_test.go`
- Modify: `services/api/internal/handlers/domain.go` (`refreshIngress`, `Delete`)
- Modify: `services/api/internal/config/config.go` (`CertResolver`)
- Modify: `services/api/cmd/server/main.go` (wiring)
- Test: `services/api/internal/handlers/domain_test.go` (add test)

**Interfaces:**
- Consumes: `deploy.CurrentImage` (Task 5).
- Produces:
  - `proxyLabels(app *entities.App, hosts []string, network, certResolver string) map[string]string`;
  - `(*DockerDeployer).SetDomainRepo(ports.DomainRepository)` and `(*DockerDeployer).SetCertResolver(string)`;
  - `config.Config.CertResolver string`, from env `ZENITH_CERT_RESOLVER`, default `"le"`.

- [ ] **Step 1: Update the label test (RED)**

Replace `TestProxyLabels_Traefik` in `docker_deployer_labels_test.go` with:
```go
func TestProxyLabels_Traefik(t *testing.T) {
	app := &entities.App{ID: "app-123", Subdomain: "my-app", Port: 8000}
	labels := proxyLabels(app, []string{"my-app.apps.example.com", "shop.customer.com"}, "zenith_default", "le")

	want := map[string]string{
		"zenith.app.id":          "app-123",
		"traefik.enable":         "true",
		"traefik.docker.network": "zenith_default",
		"traefik.http.routers.my-app.rule":                      "Host(`my-app.apps.example.com`) || Host(`shop.customer.com`)",
		"traefik.http.routers.my-app.entrypoints":               "websecure",
		"traefik.http.routers.my-app.tls":                       "true",
		"traefik.http.routers.my-app.tls.certresolver":          "le",
		"traefik.http.services.my-app.loadbalancer.server.port": "8000",
	}
	for k, v := range want {
		if labels[k] != v {
			t.Errorf("label %q = %q, want %q", k, labels[k], v)
		}
	}
	for k := range labels {
		if strings.HasPrefix(k, "caddy") {
			t.Errorf("unexpected leftover caddy label %q", k)
		}
	}
}

func TestProxyLabels_NoResolverWhenEmpty(t *testing.T) {
	app := &entities.App{ID: "a", Subdomain: "x", Port: 80}
	labels := proxyLabels(app, []string{"x.example.com"}, "zenith_default", "")
	if _, ok := labels["traefik.http.routers.x.tls.certresolver"]; ok {
		t.Error("empty resolver must not emit a certresolver label")
	}
}
```
Delete any other test in that file that calls the old `proxyLabels(app, baseDomain, network)` signature, or update its arguments to `[]string{app.Subdomain + "." + baseDomain}, network, "le"`.

Add to `services/api/internal/handlers/domain_test.go`:
```go
type recordingRedeployer struct{ image string }

func (r *recordingRedeployer) DeployApp(_ context.Context, _ *entities.App, image string) error {
	r.image = image
	return nil
}

func TestDomainAdd_RedeploysWithCurrentImage(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	domainRepo := memory.NewMemoryDomainRepository()
	planRepo := memory.NewUnlimitedPlanRepository(memory.NewMemoryUserPlanRepository())
	a, _ := appRepo.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "user-1", Name: "web", ImageURL: "nginx:1"})
	d, _ := appRepo.CreateDeployment(context.Background(), a.ID, "nginx:1")
	_ = appRepo.UpdateDeploymentStatus(context.Background(), d.ID, entities.DeployStatusActive, "", "")

	h := handlers.NewDomainHandler(domainRepo, appRepo, planRepo)
	rec := &recordingRedeployer{}
	h.SetDeployer(rec)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), h.Add)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+a.ID+"/domains", bytes.NewBufferString(`{"domain":"myapp.example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if rec.image != "nginx:1" {
		t.Errorf("redeployed image = %q, want nginx:1", rec.image)
	}
}
```
(Add imports as needed: `bytes`, `context`, `net/http/httptest`, `dto`, `entities`, `memory`, `handlers`, `fiber`. Reuse the file's existing `injectUserID`.)

- [ ] **Step 2: Run them**

Run: `cd services/api && go test ./internal/services/deploy/ -run TestProxyLabels; go test ./internal/handlers/ -run TestDomainAdd_RedeploysWithCurrentImage`
Expected:
- the label tests fail to compile (wrong `proxyLabels` arity);
- the domain test FAILs with `redeployed image = ""` (old code reads the empty `ImageTag`).

- [ ] **Step 3: Implement the deployer changes**

In `DockerDeployer`, add the fields `domainRepo ports.DomainRepository` and `certResolver string`, and these methods:
```go
// SetDomainRepo lets the deployer route an app's active custom domains.
func (d *DockerDeployer) SetDomainRepo(repo ports.DomainRepository) { d.domainRepo = repo }

// SetCertResolver names the Traefik ACME resolver app routers use ("le").
// Empty means no resolver label (Traefik serves its default certificate).
func (d *DockerDeployer) SetCertResolver(name string) { d.certResolver = name }

// appHosts is the app's generated host plus its active custom domains.
func (d *DockerDeployer) appHosts(ctx context.Context, app *entities.App) []string {
	hosts := []string{fmt.Sprintf("%s.%s", app.Subdomain, d.baseDomain)}
	if d.domainRepo == nil {
		return hosts
	}
	domains, err := d.domainRepo.ListDomainsByApp(ctx, app.ID)
	if err != nil {
		slog.Warn("docker-deploy: list custom domains failed", "app", app.Name, "error", err)
		return hosts
	}
	for _, cd := range domains {
		if cd.Status == entities.DomainStatusActive || cd.Status == entities.DomainStatusVerified {
			hosts = append(hosts, cd.Domain)
		}
	}
	return hosts
}
```
In `NewDockerDeployer`, set `certResolver: "le"` in the struct literal.
In `DeployApp`, replace `labels := proxyLabels(app, d.baseDomain, d.network)` with `labels := proxyLabels(app, d.appHosts(ctx, app), d.network, d.certResolver)`.
Replace `proxyLabels`:
```go
// proxyLabels builds the Traefik router/service labels that expose an app
// container on the given hosts over HTTPS. Routers must name the resolver:
// in Traefik v3 a router with its own tls section ignores the entrypoint's
// default resolver, which left routes on the default (self-signed) cert.
func proxyLabels(app *entities.App, hosts []string, network, certResolver string) map[string]string {
	rn := sanitizeName(app.Subdomain)
	labels := map[string]string{
		"zenith.app.id":          app.ID,
		"traefik.enable":         "true",
		"traefik.docker.network": network,
		fmt.Sprintf("traefik.http.routers.%s.rule", rn):                       hostRule(hosts),
		fmt.Sprintf("traefik.http.routers.%s.entrypoints", rn):                "websecure",
		fmt.Sprintf("traefik.http.routers.%s.tls", rn):                        "true",
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", rn): fmt.Sprintf("%d", app.Port),
	}
	if certResolver != "" {
		labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", rn)] = certResolver
	}
	return labels
}
```

- [ ] **Step 4: Domain handler uses the current image**

In `handlers/domain.go`, replace the body of `refreshIngress` after the nil check with:
```go
	imageTag, err := deploy.CurrentImage(ctx, h.appRepo, app.ID)
	if err != nil {
		slog.Warn("no deployed image, skipping route refresh", "app_id", app.ID)
		return
	}
	if err := h.deployer.DeployApp(ctx, app, imageTag); err != nil {
		slog.Warn("failed to refresh app routes", "app_id", app.ID, "error", err)
	}
```
(Import `deploy`. If Task 5 moved `CurrentImage` into `services`, use that package.) Check that `Delete` calls `refreshIngress` after removing the domain. If it does not, add `h.refreshIngress(c.Context(), app)` before its success response, loading `app` with `h.appRepo.GetApp`.

`config.go`: add the field `CertResolver string` to `Config`, and to `Load()` add `CertResolver: getEnv("ZENITH_CERT_RESOLVER", "le"),`.

`main.go`: in the `if dockerDeployer != nil {` block next to `SetEnvVarRepo`, add:
```go
		dockerDeployer.SetDomainRepo(domainRepo)
		dockerDeployer.SetCertResolver(cfg.CertResolver)
```
and replace `domainHandler.SetDeployer(deployer)` with `domainHandler.SetDeployer(backend)`.

- [ ] **Step 5: Run tests**

Run: `cd services/api && go build ./... && go test ./internal/services/deploy/ ./internal/handlers/ ./internal/config/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/api
git commit -m "fix(api): app routers name the le resolver and include active custom domains; domain changes redeploy the current image (F-12, F-18)"
```

---

### Task 7: Every user has a project (F-13)

**Files:**
- Modify: `services/api/internal/handlers/apps_v2.go:345-360`
- Test: `services/api/internal/handlers/apps_v2_test.go` (add test)

**Interfaces:**
- Consumes: `ProjectRepository.GetDefaultProject`, `ProjectRepository.CreateProject(ctx, userID, name, slug, description)`.

- [ ] **Step 1: Write the failing test**

Inspect the top of `apps_v2_test.go` for its setup helper. Then add a test that does the following:
1. Builds `handlers.NewAppHandlerV2(appRepo, "example.com", nil, nil)`, calls `SetProjectRepo(memory.NewMemoryProjectRepository())` on it, and registers `POST /api/v1/apps` with `injectUserID("user-1")`.
2. POSTs `{"name":"whoami","deploy_source":"image","image_url":"traefik/whoami:latest","port":80}`, with no `project_id`, for a user who has no projects.
3. Asserts status `201`, and that `projectRepo.ListProjectsByUser(ctx, "user-1")` returns exactly 1 project, named `Default`.

```go
func TestAppCreate_CreatesDefaultProjectWhenNone(t *testing.T) {
	appRepo := memory.NewMemoryAppRepository()
	projectRepo := memory.NewMemoryProjectRepository()
	h := handlers.NewAppHandlerV2(appRepo, "example.com", nil, nil)
	h.SetProjectRepo(projectRepo)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Post("/api/v1/apps", injectUserID("user-1"), h.Create)

	req := httptest.NewRequest("POST", "/api/v1/apps", bytes.NewBufferString(`{"name":"whoami","deploy_source":"image","image_url":"traefik/whoami:latest","port":80}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d: %s", resp.StatusCode, b)
	}
	ps, _ := projectRepo.ListProjectsByUser(context.Background(), "user-1")
	if len(ps) != 1 || ps[0].Name != "Default" {
		t.Fatalf("projects = %+v, want one named Default", ps)
	}
}
```
(If the handler's create method is not named `Create`, use the name `main.go` registers for `POST /apps`.)

- [ ] **Step 2: Run it**

Run: `cd services/api && go test ./internal/handlers/ -run TestAppCreate_CreatesDefaultProjectWhenNone`
Expected: FAIL. Either it is not 201, or there are 0 projects.

- [ ] **Step 3: Implement**

In `apps_v2.go`, replace:
```go
	} else if projectID == "" && h.projectRepo != nil {
		if dp, err := h.projectRepo.GetDefaultProject(c.Context(), userID.(string)); err == nil {
			projectID = dp.ID
		}
	}
```
with:
```go
	} else if projectID == "" && h.projectRepo != nil {
		if dp, err := h.projectRepo.GetDefaultProject(c.Context(), userID.(string)); err == nil && dp != nil {
			projectID = dp.ID
		} else {
			// First app of a fresh install: create the user's default project
			// instead of failing on the project foreign key.
			np, cErr := h.projectRepo.CreateProject(c.Context(), userID.(string), "Default", "default", "")
			if cErr != nil {
				return NewBadRequest("no project: create a project first")
			}
			projectID = np.ID
		}
	}
```

- [ ] **Step 4: Run tests**

Run: `cd services/api && go test ./internal/handlers/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/api/internal/handlers/apps_v2.go services/api/internal/handlers/apps_v2_test.go
git commit -m "fix(api): first app creates the user's default project instead of a 500 (F-13)"
```

---

### Task 8: Database credentials storage (F-16, part 1)

**Files:**
- Create: `services/api/internal/adapters/postgres/migrations/050_db_password.up.sql`, `050_db_password.down.sql`
- Modify: `services/api/internal/ports/repositories.go` (`DatabaseRepository`)
- Modify: `services/api/internal/adapters/postgres/postgres_database.go`, `services/api/internal/adapters/memory/memory_database.go`
- Test: `services/api/internal/adapters/memory/memory_database_test.go` (add test)

**Interfaces:**
- Produces, on `DatabaseRepository`:
  - `SetDatabaseProvisioned(ctx context.Context, id, host string, port int, secret string) error`, which sets host, port and secret, and status `ready`;
  - `GetDatabaseSecret(ctx context.Context, id string) (string, error)`, which returns the stored secret (encrypted or plain, as given);
  - `SetDatabasePort(ctx context.Context, id string, port int) error`.

- [ ] **Step 1: Write the failing test**

```go
func TestMemoryDatabase_SetProvisioned(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMemoryDatabaseRepository()
	db, err := repo.CreateDatabase(ctx, "", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEngineRedis, Name: "cache"})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetDatabaseProvisioned(ctx, db.ID, "zenith-db-abc", 6379, "enc:xyz"); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetDatabase(ctx, db.ID)
	if got.Host != "zenith-db-abc" || got.Port != 6379 || got.Status != entities.DatabaseStatusReady {
		t.Errorf("got %+v", got)
	}
	s, err := repo.GetDatabaseSecret(ctx, db.ID)
	if err != nil || s != "enc:xyz" {
		t.Errorf("secret = %q, %v", s, err)
	}
}
```

- [ ] **Step 2: Run it**

Run: `cd services/api && go test ./internal/adapters/memory/ -run TestMemoryDatabase_SetProvisioned`
Expected: compile error (methods undefined).

- [ ] **Step 3: Implement**

`050_db_password.up.sql`:
```sql
ALTER TABLE user_databases ADD COLUMN IF NOT EXISTS password_enc TEXT NOT NULL DEFAULT '';
```
`050_db_password.down.sql`:
```sql
ALTER TABLE user_databases DROP COLUMN IF EXISTS password_enc;
```

`ports.DatabaseRepository`: add the three methods from Interfaces.

`postgres_database.go`:
```go
// SetDatabaseProvisioned records where the database runs and its secret, and marks it ready.
func (r *PostgresDatabaseRepository) SetDatabaseProvisioned(ctx context.Context, id, host string, port int, secret string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE user_databases SET host = $1, port = $2, password_enc = $3, status = $4, updated_at = $5 WHERE id = $6`,
		host, port, secret, string(entities.DatabaseStatusReady), time.Now(), id)
	if err != nil {
		return fmt.Errorf("set database provisioned: %w", err)
	}
	return nil
}

func (r *PostgresDatabaseRepository) GetDatabaseSecret(ctx context.Context, id string) (string, error) {
	var s string
	if err := r.pool.QueryRow(ctx, `SELECT password_enc FROM user_databases WHERE id = $1`, id).Scan(&s); err != nil {
		return "", fmt.Errorf("get database secret: %w", err)
	}
	return s, nil
}

func (r *PostgresDatabaseRepository) SetDatabasePort(ctx context.Context, id string, port int) error {
	_, err := r.pool.Exec(ctx, `UPDATE user_databases SET port = $1, updated_at = $2 WHERE id = $3`, port, time.Now(), id)
	if err != nil {
		return fmt.Errorf("set database port: %w", err)
	}
	return nil
}
```

`memory_database.go`:
```go
func (r *MemoryDatabaseRepository) SetDatabaseProvisioned(_ context.Context, id, host string, port int, secret string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	db, ok := r.databases[id]
	if !ok {
		return fmt.Errorf("database not found")
	}
	db.Host, db.Port, db.Status = host, port, entities.DatabaseStatusReady
	r.passwords[id] = secret
	return nil
}

func (r *MemoryDatabaseRepository) GetDatabaseSecret(_ context.Context, id string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.passwords[id]
	if !ok {
		return "", fmt.Errorf("database not found")
	}
	return s, nil
}

func (r *MemoryDatabaseRepository) SetDatabasePort(_ context.Context, id string, port int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	db, ok := r.databases[id]
	if !ok {
		return fmt.Errorf("database not found")
	}
	db.Port = port
	return nil
}
```

- [ ] **Step 4: Run tests**

Run: `cd services/api && go build ./... && go test ./internal/adapters/... ./internal/handlers/`
Expected: PASS. If another type implements `DatabaseRepository` (check `go build` errors), add the three methods to it, returning `nil` / `""`.

- [ ] **Step 5: Commit**

```bash
git add services/api
git commit -m "feat(api): store database host/port/secret after provisioning (F-16 part 1)"
```

---

### Task 9: Docker database backend and provisioning service (F-16, part 2)

**Files:**
- Create: `services/api/internal/services/deploy/docker_db.go`
- Create: `services/api/internal/services/deploy/docker_db_test.go` (`//go:build docker`)
- Create: `services/api/internal/services/db_runtime.go`
- Create: `services/api/internal/services/db_runtime_test.go`

**Interfaces:**
- Consumes: the `DatabaseRepository` methods from Task 8.
- Produces:
  ```go
  // package services
  type DBBackend interface {
      Provision(ctx context.Context, db *entities.UserDatabase, password string) (host string, port int, err error)
      WaitReady(ctx context.Context, db *entities.UserDatabase) error
      Delete(ctx context.Context, db *entities.UserDatabase, purge bool) error
  }
  type DBRuntimeService struct{ /* … */ }
  func NewDBRuntimeService(repo ports.DatabaseRepository, backend DBBackend, crypto *pkgCrypto.EnvCrypto) *DBRuntimeService
  func (s *DBRuntimeService) Create(ctx context.Context, appID, userID string, in *dto.CreateDatabaseInput) (*entities.UserDatabase, error) // returns status provisioning; finishes in background
  func (s *DBRuntimeService) Password(ctx context.Context, db *entities.UserDatabase) (string, error)
  func (s *DBRuntimeService) Delete(ctx context.Context, db *entities.UserDatabase, purge bool) error
  func (s *DBRuntimeService) Wait(id string) // test hook: blocks until the background provision of id ends
  ```
  ```go
  // package deploy
  func NewDockerDBBackend(cli *client.Client, network string) *DockerDBBackend // implements services.DBBackend
  ```

- [ ] **Step 1: Write the failing service tests (fake backend)**

```go
// services/api/internal/services/db_runtime_test.go
package services

import (
	"context"
	"errors"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type fakeDBBackend struct {
	waitErr error
	gotPW   string
}

func (f *fakeDBBackend) Provision(_ context.Context, db *entities.UserDatabase, pw string) (string, int, error) {
	f.gotPW = pw
	return "zenith-db-" + db.ID[:8], 6379, nil
}
func (f *fakeDBBackend) WaitReady(context.Context, *entities.UserDatabase) error { return f.waitErr }
func (f *fakeDBBackend) Delete(context.Context, *entities.UserDatabase, bool) error  { return nil }

func TestDBService_ProvisionsAndStoresCredentials(t *testing.T) {
	repo := memory.NewMemoryDatabaseRepository()
	be := &fakeDBBackend{}
	s := NewDBRuntimeService(repo, be, nil)
	db, err := s.Create(context.Background(), "", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEngineRedis, Name: "cache"})
	if err != nil {
		t.Fatal(err)
	}
	s.Wait(db.ID)
	got, _ := repo.GetDatabase(context.Background(), db.ID)
	if got.Status != entities.DatabaseStatusReady || got.Host == "" || got.Port != 6379 {
		t.Fatalf("got %+v", got)
	}
	pw, err := s.Password(context.Background(), got)
	if err != nil || pw == "" || pw != be.gotPW || len(pw) < 24 {
		t.Errorf("password %q (%v), backend saw %q", pw, err, be.gotPW)
	}
}

func TestDBService_ProvisionFailureMarksError(t *testing.T) {
	repo := memory.NewMemoryDatabaseRepository()
	s := NewDBRuntimeService(repo, &fakeDBBackend{waitErr: errors.New("unhealthy")}, nil)
	db, _ := s.Create(context.Background(), "", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "pg"})
	s.Wait(db.ID)
	got, _ := repo.GetDatabase(context.Background(), db.ID)
	if got.Status != entities.DatabaseStatusError {
		t.Errorf("status = %q, want error", got.Status)
	}
}

func TestDBService_RejectsUnsupportedEngine(t *testing.T) {
	s := NewDBRuntimeService(memory.NewMemoryDatabaseRepository(), &fakeDBBackend{}, nil)
	_, err := s.Create(context.Background(), "", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEngineKafka})
	if err == nil {
		t.Error("kafka must be rejected in the Docker runtime")
	}
}
```

- [ ] **Step 2: Run it**

Run: `cd services/api && go test ./internal/services/ -run TestDBService`
Expected: compile errors (`NewDBRuntimeService` undefined).

- [ ] **Step 3: Implement `db_runtime.go`**

```go
// services/api/internal/services/db_runtime.go
package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pkgCrypto "github.com/dotechhq/zenith/services/api/internal/crypto"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// DBBackend runs database instances (Docker containers in standalone mode).
type DBBackend interface {
	Provision(ctx context.Context, db *entities.UserDatabase, password string) (host string, port int, err error)
	WaitReady(ctx context.Context, db *entities.UserDatabase) error
	Delete(ctx context.Context, db *entities.UserDatabase, purge bool) error
}

// DBRuntimeService creates database records and provisions them on a DBBackend.
type DBRuntimeService struct {
	repo    ports.DatabaseRepository
	backend DBBackend
	crypto  *pkgCrypto.EnvCrypto
	wg      sync.Map // id -> *sync.WaitGroup
}

func NewDBRuntimeService(repo ports.DatabaseRepository, backend DBBackend, crypto *pkgCrypto.EnvCrypto) *DBRuntimeService {
	return &DBRuntimeService{repo: repo, backend: backend, crypto: crypto}
}

// SupportedEngine reports whether the Docker runtime can run engine.
func SupportedEngine(e entities.DatabaseEngine) bool {
	switch e {
	case entities.DatabaseEnginePostgres, entities.DatabaseEngineMySQL, entities.DatabaseEngineRedis:
		return true
	}
	return false
}

func newDBPassword() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil // 36 hex chars, URL-safe
}

// Create inserts the record and provisions it in the background. The returned
// database has status provisioning; it turns ready or error when done.
func (s *DBRuntimeService) Create(ctx context.Context, appID, userID string, in *dto.CreateDatabaseInput) (*entities.UserDatabase, error) {
	if in.Engine == "" {
		in.Engine = entities.DatabaseEnginePostgres
	}
	if !SupportedEngine(in.Engine) {
		return nil, fmt.Errorf("engine %s is not available on this install", in.Engine)
	}
	db, err := s.repo.CreateDatabase(ctx, appID, userID, in)
	if err != nil {
		return nil, err
	}
	pw, err := newDBPassword()
	if err != nil {
		return nil, fmt.Errorf("generate password: %w", err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	s.wg.Store(db.ID, wg)
	go func() {
		defer wg.Done()
		bg, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s.provision(bg, db, pw); err != nil {
			slog.Error("database provisioning failed", "db", db.Name, "engine", db.Engine, "error", err)
			_ = s.repo.UpdateDatabaseStatus(bg, db.ID, entities.DatabaseStatusError)
		}
	}()
	return db, nil
}

func (s *DBRuntimeService) provision(ctx context.Context, db *entities.UserDatabase, pw string) error {
	host, port, err := s.backend.Provision(ctx, db, pw)
	if err != nil {
		return err
	}
	if err := s.backend.WaitReady(ctx, db); err != nil {
		return err
	}
	secret := pw
	if s.crypto != nil {
		if secret, err = s.crypto.Encrypt(db.UserID, pw); err != nil {
			return fmt.Errorf("encrypt password: %w", err)
		}
	}
	return s.repo.SetDatabaseProvisioned(ctx, db.ID, host, port, secret)
}

// Wait blocks until the background provisioning of id has finished (tests).
func (s *DBRuntimeService) Wait(id string) {
	if v, ok := s.wg.Load(id); ok {
		v.(*sync.WaitGroup).Wait()
	}
}

// Password returns the plaintext password of a provisioned database.
func (s *DBRuntimeService) Password(ctx context.Context, db *entities.UserDatabase) (string, error) {
	secret, err := s.repo.GetDatabaseSecret(ctx, db.ID)
	if err != nil || secret == "" {
		return "", fmt.Errorf("no credentials yet")
	}
	if s.crypto != nil && pkgCrypto.IsEncrypted(secret) {
		return s.crypto.Decrypt(db.UserID, secret)
	}
	return secret, nil
}

// Delete removes the instance and its record. purge also drops the data volume.
func (s *DBRuntimeService) Delete(ctx context.Context, db *entities.UserDatabase, purge bool) error {
	if err := s.backend.Delete(ctx, db, purge); err != nil {
		return err
	}
	return s.repo.DeleteDatabase(ctx, db.ID)
}
```

- [ ] **Step 4: Run the service tests**

Run: `cd services/api && go test ./internal/services/ -run TestDBService -race`
Expected: PASS.

- [ ] **Step 5: Write the Docker backend test (tag `docker`)**

```go
//go:build docker

// services/api/internal/services/deploy/docker_db_test.go
package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestDockerDBBackend_RedisLifecycle(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("no docker")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	const net = "zenith-dbtest"
	_, _ = cli.NetworkCreate(ctx, net, network.CreateOptions{})
	defer cli.NetworkRemove(context.Background(), net)

	b := NewDockerDBBackend(cli, net)
	db := &entities.UserDatabase{ID: "deadbeef-0000-0000-0000-000000000000", Engine: entities.DatabaseEngineRedis, DBName: "db_x", DBUser: "u_x"}
	host, port, err := b.Provision(ctx, db, "pw123456789012345678")
	if err != nil {
		t.Fatal(err)
	}
	defer b.Delete(context.Background(), db, true)
	if host != "zenith-db-deadbeef" || port != 6379 {
		t.Errorf("host/port = %s:%d", host, port)
	}
	if err := b.WaitReady(ctx, db); err != nil {
		t.Fatalf("not ready: %v", err)
	}
	insp, err := cli.ContainerInspect(ctx, host)
	if err != nil {
		t.Fatal(err)
	}
	if len(insp.HostConfig.PortBindings) != 0 {
		t.Errorf("database must not publish host ports: %v", insp.HostConfig.PortBindings)
	}
	if insp.HostConfig.RestartPolicy.Name != "unless-stopped" {
		t.Errorf("restart = %q", insp.HostConfig.RestartPolicy.Name)
	}
}
```

- [ ] **Step 6: Run it**

Run: `cd services/api && go test -tags docker ./internal/services/deploy/ -run TestDockerDBBackend`
Expected: compile error (`NewDockerDBBackend` undefined).

- [ ] **Step 7: Implement `docker_db.go`**

```go
// services/api/internal/services/deploy/docker_db.go
package deploy

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// DockerDBBackend runs each managed database as its own container on the
// app network. Data lives in a named volume; no host port is published.
type DockerDBBackend struct {
	cli     *client.Client
	network string
}

func NewDockerDBBackend(cli *client.Client, network string) *DockerDBBackend {
	return &DockerDBBackend{cli: cli, network: network}
}

func dbShortID(db *entities.UserDatabase) string {
	id := db.ID
	if len(id) > 8 {
		id = id[:8]
	}
	return id
}

func dbContainerName(db *entities.UserDatabase) string { return "zenith-db-" + dbShortID(db) }

type dbSpec struct {
	image string
	port  int
	env   []string
	cmd   []string
	data  string
	test  []string
}

func specFor(db *entities.UserDatabase, pw string) (dbSpec, error) {
	switch db.Engine {
	case entities.DatabaseEnginePostgres:
		return dbSpec{image: "postgres:16-alpine", port: 5432, data: "/var/lib/postgresql/data",
			env:  []string{"POSTGRES_DB=" + db.DBName, "POSTGRES_USER=" + db.DBUser, "POSTGRES_PASSWORD=" + pw},
			test: []string{"CMD-SHELL", "pg_isready -U " + db.DBUser + " -d " + db.DBName + " -h 127.0.0.1"}}, nil
	case entities.DatabaseEngineMySQL:
		return dbSpec{image: "mysql:8.4", port: 3306, data: "/var/lib/mysql",
			env:  []string{"MYSQL_DATABASE=" + db.DBName, "MYSQL_USER=" + db.DBUser, "MYSQL_PASSWORD=" + pw, "MYSQL_ROOT_PASSWORD=" + pw},
			test: []string{"CMD-SHELL", "mysqladmin ping -h 127.0.0.1 -u" + db.DBUser + " -p$MYSQL_PASSWORD --silent"}}, nil
	case entities.DatabaseEngineRedis:
		return dbSpec{image: "redis:7-alpine", port: 6379, data: "/data",
			cmd:  []string{"redis-server", "--requirepass", pw, "--appendonly", "yes"},
			test: []string{"CMD-SHELL", "redis-cli -a \"$REDIS_PASSWORD\" --no-auth-warning ping | grep -q PONG"},
			env:  []string{"REDIS_PASSWORD=" + pw}}, nil
	}
	return dbSpec{}, fmt.Errorf("engine %s not supported by the Docker runtime", db.Engine)
}

// Provision pulls the engine image and starts the container. Idempotent: an
// existing container with the same name is replaced, the volume is kept.
func (b *DockerDBBackend) Provision(ctx context.Context, db *entities.UserDatabase, pw string) (string, int, error) {
	spec, err := specFor(db, pw)
	if err != nil {
		return "", 0, err
	}
	rc, err := b.cli.ImagePull(ctx, spec.image, image.PullOptions{})
	if err != nil {
		return "", 0, fmt.Errorf("pull %s: %w", spec.image, err)
	}
	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()

	name := dbContainerName(db)
	_ = b.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})
	resp, err := b.cli.ContainerCreate(ctx,
		&container.Config{
			Image:  spec.image,
			Env:    spec.env,
			Cmd:    spec.cmd,
			Labels: map[string]string{"zenith.db.id": db.ID, "zenith.db.engine": string(db.Engine)},
			Healthcheck: &container.HealthConfig{
				Test: spec.test, Interval: 3 * time.Second, Timeout: 3 * time.Second, Retries: 40, StartPeriod: 5 * time.Second,
			},
		},
		&container.HostConfig{
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			Mounts:        []mount.Mount{{Type: mount.TypeVolume, Source: name, Target: spec.data}},
		},
		&network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{b.network: {Aliases: []string{name}}}},
		nil, name)
	if err != nil {
		return "", 0, fmt.Errorf("create %s: %w", name, err)
	}
	if err := b.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", 0, fmt.Errorf("start %s: %w", name, err)
	}
	return name, spec.port, nil
}

// WaitReady polls the container health until healthy, unhealthy, or ctx ends.
func (b *DockerDBBackend) WaitReady(ctx context.Context, db *entities.UserDatabase) error {
	name := dbContainerName(db)
	for {
		insp, err := b.cli.ContainerInspect(ctx, name)
		if err != nil {
			return fmt.Errorf("inspect %s: %w", name, err)
		}
		if insp.State != nil && insp.State.Health != nil {
			switch insp.State.Health.Status {
			case "healthy":
				return nil
			case "unhealthy":
				return fmt.Errorf("%s is unhealthy", name)
			}
		}
		if insp.State != nil && !insp.State.Running && insp.State.ExitCode != 0 {
			return fmt.Errorf("%s exited with code %d", name, insp.State.ExitCode)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s not ready: %w", name, ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}
}

// Delete removes the container; purge also removes its data volume.
func (b *DockerDBBackend) Delete(ctx context.Context, db *entities.UserDatabase, purge bool) error {
	name := dbContainerName(db)
	if err := b.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true}); err != nil && !client.IsErrNotFound(err) {
		return fmt.Errorf("remove %s: %w", name, err)
	}
	if purge {
		if err := b.cli.VolumeRemove(ctx, name, true); err != nil && !client.IsErrNotFound(err) {
			return fmt.Errorf("remove volume %s: %w", name, err)
		}
	}
	return nil
}
```
At the bottom of `db_runtime.go` nothing references `deploy` (that would create an import cycle). The compile-time assertion `var _ services.DBBackend = (*DockerDBBackend)(nil)` goes in `docker_db.go` only if `deploy` already imports `services` (check with `grep -n '"github.com/dotechhq/zenith/services/api/internal/services"' internal/services/deploy/*.go`). If it doesn't, skip the assertion; `main.go` wiring enforces it at compile time.

- [ ] **Step 8: Run the Docker test and the full unit suite**

Run: `cd services/api && go test -tags docker ./internal/services/deploy/ -run TestDockerDBBackend -v 2>&1 | tail -5 && go test ./internal/services/...`
Expected: PASS. A `zenith-db-deadbeef` container appears and is removed; `docker ps -a | grep zenith-db-deadbeef` afterwards prints nothing.

- [ ] **Step 9: Commit**

```bash
git add services/api/internal/services
git commit -m "feat(api): Docker database runtime — postgres/mysql/redis containers with health-gated provisioning (F-16 part 2)"
```

---

### Task 10: Database handlers use the runtime; CLI and UI follow (F-16, part 3)

**Files:**
- Modify: `services/api/internal/handlers/databases_v2.go` (`Create`, `CreateStandalone`, `DeleteStandalone`, `fetchCredentials`, new setter)
- Modify: `services/api/cmd/server/main.go` (wiring near line 888)
- Modify: `cli/internal/api/client.go:232-258`
- Modify: the web database create dialog (find it with `grep -rn "Coming Soon" apps/web/src/app/databases apps/web/src/components | grep -i mysql`)
- Test: `services/api/internal/handlers/databases_v2_test.go` (create or extend), `cli/internal/api/client_db_test.go` (new)

**Interfaces:**
- Consumes: `services.DBRuntimeService` (Task 9).
- Produces: `(*DatabaseHandlerV2).SetRuntime(rt *services.DBRuntimeService)`.

- [ ] **Step 1: Write the failing handler test**

```go
// services/api/internal/handlers/databases_v2_runtime_test.go
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

type okDBBackend struct{}

func (okDBBackend) Provision(_ context.Context, db *entities.UserDatabase, _ string) (string, int, error) {
	return "zenith-db-" + db.ID[:8], 5432, nil
}
func (okDBBackend) WaitReady(context.Context, *entities.UserDatabase) error    { return nil }
func (okDBBackend) Delete(context.Context, *entities.UserDatabase, bool) error { return nil }

func TestDatabaseCreateForApp_InjectsDatabaseURL(t *testing.T) {
	ctx := context.Background()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	a, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{UserID: "user-1", Name: "web", ImageURL: "nginx:1"})
	rt := services.NewDBRuntimeService(dbRepo, okDBBackend{}, nil)
	h := handlers.NewDatabaseHandlerV2(nil, dbRepo, appRepo, nil)
	h.SetRuntime(rt)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Post("/api/v1/apps/:appId/databases", injectUserID("user-1"), h.Create)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+a.ID+"/databases", bytes.NewBufferString(`{"engine":"postgresql","name":"main"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 201 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var info dto.DatabaseInfo
	_ = json.NewDecoder(resp.Body).Decode(&info)
	rt.Wait(info.ID)

	vars, _ := appRepo.GetEnvVars(ctx, a.ID)
	var url string
	for _, v := range vars {
		if v.Key == "DATABASE_URL" {
			url = v.Value
		}
	}
	if url == "" || !bytes.Contains([]byte(url), []byte("@zenith-db-")) || !bytes.Contains([]byte(url), []byte(":5432/")) {
		t.Errorf("DATABASE_URL = %q", url)
	}
}

func TestDatabaseCreateStandalone_UsesRuntime(t *testing.T) {
	dbRepo := memory.NewMemoryDatabaseRepository()
	rt := services.NewDBRuntimeService(dbRepo, okDBBackend{}, nil)
	h := handlers.NewDatabaseHandlerV2(nil, dbRepo, memory.NewMemoryAppRepository(), nil)
	h.SetRuntime(rt)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Post("/api/v1/databases", injectUserID("user-1"), h.CreateStandalone)

	req := httptest.NewRequest("POST", "/api/v1/databases", bytes.NewBufferString(`{"engine":"mysql","name":"shop"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 201 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var info dto.DatabaseInfo
	_ = json.NewDecoder(resp.Body).Decode(&info)
	rt.Wait(info.ID)
	got, _ := dbRepo.GetDatabase(context.Background(), info.ID)
	if got.Status != entities.DatabaseStatusReady || got.Host == "" {
		t.Errorf("got %+v", got)
	}
}
```

- [ ] **Step 2: Run it**

Run: `cd services/api && go test ./internal/handlers/ -run 'TestDatabaseCreate(ForApp|Standalone)'`
Expected: compile error (`SetRuntime` undefined).

- [ ] **Step 3: Implement in `databases_v2.go`**

Add the field `rt *services.DBRuntimeService` to `DatabaseHandlerV2`, and:
```go
// SetRuntime makes the handler provision real instances (Docker in standalone).
func (h *DatabaseHandlerV2) SetRuntime(rt *services.DBRuntimeService) { h.rt = rt }

// injectWhenReady writes the connection string into the app's env once the
// database is provisioned. Runs in the background.
func (h *DatabaseHandlerV2) injectWhenReady(appID string, db *entities.UserDatabase) {
	go func() {
		h.rt.Wait(db.ID)
		ctx := context.Background()
		fresh, err := h.dbRepo.GetDatabase(ctx, db.ID)
		if err != nil || fresh.Status != entities.DatabaseStatusReady {
			return
		}
		pw, err := h.rt.Password(ctx, fresh)
		if err != nil {
			return
		}
		_ = h.appRepo.SetEnvVars(ctx, appID, map[string]string{envKeyForEngine(fresh.Engine): fresh.ConnectionString(pw)})
	}()
}
```
In `fetchCredentials`, add a first branch:
```go
	if h.rt != nil {
		if pw, err := h.rt.Password(ctx, db); err == nil {
			return pw, db.ConnectionString(pw)
		}
		return "", ""
	}
```
In `Create` (per-app), directly before `// Use DatabaseService for real provisioning if available`, insert:
```go
	if h.rt != nil {
		db, err := h.rt.Create(c.Context(), appID, userID, &input)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
				return fiber.NewError(fiber.StatusConflict, err.Error())
			}
			if strings.Contains(err.Error(), "not available") {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		h.injectWhenReady(appID, db)
		return c.Status(fiber.StatusCreated).JSON(toDatabaseInfoV2(db, "", ""))
	}
```
In `CreateStandalone`, insert the same block before its `if h.dbSvc != nil {`, with `""` as the app ID and without the `injectWhenReady` call.
In `DeleteStandalone`, before `if h.dbSvc != nil {`, insert:
```go
	if h.rt != nil {
		if err := h.rt.Delete(c.Context(), db, c.QueryBool("purge", false)); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"message": "database deleted"})
	}
```
Note on the injected env:
- `envKeyForEngine` already maps postgres/mysql to `DATABASE_URL` and redis to `REDIS_URL`. Verify with `grep -n 'func envKeyForEngine' -A12 internal/handlers/*.go`.
- The injected value goes to the legacy table, which Task 5 merges into the container env.
- After injection the app must be redeployed to pick it up. The UI's "Restart now" (env apply) does that, and the acceptance run in Task 13 uses it.

`main.go`, right after `dbHandlerV2 := handlers.NewDatabaseHandlerV2(dbSvc, dbRepo, appRepo, msRepo)`:
```go
	if cfg.Mode != "saas" && dockerDeployerCli != nil {
		dbHandlerV2.SetRuntime(services.NewDBRuntimeService(dbRepo, deploy.NewDockerDBBackend(dockerDeployerCli, cfg.AppNetwork), envCrypto))
		slog.Info("database runtime: Docker containers")
	}
```
For this, keep the Docker client in a variable visible at that point. In the deployer selection block, declare `var dockerDeployerCli *dockerclient.Client` before the `if cfg.Mode != "saas"` and assign `dockerDeployerCli = dockerCli` in the success branch. If `envCrypto` is declared later than line 888, move the `SetRuntime` call to just after `envCrypto` is set (around line 622 is earlier, so it is in scope; verify).

- [ ] **Step 4: CLI uses the standalone database endpoints**

Write the test first, `cli/internal/api/client_db_test.go`:
```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDatabasesUseStandaloneEndpoints(t *testing.T) {
	var gotPath, gotQuery string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		if r.Method == "POST" {
			_ = json.NewDecoder(r.Body).Decode(&body)
			_, _ = w.Write([]byte(`{"name":"pg","engine":"postgresql","status":"provisioning","port":5432}`))
			return
		}
		_, _ = w.Write([]byte(`[{"name":"pg","engine":"postgresql","status":"ready","port":5432}]`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "tok")

	dbs, err := c.ListDatabases("proj-1")
	if err != nil || len(dbs) != 1 || gotPath != "/api/v1/databases" || gotQuery != "project_id=proj-1" {
		t.Fatalf("list: %v %v path=%s q=%s", dbs, err, gotPath, gotQuery)
	}
	if _, err := c.CreateDatabase("proj-1", &Database{Name: "pg", Engine: "postgresql"}); err != nil || gotPath != "/api/v1/databases" || body["project_id"] != "proj-1" {
		t.Fatalf("create: %v path=%s body=%v", err, gotPath, body)
	}
}
```
Run `cd cli && go test ./internal/api/ -run TestDatabasesUseStandaloneEndpoints`. Expected: FAIL. If the constructor is not `NewClient(baseURL, token)`, use the one `client.go` defines.

Then in `client.go` replace `ListDatabases` and `CreateDatabase`:
```go
func (c *Client) ListDatabases(projectID string) ([]Database, error) {
	var dbs []Database
	if err := c.do("GET", "/api/v1/databases?project_id="+url.QueryEscape(projectID), nil, &dbs); err != nil {
		return nil, err
	}
	return dbs, nil
}

func (c *Client) CreateDatabase(projectID string, db *Database) (*Database, error) {
	req := map[string]any{"name": db.Name, "engine": db.Engine, "project_id": projectID}
	var result Database
	if err := c.do("POST", "/api/v1/databases", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
```
(Add the `net/url` import.) Leave `GetDatabase` alone.

- [ ] **Step 5: UI — MySQL selectable**

In the database create dialog found by the grep above, remove the "Coming Soon" badge and the disabled state on the MySQL option, so it behaves like PostgreSQL and Redis.

- [ ] **Step 6: Run all tests**

Run: `cd services/api && go build ./... && go test ./internal/handlers/ ./internal/services/ && cd ../../cli && go test ./internal/api/ && cd ../apps/web && pnpm vitest run`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/api cli apps/web
git commit -m "feat: databases provision real Docker instances; app attach injects DATABASE_URL/REDIS_URL; CLI uses /api/v1/databases; MySQL enabled in UI (F-16)"
```

---

### Task 11: Logs from Docker; deploy logs survive restarts; k8s integrations off in standalone (F-10, F-19)

**Files:**
- Create: `services/api/internal/services/deploy/docker_logs.go`
- Modify: `services/api/internal/services/monitoring.go` (`AppLogSource`, `SetLogSource`, `GetLogs`, `StreamLogs`)
- Modify: `services/api/internal/services/deploy/pipeline.go` (persist `build_log`)
- Modify: `services/api/internal/handlers/logs.go` (`GetLogs` history fallback)
- Modify: `services/api/internal/config/config.go` (empty k8s URLs in standalone)
- Modify: `services/api/cmd/server/main.go` (wiring; backup service only in saas)
- Test: `services/api/internal/services/monitoring_logsource_test.go` (new), `services/api/internal/handlers/logs_history_test.go` (new), `services/api/internal/config/config_test.go` (add)

**Interfaces:**
- Produces:
  ```go
  // package services
  type AppLogSource interface {
      Tail(ctx context.Context, app *entities.App, since time.Duration, limit int) ([]dto.MonitoringLogEntry, error)
      Stream(ctx context.Context, app *entities.App, out chan<- dto.MonitoringLogEntry) error
  }
  func (s *MonitoringService) SetLogSource(src AppLogSource)
  // package deploy
  func NewDockerLogSource(cli *client.Client) *DockerLogSource // container name: "zenith-app-" + sanitizeName(app.Subdomain)
  ```

- [ ] **Step 1: Write the failing tests**

```go
// services/api/internal/services/monitoring_logsource_test.go
package services

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type fakeLogs struct{}

func (fakeLogs) Tail(_ context.Context, _ *entities.App, _ time.Duration, _ int) ([]dto.MonitoringLogEntry, error) {
	return []dto.MonitoringLogEntry{{Line: "GET / 200"}, {Line: "ERROR boom"}, {Line: "GET /health 200"}}, nil
}
func (fakeLogs) Stream(_ context.Context, _ *entities.App, out chan<- dto.MonitoringLogEntry) error {
	out <- dto.MonitoringLogEntry{Line: "live"}
	return nil
}

func TestMonitoring_UsesLogSourceWithSearch(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	a, _ := repo.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "u1", Name: "web", ImageURL: "nginx:1"})
	s := NewMonitoringService(nil, nil, nil, repo)
	s.SetLogSource(fakeLogs{})

	all, err := s.GetLogs(context.Background(), "u1", a.ID, "", "", 100, time.Hour)
	if err != nil || all.Total != 3 {
		t.Fatalf("all: %+v %v", all, err)
	}
	f, _ := s.GetLogs(context.Background(), "u1", a.ID, "", "boom", 100, time.Hour)
	if f.Total != 1 || f.Entries[0].Line != "ERROR boom" {
		t.Errorf("search: %+v", f)
	}
	ch := make(chan dto.MonitoringLogEntry, 1)
	if err := s.StreamLogs(context.Background(), "u1", a.ID, ch); err != nil {
		t.Fatal(err)
	}
	if (<-ch).Line != "live" {
		t.Error("stream did not use the log source")
	}
}
```

```go
// services/api/internal/handlers/logs_history_test.go
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/services/deploy"
	"github.com/gofiber/fiber/v2"
)

func TestLogHistory_FallsBackToBuildLog(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMemoryAppRepository()
	a, _ := repo.CreateApp(ctx, &dto.CreateAppInput{UserID: "user-1", Name: "web", ImageURL: "nginx:1"})
	d, _ := repo.CreateDeployment(ctx, a.ID, "nginx:1")
	_ = repo.UpdateDeploymentStatus(ctx, d.ID, entities.DeployStatusActive,
		"2026-10-12T08:00:00Z\tinfo\tDeploying web\n2026-10-12T08:00:05Z\tdeploy\t✓ web is live\n", "")

	h := handlers.NewLogHandler(deploy.NewLogHub(10), repo) // empty hub: as after an API restart
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Get("/api/v1/apps/:appId/deployments/:did/logs/history", injectUserID("user-1"), h.GetLogs)

	resp, _ := app.Test(httptest.NewRequest("GET", "/api/v1/apps/"+a.ID+"/deployments/"+d.ID+"/logs/history", nil), -1)
	var out struct {
		Items []struct{ Level, Message string } `json:"items"`
		Total int                               `json:"total"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Total != 2 || out.Items[1].Message != "✓ web is live" || out.Items[1].Level != "deploy" {
		t.Errorf("got %+v", out)
	}
}
```
(If `NewLogHandler` takes its arguments in another order, match `logs.go`.)

Add to `config_test.go`:
```go
func TestLoad_StandaloneHasNoK8sObservabilityURLs(t *testing.T) {
	t.Setenv("ZENITH_MODE", "standalone")
	t.Setenv("JWT_SECRET", "x")
	cfg := config.Load()
	if cfg.LokiURL != "" || cfg.PrometheusURL != "" {
		t.Errorf("loki=%q prom=%q, want empty in standalone", cfg.LokiURL, cfg.PrometheusURL)
	}
}
```
(Match the package name and import style used in the rest of `config_test.go`.)

- [ ] **Step 2: Run them**

Run: `cd services/api && go test ./internal/services/ -run TestMonitoring_UsesLogSource; go test ./internal/handlers/ -run TestLogHistory_FallsBackToBuildLog; go test ./internal/config/ -run TestLoad_StandaloneHasNoK8s`
Expected:
- the monitoring test fails to compile (`SetLogSource` undefined);
- the history test FAILs (`Total` is 0);
- the config test FAILs (it gets the k8s URLs).

- [ ] **Step 3: Implement the log source in `monitoring.go`**

Add the field `logs AppLogSource` to `MonitoringService`, the `AppLogSource` interface from Interfaces, and:
```go
// SetLogSource makes app logs come from the runtime (Docker) instead of Loki.
func (s *MonitoringService) SetLogSource(src AppLogSource) { s.logs = src }

func filterLogs(in []dto.MonitoringLogEntry, level, search string) []dto.MonitoringLogEntry {
	out := make([]dto.MonitoringLogEntry, 0, len(in))
	for _, e := range in {
		if search != "" && !strings.Contains(strings.ToLower(e.Line), strings.ToLower(search)) {
			continue
		}
		if level != "" && level != "all" && !strings.Contains(strings.ToLower(e.Line), strings.ToLower(level)) && e.Level != level {
			continue
		}
		out = append(out, e)
	}
	return out
}
```
At the top of `GetLogs`, after `resolveApp` succeeds, insert:
```go
	if s.logs != nil {
		app, err := s.appRepo.GetApp(ctx, appID)
		if err != nil {
			return nil, fmt.Errorf("app not found")
		}
		if limit <= 0 {
			limit = 100
		}
		if since == 0 {
			since = time.Hour
		}
		entries, err := s.logs.Tail(ctx, app, since, limit)
		if err != nil {
			return nil, err
		}
		entries = filterLogs(entries, level, search)
		return &dto.MonitoringLogsResponse{Entries: entries, Total: len(entries)}, nil
	}
```
At the top of `StreamLogs`, after `resolveApp`, insert:
```go
	if s.logs != nil {
		app, err := s.appRepo.GetApp(ctx, appID)
		if err != nil {
			return fmt.Errorf("app not found")
		}
		return s.logs.Stream(ctx, app, out)
	}
```

- [ ] **Step 4: Implement `docker_logs.go`**

```go
// services/api/internal/services/deploy/docker_logs.go
package deploy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// DockerLogSource reads app logs from the app's container (stdout+stderr).
type DockerLogSource struct{ cli *client.Client }

func NewDockerLogSource(cli *client.Client) *DockerLogSource { return &DockerLogSource{cli: cli} }

func appContainer(app *entities.App) string { return "zenith-app-" + sanitizeName(app.Subdomain) }

func (s *DockerLogSource) open(ctx context.Context, app *entities.App, opts container.LogsOptions) (io.ReadCloser, error) {
	opts.ShowStdout, opts.ShowStderr, opts.Timestamps = true, true, true
	rc, err := s.cli.ContainerLogs(ctx, appContainer(app), opts)
	if err != nil {
		return nil, fmt.Errorf("container logs %s: %w", appContainer(app), err)
	}
	pr, pw := io.Pipe()
	go func() {
		_, err := stdcopy.StdCopy(pw, pw, rc)
		_ = rc.Close()
		_ = pw.CloseWithError(err)
	}()
	return pr, nil
}

func parseDockerLine(line string) dto.MonitoringLogEntry {
	ts, rest, ok := strings.Cut(line, " ")
	if t, err := time.Parse(time.RFC3339Nano, ts); ok && err == nil {
		return dto.MonitoringLogEntry{Timestamp: t, Line: rest}
	}
	return dto.MonitoringLogEntry{Timestamp: time.Now(), Line: line}
}

func (s *DockerLogSource) Tail(ctx context.Context, app *entities.App, since time.Duration, limit int) ([]dto.MonitoringLogEntry, error) {
	r, err := s.open(ctx, app, container.LogsOptions{Since: time.Now().Add(-since).Format(time.RFC3339), Tail: fmt.Sprintf("%d", limit)})
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var out []dto.MonitoringLogEntry
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		out = append(out, parseDockerLine(sc.Text()))
	}
	return out, nil
}

func (s *DockerLogSource) Stream(ctx context.Context, app *entities.App, out chan<- dto.MonitoringLogEntry) error {
	r, err := s.open(ctx, app, container.LogsOptions{Follow: true, Tail: "50"})
	if err != nil {
		return err
	}
	defer r.Close()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		select {
		case out <- parseDockerLine(sc.Text()):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
```
Add a `docker`-tagged test `docker_logs_test.go`:
1. Run `alpine:3.20` with `sh -c 'echo hello-zenith; sleep 30'`, named `zenith-app-logtest`.
2. Call `Tail` with `&entities.App{Subdomain: "logtest"}`.
3. Assert one entry's `Line` is `hello-zenith`.
4. Force-remove the container.

Run it with `go test -tags docker ./internal/services/deploy/ -run TestDockerLogSource`.

- [ ] **Step 5: Persist and read back deploy logs**

`pipeline.go`: add the helper
```go
// buildLog renders the deployment's log history for the deployments.build_log
// column: one "RFC3339\tlevel\tmessage" line per entry.
func (p *Pipeline) buildLog(deploymentID string) string {
	if p.logHub == nil {
		return ""
	}
	var b strings.Builder
	for _, e := range p.logHub.History(deploymentID) {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", e.Timestamp.UTC().Format(time.RFC3339), e.Level, strings.ReplaceAll(e.Message, "\n", " "))
	}
	return b.String()
}
```
Then:
- In the failure branch, change `p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID, entities.DeployStatusFailed, "", err.Error())` to pass `p.buildLog(deployment.ID)` as the build log. Do this after the `emitLog` call so the error line is included.
- In the success path, change `p.appRepo.UpdateDeploymentStatus(ctx, deployment.ID, entities.DeployStatusActive, "", "")` so it runs **after** the `p.emitLog(deployment.ID, "deploy", …is live…)` line, passing `p.buildLog(deployment.ID)`.
- Add the `strings` import if missing. Check the actual field name of the hub in `Pipeline` (`logHub`) with `grep -n 'logHub' pipeline.go`.

`DockerDeployer` should emit steps into the same hub. That needs a hub reference, so pass it through the existing pipeline lines instead: in `TriggerImageDeploy` before `p.deployer.DeployApp`, add `p.emitLog(deployment.ID, "info", "Pulling image and starting container...")`.

`handlers/logs.go` `GetLogs`: after `entries := h.logHub.History(deploymentID)`, if `len(entries) == 0`, load `dep, err := h.appRepo.GetDeployment(c.Context(), deploymentID)`. If `err == nil && dep.AppID == appID && dep.BuildLog != ""`, parse each non-empty line with `strings.SplitN(line, "\t", 3)` into `logEntryJSON{Timestamp: parts[0], Level: parts[1], Message: parts[2]}`, and return those as `items`/`total`.

- [ ] **Step 6: Config and wiring**

`config.go`:
1. In `Load()`, before `return &Config{`, add:
```go
	k8sDefault := func(v string) string {
		if mode == "saas" {
			return v
		}
		return "" // standalone: no Loki/Prometheus/Grafana/Tempo unless set explicitly
	}
```
2. Change the four defaults to `getEnv("PROMETHEUS_URL", k8sDefault("http://kube-prometheus-stack-prometheus.monitoring.svc.cluster.local:9090"))`, and the same for `LOKI_URL`, `GRAFANA_URL` and `TEMPO_URL`.

`main.go`:
- After `monitoringSvc := services.NewMonitoringService(...)`:
```go
	if dockerDeployerCli != nil {
		monitoringSvc.SetLogSource(deploy.NewDockerLogSource(dockerDeployerCli))
		slog.Info("app logs: Docker container logs")
	}
```
- Change the backup condition `if cfg.S3Endpoint != "" && k8sClient != nil {` to `if cfg.Mode == "saas" && cfg.S3Endpoint != "" && k8sClient != nil {`. Backup/restore for Compose is sub-project 2b.

- [ ] **Step 7: Run tests**

Run: `cd services/api && go build ./... && go test ./... 2>&1 | tail -15 && go test -tags docker ./internal/services/deploy/ -run 'TestDockerLogSource|TestDockerDBBackend'`
Expected: all packages ok.

- [ ] **Step 8: Commit**

```bash
git add services/api
git commit -m "feat(api): app logs from Docker; deploy logs persisted to build_log; k8s observability and backup off in standalone (F-10, F-19)"
```

---

### Task 12: Install — bucket bootstrap, encryption key, Traefik always on, sslip.io default (F-04, F-07, F-12)

**Files:**
- Create: `services/api/cmd/server/storage_bootstrap.go`, `services/api/cmd/server/storage_bootstrap_test.go`
- Modify: `services/api/cmd/server/main.go` (call it)
- Modify: `docker-compose.yml`
- Modify: `infra/scripts/install.sh`
- Create: `infra/scripts/install_test.sh`
- Modify: `.env.example`

**Interfaces:**
- Produces: `ensureBucket(ctx context.Context, create func(context.Context, string) error, bucket string, attempts int, wait time.Duration) error`. It treats "already exists" / "already owned" errors as success.

- [ ] **Step 1: Write the failing Go test**

```go
// services/api/cmd/server/storage_bootstrap_test.go
package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEnsureBucket(t *testing.T) {
	calls := 0
	create := func(context.Context, string) error {
		calls++
		if calls < 3 {
			return errors.New("dial tcp storage:9000: connection refused")
		}
		return nil
	}
	if err := ensureBucket(context.Background(), create, "b", 5, time.Millisecond); err != nil || calls != 3 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}

	exists := func(context.Context, string) error {
		return errors.New("create S3 bucket b: BucketAlreadyOwnedByYou: Your previous request to create the named bucket succeeded")
	}
	if err := ensureBucket(context.Background(), exists, "b", 2, time.Millisecond); err != nil {
		t.Fatalf("already-owned must count as success: %v", err)
	}

	never := func(context.Context, string) error { return errors.New("connection refused") }
	if err := ensureBucket(context.Background(), never, "b", 2, time.Millisecond); err == nil {
		t.Fatal("want error after attempts are exhausted")
	}
}
```

- [ ] **Step 2: Run it**

Run: `cd services/api && go test ./cmd/server/ -run TestEnsureBucket`
Expected: compile error (`ensureBucket` undefined).

- [ ] **Step 3: Implement and wire**

```go
// services/api/cmd/server/storage_bootstrap.go
package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ensureBucket creates the platform bucket, retrying while object storage
// starts. An existing bucket is success. Replaces the old one-shot
// storage-init job (minio/mc), whose image is no longer published.
func ensureBucket(ctx context.Context, create func(context.Context, string) error, bucket string, attempts int, wait time.Duration) error {
	var last error
	for i := 0; i < attempts; i++ {
		err := create(ctx, bucket)
		if err == nil {
			return nil
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "alreadyownedbyyou") || strings.Contains(msg, "alreadyexists") || strings.Contains(msg, "already exists") {
			return nil
		}
		last = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return fmt.Errorf("ensure bucket %s: %w", bucket, last)
}
```
In `main.go`, right after `objStorage = s3client.NewClient(...)` inside `if cfg.S3Endpoint != "" {`:
```go
		go func(s3c *s3client.Client) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if err := ensureBucket(ctx, s3c.CreateBucket, cfg.S3PlatformBucket, 60, 2*time.Second); err != nil {
				slog.Error("platform bucket not ready", "error", err)
			} else {
				slog.Info("platform bucket ready", "bucket", cfg.S3PlatformBucket)
			}
		}(objStorage.(*s3client.Client))
```
(If `NewClient` returns `*s3client.Client` and `objStorage` is the interface, capture the concrete value in a local `s3c := s3client.NewClient(...)` first, then assign `objStorage = s3c` and pass `s3c`.)

- [ ] **Step 4: Compose changes**

In `docker-compose.yml`:
1. Delete the whole `storage-init:` service and its comment block.
2. In `api.depends_on`, remove `storage-init:` and its `condition:` line, and add `storage:` (plain, no condition).
3. `api.ports`: `- "127.0.0.1:8080:8080"`. `web.ports`: `- "127.0.0.1:3000:3000"`.
4. `api.environment` add:
   ```yaml
      SECRETS_ENCRYPTION_KEY: ${SECRETS_ENCRYPTION_KEY:-}
      ZENITH_CERT_RESOLVER: le
   ```
   and change `CORS_ORIGINS: ${CORS_ORIGINS:-http://localhost:3000}` to `CORS_ORIGINS: ${CORS_ORIGINS:-https://${ZENITH_DOMAIN:-localhost}}`.
5. `api.labels`: replace `- "traefik.http.routers.zenith-api.tls=true"` and the comment above it with:
   ```yaml
      - "traefik.http.routers.zenith-api.tls=true"
      # Name the resolver: a router with its own tls section ignores the
      # entrypoint default, which left the dashboard on the default cert (F-12).
      - "traefik.http.routers.zenith-api.tls.certresolver=le"
   ```
   Do the same for `zenith-web` (`tls.certresolver=le`).
6. `web.environment`: `NEXT_PUBLIC_API_URL: ${NEXT_PUBLIC_API_URL:-}` (empty means same origin).
7. `traefik`: delete `profiles: ["tls"]`, and update its header comment to say Traefik always runs and owns 80/443.
8. Update the stale comments that mention "`tls` profile" on `api` and `web`.

- [ ] **Step 5: Installer test (RED)**

```bash
#!/usr/bin/env bash
# infra/scripts/install_test.sh — renders install.sh's .env in a temp dir
# without Docker or network (ZENITH_DRY_RUN=1) and checks the result.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
fail() { echo "FAIL: $*" >&2; exit 1; }

run() { # $1 = workdir; remaining = env assignments
  local dir="$1"; shift
  mkdir -p "$dir"
  cp "$repo/.env.example" "$dir/.env.example"
  (cd "$dir" && env "$@" ZENITH_DRY_RUN=1 ZENITH_PUBLIC_IP=203.0.113.7 bash "$here/install.sh" >/dev/null)
}

t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT

run "$t/a"
env_a="$t/a/.env"
grep -qE '^SECRETS_ENCRYPTION_KEY=[0-9a-f]{64}$' "$env_a" || fail "SECRETS_ENCRYPTION_KEY not 64 hex"
grep -qE '^S3_ACCESS_KEY=[A-Za-z0-9]{16,}$' "$env_a" || fail "S3_ACCESS_KEY not generated"
grep -qx 'ZENITH_DOMAIN=203.0.113.7.sslip.io' "$env_a" || fail "default domain not <ip>.sslip.io"

run "$t/b" ZENITH_DOMAIN=zenith.example.com
grep -qx 'ZENITH_DOMAIN=zenith.example.com' "$t/b/.env" || fail "explicit domain not kept"

echo "install_test: ok"
```
`chmod +x infra/scripts/install_test.sh`, then run `bash infra/scripts/install_test.sh`.
Expected: FAIL (the installer has no `ZENITH_DRY_RUN`; with the stub it would try to clone or run Docker).

- [ ] **Step 6: Installer changes**

In `infra/scripts/install.sh`:
1. After the `err()` helper, add:
```bash
DRY_RUN="${ZENITH_DRY_RUN:-0}"   # tests: render .env in the current dir, no Docker/git/network

detect_public_ip() {
  if [ -n "${ZENITH_PUBLIC_IP:-}" ]; then echo "$ZENITH_PUBLIC_IP"; return; fi
  curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null \
    || hostname -I 2>/dev/null | awk '{print $1}'
}

gen_hex() { # 32 random bytes as 64 hex chars
  od -An -tx1 -N32 /dev/urandom | tr -d ' \n'
}
```
2. Wrap the Docker install/check section, the compose resolution, the `DOCKER_PREFIX` block, and the "Fetch the stack" section (clone, then `cd "$INSTALL_DIR"`) in `if [ "$DRY_RUN" != "1" ]; then … fi`. In dry-run the current directory is the install dir.
3. In the `.env` generation block, after the existing `set_var S3_SECRET_KEY …`, add:
```bash
  set_var S3_ACCESS_KEY "$(gen_secret 20)"
  set_var SECRETS_ENCRYPTION_KEY "$(gen_hex)"
  if [ -z "${ZENITH_DOMAIN:-}" ]; then
    PUBLIC_IP="$(detect_public_ip)"
    ZENITH_DOMAIN="${PUBLIC_IP}.sslip.io"
  fi
  set_var ZENITH_DOMAIN "$ZENITH_DOMAIN"
```
   Also change the creds file's `Dashboard: http://localhost:3000` to `Dashboard: https://${ZENITH_DOMAIN}`.
4. Wrap the `# ---- Start ----` section and the final banner in `if [ "$DRY_RUN" != "1" ]; then … fi`. In the banner, print:
   - `Dashboard:  https://$(grep '^ZENITH_DOMAIN=' .env | cut -d= -f2-)` in place of the localhost lines;
   - a DNS hint: `If you use your own domain, point it and *.<domain> (A records) at this server.`
   
   Remove the "set ZENITH_DOMAIN and ACME_EMAIL … --profile tls" lines.
5. Make sure `gen_secret`'s SIGPIPE guard stays as is.

`.env.example`: add commented documentation lines:
```
# Dashboard domain. Empty: the installer uses <public-ip>.sslip.io.
# Point <domain> and *.<domain> (A records) at this server for your own domain.
# ZENITH_DOMAIN=
# Generated by the installer (64 hex). Encrypts secrets at rest.
# SECRETS_ENCRYPTION_KEY=
# Show non-core (labs) pages in the dashboard.
# ZENITH_LABS=false
```

- [ ] **Step 7: Run tests**

Run: `bash infra/scripts/install_test.sh && (command -v shellcheck >/dev/null && shellcheck -S warning infra/scripts/install.sh infra/scripts/install_test.sh || echo "shellcheck not installed") && docker compose -f docker-compose.yml config --quiet && cd services/api && go test ./cmd/server/`
Expected:
- `install_test: ok`;
- shellcheck shows no warnings (or prints "not installed");
- `docker compose config` exits 0;
- Go ok.

- [ ] **Step 8: Commit**

```bash
git add services/api/cmd/server docker-compose.yml infra/scripts .env.example
git commit -m "fix(install): API creates the bucket (no minio/mc); encryption key and sslip.io domain generated; Traefik always on with le; only 80/443 public (F-04, F-07, F-12)"
```

---

### Task 13: Build, fresh install, re-run the audit scenario

**Files:**
- Modify: `docs/alpha/core-audit.md` (new §6)

**Interfaces:**
- Consumes: everything above.

This task is the acceptance run. It changes the audit host, so the executor follows these rules:
- Remove the old audit install (`cd /root/zenith-audit && docker compose --profile tls down`, keeping volumes is fine), and only that.
- **Do not push images** without asking the user.
- **Do not delete** anything under `/var/lib/rancher`.

- [ ] **Step 1: Build images locally**

```bash
cd /opt/Zenith
docker build -t ghcr.io/taikuri-infra/zenith-api:alpha-2a -f services/api/Dockerfile .
docker build -t ghcr.io/taikuri-infra/zenith-web:alpha-2a -f apps/web/Dockerfile .
```
Expected: both builds succeed. If the API Dockerfile's context is `services/api`, use the context its header comment names.

- [ ] **Step 2: Fresh install from this branch**

```bash
cd /root/zenith-audit && docker compose --profile tls down
cd /root && rm -rf zenith-2a
START=$(date +%s)
curl -fsSL https://raw.githubusercontent.com/taikuri-infra/Zenith/feat/alpha-2a-core-fixes/infra/scripts/install.sh \
  | ZENITH_DIR=zenith-2a ZENITH_BRANCH=feat/alpha-2a-core-fixes ZENITH_VERSION=alpha-2a ZENITH_DOMAIN=zenith.freezenith.com bash \
  > <scratchpad>/install-2a.log 2>&1
echo "install_seconds=$(( $(date +%s) - START ))"
```
This needs the branch pushed. Ask the user before pushing. If they decline, run `bash /opt/Zenith/infra/scripts/install.sh` with `ZENITH_REPO_URL=/opt/Zenith`.

`ZENITH_VERSION` must reach compose. If the installer does not pass it, add `ZENITH_VERSION=alpha-2a` to `/root/zenith-2a/.env` and run `docker compose up -d` again.

Expected:
- exit 0;
- `docker compose ps` shows postgres healthy, plus storage, api, web and traefik running;
- there is no `storage-init` service;
- `ss -ltnp | grep -E ':(3000|8080) '` shows only `127.0.0.1`.

- [ ] **Step 3: Re-run §3 scenario rows 1–10**

Re-run every row with the same harness as the audit (playwright-core script with `--no-sandbox`, curl, `zen`), **without** stage rerouting, plan override, or compose override.
- Step 2 uses `traefik/whoami:latest`.
- Step 6 uses `myapp.freezenith.com` (A record exists).
- Steps 4 and 9:
  1. Create Postgres, MySQL and Redis.
  2. Attach Postgres to whoami's app via the app's Databases tab, then click "Restart now".
  3. Check `docker exec zenith-app-… env | grep DATABASE_URL`.
  4. Write a row with `docker run --rm --network zenith_default postgres:16-alpine psql "<DATABASE_URL>" -c 'create table t(x int); insert into t values (1);'`.
- Step 8: `zen deploy --image traefik/whoami:v1.10`, then press Rollback on the Deployments tab (or `POST /rollback` with the previous deployment id), and check `docker ps` shows `traefik/whoami:latest`.

Also check:
- `curl -sS https://zenith.freezenith.com/` (no `-k`) returns 200;
- `curl -sS https://myapp.freezenith.com/` (no `-k`) returns 200;
- the browser network log has no request to any `freezenith.com` host other than `zenith.freezenith.com` and `myapp.freezenith.com`.

- [ ] **Step 4: Record §6 in the audit doc**

Append to `docs/alpha/core-audit.md`:
```markdown
## 6. Re-run after 2a

- **Date / commit / images:** <date>, <HEAD sha>, `zenith-{api,web}:alpha-2a` built from that commit
- **Install:** <seconds> s, <warnings or "none">
```
followed by the same 10-row table as §3, filled from Step 3. Every cell that is not `works` / `n/a` gets a new finding ID (F-21…) in §5. Commit:
```bash
git add docs/alpha/core-audit.md
git commit -m "docs(alpha): core audit — re-run after 2a"
```
````

Completion of Task 13 is the acceptance gate of the spec (§8). Any `broken` cell is reported to the user with its finding ID. 2a is only done when the user accepts the remaining gaps.
