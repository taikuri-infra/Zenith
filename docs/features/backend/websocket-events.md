# Real-Time Deployment Events (Backend Module Doc)

## 1. Purpose
Broadcast deployment lifecycle events in real-time so the frontend can update UI without page reloads. Uses SSE (Server-Sent Events) for consistency with the existing log streaming architecture.

## 2. Entities

### DeployEvent
| Field | Type | Description |
|-------|------|-------------|
| type | EventType | Event category (see below) |
| app_id | string | App being deployed |
| app_name | string | Human-readable app name |
| deployment_id | string | Deployment ID |
| status | string | Current deployment status |
| image | string | Container image tag (optional) |
| message | string | Human-readable description |
| timestamp | time.Time | When the event occurred |

### Event Types
- `deployment_started` — Pipeline goroutine started
- `build_progress` — Reserved for future granular build output
- `build_complete` — Build produced image successfully
- `deploy_started` — K8s deployment initiated
- `deploy_complete` — App is live
- `deploy_failed` — Any error in build or deploy

## 3. Services (Use Cases)
`EventHub` — In-memory pub/sub broadcaster (same pattern as `LogHub`). Global fan-out (not per-deployment). Ring buffer history (50 entries) with replay on subscribe.

## 4. Ports
None — EventHub is a pure in-memory component with no external dependencies.

## 5. Adapters
None.

## 6. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/events` | SSE stream of deployment events (requires JWT) |
| GET | `/api/v1/events/history` | JSON array of recent events |

### SSE Event Format
```
event: deploy
data: {"type":"deploy_complete","app_id":"...","message":"my-app is live","timestamp":"..."}
```

## 7. Validation Rules
- JWT required for both endpoints
- No input validation needed (read-only)

## 8. Security Model
- JWT authentication via `RequireAuth` middleware
- Events are broadcast globally (all authenticated users see all events)
- No sensitive data in events (no secrets/env vars/credentials)

## 9. Testing Strategy
- Go build verification (`go build ./...`)
- Integration: trigger deployment, verify events appear on SSE stream
- Frontend: TypeScript type check (`npx tsc --noEmit`)

## 10. Future Improvements
- Per-user event filtering (only show events for apps the user owns)
- Event persistence to database for audit trail
- WebSocket upgrade if bidirectional communication is needed
- `build_progress` events with granular build output percentage
