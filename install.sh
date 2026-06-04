#!/bin/sh
set -e

REPO="thecoderbuddy/claude-flipper"
BINARY="flipper"
INSTALL_DIR="/usr/local/bin"

# Detect OS and arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

if [ "$OS" != "linux" ]; then
  echo "This installer is for Linux only."
  echo "On macOS, use: brew install thecoderbuddy/tap/claude-flipper"
  exit 1
fi

# Fetch latest release version
VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version."
  exit 1
fi

TARBALL="claude-flipper_linux_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

echo "Installing claude-flipper ${VERSION} (linux/${ARCH})..."

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "$TMP/$TARBALL"
tar -xzf "$TMP/$TARBALL" -C "$TMP"

install -m 755 "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"

echo "Installed to $INSTALL_DIR/$BINARY"
echo ""
echo "Next steps:"
echo "  flipper setup"
echo "  source ~/.bashrc"
