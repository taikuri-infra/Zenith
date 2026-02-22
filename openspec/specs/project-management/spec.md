# Capability: Project Management

## Purpose
Manage projects as the top-level resource scope. Projects have plans (free/pro/enterprise) with resource ceilings, and all apps/databases/storage are scoped to a project.

## Requirements

### Requirement: Project CRUD
The system SHALL support creating, listing, getting, updating, and deleting projects. Each project has a name, owner, plan (free/pro/enterprise), and region. Projects are scoped to the authenticated user.

#### Scenario: Create project
- **WHEN** a user POSTs to `/api/v1/projects` with name and plan
- **THEN** a project is created and owned by the authenticated user

#### Scenario: List own projects
- **WHEN** a user GETs `/api/v1/projects`
- **THEN** only projects owned by the user's email are returned

#### Scenario: Delete project (owner only)
- **WHEN** a user with owner role DELETEs `/api/v1/projects/:id`
- **THEN** the project and its resources are removed

#### Scenario: Non-owner cannot delete
- **WHEN** a non-owner user attempts to delete a project
- **THEN** the system returns 403 Forbidden

### Requirement: Plan-Based Resource Limits
Each project plan SHALL define resource ceilings. The system tracks usage against plan limits for CPU, RAM, storage, apps, and databases.

#### Scenario: Free plan limits
- **WHEN** a free-plan project reaches its app limit
- **THEN** further app creation is denied with an upgrade prompt
