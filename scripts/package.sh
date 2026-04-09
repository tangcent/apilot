#!/usr/bin/env bash
# package.sh — build apilot binaries for all platforms
set -euo pipefail

VERSION=${1:-$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0-dev")}
OUT=bin

mkdir -p "$OUT"

echo "Building apilot v${VERSION}..."

GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o "$OUT/apilot-darwin-amd64"   ./apilot-cli
GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o "$OUT/apilot-darwin-arm64"   ./apilot-cli
GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o "$OUT/apilot-linux-amd64"    ./apilot-cli
GOOS=linux   GOARCH=arm64 go build -ldflags="-s -w" -o "$OUT/apilot-linux-arm64"    ./apilot-cli
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "$OUT/apilot-windows-amd64.exe" ./apilot-cli

echo "Done. Binaries in ./$OUT/"
ls -lh "$OUT"/
