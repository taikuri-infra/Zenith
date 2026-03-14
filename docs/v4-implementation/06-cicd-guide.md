# 06 — CI/CD & Deployment Guide

> **Read time:** 45 minutes
> **Prerequisite:** [05 — Infrastructure Guide](./05-infrastructure.md)
> **Next:** [07 — Day-2 Operations](./07-day2-operations.md)

---

## CI/CD Overview

```
                    ┌────────────────────────┐
                    │   GITHUB REPOSITORY     │
                    │   (main + staging)       │
                    └────────┬───────────────┘
                             │
          ┌──────────────────┼──────────────────┐
          │                  │                   │
    ┌─────▼─────┐    ┌──────▼──────┐    ┌──────▼──────┐
    │ PR/Push   │    │ Staging     │    │ Production  │
    │ CI Tests  │    │ Deploy      │    │ Release     │
    │           │    │             │    │             │
    │ ci.yml    │    │ deploy-     │    │ release.yml │
    │ security  │    │ staging.yml │    │ promote-to- │
    │ .yml      │    │             │    │ prod.yml    │
    └───────────┘    └──────┬──────┘    └──────┬──────┘
                            │                   │
                     ┌──────▼──────┐    ┌──────▼──────┐
                     │ SHA tags    │    │ Semver tags  │
                     │ sha-abc1234│    │ v0.9.0      │
                     │ → Harbor   │    │ → Harbor    │
                     └──────┬──────┘    └─────────────┘
                            │
                     ┌──────▼──────┐
                     │ ArgoCD Sync │
                     │ (auto)      │
                     └─────────────┘
```

---

## 11 Workflow Files

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `ci.yml` | PR, push | Tests, lint, Helm validate, Semgrep, commitlint |
| `deploy-staging.yml` | `workflow_dispatch` | Build + push SHA tag + update Helm values + push staging |
| `build-images.yml` | Called by release.yml | Multi-image build + push + sign + scan |
| `release.yml` | Push to `main` | Release Please semver + build + merge to staging |
| `promote-to-prod.yml` | `workflow_dispatch` | Retag staging→prod + cosign verify |
| `smoke-test.yml` | Scheduled/manual | Customer + owner + infra E2E tests |
| `terraform.yml` | Push to main/staging | Terraform plan (+ apply with approval) |
| `security.yml` | PR | Semgrep SAST scan |
| `security-audit.yml` | Scheduled | Trivy + dependency audit + SBOM |
| `load-test.yml` | Manual | k6 load test (1000 req/s) |
| `build-chart.yml` | Called | Helm package + push to Harbor OCI |

---

## The 3 Deployment Flows

### Flow A: Daily Staging Development (Most Common)

```bash
# 1. Write code on staging branch
git checkout staging
# ... make changes ...

# 2. Deploy via make command (triggers GitHub Action locally via act)
make deploy-api    # or deploy-web, deploy-mc, deploy-landing

# What happens behind the scenes:
# - act runs deploy-staging.yml locally
# - Tests run (go test, next lint)
# - Docker build with SHA tag (zenith-api:sha-abc1234)
# - Push to Harbor (registry.stage.freezenith.com)
# - Update values-staging.yaml with new SHA
# - Commit and push to staging branch
# - ArgoCD detects change → syncs → new pods roll out
```

**CRITICAL RULE:** Code MUST be committed and pushed to `staging` branch BEFORE running `make deploy-*`. ArgoCD watches `staging`, not your local files.

### Flow B: Formal Release (main → staging → prod)

```bash
# 1. Merge staging to main (or push conventional commits to main)
git checkout main
git merge staging
git push origin main

# 2. Release Please automatically:
#    - Analyzes conventional commits (feat:, fix:, chore:)
#    - Calculates next semver (e.g., 0.9.0 → 0.10.0)
#    - Creates release PR with CHANGELOG
#    - When PR merged: creates GitHub Release + git tag

# 3. release.yml automatically:
#    - Builds images with semver tag (zenith-api:0.10.0)
#    - Pushes to Harbor
#    - Merges main → staging
#    - Updates values-staging.yaml
#    - ArgoCD syncs staging

# 4. Verify on staging, then promote:
gh workflow run promote-to-prod.yml -f version=0.10.0
```

### Flow C: Emergency Hotfix

```bash
# 1. Fix directly on staging
git checkout staging
# ... fix ...
git commit -m "fix: critical bug in auth"
git push origin staging

# 2. Deploy immediately
make deploy-api

# 3. Later: cherry-pick to main
git checkout main
git cherry-pick <sha>
git push origin main
```

---

## Versioning Strategy (IMPORTANT)

| Environment | Tag Format | Example | Source of Truth |
|-------------|-----------|---------|----------------|
| **Staging** | SHA-based | `zenith-api:sha-abc1234` | Git commit SHA |
| **Production** | Semver | `zenith-api:0.9.0` | Release Please |

**NEVER** use arbitrary version numbers like `0.8.6` for staging. SHA is the truth.

**Why?** SHA tags are:
- Traceable (you know exactly which commit)
- Unique (no collision)
- Zero version management (no "what version should this be?")

---

## Smoke Tests

### 3 Test Suites

| Suite | Tests | What It Checks |
|-------|-------|---------------|
| Customer | 79 PASS | Auth, plan, projects, apps, databases, storage, gateways, webhooks, MFA, compliance |
| Owner | 47 PASS | Admin dashboard, support, audit, modules, infrastructure, settings, customers |
| Infrastructure | 32 PASS | DNS resolution, HTTPS, redirects, SSL certs, API endpoints |

### Running Smoke Tests

```bash
# Manually
cd infra/scripts
./smoke-test-customer.sh
./smoke-test-owner.sh
./smoke-test-infra.sh

# Via GitHub Actions
gh workflow run smoke-test.yml
```

### Smoke Test Gotchas

- **APISIX rate limit:** 100 req/60s per IP. Tests include 429 retry helper (5s backoff)
- **Avoid `!` in test passwords** — zsh/SSH escaping mangles `!` to `\!` in curl JSON
- **CI user:** `smoke-ci@zenith.dev` / `SmokeTest1234`
- **Image caching:** `imagePullPolicy: IfNotPresent` means same tag won't re-pull — always bump tag

---

## Makefile Commands

```bash
# Build
make build              # Build all Docker images
make build-api          # Build API image only
make build-web          # Build Web image only

# Push
make push-api           # Push to Harbor
make push-all           # Push all images

# Deploy (via act — local CI)
make deploy-api         # Build + push + update Helm + commit to staging
make deploy-web
make deploy-mc
make deploy-all

# Test
make test               # Run all tests
make test-api           # Go tests only
make lint               # Lint everything

# Helm
make chart-lint         # Lint Helm charts
make chart-push         # Push charts to Harbor OCI

# Terraform
make tf-plan            # terraform plan for staging-k8s
make tf-apply           # terraform apply

# CI (full pipeline locally)
make ci                 # Run tests locally
make ci-all             # Full CI: images + chart + terraform
```

---

## GitHub Secrets Required

| Secret | Purpose |
|--------|---------|
| `HARBOR_HOST` | registry.stage.freezenith.com |
| `HARBOR_USERNAME` | Harbor robot account |
| `HARBOR_PASSWORD` | Harbor robot password |
| `SMOKE_TEST_EMAIL` | smoke-ci@zenith.dev |
| `SMOKE_TEST_PASSWORD` | SmokeTest1234 |
| `STAGING_ADMIN_EMAIL` | admin@freezenith.com |
| `STAGING_ADMIN_PASSWORD` | Admin password |
| `SLACK_WEBHOOK_URL` | (optional) Slack notifications |
| `TELEGRAM_BOT_TOKEN` | (optional) Telegram notifications |
| `TELEGRAM_CHAT_ID` | (optional) Telegram chat |

---

## Dockerfiles

| App | Dockerfile | Base Image | Build Strategy |
|-----|-----------|-----------|----------------|
| zenith-api | `services/api/Dockerfile` | golang:1.22 → distroless | Multi-stage, build from repo root |
| zenith-web | `apps/web/Dockerfile` | node:20-alpine | Multi-stage, standalone output |
| zenith-mc | `apps/mission-control/Dockerfile` | node:20-alpine | Multi-stage, standalone output |
| zenith-landing | `apps/landing/Dockerfile` | node:20-alpine | Multi-stage, standalone output |
| zenith-operator | `services/operator/Dockerfile` | golang:1.22 → distroless | Multi-stage |

**Important:** API Dockerfile builds from repo root context (`docker build -f services/api/Dockerfile .`), not from within the service directory.

---

**Next → [07 — Day-2 Operations](./07-day2-operations.md)**
