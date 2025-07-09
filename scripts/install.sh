#!/usr/bin/env bash

set -e

REPO="alex3ax/redis-analyzer"
VERSION="${VERSION:-latest}"

detect_platform() {
  OS="$(uname | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"

  case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported arch: $ARCH"; exit 1 ;;
  esac

  echo "${OS}-${ARCH}"
}

download_binary() {
  PLATFORM=$(detect_platform)

  if [ "$VERSION" == "latest" ]; then
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep tag_name | cut -d '"' -f 4)
  fi

  echo "Installing redis-analyzer $VERSION for $PLATFORM"

  BINARY_NAME="redis-analyzer-${PLATFORM}"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"

  curl -L "$URL" -o redis-analyzer
  chmod +x redis-analyzer
  sudo mv redis-analyzer /usr/local/bin/

  echo "✅ Installed redis-analyzer to /usr/local/bin/"
  redis-analyzer --help
}

download_binary
