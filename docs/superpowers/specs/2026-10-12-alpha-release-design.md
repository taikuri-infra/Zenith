# Zenith Alpha Release — Design

- **Date:** 2026-10-12
- **Status:** Draft — awaiting review
- **Scope of this spec:** the alpha program roadmap (all milestones) and the detailed design of sub-project 1 (core audit). Sub-projects 2–5 each get their own spec when they start.

## 1. Product decisions

| Decision | Detail |
|---|---|
| Product | Zenith is **self-hosted only**. Anyone installs it on their own server. There is no hosted SaaS offering. |
| SaaS code | **Kept, not touched.** Billing, Stripe, plans/tiers, customers, email campaigns, Hetzner, autoscaler, Mission Control, and the free `*.apps.freezenith.com` subdomain service stay in the repo. They stay off in `ZENITH_MODE=standalone` and are out of alpha scope. |
| Alpha scope | **Core only, and it must work.** Admin login, app deploy (Docker image or Compose stack), managed databases (Postgres, MySQL, Redis), custom domains with automatic HTTPS, env vars and secrets, deploy status and logs, rollback, and the `zen` CLI. Anything else is hidden from the self-host UI. |
| Runtime order | **Docker Compose first, then k3s.** |

## 2. Alpha exit criteria

A person outside the team, with no help, can do all of the following:

1. Install on a fresh Ubuntu 22.04/24.04 server (2 GB RAM minimum for Compose) with **one command**, and reach the dashboard in **under 15 minutes**.
2. Deploy a sample app (image or compose file) with a Postgres database, attach their own domain, and get HTTPS.
3. Reboot the server and find everything back up, including data.
4. Run `zen upgrade` to the next version with no data loss.
5. Back up the platform and restore it.
6. Do all of the above on **amd64 and arm64**.

Every criterion is covered by an **automated E2E test in CI**. "Works" means the E2E test is green, not that it was checked manually on staging.

## 3. Milestones

| Release | Content | Gate |
|---|---|---|
| `alpha.1` | Compose edition, amd64 + arm64 | Full E2E suite green on a fresh VM |
| `alpha.2` | k3s single-node | **The same** E2E suite green on k3s |
| `alpha.3` | k3s multi-node | E2E suite plus node add and node-loss tests |

## 4. Sub-projects

Each sub-project gets its own spec → plan → implementation cycle.

1. **Core audit and UI cut** (this spec, §6): find out exactly what works today.
2. **Compose hardening**: fix the audit findings, reboot durability, `zen upgrade`, backup/restore, ARM images, and extract runtime interfaces as we go.
3. **E2E and release**: runtime-agnostic E2E suite, run on a fresh VM in CI, multi-arch images and CLI binaries, then tag `alpha.1`.
4. **k3s single-node**: a lightweight self-host Helm path that implements the same interfaces and passes the same E2E suite, then tag `alpha.2`.
5. **k3s multi-node**: tag `alpha.3`.

## 5. Principles that hold across all sub-projects

- **Every runtime-dependent concern sits behind an interface.** App deploy already does (`services/api/internal/services/deploy/backend.go`, `deploy.Backend`, with `Deployer` for Kubernetes and `DockerDeployer` for Docker). Databases, domains/TLS, logs, and backup get the same treatment, with a Docker implementation first. This is what makes the move to k3s an implementation task instead of a rewrite.
- **E2E tests talk only to the API and the CLI**, never to Docker or kubectl directly. One suite serves both runtimes.
- **SaaS code is left alone.** Where it sits in the core path (plan checks, for example), standalone mode routes around it, as `handlers.UnlimitedMode` already does. It is not deleted.

## 6. Sub-project 1 — Core audit and UI cut

### Goal

Produce a factual record of the state of every core feature in the Compose edition. That record is the input to sub-project 2. **Nothing gets fixed in this sub-project**; it only finds and records.

### Existing mechanisms to build on

- API: `ZENITH_MODE` (default `standalone`). SaaS-only wiring in `services/api/cmd/server/main.go` is gated by `cfg.Mode == "saas"`, and `handlers.UnlimitedMode = cfg.Mode != "saas"` disables plan limits.
- Web: sidebar entries carry `saasOnly: true` (`apps/web/src/components/sidebar.tsx`) and are hidden when `NEXT_PUBLIC_ZENITH_MODE=standalone`.
- Install: `infra/scripts/install.sh` and `zen install --edition compose`.

### Method

**A. Static audit (from code)**
- List every API route registered in standalone mode.
- List every web page reachable from the sidebar in standalone mode.
- Find pages that render mock or demo data (`mock-data.ts`, `demo-api.ts`, and the monitoring and logs pages noted in `docs/technical-debt.md`).

**B. Live audit (real install)**
- Target: test server `161.35.82.211` (Ubuntu 24.04, 4 vCPU, 16 GB). Its k3s is stopped for the duration of the audit and disabled before the reboot test. Restore afterwards with `systemctl enable --now k3s`.
- Install exactly as a user would: `curl … install.sh | bash`.
- Walk the core scenario through UI, API, and CLI, recording each step:
  1. Admin login
  2. Deploy from a Docker image
  3. Deploy from a compose file
  4. Create Postgres, MySQL, and Redis and attach them to an app
  5. Env vars and secrets
  6. Custom domain and HTTPS
  7. Deploy logs and status
  8. Rollback
  9. Server reboot, then re-check steps 1–8

**C. UI cut**
- Features visible in standalone that are not core (for example object storage, auth pools, mock monitoring, and gateway) move behind a `ZENITH_LABS` flag, default off. Their code is untouched; they are only hidden.

### Deliverables

1. `docs/alpha/core-audit.md`: a table of feature × (UI, API, CLI) × status (`works` / `broken` / `mock` / `partial`), with reproduction steps for every non-working entry.
2. A GitHub milestone `alpha.1` with one issue per finding.
3. One small PR adding the `ZENITH_LABS` flag and hiding non-core entries.

### Out of scope

Fixing anything found (sub-project 2), new features, and any change to SaaS code.
