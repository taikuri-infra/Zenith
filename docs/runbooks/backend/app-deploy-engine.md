# Runbook — App Deploy Engine

## 1. Purpose

Backend service that handles automated app deployment from Git repos to Kubernetes, including framework detection, container building, and K8s resource management.

## 2. How to Run

```bash
# Development (in-memory storage)
cd services/api && go run ./cmd/server/

# With PostgreSQL
DATABASE_URL="postgres://zenith:pass@localhost:5432/zenith?sslmode=disable" go run ./cmd/server/

# With webhook support
GITHUB_WEBHOOK_SECRET="your-secret" BASE_DOMAIN="freezenith.com" go run ./cmd/server/
```

## 3. How to Deploy

Deploy engine is part of the main API server (`zenith-api`):

```bash
ssh ghasi "cd /opt/zenith && bash infra/scripts/deploy.sh"
```

Required env vars in `zenith-secrets` K8s Secret:
- `DATABASE_URL`
- `GITHUB_WEBHOOK_SECRET`
- `BASE_DOMAIN`

## 4. Health Checks

- `GET /health` — Returns 200 with uptime
- `GET /ready` — Returns 200 when accepting traffic

## 5. Monitoring

- API request logs: `time | status | latency | method | path`
- Build pipeline: logs to stdout (`[builder]`, `[deployer]` prefixes)
- Deployment status tracked in `deployments` table

## 6. Debugging

### Common Issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| Webhook returns 401 | Bad `GITHUB_WEBHOOK_SECRET` | Verify secret matches GitHub config |
| App stuck in "building" | Pipeline goroutine crashed | Check API logs, restart server |
| Framework "unknown" | New framework not supported | Add detection in `deploy/detect.go` |
| Deploy fails | K8s client error | Check cluster connectivity |

### Useful Queries

```sql
-- Recent failed deployments
SELECT d.id, a.name, d.status, d.created_at
FROM deployments d JOIN apps a ON d.app_id = a.id
WHERE d.status = 'failed' ORDER BY d.created_at DESC LIMIT 10;

-- Apps with no successful deployment
SELECT a.name, a.status FROM apps a
WHERE NOT EXISTS (SELECT 1 FROM deployments d WHERE d.app_id = a.id AND d.status = 'active');
```

## 7. Disaster Recovery

- **Database**: PostgreSQL backup via `pg_dump`
- **In-memory mode**: No persistence — restart loses all data
- **K8s resources**: Recreated from database state on redeploy

## 8. Ownership

- **Team**: DoTech Platform
- **Codebase**: `services/api/internal/deploy/`, `services/api/internal/store/`, `services/api/internal/handlers/`

## 9. Change History

| Date | Change |
|------|--------|
| 2026-02-21 | Phase 2A: Database models + AppRepository (in-memory + PostgreSQL) |
| 2026-02-21 | Phase 2B: Framework detection (9 frameworks) + Dockerfile generation |
| 2026-02-21 | Phase 2C: API handlers (apps, webhook, deploy, env vars) |
| 2026-02-21 | Phase 2D: Build pipeline (Builder, Kaniko, Pipeline) |
| 2026-02-21 | Phase 2E: K8s deployment resources (Deployment, Service, IngressRoute) |
| 2026-02-21 | Phase 2F: Dashboard pages (app detail with 3-tab UI) |
| 2026-02-21 | Phase 3: Build log streaming — LogHub (ring buffer + pub/sub), LogHandler (SSE stream + history), routes wired in main.go; fixed fasthttp StreamWriter API (bufio.Writer); 3 new handler tests |

### Log Streaming Endpoints (Phase 3)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/apps/:id/deployments/:did/logs` | SSE stream — real-time log entries; keepalive every 30s; `event: done` when deployment finishes |
| `GET` | `/api/v1/apps/:id/deployments/:did/logs/history` | JSON snapshot — `{"items":[...],"total":N}` |

### LogHub Architecture

```
Pipeline.emitLog() → LogHub.Publish()
                            ├─ ring-buffer history (max 500 per deployment)
                            └─ fan-out to all active LogSubscribers (non-blocking, drops slow consumers)

LogHandler.StreamLogs()  → LogHub.Subscribe() → EventSource (SSE)
LogHandler.GetLogs()     → LogHub.History()    → JSON response
```
