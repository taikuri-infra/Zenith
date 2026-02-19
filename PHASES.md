# Zenith - Complete Phase Breakdown

> Every phase. Every task. Every test case. Every operator.

---

## Phase 0: Project Setup (Week 1)

**Goal:** Repository structure, CI, dev environment ready.

### Tasks

```
P0-01  Initialize Go module (github.com/freezenith/zenith)
P0-02  Initialize Next.js 15 project in web/ (TypeScript, Tailwind, shadcn/ui)
P0-03  Create Makefile (build, test, generate, lint, docker)
P0-04  Create Dockerfile for galaxy-operator (multi-stage Go build)
P0-05  Create Dockerfile for galaxy-api (multi-stage Go build)
P0-06  Create Dockerfile for web (Next.js standalone)
P0-07  Create docker-compose.yml for local dev (postgres, redis, k3s)
P0-08  Create GitHub Actions CI (lint, test, build for all 3 components)
P0-09  Create .github/ISSUE_TEMPLATE/ (bug, feature, proposal)
P0-10  Create .github/PULL_REQUEST_TEMPLATE.md
P0-11  Write LICENSE (Apache 2.0)
P0-12  Write CODE_OF_CONDUCT.md (Contributor Covenant)
P0-13  Write CONTRIBUTING.md
P0-14  Write SECURITY.md
P0-15  Write GOVERNANCE.md
P0-16  Create OWNERS file
P0-17  Setup golangci-lint config (.golangci.yml)
P0-18  Setup ESLint + Prettier for web/ (.eslintrc, .prettierrc)
P0-19  Install kubebuilder for CRD scaffolding
P0-20  Create charts/zenith/ Helm chart skeleton
```

### Test Cases
```
T0-01  `make build` compiles all 3 binaries without error
T0-02  `make test` runs (0 tests, 0 failures)
T0-03  `make lint` passes
T0-04  `make docker-build` creates all 3 images
T0-05  GitHub Actions CI passes on push
T0-06  `cd web && npm run build` succeeds
T0-07  Helm chart lints: `helm lint charts/zenith/`
```

### Definition of Done
- [ ] `git clone` + `make build` works on clean machine
- [ ] CI green
- [ ] All community files present (LICENSE, CONTRIBUTING, etc.)

---

## Phase 1: Frontend Shell + Design System (Week 2-3)

**Goal:** Complete UI with mock data. Every page visible. No backend needed.

### Tasks

```
P1-01  Install shadcn/ui components (button, input, select, dialog, table, tabs,
       badge, card, toast, skeleton, command, dropdown-menu, sheet, slider, switch)
P1-02  Create design tokens (colors, spacing, typography) in tailwind.config.ts
P1-03  Create layout: AppLayout (sidebar + header + main content area)
P1-04  Create component: Sidebar (all navigation items with icons)
P1-05  Create component: Header (project selector dropdown, notifications, avatar)
P1-06  Create component: StatusBadge (Running/Deploying/Failed/Stopped)
P1-07  Create component: ResourceBar (usage percentage bar)
P1-08  Create component: CopyButton (click to copy with toast)
P1-09  Create component: CostEstimate (inline price display)
P1-10  Create component: EmptyState (illustration + CTA)
P1-11  Create component: LogViewer (scrollable log output with auto-scroll)
P1-12  Create component: TerminalOutput (build log display)
P1-13  Create component: PlanetSelector (planet size cards)
P1-14  Create component: EnvVarEditor (key-value pair editor)
P1-15  Create component: DatabaseLinker (link DB to app)
P1-16  Create component: DomainVerifier (CNAME instructions + status)
P1-17  Create component: MetricsChart (placeholder chart component)

P1-20  Create page: /login
P1-21  Create page: /register
P1-22  Create page: /projects (project list)
P1-23  Create modal: CreateProject
P1-24  Create page: /projects/[id] (overview dashboard)
P1-25  Create page: /projects/[id]/apps (app list)
P1-26  Create page: /projects/[id]/apps/new (deploy new app - 3 options)
P1-27  Create page: /projects/[id]/apps/new/github (GitHub deploy flow)
P1-28  Create page: /projects/[id]/apps/new/docker (Docker deploy flow)
P1-29  Create page: /projects/[id]/apps/new/template (template gallery)
P1-30  Create page: /projects/[id]/apps/[appId] (app detail)
P1-31  Create tab: App > Overview
P1-32  Create tab: App > Logs
P1-33  Create tab: App > Env Vars
P1-34  Create tab: App > Domains
P1-35  Create tab: App > Scaling
P1-36  Create tab: App > Deployments (history + rollback)

P1-40  Create page: /projects/[id]/databases (database list)
P1-41  Create page: /projects/[id]/databases/new (create database)
P1-42  Create page: /projects/[id]/databases/[dbId] (database detail)
P1-43  Create tab: Database > Connection
P1-44  Create tab: Database > Backups
P1-45  Create tab: Database > Metrics
P1-46  Create tab: Database > Logs
P1-47  Create tab: Database > Settings

P1-50  Create page: /projects/[id]/storage (S3 + volumes)
P1-51  Create page: /projects/[id]/storage/buckets/new
P1-52  Create page: /projects/[id]/storage/buckets/[bucketId]

P1-55  Create page: /projects/[id]/networking (tabs: domains, LB, firewall, DNS, IPs, VPN, gateway)
P1-56  Create tab: Networking > Domains
P1-57  Create tab: Networking > Load Balancers
P1-58  Create tab: Networking > Firewalls
P1-59  Create tab: Networking > DNS
P1-60  Create tab: Networking > Floating IPs
P1-61  Create tab: Networking > VPN
P1-62  Create tab: Networking > API Gateway

P1-65  Create page: /projects/[id]/auth (users, roles, SSO, API keys)
P1-66  Create tab: Auth > Users
P1-67  Create tab: Auth > Roles
P1-68  Create tab: Auth > SSO Providers
P1-69  Create tab: Auth > API Keys

P1-70  Create page: /projects/[id]/monitoring (dashboard, alerts, logs)
P1-71  Create tab: Monitoring > Dashboard (Grafana placeholder)
P1-72  Create tab: Monitoring > Alerts
P1-73  Create tab: Monitoring > Logs (centralized)

P1-75  Create page: /projects/[id]/registry (images, push instructions)

P1-80  Create page: /projects/[id]/planets (node list + add)
P1-81  Create modal: Add a Planet

P1-85  Create page: /projects/[id]/settings (general, team, billing, danger zone)
P1-86  Create tab: Settings > General
P1-87  Create tab: Settings > Team
P1-88  Create tab: Settings > Danger Zone

P1-90  Create page: /billing (cost breakdown, invoices)

P1-95  Create page: /account (profile, API keys)

P1-98  Create mock data layer (src/lib/mock-data.ts) with realistic sample data
P1-99  Create API client skeleton (src/lib/api-client.ts) with all endpoint types
```

### Test Cases
```
T1-01  All pages render without error (no React errors in console)
T1-02  Sidebar navigation works (every link goes to correct page)
T1-03  Project selector switches project context
T1-04  All mock data displays correctly
T1-05  All modals open and close properly
T1-06  Copy buttons work (clipboard API)
T1-07  Mobile responsive (sidebar collapses to sheet)
T1-08  Dark theme renders correctly
T1-09  Empty states show when no data
T1-10  Loading skeletons display during simulated loading
T1-11  `npm run build` succeeds with no TypeScript errors
T1-12  `npm run lint` passes
```

### Definition of Done
- [ ] Every page from FRONTEND.md is implemented with mock data
- [ ] A non-technical person can click through the entire UI
- [ ] Mobile responsive
- [ ] No TypeScript errors, no console errors

---

## Phase 2: Backend API + Auth (Week 4-5)

**Goal:** Go API server with auth, project CRUD, database. Frontend connects to real API.

### Tasks

```
P2-01  Create Go API server skeleton (cmd/galaxy-api/main.go)
P2-02  Choose HTTP framework: Echo or Gin (recommend Echo for simplicity)
P2-03  Create PostgreSQL connection + migration system (golang-migrate)
P2-04  Create database schema:
         - users (id, email, name, password_hash, github_id, google_id, created_at)
         - projects (id, user_id, name, region, status, hetzner_token_encrypted, created_at)
         - api_keys (id, user_id, key_hash, name, created_at, last_used_at)
P2-05  Create middleware: JWT auth (access + refresh tokens)
P2-06  Create middleware: API key auth (for CLI/CI)
P2-07  Create middleware: CORS
P2-08  Create middleware: request logging (structured, JSON)
P2-09  Create middleware: rate limiting (per user)
P2-10  Create handler: POST /auth/register (email + password)
P2-11  Create handler: POST /auth/login (email + password → JWT)
P2-12  Create handler: POST /auth/refresh (refresh token → new JWT)
P2-13  Create handler: GET /auth/github (OAuth redirect)
P2-14  Create handler: GET /auth/github/callback (OAuth callback)
P2-15  Create handler: GET /auth/google (OAuth redirect)
P2-16  Create handler: GET /auth/google/callback
P2-17  Create handler: GET /me (current user)
P2-18  Create handler: PUT /me (update profile)
P2-19  Create handler: POST /projects (create project)
P2-20  Create handler: GET /projects (list user's projects)
P2-21  Create handler: GET /projects/:id (get project)
P2-22  Create handler: PUT /projects/:id (update project)
P2-23  Create handler: DELETE /projects/:id (delete project)
P2-24  Create handler: POST /api-keys (create API key)
P2-25  Create handler: GET /api-keys (list API keys)
P2-26  Create handler: DELETE /api-keys/:id (revoke API key)
P2-27  Encrypt Hetzner tokens at rest (AES-256-GCM)
P2-28  Create health check endpoint: GET /health
P2-29  Create OpenAPI/Swagger spec generation
P2-30  Connect frontend to real API (replace mock data calls)
P2-31  Implement auth flow in frontend (login, register, JWT storage, redirect)
P2-32  Implement project CRUD in frontend
P2-33  Implement protected routes in frontend (redirect to /login if not authed)
```

### Test Cases
```
T2-01  POST /auth/register creates user, returns JWT
T2-02  POST /auth/register rejects duplicate email
T2-03  POST /auth/login returns JWT for valid credentials
T2-04  POST /auth/login rejects invalid password
T2-05  GET /me returns user with valid JWT
T2-06  GET /me returns 401 without JWT
T2-07  POST /projects creates project
T2-08  GET /projects returns only current user's projects
T2-09  DELETE /projects/:id deletes project
T2-10  Hetzner token is encrypted in database
T2-11  Rate limiting works (429 after limit)
T2-12  API key auth works for CLI endpoints
T2-13  Frontend login flow works end-to-end
T2-14  Frontend project creation works end-to-end
T2-15  80%+ code coverage on handlers
```

### Definition of Done
- [ ] User can register, login, create project via frontend
- [ ] JWT auth works
- [ ] API key auth works
- [ ] All tests pass, 80%+ coverage

---

## Phase 3: CRDs + Galaxy Operator Foundation (Week 6-7)

**Goal:** All CRDs defined. Operator skeleton running. Project controller works.

### Tasks

```
P3-01  Install kubebuilder, initialize operator project
P3-02  Define CRD: Project (api/v1alpha1/project_types.go)
P3-03  Define CRD: Application (api/v1alpha1/application_types.go)
P3-04  Define CRD: Build (api/v1alpha1/build_types.go)
P3-05  Define CRD: Planet (api/v1alpha1/planet_types.go)
P3-06  Define CRD: Database (api/v1alpha1/database_types.go)
P3-07  Define CRD: ObjectStore (api/v1alpha1/objectstore_types.go)
P3-08  Define CRD: KeyValueStore (api/v1alpha1/keyvaluestore_types.go)
P3-09  Define CRD: BackupPolicy (api/v1alpha1/backuppolicy_types.go)
P3-10  Define CRD: Domain (api/v1alpha1/domain_types.go)
P3-11  Define CRD: Firewall (api/v1alpha1/firewall_types.go)
P3-12  Define CRD: Network (api/v1alpha1/network_types.go)
P3-13  Define CRD: FloatingIP (api/v1alpha1/floatingip_types.go)
P3-14  Define CRD: LoadBalancer (api/v1alpha1/loadbalancer_types.go)
P3-15  Define CRD: DNSZone (api/v1alpha1/dnszone_types.go)
P3-16  Define CRD: DNSRecord (api/v1alpha1/dnsrecord_types.go)
P3-17  Define CRD: VPNPeer (api/v1alpha1/vpnpeer_types.go)
P3-18  Define CRD: Gateway (api/v1alpha1/gateway_types.go)
P3-19  Define CRD: Registry (api/v1alpha1/registry_types.go)
P3-20  Define CRD: AuthRealm (api/v1alpha1/authrealm_types.go)
P3-21  Define CRD: Monitoring (api/v1alpha1/monitoring_types.go)
P3-22  Define CRD: LogPipeline (api/v1alpha1/logpipeline_types.go)
P3-23  Define CRD: AlertRule (api/v1alpha1/alertrule_types.go)
P3-24  Define CRD: MessageQueue (api/v1alpha1/messagequeue_types.go)
P3-25  Define CRD: CronTask (api/v1alpha1/crontask_types.go)
P3-26  Run `make generate` (deepcopy, client code)
P3-27  Run `make manifests` (CRD YAML generation)
P3-28  Create Hetzner client wrapper (internal/provider/hetzner/client.go)
P3-29  Implement Hetzner server operations (create, delete, list)
P3-30  Implement Hetzner volume operations (create, delete, resize)
P3-31  Implement Hetzner network operations (create, delete, subnet)
P3-32  Implement Hetzner firewall operations (create, update, delete)
P3-33  Implement Hetzner floating IP operations
P3-34  Implement Hetzner load balancer operations
P3-35  Implement Hetzner DNS operations
P3-36  Implement Hetzner object storage operations (S3 API)
P3-37  Create provider interface (internal/provider/interface.go)
P3-38  Implement ProjectController:
         - Watch Project CRD
         - Create K8s namespace (galaxy-{project-name})
         - Create ResourceQuota
         - Create LimitRange
         - Create NetworkPolicy (deny cross-namespace)
         - Create RBAC (ServiceAccount, Role, RoleBinding)
         - Update status
P3-39  Wire up operator main.go with ProjectController
P3-40  Update Helm chart with CRDs and operator deployment
P3-41  Backend API: POST /projects now also creates Project CRD in K8s
P3-42  Backend API: DELETE /projects now also deletes Project CRD
```

### Test Cases
```
T3-01  All 25 CRDs install cleanly: kubectl apply -f config/crd/
T3-02  CRD validation works (invalid spec rejected by K8s)
T3-03  ProjectController creates namespace on Project CR creation
T3-04  ProjectController creates ResourceQuota in namespace
T3-05  ProjectController creates NetworkPolicy
T3-06  ProjectController deletes namespace when Project CR deleted
T3-07  Hetzner client can create/delete a test server (integration test)
T3-08  Hetzner client can create/delete a test volume (integration test)
T3-09  Operator starts without errors
T3-10  80%+ coverage on all controller logic
```

### Operators Installed
```
(none yet - Galaxy operator only)
```

### Definition of Done
- [ ] All 25 CRDs defined and installable
- [ ] Galaxy Operator runs and Project controller works
- [ ] Hetzner client works for all resource types
- [ ] Creating a Project via API creates namespace in K8s

---

## Phase 4: Application Deployment - Docker Image (Week 8-9)

**Goal:** Deploy app from Docker image. Working ingress + SSL.

### Tasks

```
P4-01  Implement ApplicationController:
         - Watch Application CRD
         - Create K8s Deployment (image, replicas, resources, env)
         - Create K8s Service (ClusterIP, port from spec)
         - Watch for linked databases → inject env vars
         - Update status (phase: Pending → Creating → Running → Failed)
P4-02  Implement DomainController:
         - Watch Domain CRD
         - Create Ingress resource with TLS
         - Create cert-manager Certificate CR
         - Update status (Pending → VerifyDNS → Active)
P4-03  Backend API: CRUD endpoints for Application
         POST   /projects/:id/apps
         GET    /projects/:id/apps
         GET    /projects/:id/apps/:appId
         PUT    /projects/:id/apps/:appId
         DELETE /projects/:id/apps/:appId
P4-04  Backend API: CRUD endpoints for Domain
         POST   /projects/:id/domains
         GET    /projects/:id/domains
         DELETE /projects/:id/domains/:domainId
P4-05  Backend API: Environment variables endpoints
         GET    /projects/:id/apps/:appId/env
         PUT    /projects/:id/apps/:appId/env
P4-06  Backend API: App logs endpoint (WebSocket)
         GET    /projects/:id/apps/:appId/logs (WS)
P4-07  Backend API: App scaling endpoint
         PUT    /projects/:id/apps/:appId/scale
P4-08  Frontend: Connect Apps pages to real API
P4-09  Frontend: Real-time log streaming via WebSocket
P4-10  Frontend: Deploy from Docker image flow (working)
P4-11  Frontend: App detail (all tabs working with real data)
P4-12  Install Traefik IngressClass configuration
P4-13  Install cert-manager + ClusterIssuer (Let's Encrypt)
```

### Test Cases
```
T4-01  Create Application CR → K8s Deployment created
T4-02  Application with 3 replicas → 3 pods running
T4-03  Delete Application CR → Deployment + Service deleted
T4-04  Domain CR with valid DNS → SSL certificate issued
T4-05  App accessible via custom domain with HTTPS
T4-06  App logs stream via WebSocket
T4-07  Scaling from 1 to 3 replicas works
T4-08  Env var update triggers rolling restart
T4-09  Internal apps (expose: false) not accessible from internet
T4-10  Frontend deploy flow works end-to-end
```

### Operators Installed
```
+ cert-manager (for SSL)
+ Traefik (for ingress) — comes with k3s
```

### Definition of Done
- [ ] Deploy Docker image → app running with HTTPS
- [ ] Logs streaming works
- [ ] Scaling works
- [ ] Custom domains with auto SSL

---

## Phase 5: Build Pipeline - GitHub Deploy (Week 10-11)

**Goal:** Connect GitHub, auto-build, auto-deploy on push.

### Tasks

```
P5-01  Create GitHub App for Zenith (webhook + repo access)
P5-02  Implement BuildController:
         - Watch Build CRD
         - Create Kaniko Job (build Dockerfile → push to registry)
         - Stream build logs to CRD status
         - On success: update Application image tag → triggers rollout
         - On failure: update status with error
P5-03  Implement RegistryController:
         - Watch Registry CRD
         - Install Harbor via Helm (with PVC from Hetzner Volume)
         - Create registry credentials Secret
         - Update status with registry URL
P5-04  Backend API: GitHub webhook endpoint
         POST /webhooks/github
P5-05  Backend API: GitHub repo listing
         GET /github/repos
P5-06  Backend API: Build endpoints
         GET /projects/:id/apps/:appId/builds
         GET /projects/:id/apps/:appId/builds/:buildId
         GET /projects/:id/apps/:appId/builds/:buildId/logs (WS)
P5-07  Backend API: Registry endpoints
         POST /projects/:id/registry (enable registry)
         GET  /projects/:id/registry
P5-08  Implement auto-deploy: git push → webhook → Build CR → Kaniko → new image → rollout
P5-09  Frontend: GitHub connect flow (OAuth → select repo)
P5-10  Frontend: Build logs viewer (real-time)
P5-11  Frontend: Deployment history with rollback
P5-12  Frontend: Registry page (images, tags, push instructions)
P5-13  Support monorepo: path-based Dockerfile + build trigger paths
```

### Test Cases
```
T5-01  GitHub webhook triggers Build CR creation
T5-02  Build CR triggers Kaniko Job
T5-03  Kaniko builds image and pushes to Harbor registry
T5-04  Successful build updates Application image → new rollout
T5-05  Failed build shows error in Build status
T5-06  Build logs stream in real-time
T5-07  Rollback to previous version works
T5-08  Monorepo: only rebuilds when relevant paths change
T5-09  Harbor registry accessible with credentials
T5-10  Frontend GitHub connect → deploy → running end-to-end
```

### Operators Installed
```
+ Harbor (container registry)
```

### Definition of Done
- [ ] Connect GitHub repo → auto build → auto deploy on push
- [ ] Build logs visible in real-time
- [ ] Rollback to previous deployment works
- [ ] Registry running with images stored on Hetzner Volume

---

## Phase 6: PostgreSQL + Redis (Week 12-13)

**Goal:** Managed databases with auto-provisioned Hetzner Volumes.

### Tasks

```
P6-01  Install CloudNativePG operator via Helm
P6-02  Install Redis Operator (Spotahome or OpsTree) via Helm
P6-03  Implement DatabaseController (for engine: postgresql):
         - Watch Database CRD
         - Call Hetzner API: create Volume (spec.storage size)
         - Create PersistentVolume (CSI driver, volume handle)
         - Create PersistentVolumeClaim
         - Create CloudNativePG Cluster CR (referencing PVC)
         - Watch CNPG Cluster status → update Database CRD status
         - Generate and store credentials in K8s Secret
         - Set status.connectionString
P6-04  Implement DatabaseController (for engine: redis):
         - Same flow but creates Redis CR instead of CNPG
P6-05  Implement database deletion:
         - Delete service operator CR
         - Delete PVC/PV
         - Call Hetzner API: delete Volume
P6-06  Implement database linking to apps:
         - When Application references a Database
         - Inject env vars (HOST, PORT, USER, PASSWORD, NAME, URL)
         - Mount credentials Secret as env vars in Deployment
P6-07  Backend API: Database CRUD
         POST   /projects/:id/databases
         GET    /projects/:id/databases
         GET    /projects/:id/databases/:dbId
         DELETE /projects/:id/databases/:dbId
P6-08  Backend API: Database connection info
         GET /projects/:id/databases/:dbId/connection
P6-09  Backend API: Link database to app
         POST /projects/:id/apps/:appId/databases
         DELETE /projects/:id/apps/:appId/databases/:dbId
P6-10  Frontend: Database pages (list, create, detail) with real API
P6-11  Frontend: Connection info with copy buttons
P6-12  Frontend: Database linking in app env vars tab
P6-13  Frontend: Database metrics (basic CPU/RAM/connections)
```

### Test Cases
```
T6-01  Create Database (postgres) → Hetzner Volume created
T6-02  Hetzner Volume → PV/PVC created correctly
T6-03  CNPG Cluster CR created → PostgreSQL pod running
T6-04  Connection string is valid and accessible from pods
T6-05  Link database to app → env vars injected into pod
T6-06  Delete Database → PostgreSQL deleted → Volume deleted at Hetzner
T6-07  Create Database (redis) → Redis pod running
T6-08  Redis connection works from app pods
T6-09  Frontend create database flow works end-to-end
T6-10  Frontend shows connection string with working copy button
```

### Operators Installed
```
+ CloudNativePG (postgresql)
+ Redis Operator (redis)
```

### Definition of Done
- [ ] Create Postgres via UI → running in 2 minutes on Hetzner Volume
- [ ] Create Redis via UI → running in 1 minute
- [ ] Link to app → env vars auto-injected
- [ ] Delete → everything cleaned up including Hetzner Volume

---

## Phase 7: MySQL + MongoDB + KV Store (Week 14)

**Goal:** Additional database engines.

### Tasks

```
P7-01  Install MySQL Operator (Oracle MySQL Operator) via Helm
P7-02  Install MongoDB Community Operator via Helm
P7-03  Install NATS Operator via Helm
P7-04  Extend DatabaseController for engine: mysql
P7-05  Extend DatabaseController for engine: mongodb
P7-06  Implement KeyValueStoreController:
         - Create NATS cluster with JetStream KV
         - Store credentials
P7-07  Frontend: MySQL and MongoDB options in database creation
P7-08  Frontend: KV Store page
P7-09  Update Helm chart to optionally install these operators
```

### Test Cases
```
T7-01  Create MySQL database → MySQL pod on Hetzner Volume
T7-02  Create MongoDB database → MongoDB pod on Hetzner Volume
T7-03  Create KV Store → NATS KV accessible
T7-04  All connection strings work from app pods
T7-05  Delete each type → clean up volumes
```

### Operators Installed
```
+ MySQL Operator
+ MongoDB Community Operator
+ NATS Operator (for KV store)
```

---

## Phase 8: Storage + Backups (Week 15)

**Goal:** S3 buckets + volume management + automated backups.

### Tasks

```
P8-01  Implement ObjectStoreController:
         - Watch ObjectStore CRD
         - Call Hetzner S3 API: create bucket
         - Generate S3 access credentials
         - Store credentials in K8s Secret
         - Update status with endpoint, access key
P8-02  Implement BackupPolicyController:
         - Watch BackupPolicy CRD
         - Create K8s CronJob:
           - For postgres: pg_dump → upload to S3
           - For mysql: mysqldump → upload to S3
           - For mongodb: mongodump → upload to S3
         - Track backup history in status
P8-03  Implement backup restore:
         - Download from S3 → restore into database
P8-04  Backend API: ObjectStore CRUD
P8-05  Backend API: BackupPolicy CRUD
P8-06  Backend API: List backups, trigger manual backup, restore
P8-07  Backend API: Link storage to app (inject S3 env vars)
P8-08  Frontend: S3 Buckets pages (real API)
P8-09  Frontend: Volume management
P8-10  Frontend: Backup list, manual backup, restore flow
P8-11  Frontend: Link storage to app
```

### Test Cases
```
T8-01  Create ObjectStore → Hetzner S3 bucket created
T8-02  S3 credentials work (can upload/download objects)
T8-03  Link storage to app → S3_ENDPOINT etc injected
T8-04  BackupPolicy creates CronJob
T8-05  CronJob runs and uploads backup to S3
T8-06  Restore from backup works
T8-07  Delete ObjectStore → bucket deleted
```

---

## Phase 9: Networking (Week 16-17)

**Goal:** Firewalls, DNS, Floating IPs, Load Balancers, VPN, API Gateway.

### Tasks

```
P9-01  Implement FirewallController:
         - Watch Firewall CRD
         - Call Hetzner Firewall API: create/update rules
         - Apply firewall to cluster servers
P9-02  Implement NetworkController:
         - Watch Network CRD
         - Call Hetzner Network API: create private network + subnets
P9-03  Implement FloatingIPController:
         - Watch FloatingIP CRD
         - Call Hetzner API: create floating IP
         - Assign to specified server
P9-04  Implement LoadBalancerController:
         - Watch LoadBalancer CRD
         - Call Hetzner API: create LB
         - Configure targets (services → LB)
P9-05  Implement DNSZoneController + DNSRecordController:
         - Watch DNSZone/DNSRecord CRDs
         - Call Hetzner DNS API
P9-06  Implement VPNPeerController:
         - Watch VPNPeer CRD
         - Deploy WireGuard pod
         - Generate peer configuration
         - Output: peer config file for client
P9-07  Implement GatewayController:
         - Watch Gateway CRD
         - Create Traefik IngressRoute with path-based routing
         - Add middleware (rate limit, CORS, JWT auth)
P9-08  Backend API: All networking CRUD endpoints
P9-09  Frontend: All networking tabs (real API)
```

### Test Cases
```
T9-01  Firewall CR → Hetzner Firewall created with correct rules
T9-02  Network CR → Hetzner Network + subnet created
T9-03  FloatingIP CR → Hetzner Floating IP created + assigned
T9-04  LoadBalancer CR → Hetzner LB created
T9-05  DNS records created via Hetzner DNS API
T9-06  VPN peer config generated, WireGuard connection works
T9-07  Gateway routes traffic to correct services
T9-08  Gateway rate limiting works
T9-09  All deletion cleanups work
```

---

## Phase 10: Auth/IAM - Keycloak + SSO/SAML/OIDC (Week 18)

**Goal:** Authentication service with simplified UI. Full enterprise SSO support.
User sees a simple auth page (like Supabase). Behind the scenes: Keycloak handles everything.
Supports JumpCloud, Okta, Azure AD, Google Workspace, any SAML/OIDC provider.

### Tasks

```
P10-01  Install Keycloak Operator via Helm
P10-02  Implement AuthRealmController:
          - Watch AuthRealm CRD
          - Create Keycloak CR (with PVC on Hetzner Volume)
          - Configure realm, clients, roles
          - Update status with auth endpoints (login URL, JWKS URL)
P10-03  Backend API: Auth realm management
          POST   /projects/:id/auth/realm (enable auth)
          GET    /projects/:id/auth/realm
          PUT    /projects/:id/auth/realm
P10-04  Backend API: User management within realm
          POST   /projects/:id/auth/users (invite user)
          GET    /projects/:id/auth/users
          PUT    /projects/:id/auth/users/:userId
          DELETE /projects/:id/auth/users/:userId
P10-05  Backend API: Role management
          POST   /projects/:id/auth/roles
          GET    /projects/:id/auth/roles
P10-06  Backend API: SSO provider configuration (SAML + OIDC)
          POST   /projects/:id/auth/sso
          GET    /projects/:id/auth/sso
          PUT    /projects/:id/auth/sso/:providerId
          DELETE /projects/:id/auth/sso/:providerId
P10-07  Implement SAML 2.0 Identity Provider integration:
          - User provides: SAML Metadata URL or XML
          - Or manual: Entity ID, SSO URL, Certificate
          - Galaxy configures Keycloak SAML IDP automatically
          - Supports: JumpCloud, Okta, Azure AD, OneLogin, PingOne
P10-08  Implement OIDC Identity Provider integration:
          - User provides: Issuer URL, Client ID, Client Secret
          - Galaxy configures Keycloak OIDC IDP automatically
          - Supports: Google Workspace, Azure AD, Auth0, Cognito
P10-09  Implement social login providers:
          - GitHub (OAuth)
          - Google (OIDC)
          - GitLab (OAuth)
          - Pre-configured, user just enters Client ID/Secret
P10-10  Backend API: API Key management for apps
          POST   /projects/:id/auth/api-keys
          GET    /projects/:id/auth/api-keys
          DELETE /projects/:id/auth/api-keys/:keyId
P10-11  Frontend: Simplified auth management (like Supabase Auth)
          - Simple UI: user sees "Users", "Roles", "Providers"
          - NO Keycloak admin console exposed
P10-12  Frontend: SSO provider setup wizard
          - Step 1: Select provider type (SAML / OIDC / Social)
          - Step 2: Select provider (JumpCloud, Okta, Azure AD, Google, custom)
          - Step 3: Provider-specific form with instructions
            - For JumpCloud: "Go to JumpCloud → SSO → SAML → paste metadata URL"
            - For Okta: "Go to Okta Admin → Applications → Add SAML App → paste..."
            - For Azure AD: "Go to Azure Portal → Enterprise Applications → ..."
          - Step 4: Test connection
          - Step 5: Enable / set as default
P10-13  Frontend: User invitation flow (email invite → SSO login)
P10-14  Frontend: Role-based access display
P10-15  Frontend: API Keys page (create, list, revoke)
P10-16  AuthRealm CRD enhancement:
          spec:
            ssoProviders:
              - name: "jumpcloud"
                type: saml
                metadataUrl: "https://sso.jumpcloud.com/saml2/..."
              - name: "google-workspace"
                type: oidc
                issuerUrl: "https://accounts.google.com"
                clientId: "xxx"
                clientSecret:
                  secretRef: { name: google-oidc, key: secret }
            socialProviders:
              - github: { clientId: "xxx", clientSecret: ... }
              - google: { clientId: "xxx", clientSecret: ... }
            defaultProvider: "jumpcloud"   # which SSO is default
            allowEmailPassword: true       # also allow email/password login
```

### Test Cases
```
T10-01  AuthRealm CR → Keycloak pod running on Hetzner Volume
T10-02  SAML IDP configured → user can login via SAML
T10-03  OIDC IDP configured → user can login via OIDC
T10-04  Social login (GitHub) works
T10-05  User invitation email sent → user can login
T10-06  API keys work for programmatic access
T10-07  Frontend SSO wizard creates working SSO connection
T10-08  JumpCloud SAML integration works end-to-end
T10-09  Role-based access: admin vs developer vs viewer
```

### Operators Installed
```
+ Keycloak Operator
```

---

## Phase 11: Observability (Week 19-20)

**Goal:** Monitoring, logging, alerting - all built-in.

### Tasks

```
P11-01  Install kube-prometheus-stack (Prometheus + Grafana) via Helm
P11-02  Install Loki + Promtail via Helm
P11-03  Implement MonitoringController:
          - Watch Monitoring CRD
          - Configure Prometheus scrape targets for project apps
          - Create Grafana dashboards (auto-generated per project)
          - Update status with Grafana URL
P11-04  Implement LogPipelineController:
          - Watch LogPipeline CRD
          - Configure Promtail to collect logs from project namespace
P11-05  Implement AlertRuleController:
          - Watch AlertRule CRD
          - Create PrometheusRule CR
P11-06  Backend API: Metrics proxy (Prometheus query)
P11-07  Backend API: Log query proxy (Loki query)
P11-08  Backend API: Alert CRUD
P11-09  Frontend: Embedded Grafana dashboard (iframe or API-based charts)
P11-10  Frontend: Log viewer with service filter + level filter
P11-11  Frontend: Alert management (create, view active, history)
P11-12  Create default alert rules:
          - App down (0 ready pods)
          - High CPU (> 85% for 5min)
          - High memory (> 90%)
          - Disk almost full (> 80%)
          - High error rate (> 5% 5xx responses)
          - SSL cert expiring (< 7 days)
```

### Operators Installed
```
+ Prometheus (via kube-prometheus-stack)
+ Grafana (via kube-prometheus-stack)
+ Loki (logging)
+ Promtail (log collection)
```

---

## Phase 12: Planets - Node Scaling (Week 21-22)

**Goal:** Add/remove nodes with "Add a Planet" UX.

### Tasks

```
P12-01  Implement PlanetController:
          - Watch Planet CRD
          - Call Hetzner API: create server (spec.type, spec.region)
          - Run cloud-init: install k3s agent, join cluster
          - Wait for node Ready
          - Update status (Creating → Joining → Ready)
P12-02  Implement Planet deletion:
          - Cordon node (kubectl cordon)
          - Drain node (kubectl drain)
          - Delete k3s node (kubectl delete node)
          - Call Hetzner API: delete server
P12-03  Backend API: Planet CRUD
          POST   /projects/:id/planets
          GET    /projects/:id/planets
          DELETE /projects/:id/planets/:planetId
P12-04  Backend API: Node metrics
          GET /projects/:id/planets/:planetId/metrics
P12-05  Frontend: Planets page (real API)
P12-06  Frontend: Add a Planet modal (working)
P12-07  Frontend: Remove Planet with drain progress
P12-08  Frontend: Per-planet CPU/RAM/Pod metrics
```

### Test Cases
```
T12-01  Create Planet CR → Hetzner server created
T12-02  Server runs cloud-init → k3s agent joins cluster
T12-03  Node shows as Ready in kubectl get nodes
T12-04  Pods schedule on new node
T12-05  Delete Planet → node drained → server deleted at Hetzner
T12-06  Whole flow takes < 2 minutes
T12-07  Frontend shows real-time planet status
```

---

## Phase 13: Message Queues (Week 23)

**Goal:** NATS/RabbitMQ for async microservice communication.

### Tasks

```
P13-01  Implement MessageQueueController:
          - Watch MessageQueue CRD
          - For NATS: create NATS CR via NATS Operator
          - For RabbitMQ: install RabbitMQ Operator + create CR
          - Configure streams/queues
          - Store credentials
P13-02  Backend API: MessageQueue CRUD
P13-03  Frontend: Message queue management page
P13-04  Support linking queue to app (inject NATS_URL etc)
```

---

## Phase 14: VPN Peering + Cloud Connections (Week 23-24)

**Goal:** WireGuard VPN + Hybrid Cloud tunnels to AWS/GCP/Azure/on-prem.

### Tasks

```
P14-01  Implement VPNPeerController (WireGuard deployment)
P14-02  Generate peer configuration files
P14-03  Backend API: VPN CRUD
P14-04  Frontend: VPN management page
P14-05  Define CloudConnector CRD (api/v1alpha1/cloudconnector_types.go):
          - spec.provider: aws | gcp | azure | custom
          - spec.type: ipsec | wireguard
          - spec.remote: gatewayIP, cidr, presharedKey
          - spec.local: cidr
          - spec.access: allowedNamespaces, allowedCIDRs
          - spec.healthCheck: enabled, remoteIP, interval
          - status.phase: Pending | Connecting | Connected | Failed
          - status.tunnelIP, status.latency, status.lastHandshake
P14-06  Implement CloudConnectorController:
          - Watch CloudConnector CRD
          - Deploy StrongSwan pod (for IPsec) or WireGuard pod
          - Configure IPsec tunnel parameters from spec
          - Create K8s routes: remote CIDR → tunnel pod
          - Create NetworkPolicy: only allowed namespaces can use tunnel
          - Health check loop: ping remote IP, update status
          - Update status (Pending → Connecting → Connected)
P14-07  Install StrongSwan (IPsec) container image in registry
P14-08  Create tunnel configuration templates:
          - AWS Site-to-Site VPN (IKEv2, AES-256, SHA-256)
          - GCP Cloud VPN (IKEv2)
          - Azure VPN Gateway (IKEv1/v2)
          - Custom IPsec (user-defined parameters)
P14-09  Implement tunnel deletion:
          - Delete StrongSwan/WG pod
          - Remove routes
          - Remove NetworkPolicy
P14-10  Backend API: CloudConnector CRUD
          POST   /projects/:id/cloud-connections
          GET    /projects/:id/cloud-connections
          GET    /projects/:id/cloud-connections/:connId
          PUT    /projects/:id/cloud-connections/:connId
          DELETE /projects/:id/cloud-connections/:connId
P14-11  Backend API: CloudConnector health status
          GET /projects/:id/cloud-connections/:connId/health
P14-12  Frontend: Cloud Connections page (list, status, latency)
P14-13  Frontend: Create Cloud Connection wizard (provider select → config form)
P14-14  Frontend: Connection detail (health, used-by apps, configure)
P14-15  Support cloudConnectors reference in Application CRD:
          - App declares cloudConnectors: ["aws-production"]
          - Operator adds network route annotation to pods
          - Pods can reach remote CIDRs through tunnel
```

### Test Cases
```
T14-01  Create CloudConnector CR → StrongSwan pod deployed
T14-02  IPsec tunnel establishes with test VPN endpoint
T14-03  Ping health check passes through tunnel
T14-04  App pod in allowed namespace can reach remote CIDR
T14-05  App pod in non-allowed namespace CANNOT reach remote CIDR
T14-06  Delete CloudConnector → pod deleted, routes removed
T14-07  Status shows Connected + latency
T14-08  Frontend shows real-time connection health
T14-09  WireGuard type tunnel works as alternative to IPsec
```

---

## Phase 15: Billing Engine (Week 25)

**Goal:** Track Hetzner resource usage, show costs transparently.

### Tasks

```
P15-01  Implement BillingController:
          - Watch all resource CRDs
          - Create UsageRecord CRDs per resource
          - Track: start time, type, Hetzner resource ID, size
P15-02  Create cost calculator service:
          - Map Hetzner resource types to prices
          - Calculate monthly cost per resource
          - Aggregate per project
P15-03  Backend API: Billing endpoints
          GET /projects/:id/billing/current (current month)
          GET /projects/:id/billing/history
P15-04  Frontend: Billing page with real cost data
P15-05  Frontend: Cost estimate on every create action
          ("This will cost ~€X.XX/mo")
P15-06  Create pricing calculator for landing page
```

---

## Phase 16: zen CLI (Week 25-26)

**Goal:** Beautiful, interactive CLI using Charm ecosystem (bubbletea + lipgloss + bubbles + huh).

**Stack:** cobra (commands), bubbletea (TUI), lipgloss (styling), bubbles (components), huh (forms), glamour (markdown)

### Tasks

```
P16-01  Create cobra CLI skeleton (cmd/zen/main.go) with root command + version
P16-02  Set up Charm dependencies: bubbletea, lipgloss, bubbles, huh, glamour, log
P16-03  Create shared TUI theme: emerald green primary, consistent styling across all commands
P16-04  Create shared TUI components: styled table, status badge (● ○), progress bar, spinner
P16-05  Create ASCII art banner (zen version) with version info
P16-06  Implement: zen (no args) → full-screen interactive dashboard (bubbletea app)
           - Apps table with status, replicas, CPU, RAM
           - Databases summary
           - Planets summary with resource bars
           - Keyboard navigation (Tab sections, Enter detail, / search, l logs, q quit)
P16-07  Implement: zen install → interactive wizard (huh forms)
           - Hetzner token input (masked)
           - Domain input (with validation)
           - Admin email input
           - Animated progress with spinners → checkmarks + timing
P16-08  Implement: zen login / zen auth status
           - Token stored in ~/.zen/config.yaml
P16-09  Implement: zen project create/list/switch/delete
           - create: interactive name + plan selection
           - list: styled table
           - delete: type-name-to-confirm prompt
P16-10  Implement: zen deploy → auto-detect language + Dockerfile + port
           - Interactive form: app name, replicas, expose toggle, domain
           - Build progress with Docker step-by-step output
           - Success box with URL + next steps
P16-11  Implement: zen deploy --image <img> (deploy existing Docker image)
P16-12  Implement: zen deploy --github <repo> (deploy from GitHub)
P16-13  Implement: zen apps → styled table with all apps
P16-14  Implement: zen scale <app> <n> → with confirmation and progress
P16-15  Implement: zen redeploy <app> → trigger redeploy with progress
P16-16  Implement: zen rollback <app> → select from version list
P16-17  Implement: zen logs <app> --follow → color-coded streaming logs
           - INF=green, WRN=yellow, ERR=red, DBG=gray
           - Auto-format JSON logs into human-readable lines
           - Tab to switch instances, / to filter, f for full JSON
P16-18  Implement: zen status → rich overview (apps, DBs, planets, cost)
           - Planet resource bars (CPU/RAM with color)
           - Inline status badges
P16-19  Implement: zen top → real-time resource monitor (htop-style)
           - Auto-refresh every 2s
           - CPU/RAM/Net per app
           - DB connections/storage/QPS
           - Keyboard: sort, filter, navigate
P16-20  Implement: zen events --follow → live color-coded event stream
P16-21  Implement: zen db list → styled table
P16-22  Implement: zen db create → interactive wizard (engine, name, size selection)
P16-23  Implement: zen db connect <name> → auto port-forward + open psql/redis-cli/mongosh
P16-24  Implement: zen db backup/restore <name>
P16-25  Implement: zen domain add/list
P16-26  Implement: zen planet list → table with resource usage bars
P16-27  Implement: zen planet add → interactive size picker (huh selection)
P16-28  Implement: zen planet remove <name> → drain progress + delete
P16-29  Implement: zen wizard → interactive menu for everything (huh selection)
P16-30  Implement: zen config / zen config set
P16-31  Implement: zen update → self-update with progress bar
P16-32  Implement: zen restore → disaster recovery from S3 backup
P16-33  Shell completions: bash, zsh, fish (cobra built-in + custom descriptions)
P16-34  Build release binaries: goreleaser config (Linux amd64/arm64, macOS amd64/arm64, Windows)
P16-35  Create install script: curl -sfL https://get.freezenith.com | sh
           - Detects OS/arch, downloads correct binary
           - Adds to PATH, verifies checksum
P16-36  Create Homebrew formula: brew install freezenith/tap/zen
P16-37  Publish to AUR (Arch Linux), Scoop (Windows)
```

### Test Cases

```
T16-01  `zen version` shows ASCII banner + version info
T16-02  `zen` (no args) opens TUI dashboard, keyboard navigation works
T16-03  `zen install` wizard validates Hetzner token before proceeding
T16-04  `zen deploy` in Go project auto-detects Dockerfile, port, language
T16-05  `zen deploy` in Node project auto-detects package.json, port 3000
T16-06  `zen logs` color-codes INF/WRN/ERR correctly
T16-07  `zen logs` auto-formats JSON log lines into readable format
T16-08  `zen top` refreshes every 2s without flicker
T16-09  `zen db connect` opens correct shell (psql for PG, redis-cli for Redis, mongosh for Mongo)
T16-10  `zen export` → `zen apply` round-trip produces identical state
T16-11  `zen diff` shows correct additions/removals/changes with colors
T16-12  `zen wizard` presents all options and launches correct sub-wizard
T16-13  Install script works on Ubuntu 22.04, macOS 14, Windows 11 (WSL)
T16-14  Shell completions work in bash, zsh, fish
T16-15  `zen update` downloads and replaces binary, preserves config
```

### Definition of Done
- [ ] Every command has beautiful, styled output (not plain text)
- [ ] Interactive TUI dashboard works with keyboard navigation
- [ ] All wizards use huh forms (not raw prompts)
- [ ] Logs are color-coded and auto-formatted
- [ ] `zen top` is a real-time monitor
- [ ] Install script works on all platforms
- [ ] Shell completions for bash/zsh/fish

---

## Phase 17: Landing Page + Documentation (Week 26)

**Goal:** Public website and comprehensive docs.

### Tasks

```
P17-01  Build freezenith.com landing page (in web/ or separate)
P17-02  Create docs site (Docusaurus or Next.js based)
P17-03  Write: Getting Started guide
P17-04  Write: Installation guide (all providers)
P17-05  Write: Deploying your first app
P17-06  Write: Database management
P17-07  Write: Custom domains + SSL
P17-08  Write: Scaling with Planets
P17-09  Write: Architecture overview
P17-10  Write: API reference (auto-generated from OpenAPI)
P17-11  Write: CRD reference (auto-generated)
P17-12  Write: CLI reference
P17-13  Write: Contributing guide
P17-14  Write: FAQ
P17-15  SEO optimization for key terms
```

---

## Phase 18: Production Hardening (Week 27-28)

**Goal:** Production ready. Security audit prep.

### Tasks

```
P18-01  Security: Pod Security Standards enforcement
P18-02  Security: Network policy audit
P18-03  Security: Secret encryption at rest
P18-04  Security: RBAC audit (least privilege)
P18-05  Security: Container image scanning (Trivy)
P18-06  HA: Operator runs 2+ replicas with leader election
P18-07  HA: API server runs 2+ replicas
P18-08  E2E test suite: install → deploy app → add DB → link → verify → delete
P18-09  Performance: Benchmark 100 apps in single cluster
P18-10  Performance: Benchmark 50 concurrent database creations
P18-11  API versioning: v1alpha1 → review for v1beta1
P18-12  Graceful degradation: operator handles Hetzner API outages
P18-13  Disaster recovery documentation
P18-14  Release process: goreleaser config, signed artifacts
P18-15  Release v1.0.0
```

---

## Phase 19: CNCF Preparation (Week 30-31)

**Goal:** Community readiness, governance, security process.

### Tasks

```
P19-01  Finalize GOVERNANCE.md (maintainers, decision process)
```

---

## Phase 19.5: Management Plane + Mission Control (Week 30-32)

**Goal:** CAPI-based cluster management + platform operator panel.

### Tasks

```
P19M-01  Design CAPI cluster templates for Hetzner (HetznerCluster, MachineDeployment CRDs)
P19M-02  Create `zen install` flow: provision CX22 management server via Hetzner API
P19M-03  Install k3s on management server (automated via cloud-init)
P19M-04  Install CAPI + CAPH (Cluster API Provider Hetzner) on management k3s
P19M-05  Implement CAPI cluster creation: workload cluster from template
P19M-06  Implement CAPI K8s version upgrade (patch MachineDeployment.spec.version)
P19M-07  Implement rolling upgrade monitoring (watch Machine status, report progress)
P19M-08  Implement CAPI node scaling (add/remove MachineDeployment replicas)
P19M-09  Create Mission Control Go API server (cmd/zenith-mc/main.go)
P19M-10  Implement /api/clusters endpoints (list, get, create, upgrade, scale)
P19M-11  Implement /api/modules endpoints (list installed, check updates, upgrade)
P19M-12  Implement module updater: Helm upgrade for each operator (CNPG, Redis, etc.)
P19M-13  Implement /api/updates endpoint: poll freezenith.com for platform releases
P19M-14  Implement platform updater: apply new CRDs + helm upgrade zenith
P19M-15  Implement /api/tenants endpoints (list, usage, quotas, suspend)
P19M-16  Implement /api/infrastructure endpoint (Hetzner resource inventory + cost)
P19M-17  Implement /api/audit endpoint (log all admin actions)
P19M-18  Create mission-control Next.js app (apps/mission-control/)
P19M-19  Build Dashboard page (cluster overview, update notifications, recent activity)
P19M-20  Build Clusters page (list, detail, upgrade wizard, progress view)
P19M-21  Build Modules page (list with versions, update detail, bulk update)
P19M-22  Build Updates page (platform version, changelog, upgrade, history, auto-update settings)
P19M-23  Build Tenants page (list, resource usage, quotas)
P19M-24  Build Infrastructure page (Hetzner resources, cost breakdown, capacity planning)
P19M-25  Build Audit Log page (filterable, exportable)
P19M-26  Build State page (complete snapshot: platform, clusters, modules, Hetzner resources, backups)
P19M-27  Build Welcome Wizard (first-time setup: region, size, confirm, progress, done)
P19M-28  Implement platform rollback: helm rollback zenith to previous version
P19M-29  Implement health verification after upgrades (API + operator + web probes)
P19M-30  Implement SQLite state store (/var/lib/zenith-mc/state.db)
P19M-31  Implement audit log persistence in SQLite (actor, action, timestamp, cluster, details)
P19M-32  Implement module inventory tracking in SQLite (what version on which cluster)
P19M-33  Implement automated state backup: etcd snapshots + SQLite → Hetzner Object Storage every 6h
P19M-34  Implement state restore: `zen restore --from s3://zenith-backups/` (recovery from mgmt server loss)
P19M-35  Implement inter-cluster communication: Zenith Operator → Mission Control API for infra requests
P19M-36  Implement quota validation in Mission Control (check limits before provisioning)
P19M-37  Create Dockerfile for Mission Control (API + web, single image)
P19M-38  Add Mission Control to Helm chart (deployed on management cluster only)
P19M-39  Update `zen install` to two-phase: CLI creates mgmt plane → browser wizard creates platform
P19M-40  Write admin guide: "Managing Your Zenith Platform" (docs/admin-guide.md)
```

### Test Cases

```
T19M-01  `zen install` creates management server + k3s + CAPI + Mission Control end-to-end (~3 min)
T19M-02  Welcome Wizard creates workload cluster from Mission Control UI (~5 min)
T19M-03  CAPI creates workload cluster with correct K8s version and node count
T19M-04  K8s upgrade via CAPI: rolling upgrade completes, zero pod downtime
T19M-05  K8s upgrade can be paused and resumed safely
T19M-06  Module upgrade (CNPG): helm upgrade succeeds, existing databases unaffected
T19M-07  Platform upgrade: new Zenith version applied, CRDs updated, pods healthy
T19M-08  Platform rollback: reverts to previous version successfully
T19M-09  Mission Control auth is separate from user-facing auth
T19M-10  Audit log captures all admin actions with timestamp and actor in SQLite
T19M-11  Infrastructure page shows accurate Hetzner cost (verified against Hetzner dashboard)
T19M-12  Tenant suspension blocks API access for that tenant
T19M-13  Upgrade notification appears within 6 hours of new release on freezenith.com
T19M-14  State page shows complete snapshot of all clusters, modules, resources
T19M-15  Automated backup: etcd + SQLite backed up to S3 every 6 hours
T19M-16  Disaster recovery: destroy mgmt server → `zen restore` → full recovery in ~5 min
T19M-17  Zenith Operator on workload cluster communicates with Mission Control for infra requests
T19M-18  Quota check: Mission Control rejects Planet creation when tenant exceeds limit
```

### Definition of Done
- [ ] Two-phase install works: CLI → mgmt plane → Welcome Wizard → platform created
- [ ] K8s upgrades via CAPI: rolling, zero-downtime, pausable
- [ ] All modules upgradeable from Mission Control UI
- [ ] Platform updates from freezenith.com work end-to-end
- [ ] Mission Control panel functional with all 8 pages (Dashboard, Clusters, Modules, Updates, Tenants, Infra, State, Audit)
- [ ] Welcome Wizard works on first login
- [ ] SQLite state store: audit log, module inventory, tenant metadata
- [ ] Automated backups: etcd + SQLite → S3 every 6 hours
- [ ] Disaster recovery: full restore from backup in ~5 minutes
- [ ] Rollback works for both modules and platform
- [ ] Inter-cluster communication: Zenith Operator ↔ Mission Control API

---

## Phase 20: CNCF Preparation (Week 33-34)

**Goal:** Apply for CNCF Sandbox.

### Tasks

```
P20-01  Finalize GOVERNANCE.md (maintainers, decision process)
P20-02  Finalize SECURITY.md (vulnerability reporting process)
P20-03  Create adopters.md (list of production users)
P20-04  Create roadmap.md (public roadmap)
P20-05  Write CNCF Sandbox proposal
P20-06  Identify and contact 2 TOC sponsors
P20-07  Create project presentation (for TOC review)
P20-08  Submit CNCF Sandbox application
P20-09  Prepare for due diligence review
P20-10  HN launch post: "Zenith: Open-source PaaS for Kubernetes"
P20-11  Dev.to / Medium launch article
P20-12  Submit KubeCon talk proposal
```

---

## Phase 16.5: GitOps (integrated into zen CLI, Week 25-26)

**Goal:** Full GitOps support - export, import, sync, drift detection.

### Tasks

```
P16G-01  Implement `zen export project <name>` → outputs all CRDs as YAML
P16G-02  Implement `zen export` for individual resources (app, db, domain, etc.)
P16G-03  Implement `zen export --dir` → directory structure (apps/, databases/, networking/, etc.)
P16G-04  Implement `zen apply -f <file|dir>` → applies CRDs to cluster
P16G-05  Implement `zen diff -f <file|dir>` → shows diff between file and live state
P16G-06  Implement `zen apply --dry-run` → shows what would change without applying
P16G-07  Create GitSync CRD (repo, branch, path, interval, prune, webhook)
P16G-08  Implement GitSyncController - polls git repo, applies changes
P16G-09  Add GitHub/GitLab webhook endpoint for instant sync (POST /api/v1/webhooks/gitsync)
P16G-10  Implement drift detection - compare live CRDs vs git, report differences
P16G-11  Implement lock mode - resources managed by GitSync show 🔒 in UI, reject UI edits
P16G-12  Add GitOps status page in frontend (sync status, last sync, drift warnings)
P16G-13  Add "Export as YAML" button on every resource detail page in UI
P16G-14  Add project-level "Enable GitOps" toggle in Settings → shows git repo config
P16G-15  Write ArgoCD integration guide (docs/gitops-argocd.md)
P16G-16  Write FluxCD integration guide (docs/gitops-fluxcd.md)
P16G-17  Support Kustomize overlays for environment promotion (staging/production)
```

### Test Cases

```
T16G-01  `zen export project` outputs valid YAML that can be `zen apply`'d to clean cluster
T16G-02  `zen export --dir` creates correct directory structure
T16G-03  `zen diff` detects added/removed/changed resources
T16G-04  GitSync polls repo and applies new CRDs within interval
T16G-05  GitSync webhook triggers immediate sync
T16G-06  Drift detection catches manual UI changes when git-first mode enabled
T16G-07  Lock mode prevents UI edits on GitSync-managed resources
T16G-08  ArgoCD can sync Zenith CRDs from git repo (integration test)
```

### Definition of Done
- [ ] Full export/import cycle works (export → edit → apply)
- [ ] GitSync CRD works with GitHub repos
- [ ] Drift detection and lock mode functional
- [ ] Frontend shows GitOps status and export buttons

---

## Summary

```
Phase  0: Project Setup                    20 tasks     Week 1
Phase  1: Frontend Shell + Design System   99 tasks     Week 2-3
Phase  2: Backend API + Auth               33 tasks     Week 4-5
Phase  3: CRDs + Operator Foundation       42 tasks     Week 6-7
Phase  4: App Deploy (Docker)              13 tasks     Week 8-9
Phase  5: Build Pipeline (GitHub)          13 tasks     Week 10-11
Phase  6: PostgreSQL + Redis               13 tasks     Week 12-13
Phase  7: MySQL + MongoDB + KV             9 tasks      Week 14
Phase  8: Storage + Backups                11 tasks     Week 15
Phase  9: Networking                       9 tasks      Week 16-17
Phase 10: Auth/IAM (Keycloak + SSO)        16 tasks     Week 18
Phase 11: Observability                    12 tasks     Week 19-20
Phase 12: Planets (Node Scaling)           8 tasks      Week 21-22
Phase 13: Message Queues                   4 tasks      Week 23
Phase 14: VPN + Cloud Connections          15 tasks     Week 23-24
Phase 15: Billing                          6 tasks      Week 24
Phase 16: zen CLI (Charm TUI)              37 tasks     Week 25-26
Phase 16.5: GitOps                         17 tasks     Week 26-27
Phase 17: Landing Page + Docs              15 tasks     Week 27-28
Phase 18: Production Hardening             15 tasks     Week 29-30
Phase 19: CNCF Preparation                 12 tasks     Week 31
Phase 19.5: Management Plane + Mission Control 40 tasks     Week 31-33
Phase 20: CNCF Submission                  12 tasks     Week 34-35
──────────────────────────────────────────────────────────────
TOTAL                                      ~456 tasks    35 weeks
```

Each task expands to 2-5 subtasks during implementation.
Total with subtasks: **~1200-1700 units of work.**

Start with Phase 0 + 1 (project setup + full frontend).
Backend and operators follow once the UI design is validated.
