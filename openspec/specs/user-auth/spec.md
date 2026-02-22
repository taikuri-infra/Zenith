# Capability: User Authentication

## Purpose
Handle user authentication and identity management for the Zenith platform — JWT login, registration, token refresh, MFA, SSO, session management, and API key auth.

## Requirements

### Requirement: JWT-Based Login
The system SHALL authenticate users via email and password, returning a JWT access token (1h TTL) and refresh token (7d TTL) signed with HS256.

#### Scenario: Successful login
- **WHEN** a user POSTs valid email and password to `/api/v1/auth/login`
- **THEN** the system returns a JWT pair (access_token, refresh_token) with user claims (email, name, role)

#### Scenario: Invalid credentials
- **WHEN** a user POSTs invalid email or password
- **THEN** the system returns 401 Unauthorized

### Requirement: User Registration
The system SHALL allow user registration. The first registered user SHALL be assigned the `owner` role; subsequent users SHALL be assigned `developer`.

#### Scenario: First user registers as owner
- **WHEN** no users exist and a user POSTs to `/api/v1/auth/register`
- **THEN** the user is created with role `owner`

#### Scenario: Subsequent user registers as developer
- **WHEN** users already exist and a new user registers
- **THEN** the new user is created with role `developer`

### Requirement: Token Refresh
The system SHALL support token rotation via `/api/v1/auth/refresh`, issuing a new access token from a valid refresh token.

#### Scenario: Valid refresh
- **WHEN** a valid refresh token is POSTed to `/api/v1/auth/refresh`
- **THEN** a new access token is returned

#### Scenario: Expired refresh token
- **WHEN** an expired or invalid refresh token is provided
- **THEN** the system returns 401 Unauthorized

### Requirement: Role Hierarchy
The system SHALL enforce a role hierarchy: Owner(4) > Admin(3) > Developer(2) > Viewer(1). Protected endpoints SHALL require minimum role levels.

#### Scenario: Insufficient role
- **WHEN** a Developer-role user accesses an Admin-only endpoint
- **THEN** the system returns 403 Forbidden

### Requirement: API Key Authentication
The system SHALL support authentication via `X-API-Key` header as an alternative to JWT. API key auth SHALL default to `RoleDeveloper`.

#### Scenario: Valid API key
- **WHEN** a request includes a valid `X-API-Key` header
- **THEN** the system authenticates the user with Developer role

### Requirement: MFA/TOTP
The system SHALL support optional TOTP-based two-factor authentication. Users can enable/disable MFA and verify TOTP codes during login.

#### Scenario: Enable MFA
- **WHEN** a user enables MFA via the API
- **THEN** the system generates a TOTP secret and returns a provisioning URI

#### Scenario: Login with MFA enabled
- **WHEN** a user with MFA enabled provides valid credentials
- **THEN** the system requires a valid TOTP code before issuing tokens

### Requirement: SSO (SAML + OIDC)
The system SHALL support Single Sign-On via SAML 2.0 and OIDC providers. Organizations can configure SSO connections.

#### Scenario: SAML login redirect
- **WHEN** a user initiates SAML login
- **THEN** the system redirects to the configured IdP with a SAML AuthnRequest

#### Scenario: OIDC callback
- **WHEN** the OIDC provider calls back with an authorization code
- **THEN** the system exchanges the code for tokens and creates/updates the user

### Requirement: Session Management
The system SHALL track active user sessions with device info, IP, and last-active timestamps. Users can list and revoke sessions.

#### Scenario: List sessions
- **WHEN** a user requests their active sessions
- **THEN** the system returns all sessions with device, IP, and timestamps

#### Scenario: Revoke session
- **WHEN** a user revokes a specific session
- **THEN** that session's tokens are invalidated immediately
