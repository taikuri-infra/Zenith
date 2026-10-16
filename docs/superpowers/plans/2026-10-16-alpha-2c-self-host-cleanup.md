# Alpha 2c — Self-host Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A self-hosted Zenith (Compose edition) shows no SaaS surface, logs in from the CLI with email and password, tells the truth about a custom domain's DNS and certificate, and shows managed-database variables in the Environment tab.

**Architecture:** Three independent lanes (API, web, CLI) share only JSON contracts written in this plan, so they run in parallel on separate worktrees and merge into the feature branch. The API gates SaaS routes by `cfg.IsStandalone()`, adds a domain checker goroutine (DNS + TLS against Traefik) and a `ManagedEnv` helper that writes env-v2 rows; the web branches on `IS_STANDALONE`; the CLI stores access and refresh tokens and refreshes them itself.

**Tech Stack:** Go 1.25 (fiber v2, pgx, cobra, lipgloss), Next.js 14 / React (vitest + testing-library), Docker Compose, GitHub Actions.

**Spec:** docs/superpowers/specs/2026-10-16-alpha-2c-self-host-cleanup-design.md

## Global Constraints

- `ZENITH_MODE=saas` behaviour must not change: every gate is `cfg.IsStandalone()` (API) or `IS_STANDALONE` (web). No SaaS code is deleted.
- Commit messages: conventional commits, scope from `[api, web, cli, ci, infra, installer]`, subject lower-case. Trailer on every commit: `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.
- Go tests run with `go test ./... -race -count=1` in `services/api` and `cli`; web with `pnpm --filter @zenith/web test` and `pnpm --filter @zenith/web exec tsc --noEmit`; `gofmt -l` must print nothing for touched files.
- Author identity for commits: `git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit …` (the host has no global git identity).
- Registration refusal text, verbatim: `registration is disabled on self-hosted installs; the administrator account is created by the installer`.
- Domain messages, verbatim: `waiting for DNS check`; `DNS: no A/AAAA record for <domain>; create an A record to <ip>`; `DNS: <domain> points to <ips>; expected <ip>`; `DNS ok; waiting for the certificate`; `TLS check failed: <err>`; active → empty message.
- Login page tagline in standalone, verbatim: `Self-hosted PaaS`.
- CLI session-expired error, verbatim: `session expired — run zen login`.
- Version after this plan: `VERSION` = `0.1.0-alpha.4`.

## Review Focus

1. **A domain whose DNS later moves away.** The checker must send an `active` domain back to `pending` with `tls_ready=false` instead of leaving a green row (Task 6 test `TestDomainChecker_ActiveGoesBackToPendingWhenDNSMoves`).
2. **A legacy env row that is already migrated.** Running the startup migration twice must not duplicate or re-delete anything (Task 7 test `TestMigrateManagedEnv_Idempotent`).
3. **A CLI config written by an older zen (token only, no refresh token).** Commands must keep working with the bare token and never attempt a refresh (Task 3 test `TestClient_NoRefreshWithoutRefreshToken`).
4. **A 401 that is not an expired token (wrong password, revoked session).** The client refreshes once at most and then reports the session-expired error; it must not loop (Task 3 test `TestClient_RefreshFailureIsReportedOnce`).
5. **A user who opens `/login?mode=register` on a self-host.** The page must show the login form, not a register form that would 403 (Task 2 test `shows the login form even with ?mode=register`).

## Execution: lanes and merge order

Create the three lane worktrees from the feature branch `feat/alpha-2c-self-host-cleanup` (spec commit `e622934`):

```bash
cd /opt/Zenith
git worktree add .worktrees/2c-api -b feat/alpha-2c-api feat/alpha-2c-self-host-cleanup
git worktree add .worktrees/2c-web -b feat/alpha-2c-web feat/alpha-2c-self-host-cleanup
git worktree add .worktrees/2c-cli -b feat/alpha-2c-cli feat/alpha-2c-self-host-cleanup
```

| Lane | Tasks (in order) | Touches |
|------|------------------|---------|
| API | 1 → 5 → 6 → 7 | `services/api/**` only |
| Web | 2 → 8 | `apps/web/**`, `.github/workflows/ci.yml` |
| CLI | 3 → 4 | `cli/**` only |
| Controller | 9 → 10 | merges lanes, `VERSION`, docs, live acceptance |

No two lanes touch the same file, so the merges in Task 9 are conflict-free. Contracts between lanes (read them before starting any task):

- `GET /api/v1/version` → `{"version":"…","build_time":"…","go_version":"…","mode":"standalone"|"saas"}` (Task 1 produces, Task 4 consumes).
- `POST /api/v1/auth/register` in standalone → `403 {"error":"registration is disabled on self-hosted installs; the administrator account is created by the installer"}` (Task 1).
- Domain JSON gains `"status_message":"…","last_checked_at":"2026-10-16T19:00:00Z"|null`; `POST /api/v1/apps/:appId/domains/:domainId/verify` → `200` with the same domain JSON (Tasks 5–6 produce, Task 8 consumes).
- `GET /api/v1/domains` (exists) lists the user's custom domains with the same JSON (Task 8 consumes).
- env-v2 items from a managed database: `"source":"managed_service","source_id":"<database id>"` (Task 7 produces, Task 8 consumes).
- `POST /api/v1/auth/login` → `{"access_token","refresh_token","token_type","expires_in"}` or `{"mfa_required":true,"mfa_token":"…"}`; `POST /api/v1/auth/login/mfa` `{"mfa_token","code"}` → token JSON; `POST /api/v1/auth/refresh` `{"refresh_token"}` → token JSON; `POST /api/v1/auth/logout` with Bearer → 200 (all exist today; Task 3 consumes).
- `GET /api/v1/apps?project_id=<id>` → `{"items":[…]}` (exists; Task 4 consumes instead of the legacy `/projects/:id/apps`).

---

### Task 1: Standalone mode in the API — route gating, registration off, version `mode`

**Lane:** API. **Files:**
- Modify: `services/api/internal/config/config.go` (add method after `Load`)
- Modify: `services/api/cmd/server/main.go:671` (version), after `authHandler` construction (register gate), `:1245-1252`, `:1275-1283`, `:1622-1624`, `:1649`, `:1704-1705`, `:1712`, `:1726-1730`
- Modify: `services/api/internal/handlers/auth.go:18-27,181`
- Modify: `services/api/internal/handlers/health.go:24-28,68-76`
- Create: `services/api/internal/config/standalone_test.go`, `services/api/internal/handlers/auth_register_test.go`, `services/api/internal/handlers/version_test.go`, `services/api/cmd/server/routes_mode_test.go`

**Interfaces:**
- Produces: `func (c *Config) IsStandalone() bool`; `func (h *AuthHandler) SetRegistrationDisabled(disabled bool)`; `func VersionInfo(version, buildTime, mode string) fiber.Handler`; `VersionResponse.Mode string` with json tag `mode`.

- [ ] **Step 1: Write the failing config test**

```go
// services/api/internal/config/standalone_test.go
package config

import "testing"

func TestIsStandalone(t *testing.T) {
	cases := map[string]bool{"": true, "standalone": true, "saas": false}
	for mode, want := range cases {
		c := &Config{Mode: mode}
		if got := c.IsStandalone(); got != want {
			t.Errorf("Mode %q: IsStandalone() = %v, want %v", mode, got, want)
		}
	}
}
```

- [ ] **Step 2: Run it to see it fail**

Run: `cd services/api && go test ./internal/config/ -run TestIsStandalone -count=1`
Expected: FAIL, `c.IsStandalone undefined`.

- [ ] **Step 3: Add the method**

In `services/api/internal/config/config.go`, after `func Load() *Config { … }`:

```go
// IsStandalone reports whether this install is self-hosted (the default).
// Only ZENITH_MODE=saas turns it off.
func (c *Config) IsStandalone() bool { return c.Mode != "saas" }
```

Run the test again. Expected: PASS.

- [ ] **Step 4: Write the failing version-handler test**

```go
// services/api/internal/handlers/version_test.go
package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func TestVersionInfoReportsMode(t *testing.T) {
	app := fiber.New()
	app.Get("/version", handlers.VersionInfo("0.1.0-alpha.4", "2026-10-16", "standalone"))
	resp, _ := app.Test(httptest.NewRequest("GET", "/version", nil), -1)
	var body handlers.VersionResponse
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Version != "0.1.0-alpha.4" || body.Mode != "standalone" {
		t.Fatalf("got %+v", body)
	}
}
```

Run: `go test ./internal/handlers/ -run TestVersionInfoReportsMode -count=1` → FAIL (too many arguments / no field Mode).

- [ ] **Step 5: Add `Mode` to the version response**

`services/api/internal/handlers/health.go`:

```go
type VersionResponse struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	Mode      string `json:"mode"`
}

// VersionInfo reports the build and the install mode ("standalone" or "saas").
func VersionInfo(version, buildTime, mode string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(VersionResponse{
			Version:   version,
			BuildTime: buildTime,
			GoVersion: runtime.Version(),
			Mode:      mode,
		})
	}
}
```

`services/api/cmd/server/main.go:671` becomes:

```go
	api.Get("/version", handlers.VersionInfo(Version, BuildTime, cfg.Mode))
```

Run: `go build ./... && go test ./internal/handlers/ -run TestVersionInfoReportsMode -count=1` → PASS.

- [ ] **Step 6: Write the failing registration tests**

```go
// services/api/internal/handlers/auth_register_test.go
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

func TestRegisterDisabledInStandalone(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	h := handlers.NewAuthHandler(nil, nil)
	h.SetRegistrationDisabled(true)
	app.Post("/api/v1/auth/register", h.Register)

	body := `{"email":"a@b.c","password":"longenough","name":"A"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 403 {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var out map[string]string
	json.NewDecoder(resp.Body).Decode(&out)
	want := "registration is disabled on self-hosted installs; the administrator account is created by the installer"
	if out["error"] != want {
		t.Fatalf("error = %q, want %q", out["error"], want)
	}
}

func TestRegisterStillValidatesWhenEnabled(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	h := handlers.NewAuthHandler(nil, nil) // svc is never reached: the body is invalid
	app.Post("/api/v1/auth/register", h.Register)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400 (gate must not apply when enabled)", resp.StatusCode)
	}
}
```

Run: `go test ./internal/handlers/ -run 'TestRegister' -count=1` → FAIL (`SetRegistrationDisabled` undefined).

- [ ] **Step 7: Add the gate to the handler**

`services/api/internal/handlers/auth.go`:

```go
type AuthHandler struct {
	svc                  *services.AuthService
	appURL               string // frontend URL for OAuth redirect
	blacklist            *middleware.TokenBlacklist
	eventRepo            ports.UserEventRepository
	registrationDisabled bool
}

// SetRegistrationDisabled turns POST /auth/register into a 403. Self-hosted
// installs have one administrator, created by the installer.
func (h *AuthHandler) SetRegistrationDisabled(disabled bool) { h.registrationDisabled = disabled }
```

At the top of `Register`, before `var req registerRequest`:

```go
	if h.registrationDisabled {
		return fiber.NewError(fiber.StatusForbidden,
			"registration is disabled on self-hosted installs; the administrator account is created by the installer")
	}
```

Check that `handlers.ErrorHandler` renders `fiber.Error` as `{"error": msg}` (the other handler tests rely on it; if the key differs, match the test to the real key and keep the message verbatim).

In `main.go` right after `authHandler` is constructed (search `authHandler := handlers.NewAuthHandler(`), add:

```go
	authHandler.SetRegistrationDisabled(cfg.IsStandalone())
```

Run: `go test ./internal/handlers/ -run 'TestRegister' -count=1` → PASS.

- [ ] **Step 8: Write the failing route-list test**

```go
// services/api/cmd/server/routes_mode_test.go
package main

import (
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/config"
	"github.com/gofiber/fiber/v2"
)

// saasOnlyPrefixes are route families that must not exist on a self-hosted install.
var saasOnlyPrefixes = []string{
	"/api/v1/billing", "/api/v1/plan", "/api/v1/referral",
	"/api/v1/admin/plans", "/api/v1/admin/billing", "/api/v1/admin/exit-surveys",
	"/api/v1/admin/referrals", "/api/v1/admin/crm", "/api/v1/admin/users/:userId/plan",
}

func registeredPaths(t *testing.T, mode string) map[string]bool {
	t.Helper()
	cfg := &config.Config{Mode: mode, JWTSecret: "test-secret-test-secret-test-secret", K8sMode: "memory", BaseDomain: "example.com", CORSOrigins: "http://localhost:3000"}
	app := fiber.New()
	setupRoutes(app, cfg,
		memory.NewMemoryUserRepository(), memory.NewMemoryAdminRepository(),
		memory.NewMemoryCustomerRepository(), memory.NewMemoryMeteringRepository(),
		memory.NewMemoryAppRepository(), nil)
	paths := map[string]bool{}
	for _, r := range app.GetRoutes(true) {
		paths[r.Path] = true
	}
	return paths
}

func TestStandaloneRegistersNoSaaSRoutes(t *testing.T) {
	paths := registeredPaths(t, "standalone")
	for p := range paths {
		for _, prefix := range saasOnlyPrefixes {
			if strings.HasPrefix(p, prefix) {
				t.Errorf("standalone registered SaaS route %s", p)
			}
		}
	}
	if !paths["/api/v1/auth/register"] || !paths["/api/v1/apps"] {
		t.Fatalf("core routes missing: %v", paths)
	}
}
```

Run: `go test ./cmd/server/ -run TestStandaloneRegistersNoSaaSRoutes -count=1` → FAIL listing the 24 routes. If `setupRoutes` panics on a nil dependency in the test, add the missing `config.Config` field it reads (keep the test's config minimal; the audit proved `ZENITH_MODE=standalone` with no env starts cleanly, so only fields `Load()` defaults are candidates).

- [ ] **Step 9: Gate the registrations**

In `services/api/cmd/server/main.go` wrap each block exactly as follows (handlers and services above them stay unconditional):

```go
	if !cfg.IsStandalone() {
		protected.Get("/plan", planHandler.GetMyPlan)
		protected.Post("/plan/upgrade", planHandler.UpgradePlan)
	}
```

```go
	if !cfg.IsStandalone() {
		protected.Get("/referral", referralHandler.GetSummary)
		protected.Get("/referral/rewards", referralHandler.ListRewards)
		protected.Post("/referral/share", referralHandler.TrackShare)
	}
```

```go
	if !cfg.IsStandalone() {
		protected.Get("/billing", billingHandler.GetBillingStatus)
		protected.Post("/billing/checkout", billingHandler.CreateCheckoutSession)
		protected.Post("/billing/portal", billingHandler.CreatePortalSession)
		protected.Post("/billing/cancel", billingHandler.CancelSubscription)
		protected.Get("/billing/invoices", billingHandler.ListInvoices)
	}
```

```go
	if !cfg.IsStandalone() {
		protected.Post("/billing/exit-survey", exitSurveyHandler.SubmitAndCancel)
	}
```

```go
		if !cfg.IsStandalone() {
			admin.Get("/plans", customerHandler.ListPlans)
			admin.Post("/plans", customerHandler.CreatePlan)
			admin.Put("/plans/:id", customerHandler.UpdatePlan)
		}
```

```go
	if !cfg.IsStandalone() {
		admin.Post("/users/:userId/plan", adminUserHandler.SetUserPlan)
	}
```

```go
	if !cfg.IsStandalone() {
		admin.Get("/exit-surveys", exitSurveyHandler.AdminList)
		admin.Get("/exit-surveys/stats", exitSurveyHandler.AdminStats)
	}
```

```go
	if !cfg.IsStandalone() {
		admin.Get("/referrals", referralHandler.AdminList)
	}
```

```go
	crmHandler := handlers.NewAdminCRMHandler(pool)
	if !cfg.IsStandalone() {
		admin.Get("/crm/pipeline", crmHandler.GetPipeline)
		admin.Get("/crm/health-scores", crmHandler.GetHealthScores)
		admin.Post("/crm/customers/:id/notes", crmHandler.SaveNote)
		admin.Get("/crm/customers/:id/notes", crmHandler.GetNotes)
		admin.Put("/crm/customers/:id/tags", crmHandler.UpdateTags)
	}
```

The existing `if cfg.Mode == "saas" { admin.Get("/billing/overview", …) }` stays as it is.

Run: `go build ./... && go test ./cmd/server/ ./internal/handlers/ ./internal/config/ -count=1` → PASS. Then `gofmt -l cmd internal`.

- [ ] **Step 10: Commit**

```bash
git add services/api
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(api): standalone mode hides saas routes, refuses registration, reports mode

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 2: Web — login page, Docs, Overview, deploy wizard, Databases, TypeScript errors, `tsc` in CI

**Lane:** Web. **Files:**
- Modify: `apps/web/src/app/login/page.tsx` (imports; lines 21-23; 300-302; 316-363 OAuth block and divider; the register toggle near the bottom of the form)
- Modify: `apps/web/src/app/docs/page.tsx` (whole file reorganised into `StandaloneDocs` and `SaasDocs`)
- Modify: `apps/web/src/app/page.tsx:66-68,278-282,283-287`
- Modify: `apps/web/src/components/deploy-wizard.tsx:909-914`
- Modify: `apps/web/src/app/databases/page.tsx:106-121`
- Modify: `apps/web/src/hooks/__tests__/use-auth.test.ts:52-57,72-77,93-98`
- Modify: `.github/workflows/ci.yml:61-62`
- Create: `apps/web/src/lib/database-summary.ts`, `apps/web/src/lib/__tests__/database-summary.test.ts`, `apps/web/src/app/login/__tests__/page.standalone.test.tsx`, `apps/web/src/app/docs/__tests__/page.test.tsx`

**Interfaces:**
- Consumes: `IS_STANDALONE` from `@/lib/runtime-env`, `SHOW_PLANS` from `@/lib/edition`.
- Produces: `databaseSummary(dbs: { engine: string; status: string }[]): string`.

- [ ] **Step 1: Fix the three TypeScript errors (F-03)**

In `apps/web/src/hooks/__tests__/use-auth.test.ts` the three `mockResolvedValue` fixtures must satisfy `LoginResponse` (`access_token`, `refresh_token`, `token_type`, `expires_in` required; `mfa_required`/`mfa_token` optional). Replace them:

```ts
    // login success (two places: "login success sets user state" and "logout clears user state")
    vi.mocked(auth.login).mockResolvedValue({
      access_token: jwt,
      refresh_token: "refresh",
      token_type: "Bearer",
      expires_in: 3600,
    });
```

```ts
    // MFA required
    vi.mocked(auth.login).mockResolvedValue({
      access_token: "",
      refresh_token: "",
      token_type: "Bearer",
      expires_in: 0,
      mfa_required: true,
      mfa_token: "mfa-token-123",
    });
```

Run: `cd apps/web && pnpm exec tsc --noEmit && pnpm exec vitest run src/hooks/__tests__/use-auth.test.ts` → both clean.

- [ ] **Step 2: Add the typecheck step to CI**

`.github/workflows/ci.yml`, after the `Lint frontend` step:

```yaml
      - name: Typecheck frontend
        run: pnpm --filter @zenith/web exec tsc --noEmit
```

- [ ] **Step 3: Write the failing standalone login-page test**

```tsx
// apps/web/src/app/login/__tests__/page.standalone.test.tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

let searchParams = new URLSearchParams();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn() }),
  useSearchParams: () => searchParams,
}));
vi.mock("@/hooks/use-auth", () => ({
  useAuth: () => ({
    login: vi.fn(), mfaLogin: vi.fn(), register: vi.fn(), logout: vi.fn(),
    mfaToken: null, user: null, isAuthenticated: false, loading: false,
  }),
}));
vi.mock("@/lib/api", () => ({ auth: { getOAuthUrl: vi.fn().mockReturnValue("#") } }));
vi.mock("@/lib/get-api", () => ({ isDemoMode: vi.fn(() => false) }));
vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true }));

import LoginPage from "../page";

describe("LoginPage (standalone)", () => {
  beforeEach(() => { searchParams = new URLSearchParams(); });

  it("shows only email and password", () => {
    render(<LoginPage />);
    expect(screen.getByPlaceholderText("you@example.com")).toBeInTheDocument();
    expect(screen.queryByText("Google")).toBeNull();
    expect(screen.queryByText("GitHub")).toBeNull();
    expect(screen.queryByText(/create.*account|sign.*up/i)).toBeNull();
    expect(screen.queryByText(/Hetzner|Kubernetes/)).toBeNull();
    expect(screen.getByText("Self-hosted PaaS")).toBeInTheDocument();
  });

  it("shows the login form even with ?mode=register", () => {
    searchParams = new URLSearchParams("mode=register");
    render(<LoginPage />);
    expect(screen.getByText(/sign in to your account/i)).toBeInTheDocument();
    expect(screen.queryByPlaceholderText(/name/i)).toBeNull();
  });
});
```

Run: `pnpm exec vitest run src/app/login` → the new file FAILS; the existing `page.test.tsx` (SaaS rendering) must keep passing. If `runtime-env` resolves to standalone in tests (`NEXT_PUBLIC_ZENITH_MODE` unset → `IS_STANDALONE === true`), add `vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: false }))` to the top of the existing `page.test.tsx` so it keeps testing the SaaS form.

- [ ] **Step 4: Change the login page**

`apps/web/src/app/login/page.tsx`:

```tsx
import { IS_STANDALONE } from "@/lib/runtime-env";
```

```tsx
  const initialMode = !IS_STANDALONE && searchParams.get("mode") === "register" ? "register" : "login";
```

Tagline:

```tsx
          <p className="text-neutral-400 text-sm">
            {IS_STANDALONE ? "Self-hosted PaaS" : "Kubernetes-native PaaS on Hetzner Cloud"}
          </p>
```

Wrap the OAuth buttons grid **and** the "or continue with email" divider in `{!IS_STANDALONE && ( … )}`. Wrap the register/sign-in toggle at the bottom of the form (the element the existing test finds with `/create.*account|sign.*up|register/i`) in `{!IS_STANDALONE && ( … )}`.

Run: `pnpm exec vitest run src/app/login && pnpm exec tsc --noEmit` → PASS.

- [ ] **Step 5: Write the failing Docs test**

```tsx
// apps/web/src/app/docs/__tests__/page.test.tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/components/shell", () => ({ Shell: ({ children }: { children: React.ReactNode }) => <div>{children}</div> }));
vi.mock("@/lib/runtime-env", () => ({ IS_STANDALONE: true, ZENITH_MODE: "standalone" }));

import DocsPage from "../page";

describe("DocsPage (standalone)", () => {
  it("documents the compose edition only", () => {
    const { container } = render(<DocsPage />);
    const text = container.textContent || "";
    expect(text).not.toMatch(/freezenith\.com/);
    expect(text).not.toMatch(/Pro plan|sign up/i);
    expect(text).not.toMatch(/Enable Auth|Add Storage|Git Push/);
    expect(text).toMatch(/zen backup/);
    expect(text).toMatch(/zen upgrade/);
    expect(text).toMatch(/A record/);
    expect(screen.getByText(/Custom domain and HTTPS/)).toBeInTheDocument();
  });
});
```

Run: `pnpm exec vitest run src/app/docs` → FAIL.

- [ ] **Step 6: Reorganise the Docs page**

In `apps/web/src/app/docs/page.tsx` rename the current `export default function DocsPage()` to `function SaasDocs()` (no other change to it), add `import { IS_STANDALONE } from "@/lib/runtime-env";`, and add:

```tsx
function dashboardOrigin(): string {
  return typeof window !== "undefined" ? window.location.origin : "https://<your-dashboard-domain>";
}

function StandaloneDocs() {
  const origin = dashboardOrigin();
  return (
    <Shell>
      <div className="mx-auto max-w-3xl space-y-8">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <Rocket className="h-5 w-5 text-accent-400" />
            <h1 className="text-lg font-semibold text-white">Deploy an app on your own server</h1>
          </div>
          <p className="text-sm text-neutral-500">
            Zenith runs on this server with Docker Compose. Everything below works from the dashboard or the zen CLI.
          </p>
        </div>

        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <h2 className="mb-2 text-sm font-medium text-white">Prerequisites</h2>
          <ul className="space-y-1 text-xs text-neutral-400">
            <li>The administrator account the installer created (ADMIN_EMAIL / ADMIN_PASSWORD)</li>
            <li>A container image or a docker-compose.yml for your app</li>
            <li>Optional: the zen CLI on your laptop</li>
          </ul>
          <div className="mt-3">
            <CodeBlock code={`curl -fsSL https://github.com/taikuri-infra/Zenith/releases/latest/download/zen_linux_amd64 -o zen && chmod +x zen
./zen login --endpoint ${origin}`} />
          </div>
        </div>

        <div className="space-y-8">
          <Step number={1} title="Create an app" icon={GitBranch}>
            <p>
              Go to <strong className="text-white">Apps</strong> → <strong className="text-white">New App</strong>. Deploy a public image
              (for example <code className="text-neutral-300">traefik/whoami:latest</code>) or paste a docker-compose.yml. Each service
              of a compose file becomes an app in the same project, and services reach each other by their compose service name.
            </p>
            <CodeBlock code={`zen deploy --image traefik/whoami:latest`} />
          </Step>

          <Step number={2} title="Add a database and attach it" icon={Database}>
            <p>
              <strong className="text-white">Databases</strong> → <strong className="text-white">Create Database</strong> (PostgreSQL, MySQL or Redis).
              Attach it to an app: the connection string is injected as <code className="text-neutral-300">DATABASE_URL</code>,
              <code className="text-neutral-300">MYSQL_URL</code> or <code className="text-neutral-300">REDIS_URL</code> and the app restarts.
              From your laptop, open an SSH tunnel to the database:
            </p>
            <CodeBlock code={`zen db connect <name> --tunnel`} />
          </Step>

          <Step number={3} title="Environment variables and secrets" icon={Shield}>
            <p>
              The app's <strong className="text-white">Environment</strong> tab holds plain variables and secrets; secrets are stored encrypted
              and masked. Variables injected by an attached database are read-only there; detach the database to remove them.
              Click <strong className="text-white">Restart now</strong> after a change.
            </p>
          </Step>

          <Step number={4} title="Custom domain and HTTPS" icon={Globe}>
            <p>
              Create an <strong className="text-white">A record</strong> for your domain pointing at this server's public IP, then add the domain in the app's
              <strong className="text-white"> Domains</strong> tab. The domain shows <em>pending</em> until DNS points here, <em>verified</em> while the
              certificate is being issued, and <em>active</em> when HTTPS works. Use <strong className="text-white">Check now</strong> to re-check at once.
            </p>
          </Step>

          <Step number={5} title="Logs and rollback" icon={Terminal}>
            <p>
              The <strong className="text-white">Logs</strong> tab streams the container's output. <strong className="text-white">Deployments</strong> lists every
              release; <strong className="text-white">Rollback</strong> redeploys a previous image and keeps the domain and variables.
            </p>
            <CodeBlock code={`zen logs <app> --follow`} />
          </Step>

          <Step number={6} title="Backup and upgrade" icon={HardDrive}>
            <p>
              On the server, one archive holds the platform database, object storage, every managed database and the config.
              Upgrades take a backup first and roll back on a failed health check.
            </p>
            <CodeBlock code={`zen backup
zen backup restore <archive>
zen upgrade --version <next>`} />
          </Step>
        </div>

        <div className="rounded-lg border border-border bg-surface-100 p-5">
          <h2 className="mb-3 text-sm font-medium text-white">API</h2>
          <p className="mb-2 text-xs text-neutral-400">The dashboard uses the same API. Log in to get a token, then call it with a Bearer header.</p>
          <CodeBlock code={`curl -s ${origin}/api/v1/auth/login -H 'Content-Type: application/json' \\
  -d '{"email":"you@example.com","password":"…"}'
curl -s ${origin}/api/v1/apps -H "Authorization: Bearer $TOKEN"`} />
        </div>
      </div>
    </Shell>
  );
}

export default function DocsPage() {
  return IS_STANDALONE ? <StandaloneDocs /> : <SaasDocs />;
}
```

Run: `pnpm exec vitest run src/app/docs && pnpm exec tsc --noEmit && pnpm --filter @zenith/web lint` → PASS.

- [ ] **Step 7: Overview, wizard — gate plan content**

`apps/web/src/app/page.tsx`:

```tsx
  const {
    data: planData,
  } = useApi(() => (SHOW_PLANS ? userPlan.get() : Promise.resolve(null)), []);
```

(`SHOW_PLANS` is already imported in this file; if not, `import { SHOW_PLANS } from "@/lib/edition";`.) Wrap the Plan tile:

```tsx
          {SHOW_PLANS && (
            <StatCard
              label="Plan"
              value={planData?.tier || "--"}
              sub={project?.slug || ""}
            />
          )}
```

and change `label="Legacy Apps"` to `label="Apps"`.

`apps/web/src/components/deploy-wizard.tsx:909-914`:

```tsx
          {SHOW_PLANS && (
            <p className="mt-1.5 text-[11px] text-neutral-500">
              {isPro ? "Pro plan" : "Free plan"} per-app limit.
              {!isPro && (
                <span className="ml-1 text-amber-400">Upgrade to Pro for more resources.</span>
              )}
            </p>
          )}
```

Run: `pnpm exec tsc --noEmit` → clean.

- [ ] **Step 8: Databases header — failing test, helper, use it**

```ts
// apps/web/src/lib/__tests__/database-summary.test.ts
import { describe, it, expect } from "vitest";
import { databaseSummary } from "../database-summary";

describe("databaseSummary", () => {
  it("counts every engine, including MySQL", () => {
    const s = databaseSummary([
      { engine: "postgresql", status: "ready" },
      { engine: "mysql", status: "ready" },
      { engine: "redis", status: "provisioning" },
    ]);
    expect(s).toBe("3 instances, 2 ready, 1 PostgreSQL, 1 MySQL, 1 Redis");
  });
  it("omits engines with no instances", () => {
    expect(databaseSummary([{ engine: "redis", status: "ready" }])).toBe("1 instances, 1 ready, 1 Redis");
  });
});
```

```ts
// apps/web/src/lib/database-summary.ts
const ENGINES: [string, string][] = [["postgresql", "PostgreSQL"], ["mysql", "MySQL"], ["redis", "Redis"]];

export function databaseSummary(dbs: { engine: string; status: string }[]): string {
  const ready = dbs.filter((d) => d.status === "ready").length;
  const parts = [`${dbs.length} instances`, `${ready} ready`];
  for (const [engine, label] of ENGINES) {
    const n = dbs.filter((d) => d.engine === engine).length;
    if (n > 0) parts.push(`${n} ${label}`);
  }
  return parts.join(", ");
}
```

In `apps/web/src/app/databases/page.tsx` import it, delete `pgCount` and `redisCount` (keep `readyCount` if used elsewhere in the file), and render:

```tsx
            <p className="text-sm text-neutral-500">{databaseSummary(dbs)}</p>
```

Run: `pnpm exec vitest run src/lib && pnpm exec tsc --noEmit && pnpm --filter @zenith/web lint` → PASS.

- [ ] **Step 9: Full web suite and commit**

Run: `pnpm --filter @zenith/web test && pnpm --filter @zenith/web exec tsc --noEmit` → PASS.

```bash
git add apps/web .github/workflows/ci.yml
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(web): self-host login page and docs; hide plan content in standalone; typecheck in ci

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 3: CLI — `zen login` with email/password, token refresh, `zen logout`, help text, version regex

**Lane:** CLI. **Files:**
- Modify: `cli/internal/config/config.go:12-30`
- Modify: `cli/internal/api/client.go:15-30,32-56,84-99,354-365`
- Create: `cli/internal/api/auth.go`, `cli/internal/api/auth_test.go`
- Modify: `cli/cmd/login/login.go` (whole file); Create: `cli/cmd/login/login_test.go`
- Create: `cli/cmd/logout/logout.go`
- Modify: `cli/cmd/root/root.go:24-31` and `init()` (add logout)
- Modify: `cli/cmd/upgrade/upgrade.go:24`, `cli/internal/composeops/upgrade.go:37`; Create: `cli/cmd/upgrade/version_test.go`; extend `cli/internal/composeops/upgrade_test.go`
- Modify every `cliapi.NewClient(cfg.APIEndpoint, cfg.Token)` call site (find them: `grep -rn 'NewClient(cfg.APIEndpoint' cli/cmd`) to `cliapi.NewClientFromConfig(cfg)`.

**Interfaces:**
- Produces: `config.Config{RefreshToken string; TokenExpiresAt time.Time}`; `api.TokenSet{AccessToken, RefreshToken string; ExpiresAt time.Time}`; `api.ErrMFARequired` (error type with `MFAToken`); `(*Client).LoginWithPassword(email, password) (TokenSet, error)`; `(*Client).LoginMFA(mfaToken, code) (TokenSet, error)`; `(*Client).Refresh() (TokenSet, error)`; `(*Client).Logout() error`; `(*Client).SetSession(ts TokenSet)`; `Client.OnTokens func(TokenSet)`; `api.NewClientFromConfig(cfg *config.Config) *Client`; `api.ErrSessionExpired`.

- [ ] **Step 1: Config fields**

`cli/internal/config/config.go` (keep the yaml import path the file already uses; add `"time"`):

```go
type Config struct {
	// APIEndpoint is the Zenith API server URL
	APIEndpoint string `yaml:"api_endpoint"`

	// Token is the access token (or a deploy token "znt_id_…:znt_sk_…")
	Token string `yaml:"token,omitempty"`

	// RefreshToken renews Token when it expires; empty for pasted tokens.
	RefreshToken string `yaml:"refresh_token,omitempty"`

	// TokenExpiresAt is when Token stops working; zero when unknown.
	TokenExpiresAt time.Time `yaml:"token_expires_at,omitempty"`

	// Project is the current active project
	Project string `yaml:"project,omitempty"`

	// Region is the default region for cloud installs (unused on self-host)
	Region string `yaml:"region,omitempty"`

	// OutputFormat is the default output format (table, json, yaml)
	OutputFormat string `yaml:"output_format,omitempty"`
}
```

Run: `cd cli && go build ./...` → ok.

- [ ] **Step 2: Write the failing auth client tests**

```go
// cli/internal/api/auth_test.go
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func tokenJSON(access, refresh string, expiresIn int) string {
	b, _ := json.Marshal(map[string]any{"access_token": access, "refresh_token": refresh, "token_type": "Bearer", "expires_in": expiresIn})
	return string(b)
}

func TestLoginWithPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" || r.Method != "POST" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["email"] != "a@b.c" || body["password"] != "pw" {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(tokenJSON("acc", "ref", 3600)))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "")
	before := time.Now()
	ts, err := c.LoginWithPassword("a@b.c", "pw")
	if err != nil {
		t.Fatal(err)
	}
	if ts.AccessToken != "acc" || ts.RefreshToken != "ref" || ts.ExpiresAt.Before(before.Add(59*time.Minute)) {
		t.Fatalf("ts = %+v", ts)
	}
	if c.Token != "acc" {
		t.Fatal("login must set the client's token")
	}
}

func TestLoginWithPassword_MFARequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			w.Write([]byte(`{"mfa_required":true,"mfa_token":"mfa-1"}`))
		case "/api/v1/auth/login/mfa":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["mfa_token"] != "mfa-1" || body["code"] != "123456" {
				t.Errorf("body = %v", body)
			}
			w.Write([]byte(tokenJSON("acc2", "ref2", 3600)))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "")
	_, err := c.LoginWithPassword("a@b.c", "pw")
	var mfa *ErrMFARequired
	if !errors.As(err, &mfa) || mfa.MFAToken != "mfa-1" {
		t.Fatalf("err = %v", err)
	}
	ts, err := c.LoginMFA(mfa.MFAToken, "123456")
	if err != nil || ts.AccessToken != "acc2" {
		t.Fatalf("ts = %+v err = %v", ts, err)
	}
}

func TestClient_RefreshesBeforeExpiry(t *testing.T) {
	var refreshes int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			atomic.AddInt32(&refreshes, 1)
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["refresh_token"] != "ref" {
				t.Errorf("body = %v", body)
			}
			w.Write([]byte(tokenJSON("fresh", "ref2", 3600)))
		case "/api/v1/projects":
			if r.Header.Get("Authorization") != "Bearer fresh" {
				t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
			}
			w.Write([]byte(`{"items":[]}`))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "stale")
	var saved TokenSet
	c.OnTokens = func(ts TokenSet) { saved = ts }
	c.SetSession(TokenSet{AccessToken: "stale", RefreshToken: "ref", ExpiresAt: time.Now().Add(30 * time.Second)})
	if _, err := c.ListProjects(); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&refreshes) != 1 || saved.AccessToken != "fresh" || saved.RefreshToken != "ref2" {
		t.Fatalf("refreshes = %d saved = %+v", refreshes, saved)
	}
}

func TestClient_RefreshesAndRetriesOn401(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			w.Write([]byte(tokenJSON("fresh", "ref", 3600)))
		case "/api/v1/projects":
			if atomic.AddInt32(&calls, 1) == 1 {
				w.WriteHeader(401)
				return
			}
			w.Write([]byte(`{"items":[]}`))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "")
	c.SetSession(TokenSet{AccessToken: "old", RefreshToken: "ref", ExpiresAt: time.Now().Add(time.Hour)})
	if _, err := c.ListProjects(); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestClient_RefreshFailureIsReportedOnce(t *testing.T) {
	var refreshes, calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			atomic.AddInt32(&refreshes, 1)
			w.WriteHeader(401)
		case "/api/v1/projects":
			atomic.AddInt32(&calls, 1)
			w.WriteHeader(401)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "")
	c.SetSession(TokenSet{AccessToken: "old", RefreshToken: "ref", ExpiresAt: time.Now().Add(time.Hour)})
	_, err := c.ListProjects()
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("err = %v, want ErrSessionExpired", err)
	}
	if refreshes != 1 || calls != 1 {
		t.Fatalf("refreshes = %d calls = %d (must not loop)", refreshes, calls)
	}
}

func TestClient_NoRefreshWithoutRefreshToken(t *testing.T) {
	var refreshes int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/refresh" {
			atomic.AddInt32(&refreshes, 1)
		}
		w.WriteHeader(401)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "pasted-token") // old config: token only
	if _, err := c.ListProjects(); err == nil {
		t.Fatal("expected an error")
	}
	if refreshes != 0 {
		t.Fatal("must not call refresh without a refresh token")
	}
}

func TestLogout(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = r.URL.Path == "/api/v1/auth/logout" && r.Header.Get("Authorization") == "Bearer acc"
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "acc")
	if err := c.Logout(); err != nil || !hit {
		t.Fatalf("err = %v hit = %v", err, hit)
	}
}
```

Run: `go test ./internal/api/ -run 'TestLogin|TestClient_|TestLogout' -count=1` → FAIL to compile.

- [ ] **Step 3: Implement sessions in the client**

`cli/internal/api/client.go` — change the struct and `do`:

```go
// Client is the Zenith API client used by the CLI.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client

	// Session state for email/password logins. RefreshToken empty = no refresh.
	RefreshToken string
	ExpiresAt    time.Time
	// OnTokens is called after every successful login or refresh so the caller can persist the session.
	OnTokens func(TokenSet)
}
```

```go
func (c *Client) do(method, path string, body interface{}, result interface{}) error {
	if path != "/api/v1/auth/refresh" {
		c.refreshIfExpiring()
	}
	status, err := c.doOnce(method, path, body, result)
	if status == http.StatusUnauthorized && c.RefreshToken != "" && path != "/api/v1/auth/refresh" {
		if _, rerr := c.Refresh(); rerr != nil {
			return ErrSessionExpired
		}
		_, err = c.doOnce(method, path, body, result)
	}
	return err
}

// doOnce performs one request and returns the HTTP status with the error.
func (c *Client) doOnce(method, path string, body interface{}, result interface{}) (int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// setAuth adds the Authorization header. Deploy tokens are stored as
// "znt_id_xxx:znt_sk_xxx" and use the DeployToken scheme; JWTs use Bearer.
func (c *Client) setAuth(req *http.Request) {
	if c.Token == "" {
		return
	}
	if strings.HasPrefix(c.Token, "znt_id_") {
		req.Header.Set("Authorization", "DeployToken "+c.Token)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}
```

Replace the duplicated header block in `StreamAppLogs` with `c.setAuth(req)` (and call `c.refreshIfExpiring()` before building that request). Delete the old `Login`, `loginRequest` and `loginTokenResponse` from `client.go` (they move to `auth.go`); update `client_test.go` if it referenced `Login` (search `c.Login(`; replace with `LoginWithPassword` and `.AccessToken`).

```go
// cli/internal/api/auth.go
package api

import (
	"errors"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/cli/internal/config"
)

// TokenSet is what a login or refresh returns.
type TokenSet struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// ErrSessionExpired is returned when the access token is rejected and the refresh token no longer works.
var ErrSessionExpired = errors.New("session expired — run zen login")

// ErrMFARequired is returned by LoginWithPassword when the account has MFA; call LoginMFA with the code.
type ErrMFARequired struct{ MFAToken string }

func (e *ErrMFARequired) Error() string { return "multi-factor code required" }

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type mfaRequest struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	MFARequired  bool   `json:"mfa_required"`
	MFAToken     string `json:"mfa_token"`
}

func (r tokenResponse) set() TokenSet {
	return TokenSet{AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: time.Now().Add(time.Duration(r.ExpiresIn) * time.Second)}
}

// SetSession installs tokens from a saved config.
func (c *Client) SetSession(ts TokenSet) {
	c.Token, c.RefreshToken, c.ExpiresAt = ts.AccessToken, ts.RefreshToken, ts.ExpiresAt
}

func (c *Client) adopt(r tokenResponse) TokenSet {
	ts := r.set()
	c.SetSession(ts)
	if c.OnTokens != nil {
		c.OnTokens(ts)
	}
	return ts
}

// LoginWithPassword logs in with email and password. It returns *ErrMFARequired when a code is needed.
func (c *Client) LoginWithPassword(email, password string) (TokenSet, error) {
	var resp tokenResponse
	if _, err := c.doOnce("POST", "/api/v1/auth/login", loginRequest{email, password}, &resp); err != nil {
		return TokenSet{}, err
	}
	if resp.MFARequired {
		return TokenSet{}, &ErrMFARequired{MFAToken: resp.MFAToken}
	}
	if resp.AccessToken == "" {
		return TokenSet{}, fmt.Errorf("login returned no access token")
	}
	return c.adopt(resp), nil
}

// LoginMFA completes a login that required a code.
func (c *Client) LoginMFA(mfaToken, code string) (TokenSet, error) {
	var resp tokenResponse
	if _, err := c.doOnce("POST", "/api/v1/auth/login/mfa", mfaRequest{mfaToken, code}, &resp); err != nil {
		return TokenSet{}, err
	}
	return c.adopt(resp), nil
}

// Refresh exchanges the refresh token for new tokens.
func (c *Client) Refresh() (TokenSet, error) {
	if c.RefreshToken == "" {
		return TokenSet{}, ErrSessionExpired
	}
	var resp tokenResponse
	if _, err := c.doOnce("POST", "/api/v1/auth/refresh", refreshRequest{c.RefreshToken}, &resp); err != nil {
		return TokenSet{}, err
	}
	if resp.RefreshToken == "" {
		resp.RefreshToken = c.RefreshToken // server did not rotate
	}
	return c.adopt(resp), nil
}

// refreshIfExpiring renews the session when the access token expires within a minute. Errors are left to the request.
func (c *Client) refreshIfExpiring() {
	if c.RefreshToken == "" || c.ExpiresAt.IsZero() || time.Until(c.ExpiresAt) > time.Minute {
		return
	}
	_, _ = c.Refresh()
}

// Logout invalidates the access token server-side (best effort).
func (c *Client) Logout() error {
	_, err := c.doOnce("POST", "/api/v1/auth/logout", nil, nil)
	return err
}

// NewClientFromConfig builds a client that keeps the saved session fresh.
func NewClientFromConfig(cfg *config.Config) *Client {
	c := NewClient(cfg.APIEndpoint, cfg.Token)
	c.SetSession(TokenSet{AccessToken: cfg.Token, RefreshToken: cfg.RefreshToken, ExpiresAt: cfg.TokenExpiresAt})
	c.OnTokens = func(ts TokenSet) {
		cfg.Token, cfg.RefreshToken, cfg.TokenExpiresAt = ts.AccessToken, ts.RefreshToken, ts.ExpiresAt
		_ = cfg.Save()
	}
	return c
}
```

(`config` does not import `api`, so there is no cycle.) Then replace every `cliapi.NewClient(cfg.APIEndpoint, cfg.Token)` in `cli/cmd/**` with `cliapi.NewClientFromConfig(cfg)`; keep other `NewClient` uses (tests, token flags) as they are.

Run: `go build ./... && go test ./internal/api/ -race -count=1` → PASS.

- [ ] **Step 4: Write the failing login-command test**

```go
// cli/cmd/login/login_test.go
package login

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/dotechhq/zenith/cli/internal/config"
)

func TestLoginWithPasswordSavesSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			w.Write([]byte(`{"access_token":"acc","refresh_token":"ref","token_type":"Bearer","expires_in":3600}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "config.yaml")
	var out bytes.Buffer
	err := runLoginTo(&out, loginInputs{endpoint: srv.URL, email: "a@b.c", password: "pw", configPath: path,
		promptMFA: func() (string, error) { t.Fatal("no MFA expected"); return "", nil }})
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.LoadFrom(path)
	if cfg.APIEndpoint != srv.URL || cfg.Token != "acc" || cfg.RefreshToken != "ref" || cfg.TokenExpiresAt.IsZero() {
		t.Fatalf("cfg = %+v", cfg)
	}
	if !bytes.Contains(out.Bytes(), []byte("Logged in to "+srv.URL+" as a@b.c")) {
		t.Fatalf("out = %s", out.String())
	}
}

func TestLoginMFAPrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			w.Write([]byte(`{"mfa_required":true,"mfa_token":"m1"}`))
		case "/api/v1/auth/login/mfa":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["code"] != "000111" {
				t.Errorf("code = %q", body["code"])
			}
			w.Write([]byte(`{"access_token":"acc","refresh_token":"ref","token_type":"Bearer","expires_in":3600}`))
		}
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := runLoginTo(&bytes.Buffer{}, loginInputs{endpoint: srv.URL, email: "a@b.c", password: "pw", configPath: path,
		promptMFA: func() (string, error) { return "000111", nil }})
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.LoadFrom(path)
	if cfg.Token != "acc" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"invalid credentials"}`))
	}))
	defer srv.Close()
	err := runLoginTo(&bytes.Buffer{}, loginInputs{endpoint: srv.URL, email: "a@b.c", password: "bad", configPath: filepath.Join(t.TempDir(), "c.yaml")})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("invalid credentials")) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoginTokenOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := runLoginTo(&bytes.Buffer{}, loginInputs{endpoint: "https://z.example.com", token: "znt_id_a:znt_sk_b", configPath: path}); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.LoadFrom(path)
	if cfg.Token != "znt_id_a:znt_sk_b" || cfg.RefreshToken != "" {
		t.Fatalf("cfg = %+v", cfg)
	}
}
```

Run: `go test ./cmd/login/ -count=1` → FAIL to compile.

- [ ] **Step 5: Rewrite the login command**

```go
// cli/cmd/login/login.go
package login

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	cliapi "github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Cmd is the login command.
var Cmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to a Zenith server",
	Long: `Log in to your Zenith server with the email and password you use on the dashboard.

The session is stored in ~/.zen/config.yaml and renewed automatically.
A pasted token (--token) is also accepted, for example a deploy token for CI.`,
	RunE: runLogin,
}

var (
	flagEndpoint string
	flagToken    string
	flagEmail    string
	flagPassword string
)

func init() {
	Cmd.Flags().StringVar(&flagEndpoint, "endpoint", "", "Zenith server URL, e.g. https://zenith.example.com")
	Cmd.Flags().StringVar(&flagEmail, "email", "", "Account email")
	Cmd.Flags().StringVar(&flagPassword, "password", "", "Account password (or set ZEN_PASSWORD)")
	Cmd.Flags().StringVar(&flagToken, "token", "", "Use a pasted token instead of email and password")
}

// loginInputs is everything runLoginTo needs; the command fills it from flags and prompts.
type loginInputs struct {
	endpoint   string
	email      string
	password   string
	token      string
	configPath string
	promptMFA  func() (string, error)
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	reader := bufio.NewReader(os.Stdin)
	in := loginInputs{endpoint: flagEndpoint, email: flagEmail, password: flagPassword, token: flagToken, configPath: config.DefaultConfigPath()}
	if in.password == "" {
		in.password = os.Getenv("ZEN_PASSWORD")
	}
	if in.endpoint == "" {
		in.endpoint = prompt(reader, fmt.Sprintf("Server URL [%s]", cfg.APIEndpoint), cfg.APIEndpoint)
	}
	if in.token == "" {
		if in.email == "" {
			in.email = prompt(reader, "Email", "")
		}
		if in.password == "" {
			in.password = promptSecret(reader, "Password")
		}
	}
	in.promptMFA = func() (string, error) { return prompt(reader, "Code", ""), nil }
	return runLoginTo(cmd.OutOrStdout(), in)
}

// runLoginTo performs the login and saves the session. It is the testable core of the command.
func runLoginTo(w io.Writer, in loginInputs) error {
	in.endpoint = strings.TrimRight(strings.TrimSpace(in.endpoint), "/")
	if in.endpoint == "" {
		return fmt.Errorf("server URL is required (--endpoint https://zenith.example.com)")
	}
	cfg, err := config.LoadFrom(in.configPath)
	if err != nil {
		cfg = &config.Config{}
	}
	cfg.APIEndpoint = in.endpoint

	if in.token != "" {
		cfg.Token, cfg.RefreshToken, cfg.TokenExpiresAt = in.token, "", time.Time{}
		if err := cfg.SaveTo(in.configPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Fprintf(w, "%s Logged in to %s with a token\n", tui.Green("✓"), in.endpoint)
		return nil
	}
	if in.email == "" || in.password == "" {
		return fmt.Errorf("email and password are required")
	}

	client := cliapi.NewClient(in.endpoint, "")
	ts, err := client.LoginWithPassword(in.email, in.password)
	var mfa *cliapi.ErrMFARequired
	if errors.As(err, &mfa) {
		if in.promptMFA == nil {
			return fmt.Errorf("this account needs a multi-factor code")
		}
		code, perr := in.promptMFA()
		if perr != nil {
			return perr
		}
		ts, err = client.LoginMFA(mfa.MFAToken, strings.TrimSpace(code))
	}
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	cfg.Token, cfg.RefreshToken, cfg.TokenExpiresAt = ts.AccessToken, ts.RefreshToken, ts.ExpiresAt
	if err := cfg.SaveTo(in.configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(w, "%s Logged in to %s as %s\n", tui.Green("✓"), in.endpoint, in.email)
	fmt.Fprintf(w, "  Config saved to: %s\n", in.configPath)
	return nil
}

func prompt(r *bufio.Reader, label, def string) string {
	fmt.Printf("%s %s: ", tui.Cyan("?"), label)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// promptSecret reads a password without echo when stdin is a terminal.
func promptSecret(r *bufio.Reader, label string) string {
	fmt.Printf("%s %s: ", tui.Cyan("?"), label)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
```

Add the dependency: `cd cli && go get golang.org/x/term@latest && go mod tidy`. Run: `go test ./cmd/login/ -race -count=1` → PASS.

- [ ] **Step 6: `zen logout`**

```go
// cli/cmd/logout/logout.go
package logout

import (
	"fmt"
	"time"

	cliapi "github.com/dotechhq/zenith/cli/internal/api"
	"github.com/dotechhq/zenith/cli/internal/config"
	"github.com/dotechhq/zenith/cli/internal/tui"
	"github.com/spf13/cobra"
)

// Cmd is the logout command.
var Cmd = &cobra.Command{
	Use:   "logout",
	Short: "Forget the saved session",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.Token != "" && cfg.RefreshToken != "" {
			_ = cliapi.NewClientFromConfig(cfg).Logout() // best effort
		}
		cfg.Token, cfg.RefreshToken, cfg.TokenExpiresAt = "", "", time.Time{}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s Logged out of %s\n", tui.Green("✓"), cfg.APIEndpoint)
		return nil
	},
}
```

In `cli/cmd/root/root.go` import `"github.com/dotechhq/zenith/cli/cmd/logout"` and add `rootCmd.AddCommand(logout.Cmd)` after `login.Cmd`. Replace the root help:

```go
var rootCmd = &cobra.Command{
	Use:   "zen",
	Short: "Zenith - open-source self-hosted PaaS",
	Long: `Zenith is a free, open-source PaaS you run on your own server.

Install on a fresh Ubuntu box:
  zen install --edition compose --domain zenith.example.com

Then log in, deploy and operate it:
  zen login --endpoint https://zenith.example.com
  zen deploy --image traefik/whoami:latest
  zen status · zen logs · zen db connect · zen backup · zen upgrade`,
	SilenceUsage:  true,
	SilenceErrors: true,
}
```

Run: `go build ./... && go run . --help | grep -c Hetzner` → `0`.

- [ ] **Step 7: Version regex accepts `-` inside the prerelease**

```go
// cli/cmd/upgrade/version_test.go
package upgrade

import "testing"

func TestValidateVersion(t *testing.T) {
	for _, ok := range []string{"1.5.0", "0.1.0-alpha.3", "0.1.0-alpha.3-test", "0.1.0-rc-1.2"} {
		if err := validateVersion(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"v1.5.0", "1.5", "1.5.0-", "1.5.0 ", "latest"} {
		if err := validateVersion(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
```

Change both regexes to:

```go
var versionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]*[0-9A-Za-z])?$`)
```

```go
var targetRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]*[0-9A-Za-z])?$`)
```

Add to `cli/internal/composeops/upgrade_test.go`:

```go
func TestTargetReAcceptsDashInPrerelease(t *testing.T) {
	for _, ok := range []string{"1.5.0", "0.1.0-alpha.3", "0.1.0-alpha.3-test", "0.1.0-rc-1.2"} {
		if !targetRe.MatchString(ok) {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"v1.5.0", "1.5", "1.5.0-", "1.5.0 ", "latest"} {
		if targetRe.MatchString(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}
```

Run: `go test ./cmd/upgrade/ ./internal/composeops/ -run 'Version|TargetRe' -count=1` → PASS.

- [ ] **Step 8: Whole CLI suite and commit**

Run: `cd cli && gofmt -l . && go vet ./... && go test ./... -race -count=1` → clean, PASS.

```bash
git add cli
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(cli): zen login with email and password, automatic token refresh, zen logout

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 4: CLI — `zen status` for standalone

**Lane:** CLI, after Task 3. **Files:**
- Modify: `cli/internal/api/client.go:150-162` (`ListApps`), add `ServerInfo`
- Modify: `cli/cmd/status/status.go` (whole `runStatus`); extend `cli/cmd/status/status_test.go`

**Interfaces:**
- Consumes: `GET /api/v1/version` `{"version","mode"}`; `GET /api/v1/apps?project_id=`.
- Produces: `(*Client).ServerInfo() (ServerInfo, error)` with `ServerInfo{Version, Mode string}`; `renderStatus(w io.Writer, client *cliapi.Client, cfg *config.Config) error`.

- [ ] **Step 1: Write the failing tests**

Append to `cli/cmd/status/status_test.go` (add the imports `bytes`, `fmt`, `net/http`, `net/http/httptest`, `cliapi "github.com/dotechhq/zenith/cli/internal/api"`, `"github.com/dotechhq/zenith/cli/internal/config"`):

```go
func statusServer(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/version":
			fmt.Fprintf(w, `{"version":"0.1.0-alpha.4","build_time":"x","go_version":"go","mode":%q}`, mode)
		case "/api/v1/projects":
			w.Write([]byte(`{"items":[{"id":"p1","name":"default","display_name":"Default","plan":"free","region":""}]}`))
		case "/api/v1/apps":
			if r.URL.Query().Get("project_id") != "p1" {
				t.Errorf("apps must be filtered by project_id, got %q", r.URL.RawQuery)
			}
			w.Write([]byte(`{"items":[{"id":"a1","name":"whoami","status":"running","url":"https://whoami.example.com"}]}`))
		case "/api/v1/databases":
			w.Write([]byte(`[{"id":"d1","name":"pg1","engine":"postgresql","status":"ready"}]`))
		default:
			w.WriteHeader(404)
		}
	}))
}

func TestRenderStatusStandalone(t *testing.T) {
	srv := statusServer(t, "standalone")
	defer srv.Close()
	cfg := &config.Config{APIEndpoint: srv.URL, Token: "t"}
	var out bytes.Buffer
	if err := renderStatus(&out, cliapi.NewClientFromConfig(cfg), cfg); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"0.1.0-alpha.4", srv.URL, "whoami", "https://whoami.example.com", "pg1", "postgresql"} {
		if !strings.Contains(s, want) {
			t.Errorf("output lacks %q:\n%s", want, s)
		}
	}
	for _, no := range []string{"Plan", "Cost", "Region"} {
		if strings.Contains(s, no) {
			t.Errorf("standalone output must not contain %q:\n%s", no, s)
		}
	}
}

func TestRenderStatusSaaSKeepsPlan(t *testing.T) {
	srv := statusServer(t, "saas")
	defer srv.Close()
	cfg := &config.Config{APIEndpoint: srv.URL, Token: "t"}
	var out bytes.Buffer
	if err := renderStatus(&out, cliapi.NewClientFromConfig(cfg), cfg); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Plan") {
		t.Fatalf("saas output must keep the plan line:\n%s", out.String())
	}
}
```

Check how `ListDatabases` decodes `/api/v1/databases` (`cli/internal/api/client.go:255-265`) and match the fixture shape to it.

Run: `go test ./cmd/status/ -run TestRenderStatus -count=1` → FAIL to compile.

- [ ] **Step 2: Client: `ServerInfo` and the v2 apps route**

```go
// ServerInfo is GET /api/v1/version.
type ServerInfo struct {
	Version string `json:"version"`
	Mode    string `json:"mode"`
}

func (c *Client) ServerInfo() (ServerInfo, error) {
	var info ServerInfo
	err := c.do("GET", "/api/v1/version", nil, &info)
	return info, err
}

func (c *Client) ListApps(projectID string) ([]App, error) {
	path := "/api/v1/apps"
	if projectID != "" {
		path += "?project_id=" + url.QueryEscape(projectID)
	}
	var resp struct {
		Items []App `json:"items"`
	}
	if err := c.do("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}
```

Fix any `client_test.go` test that asserted the old `/projects/:id/apps` path to the new query form.

- [ ] **Step 3: Rewrite `runStatus` around `renderStatus`**

```go
func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil || cfg.Token == "" {
		fmt.Fprintln(cmd.OutOrStdout(), lipgloss.NewStyle().Foreground(tui.ColorError).Render("  Not logged in — run: zen login"))
		return nil
	}
	return renderStatus(cmd.OutOrStdout(), cliapi.NewClientFromConfig(cfg), cfg)
}

// renderStatus prints the overview. On a self-hosted server there is no plan, region or cost.
func renderStatus(w io.Writer, client *cliapi.Client, cfg *config.Config) error {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).MarginBottom(1)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorPrimary).Padding(0, 1)
	cellStyle := lipgloss.NewStyle().Foreground(tui.ColorText).Padding(0, 1)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorText).MarginTop(1)
	infoStyle := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	valStyle := lipgloss.NewStyle().Foreground(tui.ColorText)
	errStyle := lipgloss.NewStyle().Foreground(tui.ColorError)

	info, err := client.ServerInfo()
	if err != nil {
		fmt.Fprintln(w, errStyle.Render("  Failed to reach the server: "+err.Error()))
		return nil
	}
	standalone := info.Mode != "saas"

	fmt.Fprintln(w, titleStyle.Render("  Zenith Status"))
	fmt.Fprintf(w, "  %s %s (%s)\n", infoStyle.Render("Server: "), valStyle.Render(cfg.APIEndpoint), info.Version)

	projects, err := client.ListProjects()
	if err != nil {
		fmt.Fprintln(w, errStyle.Render("  Failed to fetch projects: "+err.Error()))
		return nil
	}
	if len(projects) == 0 {
		fmt.Fprintln(w, infoStyle.Render("  No projects found. Create one at the dashboard."))
		return nil
	}
	current := projects[0]
	for _, p := range projects {
		if cfg.Project != "" && (p.ID == cfg.Project || p.Name == cfg.Project || p.DisplayName == cfg.Project) {
			current = p
			break
		}
	}
	displayName := current.DisplayName
	if displayName == "" {
		displayName = current.Name
	}
	fmt.Fprintf(w, "  %s %s\n", infoStyle.Render("Project:"), valStyle.Render(displayName))
	if !standalone {
		region := current.Region
		if region == "" {
			region = cfg.Region
		}
		plan := current.Plan
		if plan == "" {
			plan = "free"
		}
		fmt.Fprintf(w, "  %s %s\n", infoStyle.Render("Region: "), valStyle.Render(region))
		fmt.Fprintf(w, "  %s %s\n", infoStyle.Render("Plan:   "), valStyle.Render(plan))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, sectionStyle.Render("  Apps"))
	apps, err := client.ListApps(current.ID)
	if err != nil {
		fmt.Fprintln(w, "  "+errStyle.Render("Failed to fetch apps: "+err.Error()))
	} else {
		rows := make([][]string, 0, len(apps))
		for _, a := range apps {
			rows = append(rows, []string{a.Name, a.Status, a.URL})
		}
		fmt.Fprint(w, renderTable([]string{"NAME", "STATUS", "URL"}, rows, headerStyle, cellStyle))
	}

	fmt.Fprintln(w, sectionStyle.Render("  Databases"))
	dbs, err := client.ListDatabases(current.ID)
	if err != nil {
		fmt.Fprintln(w, "  "+errStyle.Render("Failed to fetch databases: "+err.Error()))
	} else {
		rows := make([][]string, 0, len(dbs))
		for _, d := range dbs {
			rows = append(rows, []string{d.Name, d.Engine, d.Status})
		}
		fmt.Fprint(w, renderTable([]string{"NAME", "ENGINE", "STATUS"}, rows, headerStyle, cellStyle))
	}

	if !standalone {
		fmt.Fprintln(w, sectionStyle.Render("  Cost Estimate"))
		fmt.Fprintf(w, "  %s %s\n\n", infoStyle.Render("Monthly:"), lipgloss.NewStyle().Bold(true).Foreground(tui.ColorText).Render("€0.00"))
	}
	return nil
}
```

Update the command's `Long` text to `Show the server version, the current project's apps and databases.` and add `"io"` to the imports. Run: `go test ./cmd/status/ ./internal/api/ -race -count=1` → PASS.

- [ ] **Step 4: Commit**

```bash
gofmt -l . ; go vet ./... && go test ./... -race -count=1
git add cli
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "fix(cli): zen status reads the v2 apps route and drops plan and cost on self-host

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 5: Domain model — `status_message`, `last_checked_at`, repository method, migration

**Lane:** API, after Task 1. **Files:**
- Modify: `services/api/internal/entities/domain.go:14-21`
- Modify: `services/api/internal/dto/inputs.go:341-348`
- Modify: `services/api/internal/ports/repositories.go:224-231`
- Modify: `services/api/internal/adapters/memory/memory_domain.go` (AddDomain: drop the simulation goroutines, insert `active`; add `UpdateDomainCheck`, `ListAllDomains`)
- Modify: `services/api/internal/adapters/postgres/postgres_domain.go` (scan the two new columns everywhere rows are read; add `UpdateDomainCheck`, `ListAllDomains`)
- Create: `services/api/internal/adapters/postgres/migrations/052_domain_checks.up.sql`, `…down.sql`
- Modify: `services/api/internal/handlers/domain.go:165-174` (`toDomainInfo`)
- Create: `services/api/internal/adapters/memory/memory_domain_test.go`

**Interfaces:**
- Produces: `entities.CustomDomain{StatusMessage string; LastCheckedAt *time.Time}`; `ports.DomainRepository.UpdateDomainCheck(ctx, id string, status entities.DomainStatus, tlsReady bool, message string, checkedAt time.Time) error`; `ports.DomainRepository.ListAllDomains(ctx) ([]entities.CustomDomain, error)`; `dto.DomainInfo{StatusMessage string (json status_message); LastCheckedAt *string (json last_checked_at)}`.

- [ ] **Step 1: Failing memory-adapter test**

```go
// services/api/internal/adapters/memory/memory_domain_test.go
package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestDomainCheckRoundTrip(t *testing.T) {
	r := NewMemoryDomainRepository()
	ctx := context.Background()
	d, err := r.AddDomain(ctx, "app-1", "user-1", "a.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != entities.DomainStatusActive || d.TLSReady {
		t.Fatalf("AddDomain default = %s/%v, want active/false (the handler decides pending)", d.Status, d.TLSReady)
	}
	at := time.Date(2026, 10, 16, 19, 0, 0, 0, time.UTC)
	if err := r.UpdateDomainCheck(ctx, d.ID, entities.DomainStatusPending, false, "DNS: no A/AAAA record for a.example.com; create an A record to 1.2.3.4", at); err != nil {
		t.Fatal(err)
	}
	got, _ := r.GetDomain(ctx, d.ID)
	if got.Status != entities.DomainStatusPending || got.StatusMessage == "" || got.LastCheckedAt == nil || !got.LastCheckedAt.Equal(at) {
		t.Fatalf("got %+v", got)
	}
	all, _ := r.ListAllDomains(ctx)
	if len(all) != 1 || all[0].ID != d.ID {
		t.Fatalf("ListAllDomains = %+v", all)
	}
	if err := r.UpdateDomainCheck(ctx, "missing", entities.DomainStatusActive, true, "", at); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestAddDomainDoesNotFlipStatusByItself(t *testing.T) {
	r := NewMemoryDomainRepository()
	d, _ := r.AddDomain(context.Background(), "app-1", "user-1", "b.example.com")
	time.Sleep(10 * time.Millisecond)
	got, _ := r.GetDomain(context.Background(), d.ID)
	if got.Status != entities.DomainStatusActive {
		t.Fatalf("status changed to %s without a check", got.Status)
	}
}
```

Run: `go test ./internal/adapters/memory/ -run 'TestDomainCheck|TestAddDomainDoes' -count=1` → FAIL.

- [ ] **Step 2: Entity, DTO, port**

```go
// services/api/internal/entities/domain.go (add "time" to the imports)
type CustomDomain struct {
	ID            string       `json:"id"`
	AppID         string       `json:"app_id"`
	UserID        string       `json:"user_id"`
	Domain        string       `json:"domain"`
	Status        DomainStatus `json:"status"`
	TLSReady      bool         `json:"tls_ready"`
	StatusMessage string       `json:"status_message"`
	LastCheckedAt *time.Time   `json:"last_checked_at,omitempty"`
	Timestamps
}
```

```go
// services/api/internal/dto/inputs.go
type DomainInfo struct {
	ID            string                `json:"id"`
	AppID         string                `json:"app_id"`
	Domain        string                `json:"domain"`
	Status        entities.DomainStatus `json:"status"`
	TLSReady      bool                  `json:"tls_ready"`
	StatusMessage string                `json:"status_message"`
	LastCheckedAt *string               `json:"last_checked_at"`
	CreatedAt     string                `json:"created_at"`
}
```

```go
// services/api/internal/ports/repositories.go — DomainRepository, after UpdateDomainStatus
	// UpdateDomainCheck records one checker outcome.
	UpdateDomainCheck(ctx context.Context, id string, status entities.DomainStatus, tlsReady bool, message string, checkedAt time.Time) error
	// ListAllDomains returns every custom domain (the checker's work list).
	ListAllDomains(ctx context.Context) ([]entities.CustomDomain, error)
```

`services/api/internal/handlers/domain.go` (add `"time"` to the imports):

```go
func toDomainInfo(d *entities.CustomDomain) dto.DomainInfo {
	var checked *string
	if d.LastCheckedAt != nil {
		s := d.LastCheckedAt.UTC().Format(time.RFC3339)
		checked = &s
	}
	return dto.DomainInfo{
		ID:            d.ID,
		AppID:         d.AppID,
		Domain:        d.Domain,
		Status:        d.Status,
		TLSReady:      d.TLSReady,
		StatusMessage: d.StatusMessage,
		LastCheckedAt: checked,
		CreatedAt:     d.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
```

- [ ] **Step 3: Memory adapter**

Replace `AddDomain`'s body after the duplicate check with:

```go
	now := time.Now()
	d := &entities.CustomDomain{
		ID:     uuid.New().String(),
		AppID:  appID,
		UserID: userID,
		Domain: domain,
		Status: entities.DomainStatusActive,
		Timestamps: entities.Timestamps{CreatedAt: now, UpdatedAt: now},
	}
	r.domains[d.ID] = d
	return d, nil
```

(delete the two simulation goroutines) and add:

```go
func (r *MemoryDomainRepository) UpdateDomainCheck(_ context.Context, id string, status entities.DomainStatus, tlsReady bool, message string, checkedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.domains[id]
	if !ok {
		return fmt.Errorf("domain not found: %s", id)
	}
	at := checkedAt
	d.Status, d.TLSReady, d.StatusMessage, d.LastCheckedAt = status, tlsReady, message, &at
	d.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryDomainRepository) ListAllDomains(_ context.Context) ([]entities.CustomDomain, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]entities.CustomDomain, 0, len(r.domains))
	for _, d := range r.domains {
		out = append(out, *d)
	}
	return out, nil
}
```

Run: `go test ./internal/adapters/memory/ -run 'TestDomainCheck|TestAddDomainDoes' -count=1` → PASS. `go build ./...` now fails on the postgres adapter; continue.

- [ ] **Step 4: Migration and postgres adapter**

```sql
-- services/api/internal/adapters/postgres/migrations/052_domain_checks.up.sql
ALTER TABLE custom_domains ADD COLUMN IF NOT EXISTS status_message TEXT NOT NULL DEFAULT '';
ALTER TABLE custom_domains ADD COLUMN IF NOT EXISTS last_checked_at TIMESTAMPTZ NULL;
```

```sql
-- services/api/internal/adapters/postgres/migrations/052_domain_checks.down.sql
ALTER TABLE custom_domains DROP COLUMN IF EXISTS last_checked_at;
ALTER TABLE custom_domains DROP COLUMN IF EXISTS status_message;
```

In `postgres_domain.go`, add `status_message, last_checked_at` to every `SELECT` column list and `Scan` target (`&d.StatusMessage, &d.LastCheckedAt`; a `*time.Time` scans NULL as nil), and add:

```go
func (r *PostgresDomainRepository) UpdateDomainCheck(ctx context.Context, id string, status entities.DomainStatus, tlsReady bool, message string, checkedAt time.Time) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE custom_domains SET status = $1, tls_ready = $2, status_message = $3, last_checked_at = $4, updated_at = $5 WHERE id = $6`,
		string(status), tlsReady, message, checkedAt, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("update domain check: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("domain not found: %s", id)
	}
	return nil
}

func (r *PostgresDomainRepository) ListAllDomains(ctx context.Context) ([]entities.CustomDomain, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+domainColumns+` FROM custom_domains ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	defer rows.Close()
	return scanDomains(rows)
}
```

If the file has no shared `domainColumns` constant and `scanDomains` helper, introduce them now and use them from the existing list functions (same column order everywhere). If the migrations embed uses an explicit list (check `migrations/embed.go`), register 052 there.

Run: `go build ./... && go vet ./... && go test ./internal/handlers/ ./internal/adapters/... -count=1` → PASS (the postgres tests skip without a database, as today).

- [ ] **Step 5: Commit**

```bash
gofmt -l internal cmd
git add services/api
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(api): custom domains carry a status message and last check time

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 6: Domain checker, pending on add, `verify` endpoint, wiring

**Lane:** API, after Task 5. **Files:**
- Create: `services/api/internal/services/domain_checker.go`, `services/api/internal/services/domain_checker_test.go`, `services/api/internal/services/domain_tlsprobe_test.go`
- Modify: `services/api/internal/handlers/domain.go` (fields, `SetChecker`, `Add`, new `Verify`); extend `services/api/internal/handlers/domain_test.go`
- Modify: `services/api/cmd/server/main.go` (route after `:1329`; wiring after `domainHandler` is built)

**Interfaces:**
- Produces: `services.DomainResolver`, `services.TLSProber`, `services.NewDomainChecker(repo ports.DomainRepository, resolver DomainResolver, prober TLSProber, dashboardDomain string) *DomainChecker`; `(*DomainChecker).Check(ctx, d *entities.CustomDomain) (*entities.CustomDomain, error)`; `(*DomainChecker).CheckAll(ctx)`; `(*DomainChecker).Run(ctx)`; `(*DomainChecker).MarkPending(ctx, id string) error`; `services.TraefikTLSProber{Roots *x509.CertPool}`; `(*DomainHandler).SetChecker(*services.DomainChecker)`; `(*DomainHandler).Verify` on `POST /api/v1/apps/:appId/domains/:domainId/verify`.

- [ ] **Step 1: Failing checker tests**

```go
// services/api/internal/services/domain_checker_test.go
package services

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type fakeResolver struct{ ips map[string][]string }

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := f.ips[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	out := make([]net.IPAddr, 0, len(ips))
	for _, s := range ips {
		out = append(out, net.IPAddr{IP: net.ParseIP(s)})
	}
	return out, nil
}

type fakeProber struct {
	valid bool
	err   error
	calls int
}

func (p *fakeProber) Probe(_ context.Context, addr, serverName string) (bool, error) {
	p.calls++
	return p.valid, p.err
}

func newCheckerForTest(resolver fakeResolver, prober *fakeProber) (*DomainChecker, *memory.MemoryDomainRepository, *entities.CustomDomain) {
	repo := memory.NewMemoryDomainRepository()
	d, _ := repo.AddDomain(context.Background(), "app-1", "user-1", "app.example.com")
	c := NewDomainChecker(repo, resolver, prober, "zenith.example.com")
	c.now = func() time.Time { return time.Date(2026, 10, 16, 19, 0, 0, 0, time.UTC) }
	_ = c.MarkPending(context.Background(), d.ID)
	d, _ = repo.GetDomain(context.Background(), d.ID)
	return c, repo, d
}

func TestDomainChecker_NoRecordStaysPendingWithIP(t *testing.T) {
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}}}, &fakeProber{})
	got, err := c.Check(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != entities.DomainStatusPending || got.TLSReady {
		t.Fatalf("got %s/%v", got.Status, got.TLSReady)
	}
	if got.StatusMessage != "DNS: no A/AAAA record for app.example.com; create an A record to 1.2.3.4" {
		t.Fatalf("message = %q", got.StatusMessage)
	}
	if got.LastCheckedAt == nil {
		t.Fatal("LastCheckedAt not set")
	}
}

func TestDomainChecker_WrongIPStaysPending(t *testing.T) {
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"9.9.9.9"}}}, &fakeProber{})
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusPending || got.StatusMessage != "DNS: app.example.com points to 9.9.9.9; expected 1.2.3.4" {
		t.Fatalf("got %s %q", got.Status, got.StatusMessage)
	}
}

func TestDomainChecker_DNSOkCertPendingIsVerified(t *testing.T) {
	p := &fakeProber{valid: false}
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}, p)
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusVerified || got.TLSReady || got.StatusMessage != "DNS ok; waiting for the certificate" {
		t.Fatalf("got %s/%v %q", got.Status, got.TLSReady, got.StatusMessage)
	}
	if p.calls != 1 {
		t.Fatalf("prober calls = %d", p.calls)
	}
}

func TestDomainChecker_ValidCertIsActive(t *testing.T) {
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4", "2001:db8::1"}, "app.example.com": {"1.2.3.4"}}}, &fakeProber{valid: true})
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusActive || !got.TLSReady || got.StatusMessage != "" {
		t.Fatalf("got %s/%v %q", got.Status, got.TLSReady, got.StatusMessage)
	}
}

func TestDomainChecker_TLSErrorKeepsVerifiedWithMessage(t *testing.T) {
	c, _, d := newCheckerForTest(fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}, &fakeProber{err: errors.New("dial tcp: connection refused")})
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusVerified || !strings.HasPrefix(got.StatusMessage, "TLS check failed: ") {
		t.Fatalf("got %s %q", got.Status, got.StatusMessage)
	}
}

func TestDomainChecker_ActiveGoesBackToPendingWhenDNSMoves(t *testing.T) {
	r := fakeResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}
	c, repo, d := newCheckerForTest(r, &fakeProber{valid: true})
	got, _ := c.Check(context.Background(), d)
	if got.Status != entities.DomainStatusActive {
		t.Fatal("precondition: active")
	}
	r.ips["app.example.com"] = []string{"5.5.5.5"}
	got, _ = c.Check(context.Background(), got)
	if got.Status != entities.DomainStatusPending || got.TLSReady {
		t.Fatalf("got %s/%v, want pending/false", got.Status, got.TLSReady)
	}
	stored, _ := repo.GetDomain(context.Background(), d.ID)
	if stored.Status != entities.DomainStatusPending {
		t.Fatal("repo not updated")
	}
}

func TestDomainChecker_DashboardUnresolvableSkipsRound(t *testing.T) {
	c, repo, d := newCheckerForTest(fakeResolver{ips: map[string][]string{}}, &fakeProber{valid: true})
	c.CheckAll(context.Background())
	stored, _ := repo.GetDomain(context.Background(), d.ID)
	if stored.Status != entities.DomainStatusPending || stored.StatusMessage != "waiting for DNS check" {
		t.Fatalf("round must be skipped when the dashboard domain does not resolve: %+v", stored)
	}
}

func TestDomainChecker_Due(t *testing.T) {
	c := NewDomainChecker(memory.NewMemoryDomainRepository(), fakeResolver{}, &fakeProber{}, "zenith.example.com")
	now := time.Date(2026, 10, 16, 19, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	mk := func(status entities.DomainStatus, tls bool, age, sinceCheck time.Duration) *entities.CustomDomain {
		d := &entities.CustomDomain{Status: status, TLSReady: tls}
		d.CreatedAt = now.Add(-age)
		if sinceCheck >= 0 {
			t := now.Add(-sinceCheck)
			d.LastCheckedAt = &t
		}
		return d
	}
	cases := []struct {
		name string
		d    *entities.CustomDomain
		want bool
	}{
		{"never checked", mk(entities.DomainStatusPending, false, time.Minute, -1), true},
		{"young, checked 20s ago", mk(entities.DomainStatusPending, false, time.Minute, 20*time.Second), false},
		{"young, checked 40s ago", mk(entities.DomainStatusPending, false, time.Minute, 40*time.Second), true},
		{"2h old, checked 2m ago", mk(entities.DomainStatusPending, false, 2*time.Hour, 2*time.Minute), false},
		{"2h old, checked 6m ago", mk(entities.DomainStatusPending, false, 2*time.Hour, 6*time.Minute), true},
		{"2d old, checked 20m ago", mk(entities.DomainStatusVerified, false, 48*time.Hour, 20*time.Minute), false},
		{"2d old, checked 40m ago", mk(entities.DomainStatusVerified, false, 48*time.Hour, 40*time.Minute), true},
		{"active, checked 5m ago", mk(entities.DomainStatusActive, true, time.Minute, 5*time.Minute), false},
		{"active, checked 11m ago", mk(entities.DomainStatusActive, true, time.Minute, 11*time.Minute), true},
	}
	for _, tc := range cases {
		if got := c.due(tc.d); got != tc.want {
			t.Errorf("%s: due = %v, want %v", tc.name, got, tc.want)
		}
	}
}
```

Run: `go test ./internal/services/ -run TestDomainChecker -count=1` → FAIL to compile.

- [ ] **Step 2: Implement the checker**

```go
// services/api/internal/services/domain_checker.go
package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// DomainResolver is the DNS lookup the checker uses; *net.Resolver satisfies it.
type DomainResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// TLSProber reports whether the certificate served at addr for serverName verifies for serverName.
// (false, nil) means the handshake worked but the chain does not verify (Traefik's default certificate).
type TLSProber interface {
	Probe(ctx context.Context, addr, serverName string) (valid bool, err error)
}

// DomainChecker verifies custom domains: DNS must point at this server, then Traefik must serve a valid certificate.
type DomainChecker struct {
	repo            ports.DomainRepository
	resolver        DomainResolver
	prober          TLSProber
	dashboardDomain string
	traefikAddr     string
	stepTimeout     time.Duration
	now             func() time.Time

	mu         sync.Mutex
	expected   []net.IP
	expectedAt time.Time
}

const (
	checkEvery        = 30 * time.Second
	checkEveryAfter1h = 5 * time.Minute
	checkEveryAfter1d = 30 * time.Minute
	recheckActive     = 10 * time.Minute
	expectedIPsTTL    = 10 * time.Minute
)

func NewDomainChecker(repo ports.DomainRepository, resolver DomainResolver, prober TLSProber, dashboardDomain string) *DomainChecker {
	return &DomainChecker{repo: repo, resolver: resolver, prober: prober, dashboardDomain: dashboardDomain,
		traefikAddr: "traefik:443", stepTimeout: 5 * time.Second, now: time.Now}
}

// MarkPending sets a freshly added domain to pending with the initial message.
func (c *DomainChecker) MarkPending(ctx context.Context, id string) error {
	return c.repo.UpdateDomainCheck(ctx, id, entities.DomainStatusPending, false, "waiting for DNS check", c.now())
}

// Run checks due domains every 30 s until ctx ends.
func (c *DomainChecker) Run(ctx context.Context) {
	t := time.NewTicker(checkEvery)
	defer t.Stop()
	for {
		c.CheckAll(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// CheckAll runs one round over every due domain. Errors are logged, never fatal.
func (c *DomainChecker) CheckAll(ctx context.Context) {
	if _, err := c.expectedIPs(ctx); err != nil {
		slog.Warn("domain check: dashboard domain does not resolve; skipping round", "domain", c.dashboardDomain, "error", err)
		return
	}
	domains, err := c.repo.ListAllDomains(ctx)
	if err != nil {
		slog.Error("domain check: list", "error", err)
		return
	}
	for i := range domains {
		d := &domains[i]
		if !c.due(d) {
			continue
		}
		if _, err := c.Check(ctx, d); err != nil {
			slog.Warn("domain check failed", "domain", d.Domain, "error", err)
		}
	}
}

// due applies the backoff: 30 s for the first hour, 5 min for the first day, then 30 min; active domains every 10 min.
func (c *DomainChecker) due(d *entities.CustomDomain) bool {
	now := c.now()
	if d.LastCheckedAt == nil {
		return true
	}
	since := now.Sub(*d.LastCheckedAt)
	if d.Status == entities.DomainStatusActive && d.TLSReady {
		return since >= recheckActive
	}
	age := now.Sub(d.CreatedAt)
	switch {
	case age >= 24*time.Hour:
		return since >= checkEveryAfter1d
	case age >= time.Hour:
		return since >= checkEveryAfter1h
	default:
		return since >= checkEvery
	}
}

// Check runs the DNS and TLS steps for one domain and stores the outcome.
func (c *DomainChecker) Check(ctx context.Context, d *entities.CustomDomain) (*entities.CustomDomain, error) {
	expected, err := c.expectedIPs(ctx)
	if err != nil {
		return d, fmt.Errorf("resolve dashboard domain %s: %w", c.dashboardDomain, err)
	}
	status, tlsReady, msg := c.evaluate(ctx, d.Domain, expected)
	now := c.now()
	if err := c.repo.UpdateDomainCheck(ctx, d.ID, status, tlsReady, msg, now); err != nil {
		return d, err
	}
	out := *d
	out.Status, out.TLSReady, out.StatusMessage, out.LastCheckedAt = status, tlsReady, msg, &now
	return &out, nil
}

func (c *DomainChecker) evaluate(ctx context.Context, domain string, expected []net.IP) (entities.DomainStatus, bool, string) {
	dctx, cancel := context.WithTimeout(ctx, c.stepTimeout)
	addrs, err := c.resolver.LookupIPAddr(dctx, domain)
	cancel()
	if err != nil || len(addrs) == 0 {
		return entities.DomainStatusPending, false,
			fmt.Sprintf("DNS: no A/AAAA record for %s; create an A record to %s", domain, expected[0])
	}
	if !intersects(addrs, expected) {
		return entities.DomainStatusPending, false,
			fmt.Sprintf("DNS: %s points to %s; expected %s", domain, joinIPs(addrs), expected[0])
	}
	tctx, cancel := context.WithTimeout(ctx, c.stepTimeout)
	valid, perr := c.prober.Probe(tctx, c.traefikAddr, domain)
	cancel()
	switch {
	case perr != nil:
		return entities.DomainStatusVerified, false, "TLS check failed: " + perr.Error()
	case !valid:
		return entities.DomainStatusVerified, false, "DNS ok; waiting for the certificate"
	default:
		return entities.DomainStatusActive, true, ""
	}
}

// expectedIPs resolves the dashboard domain (cached 10 min). Its addresses are this server's public addresses.
func (c *DomainChecker) expectedIPs(ctx context.Context) ([]net.IP, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.expected) > 0 && c.now().Sub(c.expectedAt) < expectedIPsTTL {
		return c.expected, nil
	}
	rctx, cancel := context.WithTimeout(ctx, c.stepTimeout)
	defer cancel()
	addrs, err := c.resolver.LookupIPAddr(rctx, c.dashboardDomain)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no address")
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		ips = append(ips, a.IP)
	}
	sort.SliceStable(ips, func(i, j int) bool { // IPv4 first so messages show the A record
		return ips[i].To4() != nil && ips[j].To4() == nil
	})
	c.expected, c.expectedAt = ips, c.now()
	return ips, nil
}

func intersects(addrs []net.IPAddr, expected []net.IP) bool {
	for _, a := range addrs {
		for _, e := range expected {
			if a.IP.Equal(e) {
				return true
			}
		}
	}
	return false
}

func joinIPs(addrs []net.IPAddr) string {
	s := make([]string, 0, len(addrs))
	for _, a := range addrs {
		s = append(s, a.IP.String())
	}
	return strings.Join(s, ", ")
}

// TraefikTLSProber dials Traefik with SNI and full chain verification for the domain.
// Roots nil means the system roots (Let's Encrypt). The dial itself makes Traefik request the certificate.
// A handshake that fails only because the chain does not verify (Traefik's default certificate
// while ACME is pending) is reported as (false, nil); any other failure is an error.
type TraefikTLSProber struct{ Roots *x509.CertPool }

func (p *TraefikTLSProber) Probe(ctx context.Context, addr, serverName string) (bool, error) {
	d := &tls.Dialer{Config: &tls.Config{ServerName: serverName, RootCAs: p.Roots, MinVersion: tls.VersionTLS12}}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err == nil {
		conn.Close()
		return true, nil
	}
	var certErr *tls.CertificateVerificationError
	var unknownCA x509.UnknownAuthorityError
	var hostErr x509.HostnameError
	var invalid x509.CertificateInvalidError
	if errors.As(err, &certErr) || errors.As(err, &unknownCA) || errors.As(err, &hostErr) || errors.As(err, &invalid) {
		return false, nil
	}
	return false, err
}
```

(add `"errors"` to the imports.) Run: `go test ./internal/services/ -run TestDomainChecker -race -count=1` → PASS.

- [ ] **Step 3: Prober test against a real TLS server**

```go
// services/api/internal/services/domain_tlsprobe_test.go
package services

import (
	"context"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTraefikTLSProber(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "https://")
	leaf := srv.Certificate() // httptest's cert is valid for 127.0.0.1 and example.com
	name := "example.com"
	if len(leaf.DNSNames) > 0 {
		name = leaf.DNSNames[0]
	}

	trusted := x509.NewCertPool()
	trusted.AddCert(leaf)
	valid, err := (&TraefikTLSProber{Roots: trusted}).Probe(context.Background(), addr, name)
	if err != nil || !valid {
		t.Fatalf("trusted roots: valid=%v err=%v", valid, err)
	}
	valid, err = (&TraefikTLSProber{Roots: x509.NewCertPool()}).Probe(context.Background(), addr, name)
	if err != nil || valid {
		t.Fatalf("untrusted roots must be (false, nil): valid=%v err=%v", valid, err)
	}
	valid, err = (&TraefikTLSProber{Roots: trusted}).Probe(context.Background(), addr, "other.example.org")
	if err != nil || valid {
		t.Fatalf("wrong name must be (false, nil): valid=%v err=%v", valid, err)
	}
	if _, err = (&TraefikTLSProber{}).Probe(context.Background(), "127.0.0.1:1", name); err == nil {
		t.Fatal("closed port must return an error")
	}
}
```

Run: `go test ./internal/services/ -run TestTraefikTLSProber -count=1` → PASS.

- [ ] **Step 4: Handler — pending on add, `Verify`, tests**

Extend `services/api/internal/handlers/domain_test.go` (add imports `net` and `"github.com/dotechhq/zenith/services/api/internal/services"`):

```go
type stubResolver struct{ ips map[string][]string }

func (s stubResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := s.ips[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	out := make([]net.IPAddr, 0, len(ips))
	for _, s := range ips {
		out = append(out, net.IPAddr{IP: net.ParseIP(s)})
	}
	return out, nil
}

type stubProber struct{ valid bool }

func (s stubProber) Probe(context.Context, string, string) (bool, error) { return s.valid, nil }

func TestDomainAddIsPendingWithChecker(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	checker := services.NewDomainChecker(repo, stubResolver{}, stubProber{}, "zenith.example.com")
	handler.SetChecker(checker)
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(`{"domain":"app.example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := fiberApp.Test(req, -1)
	if resp.StatusCode != 201 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Status != entities.DomainStatusPending || out.StatusMessage != "waiting for DNS check" {
		t.Fatalf("got %+v", out)
	}
}

func TestDomainAddStaysActiveWithoutChecker(t *testing.T) {
	fiberApp, handler, _, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains", injectUserID("user-1"), handler.Add)
	req := httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains", bytes.NewBufferString(`{"domain":"app.example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := fiberApp.Test(req, -1)
	var out dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Status != entities.DomainStatusActive {
		t.Fatalf("saas behaviour changed: %+v", out)
	}
}

func TestDomainVerifyRunsOneCheck(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	checker := services.NewDomainChecker(repo, stubResolver{ips: map[string][]string{"zenith.example.com": {"1.2.3.4"}, "app.example.com": {"1.2.3.4"}}}, stubProber{valid: true}, "zenith.example.com")
	handler.SetChecker(checker)
	fiberApp.Post("/api/v1/apps/:appId/domains/:domainId/verify", injectUserID("user-1"), handler.Verify)
	d, _ := repo.AddDomain(context.Background(), appID, "user-1", "app.example.com")
	_ = checker.MarkPending(context.Background(), d.ID)

	resp, _ := fiberApp.Test(httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains/"+d.ID+"/verify", nil), -1)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out dto.DomainInfo
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Status != entities.DomainStatusActive || !out.TLSReady || out.LastCheckedAt == nil {
		t.Fatalf("got %+v", out)
	}
}

func TestDomainVerifyForbiddenForOtherUser(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	handler.SetChecker(services.NewDomainChecker(repo, stubResolver{}, stubProber{}, "zenith.example.com"))
	fiberApp.Post("/api/v1/apps/:appId/domains/:domainId/verify", injectUserID("user-2"), handler.Verify)
	d, _ := repo.AddDomain(context.Background(), appID, "user-1", "app.example.com")
	resp, _ := fiberApp.Test(httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains/"+d.ID+"/verify", nil), -1)
	if resp.StatusCode != 403 {
		t.Fatalf("status %d, want 403", resp.StatusCode)
	}
}

func TestDomainVerifyWithoutCheckerIs404(t *testing.T) {
	fiberApp, handler, repo, appID := setupDomainTest()
	fiberApp.Post("/api/v1/apps/:appId/domains/:domainId/verify", injectUserID("user-1"), handler.Verify)
	d, _ := repo.AddDomain(context.Background(), appID, "user-1", "app.example.com")
	resp, _ := fiberApp.Test(httptest.NewRequest("POST", "/api/v1/apps/"+appID+"/domains/"+d.ID+"/verify", nil), -1)
	if resp.StatusCode != 404 {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
}
```

Run → FAIL. Then in `domain.go` (import `services` and `time`):

```go
type DomainHandler struct {
	domainRepo ports.DomainRepository
	appRepo    ports.AppRepository
	planRepo   ports.UserPlanRepository
	deployer   AppRedeployer
	reserved   string
	checker    *services.DomainChecker // nil = no DNS/TLS checks (SaaS)
}

// SetChecker enables DNS and certificate checks: new domains start pending and /verify works.
func (h *DomainHandler) SetChecker(c *services.DomainChecker) { h.checker = c }
```

In `Add`, after `domain, err := h.domainRepo.AddDomain(...)` succeeds and before `refreshIngress`:

```go
	if h.checker != nil {
		if err := h.checker.MarkPending(c.Context(), domain.ID); err != nil {
			slog.Warn("domain: mark pending failed", "domain", domain.Domain, "error", err)
		} else if fresh, gerr := h.domainRepo.GetDomain(c.Context(), domain.ID); gerr == nil {
			domain = fresh
		}
	}
```

New handler:

```go
// Verify runs the DNS and certificate check for one domain now.
// POST /api/v1/apps/:appId/domains/:domainId/verify
func (h *DomainHandler) Verify(c *fiber.Ctx) error {
	if h.checker == nil {
		return fiber.NewError(fiber.StatusNotFound, "domain checks are not available on this install")
	}
	userID, _ := c.Locals("user_id").(string)
	domain, err := h.domainRepo.GetDomain(c.Context(), c.Params("domainId"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "domain not found")
	}
	if domain.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your domain")
	}
	checked, err := h.checker.Check(c.Context(), domain)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	return c.JSON(toDomainInfo(checked))
}
```

Run: `go test ./internal/handlers/ -run TestDomain -race -count=1` → PASS.

- [ ] **Step 5: Wire it in `main.go`**

After `appByID.Delete("/domains/:domainId", domainHandler.Delete)`:

```go
	appByID.Post("/domains/:domainId/verify", domainHandler.Verify)
```

Right after `domainHandler` is constructed and configured (`:1324` onward), add (`dockerDeployer` is the variable the 2b reconciler block uses; add `"net"` to the imports):

```go
	if cfg.IsStandalone() && dockerDeployer != nil {
		domainChecker := services.NewDomainChecker(domainRepo, net.DefaultResolver, &services.TraefikTLSProber{}, cfg.BaseDomain)
		domainHandler.SetChecker(domainChecker)
		go func() {
			time.Sleep(15 * time.Second)
			domainChecker.Run(context.Background())
		}()
		slog.Info("custom domains: DNS and certificate checks enabled", "dashboard", cfg.BaseDomain)
	}
```

If `dockerDeployer` is scoped inside an earlier block, hoist its declaration (`var dockerDeployer *deploy.DockerDeployer` before that block) so this check can see it.

Run: `go build ./... && go test ./cmd/server/ ./internal/... -race -count=1` → PASS; `gofmt -l`.

- [ ] **Step 6: Commit**

```bash
git add services/api
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(api): custom domains are checked for dns and a valid certificate; verify endpoint

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 7: Managed-database env vars on env-v2, startup migration

**Lane:** API, after Task 6. **Files:**
- Create: `services/api/internal/services/managed_env.go`, `services/api/internal/services/managed_env_test.go`
- Modify: `services/api/internal/services/database.go:22-50` (field + setter), `:196`, `:448`, `:569`, `:634`, `:889`, `:925`, `:1137`
- Modify: `services/api/internal/handlers/databases_v2.go:19-40` (field + setter), `:56`, `:187`, `:243`, `:262`, `:464-476`
- Modify: `services/api/cmd/server/main.go` (construct after `dbHandlerV2`, inject, start migration in the standalone Docker block)

**Interfaces:**
- Produces: `services.EnvKeyForEngine(engine entities.DatabaseEngine) string`; `services.Encrypter` interface `Encrypt(userID, value string) (string, error)`; `services.NewManagedEnv(vars ports.EnvVarRepository, apps ports.AppRepository) *ManagedEnv`; `(*ManagedEnv).SetEncrypter(Encrypter)`; `(*ManagedEnv).Set(ctx, appID, key, value, dbID string) error`; `(*ManagedEnv).SetRaw(ctx, appID, key, storedValue, dbID string) error`; `(*ManagedEnv).Clear(ctx, appID, dbID string) error`; `(*ManagedEnv).Has(ctx, appID, dbID string) (bool, error)`; `services.MigrateManagedEnv(ctx, dbs ports.DatabaseRepository, apps ports.AppRepository, env *ManagedEnv) (int, error)`; `(*DatabaseService).SetManagedEnv(*ManagedEnv)`; `(*DatabaseHandlerV2).SetManagedEnv(*ManagedEnv)`.

- [ ] **Step 1: Failing tests**

```go
// services/api/internal/services/managed_env_test.go
package services

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

type prefixEncrypter struct{}

func (prefixEncrypter) Encrypt(userID, value string) (string, error) { return "enc:" + userID + ":" + value, nil }

func managedEnvFixture(t *testing.T) (*ManagedEnv, *memory.MemoryEnvVarRepository, *memory.MemoryAppRepository, string) {
	t.Helper()
	vars := memory.NewMemoryEnvVarRepository()
	apps := memory.NewMemoryAppRepository()
	app, err := apps.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "user-1", Name: "web", RepoURL: "https://github.com/u/r"})
	if err != nil {
		t.Fatal(err)
	}
	return NewManagedEnv(vars, apps), vars, apps, app.ID
}

func readyDB(t *testing.T, dbs *memory.MemoryDatabaseRepository, appID string, engine entities.DatabaseEngine, name string) *entities.UserDatabase {
	t.Helper()
	db, err := dbs.CreateDatabase(context.Background(), appID, "user-1", &dto.CreateDatabaseInput{Name: name, Engine: engine})
	if err != nil {
		t.Fatal(err)
	}
	// Use the memory repo's status update (name it as the repo defines it: grep 'func (r \*MemoryDatabaseRepository) Update' memory_database.go).
	if err := dbs.UpdateDatabaseStatus(context.Background(), db.ID, entities.DatabaseStatusReady); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestManagedEnv_SetWritesEncryptedSecretWithSource(t *testing.T) {
	env, vars, _, appID := managedEnvFixture(t)
	env.SetEncrypter(prefixEncrypter{})
	if err := env.Set(context.Background(), appID, "DATABASE_URL", "postgres://x", "db-1"); err != nil {
		t.Fatal(err)
	}
	got, _ := vars.GetEnvVarsByEnvironment(context.Background(), appID, "")
	if len(got) != 1 || got[0].Key != "DATABASE_URL" || got[0].Value != "enc:user-1:postgres://x" || !got[0].IsSecret ||
		got[0].Source != entities.EnvVarSourceManagedService || got[0].SourceID != "db-1" || got[0].EnvironmentID != "" {
		t.Fatalf("got %+v", got)
	}
	has, _ := env.Has(context.Background(), appID, "db-1")
	if !has {
		t.Fatal("Has = false")
	}
}

func TestManagedEnv_ClearRemovesOnlyThatDatabase(t *testing.T) {
	env, vars, _, appID := managedEnvFixture(t)
	_ = env.Set(context.Background(), appID, "DATABASE_URL", "pg", "db-1")
	_ = env.Set(context.Background(), appID, "REDIS_URL", "redis", "db-2")
	_ = vars.SetEnvVar(context.Background(), &entities.AppEnvVar{AppID: appID, Key: "MANUAL", Value: "x", Source: entities.EnvVarSourceManual})
	if err := env.Clear(context.Background(), appID, "db-1"); err != nil {
		t.Fatal(err)
	}
	got, _ := vars.GetEnvVarsByEnvironment(context.Background(), appID, "")
	keys := map[string]bool{}
	for _, v := range got {
		keys[v.Key] = true
	}
	if keys["DATABASE_URL"] || !keys["REDIS_URL"] || !keys["MANUAL"] {
		t.Fatalf("keys after Clear = %v", keys)
	}
}

func TestMigrateManagedEnv_CopiesLegacyAndDeletesIt(t *testing.T) {
	env, vars, apps, appID := managedEnvFixture(t)
	dbs := memory.NewMemoryDatabaseRepository()
	db := readyDB(t, dbs, appID, entities.DatabaseEnginePostgreSQL, "pg1")
	_ = apps.SetEnvVars(context.Background(), appID, map[string]string{"DATABASE_URL": "enc:v1:abc", "OTHER": "keep"})

	n, err := MigrateManagedEnv(context.Background(), dbs, apps, env)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	v2, _ := vars.GetEnvVarsByEnvironment(context.Background(), appID, "")
	if len(v2) != 1 || v2[0].Value != "enc:v1:abc" || v2[0].SourceID != db.ID || v2[0].Source != entities.EnvVarSourceManagedService {
		t.Fatalf("v2 = %+v", v2)
	}
	legacy, _ := apps.GetEnvVars(context.Background(), appID)
	if len(legacy) != 1 || legacy[0].Key != "OTHER" {
		t.Fatalf("legacy = %+v (DATABASE_URL must be gone, OTHER kept)", legacy)
	}
}

func TestMigrateManagedEnv_Idempotent(t *testing.T) {
	env, vars, apps, appID := managedEnvFixture(t)
	dbs := memory.NewMemoryDatabaseRepository()
	readyDB(t, dbs, appID, entities.DatabaseEngineRedis, "r1")
	_ = apps.SetEnvVars(context.Background(), appID, map[string]string{"REDIS_URL": "redis://x"})
	if n, _ := MigrateManagedEnv(context.Background(), dbs, apps, env); n != 1 {
		t.Fatalf("first run n=%d", n)
	}
	if n, _ := MigrateManagedEnv(context.Background(), dbs, apps, env); n != 0 {
		t.Fatalf("second run n=%d, want 0", n)
	}
	v2, _ := vars.GetEnvVarsByEnvironment(context.Background(), appID, "")
	if len(v2) != 1 {
		t.Fatalf("v2 = %+v", v2)
	}
}

func TestMigrateManagedEnv_SkipsUnattachedAndNotReady(t *testing.T) {
	env, _, apps, appID := managedEnvFixture(t)
	dbs := memory.NewMemoryDatabaseRepository()
	readyDB(t, dbs, "", entities.DatabaseEnginePostgreSQL, "x") // ready but unattached
	if _, err := dbs.CreateDatabase(context.Background(), appID, "user-1", &dto.CreateDatabaseInput{Name: "y", Engine: entities.DatabaseEnginePostgreSQL}); err != nil { // provisioning
		t.Fatal(err)
	}
	_ = apps.SetEnvVars(context.Background(), appID, map[string]string{"DATABASE_URL": "pg"})
	if n, err := MigrateManagedEnv(context.Background(), dbs, apps, env); n != 0 || err != nil {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
```

Check `dto.CreateDatabaseInput` field names and the memory database repo's status-update method name; adjust `readyDB` to the real signatures. Run → FAIL to compile.

- [ ] **Step 2: Implement the helper and the migration**

```go
// services/api/internal/services/managed_env.go
package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// EnvKeyForEngine is the variable name an attached database injects into its app.
func EnvKeyForEngine(engine entities.DatabaseEngine) string {
	switch engine {
	case entities.DatabaseEngineRedis:
		return "REDIS_URL"
	case entities.DatabaseEngineMySQL:
		return "MYSQL_URL"
	case entities.DatabaseEngineMongoDB:
		return "MONGODB_URL"
	case entities.DatabaseEngineRabbitMQ:
		return "RABBITMQ_URL"
	case entities.DatabaseEngineKafka:
		return "KAFKA_BROKERS"
	default:
		return "DATABASE_URL"
	}
}

// Encrypter encrypts a secret for a user; *crypto.EnvCrypto satisfies it.
type Encrypter interface {
	Encrypt(userID, value string) (string, error)
}

// ManagedEnv writes the connection variables of attached databases as env-v2
// rows (source managed_service), so the Environment tab shows them and a
// detach removes exactly that database's variables.
type ManagedEnv struct {
	vars ports.EnvVarRepository
	apps ports.AppRepository
	enc  Encrypter
}

func NewManagedEnv(vars ports.EnvVarRepository, apps ports.AppRepository) *ManagedEnv {
	return &ManagedEnv{vars: vars, apps: apps}
}

// SetEncrypter makes Set store values encrypted, like secrets set from the dashboard.
func (m *ManagedEnv) SetEncrypter(e Encrypter) { m.enc = e }

// Set upserts the variable for the app's default environment, encrypting it when an encrypter is configured.
func (m *ManagedEnv) Set(ctx context.Context, appID, key, value, dbID string) error {
	stored := value
	if m.enc != nil {
		app, err := m.apps.GetApp(ctx, appID)
		if err != nil {
			return fmt.Errorf("managed env: app %s: %w", appID, err)
		}
		enc, err := m.enc.Encrypt(app.UserID, value)
		if err != nil {
			return fmt.Errorf("managed env: encrypt %s: %w", key, err)
		}
		stored = enc
	}
	return m.SetRaw(ctx, appID, key, stored, dbID)
}

// SetRaw upserts an already-stored value (used by the migration, which must not re-encrypt).
func (m *ManagedEnv) SetRaw(ctx context.Context, appID, key, storedValue, dbID string) error {
	return m.vars.SetEnvVar(ctx, &entities.AppEnvVar{
		AppID: appID, EnvironmentID: "", Key: key, Value: storedValue, IsSecret: true,
		Source: entities.EnvVarSourceManagedService, SourceID: dbID,
	})
}

// Clear deletes the variables that database dbID injected into appID; other sources and databases are untouched.
func (m *ManagedEnv) Clear(ctx context.Context, appID, dbID string) error {
	vars, err := m.vars.GetEnvVarsByEnvironment(ctx, appID, "")
	if err != nil {
		return err
	}
	for _, v := range vars {
		if v.Source == entities.EnvVarSourceManagedService && v.SourceID == dbID {
			if err := m.vars.DeleteEnvVar(ctx, v.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// Has reports whether appID already holds a variable from database dbID.
func (m *ManagedEnv) Has(ctx context.Context, appID, dbID string) (bool, error) {
	vars, err := m.vars.GetEnvVarsByEnvironment(ctx, appID, "")
	if err != nil {
		return false, err
	}
	for _, v := range vars {
		if v.Source == entities.EnvVarSourceManagedService && v.SourceID == dbID {
			return true, nil
		}
	}
	return false, nil
}

// MigrateManagedEnv copies, once, the legacy-table connection variable of every
// ready, attached database into env-v2 and deletes the legacy row after the copy.
// It is idempotent: a database whose env-v2 variable exists is skipped.
func MigrateManagedEnv(ctx context.Context, dbs ports.DatabaseRepository, apps ports.AppRepository, env *ManagedEnv) (int, error) {
	ready, err := dbs.ListDatabasesByStatus(ctx, entities.DatabaseStatusReady)
	if err != nil {
		return 0, err
	}
	migrated := 0
	for i := range ready {
		db := &ready[i]
		if db.AppID == "" {
			continue
		}
		has, err := env.Has(ctx, db.AppID, db.ID)
		if err != nil || has {
			continue
		}
		key := EnvKeyForEngine(db.Engine)
		legacy, err := apps.GetEnvVars(ctx, db.AppID)
		if err != nil {
			slog.Warn("managed env migration: read legacy vars", "app", db.AppID, "error", err)
			continue
		}
		for _, l := range legacy {
			if l.Key != key {
				continue
			}
			if err := env.SetRaw(ctx, db.AppID, key, l.Value, db.ID); err != nil {
				slog.Warn("managed env migration: copy failed; legacy row kept", "app", db.AppID, "key", key, "error", err)
				break
			}
			if err := apps.DeleteEnvVar(ctx, db.AppID, key); err != nil {
				slog.Warn("managed env migration: legacy row not deleted", "app", db.AppID, "key", key, "error", err)
			}
			slog.Info("managed env migration: moved variable to env-v2", "app", db.AppID, "database", db.Name, "key", key)
			migrated++
			break
		}
	}
	return migrated, nil
}
```

Check `ports.AppRepository.DeleteEnvVar`'s signature (`DeleteEnvVar(ctx, appID, key string)` as used at `database.go:448`) and `GetEnvVars`'s element type (`Key`/`Value` fields). Run: `go test ./internal/services/ -run 'TestManagedEnv|TestMigrateManagedEnv' -race -count=1` → PASS.

- [ ] **Step 3: Route the eleven call sites through the helper**

`services/api/internal/services/database.go`: add field `managedEnv *ManagedEnv` to `DatabaseService` and

```go
// SetManagedEnv routes the injected connection variables to env-v2 (Compose edition).
func (s *DatabaseService) SetManagedEnv(m *ManagedEnv) { s.managedEnv = m }

// setInjectedEnv writes the database's connection variable on its app, via env-v2 when configured.
func (s *DatabaseService) setInjectedEnv(ctx context.Context, appID, key, value, dbID string) {
	if s.managedEnv != nil {
		if err := s.managedEnv.Set(ctx, appID, key, value, dbID); err != nil {
			slog.Warn("database: inject env failed", "app", appID, "key", key, "error", err)
		}
		return
	}
	s.appRepo.SetEnvVars(ctx, appID, map[string]string{key: value})
}

func (s *DatabaseService) clearInjectedEnv(ctx context.Context, appID, key, dbID string) {
	if s.managedEnv != nil {
		if err := s.managedEnv.Clear(ctx, appID, dbID); err != nil {
			slog.Warn("database: clear env failed", "app", appID, "error", err)
		}
		return
	}
	s.appRepo.DeleteEnvVar(ctx, appID, key)
}
```

Replace: `:196` → `s.setInjectedEnv(ctx, appID, envKey, connStr, db.ID)`; `:448` → `s.clearInjectedEnv(ctx, db.AppID, envKey, db.ID)`; `:569` → `s.setInjectedEnv(ctx, db.AppID, "REDIS_URL", connStr, db.ID)`; `:634` → `s.setInjectedEnv(ctx, db.AppID, resetEnvKey, connStr, db.ID)`; `:889` → `s.setInjectedEnv(ctx, db.AppID, "RABBITMQ_URL", connStr, db.ID)`; `:925` → `s.setInjectedEnv(ctx, db.AppID, "MONGODB_URL", connStr, db.ID)`; `:1137` → `s.setInjectedEnv(ctx, db.AppID, "KAFKA_BROKERS", connStr, db.ID)`.

`services/api/internal/handlers/databases_v2.go`: add field `managedEnv *services.ManagedEnv`, `func (h *DatabaseHandlerV2) SetManagedEnv(m *services.ManagedEnv) { h.managedEnv = m }`, and:

```go
func (h *DatabaseHandlerV2) setInjectedEnv(ctx context.Context, appID, key, value, dbID string) error {
	if h.managedEnv != nil {
		return h.managedEnv.Set(ctx, appID, key, value, dbID)
	}
	return h.appRepo.SetEnvVars(ctx, appID, map[string]string{key: value})
}

func (h *DatabaseHandlerV2) clearInjectedEnv(ctx context.Context, appID, key, dbID string) {
	if h.managedEnv != nil {
		_ = h.managedEnv.Clear(ctx, appID, dbID)
		return
	}
	_ = h.appRepo.DeleteEnvVar(ctx, appID, key)
}
```

Replace `:56` with `if err := h.setInjectedEnv(ctx, appID, envKeyForEngine(fresh.Engine), fresh.ConnectionString(pw), db.ID); err != nil {` (keep the warn-and-return body); `:187` → `h.setInjectedEnv(c.Context(), appID, envKey, connStr, db.ID)` (ignore the error as the current code does); `:243` → `h.clearInjectedEnv(c.Context(), db.AppID, envKeyForEngine(db.Engine), db.ID)`; `:262` → `h.clearInjectedEnv(c.Context(), db.AppID, envKey, db.ID)`. Make `envKeyForEngine` delegate: `func envKeyForEngine(engine entities.DatabaseEngine) string { return services.EnvKeyForEngine(engine) }`.

Run: `go build ./... && go test ./internal/services/ ./internal/handlers/ -race -count=1` → PASS (existing database tests still use the legacy path because no `ManagedEnv` is set).

- [ ] **Step 4: Wire and run the migration at startup**

In `main.go`, right after `dbHandlerV2 := handlers.NewDatabaseHandlerV2(...)` (`:897`):

```go
	managedEnv := services.NewManagedEnv(envVarRepo, appRepo)
	if envCrypto != nil {
		managedEnv.SetEncrypter(envCrypto)
	}
	dbHandlerV2.SetManagedEnv(managedEnv)
	if dbSvc != nil {
		dbSvc.SetManagedEnv(managedEnv)
	}
```

(`envVarRepo` is declared at `:571` and `envCrypto` at `:624`, both before this point.) In the standalone Docker block, after the reconciler goroutine:

```go
			go func() {
				time.Sleep(20 * time.Second)
				if n, err := services.MigrateManagedEnv(context.Background(), dbRepo, appRepo, managedEnv); err != nil {
					slog.Warn("managed env migration failed", "error", err)
				} else if n > 0 {
					slog.Info("managed env migration done", "moved", n)
				}
			}()
```

Run: `go build ./... && go test ./... -race -count=1 && gofmt -l cmd internal` → PASS, clean.

- [ ] **Step 5: Commit**

```bash
git add services/api
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(api): attached databases write their variables to env-v2; legacy rows migrate at startup

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 8: Web — domain status and Check now, Networking page lists real domains, managed env badge

**Lane:** Web, after Task 2. **Files:**
- Modify: `apps/web/src/lib/api.ts:1073-1080` (`CustomDomain`), `:1575-1585` (add `verifyDomain`), add `customDomains.list`
- Create: `apps/web/src/components/domain-status.tsx`, `apps/web/src/components/__tests__/domain-status.test.tsx`
- Create: `apps/web/src/lib/env-source.ts`, `apps/web/src/lib/__tests__/env-source.test.ts`
- Modify: `apps/web/src/app/apps/[name]/page.tsx` DomainsTab (`:1840-1890`) and EnvTab row (`:796-880`)
- Modify: `apps/web/src/app/networking/page.tsx` (whole file)

**Interfaces:**
- Consumes: domain JSON with `status_message`, `last_checked_at`; `POST /api/v1/apps/:appId/domains/:domainId/verify`; `GET /api/v1/domains`; env-v2 `source === "managed_service"`.
- Produces: `<DomainStatus domain={…} onCheck={…} checking={…} />`; `isManagedEnv(ev): boolean`, `managedEnvLabel(ev): string`.

- [ ] **Step 1: API types**

```ts
export interface CustomDomain {
  id: string;
  app_id: string;
  domain: string;
  status: "pending" | "verified" | "active" | "failed" | string;
  tls_ready: boolean;
  status_message: string;
  last_checked_at: string | null;
  created_at: string;
}
```

(keep any other fields the interface already has.) In `appsDeploy`, after `deleteDomain`:

```ts
  verifyDomain: (appId: string, domainId: string) =>
    apiFetch<CustomDomain>(`/api/v1/apps/${appId}/domains/${domainId}/verify`, { method: "POST" }),
```

And a new export next to `appsDeploy`:

```ts
export const customDomains = {
  list: () => apiFetch<CustomDomain[]>("/api/v1/domains"),
};
```

- [ ] **Step 2: Failing component test**

```tsx
// apps/web/src/components/__tests__/domain-status.test.tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DomainStatus } from "../domain-status";

const base = { id: "d1", app_id: "a1", domain: "app.example.com", tls_ready: false, last_checked_at: null, created_at: "2026-10-16T00:00:00Z" };

describe("DomainStatus", () => {
  it("shows the pending message and a Check now button", () => {
    const onCheck = vi.fn();
    render(<DomainStatus domain={{ ...base, status: "pending", status_message: "DNS: no A/AAAA record for app.example.com; create an A record to 1.2.3.4" }} onCheck={onCheck} checking={false} />);
    expect(screen.getByText("pending")).toBeInTheDocument();
    expect(screen.getByText(/create an A record to 1\.2\.3\.4/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /check now/i }));
    expect(onCheck).toHaveBeenCalledTimes(1);
  });
  it("shows active with HTTPS and no message", () => {
    render(<DomainStatus domain={{ ...base, status: "active", tls_ready: true, status_message: "" }} onCheck={() => {}} checking={false} />);
    expect(screen.getByText("active")).toBeInTheDocument();
    expect(screen.getByText(/HTTPS ready/)).toBeInTheDocument();
  });
  it("disables the button while checking", () => {
    render(<DomainStatus domain={{ ...base, status: "verified", status_message: "DNS ok; waiting for the certificate" }} onCheck={() => {}} checking={true} />);
    expect(screen.getByRole("button", { name: /checking/i })).toBeDisabled();
  });
});
```

Run: `pnpm exec vitest run src/components` → FAIL.

- [ ] **Step 3: Component**

```tsx
// apps/web/src/components/domain-status.tsx
"use client";
import type { CustomDomain } from "@/lib/api";

const COLORS: Record<string, string> = {
  active: "text-emerald-400",
  verified: "text-blue-400",
  pending: "text-amber-400",
};

export function DomainStatus({ domain, onCheck, checking }: { domain: CustomDomain; onCheck: () => void; checking: boolean }) {
  const color = COLORS[domain.status] ?? "text-red-400";
  return (
    <div className="space-y-1">
      <div className="flex items-center gap-3">
        <span className={`inline-flex items-center gap-1.5 text-xs ${color}`}>
          <span className={`h-1.5 w-1.5 rounded-full ${domain.status === "pending" ? "animate-pulse" : ""} bg-current`} />
          {domain.status}
        </span>
        {domain.tls_ready && <span className="text-xs text-emerald-400">HTTPS ready</span>}
        <button
          type="button"
          onClick={onCheck}
          disabled={checking}
          className="rounded border border-border px-2 py-0.5 text-[11px] text-neutral-400 hover:text-white disabled:opacity-50 transition-colors"
        >
          {checking ? "Checking…" : "Check now"}
        </button>
      </div>
      {domain.status_message && <p className="text-xs text-neutral-500">{domain.status_message}</p>}
      {domain.last_checked_at && (
        <p className="text-[11px] text-neutral-600">checked {new Date(domain.last_checked_at).toLocaleTimeString()}</p>
      )}
    </div>
  );
}
```

Run: `pnpm exec vitest run src/components` → PASS.

- [ ] **Step 4: Use it in the app page's Domains tab**

In `DomainsTab`, add state `const [checkingId, setCheckingId] = useState<string | null>(null);` and

```tsx
  const handleCheck = async (domainId: string) => {
    setCheckingId(domainId);
    try {
      await appsDeploy.verifyDomain(appId, domainId);
      refetch();
    } catch {
      toast("error", "Domain check failed");
    } finally {
      setCheckingId(null);
    }
  };
```

Replace the Status and TLS `<td>` pair with one cell (and merge the two `<th>`s into one "Status" header with `colSpan={2}`, or keep both headers and use `colSpan={2}` on the cell):

```tsx
                  <td className="px-4 py-3" colSpan={2}>
                    <DomainStatus domain={d} onCheck={() => handleCheck(d.id)} checking={checkingId === d.id} />
                  </td>
```

(import `DomainStatus` from `@/components/domain-status`.)

- [ ] **Step 5: Managed env badge — failing test, helper, EnvTab**

```ts
// apps/web/src/lib/__tests__/env-source.test.ts
import { describe, it, expect } from "vitest";
import { isManagedEnv, managedEnvLabel } from "../env-source";

describe("env-source", () => {
  it("recognises variables injected by a managed database", () => {
    expect(isManagedEnv({ source: "managed_service", source_id: "db-1" })).toBe(true);
    expect(isManagedEnv({ source: "manual" })).toBe(false);
    expect(managedEnvLabel({ source: "managed_service", source_id: "db-1" })).toBe("from database");
  });
});
```

```ts
// apps/web/src/lib/env-source.ts
export function isManagedEnv(ev: { source: string; source_id?: string }): boolean {
  return ev.source === "managed_service";
}

export function managedEnvLabel(_ev: { source: string; source_id?: string }): string {
  return "from database";
}
```

In `EnvTab`'s row (`items.map((ev) => …)`): next to the key, render `{isManagedEnv(ev) && <span className="rounded bg-blue-500/10 px-1.5 py-0.5 text-[10px] text-blue-400">{managedEnvLabel(ev)}</span>}` and wrap the edit and delete buttons in `{!isManagedEnv(ev) && ( … )}` (the edit button at `:865` and the delete button around `:876`). Add a one-line hint under the table when any item is managed: `Variables from an attached database are read-only; detach the database to remove them.`

Run: `pnpm exec vitest run src/lib && pnpm exec tsc --noEmit` → PASS.

- [ ] **Step 6: Networking page — real custom domains, no fake modal**

Rewrite `apps/web/src/app/networking/page.tsx` so that:
- it loads `api.appsDeploy.list()` (app subdomains, as today) **and** `customDomains.list()`;
- the table has two groups: "App subdomains" (status from the app, as today) and "Custom domains" (one row per `CustomDomain`: domain, app name resolved from the apps list by `app_id`, and `<DomainStatus … />` with `onCheck` calling `appsDeploy.verifyDomain(d.app_id, d.id)` and reloading);
- the "+ Add Domain" button and the modal are removed; the empty state says `Add a custom domain from an app's Domains tab.`;
- the "Load Balancers" section is removed (it is a SaaS mock: there is no load balancer on a Compose install).

Keep the file a client component with the same `Shell`. Run: `pnpm exec tsc --noEmit && pnpm --filter @zenith/web lint && pnpm --filter @zenith/web test` → PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/web
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(web): domain dns and certificate status with check now; managed database variables shown read-only

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 9: Integrate the lanes, bump the version, full suites

**Lane:** Controller, after Tasks 4, 7 and 8 are reviewed. **Files:**
- Modify: `VERSION`
- Modify: `docs/superpowers/specs/2026-10-16-alpha-2c-self-host-cleanup-design.md` §5 (one bullet: the custom-domain UI lives on the app page's Domains tab; the Networking page lists real domains and loses its mock modal and load-balancer card)

- [ ] **Step 1: Merge the lanes**

```bash
cd /opt/Zenith
git checkout feat/alpha-2c-self-host-cleanup
git merge --no-ff feat/alpha-2c-api -m "merge: 2c api lane"
git merge --no-ff feat/alpha-2c-web -m "merge: 2c web lane"
git merge --no-ff feat/alpha-2c-cli -m "merge: 2c cli lane"
```

Expected: no conflicts (the lanes touch disjoint files). If a merge reports a conflict, stop and resolve it by hand; do not use `-X theirs`.

- [ ] **Step 2: Version and spec note**

`VERSION` → `0.1.0-alpha.4`. Edit the spec's §5 "Web Domains page" bullet to describe the app page's Domains tab and the Networking page as built in Task 8.

- [ ] **Step 3: Everything green**

```bash
(cd services/api && gofmt -l cmd internal && go vet ./... && go test ./... -race -count=1)
(cd cli && gofmt -l . && go vet ./... && go test ./... -race -count=1)
bash infra/scripts/install_test.sh && bash infra/scripts/compose_networks_test.sh
pnpm --filter @zenith/web test && pnpm --filter @zenith/web exec tsc --noEmit && pnpm --filter @zenith/web lint
(cd services/api && ZENITH_MODE=standalone ZENITH_DUMP_ROUTES=1 go run ./cmd/server 2>/dev/null | grep -Ec '^(GET|POST|PUT|DELETE) /api/v1/(billing|plan|referral|admin/(plans|billing|exit-surveys|referrals|crm))')
```

Expected: every suite PASS, `gofmt` silent, the last command prints `0`.

- [ ] **Step 4: Commit and clean up the worktrees**

```bash
git add VERSION docs/superpowers/specs/2026-10-16-alpha-2c-self-host-cleanup-design.md
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "chore(release): version 0.1.0-alpha.4; spec note on the domain ui

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
git worktree remove .worktrees/2c-api && git worktree remove .worktrees/2c-web && git worktree remove .worktrees/2c-cli
git branch -d feat/alpha-2c-api feat/alpha-2c-web feat/alpha-2c-cli
```

---

### Task 10: Live acceptance on `zenith.freezenith.com`, core-audit §8, PR, release

**Lane:** Controller. **Files:**
- Modify: `docs/alpha/core-audit.md` (new `## 8. After 2c`; findings table rows F-01, F-02, F-03, F-09, F-23, F-24, F-25, F-26, F-28 → fixed)

- [ ] **Step 1: Build and upgrade the live install**

```bash
cd /opt/Zenith
docker build -t ghcr.io/taikuri-infra/zenith-api:0.1.0-alpha.4 --build-arg VERSION=0.1.0-alpha.4 -f services/api/Dockerfile .
docker build -t ghcr.io/taikuri-infra/zenith-web:0.1.0-alpha.4 --build-arg VERSION=0.1.0-alpha.4 --build-arg NEXT_PUBLIC_ZENITH_MODE=standalone -f apps/web/Dockerfile .
(cd cli && go build -o /usr/local/bin/zen .)
cd /root/zenith-2a && git fetch /opt/Zenith feat/alpha-2c-self-host-cleanup && git tag -f v0.1.0-alpha.4 FETCH_HEAD
zen upgrade --version 0.1.0-alpha.4
curl -s 127.0.0.1:8080/api/v1/version
```

Expected: upgrade reports success with a backup taken first; version JSON shows `0.1.0-alpha.4` and `"mode":"standalone"`.

- [ ] **Step 2: Criteria 1–3 (API surface, pages, typecheck)**

```bash
docker exec zenith-2a-api-1 sh -c 'ZENITH_DUMP_ROUTES=1 /app/server' 2>/dev/null | grep -Ec '/api/v1/(billing|plan|referral|admin/crm)'
curl -s -o /dev/null -w '%{http_code}\n' -X POST https://zenith.freezenith.com/api/v1/auth/register -H 'Content-Type: application/json' -d '{"email":"x@y.z","password":"longenough","name":"x"}'
```

Expected: `0` and `403`. In a browser (or headless Chromium as in §6): `/login` shows only email and password and the tagline `Self-hosted PaaS`; `/login?mode=register` shows the login form; `/docs`, `/` (Overview), the deploy wizard review step and `/databases` show no plan, pricing or freezenith.com sign-up text; the Databases header counts MySQL. Record the browser console: no 404 from `/api/v1/plan`. CI on the PR runs `tsc` (step added in Task 2).

- [ ] **Step 3: Criterion 4 (CLI)**

```bash
rm -f ~/.zen/config.yaml
zen login --endpoint https://zenith.freezenith.com --email "$ADMIN_EMAIL" --password "$ADMIN_PASSWORD"
zen status
sed -i 's/^token_expires_at: .*/token_expires_at: 2020-01-01T00:00:00Z/' ~/.zen/config.yaml
zen status            # must succeed and rewrite token_expires_at to the future
zen logout && zen status
```

Expected: login prints `Logged in to … as …`; status lists apps with URLs and databases, no Plan/Region/Cost; the forced-expiry run succeeds without prompting; after logout `Not logged in — run: zen login`.

- [ ] **Step 4: Criterion 5 (domain)**

Pick a hostname under `freezenith.com` that does not resolve yet (`dig +short app2c.freezenith.com` → empty). In the app page's Domains tab add it → the row shows `pending` with the message naming the server IP. Create the A record at the DNS provider → within two minutes the row moves to `verified` and then `active` with HTTPS ready, with no clicks; `curl -sI https://app2c.freezenith.com | head -1` → `HTTP/2 200`. Then click Check now once and note the `checked …` time changes. Also verify the Networking page lists the domain with the same status.

- [ ] **Step 5: Criterion 6 (env)**

Attach a managed Postgres to an app → the Environment tab shows `DATABASE_URL` with the `from database` badge and no edit/delete controls; the container has the variable (`docker exec <app> env | grep DATABASE_URL`). Detach → the row disappears. For a pre-2c attachment (the §6 apps), the row is present after the upgrade's API restart (`docker logs zenith-2a-api-1 | grep 'managed env migration'`).

- [ ] **Step 6: Write §8 and commit**

Add `## 8. After 2c` to `docs/alpha/core-audit.md` in the style of §7: date/commit/images; a table with the seven criteria, status and evidence; `### 8.1 Findings` (the nine fixed, with the route dump count); `### 8.2 Known gaps` (anything deferred during the tasks). Update the §5 findings table: `F-01 (fixed in 2c)` etc. for the nine findings.

```bash
git add docs/alpha/core-audit.md
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "docs(alpha): core audit — status after 2c

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

- [ ] **Step 7: PR, merge, release**

Write the PR body to the scratchpad with this content, then push, open, watch, merge and release:

```markdown
## Summary

Alpha sub-project 2c: a self-hosted Zenith shows no SaaS surface, the CLI logs in with email and password, custom domains report their DNS and certificate state, and attached databases show their variables in the Environment tab. Spec: `docs/superpowers/specs/2026-10-16-alpha-2c-self-host-cleanup-design.md`; live acceptance in `docs/alpha/core-audit.md` §8.

## Changes

- **API:** `IsStandalone()` gates 24 billing/plan/referral/CRM/exit-survey routes; registration returns 403 on self-host; `/version` reports `mode`; domain checker (DNS + certificate via Traefik) with `status_message`, `last_checked_at` and `POST …/domains/:id/verify`; attached databases write env-v2 rows (`managed_service`) and legacy rows migrate at startup.
- **Web:** self-host login page; Compose-only Docs; plan content behind `SHOW_PLANS`; MySQL counted; domain status with Check now on the app page and the Networking page; managed variables read-only; `tsc --noEmit` fixed and run in CI.
- **CLI:** `zen login` with email/password and MFA, automatic token refresh, `zen logout`; `zen status` for standalone; help text without Zenith Cloud/Hetzner; version regex accepts `-` in a prerelease.

## Testing

- [x] `go test ./... -race` (services/api, cli), installer shell tests, web vitest, `tsc --noEmit`, lint
- [x] Standalone route dump contains no SaaS routes
- [x] Live acceptance on zenith.freezenith.com (core-audit §8): register 403, login page, docs, zen login/status/logout with forced expiry, domain pending → verified → active, env badge and migration, `zen upgrade` to 0.1.0-alpha.4

Closes #9, closes #10, closes #11, closes #17

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

```bash
git push -u origin feat/alpha-2c-self-host-cleanup
gh pr create --base main --title "feat: alpha 2c — self-host cleanup (standalone routes, zen login, domain checks, env-v2)" --body-file /tmp/claude-0/-opt-Zenith/97ff95c2-1bfd-4a40-a48e-ea031a38b179/scratchpad/pr-2c-body.md
gh pr checks --watch
gh pr merge --squash --subject "feat: alpha 2c — self-host cleanup (standalone routes, zen login, domain checks, env-v2)"
```

Release notes (scratchpad `release-2c-notes.md`): the "What's new" bullets above in user terms, the two image names `ghcr.io/taikuri-infra/zenith-api:0.1.0-alpha.4` and `zenith-web:0.1.0-alpha.4` (amd64 + arm64, prerelease, `:latest` not moved), and a one-line upgrade note (`zen upgrade --version 0.1.0-alpha.4`; `zen login` now asks for email and password). Then:

```bash
gh release create v0.1.0-alpha.4 --target main --prerelease --latest=false --title "v0.1.0-alpha.4 — self-host cleanup (alpha 2c)" --notes-file /tmp/claude-0/-opt-Zenith/97ff95c2-1bfd-4a40-a48e-ea031a38b179/scratchpad/release-2c-notes.md
```

Then watch `build-public-images.yml` and `release-cli.yml`, and confirm `docker manifest inspect ghcr.io/taikuri-infra/zenith-api:0.1.0-alpha.4` lists amd64 and arm64.
