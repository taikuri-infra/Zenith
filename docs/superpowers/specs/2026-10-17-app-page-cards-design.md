# App page without tabs — design

**Date:** 2026-10-17. **Owner decision:** "the app page goes back to how it was at the beginning": one page, cards, no tab bar. Of the four areas that did not exist then, Environment and Domains stay as cards; Logs and Auth leave the app page (sidebar pages).

## 1. Why
`apps/web/src/app/apps/[name]/page.tsx` grew from 169 lines (2026-02-15: header, Source, Resources, Recent Deployments) to 2243 lines with ten tabs — Overview, Environment, Deployments, Logs, Databases, Storage, Auth, Domains, Secrets, Releases. Five of them duplicate sidebar sections, Secrets duplicates Environment, Releases duplicates Deployments. A self-hoster sees a second navigation inside every app.

## 2. The page
Route unchanged (`/apps/[name]`, `[name]` is the app id). Client component. Layout: header, then cards in a responsive grid (`md:grid-cols-2`, wide cards `md:col-span-2`).

1. **Header** (full width): back link to `/apps`, app name, type pill (web/worker/cron), `StatusBadge`, branch, subdomain link (web), cron schedule (cron), created date. Right side: `Open` (app.url, web only) and `Logs` (link to `/logs?app=<id>`). Below it the two existing banners: failed-pod diagnostic (`monitoring.getPods`, `getDiagnosticHint`) and the sleeping banner. The "Production / Staging (Soon) / Dev (Soon)" pills are dropped.
2. **Source** (half): repository link + branch, or image (`deploy_source` image apps show `image_url` when the API returns it; otherwise "image"), framework, port, command, cron schedule, subdomain, custom domains count. Replaces "App Details" + "Quick Links".
3. **Health** (half, web apps with `health_status` only): status, uptime, response time, last check, health-check path/interval/timeout. Otherwise the slot shows **Deployments** wider.
4. **Recent Deployments** (full width): the current DeploymentsTab table (id, status, sha, created, Rollback) with `useDeployEvents` refresh, plus a collapsed **Releases** list underneath ("Releases (n)" toggle) with the Deploy button from ReleasesTab.
5. **Environment** (full width): EnvTab as is (Production/Staging toggle, add, inline edit, delete, Import .env, Show/Hide values, managed badge, Restart-now banner) **plus** the Secrets store rows in the same table, tagged `secret store`, with reveal-on-demand (`getSecretValue`), copy, set and delete. One table, one Add form with a "store as secret" choice: `is_secret` env var (default) or secret store.
6. **Resources** (half or full): attached databases (`listAppDatabases`) and buckets (`listAppBuckets`) as rows: name, engine/access, status; per-row expander for connection details / backups (DatabasesTab + DatabaseBackupsSection code) and delete; "Add database" and "Add bucket" actions (provisioning as today).
7. **Domains** (half, web apps only): DomainsTab as is (list, inline add, verify, delete, `DomainStatus`, instructions).

Removed from the app page: Logs tab (header link instead), Auth tab (sidebar Auth page; per-app auth toggle is not re-created), the tab bar and `activeTab` state.

## 3. Files
`apps/web/src/app/apps/[name]/`: `page.tsx` (shell, header, banners, grid) and one file per card: `source-card.tsx`, `health-card.tsx`, `deployments-card.tsx`, `environment-card.tsx`, `resources-card.tsx`, `domains-card.tsx`; `diagnostics.ts` for `getDiagnosticHint`. Every card takes `{ app: DeployApp }` (and `appId = app.id`), owns its data fetching with `useApi`/`getApi()` as the tabs do today, and uses the existing primitives (`StatusBadge`, `EmptyState`, `ErrorState`, `PageWithTableSkeleton`, `useToast`, `Modal`). Card chrome: `rounded-lg border border-border bg-surface-100 p-5`, heading `mb-4 text-sm font-medium text-white` (the original's white heading).

## 4. Tests
Vitest (jsdom) with the existing `vi.mock("@/lib/get-api")` pattern: one test file per card asserting the main render and one action (rollback calls `appsDeploy.rollback`; environment merges env rows and secret rows and reveals a secret on demand; resources lists a database and a bucket; domains renders nothing for a worker app and adds a domain for a web app; source shows repo/branch or image). `page.test.tsx`: renders every card for a web app, no `role="tab"`/tab buttons, the Logs link points at `/logs?app=<id>`. `pnpm exec tsc --noEmit` and `pnpm lint` stay clean (CI runs both).

## 5. Acceptance
Headless Chrome against a local `next dev` pointed at the live API (as in the 2c audit): the app page of `whoami` shows header, Source, Health, Deployments, Environment, Resources, Domains, no tab bar; Logs link opens `/logs?app=`; screenshot kept in the ledger. Ships as a prerelease later with other web changes (no release in this plan).
