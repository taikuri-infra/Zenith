# Alpha Core Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a factual record of what works in the Zenith Compose edition today (`docs/alpha/core-audit.md` plus one GitHub issue per finding), and hide non-core sidebar entries behind a `ZENITH_LABS` flag.

**Architecture:** Two small code changes support the audit. The first is a `ZENITH_DUMP_ROUTES=1` hook in the API that prints every registered route and exits, used for the static audit. The second is a pure `isNavItemVisible()` helper in the web app that the sidebar uses to apply `saasOnly` and the new `labs` tier. Everything else is investigation: a real install on the test server via `install.sh`, walking the core scenario, and recording results.

**Tech Stack:** Go 1.x + Fiber v2.52 (API), Next.js 15 + Vitest (web), Docker Compose, `gh` CLI.

**Spec:** `docs/superpowers/specs/2026-10-12-alpha-release-design.md` (§6 is this plan's scope).

## Global Constraints

- Self-hosted only; `ZENITH_MODE=standalone` is the mode under audit.
- **No changes to SaaS code.** Do not delete or edit Billing, Stripe, plans, customers, Hetzner, autoscaler, or Mission Control code.
- **Nothing found during the audit is fixed in this plan.** Findings become issues for sub-project 2.
- Core features (alpha scope): admin login, app deploy (Docker image or Compose stack), managed databases (Postgres, MySQL, Redis), custom domains with automatic HTTPS, env vars and secrets, deploy status and logs, rollback, and the `zen` CLI.
- Status vocabulary in the audit doc is exactly `works` / `broken` / `mock` / `partial`.
- Test server: `161.35.82.211` (this same machine; `/opt/Zenith` lives on it). k3s on it is stopped. Do not delete anything under `/var/lib/rancher/k3s/storage`.
- Commits use the repo identity: `git -c user.name="Babak Doraniarab" -c user.email="babak.dorani@gmail.com" commit …`, with the Co-Authored-By and Claude-Session trailers.
- Work on branch `docs/alpha-release-spec` (already checked out).

## Review Focus

1. **A section whose items are all hidden** (for example SECURITY in standalone with labs off) must not render an orphan section header. Test: Task 2, "drops sections with no visible items".
2. **`ZENITH_LABS` values other than exactly `true`** (`1`, `TRUE`, ` true `, empty, unset) must behave predictably: `true`, `1`, and `yes` in any case and with surrounding whitespace mean on; everything else means off. Test: Task 2, `parseLabsFlag` cases.
3. **SaaS mode must be unaffected:** in `saas` mode every item shows, labs or not, exactly as before. Test: Task 2, "saas mode shows everything".
4. **Flipping `ZENITH_LABS` must not need an image rebuild.** The flag has to go through `window.__ENV` via `entrypoint.sh`, like `NEXT_PUBLIC_ZENITH_MODE`. Check: Task 2, Step 7 grep.
5. **The overview page and other core pages may link to hidden labs pages.** The audit must record any such link so sub-project 2 can remove it. Check: Task 4, scenario step 10.

---

## File Structure

| File | Responsibility |
|---|---|
| `services/api/cmd/server/routes_dump.go` (create) | `dumpRoutes(app, w)`: deterministic, sorted, de-duplicated list of `METHOD /path` |
| `services/api/cmd/server/routes_dump_test.go` (create) | Unit test for `dumpRoutes` |
| `services/api/cmd/server/main.go` (modify, right after the `setupRoutes(app, …)` call at ~line 191) | If `ZENITH_DUMP_ROUTES=1`, dump and exit |
| `apps/web/src/lib/nav-visibility.ts` (create) | `parseLabsFlag()` and `isNavItemVisible()`, pure functions |
| `apps/web/src/lib/__tests__/nav-visibility.test.ts` (create) | Unit tests |
| `apps/web/src/lib/runtime-env.ts` (modify) | Export `LABS_ENABLED` |
| `apps/web/entrypoint.sh` (modify) | Inject `NEXT_PUBLIC_ZENITH_LABS` |
| `apps/web/src/components/sidebar.tsx` (modify) | Tag items with `labs: true` and filter through `isNavItemVisible` |
| `docker-compose.yml` (modify, web service env) | `NEXT_PUBLIC_ZENITH_LABS: ${ZENITH_LABS:-false}` |
| `docs/alpha/core-audit.md` (create) | The audit record |

---

### Task 1: Route dump hook in the API

**Files:**
- Create: `services/api/cmd/server/routes_dump.go`
- Create: `services/api/cmd/server/routes_dump_test.go`
- Modify: `services/api/cmd/server/main.go` (immediately after `provisioner, autoscaler, temporalW, tokenBlacklist, eventBus, redisClient := setupRoutes(app, cfg, …)` in `main`)

**Interfaces:**
- Produces: `func dumpRoutes(app *fiber.App, w io.Writer) error` (package `main`). Output is one `METHOD /path` per line, sorted by path then method, with HEAD and duplicates removed and `Use` middleware excluded.
- Produces: env var `ZENITH_DUMP_ROUTES=1`, which makes the server print routes to stdout and exit 0 before listening.

- [ ] **Step 1: Write the failing test**

`services/api/cmd/server/routes_dump_test.go`:
```go
package main

import (
	"bytes"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestDumpRoutes_SortedDedupedNoHeadNoUse(t *testing.T) {
	app := fiber.New()
	noop := func(c *fiber.Ctx) error { return nil }
	app.Use(noop)
	app.Post("/b", noop)
	app.Get("/b", noop) // Fiber also registers HEAD for GET
	app.Get("/a", noop)
	g := app.Group("/api/v1")
	g.Delete("/apps/:id", noop)

	var buf bytes.Buffer
	if err := dumpRoutes(app, &buf); err != nil {
		t.Fatalf("dumpRoutes: %v", err)
	}

	want := "GET /a\nDELETE /api/v1/apps/:id\nGET /b\nPOST /b\n"
	if got := buf.String(); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/api && go test ./cmd/server/ -run TestDumpRoutes -v`
Expected: FAIL, `undefined: dumpRoutes`

- [ ] **Step 3: Write the minimal implementation**

`services/api/cmd/server/routes_dump.go`:
```go
package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/gofiber/fiber/v2"
)

// dumpRoutes writes every registered route as "METHOD /path", one per line,
// sorted by path then method. HEAD (auto-added for GET), duplicates, and
// app.Use middleware are omitted. Used by ZENITH_DUMP_ROUTES=1 for audits.
func dumpRoutes(app *fiber.App, w io.Writer) error {
	type route struct{ method, path string }
	seen := map[string]bool{}
	var routes []route
	for _, r := range app.GetRoutes(true) {
		if r.Method == fiber.MethodHead {
			continue
		}
		key := r.Method + " " + r.Path
		if seen[key] {
			continue
		}
		seen[key] = true
		routes = append(routes, route{r.Method, r.Path})
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].path != routes[j].path {
			return routes[i].path < routes[j].path
		}
		return routes[i].method < routes[j].method
	})
	for _, r := range routes {
		if _, err := fmt.Fprintf(w, "%s %s\n", r.method, r.path); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/api && go test ./cmd/server/ -run TestDumpRoutes -v`
Expected: PASS

- [ ] **Step 5: Wire the env hook into `main`**

In `services/api/cmd/server/main.go`, directly after the `setupRoutes(app, cfg, …)` line inside `main()`:
```go
	if os.Getenv("ZENITH_DUMP_ROUTES") == "1" {
		if err := dumpRoutes(app, os.Stdout); err != nil {
			slog.Error("dump routes", "error", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
```
(`os` and `slog` are already imported in `main.go`.)

- [ ] **Step 6: Verify the hook end to end in standalone mode**

Run:
```bash
cd services/api && ZENITH_MODE=standalone JWT_SECRET=audit-only-secret-0123456789abcdef \
  ZENITH_DUMP_ROUTES=1 go run ./cmd/server > /tmp/claude-routes-standalone.txt; echo "exit=$?"
wc -l /tmp/claude-routes-standalone.txt; head -5 /tmp/claude-routes-standalone.txt
```
Expected: `exit=0` and a non-empty sorted list starting with paths such as `GET /health`. If the server panics or needs extra env without `DATABASE_URL`, record the exact error as an audit finding (Task 3) and add the minimum env needed to the command. Do not change server startup code to work around it.

- [ ] **Step 7: Run the package tests and vet**

Run: `cd services/api && go vet ./cmd/server/ && go test ./cmd/server/`
Expected: no vet output; `ok`

- [ ] **Step 8: Commit**

```bash
git add services/api/cmd/server/routes_dump.go services/api/cmd/server/routes_dump_test.go services/api/cmd/server/main.go
git -c user.name="Babak Doraniarab" -c user.email="babak.dorani@gmail.com" commit -m "feat(api): ZENITH_DUMP_ROUTES=1 prints registered routes and exits

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01T4AidohHj3qvUYek6RLV3F"
```

---

### Task 2: `ZENITH_LABS` flag and sidebar cut

**Files:**
- Create: `apps/web/src/lib/nav-visibility.ts`
- Create: `apps/web/src/lib/__tests__/nav-visibility.test.ts`
- Modify: `apps/web/src/lib/runtime-env.ts`
- Modify: `apps/web/entrypoint.sh`
- Modify: `apps/web/src/components/sidebar.tsx`
- Modify: `docker-compose.yml` (web service `environment:` block, next to `NEXT_PUBLIC_ZENITH_MODE: standalone`)

**Interfaces:**
- Produces: `parseLabsFlag(raw: string | undefined): boolean`
- Produces: `isNavItemVisible(item: { saasOnly?: boolean; labs?: boolean }, ctx: { standalone: boolean; labs: boolean }): boolean`
- Produces: `LABS_ENABLED: boolean` exported from `@/lib/runtime-env`
- Produces: runtime env `NEXT_PUBLIC_ZENITH_LABS` (compose var `ZENITH_LABS`, default `false`)

**Tier assignment** (standalone, labs off → visible set is exactly the core):
- Core (visible): Overview `/`, Apps `/apps`, Databases `/databases`, Domains `/networking`, Logs `/logs`, Docs `/docs`, Settings `/settings`
- Labs (`labs: true`): Storage `/storage`, Queues `/queues`, Auth `/auth`, IAM `/iam`, Activity `/activity`, Monitoring `/monitoring`, Registry `/registry`, CI/CD `/ci`
- SaaS (`saasOnly: true`, unchanged): API Gateway, SSO, Compliance, Audit Log, SSH Sessions, WAF Rules, Firewall, Alerts, Planets section, Marketplace, Support, Billing

- [ ] **Step 1: Write the failing tests**

`apps/web/src/lib/__tests__/nav-visibility.test.ts`:
```ts
import { describe, it, expect } from "vitest";
import { parseLabsFlag, isNavItemVisible } from "../nav-visibility";

describe("parseLabsFlag", () => {
  it.each(["true", "TRUE", " true ", "1", "yes", "Yes"])("%j is on", (v) => {
    expect(parseLabsFlag(v)).toBe(true);
  });
  it.each(["false", "0", "", "  ", "no", "enabled", undefined])("%j is off", (v) => {
    expect(parseLabsFlag(v)).toBe(false);
  });
});

describe("isNavItemVisible", () => {
  const core = {};
  const labs = { labs: true };
  const saas = { saasOnly: true };

  it("standalone, labs off: only core", () => {
    const ctx = { standalone: true, labs: false };
    expect(isNavItemVisible(core, ctx)).toBe(true);
    expect(isNavItemVisible(labs, ctx)).toBe(false);
    expect(isNavItemVisible(saas, ctx)).toBe(false);
  });

  it("standalone, labs on: core + labs, never saas", () => {
    const ctx = { standalone: true, labs: true };
    expect(isNavItemVisible(core, ctx)).toBe(true);
    expect(isNavItemVisible(labs, ctx)).toBe(true);
    expect(isNavItemVisible(saas, ctx)).toBe(false);
  });

  it("saas mode shows everything regardless of labs flag", () => {
    for (const l of [true, false]) {
      const ctx = { standalone: false, labs: l };
      expect(isNavItemVisible(core, ctx)).toBe(true);
      expect(isNavItemVisible(labs, ctx)).toBe(true);
      expect(isNavItemVisible(saas, ctx)).toBe(true);
    }
  });

  it("drops sections with no visible items", () => {
    const ctx = { standalone: true, labs: false };
    const security = [{ labs: true }, { labs: true }, { saasOnly: true }];
    expect(security.filter((i) => isNavItemVisible(i, ctx))).toHaveLength(0);
  });
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/web && pnpm vitest run src/lib/__tests__/nav-visibility.test.ts`
Expected: FAIL, cannot resolve `../nav-visibility`

- [ ] **Step 3: Write the minimal implementation**

`apps/web/src/lib/nav-visibility.ts`:
```ts
/**
 * Sidebar visibility rules for the self-hosted edition.
 *
 * - saasOnly items: shown only in saas mode.
 * - labs items: non-core features, hidden in standalone unless ZENITH_LABS is on.
 * - everything else is core and always shown.
 */

export interface NavVisibility {
  saasOnly?: boolean;
  labs?: boolean;
}

export interface NavContext {
  standalone: boolean;
  labs: boolean;
}

export function parseLabsFlag(raw: string | undefined): boolean {
  const v = (raw ?? "").trim().toLowerCase();
  return v === "true" || v === "1" || v === "yes";
}

export function isNavItemVisible(item: NavVisibility, ctx: NavContext): boolean {
  if (!ctx.standalone) return true;
  if (item.saasOnly) return false;
  if (item.labs) return ctx.labs;
  return true;
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd apps/web && pnpm vitest run src/lib/__tests__/nav-visibility.test.ts`
Expected: PASS (all cases)

- [ ] **Step 5: Export `LABS_ENABLED` from runtime-env**

In `apps/web/src/lib/runtime-env.ts`, add the import at the top (after the comment block, before `declare global`):
```ts
import { parseLabsFlag } from "./nav-visibility";
```
and append at the end of the file:
```ts
export const LABS_ENABLED = parseLabsFlag(
  getEnv("NEXT_PUBLIC_ZENITH_LABS", process.env.NEXT_PUBLIC_ZENITH_LABS, "false")
);
```

- [ ] **Step 6: Use the helper in the sidebar**

In `apps/web/src/components/sidebar.tsx`:

Replace
```ts
import { IS_STANDALONE } from "@/lib/runtime-env";

const isStandalone = IS_STANDALONE;

interface NavItem {
  name: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  saasOnly?: boolean;
}
```
with
```ts
import { IS_STANDALONE, LABS_ENABLED } from "@/lib/runtime-env";
import { isNavItemVisible } from "@/lib/nav-visibility";

const navCtx = { standalone: IS_STANDALONE, labs: LABS_ENABLED };
const visible = (i: { saasOnly?: boolean; labs?: boolean }) => isNavItemVisible(i, navCtx);

interface NavItem {
  name: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  saasOnly?: boolean;
  labs?: boolean;
}
```

Add `labs: true` to these entries in `navSections`:
```ts
      { name: "Storage", href: "/storage", icon: HardDrive, labs: true },
      { name: "Queues", href: "/queues", icon: ListOrdered, labs: true },
      { name: "Auth", href: "/auth", icon: Shield, labs: true },
      { name: "IAM", href: "/iam", icon: KeyRound, labs: true },
      { name: "Activity", href: "/activity", icon: History, labs: true },
      { name: "Monitoring", href: "/monitoring", icon: Activity, labs: true },
      { name: "Registry", href: "/registry", icon: Container, labs: true },
      { name: "CI/CD", href: "/ci", icon: GitBranch, labs: true },
```

Replace the three filters:
```ts
          .filter((s) => !isStandalone || !s.saasOnly)
```
→
```ts
          .filter((s) => visible(s))
```
```ts
            const items = section.items.filter(
              (item) => !isStandalone || !item.saasOnly
            );
```
→
```ts
            const items = section.items.filter(visible);
```
```ts
          .filter((item) => !isStandalone || !item.saasOnly)
```
→
```ts
          .filter(visible)
```
The existing `if (items.length === 0) return null;` already drops empty sections.

- [ ] **Step 7: Inject the flag at runtime**

In `apps/web/entrypoint.sh`, inside the `window.__ENV = { … }` block, after the `NEXT_PUBLIC_ZENITH_MODE` line:
```sh
  NEXT_PUBLIC_ZENITH_LABS: "${NEXT_PUBLIC_ZENITH_LABS:-false}",
```
In `docker-compose.yml`, in the web service `environment:` after `NEXT_PUBLIC_ZENITH_MODE: standalone`:
```yaml
      NEXT_PUBLIC_ZENITH_LABS: ${ZENITH_LABS:-false}
```
Verify:
```bash
grep -n "ZENITH_LABS" apps/web/entrypoint.sh docker-compose.yml apps/web/src/lib/runtime-env.ts
```
Expected: one hit in each file.

- [ ] **Step 8: Run the full web unit suite, typecheck, and lint**

Run: `cd apps/web && pnpm vitest run && pnpm exec tsc --noEmit && pnpm lint`
Expected: all tests pass, no type errors, no new lint errors in the touched files.

- [ ] **Step 9: Commit**

```bash
git add apps/web/src/lib/nav-visibility.ts apps/web/src/lib/__tests__/nav-visibility.test.ts \
  apps/web/src/lib/runtime-env.ts apps/web/entrypoint.sh apps/web/src/components/sidebar.tsx docker-compose.yml
git -c user.name="Babak Doraniarab" -c user.email="babak.dorani@gmail.com" commit -m "feat(web): ZENITH_LABS flag hides non-core sidebar entries in self-host

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01T4AidohHj3qvUYek6RLV3F"
```

---

### Task 3: Static audit

**Files:**
- Create: `docs/alpha/core-audit.md`

**Interfaces:**
- Consumes: `ZENITH_DUMP_ROUTES=1` (Task 1)
- Produces: `docs/alpha/core-audit.md` with sections `## 1. Static audit`, `## 2. Install`, `## 3. Core scenario`, `## 4. Reboot`, `## 5. Findings`. Tasks 4–6 fill sections 2–5.

- [ ] **Step 1: Create the audit doc skeleton**

`docs/alpha/core-audit.md`:
```markdown
# Zenith Alpha — Core Audit (Compose edition)

- **Date:** 2026-10-12
- **Commit audited:** <output of `git rev-parse --short HEAD`>
- **Images audited:** <`docker compose images` output from Task 4>
- **Host:** 161.35.82.211, Ubuntu 24.04, 4 vCPU / 16 GB
- **Spec:** docs/superpowers/specs/2026-10-12-alpha-release-design.md

Status values: `works` / `broken` / `mock` / `partial`.

## 1. Static audit
### 1.1 API routes in standalone mode
### 1.2 Sidebar pages in standalone mode (labs off)
### 1.3 Pages using mock or demo data

## 2. Install

## 3. Core scenario

| # | Step | UI | API | CLI | Notes / finding IDs |
|---|------|----|-----|-----|---------------------|

## 4. Reboot

## 5. Findings

| ID | Severity | Area | Summary | Repro | Issue |
|----|----------|------|---------|-------|-------|
```
Replace the `<…>` commit line with the real short SHA now.

- [ ] **Step 2: Record API routes grouped by core area**

Run:
```bash
cd services/api && ZENITH_MODE=standalone JWT_SECRET=audit-only-secret-0123456789abcdef \
  ZENITH_DUMP_ROUTES=1 go run ./cmd/server > /tmp/claude-routes-standalone.txt
wc -l < /tmp/claude-routes-standalone.txt
grep -ciE "/apps|/deploy|/releases|/rollback" /tmp/claude-routes-standalone.txt
grep -ciE "/databases|/db" /tmp/claude-routes-standalone.txt
grep -ciE "/domains" /tmp/claude-routes-standalone.txt
grep -ciE "/env" /tmp/claude-routes-standalone.txt
grep -ciE "/logs|/events" /tmp/claude-routes-standalone.txt
grep -ciE "billing|stripe|customer|plan|metering|campaign|hetzner|cluster|autoscal" /tmp/claude-routes-standalone.txt
```
Under §1.1, write: the total route count, the count per core area (apps/deploy/rollback, databases, domains, env, logs/events), and the count plus full list of SaaS-looking routes still registered in standalone (last grep, printed with `grep -iE … /tmp/claude-routes-standalone.txt`). Every SaaS-looking route that is reachable in standalone becomes a finding with severity `low`, area `api-surface`.

- [ ] **Step 3: Record sidebar pages**

From `apps/web/src/components/sidebar.tsx` after Task 2, list under §1.2 the entries visible with `standalone=true, labs=false`. It must equal the core list in Task 2 (Overview, Apps, Databases, Domains, Logs, Docs, Settings). Also list every directory under `apps/web/src/app/` that has a `page.tsx` but is in none of the three tiers:
```bash
cd apps/web/src/app && find . -name page.tsx -maxdepth 2 | sed 's#/page.tsx##; s#^\./##' | sort
```

- [ ] **Step 4: Record pages that use mock or demo data**

Run:
```bash
cd apps/web/src && grep -rlnE "mock-data|demo-api|demo-data|DEMO_MODE|mockData|MOCK_" app components hooks | sort
```
For each core page (`app/page.tsx`, `app/apps/**`, `app/databases/**`, `app/networking/**`, `app/logs/**`, `app/settings/**`), open it and check whether the mock import is used only when `DEMO_MODE` is true or unconditionally. Record each core page as `real`, `mock-when-demo`, or `mock-always` under §1.3. Every `mock-always` core page becomes a finding with severity `high`, area `ui-mock`.

- [ ] **Step 5: Commit**

```bash
git add docs/alpha/core-audit.md
git -c user.name="Babak Doraniarab" -c user.email="babak.dorani@gmail.com" commit -m "docs(alpha): core audit — static section

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01T4AidohHj3qvUYek6RLV3F"
```

---

### Task 4: Live install and core scenario

**Files:**
- Modify: `docs/alpha/core-audit.md` (sections 2 and 3)

**Interfaces:**
- Consumes: audit doc skeleton (Task 3)
- Produces: a running Compose install under `/root/zenith-audit`, admin credentials in `/root/zenith-audit/.env`, and filled sections 2–3

Base URL for app domains: `<name>.161.35.82.211.sslip.io` resolves to this host, so HTTPS can be tested without owning a domain.

- [ ] **Step 1: Pre-flight**

Run:
```bash
systemctl is-active k3s            # expect: inactive
ss -ltnp | grep -E ':(80|443|3000|8080) ' || echo "ports free"
df -h / | tail -1                  # record free space
docker version --format '{{.Server.Version}}'
```
If k3s is active or the ports are taken, stop and report. Do not kill unknown processes.

- [ ] **Step 2: Install exactly as a user would, timed**

Run:
```bash
cd /root && START=$(date +%s)
curl -fsSL https://raw.githubusercontent.com/taikuri-infra/Zenith/main/infra/scripts/install.sh \
  | ZENITH_DIR=zenith-audit bash 2>&1 | tee /tmp/claude-install.log
echo "install_seconds=$(( $(date +%s) - START ))"
```
Under §2 record: duration (pass if under 900 s), every prompt the installer asked, the printed dashboard URL and the admin user (**not the password**), and any warning or error lines (`grep -iE "warn|error|fail" /tmp/claude-install.log`). Then record `cd /root/zenith-audit && docker compose ps` and `docker compose images`. Any container not `running`/`healthy` is a finding (`critical`, area `install`).

- [ ] **Step 3: Walk the core scenario**

For each step, try it through the **UI** (browser at the dashboard URL; use the Playwright MCP tools), the **API** (`curl` with the admin token), and the **CLI** (`zen`, built with `cd /opt/Zenith/cli && go build -o /tmp/zen .`, then `zen login`). Fill one table row per step with `works` / `broken` / `mock` / `partial` per column, or `n/a` where that surface does not offer the action. Every non-`works` cell gets a finding ID (`F-01`, `F-02`, …) with exact repro steps in §5.

| # | Step | Pass condition |
|---|------|----------------|
| 1 | Admin login | Dashboard loads after login; `zen login` succeeds |
| 2 | Deploy from Docker image `traefik/whoami:latest` | App reaches running; its URL returns HTTP 200 |
| 3 | Deploy compose stack (below) | Both services run; web reaches db |
| 4 | Create Postgres, MySQL, Redis; attach Postgres to the app from step 2 | Each reaches ready; connection env is injected into the app |
| 5 | Set env var `AUDIT=1` and secret `TOKEN=s3cr3t` on the step 2 app | Redeploy shows both in the container (`docker exec … env`); the secret is not shown in plain text in the UI/API listing |
| 6 | Attach domain `whoami.161.35.82.211.sslip.io` | `curl -sSI https://whoami.161.35.82.211.sslip.io` returns 200 with a valid publicly trusted certificate (`curl` without `-k`) |
| 7 | Deploy logs and status | Build/deploy log visible live (SSE) and after the fact; status transitions are correct |
| 8 | Rollback | Deploy a second version (`traefik/whoami:v1.10`), roll back, and the running image matches the first |
| 9 | Data persists | Write a row to the Postgres from step 4 (`psql` via the connection string) for Task 5 |
| 10 | Hidden-link check | On every core page, list any link or button that leads to a labs or SaaS page |

Compose stack for step 3:
```yaml
services:
  web:
    image: traefik/whoami:latest
    ports: ["80"]
    depends_on: [db]
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_PASSWORD: audit
```

- [ ] **Step 4: Commit**

```bash
git add docs/alpha/core-audit.md
git -c user.name="Babak Doraniarab" -c user.email="babak.dorani@gmail.com" commit -m "docs(alpha): core audit — install and core scenario

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01T4AidohHj3qvUYek6RLV3F"
```

---

### Task 5: Reboot test

**Files:**
- Modify: `docs/alpha/core-audit.md` (section 4)

**Interfaces:**
- Consumes: running install and the Postgres row from Task 4 step 9

- [ ] **Step 1: Get the user's go-ahead**

This reboots the machine Claude is running on, so the session will drop. Ask the user to confirm the reboot and to resume the session afterwards. Do not reboot without an explicit yes.

- [ ] **Step 2: Disable k3s autostart and record pre-reboot state**

Run:
```bash
systemctl disable k3s
cd /root/zenith-audit && docker compose ps --format '{{.Name}} {{.State}}' > /root/claude-pre-reboot.txt
docker ps --format '{{.Names}} {{.Status}}' >> /root/claude-pre-reboot.txt
```

- [ ] **Step 3: Reboot** (after the user's yes)

Run: `systemctl reboot`

- [ ] **Step 4: After the session resumes, re-check**

Run:
```bash
sleep 60; cd /root/zenith-audit && docker compose ps
docker ps --format '{{.Names}} {{.Status}}' | diff /root/claude-pre-reboot.txt - || true
curl -sSI https://whoami.161.35.82.211.sslip.io | head -1
```
Then re-run scenario steps 1, 2 (URL only), 4 (databases ready), and 6, and read back the Postgres row from step 9. Record each under §4 with status. Anything that did not come back is a finding (`critical`, area `durability`).

- [ ] **Step 5: Commit**

```bash
git add docs/alpha/core-audit.md
git -c user.name="Babak Doraniarab" -c user.email="babak.dorani@gmail.com" commit -m "docs(alpha): core audit — reboot durability

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01T4AidohHj3qvUYek6RLV3F"
```

---

### Task 6: Findings to GitHub issues

**Files:**
- Modify: `docs/alpha/core-audit.md` (section 5, Issue column)

- [ ] **Step 1: Check gh auth**

Run: `gh auth status`
If not authenticated, stop and ask the user to run `! gh auth login`. Do not proceed without auth.

- [ ] **Step 2: Show the findings list to the user before creating anything**

Print the §5 table and ask for confirmation. Creating issues publishes them on the (public) repo.

- [ ] **Step 3: Create the milestone and labels** (after confirmation)

Run:
```bash
gh api repos/taikuri-infra/Zenith/milestones -f title=alpha.1 -f description="Compose edition alpha — see docs/superpowers/specs/2026-10-12-alpha-release-design.md" || true
for l in critical high medium low; do gh label create "sev:$l" --repo taikuri-infra/Zenith --force; done
```

- [ ] **Step 4: Create one issue per finding**

For each row:
```bash
gh issue create --repo taikuri-infra/Zenith --milestone alpha.1 --label "sev:<severity>" \
  --title "[alpha.1][<area>] <summary>" \
  --body "Found in core audit (docs/alpha/core-audit.md, <ID>).

**Repro**
<repro steps>

**Expected**
<pass condition from the scenario table>

**Actual**
<observed>"
```
Write each returned issue URL into the Issue column.

- [ ] **Step 5: Commit and push the branch**

```bash
git add docs/alpha/core-audit.md
git -c user.name="Babak Doraniarab" -c user.email="babak.dorani@gmail.com" commit -m "docs(alpha): core audit — findings linked to issues

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01T4AidohHj3qvUYek6RLV3F"
git push -u origin docs/alpha-release-spec
```
Then open a PR with `gh pr create` (body ends with the Claude Code attribution line), after confirming with the user.

---

## Self-Review Notes

- Spec §6 coverage: static audit A → Task 3 (with Task 1 hook); live audit B → Tasks 4–5; UI cut C → Task 2; deliverables 1/2/3 → Tasks 3–5, 6, and 2.
- The server restore (`systemctl enable --now k3s`) is deliberately not in this plan. k3s stays off through sub-project 2, which reuses this install; restore is a separate user decision.
