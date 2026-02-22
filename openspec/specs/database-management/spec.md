# Capability: Database Management

## Purpose
Provision and manage databases (PostgreSQL, MySQL, MongoDB, Redis) scoped to projects, with connection details and backup support.

## Requirements

### Requirement: Database Provisioning
The system SHALL support creating databases scoped to a project. Supported engines: PostgreSQL, MySQL, MongoDB, Redis. Each database has a name, engine, version, and storage allocation.

#### Scenario: Create PostgreSQL database
- **WHEN** a user POSTs to `/api/v1/projects/:id/databases` with engine `postgresql`
- **THEN** a database resource is created with a connection string

#### Scenario: Create Redis database
- **WHEN** a user POSTs with engine `redis`
- **THEN** a Redis instance resource is created

### Requirement: Database CRUD
The system SHALL support listing, getting, and deleting databases per project.

#### Scenario: List databases
- **WHEN** a user GETs `/api/v1/projects/:id/databases`
- **THEN** all databases for the project are returned

#### Scenario: Delete database
- **WHEN** a user DELETEs `/api/v1/projects/:id/databases/:name`
- **THEN** the database resource is removed

### Requirement: Database Backups (Stub)
The system SHALL expose backup endpoints for listing and creating backups, currently as stubs for future implementation.

#### Scenario: List backups
- **WHEN** a user GETs `/api/v1/projects/:id/databases/:name/backups`
- **THEN** the system returns a (currently empty) list of backups

#### Scenario: Create backup
- **WHEN** a user POSTs to `/api/v1/projects/:id/databases/:name/backups`
- **THEN** the system acknowledges the request (stub behavior)

### Requirement: Database UI
The Web Platform SHALL display databases with engine-specific icons (PostgreSQL=blue P, MySQL=orange M, MongoDB=green M, Redis=red R), connection string copy, and stats (engine, storage, port, created date).

#### Scenario: View database detail
- **WHEN** a user navigates to `/databases/:name`
- **THEN** the page shows connection string with copy button, engine icon, storage, port, and creation date
