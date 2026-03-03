# 24 — CI/CD Pipeline

> **Purpose:** Understand how code goes from a git push to running in the cluster — the full build, test, security scan, and deployment pipeline.
> **Audience:** Any developer who needs to understand why a build failed, how images are tagged, or how deployment works.
> **Last Updated:** 2026-03-03
> **Related:** [15-argocd-gitops.md](./15-argocd-gitops.md) (ArgoCD deployment), [22-day-to-day-operations.md](./22-day-to-day-operations.md) (deploy workflow), [SYSTEM-MAP.md](./SYSTEM-MAP.md) (build flow diagram)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Pipeline Architecture](#2-pipeline-architecture)
3. [Workflow: Build Images](#3-workflow-build-images)
4. [Workflow: Build Helm Chart](#4-workflow-build-helm-chart)
5. [Workflow: Security Scan](#5-workflow-security-scan)
6. [Workflow: Terraform](#6-workflow-terraform)
7. [Image Tagging Strategy](#7-image-tagging-strategy)
8. [From CI to Cluster (ArgoCD)](#8-from-ci-to-cluster-argocd)
9. [Running CI Locally](#9-running-ci-locally)
10. [Troubleshooting](#10-troubleshooting)

---

## 1. Overview

Zenith uses **GitHub Actions** for CI (build + test + scan) and **ArgoCD** for CD (deploy). They are decoupled — CI pushes images to Harbor, ArgoCD detects new images and deploys them.

```
┌─────────────────────────────────────────────────────────────────────────┐
│             CI/CD OVERVIEW                                               │
│                                                                          │
│  Developer                                                               │
│  ┌──────────┐                                                           │
│  │ git push │                                                           │
│  │ to main  │                                                           │
│  └────┬─────┘                                                           │
│       │                                                                  │
│       ▼                                                                  │
│  GitHub Actions (CI)                                                     │
│  ┌──────────────────────────────────────────────────────────────┐       │
│  │                                                               │       │
│  │  Gate 1: TEST                                                │       │
│  │  ┌───────────────────────────────────────────────────────┐   │       │
│  │  │ Go tests (race detector)  +  pnpm lint (Turbo)        │   │       │
│  │  └───────────────────────────┬───────────────────────────┘   │       │
│  │                              │ pass                           │       │
│  │                              ▼                                │       │
│  │  Gate 2: SECURITY                                             │       │
│  │  ┌───────────────────────────────────────────────────────┐   │       │
│  │  │ Semgrep SAST (Go, TS, Docker, K8s, OWASP, Secrets)   │   │       │
│  │  └───────────────────────────┬───────────────────────────┘   │       │
│  │                              │ pass                           │       │
│  │                              ▼                                │       │
│  │  PREPARE                                                      │       │
│  │  ┌───────────────────────────────────────────────────────┐   │       │
│  │  │ Extract version from Chart.yaml + generate git SHA    │   │       │
│  │  └───────────────────────────┬───────────────────────────┘   │       │
│  │                              │                                │       │
│  │                              ▼                                │       │
│  │  BUILD & PUSH (5 parallel jobs)                               │       │
│  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐              │       │
│  │  │ API  │ │ Land │ │ Web  │ │ MC   │ │ Op   │              │       │
│  │  │      │ │ ing  │ │+Demo │ │+Demo │ │ erat │              │       │
│  │  └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘              │       │
│  │     │        │        │        │        │                    │       │
│  │     └────────┴────────┴────────┴────────┘                    │       │
│  │                       │                                       │       │
│  └───────────────────────┼───────────────────────────────────────┘       │
│                          │ push images                                   │
│                          ▼                                               │
│  Harbor Registry (registry.stage.freezenith.com)                        │
│  ┌──────────────────────────────────────────────────────────────┐       │
│  │  zenith-stage/zenith-api:0.3.0                                │       │
│  │  zenith-stage/zenith-landing:0.3.0                            │       │
│  │  zenith-stage/zenith-web:0.1.0                                │       │
│  │  zenith-stage/zenith-web-demo:0.1.0                           │       │
│  │  zenith-stage/zenith-mc:0.1.0                                 │       │
│  │  zenith-stage/zenith-mc-demo:0.1.0                            │       │
│  │  zenith-stage/zenith-operator:0.2.0                           │       │
│  └──────────────────────────┬───────────────────────────────────┘       │
│                             │ Image Updater polls                        │
│                             ▼                                            │
│  ArgoCD (CD) — in zen-stage cluster                                     │
│  ┌──────────────────────────────────────────────────────────────┐       │
│  │  Image Updater detects new tag                                │       │
│  │  → Updates Helm values                                        │       │
│  │  → ArgoCD syncs                                               │       │
│  │  → Pods restart with new image                                │       │
│  └──────────────────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Pipeline Architecture

### Workflow Files

| File | Trigger | Purpose |
|------|---------|---------|
| `.github/workflows/build-images.yml` | Push to `main` (apps/, services/) | Build + push Docker images |
| `.github/workflows/build-chart.yml` | Push to `main` (infra/helm/zenith/) | Package + push Helm chart |
| `.github/workflows/security.yml` | Push + PR to `main` | Semgrep SAST security scan |
| `.github/workflows/terraform.yml` | Push + PR (infra/terraform/) | Terraform plan/apply |

### GitHub Secrets

| Secret | Purpose |
|--------|---------|
| `HARBOR_HOST` | Registry hostname (e.g., `registry.stage.freezenith.com`) |
| `HARBOR_ROBOT_USER` | Robot account for image push |
| `HARBOR_ROBOT_TOKEN` | Robot account token |
| `KUBECONFIG_STAGING` | kubeconfig for staging cluster |
| `TF_VAR_cloudflare_api_token` | Cloudflare API token |
| `TF_VAR_hcloud_token` | Hetzner Cloud API token |

---

## 3. Workflow: Build Images

The main CI pipeline. Triggered on push to `main` when app or service files change.

```
┌─────────────────────────────────────────────────────────────────────────┐
│             BUILD IMAGES WORKFLOW (build-images.yml)                      │
│                                                                          │
│  Trigger: push to main (paths: apps/, services/, packages/,             │
│           package.json, pnpm-lock.yaml) + manual dispatch               │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  JOB: test (Gate 1)                                              │   │
│  │  ┌────────────────────────────────────────────────────────────┐  │   │
│  │  │  Go tests:                                                  │  │   │
│  │  │    cd services/api && go test ./... -race -count=1         │  │   │
│  │  │                                                             │  │   │
│  │  │  Node.js lint:                                              │  │   │
│  │  │    pnpm install --frozen-lockfile                          │  │   │
│  │  │    pnpm lint                    (Turbo: all apps)          │  │   │
│  │  └────────────────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────┬───────────────────────────────────┘   │
│                                 │ must pass                              │
│                                 ▼                                        │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  JOB: security (Gate 2)                                          │   │
│  │  ┌────────────────────────────────────────────────────────────┐  │   │
│  │  │  Semgrep scan with rules:                                   │  │   │
│  │  │    p/default, p/golang, p/typescript, p/nextjs,            │  │   │
│  │  │    p/docker, p/kubernetes, p/secrets, p/owasp-top-ten      │  │   │
│  │  │                                                             │  │   │
│  │  │  Paths scanned: apps/, services/, packages/                │  │   │
│  │  └────────────────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────┬───────────────────────────────────┘   │
│                                 │ must pass                              │
│                                 ▼                                        │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  JOB: prepare                                                    │   │
│  │  ┌────────────────────────────────────────────────────────────┐  │   │
│  │  │  VERSION = grep appVersion infra/helm/zenith/Chart.yaml    │  │   │
│  │  │  SHA = git rev-parse --short HEAD                          │  │   │
│  │  │                                                             │  │   │
│  │  │  Outputs: version=0.3.0, sha=abc1234                      │  │   │
│  │  └────────────────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────┬───────────────────────────────────┘   │
│                                 │                                        │
│                                 ▼                                        │
│  ┌────────────────── BUILD JOBS (parallel) ─────────────────────────┐   │
│  │                                                                   │   │
│  │  ┌──────────────┐  Each job:                                     │   │
│  │  │ build-api    │  1. docker buildx build                        │   │
│  │  │ build-landing│  2. Push 3 tags: {VERSION}, sha-{SHA}, latest │   │
│  │  │ build-web    │  3. Web + MC also build "-demo" variant        │   │
│  │  │ build-mc     │                                                 │   │
│  │  │ build-operator│  Registry: HARBOR_HOST/zenith-stage/{name}    │   │
│  │  └──────────────┘                                                 │   │
│  │                                                                   │   │
│  │  Build args (staging-specific):                                   │   │
│  │  • Landing: NEXT_PUBLIC_APP_URL=https://app.stage.freezenith.com │   │
│  │  • Web:     NEXT_PUBLIC_LANDING_URL=https://stage.freezenith.com │   │
│  │  • MC:      NEXT_PUBLIC_LANDING_URL=https://stage.freezenith.com │   │
│  └───────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Workflow: Build Helm Chart

```
┌─────────────────────────────────────────────────────────────────────────┐
│             HELM CHART WORKFLOW (build-chart.yml)                         │
│                                                                          │
│  Trigger: push to main (paths: infra/helm/zenith/**)                    │
│                                                                          │
│  Steps:                                                                  │
│  1. Helm lint     → validates Chart.yaml + templates                    │
│  2. Helm template → dry-run render with dummy values                    │
│  3. Helm package  → creates zenith-{VERSION}.tgz                        │
│  4. Helm push     → pushes to Harbor OCI registry                       │
│                      oci://registry.stage.freezenith.com/zenith-stage   │
│                                                                          │
│  ArgoCD can pull Helm charts from Harbor OCI registry.                  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Workflow: Security Scan

```
┌─────────────────────────────────────────────────────────────────────────┐
│             SECURITY SCAN WORKFLOW (security.yml)                         │
│                                                                          │
│  Trigger: push to main, PRs to main, manual dispatch                    │
│                                                                          │
│  Tool: Semgrep (SAST — Static Application Security Testing)             │
│                                                                          │
│  Rule packs:                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  p/default       → General security patterns                       │ │
│  │  p/golang        → Go-specific: sql injection, path traversal     │ │
│  │  p/typescript    → TS-specific: XSS, prototype pollution          │ │
│  │  p/nextjs        → Next.js: SSRF, exposed API routes             │ │
│  │  p/docker        → Dockerfile: running as root, secrets in layers │ │
│  │  p/kubernetes    → K8s: privileged containers, missing limits     │ │
│  │  p/secrets       → Hardcoded passwords, API keys, tokens          │ │
│  │  p/owasp-top-ten → OWASP: injection, XSS, SSRF, etc.            │ │
│  │  p/terraform     → Terraform: insecure configs, missing encryption│ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  Output: semgrep-results.json (artifact, retained 30 days)              │
│  If findings: workflow fails → block merge                              │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Workflow: Terraform

```
┌─────────────────────────────────────────────────────────────────────────┐
│             TERRAFORM WORKFLOW (terraform.yml)                            │
│                                                                          │
│  Trigger: push/PR to infra/terraform/** + manual dispatch               │
│                                                                          │
│  Two jobs (sequential):                                                  │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  JOB 1: terraform-staging (Phase 1 — Server + DNS)                 │ │
│  │  Directory: infra/terraform/staging/                               │ │
│  │                                                                    │ │
│  │  • Creates/updates Hetzner VMs                                    │ │
│  │  • Creates/updates Cloudflare DNS records                         │ │
│  │  • Uses TF_VAR_cloudflare_api_token + TF_VAR_hcloud_token        │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │ depends on                                    │
│                          ▼                                               │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  JOB 2: terraform-staging-k8s (Phase 3 — Helm Releases)           │ │
│  │  Directory: infra/terraform/staging-k8s/                           │ │
│  │                                                                    │ │
│  │  • Installs all Helm releases (APISIX, ArgoCD, CNPG, etc.)       │ │
│  │  • Uses KUBECONFIG_STAGING secret for cluster access              │ │
│  │  • Sets registry credentials, JWT secret, admin password, etc.    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  Manual dispatch allows choosing: plan (preview) or apply (execute)     │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Image Tagging Strategy

```
┌─────────────────────────────────────────────────────────────────────────┐
│             IMAGE TAG STRATEGY                                           │
│                                                                          │
│  Each build produces 3 tags per image:                                  │
│                                                                          │
│  1. VERSION TAG:   zenith-api:0.3.0                                     │
│     Source: infra/helm/zenith/Chart.yaml → appVersion field             │
│     Stable: same tag across CI runs until version bumped                │
│                                                                          │
│  2. SHA TAG:       zenith-api:sha-abc1234                               │
│     Source: git rev-parse --short HEAD                                   │
│     Unique: every commit gets a unique tag                              │
│     ArgoCD Image Updater tracks this pattern for auto-deploy            │
│                                                                          │
│  3. LATEST TAG:    zenith-api:latest                                    │
│     Always points to the most recent build                              │
│     Used for quick testing, NOT for production                          │
│                                                                          │
│  DEMO VARIANTS (web + mission-control only):                            │
│  zenith-web-demo:0.1.0       (NEXT_PUBLIC_DEMO_MODE=true)              │
│  zenith-mc-demo:0.1.0        (NEXT_PUBLIC_DEMO_MODE=true)              │
│                                                                          │
│  VERSION BUMP PROCESS:                                                   │
│  1. Edit infra/helm/zenith/Chart.yaml → appVersion: "0.4.0"            │
│  2. Commit and push to main                                             │
│  3. CI reads new version → tags images as 0.4.0                        │
│  4. ArgoCD Image Updater picks up the new tag                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 8. From CI to Cluster (ArgoCD)

```
┌─────────────────────────────────────────────────────────────────────────┐
│             CI → CD HANDOFF                                              │
│                                                                          │
│  CI (GitHub Actions)                     CD (ArgoCD in cluster)          │
│  ──────────────────                     ──────────────────────          │
│                                                                          │
│  1. Build image                                                          │
│  2. Push to Harbor                                                       │
│     zenith-api:sha-abc1234                                              │
│                                          3. Image Updater polls Harbor  │
│                                             every 2 minutes             │
│                                                                          │
│                                          4. Detects new sha-* tag       │
│                                                                          │
│                                          5. Updates annotation on       │
│                                             ArgoCD Application with     │
│                                             new image tag               │
│                                                                          │
│                                          6. ArgoCD detects diff         │
│                                             (desired ≠ live)            │
│                                                                          │
│                                          7. ArgoCD syncs:               │
│                                             helm template → apply       │
│                                                                          │
│                                          8. Kubernetes rolling update   │
│                                             old pods → new pods         │
│                                                                          │
│  No human intervention between push to main and deploy to staging.     │
│  The pipeline is: git push → CI → Harbor → Image Updater → ArgoCD.     │
│                                                                          │
│  IMPORTANT: ArgoCD watches the `staging` branch for Helm values.        │
│  You must merge main → staging to update non-image config.              │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Running CI Locally

Use `act` (GitHub Actions local runner) or the Makefile.

```bash
# Via Makefile (recommended — simpler)
make test          # Run Go tests + Node lint
make build-api     # Build API Docker image
make build-landing # Build landing Docker image
make push-all      # Push all images to Harbor

# Via act (full GitHub Actions simulation)
make ci-images     # Simulate build-images.yml
make ci-chart      # Simulate build-chart.yml
make ci-all        # Run everything

# Manual Docker builds
docker buildx build --platform linux/amd64 \
  -t zenith-api:local \
  -f services/api/Dockerfile \
  --load .

docker buildx build --platform linux/amd64 \
  -t zenith-web:local \
  --build-arg NEXT_PUBLIC_API_URL=http://localhost:8080 \
  -f apps/web/Dockerfile \
  --load ./apps/web
```

---

## 10. Troubleshooting

### CI build failed — tests

```bash
# Run tests locally to reproduce:
cd services/api && go test ./... -race -count=1 -v
pnpm lint

# Check the GitHub Actions log for the specific test failure
```

### CI build failed — security scan

```bash
# View the Semgrep findings in the GitHub Actions artifact
# Download semgrep-results.json from the workflow run

# Run Semgrep locally:
pip install semgrep
semgrep --config=p/golang --config=p/typescript apps/ services/
```

### Image not appearing in Harbor

```bash
# Check Harbor robot account credentials
# Secrets: HARBOR_HOST, HARBOR_ROBOT_USER, HARBOR_ROBOT_TOKEN

# Verify manually:
docker login $HARBOR_HOST -u $HARBOR_ROBOT_USER -p $HARBOR_ROBOT_TOKEN
docker push $HARBOR_HOST/zenith-stage/zenith-api:test
```

### ArgoCD not picking up new image

```bash
# Check Image Updater logs
kubectl -n argocd logs deploy/argocd-image-updater --tail=50

# Check if the application has image updater annotations
kubectl -n argocd get app zenith-api -o yaml | grep image-updater

# Force sync manually
argocd app sync zenith-api --force
# Or via UI: https://argocd.stage.freezenith.com → zenith-api → Sync
```

### Build is slow

```bash
# Docker buildx uses build cache by default
# First build is slow (~5-10 min), subsequent builds are faster (~2-3 min)

# For Go builds: ensure go mod cache is warm
go mod download

# For Next.js builds: the .next/cache directory speeds up rebuilds
# This is handled automatically in CI via GitHub Actions cache
```
