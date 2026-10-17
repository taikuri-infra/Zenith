# Environment Card: Env Vars + Secret Store (Frontend Feature Doc)

## 1. Overview
The former Secrets tab is merged into the **Environment** card of the app detail page (`/apps/[id]`, one page of cards, no tab bar). The card shows the app's per-environment env vars and its secret-store entries in **one table**. Secret-store values are encrypted at rest with AES-256-GCM on the backend and shared by all environments.

## 2. UI/UX Flow
1. The Environment card (full width) has a Production / Staging toggle ("Live environment" badge on Production; Staging disabled when the project has no staging environment), **Import .env** and **Show/Hide values**
2. The table lists env vars of the active environment (lock icon for `is_secret`, a read-only badge for variables injected by an attached database) followed by secret-store rows tagged `secret store`; a key present in both stores appears as two rows
3. Env var rows: inline **Edit** (secrets must be re-entered) and **Delete** with confirm; managed rows are read-only
4. Secret-store rows: **Reveal** fetches the value on demand (cached, so Hide/Reveal does not refetch), **Copy** (✓ feedback for 2s), **Delete** with confirm
5. One **Add** form: Key, Value, Secret toggle; when Secret is on, a **Store** select picks "Env var (secret)" (default, `is_secret` env var) or "Secret store". Secret-store keys keep only A-Z, 0-9 and `_`; a double submit is ignored
6. Any env var change (add, edit, delete, import) shows a banner with **Restart now** (`envVarsV2.apply`) and Dismiss

## 3. Data Flow
- Env vars: `envVarsV2.list/set/delete/importDotEnv(appId, …, envId)`, `envVarsV2.apply(appId)`; staging env id from `environments.list(projectId)`
- Secret store: `GET /api/v1/apps/:appId/secrets` (keys only), `GET …/secrets/:key/value` (decrypt), `POST …/secrets` (`{ key, value }`), `DELETE …/secrets/:key`

## 4. Components
- `EnvironmentCard` in `app/apps/[name]/environment-card.tsx`
- Uses shared: `TableSkeleton`, `ErrorState`, `EmptyState`, `useToast`

## 5. Services/API
- `envVarsV2.*`, `environments.list`
- `appsDeploy.listSecrets`, `getSecretValue`, `setSecret`, `deleteSecret`

## 6. Hooks
- `useApi` — env vars and secret store load (and fail) independently

## 7. State Logic
- `secretValues` (fetched values cache) and `shownSecrets` (visibility) per key
- `copiedKey` with a timer held in a ref and cleared on unmount
- `pendingRestart` drives the Restart-now banner

## 8. Edge Cases
- One store failing shows its own error (with Try Again) while the other store's rows stay
- `listSecrets` returning **404** means the secret store is not configured on this install (backend has no `SECRETS_ENCRYPTION_KEY`): no error, no store rows, and the Add form offers no "Secret store" option
- Empty state only when both stores are empty and neither failed

## 9. Security Considerations
- Values typed into a password input while Secret is on
- Secret-store values only fetched on explicit Reveal, held in component state only
- Keys are sanitised because the secrets API puts the key in the URL path

## 10. Testing Strategy
- `app/apps/[name]/__tests__/environment-card.test.tsx` (merged table, reveal once, add to either store, per-store errors, 404 not configured, Restart now, copy timer cleanup)
- `pnpm exec tsc --noEmit`, `pnpm lint`

## 11. Future Improvements
- Audit log of secret access
- Secret rotation reminders
