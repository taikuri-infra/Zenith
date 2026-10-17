#!/usr/bin/env bash
# infra/e2e/run.sh — the whole E2E run: build the candidate, then
#   server A: fresh candidate install  → TestInstall TestDeployWithDatabaseAndDomain TestReboot TestBackup
#   server C: fresh candidate + A's backup archive → TestBackupRestore
#   server B: previous release, upgraded to the candidate → TestUpgrade
# Used by .github/workflows/e2e.yml and by act. `run.sh --plan` prints the sequence only.
# Env: ZENITH_E2E_ARTIFACTS (default infra/e2e/artifacts), ZENITH_E2E_UPGRADE_FROM (tag),
#      ZENITH_E2E_KEEP=1 (keep registry and network at the end).
#      Restricted networks: GOPROXY/GOSUMDB/npm_config_registry/COREPACK_NPM_REGISTRY are
#      passed into the image builds; ZENITH_E2E_REGISTRY_MIRROR (e.g. https://mirror.gcr.io)
#      becomes the Docker Hub mirror of every E2E server's dockerd (server.sh new).
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; repo="$(cd "$here/../.." && pwd)"; S="$here/server.sh"
export ZENITH_E2E_ARTIFACTS="${ZENITH_E2E_ARTIFACTS:-$here/artifacts}"
REG=zenith-e2e-registry:5000
VERSION="$(tr -d '[:space:]' < "$repo/VERSION")"
CAND="$("$S" candidate-version)"

log() { printf '\033[36m[e2e]\033[0m %s\n' "$*"; }

# semver_lt A B — true when version A sorts before B (a pre-release sorts before its release).
semver_lt() {
  local a="${1#v}" b="${2#v}" ac bc ap bp i x y
  ac="${a%%-*}"; bc="${b%%-*}"
  ap=""; bp=""
  if [ "$ac" != "$a" ]; then ap="${a#*-}"; fi
  if [ "$bc" != "$b" ]; then bp="${b#*-}"; fi
  IFS=. read -r -a x <<< "$ac"; IFS=. read -r -a y <<< "$bc"
  for i in 0 1 2; do
    if [ "${x[i]:-0}" -lt "${y[i]:-0}" ]; then return 0; fi
    if [ "${x[i]:-0}" -gt "${y[i]:-0}" ]; then return 1; fi
  done
  [ "$ap" = "$bp" ] && return 1
  [ -z "$ap" ] && return 1     # A is the release, B a pre-release of it
  [ -z "$bp" ] && return 0
  IFS=. read -r -a x <<< "$ap"; IFS=. read -r -a y <<< "$bp"
  for ((i = 0; i < ${#x[@]} || i < ${#y[@]}; i++)); do
    [ "$i" -ge "${#x[@]}" ] && return 0
    [ "$i" -ge "${#y[@]}" ] && return 1
    if [[ "${x[i]}" =~ ^[0-9]+$ && "${y[i]}" =~ ^[0-9]+$ ]]; then
      if [ "${x[i]}" -lt "${y[i]}" ]; then return 0; fi
      if [ "${x[i]}" -gt "${y[i]}" ]; then return 1; fi
    else
      if [[ "${x[i]}" < "${y[i]}" ]]; then return 0; fi
      if [[ "${x[i]}" > "${y[i]}" ]]; then return 1; fi
    fi
  done
  return 1
}

# The newest release tag (v<semver>, no e2e/candidate tags) older than VERSION.
previous_tag() {
  local t
  while read -r t; do
    [[ "$t" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || continue
    case "$t" in *e2e*) continue;; esac
    if semver_lt "$t" "$VERSION"; then echo "$t"; return 0; fi
  done < <(git -C "$repo" -c versionsort.suffix=- tag --list 'v*' --sort=-v:refname)
}
PREV="${ZENITH_E2E_UPGRADE_FROM:-$(previous_tag)}"

if [ "${1:-}" = "--plan" ]; then
  echo "candidate=$CAND previous=${PREV:-none}"
  echo "A: install candidate → TestInstall TestDeployWithDatabaseAndDomain TestReboot TestBackup"
  echo "C: install candidate + restore A's archive → TestBackupRestore"
  echo "B: install ${PREV:-<none: the run fails, no release tag older than $VERSION>} then upgrade to $CAND → TestUpgrade"
  exit 0
fi
[ "$#" -eq 0 ] || { echo "usage: run.sh [--plan]" >&2; exit 2; }
# No previous release means a shallow or tag-less checkout (e2e.yml uses fetch-depth: 0):
# fail before building anything rather than silently skipping the upgrade criterion.
if [ -z "$PREV" ]; then
  echo "run.sh: no release tag older than $VERSION in this checkout (shallow clone without tags?);" >&2
  echo "        fetch tags (git fetch --tags) or set ZENITH_E2E_UPGRADE_FROM=v<previous release>" >&2
  exit 1
fi
git -C "$repo" rev-parse -q --verify "refs/tags/$PREV" >/dev/null || { echo "run.sh: tag $PREV not found locally" >&2; exit 1; }

mkdir -p "$ZENITH_E2E_ARTIFACTS"
cleanup() {
  local rc=$?
  for s in a b c; do
    if docker inspect "zenith-e2e-$s" >/dev/null 2>&1; then "$S" logs "$s" || true; "$S" rm "$s"; fi
  done
  "$S" rm pusher
  if [ "${ZENITH_E2E_KEEP:-0}" != 1 ]; then
    "$S" down
    # Persistent runners keep their Docker daemon between runs: untag this run's images so
    # they do not pile up (the build cache, not the tags, is what keeps the next run fast).
    docker image rm -f "${images[@]:-}" >/dev/null 2>&1 || true
  fi
  if [ "$rc" = 0 ]; then log "E2E passed for $CAND"; else log "E2E FAILED for $CAND (rc=$rc); artifacts: $ZENITH_E2E_ARTIFACTS"; fi
}
trap cleanup EXIT

log "building zen and images for $CAND"
(cd "$repo/cli" && CGO_ENABLED=0 go build -ldflags "-X github.com/dotechhq/zenith/cli/cmd/version.Version=$CAND" -o "$ZENITH_E2E_ARTIFACTS/zen" .)
export ZENITH_E2E_ZEN="$ZENITH_E2E_ARTIFACTS/zen"
images=("$REG/zenith-api:$CAND" "$REG/zenith-web:$CAND" "$REG/e2e-probe:$CAND")
# Restricted networks: the caller's module/package mirrors reach the image builds as
# build args (the Dockerfiles declare them); unset ones are simply not passed.
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

# Push to the in-job registry. Fast path: the host daemon already trusts <registry ip>:5000.
# Default: load the images into a throwaway server, whose dockerd resolves and trusts the
# registry name (daemon.json), and push from there (see server.sh push).
"$S" new pusher   # also creates the network and the registry
REGIP="$(docker inspect -f '{{with index .NetworkSettings.Networks "zenith-e2e"}}{{.IPAddress}}{{end}}' zenith-e2e-registry)"
if grep -qF "\"$REGIP:5000\"" <<<"$(docker info --format '{{json .RegistryConfig.IndexConfigs}}' 2>/dev/null || true)"; then
  log "pushing directly to $REGIP:5000"
  for img in "${images[@]}"; do d="$REGIP:5000/${img#"$REG"/}"; docker tag "$img" "$d"; docker push -q "$d" >/dev/null; done
else
  "$S" push pusher "${images[@]}"
fi
"$S" rm pusher

# gotest ENVFILE ARGS... — runs the suite with exactly the variables of ENVFILE (one
# KEY=VALUE per line; values may contain spaces, so no word splitting).
gotest() {
  local envfile="$1" kv=(); shift
  mapfile -t kv < <(grep -E '^[A-Z_][A-Z0-9_]*=' "$envfile")
  (cd "$repo/cli" && env "${kv[@]}" go test -tags e2e -count=1 -timeout 40m -v ./e2e/ "$@" 2>&1 \
    | tee "$ZENITH_E2E_ARTIFACTS/$(basename "$envfile" .env)-test.log")
}

log "server A: fresh candidate install"
"$S" new a; "$S" install a candidate
gotest "$ZENITH_E2E_ARTIFACTS/a.env" -run 'TestInstall|TestDeployWithDatabaseAndDomain|TestReboot|TestBackup$'
A_PW="$(grep '^ZENITH_E2E_ADMIN_PASSWORD=' "$ZENITH_E2E_ARTIFACTS/a.env" | cut -d= -f2-)"
# shellcheck disable=SC2012 # archive names are zenith-backup-<timestamp>.tar.gz
BACKUP="$(ls -t "$ZENITH_E2E_ARTIFACTS"/zenith-backup-*.tar.gz 2>/dev/null | head -1 || true)"
[ -n "$BACKUP" ] || { log "TestBackup left no zenith-backup-*.tar.gz in $ZENITH_E2E_ARTIFACTS"; exit 1; }
"$S" logs a; "$S" rm a

log "server C: fresh candidate install + restore of $(basename "$BACKUP")"
"$S" new c; "$S" install c candidate
docker cp -q "$BACKUP" "zenith-e2e-c:/opt/zenith/restore.tar.gz"
{ cat "$ZENITH_E2E_ARTIFACTS/c.env"
  echo "ZENITH_E2E_BACKUP_FILE=/opt/zenith/restore.tar.gz"
  echo "ZENITH_E2E_EXPECT_ADMIN_EMAIL=e2e@zenith.local"
  echo "ZENITH_E2E_EXPECT_ADMIN_PASSWORD=$A_PW"; } > "$ZENITH_E2E_ARTIFACTS/c-restore.env"
gotest "$ZENITH_E2E_ARTIFACTS/c-restore.env" -run 'TestBackupRestore'
"$S" logs c; "$S" rm c

log "server B: install $PREV then upgrade to $CAND"
"$S" new b; "$S" install b release "$PREV"
gotest "$ZENITH_E2E_ARTIFACTS/b.env" -run 'TestUpgrade'
log "E2E complete for $CAND"
