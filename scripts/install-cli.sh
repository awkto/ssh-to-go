#!/bin/bash
# Install the stogo CLI from the latest GitHub release.
#   curl -fsSL https://raw.githubusercontent.com/awkto/ssh-to-go/main/scripts/install-cli.sh | bash
# Override the install location for sudo-less installs:
#   INSTALL_DIR=$HOME/.local/bin curl -fsSL ... | bash
set -euo pipefail

REPO="awkto/ssh-to-go"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY="stogo"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

ASSET="stogo-${OS}-${ARCH}"

VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep -o '"tag_name": *"v[^"]*"' | head -1 | grep -o 'v[^"]*')

if [ -z "$VERSION" ]; then
  echo "Failed to determine latest version"
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"
echo "Installing stogo ${VERSION} (${OS}/${ARCH}) to ${INSTALL_DIR}"

TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT
curl -fsSL -o "$TMP" "$URL"
chmod +x "$TMP"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"
fi

echo "Installed: $("${INSTALL_DIR}/${BINARY}" version)"
