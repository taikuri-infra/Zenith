# Zenith - Complete Frontend Design

> Every page. Every component. Every state. Every interaction.

**Tech Stack:** Next.js 15, TypeScript, Tailwind CSS, shadcn/ui, Lucide icons
**Theme:** Dark (like Supabase/Vercel)
**Accent:** Emerald green (#10b981)
**Font:** Inter (body), JetBrains Mono (code/connection strings)
**Domain:** freezenith.com

---

## Layout Structure

```
┌─────────────────────────────────────────────────────────────┐
│ ┌──────┐  Zenith    Project: my-startup ▼    [?] [bell] [avatar] │
│ │ Logo │                                                         │
├─┴──────┴────────────────────────────────────────────────────┤
│ ┌──────────┐ ┌────────────────────────────────────────────┐ │
│ │          │ │                                            │ │
│ │ SIDEBAR  │ │              MAIN CONTENT                  │ │
│ │          │ │                                            │ │
│ │ OVERVIEW │ │                                            │ │
│ │ Overview │ │                                            │ │
│ │          │ │                                            │ │
│ │ COMPUTE  │ │                                            │ │
│ │ Apps     │ │                                            │ │
│ │ Databases│ │                                            │ │
│ │ Storage  │ │                                            │ │
│ │          │ │                                            │ │
│ │ NETWORK  │ │                                            │ │
│ │ Domains  │ │                                            │ │
│ │ Gateway  │ │                                            │ │
│ │          │ │                                            │ │
│ │ SECURITY │ │                                            │ │
│ │ Auth     │ │                                            │ │
│ │ IAM      │ │                                            │ │
│ │          │ │                                            │ │
│ │ OBSERVE  │ │                                            │ │
│ │ Monitor  │ │                                            │ │
│ │ Registry │ │                                            │ │
│ │          │ │                                            │ │
│ │ INFRA    │ │                                            │ │
│ │ Planets  │ │                                            │ │
│ │          │ │                                            │ │
│ │──────────│ │                                            │ │
│ │ Settings │ │                                            │ │
│ │ Billing  │ │                                            │ │
│ │ Docs     │ │                                            │ │
│ └──────────┘ └────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

**Sidebar Groups:**
- **OVERVIEW:** Project dashboard and quick stats
- **COMPUTE:** Apps (kubectl-style full-width table), Databases (AWS RDS-style table), Storage (AWS S3-style table)
- **NETWORKING:** Domains (renamed from Networking), Gateway (Kong management)
- **SECURITY:** Auth (Keycloak-style: realms, users, clients, identity providers), IAM (platform access: API keys, team, roles)
- **OBSERVABILITY:** Monitoring (Grafana dashboards + Prometheus targets + Loki logs), Registry (ECR-style: repos, tags, vulnerability scanning, pull commands)
- **INFRASTRUCTURE:** Planets (node scaling)

---

## Page-by-Page Design

---

### 0. Landing Page (freezenith.com) - PUBLIC

```
┌─────────────────────────────────────────────────────────────┐
│ [Zenith Logo]                    Docs   Pricing   GitHub    │
│                                              [Get Started]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│         The open-source PaaS for Kubernetes.                │
│         Deploy on your own cloud. Pay only for infra.       │
│                                                             │
│    ┌──────────────────────────────────────────────────┐     │
│    │ $ zen install --provider hetzner --token hc_xxx  │     │
│    │                                                  │     │
│    │ ✓ Cluster ready                                  │     │
│    │ ✓ Platform installed                             │     │
│    │ ✓ Dashboard: https://console.yourserver.com      │     │
│    └──────────────────────────────────────────────────┘     │
│                                                             │
│         [Get Started - Free]     [View on GitHub]           │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│    WHY ZENITH?                                              │
│                                                             │
│    ┌──────────┐  ┌──────────┐  ┌──────────┐                │
│    │ 100%     │  │ Your     │  │ 14x      │                │
│    │ Free     │  │ Cloud    │  │ Cheaper   │                │
│    │          │  │          │  │          │                │
│    │ Apache   │  │ Runs on  │  │ Than AWS/ │                │
│    │ 2.0      │  │ YOUR     │  │ Fly.io   │                │
│    │ Forever  │  │ Hetzner  │  │          │                │
│    └──────────┘  └──────────┘  └──────────┘                │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│    WHAT YOU GET                                              │
│                                                             │
│    ✓ Deploy Apps (GitHub / Docker)                          │
│    ✓ Managed PostgreSQL, MySQL, MongoDB, Redis              │
│    ✓ S3-Compatible Storage                                  │
│    ✓ Auto SSL & Custom Domains                              │
│    ✓ Container Registry                                     │
│    ✓ Auth & SSO (Keycloak)                                  │
│    ✓ Monitoring & Logging                                   │
│    ✓ API Gateway                                            │
│    ✓ Firewall, DNS, Load Balancer                           │
│    ✓ Scale with "Add a Planet"                              │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│    COST COMPARISON                                          │
│                                                             │
│    100 microservices + Postgres + Redis:                     │
│                                                             │
│    AWS EKS:     $553/mo  ████████████████████████████       │
│    Fly.io:      $240/mo  █████████████                      │
│    Railway:     $180/mo  ██████████                         │
│    Zenith:       €38/mo  ███                                │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│    ┌──────────────────────────────────────────────────┐     │
│    │   Ready to deploy?                               │     │
│    │                                                  │     │
│    │   $ curl -sSL https://get.freezenith.com | sh    │     │
│    │   $ zen install --provider hetzner               │     │
│    │                                                  │     │
│    │   [Read the Docs]    [Star on GitHub]             │     │
│    └──────────────────────────────────────────────────┘     │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  Zenith is a CNCF project candidate   Apache 2.0 License   │
│  GitHub   Docs   Discord   Twitter                         │
└─────────────────────────────────────────────────────────────┘
```

---

### 1. Auth Pages

#### 1.1 Login

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│                    [Zenith Logo]                             │
│                                                             │
│               Welcome back to Zenith                        │
│                                                             │
│    ┌─────────────────────────────────────────┐              │
│    │  Email                                  │              │
│    │  ┌───────────────────────────────────┐  │              │
│    │  │ you@example.com                   │  │              │
│    │  └───────────────────────────────────┘  │              │
│    │                                         │              │
│    │  Password                               │              │
│    │  ┌───────────────────────────────────┐  │              │
│    │  │ ••••••••••                        │  │              │
│    │  └───────────────────────────────────┘  │              │
│    │                                         │              │
│    │  [        Sign In                    ]  │              │
│    │                                         │              │
│    │  ─────────── or ───────────             │              │
│    │                                         │              │
│    │  [  Continue with GitHub  ]             │              │
│    │  [  Continue with Google  ]             │              │
│    │                                         │              │
│    │  Don't have an account? Sign up         │              │
│    └─────────────────────────────────────────┘              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### 1.2 Register

```
Same as Login but:
  - Title: "Create your Zenith account"
  - Fields: Name, Email, Password, Confirm Password
  - Button: "Create Account"
  - Bottom: "Already have an account? Sign in"
```

---

### 2. Projects List (Dashboard Home)

```
┌─────────────────────────────────────────────────────────────┐
│ [Zenith]                                     [?] [🔔] [👤] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Your Projects                          [+ New Project]     │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  🟢 my-startup              Pro Plan                │   │
│  │     3 apps · 2 databases · fsn1                     │   │
│  │     Created 2 weeks ago            ~€47/mo          │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  🟢 side-project            Starter Plan            │   │
│  │     1 app · 1 database · nbg1                       │   │
│  │     Created 3 days ago             ~€5/mo           │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  🟡 experiment              Starter Plan            │   │
│  │     0 apps · 0 databases · fsn1                     │   │
│  │     Created today                  ~€0/mo           │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                                                     │   │
│  │    +  Create your first project                     │   │
│  │       Deploy apps, databases, and more              │   │
│  │                                                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### 2.1 Create Project Modal

```
┌───────────────────────────────────────┐
│  Create New Project                   │
│                                       │
│  Project Name                         │
│  ┌─────────────────────────────────┐  │
│  │ my-awesome-app                  │  │
│  └─────────────────────────────────┘  │
│  Lowercase, alphanumeric, hyphens     │
│                                       │
│  Region                               │
│  ┌─────────────────────────────────┐  │
│  │ 🇩🇪 Falkenstein (fsn1)     ▼   │  │
│  │ 🇩🇪 Nuremberg (nbg1)          │  │
│  │ 🇫🇮 Helsinki (hel1)           │  │
│  └─────────────────────────────────┘  │
│                                       │
│  [Cancel]              [Create]       │
└───────────────────────────────────────┘
```

---

### 3. Project Dashboard - Overview

```
┌─────────────────────────────────────────────────────────────┐
│ [Zenith]   Project: my-startup ▼                  [👤]      │
├──────────┬──────────────────────────────────────────────────┤
│          │                                                  │
│ Overview │  Overview                                        │
│ ──────── │                                                  │
│ Apps     │  ┌──────────┐ ┌──────────┐ ┌──────────┐         │
│ Database │  │ 🟢 3     │ │ 🟢 2     │ │ 📦 45GB  │         │
│ Storage  │  │ Apps     │ │ Databases│ │ Storage  │         │
│ Network  │  │ running  │ │ running  │ │ used     │         │
│ Auth     │  └──────────┘ └──────────┘ └──────────┘         │
│ Monitor  │                                                  │
│ Registry │  ┌──────────┐ ┌──────────┐ ┌──────────┐         │
│ Planets  │  │ 🌍 3     │ │ 🔒 Auto  │ │ 💰~€47  │         │
│          │  │ Planets  │ │ SSL      │ │ /month   │         │
│ ──────── │  │ active   │ │ enabled  │ │ estimate │         │
│ Settings │  └──────────┘ └──────────┘ └──────────┘         │
│ Billing  │                                                  │
│ Docs     │  Recent Activity                                │
│          │  ┌──────────────────────────────────────────┐    │
│          │  │ 🟢 api deployed (v23)         2 min ago  │    │
│          │  │ 📦 backup completed           1 hour ago │    │
│          │  │ 🟢 worker scaled to 3         3 hours ago│    │
│          │  │ 🗄️ orders-db created          yesterday  │    │
│          │  │ 🌍 Planet cx43 added          2 days ago │    │
│          │  └──────────────────────────────────────────┘    │
│          │                                                  │
│          │  Resource Usage (24h)                            │
│          │  ┌──────────────────────────────────────────┐    │
│          │  │  CPU ████████████░░░░░░░░  58%           │    │
│          │  │  RAM █████████████████░░░  82%           │    │
│          │  │  Disk ██████░░░░░░░░░░░░░  32%           │    │
│          │  └──────────────────────────────────────────┘    │
│          │                                                  │
└──────────┴──────────────────────────────────────────────────┘
```

---

### 4. Apps

> kubectl-style full-width table layout. Shows all apps with status, replicas, domain, and last deploy.

#### 4.1 Apps List

```
│ Apps                                     [+ Deploy App]     │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ 🟢 frontend          app.startup.com     2 replicas │   │
│  │    Next.js · github.com/org/frontend · v23          │   │
│  │    Deployed 2 minutes ago                           │   │
│  ├──────────────────────────────────────────────────────┤   │
│  │ 🟢 api-gateway        api.startup.com     2 replicas│   │
│  │    Go · github.com/org/gateway · v45                │   │
│  │    Deployed 1 hour ago                              │   │
│  ├──────────────────────────────────────────────────────┤   │
│  │ 🟢 user-service       internal            3 replicas│   │
│  │    Python · github.com/org/user-svc · v12           │   │
│  │    Deployed 3 hours ago                             │   │
│  ├──────────────────────────────────────────────────────┤   │
│  │ 🟡 payment-service    internal            1 replica │   │
│  │    Node.js · deploying...                           │   │
│  │    Build in progress (2/5 steps)                    │   │
│  └──────────────────────────────────────────────────────┘   │
```

#### 4.2 Deploy New App

```
│  Deploy a new App                                          │
│                                                             │
│  How do you want to deploy?                                │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌────────────┐  │
│  │  📦 GitHub      │  │  🐳 Docker      │  │  📋 Template│  │
│  │                 │  │                 │  │            │  │
│  │  Connect repo   │  │  Pull image     │  │  WordPress │  │
│  │  Auto-deploy    │  │  from any       │  │  Ghost     │  │
│  │  on push        │  │  registry       │  │  Strapi    │  │
│  │                 │  │                 │  │  n8n       │  │
│  └─────────────────┘  └─────────────────┘  └────────────┘  │
```

#### 4.2a Deploy from GitHub (after clicking GitHub)

```
│  Deploy from GitHub                                        │
│                                                             │
│  Step 1: Select Repository                                 │
│  ┌───────────────────────────────────────────────────┐     │
│  │ 🔍 Search repositories...                         │     │
│  ├───────────────────────────────────────────────────┤     │
│  │ ○ org/frontend          Updated 2 hours ago       │     │
│  │ ● org/user-service      Updated 1 day ago     ✓   │     │
│  │ ○ org/order-service     Updated 3 days ago        │     │
│  │ ○ org/payment-service   Updated 1 week ago        │     │
│  └───────────────────────────────────────────────────┘     │
│                                                             │
│  Step 2: Configure                                         │
│                                                             │
│  App Name                                                  │
│  ┌───────────────────────────────────────┐                 │
│  │ user-service                          │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  Branch                                                    │
│  ┌───────────────────────────────────────┐                 │
│  │ main                              ▼   │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  Dockerfile Path                                           │
│  ┌───────────────────────────────────────┐                 │
│  │ ./Dockerfile                          │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  Port                                                      │
│  ┌───────────────────────────────────────┐                 │
│  │ 8080                                  │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  Expose to internet?                                       │
│  ○ Yes (add a domain)    ● No (internal only)              │
│                                                             │
│  Replicas          Resources                               │
│  [1] [2] [3]       CPU: [0.25] [0.5] [1] [2]             │
│                    RAM: [256M] [512M] [1G] [2G]            │
│                                                             │
│  [Cancel]                              [Deploy]            │
│                                                             │
│  Estimated cost: ~€0/mo (runs on existing Planets)         │
```

#### 4.2b Deploy from Docker Image

```
│  Deploy from Docker Image                                  │
│                                                             │
│  Image URL                                                 │
│  ┌───────────────────────────────────────┐                 │
│  │ registry.example.com/my-app:latest    │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  Registry Authentication (optional)                        │
│  ┌───────────────────────────────────────┐                 │
│  │ Username                              │                 │
│  └───────────────────────────────────────┘                 │
│  ┌───────────────────────────────────────┐                 │
│  │ Password / Token                      │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  (same config fields as GitHub deploy)                     │
```

#### 4.3 App Detail Page

```
│  ← Apps    user-service                    [Redeploy] [⋮]  │
│                                                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │
│  │ 🟢 Running│ │ 3 replicas│ │ v12      │ │ internal │      │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │
│                                                             │
│  [Overview] [Logs] [Env Vars] [Domains] [Scaling] [Deploy] │
│  ─────────────────────────────────────────────────────────  │
```

#### 4.3a App > Logs Tab

```
│  [Overview] [Logs] [Env Vars] [Domains] [Scaling] [Deploy] │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Instance: [All ▼]   Level: [All ▼]   [⏸ Pause] [↓ Bottom]│
│                                                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ 14:23:01  [pod-1]  INFO  Server started on :8080      │ │
│  │ 14:23:02  [pod-1]  INFO  Connected to database        │ │
│  │ 14:23:05  [pod-2]  INFO  Server started on :8080      │ │
│  │ 14:23:05  [pod-2]  INFO  Connected to database        │ │
│  │ 14:23:10  [pod-1]  INFO  GET /health 200 1ms          │ │
│  │ 14:23:11  [pod-1]  INFO  GET /users 200 15ms          │ │
│  │ 14:23:12  [pod-3]  INFO  Server started on :8080      │ │
│  │ 14:23:15  [pod-1]  WARN  Slow query: 250ms            │ │
│  │ 14:23:20  [pod-2]  INFO  POST /users 201 45ms         │ │
│  │ ▌                                                     │ │
│  └───────────────────────────────────────────────────────┘ │
```

#### 4.3b App > Environment Variables Tab

```
│  [Overview] [Logs] [Env Vars] [Domains] [Scaling] [Deploy] │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Environment Variables                    [+ Add Variable]  │
│                                                             │
│  ┌────────────────────┬──────────────────────┬───────────┐ │
│  │ Key                │ Value                │           │ │
│  ├────────────────────┼──────────────────────┼───────────┤ │
│  │ LOG_LEVEL          │ info                 │ [✏️] [🗑️]│ │
│  │ STRIPE_KEY         │ ••••••••••sk_live    │ [✏️] [🗑️]│ │
│  │ DB_URL             │ 🔗 from: users-db    │ [auto]   │ │
│  │ REDIS_URL          │ 🔗 from: cache       │ [auto]   │ │
│  └────────────────────┴──────────────────────┴───────────┘ │
│                                                             │
│  Linked Resources (auto-injected)                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 🗄️ users-db (PostgreSQL)  → DB_HOST, DB_PORT,       │  │
│  │                              DB_USER, DB_PASSWORD,   │  │
│  │                              DB_NAME, DB_URL         │  │
│  │ 🗄️ cache (Redis)          → REDIS_URL               │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  [Save Changes]                                            │
│  ⚠️ Saving will restart the app with new variables.        │
```

#### 4.3c App > Domains Tab

```
│  [Overview] [Logs] [Env Vars] [Domains] [Scaling] [Deploy] │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Domains                                   [+ Add Domain]   │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 🟢 api.startup.com                                   │  │
│  │    SSL: ✅ Valid until 2026-05-15                     │  │
│  │    CNAME: api.startup.com → ingress.your-cluster.com │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ 🟡 api-v2.startup.com                  [Pending DNS] │  │
│  │    SSL: ⏳ Waiting for DNS verification              │  │
│  │    Add CNAME: api-v2.startup.com →                   │  │
│  │    ingress.your-cluster.com              [📋 Copy]   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Internal URL                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ http://user-service:8080               [📋 Copy]     │  │
│  │ Other apps in this project can reach this service    │  │
│  │ using the URL above.                                 │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 4.3d App > Scaling Tab

```
│  [Overview] [Logs] [Env Vars] [Domains] [Scaling] [Deploy] │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Replicas                                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  [-]  ████ 3 replicas  [+]                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Resources per Replica                                     │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  CPU:    [-]  ████ 500m (0.5 vCPU)   [+]            │  │
│  │  Memory: [-]  ████ 512 Mi            [+]            │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Current Usage                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Pod       CPU         Memory        Status          │  │
│  │  pod-1     120m/500m   280Mi/512Mi   🟢 Running      │  │
│  │  pod-2     95m/500m    310Mi/512Mi   🟢 Running      │  │
│  │  pod-3     110m/500m   295Mi/512Mi   🟢 Running      │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  [Save Changes]                                            │
```

#### 4.3e App > Deployments Tab

```
│  [Overview] [Logs] [Env Vars] [Domains] [Scaling] [Deploy] │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Deployment History                                        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 🟢 v12  Current   main@a1b2c3d   2 min ago          │  │
│  │         "fix: handle null user"                      │  │
│  │         Built in 45s · Deployed in 12s               │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ ⚪ v11            main@e4f5g6h   1 hour ago          │  │
│  │         "feat: add user search"         [Rollback]   │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ ⚪ v10            main@i7j8k9l   3 hours ago         │  │
│  │         "refactor: clean up routes"     [Rollback]   │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ 🔴 v9   Failed    main@m0n1o2p   5 hours ago         │  │
│  │         "feat: broken import"                        │  │
│  │         Build failed: ModuleNotFoundError    [Logs]  │  │
│  └──────────────────────────────────────────────────────┘  │
```

---

### 5. Databases

> AWS RDS-style table layout. Shows all databases with engine, storage, usage, and backup status.

#### 5.1 Database List

```
│  Databases                               [+ Add Database]   │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 🟢 users-db             PostgreSQL 16                │  │
│  │    20 GB · 3.2 GB used · CPU: 12%                    │  │
│  │    Last backup: 2 hours ago                          │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ 🟢 orders-db            PostgreSQL 16                │  │
│  │    50 GB · 12.8 GB used · CPU: 25%                   │  │
│  │    Last backup: 2 hours ago                          │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ 🟢 cache                Redis 7                      │  │
│  │    5 GB · 1.1 GB used · Keys: 45,231                 │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ 🟢 products             MongoDB 7                    │  │
│  │    30 GB · 8.5 GB used · Collections: 12             │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 5.2 Create Database

```
│  Create Database                                           │
│                                                             │
│  Engine                                                    │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐  │
│  │ 🐘        │ │ 🐬        │ │ 🍃        │ │ ⚡        │  │
│  │ PostgreSQL│ │ MySQL     │ │ MongoDB   │ │ Redis     │  │
│  │           │ │           │ │           │ │           │  │
│  │ [selected]│ │           │ │           │ │           │  │
│  └───────────┘ └───────────┘ └───────────┘ └───────────┘  │
│                                                             │
│  Name                                                      │
│  ┌───────────────────────────────────────┐                 │
│  │ users-db                              │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  Version                                                   │
│  ┌───────────────────────────────────────┐                 │
│  │ 16 (latest)                       ▼   │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  Storage                                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  [-]  ████████████████░░░░  20 GB  [+]              │  │
│  │  Min: 1 GB    Max: 500 GB                            │  │
│  │  Cost: ~€0.88/mo (Hetzner Volume)                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Backups                                                   │
│  ● Daily (recommended)   ○ Weekly   ○ None                 │
│  Retained for 7 days                                       │
│                                                             │
│  [Cancel]                                    [Create]      │
│                                                             │
│  ℹ️ A Hetzner Volume will be created for this database.    │
│     You only pay Hetzner's volume price (€0.044/GB/mo).    │
```

#### 5.3 Database Detail

```
│  ← Databases    users-db (PostgreSQL 16)        [⋮]       │
│                                                             │
│  [Connection] [Backups] [Metrics] [Logs] [Settings]        │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Connection Information                                    │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                                                      │  │
│  │  Host       users-db-rw                  [📋]       │  │
│  │  Port       5432                         [📋]       │  │
│  │  Database   app                          [📋]       │  │
│  │  Username   app                          [📋]       │  │
│  │  Password   ••••••••••••       [👁] [📋]            │  │
│  │                                                      │  │
│  │  ────────────────────────────────────────────────    │  │
│  │                                                      │  │
│  │  Connection String                                   │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │ postgres://app:****@users-db-rw:5432/app       │  │  │
│  │  │                                        [📋]    │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  Link to App                                         │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │  Select app:  [user-service        ▼]          │  │  │
│  │  │  Env prefix:  [DB                  ]           │  │  │
│  │  │                          [Link Database]       │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │  ℹ️ Linking auto-injects DB_HOST, DB_PORT, DB_USER, │  │
│  │     DB_PASSWORD, DB_NAME, DB_URL into your app.     │  │
│  │                                                      │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 5.3b Database > Backups Tab

```
│  [Connection] [Backups] [Metrics] [Logs] [Settings]        │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Backups                              [Create Backup Now]   │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  🟢 backup-2026-02-15-02:00    1.2 GB     2 hrs ago │  │
│  │                                   [Restore] [Delete] │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │  🟢 backup-2026-02-14-02:00    1.1 GB     26 hrs ago│  │
│  │                                   [Restore] [Delete] │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │  🟢 backup-2026-02-13-02:00    1.1 GB     2 days ago│  │
│  │                                   [Restore] [Delete] │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Schedule: Daily at 02:00 UTC · Retention: 7 days          │
│  Storage: Hetzner Object Storage (S3)                      │
```

---

### 6. Storage

> AWS S3-style table layout. Shows buckets with region, size, object count, and endpoints.

#### 6.1 S3 Buckets

```
│  Storage                                                    │
│                                                             │
│  [S3 Buckets] [Volumes]                                    │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  S3 Buckets                               [+ Create Bucket] │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  📦 uploads                                          │  │
│  │     Region: fsn1 · 23.5 GB used · 1,245 objects     │  │
│  │     Endpoint: https://fsn1.your-objectstorage.com    │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │  📦 backups                                          │  │
│  │     Region: fsn1 · 45.2 GB used · 89 objects        │  │
│  │     Endpoint: https://fsn1.your-objectstorage.com    │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 6.1a S3 Bucket Detail

```
│  ← Storage    uploads                              [⋮]     │
│                                                             │
│  Access Credentials                                        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Endpoint     https://fsn1.your-objectstorage.com    │  │
│  │  Bucket       uploads                        [📋]    │  │
│  │  Access Key   6IDOOB...                      [📋]    │  │
│  │  Secret Key   ••••••••••                [👁] [📋]    │  │
│  │  Region       fsn1                                   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Link to App                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Select app:  [user-service        ▼]                │  │
│  │  Env prefix:  [S3                  ]                 │  │
│  │                          [Link Storage]              │  │
│  │  ℹ️ Creates: S3_ENDPOINT, S3_BUCKET,                 │  │
│  │     S3_ACCESS_KEY, S3_SECRET_KEY                     │  │
│  └──────────────────────────────────────────────────────┘  │
```

---

### 7. Domains (formerly Networking)

> Renamed from "Networking" to "Domains" in sidebar under NETWORKING group.
> API Gateway moved to its own sidebar item (see 7.7 Gateway).
> Other networking features (LB, Firewall, DNS, IPs, VPN, Cloud Connections) accessible via tabs.

```
│  Domains & Networking                                      │
│                                                             │
│  [Domains] [Load Balancers] [Firewalls] [DNS]              │
│  [Floating IPs] [VPN] [Cloud Connections]                  │
│  ─────────────────────────────────────────────────────────  │
```

#### 7.1 Domains

```
│  Domains                                   [+ Add Domain]   │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 🟢 app.startup.com        → frontend     SSL ✅      │  │
│  │ 🟢 api.startup.com        → api-gateway  SSL ✅      │  │
│  │ 🟡 docs.startup.com       → docs-app     SSL ⏳      │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 7.2 Firewalls

```
│  Firewalls                                 [+ Add Rule]     │
│                                                             │
│  Inbound Rules                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ ✅ Allow  TCP  80    0.0.0.0/0    HTTP        [🗑️]  │  │
│  │ ✅ Allow  TCP  443   0.0.0.0/0    HTTPS       [🗑️]  │  │
│  │ ✅ Allow  TCP  22    10.0.0.0/8   SSH (VPN)   [🗑️]  │  │
│  │ ❌ Deny   *    *     0.0.0.0/0    Default     [—]   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ℹ️ Firewall rules are applied via Hetzner Cloud Firewall. │
```

#### 7.3 Load Balancers

```
│  Load Balancers                       [+ Create LB]        │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 🟢 main-lb           LB11 · fsn1                    │  │
│  │    IP: 95.217.xxx.xxx                                │  │
│  │    Targets: frontend, api-gateway                    │  │
│  │    Traffic: 450 GB / 1 TB included                   │  │
│  │    Cost: €5.49/mo (Hetzner LB)                       │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 7.4 DNS

```
│  DNS Zones                              [+ Add Zone]       │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 📡 startup.com           12 records                  │  │
│  │                                                      │  │
│  │    A      @         95.217.xxx.xxx         [✏️][🗑️] │  │
│  │    CNAME  www       startup.com            [✏️][🗑️] │  │
│  │    CNAME  app       ingress.cluster.com    [✏️][🗑️] │  │
│  │    CNAME  api       ingress.cluster.com    [✏️][🗑️] │  │
│  │    MX     @         mail.startup.com       [✏️][🗑️] │  │
│  │    TXT    @         v=spf1 ...             [✏️][🗑️] │  │
│  │                                 [+ Add Record]       │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 7.5 API Gateway

```
│  API Gateway                                               │
│                                                             │
│  Domain: api.startup.com                                   │
│                                                             │
│  Routes                                  [+ Add Route]     │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  /users/*        → user-service:8080        [✏️][🗑️]│  │
│  │  /orders/*       → order-service:8080       [✏️][🗑️]│  │
│  │  /payments/*     → payment-service:8080     [✏️][🗑️]│  │
│  │  /notifications/*→ notification-svc:8080    [✏️][🗑️]│  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Settings                                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Rate Limit:  [100] requests per [minute ▼]          │  │
│  │  CORS Origins: [https://app.startup.com        ]     │  │
│  │  Auth: [JWT ▼]  JWKS URL: [https://auth.../jwks ]   │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 7.6 Cloud Connections (Hybrid Cloud)

```
│  Cloud Connections                     [+ New Connection]   │
│                                                             │
│  Connect your Zenith project to AWS, GCP, Azure or          │
│  on-premises infrastructure via encrypted VPN tunnel.       │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 🟢 aws-production          AWS (eu-west-1)           │  │
│  │    Status: Connected · Latency: 45ms                 │  │
│  │    Remote CIDR: 10.100.0.0/16                        │  │
│  │    Accessible: 10.100.1.0/24, 10.100.2.0/24         │  │
│  │    Last handshake: 2 minutes ago                     │  │
│  │                                                      │  │
│  │    Used by:                                          │  │
│  │    ├── api-service (LEGACY_DB_URL → 10.100.1.50)    │  │
│  │    └── worker (AWS_REDIS → 10.100.2.30)             │  │
│  │                              [Configure] [Disconnect]│  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ 🟡 gcp-ml-pipeline         GCP (europe-west1)       │  │
│  │    Status: Connecting... (tunnel establishing)       │  │
│  │    Remote CIDR: 10.200.0.0/16                        │  │
│  │                              [Configure] [Delete]    │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 7.6a Create Cloud Connection

```
│  New Cloud Connection                                      │
│                                                             │
│  Provider                                                  │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐  │
│  │ ☁️ AWS    │ │ 🔵 GCP    │ │ 🟦 Azure  │ │ 🔧 Custom │  │
│  │           │ │           │ │           │ │ (IPsec/WG)│  │
│  │ [selected]│ │           │ │           │ │           │  │
│  └───────────┘ └───────────┘ └───────────┘ └───────────┘  │
│                                                             │
│  Connection Name                                           │
│  ┌───────────────────────────────────────┐                 │
│  │ aws-production                        │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  AWS Configuration                                         │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  ℹ️ In your AWS Console:                             │  │
│  │  1. Go to VPC → Virtual Private Gateway → Create     │  │
│  │  2. Create Site-to-Site VPN Connection                │  │
│  │  3. Download configuration → paste values below      │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  VPN Gateway IP (from AWS)                                 │
│  ┌───────────────────────────────────────┐                 │
│  │ 52.47.xxx.xxx                         │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  AWS VPC CIDR                                              │
│  ┌───────────────────────────────────────┐                 │
│  │ 10.100.0.0/16                         │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  Pre-shared Key (from AWS VPN config)                      │
│  ┌───────────────────────────────────────┐                 │
│  │ ••••••••••••••••                      │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  Accessible Subnets (which AWS subnets should be reachable)│
│  ┌───────────────────────────────────────┐                 │
│  │ 10.100.1.0/24                   [+ Add]│                 │
│  │ 10.100.2.0/24                   [🗑️] │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  Health Check                                              │
│  ┌───────────────────────────────────────┐                 │
│  │ Ping IP: [10.100.1.1     ]            │                 │
│  │ Interval: [30] seconds                │                 │
│  └───────────────────────────────────────┘                 │
│                                                             │
│  [Cancel]                              [Connect]           │
│                                                             │
│  ℹ️ This creates an encrypted IPsec tunnel between your    │
│     Zenith cluster and your AWS VPC. Your apps can then    │
│     reach AWS resources (RDS, ElastiCache, etc.) using     │
│     their private IP addresses.                            │
```

---

### 7.7 Gateway (Kong Management)

> Moved from Networking tab into its own sidebar item under NETWORKING group.

```
│  Gateway                                                   │
│                                                             │
│  [Routes] [Plugins] [Consumers] [Analytics]                │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Gateway Engine: Kong (KongIngress CRDs)                   │
│  Status: 🟢 Healthy · Uptime: 14d 6h                       │
│                                                             │
│  Routes                                  [+ Add Route]      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Path             Service              Port   Status  │  │
│  │  /api/v1/users    user-service         8080   🟢     │  │
│  │  /api/v1/orders   order-service        8080   🟢     │  │
│  │  /api/v1/payments payment-service      8080   🟢     │  │
│  │  /api/v1/notify   notification-svc     8080   🟢     │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Active Plugins                         [+ Enable Plugin]   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  jwt-auth           🟢 Enabled   Global   [Config]   │  │
│  │  rate-limiting       🟢 Enabled   Global   [Config]   │  │
│  │  cors               🟢 Enabled   Global   [Config]   │  │
│  │  request-transformer 🟢 Enabled   /api/*   [Config]   │  │
│  │  ip-restriction      ⚪ Disabled           [Enable]   │  │
│  │  bot-detection       🟢 Enabled   Global   [Config]   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Consumers                              [+ Add Consumer]    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  web-app         JWT    Rate: 1000/min   Last: 2m ago│  │
│  │  mobile-app      JWT    Rate: 500/min    Last: 5m ago│  │
│  │  partner-api     API Key Rate: 100/min   Last: 1h ago│  │
│  └──────────────────────────────────────────────────────┘  │
```

---

### 8. Auth (Keycloak-style - Zenith Auth)

> Auth handles tenant application users. Each tenant gets their own realm.
> Supports OpenID Connect, SAML, social logins, and MFA.

```
│  Auth                                                      │
│                                                             │
│  [Realms] [Users] [Clients] [Identity Providers] [Sessions]│
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Realm: my-startup                     [Realm Settings]     │
│  Protocol: OpenID Connect + SAML                            │
│                                                             │
│  Users                                    [+ Invite User]   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Email               Role        MFA    Status  Login│  │
│  │  admin@startup.com   Admin       🟢 On  🟢 Active 2h │  │
│  │  dev@startup.com     Developer   🟢 On  🟢 Active 1d │  │
│  │  view@startup.com    Viewer      ⚪ Off ⚪ Invited --  │  │
│  │  api@startup.com     Service     --     🟢 Active 5m │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Clients                                [+ Register Client] │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Client           Type           Protocol    Status   │  │
│  │  web-app          Public         OIDC        🟢 Active│  │
│  │  mobile-app       Public         OIDC        🟢 Active│  │
│  │  admin-panel      Confidential   OIDC        🟢 Active│  │
│  │  partner-api      Confidential   OIDC        🟢 Active│  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Identity Providers                    [+ Add Provider]     │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Google       OIDC    🟢 Connected    [Configure]    │  │
│  │  GitHub       OAuth   🟢 Connected    [Configure]    │  │
│  │  Azure AD     SAML    🟢 Connected    [Configure]    │  │
│  │  LDAP         LDAP    ⚪ Not configured [Setup]      │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Active Sessions                                           │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  admin@startup.com   Chrome/Mac    2h ago  [Revoke]  │  │
│  │  dev@startup.com     Firefox/Linux 1d ago  [Revoke]  │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  JWT tokens are validated automatically by Kong Gateway.    │
```

---

### 8.5 IAM (Platform Identity & Access Management)

> IAM is separate from Auth. Auth = for tenant app users. IAM = for platform access.
> Controls who can manage projects, deploy apps, and access the Zenith platform.

```
│  IAM - Platform Access                                     │
│                                                             │
│  [API Keys] [Team] [Roles] [SSO]                           │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  API Keys                               [+ Create Key]      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Name          Key              Scopes        Used   │  │
│  │  deploy-key    zen_sk_...xxx    deploy,write   2h ago│  │
│  │  ci-pipeline   zen_sk_...yyy    deploy,        1d ago│  │
│  │                                 registry:push        │  │
│  │  read-only     zen_sk_...zzz    read           5d ago│  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Available Scopes:                                         │
│  read · write · deploy · registry:push · admin             │
│                                                             │
│  Team Members                           [+ Invite Member]   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Email               Role        Status    Joined    │  │
│  │  admin@company.com   Owner       🟢 Active  Jan 2026 │  │
│  │  dev@company.com     Developer   🟢 Active  Feb 2026 │  │
│  │  ops@company.com     Admin       🟢 Active  Feb 2026 │  │
│  │  intern@company.com  Viewer      ⚪ Invited  --       │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Roles                                                     │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Owner     Full access, billing, delete project      │  │
│  │  Admin     All except billing and project deletion   │  │
│  │  Developer Deploy apps, manage DBs, view logs        │  │
│  │  Viewer    Read-only access to all resources         │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Platform SSO                           [Configure SSO]     │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  JumpCloud     SAML   🟢 Connected (default)         │  │
│  │  Okta          SAML   ⚪ Not configured   [Setup]     │  │
│  │  Azure AD      OIDC   ⚪ Not configured   [Setup]     │  │
│  └──────────────────────────────────────────────────────┘  │
```

---

### 9. Monitoring (Grafana + Prometheus + Loki)

> Full observability stack: Grafana dashboards, Prometheus metrics, Loki logs.
> Pre-built dashboards auto-generated per project.

```
│  Monitoring                                                │
│                                                             │
│  [Dashboards] [Prometheus] [Logs (Loki)] [Alerts]          │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Grafana Dashboards                    [Open Full Grafana]  │
│                                                             │
│  Pre-built Dashboards:                                     │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  📊 Platform Overview         Last 24h    [Open]      │  │
│  │     CPU: 58% · RAM: 82% · Req/s: 1,234 · Err: 0.02% │  │
│  │                                                      │  │
│  │  📊 Service Health            Last 1h     [Open]      │  │
│  │     12 services · 11 healthy · 1 warning             │  │
│  │                                                      │  │
│  │  📊 Node Metrics              Last 6h     [Open]      │  │
│  │     5 planets · CPU avg: 55% · RAM avg: 63%          │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  📊 Embedded Grafana Dashboard                        │  │
│  │                                                      │  │
│  │  ┌─────────────────┐  ┌─────────────────┐           │  │
│  │  │ CPU Usage       │  │ Memory Usage    │           │  │
│  │  │ [chart]         │  │ [chart]         │           │  │
│  │  │ Avg: 58%        │  │ Avg: 82%        │           │  │
│  │  └─────────────────┘  └─────────────────┘           │  │
│  │  ┌─────────────────┐  ┌─────────────────┐           │  │
│  │  │ Requests/sec    │  │ Error Rate      │           │  │
│  │  │ [chart]         │  │ [chart]         │           │  │
│  │  │ 1,234 req/s     │  │ 0.02%           │           │  │
│  │  └─────────────────┘  └─────────────────┘           │  │
│  │  ┌─────────────────────────────────────┐             │  │
│  │  │ Service Dependency Graph            │             │  │
│  │  │ [interactive graph]                 │             │  │
│  │  └─────────────────────────────────────┘             │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 9.0a Prometheus Targets

```
│  [Dashboards] [Prometheus] [Logs (Loki)] [Alerts]          │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Prometheus Targets                                        │
│  All services expose /metrics. Prometheus scrapes them.     │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Target              Endpoint          Scrape  Status│  │
│  │  frontend            :3000/metrics     15s     🟢 UP │  │
│  │  api-gateway         :8080/metrics     15s     🟢 UP │  │
│  │  user-service        :8080/metrics     15s     🟢 UP │  │
│  │  order-service       :8080/metrics     15s     🟢 UP │  │
│  │  payment-service     :8080/metrics     15s     🟢 UP │  │
│  │  notification-svc    :8080/metrics     15s     🟢 UP │  │
│  │  users-db (CNPG)     :9187/metrics     30s     🟢 UP │  │
│  │  orders-db (CNPG)    :9187/metrics     30s     🟢 UP │  │
│  │  cache (Redis)       :9121/metrics     30s     🟢 UP │  │
│  │  kong-gateway        :8100/metrics     15s     🟢 UP │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Total targets: 10 · All UP                                │
```

#### 9.0b Loki Logs

```
│  [Dashboards] [Prometheus] [Logs (Loki)] [Alerts]          │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Log Explorer (Loki)                                       │
│                                                             │
│  Service: [All Services ▼]  Level: [All ▼]  [Last 1h ▼]   │
│  Query: [                                         ] [Run]  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 14:23:01 user-service   INFO  GET /users 200 15ms    │  │
│  │ 14:23:02 api-gateway    INFO  GET /users 200 18ms    │  │
│  │ 14:23:05 order-service  WARN  Slow query: 250ms      │  │
│  │ 14:23:10 payment-svc    ERROR Stripe timeout          │  │
│  │ 14:23:11 payment-svc    ERROR Retry 1/3 failed        │  │
│  │ 14:23:15 payment-svc    INFO  Retry 2/3 succeeded     │  │
│  │ 14:23:20 user-service   INFO  POST /users 201 45ms   │  │
│  │ 14:23:25 kong-gateway   INFO  [jwt] validated token   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Color coding: INFO=green, WARN=yellow, ERROR=red, DEBUG=gray│
│  Logs aggregated from all pods via Loki + Promtail.        │
```

#### 9.1 Alerts

```
│  [Dashboard] [Alerts] [Logs]                               │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Active Alerts                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 🔴 HIGH   payment-service: Error rate > 5%   2m ago │  │
│  │ 🟡 WARN   orders-db: Disk usage > 80%        1h ago │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Alert Rules                             [+ Create Rule]    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ App down              Any app has 0 ready pods       │  │
│  │ High CPU              Cluster CPU > 85% for 5min     │  │
│  │ High error rate       Any app error rate > 5%        │  │
│  │ Disk almost full      Any volume > 80% used          │  │
│  │ Certificate expiring  SSL cert expires in < 7 days   │  │
│  └──────────────────────────────────────────────────────┘  │
```

#### 9.2 Logs

```
│  [Dashboard] [Alerts] [Logs]                               │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Service: [All Services ▼]  Level: [All ▼]  [Last 1h ▼]   │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 14:23:01 user-service   INFO  GET /users 200 15ms    │  │
│  │ 14:23:02 api-gateway    INFO  GET /users 200 18ms    │  │
│  │ 14:23:05 order-service  WARN  Slow query: 250ms      │  │
│  │ 14:23:10 payment-svc    ERROR Stripe timeout          │  │
│  │ 14:23:11 payment-svc    ERROR Retry 1/3 failed        │  │
│  │ 14:23:15 payment-svc    INFO  Retry 2/3 succeeded     │  │
│  │ 14:23:20 user-service   INFO  POST /users 201 45ms   │  │
│  └──────────────────────────────────────────────────────┘  │
```

---

### 10. Registry (ECR-style)

> Private container registry per project. ECR-style UI with repos, tags,
> vulnerability scanning, and pull commands.

```
│  Container Registry                                        │
│                                                             │
│  [Repositories] [Vulnerability Scans] [Lifecycle Policies] │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  Registry: registry.zenith.cloud/my-startup                │
│                                                             │
│  Repositories                          [+ Create Repo]      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Repo              Tags  Size     Scan    Last Push  │  │
│  │  frontend          3     245 MB   🟢 Pass  2 min ago │  │
│  │  user-service      5     189 MB   🟢 Pass  3h ago    │  │
│  │  api-gateway       2     312 MB   🟡 Warn  1d ago    │  │
│  │  order-service     4     201 MB   🟢 Pass  2d ago    │  │
│  │  payment-service   2     156 MB   🟢 Pass  3d ago    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Total: 5 repos · 16 tags · 1.1 GB used                   │
```

#### 10.0a Repository Detail

```
│  ← Registry    user-service                           [⋮]  │
│                                                             │
│  Tags                                                      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Tag      Size    Digest         Scan     Pushed     │  │
│  │  latest   189MB   sha256:a1b2..  🟢 Pass  3h ago     │  │
│  │  v12      189MB   sha256:a1b2..  🟢 Pass  3h ago     │  │
│  │  v11      185MB   sha256:c3d4..  🟢 Pass  1d ago     │  │
│  │  v10      185MB   sha256:e5f6..  🟡 1 med  3d ago    │  │
│  │  v9       182MB   sha256:g7h8..  🔴 2 high 1w ago    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Vulnerability Scan (latest)                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  🟢 No critical or high vulnerabilities found        │  │
│  │     0 Critical · 0 High · 1 Medium · 3 Low           │  │
│  │                                     [View Details]    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Pull Commands                                    [📋 Copy]│
│  ┌──────────────────────────────────────────────────────┐  │
│  │  docker login registry.zenith.cloud                  │  │
│  │  docker pull registry.zenith.cloud/my-startup/       │  │
│  │    user-service:latest                               │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Lifecycle Policy                      [Edit Policy]        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Keep last 10 tags · Delete untagged after 7 days    │  │
│  └──────────────────────────────────────────────────────┘  │
```

---

### 10.5 Docs (Auto-Generated Infrastructure Documentation)

Zenith automatically generates a live documentation page for the user's
entire infrastructure. This updates in real-time as resources change.
The user NEVER writes docs manually - Zenith reads all CRDs and generates it.

```
│  Documentation                              [↗ Open Full Page] │
│                                                             │
│  This documentation is auto-generated from your              │
│  infrastructure. It updates in real-time.                    │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  📋 Project: my-startup                              │  │
│  │  Region: fsn1 (Falkenstein, Germany)                  │  │
│  │  Plan: Pro · 3 Planets · Created: Feb 1, 2026        │  │
│  │                                                      │  │
│  │  ─────────────────────────────────────────────────   │  │
│  │                                                      │  │
│  │  🚀 Applications (4)                                 │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │ frontend                                       │  │  │
│  │  │   Image: org/frontend:v23                      │  │  │
│  │  │   URL: https://app.startup.com                 │  │  │
│  │  │   Replicas: 2 · CPU: 500m · RAM: 512Mi         │  │  │
│  │  │   Depends on: api-gateway                      │  │  │
│  │  │                                                │  │  │
│  │  │ api-gateway                                    │  │  │
│  │  │   Image: org/gateway:v45                       │  │  │
│  │  │   URL: https://api.startup.com                 │  │  │
│  │  │   Replicas: 2 · CPU: 1 · RAM: 1Gi             │  │  │
│  │  │   Routes:                                      │  │  │
│  │  │     /users/*    → user-service:8080            │  │  │
│  │  │     /orders/*   → order-service:8080           │  │  │
│  │  │   Depends on: user-service, order-service      │  │  │
│  │  │                                                │  │  │
│  │  │ user-service                                   │  │  │
│  │  │   Image: org/user-svc:v12                      │  │  │
│  │  │   Internal: http://user-service:8080           │  │  │
│  │  │   Replicas: 3 · CPU: 500m · RAM: 512Mi         │  │  │
│  │  │   Databases: users-db (PostgreSQL)             │  │  │
│  │  │   Env: DB_URL=postgres://...                   │  │  │
│  │  │                                                │  │  │
│  │  │ order-service                                  │  │  │
│  │  │   Image: org/order-svc:v8                      │  │  │
│  │  │   Internal: http://order-service:8080          │  │  │
│  │  │   Databases: orders-db, cache                  │  │  │
│  │  │   Cloud: aws-production (10.100.1.50)          │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  🗄️ Databases (3)                                    │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │ users-db        PostgreSQL 16 · 20 GB          │  │  │
│  │  │   Host: users-db-rw · Port: 5432               │  │  │
│  │  │   Used by: user-service                        │  │  │
│  │  │   Backup: Daily at 02:00 UTC (7-day retention) │  │  │
│  │  │   Hetzner Volume: vol-12345678                 │  │  │
│  │  │                                                │  │  │
│  │  │ orders-db       PostgreSQL 16 · 50 GB          │  │  │
│  │  │   Host: orders-db-rw · Port: 5432              │  │  │
│  │  │   Used by: order-service                       │  │  │
│  │  │                                                │  │  │
│  │  │ cache           Redis 7 · 5 GB                 │  │  │
│  │  │   Host: cache · Port: 6379                     │  │  │
│  │  │   Used by: order-service                       │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  📦 Storage (2)                                      │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │ uploads         S3 · 100 GB                    │  │  │
│  │  │   Endpoint: https://fsn1.your-objstorage.com   │  │  │
│  │  │   Used by: user-service (S3_ENDPOINT)          │  │  │
│  │  │                                                │  │  │
│  │  │ backups         S3 · 50 GB                     │  │  │
│  │  │   Backup destination for all databases         │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  🌐 Networking                                       │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │ Domains:                                       │  │  │
│  │  │   app.startup.com → frontend (SSL ✅)          │  │  │
│  │  │   api.startup.com → api-gateway (SSL ✅)       │  │  │
│  │  │                                                │  │  │
│  │  │ Load Balancer:                                 │  │  │
│  │  │   main-lb (LB11) · IP: 95.217.xxx.xxx         │  │  │
│  │  │                                                │  │  │
│  │  │ Firewall:                                      │  │  │
│  │  │   Allow TCP 80, 443 from 0.0.0.0/0            │  │  │
│  │  │   Allow TCP 22 from 10.0.0.0/8                │  │  │
│  │  │                                                │  │  │
│  │  │ Cloud Connections:                             │  │  │
│  │  │   🟢 aws-production (10.100.0.0/16) 45ms      │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  🌍 Planets (3)                                      │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │ zenith-cp-1   CX33 (4v, 8GB)   fsn1   🟢     │  │  │
│  │  │ zenith-cp-2   CX33 (4v, 8GB)   fsn1   🟢     │  │  │
│  │  │ zenith-cp-3   CX33 (4v, 8GB)   fsn1   🟢     │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  🔐 Auth                                             │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │ Provider: Keycloak                             │  │  │
│  │  │ SSO: JumpCloud (SAML) - default                │  │  │
│  │  │ Social: GitHub, Google                         │  │  │
│  │  │ Users: 3 (1 admin, 1 developer, 1 viewer)     │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  📊 Service Dependency Graph                         │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │       ┌──────────┐                             │  │  │
│  │  │       │ frontend │                             │  │  │
│  │  │       └────┬─────┘                             │  │  │
│  │  │       ┌────▼──────┐                            │  │  │
│  │  │       │api-gateway│                            │  │  │
│  │  │       └─┬──────┬──┘                            │  │  │
│  │  │    ┌────▼──┐ ┌─▼────────┐                     │  │  │
│  │  │    │user   │ │order     │                      │  │  │
│  │  │    │svc    │ │svc       │                      │  │  │
│  │  │    └──┬────┘ └┬──────┬─┘                      │  │  │
│  │  │    ┌──▼──┐ ┌──▼───┐ ┌▼────┐ ┌──────────┐     │  │  │
│  │  │    │users│ │orders│ │cache│ │aws-prod  │     │  │  │
│  │  │    │ db  │ │ db   │ │     │ │(10.100..)│     │  │  │
│  │  │    └─────┘ └──────┘ └─────┘ └──────────┘     │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  💰 Monthly Cost Breakdown                           │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │ Planets (3x CX33):           €16.47            │  │  │
│  │  │ Volumes (75 GB):             €3.30             │  │  │
│  │  │ Load Balancer (LB11):        €5.49             │  │  │
│  │  │ Object Storage (150 GB):     €4.99             │  │  │
│  │  │ TOTAL (Hetzner):             €30.25            │  │  │
│  │  │ Zenith platform:             €0.00 (free)      │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  Last updated: 2 minutes ago · Auto-refreshes        │  │
│  │                                                      │  │
│  │  [📄 Export as PDF]  [📋 Copy as Markdown]           │  │
│  │                                                      │  │
│  └──────────────────────────────────────────────────────┘  │
```

---

### 11. Planets (Node Scaling)

```
│  Planets                                [+ Add a Planet]    │
│                                                             │
│  Current Cluster: 3 Planets · 12 vCPU · 24 GB RAM          │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  🌍 zenith-cp-1        CX33 (4 vCPU, 8GB)           │  │
│  │     Status: 🟢 Ready   Region: fsn1                  │  │
│  │     CPU: ████████░░ 78%   RAM: ██████████░ 91%       │  │
│  │     Pods: 12/25            Uptime: 14 days            │  │
│  │     Role: Control Plane                               │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │  🌍 zenith-cp-2        CX33 (4 vCPU, 8GB)           │  │
│  │     Status: 🟢 Ready   Region: fsn1                  │  │
│  │     CPU: ██████░░░░ 55%   RAM: █████████░░ 85%       │  │
│  │     Pods: 10/25            Uptime: 14 days            │  │
│  │     Role: Control Plane                               │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │  🌍 zenith-cp-3        CX33 (4 vCPU, 8GB)           │  │
│  │     Status: 🟢 Ready   Region: fsn1                  │  │
│  │     CPU: █████░░░░░ 45%   RAM: ████████░░░ 72%       │  │
│  │     Pods: 8/25             Uptime: 14 days            │  │
│  │     Role: Control Plane                               │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Total: €16.47/mo (3 × CX33 €5.49)                        │
```

#### 11.1 Add a Planet Modal

```
┌───────────────────────────────────────────┐
│  Add a Planet 🌍                          │
│                                           │
│  Planet Size                              │
│  ┌───────────┐ ┌───────────┐ ┌─────────┐ │
│  │ 🌑 Nano   │ │ 🌒 Small  │ │ 🌓 Med  │ │
│  │ CX23      │ │ CX33      │ │ CX43    │ │
│  │ 2v 4GB    │ │ 4v 8GB    │ │ 8v 16GB │ │
│  │ €3.49/mo  │ │ €5.49/mo  │ │ €9.49/mo│ │
│  └───────────┘ └───────────┘ └─────────┘ │
│  ┌───────────┐ ┌───────────┐ ┌─────────┐ │
│  │ 🌔 Large  │ │ 🌕 Mega   │ │ ⭐ Ultra│ │
│  │ CX53      │ │ CCX23     │ │ CCX33   │ │
│  │ 16v 32GB  │ │ 4d 16GB   │ │ 8d 32GB │ │
│  │ €17.49/mo │ │ €24.49/mo │ │ €48.49  │ │
│  └───────────┘ └───────────┘ └─────────┘ │
│                                           │
│  Region                                   │
│  ┌─────────────────────────────────────┐  │
│  │ 🇩🇪 Falkenstein (fsn1)         ▼   │  │
│  └─────────────────────────────────────┘  │
│                                           │
│  New node will be ready in ~90 seconds.   │
│  It auto-joins your cluster.              │
│                                           │
│  [Cancel]                [Add Planet]     │
│                                           │
│  Monthly cost increase: +€5.49/mo         │
└───────────────────────────────────────────┘
```

---

### 12. Settings

```
│  Settings                                                  │
│                                                             │
│  [General] [Team] [Billing] [Danger Zone]                  │
│  ─────────────────────────────────────────────────────────  │
│                                                             │
│  General                                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Project Name                                        │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │ my-startup                                     │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  Region: fsn1 (Falkenstein, Germany)                  │  │
│  │  Created: February 1, 2026                           │  │
│  │  Plan: Pro                                           │  │
│  │  Hetzner Token: hc_****xxxx          [Update]        │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Danger Zone                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  ⚠️ Delete Project                                   │  │
│  │  This will destroy all apps, databases, storage,     │  │
│  │  and Hetzner resources. This cannot be undone.       │  │
│  │                                                      │  │
│  │  Type "my-startup" to confirm:                       │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │                                                │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                               [Delete Project]       │  │
│  └──────────────────────────────────────────────────────┘  │
```

---

### 13. Billing

```
│  Billing                                                   │
│                                                             │
│  Current Month: February 2026                              │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                                                      │  │
│  │  Estimated Cost:  €47.23                             │  │
│  │                                                      │  │
│  │  Breakdown:                                          │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │  Planets (3x CX33)            €16.47          │  │  │
│  │  │  Volumes (75 GB)              €3.30           │  │  │
│  │  │  Load Balancer (LB11)         €5.49           │  │  │
│  │  │  Object Storage (68 GB)       €4.99           │  │  │
│  │  │  Floating IPs (1)             €4.32           │  │  │
│  │  │  DNS Zone (1)                 €0.00           │  │  │
│  │  │  Bandwidth (450 GB / 20 TB)   €0.00           │  │  │
│  │  │  ──────────────────────────────────            │  │  │
│  │  │  Total (paid to Hetzner)      €34.57          │  │  │
│  │  │  Zenith platform              €0.00 (free!)   │  │  │
│  │  │  ──────────────────────────────────            │  │  │
│  │  │  TOTAL                        €34.57          │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  │  ℹ️ You pay Hetzner directly. Zenith is 100% free.   │  │
│  │                                                      │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  Hetzner Account                                           │
│  Token: hc_****xxxx               [Update Token]           │
│  ℹ️ Zenith uses your Hetzner API token to manage           │
│     resources. All costs are billed by Hetzner directly.   │
```

---

## Component Library (shadcn/ui based)

```
Base Components (from shadcn/ui):
├── Button (primary, secondary, destructive, ghost)
├── Input
├── Select
├── Slider
├── Switch / Toggle
├── Dialog / Modal
├── DropdownMenu
├── Table
├── Tabs
├── Badge (status: green, yellow, red, gray)
├── Card
├── Toast (success, error, info)
├── Skeleton (loading states)
├── Command (search palette: Cmd+K)
└── Sheet (mobile sidebar)

Custom Components (Zenith-specific):
├── StatusBadge (🟢 Running, 🟡 Deploying, 🔴 Failed, ⚪ Stopped)
├── ResourceBar (CPU/RAM/Disk usage bar with percentage)
├── CopyButton (click to copy, shows "Copied!" toast)
├── LogViewer (real-time log stream with filters)
├── PlanetSelector (planet size cards with specs + price)
├── CostEstimate (inline "~€X/mo" display)
├── ServiceGraph (interactive dependency graph)
├── MetricsChart (CPU/RAM/requests time series)
├── TerminalOutput (build logs, deploy logs)
├── EnvVarEditor (key-value editor with secret toggle)
├── DatabaseLinker (link DB to app, auto-inject env vars)
├── DomainVerifier (shows CNAME instructions + verification status)
└── EmptyState (illustration + CTA for empty lists)
```

---

## States Every Page Must Handle

```
1. Loading        → Skeleton/shimmer animation
2. Empty          → EmptyState component with CTA
3. Data           → Normal display
4. Error          → Error message with retry button
5. Creating       → Progress indicator
6. Deleting       → Confirmation dialog → loading → redirect
7. Updating       → Inline loading spinner
8. Unauthorized   → Redirect to login
```

---

## Real-time Updates

```
WebSocket connections for:
├── App logs (streaming)
├── Build logs (streaming)
├── App status changes (deploy started, deploy finished)
├── Planet joining (creating → joining → ready)
├── Database status (creating → ready)
└── Alert notifications (toast popup)

Implementation: WebSocket from frontend → Zenith API → kubectl logs/watch
```

---

## Mission Control - Platform Operator Panel (SEPARATE APP)

> This is NOT part of the user-facing Zenith UI. This is a separate app at ms.freezenith.com.
> Different login. Different audience. Only the platform operator sees this.

**URL:** `back.your-domain.com`
**Tech:** Next.js 15, TypeScript, Tailwind, shadcn/ui (same stack, different theme)
**Theme:** Dark, blue accent (#3b82f6) - visually distinct from user-facing green
**Auth:** Separate admin credentials (set during `zen install`)

### B0. Welcome Wizard (first-time only)

Shown on first login after `zen install`. No sidebar, no header. Full-screen wizard.

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│     ⬡ Welcome to Zenith                                          │
│                                                                  │
│     Let's set up your platform. This takes about 5 minutes.      │
│     You can always change these settings later.                  │
│                                                                  │
│     Step 1 of 3: Choose a region                                 │
│     ─────────────────────────────                                │
│                                                                  │
│     Where should your platform run?                              │
│     Pick the region closest to your users.                       │
│                                                                  │
│     ┌─────────────────────────────────────────────────────────┐  │
│     │  ● Falkenstein, Germany (fsn1)                          │  │
│     │    Lowest latency to Central Europe                     │  │
│     ├─────────────────────────────────────────────────────────┤  │
│     │  ○ Nuremberg, Germany (nbg1)                            │  │
│     ├─────────────────────────────────────────────────────────┤  │
│     │  ○ Helsinki, Finland (hel1)                             │  │
│     ├─────────────────────────────────────────────────────────┤  │
│     │  ○ Ashburn, USA (ash)                                   │  │
│     ├─────────────────────────────────────────────────────────┤  │
│     │  ○ Hillsboro, USA (hil)                                 │  │
│     └─────────────────────────────────────────────────────────┘  │
│                                                                  │
│                                                     [Next →]     │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│     Step 2 of 3: Choose platform size                            │
│     ────────────────────────────────                             │
│                                                                  │
│     How many resources do you need?                              │
│     You can add more nodes later at any time.                    │
│                                                                  │
│     ┌─────────────────────────────────────────────────────────┐  │
│     │  ○ Starter                                              │  │
│     │    3 nodes · 6 vCPU · 12GB RAM · ~30 apps               │  │
│     │    ~€13/mo                                              │  │
│     ├─────────────────────────────────────────────────────────┤  │
│     │  ● Standard (recommended)                               │  │
│     │    5 nodes · 10 vCPU · 20GB RAM · ~80 apps              │  │
│     │    ~€22/mo                                              │  │
│     ├─────────────────────────────────────────────────────────┤  │
│     │  ○ Large                                                │  │
│     │    8 nodes · 16 vCPU · 32GB RAM · ~150 apps             │  │
│     │    ~€36/mo                                              │  │
│     ├─────────────────────────────────────────────────────────┤  │
│     │  ○ Custom                                               │  │
│     │    Choose exact node sizes and count                    │  │
│     └─────────────────────────────────────────────────────────┘  │
│                                                                  │
│                                            [← Back]  [Next →]   │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│     Step 3 of 3: Confirm & Create                                │
│     ─────────────────────────────                                │
│                                                                  │
│     Region:     Falkenstein, Germany (fsn1)                      │
│     Size:       Standard (5x CX22)                               │
│     Est. cost:  ~€27/mo (5 nodes + management server)            │
│     Domain:     app.myplatform.com                               │
│                                                                  │
│     What will happen:                                            │
│     1. Create 5 servers on Hetzner        ~2 min                 │
│     2. Form Kubernetes cluster            ~1 min                 │
│     3. Install Zenith platform            ~2 min                 │
│     4. Configure DNS + SSL                ~30 sec                │
│                                                                  │
│                         [← Back]  [🚀 Create Platform]           │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│     Creating your platform...                                    │
│                                                                  │
│     ████████████████████░░░░░░░░░░ 55%                           │
│                                                                  │
│     ✅ Created 5 servers on Hetzner                               │
│     ✅ Kubernetes cluster formed (v1.30.2)                        │
│     🔄 Installing Zenith operators...                             │
│     ○  Configuring DNS                                           │
│     ○  Provisioning SSL certificate                              │
│                                                                  │
│     ⏱ ~2 minutes remaining                                       │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│     🎉 Your platform is ready!                                    │
│                                                                  │
│     Platform:    https://app.myplatform.com                      │
│     Admin:       https://back.myplatform.com (you are here)      │
│     K8s version: v1.30.2                                         │
│     Nodes:       5 (all healthy)                                 │
│                                                                  │
│     ┌──────────────────┐  ┌──────────────────┐                   │
│     │ Open Platform →  │  │ Go to Dashboard  │                   │
│     └──────────────────┘  └──────────────────┘                   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### B8. State Page

```
┌──────────────────────────────────────────────────────────────────┐
│ Platform State                                    [Export JSON]   │
│                                                                  │
│ Complete snapshot of everything installed on your platform.       │
│                                                                  │
│ Platform                                                         │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Zenith Version:      v1.2.1                                │   │
│ │ Installed:           Jan 28, 2026                          │   │
│ │ Management K8s:      v1.30.2 (k3s)                         │   │
│ │ Domain:              myplatform.com                         │   │
│ │ Hetzner Region:      fsn1                                  │   │
│ │ State DB:            /var/lib/zenith-mc/state.db (42MB)  │   │
│ │ Last backup:         2 hours ago → s3://zenith-backups/    │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Clusters                                                         │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Name             K8s      Nodes  Created       Status      │   │
│ │ ──────────────────────────────────────────────────────────  │   │
│ │ zenith-shared    v1.30.2  8      Dec 01, 2025  🟢 Healthy  │   │
│ │ pro-acme         v1.29.4  4      Jan 15, 2026  🟢 Healthy  │   │
│ │ pro-enterprise   v1.30.2  12     Feb 01, 2026  🟢 Healthy  │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Installed Modules (per cluster)                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Module            zenith-shared  pro-acme  pro-enterprise  │   │
│ │ ──────────────────────────────────────────────────────────  │   │
│ │ Zenith Operator   v1.2.1         v1.2.1    v1.2.1          │   │
│ │ CloudNativePG     v1.22.1        v1.22.1   v1.23.0         │   │
│ │ Redis Operator    v7.2.0         v7.2.0    v7.2.0          │   │
│ │ cert-manager      v1.14.2        v1.14.2   v1.14.2         │   │
│ │ Traefik           v2.11.0        v2.11.0   v2.11.0         │   │
│ │ Harbor            v2.10.0        -          v2.10.1         │   │
│ │ Keycloak          v24.0          -          v24.0           │   │
│ │ Prometheus        v56.2          v56.2     v56.2           │   │
│ │ Loki              v3.0.1         -          v3.0.1          │   │
│ │ NATS              v2.10.0        -          v2.10.0         │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Hetzner Resources                                                │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Resource       Count    Details                            │   │
│ │ ──────────────────────────────────────────────────────────  │   │
│ │ Servers        26       1 mgmt + 8 + 4 + 12 + 1 spare     │   │
│ │ Volumes        89       1.2TB total                        │   │
│ │ Load Balancers 4        1 per cluster + 1 mgmt             │   │
│ │ Floating IPs   3                                           │   │
│ │ Networks       3        1 per cluster                      │   │
│ │ Firewalls      6        2 per cluster (mgmt + worker)      │   │
│ │ SSH Keys       1        zenith-key                         │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Backup Status                                                    │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ What                  Last Backup    Next     Destination  │   │
│ │ ──────────────────────────────────────────────────────────  │   │
│ │ Mgmt etcd             2h ago         4h       s3://backups │   │
│ │ State DB (SQLite)     2h ago         4h       s3://backups │   │
│ │ Workload etcd (shared)2h ago         4h       s3://backups │   │
│ │ Workload etcd (acme)  2h ago         4h       s3://backups │   │
│ │ Workload etcd (ent.)  2h ago         4h       s3://backups │   │
│ │ DB backups (CNPG)     6h ago         18h      s3://backups │   │
│ └────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

### Layout

```
┌──────────────────────────────────────────────────────────────────┐
│ ┌──────┐  Mission Control             Zenith v1.2.1   [🔔] [admin] │
│ │ ⚙️   │                                                        │
├─┴──────┴─────────────────────────────────────────────────────────┤
│ ┌──────────┐ ┌──────────────────────────────────────────────┐    │
│ │          │ │                                              │    │
│ │ SIDEBAR  │ │              MAIN CONTENT                    │    │
│ │          │ │                                              │    │
│ │ Dashboard│ │                                              │    │
│ │ Clusters │ │                                              │    │
│ │ Modules  │ │                                              │    │
│ │ Updates  │ │                                              │    │
│ │ Tenants  │ │                                              │    │
│ │ Infra    │ │                                              │    │
│ │ State    │ │                                              │    │
│ │ Audit    │ │                                              │    │
│ │          │ │                                              │    │
│ │──────────│ │                                              │    │
│ │ Settings │ │                                              │    │
│ └──────────┘ └──────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

### B1. Dashboard (ms.freezenith.com)

```
┌──────────────────────────────────────────────────────────────────┐
│ Platform Overview                                                │
│                                                                  │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────┐│
│ │ Clusters     │ │ Tenants      │ │ Monthly Cost │ │ Updates  ││
│ │    3         │ │    47        │ │  €127.40     │ │ 2 avail  ││
│ │ (all healthy)│ │ (12 active)  │ │ (Hetzner)    │ │ ⚠        ││
│ └──────────────┘ └──────────────┘ └──────────────┘ └──────────┘│
│                                                                  │
│ Clusters                                                         │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Name            K8s Ver    Nodes  CPU Used  RAM Used  Health│  │
│ │ ────────────────────────────────────────────────────────── │   │
│ │ zenith-shared   v1.29.4    8      62%       58%      🟢   │   │
│ │ pro-startup-a   v1.29.4    4      45%       52%      🟢   │   │
│ │ pro-enterprise  v1.28.6    12     71%       68%      🟡   │   │
│ │                                         ⚠ upgrade avail    │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Available Updates                                                │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ 🆕 Zenith v1.3.0            Current: v1.2.1   [View]      │   │
│ │ 🆕 CloudNativePG v1.23.0    Current: v1.22.1  [View]      │   │
│ │ ⚠  Kubernetes v1.30.2       On: pro-enterprise [View]     │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Recent Activity                                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ 14:23  admin upgraded CloudNativePG v1.21 → v1.22         │   │
│ │ 12:01  CAPI scaled zenith-shared: 7 → 8 nodes             │   │
│ │ 09:45  Tenant "startup-x" created (Starter plan)           │   │
│ │ 08:12  Backup completed: all databases (47 tenants)        │   │
│ └────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

### B2. Clusters Page

```
┌──────────────────────────────────────────────────────────────────┐
│ Clusters                                         [+ New Cluster] │
│                                                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ zenith-shared                                   🟢 Healthy │   │
│ │ K8s v1.29.4 · 8 nodes · fsn1 · Shared cluster             │   │
│ │                                                            │   │
│ │ CPU: ████████████░░░░░ 62%    RAM: █████████░░░░░░ 58%     │   │
│ │ Pods: 234/500                 PVCs: 89/200                 │   │
│ │                                                            │   │
│ │ [Upgrade K8s] [Scale Nodes] [View Details]                 │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ pro-enterprise                                  🟡 Warning │   │
│ │ K8s v1.28.6 · 12 nodes · nbg1 · Dedicated (ACME Corp)     │   │
│ │                                                            │   │
│ │ ⚠ K8s v1.28 is deprecated. Upgrade to v1.29 or v1.30.    │   │
│ │                                                            │   │
│ │ CPU: ██████████████░░░ 71%    RAM: █████████████░░ 68%     │   │
│ │                                                            │   │
│ │ [Upgrade K8s ⚠] [Scale Nodes] [View Details]              │   │
│ └────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

### B2.1 Cluster Detail → Upgrade Kubernetes

```
┌──────────────────────────────────────────────────────────────────┐
│ ← Clusters / zenith-shared / Upgrade Kubernetes                  │
│                                                                  │
│ Current Version: v1.29.4                                         │
│                                                                  │
│ Available Upgrades:                                              │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ ● v1.30.2 (recommended)                                    │   │
│ │   Released: 2026-01-15                                     │   │
│ │   Changelog: Security fixes, improved scheduling,          │   │
│ │   node memory management improvements.                     │   │
│ │                                                            │   │
│ │ ○ v1.29.6 (patch only)                                     │   │
│ │   Released: 2026-02-01                                     │   │
│ │   Changelog: CVE-2026-1234 fix, etcd stability.            │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Upgrade Strategy: Rolling (zero-downtime)                        │
│                                                                  │
│ What happens:                                                    │
│ 1. New nodes created with v1.30.2                                │
│ 2. Old nodes drained one at a time                               │
│ 3. Pods moved to new nodes                                       │
│ 4. Old nodes deleted                                             │
│ 5. All 8 nodes upgraded (~15-20 min)                             │
│                                                                  │
│ ⚠ Workloads keep running. No downtime.                          │
│                                                                  │
│                            [Cancel]  [🔄 Start Upgrade]          │
└──────────────────────────────────────────────────────────────────┘
```

### B2.2 Upgrade Progress

```
┌──────────────────────────────────────────────────────────────────┐
│ ← Clusters / zenith-shared / Upgrading to v1.30.2               │
│                                                                  │
│ Progress: 3/8 nodes upgraded                                     │
│ ████████████░░░░░░░░░░░░░░░░░░░░ 37%                            │
│                                                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Node          Status              K8s Version              │   │
│ │ ─────────────────────────────────────────────────────────  │   │
│ │ planet-01     ✅ Upgraded          v1.30.2                  │   │
│ │ planet-02     ✅ Upgraded          v1.30.2                  │   │
│ │ planet-03     ✅ Upgraded          v1.30.2                  │   │
│ │ planet-04     🔄 Draining pods...  v1.29.4 → v1.30.2      │   │
│ │ planet-05     ⏳ Waiting           v1.29.4                  │   │
│ │ planet-06     ⏳ Waiting           v1.29.4                  │   │
│ │ planet-07     ⏳ Waiting           v1.29.4                  │   │
│ │ planet-08     ⏳ Waiting           v1.29.4                  │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Live log:                                                        │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ 14:23:12  Cordoned planet-04                               │   │
│ │ 14:23:15  Draining planet-04 (23 pods to relocate)         │   │
│ │ 14:23:18  Pod user-svc-abc moved to planet-02              │   │
│ │ 14:23:19  Pod order-svc-def moved to planet-03             │   │
│ │ 14:23:22  ...                                              │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Estimated time remaining: ~10 minutes                            │
│                                                                  │
│ [⏸ Pause Upgrade]  (stops after current node, safe to resume)   │
└──────────────────────────────────────────────────────────────────┘
```

### B3. Modules Page

```
┌──────────────────────────────────────────────────────────────────┐
│ Modules                                                [Refresh] │
│                                                                  │
│ Infrastructure modules installed on your workload clusters.      │
│ Updates are safe - operators handle rolling upgrades internally.  │
│                                                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Module              Installed    Latest     Status         │   │
│ │ ───────────────────────────────────────────────────────── │   │
│ │ Zenith Operator     v1.2.1       v1.3.0     🆕 Update     │   │
│ │ CloudNativePG       v1.22.1      v1.23.0    🆕 Update     │   │
│ │ Redis Operator      v7.2.0       v7.2.0     ✅ Up to date  │   │
│ │ cert-manager        v1.14.2      v1.14.2    ✅ Up to date  │   │
│ │ Traefik             v2.11.0      v2.11.0    ✅ Up to date  │   │
│ │ Harbor              v2.10.0      v2.10.1    🆕 Update      │   │
│ │ Keycloak Operator   v24.0        v24.0      ✅ Up to date  │   │
│ │ Prometheus Stack    v56.2        v56.2      ✅ Up to date  │   │
│ │ Loki                v3.0.1       v3.0.1     ✅ Up to date  │   │
│ │ NATS                v2.10.0      v2.10.0    ✅ Up to date  │   │
│ │ Linkerd             v2.14.0      v2.14.1    🆕 Update      │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ 4 updates available                   [Update All] [Select...]   │
└──────────────────────────────────────────────────────────────────┘
```

### B3.1 Module Update Detail

```
┌──────────────────────────────────────────────────────────────────┐
│ ← Modules / CloudNativePG                                        │
│                                                                  │
│ CloudNativePG - PostgreSQL Operator                              │
│                                                                  │
│ Installed: v1.22.1                                               │
│ Available: v1.23.0                                               │
│ Cluster: zenith-shared (also on pro-enterprise, pro-startup-a)   │
│                                                                  │
│ Changelog (v1.23.0):                                             │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ • Online major version upgrade for PostgreSQL              │   │
│ │ • Improved backup verification                             │   │
│ │ • PgBouncer connection pooling improvements                │   │
│ │ • Fixed: WAL archiving race condition (#4521)              │   │
│ │ • Fixed: Replica promotion timeout (#4498)                 │   │
│ │                                                            │   │
│ │ ⚠ Breaking: None                                          │   │
│ │ Min K8s: v1.27+                                            │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Impact:                                                          │
│ • 23 PostgreSQL databases across 3 clusters will be reconciled   │
│ • No downtime - operator upgrade is independent of databases     │
│ • Existing databases continue running                            │
│                                                                  │
│ Update on:                                                       │
│ ☑ zenith-shared                                                  │
│ ☑ pro-startup-a                                                  │
│ ☑ pro-enterprise                                                 │
│                                                                  │
│           [Cancel]  [Update CloudNativePG to v1.23.0]            │
└──────────────────────────────────────────────────────────────────┘
```

### B4. Updates Page (Platform Updates)

```
┌──────────────────────────────────────────────────────────────────┐
│ Platform Updates                                                 │
│                                                                  │
│ Update source: freezenith.com                                    │
│ Last checked: 2 hours ago                          [Check Now]   │
│                                                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ 🆕 Zenith v1.3.0 available                                 │   │
│ │                                                            │   │
│ │ Released: February 10, 2026                                │   │
│ │ Current: v1.2.1 (installed January 28, 2026)               │   │
│ │                                                            │   │
│ │ What's new:                                                │   │
│ │ ✦ MongoDB support (Database module)                        │   │
│ │ ✦ Cloud Connections (AWS/GCP/Azure VPN tunnels)            │   │
│ │ ✦ GitOps mode (zen export/apply/sync)                      │   │
│ │ ✦ Auto-generated infrastructure documentation              │   │
│ │ ✦ SSO/SAML/OIDC with JumpCloud, Okta, Azure AD            │   │
│ │ ✦ 47 bug fixes, 12 performance improvements               │   │
│ │                                                            │   │
│ │ Breaking changes: None                                     │   │
│ │ New CRDs: CloudConnector, GitSync, AuthRealm (v2)         │   │
│ │                                                            │   │
│ │ [Full Release Notes]          [⬆ Upgrade to v1.3.0]       │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Update History                                                   │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ v1.2.1  Jan 28, 2026  admin  ✅ Success                   │   │
│ │ v1.2.0  Jan 15, 2026  admin  ✅ Success                   │   │
│ │ v1.1.0  Dec 20, 2025  admin  ✅ Success                   │   │
│ │ v1.0.0  Dec 01, 2025  admin  ✅ Initial install           │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Auto-update settings:                                            │
│   Notify on new versions: [✅ On]                                │
│   Auto-apply patch updates (x.x.Z): [○ Off]                     │
│   Auto-apply minor updates (x.Y.0): [○ Off]                     │
└──────────────────────────────────────────────────────────────────┘
```

### B5. Tenants Page

```
┌──────────────────────────────────────────────────────────────────┐
│ Tenants                                                          │
│                                                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Project          Plan      Apps  DBs  CPU     RAM    Status│   │
│ │ ────────────────────────────────────────────────────────── │   │
│ │ my-startup       Starter   12    3    2.4/4   3.1/4  🟢   │   │
│ │ acme-corp        Pro       45    8    8.2/16  12/16  🟢   │   │
│ │ dev-agency       Starter   3     1    0.5/4   0.8/4  🟢   │   │
│ │ test-project     Starter   1     0    0.1/4   0.2/4  ⚪   │   │
│ │ enterprise-x     Pro       87    12   22/32   28/32  🟡   │   │
│ │ ...                                                        │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Total: 47 tenants · 234 apps · 89 databases                     │
│        CPU: 62% used · RAM: 58% used                             │
└──────────────────────────────────────────────────────────────────┘
```

### B6. Infrastructure Page

```
┌──────────────────────────────────────────────────────────────────┐
│ Infrastructure                                                   │
│                                                                  │
│ Hetzner Account: team@company.com                                │
│                                                                  │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────┐│
│ │ Servers      │ │ Volumes      │ │ Load Balancers│ │ Monthly  ││
│ │   25         │ │   89         │ │   4           │ │ €127.40  ││
│ │ (24 active)  │ │ (1.2TB used) │ │ (all healthy) │ │          ││
│ └──────────────┘ └──────────────┘ └──────────────┘ └──────────┘│
│                                                                  │
│ Resource Breakdown                                               │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Resource          Count    Cost/mo   Notes                 │   │
│ │ ────────────────────────────────────────────────────────── │   │
│ │ Management CX22   1        €4.49     Mission Control + CAPI    │   │
│ │ CX22 (shared)     8        €35.92    zenith-shared cluster │   │
│ │ CX32 (pro-A)      4        €29.56    pro-startup-a         │   │
│ │ CX42 (pro-B)      12       €141.48   pro-enterprise        │   │
│ │ Volumes           89       €38.70    PVCs for databases    │   │
│ │ Load Balancers    4        €22.76    Cluster ingress       │   │
│ │ Floating IPs      3        €10.71    Static IPs            │   │
│ │ Snapshots         12       €5.40     Backups               │   │
│ │ ────────────────────────────────────────────────────────── │   │
│ │ TOTAL                      €289.02                         │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Capacity Planning                                                │
│ CPU:  █████████████░░░░░░░░ 62% (used 48 of 78 vCPU)            │
│ RAM:  ██████████░░░░░░░░░░░ 58% (used 89 of 152 GB)             │
│ Disk: ████████░░░░░░░░░░░░░ 42% (used 1.2 of 2.8 TB)           │
│                                                                  │
│ 💡 Capacity looks good. No action needed.                        │
└──────────────────────────────────────────────────────────────────┘
```

### B7. Audit Log Page

```
┌──────────────────────────────────────────────────────────────────┐
│ Audit Log                                     [Export CSV]       │
│                                                                  │
│ Filter: [All ▼]  [All clusters ▼]  [Last 7 days ▼]  [Search..] │
│                                                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Time            Actor    Action                   Cluster  │   │
│ │ ────────────────────────────────────────────────────────── │   │
│ │ Feb 15 14:23    admin    Upgraded CloudNativePG    shared  │   │
│ │                          v1.22.1 → v1.23.0                 │   │
│ │ Feb 15 12:01    CAPI     Scaled nodes 7 → 8       shared  │   │
│ │ Feb 15 09:45    system   Tenant created: startup-x shared  │   │
│ │ Feb 14 22:10    admin    Upgraded Zenith v1.2.0→1.2.1 all  │   │
│ │ Feb 14 18:30    admin    Created cluster: pro-ent  pro-ent │   │
│ │ Feb 14 15:00    CAPI     K8s upgrade v1.28→v1.29   pro-A  │   │
│ │ ...                                                        │   │
│ └────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

---

## UX Principles - The User Must NEVER Get Confused

> If the user has to think, we failed. If the user needs docs to understand the UI, we failed.

### Core Rule: Zero Kubernetes Knowledge Required

The user does NOT know what these are and should NEVER see them:
- Pod, Deployment, StatefulSet, DaemonSet, ReplicaSet
- PVC, PV, StorageClass
- Ingress, IngressRoute, Service, Endpoint
- ConfigMap, Secret (show as "Environment Variables")
- Namespace (show as "Project")
- Node (show as "Planet")
- CRD, Operator, Controller
- YAML (unless they choose GitOps mode)

### Principle 1: Progressive Disclosure

Show simple first. Details on demand.

```
❌ BAD: Show everything at once
┌──────────────────────────────────────────────────────────┐
│ Create Database                                          │
│                                                          │
│ Name: [____________]    Engine: [PostgreSQL ▼]           │
│ Version: [16 ▼]         Storage: [20GB ▼]               │
│ CPU: [500m ▼]           Memory: [512Mi ▼]               │
│ Replicas: [1 ▼]         Backup Schedule: [0 3 * * * ▼]  │
│ WAL Level: [replica ▼]  Max Connections: [100]           │
│ Shared Buffers: [____]  PG Bouncer: [on/off]            │
│ ...20 more fields...                                     │
└──────────────────────────────────────────────────────────┘

✅ GOOD: Simple first, details on demand
┌──────────────────────────────────────────────────────────┐
│ Create Database                                          │
│                                                          │
│ Name: [____________]                                     │
│                                                          │
│ PostgreSQL ● │ MySQL ○ │ MongoDB ○ │ Redis ○             │
│                                                          │
│ Size:  ○ Small (1GB)   ● Medium (5GB)   ○ Large (20GB)  │
│                                                          │
│                            [Create Database]             │
│                                                          │
│ ▸ Advanced options (optional)                            │
└──────────────────────────────────────────────────────────┘
```

### Principle 2: Sensible Defaults

Every field has a good default. The user should be able to click "Create" without changing anything.

```
App defaults:
  Port: 8080 (auto-detected from Dockerfile)
  Replicas: 1
  Resources: auto (500m CPU, 512Mi RAM)
  Health check: / on app port
  Branch: main
  Auto-deploy on push: yes

Database defaults:
  Version: latest stable
  Storage: 5GB
  Backups: daily at 3am, keep 7 days
  Connection pooling: on

Domain defaults:
  SSL: auto (Let's Encrypt)
  Force HTTPS: yes
  WWW redirect: yes
```

### Principle 3: No Jargon

```
❌ Kubernetes jargon          ✅ Human language
─────────────────────         ─────────────────────
"Pod"                         "Instance"
"Node"                        "Planet" (our brand)
"Namespace"                   "Project"
"PersistentVolumeClaim"       "Storage"
"Ingress"                     "Domain"
"Service"                     (hidden, user doesn't see this)
"ConfigMap"                   "Environment Variables"
"Secret"                      "Environment Variables" (marked as sensitive)
"Replica"                     "Instance" (scale to 3 instances)
"HPA"                         "Auto-scaling" toggle
"CRD"                         (never shown to user)
"Reconciling"                 "Setting up..." / "Updating..."
"ImagePullBackOff"            "Image not found. Check the repository URL."
"CrashLoopBackOff"            "App keeps crashing. Check the logs."
"OOMKilled"                   "App ran out of memory. Increase the memory limit."
"Pending"                     "Waiting for resources..."
"Evicted"                     "Moved to another planet (auto-healed)"
```

### Principle 4: Wizard-Style Flows for Complex Actions

Multi-step actions use a wizard, not a single overloaded form.

```
Deploy an App (3 steps):

Step 1 of 3: Source
┌──────────────────────────────────────────────────────────┐
│ Where is your code?                                      │
│                                                          │
│ ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │
│ │  ◉ GitHub   │  │  ○ Docker   │  │  ○ Upload    │      │
│ │    repo     │  │    image    │  │    archive   │      │
│ └─────────────┘  └─────────────┘  └─────────────┘      │
│                                                          │
│ Repository: [startup/frontend        ▼]                  │
│ Branch:     [main                      ]                 │
│                                                          │
│                           [Back]  [Next: Build Settings] │
└──────────────────────────────────────────────────────────┘

Step 2 of 3: Build
┌──────────────────────────────────────────────────────────┐
│ How should we build it?                                  │
│                                                          │
│ ✅ We detected a Dockerfile in your repo                 │
│    Using: ./Dockerfile                                   │
│                                                          │
│ Port your app listens on: [3000]                         │
│ (auto-detected from Dockerfile EXPOSE)                   │
│                                                          │
│                           [Back]  [Next: Review & Deploy]│
└──────────────────────────────────────────────────────────┘

Step 3 of 3: Review
┌──────────────────────────────────────────────────────────┐
│ Ready to deploy!                                         │
│                                                          │
│ Source:    github.com/startup/frontend (main)            │
│ Build:    Dockerfile                                     │
│ Port:     3000                                           │
│ Size:     Small (1 CPU, 512MB RAM)                       │
│ Domain:   frontend-abc123.zenith.app (free subdomain)    │
│                                                          │
│ ℹ You can add a custom domain later.                     │
│                                                          │
│                           [Back]  [🚀 Deploy]           │
└──────────────────────────────────────────────────────────┘
```

### Principle 5: Contextual Help (Inline, Not Docs)

Every non-obvious field has a tooltip or helper text. Users should never leave the UI to understand something.

```
┌──────────────────────────────────────────────────────────┐
│ Environment Variables                                    │
│                                                          │
│ These are passed to your app at runtime.                 │
│ Database connection strings are added automatically      │
│ when you link a database.                                │
│                                                          │
│ Key              Value                   Sensitive       │
│ [DATABASE_URL  ] [postgres://...       ] [🔒 yes]       │
│ [STRIPE_KEY    ] [sk_live_...          ] [🔒 yes]       │
│ [LOG_LEVEL     ] [info                 ] [   no ]       │
│                                                          │
│ [+ Add Variable]                                         │
│                                                          │
│ ℹ Sensitive values are encrypted and hidden after save.  │
│   They're injected into your app but never shown in      │
│   logs or the UI.                                        │
└──────────────────────────────────────────────────────────┘
```

### Principle 6: Actionable Error Messages

Never show raw errors. Always tell the user what to DO.

```
❌ BAD:
"Error: ImagePullBackOff - Back-off pulling image registry.zenith.app/my-startup/user-svc:abc123"

✅ GOOD:
┌──────────────────────────────────────────────────────────┐
│ ⚠️  Deploy failed: Image not found                       │
│                                                          │
│ We couldn't pull the image for user-service.             │
│                                                          │
│ Possible causes:                                         │
│ • The build may have failed → [View Build Logs]          │
│ • The image was deleted from registry → [View Registry]  │
│ • Try redeploying → [Redeploy]                           │
└──────────────────────────────────────────────────────────┘

❌ BAD:
"Error: 0/3 nodes are available: 3 Insufficient memory."

✅ GOOD:
┌──────────────────────────────────────────────────────────┐
│ ⚠️  Not enough resources                                 │
│                                                          │
│ Your project needs more memory than is currently         │
│ available on your planets.                               │
│                                                          │
│ Used: 14.2 GB / 16 GB                                   │
│ Needed: 2 GB more                                        │
│                                                          │
│ Options:                                                 │
│ • [Add a Planet] to increase capacity                    │
│ • [Reduce app memory] in scaling settings                │
│ • [Scale down] unused apps                               │
└──────────────────────────────────────────────────────────┘
```

### Principle 7: Empty States That Guide

Every empty page should tell the user exactly what to do next.

```
❌ BAD:
┌──────────────────────────────────────────────────────────┐
│ Databases                                                │
│                                                          │
│ No databases found.                                      │
│                                                          │
└──────────────────────────────────────────────────────────┘

✅ GOOD:
┌──────────────────────────────────────────────────────────┐
│ Databases                                                │
│                                                          │
│     🗄️                                                   │
│     No databases yet                                     │
│                                                          │
│     Add a managed database for your apps.                │
│     PostgreSQL, MySQL, MongoDB, or Redis -               │
│     fully managed with automatic backups.                │
│                                                          │
│     [+ Create Database]                                  │
│                                                          │
│     💡 Tip: Database connection strings are              │
│     automatically added to linked apps.                  │
└──────────────────────────────────────────────────────────┘
```

### Principle 8: Confirmation for Destructive Actions Only

Don't ask "Are you sure?" for safe actions. Only for destructive ones.

```
Safe (no confirmation):
  - Create app, database, domain
  - Change environment variables
  - Scale up/down
  - Change branch
  - Redeploy

Needs confirmation (red modal):
  - Delete app
  - Delete database (type name to confirm)
  - Delete project (type name + "delete everything" to confirm)
  - Disconnect cloud connector
```

### Principle 9: Status Colors (Consistent Everywhere)

```
🟢 Green    = Running / Healthy / Active / Connected
🟡 Yellow   = Creating / Updating / Deploying / Building
🔴 Red      = Failed / Error / Crashed / Disconnected
⚪ Gray     = Stopped / Sleeping / Idle
🔵 Blue     = Info / Syncing / Pending
```

### Principle 10: One-Click Common Actions

The most common things a user does should be ONE click away.

```
App card → hover → quick actions appear:
┌────────────────────────────────────────────┐
│ user-service               🟢 Running      │
│ 3 instances · 245ms avg · 2.1k req/min    │
│                                            │
│ [↻ Redeploy] [📋 Logs] [⚙ Settings]      │
└────────────────────────────────────────────┘

Database card → hover → quick actions:
┌────────────────────────────────────────────┐
│ users-db                   🟢 Running      │
│ PostgreSQL 16 · 2.1GB / 5GB               │
│                                            │
│ [📋 Connection String] [💾 Backup Now]     │
└────────────────────────────────────────────┘
```

### Principle 11: No Dead Ends

Every page should have a clear next action. The user should never wonder "what now?"

```
After creating an app:
  → "Your app is deploying! While you wait, you can:"
  → [Add a custom domain]
  → [Set environment variables]
  → [Link a database]

After creating a database:
  → "Database is ready! Connect it to an app:"
  → [Link to app-name ▼]
  → [Copy connection string]

After deleting everything in a project:
  → "Project is empty. Start building:"
  → [Deploy an App]
  → [Create a Database]
```

### Principle 12: Keyboard-Friendly

```
Cmd+K / Ctrl+K  → Command palette (search anything)
Cmd+N           → New (context-aware: new app, new db, etc.)
Cmd+L           → Jump to logs
Cmd+/           → Keyboard shortcuts help
Esc             → Close modal/panel
```

### Principle 13: Mobile Responsive (Dashboard Only)

The project dashboard works on mobile for monitoring. Complex operations (create, configure) are desktop-only.

```
Mobile shows:
  ✅ Project overview (status of all apps)
  ✅ App status (running/crashed)
  ✅ Quick logs view
  ✅ Alert notifications
  ✅ One-tap redeploy

Mobile hides:
  ❌ Create wizards
  ❌ Environment variable editor
  ❌ GitOps settings
  ❌ Cloud connection setup
```
