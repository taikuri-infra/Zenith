#!/bin/bash
# =============================================================================
# Zenith Platform — Business Owner Smoke Test (Admin Userflow)
#
# Tests the complete platform owner journey:
#   admin login → dashboard → user management → support tickets
#   → audit log → settings → modules → infrastructure → autoscaler
#
# Usage:
#   ./infra/scripts/smoke-test-owner.sh [--api-url URL] [--verbose]
#
# Environment variables:
#   ZENITH_API_URL     Base URL (default: http://localhost:8080)
#   ADMIN_EMAIL        Admin email (required)
#   ADMIN_PASSWORD     Admin password (required)
#
# Returns exit 0 if all tests pass, exit 1 if any fail.
# =============================================================================

set -uo pipefail

VERBOSE=false
API_URL="${ZENITH_API_URL:-http://localhost:8080}"
EMAIL="${ADMIN_EMAIL:-}"
PASSWORD="${ADMIN_PASSWORD:-}"

for arg in "$@"; do
  case "$arg" in
    --verbose|-v) VERBOSE=true ;;
    --api-url=*) API_URL="${arg#*=}" ;;
    --email=*) EMAIL="${arg#*=}" ;;
    --password=*) PASSWORD="${arg#*=}" ;;
  esac
done

if [[ -z "$EMAIL" || -z "$PASSWORD" ]]; then
  echo "ERROR: ADMIN_EMAIL and ADMIN_PASSWORD must be set."
  echo "Usage: ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=secret ./smoke-test-owner.sh"
  exit 1
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0
TOTAL_TESTS=0
TOKEN=""
TEST_USER_ID=""
TEST_CUSTOMER_ID=""
TEST_PLAN_ID=""
TEST_TICKET_ID=""

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
echo "   Zenith Owner Smoke Test (Admin)"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "   API: ${API_URL}"
echo "   Admin: ${EMAIL}"
echo "============================================="

# =============================================
# 0. Health
# =============================================
section "0" "Infrastructure health"

HEALTH=$(curl -sf "${API_URL}/health" 2>/dev/null)
if echo "$HEALTH" | grep -q '"status":"healthy"'; then
  pass "GET /health — healthy"
else
  fail "GET /health" "$HEALTH"
fi

READY=$(curl -sf "${API_URL}/ready" 2>/dev/null)
if echo "$READY" | grep -q '"status":"ready"'; then
  pass "GET /ready — all checks ready"
else
  fail "GET /ready" "$READY"
fi

# =============================================
# 1. Admin Login
# =============================================
section "1" "Admin authentication"

LOGIN_RESP=$(curl -sf -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}" \
  "${API_URL}/api/v1/auth/login" 2>/dev/null)
if echo "$LOGIN_RESP" | grep -q '"token"'; then
  TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /auth/login — admin login successful"
else
  fail "POST /auth/login" "$LOGIN_RESP"
  echo -e "${RED}Cannot continue without admin token. Exiting.${NC}"
  exit 1
fi

# Verify admin role
STATUS_CODE=$(api_status GET "/api/v1/admin/dashboard/stats")
if [[ "$STATUS_CODE" == "200" ]]; then
  pass "Admin role verified (dashboard accessible)"
else
  fail "Admin role check — dashboard returned $STATUS_CODE"
  echo -e "${RED}User is not admin. Exiting.${NC}"
  exit 1
fi

# =============================================
# 2. Dashboard
# =============================================
section "2" "Admin dashboard"

STATS=$(api_get "/api/v1/admin/dashboard/stats")
if echo "$STATS" | grep -q '{'; then
  pass "GET /admin/dashboard/stats — dashboard stats retrieved"
else
  fail "GET /admin/dashboard/stats" "$STATS"
fi

# Dashboard usage (SaaS only)
USAGE_STATUS=$(api_status GET "/api/v1/admin/dashboard/usage")
if [[ "$USAGE_STATUS" == "200" || "$USAGE_STATUS" == "404" ]]; then
  pass "GET /admin/dashboard/usage — responded (status: $USAGE_STATUS)"
else
  fail "GET /admin/dashboard/usage (status: $USAGE_STATUS)"
fi

# =============================================
# 3. User Management
# =============================================
section "3" "User management"

# First, create a test user to manage
REG_RESP=$(curl -sf -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"owner-test-$(date +%s)@test.zenith.dev\",\"password\":\"OwnerTest123!\",\"name\":\"Owner Test\"}" \
  "${API_URL}/api/v1/auth/register" 2>/dev/null)
if echo "$REG_RESP" | grep -q '"user"'; then
  TEST_USER_ID=$(echo "$REG_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "Test user created for management tests (${TEST_USER_ID:0:8}...)"
fi

if [[ -n "$TEST_USER_ID" ]]; then
  USER_DETAIL=$(api_get "/api/v1/admin/users/${TEST_USER_ID}")
  if echo "$USER_DETAIL" | grep -q '"id"\|"email"'; then
    pass "GET /admin/users/:id — user details retrieved"
  else
    fail "GET /admin/users/:id" "$USER_DETAIL"
  fi

  USER_APPS=$(api_get "/api/v1/admin/users/${TEST_USER_ID}/apps")
  if [[ $? -eq 0 ]]; then
    pass "GET /admin/users/:id/apps — user apps retrieved"
  else
    fail "GET /admin/users/:id/apps"
  fi

  USER_DBS=$(api_get "/api/v1/admin/users/${TEST_USER_ID}/databases")
  if [[ $? -eq 0 ]]; then
    pass "GET /admin/users/:id/databases — user databases retrieved"
  else
    fail "GET /admin/users/:id/databases"
  fi

  # Set user plan
  PLAN_SET=$(api_post "/api/v1/admin/users/${TEST_USER_ID}/plan" '{"tier":"pro"}')
  if [[ $? -eq 0 ]]; then
    pass "POST /admin/users/:id/plan — user plan set to pro"
  else
    fail "POST /admin/users/:id/plan" "$PLAN_SET"
  fi
fi

# =============================================
# 4. Support Tickets (Admin view)
# =============================================
section "4" "Support ticket management"

TICKETS=$(api_get "/api/v1/admin/support/tickets")
if [[ $? -eq 0 ]]; then
  pass "GET /admin/support/tickets — ticket list retrieved"
  TEST_TICKET_ID=$(echo "$TICKETS" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
else
  fail "GET /admin/support/tickets"
fi

if [[ -n "$TEST_TICKET_ID" ]]; then
  TICKET_DETAIL=$(api_get "/api/v1/admin/support/tickets/${TEST_TICKET_ID}")
  if echo "$TICKET_DETAIL" | grep -q '"id"'; then
    pass "GET /admin/support/tickets/:id — ticket details retrieved"
  else
    fail "GET /admin/support/tickets/:id" "$TICKET_DETAIL"
  fi

  REPLY_STATUS=$(api_status POST "/api/v1/admin/support/tickets/${TEST_TICKET_ID}/reply" '{"content":"Admin reply from smoke test"}')
  if [[ "$REPLY_STATUS" == "200" || "$REPLY_STATUS" == "201" ]]; then
    pass "POST /admin/support/tickets/:id/reply — admin reply sent"
  else
    fail "POST /admin/support/tickets/:id/reply (status: $REPLY_STATUS)"
  fi

  STATUS_UPD=$(api_status PUT "/api/v1/admin/support/tickets/${TEST_TICKET_ID}/status" '{"status":"in_progress"}')
  if [[ "$STATUS_UPD" == "200" ]]; then
    pass "PUT /admin/support/tickets/:id/status — ticket status updated"
  else
    fail "PUT /admin/support/tickets/:id/status (status: $STATUS_UPD)"
  fi

  ASSIGN_STATUS=$(api_status PUT "/api/v1/admin/support/tickets/${TEST_TICKET_ID}/assign" '{"assignee_id":"self"}')
  if [[ "$ASSIGN_STATUS" == "200" || "$ASSIGN_STATUS" == "400" ]]; then
    pass "PUT /admin/support/tickets/:id/assign — assign endpoint responded (status: $ASSIGN_STATUS)"
  else
    fail "PUT /admin/support/tickets/:id/assign (status: $ASSIGN_STATUS)"
  fi
fi

# =============================================
# 5. Audit Log
# =============================================
section "5" "Audit log"

AUDIT=$(api_get "/api/v1/admin/audit")
if [[ $? -eq 0 ]]; then
  pass "GET /admin/audit — audit log retrieved"
else
  fail "GET /admin/audit"
fi

AUDIT_CSV_STATUS=$(api_status GET "/api/v1/admin/audit/export/csv")
if [[ "$AUDIT_CSV_STATUS" == "200" ]]; then
  pass "GET /admin/audit/export/csv — CSV export available"
else
  fail "GET /admin/audit/export/csv (status: $AUDIT_CSV_STATUS)"
fi

AUDIT_JSON_STATUS=$(api_status GET "/api/v1/admin/audit/export/json")
if [[ "$AUDIT_JSON_STATUS" == "200" ]]; then
  pass "GET /admin/audit/export/json — JSON export available"
else
  fail "GET /admin/audit/export/json (status: $AUDIT_JSON_STATUS)"
fi

# =============================================
# 6. Modules
# =============================================
section "6" "Module management"

MODULES=$(api_get "/api/v1/admin/modules")
if [[ $? -eq 0 ]]; then
  pass "GET /admin/modules — module list retrieved"
else
  fail "GET /admin/modules"
fi

# Module update-all
MOD_UPD_ALL=$(api_status POST "/api/v1/admin/modules/update-all" '{}')
if [[ "$MOD_UPD_ALL" == "200" || "$MOD_UPD_ALL" == "202" ]]; then
  pass "POST /admin/modules/update-all — update-all responded (status: $MOD_UPD_ALL)"
else
  fail "POST /admin/modules/update-all (status: $MOD_UPD_ALL)"
fi

# Module install (use a known module name like 'monitoring')
MOD_INSTALL=$(api_status POST "/api/v1/admin/modules/monitoring/install" '{}')
if [[ "$MOD_INSTALL" == "200" || "$MOD_INSTALL" == "202" || "$MOD_INSTALL" == "409" ]]; then
  pass "POST /admin/modules/:name/install — install responded (status: $MOD_INSTALL)"
else
  fail "POST /admin/modules/:name/install (status: $MOD_INSTALL)"
fi

# Module update
MOD_UPDATE=$(api_status POST "/api/v1/admin/modules/monitoring/update" '{}')
if [[ "$MOD_UPDATE" == "200" || "$MOD_UPDATE" == "202" || "$MOD_UPDATE" == "404" ]]; then
  pass "POST /admin/modules/:name/update — update responded (status: $MOD_UPDATE)"
else
  fail "POST /admin/modules/:name/update (status: $MOD_UPDATE)"
fi

# =============================================
# 7. Infrastructure
# =============================================
section "7" "Infrastructure overview"

INFRA=$(api_get "/api/v1/admin/infrastructure")
if [[ $? -eq 0 ]]; then
  pass "GET /admin/infrastructure — infrastructure overview retrieved"
else
  fail "GET /admin/infrastructure"
fi

# =============================================
# 8. Platform State
# =============================================
section "8" "Platform state"

STATE=$(api_get "/api/v1/admin/state")
if [[ $? -eq 0 ]]; then
  pass "GET /admin/state — platform state retrieved"
else
  fail "GET /admin/state"
fi

STATE_EXPORT_STATUS=$(api_status GET "/api/v1/admin/state/export")
if [[ "$STATE_EXPORT_STATUS" == "200" ]]; then
  pass "GET /admin/state/export — state export available"
else
  fail "GET /admin/state/export (status: $STATE_EXPORT_STATUS)"
fi

# =============================================
# 9. Settings
# =============================================
section "9" "Admin settings"

SETTINGS=$(api_get "/api/v1/admin/settings")
if [[ $? -eq 0 ]]; then
  pass "GET /admin/settings — settings retrieved"
else
  fail "GET /admin/settings"
fi

# Settings update (PUT)
SETTINGS_UPD=$(api_status PUT "/api/v1/admin/settings" '{"maintenance_mode":false}')
if [[ "$SETTINGS_UPD" == "200" ]]; then
  pass "PUT /admin/settings — settings updated"
else
  fail "PUT /admin/settings (status: $SETTINGS_UPD)"
fi

# Settings patch (PATCH)
SETTINGS_PATCH=$(curl -so /dev/null -w "%{http_code}" -X PATCH \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"maintenance_mode":false}' \
  "${API_URL}/api/v1/admin/settings" 2>/dev/null)
if [[ "$SETTINGS_PATCH" == "200" ]]; then
  pass "PATCH /admin/settings — settings patched"
else
  fail "PATCH /admin/settings (status: $SETTINGS_PATCH)"
fi

# =============================================
# 10. Updates
# =============================================
section "10" "Platform updates"

UPDATES=$(api_get "/api/v1/admin/updates/check")
if [[ $? -eq 0 ]]; then
  pass "GET /admin/updates/check — update check completed"
else
  fail "GET /admin/updates/check"
fi

UPDATE_HISTORY=$(api_get "/api/v1/admin/updates/history")
if [[ $? -eq 0 ]]; then
  pass "GET /admin/updates/history — update history retrieved"
else
  fail "GET /admin/updates/history"
fi

# Updates apply (don't actually apply, just check endpoint exists)
UPDATE_APPLY=$(api_status POST "/api/v1/admin/updates/apply" '{}')
if [[ "$UPDATE_APPLY" == "200" || "$UPDATE_APPLY" == "400" || "$UPDATE_APPLY" == "409" ]]; then
  pass "POST /admin/updates/apply — apply endpoint responded (status: $UPDATE_APPLY)"
else
  fail "POST /admin/updates/apply (status: $UPDATE_APPLY)"
fi

# =============================================
# 11. Autoscaler (may be disabled)
# =============================================
section "11" "Autoscaler"

AS_STATUS=$(api_status GET "/api/v1/admin/autoscaler/status")
if [[ "$AS_STATUS" == "200" ]]; then
  pass "GET /admin/autoscaler/status — autoscaler status available"
elif [[ "$AS_STATUS" == "404" ]]; then
  pass "GET /admin/autoscaler/status — autoscaler not enabled (expected in standalone)"
else
  fail "GET /admin/autoscaler/status (status: $AS_STATUS)"
fi

AS_NODES=$(api_status GET "/api/v1/admin/autoscaler/nodes")
if [[ "$AS_NODES" == "200" || "$AS_NODES" == "404" ]]; then
  pass "GET /admin/autoscaler/nodes — response OK"
else
  fail "GET /admin/autoscaler/nodes (status: $AS_NODES)"
fi

AS_EVENTS=$(api_status GET "/api/v1/admin/autoscaler/events")
if [[ "$AS_EVENTS" == "200" || "$AS_EVENTS" == "404" ]]; then
  pass "GET /admin/autoscaler/events — response OK"
else
  fail "GET /admin/autoscaler/events (status: $AS_EVENTS)"
fi

# =============================================
# 12. Admin Customers (SaaS CRUD)
# =============================================
section "12" "Admin customer management (SaaS)"

CUSTOMERS_STATUS=$(api_status GET "/api/v1/admin/customers")
if [[ "$CUSTOMERS_STATUS" == "200" ]]; then
  pass "GET /admin/customers — customer list retrieved"
elif [[ "$CUSTOMERS_STATUS" == "404" ]]; then
  pass "GET /admin/customers — not available (standalone mode)"
else
  fail "GET /admin/customers (status: $CUSTOMERS_STATUS)"
fi

CUSTOMER_STATS_STATUS=$(api_status GET "/api/v1/admin/customers/stats")
if [[ "$CUSTOMER_STATS_STATUS" == "200" || "$CUSTOMER_STATS_STATUS" == "404" ]]; then
  pass "GET /admin/customers/stats — responded (status: $CUSTOMER_STATS_STATUS)"
else
  fail "GET /admin/customers/stats (status: $CUSTOMER_STATS_STATUS)"
fi

# Create test customer
CREATE_CUST=$(api_post "/api/v1/admin/customers" '{"name":"Smoke Test Co","email":"smoke-cust@test.dev","plan":"free"}')
if echo "$CREATE_CUST" | grep -q '"id"'; then
  TEST_CUSTOMER_ID=$(echo "$CREATE_CUST" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /admin/customers — customer created (${TEST_CUSTOMER_ID:0:8}...)"
else
  CUST_CREATE_STATUS=$(api_status POST "/api/v1/admin/customers" '{"name":"Smoke Test Co","email":"smoke-cust@test.dev","plan":"free"}')
  if [[ "$CUST_CREATE_STATUS" == "404" ]]; then
    pass "POST /admin/customers — not available (standalone mode)"
  else
    fail "POST /admin/customers" "$CREATE_CUST"
  fi
fi

if [[ -n "$TEST_CUSTOMER_ID" ]]; then
  CUST_DETAIL=$(api_get "/api/v1/admin/customers/${TEST_CUSTOMER_ID}")
  if echo "$CUST_DETAIL" | grep -q '"id"'; then
    pass "GET /admin/customers/:id — customer detail retrieved"
  else
    fail "GET /admin/customers/:id" "$CUST_DETAIL"
  fi

  CUST_USAGE_STATUS=$(api_status GET "/api/v1/admin/customers/${TEST_CUSTOMER_ID}/usage")
  if [[ "$CUST_USAGE_STATUS" == "200" ]]; then
    pass "GET /admin/customers/:id/usage — customer usage retrieved"
  else
    fail "GET /admin/customers/:id/usage (status: $CUST_USAGE_STATUS)"
  fi

  CUST_HISTORY_STATUS=$(api_status GET "/api/v1/admin/customers/${TEST_CUSTOMER_ID}/usage/history")
  if [[ "$CUST_HISTORY_STATUS" == "200" ]]; then
    pass "GET /admin/customers/:id/usage/history — usage history retrieved"
  else
    fail "GET /admin/customers/:id/usage/history (status: $CUST_HISTORY_STATUS)"
  fi

  CUST_UPDATE=$(api_status PUT "/api/v1/admin/customers/${TEST_CUSTOMER_ID}" '{"name":"Smoke Test Updated"}')
  if [[ "$CUST_UPDATE" == "200" ]]; then
    pass "PUT /admin/customers/:id — customer updated"
  else
    fail "PUT /admin/customers/:id (status: $CUST_UPDATE)"
  fi

  CUST_CLUSTER_STATUS=$(api_status GET "/api/v1/admin/customers/${TEST_CUSTOMER_ID}/cluster")
  if [[ "$CUST_CLUSTER_STATUS" == "200" || "$CUST_CLUSTER_STATUS" == "404" ]]; then
    pass "GET /admin/customers/:id/cluster — responded (status: $CUST_CLUSTER_STATUS)"
  else
    fail "GET /admin/customers/:id/cluster (status: $CUST_CLUSTER_STATUS)"
  fi

  # Suspend + activate cycle
  CUST_SUSPEND=$(api_status POST "/api/v1/admin/customers/${TEST_CUSTOMER_ID}/suspend" '{}')
  if [[ "$CUST_SUSPEND" == "200" ]]; then
    pass "POST /admin/customers/:id/suspend — customer suspended"
  else
    fail "POST /admin/customers/:id/suspend (status: $CUST_SUSPEND)"
  fi

  CUST_ACTIVATE=$(api_status POST "/api/v1/admin/customers/${TEST_CUSTOMER_ID}/activate" '{}')
  if [[ "$CUST_ACTIVATE" == "200" ]]; then
    pass "POST /admin/customers/:id/activate — customer activated"
  else
    fail "POST /admin/customers/:id/activate (status: $CUST_ACTIVATE)"
  fi

  # Delete test customer
  CUST_DELETE=$(api_status DELETE "/api/v1/admin/customers/${TEST_CUSTOMER_ID}")
  if [[ "$CUST_DELETE" == "200" || "$CUST_DELETE" == "204" ]]; then
    pass "DELETE /admin/customers/:id — customer deleted"
  else
    fail "DELETE /admin/customers/:id (status: $CUST_DELETE)"
  fi
fi

# =============================================
# 13. Admin Plans (SaaS)
# =============================================
section "13" "Admin plan management (SaaS)"

PLANS_STATUS=$(api_status GET "/api/v1/admin/plans")
if [[ "$PLANS_STATUS" == "200" ]]; then
  pass "GET /admin/plans — plan list retrieved"
elif [[ "$PLANS_STATUS" == "404" ]]; then
  pass "GET /admin/plans — not available (standalone mode)"
else
  fail "GET /admin/plans (status: $PLANS_STATUS)"
fi

CREATE_PLAN=$(api_post "/api/v1/admin/plans" '{"name":"smoke-plan","tier":"pro","price_cents":1000,"limits":{"apps":3}}')
if echo "$CREATE_PLAN" | grep -q '"id"'; then
  TEST_PLAN_ID=$(echo "$CREATE_PLAN" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  pass "POST /admin/plans — plan created (${TEST_PLAN_ID:0:8}...)"
else
  PLAN_CREATE_STATUS=$(api_status POST "/api/v1/admin/plans" '{"name":"smoke-plan","tier":"pro","price_cents":1000}')
  if [[ "$PLAN_CREATE_STATUS" == "404" || "$PLAN_CREATE_STATUS" == "409" ]]; then
    pass "POST /admin/plans — responded (status: $PLAN_CREATE_STATUS)"
  else
    fail "POST /admin/plans" "$CREATE_PLAN"
  fi
fi

if [[ -n "$TEST_PLAN_ID" ]]; then
  PLAN_UPD=$(api_status PUT "/api/v1/admin/plans/${TEST_PLAN_ID}" '{"name":"smoke-plan-updated","price_cents":2000}')
  if [[ "$PLAN_UPD" == "200" ]]; then
    pass "PUT /admin/plans/:id — plan updated"
  else
    fail "PUT /admin/plans/:id (status: $PLAN_UPD)"
  fi
fi

# =============================================
# 14. Admin Clusters (SaaS)
# =============================================
section "14" "Admin cluster management (SaaS)"

CLUSTERS_STATUS=$(api_status GET "/api/v1/admin/clusters")
if [[ "$CLUSTERS_STATUS" == "200" || "$CLUSTERS_STATUS" == "404" ]]; then
  pass "GET /admin/clusters — responded (status: $CLUSTERS_STATUS)"
else
  fail "GET /admin/clusters (status: $CLUSTERS_STATUS)"
fi

# =============================================
# 15. Admin Tenants (SaaS)
# =============================================
section "15" "Admin tenant management (SaaS)"

TENANTS_STATUS=$(api_status GET "/api/v1/admin/tenants")
if [[ "$TENANTS_STATUS" == "200" || "$TENANTS_STATUS" == "404" ]]; then
  pass "GET /admin/tenants — responded (status: $TENANTS_STATUS)"
else
  fail "GET /admin/tenants (status: $TENANTS_STATUS)"
fi

# =============================================
# 16. Admin Billing (SaaS)
# =============================================
section "16" "Admin billing overview"

BILLING_STATUS=$(api_status GET "/api/v1/admin/billing/overview")
if [[ "$BILLING_STATUS" == "200" ]]; then
  pass "GET /admin/billing/overview — billing overview available"
elif [[ "$BILLING_STATUS" == "404" ]]; then
  pass "GET /admin/billing/overview — not available (standalone mode)"
else
  fail "GET /admin/billing/overview (status: $BILLING_STATUS)"
fi

# =============================================
# 17. Platform Metrics (Prometheus)
# =============================================
section "17" "Platform metrics"

METRICS_STATUS=$(api_status GET "/metrics")
if [[ "$METRICS_STATUS" == "200" ]]; then
  pass "GET /metrics — Prometheus metrics endpoint available"
  # Verify key business metrics are exposed
  METRICS_BODY=$(curl -sf "${API_URL}/metrics" 2>/dev/null)
  if echo "$METRICS_BODY" | grep -q "zenith_mrr_euros"; then
    pass "Business metric zenith_mrr_euros present"
  else
    fail "Business metric zenith_mrr_euros missing"
  fi
  if echo "$METRICS_BODY" | grep -q "zenith_total_users"; then
    pass "Business metric zenith_total_users present"
  else
    fail "Business metric zenith_total_users missing"
  fi
  if echo "$METRICS_BODY" | grep -q "zenith_total_apps"; then
    pass "Business metric zenith_total_apps present"
  else
    fail "Business metric zenith_total_apps missing"
  fi
else
  fail "GET /metrics (status: $METRICS_STATUS)"
fi

# =============================================
# 18. Security verification
# =============================================
section "18" "Security checks"

# Non-admin should not access admin routes
# Register a regular user and try
REG_REGULAR=$(curl -sf -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"nonadmin-$(date +%s)@test.zenith.dev\",\"password\":\"Regular123!\",\"name\":\"Regular User\"}" \
  "${API_URL}/api/v1/auth/register" 2>/dev/null)
if echo "$REG_REGULAR" | grep -q '"token"'; then
  REGULAR_TOKEN=$(echo "$REG_REGULAR" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)
  ADMIN_ACCESS=$(curl -so /dev/null -w "%{http_code}" -H "Authorization: Bearer $REGULAR_TOKEN" \
    "${API_URL}/api/v1/admin/dashboard/stats" 2>/dev/null)
  if [[ "$ADMIN_ACCESS" == "403" ]]; then
    pass "Non-admin gets 403 on admin routes"
  else
    fail "Non-admin should get 403, got $ADMIN_ACCESS"
  fi
fi

# Verify no sensitive info in health endpoint
HEALTH_CHECK=$(curl -sf "${API_URL}/health" 2>/dev/null)
if echo "$HEALTH_CHECK" | grep -q '"git_commit"'; then
  fail "Health endpoint exposes git_commit (information disclosure)"
else
  pass "Health endpoint does not expose git_commit"
fi

# Verify security headers present
SECURITY_HEADERS=$(curl -sI "${API_URL}/health" 2>/dev/null)
HEADERS_OK=true
for HEADER in "X-Content-Type-Options" "X-Frame-Options" "Referrer-Policy" "Content-Security-Policy"; do
  if ! echo "$SECURITY_HEADERS" | grep -qi "$HEADER"; then
    fail "Missing security header: $HEADER"
    HEADERS_OK=false
  fi
done
if [[ "$HEADERS_OK" == "true" ]]; then
  pass "All OWASP security headers present"
fi

# Server header should not reveal software
if echo "$SECURITY_HEADERS" | grep -qi "^Server: Zenith"; then
  fail "Server header reveals software identity"
else
  pass "Server header does not reveal software identity"
fi

# HSTS header
if echo "$SECURITY_HEADERS" | grep -qi "Strict-Transport-Security"; then
  pass "HSTS header present"
else
  fail "Missing HSTS (Strict-Transport-Security) header"
fi

# Permissions-Policy header
if echo "$SECURITY_HEADERS" | grep -qi "Permissions-Policy"; then
  pass "Permissions-Policy header present"
else
  fail "Missing Permissions-Policy header"
fi

# Admin audit log
ADMIN_AUDIT=$(api_get "/api/v1/admin/audit")
if [[ $? -eq 0 ]]; then
  pass "GET /admin/audit — admin audit log retrieved"
else
  fail "GET /admin/audit"
fi

ADMIN_AUDIT_CSV_STATUS=$(api_status GET "/api/v1/admin/audit/export/csv")
if [[ "$ADMIN_AUDIT_CSV_STATUS" == "200" ]]; then
  pass "GET /admin/audit/export/csv — admin audit CSV export"
else
  fail "GET /admin/audit/export/csv (status: $ADMIN_AUDIT_CSV_STATUS)"
fi

ADMIN_AUDIT_JSON_STATUS=$(api_status GET "/api/v1/admin/audit/export/json")
if [[ "$ADMIN_AUDIT_JSON_STATUS" == "200" ]]; then
  pass "GET /admin/audit/export/json — admin audit JSON export"
else
  fail "GET /admin/audit/export/json (status: $ADMIN_AUDIT_JSON_STATUS)"
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
  echo -e "${RED}OWNER SMOKE TEST FAILED${NC} — $FAIL_COUNT test(s) did not pass."
  exit 1
else
  echo -e "${GREEN}ALL OWNER SMOKE TESTS PASSED${NC}"
  exit 0
fi
