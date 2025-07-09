#!/usr/bin/env bash

set -e

REPO="alex3ax/redis-analyzer"
VERSION="${VERSION:-latest}"
INSTALL_DIR="/usr/local/bin"

detect_platform() {
  OS="$(uname | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"

  if [ "$OS" = "linux" ]; then
    ARCH="amd64"
  else
    case "$ARCH" in
      x86_64) ARCH="amd64" ;;
      arm64|aarch64) ARCH="arm64" ;;
      *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
    esac
  fi

  echo "${OS}-${ARCH}"
}

download_file() {
  url="$1"
  dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -sSLf -o "$dest" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$dest" "$url"
  else
    echo "❌ Neither curl nor wget is installed."
    exit 1
  fi
}

download_and_install() {
  PLATFORM=$(detect_platform)

  if [ "$VERSION" == "latest" ]; then
    if command -v curl >/dev/null 2>&1; then
      VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep tag_name | cut -d '"' -f 4)
    elif command -v wget >/dev/null 2>&1; then
      VERSION=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep tag_name | cut -d '"' -f 4)
    else
      echo "❌ Neither curl nor wget is installed."
      exit 1
    fi
  fi

  echo "📦 Installing redis-analyzer ${VERSION} for ${PLATFORM}..."

  ARCHIVE_NAME="redis-analyzer-${PLATFORM}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

  TMP_DIR=$(mktemp -d)
  cd "$TMP_DIR"

  echo "⬇️  Downloading $URL"
  download_file "$URL" "$ARCHIVE_NAME"

  echo "📂 Extracting..."
  tar -xzf "$ARCHIVE_NAME"

  if [ ! -f "redis-analyzer" ]; then
    echo "❌ Binary not found in archive"
    exit 1
  fi

  chmod +x redis-analyzer
  sudo mv redis-analyzer "$INSTALL_DIR/redis-analyzer"

  echo "✅ Installed to $INSTALL_DIR/redis-analyzer"
  "$INSTALL_DIR/redis-analyzer" --help
}

download_and_install