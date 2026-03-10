#!/bin/sh
# Zenith CLI installer
# Usage: curl -fsSL https://get.freezenith.com | sh
set -e

REPO="dotechhq/zenith-cli"
INSTALL_DIR="${ZENITH_INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "Detecting system: ${OS}/${ARCH}"

# Get latest version
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "Failed to determine latest version"
  exit 1
fi

echo "Installing zenith v${VERSION}..."

# Download
URL="https://github.com/${REPO}/releases/download/v${VERSION}/zenith_${OS}_${ARCH}.tar.gz"
TMP=$(mktemp -d)
curl -fsSL "$URL" -o "${TMP}/zenith.tar.gz"

# Extract
tar -xzf "${TMP}/zenith.tar.gz" -C "${TMP}"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/zenith" "${INSTALL_DIR}/zenith"
else
  sudo mv "${TMP}/zenith" "${INSTALL_DIR}/zenith"
fi

chmod +x "${INSTALL_DIR}/zenith"
rm -rf "$TMP"

echo "zenith v${VERSION} installed to ${INSTALL_DIR}/zenith"
echo ""
echo "Get started:"
echo "  zenith login"
echo "  zenith apps list"
