# Docker Deployer (self-host CE — run user apps without Kubernetes)

**Goal:** Let the docker-compose self-host edition actually run user apps as Docker
containers on the same host (no k8s), routed with automatic HTTPS via Caddy.

**Why:** The only deployer today (`services/api/internal/services/deploy/deployer.go`)
creates Kubernetes resources. Without a cluster the API uses an in-memory k8s client
(`NewMemoryClient`), so deployments are *simulated*. This backend makes them real for
the single-server CE. Cloud/Enterprise keeps the k8s backend.

## Architecture

Two backends behind one interface, selected by `ZENITH_MODE`:
- `standalone` → **DockerDeployer** (this plan)
- `saas` → existing k8s `Deployer` (unchanged)

Routing: swap the plain `caddy:2-alpine` for **`lucaslorentz/caddy-docker-proxy`**.
Each app container gets labels (`caddy: <sub>.<domain>`, `caddy.reverse_proxy:
{{upstreams <port>}}`) and Caddy auto-configures the route + Let's Encrypt cert.
No Caddy API calls, no reloads.

## Tasks (each ends compiling + committed)

### Task 1: Extract a Backend interface
- Add to `deploy` package:
  ```go
  type Backend interface {
      DeployApp(ctx context.Context, app *entities.App, imageTag string) error
      DeleteApp(ctx context.Context, app *entities.App) error
  }
  ```
- Confirm `*Deployer` satisfies it (it already has both methods) with a
  `var _ Backend = (*Deployer)(nil)` assertion.
- Change `NewPipeline(deployer, ...)` and `NewAppHandlerV2(..., deployer, ...)` to
  accept `Backend` instead of `*Deployer`. Verify build + tests.

### Task 2: Shared env-var resolution helper
- Extract the env-var fetch+decrypt block from `DeployApp` (deployer.go:66-118) into
  a reusable function `resolveEnvVars(ctx, app, envVarRepo, appRepo, envCrypto)
  []entities.EnvVar` so both backends share it. Keep k8s Deployer behavior identical.

### Task 3: DockerDeployer core
- New file `deploy/docker_deployer.go`. Uses the already-present Docker SDK
  (`github.com/docker/docker/client`, v28.5.2 — promote to direct dep).
- `DeployApp`:
  1. `resolveEnvVars(...)` (Task 2).
  2. Pull `imageTag` (with app.RegistryUser/Password if set).
  3. Remove any existing container named `zenith-app-<app.Subdomain>`.
  4. `ContainerCreate` + `ContainerStart`: the image, env vars, restart=unless-stopped,
     joined to the compose network, labeled for caddy-docker-proxy:
     - `caddy = <app.Subdomain>.<baseDomain>` (+ one per active custom domain)
     - `caddy.reverse_proxy = {{upstreams <app.Port>}}`
     - `zenith.app.id = <app.ID>` (for lookup/cleanup)
  5. Optional: apply plan CPU/mem limits (HostConfig.Resources).
- `DeleteApp`: stop + remove the container by name/label.
- `var _ Backend = (*DockerDeployer)(nil)`.

### Task 4: Wire selection in main.go
- Around main.go:512, choose the backend:
  ```go
  var backend deploy.Backend
  if cfg.Mode == "saas" {
      backend = deploy.NewDeployer(k8sClient, appRepo, planRepo, cfg.BaseDomain)  // + setters
  } else {
      backend = deploy.NewDockerDeployer(dockerClient, appRepo, envVarRepo, planRepo, cfg.BaseDomain)
  }
  ```
- DockerDeployer needs the same repos/crypto the k8s one gets via setters — pass them
  in the constructor. The Docker client mounts the host socket (compose change below).

### Task 5: Compose + Caddy for the CE
- Mount the Docker socket into `api` (read-only is not enough — deploy needs write):
  `/var/run/docker.sock:/var/run/docker.sock`.
- Swap the `caddy` service image to `lucaslorentz/caddy-docker-proxy:2-alpine`, mount
  the socket (read-only), and drop the static Caddyfile (labels drive config). Keep it
  under the `tls` profile; for local/no-domain, caddy-docker-proxy still routes on :80.
- App containers must join the same network caddy watches — set an explicit network
  name so both api-launched containers and caddy share it.

### Task 6: End-to-end test (Vagrant VM)
- `vagrant up`; inside, run the installer; log in; create an app pointing at a public
  sample image (e.g. `traefik/whoami` or `nginxdemos/hello`) with its port; deploy.
- Verify: a `zenith-app-<sub>` container is running, and `http://<sub>.<host>` (nip.io
  against the VM IP) returns the app. This is the real "customer deploys a project" test.

## Out of scope (v2)
Build-from-git (`docker build` on the host), replicas/scaling, per-app managed
databases, health-gated rollout. v1 = "run my image, give me a URL."

## Notes
- Docker SDK is already an (indirect) dependency — no new heavy dep.
- The k8s path is untouched; this is purely additive + the interface extraction.
- Vagrantfile (committed) is the test harness.
