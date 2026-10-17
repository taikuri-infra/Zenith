# App Deploy Dashboard (Frontend Feature Doc)

## 1. Overview

Frontend dashboard pages for managing deployed apps, viewing deployment history, and configuring environment variables. Integrates with Phase 2 backend API.

## 2. UI/UX Flow

```
/apps (list) → click app → /apps/[id] (one page of cards, no tab bar)
                              ├── Header (type, status, branch, Open, Logs → /logs?app=<id>)
                              ├── Source + Health (web apps with health status)
                              ├── Recent Deployments (rollback + collapsible Releases)
                              ├── Environment (env vars + secret store)
                              ├── Resources (databases, buckets)
                              └── Domains (web apps)
```

## 3. Data Flow

All data fetched via `appsDeploy` API client → `useApi` hook → component state.

## 4. Components

| Component | Location | Purpose |
|-----------|----------|---------|
| `AppDetailPage` | `app/apps/[name]/page.tsx` | Header, banners, card grid |
| `SourceCard` / `HealthCard` | `source-card.tsx` / `health-card.tsx` | App details, health check |
| `DeploymentsCard` | `deployments-card.tsx` | Deployment history + rollback, Releases toggle |
| `EnvironmentCard` | `environment-card.tsx` | Env var CRUD + secret store rows |
| `ResourcesCard` / `DomainsCard` | `resources-card.tsx` / `domains-card.tsx` | Databases, buckets, custom domains |

## 5. Services/API

Uses `appsDeploy` from `lib/api.ts`:
- `get(id)` — App detail
- `listDeployments(appId)` — Deployment history
- `rollback(appId, deployId)` — Rollback
- `getEnvVars(appId)` / `setEnvVars()` / `deleteEnvVar()` — Env CRUD

## 6. Hooks

- `useApi(fetcher, deps)` — Auto-fetch with loading/error states
- `useParams()` — Get `id` from URL

## 7. State Logic

- No tab state: every card owns its own data fetching (`useApi`)
- `showValues` — Toggle env var value visibility
- `newKey` / `newValue` — Controlled inputs for adding env vars

## 8. Edge Cases

- App not found → `EmptyState` component
- No deployments yet → "Push to repository" empty state
- No env vars → "Add environment variables" empty state
- Rollback of already active deployment → hidden button

## 9. Security Considerations

- JWT token auto-attached by `apiFetch` wrapper
- Env var values hidden by default (toggle to show)
- No `dangerouslySetInnerHTML`

## 10. Testing Strategy

- Build verification: `npx next build` passes
- Visual verification: manual testing in browser
- API integration: depends on backend running

## 11. Future Improvements

- Real-time deployment progress via WebSocket
- Build log streaming in Deployments tab
- Custom domain management in a new tab
- Deploy from UI button (trigger build without git push)
