#!/bin/bash
# clawkit installer — download the correct binary for this machine
# Usage: curl -fsSL https://raw.githubusercontent.com/Rockship-Team/clawkit/main/install.sh | bash
set -e

REPO="Rockship-Team/clawkit"
INSTALL_DIR="/usr/local/bin"
BINARY="clawkit"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}▸${NC} $1"; }
ok()    { echo -e "${GREEN}✓${NC} $1"; }
fail()  { echo -e "${RED}✗${NC} $1"; exit 1; }

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)  OS="linux" ;;
    darwin) OS="darwin" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *)      fail "Unsupported OS: $OS. On Windows, use: irm https://raw.githubusercontent.com/Rockship-Team/clawkit/main/install.ps1 | iex" ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)             fail "Unsupported architecture: $ARCH" ;;
esac

ASSET="${BINARY}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    ASSET="${ASSET}.exe"
fi

echo ""
echo "  clawkit installer"
echo "  ─────────────────"
echo ""
info "Detected: ${OS}/${ARCH}"

# Get latest release download URL
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
info "Downloading from: ${DOWNLOAD_URL}"

# Download to temp file
TMP_FILE=$(mktemp)
HTTP_CODE=$(curl -fsSL -o "$TMP_FILE" -w "%{http_code}" "$DOWNLOAD_URL" 2>/dev/null) || true

if [ "$HTTP_CODE" != "200" ]; then
    rm -f "$TMP_FILE"

    # Fallback: build from source
    info "No pre-built binary found. Building from source..."

    if ! command -v go &> /dev/null; then
        fail "Go is not installed. Install Go 1.22+ first: https://go.dev/dl/"
    fi

    CLONE_DIR=$(mktemp -d)
    git clone --depth 1 "https://github.com/${REPO}.git" "$CLONE_DIR" 2>/dev/null
    cd "$CLONE_DIR"
    CGO_ENABLED=0 go build -ldflags "-s -w" -o "$BINARY" ./cmd/clawkit
    TMP_FILE="${CLONE_DIR}/${BINARY}"
fi

# Install
chmod +x "$TMP_FILE"

if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY}"
else
    info "Need sudo to install to ${INSTALL_DIR}"
    sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY}"
fi

ok "Installed: $(${BINARY} version)"
echo ""
echo "  Get started:"
echo "    clawkit list"
echo "    clawkit install shop-hoa-zalo"
echo ""
