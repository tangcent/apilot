#!/usr/bin/env bash
# plugin-package.sh — package the VSCode extension as a .vsix file
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VSCODE_DIR="$ROOT_DIR/vscode-plugin"
BIN_DIR="$VSCODE_DIR/bin"

VERSION=${1:-$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.1.0")}

echo "Packaging apilot VSCode extension v${VERSION}..."

cd "$VSCODE_DIR"

if [ ! -f package-lock.json ]; then
  echo "Installing dependencies..."
  npm install
fi

echo "Compiling TypeScript..."
npm run compile

mkdir -p "$BIN_DIR"

echo "Downloading apilot binaries..."
BASE_URL="https://github.com/tangcent/apilot/releases/latest/download"

download_binary() {
  local os=$1
  local arch=$2
  local suffix=$3
  local target="$BIN_DIR/apilot-${os}-${arch}${suffix}"
  if [ ! -f "$target" ]; then
    echo "  Downloading apilot-${os}-${arch}${suffix}..."
    curl -fsSL "${BASE_URL}/apilot-${os}-${arch}${suffix}" -o "$target"
    chmod +x "$target"
  else
    echo "  apilot-${os}-${arch}${suffix} already exists, skipping..."
  fi
}

download_binary darwin arm64 ""
download_binary darwin amd64 ""
download_binary linux arm64 ""
download_binary linux amd64 ""
download_binary windows amd64 ".exe"

echo "Installing vsce..."
if ! command -v vsce &> /dev/null; then
  npm install -g @vscode/vsce
fi

echo "Updating package version to ${VERSION}..."
npm pkg set version="$VERSION"

echo "Creating .vsix package..."
vsce package --allow-missing-repository --skip-license

VSIX_FILE="apilot-${VERSION}.vsix"
if [ -f "$VSIX_FILE" ]; then
  echo "Done. Created: $VSCODE_DIR/$VSIX_FILE"
  ls -lh "$VSIX_FILE"
else
  echo "Error: Failed to create .vsix package"
  exit 1
fi
