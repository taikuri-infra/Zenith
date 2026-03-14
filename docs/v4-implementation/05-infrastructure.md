# 05 — Infrastructure Guide (Terraform, Helm, Ansible, K8s)

> **Read time:** 90 minutes
> **Prerequisite:** [04 — Frontend Complete Guide](./04-frontend-guide.md)
> **Next:** [06 — CI/CD & Deployment Guide](./06-cicd-guide.md)

---

## Infrastructure at a Glance

```
infra/
├── terraform/               ← Infrastructure as Code (49 files)
│   ├── staging/              ← Phase 1: Hetzner VM + Cloudflare DNS
│   ├── staging-k8s/          ← Phase 3: K8s platform bootstrap
│   ├── production/           ← Phase 1: Production VMs
│   ├── production-k8s/       ← Phase 3: Production K8s
│   ├── production-dr/        ← DR cluster
│   └── modules/              ← Reusable Terraform modules
│       ├── k8s-platform/     ← 25 .tf files (THE MAIN MODULE)
│       ├── k3s-server/       ← Hetzner VM provisioning
│       ├── dns/              ← Cloudflare DNS records
│       └── storage/          ← Hetzner S3 buckets
│
├── helm/                     ← 11 Helm charts, 105 templates
│   ├── zenith-api/           ← Go API server
│   ├── zenith-web/           ← Web dashboard
│   ├── zenith-mc/            ← Mission Control
│   ├── zenith-landing/       ← Landing page
│   ├── zenith-platform/      ← Shared platform resources (CNPG, secrets)
│   ├── zenith-operator/      ← K8s operator
│   ├── zenith-tenant/        ← Per-customer tenant
│   ├── zenith-demo/          ← Demo instances
│   ├── zenith/               ← Umbrella meta-chart
│   └── monitoring/           ← Prometheus stack wrapper
│
├── ansible/                  ← Server provisioning
│   ├── playbooks/            ← site.yml, infra.yml, apps.yml
│   ├── roles/                ← 20+ roles (k3s, cert-manager, cilium, etc.)
│   └── inventory/            ← staging.yml, production.yml
│
└── scripts/                  ← Smoke tests, utilities
    ├── smoke-test-customer.sh
    ├── smoke-test-owner.sh
    └── smoke-test-infra.sh
```

---

## Terraform — The k8s-platform Module (Heart of Infrastructure)

Located at `infra/terraform/modules/k8s-platform/`, this module installs EVERYTHING into the K8s cluster.

### 25 Terraform Files

| File | What It Creates | Key Resources |
|------|----------------|---------------|
| `main.tf` | Base setup | PriorityClasses (system-critical, infra-critical, platform, customer) |
| `certmanager.tf` | TLS automation | helm_release (cert-manager) + ClusterIssuer (letsencrypt-prod) |
| `sealed_secrets.tf` | Encrypted secrets | helm_release (sealed-secrets) |
| `storage.tf` | Data layer | helm_release (cnpg-operator) + CNPG Clusters (keycloak-pg, free-pg) + ScheduledBackups |
| `identity.tf` | Identity provider | helm_release (keycloak) with external CNPG database |
| `gateway.tf` | API gateway | helm_release (apisix + apisix-ingress-controller) + helm_release (external-dns) |
| `traefik.tf` | Ingress config | HelmChartConfig (crossNamespace, externalNameServices) |
| `apps.tf` | Zenith apps | helm_release (zenith-platform, zenith-api, zenith-landing) |
| `gitops.tf` | GitOps engine | helm_release (argocd + argocd-image-updater) |
| `registry.tf` | Container registry | helm_release (harbor) with S3 backend |
| `temporal.tf` | Workflow engine | helm_release (temporal) with PostgreSQL persistence |
| `observability.tf` | Monitoring stack | helm_release (kube-prometheus-stack, loki, tempo, otel-collector) |
| `observability-operators.tf` | Operator versions | Loki Operator, OTel Operator, Tempo Operator |
| `security.tf` | Security tools | helm_release (kyverno, falco, velero) |
| `identity-operators.tf` | Identity operators | Keycloak Operator, OAuth2-Proxy |
| `messaging-operators.tf` | Data operators | Redis Operator, RabbitMQ Operator, Strimzi Kafka |
| `autoscaling.tf` | Auto-scaling | helm_release (keda, keda-http-addon) |
| `pdb.tf` | HA protection | PodDisruptionBudgets for critical services |
| `auth_secrets.tf` | Encrypted secrets | SealedSecrets (Resend API key, Google OAuth) |
| `variables.tf` | Configuration | 560+ lines of variable definitions |
| `outputs.tf` | Status outputs | All Helm release statuses |

### Feature Flags (Enable/Disable Components)

```hcl
# In staging-k8s/variables.tf
variable "enable_apisix"        { default = true }
variable "enable_external_dns"  { default = true }
variable "enable_argocd"        { default = true }
variable "enable_harbor"        { default = true }
variable "enable_keycloak"      { default = true }
variable "enable_temporal"      { default = true }
variable "enable_sealed_secrets" { default = true }
variable "enable_kyverno"       { default = true }
variable "enable_falco"         { default = true }
variable "enable_velero"        { default = true }
variable "enable_keda"          { default = false }
variable "enable_monitoring"    { default = false }
variable "enable_cnpg"          { default = false }
variable "enable_v3_operators"  { default = false }
```

### Running Terraform

```bash
# Staging infrastructure (VMs + DNS)
cd infra/terraform/staging
terraform init && terraform plan && terraform apply

# Staging K8s platform (all components)
cd infra/terraform/staging-k8s
terraform init && terraform plan && terraform apply

# Just plan (safe — shows what would change)
terraform plan -var-file=terraform.tfvars
```

---

## Helm Charts — Application Packaging

### zenith-api Chart

```
infra/helm/zenith-api/
├── Chart.yaml                    ← version: 0.4.1, appVersion: 0.8.2
├── values.yaml                   ← Default values
├── values-staging.yaml           ← Staging overrides
├── values-production.yaml        ← Production overrides
└── templates/
    ├── deployment.yaml           ← API server Deployment
    ├── service.yaml              ← ClusterIP Service (port 8080)
    ├── configmap.yaml            ← Environment variables
    ├── secret.yaml               ← Sensitive environment variables
    ├── hpa.yaml                  ← HorizontalPodAutoscaler
    ├── pdb.yaml                  ← PodDisruptionBudget
    ├── serviceaccount.yaml       ← K8s ServiceAccount
    ├── servicemonitor.yaml       ← Prometheus ServiceMonitor
    ├── ingressroute.yaml         ← Traefik IngressRoute (api.stage.freezenith.com)
    ├── apisix-route.yaml         ← APISIX route (JWT, CORS, rate-limit, WAF)
    └── certificate.yaml          ← cert-manager Certificate
```

### Key Values

```yaml
# values-staging.yaml
namespace: zenith-staging
replicas: 1
image:
  registry: registry.stage.freezenith.com/zenith-stage
  repository: zenith-api
  tag: "sha-abc1234"  # SHA tags for staging
port: 8080
ingress:
  host: api.stage.freezenith.com
  apisixEnabled: true
  apisixCorsOrigins: "https://app.stage.freezenith.com,https://mc.stage.freezenith.com"
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

### APISIX Route Template (Important!)

The `apisix-route.yaml` defines two route types:

```yaml
# PUBLIC routes (no JWT needed)
- name: api-public
  paths:
    - /health
    - /ready
    - /api/v1/auth/*
    - /api/v1/auth-pools/*/signup
    - /api/v1/auth-pools/*/login
    # ... other public auth endpoints
  plugins:
    - cors (allow_origins from values)
    - limit-count (30 req/60s — strict for auth)
    - uri-blocker (SQLi, path traversal, XSS)
    - ua-restriction (block scanners)

# PROTECTED routes (JWT required — handled by API middleware, not APISIX)
- name: api-protected
  paths:
    - /api/*
  plugins:
    - cors
    - limit-count (500 req/60s — generous for authenticated users)
    - uri-blocker (same WAF rules)
    - ua-restriction (same scanner blocking)
```

---

## Ansible — Server Provisioning

### Playbooks

| Playbook | Purpose |
|----------|---------|
| `site.yml` | Full deployment (everything) |
| `infra.yml` | Infrastructure only (k3s, cert-manager) |
| `apps.yml` | Applications only (build + deploy) |
| `build.yml` | Docker builds only |
| `server-setup.yml` | Initial server provisioning |

### Key Roles

| Role | What It Does |
|------|-------------|
| `common` | Base OS setup (apt, Docker, sysctl) |
| `k3s` | Install k3s with Cilium CNI |
| `cert-manager` | Install cert-manager + Let's Encrypt |
| `cilium` | Install Cilium CNI |
| `zenith-build` | Docker build images |
| `zenith-import` | Import images to k3s |
| `zenith-api` | Deploy zenith-api |

### Running Ansible

```bash
cd infra/ansible

# Full deployment
ansible-playbook playbooks/site.yml -i inventory/staging.yml

# Just infrastructure
ansible-playbook playbooks/infra.yml -i inventory/staging.yml

# Just apps
ansible-playbook playbooks/apps.yml -i inventory/staging.yml --tags api,web

# Selective with tags
ansible-playbook playbooks/site.yml -i inventory/staging.yml --tags k3s,cert-manager
```

---

## Kubernetes Resources We Use

### CRDs (Custom Resource Definitions)

| CRD | Operator | Purpose |
|-----|----------|---------|
| `IngressRoute` | Traefik | L7 HTTP routing (NOT standard Ingress!) |
| `Certificate` | cert-manager | TLS certificate requests |
| `ClusterIssuer` | cert-manager | Let's Encrypt issuer |
| `Cluster` | CNPG | PostgreSQL cluster definition |
| `ScheduledBackup` | CNPG | Automated backup schedule |
| `ApisixRoute` | APISIX Ingress Controller | API gateway routes |
| `SealedSecret` | Sealed Secrets | Encrypted secrets for GitOps |
| `Application` | ArgoCD | GitOps application definition |
| `ScaledObject` | KEDA | Custom metric HPA |
| `HTTPScaledObject` | KEDA HTTP | HTTP-based scale-to-zero |
| `ClusterPolicy` | Kyverno | Admission policies |
| `ServiceMonitor` | Prometheus Operator | Metric scraping config |

### Important: IngressRoute vs Ingress

Zenith uses **Traefik IngressRoute CRDs**, NOT standard Kubernetes Ingress.

```yaml
# This is what we use (IngressRoute)
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: zenith-api
spec:
  entryPoints: [websecure]
  routes:
    - match: Host(`api.stage.freezenith.com`)
      kind: Rule
      services:
        - name: zenith-api
          port: 8080
  tls:
    secretName: zenith-api-tls

# This is NOT what we use (standard Ingress)
# apiVersion: networking.k8s.io/v1
# kind: Ingress  ← We don't use this!
```

---

## Networking Architecture

```
Internet → Cloudflare → Traefik (:443)
                           │
    ┌──────────────────────┼──────────────────────────────┐
    │                      │                               │
    ▼                      ▼                               ▼
Frontend Routes         APISIX Routes                 Platform Routes
(direct to pod)         (via ExternalName svc)        (direct to pod)
                           │
stage.freezenith.com    api.stage.freezenith.com    auth.stage.freezenith.com
→ zenith-landing:3000   → apisix-gateway:9080      → keycloak:8080
                        → zenith-api:8080
app.stage.freezenith.com                           argocd.stage.freezenith.com
→ zenith-web:3000                                  → argocd-server:443

mc.stage.freezenith.com                            registry.stage.freezenith.com
→ zenith-mc:3000                                   → harbor-nginx:80

*.apps.stage.freezenith.com                        hub.stage.freezenith.com
→ customer-app-pods                                → customer-harbor:80
```

### TLS Certificates

| Domain | Type | Managed By |
|--------|------|-----------|
| `*.stage.freezenith.com` | Wildcard | cert-manager (DNS-01 via Cloudflare) |
| `*.apps.stage.freezenith.com` | Wildcard | cert-manager (stored as `apps-wildcard-tls`) |
| `*.gw.stage.freezenith.com` | Wildcard | cert-manager (stored as `gw-wildcard-tls`) |
| Custom domains | Individual | cert-manager (HTTP-01, per-gateway Certificate CRD) |

---

## Backup Strategy

| What | How | Schedule | Retention | Destination |
|------|-----|----------|-----------|-------------|
| zenith-postgres | CNPG WAL archiving | Continuous + daily | 14 days | s3://zenith-backups/zenith-postgres-wal/ |
| free-pg | CNPG WAL archiving | Continuous + daily 2am | 14 days | s3://zenith-backups/free-pg-wal/ |
| keycloak-pg | CNPG WAL archiving | Continuous + daily | 14 days | s3://zenith-backups/keycloak-wal/ |
| K8s resources | Velero | Daily | 30 days | s3://zenith-backups/velero/ |
| etcd (APISIX) | CronJob snapshot | Daily | 7 days | s3://zenith-backups/etcd/ |

### Restore Procedure

```bash
# Restore CNPG from backup
kubectl apply -f - <<EOF
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: zenith-postgres-restored
spec:
  instances: 1
  imageName: ghcr.io/cloudnative-pg/postgresql:16.6  # MUST match version!
  bootstrap:
    recovery:
      source: zenith-postgres-backup
  externalClusters:
    - name: zenith-postgres-backup
      barmanObjectStore:
        destinationPath: s3://zenith-backups/zenith-postgres-wal/
        s3Credentials:
          accessKeyId: { name: cnpg-s3-credentials, key: ACCESS_KEY_ID }
          secretAccessKey: { name: cnpg-s3-credentials, key: ACCESS_SECRET_KEY }
        endpointURL: https://fsn1.your-objectstorage.com
EOF
```

**IMPORTANT:** Must specify `imageName: ghcr.io/cloudnative-pg/postgresql:16.6` — default is PG17, causes version mismatch!

---

## SSH Access

| Alias | Server | IP | Purpose |
|-------|--------|------|---------|
| `zen-stage` | Staging | 77.42.88.149 | Staging K8s cluster |
| `ghasi` | Old production | 161.35.82.211 | Legacy (being replaced) |

```bash
# Connect to staging
ssh zen-stage

# Run kubectl on staging
ssh zen-stage kubectl get pods -n zenith-staging

# Copy files to staging
rsync -avz ./file zen-stage:/opt/zenith/
```

**CRITICAL:** No GitHub SSH key on staging server — use `rsync` to copy files, not `git clone`.

---

**Next → [06 — CI/CD & Deployment Guide](./06-cicd-guide.md)**
