# Change: Add Built-in Services (Phase 3)

## Why
To compete with Supabase/Railway, each app needs one-click access to a PostgreSQL database, auth SDK, and S3 storage — provisioned automatically and connected via environment variables. This is the "batteries included" experience.

## What Changes
- Per-app PostgreSQL via CloudNativePG operator (one `Cluster` CRD per app)
- Per-app auth realm via Keycloak (one realm per tenant)
- Per-app S3 bucket via Hetzner Object Storage API
- Auto-inject connection strings as environment variables into app deployments
- Dashboard UI tabs for managing per-app database, auth, and storage
- Plan-based limits (free: 1 DB 500MB, pro: 3 DB 5GB, team: 10 DB 20GB)

## Impact
- Affected specs: app-management, database-management, storage-management, web-platform
- Affected code: `services/api/internal/deploy/`, `services/operator/`, `apps/web/`
- New dependencies: CloudNativePG operator, Keycloak, Hetzner S3 API
- **BREAKING**: App deployments will auto-inject new env vars (DATABASE_URL, AUTH_URL, S3_ENDPOINT)
