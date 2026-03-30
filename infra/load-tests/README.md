# Zenith Load Tests (k6)

## Prerequisites
- Install k6: `brew install k6`

## Run
```bash
k6 run infra/load-tests/auth-load.js
k6 run infra/load-tests/crud-load.js
k6 run infra/load-tests/spike-test.js
```

## Environment Variables
- `API_URL`: API base URL (default: `https://api.stage.freezenith.com`)
- `SMOKE_TEST_EMAIL`: Test user email
- `SMOKE_TEST_PASSWORD`: Test user password

Example with overrides:
```bash
k6 run -e API_URL=http://localhost:8080 -e SMOKE_TEST_EMAIL=test@example.com infra/load-tests/auth-load.js
```

## Rate Limiting
The staging APISIX gateway enforces 100 req/60s per IP on `/api/*` routes.
All tests account for this — sleep intervals are calculated to stay near (but not excessively over) the limit.
The spike test intentionally exceeds the limit to verify the API handles rate limiting gracefully.

## Test Descriptions

| Script | Purpose | VUs | Duration |
|--------|---------|-----|----------|
| `auth-load.js` | Login + /auth/me under load | 10+5 | ~2.5 min |
| `crud-load.js` | Read-only CRUD endpoints | 20 | ~3 min |
| `spike-test.js` | Sudden traffic spike resilience | 1-50 | ~2 min |
