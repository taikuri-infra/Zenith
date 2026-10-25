#!/usr/bin/env bash
# infra/e2e/run.sh — the whole E2E run: build the candidate, then
#   cluster A: fresh candidate install → TestInstall TestDeployWithDatabaseAndDomain TestReboot TestBackup TestUpgrade
#   cluster C: fresh candidate install + A's backup archive → TestBackupRestore
# Used by .github/workflows/e2e.yml. `run.sh --plan` prints the sequence only.
# Needs docker, helm and kubectl on PATH; downloads k3d (pinned, checksum-verified) when absent.
# Env: ZENITH_E2E_ARTIFACTS (default infra/e2e/artifacts), ZENITH_E2E_KEEP=1 (keep the clusters).
#      Restricted networks: GOPROXY/GOSUMDB/npm_config_registry/COREPACK_NPM_REGISTRY are
#      passed into the image builds.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; repo="$(cd "$here/../.." && pwd)"; C="$here/cluster.sh"
export ZENITH_E2E_ARTIFACTS="${ZENITH_E2E_ARTIFACTS:-$here/artifacts}"
IMAGE_REPO=e2e.local
CAND="$("$C" candidate-version)"

log() { printf '\033[36m[e2e]\033[0m %s\n' "$*"; }

if [ "${1:-}" = "--plan" ]; then
  echo "candidate=$CAND"
  echo "A: install candidate → TestInstall TestDeployWithDatabaseAndDomain TestReboot TestBackup TestUpgrade"
  echo "C: install candidate + restore A's archive → TestBackupRestore"
  exit 0
fi
[ "$#" -eq 0 ] || { echo "usage: run.sh [--plan]" >&2; exit 2; }

mkdir -p "$ZENITH_E2E_ARTIFACTS"
images=("$IMAGE_REPO/zenith-api:$CAND" "$IMAGE_REPO/zenith-web:$CAND" "$IMAGE_REPO/e2e-probe:$CAND")
cleanup() {
  local rc=$?
  for s in a c; do
    if [ -s "$ZENITH_E2E_ARTIFACTS/$s.kubeconfig" ]; then "$C" logs "$s" || true; fi
    if [ "${ZENITH_E2E_KEEP:-0}" != 1 ]; then "$C" rm "$s" || true; fi
  done
  # Persistent runners keep their Docker daemon between runs: untag this run's images.
  if [ "${ZENITH_E2E_KEEP:-0}" != 1 ]; then docker image rm -f "${images[@]}" >/dev/null 2>&1 || true; fi
  if [ "$rc" = 0 ]; then log "E2E passed for $CAND"; else log "E2E FAILED for $CAND (rc=$rc); artifacts: $ZENITH_E2E_ARTIFACTS"; fi
}
trap cleanup EXIT

log "building zen and images for $CAND"
(cd "$repo/cli" && CGO_ENABLED=0 go build -ldflags "-X github.com/dotechhq/zenith/cli/cmd/version.Version=$CAND" -o "$ZENITH_E2E_ARTIFACTS/zen" .)
export ZENITH_E2E_ZEN="$ZENITH_E2E_ARTIFACTS/zen"
mirror_args=()
for v in GOPROXY GOSUMDB npm_config_registry COREPACK_NPM_REGISTRY; do
  if [ -n "${!v:-}" ]; then mirror_args+=(--build-arg "$v=${!v}"); fi
done
# build_image NAME TAG [docker build args...] — quiet on success, the log tail on failure.
build_image() {
  local name="$1" tag="$2" logf; shift 2; logf="$ZENITH_E2E_ARTIFACTS/build-$name.log"
  log "building $tag (log: $logf)"
  if ! docker build -t "$tag" "${mirror_args[@]}" "$@" > "$logf" 2>&1; then
    log "build of $name FAILED; last lines of $logf:"; tail -n 40 "$logf"; return 1
  fi
}
build_image zenith-api "${images[0]}" --build-arg "VERSION=$CAND" -f "$repo/services/api/Dockerfile" "$repo"
build_image zenith-web "${images[1]}" --build-arg "VERSION=$CAND" --build-arg NEXT_PUBLIC_ZENITH_MODE=standalone -f "$repo/apps/web/Dockerfile" "$repo"
build_image e2e-probe "${images[2]}" "$here/probe"

# gotest ENVFILE ARGS... — runs the suite with exactly the variables of ENVFILE (one
# KEY=VALUE per line; values may contain spaces, so no word splitting).
gotest() {
  local envfile="$1" kv=(); shift
  mapfile -t kv < <(grep -E '^[A-Z_][A-Z0-9_]*=' "$envfile")
  (cd "$repo/cli" && env "${kv[@]}" go test -tags e2e -count=1 -timeout 40m -v ./e2e/ "$@" 2>&1 \
    | tee "$ZENITH_E2E_ARTIFACTS/$(basename "$envfile" .env)-test.log")
}

log "cluster A: fresh candidate install"
"$C" new a; "$C" install a
gotest "$ZENITH_E2E_ARTIFACTS/a.env" -run 'TestInstall|TestDeployWithDatabaseAndDomain|TestReboot|TestBackup$|TestUpgrade'
A_PW="$(grep '^ZENITH_E2E_ADMIN_PASSWORD=' "$ZENITH_E2E_ARTIFACTS/a.env" | cut -d= -f2-)"
# shellcheck disable=SC2012 # archive names are zenith-backup-<version>-<timestamp>.tar.gz
BACKUP="$(ls -t "$ZENITH_E2E_ARTIFACTS"/zenith-backup-*.tar.gz 2>/dev/null | head -1 || true)"
[ -n "$BACKUP" ] || { log "TestBackup left no zenith-backup-*.tar.gz in $ZENITH_E2E_ARTIFACTS"; exit 1; }
"$C" logs a; "$C" rm a

log "cluster C: fresh candidate install + restore of $(basename "$BACKUP")"
"$C" new c; "$C" install c
{ cat "$ZENITH_E2E_ARTIFACTS/c.env"
  echo "ZENITH_E2E_BACKUP_FILE=$BACKUP"
  echo "ZENITH_E2E_EXPECT_ADMIN_EMAIL=e2e@zenith.local"
  echo "ZENITH_E2E_EXPECT_ADMIN_PASSWORD=$A_PW"; } > "$ZENITH_E2E_ARTIFACTS/c-restore.env"
gotest "$ZENITH_E2E_ARTIFACTS/c-restore.env" -run 'TestBackupRestore'
log "E2E complete for $CAND"
