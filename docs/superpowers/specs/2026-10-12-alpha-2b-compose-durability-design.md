# Zenith Alpha — Sub-project 2b: Compose Durability and Operations — Design

- **Status:** draft for review
- **Parent:** `docs/superpowers/specs/2026-10-12-alpha-release-design.md` (sub-project 2, "Compose hardening")
- **Follows:** 2a (`2026-10-12-alpha-2a-core-fixes-design.md`), merged to `main` as `c1b22c2`

## 1. Goal and scope

Make a Compose install survive what happens to a real server: a reboot, an upgrade, a lost disk. These are alpha exit criteria 3, 4 and 5. 2b also closes the two operational gaps 2a left open: compose services can't reach each other by service name (F-21), and managed databases have no way in from outside.

**In scope (2b):**

1. **F-21:** one Docker network per project, so compose services resolve each other by their compose service name.
2. **Database access over SSH:** each managed database listens on a loopback-only host port. `zen db connect` prints a ready SSH tunnel command, or opens the tunnel itself.
3. **Reboot durability:** a startup reconciler that brings apps and databases back to their recorded state.
4. **Versioned releases (F-08):** a `VERSION` file, images built on GitHub release with the version baked in, and an installer that pins it.
5. **`zen upgrade` for Compose:** backup, then switch to the new version, run a health check, and roll back automatically on failure.
6. **`zen backup` / `zen backup restore` for Compose:** one archive that holds the platform database, object storage, every managed database, and the config.

**Next spec (2c), not here:** self-host UI and CLI cleanup (F-01, F-02, F-03, F-09, F-23, F-24, F-25, F-26, F-28), custom-domain DNS verification and `tls_ready`, and moving env writers off the legacy table.

**Already done:** ARM. The public images are built for amd64 and arm64 (`build-public-images.yml`). Running them on arm64 is checked by the E2E suite (sub-project 3).

### Success criteria

On the live install (`zenith.freezenith.com`):

1. A compose stack with `web` and `db` services: `web` reaches `db` at host `db`. An app in another project cannot resolve that `db`.
2. From a laptop: `zen db connect pg1 --tunnel`, then `psql` against `127.0.0.1` reaches the database. No new public port is open; only 80 and 443 are public.
3. After a host reboot, every app and database is back with its data, and `myapp.freezenith.com` returns 200, with no manual step. The reboot needs the owner's explicit yes at acceptance time.
4. `zen upgrade --version <next>` moves the install from version A to version B. A forced health-check failure rolls it back to A, and the data survives both.
5. `zen backup` on the install, then `zen backup restore` on a freshly installed server: apps, databases with their rows, secrets and object storage are all back.

### Out of scope

k3s, multi-node, scheduled backups, off-site backup targets (S3 upload), the web UI for backup/upgrade, and all SaaS code (left untouched; Helm/CNPG `zen upgrade`/`zen backup` paths stay as they are).

## 2. Per-project networks (F-21)

**Today:** every app and database container sits on one shared network, `zenith_default`, together with the control plane (`postgres`, `storage`, `api`). Short aliases on that network would collide with the control plane and across users, which is why 2a reverted the bare service-name alias.

**Design:**

- Each project gets its own bridge network, `zenith-proj-<project id, first 12 chars>`. `DockerDeployer` creates it on demand (idempotent), labelled `zenith.project.id`.
- An app container joins **two** networks:
  - `zenith_default`: for Traefik routing, with aliases unchanged from 2a (app name and subdomain, with control-plane names filtered out);
  - its project network, with alias = its **compose service name**.
- **Service name source:** a new nullable `apps.service_name` column (migration 051). The compose deploy flow in the web UI sends the parsed service name on app create. For apps created before 2b, the deployer falls back to the app name with the `<project slug>-` prefix removed. With neither, there is no project alias.
- A managed database attached to an app also joins that app's project network. Its alias there is its compose service name when it came from a compose import, otherwise none. Connection strings keep using the container name (`zenith-db-<id8>`), which resolves on both networks.
- Deleting the last app of a project removes the project network. A failure to remove it is logged, not an error.

The project network holds only that project's containers, so a short name like `db` can't shadow the control plane or another tenant.

## 3. Database access over SSH

**Decision (owner, 2026-10-12):** databases are internal-only. Access from outside goes through SSH. No public port, no domain.

- `DockerDBBackend.Provision` publishes the engine port on `127.0.0.1` with a host port Docker picks (`HostIP: 127.0.0.1, HostPort: ""`). After start it reads the assigned port and stores it with `SetDatabasePort` (already in the repo interface). Loopback is not reachable from the internet; it is reachable through `ssh -L`.
- The DTO gets `host_port` (loopback port) next to the in-network `host`/`port`.
- `zen db connect <name>`:
  - **default:** prints the tunnel command `ssh -N -L <local>:127.0.0.1:<host_port> <ssh target>` and the local connection string (`postgres://user:pw@127.0.0.1:<local>/db`, and the same for mysql/redis). `<local>` is the engine's default port (5432/3306/6379), overridable with `--local-port`. `<ssh target>` defaults to `root@<API host>`, overridable with `--ssh`.
  - **`--tunnel`:** runs that `ssh` command in the foreground and prints the connection string. Ctrl-C closes the tunnel.
- **Host port after a container recreate:** a recreate (by the reconciler or a restore) can pick a new port. The stored port is refreshed on every Provision, so `zen db connect` always reads the current one.

## 4. Reboot durability: startup reconciler

Docker already restarts `unless-stopped` containers after a reboot, so the common case works. The reconciler covers what Docker can't: a container that was removed, a restore onto a new server, or a crash mid-deploy.

- `services.Reconciler`, standalone only, runs once, 10 s after API start, in the background:
  - **Databases:** every database with status `ready` whose container is missing or not running is re-provisioned. The reconciler calls `DockerDBBackend.Provision` with the stored decrypted secret. The volume is kept, so the data is kept.
  - **Apps:** every app with status `running` and a recorded image (`deploy.CurrentImage`) whose container is missing or exited is redeployed with `DockerDeployer.DeployApp(app, image)`. Apps in other states are left alone.
  - Databases come first, so apps that depend on them start against a live database.
- Each item is reconciled independently: one failure is logged and does not stop the rest. The reconciler only uses the `deploy.Backend` and `DBBackend` interfaces, so a k3s implementation can provide its own.
- **Not in 2b:** a periodic loop. Restart policies cover a container that dies later.

## 5. Versioned releases (F-08)

- A `VERSION` file at the repo root holds the release version, e.g. `0.1.0-alpha.2`.
- `build-public-images.yml`:
  - on `release: published` (tag `v<VERSION>`) it builds both images with `--build-arg VERSION=<tag without v>` and `BUILD_TIME`, and labels them `org.opencontainers.image.revision` and `.version`;
  - pushes `:<version>`, and pushes `:latest` only for non-prerelease tags or when `workflow_dispatch` asks for it;
  - the web image gets the same version as a build arg, exposed in the footer and `/api/v1/version`.
- `install.sh` writes `ZENITH_VERSION=<contents of VERSION>` into a new `.env`. From then on the install runs pinned images, never a moving `:latest`.
- `zen version` shows the CLI version and the server version from the API.

## 6. `zen upgrade` for Compose

- **Where it runs:** on the server, in the install directory. Detection: `--dir` flag, else the current directory, else `~/zenith`, `/root/zenith`. The directory must contain a `docker-compose.yml` referencing `zenith-api`. Without a Compose install the existing Helm path runs unchanged.
- **Version format:** relaxed to semver with an optional prerelease (`0.1.0-alpha.2`) for both paths.

**Steps:**

1. Resolve the target: `--version`, else the latest GitHub release of the repo, including prereleases during alpha.
2. Refuse a downgrade, or the same version, unless `--force`.
3. Run `zen backup` unless `--skip-backup`, and print the archive path.
4. Record the rollback point: the current git ref of the install dir and the current `ZENITH_VERSION`.
5. `git fetch --depth 1 origin tag v<target>` and `git checkout v<target>`. This keeps `docker-compose.yml` and `install.sh` in step with the images.
6. Set `ZENITH_VERSION=<target>` in `.env`, adding keys the new `.env.example` introduced without overwriting existing values (the installer's upgrade logic, shared).
7. `docker compose pull`, then `docker compose up -d --remove-orphans`.
8. **Health check:** within 180 s, `/api/v1/health` returns 200 and `/api/v1/version` reports the target version.

**On failure in steps 5–8:** check out the old ref, restore the old `ZENITH_VERSION`, `up -d`, and re-run the health check. Then report, with the backup path. Database migrations are forward-only. If the new API already migrated the schema, the old API may not start; then the message says to run `zen backup restore <archive>`.

`--dry-run` prints the plan.

## 7. `zen backup` and `zen backup restore` for Compose

Runs on the server, with the same install-dir detection as `zen upgrade`. Without a Compose install, the existing CNPG commands run unchanged.

**`zen backup [--out DIR]`** writes `zenith-backup-<version>-<UTC timestamp>.tar.gz` to `--out`, default `<install dir>/backups`, with mode 0600. Contents:

| Path | Source |
|---|---|
| `manifest.json` | format version, Zenith version, timestamp, list of databases with engine and id |
| `platform.sql` | `pg_dump` of the control-plane database (`docker compose exec postgres`) |
| `storage.tar` | the object-storage volume, through a throwaway container with the volume mounted read-only |
| `databases/<id>.sql` | `pg_dump` (Postgres) or `mysqldump --single-transaction` (MySQL), run inside each database container |
| `databases/<id>.rdb` | Redis `SAVE`, then copy `dump.rdb` |
| `env` | the install's `.env`, which holds `SECRETS_ENCRYPTION_KEY`: without it the encrypted secrets can't be read |

A database that fails to dump fails the backup, so there is no partial archive. The command prints a reminder that the archive contains secrets.

**`zen backup restore <archive> [--yes]`** targets a fresh or existing install. It asks for confirmation unless `--yes`, because it replaces data.

1. Check the manifest. Refuse an archive from a newer Zenith version than the install.
2. `docker compose stop api web`.
3. Write `env` over `.env`, keeping the target's `ZENITH_DOMAIN` if it differs (it prints both). Then `docker compose up -d postgres storage`.
4. Drop and recreate the control-plane database and load `platform.sql`. Empty the storage volume and untar `storage.tar` into it.
5. `docker compose up -d`. The reconciler (§4) re-creates every database container and app from the restored records. Database containers come back empty if their volumes didn't exist.
6. For each database in the manifest: wait for its container to be healthy, then load its dump (`psql` / `mysql` / stop Redis, copy `dump.rdb`, start Redis).
7. Print a summary: apps, databases restored, and any failure.

**Restore order:** the platform database comes back before the API starts. This way the reconciler sees the restored state, not an empty one.

## 8. Testing

- **Unit (Go, `-race`, as CI runs it):**
  - network naming and alias selection (service_name, prefix fallback, none);
  - loopback port binding spec and port read-back;
  - reconciler decisions against fake backends (missing/exited/running × db/app × status);
  - version parsing and comparison;
  - upgrade and backup step sequencing against a fake command runner, including every rollback branch.
- **Installer:** `install_test.sh` asserts `ZENITH_VERSION` is written from `VERSION`.
- **Acceptance on the live install:** the five success criteria in §1. The upgrade test uses two locally built versions (`0.1.0-alpha.2` → `0.1.0-alpha.3-test`) tagged in a local clone, so nothing is published for the test. The restore test uses a second install directory on the same host with its own project name and ports bound to loopback. The reboot needs the owner's yes.

## 9. Risks

- **Network migration of existing apps:** apps deployed before 2b join their project network on their next deploy. The reconciler does not redeploy healthy apps just to add a network. Old compose stacks keep working through their prefixed aliases until they are redeployed.
- **Forward-only migrations limit rollback:** covered by taking a backup before every upgrade by default.
- **Backups on the same disk:** the default output dir is on the same server. The command prints a line saying to copy the archive off the server. Off-site targets are a later feature.
- **Loopback ports change on recreate:** `zen db connect` always reads the current port from the API; a saved tunnel command can go stale, and the docs say so.
