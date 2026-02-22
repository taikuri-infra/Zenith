# Capability: Deploy Engine

## Purpose
Enable deploying applications from Git repositories through an automated pipeline: clone, framework detection, Dockerfile generation, Kaniko build, and Kubernetes deployment. Includes webhook triggers, build log streaming, releases, and rollback.

## Requirements

### Requirement: App Creation from Git
The system SHALL allow users to create apps by providing a Git repository URL, branch, and optional framework hint. The system generates a unique subdomain under `BASE_DOMAIN`.

#### Scenario: Create app from repo
- **WHEN** a user POSTs a repo URL and branch to `/api/v1/apps`
- **THEN** the system creates an app record with status `pending` and a generated subdomain

### Requirement: Framework Detection
The system SHALL auto-detect the application framework from repository file markers. Supported frameworks: Next.js, Go, Python, Django, Flask, Rails, Express, Static, Dockerfile.

#### Scenario: Next.js detected
- **WHEN** a cloned repo contains `next.config.js` or `next.config.ts`
- **THEN** the system detects framework as `nextjs`

#### Scenario: Dockerfile present
- **WHEN** a cloned repo contains a `Dockerfile`
- **THEN** the system uses the existing Dockerfile instead of generating one

### Requirement: Dockerfile Generation
The system SHALL generate multi-stage Dockerfiles with non-root users for all supported frameworks when no Dockerfile exists in the repository.

#### Scenario: Generate Go Dockerfile
- **WHEN** framework is detected as `go`
- **THEN** a multi-stage Dockerfile is generated (build with `golang:alpine`, run with `alpine`, non-root user)

### Requirement: Build Pipeline
The system SHALL execute an async build pipeline: clone repo -> detect framework -> generate Dockerfile -> prepare image tag. Each step emits log entries to the LogHub.

#### Scenario: Successful build
- **WHEN** a build is triggered for an app
- **THEN** the pipeline clones, detects, generates Dockerfile, and tags the image
- **AND** each step's progress is published to LogHub

#### Scenario: Build failure
- **WHEN** any pipeline step fails
- **THEN** the deployment status is set to `failed` and error details are logged

### Requirement: Kaniko Build Execution
The system SHALL submit Kaniko Jobs to Kubernetes for in-cluster container image builds. Jobs include caching and resource limits. The system polls for completion and streams pod logs to LogHub.

#### Scenario: Kaniko job succeeds
- **WHEN** a Kaniko build job completes successfully
- **THEN** the built image is available in the configured registry
- **AND** pod logs are streamed to LogHub during execution

#### Scenario: Kaniko job fails
- **WHEN** a Kaniko build job fails or times out
- **THEN** the deployment is marked `failed` and the job is cleaned up

### Requirement: K8s Resource Deployment
After a successful build, the system SHALL create Kubernetes Deployment, Service, and Traefik IngressRoute (with TLS) resources for the app.

#### Scenario: Deploy to K8s
- **WHEN** a build completes successfully
- **THEN** the pipeline calls `DeployApp()` to create/update K8s Deployment + Service + IngressRoute

### Requirement: GitHub Webhook Integration
The system SHALL accept GitHub push webhooks at `/api/v1/webhooks/github`, verify HMAC-SHA256 signatures, match the push to registered apps by repo URL + branch, and trigger builds.

#### Scenario: Valid webhook triggers build
- **WHEN** a valid GitHub push webhook is received for a registered app's repo and branch
- **THEN** a new deployment is created and the build pipeline is triggered

#### Scenario: Invalid signature rejected
- **WHEN** a webhook with an invalid HMAC-SHA256 signature is received
- **THEN** the system returns 401 and does not trigger a build

### Requirement: GitLab and Bitbucket Webhooks
The system SHALL also accept push webhooks from GitLab and Bitbucket with their respective signature verification.

#### Scenario: GitLab push triggers build
- **WHEN** a valid GitLab push webhook is received
- **THEN** the system matches the app and triggers a build

### Requirement: Build Log Streaming
The system SHALL stream build/deploy logs via SSE at `/api/v1/apps/:id/deployments/:did/logs`. A keepalive is sent every 30s and `event: done` signals completion. Historical logs are available via `/logs/history`.

#### Scenario: Subscribe to logs
- **WHEN** a client connects to the SSE logs endpoint during an active build
- **THEN** the client receives real-time log entries as SSE events

#### Scenario: Get log history
- **WHEN** a client requests `/logs/history` for a completed deployment
- **THEN** the system returns stored log entries as a JSON array

### Requirement: Deployment Rollback
The system SHALL support rolling back to a previous deployment via `/api/v1/apps/:id/rollback`.

#### Scenario: Rollback to previous
- **WHEN** a user triggers rollback with a deployment ID
- **THEN** the system redeploys the image from the specified deployment

### Requirement: Releases
The system SHALL track image releases pushed by CI/CD. Users can list releases and trigger one-click deploys from a specific release, bypassing the build phase.

#### Scenario: Register release
- **WHEN** CI/CD POSTs to `/api/v1/apps/:appId/releases` with image URL, SHA, branch
- **THEN** the release is recorded in the database

#### Scenario: Deploy from release
- **WHEN** a user POSTs to `/api/v1/apps/:appId/releases/:rid/deploy`
- **THEN** the system deploys the pre-built image directly (no build step)

### Requirement: GitHub Action
A composite GitHub Action (`zenith-deploy`) SHALL be provided for customers to build, push to their own registry, and register the release with Zenith from their CI/CD pipelines.

#### Scenario: Customer uses zenith-deploy action
- **WHEN** a customer adds `zenith-deploy` to their GitHub workflow
- **THEN** the action builds the image, pushes to the customer's registry, and POSTs the release to Zenith API
