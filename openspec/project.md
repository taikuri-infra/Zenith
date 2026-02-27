# Project Context

## Purpose
Zenith is a **Kubernetes-native Platform-as-a-Service (PaaS)** built on Hetzner Cloud, operated by DoTech as a managed multi-tenant cloud service. It offers 4 tiers:

- **Free/Pro:** Shared k3s cluster with namespace isolation (Cilium), shared CNPG databases, shared Hetzner S3 buckets. Pro customers are sharded (~20 per CNPG Cluster) for better performance.
- **Team/Enterprise:** Dedicated VMs provisioned via CAPI+CAPH. Full kernel-level isolation. Own Keycloak, APISIX, CNPG, everything.

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

### Infrastructure (V2 Architecture)
- **Compute:** Hetzner Cloud VMs, k3s v1.34.3
- **Orchestration:** CAPI + CAPH (Cluster API Provider Hetzner) for Team/Enterprise
- **Ingress:** Traefik 3.5.1 (IngressRoute CRD for frontends)
- **API Gateway:** Apache APISIX (etcd-backed, JWT verification, CORS, rate-limiting)
- **Identity:** Keycloak (realm per customer, OIDC/SAML)
- **Database:** CloudNativePG (CNPG) operator with sharded clusters
- **Storage:** Hetzner S3 (Object Storage), Hetzner Volumes (block storage)
- **TLS:** cert-manager with DNS-01 challenge (Cloudflare API), enables Cloudflare proxy ON
- **DNS:** external-dns (Cloudflare provider, auto-creates records from Ingress)
- **CNI:** Cilium with WireGuard encryption + Hubble observability
- **GitOps:** ArgoCD (App-of-Apps pattern)
- **Provisioning:** Temporal (multi-step workflows with retry/recovery)
- **Registry:** Harbor (container images + Helm charts OCI)
- **Security:** Kyverno (policy), Falco (runtime), Sealed Secrets, cosign (image signing)
- **Monitoring:** Prometheus + Grafana + Loki + Tempo + OpenTelemetry + Hubble + Alertmanager
- **Backup:** Velero + CNPG WAL archiving + pg_dump CronJobs

### Decision Log
See `docs/v2-architecture/decisions.md` or Claude memory `decisions.md` for all key decisions (D1-D14).

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
  terraform/            # IaC (Hetzner, Cloudflare, K8s resources)
  ansible/              # Server configuration (k3s, Cilium, hcloud-csi)
  helm/                 # Modular Helm charts:
    zenith-platform/    #   Shared resources (secrets, postgres, certs, middleware)
    zenith-api/         #   Go API server
    zenith-landing/     #   Landing page
    zenith-demo/        #   Demo MC + Web
    zenith-tenant/      #   Per-customer deployments
    monitoring/         #   Prometheus + Grafana + Loki
  argocd/               # ArgoCD Application manifests (App-of-Apps)
  k8s/                  # Raw K8s manifests (legacy, production only)
  scripts/              # deploy.sh (legacy), cloudflare-dns.sh
docs/
  v2-architecture/      # V2 design docs (phases, security, backup, flows)
```

## Deployment Pipeline (V2 — 4 Phases)

```
Phase 1: Terraform → Hetzner VM + Cloudflare DNS
Phase 2: Ansible → k3s + Cilium + hcloud-csi
Phase 3: Terraform → All infra (cert-manager, CNPG, APISIX, Keycloak, ArgoCD, monitoring, ...)
Phase 4: ArgoCD → Application charts (automatic from Git push)
```

See `docs/v2-architecture/` for detailed documentation per phase.

## Project Conventions

### Code Style
- **Go:** Standard Go formatting (`gofmt`), error wrapping with `fmt.Errorf`, no globals
- **TypeScript:** Strict mode, functional React components, named exports
- **CSS:** Tailwind utility classes, no custom CSS files
- **Naming:** kebab-case files, PascalCase components/types, camelCase functions/variables

### Architecture Patterns
- **4-tier model:** Free (shared), Pro (shared sharded), Team (dedicated), Enterprise (dedicated)
- **CRD-driven:** User action → API creates CRD → Operator watches → provisions infrastructure
- **API switching:** `getApi()` returns real or demo API based on build-time `NEXT_PUBLIC_DEMO_MODE`
- **Repository pattern:** Store interfaces with memory + PostgreSQL implementations
- **Backend target:** Clean/Hexagonal architecture (entities, services, ports, adapters, dto)
- **Demo mode:** Build-time env var `NEXT_PUBLIC_DEMO_MODE=true` bakes mock data into JS bundle
- **Routing split:** Frontends go through Traefik directly, backends go through APISIX
- **Provisioning:** Temporal workflows for customer creation, tier changes, CAPI clusters
- **GitOps:** ArgoCD for apps, Terraform for infrastructure

### Testing Strategy
- **Go:** `go test ./internal/... -count=1` — unit tests for handlers, stores, deploy pipeline
- **Frontend:** No test framework configured yet
- **Coverage:** 89+ unit tests across deploy engine components

### Git Workflow
- Main branch: `main`
- Feature branches: `feature/description` or `setup/description`
- Commit style: conventional commits (`feat:`, `fix:`, `docs:`, `refactor:`)

## Domain Context

### Business Model
- 4 tiers: Free, Pro, Team, Enterprise
- Free/Pro share infrastructure (namespace isolation)
- Team/Enterprise get dedicated infrastructure (CAPI)
- Resources provisioned on-demand
- Billing: Stripe (international) + Toman/IRR (via Fairbroker)

### Key Terminology
- **Mission Control (MC):** The operator admin panel / management plane
- **Web Platform:** The user-facing dashboard where developers manage apps/databases
- **Workload Cluster:** CAPI-managed K8s cluster per Team/Enterprise customer
- **Deploy Engine:** System for deploying apps from Git repos (clone → detect → Dockerfile → Kaniko → K8s)
- **CRD:** Custom Resource Definition — everything in Zenith is modeled as a K8s CRD
- **CNPG Shard:** A CloudNativePG Cluster serving ~20 Pro customers

### Domain Convention
- `ms.{domain}` = Mission Control, `cloud.{domain}` = Web Platform
- Root domain stays for the customer
- Free users: `<customer>.freezenith.com` (auto via external-dns)
- Pro+: can bring custom domain

## Important Constraints
- **Traefik IngressRoute CRD:** Must use IngressRoute, NOT standard Ingress
- **Demo mode is build-time:** `NEXT_PUBLIC_DEMO_MODE` is baked into JS bundle
- **Deploy order:** namespaces → deployments/services → certificates → wait → ingress routes
- **Dockerfiles build from repo root:** `docker build -f apps/X/Dockerfile .`
- **Hetzner only:** No AWS, GCP, or Azure
- **APISIX not Kong:** Decision D1 (see decisions.md)
- **ArgoCD not FluxCD:** Decision D4

## External Dependencies
- **Hetzner Cloud:** VMs, volumes, object storage, load balancers
- **Cloudflare:** DNS management, CDN, DDoS protection, WAF
- **GitHub:** Code hosting, webhook integration, CI/CD via Actions
- **Stripe:** Payment processing (planned)
- **Let's Encrypt:** TLS certificates via cert-manager (DNS-01)
