# Capability: App Management

## Purpose
Manage applications on the Zenith platform — both legacy CRD-based apps and V2 deploy engine apps. Includes environment variables, encrypted secrets, and deployment history.

## Requirements

### Requirement: Legacy CRD-Based Apps
The system SHALL support CRUD for legacy apps scoped to a project. Apps are defined by name, image, replicas, port, environment variables, and domain. Data is stored as K8s CRDs via `MemoryClient`.

#### Scenario: Create legacy app
- **WHEN** a user POSTs to `/api/v1/projects/:id/apps` with name, image, replicas, port
- **THEN** the system creates a CRD-based app resource

#### Scenario: Redeploy legacy app
- **WHEN** a user POSTs to `/api/v1/projects/:id/apps/:name/redeploy`
- **THEN** the system triggers a redeployment of the app

### Requirement: Deploy Engine Apps (V2)
The system SHALL support a second app model (Phase 2) backed by the deploy engine. These apps are created from Git repos, support deployments, env vars, and secrets. Stored in PostgreSQL or in-memory.

#### Scenario: Create V2 app
- **WHEN** a user POSTs to `/api/v1/apps` with repo URL and branch
- **THEN** a deploy-engine app is created with auto-detected framework and generated subdomain

#### Scenario: List user's apps
- **WHEN** a user GETs `/api/v1/apps`
- **THEN** only apps owned by the authenticated user are returned

### Requirement: Environment Variables
The system SHALL support CRUD for per-app environment variables via `/api/v1/apps/:id/env`. Variables are stored as key-value pairs.

#### Scenario: Set env vars
- **WHEN** a user PUTs key-value pairs to `/api/v1/apps/:id/env`
- **THEN** the variables are stored and available to future deployments

#### Scenario: Delete env var
- **WHEN** a user DELETEs `/api/v1/apps/:id/env/:key`
- **THEN** the variable is removed

### Requirement: App Secrets
The system SHALL support encrypted secrets per app using AES-256-GCM. Secrets are stored encrypted in the database. Listing returns keys only; values are decrypted on explicit request.

#### Scenario: Create secret
- **WHEN** a user POSTs a key-value pair to `/api/v1/apps/:appId/secrets`
- **THEN** the value is encrypted with AES-256-GCM and stored

#### Scenario: List secrets (keys only)
- **WHEN** a user GETs `/api/v1/apps/:appId/secrets`
- **THEN** only secret keys are returned (no values)

#### Scenario: Reveal secret value
- **WHEN** a user GETs `/api/v1/apps/:appId/secrets/:key/value`
- **THEN** the value is decrypted and returned

### Requirement: Deployment History
The system SHALL track all deployments per app with status, git SHA, build/deploy logs, and timestamps. Users can list and inspect individual deployments.

#### Scenario: List deployments
- **WHEN** a user GETs `/api/v1/apps/:id/deployments`
- **THEN** all deployments for the app are returned ordered by creation time
