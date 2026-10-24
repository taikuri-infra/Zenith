#!/usr/bin/env bash
set -euo pipefail

# Zenith on k3s — one-command installer
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/taikuri-infra/Zenith/main/infra/scripts/install-k3s.sh | sudo ZENITH_DOMAIN=zenith.example.com bash
#
# Installs k3s (if missing), Helm, cert-manager and CloudNativePG, then the
# zenith-selfhost chart. Safe to re-run: existing secrets are kept.
#
# Env: ZENITH_DOMAIN (default <public-ip>.sslip.io), ZENITH_EMAIL (Let's Encrypt),
#      ZENITH_VERSION (image tag, default latest), ZENITH_ADMIN_EMAIL, ZENITH_ADMIN_PASSWORD,
#      ZENITH_REPO_URL / ZENITH_BRANCH, ZENITH_CHART (local chart path, skips git),
#      ZENITH_DRY_RUN=1 (print the plan, change nothing).

REPO_URL="${ZENITH_REPO_URL:-https://github.com/taikuri-infra/Zenith.git}"
REPO_BRANCH="${ZENITH_BRANCH:-main}"
IMAGE_REPO="${ZENITH_IMAGE_REPO:-ghcr.io/taikuri-infra}"
VERSION="${ZENITH_VERSION:-latest}"
NS="${ZENITH_NAMESPACE:-zenith-system}"
DRY_RUN="${ZENITH_DRY_RUN:-0}"
export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"

log()  { printf '\033[36m==>\033[0m %s\n' "$1"; }
warn() { printf '\033[33m!\033[0m  %s\n' "$1"; }
err()  { printf '\033[31mError:\033[0m %s\n' "$1" >&2; }
run()  { if [ "$DRY_RUN" = 1 ]; then printf '[dry-run] %s\n' "$*"; else "$@"; fi; }

rand() { openssl rand -hex "${1:-16}"; }

resolve_domain() {
  [ -n "${ZENITH_DOMAIN:-}" ] && return 0
  local ip="${ZENITH_PUBLIC_IP:-}"
  [ -n "$ip" ] || ip="$(curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null || true)"
  if ! [[ "$ip" =~ ^[0-9]+(\.[0-9]+){3}$ ]]; then
    err "Could not detect the public IP. Set ZENITH_DOMAIN=<your domain> and re-run."
    exit 1
  fi
  ZENITH_DOMAIN="${ip}.sslip.io"
}

preflight() {
  [ "$DRY_RUN" = 1 ] && return 0
  [ "$(id -u)" = 0 ] || { err "Run as root (sudo)."; exit 1; }
  for c in curl openssl; do command -v "$c" >/dev/null || { err "$c is required."; exit 1; }; done
  if ss -ltn 2>/dev/null | grep -qE ':(80|443) ' && ! command -v k3s >/dev/null; then
    err "Ports 80/443 are in use; free them before installing (k3s Traefik needs both)."
    exit 1
  fi
}

install_k3s() {
  if command -v k3s >/dev/null && kubectl get nodes >/dev/null 2>&1; then log "k3s already running"; return; fi
  log "Installing k3s"
  if [ "$DRY_RUN" = 1 ]; then echo "[dry-run] curl -sfL https://get.k3s.io | sh -"; return; fi
  curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--write-kubeconfig-mode 0644" sh -
  for _ in $(seq 1 60); do kubectl get nodes 2>/dev/null | grep -q ' Ready' && return; sleep 3; done
  err "k3s did not become Ready"; exit 1
}

install_helm() {
  command -v helm >/dev/null && return
  log "Installing Helm"
  if [ "$DRY_RUN" = 1 ]; then echo "[dry-run] install helm"; return; fi
  local ver="${HELM_VERSION:-v3.17.1}" arch tmp
  case "$(uname -m)" in x86_64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) err "Unsupported CPU: $(uname -m)"; exit 1 ;; esac
  tmp="$(mktemp -d)"
  curl -fsSL -o "$tmp/helm.tgz" "https://get.helm.sh/helm-${ver}-linux-${arch}.tar.gz"
  curl -fsSL -o "$tmp/helm.sha256" "https://get.helm.sh/helm-${ver}-linux-${arch}.tar.gz.sha256sum"
  (cd "$tmp" && echo "$(awk '{print $1}' helm.sha256)  helm.tgz" | sha256sum -c -) || { err "Helm checksum mismatch"; exit 1; }
  tar -xzf "$tmp/helm.tgz" -C "$tmp"
  install -m 0755 "$tmp/linux-${arch}/helm" /usr/local/bin/helm
  rm -rf "$tmp"
}

install_operators() {
  log "Installing cert-manager and CloudNativePG"
  run helm repo add jetstack https://charts.jetstack.io --force-update
  run helm repo add cnpg https://cloudnative-pg.github.io/charts --force-update
  run helm repo update
  run helm upgrade --install cert-manager jetstack/cert-manager -n cert-manager --create-namespace \
    --set crds.enabled=true --wait --timeout 8m
  run helm upgrade --install cnpg cnpg/cloudnative-pg -n cnpg-system --create-namespace --wait --timeout 8m
}

ensure_secrets() {
  if [ "$DRY_RUN" = 1 ]; then echo "[dry-run] create secret zenith-secrets in $NS"; return; fi
  kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  if kubectl -n "$NS" get secret zenith-secrets >/dev/null 2>&1; then log "Keeping existing zenith-secrets"; return; fi
  ADMIN_EMAIL="${ZENITH_ADMIN_EMAIL:-admin@${ZENITH_DOMAIN}}"
  ADMIN_PASSWORD="${ZENITH_ADMIN_PASSWORD:-$(rand 12)}"
  kubectl -n "$NS" create secret generic zenith-secrets \
    --from-literal=jwt-secret="$(rand 32)" \
    --from-literal=admin-email="$ADMIN_EMAIL" \
    --from-literal=admin-password="$ADMIN_PASSWORD" \
    --from-literal=secrets-encryption-key="$(rand 32)" \
    --from-literal=s3-access-key=zenith \
    --from-literal=s3-secret-key="$(rand 16)" >/dev/null
  CREATED_ADMIN=1
}

fetch_chart() {
  if [ -n "${ZENITH_CHART:-}" ]; then CHART="$ZENITH_CHART"; return; fi
  CHART_DIR="$(mktemp -d)"
  log "Fetching chart ($REPO_BRANCH)"
  run git clone --depth 1 --branch "$REPO_BRANCH" "$REPO_URL" "$CHART_DIR"
  CHART="$CHART_DIR/infra/helm/zenith-selfhost"
}

install_zenith() {
  log "Installing Zenith ($VERSION) at https://$ZENITH_DOMAIN"
  run helm upgrade --install zenith "$CHART" -n "$NS" --wait --timeout 10m \
    --set domain="$ZENITH_DOMAIN" \
    --set tls.issuer.email="${ZENITH_EMAIL:-admin@${ZENITH_DOMAIN}}" \
    --set images.api="$IMAGE_REPO/zenith-api:$VERSION" \
    --set images.web="$IMAGE_REPO/zenith-web:$VERSION"
}

main() {
  resolve_domain
  preflight
  install_k3s
  install_helm
  install_operators
  ensure_secrets
  fetch_chart
  install_zenith
  echo
  log "Zenith is ready: https://$ZENITH_DOMAIN"
  if [ "${CREATED_ADMIN:-0}" = 1 ]; then
    echo "   admin:    $ADMIN_EMAIL"
    echo "   password: $ADMIN_PASSWORD   (change it after logging in)"
  fi
  echo "   Add nodes: k3s agent join token is in /var/lib/rancher/k3s/server/node-token"
}

main "$@"
