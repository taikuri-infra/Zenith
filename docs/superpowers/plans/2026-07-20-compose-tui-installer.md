# FreeZenith Compose-Edition Installer (`zen install --edition compose`) — Build Plan

Date: 2026-07-20
Spec: `docs/superpowers/specs/2026-07-19-zen-compose-install-tui-design.md`

## What changed since the spec (resolves two of its open questions)
- **Reverse proxy is Traefik, not Caddy** (spec Open Q3). The Compose stack migrated
  Caddy → Traefik on 2026-07-20 and is verified end-to-end. The installer wires the
  Traefik `tls` profile + a domain; no Caddy anywhere.
- **Prebuilt images exist and are verified** (spec Open Q1). Multi-arch
  `ghcr.io/taikuri-infra/zenith-{api,web}:latest` (amd64+arm64) run on a clean box —
  step "run stack" uses them, never `--build`.
- The whole Compose stack (postgres, api, web, rustfs, traefik) + `install.sh` are
  proven on a clean Ubuntu 24.04 LTS VM, including the sudo-fallback for a
  freshly-installed Docker group. The CLI installer should reuse that exact logic.

## Reuse (from the existing `zen install` Hetzner/k3s path — already in `cli/`)
- `install.Step{Name, Description, Action func(*Config) error, Duration}` — same type.
- `install.Config` — extend, don't fork.
- `cmd/install/install.go:runSteps` — **edition-agnostic**: it iterates whatever steps
  it's given, handles `--resume` via `installstate.IsStepComplete/MarkStepComplete`,
  and renders progress. Reuse as-is; only the step *list* differs.
- `installstate` (State + Save/Load/MarkStepComplete/IsStepComplete) — resume for free.
- `dialSSH(cfg)` — SSH with retry + host-key TOFU.
- `tui/styles.go` — colors/styles. `huh` wizard pattern. Dry-run = per-Action guard.
- `go.mod` already has bubbletea/huh/lipgloss.
- To add `uninstall`: new `cmd/uninstall/uninstall.go` (`var Cmd`), register in
  `cmd/root/root.go` `init()` (template: `cmd/upgrade`).

## Phase A — Core Compose install (buildable & testable NOW; no new infra)
Ships a working `zen install --edition compose` against a local box or any SSH host,
with a user-provided domain (or localhost). This is the MVP the TUI wraps.

1. **`--edition compose|cloud` flag** on `zen install` (default: interactive prompt).
   `cloud` = existing path, untouched. Add `Edition` + `Target` (local|ssh) +
   `Slug`/`ComposeDomain` fields to `Config`.
2. **Compose wizard branch** (`huh`): target (local / ssh host+user+key), domain
   (bring-your-own or localhost), admin email.
3. **`GetComposeInstallSteps(cfg) []Step`** (parallel to `GetInstallSteps`):
   1. Connect — `dialSSH` or local exec.
   2. Ensure Docker — check `docker`/`compose`; if missing, **confirm** then run
      get.docker.com; carry over the `install.sh` DOCKER_GID + sudo-fallback logic.
   3. Fetch stack + generate `.env` (JWT_SECRET, admin creds, DOCKER_GID, domain).
   4. `docker compose up -d` (prebuilt images; `--profile tls` when a real domain is set).
   5. Wait for health — poll API `/health` + web, backoff retry.
   6. Auto-login — save creds to `~/.zen/credentials` + `installstate`.
   All Actions follow the `if cfg.DryRun { return nil }` guard pattern.
4. **`zen uninstall`** — `docker compose down -v`, clear `~/.zen` state (subdomain
   removal deferred to Phase B).
5. **Tests** — unit per step (pattern: `installer_test.go`); integration against a
   **local Docker** target (pattern: `installer_integration_test.go`); a `--resume`
   test (kill mid-install, re-run, assert completed steps skipped).

## Phase B — Free `*.apps.freezenith.com` subdomain + auto-HTTPS
The spec's headline "zero DNS knowledge" feature. Riskiest (hosted infra), so separate.
- **OPEN/BLOCKER:** the subdomain-registration service needs API control of the
  `freezenith.com` DNS zone. The `CLOUDFLARE_DOTECH` token available today does **not**
  see that zone — need the correct Cloudflare account/token (or pick a base domain we
  do control). Resolve before building.
- Registration endpoint: detect public IP → create `<slug>.apps.freezenith.com` A record
  → return hostname. Rate-limit per source IP; slug-collision handling; cleanup job for
  abandoned/unhealthy installs.
- Wire Traefik ACME (HTTP-01) to the registered hostname — mechanism already verified
  today; only needs the public hostname handed in.

## Phase C — "Show off" polish
- Real Bubble Tea progress model (replace the ANSI `runSteps` overwrite) + a designed
  "you're live" screen (URL, QR code / clickable link, credentials path).
- AI-CLI handoff (claude/codex/copilot) for the first deploy — needs the
  invocation-contract spike (spec Open Q4); gate behind whichever CLI is authenticated.

## Suggested first step
Phase A, tasks 1–3: the `--edition` flag + `GetComposeInstallSteps` with
Connect → Ensure Docker → Fetch/.env → compose up → health → auto-login, TDD against a
local Docker target. That yields a working `zen install --edition compose` we can run on
the same lima VM used today, before any hosted-subdomain infra.
