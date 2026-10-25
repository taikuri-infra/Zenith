#!/usr/bin/env bash
# infra/scripts/install_k3s_test.sh — checks install-k3s.sh without a cluster: the dry run
# prints the plan, the environment knobs reach the Helm values, and the chart renders.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; repo="$(cd "$here/../.." && pwd)"
script="$here/install-k3s.sh"
fail() { echo "install_k3s_test: FAIL: $*" >&2; exit 1; }

dry() { env -i PATH="$PATH" HOME="$HOME" ZENITH_DRY_RUN=1 ZENITH_CHART="$repo/infra/helm/zenith-selfhost" "$@" bash "$script"; }

out="$(dry ZENITH_DOMAIN=zenith.example.com ZENITH_EMAIL=ops@example.com ZENITH_VERSION=0.3.0)"
grep -q 'helm upgrade --install zenith ' <<<"$out" || fail "no zenith helm install in the plan"
grep -q -- '--set domain=zenith.example.com' <<<"$out" || fail "domain not passed to the chart"
grep -q -- '--set tls.issuer.email=ops@example.com' <<<"$out" || fail "email not passed to the chart"
grep -q 'zenith-api:0.3.0' <<<"$out" || fail "version not in the image tag"
for op in cloudnative-pg mariadb-operator redis-operator cert-manager; do
  grep -q "helm upgrade --install .*$op" <<<"$out" || fail "$op is not installed"
done
grep -q 'tls.appCerts' <<<"$out" && fail "a real domain should keep per-app certificates on (chart default)"

out="$(dry ZENITH_DOMAIN=1.2.3.4.sslip.io)"
grep -q -- '--set tls.appCerts=false' <<<"$out" || fail "sslip.io domains should turn per-app certificates off"

out="$(dry ZENITH_DOMAIN=z.example.com ZENITH_TLS_ISSUER=my-ca ZENITH_PULL_POLICY=Never ZENITH_APP_CERTS=true)"
grep -q -- '--set tls.issuerName=my-ca' <<<"$out" || fail "ZENITH_TLS_ISSUER ignored"
grep -q -- '--set tls.issuer.create=false' <<<"$out" || fail "own issuer must not create one"
grep -q -- '--set images.pullPolicy=Never' <<<"$out" || fail "ZENITH_PULL_POLICY ignored"
grep -q -- '--set tls.appCerts=true' <<<"$out" || fail "ZENITH_APP_CERTS ignored"

if command -v helm >/dev/null; then
  helm template z "$repo/infra/helm/zenith-selfhost" --set domain=z.example.com --set tls.issuer.email=a@b.c >/dev/null \
    || fail "chart does not render"
  msg="$(helm template z "$repo/infra/helm/zenith-selfhost" --set domain=z.example.com 2>&1 || true)"
  grep -q 'tls.issuer.email is required' <<<"$msg" || fail "chart should require tls.issuer.email"
fi
echo "install_k3s_test: ok"
