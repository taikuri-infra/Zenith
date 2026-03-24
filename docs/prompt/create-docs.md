# Zenith Platform — Complete Documentation Generator

> **What this is:** A prompt you give to an AI agent to generate comprehensive, per-service documentation for the entire Zenith platform.
> **When to use:** After finalizing a version (e.g., V2, V3, V5...) and you want a full reference doc set.
> **Output:** A folder `docs/vN-architecture/` with numbered files, one per technology/service.

---

## PROMPT START — Copy everything below this line

---

You are a senior platform engineer documenting the **Zenith PaaS** platform. Your job is to create a complete documentation set for the current state of the codebase. This documentation must be so thorough that a junior developer with basic Kubernetes knowledge can understand every component, debug issues, and make changes confidently.

### PHASE 1: DISCOVERY — Scan the entire codebase

Before writing ANY documentation, you MUST scan and catalog every technology in the project. Do NOT rely on memory or assumptions — read the actual files.

#### 1.1 — Infrastructure Discovery

Scan these locations and extract every technology, version, and configuration:

```
# Terraform — every .tf file defines a platform service
infra/terraform/modules/k8s-platform/*.tf
  → Read EVERY .tf file. Each one is a service (e.g., gateway.tf = APISIX, identity.tf = Keycloak,
    observability.tf = Prometheus+Grafana+Loki, registry.tf = Harbor, etc.)
  → Extract: Helm chart names, versions, namespaces, resource limits, PVC sizes, replicas
  → Extract: all helm_release resources, kubernetes_manifest resources, null_resource provisioners

infra/terraform/modules/dns/*.tf          → DNS provider, records, wildcard patterns
infra/terraform/modules/k3s-server/*.tf   → Server provisioning, cloud-init, firewall rules
infra/terraform/modules/storage/*.tf      → S3/object storage configuration
infra/terraform/main.tf                   → Module composition, provider versions
infra/terraform/variables.tf              → All configurable parameters
```

```
# Ansible — server bootstrap and K3s installation
infra/ansible/playbooks/*.yml             → All playbooks (server-setup, infra, apps, build, teardown)
infra/ansible/roles/*/                    → Each role (k3s, cilium, cert-manager, common)
infra/ansible/group_vars/                 → Variable defaults per environment
infra/ansible/inventory/                  → Server inventory
```

```
# Helm Charts — application deployment templates
infra/helm/*/                             → Every Helm chart
  → For each chart: read Chart.yaml (dependencies), values.yaml (defaults),
    values-staging.yaml, values-production.yaml, and ALL templates/*.yaml
  → Pay special attention to:
    - zenith-platform/ (shared platform services: Keycloak secrets, CNPG, etc.)
    - zenith-api/ (API deployment, service, IngressRoute, APISIX routes)
    - zenith-operator/ (CRDs in crds/ folder)
    - monitoring/ (Prometheus, Grafana, Loki, Alertmanager — check Chart.yaml dependencies)
    - zenith-tenant/ (per-customer namespace template)
```

```
# ArgoCD Applications — GitOps definitions
infra/argocd/staging/*.yaml               → Every ArgoCD Application CR
  → Extract: source repo, target revision, sync policy, namespace, Helm values
```

```
# GitHub Actions — CI/CD pipelines
.github/workflows/*.yml                   → Every workflow
  → Extract: triggers, jobs, steps, secrets used, Docker build targets
  → Map: which workflow deploys which service
```

```
# Kubernetes manifests (non-Helm)
infra/k8s/*.yaml                          → Raw K8s manifests (demo, ingress, certs, postgres, etc.)
```

#### 1.2 — Backend Discovery

```
# Go API Service — the core backend
services/api/internal/entities/*.go       → ALL domain entities (read EVERY file)
services/api/internal/ports/*.go          → ALL interfaces (repositories.go, infrastructure.go)
services/api/internal/services/*.go       → ALL business logic (read EVERY file, including subdirectories)
services/api/internal/services/deploy/    → Build/deploy pipeline logic
services/api/internal/services/cluster/   → Cluster provisioning
services/api/internal/services/temporal/  → Temporal workflow definitions
services/api/internal/services/autoscale/ → Autoscaler logic
services/api/internal/handlers/*.go       → ALL HTTP handlers (read EVERY file)
services/api/internal/middleware/*.go     → Auth, logging, security middleware
services/api/internal/adapters/*/         → ALL adapter implementations:
  - postgres/      → PostgreSQL queries (read *.go files for SQL patterns)
  - memory/        → In-memory test implementations
  - k8sclient/     → Kubernetes API interactions
  - keycloakclient/ → Keycloak realm/user management
  - stripeclient/  → Stripe billing integration
  - harborclient/  → Harbor registry management
  - s3client/      → Hetzner S3 object storage
  - hetznerclient/ → Hetzner Cloud API (servers, firewalls)
  - capiclient/    → Cluster API provisioning
  - lokiclient/    → Loki log queries
  - promclient/    → Prometheus metric queries
  - natsclient/    → NATS messaging
  - redisclient/   → Redis caching/sessions
  - resendclient/  → Resend email delivery
services/api/internal/config/config.go    → ALL environment variables and defaults
services/api/internal/telemetry/          → OpenTelemetry setup
services/api/cmd/server/main.go           → DI composition root (how everything wires together)
services/api/Dockerfile                   → Build process, multi-stage layers
services/api/go.mod                       → All Go dependencies with versions
```

```
# Go Operator
services/operator/internal/controllers/   → Kubernetes operator reconcilers
services/operator/internal/provider/      → Provider implementations
services/operator/Dockerfile
services/operator/go.mod
```

```
# Go CLI
services/cli/cmd/                         → CLI command structure
services/cli/internal/                    → CLI logic
services/cli/go.mod
```

```
# Terraform Provider
services/terraform-provider-zenith/       → Custom TF provider for Zenith API
```

```
# Database Migrations — the COMPLETE schema
services/api/internal/adapters/postgres/migrations/*.up.sql
  → Read ALL migration files in order. This defines the entire database schema.
  → Document every table, column, index, and constraint.
```

#### 1.3 — Frontend Discovery

```
# Web Dashboard (customer-facing)
apps/web/src/                             → Next.js app
  → app/ directory structure (pages, layouts)
  → lib/api.ts (API client — lists ALL endpoints the frontend calls)
  → lib/demo-api.ts (demo mode mock)
  → components/ (UI component library)
  → package.json (dependencies, versions)

# Mission Control (admin panel)
apps/mission-control/src/                 → Next.js admin app
  → Same structure scan as web

# Landing Page
apps/landing/src/                         → Marketing site
  → package.json

# Shared UI Package
packages/ui/                              → Shared component library
```

#### 1.4 — Other Files

```
docker-compose.yml                        → Local development setup
Makefile                                  → All make targets (build, deploy, test, lint)
.lich/                                    → Lich framework configuration
openspec/                                 → OpenSpec change management
scripts/                                  → Utility scripts
.semgrep.yml                              → Security scanning rules
```

---

### PHASE 2: GENERATE THE DOCUMENTATION

Create a folder `docs/vN-architecture/` (replace N with the version number).

#### File Naming Convention

```
00-overview.md                    — Platform overview + architecture diagram + document index
01-hetzner-infrastructure.md      — Hetzner Cloud servers, networking, firewall
02-ansible-bootstrap.md           — Server provisioning, K3s installation
03-k3s-kubernetes.md              — K3s cluster configuration, node setup
04-cilium-networking.md           — CNI, network policies, WireGuard encryption, Hubble
05-traefik-ingress.md             — TLS termination, IngressRoutes, middleware
06-apisix-gateway.md              — API gateway, plugins, rate limiting, JWT
07-cert-manager-tls.md            — Certificate management, Let's Encrypt, ClusterIssuers
08-external-dns.md                — Automatic DNS record management
09-argocd-gitops.md               — GitOps deployment, Application CRDs, sync policies
10-keycloak-identity.md           — Identity management, realms, OIDC, SSO
11-cnpg-postgresql.md             — CloudNativePG, database clusters, backups, WAL archiving
12-harbor-registry.md             — Container registry, projects, robot accounts, replication
13-monitoring-stack.md            — Prometheus, Grafana, Loki, Alertmanager, dashboards
14-sealed-secrets.md              — Secret management, encryption, rotation
15-temporal-workflows.md          — Workflow engine, provisioning workflows, activities
16-keda-autoscaling.md            — Event-driven autoscaling, scale-to-zero, HTTPScaledObject
17-hetzner-s3-storage.md          — Object storage, buckets, tenant isolation
18-backend-architecture.md        — Go API service, hexagonal architecture, dependency rules
19-api-endpoints.md               — Complete API reference (every route, method, auth requirement)
20-database-schema.md             — Complete schema from migrations, ER diagram, indexes
21-frontend-web.md                — Customer dashboard, Next.js, pages, components
22-frontend-mission-control.md    — Admin panel, features, pages
23-frontend-landing.md            — Marketing site
24-ci-cd-pipelines.md             — GitHub Actions workflows, build process, versioning strategy
25-helm-charts.md                 — All Helm charts, values, templates
26-terraform-modules.md           — IaC modules, state management, apply workflow
27-operator.md                    — Kubernetes operator, CRDs, reconciliation loops
28-cli.md                         — CLI tool, commands, installation
29-terraform-provider.md          — Custom Terraform provider
30-security-model.md              — Defense-in-depth, all security layers, WAF, network policies
31-backup-disaster-recovery.md    — CNPG backups, Velero, S3, restore procedures
32-observability.md               — Metrics, logs, traces, dashboards, alerting rules
33-multi-tenancy.md               — Namespace isolation, per-tenant resources, plan limits
34-billing-payments.md            — Stripe integration, plans, subscriptions, metering
35-email-notifications.md         — Resend, email templates, notification system
36-local-development.md           — docker-compose, dev setup, mock adapters
37-day-to-day-operations.md       — Runbooks, common tasks, troubleshooting
IMPLEMENTATION.md                 — Implementation status, what's done, what's pending
SYSTEM-MAP.md                     — Complete system map, service dependencies, port numbers
VERIFICATION.md                   — How to verify each component is working
HANDOVER.md                       — Quick-start for new team members
```

> **IMPORTANT:** This file list is a TEMPLATE. After Phase 1 discovery, you may need to ADD files for technologies you found that aren't listed here, or REMOVE files for technologies that don't exist in this version. The goal is ONE file per technology/service — no gaps, no duplicates.

#### Required Style for EVERY File

Every documentation file MUST follow this exact format:

```markdown
# NN — Service/Technology Name

> **Purpose:** One-line description of what this does and why it exists in Zenith
> **Audience:** Who should read this (all developers, backend only, infra only, etc.)
> **Last Updated:** YYYY-MM-DD
> **Author:** Babak + Claude (Documentation Session)
> **Related:** [XX-other-doc.md](./XX-other-doc.md) (relationship description)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose It](#2-why-we-chose-it)
3. [Architecture Diagram](#3-architecture-diagram)
4. [How It Works](#4-how-it-works)
5. [Configuration Reference](#5-configuration-reference)
6. [Integration Points](#6-integration-points)
7. [Troubleshooting](#7-troubleshooting)
8. [Upgrade Path](#8-upgrade-path)

---

## 1. Overview

[2-3 paragraphs explaining what this component does, where it runs,
and its role in the Zenith platform]

---

## 2. Why We Chose It

| Feature | This Tool | Alternative A | Alternative B |
|---------|-----------|---------------|---------------|
| ...     | ...       | ...           | ...           |

**Decision:** [Why this was chosen over alternatives — include cost,
learning goals, ecosystem fit]

---

## 3. Architecture Diagram

[ASCII box-drawing diagram showing this component in context.
Use ┌─┐│└─┘ box characters. Show data flow with arrows ──▶.
Include namespace, port numbers, resource limits.]

```
┌─────────────────────────────────────────────────────────────┐
│                    COMPONENT IN CONTEXT                       │
│                    Namespace: xxx                             │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │ Upstream      │───▶│ This Service │───▶│ Downstream   │  │
│  │ :port         │    │ :port        │    │ :port        │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## 4. How It Works

### 4.1 — [Flow/Process Name]

Step-by-step explanation with diagrams:

```
Step 1: [What happens]
    │
    ▼
Step 2: [What happens next]
    │
    ▼
Step 3: [Result]
```

### 4.2 — [Another Flow]

[...]

---

## 5. Configuration Reference

### Where Config Lives

| File | What It Configures |
|------|--------------------|
| `infra/terraform/modules/k8s-platform/xxx.tf` | Helm release, versions |
| `infra/helm/xxx/values.yaml` | Default values |
| `infra/helm/xxx/values-staging.yaml` | Staging overrides |

### Key Parameters

| Parameter | Default | Staging | Production | Description |
|-----------|---------|---------|------------|-------------|
| `replicas` | 1 | 1 | 3 | Number of pods |
| `resources.cpu` | 100m | 100m | 500m | CPU request |
| `resources.memory` | 128Mi | 128Mi | 512Mi | Memory request |

### Environment Variables (if applicable)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `XXX_URL` | Yes | — | Connection URL |

---

## 6. Integration Points

### What Depends on This

| Component | How It Uses This | Impact If Down |
|-----------|------------------|----------------|
| zenith-api | Calls XXX API | Feature Y breaks |

### What This Depends On

| Dependency | Purpose | Impact If Down |
|------------|---------|----------------|
| PostgreSQL | State storage | Complete outage |

---

## 7. Troubleshooting

### Problem: [Common Issue Name]

**Symptoms:**
- What the user/operator sees

**Diagnosis:**
```bash
# Commands to investigate
kubectl get pods -n namespace
kubectl logs -n namespace pod-name
```

**Fix:**
```bash
# Commands to fix
kubectl rollout restart -n namespace deployment/xxx
```

### Problem: [Another Common Issue]

[...]

---

## 8. Upgrade Path

### Current Version
- Chart: `x.y.z`
- App: `a.b.c`

### How to Upgrade
1. Update version in `infra/terraform/modules/k8s-platform/xxx.tf`
2. Run `terraform plan` to verify
3. Run `terraform apply`
4. Verify: [specific verification steps]

### Breaking Changes to Watch For
- [List any known gotchas between versions]
```

---

### PHASE 3: QUALITY RULES

These rules are NON-NEGOTIABLE. Violating any of them makes the documentation useless.

#### Rule 1: Read Before You Write
- You MUST read the actual file before documenting it
- NEVER guess versions, port numbers, resource limits, or configurations
- If a value is in `values-staging.yaml`, document the STAGING value. If in `values-production.yaml`, document the PRODUCTION value. Show BOTH.

#### Rule 2: Every Diagram Must Be Accurate
- ASCII diagrams must show REAL namespace names, REAL port numbers, REAL service names
- Show actual data flow, not abstract concepts
- Include replica counts and resource limits in diagrams

#### Rule 3: Every File Must Be Self-Contained
- A reader should understand the component by reading ONLY that file
- Cross-reference other files but don't require reading them first
- Include enough context in each file to be independently useful

#### Rule 4: Show Real Examples
- Configuration examples must be REAL values from the codebase, not placeholders
- Code snippets must be copy-pasteable and correct
- Troubleshooting commands must work in the actual cluster

#### Rule 5: No Fluff
- No motivational text ("In today's cloud-native world...")
- No obvious statements ("Kubernetes is a container orchestration platform...")
- Every sentence must convey information a developer needs
- Be concise but complete — if it takes 100 lines to be thorough, use 100 lines

#### Rule 6: Document What EXISTS, Not What's Planned
- Only document currently implemented, deployed, working components
- If something is partially implemented, note it clearly with status markers:
  - ✅ Implemented and deployed
  - 🔧 Implemented but not yet deployed
  - 📋 Planned but not implemented
- Do NOT document future plans as if they're current reality

#### Rule 7: Complete Technology Coverage
After finishing all files, verify:
- Every Terraform resource has been documented
- Every Helm chart has been documented
- Every Go service/handler has been documented
- Every GitHub workflow has been documented
- Every database table has been documented
- Every adapter/integration has been documented
- If you find something undocumented, CREATE a file for it

---

### PHASE 4: FINAL VERIFICATION

After creating all files, generate `VERIFICATION.md` with:

1. **Coverage Checklist** — List every technology found in Phase 1, mark ✅ if documented
2. **Cross-Reference Matrix** — Which doc references which other docs
3. **Verification Commands** — One command per component to verify it's running
4. **File Sizes** — Each file should be 300-1500 lines. Under 300 means it's too shallow. Over 1500 means it should be split.

---

## PROMPT END

---

## Usage Instructions

1. Open a new Claude Code conversation in the Zenith project root
2. Copy everything between `PROMPT START` and `PROMPT END` above
3. Replace `vN` with your target version number (e.g., `v6`)
4. Paste the prompt
5. Claude will:
   - Spend time reading the codebase (Phase 1)
   - Generate 35+ documentation files (Phase 2)
   - Self-verify coverage (Phase 4)
6. Review the output, especially:
   - Are all technologies covered?
   - Are diagrams accurate?
   - Are version numbers correct?
7. Commit the docs folder

### Tips

- **Token limit:** This generates a LOT of content. If Claude runs out of context, say "continue from file NN" and it will pick up where it left off.
- **Incremental updates:** To update just one file, say "Regenerate file 06-apisix-gateway.md with the latest state of the codebase"
- **Adding a new service:** Say "Add file NN-new-service.md following the standard template"
- **Style reference:** If Claude deviates from the style, point it to `docs/v2-architecture/13-apisix-gateway.md` as the gold standard example
