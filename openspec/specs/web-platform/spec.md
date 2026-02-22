# Capability: Web Platform UI

## Purpose
The user-facing Next.js dashboard where developers manage apps, databases, storage, deploy engine, settings, and all platform services scoped to their project.

## Requirements

### Requirement: Project Overview Dashboard
The Web Platform SHALL display a project overview at `/` with stat cards (apps, deploy engine running/building, databases, region, status), deploy engine card grid, legacy apps table, and databases list.

#### Scenario: View dashboard
- **WHEN** a user navigates to the Web Platform root
- **THEN** project stats, deploy engine cards, legacy apps, and databases are displayed

### Requirement: Deploy Engine Page
The Web Platform SHALL provide a dedicated Deploy page at `/deploy` showing app cards with status dots (green=running, amber=building, red=failed), framework labels, branch display, URLs, "Deploy from Git" modal, and delete with confirmation.

#### Scenario: Deploy from Git
- **WHEN** a user clicks "Deploy from Git" and fills repo URL + branch
- **THEN** a new deploy engine app is created and build begins

### Requirement: App Detail Page
The Web Platform SHALL provide an app detail page at `/apps/[id]` with tabs: Overview (details + quick links), Deployments (table + rollback), Releases (image versions + one-click Deploy), Logs (SSE build log viewer), Secrets (add/reveal/delete), Environment (CRUD env vars).

#### Scenario: View app logs
- **WHEN** a user navigates to the Logs tab of an active deployment
- **THEN** build logs stream in real-time via SSE in a terminal-style viewer

#### Scenario: One-click deploy from release
- **WHEN** a user clicks Deploy on a release entry
- **THEN** the pre-built image is deployed without rebuilding

### Requirement: Sidebar Navigation
The sidebar SHALL organize pages into sections: Overview, Deploy (Rocket icon), Compute (Apps, Databases, Storage), Networking (Gateway, Domains), Security (Auth, IAM), Observability (Monitoring, Registry), Infrastructure (Planets), and bottom items (Docs, Billing, Settings).

#### Scenario: Navigate via sidebar
- **WHEN** a user clicks a sidebar item
- **THEN** the corresponding page is displayed with the item highlighted as active

### Requirement: Settings Page
The Settings page SHALL show project name, plan badge, region, and a danger zone (delete project). Phase 6.5 adds tabs: API Keys, MFA, Webhooks, Sessions, Security, General.

#### Scenario: Delete project
- **WHEN** a user clicks delete in the danger zone and confirms
- **THEN** the project is deleted and user is redirected

### Requirement: Auth Flow
The Web Platform SHALL support JWT-based auth with login/register/OAuth (Google, GitHub). Tokens stored in localStorage (`zenith_access_token`, `zenith_refresh_token`). Auto-refresh on 401.

#### Scenario: Auto token refresh
- **WHEN** an API call returns 401 and a refresh token exists
- **THEN** the system attempts token refresh and retries the request

### Requirement: Placeholder Pages
Several pages (Monitoring, Gateway, Auth, IAM, Registry, Networking, Planets) SHALL display hardcoded mock data or placeholder content until backend APIs are connected.

#### Scenario: Monitoring page
- **WHEN** a user navigates to `/monitoring`
- **THEN** mock Grafana dashboards, Prometheus targets, and Loki logs are displayed
