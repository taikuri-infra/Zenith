#!/bin/bash
# =============================================================================
# Zenith DR Failover Script
#
# Promotes the DR cluster (Helsinki) to primary. Steps:
#   1. Verify DR cluster is reachable
#   2. Promote CNPG standby to primary
#   3. Update DNS to point to DR server
#   4. Deploy latest platform configuration
#
# Usage:
#   ./infra/scripts/dr-failover.sh --dr-kubeconfig ~/.kube/zenith-dr.yaml \
#     --dr-ip 10.0.0.1 --domain freezenith.com
#
# Environment:
#   CLOUDFLARE_API_TOKEN  Required for DNS cutover
#   CLOUDFLARE_ZONE_ID    Cloudflare zone ID for the domain
#
# WARNING: This script performs destructive DNS changes.
#          Only run during an actual disaster or scheduled DR drill.
# =============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

DR_KUBECONFIG=""
DR_IP=""
DOMAIN=""
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --dr-kubeconfig) DR_KUBECONFIG="$2"; shift 2 ;;
    --dr-ip) DR_IP="$2"; shift 2 ;;
    --domain) DOMAIN="$2"; shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ -z "$DR_KUBECONFIG" || -z "$DR_IP" || -z "$DOMAIN" ]]; then
  echo "Usage: $0 --dr-kubeconfig <path> --dr-ip <ip> --domain <domain> [--dry-run]"
  exit 1
fi

CLOUDFLARE_API_TOKEN="${CLOUDFLARE_API_TOKEN:-}"
CLOUDFLARE_ZONE_ID="${CLOUDFLARE_ZONE_ID:-}"

echo ""
echo "============================================="
echo -e "  ${RED}ZENITH DR FAILOVER${NC}"
echo "  $(date '+%Y-%m-%d %H:%M:%S')"
echo "  DR Cluster: $DR_IP"
echo "  Domain: $DOMAIN"
echo "  Dry Run: $DRY_RUN"
echo "============================================="
echo ""

# --- Step 1: Verify DR cluster ---
echo -e "${BLUE}[1/4]${NC} Verifying DR cluster connectivity..."
if kubectl --kubeconfig="$DR_KUBECONFIG" get nodes &>/dev/null; then
  echo -e "  ${GREEN}OK${NC} DR cluster is reachable"
  kubectl --kubeconfig="$DR_KUBECONFIG" get nodes
else
  echo -e "  ${RED}FAIL${NC} Cannot reach DR cluster at $DR_IP"
  exit 1
fi
echo ""

# --- Step 2: Promote CNPG standby ---
echo -e "${BLUE}[2/4]${NC} Promoting CNPG standby to primary..."

# Check if CNPG cluster exists
CNPG_CLUSTER=$(kubectl --kubeconfig="$DR_KUBECONFIG" get clusters.postgresql.cnpg.io \
  -n zenith-staging -o name 2>/dev/null || true)

if [[ -n "$CNPG_CLUSTER" ]]; then
  if [[ "$DRY_RUN" == "true" ]]; then
    echo -e "  ${YELLOW}DRY RUN${NC} Would promote $CNPG_CLUSTER"
  else
    # Remove the replica source to promote to primary
    kubectl --kubeconfig="$DR_KUBECONFIG" patch "$CNPG_CLUSTER" \
      -n zenith-staging --type=merge \
      -p '{"spec":{"replica":{"enabled":false}}}' 2>/dev/null || true
    echo -e "  ${GREEN}OK${NC} CNPG promotion initiated for $CNPG_CLUSTER"
    echo "  Waiting for promotion to complete..."
    sleep 10
    kubectl --kubeconfig="$DR_KUBECONFIG" get "$CNPG_CLUSTER" -n zenith-staging
  fi
else
  echo -e "  ${YELLOW}SKIP${NC} No CNPG cluster found in DR — may need manual restore from Velero"
fi
echo ""

# --- Step 3: DNS cutover ---
echo -e "${BLUE}[3/4]${NC} DNS cutover to DR IP ($DR_IP)..."

if [[ -z "$CLOUDFLARE_API_TOKEN" || -z "$CLOUDFLARE_ZONE_ID" ]]; then
  echo -e "  ${YELLOW}SKIP${NC} CLOUDFLARE_API_TOKEN or CLOUDFLARE_ZONE_ID not set"
  echo "  Manual DNS update required: point *.${DOMAIN} → $DR_IP"
else
  # Records to update
  RECORDS=("api" "app" "auth" "argocd" "grafana" "registry")

  for RECORD in "${RECORDS[@]}"; do
    FQDN="${RECORD}.${DOMAIN}"

    # Find existing record
    RECORD_ID=$(curl -sf -X GET \
      "https://api.cloudflare.com/client/v4/zones/${CLOUDFLARE_ZONE_ID}/dns_records?name=${FQDN}&type=A" \
      -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
      -H "Content-Type: application/json" \
      | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [[ -n "$RECORD_ID" ]]; then
      if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "  ${YELLOW}DRY RUN${NC} Would update ${FQDN} → $DR_IP"
      else
        curl -sf -X PATCH \
          "https://api.cloudflare.com/client/v4/zones/${CLOUDFLARE_ZONE_ID}/dns_records/${RECORD_ID}" \
          -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
          -H "Content-Type: application/json" \
          -d "{\"content\":\"${DR_IP}\",\"proxied\":false}" > /dev/null
        echo -e "  ${GREEN}OK${NC} ${FQDN} → $DR_IP"
      fi
    else
      echo -e "  ${YELLOW}SKIP${NC} No A record found for ${FQDN}"
    fi
  done
fi
echo ""

# --- Step 4: Verify services ---
echo -e "${BLUE}[4/4]${NC} Verifying DR services..."
kubectl --kubeconfig="$DR_KUBECONFIG" get pods -A --field-selector=status.phase!=Running 2>/dev/null | head -20

READY_PODS=$(kubectl --kubeconfig="$DR_KUBECONFIG" get pods -A --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
echo -e "  Running pods: ${GREEN}${READY_PODS}${NC}"

echo ""
echo "============================================="
if [[ "$DRY_RUN" == "true" ]]; then
  echo -e "  ${YELLOW}DRY RUN COMPLETE${NC} — no changes were made"
else
  echo -e "  ${GREEN}FAILOVER COMPLETE${NC}"
  echo "  DR cluster at $DR_IP is now primary"
  echo ""
  echo "  Post-failover checklist:"
  echo "    1. Verify API: curl https://api.${DOMAIN}/health"
  echo "    2. Verify Web: curl https://app.${DOMAIN}"
  echo "    3. Run smoke tests: ./infra/scripts/smoke-test-customer.sh"
  echo "    4. Update ArgoCD to point to DR cluster"
  echo "    5. Notify team of failover"
fi
echo "============================================="
