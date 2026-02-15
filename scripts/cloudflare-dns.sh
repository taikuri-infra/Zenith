#!/bin/bash
# =============================================================================
# Cloudflare DNS Record Management for Zenith Platform
# Usage: ./scripts/cloudflare-dns.sh [create|delete|status]
#
# Creates/deletes DNS A records for all Zenith domains.
# Idempotent: checks if records exist before creating.
# Proxied is OFF so cert-manager can issue Let's Encrypt certificates.
# =============================================================================

set -euo pipefail

CF_TOKEN="ximk5-d_hldQ42I2eUO9sM7ghsq2nv945KRjhFvO"
SERVER_IP="161.35.82.211"

# Zone IDs
FREEZENITH_ZONE="37ac5735b1cf9099ccedd4e038d99465"
EMBERMIND_ZONE="3a9f6cca73ea2653b2bdef6cc6a203b8"

CF_API="https://api.cloudflare.com/client/v4"

ACTION="${1:-status}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${BLUE}[INFO]${NC}  $1"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Define all DNS records: "zone_id name type content proxied"
RECORDS=(
  "${FREEZENITH_ZONE} freezenith.com A ${SERVER_IP} false"
  "${FREEZENITH_ZONE} www.freezenith.com A ${SERVER_IP} false"
  "${FREEZENITH_ZONE} api.freezenith.com A ${SERVER_IP} false"
  "${FREEZENITH_ZONE} demo-ms.freezenith.com A ${SERVER_IP} false"
  "${FREEZENITH_ZONE} demo-cloud.freezenith.com A ${SERVER_IP} false"
  "${EMBERMIND_ZONE} ms.embermind.app A ${SERVER_IP} false"
  "${EMBERMIND_ZONE} cloud.embermind.app A ${SERVER_IP} false"
)

# Get existing record ID for a given zone, name, and type
get_record_id() {
  local zone_id="$1"
  local name="$2"
  local type="$3"

  local response
  response=$(curl -s -X GET \
    "${CF_API}/zones/${zone_id}/dns_records?type=${type}&name=${name}" \
    -H "Authorization: Bearer ${CF_TOKEN}" \
    -H "Content-Type: application/json")

  local record_id
  record_id=$(echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
results = data.get('result', [])
if results:
    print(results[0]['id'])
else:
    print('')
" 2>/dev/null || echo "")

  echo "$record_id"
}

# Create a DNS record
create_record() {
  local zone_id="$1"
  local name="$2"
  local type="$3"
  local content="$4"
  local proxied="$5"

  # Check if record already exists
  local existing_id
  existing_id=$(get_record_id "$zone_id" "$name" "$type")

  if [[ -n "$existing_id" ]]; then
    # Update existing record
    log_warn "Record ${name} (${type}) already exists (id: ${existing_id}), updating..."
    local response
    response=$(curl -s -X PUT \
      "${CF_API}/zones/${zone_id}/dns_records/${existing_id}" \
      -H "Authorization: Bearer ${CF_TOKEN}" \
      -H "Content-Type: application/json" \
      --data "{
        \"type\": \"${type}\",
        \"name\": \"${name}\",
        \"content\": \"${content}\",
        \"proxied\": ${proxied},
        \"ttl\": 1
      }")

    local success
    success=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('success', False))" 2>/dev/null)

    if [[ "$success" == "True" ]]; then
      log_ok "Updated ${name} -> ${content}"
    else
      log_error "Failed to update ${name}: $(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('errors', []))" 2>/dev/null)"
      return 1
    fi
  else
    # Create new record
    log_info "Creating ${name} (${type}) -> ${content}..."
    local response
    response=$(curl -s -X POST \
      "${CF_API}/zones/${zone_id}/dns_records" \
      -H "Authorization: Bearer ${CF_TOKEN}" \
      -H "Content-Type: application/json" \
      --data "{
        \"type\": \"${type}\",
        \"name\": \"${name}\",
        \"content\": \"${content}\",
        \"proxied\": ${proxied},
        \"ttl\": 1
      }")

    local success
    success=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('success', False))" 2>/dev/null)

    if [[ "$success" == "True" ]]; then
      log_ok "Created ${name} -> ${content}"
    else
      log_error "Failed to create ${name}: $(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('errors', []))" 2>/dev/null)"
      return 1
    fi
  fi
}

# Delete a DNS record
delete_record() {
  local zone_id="$1"
  local name="$2"
  local type="$3"

  local existing_id
  existing_id=$(get_record_id "$zone_id" "$name" "$type")

  if [[ -z "$existing_id" ]]; then
    log_warn "Record ${name} (${type}) does not exist, skipping delete."
    return 0
  fi

  log_info "Deleting ${name} (${type}) (id: ${existing_id})..."
  local response
  response=$(curl -s -X DELETE \
    "${CF_API}/zones/${zone_id}/dns_records/${existing_id}" \
    -H "Authorization: Bearer ${CF_TOKEN}" \
    -H "Content-Type: application/json")

  local success
  success=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('success', False))" 2>/dev/null)

  if [[ "$success" == "True" ]]; then
    log_ok "Deleted ${name}"
  else
    log_error "Failed to delete ${name}: $(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('errors', []))" 2>/dev/null)"
    return 1
  fi
}

# Show status of all records
show_status() {
  echo ""
  echo "=== Zenith DNS Record Status ==="
  echo ""

  for record in "${RECORDS[@]}"; do
    read -r zone_id name type content proxied <<< "$record"

    local existing_id
    existing_id=$(get_record_id "$zone_id" "$name" "$type")

    if [[ -n "$existing_id" ]]; then
      log_ok "${name} (${type}) -> exists (id: ${existing_id})"
    else
      log_warn "${name} (${type}) -> NOT FOUND"
    fi
  done

  echo ""
}

# Main
case "$ACTION" in
  create)
    echo ""
    echo "=== Creating Zenith DNS Records ==="
    echo "Target IP: ${SERVER_IP}"
    echo ""

    fail_count=0
    for record in "${RECORDS[@]}"; do
      read -r zone_id name type content proxied <<< "$record"
      if ! create_record "$zone_id" "$name" "$type" "$content" "$proxied"; then
        ((fail_count++))
      fi
    done

    echo ""
    if [[ $fail_count -eq 0 ]]; then
      log_ok "All DNS records created/updated successfully!"
    else
      log_error "${fail_count} record(s) failed."
      exit 1
    fi
    ;;

  delete)
    echo ""
    echo "=== Deleting Zenith DNS Records ==="
    echo ""

    read -p "Are you sure you want to delete all Zenith DNS records? (y/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      log_info "Aborted."
      exit 0
    fi

    fail_count=0
    for record in "${RECORDS[@]}"; do
      read -r zone_id name type content proxied <<< "$record"
      if ! delete_record "$zone_id" "$name" "$type"; then
        ((fail_count++))
      fi
    done

    echo ""
    if [[ $fail_count -eq 0 ]]; then
      log_ok "All DNS records deleted successfully!"
    else
      log_error "${fail_count} record(s) failed."
      exit 1
    fi
    ;;

  status)
    show_status
    ;;

  *)
    echo "Usage: $0 [create|delete|status]"
    echo ""
    echo "  create  - Create or update all DNS records"
    echo "  delete  - Delete all DNS records (with confirmation)"
    echo "  status  - Show current status of DNS records"
    exit 1
    ;;
esac
