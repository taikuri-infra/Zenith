# Zenith Alpha — Core Audit (Compose edition)

- **Date:** 2026-10-12
- **Commit audited:** aa8a081
- **Images audited:** `ghcr.io/taikuri-infra/zenith-api:latest` (built 2026-07-20) and `zenith-web:latest` (built 2026-07-19), as pulled by the installer on 2026-10-12
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

Command, run from `/root` on 2026-10-12 08:00 UTC (k3s stopped, ports 80/443/3000/8080 free, Docker 29.1.3):

```
curl -fsSL https://raw.githubusercontent.com/taikuri-infra/Zenith/main/infra/scripts/install.sh | ZENITH_DIR=zenith-audit bash
```

- **Result: failed after 5 s** (exit 1). `docker compose up` aborts with `storage-init Error pull access denied for minio/mc, repository does not exist`. `minio/mc` and `minio/minio` are no longer on Docker Hub, and `quay.io/minio/mc` returns 401. **Every fresh install fails** (F-04).
- To keep going, the audit added `docker-compose.override.yml` in `/root/zenith-audit` only. It swaps `storage-init` for `amazon/aws-cli` running `s3 mb`. After that, `docker compose up -d` finished in 32 s (about 37 s end to end).
- Prompts: none.
- Printed: `Dashboard: http://localhost:3000`, `API: http://localhost:8080`, admin `admin@localhost`. The password is printed to the terminal and saved in `.zenith-credentials`. Only `localhost` URLs are printed, although 3000/8080 are bound on `0.0.0.0`.
- `docker compose ps`: postgres running (healthy), storage running, storage-init exited (0), api running, web running. With `--profile tls`, traefik is also running.
- API startup warnings:
  - `SECRETS_ENCRYPTION_KEY not set — env var encryption disabled` (F-07).
  - k8s-only integrations wired up in Compose mode: Loki/Prometheus `*.svc.cluster.local`, "real backup service enabled (K8s Jobs + S3)", "CNPG not configured, metadata-only mode" (F-10).
- `/health` reports `version: dev`. The images carry no revision label and are older than `main` (F-08).

TLS, following the installer's printed instructions (set `ZENITH_DOMAIN=zenith.161.35.82.211.sslip.io`, then `docker compose --profile tls up -d`):

- Dashboard on `https://zenith.161.35.82.211.sslip.io` serves `TRAEFIK DEFAULT CERT`, which is self-signed. The `zenith-web` and `zenith-api` routers set `tls=true` with no `certresolver`, so Let's Encrypt is never asked for the dashboard's host (F-12).
- The `ACME_EMAIL` named by the installer is not read anywhere. `traefik/traefik.yml` has `email: ""`.
- App subdomains (`<sub>.<ZENITH_DOMAIN>`) do get a valid Let's Encrypt certificate, after a redeploy.

## 3. Core scenario

Surfaces:
- **UI:** headless Chromium driving the published web image. Because of F-05, the harness rewrote requests to `api.stage.freezenith.com` into `localhost:8080`; without that, no login works.
- **API:** `curl` with the admin JWT.
- **CLI:** `zen` built from this branch, `zen login --endpoint http://localhost:8080 --token <JWT>`.

The admin account starts on the SaaS `free` plan in standalone mode: 1 app, 1 database, no custom domains (F-06). The audit set it to `enterprise` through `POST /api/v1/admin/users/:id/plan` from step 4 onward.

| # | Step | UI | API | CLI | Notes / finding IDs |
|---|------|----|-----|-----|---------------------|
| 1 | Admin login | broken | works | works | UI posts credentials to `api.stage.freezenith.com` (F-05). CLI login needs a 1-hour JWT pasted by hand (F-09). |
| 2 | Deploy image `traefik/whoami:latest` | partial | works | works | Without a project, UI deploy returns a bare 500 (FK `fk_apps_project`), and the UI has no plain "create project" (F-13). After that, the container runs and is reachable in-network. The public URL only works with the `tls` profile and after a redeploy picks up `ZENITH_DOMAIN`. The Apps list showed "0 apps" while one existed. App status sticks at `deploying` and later flips to `failed` while the container keeps running (F-14). Memory limit not applied (`HostConfig.Memory=0`). |
| 3 | Compose stack | broken | broken | n/a | Import and image check work. On deploy, the `db` service gets no `environment:` (`POSTGRES_PASSWORD` dropped), so Postgres crash-loops and `web` never starts (F-15). A duplicate project name returns 409 and the UI stays on step 1 with no error. |
| 4 | Postgres / MySQL / Redis + attach | broken | broken | broken | Records stay `provisioning` forever, with an empty host and password and no container ("metadata-only mode"). MySQL and Redis report port 5432. The UI shows MySQL as "Coming Soon". `zen db list` shows none (F-16). Attach not tested: there is no ready database. |
| 5 | Env `AUDIT=1` + secret `TOKEN` | partial | broken | n/a | Saving works and the secret is masked in the UI and API listing. "Restart now" and `POST env-v2/apply` return 409 "no deployment found" although an active deployment exists. A redeploy via `zen deploy` does not inject the vars either (F-17). Stored in plaintext because of F-07. The Secrets tab returns 404. |
| 6 | Domain `whoami.161.35.82.211.sslip.io` + HTTPS | broken | broken | n/a | Free plan returns 403 (F-06). The UI tells the user to CNAME to `<uuid>.freezenith.com`, and Networking lists `whoami-9413.apps.stage.freezenith.com` (F-11). With the plan raised, the domain is saved as `active`, but no route is ever added (no traefik label), so it does not resolve (F-18). The generated subdomain does get a valid public certificate. |
| 7 | Deploy logs and status | broken | broken | broken | App logs query `loki.monitoring.svc.cluster.local` and return 500. Deploy-log history is empty and the SSE stream closes with nothing. `zen logs` fails with connection reset (F-19). Status is wrong, see F-14. |
| 8 | Rollback | n/a | broken | n/a | `zen deploy --image traefik/whoami:v1.10` works (5 s). `POST /rollback` without a body returns 400. With `deployment_id` it returns 200 "rollback initiated", then logs `deploy failed` with an empty error, and the image stays v1.10 (F-20). The UI has no rollback control. |
| 9 | Data persists | broken | broken | broken | Blocked by F-16: there is no real database to write to. |
| 10 | Hidden-link check | broken | n/a | n/a | The published web image predates `ZENITH_LABS`. The sidebar shows all labs entries. The overview shows "Plan free / Upgrade to Pro", "Share Zenith, get 1 month Pro free" and legacy-app counters. The deploy wizard shows "Upgrade to Pro" for the registry, S3 and resources. The login page says "Kubernetes-native PaaS on Hetzner Cloud" and offers Google/GitHub/Sign up. Billing pages are prefetched (F-11). |

Open registration (F-02): `/register` is reachable and `POST /auth/register` returns 200 "check your email". With no email service the account can never be verified, so it cannot log in. That is not exploitable, but it is a dead end.


## 4. Reboot

## 5. Findings

| ID | Severity | Area | Summary | Repro | Issue |
|----|----------|------|---------|-------|-------|
| F-01 | low | api-surface | 12 billing/plan/CRM routes are registered in standalone mode | `ZENITH_DUMP_ROUTES=1` dump, §1.1 | |
| F-02 | medium | auth | Open registration on a public self-host; signup returns 200 but the account can never be verified (no email service) | `POST /api/v1/auth/register`, then login returns 401 | |
| F-03 | low | build | 3 TypeScript errors in `use-auth.test.ts`; `tsc --noEmit` fails | `cd apps/web && pnpm exec tsc --noEmit` | |
| F-04 | critical | install | Installer fails on every fresh box: `minio/mc` no longer pullable (Docker Hub denied, quay 401) | run install.sh | |
| F-05 | critical | web/security | Published web image bakes `NEXT_PUBLIC_API_URL=https://api.stage.freezenith.com` (`apps/web/Dockerfile:26` default ARG) and the runtime `env.js` override loses to it; admin credentials are POSTed to the staging host | log in at :3000, watch network | |
| F-06 | critical | api | Standalone install enforces the SaaS free plan (1 app, 1 db, no Redis, no custom domain, no backups). `domain.go:52` and `gateway.go:36` ignore `UnlimitedMode` even in current source | `POST /apps/:id/domains` returns 403 | |
| F-07 | high | security | Installer doesn't set `SECRETS_ENCRYPTION_KEY`, so secrets are stored in plaintext | api startup WARN | |
| F-08 | medium | release | Public images are built by hand (workflow_dispatch), carry no revision, report `version: dev`, and are 3 months behind `main` | `docker image inspect` | |
| F-09 | low | cli | `zen` help and login describe Hetzner/Zenith Cloud; login needs a 1-hour JWT pasted by hand | `zen --help`, `zen login --help` | |
| F-10 | high | api | Compose mode wires k8s-only services (Loki, Prometheus, CNPG, backup K8s Jobs) | api startup log | |
| F-11 | high | web | SaaS/upsell/stage-domain UI shown in standalone: Upgrade to Pro, referral, Hetzner tagline, CNAME to `freezenith.com`, `apps.stage.freezenith.com` URLs | §3 rows 6 and 10 | |
| F-12 | critical | tls | Dashboard/API routers on the `tls` profile never request a Let's Encrypt cert (self-signed default); `ACME_EMAIL` unused | `curl https://$ZENITH_DOMAIN` fails cert verification | |
| F-13 | high | web | UI deploy without a project returns 500 (FK violation); no way to create an empty project in the UI | Deploy App on a fresh install | |
| F-14 | high | deploy | App status wrong: stuck `deploying`, then `failed` while the container runs | `GET /apps` vs `docker ps` | |
| F-15 | critical | deploy | Compose import drops service `environment:`; Postgres crash-loops and dependants never start | §3 row 3 stack | |
| F-16 | critical | databases | Managed Postgres/MySQL/Redis are metadata-only in Compose: no container, empty host and password, stuck `provisioning` | `POST /databases` | |
| F-17 | critical | env | Env changes never reach the container: `env-v2/apply` returns 409 "no deployment found"; redeploy doesn't inject vars; Secrets tab 404 | §3 row 5 | |
| F-18 | critical | domains | Custom domain saved `active` but never routed (no traefik label) | add domain, then curl | |
| F-19 | critical | logs | App logs need Loki (k8s), returning 500; deploy logs empty; `zen logs` stream resets | §3 row 7 | |
| F-20 | critical | deploy | Rollback returns 200 then fails silently (`deploy failed`, empty error); image unchanged | `POST /rollback {deployment_id}` | |
