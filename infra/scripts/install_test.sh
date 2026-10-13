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
  cp "$repo/VERSION" "$dir/VERSION"
  (cd "$dir" && env "$@" ZENITH_DRY_RUN=1 ZENITH_PUBLIC_IP=203.0.113.7 bash "$here/install.sh" >/dev/null)
}

t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT

run "$t/a"
env_a="$t/a/.env"
grep -qE '^SECRETS_ENCRYPTION_KEY=[0-9a-f]{64}$' "$env_a" || fail "SECRETS_ENCRYPTION_KEY not 64 hex"
grep -qE '^S3_ACCESS_KEY=[A-Za-z0-9]{16,}$' "$env_a" || fail "S3_ACCESS_KEY not generated"
grep -qx 'ZENITH_DOMAIN=203.0.113.7.sslip.io' "$env_a" || fail "default domain not <ip>.sslip.io"
grep -qx "ZENITH_VERSION=$(cat "$repo/VERSION")" "$env_a" || fail "ZENITH_VERSION not pinned from VERSION"

run "$t/b" ZENITH_DOMAIN=zenith.example.com
grep -qx 'ZENITH_DOMAIN=zenith.example.com' "$t/b/.env" || fail "explicit domain not kept"

# stubs: nothing real (network, docker, git) may run in any case below
stubs="$t/stubs"; mkdir -p "$stubs"
for c in docker git; do printf '#!/bin/sh\necho "stub %s called" >&2; exit 97\n' "$c" >"$stubs/$c"; done
for c in curl hostname; do printf '#!/bin/sh\nexit 1\n' >"$stubs/$c"; done
chmod +x "$stubs"/*

# (a) re-run keeps the encryption key byte-identical
key_before="$(grep '^SECRETS_ENCRYPTION_KEY=' "$env_a")"
(cd "$t/a" && ZENITH_DRY_RUN=1 bash "$here/install.sh" >/dev/null)
[ "$key_before" = "$(grep '^SECRETS_ENCRYPTION_KEY=' "$env_a")" ] || fail "re-run changed SECRETS_ENCRYPTION_KEY"

# (b) old-style .env gets domain + key added, other values kept
mkdir -p "$t/old"; printf 'JWT_SECRET=keepme\nSECRETS_ENCRYPTION_KEY=\nZENITH_VERSION=0.0.1\n' >"$t/old/.env"
cp "$repo/VERSION" "$t/old/VERSION"
(cd "$t/old" && env ZENITH_DRY_RUN=1 ZENITH_PUBLIC_IP=203.0.113.9 bash "$here/install.sh" >/dev/null)
grep -qx 'ZENITH_DOMAIN=203.0.113.9.sslip.io' "$t/old/.env" || fail "upgrade: domain not added"
grep -qE '^SECRETS_ENCRYPTION_KEY=[0-9a-f]{64}$' "$t/old/.env" || fail "upgrade: key not added"
grep -qx 'JWT_SECRET=keepme' "$t/old/.env" || fail "upgrade: existing value changed"
grep -qx "ZENITH_VERSION=$(cat "$repo/VERSION")" "$t/old/.env" || fail "upgrade: ZENITH_VERSION not updated from VERSION"

# (c) IP lookup failure: clean error, no .env created
mkdir -p "$t/noip"; cp "$repo/.env.example" "$t/noip/"
if out="$(cd "$t/noip" && env -u ZENITH_PUBLIC_IP -u ZENITH_DOMAIN PATH="$stubs:$PATH" ZENITH_DRY_RUN=1 bash "$here/install.sh" 2>&1)"; then
  fail "installer should fail when the public IP cannot be detected"
fi
echo "$out" | grep -q "Could not detect this server's public IP" || fail "missing IP error message"
[ ! -e "$t/noip/.env" ] || fail ".env created despite IP failure"

echo "install_test: ok"
