# Capability: Demo Mode

## Purpose
Allow anyone to experience the Zenith platform without a backend by switching to mock data at build time, with separate Docker images for demo vs production.

## Requirements

### Requirement: Build-Time Demo Switching
The system SHALL support a demo mode activated by the build-time environment variable `NEXT_PUBLIC_DEMO_MODE=true`. When enabled, `getApi()` returns a mock API client instead of the real HTTP client.

#### Scenario: Demo mode active
- **WHEN** the app is built with `NEXT_PUBLIC_DEMO_MODE=true`
- **THEN** `getApi()` returns the demo API client with mock data

#### Scenario: Production mode
- **WHEN** the app is built without the demo flag
- **THEN** `getApi()` returns the real API client pointing to the backend

### Requirement: Mock Data
The demo API SHALL return hardcoded mock data after a 300ms simulated delay. The mock dataset includes apps, databases, storage, clusters, tenants, modules, audit entries, and infrastructure stats.

#### Scenario: Demo API returns mock data
- **WHEN** any API method is called in demo mode
- **THEN** mock data is returned after 300ms delay

#### Scenario: Mutations blocked in demo
- **WHEN** a mutation (create, update, delete) is attempted in demo mode
- **THEN** the system throws "Not available in demo mode"

### Requirement: Demo UI Elements
In demo mode, the system SHALL show a `DemoBanner` ("Demo Mode - Viewing with sample data") and `DemoButton` components that intercept clicks with "Available in your own installation" tooltip.

#### Scenario: Demo banner visible
- **WHEN** the app is in demo mode
- **THEN** an emerald banner is displayed at the top of every page

#### Scenario: Demo button intercept
- **WHEN** a user clicks an action button in demo mode
- **THEN** a tooltip shows "Available in your own installation" instead of executing

### Requirement: Separate Docker Images
The system SHALL produce separate Docker images for demo and production: `zenith-mc-demo:latest` / `zenith-web-demo:latest` (with baked-in mock data) vs `zenith-mc:latest` / `zenith-web:latest` (connecting to real API).

#### Scenario: Demo image build
- **WHEN** a Docker build includes `--build-arg NEXT_PUBLIC_DEMO_MODE=true`
- **THEN** the resulting image serves the demo version with mock data

### Requirement: Auth Bypass in Demo
In demo mode, the login page SHALL auto-redirect to `/` and `useAuth()` SHALL return a fake admin user (`admin@zenith.dev`), skipping all auth checks.

#### Scenario: Demo login bypass
- **WHEN** a user navigates to `/login` in demo mode
- **THEN** the user is auto-redirected to `/` as a fake admin
