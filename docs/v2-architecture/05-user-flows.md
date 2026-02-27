# User Flows: Admin, Customer, and Developer

> **Zenith V2 Platform Architecture -- User Experience Design**
>
> This document traces the journey of each persona through the Zenith platform,
> from first interaction to day-to-day operations.

> **Status:** Design Complete, Implementation Pending
> **Last Updated:** 2026-02-25
> **Author:** Babak + Claude (Platform Architecture Session)

---

## Table of Contents

1. [Personas Overview](#personas-overview)
2. [Platform Admin Flows](#platform-admin-flows)
   - [Initial Platform Setup](#initial-platform-setup)
   - [Day-to-Day Operations](#day-to-day-operations)
   - [Customer Management](#customer-management)
   - [Incident Response](#incident-response)
   - [Scaling Operations](#scaling-operations)
3. [Customer Developer Flows](#customer-developer-flows)
   - [Signup and Onboarding](#signup-and-onboarding)
   - [Login and Authentication](#login-and-authentication)
   - [Application Deployment](#application-deployment)
   - [Database Management](#database-management)
   - [Storage Management (S3)](#storage-management-s3)
   - [Custom Domain Setup](#custom-domain-setup)
   - [Tier Upgrade](#tier-upgrade)
4. [Zenith Developer Flows](#zenith-developer-flows)
   - [Local Development](#local-development)
   - [CI/CD Pipeline](#cicd-pipeline)
   - [Adding New Features](#adding-new-features)
   - [Debugging in Staging](#debugging-in-staging)
5. [Request Flow Diagrams](#request-flow-diagrams)
   - [Frontend Request Path](#frontend-request-path)
   - [Backend Request Path](#backend-request-path)
   - [Cross-Service Communication](#cross-service-communication)
6. [Error Handling and Edge Cases](#error-handling-and-edge-cases)

---

## Personas Overview

Zenith serves three distinct personas. Each has different goals, different tools, and different levels of access. Understanding these personas is critical to understanding why the platform is designed the way it is.

```
+-------------------------------------------------------------------+
|                     ZENITH V2 PERSONAS                            |
+-------------------------------------------------------------------+
|                                                                   |
|  PLATFORM ADMIN (Babak / DoTech)                                  |
|  +-------------------------------------------------------------+ |
|  | Tools: Grafana, ArgoCD, Mission Control, kubectl, Terraform  | |
|  | Goal:  Keep the platform running, onboard customers          | |
|  | Access: Full cluster admin, all namespaces, all secrets      | |
|  +-------------------------------------------------------------+ |
|                                                                   |
|  CUSTOMER DEVELOPER (End User)                                    |
|  +-------------------------------------------------------------+ |
|  | Tools: Web Platform (freezenith.com), Git, CLI (future)      | |
|  | Goal:  Deploy apps, manage databases, ship product           | |
|  | Access: Own namespace only, own Keycloak realm, own resources | |
|  +-------------------------------------------------------------+ |
|                                                                   |
|  ZENITH DEVELOPER (DoTech Engineering Team)                       |
|  +-------------------------------------------------------------+ |
|  | Tools: lich CLI, VS Code, GitHub, ArgoCD, staging cluster    | |
|  | Goal:  Build and improve the Zenith platform itself          | |
|  | Access: Code repo, staging cluster, CI/CD pipelines          | |
|  +-------------------------------------------------------------+ |
|                                                                   |
+-------------------------------------------------------------------+
```

| Persona | Primary Interface | Authentication | Scope |
|---------|------------------|----------------|-------|
| **Platform Admin** | Grafana, ArgoCD, kubectl, Mission Control | kubeconfig + OIDC | Entire cluster |
| **Customer Developer** | Web Platform (browser) | Keycloak OIDC (own realm) | Own namespace |
| **Zenith Developer** | lich CLI, GitHub, ArgoCD | GitHub SSO + kubeconfig (staging) | Code + staging |

---

## Platform Admin Flows

The Platform Admin is the operator of the Zenith infrastructure. This person (currently Babak and the DoTech team) is responsible for the health of the entire platform, all customer environments, and all infrastructure components.

### Initial Platform Setup

This is the one-time flow to bring Zenith V2 from zero to a running platform. It follows the four-phase deployment pipeline defined in the overview document.

```
PLATFORM ADMIN: Initial Setup Flow
===================================

  Admin workstation
       |
       | Step 1: terraform apply (Phase 1)
       v
  +------------------+
  | Hetzner Cloud    |   Creates: VMs, firewalls, SSH keys, volumes
  | Cloudflare       |   Creates: DNS zones, A records, API tokens
  +------------------+
       |
       | Step 2: ansible-playbook (Phase 2)
       v
  +------------------+
  | k3s cluster      |   Installs: k3s, Cilium (replaces Flannel),
  | (bare metal)     |   hcloud CSI driver, OS hardening, sysctl
  +------------------+
       |
       | Step 3: terraform apply (Phase 3)
       v
  +------------------+
  | Bootstrapped     |   Installs via Helm:
  | cluster          |     cert-manager, CNPG operator, APISIX + etcd,
  |                  |     Keycloak, external-dns, Temporal, Harbor,
  |                  |     Prometheus + Grafana + Loki + Tempo,
  |                  |     ArgoCD, Kyverno, Falco, Sealed Secrets,
  |                  |     Velero
  +------------------+
       |
       | Step 4: git push (Phase 4 -- automatic)
       v
  +------------------+
  | Live platform    |   ArgoCD syncs: zenith-api, zenith-landing,
  |                  |   zenith-admin, demo environments
  +------------------+
       |
       | Step 5: Verify
       v
  Admin opens:
    - https://freezenith.com        (landing page loads)
    - https://grafana.freezenith.com (dashboards healthy)
    - https://argocd.freezenith.com  (all apps synced)
    - https://admin.freezenith.com   (admin panel functional)
```

**Detailed steps for each phase:**

1. **Phase 1 -- Hetzner + Cloudflare (5 minutes)**
   - `cd infra/terraform/staging && terraform init && terraform apply`
   - Terraform creates the Hetzner VM (CX41 or equivalent), attaches a firewall allowing only ports 80, 443, 6443 (K8s API), and 22 (SSH).
   - Cloudflare DNS records are created for `freezenith.com`, `*.freezenith.com`, and `api.freezenith.com`.
   - Outputs: server IP, SSH key path, Cloudflare zone ID.

2. **Phase 2 -- Ansible + k3s (10 minutes)**
   - `cd infra/ansible && ansible-playbook playbooks/site.yml -i inventory/staging.yml`
   - Ansible SSHs into the server, hardens the OS (fail2ban, sysctl, unattended-upgrades), installs k3s with `--flannel-backend=none` (Cilium will be the CNI), deploys Cilium with WireGuard encryption and Hubble observability.
   - Outputs: kubeconfig file at `~/.kube/config` (or specified path).

3. **Phase 3 -- Cluster Bootstrap (15-20 minutes)**
   - `cd infra/terraform/staging-k8s && terraform init && terraform apply`
   - This is the longest phase. Terraform installs 15+ Helm charts in dependency order. cert-manager must be ready before Keycloak (needs TLS). CNPG operator must be ready before creating database clusters. ArgoCD must be ready before Phase 4.
   - Admin monitors progress: `kubectl get pods -A --watch`

4. **Phase 4 -- ArgoCD Apps (automatic, 2-5 minutes)**
   - ArgoCD detects the Application CRDs created in Phase 3, pulls the latest manifests from the Git repository, and deploys zenith-api, zenith-landing, and other platform services.
   - No manual action required. Admin verifies via ArgoCD UI.

5. **Verification checklist:**
   - All pods in `Running` or `Completed` state
   - Grafana dashboards show metrics flowing
   - Keycloak admin console accessible
   - APISIX dashboard shows routes configured
   - Harbor UI accessible, Trivy scanner operational
   - Temporal Web UI shows healthy workers

---

### Day-to-Day Operations

Once the platform is running, the admin's daily routine involves monitoring and maintenance.

```
PLATFORM ADMIN: Daily Operations
=================================

  Morning check (5 min):
  +-------------------+     +-------------------+     +-------------------+
  | Grafana           | --> | ArgoCD            | --> | Alertmanager      |
  | - Node CPU/mem    |     | - All apps synced |     | - No firing alerts|
  | - Pod count       |     | - No drift        |     | - Silence expired |
  | - Disk usage      |     | - Health green    |     |   alerts reviewed |
  | - CNPG replication|     +-------------------+     +-------------------+
  | - Loki log volume |
  +-------------------+

  Weekly tasks:
  +-------------------+     +-------------------+     +-------------------+
  | Backup verify     | --> | Security review   | --> | Capacity planning |
  | - CNPG WAL valid  |     | - Falco alerts    |     | - Disk growth     |
  | - Velero restore  |     | - Kyverno blocks  |     | - Pod density     |
  |   test (staging)  |     | - Hubble flows    |     | - DB shard fill   |
  | - S3 accessible   |     | - Trivy scan new  |     | - S3 bucket sizes |
  +-------------------+     +-------------------+     +-------------------+
```

**Grafana dashboards the admin uses daily:**

| Dashboard | What It Shows | Red Flags |
|-----------|--------------|-----------|
| **Cluster Overview** | Node CPU, memory, disk, pod count | CPU > 80%, disk > 85% |
| **CNPG Health** | Replication lag, WAL archiving, connections | Lag > 10s, archiving failed |
| **APISIX Traffic** | Request rate, latency p99, error rate | p99 > 2s, 5xx > 1% |
| **Cilium/Hubble** | Network flows, policy drops, DNS latency | Policy drops in customer NS |
| **Keycloak** | Login rate, failed logins, realm count | Failed logins spike (brute force) |
| **Loki Logs** | Log volume, error rate by namespace | Error rate spike in any NS |
| **Temporal** | Workflow success rate, queue depth | Failed workflows, queue buildup |

---

### Customer Management

The admin manages customer lifecycles through the Mission Control admin panel and, when needed, direct kubectl access.

```
PLATFORM ADMIN: Customer Lifecycle
====================================

  Create Customer (via Admin Panel):
  +-------------------+
  | Admin Panel       |
  | POST /api/admin/  |
  |   customers       |
  |                   |
  | Fields:           |
  |  - name           |
  |  - email          |
  |  - tier (Free/    |
  |    Pro/Team/Ent)  |
  |  - domain (opt)   |
  +--------+----------+
           |
           v
  +-------------------+
  | zenith-api        |  Validates input, checks quotas
  | starts Temporal   |
  | workflow          |
  +--------+----------+
           |
           v
  +-----------------------------------------+
  | Temporal: provision-customer             |
  |                                         |
  | [Keycloak] --> [Database] --> [S3]      |
  |     --> [Namespace] --> [Secrets]        |
  |     --> [Deployments] --> [Ingress]      |
  |     --> [DNS] --> [TLS] --> [Notify]     |
  +-----------------------------------------+
           |
           v
  Customer receives welcome email
  Environment ready at <customer>.freezenith.com
```

**Upgrade customer tier (e.g., Free to Pro):**

1. Admin opens customer in Mission Control admin panel.
2. Selects new tier (Pro).
3. zenith-api starts Temporal workflow: `upgrade-customer-tier`.
4. Temporal activities:
   - Migrate database from free-pg cluster to a pro shard (pg_dump + pg_restore).
   - Update ResourceQuota in namespace (higher CPU/memory/pod limits).
   - Update APISIX rate-limit plugin config (higher limits).
   - Update Keycloak realm settings (enable additional features).
   - Enable custom domain support in ingress configuration.
   - Notify customer of upgrade completion.
5. Zero downtime -- old resources remain until migration is verified.

**Downgrade customer tier (e.g., Pro to Free):**

1. Admin verifies customer is within Free tier resource limits.
2. If customer exceeds limits (e.g., 3 databases, Free allows 1), admin notifies customer to reduce usage first.
3. Once within limits, same Temporal workflow in reverse.
4. Database migrated back to shared free-pg cluster.
5. Custom domain removed (CNAME record deleted from external-dns).

**Delete customer:**

1. Admin marks customer for deletion in Mission Control.
2. 30-day grace period begins (customer data retained but environment stopped).
3. After 30 days, Temporal workflow: `delete-customer`.
4. Activities: delete namespace, drop database, delete S3 bucket, delete Keycloak realm, delete DNS records.
5. Final backup taken before deletion and retained for 90 days.

---

### Incident Response

When something goes wrong, the admin follows a structured incident response flow.

```
INCIDENT RESPONSE FLOW
=======================

  Alert fires (Alertmanager -> Slack/Telegram)
       |
       v
  +------------------+
  | Triage (5 min)   |   Severity?
  |                  |     P1: Platform down (all customers affected)
  |                  |     P2: Single customer down
  |                  |     P3: Degraded performance
  |                  |     P4: Non-urgent (cleanup, optimization)
  +--------+---------+
           |
           v
  +------------------+
  | Investigate      |   Tools:
  |                  |     - Grafana dashboards (metrics)
  |                  |     - Loki (logs)
  |                  |     - Tempo (distributed traces)
  |                  |     - Hubble UI (network flows)
  |                  |     - kubectl describe/logs
  +--------+---------+
           |
           v
  +------------------+
  | Remediate        |   Common actions:
  |                  |     - Restart pod (kubectl rollout restart)
  |                  |     - Scale up (kubectl scale)
  |                  |     - Apply hotfix (git push -> ArgoCD sync)
  |                  |     - Block attacker IP (Cloudflare WAF rule)
  |                  |     - Isolate namespace (CiliumNetworkPolicy)
  +--------+---------+
           |
           v
  +------------------+
  | Post-mortem      |   Within 48 hours:
  |                  |     - Root cause analysis
  |                  |     - Timeline of events
  |                  |     - What failed, what caught it
  |                  |     - Prevention measures
  +------------------+
```

**Common incident scenarios and response:**

| Scenario | Detection | Response |
|----------|-----------|----------|
| Customer pod OOMKilled | Grafana alert: pod restart count | Check LimitRange, increase memory if justified, or advise customer to optimize |
| CNPG replication lag > 30s | CNPG dashboard alert | Check disk I/O, WAL archiving, network between primary/replica |
| Falco: shell spawned in container | Falco alert -> Slack | Immediately isolate namespace with deny-all CiliumNetworkPolicy, investigate logs, check for compromise |
| APISIX 5xx spike | APISIX dashboard | Check backend pod health, recent deployments, Keycloak availability |
| Disk usage > 90% | Node exporter alert | Identify large consumers (Loki retention? Customer uploads?), expand volume or clean up |
| Certificate expiry < 7 days | cert-manager alert | Check cert-manager logs, verify DNS-01 solver, manual renewal if needed |

---

### Scaling Operations

As the platform grows, the admin adds capacity.

```
SCALING DECISION TREE
======================

  Is cluster resource usage > 70%?
       |
       +-- CPU pressure ---------> Add nodes (Terraform: increase node count)
       |
       +-- Memory pressure ------> Add nodes OR upgrade server type
       |
       +-- Disk pressure --------> Expand Hetzner Volumes (CSI resize)
       |
       +-- Pod density ----------> Add nodes (max ~110 pods/node)
       |
       +-- DB shard full --------> Create new CNPG shard
       |     (> 20 Pro customers    (Terraform: new Cluster CR)
       |      per shard)
       |
       +-- S3 throughput --------> Hetzner handles this (managed service)
       |
       +-- Keycloak sessions ----> Scale Keycloak replicas
       |     (> 1000 concurrent)
       |
       +-- APISIX latency ------> Scale APISIX replicas + etcd
            (p99 > 500ms)
```

**Adding a new node to the shared cluster:**

1. Update `infra/terraform/staging/variables.tf` -- increase `node_count`.
2. `terraform apply` -- Hetzner provisions new VM.
3. Ansible runs on new node -- installs k3s agent, joins cluster.
4. Cilium automatically extends to new node (DaemonSet).
5. Kubernetes scheduler starts placing pods on new node.
6. Verify: `kubectl get nodes` shows new node Ready.

**Creating a new CNPG shard (when Pro shard reaches 20 customers):**

1. Create new Cluster CR in `infra/k8s/cnpg/pro-shard-N.yaml`.
2. Apply via ArgoCD (git push) or direct `kubectl apply`.
3. CNPG operator creates primary + replica pods with Hetzner Volumes.
4. Configure WAL archiving to S3 for new shard.
5. Update zenith-api shard assignment logic to include new shard.
6. New Pro customers are automatically assigned to the shard with capacity.

---

## Customer Developer Flows

The Customer Developer is the end user of Zenith -- a developer who wants to deploy their applications without managing infrastructure. Every interaction goes through the Web Platform at `freezenith.com` (or their custom domain for Pro+ tiers).

### Signup and Onboarding

This is the most critical flow in the platform. A smooth signup experience determines whether a customer stays or leaves. The goal: from clicking "Sign Up" to a running environment in under 2 minutes (Free/Pro) or under 20 minutes (Team/Enterprise).

```
CUSTOMER SIGNUP FLOW (Free/Pro)
================================

  Customer visits freezenith.com
       |
       | Clicks "Get Started"
       v
  +-------------------+
  | Registration Form |   Fields:
  |                   |     - Email
  |                   |     - Password
  |                   |     - Company name
  |                   |     - Tier selection (Free default)
  +--------+----------+
           |
           | POST /api/v1/auth/register
           v
  +-------------------+
  | zenith-api        |
  |                   |  1. Validate email (not disposable, not duplicate)
  |                   |  2. Check platform capacity (room for new customer?)
  |                   |  3. Create customer record in platform DB
  |                   |  4. Start Temporal workflow
  +--------+----------+
           |
           v
  +---------------------------------------------------+
  | Temporal workflow: provision-customer               |
  |                                                   |
  | Activity 1: CreateKeycloakRealm        [~5s]      |
  |   - POST /admin/realms                            |
  |   - Create realm: <customer-slug>                 |
  |   - Create client: zenith-web                     |
  |   - Create default roles: admin, developer        |
  |   - Create initial admin user                     |
  |                                                   |
  | Activity 2: CreateDatabase             [~3s]      |
  |   - Connect to free-pg or assigned pro shard      |
  |   - CREATE DATABASE customer_<slug>               |
  |   - CREATE USER customer_<slug> WITH PASSWORD ... |
  |   - GRANT ALL PRIVILEGES                          |
  |                                                   |
  | Activity 3: CreateS3Bucket             [~5s]      |
  |   - Hetzner API: create bucket                    |
  |   - Create access key + secret key                |
  |   - Set bucket policy (private by default)        |
  |                                                   |
  | Activity 4: CreateK8sNamespace         [~2s]      |
  |   - Create namespace: zenith-<slug>               |
  |   - Labels: tier=free, customer=<slug>            |
  |   - Apply ResourceQuota (Free tier limits)        |
  |   - Apply LimitRange (per-pod limits)             |
  |   - Apply PodSecurityStandard: restricted         |
  |   - Apply CiliumNetworkPolicy (default deny +     |
  |     allow egress to APISIX, Keycloak, DNS)        |
  |                                                   |
  | Activity 5: CreateK8sSecrets           [~1s]      |
  |   - Secret: db-credentials                        |
  |   - Secret: s3-credentials                        |
  |   - Secret: keycloak-credentials                  |
  |                                                   |
  | Activity 6: CreateIngressRoutes        [~2s]      |
  |   - Traefik IngressRoute: <slug>.freezenith.com   |
  |     -> customer-frontend:3000                     |
  |   - ApisixRoute: api.<slug>.freezenith.com/*      |
  |     -> customer-backend:8080 (JWT protected)      |
  |   - ApisixPluginConfig: CORS, rate-limit          |
  |                                                   |
  | Activity 7: WaitForDNS                 [~30s]     |
  |   - external-dns creates Cloudflare A record      |
  |   - Poll until DNS resolves correctly             |
  |                                                   |
  | Activity 8: WaitForTLSCert             [~30s]     |
  |   - cert-manager creates Certificate CR           |
  |   - DNS-01 challenge via Cloudflare API           |
  |   - Poll until TLS Secret exists                  |
  |                                                   |
  | Activity 9: NotifyCustomerReady        [~1s]      |
  |   - Update customer status: READY                 |
  |   - Send welcome email with login URL             |
  |   - WebSocket push to browser: "Ready!"           |
  +---------------------------------------------------+
           |
           | Total time: ~60-90 seconds
           v
  +-------------------+
  | Customer browser  |   "Your environment is ready!"
  | auto-redirects to |   URL: https://<slug>.freezenith.com
  | Web Platform      |   Login with credentials from email
  +-------------------+
```

**What the customer sees during provisioning:**

The browser shows a progress screen with real-time updates via WebSocket:

```
  Setting up your environment...

  [====] Creating identity realm        (done)
  [====] Provisioning database          (done)
  [====] Creating storage bucket        (done)
  [==  ] Configuring networking         (in progress)
  [    ] Securing with TLS certificate  (pending)
  [    ] Final verification             (pending)

  Estimated time remaining: 45 seconds
```

**If provisioning fails at any step:**

Temporal automatically retries failed activities (up to 3 times with exponential backoff). If an activity fails permanently:
- Temporal marks the workflow as failed.
- zenith-api sets customer status to `PROVISIONING_FAILED`.
- Admin receives alert via Alertmanager.
- Customer sees: "We're having trouble setting up your environment. Our team has been notified."
- Temporal's built-in compensation (saga pattern) rolls back completed steps.

---

### Login and Authentication

Every customer gets their own Keycloak realm. This means their users, roles, and sessions are completely isolated from other customers.

```
CUSTOMER LOGIN FLOW (OIDC Authorization Code + PKCE)
=====================================================

  Customer visits <slug>.freezenith.com
       |
       | Browser loads Next.js frontend
       v
  +-------------------+
  | Frontend detects  |   No valid session cookie
  | unauthenticated   |
  +--------+----------+
           |
           | Redirect to Keycloak
           v
  +-------------------+
  | Keycloak          |   URL: auth.freezenith.com/realms/<slug>/
  | Login Page        |        protocol/openid-connect/auth
  |                   |        ?client_id=zenith-web
  |                   |        &redirect_uri=https://<slug>.freezenith.com/callback
  |                   |        &response_type=code
  |                   |        &code_challenge=<PKCE>
  |                   |        &scope=openid profile email
  +--------+----------+
           |
           | User enters email + password
           v
  +-------------------+
  | Keycloak verifies |   Checks credentials against realm user store
  | credentials       |   Checks MFA if enabled (Pro+ feature)
  +--------+----------+
           |
           | Redirect back with authorization code
           v
  +-------------------+
  | Frontend callback |   POST to Keycloak token endpoint:
  | /callback         |     grant_type=authorization_code
  |                   |     code=<auth_code>
  |                   |     code_verifier=<PKCE_verifier>
  +--------+----------+
           |
           | Keycloak returns: access_token, refresh_token, id_token
           v
  +-------------------+
  | Frontend stores   |   access_token: in memory (short-lived, ~5 min)
  | tokens            |   refresh_token: httpOnly cookie (longer-lived, ~30 min)
  |                   |   id_token: user info display
  +--------+----------+
           |
           | API calls include Authorization: Bearer <access_token>
           v
  +-------------------+
  | Traefik           |   Routes api.<slug>.freezenith.com to APISIX
  +--------+----------+
           |
           v
  +-------------------+
  | APISIX            |   jwt-auth plugin:
  |                   |     1. Extract Bearer token from header
  |                   |     2. Fetch JWKS from Keycloak:
  |                   |        auth.freezenith.com/realms/<slug>/
  |                   |        protocol/openid-connect/certs
  |                   |     3. Verify signature (RS256)
  |                   |     4. Check exp, aud, iss claims
  |                   |     5. If valid -> forward to backend
  |                   |     6. If invalid -> 401 Unauthorized
  +--------+----------+
           |
           | Headers added by APISIX:
           |   X-Consumer-Username: user@example.com
           |   X-Consumer-Realm: <slug>
           |   X-Consumer-Roles: admin,developer
           v
  +-------------------+
  | Backend pod       |   Trusts APISIX headers (no re-verification)
  | (customer API)    |   Uses X-Consumer-Realm to scope DB queries
  +-------------------+
```

**Token refresh flow:**

When the access token expires (every 5 minutes), the frontend silently refreshes it:

1. Frontend detects 401 response from API.
2. Frontend sends refresh_token to Keycloak token endpoint.
3. Keycloak validates refresh_token, issues new access_token + refresh_token.
4. Frontend retries the original API call with the new access_token.
5. If refresh_token is also expired, redirect to login page.

---

### Application Deployment

This is the core value proposition of Zenith: customers push code, and the platform builds and deploys it automatically.

```
APPLICATION DEPLOYMENT FLOW (Git Push)
=======================================

  Developer pushes code to Git repository
       |
       | Webhook fires to zenith-api
       v
  +-------------------+
  | zenith-api        |   Validates webhook signature
  | receives webhook  |   Identifies customer + repo + branch
  +--------+----------+
           |
           v
  +-------------------+
  | zenith-api starts |   Temporal workflow: build-and-deploy
  | Temporal workflow  |
  +--------+----------+
           |
           v
  +----------------------------------------------------+
  | Activity 1: CloneRepository              [~10s]    |
  |   - Git clone into temporary PVC                   |
  |   - Checkout correct branch/commit                 |
  |                                                    |
  | Activity 2: DetectBuildConfig            [~2s]     |
  |   - Look for Dockerfile (preferred)                |
  |   - Look for package.json (Node.js buildpack)      |
  |   - Look for go.mod (Go buildpack)                 |
  |   - Look for requirements.txt (Python buildpack)   |
  |                                                    |
  | Activity 3: BuildImage (Kaniko)          [~60-120s]|
  |   - Kaniko pod runs in build namespace             |
  |   - Builds image from Dockerfile or buildpack      |
  |   - Pushes to Harbor:                              |
  |     harbor.freezenith.com/<customer>/<app>:<sha>   |
  |   - Trivy scans image for vulnerabilities          |
  |   - cosign signs image with cluster key            |
  |                                                    |
  | Activity 4: UpdateDeployment             [~5s]     |
  |   - Update Deployment image tag in customer NS     |
  |   - Rolling update: new pods come up, old drain    |
  |   - Health check: wait for readiness probe         |
  |                                                    |
  | Activity 5: NotifyDeployComplete         [~1s]     |
  |   - Update deploy history in platform DB           |
  |   - WebSocket push to Web Platform                 |
  |   - "Deploy v1.2.3 (abc1234) succeeded"            |
  +----------------------------------------------------+
           |
           v
  Customer sees in Web Platform:
  +---------------------------------------------------+
  | Recent Deploys                                     |
  | ------------------------------------------------- |
  | v1.2.3 (abc1234)  |  2 min ago  |  SUCCESS  | [Log]|
  | v1.2.2 (def5678)  |  1 day ago  |  SUCCESS  | [Log]|
  | v1.2.1 (ghi9012)  |  3 days ago |  FAILED   | [Log]|
  +---------------------------------------------------+
```

**Alternative: Upload Docker image directly:**

For customers who build images in their own CI/CD:

1. Customer authenticates to Harbor: `docker login harbor.freezenith.com`
2. Customer pushes image: `docker push harbor.freezenith.com/<customer>/<app>:<tag>`
3. Harbor Trivy scans the image automatically.
4. Kyverno verifies image is signed (or from allowed registry).
5. Customer triggers deploy from Web Platform (select image + tag).
6. zenith-api updates the Deployment with new image reference.

**Rollback:**

1. Customer clicks "Rollback" on a previous successful deploy in the Web Platform.
2. zenith-api updates Deployment to previous image tag.
3. Kubernetes rolling update replaces current pods with previous version.
4. Total rollback time: ~10-30 seconds.

---

### Database Management

Customers create and manage PostgreSQL databases through the Web Platform.

```
DATABASE CREATION FLOW
=======================

  Customer opens Web Platform -> Databases -> "New Database"
       |
       | POST /api/v1/databases
       | Body: { "name": "my-app-db" }
       v
  +-------------------+
  | zenith-api        |   Validates:
  |                   |     - Name is valid (alphanumeric + hyphens)
  |                   |     - Customer hasn't exceeded tier DB limit
  |                   |       (Free: 1 DB, Pro: 5 DBs)
  |                   |     - Database name is unique for customer
  +--------+----------+
           |
           v
  +-------------------+
  | zenith-api        |   Connects to assigned CNPG cluster:
  | executes SQL      |     - Free -> free-pg primary
  |                   |     - Pro  -> pro-shard-N primary
  |                   |
  |                   |   SQL:
  |                   |     CREATE DATABASE "customer_<slug>_my_app_db";
  |                   |     CREATE USER "customer_<slug>_my_app_db"
  |                   |       WITH PASSWORD '<generated>';
  |                   |     GRANT ALL ON DATABASE ... TO ...;
  +--------+----------+
           |
           v
  +-------------------+
  | zenith-api        |   Creates/updates K8s Secret in customer NS:
  | stores creds      |     db-my-app-db-credentials:
  |                   |       DB_HOST: free-pg-rw.zenith-shared.svc
  |                   |       DB_PORT: 5432
  |                   |       DB_NAME: customer_<slug>_my_app_db
  |                   |       DB_USER: customer_<slug>_my_app_db
  |                   |       DB_PASSWORD: <generated>
  +--------+----------+
           |
           v
  Customer sees in Web Platform:
  +---------------------------------------------------+
  | Databases                                          |
  | -------------------------------------------------- |
  | my-app-db  | 0 MB | 0 connections | [Connect Info] |
  +---------------------------------------------------+
  | "Connect Info" shows the Secret name and           |
  | environment variables to add to their deployment.  |
  +---------------------------------------------------+
```

**Customer uses the database in their application:**

The customer adds environment variable references to their app configuration via the Web Platform:

```yaml
# Zenith auto-generates this in the Deployment spec:
env:
  - name: DB_HOST
    valueFrom:
      secretKeyRef:
        name: db-my-app-db-credentials
        key: DB_HOST
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: db-my-app-db-credentials
        key: DB_PASSWORD
```

The customer's application code connects using standard PostgreSQL drivers with these environment variables. No special SDK required.

---

### Storage Management (S3)

Every customer gets an S3 bucket automatically at signup. They manage files through the Web Platform or directly via S3-compatible APIs.

```
S3 FILE UPLOAD FLOW
====================

  Option A: Via Web Platform UI (simple)
  ----------------------------------------
  Customer opens Web Platform -> Storage -> "Upload File"
       |
       | Browser uploads file to zenith-api
       v
  +-------------------+
  | zenith-api        |   Validates:
  |                   |     - File size within tier limit
  |                   |     - Total storage within tier quota
  |                   |     - File type allowed
  +--------+----------+
           |
           | zenith-api uploads to Hetzner S3
           v
  +-------------------+
  | Hetzner S3        |   Bucket: zenith-<slug>-data
  | customer bucket   |   Key: uploads/<filename>
  +-------------------+


  Option B: Via S3 API (programmatic)
  ----------------------------------------
  Customer gets S3 credentials from Web Platform -> Storage -> "API Keys"
       |
       | Uses any S3-compatible SDK or CLI
       v
  +-------------------+
  | Customer app      |   const s3 = new S3Client({
  | (their code)      |     endpoint: "https://s3.hetzner.cloud",
  |                   |     credentials: {
  |                   |       accessKeyId: process.env.S3_ACCESS_KEY,
  |                   |       secretAccessKey: process.env.S3_SECRET_KEY,
  |                   |     }
  |                   |   });
  |                   |   await s3.putObject({ Bucket: "zenith-<slug>-data", ... });
  +-------------------+
```

**Storage quotas by tier:**

| Tier | Storage Limit | Max File Size | Buckets |
|------|--------------|---------------|---------|
| Free | 1 GB | 50 MB | 1 |
| Pro | 50 GB | 500 MB | 3 |
| Team | 500 GB | 5 GB | Unlimited |
| Enterprise | Custom | Custom | Unlimited |

---

### Custom Domain Setup

Pro, Team, and Enterprise customers can use their own domains instead of `<slug>.freezenith.com`.

```
CUSTOM DOMAIN SETUP FLOW
==========================

  Customer opens Web Platform -> Settings -> "Custom Domain"
       |
       | Enters: app.their-company.com
       v
  +-------------------+
  | Web Platform      |   Shows instructions:
  |                   |   "Add a CNAME record pointing to:"
  |                   |   app.their-company.com -> proxy.freezenith.com
  +-------------------+
           |
           | Customer adds CNAME in their DNS provider
           v
  +-------------------+
  | Customer clicks   |   POST /api/v1/domains
  | "Verify Domain"   |   Body: { "domain": "app.their-company.com" }
  +--------+----------+
           |
           v
  +-------------------+
  | zenith-api        |   1. DNS lookup: verify CNAME points to
  |                   |      proxy.freezenith.com
  |                   |   2. If not: return error "CNAME not found"
  |                   |   3. If yes: proceed
  +--------+----------+
           |
           v
  +-------------------+
  | zenith-api        |   4. Create Certificate CR:
  | provisions TLS    |      - name: app-their-company-com-tls
  |                   |      - issuer: letsencrypt-prod
  |                   |      - challenge: DNS-01 (Cloudflare solver)
  |                   |   5. Wait for cert-manager to issue cert
  +--------+----------+
           |
           v
  +-------------------+
  | zenith-api        |   6. Create/update Traefik IngressRoute:
  | updates routing   |      - match: Host(`app.their-company.com`)
  |                   |      - route to: customer-frontend:3000
  |                   |   7. Create/update APISIX route:
  |                   |      - api.their-company.com -> backend
  +--------+----------+
           |
           v
  Customer sees: "Custom domain verified and active!"
  https://app.their-company.com now serves their frontend
  https://api.their-company.com now routes to their backend
```

**Note on DNS-01 challenges:** Zenith uses DNS-01 instead of HTTP-01 for TLS certificates. This means cert-manager uses the Cloudflare API to create a TXT record for the challenge. For custom domains, this only works if the customer's domain CNAME points to our Cloudflare-proxied domain. The Cloudflare DNS-01 solver handles the challenge on our zone.

---

### Tier Upgrade

Customers can upgrade their tier at any time through the Web Platform. Payment is handled via Stripe.

```
TIER UPGRADE FLOW (Free -> Pro)
================================

  Customer opens Web Platform -> Settings -> "Upgrade Plan"
       |
       | Selects "Pro" tier
       v
  +-------------------+
  | Web Platform      |   Shows comparison:
  | upgrade page      |     Free:  1 DB, 1 GB S3, no custom domain
  |                   |     Pro:   5 DB, 50 GB S3, custom domain, MFA
  |                   |   Price: $29/month
  +--------+----------+
           |
           | Clicks "Upgrade to Pro"
           v
  +-------------------+
  | Stripe Checkout   |   Stripe hosted payment page
  |                   |   Customer enters card details
  |                   |   Stripe creates subscription
  +--------+----------+
           |
           | Webhook: checkout.session.completed
           v
  +-------------------+
  | zenith-api        |   Validates Stripe webhook signature
  | receives webhook  |   Updates customer tier in platform DB
  +--------+----------+
           |
           | Starts Temporal workflow: upgrade-customer-tier
           v
  +---------------------------------------------------+
  | Activity 1: MigrateDatabase            [~60s]     |
  |   - pg_dump from free-pg cluster                   |
  |   - pg_restore to assigned pro shard               |
  |   - Verify data integrity                          |
  |   - Update Secret with new connection string       |
  |   - DROP old database from free-pg                 |
  |                                                    |
  | Activity 2: UpdateResourceQuota        [~2s]       |
  |   - CPU: 500m -> 2000m                             |
  |   - Memory: 512Mi -> 4Gi                           |
  |   - Pods: 5 -> 20                                  |
  |                                                    |
  | Activity 3: UpdateRateLimits           [~2s]       |
  |   - APISIX rate-limit: 100 req/min -> 1000 req/min|
  |                                                    |
  | Activity 4: EnableProFeatures          [~5s]       |
  |   - Keycloak: enable MFA configuration             |
  |   - Enable custom domain capability                |
  |   - Enable additional database creation (up to 5)  |
  |                                                    |
  | Activity 5: NotifyUpgradeComplete      [~1s]       |
  |   - Email: "Welcome to Zenith Pro!"                |
  |   - WebSocket: "Upgrade complete"                  |
  +---------------------------------------------------+
           |
           v
  Customer sees: "You're now on Zenith Pro!"
  New features immediately available in Web Platform
```

---

## Zenith Developer Flows

The Zenith Developer works on the platform itself: the Go API, the Next.js frontends, the Helm charts, the Terraform modules. Their workflow is designed around the `lich` CLI and GitOps.

### Local Development

```
LOCAL DEVELOPMENT FLOW
=======================

  Developer clones repo
       |
       | lich start
       v
  +---------------------------------------------------+
  | Local environment starts:                          |
  |                                                    |
  |  +----------------+  +----------------+            |
  |  | PostgreSQL     |  | Redis          |            |
  |  | (Docker)       |  | (Docker)       |            |
  |  | port: 5432     |  | port: 6379     |            |
  |  +----------------+  +----------------+            |
  |                                                    |
  |  +----------------+  +----------------+            |
  |  | zenith-api     |  | Temporal       |            |
  |  | (go run)       |  | (Docker)       |            |
  |  | port: 8080     |  | port: 7233     |            |
  |  +----------------+  +----------------+            |
  |                                                    |
  |  +----------------+  +----------------+            |
  |  | zenith-web     |  | zenith-admin   |            |
  |  | (next dev)     |  | (next dev)     |            |
  |  | port: 3000     |  | port: 3100     |            |
  |  +----------------+  +----------------+            |
  |                                                    |
  |  +----------------+                                |
  |  | zenith-landing |                                |
  |  | (next dev)     |                                |
  |  | port: 3200     |                                |
  |  +----------------+                                |
  +---------------------------------------------------+
           |
           v
  Developer opens:
    http://localhost:3000  (Web Platform)
    http://localhost:3100  (Admin Panel)
    http://localhost:3200  (Landing Page)
    http://localhost:8080  (API -- Swagger/OpenAPI)
```

**Key points about local development:**

- `lich start` uses Docker Compose for infrastructure (Postgres, Redis, Temporal) and runs application code natively for fast hot-reload.
- The Go API uses air for live-reload on file changes.
- Next.js apps use built-in hot module replacement.
- Local Keycloak is optional -- developers can use mock auth for most development.
- Environment variables are loaded from `.env.local` (gitignored).

### CI/CD Pipeline

```
CI/CD PIPELINE
===============

  Developer pushes to feature branch
       |
       | GitHub webhook triggers CI
       v
  +---------------------------------------------------+
  | GitHub Actions (or local via: lich ci backend -l)  |
  |                                                    |
  | Stage 1: Lint + Type Check           [~30s]        |
  |   - Go: golangci-lint                              |
  |   - TypeScript: eslint + tsc --noEmit              |
  |   - Terraform: terraform fmt + validate            |
  |   - Ansible: ansible-lint                          |
  |                                                    |
  | Stage 2: Unit Tests                  [~60s]        |
  |   - Go: go test ./... -race -cover                 |
  |   - TypeScript: vitest                             |
  |                                                    |
  | Stage 3: Integration Tests           [~120s]       |
  |   - Go: tests with test containers (Postgres)      |
  |   - API: endpoint tests with httptest              |
  |                                                    |
  | Stage 4: Build                       [~60s]        |
  |   - Docker build all images                        |
  |   - Trivy vulnerability scan                       |
  |   - cosign sign images                             |
  +---------------------------------------------------+
       |
       | All stages pass -> PR is mergeable
       v
  +-------------------+
  | Pull Request      |   Reviewer checks:
  | Review            |     - Code quality
  |                   |     - Test coverage
  |                   |     - Architecture compliance
  +--------+----------+
           |
           | Merge to main
           v
  +---------------------------------------------------+
  | ArgoCD detects change in main branch               |
  |                                                    |
  | For staging:                                       |
  |   - Auto-sync enabled                              |
  |   - ArgoCD pulls latest, applies to staging cluster|
  |   - Developer verifies in staging                  |
  |                                                    |
  | For production:                                    |
  |   - Manual sync (or tag-triggered)                 |
  |   - Developer creates git tag: v1.2.3              |
  |   - CI builds production images with tag           |
  |   - ArgoCD Image Updater detects new tag           |
  |   - ArgoCD syncs to production cluster             |
  +---------------------------------------------------+
```

**Deploy to staging (automatic):**

1. Developer merges PR to `main`.
2. CI builds new images, pushes to Harbor.
3. ArgoCD detects image change (Image Updater watches Harbor).
4. ArgoCD updates staging Deployment with new image tag.
5. Kubernetes rolling update deploys new pods.
6. Developer verifies at `https://staging.freezenith.com`.

**Deploy to production (tag-triggered):**

1. Developer creates release: `git tag v1.2.3 && git push --tags`.
2. CI builds production images tagged `v1.2.3`.
3. ArgoCD syncs production to the tagged version.
4. Canary or blue-green deployment (configurable per service).

---

### Adding New Features

When a Zenith developer needs to add a new capability to the platform.

```
ADDING A NEW FEATURE
======================

  Example: Add "Environment Variables" management for customers

  Step 1: Scaffold
  +-------------------+
  | lich make service  |   Generates:
  |   env-vars        |     - services/api/internal/envvars/handler.go
  |                   |     - services/api/internal/envvars/service.go
  |                   |     - services/api/internal/envvars/repository.go
  |                   |     - services/api/internal/envvars/models.go
  |                   |     - services/api/internal/envvars/handler_test.go
  +--------+----------+
           |
  Step 2: Implement
  +-------------------+
  | Developer writes  |   - Define models (EnvVar struct)
  | business logic    |   - Implement repository (Postgres queries)
  |                   |   - Implement service (validation + orchestration)
  |                   |   - Implement handler (HTTP endpoints)
  |                   |   - Wire into router (main.go)
  +--------+----------+
           |
  Step 3: Test
  +-------------------+
  | lich test         |   - Unit tests (service logic)
  | lich ci backend -l|   - Integration tests (with test DB)
  |                   |   - API tests (endpoint behavior)
  +--------+----------+
           |
  Step 4: Frontend
  +-------------------+
  | Add UI in         |   - New page: /env-vars
  | zenith-web        |   - API client for new endpoints
  |                   |   - Form for CRUD operations
  +--------+----------+
           |
  Step 5: Deploy
  +-------------------+
  | git push          |   - CI validates
  | create PR         |   - Review + merge
  | ArgoCD syncs      |   - Live in staging
  +-------------------+
```

---

### Debugging in Staging

When something is broken in staging and the developer needs to investigate.

```
DEBUGGING IN STAGING
=====================

  Developer notices issue (or alert fires)
       |
       v
  +-------------------+
  | Step 1: Logs      |   Check Loki via Grafana:
  |                   |     - Filter by namespace, pod, container
  |                   |     - Search for error patterns
  |                   |     - Time-range around incident
  |                   |
  |                   |   Or via kubectl:
  |                   |     kubectl logs -n zenith-platform \
  |                   |       deployment/zenith-api --tail=200
  +--------+----------+
           |
           v
  +-------------------+
  | Step 2: Traces    |   Check Tempo via Grafana:
  |                   |     - Find slow or failed requests
  |                   |     - Trace spans across services
  |                   |     - Identify bottleneck service
  +--------+----------+
           |
           v
  +-------------------+
  | Step 3: Metrics   |   Check Prometheus via Grafana:
  |                   |     - Pod CPU/memory over time
  |                   |     - Request rate and error rate
  |                   |     - Database connection pool
  +--------+----------+
           |
           v
  +-------------------+
  | Step 4: Network   |   Check Hubble via Hubble UI:
  |                   |     - Network flow map
  |                   |     - Dropped packets (policy denials)
  |                   |     - DNS resolution issues
  +--------+----------+
           |
           v
  +-------------------+
  | Step 5: Fix       |   - Fix code locally
  |                   |   - Push to branch
  |                   |   - CI validates
  |                   |   - Merge -> ArgoCD auto-deploys to staging
  |                   |   - Verify fix in staging
  +-------------------+
```

**Common debugging scenarios:**

| Symptom | First Check | Likely Cause |
|---------|------------|--------------|
| 502 Bad Gateway | Pod logs + readiness probe | Pod crashing, OOM, wrong port |
| 401 Unauthorized | APISIX logs + Keycloak health | Keycloak down, JWKS cache stale, token expired |
| Slow responses | Tempo traces + DB metrics | Slow SQL query, missing index, connection pool exhausted |
| Intermittent failures | Hubble flows + Cilium policies | Network policy blocking, DNS resolution timeout |
| Deploy stuck | ArgoCD UI + pod events | Image pull error, resource quota exceeded, PVC pending |

---

## Request Flow Diagrams

These diagrams show exactly what happens when a request travels through the Zenith platform, from the user's browser to the backend pod and back.

### Frontend Request Path

When a customer (or their end user) visits the customer's frontend application.

```
FRONTEND REQUEST PATH
======================

  User's browser
       |
       | GET https://acme.freezenith.com/dashboard
       v
  +-------------------+
  | Cloudflare        |   1. DDoS check (pass)
  | Edge              |   2. WAF rules (pass)
  |                   |   3. Cache check (miss for HTML, hit for static)
  |                   |   4. Proxy to origin: Hetzner IP
  +--------+----------+
           |
           | HTTPS (Cloudflare -> Hetzner, Full Strict SSL)
           v
  +-------------------+
  | Traefik           |   5. TLS termination (Let's Encrypt cert)
  | (k3s built-in)    |   6. Match IngressRoute:
  |                   |        Host(`acme.freezenith.com`)
  |                   |   7. Route to Service: customer-frontend
  |                   |      in namespace: zenith-acme
  +--------+----------+
           |
           | HTTP (cluster-internal, Cilium WireGuard encrypted)
           v
  +-------------------+
  | customer-frontend |   8. Next.js serves HTML page
  | pod               |   9. Browser receives HTML + JS bundle
  | (zenith-acme NS)  |  10. Browser renders dashboard
  +-------------------+

  Total hops: 3 (Cloudflare -> Traefik -> Pod)
  No APISIX involved for frontend requests.
  Typical latency: 50-150ms (depends on user location + Cloudflare edge)
```

**Why no APISIX for frontends?**

Frontends serve static HTML/JS/CSS. There is no authentication needed at the routing layer because:
- The Next.js app handles its own authentication state (tokens in memory).
- Static assets are public by nature.
- Adding APISIX would add unnecessary latency (~5-10ms per hop).
- Traefik is already present (k3s built-in) and handles TLS efficiently.

---

### Backend Request Path

When a customer's frontend makes an API call to the backend.

```
BACKEND REQUEST PATH (Authenticated)
======================================

  Customer frontend (browser)
       |
       | POST https://api.acme.freezenith.com/v1/projects
       | Headers:
       |   Authorization: Bearer eyJhbGciOiJSUzI1...
       |   Content-Type: application/json
       |   Origin: https://acme.freezenith.com
       v
  +-------------------+
  | Cloudflare        |   1. DDoS check (pass)
  | Edge              |   2. WAF rules (pass -- checks for SQLi, XSS)
  |                   |   3. Edge rate-limit check (pass)
  |                   |   4. Proxy to origin
  +--------+----------+
           |
           v
  +-------------------+
  | Traefik           |   5. TLS termination
  |                   |   6. Match IngressRoute:
  |                   |        Host(`api.acme.freezenith.com`)
  |                   |        ingressClass: apisix
  |                   |   7. Forward to APISIX service
  +--------+----------+
           |
           v
  +-------------------+
  | APISIX            |   8. Match route: /v1/projects
  | (API Gateway)     |   9. Plugin chain executes:
  |                   |
  |                   |   [cors plugin]
  |                   |     - Check Origin header
  |                   |     - Verify origin is in allowed list:
  |                   |       https://acme.freezenith.com
  |                   |     - Add CORS response headers
  |                   |
  |                   |   [jwt-auth plugin]
  |                   |     - Extract Bearer token
  |                   |     - Fetch JWKS from Keycloak (cached):
  |                   |       auth.freezenith.com/realms/acme/
  |                   |       protocol/openid-connect/certs
  |                   |     - Verify RS256 signature
  |                   |     - Check claims: exp, iss, aud
  |                   |     - If invalid: return 401
  |                   |     - If valid: add consumer headers
  |                   |
  |                   |   [limit-count plugin]
  |                   |     - Check rate: 1000 req/min (Pro tier)
  |                   |     - If exceeded: return 429
  |                   |
  |                   |   [opentelemetry plugin]
  |                   |     - Create trace span
  |                   |     - Inject trace context headers
  |                   |
  |                   |  10. Forward to upstream:
  |                   |      customer-backend.zenith-acme.svc:8080
  +--------+----------+
           |
           | Headers added by APISIX:
           |   X-Consumer-Username: alice@acme.com
           |   X-Consumer-Realm: acme
           |   X-Consumer-Roles: admin
           |   X-Request-Id: <uuid>
           |   traceparent: 00-<trace-id>-<span-id>-01
           v
  +-------------------+
  | Cilium            |  11. Network policy check:
  | (CNI)             |      - Source: apisix namespace (allowed)
  |                   |      - Dest: zenith-acme, port 8080 (allowed)
  |                   |      - Protocol: HTTP (L7 allowed)
  |                   |  12. WireGuard encrypts pod-to-pod traffic
  +--------+----------+
           |
           v
  +-------------------+
  | customer-backend  |  13. Receives verified request
  | pod               |  14. Reads X-Consumer-Realm: acme
  | (zenith-acme NS)  |  15. Scopes all DB queries to customer_acme DB
  |                   |  16. Processes business logic
  |                   |  17. Returns JSON response
  +-------------------+
           |
           | Response travels back through the same path
           v
  Browser receives:
    HTTP 201 Created
    { "id": "proj_123", "name": "My Project" }

  Total hops: 5 (Cloudflare -> Traefik -> APISIX -> Cilium -> Pod)
  Typical latency: 80-200ms
```

**Backend request path (public / unauthenticated):**

For endpoints that do not require authentication (webhooks, unsubscribe links):

```
  Same path as above, but at APISIX:
    - Route matches a public route (e.g., /v1/webhooks/*)
    - jwt-auth plugin is NOT configured on this route
    - cors plugin still applies
    - limit-count still applies (tighter limits for public endpoints)
    - Request forwarded to backend without X-Consumer-* headers
    - Backend must validate webhook signatures itself
```

---

### Cross-Service Communication

When one pod needs to communicate with another pod within the cluster.

```
CROSS-SERVICE COMMUNICATION
=============================

  Example: zenith-api needs to call Keycloak Admin API

  zenith-api pod (zenith-platform NS)
       |
       | HTTPS POST keycloak.keycloak.svc.cluster.local:8443
       |   /admin/realms/acme/users
       v
  +-------------------+
  | Cilium            |   1. Check CiliumNetworkPolicy:
  | (CNI layer)       |      - Source: zenith-platform/zenith-api
  |                   |      - Dest: keycloak/keycloak, port 8443
  |                   |      - ALLOW (explicit egress rule)
  |                   |
  |                   |   2. WireGuard encryption:
  |                   |      - Transparent to pods
  |                   |      - All pod-to-pod traffic encrypted
  |                   |
  |                   |   3. Hubble records flow:
  |                   |      - Source/dest pod labels
  |                   |      - Port, protocol, verdict (allowed)
  |                   |      - Latency (for flow metrics)
  +--------+----------+
           |
           v
  +-------------------+
  | Keycloak pod      |   4. Receives request
  | (keycloak NS)     |   5. Authenticates via service account token
  |                   |   6. Processes admin API call
  |                   |   7. Returns response
  +-------------------+


  Example: Customer pod trying to access another customer's pod

  customer-a-backend (zenith-customer-a NS)
       |
       | TCP to customer-b-backend.zenith-customer-b.svc:8080
       v
  +-------------------+
  | Cilium            |   1. Check CiliumNetworkPolicy:
  | (CNI layer)       |      - Source: zenith-customer-a/*
  |                   |      - Dest: zenith-customer-b/*
  |                   |      - DENY (default deny + no explicit allow)
  |                   |
  |                   |   2. Packet dropped
  |                   |   3. Hubble records: DROPPED (policy)
  |                   |   4. If repeated: Falco alert
  +-------------------+

  Result: Connection refused. Customers cannot communicate
  with each other. Each namespace is an isolated island.
```

**What each pod is allowed to communicate with (CiliumNetworkPolicy):**

```
Customer pod (zenith-<slug> namespace):
  Egress allowed to:
    +-- kube-dns (DNS resolution)                   [53/UDP]
    +-- APISIX (if backend needs to call own API)   [9080/TCP]
    +-- Keycloak (token validation)                 [8443/TCP]
    +-- CNPG cluster (database)                     [5432/TCP]
    +-- Hetzner S3 (external, via egress allow)     [443/TCP]

  Egress denied to:
    +-- Other customer namespaces                   [BLOCKED]
    +-- Kubernetes API server                       [BLOCKED]
    +-- Node network (host namespace)               [BLOCKED]
    +-- Infrastructure namespaces                   [BLOCKED]
      (except explicitly allowed services above)

  Ingress allowed from:
    +-- Traefik (frontend traffic)                  [3000/TCP]
    +-- APISIX (backend traffic)                    [8080/TCP]

  Ingress denied from:
    +-- Other customer namespaces                   [BLOCKED]
    +-- Direct internet access                      [BLOCKED]
```

---

## Error Handling and Edge Cases

Real systems fail. This section documents what happens when things go wrong at each stage of the user flows.

### Signup Failures

| Failure Point | What Happens | User Sees | Recovery |
|---------------|-------------|-----------|----------|
| Email already exists | zenith-api rejects immediately | "Email already registered" | Login instead |
| Keycloak realm creation fails | Temporal retries 3x | Progress bar stalls at "Creating identity" | Auto-retry; if all fail, admin alert |
| Database creation fails (shard full) | Temporal retries on next shard | Progress bar stalls at "Provisioning database" | Auto-failover to next shard; if all full, admin alert |
| S3 bucket creation fails | Temporal retries 3x | Progress bar stalls at "Creating storage" | Hetzner API issue; auto-retry |
| DNS propagation timeout | Temporal waits up to 5 min | "Configuring networking (this may take a moment)" | Usually resolves; if not, admin investigates Cloudflare |
| TLS cert issuance timeout | Temporal waits up to 10 min | "Securing with TLS (this may take a moment)" | DNS-01 challenge may be slow; auto-retry |
| Any permanent failure | Temporal saga rollback | "Setup failed. Our team has been notified." | Admin investigates; customer can retry or contact support |

### Deployment Failures

| Failure Point | What Happens | User Sees | Recovery |
|---------------|-------------|-----------|----------|
| Git clone fails | Temporal retries | "Build failed: could not clone repository" | Customer checks repo URL and access |
| Docker build fails | Build logs captured | "Build failed" + build log | Customer fixes Dockerfile |
| Trivy finds critical CVE | Build blocked by policy | "Image blocked: critical vulnerability found" | Customer updates base image |
| Image push to Harbor fails | Temporal retries | "Build failed: registry error" | Transient; auto-retry |
| Deployment rollout fails | Kubernetes rollback | "Deploy failed: health check timeout" | Customer checks app startup; auto-rollback to previous |
| Resource quota exceeded | Deployment rejected | "Deploy failed: quota exceeded" | Customer reduces resource requests or upgrades tier |

### Authentication Failures

| Failure Point | What Happens | User Sees | Recovery |
|---------------|-------------|-----------|----------|
| Keycloak down | APISIX returns 401 (JWKS unavailable) | "Service temporarily unavailable" | Admin restarts Keycloak pod; JWKS cached for short period |
| Token expired | APISIX returns 401 | Frontend auto-refreshes token | Transparent to user if refresh token valid |
| Refresh token expired | Keycloak rejects refresh | Redirect to login page | User logs in again |
| CORS mismatch | Browser blocks response | Network error in console | Customer checks domain configuration |
| Rate limit exceeded | APISIX returns 429 | "Too many requests, please slow down" | Wait and retry; upgrade tier for higher limits |

---

## Summary: Key Principles

The user flows in Zenith V2 are designed around several core principles:

1. **Automation over manual intervention.** Customer signup, deployment, database creation, TLS provisioning -- all automated via Temporal workflows. The admin should only intervene for exceptional cases.

2. **Isolation by default.** Every customer gets their own Keycloak realm, their own database, their own S3 bucket, their own namespace with deny-all network policy. Nothing is shared unless explicitly designed to be (like the CNPG operator or APISIX gateway).

3. **Observability at every layer.** Every request is traced (Tempo), every log is collected (Loki), every metric is scraped (Prometheus), every network flow is recorded (Hubble). When something fails, the admin has the tools to find out why within minutes.

4. **Graceful degradation.** Temporal retries failed activities. Kubernetes restarts crashed pods. APISIX caches JWKS keys. ArgoCD self-heals drift. The platform recovers from transient failures without human intervention.

5. **Clear tier boundaries.** Free customers get limited but functional environments. Each tier upgrade unlocks specific capabilities (more databases, custom domains, higher rate limits, MFA). The upgrade path is smooth and automated.

6. **Developer experience first.** Whether it is the Zenith developer using `lich start` for local development, or the customer developer pushing code via Git, the goal is minimal friction. Push code, get a running application.

---

*Related documents:*
- [00-overview.md](./00-overview.md) -- Platform architecture overview
- [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) -- Cluster bootstrap details
- [06-security-model.md](./06-security-model.md) -- Defense-in-depth security architecture
- [09-migration-v1-to-v2.md](./09-migration-v1-to-v2.md) -- Migration plan from V1 to V2
