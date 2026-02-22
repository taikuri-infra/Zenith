## MODIFIED Requirements

### Requirement: Deploy Engine Apps (V2)
The system SHALL support a second app model (Phase 2) backed by the deploy engine. These apps are created from Git repos, support deployments, env vars, and secrets. Stored in PostgreSQL or in-memory. Internal architecture uses Clean Architecture layers: handlers -> services -> ports <- adapters.

#### Scenario: Create V2 app
- **WHEN** a user POSTs to `/api/v1/apps` with repo URL and branch
- **THEN** a deploy-engine app is created with auto-detected framework and generated subdomain

#### Scenario: List user's apps
- **WHEN** a user GETs `/api/v1/apps`
- **THEN** only apps owned by the authenticated user are returned
