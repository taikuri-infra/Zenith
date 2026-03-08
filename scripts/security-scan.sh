#!/bin/bash
# =============================================================================
# Zenith Platform Security Scan
# Runs kube-bench, kube-hunter, trivy, and kubeaudit against the cluster
# Usage: ./scripts/security-scan.sh [--kubeconfig <path>] [--output <dir>]
# =============================================================================
set -euo pipefail

KUBECONFIG="${KUBECONFIG:-$HOME/.kube/config}"
OUTPUT_DIR="${2:-./security-reports}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

while [[ $# -gt 0 ]]; do
  case $1 in
    --kubeconfig) KUBECONFIG="$2"; shift 2 ;;
    --output) OUTPUT_DIR="$2"; shift 2 ;;
    *) shift ;;
  esac
done

mkdir -p "$OUTPUT_DIR"

echo "================================================"
echo "  Zenith Security Scan — $TIMESTAMP"
echo "  Kubeconfig: $KUBECONFIG"
echo "  Output: $OUTPUT_DIR"
echo "================================================"
echo ""

# --- 1. kube-bench: CIS Kubernetes Benchmark ---
echo "[1/4] Running kube-bench (CIS Benchmark)..."
if command -v kube-bench &>/dev/null; then
  kube-bench run --json > "$OUTPUT_DIR/kube-bench_${TIMESTAMP}.json" 2>&1 || true
  echo "  -> $OUTPUT_DIR/kube-bench_${TIMESTAMP}.json"
else
  echo "  -> Running via kubectl job..."
  kubectl apply -f - <<'EOF'
apiVersion: batch/v1
kind: Job
metadata:
  name: kube-bench
  namespace: default
spec:
  template:
    spec:
      hostPID: true
      containers:
        - name: kube-bench
          image: aquasec/kube-bench:latest
          command: ["kube-bench", "run", "--json"]
          volumeMounts:
            - name: var-lib-etcd
              mountPath: /var/lib/etcd
              readOnly: true
            - name: etc-kubernetes
              mountPath: /etc/kubernetes
              readOnly: true
      restartPolicy: Never
      volumes:
        - name: var-lib-etcd
          hostPath:
            path: /var/lib/etcd
        - name: etc-kubernetes
          hostPath:
            path: /etc/kubernetes
  backoffLimit: 0
EOF
  echo "  -> Job submitted. Retrieve results with:"
  echo "     kubectl logs job/kube-bench > $OUTPUT_DIR/kube-bench_${TIMESTAMP}.json"
fi
echo ""

# --- 2. kube-hunter: Penetration Testing ---
echo "[2/4] Running kube-hunter..."
if command -v kube-hunter &>/dev/null; then
  kube-hunter --pod --report json > "$OUTPUT_DIR/kube-hunter_${TIMESTAMP}.json" 2>&1 || true
  echo "  -> $OUTPUT_DIR/kube-hunter_${TIMESTAMP}.json"
else
  kubectl run kube-hunter \
    --image=aquasec/kube-hunter:latest \
    --restart=Never \
    --rm -i \
    -- --pod --report json \
    > "$OUTPUT_DIR/kube-hunter_${TIMESTAMP}.json" 2>&1 || true
  echo "  -> $OUTPUT_DIR/kube-hunter_${TIMESTAMP}.json"
fi
echo ""

# --- 3. Trivy: Vulnerability Scanning ---
echo "[3/4] Running Trivy (image + k8s scan)..."
if command -v trivy &>/dev/null; then
  # Scan the cluster for misconfigurations
  trivy k8s --report summary \
    --format json \
    --output "$OUTPUT_DIR/trivy-k8s_${TIMESTAMP}.json" \
    cluster 2>&1 || true
  echo "  -> $OUTPUT_DIR/trivy-k8s_${TIMESTAMP}.json"

  # Scan key platform images
  for IMAGE in \
    "ghcr.io/cloudnative-pg/cloudnative-pg:1.23" \
    "quay.io/keycloak/keycloak:25.0" \
    "apache/apisix:3.10" \
    "grafana/loki:3.0" \
    "grafana/tempo:2.5"; do
    SAFE_NAME=$(echo "$IMAGE" | tr '/:' '_')
    trivy image --format json \
      --output "$OUTPUT_DIR/trivy-image-${SAFE_NAME}_${TIMESTAMP}.json" \
      "$IMAGE" 2>&1 || true
  done
  echo "  -> Image scan results in $OUTPUT_DIR/trivy-image-*"
else
  echo "  -> trivy not found. Install: https://aquasecurity.github.io/trivy/"
fi
echo ""

# --- 4. kubeaudit: Security Auditing ---
echo "[4/4] Running kubeaudit..."
if command -v kubeaudit &>/dev/null; then
  kubeaudit all --json \
    --kubeconfig "$KUBECONFIG" \
    > "$OUTPUT_DIR/kubeaudit_${TIMESTAMP}.json" 2>&1 || true
  echo "  -> $OUTPUT_DIR/kubeaudit_${TIMESTAMP}.json"
else
  echo "  -> kubeaudit not found. Install: https://github.com/Shopify/kubeaudit"
fi
echo ""

# --- Summary ---
echo "================================================"
echo "  Scan Complete"
echo "  Reports: $OUTPUT_DIR/"
echo "================================================"
echo ""
echo "Files generated:"
ls -la "$OUTPUT_DIR"/*"${TIMESTAMP}"* 2>/dev/null || echo "  (check individual tool output)"
echo ""
echo "Next steps:"
echo "  1. Review critical/high findings in each report"
echo "  2. Fix findings and re-scan"
echo "  3. Document exceptions in docs/security-exceptions.md"
