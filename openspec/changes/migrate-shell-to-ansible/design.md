# Design: Migrate Shell Scripts to Ansible

## Context

Zenith currently deploys via `infra/scripts/deploy.sh` — a 260-line bash script that builds Docker images, imports them into k3s, applies raw K8s manifests, and restarts deployments. Infrastructure components (KEDA, cert-manager) have separate shell installers. Secrets are created manually with `kubectl create secret`. There is no staging environment, no environment separation, and no way to deploy consistently across multiple servers.

The project needs staging/production parity, automated infra provisioning, and a path to install all the missing SaaS components (KEDA, Harbor, CloudNativePG, Keycloak, monitoring stack).

## Goals
- Same playbooks deploy to staging AND production (different inventory)
- Idempotent — safe to run repeatedly
- Modular roles — deploy individual components independently
- Secrets encrypted in repo via Ansible Vault
- Support both fresh install and incremental updates
- Tags for partial runs (just build, just deploy apps, just infra)

## Non-Goals
- Replacing docker-compose for local dev (stays as-is)
- Multi-cloud support (Hetzner only for now)
- Replacing Terraform for DNS (Ansible orchestrates Terraform)
- GitOps via FluxCD (separate future change)

## Directory Structure

```
infra/ansible/
├── ansible.cfg                    # Ansible configuration
├── requirements.yml               # Galaxy collections
├── inventory/
│   ├── staging.yml                # Staging host(s) + vars
│   └── production.yml             # Production host(s) + vars
├── group_vars/
│   ├── all.yml                    # Shared defaults
│   ├── staging.yml                # Staging overrides
│   └── production.yml             # Production overrides
├── vault/
│   ├── staging.yml                # Encrypted staging secrets
│   └── production.yml             # Encrypted production secrets
├── playbooks/
│   ├── site.yml                   # Full deployment (infra + apps)
│   ├── infra.yml                  # Infrastructure only
│   ├── apps.yml                   # Build + deploy Zenith apps only
│   ├── build.yml                  # Build images only (no deploy)
│   └── teardown.yml               # Remove everything (careful!)
└── roles/
    ├── common/                    # Base: apt packages, firewall, swap
    ├── k3s/                       # k3s install/upgrade
    ├── cert-manager/              # cert-manager + ClusterIssuer
    ├── traefik-config/            # Traefik middlewares, TLS options
    ├── postgres/                  # PostgreSQL StatefulSet + migrations
    ├── keda/                      # KEDA + HTTP Add-on + cold-start page
    ├── harbor/                    # Container registry (Helm)
    ├── cloudnativepg/             # CloudNativePG operator (Helm)
    ├── keycloak/                  # Keycloak IAM (Helm)
    ├── monitoring/                # kube-prometheus-stack + Loki (Helm)
    ├── dns/                       # Run Terraform for Cloudflare DNS
    ├── zenith-build/              # Docker build all images
    ├── zenith-import/             # docker save | k3s ctr images import
    ├── zenith-namespaces/         # Create K8s namespaces + secrets
    ├── zenith-api/                # API deployment + service
    ├── zenith-landing/            # Landing page deployment
    ├── zenith-mc/                 # Mission Control (real + demo)
    ├── zenith-web/                # Web Platform (real + demo)
    └── zenith-ingress/            # IngressRoutes + certificates
```

## Decisions

### 1. Ansible over Terraform for K8s resources
**Decision:** Use Ansible `kubernetes.core.k8s` module for K8s manifests, keep Terraform only for DNS.
**Rationale:** Terraform is great for cloud infrastructure (DNS, servers) but awkward for K8s resources that change frequently (deployments, services). Ansible handles both server provisioning and K8s resource management in one tool. The existing raw K8s YAML files can be used directly as Jinja2 templates.

### 2. Ansible Vault for secrets
**Decision:** Store encrypted secrets in `infra/ansible/vault/{env}.yml`, decrypt at runtime with `--ask-vault-pass` or vault password file.
**Rationale:** Secrets currently created manually via `kubectl create secret` with random values. Vault keeps secrets version-controlled but encrypted. Each environment has its own vault file.

### 3. Reuse existing K8s manifests as templates
**Decision:** Copy `infra/k8s/*.yaml` into Ansible role templates with Jinja2 variables for environment-specific values (image tags, replica counts, resource limits, domains).
**Rationale:** The existing manifests work. No need to rewrite them — just parameterize the values that differ between environments.

### 4. Helm for infrastructure components, raw YAML for Zenith apps
**Decision:** Use `kubernetes.core.helm` for third-party components (KEDA, cert-manager, Harbor, CloudNativePG, Keycloak, monitoring). Use `kubernetes.core.k8s` with templated YAML for Zenith's own deployments.
**Rationale:** Infrastructure operators (KEDA, CloudNativePG) are best managed via their official Helm charts — version pinned, values-configurable. Zenith's own deployments are simple enough that raw YAML with Jinja2 is cleaner than maintaining a Helm chart.

### 5. Tags for partial runs
**Decision:** Every role gets tags: `infra`, `apps`, `build`, `deploy`, `keda`, `monitoring`, etc.
**Usage:**
```bash
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags build,deploy
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags keda
ansible-playbook playbooks/site.yml -i inventory/production.yml  # full run
```

### 6. Rolling deployment strategy
**Decision:** Build images first, import into k3s, then restart deployments one-by-one with rollout status checks.
**Rationale:** Same strategy as current `deploy.sh` but with proper error handling and the ability to stop on failure.

## Environment Separation

```yaml
# inventory/staging.yml
all:
  hosts:
    staging-server:
      ansible_host: <staging-ip>
      ansible_user: root
  vars:
    env_name: staging
    domain: staging.freezenith.com
    zenith_namespaces:
      - zenith-platform
    enable_demo: true
    enable_keda: false        # optional in staging
    enable_monitoring: false   # optional in staging

# inventory/production.yml
all:
  hosts:
    production-server:
      ansible_host: 161.35.82.211
      ansible_user: root
  vars:
    env_name: production
    domain: freezenith.com
    zenith_namespaces:
      - zenith-platform
      - zenith-embermind
    enable_demo: true
    enable_keda: true
    enable_monitoring: true
    customer_domains:
      - name: embermind
        mc_domain: ms.embermind.app
        web_domain: cloud.embermind.app
```

## Migration Plan

1. **Phase 1:** Create Ansible structure + common/k3s/cert-manager roles. Test on staging.
2. **Phase 2:** Migrate Zenith app deployment (build, import, deploy, ingress). Test on staging.
3. **Phase 3:** Add infra roles (KEDA, PostgreSQL, monitoring). Test on staging.
4. **Phase 4:** Add vault secrets. Deploy to production via Ansible.
5. **Phase 5:** Deprecate `infra/scripts/deploy.sh`. Add future infra roles (Harbor, CloudNativePG, Keycloak).

After Phase 4, `deploy.sh` is no longer used. Deployment becomes:
```bash
ansible-playbook infra/ansible/playbooks/site.yml -i infra/ansible/inventory/production.yml --ask-vault-pass
```

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Learning curve for team | Ansible is widely known; playbooks are readable YAML |
| Migration period where both shell + Ansible exist | Short — Phase 1-2 can be done in a day |
| Ansible controller needs to be installed on dev machine | `pip install ansible` or `brew install ansible` |
| Vault password management | Use `--vault-password-file` with `.vault_pass` in `.gitignore` |
| k3s containerd import is k3s-specific | Encapsulated in `zenith-import` role; easy to swap for registry push later |

## Open Questions
- Should we set up a dedicated staging server now, or reuse the existing server with a separate namespace?
- Should Harbor be part of this migration or a separate future change?
