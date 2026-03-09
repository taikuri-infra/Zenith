# CI/CD Pipelines

All workflows live in `.github/workflows/`. Run locally with [act](https://github.com/nektos/act).

```bash
# General syntax
act <event> -W .github/workflows/<file>.yml -j <job-name> --container-architecture linux/amd64

# Secrets are loaded from .secrets file automatically
```

---

## Overview

| # | Pipeline | File | Jobs | Auto Trigger | Manual |
|---|----------|------|------|-------------|--------|
| 1 | **Release** | `release.yml` | 3 | Push to `main` | No |
| 2 | **CI** | `ci.yml` | 3 | PR to `main`/`staging` | No |
| 3 | Build & Push Docker Images | `build-images.yml` | 6 | Called by Release workflow | Yes |
| 4 | Build & Push Helm Chart | `build-chart.yml` | 1 | Push to `main` (infra/helm/zenith/) | Yes |
| 5 | Security Analysis | `security.yml` | 1 | Push to `main` + PR to `main` (apps/services/infra) | Yes |
| 6 | Smoke Tests | `smoke-test.yml` | 4 | After Release or Build Images success + Cron (every 6h) | Yes |
| 7 | Terraform | `terraform.yml` | 2 | Push to `main` + PR to `main` (infra/terraform/) | Yes (plan/apply) |
| 8 | Promote to Production | `promote-to-prod.yml` | 3 | Never | Yes (version + components) |

**Composite Action** (for customers, not internal):

| Action | File | Purpose |
|--------|------|---------|
| Zenith Deploy | `.github-actions/zenith-deploy/action.yml` | Customer CI: Docker build + push + register release with Zenith API |

---

## 1. Release (`release.yml`) — NEW

Automated semantic versioning via [Release Please](https://github.com/googleapis/release-please). Runs on every push to `main`.

**Trigger:** Push to `main` (always runs, but only creates releases when conventional commits warrant one).

**Flow:** `release-please` → `build-and-push` (if release created) → `update-staging` (if release created)

| # | Job | Depends On | What It Does |
|---|-----|------------|-------------|
| 1 | `release-please` | — | Creates/updates a Release PR, or creates a GitHub Release + git tag when the Release PR is merged |
| 2 | `build-and-push` | release-please | Calls `build-images.yml` as reusable workflow with the release version |
| 3 | `update-staging` | build-and-push | Merges `main` → `staging`, updates all `values-staging.yaml` image tags, pushes to `staging` |

**How it works:**
1. Developer pushes `feat(api): add billing endpoint` to `main`
2. Release Please creates a PR: "chore(main): release 0.9.0" with CHANGELOG + version bumps
3. Team reviews and merges the Release PR
4. Release Please creates a GitHub Release + tag `v0.9.0`
5. Build jobs build + push all Docker images tagged `0.9.0`
6. Staging branch is updated → ArgoCD auto-syncs

**Config files:**
- `release-please-config.json` — Release type, changelog sections, extra files (Chart.yaml)
- `.release-please-manifest.json` — Current version tracker
- `version.txt` — Simple version file (updated by Release Please)

**Cannot run in act** — Release Please is a GitHub API client. Use the CLI for local testing:
```bash
release-please release-pr --repo-url=https://github.com/DoTechHQ/Zenith --token=$(gh auth token) --dry-run
```

---

## 2. CI (`ci.yml`) — NEW

Runs on all pull requests to `main` or `staging`. Validates code quality before merge.

**Trigger:** PR to `main` or `staging`.

| # | Job | What It Does | act Command |
|---|-----|-------------|-------------|
| 1 | `test` | Go tests (API, CLI, TF provider), pnpm lint, Helm lint | `act pull_request -j test` |
| 2 | `security` | Semgrep SAST scan (OWASP, secrets, k8s, docker) | `act pull_request -j security` |
| 3 | `commitlint` | Validates conventional commit messages | `npx commitlint --from HEAD~5` |

**Note:** These tests used to live inside `build-images.yml` as gates. They were moved here so PRs (including Release PRs) are validated before merge, and the build workflow only builds.

---

## 3. Build & Push Docker Images (`build-images.yml`)

Builds all platform Docker images, pushes to Harbor, scans with Trivy.

**Trigger:** Called by `release.yml` (via `workflow_call`) with a version input. Also available as manual `workflow_dispatch` for emergency builds.

**Flow:** `prepare` → 5 builds (parallel)

| # | Job | Depends On | What It Does | act Command |
|---|-----|------------|-------------|-------------|
| 1 | `prepare` | — | Uses version from input or reads from `Chart.yaml`, generates short SHA | `act workflow_dispatch -j prepare --input version=0.8.0` |
| 2 | `build-api` | prepare | Docker build `zenith-api` + Trivy scan (CRITICAL/HIGH) | `act workflow_dispatch -j build-api --input version=0.8.0` |
| 3 | `build-landing` | prepare | Docker build `zenith-landing` | `act workflow_dispatch -j build-landing --input version=0.8.0` |
| 4 | `build-mc` | prepare | Docker build `zenith-mc` + `zenith-mc-demo` (2 images) | `act workflow_dispatch -j build-mc --input version=0.8.0` |
| 5 | `build-web` | prepare | Docker build `zenith-web` + `zenith-web-demo` (2 images) + Trivy scan | `act workflow_dispatch -j build-web --input version=0.8.0` |
| 6 | `build-operator` | prepare | Docker build `zenith-operator` | `act workflow_dispatch -j build-operator --input version=0.8.0` |

**Images built (8 total):**

| Image | Source | Trivy Scan |
|-------|--------|-----------|
| `zenith-api` | `services/api/Dockerfile` | Yes |
| `zenith-landing` | `apps/landing/Dockerfile` | No |
| `zenith-mc` | `apps/mission-control/Dockerfile` | No |
| `zenith-mc-demo` | `apps/mission-control/Dockerfile` (DEMO_MODE=true) | No |
| `zenith-web` | `apps/web/Dockerfile` | Yes |
| `zenith-web-demo` | `apps/web/Dockerfile` (DEMO_MODE=true) | No |
| `zenith-operator` | `services/operator/Dockerfile` | No |

**Tags per image:** `:<version>`, `:sha-<short>`, `:latest`

---

## 4. Build & Push Helm Chart (`build-chart.yml`)

Packages the umbrella Helm chart and pushes to Harbor OCI registry.

**Trigger:** Push to `main` when `infra/helm/zenith/` changes. Manual.

| # | Job | What It Does | act Command |
|---|-----|-------------|-------------|
| 1 | `build-chart` | Helm lint → template validate → package → push to Harbor OCI | `act push -j build-chart` |

**Steps detail:**
1. `helm lint infra/helm/zenith/` — Validates chart structure
2. `helm template test infra/helm/zenith/ --set ...` — Validates rendered templates
3. `helm package infra/helm/zenith/` — Creates `.tgz`
4. `helm push *.tgz oci://registry.stage.freezenith.com/zenith-stage` — Pushes to Harbor

---

## 5. Security Analysis (`security.yml`)

Standalone Semgrep SAST scan (also runs as a gate inside `build-images.yml`).

**Trigger:** Push to `main` or PR to `main` when `apps/`, `services/`, `packages/`, `infra/helm/`, or `infra/terraform/` change. Manual.

| # | Job | What It Does | act Command |
|---|-----|-------------|-------------|
| 1 | `semgrep` | Semgrep scan with OWASP, secrets, k8s, docker rulesets → upload JSON artifact | `act push -j semgrep` |

**Rulesets:** `p/default`, `p/golang`, `p/typescript`, `p/nextjs`, `p/terraform`, `p/docker`, `p/kubernetes`, `p/secrets`, `p/owasp-top-ten`

---

## 6. Smoke Tests (`smoke-test.yml`)

End-to-end tests against live staging API. Verifies customer userflow, admin userflow, and infrastructure health.

**Trigger:**
- After `Release` or `Build & Push Docker Images` workflow completes successfully
- Scheduled: every 6 hours (`0 */6 * * *`)
- Manual with optional `api_url` input (default: `https://api.stage.freezenith.com`)

**Flow:** 3 tests (parallel) → notify

| # | Job | Depends On | What It Does | act Command |
|---|-----|------------|-------------|-------------|
| 1 | `customer-smoke-test` | — | 79 tests: auth, apps, billing, gateway, storage, settings, MFA, ... | `act workflow_dispatch -j customer-smoke-test --input api_url=https://api.stage.freezenith.com` |
| 2 | `owner-smoke-test` | — | 47 tests: admin login, plans, clusters, tenants, audit, security, ... | `act workflow_dispatch -j owner-smoke-test --input api_url=https://api.stage.freezenith.com` |
| 3 | `infrastructure-e2e` | — | 32 tests: DNS resolution, TLS certs, service health, k8s endpoints | `act workflow_dispatch -j infrastructure-e2e` |
| 4 | `notify` | 1, 2, 3 | Sends Slack + Telegram report with per-job status | — |

**Required secrets:** `SMOKE_TEST_EMAIL`, `SMOKE_TEST_PASSWORD`, `STAGING_ADMIN_EMAIL`, `STAGING_ADMIN_PASSWORD`

**Note:** Infrastructure E2E will fail DNS checks when run locally via act (Docker container can't resolve `*.stage.freezenith.com`). This is expected.

---

## 7. Terraform (`terraform.yml`)

Plans and applies Terraform for staging infrastructure.

**Trigger:**
- PR to `main` when `infra/terraform/` changes → runs `plan` only
- Push to `main` when `infra/terraform/` changes → runs `apply`
- Manual: choose `plan` or `apply`

| # | Job | When | What It Does | act Command |
|---|-----|------|-------------|-------------|
| 1 | `terraform-plan` | PR / manual(plan) | Plan for `staging` + `staging-k8s` (matrix, parallel) → Post diff to PR | `act pull_request -j terraform-plan` |
| 2 | `terraform-apply` | Push main / manual(apply) | Apply `staging` then `staging-k8s` (sequential). Protected by `staging-terraform` environment. | `act push -j terraform-apply` |

**Terraform directories:**

| Directory | What It Manages |
|-----------|----------------|
| `infra/terraform/staging` | Hetzner server, Cloudflare DNS |
| `infra/terraform/staging-k8s` | K8s Helm releases (API, web, APISIX, cert-manager, ...) |

**Required secrets:** `CLOUDFLARE_API_TOKEN`, `HCLOUD_TOKEN`, `HARBOR_HOST`, `HARBOR_ROBOT_USER`, `HARBOR_ROBOT_TOKEN`, `JWT_SECRET`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `DB_PASSWORD`, `KUBECONFIG_STAGING`

---

## 8. Promote to Production (`promote-to-prod.yml`)

Manual-only pipeline for promoting tested staging images to production.

**Trigger:** Manual only. Inputs: `version` (required), `skip_smoke_test` (bool), `components` (all/api-only/web-only/admin-only).

**Flow:** smoke-test → promote-images → deploy (sequential)

| # | Job | Depends On | What It Does | act Command |
|---|-----|------------|-------------|-------------|
| 1 | `staging-smoke-test` | — | Run customer smoke test on staging + verify image exists in Harbor | `act workflow_dispatch -j staging-smoke-test --input version=0.7.24` |
| 2 | `promote-images` | 1 | Docker pull from `zenith-stage` → retag to `zenith-prod` → push | `act workflow_dispatch -j promote-images --input version=0.7.24` |
| 3 | `deploy` | 2 | Update Helm `Chart.yaml` appVersion → git commit → git tag → push | `act workflow_dispatch -j deploy --input version=0.7.24` |

**Protected by:** `production` GitHub environment (requires approval).

---

## Quick Reference: Run All Jobs Locally

```bash
# CI — tests (Go + Node + Helm)
act pull_request -W .github/workflows/ci.yml -j test --container-architecture linux/amd64

# CI — security scan
act pull_request -W .github/workflows/ci.yml -j security --container-architecture linux/amd64

# Commitlint (local, no act needed)
npx commitlint --from HEAD~5

# Release Please dry-run (local, no act — uses GitHub API)
release-please release-pr --repo-url=https://github.com/DoTechHQ/Zenith --token=$(gh auth token) --dry-run

# Build images (manual, with version)
act workflow_dispatch -W .github/workflows/build-images.yml -j build-api --container-architecture linux/amd64 --input version=0.8.0

# Security scan (standalone)
act push -W .github/workflows/security.yml -j semgrep --container-architecture linux/amd64

# Smoke tests (individual)
act workflow_dispatch -W .github/workflows/smoke-test.yml -j customer-smoke-test --container-architecture linux/amd64 --input api_url=https://api.stage.freezenith.com
act workflow_dispatch -W .github/workflows/smoke-test.yml -j owner-smoke-test --container-architecture linux/amd64 --input api_url=https://api.stage.freezenith.com
act workflow_dispatch -W .github/workflows/smoke-test.yml -j infrastructure-e2e --container-architecture linux/amd64

# Terraform plan
act pull_request -W .github/workflows/terraform.yml -j terraform-plan --container-architecture linux/amd64

# Helm chart
act push -W .github/workflows/build-chart.yml -j build-chart --container-architecture linux/amd64
```

**Notes:**
- All commands assume `.secrets` file exists in repo root with required secrets
- Add `--container-architecture linux/amd64` on Apple Silicon (M1/M2/M3)
- `.actrc` configures the default runner image
- Build jobs that push to Harbor will actually push — be careful running `build-*` jobs locally
- Release Please **cannot** run in act (requires GitHub API). Use the CLI `--dry-run` instead
