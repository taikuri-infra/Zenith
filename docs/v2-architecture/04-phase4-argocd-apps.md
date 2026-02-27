# Phase 4: ArgoCD Applications (GitOps)

> **Zenith V2 Platform Architecture -- Phase 4 of 5**
>
> This phase is different from the others: it is automatic. Once ArgoCD is running
> (installed in Phase 3), it continuously watches Git and deploys applications.
> There is no `terraform apply` or `ansible-playbook` to run. You push code to Git,
> and ArgoCD handles the rest.

---

## Table of Contents

1. [Why This Phase Exists](#why-this-phase-exists)
2. [How ArgoCD Works](#how-argocd-works)
3. [App-of-Apps Pattern](#app-of-apps-pattern)
4. [Application Manifests](#application-manifests)
   - [Root Application](#root-application)
   - [zenith-api Application](#zenith-api-application)
   - [zenith-landing Application](#zenith-landing-application)
   - [zenith-demo Application](#zenith-demo-application)
   - [zenith-tenant Application](#zenith-tenant-application)
5. [Helm Charts in Harbor](#helm-charts-in-harbor)
6. [ArgoCD Image Updater](#argocd-image-updater)
7. [Developer Workflow](#developer-workflow)
8. [Per-Environment Configuration](#per-environment-configuration)
9. [Customer Onboarding via ArgoCD](#customer-onboarding-via-argocd)
10. [Sync Policies](#sync-policies)
11. [Rollback Procedures](#rollback-procedures)
12. [How to Run](#how-to-run)
13. [Verification Checklist](#verification-checklist)
14. [Troubleshooting](#troubleshooting)
15. [What Happens Next](#what-happens-next)

---

## Why This Phase Exists

Phases 1 through 3 built everything from the ground up: a Hetzner VM (Phase 1), a k3s
cluster (Phase 2), and 14 infrastructure components (Phase 3). At the end of Phase 3,
ArgoCD is installed and watching a Git repository. But no applications are deployed yet.

Phase 4 is where the applications come online. Unlike Phases 1-3, this is not a one-time
operation. Phase 4 describes the **continuous deployment loop** that runs for the lifetime
of the platform:

```
Phases 1-3: One-time setup               Phase 4: Continuous loop
(run once, or when infra changes)         (runs forever, automatically)

+----------+  +----------+  +----------+     +---------------------------+
| Phase 1  |->| Phase 2  |->| Phase 3  |--->| Git push --> ArgoCD sync  |
| Hetzner  |  | k3s      |  | Infra    |    | --> App deployed          |
| + CF DNS |  | + Cilium |  | + ArgoCD |    | --> Repeat on every push  |
+----------+  +----------+  +----------+     +---------------------------+
                                                       ^
                                                       | (this is Phase 4)
```

### The Separation of Concerns (Decision D10)

A critical design decision in Zenith V2 is the split between what Terraform manages and
what ArgoCD manages:

| Concern | Tool | State Source of Truth | Examples |
|---------|------|----------------------|----------|
| **Infrastructure** | Terraform | `terraform.tfstate` | cert-manager, CNPG operator, APISIX, Keycloak, Harbor, Monitoring |
| **Applications** | ArgoCD | Git repository | zenith-api, zenith-landing, zenith-demo, customer apps |

Why this split?

- **Infrastructure changes are infrequent and high-risk.** You do not upgrade cert-manager
  five times a day. When you do, you want a `terraform plan` to review the blast radius
  before applying. Terraform's plan/apply model is designed for exactly this.

- **Application changes are frequent and low-risk.** Developers push API changes multiple
  times a day. You want these deployed automatically with zero human intervention. ArgoCD's
  continuous reconciliation loop is designed for exactly this.

- **Infrastructure has complex dependencies.** cert-manager must exist before CNPG. CNPG
  must exist before Keycloak. Terraform's dependency graph handles this natively. ArgoCD
  has no concept of inter-application dependencies.

- **Applications are independent.** zenith-api does not depend on zenith-landing at deploy
  time. They can be synced in any order, rolled back independently, and scaled separately.
  ArgoCD treats each Application as an isolated unit, which matches this reality.

### Why ArgoCD and Not FluxCD? (Decision D4)

Both ArgoCD and FluxCD are CNCF-graduated GitOps tools. The decision to use ArgoCD was
driven by three factors:

1. **Web UI.** ArgoCD ships with a production-ready web dashboard that shows application
   health, sync status, resource tree, and diff visualization. This UI can be integrated
   into the Zenith admin panel, giving operators a single pane of glass. FluxCD is
   CLI-only (Weave GitOps adds a UI, but it is a separate product).

2. **App-of-Apps pattern.** ArgoCD's native Application CRD makes the App-of-Apps pattern
   trivial to implement. One root Application points to a directory of Application
   manifests. FluxCD achieves this with Kustomization dependencies, which is more complex
   and less intuitive.

3. **Certification preparation.** Babak is preparing for the Certified Argo Project
   Associate (CAPA) exam. Using ArgoCD in a production environment provides hands-on
   experience that directly supports this goal.

---

## How ArgoCD Works

ArgoCD is a Kubernetes controller that implements the GitOps pattern. Its core
reconciliation loop is:

```
Every 3 minutes (configurable):

1. ArgoCD pulls the latest commit from the Git repository
2. It renders the manifests (Helm template, Kustomize build, or plain YAML)
3. It compares the rendered manifests against the live cluster state
4. If there is a difference:
   a. With auto-sync ON:  ArgoCD applies the changes automatically
   b. With auto-sync OFF: ArgoCD marks the Application as "OutOfSync"
                           and waits for manual sync
```

### ArgoCD Components

ArgoCD consists of four services, all installed in Phase 3 in the `argocd` namespace:

```
namespace: argocd
  |
  |-- argocd-server (Deployment)
  |     The API server and web UI. Serves the ArgoCD dashboard and handles
  |     API requests from the CLI and UI. TLS is terminated by Traefik, so
  |     ArgoCD runs in --insecure mode internally.
  |     Ingress: argocd.stage.freezenith.com
  |
  |-- argocd-repo-server (Deployment)
  |     Clones Git repositories, renders Helm charts and Kustomize overlays.
  |     This is where `helm template` runs. It caches rendered manifests to
  |     avoid re-rendering on every reconciliation cycle.
  |
  |-- argocd-application-controller (StatefulSet)
  |     The core reconciliation engine. Watches Application CRDs, compares
  |     desired state (from Git) against live state (from Kubernetes API),
  |     and applies diffs when auto-sync is enabled.
  |
  |-- argocd-redis (Deployment)
  |     In-memory cache for the application controller. Stores rendered
  |     manifests, cluster cache, and reconciliation state.
  |
  |-- argocd-image-updater (Deployment)  [installed alongside ArgoCD]
  |     Watches container registries (Harbor) for new image tags. When a
  |     new tag is detected, it updates Application annotations to trigger
  |     a sync with the new image version.
```

### How ArgoCD Authenticates to Git and Harbor

ArgoCD needs credentials for two things:

1. **Git repository access** (to read Application manifests and Helm charts).
   Configured in Phase 3 via Terraform:
   ```hcl
   set_sensitive {
     name  = "configs.repositories.zenith.url"
     value = "https://github.com/DoTech/Zenith.git"
   }
   set_sensitive {
     name  = "configs.repositories.zenith.password"
     value = var.github_token
   }
   ```

2. **Harbor OCI registry access** (to pull Helm charts stored as OCI artifacts).
   Configured as a repository credential in ArgoCD:
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: harbor-helm-creds
     namespace: argocd
     labels:
       argocd.argoproj.io/secret-type: repository
   stringData:
     type: helm
     name: harbor-charts
     url: harbor.freezenith.com/charts
     enableOCI: "true"
     username: argocd-pull
     password: <harbor-robot-account-token>
   ```

---

## App-of-Apps Pattern

The App-of-Apps pattern is the backbone of Zenith's ArgoCD deployment strategy. The
concept is simple: instead of creating Application CRDs manually (via `kubectl apply` or
Terraform), you create ONE root Application that points to a directory in Git. That
directory contains more Application manifests. ArgoCD applies them all and keeps them in
sync.

### Why This Pattern?

Without App-of-Apps, adding a new application requires:
1. Writing a Helm chart
2. Writing an ArgoCD Application manifest
3. Manually applying the Application manifest with `kubectl apply`

With App-of-Apps, step 3 becomes "commit the Application YAML to Git." ArgoCD
automatically detects the new file and creates the Application. This is GitOps all the
way down.

### Directory Structure

```
infra/argocd/
  staging/                          <-- ArgoCD watches this directory
    root-app.yaml                   <-- The parent Application (created by Terraform in Phase 3)
    zenith-api.yaml                 <-- Application: zenith-api Helm chart
    zenith-landing.yaml             <-- Application: zenith-landing Helm chart
    zenith-demo.yaml                <-- Application: zenith-demo Helm chart
    zenith-tenant.yaml              <-- Application: zenith-tenant Helm chart (template)
    zenith-platform.yaml            <-- Application: zenith-platform shared resources
    tenants/                        <-- Per-customer overrides (generated by Temporal)
      embermind.yaml                <-- Customer-specific Application pointing to zenith-tenant chart
      acme-corp.yaml                <-- Another customer

  production/                       <-- Same structure, different values
    root-app.yaml
    zenith-api.yaml
    zenith-landing.yaml
    zenith-demo.yaml
    zenith-platform.yaml
    tenants/
      embermind.yaml
```

### How It Flows

```
Terraform (Phase 3)
    |
    | Creates ONE resource: the root Application
    |
    v
Root Application (zenith-apps)
    |
    | source.path = infra/argocd/staging/
    |
    | ArgoCD reads all YAML files in that directory
    |
    +---> zenith-platform.yaml   --> deploys zenith-platform Helm chart
    +---> zenith-api.yaml        --> deploys zenith-api Helm chart
    +---> zenith-landing.yaml    --> deploys zenith-landing Helm chart
    +---> zenith-demo.yaml       --> deploys zenith-demo Helm chart
    +---> zenith-tenant.yaml     --> deploys zenith-tenant Helm chart (embermind)
    +---> tenants/embermind.yaml --> deploys customer-specific overrides
    +---> tenants/acme-corp.yaml --> deploys another customer
```

If a developer adds `tenants/new-customer.yaml` to Git and pushes, ArgoCD
automatically detects it within 3 minutes and creates a new Application -- which
then deploys the customer's infrastructure. No manual intervention.

### Helm Charts (5 Modular Charts)

The Application manifests point to Helm charts stored either in the Git repository
or in Harbor as OCI artifacts:

```
infra/helm/
  zenith-platform/    Shared resources: Sealed Secrets, CNPG credentials,
  |                   certificates, APISIX middleware, cross-cutting concerns.
  |                   Deployed to: zenith-platform namespace
  |
  zenith-api/         The Go API server. Deployment, Service, IngressRoute,
  |                   APISIX routes, HPA, PDB, ServiceMonitor.
  |                   Deployed to: zenith-platform namespace
  |
  zenith-landing/     The freezenith.com landing page. Next.js standalone build.
  |                   Deployment, Service, IngressRoute.
  |                   Deployed to: zenith-platform namespace
  |
  zenith-demo/        Demo Mission Control + Demo Web Platform.
  |                   Two Deployments, two Services, IngressRoutes.
  |                   Deployed to: zenith-platform namespace
  |
  zenith-tenant/      Template chart for per-customer deployments.
                      Frontend Deployment, Backend Deployment, Services,
                      IngressRoutes, APISIX routes, CiliumNetworkPolicy,
                      ResourceQuota, LimitRange.
                      Deployed to: zenith-<customer> namespace (one per customer)
```

---

## Application Manifests

Each Application manifest is a Kubernetes custom resource of kind `Application` in the
`argoproj.io/v1alpha1` API group. Below are the complete manifests for every application
in the Zenith platform.

### Root Application

This is the ONLY Application created by Terraform (in Phase 3). All other Applications
are children managed by ArgoCD itself via the App-of-Apps pattern.

```yaml
# infra/argocd/staging/root-app.yaml
#
# This file exists in Git for documentation purposes.
# The actual resource is created by Terraform in Phase 3 (argocd.tf).
# If ArgoCD is reinstalled, Terraform recreates this Application.

apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: zenith-apps
  namespace: argocd
  # The root app is in the "default" ArgoCD project.
  # Child apps can be in the "default" project or a custom project with
  # restricted permissions (e.g., "tenant-apps" project).
spec:
  project: default

  source:
    repoURL: https://github.com/DoTech/Zenith.git
    targetRevision: staging       # Branch: "staging" for staging, "main" for production
    path: infra/argocd/staging    # Directory containing child Application manifests

  destination:
    server: https://kubernetes.default.svc
    namespace: argocd             # Child Applications are created in the argocd namespace

  syncPolicy:
    automated:
      prune: true                 # Delete Applications removed from Git
      selfHeal: true              # Revert manual changes to Applications
    syncOptions:
      - CreateNamespace=true      # Create target namespaces if they do not exist
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

**Key fields explained:**

- `targetRevision: staging` -- ArgoCD tracks the `staging` branch. For production, this
  is `main`. This means you can test Application manifest changes on `staging` before
  merging to `main`.

- `path: infra/argocd/staging` -- ArgoCD reads every YAML file in this directory and
  applies them. Subdirectories (like `tenants/`) are included.

- `prune: true` -- If you delete `tenants/old-customer.yaml` from Git, ArgoCD deletes
  the corresponding Application. This is how customer offboarding works: remove the
  file, push, and the customer's entire namespace is cleaned up.

- `selfHeal: true` -- If someone manually edits an Application via `kubectl edit`,
  ArgoCD reverts the change within the next reconciliation cycle. Git is the source of
  truth, not the live cluster.

### zenith-api Application

```yaml
# infra/argocd/staging/zenith-api.yaml

apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: zenith-api
  namespace: argocd
  annotations:
    # ArgoCD Image Updater annotations (see Image Updater section)
    argocd-image-updater.argoproj.io/image-list: api=harbor.freezenith.com/zenith/zenith-api
    argocd-image-updater.argoproj.io/api.update-strategy: semver
    argocd-image-updater.argoproj.io/api.helm.image-name: image.repository
    argocd-image-updater.argoproj.io/api.helm.image-tag: image.tag
  finalizers:
    - resources-finalizer.argocd.argoproj.io    # Clean up K8s resources when Application is deleted
spec:
  project: default

  source:
    # Option A: Helm chart from Git repository
    repoURL: https://github.com/DoTech/Zenith.git
    targetRevision: staging
    path: infra/helm/zenith-api

    # Option B: Helm chart from Harbor OCI registry (used after CI publishes charts)
    # repoURL: harbor.freezenith.com/charts
    # chart: zenith-api
    # targetRevision: 1.0.0

    helm:
      releaseName: zenith-api
      valueFiles:
        - values.yaml                      # Base values (chart defaults)
        - values-staging.yaml              # Environment-specific overrides
      parameters:
        - name: image.repository
          value: harbor.freezenith.com/zenith/zenith-api
        - name: image.tag
          value: v1.0.0                    # Updated automatically by Image Updater
        - name: replicaCount
          value: "1"                       # Staging: 1 replica. Production: 2+
        - name: ingress.host
          value: api.stage.freezenith.com

  destination:
    server: https://kubernetes.default.svc
    namespace: zenith-platform

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true               # Required for large CRDs (APISIX routes)
```

**What the zenith-api Helm chart deploys:**

```
namespace: zenith-platform
  |
  |-- Deployment: zenith-api (1-2 replicas)
  |     |-- Container: zenith-api
  |     |     image: harbor.freezenith.com/zenith/zenith-api:v1.0.0
  |     |     ports: 8080 (HTTP), 9090 (metrics)
  |     |     env: from Secrets (DB creds, Keycloak URL, S3 keys, Temporal host)
  |     |     resources: requests 100m/128Mi, limits 500m/512Mi
  |     |     probes: /health (liveness), /ready (readiness)
  |
  |-- Service: zenith-api (ClusterIP, port 8080)
  |
  |-- IngressRoute: zenith-api (Traefik -> APISIX)
  |     Host: api.stage.freezenith.com
  |
  |-- ApisixRoute: zenith-api-protected
  |     Routes /v1/* through the JWT-protected APISIX gateway
  |
  |-- ApisixRoute: zenith-api-public
  |     Routes /v1/webhooks/*, /health through the public APISIX gateway
  |
  |-- HorizontalPodAutoscaler: zenith-api
  |     Min: 1, Max: 5, Target CPU: 70%
  |
  |-- PodDisruptionBudget: zenith-api
  |     minAvailable: 1
  |
  |-- ServiceMonitor: zenith-api
  |     Scrapes :9090/metrics for Prometheus
```

### zenith-landing Application

```yaml
# infra/argocd/staging/zenith-landing.yaml

apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: zenith-landing
  namespace: argocd
  annotations:
    argocd-image-updater.argoproj.io/image-list: landing=harbor.freezenith.com/zenith/zenith-landing
    argocd-image-updater.argoproj.io/landing.update-strategy: semver
    argocd-image-updater.argoproj.io/landing.helm.image-name: image.repository
    argocd-image-updater.argoproj.io/landing.helm.image-tag: image.tag
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default

  source:
    repoURL: https://github.com/DoTech/Zenith.git
    targetRevision: staging
    path: infra/helm/zenith-landing
    helm:
      releaseName: zenith-landing
      valueFiles:
        - values.yaml
        - values-staging.yaml
      parameters:
        - name: image.repository
          value: harbor.freezenith.com/zenith/zenith-landing
        - name: image.tag
          value: v1.0.0
        - name: ingress.host
          value: stage.freezenith.com

  destination:
    server: https://kubernetes.default.svc
    namespace: zenith-platform

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

**What the zenith-landing Helm chart deploys:**

```
namespace: zenith-platform
  |
  |-- Deployment: zenith-landing (1 replica)
  |     |-- Container: zenith-landing
  |     |     image: harbor.freezenith.com/zenith/zenith-landing:v1.0.0
  |     |     port: 3000 (Next.js standalone)
  |     |     resources: requests 50m/64Mi, limits 200m/256Mi
  |
  |-- Service: zenith-landing (ClusterIP, port 3000)
  |
  |-- IngressRoute: zenith-landing (Traefik, direct to pod -- no APISIX)
  |     Host: stage.freezenith.com
  |     Host: www.stage.freezenith.com (redirect to root)
  |
  |-- Certificate: stage-freezenith-com-tls (cert-manager, DNS-01)
```

### zenith-demo Application

```yaml
# infra/argocd/staging/zenith-demo.yaml

apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: zenith-demo
  namespace: argocd
  annotations:
    argocd-image-updater.argoproj.io/image-list: >-
      mc=harbor.freezenith.com/zenith/zenith-mc-demo,
      web=harbor.freezenith.com/zenith/zenith-web-demo
    argocd-image-updater.argoproj.io/mc.update-strategy: semver
    argocd-image-updater.argoproj.io/mc.helm.image-name: mc.image.repository
    argocd-image-updater.argoproj.io/mc.helm.image-tag: mc.image.tag
    argocd-image-updater.argoproj.io/web.update-strategy: semver
    argocd-image-updater.argoproj.io/web.helm.image-name: web.image.repository
    argocd-image-updater.argoproj.io/web.helm.image-tag: web.image.tag
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default

  source:
    repoURL: https://github.com/DoTech/Zenith.git
    targetRevision: staging
    path: infra/helm/zenith-demo
    helm:
      releaseName: zenith-demo
      valueFiles:
        - values.yaml
        - values-staging.yaml
      parameters:
        - name: mc.image.repository
          value: harbor.freezenith.com/zenith/zenith-mc-demo
        - name: mc.image.tag
          value: v1.0.0
        - name: mc.ingress.host
          value: demo-ms.stage.freezenith.com
        - name: web.image.repository
          value: harbor.freezenith.com/zenith/zenith-web-demo
        - name: web.image.tag
          value: v1.0.0
        - name: web.ingress.host
          value: demo-cloud.stage.freezenith.com

  destination:
    server: https://kubernetes.default.svc
    namespace: zenith-platform

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

**What the zenith-demo Helm chart deploys:**

```
namespace: zenith-platform
  |
  |-- Deployment: zenith-mc-demo (1 replica)
  |     |-- Container: zenith-mc-demo
  |     |     image: harbor.freezenith.com/zenith/zenith-mc-demo:v1.0.0
  |     |     port: 3100
  |     |     env: NEXT_PUBLIC_DEMO_MODE=true (baked into image at build time)
  |
  |-- Deployment: zenith-web-demo (1 replica)
  |     |-- Container: zenith-web-demo
  |     |     image: harbor.freezenith.com/zenith/zenith-web-demo:v1.0.0
  |     |     port: 3000
  |     |     env: NEXT_PUBLIC_DEMO_MODE=true (baked into image at build time)
  |
  |-- Service: zenith-mc-demo (ClusterIP, port 3100)
  |-- Service: zenith-web-demo (ClusterIP, port 3000)
  |
  |-- IngressRoute: zenith-mc-demo (Traefik, direct)
  |     Host: demo-ms.stage.freezenith.com
  |
  |-- IngressRoute: zenith-web-demo (Traefik, direct)
  |     Host: demo-cloud.stage.freezenith.com
```

### zenith-tenant Application

The tenant Application is a template. Each customer gets their own copy with
customer-specific values. The Temporal provisioning workflow generates the
customer YAML and commits it to Git.

```yaml
# infra/argocd/staging/tenants/embermind.yaml
#
# This file was generated by the Temporal provisioning workflow.
# Do NOT edit manually. Changes will be overwritten on next provision cycle.

apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: tenant-embermind
  namespace: argocd
  labels:
    zenith.io/component: tenant
    zenith.io/customer: embermind
    zenith.io/tier: pro
  annotations:
    argocd-image-updater.argoproj.io/image-list: >-
      mc=harbor.freezenith.com/zenith/zenith-mc,
      web=harbor.freezenith.com/zenith/zenith-web
    argocd-image-updater.argoproj.io/mc.update-strategy: semver
    argocd-image-updater.argoproj.io/mc.helm.image-name: mc.image.repository
    argocd-image-updater.argoproj.io/mc.helm.image-tag: mc.image.tag
    argocd-image-updater.argoproj.io/web.update-strategy: semver
    argocd-image-updater.argoproj.io/web.helm.image-name: web.image.repository
    argocd-image-updater.argoproj.io/web.helm.image-tag: web.image.tag
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: tenant-apps              # Restricted project (see below)

  source:
    repoURL: https://github.com/DoTech/Zenith.git
    targetRevision: staging
    path: infra/helm/zenith-tenant
    helm:
      releaseName: tenant-embermind
      valueFiles:
        - values.yaml
      parameters:
        # Customer identity
        - name: customer.name
          value: embermind
        - name: customer.tier
          value: pro
        - name: customer.domain
          value: embermind.app

        # Mission Control
        - name: mc.image.repository
          value: harbor.freezenith.com/zenith/zenith-mc
        - name: mc.image.tag
          value: v1.0.0
        - name: mc.ingress.host
          value: ms.embermind.app

        # Web Platform
        - name: web.image.repository
          value: harbor.freezenith.com/zenith/zenith-web
        - name: web.image.tag
          value: v1.0.0
        - name: web.ingress.host
          value: cloud.embermind.app

        # Backend API
        - name: api.ingress.host
          value: api.embermind.app

        # Resource limits (per tier)
        - name: resourceQuota.cpu
          value: "4"
        - name: resourceQuota.memory
          value: 8Gi
        - name: resourceQuota.pods
          value: "20"

        # Database (credentials injected as Sealed Secrets)
        - name: database.host
          value: pro-shard-1-pg-rw.zenith-shared.svc.cluster.local
        - name: database.name
          value: customer_embermind

        # Keycloak realm
        - name: keycloak.realm
          value: embermind
        - name: keycloak.issuerUrl
          value: https://auth.freezenith.com/realms/embermind

  destination:
    server: https://kubernetes.default.svc
    namespace: zenith-embermind      # Each customer gets their own namespace

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
    retry:
      limit: 3
      backoff:
        duration: 10s
        factor: 2
        maxDuration: 2m
```

**What the zenith-tenant Helm chart deploys (per customer):**

```
namespace: zenith-embermind
  |
  |-- Deployment: embermind-mc (1 replica)
  |     Mission Control frontend
  |
  |-- Deployment: embermind-web (1 replica)
  |     Web Platform frontend
  |
  |-- Deployment: embermind-api (1 replica)
  |     Customer backend API (if customer has custom backend)
  |
  |-- Service: embermind-mc (ClusterIP, port 3100)
  |-- Service: embermind-web (ClusterIP, port 3000)
  |-- Service: embermind-api (ClusterIP, port 8080)
  |
  |-- IngressRoute: embermind-mc
  |     Host: ms.embermind.app (Traefik, direct)
  |
  |-- IngressRoute: embermind-web
  |     Host: cloud.embermind.app (Traefik, direct)
  |
  |-- ApisixRoute: embermind-api-protected
  |     Host: api.embermind.app, JWT-protected via Keycloak realm "embermind"
  |
  |-- ApisixRoute: embermind-api-public
  |     Webhooks, public endpoints
  |
  |-- CiliumNetworkPolicy: default-deny + explicit allows
  |-- ResourceQuota: cpu=4, memory=8Gi, pods=20
  |-- LimitRange: per-pod defaults
  |-- Certificate: embermind-app-tls (cert-manager)
  |
  |-- SealedSecret: embermind-db-credentials
  |-- SealedSecret: embermind-s3-credentials
  |-- SealedSecret: embermind-keycloak-credentials
```

### ArgoCD Project for Tenant Apps

Tenant Applications run in a restricted ArgoCD Project to enforce isolation. The
`tenant-apps` project limits what namespaces and cluster resources tenant Applications
can access:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: tenant-apps
  namespace: argocd
spec:
  description: Restricted project for customer tenant deployments

  # Only allow deploying to zenith-* namespaces
  destinations:
    - namespace: "zenith-*"
      server: https://kubernetes.default.svc

  # Only allow specific resource types (no ClusterRoles, no CRDs, no Nodes)
  namespaceResourceWhitelist:
    - group: ""
      kind: ConfigMap
    - group: ""
      kind: Secret
    - group: ""
      kind: Service
    - group: ""
      kind: ServiceAccount
    - group: apps
      kind: Deployment
    - group: apps
      kind: StatefulSet
    - group: networking.k8s.io
      kind: NetworkPolicy
    - group: autoscaling
      kind: HorizontalPodAutoscaler
    - group: policy
      kind: PodDisruptionBudget
    - group: traefik.io
      kind: IngressRoute
    - group: apisix.apache.org
      kind: ApisixRoute
    - group: cilium.io
      kind: CiliumNetworkPolicy
    - group: bitnami.com
      kind: SealedSecret

  # Only allow images from Harbor
  sourceRepos:
    - https://github.com/DoTech/Zenith.git
    - harbor.freezenith.com/charts/*

  # Cluster-scoped resources that tenant apps can create
  clusterResourceWhitelist:
    - group: ""
      kind: Namespace
```

---

## Helm Charts in Harbor

Application Helm charts can be stored in two locations:

1. **Git repository** (used during development and early staging). ArgoCD reads charts
   directly from the `infra/helm/` directory in the Git repo. This is simple and requires
   no chart publishing step.

2. **Harbor OCI registry** (used in production). Charts are published as OCI artifacts
   during CI. This decouples chart versions from Git branches and allows semantic
   versioning of charts independent of application code.

### OCI Chart URLs in Harbor

```
harbor.freezenith.com/charts/zenith-platform:1.0.0
harbor.freezenith.com/charts/zenith-api:1.2.3
harbor.freezenith.com/charts/zenith-landing:1.0.1
harbor.freezenith.com/charts/zenith-demo:1.0.0
harbor.freezenith.com/charts/zenith-tenant:1.1.0
```

### Publishing a Chart to Harbor

```bash
# From the chart directory
cd infra/helm/zenith-api

# Package the chart
helm package .
# Successfully packaged chart and saved it to: zenith-api-1.2.3.tgz

# Login to Harbor
helm registry login harbor.freezenith.com -u admin

# Push as OCI artifact
helm push zenith-api-1.2.3.tgz oci://harbor.freezenith.com/charts
# Pushed: harbor.freezenith.com/charts/zenith-api:1.2.3
```

In production, this is automated by the CI pipeline (see Developer Workflow below).
Developers never push charts manually.

### Switching an Application from Git to Harbor

When ready to use Harbor-hosted charts, update the Application manifest:

```yaml
# Before (Git source):
source:
  repoURL: https://github.com/DoTech/Zenith.git
  targetRevision: staging
  path: infra/helm/zenith-api

# After (Harbor OCI source):
source:
  repoURL: harbor.freezenith.com/charts
  chart: zenith-api
  targetRevision: 1.2.3           # Semantic version
```

---

## ArgoCD Image Updater

ArgoCD Image Updater is a companion controller that automates the "last mile" of
continuous deployment: detecting new container images in Harbor and updating
Application manifests to use them.

### The Problem It Solves

Without Image Updater, the deployment workflow has a manual step:

```
Developer pushes code
  --> CI builds image, pushes to Harbor as zenith-api:v1.2.4
  --> ??? Someone must update the Application manifest with "v1.2.4" ???
  --> ArgoCD syncs the new version
```

With Image Updater, step 3 is automatic:

```
Developer pushes code
  --> CI builds image, pushes to Harbor as zenith-api:v1.2.4
  --> Image Updater detects the new tag in Harbor
  --> Image Updater updates the Application's image tag parameter
  --> ArgoCD syncs the new version
```

### How It Works

Image Updater watches the container registry for new tags matching a configured
strategy. When a new qualifying tag appears:

1. Image Updater reads the Application's current image tag from its annotations
2. It compares against available tags in the registry
3. If a newer tag exists (according to the update strategy), it writes the new tag
   back to the Application as a parameter override
4. ArgoCD detects the Application has changed and syncs it

```
Harbor Registry                Image Updater              ArgoCD Application
     |                              |                           |
     | New image pushed:            |                           |
     | zenith-api:v1.2.4            |                           |
     |                              |                           |
     |   <-- polls every 2 min --   |                           |
     |                              |                           |
     | Tags: v1.2.3, v1.2.4        |                           |
     |   ----------------------->   |                           |
     |                              |                           |
     |                              | Current: v1.2.3           |
     |                              | Latest:  v1.2.4           |
     |                              |                           |
     |                              | Update image.tag=v1.2.4   |
     |                              |   ----------------------> |
     |                              |                           |
     |                              |               ArgoCD syncs:
     |                              |               Deployment updated
     |                              |               Rolling update starts
```

### Update Strategies

| Strategy | Behavior | Use Case |
|----------|----------|----------|
| `semver` | Picks the highest SemVer tag matching a constraint | Production: `~1.2` allows 1.2.x but not 1.3.0 |
| `latest` | Picks the tag with the most recent build date | Staging: always deploy the newest build |
| `digest` | Picks the most recently pushed tag (by digest) | Feature branches: deploy whatever was pushed last |
| `name` | Alphabetical sort of tag names | Rarely used |

**Staging** uses `semver` with no constraint (any new version is deployed).
**Production** uses `semver` with a constraint (e.g., `~1.2`) to prevent accidental
major version deployments.

### Image Updater Annotations Reference

These annotations on an Application tell Image Updater what to watch:

```yaml
annotations:
  # List of images to watch (alias=registry/image)
  argocd-image-updater.argoproj.io/image-list: api=harbor.freezenith.com/zenith/zenith-api

  # Update strategy for each alias
  argocd-image-updater.argoproj.io/api.update-strategy: semver

  # Optional: SemVer constraint (only in production)
  # argocd-image-updater.argoproj.io/api.allow-tags: regexp:^v1\.2\.\d+$

  # Helm parameter names to update
  argocd-image-updater.argoproj.io/api.helm.image-name: image.repository
  argocd-image-updater.argoproj.io/api.helm.image-tag: image.tag

  # Write-back method: "argocd" updates the Application resource directly
  # (alternative: "git" commits changes to Git -- more GitOps-pure but slower)
  argocd-image-updater.argoproj.io/write-back-method: argocd
```

---

## Developer Workflow

The end-to-end flow from code change to production deployment:

```
Step 1: Developer pushes code
+-----------------------------------------------------+
|  git push origin feature/add-endpoint                |
|  --> Pull request created                            |
|  --> PR reviewed and merged to main (or staging)     |
+-----------------------------------------------------+
                    |
                    v
Step 2: GitHub Actions CI pipeline
+-----------------------------------------------------+
|  Trigger: push to main (or staging) branch           |
|                                                       |
|  Jobs (parallel where possible):                     |
|  1. Build Docker image                               |
|     docker build -f apps/api/Dockerfile .            |
|                                                       |
|  2. Run tests                                        |
|     go test ./...                                    |
|                                                       |
|  3. Lint                                             |
|     golangci-lint run                                |
|                                                       |
|  4. Security scan                                    |
|     trivy image zenith-api:$SHA                      |
|     (fail CI if CRITICAL CVEs found)                 |
|                                                       |
|  5. Sign image                                       |
|     cosign sign harbor.freezenith.com/zenith/...     |
|                                                       |
|  6. Push to Harbor                                   |
|     docker tag zenith-api:$SHA                       |
|       harbor.freezenith.com/zenith/zenith-api:v1.2.4 |
|     docker push harbor...                            |
|                                                       |
|  7. (Optional) Package and push Helm chart           |
|     helm package infra/helm/zenith-api               |
|     helm push zenith-api-1.2.4.tgz                   |
|       oci://harbor.freezenith.com/charts             |
+-----------------------------------------------------+
                    |
                    v
Step 3: ArgoCD Image Updater detects new image
+-----------------------------------------------------+
|  Image Updater polls Harbor every 2 minutes          |
|  Detects: zenith-api:v1.2.4 (newer than v1.2.3)     |
|  Updates Application parameter: image.tag = v1.2.4   |
+-----------------------------------------------------+
                    |
                    v
Step 4: ArgoCD syncs the Application
+-----------------------------------------------------+
|  Application controller detects parameter change     |
|  Renders Helm chart with new image tag               |
|  Compares against live cluster state                 |
|  Applies diff: Deployment spec.template updated      |
+-----------------------------------------------------+
                    |
                    v
Step 5: Kubernetes performs rolling update
+-----------------------------------------------------+
|  New pod created with v1.2.4 image                   |
|  Readiness probe passes                              |
|  Traffic shifted to new pod                          |
|  Old pod (v1.2.3) terminated                         |
|  Zero downtime achieved                              |
+-----------------------------------------------------+
                    |
                    v
Step 6: Observability
+-----------------------------------------------------+
|  Prometheus scrapes new pod metrics                   |
|  Grafana dashboards show deployment event             |
|  Loki captures pod startup logs                      |
|  Tempo traces new API requests                       |
|  If errors spike: alert fires via Alertmanager       |
+-----------------------------------------------------+
```

### Time from Push to Live

| Stage | Time |
|-------|------|
| CI pipeline (build, test, scan, push) | 3-5 minutes |
| Image Updater detection | 0-2 minutes (polling interval) |
| ArgoCD sync | ~30 seconds |
| Rolling update | 30-60 seconds |
| **Total** | **4-8 minutes** |

---

## Per-Environment Configuration

Each environment (staging, production) has its own directory of Application manifests
and its own Helm value overrides. The differences are captured in environment-specific
values files within each Helm chart.

### Values File Hierarchy

```
infra/helm/zenith-api/
  values.yaml                  # Base defaults (shared across all environments)
  values-staging.yaml          # Staging overrides
  values-production.yaml       # Production overrides
```

### Key Differences Between Environments

| Setting | Staging | Production |
|---------|---------|------------|
| `replicaCount` | 1 | 2-3 |
| `image.tag` | Latest semver | Pinned semver with constraint |
| `resources.requests.cpu` | 100m | 250m |
| `resources.requests.memory` | 128Mi | 256Mi |
| `resources.limits.cpu` | 500m | 1000m |
| `resources.limits.memory` | 512Mi | 1024Mi |
| `hpa.minReplicas` | 1 | 2 |
| `hpa.maxReplicas` | 3 | 10 |
| `ingress.host` | `api.stage.freezenith.com` | `api.freezenith.com` |
| `ingress.tls.issuer` | letsencrypt-staging | letsencrypt-prod |
| ArgoCD `targetRevision` | `staging` branch | `main` branch |
| Image Updater constraint | No constraint (any new version) | `~1.x` (minor versions only) |
| Sync policy | auto-sync + auto-prune | auto-sync + auto-prune (with notification) |

### Production Branch Strategy

```
feature/add-endpoint  -->  staging branch  -->  main branch
                           |                    |
                           ArgoCD staging       ArgoCD production
                           (auto-deploys)       (auto-deploys)
```

1. Developer creates feature branch, opens PR against `staging`
2. PR merged to `staging` -- CI builds and pushes image
3. ArgoCD deploys to staging automatically
4. After validation, PR from `staging` to `main`
5. Merge to `main` -- CI builds with production tag
6. ArgoCD deploys to production automatically

---

## Customer Onboarding via ArgoCD

When a new customer signs up, the Temporal provisioning workflow (see Phase 3 and the
overview document) performs these ArgoCD-related steps:

### Step-by-Step: Adding a New Tenant

```
1. Temporal Activity: CreateTenantApplication()
   |
   |  Generates a YAML file from template:
   |    infra/argocd/staging/tenants/<customer>.yaml
   |
   |  The YAML is an ArgoCD Application manifest pointing to the
   |  zenith-tenant Helm chart with customer-specific parameters.
   |
   v
2. Temporal Activity: CommitToGit()
   |
   |  Uses the GitHub API (or a service account with push access) to:
   |    git add infra/argocd/staging/tenants/<customer>.yaml
   |    git commit -m "feat: provision tenant <customer>"
   |    git push origin staging
   |
   v
3. ArgoCD Root Application Sync
   |
   |  Within 3 minutes, ArgoCD detects the new file in
   |  infra/argocd/staging/ and creates the Application.
   |
   v
4. ArgoCD Tenant Application Sync
   |
   |  The new Application renders the zenith-tenant Helm chart
   |  with customer-specific values and applies all resources:
   |    - Namespace: zenith-<customer>
   |    - Deployments (MC, Web, API)
   |    - Services, IngressRoutes, APISIX routes
   |    - CiliumNetworkPolicy, ResourceQuota, LimitRange
   |    - SealedSecrets (DB, S3, Keycloak credentials)
   |
   v
5. external-dns + cert-manager
   |
   |  IngressRoute annotations trigger:
   |    - external-dns creates DNS records in Cloudflare
   |    - cert-manager issues TLS certificates
   |
   v
6. Customer Environment Ready
   |
   |  All pods are running, DNS resolves, TLS is active.
   |  Temporal marks the provisioning workflow as complete.
   |  Customer receives "Your environment is ready!" notification.
```

### Customer Offboarding

Offboarding is the reverse: delete the file from Git.

```bash
# Remove the customer's Application manifest
git rm infra/argocd/staging/tenants/old-customer.yaml
git commit -m "feat: deprovision tenant old-customer"
git push origin staging
```

Because the root Application has `prune: true`, ArgoCD detects the missing file and
deletes the tenant Application. The tenant Application has the
`resources-finalizer.argocd.argoproj.io` finalizer, which causes ArgoCD to delete
all Kubernetes resources (Deployments, Services, IngressRoutes, Secrets, Namespace)
before the Application itself is removed.

The Temporal workflow also handles non-Kubernetes cleanup:
- Deletes the Keycloak realm
- Drops the PostgreSQL database
- Deletes the S3 bucket
- Removes Cloudflare DNS records (if not handled by external-dns)

---

## Sync Policies

ArgoCD sync policies control how and when changes are applied to the cluster. Zenith
uses different policies for different application types.

### Auto-Sync (Enabled for All Apps)

```yaml
syncPolicy:
  automated:
    prune: true       # Delete resources removed from Git
    selfHeal: true    # Revert manual changes made via kubectl
```

**What `prune` does:**
If a Deployment exists in the cluster but not in the Helm chart's rendered output
(because it was removed from the chart), ArgoCD deletes it. Without prune, orphaned
resources accumulate in the cluster.

**What `selfHeal` does:**
If someone runs `kubectl edit deployment zenith-api` and changes the replica count
from 2 to 5, ArgoCD reverts it to 2 (the value in Git) within seconds. This enforces
Git as the single source of truth.

### Sync Options

```yaml
syncOptions:
  - CreateNamespace=true        # Create the target namespace if it does not exist
  - ServerSideApply=true        # Use server-side apply for large/complex resources
  - PrunePropagationPolicy=foreground  # Wait for child resources to be deleted
  - PruneLast=true              # Prune after all other sync operations complete
```

### Retry Policy

```yaml
retry:
  limit: 5                     # Retry up to 5 times on failure
  backoff:
    duration: 5s               # First retry after 5 seconds
    factor: 2                  # Double the wait each time (5s, 10s, 20s, 40s, 80s)
    maxDuration: 3m            # Never wait more than 3 minutes
```

**When retries help:** Transient failures like "connection refused" to the Harbor
registry, "resource version conflict" from concurrent updates, or "webhook timeout"
from Kyverno under load.

### Sync Waves (Ordering Within an Application)

Some resources within a Helm chart must be created in a specific order. For example,
a Namespace must exist before a Deployment in that namespace. ArgoCD supports sync
waves via annotations:

```yaml
# In the Helm chart templates:

# Wave 0: Namespace and RBAC (created first)
apiVersion: v1
kind: Namespace
metadata:
  name: zenith-embermind
  annotations:
    argocd.argoproj.io/sync-wave: "0"

# Wave 1: Secrets and ConfigMaps
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"

# Wave 2: Deployments and Services (need Secrets to exist)
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "2"

# Wave 3: Ingress and APISIX routes (need Services to exist)
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "3"
```

---

## Rollback Procedures

ArgoCD maintains a history of every sync operation. If a deployment introduces a
regression, you can roll back to any previous version.

### Option 1: ArgoCD UI Rollback

```
1. Open the ArgoCD dashboard: https://argocd.stage.freezenith.com
2. Click on the affected Application (e.g., "zenith-api")
3. Click "History and Rollback" tab
4. Find the last known good revision
5. Click "Rollback" button
6. Confirm the rollback

ArgoCD will:
- Render the Helm chart at the previous revision
- Apply the diff (effectively reverting the Deployment to the old image)
- Kubernetes performs a rolling update to the old version
```

### Option 2: ArgoCD CLI Rollback

```bash
# List sync history
argocd app history zenith-api --server argocd.stage.freezenith.com

# Output:
# ID  DATE                 REVISION
# 5   2026-02-25 14:30:00  abc1234 (v1.2.4)   <-- bad
# 4   2026-02-25 10:15:00  def5678 (v1.2.3)   <-- good
# 3   2026-02-24 16:00:00  ghi9012 (v1.2.2)

# Roll back to revision 4
argocd app rollback zenith-api 4 --server argocd.stage.freezenith.com
```

### Option 3: Git Revert (Recommended for Production)

The most GitOps-pure approach: revert the commit in Git and let ArgoCD sync the
revert automatically.

```bash
# Find the bad commit
git log --oneline infra/helm/zenith-api/

# Revert it
git revert <bad-commit-sha>
git push origin main

# ArgoCD detects the revert and syncs automatically
# This creates an auditable trail: every change and rollback is in Git history
```

### Option 4: Pin to a Specific Image Tag

If the issue is the container image (not the Helm chart), override the image tag:

```bash
# Temporarily override the image tag in the Application
argocd app set zenith-api \
  --parameter image.tag=v1.2.3 \
  --server argocd.stage.freezenith.com

# Note: selfHeal will revert this if the Git source differs.
# To make it stick, update the Application manifest in Git.
```

### Rollback Timeline

| Method | Time to Rollback | Audit Trail | GitOps Pure |
|--------|-----------------|-------------|-------------|
| ArgoCD UI | ~30 seconds | ArgoCD history | No |
| ArgoCD CLI | ~30 seconds | ArgoCD history | No |
| Git revert | 2-5 minutes | Git history | Yes |
| Image tag override | ~30 seconds | ArgoCD history | No |

**Recommendation:** Use Git revert for production rollbacks. It creates the clearest
audit trail and does not fight with selfHeal. Use ArgoCD UI/CLI for staging when
speed matters more than process.

---

## How to Run

Phase 4 is mostly automatic, but there are some initial setup tasks and ongoing
operations to understand.

### Initial Setup (One-Time)

After Phase 3 completes:

```bash
# 1. Verify ArgoCD is running
kubectl -n argocd get pods
# argocd-server-<hash>                    1/1     Running
# argocd-repo-server-<hash>               1/1     Running
# argocd-application-controller-0         1/1     Running
# argocd-redis-<hash>                     1/1     Running

# 2. Verify the root Application exists
kubectl -n argocd get application zenith-apps
# NAME          SYNC STATUS   HEALTH STATUS
# zenith-apps   Synced        Healthy

# 3. Access the ArgoCD UI
kubectl -n argocd port-forward svc/argocd-server 8443:443
# Open https://localhost:8443
# Username: admin
# Password: <argocd_admin_password from Phase 3 tfvars>

# 4. (Optional) Install the ArgoCD CLI
brew install argocd    # macOS
# or: curl -sSL -o argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64

# 5. Login with the CLI
argocd login argocd.stage.freezenith.com --username admin --password <password>
```

### Creating the Application Manifests Directory

If the `infra/argocd/staging/` directory does not yet exist in the Git repository:

```bash
# Create the directory structure
mkdir -p infra/argocd/staging/tenants
mkdir -p infra/argocd/production/tenants

# Create the Application manifests (copy from the examples in this document)
# Each YAML file in infra/argocd/staging/ becomes an ArgoCD Application

# Commit and push
git add infra/argocd/
git commit -m "feat: add ArgoCD application manifests for staging"
git push origin staging
```

ArgoCD will detect the new files within 3 minutes and create the Applications.

### Adding a New Application

To deploy a new application via ArgoCD:

```bash
# 1. Create the Helm chart
mkdir -p infra/helm/my-new-app
# Add Chart.yaml, values.yaml, templates/

# 2. Create the ArgoCD Application manifest
cat > infra/argocd/staging/my-new-app.yaml << 'EOF'
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-new-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/DoTech/Zenith.git
    targetRevision: staging
    path: infra/helm/my-new-app
    helm:
      releaseName: my-new-app
      valueFiles:
        - values.yaml
        - values-staging.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: zenith-platform
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
EOF

# 3. Commit and push
git add infra/helm/my-new-app/ infra/argocd/staging/my-new-app.yaml
git commit -m "feat: add my-new-app ArgoCD application"
git push origin staging

# 4. ArgoCD detects the new file and creates the Application automatically
# No kubectl apply needed. No Terraform needed.
```

### Removing an Application

```bash
# Remove the Application manifest from Git
git rm infra/argocd/staging/my-new-app.yaml
git commit -m "feat: remove my-new-app"
git push origin staging

# ArgoCD prunes the Application (because prune: true on the root app)
# The Application's finalizer deletes all K8s resources it created
```

---

## Verification Checklist

After the Application manifests are pushed to Git, run through this checklist:

```bash
# ============================================================
# Phase 4 Verification Checklist
# ============================================================

export KUBECONFIG=~/.kube/zenith-staging.yaml

# 1. Root Application is Synced and Healthy
kubectl -n argocd get application zenith-apps -o jsonpath='{.status.sync.status}'
# Expected: Synced

kubectl -n argocd get application zenith-apps -o jsonpath='{.status.health.status}'
# Expected: Healthy

# 2. All child Applications exist
kubectl -n argocd get applications
# NAME              SYNC STATUS   HEALTH STATUS
# zenith-apps       Synced        Healthy
# zenith-platform   Synced        Healthy
# zenith-api        Synced        Healthy
# zenith-landing    Synced        Healthy
# zenith-demo       Synced        Healthy
# tenant-embermind  Synced        Healthy

# 3. Application pods are running
kubectl -n zenith-platform get pods
# NAME                              READY   STATUS    RESTARTS
# zenith-api-<hash>                 1/1     Running   0
# zenith-landing-<hash>             1/1     Running   0
# zenith-mc-demo-<hash>             1/1     Running   0
# zenith-web-demo-<hash>            1/1     Running   0

# 4. Customer namespace exists and pods are running
kubectl -n zenith-embermind get pods
# NAME                          READY   STATUS    RESTARTS
# embermind-mc-<hash>           1/1     Running   0
# embermind-web-<hash>          1/1     Running   0

# 5. Endpoints are reachable
curl -sI https://stage.freezenith.com
# HTTP/2 200

curl -sI https://api.stage.freezenith.com/health
# HTTP/2 200

curl -sI https://demo-ms.stage.freezenith.com
# HTTP/2 200

curl -sI https://demo-cloud.stage.freezenith.com
# HTTP/2 200

curl -sI https://ms.embermind.app
# HTTP/2 200

curl -sI https://cloud.embermind.app
# HTTP/2 200

# 6. ArgoCD Image Updater is running
kubectl -n argocd get pods -l app.kubernetes.io/name=argocd-image-updater
# argocd-image-updater-<hash>    1/1     Running

# 7. Image Updater can reach Harbor
kubectl -n argocd logs -l app.kubernetes.io/name=argocd-image-updater --tail=10
# Should NOT contain authentication errors

# 8. ArgoCD UI is accessible
kubectl -n argocd port-forward svc/argocd-server 8443:443
# Open https://localhost:8443 -- all Applications should show green (Synced + Healthy)

# 9. Verify sync policies are correct
for app in zenith-api zenith-landing zenith-demo; do
  echo "=== $app ==="
  kubectl -n argocd get application $app -o jsonpath='{.spec.syncPolicy}' | jq .
done
# Each should show: automated.prune=true, automated.selfHeal=true

echo "Phase 4 verification complete."
```

---

## Troubleshooting

### Application stuck in "OutOfSync"

```bash
# Check what ArgoCD thinks is different
argocd app diff zenith-api --server argocd.stage.freezenith.com

# Common cause: Helm values produce non-deterministic output (timestamps, random IDs)
# Fix: Use helm.skipCrds or add ignoreDifferences to the Application spec

# Force a sync
argocd app sync zenith-api --server argocd.stage.freezenith.com
```

### Application stuck in "Progressing"

```bash
# The Deployment is rolling out but pods are not becoming Ready
kubectl -n zenith-platform get pods -l app=zenith-api
kubectl -n zenith-platform describe pod zenith-api-<hash>
kubectl -n zenith-platform logs zenith-api-<hash>

# Common causes:
# - Image pull error (Harbor credentials wrong, image does not exist)
# - Readiness probe failing (application crashing on startup)
# - Resource limits too low (OOMKilled)
```

### Application stuck in "Degraded"

```bash
# At least one resource is unhealthy
argocd app get zenith-api --server argocd.stage.freezenith.com

# Look for resources with health status != Healthy
# Common causes:
# - PVC pending (storageClass not available)
# - Service has no endpoints (selector does not match any pods)
# - Certificate not issued (cert-manager issue)
```

### Image Updater not detecting new images

```bash
# Check Image Updater logs
kubectl -n argocd logs -l app.kubernetes.io/name=argocd-image-updater --tail=50

# Common causes:
# - Registry credentials expired (recreate the harbor-helm-creds Secret)
# - Image tag does not match the update strategy (e.g., non-semver tag with semver strategy)
# - Application annotations have a typo

# Force a check
kubectl -n argocd rollout restart deployment argocd-image-updater
```

### Root Application not detecting new files

```bash
# Check the repo server can access Git
kubectl -n argocd logs -l app.kubernetes.io/name=argocd-repo-server --tail=50

# Common causes:
# - GitHub token expired
# - Branch name mismatch (targetRevision vs actual branch)
# - Path mismatch (source.path does not match directory in Git)

# Force a refresh
argocd app get zenith-apps --refresh --server argocd.stage.freezenith.com
```

### Sync conflict: "the object has been modified"

```bash
# This happens when ArgoCD and another controller (e.g., HPA) modify the same resource

# Fix: Add ignoreDifferences for the conflicting field
spec:
  ignoreDifferences:
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas     # Ignore replica count (managed by HPA, not ArgoCD)
```

### Namespace stuck in "Terminating"

```bash
# Check for finalizers preventing deletion
kubectl get namespace zenith-old-customer -o json | jq '.spec.finalizers'

# Remove stuck finalizers (use with caution)
kubectl get namespace zenith-old-customer -o json | \
  jq '.spec.finalizers = []' | \
  kubectl replace --raw /api/v1/namespaces/zenith-old-customer/finalize -f -
```

---

## What Happens Next

With Phase 4 complete, the Zenith V2 platform is fully operational:

```
+-----------------------------------------------------------------------+
|                     Zenith V2 Platform: OPERATIONAL                    |
|                                                                       |
|  Infrastructure (managed by Terraform):                               |
|    cert-manager, CNPG, Keycloak, APISIX, external-dns, Temporal,     |
|    Harbor, Kyverno, Falco, Sealed Secrets, Velero, Monitoring, ArgoCD |
|                                                                       |
|  Applications (managed by ArgoCD):                                    |
|    zenith-api, zenith-landing, zenith-demo, tenant-embermind, ...     |
|                                                                       |
|  Continuous Deployment:                                               |
|    Git push --> CI --> Harbor --> Image Updater --> ArgoCD --> Live    |
|                                                                       |
|  Customer Provisioning:                                               |
|    Signup --> Temporal --> Git commit --> ArgoCD --> Tenant live       |
|                                                                       |
+-----------------------------------------------------------------------+
```

### What You Can Now Do

1. **Push code and see it deployed automatically.** Change a Go endpoint, push to
   staging, and within 5-8 minutes it is live.

2. **Onboard customers without manual intervention.** The signup flow triggers Temporal,
   which creates Keycloak realm + database + S3 bucket + ArgoCD Application. The
   customer's environment is ready in minutes.

3. **Roll back any application in seconds.** Use the ArgoCD UI, CLI, or Git revert.
   Every deployment is auditable in Git history.

4. **Scale confidently.** HPA handles pod scaling, ArgoCD handles deployment, and the
   App-of-Apps pattern handles adding new applications. Adding the 100th customer is
   the same process as adding the 1st.

5. **Sleep at night.** Velero backs up the cluster, CNPG archives WAL to S3, Falco
   watches for anomalies, Kyverno blocks bad deployments, and Cilium encrypts all
   pod traffic. If something breaks, the monitoring stack alerts you before customers
   notice.

### Related Documents

| Document | Description |
|----------|-------------|
| [00-overview.md](./00-overview.md) | Full architecture overview and system diagrams |
| [01-phase1-hetzner-cloudflare.md](./01-phase1-hetzner-cloudflare.md) | Phase 1: Hetzner VM + Cloudflare DNS |
| [02-phase2-ansible-k3s.md](./02-phase2-ansible-k3s.md) | Phase 2: Ansible + k3s + Cilium |
| [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) | Phase 3: All infra components via Terraform |
| [06-security-model.md](./06-security-model.md) | Defense-in-depth security architecture |
| [09-migration-v1-to-v2.md](./09-migration-v1-to-v2.md) | Migration plan from V1 to V2 |
