#!/bin/bash
# Cross-compile clawkit for all platforms
set -e

VERSION="0.1.0"
OUT="dist"

mkdir -p "$OUT"

echo "Building clawkit v${VERSION}..."

CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "-s -w -X main.version=${VERSION}" -o "${OUT}/clawkit-mac-arm" .
echo "  ✓ macOS ARM64"

CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "-s -w -X main.version=${VERSION}" -o "${OUT}/clawkit-mac-intel" .
echo "  ✓ macOS AMD64"

CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "-s -w -X main.version=${VERSION}" -o "${OUT}/clawkit-linux" .
echo "  ✓ Linux AMD64"

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.version=${VERSION}" -o "${OUT}/clawkit.exe" .
echo "  ✓ Windows AMD64"

echo ""
echo "Done! Binaries in ${OUT}/"
ls -lh "${OUT}/"
