#!/usr/bin/env bash
set -euo pipefail

# Zenith deployed versions
# Usage: ./scripts/versions.sh [stage|prod|all]

STAGE_HOST="zen-stage"
STAGE_NS="zenith-staging"
PROD_HOST="zen-prod"
PROD_NS="zenith-production"

BOLD='\033[1m'
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
DIM='\033[2m'
RESET='\033[0m'

target="${1:-all}"

fetch_versions() {
  local host="$1" ns="$2"
  ssh -o ConnectTimeout=5 "$host" \
    "kubectl -n $ns get deploy -o custom-columns='NAME:.metadata.name,IMAGE:.spec.template.spec.containers[0].image,READY:.status.readyReplicas,DESIRED:.status.replicas' --no-headers" 2>/dev/null
}

print_env() {
  local env_name="$1" host="$2" ns="$3" color="$4"

  echo ""
  echo -e "${BOLD}${color}━━━ ${env_name} ━━━${RESET}"
  echo -e "${DIM}host: ${host} | namespace: ${ns}${RESET}"
  echo ""

  local output
  if ! output=$(fetch_versions "$host" "$ns" 2>&1); then
    echo -e "  ${RED}unreachable${RESET} (SSH failed or namespace not found)"
    return 1
  fi

  if [ -z "$output" ]; then
    echo -e "  ${YELLOW}no deployments found${RESET}"
    return 0
  fi

  printf "  ${DIM}%-22s %-12s %-10s${RESET}\n" "APP" "VERSION" "STATUS"
  printf "  ${DIM}%-22s %-12s %-10s${RESET}\n" "───────────────────" "─────────" "────────"

  while IFS= read -r line; do
    name=$(echo "$line" | awk '{print $1}')
    image=$(echo "$line" | awk '{print $2}')
    ready=$(echo "$line" | awk '{print $3}')
    desired=$(echo "$line" | awk '{print $4}')

    # Extract tag from image
    tag="${image##*:}"

    # Status
    if [ "$ready" = "$desired" ] && [ "$ready" != "<none>" ]; then
      status="${GREEN}● ${ready}/${desired}${RESET}"
    elif [ "$ready" = "<none>" ] || [ "$ready" = "0" ]; then
      status="${RED}○ 0/${desired}${RESET}"
    else
      status="${YELLOW}◐ ${ready}/${desired}${RESET}"
    fi

    printf "  %-22s %-12s %b\n" "$name" "$tag" "$status"
  done <<< "$output"
}

echo -e "${BOLD}${CYAN}"
echo "  ╔══════════════════════════════════╗"
echo "  ║       Zenith Deploy Status       ║"
echo "  ╚══════════════════════════════════╝"
echo -e "${RESET}"

if [ "$target" = "stage" ] || [ "$target" = "all" ]; then
  print_env "STAGING" "$STAGE_HOST" "$STAGE_NS" "$CYAN"
fi

if [ "$target" = "prod" ] || [ "$target" = "all" ]; then
  print_env "PRODUCTION" "$PROD_HOST" "$PROD_NS" "$GREEN" || true
fi

echo ""
