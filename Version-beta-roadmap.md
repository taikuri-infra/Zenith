# Zenith Version Beta — Roadmap to Operational Staging SaaS

> This document tracks every phase, step, and task from current state to a fully functional staging SaaS that can be tested end-to-end as a real product.

**Goal:** A customer can visit `stage.freezenith.com`, sign up, create a project, deploy an app from GitHub, create a database, and see their usage — all on the staging environment. Admin can manage everything via Mission Control.

**Date:** Feb 2026
**Branch:** `openspec/infra-pipeline-v1`

---

## Overview: 8 Phases, 30 Steps

| Phase | Name | Steps | Spec | Status |
|---|---|---|---|---|
| 1 | CI/CD Pipeline | 7 | `phase1-ci-cd-pipeline` | 🟡 In Progress |
| 2 | Platform Bootstrap | 4 | `phase2-platform-bootstrap` | ⬜ Not Started |
| 3 | Auth & Identity | 4 | `phase3-auth-identity` | ⬜ Not Started |
| 4 | Customer Onboarding | 4 | `phase4-customer-onboarding` | ⬜ Not Started |
| 5 | Deploy Engine Live | 4 | `phase5-deploy-engine-live` | ⬜ Not Started |
| 6 | Data Services | 4 | `phase6-data-services` | ⬜ Not Started |
| 7 | Observability | 4 | `phase7-observability` | ⬜ Not Started |
| 8 | Billing & Metering | 4 | `phase8-billing-metering` | ⬜ Not Started |

**Dependency chain:**
```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6
                                                       ↓
                                              Phase 7 (can start after Phase 5)
                                              Phase 8 (needs Phase 7 for metering)
```

---

## Phase 1: CI/CD Pipeline
**Spec:** `openspec/changes/phase1-ci-cd-pipeline/proposal.md`

**What we're building:** Automated CI/CD that builds Docker images and Helm charts, pushes to Harbor, and deploys via Terraform.

**Why:** Without this, we can't deploy anything to staging in a reproducible way.

### Steps & Tasks

| Step | Task | Type | Status |
|---|---|---|---|
| 1.1 | Create `.github/workflows/build-images.yml` | Code | ⬜ |
| 1.1 | Build all 7 Docker images (api, landing, mc, web, operator, mc-demo, web-demo) | Code | ⬜ |
| 1.1 | Tag images with semver + SHA | Code | ⬜ |
| **1.1** | **Create Harbor robot account `ci-push` in zenith-stage project** | **Manual** | ⬜ |
| **1.1** | **Add `HARBOR_HOST`, `HARBOR_ROBOT_USER`, `HARBOR_ROBOT_TOKEN` to GitHub secrets** | **Manual** | ⬜ |
| 1.2 | Add `kubernetes_secret` for `harbor-registry-creds` to Terraform staging-k8s | Code | ⬜ |
| **1.2** | **Create Harbor robot account `k8s-pull` (pull-only)** | **Manual** | ⬜ |
| **1.2** | **Add pull credentials to `staging-k8s/terraform.tfvars`** | **Manual** | ⬜ |
| 1.3 | Create `.github/workflows/build-chart.yml` | Code | ⬜ |
| 1.3 | Package Helm chart with version from `Chart.yaml` | Code | ⬜ |
| 1.3 | Push chart to `oci://registry.stage.freezenith.com/zenith-stage` | Code | ⬜ |
| 1.4 | Update `k8s-platform` module: Helm release from Harbor OCI | Code | ⬜ |
| 1.4 | Update `staging-k8s/main.tf` with Harbor credentials + chart version | Code | ⬜ |
| 1.4 | Update Helm values: `imagePullPolicy: IfNotPresent`, add registry prefix | Code | ⬜ |
| **1.4** | **Get kubeconfig from staging server** | **Manual** | ⬜ |
| **1.4** | **Create `staging-k8s/terraform.tfvars`** | **Manual** | ⬜ |
| 1.5 | Create `.github/workflows/terraform.yml` (plan on PR, apply on merge) | Code | ⬜ |
| **1.5** | **Add `HCLOUD_TOKEN`, `CLOUDFLARE_API_TOKEN`, `KUBECONFIG_STAGING` to GitHub secrets** | **Manual** | ⬜ |
| 1.6 | Define version tagging convention in docs | Code | ⬜ |
| 1.7 | Create `.secrets.example`, `.actrc`, `Makefile` for local CI | Code | ⬜ |
| **1.7** | **Create `.secrets` file with real values** | **Manual** | ⬜ |
| **1.7** | **Install `act`: `brew install act`** | **Manual** | ⬜ |

**Acceptance Criteria:**
- [ ] `make ci-images` builds and pushes 7 images to Harbor
- [ ] `make ci-chart` packages and pushes Helm chart to Harbor
- [ ] `make ci-terraform` runs terraform plan
- [ ] Images visible in Harbor UI under `zenith-stage`
- [ ] `act` runs all workflows locally

**Verify:** Open Harbor UI → `zenith-stage` project → see all images and chart with version tags.

---

## Phase 2: Platform Bootstrap
**Spec:** `openspec/changes/phase2-platform-bootstrap/proposal.md`

**What we're building:** Deploy the full Zenith platform on staging k3s via Terraform Phase 3.

**Why:** This is the first time the entire platform runs in a state-tracked, reproducible way.

### Steps & Tasks

| Step | Task | Type | Status |
|---|---|---|---|
| 2.1 | Run Ansible to get kubeconfig | Code/Run | ⬜ |
| **2.1** | **Verify SSH access to 77.42.88.149** | **Manual** | ⬜ |
| 2.2 | Create `staging-k8s/terraform.tfvars` | Code | ⬜ |
| **2.2** | **Generate JWT secret: `openssl rand -hex 32`** | **Manual** | ⬜ |
| **2.2** | **Choose admin email + password for staging** | **Manual** | ⬜ |
| **2.2** | **Choose DB password for PostgreSQL** | **Manual** | ⬜ |
| 2.3 | `terraform init && terraform apply` in staging-k8s | Run | ⬜ |
| 2.4 | Verify all pods running and endpoints accessible | Verify | ⬜ |

**Acceptance Criteria:**
- [ ] All pods `Running` in `zenith-staging` namespace
- [ ] `stage.freezenith.com` serves landing page
- [ ] `api.stage.freezenith.com/health` returns 200
- [ ] PostgreSQL running with persistent volume
- [ ] TLS certificates issued by cert-manager
- [ ] `terraform plan` shows no drift

**Verify:**
```bash
kubectl --kubeconfig ~/.kube/zenith-staging.yaml get pods -n zenith-staging
curl -s https://stage.freezenith.com | head -1
curl -s https://api.stage.freezenith.com/health
```

---

## Phase 3: Auth & Identity
**Spec:** `openspec/changes/phase3-auth-identity/proposal.md`

**What we're building:** Real authentication — admin logs into MC, users register and log into Web Platform.

**Why:** Without auth, no one can use the platform.

### Steps & Tasks

| Step | Task | Type | Status |
|---|---|---|---|
| 3.1 | Verify admin user seeded on API startup | Code | ⬜ |
| 3.1 | Test login API endpoint | Verify | ⬜ |
| 3.2 | Set `NEXT_PUBLIC_API_URL` for MC in Helm values | Code | ⬜ |
| 3.2 | Verify MC login page works with real credentials | Verify | ⬜ |
| 3.3 | Set `NEXT_PUBLIC_API_URL` for Web in Helm values | Code | ⬜ |
| 3.3 | Verify Web Platform register + login works | Verify | ⬜ |
| 3.4 | Test token refresh (wait 1h or force expire) | Verify | ⬜ |

**Acceptance Criteria:**
- [ ] Admin logs into MC with real credentials
- [ ] New user registers via Web Platform
- [ ] JWT authentication works end-to-end (all API calls authenticated)
- [ ] Token refresh works transparently
- [ ] Demo mode still works on separate demo deployments

**Verify:** Login at `ms.stage.freezenith.com/login` → see real dashboard data.

---

## Phase 4: Customer Onboarding
**Spec:** `openspec/changes/phase4-customer-onboarding/proposal.md`

**What we're building:** Users create projects, each gets a k8s namespace with resource quotas.

**Why:** This is the core SaaS value — multi-tenancy.

### Steps & Tasks

| Step | Task | Type | Status |
|---|---|---|---|
| 4.1 | Wire API project creation to create k8s namespace | Code | ⬜ |
| 4.1 | Apply ResourceQuota per tier (Free/Pro limits) | Code | ⬜ |
| 4.1 | Apply CiliumNetworkPolicy for namespace isolation | Code | ⬜ |
| 4.2 | Wire MC `/tenants` page to real API data | Code | ⬜ |
| 4.2 | Wire MC dashboard stats to real counts | Code | ⬜ |
| 4.3 | Wire Web Platform project selector | Code | ⬜ |
| 4.3 | Scope all Web Platform API calls to project | Code | ⬜ |
| 4.4 | Configure wildcard DNS `*.apps.stage.freezenith.com` | Code | ⬜ |
| **4.4** | **Add wildcard DNS record in Cloudflare** | **Manual** | ⬜ |

**Acceptance Criteria:**
- [ ] User creates project → namespace + ResourceQuota in k8s
- [ ] MC shows real customer list with resource usage
- [ ] Web Platform scoped to user's project
- [ ] Free tier: 1 pod, 256Mi RAM, 250m CPU
- [ ] Pro tier: 10 pods, 4Gi RAM, 2 CPU
- [ ] Cross-namespace traffic blocked by Cilium

**Verify:** Create project via API → `kubectl get ns zen-{name}` exists with ResourceQuota.

---

## Phase 5: Deploy Engine Live
**Spec:** `openspec/changes/phase5-deploy-engine-live/proposal.md`

**What we're building:** Users deploy apps from Git repos with automatic builds (Kaniko) and deployments.

**Why:** This is what customers are paying for.

### Steps & Tasks

| Step | Task | Type | Status |
|---|---|---|---|
| 5.1 | Configure Kaniko with Harbor registry in API | Code | ⬜ |
| 5.1 | Create k8s ServiceAccount + Secret for Kaniko Harbor auth | Code | ⬜ |
| **5.1** | **Create Harbor robot account `kaniko-push` for built images** | **Manual** | ⬜ |
| 5.2 | Wire Deployer to create resources in user's namespace | Code | ⬜ |
| 5.2 | IngressRoute with TLS for app subdomains | Code | ⬜ |
| 5.3 | Wire GitHub webhook handler on staging | Code | ⬜ |
| **5.3** | **Configure GitHub webhook on a test repo** | **Manual** | ⬜ |
| 5.4 | Wire Web Platform `/deploy` page to real API | Code | ⬜ |
| 5.4 | Wire build log viewer to real SSE stream | Code | ⬜ |
| 5.4 | Wire deployment history with rollback | Code | ⬜ |

**Acceptance Criteria:**
- [ ] Deploy app from Git → build runs → app accessible via HTTPS
- [ ] GitHub push triggers auto-deploy
- [ ] Build logs stream in real-time
- [ ] Rollback to previous deployment works
- [ ] Web Platform Deploy page fully functional

**Verify:** Deploy a Next.js app from GitHub → access at `myapp.apps.stage.freezenith.com`.

---

## Phase 6: Data Services
**Spec:** `openspec/changes/phase6-data-services/proposal.md`

**What we're building:** Per-customer PostgreSQL databases (CloudNativePG) and object storage.

**Why:** Apps need databases and storage.

### Steps & Tasks

| Step | Task | Type | Status |
|---|---|---|---|
| 6.1 | Add CloudNativePG Helm release to k8s-platform module | Code | ⬜ |
| 6.1 | `terraform apply` to install CNPG operator | Run | ⬜ |
| 6.2 | Wire API database creation to CNPG Cluster CRD | Code | ⬜ |
| 6.2 | Generate connection strings | Code | ⬜ |
| 6.2 | Enforce tier storage limits | Code | ⬜ |
| 6.3 | Deploy MinIO via Helm OR wire Hetzner S3 API | Code | ⬜ |
| 6.3 | Wire API storage creation | Code | ⬜ |
| **6.3** | **If Hetzner S3: provide API credentials** | **Manual** | ⬜ |
| 6.4 | Wire Web Platform database + storage pages to real API | Code | ⬜ |

**Acceptance Criteria:**
- [ ] CloudNativePG operator running
- [ ] Users create PostgreSQL databases with connection strings
- [ ] Users create storage buckets with access credentials
- [ ] Databases persist across pod restarts
- [ ] Tier limits enforced

**Verify:** Create database via Web Platform → connect to it with `psql`.

---

## Phase 7: Observability
**Spec:** `openspec/changes/phase7-observability/proposal.md`

**What we're building:** Monitoring (Prometheus + Grafana), logging (Loki), and real metrics in MC and Web Platform.

**Why:** Can't operate a SaaS without observability.

### Steps & Tasks

| Step | Task | Type | Status |
|---|---|---|---|
| 7.1 | Enable `monitoring = true` in Terraform staging-k8s | Code | ⬜ |
| 7.1 | `terraform apply` to deploy monitoring stack | Run | ⬜ |
| **7.1** | **Add Grafana DNS record (optional)** | **Manual** | ⬜ |
| 7.2 | Wire API `/metrics` endpoint with ServiceMonitor | Code | ⬜ |
| 7.3 | Wire MC infrastructure page to Prometheus queries | Code | ⬜ |
| 7.4 | Wire Web Platform monitoring page to real metrics | Code | ⬜ |

**Acceptance Criteria:**
- [ ] Prometheus + Grafana + Loki running
- [ ] Pre-built dashboards available
- [ ] MC shows real infrastructure metrics
- [ ] Web Platform shows real app metrics
- [ ] Alerting rules configured

**Verify:** Open Grafana → see node/pod metrics dashboards.

---

## Phase 8: Billing & Metering
**Spec:** `openspec/changes/phase8-billing-metering/proposal.md`

**What we're building:** Stripe subscriptions, resource usage tracking, tier enforcement, self-service upgrades.

**Why:** Revenue.

### Steps & Tasks

| Step | Task | Type | Status |
|---|---|---|---|
| 8.1 | Add Stripe Go SDK to API | Code | ⬜ |
| 8.1 | Create checkout session + customer portal endpoints | Code | ⬜ |
| 8.1 | Stripe webhook handler | Code | ⬜ |
| **8.1** | **Create Stripe account + Products + Prices** | **Manual** | ⬜ |
| **8.1** | **Add Stripe secrets to Terraform** | **Manual** | ⬜ |
| **8.1** | **Configure Stripe webhook URL** | **Manual** | ⬜ |
| 8.2 | Metering job: query Prometheus for per-namespace usage | Code | ⬜ |
| 8.2 | `resource_usage` table + API endpoint | Code | ⬜ |
| 8.3 | Pre-action limit checks (block when tier exceeded) | Code | ⬜ |
| 8.4 | Wire Web Platform `/billing` page to real data | Code | ⬜ |
| 8.4 | Upgrade button → Stripe checkout | Code | ⬜ |

**Acceptance Criteria:**
- [ ] Free user blocked at tier limits (gets "Upgrade" prompt)
- [ ] Pro checkout via Stripe works
- [ ] Plan upgrade increases resource limits automatically
- [ ] Usage tracked and displayed in Web Platform
- [ ] Invoice history accessible
- [ ] Failed payments → grace period → suspension

**Verify:** Sign up as free user → hit pod limit → upgrade to Pro → deploy more pods.

---

## Summary: All Manual Steps (Your Checklist)

Everything you need to do with your hands (not code):

### Phase 1
- [ ] Create Harbor robot account `ci-push` (push to zenith-stage)
- [ ] Create Harbor robot account `k8s-pull` (pull from zenith-stage)
- [ ] Add GitHub secrets: `HARBOR_HOST`, `HARBOR_ROBOT_USER`, `HARBOR_ROBOT_TOKEN`
- [ ] Add GitHub secrets: `HCLOUD_TOKEN`, `CLOUDFLARE_API_TOKEN`, `KUBECONFIG_STAGING`
- [ ] Create `.secrets` file for local `act`
- [ ] Install `act`: `brew install act`

### Phase 2
- [ ] Verify SSH to `77.42.88.149`
- [ ] Generate JWT secret
- [ ] Choose admin email + password
- [ ] Choose DB password
- [ ] Create `staging-k8s/terraform.tfvars`

### Phase 4
- [ ] Add wildcard DNS `*.apps.stage.freezenith.com → 77.42.88.149` in Cloudflare

### Phase 5
- [ ] Create Harbor robot account `kaniko-push`
- [ ] Configure GitHub webhook on test repo

### Phase 6
- [ ] (If Hetzner S3) Provide S3 API credentials

### Phase 7
- [ ] (Optional) Add Grafana DNS record

### Phase 8
- [ ] Create Stripe account + Products + Prices
- [ ] Add Stripe secrets
- [ ] Configure Stripe webhook URL

---

## End State

After all 8 phases, staging looks like this:

```
Customer visits stage.freezenith.com
  → Signs up (Phase 3)
  → Creates project (Phase 4)
    → Namespace created with ResourceQuota
    → Cilium network isolation applied
  → Deploys app from GitHub (Phase 5)
    → Kaniko builds image → Harbor → k8s
    → App live at myapp.apps.stage.freezenith.com
  → Creates PostgreSQL database (Phase 6)
    → CloudNativePG cluster in their namespace
    → Connection string provided
  → Sees monitoring dashboards (Phase 7)
    → Request rate, error rate, resource usage
  → Upgrades to Pro via Stripe (Phase 8)
    → Limits increased, more pods allowed

Admin opens ms.stage.freezenith.com
  → Sees all customers, resource usage, clusters
  → Manages tenants, modules, infrastructure
  → Real metrics from Prometheus

Everything tracked:
  → Terraform state file knows what's installed
  → Harbor has all versioned images + charts
  → Git has all infrastructure code
  → Upgrade = bump version → terraform apply
```
