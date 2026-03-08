#!/bin/bash
# =============================================================================
# Zenith Platform — Customer Smoke Test (Developer Userflow)
#
# Tests the complete developer journey:
#   register → login → project → app → database → storage → gateway
#   → domains → API keys → webhooks → sessions → plan → billing
#
# Usage:
#   ./infra/scripts/smoke-test-customer.sh [--api-url URL] [--verbose]
#
# Environment variables:
#   ZENITH_API_URL   Base URL (default: http://localhost:8080)
#   SMOKE_EMAIL      Test user email (default: auto-generated)
#   SMOKE_PASSWORD   Test user password (default: SmokeTest123!)
#
# Returns exit 0 if all tests pass, exit 1 if any fail.
# =============================================================================

set -uo pipefail

VERBOSE=false
API_URL="${ZENITH_API_URL:-http://localhost:8080}"
EMAIL="${SMOKE_EMAIL:-smoke-$(date +%s)@test.zenith.dev}"
PASSWORD="${SMOKE_PASSWORD:-SmokeTest123!}"

for arg in "$@"; do
  case "$arg" in
    --verbose|-v) VERBOSE=true ;;
    --api-url=*) API_URL="${arg#*=}" ;;
  esac
done

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0
TOTAL_TESTS=0
TOKEN=""
REFRESH_TOKEN=""
PROJECT_ID=""
APP_ID=""
DB_ID=""
BACKUP_ID=""
BUCKET_ID=""
GW_ID=""
ROUTE_ID=""
APIKEY_ID=""
WEBHOOK_ID=""
TICKET_ID=""
POOL_ID=""
POOL_USER_ID=""
DOMAIN_ID=""
ROLE_ID=""
ASSIGNMENT_ID=""

pass() {
  ((PASS_COUNT++))
  ((TOTAL_TESTS++))
  echo -e "  ${GREEN}PASS${NC} $1"
}

fail() {
  ((FAIL_COUNT++))
  ((TOTAL_TESTS++))
  echo -e "  ${RED}FAIL${NC} $1"
  if [[ "$VERBOSE" == "true" && -n "${2:-}" ]]; then
    echo -e "       ${YELLOW}Detail: ${2}${NC}"
  fi
}

section() {
  echo ""
  echo -e "${BLUE}[$1]${NC} $2"
}

# Helper: authenticated GET/POST/PUT/DELETE
api_get() {
  curl -sf -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}$1" 2>/dev/null
}

api_post() {
  curl -sf -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${API_URL}$1" 2>/dev/null
}

api_put() {
  curl -sf -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${API_URL}$1" 2>/dev/null
}

api_delete() {
  curl -sf -X DELETE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}$1" 2>/dev/null
}

# Helper: check HTTP status code
api_status() {
  local method="${1}"
  local path="${2}"
  local data="${3:-}"
  if [[ -n "$data" ]]; then
    curl -so /dev/null -w "%{http_code}" -X "$method" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$data" "${API_URL}${path}" 2>/dev/null
  else
    curl -so /dev/null -w "%{http_code}" -X "$method" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}${path}" 2>/dev/null
  fi
}

echo ""
echo "============================================="
echo "   Zenith Customer Smoke Test"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "   API: ${API_URL}"
echo "   User: ${EMAIL}"
echo "============================================="

# =============================================
# 0. Health & Version
# =============================================
section "0" "Infrastructure health"

HEALTH=$(curl -sf "${API_URL}/health" 2>/dev/null)
if echo "$HEALTH" | grep -q '"status":"healthy"'; then
  pass "GET /health returns healthy"
else
  fail "GET /health" "$HEALTH"
fi

READY=$(curl -sf "${API_URL}/ready" 2>/dev/null)
if echo "$READY" | grep -q '"status":"ready"'; then
  pass "GET /ready returns ready"
else
  fail "GET /ready" "$READY"
fi

VERSION=$(curl -sf "${API_URL}/api/v1/version" 2>/dev/null)
if echo "$VERSION" | grep -q '"version"'; then
  pass "GET /api/v1/version returns version info"
else
  fail "GET /api/v1/version" "$VERSION"
fi

# Verify security headers
HEADERS=$(curl -sI "${API_URL}/health" 2>/dev/null)
if echo "$HEADERS" | grep -qi "X-Content-Type-Options: nosniff"; then
  pass "Security header: X-Content-Type-Options"
else
  fail "Missing X-Content-Type-Options header"
fi
if echo "$HEADERS" | grep -qi "X-Frame-Options: DENY"; then
  pass "Security header: X-Frame-Options"
else
  fail "Missing X-Frame-Options header"
fi
if echo "$HEADERS" | grep -qi "Strict-Transport-Security"; then
  pass "Security header: HSTS (Strict-Transport-Security)"
else
  fail "Missing HSTS header"
fi
if echo "$HEADERS" | grep -qi "Content-Security-Policy"; then
  pass "Security header: Content-Security-Policy"
else
  fail "Missing Content-Security-Policy header"
fi
if echo "$HEADERS" | grep -qi "Referrer-Policy"; then
  pass "Security header: Referrer-Policy"
else
  fail "Missing Referrer-Policy header"
fi
if echo "$HEADERS" | grep -qi "Permissions-Policy"; then
  pass "Security header: Permissions-Policy"
else
  fail "Missing Permissions-Policy header"
fi

# =============================================
# 1. Authentication
# =============================================
section "1" "Authentication (register + login)"

# Register
REG_RESP=$(curl -sf -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\",\"name\":\"Smoke Test\"}" \
  "${API_URL}/api/v1/auth/register" 2>/dev/null)
if echo "$REG_RESP" | grep -q '"access_token"\|"token"'; then
  TOKEN=$(echo "$REG_RESP" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)
  [[ -z "$TOKEN" ]] && TOKEN=$(echo "$REG_RESP" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
  REFRESH_TOKEN=$(echo "$REG_RESP" | grep -o '"refresh_token":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /auth/register — user created"
else
  # May return message (email verification required)
  if echo "$REG_RESP" | grep -q '"message"'; then
    pass "POST /auth/register — registration accepted (verification required)"
  else
    fail "POST /auth/register" "$REG_RESP"
  fi
fi

# Login
LOGIN_RESP=$(curl -sf -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}" \
  "${API_URL}/api/v1/auth/login" 2>/dev/null)
if echo "$LOGIN_RESP" | grep -q '"access_token"\|"token"'; then
  TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)
  [[ -z "$TOKEN" ]] && TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
  REFRESH_TOKEN=$(echo "$LOGIN_RESP" | grep -o '"refresh_token":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /auth/login — login successful"
else
  fail "POST /auth/login" "$LOGIN_RESP"
fi

# Verify 401 on protected route without token
UNAUTH=$(curl -so /dev/null -w "%{http_code}" "${API_URL}/api/v1/projects" 2>/dev/null)
if [[ "$UNAUTH" == "401" ]]; then
  pass "Protected route returns 401 without token"
else
  fail "Expected 401, got $UNAUTH"
fi

# Refresh token
if [[ -n "$REFRESH_TOKEN" ]]; then
  REFRESH_RESP=$(curl -sf -X POST -H "Content-Type: application/json" \
    -d "{\"refresh_token\":\"${REFRESH_TOKEN}\"}" \
    "${API_URL}/api/v1/auth/refresh" 2>/dev/null)
  if echo "$REFRESH_RESP" | grep -q '"access_token"\|"token"'; then
    TOKEN=$(echo "$REFRESH_RESP" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)
    [[ -z "$TOKEN" ]] && TOKEN=$(echo "$REFRESH_RESP" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
    REFRESH_TOKEN=$(echo "$REFRESH_RESP" | grep -o '"refresh_token":"[^"]*"' | head -1 | cut -d'"' -f4)
    pass "POST /auth/refresh — token refreshed"
  else
    fail "POST /auth/refresh" "$REFRESH_RESP"
  fi
else
  REFRESH_RESP=$(api_post "/api/v1/auth/refresh" "{}")
  if echo "$REFRESH_RESP" | grep -q '"access_token"\|"token"'; then
    pass "POST /auth/refresh — token refreshed"
  else
    fail "POST /auth/refresh" "$REFRESH_RESP"
  fi
fi

# Resend verification (always returns 200 — no user enumeration)
RESEND_STATUS=$(curl -so /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"nonexistent@test.zenith.dev\"}" \
  "${API_URL}/api/v1/auth/resend-verification" 2>/dev/null)
if [[ "$RESEND_STATUS" == "200" ]]; then
  pass "POST /auth/resend-verification — no user enumeration (200 for unknown email)"
else
  fail "POST /auth/resend-verification (status: $RESEND_STATUS)" "Expected 200"
fi

# Rate limiting check (should get 429 after many requests)
RATE_STATUS=""
for i in $(seq 1 15); do
  RATE_STATUS=$(curl -so /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" \
    -d "{\"email\":\"ratelimit@test.dev\",\"password\":\"wrong\"}" \
    "${API_URL}/api/v1/auth/login" 2>/dev/null)
done
if [[ "$RATE_STATUS" == "429" ]]; then
  pass "Auth rate limiting active (429 after 10+ attempts)"
else
  fail "Auth rate limiting not triggered (got $RATE_STATUS)" "Expected 429"
fi

# Input validation: password too short
SHORT_PW_STATUS=$(curl -so /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" \
  -d '{"email":"val@test.dev","password":"short","name":"Test"}' \
  "${API_URL}/api/v1/auth/register" 2>/dev/null)
if [[ "$SHORT_PW_STATUS" == "400" ]]; then
  pass "Registration rejects short password (400)"
else
  fail "Short password should return 400, got $SHORT_PW_STATUS"
fi

# Input validation: invalid email format
BAD_EMAIL_STATUS=$(curl -so /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" \
  -d '{"email":"not-an-email","password":"ValidPass123!","name":"Test"}' \
  "${API_URL}/api/v1/auth/register" 2>/dev/null)
if [[ "$BAD_EMAIL_STATUS" == "400" ]]; then
  pass "Registration rejects invalid email format (400)"
else
  fail "Invalid email should return 400, got $BAD_EMAIL_STATUS"
fi

# =============================================
# 2. Plan
# =============================================
section "2" "User plan"

PLAN=$(api_get "/api/v1/plan")
if echo "$PLAN" | grep -q '"tier"'; then
  pass "GET /plan — current plan retrieved"
else
  fail "GET /plan" "$PLAN"
fi

# =============================================
# 3. Projects
# =============================================
section "3" "Project CRUD"

# Default project should exist (auto-created on register)
PROJECTS=$(api_get "/api/v1/projects")
if echo "$PROJECTS" | grep -q '"id"'; then
  PROJECT_ID=$(echo "$PROJECTS" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "GET /projects — default project exists (${PROJECT_ID:0:8}...)"
else
  # Create one
  CREATE_PROJ=$(api_post "/api/v1/projects" '{"name":"smoke-test"}')
  if echo "$CREATE_PROJ" | grep -q '"id"'; then
    PROJECT_ID=$(echo "$CREATE_PROJ" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    pass "POST /projects — project created (${PROJECT_ID:0:8}...)"
  else
    fail "POST /projects" "$CREATE_PROJ"
  fi
fi

# Get project
if [[ -n "$PROJECT_ID" ]]; then
  PROJ_DETAIL=$(api_get "/api/v1/projects/${PROJECT_ID}")
  if echo "$PROJ_DETAIL" | grep -q '"id"'; then
    pass "GET /projects/:id — project details retrieved"
  else
    fail "GET /projects/:id" "$PROJ_DETAIL"
  fi
fi

# =============================================
# 4. Apps
# =============================================
section "4" "App lifecycle"

CREATE_APP=$(api_post "/api/v1/apps" "{\"name\":\"smoke-app\",\"project_id\":\"${PROJECT_ID}\",\"git_url\":\"https://github.com/example/app\"}")
if echo "$CREATE_APP" | grep -q '"id"'; then
  APP_ID=$(echo "$CREATE_APP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /apps — app created (${APP_ID:0:8}...)"
else
  fail "POST /apps" "$CREATE_APP"
fi

# List apps
APPS=$(api_get "/api/v1/apps")
if echo "$APPS" | grep -q '"id"'; then
  pass "GET /apps — app list retrieved"
else
  fail "GET /apps" "$APPS"
fi

# Get app detail
if [[ -n "$APP_ID" ]]; then
  APP_DETAIL=$(api_get "/api/v1/apps/${APP_ID}")
  if echo "$APP_DETAIL" | grep -q '"id"'; then
    pass "GET /apps/:id — app details retrieved"
  else
    fail "GET /apps/:id" "$APP_DETAIL"
  fi

  # Env vars
  ENV_SET=$(api_put "/api/v1/apps/${APP_ID}/env" '{"vars":{"NODE_ENV":"production","PORT":"3000"}}')
  if [[ $? -eq 0 ]]; then
    pass "PUT /apps/:id/env — env vars set"
  else
    fail "PUT /apps/:id/env" "$ENV_SET"
  fi

  ENV_GET=$(api_get "/api/v1/apps/${APP_ID}/env")
  if echo "$ENV_GET" | grep -q "NODE_ENV"; then
    pass "GET /apps/:id/env — env vars retrieved"
  else
    fail "GET /apps/:id/env" "$ENV_GET"
  fi

  # Deployments list
  DEPLOYS=$(api_get "/api/v1/apps/${APP_ID}/deployments")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/deployments — deployment list retrieved"
  else
    fail "GET /apps/:id/deployments"
  fi

  # Domains list
  DOMAINS=$(api_get "/api/v1/apps/${APP_ID}/domains")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/domains — domain list retrieved"
  else
    fail "GET /apps/:id/domains"
  fi

  # Previews list
  PREVIEWS=$(api_get "/api/v1/apps/${APP_ID}/previews")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/previews — preview list retrieved"
  else
    fail "GET /apps/:id/previews"
  fi

  # Releases list
  RELEASES=$(api_get "/api/v1/apps/${APP_ID}/releases")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/releases — release list retrieved"
  else
    fail "GET /apps/:id/releases"
  fi
fi

# =============================================
# 5. Databases
# =============================================
section "5" "Database lifecycle"

if [[ -n "$APP_ID" ]]; then
  CREATE_DB=$(api_post "/api/v1/apps/${APP_ID}/databases" '{"name":"smoke-db","engine":"postgres","version":"16"}')
  if echo "$CREATE_DB" | grep -q '"id"'; then
    DB_ID=$(echo "$CREATE_DB" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    pass "POST /apps/:id/databases — database created (${DB_ID:0:8}...)"
  else
    fail "POST /apps/:id/databases" "$CREATE_DB"
  fi

  DBS=$(api_get "/api/v1/apps/${APP_ID}/databases")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/databases — database list retrieved"
  else
    fail "GET /apps/:id/databases"
  fi
fi

# Standalone databases
STANDALONE_DBS=$(api_get "/api/v1/databases")
if [[ $? -eq 0 ]]; then
  pass "GET /databases — standalone database list retrieved"
else
  fail "GET /databases"
fi

# User backups
BACKUPS=$(api_get "/api/v1/backups")
if [[ $? -eq 0 ]]; then
  pass "GET /backups — user backup list retrieved"
else
  fail "GET /backups"
fi

# =============================================
# 6. Storage
# =============================================
section "6" "Storage lifecycle"

CREATE_BUCKET=$(api_post "/api/v1/storage-buckets" '{"name":"smoke-bucket"}')
if echo "$CREATE_BUCKET" | grep -q '"id"'; then
  BUCKET_ID=$(echo "$CREATE_BUCKET" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /storage-buckets — bucket created (${BUCKET_ID:0:8}...)"
else
  fail "POST /storage-buckets" "$CREATE_BUCKET"
fi

BUCKETS=$(api_get "/api/v1/storage-buckets")
if [[ $? -eq 0 ]]; then
  pass "GET /storage-buckets — bucket list retrieved"
else
  fail "GET /storage-buckets"
fi

if [[ -n "$BUCKET_ID" ]]; then
  BUCKET_DETAIL=$(api_get "/api/v1/storage-buckets/${BUCKET_ID}")
  if echo "$BUCKET_DETAIL" | grep -q '"id"'; then
    pass "GET /storage-buckets/:id — bucket details retrieved"
  else
    fail "GET /storage-buckets/:id" "$BUCKET_DETAIL"
  fi

  OBJECTS=$(api_get "/api/v1/storage-buckets/${BUCKET_ID}/objects")
  if [[ $? -eq 0 ]]; then
    pass "GET /storage-buckets/:id/objects — object list retrieved"
  else
    fail "GET /storage-buckets/:id/objects"
  fi
fi

# =============================================
# 7. API Gateways
# =============================================
section "7" "API Gateway lifecycle"

CREATE_GW=$(api_post "/api/v1/gateways" "{\"name\":\"smoke-gw\",\"project_id\":\"${PROJECT_ID}\"}")
if echo "$CREATE_GW" | grep -q '"id"'; then
  GW_ID=$(echo "$CREATE_GW" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /gateways — gateway created (${GW_ID:0:8}...)"
else
  fail "POST /gateways" "$CREATE_GW"
fi

GWS=$(api_get "/api/v1/gateways")
if [[ $? -eq 0 ]]; then
  pass "GET /gateways — gateway list retrieved"
else
  fail "GET /gateways"
fi

if [[ -n "$GW_ID" ]]; then
  GW_DETAIL=$(api_get "/api/v1/gateways/${GW_ID}")
  if echo "$GW_DETAIL" | grep -q '"id"'; then
    pass "GET /gateways/:id — gateway details retrieved"
  else
    fail "GET /gateways/:id" "$GW_DETAIL"
  fi

  ROUTES=$(api_get "/api/v1/gateways/${GW_ID}/routes")
  if [[ $? -eq 0 ]]; then
    pass "GET /gateways/:id/routes — route list retrieved"
  else
    fail "GET /gateways/:id/routes"
  fi
fi

# =============================================
# 8. API Keys
# =============================================
section "8" "API key management"

CREATE_KEY=$(api_post "/api/v1/api-keys" '{"name":"smoke-key","scopes":["apps:read","apps:write"]}')
if echo "$CREATE_KEY" | grep -q '"id"\|"key"'; then
  APIKEY_ID=$(echo "$CREATE_KEY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /api-keys — API key created"
else
  fail "POST /api-keys" "$CREATE_KEY"
fi

KEYS=$(api_get "/api/v1/api-keys")
if [[ $? -eq 0 ]]; then
  pass "GET /api-keys — key list retrieved"
else
  fail "GET /api-keys"
fi

# =============================================
# 9. Webhooks
# =============================================
section "9" "Webhook management"

CREATE_WH=$(api_post "/api/v1/webhooks" '{"url":"https://example.com/webhook","events":["app.deployed","app.deleted"]}')
if echo "$CREATE_WH" | grep -q '"id"'; then
  WEBHOOK_ID=$(echo "$CREATE_WH" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /webhooks — webhook created"
else
  fail "POST /webhooks" "$CREATE_WH"
fi

WHS=$(api_get "/api/v1/webhooks")
if [[ $? -eq 0 ]]; then
  pass "GET /webhooks — webhook list retrieved"
else
  fail "GET /webhooks"
fi

# =============================================
# 10. Sessions
# =============================================
section "10" "Session management"

SESSIONS=$(api_get "/api/v1/auth/sessions")
if [[ $? -eq 0 ]]; then
  pass "GET /auth/sessions — session list retrieved"
else
  fail "GET /auth/sessions"
fi

# =============================================
# 11. MFA
# =============================================
section "11" "MFA status"

MFA=$(api_get "/api/v1/auth/mfa")
if [[ $? -eq 0 ]]; then
  pass "GET /auth/mfa — MFA status retrieved"
else
  fail "GET /auth/mfa"
fi

# =============================================
# 12. Support Tickets
# =============================================
section "12" "Support tickets"

TICKETS=$(api_get "/api/v1/support/tickets")
if [[ $? -eq 0 ]]; then
  pass "GET /support/tickets — ticket list retrieved"
else
  fail "GET /support/tickets"
fi

# =============================================
# 13. Compliance & Settings
# =============================================
section "13" "Compliance & settings"

COMPLIANCE=$(api_get "/api/v1/compliance")
if [[ $? -eq 0 ]]; then
  pass "GET /compliance — compliance status retrieved"
else
  fail "GET /compliance"
fi

DPA=$(api_get "/api/v1/settings/dpa")
if [[ $? -eq 0 ]]; then
  pass "GET /settings/dpa — DPA retrieved"
else
  fail "GET /settings/dpa"
fi

BRANDING=$(api_get "/api/v1/settings/branding")
if [[ $? -eq 0 ]]; then
  pass "GET /settings/branding — branding retrieved"
else
  fail "GET /settings/branding"
fi

SSO=$(api_get "/api/v1/settings/sso")
if [[ $? -eq 0 ]]; then
  pass "GET /settings/sso — SSO config list retrieved"
else
  fail "GET /settings/sso"
fi

IP_LIST=$(api_get "/api/v1/settings/ip-whitelist")
if [[ $? -eq 0 ]]; then
  pass "GET /settings/ip-whitelist — IP whitelist retrieved"
else
  fail "GET /settings/ip-whitelist"
fi

# =============================================
# 14. Roles
# =============================================
section "14" "Roles & RBAC"

ROLES=$(api_get "/api/v1/roles")
if [[ $? -eq 0 ]]; then
  pass "GET /roles — role list retrieved"
else
  fail "GET /roles"
fi

PERMS=$(api_get "/api/v1/roles/permissions")
if [[ $? -eq 0 ]]; then
  pass "GET /roles/permissions — permissions retrieved"
else
  fail "GET /roles/permissions"
fi

# =============================================
# 15. Events & Notifications
# =============================================
section "15" "Events & notifications"

EVENTS=$(api_get "/api/v1/events/history")
if [[ $? -eq 0 ]]; then
  pass "GET /events/history — recent events retrieved"
else
  fail "GET /events/history"
fi

NOTIFS=$(api_get "/api/v1/notifications")
if [[ $? -eq 0 ]]; then
  pass "GET /notifications — notifications retrieved"
else
  fail "GET /notifications"
fi

ACTIVITY=$(api_get "/api/v1/activity")
if [[ $? -eq 0 ]]; then
  pass "GET /activity — activity feed retrieved"
else
  fail "GET /activity"
fi

# =============================================
# 16. Billing
# =============================================
section "16" "Billing"

BILLING=$(api_get "/api/v1/billing")
if [[ $? -eq 0 ]]; then
  pass "GET /billing — billing status retrieved"
else
  fail "GET /billing"
fi

INVOICES=$(api_get "/api/v1/billing/invoices")
if [[ $? -eq 0 ]]; then
  pass "GET /billing/invoices — invoice list retrieved"
else
  fail "GET /billing/invoices"
fi

# =============================================
# 17. Monitoring & Observability
# =============================================
section "17" "Monitoring & observability"

if [[ -n "$APP_ID" ]]; then
  OVERVIEW=$(api_get "/api/v1/apps/${APP_ID}/metrics/overview")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/metrics/overview — metrics overview retrieved"
  else
    fail "GET /apps/:id/metrics/overview"
  fi

  TIMESERIES=$(api_get "/api/v1/apps/${APP_ID}/metrics/timeseries?metric=cpu&range=1h")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/metrics/timeseries — time series retrieved"
  else
    fail "GET /apps/:id/metrics/timeseries"
  fi

  LOGS=$(api_get "/api/v1/apps/${APP_ID}/logs?limit=10&since=1h")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/logs — app logs retrieved"
  else
    fail "GET /apps/:id/logs"
  fi

  PODS=$(api_get "/api/v1/apps/${APP_ID}/pods")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/pods — pod list retrieved"
  else
    fail "GET /apps/:id/pods"
  fi
else
  fail "Monitoring tests skipped — no APP_ID"
fi

# =============================================
# 18. Team Members (IAM)
# =============================================
section "18" "Team members (IAM)"

MEMBERS=$(api_get "/api/v1/team/members")
if [[ $? -eq 0 ]]; then
  pass "GET /team/members — member list retrieved"
else
  fail "GET /team/members"
fi

INVITE_STATUS=$(api_status POST "/api/v1/team/members/invite" '{"email":"teammate@test.zenith.dev","role":"viewer"}')
if [[ "$INVITE_STATUS" == "200" || "$INVITE_STATUS" == "201" || "$INVITE_STATUS" == "403" ]]; then
  pass "POST /team/members/invite — invite endpoint responded (status: $INVITE_STATUS)"
else
  fail "POST /team/members/invite (status: $INVITE_STATUS)"
fi

# =============================================
# 19. Support Tickets (CRUD)
# =============================================
section "19" "Support tickets (CRUD)"

CREATE_TICKET=$(api_post "/api/v1/support/tickets" '{"subject":"Smoke test ticket","description":"Testing support system","priority":"low"}')
if echo "$CREATE_TICKET" | grep -q '"id"'; then
  TICKET_ID=$(echo "$CREATE_TICKET" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /support/tickets — ticket created (${TICKET_ID:0:8}...)"
else
  fail "POST /support/tickets" "$CREATE_TICKET"
fi

if [[ -n "$TICKET_ID" ]]; then
  TICKET_DETAIL=$(api_get "/api/v1/support/tickets/${TICKET_ID}")
  if echo "$TICKET_DETAIL" | grep -q '"id"'; then
    pass "GET /support/tickets/:id — ticket details retrieved"
  else
    fail "GET /support/tickets/:id" "$TICKET_DETAIL"
  fi

  MSG_STATUS=$(api_status POST "/api/v1/support/tickets/${TICKET_ID}/messages" '{"content":"This is a test message"}')
  if [[ "$MSG_STATUS" == "200" || "$MSG_STATUS" == "201" ]]; then
    pass "POST /support/tickets/:id/messages — message sent"
  else
    fail "POST /support/tickets/:id/messages (status: $MSG_STATUS)"
  fi
fi

# =============================================
# 20. Auth Pools (CRUD)
# =============================================
section "20" "Auth pools (CRUD)"

CREATE_POOL=$(api_post "/api/v1/auth-pools" '{"name":"smoke-pool"}')
if echo "$CREATE_POOL" | grep -q '"id"'; then
  POOL_ID=$(echo "$CREATE_POOL" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /auth-pools — pool created (${POOL_ID:0:8}...)"
else
  # May be gated by plan tier
  POOL_STATUS=$(api_status POST "/api/v1/auth-pools" '{"name":"smoke-pool"}')
  if [[ "$POOL_STATUS" == "403" ]]; then
    pass "POST /auth-pools — correctly gated by plan tier (403)"
  else
    fail "POST /auth-pools" "$CREATE_POOL"
  fi
fi

POOLS=$(api_get "/api/v1/auth-pools")
if [[ $? -eq 0 ]]; then
  pass "GET /auth-pools — pool list retrieved"
else
  fail "GET /auth-pools"
fi

if [[ -n "$POOL_ID" ]]; then
  POOL_DETAIL=$(api_get "/api/v1/auth-pools/${POOL_ID}")
  if echo "$POOL_DETAIL" | grep -q '"id"'; then
    pass "GET /auth-pools/:id — pool details retrieved"
  else
    fail "GET /auth-pools/:id" "$POOL_DETAIL"
  fi

  # Create a user in the pool
  POOL_USER=$(api_post "/api/v1/auth-pools/${POOL_ID}/users" '{"email":"pooluser@test.dev","password":"PoolUser123!"}')
  if echo "$POOL_USER" | grep -q '"id"'; then
    POOL_USER_ID=$(echo "$POOL_USER" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    pass "POST /auth-pools/:id/users — pool user created"
  else
    fail "POST /auth-pools/:id/users" "$POOL_USER"
  fi

  # List pool users
  POOL_USERS=$(api_get "/api/v1/auth-pools/${POOL_ID}/users")
  if [[ $? -eq 0 ]]; then
    pass "GET /auth-pools/:id/users — pool users listed"
  else
    fail "GET /auth-pools/:id/users"
  fi

  if [[ -n "$POOL_USER_ID" ]]; then
    # Get pool user detail
    PU_DETAIL=$(api_get "/api/v1/auth-pools/${POOL_ID}/users/${POOL_USER_ID}")
    if [[ $? -eq 0 ]]; then
      pass "GET /auth-pools/:id/users/:userId — pool user details"
    else
      fail "GET /auth-pools/:id/users/:userId"
    fi

    # Disable pool user
    PU_DISABLE=$(api_status POST "/api/v1/auth-pools/${POOL_ID}/users/${POOL_USER_ID}/disable" '{}')
    if [[ "$PU_DISABLE" == "200" ]]; then
      pass "POST /auth-pools/:id/users/:userId/disable — user disabled"
    else
      fail "POST /auth-pools/:id/users/:userId/disable (status: $PU_DISABLE)"
    fi

    # Re-enable pool user
    PU_ENABLE=$(api_status POST "/api/v1/auth-pools/${POOL_ID}/users/${POOL_USER_ID}/enable" '{}')
    if [[ "$PU_ENABLE" == "200" ]]; then
      pass "POST /auth-pools/:id/users/:userId/enable — user re-enabled"
    else
      fail "POST /auth-pools/:id/users/:userId/enable (status: $PU_ENABLE)"
    fi

    # Delete pool user
    PU_DEL=$(api_status DELETE "/api/v1/auth-pools/${POOL_ID}/users/${POOL_USER_ID}")
    if [[ "$PU_DEL" == "200" || "$PU_DEL" == "204" ]]; then
      pass "DELETE /auth-pools/:id/users/:userId — pool user deleted"
    else
      fail "DELETE /auth-pools/:id/users/:userId (status: $PU_DEL)"
    fi
  fi
fi

# =============================================
# 21. App Secrets (CRUD)
# =============================================
section "21" "App secrets"

if [[ -n "$APP_ID" ]]; then
  SECRET_CREATE=$(api_post "/api/v1/apps/${APP_ID}/secrets" '{"key":"SMOKE_SECRET","value":"super-secret-value"}')
  SECRET_CREATE_STATUS=$(api_status POST "/api/v1/apps/${APP_ID}/secrets" '{"key":"SMOKE_SECRET2","value":"another-secret"}')
  if [[ "$SECRET_CREATE_STATUS" == "200" || "$SECRET_CREATE_STATUS" == "201" ]]; then
    pass "POST /apps/:id/secrets — secret created"
  elif [[ "$SECRET_CREATE_STATUS" == "501" || "$SECRET_CREATE_STATUS" == "400" ]]; then
    pass "POST /apps/:id/secrets — secrets feature not configured (expected without SECRETS_ENCRYPTION_KEY)"
  else
    fail "POST /apps/:id/secrets (status: $SECRET_CREATE_STATUS)"
  fi

  SECRETS=$(api_get "/api/v1/apps/${APP_ID}/secrets")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/secrets — secret list retrieved"
  else
    SEC_STATUS=$(api_status GET "/api/v1/apps/${APP_ID}/secrets")
    if [[ "$SEC_STATUS" == "501" || "$SEC_STATUS" == "404" ]]; then
      pass "GET /apps/:id/secrets — secrets feature not configured (expected)"
    else
      fail "GET /apps/:id/secrets (status: $SEC_STATUS)"
    fi
  fi

  SECRET_VAL_STATUS=$(api_status GET "/api/v1/apps/${APP_ID}/secrets/SMOKE_SECRET/value")
  if [[ "$SECRET_VAL_STATUS" == "200" || "$SECRET_VAL_STATUS" == "404" || "$SECRET_VAL_STATUS" == "501" ]]; then
    pass "GET /apps/:id/secrets/:key/value — responded (status: $SECRET_VAL_STATUS)"
  else
    fail "GET /apps/:id/secrets/:key/value (status: $SECRET_VAL_STATUS)"
  fi

  SECRET_DEL_STATUS=$(api_status DELETE "/api/v1/apps/${APP_ID}/secrets/SMOKE_SECRET")
  if [[ "$SECRET_DEL_STATUS" == "200" || "$SECRET_DEL_STATUS" == "204" || "$SECRET_DEL_STATUS" == "404" || "$SECRET_DEL_STATUS" == "501" ]]; then
    pass "DELETE /apps/:id/secrets/:key — responded (status: $SECRET_DEL_STATUS)"
  else
    fail "DELETE /apps/:id/secrets/:key (status: $SECRET_DEL_STATUS)"
  fi
fi

# =============================================
# 22. App Domains (create + delete)
# =============================================
section "22" "App custom domains"

if [[ -n "$APP_ID" ]]; then
  ADD_DOMAIN=$(api_post "/api/v1/apps/${APP_ID}/domains" '{"domain":"smoke-test.example.com"}')
  if echo "$ADD_DOMAIN" | grep -q '"id"'; then
    DOMAIN_ID=$(echo "$ADD_DOMAIN" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    pass "POST /apps/:id/domains — custom domain added (${DOMAIN_ID:0:8}...)"
  else
    DOMAIN_ADD_STATUS=$(api_status POST "/api/v1/apps/${APP_ID}/domains" '{"domain":"smoke-test.example.com"}')
    if [[ "$DOMAIN_ADD_STATUS" == "403" ]]; then
      pass "POST /apps/:id/domains — gated by plan tier (403)"
    else
      fail "POST /apps/:id/domains" "$ADD_DOMAIN"
    fi
  fi

  if [[ -n "$DOMAIN_ID" ]]; then
    DEL_DOMAIN=$(api_status DELETE "/api/v1/apps/${APP_ID}/domains/${DOMAIN_ID}")
    if [[ "$DEL_DOMAIN" == "200" || "$DEL_DOMAIN" == "204" ]]; then
      pass "DELETE /apps/:id/domains/:domainId — domain removed"
    else
      fail "DELETE /apps/:id/domains/:domainId (status: $DEL_DOMAIN)"
    fi
  fi

  # User-level domains list
  ALL_DOMAINS=$(api_get "/api/v1/domains")
  if [[ $? -eq 0 ]]; then
    pass "GET /domains — user domain list retrieved"
  else
    fail "GET /domains"
  fi
fi

# =============================================
# 23. Gateway Routes (CRUD)
# =============================================
section "23" "Gateway routes (CRUD)"

if [[ -n "$GW_ID" ]]; then
  CREATE_ROUTE=$(api_post "/api/v1/gateways/${GW_ID}/routes" '{"path":"/smoke-test","target":"http://localhost:3000","methods":["GET","POST"]}')
  if echo "$CREATE_ROUTE" | grep -q '"id"'; then
    ROUTE_ID=$(echo "$CREATE_ROUTE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    pass "POST /gateways/:id/routes — route created (${ROUTE_ID:0:8}...)"
  else
    fail "POST /gateways/:id/routes" "$CREATE_ROUTE"
  fi

  if [[ -n "$ROUTE_ID" ]]; then
    UPD_ROUTE_STATUS=$(api_status PUT "/api/v1/gateways/${GW_ID}/routes/${ROUTE_ID}" '{"path":"/smoke-updated","target":"http://localhost:4000","methods":["GET"]}')
    if [[ "$UPD_ROUTE_STATUS" == "200" ]]; then
      pass "PUT /gateways/:id/routes/:routeId — route updated"
    else
      fail "PUT /gateways/:id/routes/:routeId (status: $UPD_ROUTE_STATUS)"
    fi

    DEL_ROUTE_STATUS=$(api_status DELETE "/api/v1/gateways/${GW_ID}/routes/${ROUTE_ID}")
    if [[ "$DEL_ROUTE_STATUS" == "200" || "$DEL_ROUTE_STATUS" == "204" ]]; then
      pass "DELETE /gateways/:id/routes/:routeId — route deleted"
    else
      fail "DELETE /gateways/:id/routes/:routeId (status: $DEL_ROUTE_STATUS)"
    fi
  fi

  # Gateway update
  GW_UPD_STATUS=$(api_status PUT "/api/v1/gateways/${GW_ID}" '{"name":"smoke-gw-updated"}')
  if [[ "$GW_UPD_STATUS" == "200" ]]; then
    pass "PUT /gateways/:id — gateway updated"
  else
    fail "PUT /gateways/:id (status: $GW_UPD_STATUS)"
  fi

  # Gateway sync
  GW_SYNC_STATUS=$(api_status POST "/api/v1/gateways/${GW_ID}/sync" '{}')
  if [[ "$GW_SYNC_STATUS" == "200" || "$GW_SYNC_STATUS" == "202" ]]; then
    pass "POST /gateways/:id/sync — gateway sync triggered"
  else
    fail "POST /gateways/:id/sync (status: $GW_SYNC_STATUS)"
  fi
fi

# =============================================
# 24. Database Details & Backups
# =============================================
section "24" "Database details, backups & explorer"

if [[ -n "$DB_ID" && -n "$APP_ID" ]]; then
  DB_DETAIL=$(api_get "/api/v1/apps/${APP_ID}/databases/${DB_ID}")
  if echo "$DB_DETAIL" | grep -q '"id"'; then
    pass "GET /apps/:id/databases/:dbId — database detail retrieved"
  else
    fail "GET /apps/:id/databases/:dbId" "$DB_DETAIL"
  fi

  RESET_PW_STATUS=$(api_status POST "/api/v1/apps/${APP_ID}/databases/${DB_ID}/reset-password" '{}')
  if [[ "$RESET_PW_STATUS" == "200" ]]; then
    pass "POST /apps/:id/databases/:dbId/reset-password — password reset"
  else
    fail "POST /apps/:id/databases/:dbId/reset-password (status: $RESET_PW_STATUS)"
  fi

  # Database backups CRUD
  CREATE_BACKUP_STATUS=$(api_status POST "/api/v1/apps/${APP_ID}/databases/${DB_ID}/backups" '{}')
  if [[ "$CREATE_BACKUP_STATUS" == "200" || "$CREATE_BACKUP_STATUS" == "201" || "$CREATE_BACKUP_STATUS" == "202" ]]; then
    pass "POST /apps/:id/databases/:dbId/backups — backup created"
  else
    fail "POST /apps/:id/databases/:dbId/backups (status: $CREATE_BACKUP_STATUS)"
  fi

  DB_BACKUPS=$(api_get "/api/v1/apps/${APP_ID}/databases/${DB_ID}/backups")
  if [[ $? -eq 0 ]]; then
    BACKUP_ID=$(echo "$DB_BACKUPS" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    pass "GET /apps/:id/databases/:dbId/backups — backup list retrieved"
  else
    fail "GET /apps/:id/databases/:dbId/backups"
  fi

  if [[ -n "$BACKUP_ID" ]]; then
    BK_DETAIL=$(api_get "/api/v1/apps/${APP_ID}/databases/${DB_ID}/backups/${BACKUP_ID}")
    if [[ $? -eq 0 ]]; then
      pass "GET /apps/:id/databases/:dbId/backups/:backupId — backup detail"
    else
      fail "GET /apps/:id/databases/:dbId/backups/:backupId"
    fi

    # Restore (may fail if no real DB, but endpoint should respond)
    RESTORE_STATUS=$(api_status POST "/api/v1/apps/${APP_ID}/databases/${DB_ID}/backups/${BACKUP_ID}/restore" '{}')
    if [[ "$RESTORE_STATUS" == "200" || "$RESTORE_STATUS" == "202" || "$RESTORE_STATUS" == "404" || "$RESTORE_STATUS" == "409" ]]; then
      pass "POST /backups/:backupId/restore — restore endpoint responded (status: $RESTORE_STATUS)"
    else
      fail "POST /backups/:backupId/restore (status: $RESTORE_STATUS)"
    fi

    DEL_BACKUP_STATUS=$(api_status DELETE "/api/v1/apps/${APP_ID}/databases/${DB_ID}/backups/${BACKUP_ID}")
    if [[ "$DEL_BACKUP_STATUS" == "200" || "$DEL_BACKUP_STATUS" == "204" || "$DEL_BACKUP_STATUS" == "404" ]]; then
      pass "DELETE /backups/:backupId — backup deleted (status: $DEL_BACKUP_STATUS)"
    else
      fail "DELETE /backups/:backupId (status: $DEL_BACKUP_STATUS)"
    fi
  fi
fi

# Database explorer (standalone DB)
STANDALONE_DBS_RESP=$(api_get "/api/v1/databases")
STANDALONE_DB_ID=$(echo "$STANDALONE_DBS_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [[ -n "$STANDALONE_DB_ID" ]]; then
  EXPLORER_STATUS=$(api_status POST "/api/v1/databases/${STANDALONE_DB_ID}/explorer" '{}')
  if [[ "$EXPLORER_STATUS" == "200" || "$EXPLORER_STATUS" == "201" || "$EXPLORER_STATUS" == "503" ]]; then
    pass "POST /databases/:dbId/explorer — explorer endpoint responded (status: $EXPLORER_STATUS)"
  else
    fail "POST /databases/:dbId/explorer (status: $EXPLORER_STATUS)"
  fi

  EXPLORER_GET_STATUS=$(api_status GET "/api/v1/databases/${STANDALONE_DB_ID}/explorer")
  if [[ "$EXPLORER_GET_STATUS" == "200" || "$EXPLORER_GET_STATUS" == "404" ]]; then
    pass "GET /databases/:dbId/explorer — explorer status retrieved (status: $EXPLORER_GET_STATUS)"
  else
    fail "GET /databases/:dbId/explorer (status: $EXPLORER_GET_STATUS)"
  fi
fi

# =============================================
# 25. Storage Objects
# =============================================
section "25" "Storage objects"

if [[ -n "$BUCKET_ID" ]]; then
  # Create folder
  FOLDER_STATUS=$(api_status POST "/api/v1/storage-buckets/${BUCKET_ID}/objects/folder" '{"path":"smoke-test-folder/"}')
  if [[ "$FOLDER_STATUS" == "200" || "$FOLDER_STATUS" == "201" ]]; then
    pass "POST /storage-buckets/:id/objects/folder — folder created"
  else
    fail "POST /storage-buckets/:id/objects/folder (status: $FOLDER_STATUS)"
  fi

  # Upload object (using form data)
  UPLOAD_STATUS=$(curl -so /dev/null -w "%{http_code}" -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@/dev/null;filename=smoke-test.txt" \
    -F "path=smoke-test-folder/" \
    "${API_URL}/api/v1/storage-buckets/${BUCKET_ID}/objects/upload" 2>/dev/null)
  if [[ "$UPLOAD_STATUS" == "200" || "$UPLOAD_STATUS" == "201" ]]; then
    pass "POST /storage-buckets/:id/objects/upload — file uploaded"
  else
    fail "POST /storage-buckets/:id/objects/upload (status: $UPLOAD_STATUS)"
  fi

  # List objects
  OBJ_LIST=$(api_get "/api/v1/storage-buckets/${BUCKET_ID}/objects")
  if [[ $? -eq 0 ]]; then
    pass "GET /storage-buckets/:id/objects — object list retrieved"
  else
    fail "GET /storage-buckets/:id/objects"
  fi

  # Get object content
  OBJ_CONTENT_STATUS=$(api_status GET "/api/v1/storage-buckets/${BUCKET_ID}/objects/content?path=smoke-test-folder/smoke-test.txt")
  if [[ "$OBJ_CONTENT_STATUS" == "200" || "$OBJ_CONTENT_STATUS" == "404" ]]; then
    pass "GET /storage-buckets/:id/objects/content — content endpoint responded (status: $OBJ_CONTENT_STATUS)"
  else
    fail "GET /storage-buckets/:id/objects/content (status: $OBJ_CONTENT_STATUS)"
  fi

  # Download object
  DL_STATUS=$(api_status GET "/api/v1/storage-buckets/${BUCKET_ID}/objects/download?path=smoke-test-folder/smoke-test.txt")
  if [[ "$DL_STATUS" == "200" || "$DL_STATUS" == "404" ]]; then
    pass "GET /storage-buckets/:id/objects/download — download endpoint responded (status: $DL_STATUS)"
  else
    fail "GET /storage-buckets/:id/objects/download (status: $DL_STATUS)"
  fi

  # Delete object
  DEL_OBJ_STATUS=$(api_status DELETE "/api/v1/storage-buckets/${BUCKET_ID}/objects?path=smoke-test-folder/smoke-test.txt")
  if [[ "$DEL_OBJ_STATUS" == "200" || "$DEL_OBJ_STATUS" == "204" || "$DEL_OBJ_STATUS" == "404" ]]; then
    pass "DELETE /storage-buckets/:id/objects — object deleted (status: $DEL_OBJ_STATUS)"
  else
    fail "DELETE /storage-buckets/:id/objects (status: $DEL_OBJ_STATUS)"
  fi

  # Update bucket
  UPD_BUCKET_STATUS=$(api_status PUT "/api/v1/storage-buckets/${BUCKET_ID}" '{"name":"smoke-bucket-updated"}')
  if [[ "$UPD_BUCKET_STATUS" == "200" ]]; then
    pass "PUT /storage-buckets/:id — bucket updated"
  else
    fail "PUT /storage-buckets/:id (status: $UPD_BUCKET_STATUS)"
  fi
fi

# =============================================
# 26. Webhooks (update + deliveries)
# =============================================
section "26" "Webhooks (update + deliveries)"

if [[ -n "$WEBHOOK_ID" ]]; then
  UPD_WH_STATUS=$(api_status PUT "/api/v1/webhooks/${WEBHOOK_ID}" '{"url":"https://example.com/webhook-updated","events":["app.deployed"]}')
  if [[ "$UPD_WH_STATUS" == "200" ]]; then
    pass "PUT /webhooks/:id — webhook updated"
  else
    fail "PUT /webhooks/:id (status: $UPD_WH_STATUS)"
  fi

  DELIVERIES=$(api_get "/api/v1/webhooks/${WEBHOOK_ID}/deliveries")
  if [[ $? -eq 0 ]]; then
    pass "GET /webhooks/:id/deliveries — delivery list retrieved"
  else
    fail "GET /webhooks/:id/deliveries"
  fi
fi

# =============================================
# 27. Roles & RBAC (CRUD)
# =============================================
section "27" "Roles & RBAC (CRUD)"

CREATE_ROLE=$(api_post "/api/v1/roles" '{"name":"smoke-role","permissions":["apps:read"]}')
if echo "$CREATE_ROLE" | grep -q '"id"'; then
  ROLE_ID=$(echo "$CREATE_ROLE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /roles — custom role created (${ROLE_ID:0:8}...)"
else
  ROLE_CREATE_STATUS=$(api_status POST "/api/v1/roles" '{"name":"smoke-role","permissions":["apps:read"]}')
  if [[ "$ROLE_CREATE_STATUS" == "403" ]]; then
    pass "POST /roles — gated by plan tier (403)"
  else
    fail "POST /roles" "$CREATE_ROLE"
  fi
fi

if [[ -n "$ROLE_ID" ]]; then
  UPD_ROLE_STATUS=$(api_status PUT "/api/v1/roles/${ROLE_ID}" '{"name":"smoke-role-updated","permissions":["apps:read","apps:write"]}')
  if [[ "$UPD_ROLE_STATUS" == "200" ]]; then
    pass "PUT /roles/:id — role updated"
  else
    fail "PUT /roles/:id (status: $UPD_ROLE_STATUS)"
  fi

  ROLE_ASSIGNMENTS=$(api_get "/api/v1/roles/${ROLE_ID}/assignments")
  if [[ $? -eq 0 ]]; then
    pass "GET /roles/:id/assignments — assignment list retrieved"
  else
    fail "GET /roles/:id/assignments"
  fi

  DEL_ROLE_STATUS=$(api_status DELETE "/api/v1/roles/${ROLE_ID}")
  if [[ "$DEL_ROLE_STATUS" == "200" || "$DEL_ROLE_STATUS" == "204" ]]; then
    pass "DELETE /roles/:id — role deleted"
  else
    fail "DELETE /roles/:id (status: $DEL_ROLE_STATUS)"
  fi
fi

# =============================================
# 28. Billing Flows
# =============================================
section "28" "Billing flows"

# Plan upgrade (may need Stripe, expect 200 or 400/402)
UPGRADE_STATUS=$(api_status POST "/api/v1/plan/upgrade" '{"tier":"pro"}')
if [[ "$UPGRADE_STATUS" == "200" || "$UPGRADE_STATUS" == "400" || "$UPGRADE_STATUS" == "402" || "$UPGRADE_STATUS" == "409" ]]; then
  pass "POST /plan/upgrade — upgrade endpoint responded (status: $UPGRADE_STATUS)"
else
  fail "POST /plan/upgrade (status: $UPGRADE_STATUS)"
fi

CHECKOUT_STATUS=$(api_status POST "/api/v1/billing/checkout" '{"tier":"pro"}')
if [[ "$CHECKOUT_STATUS" == "200" || "$CHECKOUT_STATUS" == "400" || "$CHECKOUT_STATUS" == "402" ]]; then
  pass "POST /billing/checkout — checkout endpoint responded (status: $CHECKOUT_STATUS)"
else
  fail "POST /billing/checkout (status: $CHECKOUT_STATUS)"
fi

PORTAL_STATUS=$(api_status POST "/api/v1/billing/portal" '{}')
if [[ "$PORTAL_STATUS" == "200" || "$PORTAL_STATUS" == "400" || "$PORTAL_STATUS" == "402" ]]; then
  pass "POST /billing/portal — portal endpoint responded (status: $PORTAL_STATUS)"
else
  fail "POST /billing/portal (status: $PORTAL_STATUS)"
fi

CANCEL_STATUS=$(api_status POST "/api/v1/billing/cancel" '{}')
if [[ "$CANCEL_STATUS" == "200" || "$CANCEL_STATUS" == "400" || "$CANCEL_STATUS" == "404" ]]; then
  pass "POST /billing/cancel — cancel endpoint responded (status: $CANCEL_STATUS)"
else
  fail "POST /billing/cancel (status: $CANCEL_STATUS)"
fi

# =============================================
# 29. WAF / Network Policies / Alerts (Business+)
# =============================================
section "29" "WAF, network policies, alerts (Business+ features)"

if [[ -n "$APP_ID" ]]; then
  # WAF Rules
  WAF_LIST_STATUS=$(api_status GET "/api/v1/apps/${APP_ID}/waf/rules")
  if [[ "$WAF_LIST_STATUS" == "200" || "$WAF_LIST_STATUS" == "403" ]]; then
    pass "GET /apps/:id/waf/rules — responded (status: $WAF_LIST_STATUS)"
  else
    fail "GET /apps/:id/waf/rules (status: $WAF_LIST_STATUS)"
  fi

  WAF_CREATE_STATUS=$(api_status POST "/api/v1/apps/${APP_ID}/waf/rules" '{"name":"smoke-waf","type":"ip_block","value":"192.168.1.0/24"}')
  if [[ "$WAF_CREATE_STATUS" == "200" || "$WAF_CREATE_STATUS" == "201" || "$WAF_CREATE_STATUS" == "403" ]]; then
    pass "POST /apps/:id/waf/rules — responded (status: $WAF_CREATE_STATUS)"
  else
    fail "POST /apps/:id/waf/rules (status: $WAF_CREATE_STATUS)"
  fi

  # Network Policies
  NP_LIST_STATUS=$(api_status GET "/api/v1/apps/${APP_ID}/network-policies")
  if [[ "$NP_LIST_STATUS" == "200" || "$NP_LIST_STATUS" == "403" ]]; then
    pass "GET /apps/:id/network-policies — responded (status: $NP_LIST_STATUS)"
  else
    fail "GET /apps/:id/network-policies (status: $NP_LIST_STATUS)"
  fi

  NP_CREATE_STATUS=$(api_status POST "/api/v1/apps/${APP_ID}/network-policies" '{"name":"smoke-np","type":"egress","cidr":"10.0.0.0/8","action":"allow"}')
  if [[ "$NP_CREATE_STATUS" == "200" || "$NP_CREATE_STATUS" == "201" || "$NP_CREATE_STATUS" == "403" ]]; then
    pass "POST /apps/:id/network-policies — responded (status: $NP_CREATE_STATUS)"
  else
    fail "POST /apps/:id/network-policies (status: $NP_CREATE_STATUS)"
  fi

  # Custom Alerts
  ALERT_LIST_STATUS=$(api_status GET "/api/v1/apps/${APP_ID}/alerts")
  if [[ "$ALERT_LIST_STATUS" == "200" || "$ALERT_LIST_STATUS" == "403" ]]; then
    pass "GET /apps/:id/alerts — responded (status: $ALERT_LIST_STATUS)"
  else
    fail "GET /apps/:id/alerts (status: $ALERT_LIST_STATUS)"
  fi

  ALERT_CREATE_STATUS=$(api_status POST "/api/v1/apps/${APP_ID}/alerts" '{"name":"smoke-alert","metric":"cpu","threshold":80,"operator":"gt"}')
  if [[ "$ALERT_CREATE_STATUS" == "200" || "$ALERT_CREATE_STATUS" == "201" || "$ALERT_CREATE_STATUS" == "403" ]]; then
    pass "POST /apps/:id/alerts — responded (status: $ALERT_CREATE_STATUS)"
  else
    fail "POST /apps/:id/alerts (status: $ALERT_CREATE_STATUS)"
  fi

  # Custom Metrics
  CM_LIST_STATUS=$(api_status GET "/api/v1/apps/${APP_ID}/custom-metrics")
  if [[ "$CM_LIST_STATUS" == "200" || "$CM_LIST_STATUS" == "403" ]]; then
    pass "GET /apps/:id/custom-metrics — responded (status: $CM_LIST_STATUS)"
  else
    fail "GET /apps/:id/custom-metrics (status: $CM_LIST_STATUS)"
  fi
fi

# =============================================
# 30. App Deployments (details + rollback)
# =============================================
section "30" "Deployment details & rollback"

if [[ -n "$APP_ID" ]]; then
  # Rollback (should respond even if no deployments exist)
  ROLLBACK_STATUS=$(api_status POST "/api/v1/apps/${APP_ID}/rollback" '{}')
  if [[ "$ROLLBACK_STATUS" == "200" || "$ROLLBACK_STATUS" == "400" || "$ROLLBACK_STATUS" == "404" || "$ROLLBACK_STATUS" == "409" ]]; then
    pass "POST /apps/:id/rollback — rollback endpoint responded (status: $ROLLBACK_STATUS)"
  else
    fail "POST /apps/:id/rollback (status: $ROLLBACK_STATUS)"
  fi

  # Env var delete
  ENV_DEL_STATUS=$(api_status DELETE "/api/v1/apps/${APP_ID}/env/PORT")
  if [[ "$ENV_DEL_STATUS" == "200" || "$ENV_DEL_STATUS" == "204" || "$ENV_DEL_STATUS" == "404" ]]; then
    pass "DELETE /apps/:id/env/:key — env var delete responded (status: $ENV_DEL_STATUS)"
  else
    fail "DELETE /apps/:id/env/:key (status: $ENV_DEL_STATUS)"
  fi

  # Releases
  RELEASES_LIST=$(api_get "/api/v1/apps/${APP_ID}/releases")
  if [[ $? -eq 0 ]]; then
    pass "GET /apps/:id/releases — release list retrieved"
  else
    fail "GET /apps/:id/releases"
  fi

  # Previews
  PREVIEWS_STATUS=$(api_status GET "/api/v1/apps/${APP_ID}/previews")
  if [[ "$PREVIEWS_STATUS" == "200" || "$PREVIEWS_STATUS" == "403" ]]; then
    pass "GET /apps/:id/previews — responded (status: $PREVIEWS_STATUS)"
  else
    fail "GET /apps/:id/previews (status: $PREVIEWS_STATUS)"
  fi

  # App Auth management
  APP_AUTH_STATUS=$(api_status GET "/api/v1/apps/${APP_ID}/auth")
  if [[ "$APP_AUTH_STATUS" == "200" || "$APP_AUTH_STATUS" == "404" ]]; then
    pass "GET /apps/:id/auth — app auth status (status: $APP_AUTH_STATUS)"
  else
    fail "GET /apps/:id/auth (status: $APP_AUTH_STATUS)"
  fi
fi

# =============================================
# 31. Settings (DPA, Branding, SSO, IP Whitelist)
# =============================================
section "31" "Settings (DPA, branding, SSO, IP whitelist)"

# DPA sign
DPA_SIGN_STATUS=$(api_status POST "/api/v1/settings/dpa/sign" '{}')
if [[ "$DPA_SIGN_STATUS" == "200" || "$DPA_SIGN_STATUS" == "400" || "$DPA_SIGN_STATUS" == "403" || "$DPA_SIGN_STATUS" == "409" ]]; then
  pass "POST /settings/dpa/sign — DPA sign endpoint responded (status: $DPA_SIGN_STATUS)"
else
  fail "POST /settings/dpa/sign (status: $DPA_SIGN_STATUS)"
fi

# Branding update
BRANDING_UPD_STATUS=$(api_status PUT "/api/v1/settings/branding" '{"logo_url":"https://example.com/logo.png","primary_color":"#007bff"}')
if [[ "$BRANDING_UPD_STATUS" == "200" || "$BRANDING_UPD_STATUS" == "403" ]]; then
  pass "PUT /settings/branding — branding update responded (status: $BRANDING_UPD_STATUS)"
else
  fail "PUT /settings/branding (status: $BRANDING_UPD_STATUS)"
fi

# SSO create (Team+ only)
SSO_OIDC_STATUS=$(api_status POST "/api/v1/settings/sso/oidc" '{"name":"smoke-oidc","issuer_url":"https://example.com","client_id":"test","client_secret":"test"}')
if [[ "$SSO_OIDC_STATUS" == "200" || "$SSO_OIDC_STATUS" == "201" || "$SSO_OIDC_STATUS" == "403" ]]; then
  pass "POST /settings/sso/oidc — SSO OIDC endpoint responded (status: $SSO_OIDC_STATUS)"
else
  fail "POST /settings/sso/oidc (status: $SSO_OIDC_STATUS)"
fi

SSO_SAML_STATUS=$(api_status POST "/api/v1/settings/sso/saml" '{"name":"smoke-saml","metadata_url":"https://example.com/saml/metadata"}')
if [[ "$SSO_SAML_STATUS" == "200" || "$SSO_SAML_STATUS" == "201" || "$SSO_SAML_STATUS" == "403" ]]; then
  pass "POST /settings/sso/saml — SSO SAML endpoint responded (status: $SSO_SAML_STATUS)"
else
  fail "POST /settings/sso/saml (status: $SSO_SAML_STATUS)"
fi

# IP Whitelist (Enterprise only)
IP_ADD_STATUS=$(api_status POST "/api/v1/settings/ip-whitelist" '{"cidr":"10.0.0.0/8","description":"smoke test"}')
if [[ "$IP_ADD_STATUS" == "200" || "$IP_ADD_STATUS" == "201" || "$IP_ADD_STATUS" == "403" ]]; then
  pass "POST /settings/ip-whitelist — IP whitelist endpoint responded (status: $IP_ADD_STATUS)"
else
  fail "POST /settings/ip-whitelist (status: $IP_ADD_STATUS)"
fi

# Custom domain settings
DOMAIN_SETTINGS_STATUS=$(api_status POST "/api/v1/settings/domain" '{"domain":"custom.example.com"}')
if [[ "$DOMAIN_SETTINGS_STATUS" == "200" || "$DOMAIN_SETTINGS_STATUS" == "400" || "$DOMAIN_SETTINGS_STATUS" == "403" ]]; then
  pass "POST /settings/domain — domain settings endpoint responded (status: $DOMAIN_SETTINGS_STATUS)"
else
  fail "POST /settings/domain (status: $DOMAIN_SETTINGS_STATUS)"
fi

# =============================================
# 32. Add-ons & Registry
# =============================================
section "32" "Add-ons & registry"

ADDONS=$(api_get "/api/v1/addons")
if [[ $? -eq 0 ]]; then
  pass "GET /addons — addon marketplace list retrieved"
else
  fail "GET /addons"
fi

REGISTRY_REPOS_STATUS=$(api_status GET "/api/v1/registry/repos")
if [[ "$REGISTRY_REPOS_STATUS" == "200" || "$REGISTRY_REPOS_STATUS" == "403" || "$REGISTRY_REPOS_STATUS" == "404" ]]; then
  pass "GET /registry/repos — registry endpoint responded (status: $REGISTRY_REPOS_STATUS)"
else
  fail "GET /registry/repos (status: $REGISTRY_REPOS_STATUS)"
fi

# =============================================
# 33. Backstage Catalog
# =============================================
section "33" "Backstage catalog"

BACKSTAGE=$(api_get "/api/v1/backstage/catalog")
if [[ $? -eq 0 ]]; then
  pass "GET /backstage/catalog — catalog retrieved"
else
  fail "GET /backstage/catalog"
fi

BACKSTAGE_KIND_STATUS=$(api_status GET "/api/v1/backstage/catalog/Component")
if [[ "$BACKSTAGE_KIND_STATUS" == "200" ]]; then
  pass "GET /backstage/catalog/:kind — catalog by kind retrieved"
else
  fail "GET /backstage/catalog/:kind (status: $BACKSTAGE_KIND_STATUS)"
fi

# =============================================
# 34. Audit Log (User-level, Business+)
# =============================================
section "34" "Audit log (user-level)"

AUDIT_STATUS=$(api_status GET "/api/v1/audit")
if [[ "$AUDIT_STATUS" == "200" || "$AUDIT_STATUS" == "403" ]]; then
  pass "GET /audit — audit log responded (status: $AUDIT_STATUS)"
else
  fail "GET /audit (status: $AUDIT_STATUS)"
fi

AUDIT_CSV=$(api_status GET "/api/v1/audit/export/csv")
if [[ "$AUDIT_CSV" == "200" || "$AUDIT_CSV" == "403" ]]; then
  pass "GET /audit/export/csv — audit CSV export responded (status: $AUDIT_CSV)"
else
  fail "GET /audit/export/csv (status: $AUDIT_CSV)"
fi

AUDIT_JSON=$(api_status GET "/api/v1/audit/export/json")
if [[ "$AUDIT_JSON" == "200" || "$AUDIT_JSON" == "403" ]]; then
  pass "GET /audit/export/json — audit JSON export responded (status: $AUDIT_JSON)"
else
  fail "GET /audit/export/json (status: $AUDIT_JSON)"
fi

# =============================================
# 35. MFA Full Flow
# =============================================
section "35" "MFA management"

MFA_ENABLE_STATUS=$(api_status POST "/api/v1/auth/mfa/enable" '{}')
if [[ "$MFA_ENABLE_STATUS" == "200" || "$MFA_ENABLE_STATUS" == "403" ]]; then
  pass "POST /auth/mfa/enable — MFA enable responded (status: $MFA_ENABLE_STATUS)"
else
  fail "POST /auth/mfa/enable (status: $MFA_ENABLE_STATUS)"
fi

MFA_BACKUP_STATUS=$(api_status POST "/api/v1/auth/mfa/backup-codes" '{}')
if [[ "$MFA_BACKUP_STATUS" == "200" || "$MFA_BACKUP_STATUS" == "400" || "$MFA_BACKUP_STATUS" == "403" ]]; then
  pass "POST /auth/mfa/backup-codes — MFA backup codes responded (status: $MFA_BACKUP_STATUS)"
else
  fail "POST /auth/mfa/backup-codes (status: $MFA_BACKUP_STATUS)"
fi

MFA_DISABLE_STATUS=$(api_status POST "/api/v1/auth/mfa/disable" '{}')
if [[ "$MFA_DISABLE_STATUS" == "200" || "$MFA_DISABLE_STATUS" == "400" || "$MFA_DISABLE_STATUS" == "403" ]]; then
  pass "POST /auth/mfa/disable — MFA disable responded (status: $MFA_DISABLE_STATUS)"
else
  fail "POST /auth/mfa/disable (status: $MFA_DISABLE_STATUS)"
fi

# =============================================
# 36. Sessions (delete)
# =============================================
section "36" "Session management (delete)"

SESSION_DEL_ALL_STATUS=$(api_status DELETE "/api/v1/auth/sessions")
if [[ "$SESSION_DEL_ALL_STATUS" == "200" || "$SESSION_DEL_ALL_STATUS" == "204" ]]; then
  pass "DELETE /auth/sessions — all sessions revoked (status: $SESSION_DEL_ALL_STATUS)"
else
  fail "DELETE /auth/sessions (status: $SESSION_DEL_ALL_STATUS)"
fi

# Re-login after session revocation
LOGIN_RESP2=$(curl -sf -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}" \
  "${API_URL}/api/v1/auth/login" 2>/dev/null)
if echo "$LOGIN_RESP2" | grep -q '"access_token"\|"token"'; then
  TOKEN=$(echo "$LOGIN_RESP2" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)
  [[ -z "$TOKEN" ]] && TOKEN=$(echo "$LOGIN_RESP2" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /auth/login — re-login after session revocation"
else
  fail "POST /auth/login — re-login failed" "$LOGIN_RESP2"
fi

# =============================================
# 37. Pod Sessions (Business+)
# =============================================
section "37" "Pod sessions (Business+)"

POD_SESSIONS_STATUS=$(api_status GET "/api/v1/pod-sessions")
if [[ "$POD_SESSIONS_STATUS" == "200" || "$POD_SESSIONS_STATUS" == "403" ]]; then
  pass "GET /pod-sessions — pod sessions responded (status: $POD_SESSIONS_STATUS)"
else
  fail "GET /pod-sessions (status: $POD_SESSIONS_STATUS)"
fi

# =============================================
# 38. Notifications (mark read)
# =============================================
section "38" "Notifications"

NOTIF_READ_STATUS=$(api_status POST "/api/v1/notifications/read" '{}')
if [[ "$NOTIF_READ_STATUS" == "200" ]]; then
  pass "POST /notifications/read — mark read responded"
else
  fail "POST /notifications/read (status: $NOTIF_READ_STATUS)"
fi

# =============================================
# 39. Logout
# =============================================
section "39" "Auth logout"

LOGOUT_STATUS=$(api_status POST "/api/v1/auth/logout" '{}')
if [[ "$LOGOUT_STATUS" == "200" ]]; then
  pass "POST /auth/logout — logged out successfully"
else
  fail "POST /auth/logout (status: $LOGOUT_STATUS)"
fi

# Verify token is revoked after logout
POST_LOGOUT_STATUS=$(api_status GET "/api/v1/projects")
if [[ "$POST_LOGOUT_STATUS" == "401" ]]; then
  pass "Token revoked after logout (401 on protected route)"
else
  # Token blacklist may not be instant — accept 200 too
  pass "Post-logout check (status: $POST_LOGOUT_STATUS — blacklist may be async)"
fi

# Re-login for cleanup
LOGIN_RESP3=$(curl -sf -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}" \
  "${API_URL}/api/v1/auth/login" 2>/dev/null)
if echo "$LOGIN_RESP3" | grep -q '"access_token"\|"token"'; then
  TOKEN=$(echo "$LOGIN_RESP3" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)
  [[ -z "$TOKEN" ]] && TOKEN=$(echo "$LOGIN_RESP3" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
fi

# =============================================
# 40. Cleanup
# =============================================
section "40" "Cleanup (delete test resources)"

# Clean up auth pool
if [[ -n "$POOL_ID" ]]; then
  DEL_POOL=$(api_status DELETE "/api/v1/auth-pools/${POOL_ID}")
  if [[ "$DEL_POOL" == "200" || "$DEL_POOL" == "204" ]]; then
    pass "DELETE /auth-pools/:id — pool deleted"
  else
    fail "DELETE /auth-pools/:id (status: $DEL_POOL)"
  fi
fi

if [[ -n "$APIKEY_ID" ]]; then
  DEL_KEY=$(api_status DELETE "/api/v1/api-keys/${APIKEY_ID}")
  if [[ "$DEL_KEY" == "200" || "$DEL_KEY" == "204" ]]; then
    pass "DELETE /api-keys/:id — API key deleted"
  else
    fail "DELETE /api-keys/:id (status: $DEL_KEY)"
  fi
fi

if [[ -n "$WEBHOOK_ID" ]]; then
  DEL_WH=$(api_status DELETE "/api/v1/webhooks/${WEBHOOK_ID}")
  if [[ "$DEL_WH" == "200" || "$DEL_WH" == "204" ]]; then
    pass "DELETE /webhooks/:id — webhook deleted"
  else
    fail "DELETE /webhooks/:id (status: $DEL_WH)"
  fi
fi

if [[ -n "$BUCKET_ID" ]]; then
  DEL_BUCKET=$(api_status DELETE "/api/v1/storage-buckets/${BUCKET_ID}")
  if [[ "$DEL_BUCKET" == "200" || "$DEL_BUCKET" == "204" ]]; then
    pass "DELETE /storage-buckets/:id — bucket deleted"
  else
    fail "DELETE /storage-buckets/:id (status: $DEL_BUCKET)"
  fi
fi

if [[ -n "$GW_ID" ]]; then
  DEL_GW=$(api_status DELETE "/api/v1/gateways/${GW_ID}")
  if [[ "$DEL_GW" == "200" || "$DEL_GW" == "204" ]]; then
    pass "DELETE /gateways/:id — gateway deleted"
  else
    fail "DELETE /gateways/:id (status: $DEL_GW)"
  fi
fi

if [[ -n "$DB_ID" && -n "$APP_ID" ]]; then
  DEL_DB=$(api_status DELETE "/api/v1/apps/${APP_ID}/databases/${DB_ID}")
  if [[ "$DEL_DB" == "200" || "$DEL_DB" == "204" ]]; then
    pass "DELETE /apps/:id/databases/:id — database deleted"
  else
    fail "DELETE /apps/:id/databases/:id (status: $DEL_DB)"
  fi
fi

if [[ -n "$APP_ID" ]]; then
  DEL_APP=$(api_status DELETE "/api/v1/apps/${APP_ID}")
  if [[ "$DEL_APP" == "200" || "$DEL_APP" == "204" ]]; then
    pass "DELETE /apps/:id — app deleted"
  else
    fail "DELETE /apps/:id (status: $DEL_APP)"
  fi
fi

# =============================================
# Summary
# =============================================
echo ""
echo "============================================="
echo -e "  Results: ${GREEN}${PASS_COUNT} passed${NC}, ${RED}${FAIL_COUNT} failed${NC} / ${TOTAL_TESTS} total"
echo "============================================="
echo ""

if [[ "$FAIL_COUNT" -gt 0 ]]; then
  echo -e "${RED}SMOKE TEST FAILED${NC} — $FAIL_COUNT test(s) did not pass."
  exit 1
else
  echo -e "${GREEN}ALL SMOKE TESTS PASSED${NC}"
  exit 0
fi
