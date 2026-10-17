# Zenith Alpha — Sub-project 3: E2E Suite and the alpha.1 Release — Design

- **Date:** 2026-10-17
- **Status:** approved by the owner in conversation (design); spec written for the plan
- **Inputs:** docs/superpowers/specs/2026-10-12-alpha-release-design.md (§2 exit criteria, §3 milestones, §5 principles), docs/alpha/core-audit.md §6–§8, docs/ci-cd-pipelines.md (workflows run locally with `act`)
- **Ships as:** `v0.1.0`, released as "alpha.1 (Compose edition)"

## 1. Goal and scope

The alpha roadmap says every exit criterion is covered by an automated E2E test in CI, and "works" means that test is green. 2a–2c proved the criteria by hand on `zenith.freezenith.com`. This sub-project turns that into a suite that installs Zenith on a fresh server inside a CI job, drives it only through the API and the `zen` CLI, and checks all six criteria on amd64 and arm64. When it is green on `main`, `v0.1.0` is tagged as the alpha.1 milestone.

**Decisions taken with the owner on 2026-10-17:**

1. **Where it runs.** The job creates its own "fresh server": a privileged Docker-in-Docker container built from `ubuntu:24.04` with Docker installed, so the documented one-command install runs on a clean Ubuntu userland. The same job runs on this host with `act` (amd64) and on GitHub with a matrix of `ubuntu-latest` (amd64) and `ubuntu-24.04-arm` (arm64; free for the public repo). No cloud secrets, no cost. A real-VM job (real Let's Encrypt, real reboot) is not part of this sub-project; the live install keeps covering that by hand.
2. **Milestone tag.** `v0.1.0`, GitHub release titled `v0.1.0 — alpha.1 (Compose edition)`, still flagged prerelease so `:latest` does not move. k3s single-node becomes `v0.2.0` (alpha.2) later.
3. **What "install" means in the suite.** `infra/scripts/install.sh`, the one-liner the README documents. Because a server without a public IP cannot get a Let's Encrypt certificate, `install.sh` gains `ZENITH_TLS_MODE=selfsigned` (a local CA and a wildcard leaf), the same mode `zen install --edition compose --tls-mode selfsigned` already offers. This is also a real feature for LAN and offline installs.
4. **Suite language.** Go, in the `cli` module (`cli/e2e`, build tag `e2e`), so it reuses the CLI's API client and runs the real `zen` binary. It never talks to Docker.

**In scope (3):**

| # | Item | Area |
|---|------|------|
| 1 | E2E suite: six criteria as named tests, API + `zen` only, harness hooks for reboot, server exec and a second server | cli/e2e |
| 2 | Harness: fresh-server image, server lifecycle, in-job image registry, orchestration script, bash tests | infra/e2e |
| 3 | Workflow `e2e.yml`: PR (path-filtered), `main`, manual; amd64 + arm64; runnable with `act`; artifacts on failure | .github, docs |
| 4 | Product changes the suite needs: `ZENITH_IMAGE_REPO`, `ZENITH_CERT_RESOLVER`, `install.sh` self-signed mode and env overrides, API trusts the install CA, wildcard leaf | compose, installer, api, cli |
| 5 | Docs: README install section, `docs/ci-cd-pipelines.md` E2E + `act` command; core-audit §9 | docs |
| 6 | Release: `VERSION` 0.1.0, prerelease `v0.1.0` from a green `main` | release |

**Out of scope:** k3s hooks beyond the interface the suite already exposes; a Hetzner/real-VM job; load or soak tests; multi-user; SaaS mode (unchanged); the labs pages.

### Success criteria

1. `.github/workflows/e2e.yml` is green on `main` for both `amd64` and `arm64`, and `act pull_request -W .github/workflows/e2e.yml -j e2e` is green on this host for amd64, with a total wall time under 45 minutes per architecture.
2. Each exit criterion of the roadmap maps to a named test that fails when the behaviour is broken: `TestInstall` (one command, dashboard reachable, under 15 minutes), `TestDeployWithDatabaseAndDomain` (image app, Postgres, custom domain, HTTPS valid against the install CA), `TestReboot` (everything back with data), `TestUpgrade` (previous release → candidate, no data loss), `TestBackupRestore` (archive from one server restored on a fresh one), and the matrix supplies amd64 + arm64.
3. `install.sh` with `ZENITH_TLS_MODE=selfsigned` produces a working HTTPS install whose CA is printed in the summary; the Go installer writes the same files.
4. `v0.1.0` is published from a `main` commit whose E2E run is green, with images `ghcr.io/taikuri-infra/zenith-{api,web}:0.1.0` for both architectures, and the README one-liner installs it.

## 2. The E2E suite (`cli/e2e`)

**Today:** `infra/scripts/e2e-*.sh|py` target the SaaS staging cluster and need staging credentials; nothing installs Zenith. The CLI module has an API client (`cli/internal/api`) with login, apps, databases, domains, env-v2 and version.

**Design:**

- Package `e2e` in the `cli` module, files `config.go`, `client.go`, `install_test.go`, `deploy_test.go`, `reboot_test.go`, `upgrade_test.go`, `backup_test.go`, all with `//go:build e2e`. Run with `go test -tags e2e -count=1 -timeout 40m ./e2e/ -run <Test>`.
- **Configuration is environment only** (the harness sets it; a k3s harness would set the same names):

  | Variable | Meaning |
  |---|---|
  | `ZENITH_E2E_ENDPOINT` | `https://<domain>` of the server under test |
  | `ZENITH_E2E_DOMAIN` | the install domain (`<ip>.sslip.io` in CI) |
  | `ZENITH_E2E_ADMIN_EMAIL`, `ZENITH_E2E_ADMIN_PASSWORD` | the installer's admin |
  | `ZENITH_E2E_CA_FILE` | PEM of the install CA; when set the suite's HTTP client and `zen` trust it (`SSL_CERT_FILE` for `zen`); unset → system roots |
  | `ZENITH_E2E_ZEN` | path to the candidate `zen` binary on the job host |
  | `ZENITH_E2E_INSTALL_SECONDS` | wall time of the install, measured by the harness |
  | `ZENITH_E2E_SERVER_EXEC` | command prefix that runs a command *on the server* with the install directory as cwd (DinD: `docker exec -w /opt/zenith zenith-e2e-<name>`); the suite uses it for `zen backup`, `zen backup restore` and `zen upgrade`, which operate on the install directory |
  | `ZENITH_E2E_REBOOT_CMD` | command that reboots the server and returns when it is up again (DinD: `infra/e2e/server.sh reboot <name>`) |
  | `ZENITH_E2E_COPY_FROM_SERVER` | command prefix `… <remote path> <local path>` (DinD: `docker cp zenith-e2e-<name>:`) used to fetch the backup archive |
  | `ZENITH_E2E_CANDIDATE_VERSION` | version the upgrade test upgrades to (`<VERSION>-e2e.<sha7>`) |
  | `ZENITH_E2E_BACKUP_FILE` | (restore run) path of the archive already placed on the server |
  | `ZENITH_E2E_EXPECT_ADMIN_EMAIL` | (restore run) admin of the backed-up server, who must be able to log in after the restore |

- **Tests and assertions** (each logs what it did; failures include the response body):
  - `TestInstall`: `ZENITH_E2E_INSTALL_SECONDS` < 900; `GET /health` 200; `GET /` (dashboard) 200 over HTTPS with the CA; `GET /api/v1/version` equals the candidate version and `mode` is `standalone`; `POST /auth/register` is 403; `zen login` with email and password succeeds.
  - `TestDeployWithDatabaseAndDomain`: `zen deploy --image traefik/whoami:latest` (or `POST /apps` + deploy) → app `running`, its generated URL returns 200 over HTTPS with the CA; create a Postgres attached to the app → `ready`; through the database explorer endpoint run `CREATE TABLE e2e(k text)`, `INSERT 'alpha-1'`, `SELECT` → the row is there; the app's env-v2 lists `DATABASE_URL` with `source` `managed_service`; add custom domain `myapp.<ZENITH_E2E_DOMAIN>` → within 180 s `status` `active` and `tls_ready` true; `GET https://myapp.<domain>/` 200 with a certificate chain valid against the CA.
  - `TestReboot`: records app URL, domain, row; runs `ZENITH_E2E_REBOOT_CMD`; waits for `/health` (≤ 300 s); app URL 200, database `ready`, `SELECT` returns the row, domain still `active`, `zen status` lists the app and the database.
  - `TestBackupRestore` (two runs): on server A, `zen backup` via `ZENITH_E2E_SERVER_EXEC` → archive path parsed from its output; `ZENITH_E2E_COPY_FROM_SERVER` fetches it. On server C (fresh candidate install, run with `ZENITH_E2E_BACKUP_FILE` set) `zen backup restore <file>` via `SERVER_EXEC` → after the API is back: login with `ZENITH_E2E_EXPECT_ADMIN_EMAIL` + A's password, the app exists and is `running`, the Postgres is `ready` and `SELECT` returns the row.
  - `TestUpgrade` (server B, installed from the previous release): deploy an app and write a row as above; `zen upgrade --version $ZENITH_E2E_CANDIDATE_VERSION` via `SERVER_EXEC` → exit 0, `/version` equals the candidate, app URL 200, row present, `zen status` works.
- The suite never shells out to `docker`; every server-side action goes through the hooks. That is the seam a k3s harness fills later.
- A small unit test (no build tag) covers config parsing and the hook command splitting.

## 3. The harness (`infra/e2e`)

- `Dockerfile.server`: `ubuntu:24.04`, `apt-get install -y curl git openssl ca-certificates iproute2` plus Docker CE and the compose plugin from the official apt repo; `/etc/docker/daemon.json` allows the insecure in-job registry `zenith-e2e-registry:5000`; entrypoint runs `dockerd` in the foreground. Built once per job as `zenith-e2e-server:local`.
- `server.sh` subcommands:
  - `new <name>`: creates network `zenith-e2e` (once), starts `zenith-e2e-registry` (`registry:2`) on it (once), starts `zenith-e2e-<name>` privileged on it with the image above, waits for its `docker info`.
  - `install <name> candidate|release <tag>`: copies the repository (candidate: this checkout, tagged `v<candidate version>`; release: the local git tag) into the server as a bare repo at `/src/zenith.git`; runs `install.sh` inside with `ZENITH_REPO_URL=/src/zenith.git ZENITH_BRANCH=<tag> ZENITH_DIR=/opt/zenith ZENITH_DOMAIN=<container ip>.sslip.io ZENITH_PUBLIC_IP=<container ip> ZENITH_TLS_MODE=selfsigned ADMIN_EMAIL=e2e@zenith.local` and, for the candidate, `ZENITH_IMAGE_REPO=zenith-e2e-registry:5000`; measures wall time; copies `/opt/zenith/traefik/certs/zenith-ca.crt` out; copies the candidate `zen` into the server at `/usr/local/bin/zen`; writes `<workdir>/<name>.env` with every `ZENITH_E2E_*` variable of §2.
  - `reboot <name>`: `docker restart`, then waits for `/health` 200 through the container IP (≤ 300 s).
  - `exec <name> -- <cmd…>`, `cp <name> <remote> <local>`, `rm <name>`, `logs <name>` (docker logs of the server and `docker compose logs` inside it, into the artifacts dir).
- `run.sh` (what the workflow and `act` call): builds `zen` for the host arch, builds the api and web images from the checkout tagged `<VERSION>-e2e.<sha7>` and pushes them to the in-job registry, then: server A candidate → `go test -run 'TestInstall|TestDeploy|TestReboot|TestBackup'`; server C candidate with A's archive → `go test -run TestBackupRestore` (restore half); server B release `<previous tag>` → `go test -run TestUpgrade`; always collects logs and removes servers. Previous tag = the newest `v*` tag older than the candidate (`git tag --sort=-v:refname | grep -v e2e | head -1`), overridable with `ZENITH_E2E_UPGRADE_FROM`.
- Reachability: the suite runs on the job host (GitHub runner) or in the `act` job container; `server.sh` connects the job container to `zenith-e2e` when it detects `act` (`$ACT` set), so container IPs are reachable either way. Ports are not published on the host.
- `server_test.sh`: bash tests that need no Docker: the Dockerfile exists and pins `ubuntu:24.04`; `server.sh env` renders the expected `ZENITH_E2E_*` set from a fake install `.env`; `run.sh --plan` prints the sequence without executing. Run by `ci.yml` next to `install_test.sh`.

## 4. The workflow (`.github/workflows/e2e.yml`)

- `on`: `pull_request` with `paths` `services/api/**`, `cli/**`, `apps/web/**`, `docker-compose.yml`, `traefik/**`, `infra/scripts/**`, `infra/e2e/**`, `VERSION`, `.github/workflows/e2e.yml`; `push` to `main`; `workflow_dispatch`.
- One job `e2e` with `strategy.matrix.include`: `{runner: ubuntu-latest, arch: amd64}` and `{runner: ubuntu-24.04-arm, arch: arm64}`; `if: ${{ !env.ACT || matrix.arch == 'amd64' }}` on the arm entry so `act` runs only amd64; `timeout-minutes: 60`; `concurrency` per ref, cancel in progress.
- Steps: checkout with tags (`fetch-depth: 0`), setup-go from `cli/go.mod`, `bash infra/e2e/run.sh`, upload `infra/e2e/artifacts/**` on failure.
- `docs/ci-cd-pipelines.md` gains the job and the exact `act` command (`act pull_request -W .github/workflows/e2e.yml -j e2e --container-architecture linux/amd64 --privileged`), with the note that the job needs the Docker socket and about 25 minutes.

## 5. Product changes the suite needs

- **Image repository and cert resolver are variables.** `docker-compose.yml`: `image: ${ZENITH_IMAGE_REPO:-ghcr.io/taikuri-infra}/zenith-api:${ZENITH_VERSION:-latest}` (and web); `ZENITH_CERT_RESOLVER: ${ZENITH_CERT_RESOLVER-le}` for the api service (empty disables ACME on app routers). `.env.example` documents both. `install.sh` writes `ZENITH_IMAGE_REPO` when the variable is set and `ZENITH_CERT_RESOLVER=` (empty) in self-signed mode.
- **`install.sh` self-signed mode.** `ZENITH_TLS_MODE` (`http01` default, `selfsigned`). Self-signed: with `openssl`, a CA (`zenith-ca.crt/.key`, 10 years) and a leaf for `<domain>` with SANs `<domain>` and `*.<domain>` (2 years) under `traefik/certs/`; `traefik/dynamic/certs.yml` with the exact content `zen install` writes; `traefik/traefik.yml` with the `certResolver: le` lines removed from `websecure` (the same transform the Go installer applies); `.env` gets `ZENITH_TLS_MODE=selfsigned`, `ZENITH_TLS_CA_FILE=/etc/zenith/certs/zenith-ca.crt`, `ZENITH_CERT_RESOLVER=`; the summary prints where the CA is and how to trust it. `ADMIN_PASSWORD` honours an existing environment value (today it is always generated). `install_test.sh` covers the new mode in dry-run (files rendered, env keys present, http01 unchanged).
- **The Go installer matches.** `tlscert.GenerateLeaf` adds the wildcard SAN; `zen install --edition compose --tls-mode selfsigned` writes the same three `.env` keys. One Go test asserts the Traefik YAML the two installers produce is byte-identical (the bash test writes its output to a temp dir and the Go test reads the expected strings from `traefik.go`).
- **The API trusts the install CA.** Compose mounts `./traefik/certs:/etc/zenith/certs:ro` into `api`. `config.Config.TLSCAFile` (`ZENITH_TLS_CA_FILE`); when the file exists, `main.go` builds the domain checker's `TraefikTLSProber` with `Roots` = system pool plus that CA, so a custom domain under the install domain becomes `active` on a self-signed install. Missing or unreadable file → system roots and a warning. A unit test covers the pool construction.
- **Custom subdomains of the install domain (standalone only).** The API reserved the base domain and every name under it, which on a self-host blocks the only names the self-signed wildcard leaf covers. In standalone mode `validate.CustomDomain` allows subdomains of the base domain as custom domains; it still refuses the base itself (400) and any name equal to an existing app host (409, `that host belongs to an app`). SaaS mode is unchanged. The suite's custom-domain step uses `myapp.<server domain>`.
- **Candidate versions.** `zen upgrade`'s version regex already accepts `<VERSION>-e2e.<sha7>`; the harness tags the candidate in the bare repo so `git fetch --depth 1 origin tag v<candidate>` works.

## 6. Docs and release

- README: the one-liner, what the install gives you, `ZENITH_TLS_MODE=selfsigned` for LAN/offline installs, and a link to the release page for `zen`.
- `docs/ci-cd-pipelines.md`: the E2E job, the `act` command, the artifacts.
- `docs/alpha/core-audit.md` §9 "alpha.1 — E2E": the run links for both architectures and the mapping test → criterion.
- Release: this sub-project's PR sets `VERSION` to `0.1.0`; after the squash merge, when `e2e.yml` on `main` is green for both architectures, a GitHub prerelease `v0.1.0` titled `v0.1.0 — alpha.1 (Compose edition)` is created (`--latest=false`); `build-public-images.yml` publishes `0.1.0` images; the live install is upgraded to it with `zen upgrade --version 0.1.0`.

## 7. Testing

- `cli/e2e`: the suite itself (tagged) plus an untagged unit test for config and hook parsing; `go vet -tags e2e ./e2e/` in `ci.yml` so the suite always compiles.
- `infra/e2e/server_test.sh` and the extended `install_test.sh` in `ci.yml`.
- API: unit test for the CA pool; CLI: `tlscert` wildcard SAN test and installer `.env` keys test; compose: `docker compose config` with and without the new variables.
- The E2E run itself, on this host with `act` before the PR is opened, then on GitHub for both architectures.

## 8. Risks

- **Resources.** A GitHub runner has 2 vCPU / 7 GB; three sequential installs plus image builds fit in ~25 min on amd64; the arm runner is similar. Servers are removed after each phase to free memory. If arm64 proves slower than 45 min, the arm job keeps only `TestInstall` + `TestDeploy` + `TestReboot` and the others run amd64-only (recorded in §9 if it happens).
- **Privileged DinD.** Required for a nested Docker daemon; the job only runs trusted code from this repository. Under `act` the job container must see the host Docker socket (act's default).
- **External DNS for `sslip.io`.** The suite and Traefik resolve `<ip>.sslip.io` through public DNS. CI has internet; an offline `act` run would fail at DNS — accepted.
- **Duplicated self-signed logic** (bash and Go). Contained by the byte-equality test on the generated Traefik files.
- **Upgrade from the previous release** pulls public ghcr images; the first run after a release with no previous `v*` tag older than the candidate skips `TestUpgrade` with a clear message.
- **Timing flakiness** (certificate issuance is instant in self-signed mode, but domain checks run every 30 s): every wait has a bounded timeout and logs the last state on failure.
