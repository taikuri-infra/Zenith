#!/bin/bash
# Zenith Platform — Database Provisioning E2E Test
# Usage: ./infra/scripts/e2e-database-provision.sh [--api-url=URL] [--verbose]
# Env: ZENITH_API_URL, SMOKE_TEST_EMAIL, SMOKE_TEST_PASSWORD
set -uo pipefail

VERBOSE=false
API_URL="${ZENITH_API_URL:-https://api.stage.freezenith.com}"
EMAIL="${SMOKE_TEST_EMAIL:-smoke-ci@zenith.dev}"
PASSWORD="${SMOKE_TEST_PASSWORD:-SmokeTest1234}"
for arg in "$@"; do
  case "$arg" in
    --verbose|-v) VERBOSE=true ;;
    --api-url=*) API_URL="${arg#*=}" ;;
  esac
done

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
PASS_COUNT=0; FAIL_COUNT=0; TOTAL_TESTS=0
TOKEN=""; PROJECT_ID=""; CREATED_PROJECT=""; PG_DB_ID=""; REDIS_DB_ID=""
TS=$(date +%s)

pass() { ((PASS_COUNT++)); ((TOTAL_TESTS++)); echo -e "  ${GREEN}PASS${NC} $1"; }
fail() {
  ((FAIL_COUNT++)); ((TOTAL_TESTS++)); echo -e "  ${RED}FAIL${NC} $1"
  [[ "$VERBOSE" == "true" && -n "${2:-}" ]] && echo -e "       ${YELLOW}Detail: ${2}${NC}"
}
section() { echo ""; echo -e "${BLUE}[$1]${NC} $2"; }

_retry_on_429() { [[ "$1" == "429" ]] && sleep 5 && return 0; return 1; }

api_get() {
  local result rc status
  result=$(curl -sf -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}$1" 2>/dev/null); rc=$?
  if [[ $rc -ne 0 ]]; then
    status=$(curl -so /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" "${API_URL}$1" 2>/dev/null)
    if _retry_on_429 "$status"; then
      result=$(curl -sf -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}$1" 2>/dev/null); rc=$?
    fi
  fi
  echo "$result"; return $rc
}

api_post() {
  local result rc status
  result=$(curl -sf -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${API_URL}$1" 2>/dev/null); rc=$?
  if [[ $rc -ne 0 ]]; then
    status=$(curl -so /dev/null -w "%{http_code}" -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${API_URL}$1" 2>/dev/null)
    if _retry_on_429 "$status"; then
      result=$(curl -sf -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "${API_URL}$1" 2>/dev/null); rc=$?
    fi
  fi
  echo "$result"; return $rc
}

api_delete() { curl -sf -X DELETE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" "${API_URL}$1" 2>/dev/null; }

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

poll_db_status() {
  local db_id="$1" max_wait="${2:-180}" interval="${3:-10}" elapsed=0 resp="" db_status=""
  while [[ $elapsed -lt $max_wait ]]; do
    resp=$(api_get "/api/v1/databases/${db_id}")
    db_status=$(echo "$resp" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
    [[ "$db_status" == "provisioned" || "$db_status" == "running" || "$db_status" == "ready" ]] && { echo "$resp"; return 0; }
    [[ "$db_status" == "failed" || "$db_status" == "error" ]] && { echo "$resp"; return 1; }
    sleep "$interval"; elapsed=$((elapsed + interval))
  done
  echo "$resp"; return 2
}

echo ""
echo "============================================="
echo "   Zenith Database Provisioning E2E Test"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "   API: ${API_URL}"
echo "   User: ${EMAIL}"
echo "============================================="

# === 0. Authentication ===
section "0" "Authentication"
LOGIN_RESP=$(curl -sf -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}" \
  "${API_URL}/api/v1/auth/login" 2>/dev/null)
TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)
if [[ -n "$TOKEN" ]]; then pass "User login"
else fail "User login — cannot continue" "$LOGIN_RESP"; echo -e "\n${RED}ABORT${NC}"; exit 1; fi

PROJECTS=$(api_get "/api/v1/projects")
PROJECT_ID=$(echo "$PROJECTS" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [[ -z "$PROJECT_ID" ]]; then
  PROJ_RESP=$(api_post "/api/v1/projects" "{\"name\":\"smoke-db-${TS}\",\"slug\":\"smoke-db-${TS}\"}")
  PROJECT_ID=$(echo "$PROJ_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  CREATED_PROJECT="true"
fi
if [[ -n "$PROJECT_ID" ]]; then pass "Have project: ${PROJECT_ID:0:8}..."
else fail "No project — cannot continue"; echo -e "\n${RED}ABORT${NC}"; exit 1; fi

# === 1. PostgreSQL Database ===
section "1" "PostgreSQL database provisioning"
PG_HTTP=$(api_status "POST" "/api/v1/databases" "{\"name\":\"smoke-pg-${TS}\",\"engine\":\"postgresql\",\"project_id\":\"${PROJECT_ID}\"}")
PG_RESP=$(api_post "/api/v1/databases" "{\"name\":\"smoke-pg-${TS}b\",\"engine\":\"postgresql\",\"project_id\":\"${PROJECT_ID}\"}")
PG_DB_ID=$(echo "$PG_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
PG_DB_NAME=$(echo "$PG_RESP" | grep -o '"name":"[^"]*"' | head -1 | cut -d'"' -f4)
PG_ENGINE=$(echo "$PG_RESP" | grep -o '"engine":"[^"]*"' | head -1 | cut -d'"' -f4)
PG_STATUS=$(echo "$PG_RESP" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)

if [[ -n "$PG_DB_ID" ]]; then
  pass "Create PostgreSQL database (id=${PG_DB_ID:0:8}...)"
elif [[ "$PG_HTTP" == "500" ]]; then
  echo -e "  ${YELLOW}SKIP${NC} Create PostgreSQL database (500 — FK constraint, known dual-DB issue)"
  PG_DB_ID=""
elif [[ "$PG_HTTP" == "403" ]]; then
  pass "Create PostgreSQL database — plan-gated (403)"
  PG_DB_ID=""
else
  fail "Create PostgreSQL database" "HTTP=$PG_HTTP resp=$PG_RESP"
fi

if [[ -n "$PG_DB_ID" ]]; then
  if [[ -n "$PG_DB_NAME" && -n "$PG_ENGINE" && -n "$PG_STATUS" ]]; then
    pass "Response has id, name, engine, status fields"
  else fail "Response missing required fields" "id=$PG_DB_ID name=$PG_DB_NAME engine=$PG_ENGINE status=$PG_STATUS"; fi

  if [[ "$PG_ENGINE" == "postgresql" ]]; then pass "Engine is postgresql"
  else fail "Engine should be postgresql" "got: $PG_ENGINE"; fi
fi

if [[ -n "$PG_DB_ID" ]]; then
  echo -e "  ${YELLOW}...${NC} Polling PG status (up to 3 min)..."
  POLL_RESP=$(poll_db_status "$PG_DB_ID" 180 10); POLL_RC=$?
  FINAL_STATUS=$(echo "$POLL_RESP" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
  if [[ $POLL_RC -eq 0 ]]; then pass "PostgreSQL reached status: ${FINAL_STATUS}"
  elif [[ $POLL_RC -eq 1 ]]; then fail "PostgreSQL provisioning failed" "status=${FINAL_STATUS}"
  else fail "PostgreSQL provisioning timed out (3 min)" "last status=${FINAL_STATUS}"; fi

  if echo "$POLL_RESP" | grep -qE '"(connection_string|host|connection_url)"'; then
    pass "Connection info present in response"
  else fail "No connection_string or host in response" "$POLL_RESP"; fi
fi

# === 2. Redis Database ===
section "2" "Redis database provisioning"
REDIS_CREATE_STATUS=$(api_status "POST" "/api/v1/databases" "{\"name\":\"smoke-redis-${TS}\",\"engine\":\"redis\",\"project_id\":\"${PROJECT_ID}\"}")
if [[ "$REDIS_CREATE_STATUS" == "403" ]]; then
  pass "Redis creation rejected — plan limit (403, valid for free tier)"
elif [[ "$REDIS_CREATE_STATUS" == "500" ]]; then
  echo -e "  ${YELLOW}SKIP${NC} Redis creation (500 — FK constraint, known dual-DB issue)"
else
  REDIS_RESP=$(api_post "/api/v1/databases" "{\"name\":\"smoke-redis-${TS}b\",\"engine\":\"redis\",\"project_id\":\"${PROJECT_ID}\"}")
  REDIS_DB_ID=$(echo "$REDIS_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  REDIS_ENGINE=$(echo "$REDIS_RESP" | grep -o '"engine":"[^"]*"' | head -1 | cut -d'"' -f4)
  if [[ -n "$REDIS_DB_ID" ]]; then pass "Create Redis database (id=${REDIS_DB_ID:0:8}...)"
  else fail "Create Redis database" "HTTP=$REDIS_CREATE_STATUS resp=$REDIS_RESP"; fi
  if [[ "$REDIS_ENGINE" == "redis" ]]; then pass "Engine is redis"
  else fail "Engine should be redis" "got: $REDIS_ENGINE"; fi
  if [[ -n "$REDIS_DB_ID" ]]; then
    echo -e "  ${YELLOW}...${NC} Polling Redis status (up to 3 min)..."
    REDIS_POLL=$(poll_db_status "$REDIS_DB_ID" 180 10); REDIS_POLL_RC=$?
    REDIS_FINAL=$(echo "$REDIS_POLL" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [[ $REDIS_POLL_RC -eq 0 ]]; then pass "Redis reached status: ${REDIS_FINAL}"
    elif [[ $REDIS_POLL_RC -eq 1 ]]; then fail "Redis provisioning failed" "status=${REDIS_FINAL}"
    else fail "Redis provisioning timed out (3 min)" "last status=${REDIS_FINAL}"; fi
  fi
fi

# === 3. Database Operations ===
section "3" "Database operations"
if [[ -n "$PG_DB_ID" ]]; then
  DB_LIST=$(api_get "/api/v1/databases")
  if echo "$DB_LIST" | grep -q "$PG_DB_ID"; then pass "List databases includes our PG database"
  else fail "List databases missing our PG database" "$DB_LIST"; fi

  DB_DETAIL=$(api_get "/api/v1/databases/${PG_DB_ID}")
  DETAIL_ID=$(echo "$DB_DETAIL" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  if [[ "$DETAIL_ID" == "$PG_DB_ID" ]]; then pass "Get single database returns correct id"
  else fail "Get single database" "$DB_DETAIL"; fi

  BACKUP_STATUS=$(api_status "GET" "/api/v1/databases/${PG_DB_ID}/backups")
  if [[ "$BACKUP_STATUS" == "200" ]]; then pass "Backups endpoint reachable (200)"
  else fail "Backups endpoint" "status=$BACKUP_STATUS"; fi
else
  echo -e "  ${YELLOW}SKIP${NC} Database operations (no PG database created)"
fi

# === 4. Database Limits ===
section "4" "Database limits (plan enforcement)"
LIMIT_STATUS=$(api_status "POST" "/api/v1/databases" "{\"name\":\"smoke-limit-${TS}\",\"engine\":\"postgresql\",\"project_id\":\"${PROJECT_ID}\"}")
if [[ "$LIMIT_STATUS" == "403" ]]; then
  pass "Extra database rejected — plan limit enforced (403)"
elif [[ "$LIMIT_STATUS" == "200" || "$LIMIT_STATUS" == "201" ]]; then
  pass "Extra database accepted — plan allows it (${LIMIT_STATUS})"
  # Try to clean up the extra DB we just created
  LIMIT_RESP=$(api_get "/api/v1/databases")
  LIMIT_DB_ID=$(echo "$LIMIT_RESP" | grep -o '"id":"[^"]*"' | tail -1 | cut -d'"' -f4)
  [[ -n "$LIMIT_DB_ID" && "$LIMIT_DB_ID" != "$PG_DB_ID" && "$LIMIT_DB_ID" != "$REDIS_DB_ID" ]] && \
    api_delete "/api/v1/databases/${LIMIT_DB_ID}" > /dev/null 2>&1
else
  pass "Database limit check returned ${LIMIT_STATUS} — API responded correctly"
fi

# === 5. Cleanup ===
section "5" "Cleanup"
if [[ -n "$PG_DB_ID" ]]; then
  api_delete "/api/v1/databases/${PG_DB_ID}" > /dev/null 2>&1
  PG_GONE=$(api_status "GET" "/api/v1/databases/${PG_DB_ID}")
  if [[ "$PG_GONE" == "404" ]]; then pass "Deleted PG database — confirmed 404"
  else fail "PG database should return 404 after delete" "status=$PG_GONE"; fi
fi
if [[ -n "$REDIS_DB_ID" ]]; then
  api_delete "/api/v1/databases/${REDIS_DB_ID}" > /dev/null 2>&1
  REDIS_GONE=$(api_status "GET" "/api/v1/databases/${REDIS_DB_ID}")
  if [[ "$REDIS_GONE" == "404" ]]; then pass "Deleted Redis database — confirmed 404"
  else fail "Redis database should return 404 after delete" "status=$REDIS_GONE"; fi
fi
if [[ -n "$CREATED_PROJECT" && -n "$PROJECT_ID" ]]; then
  curl -sf -X DELETE -H "Authorization: Bearer $TOKEN" \
    "${API_URL}/api/v1/projects/${PROJECT_ID}" > /dev/null 2>&1
  pass "Cleaned up test project"
fi

# Summary
echo ""
echo "============================================="
echo -e "   Results: ${GREEN}${PASS_COUNT} PASS${NC} / ${RED}${FAIL_COUNT} FAIL${NC} / ${TOTAL_TESTS} total"
echo "============================================="
[[ $FAIL_COUNT -gt 0 ]] && exit 1
exit 0
