# Phase 2: Platform Bootstrap — Deploy Zenith to Staging k3s

## Summary

Deploy the complete Zenith platform on the staging k3s cluster (`77.42.88.149`) using Terraform Phase 3. All components running, accessible via HTTPS, with persistent PostgreSQL.

## Prerequisites

- Phase 1 complete (images + charts in Harbor)
- Staging server running k3s (done)
- Harbor registry running (done)

## Steps

### Step 2.1: Fetch & Configure Kubeconfig

**What:** Get kubeconfig from staging server, patch server URL.

**Build:**
- Run `ansible-playbook playbooks/server-setup.yml` OR manual `scp`
- Kubeconfig saved to `~/.kube/zenith-staging.yaml`

**Your manual work:**
- Verify SSH access to `77.42.88.149`

**Verify:**
```bash
kubectl --kubeconfig ~/.kube/zenith-staging.yaml get nodes
```

### Step 2.2: Create Terraform tfvars for staging-k8s

**What:** Configure all variables for the k8s-platform module.

**Build:**
- `infra/terraform/staging-k8s/terraform.tfvars` with:
  - `kubeconfig_path`
  - `zenith_chart_version`
  - Harbor registry credentials
  - JWT secret, admin email/password, DB password
  - Feature flags (keda=false, monitoring=false for now)

**Your manual work:**
1. Generate a JWT secret: `openssl rand -hex 32`
2. Choose admin email + password for staging
3. Choose DB password for PostgreSQL
4. Harbor pull credentials (from Phase 1 Step 2)

**Verify:**
```bash
cd infra/terraform/staging-k8s && terraform validate
```

### Step 2.3: Terraform Apply — Deploy Platform

**What:** Install cert-manager + ClusterIssuer + Zenith Helm chart into k3s.

**Build:**
- `terraform init && terraform apply` in `staging-k8s/`
- Creates: namespace, cert-manager, ClusterIssuer, PostgreSQL, API, Landing, MC, Web, Demo apps, Secrets, Ingress, Certificates

**Your manual work:**
- Review `terraform plan` output before apply
- Ensure Cloudflare DNS is pointing to `77.42.88.149` (done in Phase 1)

**Verify:**
```bash
kubectl --kubeconfig ~/.kube/zenith-staging.yaml get pods -n zenith-staging
# All pods Running

curl -s https://stage.freezenith.com | head -5
# Landing page HTML

curl -s https://api.stage.freezenith.com/health
# {"status":"ok"}
```

### Step 2.4: Verify All Endpoints

**What:** Confirm all staging URLs serve correct content.

**Verify:**
| URL | Expected |
|---|---|
| https://stage.freezenith.com | Landing page |
| https://api.stage.freezenith.com/health | `{"status":"ok"}` |
| https://ms.stage.freezenith.com | MC (login page or demo) |
| https://cloud.stage.freezenith.com | Web Platform (login page or demo) |

## Acceptance Criteria

- [ ] All pods running in `zenith-staging` namespace
- [ ] PostgreSQL pod with persistent volume
- [ ] cert-manager issued TLS certificates for all domains
- [ ] Landing page accessible at `stage.freezenith.com`
- [ ] API health check returns 200
- [ ] Terraform state tracks all resources
- [ ] `terraform plan` shows no changes after apply
