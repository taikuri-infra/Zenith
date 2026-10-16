# Zenith Alpha — Sub-project 2c: Self-host Cleanup — Design

- **Date:** 2026-10-16
- **Status:** draft for owner review
- **Inputs:** docs/alpha/core-audit.md (§5 findings, §6/§7 status), docs/superpowers/specs/2026-10-12-alpha-2b-compose-durability-design.md ("Next spec (2c)"), docs/superpowers/specs/2026-10-12-alpha-release-design.md (exit criteria)
- **Ships as:** `0.1.0-alpha.4`, with core-audit §8

## 1. Goal and scope

A person outside the team who self-hosts Zenith (Compose edition) must see no trace of the SaaS product, must be able to log in from the CLI without copying a token, and must get a truthful answer about a custom domain's DNS and certificate. 2c closes every finding the 2a audit left open except the ones the alpha exit criteria do not need.

**Decisions taken with the owner on 2026-10-16:**

1. **Single admin.** A self-host install has one account: the administrator the installer creates from `ADMIN_EMAIL` / `ADMIN_PASSWORD`. Registration is off in standalone mode. No invite flow, no user management (alpha is single-user).
2. **`zen login` with email and password.** The CLI logs in like the dashboard does, stores the access and refresh tokens, and refreshes them itself. No personal API keys on the core routes (left for later).
3. **Custom domains are accepted before DNS is ready.** A new domain is saved and routed at once, with status `pending` and a message that tells the user which record to create. The API re-checks DNS and the certificate until both are good.

**In scope (2c):**

| # | Item | Findings | Area |
|---|------|----------|------|
| 1 | SaaS route groups not registered in standalone; registration disabled | F-01, F-02 | api |
| 2 | Login page for self-host; SaaS leftovers on Docs, Overview, deploy wizard, Databases; TypeScript errors; `tsc` in CI | F-25, F-23, F-24, F-03 | web, ci |
| 3 | `zen login` with email/password and token refresh; `zen logout`; `zen status` for standalone; help text | F-09, F-28 | cli |
| 4 | Custom-domain DNS verification and `tls_ready` | F-26 | api, web |
| 5 | Managed-database env vars written to env-v2, visible in the Environment tab | F-26 | api, web |
| 6 | Version string check accepts `-` inside a prerelease (carried from 2b) | — | cli |
| 7 | core-audit §8, release `0.1.0-alpha.4`, close issues #9, #10, #11, #17 | — | docs, release |

**Out of scope:** multi-user and invites; API keys on core routes; the E2E suite (sub-project 3); any change to SaaS behaviour when `ZENITH_MODE=saas`; deleting SaaS code; k3s; the remaining 2b deferred minors (detached HEAD after rollback, `BUILD_TIME`, test gaps) — they stay listed in §7.2 of the audit.

### Success criteria

All on the live install (`zenith.freezenith.com`), recorded in core-audit §8:

1. `ZENITH_DUMP_ROUTES=1` in standalone lists none of the billing, plan, referral, CRM or exit-survey routes (§2). `POST /api/v1/auth/register` returns 403 with the message in §2.
2. The login page shows only email and password: no Google, GitHub or Sign up, no Hetzner text. `/login?mode=register` shows the login form. Docs, Overview, the deploy wizard and the Databases page show no plan, pricing or freezenith.com sign-up content; the Databases header counts MySQL.
3. `pnpm exec tsc --noEmit` passes in `apps/web`, and CI runs it.
4. `zen login` with the admin email and password succeeds without pasting a token; `zen status` lists the install's apps and databases with no plan or cost lines; a command run after the access token has expired succeeds without a new login; `zen logout` clears the session.
5. A domain added before its DNS record exists is saved as `pending` with a message naming the expected IP. After the A record is created, the domain becomes `verified` and then `active` with `tls_ready=true` within two minutes, with no user action, and `curl https://<domain>` returns 200 with a valid certificate. "Check now" in the UI runs the check immediately.
6. Attaching a managed database to an app shows its connection variable in the app's Environment tab, marked as coming from that database; detaching removes it. Databases attached before 2c appear there after one API restart.
7. `0.1.0-alpha.4` images are published from a GitHub prerelease and the live install is upgraded to them with `zen upgrade`.

## 2. Standalone mode in the API (F-01, F-02)

**Today:** `config.Config.Mode` is `standalone` unless `ZENITH_MODE=saas`, but route registration in `cmd/server/main.go` ignores it. 24 routes in the billing, plan, referral, CRM and exit-survey families are registered in both modes (the audit's route dump listed 12 of them by path shape). `POST /api/v1/auth/register` is open on every install; an account created this way can never be verified because there is no email service.

**Design:**

- `config.Config` gets `IsStandalone() bool` (`Mode != "saas"`). All mode checks in the API go through it.
- In `main.go`, the following registrations are wrapped in `if !cfg.IsStandalone()` so they do not exist in standalone:
  - `protected`: `GET /plan`, `POST /plan/upgrade`, `GET /referral`, `GET /referral/rewards`, `POST /referral/share`, `GET /billing`, `POST /billing/checkout`, `POST /billing/portal`, `POST /billing/cancel`, `GET /billing/invoices`, `POST /billing/exit-survey`;
  - `admin`: `GET/POST /plans`, `PUT /plans/:id`, `POST /users/:userId/plan`, `GET /billing/overview`, `GET /exit-surveys`, `GET /exit-surveys/stats`, `GET /referrals`, `GET /crm/pipeline`, `GET /crm/health-scores`, `POST/GET /crm/customers/:id/notes`, `PUT /crm/customers/:id/tags`.

  The handlers, services and repositories behind them are untouched; only the registration is conditional. In SaaS mode nothing changes.
- A test starts the router in standalone mode and asserts that no registered route path starts with `/api/v1/billing`, `/api/v1/plan`, `/api/v1/referral`, `/api/v1/admin/plans`, `/api/v1/admin/billing`, `/api/v1/admin/exit-surveys`, `/api/v1/admin/referrals` or `/api/v1/admin/crm`, and a second run in SaaS mode asserts they are present. The route list comes from the same enumeration `ZENITH_DUMP_ROUTES` uses.
- Registration: `AuthHandler` gets `SetRegistrationDisabled(bool)`; `main.go` sets it from `cfg.IsStandalone()`. When disabled, `Register` returns `403` with `{"error":"registration is disabled on self-hosted installs; the administrator account is created by the installer"}` before parsing the body. The route stays registered so the answer is a clear message, not a 404. `/auth/verify-email` and `/auth/resend-verification` stay as they are (they cannot be reached without a registration).
- The OAuth start routes stay registered; the web stops linking to them in standalone (§3). Removing them is not needed for the exit criteria and would touch SaaS auth code.

## 3. Web: login page and SaaS leftovers (F-25, F-23, F-24, F-03)

**Today:** `apps/web/src/lib/runtime-env.ts` exports `IS_STANDALONE` and `edition.ts` exports `SHOW_PLANS = !IS_STANDALONE`, but the login page, the Docs page, the Overview plan tile, the deploy wizard and the Databases header do not use them. `tsc --noEmit` fails on three fixtures in `src/hooks/__tests__/use-auth.test.ts` that pass `mfa_required: false` together with `mfa_token` to a `LoginResponse` union. CI runs `pnpm turbo lint` but not `tsc`.

**Design:**

- **Login page** (`src/app/login/page.tsx`). When `IS_STANDALONE`:
  - the tagline becomes "Self-hosted PaaS" (no Hetzner, no Kubernetes);
  - the Google and GitHub buttons and the divider above them are not rendered;
  - the "Create an account" / "Sign in" toggle is not rendered and `mode` can only be `login` or `mfa`; `?mode=register` is ignored;
  - the MFA step is unchanged.
  When not standalone the page is unchanged. A vitest renders the page with `IS_STANDALONE` mocked both ways and asserts the presence or absence of those elements, in the style of `src/lib/__tests__/edition.test.ts`.
- **Docs page** (`src/app/docs/page.tsx`). In standalone the page documents the Compose edition only, in this order: Prerequisites (the admin account from the installer; install `zen` with the release one-liner), 1 Create an app (image or compose file), 2 Add a database and attach it, 3 Environment variables and secrets, 4 Custom domain and HTTPS (A record to the server's IP, status pending → active), 5 Logs and rollback, 6 Backup and upgrade (`zen backup`, `zen upgrade`). The "Enable Auth", "Add Storage", "Deploy via Git Push" and "Complete Example" sections and the API Quick Reference with `api.freezenith.com` are not rendered in standalone; API examples that remain use the dashboard's own origin. The SaaS rendering is unchanged.
- **Overview** (`src/app/page.tsx`): the Plan `StatCard` is rendered only when `SHOW_PLANS`, and the plan query is not issued when `SHOW_PLANS` is false (after §2 it would 404). The "Legacy Apps" tile is labelled "Apps".
- **Deploy wizard** (`src/components/deploy-wizard.tsx`): the "Pro plan / Free plan per-app limit" sentence is rendered only when `SHOW_PLANS`.
- **Databases** (`src/app/databases/page.tsx`): the header counts gain MySQL (`pgCount`, `mysqlCount`, `redisCount`).
- **F-03:** the three fixtures in `use-auth.test.ts` are changed to values that satisfy `LoginResponse` (an MFA-required response carries `mfa_token` and no access token; a plain login carries the tokens and no `mfa_token`). No change to the hook.
- **CI:** `ci.yml` gets a "Typecheck frontend" step after "Lint frontend": `pnpm --filter @zenith/web exec tsc --noEmit`.

## 4. CLI: `zen login`, `zen logout`, `zen status` (F-09, F-28)

**Today:** `zen login` asks for an endpoint (default `https://api.freezenith.com`) and a token, and saves both to `~/.zen/config.yaml`. The API issues 1-hour access tokens and 7-day refresh tokens, and already has `POST /auth/login`, `POST /auth/login/mfa`, `POST /auth/refresh` and `POST /auth/logout`; `cli/internal/api.Client.Login` exists but discards the refresh token. `zen status` fetches apps from `/api/v1/projects/:id/apps`, the legacy CRD route, which returns 404 on Compose, and prints Plan and a Cost Estimate placeholder. Help texts mention Zenith Cloud and Hetzner.

**Design:**

- **Config** (`cli/internal/config`): `Token` stays the access token. New fields `RefreshToken string yaml:"refresh_token,omitempty"` and `TokenExpiresAt time.Time yaml:"token_expires_at,omitempty"`. The file keeps mode 0600.
- **`zen login`:**
  - flags `--endpoint`, `--email`, `--password` (also read from `ZEN_PASSWORD` for scripts), and the existing `--token` for a pasted token;
  - prompts for whatever is missing: endpoint (default: the saved one; there is no freezenith.com default any more; empty is an error), email, password (hidden input);
  - calls `POST /api/v1/auth/login`; if the response says `mfa_required`, prompts for the code and calls `POST /api/v1/auth/login/mfa`;
  - stores access token, refresh token and `now + expires_in`; prints `Logged in to <endpoint> as <email>`;
  - with `--token` it stores only the token (no refresh), as today.
- **Refresh** (`cli/internal/api.Client`): before a request, if a refresh token is stored and the access token expires in under 60 s, the client calls `POST /auth/refresh`, stores the returned tokens (whatever the server returns, rotated or not) and saves the config. On a 401 with a refresh token present it refreshes once and retries the request once. If the refresh fails the command errors with `session expired — run zen login`. Deploy tokens (`DeployToken` header) are never refreshed.
- **`zen logout`:** `POST /auth/logout` (best effort) and clears `Token`, `RefreshToken`, `TokenExpiresAt`.
- **`zen status`** asks `GET /api/v1/version`, which gains a `mode` field (`standalone` or `saas`). In standalone it prints: Server (endpoint, version), Apps (name, status, URL) from `GET /api/v1/apps` filtered to the current project when one is selected, Databases (name, engine, status) from `GET /api/v1/databases`. It prints no Project/Region/Plan header and no Cost section. In SaaS mode the command is unchanged except that apps come from `/api/v1/apps` too (the legacy route is gone from Compose and not needed on k8s either).
- **Help text:** root, `login` and `status` describe Zenith, not Zenith Cloud or Hetzner. `config.Region`'s comment is reworded; the field stays.
- **Version check (item 6):** the target-version regexes in `cli/cmd/upgrade` and `cli/internal/composeops` accept `-` inside the prerelease part (semver `[0-9A-Za-z.-]`). The image workflow already accepts it.

## 5. Custom domains: DNS verification and `tls_ready` (F-26)

**Today:** `POST /apps/:id/domains` saves the domain as `active` and refreshes the Traefik labels; `tls_ready` is written `false` and never updated. There is no DNS check, so a typo or a missing record looks identical to a working domain. The `DomainStatus` enum already has `pending`, `verified`, `active`, `failed`.

**Design:**

- **Data:** `custom_domains` gains `status_message text not null default ''` and `last_checked_at timestamptz null` (next migration number). `entities.CustomDomain` and the API response carry both. `ports.DomainRepository` gains `UpdateDomainCheck(ctx, id string, status DomainStatus, tlsReady bool, message string, checkedAt time.Time) error`; the memory and postgres adapters implement it. `UpdateDomainStatus` stays for existing callers.
- **Add flow:** the domain is saved with status `pending`, `tls_ready=false`, message `waiting for DNS check`, and the Traefik labels are applied at once (as today), so Let's Encrypt can issue as soon as DNS resolves.
- **State machine** (the only transitions the checker makes):
  - `pending`: DNS does not point here. Message `DNS: no A/AAAA record for <domain>; create an A record to <ip>` or `DNS: <domain> points to <x>; expected <ip>`.
  - `verified`: DNS is right, certificate not yet issued. Message `DNS ok; waiting for the certificate`.
  - `active` + `tls_ready=true`: Traefik serves a valid certificate for the domain. Message empty.
  - A domain can move back from `active` to `pending` if DNS later changes away; `tls_ready` is then set false. `failed` is not used.
- **Checker** (`services/domain_checker.go`, standalone only, started in `main.go` after the reconciler):
  - a single goroutine; every 30 s it lists domains that are not (`active` and `tls_ready`) plus active ones not checked in the last 10 min, and checks them one at a time with a 5 s timeout per step;
  - expected IPs = A/AAAA of `ZENITH_DOMAIN`, resolved with the system resolver and cached 10 min; if that resolution fails the round is skipped with a warning (the installer already requires the dashboard domain to resolve);
  - DNS step: `net.DefaultResolver.LookupIPAddr(domain)` (follows CNAMEs); the result must intersect the expected IPs;
  - TLS step: a TLS dial to `traefik:443` on `zenith_edge` with `ServerName=<domain>` and the system roots. A verified chain for the domain → `active`. A handshake that fails verification (Traefik's default certificate while ACME is pending) → `verified`. The dial itself prompts Traefik to request the certificate, so a `verified` domain normally becomes `active` on the next round;
  - backoff: domains older than one hour are checked every 5 min, older than a day every 30 min; checking never stops until the domain is deleted;
  - every outcome writes `status`, `tls_ready`, `status_message`, `last_checked_at` through `UpdateDomainCheck`; errors are logged with the domain and never abort the loop.
- **Check now:** `POST /apps/:id/domains/:domainId/verify` runs one check for that domain inline and returns the updated domain (owner-checked like the other domain routes).
- **Web:** the custom-domain UI lives on the app page's Domains tab (`src/app/apps/[name]/page.tsx`): each row shows the status badge (`pending` / `verified` / `active`), `status_message`, the last check time and a "Check now" button that calls verify. The Networking page (`src/app/networking/page.tsx`) lists app subdomains and the user's real custom domains from `GET /api/v1/domains` with the same status cell; its mock "Add Domain" modal and "Load Balancers" card, which were never backed by anything, are removed. The message already contains the IP, so no extra endpoint is needed.
- **Routing:** every custom domain is routed by Traefik whatever its status (the deployer no longer filters on `active`/`verified`). Traefik requests an ACME certificate only when its configuration changes, so the checker refreshes the app's routes (a redeploy) when a domain enters `verified` and again at most every 10 minutes while it stays `verified`.
- **Resolver errors:** only NXDOMAIN (or an empty answer) means "no record"; any other lookup error leaves the status and `tls_ready` unchanged and stores `DNS check failed: <err>`.
- The DNS and TLS steps are behind two small interfaces (resolver, dialer) so tests inject fakes.

## 6. Managed-database env vars on env-v2 (F-26)

**Today:** attaching, resetting or deleting a managed database writes or deletes the app's connection variable through `AppRepository.SetEnvVars` / `DeleteEnvVar` (the legacy table) at eleven call sites in `services/database.go` and `handlers/databases_v2.go`. The Environment tab reads env-v2 (`/apps/:id/env-v2`), so the variable is invisible there. `DockerDeployer` merges legacy vars first and then env-v2 vars for the app's environment, so deploys do see it.

**Design:**

- A helper in the database service, `managedEnv`, with `Set(ctx, appID, key, value, dbID)` and `Clear(ctx, appID, dbID)`:
  - `Set` upserts an `AppEnvVar{Key, Value, IsSecret: true, Source: managed_service, SourceID: dbID, EnvironmentID: <the app's default environment>}` through `EnvVarRepository`;
  - `Clear` removes the env-v2 vars of that app whose `Source` is `managed_service` and `SourceID` is that database (other databases' vars stay). If the repository has no such method, it lists and deletes by id.
- The eleven call sites use the helper instead of the legacy table. The legacy table is no longer written by database code. Reading stays: `DockerDeployer` keeps merging legacy then env-v2, so env-v2 wins when both hold the same key.
- **Migration at startup** (standalone, after the reconciler, idempotent): for every managed database with an app, if the legacy table has the engine's key (`envKeyForEngine`) and env-v2 has no `managed_service` var with that `SourceID`, the value is copied into env-v2 and the legacy row is deleted. Each copy is logged. A failure to copy keeps the legacy row (deploys still work).
- **Environment tab:** vars whose `source` is `managed_service` show a "from database" badge and no edit or delete control; the only way to remove them is to detach the database. The vars' `source` and `source_id` are already in the env-v2 response type (`lib/api.ts`).
- After `Set` or `Clear` the handler behaviour that already exists (redeploy on attach, §6.1 of the audit) is unchanged.

## 7. Release and audit

- `VERSION` becomes `0.1.0-alpha.4` on the branch. After the squash merge to `main`, a GitHub prerelease `v0.1.0-alpha.4` builds and publishes the images (as for alpha.3). The live install is upgraded with `zen upgrade --version 0.1.0-alpha.4` as part of acceptance.
- core-audit §8 ("After 2c") records every success criterion with evidence, re-runs the route dump of §1.1, and updates the findings table: F-01, F-02, F-03, F-09, F-23, F-24, F-25, F-26, F-28 fixed.
- Issues #9, #10, #11 and #17 are closed by the PR.

## 8. Testing

- **API:** route-list test for both modes (§2); handler test for the 403 registration; domain checker tests with fake resolver and dialer covering every transition in §5 including `active` → `pending`; `verify` handler test; env helper and startup-migration tests against the memory repositories (attach, detach with two databases, idempotent re-run, failure keeps legacy).
- **Web:** vitest for the login page in both modes, for the Docs page in standalone (no `freezenith.com`, no "Pro"), for the Databases header counts; `tsc --noEmit` in CI.
- **CLI:** tests against an `httptest` server for login (plain, MFA, wrong password), refresh before expiry, refresh-and-retry on 401, refresh failure message, `logout`, and `status` output in standalone and SaaS mode; version regex tests with `-` in the prerelease.
- **Live (Task "acceptance"):** the seven success criteria in §1, on the live install, after `zen upgrade` to alpha.4; the domain criterion uses a real record under `freezenith.com`.

## 9. Risks

- **Hairpin and resolver inside the container.** The TLS step dials `traefik:443` on `zenith_edge`, not the public IP, so it does not depend on hairpin NAT. DNS resolution uses Docker's embedded DNS, which forwards to the host's resolvers; a host that resolves `ZENITH_DOMAIN` to a private address (split DNS) would make every domain look wrong. The message shows the expected IP, which makes that visible; the installer's own requirement that the domain resolves publicly covers the normal case.
- **Refresh token rotation.** If the API rotates refresh tokens, a stale CLI config on a second machine is logged out; the client stores whatever `/auth/refresh` returns, so one machine always works.
- **Legacy env deletion during migration.** The legacy row is deleted only after the env-v2 write succeeded, and deploys read both tables, so there is no window without the variable.
- **Route gating and the web.** Any standalone page that still called a gated route would now 404 instead of 403; the Overview plan query is the only one found (`page.tsx`), and the vitest for the login page plus the route-list test guard the rest. The live acceptance browses every core page with the browser console open.
- **Docs page rewrite drifts from the product.** The page documents only what the Compose alpha does today; the E2E sub-project is the next place that keeps it honest.
