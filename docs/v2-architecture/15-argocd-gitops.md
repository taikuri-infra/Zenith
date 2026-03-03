# 15 — ArgoCD GitOps

> **Purpose:** Understand how application deployments are automated via GitOps, including the App-of-Apps pattern, image updater, and sync strategies.
> **Audience:** Any developer who needs to deploy apps, debug sync issues, or understand the CD pipeline.
> **Last Updated:** 2026-03-03
> **Related:** [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) (installation steps), [04-phase4-argocd-apps.md](./04-phase4-argocd-apps.md) (application definitions), [SYSTEM-MAP.md](./SYSTEM-MAP.md) (deployment flow path)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose It](#2-why-we-chose-it)
3. [Architecture Diagram](#3-architecture-diagram)
4. [App-of-Apps Pattern](#4-app-of-apps-pattern)
5. [How a Deployment Works (End to End)](#5-how-a-deployment-works-end-to-end)
6. [Image Updater — Automatic Image Promotion](#6-image-updater--automatic-image-promotion)
7. [Sync Waves — Ordered Deployments](#7-sync-waves--ordered-deployments)
8. [Configuration Reference](#8-configuration-reference)
9. [Troubleshooting](#9-troubleshooting)
10. [Upgrade Path](#10-upgrade-path)

---

## 1. Overview

**ArgoCD** is the GitOps engine that automatically deploys applications to the cluster whenever code changes in Git. It:

- Watches the `staging` branch of the GitHub repository
- Reads Helm charts from `infra/argocd/staging/` (App-of-Apps root)
- Syncs Kubernetes resources to match the Git state
- Auto-heals: if someone manually changes a resource, ArgoCD reverts it
- Auto-prunes: if a resource is removed from Git, ArgoCD deletes it from the cluster

```
The GitOps loop:

  Git (desired state) ───▶ ArgoCD ───▶ K8s cluster (actual state)
       │                      │              │
       │                      │ compare      │
       │                      ◀──────────────┘
       │                      │
       │                if different:
       │                sync (apply changes)
       │                      │
       └──────────────────────┘
```

---

## 2. Why We Chose It

| Feature | ArgoCD | FluxCD | Jenkins CD | Spinnaker |
|---------|--------|--------|------------|-----------|
| GitOps native | Yes | Yes | No (CI tool) | Partial |
| Web UI | Excellent | None | Limited | Complex |
| App-of-Apps pattern | Yes | Yes (Kustomization) | No | No |
| Image auto-update | Built-in (Image Updater) | Built-in | External | External |
| Helm support | First-class | First-class | Plugins | Plugins |
| Sync waves | Yes (ordered deployments) | Dependencies | No | Pipelines |
| RBAC / SSO | Built-in | No | Plugin | Built-in |

**Decision:** ArgoCD has the best UI (critical for debugging), native Helm support, and Babak is preparing for the ArgoCD certification exam.

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ARGOCD IN THE ZENITH CLUSTER                        │
│                         Namespace: argocd                                   │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    ARGOCD COMPONENTS                                   │  │
│  │                                                                       │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ ArgoCD Server (Deployment, 1 replica)                            │ │  │
│  │  │ Port: 80 (HTTP, server.insecure=true — Traefik handles TLS)     │ │  │
│  │  │                                                                  │ │  │
│  │  │ - API server for CLI + Web UI                                    │ │  │
│  │  │ - Serves dashboard at argocd.stage.freezenith.com                │ │  │
│  │  │ Resources: 100m CPU, 256Mi RAM                                   │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  │                                                                       │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ ArgoCD Application Controller (Deployment)                       │ │  │
│  │  │                                                                  │ │  │
│  │  │ - The brain: watches Applications, compares Git vs cluster       │ │  │
│  │  │ - Detects drift and triggers sync                                │ │  │
│  │  │ - Applies Kubernetes manifests to the cluster                    │ │  │
│  │  │ - Respects sync waves for ordered deployments                    │ │  │
│  │  │ Resources: 200m CPU, 512Mi RAM                                   │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  │                                                                       │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ ArgoCD Repo Server (Deployment, 1 replica)                       │ │  │
│  │  │                                                                  │ │  │
│  │  │ - Clones Git repos (GitHub via PAT token)                        │ │  │
│  │  │ - Renders Helm charts → Kubernetes manifests                     │ │  │
│  │  │ - Caches rendered manifests                                      │ │  │
│  │  │ Resources: 100m CPU, 256Mi RAM                                   │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  │                                                                       │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ ArgoCD Image Updater (Deployment)                                │ │  │
│  │  │                                                                  │ │  │
│  │  │ - Polls Harbor for new image tags                                │ │  │
│  │  │ - Registry: registry.stage.freezenith.com                        │ │  │
│  │  │ - Credentials: harbor-image-updater-creds Secret                 │ │  │
│  │  │ - When new tag found → updates Application image override        │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  CONNECTIONS:                                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                                                                       │  │
│  │  ArgoCD ──HTTPS──▶ GitHub (taikuri-infra/Zenith.git)                 │  │
│  │           PAT auth   Branch: staging                                  │  │
│  │                      Path: infra/argocd/staging/                      │  │
│  │                                                                       │  │
│  │  ArgoCD ──HTTPS──▶ K8s API (:6443)                                   │  │
│  │           SA auth    Deploy/sync resources                            │  │
│  │                                                                       │  │
│  │  Image Updater ──HTTPS──▶ Internal Harbor                            │  │
│  │           Robot auth   registry.stage.freezenith.com                   │  │
│  │                        Check for new image tags                       │  │
│  │                                                                       │  │
│  │  Traefik ──HTTP──▶ ArgoCD Server (:80)                               │  │
│  │                     argocd.stage.freezenith.com                        │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. App-of-Apps Pattern

ArgoCD uses the **App-of-Apps** pattern: one root Application manages all child Applications.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    APP-OF-APPS HIERARCHY                                  │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ ROOT APPLICATION: "zenith-apps"                                    │ │
│  │ Project: default                                                   │ │
│  │ Source: github.com/taikuri-infra/Zenith.git                        │ │
│  │ Branch: staging                                                    │ │
│  │ Path: infra/argocd/staging/  (directory, recurse=true)             │ │
│  │ Sync: automated (prune + selfHeal)                                 │ │
│  └────────────────────────────────┬───────────────────────────────────┘ │
│                                   │                                      │
│                                   │ Discovers child Application YAMLs:   │
│                                   │                                      │
│  ┌────────────────────────────────┼───────────────────────────────────┐ │
│  │                                ▼                                    │ │
│  │  infra/argocd/staging/                                             │ │
│  │  ├── zenith-platform.yaml   (sync-wave: 0)  ← deploys first       │ │
│  │  ├── zenith-operator.yaml   (sync-wave: -1) ← deploys even before │ │
│  │  ├── zenith-api.yaml        (sync-wave: 1)  ← deploys after plat  │ │
│  │  ├── zenith-landing.yaml    (sync-wave: 1)                        │ │
│  │  ├── zenith-web.yaml        (sync-wave: 1)                        │ │
│  │  └── zenith-demo.yaml       (sync-wave: 1)                        │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                   │                                      │
│                                   ▼                                      │
│  Each child Application deploys a Helm chart:                            │
│                                                                          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐  │
│  │ zenith-platform   │  │ zenith-api        │  │ zenith-landing       │  │
│  │                   │  │                   │  │                      │  │
│  │ Chart:            │  │ Chart:            │  │ Chart:               │  │
│  │ infra/helm/       │  │ infra/helm/       │  │ infra/helm/          │  │
│  │ zenith-platform   │  │ zenith-api        │  │ zenith-landing       │  │
│  │                   │  │                   │  │                      │  │
│  │ Values:           │  │ Values:           │  │ Values:              │  │
│  │ values-staging    │  │ values-staging    │  │ values-staging       │  │
│  │ .yaml             │  │ .yaml             │  │ .yaml                │  │
│  │                   │  │                   │  │                      │  │
│  │ Namespace:        │  │ Namespace:        │  │ Namespace:           │  │
│  │ zenith-staging    │  │ zenith-staging    │  │ zenith-staging       │  │
│  └──────────────────┘  └──────────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. How a Deployment Works (End to End)

```
Developer                  GitHub            ArgoCD              K8s Cluster
    │                        │                  │                      │
    │  git push (staging)    │                  │                      │
    ├───────────────────────▶│                  │                      │
    │                        │                  │                      │
    │                        │  webhook/poll    │                      │
    │                        │ (every 3 min or  │                      │
    │                        │  on webhook)     │                      │
    │                        ├─────────────────▶│                      │
    │                        │                  │                      │
    │                        │                  │  1. Clone repo       │
    │                        │                  │     (staging branch) │
    │                        │                  │                      │
    │                        │                  │  2. Render Helm      │
    │                        │                  │     charts with      │
    │                        │                  │     values-staging   │
    │                        │                  │     .yaml            │
    │                        │                  │                      │
    │                        │                  │  3. Compare rendered │
    │                        │                  │     manifests vs     │
    │                        │                  │     live cluster     │
    │                        │                  │                      │
    │                        │                  │  4. If different:    │
    │                        │                  │     SYNC             │
    │                        │                  ├─────────────────────▶│
    │                        │                  │                      │
    │                        │                  │  5. Apply manifests  │
    │                        │                  │     respecting sync  │
    │                        │                  │     waves:           │
    │                        │                  │     wave -1: operator│
    │                        │                  │     wave 0: platform │
    │                        │                  │     wave 1: api,     │
    │                        │                  │       landing, web   │
    │                        │                  │                      │
    │                        │                  │  6. Wait for health  │
    │                        │                  │     checks to pass   │
    │                        │                  │                      │
    │                        │                  │  7. Mark as Synced   │
    │                        │                  │     + Healthy        │
    │                        │                  │                      │
    │  See result in         │                  │                      │
    │  ArgoCD UI             │                  │                      │
    ▼                        ▼                  ▼                      ▼

IMPORTANT: ArgoCD watches the "staging" branch, NOT "main"!
Always merge main → staging and push both branches.
```

---

## 6. Image Updater — Automatic Image Promotion

```
┌─────────────────────────────────────────────────────────────────────────┐
│              IMAGE UPDATER FLOW                                          │
│                                                                          │
│  CI Pipeline (GitHub Actions)                                            │
│       │                                                                  │
│       │ 1. Build Docker image                                           │
│       │ 2. Push to Harbor:                                              │
│       │    registry.stage.freezenith.com/zenith-stage/zenith-api:0.4.6  │
│       ▼                                                                  │
│  ┌──────────────────┐                                                   │
│  │ Internal Harbor   │  Image stored with tag 0.4.6                     │
│  │ (separate server) │                                                   │
│  └──────────┬────────┘                                                   │
│             │                                                            │
│             │ ArgoCD Image Updater polls every 2 minutes                 │
│             ▼                                                            │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │ ArgoCD Image Updater                                              │   │
│  │                                                                   │   │
│  │ 1. Poll Harbor API: list tags for zenith-api image               │   │
│  │ 2. Compare: current deployed = 0.4.5, latest = 0.4.6            │   │
│  │ 3. New tag found! → Update Application image override             │   │
│  │ 4. ArgoCD detects Application change → triggers sync              │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│             │                                                            │
│             ▼                                                            │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │ ArgoCD Sync                                                       │   │
│  │                                                                   │   │
│  │ 1. Render Helm chart with new image tag                          │   │
│  │ 2. Update Deployment spec: image = zenith-api:0.4.6             │   │
│  │ 3. Rolling update: new pod starts, old pod terminates            │   │
│  │ 4. Health check passes → Synced + Healthy                        │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  Registry credentials:                                                   │
│    Secret: harbor-image-updater-creds (in argocd namespace)              │
│    Format: robot$username:password                                       │
│    Created by: gitops.tf kubernetes_secret_v1                            │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Sync Waves — Ordered Deployments

```
┌─────────────────────────────────────────────────────────────────────────┐
│              SYNC WAVE EXECUTION ORDER                                    │
│                                                                          │
│  Wave -1 (deploys first):                                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ zenith-operator                                                    │ │
│  │ CRD operator — must be running before any custom resources         │ │
│  │ Wait: until operator pod is Ready                                  │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                         │ Done ✓                                         │
│                         ▼                                                │
│  Wave 0:                                                                 │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ zenith-platform                                                    │ │
│  │ Shared infrastructure: CNPG cluster, Keycloak config, secrets,     │ │
│  │ build namespaces, certificates, middleware                         │ │
│  │ Wait: until all resources are healthy                              │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                         │ Done ✓                                         │
│                         ▼                                                │
│  Wave 1 (all deploy in parallel):                                        │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────────────┐   │
│  │ zenith-api      │ │ zenith-landing  │ │ zenith-web              │   │
│  │ Go backend      │ │ Next.js landing │ │ Next.js dashboard       │   │
│  │                 │ │                 │ │                         │   │
│  │ zenith-demo     │ │                 │ │                         │   │
│  │ Demo instances  │ │                 │ │                         │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────────────┘   │
│                                                                          │
│  Why waves?                                                              │
│    - Platform chart creates the PG cluster that API needs                │
│    - Operator chart installs CRDs that platform chart uses               │
│    - Without waves, API might start before DB exists → crash loop        │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Configuration Reference

### Terraform File

**File:** `infra/terraform/modules/k8s-platform/gitops.tf`

### ArgoCD Settings

| Setting | Value | Purpose |
|---------|-------|---------|
| `server.replicas` | 1 | API/UI server |
| `server.insecure` | true | Traefik handles TLS |
| `repoServer.replicas` | 1 | Git clone + Helm render |
| `controller.resources.cpu` | 200m | App controller CPU |
| `controller.resources.memory` | 512Mi | App controller RAM |
| `extensions.enabled` | true | Allow UI extensions |

### Repository Configuration

| Setting | Value |
|---------|-------|
| URL | `https://github.com/taikuri-infra/Zenith.git` |
| Auth | GitHub PAT (set_sensitive) |
| Type | git |

### Root Application

| Setting | Value |
|---------|-------|
| Name | zenith-apps |
| Project | default |
| Source repo | taikuri-infra/Zenith.git |
| Target revision | `staging` branch |
| Path | `infra/argocd/staging/` |
| Directory recurse | true |
| Auto-sync | prune + selfHeal |

### AppProject: tenant-apps

| Setting | Value |
|---------|-------|
| Name | tenant-apps |
| Description | Tenant deployments — restricted to zenith-* namespaces |
| Allowed repos | taikuri-infra/Zenith.git only |
| Allowed destinations | zenith-* namespaces only |
| Cluster resources | None (no ClusterRoles, etc.) |

---

## 9. Troubleshooting

### Application stuck in "OutOfSync"

```bash
# 1. Check what's different
argocd app diff zenith-api
# Or in UI: click app → "Diff" tab

# 2. Common cause: Helm values changed but not pushed to staging branch
git log --oneline staging..main
# If there are commits on main not in staging → merge!
git checkout staging && git merge main && git push origin staging

# 3. Force sync
argocd app sync zenith-api --force
```

### Application stuck in "Progressing"

```bash
# Pods not becoming Ready
kubectl get pods -n zenith-staging -l app=zenith-api
kubectl describe pod -n zenith-staging <pod-name>
kubectl logs -n zenith-staging <pod-name>

# Common causes:
# - Image pull error (wrong tag or missing pull secret)
# - Readiness probe failing (app crashing)
# - Resource limits too low (OOMKilled)
```

### Image Updater not detecting new tags

```bash
# 1. Check Image Updater logs
kubectl logs -n argocd deploy/argocd-image-updater --tail=50

# 2. Check credentials
kubectl get secret harbor-image-updater-creds -n argocd -o yaml

# 3. Test Harbor API manually
curl -u "robot\$username:password" \
  https://registry.stage.freezenith.com/v2/zenith-stage/zenith-api/tags/list

# 4. Check Application annotations (Image Updater reads these)
kubectl get app zenith-api -n argocd -o yaml | grep -A5 annotations
```

### "ComparisonError" in ArgoCD UI

```bash
# Usually means the Helm chart can't render
# 1. Check Repo Server logs
kubectl logs -n argocd deploy/argocd-repo-server --tail=50

# 2. Try rendering locally
helm template zenith-api infra/helm/zenith-api/ \
  -f infra/helm/zenith-api/values-staging.yaml
```

---

## 10. Upgrade Path

### Upgrading ArgoCD

```bash
# Update version in variables.tf, then:
terraform plan -target=helm_release.argocd
terraform apply -target=helm_release.argocd

# Verify
kubectl get pods -n argocd
argocd version
```

### Adding a new application

1. Create Helm chart in `infra/helm/zenith-<name>/`
2. Create ArgoCD Application YAML in `infra/argocd/staging/zenith-<name>.yaml`
3. Add values-staging.yaml with environment-specific config
4. Push to staging branch → ArgoCD auto-discovers and syncs

### Changing the watched branch

Update `argocd_target_revision` variable in Terraform:
```hcl
variable "argocd_target_revision" {
  default = "staging"  # Change to "main" or "production"
}
```
