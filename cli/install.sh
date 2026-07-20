#!/bin/sh
# FreeZenith `zen` CLI installer.
#
#   curl -fsSL https://raw.githubusercontent.com/taikuri-infra/Zenith/main/cli/install.sh | sh
#
# Detects your OS/arch, downloads the latest zen release, verifies its checksum,
# and installs it to /usr/local/bin (or ~/.local/bin).
set -eu

REPO="taikuri-infra/Zenith"
API="https://api.github.com/repos/${REPO}/releases/latest"

err() { printf 'Error: %s\n' "$1" >&2; exit 1; }

command -v curl >/dev/null 2>&1 || err "curl is required"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) err "unsupported architecture: $arch" ;;
esac
case "$os" in
  linux|darwin) ;;
  *) err "unsupported OS: $os" ;;
esac
asset="zen_${os}_${arch}"

tag=$(curl -fsSL "$API" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/')
[ -n "$tag" ] || err "could not determine the latest release"
base="https://github.com/${REPO}/releases/download/${tag}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

printf 'Downloading zen %s (%s)...\n' "$tag" "$asset"
curl -fsSL "$base/$asset" -o "$tmp/zen" || err "download failed (no $asset in release $tag?)"

# Verify checksum when the release ships one.
if curl -fsSL "$base/zen_checksums.txt" -o "$tmp/sums" 2>/dev/null; then
  want=$(grep " ${asset}\$" "$tmp/sums" | awk '{print $1}' | head -1)
  if [ -n "$want" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      got=$(sha256sum "$tmp/zen" | awk '{print $1}')
    else
      got=$(shasum -a 256 "$tmp/zen" | awk '{print $1}')
    fi
    [ "$want" = "$got" ] || err "checksum mismatch — refusing to install"
  fi
fi

chmod +x "$tmp/zen"

dest="/usr/local/bin"
if [ -w "$dest" ]; then
  mv "$tmp/zen" "$dest/zen"
elif command -v sudo >/dev/null 2>&1; then
  sudo mv "$tmp/zen" "$dest/zen"
else
  dest="$HOME/.local/bin"
  mkdir -p "$dest"
  mv "$tmp/zen" "$dest/zen"
  case ":$PATH:" in
    *":$dest:"*) ;;
    *) printf 'Note: add %s to your PATH.\n' "$dest" ;;
  esac
fi

printf '\nzen %s installed to %s/zen.\n' "$tag" "$dest"
printf 'Next: zen install   (self-host FreeZenith on any Linux box)\n'
