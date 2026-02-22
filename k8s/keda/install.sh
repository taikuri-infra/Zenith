#!/usr/bin/env bash
# install.sh — Installs KEDA + HTTP Add-on for scale-to-zero support.
# Idempotent: safe to re-run.
#
# Usage: bash k8s/keda/install.sh

set -euo pipefail

KEDA_NS="${KEDA_NAMESPACE:-keda}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> Adding KEDA Helm repo"
helm repo add kedacore https://kedacore.github.io/charts 2>/dev/null || true
helm repo update kedacore

echo "==> Creating namespace ${KEDA_NS}"
kubectl create namespace "${KEDA_NS}" --dry-run=client -o yaml | kubectl apply -f -

echo "==> Installing KEDA"
helm upgrade --install keda kedacore/keda \
  --namespace "${KEDA_NS}" \
  --values "${SCRIPT_DIR}/values-keda.yaml" \
  --wait

echo "==> Installing KEDA HTTP Add-on"
helm upgrade --install keda-http-addon kedacore/keda-add-ons-http \
  --namespace "${KEDA_NS}" \
  --values "${SCRIPT_DIR}/values-http-addon.yaml" \
  --wait

echo "==> Applying cold-start page service"
kubectl apply -f "${SCRIPT_DIR}/cold-start-service.yaml"

echo "==> Applying error-page middleware"
kubectl apply -f "${SCRIPT_DIR}/error-page-middleware.yaml"

echo "==> KEDA + HTTP Add-on installed successfully"
echo "    Namespace: ${KEDA_NS}"
echo "    Cold-start page: zenith-apps/cold-start-page"
