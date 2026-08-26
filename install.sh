#!/usr/bin/env bash

set -euo pipefail

REPO="lucasepe/cliphub"
BINARY="cliphub"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

usage() {
  echo "Usage: $0 [version]"
  echo
  echo "Examples:"
  echo "  $0"
  echo "  $0 1.2.3"
  echo "  $0 v1.2.3"
  exit 1
}

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "❌ Missing required command: $1"
    exit 1
  fi
}

need curl
need unzip
need uname
need find

if [[ $# -gt 1 ]]; then
  usage
elif [[ $# -eq 1 ]]; then
  VERSION="${1#v}"
  LATEST_TAG="v${VERSION}"
else
  # Try to fetch the latest release
  JSON=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" || true)
  LATEST_TAG=$(echo "$JSON" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)

  # Fallback: if no official "latest" release exists, use the most recent tag
  if [[ -z "$LATEST_TAG" || "$LATEST_TAG" == "null" ]]; then
    echo "⚠️  No 'latest' release found, falling back to tags..."
    LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/tags" \
      | grep '"name":' \
      | sed -E 's/.*"([^"]+)".*/\1/' \
      | head -n 1 || true)
  fi

  if [[ -z "$LATEST_TAG" || "$LATEST_TAG" == "null" ]]; then
    echo "❌ Could not determine latest version."
    exit 1
  fi

  VERSION="${LATEST_TAG#v}"
fi

# Detect OS.
OS="$(uname | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux|darwin) ;;
  msys*|cygwin*|mingw*) OS="windows" ;;
  *) echo "❌ Unsupported OS: $OS" && exit 1 ;;
esac

# Detect architecture.
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "❌ Unsupported architecture: $ARCH" && exit 1 ;;
esac

EXT="zip"
if [[ "$OS" == "windows" ]]; then
  # The current release workflow publishes only one Windows archive.
  ASSET="${BINARY}-${OS}.${EXT}"
  BIN_NAME="${BINARY}.exe"
else
  ASSET="${BINARY}-${OS}-${ARCH}.${EXT}"
  BIN_NAME="$BINARY"
fi
URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ASSET}"
TMP_DIR=$(mktemp -d)

cleanup() {
  rm -rf "$ASSET" "$TMP_DIR"
}
trap cleanup EXIT

echo "📦 Downloading $ASSET from $LATEST_TAG..."
echo "🔗 $URL"
curl -fL "$URL" -o "$ASSET"

echo "📂 Extracting to $TMP_DIR..."
unzip -o "$ASSET" -d "$TMP_DIR" >/dev/null

if [ ! -w "$INSTALL_DIR" ]; then
  echo "⚠️  No permission for $INSTALL_DIR, falling back to $HOME/.local/bin"
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
  echo "👉 Make sure $INSTALL_DIR is in your PATH"
fi

BIN_PATH=$(find "$TMP_DIR" -type f -name "$BIN_NAME" | head -n 1)
if [[ -z "$BIN_PATH" ]]; then
  echo "❌ Could not find the '$BIN_NAME' binary inside ZIP"
  exit 1
fi

echo "🚀 Installing $BINARY to $INSTALL_DIR..."
chmod +x "$BIN_PATH"
mv "$BIN_PATH" "$INSTALL_DIR/$BIN_NAME"

echo "✅ $BINARY $VERSION installed successfully!"
