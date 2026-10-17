# E2E harness (`infra/e2e`)

The harness proves the alpha exit criteria on a **fresh server**: it starts a privileged
Docker-in-Docker container that looks like a new Ubuntu 24.04 box with Docker CE, runs
`infra/scripts/install.sh` inside it exactly as a user would, and hands the server to the Go
suite in `cli/e2e`. The suite never calls `docker`; it reads `artifacts/<name>.env` and reaches
the server through three hook commands.

| File | What it is |
|---|---|
| `Dockerfile.server`, `entrypoint.sh`, `daemon.json` | the server image `zenith-e2e-server:local` (dockerd as PID 1, cgroup v2 nesting, trusts the in-job registry) |
| `server.sh` | `new`, `install`, `reboot`, `exec`, `cp`, `push`, `logs`, `rm`, `down`, `env`, `ip` |
| `run.sh` | the whole run used by `.github/workflows/e2e.yml` and `act` |
| `probe/` | `e2e-probe`, the sample app the suite deploys (`/`, `/env`, `/write?v=`, `/read` against `DATABASE_URL`) |
| `server_test.sh` | checks that need no Docker (run by `ci.yml`) |

## What `run.sh` does

1. Builds `zen` and the `zenith-api`, `zenith-web` and `e2e-probe` images for the candidate
   version `<VERSION>-e2e.<sha7>`, and pushes them to `zenith-e2e-registry:5000` (a `registry:2`
   container on the Docker network `zenith-e2e`). The job host cannot resolve that name, so the
   images are loaded into a throwaway server and pushed from its dockerd.
2. **Server A** — fresh candidate install → `TestInstall`, `TestDeployWithDatabaseAndDomain`,
   `TestReboot`, `TestBackup` (the backup archive lands in `artifacts/`).
3. **Server C** — fresh candidate install, A's archive copied to `/opt/zenith/restore.tar.gz` →
   `TestBackupRestore` with A's admin credentials.
4. **Server B** — the newest release tag older than `VERSION` (override with
   `ZENITH_E2E_UPGRADE_FROM=v…`) → `TestUpgrade` to the candidate. When the checkout has no such
   tag (a shallow clone without tags) `run.sh` fails before building anything. A release whose
   `install.sh` predates `ZENITH_TLS_MODE` gets the same self-signed CA and leaf seeded into
   `/opt/zenith/traefik` after its install (its `traefik.yml` is left alone), so the suite's TLS
   stays strict on every server.
5. Always collects logs into `artifacts/<name>/` and removes the servers, the registry and the
   network (`ZENITH_E2E_KEEP=1` keeps the registry and network). Cleanup removes every
   `zenith-e2e-*` container, so two runs must not share a Docker daemon.

Each install copies the repository into the server as a bare repo at `/src/zenith.git`, with
the tag `v<candidate>` added there (not in your checkout) at a commit on top of `HEAD` whose only
change is `VERSION=<candidate>`, so the installer pins `ZENITH_VERSION` to the tag the candidate
images carry. It runs that tag's `install.sh` with
`ZENITH_REPO_URL=/src/zenith.git ZENITH_BRANCH=<tag> ZENITH_DIR=/opt/zenith
ZENITH_DOMAIN=<ip>.sslip.io ZENITH_PUBLIC_IP=<ip> ZENITH_TLS_MODE=selfsigned
ADMIN_EMAIL=e2e@zenith.local` (and `ZENITH_IMAGE_REPO=zenith-e2e-registry:5000` for the
candidate). The install time includes waiting for `/health`; every wait is bounded (300 s for
`/health`, `ZENITH_E2E_INSTALL_TIMEOUT` default 1500 s for `install.sh`), and a stuck server's
logs are collected before the command fails.

## Running it by hand

Needs Docker (privileged containers), Go, git with tags, curl, and internet (Docker CE apt repo,
images, `sslip.io` DNS). About 25 minutes.

```bash
bash infra/e2e/run.sh --plan     # print the sequence only
bash infra/e2e/run.sh            # the whole run; output in infra/e2e/artifacts/
```

`run.sh` always builds `zen` from this checkout into `artifacts/zen` and sets
`ZENITH_E2E_ZEN` to it itself; a value already in your environment is ignored. Only when you
drive `server.sh` directly (below) do you set `ZENITH_E2E_ZEN` yourself.

Or drive one server yourself:

```bash
cd cli && go build -o ../infra/e2e/artifacts/zen . && cd ..
export ZENITH_E2E_ZEN=$PWD/infra/e2e/artifacts/zen
bash infra/e2e/server.sh new a
# (push candidate images first: see run.sh, or install a release instead)
bash infra/e2e/server.sh install a release v0.1.0
set -a; . infra/e2e/artifacts/a.env; set +a
(cd cli && go test -tags e2e -count=1 -v ./e2e/ -run TestInstall)
bash infra/e2e/server.sh exec a -- docker compose ps
bash infra/e2e/server.sh rm a; bash infra/e2e/server.sh down
```

Under `act` (`act pull_request -W .github/workflows/e2e.yml -j e2e --container-architecture
linux/amd64 --privileged`) the job container is joined to `zenith-e2e` so server IPs are
reachable; no ports are published on the host.

## The suite contract (`artifacts/<name>.env`)

`ZENITH_E2E_ENDPOINT`, `ZENITH_E2E_DOMAIN`, `ZENITH_E2E_ADMIN_EMAIL`, `ZENITH_E2E_ADMIN_PASSWORD`,
`ZENITH_E2E_CA_FILE`, `ZENITH_E2E_ZEN`, `ZENITH_E2E_INSTALL_SECONDS`, `ZENITH_E2E_SERVER_EXEC`,
`ZENITH_E2E_REBOOT_CMD`, `ZENITH_E2E_COPY_FROM_SERVER`, `ZENITH_E2E_CANDIDATE_VERSION`,
`ZENITH_E2E_PROBE_IMAGE`, `ZENITH_E2E_ARTIFACTS`; the restore run adds `ZENITH_E2E_BACKUP_FILE`,
`ZENITH_E2E_EXPECT_ADMIN_EMAIL`, `ZENITH_E2E_EXPECT_ADMIN_PASSWORD`. The hooks are split on
whitespace (so the checkout path must not contain spaces):

- `ZENITH_E2E_SERVER_EXEC` = `server.sh exec <name> --`; the suite appends the command words; cwd `/opt/zenith`.
- `ZENITH_E2E_REBOOT_CMD` = `server.sh reboot <name>`; exits 0 only once `/health` answers again.
- `ZENITH_E2E_COPY_FROM_SERVER` = `server.sh cp <name>`; the suite appends `<remote> <local>`.
