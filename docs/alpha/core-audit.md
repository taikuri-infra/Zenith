# Zenith Alpha — Core Audit (Compose edition)

- **Date:** 2026-10-12
- **Commit audited:** aa8a081
- **Images audited:** see §2
- **Host:** 161.35.82.211, Ubuntu 24.04, 4 vCPU / 16 GB
- **Spec:** docs/superpowers/specs/2026-10-12-alpha-release-design.md

Status values: `works` / `broken` / `mock` / `partial`.

## 1. Static audit

### 1.1 API routes in standalone mode

Source: `ZENITH_MODE=standalone ZENITH_DUMP_ROUTES=1 go run ./cmd/server` (in-memory stores, no `DATABASE_URL`). slog writes JSON log lines to the same stdout; they were filtered out with `grep -v '^{'`.

| Area | Routes |
|---|---|
| **Total registered** | **402** |
| Apps / deploy / releases / rollback | 90 |
| Databases | 29 |
| Domains | 7 |
| Env vars | 10 |
| Logs / events | 12 |
| SaaS-looking routes still registered | 12 |

SaaS-looking routes reachable in standalone mode (finding F-01):

```
GET  /api/v1/admin/crm/customers/:id/notes
POST /api/v1/admin/crm/customers/:id/notes
PUT  /api/v1/admin/crm/customers/:id/tags
POST /api/v1/admin/users/:userId/plan
GET  /api/v1/billing
POST /api/v1/billing/cancel
POST /api/v1/billing/checkout
POST /api/v1/billing/exit-survey
GET  /api/v1/billing/invoices
POST /api/v1/billing/portal
GET  /api/v1/plan
POST /api/v1/plan/upgrade
```

The API starts in standalone mode without `DATABASE_URL` (in-memory stores) and without any SaaS env vars. No startup errors.

### 1.2 Sidebar pages in standalone mode (labs off)

After the `ZENITH_LABS` change (commit aa8a081), the visible sidebar is exactly the core set:

- Overview `/`, Apps `/apps`, Databases `/databases`, Domains `/networking`, Logs `/logs`, Docs `/docs`, Settings `/settings`

Labs (hidden unless `ZENITH_LABS=true`): Storage, Queues, Auth, IAM, Activity, Monitoring, Registry, CI/CD.

Pages with a `page.tsx` that sit in no sidebar tier. These are flow pages, reached by links rather than the menu: `invite`, `login`, `onboarding`, `register`, `verify-email`. `onboarding` and `register` need checking in the live audit (§3): a self-hosted install with a single admin may not want open registration (finding F-02, pending live check).

### 1.3 Pages using mock or demo data

All mock data is reached through `getApi()` (`apps/web/src/lib/get-api.ts`), which returns `demoApi` only when `NEXT_PUBLIC_DEMO_MODE=true`. The direct mock references are also gated:

| File | Gate | Status |
|---|---|---|
| `app/logs/page.tsx` | `isDemoMode()` | mock-when-demo |
| `hooks/use-deploy-logs.ts` | `DEMO_MODE` | mock-when-demo |
| `hooks/use-deploy-events.ts` | `DEMO_MODE` | mock-when-demo |

**No core page renders mock data unconditionally.** Whether the real endpoints return real data (for example `/api/v1/logs` and `/api/v1/apps/:appId/logs` without Loki in the Compose stack) is checked live in §3.

### 1.4 Build hygiene

- `apps/web`: `tsc --noEmit` reports 3 pre-existing type errors in `src/hooks/__tests__/use-auth.test.ts` (`LoginResponse` fixtures missing `token_type` and `expires_in`). Vitest still passes (36/36). Finding F-03.

## 2. Install

## 3. Core scenario

| # | Step | UI | API | CLI | Notes / finding IDs |
|---|------|----|-----|-----|---------------------|

## 4. Reboot

## 5. Findings

| ID | Severity | Area | Summary | Repro | Issue |
|----|----------|------|---------|-------|-------|
| F-01 | low | api-surface | 12 billing/plan/CRM routes are registered in standalone mode | `ZENITH_DUMP_ROUTES=1` dump, §1.1 | |
| F-02 | medium | auth | Open registration and onboarding flows exist in a single-admin self-host install (pending live check) | §1.2, verify in §3 | |
| F-03 | low | build | 3 TypeScript errors in `use-auth.test.ts`; `tsc --noEmit` fails | `cd apps/web && pnpm exec tsc --noEmit` | |
