# Capability: Admin Dashboard

## Purpose
Provide the operator admin panel (Mission Control) for managing clusters, tenants, modules, infrastructure, platform updates, audit logs, and settings across the entire Zenith installation.

## Requirements

### Requirement: Dashboard Stats
The system SHALL provide aggregated stats for the admin dashboard: cluster count, tenant count, monthly cost, and pending updates.

#### Scenario: Get dashboard stats
- **WHEN** an admin GETs `/api/v1/admin/dashboard/stats`
- **THEN** the system returns cluster/tenant/cost/update counts

### Requirement: Cluster Management
The system SHALL support CRUD for CAPI-managed Kubernetes clusters. Clusters are stored as `cluster.x-k8s.io/v1beta1` CRDs in the `zenith-system` namespace. Admins can upgrade K8s versions.

#### Scenario: List clusters
- **WHEN** an admin GETs `/api/v1/admin/clusters`
- **THEN** all CAPI clusters are returned with name, region, K8s version, node count, CPU/RAM usage

#### Scenario: Upgrade K8s version
- **WHEN** an admin POSTs to `/api/v1/admin/clusters/:name/upgrade` with target version
- **THEN** the cluster upgrade is initiated

### Requirement: Tenant Management
The system SHALL derive tenants from Project CRDs. Admins can list tenants, view details (plan, app/db counts, resource usage), and suspend tenants.

#### Scenario: List tenants
- **WHEN** an admin GETs `/api/v1/admin/tenants`
- **THEN** all tenants are returned with plan, resource usage, and status

#### Scenario: Suspend tenant
- **WHEN** an admin POSTs to `/api/v1/admin/tenants/:id/suspend`
- **THEN** the tenant is suspended and resources are frozen

### Requirement: Module Management
The system SHALL manage installed K8s modules/operators. Admins can list modules (with installed/latest versions), install, uninstall, update individual modules, or update all at once. 11 modules are pre-seeded.

#### Scenario: Update all modules
- **WHEN** an admin POSTs to `/api/v1/admin/modules/update-all`
- **THEN** all outdated modules are updated to their latest versions

### Requirement: Audit Log
The system SHALL maintain a paginated audit log of admin actions with actor, action, cluster, and timestamp. Supports limit/offset pagination.

#### Scenario: Query audit log
- **WHEN** an admin GETs `/api/v1/admin/audit` with limit and offset
- **THEN** paginated audit entries are returned

### Requirement: Platform Updates
The system SHALL support checking for available platform updates, applying updates, and viewing update history.

#### Scenario: Check for updates
- **WHEN** an admin GETs `/api/v1/admin/updates/check`
- **THEN** the system returns available update info (version, features, breaking changes)

#### Scenario: Apply update
- **WHEN** an admin POSTs to `/api/v1/admin/updates/apply`
- **THEN** the platform update is applied

### Requirement: Infrastructure Overview
The system SHALL provide a summary of Hetzner infrastructure: servers, volumes, load balancers, and monthly cost.

#### Scenario: Get infrastructure
- **WHEN** an admin GETs `/api/v1/admin/infrastructure`
- **THEN** resource counts and cost breakdown are returned

### Requirement: Platform State
The system SHALL expose full platform state including version, installed date, K8s version, domain/TLS status, and module version matrix per cluster. State can be exported as JSON.

#### Scenario: Export state
- **WHEN** an admin GETs `/api/v1/admin/state/export`
- **THEN** the full state is returned as a downloadable JSON file

### Requirement: Platform Settings
The system SHALL support getting and updating platform settings (name, domain, backup config). Supports both full PUT and partial PATCH updates.

#### Scenario: Update settings
- **WHEN** an admin PUTs new settings to `/api/v1/admin/settings`
- **THEN** the platform settings are updated

### Requirement: Admin UI (Mission Control)
Mission Control SHALL provide 10 pages: Dashboard, Clusters, Cluster Detail, Modules, Tenants, Infrastructure, Updates, State, Audit, Settings. Each page shows relevant data with stat cards, tables, progress bars, and action buttons.

#### Scenario: Dashboard page
- **WHEN** an admin navigates to MC root `/`
- **THEN** 4 stat cards, cluster table, updates list, and audit activity are displayed
