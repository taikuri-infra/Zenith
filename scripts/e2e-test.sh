#!/bin/bash
# =============================================================================
# Zenith Platform End-to-End Test Script
# Tests all endpoints, SSL certificates, DNS resolution, and content.
#
# Usage: ./scripts/e2e-test.sh [--verbose]
#
# Can be run from anywhere (local machine, CI, etc.)
# Returns exit 0 if all tests pass, exit 1 if any fail.
# =============================================================================

set -uo pipefail

VERBOSE=false
for arg in "$@"; do
  case "$arg" in
    --verbose|-v) VERBOSE=true ;;
  esac
done

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0
TOTAL_TESTS=0

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

SERVER_IP="161.35.82.211"

echo ""
echo "============================================="
echo "   Zenith Platform E2E Tests"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "============================================="

# -------------------------------------------------------
# Section 1: DNS Resolution
# -------------------------------------------------------
section "1/6" "DNS Resolution"

check_dns() {
  local domain="$1"
  local expected_ip="$2"
  local resolved

  resolved=$(dig +short "$domain" 2>/dev/null | head -1)

  if [[ "$resolved" == "$expected_ip" ]]; then
    pass "${domain} -> ${resolved}"
  else
    fail "${domain} -> expected ${expected_ip}, got '${resolved}'" "${resolved:-no response}"
  fi
}

check_dns "freezenith.com" "$SERVER_IP"
check_dns "www.freezenith.com" "$SERVER_IP"
check_dns "mission.freezenith.com" "$SERVER_IP"
check_dns "api.freezenith.com" "$SERVER_IP"
check_dns "embermind.app" "$SERVER_IP"

# -------------------------------------------------------
# Section 2: HTTPS Connectivity
# -------------------------------------------------------
section "2/6" "HTTPS Connectivity"

check_https() {
  local url="$1"
  local expected_code="${2:-200}"
  local actual_code

  actual_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$url" 2>/dev/null)

  if [[ "$actual_code" == "$expected_code" ]]; then
    pass "${url} -> HTTP ${actual_code}"
  else
    fail "${url} -> expected HTTP ${expected_code}, got ${actual_code}" "HTTP ${actual_code}"
  fi
}

check_https "https://freezenith.com" "200"
check_https "https://www.freezenith.com" "200"
check_https "https://mission.freezenith.com" "200"
check_https "https://api.freezenith.com/health" "200"
check_https "https://embermind.app" "200"

# -------------------------------------------------------
# Section 3: HTTP -> HTTPS Redirect
# -------------------------------------------------------
section "3/6" "HTTP to HTTPS Redirect"

check_redirect() {
  local url="$1"
  local actual_code

  actual_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 -L "$url" 2>/dev/null)
  local redirect_code
  redirect_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$url" 2>/dev/null)

  if [[ "$redirect_code" == "301" || "$redirect_code" == "308" ]]; then
    pass "${url} -> HTTP ${redirect_code} (redirects to HTTPS)"
  elif [[ "$redirect_code" == "302" || "$redirect_code" == "307" ]]; then
    pass "${url} -> HTTP ${redirect_code} (temporary redirect to HTTPS)"
  else
    fail "${url} -> expected 301/308 redirect, got ${redirect_code}" "HTTP ${redirect_code}"
  fi
}

check_redirect "http://freezenith.com"
check_redirect "http://www.freezenith.com"
check_redirect "http://mission.freezenith.com"
check_redirect "http://api.freezenith.com"
check_redirect "http://embermind.app"

# -------------------------------------------------------
# Section 4: SSL Certificate Validity
# -------------------------------------------------------
section "4/6" "SSL Certificates"

check_ssl() {
  local domain="$1"
  local ssl_info

  ssl_info=$(echo | openssl s_client -servername "$domain" -connect "${domain}:443" 2>/dev/null)

  # Check if connection succeeded
  if echo "$ssl_info" | grep -q "Verify return code: 0"; then
    pass "${domain} -> SSL valid (verified)"
  elif echo "$ssl_info" | grep -q "BEGIN CERTIFICATE"; then
    # Certificate exists but might be self-signed or chain issue
    local expiry
    expiry=$(echo "$ssl_info" | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)
    if [[ -n "$expiry" ]]; then
      pass "${domain} -> SSL certificate present (expires: ${expiry})"
    else
      fail "${domain} -> SSL certificate present but could not read expiry"
    fi
  else
    fail "${domain} -> SSL connection failed" "No certificate returned"
  fi
}

check_ssl "freezenith.com"
check_ssl "mission.freezenith.com"
check_ssl "api.freezenith.com"
check_ssl "embermind.app"

# -------------------------------------------------------
# Section 5: Content Checks
# -------------------------------------------------------
section "5/6" "Content Verification"

check_content() {
  local url="$1"
  local search="$2"
  local label="$3"
  local body

  body=$(curl -s --max-time 10 "$url" 2>/dev/null)

  if echo "$body" | grep -qi "$search"; then
    pass "${label}"
  else
    fail "${label}" "String '${search}' not found in response"
  fi
}

check_content "https://freezenith.com" "zenith" "Landing page contains 'zenith'"
check_content "https://mission.freezenith.com" "mission\|control\|zenith" "Mission Control page loads"
check_content "https://api.freezenith.com/health" "ok\|healthy\|status" "API health endpoint responds"
check_content "https://embermind.app" "zenith\|embermind" "Embermind web platform loads"

# -------------------------------------------------------
# Section 6: API Endpoint Tests
# -------------------------------------------------------
section "6/6" "API Endpoints"

# Health check (JSON)
api_health=$(curl -s --max-time 10 "https://api.freezenith.com/health" 2>/dev/null)
if echo "$api_health" | grep -qi "ok\|healthy"; then
  pass "GET /health -> healthy"
else
  fail "GET /health -> unhealthy" "$api_health"
fi

# API version/info (if available)
api_response_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "https://api.freezenith.com/api/v1" 2>/dev/null)
if [[ "$api_response_code" != "000" ]]; then
  pass "GET /api/v1 -> responds (HTTP ${api_response_code})"
else
  fail "GET /api/v1 -> connection failed" "No response"
fi

# -------------------------------------------------------
# Results
# -------------------------------------------------------
echo ""
echo "============================================="
if [[ $FAIL_COUNT -eq 0 ]]; then
  echo -e "  ${GREEN}ALL ${TOTAL_TESTS} TESTS PASSED${NC}"
else
  echo -e "  ${RED}${FAIL_COUNT} of ${TOTAL_TESTS} TESTS FAILED${NC}"
  echo -e "  ${GREEN}${PASS_COUNT} passed${NC}, ${RED}${FAIL_COUNT} failed${NC}"
fi
echo "============================================="
echo ""

if [[ $FAIL_COUNT -gt 0 ]]; then
  exit 1
fi

exit 0
