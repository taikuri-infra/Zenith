# AI Handover Document — Zenith Platform

> **Purpose:** This document enables seamless continuation of work across different AI accounts/sessions.
> **Last Updated:** 2026-02-27
> **Git Tag:** `v2.0.0-alpha.3` on branch `openspec/infra-pipeline-v1`
> **How to use:** When starting a new AI session, say:
> "Read `/Users/babak/codes/DoTech/Zenith/docs/v2-architecture/HANDOVER.md` and continue from where the previous session left off."

---

## Quick Context (Read First)

**Zenith** is a Kubernetes-native PaaS on Hetzner Cloud. We are designing and implementing the **V2 architecture** — a complete platform redesign with multi-tenant isolation, automated provisioning, and defense-in-depth security.

**Owner:** Babak — experienced DevOps engineer pursuing Golden Kube Astronaut certification + ArgoCD exam. Loves learning, prefers clean cloud-native solutions, Hetzner-only infrastructure.

**Language:** Babak speaks Farsi and English in conversation but all code/docs are in English. Always respond in English.

**3-Phase Deployment Pipeline:**
```
1. Terraform (staging/)        → Creates the Hetzner server + DNS records
2. Ansible                     → SSHes in, installs k3s + Cilium + prerequisite secrets
3. Terraform (staging-k8s/)    → Connects to k3s cluster, deploys all Helm charts
```

---

## WHERE WE ARE RIGHT NOW (2026-02-27)

### ✅ COMPLETED — Code is Written and Committed

| Phase | Status | Details |
|-------|--------|---------|
| **Phase 1: Terraform (Hetzner + DNS)** | ✅ Code done, applied on staging | Server cx42 running, all DNS records exist |
| **Phase 2: Ansible roles** | ✅ Code done, **NOT run yet** | k3s, Cilium+WireGuard, cert-manager DNS-01 secret roles updated |
| **Phase 3: Terraform (k8s-platform)** | ✅ Code done, **NOT applied yet** | All 22 V2 components written, `terraform validate` passes |
| **Phase 4-8** | ❌ Not started | ArgoCD App-of-Apps, Temporal workflows, migration |

### What "Code Done" Means

All the Terraform HCL for the k8s-platform module is **written and validated** (`terraform validate` → Success). The module was recently split from one 1806-line `main.tf` into 14 focused files:

```
infra/terraform/modules/k8s-platform/
├── main.tf            ← terraform block + PriorityClasses only (~105 lines)
├── certmanager.tf     ← cert-manager + ClusterIssuer (DNS-01)
├── sealed_secrets.tf  ← Sealed Secrets
├── storage.tf         ← CNPG operator + Keycloak PG + Free PG clusters
├── identity.tf        ← Keycloak
├── gateway.tf         ← APISIX + etcd + external-dns
├── gitops.tf          ← ArgoCD + Image Updater
├── registry.tf        ← Harbor
├── temporal.tf        ← Temporal workflow engine
├── security.tf        ← Kyverno + Falco + Velero
├── observability.tf   ← Prometheus + Loki + Tempo + OTel + Hubble UI
├── autoscaling.tf     ← KEDA
├── apps.tf            ← zenith-platform, api, landing, demo
├── tenant.tf          ← zenith-tenant (per-customer)
├── variables.tf       ← All input variables (480+ lines)
└── outputs.tf         ← All outputs (90 lines)
```

### What Has NOT Been Run Yet

1. **Ansible playbook** (Phase 2 task 4.5) — the roles are updated but the playbook hasn't been executed on the server
2. **`terraform apply`** for `staging-k8s/` — the Terraform code validates but has never been applied to the cluster
3. **Manual steps** — creating Temporal databases in CNPG, generating secrets for `terraform.tfvars`

---

## WHAT TO DO NEXT (Step-by-Step)

### Step 1: Generate Secrets (BEFORE anything else)

```bash
# Generate these and add to infra/terraform/staging-k8s/terraform.tfvars
openssl rand -hex 32  # → keycloak_db_password
openssl rand -hex 32  # → keycloak_admin_password
openssl rand -hex 32  # → temporal_db_password

# You also need these from provider dashboards:
# cloudflare_api_token  → Cloudflare Dashboard → API Tokens
# s3_access_key         → Hetzner Console → Object Storage
# s3_secret_key         → Same
# temporal_db_user      → set to "temporal"
```

### Step 2: Run Ansible (Phase 2)

```bash
cd infra/ansible
ansible-playbook -i inventory/staging.yml playbooks/site.yml
```

**Verify:**
```bash
kubectl get nodes                    # k3s running
cilium status                        # Cilium OK, WireGuard ON
ssh root@77.42.88.149 "k3s secrets-encrypt status"  # Enabled
kubectl get secret cloudflare-api-token -n cert-manager  # Exists
```

### Step 3: Run Terraform Apply (Phase 3)

```bash
cd infra/terraform/staging-k8s
terraform plan    # Review first!
terraform apply   # Deploy all 22 components
```

> **⚠️ IMPORTANT:** Monitor memory with `kubectl top nodes` after apply.
> cx42 = 8 vCPU / 16 GB RAM. If memory > 80%, deploy in waves.

### Step 4: Manual Post-Apply Steps

```bash
# Create Temporal databases (BEFORE Temporal pod can start)
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "CREATE DATABASE temporal;"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "CREATE DATABASE temporal_visibility;"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "CREATE USER temporal WITH PASSWORD '<generated>';"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "GRANT ALL ON DATABASE temporal TO temporal;"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "GRANT ALL ON DATABASE temporal_visibility TO temporal;"
```

### Step 5: Verify Everything Works

```bash
kubectl get pods -A                      # All pods running
kubectl get certificates -A              # TLS certs issued
kubectl get clusterissuer               # letsencrypt-prod Ready
```

### Step 6: Continue with Phase 4+ (from IMPLEMENTATION.md)

After Steps 1-5 are done, continue with `IMPLEMENTATION.md` from these unchecked tasks:

| Task | IMPLEMENTATION.md Line | Description |
|------|----------------------|-------------|
| 5.21 | ~2726 | ResourceQuota + LimitRange for customer namespaces |
| 5.22 | ~2796 | PodDisruptionBudgets for HA services |
| 6.1 | ~3004 | ArgoCD root Application (App-of-Apps) |
| 6.2 | ~3064 | Individual ArgoCD Application manifests |
| 6.3 | ~3189 | Sync wave annotations |
| 6.4 | ~3217 | End-to-end auto-deploy test |
| 7.1 | ~3279 | Temporal provisioning workflow |
| 7.2 | ~3427 | Temporal worker registration |
| 7.3 | ~3468 | End-to-end provisioning test |
| M1-M6 | ~3535+ | V1→V2 migration phases |

---

## Key Decisions (DO NOT CHANGE without discussing with Babak)

1. **APISIX** (not Kong) — API gateway, etcd-backed
2. **Keycloak** — Identity, realm per customer
3. **ArgoCD** (not FluxCD) — GitOps
4. **Temporal** — Provisioning workflows
5. **Cilium + WireGuard** — CNI with encryption
6. **Hetzner only** — S3, Volumes, VMs
7. **DNS-01 for cert-manager** — Enables Cloudflare proxy ON
8. **Frontends bypass APISIX** — Only backends go through gateway

---

## Codebase Structure

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
  infra/helm/              # Helm charts (zenith-platform, zenith-api, zenith-landing, zenith-demo, zenith-tenant)
  docs/v2-architecture/    # V2 design docs (this directory)
  openspec/                # Spec-driven development
  .lich/                   # Lich framework rules (AI behavior, backend, frontend, infra)
```

### Live Deployments
- **Production:** Not yet
- **Staging** (77.42.88.149 — Hetzner): V1 running (Terraform + Helm, cert-manager, Kong, CNPG, KEDA, monitoring)
- **Internal Harbor** (65.108.210.253): Platform images + Helm charts at `registry.stage.freezenith.com` (separate server, NOT in-cluster)
- **Customer Harbor** (in-cluster): Pro-tier customer registry at `hub.stage.freezenith.com` (one project per pro customer, with storage quotas)

---

## V2 Component List (22 Components in k8s-platform Module)

| # | Component | File | Namespace | Enable Flag |
|---|-----------|------|-----------|-------------|
| 1 | PriorityClasses | main.tf | cluster-wide | always |
| 2 | cert-manager + ClusterIssuer | certmanager.tf | cert-manager | always |
| 3 | Sealed Secrets | sealed_secrets.tf | sealed-secrets | `enable_sealed_secrets` |
| 4 | CNPG Operator | storage.tf | cnpg-system | `enable_cnpg` |
| 5 | Keycloak CNPG Cluster | storage.tf | keycloak | `enable_keycloak` |
| 6 | Free PG Cluster | storage.tf | zenith-shared | always |
| 7 | Keycloak | identity.tf | keycloak | `enable_keycloak` |
| 8 | APISIX + etcd | gateway.tf | apisix | `enable_apisix` |
| 9 | APISIX Ingress Controller | gateway.tf | apisix | `enable_apisix` |
| 10 | external-dns | gateway.tf | external-dns | `enable_external_dns` |
| 11 | ArgoCD | gitops.tf | argocd | `enable_argocd` |
| 12 | ArgoCD Image Updater | gitops.tf | argocd | `enable_argocd` |
| 13 | Customer Harbor (Pro-tier) | registry.tf | harbor | `enable_harbor` |
| 14 | Temporal | temporal.tf | temporal | `enable_temporal` |
| 15 | Kyverno | security.tf | kyverno | `enable_kyverno` |
| 16 | Falco | security.tf | falco | `enable_falco` |
| 17 | Velero | security.tf | velero | `enable_velero` |
| 18 | Prometheus+Grafana | observability.tf | monitoring | `enable_monitoring` |
| 19 | Loki | observability.tf | monitoring | `enable_monitoring` |
| 20 | Tempo | observability.tf | monitoring | `enable_monitoring` |
| 21 | OTel Collector | observability.tf | monitoring | `enable_monitoring` |
| 22 | Hubble UI IngressRoute | observability.tf | kube-system | always |
| 23 | KEDA + HTTP Addon | autoscaling.tf | keda | `enable_keda` |
| 24 | zenith-platform | apps.tf | zenith-platform | always |
| 25 | zenith-api | apps.tf | zenith-platform | always |
| 26 | zenith-landing | apps.tf | zenith-platform | always |
| 27 | zenith-demo | apps.tf | zenith-platform | `enable_demo` |
| 28 | zenith-tenant | tenant.tf | zenith-platform | `enable_tenants` |

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
| `IMPLEMENTATION.md` | **In Progress** | Step-by-step execution guide with checkboxes |
| `HANDOVER.md` | Complete | This file |
| `developers.md` | Complete | Developer experience guide |

---

## Key Files to Read Before Working

| Priority | File | Why |
|----------|------|-----|
| 1 | `docs/v2-architecture/HANDOVER.md` | This file — current status |
| 2 | `docs/v2-architecture/IMPLEMENTATION.md` | Step-by-step guide with checkboxes |
| 3 | `docs/v2-architecture/00-overview.md` | Full V2 architecture design |
| 4 | `agentlog.md` | Complete change history |
| 5 | `AGENTS.md` | Master AI prompt, Lich framework rules |

---

## Rules for Working on This Project

1. **Always read AGENTS.md first** — It has the Lich framework rules
2. **Always update agentlog.md** — Log WHAT, WHY, WHEN for every change
3. **Security first** — Every design decision considers backup, isolation, encryption
4. **Hetzner only** — No AWS, no GCP, no Azure
5. **APISIX not Kong** — Decision D1
6. **ArgoCD not FluxCD** — Decision D4
7. **Babak speaks Farsi** — Reply in English but understand Farsi requests
8. **Check IMPLEMENTATION.md** — Use checkboxes to track progress

---

## Contact & Resources

- **GitHub:** github.com/taikuri-infra/Zenith (private)
- **Branch:** `openspec/infra-pipeline-v1`
- **Latest Tag:** `v2.0.0-alpha.3`
- **Internal Harbor (platform images):** https://registry.stage.freezenith.com
- **Customer Harbor (pro-tier):** https://hub.stage.freezenith.com
- **Staging:** https://stage.freezenith.com
- **Production:** https://freezenith.com (V1 only)
- **Server SSH:** `ssh ghasi` (configured in ~/.ssh/config)
