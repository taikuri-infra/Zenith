# Deploy Events (Frontend Feature Doc)

## 1. Overview
Real-time deployment status updates via SSE (Server-Sent Events). The Deploy page and App Detail page auto-refresh their data when deployment lifecycle events are received.

## 2. UI/UX Flow
1. User navigates to the Deploy page or App Detail page
2. An SSE connection is established to `GET /api/v1/events`
3. When a deployment event arrives, the page auto-refetches relevant data
4. App cards on the Deploy page update status dots (green/amber/red) in real-time
5. Deployment rows in the App Detail page update without manual refresh
6. Connection indicator is available (not yet shown in UI)

## 3. Data Flow
- SSE stream: `GET /api/v1/events?token=<jwt>` → EventSource
- Named event: `deploy` → JSON payload with `DeployEventData`
- Auto-reconnect after 5 seconds on connection loss
- Skipped in demo mode (`NEXT_PUBLIC_DEMO_MODE=true`)

## 4. Components
No new visual components — existing pages are wired to auto-refresh.

## 5. Services/API
- `getAccessToken()` from `api.ts` — provides JWT for SSE connection

## 6. Hooks
- `useDeployEvents(onEvent?)` — connects to SSE, dispatches events, returns `{ connected, lastEvent }`

## 7. State Logic
- `connected: boolean` — tracks SSE connection status
- `lastEvent: DeployEventData | null` — most recent event for inspection
- `onEventRef` — stable ref to avoid reconnection cycles

## 8. Edge Cases
- Demo mode: hook returns immediately without connecting
- No token: hook returns without connecting
- Server unavailable: reconnects every 5 seconds
- Multiple tabs: each tab gets its own SSE connection

## 9. Security Considerations
- JWT passed as query parameter (SSE limitation — EventSource doesn't support headers)
- No sensitive data in events

## 10. Testing Strategy
- `npx tsc --noEmit` (0 errors)
- Visual: deploy an app → observe status dot changing in real-time

## 11. Future Improvements
- Connection status indicator in the UI (green dot: "Live")
- Toast notifications on deploy_complete / deploy_failed
- Event log sidebar showing recent activity
