#!/bin/bash
# =============================================================================
# Zenith Platform Deployment Script
# Runs on the server (ghasi) at /opt/zenith
#
# Usage: ./scripts/deploy.sh [--skip-build] [--skip-pull]
#
# Options:
#   --skip-build    Skip Docker image builds (use existing images)
#   --skip-pull     Skip git pull (deploy current code)
# =============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_step()  { echo -e "\n${BLUE}==>${NC} ${1}"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    ${1}"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  ${1}"; }
log_error() { echo -e "${RED}[ERROR]${NC} ${1}"; }

SKIP_BUILD=false
SKIP_PULL=false

for arg in "$@"; do
  case "$arg" in
    --skip-build) SKIP_BUILD=true ;;
    --skip-pull)  SKIP_PULL=true ;;
    *)            echo "Unknown option: $arg"; exit 1 ;;
  esac
done

cd /opt/zenith

echo ""
echo "============================================="
echo "   Zenith Platform Deployment"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "============================================="

# -------------------------------------------------------
# Step 1: Pull latest code
# -------------------------------------------------------
if [[ "$SKIP_PULL" == "false" ]]; then
  log_step "Pulling latest code from origin/main..."
  git pull origin main
  log_ok "Code updated"
else
  log_warn "Skipping git pull (--skip-pull)"
fi

# -------------------------------------------------------
# Step 2: Build Docker images
# -------------------------------------------------------
if [[ "$SKIP_BUILD" == "false" ]]; then
  log_step "Building Docker images..."

  echo "  Building zenith-landing..."
  docker build -t zenith-landing:latest -f apps/landing/Dockerfile . --quiet
  log_ok "zenith-landing:latest built"

  echo "  Building zenith-mc..."
  docker build -t zenith-mc:latest -f apps/mission-control/Dockerfile . --quiet
  log_ok "zenith-mc:latest built"

  echo "  Building zenith-web..."
  docker build -t zenith-web:latest -f apps/web/Dockerfile . --quiet
  log_ok "zenith-web:latest built"

  echo "  Building zenith-api..."
  docker build -t zenith-api:latest -f services/api/Dockerfile . --quiet
  log_ok "zenith-api:latest built"
else
  log_warn "Skipping Docker builds (--skip-build)"
fi

# -------------------------------------------------------
# Step 3: Import images into k3s containerd (if using k3s)
# -------------------------------------------------------
log_step "Importing Docker images into k3s..."

for img in zenith-landing zenith-mc zenith-web zenith-api; do
  if docker image inspect "${img}:latest" > /dev/null 2>&1; then
    docker save "${img}:latest" | sudo k3s ctr images import -
    log_ok "Imported ${img}:latest into k3s"
  else
    log_warn "Image ${img}:latest not found, skipping import"
  fi
done

# -------------------------------------------------------
# Step 4: Apply Kubernetes manifests
# -------------------------------------------------------
log_step "Applying Kubernetes manifests..."

kubectl apply -f k8s/namespace.yaml
log_ok "Namespaces created/updated"

kubectl apply -f k8s/landing.yaml
log_ok "Landing deployment applied"

kubectl apply -f k8s/mission-control.yaml
log_ok "Mission Control deployment applied"

kubectl apply -f k8s/api.yaml
log_ok "API deployment applied"

kubectl apply -f k8s/web.yaml
log_ok "Web (embermind) deployment applied"

kubectl apply -f k8s/certificates.yaml
log_ok "TLS certificates applied"

kubectl apply -f k8s/ingress.yaml
log_ok "Ingress routes applied"

# -------------------------------------------------------
# Step 5: Restart deployments to pick up new images
# -------------------------------------------------------
log_step "Restarting deployments..."

kubectl rollout restart deployment/zenith-landing -n zenith-platform
kubectl rollout restart deployment/zenith-mc -n zenith-platform
kubectl rollout restart deployment/zenith-api -n zenith-platform
kubectl rollout restart deployment/zenith-web -n zenith-embermind
log_ok "All deployments restarted"

# -------------------------------------------------------
# Step 6: Wait for rollouts to complete
# -------------------------------------------------------
log_step "Waiting for rollouts to complete..."

echo "  Waiting for zenith-landing..."
kubectl rollout status deployment/zenith-landing -n zenith-platform --timeout=120s
log_ok "zenith-landing is ready"

echo "  Waiting for zenith-mc..."
kubectl rollout status deployment/zenith-mc -n zenith-platform --timeout=120s
log_ok "zenith-mc is ready"

echo "  Waiting for zenith-api..."
kubectl rollout status deployment/zenith-api -n zenith-platform --timeout=120s
log_ok "zenith-api is ready"

echo "  Waiting for zenith-web..."
kubectl rollout status deployment/zenith-web -n zenith-embermind --timeout=120s
log_ok "zenith-web is ready"

# -------------------------------------------------------
# Step 7: Show deployment status
# -------------------------------------------------------
log_step "Deployment status:"

echo ""
echo "--- zenith-platform namespace ---"
kubectl get pods -n zenith-platform -o wide
echo ""
kubectl get svc -n zenith-platform
echo ""
kubectl get certificate -n zenith-platform
echo ""

echo "--- zenith-embermind namespace ---"
kubectl get pods -n zenith-embermind -o wide
echo ""
kubectl get svc -n zenith-embermind
echo ""
kubectl get certificate -n zenith-embermind
echo ""

echo "--- IngressRoutes ---"
kubectl get ingressroute -n zenith-platform
kubectl get ingressroute -n zenith-embermind
echo ""

echo "============================================="
echo "   Deployment complete!"
echo "   $(date '+%Y-%m-%d %H:%M:%S')"
echo "============================================="
echo ""
echo "Endpoints:"
echo "  Landing:          https://freezenith.com"
echo "  Mission Control:  https://mission.freezenith.com"
echo "  API:              https://api.freezenith.com/health"
echo "  Embermind Web:    https://embermind.app"
echo ""
