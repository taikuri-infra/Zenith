# Zenith Ansible Deployment

Idempotent deployment for staging and production. Same playbooks, different inventory.

## Prerequisites

```bash
pip install ansible
ansible-galaxy collection install -r requirements.yml
```

## Quick Start

```bash
# Full deploy to production
ansible-playbook playbooks/site.yml -i inventory/production.yml

# Full deploy to staging
ansible-playbook playbooks/site.yml -i inventory/staging.yml

# Build + deploy apps only (skip infra)
ansible-playbook playbooks/apps.yml -i inventory/production.yml

# Build images only (no deploy)
ansible-playbook playbooks/build.yml -i inventory/production.yml

# Infrastructure only (no apps)
ansible-playbook playbooks/infra.yml -i inventory/production.yml

# Deploy specific components via tags
ansible-playbook playbooks/site.yml -i inventory/production.yml --tags api
ansible-playbook playbooks/site.yml -i inventory/production.yml --tags build,deploy
ansible-playbook playbooks/site.yml -i inventory/production.yml --tags keda,monitoring

# Teardown (interactive confirmation)
ansible-playbook playbooks/teardown.yml -i inventory/staging.yml
```

## Secrets (Ansible Vault)

```bash
# Create vault file
ansible-vault create vault/production.yml
ansible-vault create vault/staging.yml

# Use vault in playbooks
ansible-playbook playbooks/site.yml -i inventory/production.yml --extra-vars @vault/production.yml --ask-vault-pass
```

Vault file format:
```yaml
vault_jwt_secret: "<random-64-char-hex>"
vault_admin_email: "admin@embermind.app"
vault_admin_password: "<strong-password>"
vault_db_password: "<strong-password>"
vault_cloudflare_token: "<cloudflare-api-token>"
```

## Tags Reference

| Tag | What it does |
|-----|-------------|
| `common` | Base server setup (apt, Docker, sysctl) |
| `k3s` | Install/verify k3s |
| `cert-manager` | cert-manager + Let's Encrypt ClusterIssuer |
| `traefik` | Traefik redirect middlewares |
| `postgres` | PostgreSQL StatefulSet |
| `keda` | KEDA + HTTP Add-on + cold-start page |
| `monitoring` | Prometheus + Grafana + Loki + Promtail |
| `dns` | Cloudflare DNS via Terraform |
| `build` | Docker image builds |
| `import` | Import images into k3s |
| `deploy` | All app deployments |
| `api` | API server only |
| `landing` | Landing page only |
| `mc` | Mission Control (demo + customer) |
| `web` | Web Platform (demo + customer) |
| `ingress` | IngressRoutes + TLS certificates |
| `demo` | Demo deployments only |
| `customers` | Customer tenant deployments only |
| `restart` | Restart deployments (force rollout) |

## Directory Structure

```
ansible/
├── ansible.cfg              # Ansible configuration
├── requirements.yml         # Galaxy collections
├── inventory/
│   ├── staging.yml          # Staging host + vars
│   └── production.yml       # Production host + vars
├── group_vars/
│   ├── all.yml              # Shared defaults
│   ├── staging.yml          # Staging overrides
│   └── production.yml       # Production overrides
├── vault/                   # Encrypted secrets (per env)
├── playbooks/
│   ├── site.yml             # Full deployment
│   ├── infra.yml            # Infrastructure only
│   ├── apps.yml             # Applications only
│   ├── build.yml            # Build images only
│   └── teardown.yml         # Remove resources
└── roles/
    ├── common/              # Base server setup
    ├── k3s/                 # k3s installation
    ├── cert-manager/        # TLS certificate management
    ├── traefik-config/      # Ingress middlewares
    ├── postgres/            # PostgreSQL database
    ├── keda/                # Scale-to-zero
    ├── monitoring/          # Prometheus + Grafana + Loki
    ├── dns/                 # Cloudflare DNS
    ├── zenith-build/        # Docker image builds
    ├── zenith-import/       # Import to k3s containerd
    ├── zenith-namespaces/   # K8s namespaces + secrets
    ├── zenith-api/          # API deployment
    ├── zenith-landing/      # Landing page
    ├── zenith-mc/           # Mission Control
    ├── zenith-web/          # Web Platform
    └── zenith-ingress/      # IngressRoutes + certs
```
