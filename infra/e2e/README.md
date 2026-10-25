# E2E harness (`infra/e2e`)

The harness proves the exit criteria on a **fresh cluster**: it creates a k3d cluster (k3s in
Docker), runs `zen install` against it exactly as a user would on a server (with the candidate
chart and images), and hands the cluster to the Go suite in `cli/e2e`. The suite never calls
`docker` or `kubectl`: it reads `artifacts/<name>.env` and reaches the cluster through three hook
commands (exec, reboot, copy).

| File | What it is |
|---|---|
| `cluster.sh` | `new`, `install`, `reboot`, `logs`, `rm`, `candidate-version` |
| `run.sh` | the whole run used by `.github/workflows/e2e.yml` |
| `probe/` | `e2e-probe`, the sample app the suite deploys (`/`, `/env`, `/write?v=`, `/read` against `DATABASE_URL`) |

## What `run.sh` does

1. Builds `zen` and the `zenith-api`, `zenith-web` and `e2e-probe` images for the candidate
   version `<VERSION>-e2e.<sha7>` (tag `e2e.local/<name>:<version>`).
2. **Cluster A** — `cluster.sh new a`, then `install a`: loads the images into k3d, installs
   cert-manager with a CA `ClusterIssuer`, and runs `zen install --domain <node-ip>.sslip.io`
   with `ZENITH_CHART` pointing at this checkout. Then `TestInstall`,
   `TestDeployWithDatabaseAndDomain`, `TestReboot`, `TestBackup` and `TestUpgrade`.
3. **Cluster C** — a second fresh install, A's archive restored into it: `TestBackupRestore`.
4. Always collects pod logs and events into `artifacts/<name>/` and deletes the clusters
   (`ZENITH_E2E_KEEP=1` keeps them).

The node's Docker IP is the server address (`<ip>.sslip.io`), reachable from the host and from
pods, so DNS checks, custom domains and strict TLS (against the harness CA) behave as on a real
server. The install needs internet access (Helm charts for cert-manager and the operators).

`TestUpgrade` re-applies the candidate over itself with `zen upgrade --force`: there is no earlier
Kubernetes release to upgrade from yet. When one exists, install it first and upgrade to the candidate.

## Restricted networks

`GOPROXY`, `GOSUMDB`, `npm_config_registry` and `COREPACK_NPM_REGISTRY` are passed to the image
builds as build args. Docker Hub and Helm chart mirrors are not handled by the harness.

Run it with `bash infra/e2e/run.sh`; `run.sh --plan` prints the sequence only.
