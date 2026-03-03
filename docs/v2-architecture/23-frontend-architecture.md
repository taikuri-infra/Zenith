# 23 — Frontend Architecture (Next.js Apps)

> **Purpose:** Understand how the three Next.js applications are structured, how they communicate with the backend, and how to add features.
> **Audience:** Any developer working on the frontend UI — landing page, user dashboard, or admin panel.
> **Last Updated:** 2026-03-03
> **Related:** [21-local-development-setup.md](./21-local-development-setup.md) (running locally), [22-day-to-day-operations.md](./22-day-to-day-operations.md) (adding pages), [13-apisix-gateway.md](./13-apisix-gateway.md) (API routing)

---

## Table of Contents

1. [Overview](#1-overview)
2. [The Three Apps](#2-the-three-apps)
3. [Monorepo Structure](#3-monorepo-structure)
4. [Shared Patterns](#4-shared-patterns)
5. [Data Fetching & API Communication](#5-data-fetching--api-communication)
6. [Authentication Flow](#6-authentication-flow)
7. [Demo Mode](#7-demo-mode)
8. [Styling & Design System](#8-styling--design-system)
9. [App: Landing Page (`apps/landing/`)](#9-app-landing-page-appslanding)
10. [App: Web Dashboard (`apps/web/`)](#10-app-web-dashboard-appsweb)
11. [App: Mission Control (`apps/mission-control/`)](#11-app-mission-control-appsmission-control)
12. [Docker Build & Deployment](#12-docker-build--deployment)
13. [Troubleshooting](#13-troubleshooting)

---

## 1. Overview

Zenith has three Next.js applications, each serving a different audience:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    THREE FRONTEND APPS                                    │
│                                                                          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐      │
│  │  LANDING PAGE     │  │  WEB DASHBOARD   │  │  MISSION CONTROL │      │
│  │  apps/landing/    │  │  apps/web/       │  │  apps/mission-   │      │
│  │                    │  │                   │  │  control/        │      │
│  │  WHO: Public       │  │  WHO: Customers   │  │  WHO: Operators  │      │
│  │                    │  │                   │  │  (Zenith team)   │      │
│  │  WHAT:             │  │  WHAT:            │  │  WHAT:           │      │
│  │  • Marketing site  │  │  • App management │  │  • Cluster mgmt  │      │
│  │  • Pricing         │  │  • Database mgmt  │  │  • Customer mgmt │      │
│  │  • Documentation   │  │  • Deploy engine  │  │  • Billing plans │      │
│  │  • Sign up CTAs    │  │  • Auth/IAM       │  │  • Audit logs    │      │
│  │                    │  │  • Billing        │  │  • Platform state│      │
│  │  AUTH: None        │  │  • Monitoring     │  │                   │      │
│  │  (public pages)    │  │                   │  │  AUTH: Admin JWT  │      │
│  │                    │  │  AUTH: User JWT   │  │                   │      │
│  │  PORT: 3200        │  │  PORT: 3000       │  │  PORT: 3100      │      │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘      │
│           │                      │                      │                │
│           │ links to             │ REST API             │ REST API       │
│           ▼                      ▼                      ▼                │
│  ┌──────────────┐      ┌──────────────────────────────────────┐        │
│  │ Web Dashboard │      │         zenith-api (Go / Fiber)      │        │
│  │ (login/signup)│      │         localhost:8080                │        │
│  └──────────────┘      └──────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. The Three Apps

| App | Port | URL (Staging) | Audience | Auth |
|-----|------|---------------|----------|------|
| **Landing** | 3200 | `stage.freezenith.com` | Public visitors | None |
| **Web** | 3000 | `app.stage.freezenith.com` | Platform customers | JWT (user) |
| **Mission Control** | 3100 | `mc.stage.freezenith.com` | Zenith operators | JWT (admin) |

---

## 3. Monorepo Structure

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    MONOREPO LAYOUT                                        │
│                                                                          │
│  Zenith/                                                                 │
│  ├── apps/                         ← Frontend applications               │
│  │   ├── landing/                  ← Marketing landing page              │
│  │   │   ├── src/                                                        │
│  │   │   │   ├── app/              ← Pages (App Router)                 │
│  │   │   │   ├── components/       ← UI components                      │
│  │   │   │   └── lib/              ← Utilities, URL config             │
│  │   │   ├── package.json                                                │
│  │   │   ├── next.config.ts                                              │
│  │   │   ├── tailwind.config.ts                                          │
│  │   │   └── Dockerfile                                                  │
│  │   ├── web/                      ← User dashboard                      │
│  │   │   ├── src/                                                        │
│  │   │   │   ├── app/              ← Pages (25+ routes)                 │
│  │   │   │   ├── components/       ← Shared UI (shell, sidebar, etc.)   │
│  │   │   │   ├── hooks/            ← useApi, useAuth, useMutation       │
│  │   │   │   └── lib/              ← api.ts, mock-data.ts, demo-api.ts │
│  │   │   ├── package.json                                                │
│  │   │   └── Dockerfile                                                  │
│  │   └── mission-control/          ← Admin/operator panel                │
│  │       ├── src/                  ← (same structure as web/)            │
│  │       ├── package.json                                                │
│  │       └── Dockerfile                                                  │
│  ├── packages/                     ← Shared packages                     │
│  │   └── ui/                       ← @zenith/ui (shared component lib)  │
│  ├── package.json                  ← Root (pnpm workspaces + Turbo)     │
│  ├── pnpm-workspace.yaml           ← Workspace definition               │
│  └── turbo.json                    ← Task orchestration                  │
│                                                                          │
│  Package Manager: pnpm 10.29.3                                           │
│  Build System: Turborepo                                                 │
│  Framework: Next.js 15.1.0 (App Router)                                  │
│  React: 19                                                               │
│  Node: 20+ (Alpine in Docker)                                            │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Shared Patterns

### Page Layout

Every authenticated page uses the `Shell` component:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    PAGE LAYOUT                                           │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  DemoBanner (shown only when NEXT_PUBLIC_DEMO_MODE=true)        │    │
│  │  "This is a demo environment..."                                │    │
│  ├──────────┬──────────────────────────────────────────────────────┤    │
│  │          │                                                       │    │
│  │  Sidebar │  Header (top bar — page title, user menu)            │    │
│  │  (224px) │  ──────────────────────────────────────────────────  │    │
│  │          │                                                       │    │
│  │  Links:  │  Main Content Area (scrollable)                      │    │
│  │  • Home  │                                                       │    │
│  │  • Apps  │  ┌──────────┐  ┌──────────┐  ┌──────────┐          │    │
│  │  • Deploy│  │ StatCard │  │ StatCard │  │ StatCard │          │    │
│  │  • DBs   │  └──────────┘  └──────────┘  └──────────┘          │    │
│  │  • Store │                                                       │    │
│  │  • Auth  │  ┌────────────────────────────────────────┐          │    │
│  │  • IAM   │  │ Data Table / Content                    │          │    │
│  │  • ...   │  │                                          │          │    │
│  │          │  └────────────────────────────────────────┘          │    │
│  └──────────┴──────────────────────────────────────────────────────┘    │
│                                                                          │
│  Components:                                                             │
│  • Shell     → src/components/shell.tsx    (wraps Sidebar + Header)     │
│  • Sidebar   → src/components/sidebar.tsx  (nav links, filtered by mode)│
│  • Header    → src/components/header.tsx   (top bar, user dropdown)     │
│  • StatCard  → src/components/stat-card.tsx (KPI display)               │
│  • StatusBadge → src/components/status-badge.tsx (running/error/etc.)   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Component Naming Convention

| Pattern | Example | Purpose |
|---------|---------|---------|
| `page.tsx` | `app/databases/page.tsx` | Route page (Next.js App Router) |
| `layout.tsx` | `app/layout.tsx` | Shared layout wrapper |
| `loading.tsx` | `app/loading.tsx` | Loading state |
| Kebab-case components | `stat-card.tsx` | Reusable UI component |
| Hooks with `use-` prefix | `hooks/use-api.ts` | Custom React hooks |
| Lib files | `lib/api.ts` | Utilities, API clients |

---

## 5. Data Fetching & API Communication

```
┌─────────────────────────────────────────────────────────────────────────┐
│             DATA FETCHING ARCHITECTURE                                    │
│                                                                          │
│  Page Component                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  const { data, loading, error } = useApi(                          │ │
│  │    () => api.apps.list(), []                                       │ │
│  │  );                                                                │ │
│  └────────────────────┬───────────────────────────────────────────────┘ │
│                       │                                                  │
│                       ▼                                                  │
│  useApi Hook (hooks/use-api.ts)                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  • Takes a fetcher function and dependency array                   │ │
│  │  • Manages loading, data, error state                             │ │
│  │  • Calls fetcher on mount and when deps change                    │ │
│  │  • Returns { data: T | null, loading: boolean, error: Error | null }│ │
│  └────────────────────┬───────────────────────────────────────────────┘ │
│                       │                                                  │
│                       ▼                                                  │
│  API Client (lib/api.ts)                                                 │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  fetchWithAuth(path, options)                                      │ │
│  │  ┌────────────────────────────────────────────────────────────┐   │ │
│  │  │  1. Read access_token from localStorage                    │   │ │
│  │  │  2. Set Authorization: Bearer <token>                      │   │ │
│  │  │  3. Fetch NEXT_PUBLIC_API_URL + path                      │   │ │
│  │  │  4. If 401 → try refresh_token → retry once               │   │ │
│  │  │  5. If still 401 → throw UnauthorizedError → redirect      │   │ │
│  │  │  6. If 4xx/5xx → throw ApiError                           │   │ │
│  │  │  7. Return response                                        │   │ │
│  │  └────────────────────────────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                       │                                                  │
│                       ▼                                                  │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  zenith-api (Go backend at NEXT_PUBLIC_API_URL)                    │ │
│  │  Default: http://localhost:8080                                    │ │
│  │  Staging: https://api.stage.freezenith.com                        │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  FOR MUTATIONS (POST/PUT/DELETE):                                        │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  const { mutate, loading } = useMutation(                          │ │
│  │    (data) => api.apps.create(data)                                │ │
│  │  );                                                                │ │
│  │  // Call: mutate({ name: "my-app", ... })                         │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

### API Client Organization (Web App)

The API client in `apps/web/src/lib/api.ts` is organized by domain:

| Domain | Methods | Example |
|--------|---------|---------|
| `auth` | login, register, logout, refresh, oauth | `api.auth.login(email, password)` |
| `projects` | list, get, create, update, delete | `api.projects.list()` |
| `apps` | list, get, create, deploy, logs | `api.apps.list(projectId)` |
| `appsDeploy` | list, get, create, builds, logs | `api.appsDeploy.list()` |
| `databases` | list, get, create, delete | `api.databases.list()` |
| `storage` | list, create, delete | `api.storage.list()` |
| `billing` | checkout, portal, invoices | `api.billing.checkout(plan)` |
| `userPlan` | get, usage | `api.userPlan.get()` |

---

## 6. Authentication Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│             AUTHENTICATION FLOW                                          │
│                                                                          │
│  LOGIN:                                                                  │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  1. User enters email + password on /login page                    │ │
│  │  2. POST /api/v1/auth/login { email, password }                   │ │
│  │  3. API returns { access_token, refresh_token, user }             │ │
│  │  4. Frontend stores tokens in localStorage:                       │ │
│  │     localStorage.setItem("access_token", token)                   │ │
│  │     localStorage.setItem("refresh_token", refreshToken)           │ │
│  │  5. Redirect to / (dashboard)                                     │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  AUTHENTICATED REQUEST:                                                  │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  1. fetchWithAuth reads access_token from localStorage            │ │
│  │  2. Sets header: Authorization: Bearer <access_token>             │ │
│  │  3. Makes request to API                                          │ │
│  │  4. If 401 response:                                              │ │
│  │     a. Try POST /api/v1/auth/refresh with refresh_token           │ │
│  │     b. If refresh succeeds → store new tokens → retry request     │ │
│  │     c. If refresh fails → clear localStorage → redirect to /login │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  LOGOUT:                                                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  1. Clear localStorage (access_token, refresh_token)              │ │
│  │  2. Redirect to /login                                             │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  TOKEN STORAGE:                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  localStorage:                                                     │ │
│  │    access_token   → JWT, short-lived (~15 min)                    │ │
│  │    refresh_token  → Longer-lived (~7 days)                        │ │
│  │                                                                    │ │
│  │  NOTE: In SaaS mode (production), JWT is verified by APISIX       │ │
│  │  at the gateway level. The backend trusts APISIX headers.         │ │
│  │  In standalone mode, the backend verifies JWT itself.             │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Demo Mode

```
┌─────────────────────────────────────────────────────────────────────────┐
│             DEMO MODE ARCHITECTURE                                       │
│                                                                          │
│  When NEXT_PUBLIC_DEMO_MODE=true:                                       │
│                                                                          │
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────────────────┐   │
│  │ Page        │ →  │ get-api.ts   │ →  │ demo-api.ts              │   │
│  │ Component   │    │ (switcher)   │    │ (returns mock data)      │   │
│  └─────────────┘    └──────────────┘    └──────────┬───────────────┘   │
│                                                     │                    │
│                                                     ▼                    │
│                                          ┌──────────────────────────┐   │
│                                          │ mock-data.ts              │   │
│                                          │                           │   │
│                                          │ • Mock projects           │   │
│                                          │ • Mock apps (3 running)   │   │
│                                          │ • Mock databases          │   │
│                                          │ • Mock domains            │   │
│                                          │ • Mock auth/users         │   │
│                                          │ • Mock billing            │   │
│                                          │ • Mock gateway routes     │   │
│                                          │                           │   │
│                                          │ Demo user: demo@zenith.dev│   │
│                                          └──────────────────────────┘   │
│                                                                          │
│  When NEXT_PUBLIC_DEMO_MODE=false (or unset):                           │
│                                                                          │
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────────────────┐   │
│  │ Page        │ →  │ get-api.ts   │ →  │ api.ts                    │   │
│  │ Component   │    │ (switcher)   │    │ (real HTTP to backend)   │   │
│  └─────────────┘    └──────────────┘    └──────────────────────────┘   │
│                                                                          │
│  USE CASES:                                                              │
│  • Frontend development without running the API backend                 │
│  • Public demo environment (no real data exposed)                       │
│  • CI builds publish demo variants (zenith-web-demo, zenith-mc-demo)   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Styling & Design System

```
┌─────────────────────────────────────────────────────────────────────────┐
│             DESIGN SYSTEM                                                │
│                                                                          │
│  Framework: Tailwind CSS 3.4.17                                          │
│  Mode: Dark only (no light theme)                                       │
│  Icons: Lucide React                                                    │
│  Fonts: Inter (sans), JetBrains Mono (monospace)                        │
│  Component Library: None — custom HTML + Tailwind                       │
│                                                                          │
│  COLOR PALETTE:                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  SURFACES (dark backgrounds)                                       │ │
│  │  ┌─────────┬────────────┬───────────────────────────────────────┐ │ │
│  │  │ Token   │ Hex        │ Usage                                 │ │ │
│  │  ├─────────┼────────────┼───────────────────────────────────────┤ │ │
│  │  │ DEFAULT │ #0a0a0a    │ Page background                      │ │ │
│  │  │ 50      │ #111111    │ Card background                      │ │ │
│  │  │ 100     │ #1a1a1a    │ Elevated cards                       │ │ │
│  │  │ 200     │ #1e1e1e    │ Hover state                          │ │ │
│  │  │ 300     │ #222222    │ Active state                          │ │ │
│  │  └─────────┴────────────┴───────────────────────────────────────┘ │ │
│  │                                                                    │ │
│  │  BORDERS                                                           │ │
│  │  ┌─────────┬────────────┬───────────────────────────────────────┐ │ │
│  │  │ DEFAULT │ #1e1e1e    │ Default border                        │ │ │
│  │  │ hover   │ #2e2e2e    │ Hover borders                        │ │ │
│  │  │ active  │ #3e3e3e    │ Active/focus borders                 │ │ │
│  │  └─────────┴────────────┴───────────────────────────────────────┘ │ │
│  │                                                                    │ │
│  │  ACCENT COLORS                                                     │ │
│  │  ┌──────────────────┬────────────┬──────────────────────────────┐ │ │
│  │  │ Web + Landing    │ Emerald    │ #10b981 (buttons, links)     │ │ │
│  │  │ Mission Control  │ Blue       │ #3b82f6 (distinct from user) │ │ │
│  │  └──────────────────┴────────────┴──────────────────────────────┘ │ │
│  │                                                                    │ │
│  │  STATUS COLORS                                                     │ │
│  │  ┌──────────────────┬────────────┬──────────────────────────────┐ │ │
│  │  │ Running/Active   │ Emerald    │ Green dot / badge            │ │ │
│  │  │ Deploying/Build  │ Amber      │ Yellow spinner               │ │ │
│  │  │ Error/Failed     │ Red        │ Red badge / alert            │ │ │
│  │  │ Stopped/Inactive │ Neutral    │ Gray text                    │ │ │
│  │  └──────────────────┴────────────┴──────────────────────────────┘ │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 9. App: Landing Page (`apps/landing/`)

### Routes

| Path | Page | Type |
|------|------|------|
| `/` | Hero + Features + Pricing + Architecture + CTA | Static/SSG |
| `/pricing` | Detailed pricing tiers (Cloud vs Self-Hosted) | Static/SSG |
| `/docs` | Documentation links | Static/SSG |

### Key Components

| Component | Purpose |
|-----------|---------|
| `hero-section.tsx` | Word-by-word animated headline with terminal demo |
| `architecture-diagram.tsx` | Visual tech stack (Go, K8s, Next.js, etc.) |
| `cloud-pricing.tsx` | Cloud tier pricing cards |
| `cost-calculator.tsx` | Self-hosted cost calculator |
| `deploy-options.tsx` | Cloud vs Self-Hosted comparison |
| `animated-terminal.tsx` | Glowing terminal with animated CLI commands |
| `feature-card.tsx` | Feature showcase cards |

### Animations

Uses **Framer Motion** (only app with animations):
- Word-by-word headline reveal
- Aurora orb background effects
- Terminal glow border
- Scroll-triggered fade-in effects

---

## 10. App: Web Dashboard (`apps/web/`)

### Routes (25+ pages)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  /                    Overview — stats, plan usage, service health       │
│  /apps                Legacy apps list (v1 API)                         │
│  /apps/[name]         App detail page                                   │
│  /deploy              Deploy engine apps                                │
│  /deploy/[id]         Deploy detail with build logs                     │
│  /databases           Database list                                     │
│  /databases/[name]    Database detail                                   │
│  /storage             S3 buckets                                        │
│  /registry            Container registry (Pro+)                         │
│  /networking          Custom domains                                    │
│  /gateway             APISIX routes & plugins (SaaS only)              │
│  /auth                Keycloak realms, users, clients                   │
│  /iam                 Team members, API keys, roles                     │
│  /monitoring          Grafana dashboards                                │
│  /planets             Infrastructure nodes (SaaS only)                  │
│  /billing             Stripe checkout & invoices                        │
│  /settings            User settings                                     │
│  /login               Login page (public)                               │
│  /register            Registration page (public)                        │
│  /docs                Documentation (public)                            │
└─────────────────────────────────────────────────────────────────────────┘
```

### Conditional Navigation (SaaS vs Standalone)

The sidebar shows different items based on `NEXT_PUBLIC_ZENITH_MODE`:

| Item | SaaS | Standalone |
|------|------|------------|
| Overview | Yes | Yes |
| Apps / Deploy | Yes | Yes |
| Databases | Yes | Yes |
| Storage | Yes | Yes |
| Auth / IAM | Yes | Yes |
| Gateway | Yes | No |
| Planets | Yes | No |
| Registry | Yes | No |
| Billing | Yes | No |
| Monitoring | Yes | Yes |

### Real-Time Features

- **WebSocket** for deploy updates: `useWebSocket(projectId)` — streams build logs
- **useDeployLogs(appId)** — terminal-like build output with auto-scroll
- **useDeployEvents(appId)** — deployment status transitions (queued → building → deploying → running)

---

## 11. App: Mission Control (`apps/mission-control/`)

### Routes

```
┌─────────────────────────────────────────────────────────────────────────┐
│  /                    Dashboard — clusters, MRR, updates, activity       │
│  /clusters            Cluster list (K8s version, nodes, usage)          │
│  /clusters/[name]     Cluster detail (CPU, RAM, pods)                   │
│  /customers           Customer list                                     │
│  /customers/[id]      Customer detail (usage, plan, cluster)            │
│  /customers/new       Create new customer                               │
│  /plans               Billing plans (create/edit)                       │
│  /tenants             Shared cluster tenants                            │
│  /modules             Platform modules (APISIX, cert-manager, etc.)     │
│  /updates             Platform version updates                          │
│  /infrastructure      Hetzner nodes, volumes, LBs                       │
│  /audit               Audit log (who, what, when)                       │
│  /state               Platform state export                             │
│  /settings            Platform-wide settings                            │
│  /login               Admin login                                       │
└─────────────────────────────────────────────────────────────────────────┘
```

### API Client Differences

Mission Control uses a **class-based** API client (vs the web app's function-based):

```
Web app:     api.apps.list()           ← Object with grouped functions
MC app:      client.clusters.list()    ← ApiClient class instance
```

Both use the same auth pattern (Bearer JWT, auto-refresh on 401).

### Admin-Only Features

| Feature | What It Does |
|---------|-------------|
| Cluster management | View/create/delete K8s clusters |
| Customer management | Create/suspend/delete customers, view usage |
| Plan management | Create/edit billing plans (Free, Pro, Team, Enterprise) |
| Module management | Install/update/uninstall platform components |
| Audit log | Full trail of operator actions with filters |
| Platform updates | Check and apply Zenith version updates |
| Infrastructure view | Hetzner nodes, volumes, load balancers |
| State export | Export entire platform state as YAML/JSON |

---

## 12. Docker Build & Deployment

```
┌─────────────────────────────────────────────────────────────────────────┐
│             DOCKER BUILD PROCESS (all 3 apps share this pattern)         │
│                                                                          │
│  Stage 1: BASE                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  FROM node:20-alpine                                               │ │
│  │  Enable corepack (pnpm)                                            │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Stage 2: DEPS (install node_modules)                                   │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  COPY package.json pnpm-lock.yaml                                  │ │
│  │  pnpm install --frozen-lockfile                                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Stage 3: BUILDER (build Next.js)                                       │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  COPY src/ public/ next.config.ts tailwind.config.ts               │ │
│  │                                                                    │ │
│  │  ARG NEXT_PUBLIC_API_URL      ← Build-time URL injection          │ │
│  │  ARG NEXT_PUBLIC_APP_URL      ← (varies per app)                  │ │
│  │  ARG NEXT_PUBLIC_DEMO_MODE    ← demo variant builds               │ │
│  │                                                                    │ │
│  │  pnpm build                   ← Next.js build (output: standalone) │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                          │                                               │
│                          ▼                                               │
│  Stage 4: RUNNER (production image)                                     │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  FROM node:20-alpine                                               │ │
│  │  USER nextjs:1001 (non-root)                                       │ │
│  │  COPY --from=builder .next/standalone/                             │ │
│  │  COPY --from=builder .next/static/                                 │ │
│  │  ENV NODE_ENV=production                                           │ │
│  │  CMD ["node", "server.js"]                                         │ │
│  │                                                                    │ │
│  │  Final image size: ~150-200MB                                      │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  CI builds BOTH normal and demo variants for web + mission-control:     │
│  • zenith-web:0.1.0        (real API)                                   │
│  • zenith-web-demo:0.1.0   (NEXT_PUBLIC_DEMO_MODE=true)                │
│  • zenith-mc:0.1.0         (real API)                                   │
│  • zenith-mc-demo:0.1.0    (NEXT_PUBLIC_DEMO_MODE=true)                │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 13. Troubleshooting

### "CORS policy" error in browser console

```bash
# The API needs to allow the frontend's origin
# For local dev, set in API environment:
CORS_ORIGINS=http://localhost:3000,http://localhost:3100,http://localhost:3200

# For staging, APISIX handles CORS (cors plugin on routes)
```

### API calls return 401 but user is logged in

```bash
# Check if access_token is expired
# Open browser DevTools → Application → Local Storage
# Copy access_token, paste at jwt.io to see expiry

# If expired and refresh fails, clear localStorage and re-login:
# Browser console:
localStorage.clear()
window.location.href = '/login'
```

### Page shows blank white screen

```bash
# Check browser console for errors
# Common cause: NEXT_PUBLIC_API_URL not set at build time
# Next.js inlines env vars at BUILD time, not runtime

# Fix: rebuild the Docker image with correct build args
docker build --build-arg NEXT_PUBLIC_API_URL=http://localhost:8080 .
```

### Sidebar shows wrong items

```bash
# Check NEXT_PUBLIC_ZENITH_MODE
# "saas" → all items visible
# "standalone" → SaaS-only items hidden (gateway, planets, registry, billing)

# Set in .env.local:
NEXT_PUBLIC_ZENITH_MODE=saas
```
