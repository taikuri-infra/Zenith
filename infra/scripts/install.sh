#!/usr/bin/env bash
set -euo pipefail

# FreeZenith — one-command installer
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/taikuri-infra/Zenith/main/infra/scripts/install.sh | bash
#
# Installs Docker if missing, fetches the stack, generates strong secrets, and
# starts FreeZenith. Pulls prebuilt public images (no build needed).

REPO_URL="${ZENITH_REPO_URL:-https://github.com/taikuri-infra/Zenith.git}"
REPO_BRANCH="${ZENITH_BRANCH:-main}"
INSTALL_DIR="${ZENITH_DIR:-zenith}"

log()  { printf '\033[36m==>\033[0m %s\n' "$1"; }
warn() { printf '\033[33m!\033[0m  %s\n' "$1"; }
err()  { printf '\033[31mError:\033[0m %s\n' "$1" >&2; }

DRY_RUN="${ZENITH_DRY_RUN:-0}"   # tests: render .env in the current dir, no Docker/git/network

# Sets PUBLIC_IP and IP_SOURCE (ipify | env | local). Runs in the current shell
# (no command substitution) so the globals survive. Never fails.
detect_public_ip() {
  PUBLIC_IP=""; IP_SOURCE=""
  if [ -n "${ZENITH_PUBLIC_IP:-}" ]; then PUBLIC_IP="$ZENITH_PUBLIC_IP"; IP_SOURCE="env"; return 0; fi
  PUBLIC_IP="$(curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null || true)"
  if [ -n "$PUBLIC_IP" ]; then IP_SOURCE="ipify"; return 0; fi
  PUBLIC_IP="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
  IP_SOURCE="local"
  return 0
}

# Ensure ZENITH_DOMAIN is set: explicit, else <public-ip>.sslip.io. Exits 1
# (before any file is written) when no valid IP can be found.
resolve_domain() {
  [ -n "${ZENITH_DOMAIN:-}" ] && return 0
  detect_public_ip
  if ! [[ "$PUBLIC_IP" =~ ^[0-9]+(\.[0-9]+){3}$ ]]; then
    err "Could not detect this server's public IP. Set ZENITH_DOMAIN=<your domain> (or ZENITH_PUBLIC_IP=<ip>) and re-run."
    exit 1
  fi
  if [ "$IP_SOURCE" = "local" ]; then
    warn "Public IP lookup failed; using local address $PUBLIC_IP. If it is private, Let's Encrypt cannot issue a certificate for it."
  fi
  ZENITH_DOMAIN="${PUBLIC_IP}.sslip.io"
}

# In-place edit that works on both GNU and BSD/macOS sed.
set_var() {
  local key="$1" val="$2"
  if grep -qE "^${key}=" .env; then
    sed -i.bak -E "s|^${key}=.*|${key}=${val}|" .env
  else
    printf '%s=%s\n' "$key" "$val" >>.env
  fi
}

# True when KEY has a non-empty value in .env.
has_var() { grep -qE "^$1=." .env; }

gen_hex() { # 32 random bytes as 64 hex chars
  od -An -tx1 -N32 /dev/urandom | tr -d ' \n'
}

# Generate a URL-safe secret. Avoid shell-hostile characters (no quotes/$/etc).
gen_secret() {
  local len="${1:-32}"
  # `head -c` closes the pipe early, sending SIGPIPE to tr; under `set -o
  # pipefail` that would abort the whole script. The trailing `|| true` absorbs
  # it — the secret is already fully captured by the time the pipe breaks.
  LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom 2>/dev/null | head -c "$len" || true
}

# ---- Docker ----------------------------------------------------------------
SUDO=""
DOCKER_PREFIX=""
COMPOSE="docker compose"
if [ "$DRY_RUN" != "1" ]; then
if [ "$(id -u)" -ne 0 ]; then SUDO="sudo"; fi

if ! command -v docker >/dev/null 2>&1; then
  warn "Docker is not installed."
  log "Installing Docker via the official get.docker.com script..."
  if ! curl -fsSL https://get.docker.com | $SUDO sh; then
    err "Automatic Docker install failed. Install it manually: https://docs.docker.com/get-docker/"
    exit 1
  fi
  # Let the current user run docker without sudo on subsequent logins.
  if [ -n "$SUDO" ] && [ -n "${USER:-}" ]; then
    $SUDO usermod -aG docker "$USER" 2>/dev/null || true
  fi
fi

# Resolve the compose command (v2 plugin preferred).
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  err "Docker Compose not found. Install: https://docs.docker.com/compose/install/"
  exit 1
fi
log "Docker and Docker Compose ready."

# A freshly-added docker group membership (the usermod above) does NOT apply to
# this already-running shell — only to future logins. So `docker` can be
# unreachable right now even though the daemon is up and running. Detect that and
# fall back to sudo for the container operations. On a box where the user already
# has docker access (or is root), no sudo is used.
DOCKER_PREFIX=""
if ! docker info >/dev/null 2>&1; then
  if [ -n "$SUDO" ] && $SUDO docker info >/dev/null 2>&1; then
    DOCKER_PREFIX="$SUDO"
    warn "Docker group access isn't active in this shell yet — using sudo for now."
    warn "Log out and back in to use 'docker' without sudo."
  fi
fi

# ---- Fetch the stack -------------------------------------------------------
if [ -d "$INSTALL_DIR/.git" ]; then
  log "Updating existing checkout in '$INSTALL_DIR'..."
  git -C "$INSTALL_DIR" pull --ff-only
else
  log "Fetching FreeZenith ($REPO_BRANCH)..."
  git clone --depth 1 --branch "$REPO_BRANCH" "$REPO_URL" "$INSTALL_DIR"
fi
cd "$INSTALL_DIR"
fi  # DRY_RUN

# ---- Generate .env with strong secrets -------------------------------------
CREDS_FILE=".zenith-credentials"
if [ ! -f .env ]; then
  resolve_domain
  log "Generating .env with strong, unique secrets..."
  cp .env.example .env

  JWT_SECRET="$(gen_secret 48)"
  ADMIN_PASSWORD="$(gen_secret 20)"
  DB_PASSWORD="$(gen_secret 24)"
  S3_SECRET_KEY="$(gen_secret 24)"
  ADMIN_EMAIL="${ADMIN_EMAIL:-admin@localhost}"

  set_var JWT_SECRET "$JWT_SECRET"
  set_var ADMIN_EMAIL "$ADMIN_EMAIL"
  set_var ADMIN_PASSWORD "$ADMIN_PASSWORD"
  set_var DB_PASSWORD "$DB_PASSWORD"
  set_var S3_SECRET_KEY "$S3_SECRET_KEY"
  set_var S3_ACCESS_KEY "$(gen_secret 20)"
  set_var SECRETS_ENCRYPTION_KEY "$(gen_hex)"
  set_var ZENITH_DOMAIN "$ZENITH_DOMAIN"

  # The API deploys user apps via the Docker socket and runs as a non-root user,
  # so it needs the host's docker group GID.
  # `|| true`: getent may be absent (macOS / minimal images) and pipefail would
  # otherwise abort the installer on the failed command substitution.
  DOCKER_GID="$(getent group docker 2>/dev/null | cut -d: -f3 || true)"
  [ -z "$DOCKER_GID" ] && DOCKER_GID="$(stat -c '%g' /var/run/docker.sock 2>/dev/null || echo 999)"
  [ -z "$DOCKER_GID" ] && DOCKER_GID=999
  set_var DOCKER_GID "$DOCKER_GID"
  rm -f .env.bak

  # Persist credentials to a root-only file so they're not lost.
  umask 077
  cat >"$CREDS_FILE" <<EOF
FreeZenith admin credentials (generated $(date -u '+%Y-%m-%d %H:%M UTC'))
Dashboard: https://${ZENITH_DOMAIN}
Email:     ${ADMIN_EMAIL}
Password:  ${ADMIN_PASSWORD}
EOF
  ADMIN_CREDS_GENERATED=1
else
  log ".env already exists — keeping current configuration."
  # Upgrade path: the current compose needs these two. Add only if absent;
  # an existing SECRETS_ENCRYPTION_KEY is never overwritten.
  if ! has_var ZENITH_DOMAIN; then
    resolve_domain
    set_var ZENITH_DOMAIN "$ZENITH_DOMAIN"
    log "Added ZENITH_DOMAIN=$ZENITH_DOMAIN to .env"
  fi
  if ! has_var SECRETS_ENCRYPTION_KEY; then
    set_var SECRETS_ENCRYPTION_KEY "$(gen_hex)"
    log "Added SECRETS_ENCRYPTION_KEY to .env"
  fi
  rm -f .env.bak
  ADMIN_CREDS_GENERATED=0
fi

if [ "$DRY_RUN" != "1" ]; then
# ---- Start -----------------------------------------------------------------
log "Pulling images and starting FreeZenith..."
$DOCKER_PREFIX $COMPOSE up -d

printf '\n\033[32m============================================\033[0m\n'
printf '  FreeZenith is running!\n\n'
printf '  Dashboard:  https://%s\n\n' "$(grep '^ZENITH_DOMAIN=' .env | cut -d= -f2-)"
if [ "$ADMIN_CREDS_GENERATED" -eq 1 ]; then
  printf '  Admin email:    %s\n' "$ADMIN_EMAIL"
  printf '  Admin password: \033[1m%s\033[0m\n' "$ADMIN_PASSWORD"
  printf '  (also saved to %s/%s)\n\n' "$(pwd)" "$CREDS_FILE"
else
  printf '  Admin credentials are unchanged (see your existing .env / %s).\n\n' "$CREDS_FILE"
fi
printf '  If you use your own domain, point it and *.<domain> (A records) at this server.\n'
printf '\033[32m============================================\033[0m\n'
fi  # DRY_RUN
