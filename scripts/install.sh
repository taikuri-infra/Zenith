#!/usr/bin/env bash
set -euo pipefail

# Zenith — One-liner install script
# Usage: curl -fsSL https://raw.githubusercontent.com/dotechhq/zenith/main/scripts/install.sh | bash

REPO_URL="https://github.com/dotechhq/zenith.git"
INSTALL_DIR="zenith"

echo "==> Zenith Installer"
echo ""

# Check for Docker
if ! command -v docker &>/dev/null; then
  echo "Error: Docker is not installed."
  echo "Install Docker: https://docs.docker.com/get-docker/"
  exit 1
fi

# Check for Docker Compose (v2 plugin or standalone)
if docker compose version &>/dev/null; then
  COMPOSE="docker compose"
elif command -v docker-compose &>/dev/null; then
  COMPOSE="docker-compose"
else
  echo "Error: Docker Compose is not installed."
  echo "Install Docker Compose: https://docs.docker.com/compose/install/"
  exit 1
fi

echo "==> Docker and Docker Compose found"

# Clone or update repo
if [ -d "$INSTALL_DIR" ]; then
  echo "==> Directory '$INSTALL_DIR' exists, pulling latest..."
  cd "$INSTALL_DIR"
  git pull --ff-only
else
  echo "==> Cloning Zenith..."
  git clone "$REPO_URL" "$INSTALL_DIR"
  cd "$INSTALL_DIR"
fi

# Create .env if it doesn't exist
if [ ! -f .env ]; then
  echo "==> Generating .env from template..."
  cp .env.example .env

  # Generate a random JWT secret
  JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c 64)
  if [ "$(uname)" = "Darwin" ]; then
    sed -i '' "s/^JWT_SECRET=$/JWT_SECRET=${JWT_SECRET}/" .env
  else
    sed -i "s/^JWT_SECRET=$/JWT_SECRET=${JWT_SECRET}/" .env
  fi

  echo "==> JWT_SECRET generated"
else
  echo "==> .env already exists, keeping current configuration"
fi

# Start services
echo "==> Starting Zenith..."
$COMPOSE up -d --build

echo ""
echo "============================================"
echo "  Zenith is running!"
echo ""
echo "  Dashboard:  http://localhost:3000"
echo "  API:        http://localhost:8080"
echo ""
echo "  Login with:"
echo "    Email:    admin@localhost"
echo "    Password: changeme"
echo ""
echo "  Edit .env to change admin credentials"
echo "  and restart with: $COMPOSE up -d"
echo "============================================"
