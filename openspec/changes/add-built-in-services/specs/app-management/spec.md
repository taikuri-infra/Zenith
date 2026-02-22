## ADDED Requirements

### Requirement: Per-App PostgreSQL Database
The system SHALL allow provisioning a dedicated PostgreSQL database per app via CloudNativePG. The connection string SHALL be auto-injected as `DATABASE_URL` in the app's environment.

#### Scenario: Provision per-app database
- **WHEN** a user enables a database for their app
- **THEN** a CloudNativePG Cluster is created and `DATABASE_URL` is injected into the app deployment

#### Scenario: Plan limit enforced
- **WHEN** a free-plan user tries to create a second database
- **THEN** the system returns 403 with an upgrade prompt

### Requirement: Per-App Auth Realm
The system SHALL allow enabling built-in authentication per app via Keycloak. A realm and client are created, and `AUTH_URL` + `AUTH_CLIENT_ID` are auto-injected.

#### Scenario: Enable app auth
- **WHEN** a user enables auth for their app
- **THEN** a Keycloak realm is created with a client, and auth env vars are injected

### Requirement: Per-App S3 Storage
The system SHALL allow provisioning an S3-compatible storage bucket per app via Hetzner Object Storage. Credentials are auto-injected as environment variables.

#### Scenario: Create app storage
- **WHEN** a user creates storage for their app
- **THEN** an S3 bucket is created and `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` are injected
