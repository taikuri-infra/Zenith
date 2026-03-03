# 21 — Local Development Setup

> **Purpose:** Get a new developer from zero to running the entire Zenith platform locally in under 30 minutes.
> **Audience:** Any developer joining the team for the first time.
> **Last Updated:** 2026-03-03
> **Related:** [SYSTEM-MAP.md](./SYSTEM-MAP.md) (system overview), [10-backend-architecture.md](./10-backend-architecture.md) (Go code structure), [23-frontend-architecture.md](./23-frontend-architecture.md) (Next.js apps)

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Repository Setup](#2-repository-setup)
3. [Architecture: What Runs Where](#3-architecture-what-runs-where)
4. [Option A: Docker Compose (Recommended for First Day)](#4-option-a-docker-compose-recommended-for-first-day)
5. [Option B: Local Processes (For Active Development)](#5-option-b-local-processes-for-active-development)
6. [Connecting to Staging Cluster](#6-connecting-to-staging-cluster)
7. [Environment Variables Reference](#7-environment-variables-reference)
8. [IDE Setup](#8-ide-setup)
9. [Common Development Workflows](#9-common-development-workflows)
10. [Troubleshooting](#10-troubleshooting)

---

## 1. Prerequisites

Install these tools before starting:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    REQUIRED TOOLS                                        │
│                                                                          │
│  Tool              Version    Purpose              Install               │
│  ────              ───────    ───────              ───────               │
│  Node.js           20+        Frontend apps        brew install node     │
│  pnpm              10.29+     Package manager      npm install -g pnpm   │
│  Go                1.23+      Backend API          brew install go       │
│  Docker            24+        Containers           Docker Desktop        │
│  Git               2.40+      Version control      brew install git      │
│                                                                          │
│  OPTIONAL (for cluster work):                                            │
│  kubectl           1.30+      K8s CLI              brew install kubectl  │
│  helm              3.14+      Chart management     brew install helm     │
│  kubeseal          0.27+      Sealed Secrets CLI   brew install kubeseal │
│  terraform         1.7+       Infrastructure       brew install terraform│
│  ansible           2.16+      Server config        brew install ansible  │
│                                                                          │
│  OPTIONAL (for CI locally):                                              │
│  act               0.2+       Run GitHub Actions   brew install act      │
│  docker-compose    2.20+      Multi-container      Bundled with Docker   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Verify Installation

```bash
node --version    # v20.x.x
pnpm --version    # 10.29.x
go version        # go1.23.x
docker --version  # Docker 24.x.x
git --version     # git 2.x.x
```

---

## 2. Repository Setup

```bash
# 1. Clone the repository
git clone git@github.com:taikuri-infra/Zenith.git
cd Zenith

# 2. Install Node.js dependencies (all 3 frontend apps + shared packages)
pnpm install

# 3. Copy environment files
cp .env.example .env
cp apps/landing/.env.example apps/landing/.env.local
cp apps/web/.env.example apps/web/.env.local
cp apps/mission-control/.env.example apps/mission-control/.env.local

# 4. Edit .env with your values
#    At minimum, set JWT_SECRET (32+ chars):
echo 'JWT_SECRET=your-super-secret-key-at-least-32-chars-long' >> .env
```

---

## 3. Architecture: What Runs Where

```
┌─────────────────────────────────────────────────────────────────────────┐
│                LOCAL DEVELOPMENT ARCHITECTURE                            │
│                                                                          │
│  Your laptop runs everything needed for development:                     │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │  FRONTEND (Node.js / Next.js — hot reload)                          │ │
│  │                                                                     │ │
│  │  localhost:3200  ─── Landing Page    (apps/landing/)                │ │
│  │  localhost:3000  ─── Web Dashboard   (apps/web/)                   │ │
│  │  localhost:3100  ─── Mission Control (apps/mission-control/)       │ │
│  │                                                                     │ │
│  │  All three talk to the API at localhost:8080                        │ │
│  └────────────────────────────────┬────────────────────────────────────┘ │
│                                   │ HTTP (REST)                          │
│                                   ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │  BACKEND (Go / Fiber)                                               │ │
│  │                                                                     │ │
│  │  localhost:8080  ─── zenith-api  (services/api/)                   │ │
│  │                                                                     │ │
│  │  Modes:                                                             │ │
│  │    ZENITH_MODE=standalone  →  No Keycloak, Temporal, APISIX        │ │
│  │    K8S_MODE=memory         →  In-memory K8s (no real cluster)      │ │
│  │    DATABASE_URL=""         →  In-memory stores (no PostgreSQL)     │ │
│  │                                                                     │ │
│  │  OR with PostgreSQL:                                                │ │
│  │    DATABASE_URL=postgres://zenith:zenith@localhost:5432/zenith      │ │
│  └────────────────────────────────┬────────────────────────────────────┘ │
│                                   │ SQL (optional)                       │
│                                   ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │  DATABASE (Docker or native)                                        │ │
│  │                                                                     │ │
│  │  localhost:5432  ─── PostgreSQL 16  (via docker-compose)           │ │
│  │                                                                     │ │
│  │  User: zenith  │  Password: zenith  │  Database: zenith            │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  WHAT YOU DON'T NEED LOCALLY:                                            │
│  ───────────────────────────                                             │
│  Traefik, APISIX, Cilium, Keycloak, Temporal, ArgoCD, cert-manager,    │
│  external-dns, Kyverno, Falco, Velero, Sealed Secrets, Harbor, CNPG    │
│                                                                          │
│  The API runs in "standalone" mode without any of these.                │
│  For features that need them (provisioning, auth), use the staging      │
│  cluster or mock/in-memory implementations.                              │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Option A: Docker Compose (Recommended for First Day)

The simplest way to run everything — one command starts the API + PostgreSQL + Web UI.

```bash
# Start everything
docker compose up -d

# Check status
docker compose ps

# View API logs
docker compose logs -f api

# Stop everything
docker compose down
```

```
┌─────────────────────────────────────────────────────────────────────────┐
│                 DOCKER COMPOSE SERVICES                                  │
│                                                                          │
│  Service     Port    Image              Health Check                     │
│  ───────     ────    ─────              ────────────                     │
│  postgres    5432    postgres:16-alpine  pg_isready                      │
│  api         8080    zenith-api          /api/v1/health                  │
│  web         3000    zenith-web          (Next.js)                       │
│                                                                          │
│  Startup order:   postgres → api → web                                  │
│  Environment:     ZENITH_MODE=standalone, K8S_MODE=memory               │
│  Data:            PostgreSQL data persists in Docker volume              │
└─────────────────────────────────────────────────────────────────────────┘
```

### Verify It Works

```bash
# Health check
curl http://localhost:8080/api/v1/health
# Response: {"status":"ok"}

# Register a user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@test.com","password":"Password123","name":"Dev User"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@test.com","password":"Password123"}'

# Open the web dashboard
open http://localhost:3000
```

---

## 5. Option B: Local Processes (For Active Development)

Run each service as a native process for faster hot-reload during active development.

### Step 1: Start PostgreSQL (via Docker)

```bash
docker run -d --name zenith-pg \
  -e POSTGRES_USER=zenith \
  -e POSTGRES_PASSWORD=zenith \
  -e POSTGRES_DB=zenith \
  -p 5432:5432 \
  postgres:16-alpine
```

### Step 2: Start the Go API

```bash
cd services/api

# Set environment variables
export JWT_SECRET="your-super-secret-key-at-least-32-chars-long"
export DATABASE_URL="postgres://zenith:zenith@localhost:5432/zenith?sslmode=disable"
export ZENITH_MODE=standalone
export K8S_MODE=memory
export ADMIN_EMAIL=admin@localhost
export ADMIN_PASSWORD=changeme
export ENVIRONMENT=development
export CORS_ORIGINS="http://localhost:3000,http://localhost:3100,http://localhost:3200"

# Run (with auto-restart on file changes — install air first)
go install github.com/air-verse/air@latest
air

# OR without air (manual restart on changes)
go run ./cmd/server/
```

### Step 3: Start the Frontend Apps

```bash
# Terminal 1: Web Dashboard (port 3000)
pnpm dev:web

# Terminal 2: Mission Control (port 3100)
pnpm dev:mc

# Terminal 3: Landing Page (port 3200)
pnpm dev:landing

# OR start all three at once via Turbo:
pnpm dev
```

```
┌─────────────────────────────────────────────────────────────────────────┐
│                 TERMINAL LAYOUT (recommended)                            │
│                                                                          │
│  ┌──────────────────────┐  ┌──────────────────────┐                     │
│  │  Terminal 1           │  │  Terminal 2           │                     │
│  │  Go API               │  │  Web Dashboard        │                     │
│  │                        │  │                        │                     │
│  │  cd services/api      │  │  pnpm dev:web          │                     │
│  │  air                   │  │                        │                     │
│  │  (watching *.go)       │  │  (localhost:3000)      │                     │
│  │                        │  │  (hot reload)          │                     │
│  ├──────────────────────┤  ├──────────────────────┤                     │
│  │  Terminal 3           │  │  Terminal 4           │                     │
│  │  Mission Control      │  │  Git / General        │                     │
│  │                        │  │                        │                     │
│  │  pnpm dev:mc          │  │  git status            │                     │
│  │                        │  │  make test             │                     │
│  │  (localhost:3100)      │  │  kubectl ...           │                     │
│  │  (hot reload)          │  │                        │                     │
│  └──────────────────────┘  └──────────────────────┘                     │
└─────────────────────────────────────────────────────────────────────────┘
```

### Demo Mode (No API Needed)

If you only need to work on the frontend UI without a running API:

```bash
# In apps/web/.env.local:
NEXT_PUBLIC_DEMO_MODE=true

# Start the web app — it uses mock data from lib/mock-data.ts
pnpm dev:web
```

---

## 6. Connecting to Staging Cluster

For features that require real Kubernetes (Temporal, Keycloak, CNPG), connect to the staging cluster.

```bash
# 1. Get kubeconfig from team lead (stored in 1Password / secure vault)
#    Save it to ~/.kube/config-zenith-staging

# 2. Set kubeconfig
export KUBECONFIG=~/.kube/config-zenith-staging

# 3. Verify connection
kubectl get nodes
# NAME        STATUS   ROLES                  AGE
# zen-stage   Ready    control-plane,master   ...

# 4. Check running pods
kubectl get pods -n zenith-staging

# 5. Port-forward to staging API (for debugging)
kubectl port-forward -n zenith-staging svc/zenith-api 9080:8080
# Now staging API is available at localhost:9080

# 6. Port-forward to staging database
kubectl port-forward -n zenith-staging svc/free-pg-rw 15432:5432
# Connect: psql -h localhost -p 15432 -U postgres

# 7. Access ArgoCD UI
kubectl port-forward -n argocd svc/argocd-server 9443:443
# Open: https://localhost:9443

# 8. Access Grafana
kubectl port-forward -n monitoring svc/kube-prometheus-stack-grafana 3333:80
# Open: http://localhost:3333
```

```
┌─────────────────────────────────────────────────────────────────────────┐
│                 STAGING ACCESS DIAGRAM                                    │
│                                                                          │
│  YOUR LAPTOP                              ZEN-STAGE CLUSTER              │
│  ──────────                              ─────────────────              │
│                                                                          │
│  localhost:9080 ──── port-forward ────── zenith-api:8080                │
│  localhost:15432 ─── port-forward ────── free-pg-rw:5432               │
│  localhost:9443 ──── port-forward ────── argocd-server:443             │
│  localhost:3333 ──── port-forward ────── grafana:80                     │
│                                                                          │
│  Public URLs (no port-forward needed):                                   │
│  https://api.stage.freezenith.com        zenith-api (via APISIX)        │
│  https://app.stage.freezenith.com        zenith-web                     │
│  https://stage.freezenith.com            zenith-landing                 │
│  https://argocd.stage.freezenith.com     ArgoCD UI                      │
│  https://grafana.stage.freezenith.com    Grafana dashboards             │
│  https://auth.stage.freezenith.com       Keycloak admin                 │
│  https://temporal.stage.freezenith.com   Temporal Web UI                │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Environment Variables Reference

### Go API (`services/api/`)

| Variable | Required | Default | Purpose |
|----------|----------|---------|---------|
| `JWT_SECRET` | Yes | — | Token signing key (min 32 chars) |
| `PORT` | No | 8080 | API listen port |
| `ENVIRONMENT` | No | development | `development` or `production` |
| `ZENITH_MODE` | No | standalone | `standalone` (local) or `saas` (cluster) |
| `K8S_MODE` | No | memory | `memory` (fake) or `real` (actual cluster) |
| `DATABASE_URL` | No | — | PostgreSQL connection string (empty = in-memory) |
| `ADMIN_EMAIL` | No | admin@localhost | Seeded admin account email |
| `ADMIN_PASSWORD` | No | changeme | Seeded admin account password |
| `CORS_ORIGINS` | No | * | Comma-separated allowed origins |
| `BASE_DOMAIN` | No | freezenith.com | Used for DNS and routing |

**SaaS-only variables** (not needed locally):

| Variable | Purpose |
|----------|---------|
| `TEMPORAL_ENABLED` | Enable Temporal worker |
| `TEMPORAL_HOST` | Temporal frontend address |
| `KEYCLOAK_URL` | Keycloak base URL |
| `KEYCLOAK_ADMIN_USER` / `_PASSWORD` | Keycloak admin credentials |
| `STRIPE_SECRET_KEY` | Stripe billing |
| `S3_ENDPOINT` / `S3_ACCESS_KEY` / `S3_SECRET_KEY` | Hetzner S3 |
| `CNPG_ADMIN_DSN` | CNPG admin connection for CREATE DATABASE |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector endpoint |

### Frontend Apps

| Variable | App(s) | Default | Purpose |
|----------|--------|---------|---------|
| `NEXT_PUBLIC_API_URL` | web, mc | http://localhost:8080 | Backend API URL |
| `NEXT_PUBLIC_DEMO_MODE` | web, mc | false | Use mock data (no API needed) |
| `NEXT_PUBLIC_ZENITH_MODE` | web | saas | `saas` or `standalone` (changes sidebar nav) |
| `NEXT_PUBLIC_APP_URL` | landing | https://app.freezenith.com | Login/register URL |
| `NEXT_PUBLIC_LANDING_URL` | web, mc | https://freezenith.com | Link back to landing |

---

## 8. IDE Setup

### VS Code (Recommended)

```
Recommended extensions:
  ┌────────────────────────────────────────────────────────────────────┐
  │  Go                          golang.Go          │ Go language      │
  │  ESLint                      dbaeumer.eslint     │ JS/TS linting    │
  │  Tailwind CSS IntelliSense   bradlc.tailwindcss  │ Class autocomplete│
  │  Prettier                    esbenp.prettier     │ Code formatting  │
  │  YAML                        redhat.vscode-yaml  │ K8s manifests    │
  │  Kubernetes                  ms-kubernetes-tools │ K8s support      │
  │  HashiCorp Terraform         hashicorp.terraform │ .tf files        │
  │  Thunder Client              rangav.thunder      │ API testing      │
  └────────────────────────────────────────────────────────────────────┘
```

### GoLand / IntelliJ

- Import the `services/api` module as a Go project
- Set GOPATH to default, enable Go modules
- Add run configuration: `cmd/server/main.go` with env vars from section 7

---

## 9. Common Development Workflows

### Run Tests

```bash
# Go API tests
cd services/api && go test ./... -race -count=1

# Frontend lint (all apps)
pnpm lint

# All tests + lint via Makefile
make test

# Run CI locally via act (requires Docker)
make ci-images
```

### Build Docker Images

```bash
# Build API image
make build-api

# Build all images
make build-api build-landing build-web build-mc

# Push to Harbor (requires registry credentials)
make push-all
```

### Database Operations

```bash
# Connect to local PostgreSQL
psql postgres://zenith:zenith@localhost:5432/zenith

# Migrations run automatically on API startup
# To create a new migration:
# Add SQL files to services/api/internal/adapters/postgres/migrations/
```

---

## 10. Troubleshooting

### API won't start

```bash
# Check if PostgreSQL is running (if using DATABASE_URL)
docker ps | grep postgres
# If not: docker start zenith-pg

# Check if port 8080 is already in use
lsof -i :8080
# Kill the process or change PORT env var

# Check JWT_SECRET is set
echo $JWT_SECRET
# Must be at least 32 characters
```

### Frontend can't connect to API

```bash
# Check API is running
curl http://localhost:8080/api/v1/health

# Check CORS is configured
# API needs: CORS_ORIGINS=http://localhost:3000

# Check browser console for errors
# Common: "CORS policy" error → fix CORS_ORIGINS
```

### pnpm install fails

```bash
# Clear cache and retry
pnpm store prune
rm -rf node_modules
pnpm install

# If lockfile issues:
rm pnpm-lock.yaml
pnpm install
```

### Go dependency issues

```bash
cd services/api
go mod tidy
go mod download
```

### Port already in use

```bash
# Find and kill process on port
lsof -ti :8080 | xargs kill
lsof -ti :3000 | xargs kill
lsof -ti :5432 | xargs kill
```
