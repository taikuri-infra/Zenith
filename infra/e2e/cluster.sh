#!/usr/bin/env bash
# infra/e2e/cluster.sh — the E2E harness's "fresh server": a k3d cluster (k3s in Docker).
#
# `zen install` runs on the host against the cluster exactly as it would on a server, with the
# candidate chart and images. The Go suite (cli/e2e) never calls docker or kubectl: it reads
# <artifacts>/<name>.env and drives the cluster through three hook commands (exec, reboot, cp).
#
# Usage:
#   cluster.sh new <name>                create an empty k3s cluster
#   cluster.sh install <name>            zen install the candidate (images loaded from the host daemon)
#   cluster.sh reboot <name>             restart the node; 0 once /api/v1/version answers again
#   cluster.sh logs <name>               collect pod logs and events into <artifacts>/<name>/
#   cluster.sh rm <name>                 delete the cluster
#   cluster.sh candidate-version         print <VERSION>-e2e.<sha7>
# Env: ZENITH_E2E_ARTIFACTS (default infra/e2e/artifacts), ZENITH_E2E_ZEN (the zen binary).
set -euo pipefail

K3D_VERSION=v5.8.3
K3D_SHA256_amd64=dbaa79a76ace7f4ca230a1ff41dc7d8a5036a8ad0309e9c54f9bf3836dbe853e
K3D_SHA256_arm64=0b8110f2229631af7402fb828259330985918b08fefd38b7f1b788a1c8687216
ADMIN_EMAIL=e2e@zenith.local
IMAGE_REPO=e2e.local
ISSUER=zenith-e2e-ca
HEALTH_TIMEOUT=300

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
ART="${ZENITH_E2E_ARTIFACTS:-$here/artifacts}"

log() { printf '\033[36m[e2e]\033[0m %s\n' "$*" >&2; }
die() { printf '\033[31m[e2e]\033[0m %s\n' "$*" >&2; exit 1; }
cname() { [ -n "${1:-}" ] || die "missing cluster name"; echo "zenith-e2e-$1"; }
node() { echo "k3d-$(cname "$1")-server-0"; }
kubeconfig() { echo "$ART/$1.kubeconfig"; }
candidate_version() { echo "$(tr -d '[:space:]' < "$repo/VERSION")-e2e.$(git -C "$repo" rev-parse --short=7 HEAD)"; }

ensure_k3d() {
  if command -v k3d >/dev/null; then return; fi
  local arch want f
  case "$(uname -m)" in x86_64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) die "unsupported CPU $(uname -m)" ;; esac
  want="K3D_SHA256_$arch"; mkdir -p "$ART/bin"; f="$ART/bin/k3d"
  if [ ! -x "$f" ]; then
    log "downloading k3d $K3D_VERSION"
    curl -fsSL -o "$f" "https://github.com/k3d-io/k3d/releases/download/$K3D_VERSION/k3d-linux-$arch"
    echo "${!want}  $f" | sha256sum -c - >/dev/null || { rm -f "$f"; die "k3d checksum mismatch"; }
    chmod +x "$f"
  fi
  export PATH="$ART/bin:$PATH"
}

node_ip() { docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$(node "$1")"; }

wait_api() { # wait_api URL CA
  local url="$1" ca="$2" end=$((SECONDS + HEALTH_TIMEOUT))
  until curl -fsS --max-time 5 --cacert "$ca" "$url/api/v1/version" >/dev/null 2>&1; do
    [ "$SECONDS" -lt "$end" ] || die "$url/api/v1/version did not answer within ${HEALTH_TIMEOUT}s"
    sleep 3
  done
}

cmd_new() {
  ensure_k3d
  local name="$1"; mkdir -p "$ART"
  log "creating cluster $(cname "$name")"
  k3d cluster create "$(cname "$name")" --servers 1 --agents 0 --no-lb --wait --timeout 300s \
    --kubeconfig-update-default=false --kubeconfig-switch-context=false >/dev/null
  k3d kubeconfig get "$(cname "$name")" > "$(kubeconfig "$name")"
  chmod 600 "$(kubeconfig "$name")"
}

# cert-manager and a CA ClusterIssuer first: the installer then reuses cert-manager and
# `zen install` points the chart at this issuer instead of Let's Encrypt.
install_ca() {
  local kc="$1"
  helm repo add jetstack https://charts.jetstack.io --force-update >/dev/null
  helm --kubeconfig "$kc" upgrade --install cert-manager jetstack/cert-manager -n cert-manager --create-namespace \
    --set crds.enabled=true --wait --timeout 8m >/dev/null
  kubectl --kubeconfig "$kc" apply -f - >/dev/null <<YAML
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata: {name: ${ISSUER}-bootstrap}
spec: {selfSigned: {}}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata: {name: ${ISSUER}, namespace: cert-manager}
spec:
  isCA: true
  commonName: Zenith E2E CA
  secretName: ${ISSUER}
  privateKey: {algorithm: ECDSA, size: 256}
  issuerRef: {name: ${ISSUER}-bootstrap, kind: ClusterIssuer}
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata: {name: ${ISSUER}}
spec: {ca: {secretName: ${ISSUER}}}
YAML
  kubectl --kubeconfig "$kc" -n cert-manager wait --for=condition=Ready "certificate/$ISSUER" --timeout=120s >/dev/null
  kubectl --kubeconfig "$kc" wait --for=condition=Ready "clusterissuer/$ISSUER" --timeout=120s >/dev/null
}

cmd_install() {
  ensure_k3d
  local name="$1" cand kc ip domain pw ca zen start
  cand="$(candidate_version)"; kc="$(kubeconfig "$name")"; zen="${ZENITH_E2E_ZEN:?ZENITH_E2E_ZEN is not set}"
  ip="$(node_ip "$name")"; [ -n "$ip" ] || die "no IP for $(node "$name")"
  domain="$ip.sslip.io"; pw="$(openssl rand -hex 12)"; ca="$ART/$name-ca.crt"

  log "loading images into $(cname "$name")"
  k3d image import -c "$(cname "$name")" "$IMAGE_REPO/zenith-api:$cand" "$IMAGE_REPO/zenith-web:$cand" "$IMAGE_REPO/e2e-probe:$cand" >/dev/null
  install_ca "$kc"

  log "zen install on $domain"
  start=$SECONDS
  KUBECONFIG="$kc" ZENITH_CHART="$repo/infra/helm/zenith-selfhost" ZENITH_IMAGE_REPO="$IMAGE_REPO" \
    ZENITH_PULL_POLICY=Never ZENITH_TLS_ISSUER="$ISSUER" ZENITH_APP_CERTS=true \
    "$zen" install --domain "$domain" --version "$cand" --admin-email "$ADMIN_EMAIL" --admin-password "$pw"
  local secs=$((SECONDS - start))

  kubectl --kubeconfig "$kc" -n cert-manager get secret "$ISSUER" -o jsonpath='{.data.ca\.crt}' | base64 -d > "$ca"
  wait_api "https://$domain" "$ca"
  cmd_env "$name" "$domain" "$pw" "$secs" "$cand"
}

# The suite environment; the hooks run on this host with the cluster's kubeconfig.
cmd_env() {
  local name="$1" domain="$2" pw="$3" secs="$4" cand="$5" kc; kc="$(kubeconfig "$name")"
  {
    echo "ZENITH_E2E_ENDPOINT=https://$domain"
    echo "ZENITH_E2E_DOMAIN=$domain"
    echo "ZENITH_E2E_ADMIN_EMAIL=$ADMIN_EMAIL"
    echo "ZENITH_E2E_ADMIN_PASSWORD=$pw"
    echo "ZENITH_E2E_CA_FILE=$ART/$name-ca.crt"
    echo "ZENITH_E2E_ZEN=${ZENITH_E2E_ZEN}"
    echo "ZENITH_E2E_INSTALL_SECONDS=$secs"
    echo "ZENITH_E2E_SERVER_EXEC=env PATH=$(dirname "${ZENITH_E2E_ZEN}"):$PATH KUBECONFIG=$kc ZENITH_CHART=$repo/infra/helm/zenith-selfhost"
    echo "ZENITH_E2E_REBOOT_CMD=bash $here/cluster.sh reboot $name"
    echo "ZENITH_E2E_COPY_FROM_SERVER=cp"
    echo "ZENITH_E2E_CANDIDATE_VERSION=$cand"
    echo "ZENITH_E2E_PROBE_IMAGE=$IMAGE_REPO/e2e-probe:$cand"
    echo "ZENITH_E2E_ARTIFACTS=$ART"
  } > "$ART/$name.env"
  log "suite environment: $ART/$name.env"
}

cmd_reboot() {
  local name="$1" env_file="$ART/$1.env" endpoint ca
  endpoint="$(grep '^ZENITH_E2E_ENDPOINT=' "$env_file" | cut -d= -f2-)"
  ca="$(grep '^ZENITH_E2E_CA_FILE=' "$env_file" | cut -d= -f2-)"
  docker restart "$(node "$name")" >/dev/null
  wait_api "$endpoint" "$ca"
}

cmd_logs() {
  local name="$1" kc out; kc="$(kubeconfig "$name")"; out="$ART/$name"
  [ -s "$kc" ] || return 0
  mkdir -p "$out"
  kubectl --kubeconfig "$kc" get pods -A -o wide > "$out/pods.txt" 2>&1 || true
  kubectl --kubeconfig "$kc" get events -A --sort-by=.lastTimestamp > "$out/events.txt" 2>&1 || true
  for ns in zenith-system zenith-apps; do
    kubectl --kubeconfig "$kc" -n "$ns" logs --all-containers --prefix --tail=500 -l '!cnpg.io/cluster' > "$out/$ns.log" 2>&1 || true
  done
}

cmd_rm() { ensure_k3d; k3d cluster delete "$(cname "$1")" >/dev/null 2>&1 || true; }

case "${1:-}" in
  new) cmd_new "${2:-}" ;;
  install) cmd_install "${2:-}" ;;
  reboot) cmd_reboot "${2:-}" ;;
  logs) cmd_logs "${2:-}" ;;
  rm) cmd_rm "${2:-}" ;;
  candidate-version) candidate_version ;;
  *) sed -n 2,17p "$0" >&2; exit 2 ;;
esac
