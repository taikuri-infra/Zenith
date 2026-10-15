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

**Not run.** Managed databases are metadata-only in Compose (F-16), so there was no database data to check after a reboot. The reboot was also on a shared host. Skipping was proposed to and accepted by the project owner on 2026-10-12.

Still untested until it is re-run after F-16 is fixed:
- all stack containers come back (`restart: unless-stopped`) after a host reboot;
- deployed app containers come back and their URL still returns 200 over HTTPS;
- Traefik keeps its issued certificates (`traefik_letsencrypt` volume);
- a row written to a managed Postgres survives.

## 5. Findings

| ID | Severity | Area | Summary | Repro | Issue |
|----|----------|------|---------|-------|-------|
| F-01 | low | api-surface | 12 billing/plan/CRM routes are registered in standalone mode | `ZENITH_DUMP_ROUTES=1` dump, §1.1 | #9 |
| F-02 | medium | auth | Open registration on a public self-host; signup returns 200 but the account can never be verified (no email service) | `POST /api/v1/auth/register`, then login returns 401 | #10 |
| F-03 | low | build | 3 TypeScript errors in `use-auth.test.ts`; `tsc --noEmit` fails | `cd apps/web && pnpm exec tsc --noEmit` | #11 |
| F-04 | critical | install | Installer fails on every fresh box: `minio/mc` no longer pullable (Docker Hub denied, quay 401) | run install.sh | #12 |
| F-05 | critical | web/security | Published web image bakes `NEXT_PUBLIC_API_URL=https://api.stage.freezenith.com` (`apps/web/Dockerfile:26` default ARG) and the runtime `env.js` override loses to it; admin credentials are POSTed to the staging host | log in at :3000, watch network | #13 |
| F-06 | critical | api | Standalone install enforces the SaaS free plan (1 app, 1 db, no Redis, no custom domain, no backups). `domain.go:52` and `gateway.go:36` ignore `UnlimitedMode` even in current source | `POST /apps/:id/domains` returns 403 | #14 |
| F-07 | high | security | Installer doesn't set `SECRETS_ENCRYPTION_KEY`, so secrets are stored in plaintext | api startup WARN | #15 |
| F-08 | medium | release | Public images are built by hand (workflow_dispatch), carry no revision, report `version: dev`, and are 3 months behind `main` | `docker image inspect` | #16 |
| F-09 | low | cli | `zen` help and login describe Hetzner/Zenith Cloud; login needs a 1-hour JWT pasted by hand | `zen --help`, `zen login --help` | #17 |
| F-10 | high | api | Compose mode wires k8s-only services (Loki, Prometheus, CNPG, backup K8s Jobs) | api startup log | #18 |
| F-11 | high | web | SaaS/upsell/stage-domain UI shown in standalone: Upgrade to Pro, referral, Hetzner tagline, CNAME to `freezenith.com`, `apps.stage.freezenith.com` URLs | §3 rows 6 and 10 | #19 |
| F-12 | critical | tls | Dashboard/API routers on the `tls` profile never request a Let's Encrypt cert (self-signed default); `ACME_EMAIL` unused | `curl https://$ZENITH_DOMAIN` fails cert verification | #20 |
| F-13 | high | web | UI deploy without a project returns 500 (FK violation); no way to create an empty project in the UI | Deploy App on a fresh install | #21 |
| F-14 | high | deploy | App status wrong: stuck `deploying`, then `failed` while the container runs | `GET /apps` vs `docker ps` | #22 |
| F-15 | critical | deploy | Compose import drops service `environment:`; Postgres crash-loops and dependants never start | §3 row 3 stack | #23 |
| F-16 | critical | databases | Managed Postgres/MySQL/Redis are metadata-only in Compose: no container, empty host and password, stuck `provisioning` | `POST /databases` | #24 |
| F-17 | critical | env | Env changes never reach the container: `env-v2/apply` returns 409 "no deployment found"; redeploy doesn't inject vars; Secrets tab 404 | §3 row 5 | #25 |
| F-18 | critical | domains | Custom domain saved `active` but never routed (no traefik label) | add domain, then curl | #26 |
| F-19 | critical | logs | App logs need Loki (k8s), returning 500; deploy logs empty; `zen logs` stream resets | §3 row 7 | #27 |
| F-20 | critical | deploy | Rollback returns 200 then fails silently (`deploy failed`, empty error); image unchanged | `POST /rollback {deployment_id}` | #28 |
| F-21 | high | deploy | Compose services can't reach each other by compose service name. Network aliases use the prefixed app name (`stack1-db`), not `db` | §6 row 3: `pg_isready -h db` returns no response | |
| F-22 | medium (fixed in a39b88e) | deploy | Private compose services still get a public Traefik router and an https URL | §6 row 3: `traefik.enable=true` on `stack1-db` | |
| F-23 | low | web | The Docs page has SaaS content (sign up at freezenith.com, api.freezenith.com examples) | `/docs` | |
| F-24 | low | web | Plan leftovers: overview "Plan enterprise" tile, deploy wizard "(Pro plan)" resource label; the Databases header count omits MySQL | `/`, the deploy wizard review step, `/databases` | |
| F-25 | medium | web/auth | The login page shows the SaaS tagline (Hetzner) and Google/GitHub/Sign up, which don't work in self-host (related to F-02) | `/login` | |
| F-26 | low | api | Custom domain `tls_ready` never becomes true; database env injected on attach is not visible in the Environment tab (legacy table) | `GET /apps/:id/domains`; the app Environment tab | |
| F-27 | high (fixed in a39b88e) | cli | `zen db list/create/connect` are stubs and never call the API | `zen db list` prints "No databases found" while 4 exist | |
| F-28 | medium | cli | `zen status` apps section returns 404 (legacy project endpoint) and shows "Plan: free" and a cost estimate | `zen status` | |

## 6. Re-run after 2a

- **Date / commit / images:** 2026-10-12, `93626e7`. `zenith-api:alpha-2a` and `zenith-web:alpha-2a` were built locally from that commit, and no image was pushed.
- **Install:** `install.sh` from the branch (`ZENITH_REPO_URL=/opt/Zenith`, `ZENITH_DOMAIN=zenith.freezenith.com`). It finished in **9 s** with exit 0 and no warnings.
  - Services: postgres (healthy), storage, api, web and traefik. There is no `storage-init`.
  - API log: `platform bucket ready`, `database runtime: Docker containers`, `app logs: Docker container logs`. There is no `SECRETS_ENCRYPTION_KEY` warning.
  - Ports: only `0.0.0.0:80` and `0.0.0.0:443` are public. 3000 and 8080 are bound to `127.0.0.1`. Database containers publish no host ports.
  - `https://zenith.freezenith.com` serves a Let's Encrypt certificate (issuer YR2), and `curl` without `-k` returns 200.
- **Harness:** headless Chromium against `https://zenith.freezenith.com`, with **no** stage rerouting, plan override or compose override. Plus curl and `zen` (rebuilt from the branch).
  - The browser logged no request to any host other than `zenith.freezenith.com`.
  - Two UI requests (domain add, env apply) were aborted with `ERR_NETWORK_CHANGED`. The browser runs on the Docker host, and recreating a container changes the host's interfaces. Both changes were applied on the server side. This is a harness artifact, not a product finding.

| # | Step | UI | API | CLI | Notes / finding IDs |
|---|------|----|-----|-----|---------------------|
| 1 | Admin login | works | works | works | Login goes to the same origin. The sidebar shows the core pages only. |
| 2 | Deploy image `traefik/whoami:latest` | works | works | works | The `Default` project is created automatically. The app reaches `running`, and `https://whoami-4cb8.zenith.freezenith.com` returns 200 with a valid certificate. |
| 3 | Compose stack | partial | partial | n/a | Both services run, and `db` gets `POSTGRES_PASSWORD`. A service cannot reach another by its compose name (`db` does not resolve; `stack1-db` does) (F-21). The private `db` service still gets a public Traefik router (F-22). |
| 4 | Postgres / MySQL / Redis + attach | works | works | mock | All three become `ready` as real containers, healthy on 5432/3306/6379, and purge removes their volumes. App attach injects `DATABASE_URL`. `zen db list/create/connect` are stubs that never call the API (F-27). |
| 5 | Env `AUDIT=1` + secret `TOKEN` | works | works | n/a | "Restart now" puts both values in the container. The secret is masked in the UI and API and stored encrypted (`enc:v1…`). |
| 6 | Domain `myapp.freezenith.com` + HTTPS | works | works | n/a | Returns 200 with a valid Let's Encrypt certificate whose SAN covers `myapp.freezenith.com` and the generated host. `tls_ready` stays `false` (F-26). |
| 7 | Deploy logs and status | works | works | works | App logs come from Docker. Deploy-log history survives (3 lines). The SSE stream stays open, and `zen logs` streams. |
| 8 | Rollback | works | works | n/a | `zen deploy --image traefik/whoami:v1.10`, then the Rollback button. The container is back on `whoami:latest`. Deployments read superseded/active, and the env vars and domain are kept. |
| 9 | Data persists | works | works | n/a | A row was written and read back through the injected `DATABASE_URL`. |
| 10 | Hidden-link check | partial | n/a | n/a | No core page links to a labs or SaaS page. The login page still says "Kubernetes-native PaaS on Hetzner Cloud" and shows Google/GitHub/Sign up (F-25). The Docs page has freezenith.com sign-up and API examples (F-23). Plan leftovers remain (F-24). |

**Result:** the 10 findings that were critical in §5 are fixed. Remaining gaps: F-21 and F-27 (high), F-22, F-25 and F-28 (medium), F-23, F-24 and F-26 (low).

### 6.1 After the final-review fixes (`a39b88e`)

Images were rebuilt from `a39b88e` and the stack recreated. Live checks:

- Custom domain ``x`) || PathPrefix(`/`` returns 400 "invalid domain". `zenith.freezenith.com` returns 400 "reserved for Zenith".
- **F-22 fixed:** a new compose import's private `db` service has no Traefik labels. The public `web` service keeps its router.
- **Attaching a database redeploys the app:** `REDIS_URL` appeared in the running container with no manual restart. `myapp.freezenith.com` stayed at 200.
- **F-27 fixed:** `zen db list` lists the 5 real databases.
- **F-21 still open:** a bare compose-service alias on the shared `zenith_default` network would collide with control-plane names (`postgres`, `storage`) and across projects. The fix needs one network per project and is moved to 2b.


## 7. After 2b

- **Date / commit / images:** live acceptance on 2026-10-13, post-reboot checks on 2026-10-14. Branch `feat/alpha-2b-compose-durability` at `060dc85`. `zenith-api` and `zenith-web` were built locally from the branch as `0.1.0-alpha.2` (before the final fix wave) and `0.1.0-alpha.3`. The public `0.1.0-alpha.3` images are published by the GitHub release made at the merge of this branch.
- **Install:** the §6 install at `zenith.freezenith.com`, upgraded in place with `zen upgrade` rather than reinstalled, so the apps and databases created before 2b were part of the test. `.env` carries `ZENITH_VERSION=0.1.0-alpha.3`, and the compose file pins `ghcr.io/taikuri-infra/zenith-api:${ZENITH_VERSION}` and `zenith-web:${ZENITH_VERSION}` (F-08).
- **Spec:** docs/superpowers/specs/2026-10-12-alpha-2b-compose-durability-design.md. Task-by-task evidence: `.superpowers/sdd/2026-10-13-alpha-2b-compose-durability/progress.md`.

The spec's five success criteria, checked on the live install:

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | F-21: compose services reach each other by compose service name; another project cannot | works | A stack with `web`, `api` and `postgres` in one project and a second stack in another project. Each project gets its own bridge network (`zenith-proj-abe674e3-68c`, `zenith-proj-d17a14fc-246`); a stack's containers join their project network, with the compose service name as the alias, plus `zenith_default` for Traefik. `web` reached `postgres` by name; the other project's `web` could not resolve it. Apps deployed before 2b stay on `zenith_default` only until they are redeployed. |
| 2 | Managed databases reachable over SSH, no new public port | works | Every managed database publishes its port on `127.0.0.1` only (`127.0.0.1:32768`–`32773` after the reboot). `zen db connect tun1 --tunnel` opened the tunnel over SSH to the host, and the database answered on `127.0.0.1`. Zenith's public listeners are still only 80 and 443. |
| 3 | Reboot: every app and database back with its data, no manual step | works | Owner-approved host reboot at 09:47 UTC on 2026-10-13, checked 22 h later. The running container set matches the pre-reboot list (the only difference is an unrelated, exited xrdp container). `zenith.freezenith.com` and `myapp.freezenith.com` return 200. `/api/v1/version` reports `0.1.0-alpha.3`. The marker rows in `pg1` (`backup-marker`), `rd1` (`bkmarker=yes`) and `tun1` (`before-rm`) are intact. `user_databases.host_port` equals the live loopback port for all 6 databases, so the reconciler refreshed the Docker-picked ports after the restart. Before the reboot, the reconciler was also tested by removing a database container and an app container; both were recreated with their data (host port refresh 32836→32837). |
| 4 | `zen upgrade` A→B; a forced failure rolls back to A; data survives both | works | `0.1.0-alpha.2` → `0.1.0-alpha.3.test` in 11 s, with a backup taken first. A forced-failure upgrade to a deliberately broken `0.1.0-alpha.4.bad` rolled back to `0.1.0-alpha.3.test` automatically, and the data was intact after both. |
| 5 | `zen backup`, then `zen backup restore` on a fresh server | works | `zen backup` archived the install in 5 s (platform database, object storage, every managed database, `.env`), archive mode 0600. Restore into an isolated Docker-in-Docker install brought back login, apps, and the Postgres and MySQL rows. Redis first came back empty: the server loads its AOF and ignored the restored RDB. The final fix wave makes restore rebuild the AOF from the RDB; re-tested, Redis data restored. |

### 7.1 Findings

- **F-08 fixed:** `VERSION` file, images built from a GitHub release carry the version and revision, the installer pins `ZENITH_VERSION`, and `/api/v1/version` reports it. `build-public-images.yml` refuses a release tag that differs from `VERSION`.
- **F-21 fixed:** one Docker network per project, compose-service-name aliases.
- **§4 (reboot) is no longer "not run":** criterion 3 above covers every item listed there, including a row in a managed Postgres surviving the reboot.
- Databases created before 2b had no loopback port; the reconciler now re-provisions such a database at startup (volume and password kept, one restart). Done on the live install, data intact.
- Still open, for 2c: F-01, F-02, F-03, F-09, F-23, F-24, F-25, F-26, F-28.

### 7.2 Known gaps (deferred, not blocking)

- The version check in `zen upgrade` and `composeops` rejects a `-` inside the prerelease part (`0.1.0-alpha.3-test`), which semver allows.
- `zen self-update` reads GitHub's latest non-prerelease release, so it does not see alpha releases. `zen upgrade` uses the newest release including prereleases.
- After a rollback the install's checkout is a detached HEAD at the previous tag.
- `BUILD_TIME` in published images is the repository's `updated_at`, not the build time; locally built images report `unknown`.
- No integration test covers the Docker network lifecycle; the HTTP paths of the upgrade health check and release lookup are untested; the `zen backup --dir` error branch is untested.
- The postgres integration test `TestDeploymentCRUD` fails on `main` as well (pre-existing, not from 2b).

**Result:** alpha exit criteria 3 (reboot), 4 (upgrade) and 5 (backup/restore) are met on the live install, and F-08 and F-21 are closed. 2b is complete. 2c is the self-host UI and CLI cleanup.
