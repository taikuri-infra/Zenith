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
  [ -d "$dir/traefik" ] || cp -r "$repo/traefik" "$dir/traefik"
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

# --- self-signed mode renders certs, Traefik config and the three env keys ---
(umask 077; run "$t/ss" ZENITH_TLS_MODE=selfsigned)   # strict umask: modes must not follow it
env_ss="$t/ss/.env"
grep -qE '^ZENITH_TLS_MODE=selfsigned$' "$env_ss" || fail "ZENITH_TLS_MODE missing"
grep -qE '^ZENITH_TLS_CA_FILE=/etc/zenith/certs/zenith-ca.crt$' "$env_ss" || fail "ZENITH_TLS_CA_FILE missing"
grep -qE '^ZENITH_CERT_RESOLVER=$' "$env_ss" || fail "ZENITH_CERT_RESOLVER must be empty in selfsigned mode"
for f in traefik/certs/zenith.crt traefik/certs/zenith.key traefik/certs/zenith-ca.crt traefik/certs/zenith-ca.key traefik/dynamic/certs.yml traefik/traefik.yml; do
  [ -s "$t/ss/$f" ] || fail "selfsigned: $f not written"
done
grep -q 'certResolver' "$t/ss/traefik/traefik.yml" && fail "selfsigned traefik.yml must not name a certResolver"
grep -qE '^certificatesResolvers:' "$t/ss/traefik/traefik.yml" && fail "selfsigned traefik.yml must not define certificatesResolvers"
# the api container (non-root) reads the CA: certificates 0644, keys 0600
for f in zenith.crt zenith-ca.crt; do [ "$(stat -c %a "$t/ss/traefik/certs/$f")" = 644 ] || fail "selfsigned: $f mode $(stat -c %a "$t/ss/traefik/certs/$f"), want 644"; done
for f in zenith.key zenith-ca.key; do [ "$(stat -c %a "$t/ss/traefik/certs/$f")" = 600 ] || fail "selfsigned: $f mode $(stat -c %a "$t/ss/traefik/certs/$f"), want 600"; done
openssl x509 -in "$t/ss/traefik/certs/zenith.crt" -noout -ext subjectAltName | grep -q 'DNS:\*\.203.0.113.7.sslip.io' || fail "leaf lacks the wildcard SAN"
openssl verify -CAfile "$t/ss/traefik/certs/zenith-ca.crt" "$t/ss/traefik/certs/zenith.crt" >/dev/null 2>&1 || fail "leaf does not verify against the CA"
diff <(sed -n '/^tls:/,$p' "$t/ss/traefik/dynamic/certs.yml") - <<'EOF' || fail "certs.yml content differs from the Go installer's"
tls:
  stores:
    default:
      defaultCertificate:
        certFile: /etc/traefik/certs/zenith.crt
        keyFile: /etc/traefik/certs/zenith.key
  certificates:
    - certFile: /etc/traefik/certs/zenith.crt
      keyFile: /etc/traefik/certs/zenith.key
EOF

# --- http01 (default) is untouched and image repo passes through ---
run "$t/http" ZENITH_IMAGE_REPO=zenith-e2e-registry:5000
grep -qE '^ZENITH_TLS_MODE=' "$t/http/.env" && fail "http01 must not write ZENITH_TLS_MODE"
grep -qE '^ZENITH_IMAGE_REPO=zenith-e2e-registry:5000$' "$t/http/.env" || fail "ZENITH_IMAGE_REPO not written"
[ -e "$t/http/traefik/dynamic/certs.yml" ] && fail "http01 must not write certs.yml"

# --- selfsigned keys are added once and http01 stays untouched on re-run ---
run "$t/ss" ZENITH_TLS_MODE=selfsigned
[ "$(grep -c '^ZENITH_TLS_MODE=' "$env_ss")" = 1 ] || fail "ZENITH_TLS_MODE duplicated on re-run"
cp "$t/http/.env" "$t/http/.env.before"; run "$t/http"; diff -q "$t/http/.env" "$t/http/.env.before" >/dev/null || fail "re-run changed an http01 .env"

# --- ADMIN_PASSWORD from the environment is honoured ---
run "$t/pw" ADMIN_PASSWORD=Fixed-Password-123
grep -qE '^ADMIN_PASSWORD=Fixed-Password-123$' "$t/pw/.env" || fail "ADMIN_PASSWORD override ignored"

# --- set_var keeps sed-special characters verbatim and refuses a newline ---
run "$t/pw2" 'ADMIN_PASSWORD=p&ss|w\x'
[ "$(grep '^ADMIN_PASSWORD=' "$t/pw2/.env" | cut -d= -f2-)" = 'p&ss|w\x' ] || fail "ADMIN_PASSWORD with & | \\ not stored verbatim: $(grep '^ADMIN_PASSWORD=' "$t/pw2/.env")"
if out="$(run "$t/pw3" "ADMIN_PASSWORD=$(printf 'a\nb')" 2>&1)"; then fail "a newline in a value must be rejected"; fi
echo "$out" | grep -q 'must not contain a newline' || fail "missing newline error message: $out"
[ ! -e "$t/pw3/.env" ] || fail ".env created despite the newline in ADMIN_PASSWORD"

echo "install_test: ok"
