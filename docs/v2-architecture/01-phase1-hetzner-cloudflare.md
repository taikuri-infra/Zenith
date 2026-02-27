# Phase 1: Hetzner + Cloudflare (Terraform)

> **Zenith V2 Platform Architecture -- Phase 1 of 5**
>
> This phase creates the foundational cloud resources that everything else depends on:
> a Hetzner VM and Cloudflare DNS records. It is the ONLY phase that provisions
> cloud resources. Phases 2-5 operate entirely ON the server created here.

---

## Table of Contents

1. [Why This Phase Exists](#why-this-phase-exists)
2. [What Gets Created](#what-gets-created)
3. [Architecture Overview](#architecture-overview)
4. [Terraform Resource Graph](#terraform-resource-graph)
5. [Module Breakdown](#module-breakdown)
6. [Variables Reference](#variables-reference)
7. [How to Run](#how-to-run)
8. [Outputs](#outputs)
9. [Security Model](#security-model)
10. [Staging vs Production](#staging-vs-production)
11. [DNS Strategy and Cloudflare Proxy](#dns-strategy-and-cloudflare-proxy)
12. [Troubleshooting](#troubleshooting)

---

## Why This Phase Exists

Before we can install Kubernetes, deploy containers, or configure TLS certificates, we
need two things:

1. **A server to run on** -- Hetzner Cloud provides cost-effective VMs with predictable
   pricing. A cx32 (4 vCPU / 8GB RAM) costs roughly EUR 7.50/month, which is an order of
   magnitude cheaper than equivalent AWS/GCP instances.

2. **DNS records pointing to that server** -- Every service (landing page, API, Mission
   Control, Web Platform) needs a hostname. Cloudflare manages our DNS zones and provides
   DDoS protection, CDN caching, and a global anycast network at no additional cost.

The reason these two concerns are bundled into a single phase is that they form a
**dependency pair**: DNS records need the server's IP address, and the server needs to
exist before DNS can point to it. Terraform handles this dependency graph naturally
through resource references.

### Why Terraform (not Ansible, not scripts)?

Terraform is the right tool for **cloud resource lifecycle management** because:

- **Declarative state**: Terraform tracks what exists. If someone deletes a DNS record
  manually, `terraform apply` recreates it. If the server is destroyed, Terraform knows.
- **Plan before apply**: `terraform plan` shows exactly what will change before any
  mutation occurs. This is critical for production infrastructure.
- **Provider ecosystem**: The `hetznercloud/hcloud` and `cloudflare/cloudflare` providers
  are first-class, well-maintained, and handle API idempotency correctly.
- **Dependency ordering**: Terraform builds a directed acyclic graph (DAG) of resources
  and creates/destroys them in the correct order automatically.

Ansible is the wrong tool for cloud provisioning (it has no state file, so it cannot
detect drift or handle destroys cleanly). We use Ansible in Phase 2 for what it excels
at: server configuration management.

---

## What Gets Created

### Hetzner Resources

| Resource | Purpose | Module |
|----------|---------|--------|
| `hcloud_ssh_key` | SSH public key uploaded to Hetzner for server access | `k3s-server` |
| `hcloud_firewall` | Inbound rules: 22 (SSH), 80 (HTTP), 443 (HTTPS), 6443 (k3s API), 10250 (kubelet) | `k3s-server` |
| `hcloud_server` | The VM itself (Ubuntu 24.04, cx23/cx32 depending on env) | `k3s-server` |

### Cloudflare Resources

| Resource | Purpose | Module |
|----------|---------|--------|
| `cloudflare_record` (platform) | A records for platform subdomains (api, ms, cloud, etc.) | `dns` |
| `cloudflare_record` (customer) | A records for customer subdomains (per-tenant) | `dns` |

### What is NOT created here

- No Kubernetes resources (Phase 2: Ansible)
- No TLS certificates (Phase 2: cert-manager via Ansible)
- No application deployments (Phase 3+)
- No Hetzner Volumes (managed by hcloud-csi at runtime)
- No Object Storage buckets (production only, optional)

---

## Architecture Overview

```
                         +-------------------+
                         |   Cloudflare CDN  |
                         |  (DNS + Proxy)    |
                         +--------+----------+
                                  |
                    DNS A Records |  (all subdomains)
                                  |
                         +--------v----------+
                         |  Hetzner Cloud    |
                         |  Firewall         |
                         |  +--------------+ |
                         |  | Ports Open:  | |
                         |  | 22   (SSH)   | |
                         |  | 80   (HTTP)  | |
                         |  | 443  (HTTPS) | |
                         |  | 6443 (k3s)*  | |
                         |  | 10250(kube)* | |
                         |  +--------------+ |
                         +--------+----------+
                                  |  * = restricted to ssh_allowed_ips
                                  |
                         +--------v----------+
                         |  hcloud_server    |
                         |  (Ubuntu 24.04)   |
                         |                   |
                         |  Name: zenith-    |
                         |   staging / prod  |
                         |                   |
                         |  SSH key auth     |
                         |  (no passwords)   |
                         +-------------------+
                                  |
                                  | Public IPv4
                                  |
                    +-------------+-------------+
                    |                           |
          Cloudflare A records           terraform output
          point here                     "server_ip"
```

### DNS Topology (Staging)

```
freezenith.com (Cloudflare Zone)
    |
    +-- stage.freezenith.com          --> Server IP (landing)
    +-- api.stage.freezenith.com      --> Server IP (API)
    +-- ms.stage.freezenith.com       --> Server IP (Mission Control)
    +-- cloud.stage.freezenith.com    --> Server IP (Web Platform)
    +-- grafana.stage.freezenith.com  --> Server IP (monitoring)
    +-- prometheus.stage.freezenith.com --> Server IP (monitoring)
    +-- embermind-ms.stage.freezenith.com  --> Server IP (customer MC)
    +-- embermind.stage.freezenith.com     --> Server IP (customer Web)
```

### DNS Topology (Production)

```
freezenith.com (Cloudflare Zone)         embermind.app (Cloudflare Zone)
    |                                        |
    +-- freezenith.com (root)                +-- ms.embermind.app
    +-- www.freezenith.com                   +-- cloud.embermind.app
    +-- api.freezenith.com
    +-- demo-ms.freezenith.com
    +-- demo-cloud.freezenith.com
```

---

## Terraform Resource Graph

This is the DAG that Terraform builds internally. Resources at the bottom depend on
resources above them.

```
terraform (providers: hcloud, cloudflare)
    |
    +-- provider "hcloud"
    |       |
    |       +-- hcloud_ssh_key.zenith
    |       |       |
    |       |       v
    |       +-- hcloud_firewall.zenith
    |       |       |
    |       |       v
    |       +-- hcloud_server.zenith --------+
    |               |                        |
    |               | (output: server_ip)    |
    |               v                        |
    +-- provider "cloudflare"                |
            |                                |
            +-- cloudflare_record.platform --+  (uses server_ip)
            |       |
            |       +-- "stage" (root)
            |       +-- "api.stage"
            |       +-- "ms.stage"
            |       +-- "cloud.stage"
            |       +-- "grafana.stage"
            |       +-- "prometheus.stage"
            |
            +-- cloudflare_record.customer
                    |
                    +-- "embermind-ms.stage"
                    +-- "embermind.stage"
```

Key insight: the DNS records depend on the server's IP address. If `create_server = true`,
the IP comes from `module.staging_server[0].server_ip`. If `create_server = false` (using
an existing server), it comes from `var.existing_server_ip`. This conditional is handled
in the staging `main.tf`:

```hcl
server_ip = var.create_server ? module.staging_server[0].server_ip : var.existing_server_ip
```

---

## Module Breakdown

### Module: `k3s-server`

**Path**: `infra/terraform/modules/k3s-server/`

This module encapsulates all Hetzner resources needed for a single server. It is reused
for staging (one server, role "all-in-one") and production (two servers: "management" +
"cluster").

```hcl
# infra/terraform/modules/k3s-server/main.tf

# SSH key -- uploaded to Hetzner so the server can be accessed
resource "hcloud_ssh_key" "zenith" {
  name       = "${var.name}-ssh-key"
  public_key = var.ssh_public_key

  lifecycle {
    ignore_changes = [public_key]
    # Why: If you rotate your SSH key locally, you don't want Terraform to
    # destroy and recreate the server. Key rotation is handled out-of-band.
  }
}

# Firewall -- defense in depth at the cloud provider level
resource "hcloud_firewall" "zenith" {
  name = "${var.name}-firewall"

  # SSH: restricted to ssh_allowed_ips (default: all, but you should lock this down)
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "22"
    source_ips = var.ssh_allowed_ips
  }

  # HTTP: open to all (needed for ACME HTTP-01 challenges and redirects)
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "80"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  # HTTPS: open to all (serves all application traffic)
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "443"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  # k3s API: restricted (only operators need kubectl access)
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "6443"
    source_ips = var.ssh_allowed_ips
  }

  # kubelet: restricted (for monitoring and debugging)
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "10250"
    source_ips = var.ssh_allowed_ips
  }
}

# The server itself
resource "hcloud_server" "zenith" {
  name        = var.name
  server_type = var.server_type   # cx23, cx32, cx42, etc.
  image       = var.image         # ubuntu-24.04
  location    = var.location      # hel1, nbg1, fsn1
  ssh_keys    = [hcloud_ssh_key.zenith.id]
  firewall_ids = [hcloud_firewall.zenith.id]

  labels = merge({
    "zenith.dev/managed-by"  = "terraform"
    "zenith.dev/environment" = var.environment
    "zenith.dev/role"        = var.role
  }, var.extra_labels)

  user_data = var.user_data  # Optional cloud-init script
}
```

**Why these specific firewall ports?**

| Port | Protocol | Open To | Reason |
|------|----------|---------|--------|
| 22 | TCP | `ssh_allowed_ips` | SSH access for Ansible and operator access |
| 80 | TCP | Everyone | HTTP-to-HTTPS redirects, ACME HTTP-01 challenges |
| 443 | TCP | Everyone | All application HTTPS traffic via Traefik |
| 6443 | TCP | `ssh_allowed_ips` | Kubernetes API server (kubectl from your laptop) |
| 10250 | TCP | `ssh_allowed_ips` | Kubelet API (metrics, exec, logs) |

Ports 6443 and 10250 are restricted to `ssh_allowed_ips` because exposing the Kubernetes
API to the internet is a serious security risk. In production, you should set this to your
office/VPN IP ranges.

### Module: `dns`

**Path**: `infra/terraform/modules/dns/`

This module creates Cloudflare A records for both platform services and customer tenants.
It uses `for_each` to iterate over a map of records, making it easy to add or remove
subdomains.

```hcl
# infra/terraform/modules/dns/main.tf

# Platform records: all subdomains under a single zone (e.g., freezenith.com)
resource "cloudflare_record" "platform" {
  for_each = var.platform_records

  zone_id = var.zone_id
  name    = each.value.name
  content = var.server_ip
  type    = "A"
  ttl     = 1        # 1 = "automatic" in Cloudflare (uses their default)
  proxied = false     # See "DNS Strategy" section below for why
}

# Customer records: may span multiple Cloudflare zones
resource "cloudflare_record" "customer" {
  for_each = var.customer_records

  zone_id = each.value.zone_id   # Each customer may have their own zone
  name    = each.value.name
  content = var.server_ip
  type    = "A"
  ttl     = 1
  proxied = false
}
```

**Why `for_each` instead of individual resources?**

The `for_each` pattern keeps the module DRY. Adding a new subdomain is a one-line change
in the calling module:

```hcl
platform_records = {
  root    = { name = "stage" }
  api     = { name = "api.stage" }
  ms      = { name = "ms.stage" }
  cloud   = { name = "cloud.stage" }
  grafana = { name = "grafana.stage" }  # <-- just add a line
}
```

Compare this to the old approach (still in `infra/terraform/dns.tf`) where each record
was a separate `resource` block -- seven resources for seven subdomains, each with
duplicated boilerplate.

---

## Variables Reference

### Staging (`infra/terraform/staging/variables.tf`)

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `cloudflare_api_token` | `string` (sensitive) | -- | Cloudflare API token with DNS:Edit permission |
| `hcloud_token` | `string` (sensitive) | -- | Hetzner Cloud API token |
| `create_server` | `bool` | `true` | Whether to create a new Hetzner server |
| `existing_server_ip` | `string` | `""` | IP of existing server (when `create_server = false`) |
| `server_type` | `string` | `"cx23"` | Hetzner server type (cx23 = 2 vCPU / 4GB) |
| `hetzner_location` | `string` | `"hel1"` | Hetzner datacenter (hel1 = Helsinki) |
| `ssh_public_key` | `string` | -- | Contents of your `~/.ssh/id_ed25519.pub` |
| `ssh_allowed_ips` | `list(string)` | `["0.0.0.0/0", "::/0"]` | IPs allowed for SSH and k3s API |
| `freezenith_zone_id` | `string` | `"37ac..."` | Cloudflare zone ID for freezenith.com |

### Production (`infra/terraform/production/variables.tf`)

Production adds these variables (in addition to the above):

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `management_server_type` | `string` | `"cx32"` | Server type for management plane (4 vCPU / 8GB) |
| `cluster_server_type` | `string` | `"cx32"` | Server type for customer workload cluster |
| `customer_domains` | `map(object)` | embermind defaults | Customer DNS records with per-tenant zone IDs |
| `hetzner_s3_access_key` | `string` (sensitive) | `""` | Object Storage access key |
| `hetzner_s3_secret_key` | `string` (sensitive) | `""` | Object Storage secret key |

### Hetzner Server Types (Quick Reference)

| Type | vCPU | RAM | Disk | Price/mo | Use Case |
|------|------|-----|------|----------|----------|
| `cx22` | 2 | 4 GB | 40 GB | ~EUR 4 | Development |
| `cx32` | 4 | 8 GB | 80 GB | ~EUR 7.50 | Staging, small production |
| `cx42` | 8 | 16 GB | 160 GB | ~EUR 15 | Medium production |
| `cx52` | 16 | 32 GB | 320 GB | ~EUR 30 | Large production |
| `ccx33` | 4 | 16 GB | 80 GB | ~EUR 15 | Dedicated (no noisy neighbors) |

For staging, `cx22` or `cx32` is sufficient. For production, start with `cx32` and
scale up to `cx42` or dedicated (`ccx*`) when traffic demands it.

### terraform.tfvars.example

```hcl
# Copy to terraform.tfvars and fill in values

cloudflare_api_token = ""
freezenith_zone_id   = "37ac5735b1cf9099ccedd4e038d99465"

# Option A: Use an existing server (no Hetzner token needed)
create_server      = false
existing_server_ip = "161.35.82.211"

# Option B: Create a new Hetzner server
# create_server    = true
# hcloud_token     = ""
# server_type      = "cx23"
# hetzner_location = "hel1"
# ssh_public_key   = "ssh-ed25519 AAAA..."

# S3 backend credentials (if using remote state):
# export AWS_ACCESS_KEY_ID="..."
# export AWS_SECRET_ACCESS_KEY="..."
```

---

## How to Run

### Prerequisites

1. **Terraform >= 1.5** installed locally
2. A **Hetzner Cloud** account with an API token (Project > Security > API Tokens)
3. A **Cloudflare** account with an API token (Profile > API Tokens > Create Token)
   - Permission needed: `Zone:DNS:Edit` for the relevant zones
4. An **SSH key pair** (Ed25519 recommended): `ssh-keygen -t ed25519`

### Step-by-step

```bash
# 1. Navigate to the staging Terraform directory
cd infra/terraform/staging

# 2. Create your tfvars file from the example
cp terraform.tfvars.example terraform.tfvars

# 3. Edit terraform.tfvars with your actual values
#    - Set cloudflare_api_token
#    - Set hcloud_token (if creating a new server)
#    - Set ssh_public_key (if creating a new server)
#    - Choose Option A (existing server) or Option B (new server)

# 4. Initialize Terraform (downloads providers)
terraform init

# 5. Preview what will be created
terraform plan

# 6. Apply (creates resources)
terraform apply

# 7. Note the outputs -- you'll need server_ip for Phase 2
terraform output
```

### Expected Output

```
Apply complete! Resources: 9 added, 0 changed, 0 destroyed.

Outputs:

server_ip = "5.161.xxx.xxx"

dns_records = {
  "api"        = "api.stage.freezenith.com"
  "cloud"      = "cloud.stage.freezenith.com"
  "grafana"    = "grafana.stage.freezenith.com"
  "ms"         = "ms.stage.freezenith.com"
  "prometheus" = "prometheus.stage.freezenith.com"
  "root"       = "stage.freezenith.com"
}

ansible_inventory_hint = "ansible_host: 5.161.xxx.xxx"
```

The `ansible_inventory_hint` output tells you exactly what to put in your Ansible
inventory for Phase 2.

### Destroying Resources

```bash
# Preview what will be destroyed
terraform plan -destroy

# Destroy everything (server + DNS records)
terraform destroy
```

**Warning**: `terraform destroy` will delete the Hetzner server and all its data.
Persistent data should be backed up before destroying.

---

## Outputs

| Output | Description | Used By |
|--------|-------------|---------|
| `server_ip` | Public IPv4 address of the created server | Ansible inventory (Phase 2) |
| `dns_records` | Map of all created DNS hostnames | Verification, documentation |
| `ansible_inventory_hint` | Ready-to-paste line for Ansible inventory YAML | Phase 2 setup |

---

## Security Model

### Defense in Depth

The security model for Phase 1 is layered:

```
Layer 1: Cloudflare
    |-- DDoS protection (when proxy is ON)
    |-- Rate limiting (configurable)
    |-- WAF rules (configurable)
    |-- Hides origin IP from public DNS lookups
    |
Layer 2: Hetzner Firewall
    |-- Blocks all ports except 22, 80, 443, 6443, 10250
    |-- SSH and k3s API restricted to ssh_allowed_ips
    |-- Applied at the hypervisor level (not iptables -- cannot be bypassed from inside the VM)
    |
Layer 3: SSH Key Authentication
    |-- No password authentication
    |-- Ed25519 keys only (uploaded via hcloud_ssh_key)
    |-- Root access via SSH (locked down in Phase 2 with Ansible)
```

### Principle of Least Privilege

- **Cloudflare API token**: Scoped to `Zone:DNS:Edit` for specific zones only. It cannot
  modify WAF rules, SSL settings, or other zone configurations.
- **Hetzner API token**: Project-scoped. It can only manage resources within the Zenith
  project, not other Hetzner projects.
- **SSH**: Key-based only. The `hcloud_server` resource uses `ssh_keys` which configures
  the server to reject password authentication from first boot.
- **Firewall**: Management ports (22, 6443, 10250) are restricted to `ssh_allowed_ips`.
  In production, this should be your office IP or VPN CIDR, not `0.0.0.0/0`.

### Secret Management

Sensitive values are stored in `terraform.tfvars` which is **gitignored** (see
`infra/terraform/staging/.gitignore`). The file is never committed to version control.

```
# .gitignore
*.tfvars
!terraform.tfvars.example
.terraform/
```

For team environments, consider Terraform remote state with encryption (e.g., Hetzner S3
backend, which is stubbed out in the staging `main.tf` as a TODO).

---

## Staging vs Production

### Staging Architecture

```
+---------------------------+
|  Single Hetzner Server    |
|  (cx23: 2 vCPU / 4 GB)   |
|                           |
|  Role: all-in-one         |
|  Name: zenith-staging     |
|                           |
|  Runs EVERYTHING:         |
|  - k3s control plane      |
|  - Traefik ingress        |
|  - Platform apps          |
|  - Customer apps          |
|  - PostgreSQL             |
+---------------------------+
```

### Production Architecture

```
+---------------------------+     +---------------------------+
|  Management Server        |     |  Cluster Server           |
|  (cx32: 4 vCPU / 8 GB)   |     |  (cx32: 4 vCPU / 8 GB)   |
|                           |     |                           |
|  Role: management         |     |  Role: cluster            |
|  Name: zenith-prod-mgmt   |     |  Name: zenith-prod-clstr  |
|                           |     |                           |
|  Runs:                    |     |  Runs:                    |
|  - Zenith API             |     |  - Customer workloads     |
|  - Mission Control        |     |  - Customer databases     |
|  - Landing Page           |     |  - KEDA autoscaling       |
|  - PostgreSQL             |     |  - Monitoring stack       |
|  - CAPI controller        |     |                           |
+---------------------------+     +---------------------------+
         |                                   |
         +---- DNS points to management -----+
               (management is the front door)
```

The key difference: staging uses a single server with `create_server` toggle and
`existing_server_ip` fallback. Production always creates two servers with no toggle --
separation of management and workload planes is mandatory in production.

---

## DNS Strategy and Cloudflare Proxy

### Current State: Proxy OFF (`proxied = false`)

The current Terraform configuration sets `proxied = false` on all DNS records. This means:

- DNS resolves directly to the Hetzner server's IP address
- No Cloudflare CDN or DDoS protection
- cert-manager HTTP-01 challenges work without issues (Cloudflare is not intercepting
  traffic)

This was the pragmatic choice for V1 because HTTP-01 ACME challenges require the
challenge server to be reachable on port 80, and Cloudflare's proxy intercepts port 80
traffic.

### V2 Target: Proxy ON with DNS-01 Challenges

In V2, we switch cert-manager from HTTP-01 to DNS-01 solver. This decouples TLS
certificate issuance from HTTP traffic entirely:

```
# V1 (current) -- HTTP-01 solver
# cert-manager proves domain ownership by serving a token on port 80
# Cloudflare proxy MUST be OFF (it would intercept the challenge)
solvers:
  - http01:
      ingress:
        ingressClassName: traefik

# V2 (target) -- DNS-01 solver
# cert-manager proves domain ownership by creating a TXT record in Cloudflare
# Cloudflare proxy can be ON (HTTP traffic is irrelevant to the challenge)
solvers:
  - dns01:
      cloudflare:
        apiTokenSecretRef:
          name: cloudflare-api-token
          key: api-token
```

With DNS-01, we can safely turn Cloudflare proxy ON for all records:

```hcl
resource "cloudflare_record" "platform" {
  for_each = var.platform_records

  zone_id = var.zone_id
  name    = each.value.name
  content = var.server_ip
  type    = "A"
  ttl     = 1
  proxied = true   # <-- Safe with DNS-01 challenges
}
```

**Benefits of Proxy ON:**

1. **DDoS protection**: Cloudflare absorbs volumetric attacks before they reach Hetzner
2. **Origin IP hidden**: `dig stage.freezenith.com` returns Cloudflare IPs, not the server
3. **CDN caching**: Static assets (JS, CSS, images) served from Cloudflare edge
4. **HTTP/3 and QUIC**: Automatic protocol upgrades at the Cloudflare edge
5. **Rate limiting**: Configurable per-endpoint rate limits at the CDN layer

**Trade-off**: Cloudflare proxy adds ~10-50ms latency for non-cached requests (traffic
routes through Cloudflare's nearest PoP). For an API with p99 latency targets, measure
this before enabling.

---

## Troubleshooting

### "Error: unauthorized" from Hetzner provider

Your `hcloud_token` is invalid or expired. Generate a new one from the Hetzner Cloud
Console under Project > Security > API Tokens.

### "Error: Invalid API token" from Cloudflare provider

Your `cloudflare_api_token` needs `Zone:DNS:Edit` permission for the specific zones.
Verify the token scope at Cloudflare Dashboard > Profile > API Tokens.

### "Error: SSH key already exists"

The `hcloud_ssh_key` resource has `lifecycle { ignore_changes = [public_key] }`. If you
need to update the key, either:
- Remove the old key from the Hetzner Console and run `terraform apply`
- Or use `terraform taint module.staging_server[0].hcloud_ssh_key.zenith` to force
  recreation

### DNS records not resolving

After `terraform apply`, DNS propagation takes 30 seconds to 5 minutes depending on the
TTL and your local DNS cache. Verify with:

```bash
# Check Cloudflare directly (bypasses local cache)
dig @1.1.1.1 stage.freezenith.com A

# Check from Google DNS
dig @8.8.8.8 stage.freezenith.com A
```

### Server created but cannot SSH

1. Check the firewall allows your IP on port 22:
   ```bash
   terraform state show 'module.staging_server[0].hcloud_firewall.zenith'
   ```
2. Verify your SSH key matches what was uploaded:
   ```bash
   terraform state show 'module.staging_server[0].hcloud_ssh_key.zenith'
   ```
3. Try connecting with verbose output:
   ```bash
   ssh -vvv root@$(terraform output -raw server_ip)
   ```

### State lock / concurrent access

The staging backend is currently local (file-based). Do not run `terraform apply`
concurrently from two terminals. For team environments, enable the S3 backend (commented
out in `main.tf`):

```hcl
backend "s3" {
  bucket    = "zenithstage"
  key       = "staging/terraform.tfstate"
  endpoints = { s3 = "https://hel1.your-objectstorage.com" }
  region    = "main"
  # ... Hetzner S3 compatibility flags
}
```

---

## What Happens Next

After Phase 1 completes, you have:

- A running Hetzner VM with a public IP address
- DNS records pointing all subdomains to that IP
- SSH key access configured
- Firewall rules applied at the hypervisor level

The server is a **blank Ubuntu 24.04 machine**. It has no Kubernetes, no Docker, no
application code. Phase 2 (Ansible + k3s) takes this bare server and turns it into a
functioning Kubernetes cluster.

**Proceed to**: [Phase 2: Ansible + k3s](./02-phase2-ansible-k3s.md)
