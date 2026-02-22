# Releases Tab (Frontend Feature Doc)

## 1. Overview
The Releases Tab is a panel within the app detail page (`/apps/[id]`) that displays versioned image builds registered by CI pipelines via `zenith-actions`. Users can one-click deploy any release version.

## 2. UI/UX Flow
1. User navigates to an app detail page and clicks the **Releases** tab (Tag icon)
2. A table lists all releases sorted by newest first
3. The most recent release is highlighted with a "latest" badge
4. Each row shows: image tag, git SHA (truncated to 8 chars), branch, commit message, date
5. User clicks **Deploy** button → triggers async deployment
6. Button shows a spinner during deploy, then a "Triggered" confirmation for 3 seconds
7. All Deploy buttons are disabled while one deployment is in-flight

## 3. Data Flow
- **List**: `GET /api/v1/apps/:appId/releases` → returns array of release objects
- **Deploy**: `POST /api/v1/apps/:appId/releases/:releaseId/deploy` → creates deployment record, triggers async pipeline with pre-built image (skips build, goes straight to K8s deploy)

## 4. Components
- `ReleasesTab` — Main component, manages deploy state
- Uses shared: `PageWithTableSkeleton`, `ErrorState`, `EmptyState`

## 5. Services/API
- `appsDeploy.listReleases(appId)` — list all releases
- `appsDeploy.deployRelease(appId, releaseId)` — trigger deployment of specific version

## 6. Hooks
- `useApi` — for listing releases with loading/error states

## 7. State Logic
- `deployingId` — tracks which release is currently being deployed (shows spinner)
- `deployedId` — tracks which release was just deployed (shows ✓ Triggered for 3s)

## 8. Edge Cases
- Empty state when no releases exist (directs user to set up `zenith-actions`)
- Long commit messages are truncated with ellipsis
- Missing git SHA shows em-dash fallback

## 9. Security Considerations
- Deploy action requires JWT authentication
- CI/CD registers releases via API key auth

## 10. Testing Strategy
- TypeScript type check via `npx tsc --noEmit`
- Demo mode provides 4 mock releases with realistic data
- Visual verification in demo mode

## 11. Future Improvements
- Deployment status polling after triggering deploy
- Live SSE log streaming for the triggered deployment
- Release diff view (compare two releases)
- Release notes / changelog auto-generation from commit messages
