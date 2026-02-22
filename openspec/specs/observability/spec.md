# Capability: Observability

## Purpose
Platform observability through OpenTelemetry traces/metrics export, Backstage catalog integration, and health/readiness probes.

## Requirements

### Requirement: OpenTelemetry Integration
The system SHALL support OpenTelemetry traces and metrics export via OTLP gRPC. Activation is opt-in via `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable. Health and readiness probes (`/health`, `/ready`) are excluded from tracing.

#### Scenario: OTel enabled
- **WHEN** `OTEL_EXPORTER_OTLP_ENDPOINT` is set
- **THEN** the API server exports traces and metrics to the configured OTLP endpoint

#### Scenario: OTel disabled
- **WHEN** `OTEL_EXPORTER_OTLP_ENDPOINT` is not set
- **THEN** no telemetry is exported and there is no performance impact

### Requirement: Backstage Catalog
The system SHALL expose Backstage catalog entities at `/api/v1/backstage/catalog`, converting Zenith CRDs to Backstage-compatible entity format. The endpoint is JWT-protected.

#### Scenario: Get catalog entities
- **WHEN** an authenticated user GETs `/api/v1/backstage/catalog`
- **THEN** Zenith resources are returned as Backstage catalog entities

### Requirement: Health and Readiness Probes
The system SHALL expose `/health` (liveness) and `/ready` (readiness) endpoints returning server status, version, and uptime.

#### Scenario: Health check
- **WHEN** a client GETs `/health`
- **THEN** the system returns `{"status": "ok", "version": "...", "uptime": "..."}`

#### Scenario: Readiness check
- **WHEN** a client GETs `/ready`
- **THEN** the system returns readiness status indicating the server can accept traffic
