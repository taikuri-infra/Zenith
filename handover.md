# Zenith — Developer Handover
*Last Updated: 2026-02-22T17:00*

## Current Status & Completed Work
Phase 6.5 (Pro/Team/Enterprise Features) is **100% complete** (17/17 tasks). All backend API stubs, memory stores, handlers, routes, frontend types, demo mocks, and web build verified.

In the immediate previous sessions, we completed the backend for **Phases 5, 6, and 7**.
1. **Phase 5 (App Secrets)**: 
   - Created `app_secrets` table. 
   - Implemented AES-256-GCM encryption/decryption in `pkg/crypto/aes.go` (requires `SECRETS_ENCRYPTION_KEY` env var).
   - Created `SecretHandler` for `GET /apps/:appId/secrets` (lists keys only), `GET /apps/:appId/secrets/:key/value` (decrypts), `POST /apps/:appId/secrets` (encrypts and stores), and `DELETE`.
   - Backend is fully tested and functional. **The Frontend UI tab for Secrets is NOT yet built.**
2. **Phase 6 (Releases & Deploy Flow)**:
   - Created `app_releases` table to track images pushed by CI/CD.
   - Created `ReleaseHandler` to register releases (`POST /apps/:appId/releases`), list them, and trigger deploys (`POST /apps/:appId/releases/:rid/deploy`).
   - Added `TriggerImageDeploy` to `deploy/pipeline.go` allowing direct deployment of a pre-built image, bypassing the build phase.
   - **The Frontend UI Deployments tab with the version list and one-click Deploy button is NOT yet built.**
3. **Phase 7 (GitHub Actions)**:
   - Created a composite GitHub Action at `.github-actions/zenith-deploy/action.yml` that customers can use in their repos to build, push to their own registry, and register the release with Zenith.

**Tests:** `GO111MODULE=on go test ./internal/... -count=1` is completely clean except for two pre-existing flaky cluster tests (`TestScaleClusterEndpoint`, `TestUpgradeClusterEndpoint`) in `handlers_test.go` which pass when run individually.

---

## Architectural Decisions (Confirmed)
We have made significant structural decisions for the Zenith infrastructure (fully documented in `app_explanation.md`):
- **Privacy Model:** Customer image building and hosting happen entirely within the customer's own cluster (via Kaniko) and registry (e.g. GHCR, ECR). Zenith orchestrates, but never pulls or hosts customer images in its own infra.
- **GitOps Engine:** **FluxCD** (chosen over ArgoCD specifically for its native integration with CAPI via `ClusterResourceSet` to auto-bootstrap new customer clusters).
- **Networking/Security:** **Cilium** (eBPF, ClusterMesh for multi-cluster routing, mTLS between control plane and data planes).
- **API Gateway:** **Kong** (DB-less Ingress Controller).
- **Database Provisioning:** **CloudNativePG** for spinning up Postgres clusters per tenant.
- **IAM:** **Keycloak** (one realm per tenant).
- **Internal Chart Storage:** **Harbor** (only used internally to store Zenith's own infrastructure Helm charts, NOT customer app images).
- **Secret Management (Infra):** **Sealed Secrets** for GitOps state.

---

## Immediate Next Steps
The new account/session should pick up from here. The priorities are:

### 1. ✅ Frontend UI for Phases 5 & 6 (COMPLETED)
- **Secrets Tab:** Added to App Dashboard (`apps/[id]/page.tsx`) with List, Add, Reveal (decrypt), Copy, Delete. Demo mode has 3 mock secrets.
- **Releases Tab:** Added to App Dashboard with image version table, git SHA, branch, commit message, one-click Deploy button. Demo mode has 4 mock releases.

### 2. Finish the Remaining "Not Yet Wired" Backend Phases (COMPLETED)
- **Phase 8 (Real-Time Events):** ✅ COMPLETED — Created `EventHub` (in-memory pub/sub broadcaster), SSE endpoints at `/api/v1/events`, Pipeline emits 6 event types. Frontend `useDeployEvents` hook auto-refreshes Deploy page cards and App Detail deployment rows.
- **Phase 9 (OpenTelemetry):** ✅ COMPLETED — Wired `telemetry.Init()` + `telemetry.Middleware()` into main.go. Opt-in via `OTEL_EXPORTER_OTLP_ENDPOINT`. Traces + metrics exported via OTLP gRPC. Skips `/health` and `/ready`.
- **Phase 10 (Backstage Catalog):** ✅ COMPLETED — Wired `BackstageHandler` into `main.go` protecting `/api/v1/backstage/catalog` routes. 

### 3. Move to Infrastructure (Phase 11+)
- **Kubernetes Client:** ✅ COMPLETED — Replaced `MemoryClient` with `RealClient` using `client-go` v0.35.1. Supports auto-detection (in-cluster vs kubeconfig). Switchable via `K8S_MODE` environment variable.
- **Next:** Begin building the CAPI provisioning package (`internal/cluster`).

### 4. Backend Refactoring (Lich Architecture)
- **Phase A (Entities & DTOs):** ✅ COMPLETED — Created `internal/entities/` (pure domain types without json tags where possible) and `internal/dto/` (API input/output shapes). Separated God-structs from `models/` into these new directories.
- **Next Refactoring Phases (B, C, D):** Move logic into `services/`, abstract storage to `ports/` and `adapters/`, and refactor handlers to only handle HTTP mapping.

### 5. ✅ Phase 6.5: Pro/Team/Enterprise Features (17/17 COMPLETED)
All features built with backend (entities, memory stores, handlers, routes) + frontend (types, API clients, demo mocks). Web build passes.

**Pro Features:**
- **S65-01** MFA/2FA (TOTP) — `handlers/mfa.go`, `store/memory_mfa.go`, `entities/mfa.go`
- **S65-02** GitLab/Bitbucket webhooks — `HandleGitLabPush`, `HandleBitbucketPush` in `handlers/webhook.go`
- **S65-03** User-defined webhooks — `handlers/user_webhook.go`, `store/memory_webhook.go`, `entities/webhook.go`
- **S65-04** API keys — `handlers/api_keys.go`, `store/memory_apikey.go`, `entities/api_key.go`

**Team Features:**
- **S65-05/06** SSO (SAML + OIDC) — `handlers/sso.go`, `store/memory_sso.go`, `entities/sso.go`
- **S65-07** Session management — `handlers/session.go`, `store/memory_session.go`, `entities/session.go`
- **S65-08** Audit log export (CSV/JSON) — `handlers/audit_export.go`
- **S65-09** DPA (Data Processing Agreement) — `handlers/branding.go`, `store/memory_branding.go`, `entities/branding.go`
- **S65-10** Preview deployments — `handlers/preview.go`, `store/memory_preview.go`, `entities/preview.go`

**Enterprise Features:**
- **S65-11** SCIM 2.0 provisioning — `handlers/scim.go`
- **S65-12** Custom roles (RBAC) — `handlers/role.go`, `store/memory_role.go`, `entities/role.go`
- **S65-13** IP whitelisting — `handlers/ip_whitelist.go`, `store/memory_ipwhitelist.go`, `entities/ip_whitelist.go`
- **S65-14** Compliance dashboard — `handlers/compliance.go`
- **S65-15/16/17** White-label (branding, custom domain, hide badge) — in `handlers/branding.go`

**Frontend:** Settings page has 6 tabs (API Keys, MFA, Webhooks, Sessions, Security, General). All demo mocks in `demo-api.ts`.

---

## Immediate Next Steps
1. **OpenAPI Spec** — Generate/write OpenAPI spec for all API endpoints
2. **Phase 4** (KEDA autoscaling) — requires real K8s
3. **Phase 5** (Hetzner Autoscaler) — requires real infra
4. **Phase 6** (Stripe Billing) — requires Stripe setup

---

## Rules to Follow
- Always append to `agentlog.md` with WHAT, WHY, and WHEN for every single architectural or code adjustment.
- **Documentation is Mandatory:** No feature is complete without a corresponding doc in `docs/` and an entry in `agentlog.md`. (See `docs/features/backend/*.md`)
- Read `app_explanation.md` and `agentlog.md` if you ever need deep context on the system.
- **Tests:** `GO111MODULE=on go test ./...` passes. Only known failure is `TestScaleClusterEndpoint` (pre-existing, needs CAPI CRD object).
