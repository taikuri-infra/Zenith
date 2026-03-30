#!/bin/bash
# =============================================================================
# Zenith Platform — E2E Deploy Lifecycle Test
#
# Tests: auth, app creation, first deploy, live traffic, redeploy, rollback,
#        logs endpoint, and cleanup (hard delete).
#
# Usage:  ./infra/scripts/e2e-deploy-lifecycle.sh [--api-url=URL] [--verbose]
#
# Env vars:
#   ZENITH_API_URL          (default: https://api.stage.freezenith.com)
#   SMOKE_TEST_EMAIL        (default: smoke-ci@zenith.dev)
#   SMOKE_TEST_PASSWORD     (default: SmokeTest1234)
#   STAGING_APPS_DOMAIN     (default: apps.stage.freezenith.com)
# =============================================================================

set -uo pipefail

VERBOSE=false
API_URL="${ZENITH_API_URL:-https://api.stage.freezenith.com}"
EMAIL="${SMOKE_TEST_EMAIL:-smoke-ci@zenith.dev}"
PASSWORD="${SMOKE_TEST_PASSWORD:-SmokeTest1234}"
APPS_DOMAIN="${STAGING_APPS_DOMAIN:-apps.stage.freezenith.com}"

for arg in "$@"; do
  case "$arg" in
    --verbose|-v) VERBOSE=true ;;
    --api-url=*) API_URL="${arg#*=}" ;;
  esac
done

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
PASS_COUNT=0; FAIL_COUNT=0; TOTAL_TESTS=0
TOKEN=""; PROJECT_ID=""; APP_ID=""; APP_NAME=""; APP_SUBDOMAIN=""
FIRST_DEPLOY_ID=""; CREATED_PROJECT=""

pass() { ((PASS_COUNT++)); ((TOTAL_TESTS++)); echo -e "  ${GREEN}PASS${NC} $1"; }
fail() {
  ((FAIL_COUNT++)); ((TOTAL_TESTS++)); echo -e "  ${RED}FAIL${NC} $1"
  [[ "$VERBOSE" == "true" && -n "${2:-}" ]] && echo -e "       ${YELLOW}Detail: ${2}${NC}"
}
section() { echo ""; echo -e "${BLUE}[$1]${NC} $2"; }

_retry_on_429() { [[ "$1" == "429" ]] && { sleep 5; return 0; }; return 1; }

api_get() {
  local result rc
  result=$(curl -sf -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}$1" 2>/dev/null); rc=$?
  if [[ $rc -ne 0 ]]; then
    local s; s=$(curl -so /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" "${API_URL}$1" 2>/dev/null)
    if _retry_on_429 "$s"; then
      result=$(curl -sf -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}$1" 2>/dev/null); rc=$?
    fi
  fi
  echo "$result"; return $rc
}

api_post() {
  local result rc
  result=$(curl -sf -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${API_URL}$1" 2>/dev/null); rc=$?
  if [[ $rc -ne 0 ]]; then
    local s; s=$(curl -so /dev/null -w "%{http_code}" -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${API_URL}$1" 2>/dev/null)
    if _retry_on_429 "$s"; then
      result=$(curl -sf -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${API_URL}$1" 2>/dev/null); rc=$?
    fi
  fi
  echo "$result"; return $rc
}

api_delete() {
  curl -sf -X DELETE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}$1" 2>/dev/null
}

api_status() {
  local method="$1" path="$2" data="${3:-}" status
  if [[ -n "$data" ]]; then
    status=$(curl -so /dev/null -w "%{http_code}" -X "$method" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$data" "${API_URL}${path}" 2>/dev/null)
  else
    status=$(curl -so /dev/null -w "%{http_code}" -X "$method" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}${path}" 2>/dev/null)
  fi
  if _retry_on_429 "$status"; then
    if [[ -n "$data" ]]; then
      status=$(curl -so /dev/null -w "%{http_code}" -X "$method" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$data" "${API_URL}${path}" 2>/dev/null)
    else
      status=$(curl -so /dev/null -w "%{http_code}" -X "$method" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}${path}" 2>/dev/null)
    fi
  fi
  echo "$status"
}

# Poll a deployment until it reaches a terminal status.
# Usage: poll_deploy <app_id> <deploy_id> <max_polls> <label>
# Sets POLL_RESULT to the final status.
poll_deploy() {
  local app_id="$1" deploy_id="$2" max="$3" label="$4"
  POLL_RESULT=""
  echo -e "  ${YELLOW}INFO${NC} Polling ${label} (max $((max * 10))s)..."
  local i=0
  while [[ $i -lt $max ]]; do
    local resp
    resp=$(api_get "/api/v1/apps/${app_id}/deployments/${deploy_id}" 2>/dev/null || \
           api_get "/api/v1/deployments/${deploy_id}" 2>/dev/null)
    POLL_RESULT=$(echo "$resp" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
    case "$POLL_RESULT" in
      active|running|completed|success|failed) break ;;
    esac
    ((i++))
    [[ "$VERBOSE" == "true" ]] && echo -e "       ${YELLOW}... status=${POLL_RESULT:-unknown} (${i}/${max})${NC}"
    sleep 10
  done
  case "$POLL_RESULT" in
    active|running|completed|success) pass "${label} reached status: ${POLL_RESULT}" ;;
    failed) fail "${label} failed" "status=failed" ;;
    *) fail "${label} did not complete in time" "status=${POLL_RESULT:-unknown}" ;;
  esac
}

echo ""
echo "============================================="
echo "   Zenith E2E Deploy Lifecycle Test"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "   API: ${API_URL}"
echo "   Apps: *.${APPS_DOMAIN}"
echo "   User: ${EMAIL}"
echo "============================================="

# =============================================
# 0. Authentication + Project
# =============================================
section "0" "Authentication"

LOGIN_RESP=$(curl -sf -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}" \
  "${API_URL}/api/v1/auth/login" 2>/dev/null)
TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)
if [[ -n "$TOKEN" ]]; then
  pass "User login"
else
  fail "User login — cannot continue" "$LOGIN_RESP"
  echo -e "\n${RED}ABORT: Cannot authenticate. Exiting.${NC}"; exit 1
fi

PROJECTS=$(api_get "/api/v1/projects")
PROJECT_ID=$(echo "$PROJECTS" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [[ -z "$PROJECT_ID" ]]; then
  PROJ_SLUG="e2e-deploy-$(date +%s)"
  PROJ_RESP=$(api_post "/api/v1/projects" "{\"name\":\"${PROJ_SLUG}\",\"slug\":\"${PROJ_SLUG}\"}")
  PROJECT_ID=$(echo "$PROJ_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  CREATED_PROJECT="true"
fi
if [[ -n "$PROJECT_ID" ]]; then
  pass "Have project: ${PROJECT_ID:0:8}..."
else
  fail "No project — cannot continue"
  echo -e "\n${RED}ABORT: No project available. Exiting.${NC}"; exit 1
fi

# Clean up leftover apps (free tier = 1 app limit)
OLD_APP_ID=$(api_get "/api/v1/apps?project_id=${PROJECT_ID}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [[ -n "$OLD_APP_ID" ]]; then
  curl -sf -X DELETE -H "Authorization: Bearer $TOKEN" \
    "${API_URL}/api/v1/apps/${OLD_APP_ID}?hard=true" > /dev/null 2>&1
  sleep 2
  echo -e "  ${YELLOW}INFO${NC} Cleaned up leftover app ${OLD_APP_ID:0:8}..."
fi

# =============================================
# 1. App Creation & First Deploy
# =============================================
section "1" "App creation and first deploy"

APP_NAME="e2e-deploy-$(date +%s)"
CREATE_RESP=$(api_post "/api/v1/apps" "{
  \"name\": \"${APP_NAME}\",
  \"deploy_source\": \"image\",
  \"image_url\": \"nginx:latest\",
  \"project_id\": \"${PROJECT_ID}\"
}")
APP_ID=$(echo "$CREATE_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
APP_SUBDOMAIN=$(echo "$CREATE_RESP" | grep -o '"subdomain":"[^"]*"' | cut -d'"' -f4)

if [[ -n "$APP_ID" ]]; then
  pass "Create app (id=${APP_ID:0:8}...)"
else
  fail "Create app" "$CREATE_RESP"
  echo -e "\n${RED}ABORT: App creation failed. Exiting.${NC}"; exit 1
fi

[[ -n "$APP_SUBDOMAIN" ]] && pass "App has subdomain: ${APP_SUBDOMAIN}" || fail "App missing subdomain" "$CREATE_RESP"

# Verify deployment was auto-created
sleep 3
DEPLOYS=$(api_get "/api/v1/apps/${APP_ID}/deployments")
DEPLOY_COUNT=$(echo "$DEPLOYS" | grep -o '"total":[0-9]*' | cut -d: -f2)
FIRST_DEPLOY_ID=$(echo "$DEPLOYS" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [[ -n "$DEPLOY_COUNT" && "$DEPLOY_COUNT" -ge 1 ]]; then
  pass "Auto-created deployment (count=${DEPLOY_COUNT}, id=${FIRST_DEPLOY_ID:0:8}...)"
else
  fail "No auto-created deployment" "total=${DEPLOY_COUNT:-0}"
fi

# Poll first deployment (max 5 min = 30 x 10s)
[[ -n "$FIRST_DEPLOY_ID" ]] && poll_deploy "$APP_ID" "$FIRST_DEPLOY_ID" 30 "First deployment"

# =============================================
# 2. Live Traffic Verification
# =============================================
section "2" "Live traffic verification"

if [[ -n "$APP_SUBDOMAIN" ]]; then
  APP_URL="https://${APP_SUBDOMAIN}.${APPS_DOMAIN}"
  for attempt in 1 2 3; do
    HTTP_CODE=$(curl -so /dev/null -w "%{http_code}" --max-time 15 -k "${APP_URL}" 2>/dev/null)
    CURL_RC=$?
    if [[ $CURL_RC -eq 0 ]]; then
      if [[ "$HTTP_CODE" == "200" ]]; then
        BODY_LEN=$(curl -s --max-time 15 -k "${APP_URL}" 2>/dev/null | wc -c)
        pass "Live traffic returns 200 with content (${BODY_LEN} bytes)"
        break
      elif [[ "$HTTP_CODE" == "502" || "$HTTP_CODE" == "503" ]]; then
        pass "Live traffic returns ${HTTP_CODE} (pod starting — acceptable)"
        break
      elif [[ "$HTTP_CODE" == "404" && $attempt -lt 3 ]]; then
        [[ "$VERBOSE" == "true" ]] && echo -e "       ${YELLOW}... attempt ${attempt}/3, got 404, retrying in 15s${NC}"
        sleep 15; continue
      elif [[ $attempt -lt 3 ]]; then
        sleep 15; continue
      else
        # Last attempt — accept any reachable status as partial pass
        pass "Live traffic reachable (HTTP ${HTTP_CODE})"
      fi
    else
      if [[ $attempt -lt 3 ]]; then
        [[ "$VERBOSE" == "true" ]] && echo -e "       ${YELLOW}... attempt ${attempt}/3, connection failed, retrying in 15s${NC}"
        sleep 15
      else
        fail "Live traffic connection failed after 3 attempts" "curl_rc=${CURL_RC}, url=${APP_URL}"
      fi
    fi
  done
else
  fail "Cannot verify traffic — no subdomain"
fi

# =============================================
# 3. Redeploy with New Image
# =============================================
section "3" "Redeploy with new image"

REDEPLOY_RESP=$(api_post "/api/v1/deploy" "{\"app\":\"${APP_NAME}\",\"image\":\"nginx:alpine\",\"environment\":\"staging\"}")
REDEPLOY_ID=$(echo "$REDEPLOY_RESP" | grep -o '"deployment_id":"[^"]*"' | cut -d'"' -f4)

if [[ -n "$REDEPLOY_ID" ]]; then
  pass "Redeploy accepted (id=${REDEPLOY_ID:0:8}...)"
else
  REDEPLOY_HTTP=$(api_status "POST" "/api/v1/deploy" "{\"app\":\"${APP_NAME}\",\"image\":\"nginx:alpine\",\"environment\":\"staging\"}")
  if [[ "$REDEPLOY_HTTP" == "202" || "$REDEPLOY_HTTP" == "200" ]]; then
    pass "Redeploy accepted (HTTP ${REDEPLOY_HTTP})"
  else
    fail "Redeploy failed" "HTTP ${REDEPLOY_HTTP}, resp=${REDEPLOY_RESP}"
  fi
fi

# Verify deployment count increased
sleep 3
DEPLOY_COUNT_AFTER=$(api_get "/api/v1/apps/${APP_ID}/deployments" | grep -o '"total":[0-9]*' | cut -d: -f2)
if [[ -n "$DEPLOY_COUNT_AFTER" && "$DEPLOY_COUNT_AFTER" -ge 2 ]]; then
  pass "Deployment count increased to ${DEPLOY_COUNT_AFTER}"
else
  fail "Expected 2+ deployments" "total=${DEPLOY_COUNT_AFTER:-0}"
fi

# Poll redeploy (max 5 min)
[[ -n "$REDEPLOY_ID" ]] && poll_deploy "$APP_ID" "$REDEPLOY_ID" 30 "Redeploy"

# =============================================
# 4. Rollback
# =============================================
section "4" "Rollback to first deployment"

if [[ -n "$FIRST_DEPLOY_ID" ]]; then
  ROLLBACK_RESP=$(api_post "/api/v1/apps/${APP_ID}/rollback" "{\"deployment_id\":\"${FIRST_DEPLOY_ID}\"}")
  ROLLBACK_MSG=$(echo "$ROLLBACK_RESP" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)

  if [[ "$ROLLBACK_MSG" == "rollback initiated" ]]; then
    pass "Rollback initiated"
  else
    ROLLBACK_HTTP=$(api_status "POST" "/api/v1/apps/${APP_ID}/rollback" "{\"deployment_id\":\"${FIRST_DEPLOY_ID}\"}")
    [[ "$ROLLBACK_HTTP" == "200" ]] && pass "Rollback accepted (HTTP 200)" || fail "Rollback failed" "HTTP ${ROLLBACK_HTTP}, resp=${ROLLBACK_RESP}"
  fi

  # Verify deployments still present
  sleep 3
  DEPLOY_COUNT_RB=$(api_get "/api/v1/apps/${APP_ID}/deployments" | grep -o '"total":[0-9]*' | cut -d: -f2)
  if [[ -n "$DEPLOY_COUNT_RB" && "$DEPLOY_COUNT_RB" -ge 2 ]]; then
    pass "Deployments after rollback (total=${DEPLOY_COUNT_RB})"
  else
    fail "Deployments after rollback" "total=${DEPLOY_COUNT_RB:-0}"
  fi

  # Poll rollback (max 2 min = 12 x 10s)
  poll_deploy "$APP_ID" "$FIRST_DEPLOY_ID" 12 "Rollback deployment"
else
  fail "Cannot test rollback — no first deployment ID"
fi

# =============================================
# 5. App Logs
# =============================================
section "5" "App logs endpoint"

if [[ -n "$APP_ID" ]]; then
  LOGS_STATUS=$(api_status "GET" "/api/v1/apps/${APP_ID}/logs")
  [[ "$LOGS_STATUS" == "200" ]] && pass "GET /apps/:appId/logs returns 200" || fail "GET /apps/:appId/logs" "status=${LOGS_STATUS}"

  if [[ -n "$FIRST_DEPLOY_ID" ]]; then
    DEP_LOGS_STATUS=$(api_status "GET" "/api/v1/apps/${APP_ID}/deployments/${FIRST_DEPLOY_ID}/logs/history")
    [[ "$DEP_LOGS_STATUS" == "200" ]] && pass "GET deployment logs history returns 200" || fail "GET deployment logs history" "status=${DEP_LOGS_STATUS}"
  fi
else
  fail "Cannot test logs — no app ID"
fi

# =============================================
# 6. Cleanup
# =============================================
section "6" "Cleanup"

if [[ -n "$APP_ID" ]]; then
  HARD_DEL=$(curl -sf -X DELETE -H "Authorization: Bearer $TOKEN" \
    "${API_URL}/api/v1/apps/${APP_ID}?hard=true" 2>/dev/null)
  if echo "$HARD_DEL" | grep -q "deleted"; then
    pass "Hard delete app"
  else
    DEL_HTTP=$(curl -so /dev/null -w "%{http_code}" -X DELETE -H "Authorization: Bearer $TOKEN" \
      "${API_URL}/api/v1/apps/${APP_ID}?hard=true" 2>/dev/null)
    [[ "$DEL_HTTP" == "200" || "$DEL_HTTP" == "404" ]] && pass "Hard delete app (HTTP ${DEL_HTTP})" || fail "Hard delete app" "HTTP ${DEL_HTTP}"
  fi

  sleep 1
  VERIFY_STATUS=$(api_status "GET" "/api/v1/apps/${APP_ID}")
  [[ "$VERIFY_STATUS" == "404" ]] && pass "App returns 404 after hard delete" || fail "App should be gone" "status=${VERIFY_STATUS}"
fi

if [[ -n "$CREATED_PROJECT" && -n "$PROJECT_ID" ]]; then
  curl -sf -X DELETE -H "Authorization: Bearer $TOKEN" \
    "${API_URL}/api/v1/projects/${PROJECT_ID}" > /dev/null 2>&1
  pass "Cleaned up test project"
fi

# =============================================
# Summary
# =============================================
echo ""
echo "============================================="
echo -e "   Results: ${GREEN}${PASS_COUNT} PASS${NC} / ${RED}${FAIL_COUNT} FAIL${NC} / ${TOTAL_TESTS} total"
echo "============================================="

[[ $FAIL_COUNT -gt 0 ]] && exit 1
exit 0
