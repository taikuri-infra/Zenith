# Project Context

## Purpose
Zenith is a **Kubernetes-native Platform-as-a-Service (PaaS)** built on Hetzner Cloud, operated by DoTech as a managed multi-tenant enterprise cloud service. Customers buy "Zenith Enterprise" plans with guaranteed resource ceilings (CPU, RAM, S3, DB storage). CAPI provisions a **dedicated Kubernetes cluster per customer** on Hetzner.

**Future open-core model:** Extract a free, self-hostable version where customers install their own Mission Control. SaaS is the premium managed offering. Codebase supports both modes via `ZENITH_MODE` flag (`saas` vs `standalone`).

**Domain:** freezenith.com

## Tech Stack

### Frontend
- Next.js 15, React 19, TypeScript, Tailwind CSS
- Framer Motion (landing page animations)
- pnpm workspaces + Turborepo

### Backend
- Go 1.25, Fiber v2, JWT (golang-jwt/v5)
- PostgreSQL (persistent state) or in-memory stores (dev mode)
- controller-runtime / kubebuilder (K8s operator)
- Cobra + Charm TUI (CLI)

### Infrastructure
- Hetzner Cloud (k3s v1.34.3, CX22 server)
- CAPI + CAPH (Cluster API Provider Hetzner)
- Traefik 3.5.1 (IngressRoute CRD, NOT standard Ingress)
- cert-manager (letsencrypt-prod, HTTP-01)
- Cloudflare DNS (Terraform + bash script)

### Decided but Not Yet Built
- FluxCD (GitOps), Cilium (CNI/eBPF), Kong (API Gateway DB-less)
- CloudNativePG (per-tenant Postgres), Keycloak (per-tenant IAM)
- Harbor (internal Helm charts only), kube-prometheus-stack
- Kaniko (in-cluster image builds on customer cluster)

## Monorepo Structure

```
Zenith/
apps/
  landing/              # Marketing site (freezenith.com, port 3200/3000)
  mission-control/      # Operator admin panel (ms.{domain}, port 3100)
  web/                  # User dashboard (cloud.{domain}, port 3000)
packages/
  ui/                   # Shared @zenith/ui (cn() utility only for now)
services/
  api/                  # Go REST API (port 8080)
  auth/                 # Go OIDC/SAML auth service (port 8090)
  operator/             # Go K8s operator (controller-runtime)
cli/                    # zen CLI (Go + Cobra + Charm TUI)
infra/
  terraform/            # Cloudflare DNS as code
  ansible/              # Ansible deployment (playbooks, roles, inventory)
  helm/                 # Helm charts (zenith, monitoring)
  k8s/                  # Raw K8s manifests (currently used in prod)
  scripts/              # deploy.sh, cloudflare-dns.sh
```

## Project Conventions

### Code Style
- **Go:** Standard Go formatting (`gofmt`), error wrapping with `fmt.Errorf`, no globals
- **TypeScript:** Strict mode, functional React components, named exports
- **CSS:** Tailwind utility classes, no custom CSS files
- **Naming:** kebab-case files, PascalCase components/types, camelCase functions/variables

### Architecture Patterns
- **Two planes:** Management plane (single k3s) + Workload clusters (CAPI-managed per customer)
- **CRD-driven:** User action -> API creates CRD -> Operator watches -> provisions infrastructure
- **API switching:** `getApi()` returns real or demo API based on build-time `NEXT_PUBLIC_DEMO_MODE`
- **Repository pattern:** Store interfaces with memory + PostgreSQL implementations
- **Backend target:** Clean/Hexagonal architecture (entities, services, ports, adapters, dto). Phase A (entities/dto) done, phases B-D pending.
- **Demo mode:** Build-time env var `NEXT_PUBLIC_DEMO_MODE=true` bakes mock data into JS bundle. Separate Docker images for demo vs production.

### Testing Strategy
- **Go:** `go test ./internal/... -count=1` — unit tests for handlers, stores, deploy pipeline
- **Frontend:** No test framework configured yet
- **Known flaky:** `TestScaleClusterEndpoint`, `TestUpgradeClusterEndpoint` (need CAPI CRDs)
- **Coverage:** 89+ unit tests across deploy engine components

### Git Workflow
- Main branch: `main`
- Feature branches: `feature/description` or `setup/description`
- Commit style: conventional commits (`feat:`, `fix:`, `docs:`, `refactor:`)
- Deploy: `ssh ghasi "cd /opt/zenith && bash infra/scripts/deploy.sh"` (git pull + docker build + k3s import)

## Domain Context

### Business Model
- DoTech runs the management plane. Customers buy Enterprise plans (EUR 2,000-4,000/mo)
- Infrastructure cost ~EUR 553/mo per pool (70-85% margin)
- Resource ceilings guaranteed but provisioned on-demand
- Hetzner cluster autoscaler scales underlying infra
- Billing: Stripe (international) + Toman/IRR (via Fairbroker)

### Key Terminology
- **Mission Control (MC):** The operator admin panel / management plane (NOT "back-zenith")
- **Web Platform:** The user-facing dashboard where developers manage apps/databases
- **Workload Cluster:** CAPI-managed K8s cluster per customer on Hetzner
- **Deploy Engine:** Phase 2 system for deploying apps from Git repos (clone -> detect framework -> Dockerfile -> Kaniko -> K8s)
- **CRD:** Custom Resource Definition — everything in Zenith is modeled as a K8s CRD

### Domain Convention
- `ms.{domain}` = Mission Control, `cloud.{domain}` = Web Platform
- Root domain stays for the customer

## Live Endpoints
| URL | Service | Description |
|-----|---------|-------------|
| https://freezenith.com | zenith-landing | Marketing site |
| https://demo-ms.freezenith.com | zenith-mc-demo | Demo Mission Control |
| https://demo-cloud.freezenith.com | zenith-web-demo | Demo Web Platform |
| https://api.freezenith.com | zenith-api | Go REST API |
| https://ms.embermind.app | zenith-mc | Customer MC |
| https://cloud.embermind.app | zenith-web | Customer Web Platform |

## Important Constraints
- **Traefik IngressRoute CRD:** Must use IngressRoute, NOT standard Ingress
- **Demo mode is build-time:** `NEXT_PUBLIC_DEMO_MODE` is baked into JS bundle, not runtime switchable
- **Image pull policy:** All images use `imagePullPolicy: Never` (local builds imported into k3s containerd)
- **Deploy order:** namespaces -> deployments/services -> certificates -> wait -> ingress routes
- **HTTP redirects after certs:** IngressRoutes for HTTP redirect must be applied AFTER cert-manager issues certs (HTTP-01 conflict)
- **Cloudflare proxy OFF:** Required for cert-manager HTTP-01 challenges
- **Dockerfiles build from repo root:** `docker build -f apps/X/Dockerfile .`
- **Privacy model:** Customer images never touch Zenith infrastructure. Build via Kaniko on customer cluster, push to customer's own registry.

## Current Development Status (Feb 2026)

### Completed
- All 3 frontend apps (landing, MC, web) — live and deployed
- Go API server — all CRUD + admin + deploy engine endpoints, JWT auth
- Go Auth service — OIDC endpoints, realm management
- K8s Operator — 8 CRD types, 8 controllers
- CLI `zen install` — interactive wizard
- Deploy Engine (Phases 2-4) — end-to-end: Git clone -> framework detection -> Dockerfile -> Kaniko -> K8s deploy
- Build log streaming (SSE)
- GitHub webhook integration (HMAC-SHA256)
- Phase 6.5 — 17 Pro/Team/Enterprise features (MFA, SSO, SCIM, RBAC, IP whitelist, preview deploys, etc.)
- Backend refactoring Phase A (entities/dto separation)
- Real K8s client (`RealClient` using client-go)
- OpenTelemetry + Backstage catalog wiring

### Not Yet Built
- Backend refactoring Phases B-D (services, ports/adapters)
- Auth service integration with login flows
- Several Web pages use hardcoded mock data (monitoring, gateway, auth, IAM, registry)
- KEDA autoscaling, Hetzner autoscaler, Stripe billing
- OpenAPI spec generation
- Custom domain management with automatic TLS
- Full CLI commands (deploy, logs, status, db)
- Production Helm chart deployment (currently raw manifests)

## External Dependencies
- **Hetzner Cloud:** Server hosting, volumes, object storage, load balancers, DNS
- **Cloudflare:** DNS management (Terraform provider)
- **GitHub:** Webhook integration, `zenith-actions` GitHub Action
- **Stripe:** Payment processing (planned, not yet integrated)
- **Let's Encrypt:** TLS certificates via cert-manager
