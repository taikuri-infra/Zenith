#!/usr/bin/env bash
# infra/scripts/install_test.sh — renders install.sh's .env in a temp dir
# without Docker or network (ZENITH_DRY_RUN=1) and checks the result.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
fail() { echo "FAIL: $*" >&2; exit 1; }

run() { # $1 = workdir; remaining = env assignments
  local dir="$1"; shift
  mkdir -p "$dir"
  cp "$repo/.env.example" "$dir/.env.example"
  (cd "$dir" && env "$@" ZENITH_DRY_RUN=1 ZENITH_PUBLIC_IP=203.0.113.7 bash "$here/install.sh" >/dev/null)
}

t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT

run "$t/a"
env_a="$t/a/.env"
grep -qE '^SECRETS_ENCRYPTION_KEY=[0-9a-f]{64}$' "$env_a" || fail "SECRETS_ENCRYPTION_KEY not 64 hex"
grep -qE '^S3_ACCESS_KEY=[A-Za-z0-9]{16,}$' "$env_a" || fail "S3_ACCESS_KEY not generated"
grep -qx 'ZENITH_DOMAIN=203.0.113.7.sslip.io' "$env_a" || fail "default domain not <ip>.sslip.io"

run "$t/b" ZENITH_DOMAIN=zenith.example.com
grep -qx 'ZENITH_DOMAIN=zenith.example.com' "$t/b/.env" || fail "explicit domain not kept"

echo "install_test: ok"
