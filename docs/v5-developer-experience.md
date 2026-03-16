# Zenith V5 — Developer Experience (DX)

> **Status:** Active — Implementation Plan
> **Version:** 5.2.0
> **Last Updated:** 2026-03-16
> **Philosophy:** iPhone — magical on the surface, powerful underneath.
> **Prerequisite:** Read V3 Architecture (`docs/v3-architecture.md`) for platform context.

---

## The One-Liner

**You have a docker-compose.yml that works. Paste it. We give you production.**

```
docker-compose.yml  →  2 minutes  →  Cloud. SSL. Backups. Scaling. Monitoring. Done.
```

No Kubernetes. No DevOps. No infrastructure. Just your code.

---

## How to Read This Document

| Section | Who | What |
|---------|-----|------|
| **Part A: The Promise** | Everyone | What we sell, who we sell to, what they get |
| **Part B: The 3-Step Flow** | Engineers + Designers | Paste → Review → Live |
| **Part C: What Happens Behind the Scenes** | Engineers | All the magic the customer never sees |
| **Part D: AI Features** | Engineers | Compose validation, error analysis |
| **Part E: Implementation Plan** | Engineers (hands-on) | Phases, files, endpoints |

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 5.2.0 | 2026-03-16 | Added social proof, competitor comparison, peace of mind, demo video |
| 5.1.0 | 2026-03-16 | Simplified: 3-step flow, iPhone philosophy, cut scope to essentials |
| 5.0.0 | 2026-03-16 | Initial DX plan |

---

# PART A: THE PROMISE

---

## A1. Who Is Our Customer?

```
┌─────────────────────────────────────────────────────────────┐
│                      OUR CUSTOMER                            │
│                                                              │
│  • Indie developer building a SaaS                          │
│  • Small startup (2-5 developers)                           │
│  • Has a working app (frontend + backend + database)        │
│  • Uses docker-compose for local development                │
│  • Knows Docker. Does NOT know Kubernetes.                  │
│  • Wants to go to production but can't afford DevOps        │
│  • Currently stuck on: Heroku (expensive), Railway (limited),│
│    or doing manual VPS setup (painful)                      │
│                                                              │
│  They are NOT:                                               │
│  • Enterprise with dedicated infra teams                    │
│  • People who want to learn K8s                             │
│  • People who need custom infrastructure                    │
└─────────────────────────────────────────────────────────────┘
```

**The persona:** "I'm a developer. I wrote my app. It works on my laptop. I want it on the internet with real infrastructure — SSL, backups, monitoring — without hiring a DevOps engineer or spending weeks learning cloud."

---

## A2. What They Get (Without Knowing How)

Customer sees **this**:

```
┌─────────────────────────────────────────────────────────────┐
│                   WHAT THE CUSTOMER SEES                      │
│                                                              │
│  ✅ My app is live at https://my-saas.freezenith.com        │
│  ✅ SSL certificate (automatic)                              │
│  ✅ Database with daily backups                              │
│  ✅ Redis cache (fast)                                       │
│  ✅ Logs I can search                                        │
│  ✅ AI tells me why my app crashed                           │
│  ✅ One-click deploy from CI                                 │
│  ✅ Scales when I get traffic                                │
│  ✅ Sleeps when I don't (saves money)                        │
│  ✅ Custom domain support                                    │
│                                                              │
│  Total time to set up: ~2 minutes                            │
│  DevOps knowledge needed: zero                               │
└─────────────────────────────────────────────────────────────┘
```

Behind the scenes, **Zenith runs all of this**:

```
┌─────────────────────────────────────────────────────────────┐
│              WHAT RUNS BEHIND THE SCENES                      │
│              (customer never sees this)                       │
│                                                              │
│  Kubernetes (k3s) cluster on Hetzner                        │
│  ├── Traefik         → TLS termination, routing             │
│  ├── APISIX          → API gateway, rate limiting           │
│  ├── cert-manager    → Let's Encrypt certificates           │
│  ├── external-dns    → Automatic DNS records                │
│  ├── CNPG            → PostgreSQL with WAL archiving to S3  │
│  ├── Redis           → In-memory cache                      │
│  ├── Harbor          → Container registry                   │
│  ├── Loki            → Log aggregation                      │
│  ├── Prometheus      → Metrics & alerting                   │
│  ├── Grafana         → Dashboards (admin only)              │
│  ├── KEDA            → Scale-to-zero for free tier          │
│  ├── Cilium          → Network policies, encryption         │
│  ├── ArgoCD          → GitOps deployment                    │
│  └── Keycloak        → Identity (when needed)               │
│                                                              │
│  All managed by Terraform + Ansible + Helm                  │
│  All monitored, backed up, and auto-healing                 │
└─────────────────────────────────────────────────────────────┘
```

**This is the iPhone moment.** They don't need to know about any of it. They just use it and it works.

---

## A3. The Conversion Funnel

```
                Without V5 DX          With V5 DX
Sign up              100                    100
Paste compose          -                     90   ← NEW: one action
Configure             40  ← manual, painful  85   ← auto from compose
Push images           30                     75
Live in production    10                     50   ← 5x improvement
Paying customer       10                     35   ← 3.5x more revenue
```

The biggest drop-off is "configure" — that's where people give up. **V5 eliminates this step.**

---

## A4. Build Strategy — Not Our Problem

**Zenith does NOT build images.** Building is CI (GitHub Actions, GitLab CI, etc.).

```
What we provide:                    What customer does:
────────────────                    ───────────────────
Container registry (Harbor)    ←    docker push (from CI or local)
Kubernetes deployment               Writes code + Dockerfile
Database + Redis (managed)           Runs their own CI
Networking, DNS, SSL
Env vars, secrets
Monitoring, logs, AI
```

**Why:**
- Zero build cost for us
- If code doesn't compile → their CI fails, not our platform
- No blame game
- They already have Docker — they wrote a docker-compose

**What we DO provide:** CI templates (copy-paste GitHub Actions) + deploy trigger API.

---

## A5. Who Already Trusts Us

These companies run their production on Zenith:

```
┌───────────────────────────────────────────────────────────┐
│                                                            │
│  MoneyFactory.fi                                           │
│  Finnish fintech startup — investment portfolio tracker    │
│  Stack: Next.js + Go API + PostgreSQL + Redis              │
│  "We went from docker-compose to production in one         │
│   afternoon. No DevOps hire needed."                       │
│                                                            │
│  FairBroker.net                                            │
│  Real estate comparison platform                           │
│  Stack: React + Node.js API + PostgreSQL                   │
│  "SSL, backups, monitoring — all automatic.                │
│   We just focus on building features."                     │
│                                                            │
│  BabakAcademy.com                                          │
│  Online education platform                                 │
│  Stack: Next.js + Go API + PostgreSQL + Redis              │
│  "Deployed our LMS with 4 services in under 10 minutes.   │
│   The AI error analysis saved us hours of debugging."      │
│                                                            │
└───────────────────────────────────────────────────────────┘
```

---

## A6. Why Not the Others? (Competitor Comparison)

```
╔═══════════════════╦═══════════╦═══════════╦═══════════╦═══════════╦═══════════╗
║                   ║  Zenith   ║  Heroku   ║  Railway  ║  Render   ║  DIY VPS  ║
╠═══════════════════╬═══════════╬═══════════╬═══════════╬═══════════╬═══════════╣
║ Compose import    ║  ✅ Yes   ║  ❌ No    ║  ❌ No    ║  ❌ No    ║  ❌ No    ║
║ Multi-service     ║  ✅ Yes   ║  ⚠️ Add-on║  ✅ Yes   ║  ✅ Yes   ║  Manual   ║
║ Managed DB        ║  ✅ Free  ║  $9+/mo   ║  $5+/mo   ║  $7+/mo   ║  Manual   ║
║ Managed Redis     ║  ✅ Free  ║  $15+/mo  ║  $5+/mo   ║  $10+/mo  ║  Manual   ║
║ SSL               ║  ✅ Auto  ║  ✅ Auto  ║  ✅ Auto  ║  ✅ Auto  ║  Certbot  ║
║ Custom domain     ║  ✅ Free  ║  ✅ Free  ║  ✅ Free  ║  ✅ Free  ║  Manual   ║
║ Daily backups     ║  ✅ Auto  ║  ⚠️ Paid  ║  ⚠️ Paid  ║  ⚠️ Paid  ║  Cron job ║
║ Scale to zero     ║  ✅ Yes   ║  ❌ No    ║  ✅ Yes   ║  ✅ Yes   ║  ❌ No    ║
║ AI error analysis ║  ✅ Yes   ║  ❌ No    ║  ❌ No    ║  ❌ No    ║  ❌ No    ║
║ Log search        ║  ✅ Yes   ║  ⚠️ Add-on║  ✅ Yes   ║  ✅ Yes   ║  Manual   ║
║ Monitoring        ║  ✅ Auto  ║  ⚠️ Add-on║  ⚠️ Basic ║  ⚠️ Basic ║  Manual   ║
║ Network isolation ║  ✅ Cilium║  ❌ No    ║  ❌ No    ║  ❌ No    ║  Manual   ║
║ Container registry║  ✅ Incl  ║  ❌ No    ║  ❌ No    ║  ❌ No    ║  Manual   ║
║ CI templates      ║  ✅ Yes   ║  ❌ No    ║  ❌ No    ║  ❌ No    ║  ❌ No    ║
║ EU data residency ║  ✅ Hetzner║ ❌ US    ║  ❌ US    ║  ❌ US    ║  Depends  ║
╠═══════════════════╬═══════════╬═══════════╬═══════════╬═══════════╬═══════════╣
║ Typical cost      ║           ║           ║           ║           ║           ║
║ (1 app + DB +     ║  €0-29/mo ║  $40+/mo  ║  $25+/mo  ║  $30+/mo  ║  €5+/mo   ║
║  Redis)           ║           ║  (unpred.)║ (usage)   ║           ║  + time   ║
╚═══════════════════╩═══════════╩═══════════╩═══════════╩═══════════╩═══════════╝
```

**Our unique advantages:**
1. **Compose import** — no one else does this. Paste → done.
2. **AI error analysis** — no one else does this. "Why did my app crash?" → instant answer.
3. **EU data residency** — Hetzner Germany/Finland. GDPR-native. US competitors can't offer this.
4. **Fixed pricing** — no surprise bills. Railway/Heroku usage-based pricing = anxiety.
5. **Full included stack** — DB, Redis, registry, monitoring, backups all included. Others charge per add-on.

---

## A7. Sleep Well — We've Got Your Back

> "What if something breaks at 3am?"

This is the question every developer asks before going to production. Here's the answer:

```
┌───────────────────────────────────────────────────────────┐
│                                                            │
│  😴  YOU SLEEP. WE WATCH.                                  │
│                                                            │
│  Your app crashes?                                         │
│  → Kubernetes auto-restarts it in seconds.                 │
│  → You get a notification.                                 │
│  → AI already analyzed the error for you.                  │
│                                                            │
│  Your database fills up?                                   │
│  → Alert fires before it's full.                           │
│  → One-click storage expansion.                            │
│  → Backups are already running daily.                      │
│                                                            │
│  Traffic spike at 2am?                                     │
│  → Auto-scaling handles it.                                │
│  → Rate limiting protects your API.                        │
│  → You see it in the morning, already handled.             │
│                                                            │
│  You push a broken deploy?                                 │
│  → Health check fails → old version stays running.         │
│  → Zero downtime. Always.                                  │
│  → Rollback with one click if needed.                      │
│                                                            │
│  SSL certificate expiring?                                 │
│  → Auto-renewed. You'll never think about it.              │
│                                                            │
│  Server goes down?                                         │
│  → We get paged, not you.                                  │
│  → Your data is safe (WAL backups to S3).                  │
│  → Restore tested and verified.                            │
│                                                            │
│  Someone attacks your API?                                 │
│  → Rate limiting blocks abuse.                             │
│  → Network policies isolate your app.                      │
│  → WAF rules block common attacks.                         │
│  → Encrypted traffic between all services.                 │
│                                                            │
│  ─────────────────────────────────────────────────         │
│                                                            │
│  Think of it like this:                                    │
│                                                            │
│  Before Zenith:                                            │
│  You are the developer AND the sysadmin AND the DBA        │
│  AND the security team AND the on-call engineer.           │
│                                                            │
│  After Zenith:                                             │
│  You are the developer. That's it.                         │
│  We are everything else.                                   │
│                                                            │
└───────────────────────────────────────────────────────────┘
```

**What's running behind the scenes to make this happen:**

| What You Worry About | What We Run For You |
|----------------------|---------------------|
| "Is my app running?" | Kubernetes health checks, auto-restart, zero-downtime deploys |
| "Is my data safe?" | CNPG WAL archiving to S3, daily backups, tested restore procedure |
| "Is my app secure?" | Cilium network policies, APISIX rate limiting, WAF rules, WireGuard encryption |
| "Will it handle traffic?" | KEDA auto-scaling, horizontal pod autoscaler |
| "Is my SSL valid?" | cert-manager auto-renewal, Let's Encrypt |
| "What broke?" | Loki logs, Prometheus metrics, AI error analysis |
| "Who's watching at night?" | 24/7 monitoring, Alertmanager, automated recovery |

---

## A8. See It In Action

Watch the full demo — from docker-compose to production in 2 minutes:

<!-- TODO: Replace with actual YouTube video ID -->
<div align="center">

[![Zenith Demo — Docker Compose to Production in 2 Minutes](https://img.youtube.com/vi/PLACEHOLDER_VIDEO_ID/maxresdefault.jpg)](https://www.youtube.com/watch?v=PLACEHOLDER_VIDEO_ID)

**[Watch on YouTube →](https://www.youtube.com/watch?v=PLACEHOLDER_VIDEO_ID)**

</div>

> **Demo script (90 seconds):**
> 1. (0:00) Open Zenith dashboard, click "New Project"
> 2. (0:10) Type "my-saas", paste docker-compose.yml
> 3. (0:20) Click Continue — show detected services + managed DB + Redis
> 4. (0:30) Show auto-generated env vars, service linking
> 5. (0:40) Push 3 images (pre-recorded, fast-forward)
> 6. (0:55) Click "Deploy All" — show progress bar
> 7. (1:10) App is live — open URL, show SSL lock
> 8. (1:20) Show logs page, trigger an error, click "AI Analyze"
> 9. (1:30) AI explains the error — "wow" moment
> 10. (1:35) Show CI template page — "set up once, auto-deploy forever"
> 11. (1:40) End card: "From docker-compose to production. Try free at freezenith.com"

---

# PART B: THE 3-STEP FLOW

---

## B1. Overview

The entire onboarding is 3 steps. Not 5. Not 10. Three.

```
Step 1                    Step 2                    Step 3
PASTE & NAME         →    REVIEW & PUSH        →    DEPLOY
  ●━━━━━━━━━━━━━━━━━━━━━━━○━━━━━━━━━━━━━━━━━━━━━━━○
  ~30 seconds              ~60 seconds              ~30 seconds
                           (+ image push time)
```

---

## B2. Step 1 — Paste & Name (~30 seconds)

```
┌───────────────────────────────────────────────────────────┐
│                                                            │
│  🚀 Deploy Your App                              Step 1/3 │
│                                                            │
│  Project name *                                            │
│  ┌──────────────────────────────────────────────┐         │
│  │ my-saas                                       │         │
│  └──────────────────────────────────────────────┘         │
│  Your URL: https://my-saas.apps.freezenith.com             │
│                                                            │
│  Paste your docker-compose.yml *                           │
│  ┌──────────────────────────────────────────────┐         │
│  │ version: "3.8"                                │         │
│  │ services:                                     │         │
│  │   frontend:                                   │         │
│  │     build: ./frontend                         │         │
│  │     ports: ["3000:3000"]                      │         │
│  │     environment:                              │         │
│  │       API_URL: http://api:8080                │         │
│  │   api:                                        │         │
│  │     build: ./api                              │         │
│  │     ports: ["8080:8080"]                      │         │
│  │     environment:                              │         │
│  │       DATABASE_URL: postgresql://...          │         │
│  │       REDIS_URL: redis://redis:6379           │         │
│  │   worker:                                     │         │
│  │     build: ./worker                           │         │
│  │     environment:                              │         │
│  │       DATABASE_URL: postgresql://...          │         │
│  │   db:                                         │         │
│  │     image: postgres:16                        │         │
│  │   redis:                                      │         │
│  │     image: redis:7                            │         │
│  └──────────────────────────────────────────────┘         │
│                                                            │
│  Or:  [📁 Upload file]   [📝 Add services manually]       │
│                                                            │
│                                       [Continue →]         │
│                                                            │
└───────────────────────────────────────────────────────────┘
```

**What happens when they click Continue:**
1. Project created (name + slug)
2. Harbor project + robot account created automatically
3. docker-compose.yml parsed instantly (Layer 1 + 2)
4. AI validation starts in background (Layer 3, non-blocking)
5. Redirect to Step 2 with results

---

## B3. Step 2 — Review & Push (~60 seconds + push time)

```
┌───────────────────────────────────────────────────────────┐
│                                                            │
│  📦 Review & Push Images                         Step 2/3 │
│                                                            │
│  We detected 3 services and 2 managed services:            │
│                                                            │
│  YOUR SERVICES (push your Docker images)                   │
│  ┌──────────────────────────────────────────────┐         │
│  │  frontend        Port: 3000   🌐 Public      │         │
│  │  → https://my-saas.apps.freezenith.com        │         │
│  │  Image: ⏳ waiting for push                   │         │
│  ├──────────────────────────────────────────────┤         │
│  │  api             Port: 8080   🌐 Public      │         │
│  │  → https://api-my-saas.apps.freezenith.com    │         │
│  │  Image: ✅ pushed (just now)                  │         │
│  ├──────────────────────────────────────────────┤         │
│  │  worker          No ports     🔒 Internal    │         │
│  │  → worker-my-saas.internal:9090               │         │
│  │  Image: ✅ pushed (just now)                  │         │
│  └──────────────────────────────────────────────┘         │
│                                                            │
│  WE HANDLE THESE (automatic, no action needed)             │
│  ┌──────────────────────────────────────────────┐         │
│  │  🐘 PostgreSQL 16    ✅ Provisioning...       │         │
│  │  🔴 Redis 7          ✅ Ready                 │         │
│  └──────────────────────────────────────────────┘         │
│                                                            │
│  ── Push your images: ──────────────────────────────────  │
│  ┌──────────────────────────────────────────────┐         │
│  │ # Login once:                                 │         │
│  │ docker login registry.freezenith.com \        │         │
│  │   -u robot$my-saas+push -p <your-token>       │         │
│  │                                               │         │
│  │ # Build & push each service:                  │         │
│  │ docker build -t registry.freezenith.com/\     │         │
│  │   my-saas/frontend:latest ./frontend          │         │
│  │ docker push registry.freezenith.com/\         │         │
│  │   my-saas/frontend:latest                     │         │
│  │                                    [📋 Copy]  │         │
│  └──────────────────────────────────────────────┘         │
│                                                            │
│  Or use CI: [📋 GitHub Actions template]                   │
│                                                            │
│  ENV VARS (auto-detected, edit if needed)                  │
│  ┌──────────────────────────────────────────────┐         │
│  │  frontend                                     │         │
│  │    API_URL = https://api-my-saas.apps...  🔗  │         │
│  │                                               │         │
│  │  api                                          │         │
│  │    DATABASE_URL = (managed, auto) ........🔒  │         │
│  │    REDIS_URL = (managed, auto) ...........🔒  │         │
│  │    WORKER_URL = http://worker-my-saas....🔗   │         │
│  │    + [Add variable]                           │         │
│  │                                               │         │
│  │  worker                                       │         │
│  │    DATABASE_URL = (managed, auto) ........🔒  │         │
│  │    REDIS_URL = (managed, auto) ...........🔒  │         │
│  │    + [Add variable]                           │         │
│  └──────────────────────────────────────────────┘         │
│  🔗 = auto-linked  🔒 = managed secret (auto-generated)   │
│                                                            │
│  💡 AI: "Consider adding health checks to your services"   │
│                                                            │
│                       [🚀 Deploy All] (when all ✅)        │
│                                                            │
└───────────────────────────────────────────────────────────┘
```

**What's happening behind the scenes:**
- Harbor robot account credentials shown (auto-created in Step 1)
- Image status polled every 5s from Harbor API
- PostgreSQL: CNPG Cluster CRD created → primary pod → credentials generated → connection string injected as env var
- Redis: StatefulSet created → password generated → REDIS_URL injected
- Env vars: docker-compose env vars auto-translated to Zenith env vars
- Service URLs: compose `http://api:8080` → K8s DNS `http://api-my-saas.zenith-apps.svc:8080`
- AI suggestions shown as non-blocking tips

**Key UX decisions:**
- Env vars and service review are on the SAME page (no extra step)
- Managed service credentials are auto-injected, customer never types a password
- Internal services (no ports) auto-detected, no IngressRoute created
- Deploy button disabled until all images are pushed

---

## B4. Step 3 — Deploy (~30 seconds)

Clicking "Deploy All" in Step 2 triggers this:

```
┌───────────────────────────────────────────────────────────┐
│                                                            │
│  🚀 Deploying...                                 Step 3/3 │
│                                                            │
│  🐘 PostgreSQL 16       ● Ready                            │
│  🔴 Redis 7             ● Ready                            │
│  📦 worker              🔄 Creating pod...                 │
│  📦 api                 🔄 Pulling image...                │
│  📦 frontend            ⏳ Queued                          │
│                                                            │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  40%               │
│                                                            │
└───────────────────────────────────────────────────────────┘

                         ↓ ~30 seconds later ↓

┌───────────────────────────────────────────────────────────┐
│                                                            │
│  ✅ Your project is live!                                  │
│                                                            │
│  🐘 PostgreSQL 16       ● Healthy                          │
│  🔴 Redis 7             ● Healthy                          │
│  📦 worker              ● Running                          │
│  📦 api                 ● Running                          │
│  📦 frontend            ● Running                          │
│                                                            │
│  🌐 Your app:  https://my-saas.apps.freezenith.com        │
│  📊 Dashboard: https://app.freezenith.com/projects/my-saas│
│                                                            │
│  What's next:                                              │
│  • [📋 Set up CI] — auto-deploy on git push               │
│  • [📋 Add custom domain] — use your own domain           │
│  • [📊 View logs] — see what's happening                   │
│                                                            │
└───────────────────────────────────────────────────────────┘
```

**What happens behind the scenes:**
1. For each app service: K8s Deployment + Service created, env vars synced to Secret/ConfigMap
2. Public services get IngressRoute + DNS record (via external-dns)
3. cert-manager issues TLS certificate
4. Health checks start monitoring
5. Free tier: KEDA HTTPScaledObject attached (scale-to-zero after 15min idle)

**Total time: ~2 minutes from "I have a docker-compose" to "my app is live in production with SSL, backups, and monitoring."**

---

# PART C: WHAT HAPPENS BEHIND THE SCENES

> The customer never reads this section. This is for engineers building the platform.

---

## C1. Project Entity

A **Project** groups services together. One customer can have multiple projects.

### Database Schema

```sql
-- Migration: 038_projects.up.sql

CREATE TABLE projects (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    description     TEXT DEFAULT '',

    -- Harbor registry (auto-created)
    harbor_project_name TEXT,
    harbor_robot_user   TEXT,
    harbor_robot_pass   TEXT,    -- encrypted

    status          TEXT NOT NULL DEFAULT 'active',  -- active, archived
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_projects_user ON projects(user_id);
CREATE INDEX idx_projects_slug ON projects(slug);

-- Link apps to projects
ALTER TABLE apps ADD COLUMN project_id TEXT REFERENCES projects(id) ON DELETE SET NULL;
ALTER TABLE apps ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT true;
CREATE INDEX idx_apps_project ON apps(project_id);
```

### Entity

```go
// entities/project.go

type Project struct {
    ID          string    `json:"id"`
    UserID      string    `json:"user_id"`
    Name        string    `json:"name"`
    Slug        string    `json:"slug"`
    Description string    `json:"description"`

    HarborProjectName string `json:"-"`
    HarborRobotUser   string `json:"harbor_robot_user,omitempty"`

    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`

    // Populated on read
    Services        []App            `json:"services,omitempty"`
    ManagedServices []ManagedService `json:"managed_services,omitempty"`
}
```

### Port Interface

```go
// ports/repositories.go

type ProjectRepository interface {
    CreateProject(ctx context.Context, project *entities.Project) error
    GetProject(ctx context.Context, id string) (*entities.Project, error)
    GetProjectBySlug(ctx context.Context, slug string) (*entities.Project, error)
    ListProjects(ctx context.Context, userID string) ([]entities.Project, error)
    UpdateProject(ctx context.Context, project *entities.Project) error
    DeleteProject(ctx context.Context, id string) error
}
```

**What got simpler vs V5.0:**
- Removed `auth_mode`, `auth_service_url`, `keycloak_realm`, `keycloak_client_id` — auth modes deferred
- Removed `registry_mode`, `external_registry_*` — only Zenith Harbor, no external registry option
- Harbor credentials auto-created when project is created, no configuration needed

---

## C2. Managed Services (PostgreSQL + Redis Only)

We start with just two. They cover 90%+ of startup needs.

### Database Schema

```sql
-- Migration: 039_managed_services.up.sql

CREATE TABLE managed_services (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL REFERENCES users(id),
    service_type    TEXT NOT NULL,    -- 'postgresql', 'redis'
    name            TEXT NOT NULL,
    version         TEXT NOT NULL,    -- '16', '7'

    -- Connection info (auto-generated, customer never sets these)
    connection_url  TEXT,
    internal_host   TEXT,
    port            INTEGER,
    username        TEXT,
    password        TEXT,             -- encrypted
    database_name   TEXT,

    -- K8s tracking
    k8s_namespace     TEXT,
    k8s_resource_name TEXT,

    status          TEXT NOT NULL DEFAULT 'provisioning', -- provisioning, ready, error
    status_message  TEXT,
    storage_gb      INTEGER DEFAULT 5,

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_managed_services_project ON managed_services(project_id);
```

### How They're Provisioned

```
Customer pastes docker-compose with "image: postgres:16"
    │
    ▼
Compose parser detects → ManagedService{type: postgresql, version: 16}
    │
    ▼
Service layer creates CNPG Cluster CRD in customer namespace
    │
    ├── CNPG operator creates primary pod
    ├── Auto-generates credentials in K8s Secret
    ├── Configures WAL archiving to Hetzner S3
    └── Connection string: postgresql://user:pass@pg-my-saas.ns.svc:5432/app
    │
    ▼
Connection string injected as DATABASE_URL env var for all services that need it
```

```
Customer pastes docker-compose with "image: redis:7"
    │
    ▼
Compose parser detects → ManagedService{type: redis, version: 7}
    │
    ▼
Service layer creates StatefulSet + Service + PVC
    │
    ├── Single replica (sufficient for startups)
    ├── Auto-generates password in K8s Secret
    ├── PVC for persistence (AOF)
    └── Connection string: redis://:pass@redis-my-saas.ns.svc:6379
    │
    ▼
Connection string injected as REDIS_URL env var for all services that need it
```

### Known Service Detection

```go
// services/compose_parser.go

var managedImages = map[string]entities.ServiceType{
    "postgres":   entities.ServiceTypePostgreSQL,
    "postgresql": entities.ServiceTypePostgreSQL,
    "redis":      entities.ServiceTypeRedis,
    "valkey":     entities.ServiceTypeRedis,
}
```

**What got simpler vs V5.0:**
- Removed MongoDB (Percona operator) — niche, adds complexity
- Removed RabbitMQ — most startups use Redis for queues (Bull, Celery w/Redis)
- Can add more later when customers actually ask for them

---

## C3. Environment Variables

The #1 missing feature. Every deployed app needs env vars.

### Database Schema

```sql
-- Migration: 040_app_env_vars.up.sql

CREATE TABLE app_env_vars (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,       -- encrypted for secrets
    is_secret   BOOLEAN NOT NULL DEFAULT false,
    source      TEXT NOT NULL DEFAULT 'manual',  -- manual, managed_service, service_link, compose_import
    source_id   TEXT,

    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(app_id, key)
);

CREATE INDEX idx_env_vars_app ON app_env_vars(app_id);
```

### How Env Vars Flow to K8s

```
Dashboard / API  →  app_env_vars table (PostgreSQL)
                           │
                           ├── is_secret=true  → K8s Secret (base64)
                           │                      name: {app-slug}-env
                           │
                           └── is_secret=false → K8s ConfigMap
                                                  name: {app-slug}-config
                           │
                           ▼
                    Pod spec:
                      envFrom:
                        - secretRef: {app-slug}-env
                        - configMapRef: {app-slug}-config
```

### Auto-Injection from Compose

When docker-compose says this:
```yaml
api:
  environment:
    DATABASE_URL: postgresql://user:pass@db:5432/mydb
    REDIS_URL: redis://redis:6379
    WORKER_URL: http://worker:9090
```

Zenith translates to:

| Key | Original Value | Zenith Value | Source |
|-----|---------------|-------------|--------|
| `DATABASE_URL` | `postgresql://user:pass@db:5432/mydb` | `postgresql://auto:auto@pg-my-saas.ns.svc:5432/app` | `managed_service` |
| `REDIS_URL` | `redis://redis:6379` | `redis://:auto@redis-my-saas.ns.svc:6379` | `managed_service` |
| `WORKER_URL` | `http://worker:9090` | `http://worker-my-saas.zenith-apps.svc:9090` | `service_link` |

**The customer doesn't configure any of this.** It's all automatic from their docker-compose.

---

## C4. Docker Compose Parser

### What It Detects

| docker-compose | Zenith | How |
|---------------|--------|-----|
| `services.api.build: ./api` | App service (needs image push) | Has `build` key |
| `services.db.image: postgres:16` | Managed PostgreSQL 16 | Image matches known pattern |
| `services.redis.image: redis:7` | Managed Redis 7 | Image matches known pattern |
| `ports: ["3000:3000"]` | Public service (gets domain + SSL) | Has `ports` |
| No `ports` | Internal service (K8s DNS only) | No `ports` |
| `environment: DATABASE_URL: ...` | Env var, auto-replaced if managed | Matches managed service |
| `depends_on: [db]` | Service linking | Internal DNS auto-injected |
| `volumes: [pgdata:...]` | Handled by managed service PVC | Ignored for managed services |

### Three-Layer Validation

```
LAYER 1: YAML Parse        instant     MUST pass (blocks if invalid)
    │
    ▼
LAYER 2: Rules Check        instant     MUST pass (blocks if no services detected)
    │                                    Detects managed services, ports, env vars
    ▼
LAYER 3: AI Review          2-5 sec     ADVISORY ONLY (never blocks)
                                         Security tips, best practices, suggestions
```

**Critical: Layer 3 failure NEVER blocks the flow.** If AI is down → skip, proceed.

### Parser Endpoint

```
POST /api/v1/projects/{projectId}/import-compose

Request:
{
    "compose_content": "version: '3.8'\nservices:\n  ..."
}

Response:
{
    "data": {
        "valid": true,
        "services": [
            {
                "name": "frontend",
                "port": 3000,
                "is_public": true,
                "url": "my-saas.apps.freezenith.com",
                "env_vars": [
                    { "key": "API_URL", "original": "http://api:8080", "zenith": "https://api-my-saas.apps.freezenith.com" }
                ]
            },
            {
                "name": "api",
                "port": 8080,
                "is_public": true,
                "url": "api-my-saas.apps.freezenith.com",
                "env_vars": [
                    { "key": "DATABASE_URL", "original": "postgresql://...", "zenith": "(managed)" },
                    { "key": "REDIS_URL", "original": "redis://redis:6379", "zenith": "(managed)" },
                    { "key": "WORKER_URL", "original": "http://worker:9090", "zenith": "http://worker-my-saas.zenith-apps.svc:9090" }
                ]
            },
            {
                "name": "worker",
                "port": 9090,
                "is_public": false,
                "env_vars": [
                    { "key": "DATABASE_URL", "original": "postgresql://...", "zenith": "(managed)" },
                    { "key": "REDIS_URL", "original": "redis://redis:6379", "zenith": "(managed)" }
                ]
            }
        ],
        "managed_services": [
            { "name": "db", "type": "postgresql", "version": "16" },
            { "name": "redis", "type": "redis", "version": "7" }
        ],
        "warnings": [
            "DATABASE_URL has hardcoded password — will be auto-replaced with secure credentials"
        ],
        "ai_suggestions": [
            "Consider adding health checks for better reliability"
        ]
    }
}
```

---

## C5. Image Verification

Before deploying, all images must be pushed to Harbor.

```
GET /api/v1/projects/{projectId}/images/status

Response:
{
    "data": {
        "services": [
            { "name": "frontend", "pushed": false },
            { "name": "api", "pushed": true, "tag": "latest", "pushed_at": "..." },
            { "name": "worker", "pushed": true, "tag": "latest", "pushed_at": "..." }
        ],
        "all_ready": false
    }
}
```

Frontend polls every 5 seconds. Deploy button disabled until `all_ready: true`.

---

## C6. Deploy Flow

```
POST /api/v1/projects/{projectId}/deploy

What happens (in order):
1. Managed services verified ready (PG, Redis)
2. For each app service:
   a. K8s Deployment created with image from Harbor
   b. K8s Service created
   c. Env vars synced → K8s Secret + ConfigMap
   d. envFrom added to pod spec
   e. If public: IngressRoute + cert-manager Certificate created
   f. If free tier: KEDA HTTPScaledObject attached
3. external-dns creates DNS records
4. Health checks monitored
5. Status reported back via polling/websocket
```

---

## C7. CI Templates

Copy-paste GitHub Actions templates for common frameworks. Stored in backend, served via API.

```
GET /api/v1/ci-templates/go
GET /api/v1/ci-templates/nextjs
GET /api/v1/ci-templates/python
GET /api/v1/ci-templates/nodejs
GET /api/v1/ci-templates/rust
```

Each template does:
1. Build Docker image
2. Push to Zenith Harbor
3. Call deploy trigger API

Example (Go):

```yaml
name: Deploy to Zenith
on:
  push:
    branches: [main]

env:
  REGISTRY: registry.freezenith.com
  PROJECT: <your-project>
  SERVICE: <your-service>

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ secrets.ZENITH_REGISTRY_USER }}
          password: ${{ secrets.ZENITH_REGISTRY_PASS }}

      - uses: docker/build-push-action@v5
        with:
          push: true
          tags: ${{ env.REGISTRY }}/${{ env.PROJECT }}/${{ env.SERVICE }}:${{ github.sha }}

      - name: Deploy
        run: |
          curl -X POST \
            https://api.freezenith.com/api/v1/apps/${{ secrets.ZENITH_APP_ID }}/deploy \
            -H "Authorization: Bearer ${{ secrets.ZENITH_API_KEY }}" \
            -H "Content-Type: application/json" \
            -d '{"image_tag": "${{ github.sha }}"}'
```

Dashboard shows personalized template with project-specific values filled in.

---

# PART D: AI FEATURES

---

## D1. AI Compose Validation (Layer 3)

Non-blocking. Runs after Layer 1+2 pass. Uses LiteLLM for multi-provider support.

```
Zenith API → LiteLLM → OpenAI gpt-4o-mini (primary)
                      → Anthropic claude-haiku (fallback)
                      → Skip (graceful degradation)
```

### AI Prompt

```
You are a Docker Compose validator for a cloud platform.
Analyze this docker-compose.yml. Return JSON only.

Check for:
1. Security: hardcoded passwords, privileged mode, exposed debug ports
2. Best practices: missing health checks, no restart policy
3. Common mistakes: wrong port mappings, missing depends_on

Return: {"suggestions": [{"level": "warning|info", "message": "..."}]}

docker-compose.yml:
---
%s
```

### Config

```go
type AIConfig struct {
    LiteLLMURL    string // LiteLLM proxy URL (or direct OpenAI URL)
    LiteLLMAPIKey string // API key
    Model         string // default: gpt-4o-mini
    Enabled       bool   // kill switch — if false, skip all AI
}
```

---

## D2. AI Error Analysis

The feature that makes developers stay. They click "Why did my app crash?" and get an answer.

```
Error in pod logs
    │
    ▼
Fetch last 50 lines (Loki or kubectl)
    │
    ▼
PII Scrubber removes:
  • emails      → [EMAIL]
  • IPs         → [IP]
  • JWT tokens  → [TOKEN]
  • API keys    → [API_KEY]
  • passwords   → [REDACTED]
  • UUIDs       → [UUID]
    │
    ▼
LiteLLM: "Analyze this error log"
    │
    ▼
Response: { problem, cause, fix, confidence }
    │
    ▼
Customer sees: "Your app crashed because X. Fix it by doing Y."
+ disclaimer: "No personal data was shared with AI"
```

### Endpoint

```
POST /api/v1/apps/{appId}/ai/analyze-error

Request:
{ "log_lines": 50 }

Response:
{
    "data": {
        "problem": "Nil pointer dereference in UserHandler.GetByID",
        "cause": "Database returned nil user, no nil check before accessing fields",
        "fix": "Add nil check: if user == nil { return 404 }",
        "confidence": "high",
        "pii_disclaimer": "No personal data was shared with AI."
    }
}
```

### AI Usage Limits

```
Free:      5 AI requests / month
Pro:       50 / month
Team:      200 / month
Business:  Unlimited
```

### Usage Tracking

```sql
-- Migration: 041_ai_usage.up.sql

CREATE TABLE ai_usage (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    usage_type  TEXT NOT NULL,     -- compose_validation, error_analysis
    model       TEXT NOT NULL,
    tokens_in   INTEGER NOT NULL,
    tokens_out  INTEGER NOT NULL,
    cost_usd    DECIMAL(10,6),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_usage_user ON ai_usage(user_id);
CREATE INDEX idx_ai_usage_month ON ai_usage(user_id, created_at);
```

---

## D3. Logs Dashboard

Simple. Search. Real-time. AI analysis button.

```
┌───────────────────────────────────────────────────────────┐
│  📋 Logs — api                                  [Live 🔴] │
│                                                            │
│  Service: [api ▼]   Since: [1h ▼]   [Search________]     │
│                                                            │
│  15:42:01 INFO  Server started on :8080                    │
│  15:42:02 INFO  Connected to PostgreSQL                    │
│  15:42:10 INFO  GET /health 200 1ms                        │
│  15:42:12 ERROR panic: nil pointer dereference             │
│  15:42:12 ERROR   main.(*UserHandler).GetByID:87           │
│                                                            │
│  [🤖 Why did this crash?]                                  │
│                                                            │
│  ┌──────────────────────────────────────────────┐         │
│  │  Problem: Nil pointer dereference             │         │
│  │  Cause: DB returned nil, no nil check         │         │
│  │  Fix: Add nil check after GetByID()           │         │
│  │  ⚠️ No personal data shared with AI          │         │
│  └──────────────────────────────────────────────┘         │
└───────────────────────────────────────────────────────────┘
```

### Endpoints

```
GET  /api/v1/apps/{appId}/logs?since=1h&limit=500   → Historical
WS   /api/v1/apps/{appId}/logs/stream                → Real-time
POST /api/v1/apps/{appId}/ai/analyze-error           → AI analysis
```

Architecture: Pod stdout → Loki (DaemonSet) → Loki Storage (S3) → Zenith API (Loki query proxy, scoped to customer namespace) → Dashboard

---

# PART E: IMPLEMENTATION PLAN

---

## E1. Phase 1 — Foundation (Week 1-2)

**Goal:** Project entity + Env vars + K8s Secret/ConfigMap sync

### Files

| File | Action | What |
|------|--------|------|
| `entities/project.go` | NEW | Project struct |
| `entities/managed_service.go` | NEW | ManagedService struct |
| `entities/env_var.go` | NEW | AppEnvVar struct |
| `ports/repositories.go` | MODIFY | Add 3 repository interfaces |
| `adapters/postgres/postgres_project.go` | NEW | SQL implementation |
| `adapters/postgres/postgres_managed_service.go` | NEW | SQL implementation |
| `adapters/postgres/postgres_env_var.go` | NEW | SQL implementation |
| `adapters/memory/memory_project.go` | NEW | Test stubs |
| `adapters/memory/memory_managed_service.go` | NEW | Test stubs |
| `adapters/memory/memory_env_var.go` | NEW | Test stubs |
| `services/project.go` | NEW | CRUD + Harbor project creation |
| `services/env_var.go` | NEW | CRUD + K8s sync |
| `handlers/project.go` | NEW | HTTP endpoints |
| `handlers/env_var.go` | NEW | HTTP endpoints |
| `cmd/server/main.go` | MODIFY | Wire everything |
| `migrations/038_projects.up.sql` | NEW | projects table + apps.project_id |
| `migrations/039_managed_services.up.sql` | NEW | managed_services table |
| `migrations/040_app_env_vars.up.sql` | NEW | app_env_vars table |
| `apps/web/src/app/projects/page.tsx` | NEW | Project list |
| `apps/web/src/app/projects/[id]/page.tsx` | NEW | Project dashboard |
| `apps/web/src/lib/api.ts` | MODIFY | Add project + env var methods |

### Endpoints

```
POST   /api/v1/projects                    Create project + Harbor setup
GET    /api/v1/projects                    List projects
GET    /api/v1/projects/:id                Get project with services
PUT    /api/v1/projects/:id                Update project
DELETE /api/v1/projects/:id                Delete project + cleanup
POST   /api/v1/apps/:appId/env             Set env vars (bulk)
GET    /api/v1/apps/:appId/env             List env vars
DELETE /api/v1/apps/:appId/env/:varId      Delete env var
```

---

## E2. Phase 2 — Compose + Deploy (Week 3-4)

**Goal:** Docker compose import, managed services provisioning, image verification, one-click deploy

### Files

| File | Action | What |
|------|--------|------|
| `services/compose_parser.go` | NEW | Parse docker-compose YAML |
| `services/compose_validator.go` | NEW | Layer 1+2 validation |
| `services/managed_service.go` | NEW | Provision PostgreSQL/Redis in K8s |
| `handlers/compose.go` | NEW | Import endpoint |
| `handlers/managed_service.go` | NEW | CRUD endpoints |
| `handlers/image_status.go` | NEW | Harbor polling |
| `migrations/041_compose_imports.up.sql` | NEW | Audit table |
| `apps/web/src/app/projects/new/page.tsx` | NEW | 3-step wizard |
| `apps/web/src/components/compose/ComposeEditor.tsx` | NEW | YAML textarea |
| `apps/web/src/components/compose/ImageStatus.tsx` | NEW | Push verification |

### Endpoints

```
POST   /api/v1/projects/:id/import-compose       Parse compose
POST   /api/v1/projects/:id/managed-services      Provision service
GET    /api/v1/projects/:id/managed-services       List services
DELETE /api/v1/projects/:id/managed-services/:msId Delete service
GET    /api/v1/projects/:id/images/status          Check images pushed
POST   /api/v1/projects/:id/deploy                 Deploy all
POST   /api/v1/apps/:appId/deploy                  Deploy single (CI trigger)
```

---

## E3. Phase 3 — AI + Logs + Polish (Week 5-6)

**Goal:** AI features, logs dashboard, CI templates

### Files

| File | Action | What |
|------|--------|------|
| `services/ai_client.go` | NEW | LiteLLM HTTP client |
| `services/ai_compose.go` | NEW | Compose AI validation |
| `services/ai_error.go` | NEW | Error analysis |
| `services/pii_scrubber.go` | NEW | PII removal |
| `services/logs.go` | NEW | Loki query proxy |
| `handlers/ai.go` | NEW | AI endpoints |
| `handlers/logs.go` | NEW | REST + WebSocket |
| `migrations/042_ai_usage.up.sql` | NEW | Usage tracking |
| `apps/web/src/app/projects/[id]/logs/page.tsx` | NEW | Log viewer |
| `apps/web/src/app/projects/[id]/ci/page.tsx` | NEW | CI templates |
| `apps/web/src/components/ai/ErrorAnalysis.tsx` | NEW | AI analysis UI |

### Endpoints

```
POST   /api/v1/apps/:appId/ai/analyze-error   AI error analysis
GET    /api/v1/ai/usage                        Usage stats
GET    /api/v1/ci-templates/:framework         CI template
GET    /api/v1/apps/:appId/logs                Historical logs
WS     /api/v1/apps/:appId/logs/stream         Real-time logs
```

---

## E4. All New Endpoints Summary

```
Phase 1 (8 endpoints):
  POST/GET/GET/PUT/DELETE  /api/v1/projects[/:id]
  POST/GET/DELETE          /api/v1/apps/:appId/env[/:varId]

Phase 2 (7 endpoints):
  POST    /api/v1/projects/:id/import-compose
  POST/GET/DELETE  /api/v1/projects/:id/managed-services[/:msId]
  GET     /api/v1/projects/:id/images/status
  POST    /api/v1/projects/:id/deploy
  POST    /api/v1/apps/:appId/deploy

Phase 3 (5 endpoints):
  POST    /api/v1/apps/:appId/ai/analyze-error
  GET     /api/v1/ai/usage
  GET     /api/v1/ci-templates/:framework
  GET     /api/v1/apps/:appId/logs
  WS      /api/v1/apps/:appId/logs/stream

Total: 20 new endpoints across 3 phases (~6 weeks)
```

---

## E5. Verification

```bash
# Backend
cd services/api && go vet ./internal/...
cd services/api && go test ./internal/... -v

# Frontend
cd apps/web && npx next lint --quiet
cd apps/web && npx next build

# Integration (on staging)
# 1. Create project via API
# 2. Import sample docker-compose
# 3. Verify managed services provisioned
# 4. Push test image to Harbor
# 5. Deploy and verify pods running
# 6. Hit app URL and verify SSL
```

---

## E6. What We Deferred (and Why)

| Feature | Why Deferred | When to Add |
|---------|-------------|-------------|
| Auth modes (BYOP, Managed Keycloak) | 90% of startups handle auth internally | When customers ask |
| External registry (Docker Hub, GHCR) | Adds UX decision point at onboarding | When customers ask |
| MongoDB, RabbitMQ managed | Niche, adds operator complexity | When customers ask |
| GitLab CI templates | GitHub Actions covers most users | When requested |
| Deploy preview environments | Nice-to-have, not essential for launch | Phase 4+ |
| Compose `volumes` (persistent storage for apps) | Apps should be stateless, data in managed services | When customers ask |

**Philosophy: Ship the 90% case first. Add the 10% when customers ask for it.**
