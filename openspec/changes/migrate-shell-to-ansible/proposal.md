# Change: Migrate Shell Scripts to Ansible

## Why
The platform is deployed via ad-hoc shell scripts (`deploy.sh`, `install.sh`, `cloudflare-dns.sh`, `k8s/keda/install.sh`) with hardcoded secrets and no environment separation. This makes staging/production parity impossible, infra component installation manual, and secrets management insecure. Ansible provides idempotent, environment-aware deployment with vault-encrypted secrets — same playbooks for staging and production, different inventory.

## What Changes
- Replace `scripts/deploy.sh` with Ansible playbooks and roles
- Replace `scripts/cloudflare-dns.sh` with Ansible DNS role (or keep Terraform)
- Replace `k8s/keda/install.sh` with Ansible KEDA role
- Add roles for all infrastructure components: k3s, cert-manager, PostgreSQL, KEDA, Harbor, CloudNativePG, Keycloak, monitoring
- Add roles for all Zenith app deployments: API, Web, MC, Landing, Demo
- Add inventory files for staging and production environments
- Add Ansible Vault for secrets management (JWT secrets, DB passwords, API tokens)
- Add image build + import playbook (replaces Docker build/save/import steps)
- Keep `docker-compose.yml` and `scripts/install.sh` for local standalone mode (unaffected)
- Keep Terraform for Cloudflare DNS (Ansible orchestrates Terraform, doesn't replace it)

## Impact
- Affected specs: deploy-engine (deployment process changes)
- Affected code: `scripts/deploy.sh` (replaced), `scripts/cloudflare-dns.sh` (replaced by Terraform orchestration), `k8s/keda/install.sh` (replaced)
- New directory: `ansible/` at repo root
- New dependencies: Ansible >= 2.15, `kubernetes.core` collection, `community.general` collection
- No breaking API or application changes — deployment mechanism changes, not behavior
- Existing `k8s/` manifests reused as templates within Ansible roles
- Existing `helm/` charts can be adopted by Ansible roles via `kubernetes.core.helm`
