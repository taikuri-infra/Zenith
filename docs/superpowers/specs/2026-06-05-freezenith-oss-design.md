# FreeZenith — Open-Source Self-Hosted Cloud Platform

**Date:** 2026-06-05  
**Status:** Approved  
**Author:** Babak Doraniarab  
**Edition:** Community Edition v1.0  

---

## 1. Vision

FreeZenith is the open-source, self-hosted edition of Zenith — a Kubernetes-native PaaS that anyone can install on their own Hetzner account in under 10 minutes with a single command.

It is differentiated from every other self-hosted PaaS (Coolify, CapRover, Dokku) by four pillars:

1. **Managed databases** — Postgres, Redis, MongoDB backed by production-grade Kubernetes operators (CNPG, Spotahome, Percona) with automatic backups and point-in-time restore
2. **Full observability out of the box** — every app gets logs (Loki), metrics (Prometheus/Grafana), and traces (Tempo/OTel) from the moment it starts, with zero configuration
3. **Real Kubernetes scale** — starts on a single cx32 server (€13/mo), scales to multi-node clusters with `zen node add` — no Docker Swarm ceiling
4. **Auth-as-a-service** — Keycloak-powered auth realms deployed per project with pre-configured OAuth2/OIDC providers

**Brand positioning:** "The European open-source cloud platform that makes Kubernetes invisible and developer experience magical — runs on your Hetzner account, owned by you."

---

## 2. Product Model

### Two editions, one codebase

```
github.com/dotechhq/zenith  (public repo, MIT license)
│
├── Community Edition (CE)       ← FreeZenith
│   ZENITH_EDITION=community
│   • Apps, deployments, environments
│   • Managed databases (Postgres, Redis, MongoDB)
│   • Object storage (S3-compatible, Hetzner)
│   • API Gateway (APISIX — rate limiting, auth, routing)
│   • Auth realms (Keycloak — OAuth2/OIDC/SAML)
│   • Full observability (Loki, Prometheus, Grafana, Tempo, OTel)
│   • Auto-scaling (KEDA — HTTP, CPU, queue, custom metrics)
│   • RBAC (owner, admin, developer, viewer roles)
│   • Environments (production + staging per project)
│   • Deploy tokens + GitHub Action + Terraform provider
│   • Backups + restore (CNPG WAL + S3)
│   • Audit log, webhooks, notifications
│   • Private container registry (Harbor)
│   • `zen` CLI for local dev and management
│   • Self-service dashboard (zenith-web)
│   • Admin panel (Mission Control, scoped to own installation)
│
└── Enterprise/SaaS Layer        ← freezenith.com (ZENITH_EDITION=enterprise)
    Adds on top of CE:
    • Multi-tenancy (isolated namespaces per customer)
    • Billing (Stripe subscriptions + usage metering)
    • Plan tiers with enforced limits
    • Customer CRM + Mission Control SaaS dashboards (MRR, churn)
    • Plan upgrade Temporal workflows
    • Cluster autoscaler (adds/removes Hetzner nodes on demand)
    • SCIM (enterprise user provisioning)
    • Referral system, email campaigns, dormant cleanup
    • Support SLA management (Gold/Platinum tiers)
```

### The SaaS is just CE + operations

`freezenith.com` = FreeZenith CE installed on Zenith team's Hetzner account + the enterprise layer enabled. The competitive moat is not the code — it is the operational expertise: 99.9% uptime, incident response, upgrade management, and support SLAs. This cannot be cloned from GitHub.

### Repository strategy

One public repo. The enterprise layer code is visible in the public repo but inactive without `ZENITH_EDITION=enterprise` and the corresponding credentials (Stripe keys, etc.). This is the Supabase/Posthog model. No syncing between two repos. No code divergence.

---

## 3. The Installer: `zen install`

### User experience

```
$ zen install

  Welcome to FreeZenith

  Domain             mycloud.example.com
  Hetzner API token  ••••••••••••••••••••
  Server size        cx32 — 4 vCPU / 8GB RAM (€13/mo)  [recommended]
  Region             Helsinki (hel1)
  DNS provider       Cloudflare (automatic) / manual

  ──────────────────────────────────────────────
  [1/6] Provisioning Hetzner server ...  ✓ 65.21.x.x (58s)
  [2/6] Configuring DNS ...              ✓ 3 records created (12s)
  [3/6] Installing k3s ...               ✓ v1.34.3 ready (1m 48s)
  [4/6] Installing FreeZenith ...        ✓ 14 components healthy (5m 22s)
  [5/6] Issuing TLS certificates ...     ✓ Let's Encrypt issued (1m 10s)
  [6/6] Health check ...                 ✓ All systems operational (22s)
  ──────────────────────────────────────────────

  FreeZenith is ready!

  Dashboard   https://cloud.mycloud.example.com
  Admin user  admin@mycloud.example.com
  Password    ePxK9mV2qLrN

  Get started:
    zen login https://cloud.mycloud.example.com
    zen deploy --image nginx:latest --name my-first-app

  Total time: 9m 52s
```

### Non-interactive mode

```bash
zen install \
  --domain mycloud.example.com \
  --hetzner-token $HCLOUD_TOKEN \
  --cloudflare-token $CF_TOKEN \
  --region hel1 \
  --server-type cx32 \
  --admin-email admin@example.com
```

### Installation steps (internal)

| Step | What happens | Implementation |
|------|-------------|----------------|
| 1. Provision server | Hetzner API: create SSH keypair → firewall (22/80/443/6443) → cx32 server → wait for SSH | `hcloud` Go SDK in `provisionHetznerServer()` |
| 2. Configure DNS | Cloudflare API: A records for `domain`, `*.apps.domain`, `*.gw.domain` | Cloudflare Go client in `configureDNS()` |
| 3. Bootstrap k3s | SSH → run k3s install script → install Helm 3 → write kubeconfig | SSH + remote exec in `bootstrapK3s()` |
| 4. Install platform | `helm upgrade --install zenith oci://ghcr.io/dotechhq/zenith/charts/zenith --values values-community.yaml` | `InstallViaHelm()` in `helm.go` |
| 5. Issue TLS | Wait for cert-manager to issue certs (poll `kubectl get certificate`) | `issueSSL()` with timeout + retry |
| 6. Health check | HTTP poll `https://cloud.domain/api/health` until 200 | `waitForHealthy()` with exponential backoff |

### DNS fallback (manual mode)

When `--dns-provider manual` or Cloudflare token not provided:

```
  Manual DNS setup required.
  
  Add these records to your DNS provider:
  
  Type  Name                           Value
  A     mycloud.example.com            65.21.x.x
  A     *.apps.mycloud.example.com     65.21.x.x
  A     *.gw.mycloud.example.com       65.21.x.x

  Press Enter when DNS is configured (zen will wait for propagation)...
```

---

## 4. The `zen upgrade` Command

Every release of FreeZenith must be safely upgradeable from the previous version. Without this, every release risks breaking existing installations.

### Design

```bash
zen upgrade [--version v1.1.0] [--dry-run]
```

Steps:
1. **Pre-flight check**: verify cluster is reachable, check current version, check disk space
2. **Backup trigger**: run `zen backup create --reason=upgrade` (snapshot CNPG databases to S3)
3. **Dry run mode**: run `helm diff upgrade` to show what will change (requires `helm-diff` plugin)
4. **Upgrade**: `helm upgrade zenith oci://ghcr.io/dotechhq/zenith/charts/zenith --version $VERSION`
5. **Wait for rollout**: poll each component's rollout status
6. **Health check**: same as install step 6
7. **Rollback on failure**: if health check fails after 5 minutes, `helm rollback zenith`

### Version compatibility

The `zenith` umbrella chart carries a `compatibility` annotation. If upgrading more than one minor version at a time (e.g., v1.0 → v1.3), `zen upgrade` blocks and tells the user to upgrade one minor version at a time.

---

## 5. Codebase Changes

### A — Edition boundary in the API

Add `ZENITH_EDITION` env var (default: `community`). At API startup, register only the handlers and services appropriate for the edition.

**Enterprise-only (gated behind `edition == "enterprise"`):**
- `services/billing/`
- `adapters/stripe/`
- `handlers/stripe_webhook.go`
- `services/referral/`, `services/exit_survey/`, `services/email_campaign/`
- `services/dormant_cleanup/`
- `handlers/admin_crm.go`, `handlers/admin_analytics.go`
- `handlers/scim.go`
- Multi-tenant customer provisioning + plan enforcement logic

**Community (everyone gets):**
- All app/deployment/environment handlers
- All database/storage/gateway handlers
- Auth, MFA, SSO (OIDC/SAML — not SCIM)
- RBAC, team management, audit log
- Observability handlers (logs, metrics, traces)
- Backup/restore handlers
- Admin panel (own installation management only)
- Webhook, notification, deploy token handlers

### B — Helm chart additions

New files in `infra/helm/zenith/`:
- `values-community.yaml` — disables Stripe, CRM, multi-tenant quota components
- `values-enterprise.yaml` — enables everything (rename current staging values, extend)

### C — Stale Lich/moneyFactory cleanup

Files to update:
- `.lich/AI_CONTEXT.md` — replace moneyFactory/fastapi content with Zenith/Go context
- `.lich/PROJECT_CONFIG.yaml` — update project metadata
- `AGENTS.md` — remove moneyFactory references, keep Lich framework rules

### D — Helm chart publication (CI)

Add GitHub Actions workflow `publish-chart.yml`:
- Trigger: push tag `v*`
- Action: `helm package` + `helm push` to `oci://ghcr.io/dotechhq/zenith/charts/zenith`

### E — Installer reliability pass

After the installer is functionally complete, run it against 10 clean Hetzner accounts and track every failure mode. Common expected issues:
- Operator startup ordering (CNPG before Keycloak dependency)
- cert-manager ACME rate limits during testing (use staging LE issuer in dev)
- Harbor registry initialization timing
- DNS propagation variance (Cloudflare is fast; other providers can be slow)

Fix every failure mode before launch.

---

## 6. What the User Sees After Install

The "first 5 minutes" experience must make the four differentiators immediately obvious.

### The dashboard (post-install)

```
Welcome to FreeZenith!  Let's deploy something.

┌─────────────────────────────────────────────────┐
│  Quick start                                     │
│  ○ Deploy an app        → New App               │
│  ○ Create a database    → New Database          │
│  ○ Set up auth          → New Auth Realm        │
│  ○ Explore observability → Logs / Metrics        │
└─────────────────────────────────────────────────┘

Platform health: All systems operational  ✓
Observability:   Logs ✓  Metrics ✓  Traces ✓
```

### The "wow" demo (deploy an app with a database and auth in 3 clicks)

1. Click "New App" → paste a Docker image → deployed in 60 seconds with a live URL
2. Click "New Database" → Postgres created in 60 seconds → connection string auto-injected as env var into the app
3. Click "New Auth Realm" → Keycloak realm created → SDK snippet shown for Next.js/React/Vue

This is the demo that goes on the GitHub README, the HackerNews post, and the docs homepage.

---

## 7. Installer Reliability (Score 10/10 requirement)

The installer must handle these failure modes gracefully:

| Failure | Behavior |
|---------|----------|
| Hetzner API rate limit | Retry with exponential backoff, max 3 attempts |
| SSH connection refused (server not ready) | Retry for up to 3 minutes before failing |
| Cloudflare API error | Show manual DNS instructions as fallback |
| Helm install timeout (operator slow to start) | Extend timeout to 15min, show component-level progress |
| cert-manager fails to issue cert | Check ACME rate limit, show diagnostic URL |
| DNS propagation slow | Poll for up to 15 minutes before declaring failure |
| Partial install (interrupted) | `zen install --resume` detects which steps completed and continues |
| Already installed (re-run) | Detect existing install, suggest `zen upgrade` |

### `zen install --resume`

If the install is interrupted (Ctrl+C, network drop, etc.), `zen install` on re-run detects the incomplete state and continues from the last successful step. State is written to `~/.zen/install-state.json`.

---

## 8. Launch Plan

### Phase 1 — Foundation (4 weeks)

| Week | Work |
|------|------|
| 1 | Codebase cleanup: edition boundary, Helm values-community.yaml, Lich/moneyFactory cleanup |
| 2 | Implement `provisionHetznerServer()` + `configureDNS()` (Hetzner + Cloudflare SDKs) |
| 3 | Implement `bootstrapK3s()` + `InstallViaHelm()` + `waitForHealthy()` |
| 4 | End-to-end test: `zen install` on a fresh Hetzner account — working cloud in <10 min |

### Phase 2 — Hardening (2 weeks)

| Week | Work |
|------|------|
| 5 | Design and implement `zen upgrade` command |
| 6 | Installer reliability pass: 10 clean installs, fix all failure modes |

### Phase 3 — Polish + Docs (2 weeks)

| Week | Work |
|------|------|
| 7 | Documentation: install guide, quickstart, "deploy your first app" tutorial |
| 8 | README redesign, `install.freezenith.com` landing page, demo GIF/video |

### Phase 4 — Launch (1 week)

- Flip GitHub repo to public
- HackerNews "Show HN: FreeZenith — open-source Railway/Heroku on your own Hetzner"
- Post on: Reddit r/selfhosted, r/devops, Dev.to, LinkedIn (32K), YouTube (4K)
- Track GitHub stars, Discord joins, install counts

### Phase 5 — Iterate + SaaS (ongoing, starts month 3)

- Process community GitHub issues and PRs
- Add `zen node add/remove` for cluster scaling
- Enable `freezenith.com` (ZENITH_EDITION=enterprise on own cluster)
- Onboard first paid managed customers
- Roadmap: provider expansion (DigitalOcean, Linode, bare metal)

---

## 9. Success Metrics (90 days post-launch)

| Metric | Target |
|--------|--------|
| GitHub stars | 2,000+ |
| Successful self-installs | 500+ |
| Discord/community members | 300+ |
| freezenith.com paid customers | 20+ |
| `zen install` success rate | >95% |
| Install time p50 | <10 minutes |

---

## 10. Open Questions (to resolve during implementation)

1. **Server type minimum**: Is cx22 (2 vCPU / 4GB) enough for the full stack? Or is cx32 the minimum for observability? Need to test.
2. **Harbor necessity in CE**: Harbor adds ~2min to install time and 1.5GB RAM. Is a private registry essential for CE, or optional (`--with-registry` flag)?
3. **Keycloak startup time**: Keycloak takes 2-4 minutes to be ready. Does this dominate the install time? Consider deferring its startup check.
4. **Multi-region**: Phase 1 targets hel1/nbg1/fsn1. How does the installer handle server creation outside these regions? Validate and block unknown regions early.
5. **Existing server**: The wizard supports `--provider existing` (bring your own server). How much of the installer works on a non-Hetzner bare-metal machine? Plan for this in Phase 2.

---

## 11. What We Are Not Building (YAGNI)

- **DigitalOcean / Linode support** — Hetzner-only for v1. Expand after community feedback.
- **Windows support for `zen` CLI** — Linux/macOS only for v1.
- **GUI installer** — The TUI + non-interactive flags cover all cases.
- **Auto-TLS without a domain** — Users must have a domain. nip.io workarounds are not supported in v1.
- **Multi-cluster FreeZenith** — CE is single-cluster. Multi-cluster is enterprise only.
- **Mobile dashboard** — Not a priority for v1.
- **Harbor bundled by default** — Harbor adds ~2 min install time + 1.5GB RAM. In v1, it is optional (`--with-registry` flag). Users can always add it later with `zen component add registry`.
