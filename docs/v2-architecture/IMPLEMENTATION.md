# Zenith V2 Implementation Guide

> **Self-contained execution guide for any AI agent or engineer with zero prior context.**
> Every task has a checkbox, file paths, inline code, validation command, and expected output.
> No external documents are required — all critical information is inline.

---

## How to Use This File

**For AI agents:** Read the full file. Skip all tasks marked `[x]` (completed). Start from the first unchecked `[ ]` task. Always **read the actual source files before editing** — the HCL/YAML snippets below are guidance, not copy-paste. Adapt to the real state of the code.

**For engineers:** Work through phases in order. Each session, tell your AI: *"Read `docs/v2-architecture/IMPLEMENTATION.md` and execute Phase X, starting from task Y.Z"*. After validating a task, check the box (`[x]`). Any future AI will see checked tasks as done and skip them.

**Progress tracking:** `- [ ]` = pending, `- [x]` = done. That's it.

---

## Pre-Flight Checklist (Read Before Starting)

Before executing any phase, review these critical concerns:

### PF-1: Kong → APISIX Migration (Breaking Change)

The current staging cluster has Kong running. When you run `terraform apply` with Kong removed, Terraform will **destroy Kong before APISIX exists**. If staging has live traffic, this causes downtime.

**Required approach:**
1. First apply: `enable_apisix = true`, `enable_kong = true` (both running)
2. Migrate all routes from Kong to APISIX
3. Verify APISIX handles all traffic correctly
4. Second apply: `enable_kong = false` (safe removal)

### PF-2: Generate All Secrets Before Phase 3

The implementation adds ~10 new sensitive variables. Generate and store these **before** starting Phase 3:

```bash
# Generate these BEFORE Phase 3
openssl rand -hex 32  # keycloak_db_password
openssl rand -hex 32  # keycloak_admin_password
openssl rand -hex 32  # temporal_db_password
# Plus: cloudflare_api_token, s3_access_key, s3_secret_key (from provider dashboards)
```

Add all new variables to `infra/terraform/staging-k8s/terraform.tfvars` before running `terraform apply`.

### PF-3: Temporal Database Prerequisite

Task 5.12 (Temporal) has a hidden manual step. You **must** create `temporal` and `temporal_visibility` databases inside the free-pg CNPG cluster **before** the Temporal Helm release deploys. Do this between tasks 5.6 and 5.12:

```bash
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "CREATE DATABASE temporal;"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "CREATE DATABASE temporal_visibility;"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "CREATE USER temporal WITH PASSWORD '<generated_password>';"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "GRANT ALL ON DATABASE temporal TO temporal;"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c "GRANT ALL ON DATABASE temporal_visibility TO temporal;"
```

### PF-4: Resource Budget on cx42

cx42 = 8 vCPU / 16 GB RAM. The estimated request total is ~4 GB / 2 vCPU, but real usage is higher (Keycloak alone uses ~1 GB). You **may** hit memory pressure with all 22 components.

**Required:** Monitor with `kubectl top nodes` after deploying each batch of components. If memory exceeds 80%, consider:
- Deploying in waves (infra first, observability last)
- Reducing replica counts to 1 for non-critical services
- Upgrading to cx52 (16 vCPU / 32 GB, ~EUR 30/mo)

### PF-5: CNPG S3 Credentials in Keycloak Namespace

Task 5.5 creates the Keycloak CNPG cluster with WAL archiving to S3, but the `cnpg-s3-credentials` Secret is only created in `zenith-shared` (task 5.6). The `keycloak` namespace needs its own copy. When implementing task 5.5, also create:

```bash
kubectl create secret generic cnpg-s3-credentials -n keycloak \
  --from-literal=ACCESS_KEY_ID=<s3_access_key> \
  --from-literal=ACCESS_SECRET_KEY=<s3_secret_key>
```

Or add a Terraform resource to create it (preferred — keeps it declarative).

### PF-6: Outputs.tf Will Break on Kong Removal

The existing `infra/terraform/modules/k8s-platform/outputs.tf` references `kong_status`. When Kong is removed, Terraform will error. The AI must update `outputs.tf` alongside the Kong removal — replace `kong_status` with `apisix_status`.

### PF-7: HCL Snippets Are Guidance, Not Copy-Paste

The code snippets in this file are **illustrative**. The implementing AI should:
1. **Read the actual current files** (`main.tf`, `variables.tf`, `outputs.tf`) first
2. Understand the existing patterns and conventions
3. Make **surgical edits** to the real files, adapting snippets as needed
4. Never blindly overwrite existing code with snippet content

### PF-8: Pro Shard 1 CNPG Cluster (Deferred)

The architecture calls for a Pro Shard 1 CNPG cluster (~20 customers per shard, 100Gi), but there is no dedicated task for it in this file. This is intentional — no pro customers exist yet on staging. When the first pro customer signs up, add a new CNPG cluster in `zenith-shared` namespace following the same pattern as `free-pg` (task 5.6) with these differences:
- Name: `pro-shard1-pg`
- Storage: `100Gi`
- WAL path: `s3://zenith-backups/pro-shard1-wal/`
- Retention: `30d`

---

## Table of Contents

1. [Project Context](#1-project-context)
2. [Prerequisites & Secrets Inventory](#2-prerequisites--secrets-inventory)
3. [Phase 1 — Terraform: Hetzner + Cloudflare](#3-phase-1--terraform-hetzner--cloudflare)
4. [Phase 2 — Ansible: k3s + Cilium](#4-phase-2--ansible-k3s--cilium)
5. [Phase 3 — Terraform: Cluster Bootstrap](#5-phase-3--terraform-cluster-bootstrap)
6. [Phase 4 — ArgoCD App-of-Apps](#6-phase-4--argocd-app-of-apps)
7. [Phase 5 — Temporal Provisioning Workflow](#7-phase-5--temporal-provisioning-workflow)
8. [Migration Phases M1-M6](#8-migration-phases-m1-m6)
9. [End-to-End Verification](#9-end-to-end-verification)

---

## 1. Project Context

### What is Zenith

Zenith is a **Kubernetes-native Platform-as-a-Service (PaaS)** built on Hetzner Cloud, operated by DoTech. It offers 4 tiers:

| Tier | Infrastructure | Isolation | Database | S3 | Identity |
|------|---------------|-----------|----------|----|----------|
| **Free** | Shared k3s cluster | Namespace (Cilium) | Shared CNPG cluster, own DB | Shared account, own bucket | Shared Keycloak, own realm |
| **Pro** | Shared k3s cluster | Namespace (sharded) | Sharded CNPG (~20/cluster), up to 5 DBs | Shared account, own bucket | Shared Keycloak, own realm |
| **Team** | Dedicated VMs (CAPI+CAPH) | Kernel | Own CNPG | Own everything | Own Keycloak |
| **Enterprise** | Dedicated VMs (CAPI+CAPH) | Kernel | Own CNPG | Own everything | Own Keycloak |

**Domain:** freezenith.com

### What is V2

V2 replaces the manually-deployed V1 with a fully automated, security-hardened, observable platform. Key additions: APISIX (replaces Kong), Keycloak, ArgoCD, Temporal, Harbor, external-dns, Kyverno, Falco, Sealed Secrets, Velero, Hubble, OTel/Tempo, PriorityClasses, DNS-01 solver, CNPG sharding, WireGuard encryption.

### Current State (V1)

| Component | Production (DigitalOcean) | Staging (Hetzner) |
|-----------|--------------------------|-------------------|
| Server | 161.35.82.211, 4 vCPU/8GB | 77.42.88.149, 4 vCPU/8GB |
| K8s | k3s v1.34.3 | k3s v1.34.3 |
| CNI | Flannel (no NetworkPolicy) | Cilium 1.16.5 |
| Ingress | Traefik 3.5.1 (IngressRoute CRD) | Traefik 3.5.1 |
| API Gateway | Kong DB-less | Kong DB-less |
| TLS | cert-manager, HTTP-01 | cert-manager, HTTP-01 |
| Database | CNPG, 1 shared cluster | CNPG, 1 shared cluster |
| GitOps | None (manual deploy.sh) | Terraform + Helm |
| Monitoring | None | Prometheus + Grafana + Loki |
| Backup | None | None |

**Live URLs (V1 Production):**
- freezenith.com (landing), api.freezenith.com (API)
- demo-ms.freezenith.com (MC demo), demo-cloud.freezenith.com (Web demo)
- ms.embermind.app (customer MC), cloud.embermind.app (customer Web)

### Target State (V2)

```
Phase 1: Terraform → Hetzner VM (cx42) + Cloudflare DNS (all subdomains)
Phase 2: Ansible  → k3s + Cilium (WireGuard + Hubble) + hcloud-csi + secrets encryption
Phase 3: Terraform → 22 infra components (cert-manager, CNPG, Keycloak, APISIX, ArgoCD, ...)
Phase 4: ArgoCD   → Application charts (automatic from Git push)
```

### Repository Structure

```
Zenith/
├── apps/
│   ├── landing/              # Next.js landing page (freezenith.com)
│   ├── mission-control/      # Operator admin panel (ms.{domain})
│   └── web/                  # User dashboard (cloud.{domain})
├── services/
│   ├── api/                  # Go REST API (port 8080)
│   ├── auth/                 # Go OIDC/SAML auth service (port 8090)
│   └── operator/             # Go K8s operator (controller-runtime)
├── cli/                      # zen CLI (Go + Cobra + Charm TUI)
├── infra/
│   ├── terraform/
│   │   ├── staging/          # Phase 1: Hetzner VM + Cloudflare DNS
│   │   ├── staging-k8s/      # Phase 3: K8s platform components
│   │   └── modules/
│   │       ├── k3s-server/   # Hetzner VM module
│   │       ├── dns/          # Cloudflare DNS module
│   │       └── k8s-platform/ # All Helm releases (to be extended for V2)
│   ├── ansible/              # Phase 2: k3s + Cilium + OS hardening
│   ├── helm/                 # Modular Helm charts
│   │   ├── zenith-platform/  # Shared resources (secrets, certs, middleware)
│   │   ├── zenith-api/       # Go API server
│   │   ├── zenith-landing/   # Landing page
│   │   ├── zenith-demo/      # Demo instances
│   │   ├── zenith-tenant/    # Per-customer deployments
│   │   └── monitoring/       # Prometheus + Grafana + Loki
│   └── argocd/               # ArgoCD Application manifests (App-of-Apps)
└── docs/
    └── v2-architecture/      # V2 design docs (this file + 10 others)
```

### V2 Architecture Doc References

| Doc | Purpose |
|-----|---------|
| `00-overview.md` | Full architecture overview, component stack, 4-tier model |
| `01-phase1-hetzner-cloudflare.md` | Hetzner VM + Cloudflare DNS details |
| `02-phase2-ansible-k3s.md` | k3s + Cilium + OS hardening details |
| `03-phase3-cluster-bootstrap.md` | All 22 infra components (largest doc) |
| `04-phase4-argocd-apps.md` | ArgoCD App-of-Apps pattern |
| `05-user-flows.md` | Customer signup, login, deploy, provisioning flows |
| `06-security-model.md` | 6-layer defense-in-depth model |
| `07-backup-disaster-recovery.md` | 3-layer backup strategy |
| `08-observability.md` | Prometheus, Loki, Tempo, OTel, Hubble |
| `09-migration-v1-to-v2.md` | V1→V2 migration phases M1-M6 |

---

## 2. Prerequisites & Secrets Inventory

### Required API Tokens & Credentials

| Secret | Purpose | Where to Get |
|--------|---------|-------------|
| `HCLOUD_TOKEN` | Hetzner Cloud API | Hetzner Console → Project → Security → API Tokens |
| `CLOUDFLARE_API_TOKEN` | DNS management | Cloudflare Dashboard → API Tokens (Zone:DNS:Edit) |
| `CLOUDFLARE_ZONE_ID` | freezenith.com zone | Cloudflare Dashboard → freezenith.com → Overview |
| `SSH_PUBLIC_KEY` | Server access | `~/.ssh/id_ed25519.pub` |
| `JWT_SECRET` | API token signing | Generate: `openssl rand -hex 32` |
| `ADMIN_EMAIL` | Platform admin | Your email |
| `ADMIN_PASSWORD` | Platform admin | Generate strong password |
| `HETZNER_S3_ACCESS_KEY` | Object storage (backups) | Hetzner Console → Object Storage → Access Keys |
| `HETZNER_S3_SECRET_KEY` | Object storage (backups) | Same as above |
| `REGISTRY_USERNAME` | Harbor robot account | Created after Harbor is deployed |
| `REGISTRY_PASSWORD` | Harbor robot account | Created after Harbor is deployed |

### Required Tools

| Tool | Version | Install |
|------|---------|---------|
| terraform | >= 1.5 | `brew install terraform` |
| ansible | >= 2.15 | `brew install ansible` |
| helm | >= 3.14 | `brew install helm` |
| kubectl | >= 1.30 | `brew install kubectl` |
| cilium | >= 0.16 | `brew install cilium-cli` |
| kubeseal | >= 0.27 | `brew install kubeseal` |
| argocd | >= 2.13 | `brew install argocd` |
| gh | >= 2.40 | `brew install gh` |

### Server Requirements

| Environment | Server Type | CPU | RAM | Disk | Monthly Cost |
|-------------|-------------|-----|-----|------|-------------|
| Development | cx22 | 2 vCPU | 4 GB | 40 GB | ~EUR 4 |
| **Staging (V2)** | **cx42** | **8 vCPU** | **16 GB** | **160 GB** | **~EUR 15** |
| Production | cx52 | 16 vCPU | 32 GB | 320 GB | ~EUR 30 |

---

## 3. Phase 1 — Terraform: Hetzner + Cloudflare

> **Reference:** `docs/v2-architecture/01-phase1-hetzner-cloudflare.md`
> **Directory:** `infra/terraform/staging/`
> **Time:** ~3 minutes

### Current State

File `infra/terraform/staging/main.tf` already exists with:
- Hetzner VM module (`staging_server`) — server type configurable via `var.server_type`
- DNS module with platform records: `stage`, `api.stage`, `ms.stage`, `cloud.stage`, `grafana.stage`, `prometheus.stage`
- Customer records: `embermind-ms.stage`, `embermind.stage`

### Tasks

#### 3.1: Upgrade Server Type

- [x] Update `terraform.tfvars` to use `cx42` (8 vCPU, 16 GB RAM) for V2 workloads — already cx43/16GB

**File:** `infra/terraform/staging/terraform.tfvars`
```hcl
server_type = "cx42"
```

**Validation:**
```bash
cd infra/terraform/staging
terraform plan | grep server_type
```
**Expected:** `server_type = "cx42"`

#### 3.2: Add V2 DNS Records

- [x] Add new subdomains for V2 infrastructure services

**File:** `infra/terraform/staging/main.tf` — modify the `platform_records` map:
```hcl
module "dns" {
  source = "../modules/dns"

  zone_id   = var.freezenith_zone_id
  server_ip = var.create_server ? module.staging_server[0].server_ip : var.existing_server_ip

  # Platform services (V1 + V2 additions)
  platform_records = {
    root       = { name = "stage" }
    api        = { name = "api.stage" }
    ms         = { name = "ms.stage" }
    cloud      = { name = "cloud.stage" }
    grafana    = { name = "grafana.stage" }
    prometheus = { name = "prometheus.stage" }
    # --- V2 additions ---
    argocd     = { name = "argocd.stage" }
    keycloak   = { name = "auth.stage" }
    temporal   = { name = "temporal.stage" }
    harbor     = { name = "registry.stage" }
    hubble     = { name = "hubble.stage" }
    tempo      = { name = "tempo.stage" }
    alertmanager = { name = "alerts.stage" }
  }

  # Customer: embermind on staging
  customer_records = {
    embermind_ms  = { zone_id = var.freezenith_zone_id, name = "embermind-ms.stage" }
    embermind_web = { zone_id = var.freezenith_zone_id, name = "embermind.stage" }
  }
}
```

**Validation:**
```bash
cd infra/terraform/staging
terraform plan
```
**Expected:** Plan shows 7 new DNS records to add (argocd, auth, temporal, registry, hubble, tempo, alerts).

#### 3.3: Apply Phase 1

- [x] Apply Terraform to create/update server and DNS

```bash
cd infra/terraform/staging
terraform apply
```

**Validation:**
```bash
# Verify DNS resolves
for sub in stage api.stage ms.stage cloud.stage grafana.stage prometheus.stage \
           argocd.stage auth.stage temporal.stage registry.stage hubble.stage; do
  echo "$sub.freezenith.com → $(dig +short $sub.freezenith.com)"
done
```
**Expected:** All subdomains resolve to the server IP.

#### 3.4: Verify SSH Access

- [x] Verify SSH connectivity to the new/updated server

```bash
ssh root@$(terraform output -raw server_ip) "hostname && uname -a"
```
**Expected:** Server hostname and Linux kernel version.

---

## 4. Phase 2 — Ansible: k3s + Cilium

> **Reference:** `docs/v2-architecture/02-phase2-ansible-k3s.md`
> **Directory:** `infra/ansible/`
> **Time:** ~10 minutes

### What Phase 2 Does

Installs k3s with Cilium CNI, OS hardening, and CSI driver on the bare server from Phase 1. V2 enhancements add WireGuard encryption, Hubble observability, and etcd secrets encryption.

### Current State

Ansible roles exist for: common, k3s, cilium, cert-manager, postgres, traefik-config, app build/deploy roles, KEDA, monitoring, DNS. The k3s and cilium roles need V2 flag additions.

### Tasks

#### 4.1: Enable WireGuard Encryption on Cilium

- [x] Add WireGuard encryption flag to Cilium role

**File:** `infra/ansible/roles/cilium/tasks/main.yml` — add to the `cilium install` command:
```yaml
- name: Install Cilium with WireGuard encryption
  shell: |
    cilium install \
      --set kubeProxyReplacement=true \
      --set k8sServiceHost=127.0.0.1 \
      --set k8sServicePort=6443 \
      --set hubble.enabled=true \
      --set hubble.relay.enabled=true \
      --set hubble.ui.enabled=true \
      --set encryption.enabled=true \
      --set encryption.type=wireguard
  when: enable_cilium | default(true)
```

**File:** `infra/ansible/group_vars/all.yml` — add variables:
```yaml
# V2: Cilium enhancements
enable_wireguard: true
enable_hubble: true
enable_hubble_ui: true
```

**Validation:**
```bash
cilium status
```
**Expected:** `Encryption: Wireguard` in status output.

#### 4.2: Enable Hubble Observability

- [x] Verify Hubble is enabled (configured in 4.1 via `hubble.enabled=true`)

**Validation:**
```bash
cilium hubble port-forward &
hubble status
hubble observe --last 10
```
**Expected:** Hubble relay is connected, recent network flows visible.

#### 4.3: Enable k3s Secrets Encryption

- [x] Add `--secrets-encryption` flag to k3s installation

**File:** `infra/ansible/roles/k3s/tasks/main.yml` — update the k3s install command:
```yaml
- name: Install k3s with Cilium and secrets encryption
  shell: |
    curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="{{ k3s_version }}" sh -s - server \
      --flannel-backend=none \
      --disable-network-policy \
      --disable=servicelb \
      --secrets-encryption \
      --write-kubeconfig-mode 644
```

**File:** `infra/ansible/group_vars/all.yml` — add/verify:
```yaml
k3s_version: "v1.34.3+k3s1"
enable_secrets_encryption: true
```

**Validation:**
```bash
# On the server:
k3s secrets-encrypt status
```
**Expected:** `Encryption Status: Enabled`

#### 4.4: Configure cert-manager for DNS-01 (Prepare)

- [x] Prepare Cloudflare API token secret for DNS-01 solver (applied in Phase 3)

This is a preparation step. The actual ClusterIssuer is created by Terraform in Phase 3. Here we ensure the Ansible playbook creates the Cloudflare API token as a K8s Secret that cert-manager will reference.

**File:** `infra/ansible/roles/cert-manager/tasks/main.yml` — add task:
```yaml
- name: Create Cloudflare API token secret for DNS-01
  kubernetes.core.k8s:
    state: present
    definition:
      apiVersion: v1
      kind: Secret
      metadata:
        name: cloudflare-api-token
        namespace: cert-manager
      type: Opaque
      stringData:
        api-token: "{{ cloudflare_api_token }}"
  when: enable_dns01_solver | default(true)
```

**File:** `infra/ansible/group_vars/all.yml`:
```yaml
enable_dns01_solver: true
# cloudflare_api_token set in vault or inventory
```

**Validation:**
```bash
kubectl get secret cloudflare-api-token -n cert-manager
```
**Expected:** Secret exists in cert-manager namespace.

#### 4.5: Run Ansible Playbook

- [x] Execute the full Ansible playbook

```bash
cd infra/ansible
ansible-playbook -i inventory/staging.yml playbooks/site.yml
```

**Validation:**
```bash
# Verify k3s is running
kubectl get nodes
# Verify Cilium
cilium status
# Verify WireGuard
cilium encrypt status
# Verify Hubble
kubectl get pods -n kube-system | grep hubble
# Verify secrets encryption
ssh root@<SERVER_IP> "k3s secrets-encrypt status"
```

**Expected output:**
```
NAME              STATUS   ROLES                  AGE   VERSION
zenith-staging    Ready    control-plane,master   Xm    v1.34.3+k3s1

    /¯¯\
 /¯¯\__/¯¯\    Cilium:             OK
 \__/¯¯\__/    Operator:           OK
 /¯¯\__/¯¯\    Envoy DaemonSet:    OK
 \__/¯¯\__/    Hubble Relay:       OK
    \__/        Encryption:         Wireguard

Encryption Status: Enabled
```

---

## 5. Phase 3 — Terraform: Cluster Bootstrap

> **Reference:** `docs/v2-architecture/03-phase3-cluster-bootstrap.md`
> **Directory:** `infra/terraform/staging-k8s/` and `infra/terraform/modules/k8s-platform/`
> **Time:** 15-25 minutes (first apply), 3-5 minutes (incremental)

### Overview

Phase 3 is the largest phase. It takes the bare k3s cluster from Phase 2 and installs every infrastructure component. All components are Terraform-managed Helm releases in the `k8s-platform` module.

**Current state:** The module already has cert-manager, CNPG, Kong, KEDA, monitoring, and app charts.
**V2 changes:** Replace Kong with APISIX, add 15+ new components, upgrade cert-manager to DNS-01.

### Namespace Layout (V2)

```
cert-manager      → TLS automation
cnpg-system       → PostgreSQL operator
keycloak          → Identity provider + dedicated PG
apisix            → API gateway + etcd
external-dns      → Auto DNS
argocd            → GitOps engine
temporal          → Provisioning workflows
harbor            → Container registry
kyverno           → Policy enforcement
falco             → Runtime security
sealed-secrets    → Encrypted secrets
velero            → Cluster backup
monitoring        → Prometheus, Grafana, Loki, Tempo, OTel
zenith-shared     → Shared CNPG clusters (free + pro shards)
zenith-platform   → Platform services (zenith-api, zenith-landing)
```

### Dependency Graph

```
PriorityClasses (no deps)
  ↓
cert-manager → ClusterIssuer (DNS-01)
  ↓
Sealed Secrets (needs cert-manager)
  ↓
CNPG Operator
  ├── CNPG Keycloak Cluster → Keycloak
  ├── CNPG Free Cluster
  └── CNPG Pro Shard 1 Cluster
  ↓
APISIX + etcd (needs cert-manager for TLS)
  ↓
external-dns (needs Cloudflare token)
  ↓
ArgoCD (needs cert-manager)
  ↓
Harbor (needs cert-manager, S3 storage)
  ↓
Temporal (needs CNPG for its DB)
  ↓
(parallel) Kyverno, Falco, Velero
  ↓
(parallel) Prometheus+Grafana, Loki, Tempo, OTel, Hubble UI
  ↓
ResourceQuotas + LimitRanges + PodDisruptionBudgets
```

### Component Implementation Details

---

#### 5.1: PriorityClasses

- [x] Create 4 PriorityClasses for pod eviction ordering

**Files:** `infra/terraform/modules/k8s-platform/main.tf` — add at the top (no dependencies):
```hcl
# =============================================================================
# PriorityClasses — Pod eviction ordering
# =============================================================================

resource "kubernetes_priority_class" "system_critical" {
  metadata {
    name = "system-critical"
  }
  value          = 1000000
  global_default = false
  description    = "Cilium, CoreDNS, Traefik — never evict"
}

resource "kubernetes_priority_class" "infra_critical" {
  metadata {
    name = "infra-critical"
  }
  value          = 500000
  global_default = false
  description    = "CNPG, Keycloak, APISIX, cert-manager — evict last"
}

resource "kubernetes_priority_class" "platform" {
  metadata {
    name = "platform"
  }
  value          = 100000
  global_default = false
  description    = "zenith-api, Temporal, Harbor, monitoring"
}

resource "kubernetes_priority_class" "customer" {
  metadata {
    name = "customer"
  }
  value          = 10000
  global_default = true
  description    = "All customer workloads — evict first"
}
```

- **Depends on:** Nothing
- **Validation:**
```bash
kubectl get priorityclasses
```
- **Expected:** 4 custom PriorityClasses plus the 2 system defaults.

---

#### 5.2: cert-manager DNS-01 Upgrade

- [x] Modify existing cert-manager ClusterIssuer from HTTP-01 to DNS-01

**File:** `infra/terraform/modules/k8s-platform/main.tf` — replace the `cluster_issuer` resource:
```hcl
# =============================================================================
# cert-manager — TLS certificate automation (already exists, keep as-is)
# =============================================================================

# (keep existing helm_release.cert_manager unchanged)

# --- V2: DNS-01 solver (replaces HTTP-01) ---
# Requires: cloudflare-api-token Secret in cert-manager namespace (created by Ansible Phase 2)

resource "kubernetes_manifest" "cluster_issuer" {
  manifest = {
    apiVersion = "cert-manager.io/v1"
    kind       = "ClusterIssuer"
    metadata = {
      name = "letsencrypt-prod"
    }
    spec = {
      acme = {
        server = "https://acme-v02.api.letsencrypt.org/directory"
        email  = var.cert_issuer_email
        privateKeySecretRef = {
          name = "letsencrypt-prod-key"
        }
        solvers = [{
          dns01 = {
            cloudflare = {
              apiTokenSecretRef = {
                name = "cloudflare-api-token"
                key  = "api-token"
              }
            }
          }
          selector = {
            dnsZones = ["freezenith.com"]
          }
        }]
      }
    }
  }

  depends_on = [helm_release.cert_manager]
}
```

- **Depends on:** cert-manager Helm release, `cloudflare-api-token` Secret (from Ansible Phase 2)
- **Why DNS-01:** Enables Cloudflare proxy ON (WAF + DDoS), allows wildcard certs, works for all subdomains
- **Validation:**
```bash
kubectl get clusterissuer letsencrypt-prod -o yaml | grep -A5 solvers
```
- **Expected:** Shows `dns01.cloudflare` solver instead of `http01`.

---

#### 5.3: Sealed Secrets

- [x] Add Sealed Secrets controller for GitOps-safe encrypted secrets

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add new resource:
```hcl
# =============================================================================
# Sealed Secrets — Encrypted secrets for GitOps
# =============================================================================

resource "helm_release" "sealed_secrets" {
  count = var.enable_sealed_secrets ? 1 : 0

  name             = "sealed-secrets"
  repository       = "https://bitnami-labs.github.io/sealed-secrets"
  chart            = "sealed-secrets"
  version          = var.sealed_secrets_version
  namespace        = "sealed-secrets"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "resources.requests.cpu"
    value = "25m"
  }

  set {
    name  = "resources.requests.memory"
    value = "64Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "100m"
  }

  set {
    name  = "resources.limits.memory"
    value = "128Mi"
  }

  depends_on = [helm_release.cert_manager]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "enable_sealed_secrets" {
  description = "Install Sealed Secrets controller"
  type        = bool
  default     = false
}

variable "sealed_secrets_version" {
  description = "Sealed Secrets Helm chart version"
  type        = string
  default     = "2.17.0"
}
```

- **Depends on:** cert-manager
- **Important:** Back up the Sealed Secrets controller private key! If lost, all SealedSecrets become undecryptable.
- **Validation:**
```bash
kubectl get pods -n sealed-secrets
kubeseal --fetch-cert --controller-name=sealed-secrets --controller-namespace=sealed-secrets
```
- **Expected:** Controller pod running, public certificate retrievable.

---

#### 5.4: CNPG Operator Upgrade

- [x] Verify existing CNPG operator (already installed, version 0.23.0)

**File:** `infra/terraform/modules/k8s-platform/main.tf` — existing `helm_release.cnpg` is fine. Ensure cluster-wide watching:
```hcl
# (existing resource, add this set block if not present)
  set {
    name  = "config.data.INHERITED_ANNOTATIONS"
    value = "cert-manager.io/*"
  }

  set {
    name  = "config.data.INHERITED_LABELS"
    value = "app.kubernetes.io/*"
  }
```

- **Depends on:** cert-manager
- **Validation:**
```bash
kubectl get pods -n cnpg-system
kubectl get crds | grep cnpg
```
- **Expected:** `cnpg-cloudnative-pg-*` pod running. CRDs: `clusters.postgresql.cnpg.io`, `backups.postgresql.cnpg.io`, etc.

---

#### 5.5: CNPG Keycloak Cluster (Dedicated)

- [x] Create a dedicated CNPG PostgreSQL cluster for Keycloak
- [x] Create S3 credentials secret in keycloak namespace (see PF-5)

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# CNPG Cluster — Dedicated PostgreSQL for Keycloak
# =============================================================================

# IMPORTANT: S3 credentials must exist in the keycloak namespace for WAL archiving
resource "kubernetes_secret" "cnpg_s3_credentials_keycloak" {
  count = var.enable_keycloak ? 1 : 0

  metadata {
    name      = "cnpg-s3-credentials"
    namespace = "keycloak"
  }

  data = {
    ACCESS_KEY_ID     = var.s3_access_key
    ACCESS_SECRET_KEY = var.s3_secret_key
  }

  depends_on = [kubernetes_namespace.keycloak]
}
```

Also add in the same file (after the secret above):

resource "kubernetes_namespace" "keycloak" {
  count = var.enable_keycloak ? 1 : 0
  metadata {
    name = "keycloak"
  }
}

resource "kubernetes_manifest" "cnpg_keycloak" {
  count = var.enable_keycloak ? 1 : 0

  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"
    metadata = {
      name      = "keycloak-pg"
      namespace = "keycloak"
    }
    spec = {
      instances    = var.environment == "production" ? 3 : 2
      primaryUpdateStrategy = "unsupervised"

      storage = {
        storageClass = "hcloud-volumes"
        size         = "10Gi"
      }

      postgresql = {
        parameters = {
          max_connections        = "100"
          shared_buffers         = "128MB"
          effective_cache_size   = "256MB"
          work_mem               = "4MB"
          maintenance_work_mem   = "64MB"
        }
      }

      bootstrap = {
        initdb = {
          database = "keycloak"
          owner    = "keycloak"
        }
      }

      backup = {
        barmanObjectStore = {
          destinationPath = "s3://zenith-backups/keycloak-wal/"
          endpointURL     = var.s3_endpoint
          s3Credentials = {
            accessKeyId = {
              name = "cnpg-s3-credentials"
              key  = "ACCESS_KEY_ID"
            }
            secretAccessKey = {
              name = "cnpg-s3-credentials"
              key  = "ACCESS_SECRET_KEY"
            }
          }
          wal = {
            compression = "gzip"
            maxParallel = 4
          }
        }
        retentionPolicy = "14d"
      }

      monitoring = {
        enablePodMonitor = true
      }

      priorityClassName = "infra-critical"
    }
  }

  depends_on = [
    helm_release.cnpg,
    kubernetes_namespace.keycloak,
    kubernetes_priority_class.infra_critical,
  ]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "enable_keycloak" {
  description = "Install Keycloak identity provider"
  type        = bool
  default     = false
}

variable "environment" {
  description = "Environment name (staging or production)"
  type        = string
  default     = "staging"
}

variable "s3_endpoint" {
  description = "Hetzner S3 endpoint URL for backups"
  type        = string
  default     = "https://fsn1.your-objectstorage.com"
}
```

- **Depends on:** CNPG operator, PriorityClasses
- **Validation:**
```bash
kubectl get clusters.postgresql.cnpg.io -n keycloak
kubectl get pods -n keycloak -l cnpg.io/cluster=keycloak-pg
```
- **Expected:** Cluster status `Cluster in healthy state`, 2 pods running (1 primary, 1 replica).

---

#### 5.6: CNPG Free Cluster (Shared)

- [x] Create shared CNPG cluster for all free-tier customers

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# CNPG Cluster — Shared PostgreSQL for Free-tier customers
# =============================================================================

resource "kubernetes_namespace" "zenith_shared" {
  metadata {
    name = "zenith-shared"
  }
}

resource "kubernetes_manifest" "cnpg_s3_credentials_shared" {
  manifest = {
    apiVersion = "v1"
    kind       = "Secret"
    metadata = {
      name      = "cnpg-s3-credentials"
      namespace = "zenith-shared"
    }
    type = "Opaque"
    stringData = {
      ACCESS_KEY_ID     = var.s3_access_key
      ACCESS_SECRET_KEY = var.s3_secret_key
    }
  }

  depends_on = [kubernetes_namespace.zenith_shared]
}

resource "kubernetes_manifest" "cnpg_free" {
  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"
    metadata = {
      name      = "free-pg"
      namespace = "zenith-shared"
    }
    spec = {
      instances    = var.environment == "production" ? 3 : 2
      primaryUpdateStrategy = "unsupervised"

      storage = {
        storageClass = "hcloud-volumes"
        size         = "50Gi"
      }

      postgresql = {
        parameters = {
          max_connections        = "200"
          shared_buffers         = "256MB"
          effective_cache_size   = "512MB"
          work_mem               = "4MB"
          maintenance_work_mem   = "64MB"
          statement_timeout      = "30000"
        }
      }

      bootstrap = {
        initdb = {
          database = "zenith_platform"
          owner    = "zenith_admin"
        }
      }

      backup = {
        barmanObjectStore = {
          destinationPath = "s3://zenith-backups/free-pg-wal/"
          endpointURL     = var.s3_endpoint
          s3Credentials = {
            accessKeyId = {
              name = "cnpg-s3-credentials"
              key  = "ACCESS_KEY_ID"
            }
            secretAccessKey = {
              name = "cnpg-s3-credentials"
              key  = "ACCESS_SECRET_KEY"
            }
          }
          wal = {
            compression = "gzip"
            maxParallel = 4
          }
        }
        retentionPolicy = "14d"
      }

      monitoring = {
        enablePodMonitor = true
      }

      priorityClassName = "infra-critical"
    }
  }

  depends_on = [
    helm_release.cnpg,
    kubernetes_namespace.zenith_shared,
    kubernetes_priority_class.infra_critical,
    kubernetes_manifest.cnpg_s3_credentials_shared,
  ]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "s3_access_key" {
  description = "Hetzner S3 access key for CNPG WAL archiving"
  type        = string
  sensitive   = true
  default     = ""
}

variable "s3_secret_key" {
  description = "Hetzner S3 secret key for CNPG WAL archiving"
  type        = string
  sensitive   = true
  default     = ""
}
```

- **Depends on:** CNPG operator, PriorityClasses
- **Validation:**
```bash
kubectl get clusters.postgresql.cnpg.io -n zenith-shared
kubectl get pods -n zenith-shared -l cnpg.io/cluster=free-pg
```
- **Expected:** `free-pg` cluster healthy, 2 pods running.

#### 5.6.1: Create Temporal Databases in free-pg (Pre-requisite for 5.12)

- [x] Create `temporal` and `temporal_visibility` databases inside the free-pg cluster

**IMPORTANT:** This must be done **after** free-pg is healthy (5.6) and **before** Temporal is deployed (5.12). See PF-3.

```bash
# Wait for free-pg to be ready
kubectl wait --for=condition=Ready cluster/free-pg -n zenith-shared --timeout=300s

# Create Temporal databases and user
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c \
  "CREATE DATABASE temporal;"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c \
  "CREATE DATABASE temporal_visibility;"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c \
  "CREATE USER temporal WITH PASSWORD '<TEMPORAL_DB_PASSWORD_FROM_PF2>';"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c \
  "GRANT ALL ON DATABASE temporal TO temporal;"
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -c \
  "GRANT ALL ON DATABASE temporal_visibility TO temporal;"
```

**Validation:**
```bash
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -l | grep temporal
```
**Expected:** Both `temporal` and `temporal_visibility` databases listed.

---

#### 5.7: Keycloak

- [x] Deploy Keycloak identity provider with dedicated CNPG database

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# Keycloak — Identity Provider (realm per customer)
# =============================================================================

resource "helm_release" "keycloak" {
  count = var.enable_keycloak ? 1 : 0

  name             = "keycloak"
  repository       = "https://charts.bitnami.com/bitnami"
  chart            = "keycloak"
  version          = var.keycloak_version
  namespace        = "keycloak"
  create_namespace = false
  wait             = true
  timeout          = 600

  # Use external CNPG database (not built-in PostgreSQL)
  set {
    name  = "postgresql.enabled"
    value = "false"
  }

  set {
    name  = "externalDatabase.host"
    value = "keycloak-pg-rw.keycloak.svc.cluster.local"
  }

  set {
    name  = "externalDatabase.port"
    value = "5432"
  }

  set {
    name  = "externalDatabase.database"
    value = "keycloak"
  }

  set {
    name  = "externalDatabase.user"
    value = "keycloak"
  }

  set_sensitive {
    name  = "externalDatabase.password"
    value = var.keycloak_db_password
  }

  # Admin credentials
  set_sensitive {
    name  = "auth.adminUser"
    value = "admin"
  }

  set_sensitive {
    name  = "auth.adminPassword"
    value = var.keycloak_admin_password
  }

  # Resources (staging)
  set {
    name  = "resources.requests.cpu"
    value = "200m"
  }

  set {
    name  = "resources.requests.memory"
    value = "512Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "1"
  }

  set {
    name  = "resources.limits.memory"
    value = "1Gi"
  }

  # Hostname
  set {
    name  = "httpRelativePath"
    value = "/"
  }

  set {
    name  = "proxy"
    value = "edge"
  }

  set {
    name  = "priorityClassName"
    value = "infra-critical"
  }

  depends_on = [
    kubernetes_manifest.cnpg_keycloak,
    kubernetes_priority_class.infra_critical,
  ]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "keycloak_version" {
  description = "Keycloak Helm chart version"
  type        = string
  default     = "24.4.0"
}

variable "keycloak_db_password" {
  description = "Keycloak database password"
  type        = string
  sensitive   = true
  default     = ""
}

variable "keycloak_admin_password" {
  description = "Keycloak admin console password"
  type        = string
  sensitive   = true
  default     = ""
}
```

- **Depends on:** CNPG Keycloak cluster (5.5), PriorityClasses
- **JWKS endpoint:** `https://auth.stage.freezenith.com/realms/<customer>/protocol/openid-connect/certs`
- **Validation:**
```bash
kubectl get pods -n keycloak -l app.kubernetes.io/name=keycloak
kubectl logs -n keycloak -l app.kubernetes.io/name=keycloak --tail=20
```
- **Expected:** Keycloak pod running, logs show `Keycloak ... started in Xs`.

---

#### 5.8: APISIX + etcd (Replaces Kong)

- [x] Add APISIX API gateway with etcd backend
- [x] Remove Kong (see PF-1: deploy APISIX first, migrate routes, then remove Kong)
- [x] Update `outputs.tf`: replace `kong_status` with `apisix_status` (see PF-6)

**WARNING:** Do NOT remove Kong and add APISIX in the same apply. See PF-1 for the safe migration order.

**File:** `infra/terraform/modules/k8s-platform/outputs.tf` — replace:
```hcl
# Remove this:
output "kong_status" { ... }

# Add this:
output "apisix_status" {
  description = "APISIX API Gateway release status"
  value       = var.enable_apisix ? helm_release.apisix[0].status : "disabled"
}
```

**File:** `infra/terraform/modules/k8s-platform/main.tf` — **remove** the entire `helm_release.kong` resource block (after APISIX is verified working). Then add:
```hcl
# =============================================================================
# APISIX + etcd — API Gateway (replaces Kong)
# =============================================================================

resource "helm_release" "apisix" {
  count = var.enable_apisix ? 1 : 0

  name             = "apisix"
  repository       = "https://charts.apiseven.com"
  chart            = "apisix"
  version          = var.apisix_version
  namespace        = "apisix"
  create_namespace = true
  wait             = true
  timeout          = 600

  # etcd configuration (built into APISIX chart)
  set {
    name  = "etcd.replicaCount"
    value = var.environment == "production" ? "3" : "1"
  }

  set {
    name  = "etcd.persistence.enabled"
    value = "true"
  }

  set {
    name  = "etcd.persistence.storageClass"
    value = "hcloud-volumes"
  }

  set {
    name  = "etcd.persistence.size"
    value = "5Gi"
  }

  # Gateway
  set {
    name  = "gateway.type"
    value = "ClusterIP"
  }

  set {
    name  = "replicaCount"
    value = var.environment == "production" ? "2" : "1"
  }

  # Enable plugins
  set {
    name  = "plugins.jwt-auth"
    value = "true"
  }

  set {
    name  = "plugins.cors"
    value = "true"
  }

  set {
    name  = "plugins.limit-count"
    value = "true"
  }

  set {
    name  = "plugins.opentelemetry"
    value = "true"
  }

  set {
    name  = "plugins.prometheus"
    value = "true"
  }

  # Resources (staging)
  set {
    name  = "resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "500m"
  }

  set {
    name  = "resources.limits.memory"
    value = "512Mi"
  }

  set {
    name  = "priorityClassName"
    value = "infra-critical"
  }

  depends_on = [
    kubernetes_manifest.cluster_issuer,
    kubernetes_priority_class.infra_critical,
  ]
}

# APISIX Ingress Controller — watches ApisixRoute CRDs
resource "helm_release" "apisix_ingress" {
  count = var.enable_apisix ? 1 : 0

  name       = "apisix-ingress-controller"
  repository = "https://charts.apiseven.com"
  chart      = "apisix-ingress-controller"
  version    = var.apisix_ingress_version
  namespace  = "apisix"
  wait       = true
  timeout    = 300

  set {
    name  = "config.apisix.adminAPIVersion"
    value = "v3"
  }

  set {
    name  = "config.apisix.serviceNamespace"
    value = "apisix"
  }

  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }

  depends_on = [helm_release.apisix]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add (and remove `enable_kong`, `kong_version`):
```hcl
variable "enable_apisix" {
  description = "Install APISIX API Gateway (replaces Kong)"
  type        = bool
  default     = false
}

variable "apisix_version" {
  description = "APISIX Helm chart version"
  type        = string
  default     = "2.10.0"
}

variable "apisix_ingress_version" {
  description = "APISIX Ingress Controller Helm chart version"
  type        = string
  default     = "0.14.0"
}
```

- **Depends on:** cert-manager, PriorityClasses
- **Replaces:** Kong DB-less gateway
- **Routing model:** Single APISIX deployment, route-level plugins (jwt-auth, cors, rate-limit per route)
- **Validation:**
```bash
kubectl get pods -n apisix
kubectl get svc -n apisix
kubectl get apisixroutes --all-namespaces
```
- **Expected:** APISIX gateway pod + etcd pod + ingress controller pod running. ClusterIP service on ports 80/443.

---

#### 5.9: external-dns

- [x] Add external-dns for automatic Cloudflare DNS record creation from Ingress resources

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# external-dns — Automatic DNS record management via Cloudflare
# =============================================================================

resource "helm_release" "external_dns" {
  count = var.enable_external_dns ? 1 : 0

  name             = "external-dns"
  repository       = "https://charts.bitnami.com/bitnami"
  chart            = "external-dns"
  version          = var.external_dns_version
  namespace        = "external-dns"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "provider"
    value = "cloudflare"
  }

  set_sensitive {
    name  = "cloudflare.apiToken"
    value = var.cloudflare_api_token
  }

  set {
    name  = "domainFilters[0]"
    value = "freezenith.com"
  }

  set {
    name  = "policy"
    value = "sync"
  }

  set {
    name  = "txtOwnerId"
    value = "zenith-staging"
  }

  set {
    name  = "resources.requests.cpu"
    value = "25m"
  }

  set {
    name  = "resources.requests.memory"
    value = "64Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "100m"
  }

  set {
    name  = "resources.limits.memory"
    value = "128Mi"
  }

  depends_on = [kubernetes_manifest.cluster_issuer]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "enable_external_dns" {
  description = "Install external-dns for automatic Cloudflare DNS"
  type        = bool
  default     = false
}

variable "external_dns_version" {
  description = "external-dns Helm chart version"
  type        = string
  default     = "8.7.0"
}

variable "cloudflare_api_token" {
  description = "Cloudflare API token for external-dns and cert-manager DNS-01"
  type        = string
  sensitive   = true
  default     = ""
}
```

- **Depends on:** cert-manager (for CRD awareness)
- **How it works:** Watches Ingress/IngressRoute resources, creates A/CNAME records in Cloudflare
- **Validation:**
```bash
kubectl get pods -n external-dns
kubectl logs -n external-dns -l app.kubernetes.io/name=external-dns --tail=20
```
- **Expected:** Pod running, logs show `All records are already up to date` or records being created.

---

#### 5.10: ArgoCD

- [x] Add ArgoCD for GitOps application deployment (App-of-Apps pattern)

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# ArgoCD — GitOps Deployment Engine (App-of-Apps)
# =============================================================================

resource "helm_release" "argocd" {
  count = var.enable_argocd ? 1 : 0

  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = var.argocd_version
  namespace        = "argocd"
  create_namespace = true
  wait             = true
  timeout          = 600

  # Server
  set {
    name  = "server.replicas"
    value = "1"
  }

  set {
    name  = "server.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "server.resources.requests.memory"
    value = "256Mi"
  }

  # Repo Server
  set {
    name  = "repoServer.replicas"
    value = "1"
  }

  set {
    name  = "repoServer.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "repoServer.resources.requests.memory"
    value = "256Mi"
  }

  # Application Controller
  set {
    name  = "controller.resources.requests.cpu"
    value = "200m"
  }

  set {
    name  = "controller.resources.requests.memory"
    value = "512Mi"
  }

  # Image Updater (sidecar for Harbor image detection)
  set {
    name  = "server.extensions.enabled"
    value = "true"
  }

  set {
    name  = "configs.params.server\\.insecure"
    value = "true"
  }

  depends_on = [kubernetes_manifest.cluster_issuer]
}

# ArgoCD Image Updater
resource "helm_release" "argocd_image_updater" {
  count = var.enable_argocd ? 1 : 0

  name       = "argocd-image-updater"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argocd-image-updater"
  version    = var.argocd_image_updater_version
  namespace  = "argocd"
  wait       = true
  timeout    = 300

  set {
    name  = "config.registries[0].name"
    value = "harbor"
  }

  set {
    name  = "config.registries[0].prefix"
    value = "harbor.freezenith.com"
  }

  set {
    name  = "config.registries[0].api_url"
    value = "https://harbor.freezenith.com"
  }

  set {
    name  = "config.registries[0].default"
    value = "true"
  }

  depends_on = [helm_release.argocd]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "enable_argocd" {
  description = "Install ArgoCD GitOps engine"
  type        = bool
  default     = false
}

variable "argocd_version" {
  description = "ArgoCD Helm chart version"
  type        = string
  default     = "7.8.0"
}

variable "argocd_image_updater_version" {
  description = "ArgoCD Image Updater Helm chart version"
  type        = string
  default     = "0.11.0"
}
```

- **Depends on:** cert-manager
- **Validation:**
```bash
kubectl get pods -n argocd
kubectl get svc -n argocd argocd-server
# Get initial admin password:
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```
- **Expected:** 4+ pods running (server, repo-server, controller, redis, image-updater). Admin password retrieved.

---

#### 5.11: Harbor

- [x] Add Harbor container and Helm chart registry with S3 backend

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# Harbor — Container & Helm Chart Registry
# =============================================================================

resource "helm_release" "harbor" {
  count = var.enable_harbor ? 1 : 0

  name             = "harbor"
  repository       = "https://helm.goharbor.io"
  chart            = "harbor"
  version          = var.harbor_version
  namespace        = "harbor"
  create_namespace = true
  wait             = true
  timeout          = 600

  set {
    name  = "expose.type"
    value = "clusterIP"
  }

  set {
    name  = "expose.tls.enabled"
    value = "false"
  }

  set {
    name  = "externalURL"
    value = "https://registry.stage.freezenith.com"
  }

  # S3 storage backend
  set {
    name  = "persistence.imageChartStorage.type"
    value = "s3"
  }

  set {
    name  = "persistence.imageChartStorage.s3.region"
    value = "fsn1"
  }

  set {
    name  = "persistence.imageChartStorage.s3.bucket"
    value = "zenith-harbor"
  }

  set_sensitive {
    name  = "persistence.imageChartStorage.s3.accesskey"
    value = var.s3_access_key
  }

  set_sensitive {
    name  = "persistence.imageChartStorage.s3.secretkey"
    value = var.s3_secret_key
  }

  set {
    name  = "persistence.imageChartStorage.s3.regionendpoint"
    value = var.s3_endpoint
  }

  # Trivy vulnerability scanning
  set {
    name  = "trivy.enabled"
    value = "true"
  }

  # Admin password
  set_sensitive {
    name  = "harborAdminPassword"
    value = var.admin_password
  }

  # Resources (staging)
  set {
    name  = "core.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "core.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "priorityClassName"
    value = "platform"
  }

  depends_on = [
    kubernetes_manifest.cluster_issuer,
    kubernetes_priority_class.platform,
  ]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "enable_harbor" {
  description = "Install Harbor container registry"
  type        = bool
  default     = false
}

variable "harbor_version" {
  description = "Harbor Helm chart version"
  type        = string
  default     = "1.16.0"
}
```

- **Depends on:** cert-manager, S3 credentials, PriorityClasses
- **Validation:**
```bash
kubectl get pods -n harbor
curl -sk https://registry.stage.freezenith.com/api/v2.0/health
```
- **Expected:** Harbor core, registry, portal, database, redis, trivy pods running. Health endpoint returns `{"status":"healthy"}`.

---

#### 5.12: Temporal

- [x] Add Temporal workflow engine for customer provisioning

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# Temporal — Provisioning Workflow Engine
# =============================================================================

resource "helm_release" "temporal" {
  count = var.enable_temporal ? 1 : 0

  name             = "temporal"
  repository       = "https://go.temporal.io/helm-charts"
  chart            = "temporal"
  version          = var.temporal_version
  namespace        = "temporal"
  create_namespace = true
  wait             = true
  timeout          = 600

  # Use external PostgreSQL (CNPG free cluster)
  set {
    name  = "server.config.persistence.default.sql.driver"
    value = "postgres12"
  }

  set {
    name  = "server.config.persistence.default.sql.host"
    value = "free-pg-rw.zenith-shared.svc.cluster.local"
  }

  set {
    name  = "server.config.persistence.default.sql.port"
    value = "5432"
  }

  set {
    name  = "server.config.persistence.default.sql.database"
    value = "temporal"
  }

  set_sensitive {
    name  = "server.config.persistence.default.sql.user"
    value = var.temporal_db_user
  }

  set_sensitive {
    name  = "server.config.persistence.default.sql.password"
    value = var.temporal_db_password
  }

  # Visibility store (same PG, different DB)
  set {
    name  = "server.config.persistence.visibility.sql.driver"
    value = "postgres12"
  }

  set {
    name  = "server.config.persistence.visibility.sql.host"
    value = "free-pg-rw.zenith-shared.svc.cluster.local"
  }

  set {
    name  = "server.config.persistence.visibility.sql.port"
    value = "5432"
  }

  set {
    name  = "server.config.persistence.visibility.sql.database"
    value = "temporal_visibility"
  }

  set_sensitive {
    name  = "server.config.persistence.visibility.sql.user"
    value = var.temporal_db_user
  }

  set_sensitive {
    name  = "server.config.persistence.visibility.sql.password"
    value = var.temporal_db_password
  }

  # Disable built-in dependencies (using external PG)
  set {
    name  = "cassandra.enabled"
    value = "false"
  }

  set {
    name  = "mysql.enabled"
    value = "false"
  }

  set {
    name  = "postgresql.enabled"
    value = "false"
  }

  set {
    name  = "elasticsearch.enabled"
    value = "false"
  }

  # Temporal Web UI
  set {
    name  = "web.enabled"
    value = "true"
  }

  # Resources (staging)
  set {
    name  = "server.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "server.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "server.priorityClassName"
    value = "platform"
  }

  depends_on = [
    kubernetes_manifest.cnpg_free,
    kubernetes_priority_class.platform,
  ]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "enable_temporal" {
  description = "Install Temporal workflow engine"
  type        = bool
  default     = false
}

variable "temporal_version" {
  description = "Temporal Helm chart version"
  type        = string
  default     = "0.46.0"
}

variable "temporal_db_user" {
  description = "Temporal database user"
  type        = string
  sensitive   = true
  default     = "temporal"
}

variable "temporal_db_password" {
  description = "Temporal database password"
  type        = string
  sensitive   = true
  default     = ""
}
```

- **Depends on:** CNPG Free cluster (for database), PriorityClasses
- **Pre-requisite:** Create `temporal` and `temporal_visibility` databases in the free-pg CNPG cluster before deploying
- **Validation:**
```bash
kubectl get pods -n temporal
kubectl get svc -n temporal temporal-web
```
- **Expected:** Temporal server (frontend, history, matching, worker) + web UI pods running.

---

#### 5.13: Kyverno

- [x] Add Kyverno policy engine with 11 admission policies

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# Kyverno — Admission Policy Engine
# =============================================================================

resource "helm_release" "kyverno" {
  count = var.enable_kyverno ? 1 : 0

  name             = "kyverno"
  repository       = "https://kyverno.github.io/kyverno"
  chart            = "kyverno"
  version          = var.kyverno_version
  namespace        = "kyverno"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "replicaCount"
    value = "1"
  }

  set {
    name  = "resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "500m"
  }

  set {
    name  = "resources.limits.memory"
    value = "512Mi"
  }
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "enable_kyverno" {
  description = "Install Kyverno policy engine"
  type        = bool
  default     = false
}

variable "kyverno_version" {
  description = "Kyverno Helm chart version"
  type        = string
  default     = "3.3.4"
}
```

- **Depends on:** Nothing (can be installed in parallel with others)
- **Policies to add later (via K8s manifests):**
  1. `disallow-host-path` — Block hostPath volumes
  2. `disallow-host-namespaces` — Block hostNetwork/hostPID/hostIPC
  3. `disallow-privileged-containers` — Block privileged: true
  4. `disallow-capabilities` — Drop all except NET_BIND_SERVICE
  5. `require-run-as-non-root` — All containers non-root
  6. `require-labels` — Enforce customer, tier, component labels
  7. `require-resource-limits` — CPU + memory required
  8. `restrict-image-registries` — Only harbor.freezenith.com
  9. `verify-image-signature` — cosign verification
  10. `generate-default-deny-np` — Auto-generate default-deny NetworkPolicy
  11. `disallow-latest-tag` — Block :latest image tags
- **Validation:**
```bash
kubectl get pods -n kyverno
kubectl get clusterpolicies
```
- **Expected:** Kyverno controller pod running. ClusterPolicies listed after manifest application.

---

#### 5.14: Falco

- [x] Add Falco runtime security detection with eBPF driver

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# Falco — Runtime Security Detection (eBPF)
# =============================================================================

resource "helm_release" "falco" {
  count = var.enable_falco ? 1 : 0

  name             = "falco"
  repository       = "https://falcosecurity.github.io/charts"
  chart            = "falco"
  version          = var.falco_version
  namespace        = "falco"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "driver.kind"
    value = "ebpf"
  }

  set {
    name  = "falcosidekick.enabled"
    value = "true"
  }

  set {
    name  = "resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "500m"
  }

  set {
    name  = "resources.limits.memory"
    value = "512Mi"
  }
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "enable_falco" {
  description = "Install Falco runtime security"
  type        = bool
  default     = false
}

variable "falco_version" {
  description = "Falco Helm chart version"
  type        = string
  default     = "4.15.0"
}
```

- **Depends on:** Nothing (DaemonSet, runs on every node)
- **Detects:** Shell in container, unexpected outbound connections, crypto miners, privilege escalation, sensitive file reads
- **Validation:**
```bash
kubectl get pods -n falco
kubectl logs -n falco -l app.kubernetes.io/name=falco --tail=10
```
- **Expected:** Falco DaemonSet pod(s) running on every node. Logs show rules loaded.

---

#### 5.15: Velero

- [x] Add Velero for cluster backup to Hetzner S3

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# Velero — Cluster Backup to Hetzner S3
# =============================================================================

resource "helm_release" "velero" {
  count = var.enable_velero ? 1 : 0

  name             = "velero"
  repository       = "https://vmware-tanzu.github.io/helm-charts"
  chart            = "velero"
  version          = var.velero_version
  namespace        = "velero"
  create_namespace = true
  wait             = true
  timeout          = 300

  # AWS plugin for S3-compatible storage
  set {
    name  = "initContainers[0].name"
    value = "velero-plugin-for-aws"
  }

  set {
    name  = "initContainers[0].image"
    value = "velero/velero-plugin-for-aws:v1.11.0"
  }

  set {
    name  = "initContainers[0].volumeMounts[0].name"
    value = "plugins"
  }

  set {
    name  = "initContainers[0].volumeMounts[0].mountPath"
    value = "/target"
  }

  set {
    name  = "configuration.backupStorageLocation[0].name"
    value = "default"
  }

  set {
    name  = "configuration.backupStorageLocation[0].provider"
    value = "aws"
  }

  set {
    name  = "configuration.backupStorageLocation[0].bucket"
    value = "zenith-backups"
  }

  set {
    name  = "configuration.backupStorageLocation[0].prefix"
    value = "velero"
  }

  set {
    name  = "configuration.backupStorageLocation[0].config.region"
    value = "fsn1"
  }

  set {
    name  = "configuration.backupStorageLocation[0].config.s3ForcePathStyle"
    value = "true"
  }

  set {
    name  = "configuration.backupStorageLocation[0].config.s3Url"
    value = var.s3_endpoint
  }

  # Credentials
  set_sensitive {
    name  = "credentials.secretContents.cloud"
    value = "[default]\naws_access_key_id=${var.s3_access_key}\naws_secret_access_key=${var.s3_secret_key}\n"
  }

  # Schedule daily backup at 03:00 UTC
  set {
    name  = "schedules.daily-backup.disabled"
    value = "false"
  }

  set {
    name  = "schedules.daily-backup.schedule"
    value = "0 3 * * *"
  }

  set {
    name  = "schedules.daily-backup.template.ttl"
    value = "720h"
  }

  set {
    name  = "schedules.daily-backup.template.excludedNamespaces[0]"
    value = "velero"
  }

  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "enable_velero" {
  description = "Install Velero cluster backup"
  type        = bool
  default     = false
}

variable "velero_version" {
  description = "Velero Helm chart version"
  type        = string
  default     = "8.2.0"
}
```

- **Depends on:** S3 credentials
- **Backup schedule:** Daily at 03:00 UTC, 30-day retention, all namespaces except velero
- **Validation:**
```bash
kubectl get pods -n velero
velero backup-location get
velero schedule get
```
- **Expected:** Velero pod running, backup location `Available`, daily schedule listed.

---

#### 5.16: Prometheus + Grafana + Alertmanager

- [x] Upgrade existing monitoring stack with V2 enhancements

**File:** `infra/terraform/modules/k8s-platform/main.tf` — the existing `helm_release.monitoring` resource already installs kube-prometheus-stack. For V2, we need to upgrade the Helm chart to `kube-prometheus-stack` 68.4.0 from the official repo instead of the local chart.

Replace the existing monitoring block or add alongside it:
```hcl
# =============================================================================
# Monitoring — Prometheus + Grafana + Alertmanager (V2 upgrade)
# =============================================================================

resource "helm_release" "prometheus_stack" {
  count = var.enable_monitoring ? 1 : 0

  name             = "kube-prometheus-stack"
  repository       = "https://prometheus-community.github.io/helm-charts"
  chart            = "kube-prometheus-stack"
  version          = var.prometheus_stack_version
  namespace        = "monitoring"
  create_namespace = true
  wait             = true
  timeout          = 600

  # Grafana
  set_sensitive {
    name  = "grafana.adminPassword"
    value = var.admin_password
  }

  set {
    name  = "grafana.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "grafana.resources.requests.memory"
    value = "256Mi"
  }

  # Prometheus
  set {
    name  = "prometheus.prometheusSpec.retention"
    value = var.environment == "production" ? "90d" : "15d"
  }

  set {
    name  = "prometheus.prometheusSpec.resources.requests.cpu"
    value = "200m"
  }

  set {
    name  = "prometheus.prometheusSpec.resources.requests.memory"
    value = "512Mi"
  }

  set {
    name  = "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName"
    value = "hcloud-volumes"
  }

  set {
    name  = "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage"
    value = var.environment == "production" ? "50Gi" : "20Gi"
  }

  # Alertmanager
  set {
    name  = "alertmanager.alertmanagerSpec.resources.requests.cpu"
    value = "25m"
  }

  set {
    name  = "alertmanager.alertmanagerSpec.resources.requests.memory"
    value = "64Mi"
  }

  # ServiceMonitor for APISIX, Keycloak, CNPG, Temporal
  set {
    name  = "prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues"
    value = "false"
  }

  set {
    name  = "prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues"
    value = "false"
  }

  set {
    name  = "prometheus.prometheusSpec.priorityClassName"
    value = "platform"
  }

  depends_on = [kubernetes_priority_class.platform]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "prometheus_stack_version" {
  description = "kube-prometheus-stack Helm chart version"
  type        = string
  default     = "68.4.0"
}
```

- **Depends on:** PriorityClasses
- **Scrape targets:** kubelet, node-exporter, kube-state-metrics, APISIX (:9090), Temporal (:9090), Keycloak (:8080), CNPG (:9187), Cilium (:9090), Hubble (:9090)
- **Validation:**
```bash
kubectl get pods -n monitoring
kubectl get svc -n monitoring | grep grafana
```
- **Expected:** Prometheus, Grafana, Alertmanager, node-exporter, kube-state-metrics pods running.

---

#### 5.17: Loki

- [x] Upgrade Loki for centralized log aggregation

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# Loki — Log Aggregation
# =============================================================================

resource "helm_release" "loki" {
  count = var.enable_monitoring ? 1 : 0

  name       = "loki"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki"
  version    = var.loki_version
  namespace  = "monitoring"
  wait       = true
  timeout    = 600

  set {
    name  = "deploymentMode"
    value = "SingleBinary"
  }

  set {
    name  = "singleBinary.replicas"
    value = "1"
  }

  set {
    name  = "singleBinary.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "singleBinary.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "loki.storage.type"
    value = "filesystem"
  }

  set {
    name  = "singleBinary.persistence.enabled"
    value = "true"
  }

  set {
    name  = "singleBinary.persistence.storageClass"
    value = "hcloud-volumes"
  }

  set {
    name  = "singleBinary.persistence.size"
    value = "10Gi"
  }

  set {
    name  = "loki.auth_enabled"
    value = "false"
  }

  depends_on = [helm_release.prometheus_stack]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "loki_version" {
  description = "Loki Helm chart version"
  type        = string
  default     = "6.24.0"
}
```

- **Depends on:** Prometheus stack (for Grafana data source integration)
- **Log pipeline:** Kubelet → Promtail (DaemonSet, included in Loki chart) → Loki → Grafana
- **Validation:**
```bash
kubectl get pods -n monitoring -l app.kubernetes.io/name=loki
```
- **Expected:** Loki single-binary pod running.

---

#### 5.18: Tempo

- [x] Add Tempo for distributed trace storage

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# Tempo — Distributed Trace Storage
# =============================================================================

resource "helm_release" "tempo" {
  count = var.enable_monitoring ? 1 : 0

  name       = "tempo"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "tempo"
  version    = var.tempo_version
  namespace  = "monitoring"
  wait       = true
  timeout    = 300

  set {
    name  = "tempo.resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "tempo.resources.requests.memory"
    value = "128Mi"
  }

  set {
    name  = "persistence.enabled"
    value = "true"
  }

  set {
    name  = "persistence.storageClassName"
    value = "hcloud-volumes"
  }

  set {
    name  = "persistence.size"
    value = "10Gi"
  }

  depends_on = [helm_release.prometheus_stack]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "tempo_version" {
  description = "Tempo Helm chart version"
  type        = string
  default     = "1.15.0"
}
```

- **Depends on:** Prometheus stack
- **Trace flow:** APISIX (OTel plugin) → OTel Collector → Tempo → Grafana
- **Validation:**
```bash
kubectl get pods -n monitoring -l app.kubernetes.io/name=tempo
```
- **Expected:** Tempo pod running.

---

#### 5.19: OpenTelemetry Collector

- [x] Add OTel Collector as DaemonSet for trace and metric collection

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# OpenTelemetry Collector — Trace & Metric Pipeline
# =============================================================================

resource "helm_release" "otel_collector" {
  count = var.enable_monitoring ? 1 : 0

  name       = "otel-collector"
  repository = "https://open-telemetry.github.io/opentelemetry-helm-charts"
  chart      = "opentelemetry-collector"
  version    = var.otel_collector_version
  namespace  = "monitoring"
  wait       = true
  timeout    = 300

  set {
    name  = "mode"
    value = "daemonset"
  }

  set {
    name  = "config.exporters.otlp.endpoint"
    value = "tempo.monitoring.svc.cluster.local:4317"
  }

  set {
    name  = "config.exporters.otlp.tls.insecure"
    value = "true"
  }

  set {
    name  = "config.receivers.otlp.protocols.grpc.endpoint"
    value = "0.0.0.0:4317"
  }

  set {
    name  = "config.receivers.otlp.protocols.http.endpoint"
    value = "0.0.0.0:4318"
  }

  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }

  depends_on = [helm_release.tempo]
}
```

**File:** `infra/terraform/modules/k8s-platform/variables.tf` — add:
```hcl
variable "otel_collector_version" {
  description = "OpenTelemetry Collector Helm chart version"
  type        = string
  default     = "0.108.0"
}
```

- **Depends on:** Tempo (sends traces to it)
- **Pipeline:** APISIX/Go backend → OTel Collector (OTLP receiver) → Tempo (OTLP exporter)
- **Validation:**
```bash
kubectl get pods -n monitoring -l app.kubernetes.io/name=opentelemetry-collector
```
- **Expected:** OTel Collector DaemonSet pod(s) running on every node.

---

#### 5.20: Hubble UI

- [x] Verify Hubble UI is accessible (enabled via Cilium in Phase 2)

Hubble was enabled in Phase 2 via Cilium flags (`hubble.enabled=true`, `hubble.ui.enabled=true`). This step verifies it and creates an IngressRoute for external access.

**File:** Create a Traefik IngressRoute for Hubble UI (can be a kubernetes_manifest in Terraform or in the zenith-platform Helm chart):
```hcl
resource "kubernetes_manifest" "hubble_ingressroute" {
  manifest = {
    apiVersion = "traefik.io/v1alpha1"
    kind       = "IngressRoute"
    metadata = {
      name      = "hubble-ui"
      namespace = "kube-system"
      annotations = {
        "cert-manager.io/cluster-issuer" = "letsencrypt-prod"
      }
    }
    spec = {
      entryPoints = ["websecure"]
      routes = [{
        match = "Host(`hubble.stage.freezenith.com`)"
        kind  = "Rule"
        services = [{
          name = "hubble-ui"
          port = 80
        }]
      }]
      tls = {
        secretName = "hubble-tls"
      }
    }
  }
}
```

- **Depends on:** Cilium (Phase 2), cert-manager
- **Validation:**
```bash
kubectl get pods -n kube-system | grep hubble
cilium hubble ui
```
- **Expected:** hubble-relay and hubble-ui pods running. UI shows service map and network flows.

---

#### 5.21: ResourceQuotas + LimitRanges (Templates)

- [x] Create template ResourceQuota and LimitRange for customer namespaces

These are templates applied by the Temporal provisioning workflow when creating customer namespaces. Store them as ConfigMaps or apply via the zenith-tenant Helm chart.

**ResourceQuota by tier:**
```yaml
# Free tier
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tier-quota
  namespace: zenith-<customer>
spec:
  hard:
    requests.cpu: "2"
    requests.memory: 2Gi
    limits.cpu: "4"
    limits.memory: 4Gi
    pods: "10"
    persistentvolumeclaims: "2"
    requests.storage: 10Gi
---
# Pro tier
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tier-quota
  namespace: zenith-<customer>
spec:
  hard:
    requests.cpu: "8"
    requests.memory: 16Gi
    limits.cpu: "16"
    limits.memory: 32Gi
    pods: "50"
    persistentvolumeclaims: "10"
    requests.storage: 100Gi
```

**LimitRange (all tiers):**
```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
  namespace: zenith-<customer>
spec:
  limits:
  - default:
      cpu: 250m
      memory: 256Mi
    defaultRequest:
      cpu: 50m
      memory: 64Mi
    max:
      cpu: "2"
      memory: 2Gi
    min:
      cpu: 50m
      memory: 64Mi
    type: Container
```

- **Depends on:** Nothing (templates, applied per-customer by Temporal)
- **Validation:** Applied during customer provisioning (Phase 5)

---

#### 5.22: PodDisruptionBudgets

- [x] Create PDBs for all HA infrastructure services

**File:** `infra/terraform/modules/k8s-platform/main.tf` — add:
```hcl
# =============================================================================
# PodDisruptionBudgets — HA protection for infrastructure
# =============================================================================

locals {
  pdb_configs = {
    "keycloak" = {
      namespace      = "keycloak"
      match_labels   = { "app.kubernetes.io/name" = "keycloak" }
      min_available  = 1
    }
    "apisix" = {
      namespace      = "apisix"
      match_labels   = { "app.kubernetes.io/name" = "apisix" }
      min_available  = 1
    }
    "argocd-server" = {
      namespace      = "argocd"
      match_labels   = { "app.kubernetes.io/component" = "server" }
      min_available  = 1
    }
    "cnpg-operator" = {
      namespace      = "cnpg-system"
      match_labels   = { "app.kubernetes.io/name" = "cloudnative-pg" }
      min_available  = 1
    }
    "prometheus" = {
      namespace      = "monitoring"
      match_labels   = { "app.kubernetes.io/name" = "prometheus" }
      min_available  = 1
    }
    "grafana" = {
      namespace      = "monitoring"
      match_labels   = { "app.kubernetes.io/name" = "grafana" }
      min_available  = 1
    }
  }
}

resource "kubernetes_pod_disruption_budget" "infra" {
  for_each = local.pdb_configs

  metadata {
    name      = "${each.key}-pdb"
    namespace = each.value.namespace
  }

  spec {
    min_available = each.value.min_available

    selector {
      match_labels = each.value.match_labels
    }
  }
}
```

- **Depends on:** All infrastructure services deployed
- **Validation:**
```bash
kubectl get pdb --all-namespaces
```
- **Expected:** PDBs listed for keycloak, apisix, argocd-server, cnpg-operator, prometheus, grafana.

---

### Phase 3: Update staging-k8s Module Call

After adding all V2 components to the `k8s-platform` module, update the staging-k8s instantiation:

**File:** `infra/terraform/staging-k8s/main.tf` — update the module call:
```hcl
module "platform" {
  source = "../modules/k8s-platform"

  platform_namespace = "zenith-staging"
  cert_issuer_email  = "admin@freezenith.com"
  environment        = "staging"

  # Helm charts from Harbor OCI registry
  chart_repository = "oci://${var.registry_host}/zenith-stage"
  chart_version    = var.zenith_chart_version

  # Per-chart values files
  platform_values_file = "${path.module}/../../helm/zenith-platform/values-staging.yaml"
  api_values_file      = "${path.module}/../../helm/zenith-api/values-staging.yaml"
  landing_values_file  = "${path.module}/../../helm/zenith-landing/values-staging.yaml"
  demo_values_file     = "${path.module}/../../helm/zenith-demo/values-staging.yaml"
  tenant_values_file   = "${path.module}/../../helm/zenith-tenant/values-staging.yaml"

  # Registry credentials
  registry_host     = var.registry_host
  registry_username = var.registry_username
  registry_password = var.registry_password

  # App secrets
  jwt_secret     = var.jwt_secret
  admin_email    = var.admin_email
  admin_password = var.admin_password

  # --- V2 Feature Flags ---
  enable_cnpg            = true
  enable_apisix          = true     # Replaces Kong
  enable_keycloak        = true     # NEW
  enable_argocd          = true     # NEW
  enable_temporal        = true     # NEW
  enable_harbor          = true     # NEW
  enable_external_dns    = true     # NEW
  enable_sealed_secrets  = true     # NEW
  enable_kyverno         = true     # NEW
  enable_falco           = true     # NEW
  enable_velero          = true     # NEW
  enable_keda            = true
  enable_monitoring      = true
  enable_demo            = false
  enable_tenants         = true

  # --- V2 Credentials ---
  cloudflare_api_token     = var.cloudflare_api_token
  s3_access_key            = var.s3_access_key
  s3_secret_key            = var.s3_secret_key
  s3_endpoint              = "https://fsn1.your-objectstorage.com"
  keycloak_db_password     = var.keycloak_db_password
  keycloak_admin_password  = var.keycloak_admin_password
  temporal_db_user         = "temporal"
  temporal_db_password     = var.temporal_db_password

  # Monitoring
  monitoring_chart_path  = "${path.module}/../../helm/monitoring"
  monitoring_values_file = "${path.module}/../../helm/monitoring/values.yaml"
  monitoring_domain      = "stage.freezenith.com"
}
```

### Apply Phase 3

```bash
cd infra/terraform/staging-k8s
terraform init -upgrade
terraform plan
terraform apply
```

**Validation (full Phase 3):**
```bash
# All namespaces should exist
kubectl get ns

# All pods should be Running
kubectl get pods --all-namespaces | grep -v Running | grep -v Completed

# Quick health check per component
kubectl get pods -n cert-manager          # cert-manager controller
kubectl get pods -n cnpg-system           # CNPG operator
kubectl get pods -n keycloak              # Keycloak + PG
kubectl get pods -n apisix               # APISIX + etcd + ingress controller
kubectl get pods -n external-dns          # external-dns
kubectl get pods -n argocd               # ArgoCD server + repo + controller
kubectl get pods -n temporal             # Temporal frontend + history + matching + worker + web
kubectl get pods -n harbor               # Harbor core + registry + portal + ...
kubectl get pods -n kyverno              # Kyverno controller
kubectl get pods -n falco                # Falco DaemonSet
kubectl get pods -n sealed-secrets        # Sealed Secrets controller
kubectl get pods -n velero               # Velero server
kubectl get pods -n monitoring           # Prometheus + Grafana + Loki + Tempo + OTel
kubectl get pods -n zenith-shared         # CNPG free-pg + pro-shard-1 clusters
kubectl get priorityclasses              # 4 custom PriorityClasses
kubectl get pdb --all-namespaces         # PodDisruptionBudgets
```

---

## 6. Phase 4 — ArgoCD App-of-Apps

> **Reference:** `docs/v2-architecture/04-phase4-argocd-apps.md`
> **Directory:** `infra/argocd/`
> **Time:** ~15 minutes (setup), then continuous

### Overview

After Phase 3, ArgoCD is running but has no applications to manage. Phase 4 creates the App-of-Apps pattern: one root Application that watches a directory of Application manifests, enabling fully automated deployment from Git push.

### Directory Structure

```
infra/argocd/
├── staging/
│   ├── zenith-api.yaml          # Go API server Application
│   ├── zenith-landing.yaml      # Landing page Application
│   ├── zenith-platform.yaml     # Shared resources Application
│   └── tenants/
│       └── embermind.yaml       # Customer-specific Application
└── production/
    ├── zenith-api.yaml
    ├── zenith-landing.yaml
    ├── zenith-platform.yaml
    └── tenants/
        └── embermind.yaml
```

### Tasks

#### 6.1: Create Root Application (App-of-Apps)

- [x] Create the root ArgoCD Application that watches `infra/argocd/staging/`

This is created by Terraform in Phase 3 as part of the ArgoCD setup. Add to `k8s-platform/main.tf`:

```hcl
resource "kubernetes_manifest" "argocd_root_app" {
  count = var.enable_argocd ? 1 : 0

  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "zenith-apps"
      namespace = "argocd"
    }
    spec = {
      project = "default"
      source = {
        repoURL        = "https://github.com/DoTech/Zenith.git"
        targetRevision = "main"
        path           = "infra/argocd/${var.environment}"
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "argocd"
      }
      syncPolicy = {
        automated = {
          prune    = true
          selfHeal = true
        }
        syncOptions = [
          "CreateNamespace=true",
          "ServerSideApply=true",
        ]
        retry = {
          limit = 5
          backoff = {
            duration    = "5s"
            factor      = 2
            maxDuration = "3m"
          }
        }
      }
    }
  }

  depends_on = [helm_release.argocd]
}
```

**Validation:**
```bash
argocd app list
argocd app get zenith-apps
```
**Expected:** `zenith-apps` application is `Synced` and `Healthy`.

#### 6.2: Create Application Manifests

- [x] Create the individual Application manifests for each service

**File:** `infra/argocd/staging/zenith-api.yaml`
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: zenith-api
  namespace: argocd
  annotations:
    argocd-image-updater.argoproj.io/image-list: api=harbor.freezenith.com/zenith/zenith-api
    argocd-image-updater.argoproj.io/api.update-strategy: semver
    argocd-image-updater.argoproj.io/api.helm.image-name: image.repository
    argocd-image-updater.argoproj.io/api.helm.image-tag: image.tag
spec:
  project: default
  source:
    repoURL: https://github.com/DoTech/Zenith.git
    targetRevision: main
    path: infra/helm/zenith-api
    helm:
      valueFiles:
        - values-staging.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: zenith-platform
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

**File:** `infra/argocd/staging/zenith-landing.yaml`
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: zenith-landing
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/DoTech/Zenith.git
    targetRevision: main
    path: infra/helm/zenith-landing
    helm:
      valueFiles:
        - values-staging.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: zenith-platform
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

**File:** `infra/argocd/staging/zenith-platform.yaml`
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: zenith-platform
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/DoTech/Zenith.git
    targetRevision: main
    path: infra/helm/zenith-platform
    helm:
      valueFiles:
        - values-staging.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: zenith-platform
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
```

**File:** `infra/argocd/staging/tenants/embermind.yaml`
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: embermind
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/DoTech/Zenith.git
    targetRevision: main
    path: infra/helm/zenith-tenant
    helm:
      valueFiles:
        - values-staging.yaml
      parameters:
        - name: customer.slug
          value: embermind
        - name: customer.tier
          value: free
        - name: customer.domain
          value: embermind.stage.freezenith.com
  destination:
    server: https://kubernetes.default.svc
    namespace: zenith-embermind
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

#### 6.3: Sync Waves (Ordering)

- [x] Add sync wave annotations to Helm chart templates

In each Helm chart's templates, add sync wave annotations to control ordering:

```yaml
# Wave 0: Namespaces + RBAC
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "0"
---
# Wave 1: Secrets + ConfigMaps
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
---
# Wave 2: Deployments + Services
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "2"
---
# Wave 3: IngressRoutes + APISIX routes
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "3"
```

#### 6.4: Verify ArgoCD Deployment Pipeline

- [x] Push a change and verify automatic deployment

```bash
# Commit the argocd manifests
git add infra/argocd/staging/
git commit -m "feat: ArgoCD App-of-Apps manifests for staging"
git push

# Wait 30-60 seconds for ArgoCD to detect
argocd app list
argocd app get zenith-api
```

**Expected:** All applications show `Synced` and `Healthy` status.

**Developer workflow after setup:**
```
1. Push code to Git → 2. GitHub Actions CI (build, test, push to Harbor)
3. Image Updater detects new image (2-min poll) → 4. Updates Application
5. ArgoCD syncs (30s) → 6. Rolling update (30-60s)
Total: 4-8 minutes from push to live
```

---

## 7. Phase 5 — Temporal Provisioning Workflow

> **Reference:** `docs/v2-architecture/05-user-flows.md`
> **Directory:** `services/api/internal/temporal/`
> **Time:** Development time varies

### Overview

Temporal handles multi-step customer provisioning with automatic retry and recovery. When a user registers, the API starts a Temporal workflow that creates all required resources.

### Workflow Structure

```
services/api/internal/temporal/
├── workflows/
│   ├── provision_customer.go        # Main workflow definition
│   ├── upgrade_customer_tier.go     # Tier upgrade workflow
│   └── deprovision_customer.go      # Cleanup workflow
├── activities/
│   ├── create_keycloak_realm.go     # Activity 1: Keycloak realm
│   ├── create_database.go           # Activity 2: CNPG database
│   ├── create_s3_bucket.go          # Activity 3: Hetzner S3 bucket
│   ├── create_namespace.go          # Activity 4: K8s namespace + quota + limits
│   ├── create_secrets.go            # Activity 5: DB, S3, Keycloak creds
│   ├── create_dns_record.go         # Activity 6: Cloudflare via external-dns
│   ├── wait_for_tls_cert.go         # Activity 7: cert-manager certificate
│   ├── create_apisix_routes.go      # Activity 8: APISIX routes + plugins
│   ├── create_argocd_app.go         # Activity 9: ArgoCD Application manifest
│   └── notify_customer_ready.go     # Activity 10: Webhook/email notification
└── worker/
    └── main.go                      # Temporal worker registration
```

### Tasks

#### 7.1: Implement Provisioning Workflow

- [x] Create the main provisioning workflow that orchestrates all 10 activities

**File:** `services/api/internal/temporal/workflows/provision_customer.go`
```go
package workflows

import (
    "time"
    "go.temporal.io/sdk/workflow"
    "github.com/DoTech/Zenith/services/api/internal/temporal/activities"
)

type ProvisionCustomerInput struct {
    CustomerSlug string
    CustomerTier string // "free", "pro", "team", "enterprise"
    Email        string
    Domain       string // e.g., "embermind.stage.freezenith.com"
}

type ProvisionCustomerOutput struct {
    Namespace    string
    DatabaseHost string
    S3Bucket     string
    KeycloakRealm string
    TLSURL       string
}

func ProvisionCustomer(ctx workflow.Context, input ProvisionCustomerInput) (*ProvisionCustomerOutput, error) {
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: 2 * time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            InitialInterval:    5 * time.Second,
            BackoffCoefficient: 2.0,
            MaximumAttempts:    5,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, ao)

    // Activity 1: Create Keycloak realm (~5s)
    var realmResult activities.KeycloakRealmResult
    err := workflow.ExecuteActivity(ctx, activities.CreateKeycloakRealm, activities.KeycloakRealmInput{
        RealmName:    input.CustomerSlug,
        AdminEmail:   input.Email,
    }).Get(ctx, &realmResult)
    if err != nil {
        return nil, err
    }

    // Activity 2: Create database (~3s)
    var dbResult activities.DatabaseResult
    err = workflow.ExecuteActivity(ctx, activities.CreateDatabase, activities.DatabaseInput{
        CustomerSlug: input.CustomerSlug,
        Tier:         input.CustomerTier,
    }).Get(ctx, &dbResult)
    if err != nil {
        return nil, err
    }

    // Activity 3: Create S3 bucket (~5s)
    var s3Result activities.S3BucketResult
    err = workflow.ExecuteActivity(ctx, activities.CreateS3Bucket, activities.S3BucketInput{
        BucketName: "zenith-" + input.CustomerSlug + "-data",
    }).Get(ctx, &s3Result)
    if err != nil {
        return nil, err
    }

    // Activity 4: Create K8s namespace + ResourceQuota + LimitRange + NetworkPolicy (~2s)
    var nsResult activities.NamespaceResult
    err = workflow.ExecuteActivity(ctx, activities.CreateNamespace, activities.NamespaceInput{
        CustomerSlug: input.CustomerSlug,
        Tier:         input.CustomerTier,
    }).Get(ctx, &nsResult)
    if err != nil {
        return nil, err
    }

    // Activity 5: Create K8s secrets (DB, S3, Keycloak creds) (~1s)
    err = workflow.ExecuteActivity(ctx, activities.CreateSecrets, activities.SecretsInput{
        Namespace:     "zenith-" + input.CustomerSlug,
        DBCredentials: dbResult,
        S3Credentials: s3Result,
        KeycloakCreds: realmResult,
    }).Get(ctx, nil)
    if err != nil {
        return nil, err
    }

    // Activity 6: Create APISIX routes + plugins (~2s)
    err = workflow.ExecuteActivity(ctx, activities.CreateApisixRoutes, activities.ApisixRoutesInput{
        CustomerSlug: input.CustomerSlug,
        Domain:       input.Domain,
        Tier:         input.CustomerTier,
    }).Get(ctx, nil)
    if err != nil {
        return nil, err
    }

    // Activity 7: Wait for DNS propagation (~30s)
    err = workflow.ExecuteActivity(ctx, activities.CreateDNSRecord, activities.DNSInput{
        Domain: input.Domain,
    }).Get(ctx, nil)
    if err != nil {
        return nil, err
    }

    // Activity 8: Wait for TLS certificate (~30s)
    err = workflow.ExecuteActivity(ctx, activities.WaitForTLSCert, activities.TLSInput{
        Domain:    input.Domain,
        Namespace: "zenith-" + input.CustomerSlug,
    }).Get(ctx, nil)
    if err != nil {
        return nil, err
    }

    // Activity 9: Create ArgoCD Application (~2s)
    err = workflow.ExecuteActivity(ctx, activities.CreateArgoCDApp, activities.ArgoCDInput{
        CustomerSlug: input.CustomerSlug,
        Tier:         input.CustomerTier,
        Domain:       input.Domain,
    }).Get(ctx, nil)
    if err != nil {
        return nil, err
    }

    // Activity 10: Notify customer ready (~1s)
    err = workflow.ExecuteActivity(ctx, activities.NotifyCustomerReady, activities.NotifyInput{
        Email:  input.Email,
        Domain: input.Domain,
    }).Get(ctx, nil)
    if err != nil {
        return nil, err
    }

    return &ProvisionCustomerOutput{
        Namespace:     "zenith-" + input.CustomerSlug,
        DatabaseHost:  dbResult.Host,
        S3Bucket:      s3Result.BucketName,
        KeycloakRealm: realmResult.RealmName,
        TLSURL:        "https://" + input.Domain,
    }, nil
}
```

**Total provisioning time:** 60-90 seconds for Free/Pro tier.

#### 7.2: Register Temporal Worker

- [x] Create worker that registers all activities

**File:** `services/api/internal/temporal/worker/main.go`
```go
package worker

import (
    "go.temporal.io/sdk/client"
    "go.temporal.io/sdk/worker"
    "github.com/DoTech/Zenith/services/api/internal/temporal/activities"
    "github.com/DoTech/Zenith/services/api/internal/temporal/workflows"
)

const TaskQueue = "zenith-provisioning"

func StartWorker(c client.Client) error {
    w := worker.New(c, TaskQueue, worker.Options{})

    // Register workflows
    w.RegisterWorkflow(workflows.ProvisionCustomer)
    w.RegisterWorkflow(workflows.UpgradeCustomerTier)
    w.RegisterWorkflow(workflows.DeprovisionCustomer)

    // Register activities
    w.RegisterActivity(activities.CreateKeycloakRealm)
    w.RegisterActivity(activities.CreateDatabase)
    w.RegisterActivity(activities.CreateS3Bucket)
    w.RegisterActivity(activities.CreateNamespace)
    w.RegisterActivity(activities.CreateSecrets)
    w.RegisterActivity(activities.CreateApisixRoutes)
    w.RegisterActivity(activities.CreateDNSRecord)
    w.RegisterActivity(activities.WaitForTLSCert)
    w.RegisterActivity(activities.CreateArgoCDApp)
    w.RegisterActivity(activities.NotifyCustomerReady)

    return w.Run(worker.InterruptCh())
}
```

#### 7.3: Test Provisioning

- [x] Trigger a test provisioning workflow and verify all resources

```bash
# Via Temporal CLI
temporal workflow start \
  --task-queue zenith-provisioning \
  --type ProvisionCustomer \
  --input '{"CustomerSlug":"test-customer","CustomerTier":"free","Email":"test@example.com","Domain":"test-customer.stage.freezenith.com"}'

# Or via zenith-api endpoint
curl -X POST https://api.stage.freezenith.com/v1/admin/customers \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"slug":"test-customer","tier":"free","email":"test@example.com"}'
```

**Validation checklist:**
```bash
# Keycloak realm created
curl -s https://auth.stage.freezenith.com/realms/test-customer | jq .realm

# Database created
kubectl exec -n zenith-shared free-pg-1 -- psql -U zenith_admin -l | grep test_customer

# S3 bucket created (via Hetzner API or mc client)
mc ls hetzner/zenith-test-customer-data

# Namespace + resources
kubectl get ns zenith-test-customer
kubectl get resourcequota -n zenith-test-customer
kubectl get limitrange -n zenith-test-customer
kubectl get ciliumnetworkpolicy -n zenith-test-customer

# Secrets
kubectl get secrets -n zenith-test-customer

# APISIX route
kubectl get apisixroutes -n zenith-test-customer

# TLS certificate
kubectl get certificates -n zenith-test-customer

# ArgoCD application
argocd app get test-customer

# DNS resolves
dig +short test-customer.stage.freezenith.com
```

**Expected:** All resources exist, DNS resolves, TLS valid, ArgoCD app synced.

---

## 8. Migration Phases M1-M6

> **Reference:** `docs/v2-architecture/09-migration-v1-to-v2.md`
> **Strategy:** Blue-Green — build V2 alongside V1, switch when ready

### Migration Strategy

**We do NOT migrate V1 in-place.** Instead:
1. Build V2 on a NEW Hetzner server (staging-v2)
2. Test everything on staging-v2
3. When ready, point DNS from production to staging-v2
4. Decommission old servers

### Phase M1: New Staging Server (Week 1-2)

- [x] Create new Hetzner server (cx42) for V2
- [x] Run Phase 1 Terraform (server + DNS for v2.stage.freezenith.com)
- [x] Run Phase 2 Ansible (k3s + Cilium + hcloud-csi + WireGuard)
- [x] Run Phase 3 Terraform (all V2 infra components)

**Validation checklist:**
- [x] k3s running with Cilium CNI + WireGuard
- [x] cert-manager issuing test certificate (DNS-01)
- [x] CNPG operator watching namespaces
- [x] Keycloak accessible, admin realm working
- [x] APISIX routing test requests
- [x] external-dns creating DNS records
- [x] ArgoCD UI accessible at argocd.stage.freezenith.com
- [x] Temporal UI accessible at temporal.stage.freezenith.com
- [x] Harbor accessible at registry.stage.freezenith.com
- [x] Monitoring dashboards loading (Grafana, Prometheus)
- [x] Hubble UI showing network flows

### Phase M2: Deploy Zenith Apps (Week 2-3)

- [x] Push Helm charts to Harbor
- [x] ArgoCD syncs application charts automatically
- [x] Test: landing page accessible
- [x] Test: API health endpoint responding
- [ ] Test: Keycloak login flow working
- [ ] Test: APISIX JWT verification working

**Validation checklist:**
- [x] Landing page loads at v2.stage.freezenith.com
- [x] API responds at api.v2.stage.freezenith.com
- [ ] Keycloak login/register works
- [ ] APISIX correctly routes protected/public routes
- [x] ArgoCD shows all apps synced and healthy

### Phase M3: Customer Provisioning (Week 3-4)

- [x] Trigger provision-customer workflow via API
- [ ] Verify: Keycloak realm created
- [x] Verify: Database created in shared CNPG cluster
- [ ] Verify: S3 bucket created
- [x] Verify: K8s namespace with all resources
- [x] Verify: DNS record created
- [x] Verify: TLS certificate issued
- [ ] Verify: Customer frontend accessible
- [ ] Verify: Customer backend API through APISIX with JWT

**Validation checklist:**
- [x] Full provisioning workflow completes without errors
- [ ] Customer can log in via Keycloak
- [ ] Customer frontend loads
- [ ] Customer API calls work through APISIX
- [ ] Network isolation: customer A cannot reach customer B namespace
- [x] ResourceQuota enforced
- [ ] Backup CronJobs running

### Phase M4: Data Migration (Week 4-5)

- [ ] pg_dump from V1 production PostgreSQL
- [ ] Create customer DB in V2 CNPG cluster
- [ ] pg_restore into V2
- [ ] Migrate or recreate Keycloak realm and users
- [ ] Migrate S3 data (if any)
- [ ] Verify: customer can access their data on V2

**For embermind.app (current customer):**
```bash
# 1. Dump from V1 production
ssh root@161.35.82.211 "kubectl exec -n zenith-platform cnpg-cluster-1 -- \
  pg_dump -U postgres embermind_db" > embermind_v1.sql

# 2. Create database on V2
kubectl exec -n zenith-shared free-pg-1 -- \
  psql -U zenith_admin -c "CREATE DATABASE embermind_db;"
kubectl exec -n zenith-shared free-pg-1 -- \
  psql -U zenith_admin -c "CREATE USER embermind WITH PASSWORD 'GENERATED_PASSWORD';"
kubectl exec -n zenith-shared free-pg-1 -- \
  psql -U zenith_admin -c "GRANT ALL ON DATABASE embermind_db TO embermind;"

# 3. Restore
cat embermind_v1.sql | kubectl exec -i -n zenith-shared free-pg-1 -- \
  psql -U zenith_admin embermind_db

# 4. Verify
kubectl exec -n zenith-shared free-pg-1 -- \
  psql -U zenith_admin embermind_db -c "\dt"
```

**Rollback:** V1 is still running, no data was modified on V1.

### Phase M5: DNS Cutover (Week 5-6)

- [ ] Update Cloudflare DNS: freezenith.com → V2 server IP
- [ ] Update Cloudflare DNS: api.freezenith.com → V2 server IP
- [ ] Update Cloudflare DNS: ms.embermind.app → V2 server IP
- [ ] Update Cloudflare DNS: cloud.embermind.app → V2 server IP
- [ ] Enable Cloudflare proxy (WAF + DDoS protection)
- [ ] Monitor: check all endpoints responding
- [ ] Keep V1 server running for 2 weeks (rollback safety)

**Rollback plan:**
```bash
# If V2 has issues, point DNS back to V1 (< 5 minute recovery):
# In Cloudflare Dashboard or via API:
# freezenith.com → 161.35.82.211 (V1 production IP)
# V1 data may be slightly behind since cutover moment
```

### Phase M6: Cleanup (Week 7)

- [ ] Verify V2 running smoothly for 2+ weeks
- [ ] Take final backup of V1 server
- [ ] Delete V1 DigitalOcean server (161.35.82.211)
- [ ] Delete V1 Hetzner staging server (77.42.88.149) if separate from V2
- [ ] Clean up old DNS records
- [ ] Archive V1 Terraform state
- [ ] Update all documentation to reflect V2

### Risk Matrix

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| V2 infra component fails | High | Medium | Keep V1 running, DNS rollback in 5min |
| Data loss during migration | Critical | Low | pg_dump before and after, verify data |
| Customer downtime during cutover | Medium | Medium | Low-TTL DNS, cutover during low traffic |
| Keycloak realm misconfiguration | High | Medium | Test thoroughly on staging first |
| APISIX routing misconfiguration | High | Medium | Compare V1 Kong routes with V2 APISIX |
| Cilium NetworkPolicy blocks traffic | Medium | Medium | Start with audit mode, then enforce |
| cert-manager DNS-01 fails | Medium | Low | Test with staging cert first |

---

## 9. End-to-End Verification

### Full Smoke Test

Run this after all phases are complete:

```bash
#!/bin/bash
set -e
echo "=== Zenith V2 End-to-End Smoke Test ==="

# 1. Cluster health
echo "--- Cluster Health ---"
kubectl get nodes
kubectl get pods --all-namespaces | grep -v Running | grep -v Completed | grep -v NAME && \
  echo "WARN: Some pods not running!" || echo "OK: All pods running"

# 2. Infrastructure components
echo "--- Infrastructure Components ---"
for ns in cert-manager cnpg-system keycloak apisix external-dns argocd \
           temporal harbor kyverno falco sealed-secrets velero monitoring zenith-shared; do
  count=$(kubectl get pods -n $ns --no-headers 2>/dev/null | grep Running | wc -l)
  echo "$ns: $count running pods"
done

# 3. CNPG clusters
echo "--- CNPG Clusters ---"
kubectl get clusters.postgresql.cnpg.io --all-namespaces

# 4. Certificates
echo "--- TLS Certificates ---"
kubectl get certificates --all-namespaces

# 5. ArgoCD applications
echo "--- ArgoCD Applications ---"
argocd app list 2>/dev/null || kubectl get applications -n argocd

# 6. DNS resolution
echo "--- DNS Resolution ---"
for sub in stage api.stage auth.stage argocd.stage temporal.stage \
           registry.stage grafana.stage hubble.stage; do
  ip=$(dig +short $sub.freezenith.com 2>/dev/null)
  echo "$sub.freezenith.com → ${ip:-FAILED}"
done

# 7. Endpoints
echo "--- Endpoint Checks ---"
for url in "https://stage.freezenith.com" \
           "https://api.stage.freezenith.com/health" \
           "https://auth.stage.freezenith.com" \
           "https://argocd.stage.freezenith.com" \
           "https://grafana.stage.freezenith.com"; do
  status=$(curl -sk -o /dev/null -w "%{http_code}" "$url" 2>/dev/null)
  echo "$url → HTTP $status"
done

# 8. Velero backup status
echo "--- Backup Status ---"
velero backup-location get 2>/dev/null || echo "Velero CLI not configured"

# 9. Priority classes
echo "--- Priority Classes ---"
kubectl get priorityclasses | grep -E "system-critical|infra-critical|platform|customer"

# 10. PDBs
echo "--- Pod Disruption Budgets ---"
kubectl get pdb --all-namespaces --no-headers | wc -l
echo "PDBs configured"

echo "=== Smoke Test Complete ==="
```

### Security Validation

```bash
# 1. NetworkPolicy — verify default deny in customer namespaces
kubectl get ciliumnetworkpolicy -n zenith-embermind

# 2. Pod Security Standards — verify restricted PSS
kubectl get ns zenith-embermind -o jsonpath='{.metadata.labels}' | jq .

# 3. Kyverno policies — list active policies
kubectl get clusterpolicies

# 4. Falco — check for alerts
kubectl logs -n falco -l app.kubernetes.io/name=falco --tail=20 | grep -i alert

# 5. Secrets encryption — verify etcd encryption
ssh root@<SERVER_IP> "k3s secrets-encrypt status"

# 6. WireGuard — verify encryption
cilium encrypt status

# 7. Hubble — verify network flow visibility
cilium hubble port-forward &
hubble observe -n zenith-embermind --last 20
```

### Observability Validation

```bash
# 1. Metrics — Prometheus scrape targets
kubectl port-forward -n monitoring svc/kube-prometheus-stack-prometheus 9090 &
curl -s localhost:9090/api/v1/targets | jq '.data.activeTargets | length'

# 2. Logs — Loki ingestion
kubectl port-forward -n monitoring svc/loki 3100 &
curl -s localhost:3100/ready

# 3. Traces — Tempo health
kubectl port-forward -n monitoring svc/tempo 3200 &
curl -s localhost:3200/ready

# 4. Dashboards — Grafana
# Access at https://grafana.stage.freezenith.com
# Verify dashboards: Cluster Overview, CNPG Health, APISIX Traffic, Cilium/Hubble

# 5. Alertmanager
kubectl port-forward -n monitoring svc/kube-prometheus-stack-alertmanager 9093 &
curl -s localhost:9093/api/v2/alerts | jq '. | length'
```

### Backup Validation

```bash
# 1. Velero — trigger manual backup and verify
velero backup create test-backup --wait
velero backup describe test-backup
velero backup delete test-backup

# 2. CNPG WAL — verify archiving
kubectl get clusters.postgresql.cnpg.io -n zenith-shared free-pg -o jsonpath='{.status.conditions}' | jq .

# 3. pg_dump — manual test
kubectl exec -n zenith-shared free-pg-1 -- \
  pg_dump -U zenith_admin zenith_platform | gzip > /tmp/test-backup.sql.gz
ls -la /tmp/test-backup.sql.gz  # Should be > 100 bytes

# 4. Keycloak — verify realm export capability
kubectl exec -n keycloak deploy/keycloak -- \
  /opt/bitnami/keycloak/bin/kc.sh export --realm master --dir /tmp/export
```

---

## Quick Reference: Component Versions

| Component | Helm Chart | Version | Namespace |
|-----------|------------|---------|-----------|
| cert-manager | jetstack/cert-manager | v1.17.2 | cert-manager |
| CNPG Operator | cnpg/cloudnative-pg | 0.23.0 | cnpg-system |
| Keycloak | bitnami/keycloak | 24.4.0 | keycloak |
| APISIX | apisix/apisix | 2.10.0 | apisix |
| APISIX Ingress | apisix/apisix-ingress-controller | 0.14.0 | apisix |
| external-dns | bitnami/external-dns | 8.7.0 | external-dns |
| ArgoCD | argoproj/argo-cd | 7.8.0 | argocd |
| ArgoCD Image Updater | argoproj/argocd-image-updater | 0.11.0 | argocd |
| Temporal | temporalio/temporal | 0.46.0 | temporal |
| Harbor | harbor/harbor | 1.16.0 | harbor |
| Kyverno | kyverno/kyverno | 3.3.4 | kyverno |
| Falco | falcosecurity/falco | 4.15.0 | falco |
| Sealed Secrets | bitnami-labs/sealed-secrets | 2.17.0 | sealed-secrets |
| Velero | vmware-tanzu/velero | 8.2.0 | velero |
| Prometheus Stack | prometheus-community/kube-prometheus-stack | 68.4.0 | monitoring |
| Loki | grafana/loki | 6.24.0 | monitoring |
| Tempo | grafana/tempo | 1.15.0 | monitoring |
| OTel Collector | open-telemetry/opentelemetry-collector | 0.108.0 | monitoring |

## Quick Reference: Helm Repositories

```bash
helm repo add jetstack https://charts.jetstack.io
helm repo add cnpg https://cloudnative-pg.github.io/charts
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add apisix https://charts.apiseven.com
helm repo add argo https://argoproj.github.io/argo-helm
helm repo add temporal https://go.temporal.io/helm-charts
helm repo add harbor https://helm.goharbor.io
helm repo add kyverno https://kyverno.github.io/kyverno
helm repo add falco https://falcosecurity.github.io/charts
helm repo add sealed-secrets https://bitnami-labs.github.io/sealed-secrets
helm repo add vmware-tanzu https://vmware-tanzu.github.io/helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
```

## Quick Reference: Key Architecture Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | APISIX over Kong | etcd-backed, richer plugins, no Enterprise licensing cost |
| D2 | CNPG sharding (Free shared, Pro ~20/cluster) | Cost efficiency + blast radius isolation |
| D3 | Keycloak dedicated CNPG | Critical path, must not be affected by app DB load |
| D4 | ArgoCD over FluxCD | Built-in UI, App-of-Apps, certification prep |
| D5 | Temporal for provisioning | Durable workflows, activity-level retries |
| D8 | Frontend bypasses API gateway | Traefik direct for frontends, APISIX for backends only |
| D9 | Single APISIX, route-level plugins | No need for multiple gateway deployments |
| D10 | Terraform for infra, ArgoCD for apps | Clear separation: slow/reviewed vs fast/continuous |
| D11 | Defense-in-depth (6 layers) | Cilium → APISIX → PSS → Kyverno → Falco → Harbor |
| D12 | 3-layer backup | WAL (seconds RPO) → pg_dump (hours) → Velero (days) |
