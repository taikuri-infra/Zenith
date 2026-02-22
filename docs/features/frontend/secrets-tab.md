# Secrets Tab (Frontend Feature Doc)

## 1. Overview
The Secrets Tab is a panel within the app detail page (`/apps/[id]`) that lets users manage encrypted key-value secrets for their deployed applications. Secrets are encrypted at rest using AES-256-GCM on the backend.

## 2. UI/UX Flow
1. User navigates to an app detail page and clicks the **Secrets** tab (KeyRound icon)
2. Existing secrets are listed in a table showing only the key names (values are masked)
3. User can **Reveal** a secret by clicking the unlock button — this calls the backend decrypt API
4. Revealed values can be **Copied** to clipboard with a single click
5. User can **Hide** the revealed value to re-mask it
6. User can **Add** a new secret via the form at the top (key is auto-uppercased, only A-Z, 0-9, `_`)
7. User can **Delete** a secret with the trash icon (no confirmation, instant delete)

## 3. Data Flow
- **List**: `GET /api/v1/apps/:appId/secrets` → returns secret keys + metadata, never values
- **Reveal**: `GET /api/v1/apps/:appId/secrets/:key/value` → backend decrypts, returns plaintext
- **Add/Update**: `POST /api/v1/apps/:appId/secrets` with `{ key, value }` → backend encrypts with AES-256-GCM
- **Delete**: `DELETE /api/v1/apps/:appId/secrets/:key`

## 4. Components
- `SecretsTab` — Main component, manages state for add form, reveal map, loading states
- Uses shared: `PageWithTableSkeleton`, `ErrorState`, `EmptyState`

## 5. Services/API
- `appsDeploy.listSecrets(appId)` — list keys
- `appsDeploy.getSecretValue(appId, key)` — decrypt and return value
- `appsDeploy.setSecret(appId, key, value)` — encrypt and store
- `appsDeploy.deleteSecret(appId, key)` — remove

## 6. Hooks
- `useApi` — for listing secrets with loading/error states

## 7. State Logic
- `revealedValues: Record<string, string>` — cache of decrypted values keyed by secret name
- `revealingKey` / `deletingKey` — track which row is loading
- `copiedKey` — track clipboard copy feedback (2s timeout)

## 8. Edge Cases
- Empty state when no secrets exist
- Key input validation (uppercase, alphanumeric + underscore only)
- Loading spinners during API calls (add, reveal, delete)
- Dev mode: backend returns `nil` for SecretHandler when SECRETS_ENCRYPTION_KEY is empty

## 9. Security Considerations
- Values are typed into `<input type="password">` to prevent shoulder-surfing during input
- Decrypted values are only fetched on explicit "Reveal" click
- Values are held in component state only (not persisted client-side)
- Note about AES-256-GCM encryption is displayed to users

## 10. Testing Strategy
- TypeScript type check via `npx tsc --noEmit`
- Demo mode provides 3 mock secrets with mock decrypted values
- Visual verification in demo mode

## 11. Future Improvements
- Confirmation dialog before delete
- Toast notifications for success/error
- Audit log of secret access (who revealed what, when)
- Secret rotation reminders
