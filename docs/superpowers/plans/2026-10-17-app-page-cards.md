# App page without tabs — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development. Steps use checkbox syntax.

**Goal:** replace the ten-tab app page with one page of cards (spec: docs/superpowers/specs/2026-10-17-app-page-cards-design.md).
**Architecture:** `page.tsx` keeps shell + header + banners + grid; each card is its own file with its own data fetching; the old tab code is the source to extract from and is deleted at the end.
**Tech:** Next.js 15 app router, React, Tailwind, lucide-react, vitest + testing-library (jsdom), `@/lib/get-api`.
**Spec:** docs/superpowers/specs/2026-10-17-app-page-cards-design.md

## Global Constraints
- No new dependencies. `pnpm exec tsc --noEmit` and `pnpm lint` clean. Existing tests keep passing (`pnpm test`).
- Cards use the existing API client functions only (no API changes); names in the spec §2.
- Card chrome and heading classes from spec §3; mobile: cards stack (`grid-cols-1 md:grid-cols-2`).
- Keep `useDeployEvents` refresh for deployments and the `pendingRestart` → `envVarsV2.apply` banner.
- Nothing from the Logs or Auth tabs survives on the page.

## Review Focus
1. A worker/cron app (no `url`, no subdomain): no Domains card, no Open button, no Health card; Source shows command/cron.
2. An image-deployed app with no `repo_url`: Source shows the image line, not an empty repository link.
3. Environment with both env vars and secret-store entries whose keys collide: both rows render, tagged, no crash; reveal fetches the secret value once.
4. Deployments list empty + releases present: Releases still deployable; rollback hidden for the active deployment.
5. API error on one card (e.g. buckets 500): that card shows `ErrorState` with retry; the rest of the page renders.

## Lanes
| Task | Files | Lane | After |
|---|---|---|---|
| 1 | page.tsx (new layout + header + banners), source-card.tsx, health-card.tsx, diagnostics.ts, legacy-tabs.tsx (old file moved, not imported), stub card files, page.test.tsx, source-card.test.tsx | foundation | — |
| 2 | deployments-card.tsx + test | A | 1 |
| 3 | environment-card.tsx + test | B | 1 |
| 4 | resources-card.tsx + test | C | 1 |
| 5 | domains-card.tsx + test | D | 1 |
| 6 | delete legacy-tabs.tsx, tsc/lint/test, UI acceptance, docs | integration | 2–5 |

### Task 1: Foundation — new page layout, header, Source and Health cards, stubs
- [ ] `git mv 'src/app/apps/[name]/page.tsx' 'src/app/apps/[name]/legacy-tabs.tsx'`; add `// legacy — deleted in Task 6, not imported` at its top; make it not type-check as a page (rename default export to `LegacyAppPage`).
- [ ] New `page.tsx`: `"use client"`; loads the app with `useApi(() => appsDeploy.get(appId))` exactly as the legacy shell did (same loading skeleton `AppDetailSkeleton` or `PageWithTableSkeleton`, same `ErrorState`); header per spec §2.1 (copy the legacy header markup lines ~107-154, drop the environment pills, add `Open` and `Logs` buttons: `Link href={`/logs?app=${app.id}`}`); banners (failed-pod diagnostic via `monitoring.getPods` + `getDiagnosticHint` moved to `diagnostics.ts`; sleeping banner); grid rendering `<SourceCard app>`, `<HealthCard app>` (web + health_status only), `<DeploymentsCard app>`, `<EnvironmentCard app>`, `<ResourcesCard app>`, `<DomainsCard app>` (web only).
- [ ] `source-card.tsx` from the legacy "App Details" + "Quick Links" (spec §2.2). `health-card.tsx` from the legacy Health Check card.
- [ ] Stub files `deployments-card.tsx`, `environment-card.tsx`, `resources-card.tsx`, `domains-card.tsx`: `export function XCard({ app }: { app: DeployApp }) { return null; }` with a `// Task N` comment — these are the lane contracts.
- [ ] Tests: `__tests__/page.test.tsx` (web app: header name, Logs link href, Source and Health headings present, `screen.queryAllByRole("tab")` empty and no "Overview"/"Secrets"/"Releases" buttons) and `__tests__/source-card.test.tsx` (git app shows repo+branch; image app without repo_url shows the image line; cron app shows schedule). Mock `@/lib/get-api`, `@/lib/api` (monitoring), `next/navigation` (`useParams`), `@/components/shell` (passthrough).
- [ ] `pnpm exec tsc --noEmit && pnpm lint && pnpm test`; commit `feat(web): app page as cards — layout, header, source and health`.

### Task 2: Deployments card (+ Releases)
- [ ] `deployments-card.tsx`: move `DeploymentsTab` (legacy ~395-478) incl. `useDeployEvents` refresh and rollback; underneath, a "Releases (n)" toggle that lazily loads `listReleases` and renders the ReleasesTab rows with Deploy (`deployRelease`). Heading "Recent Deployments".
- [ ] Test: renders two deployments, Rollback only on the non-active row, click → `appsDeploy.rollback(appId, id)` called; releases toggle loads and shows a release with a Deploy button.
- [ ] Commit `feat(web): deployments card with releases`.

### Task 3: Environment card (+ secrets)
- [ ] `environment-card.tsx`: move `EnvTab` (legacy ~480-897) unchanged in behaviour; append the secret-store rows (`listSecrets`) to the same table tagged `secret store` with reveal (`getSecretValue` once, toggle hides client-side), copy, delete (`deleteSecret`); the Add form gets a select "Env var (secret) / Secret store" → `envVarsV2.set(..., is_secret: true)` or `setSecret`. Keep Import .env, Show/Hide values, managed badge, Restart-now banner.
- [ ] Test: env rows + secret rows both render (same key in both → two rows); reveal calls `getSecretValue` once; add via secret store calls `setSecret`.
- [ ] Commit `feat(web): environment card merges the secret store`.

### Task 4: Resources card
- [ ] `resources-card.tsx`: databases (`listAppDatabases`, `createAppDatabase`, `deleteAppDatabase`, `getAppDatabase` details, `DatabaseBackupsSection` moved along) and buckets (`listAppBuckets`, `createAppBucket`, `deleteAppBucket`) as rows with a per-row expander for details/backups; "Add database" (engine select) and "Add bucket" (name, access) inline forms. Independent loading/error per list (`ErrorState` with retry on the failing list only).
- [ ] Test: one database and one bucket render; buckets 500 → bucket ErrorState while the database row stays; Add database with engine → `createAppDatabase(appId, {engine})`.
- [ ] Commit `feat(web): resources card — databases and buckets`.

### Task 5: Domains card
- [ ] `domains-card.tsx`: move `DomainsTab` (legacy ~1730-1905) as is; returns `null` unless `app.app_type` is web (or undefined) — the page also gates it.
- [ ] Test: web app renders list + add form, add calls `addDomain(appId, domain)`; worker app renders nothing.
- [ ] Commit `feat(web): domains card`.

### Task 6: Integration
- [ ] Delete `legacy-tabs.tsx`; grep the repo for `activeTab`, `LogsTab`, `AuthTab`, `SecretsTab`, `ReleasesTab` leftovers; `pnpm exec tsc --noEmit && pnpm lint && pnpm test`.
- [ ] UI acceptance (controller): `next dev` against the live API, headless Chrome screenshot of `/apps/<whoami id>` and of a worker/cron app if one exists; record in the ledger.
- [ ] Docs: one line in `docs/alpha/core-audit.md` is NOT needed; update `apps/web/README` if it lists the tabs (grep "Overview").
- [ ] Commit `refactor(web): remove the legacy tab page`.
