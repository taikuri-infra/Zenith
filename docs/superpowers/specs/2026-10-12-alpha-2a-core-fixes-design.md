# Zenith Alpha — Sub-project 2a: Core Fixes (Compose) — Design

- **Date:** 2026-10-12
- **Parent spec:** `docs/superpowers/specs/2026-10-12-alpha-release-design.md` (§4 sub-project 2, §5 principles)
- **Input:** `docs/alpha/core-audit.md` (findings F-01 to F-20, issues #9–#28, milestone alpha.1)
- **Status:** approved in conversation 2026-10-12

## 1. Goal and scope

Make the core scenario from the audit (§3, steps 1–10) work end to end on a fresh Compose install, through UI, API and CLI.

Sub-project 2 in the parent spec is split in two:

- **2a (this spec):** the findings on the core path. Critical: F-04, F-05, F-06, F-12, F-15, F-16, F-17, F-18, F-19, F-20. High on the core path: F-07, F-10, F-11, F-13, F-14.
- **2b (later spec):** reboot durability, `zen upgrade`, backup/restore, ARM images. Medium/low findings (F-01, F-02, F-03, F-08, F-09) go there or later.

### Success criteria

On a fresh Ubuntu box, `install.sh` followed by the audit's 10-step scenario passes every UI/API/CLI cell marked in §8, with:

- no `ZENITH_LABS`, no plan override, no harness rerouting, no compose override;
- the dashboard reachable at `http://<server-ip>` with no domain, and at `https://<ZENITH_DOMAIN>` with a publicly trusted certificate once a domain is set;
- no request from the browser to any `freezenith.com` host.

### Out of scope

k3s, multi-node, CI E2E (sub-project 3), SaaS code changes beyond routing around it in standalone mode, new features.

## 2. Plan limits in standalone (F-06, F-11)

**Problem.** Standalone installs get the SaaS `free` tier: 1 app, 1 database, no Redis, no custom domain. `UnlimitedMode` is honoured in only two places, and about 30 handlers and services call `GetUserPlan` directly.

**Design.** Fix it at the single source. In standalone mode, the plan repository the API wires up returns an unlimited plan for every user:

- Add `unlimitedPlanRepo` (adapter, `internal/adapters/...`), which wraps `ports.UserPlanRepository`. `GetUserPlan` returns tier `enterprise` with `entities.PlanLimitsFor(enterprise)`, or an explicit unlimited limits value if enterprise is capped. Writes pass through or no-op.
- `main.go` wires `unlimitedPlanRepo` when `cfg.Mode != "saas"`. All existing `GetUserPlan` callers then see unlimited without being edited.
- `UnlimitedMode` stays, for `CheckLimit` and `apps_v2.go`.
- `GET /api/v1/plan` returns that plan, so the web gets `tier: enterprise`.

**UI.** In standalone (`IS_STANDALONE`), these are not rendered:

- the tier badge, "Upgrade to Pro" and the referral banner on the overview;
- `UpgradeNudge`, `ReferralCard`, the referral tab in Settings;
- upsell lines in `deploy-wizard.tsx` and `onboarding-wizard.tsx`;
- tier-gated "Pro" notices.

The custom-domain dialog tells the user to point an **A record at the server's IP**, not a CNAME at `<id>.freezenith.com`. The Networking page shows the app's real host (`<sub>.<ZENITH_DOMAIN>`), not `apps.stage.freezenith.com`.

## 3. One front door: Traefik always on, same-origin API (F-05, F-12)

**Problem.** The web image bakes `https://api.stage.freezenith.com`. The runtime override is a module-level constant, so it loses to the build-time value. The default `http://localhost:8080` cannot work from a remote browser. The dashboard router has no certresolver, so it never gets a trusted certificate.

**Design.**

- **Traefik is always on.** Drop `profiles: ["tls"]`. Two modes, chosen by the installer and written to `traefik/traefik.yml`:
  - **No domain (default):**
    - The `web` entrypoint (`:80`) serves HTTP.
    - Routers match any host: `PathPrefix(/api)` goes to the API, everything else to the web.
    - App routes are `<sub>.<server-ip>.sslip.io` over HTTP. This is so apps are reachable with no DNS. The base domain falls back to `<ip>.sslip.io` when `ZENITH_DOMAIN` is empty.
  - **Domain set (`ZENITH_DOMAIN`):**
    - `:80` redirects to `:443`.
    - The dashboard and API routers carry `tls.certresolver=le`.
    - Apps get `<sub>.<ZENITH_DOMAIN>` with `le`.
    - The ACME email comes from `ACME_EMAIL`. Empty is allowed.
- **Same-origin API.** The web calls the API at `window.location.origin` by default:
  - `API_BASE_URL` resolves at call time, not module load. The order is: runtime `window.__ENV.NEXT_PUBLIC_API_URL` if non-empty, then `window.location.origin` in the browser, then the build-time value on the server.
  - The Dockerfile default ARG becomes empty.
  - The stage URL is removed from the self-host build path. `build-images.yml` and `deploy-staging.yml` keep passing their own value explicitly.
- Ports 3000 and 8080 are no longer published on `0.0.0.0`. They are bound to `127.0.0.1` for local debugging.

## 4. Docker deploy correctness (F-13, F-14, F-17, F-18, F-20)

**4.1 Deployment image.**
- `CreateDeployment` writes `image_tag`.
- Every caller passes the image: `apps_v2.go`, `ci_deploy.go`, the redeploy path and rollback.
- Rollback and domain refresh then have an image to use.

**4.2 App status.**
- After `DockerDeployer.DeployApp` succeeds, the pipeline sets `apps.status = running`. On error it sets `failed`, as today.
- The failure event carries an `error` key, so the log line is no longer empty.
- No reconciler in 2a; a container that dies later is a 2b concern.

**4.3 Env vars: one table.**
- `app_env_vars_v2` is the single source.
- `appRepo.SetEnvVars`, used by app create and compose deploy, writes to v2 with `environment_id = NULL` (the app-wide default).
- `DockerDeployer.resolveEnvVars` merges two scopes, with environment-scoped vars overriding:
  - app-wide (`environment_id IS NULL`);
  - the deploy's environment.
- Secrets are decrypted with the env crypto when set.
- The legacy table is still read when v2 is empty, for existing installs; 2b migrates it.

**4.4 Apply and restart.**
- `env-v2/apply` looks for the latest **deployment**, not release.
- It redeploys through the pipeline, so the same `deploy.Backend` (`DockerDeployer` in standalone) is used.
- `SetRestarter` and the domain handler get the runtime `deploy.Backend`, not the k8s `Deployer`.

**4.5 Custom domains.**
- `DockerDeployer` gets a domain lister. `proxyLabels` adds every active custom domain of the app to the Host rule.
- In domain mode each router carries `tls.certresolver=le`.
- Adding or removing a domain triggers a redeploy of the current image through the pipeline. This recreates the container with new labels.

**4.6 Project required.**
- On first login in standalone, the API creates a project named `default` if the user has none.
- `POST /apps` without a project returns 400 with a clear message, not 500.
- The web preselects the only project.

## 5. Databases on Docker (F-16)

Per parent spec §5, databases go behind an interface with a Docker implementation first.

```go
// internal/services/dbruntime (new)
type Backend interface {
    Provision(ctx context.Context, db *entities.Database) (Credentials, error) // idempotent
    Delete(ctx context.Context, db *entities.Database) error
    Status(ctx context.Context, db *entities.Database) (Status, error)
}
type Credentials struct{ Host string; Port int; User, Password, DBName string }
```

**Docker implementation:**

| Engine | Image | Port | Volume |
|---|---|---|---|
| postgresql | `postgres:16-alpine` | 5432 | `zenith-db-<id>` |
| mysql | `mysql:8.4` | 3306 | `zenith-db-<id>` |
| redis | `redis:7-alpine` (`--requirepass`) | 6379 | `zenith-db-<id>` |

- Container name: `zenith-db-<short-id>`, on `zenith_default`, with restart `unless-stopped` and label `zenith.db.id`.
- Not published on the host. Apps reach it by container name.
- The password is generated with `crypto/rand` and stored encrypted (`SECRETS_ENCRYPTION_KEY`).
- Status is set to `ready` when the engine health check passes (`pg_isready`, `mysqladmin ping`, `redis-cli ping`), polled with a timeout. Otherwise it is `failed` with the error.
- The per-engine port is stored (fixes 5432 for every engine).
- Attaching a database to an app writes `DATABASE_URL` (Postgres/MySQL) or `REDIS_URL` into the app's v2 env, then redeploys the app.
- Delete removes the container. The volume is kept unless `?purge=true`.
- `CreateStandalone` and `Create` call the backend when one is configured. Metadata-only mode remains only when no backend is set (tests, dev).
- MySQL is no longer "Coming Soon" in the UI.

## 6. Logs on Docker (F-10, F-19)

```go
// internal/services/logruntime (new)
type Backend interface {
    Tail(ctx context.Context, app *entities.App, since time.Time, limit int) ([]LogLine, error)
    Stream(ctx context.Context, app *entities.App, since time.Time) (<-chan LogLine, error)
}
```

- The Docker implementation reads `ContainerLogs` of `zenith-app-<subdomain>` (stdout and stderr, demuxed, timestamps).
- `GET /apps/:id/logs` and `/logs/stream` use the logs backend. In standalone the Loki client is not wired.
- **Deploy logs:**
  - `DockerDeployer` emits pull/create/start lines into the existing LogHub.
  - The pipeline writes the collected lines to `deployments.build_log` when the deploy ends.
  - `/logs/history` falls back to `build_log` when the hub has nothing. This covers API restarts.
- **In standalone, k8s-only integrations are not wired:** Loki, Prometheus, the CNPG backup service. Startup logs say they are off.
- `zen logs` must stream from the same endpoint.

## 7. Install (F-04, F-07, F-15)

- **Remove `storage-init` and `minio/mc`.** On startup the API ensures the platform bucket exists (`HeadBucket`, then `CreateBucket`) with retries until storage is ready. The API no longer depends on a one-shot job.
- `install.sh` generates `SECRETS_ENCRYPTION_KEY` (64 hex) and `S3_ACCESS_KEY`. The compose file passes them to the API.
- `install.sh` detects the public IP and writes the Traefik config:
  - no domain: HTTP mode;
  - `ZENITH_DOMAIN` set: domain mode with `le`.
- It prints `http://<ip>` or `https://<domain>`, not `localhost`.
- **Compose env (F-15):** with §4.3 in place, `environment:` from an imported compose file reaches the container. This is covered by a test on the compose → app → container env path.

## 8. Acceptance: the audit scenario, re-run

The audit's §3 table is the acceptance test. Required cells after 2a:

| # | Step | UI | API | CLI |
|---|------|----|-----|-----|
| 1 | Admin login | works | works | works |
| 2 | Deploy image | works | works | works |
| 3 | Compose stack | works | works | n/a |
| 4 | Postgres/MySQL/Redis + attach | works | works | works (`zen db create/list`) |
| 5 | Env + secret | works | works | n/a |
| 6 | Custom domain + HTTPS | works | works | n/a |
| 7 | Deploy logs + status | works | works | works (`zen logs`) |
| 8 | Rollback | works (button on Deployments) | works | n/a |
| 9 | Data written to Postgres | works | works | works |
| 10 | No SaaS/upsell link on core pages | works | n/a | n/a |

The re-run is recorded in `docs/alpha/core-audit.md` as a new §6 "Re-run after 2a", with the same table format.

## 9. Testing

- **Go:** unit tests per change, using the existing memory repos and fakes. The Docker backends (deploy, db, logs) get tests behind a `docker` build tag that run against the local daemon. The CI job for them is in sub-project 3.
- **Web:** vitest for `API_BASE_URL` resolution, standalone upsell gating, and the domain dialog text.
- **Installer:** shellcheck, plus a dry run that renders `.env` and `traefik.yml` for both modes into a temp dir.
- **Acceptance:** §8 re-run on a fresh install on the audit host, with images built from this branch (pushed as `ghcr.io/taikuri-infra/zenith-{api,web}:alpha-2a`, or loaded locally if push is not approved).

## 10. Risks

- **Labels cannot change on a running container.** Domain changes recreate the container, which causes a short downtime per change. This is accepted for alpha.
- **The legacy env table fallback** keeps two read paths until 2b migrates them.
- **A database container per instance** uses memory: about 30–100 MB idle for Postgres and around 400 MB for MySQL. The 2 GB-box exit criterion holds for one app and one Postgres. MySQL on a 2 GB box is documented as tight.
