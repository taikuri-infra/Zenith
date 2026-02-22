# Capability: Enterprise Features

## Purpose
Enterprise-tier features including compliance dashboard, audit export, white-label branding, custom domains, DPA management, preview deployments, and user-defined webhooks.

## Requirements

### Requirement: Compliance Dashboard
The system SHALL provide a compliance overview showing SOC2, GDPR, and HIPAA readiness status with checklist items and remediation guidance.

#### Scenario: View compliance status
- **WHEN** an admin accesses the compliance dashboard
- **THEN** compliance frameworks are shown with pass/fail status per requirement

### Requirement: Audit Log Export
The system SHALL support exporting audit logs in CSV and JSON formats with date range filtering.

#### Scenario: Export as CSV
- **WHEN** an admin requests audit export with format `csv` and date range
- **THEN** a CSV file with filtered audit entries is returned

#### Scenario: Export as JSON
- **WHEN** an admin requests audit export with format `json`
- **THEN** a JSON file with filtered audit entries is returned

### Requirement: White-Label Branding
The system SHALL support custom branding: logo, colors, favicon, and platform name per organization.

#### Scenario: Set custom branding
- **WHEN** an admin uploads a logo and sets brand colors
- **THEN** the platform UI reflects the custom branding

### Requirement: Custom Domain for Platform
The system SHALL support serving the platform under a customer's own domain with automatic TLS provisioning.

#### Scenario: Configure custom domain
- **WHEN** an admin sets a custom domain for their organization
- **THEN** the system configures DNS verification and provisions a TLS certificate

### Requirement: Hide Zenith Badge
The system SHALL support hiding the "Powered by Zenith" badge for Enterprise plan customers.

#### Scenario: Hide badge
- **WHEN** an Enterprise admin enables the hide-badge setting
- **THEN** the platform UI no longer shows the Zenith badge

### Requirement: Data Processing Agreement (DPA)
The system SHALL allow organizations to upload and manage their Data Processing Agreement documents.

#### Scenario: Upload DPA
- **WHEN** an admin uploads a DPA document
- **THEN** the document is stored and associated with the organization

### Requirement: Preview Deployments
The system SHALL support creating preview/staging deployments from pull requests or branches, with unique URLs and automatic cleanup.

#### Scenario: Create preview deployment
- **WHEN** a user creates a preview from a branch
- **THEN** the system deploys the branch to a unique preview URL

#### Scenario: Auto-cleanup preview
- **WHEN** a preview deployment's source branch is merged or deleted
- **THEN** the preview is automatically cleaned up

### Requirement: User-Defined Webhooks
The system SHALL support user-configured webhooks that fire on platform events (deployment success/failure, app created/deleted, etc.). Webhooks use HMAC-SHA256 signing.

#### Scenario: Create webhook
- **WHEN** a user creates a webhook with URL and event types
- **THEN** the system stores the webhook and generates an HMAC secret

#### Scenario: Webhook fires on deployment
- **WHEN** a deployment succeeds for an app with a configured webhook
- **THEN** the system POSTs a signed payload to the webhook URL
