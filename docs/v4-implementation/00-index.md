# Zenith V4 — Complete Platform Guide & 10/10 Implementation Plan

> **Status:** Active — Single Source of Truth for Development
> **Version:** 4.0.0
> **Created:** 2026-03-13
> **Audience:** Junior developers joining the team, senior architects, AI agents
> **Goal:** Bring every section of Zenith to 10/10

---

## How to Read This Guide

You are a new developer joining the Zenith team. This guide will take you from zero to productive in **5 days**. Read it in order — each section builds on the previous one.

### Day 1: Understand the Product (2-3 hours)
| # | Document | What You'll Learn |
|---|----------|-------------------|
| 01 | [What is Zenith](./01-what-is-zenith.md) | The product, who pays, what they get, pricing strategy |

### Day 2: Understand the Architecture (4-5 hours)
| # | Document | What You'll Learn |
|---|----------|-------------------|
| 02 | [Architecture Deep Dive](./02-architecture.md) | System overview, 8 layers, namespaces, request flows, security |

### Day 3: Learn the Backend (6-8 hours)
| # | Document | What You'll Learn |
|---|----------|-------------------|
| 03 | [Backend Complete Guide](./03-backend-guide.md) | Go backend, hexagonal architecture, every entity/service/handler |

### Day 4: Learn the Frontend + Infrastructure (6-8 hours)
| # | Document | What You'll Learn |
|---|----------|-------------------|
| 04 | [Frontend Complete Guide](./04-frontend-guide.md) | 3 Next.js apps, pages, hooks, API client, styling |
| 05 | [Infrastructure Guide](./05-infrastructure.md) | Terraform, Helm, Ansible, Kubernetes, networking |

### Day 5: CI/CD, Operations, and Start Contributing (4-6 hours)
| # | Document | What You'll Learn |
|---|----------|-------------------|
| 06 | [CI/CD & Deployment Guide](./06-cicd-guide.md) | GitHub Actions, staging deploy, production release, smoke tests |
| 07 | [Day-2 Operations](./07-day2-operations.md) | Daily tasks, debugging, runbooks, monitoring |
| 08 | [10/10 Implementation Plan](./08-implementation-plan.md) | What we need to build to reach 10/10 in every section |
| 09 | [Quick Reference](./09-quick-reference.md) | Cheatsheet: commands, paths, URLs, credentials |

---

## Current Scores (2026-03-13)

| Section | Score | Target | Gap |
|---------|-------|--------|-----|
| Auth & Security | 9/10 | 10/10 | Account lockout, auth audit events |
| Database & Migrations | 8.5/10 | 10/10 | RLS, read replicas, migration CLI |
| API Gateway | 8.5/10 | 10/10 | Domain verification, per-gateway rate-limit |
| App Deployment | 8/10 | 10/10 | Rollback UI, deploy history, build log streaming |
| Storage (S3) | 7.5/10 | 10/10 | Quota enforcement, file preview, CDN |
| Monitoring | 7/10 | 10/10 | Tracing, log aggregation, custom alerts, uptime |
| CI/CD | 8/10 | 10/10 | E2E in CI, preview envs, canary deploys |
| Frontend (Web) | 7/10 | 10/10 | Tests, error boundaries, a11y, i18n |
| Frontend (Admin) | 7.5/10 | 10/10 | Real-time, bulk ops, export |
| Infrastructure | 8.5/10 | 10/10 | DR automation, multi-cluster, cost monitoring |
| Billing | 6/10 | 10/10 | **Stripe integration, invoices, metering** |
| Team & RBAC | 7/10 | 10/10 | Granular permissions, activity feed |
| Support | 7/10 | 10/10 | SLA tracking, real-time chat, KB |
| Landing | 6.5/10 | 10/10 | SEO, blog, docs site, social proof |

**Overall: 7.5/10 → Target: 10/10**

---

## Critical Rules

1. **Read `AGENTS.md`** before writing any code — it has all Lich Framework rules
2. **Never write code manually** — use `lich make entity/service/api` generators
3. **Hexagonal architecture is non-negotiable** — entities never import infrastructure
4. **Staging branch = deployment** — ArgoCD watches `staging`, not `main`
5. **SHA tags for staging, semver for production** — never arbitrary version numbers
6. **Operators first** — every stateful service uses a K8s Operator (not plain Helm)
7. **Customers never see infrastructure names** — they see "API Gateway", not "APISIX"
8. **Test before deploy** — `go vet`, `next lint`, smoke tests must pass

---

## Repository Structure (Bird's Eye View)

```
Zenith/
├── apps/                          ← 3 Frontend Applications
│   ├── landing/                   ← Public marketing site (Next.js, port 3200)
│   ├── web/                       ← Customer dashboard (Next.js, port 3000)
│   └── mission-control/           ← Admin panel (Next.js, port 3100)
│
├── services/                      ← Backend Services
│   ├── api/                       ← Main API (Go/Fiber, port 8080)
│   │   ├── cmd/server/main.go     ← DI wiring, route registration (1500 lines)
│   │   └── internal/              ← Hexagonal architecture layers
│   │       ├── entities/          ← 37 domain model files (pure Go, zero imports)
│   │       ├── ports/             ← 46 interfaces (repositories + infrastructure)
│   │       ├── services/          ← 24 business logic files
│   │       ├── handlers/          ← 83 HTTP handler files (376 endpoints)
│   │       ├── adapters/          ← 16 adapter packages
│   │       │   ├── postgres/      ← 34 PostgreSQL implementations + 37 migrations
│   │       │   ├── memory/        ← 45 in-memory implementations (dev/test)
│   │       │   ├── k8sclient/     ← Kubernetes client (real + memory)
│   │       │   ├── stripeclient/  ← Stripe payments
│   │       │   ├── keycloakclient/← Keycloak identity
│   │       │   ├── s3client/      ← Hetzner S3
│   │       │   ├── promclient/    ← Prometheus metrics
│   │       │   ├── lokiclient/    ← Loki logs
│   │       │   └── ...            ← 8 more adapter packages
│   │       ├── dto/               ← Request/response shapes
│   │       ├── middleware/        ← 10 middleware files (auth, CORS, security)
│   │       └── config/            ← 65+ environment variables
│   └── operator/                  ← Kubernetes operator for Zenith CRDs
│
├── infra/                         ← Infrastructure as Code
│   ├── terraform/                 ← 49 Terraform files
│   │   ├── staging/               ← Hetzner VM + Cloudflare DNS
│   │   ├── staging-k8s/           ← K8s platform (calls k8s-platform module)
│   │   ├── production/            ← Production VMs
│   │   ├── production-k8s/        ← Production K8s
│   │   └── modules/               ← Reusable modules
│   │       ├── k8s-platform/      ← 25 .tf files (cert-manager→APISIX→ArgoCD→everything)
│   │       ├── k3s-server/        ← Hetzner VM provisioning
│   │       ├── dns/               ← Cloudflare DNS
│   │       └── storage/           ← Hetzner S3
│   ├── helm/                      ← 11 Helm charts, 105 templates
│   │   ├── zenith-api/            ← API server chart
│   │   ├── zenith-web/            ← Web dashboard chart
│   │   ├── zenith-mc/             ← Mission Control chart
│   │   ├── zenith-landing/        ← Landing page chart
│   │   ├── zenith-platform/       ← Shared platform resources
│   │   └── ...                    ← 6 more charts
│   ├── ansible/                   ← Server provisioning playbooks
│   └── scripts/                   ← Smoke tests, utilities
│
├── .github/workflows/             ← 11 CI/CD workflows
├── docs/                          ← Documentation (you are here)
├── .lich/                         ← Lich CLI framework
├── Makefile                       ← Build, test, deploy automation
└── AGENTS.md                      ← AI agent rules (= CLAUDE.md)
```

---

## Key URLs

| Service | Staging URL | Purpose |
|---------|-------------|---------|
| Landing | stage.freezenith.com | Marketing site |
| Web Dashboard | app.stage.freezenith.com | Customer platform |
| Mission Control | mc.stage.freezenith.com | Admin panel |
| API | api.stage.freezenith.com | Backend API |
| Keycloak | auth.stage.freezenith.com | Identity provider |
| Harbor (Internal) | registry.stage.freezenith.com | Platform images |
| Harbor (Customer) | hub.stage.freezenith.com | Customer images |
| ArgoCD | argocd.stage.freezenith.com | GitOps dashboard |
| Customer Apps | *.apps.stage.freezenith.com | Deployed customer apps |
| API Gateways | *.gw.stage.freezenith.com | Customer API gateways |

---

## Team & Contacts

| Role | Who | Contact |
|------|-----|---------|
| Founder / DevOps Lead | Babak | — |
| AI Assistant | Claude | Active in codebase |

---

**Start reading → [01 — What is Zenith](./01-what-is-zenith.md)**
