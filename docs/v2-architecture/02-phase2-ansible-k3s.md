# Phase 2: Ansible + k3s + Cilium

> **Zenith V2 Platform Architecture -- Phase 2 of 5**
>
> This phase transforms the blank Ubuntu 24.04 server from Phase 1 into a fully
> operational Kubernetes cluster. It installs k3s, Cilium CNI, cert-manager, and
> configures the server for production workloads. After this phase, you have a
> working cluster with `kubectl` access, TLS certificate automation, and
> enterprise-grade networking -- ready for application deployment in Phase 3.

---

## Table of Contents

1. [Why This Phase Exists](#why-this-phase-exists)
2. [What Gets Installed](#what-gets-installed)
3. [Architecture Overview](#architecture-overview)
4. [Ansible Project Structure](#ansible-project-structure)
5. [Role Breakdown](#role-breakdown)
6. [Variables Reference](#variables-reference)
7. [How to Run](#how-to-run)
8. [V2 Enhancements](#v2-enhancements)
9. [Verification Checklist](#verification-checklist)
10. [Troubleshooting](#troubleshooting)
11. [What Happens Next](#what-happens-next)

---

## Why This Phase Exists

Phase 1 gave us a server and DNS records. But that server is a blank Ubuntu 24.04 VM --
no container runtime, no Kubernetes, no networking stack, no TLS, no package dependencies.
It can accept SSH connections, and that is it.

Phase 2 uses **Ansible** to bridge that gap. We need to:

1. **Install base packages** -- curl, git, Docker, Helm, and everything else the system needs
2. **Configure the Linux kernel** -- disable swap, enable IP forwarding, load bridge modules
3. **Install k3s** -- a lightweight Kubernetes distribution that runs the entire control plane
   and data plane in a single binary
4. **Install Cilium** -- a CNI (Container Network Interface) plugin that replaces k3s's
   default Flannel with eBPF-based networking, NetworkPolicy enforcement, and WireGuard
   encryption
5. **Install cert-manager** -- automated TLS certificate provisioning via Let's Encrypt
6. **Install hcloud-csi** -- a CSI (Container Storage Interface) driver that provisions
   Hetzner Volumes on demand for persistent storage

### Why Ansible (not Terraform, not shell scripts)?

Ansible is the right tool for **server configuration management** because:

- **Idempotent**: Run the playbook 10 times, get the same result. Unlike a bash script,
  Ansible checks the current state before making changes. The `creates:` parameter on
  shell tasks, `stat` checks for binaries, and `when:` conditions all ensure that nothing
  is re-executed unnecessarily.
- **Role-based decomposition**: Each concern (base packages, k3s, Cilium, cert-manager) is
  isolated in its own role with its own tasks. You can run just one role, test it in
  isolation, or swap it out entirely.
- **Inventory-driven**: The same playbook works for staging and production by swapping the
  inventory file. No code duplication, no if-else branches for environment differences.
- **SSH-native**: Ansible connects over SSH (which Phase 1 already configured). There is no
  agent to install on the server, no daemon to manage, no port to open. It uses the exact
  same SSH key that Terraform uploaded in Phase 1.
- **Variable hierarchy**: Defaults in `group_vars/all.yml`, environment overrides in
  `group_vars/staging.yml`, and host-specific values in `inventory/staging.yml` merge
  automatically. You define a value once and override it where needed.

**Why not Terraform for this?** Terraform manages cloud API resources (VMs, DNS records,
firewalls). It is poor at managing the *inside* of a server -- installing packages,
configuring systemd services, or running shell commands conditionally. Terraform's
`remote-exec` provisioner exists but is fragile, has no retry logic, and lacks idempotency.

**Why not a shell script?** Shell scripts are not idempotent. Running `apt install docker-ce`
twice is harmless, but running `curl | sh` to install k3s twice may reinstall and restart
the cluster. Shell scripts also lack inventory management, variable hierarchies, role
composition, and the ability to run a subset of tasks via tags.

---

## What Gets Installed

### Server Layer Diagram

```
+====================================================================+
|  Ubuntu 24.04 Server (Hetzner VM from Phase 1)                     |
|                                                                     |
|  Layer 0: Base System                                               |
|  +----------------------------------------------------------------+ |
|  |  apt packages: curl, wget, git, jq, unzip, ca-certificates    | |
|  |  python3-pip, apt-transport-https, gnupg, lsb-release         | |
|  +----------------------------------------------------------------+ |
|                                                                     |
|  Layer 1: Container Tooling                                         |
|  +----------------------------------------------------------------+ |
|  |  Docker CE     -- builds container images (docker build)       | |
|  |  Helm 3        -- installs Kubernetes charts (cert-manager)    | |
|  +----------------------------------------------------------------+ |
|                                                                     |
|  Layer 2: Kernel Configuration                                      |
|  +----------------------------------------------------------------+ |
|  |  swap: OFF               (kubelet requirement)                 | |
|  |  br_netfilter: loaded    (bridge traffic through iptables)     | |
|  |  ip_forward: 1           (pod-to-pod routing)                  | |
|  |  bridge-nf-call-iptables: 1  (Services + NetworkPolicy)       | |
|  +----------------------------------------------------------------+ |
|                                                                     |
|  Layer 3: Kubernetes (k3s v1.34.3+k3s1)                            |
|  +----------------------------------------------------------------+ |
|  |  Control Plane:                                                | |
|  |    kube-apiserver (:6443)                                      | |
|  |    etcd (embedded SQLite or embedded etcd)                     | |
|  |    kube-scheduler                                              | |
|  |    kube-controller-manager                                     | |
|  |                                                                | |
|  |  Data Plane:                                                   | |
|  |    kubelet (:10250)                                            | |
|  |    containerd (container runtime)                              | |
|  |                                                                | |
|  |  Built-in Addons:                                              | |
|  |    Traefik v3 (:80, :443)  -- ingress controller               | |
|  |    CoreDNS                 -- cluster DNS                      | |
|  |    Metrics Server          -- resource metrics for HPA         | |
|  |    Local Path Provisioner  -- default StorageClass (ephemeral) | |
|  +----------------------------------------------------------------+ |
|                                                                     |
|  Layer 4: CNI -- Cilium 1.16.5 (replaces Flannel)                  |
|  +----------------------------------------------------------------+ |
|  |  cilium-agent (DaemonSet)    -- eBPF datapath on every node    | |
|  |  cilium-operator             -- cluster-wide Cilium management | |
|  |  kube-proxy replacement      -- eBPF replaces iptables         | |
|  |  NetworkPolicy enforcement   -- L3/L4/L7 isolation             | |
|  |  [V2] WireGuard encryption   -- pod-to-pod encryption          | |
|  |  [V2] Hubble relay + UI      -- network flow observability     | |
|  +----------------------------------------------------------------+ |
|                                                                     |
|  Layer 5: Cluster Services                                          |
|  +----------------------------------------------------------------+ |
|  |  cert-manager v1.17.2   -- TLS certificates from Let's Encrypt| |
|  |  [V2] hcloud-csi        -- Hetzner Volume StorageClass         | |
|  +----------------------------------------------------------------+ |
+====================================================================+
```

### Component Versions (from `group_vars/all.yml`)

| Component | Version | Variable | Purpose |
|-----------|---------|----------|---------|
| k3s | v1.34.3+k3s1 | `k3s_version` | Lightweight Kubernetes distribution |
| Cilium | 1.16.5 | `cilium_version` | eBPF-based CNI with NetworkPolicy |
| cert-manager | v1.17.2 | `cert_manager_version` | Automated TLS certificates |
| Helm | Latest 3.x | (installed via script) | Kubernetes package manager |
| Docker CE | Latest stable | (Ubuntu apt repo) | Container image builds |
| PostgreSQL | 16-alpine | `postgres_image` | Application database |

---

## Architecture Overview

### Single-Node Staging

```
+=========================================================================+
|  zenith-staging (Hetzner cx22/cx32)                                     |
|  Ubuntu 24.04 | k3s v1.34.3 | Cilium 1.16.5                           |
|                                                                         |
|  Linux Kernel                                                           |
|  +-- br_netfilter loaded, ip_forward=1, swap disabled                  |
|                                                                         |
|  k3s (single binary, all components in one process)                     |
|  +-----------------------------------+                                  |
|  |  Control Plane                    |                                  |
|  |  +-- kube-apiserver (:6443)       |                                  |
|  |  +-- etcd (embedded)              |                                  |
|  |  +-- kube-scheduler               |                                  |
|  |  +-- kube-controller-manager      |                                  |
|  +-----------------------------------+                                  |
|  |  Data Plane                       |                                  |
|  |  +-- kubelet (:10250)             |                                  |
|  |  +-- containerd (container runtime)|                                 |
|  +-----------------------------------+                                  |
|                                                                         |
|  Cilium (DaemonSet in kube-system)                                      |
|  +-----------------------------------+                                  |
|  |  +-- cilium-agent (per node)      |                                  |
|  |  +-- cilium-operator              |                                  |
|  |  +-- [V2] hubble-relay            |                                  |
|  |  +-- [V2] hubble-ui               |                                  |
|  +-----------------------------------+                                  |
|                                                                         |
|  Built-in k3s Addons                                                    |
|  +-- Traefik (:80, :443) -- ingress controller                         |
|  +-- CoreDNS             -- cluster DNS (*.svc.cluster.local)          |
|  +-- Metrics Server      -- resource metrics for kubectl top / HPA     |
|  +-- Local Path Provisioner -- default StorageClass for PVCs           |
|                                                                         |
|  Cluster Services (Helm-managed)                                        |
|  +-- cert-manager        -- TLS certificates from Let's Encrypt        |
|  +-- [V2] hcloud-csi     -- Hetzner Volume dynamic provisioning        |
+=========================================================================+
```

### Network Flow (Request Lifecycle)

This shows how a user request travels from the browser to an application pod:

```
Client Browser
    |
    | HTTPS request to api.stage.freezenith.com
    v
Cloudflare DNS (resolves to Hetzner server IP)
    |
    | TCP connection to :443
    v
Hetzner Firewall (allows :443 from all IPs)
    |
    v
Traefik Ingress Controller (k3s built-in, port :443)
    |
    | Matches IngressRoute: Host(`api.stage.freezenith.com`)
    | Terminates TLS using cert-manager certificate
    v
Cilium eBPF Datapath (pod networking layer)
    |
    | Routes to ClusterIP Service -> Pod IP
    v
Application Pod (zenith-api on :8080)
    |
    | Processes request, queries database
    v
Cilium eBPF Datapath (pod-to-pod)
    |
    | [V2: WireGuard encrypted]
    v
PostgreSQL Pod (zenith-postgres on :5432)
```

### Pod-to-Pod Communication (with Cilium)

```
+----- Pod A (zenith-api) ------+     +----- Pod B (postgres) -------+
|  eth0: 10.42.0.15             |     |  eth0: 10.42.0.22            |
|                                |     |                               |
|  App --> TCP:5432 ------------|---->|  PostgreSQL                   |
+--------------------------------+     +-------------------------------+
        |                                      |
        +------ Cilium eBPF datapath ----------+
        |  - NetworkPolicy: ALLOW api->postgres:5432                  |
        |  - NetworkPolicy: DENY  web->postgres:5432                  |
        |  - [V2] WireGuard: ChaCha20-Poly1305 encrypted             |
        +-------------------------------------------------------------+
```

This is why Cilium matters: without it (using Flannel), there is no NetworkPolicy
enforcement. Any pod in any namespace can connect to any other pod. With Cilium, you
define explicit allow rules, and everything else is denied by default when you create a
`default-deny` policy.

---

## Ansible Project Structure

```
infra/ansible/
    |
    +-- ansible.cfg                 # Ansible configuration
    +-- requirements.yml            # Galaxy collection dependencies
    |
    +-- inventory/
    |   +-- staging.yml             # Staging: host IP, domains, namespaces, tenants
    |   +-- production.yml          # Production: same structure, different values
    |
    +-- group_vars/
    |   +-- all.yml                 # Shared defaults (versions, images, ports, resources)
    |   +-- staging.yml             # Staging overrides (lower resources, test secrets)
    |   +-- production.yml          # Production overrides (real secrets, full resources)
    |
    +-- playbooks/
    |   +-- site.yml                # Full deployment: infra + build + deploy
    |   +-- server-setup.yml        # Phase 2 only: common + k3s + kubeconfig
    |   +-- infra.yml               # Infrastructure only (no app deployment)
    |   +-- build.yml               # Build and import container images
    |   +-- apps.yml                # Deploy applications only
    |   +-- teardown.yml            # Destroy everything
    |
    +-- roles/
        |
        +-- Phase 2 roles (this document):
        |   +-- common/tasks/main.yml        # apt, Docker, Helm, swap, sysctl
        |   +-- k3s/tasks/main.yml           # k3s install, kubeconfig, kubectl alias
        |   +-- cilium/tasks/main.yml        # Cilium CLI install, cilium deploy
        |   +-- cert-manager/tasks/main.yml  # Helm install, ClusterIssuer
        |
        +-- Phase 3+ roles (later documents):
            +-- postgres/               # PostgreSQL StatefulSet
            +-- traefik-config/         # Traefik IngressRoute configuration
            +-- zenith-build/           # Docker image builds
            +-- zenith-import/          # Image import to k3s containerd
            +-- zenith-namespaces/      # Kubernetes namespace creation
            +-- zenith-api/             # API deployment
            +-- zenith-landing/         # Landing page deployment
            +-- zenith-mc/              # Mission Control deployment
            +-- zenith-web/             # Web Platform deployment
            +-- zenith-ingress/         # IngressRoute + TLS certificates
            +-- keda/                   # KEDA autoscaler (optional)
            +-- monitoring/             # Prometheus + Grafana + Loki (optional)
            +-- dns/                    # DNS verification
```

### Variable Hierarchy

Ansible merges variables in a specific order. Later levels override earlier ones:

```
group_vars/all.yml              <-- Base defaults
    |                               k3s_version, cilium_version, image names, ports,
    |                               resource requests/limits, PostgreSQL config
    v
group_vars/staging.yml          <-- Environment overrides
  OR group_vars/production.yml      Lower resources for staging, vault secrets,
    |                               CORS origins
    v
inventory/staging.yml vars      <-- Host-specific values
  OR inventory/production.yml       Domain names, namespaces, tenant list,
    |                               enable_cilium, enable_keda, enable_monitoring
    v
--extra-vars (CLI)              <-- Highest priority (one-off overrides)
                                    ansible-playbook ... -e "k3s_version=v1.35.0+k3s1"
```

For example, `resources.api.limits.cpu` is `250m` in `all.yml` but overridden to `150m`
in `group_vars/staging.yml` because the staging server is smaller. And `enable_cilium`
defaults to `false` in `all.yml` but is set to `true` in `inventory/staging.yml` because
staging is our testbed for Cilium before enabling it in production.

### The Main Playbook (`site.yml`)

```yaml
# playbooks/site.yml -- Full Zenith deployment (infrastructure + applications)
#
# Tags allow running subsets:
#   --tags infra          # Just infrastructure (common, k3s, cilium, cert-manager, postgres)
#   --tags build,deploy   # Just build images and deploy apps
#   --tags k3s,cilium     # Just k3s and Cilium

- name: Deploy Zenith Platform
  hosts: all
  gather_facts: true

  roles:
    # --- Infrastructure (Phase 2) ---
    - role: common
      tags: [common, infra]

    - role: k3s
      tags: [k3s, infra]

    - role: cilium
      when: enable_cilium | default(false)     # Conditional: skip if Cilium not wanted
      tags: [cilium, infra]

    - role: cert-manager
      tags: [cert-manager, infra]

    - role: postgres
      tags: [postgres, infra]

    # --- Build & Deploy (Phase 3+) ---
    - role: zenith-build
      tags: [build]

    - role: zenith-import
      tags: [build, import]

    # ... (application roles follow)
```

Notice that the `cilium` role has a `when: enable_cilium | default(false)` guard. This
means Cilium is only installed when the inventory explicitly enables it. Staging sets
`enable_cilium: true`, while the default in `all.yml` is `false`. This lets you test
Cilium in staging before rolling it out to production.

---

## Role Breakdown

### Role: `common` -- Base Server Setup

**Path**: `infra/ansible/roles/common/tasks/main.yml`

**Purpose**: Install prerequisite packages, Docker CE, Helm 3, and configure the Linux
kernel for Kubernetes networking. This role transforms a vanilla Ubuntu server into one
that can run containers and route network traffic correctly.

```yaml
# roles/common/tasks/main.yml (full, annotated)

# --- Step 1: System packages ---
- name: Update apt cache
  ansible.builtin.apt:
    update_cache: true
    cache_valid_time: 3600          # Don't re-update if refreshed < 1 hour ago

- name: Install base packages
  ansible.builtin.apt:
    name:
      - curl                        # HTTP client (download scripts)
      - wget                        # HTTP client (alternative to curl)
      - git                         # Clone the Zenith repo onto the server
      - apt-transport-https         # HTTPS support for apt repos
      - ca-certificates             # TLS certificate bundle
      - gnupg                       # GPG for verifying repo signing keys
      - lsb-release                 # Ubuntu release info (used by Docker repo)
      - python3-pip                 # Python packages (Ansible deps on remote)
      - jq                          # JSON processing (debug/scripting)
      - unzip                       # Archive extraction
    state: present

# --- Step 2: Docker CE ---
# Why Docker when k3s uses containerd? Because we build images with `docker build`
# on the server and import them into k3s via `docker save | k3s ctr images import`.
# This avoids needing a container registry (no Docker Hub, no GHCR, no self-hosted).
- name: Install Docker (if not present)
  when: ansible_facts.packages is not defined or 'docker-ce' not in ansible_facts.packages
  block:
    - name: Create keyrings directory
      ansible.builtin.file:
        path: /etc/apt/keyrings
        state: directory
        mode: "0755"

    - name: Download Docker GPG key
      ansible.builtin.shell: |
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
          | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        chmod a+r /etc/apt/keyrings/docker.gpg
      args:
        creates: /etc/apt/keyrings/docker.gpg    # Idempotent: skip if file exists

    - name: Add Docker repository
      ansible.builtin.apt_repository:
        repo: >-
          deb [signed-by=/etc/apt/keyrings/docker.gpg]
          https://download.docker.com/linux/ubuntu
          {{ ansible_facts['distribution_release'] }} stable
        state: present

    - name: Install Docker CE
      ansible.builtin.apt:
        name: [docker-ce, docker-ce-cli, containerd.io]
        state: present
        update_cache: true

    - name: Enable and start Docker
      ansible.builtin.systemd:
        name: docker
        enabled: true
        state: started

# --- Step 3: Helm ---
# Helm is needed to install cert-manager, monitoring stack, and other charts.
# The official install script is idempotent via the `creates:` guard.
- name: Install Helm (if not present)
  ansible.builtin.shell: |
    curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
  args:
    creates: /usr/local/bin/helm

# --- Step 4: Kernel configuration for Kubernetes ---
- name: Disable swap
  ansible.builtin.command: swapoff -a
  changed_when: false

- name: Remove swap from fstab
  ansible.builtin.lineinfile:
    path: /etc/fstab
    regexp: '.*swap.*'
    state: absent

- name: Load br_netfilter kernel module
  community.general.modprobe:
    name: br_netfilter
    state: present

- name: Persist br_netfilter module
  ansible.builtin.lineinfile:
    path: /etc/modules-load.d/k8s.conf
    line: br_netfilter
    create: true

- name: Set sysctl for k8s networking
  ansible.posix.sysctl:
    name: "{{ item.key }}"
    value: "{{ item.value }}"
    sysctl_set: true
    reload: true
  loop:
    - { key: net.bridge.bridge-nf-call-iptables,  value: "1" }
    - { key: net.bridge.bridge-nf-call-ip6tables, value: "1" }
    - { key: net.ipv4.ip_forward,                 value: "1" }
```

**Why each kernel setting matters:**

| Setting | What It Does | What Breaks Without It |
|---------|-------------|----------------------|
| `swapoff -a` | Disables swap memory entirely | kubelet refuses to start. Kubernetes assumes it has full control over memory allocation. Swap introduces unpredictable latency and defeats resource limits. |
| `br_netfilter` | Makes bridged (container) traffic visible to iptables | Pod-to-pod traffic silently drops. NetworkPolicies and kube-proxy Service routing fail because iptables never sees the packets. |
| `net.ipv4.ip_forward=1` | Allows the kernel to route packets between network interfaces | Pods cannot reach the internet or each other across nodes. The kernel drops forwarded packets instead of routing them. |
| `bridge-nf-call-iptables=1` | Bridge traffic passes through iptables rules | Kubernetes Services (ClusterIP) stop working. kube-proxy relies on iptables rules for Service routing, and those rules only fire if bridge traffic goes through iptables. |

**Why swap is a security concern too:** When the kernel swaps memory pages to disk,
sensitive data (API keys, JWT secrets, database passwords, encryption keys held in memory)
may be written to the swap partition in plaintext. Disabling swap ensures sensitive data
stays in RAM, which is volatile and cleared on reboot.

### Role: `k3s` -- Kubernetes Installation

**Path**: `infra/ansible/roles/k3s/tasks/main.yml`

**Purpose**: Install k3s, the lightweight Kubernetes distribution. k3s packages the entire
Kubernetes control plane (API server, etcd, scheduler, controller-manager) plus the data
plane (kubelet, containerd) into a single ~70 MB binary.

#### Why k3s (not full Kubernetes, not k0s, not MicroK8s)?

| Feature | k3s | Full k8s (kubeadm) | k0s | MicroK8s |
|---------|-----|---------------------|-----|----------|
| Binary size | ~70 MB | ~300+ MB | ~160 MB | ~200 MB |
| Base memory overhead | ~512 MB | ~1.5 GB | ~800 MB | ~1 GB |
| Embedded etcd | Yes | No (separate cluster) | Yes | No (dqlite) |
| Built-in ingress | Traefik | No | No | No |
| Built-in storage | Local Path | No | No | No |
| CNCF certified | Yes | Yes | Yes | No |
| Maintained by | SUSE/Rancher | Kubernetes SIG | Mirantis | Canonical |

k3s wins for single-node and small-cluster deployments because:

1. **Low overhead**: 512 MB base memory means more room for workloads on a cx22 (4 GB RAM)
   or cx32 (8 GB RAM). Full kubeadm Kubernetes eats 1.5 GB before you deploy anything.
2. **Batteries included**: Traefik, CoreDNS, Metrics Server, and Local Path Provisioner
   ship with k3s. No separate Helm installs needed for basic cluster functionality.
3. **Single binary**: `k3s server` starts everything. No etcd cluster to bootstrap, no
   separate kubelet service to configure, no control-plane PKI to set up manually.
4. **Auto-managed addons**: Traefik and CoreDNS are deployed as HelmChart resources that
   k3s auto-manages. Upgrades happen by upgrading k3s itself.
5. **Production-ready**: k3s is CNCF certified and backed by SUSE/Rancher. It runs in
   production at thousands of companies, from edge devices to cloud servers.

```yaml
# roles/k3s/tasks/main.yml (full, annotated)

- name: Check if k3s is installed
  ansible.builtin.stat:
    path: /usr/local/bin/k3s
  register: k3s_binary

# --- Path A: Install k3s with Cilium-compatible flags ---
- name: Install k3s (with Cilium CNI)
  when: not k3s_binary.stat.exists and (enable_cilium | default(false))
  ansible.builtin.shell: |
    curl -sfL https://get.k3s.io | \
      INSTALL_K3S_VERSION="{{ k3s_version }}" \
      INSTALL_K3S_EXEC="--flannel-backend=none --disable-network-policy --disable=servicelb" \
      sh -
  args:
    creates: /usr/local/bin/k3s

# --- Path B: Install k3s with default Flannel CNI ---
- name: Install k3s (default Flannel)
  when: not k3s_binary.stat.exists and not (enable_cilium | default(false))
  ansible.builtin.shell: |
    curl -sfL https://get.k3s.io | \
      INSTALL_K3S_VERSION="{{ k3s_version }}" \
      sh -
  args:
    creates: /usr/local/bin/k3s

# --- Wait for the cluster to come up ---
- name: Wait for k3s to be ready
  ansible.builtin.command: k3s kubectl get node
  register: k3s_ready
  retries: 30          # Up to 30 retries
  delay: 5             # 5 seconds apart = 150 seconds max wait
  until: k3s_ready.rc == 0
  changed_when: false

# --- Make kubeconfig accessible ---
- name: Ensure kubeconfig is accessible
  ansible.builtin.file:
    path: "{{ kubeconfig_path }}"    # /etc/rancher/k3s/k3s.yaml
    mode: "0644"

- name: Set KUBECONFIG environment variable
  ansible.builtin.lineinfile:
    path: /etc/environment
    line: "KUBECONFIG={{ kubeconfig_path }}"
    regexp: "^KUBECONFIG="

# --- Convenience alias ---
- name: Create kubectl alias
  ansible.builtin.lineinfile:
    path: /root/.bashrc
    line: "alias kubectl='k3s kubectl'"
    regexp: "^alias kubectl="
```

#### k3s Installation Flags Explained

When Cilium is enabled (`enable_cilium: true`), k3s is installed with three critical flags.
Each one disables a k3s built-in component that Cilium replaces:

| Flag | What It Disables | Why |
|------|-----------------|-----|
| `--flannel-backend=none` | Flannel CNI entirely | Cilium will be the CNI. Running two CNIs causes IP address conflicts and routing loops. You cannot have two CNIs assigning pod IPs. |
| `--disable-network-policy` | k3s built-in NetworkPolicy controller | Cilium provides its own NetworkPolicy implementation that supports L3/L4/L7. Running both causes duplicate enforcement and confusing behavior. |
| `--disable=servicelb` | Klipper ServiceLB | Klipper is a simple LoadBalancer implementation for bare-metal. Traefik handles all ingress traffic, and we do not use LoadBalancer-type Services. Disabling it saves memory. |

**What happens if you forget `--flannel-backend=none`?** Both Flannel and Cilium try to
assign IP addresses to pods. Pods get two IP addresses, routing tables conflict, and
networking breaks silently -- some connections work, others hang. Debugging this is
extremely difficult because `kubectl` shows pods as Running but they cannot communicate.

#### k3s Built-in Components

When k3s starts, it automatically deploys these as Kubernetes resources in `kube-system`:

```
kube-system namespace:
    +-- traefik (Deployment)            -- Ingress controller (ports 80, 443)
    +-- coredns (Deployment)            -- Cluster DNS (resolves *.svc.cluster.local)
    +-- metrics-server (Deployment)     -- Resource metrics for kubectl top / HPA
    +-- local-path-provisioner (Dep.)   -- Default StorageClass for PVCs

    (After Cilium install:)
    +-- cilium (DaemonSet)              -- CNI agent on every node
    +-- cilium-operator (Deployment)    -- Cluster-wide Cilium management
```

### Role: `cilium` -- CNI Installation

**Path**: `infra/ansible/roles/cilium/tasks/main.yml`

**Purpose**: Install Cilium as the Container Network Interface (CNI) plugin. Cilium uses
eBPF (extended Berkeley Packet Filter) to implement networking, security, and observability
at the Linux kernel level. It replaces both Flannel (pod networking) and kube-proxy
(Service routing) with a single, faster, more capable system.

#### Why Cilium (not Flannel, not Calico, not Weave)?

| Feature | Cilium | Flannel | Calico | Weave |
|---------|--------|---------|--------|-------|
| eBPF-based | Yes | No (overlay) | Partial (eBPF DP) | No |
| NetworkPolicy | L3/L4/L7 | **None** | L3/L4 only | L3/L4 only |
| Encryption | WireGuard or IPsec | **None** | WireGuard | IPsec only |
| Hubble observability | Yes | No | No | No |
| kube-proxy replacement | Full | No | Partial | No |
| Performance overhead | Very low (eBPF) | Low | Medium (iptables) | High |
| CNCF status | Graduated | N/A | N/A | N/A |

**Flannel** is k3s's default CNI. It is simple and reliable for basic pod networking, but
it provides **zero** security features -- no NetworkPolicy support, no encryption, no
observability. For a platform that hosts customer workloads in shared namespaces, this is
not acceptable.

**Calico** supports NetworkPolicy but its iptables-based datapath is slower than Cilium's
eBPF implementation. Calico has a newer eBPF datapath, but it is less mature than Cilium's.

**Cilium** provides everything we need:

1. **NetworkPolicy enforcement**: Isolate tenant namespaces from each other. Pod A in
   `zenith-embermind-staging` must not be able to reach Pod B in `zenith-staging` unless
   explicitly allowed by a NetworkPolicy rule. Cilium enforces this at L3 (IP), L4 (port),
   and L7 (HTTP path, gRPC method).
2. **WireGuard encryption**: Encrypt all pod-to-pod traffic. Even if an attacker compromises
   one container, they cannot sniff traffic from other pods on the wire.
3. **Hubble observability**: Real-time network flow visibility -- which pod talked to which,
   what was blocked by NetworkPolicy, latency percentiles, DNS query logs.
4. **kube-proxy replacement**: Cilium's eBPF implementation of Service routing is faster than
   kube-proxy's iptables rules and scales better (no O(n) iptables chain scanning).

#### What is eBPF?

eBPF (extended Berkeley Packet Filter) is a technology built into the Linux kernel (5.x+)
that allows running sandboxed programs inside the kernel without modifying kernel source
code or loading kernel modules. Think of it as "JavaScript for the Linux kernel."

Traditional CNIs like Flannel and Calico use **iptables** for packet routing. Every packet
traverses a chain of iptables rules, which are evaluated sequentially. With hundreds of
Services, this becomes a performance bottleneck.

Cilium uses **eBPF programs** attached to network interfaces. These programs run in the
kernel's fast path, making routing decisions in O(1) time using hash maps instead of O(n)
iptables chains. The result: lower latency, higher throughput, and better CPU efficiency.

```
Traditional CNI (iptables):                 Cilium (eBPF):

  Packet arrives                              Packet arrives
       |                                           |
       v                                           v
  iptables PREROUTING chain                   eBPF program (tc ingress)
       |                                           |
       v                                           | O(1) hash map lookup
  Rule 1: match? no -> next                        | for Service/Pod IP
  Rule 2: match? no -> next                        |
  Rule 3: match? no -> next                        v
  ...                                         Packet routed (done)
  Rule N: match? yes -> DNAT
       |
       v
  iptables FORWARD chain
  ... (more rules)
       |
       v
  Packet routed
```

```yaml
# roles/cilium/tasks/main.yml (full, annotated)

# --- Step 1: Install the Cilium CLI binary ---
- name: Check if Cilium CLI is installed
  ansible.builtin.stat:
    path: /usr/local/bin/cilium
  register: cilium_cli

- name: Install Cilium CLI
  when: not cilium_cli.stat.exists
  ansible.builtin.shell: |
    CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
    GOOS=linux GOARCH=amd64
    curl -L --fail --remote-name-all \
      https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-${GOOS}-${GOARCH}.tar.gz{,.sha256sum}
    sha256sum --check cilium-${GOOS}-${GOARCH}.tar.gz.sha256sum
    tar xzvfC cilium-${GOOS}-${GOARCH}.tar.gz /usr/local/bin
    rm cilium-${GOOS}-${GOARCH}.tar.gz{,.sha256sum}
  args:
    creates: /usr/local/bin/cilium

# --- Step 2: Install Cilium into the Kubernetes cluster ---
- name: Check if Cilium is installed in cluster
  ansible.builtin.command: >
    k3s kubectl get daemonset cilium -n kube-system
  register: cilium_ds
  failed_when: false
  changed_when: false

- name: Install Cilium via CLI
  when: cilium_ds.rc != 0
  ansible.builtin.command: >
    cilium install
    --version {{ cilium_version }}
    --set kubeProxyReplacement=true
    --set k8sServiceHost=127.0.0.1
    --set k8sServicePort=6443
  environment:
    KUBECONFIG: "{{ kubeconfig_path }}"

# --- Step 3: Wait for Cilium to be fully operational ---
- name: Wait for Cilium to be ready
  ansible.builtin.command: cilium status --wait
  environment:
    KUBECONFIG: "{{ kubeconfig_path }}"
  register: cilium_status
  retries: 12           # Up to 12 retries
  delay: 10             # 10 seconds apart = 120 seconds max wait
  until: cilium_status.rc == 0
  changed_when: false
```

#### Cilium Installation Flags Explained

| Setting | Value | Why |
|---------|-------|-----|
| `--version 1.16.5` | Pin to specific version | Prevents accidental upgrades. Cilium minor versions can change behavior. |
| `--set kubeProxyReplacement=true` | Replace kube-proxy entirely | Cilium's eBPF Service routing is faster than iptables. Eliminates kube-proxy's memory usage and iptables rule bloat. |
| `--set k8sServiceHost=127.0.0.1` | API server address | k3s runs the API server on localhost. Cilium needs to know where to find it. |
| `--set k8sServicePort=6443` | API server port | Default k3s API server port. |

### Role: `cert-manager` -- TLS Certificate Automation

**Path**: `infra/ansible/roles/cert-manager/tasks/main.yml`

**Purpose**: Install cert-manager, a Kubernetes-native certificate management controller.
cert-manager automates the issuance and renewal of TLS certificates from Let's Encrypt
(or other ACME providers). Without it, you would need to manually generate certificates,
upload them as Kubernetes Secrets, and remember to renew them every 90 days.

#### Why cert-manager?

TLS certificates from Let's Encrypt are free but expire every 90 days. Managing this
manually for 6-10 subdomains across staging and production is error-prone. cert-manager:

- **Automates issuance**: Create a `Certificate` resource, cert-manager handles the rest
- **Automates renewal**: Renews certificates 30 days before expiry (configurable)
- **Handles ACME challenges**: Proves domain ownership via HTTP-01 or DNS-01 automatically
- **Stores certificates as Secrets**: Traefik reads them natively from Kubernetes Secrets
- **Supports multiple issuers**: Let's Encrypt staging (for testing), Let's Encrypt
  production, self-signed, and custom CAs

```yaml
# roles/cert-manager/tasks/main.yml (full, annotated)

# --- Step 1: Add the Jetstack Helm repository ---
- name: Add cert-manager Helm repo
  kubernetes.core.helm_repository:
    name: jetstack
    repo_url: https://charts.jetstack.io
    kubeconfig: "{{ kubeconfig_path }}"

# --- Step 2: Install cert-manager via Helm ---
- name: Install cert-manager via Helm
  kubernetes.core.helm:
    name: cert-manager
    chart_ref: jetstack/cert-manager
    chart_version: "{{ cert_manager_version }}"    # v1.17.2
    release_namespace: cert-manager
    create_namespace: true
    kubeconfig: "{{ kubeconfig_path }}"
    values:
      crds:
        enabled: true          # Install CRDs (Certificate, Issuer, ClusterIssuer)
      replicaCount: 1          # Single replica (sufficient for single-node)
      resources:
        requests:
          cpu: 25m
          memory: 64Mi
        limits:
          cpu: 100m
          memory: 128Mi
    wait: true                 # Block until all pods are Running
    wait_timeout: 300s         # 5 minute timeout

# --- Step 3: Wait for the webhook to be ready ---
# cert-manager's webhook validates Certificate resources. If it is not ready,
# creating a ClusterIssuer will fail with a webhook timeout error.
- name: Wait for cert-manager webhook to be ready
  kubernetes.core.k8s_info:
    api_version: v1
    kind: Pod
    namespace: cert-manager
    label_selectors:
      - app.kubernetes.io/component=webhook
    kubeconfig: "{{ kubeconfig_path }}"
  register: webhook_pods
  retries: 30
  delay: 10
  until: >-
    webhook_pods.resources | length > 0 and
    webhook_pods.resources[0].status.phase == 'Running'

# --- Step 4: Create the ClusterIssuer ---
- name: Create letsencrypt-prod ClusterIssuer
  kubernetes.core.k8s:
    kubeconfig: "{{ kubeconfig_path }}"
    state: present
    definition:
      apiVersion: cert-manager.io/v1
      kind: ClusterIssuer
      metadata:
        name: letsencrypt-prod
      spec:
        acme:
          server: https://acme-v02.api.letsencrypt.org/directory
          email: "{{ cert_issuer_email }}"     # admin@freezenith.com
          privateKeySecretRef:
            name: letsencrypt-prod-key
          solvers:
            - http01:
                ingress:
                  ingressClassName: traefik     # Use Traefik for HTTP-01 challenges
```

**How HTTP-01 challenges work:**

```
1. You create a Certificate resource for "api.stage.freezenith.com"

2. cert-manager contacts Let's Encrypt:
   "I want a cert for api.stage.freezenith.com"

3. Let's Encrypt responds:
   "Prove you control that domain. Serve this token at
    http://api.stage.freezenith.com/.well-known/acme-challenge/TOKEN"

4. cert-manager creates a temporary Ingress + Pod that serves the token

5. Let's Encrypt fetches the URL and verifies the token

6. Let's Encrypt issues the certificate

7. cert-manager stores it as a Kubernetes Secret (type: kubernetes.io/tls)

8. Traefik reads the Secret and uses the certificate for HTTPS
```

This is why Cloudflare proxy must be OFF with HTTP-01 challenges -- Cloudflare would
intercept the HTTP request at step 5. The V2 plan switches to DNS-01 challenges
(documented in Phase 1) which avoids this limitation.

---

## Variables Reference

### Shared Defaults (`group_vars/all.yml`)

| Variable | Value | Purpose |
|----------|-------|---------|
| `k3s_version` | `v1.34.3+k3s1` | Pinned k3s version for reproducible installs |
| `kubeconfig_path` | `/etc/rancher/k3s/k3s.yaml` | Where k3s writes the kubeconfig |
| `cilium_version` | `"1.16.5"` | Pinned Cilium version |
| `enable_cilium` | `false` | Default OFF; staging overrides to `true` |
| `cert_manager_version` | `v1.17.2` | Pinned cert-manager Helm chart version |
| `cert_issuer_email` | `admin@freezenith.com` | Email for Let's Encrypt registration |
| `postgres_image` | `postgres:16-alpine` | PostgreSQL container image |
| `postgres_db` | `zenith` | Database name |
| `postgres_user` | `zenith` | Database user |
| `postgres_storage_size` | `5Gi` | PVC size for PostgreSQL data |
| `image_tag` | `latest` | Docker image tag for all Zenith images |
| `image_pull_policy` | `Never` | Use locally imported images (no registry) |

### Staging Inventory (`inventory/staging.yml`)

| Variable | Value | Purpose |
|----------|-------|---------|
| `ansible_host` | `77.42.88.149` | Server IP from Phase 1 |
| `ansible_user` | `root` | SSH user |
| `env_name` | `staging` | Environment identifier |
| `base_domain` | `stage.freezenith.com` | Root domain for staging |
| `api_domain` | `api.stage.freezenith.com` | API endpoint |
| `mc_domain` | `ms.stage.freezenith.com` | Mission Control endpoint |
| `web_domain` | `cloud.stage.freezenith.com` | Web Platform endpoint |
| `platform_namespace` | `zenith-staging` | Kubernetes namespace for platform |
| `enable_cilium` | `true` | Cilium enabled in staging |
| `enable_keda` | `false` | No autoscaling in staging |
| `enable_monitoring` | `false` | No Prometheus/Grafana in staging |
| `platform_tls_secret` | `staging-tls` | Name of the TLS Secret |

### Staging Overrides (`group_vars/staging.yml`)

Staging uses lower resource limits because the server is smaller:

| Resource | all.yml (default) | staging.yml (override) |
|----------|-------------------|----------------------|
| API CPU request | 50m | 25m |
| API memory request | 64Mi | 32Mi |
| API CPU limit | 250m | 150m |
| API memory limit | 256Mi | 128Mi |
| PostgreSQL CPU limit | 500m | 250m |
| PostgreSQL memory limit | 512Mi | 256Mi |

---

## How to Run

### Prerequisites

1. **Phase 1 completed**: A Hetzner server exists with a known IP address
2. **Ansible >= 2.15** installed locally: `pip install ansible`
3. **Ansible collections** installed:
   ```bash
   cd infra/ansible
   ansible-galaxy collection install -r requirements.yml
   ```
   This installs `kubernetes.core`, `community.general`, and `ansible.posix` which are
   used by the roles for Helm, modprobe, and sysctl tasks.
4. **SSH access** to the server using the key from Phase 1

### Step-by-step: Infrastructure Only (Phase 2)

```bash
# 1. Navigate to the Ansible directory
cd infra/ansible

# 2. Update the inventory with the server IP from Phase 1
#    Edit inventory/staging.yml and set:
#      ansible_host: <server_ip from terraform output>

# 3. Verify SSH connectivity
ansible -i inventory/staging.yml all -m ping

# Expected output:
# zenith-staging | SUCCESS => {
#     "ping": "pong"
# }

# 4. Run ONLY the infrastructure roles (Phase 2)
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags infra

# This runs these roles in order:
#   1. common     -- apt packages, Docker, Helm, kernel config
#   2. k3s        -- install k3s, wait for ready, kubeconfig
#   3. cilium     -- install Cilium CLI, deploy Cilium to cluster
#   4. cert-manager -- Helm install cert-manager, create ClusterIssuer
#   5. postgres   -- deploy PostgreSQL StatefulSet + headless Service

# 5. Verify the cluster is running
ssh root@<server_ip> "k3s kubectl get nodes"
ssh root@<server_ip> "k3s kubectl get pods -A"
```

### Step-by-step: Full Deployment (Phases 2-5 in one command)

```bash
# Run the complete site playbook (infrastructure + build + deploy)
ansible-playbook playbooks/site.yml -i inventory/staging.yml
```

### Running Specific Tags

Tags let you run subsets of the playbook. This is essential for day-2 operations:

```bash
# Just k3s and Cilium (skip everything else)
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags k3s,cilium

# Just cert-manager (after fixing a certificate issue)
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags cert-manager

# Infrastructure only (no app builds or deploys)
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags infra

# Build images and deploy apps (skip infrastructure)
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags build,deploy
```

### Available Tags Reference

| Tag | Roles Executed | When to Use |
|-----|---------------|-------------|
| `common` | common | Updating base packages, adding new apt packages |
| `k3s` | k3s | Upgrading k3s version, debugging k3s issues |
| `cilium` | cilium | Upgrading Cilium, changing Cilium configuration |
| `cert-manager` | cert-manager | TLS certificate issues, upgrading cert-manager |
| `infra` | common, k3s, cilium, cert-manager, postgres | Full infrastructure refresh |
| `postgres` | postgres | Database issues, storage changes |
| `build` | zenith-build, zenith-import | Rebuilding container images after code changes |
| `deploy` | namespaces, traefik, api, landing, mc, web, ingress | Redeploying applications |
| `monitoring` | monitoring (Prometheus, Grafana, Loki) | Setting up observability stack |
| `keda` | keda | Setting up autoscaling |

### Expected Output

```
PLAY [Deploy Zenith Platform] ************************************************

TASK [Display deployment info] ************************************************
ok: [zenith-staging] => {
    "msg": "Deploying Zenith to staging\nHost: 77.42.88.149\n..."
}

TASK [common : Update apt cache] **********************************************
ok: [zenith-staging]

TASK [common : Install base packages] *****************************************
ok: [zenith-staging]

TASK [common : Install Docker (if not present)] *******************************
skipping: [zenith-staging]    <-- already installed from previous run

TASK [common : Disable swap] **************************************************
ok: [zenith-staging]

TASK [k3s : Check if k3s is installed] ****************************************
ok: [zenith-staging]

TASK [k3s : Install k3s (with Cilium CNI)] ************************************
skipping: [zenith-staging]    <-- already installed

TASK [k3s : Wait for k3s to be ready] *****************************************
ok: [zenith-staging]

TASK [cilium : Check if Cilium CLI is installed] ******************************
ok: [zenith-staging]

TASK [cilium : Install Cilium via CLI] ****************************************
skipping: [zenith-staging]    <-- already installed

TASK [cilium : Wait for Cilium to be ready] ***********************************
ok: [zenith-staging]

TASK [cert-manager : Install cert-manager via Helm] ***************************
ok: [zenith-staging]

TASK [cert-manager : Create letsencrypt-prod ClusterIssuer] *******************
ok: [zenith-staging]

PLAY RECAP ********************************************************************
zenith-staging : ok=18  changed=0  unreachable=0  failed=0  skipped=5
```

Notice the `skipping` entries -- this is Ansible's idempotency in action. Components
already installed from a previous run are detected and skipped. The playbook is safe to
run repeatedly.

---

## V2 Enhancements

These features are planned for the V2 platform but not yet implemented in the Ansible
roles. They are documented here so you understand the full target architecture.

### 1. hcloud-csi -- Hetzner Volume StorageClass

**What**: A CSI (Container Storage Interface) driver that dynamically provisions Hetzner
Cloud Volumes when a PersistentVolumeClaim is created.

**Why**: k3s ships with the `local-path-provisioner` StorageClass, which creates PVCs on
the server's local disk. This has a critical limitation: **if the server is destroyed, the
data is gone**. For PostgreSQL and other stateful workloads, we need storage that survives
server replacement.

hcloud-csi creates Hetzner Volumes, which are network-attached SSDs that exist
independently of any server. You can detach a Volume from a destroyed server and attach it
to a new one, preserving all data.

**How it works:**

```
1. A PVC is created:
   apiVersion: v1
   kind: PersistentVolumeClaim
   metadata:
     name: postgres-data
   spec:
     storageClassName: hcloud-volumes    # <-- triggers hcloud-csi
     accessModes: [ReadWriteOnce]
     resources:
       requests:
         storage: 10Gi

2. hcloud-csi controller sees the PVC and calls Hetzner API:
   POST /v1/volumes { "size": 10, "location": "hel1", "format": "ext4" }

3. Hetzner creates a 10 GB SSD Volume and returns its ID

4. hcloud-csi attaches the Volume to the server (appears as /dev/sdb)

5. hcloud-csi formats it with ext4 and mounts it into the Pod

6. If the server is destroyed, the Volume persists in Hetzner Cloud
   You can attach it to a new server and mount the same data
```

**Comparison with local-path:**

| Storage | Persistence | Performance | Cost | Use Case |
|---------|-------------|-------------|------|----------|
| `local-path` (k3s default) | Lost if server dies | Fast (local NVMe) | Free (included in VM disk) | Ephemeral data, caches, build artifacts |
| `hcloud-volumes` | Survives server destruction | Good (network SSD) | EUR 0.052/GB/month | Databases, persistent state, backups |

**Implementation plan** (new Ansible role `hcloud-csi`):

```yaml
# roles/hcloud-csi/tasks/main.yml (planned)

- name: Create hcloud-csi secret
  kubernetes.core.k8s:
    definition:
      apiVersion: v1
      kind: Secret
      metadata:
        name: hcloud-csi
        namespace: kube-system
      stringData:
        token: "{{ hcloud_token }}"    # Hetzner API token

- name: Install hcloud-csi via Helm
  kubernetes.core.helm:
    name: hcloud-csi
    chart_ref: hcloud/hcloud-csi-driver
    release_namespace: kube-system
    values:
      storageClasses:
        - name: hcloud-volumes
          defaultStorageClass: false    # Keep local-path as default
          reclaimPolicy: Delete
          volumeBindingMode: WaitForFirstConsumer
          allowVolumeExpansion: true
```

### 2. etcd Encryption at Rest

**What**: The `--secrets-encryption` flag for k3s enables AES-CBC encryption for all
Kubernetes Secrets stored in the embedded etcd database.

**Why**: Without this flag, Kubernetes Secrets are stored in etcd as base64-encoded
plaintext. Base64 is **not encryption** -- it is just encoding. Anyone with access to the
etcd data files (on disk at `/var/lib/rancher/k3s/server/db/`) can decode every Secret.

```
Without --secrets-encryption:
    etcd stores:  {"data":{"password":"bXlwYXNz"}}
    echo "bXlwYXNz" | base64 -d  -->  mypass

With --secrets-encryption:
    etcd stores:  <AES-CBC encrypted blob>
    Cannot be read without the encryption key managed by k3s
```

**What it protects against:**

- Physical disk theft (data center compromise at Hetzner)
- Unauthorized access to etcd data files on the server
- Backup exposure (etcd snapshots contain encrypted data)

**What it does NOT protect against:**

- A user with `kubectl get secret` RBAC access (they see decrypted values via the API --
  encryption at rest only protects the on-disk format)
- Root access on the server (the encryption key is stored on disk at
  `/var/lib/rancher/k3s/server/cred/encryption-config.json`)

**Implementation** (add to k3s install flags):

```bash
INSTALL_K3S_EXEC="--flannel-backend=none --disable-network-policy --disable=servicelb --secrets-encryption"
```

### 3. Hubble Observability

**What**: Hubble is Cilium's built-in observability platform. It provides real-time
visibility into network flows, DNS queries, HTTP requests, and NetworkPolicy enforcement
across the cluster.

**Why**: When debugging networking issues ("why can't pod A reach pod B?"), you currently
have to look at Cilium agent logs, manually check NetworkPolicies, and use `tcpdump`. Hubble
replaces all of that with a single dashboard that shows every network flow, whether it was
allowed or denied, and why.

**Components:**

```
+-- hubble-relay (Deployment)
|   Aggregates flow data from all cilium-agent instances.
|   Exposes gRPC API for querying flows.
|
+-- hubble-ui (Deployment)
|   Web dashboard for visualizing network flows.
|   Shows service map, flow table, and policy verdicts.
|   Exposed behind auth via Traefik IngressRoute.
|
+-- hubble CLI (installed on operator laptop)
    Command-line tool for querying flows:
    $ hubble observe --namespace zenith-staging --verdict DROPPED
    $ hubble observe --to-pod zenith-postgres --protocol TCP --port 5432
```

**Implementation** (add to Cilium install flags):

```bash
cilium install \
  --version 1.16.5 \
  --set kubeProxyReplacement=true \
  --set k8sServiceHost=127.0.0.1 \
  --set k8sServicePort=6443 \
  --set hubble.enabled=true \
  --set hubble.relay.enabled=true \
  --set hubble.ui.enabled=true
```

### 4. WireGuard Pod-to-Pod Encryption

**What**: Transparent encryption of all pod-to-pod traffic using WireGuard, a modern
cryptographic VPN protocol built into the Linux kernel (5.6+).

**Why**: By default, pod-to-pod traffic is sent in plaintext over the node's network
interface. On a shared hosting platform (where Hetzner neighbors could theoretically sniff
traffic), or in multi-node clusters (where inter-node traffic crosses the physical network),
encrypting pod-to-pod traffic is a defense-in-depth measure.

**How it works in Cilium:**

```
Node A                                            Node B
+--------------------+                            +--------------------+
| Pod: zenith-api    |                            | Pod: postgres      |
| 10.42.0.15:8080    |                            | 10.42.0.22:5432    |
+--------+-----------+                            +--------+-----------+
         |                                                 ^
         v                                                 |
+--------+-----------+                            +--------+-----------+
| Cilium Agent       |                            | Cilium Agent       |
| (eBPF datapath)    |                            | (eBPF datapath)    |
|                    |                            |                    |
| WireGuard Tunnel   |============================| WireGuard Tunnel   |
| (ChaCha20-Poly1305 encrypted)                                       |
+--------------------+                            +--------------------+

- Encryption is transparent to pods (no code changes, no TLS needed)
- Each node has a WireGuard keypair (auto-generated by Cilium)
- Keys are stored as Kubernetes Secrets and auto-rotated
- Overhead: ~5% throughput reduction, ~0.1ms additional latency per hop
- WireGuard is faster than IPsec (~20% overhead) and has a smaller attack surface
  (4,000 lines of code vs 400,000 for IPsec)
```

**Implementation** (add to Cilium install flags):

```bash
cilium install \
  --version 1.16.5 \
  --set kubeProxyReplacement=true \
  --set k8sServiceHost=127.0.0.1 \
  --set k8sServicePort=6443 \
  --set encryption.enabled=true \
  --set encryption.type=wireguard
```

**Why WireGuard over IPsec?** WireGuard uses ChaCha20-Poly1305 (same cipher used by
HTTPS/TLS 1.3). It has a ~5% throughput overhead compared to IPsec's ~20%. WireGuard's
codebase is ~4,000 lines of C (auditable) versus IPsec's ~400,000 lines. WireGuard has
been in the Linux kernel since 5.6 and requires no additional kernel modules.

---

## Verification Checklist

After Phase 2 completes, run these commands to verify everything is working:

### 1. Node Status

```bash
ssh root@<server_ip> "k3s kubectl get nodes"

# Expected:
# NAME              STATUS   ROLES                  AGE   VERSION
# zenith-staging    Ready    control-plane,master   5m    v1.34.3+k3s1

# "Ready" means kubelet is running and the node is accepting workloads.
# If it says "NotReady", Cilium may not be ready yet. Wait 60 seconds.
```

### 2. System Pods

```bash
ssh root@<server_ip> "k3s kubectl get pods -A"

# Expected (with Cilium enabled):
# NAMESPACE      NAME                                      READY   STATUS
# kube-system    cilium-xxxxx                              1/1     Running
# kube-system    cilium-operator-xxxxx                     1/1     Running
# kube-system    coredns-xxxxx                             1/1     Running
# kube-system    local-path-provisioner-xxxxx              1/1     Running
# kube-system    metrics-server-xxxxx                      1/1     Running
# kube-system    traefik-xxxxx                             1/1     Running
# cert-manager   cert-manager-xxxxx                        1/1     Running
# cert-manager   cert-manager-cainjector-xxxxx             1/1     Running
# cert-manager   cert-manager-webhook-xxxxx                1/1     Running

# All pods should be Running. If any show CrashLoopBackOff or Pending,
# check the Troubleshooting section.
```

### 3. Cilium Status

```bash
ssh root@<server_ip> "cilium status"

# Expected:
#     /\         Cilium:          OK
#    /\ \        Operator:        OK
#   /\  /\       Hubble Relay:    disabled    <-- (enabled in V2)
#  / \/  \ \
# |        |    KubeProxyReplacement:   True
# |        |    Encryption:             Disabled  <-- (enabled in V2)
```

### 4. Cilium Connectivity Test

```bash
ssh root@<server_ip> "cilium connectivity test"

# This runs a suite of network connectivity tests (takes 2-5 minutes):
# - Pod-to-pod communication
# - Pod-to-Service communication
# - Pod-to-external communication
# - NetworkPolicy enforcement
# All tests should pass.
```

### 5. cert-manager Status

```bash
ssh root@<server_ip> "k3s kubectl get clusterissuer"

# Expected:
# NAME               READY   AGE
# letsencrypt-prod   True    5m

# "True" means cert-manager successfully registered with Let's Encrypt.
# If "False", check cert-manager logs:
# k3s kubectl logs -n cert-manager -l app.kubernetes.io/name=cert-manager
```

### 6. StorageClass

```bash
ssh root@<server_ip> "k3s kubectl get storageclass"

# Expected:
# NAME                   PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE
# local-path (default)   rancher.io/local-path   Delete          WaitForFirstConsumer
# [V2] hcloud-volumes    csi.hetzner.cloud       Delete          WaitForFirstConsumer
```

### 7. Docker

```bash
ssh root@<server_ip> "docker --version && docker info | head -5"

# Expected:
# Docker version 27.x.x, build xxxxxxx
# Containers: 0
# Running: 0
# ...
```

### 8. Kernel Configuration

```bash
ssh root@<server_ip> "swapon --show && sysctl net.ipv4.ip_forward net.bridge.bridge-nf-call-iptables"

# Expected:
# (no output from swapon -- swap is off)
# net.ipv4.ip_forward = 1
# net.bridge.bridge-nf-call-iptables = 1
```

---

## Troubleshooting

### Ansible cannot connect to the server

```bash
# Test raw SSH first (bypass Ansible)
ssh -i ~/.ssh/id_ed25519 root@<server_ip>

# If SSH works but Ansible doesn't, debug with triple-verbose:
ansible -i inventory/staging.yml all -m ping -vvv

# Common causes:
# 1. Wrong SSH key path in inventory (ansible_ssh_private_key_file)
# 2. Hetzner firewall blocking port 22 (check Phase 1 Terraform)
# 3. Host key changed (new server reusing old IP):
ssh-keygen -R <server_ip>
```

### k3s installation hangs or fails

```bash
# SSH to the server and check k3s service logs
ssh root@<server_ip>
journalctl -u k3s -f --no-pager | tail -50

# Common causes:
# 1. Insufficient memory (k3s needs ~512 MB free)
free -h

# 2. DNS resolution failing on the server
cat /etc/resolv.conf
nslookup github.com

# 3. Outbound traffic blocked (k3s downloads container images on first start)
curl -sfL https://get.k3s.io | head -1
```

### k3s installed but node shows NotReady

```bash
# This usually means the CNI is not ready.
# If enable_cilium=true, k3s starts WITHOUT a CNI (Flannel is disabled).
# The node stays NotReady until Cilium is installed.

# Check if Cilium pods exist:
k3s kubectl get pods -n kube-system -l app.kubernetes.io/name=cilium-agent

# If no pods: Cilium role hasn't run yet. Run:
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags cilium

# If pods exist but CrashLoopBackOff: check Cilium logs
k3s kubectl logs -n kube-system -l app.kubernetes.io/name=cilium-agent --tail=50
```

### Cilium pods stuck in CrashLoopBackOff

```bash
# Check Cilium agent logs for the actual error
k3s kubectl logs -n kube-system -l app.kubernetes.io/name=cilium-agent --tail=50

# Common causes:

# 1. k3s was installed WITHOUT --flannel-backend=none
#    Both Flannel and Cilium try to own networking --> conflict
#    Fix: Uninstall k3s and reinstall with correct flags
/usr/local/bin/k3s-uninstall.sh
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags k3s,cilium

# 2. Kernel too old for eBPF
uname -r
# Needs >= 5.4 for full eBPF support. Ubuntu 24.04 ships with 6.x, so unlikely.

# 3. Missing kernel modules
lsmod | grep -E 'cilium|vxlan|wireguard'
```

### cert-manager ClusterIssuer shows READY=False

```bash
# Check cert-manager controller logs
k3s kubectl logs -n cert-manager -l app.kubernetes.io/name=cert-manager --tail=50

# Common causes:
# 1. cert-manager webhook not ready (race condition)
#    Fix: Wait 60 seconds and re-run
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags cert-manager

# 2. Cannot reach Let's Encrypt ACME endpoint
#    Fix: Check outbound connectivity
curl -s https://acme-v02.api.letsencrypt.org/directory | jq .
```

### Cannot reach k3s API from local machine

```bash
# Check if port 6443 is reachable
nc -zv <server_ip> 6443

# If not reachable: Hetzner firewall is blocking it
# Check ssh_allowed_ips in your Terraform config (Phase 1)

# If reachable but kubectl fails with TLS error:
kubectl --kubeconfig ~/.kube/zenith-staging.yaml get nodes
# Error: x509: certificate is valid for 127.0.0.1, not <server_ip>

# Fix: The kubeconfig server URL must be patched from 127.0.0.1 to the server IP
# Check your kubeconfig:
grep "server:" ~/.kube/zenith-staging.yaml
# Should show: server: https://<server_ip>:6443
# NOT: server: https://127.0.0.1:6443

# Temporary workaround (not for production):
# Set insecure-skip-tls-verify: true in the kubeconfig

# Proper fix: install k3s with --tls-san=<server_ip> (V2 enhancement)
```

### Pods cannot pull images (ErrImagePull)

```bash
# k3s uses containerd, NOT Docker. Images built with `docker build` are not
# automatically visible to k3s. They must be imported:
docker save zenith-api:latest | k3s ctr images import -

# Verify the image is available to k3s:
k3s ctr images list | grep zenith

# Also verify the Deployment uses imagePullPolicy: Never
k3s kubectl get deploy zenith-api -o jsonpath='{.spec.template.spec.containers[0].imagePullPolicy}'
# Should output: Never
```

### Full reinstall (nuclear option)

```bash
# On the server:
/usr/local/bin/k3s-uninstall.sh      # Removes k3s, all pods, all data
rm -rf /var/lib/rancher/k3s           # Clean up residual data
rm -rf /etc/rancher/k3s               # Remove kubeconfig and certificates

# Then re-run Ansible from scratch:
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags infra
```

---

## What Happens Next

After Phase 2 completes, you have:

- A running k3s cluster with a single Ready node
- Cilium CNI providing pod networking and NetworkPolicy enforcement
- cert-manager with a `letsencrypt-prod` ClusterIssuer ready to issue TLS certificates
- Traefik ingress controller listening on ports 80 and 443
- CoreDNS, Metrics Server, and Local Path Provisioner as built-in k3s addons
- Docker installed for building container images on the server
- A kubeconfig file for remote `kubectl` access

The cluster is **empty** -- no application pods, no custom namespaces, no TLS certificates
for your domains. Phase 3 takes this bare cluster and bootstraps it with everything
needed to run the Zenith platform:

- **Phase 3**: Cluster bootstrap -- namespaces, TLS certificates, PostgreSQL, Traefik
  configuration, image builds, application deployments, and ingress routes

**Previous**: [Phase 1: Hetzner + Cloudflare](./01-phase1-hetzner-cloudflare.md)
**Next**: [Phase 3: Cluster Bootstrap](./03-phase3-cluster-bootstrap.md)
