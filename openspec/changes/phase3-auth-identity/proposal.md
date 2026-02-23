# Phase 3: Auth & Identity — Real Login/Register Flow

## Summary

Wire the auth service so users can register, log in, and access Mission Control and Web Platform with real JWT authentication on staging.

## Prerequisites

- Phase 2 complete (platform running on staging)
- API server accessible at `api.stage.freezenith.com`

## Steps

### Step 3.1: Seed Admin User on Startup

**What:** API server creates admin user on first boot using `ADMIN_EMAIL` + `ADMIN_PASSWORD` env vars.

**Build:**
- Verify `main.go` seeds admin user when `DATABASE_URL` is set
- Admin user gets `RoleOwner` (highest privilege)
- If admin already exists, skip seeding (idempotent)

**Your manual work:** None — credentials are in Terraform secrets.

**Verify:**
```bash
curl -s -X POST https://api.stage.freezenith.com/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@freezenith.com","password":"YOUR_PASSWORD"}'
# Returns: {"access_token":"...", "refresh_token":"..."}
```

### Step 3.2: MC Login Integration

**What:** Mission Control connects to real API for authentication.

**Build:**
- MC production build uses `NEXT_PUBLIC_API_URL=https://api.stage.freezenith.com`
- Login page calls real `/api/v1/auth/login`
- JWT stored in localStorage, sent with all API calls
- Shell component enforces auth (redirects to `/login` if no token)

**Your manual work:**
- Update Helm values: `mc.env.NEXT_PUBLIC_API_URL`

**Verify:**
1. Go to `https://ms.stage.freezenith.com/login`
2. Login with admin email + password
3. Dashboard loads with real data (not demo data)

### Step 3.3: Web Platform Login Integration

**What:** Web Platform connects to real API for user authentication.

**Build:**
- Same as MC but with user-scoped endpoints
- Register flow creates new user accounts
- Project selection after login

**Your manual work:**
- Update Helm values: `web.env.NEXT_PUBLIC_API_URL`

**Verify:**
1. Go to `https://cloud.stage.freezenith.com/login`
2. Register a new user
3. Login and see the dashboard

### Step 3.4: Token Refresh Flow

**What:** Access tokens auto-refresh using refresh tokens.

**Build:**
- API `tryRefreshToken()` in frontend catches 401, calls `/api/v1/auth/refresh`
- New access token replaces old one in localStorage
- If refresh fails, redirect to login

**Verify:**
- Wait 1+ hours, try to use MC — should auto-refresh without login prompt

## Acceptance Criteria

- [ ] Admin can log in to MC with real credentials
- [ ] New users can register via Web Platform
- [ ] JWT authentication works end-to-end
- [ ] Token refresh works transparently
- [ ] Invalid tokens redirect to login page
- [ ] Demo mode still works on demo endpoints (separate deployments)
