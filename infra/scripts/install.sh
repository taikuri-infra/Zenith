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

# Generate a URL-safe secret. Avoid shell-hostile characters (no quotes/$/etc).
gen_secret() {
  local len="${1:-32}"
  LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c "$len"
}

# ---- Docker ----------------------------------------------------------------
SUDO=""
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

# ---- Fetch the stack -------------------------------------------------------
if [ -d "$INSTALL_DIR/.git" ]; then
  log "Updating existing checkout in '$INSTALL_DIR'..."
  git -C "$INSTALL_DIR" pull --ff-only
else
  log "Fetching FreeZenith ($REPO_BRANCH)..."
  git clone --depth 1 --branch "$REPO_BRANCH" "$REPO_URL" "$INSTALL_DIR"
fi
cd "$INSTALL_DIR"

# ---- Generate .env with strong secrets -------------------------------------
CREDS_FILE=".zenith-credentials"
if [ ! -f .env ]; then
  log "Generating .env with strong, unique secrets..."
  cp .env.example .env

  JWT_SECRET="$(gen_secret 48)"
  ADMIN_PASSWORD="$(gen_secret 20)"
  DB_PASSWORD="$(gen_secret 24)"
  S3_SECRET_KEY="$(gen_secret 24)"
  ADMIN_EMAIL="${ADMIN_EMAIL:-admin@localhost}"

  # In-place edit that works on both GNU and BSD/macOS sed.
  set_var() {
    local key="$1" val="$2"
    if grep -qE "^${key}=" .env; then
      sed -i.bak -E "s|^${key}=.*|${key}=${val}|" .env
    else
      printf '%s=%s\n' "$key" "$val" >>.env
    fi
  }
  set_var JWT_SECRET "$JWT_SECRET"
  set_var ADMIN_EMAIL "$ADMIN_EMAIL"
  set_var ADMIN_PASSWORD "$ADMIN_PASSWORD"
  set_var DB_PASSWORD "$DB_PASSWORD"
  set_var S3_SECRET_KEY "$S3_SECRET_KEY"

  # The API deploys user apps via the Docker socket and runs as a non-root user,
  # so it needs the host's docker group GID.
  DOCKER_GID="$(getent group docker 2>/dev/null | cut -d: -f3)"
  [ -z "$DOCKER_GID" ] && DOCKER_GID="$(stat -c '%g' /var/run/docker.sock 2>/dev/null || echo 999)"
  set_var DOCKER_GID "$DOCKER_GID"
  rm -f .env.bak

  # Persist credentials to a root-only file so they're not lost.
  umask 077
  cat >"$CREDS_FILE" <<EOF
FreeZenith admin credentials (generated $(date -u '+%Y-%m-%d %H:%M UTC'))
Dashboard: http://localhost:3000
Email:     ${ADMIN_EMAIL}
Password:  ${ADMIN_PASSWORD}
EOF
  ADMIN_CREDS_GENERATED=1
else
  log ".env already exists — keeping current configuration."
  ADMIN_CREDS_GENERATED=0
fi

# ---- Start -----------------------------------------------------------------
log "Pulling images and starting FreeZenith..."
$COMPOSE up -d

printf '\n\033[32m============================================\033[0m\n'
printf '  FreeZenith is running!\n\n'
printf '  Dashboard:  http://localhost:3000\n'
printf '  API:        http://localhost:8080\n\n'
if [ "$ADMIN_CREDS_GENERATED" -eq 1 ]; then
  printf '  Admin email:    %s\n' "$ADMIN_EMAIL"
  printf '  Admin password: \033[1m%s\033[0m\n' "$ADMIN_PASSWORD"
  printf '  (also saved to %s/%s)\n\n' "$(pwd)" "$CREDS_FILE"
else
  printf '  Admin credentials are unchanged (see your existing .env / %s).\n\n' "$CREDS_FILE"
fi
printf '  Add your own domain + HTTPS:\n'
printf '    set ZENITH_DOMAIN and ACME_EMAIL in .env, then:\n'
printf '    %s --profile tls up -d\n' "$COMPOSE"
printf '\033[32m============================================\033[0m\n'
