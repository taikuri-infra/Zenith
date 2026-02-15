# Zenith - AI Agent Configuration

## What is Zenith?
100% free, open-source, Kubernetes-native PaaS on Hetzner Cloud.
- One command installs everything: `zen install --provider hetzner --token hc_xxx`
- Users get: Apps, Databases, Storage, Auth, Gateway, Monitoring, Registry
- Operators get: Mission Control (cluster management, upgrades, modules)
- Domain: freezenith.com

## Project Structure
```
/opt/zenith/
├── apps/
│   ├── web/              # User-facing platform (Next.js 15, port 3000)
│   ├── mission-control/  # Operator panel (Next.js 15, port 3100)
│   └── landing/          # freezenith.com (TODO)
├── packages/
│   └── ui/               # Shared design system
├── services/
│   ├── api/              # Go API server (TODO)
│   ├── operator/         # Zenith K8s operator (TODO)
│   └── auth/             # Auth service - Keycloak-like (TODO)
├── cli/                  # zen CLI - Go + Charm TUI (TODO)
├── helm/                 # Helm charts (TODO)
├── docs/                 # Architecture & design docs
│   ├── ARCHITECTURE.md   # System architecture
│   ├── FRONTEND.md       # Frontend design spec
│   ├── PHASES.md         # Implementation phases (~456 tasks)
│   ├── MICROSERVICES.md  # Service design
│   └── DESIGN.md         # Visual design system
├── CLAUDE.md             # THIS FILE - AI agent entry point
└── TASKS.md              # Implementation tasks - READ THIS TO START WORKING
```

## Must Read Before Working
1. **TASKS.md** - Prioritized implementation tasks (START HERE)
2. **docs/ARCHITECTURE.md** - System architecture and decisions
3. **docs/FRONTEND.md** - Frontend page designs
4. **docs/PHASES.md** - Full implementation phases

## Tech Stack
| Component | Technology |
|-----------|-----------|
| Frontend | Next.js 15, TypeScript, Tailwind CSS, Lucide icons |
| Backend API | Go (Fiber/Echo), CRD-driven |
| Operator | Go, controller-runtime, kubebuilder |
| CLI | Go, Cobra, Charm (bubbletea, lipgloss, bubbles, huh) |
| Auth | Keycloak-like (OpenID Connect + SAML), per-tenant realms |
| API Gateway | Kong (Kubernetes Operator, CRDs) |
| Monitoring | Grafana + Loki + Prometheus |
| Registry | Harbor or custom (ECR-like) |
| Database Operators | CNPG (PostgreSQL), Redis Operator, MongoDB Operator |
| Cluster Management | CAPI + CAPH (Cluster API Provider Hetzner) |
| Infrastructure | Hetzner Cloud (servers, volumes, LBs, DNS, object storage) |

## Architecture
```
Internet
  ├── LB → Frontend Apps (Next.js, React, etc.)
  ├── LB → Kong API Gateway → Backend Services
  │           ├── JWT validation via Zenith Auth
  │           ├── Rate limiting, CORS, plugins
  │           └── Route management per tenant
  └── Mobile → Kong API Gateway → Backend Services

Management Plane (€5 CX22):
  k3s + CAPI + CAPH → manages tenant clusters
  Mission Control UI → operator dashboard

Tenant Clusters (CAPI-managed):
  Zenith Operator → watches CRDs → creates Hetzner resources
  Service Operators → CNPG, Redis, etc.
```

## Current State (Feb 2026)
- Frontend mockups: Web platform (17 pages) + Mission Control (10 pages)
- Design system: Dark theme, emerald accent, shared UI package
- Architecture docs: Complete
- Backend API: Not started
- Zenith Operator: Not started
- CLI: Not started
- Auth Service: Not started
- Helm Charts: Not started

## Key Decisions
- Everything is a CRD. User creates app -> Backend creates CRD -> Operator reconciles.
- Auth is built-in (not external Keycloak). Each tenant gets a realm.
- Kong for API Gateway (has K8s operator, integrates with JWT).
- CAPI for cluster lifecycle (zero-downtime K8s upgrades).
- GitOps-friendly: `zen export`, `zen apply`, `zen diff`.
- UX must be dead simple. Progressive disclosure. No jargon.
