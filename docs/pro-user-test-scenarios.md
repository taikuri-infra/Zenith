# Pro-Tier User Test Scenarios

> Manual QA checklist for validating the Zenith platform from a real Pro user's perspective.
> Each scenario is self-contained — follow top to bottom, check off as you go.

---

## Prerequisites

- **Account**: Pro-tier user on staging (`https://app.stage.freezenith.com`)
- **CLI**: `zen` CLI installed and authenticated (`zen login`)
- **Tools**: `curl`, `docker`, a browser
- **Test domain** (optional): A domain you control for custom domain testing
- **Private Docker Hub repo** (optional): For private registry testing

---

## Scenario 1: Deploy from Public Docker Hub Image

**Goal**: Deploy a single public image (simplest happy path).

- [ ] Log in to `https://app.stage.freezenith.com`
- [ ] Create a new project (e.g., `test-public-deploy`)
- [ ] Click "Deploy App"
- [ ] Enter image: `nginx:latest`
- [ ] Set app name: `my-nginx`
- [ ] Set port: `80`
- [ ] Click Deploy
- [ ] Wait for status to become `running` / `active`
- [ ] Open `https://my-nginx-<slug>.apps.stage.freezenith.com` — should show nginx welcome page
- [ ] Check app detail page shows correct image, port, status
- [ ] Delete the app
- [ ] Verify app is removed from the list

**Known issue**: Deploy status may report `failed` even when the app is running (deploy pipeline bug).

---

## Scenario 2: Deploy from Docker Compose (Public Images)

**Goal**: Paste a docker-compose.yml and have Zenith auto-create everything.

Use this sample compose:

```yaml
version: "3.8"
services:
  web:
    image: nginx:alpine
    ports:
      - "80:80"
  api:
    image: httpbin/httpbin:latest
    ports:
      - "8080:80"
```

- [ ] Create a new project via "New Project" wizard
- [ ] Paste the compose YAML above
- [ ] Verify wizard detects 2 services: `web` (public) and `api` (public)
- [ ] Verify no managed services detected (no databases in this compose)
- [ ] Proceed through env var step (no changes needed)
- [ ] Click Deploy All
- [ ] Verify both apps are created and deploying
- [ ] Wait for both to reach `running` / `active`
- [ ] Open web app URL — should show nginx welcome page
- [ ] Open api app URL — should show httpbin interface
- [ ] Clean up: delete both apps, then delete the project

---

## Scenario 3: Deploy from Docker Compose (With Managed Database)

**Goal**: Compose with an app + PostgreSQL — Zenith provisions a real managed database.

```yaml
version: "3.8"
services:
  backend:
    image: httpbin/httpbin:latest
    ports:
      - "8080:80"
    environment:
      DATABASE_URL: postgres://postgres:secret@db:5432/myapp
    depends_on:
      - db
  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: myapp
```

- [ ] Create new project via wizard
- [ ] Paste compose YAML
- [ ] Verify wizard detects: 1 app service (`backend`), 1 managed service (`db` → PostgreSQL)
- [ ] Verify `DATABASE_URL` env var is translated to use Zenith's managed PG connection string
- [ ] Deploy All
- [ ] Verify managed PostgreSQL is provisioned (status: `provisioned` / `running` / `ready`)
- [ ] Verify backend app is deployed with correct `DATABASE_URL` pointing to managed PG
- [ ] Open backend URL — should respond
- [ ] Clean up: delete app, delete database, delete project

**Known issue**: Database provisioning may return 500 (FK constraint across dual-CNPG clusters).

---

## Scenario 4: Deploy from Private Docker Hub Registry

**Goal**: Deploy an image from a private Docker Hub repository.

- [ ] Have a private image on Docker Hub (e.g., `yourusername/private-app:latest`)
- [ ] Create a new project
- [ ] Click Deploy App
- [ ] Enter the private image URL
- [ ] Toggle "Private registry" ON
- [ ] Enter Docker Hub username and password/token
- [ ] Set port and app name
- [ ] Deploy
- [ ] Verify deployment succeeds (imagePullSecret created as `regcred-<app-subdomain>`)
- [ ] Verify app is accessible at its URL
- [ ] Clean up

---

## Scenario 5: Deploy Unknown/Third-Party Service (n8n, Ghost, etc.)

**Goal**: Deploy a service Zenith doesn't have special support for — treated as generic container.

```yaml
version: "3.8"
services:
  n8n:
    image: n8nio/n8n:latest
    ports:
      - "5678:5678"
    environment:
      N8N_PORT: 5678
      GENERIC_TIMEZONE: Europe/Helsinki
```

- [ ] Create new project via compose wizard
- [ ] Paste the compose YAML
- [ ] Verify wizard detects `n8n` as an app service (not managed)
- [ ] Deploy
- [ ] Wait for app to start
- [ ] Open the n8n URL — should show n8n setup page
- [ ] Verify environment variables are passed correctly
- [ ] Clean up

---

## Scenario 6: Custom Domain

**Goal**: Assign a custom domain to a deployed app and verify TLS.

Prerequisites: You control a domain and can set DNS records.

- [ ] Deploy any app (e.g., nginx from Scenario 1)
- [ ] Go to app detail page → Custom Domains section
- [ ] Click "Add Domain"
- [ ] Enter your domain (e.g., `myapp.example.com`)
- [ ] Set DNS: CNAME `myapp.example.com` → `<app-subdomain>.apps.stage.freezenith.com`
- [ ] Wait for DNS propagation (1-5 minutes)
- [ ] Verify domain status changes: `pending` → `active`
- [ ] Open `https://myapp.example.com` — should show the app with valid TLS
- [ ] Verify certificate is issued by Let's Encrypt
- [ ] Remove the custom domain
- [ ] Verify app is still accessible at the default `.apps.stage.freezenith.com` URL
- [ ] Clean up

**Note**: No DNS verification check exists yet — domain goes active immediately. User must configure DNS themselves.

---

## Scenario 7: S3 Storage Bucket (CRUD)

**Goal**: Create a storage bucket, upload/download files, delete.

### 7a: Standalone Bucket
- [ ] Go to Storage section (or use API)
- [ ] Create a new storage bucket (e.g., `test-uploads`)
- [ ] Verify bucket status becomes `active`
- [ ] Upload a test file (e.g., `hello.txt` with content "Hello Zenith")
- [ ] List objects — verify `hello.txt` appears
- [ ] Download the file — verify content matches
- [ ] Create a folder (e.g., `images/`)
- [ ] Upload a file into the folder
- [ ] List objects with prefix `images/` — verify file appears
- [ ] Delete the file
- [ ] Delete the folder
- [ ] Delete the bucket
- [ ] Verify bucket is removed

### 7b: Per-App Bucket
- [ ] Deploy an app
- [ ] Create a storage bucket attached to the app
- [ ] Verify `S3_ENDPOINT` and `S3_BUCKET` env vars are injected into the app
- [ ] Upload a file via presigned URL
- [ ] Download via presigned URL
- [ ] Delete bucket
- [ ] Verify env vars are removed from the app
- [ ] Clean up

---

## Scenario 8: Database Backup & Restore

**Goal**: Create a database backup, verify it exists, restore from it, download it.

Prerequisites: Have a provisioned PostgreSQL database.

- [ ] Create a project + deploy an app with a managed PostgreSQL database
- [ ] Wait for database to be `ready` / `provisioned`
- [ ] Trigger a backup: `POST /api/v1/apps/:appId/databases/:dbId/backups`
- [ ] Verify backup status becomes `completed`
- [ ] List backups — verify the backup appears with size > 0
- [ ] Download the backup file (verify file is a valid .sql.gz)
- [ ] Restore from backup: `POST /api/v1/.../backups/:backupId/restore`
- [ ] Verify database status goes `provisioning` → `ready`
- [ ] Verify data is intact after restore
- [ ] Delete the backup
- [ ] Clean up

**Implementation**: Backup triggers a K8s Job (pg_dump → gzip → S3). Restore triggers a K8s Job (S3 → gunzip → psql). Download returns a presigned S3 URL (24h expiry). Requires `cnpg-s3-credentials` secret in `zenith-apps` namespace.

---

## Scenario 9: Redeploy & Rollback

**Goal**: Update an app's image, then roll back to the previous version.

- [ ] Deploy an app with `nginx:1.25`
- [ ] Wait for `running` / `active` status
- [ ] Open the app URL — verify it works
- [ ] Redeploy with `nginx:1.26-alpine` via CI deploy endpoint or UI
- [ ] Verify a second deployment is created
- [ ] Wait for new deployment to be active
- [ ] Verify the app is now running `nginx:1.26-alpine`
- [ ] Trigger rollback to the first deployment
- [ ] Verify rollback deployment is created
- [ ] Wait for rollback to complete
- [ ] Verify app is back on `nginx:1.25`
- [ ] Clean up

**Known issue**: Deploy status may report `failed`. Rollback flow needs verification.

---

## Scenario 10: App Logs

**Goal**: View live logs and deployment history logs.

- [ ] Deploy any app (e.g., nginx)
- [ ] Generate some traffic (curl the app URL a few times)
- [ ] View live logs: `GET /api/v1/apps/:appId/logs`
- [ ] Verify log output contains nginx access log entries
- [ ] View deployment logs: `GET /api/v1/apps/:appId/deployments/:did/logs/history`
- [ ] Verify deployment log shows build/deploy output
- [ ] Clean up

**Known issue**: Logs endpoint returns 500 on staging.

---

## Scenario 11: Environment Variables

**Goal**: Set, update, and delete environment variables for an app.

- [ ] Deploy any app
- [ ] Set env vars: `POST /api/v1/apps/:appId/env` with `{"vars": {"FOO": "bar", "SECRET": "mysecret"}}`
- [ ] Verify env vars are stored (GET should return them)
- [ ] Verify sensitive values are encrypted at rest
- [ ] Update an env var (change `FOO` to `baz`)
- [ ] Delete an env var
- [ ] Verify the app reloads with updated env vars
- [ ] Import from `.env` file format
- [ ] Verify max 100 env vars limit is enforced
- [ ] Clean up

---

## Scenario 12: Load Test (Pro Tier Capacity)

**Goal**: Determine how many concurrent users a Pro-tier (€29/mo) app can handle.

Prerequisites: k6 installed (`brew install k6`).

- [ ] Deploy a real app (not just nginx — something that uses CPU)
- [ ] Run auth load test: `k6 run infra/load-tests/auth-load.js`
- [ ] Record results: p95 latency, error rate, requests/sec
- [ ] Run CRUD load test: `k6 run infra/load-tests/crud-load.js`
- [ ] Record results
- [ ] Run spike test: `k6 run infra/load-tests/spike-test.js`
- [ ] Record: at what VU count does the API start returning 429/500?
- [ ] Document Pro tier capacity: "Pro handles ~X concurrent users with p95 < Yms"

---

## Scenario 13: Team Collaboration

**Goal**: Invite a team member and verify RBAC.

- [ ] Go to Team section
- [ ] Invite a team member by email (valid email format)
- [ ] Verify invitation is sent
- [ ] (If possible) Accept invitation from another account
- [ ] Verify team member can see the project
- [ ] Verify team member CANNOT perform admin actions (based on role)
- [ ] Remove team member
- [ ] Verify access is revoked

---

## Scenario 14: API Keys & Webhooks

**Goal**: Create API keys and configure webhooks.

- [ ] Create an API key
- [ ] Use the API key to make an authenticated request
- [ ] Verify API key has correct scopes
- [ ] Rotate the API key
- [ ] Verify old key no longer works
- [ ] Create a webhook for deploy events
- [ ] Trigger a deploy
- [ ] Verify webhook is called with correct payload
- [ ] Delete webhook and API key

---

## Scenario 15: Deploy Tokens (CI/CD)

**Goal**: Create deploy tokens for automated CI/CD pipelines.

- [ ] Create a deploy token: `POST /api/v1/deploy-tokens`
- [ ] Verify token format: `znt_id_...` / `znt_sk_...`
- [ ] Use the token to trigger a deploy via API
- [ ] Verify deploy is created successfully
- [ ] Rotate the token (24h grace period for old token)
- [ ] Verify old token still works within grace period
- [ ] After grace period, verify old token is rejected
- [ ] Revoke the token
- [ ] Verify revoked token no longer works

---

## Bug Tracker

Track issues found during manual testing:

| # | Scenario | Issue | Severity | Status |
|---|----------|-------|----------|--------|
| 1 | 1, 2, 3, 9 | Deploy status reports `failed` even when app works | HIGH | Open |
| 2 | 10 | Logs endpoint returns 500 | HIGH | Open |
| 3 | 3, 8 | Database provisioning FK constraint 500 | HIGH | Fixed — passes project_id |
| 4 | 8 | DB backup is record-only, no actual pg_dump | HIGH | Fixed — K8s Job pg_dump → S3 |
| 5 | 8 | DB restore is a stub (no pg_restore) | HIGH | Fixed — K8s Job S3 → pg_restore |
| 6 | 8 | DB backup download endpoint missing | MEDIUM | Fixed — presigned S3 URL |
| 7 | 6 | No DNS verification for custom domains | LOW | Open |

---

## How to Update This Document

As you test, mark checkboxes `[x]` and add issues to the Bug Tracker table. After fixing an issue, update its status to `Fixed` and note the commit hash.
