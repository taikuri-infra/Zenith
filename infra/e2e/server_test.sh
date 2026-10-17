#!/usr/bin/env bash
# infra/e2e/server_test.sh — checks of the E2E harness that need no Docker (run by ci.yml).
# A fake `docker` on PATH fails the test if anything here tries to reach a daemon.
# shellcheck disable=SC2016 # single-quoted bash -c bodies expand their own "$1"
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
pass() { printf '  ok   %s\n' "$*"; }
fail() { printf '  FAIL %s\n' "$*"; fails=$((fails + 1)); }
check() { local what="$1"; shift; if "$@"; then pass "$what"; else fail "$what"; fi; }

mkdir -p "$tmp/bin"
printf '#!/bin/sh\necho "docker $*" >> "%s/docker-calls"\nexit 99\n' "$tmp" > "$tmp/bin/docker"
chmod +x "$tmp/bin/docker"
export PATH="$tmp/bin:$PATH" ZENITH_E2E_ARTIFACTS="$tmp/art"
unset ACT ZENITH_E2E_UPGRADE_FROM ZENITH_E2E_ZEN

echo "Dockerfile.server / daemon.json"
check "Dockerfile.server starts with FROM ubuntu:24.04" test "$(head -1 "$here/Dockerfile.server")" = "FROM ubuntu:24.04"
check "Dockerfile.server installs docker-ce and the compose plugin" grep -qE 'install .*docker-ce .*docker-compose-plugin' "$here/Dockerfile.server"
check "Docker apt arch comes from dpkg (amd64 + arm64)" grep -qF 'arch=$(dpkg --print-architecture)' "$here/Dockerfile.server"
check "daemon.json is valid JSON" python3 -c 'import json,sys; json.load(open(sys.argv[1]))' "$here/daemon.json"
check "daemon.json pins the inner address pool away from the outer network" python3 -c "import json,sys; d=json.load(open(sys.argv[1])); sys.exit(0 if d.get('default-address-pools')==[{'base':'10.200.0.0/16','size':24}] else 1)" "$here/daemon.json"
check "server.sh creates the outer network on 10.199.0.0/24" grep -q -- '--subnet 10.199.0.0/24' "$here/server.sh"
check "daemon.json allows the in-job registry" python3 -c 'import json,sys; assert "zenith-e2e-registry:5000" in json.load(open(sys.argv[1]))["insecure-registries"]' "$here/daemon.json"

echo "scripts"
check "bash -n server.sh run.sh entrypoint.sh daemon-mirror.sh" bash -n "$here/server.sh" "$here/run.sh" "$here/entrypoint.sh daemon-mirror.sh"
for f in server.sh run.sh entrypoint.sh; do check "$f is executable" test -x "$here/$f"; done
if command -v shellcheck >/dev/null 2>&1; then
  check "shellcheck clean" shellcheck -x "$here/server.sh" "$here/run.sh" "$here/entrypoint.sh" "$here/daemon-mirror.sh" "$here/server_test.sh"
else
  echo "  skip shellcheck (not installed)"
fi

echo "server.sh env"
out="$("$here/server.sh" env demo 1.2.3.4.sslip.io pw 42)"
for v in ZENITH_E2E_ENDPOINT ZENITH_E2E_DOMAIN ZENITH_E2E_ADMIN_EMAIL ZENITH_E2E_ADMIN_PASSWORD ZENITH_E2E_CA_FILE \
         ZENITH_E2E_ZEN ZENITH_E2E_INSTALL_SECONDS ZENITH_E2E_SERVER_EXEC ZENITH_E2E_REBOOT_CMD \
         ZENITH_E2E_COPY_FROM_SERVER ZENITH_E2E_CANDIDATE_VERSION ZENITH_E2E_PROBE_IMAGE ZENITH_E2E_ARTIFACTS; do
  check "$v printed exactly once" test "$(grep -c "^$v=" <<< "$out")" = 1
done
check "exactly 13 variables, no restore-only ones" test "$(grep -c '^ZENITH_E2E_[A-Z_]*=' <<< "$out")" = 13
val() { grep "^$1=" <<< "$out" | cut -d= -f2-; }
cand="$(tr -d '[:space:]' < "$here/../../VERSION")-e2e.$(git -C "$here" rev-parse --short=7 HEAD)"
check "endpoint is https://<domain>" test "$(val ZENITH_E2E_ENDPOINT)" = "https://1.2.3.4.sslip.io"
check "admin is e2e@zenith.local / pw" test "$(val ZENITH_E2E_ADMIN_EMAIL):$(val ZENITH_E2E_ADMIN_PASSWORD)" = "e2e@zenith.local:pw"
check "install seconds" test "$(val ZENITH_E2E_INSTALL_SECONDS)" = 42
check "candidate version <VERSION>-e2e.<sha7>" test "$(val ZENITH_E2E_CANDIDATE_VERSION)" = "$cand"
check "probe image in the in-job registry" test "$(val ZENITH_E2E_PROBE_IMAGE)" = "zenith-e2e-registry:5000/e2e-probe:$cand"
check "CA file in the artifacts dir" test "$(val ZENITH_E2E_CA_FILE)" = "$tmp/art/demo-ca.crt"
check "artifacts dir" test "$(val ZENITH_E2E_ARTIFACTS)" = "$tmp/art"
check "exec hook" test "$(val ZENITH_E2E_SERVER_EXEC)" = "$here/server.sh exec demo --"
check "reboot hook" test "$(val ZENITH_E2E_REBOOT_CMD)" = "$here/server.sh reboot demo"
check "copy hook" test "$(val ZENITH_E2E_COPY_FROM_SERVER)" = "$here/server.sh cp demo"
read -r -a hook <<< "$(val ZENITH_E2E_SERVER_EXEC)"
check "exec hook splits into 4 words, first is executable" test "${#hook[@]}" = 4 -a -x "${hook[0]}"
check "env with a missing argument fails" bash -c '! "$1" env demo 1.2.3.4.sslip.io pw >/dev/null 2>&1' _ "$here/server.sh"

echo "server.sh argument checks (no Docker reached)"
check "unknown subcommand exits 2" bash -c '"$1" bogus >/dev/null 2>&1; test $? = 2' _ "$here/server.sh"
check "install with a bad mode fails" bash -c '! "$1" install demo nightly >/dev/null 2>&1' _ "$here/server.sh"
check "install release without a tag fails" bash -c '! "$1" install demo release >/dev/null 2>&1' _ "$here/server.sh"
check "install without ZENITH_E2E_ZEN fails" bash -c '! "$1" install demo candidate >/dev/null 2>&1' _ "$here/server.sh"
check "exec without a command fails" bash -c '! "$1" exec demo -- >/dev/null 2>&1' _ "$here/server.sh"
check "push of a non-registry image fails" bash -c '! "$1" push demo busybox:latest >/dev/null 2>&1' _ "$here/server.sh"

echo "run.sh"
plan="$("$here/run.sh" --plan)"
check "plan prints candidate=" grep -qx "candidate=$cand previous=.*" <<< "$plan"
check "plan prints three server lines" test "$(grep -cE '^[ABC]: ' <<< "$plan")" = 3
plan="$(ZENITH_E2E_UPGRADE_FROM=v9.9.9 "$here/run.sh" --plan)"
check "ZENITH_E2E_UPGRADE_FROM overrides the previous tag" grep -q '^B: install v9.9.9 ' <<< "$plan"
check "run.sh rejects unknown arguments" bash -c '"$1" --bogus >/dev/null 2>&1; test $? = 2' _ "$here/run.sh"
git clone -q --no-tags "$here/../.." "$tmp/notags"
cp "$here/run.sh" "$here/server.sh" "$tmp/notags/infra/e2e/"
nt="$(cd "$tmp" && "$tmp/notags/infra/e2e/run.sh" 2>&1)" && ntrc=0 || ntrc=$?
check "run.sh fails when the checkout has no older release tag" test "$ntrc" = 1
check "... and says how to fix it" grep -q 'ZENITH_E2E_UPGRADE_FROM' <<< "$nt"
check "... before building or calling docker" test ! -e "$tmp/docker-calls" -a ! -e "$tmp/notags/infra/e2e/artifacts/zen"
check "a missing override tag fails too" bash -c 'ZENITH_E2E_UPGRADE_FROM=v9.9.9 "$1" >/dev/null 2>&1; test $? = 1' _ "$tmp/notags/infra/e2e/run.sh"
eval "$(sed -n '/^semver_lt() {/,/^}/p' "$here/run.sh")"
lt() { semver_lt "$1" "$2"; }
nlt() { ! semver_lt "$1" "$2"; }
check "v0.1.0-alpha.3 < 0.1.0-alpha.4" lt v0.1.0-alpha.3 0.1.0-alpha.4
check "v0.1.0-alpha.4 < 0.1.0 (pre-release before release)" lt v0.1.0-alpha.4 0.1.0
check "v0.1.0-alpha.10 > 0.1.0-alpha.9 (numeric identifiers)" nlt v0.1.0-alpha.10 0.1.0-alpha.9
check "v0.1.0 is not < 0.1.0" nlt v0.1.0 0.1.0
check "v0.1.0 is not < 0.1.0-alpha.4" nlt v0.1.0 0.1.0-alpha.4
check "v2.0.0-alpha.3 is not < 0.1.0" nlt v2.0.0-alpha.3 0.1.0
check "v0.9.0 < 0.10.0" lt v0.9.0 0.10.0

echo "probe"
if command -v go >/dev/null 2>&1; then
  check "probe go vet" bash -c 'cd "$1" && go vet ./...' _ "$here/probe"
else
  echo "  skip probe go vet (go not installed)"
fi
check "probe go.mod pins pgx like services/api" test \
  "$(grep -oE 'github.com/jackc/pgx/v5 v[^ ]+' "$here/probe/go.mod")" = \
  "$(grep -oE 'github.com/jackc/pgx/v5 v[^ ]+' "$here/../../services/api/go.mod")"

echo "candidate tag (VERSION=<candidate> on top of HEAD)"
git clone -q --bare "$here/../.." "$tmp/bare.git"
"$here/server.sh" tag-candidate "$tmp/bare.git"
check "VERSION at v<candidate> is the candidate version" test "$(git -C "$tmp/bare.git" show "v$cand:VERSION")" = "$cand"
check "v<candidate> is a child of HEAD" test "$(git -C "$tmp/bare.git" rev-parse "v$cand^")" = "$(git -C "$here" rev-parse HEAD)"
check "v<candidate> changes only VERSION" test "$(git -C "$tmp/bare.git" diff --name-only "v$cand^" "v$cand")" = VERSION
check "the checkout got no candidate tag" test -z "$(git -C "$here" tag --list "v$cand")"
check "no helper ref left in the bare copy" test -z "$(git -C "$tmp/bare.git" for-each-ref refs/e2e)"
"$here/server.sh" tag-candidate "$tmp/bare.git"
check "tagging twice is fine" test "$(git -C "$tmp/bare.git" show "v$cand:VERSION")" = "$cand"

echo "self-signed seeding for release installs"
printf '#!/usr/bin/env bash\necho old installer\n' > "$tmp/old-install.sh"
printf '#!/usr/bin/env bash\nTLS_MODE="${ZENITH_TLS_MODE:-http01}"\n' > "$tmp/new-install.sh"
check "install.sh without ZENITH_TLS_MODE needs seeding" "$here/server.sh" needs-tls-seed "$tmp/old-install.sh"
check "install.sh with ZENITH_TLS_MODE does not" bash -c '! "$1" needs-tls-seed "$2"' _ "$here/server.sh" "$tmp/new-install.sh"
if command -v openssl >/dev/null 2>&1; then
  inst="$tmp/opt-zenith"; certs="$inst/traefik/certs"; mkdir -p "$inst/traefik"
  printf 'certificatesResolvers:\n  le: {}\n' > "$inst/traefik/traefik.yml"; cp "$inst/traefik/traefik.yml" "$tmp/traefik.yml.orig"
  "$here/server.sh" seed-tls "$inst" 1.2.3.4.sslip.io >/dev/null 2>&1
  check "leaf verifies against the CA" bash -c 'openssl verify -CAfile "$1" "$2" >/dev/null 2>&1' _ "$certs/zenith-ca.crt" "$certs/zenith.crt"
  sans="$(openssl x509 -in "$certs/zenith.crt" -noout -ext subjectAltName 2>/dev/null)"
  check "leaf SAN has the domain" grep -q 'DNS:1.2.3.4.sslip.io' <<< "$sans"
  check "leaf SAN has the wildcard" grep -q 'DNS:\*.1.2.3.4.sslip.io' <<< "$sans"
  check "leaf is for serverAuth" bash -c 'openssl x509 -in "$1" -noout -ext extendedKeyUsage | grep -q "TLS Web Server Authentication"' _ "$certs/zenith.crt"
  check "leaf file carries the CA (chain)" test "$(grep -c 'BEGIN CERTIFICATE' "$certs/zenith.crt")" = 2
  check "CA is CN=Zenith Local CA, CA:TRUE" bash -c 'openssl x509 -in "$1" -noout -subject -ext basicConstraints | tr -d " " | grep -q "CN=ZenithLocalCA" && openssl x509 -in "$1" -noout -ext basicConstraints | grep -q "CA:TRUE"' _ "$certs/zenith-ca.crt"
  check "keys are 0600" test "$(stat -c %a "$certs/zenith.key" "$certs/zenith-ca.key" | sort -u)" = 600
  check "certificates are 0644 (the api container reads the CA)" test "$(stat -c %a "$certs/zenith.crt" "$certs/zenith-ca.crt" | sort -u)" = 644
  check "certs.yml is byte-identical to install.sh's (which matches the Go installer's)" \
    cmp -s "$inst/traefik/dynamic/certs.yml" <(sed -n "/^  cat > traefik\/dynamic\/certs.yml <<'EOF'\$/,/^EOF\$/{//!p}" "$here/../scripts/install.sh")
  check "certs.yml sets the default certificate" grep -q 'defaultCertificate:' "$inst/traefik/dynamic/certs.yml"
  check "certs.yml points at /etc/traefik/certs" test "$(grep -c -e 'certFile: /etc/traefik/certs/zenith.crt' -e 'keyFile: /etc/traefik/certs/zenith.key' "$inst/traefik/dynamic/certs.yml")" = 4
  check "traefik.yml left alone" cmp -s "$inst/traefik/traefik.yml" "$tmp/traefik.yml.orig"
  ca1="$(openssl x509 -in "$certs/zenith-ca.crt" -noout -fingerprint)"
  "$here/server.sh" seed-tls "$inst" 1.2.3.4.sslip.io >/dev/null 2>&1
  check "seeding again keeps the CA" test "$(openssl x509 -in "$certs/zenith-ca.crt" -noout -fingerprint)" = "$ca1"
  check "server.sh ships the seeding with the same openssl commands as write_selfsigned_tls" \
    grep -q 'basicConstraints=critical,CA:TRUE" -addext "keyUsage=critical,keyCertSign,cRLSign' "$here/server.sh"
else
  echo "  skip seeding checks (openssl not installed)"
fi

check "nothing called docker" test ! -e "$tmp/docker-calls"

echo "registry mirror (daemon-mirror.sh, no docker)"
m="$tmp/mirror"; mkdir -p "$m"
cp "$here/daemon.json" "$m/d.json"
check "mirror added to daemon.json" bash "$here/daemon-mirror.sh" "$m/d.json" https://mirror.example
check "daemon.json still valid JSON" python3 -m json.tool "$m/d.json"
check "insecure registry kept" grep -q 'zenith-e2e-registry:5000' "$m/d.json"
check "mirror present" grep -q '"https://mirror.example"' "$m/d.json"
bash "$here/daemon-mirror.sh" "$m/d.json" https://mirror.example
check "adding the same mirror twice keeps one entry" test "$(grep -o 'https://mirror.example' "$m/d.json" | wc -l)" = 1
bash "$here/daemon-mirror.sh" "$m/d.json" https://second.example
check "a second mirror is appended" python3 -c "import json,sys; d=json.load(open(sys.argv[1])); sys.exit(0 if d['registry-mirrors']==['https://mirror.example','https://second.example'] else 1)" "$m/d.json"
printf '{}' > "$m/e.json"; bash "$here/daemon-mirror.sh" "$m/e.json" https://mirror.example
check "an empty {} config gets the key (valid JSON)" python3 -m json.tool "$m/e.json"
check "a non-http mirror is refused" bash -c '! "$1" "$2" ftp://bad >/dev/null 2>&1' _ "$here/daemon-mirror.sh" "$m/d.json"
check "entrypoint calls the mirror script when the variable is set" grep -q 'ZENITH_E2E_REGISTRY_MIRROR' "$here/entrypoint.sh"
check "server.sh passes the mirror variable into the server" grep -q 'ZENITH_E2E_REGISTRY_MIRROR=' "$here/server.sh"
check "Dockerfile.server ships the mirror script" grep -q 'daemon-mirror.sh' "$here/Dockerfile.server"
check "run.sh passes GOPROXY/GOSUMDB/npm registry into image builds" grep -q 'GOPROXY GOSUMDB npm_config_registry COREPACK_NPM_REGISTRY' "$here/run.sh"
check "api Dockerfile declares ARG GOPROXY" grep -q '^ARG GOPROXY' "$here/../../services/api/Dockerfile"
check "probe Dockerfile declares ARG GOPROXY" grep -q '^ARG GOPROXY' "$here/probe/Dockerfile"
check "web Dockerfile declares ARG npm_config_registry" grep -q '^ARG npm_config_registry' "$here/../../apps/web/Dockerfile"

echo "reboot refuses a changed IP (scripted fake docker)"
mkdir -p "$tmp/fake2"
cat > "$tmp/fake2/docker" <<'EOF'
#!/bin/sh
case "$1" in inspect) echo "$FAKE_IP";; esac
exit 0
EOF
chmod +x "$tmp/fake2/docker"
mkdir -p "$tmp/art"; echo 10.9.0.2 > "$tmp/art/demo.ip"
rb="$(PATH="$tmp/fake2:$PATH" FAKE_IP=10.9.0.7 timeout 20 "$here/server.sh" reboot demo 2>&1)" && rbrc=0 || rbrc=$?
check "reboot on a new IP fails" test "$rbrc" != 0 -a "$rbrc" != 124
check "the message names both IPs" bash -c 'grep -q 10.9.0.7 <<< "$1" && grep -q 10.9.0.2 <<< "$1"' _ "$rb"
if [ -e "$tmp/docker-calls" ]; then cat "$tmp/docker-calls"; fi

if [ "$fails" -gt 0 ]; then echo "e2e_harness_test: $fails failure(s)"; exit 1; fi
echo "e2e_harness_test: ok"
