# Capability: Real-Time Events

## Purpose
Broadcast platform events (deployment status changes, build progress) in real-time via SSE to connected frontend clients for live UI updates.

## Requirements

### Requirement: EventHub SSE Broadcasting
The system SHALL provide a server-sent events (SSE) endpoint at `/api/v1/events` that broadcasts platform events to connected clients. The EventHub is an in-memory pub/sub broadcaster.

#### Scenario: Subscribe to events
- **WHEN** a client connects to the SSE events endpoint
- **THEN** the client receives real-time events as they occur

#### Scenario: Deployment event emitted
- **WHEN** a deployment status changes (building, deploying, succeeded, failed)
- **THEN** the EventHub publishes the event to all subscribers

### Requirement: Event Types
The pipeline SHALL emit 6 event types: `deployment_started`, `build_progress`, `build_completed`, `deploy_started`, `deploy_completed`, `deployment_failed`.

#### Scenario: Build progress event
- **WHEN** a build pipeline step completes
- **THEN** a `build_progress` event is emitted with step details

### Requirement: Frontend Auto-Refresh
The Web Platform SHALL use the `useDeployEvents` hook to automatically refresh Deploy page cards and App Detail deployment rows when events are received.

#### Scenario: Deploy page auto-updates
- **WHEN** a deployment status event is received via SSE
- **THEN** the Deploy page card grid updates without manual refresh
