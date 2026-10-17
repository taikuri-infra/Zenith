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
check "daemon.json allows the in-job registry" python3 -c 'import json,sys; assert "zenith-e2e-registry:5000" in json.load(open(sys.argv[1]))["insecure-registries"]' "$here/daemon.json"

echo "scripts"
check "bash -n server.sh run.sh entrypoint.sh" bash -n "$here/server.sh" "$here/run.sh" "$here/entrypoint.sh"
for f in server.sh run.sh entrypoint.sh; do check "$f is executable" test -x "$here/$f"; done
if command -v shellcheck >/dev/null 2>&1; then
  check "shellcheck clean" shellcheck -x "$here/server.sh" "$here/run.sh" "$here/entrypoint.sh" "$here/server_test.sh"
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

check "nothing called docker" test ! -e "$tmp/docker-calls"
if [ -e "$tmp/docker-calls" ]; then cat "$tmp/docker-calls"; fi

if [ "$fails" -gt 0 ]; then echo "e2e_harness_test: $fails failure(s)"; exit 1; fi
echo "e2e_harness_test: ok"
