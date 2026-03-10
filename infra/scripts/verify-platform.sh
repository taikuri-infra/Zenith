#!/bin/bash
# =============================================================================
# Zenith Platform Verification Script
#
# Verifies:
#   P1-13: Observability stack (Grafana → Loki, Tempo, Prometheus)
#   P2-06: OAuth / OIDC providers (Google, GitHub, Keycloak)
#
# Usage:
#   ./infra/scripts/verify-platform.sh [--api-url URL] [--grafana-url URL]
#
# Environment variables:
#   ZENITH_API_URL      API base URL (default: https://api.stage.freezenith.com)
#   GRAFANA_URL         Grafana base URL (default: https://grafana.stage.freezenith.com)
#   GRAFANA_TOKEN       Grafana service account token (optional, for datasource queries)
#   KEYCLOAK_URL        Keycloak base URL (default: https://auth.stage.freezenith.com)
#
# Returns exit 0 if all checks pass, exit 1 if any fail.
# =============================================================================

set -uo pipefail

API_URL="${ZENITH_API_URL:-https://api.stage.freezenith.com}"
GRAFANA_URL="${GRAFANA_URL:-https://grafana.stage.freezenith.com}"
GRAFANA_TOKEN="${GRAFANA_TOKEN:-}"
KEYCLOAK_URL="${KEYCLOAK_URL:-https://auth.stage.freezenith.com}"
KEYCLOAK_REALM="${KEYCLOAK_REALM:-zenith}"

for arg in "$@"; do
  case "$arg" in
    --api-url=*) API_URL="${arg#*=}" ;;
    --grafana-url=*) GRAFANA_URL="${arg#*=}" ;;
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
  if [[ -n "${2:-}" ]]; then
    echo -e "       ${YELLOW}Detail: ${2}${NC}"
  fi
}

section() {
  echo ""
  echo -e "${BLUE}[$1]${NC} $2"
}

echo ""
echo "============================================="
echo "   Zenith Platform Verification"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "   API:      ${API_URL}"
echo "   Grafana:  ${GRAFANA_URL}"
echo "   Keycloak: ${KEYCLOAK_URL}"
echo "============================================="

# =============================================
# P1-13: Observability Stack
# =============================================
section "P1-13" "Observability stack verification"

# --- Grafana health ---
GRAFANA_HEALTH=$(curl -sf "${GRAFANA_URL}/api/health" 2>/dev/null)
if echo "$GRAFANA_HEALTH" | grep -qi '"database":"ok"'; then
  pass "Grafana health check — database OK"
else
  fail "Grafana health check" "$GRAFANA_HEALTH"
fi

# Build auth header for Grafana API
GRAFANA_AUTH=""
if [[ -n "$GRAFANA_TOKEN" ]]; then
  GRAFANA_AUTH="-H \"Authorization: Bearer $GRAFANA_TOKEN\""
fi

# --- Loki datasource ---
LOKI_DS=$(curl -sf ${GRAFANA_AUTH:+-H "Authorization: Bearer $GRAFANA_TOKEN"} \
  "${GRAFANA_URL}/api/datasources" 2>/dev/null)
if echo "$LOKI_DS" | grep -qi '"type":"loki"'; then
  pass "Grafana has Loki datasource configured"

  # Query Loki for recent logs
  LOKI_QUERY=$(curl -sf ${GRAFANA_AUTH:+-H "Authorization: Bearer $GRAFANA_TOKEN"} \
    "${GRAFANA_URL}/api/datasources/proxy/uid/loki/loki/api/v1/query?query=%7Bnamespace%3D%22zenith-staging%22%7D&limit=1" 2>/dev/null)
  if echo "$LOKI_QUERY" | grep -q '"result"'; then
    pass "Loki returns log data for zenith-staging"
  else
    fail "Loki query returned no data" "Query may need namespace adjustment"
  fi
else
  fail "Loki datasource not found in Grafana" "$LOKI_DS"
fi

# --- Tempo datasource ---
if echo "$LOKI_DS" | grep -qi '"type":"tempo"'; then
  pass "Grafana has Tempo datasource configured"
else
  fail "Tempo datasource not found in Grafana"
fi

# --- Prometheus datasource + scraping ---
if echo "$LOKI_DS" | grep -qi '"type":"prometheus"'; then
  pass "Grafana has Prometheus datasource configured"

  # Query Prometheus for 'up' metric
  PROM_UP=$(curl -sf ${GRAFANA_AUTH:+-H "Authorization: Bearer $GRAFANA_TOKEN"} \
    "${GRAFANA_URL}/api/datasources/proxy/uid/prometheus/api/v1/query?query=up" 2>/dev/null)
  if echo "$PROM_UP" | grep -q '"result"'; then
    UP_COUNT=$(echo "$PROM_UP" | grep -o '"value"' | wc -l)
    if [[ "$UP_COUNT" -gt 0 ]]; then
      pass "Prometheus 'up' metric has ${UP_COUNT} targets"
    else
      fail "Prometheus 'up' returned 0 targets"
    fi
  else
    fail "Prometheus query failed" "$PROM_UP"
  fi
else
  fail "Prometheus datasource not found in Grafana"
fi

# =============================================
# P2-06: OAuth / OIDC Providers
# =============================================
section "P2-06" "OAuth / OIDC provider verification"

# --- Google OAuth redirect ---
GOOGLE_STATUS=$(curl -so /dev/null -w "%{http_code}" -L --max-redirs 0 \
  "${API_URL}/api/v1/auth/oauth/google" 2>/dev/null)
if [[ "$GOOGLE_STATUS" == "302" || "$GOOGLE_STATUS" == "307" ]]; then
  pass "Google OAuth redirects (status: $GOOGLE_STATUS)"
else
  # API might return a JSON url instead of redirect
  GOOGLE_RESP=$(curl -sf "${API_URL}/api/v1/auth/oauth/google" 2>/dev/null)
  if echo "$GOOGLE_RESP" | grep -qi 'accounts.google.com\|url'; then
    pass "Google OAuth returns redirect URL"
  else
    fail "Google OAuth endpoint (status: $GOOGLE_STATUS)" "$GOOGLE_RESP"
  fi
fi

# --- GitHub OAuth redirect ---
GITHUB_STATUS=$(curl -so /dev/null -w "%{http_code}" -L --max-redirs 0 \
  "${API_URL}/api/v1/auth/oauth/github" 2>/dev/null)
if [[ "$GITHUB_STATUS" == "302" || "$GITHUB_STATUS" == "307" ]]; then
  pass "GitHub OAuth redirects (status: $GITHUB_STATUS)"
else
  GITHUB_RESP=$(curl -sf "${API_URL}/api/v1/auth/oauth/github" 2>/dev/null)
  if echo "$GITHUB_RESP" | grep -qi 'github.com\|url'; then
    pass "GitHub OAuth returns redirect URL"
  else
    fail "GitHub OAuth endpoint (status: $GITHUB_STATUS)" "$GITHUB_RESP"
  fi
fi

# --- Keycloak OIDC discovery ---
KC_DISCOVERY=$(curl -sf \
  "${KEYCLOAK_URL}/realms/${KEYCLOAK_REALM}/.well-known/openid-configuration" 2>/dev/null)
if echo "$KC_DISCOVERY" | grep -q '"issuer"'; then
  pass "Keycloak OIDC discovery endpoint responds"
  if echo "$KC_DISCOVERY" | grep -q '"authorization_endpoint"'; then
    pass "Keycloak OIDC has authorization_endpoint"
  else
    fail "Keycloak OIDC missing authorization_endpoint"
  fi
  if echo "$KC_DISCOVERY" | grep -q '"token_endpoint"'; then
    pass "Keycloak OIDC has token_endpoint"
  else
    fail "Keycloak OIDC missing token_endpoint"
  fi
else
  fail "Keycloak OIDC discovery failed" "$KC_DISCOVERY"
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
  echo -e "${RED}VERIFICATION FAILED${NC} — $FAIL_COUNT check(s) did not pass."
  exit 1
else
  echo -e "${GREEN}ALL PLATFORM CHECKS PASSED${NC}"
  exit 0
fi
