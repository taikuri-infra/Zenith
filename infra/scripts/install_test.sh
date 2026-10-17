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

run_err() { # like run, but ZENITH_PUBLIC_IP comes from the caller; prints stderr
  local dir="$1"; shift
  mkdir -p "$dir"
  cp "$repo/.env.example" "$dir/.env.example"
  cp "$repo/VERSION" "$dir/VERSION"
  [ -d "$dir/traefik" ] || cp -r "$repo/traefik" "$dir/traefik"
  (cd "$dir" && env -u ZENITH_PUBLIC_IP -u ZENITH_DOMAIN "$@" ZENITH_DRY_RUN=1 bash "$here/install.sh" 2>&1 >/dev/null)
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
grep -qE '^ZENITH_TRAEFIK_CONFIG=/etc/traefik/traefik.local.yml$' "$env_ss" || fail "ZENITH_TRAEFIK_CONFIG missing"
for f in traefik/certs/zenith.crt traefik/certs/zenith.key traefik/certs/zenith-ca.crt traefik/certs/zenith-ca.key traefik/dynamic/certs.yml traefik/traefik.local.yml; do
  [ -s "$t/ss/$f" ] || fail "selfsigned: $f not written"
done
# the tracked traefik.yml is never written (zen upgrade's git checkout must stay clean)
cmp -s "$t/ss/traefik/traefik.yml" "$repo/traefik/traefik.yml" || fail "selfsigned: tracked traefik/traefik.yml was modified"
grep -q 'certResolver' "$t/ss/traefik/traefik.local.yml" && fail "traefik.local.yml must not name a certResolver"
grep -qE '^certificatesResolvers:' "$t/ss/traefik/traefik.local.yml" && fail "traefik.local.yml must not define certificatesResolvers"
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
[ -e "$t/http/traefik/traefik.local.yml" ] && fail "http01 must not write traefik.local.yml"
grep -qE '^ZENITH_TRAEFIK_CONFIG=' "$t/http/.env" && fail "http01 must not write ZENITH_TRAEFIK_CONFIG"
cmp -s "$t/http/traefik/traefik.yml" "$repo/traefik/traefik.yml" || fail "http01: tracked traefik/traefik.yml was modified"

# --- selfsigned keys are added once and http01 stays untouched on re-run ---
run "$t/ss" ZENITH_TLS_MODE=selfsigned
[ "$(grep -c '^ZENITH_TLS_MODE=' "$env_ss")" = 1 ] || fail "ZENITH_TLS_MODE duplicated on re-run"
cp "$t/http/.env" "$t/http/.env.before"; run "$t/http"; diff -q "$t/http/.env" "$t/http/.env.before" >/dev/null || fail "re-run changed an http01 .env"

# --- an alpha.1 self-signed install (traefik.yml overwritten, no ZENITH_TRAEFIK_CONFIG) is migrated on re-run ---
mig="$t/mig"; mkdir -p "$mig"; cp -r "$repo/traefik" "$mig/traefik"
git -C "$mig" init -q && git -C "$mig" add traefik/traefik.yml && git -C "$mig" -c user.name=t -c user.email=t@t commit -qm init
cp "$t/ss/traefik/traefik.local.yml" "$mig/traefik/traefik.yml"; cp "$mig/traefik/traefik.yml" "$t/mig.old"
grep -v '^ZENITH_TRAEFIK_CONFIG=' "$env_ss" >"$mig/.env"
out="$(cd "$mig" && env -u ZENITH_TLS_MODE ZENITH_DRY_RUN=1 bash "$here/install.sh" 2>&1)" || fail "migration re-run failed: $out"
echo "$out" | grep -q 'edit traefik/traefik.local.yml from now on$' || fail "migration not logged with the edit hint: $out"
cmp -s "$mig/traefik/traefik.yml" "$repo/traefik/traefik.yml" || fail "migration: traefik/traefik.yml not restored"
cmp -s "$mig/traefik/traefik.local.yml" "$t/mig.old" || fail "migration: modified traefik.yml not moved to traefik.local.yml"
grep -qx 'ZENITH_TRAEFIK_CONFIG=/etc/traefik/traefik.local.yml' "$mig/.env" || fail "migration: ZENITH_TRAEFIK_CONFIG not set"
[ "$(grep -c '^ZENITH_TRAEFIK_CONFIG=' "$mig/.env")" = 1 ] || fail "migration: key duplicated"
(cd "$mig" && ZENITH_DRY_RUN=1 bash "$here/install.sh" >/dev/null) || fail "second re-run after migration failed"
[ "$(grep -c '^ZENITH_TRAEFIK_CONFIG=' "$mig/.env")" = 1 ] || fail "migration: key duplicated on second re-run"
# --- an http01 install whose traefik.yml an older `zen install` overwrote is migrated too ---
mh="$t/mighttp"; mkdir -p "$mh"; cp -r "$repo/traefik" "$mh/traefik"
git -C "$mh" init -q && git -C "$mh" add traefik/traefik.yml && git -C "$mh" -c user.name=t -c user.email=t@t commit -qm init
printf '# Generated by `zen install --edition compose`.\ncertificatesResolvers:\n  le: {}\n' >"$mh/traefik/traefik.yml"; cp "$mh/traefik/traefik.yml" "$t/mighttp.old"
cp "$t/http/.env" "$mh/.env"
out="$(cd "$mh" && ZENITH_DRY_RUN=1 bash "$here/install.sh" 2>&1)" || fail "http01 migration re-run failed: $out"
cmp -s "$mh/traefik/traefik.yml" "$repo/traefik/traefik.yml" || fail "http01 migration: traefik/traefik.yml not restored"
cmp -s "$mh/traefik/traefik.local.yml" "$t/mighttp.old" || fail "http01 migration: modified traefik.yml not moved"
grep -qx 'ZENITH_TRAEFIK_CONFIG=/etc/traefik/traefik.local.yml' "$mh/.env" || fail "http01 migration: key not set"
grep -qE '^ZENITH_TLS_MODE=' "$mh/.env" && fail "http01 migration must not write ZENITH_TLS_MODE"
# key set and traefik.local.yml live, but the tracked file still modified: only restore it
ml="$t/miglive"; mkdir -p "$ml"; cp -r "$repo/traefik" "$ml/traefik"
git -C "$ml" init -q && git -C "$ml" add traefik/traefik.yml && git -C "$ml" -c user.name=t -c user.email=t@t commit -qm init
printf 'generated\n' >"$ml/traefik/traefik.yml"; printf 'live\n' >"$ml/traefik/traefik.local.yml"
cp "$t/http/.env" "$ml/.env"; echo 'ZENITH_TRAEFIK_CONFIG=/etc/traefik/traefik.local.yml' >>"$ml/.env"; cp "$ml/.env" "$t/miglive.env"
out="$(cd "$ml" && ZENITH_DRY_RUN=1 bash "$here/install.sh" 2>&1)" || fail "live-local re-run failed: $out"
cmp -s "$ml/traefik/traefik.yml" "$repo/traefik/traefik.yml" || fail "live-local: tracked traefik.yml not restored"
[ "$(cat "$ml/traefik/traefik.local.yml")" = live ] || fail "live-local: traefik.local.yml changed"
diff -q "$ml/.env" "$t/miglive.env" >/dev/null || fail "live-local: .env changed"
echo "$out" | grep -q 'edit traefik/traefik.local.yml from now on$' || fail "live-local: not logged: $out"
# git checkout failing during the migration loses nothing
mf="$t/migfail"; mkdir -p "$mf"; cp -r "$repo/traefik" "$mf/traefik"
git -C "$mf" init -q && git -C "$mf" add traefik/traefik.yml && git -C "$mf" -c user.name=t -c user.email=t@t commit -qm init
printf 'generated\n' >"$mf/traefik/traefik.yml"; cp "$t/http/.env" "$mf/.env"
gitfail="$t/gitfail"; mkdir -p "$gitfail"; realgit="$(command -v git)"
printf '#!/bin/sh\ncase "$1" in checkout) exit 1;; esac\nexec %s "$@"\n' "$realgit" >"$gitfail/git"; chmod +x "$gitfail/git"
out="$(cd "$mf" && env PATH="$gitfail:$PATH" ZENITH_DRY_RUN=1 bash "$here/install.sh" 2>&1)" || fail "re-run with failing checkout aborted: $out"
[ "$(cat "$mf/traefik/traefik.yml")" = generated ] || fail "failing checkout: traefik.yml lost"
[ -e "$mf/traefik/traefik.local.yml" ] && fail "failing checkout: traefik.local.yml left behind"
grep -q '^ZENITH_TRAEFIK_CONFIG=' "$mf/.env" && fail "failing checkout: key left in .env"
# an unmodified http01 checkout is left alone
mu="$t/migclean"; mkdir -p "$mu"; cp -r "$repo/traefik" "$mu/traefik"
git -C "$mu" init -q && git -C "$mu" add traefik/traefik.yml && git -C "$mu" -c user.name=t -c user.email=t@t commit -qm init
cp "$t/http/.env" "$mu/.env"; (cd "$mu" && ZENITH_DRY_RUN=1 bash "$here/install.sh" >/dev/null) || fail "clean http01 re-run failed"
diff -q "$mu/.env" "$t/http/.env" >/dev/null || fail "clean http01 re-run changed .env"
[ -e "$mu/traefik/traefik.local.yml" ] && fail "clean http01 re-run wrote traefik.local.yml"

# --- ADMIN_PASSWORD from the environment is honoured ---
run "$t/pw" ADMIN_PASSWORD=Fixed-Password-123
grep -qE '^ADMIN_PASSWORD=Fixed-Password-123$' "$t/pw/.env" || fail "ADMIN_PASSWORD override ignored"

# --- an address that is not on this machine (NAT) is warned about, not fatal ---
out="$(run_err "$t/nat" ZENITH_PUBLIC_IP=198.51.100.23)" || fail "NAT warning must not be fatal"
echo "$out" | grep -q 'address 198.51.100.23 is not on this machine (NAT?); set ZENITH_PUBLIC_IP or ZENITH_DOMAIN if the dashboard is unreachable' || fail "missing NAT warning: $out"
grep -qx 'ZENITH_DOMAIN=198.51.100.23.sslip.io' "$t/nat/.env" || fail "NAT: domain not <ip>.sslip.io"
out="$(run_err "$t/natdom" ZENITH_DOMAIN=zenith.example.com)" || fail "explicit domain run failed"
echo "$out" | grep -q 'is not on this machine' && fail "NAT warning printed although ZENITH_DOMAIN is set: $out"
# a local address (hostname -I stubbed to report it, no `ip`) is not warned about
locstubs="$t/locstubs"; mkdir -p "$locstubs"
printf '#!/bin/sh\necho 198.51.100.24 10.0.0.5\n' >"$locstubs/hostname"; printf '#!/bin/sh\nexit 1\n' >"$locstubs/ip"; chmod +x "$locstubs"/*
out="$(run_err "$t/natloc" PATH="$locstubs:$PATH" ZENITH_PUBLIC_IP=198.51.100.24)" || fail "local-address run failed"
echo "$out" | grep -q 'is not on this machine' && fail "NAT warning printed for a local address: $out"
# neither `ip` nor `hostname` exists: cannot tell, so no warning
noipbin="$t/noipbin"; mkdir -p "$noipbin"
for f in /usr/local/bin/* /usr/bin/* /bin/* /usr/sbin/* /sbin/*; do
  n="${f##*/}"; case "$n" in ip|hostname) continue;; esac
  [ -e "$noipbin/$n" ] || ln -s "$f" "$noipbin/$n"
done
out="$(run_err "$t/natnotool" PATH="$noipbin" ZENITH_PUBLIC_IP=198.51.100.25)" || fail "no-ip/hostname run failed: $out"
echo "$out" | grep -q 'is not on this machine' && fail "NAT warning printed although neither ip nor hostname exists: $out"

# --- set_var keeps sed-special characters verbatim and refuses a newline ---
run "$t/pw2" 'ADMIN_PASSWORD=p&ss|w\x'
[ "$(grep '^ADMIN_PASSWORD=' "$t/pw2/.env" | cut -d= -f2-)" = 'p&ss|w\x' ] || fail "ADMIN_PASSWORD with & | \\ not stored verbatim: $(grep '^ADMIN_PASSWORD=' "$t/pw2/.env")"
if out="$(run "$t/pw3" "ADMIN_PASSWORD=$(printf 'a\nb')" 2>&1)"; then fail "a newline in a value must be rejected"; fi
echo "$out" | grep -q 'must not contain a newline' || fail "missing newline error message: $out"
if out="$(run "$t/pw4" 'ADMIN_PASSWORD=pa$$word' 2>&1)"; then fail "a \$ in ADMIN_PASSWORD must be rejected (compose interpolates it)"; fi
echo "$out" | grep -q 'must not contain \$' || fail "missing \$ error message: $out"
[ ! -e "$t/pw3/.env" ] || fail ".env created despite the newline in ADMIN_PASSWORD"

echo "install_test: ok"
