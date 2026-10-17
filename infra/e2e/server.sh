#!/usr/bin/env bash
# infra/e2e/server.sh — the E2E harness's "fresh server" driver.
#
# A server is a privileged Docker-in-Docker container (zenith-e2e-<name>, image
# zenith-e2e-server:local built from Dockerfile.server) on the network zenith-e2e, next to an
# in-job registry (zenith-e2e-registry:5000). install.sh runs inside it exactly as a user would
# run it. The Go suite (cli/e2e) never calls docker: it reads <artifacts>/<name>.env and drives
# the server through the three hooks this script provides (exec, reboot, cp).
#
# Usage:
#   server.sh new <name>                          start a fresh server (network + registry once)
#   server.sh install <name> candidate            install this checkout as v<candidate version>
#   server.sh install <name> release <tag>        install a released tag (e.g. v0.1.0)
#   server.sh reboot <name>                       restart the server; 0 once /api/v1/version answers again
#   server.sh exec <name> -- <cmd...>             run a command on the server, cwd /opt/zenith
#   server.sh cp <name> <remote> <local>          copy a file out of the server
#   server.sh push <name> <image...>              push host images to the registry via the server
#   server.sh logs <name>                         collect dockerd + compose logs into artifacts
#   server.sh rm <name>                           remove the server and its docker volume
#   server.sh down                                remove every server, the registry and the network
#   server.sh env <name> <domain> <password> <s>  print the suite env (used by install)
#   server.sh ip <name>                           print the server's IP on zenith-e2e
set -euo pipefail

NET=zenith-e2e
REG=zenith-e2e-registry
IMG=zenith-e2e-server:local
INSTALL_DIR=/opt/zenith
ADMIN_EMAIL=e2e@zenith.local
HEALTH_TIMEOUT=300

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
ART="${ZENITH_E2E_ARTIFACTS:-$here/artifacts}"

log() { printf '\033[36m[e2e]\033[0m %s\n' "$*" >&2; }
die() { printf '\033[31m[e2e]\033[0m %s\n' "$*" >&2; exit 1; }
cname() { [ -n "${1:-}" ] || die "missing server name"; echo "zenith-e2e-$1"; }
candidate_version() { echo "$(tr -d '[:space:]' < "$repo/VERSION")-e2e.$(git -C "$repo" rev-parse --short=7 HEAD)"; }

# ---- shared infrastructure -------------------------------------------------------------

# The id of the container this script runs in (act job container), or nothing on a plain host.
self_container() {
  [ -f /.dockerenv ] || return 0
  local id
  id="$(sed -nE 's#.*/containers/([0-9a-f]{64})/.*#\1#p' /proc/self/mountinfo 2>/dev/null | head -1)"
  if [ -n "$id" ]; then echo "$id"; else hostname; fi
}

ensure_network() {
  docker network inspect "$NET" >/dev/null 2>&1 || docker network create "$NET" >/dev/null
  # Under act the job itself is a container: join it to the network so container IPs (and
  # the registry name) are reachable from the suite. Ports are never published on the host.
  if [ -n "${ACT:-}" ]; then
    local self; self="$(self_container)"
    if [ -n "$self" ]; then docker network connect "$NET" "$self" >/dev/null 2>&1 || true; fi
  fi
}

ensure_registry() {
  if docker inspect "$REG" >/dev/null 2>&1; then
    [ "$(docker inspect -f '{{.State.Running}}' "$REG")" = true ] || docker start "$REG" >/dev/null
  else
    docker run -d --name "$REG" --network "$NET" --restart unless-stopped registry:2 >/dev/null
  fi
}

# Built once per job; later calls are a cache hit (and pick up Dockerfile changes locally).
ensure_image() {
  log "building $IMG (cached after the first time)"
  docker build -q -t "$IMG" -f "$here/Dockerfile.server" "$here" >/dev/null
}

ip_of() {
  local ip
  ip="$(docker inspect -f "{{with index .NetworkSettings.Networks \"$NET\"}}{{.IPAddress}}{{end}}" "$(cname "$1")" 2>/dev/null || true)"
  [ -n "$ip" ] || die "server $1 is not running on network $NET"
  echo "$ip"
}

# ---- lifecycle -------------------------------------------------------------------------

cmd_new() {
  local name="${1:-}" c; c="$(cname "$name")"
  ensure_network; ensure_registry; ensure_image
  docker rm -fv "$c" >/dev/null 2>&1 || true
  docker run -d --privileged --name "$c" --network "$NET" --hostname "$name" "$IMG" >/dev/null
  local deadline=$((SECONDS + 120))
  while [ "$SECONDS" -lt "$deadline" ]; do
    if docker exec "$c" docker info >/dev/null 2>&1; then log "server $name up at $(ip_of "$name")"; return 0; fi
    sleep 2
  done
  log "dockerd in $c did not come up within 120s"; cmd_logs "$name"; return 1
}

cmd_rm() { docker rm -fv "$(cname "${1:-}")" >/dev/null 2>&1 || true; }

cmd_down() {
  local ids
  ids="$(docker ps -aq --filter "name=^zenith-e2e-")"
  # shellcheck disable=SC2086 # ids is a whitespace-separated list of container ids
  [ -z "$ids" ] || docker rm -fv $ids >/dev/null
  if [ -n "${ACT:-}" ]; then
    local self; self="$(self_container)"
    if [ -n "$self" ]; then docker network disconnect -f "$NET" "$self" >/dev/null 2>&1 || true; fi
  fi
  docker network rm "$NET" >/dev/null 2>&1 || true
}

# The IP recorded at install time ($ART/<name>.ip); the install domain is <ip>.sslip.io.
# Falls back to the current IP for a server that was never installed.
recorded_ip() {
  if [ -s "$ART/$1.ip" ]; then cat "$ART/$1.ip"; else ip_of "$1"; fi
}

# Polls https://<recorded ip>.sslip.io/api/v1/version through that IP (no public DNS needed) for at
# most $2 seconds. On timeout it collects the server's logs and fails.
wait_health() {
  local name="$1" limit="$2" ip host deadline
  ip="$(recorded_ip "$name")"; host="$ip.sslip.io"; deadline=$((SECONDS + limit))
  while [ "$SECONDS" -lt "$deadline" ]; do
    if curl -skf --max-time 3 --resolve "$host:443:$ip" "https://$host/api/v1/version" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  log "$name: /api/v1/version not answering after ${limit}s"; cmd_logs "$name"; return 1
}

cmd_reboot() {
  local name="${1:-}" before after
  before="$(recorded_ip "$name")"
  log "rebooting $name"
  docker restart -t 30 "$(cname "$name")" >/dev/null
  # The install domain is <ip>.sslip.io: a server that comes back on another IP no longer
  # matches its own domain, certificates or .env, so that is a failure, not a retry.
  after="$(ip_of "$name")"
  if [ "$after" != "$before" ]; then
    log "$name came back on IP $after, but it was installed on $before (domain $before.sslip.io)"
    cmd_logs "$name"; return 1
  fi
  wait_health "$name" "$HEALTH_TIMEOUT"
  log "$name is back"
}

# ---- hooks the suite calls -------------------------------------------------------------

cmd_exec() {
  local name="${1:-}"; shift || true
  if [ "${1:-}" = "--" ]; then shift; fi
  [ "$#" -gt 0 ] || die "usage: server.sh exec <name> -- <cmd...>"
  # cwd is the install directory; before install.sh has created it, the server's /.
  # shellcheck disable=SC2016 # expanded by the server's sh
  docker exec "$(cname "$name")" sh -c 'cd "$0" 2>/dev/null || cd /; exec "$@"' "$INSTALL_DIR" "$@"
}

cmd_cp() {
  [ "$#" -eq 3 ] || die "usage: server.sh cp <name> <remote> <local>"
  docker cp -q "$(cname "$1"):$2" "$3"
}

cmd_logs() {
  local name="${1:-}" c d; c="$(cname "$name")"; d="$ART/$name"
  mkdir -p "$d"
  docker logs "$c" > "$d/dockerd.log" 2>&1 || true
  docker exec -w "$INSTALL_DIR" "$c" docker compose logs --no-color --timestamps > "$d/compose.log" 2>&1 || true
  docker exec "$c" docker ps -a > "$d/docker-ps.txt" 2>&1 || true
  log "logs of $name in $d"
}

# ---- images ----------------------------------------------------------------------------

# Pushes images that exist on the job host into the in-job registry. The job host's daemon
# cannot resolve zenith-e2e-registry (and does not trust it as an insecure registry), so the
# images are loaded into a server's dockerd, which resolves the name on the network and
# trusts it (daemon.json), and pushed from there. Removed again afterwards so that server
# stays fresh. Image names must already be zenith-e2e-registry:5000/<repo>:<tag>.
cmd_push() {
  local name="${1:-}" c img; c="$(cname "$name")"; shift || true
  [ "$#" -gt 0 ] || die "usage: server.sh push <name> <image...>"
  for img in "$@"; do
    case "$img" in "$REG:5000/"*) ;; *) die "image $img must be named $REG:5000/<repo>:<tag>";; esac
  done
  docker save "$@" | docker exec -i "$c" docker load -q >/dev/null
  for img in "$@"; do
    log "push $img"
    docker exec "$c" docker push -q "$img" >/dev/null
  done
  docker exec "$c" docker image rm "$@" >/dev/null 2>&1 || true
}

# ---- install ---------------------------------------------------------------------------

# tag_candidate BARE — tags v<candidate> in the bare repo BARE at a commit on top of the
# checkout's HEAD whose only change is VERSION=<candidate>, so the installer pins
# ZENITH_VERSION=<candidate> (the tag the candidate images carry) and /version, backups and
# `zen upgrade` all agree on it. Plumbing only: the checkout and its refs stay untouched.
tag_candidate() {
  local bare="$1" cand base blob tree commit idx
  cand="$(candidate_version)"; base="$(git -C "$repo" rev-parse HEAD)"
  # Make sure HEAD's commit is in the bare copy (a detached CI HEAD is not on any branch).
  git -C "$repo" push -q --no-verify --force "$bare" "$base:refs/e2e/base"
  idx="$(mktemp)"; rm -f "$idx"
  blob="$(printf '%s\n' "$cand" | git -C "$bare" hash-object -w --stdin)"
  GIT_INDEX_FILE="$idx" git -C "$bare" read-tree "$base"
  GIT_INDEX_FILE="$idx" git -C "$bare" update-index --cacheinfo "100644,$blob,VERSION"
  tree="$(GIT_INDEX_FILE="$idx" git -C "$bare" write-tree)"
  rm -f "$idx"
  commit="$(git -C "$bare" -c user.name=zenith-e2e -c user.email=e2e@zenith.local \
    commit-tree "$tree" -p "$base" -m "e2e: VERSION=$cand")"
  git -C "$bare" tag -f "v$cand" "$commit" >/dev/null
  git -C "$bare" update-ref -d refs/e2e/base
}

# needs_tls_seed INSTALL_SH — true when that install.sh predates ZENITH_TLS_MODE (it cannot
# write the self-signed CA itself, so the harness seeds it after the install).
needs_tls_seed() { ! grep -q 'ZENITH_TLS_MODE' "$1"; }

# seed_selfsigned_tls DOMAIN — run in the install directory. The same CA, leaf and file-provider
# config install.sh writes for ZENITH_TLS_MODE=selfsigned (write_selfsigned_tls), except that
# traefik/traefik.yml is left alone: its ACME resolver keeps failing and Traefik serves the
# default certificate, which is this leaf. Shipped into the server with `declare -f`.
seed_selfsigned_tls() {
  local domain="$1" certs="traefik/certs"
  mkdir -p "$certs" traefik/dynamic
  if [ ! -s "$certs/zenith-ca.crt" ]; then
    openssl ecparam -name prime256v1 -genkey -noout -out "$certs/zenith-ca.key"
    openssl req -x509 -new -key "$certs/zenith-ca.key" -sha256 -days 3650 -subj "/CN=Zenith Local CA" \
      -addext "basicConstraints=critical,CA:TRUE" -addext "keyUsage=critical,keyCertSign,cRLSign" -out "$certs/zenith-ca.crt"
  fi
  if [ ! -s "$certs/zenith.crt" ]; then
    openssl ecparam -name prime256v1 -genkey -noout -out "$certs/zenith.key"
    openssl req -new -key "$certs/zenith.key" -subj "/CN=$domain" -out "$certs/zenith.csr"
    printf 'subjectAltName=DNS:%s,DNS:*.%s\nextendedKeyUsage=serverAuth\nkeyUsage=digitalSignature,keyEncipherment\n' "$domain" "$domain" > "$certs/leaf.ext"
    openssl x509 -req -in "$certs/zenith.csr" -CA "$certs/zenith-ca.crt" -CAkey "$certs/zenith-ca.key" -CAcreateserial \
      -days 730 -sha256 -extfile "$certs/leaf.ext" -out "$certs/zenith.crt"
    cat "$certs/zenith-ca.crt" >> "$certs/zenith.crt"
    rm -f "$certs/zenith.csr" "$certs/leaf.ext" "$certs/zenith-ca.srl"
  fi
  # Keys owner-only; certificates world-readable (the api container runs as a
  # non-root user and reads zenith-ca.crt), whatever the host umask is.
  chmod 600 "$certs"/*.key
  chmod 644 "$certs"/*.crt
  cat > traefik/dynamic/certs.yml <<'EOF'
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
}

# cmd_install NAME candidate|release TAG — copies the repository into the server as a bare
# repo at /src/zenith.git (always with the candidate tag v<candidate>, so `zen upgrade` to the
# candidate can fetch it on a release install) and runs the chosen tag's install.sh inside.
cmd_install() {
  local name="${1:-}" mode="${2:-}" tag="${3:-}" c ip domain cand
  c="$(cname "$name")"
  case "$mode" in
    candidate) ;;
    release) [ -n "$tag" ] || die "usage: server.sh install <name> release <tag>"
             git -C "$repo" rev-parse -q --verify "refs/tags/$tag" >/dev/null || die "no local tag $tag";;
    *) die "usage: server.sh install <name> candidate|release <tag>";;
  esac
  if [ -z "${ZENITH_E2E_ZEN:-}" ] || [ ! -x "${ZENITH_E2E_ZEN:-}" ]; then die "ZENITH_E2E_ZEN must point to the candidate zen binary"; fi
  ZENITH_E2E_ZEN="$(cd "$(dirname "$ZENITH_E2E_ZEN")" && pwd)/$(basename "$ZENITH_E2E_ZEN")"
  export ZENITH_E2E_ZEN
  cand="$(candidate_version)"
  if [ "$mode" = candidate ]; then tag="v$cand"; fi
  ip="$(ip_of "$name")"; domain="$ip.sslip.io"
  mkdir -p "$ART"

  echo "$ip" > "$ART/$name.ip"
  local work seed=0; work="$(mktemp -d)"
  git clone -q --bare "$repo" "$work/zenith.git"
  tag_candidate "$work/zenith.git"
  if [ "$mode" = release ]; then
    git -C "$work/zenith.git" show "$tag:infra/scripts/install.sh" > "$work/install.sh"
    if needs_tls_seed "$work/install.sh"; then seed=1; fi
  fi
  docker exec "$c" sh -c 'rm -rf /src/zenith.git && mkdir -p /src'
  docker cp -q "$work/zenith.git" "$c:/src/zenith.git"
  # docker cp keeps the host uid (1001 on GitHub runners); git refuses a repo owned by someone else.
  docker exec "$c" chown -R 0:0 /src/zenith.git
  rm -rf "$work"
  docker cp -q "$ZENITH_E2E_ZEN" "$c:/usr/local/bin/zen"
  docker exec "$c" chmod 0755 /usr/local/bin/zen

  local envs=(
    "ZENITH_REPO_URL=/src/zenith.git" "ZENITH_BRANCH=$tag" "ZENITH_DIR=$INSTALL_DIR"
    "ZENITH_DOMAIN=$domain" "ZENITH_PUBLIC_IP=$ip" "ZENITH_TLS_MODE=selfsigned" "ADMIN_EMAIL=$ADMIN_EMAIL"
  )
  if [ "$mode" = candidate ]; then envs+=("ZENITH_IMAGE_REPO=$REG:5000"); fi
  if [ -n "${ZENITH_E2E_ADMIN_PASSWORD_OVERRIDE:-}" ]; then envs+=("ADMIN_PASSWORD=$ZENITH_E2E_ADMIN_PASSWORD_OVERRIDE"); fi
  local args=() kv
  for kv in "${envs[@]}"; do args+=(-e "$kv"); done

  log "installing $tag on $name ($domain); log: $ART/$name-install.log"
  local start=$SECONDS
  # The chosen tag's own install.sh, straight from the bare repo (the curl only fails fast
  # when the repo copy is missing). Bounded: a hung install fails instead of blocking the job.
  # `timeout` kills only the docker exec client; install.sh may keep running inside the
  # server, which is removed afterwards (run.sh cleanup / server.sh rm).
  # shellcheck disable=SC2016 # expanded by the server's bash, from the -e variables
  if ! timeout "${ZENITH_E2E_INSTALL_TIMEOUT:-1500}" docker exec "${args[@]}" "$c" bash -c '
      set -euo pipefail
      cd /
      curl -fsSL file:///src/zenith.git/HEAD >/dev/null
      git --git-dir=/src/zenith.git show "$ZENITH_BRANCH:infra/scripts/install.sh" > /root/install.sh
      bash /root/install.sh </dev/null' > "$ART/$name-install.log" 2>&1; then
    log "install on $name failed after $((SECONDS - start))s; last lines:"
    tail -n 40 "$ART/$name-install.log" >&2 || true
    cmd_logs "$name"; return 1
  fi
  # "Installed" means usable: the wall time includes waiting for /api/v1/version.
  wait_health "$name" "$HEALTH_TIMEOUT"
  local secs=$((SECONDS - start))

  if [ "$seed" = 1 ]; then
    # $tag's install.sh has no ZENITH_TLS_MODE: seed the self-signed CA + leaf the candidate
    # installer writes, let Traefik pick them up, and wait for /api/v1/version again.
    log "$tag predates ZENITH_TLS_MODE: seeding the self-signed CA for $domain"
    docker exec -w "$INSTALL_DIR" "$c" bash -c "set -euo pipefail; $(declare -f seed_selfsigned_tls); seed_selfsigned_tls \"\$1\"" _ "$domain"
    docker exec -w "$INSTALL_DIR" "$c" docker compose restart traefik >/dev/null
    wait_health "$name" "$HEALTH_TIMEOUT"
  fi

  if [ "$mode" = release ]; then
    # The released compose file ignores this; once `zen upgrade` checks out the candidate,
    # its compose file pulls the candidate images from the in-job registry.
    docker exec -w "$INSTALL_DIR" "$c" sh -c "grep -q '^ZENITH_IMAGE_REPO=' .env || echo 'ZENITH_IMAGE_REPO=$REG:5000' >> .env"
  fi

  rm -f "$ART/$name-ca.crt"
  if ! docker cp -q "$c:$INSTALL_DIR/traefik/certs/zenith-ca.crt" "$ART/$name-ca.crt" 2>/dev/null; then
    log "no CA at $INSTALL_DIR/traefik/certs/zenith-ca.crt on $name"; cmd_logs "$name"; return 1
  fi
  local pw
  pw="$(docker exec -w "$INSTALL_DIR" "$c" sh -c "grep '^ADMIN_PASSWORD=' .env | tail -1 | cut -d= -f2-")"
  [ -n "$pw" ] || { log "no ADMIN_PASSWORD in $INSTALL_DIR/.env on $name"; return 1; }
  cmd_env "$name" "$domain" "$pw" "$secs" > "$ART/$name.env"
  log "$name installed in ${secs}s at https://$domain (env: $ART/$name.env)"
}

# cmd_env NAME DOMAIN PASSWORD SECONDS — prints the suite env for a server. The hook values
# are split on whitespace by the suite, so the paths in them must not contain any.
cmd_env() {
  [ "$#" -eq 4 ] || die "usage: server.sh env <name> <domain> <password> <seconds>"
  local name="$1" domain="$2" pw="$3" secs="$4" s="$here/server.sh" cand
  case "$s" in *[[:space:]]*) die "path $s contains whitespace; the suite splits hook commands on whitespace";; esac
  cand="$(candidate_version)"
  printf 'ZENITH_E2E_ENDPOINT=https://%s\n' "$domain"
  printf 'ZENITH_E2E_DOMAIN=%s\n' "$domain"
  printf 'ZENITH_E2E_ADMIN_EMAIL=%s\n' "$ADMIN_EMAIL"
  printf 'ZENITH_E2E_ADMIN_PASSWORD=%s\n' "$pw"
  printf 'ZENITH_E2E_CA_FILE=%s/%s-ca.crt\n' "$ART" "$name"
  printf 'ZENITH_E2E_ZEN=%s\n' "${ZENITH_E2E_ZEN:-zen}"
  printf 'ZENITH_E2E_INSTALL_SECONDS=%s\n' "$secs"
  printf 'ZENITH_E2E_SERVER_EXEC=%s exec %s --\n' "$s" "$name"
  printf 'ZENITH_E2E_REBOOT_CMD=%s reboot %s\n' "$s" "$name"
  printf 'ZENITH_E2E_COPY_FROM_SERVER=%s cp %s\n' "$s" "$name"
  printf 'ZENITH_E2E_CANDIDATE_VERSION=%s\n' "$cand"
  printf 'ZENITH_E2E_PROBE_IMAGE=%s:5000/e2e-probe:%s\n' "$REG" "$cand"
  printf 'ZENITH_E2E_ARTIFACTS=%s\n' "$ART"
}

cmd="${1:-}"; shift || true
case "$cmd" in
  new|install|reboot|exec|cp|push|rm|down|logs|env) "cmd_$cmd" "$@" ;;
  ip) ip_of "${1:-}" ;;
  candidate-version) candidate_version ;;
  # Internal, used by server_test.sh (no Docker): the pieces of `install`.
  tag-candidate) tag_candidate "${1:?bare repo}" ;;
  needs-tls-seed) needs_tls_seed "${1:?install.sh}" ;;
  seed-tls) (cd "${1:?install dir}" && seed_selfsigned_tls "${2:?domain}") ;;
  *) sed -n '11,22p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2; exit 2 ;;
esac
