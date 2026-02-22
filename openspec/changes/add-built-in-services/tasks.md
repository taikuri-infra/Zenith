## 1. Per-App PostgreSQL

- [ ] 1.1 Install CloudNativePG operator on management cluster
- [ ] 1.2 Create `PerAppDatabase` CRD (app_id, engine, size_limit, status)
- [ ] 1.3 API endpoint: `POST /api/v1/apps/:id/database` — provisions CloudNativePG Cluster
- [ ] 1.4 Auto-inject `DATABASE_URL` env var into app deployment
- [ ] 1.5 Plan-based size limits enforcement
- [ ] 1.6 Connection pooling via PgBouncer sidecar

## 2. Per-App Auth (Keycloak)

- [ ] 2.1 Deploy Keycloak with per-tenant realm configuration
- [ ] 2.2 API endpoint: `POST /api/v1/apps/:id/auth/enable` — creates realm + client
- [ ] 2.3 Auto-inject `AUTH_URL`, `AUTH_CLIENT_ID` env vars
- [ ] 2.4 SDK snippet generation for frontend frameworks (React, Next.js)
- [ ] 2.5 Plan-based user limits (free: 1K, pro: 10K, team: 100K)

## 3. Per-App S3 Storage

- [ ] 3.1 Hetzner Object Storage API client
- [ ] 3.2 API endpoint: `POST /api/v1/apps/:id/storage` — creates bucket
- [ ] 3.3 Auto-inject `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` env vars
- [ ] 3.4 Plan-based storage limits

## 4. Frontend UI

- [ ] 4.1 App detail page: Database tab (connection string, usage, status)
- [ ] 4.2 App detail page: Auth tab (realm config, user count, SDK snippet)
- [ ] 4.3 App detail page: Storage tab (bucket info, objects, usage)
- [ ] 4.4 Demo mode mocks for all new tabs

## 5. Testing

- [ ] 5.1 Unit tests for provisioning logic
- [ ] 5.2 Integration tests with mock operators
- [ ] 5.3 Plan limit enforcement tests
