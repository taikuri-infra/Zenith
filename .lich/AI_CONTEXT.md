# AI Context — Zenith / FreeZenith

> This file gives an AI agent the minimum context needed to work in this codebase.

## Project Identity

```yaml
name: Zenith / FreeZenith
slug: zenith
type: paas_platform
description: Open-source Kubernetes-native PaaS — deploys on Hetzner Cloud with a single command
author: Babak Doraniarab
email: babak@dotech.fi
editions:
  community: ZENITH_MODE=standalone (free, self-hosted)
  enterprise: ZENITH_MODE=saas (freezenith.com managed cloud)
```

## Stack

```yaml
backend: Go (clean architecture — entities → ports → services → adapters → handlers)
frontends:
  web: Next.js (customer dashboard — cloud.*)
  mc:  Next.js (mission control admin — mission.*)
  landing: Next.js (marketing site)
database: PostgreSQL via CloudNativePG (CNPG) operator
cache: Redis
auth: Keycloak (OAuth2/OIDC/SAML)
gateway: APISIX
kubernetes: k3s (single-node CE, multi-node enterprise)
helm: umbrella chart at infra/helm/zenith/
observability: Loki + Prometheus + Grafana + Tempo + OTel
autoscaler: KEDA (HTTP + custom metrics)
ci_cd: GitHub Actions → Harbor registry → ArgoCD (staging watches staging branch)
infrastructure: Hetzner Cloud only (hel1/fsn1/nbg1/ash)
```

## Repository Layout

```
services/
  api/                  Go API — cmd/server/main.go bootstraps everything
    internal/
      entities/         Pure domain models (no DB/HTTP deps)
      ports/            Interfaces (repository contracts, service interfaces)
      services/         Business logic
      adapters/         Implementations (postgres/, keycloakclient/, s3client/, ...)
      handlers/         HTTP handlers (Fiber framework)
      config/           Config loaded from env vars (ZENITH_MODE, DB_*, ...)
  operator/             Kubernetes operator (Go, controller-runtime)
  github-action/        GitHub Action for zen deploy
  terraform-provider-zenith/

apps/
  web/                  Customer dashboard (Next.js, src/app/ App Router)
  mission-control/      Admin panel (Next.js, src/app/ App Router)
  landing/              Marketing site (Next.js)

cli/                    zen CLI (Go, cobra + charmbracelet TUI)
  cmd/                  Subcommands: install, upgrade, backup, deploy, login, ...
  internal/             Packages: install, hetzner, sshclient, k3s, cloudflare, ...

infra/
  helm/zenith/          Umbrella Helm chart (OCI: ghcr.io/dotechhq/zenith/charts/zenith)
    values-community.yaml   CE defaults (ZENITH_MODE=standalone, no billing)
    values-staging.yaml
    values-production.yaml
  terraform/            Hetzner Cloud infra (staging, production, DR)
```

## Architecture Rules

- **Entities** are pure Go structs — no DB tags, no HTTP deps
- **Services** depend only on port interfaces — never on adapters directly
- **Adapters** implement ports — postgres/, keycloakclient/, s3client/, etc.
- **Handlers** call services, never repos directly
- New features: `lich make entity <name>` / `lich make service <name>` / `lich make api <name>`

## Edition Boundary

```go
// config.go
cfg.Mode = getEnv("ZENITH_MODE", "standalone") // "standalone" | "saas"

// main.go pattern
if cfg.Mode == "saas" {
    // register billing, stripe, CRM, SCIM, referral, dormant-cleanup handlers
}
```

Community Edition gets everything EXCEPT: billing, Stripe webhooks, CRM, SCIM, referral, email campaigns, dormant cleanup, multi-tenant quota enforcement.

## Frontend Pattern

Both web and mc use:
```ts
import { getApi } from "@/lib/get-api";   // returns real or demo API
import { useApi } from "@/hooks/use-api"; // data fetching hook with loading/error states

const apiClient = getApi();
const { data, loading, error, refetch } = useApi(() => apiClient.someResource.list());
```

## Key Environment Variables

```
ZENITH_MODE          standalone | saas
DB_HOST / DB_PORT / DB_USER / DB_NAME / DB_PASSWORD
JWT_SECRET
HETZNER_TOKEN
CLOUDFLARE_TOKEN
HARBOR_HOST / HARBOR_ROBOT_USER / HARBOR_ROBOT_TOKEN
KEYCLOAK_URL / KEYCLOAK_REALM / KEYCLOAK_CLIENT_ID / KEYCLOAK_CLIENT_SECRET
S3_ENDPOINT / S3_ACCESS_KEY / S3_SECRET_KEY / S3_BUCKET
```

## When Making Changes

1. Follow the rule files in `.lich/rules/` (backend.md, frontend.md, security.md)
2. Write tests for new code — TDD preferred
3. Update `agentlog.md` with WHAT, WHY, WHEN
4. New handlers must be registered in `cmd/server/main.go`
5. New CLI commands must be registered in `cli/cmd/root/root.go`
6. Staging deploys: commit to `staging` branch and push — ArgoCD auto-syncs
