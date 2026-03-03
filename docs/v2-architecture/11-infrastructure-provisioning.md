# 11 — Infrastructure Provisioning (Terraform + Ansible)

> **Purpose:** Understand how the entire Zenith platform is provisioned from bare metal to running services.
> **Audience:** Any developer who needs to set up, maintain, or debug the provisioning pipeline.
> **Last Updated:** 2026-03-03
> **Related:** [01-phase1-hetzner-cloudflare.md](./01-phase1-hetzner-cloudflare.md), [02-phase2-ansible-k3s.md](./02-phase2-ansible-k3s.md), [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) (detailed phase docs), [SYSTEM-MAP.md](./SYSTEM-MAP.md) (full system overview)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose This Stack](#2-why-we-chose-this-stack)
3. [Architecture Diagram](#3-architecture-diagram)
4. [The 3-Layer Provisioning Model](#4-the-3-layer-provisioning-model)
5. [Layer 1: Terraform — Cloud Resources](#5-layer-1-terraform--cloud-resources)
6. [Layer 2: Ansible — OS & k3s](#6-layer-2-ansible--os--k3s)
7. [Layer 3: Terraform — Cluster Bootstrap](#7-layer-3-terraform--cluster-bootstrap)
8. [Configuration Reference](#8-configuration-reference)
9. [Request Flow: Full Environment Setup](#9-request-flow-full-environment-setup)
10. [Troubleshooting](#10-troubleshooting)
11. [Upgrade Path](#11-upgrade-path)

---

## 1. Overview

Zenith uses a **3-layer provisioning model** to go from "nothing" to "fully running platform":

1. **Terraform (Hetzner + Cloudflare)** — Creates the VM, firewall, SSH keys, DNS records
2. **Ansible (OS + k3s)** — Installs packages, k3s, Cilium CNI, and prerequisite secrets
3. **Terraform (k8s-platform)** — Installs all 22+ Helm releases into the running cluster

After these 3 layers, **ArgoCD** takes over and manages all application deployments via GitOps.

---

## 2. Why We Chose This Stack

| Tool | Alternative Considered | Why We Chose It |
|------|----------------------|----------------|
| **Terraform** | Pulumi, OpenTofu | Industry standard IaC, excellent Hetzner + Cloudflare providers, HCL is readable |
| **Ansible** | cloud-init, Packer | Idempotent, agentless (SSH only), can re-run safely, good for OS-level config |
| **k3s** | k0s, kubeadm, RKE2 | Lightweight, single-binary, built-in Traefik + CSI, perfect for Hetzner VMs |
| **Hetzner** | AWS, GCP, DigitalOcean | 80% cheaper, EU data residency, simple API, no egress fees |

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    PROVISIONING PIPELINE                                         │
│                    (from nothing to running platform)                            │
│                                                                                 │
│  DEVELOPER LAPTOP                                                               │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                                                                           │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐     │  │
│  │  │  LAYER 1: terraform apply  (infra/terraform/staging/)           │     │  │
│  │  │                                                                  │     │  │
│  │  │  Creates:                                                        │     │  │
│  │  │    ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐   │     │  │
│  │  │    │ Hetzner VM   │  │ Hetzner      │  │ Cloudflare DNS     │   │     │  │
│  │  │    │ Ubuntu 25.04 │  │ Firewall     │  │ *.stage.freezenith │   │     │  │
│  │  │    │ CPX41 (stg)  │  │ SSH + HTTP/S │  │ .com → VM IP       │   │     │  │
│  │  │    └──────┬───────┘  └──────────────┘  └────────────────────┘   │     │  │
│  │  └───────────┼──────────────────────────────────────────────────────┘     │  │
│  │              │ SSH (port 22)                                               │  │
│  │              ▼                                                             │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐     │  │
│  │  │  LAYER 2: ansible-playbook  (infra/ansible/)                    │     │  │
│  │  │                                                                  │     │  │
│  │  │  On the VM, installs:                                            │     │  │
│  │  │    ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │     │  │
│  │  │    │ apt pkgs │  │ Docker   │  │ k3s      │  │ Cilium CLI   │  │     │  │
│  │  │    │ curl,git │  │ engine   │  │ server   │  │ + Cilium CNI │  │     │  │
│  │  │    │ jq,htop  │  │          │  │ (no      │  │ + WireGuard  │  │     │  │
│  │  │    │ python3  │  │          │  │  flannel) │  │ + Hubble     │  │     │  │
│  │  │    └──────────┘  └──────────┘  └──────────┘  └──────────────┘  │     │  │
│  │  │                                                                  │     │  │
│  │  │  Also:                                                           │     │  │
│  │  │    - Disables swap, sets sysctl for k8s networking               │     │  │
│  │  │    - Fetches kubeconfig → saves to ~/.kube/zenith-staging.yaml   │     │  │
│  │  │    - Creates cloudflare-api-token Secret for cert-manager DNS-01 │     │  │
│  │  └──────────────────────────────────────────────────────────────────┘     │  │
│  │              │ Now cluster is running with kubectl access                  │  │
│  │              ▼                                                             │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐     │  │
│  │  │  LAYER 3: terraform apply  (infra/terraform/staging-k8s/)       │     │  │
│  │  │                                                                  │     │  │
│  │  │  Installs 22+ components via Helm + K8s manifests:               │     │  │
│  │  │                                                                  │     │  │
│  │  │  ┌─────────────┐ ┌────────────┐ ┌───────────┐ ┌─────────────┐  │     │  │
│  │  │  │cert-manager │ │CNPG + PG   │ │Keycloak   │ │APISIX+etcd  │  │     │  │
│  │  │  │+ClusterIssue│ │clusters    │ │           │ │+ingress ctrl│  │     │  │
│  │  │  └─────────────┘ └────────────┘ └───────────┘ └─────────────┘  │     │  │
│  │  │  ┌─────────────┐ ┌────────────┐ ┌───────────┐ ┌─────────────┐  │     │  │
│  │  │  │external-dns │ │ArgoCD+     │ │Harbor     │ │Temporal     │  │     │  │
│  │  │  │(Cloudflare) │ │ImageUpdater│ │(customer) │ │             │  │     │  │
│  │  │  └─────────────┘ └────────────┘ └───────────┘ └─────────────┘  │     │  │
│  │  │  ┌─────────────┐ ┌────────────┐ ┌───────────┐ ┌─────────────┐  │     │  │
│  │  │  │Kyverno      │ │Falco       │ │Velero     │ │Sealed       │  │     │  │
│  │  │  │             │ │            │ │           │ │Secrets      │  │     │  │
│  │  │  └─────────────┘ └────────────┘ └───────────┘ └─────────────┘  │     │  │
│  │  │  ┌─────────────┐ ┌────────────┐ ┌───────────┐ ┌─────────────┐  │     │  │
│  │  │  │Prometheus   │ │Loki        │ │Tempo      │ │OTel         │  │     │  │
│  │  │  │+Grafana     │ │            │ │           │ │Collector    │  │     │  │
│  │  │  └─────────────┘ └────────────┘ └───────────┘ └─────────────┘  │     │  │
│  │  │  ┌─────────────┐ ┌────────────┐ ┌───────────┐                  │     │  │
│  │  │  │KEDA+HTTP    │ │Hetzner CSI │ │Traefik    │                  │     │  │
│  │  │  │add-on       │ │            │ │config     │                  │     │  │
│  │  │  └─────────────┘ └────────────┘ └───────────┘                  │     │  │
│  │  │                                                                  │     │  │
│  │  │  + Application Helm charts:                                      │     │  │
│  │  │    zenith-platform, zenith-api, zenith-landing, zenith-demo      │     │  │
│  │  └──────────────────────────────────────────────────────────────────┘     │  │
│  │              │ ArgoCD is now running                                       │  │
│  │              ▼                                                             │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐     │  │
│  │  │  LAYER 4: ArgoCD  (automatic — no manual steps)                 │     │  │
│  │  │                                                                  │     │  │
│  │  │  Watches: github.com/taikuri-infra/Zenith.git (staging branch)  │     │  │
│  │  │  Path: infra/argocd/staging/                                     │     │  │
│  │  │  App-of-Apps: zenith-apps (root Application)                     │     │  │
│  │  │                                                                  │     │  │
│  │  │  Auto-syncs: zenith-platform → zenith-operator → zenith-api     │     │  │
│  │  │              → zenith-landing → zenith-web → zenith-demo         │     │  │
│  │  │                                                                  │     │  │
│  │  │  From here: git push → ArgoCD detects → auto-deploys            │     │  │
│  │  └──────────────────────────────────────────────────────────────────┘     │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. The 3-Layer Provisioning Model

### Why 3 layers instead of 1?

Each layer has different **requirements** and **state**:

```
Layer 1 (Terraform):  Talks to Hetzner API + Cloudflare API
                      State: terraform.tfstate (cloud resources)
                      Needs: Hetzner token, Cloudflare token

Layer 2 (Ansible):    Talks to VM via SSH
                      State: none (idempotent)
                      Needs: SSH key, VM IP from Layer 1

Layer 3 (Terraform):  Talks to K8s API (kubectl/Helm)
                      State: terraform.tfstate (k8s resources)
                      Needs: kubeconfig from Layer 2
```

You **cannot** combine them because:
- Layer 1 creates the VM that Layer 2 configures
- Layer 2 creates the K8s cluster that Layer 3 deploys into
- Each layer needs the **output** of the previous layer

---

## 5. Layer 1: Terraform — Cloud Resources

**Directory:** `infra/terraform/staging/`

### What it creates:

```
Hetzner Cloud                          Cloudflare
┌────────────────────────────────┐     ┌────────────────────────────────┐
│ Server: zenith-staging         │     │ Zone: freezenith.com           │
│   Type: CPX41 (or configurable)│     │                                │
│   OS: Ubuntu 25.04             │     │ A Records:                     │
│   Location: configurable       │     │   stage      → <VM IP>         │
│   SSH key: from var             │     │   api.stage  → <VM IP>         │
│                                │     │   app.stage  → <VM IP>         │
│ Firewall:                      │     │   argocd.stage → <VM IP>       │
│   Allow: 22 (SSH)              │     │   auth.stage → <VM IP>         │
│   Allow: 80 (HTTP)             │     │   grafana.stage → <VM IP>      │
│   Allow: 443 (HTTPS)           │     │   hub.stage  → <VM IP>         │
│   Allow: 6443 (K8s API)        │     │   hubble.stage → <VM IP>       │
│                                │     │   temporal.stage → <VM IP>     │
│                                │     │   *.apps.stage → <VM IP>       │
│                                │     │   ... (see DNS map)            │
└────────────────────────────────┘     └────────────────────────────────┘
```

### Key commands:

```bash
cd infra/terraform/staging

# Initialize (first time or after provider changes)
terraform init

# Preview changes
terraform plan

# Apply changes
terraform apply

# Outputs (VM IP, etc.)
terraform output
```

### Module structure:

```
infra/terraform/
├── staging/
│   ├── main.tf          # Server + DNS module usage
│   ├── variables.tf     # Input variables
│   └── terraform.tfvars # Actual values (gitignored)
├── modules/
│   ├── k3s-server/      # Hetzner VM + firewall module
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── outputs.tf
│   └── dns/             # Cloudflare DNS module
│       ├── main.tf
│       ├── variables.tf
│       └── outputs.tf
```

---

## 6. Layer 2: Ansible — OS & k3s

**Directory:** `infra/ansible/`

### What it installs:

```
On the VM (via SSH):

┌─────────────────────────────────────────────────────────────────┐
│  Role: common                                                    │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ apt packages: curl, wget, git, python3, jq, htop, etc.    │  │
│  │ Docker: from official repo (docker.io GPG key)             │  │
│  │ Helm: from upstream install script                          │  │
│  │ Kernel: disable swap, load br_netfilter                    │  │
│  │ Sysctl: net.bridge.bridge-nf-call-iptables = 1            │  │
│  │         net.ipv4.ip_forward = 1                            │  │
│  │ Falco: eBPF kernel tunables                                 │  │
│  └────────────────────────────────────────────────────────────┘  │
│                         ▼                                        │
│  Role: k3s                                                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ k3s install with flags:                                     │  │
│  │   --flannel-backend=none      (we use Cilium instead)       │  │
│  │   --disable-network-policy    (Cilium handles this)         │  │
│  │   --disable=servicelb         (Traefik handles LB)          │  │
│  │   --secrets-encryption        (encrypt etcd secrets at rest)│  │
│  │                                                              │  │
│  │ Post-install:                                                │  │
│  │   chmod 644 /etc/rancher/k3s/k3s.yaml  (readable kubeconfig)│  │
│  │   Fetch kubeconfig → save to local ~/.kube/                  │  │
│  │   Patch kubeconfig: 127.0.0.1 → actual VM IP               │  │
│  └────────────────────────────────────────────────────────────┘  │
│                         ▼                                        │
│  Role: cilium                                                    │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ Download + install Cilium CLI                               │  │
│  │ cilium install with:                                        │  │
│  │   --set kubeProxyReplacement=true                           │  │
│  │   --set encryption.enabled=true                             │  │
│  │   --set encryption.type=wireguard   (pod-to-pod encryption) │  │
│  │   --set hubble.enabled=true                                 │  │
│  │   --set hubble.ui.enabled=true                              │  │
│  │ Wait for cilium status = OK                                 │  │
│  └────────────────────────────────────────────────────────────┘  │
│                         ▼                                        │
│  Role: cert-manager                                              │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ Create cloudflare-api-token Secret in cert-manager ns       │  │
│  │ (needed by Terraform Layer 3 for DNS-01 solver)             │  │
│  └────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Key commands:

```bash
cd infra/ansible

# Full server setup (Phase 2)
ansible-playbook playbooks/server-setup.yml -i inventory/staging.yml

# Just infrastructure (skip apps)
ansible-playbook playbooks/infra.yml -i inventory/staging.yml

# Full site (infra + apps — V1 style)
ansible-playbook playbooks/site.yml -i inventory/staging.yml

# Verify kubeconfig works
export KUBECONFIG=~/.kube/zenith-staging.yaml
kubectl get nodes
```

### Playbook structure:

```
infra/ansible/
├── ansible.cfg              # SSH pipelining, fact caching
├── inventory/
│   └── staging.yml          # Host IP, SSH user, variables
├── playbooks/
│   ├── site.yml             # Full deployment (roles + apps)
│   ├── server-setup.yml     # Phase 2 only (common + k3s + cilium)
│   ├── infra.yml            # Infrastructure roles only
│   ├── apps.yml             # Application roles only
│   ├── build.yml            # Build images only
│   └── teardown.yml         # Destroy customer resources (with confirmation)
└── roles/
    ├── common/tasks/main.yml     # Base packages, Docker, kernel config
    ├── k3s/tasks/main.yml        # k3s installation
    ├── cilium/tasks/main.yml     # Cilium CNI installation
    └── cert-manager/tasks/main.yml # Prerequisite secrets
```

---

## 7. Layer 3: Terraform — Cluster Bootstrap

**Directory:** `infra/terraform/staging-k8s/` (calls module `modules/k8s-platform/`)

### What it installs (in dependency order):

```
┌─────────────────────────────────────────────────────────────────────┐
│  INSTALL ORDER (Terraform handles dependencies automatically)       │
│                                                                     │
│  Phase A: Foundations (no dependencies)                              │
│  ┌───────────────┐ ┌───────────────┐ ┌───────────────────────────┐ │
│  │ PriorityClass │ │ Hetzner CSI   │ │ cert-manager              │ │
│  │ (4 classes)   │ │ (kube-system) │ │ + letsencrypt-prod issuer │ │
│  └───────┬───────┘ └───────┬───────┘ └───────────┬───────────────┘ │
│          │                 │                     │                  │
│  Phase B: Depends on cert-manager + CSI                             │
│  ┌───────┴───────┐ ┌──────┴────────┐ ┌──────────┴──────────────┐  │
│  │ Sealed        │ │ CNPG operator │ │ Traefik HelmChartConfig  │  │
│  │ Secrets       │ │ (cnpg-system) │ │ (cross-namespace routing)│  │
│  └───────────────┘ └──────┬────────┘ └──────────────────────────┘  │
│                           │                                         │
│  Phase C: Depends on CNPG operator                                  │
│  ┌────────────────────────┴──────────────────────────────────────┐  │
│  │ keycloak-pg Cluster     │  free-pg Cluster                    │  │
│  │ (keycloak ns, 2 inst)   │  (zenith-shared ns, 2 inst)        │  │
│  └────────────┬────────────┘──────────────┬──────────────────────┘  │
│               │                           │                         │
│  Phase D: Depends on PG clusters                                    │
│  ┌────────────┴───┐ ┌────────────────────┴───────────────────────┐ │
│  │ Keycloak       │ │ Temporal (uses free-pg for persistence)    │ │
│  │ (uses kc-pg)   │ │                                            │ │
│  └────────────────┘ └────────────────────────────────────────────┘ │
│                                                                     │
│  Phase E: Depends on cert-manager ClusterIssuer                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐│
│  │ APISIX + │ │external- │ │ ArgoCD + │ │ Harbor + │ │ KEDA +   ││
│  │ etcd +   │ │dns       │ │ Image    │ │ TLS +    │ │ HTTP     ││
│  │ ingress  │ │(Cloudflar│ │ Updater  │ │ Ingress  │ │ add-on   ││
│  │ ctrl     │ │ e)       │ │          │ │ Route    │ │          ││
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘│
│                                                                     │
│  Phase F: Independent (no strict deps)                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                           │
│  │ Kyverno  │ │ Falco    │ │ Velero   │                           │
│  └──────────┘ └──────────┘ └──────────┘                           │
│                                                                     │
│  Phase G: Depends on cert-manager                                   │
│  ┌──────────────────────────────────────────────────────┐          │
│  │ Prometheus + Grafana + Alertmanager                   │          │
│  │ Loki (depends on Prometheus stack)                    │          │
│  │ Tempo (depends on Prometheus stack)                   │          │
│  │ OTel Collector (depends on Tempo)                     │          │
│  │ Hubble UI IngressRoute                                │          │
│  └──────────────────────────────────────────────────────┘          │
│                                                                     │
│  Phase H: Application charts (depends on platform + CNPG)          │
│  ┌──────────────────────────────────────────────────────┐          │
│  │ zenith-platform (sync-wave 0) → zenith-api (wave 1)  │          │
│  │ → zenith-landing (wave 1) → zenith-demo (wave 1)     │          │
│  └──────────────────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────────────────┘
```

### Module file layout:

```
infra/terraform/modules/k8s-platform/
├── main.tf             # Terraform block + PriorityClasses
├── certmanager.tf      # cert-manager + ClusterIssuer
├── sealed_secrets.tf   # Sealed Secrets controller
├── storage.tf          # Hetzner CSI + CNPG operator + PG clusters + backups
├── identity.tf         # Keycloak
├── gateway.tf          # APISIX + etcd + external-dns
├── traefik.tf          # Traefik HelmChartConfig (cross-namespace routing)
├── gitops.tf           # ArgoCD + Image Updater + App-of-Apps
├── registry.tf         # Harbor (customer registry) + TLS
├── temporal.tf         # Temporal workflow engine
├── security.tf         # Kyverno + Falco + Velero
├── observability.tf    # Prometheus + Loki + Tempo + OTel + Hubble UI
├── autoscaling.tf      # KEDA + HTTP add-on
├── pdb.tf              # PodDisruptionBudgets for HA
├── apps.tf             # zenith-platform, api, landing, demo Helm releases
├── variables.tf        # All input variables
└── outputs.tf          # All outputs
```

### Key commands:

```bash
cd infra/terraform/staging-k8s

# Initialize
terraform init

# Preview what will be installed
terraform plan

# Install everything
terraform apply

# Install specific component only
terraform apply -target=helm_release.apisix

# Destroy specific component
terraform destroy -target=helm_release.apisix
```

---

## 8. Configuration Reference

### Environment Variables Required

| Variable | Layer | Purpose | Where to get it |
|----------|-------|---------|----------------|
| `hcloud_token` | 1 + 3 | Hetzner API token | Hetzner Cloud Console → API Tokens |
| `cloudflare_api_token` | 1 + 3 | Cloudflare API token | Cloudflare → My Profile → API Tokens |
| `ssh_public_key` | 1 | SSH public key for VM | `~/.ssh/id_ed25519.pub` |
| `s3_access_key` | 3 | Hetzner S3 access key | Hetzner Cloud Console → Object Storage |
| `s3_secret_key` | 3 | Hetzner S3 secret key | Same as above |
| `github_token` | 3 | GitHub PAT for ArgoCD | GitHub → Settings → Developer settings |
| `admin_password` | 3 | Admin password (Grafana, Harbor) | Generate a strong password |
| `keycloak_admin_password` | 3 | Keycloak admin password | Generate a strong password |
| `keycloak_db_password` | 3 | Keycloak CNPG DB password | Generate a strong password |
| `temporal_db_password` | 3 | Temporal DB password | Generate a strong password |
| `registry_host` | 3 | Harbor host | `registry.stage.freezenith.com` |
| `registry_username` | 3 | Harbor robot account | Harbor → Robot Accounts |
| `registry_password` | 3 | Harbor robot password | Same as above |

### Key Terraform Variables (Layer 3)

| Variable | Default | Description |
|----------|---------|-------------|
| `environment` | `"staging"` | Affects replica counts, storage sizes, retention |
| `enable_apisix` | `true` | Enable APISIX gateway |
| `enable_argocd` | `true` | Enable ArgoCD GitOps |
| `enable_cnpg` | `true` | Enable CNPG operator |
| `enable_keycloak` | `true` | Enable Keycloak identity |
| `enable_temporal` | `true` | Enable Temporal workflows |
| `enable_harbor` | `true` | Enable customer Harbor |
| `enable_kyverno` | `true` | Enable policy engine |
| `enable_falco` | `true` | Enable runtime security |
| `enable_velero` | `true` | Enable backup |
| `enable_monitoring` | `true` | Enable full observability stack |
| `enable_keda` | `true` | Enable scale-to-zero |
| `argocd_target_revision` | `"staging"` | Git branch ArgoCD watches |
| `cluster_domain` | `"stage.freezenith.com"` | Domain for IngressRoutes |

---

## 9. Request Flow: Full Environment Setup

Complete sequence to set up a new environment from scratch:

```
Developer                    Hetzner Cloud             Cloudflare            VM (SSH)             K8s Cluster
    │                            │                        │                    │                      │
    │  terraform apply (Layer 1) │                        │                    │                      │
    ├───────────────────────────▶│                        │                    │                      │
    │                            │ Create VM              │                    │                      │
    │                            │ Create Firewall        │                    │                      │
    │                            │◀──── VM IP ────────────┤                    │                      │
    │                            │                        │                    │                      │
    ├────────────────────────────┼───────────────────────▶│                    │                      │
    │                            │                        │ Create DNS records │                      │
    │                            │                        │ *.stage → VM IP    │                      │
    │                            │                        │                    │                      │
    │  ansible-playbook (Layer 2)│                        │                    │                      │
    ├────────────────────────────┼────────────────────────┼───────────────────▶│                      │
    │                            │                        │                    │ Install packages     │
    │                            │                        │                    │ Install k3s          │
    │                            │                        │                    │ Install Cilium       │
    │                            │                        │                    │ Create secrets       │
    │◀─────── kubeconfig ────────┼────────────────────────┼────────────────────┤                      │
    │                            │                        │                    │                      │
    │  terraform apply (Layer 3) │                        │                    │                      │
    ├────────────────────────────┼────────────────────────┼────────────────────┼─────────────────────▶│
    │                            │                        │                    │                      │
    │                            │                        │                    │  Helm installs:      │
    │                            │                        │                    │  cert-manager        │
    │                            │                        │                    │  CNPG + PG clusters  │
    │                            │                        │                    │  Keycloak            │
    │                            │                        │                    │  APISIX + etcd       │
    │                            │                        │                    │  ArgoCD              │
    │                            │                        │                    │  ... (22 total)      │
    │                            │                        │                    │                      │
    │  Done! Platform is live.   │                        │                    │                      │
    │  ArgoCD will auto-deploy   │                        │                    │                      │
    │  apps on git push.         │                        │                    │                      │
    ▼                            ▼                        ▼                    ▼                      ▼
```

---

## 10. Troubleshooting

### Layer 1: Terraform (Hetzner + Cloudflare)

**Problem:** `Error: creating server: server limit reached`
```bash
# Check current server count
hcloud server list
# Solution: Delete old servers or request limit increase in Hetzner Console
```

**Problem:** `Error: cloudflare_record: record already exists`
```bash
# Import existing record into Terraform state
terraform import 'module.dns.cloudflare_record.platform["api"]' <zone_id>/<record_id>
# Find record ID: cloudflare API or CF dashboard
```

### Layer 2: Ansible

**Problem:** `UNREACHABLE! SSH connection timed out`
```bash
# Check if VM is reachable
ping <VM_IP>
# Check if SSH key is correct
ssh -i ~/.ssh/id_ed25519 root@<VM_IP>
# Check Hetzner firewall allows port 22
hcloud firewall list
```

**Problem:** `k3s failed to start`
```bash
# SSH into the VM and check k3s status
ssh root@<VM_IP>
systemctl status k3s
journalctl -u k3s -f
# Common fix: Cilium can't start if flannel wasn't disabled
# Re-run Ansible with --tags k3s
```

**Problem:** `Cilium not ready`
```bash
ssh root@<VM_IP>
cilium status
cilium connectivity test
# If stuck: restart k3s and re-run cilium install
systemctl restart k3s
```

### Layer 3: Terraform (k8s-platform)

**Problem:** `Error: timed out waiting for the condition (Helm release)`
```bash
# Check which pods are failing
kubectl get pods -A | grep -v Running
# Check specific component
kubectl describe pod -n <namespace> <pod-name>
kubectl logs -n <namespace> <pod-name>
```

**Problem:** `Error: cannot patch resource — field is immutable`
```bash
# Usually happens with PriorityClasses, StorageClasses, CRDs
# Solution: Delete the resource manually, then terraform apply
kubectl delete priorityclass <name>
terraform apply
```

**Problem:** `Helm release stuck in pending-install`
```bash
# List Helm releases with status
helm list -A
# Force delete the stuck release
helm uninstall <release-name> -n <namespace>
# Re-run terraform apply
```

---

## 11. Upgrade Path

### Upgrading k3s

```bash
# On the VM:
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=v1.35.0+k3s1 sh -s - \
  --flannel-backend=none \
  --disable-network-policy \
  --disable=servicelb \
  --secrets-encryption

# Verify
kubectl get nodes  # should show new version
```

### Upgrading Helm chart versions

```hcl
# In infra/terraform/modules/k8s-platform/variables.tf
# Update the version variable, then:
terraform plan  # preview changes
terraform apply # upgrade
```

### Adding a new component

1. Create a new `.tf` file in `modules/k8s-platform/`
2. Add `enable_<component>` variable with default `true`
3. Add `<component>_version` variable
4. Write the `helm_release` resource with appropriate `depends_on`
5. Run `terraform plan` → `terraform apply`

### Environment parity (staging → production)

Same module, different variables:
```
infra/terraform/staging-k8s/     → calls modules/k8s-platform/ with staging vars
infra/terraform/production-k8s/  → calls modules/k8s-platform/ with production vars
```

Production overrides: more replicas, larger storage, longer retention, HA mode.
