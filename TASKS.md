# Zenith Implementation Tasks

> Read CLAUDE.md first for project context. Read docs/*.md for detailed architecture.

## How to Use This File
- Tasks are ordered by priority and dependency
- Each task has: description, acceptance criteria, files to create/modify
- Mark tasks [x] when complete
- The frontend mockups (apps/web/, apps/mission-control/) are your UI reference

---

## Phase 1: Project Foundation (Week 1)

### 1.1 Go Backend API Scaffold
- [x] Create `services/api/` with Go module
- [x] Use Fiber or Echo framework
- [x] Health check endpoint: GET /health
- [x] Project structure: cmd/, internal/handlers/, internal/models/, internal/middleware/
- [x] Docker multi-stage build
- [x] Makefile with: build, test, lint, docker-build
- **Files:** services/api/**

### 1.2 Kubernetes CRD Definitions
- [x] Define CRDs in Go structs (kubebuilder markers):
  - `Project` - tenant project (name, owner, plan, status)
  - `App` - application deployment (name, image, replicas, env, ports, domain)
  - `Database` - managed database (engine, version, storage, backups)
  - `StorageBucket` - object storage (name, access, versioning)
  - `Domain` - custom domain (domain, app, ssl)
  - `AuthRealm` - auth realm (name, providers, clients)
  - `GatewayRoute` - Kong route (path, methods, service, plugins)
- [x] Generate YAML manifests with controller-gen
- [x] Register CRDs in the operator
- **Files:** services/operator/api/v1alpha1/

### 1.3 Zenith Operator Scaffold
- [ ] Create `services/operator/` with kubebuilder
- [ ] Reconcilers for each CRD type
- [ ] Hetzner Cloud client integration
- [ ] Event recording for status updates
- **Files:** services/operator/**

### 1.4 CLI Scaffold
- [ ] Create `cli/` with Go module + Cobra
- [ ] `zen version`, `zen help` commands
- [ ] Config file: ~/.zen/config.yaml
- [ ] Charm TUI utilities setup (lipgloss, bubbletea)
- **Files:** cli/**

---

## Phase 2: Core API (Week 2)

### 2.1 Authentication & Authorization
- [ ] JWT middleware (validate tokens from Zenith Auth)
- [ ] User model: id, email, name, role, project_id
- [ ] API key authentication (for CI/CD)
- [ ] RBAC: Owner, Admin, Developer, Viewer
- **Files:** services/api/internal/middleware/auth.go, services/api/internal/models/user.go

### 2.2 Project Management API
- [ ] POST /api/v1/projects - create project
- [ ] GET /api/v1/projects - list user's projects
- [ ] GET /api/v1/projects/:id - get project details
- [ ] PUT /api/v1/projects/:id - update project
- [ ] DELETE /api/v1/projects/:id - delete project (danger zone)
- [ ] Each project creates a K8s namespace
- **Files:** services/api/internal/handlers/projects.go

### 2.3 Apps API
- [ ] POST /api/v1/projects/:id/apps - deploy app (creates App CRD)
- [ ] GET /api/v1/projects/:id/apps - list apps
- [ ] GET /api/v1/projects/:id/apps/:name - app details (status, metrics)
- [ ] PUT /api/v1/projects/:id/apps/:name - update (replicas, env, image)
- [ ] DELETE /api/v1/projects/:id/apps/:name - delete app
- [ ] POST /api/v1/projects/:id/apps/:name/redeploy - trigger redeploy
- **Files:** services/api/internal/handlers/apps.go

### 2.4 Databases API
- [ ] CRUD for databases (creates Database CRD -> CNPG/Redis operator)
- [ ] Connection string generation
- [ ] Backup management (list, create, restore)
- **Files:** services/api/internal/handlers/databases.go

### 2.5 Storage API
- [ ] CRUD for storage buckets (maps to Hetzner Object Storage)
- [ ] Access control (private/public)
- [ ] Lifecycle policies
- **Files:** services/api/internal/handlers/storage.go

---

## Phase 3: Auth Service (Week 3)

### 3.1 Auth Service Core
- [ ] Create `services/auth/` - Go service
- [ ] OpenID Connect provider (token endpoint, userinfo, jwks)
- [ ] Realm management (CRUD)
- [ ] User management within realms (register, login, MFA)
- [ ] Client management (public/confidential, redirect URIs)
- [ ] Session management
- **Files:** services/auth/**

### 3.2 Identity Providers
- [ ] Google OAuth integration
- [ ] GitHub OAuth integration
- [ ] SAML support (for enterprise)
- [ ] OIDC federation
- **Files:** services/auth/internal/providers/

### 3.3 Kong Integration
- [ ] JWT plugin configuration (auto-configure Kong to validate Zenith Auth JWTs)
- [ ] Per-realm JWT validation
- [ ] Consumer management synced with Auth clients
- **Files:** services/auth/internal/kong/

---

## Phase 4: Operator Reconcilers (Week 4-5)

### 4.1 App Reconciler
- [ ] Watch App CRDs
- [ ] Create Deployment + Service + Ingress
- [ ] Health checks and readiness probes
- [ ] Auto-scaling based on spec
- [ ] Rolling updates
- **Files:** services/operator/internal/controllers/app_controller.go

### 4.2 Database Reconciler
- [ ] Watch Database CRDs
- [ ] PostgreSQL: Create CNPG Cluster CR
- [ ] Redis: Create Redis CR
- [ ] MySQL: Create MySQL CR
- [ ] Hetzner Volume provisioning for storage
- [ ] Connection secret generation
- [ ] Automated backups to Hetzner Object Storage
- **Files:** services/operator/internal/controllers/database_controller.go

### 4.3 Domain Reconciler
- [ ] Watch Domain CRDs
- [ ] Create/update Hetzner DNS records
- [ ] cert-manager Certificate resources
- [ ] Kong Ingress configuration
- **Files:** services/operator/internal/controllers/domain_controller.go

### 4.4 Storage Reconciler
- [ ] Watch StorageBucket CRDs
- [ ] Hetzner Object Storage bucket creation
- [ ] Access policy management
- [ ] Credential generation (S3-compatible keys)
- **Files:** services/operator/internal/controllers/storage_controller.go

---

## Phase 5: CLI (Week 6)

### 5.1 zen install
- [ ] Interactive TUI wizard (Charm huh)
- [ ] Hetzner token input + validation
- [ ] Server type selection
- [ ] Region selection
- [ ] Progress animation (bubbletea)
- [ ] Creates management plane (k3s + CAPI)
- **Files:** cli/cmd/install.go, cli/internal/install/

### 5.2 zen deploy
- [ ] Auto-detect project type (Dockerfile, package.json, go.mod, etc.)
- [ ] Build + push to registry
- [ ] Create/update App CRD
- [ ] Stream deployment progress
- **Files:** cli/cmd/deploy.go

### 5.3 zen status / zen top / zen logs
- [ ] `zen status` - rich project overview (lipgloss tables)
- [ ] `zen top` - real-time resource monitor (htop-style, bubbletea)
- [ ] `zen logs` - color-coded log streaming from Loki
- **Files:** cli/cmd/status.go, cli/cmd/top.go, cli/cmd/logs.go

### 5.4 zen db connect
- [ ] Auto port-forward to database
- [ ] Launch psql/redis-cli/mongosh
- **Files:** cli/cmd/db.go

---

## Phase 6: Monitoring Stack (Week 7)

### 6.1 Prometheus Setup
- [ ] Helm chart values for kube-prometheus-stack
- [ ] ServiceMonitor CRDs for all services
- [ ] Pre-built alerting rules
- **Files:** helm/monitoring/

### 6.2 Grafana Dashboards
- [ ] Platform Overview dashboard (JSON)
- [ ] Service Health dashboard
- [ ] Node Metrics dashboard
- [ ] Tenant-specific dashboards (auto-generated)
- **Files:** helm/monitoring/dashboards/

### 6.3 Loki Setup
- [ ] Loki + Promtail Helm chart values
- [ ] Log retention policies
- [ ] Multi-tenant log separation
- **Files:** helm/monitoring/

---

## Phase 7: Connect Frontend to Backend (Week 8)

### 7.1 API Client
- [ ] Replace mock data in apps/web/ with real API calls
- [ ] Create shared API client library
- [ ] WebSocket for real-time updates (deployment progress, logs)
- [ ] Error handling and loading states
- **Files:** apps/web/src/lib/api.ts, apps/web/src/hooks/

### 7.2 Auth Integration
- [ ] Login/register pages
- [ ] OAuth flow (Google, GitHub)
- [ ] JWT token management (refresh, storage)
- [ ] Protected routes
- **Files:** apps/web/src/app/login/, apps/web/src/middleware.ts

---

## Phase 8: Helm Charts & Deployment (Week 9)

### 8.1 Platform Helm Chart
- [ ] Chart for: API, Operator, Auth, Kong, Monitoring stack
- [ ] Values.yaml with sensible defaults
- [ ] NOTES.txt with post-install instructions
- **Files:** helm/zenith/

### 8.2 zen install Integration
- [ ] CLI installs platform via Helm chart
- [ ] Post-install verification
- [ ] Welcome wizard redirect
- **Files:** cli/internal/install/helm.go

---

## Future Phases (See docs/PHASES.md for full detail)
- Phase 9: GitOps (zen export/apply/diff, GitSync CRD)
- Phase 10: Management Plane (CAPI, Mission Control backend)
- Phase 11: Landing page (freezenith.com)
- Phase 12: CNCF integrations (OpenTelemetry, Backstage, Crossplane)
