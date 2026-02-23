# Infrastructure Pipeline V1 — From MVP to Versioned SaaS

## Summary

Transition Zenith from manual `deploy.sh` on a single server to a fully versioned, CI/CD-driven infrastructure pipeline using Harbor (OCI registry), Terraform (state-tracked Helm deployments), and GitHub Actions (automated builds). Everything becomes Infrastructure as Code — reproducible, versionable, and promotable from staging to production.

## Context

### What We Have (Done)

| Component | Status | Details |
|---|---|---|
| Staging server | Running | `77.42.88.149`, k3s + Docker + Helm (Hetzner, cx23, hel1) |
| Harbor registry | Running | `registry.stage.freezenith.com` (65.108.210.253) |
| Harbor projects | Created | `zenith-stage`, `zenith`, `fairbroker-stage`, `fairbroker`, `library` |
| Terraform Phase 1 | Done | Server + Cloudflare DNS provisioning |
| Ansible Phase 2 | Done | Server config (packages, Docker, k3s, Helm) |
| Terraform Phase 3 module | Created | `k8s-platform` module (Helm releases — not yet applied) |
| Production server (ghasi) | Running | `161.35.82.211`, k3s, manual deploy via `deploy.sh` |
| Harbor infra project | Done | Separate repo `taikuri-infra/harbor-registry` |
| All app code | Working | Landing, MC, Web, API, Operator — live on ghasi |
| Dockerfiles | Exist | 5 Dockerfiles (landing, mc, web, api, operator) |
| Helm chart | Exists | `infra/helm/zenith/` with staging + default values |

### What's Missing

1. No CI/CD pipeline — images are built manually on the server
2. No images in Harbor — k3s uses `imagePullPolicy: Never` with local import
3. Terraform Phase 3 not applied — k8s resources not state-tracked
4. No versioning strategy — everything is `:latest`
5. No promotion workflow — staging and production are disconnected

### Architecture Decision

**Terraform manages ALL state.** No FluxCD. Upgrade = bump version in tfvars → `terraform apply`.

**Harbor is the single source of truth** for all Docker images and Helm charts.

**CI/CD builds and pushes.** GitHub Actions (runnable locally with `act`) build, test, and push artifacts.

### SaaS Pricing Tiers

| Tier | Infra Model | Isolation |
|---|---|---|
| **Free** | Namespace in shared cluster | Cilium network policies, ResourceQuota |
| **Pro** (€29/mo) | Namespace in shared cluster | Cilium L7 + bandwidth limits |
| **Team** | CAPI cluster on shared machines | Separate control plane + etcd |
| **Enterprise** | CAPI cluster on dedicated machines | Full kernel-level isolation |

### Harbor Project Strategy

| Project | Purpose |
|---|---|
| `zenith-stage` | Push here, test here. All CI builds go here. |
| `zenith` | Verified versions only. Promote from zenith-stage. |
| `fairbroker-stage` | Fairbroker staging builds. |
| `fairbroker` | Fairbroker verified versions. |

---

## Steps

### Step 1: Docker Images — Build & Push to Harbor

**Goal:** All 5 Zenith Docker images built via CI and pushed to Harbor with semantic version tags.

**What to build:**
- `.github/workflows/build-images.yml` — GitHub Action that builds all 5 images
- Each image tagged as: `registry.stage.freezenith.com/zenith-stage/<app>:<version>`
- Version format: `v0.2.0` (from `Chart.yaml` appVersion), plus `sha-<short>` for traceability
- Demo images (mc-demo, web-demo) built with `--build-arg NEXT_PUBLIC_DEMO_MODE=true`

**Images to build:**
| Image | Dockerfile | Context |
|---|---|---|
| `zenith-api` | `services/api/Dockerfile` | repo root |
| `zenith-landing` | `apps/landing/Dockerfile` | repo root |
| `zenith-mc` | `apps/mission-control/Dockerfile` | repo root |
| `zenith-web` | `apps/web/Dockerfile` | repo root |
| `zenith-operator` | `services/operator/Dockerfile` | repo root |
| `zenith-mc-demo` | `apps/mission-control/Dockerfile` + `NEXT_PUBLIC_DEMO_MODE=true` | repo root |
| `zenith-web-demo` | `apps/web/Dockerfile` + `NEXT_PUBLIC_DEMO_MODE=true` | repo root |

**Manual steps (you do this):**
1. Create a Harbor robot account for CI (Harbor UI → `zenith-stage` project → Robot Accounts → New)
   - Name: `ci-push`
   - Permissions: Push, Pull
   - Save the token — you'll need it for GitHub secrets
2. Add GitHub repo secrets (Settings → Secrets → Actions):
   - `HARBOR_HOST` = `registry.stage.freezenith.com`
   - `HARBOR_ROBOT_USER` = `robot$zenith-stage+ci-push` (Harbor generates this)
   - `HARBOR_ROBOT_TOKEN` = (the token from step 1)

**How to verify:**
```bash
# Run locally with act
act -j build-images --secret-file .secrets

# Check Harbor UI — images should appear in zenith-stage project
# Or via CLI:
docker login registry.stage.freezenith.com
docker pull registry.stage.freezenith.com/zenith-stage/zenith-api:v0.2.0
```

---

### Step 2: imagePullSecret in k8s

**Goal:** k3s staging cluster can pull images from private Harbor registry.

**What to build:**
- Add `kubernetes_secret` resource to Terraform `staging-k8s/main.tf` for `harbor-registry-creds`
- Type: `kubernetes.io/dockerconfigjson`
- Created in every namespace that needs it (`zenith-staging`, `zenith-platform`)

**Manual steps (you do this):**
1. Create a Harbor robot account for k8s pull (separate from CI push):
   - Name: `k8s-pull`
   - Permissions: Pull only
   - Save credentials
2. Add to `staging-k8s/terraform.tfvars`:
   - `harbor_pull_user` = `robot$zenith-stage+k8s-pull`
   - `harbor_pull_token` = (the token)

**How to verify:**
```bash
# After terraform apply:
kubectl --kubeconfig ~/.kube/zenith-staging.yaml get secret harbor-registry-creds -n zenith-staging
# Should show type: kubernetes.io/dockerconfigjson
```

---

### Step 3: Helm Chart — Package & Push to Harbor

**Goal:** Zenith Helm chart versioned and stored in Harbor as OCI artifact.

**What to build:**
- `.github/workflows/build-chart.yml` — GitHub Action that packages and pushes Helm chart
- Chart version bumped in `Chart.yaml` (follows semver)
- Pushed to: `oci://registry.stage.freezenith.com/zenith-stage/zenith`

**What to update:**
- `infra/helm/zenith/Chart.yaml` — set `appVersion` to match image version
- `infra/helm/zenith/values-staging.yaml` — change `imagePullPolicy: Never` to `IfNotPresent`, add image registry prefix and `imagePullSecrets`

**Manual steps (you do this):**
- Same robot account from Step 1 can push Helm charts (OCI uses same credentials)

**How to verify:**
```bash
# Run locally with act
act -j build-chart --secret-file .secrets

# Pull chart from Harbor
helm pull oci://registry.stage.freezenith.com/zenith-stage/zenith --version 0.2.0

# Or install directly (dry-run)
helm template test oci://registry.stage.freezenith.com/zenith-stage/zenith --version 0.2.0
```

---

### Step 4: Update Terraform Phase 3 — Deploy from Harbor OCI

**Goal:** Terraform `staging-k8s` deploys Helm chart from Harbor instead of local path.

**What to build:**
- Update `infra/terraform/modules/k8s-platform/main.tf`:
  - `helm_release.zenith` → `repository = "oci://registry.stage.freezenith.com/zenith-stage"`, `chart = "zenith"`, `version = var.zenith_chart_version`
  - Add `repository_username` and `repository_password` for OCI auth
- Update `infra/terraform/staging-k8s/main.tf`:
  - Add `zenith_chart_version` variable
  - Pass Harbor credentials
- Update `infra/terraform/staging-k8s/terraform.tfvars`:
  - `zenith_chart_version = "0.2.0"`

**Manual steps (you do this):**
1. Get the kubeconfig from the staging server:
   ```bash
   ansible-playbook playbooks/server-setup.yml -i inventory/staging.yml
   # Or manually: scp root@77.42.88.149:/etc/rancher/k3s/k3s.yaml ~/.kube/zenith-staging.yaml
   ```
2. Create `staging-k8s/terraform.tfvars` with:
   - `kubeconfig_path` = `~/.kube/zenith-staging.yaml`
   - `zenith_chart_version` = `0.2.0`
   - Harbor credentials
   - Secrets (JWT, admin password, DB password)

**How to verify:**
```bash
cd infra/terraform/staging-k8s
terraform init
terraform plan    # Should show helm_release resources
terraform apply   # Deploys everything

# Check:
kubectl --kubeconfig ~/.kube/zenith-staging.yaml get pods -n zenith-staging
kubectl --kubeconfig ~/.kube/zenith-staging.yaml get helmrelease -A  # Not applicable (no Flux)
helm --kubeconfig ~/.kube/zenith-staging.yaml list -n zenith-staging
```

---

### Step 5: Terraform CI — Plan on PR, Apply on Merge

**Goal:** Terraform changes are validated on PRs and auto-applied on merge.

**What to build:**
- `.github/workflows/terraform.yml` — GitHub Action
  - On PR: `terraform init` + `terraform plan` → output as PR comment
  - On merge to main: `terraform apply -auto-approve` (with approval gate for production)
- Separate jobs for each Terraform workspace:
  - `staging` (Phase 1 — server + DNS)
  - `staging-k8s` (Phase 3 — Helm releases)

**Manual steps (you do this):**
1. Add GitHub repo secrets:
   - `HCLOUD_TOKEN` = Hetzner Cloud API token
   - `CLOUDFLARE_API_TOKEN` = Cloudflare API token
   - `KUBECONFIG_STAGING` = base64-encoded kubeconfig content
   - `TF_VAR_jwt_secret` = JWT secret for the platform
   - `TF_VAR_admin_email` = admin email
   - `TF_VAR_admin_password` = admin password
   - `TF_VAR_db_password` = database password

**How to verify:**
```bash
# Run locally with act
act -j terraform-plan --secret-file .secrets

# Or just run Terraform directly
cd infra/terraform/staging-k8s && terraform plan
```

---

### Step 6: Version Tagging Strategy

**Goal:** Clear versioning scheme for the entire platform.

**What to build:**
- Version lives in `infra/helm/zenith/Chart.yaml`:
  - `version` = Helm chart version (bumped when chart templates change)
  - `appVersion` = app code version (bumped when app code changes)
- All images share the same `appVersion` (monorepo = atomic release)
- Tags:
  - `v0.2.0` — release tag
  - `sha-abc1234` — commit SHA (for CI traceability)
  - `latest` — always points to most recent build on main
- Git tags: `v0.2.0` tag on the commit that produced the release
- Upgrade workflow:
  1. Code merged → CI builds `v0.2.1` images + chart → pushed to Harbor `zenith-stage`
  2. Update `staging-k8s/terraform.tfvars`: `zenith_chart_version = "0.2.1"`
  3. `terraform apply` → staging upgraded
  4. QA tests staging
  5. Promote: copy artifacts from `zenith-stage` → `zenith` in Harbor
  6. Update `production-k8s/terraform.tfvars`: `zenith_chart_version = "0.2.1"`
  7. `terraform apply` → production upgraded

**Manual steps (you do this):**
- Nothing — this is a convention. Just follow it.

**How to verify:**
```bash
# Check what versions exist in Harbor
helm show chart oci://registry.stage.freezenith.com/zenith-stage/zenith --version 0.2.0

# Check running version
kubectl --kubeconfig ~/.kube/zenith-staging.yaml get pods -n zenith-staging -o jsonpath='{.items[*].spec.containers[*].image}'
```

---

### Step 7: Local Development with `act`

**Goal:** All GitHub Actions runnable locally without pushing to GitHub.

**What to build:**
- `.secrets` file (gitignored) with all required secrets
- `.actrc` file with default flags
- `Makefile` with convenience targets:
  ```makefile
  ci-images:     act -j build-images --secret-file .secrets
  ci-chart:      act -j build-chart --secret-file .secrets
  ci-terraform:  act -j terraform-plan --secret-file .secrets
  ci-all:        act --secret-file .secrets
  ```

**Manual steps (you do this):**
1. Create `.secrets` file (copy from `.secrets.example` and fill in values)
2. Ensure `act` is installed: `brew install act`

**How to verify:**
```bash
make ci-images    # Should build and push all images
make ci-chart     # Should package and push chart
make ci-terraform # Should run terraform plan
```

---

## Files Created/Modified Summary

### New Files
| File | Purpose |
|---|---|
| `.github/workflows/build-images.yml` | CI: build + push Docker images to Harbor |
| `.github/workflows/build-chart.yml` | CI: package + push Helm chart to Harbor |
| `.github/workflows/terraform.yml` | CI: terraform plan/apply |
| `.secrets.example` | Template for local `act` secrets |
| `.actrc` | Default `act` configuration |
| `Makefile` | Convenience targets for CI |

### Modified Files
| File | Change |
|---|---|
| `infra/helm/zenith/Chart.yaml` | Bump version, set appVersion |
| `infra/helm/zenith/values.yaml` | Add image registry prefix, imagePullSecrets |
| `infra/helm/zenith/values-staging.yaml` | Registry prefix, imagePullPolicy → IfNotPresent |
| `infra/terraform/modules/k8s-platform/main.tf` | Helm release from Harbor OCI |
| `infra/terraform/modules/k8s-platform/variables.tf` | Add chart version + Harbor creds vars |
| `infra/terraform/staging-k8s/main.tf` | Pass Harbor creds + chart version |
| `infra/terraform/staging-k8s/variables.tf` | New variables |
| `.gitignore` | Add `.secrets`, `.actrc` |

---

## Execution Order

```
Step 1 → Step 2 → Step 3 → Step 4 → Step 5 → Step 6 → Step 7
  │         │        │         │
  │         │        │         └── terraform apply → platform running on staging
  │         │        └── Helm chart in Harbor
  │         └── k8s can pull from Harbor
  └── Docker images in Harbor
```

Steps 1-4 are sequential (each depends on the previous).
Steps 5-7 can be done in parallel after Step 4.

---

## Success Criteria

After all 7 steps:
- `git push` to main → GitHub Action builds images + chart → pushed to Harbor `zenith-stage`
- `make ci-images` locally → same thing via `act`
- Change version in `staging-k8s/terraform.tfvars` → `terraform apply` → staging upgraded
- All state tracked in Terraform state file
- Same process works for production (different tfvars, different Harbor project)
- Harbor UI shows all images + charts with version history

## Future Steps (Not in This Proposal)

- **Promotion script**: Harbor API to copy artifacts from `zenith-stage` → `zenith`
- **Production environment**: Same Terraform modules, different variables
- **CAPI + Team tier**: Cluster provisioning for paying customers
- **Terraform remote backend**: Move state from git to Hetzner S3
- **Monitoring**: kube-prometheus-stack deployment via Terraform
