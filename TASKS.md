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
- [x] Create `services/operator/` with kubebuilder
- [x] Reconcilers for each CRD type
- [x] Hetzner Cloud client integration
- [x] Event recording for status updates
- **Files:** services/operator/**

### 1.4 CLI Scaffold
- [x] Create `cli/` with Go module + Cobra
- [x] `zen version`, `zen help` commands
- [x] Config file: ~/.zen/config.yaml
- [x] Charm TUI utilities setup (lipgloss, bubbletea)
- **Files:** cli/**

---

## Phase 2: Core API (Week 2)

### 2.1 Authentication & Authorization
- [x] JWT middleware (validate tokens from Zenith Auth)
- [x] User model: id, email, name, role, project_id
- [x] API key authentication (for CI/CD)
- [x] RBAC: Owner, Admin, Developer, Viewer
- **Files:** services/api/internal/middleware/auth.go, services/api/internal/models/user.go

### 2.2 Project Management API
- [x] POST /api/v1/projects - create project
- [x] GET /api/v1/projects - list user's projects
- [x] GET /api/v1/projects/:id - get project details
- [x] PUT /api/v1/projects/:id - update project
- [x] DELETE /api/v1/projects/:id - delete project (danger zone)
- [x] Each project creates a K8s namespace
- **Files:** services/api/internal/handlers/projects.go

### 2.3 Apps API
- [x] POST /api/v1/projects/:id/apps - deploy app (creates App CRD)
- [x] GET /api/v1/projects/:id/apps - list apps
- [x] GET /api/v1/projects/:id/apps/:name - app details (status, metrics)
- [x] PUT /api/v1/projects/:id/apps/:name - update (replicas, env, image)
- [x] DELETE /api/v1/projects/:id/apps/:name - delete app
- [x] POST /api/v1/projects/:id/apps/:name/redeploy - trigger redeploy
- **Files:** services/api/internal/handlers/apps.go

### 2.4 Databases API
- [x] CRUD for databases (creates Database CRD -> CNPG/Redis operator)
- [x] Connection string generation
- [x] Backup management (list, create, restore)
- **Files:** services/api/internal/handlers/databases.go

### 2.5 Storage API
- [x] CRUD for storage buckets (maps to Hetzner Object Storage)
- [x] Access control (private/public)
- [x] Lifecycle policies
- **Files:** services/api/internal/handlers/storage.go

---

## Phase 3: Auth Service (Week 3)

### 3.1 Auth Service Core
- [x] Create `services/auth/` - Go service
- [x] OpenID Connect provider (token endpoint, userinfo, jwks)
- [x] Realm management (CRUD)
- [x] User management within realms (register, login, MFA)
- [x] Client management (public/confidential, redirect URIs)
- [x] Session management
- **Files:** services/auth/**

### 3.2 Identity Providers
- [x] Google OAuth integration
- [x] GitHub OAuth integration
- [x] SAML support (for enterprise)
- [x] OIDC federation
- **Files:** services/auth/internal/providers/

### 3.3 Kong Integration
- [x] JWT plugin configuration (auto-configure Kong to validate Zenith Auth JWTs)
- [x] Per-realm JWT validation
- [x] Consumer management synced with Auth clients
- **Files:** services/auth/internal/kong/

---

## Phase 4: Operator Reconcilers (Week 4-5)

### 4.1 App Reconciler
- [x] Watch App CRDs
- [x] Create Deployment + Service + Ingress
- [x] Health checks and readiness probes
- [x] Auto-scaling based on spec
- [x] Rolling updates
- **Files:** services/operator/internal/controllers/app_controller.go

### 4.2 Database Reconciler
- [x] Watch Database CRDs
- [x] PostgreSQL: Create CNPG Cluster CR
- [x] Redis: Create Redis CR
- [x] MySQL: Create MySQL CR
- [x] Hetzner Volume provisioning for storage
- [x] Connection secret generation
- [x] Automated backups to Hetzner Object Storage
- **Files:** services/operator/internal/controllers/database_controller.go

### 4.3 Domain Reconciler
- [x] Watch Domain CRDs
- [x] Create/update Hetzner DNS records
- [x] cert-manager Certificate resources
- [x] Kong Ingress configuration
- **Files:** services/operator/internal/controllers/domain_controller.go

### 4.4 Storage Reconciler
- [x] Watch StorageBucket CRDs
- [x] Hetzner Object Storage bucket creation
- [x] Access policy management
- [x] Credential generation (S3-compatible keys)
- **Files:** services/operator/internal/controllers/storage_controller.go

---

## Phase 5: CLI (Week 6)

### 5.1 zen install
- [x] Interactive TUI wizard (Charm huh)
- [x] Hetzner token input + validation
- [x] Server type selection
- [x] Region selection
- [x] Progress animation (bubbletea)
- [x] Creates management plane (k3s + CAPI)
- **Files:** cli/cmd/install.go, cli/internal/install/

### 5.2 zen deploy
- [x] Auto-detect project type (Dockerfile, package.json, go.mod, etc.)
- [x] Build + push to registry
- [x] Create/update App CRD
- [x] Stream deployment progress
- **Files:** cli/cmd/deploy.go

### 5.3 zen status / zen top / zen logs
- [x] `zen status` - rich project overview (lipgloss tables)
- [x] `zen top` - real-time resource monitor (htop-style, bubbletea)
- [x] `zen logs` - color-coded log streaming from Loki
- **Files:** cli/cmd/status.go, cli/cmd/top.go, cli/cmd/logs.go

### 5.4 zen db connect
- [x] Auto port-forward to database
- [x] Launch psql/redis-cli/mongosh
- **Files:** cli/cmd/db.go

---

## Phase 6: Monitoring Stack (Week 7)

### 6.1 Prometheus Setup
- [x] Helm chart values for kube-prometheus-stack
- [x] ServiceMonitor CRDs for all services
- [x] Pre-built alerting rules
- **Files:** helm/monitoring/

### 6.2 Grafana Dashboards
- [x] Platform Overview dashboard (JSON)
- [x] Service Health dashboard
- [x] Node Metrics dashboard
- [x] Tenant-specific dashboards (auto-generated)
- **Files:** helm/monitoring/dashboards/

### 6.3 Loki Setup
- [x] Loki + Promtail Helm chart values
- [x] Log retention policies
- [x] Multi-tenant log separation
- **Files:** helm/monitoring/

---

## Phase 7: Connect Frontend to Backend (Week 8)

### 7.1 API Client
- [x] Replace mock data in apps/web/ with real API calls
- [x] Create shared API client library
- [x] WebSocket for real-time updates (deployment progress, logs)
- [x] Error handling and loading states
- **Files:** apps/web/src/lib/api.ts, apps/web/src/hooks/

### 7.2 Auth Integration
- [x] Login/register pages
- [x] OAuth flow (Google, GitHub)
- [x] JWT token management (refresh, storage)
- [x] Protected routes
- **Files:** apps/web/src/app/login/, apps/web/src/middleware.ts

---

## Phase 8: Helm Charts & Deployment (Week 9)

### 8.1 Platform Helm Chart
- [x] Chart for: API, Operator, Auth, Kong, Monitoring stack
- [x] Values.yaml with sensible defaults
- [x] NOTES.txt with post-install instructions
- **Files:** helm/zenith/

### 8.2 zen install Integration
- [x] CLI installs platform via Helm chart
- [x] Post-install verification
- [x] Welcome wizard redirect
- **Files:** cli/internal/install/helm.go

---

## Phase 9: GitOps (Week 10)

### 9.1 GitSync CRD
- [x] Define GitSync CRD in Go structs (repoURL, branch, path, interval, autoSync, pruneResources)
- [x] Register GitSync CRD with scheme
- [x] Add deepcopy methods
- **Files:** services/operator/api/v1alpha1/gitsync_types.go, services/operator/api/v1alpha1/zz_generated.deepcopy.go

### 9.2 GitSync Controller
- [x] Watch GitSync CRDs
- [x] Create sync ConfigMap for tracking state
- [x] Track last synced commit hash
- [x] Handle periodic requeue for AutoSync
- [x] Finalizer pattern for cleanup
- [x] Manifest parsing utility (ParseManifests)
- [x] Manifest apply utility (ApplyManifest)
- **Files:** services/operator/internal/controllers/gitsync_controller.go

### 9.3 CLI GitOps Commands
- [x] `zen export` - Export Zenith resources to YAML/JSON files
- [x] `zen apply` - Apply Zenith resource manifests from files/directories
- [x] `zen diff` - Show diff between local manifests and cluster state
- [x] Wire commands into root.go
- **Files:** cli/cmd/export/export.go, cli/cmd/apply/apply.go, cli/cmd/diff/diff.go, cli/cmd/root/root.go

### 9.4 Tests
- [x] GitSync controller tests (create, defaults, autosync, finalizer, deletion, not found)
- [x] ParseManifests tests (multi-doc, empty, invalid)
- [x] Export command tests (YAML/JSON marshal, file parsing, directory collection)
- [x] Apply command tests (data parsing, directory collection, result types)
- [x] Diff command tests (compare specs: identical, modified, added, removed, multiple, empty)
- **Files:** services/operator/internal/controllers/controllers_test.go, cli/cmd/export/export_test.go, cli/cmd/apply/apply_test.go, cli/cmd/diff/diff_test.go

---

## Future Phases (See docs/PHASES.md for full detail)
- Phase 10: Management Plane (CAPI, Mission Control backend)
- Phase 11: Landing page (freezenith.com)
- Phase 12: CNCF integrations (OpenTelemetry, Backstage, Crossplane)
