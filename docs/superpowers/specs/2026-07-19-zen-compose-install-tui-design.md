# Zen Compose Install TUI + AI CLI Handoff

Status: Draft (v2 — ease-of-use upgrade)
Date: 2026-07-19

## Motivation

FreeZenith's self-hosted edition (Postgres + API + Web + RustFS, `docker-compose.yml`)
is currently installed via a plain bash script (`infra/scripts/install.sh`) that clones
the repo and runs `docker compose up -d --build` with no interactive feedback, no
provider flexibility beyond "you already have Docker," and no bridge into ongoing
management.

Separately, the `zen` CLI (`cli/`, Go, Cobra) already has a polished install
experience — but only for the Hetzner-provisioned Kubernetes/Mission Control path
(`cli/cmd/install/install.go`, `cli/internal/install/installer.go`). It has no
concept of installing the Compose edition.

**The bar for this spec is not "it works," it's "a person with a freshly bought
server gets a live, HTTPS-secured, working app in one sitting, with nothing to
figure out."** That standard drove every decision below — most importantly, it's
why this version removes the domain/DNS requirement entirely instead of just
automating it for people who already have one.

## Goals

- `zen install` gains a Compose-target path: pick "existing server" (any provider,
  any Linux box reachable over SSH, min 20GB RAM recommended) or "local," SSH in
  (or run locally), install Docker if missing, run the `docker-compose.yml` stack
  (Postgres, API, Web, RustFS), and report progress as a real Bubble Tea TUI.
- **Every install gets a live HTTPS URL with zero domain/DNS knowledge required.**
  The installer detects the server's public IP, registers a subdomain under a
  domain FreeZenith operates (e.g. `pretty-falcon.apps.freezenith.com`), and the
  stack auto-provisions a real Let's Encrypt certificate for it — no manual DNS
  records, no "add these A records" screen. A custom domain is something you
  attach later, not a blocker to seeing your app live.
- Install and first deploy are **one continuous session**, not two: the wizard
  doesn't stop at "installed," it rolls straight into "deploy your first app"
  (including the AI-CLI handoff), ending in a designed, distinct "you're live"
  screen — not a printed credentials block.
- A `--dry-run` preview shows exactly what will happen before any real action is
  taken.
- Transient failures (a flaky pull, DNS propagation lag) auto-retry with backoff
  before surfacing an error to the user.
- The Compose path supports `--resume`, reusing the existing `installstate`
  pattern from the Hetzner path — a failure never means starting completely over.
- `zen uninstall` cleanly tears the stack down — ease of use includes ease of
  reversal.
- After install succeeds, the CLI automatically performs the equivalent of `zen
  login`, and writes credentials to a local `.zen/credentials` file (not just a
  terminal you have to copy from) — so `zen status` / `zen logs` / `zen deploy`
  work immediately with no extra manual step and no fumbled copy-paste.
- Distributed the same way the Hetzner path already is: a single static Go binary,
  no interpreter or runtime dependency on the target machine.

## Non-goals (out of scope for this spec)

- No Tkinter/Electron/desktop GUI. `apps/web` (the existing Next.js dashboard) is
  already the graphical management surface; this spec does not duplicate it.
- No MCP server. Programmatic AI-agent access to a running instance is a separate,
  later surface — this spec covers the human-facing installer/CLI only.
- No Kubernetes/Enterprise edition changes. The existing Hetzner+K8s `zen install`
  path is untouched; this adds a second, independent path for the Compose target.
- Custom-domain attachment (pointing your own domain at an existing install) is
  not designed here — v1 ships every install on its free `*.apps.freezenith.com`
  subdomain; "bring your own domain afterward" is a fast-follow, not v1.
- Does not fix the Phase 2 installer bugs already tracked in
  `docs/superpowers/plans/2026-06-05-freezenith-phase2.md` — those are specific to
  the Hetzner/K8s path and are a separate, already-planned effort.

## Existing building blocks this reuses

- `cli/go.mod` already depends on `charmbracelet/bubbletea`, `charmbracelet/huh`,
  and `charmbracelet/lipgloss` — no new TUI dependency needed.
- `cli/cmd/install/install.go` already has the Cobra command, flag-parsing
  scaffolding, and a styled step-runner (`runSteps`) this follows the same
  pattern as.
- `cli/internal/installstate` already implements the completed-step persistence
  used for `--resume` on the Hetzner path — reused as-is for the Compose path
  rather than building a second mechanism.
- `cli/cmd/login/login.go` is already generic — works against any Zenith API,
  self-hosted or cloud. The new flow calls this same logic programmatically.
- `cli/cmd/status`, `logs`, `deploy`, `db`, `backup` all talk to the API through
  `cli/internal/api` and are not Hetzner/K8s-specific — they work unmodified
  against a Compose-hosted instance once logged in.
- Cloudflare-managed DNS for `freezenith.com` already exists and is
  Terraform-managed (used today for the SaaS platform's own wildcard app
  routing, `*.apps.stage.freezenith.com`) — the new subdomain-registration piece
  extends the same account/pattern rather than introducing a new DNS provider.
- `docker-compose.yml` (Postgres, API, Web, RustFS, added earlier today) is the
  stack being installed — unchanged by this spec except for the reverse-proxy
  swap described below.

## New components

### 1. Reverse proxy swap: Caddy instead of the commented-out Traefik block

The current `docker-compose.yml` has a Traefik/TLS block entirely commented out,
requiring manual ACME/domain wiring. Replace it with **Caddy**: a single
`caddy:2-alpine` service with automatic HTTPS built in (no ACME email/cert-storage
config to hand-write). Caddy fetches its own Let's Encrypt certificate via HTTP-01
once its hostname resolves to the box's IP — which is exactly what the subdomain
registration step (below) sets up automatically. This is a smaller, more
"just works" fit for a single-box install than Traefik, which is oriented at the
multi-service Kubernetes routing this project already uses elsewhere.

### 2. Subdomain registration service

A small new endpoint (hosted alongside the existing Zenith API, or as a minimal
standalone service) that:
1. Receives the installer's detected public IP and a generated slug
   (`pretty-falcon`).
2. Creates a Cloudflare DNS A record: `pretty-falcon.apps.freezenith.com -> <ip>`.
3. Returns the hostname to the installer, which passes it to Caddy as the site
   address.

**This is real, ongoing infrastructure FreeZenith commits to operating** — it
needs basic abuse protection from day one (rate limit per source IP, and a
cleanup job that removes DNS records for installs that never come up healthy or
are explicitly uninstalled) — not full account/quota management, but not
nothing either. Flagged explicitly in Open Questions below for the implementation
plan to size properly.

### 3. Install target selection

`zen install` gains a `--edition compose|cloud` flag (default: interactive prompt
if neither `--edition` nor the existing Hetzner-specific flags are given),
additive to the existing flag set — current Hetzner/K8s behavior is unchanged
when its flags are passed.

### 4. Compose install steps (single continuous flow)

A new step list (parallel to `GetInstallSteps`), run inside one Bubble Tea
session with `--dry-run` support and per-step auto-retry on transient failure
classes (network timeouts, transient SSH drops, DNS propagation checks):

1. **Preview** (if `--dry-run`): print every step below with what it will do and
   exit without touching anything.
2. Connect — SSH to the target host or detect local execution.
3. Ensure Docker — check for `docker`/`docker compose`; if missing, show a clear
   confirmation ("I'll run get.docker.com's official install script on this
   host — proceed?") before installing. This is a hard-to-reverse,
   affects-the-target-machine action and must not happen silently.
4. Detect public IP + register subdomain (component 2) → get back
   `<slug>.apps.freezenith.com`.
5. Fetch stack — pull `docker-compose.yml`, generate `.env` (random
   `JWT_SECRET`, admin credentials, the registered hostname for Caddy).
6. Run stack — `docker compose up -d`, using **prebuilt images** (not `--build`
   — see Open Questions, this is the fix for the earlier-identified 13-14GB
   build-time RAM spike).
7. Wait for health — poll the API's health endpoint and Caddy's cert issuance,
   with backoff retry, until both are ready.
8. Auto-login — save credentials to `.zen/credentials` locally and to the CLI's
   config, so the CLI is immediately usable with no manual `zen login` step.
9. **Continue straight into first deploy**: prompt to hand off to `claude` /
   `codex` / `copilot` (whichever the user has authenticated) to scaffold and
   deploy a first app against the new instance, or skip.
10. Designed "you're live" screen: the URL, a QR code or clickable link,
    credentials file location — a distinct visual close, not a plain printed box.

Every completed step is persisted via `installstate` (reused from the Hetzner
path), so `zen install --resume` picks up from the last completed step instead
of starting over.

### 5. `zen uninstall`

Tears down the Compose stack (`docker compose down -v`), removes the registered
subdomain via the same registration service, and clears local `.zen` state.

## Data flow (Compose install, happy path)

```
zen install --edition compose [--dry-run]
  -> (dry-run: preview all steps, exit)
  -> prompt: existing server (ssh-host/ssh-user/ssh-key) or local
  -> connect (ssh or local exec, retry w/ backoff on transient failure)
  -> ensure docker (confirm before installing if missing)
  -> detect public IP -> register subdomain -> <slug>.apps.freezenith.com
  -> write .env (JWT_SECRET, admin email/password, hostname)
  -> docker compose up -d  (prebuilt images; Postgres, API, Web, RustFS, Caddy)
  -> wait for API health + Caddy cert issuance (retry w/ backoff)
  -> save credentials to .zen/credentials + CLI config (auto zen login)
  -> prompt: deploy first app now via claude/codex/copilot, or skip
  -> designed "you're live" screen: URL + QR/link + credentials file path
```

Each completed step persists to `installstate`; `zen install --resume` continues
from the last completed step.

## Error handling

- Two error classes, handled differently:
  - **Transient** (network blips, DNS propagation lag, momentary SSH drop):
    auto-retry with exponential backoff, capped attempts, before surfacing
    anything to the user.
  - **Hard failures** (bad credentials, Docker install declined, disk full):
    stop immediately, print the failing step and a specific, actionable message
    — not a generic "step failed."
- Any hard failure leaves `installstate` intact, so `--resume` picks up cleanly.
- `zen uninstall` is the documented recovery path if a user wants to abandon and
  start clean rather than resume.

## Testing

- Unit tests for the new step functions, following the existing pattern in
  `cli/internal/install/installer_test.go`.
- An integration test analogous to
  `cli/internal/install/installer_integration_test.go`, using a local Docker
  target instead of a real remote SSH host.
- A test specifically for `--resume` on the Compose path: kill the process
  mid-install, re-run, assert completed steps are skipped.
- A test for the subdomain registration service's rate limiting.
- Manual verification: run `zen install --edition compose` against a real fresh
  VPS (any provider) before calling this done.

## Open questions (to resolve in the implementation plan, not here)

1. **Prebuilt images.** Step 6 assumes `zenith-api`/`zenith-web` images are
   published (e.g. to GHCR) so install doesn't compile from source on the target
   box. Separate work this flow depends on — needs sequencing.
2. **Subdomain service sizing.** Rate limit thresholds, slug collision handling,
   and the cleanup job's exact trigger (how long before an unhealthy/abandoned
   install's DNS record is reclaimed) need to be sized — not decided here.
3. **Caddy vs. hand-rolled ACME.** Caddy is the recommended default; confirm it
   plays cleanly with RustFS/API/Web all behind it in the same compose network
   before locking this in during implementation.
4. Exact CLI invocation contract for the `claude`/`codex`/`copilot` handoff
   (flags, stdin priming vs. a generated prompt file) needs a short spike —
   each tool's non-interactive/resume flags differ and should be checked against
   current versions before locking the interface.
