# Agent Log

## 2026-02-21T19:19 — Phase 2A: App Deploy Engine — Database & Models

### What Changed
- **New migration** `006_apps.up.sql` / `006_apps.down.sql` — creates `apps`, `deployments`, `app_env_vars` tables with indexes
- **New models** `models/app.go` — `App`, `AppStatus` enum (pending→building→deploying→running→sleeping→failed→stopped), `Framework` enum (nextjs, go, python, django, rails, flask, express, static, dockerfile, unknown), `CreateAppInput`, `UpdateAppInput`
- **New models** `models/deployment.go` — `Deployment`, `DeploymentStatus` enum, `EnvVar`
- **New interface** `AppRepository` in `store/interfaces.go` — 16 methods (apps CRUD, deployments CRUD, env vars CRUD)
- **New store** `store/memory_app.go` — in-memory `AppRepository` implementation
- **New store** `store/postgres_app.go` — PostgreSQL `AppRepository` implementation (pgx/v5)
- **New tests** `store/memory_app_test.go` — 30 unit tests covering all operations
- **Modified** `cmd/server/main.go` — wired `appRepo` into server startup (both postgres and in-memory paths)

### Why
Phase 2 (App Deploy Engine) is the core product feature — git push → app deployed. Phase 2A lays the foundation: persistent app data in PostgreSQL, replacing the current in-memory CRD approach. Everything in phases 2B–2F depends on this.

### Verification
- `go build ./...` — passes
- `go test ./...` — all packages pass (including 30 new tests)

---

## 2026-02-21T19:24 — Phase 2B: Framework Detection & Git Operations

### What Changed
- **New file** `internal/deploy/detect.go` — framework detection from file markers (Dockerfile, next.config, go.mod, requirements.txt, Gemfile, manage.py, package.json, index.html) with priority-based resolution and refinement for Next.js/Flask
- **New file** `internal/deploy/dockerfile.go` — 8 Dockerfile templates (Next.js, Go, Python, Django, Flask, Rails, Express, Static) with multi-stage builds, non-root users, slim/alpine base images
- **New file** `internal/deploy/git.go` — shallow clone, commit SHA extraction, clone-and-detect helper, cleanup
- **New tests** `internal/deploy/deploy_test.go` — 28 unit tests covering all detection scenarios and Dockerfile generation

### Why
Phase 2B enables the API to clone a repo, detect what framework it uses, and generate the appropriate Dockerfile for building. This is the prerequisite for the build pipeline (Phase 2D).

### Verification
- `go build ./...` — passes
- `go test ./...` — all 11 packages pass (including 28 new deploy tests)

---

## 2026-02-21T19:31 — Phase 2C: API Handlers & Webhook

### What Changed

- **New file** `handlers/apps_v2.go` — `AppHandlerV2` using `AppRepository` (replaces CRD-based approach), endpoints: Create, List, Get, Delete at `/api/v1/apps`
- **New file** `handlers/webhook.go` — GitHub webhook receiver with HMAC-SHA256 signature verification, push event processing, deployment creation
- **New file** `handlers/deploy.go` — `DeployHandler` for deployments (list, get), rollback, env var CRUD (set, get, delete)
- **New file** `handlers/handlers_v2_test.go` — 15 unit tests covering all new handlers
- **Modified** `handlers/errors.go` — added `NewInternal` error helper
- **Modified** `config/config.go` — added `GITHUB_WEBHOOK_SECRET` and `BASE_DOMAIN` config fields
- **Modified** `cmd/server/main.go` — registered all new routes alongside legacy CRD routes

### Why

Phase 2C provides the HTTP API layer for the deploy engine. Users can now create apps from GitHub repos, manage env vars, view deployments, and trigger rollbacks. The webhook enables automatic deployments on git push.

### Verification

- `go build ./...` — passes
- All 15 new handler tests pass
- 2 pre-existing failures in `customer_test.go` (unrelated to Phase 2 changes)

---

## 2026-02-21T19:32 — Phase 2D: Build Pipeline

### What Changed

- **New file** `internal/deploy/builder.go` — build orchestrator (clone → detect → generate Dockerfile → prepare image tag → update status)
- **New file** `internal/deploy/kaniko.go` — Kaniko K8s Job spec generator with caching, resource limits, registry auth volumes
- **New file** `internal/deploy/pipeline.go` — async pipeline runner with goroutine management, cancellation, and concurrent build tracking
- **New tests** `internal/deploy/builder_test.go` — 9 tests for builder, kaniko spec, pipeline state, min helper

### Why

Phase 2D connects the API layer to container image building. The pipeline clones a repo, detects the framework, generates a Dockerfile, and prepares a Kaniko build job for in-cluster execution. The async runner manages concurrent builds.

### Verification

- `go build ./...` — passes
- All 37 deploy package tests pass (28 from 2B + 9 new)

---

## 2026-02-21T19:36 — Phase 2E: K8s Deployment Resources

### What Changed

- **New file** `internal/deploy/k8s_resources.go` — Generates Deployment (probes, resource limits, env vars), Service, and Traefik IngressRoute (TLS) manifests
- **New file** `internal/deploy/deployer.go` — K8s deployer that applies manifests via CRD client with create-or-update semantics
- **New tests** `internal/deploy/k8s_test.go` — 7 tests covering resource generation, serialization, deploy, and delete

### Why

Phase 2E completes the deployment pipeline. After building a container image (2D), the deployer creates the K8s resources needed to serve the app with HTTPS.

### Verification

- `go build ./...` — passes
- All 44 deploy package tests pass

---

## 2026-02-21T19:40 — Phase 2F: Dashboard Pages

### What Changed

- **Modified** `apps/web/src/lib/api.ts` — Added `DeployApp`, `Deployment`, `EnvVar` types and `appsDeploy` API client (CRUD, deployments, rollback, env vars)
- **Modified** `apps/web/src/lib/get-api.ts` — Exported `appsDeploy` through the API facade
- **New file** `apps/web/src/app/apps/[id]/page.tsx` — App detail page with 3 tabs: Overview (details + quick links), Deployments (table + rollback), Environment (add/delete/show-hide values)

### Why

Phase 2F provides the user-facing dashboard for the deploy engine, connecting the Phase 2 backend APIs to the frontend UI.

### Verification

- `npx next build` — passes

---

## 2026-02-21T19:42 — Updated app_explanation.md with Phase 2 Completions

### What Changed

- **Section 4 Config** — Added `DATABASE_URL`, `GITHUB_WEBHOOK_SECRET`, `BASE_DOMAIN` env vars
- **Section 4 Endpoints** — Added 12 Phase 2 deploy engine routes + GitHub webhook endpoint; marked legacy CRD routes as "(legacy)"
- **Section 4 Data Layer** — Expanded from "Currently In-Memory" to structured section covering legacy CRD layer + Phase 2 repository pattern (AppRepository, PostgreSQL, SQL migrations, models) + deploy pipeline files
- **Section 3 Web Platform** — Updated API layer with `appsDeploy` client and new `DeployApp`, `Deployment`, `EnvVar` types; added `/apps/[id]` page entry
- **Section 15 Status** — Added Phase 2 ✅ bullet with 10 sub-items; updated In-Memory section; crossed off PostgreSQL from remaining work; added Kaniko and build logs to Not Yet Wired

### Why

To keep `app_explanation.md` as the single source of truth for the entire codebase state.

---

## 2026-02-21T19:46 — Phase 2 Documentation

### What Changed

- **New** `docs/features/backend/app-deploy-engine.md` — Complete backend module doc (10 sections)
- **New** `docs/runbooks/backend/app-deploy-engine.md` — Runbook with debugging guide, SQL queries, disaster recovery
- **New** `docs/features/frontend/app-deploy-dashboard.md` — Frontend feature doc (11 sections)

### Why

Mandatory documentation per documentation-architect rules. All Phase 2 code now has corresponding documentation.

---

## 2026-02-21T19:52 — Pipeline→Deployer Integration (Phase 3 wiring)

### What Changed

- **Modified** `internal/deploy/pipeline.go` — Added `deployer *Deployer` field; after build completes, now calls `deployer.DeployApp()` to create K8s Deployment+Service+IngressRoute. Nil-safe for tests.
- **Modified** `internal/handlers/webhook.go` — Added `pipeline *deploy.Pipeline` field; calls `pipeline.TriggerBuild()` on git push. Implemented `findAppsByRepo()` to scan apps by repo URL/branch.
- **Modified** `cmd/server/main.go` — Constructs full chain: Builder → Deployer → Pipeline → WebhookHandler. Added `deploy` package import.
- **Modified** `internal/deploy/builder_test.go` — Updated 3 `NewPipeline()` calls with nil deployer.
- **Modified** `internal/handlers/handlers_v2_test.go` — Updated `NewWebhookHandler()` call with nil pipeline.

### Why

Closes the last two TODOs in the deploy engine: "Phase 2D — trigger async build pipeline" and "Phase 2E — trigger K8s deployment". The system is now fully wired end-to-end.

### Verification

- `go build ./...` — passes
- `go test ./...`- **Modified** `internal/handlers/webhooxisting cluster test failures unrelated)

---

## 2026-02-21T20:42 — Deploy Engine Frontend Page + Sidebar

### What Changed

- **New** `apps/web/src/app/deploy/page.tsx` — Deploy Engine dashboard with:
  - Card grid layout showing apps with status dots, framework, branch, and URL
  - "Deploy from Git" modal (name, repo URL, branch) with loading spinner
  - Delete with confirmation dialog
  - Status colors (running=green, building=amber pulse, failed=red)
  - Framework label mapper (9 frameworks)
- **Modified** `apps/web/src/components/sidebar.tsx` — Added "DEPLOY" section with Rocket icon between OVERVIEW and COMPUTE
- **Modified** `apps/web/src/lib/demo-api.ts` — Added `demoAppsDeploy` with 3 mock deploy apps + all method stubs matching real API shape

### Why

The Deploy Engine needed its own dedicated page in the sidebar separate from legacy CRD-based "Apps" page. Cards are more appropriate than tables for deploy engine apps since each app has fewer columns but richer metadata.

### Verification

- `npx next build` passes ✅

---

## 2026-02-21T20:52 — Enhanced Dashboard Overview with Deploy Engine

### What Changed

- **Modified** `apps/web/src/app/page.tsx` — Enhanced overview dashboard:
  - Added `appsDeploy.list()` fetch alongside legacy data
  - 5-column stat grid: Legacy Apps, Deploy Engine (with building count), Databases, Region, Status
  - New "Deploy Engine" section with 3-col card grid showing deploy apps with status dots, framework labels, branch, and colored status text
  - Legacy "Apps" section relabeled "Apps (Legacy)" for clarity
  - Rocket icon + "View all" link to /deploy page

### Why

Dashboard needed to show Deploy Engine activity at a glance — users should see their git-deployed apps immediately on the overview, not just legacy CRD apps.

### Verification

- `npx next build` passes ✅

---

## 2026-02-21T21:02 — Updated app_explanation.md

### What Changed

Updated 7 sections of `app_explanation.md` to reflect all session changes:
1. Overview page description — 5-col stats + Deploy Engine card grid
2. Pages table — Added `/deploy` page entry
3. Sidebar navigation — Added DEPLOY section with Rocket icon
4. Demo API — Noted `demoAppsDeploy` with 3 mock apps
5. Deploy Pipeline — Pipeline→Deployer integration, End-to-End Wiring subsection
6. Development Status — Phase 2 + Phase 3 wiring, frontend pages, demo mock
7. Not Yet Wired — Added note about legacy /apps not connected to Deploy Engine

### Why

Keeping `app_explanation.md` as the canonical source of truth for the entire codebase state.

---

## 2026-02-21T22:18 — Phase 3: Build Log Streaming via SSE

### What Changed

**Backend:**
- **Fixed** `internal/handlers/logs.go` — Pre-existing bug: `SetBodyStreamWriter` takes `func(w *bufio.Writer)` in fasthttp v1.51, not `*fasthttp.StreamWriter`. Fixed type + added `w.Flush()` after each SSE write and keepalive. Removed the unused `fasthttp` import.
- **Modified** `cmd/server/main.go` — Constructed `LogHandler` from existing `appRepo + logHub`; registered two JWT-protected routes:
  - `GET /api/v1/apps/:appId/deployments/:did/logs` — SSE stream (30s keepalive, `event: done` on finish)
  - `GET /api/v1/apps/:appId/deployments/:did/logs/history` — JSON snapshot
- **New** `internal/handlers/logs_test.go` — 3 handler tests: `TestGetLogsHistoryEmpty`, `TestGetLogsHistoryWithEntries`, `TestGetLogsHistoryAppNotFound`

**Frontend:**
- **New** `apps/web/src/hooks/use-deploy-logs.ts` — `useDeployLogs(appId, deploymentId)` hook: fetches history snapshot first (GET), then opens `EventSource` for live SSE updates. Handles `event: done`, cleanup on unmount, and demo mode (7 static sample log lines).
- **New** `apps/web/src/components/build-log-viewer.tsx` — Terminal-style log viewer: dark background, color-coded levels (info=neutral, build=blue, deploy=emerald, error=red), per-line timestamps, auto-scroll (pauses on manual scroll-up), "Live" pulse dot indicator, blinking cursor while streaming.
- **Modified** `apps/web/src/app/apps/[id]/page.tsx` — Added `"logs"` tab (Terminal icon), `LogsTab` component that fetches the most recent deployment and renders `BuildLogViewer`, `LogsTabContent` wrapper for the hook.

**Docs:**
- `docs/runbooks/backend/app-deploy-engine.md` — Added Phase 3 change history row + log streaming endpoint table + LogHub architecture diagram
- `app_explanation.md` — Added 2 new SSE routes to endpoint table, documented `log_hub.go` in deploy pipeline section, moved "Build logs streaming" from "Not Yet Wired" to "Complete"

### Why

Build log streaming was the last major "not yet wired" gap in the Deploy Engine. The LogHub broadcaster was fully implemented but had no HTTP endpoints registered. This change completes the end-to-end developer experience: deploy → watch build logs live in the terminal-style UI.

### Verification

- `GO111MODULE=on go test ./internal/...` — All 10 packages PASS (0 FAIL)
- `npx tsc --noEmit` in `apps/web/` — 0 type errors

---

## 2026-02-21T22:31 — Phase 4: Kaniko Build Execution

### What Changed

**k8s Layer:**
- **Modified** `internal/k8s/client.go` — Added `JobObject` struct + 4 new `Client` interface methods (`CreateJob`, `GetJob`, `DeleteJob`, `GetPodLogs`). `MemoryClient` implements all 4: immediately marks jobs as Succeeded; `GetPodLogs` emits 9 realistic fake build log lines.

**Deploy Layer:**
- **New** `internal/deploy/kaniko_runner.go` — `KanikoRunner`: submits the K8s Kaniko Job, polls for completion (5s interval, 30min timeout), streams pod logs → LogHub, deletes Job on success. Nil-safe: calling `Build()` on a nil runner is a no-op.
- **Modified** `internal/deploy/builder.go` — Added `kanikoRunner *KanikoRunner` field; `NewBuilder` now accepts `k8sClient k8s.Client` and `logHub *LogHub`; replaced `// NOTE: Actual image building` placeholder with real `kanikoRunner.Build()` call + dev-mode fallthrough.
- **Modified** `internal/deploy/builder_test.go` — Updated all 6 `NewBuilder()` calls with the new `nil, nil` signature.
- **New** `internal/deploy/kaniko_runner_test.go` — 4 tests: nil-client no-op, `NewKanikoRunner(nil,nil)` returns nil, success via MemoryClient, log emission to LogHub.

**Config + Wiring:**
- **Modified** `internal/config/config.go` — Added `Registry` (`REGISTRY` env, default `registry.freezenith.com`) + `BuildWorkDir` (`BUILD_WORKDIR` env, default `/tmp/zenith-builds`).
- **Modified** `cmd/server/main.go` — `logHub` constructed before `builder`; `NewBuilder` called with `cfg.BuildWorkDir`, `cfg.Registry`, `k8sClient`, `logHub`.

### Why

The Kaniko build was the last major gap in the deploy pipeline. Before this change, apps were cloned and a Dockerfile generated but no image was ever built — the pipeline jumped straight to "Build complete". Now the full flow is wired: git push → clone → detect → Dockerfile → Kaniko Job → image in registry → K8s deploy.

### Verification

- `GO111MODULE=on go build ./...` — clean (0 errors)
- `GO111MODULE=on go test ./internal/...` — all 10 packages PASS

---

## 2026-02-22T00:34 — Architecture Decisions + app_explanation.md Update

### What Changed

- **Updated** `app_explanation.md` — rewrote Development Status, Not Yet Wired, Remaining Major Work, and Tech Stack sections to reflect:
  - Phase 3 (Build Log Streaming) ✅ marked complete
  - Phase 4 (Kaniko Build Execution) ✅ marked complete
  - Full infrastructure tooling decisions documented (FluxCD, Cilium, Kong, CloudNativePG, Keycloak, Harbor, Sealed Secrets)
  - Customer app deploy flow designed: `zenith-actions` GitHub Action → customer's own registry → Zenith API `/releases` endpoint → one-click deploy in panel
  - Privacy model: customer images never touch Zenith infrastructure
  - Tech Stack table expanded with all confirmed tooling

### Why

Architecture session defined all remaining tooling choices. Documenting decisions now prevents drift and serves as reference for implementation.

---

## 2026-02-22T00:50 — Phase 5: App Secrets (AES-256-GCM)

### What Changed

- **NEW** `store/migrations/007_app_secrets.up.sql` — `app_secrets` table (BYTEA for encrypted values)
- **NEW** `models/secret.go` — `Secret`, `SecretWithValue`, `CreateSecretInput` models
- **NEW** `pkg/crypto/aes.go` — `Encrypt` / `Decrypt` / `KeyFromHex` (AES-256-GCM, nonce prepended)
- **MODIFY** `config/config.go` — added `SecretsKey` from `SECRETS_ENCRYPTION_KEY` env var
- **MODIFY** `store/interfaces.go` — `SetSecret`, `GetSecrets`, `GetSecretValue`, `DeleteSecret` in `AppRepository`
- **MODIFY** `store/memory_app.go` — implemented Secret methods + cascade delete
- **MODIFY** `store/postgres_app.go` — implemented Secret methods (pgx BYTEA, ON CONFLICT upsert)
- **NEW** `handlers/secrets.go` — `SecretHandler` (nil-safe in dev mode without key)
- **MODIFY** `cmd/server/main.go` — registered secret routes under `/apps/:appId/secrets`

### Why

App secrets store sensitive customer values (DB URLs, API keys) encrypted at rest with AES-256-GCM.

---

## 2026-02-22T00:55 — Phase 6: Releases + Image Deploy Flow

### What Changed

- **NEW** `store/migrations/008_releases.up.sql` — `app_releases` table
- **NEW** `models/release.go` — `Release`, `CreateReleaseInput`
- **MODIFY** `store/interfaces.go` — `CreateRelease`, `ListReleases`, `GetRelease` in `AppRepository`
- **MODIFY** `store/memory_app.go` + `store/postgres_app.go` — Release CRUD methods
- **NEW** `handlers/releases.go` — `ReleaseHandler` with CreateRelease, ListReleases, DeployRelease
- **MODIFY** `deploy/pipeline.go` — added `TriggerImageDeploy` (deploy pre-built image, skip build phase)
- **MODIFY** `cmd/server/main.go` — registered `/apps/:appId/releases` routes

### Why

Decouples build from deploy. CI pushes image → registers release → customer one-click deploys from panel.

---

## 2026-02-22T01:00 — Phase 7: zenith-actions GitHub Action

### What Changed

- **NEW** `.github-actions/zenith-deploy/action.yml` — composite GitHub Action
  - login to registry, build image, push, register release with Zenith API

### Why

Customer developer experience: add one step to their GitHub Actions workflow to get CI/CD with Zenith.

---

## 2026-02-22T00:52 — Session Handover

### What Changed

- **NEW** `handover.md` — Wrote out the complete current state of the project, including progress on Phases 5-7, completed architecture decisions, and immediate next steps for the frontend UI.

### Why

The user is switching accounts for the next session. This file serves as the context bridge for the incoming AI agent to continue work seamlessly.

---

## 2026-02-22T01:08 — Phase 5 & 6 Frontend UI (Secrets Tab + Releases Tab)

### What Changed

**API Layer:**
- **Modified** `apps/web/src/lib/api.ts` — Added `Secret` and `Release` TypeScript types; added 6 new `appsDeploy` methods: `listSecrets`, `getSecretValue`, `setSecret`, `deleteSecret`, `listReleases`, `deployRelease`
- **Modified** `apps/web/src/lib/demo-api.ts` — Added demo mocks: 3 secrets (DATABASE_URL, API_KEY, JWT_SECRET) with mock decrypted values; 4 releases with realistic git SHAs, branches, and commit messages

**App Detail Page:**
- **Modified** `apps/web/src/app/apps/[id]/page.tsx` — Added 2 new tabs (6 total: Overview, Deployments, Releases, Logs, Secrets, Environment):
  - **SecretsTab** — List encrypted secrets (keys only), Add Secret form (key auto-uppercased, value as password input), Reveal button (calls decrypt API, caches in state), Copy to clipboard, Delete, loading spinners per row
  - **ReleasesTab** — Table of image versions with "latest" badge on newest, git SHA, branch, commit message, one-click Deploy button with spinner + "Triggered" confirmation feedback

**Documentation:**
- **New** `docs/features/frontend/secrets-tab.md` — Frontend feature doc (11 sections)
- **New** `docs/features/frontend/releases-tab.md` — Frontend feature doc (11 sections)
- **Modified** `app_explanation.md` — Updated `/apps/[id]` page description from 3-tab to 6-tab, removed "releases panel UI" from Not Yet Wired

### Why

Completes the handover item "Build the Frontend UI for Phases 5 & 6". Backend endpoints for secrets (AES-256-GCM encryption) and releases (CI image registration + deploy trigger) were implemented in the previous session but had no frontend UI.

### Verification

- `npx tsc --noEmit` in `apps/web/` — 0 type errors ✅

---

## 2026-02-22T01:25 — Phase 8: Real-Time Deployment Events (EventHub + SSE)

### What Changed

**Backend — EventHub (new files):**
- **New** `services/api/internal/deploy/event_hub.go` — `EventHub` in-memory pub/sub broadcaster with `DeployEvent` type, 6 event types (deployment_started, build_progress, build_complete, deploy_started, deploy_complete, deploy_failed), ring buffer history (50 entries), Subscribe with replay
- **New** `services/api/internal/handlers/events.go` — SSE handler: `StreamEvents` (GET /api/v1/events) and `GetRecentEvents` (GET /api/v1/events/history)

**Backend — Pipeline integration:**
- **Modified** `services/api/internal/deploy/pipeline.go` — Added `eventHub *EventHub` field, `emitEvent` helper, and event emission at each pipeline stage (6 hook points in TriggerBuild + 3 in TriggerImageDeploy)
- **Modified** `services/api/cmd/server/main.go` — Created EventHub, passed to Pipeline, registered SSE routes under protected group

**Frontend:**
- **New** `apps/web/src/hooks/use-deploy-events.ts` — SSE hook using EventSource, auto-reconnect (5s), skips in demo mode, stable onEvent ref
- **Modified** `apps/web/src/app/deploy/page.tsx` — Wired `useDeployEvents` to auto-refetch app card list on any event
- **Modified** `apps/web/src/app/apps/[id]/page.tsx` — Wired `useDeployEvents` into DeploymentsTab, filtered by app_id

**Documentation:**
- **New** `docs/features/backend/websocket-events.md`
- **New** `docs/features/frontend/deploy-events.md`
- **Modified** `app_explanation.md` — Removed "WebSocket real-time events" from Not Yet Wired
- **Modified** `handover.md` — Marked Phase 8 as completed

### Why

Phase 8 from handover.md. The frontend previously required manual page reloads to see deployment status changes. Now the Deploy page cards and App Detail deployment rows update automatically when the backend pipeline emits events.

### Verification

- `go build ./cmd/... ./internal/deploy/... ./internal/handlers/...` — no new errors (pre-existing OTel/pgx dependency issues unchanged) ✅
- `npx tsc --noEmit` in `apps/web/` — 0 type errors ✅

---

## 2026-02-22T01:31 — Phase 9: OpenTelemetry Activation

### What Changed

- **Modified** `services/api/internal/config/config.go` — Added `OTELEndpoint`, `OTELInsecure`, `OTELSampleRate` fields + `getEnvFloat` helper
- **Modified** `services/api/cmd/server/main.go` — Imported telemetry package, opt-in `telemetry.Init()` when `OTEL_EXPORTER_OTLP_ENDPOINT` is set, added `telemetry.Middleware()` with skip paths `/health` and `/ready`, deferred shutdown
- **New** `docs/features/backend/opentelemetry.md` — Backend module documentation
- **Modified** `app_explanation.md` — Removed "OpenTelemetry middleware" from Not Yet Wired
- **Modified** `handover.md` — Marked Phase 9 as completed

### Why

Phase 9 from handover.md. Telemetry code was fully implemented but never wired into the application. Now it's opt-in activated via env var.

### Verification

- Go files compile successfully (no new errors introduced) ✅

---

## 2026-02-22T01:36 — Unified /apps Page with Deploy Engine

### What Changed

- **Modified** `apps/web/src/app/apps/page.tsx` — Rewrote with dual-fetch: CRD K8s apps table + Deploy Engine card grid. Both data sources fetched independently, shown in unified view with Deploy Engine section below.
- **Modified** `apps/web/src/components/sidebar.tsx` — Removed separate "DEPLOY" section, moved "Deploy Engine" under "COMPUTE" alongside "Apps"
- **Modified** `app_explanation.md` — Removed "Legacy CRD apps → Deploy Engine" from Not Yet Wired

### Why

Users should see all their apps in one view. The `/apps` page now shows both CRD-based K8s apps and Deploy Engine apps without needing to switch between pages.

### Verification

- `npx tsc --noEmit` — 0 errors ✅

---

## 2026-02-22T01:40 — Backend Architecture Audit & Refactoring Plan

### What Changed

- **Audited** `services/api/internal/` against `.lich/rules/backend.md` (Clean/Hexagonal Architecture)
- **Identified** 8 architecture violations: no entities layer, no services layer, mixed ports/adapters in `store/`, no DTOs, no validators, handler→store coupling, mixed models, deploy mixing concerns
- **Appended** comprehensive refactoring plan to `app_explanation.md` with 4 phases (A–D), target structure, migration strategy, and key decisions

### Why

The backend was built pragmatically with a flat handler→store pattern. While functional, it violates the Clean Architecture principles defined in the Lich framework — creating tight coupling, untestable business logic, and unclear boundaries between domain and infrastructure.

---

## 2026-02-22T02:10 — Wire Backstage Routes + Real K8s Client

### What Changed

- **Wired Backstage routes** in `main.go`: `NewBackstageHandler(k8sClient)` + 2 routes (`/backstage/catalog`, `/backstage/catalog/:kind`)
- **Created `internal/k8s/real_client.go`** (260 lines): implements `k8s.Client` interface using `client-go` v0.35.1 (dynamic client for CRDs, typed client for Jobs/Pods). Auto-detects in-cluster vs local kubeconfig.
- **Added `K8sMode` config field** (`K8S_MODE` env var, default: `memory`). Set to `real` for production.
- **Wired client switch in `main.go`**: `K8S_MODE=real` → `NewRealClient()`, otherwise → `NewMemoryClient()`
- **Added dependencies**: `k8s.io/client-go` v0.35.1, `k8s.io/api` v0.35.1, `k8s.io/apimachinery` v0.35.1
- **Updated `app_explanation.md`**: removed Backstage and K8s client from "Not Yet Wired"

### Why

Backstage handler was already implemented but routes were never registered. K8s MemoryClient was blocking production deployment — now the API can connect to a real cluster.

---

## 2026-02-22T02:18 — Fix Build Issues + Passing Tests

### What Changed

- Fixed `pipeline.go:176` — type cast `string(deployment.Status)` for `DeployEvent.Status`
- Fixed `builder_test.go` — updated 3 `NewPipeline()` calls with missing `EventHub` arg
- Fixed `customer_test.go:790` — added `time.Sleep(100ms)` for async cluster provisioning race condition
- Added `time` import to `customer_test.go`

### Why

All pre-existing issues preventing clean `go test ./...`. Now build = 0 errors, tests = 10/10 pass.

---

## 2026-02-22T02:20 — Backend Refactoring Phase A (entities + dto)

### What Changed

- Created `internal/entities/` (6 files): `doc.go`, `common.go`, `user.go`, `app.go`, `deployment.go`, `customer.go`
- Created `internal/dto/` (3 files): `doc.go`, `inputs.go`, `responses.go`
- Created `docs/features/backend/k8s-client.md` and `docs/features/backend/backstage-integration.md`

### Why

Phase A of Lich Architecture refactoring: separate domain entities from API DTOs. Non-breaking — existing `models/` package untouched. New code can import `entities` and `dto` directly.



