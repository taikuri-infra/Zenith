# AI Handover Document — Zenith Platform

> **Purpose:** This document enables seamless continuation of work across different AI accounts/sessions.
> **Last Updated:** 2026-02-25
> **How to use:** When starting a new AI session, paste this instruction:
> "Read /Users/babak/codes/DoTech/Zenith/docs/v2-architecture/HANDOVER.md and continue from where the previous session left off."

---

## Quick Context (Read First)

**Zenith** is a Kubernetes-native PaaS on Hetzner Cloud. We are designing and implementing the **V2 architecture** — a complete platform redesign with multi-tenant isolation, automated provisioning, and defense-in-depth security.

**Owner:** Babak — experienced DevOps engineer pursuing Golden Kube Astronaut certification + ArgoCD exam. Loves learning, prefers clean cloud-native solutions, Hetzner-only infrastructure.

**Language:** Babak speaks Farsi in conversation but all code/docs are in English.

---

## What Exists Right Now

### Codebase Structure
```
Zenith/
  apps/landing/            # Next.js marketing site (LIVE)
  apps/mission-control/    # Next.js admin panel (LIVE)
  apps/web/                # Next.js user dashboard (LIVE)
  services/api/            # Go REST API, Fiber v2, 73 endpoints (LIVE)
  services/auth/           # Go OIDC/SAML auth service (built, not integrated)
  services/operator/       # Go K8s operator, 8 CRDs (built, not integrated)
  cli/                     # zen CLI with Cobra + Charm TUI
  packages/ui/             # @zenith/ui shared package
  infra/terraform/         # IaC (staging server, staging-k8s, modules)
  infra/ansible/           # Server config (k3s, Docker)
  infra/helm/              # Helm charts (zenith-platform, zenith-api, zenith-landing, zenith-demo, zenith-tenant + old monolithic zenith/)
  docs/v2-architecture/    # V2 design docs (this directory)
  openspec/                # Spec-driven development (specs + change proposals)
  .lich/                   # Lich framework rules (AI behavior, backend, frontend, infra)
```

### Live Deployments
- **Production** we dont have it yet.
- **Staging** (77.42.88.149 — Hetzner): Terraform + Helm, cert-manager, Kong, CNPG, KEDA, monitoring
- **Harbor** (65.108.210.253): Container + chart registry

### V1 → V2 Status
The monolithic Helm chart has been split into 5 modular charts (done). V2 architecture is fully designed but NOT yet implemented. Current infra is V1.

---

## V2 Architecture Summary

### 4-Tier Model
- **Free/Pro:** Shared k3s cluster, namespace isolation (Cilium), shared CNPG (Free: one cluster for all, Pro: sharded ~20/cluster)
- **Team/Enterprise:** Dedicated VMs via CAPI+CAPH, full kernel isolation

### Key Decisions (DO NOT CHANGE without discussing with Babak)
1. **APISIX** (not Kong) — API gateway, etcd-backed
2. **Keycloak** — Identity, realm per customer
3. **ArgoCD** (not FluxCD) — GitOps
4. **Temporal** — Provisioning workflows
5. **Cilium + WireGuard** — CNI with encryption
6. **Hetzner only** — S3, Volumes, VMs
7. **DNS-01 for cert-manager** — Enables Cloudflare proxy ON
8. **Frontends bypass APISIX** — Only backends go through gateway

### 4-Phase Deployment
```
Phase 1: Terraform → Hetzner VM + Cloudflare DNS
Phase 2: Ansible → k3s + Cilium + hcloud-csi
Phase 3: Terraform → All infra (cert-manager, CNPG, APISIX, Keycloak, ArgoCD, monitoring, etc.)
Phase 4: ArgoCD → Application charts (auto from Git)
```

### Full Component List (6 Layers)
```
Layer 1 Networking: Traefik, APISIX+etcd, Cilium+Hubble, external-dns
Layer 2 Security: Keycloak, cert-manager, Kyverno, Falco, Sealed Secrets
Layer 3 Data: CNPG Operator, Keycloak PG, Free PG, Pro PG shards, Hetzner S3
Layer 4 Platform: zenith-api, zenith-admin, Temporal, Harbor, ArgoCD
Layer 5 Observability: Prometheus, Grafana, Loki, Tempo, OTel Collector, Hubble, Alertmanager
Layer 6 Resilience: Velero, CNPG WAL→S3, pg_dump CronJobs, PriorityClasses, PDBs, ResourceQuota
```

---

## Memory Files (Auto-loaded by Claude)

These files are in `/Users/babak/.claude/projects/-Users-babak-codes-DoTech-Zenith/memory/`:

| File | Content |
|------|---------|
| `MEMORY.md` | Quick reference (auto-loaded every session) |
| `architecture.md` | Full V2 architecture design |
| `decisions.md` | All key decisions with rationale (D1-D14) |

**Important:** If you're a different AI (not Claude Code), read these files manually.

---

## Documentation Index

All V2 docs are in `docs/v2-architecture/`:

| File | Status | Content |
|------|--------|---------|
| `00-overview.md` | Complete | Master architecture overview |
| `01-phase1-hetzner-cloudflare.md` | Complete | Phase 1 detail |
| `02-phase2-ansible-k3s.md` | Complete | Phase 2 detail |
| `03-phase3-cluster-bootstrap.md` | Complete | Phase 3 detail (biggest) |
| `04-phase4-argocd-apps.md` | Complete | Phase 4 detail |
| `05-user-flows.md` | Complete | Customer, admin, developer flows |
| `06-security-model.md` | Complete | Defense-in-depth (6 layers) |
| `07-backup-disaster-recovery.md` | Complete | Backup strategy + RPO/RTO |
| `08-observability.md` | Complete | Monitoring, logging, tracing |
| `09-migration-v1-to-v2.md` | Complete | V1→V2 migration plan (6 weeks) |
| `HANDOVER.md` | Complete | This file |

---

## What Needs To Be Done Next

### Immediate Priority: Implement V2 Infrastructure

The design is complete. Implementation should follow the 4-phase pipeline:

1. **Update Terraform Phase 1** — Add new Hetzner server for V2 staging
2. **Update Ansible Phase 2** — Add Cilium role, hcloud-csi, etcd encryption
3. **Rewrite Terraform Phase 3** — Replace monolithic zenith helm_release with all V2 components (APISIX, Keycloak, external-dns, Temporal, etc.)
4. **Create ArgoCD manifests** — App-of-Apps in `infra/argocd/staging/`

### Secondary: Backend Updates for V2

- Integrate Keycloak (replace in-house JWT with Keycloak realm-based auth)
- Add Temporal workflows (customer provisioning)
- Add Hetzner S3 API calls (bucket creation)
- Add database creation logic (SQL in assigned CNPG shard)
- Add APISIX route CRD generation (per-customer)

### Tertiary: OpenSpec Updates

Some openspec specs reference Kong, FluxCD, or old architecture. These need updating:
- `openspec/project.md` — Update tech stack (APISIX, ArgoCD, Temporal)
- Various specs may reference Kong plugins → update to APISIX
- Add new specs for: Keycloak integration, Temporal workflows, APISIX routing

---

## Key Files to Read Before Working

| Priority | File | Why |
|----------|------|-----|
| 1 | `docs/v2-architecture/00-overview.md` | Full V2 design |
| 2 | `AGENTS.md` | Master AI prompt, Lich framework rules |
| 3 | `agentlog.md` | Complete change history |
| 4 | `docs/v2-architecture/HANDOVER.md` | This file |
| 5 | `openspec/project.md` | Project conventions |
| 6 | `.lich/rules/ai-behavior.md` | Lich-first decision logic |

---

## Rules for Working on This Project

1. **Always read AGENTS.md first** — It has the Lich framework rules
2. **Always update agentlog.md** — Log WHAT, WHY, WHEN for every change
3. **Use `lich make` commands** — Never create entities/services/APIs manually
4. **Security first** — Every design decision considers backup, isolation, encryption
5. **Hetzner only** — No AWS, no GCP, no Azure
6. **APISIX not Kong** — Decision D1
7. **ArgoCD not FluxCD** — Decision D4
8. **Babak speaks Farsi** — Reply in English but understand Farsi requests

---

## How to Continue a Session

```
Step 1: Read this HANDOVER.md
Step 2: Read the memory files (architecture.md, decisions.md)
Step 3: Check agentlog.md for latest changes
Step 4: Ask Babak what to work on next (or check the TODO above)
Step 5: Follow the Lich framework rules from AGENTS.md
Step 6: Update agentlog.md when done
```

---

## Contact & Resources

- **GitHub:** github.com/DoTech/Zenith (private)
- **Harbor:** https://registry.stage.freezenith.com
- **Staging:** https://stage.freezenith.com
- **Production:** https://freezenith.com
- **Server SSH:** `ssh ghasi` (configured in ~/.ssh/config)
