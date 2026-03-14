# 04 — Frontend Complete Guide (Next.js 15)

> **Read time:** 90 minutes
> **Prerequisite:** [03 — Backend Complete Guide](./03-backend-guide.md)
> **Next:** [05 — Infrastructure Guide](./05-infrastructure.md)

---

## Overview

Three Next.js 15 apps, all using React 19, TypeScript, Tailwind CSS, and Lucide icons.

| App | Port | Path | Pages | Purpose |
|-----|------|------|-------|---------|
| **Web** | 3000 | `apps/web/` | 40 | Customer dashboard |
| **Mission Control** | 3100 | `apps/mission-control/` | 39 | Admin/operator panel |
| **Landing** | 3200 | `apps/landing/` | 3 | Marketing site |

Shared package: `@zenith/ui` (workspace) — shared components between apps.

---

## Tech Stack

| Tool | Version | Purpose |
|------|---------|---------|
| Next.js | 15.1 | React framework (App Router) |
| React | 19 | UI library |
| TypeScript | 5.x | Type safety |
| Tailwind CSS | 3.4 | Styling (dark mode, class-based) |
| Lucide React | 0.469 | Icons |
| Framer Motion | 11.15 | Animations (landing only) |
| Recharts | 2.15 | Charts (mission-control only) |
| QRCode.react | 4.2 | MFA QR codes (web only) |

**Build output:** `standalone` mode for Docker deployment.

---

## App 1: Web Dashboard (`apps/web/`)

### Directory Structure
```
apps/web/src/
├── app/                        ← 40 pages (App Router)
│   ├── page.tsx                ← Dashboard (overview, stats, health)
│   ├── login/page.tsx          ← Login form
│   ├── register/page.tsx       ← Sign-up with UTM tracking
│   ├── verify-email/page.tsx   ← Email verification
│   ├── onboarding/page.tsx     ← First-time wizard
│   ├── apps/page.tsx           ← App list + Deploy Wizard
│   ├── apps/[name]/page.tsx    ← App detail (status, logs, env, domain)
│   ├── databases/page.tsx      ← Database list + create
│   ├── databases/[name]/       ← DB detail (connection, backup, explorer)
│   ├── storage/page.tsx        ← Storage bucket list
│   ├── storage/[id]/page.tsx   ← File browser (upload, download, folders)
│   ├── gateway/page.tsx        ← API Gateway (routes, domains, analytics)
│   ├── auth/page.tsx           ← Auth pools list
│   ├── auth/[poolId]/page.tsx  ← Auth pool detail (users, roles, MFA)
│   ├── monitoring/page.tsx     ← Metrics overview
│   ├── monitoring/[appId]/     ← App-specific monitoring
│   ├── registry/page.tsx       ← Container registry (Harbor)
│   ├── networking/page.tsx     ← Custom domains, DNS, TLS
│   ├── settings/page.tsx       ← Account, plan, billing
│   ├── billing/page.tsx        ← Upgrade/downgrade, payment
│   ├── support/page.tsx        ← Support tickets list
│   ├── support/[id]/page.tsx   ← Ticket detail + messages
│   ├── iam/page.tsx            ← Team RBAC
│   ├── alerts/page.tsx         ← Alert rules
│   ├── logs/page.tsx           ← Log viewer
│   ├── firewall/page.tsx       ← WAF rules
│   ├── compliance/page.tsx     ← SOC2/GDPR (Business+)
│   ├── audit/page.tsx          ← Audit log (Team+)
│   ├── ssh-sessions/page.tsx   ← Pod exec sessions
│   ├── queues/page.tsx         ← Message queues
│   ├── marketplace/page.tsx    ← Add-ons
│   ├── docs/page.tsx           ← API documentation
│   └── invite/page.tsx         ← Team invitations
│
├── components/                 ← 20 reusable components
│   ├── shell.tsx               ← Main layout (sidebar + header + content)
│   ├── sidebar.tsx             ← Left navigation
│   ├── header.tsx              ← Top bar (user menu, notifications)
│   ├── stat-card.tsx           ← KPI card (label, value, trend)
│   ├── status-badge.tsx        ← Status indicator (running/failed/deploying)
│   ├── modal.tsx               ← Reusable dialog
│   ├── toast.tsx               ← Toast notifications
│   ├── loading-skeleton.tsx    ← Skeleton loaders
│   ├── error-state.tsx         ← Error with retry
│   ├── empty-state.tsx         ← Empty data with CTA
│   ├── deploy-wizard.tsx       ← Multi-step deploy form
│   ├── database-explorer.tsx   ← DB table browser
│   ├── build-log-viewer.tsx    ← Streaming build logs (SSE)
│   ├── onboarding-wizard.tsx   ← First-time setup
│   ├── demo-banner.tsx         ← Demo mode notice
│   ├── upgrade-nudge.tsx       ← Plan upgrade upsell
│   ├── mfa-banner.tsx          ← MFA setup encouragement
│   └── referral-card.tsx       ← Share referral link
│
├── hooks/                      ← 7 custom hooks
│   ├── use-api.ts              ← Generic data fetching (loading/error/refetch)
│   ├── use-mutation.ts         ← API mutations (POST/PUT/DELETE)
│   ├── use-auth.ts             ← Auth state (login/register/MFA/logout)
│   ├── use-project.ts          ← Current project ID from context
│   ├── use-click-outside.ts    ← Close modal on outside click
│   ├── use-deploy-logs.ts      ← SSE deploy log streaming
│   └── use-websocket.ts        ← WebSocket for real-time events
│
└── lib/                        ← Utilities
    ├── api.ts                  ← API client (2314 lines! — master reference)
    ├── demo-api.ts             ← Mock API for demo mode
    ├── get-api.ts              ← Returns real or demo API based on mode
    ├── mock-data.ts            ← Static mock data
    └── runtime-env.ts          ← Runtime environment variables
```

### Page Pattern (Every Page Follows This)

```tsx
"use client";

import { Shell } from "@/components/shell";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { Loading, Error, Empty } from "@/components/...";

export default function MyFeaturePage() {
  const api = getApi();
  const { data, loading, error, refetch } = useApi(
    () => api.myFeature.list(),
    []
  );

  if (loading) return <Shell><LoadingSkeleton /></Shell>;
  if (error) return <Shell><ErrorState onRetry={refetch} /></Shell>;
  if (!data?.length) return <Shell><EmptyState /></Shell>;

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-white">My Feature</h1>
          <button className="px-4 py-2 bg-accent-600 ...">Create</button>
        </div>
        {/* Content */}
      </div>
    </Shell>
  );
}
```

### API Client (`api.ts` — 2314 Lines)

This is the single most important file in the frontend. Every API call goes through it.

**Structure:**
```typescript
// api.ts
class Api {
  auth = {
    login(email, password): Promise<LoginResponse>,
    register(email, password, name): Promise<RegisterResponse>,
    verifyEmail(token): Promise<LoginResponse>,
    logout(): Promise<void>,
    // ...8 methods
  };

  projects = {
    list(): Promise<{ items: Project[], total: number }>,
    get(id): Promise<Project>,
    create(input): Promise<Project>,
    // ...5 methods
  };

  appsDeploy = {
    list(): Promise<{ items: DeployApp[] }>,
    get(id): Promise<DeployApp>,
    create(input): Promise<DeployApp>,
    // ...6 methods
  };

  standaloneDatabases = {
    list(): Promise<AppDatabase[]>,
    create(input): Promise<AppDatabase>,
    startExplorer(id): Promise<{ url: string }>,
    // ...7 methods
  };

  storageBuckets = {
    list(): Promise<StorageBucketV2[]>,
    listObjects(bucketId, prefix): Promise<{ objects: StorageObject[] }>,
    uploadObject(bucketId, key, file, onProgress): Promise<void>,
    // ...12 methods
  };

  gateways = {
    list(): Promise<{ items: Gateway[] }>,
    listDomains(gwId): Promise<GatewayCustomDomain[]>,
    addDomain(gwId, domain): Promise<GatewayCustomDomain>,
    getAnalytics(gwId): Promise<GatewayAnalyticsOverview>,
    // ...18 methods
  };

  // ... 20+ more namespaces
}
```

### Hooks Deep Dive

**`useApi<T>`** — The workhorse hook
```typescript
const { data, loading, error, refetch } = useApi<App[]>(
  () => api.apps.list(projectId),
  [projectId]  // re-fetch when projectId changes
);
```

**`useMutation<TData, TVars>`** — For create/update/delete
```typescript
const { mutate, loading, error } = useMutation(
  (name: string) => api.apps.create({ name, image: "nginx" })
);

const handleCreate = async () => {
  await mutate("my-app");
  toast("success", "App created!");
  refetch();  // from useApi
};
```

**`useAuth`** — Authentication state
```typescript
const { user, isAuthenticated, login, logout, mfaLogin } = useAuth();
```

**`useDeployLogs`** — SSE streaming
```typescript
const { entries, streaming } = useDeployLogs(appId, deploymentId);
// entries: LogEntry[] (auto-appends as they stream in)
```

### Demo Mode

When `NEXT_PUBLIC_DEMO_MODE=true`:
- `getApi()` returns demo API instead of real API
- All methods return realistic mock data
- No real backend needed
- Used for: local development, CI smoke tests, product demos

```typescript
// lib/get-api.ts
export function getApi() {
  return isDemoMode() ? demoApi : realApi;
}
```

### Styling Conventions

```
Colors:
  accent: emerald (#10b981)      ← Primary brand color
  surface: dark grays (0a0a0a)   ← Background
  border: #262626 (hover: #404040)

Typography:
  Font: Inter (body), JetBrains Mono (code)
  Headings: text-white, font-bold
  Body: text-zinc-400
  Links: text-accent-400

Layout:
  Shell → sidebar (left) + header (top) + content (center)
  Content: max-w-7xl mx-auto px-4 py-6
  Cards: bg-surface border border-border rounded-xl p-6
  Tables: overflow-x-auto, border-b border-border
  Grids: grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4
```

---

## App 2: Mission Control (`apps/mission-control/`)

### Key Differences from Web

| Aspect | Web | Mission Control |
|--------|-----|-----------------|
| Audience | Customers | Operators (Zenith team) |
| Auth | User JWT | Admin JWT |
| Charts | None | Recharts (line, bar, pie) |
| API | Customer endpoints | Admin endpoints |
| Pages | 40 (customer features) | 39 (platform management) |

### Notable Pages

| Page | What It Shows |
|------|--------------|
| `/` (Command Center) | MRR, active customers, churn rate, service health grid |
| `/customers` | Customer table with plan, status, cluster info |
| `/clusters` | K8s cluster status, nodes, capacity |
| `/services` | All services (Traefik, CNPG, APISIX, etc.) with health |
| `/security` | Security posture score, MFA adoption, vulnerabilities |
| `/logs` | Loki log query with label filtering |
| `/traces` | Tempo trace search |
| `/backups` | Velero + CNPG backup status |
| `/gitops` | ArgoCD app sync status |
| `/analytics` | Revenue trends, cohort analysis, growth metrics |
| `/crm` | Sales pipeline, customer health scores |

### `useApiWithFallback` Hook (MC-specific)

```typescript
// Falls back to demo data if API fails (graceful degradation)
const { data, loading, isDemo } = useApiWithFallback(
  () => api.analytics.revenue(),
  demoRevenueData,
  undefined,
  []
);
```

---

## App 3: Landing Page (`apps/landing/`)

### Pages

| Path | What |
|------|------|
| `/` | Hero + features + deploy options + architecture diagram |
| `/pricing` | Tier comparison + cost calculator + FAQ |
| `/docs` | API documentation + SDK links |

### Key Components

- `animated-terminal.tsx` — Live terminal animation (Framer Motion)
- `pricing-card.tsx` — Tier cards with feature comparison
- `cost-calculator.tsx` — Interactive: select CPU/RAM/storage → see monthly cost
- `architecture-diagram.tsx` — SVG system architecture

---

## How to Add a New Page

### Step 1: Create the Page File

```bash
# Create new page
touch apps/web/src/app/my-feature/page.tsx
```

### Step 2: Write the Page

```tsx
"use client";

import { Shell } from "@/components/shell";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { Plus } from "lucide-react";

export default function MyFeaturePage() {
  const api = getApi();
  const { data, loading, error, refetch } = useApi(
    () => api.myFeature.list(),
    []
  );

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-white">My Feature</h1>
          <button className="flex items-center gap-2 px-4 py-2 bg-accent-600 hover:bg-accent-700 text-white rounded-lg text-sm font-medium transition-colors">
            <Plus className="h-4 w-4" />
            Create New
          </button>
        </div>

        {loading && <div className="text-zinc-400">Loading...</div>}

        {data && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data.map((item) => (
              <div key={item.id} className="bg-surface border border-border rounded-xl p-6">
                <h3 className="text-white font-medium">{item.name}</h3>
                <p className="text-zinc-400 text-sm mt-1">{item.description}</p>
              </div>
            ))}
          </div>
        )}
      </div>
    </Shell>
  );
}
```

### Step 3: Add API Methods

```typescript
// lib/api.ts — add to Api class
myFeature = {
  list: () => this.get<MyFeature[]>("/api/v1/my-feature"),
  create: (input: CreateMyFeatureInput) => this.post<MyFeature>("/api/v1/my-feature", input),
  delete: (id: string) => this.del(`/api/v1/my-feature/${id}`),
};
```

### Step 4: Add Demo Data

```typescript
// lib/demo-api.ts
myFeature: {
  list: async () => [
    { id: "1", name: "Demo Item", description: "Demo description" },
  ],
  create: async (input) => ({ id: "new-1", ...input }),
  delete: async () => {},
},
```

### Step 5: Test

```bash
cd apps/web && pnpm dev  # Visit http://localhost:3000/my-feature
```

---

## Running Locally

```bash
# Install dependencies
pnpm install

# Run all three apps
pnpm dev

# Or run individually
cd apps/web && pnpm dev              # port 3000
cd apps/mission-control && pnpm dev  # port 3100
cd apps/landing && pnpm dev          # port 3200

# Lint
cd apps/web && npx next lint --quiet

# Test
cd apps/web && pnpm test
```

---

**Next → [05 — Infrastructure Guide](./05-infrastructure.md)**
