# Capability: Access Control

## Purpose
Fine-grained access control including API keys, custom RBAC roles, SCIM 2.0 provisioning, and IP whitelisting for enterprise security.

## Requirements

### Requirement: API Key Management
The system SHALL support creating, listing, and revoking API keys. Each key has a name, scoped permissions, and expiry. Keys are hashed before storage.

#### Scenario: Create API key
- **WHEN** a user creates an API key with name and scopes
- **THEN** the system returns the key value (shown once) and stores the hash

#### Scenario: Revoke API key
- **WHEN** a user revokes an API key by ID
- **THEN** the key is invalidated and can no longer authenticate requests

### Requirement: Custom Roles (RBAC)
The system SHALL support creating custom roles with fine-grained permissions. Roles can be assigned to users. Built-in roles (owner, admin, developer, viewer) are always available.

#### Scenario: Create custom role
- **WHEN** an admin creates a role with specific permissions
- **THEN** the role is available for assignment to users

#### Scenario: Assign role to user
- **WHEN** an admin assigns a custom role to a user
- **THEN** the user's access is governed by that role's permissions

### Requirement: SCIM 2.0 Provisioning
The system SHALL support SCIM 2.0 endpoints for automated user provisioning and deprovisioning from external identity providers.

#### Scenario: SCIM create user
- **WHEN** an IdP POSTs a SCIM user resource
- **THEN** the user is created in Zenith with mapped attributes

#### Scenario: SCIM deactivate user
- **WHEN** an IdP PATCHes a user to inactive
- **THEN** the user's access is revoked

### Requirement: IP Whitelisting
The system SHALL support IP whitelist rules per organization. When enabled, only requests from whitelisted IPs/CIDRs are allowed.

#### Scenario: Add IP whitelist rule
- **WHEN** an admin adds a CIDR range to the whitelist
- **THEN** only requests from that range are permitted

#### Scenario: Request from non-whitelisted IP
- **WHEN** IP whitelisting is enabled and a request comes from a non-whitelisted IP
- **THEN** the system returns 403 Forbidden
